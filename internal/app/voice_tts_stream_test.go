package app

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStreamBreezeVoiceSSE_EmitsPlayableChunks(t *testing.T) {
	pcm1 := []byte{0x01, 0x00, 0x02, 0x00}
	pcm2 := []byte{0x03, 0x00, 0x04, 0x00}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/speech" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Sample-Rate", "24000")
		fmt.Fprintf(w, "data: {\"type\":\"speech.audio.delta\",\"audio\":%q}\n\n", base64.StdEncoding.EncodeToString(pcm1))
		fmt.Fprintf(w, "data: {\"type\":\"speech.audio.delta\",\"audio\":%q}\n\n", base64.StdEncoding.EncodeToString(pcm2))
		fmt.Fprint(w, "data: {\"type\":\"speech.audio.done\"}\n\n")
	}))
	defer srv.Close()

	a := &mediaState{core: &core{}}
	var chunks [][]byte
	n, err := a.streamBreezeVoiceSSE(
		context.Background(), srv.URL, "breeze-tts-2", "你好。", "冷静平淡",
		func(audio []byte, mimeType string) error {
			if mimeType != "audio/wav" {
				t.Errorf("mimeType = %q", mimeType)
			}
			chunks = append(chunks, append([]byte(nil), audio...))
			return nil
		},
	)
	if err != nil {
		t.Fatalf("streamBreezeVoiceSSE: %v", err)
	}
	if n != 2 || len(chunks) != 2 {
		t.Fatalf("chunks = %d/%d, want 2/2", n, len(chunks))
	}
	for i, chunk := range chunks {
		if len(chunk) < 44 || string(chunk[:4]) != "RIFF" || string(chunk[8:12]) != "WAVE" {
			t.Fatalf("chunk %d 不是可播放 WAV: %q", i, chunk)
		}
	}
}
