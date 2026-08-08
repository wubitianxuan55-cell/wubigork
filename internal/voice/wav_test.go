package voice

import (
	"encoding/binary"
	"testing"
)

func TestWrapPCMAsWAV(t *testing.T) {
	pcm := []byte{0x00, 0x00, 0x01, 0x02}
	wav := wrapPCMAsWAV(pcm)

	if len(wav) != 44+len(pcm) {
		t.Fatalf("expected %d bytes, got %d", 44+len(pcm), len(wav))
	}
	if string(wav[0:4]) != "RIFF" {
		t.Errorf("missing RIFF magic: %q", wav[0:4])
	}
	if string(wav[8:12]) != "WAVE" {
		t.Errorf("missing WAVE magic: %q", wav[8:12])
	}
	if string(wav[12:16]) != "fmt " {
		t.Errorf("missing fmt chunk: %q", wav[12:16])
	}
	if string(wav[36:40]) != "data" {
		t.Errorf("missing data chunk: %q", wav[36:40])
	}

	if got := binary.LittleEndian.Uint32(wav[4:8]); got != uint32(36+len(pcm)) {
		t.Errorf("RIFF size = %d, want %d", got, 36+len(pcm))
	}
	if got := binary.LittleEndian.Uint16(wav[20:22]); got != 1 {
		t.Errorf("audio format = %d, want 1 (PCM)", got)
	}
	if got := binary.LittleEndian.Uint16(wav[22:24]); got != 1 {
		t.Errorf("channels = %d, want 1", got)
	}
	if got := binary.LittleEndian.Uint32(wav[24:28]); got != SampleRate {
		t.Errorf("sample rate = %d, want %d", got, SampleRate)
	}
	if got := binary.LittleEndian.Uint16(wav[34:36]); got != 16 {
		t.Errorf("bits per sample = %d, want 16", got)
	}
	if got := binary.LittleEndian.Uint32(wav[40:44]); got != uint32(len(pcm)) {
		t.Errorf("data size = %d, want %d", got, len(pcm))
	}
	for i := 0; i < len(pcm); i++ {
		if wav[44+i] != pcm[i] {
			t.Fatalf("PCM payload mismatch at offset %d", i)
		}
	}
}

func TestWAVHeaderEmpty(t *testing.T) {
	wav := wrapPCMAsWAV(nil)
	if len(wav) != 44 {
		t.Fatalf("empty PCM should produce a 44-byte header, got %d", len(wav))
	}
}
