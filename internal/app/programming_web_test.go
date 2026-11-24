package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func resetProgrammingWebTestState(t *testing.T) {
	t.Helper()
	programmingWebMu.Lock()
	programmingWebCmd = nil
	programmingWebJob = 0
	programmingWebPID = 0
	programmingWebStartedAt = time.Time{}
	programmingWebToken = ""
	programmingWebStarting = false
	programmingWebDetached = false
	programmingWebMu.Unlock()
}

func TestExtractProgrammingWebToken(t *testing.T) {
	line := "\x1b[?25h(node:1) warning\ndsh web: http://127.0.0.1:3091/?token=Abc_123-test\n"
	if got := extractProgrammingWebToken(line); got != "Abc_123-test" {
		t.Fatalf("token = %q, want Abc_123-test", got)
	}
	if got := extractProgrammingWebToken("dsh web: no token here"); got != "" {
		t.Fatalf("missing token should be empty, got %q", got)
	}
}

func TestProgrammingWebTailLines(t *testing.T) {
	got := programmingWebTailLines("one\r\ntwo\nthree\n", 2)
	if len(got) != 2 || got[0] != "two" || got[1] != "three" {
		t.Fatalf("tail = %#v", got)
	}
}

func TestProgrammingWebLogRedactsTokenInPlace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dsh-web.log")
	token := "Abc_123-test-token"
	if err := os.WriteFile(path, []byte("start\ndsh web: http://dsh.wails.localhost:3091/?token="+token+"\nnext\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := captureProgrammingWebToken(path, 0); got != token {
		t.Fatalf("captured token = %q, want %q", got, token)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if strings.Contains(text, token) || !strings.Contains(text, "?token=<redacted>") {
		t.Fatalf("token not redacted: %q", text)
	}
	if len(raw) != len("start\ndsh web: http://dsh.wails.localhost:3091/?token="+token+"\nnext\n") {
		t.Fatalf("in-place redaction changed log length: %d", len(raw))
	}
}

func TestProgrammingWebAdoptsDetachedSession(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dsh")
	t.Setenv("GAEA_DSH_ROOT", root)
	started := time.Now().Add(-45 * time.Second).Truncate(time.Second)
	if err := saveProgrammingWebSession(programmingWebSessionPath(root), programmingWebSession{
		PID:       4242,
		Token:     "persisted-token",
		Root:      root,
		StartedAt: started,
	}); err != nil {
		t.Fatal(err)
	}
	resetProgrammingWebTestState(t)
	t.Cleanup(func() { resetProgrammingWebTestState(t) })
	oldPortOpen := programmingWebPortOpenFn
	oldAlive := programmingWebPIDAliveFn
	programmingWebPortOpenFn = func() bool { return true }
	programmingWebPIDAliveFn = func(pid int) bool { return pid == 4242 }
	t.Cleanup(func() {
		programmingWebPortOpenFn = oldPortOpen
		programmingWebPIDAliveFn = oldAlive
	})

	status := (&App{}).GetProgrammingWebStatus()
	if status["owned"] != true || status["running"] != true || status["detached"] != true {
		t.Fatalf("detached session not adopted: %+v", status)
	}
	if got := status["url"]; got != programmingWebBaseURL+"/?token=persisted-token" {
		t.Fatalf("adopted url = %v", got)
	}
	if uptime, _ := status["uptime_s"].(int64); uptime < 40 {
		t.Fatalf("adopted uptime = %d, want >= 40", uptime)
	}
}

func TestOpenProgrammingWebExternalPersistsSession(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dsh")
	t.Setenv("GAEA_DSH_ROOT", root)
	resetProgrammingWebTestState(t)
	t.Cleanup(func() { resetProgrammingWebTestState(t) })
	oldPortOpen := programmingWebPortOpenFn
	oldAlive := programmingWebPIDAliveFn
	oldRelease := programmingWebReleaseTrackedFn
	programmingWebPortOpenFn = func() bool { return true }
	programmingWebPIDAliveFn = func(pid int) bool { return pid == 5151 }
	released := false
	programmingWebReleaseTrackedFn = func(_ *exec.Cmd, job uintptr) error {
		if job != 77 {
			t.Fatalf("release job = %d, want 77", job)
		}
		released = true
		return nil
	}
	t.Cleanup(func() {
		programmingWebPortOpenFn = oldPortOpen
		programmingWebPIDAliveFn = oldAlive
		programmingWebReleaseTrackedFn = oldRelease
	})

	started := time.Now().Add(-time.Minute).Truncate(time.Second)
	programmingWebMu.Lock()
	programmingWebCmd = exec.Command("cmd")
	programmingWebJob = 77
	programmingWebPID = 5151
	programmingWebStartedAt = started
	programmingWebToken = "external-token"
	programmingWebMu.Unlock()

	url, err := (&App{}).OpenProgrammingWebExternal()
	if err != nil {
		t.Fatal(err)
	}
	if url != programmingWebBaseURL+"/?token=external-token" || !released {
		t.Fatalf("url=%q released=%v", url, released)
	}
	session, ok, err := loadProgrammingWebSession(programmingWebSessionPath(root))
	if err != nil || !ok {
		t.Fatalf("session load ok=%v err=%v", ok, err)
	}
	if session.PID != 5151 || session.Token != "external-token" || session.Root != root {
		t.Fatalf("session = %+v", session)
	}
	programmingWebMu.Lock()
	defer programmingWebMu.Unlock()
	if programmingWebJob != 0 || !programmingWebDetached {
		t.Fatalf("job=%d detached=%v", programmingWebJob, programmingWebDetached)
	}
}

func TestProgrammingWebPreflightIsolatedInstall(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	agentsHome := filepath.Join(root, "home", "agents")
	base := filepath.Join(root, "node_modules", "@deepseek-ai")
	// 版本用期望常量拼接（不是钉字面量）——DSH 升级只抬 programmingWebExpected 一处。
	writeTestFile(t, filepath.Join(base, "dsh", "package.json"), `{"version":"`+programmingWebExpected+`"}`)
	writeTestFile(t, filepath.Join(base, "dsh", "lib", "bin.js"), "// cli")
	writeTestFile(t, filepath.Join(home, "profiles", "web", "package.json"), `{"name":"dsh-profile-web"}`)
	writeTestFile(t, filepath.Join(home, "profiles", "web", "node_modules", "dsh-wubi", "package.json"), `{"name":"dsh-wubi","version":"2.0.0"}`)
	// REA 就绪判定挂红队预设的 rea-mcp 行（2026-11-10 前查 cordis.patch.yml 标记行）。
	writeTestFile(t, filepath.Join(home, ".agent-presets", "redteam", "agent.cordis.yml"),
		"- id: rea-mcp\n  command: node\n  serverName: rea\n")
	writeTestFile(t, filepath.Join(root, "rea", "node_modules", "rea-agents", "scripts", "rea.mjs"), "// rea")
	writeTestFile(t, filepath.Join(agentsHome, "skills", "reverse-engineer-anything", "SKILL.md"), "# REA")

	t.Setenv("GAEA_DSH_ROOT", root)
	t.Setenv("GAEA_DSH_HOME", home)
	t.Setenv("GAEA_DSH_AGENTS_HOME", agentsHome)
	oldPortOpen := programmingWebPortOpenFn
	programmingWebPortOpenFn = func() bool { return false }
	t.Cleanup(func() { programmingWebPortOpenFn = oldPortOpen })

	pre := (&App{}).GetProgrammingWebPreflight()
	if pre["all_ready"] != true {
		t.Fatalf("expected all_ready=true, got %+v", pre)
	}
	if pre["isolated"] != true || pre["version_ok"] != true {
		t.Fatalf("isolation/version checks failed: %+v", pre)
	}
	if pre["rea_ready"] != true || pre["agents_home"] != agentsHome {
		t.Fatalf("REA/agents checks failed: %+v", pre)
	}
}

func TestProgrammingWebEnvKeepsAgentsHomeIsolated(t *testing.T) {
	root := filepath.Join(t.TempDir(), "dsh")
	agentsHome := filepath.Join(root, "home", "agents")
	t.Setenv("GAEA_DSH_AGENTS_HOME", agentsHome)

	env := programmingWebEnv(root)
	want := "DSH_AGENTS_HOME=" + agentsHome
	for _, entry := range env {
		if entry == want {
			return
		}
	}
	t.Fatalf("%s missing from environment", want)
}

func TestProgrammingWebLiveStart(t *testing.T) {
	if os.Getenv("GAEA_DSH_LIVE_TEST") != "1" {
		t.Skip("set GAEA_DSH_LIVE_TEST=1 to start the isolated local DSH Web")
	}
	a := &App{}
	if status := a.GetProgrammingWebStatus(); status["running"] == true {
		t.Fatalf("port %d must be free before the live test: %+v", programmingWebPort, status)
	}
	if err := a.StartProgrammingWeb(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	t.Cleanup(func() {
		if err := a.StopProgrammingWeb(); err != nil {
			t.Logf("stop failed: %v", err)
		}
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) && programmingWebPortOpenFn() {
			time.Sleep(100 * time.Millisecond)
		}
	})
	status := a.GetProgrammingWebStatus()
	if status["owned"] != true || status["running"] != true {
		t.Fatalf("owned running status expected: %+v", status)
	}
	url, _ := status["url"].(string)
	if !strings.Contains(url, "token=") {
		t.Fatalf("tokenized URL expected: %+v", status)
	}
	tail := a.ProgrammingWebLogTail(30)
	if tail["exists"] != true {
		t.Fatalf("log expected after start: %+v", tail)
	}
	for _, line := range tail["lines"].([]string) {
		if strings.Contains(line, "?token=") && !strings.Contains(line, "?token=<redacted>") {
			t.Fatalf("log leaked the launch token: %q", line)
		}
	}
}

func TestProgrammingWebExternalHelper(t *testing.T) {
	if os.Getenv("GAEA_DSH_EXTERNAL_HELPER") != "1" {
		t.Skip("external-session helper")
	}
	a := &App{}
	if err := a.StartProgrammingWeb(); err != nil {
		t.Fatalf("start failed: %v", err)
	}
	url, err := a.OpenProgrammingWebExternal()
	if err != nil {
		t.Fatalf("open external failed: %v", err)
	}
	if !strings.Contains(url, "token=") {
		t.Fatalf("tokenized url expected: %q", url)
	}
	fmt.Println("external DSH Web ready")
}

// The helper test process exits without stopping DSH Web. The parent then
// verifies the detached service, token and PID survived and can be re-adopted.
func TestProgrammingWebExternalSurvivesOwnerExit(t *testing.T) {
	if os.Getenv("GAEA_DSH_LIVE_TEST") != "1" {
		t.Skip("set GAEA_DSH_LIVE_TEST=1 to start the isolated local DSH Web")
	}
	resetProgrammingWebTestState(t)
	t.Cleanup(func() { resetProgrammingWebTestState(t) })
	if programmingWebPortOpenFn() {
		t.Fatalf("port %d must be free before the live test", programmingWebPort)
	}

	helper := exec.Command(os.Args[0], "-test.run=^TestProgrammingWebExternalHelper$", "-test.v")
	helper.Env = append(os.Environ(), "GAEA_DSH_EXTERNAL_HELPER=1")
	out, err := helper.CombinedOutput()
	if err != nil {
		t.Fatalf("external helper failed: %v\n%s", err, out)
	}

	deadline := time.Now().Add(5 * time.Second)
	for !programmingWebPortOpenFn() && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	status := (&App{}).GetProgrammingWebStatus()
	if status["running"] != true || status["owned"] != true || status["detached"] != true {
		t.Fatalf("detached DSH Web was not re-adopted: %+v\n%s", status, out)
	}
	if err := (&App{}).StopProgrammingWeb(); err != nil {
		t.Fatalf("stop adopted DSH Web: %v", err)
	}
	for time.Now().Before(deadline) && programmingWebPortOpenFn() {
		time.Sleep(100 * time.Millisecond)
	}
	if programmingWebPortOpenFn() {
		t.Fatalf("port %d still open after stop", programmingWebPort)
	}
}

func writeTestFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(text)), 0o644); err != nil {
		t.Fatal(err)
	}
}
