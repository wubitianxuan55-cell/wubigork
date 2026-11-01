// voice_v2_service_test.go — Voice V2 监护逻辑测试（probe/spawn 全桩，
// 绝不真拉 GPU 服务——internal/app 测试隔离铁律）。
package app

import (
	"os/exec"
	"testing"
	"time"
)

func withVoiceV2Stubs(t *testing.T, probe func(port int) bool, spawn func(name string, cmd *exec.Cmd) error) {
	t.Helper()
	oldProbe, oldSpawn, oldKill := voiceV2Probe, voiceV2Spawn, voiceV2KillPort
	voiceV2Probe, voiceV2Spawn = probe, spawn
	voiceV2KillPort = func(int) bool { return false }
	t.Cleanup(func() { voiceV2Probe, voiceV2Spawn, voiceV2KillPort = oldProbe, oldSpawn, oldKill })
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
	withVoiceV2Stubs(t, func(port int) bool { return port == voiceV2BreezePort }, func(string, *exec.Cmd) error {
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
