package modelengine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// TestFetchGLMCatalogRemote 远程拉取单次语义：v2 响应（顶层 version 存在）
// 原样写缓存；同版本跳过写入；裸数组（无 version）/坏 JSON/非 200 一律
// 拒绝采纳且不动缓存。
func TestFetchGLMCatalogRemote(t *testing.T) {
	dir := t.TempDir()
	cache := filepath.Join(dir, "glm_catalog_remote.json")
	client := &http.Client{Timeout: 2 * time.Second}

	v5a := []byte(`{"version":5,"updated":"2026-09-01","models":[{"id":"glm-from-remote-a"}]}`)
	v5b := []byte(`{"version":5,"updated":"2026-09-02","models":[{"id":"glm-from-remote-b"}]}`)
	current := v5a
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(current)
	}))
	defer srv.Close()

	// ① v2 响应：写缓存（原样落盘）
	if ok := fetchGLMCatalogRemoteOnce(client, srv.URL, cache); !ok {
		t.Fatal("v2 响应应被采纳")
	}
	got, err := os.ReadFile(cache)
	if err != nil || string(got) != string(v5a) {
		t.Fatalf("缓存内容 = %q (err=%v), want 原样 v5a", string(got), err)
	}
	if v := glmCatalogCacheVersion(cache); v != 5 {
		t.Errorf("缓存版本 = %d, want 5", v)
	}

	// ② 同版本不同内容：跳过写入（缓存仍为 v5a）
	current = v5b
	if ok := fetchGLMCatalogRemoteOnce(client, srv.URL, cache); !ok {
		t.Fatal("同版本拉取应视为成功")
	}
	if got, _ = os.ReadFile(cache); string(got) != string(v5a) {
		t.Fatalf("同版本不应改写缓存, got %q", string(got))
	}

	// ③ 裸数组（顶层无 version）：拒绝采纳，缓存不变
	current = []byte(`[{"id":"glm-x"}]`)
	if ok := fetchGLMCatalogRemoteOnce(client, srv.URL, cache); ok {
		t.Fatal("无 version 响应应被拒绝")
	}
	if got, _ = os.ReadFile(cache); string(got) != string(v5a) {
		t.Fatalf("被拒响应不应改写缓存, got %q", string(got))
	}

	// ④ 坏 JSON：拒绝
	current = []byte(`{not-json`)
	if ok := fetchGLMCatalogRemoteOnce(client, srv.URL, cache); ok {
		t.Fatal("坏 JSON 应被拒绝")
	}

	// ⑤ 非 200：拒绝
	errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer errSrv.Close()
	if ok := fetchGLMCatalogRemoteOnce(client, errSrv.URL, cache); ok {
		t.Fatal("非 200 应被拒绝")
	}
	if got, _ = os.ReadFile(cache); string(got) != string(v5a) {
		t.Fatalf("失败拉取不应改写缓存, got %q", string(got))
	}
}

// TestGLMCatalogRemoteCachePriority 生效优先级：覆盖文件 > 远程缓存 > 内嵌；
// 坏缓存该层被忽略并热恢复；目录价经 estimatePrice 生效；glmCatalogInfo
// 透传版本与来源。
func TestGLMCatalogRemoteCachePriority(t *testing.T) {
	defer setGLMCatalogPath("")
	defer setGLMCatalogRemotePath("")
	dir := t.TempDir()
	remote := filepath.Join(dir, "glm_catalog_remote.json")
	ov := filepath.Join(dir, "override.json")

	base := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	writeAt := func(path, content string, mod time.Time) {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, mod, mod); err != nil {
			t.Fatal(err)
		}
	}

	// 远程缓存层：glm-5.2 带国内价（压过内置表回退值 1.4/4.4 USD）+ 新条目
	writeAt(remote, `{"version":7,"updated":"2026-09-01","models":[{"id":"glm-5.2","price_in":2,"price_out":6,"currency":"CNY"},{"id":"glm-remote-only"}]}`, base)
	setGLMCatalogRemotePath(remote)
	models := glmStaticModels()
	if len(models) != 46 {
		t.Fatalf("远程合并后数量 = %d, want 46", len(models))
	}
	if p := estimatePrice("glm", "glm-5.2"); p.InputPerM != 2 || p.OutputPerM != 6 || p.Currency != "CNY" {
		t.Errorf("远程目录价应生效 = %+v, want 2/6 CNY", p)
	}
	version, source := glmCatalogInfo()
	if version != "7" || source != "remote 7" {
		t.Errorf("info = %q/%q, want 7/remote 7", version, source)
	}

	// 覆盖文件层盖在远程之上：同 ID 价格以覆盖为准，远程-only 条目仍在
	writeAt(ov, `{"version":9,"models":[{"id":"glm-5.2","price_in":3,"price_out":7,"currency":"CNY"}]}`, base.Add(time.Minute))
	setGLMCatalogPath(ov)
	if p := estimatePrice("glm", "glm-5.2"); p.InputPerM != 3 || p.Currency != "CNY" {
		t.Errorf("覆盖价应压过远程价 = %+v, want 3 CNY", p)
	}
	if p := estimatePrice("glm", "glm-remote-only"); p.Currency != "" {
		t.Errorf("远程-only 条目不应消失, price=%+v", p)
	}
	version, source = glmCatalogInfo()
	if version != "9" || source != "override" {
		t.Errorf("info = %q/%q, want 9/override", version, source)
	}

	// 远程缓存损坏：该层被忽略（glm-remote-only 消失），覆盖层仍生效
	writeAt(remote, `{bad json`, base.Add(2*time.Minute))
	if models = glmStaticModels(); len(models) != 45 {
		t.Errorf("坏缓存被忽略后数量 = %d, want 45", len(models))
	}
	if p := estimatePrice("glm", "glm-5.2"); p.InputPerM != 3 {
		t.Errorf("覆盖层应不受坏缓存影响 = %+v, want 3 CNY", p)
	}

	// 缓存恢复（mtime 变化）：热重载生效
	writeAt(remote, `{"version":8,"models":[{"id":"glm-remote-only"}]}`, base.Add(3*time.Minute))
	if models = glmStaticModels(); len(models) != 46 {
		t.Errorf("缓存热恢复后数量 = %d, want 46", len(models))
	}
	version, source = glmCatalogInfo()
	if version != "9" || source != "override" {
		t.Errorf("覆盖仍在时 info = %q/%q, want 9/override", version, source)
	}
}

// TestStartGLMCatalogRemote Manager 级集成：url 空=拉取禁用（缓存路径仍
// 注入）；非空=启动即异步拉取落盘，目录价热生效；Stop 幂等且循环退出。
func TestStartGLMCatalogRemote(t *testing.T) {
	defer setGLMCatalogPath("")
	defer setGLMCatalogRemotePath("")
	dir := t.TempDir()
	cache := filepath.Join(dir, "glm_catalog_remote.json")

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(`{"version":11,"updated":"2026-09-02","models":[{"id":"glm-remote-priced","price_in":2,"price_out":5,"currency":"CNY"}]}`))
	}))
	defer srv.Close()

	// ① url 空：不发起任何请求
	m := NewManager("", "")
	m.StartGLMCatalogRemote(context.Background(), "", cache)
	time.Sleep(80 * time.Millisecond)
	if n := atomic.LoadInt32(&hits); n != 0 {
		t.Fatalf("url 空不应发起请求, hits=%d", n)
	}
	m.StopGLMCatalogRemote() // 未启动循环：空操作不 panic

	// ② 非空 url：启动即异步拉取，缓存落盘且估算热生效
	m2 := NewManager("", "")
	m2.StartGLMCatalogRemote(context.Background(), srv.URL, cache)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(cache); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, err := os.Stat(cache); err != nil {
		t.Fatal("异步拉取未落盘缓存")
	}
	if atomic.LoadInt32(&hits) < 1 {
		t.Error("应发起至少一次请求")
	}
	if p := estimatePrice("glm", "glm-remote-priced"); p.InputPerM != 2 || p.OutputPerM != 5 || p.Currency != "CNY" {
		t.Errorf("远程价应热生效 = %+v, want 2/5 CNY", p)
	}

	// ③ Stop：循环退出（不再有新请求）且幂等
	m2.StopGLMCatalogRemote()
	n1 := atomic.LoadInt32(&hits)
	time.Sleep(80 * time.Millisecond)
	if atomic.LoadInt32(&hits) != n1 {
		t.Error("Stop 后不应继续拉取")
	}
	m2.StopGLMCatalogRemote()
}
