package sound

import (
	"bytes"
	"embed"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const attackVariants = 3
const missVariants = 2
const recordedCombatGain = 0.35

// Файлы ударов и промахов хранятся с запасом (пики исходных MP3 выходят
// за полную шкалу); воспроизведение возвращает тот же уровень, громкость не меняется.
const (
	attackGain = recordedCombatGain * 1.19 // файл ослаблен на 1,5 дБ
	missGain   = recordedCombatGain * 1.06 // файл ослаблен на 0,5 дБ
)

// Записи Juggernaut, переведённые в WAV PCM 44,1 кГц стерео для встраивания.
// Во время игры не нужны ни сетевые запросы, ни внешний декодер.
//
//go:embed clips/attack-*.wav clips/miss-*.wav clips/blade-*.wav
var combatRecordings embed.FS

// Отдельные записи для способностей: Blade Dance, наложенный на 80% поверх
// обычной записи атаки при критическом ударе, и громкий звон клинка
// при парировании. Файлы конвертированы чисто и с запасом; усиление поднимает
// их до громкости записей атаки здесь, при воспроизведении, а не ограничением
// файла (от него они звучали грязно).
type abilityClip struct {
	name string
	gain float64
}

var abilityRecordings = map[Cue]abilityClip{
	Critical: {"clips/blade-dance.wav", recordedCombatGain * 1.61 * 0.8}, // +4,1 дБ, затем 80% под атакой
	Parry:    {"clips/blade-ring.wav", recordedCombatGain * 1.92},        // +5.7 dB
}

func loadAttackSamples() ([attackVariants][]byte, error) {
	var result [attackVariants][]byte
	err := loadRecordings("attack", result[:], attackGain)
	return result, err
}

func loadMissSamples() ([missVariants][]byte, error) {
	var result [missVariants][]byte
	err := loadRecordings("miss", result[:], missGain)
	return result, err
}

func loadRecordings(family string, result [][]byte, gain float64) error {
	for i := range result {
		pcm, err := loadRecording(fmt.Sprintf("clips/%s-%d.wav", family, i+1), gain)
		if err != nil {
			return err
		}
		result[i] = pcm
	}
	return nil
}

func loadRecording(name string, gain float64) ([]byte, error) {
	data, err := combatRecordings.ReadFile(name)
	if err != nil {
		return nil, err
	}
	stream, err := wav.DecodeWithSampleRate(SampleRate, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", name, err)
	}
	pcm, err := io.ReadAll(stream)
	if err != nil || len(pcm) == 0 || len(pcm)%4 != 0 {
		return nil, fmt.Errorf("invalid PCM recording %s: %v", name, err)
	}
	prepareRecording(pcm, gain)
	return pcm, nil
}

func prepareRecording(data []byte, gain float64) {
	frames := len(data) / 4
	for i := 0; i < frames; i++ {
		// Сохраняем высоту, тайминг и спектр записи. Только снижаем уровень
		// и сглаживаем 3 мс на границах файла, чтобы не было щелчков.
		fade := math.Min(1, float64(i)/(SampleRate*0.003))
		fade *= math.Min(1, float64(frames-1-i)/(SampleRate*0.003))
		for ch := 0; ch < 2; ch++ {
			offset := i*4 + ch*2
			v := int16(binary.LittleEndian.Uint16(data[offset:]))
			binary.LittleEndian.PutUint16(data[offset:], uint16(int16(float64(v)*gain*fade)))
		}
	}
}
