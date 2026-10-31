package tts

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeBreezeServer 假 breeze-server：捕获请求体并按 handler 回应。
type fakeBreezeServer struct {
	t      *testing.T
	server *httptest.Server
	mu     chan string // 最近一次请求体（容量 1，非阻塞语义由调用方保证单请求）
}

func newFakeBreezeServer(t *testing.T, respond func(w http.ResponseWriter, r *http.Request)) *fakeBreezeServer {
	t.Helper()
	fb := &fakeBreezeServer{t: t, mu: make(chan string, 1)}
	fb.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		select {
		case fb.mu <- string(body):
		default:
		}
		respond(w, r)
	}))
	t.Cleanup(fb.server.Close)
	return fb
}

func (fb *fakeBreezeServer) lastBody() string {
	select {
	case b := <-fb.mu:
		return b
	default:
		return ""
	}
}

// TestBreezeProviderRegistered breeze kind 已注册且工厂字段映射正确。
func TestBreezeProviderRegistered(t *testing.T) {
	p, err := NewTTSProvider("breeze", TTSConfig{
		BaseURL: "http://127.0.0.1:8012/v1", Model: "breeze-tts-2",
		VoiceDescription: "沉稳的青年男声",
	})
	if err != nil {
		t.Fatalf("NewTTSProvider(breeze): %v", err)
	}
	if p.Name() != "breeze" {
		t.Errorf("Name = %q, want breeze", p.Name())
	}
}

// TestBreezeSynthesize_FieldMapping 请求体字段口径（breeze-server OpenAI JSON 路径）：
// input/response_format=wav 恒发；voice 留空不透传（design 模式）；instructions =
// 音色描述 + Emotion/Style 情绪词拼接；响应 audio/wav 二进制直读。
func TestBreezeSynthesize_FieldMapping(t *testing.T) {
	wav := []byte("RIFF....fake-wav-bytes")
	fb := newFakeBreezeServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write(wav)
	})

	p, err := NewTTSProvider("breeze", TTSConfig{
		BaseURL:          fb.server.URL,
		Model:            "breeze-tts-2",
		VoiceDescription: "沉稳的青年男声",
	})
	if err != nil {
		t.Fatalf("NewTTSProvider: %v", err)
	}

	audio, mime, err := p.SynthesizeWithParams("你好，世界。", TTSParams{Emotion: "平静", Style: "语速稍慢"})
	if err != nil {
		t.Fatalf("SynthesizeWithParams: %v", err)
	}
	if string(audio) != string(wav) {
		t.Errorf("audio = %q, want 原样 wav 字节", audio)
	}
	if mime != "audio/wav" {
		t.Errorf("mime = %q, want audio/wav", mime)
	}

	var body map[string]interface{}
	if err := json.Unmarshal([]byte(fb.lastBody()), &body); err != nil {
		t.Fatalf("请求体解析: %v (body=%q)", err, fb.lastBody())
	}
	if body["input"] != "你好，世界。" {
		t.Errorf("input = %v", body["input"])
	}
	if body["response_format"] != "wav" {
		t.Errorf("response_format = %v, want wav", body["response_format"])
	}
	if _, has := body["voice"]; has {
		t.Errorf("voice 留空不应透传（design 模式），got %v", body["voice"])
	}
	ins, _ := body["instructions"].(string)
	if ins == "" || !containsAll(ins, "沉稳的青年男声", "平静", "语速稍慢") {
		t.Errorf("instructions 应含描述+情绪+风格, got %q", ins)
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

// TestBreezeSynthesize_VoiceSent 配置了音色 id 时透传 voice（克隆/已存音色模式）。
func TestBreezeSynthesize_VoiceSent(t *testing.T) {
	fb := newFakeBreezeServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write([]byte("RIFF"))
	})
	p, err := NewTTSProvider("breeze", TTSConfig{
		BaseURL: fb.server.URL, Model: "breeze-tts-2", Voice: "narrator",
	})
	if err != nil {
		t.Fatalf("NewTTSProvider: %v", err)
	}
	if _, _, err := p.SynthesizeWithMime("hi"); err != nil {
		t.Fatalf("SynthesizeWithMime: %v", err)
	}
	var body map[string]interface{}
	_ = json.Unmarshal([]byte(fb.lastBody()), &body)
	if body["voice"] != "narrator" {
		t.Errorf("voice = %v, want narrator", body["voice"])
	}
}

// TestBreezeSynthesize_ErrorPropagation 非 200（如 voice 未知名 404）错误透传。
func TestBreezeSynthesize_ErrorPropagation(t *testing.T) {
	fb := newFakeBreezeServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"unknown voice"}}`))
	})
	p, err := NewTTSProvider("breeze", TTSConfig{
		BaseURL: fb.server.URL, Model: "breeze-tts-2", Voice: "alloy",
	})
	if err != nil {
		t.Fatalf("NewTTSProvider: %v", err)
	}
	if _, _, err := p.SynthesizeWithMime("hi"); err == nil {
		t.Fatal("404 应返回 error")
	} else if !containsAll(err.Error(), "404", "alloy") {
		t.Errorf("错误信息应含状态码与音色名, got %v", err)
	}
}
