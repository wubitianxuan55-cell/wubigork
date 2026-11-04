// voice_v2_service_test.go — Voice V2 监护逻辑测试（probe/spawn/realtimeStart 全桩，
// 绝不真拉 GPU 服务——internal/app 测试隔离铁律）。
// 刀6 起 Go 原生管线=唯一路径且 8792 网关退役：sidecar/4B/网关拉起链已退役，
// 本文件只测 breeze→realtimeStart 链与 Status 快照语义。
package app

import (
	"net"
	"os/exec"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/voice/realtime"
)

// withVoiceV2GoRealtime 置位/复位进程内 realtime 服务器标记（Status 的 rt 位）。
func withVoiceV2GoRealtime(t *testing.T, up bool) {
	t.Helper()
	voiceV2RealtimeMu.Lock()
	old := voiceV2RealtimeSrv
	if up {
		voiceV2RealtimeSrv = &realtime.Server{}
	} else {
		voiceV2RealtimeSrv = nil
	}
	voiceV2RealtimeMu.Unlock()
	t.Cleanup(func() {
		voiceV2RealtimeMu.Lock()
		voiceV2RealtimeSrv = old
		voiceV2RealtimeMu.Unlock()
	})
}

func withVoiceV2StartStub(t *testing.T, fn func(*mediaState) error) {
	t.Helper()
	old := voiceV2RealtimeStartFn
	voiceV2RealtimeStartFn = fn
	t.Cleanup(func() { voiceV2RealtimeStartFn = old })
}

func TestVoiceV2TCPProbeRecognizesListeningService(t *testing.T) {
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

	// rt 已接管 + breeze 探活全绿 → ready（网关退役后 LLM 腿无本地端口）
	withVoiceV2Stubs(t, func(int) bool { return true }, func(string, *exec.Cmd) error {
		t.Fatal("Status 是只读探测，不得 spawn")
		return nil
	})
	withVoiceV2GoRealtime(t, true)
	if st := a.VoiceV2Status(); st["ready"] != true {
		t.Fatalf("rt+breeze 全活应 ready，got %v", st)
	}

	// rt 未接管（服务器未启动）→ not ready（探活再多也补不了进程内态）
	withVoiceV2GoRealtime(t, false)
	if st := a.VoiceV2Status(); st["ready"] != false {
		t.Fatalf("rt 缺位应 not ready，got %v", st)
	}
}

func TestVoiceV2StatusProbesEachDependencyOnce(t *testing.T) {
	a := &mediaState{core: &core{}}
	calls := map[int]int{}
	withVoiceV2Stubs(t, func(port int) bool {
		calls[port]++
		return true
	}, func(string, *exec.Cmd) error { return nil })
	withVoiceV2GoRealtime(t, true)

	st := a.VoiceV2Status()
	if st["ready"] != true {
		t.Fatalf("依赖全活应 ready，got %v", st)
	}
	// 只探 breeze 一端口；8791 是进程内态不走探活（网关退役后无第二端口）
	for _, port := range []int{voiceV2BreezePort} {
		if calls[port] != 1 {
			t.Fatalf("port %d probe calls = %d, want 1（同一快照不得重复探测）", port, calls[port])
		}
	}
	if calls[voiceV2SidecarPort] != 0 {
		t.Fatalf("8791 不应被端口探活（rt=进程内服务器态），got %d 次", calls[voiceV2SidecarPort])
	}
}

func TestVoiceV2EnsureIdempotentWhenReady(t *testing.T) {
	a := &mediaState{core: &core{}}
	withVoiceV2Stubs(t, func(int) bool { return true }, func(string, *exec.Cmd) error {
		t.Fatal("已就绪时 Ensure 不得 spawn")
		return nil
	})
	withVoiceV2GoRealtime(t, true)
	st := a.VoiceV2Ensure()
	if st["ready"] != true || st["starting"] != false {
		t.Fatalf("已就绪应立即返回 ready，got %v", st)
	}
}

func TestVoiceV2EnsureRunsNativeChainWhenNotReady(t *testing.T) {
	a := &mediaState{core: &core{}}
	spawned := 0
	withVoiceV2Stubs(t, func(port int) bool {
		// breeze 就绪；8791 未接管
		return port != voiceV2SidecarPort
	}, func(string, *exec.Cmd) error {
		spawned++
		return nil
	})
	withVoiceV2GoRealtime(t, false)
	withVoiceV2StartStub(t, func(*mediaState) error { return nil })
	st := a.VoiceV2Ensure()
	if st["starting"] != true {
		t.Fatalf("未就绪应返回 starting，got %v", st)
	}
	// Go 唯一路径链不得 spawn 任何外部进程（breeze 已就绪直过）
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
	if spawned != 0 {
		t.Fatalf("Go 唯一路径链不得 spawn 外部进程，got %d 次", spawned)
	}
	voiceV2RealtimeMu.Lock()
	up := voiceV2RealtimeSrv != nil
	voiceV2RealtimeMu.Unlock()
	if up {
		t.Fatal("StartStub 打桩下不得真启动进程内服务器")
	}
}

func TestVoiceV2EnsureReportsStartFailure(t *testing.T) {
	a := &mediaState{core: &core{}}
	// breeze/gateway 就绪、装配失败 → 异步 emit error 后 ensuring 复位。
	withVoiceV2Stubs(t, func(port int) bool {
		return port != voiceV2SidecarPort
	}, func(string, *exec.Cmd) error {
		return errBoom
	})
	withVoiceV2GoRealtime(t, false)
	withVoiceV2StartStub(t, func(*mediaState) error { return errBoom })
	st := a.VoiceV2Ensure()
	if st["starting"] != true {
		t.Fatalf("未就绪应返回 starting，got %v", st)
	}
	// 二次 Ensure 在 ensuring 窗口内 → 仍 starting（不重复拉起）
	if st2 := a.VoiceV2Ensure(); st2["starting"] != true {
		t.Fatalf("ensuring 中应幂等 starting，got %v", st2)
	}
	// 等 goroutine 收尾（装配失败立即返回，2s 足够）
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
		t.Fatal("装配失败后 ensuring 应回落")
	}
}

var errBoom = &boomErr{}

type boomErr struct{}

func (*boomErr) Error() string { return "boom" }
