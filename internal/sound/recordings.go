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

// Supplied Juggernaut recordings, converted to stereo PCM WAV for embedding.
// No runtime network requests or external decoder executable are required.
//
//go:embed clips/attack-*.wav clips/miss-*.wav
var combatRecordings embed.FS

func loadAttackSamples() ([attackVariants][]byte, error) {
	var result [attackVariants][]byte
	err := loadRecordings("attack", result[:])
	return result, err
}

func loadMissSamples() ([missVariants][]byte, error) {
	var result [missVariants][]byte
	err := loadRecordings("miss", result[:])
	return result, err
}

func loadRecordings(family string, result [][]byte) error {
	for i := range result {
		name := fmt.Sprintf("clips/%s-%d.wav", family, i+1)
		data, err := combatRecordings.ReadFile(name)
		if err != nil {
			return err
		}
		stream, err := wav.DecodeWithSampleRate(SampleRate, bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("decode %s: %w", name, err)
		}
		pcm, err := io.ReadAll(stream)
		if err != nil || len(pcm) == 0 || len(pcm)%4 != 0 {
			return fmt.Errorf("invalid PCM recording %s: %v", name, err)
		}
		prepareRecording(pcm)
		result[i] = pcm
	}
	return nil
}

func prepareRecording(data []byte) {
	frames := len(data) / 4
	for i := 0; i < frames; i++ {
		// Preserve the recording's pitch, timing and spectrum. Only lower the
		// level and taper 3ms at file boundaries to prevent playback clicks.
		fade := math.Min(1, float64(i)/(SampleRate*0.003))
		fade *= math.Min(1, float64(frames-1-i)/(SampleRate*0.003))
		for ch := 0; ch < 2; ch++ {
			offset := i*4 + ch*2
			v := int16(binary.LittleEndian.Uint16(data[offset:]))
			binary.LittleEndian.PutUint16(data[offset:], uint16(int16(float64(v)*recordedCombatGain*fade)))
		}
	}
}
