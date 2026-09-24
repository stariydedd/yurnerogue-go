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

// The attack and miss files keep headroom (their source MP3 peaks reach above
// full scale); playback restores the same amount, so loudness is unchanged.
const (
	attackGain = recordedCombatGain * 1.19 // file lowered by 1.5 dB
	missGain   = recordedCombatGain * 1.06 // file lowered by 0.5 dB
)

// Supplied Juggernaut recordings, converted to 44.1 kHz stereo PCM WAV for embedding.
// No runtime network requests or external decoder executable are required.
//
//go:embed clips/attack-*.wav clips/miss-*.wav clips/blade-*.wav
var combatRecordings embed.FS

// Single recordings for ability cues: Blade Dance, layered at 80% over a
// regular attack recording on a Critical Strike, and a large blade ring for a
// parried hit. The files are clean conversions with headroom; their gain lifts
// them to the attack recordings' loudness here, in playback, instead of
// limiting the file (which made them sound dirty).
type abilityClip struct {
	name string
	gain float64
}

var abilityRecordings = map[Cue]abilityClip{
	Critical: {"clips/blade-dance.wav", recordedCombatGain * 1.61 * 0.8}, // +4.1 dB, then 80% under the attack
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
		// Preserve the recording's pitch, timing and spectrum. Only lower the
		// level and taper 3ms at file boundaries to prevent playback clicks.
		fade := math.Min(1, float64(i)/(SampleRate*0.003))
		fade *= math.Min(1, float64(frames-1-i)/(SampleRate*0.003))
		for ch := 0; ch < 2; ch++ {
			offset := i*4 + ch*2
			v := int16(binary.LittleEndian.Uint16(data[offset:]))
			binary.LittleEndian.PutUint16(data[offset:], uint16(int16(float64(v)*gain*fade)))
		}
	}
}
