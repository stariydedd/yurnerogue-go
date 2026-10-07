package render

import (
	"image"
	"math"
	"sort"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

// Декорации опираются только на открытую местность. Неизвестные комнаты должны выглядеть
// ровно как обычный лес: их геометрия не должна влиять на расстановку деревьев и свет.
type forestView struct {
	ground   map[domain.Point]bool
	distance map[domain.Point]int
	// lights хранит яркость каждой клетки по строкам, заполняется один раз на вид:
	// световое поле опрашивается тысячи раз за перестройку, и поиск по map
	// делал это самой медленной её частью.
	lights []float32
	// cells повторяет ground по строкам для горячих циклов, заполняется вместе с lights;
	// seen делает то же для видимых клеток.
	cells []bool
	seen  []bool
}

func (v forestView) isGround(x, y int) bool {
	if v.cells != nil {
		return x >= 0 && y >= 0 && x < domain.Cols && y < domain.Rows && v.cells[y*domain.Cols+x]
	}
	return v.ground[domain.Point{X: x, Y: y}]
}

var forestDirs = []domain.Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}}

// visibleSignature считает отпечаток видимого множества независимо от порядка в map.
func visibleSignature(visible map[domain.Point]bool) uint64 {
	sum := uint64(len(visible))
	for p, seen := range visible {
		if !seen {
			continue
		}
		// Финализатор splitmix64 разносит соседние клетки подальше друг от друга.
		h := uint64(uint32(p.X))<<32 | uint64(uint32(p.Y))
		h ^= h >> 30
		h *= 0xbf58476d1ce4e5b9
		h ^= h >> 27
		h *= 0x94d049bb133111eb
		h ^= h >> 31
		sum += h
	}
	return sum
}

// Запоминаем только открытый пол, включая коридоры, для этого уровня. Это
// состояние отрисовки: оно не должно раскрывать предметы или менять видимость в игре.
type forestMemory struct {
	level  *domain.Level
	ground map[domain.Point]bool
}

func (m *forestMemory) reveal(level *domain.Level, grid domain.Grid, vis domain.Visibility) domain.Visibility {
	if m.level != level || m.ground == nil {
		m.level = level
		m.ground = map[domain.Point]bool{}
	}
	for _, cells := range []map[domain.Point]bool{vis.Visible, vis.Explored} {
		for p, known := range cells {
			if known && isFloor(grid.At(p.X, p.Y)) {
				m.ground[p] = true
			}
		}
	}
	return domain.Visibility{Visible: vis.Visible, Explored: m.ground}
}

func newForestView(grid domain.Grid, vis domain.Visibility) forestView {
	v := forestView{ground: map[domain.Point]bool{}, distance: map[domain.Point]int{}}
	var queue []domain.Point
	for y := 0; y < domain.Rows; y++ {
		for x := 0; x < domain.Cols; x++ {
			p := domain.Point{X: x, Y: y}
			if (vis.Visible[p] || vis.Explored[p]) && isFloor(grid.At(x, y)) {
				v.ground[p] = true
				if vis.Visible[p] {
					v.distance[p] = 0
					queue = append(queue, p)
				}
			}
		}
	}
	for i := 0; i < len(queue); i++ {
		p := queue[i]
		d := v.distance[p]
		if d >= 7 {
			continue
		}
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				q := domain.Point{X: p.X + dx, Y: p.Y + dy}
				if _, ok := v.distance[q]; !ok && domain.InBounds(q.X, q.Y) {
					v.distance[q] = d + 1
					queue = append(queue, q)
				}
			}
		}
	}
	v.lights = make([]float32, domain.Cols*domain.Rows)
	for i := range v.lights {
		v.lights[i] = forestLightLevels[len(forestLightLevels)-1]
	}
	for p, d := range v.distance {
		v.lights[p.Y*domain.Cols+p.X] = forestLightLevels[d]
	}
	v.cells = make([]bool, domain.Cols*domain.Rows)
	for p := range v.ground {
		if domain.InBounds(p.X, p.Y) {
			v.cells[p.Y*domain.Cols+p.X] = true
		}
	}
	v.seen = make([]bool, domain.Cols*domain.Rows)
	for p, visible := range vis.Visible {
		if visible && domain.InBounds(p.X, p.Y) {
			v.seen[p.Y*domain.Cols+p.X] = true
		}
	}
	return v
}

// Несколько пикселей листьев или корней могут заходить за край, но не в центр
// проходимой клетки. Так однорядный коридор остаётся открытым по всей длине.
func (v forestView) fits(rect image.Rectangle, fringe int) bool {
	for y := max(0, rect.Min.Y/TileSize-1); y <= min(domain.Rows-1, rect.Max.Y/TileSize); y++ {
		for x := max(0, rect.Min.X/TileSize-1); x <= min(domain.Cols-1, rect.Max.X/TileSize); x++ {
			if v.isGround(x, y) && rect.Overlaps(image.Rect(x*TileSize, y*TileSize, (x+1)*TileSize, (y+1)*TileSize).Inset(fringe)) {
				return false
			}
		}
	}
	return true
}

// forestLightLevels: яркость по расстоянию в клетках от видимого.
var forestLightLevels = [...]float32{1, .98, .9, .76, .58, .43, .32, .25, .2}

func (v forestView) tileLight(x, y int) float32 {
	x, y = clamp(x, 0, domain.Cols-1), clamp(y, 0, domain.Rows-1)
	if v.lights != nil {
		return v.lights[y*domain.Cols+x]
	}
	d, ok := v.distance[domain.Point{X: x, Y: y}]
	if !ok {
		d = len(forestLightLevels) - 1
	}
	return forestLightLevels[d]
}

// lightAt: непрерывное световое поле; яркость клетки стоит в её центре
// и смешивается между центрами. Ступенчатый свет по клеткам рисовал жёсткие
// прямые края по сетке клеток вокруг освещённой комнаты.
func (v forestView) lightAt(px, py float64) float32 {
	fx, fy := px/TileSize-.5, py/TileSize-.5
	x0, y0 := int(math.Floor(fx)), int(math.Floor(fy))
	tx, ty := float32(fx-float64(x0)), float32(fy-float64(y0))
	top := v.tileLight(x0, y0)*(1-tx) + v.tileLight(x0+1, y0)*tx
	bottom := v.tileLight(x0, y0+1)*(1-tx) + v.tileLight(x0+1, y0+1)*tx
	return top*(1-ty) + bottom*ty
}

// light: самая яркая точка области предмета на непрерывном поле, поэтому
// куст, заходящий в свет, освещён, а соседи затеняются плавно.
func (v forestView) light(rect image.Rectangle) float32 {
	if rect.Dx() <= 1 && rect.Dy() <= 1 {
		return v.lightAt(float64(rect.Min.X), float64(rect.Min.Y))
	}
	var best float32
	for i := 0; i <= 2; i++ {
		for j := 0; j <= 2; j++ {
			x := float64(rect.Min.X) + float64(rect.Dx())*float64(i)/2
			y := float64(rect.Min.Y) + float64(rect.Dy())*float64(j)/2
			best = max(best, v.lightAt(x, y))
		}
	}
	return best
}

type forestProp struct {
	role        string
	frame       int
	rect        image.Rectangle
	rotation    int
	groundCover bool
	lightAnchor image.Rectangle
}

// Укоренённый предмет и его мох берут свет в точке касания с землёй,
// а не у того пикселя кроны, который оказался ближе к поляне.
func forestFoot(p forestProp) image.Rectangle {
	x, y := (p.rect.Min.X+p.rect.Max.X)/2, p.rect.Max.Y-4
	return image.Rect(x-p.rect.Dx()/4, y-6, x+p.rect.Dx()/4, y+6)
}

func forestBed(p forestProp) forestProp {
	foot := forestFoot(p)
	x, y := (foot.Min.X+foot.Max.X)/2, (foot.Min.Y+foot.Max.Y)/2
	w := p.rect.Dx() + 24
	return forestProp{
		role: "moss", frame: p.frame, rect: image.Rect(x-w/2, y-w/4, x+w/2, y+w/4),
		groundCover: true, lightAnchor: foot,
	}
}

func forestRootGrass(p forestProp) forestProp {
	return forestProp{role: "verge", frame: p.frame,
		rect:        image.Rect(p.rect.Min.X+p.rect.Dx()/5, p.rect.Max.Y-14, p.rect.Max.X-p.rect.Dx()/5, p.rect.Max.Y),
		lightAnchor: forestFoot(p)}
}

func (p forestProp) lightingBounds() image.Rectangle {
	if !p.lightAnchor.Empty() {
		return p.lightAnchor
	}
	return p.rect
}

// Границы кроны и ствола не включают прозрачные поля спрайта, поэтому рощи
// могут переплетаться, не пряча стволы за кронами переднего плана.
func forestTreeSilhouette(p forestProp) (image.Rectangle, image.Rectangle) {
	// Четыре упакованных дерева стоят на общей линии корней, но верх кроны у них разный.
	tops := [...]int{8, 4, 22, 34}
	frame := ((p.frame % 4) + 4) % 4
	crown := image.Rect(p.rect.Min.X+5, p.rect.Min.Y+p.rect.Dy()*tops[frame]/128,
		p.rect.Max.X-5, p.rect.Min.Y+p.rect.Dy()*88/128)
	trunk := image.Rect(p.rect.Min.X+p.rect.Dx()/3, crown.Max.Y,
		p.rect.Max.X-p.rect.Dx()/3, p.rect.Max.Y-8)
	return crown, trunk
}

// Сначала выбираем весь лес, потом отбрасываем открытый пол. Убранные деревья
// всё равно занимают своё место, чтобы открытие тропы не перетасовывало соседей.
func forestTrees(v forestView) []forestProp {
	type candidate struct {
		prop     forestProp
		priority int
	}
	var candidates []candidate
	add := func(x, y, h, priority int) {
		w, height := 100+(h/113)%21, 128+(h/199)%25
		rect := image.Rect(x-w/2, y-height, x+w/2, y)
		candidates = append(candidates, candidate{forestProp{role: "tree", frame: h / 7, rect: rect}, priority})
	}
	for gy := 0; gy <= (domain.Rows*TileSize+160)/56; gy++ {
		for gx := 0; gx <= (domain.Cols*TileSize+160)/56; gx++ {
			h := cellHash(gx+317, gy+911)
			if h%4 == 0 {
				continue
			}
			x := gx*56 + (gy%2)*28 + h%41 - 20
			y := gy*56 + (h/41)%43 - 21
			add(x, y, h, 10000+h%100000)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].priority < candidates[j].priority })
	var trees []forestProp
	space := forestSpace{}
	for _, c := range candidates {
		crown, trunk := forestTreeSilhouette(c.prop)
		if space.free(crown.Inset(-6)) && space.free(trunk.Inset(-6)) {
			space.occupy(crown)
			space.occupy(trunk)
			if v.fits(c.prop.rect, 5) {
				trees = append(trees, c.prop)
			}
		}
	}
	return trees
}

// Пространственные корзины делают выбор по всему миру дешёвым даже на больших экранах.
// Корзины только опрашиваются, а не обходятся как map, поэтому порядок детерминирован.
type forestSpace map[image.Point][]image.Rectangle

func (s forestSpace) free(rect image.Rectangle) bool {
	for y := rect.Min.Y / 64; y <= (rect.Max.Y-1)/64; y++ {
		for x := rect.Min.X / 64; x <= (rect.Max.X-1)/64; x++ {
			for _, occupied := range s[image.Pt(x, y)] {
				if rect.Overlaps(occupied) {
					return false
				}
			}
		}
	}
	return true
}

func (s forestSpace) occupy(rect image.Rectangle) {
	for y := rect.Min.Y / 64; y <= (rect.Max.Y-1)/64; y++ {
		for x := rect.Min.X / 64; x <= (rect.Max.X-1)/64; x++ {
			p := image.Pt(x, y)
			s[p] = append(s[p], rect)
		}
	}
}

// Низкая листва может окружать тонкий ствол, но должна оставаться ниже кроны.
// Защищённый силуэт не даёт растениям выглядеть как сваленные в кучу деревья.
func (s forestSpace) protectTree(p forestProp) {
	crown, trunk := forestTreeSilhouette(p)
	s.occupy(crown)
	s.occupy(trunk)
}

// worldTrees: расстановка деревьев неоткрытого мира. Она ни от чего не зависит,
// поэтому считается один раз, а не при каждой перерисовке местности.
var worldTrees = sync.OnceValue(func() []forestProp { return forestTrees(forestView{}) })

// worldScenery: все высокие или твёрдые предметы неоткрытого мира, то есть деревья,
// кладка и группы растений. Расстояния между ними зависят только от позиции в мире,
// поэтому расстановка считается один раз, а открытая земля её только фильтрует.
// Раньше она строилась заново при каждой перестройке местности, на каждом шаге по коридору.
var worldScenery = sync.OnceValue(func() []forestProp {
	// Все высокие и твёрдые предметы участвуют в расстояниях даже за пределами экрана.
	scenery := append([]forestProp(nil), worldTrees()...)
	space := forestSpace{}
	for _, tree := range scenery {
		space.protectTree(tree)
	}
	add := func(role string, frame int, rect image.Rectangle) bool {
		if space.free(rect) {
			scenery = append(scenery, forestProp{role: role, frame: frame, rect: rect})
			inset := 5
			if role == "bush" && rect.Dy() > 30 {
				inset = 10
			}
			space.occupy(rect.Inset(inset))
			return true
		}
		return false
	}
	// Оставляем промежутки под кладку, пока их не заняли большие кусты. Обломки
	// и изредка мелкий щебень образуют рыхлые группы, а не сплошные стены.
	for gy := 0; gy <= (domain.Rows*TileSize+96)/72; gy++ {
		for gx := 0; gx <= (domain.Cols*TileSize+96)/80; gx++ {
			h := cellHash(gx+419, gy+827)
			if h%5 == 0 {
				continue
			}
			x, y := gx*80+(gy%2)*40+h%49-24, gy*72+(h/49)%41-20
			w, height := 44+(h/71)%17, 42+(h/137)%13
			frame := 0
			if h%3 == 0 {
				frame = 2
				height += 12 // У картинки упавшего блока есть запас места над обломками.
			}
			if h%13 == 0 {
				frame = 1
				height = 72
			}
			if add("ruins", frame, image.Rect(x-w/2, y-height, x+w/2, y)) && h%3 == 0 {
				side := 1
				if h%2 == 0 {
					side = -1
				}
				cx, cy := x+side*(w/2+12), y+6-(h/17)%13
				add("ruins", 2, image.Rect(cx-17, cy-40, cx+17, cy))
			}
		}
	}
	// Собираем группы вокруг существующих опор: сначала широкие листья, потом мелкие
	// папоротники-спутники. Корни и кладка остаются различимыми между растениями.
	anchors := append([]forestProp(nil), scenery...)
	for _, anchor := range anchors {
		if anchor.role != "tree" && anchor.role != "ruins" {
			continue
		}
		x, y := (anchor.rect.Min.X+anchor.rect.Max.X)/2, anchor.rect.Max.Y
		h := cellHash(x+271, y+613)
		for i, side := range []int{-1, 1} {
			w, height := 48+(h/31+i*7)%19, 38+(h/73+i*5)%15
			cx, cy := x+side*(anchor.rect.Dx()/3+8+(h/19)%13), y+12+(h/43)%13
			frame := (h/7 + i) % 3
			if anchor.role == "ruins" && (h+i)%7 == 0 {
				frame = 3
			}
			add("bush", frame, image.Rect(cx-w/2, cy-height, cx+w/2, cy))
		}
	}
	// Низкие несимметричные группы занимают промежутки между деревьями, а не только край
	// поляны. Каменные группы прерывают зелень, не становясь новой стеной.
	for gy := 0; gy <= (domain.Rows*TileSize+96)/28; gy++ {
		for gx := 0; gx <= (domain.Cols*TileSize+96)/32; gx++ {
			h := cellHash(gx+701, gy+239)
			if h%5 == 0 {
				continue
			}
			x := gx*32 + (gy%2)*16 + h%29 - 14
			y := gy*28 + (h/29)%27 - 13
			role, frame := "bush", (h/11)%3
			w, height := 54+(h/31)%29, 44+(h/73)%25
			if h%6 == 0 {
				role, frame = "ruins", ((h/17)%2)*2
				w, height = 46+(h/31)%11, 58+(h/73)%13
			} else if h%47 == 0 {
				frame = 3
			}
			add(role, frame, image.Rect(x-w/2, y-height, x+w/2, y))
		}
	}
	// Отдельный проход мелких растений заполняет оставшиеся карманы, не увеличивая
	// фоновый ковёр листвы. Папоротники и редкие голубые побеги остаются различимыми
	// у кладки и под деревьями, с той же защитой стволов и коридоров.
	for gy := 0; gy <= (domain.Rows*TileSize+48)/24; gy++ {
		for gx := 0; gx <= (domain.Cols*TileSize+48)/28; gx++ {
			h := cellHash(gx+1019, gy+367)
			if h%3 == 0 {
				continue
			}
			x, y := gx*28+(gy%2)*14+h%19-9, gy*24+(h/19)%17-8
			w, height := 20+(h/43)%13, 18+(h/101)%13
			frame := 1 + (h/7)%2
			if h%11 == 0 {
				frame = 3
			}
			add("bush", frame, image.Rect(x-w/2, y-height, x+w/2, y))
		}
	}
	return scenery
})

func forestProps(v forestView, viewport image.Rectangle) []forestProp {
	// Обочина: это почвопокровные растения, они намеренно перекрывают корни и другие края.
	var props []forestProp
	// Предметы на границе создаются в порядке координат, а не обхода map,
	// поэтому равные глубины и варианты стабильны между кадрами. Кусок края
	// уходит меньше чем на три клетки от своей клетки, так что добавить его могут
	// только клетки рядом с областью; чанку больше не нужно обходить всю карту.
	for y := max(0, viewport.Min.Y/TileSize-3); y <= min(domain.Rows-1, viewport.Max.Y/TileSize+3); y++ {
		for x := max(0, viewport.Min.X/TileSize-3); x <= min(domain.Cols-1, viewport.Max.X/TileSize+3); x++ {
			if !v.isGround(x, y) {
				continue
			}
			for i, d := range forestDirs {
				if v.isGround(x+d.X, y+d.Y) {
					continue
				}
				h := cellHash(x*5+i, y)
				ex, ey := x*TileSize+16+d.X*16, y*TileSize+16+d.Y*16
				// Низкая непрерывная кайма связывает отдельные кусты и корни
				// с полом. Её повёрнутые границы подчиняются той же защите коридоров.
				// У каждого куска своя длина, глубина, вылет и сдвиг вдоль
				// края, поэтому кайма не повторяет сетку клеток одной линией.
				length, depth := 58+(h/13)%15, 24+h%17
				reach, slide := (h/37)%10, (h/191)%13-6
				vw, vh, rotation := length, depth, 0
				switch {
				case d.X == 1:
					vw, vh, rotation = depth, length, 1
				case d.X == -1:
					vw, vh, rotation = depth, length, 3
				case d.Y == 1:
					rotation = 2
				}
				// Листва следует за тем же неровным берегом, что и земля,
				// а не строит поверх него идеально прямую границу.
				along, boundary := ex, ey
				if d.X != 0 {
					along, boundary = ey, ex
				}
				bank := max(0, int(groundWave(along, boundary, i)))
				vx, vy := ex+d.X*(vw/2-5+bank+reach), ey+d.Y*(vh/2-5+bank+reach)
				if d.X != 0 {
					vy += slide
				} else {
					vx += slide
				}
				vr := image.Rect(vx-vw/2, vy-vh/2, vx+vw/2, vy+vh/2)
				if vr.Overlaps(viewport) && v.fits(vr, 5) {
					props = append(props, forestProp{role: "verge", frame: h, rect: vr, rotation: rotation})
				}
			}
		}
	}
	for _, p := range worldScenery() {
		// Большая часть мира вне этой области: отбрасываем её до проверки земли.
		if !p.rect.Inset(-propReach).Overlaps(viewport) {
			continue
		}
		// Фильтруем только после того, как каждый твёрдый предмет занял своё постоянное место.
		// В промежутки, оставленные новой открытой землёй, замена не появляется.
		if !v.fits(p.rect, 5) {
			continue
		}
		if p.role == "tree" || p.role == "ruins" {
			p.lightAnchor = forestFoot(p)
			bed := forestBed(p)
			if bed.rect.Overlaps(viewport) {
				props = append(props, bed)
			}
		}
		if p.rect.Overlaps(viewport) {
			props = append(props, p)
		}
	}
	// Затенённый низкий подлесок связывает отдельные силуэты. Он рисуется
	// ПОЗАДИ стволов и камней, а не поверх них, как ещё один ряд крон.
	area := viewport.Inset(-96)
	for gy := max(0, area.Min.Y/24); gy <= area.Max.Y/24; gy++ {
		for gx := max(0, area.Min.X/28); gx <= area.Max.X/28; gx++ {
			h := cellHash(gx+127, gy+563)
			if h%11 == 0 {
				continue
			}
			x, y := gx*28+(gy%2)*14+h%21-10, gy*24+(h/25)%21-10
			w, height := 46+(h/53)%25, 38+(h/97)%21
			rect := image.Rect(x-w/2, y-height, x+w/2, y)
			if rect.Overlaps(viewport) && v.fits(rect, 5) {
				frame := 1 + (h/7)%2
				if h%7 == 0 {
					frame = 0
				}
				props = append(props, forestProp{role: "bush", frame: frame, rect: rect, groundCover: true})
			}
		}
	}
	sort.SliceStable(props, func(i, j int) bool {
		if props[i].groundCover != props[j].groundCover {
			return props[i].groundCover
		}
		if (props[i].role == "moss") != (props[j].role == "moss") {
			return props[j].role == "moss"
		}
		if (props[i].role == "verge") != (props[j].role == "verge") {
			return props[i].role == "verge"
		}
		return props[i].rect.Max.Y < props[j].rect.Max.Y
	})
	return props
}

// forestScene: то, что считает отрисовка местности перед рисованием: вид,
// предметы и поляны всей кешированной области. Полосы одной перестройки
// делят её, поэтому каждая полоса только рисует.
type forestScene struct {
	level     *domain.Level
	vis       domain.Visibility
	paths     map[domain.Point]bool
	view      forestView
	props     []forestProp
	clearings []sceneClearing
}

type sceneClearing struct {
	clearing
	plants []forestProp
	moss   []float32 // mossGrid поляны
}

// propReach: насколько пиксели предмета могут выходить за его прямоугольник
// (поворот, трава у корней, тень касания), чтобы полосы никогда его не обрезали.
const propReach = 64

func propNear(p forestProp, region image.Rectangle) bool {
	return p.rect.Union(p.lightAnchor).Inset(-propReach).Overlaps(region)
}

// drawForestRegion рисует местность одной области мира в dst, у которого
// начало координат в мире стоит в (camX, camY). Каждый слой сохраняет порядок полной отрисовки,
// поэтому полосы одной сцены стыкуются без швов.
func (r *Renderer) drawForestRegion(dst *ebiten.Image, s *forestScene, region image.Rectangle, camX, camY int) {
	v, vis, paths := s.view, s.vis, s.paths
	// Сплошная затенённая трава и почва под лесом. Это декоративная земля,
	// не связанная со скрытыми комнатами, а не второй слой парящих крон.
	for y := region.Min.Y / TileSize; y <= region.Max.Y/TileSize; y++ {
		for x := region.Min.X / TileSize; x <= region.Max.X/TileSize; x++ {
			rect := image.Rect(x*TileSize, y*TileSize, (x+1)*TileSize, (y+1)*TileSize)
			r.drawForestSoil(dst, v, rect, camX, camY, cellHash(x, y))
		}
	}
	// Густая низкая листва идёт под корнями, а не вокруг больших ореолов исключения.
	// Мох затем возвращает точку касания с землёй, не вырезая промежуток
	// в окружающей растительности. Оба слоя ниже непрозрачных проходимых клеток.
	for _, prop := range s.props {
		if prop.groundCover && prop.role == "bush" && propNear(prop, region) {
			r.drawForestProp(dst, prop, camX, camY, v.light(prop.lightingBounds())*.66)
		}
	}
	for _, prop := range s.props {
		if prop.role == "moss" && propNear(prop, region) {
			r.drawForestProp(dst, prop, camX, camY, v.light(prop.lightingBounds())*.85)
			r.drawForestContact(dst, prop.lightAnchor, camX, camY)
		}
	}
	// Включаем клетки сразу за областью, чей декоративный берег заходит внутрь.
	for y := max(0, (region.Min.Y-TileSize)/TileSize); y <= min(domain.Rows-1, (region.Max.Y+TileSize)/TileSize); y++ {
		for x := max(0, (region.Min.X-TileSize)/TileSize); x <= min(domain.Cols-1, (region.Max.X+TileSize)/TileSize); x++ {
			p := domain.Point{X: x, Y: y}
			if !v.isGround(x, y) {
				continue
			}
			role := "floor"
			if paths[p] {
				role = "path"
			}
			h := cellHash(x, y)
			r.drawCachedGround(dst, s.level, v, paths, p, camX, camY, vis.Visible[p])
			// Редкие низкие растения на поляне, а не вторая сетка препятствий.
			if role == "floor" && h%31 == 0 && vis.Visible[p] {
				r.drawTile(dst, "decor", x, y, camX, camY, h/31)
			}
		}
	}
	r.drawClearingBanks(dst, v, vis, s.clearings, region, camX, camY)
	for _, prop := range s.props {
		if prop.groundCover || !propNear(prop, region) {
			continue
		}
		light := v.light(prop.lightingBounds())
		r.drawForestProp(dst, prop, camX, camY, light)
	}
}

// Интерполируем свет между общими углами клеток. Одинаковая яркость на клетку
// выдавала бы шахматку на спокойной почве между растениями.
func (r *Renderer) drawForestSoil(dst *ebiten.Image, v forestView, rect image.Rectangle, camX, camY, frame int) {
	img := r.sprites.Frame("floor", frame)
	if img == nil {
		return
	}
	b := img.Bounds()
	points := [4]image.Point{rect.Min, {X: rect.Max.X, Y: rect.Min.Y}, {X: rect.Min.X, Y: rect.Max.Y}, rect.Max}
	sources := [4]image.Point{b.Min, {X: b.Max.X, Y: b.Min.Y}, {X: b.Min.X, Y: b.Max.Y}, b.Max}
	var vertices [4]ebiten.Vertex
	for i, p := range points {
		light := v.light(image.Rect(p.X, p.Y, p.X+1, p.Y+1)) * .62
		vertices[i] = ebiten.Vertex{DstX: float32(p.X - camX), DstY: float32(p.Y - camY),
			SrcX: float32(sources[i].X), SrcY: float32(sources[i].Y),
			ColorR: light, ColorG: light, ColorB: light, ColorA: 1}
	}
	dst.DrawTriangles(vertices[:], []uint16{0, 1, 2, 1, 3, 2}, img, nil)
}

// Маленькая мягкая тень касания лежит на земле прямо под корнями,
// без смещения, как у парящего предмета. Она рисуется до троп.
func (r *Renderer) drawForestContact(dst *ebiten.Image, foot image.Rectangle, camX, camY int) {
	const segments = 24
	var vertices [segments + 1]ebiten.Vertex
	var indices [segments * 3]uint16
	x, y := float32(foot.Min.X+foot.Max.X)/2-float32(camX), float32(foot.Min.Y+foot.Max.Y)/2-float32(camY)
	vertices[0] = ebiten.Vertex{DstX: x, DstY: y, SrcX: 1, SrcY: 1, ColorA: .62}
	for i := 0; i < segments; i++ {
		a := float64(i) * 2 * math.Pi / segments
		vertices[i+1] = ebiten.Vertex{DstX: x + float32(math.Cos(a))*float32(foot.Dx())*.64,
			DstY: y + float32(math.Sin(a))*float32(foot.Dy())*.8, SrcX: 1, SrcY: 1}
		indices[i*3], indices[i*3+1], indices[i*3+2] = 0, uint16(i+1), uint16((i+1)%segments+1)
	}
	dst.DrawTriangles(vertices[:], indices[:], r.dim, nil)
}

func (r *Renderer) drawForestProp(dst *ebiten.Image, p forestProp, camX, camY int, light float32) {
	img := r.sprites.Frame(p.role, p.frame)
	if img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	w, h := float64(p.rect.Dx()), float64(p.rect.Dy())
	if p.rotation%2 != 0 {
		w, h = h, w
	}
	op.GeoM.Scale(w/float64(img.Bounds().Dx()), h/float64(img.Bounds().Dy()))
	op.GeoM.Translate(-w/2, -h/2)
	op.GeoM.Rotate(float64(p.rotation) * math.Pi / 2)
	op.GeoM.Translate(float64(p.rect.Min.X-camX)+float64(p.rect.Dx())/2, float64(p.rect.Min.Y-camY)+float64(p.rect.Dy())/2)
	op.ColorScale.Scale(light, light, light, 1)
	if p.role == "moss" {
		op.ColorScale.ScaleAlpha(.72)
	}
	dst.DrawImage(img, op)
	if p.role == "tree" {
		// Несколько травинок переднего плана закрывают нижние пиксели корней; остальной
		// ствол остаётся открытым. Эта кайма целиком помещается внутри дерева.
		r.drawForestProp(dst, forestRootGrass(p), camX, camY, light*.86)
	}
}
