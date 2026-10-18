package app

// strata_proc_test.go — Strata 引擎生命周期测试。只测零副作用面：探活解析、
// 外部项目路径与启动命令契约（httptest / 纯函数注入，不碰真实 8091、不拉模型）。

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/config"
)

func TestStrataProbeAt_ParsesServedModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"qwen3.6-35b-a3b-uncensored-q4_k_p","status":{"value":"loaded"}}]}`))
	}))
	defer srv.Close()

	served, ok := strataProbeAt(srv.URL + "/v1")
	if !ok {
		t.Fatal("expected probe ok on 200")
	}
	if len(served) != 1 || served[0] != "qwen3.6-35b-a3b-uncensored-q4_k_p" {
		t.Fatalf("unexpected served models: %v", served)
	}
}

func TestStrataProbeAt_NotOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	if _, ok := strataProbeAt(srv.URL + "/v1"); ok {
		t.Fatal("expected probe not-ok on 404")
	}
}

func TestStrataEngineStatus_Shape(t *testing.T) {
	a := &App{}
	st := a.StrataEngineStatus()
	if _, ok := st["running"]; !ok {
		t.Fatalf("status missing running key: %v", st)
	}
	if st["port"] != strataPort {
		t.Fatalf("unexpected port: %v", st["port"])
	}
	if st["project"] != config.DefaultStrataPath {
		t.Fatalf("unexpected project: %v", st["project"])
	}
	if st["default_profile"] != "qwen36_uncensored" {
		t.Fatalf("unexpected default profile: %v", st["default_profile"])
	}
	profiles, ok := st["profiles"].([]map[string]string)
	if !ok || len(profiles) != len(strataProfiles) {
		t.Fatalf("profiles payload = %#v", st["profiles"])
	}
}

func TestStrataProjectPath_DefaultAndConfigured(t *testing.T) {
	if got := (&App{}).strataProjectPath(); got != config.DefaultStrataPath {
		t.Fatalf("nil config project = %q, want %q", got, config.DefaultStrataPath)
	}
	a := &App{core: &core{cfg: &config.Config{StrataPath: `D:\engines\strata`}}}
	if got := a.strataProjectPath(); got != `D:\engines\strata` {
		t.Fatalf("configured project = %q", got)
	}
}

func TestStrataServeInvocation_ExternalProjectContract(t *testing.T) {
	project := `C:\AI\strata\repo`
	for _, profile := range []string{"qwen36_uncensored"} {
		bin, argv, cfgPath, err := strataServeInvocation(project, profile, 8091)
		if err != nil {
			t.Fatalf("invocation(%s): %v", profile, err)
		}
		wantPython := filepath.Join(project, strataPythonEntry)
		if bin != wantPython {
			t.Fatalf("launcher = %q, want %q", bin, wantPython)
		}
		wantServer := filepath.Join(project, strataServeEntry)
		if argv[0] != wantServer {
			t.Fatalf("server arg = %q, want %q", argv[0], wantServer)
		}
		joined := strings.Join(argv, " ")
		for _, want := range []string{
			"--engine strata",
			"--config " + cfgPath,
			"--port 8091",
		} {
			if !strings.Contains(joined, want) {
				t.Fatalf("invocation missing %q: %s", want, joined)
			}
		}
		if !strings.HasPrefix(cfgPath, project) {
			t.Fatalf("config outside project: %s", cfgPath)
		}
	}
}

func TestStrataServeInvocation_RejectsUnknownProfile(t *testing.T) {
	if _, _, _, err := strataServeInvocation(`C:\AI\strata\repo`, "turbo", 8091); err == nil {
		t.Fatal("unknown profile must be rejected")
	}
}

func TestStrataProfileFromModels(t *testing.T) {
	cases := []struct {
		profile string
		model   string
		want    string
	}{
		{"qwen36_uncensored", "qwen3.6-35b-a3b-uncensored-q4_k_p", "qwen36_uncensored"},
	}
	for _, tc := range cases {
		if got := strataProfileFromModels([]string{tc.model}); got != tc.want {
			t.Fatalf("%s: profile = %q, want %q", tc.profile, got, tc.want)
		}
	}
	// 已退役档位（flash IQ2 / 普通 UD）不再映射：gaea 目录只认 uncensored 档。
	for _, model := range []string{"qwen3.8-flash-next-iq2_xs", "qwen3.6-35b-a3b-ud-q4_k_m", "unknown-model"} {
		if got := strataProfileFromModels([]string{model}); got != "" {
			t.Fatalf("retired/unknown model %s: profile = %q, want empty", model, got)
		}
	}
}

func TestReadLogTail(t *testing.T) {
	path := filepath.Join(t.TempDir(), "engine.log")
	if err := os.WriteFile(path, []byte("first line\nlast line: missing model\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readLogTail(path, 64); got != "first line\nlast line: missing model" {
		t.Fatalf("tail = %q", got)
	}
}
