package sound

import (
	"encoding/binary"
	"testing"
)

func TestStereoFloat32SplitsChannels(t *testing.T) {
	samples := []int16{1000, -1000, 32767, -32768}
	pcm := make([]byte, len(samples)*2)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(pcm[i*2:], uint16(s))
	}
	left, right := stereoFloat32(pcm)
	if len(left) != 2 || len(right) != 2 ||
		left[0] != 1000.0/32768 || left[1] != 32767.0/32768 ||
		right[0] != -1000.0/32768 || right[1] != -1 {
		t.Fatalf("got %v %v", left, right)
	}
}
