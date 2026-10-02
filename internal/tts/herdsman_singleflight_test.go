package tts

// herdsman_singleflight_test.go — AP6-10：SupportedSpeakers 的 /v1/audio/info
// 探测必须 ①并发只发一次（single-flight）；②失败不缓存（可重试）；
// ③HTTP 不在锁内（不阻塞其他调用方）；④跟随者有界等待（不挂死合成链）。

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// dropSpeakerCache 清理全局缓存/在跑标记，避免用例之间互相污染（包级全局）。
func dropSpeakerCache(model string) {
	speakerCacheMu.Lock()
	delete(speakerCache, model)
	delete(speakerFlights, model)
	speakerCacheMu.Unlock()
}

// TestSupportedSpeakersSingleFlight 并发 N 个 SupportedSpeakers 只发 1 次 HTTP，
// 且所有并发调用拿到同一份结果。
func TestSupportedSpeakersSingleFlight(t *testing.T) {
	const model = "qwen3-tts-customvoice-sf1"
	dropSpeakerCache(model)
	defer dropSpeakerCache(model)

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/audio/info") {
			http.NotFound(w, r)
			return
		}
		atomic.AddInt32(&hits, 1)
		time.Sleep(120 * time.Millisecond) // 制造并发窗口：旧实现会排队发 N 次
		_, _ = w.Write([]byte(`{"supported_speakers":["serena","vivian"]}`))
	}))
	defer srv.Close()

	h := NewHerdsmanTTS(srv.URL+"/v1", model, "cherry")
	const n = 8
	start := make(chan struct{})
	results := make([][]string, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i] = h.SupportedSpeakers()
		}(i)
	}
	close(start)
	wg.Wait()

	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("并发 %d 次 SupportedSpeakers 应只发 1 次 HTTP，实际 %d 次", n, got)
	}
	for i, got := range results {
		if len(got) != 2 || got[0] != "serena" || got[1] != "vivian" {
			t.Fatalf("第 %d 个并发调用结果不符：%v", i, got)
		}
	}
}

// TestSupportedSpeakersFailureNotCached 首个探测失败（HTTP 500）不得写缓存：
// 第二次调用必须重新探测（可重试），且第二次成功后正常回填。
func TestSupportedSpeakersFailureNotCached(t *testing.T) {
	const model = "qwen3-tts-customvoice-sf2"
	dropSpeakerCache(model)
	defer dropSpeakerCache(model)

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{"supported_speakers":["aiden"]}`))
	}))
	defer srv.Close()

	h := NewHerdsmanTTS(srv.URL+"/v1", model, "cherry")
	if got := h.SupportedSpeakers(); got != nil {
		t.Fatalf("首次失败应返回 nil：%v", got)
	}
	got := h.SupportedSpeakers()
	if len(got) != 1 || got[0] != "aiden" {
		t.Fatalf("失败不得缓存：第二次应重试并拿到结果，got=%v hits=%d", got, atomic.LoadInt32(&hits))
	}
	if n := atomic.LoadInt32(&hits); n != 2 {
		t.Fatalf("失败后第二次必须重新探测（hits=2），实际 %d", n)
	}
	// 第三次：已缓存 → 不再发 HTTP。
	if got := h.SupportedSpeakers(); len(got) != 1 {
		t.Fatalf("缓存命中结果不符：%v", got)
	}
	if n := atomic.LoadInt32(&hits); n != 2 {
		t.Fatalf("成功后应命中缓存（hits=2），实际 %d", n)
	}
}

// TestSupportedSpeakersHTTPOutsideLock 慢 handler 期间 speakerCacheMu 必须可用
// （HTTP 不得在锁内）；同模型的另一个调用方走 single-flight 等待，不同模型的
// 调用方不被阻塞。
func TestSupportedSpeakersHTTPOutsideLock(t *testing.T) {
	const slowModel = "qwen3-tts-customvoice-sf3-slow"
	const fastModel = "qwen3-tts-customvoice-sf3-fast"
	dropSpeakerCache(slowModel)
	dropSpeakerCache(fastModel)
	defer dropSpeakerCache(slowModel)
	defer dropSpeakerCache(fastModel)

	inHandler := make(chan struct{}, 1)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Query().Get("model"), "slow") {
			inHandler <- struct{}{}
			<-release
		}
		_, _ = w.Write([]byte(`{"supported_speakers":["serena"]}`))
	}))
	defer srv.Close()

	slow := NewHerdsmanTTS(srv.URL+"/v1", slowModel, "cherry")
	fast := NewHerdsmanTTS(srv.URL+"/v1", fastModel, "cherry")

	done := make(chan []string, 1)
	go func() { done <- slow.SupportedSpeakers() }()
	select {
	case <-inHandler: // 探测已在 HTTP 中（旧实现此刻整段持有 speakerCacheMu）
	case <-time.After(2 * time.Second):
		t.Fatal("慢探测未进入 handler")
	}

	// ① 锁可用：HTTP 不在锁内（TryLock 命中即证明没有别的持有者）。
	if !speakerCacheMu.TryLock() {
		close(release)
		t.Fatal("HTTP 探测期间 speakerCacheMu 被整段持有（AP6-10 未修）")
	}
	speakerCacheMu.Unlock()

	// ② 不同模型不被阻塞：自己的探测立即完成。
	fastDone := make(chan []string, 1)
	go func() { fastDone <- fast.SupportedSpeakers() }()
	select {
	case got := <-fastDone:
		if len(got) != 1 || got[0] != "serena" {
			t.Fatalf("不同模型结果不符：%v", got)
		}
	case <-time.After(1 * time.Second):
		close(release)
		t.Fatal("不同模型的 SupportedSpeakers 被慢探测阻塞（锁面未收敛）")
	}

	close(release)
	select {
	case got := <-done:
		if len(got) != 1 {
			t.Fatalf("慢探测结果不符：%v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("慢探测未返回")
	}
}

// TestSupportedSpeakersFollowerBoundedWait 跟随者有界等待：探测迟迟不返回时，
// 跟随者按上限返回 nil（诚实降级），而不是无限挂住合成链。
func TestSupportedSpeakersFollowerBoundedWait(t *testing.T) {
	const model = "qwen3-tts-customvoice-sf4"
	dropSpeakerCache(model)
	defer dropSpeakerCache(model)

	old := speakerProbeWaitFn
	speakerProbeWaitFn = func(string) time.Duration { return 150 * time.Millisecond }
	defer func() { speakerProbeWaitFn = old }()

	inHandler := make(chan struct{}, 1)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inHandler <- struct{}{}
		<-release
		_, _ = w.Write([]byte(`{"supported_speakers":["serena"]}`))
	}))
	defer srv.Close()

	h := NewHerdsmanTTS(srv.URL+"/v1", model, "cherry")
	leader := make(chan []string, 1)
	go func() { leader <- h.SupportedSpeakers() }()
	select {
	case <-inHandler:
	case <-time.After(2 * time.Second):
		t.Fatal("探测未进入 handler")
	}

	start := time.Now()
	if got := h.SupportedSpeakers(); got != nil {
		t.Fatalf("跟随者超时应返回 nil：%v", got)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("跟随者未被有界等待约束：%v", elapsed)
	}
	close(release)
	select {
	case got := <-leader:
		if len(got) != 1 {
			t.Fatalf("leader 结果不符：%v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("leader 未返回")
	}
}

// TestSupportedSpeakersSingleFlightPerModel 不同模型的探测互不串行：两个模型
// 各自的 8 个并发调用 → 各 1 次 HTTP，总计 2 次。
func TestSupportedSpeakersSingleFlightPerModel(t *testing.T) {
	models := []string{"qwen3-tts-customvoice-sf5a", "qwen3-tts-customvoice-sf5b"}
	for _, m := range models {
		dropSpeakerCache(m)
		defer dropSpeakerCache(m)
	}
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		time.Sleep(80 * time.Millisecond)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"supported_speakers":["%s"]}`, r.URL.Query().Get("model"))))
	}))
	defer srv.Close()

	var wg sync.WaitGroup
	for _, m := range models {
		h := NewHerdsmanTTS(srv.URL+"/v1", m, "cherry")
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func(h *HerdsmanTTS, m string) {
				defer wg.Done()
				if got := h.SupportedSpeakers(); len(got) != 1 || got[0] != m {
					t.Errorf("模型 %s 结果不符：%v", m, got)
				}
			}(h, m)
		}
	}
	wg.Wait()
	if n := atomic.LoadInt32(&hits); n != 2 {
		t.Fatalf("两个模型各 4 个并发调用应共发 2 次 HTTP，实际 %d 次", n)
	}
}
