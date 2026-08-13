package tts

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIEndpoint(t *testing.T) {
	cases := []struct{ base, path, want string }{
		{"http://localhost:8080/v1", "/audio/speech", "http://localhost:8080/v1/audio/speech"},
		{"http://localhost:8080/v1/", "/audio/speech", "http://localhost:8080/v1/audio/speech"},
		{"https://api.deepseek.com", "/audio/speech", "https://api.deepseek.com/v1/audio/speech"},
		{"http://localhost:11434/v1", "/audio/transcriptions", "http://localhost:11434/v1/audio/transcriptions"},
	}
	for _, c := range cases {
		if got := openAIEndpoint(c.base, c.path); got != c.want {
			t.Errorf("openAIEndpoint(%q, %q) = %q, want %q", c.base, c.path, got, c.want)
		}
	}
}

func TestDefaultVoiceForModel(t *testing.T) {
	if got := defaultVoiceForModel("qwen3-tts-customvoice"); got != "serena" {
		t.Errorf("customvoice 默认音色应为 serena, got %q", got)
	}
	if got := defaultVoiceForModel("edge-tts"); got != "zh-CN-YunxiNeural" {
		t.Errorf("edge-tts 默认音色应为 zh-CN-YunxiNeural, got %q", got)
	}
	if got := defaultVoiceForModel("qwen3-tts-voicedesign"); got != "" {
		t.Errorf("voicedesign 不应传音色, got %q", got)
	}
	if got := defaultVoiceForModel("qwen3-tts-voiceclone"); got != "" {
		t.Errorf("voiceclone 不应传音色, got %q", got)
	}
	if got := defaultVoiceForModel("voxcpm2"); got != "" {
		t.Errorf("voxcpm2 不应传音色, got %q", got)
	}
}

func TestNewHerdsmanTTSWithClone_BuildBody(t *testing.T) {
	h := NewHerdsmanTTSWithClone("http://localhost:8080/v1", "qwen3-tts-voiceclone", "data:audio/wav;base64,AAAA", "参考文本")
	body := h.buildBody("要合成的文本", "")
	if body["model"] != "qwen3-tts-voiceclone" || body["input"] != "要合成的文本" {
		t.Errorf("model/input 不符: %+v", body)
	}
	if body["ref_audio"] != "data:audio/wav;base64,AAAA" || body["ref_text"] != "参考文本" {
		t.Errorf("ref_audio/ref_text 不符: %+v", body)
	}
	if _, ok := body["voice"]; ok {
		t.Errorf("voiceclone 不应携带 voice: %+v", body)
	}
	if got := h.resolveVoice(); got != "" {
		t.Errorf("voiceclone resolveVoice 应为空, got %q", got)
	}
}

func TestFetchAudio_DataURI(t *testing.T) {
	payload := []byte("RIFF fake wav bytes")
	uri := "data:audio/wav;base64," + base64.StdEncoding.EncodeToString(payload)
	h := NewHerdsmanTTS("http://localhost:8080/v1", "qwen3-tts-customvoice", "serena")

	data, mime, err := h.fetchAudio(uri)
	if err != nil {
		t.Fatalf("fetchAudio data URI 失败: %v", err)
	}
	if string(data) != string(payload) {
		t.Errorf("data URI 解码内容不符: %q", data)
	}
	if mime != "audio/wav" {
		t.Errorf("mime 应为 audio/wav, got %q", mime)
	}
}

func TestFetchAudio_RelativePath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/audio/abc.wav" {
			w.Header().Set("Content-Type", "audio/wav")
			_, _ = w.Write([]byte("wav-data"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	h := NewHerdsmanTTS(srv.URL+"/v1", "qwen3-tts-customvoice", "serena")
	data, _, err := h.fetchAudio("/audio/abc.wav")
	if err != nil {
		t.Fatalf("fetchAudio 相对路径失败: %v", err)
	}
	if string(data) != "wav-data" {
		t.Errorf("相对路径下载内容不符: %q", data)
	}
}

func TestFetchAudio_RootRelativePathFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/audio/abc.wav" {
			_, _ = w.Write([]byte("root-wav"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	h := NewHerdsmanTTS(srv.URL+"/v1", "qwen3-tts-customvoice", "serena")
	data, _, err := h.fetchAudio("/audio/abc.wav")
	if err != nil {
		t.Fatalf("fetchAudio 根路径回退失败: %v", err)
	}
	if string(data) != "root-wav" {
		t.Errorf("根路径下载内容不符: %q", data)
	}
}

func TestResolveVoice_FallsBackToSupported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/audio/info") {
			_, _ = w.Write([]byte(`{"supported_speakers":["aiden","dylan","serena","vivian"]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	// 配置了不支持的 Cherry，应回退到首选 serena
	h := NewHerdsmanTTS(srv.URL+"/v1", "qwen3-tts-customvoice", "Cherry")
	if got := h.resolveVoice(); got != "serena" {
		t.Errorf("resolveVoice 应回退 serena, got %q", got)
	}

	// 配置了支持的音色应保留
	h2 := NewHerdsmanTTS(srv.URL+"/v1", "qwen3-tts-customvoice", "vivian")
	if got := h2.resolveVoice(); got != "vivian" {
		t.Errorf("resolveVoice 应保留 vivian, got %q", got)
	}
}
