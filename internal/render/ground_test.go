package render

import (
	"image"
	"math"
	"reflect"
	"testing"

	"github.com/stariydedd/yurnerogue-go/internal/domain"
)

func TestGroundFogMatchesRememberedSprites(t *testing.T) {
	fog := float32(ExploredDim) / 255
	for shade := 0; shade <= 3; shade++ {
		visibleLight := 1 - fog*groundDimScale(shade, true)
		rememberedLight := 1 - fog*groundDimScale(shade, false)
		want := visibleLight * (1 - fog)
		if math.Abs(float64(rememberedLight-want)) > 1e-6 {
			t.Fatalf("shade %d: remembered floor light %f, want %f", shade, rememberedLight, want)
		}
	}
	if groundDimScale(0, false) != 1 {
		t.Fatal("fog opacity applied twice")
	}
}

func TestGroundSampleReconstructsContinuousSurface(t *testing.T) {
	for y := -16; y < 24; y++ {
		for x := -16; x < 24; x++ {
			frame, sample := groundSample(x, y)
			if frame < 0 || frame >= 4 || !sample.In(image.Rect(0, 0, 128, 128)) || sample.Size() != image.Pt(TileSize, TileSize) {
				t.Fatalf("invalid sample at %d,%d: %d %v", x, y, frame, sample)
			}
			whole := sample.Add(image.Pt(frame%2*128, frame/2*128))
			want := image.Pt((x%8+8)%8*TileSize, (y%8+8)%8*TileSize)
			if whole.Min != want {
				t.Fatalf("discontinuous surface at %d,%d: %v, want %v", x, y, whole.Min, want)
			}
		}
	}
}

func TestGroundContoursKeepCoresAndDoNotOverwriteNeighbours(t *testing.T) {
	for seed := int64(1); seed <= 12; seed++ {
		s := domain.NewSessionSeed(seed)
		grid := s.BuildGrid(false)
		v := newForestView(grid, s.ComputeVisibility(grid))
		for p := range v.ground {
			covered := map[image.Point]bool{}
			cell := image.Rect(p.X*32, p.Y*32, (p.X+1)*32, (p.Y+1)*32)
			for _, span := range groundContour(v, p) {
				if span.rect.Empty() || !span.rect.In(cell.Inset(-10)) || span.shade < 0 || span.shade > 3 {
					t.Fatalf("invalid span: %v", span)
				}
				for y := span.rect.Min.Y; y < span.rect.Max.Y; y++ {
					for x := span.rect.Min.X; x < span.rect.Max.X; x++ {
						pixel := image.Pt(x, y)
						if covered[pixel] {
							t.Fatal("overlapping spans would double the shadow")
						}
						covered[pixel] = true
						if !pixel.In(cell) && v.ground[domain.Point{X: x / 32, Y: y / 32}] {
							t.Fatal("bank overwrites another floor cell")
						}
					}
				}
			}
			for y := cell.Min.Y + 5; y < cell.Max.Y-5; y++ {
				for x := cell.Min.X + 5; x < cell.Max.X-5; x++ {
					if !covered[image.Pt(x, y)] {
						t.Fatalf("floor core cut out at %d,%d", x, y)
					}
				}
			}
		}
	}
}

func TestGroundContourRoundCornersAndContinuousBanks(t *testing.T) {
	v := forestView{ground: map[domain.Point]bool{}}
	for y := 10; y < 14; y++ {
		for x := 10; x < 14; x++ {
			v.ground[domain.Point{X: x, Y: y}] = true
		}
	}
	p := domain.Point{X: 10, Y: 10}
	spans := groundContour(v, p)
	corner := image.Pt(320, 320)
	for _, span := range spans {
		if corner.In(span.rect) {
			t.Fatal("outer corner still square")
		}
	}
	interior := groundContour(v, domain.Point{X: 11, Y: 11})
	if len(interior) != 1 || interior[0].shade != 0 || interior[0].rect != image.Rect(352, 352, 384, 384) {
		t.Fatal("interior has seams or shadows")
	}
	for side := range forestDirs {
		for along := -96; along < 320; along++ {
			a, b := groundWave(along, 320, side), groundWave(along+1, 320, side)
			if a < -3 || a > 9 || a-b > .4 || b-a > .4 {
				t.Fatal("discontinuous or excessive wave")
			}
		}
	}
}

func TestGroundContoursIgnoreUnknownGeometry(t *testing.T) {
	s := domain.NewSessionSeed(21)
	a, b := s.BuildGrid(false), s.BuildGrid(false)
	vis := s.ComputeVisibility(a)
	for y := 0; y < domain.Rows; y++ {
		for x := 0; x < domain.Cols; x++ {
			p := domain.Point{X: x, Y: y}
			if !vis.Visible[p] && !vis.Explored[p] {
				b[y][x] = domain.SymRoomFloor
			}
		}
	}
	va, vb := newForestView(a, vis), newForestView(b, vis)
	for p := range va.ground {
		if !reflect.DeepEqual(groundContour(va, p), groundContour(vb, p)) {
			t.Fatal("hidden geometry changes bank")
		}
	}
}

func BenchmarkGroundContours(b *testing.B) {
	s := domain.NewSessionSeed(21)
	grid := s.BuildGrid(false)
	v := newForestView(grid, s.ComputeVisibility(grid))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for p := range v.ground {
			groundContour(v, p)
		}
	}
}

func TestTrailFringePreservesWalkableCentre(t *testing.T) {
	for y := -2; y < 12; y++ {
		for x := -2; x < 12; x++ {
			p := domain.Point{X: x, Y: y}
			cell := image.Rect(x*TileSize, y*TileSize, (x+1)*TileSize, (y+1)*TileSize)
			for side := range forestDirs {
				strips := trailFringe(p, side)
				if len(strips) != TileSize/4 {
					t.Fatal("incomplete edge")
				}
				for _, strip := range strips {
					if strip.Empty() || !strip.In(cell) || strip.Overlaps(cell.Inset(5)) {
						t.Fatalf("unsafe fringe at %v: %v", p, strip)
					}
				}
			}
		}
	}
}

func TestTrailFringesOnlyUseKnownConnections(t *testing.T) {
	p := domain.Point{X: 10, Y: 10}
	v := forestView{ground: map[domain.Point]bool{p: true}}
	paths := map[domain.Point]bool{p: true}
	before := trailFringes(v, paths, p)
	for _, d := range forestDirs {
		paths[domain.Point{X: p.X + d.X, Y: p.Y + d.Y}] = true
	}
	if !reflect.DeepEqual(before, trailFringes(v, paths, p)) {
		t.Fatal("hidden passages change visible trail edges")
	}
	for side, d := range forestDirs {
		q := domain.Point{X: p.X + d.X, Y: p.Y + d.Y}
		v.ground[q] = true
		var want []image.Rectangle
		for other := range forestDirs {
			if other != side {
				want = append(want, trailFringe(p, other)...)
			}
		}
		if got := trailFringes(v, paths, p); !reflect.DeepEqual(got, want) {
			t.Fatalf("grass crosses known path connection on side %d", side)
		}
		delete(v.ground, q)
	}
}
