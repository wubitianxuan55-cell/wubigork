package browser

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// ── Edge 定位器 ─────────────────────────────────────────────────────────

// TestFindEdge_EnvOverride 环境变量最高优先，且不校验存在性。
func TestFindEdge_EnvOverride(t *testing.T) {
	t.Setenv("GAEA_BROWSER_EXE", filepath.Join(t.TempDir(), "fake-msedge.exe"))
	exe, err := FindEdge()
	if err != nil {
		t.Fatalf("FindEdge: %v", err)
	}
	if exe != os.Getenv("GAEA_BROWSER_EXE") {
		t.Fatalf("exe = %q, want env 值原样返回", exe)
	}
}

// TestFindEdge 表驱动覆盖候选路径 → PATH → 失败全链（getenv/lookPath 注入）。
func TestFindEdge(t *testing.T) {
	dir := t.TempDir()
	exe86 := filepath.Join(dir, "x86", "Microsoft", "Edge", "Application", "msedge.exe")
	exe64 := filepath.Join(dir, "x64", "Microsoft", "Edge", "Application", "msedge.exe")
	for _, p := range []string{exe86, exe64} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("MZ"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cases := []struct {
		name     string
		env      map[string]string
		exeEnv   string
		lookPath func(string) (string, error)
		want     string
		wantErr  bool
	}{
		{
			name: "x86 候选存在优先",
			env:  map[string]string{"ProgramFiles(x86)": dir + `\x86`, "ProgramFiles": dir + `\x64`},
			want: exe86,
		},
		{
			name: "缺 x86 回退 ProgramFiles",
			env:  map[string]string{"ProgramFiles": dir + `\x64`},
			want: exe64,
		},
		{
			name:     "候选皆缺回退 PATH",
			env:      map[string]string{},
			lookPath: func(string) (string, error) { return `C:\edge\msedge.exe`, nil },
			want:     `C:\edge\msedge.exe`,
		},
		{
			name:     "全链失败报错",
			env:      map[string]string{},
			lookPath: func(string) (string, error) { return "", exec.ErrNotFound },
			wantErr:  true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			getenv := func(k string) string { return tc.env[k] }
			look := tc.lookPath
			if look == nil {
				look = func(string) (string, error) { return "", exec.ErrNotFound }
			}
			exe, err := findEdge(getenv, look)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got %q", exe)
				}
				return
			}
			if err != nil {
				t.Fatalf("findEdge: %v", err)
			}
			if exe != tc.want {
				t.Fatalf("exe = %q, want %q", exe, tc.want)
			}
		})
	}
}

// ── URL 白名单 ──────────────────────────────────────────────────────────

// TestValidateURL navigate 只接受 http/https；loopback 天然放行。
func TestValidateURL(t *testing.T) {
	cases := []struct {
		url     string
		wantErr bool
	}{
		{"http://example.com/a?b=1", false},
		{"https://example.com", false},
		{"http://127.0.0.1:8080/", false},
		{"http://localhost:3000/x", false},
		{"file:///C:/Windows/system.ini", true},
		{"javascript:alert(1)", true},
		{"data:text/html,<b>x</b>", true},
		{"about:blank", true},
		{"ftp://example.com/x", true},
		{"", true},
		{"//example.com/no-scheme", true},
		{"://broken", true},
	}
	for _, c := range cases {
		err := ValidateURL(c.url)
		if (err != nil) != c.wantErr {
			t.Errorf("ValidateURL(%q) err = %v, wantErr %v", c.url, err, c.wantErr)
		}
	}
}

// TestFreePort 取号端口可立即绑定复用（基本健全性）。
func TestFreePort(t *testing.T) {
	port, err := freePort()
	if err != nil {
		t.Fatal(err)
	}
	if port <= 0 {
		t.Fatalf("port = %d", port)
	}
	l, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("归还端口后无法重新监听: %v", err)
	}
	_ = l.Close()
}

// ── 无注入时的守门行为 ──────────────────────────────────────────────────

// TestNavigateRejectsSchemeBeforeLaunch 非法 URL 在任何浏览器拉起之前被拒。
func TestNavigateRejectsSchemeBeforeLaunch(t *testing.T) {
	m := NewManager(Options{InjectHTTPBase: "http://127.0.0.1:1"}) // 探活必败的假端点
	for _, raw := range []string{"file:///C:/x", "javascript:alert(1)", "about:blank", ""} {
		if _, err := m.Navigate(context.Background(), raw, 5); err == nil {
			t.Errorf("Navigate(%q) 应拒绝", raw)
		}
	}
}

// TestNavTimeoutClamp 等待时长归一化。
func TestNavTimeoutClamp(t *testing.T) {
	cases := []struct {
		secs int
		want time.Duration
	}{
		{0, 20 * time.Second},
		{2, 5 * time.Second},
		{30, 30 * time.Second},
		{600, 120 * time.Second},
	}
	for _, c := range cases {
		if got := navTimeout(c.secs); got != c.want {
			t.Errorf("navTimeout(%d) = %v, want %v", c.secs, got, c.want)
		}
	}
}
