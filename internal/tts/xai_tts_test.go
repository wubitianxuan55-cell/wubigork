package tts

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestXaiSupportedVoices(t *testing.T) {
	voices := XaiSupportedVoices()
	if len(voices) < 5 {
		t.Fatalf("音色数量 = %d, want >= 5", len(voices))
	}
	for _, v := range []string{"eve", "ara", "rex", "sal", "leo", "lumen", "zenith"} {
		if !IsXaiVoice(v) {
			t.Errorf("IsXaiVoice(%q) = false, want true", v)
		}
	}
	if IsXaiVoice("EVE") != true {
		t.Errorf("IsXaiVoice(EVE) 应大小写不敏感")
	}
	if IsXaiVoice("serena") {
		t.Errorf("IsXaiVoice(serena) = true，serena 不是 xAI 音色")
	}
	if IsXaiVoice("") {
		t.Errorf("IsXaiVoice(空) = true，应为 false")
	}
}

func TestXaiTTS_SynthesizeWithMime(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("解析请求体失败: %v", err)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write([]byte{0xFF, 0xF3, 0x01, 0x02})
	}))
	defer srv.Close()

	xtts := NewXaiTTS(srv.URL, "EVE", func() (string, error) { return "tok-123", nil }, nil)
	audio, mime, err := xtts.SynthesizeWithMime("好的，晚安。")
	if err != nil {
		t.Fatalf("SynthesizeWithMime: %v", err)
	}
	if len(audio) != 4 {
		t.Errorf("音频长度 = %d, want 4", len(audio))
	}
	if mime != "audio/mpeg" {
		t.Errorf("mime = %q, want audio/mpeg", mime)
	}
	if gotPath != "/tts" {
		t.Errorf("path = %q, want /tts", gotPath)
	}
	if gotAuth != "Bearer tok-123" {
		t.Errorf("Authorization = %q, want Bearer tok-123", gotAuth)
	}
	if gotBody["text"] != "好的，晚安。" {
		t.Errorf("text = %q", gotBody["text"])
	}
	if gotBody["voice_id"] != "eve" {
		t.Errorf("voice_id = %q, want eve（大小写归一）", gotBody["voice_id"])
	}
	if gotBody["language"] != "zh" {
		t.Errorf("language = %q, want zh", gotBody["language"])
	}
}

func TestXaiTTS_FallbackVoice(t *testing.T) {
	xtts := NewXaiTTS("http://x", "not-a-voice", func() (string, error) { return "", nil }, nil)
	if xtts.voiceID != "eve" {
		t.Errorf("无效音色应回退 eve, got %q", xtts.voiceID)
	}
	if !strings.EqualFold(xtts.voiceID, "eve") {
		t.Errorf("voiceID 应归一为小写, got %q", xtts.voiceID)
	}
}
