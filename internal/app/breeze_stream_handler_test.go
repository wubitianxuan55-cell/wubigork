package app

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/httpbridge"
	"github.com/gaea/gaea/internal/modelengine"
)

func TestWrapPCMAsWAV(t *testing.T) {
	pcm := []byte{1, 2, 3, 4}
	wav := wrapPCMAsWAV(pcm, 24000)
	if len(wav) != 44+len(pcm) {
		t.Fatalf("长度 = %d, want %d", len(wav), 44+len(pcm))
	}
	if string(wav[0:4]) != "RIFF" || string(wav[8:12]) != "WAVE" || string(wav[36:40]) != "data" {
		t.Errorf("魔数不符: %q %q %q", wav[0:4], wav[8:12], wav[36:40])
	}
	if got := binary.LittleEndian.Uint32(wav[40:]); got != uint32(len(pcm)) {
		t.Errorf("data 块长 = %d, want %d", got, len(pcm))
	}
	if got := binary.LittleEndian.Uint32(wav[24:]); got != 24000 {
		t.Errorf("采样率 = %d, want 24000", got)
	}
}

// TestStreamBreezeTTS_HappyPath SSE happy path：delta→迷你 WAV chunk 事件、
// done 收尾；事件经 httpbridge hub 订阅捕获（tts_service_test 同款机制）。
func TestStreamBreezeTTS_HappyPath(t *testing.T) {
	sseBody := "data: {\"type\":\"speech.audio.delta\",\"audio\":\"" +
		base64.StdEncoding.EncodeToString([]byte{1, 2, 3, 4}) + "\"}\n\n" +
		"data: {\"type\":\"speech.audio.delta\",\"audio\":\"" +
		base64.StdEncoding.EncodeToString([]byte{5, 6}) + "\"}\n\n" +
		"data: {\"type\":\"speech.audio.done\",\"usage\":{}}\n\n"
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	mux.HandleFunc("/v1/audio/speech", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(400)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sseBody))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	a := &mediaState{core: &core{
		ctx:       context.Background(),
		cfg:       &config.Config{BreezePort: portFromURL(ts.URL)},
		engineMgr: modelengine.NewManager("", ""),
	}}
	a.activeTTSEngine = "breeze"
	a.activeTTSModel = "breeze-tts-2"
	// POST 走 eng.BaseURL（种子默认 8012）——覆写到测试服务端口。
	// Enabled 必须显式 true：SaveEngine 部分更新对 bool 无三态，零值即写 false。
	if err := a.engineMgr.SaveEngine(modelengine.EngineConfig{ID: "breeze", BaseURL: ts.URL + "/v1", Enabled: true}); err != nil {
		t.Fatal(err)
	}

	// 订阅 tts-stream（httpbridge 全局 hub）
	tsHub := httptest.NewServer(httpbridge.New(nil).Handler())
	defer tsHub.Close()
	eventCh := make(chan string, 64)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		req, _ := http.NewRequestWithContext(ctx, "GET", tsHub.URL+"/api/stream?id=tts-stream", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			if line := sc.Text(); strings.HasPrefix(line, "data: ") {
				select {
				case eventCh <- strings.TrimPrefix(line, "data: "):
				default:
				}
			}
		}
	}()
	// 等 SSE connected
	waitStreamEvent(t, eventCh, func(m map[string]interface{}) bool {
		_, ok := m["id"]
		return ok
	}, "SSE connected")

	if !a.streamBreezeTTS("你好，流播测试。") {
		t.Fatal("应接管本次朗读")
	}
	// 收事件：2 chunk（audio/wav）+ 1 done
	var chunks, dones int
	deadline := time.After(5 * time.Second)
	for chunks < 2 || dones < 1 {
		select {
		case raw := <-eventCh:
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(raw), &m); err != nil {
				continue
			}
			switch m["type"] {
			case "chunk":
				chunks++
				if m["mimeType"] != "audio/wav" || m["engine"] != "breeze" {
					t.Errorf("chunk 字段不符: %v", m)
				}
			case "done":
				dones++
			}
		case <-deadline:
			t.Fatalf("事件不全: chunks=%d dones=%d", chunks, dones)
		}
	}
}

// TestStreamBreezeTTS_Fallback 服务不可达 → 未接管（调用方走回退逐句链）。
// ensure/probe 双桩隔离：真装机单测绝不能拉起真实 GPU 服务。
func TestStreamBreezeTTS_Fallback(t *testing.T) {
	origEnsure, origProbe := breezeStreamEnsure, breezeStreamProbe
	origInterval, origAttempts := breezeStreamPollInterval, breezeStreamPollAttempts
	breezeStreamEnsure = func(_ *mediaState) {}
	breezeStreamProbe = func(_ *mediaState) bool { return false }
	breezeStreamPollAttempts = 1
	t.Cleanup(func() {
		breezeStreamEnsure, breezeStreamProbe = origEnsure, origProbe
		breezeStreamPollInterval, breezeStreamPollAttempts = origInterval, origAttempts
	})

	a := &mediaState{core: &core{
		cfg:       &config.Config{BreezePort: 1},
		engineMgr: modelengine.NewManager("", ""),
	}}
	a.activeTTSEngine = "breeze"
	if a.streamBreezeTTS("text") {
		t.Fatal("服务不可达应返回 false（未接管）")
	}
}
