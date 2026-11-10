package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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

func TestProgrammingWebPreflightIsolatedInstall(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	agentsHome := filepath.Join(root, "home", "agents")
	base := filepath.Join(root, "node_modules", "@deepseek-ai")
	writeTestFile(t, filepath.Join(base, "dsh", "package.json"), `{"version":"0.2.2-rc.1"}`)
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

func writeTestFile(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(text)), 0o644); err != nil {
		t.Fatal(err)
	}
}
