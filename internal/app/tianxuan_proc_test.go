package app

// tianxuan_proc_test.go — 天璇引擎生命周期测试。只测零副作用面：探活解析
// （httptest 注入，不碰真实 8094）与 Start 的早退分支（binary/model 缺失在
// 机器上不可假设——用可注入性不足即不测的口径，真实启停由走查承担）。

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTianxuanProbeAt_ParsesServedModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"Qwen3.8-27B-DFlash2"},{"id":"mmproj"}]}`))
	}))
	defer srv.Close()

	served, ok := tianxuanProbeAt(srv.URL)
	if !ok {
		t.Fatal("expected probe ok on 200")
	}
	if len(served) != 2 || served[0] != "Qwen3.8-27B-DFlash2" {
		t.Fatalf("unexpected served models: %v", served)
	}
}

func TestTianxuanProbeAt_NotOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	if _, ok := tianxuanProbeAt(srv.URL); ok {
		t.Fatal("expected probe not-ok on 404")
	}
}

func TestTianxuanProbeAt_Unreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // 立即关闭：端口不可达
	if _, ok := tianxuanProbeAt(srv.URL); ok {
		t.Fatal("expected probe not-ok on unreachable")
	}
}

func TestTianxuanEngineStatus_Shape(t *testing.T) {
	a := &App{}
	st := a.TianxuanEngineStatus()
	if _, ok := st["running"]; !ok {
		t.Fatalf("status missing running key: %v", st)
	}
	if st["port"] != tianxuanPort {
		t.Fatalf("unexpected port: %v", st["port"])
	}
	// running 的布尔值随本机 8094 真实状态而变，不断言方向。
}

func TestTianxuanServeArgs_UseLookupPreset(t *testing.T) {
	args := strings.Join(tianxuanServeArgs(), " ")
	for _, want := range []string{
		"--speculative dflash2",
		"--draft-tokens 7",
		"--served-model-name Qwen3.8-27B-DFlash2",
	} {
		if !strings.Contains(args, want) {
			t.Fatalf("serve preset missing %q: %s", want, args)
		}
	}
	if tianxuanLookupEnv != "GUFO_QWEN27_LOOKUP=3" {
		t.Fatalf("unexpected lookup env: %q", tianxuanLookupEnv)
	}
}
