package app

// tianxuan_proc_test.go — 天璇引擎生命周期测试。只测零副作用面：探活解析、
// 外部项目路径与启动命令契约（httptest / 纯函数注入，不碰真实 8094、不拉模型）。

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/config"
)

func TestTianxuanProbeAt_ParsesServedModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"qwen3.6-35b-a3b-uncensored"},{"id":"mmproj"}]}`))
	}))
	defer srv.Close()

	served, ok := tianxuanProbeAt(srv.URL)
	if !ok {
		t.Fatal("expected probe ok on 200")
	}
	if len(served) != 2 || served[0] != "qwen3.6-35b-a3b-uncensored" {
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
	if st["project"] != config.DefaultTianxuanPath {
		t.Fatalf("unexpected project: %v", st["project"])
	}
	if st["default_profile"] != "uncensored" {
		t.Fatalf("unexpected default profile: %v", st["default_profile"])
	}
	profiles, ok := st["profiles"].([]map[string]string)
	if !ok || len(profiles) != len(tianxuanProfiles) {
		t.Fatalf("profiles payload = %#v", st["profiles"])
	}
	// running 的布尔值随本机 8094 真实状态而变，不断言方向。
}

func TestTianxuanProjectPath_DefaultAndConfigured(t *testing.T) {
	if got := (&App{}).tianxuanProjectPath(); got != config.DefaultTianxuanPath {
		t.Fatalf("nil config project = %q, want %q", got, config.DefaultTianxuanPath)
	}
	a := &App{core: &core{cfg: &config.Config{TianxuanPath: `D:\engines\tianxuan`}}}
	if got := a.tianxuanProjectPath(); got != `D:\engines\tianxuan` {
		t.Fatalf("configured project = %q", got)
	}
}

func TestTianxuanServeInvocation_ExternalProjectContract(t *testing.T) {
	project := `C:\AI\tianxuan`
	stdoutLog := filepath.Join(project, `logs\engine.out.log`)
	stderrLog := filepath.Join(project, `logs\engine.err.log`)
	for _, profile := range []string{"uncensored"} {
		bin, argv, err := tianxuanServeInvocation(project, profile, 8094, stdoutLog, stderrLog)
		if err != nil {
			t.Fatalf("invocation(%s): %v", profile, err)
		}
		if bin != "powershell.exe" {
			t.Fatalf("launcher = %q, want powershell.exe", bin)
		}
		joined := strings.Join(argv, " ")
		wantScript := filepath.Join(project, tianxuanServeScript)
		for _, want := range []string{
			"-File " + wantScript,
			"-Profile " + profile,
			"-Port 8094",
			"-StdoutLog " + stdoutLog,
			"-StderrLog " + stderrLog,
		} {
			if !strings.Contains(joined, want) {
				t.Fatalf("invocation missing %q: %s", want, joined)
			}
		}
		if strings.Contains(joined, "gufo-server.exe") || strings.Contains(joined, "--spec-type") {
			t.Fatalf("gaea must not duplicate engine binary/args: %s", joined)
		}
	}
}

func TestTianxuanServeInvocation_RejectsUnknownProfile(t *testing.T) {
	if _, _, err := tianxuanServeInvocation(`C:\AI\tianxuan`, "turbo", 8094, "out", "err"); err == nil {
		t.Fatal("unknown profile must be rejected")
	}
}

func TestParseListeningPortPID_ListenAddresses(t *testing.T) {
	out := `  TCP    127.0.0.1:18094      0.0.0.0:0              LISTENING       111
  TCP    0.0.0.0:8094          0.0.0.0:0              LISTENING       222
  TCP    [::]:8094             [::]:0                 LISTENING       333
  TCP    127.0.0.1:8094         127.0.0.1:50120        ESTABLISHED     444`
	if got := parseListeningPortPID(out, 8094); got != 222 {
		t.Fatalf("pid = %d, want first matching listener 222", got)
	}
}

func TestTianxuanProfileFromModels(t *testing.T) {
	cases := []struct {
		profile string
		model   string
		want    string
	}{
		{"uncensored", "qwen3.6-35b-a3b-uncensored", "uncensored"},
		// flash 档模型名同时含 uncensored 字样，flash 判定必须在前。
		{"flash", "qwen3.8-flash-next-uncensored", "flash"},
	}
	for _, tc := range cases {
		if got := tianxuanProfileFromModels([]string{tc.model}); got != tc.want {
			t.Fatalf("%s: profile = %q, want %q", tc.profile, got, tc.want)
		}
	}
	// 已退役档位（base/fast 普通 35B）不再映射：目录只留两个 uncensored 档。
	for _, model := range []string{"qwen3.6-35b-a3b", "Qwen3.6-35B-A3B-UD-Q4_K_M", "unknown-model"} {
		if got := tianxuanProfileFromModels([]string{model}); got != "" {
			t.Fatalf("retired/unknown model %s: profile = %q, want empty", model, got)
		}
	}
}
