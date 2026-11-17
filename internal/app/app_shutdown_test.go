package app

// B1（codex 蒸馏第四批 P1）关停步骤超时：挂起步骤不阻塞整体退出（A4 的
// 进程级等价面；DB 关闭类不走 shutdownStep，不在本测试面）。

import (
	"strings"
	"testing"
	"time"
)

func TestShutdownStepCompletes(t *testing.T) {
	done := make(chan struct{})
	shutdownStep("fast", time.Second, func() error {
		close(done)
		return nil
	})
	select {
	case <-done:
	default:
		t.Fatal("step did not run")
	}
}

func TestShutdownStepTimesOutAndContinues(t *testing.T) {
	start := time.Now()
	release := make(chan struct{})
	defer close(release)
	shutdownStep("hang", 50*time.Millisecond, func() error {
		<-release
		return nil
	})
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("hung step blocked shutdown for %v, want ~timeout", d)
	}
}

func TestShutdownStepTailStderr(t *testing.T) {
	long := strings.Repeat("x", stderrTailLimit+1000)
	got := tailStderr(long)
	if len(got) != stderrTailLimit+3 { // "…" + 尾部 4KiB
		t.Errorf("tail length = %d, want %d", len(got), stderrTailLimit+3)
	}
	if !strings.HasPrefix(got, "…") || !strings.HasSuffix(got, long[len(long)-stderrTailLimit:]) {
		t.Error("tail must keep the END of the text (错误与结论在尾)")
	}
	if tailStderr("short") != "short" {
		t.Error("short text must pass through unchanged")
	}
}
