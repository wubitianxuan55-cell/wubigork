package builtin

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/sandbox"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// ─── 刀1: bash 会话状态锚定（v4.378，Reasonix persistentshell 留池·取道不取器）──

// stateTestBash 返回挂了状态锚、以 workDir 为工作空间的 bash 实例。真实 bash
// 不可用的宿主（PowerShell 回退）下端到端用例跳过。
func stateTestBash(t *testing.T) bash {
	t.Helper()
	sh := sandbox.ResolveShell()
	if sh.Kind != sandbox.ShellBash {
		t.Skip("bash not available on this host")
	}
	return bash{workDir: t.TempDir(), shell: sh, st: newShellState()}
}

func runBash(t *testing.T, b bash, cmd string) (string, error) {
	t.Helper()
	args, err := json.Marshal(map[string]string{"command": cmd})
	if err != nil {
		t.Fatal(err)
	}
	return b.Execute(context.Background(), args)
}

// cwd/env 跨调用存活：第一条命令 cd 进子目录并 export 变量，第二条命令不再
// 带 cd/export，应直接看到子目录 pwd 与变量值。
func TestBashShellStatePersistsAcrossCalls(t *testing.T) {
	b := stateTestBash(t)
	sub := filepath.Join(b.workDir, "state-sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := runBash(t, b, `cd "`+sub+`" && export GAEA_STATE_E2E=hello42`); err != nil {
		t.Fatalf("first call: %v", err)
	}

	out, err := runBash(t, b, `pwd; printf '|%s' "$GAEA_STATE_E2E"`)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if !strings.Contains(out, "state-sub") {
		t.Fatalf("second call must start in the anchored directory, got %q", out)
	}
	if !strings.Contains(out, "|hello42") {
		t.Fatalf("exported variable must persist, got %q", out)
	}
}

// 显式 exit 走 EXIT trap：终态照采、退出码照传。
func TestBashShellStateExitStillAdoptsAndPropagates(t *testing.T) {
	b := stateTestBash(t)
	sub := filepath.Join(b.workDir, "exit-sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := runBash(t, b, `cd "`+sub+`"; exit 7`)
	if err == nil || !strings.Contains(err.Error(), "7") {
		t.Fatalf("exit 7 must propagate, got %v", err)
	}
	out, err := runBash(t, b, `pwd`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "exit-sub") {
		t.Fatalf("explicit exit must still adopt cwd, got %q", out)
	}
}

// 保守采纳：探针缺失/形状不对/目录已消失/超限转储一律不更新锚定态。
func TestShellStateConservativeAdopt(t *testing.T) {
	s := newShellState()
	missing := filepath.Join(t.TempDir(), "nope.bin")
	s.adopt(missing) // 文件缺失：无锚定
	if s.cwd != "" || s.env != "" {
		t.Fatalf("missing probe must not adopt, got cwd=%q env=%d", s.cwd, len(s.env))
	}

	malformed := filepath.Join(t.TempDir(), "bad.bin")
	if err := os.WriteFile(malformed, []byte("no nul separator"), 0o644); err != nil {
		t.Fatal(err)
	}
	s.adopt(malformed)
	if s.cwd != "" || s.env != "" {
		t.Fatalf("malformed probe must not adopt, got cwd=%q env=%d", s.cwd, len(s.env))
	}

	s.adopt(writeProbe(t, filepath.Join(t.TempDir(), "vanished"), "GONE=1"))
	if s.cwd != "" {
		t.Fatalf("nonexistent directory must not adopt cwd, got %q", s.cwd)
	}

	huge := "declare -x BIG=" + strings.Repeat("x", envDumpMaxBytes)
	s.adopt(writeProbe(t, t.TempDir(), huge))
	if s.env != "" {
		t.Fatalf("oversized dump must not adopt env, got %d bytes", len(s.env))
	}
}

// 合法形状采纳：cwd 进锚、env 进锚。
func TestShellStateAdoptsWellFormedProbe(t *testing.T) {
	s := newShellState()
	dir := t.TempDir()
	s.adopt(writeProbe(t, dir, "declare -x GAEA_A=\"1\"\ndeclare -x GAEA_B=\"2\"\n"))
	if s.cwd != dir {
		t.Fatalf("cwd must anchor, got %q want %q", s.cwd, dir)
	}
	if !strings.Contains(s.env, `GAEA_A="1"`) {
		t.Fatalf("env must anchor, got %q", s.env)
	}
}

// 锚定目录消失自愈：workDir 回落 fallback 且清掉死锚。
func TestShellStateWorkDirFallsBackWhenDeleted(t *testing.T) {
	s := newShellState()
	gone := filepath.Join(t.TempDir(), "gone")
	if err := os.MkdirAll(gone, 0o755); err != nil {
		t.Fatal(err)
	}
	s.cwd = gone
	if err := os.RemoveAll(gone); err != nil {
		t.Fatal(err)
	}
	fb := t.TempDir()
	if got := s.workDir(fb); got != fb {
		t.Fatalf("deleted anchor must fall back, got %q", got)
	}
	if s.cwd != "" {
		t.Fatalf("dead anchor must be cleared, got %q", s.cwd)
	}
}

// 会话切换重置：Reset 后 cwd/env 清空、探针目录整树删除。
func TestShellStateReset(t *testing.T) {
	s := newShellState()
	if _, _, p := s.begin("true"); p == "" {
		t.Fatal("begin must materialize the probe directory")
	}
	dir := t.TempDir()
	s.adopt(writeProbe(t, dir, "declare -x X=\"1\"\n"))
	if s.cwd == "" || s.env == "" || s.dir == "" {
		t.Fatal("precondition: anchor must hold state")
	}
	probeDir := s.dir
	s.reset()
	if s.cwd != "" || s.env != "" || s.dir != "" {
		t.Fatal("reset must clear anchor")
	}
	if _, err := os.Stat(probeDir); !os.IsNotExist(err) {
		t.Fatal("reset must remove the probe directory")
	}
}

// nil 锚全链路安全：workDir 回落、begin 原样、reset/adopt no-op。
func TestShellStateNilSafe(t *testing.T) {
	var s *shellState
	fb := t.TempDir()
	if got := s.workDir(fb); got != fb {
		t.Fatalf("nil anchor workDir must fall back, got %q", got)
	}
	wrapped, env, probe := s.begin("echo hi")
	if wrapped != "echo hi" || env != nil || probe != "" {
		t.Fatal("nil anchor begin must pass through")
	}
	s.adopt("")
	s.reset()
}

// bash 经 tool.SessionStateResetter 暴露重置能力（controller 会话切换钩子
// 的类型断言契约）。
func TestBashImplementsSessionStateResetter(t *testing.T) {
	b := stateTestBash(t)
	r, ok := any(b).(tool.SessionStateResetter)
	if !ok {
		t.Fatal("bash must implement tool.SessionStateResetter")
	}
	dir := t.TempDir()
	b.st.adopt(writeProbe(t, dir, "declare -x Y=\"2\"\n"))
	r.ResetSessionState()
	if b.st.cwd != "" || b.st.env != "" {
		t.Fatal("ResetSessionState must clear the anchor")
	}
}

// writeProbe 构造一枚形状合法的探针文件（pwd\0dump）并返回其路径。pwd 可以
// 是不存在的路径（消失目录用例——探针文件本身总写在真实临时目录里）。
func writeProbe(t *testing.T, pwd, dump string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "probe.bin")
	if err := os.WriteFile(p, append([]byte(pwd+"\x00"), dump...), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}
