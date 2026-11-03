// voice_v2_service_test.go — Voice V2 监护逻辑测试（probe/spawn 全桩，
// 绝不真拉 GPU 服务——internal/app 测试隔离铁律）。
package app

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestFindVoiceV2RepoRootStopsAtWindowsVolumeRoot(t *testing.T) {
	start := filepath.VolumeName(t.TempDir()) + string(filepath.Separator)
	done := make(chan string, 1)
	go func() {
		done <- findRepoRootFromStarts([]string{start}, func(string) bool { return false })
	}()
	select {
	case got := <-done:
		if got != "" {
			t.Fatalf("空构建树应返回空路径，got %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("仓库根目录查找在卷根目录未终止")
	}
}

func TestFindVoiceV2RepoRootChecksAncestorOnce(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b", "c")
	starts := []string{nested, nested}
	calls := make(map[string]int)
	got := findRepoRootFromStarts(starts, func(dir string) bool {
		calls[dir]++
		return dir == root
	})
	if got != root {
		t.Fatalf("应找到仓库祖先 %q，got %q", root, got)
	}
	for dir, count := range calls {
		if count != 1 {
			t.Fatalf("重复候选 %q 被检查 %d 次，want 1", dir, count)
		}
	}
}

func TestRepoRootFromExplicitEnvironment(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "scripts", "voice_v2")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	for _, name := range []string{"gaea-spike.yaml", "gaea_sidecar_launcher.py"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("test"), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}
	t.Setenv(voiceV2RepoRootEnv, root)
	if got := repoRootFromExe(); got != root {
		t.Fatalf("显式仓库根应优先，got %q want %q", got, root)
	}
}

func TestVoiceV2TCPProbeRecognizesListeningSidecar(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	addr := ln.Addr().String()
	if !voiceV2TCPProbe(addr, 500*time.Millisecond) {
		t.Fatal("监听中的 TCP 端口应判定就绪")
	}
	_ = ln.Close()
	if voiceV2TCPProbe(addr, 100*time.Millisecond) {
		t.Fatal("关闭后的 TCP 端口不应判定就绪")
	}
}

func withVoiceV2Stubs(t *testing.T, probe func(port int) bool, spawn func(name string, cmd *exec.Cmd) error) {
	t.Helper()
	setVoiceV2Error("")
	oldProbe, oldSpawn, oldKill := voiceV2Probe, voiceV2Spawn, voiceV2KillPort
	voiceV2Probe, voiceV2Spawn = probe, spawn
	voiceV2KillPort = func(int) bool { return false }
	t.Cleanup(func() {
		voiceV2Probe, voiceV2Spawn, voiceV2KillPort = oldProbe, oldSpawn, oldKill
		setVoiceV2Error("")
	})
}

func TestVoiceV2StatusReadyGate(t *testing.T) {
	a := &mediaState{core: &core{}}

	withVoiceV2Stubs(t, func(port int) bool { return true }, func(string, *exec.Cmd) error {
		t.Fatal("全就绪时不得 spawn")
		return nil
	})
	if st := a.VoiceV2Status(); st["ready"] != true {
		t.Fatalf("三依赖全活应 ready，got %v", st)
	}

	withVoiceV2Stubs(t, func(port int) bool { return port != voiceV2LLMPort }, func(string, *exec.Cmd) error {
		t.Fatal("Status 是只读探测，不得 spawn")
		return nil
	})
	if st := a.VoiceV2Status(); st["ready"] != false {
		t.Fatalf("4B 缺位应 not ready，got %v", st)
	}
}

func TestVoiceV2StatusProbesEachDependencyOnce(t *testing.T) {
	a := &mediaState{core: &core{}}
	calls := map[int]int{}
	withVoiceV2Stubs(t, func(port int) bool {
		calls[port]++
		return true
	}, func(string, *exec.Cmd) error { return nil })

	st := a.VoiceV2Status()
	if st["ready"] != true {
		t.Fatalf("三依赖全活应 ready，got %v", st)
	}
	for _, port := range []int{voiceV2SidecarPort, voiceV2LLMPort, voiceV2BreezePort, voiceV2GatewayPort} {
		if calls[port] != 1 {
			t.Fatalf("port %d probe calls = %d, want 1（同一快照不得重复探测）", port, calls[port])
		}
	}
}

func TestVoiceV2EnsureIdempotentWhenReady(t *testing.T) {
	a := &mediaState{core: &core{}}
	withVoiceV2Stubs(t, func(int) bool { return true }, func(string, *exec.Cmd) error {
		t.Fatal("已就绪时 Ensure 不得 spawn")
		return nil
	})
	st := a.VoiceV2Ensure()
	if st["ready"] != true || st["starting"] != false {
		t.Fatalf("已就绪应立即返回 ready，got %v", st)
	}
}

func TestVoiceV2EnsureReportsSpawnFailure(t *testing.T) {
	a := &mediaState{core: &core{}}
	// breeze 就绪、4B 缺位；spawn 一律失败 → 异步 emit error 后 ensuring 复位。
	// detachedCmd 会先因 voiceV2LLMBin 不存在报错（测试机若存在该路径则走 spawn 失败）。
	withVoiceV2Stubs(t, func(port int) bool {
		return port == voiceV2BreezePort || port == voiceV2GatewayPort
	}, func(string, *exec.Cmd) error {
		return errBoom
	})
	st := a.VoiceV2Ensure()
	if st["starting"] != true {
		t.Fatalf("未就绪应返回 starting，got %v", st)
	}
	// 二次 Ensure 在 ensuring 窗口内 → 仍 starting（不重复拉起）
	if st2 := a.VoiceV2Ensure(); st2["starting"] != true {
		t.Fatalf("ensuring 中应幂等 starting，got %v", st2)
	}
	// 等 goroutine 收尾（spawn 失败立即返回，2s 足够）
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		voiceV2EnsureMu.Lock()
		busy := voiceV2Ensuring
		voiceV2EnsureMu.Unlock()
		if !busy {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	voiceV2EnsureMu.Lock()
	busy := voiceV2Ensuring
	voiceV2EnsureMu.Unlock()
	if busy {
		t.Fatal("spawn 失败后 ensuring 应回落")
	}
}

var errBoom = &boomErr{}

type boomErr struct{}

func (*boomErr) Error() string { return "boom" }
