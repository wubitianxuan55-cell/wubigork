package control

// B4（codex 蒸馏第四批 P5）回合设置快照与受限更新验收。
// 拍板（2026-11-17）：权限算快照维度——回合内切换被拒并给理由，
// 在跑回合按开始时的级别跑完；回合外立即生效（行为与改动前一致）。

import (
	"context"
	"strings"
	"testing"
	"time"
)

func startBlockedTurn(c *Controller) (started chan struct{}, release chan struct{}) {
	started = make(chan struct{})
	release = make(chan struct{})
	if !c.tryStartTurn(func(ctx context.Context) error {
		close(started)
		<-release
		return nil
	}) {
		panic("tryStartTurn failed")
	}
	<-started
	return started, release
}

// 回合内切权限 → Rejected + 理由含快照级别与目标级别；回合不撕裂。
func TestStepGuardRejectsPermLevelMidTurn(t *testing.T) {
	rec := newAttentionRecorder()
	c := New(Options{Sink: rec})
	_, release := startBlockedTurn(c)
	defer close(release)

	if got := c.PermLevelSnapshot(); got != "ask" {
		t.Fatalf("snapshot = %q, want ask（默认级别）", got)
	}
	status, reason := c.SetPermLevelReported("yolo")
	if status != StepRejected {
		t.Fatalf("status = %q, want rejected", status)
	}
	if !strings.Contains(reason, "ask") || !strings.Contains(reason, "yolo") {
		t.Errorf("reason 缺快照/目标级别: %q", reason)
	}
	if c.PermLevel() != "ask" {
		t.Errorf("permLevel 被换到 %q，回合内不得变更", c.PermLevel())
	}
}

// 回合外切权限 → Applied，级别真实生效（与改动前行为一致）。
func TestStepGuardAppliesPermLevelWhenIdle(t *testing.T) {
	rec := newAttentionRecorder()
	c := New(Options{Sink: rec})
	status, reason := c.SetPermLevelReported("auto")
	if status != StepApplied || reason != "" {
		t.Fatalf("status=%q reason=%q, want applied/空", status, reason)
	}
	if c.PermLevel() != "auto" {
		t.Fatalf("permLevel = %q, want auto", c.PermLevel())
	}
}

// 回合结束 → 快照清空、切换恢复可用（拒绝只存在于回合内）。
func TestStepGuardReleasesAfterTurn(t *testing.T) {
	rec := newAttentionRecorder()
	c := New(Options{Sink: rec})
	_, release := startBlockedTurn(c)
	if _, r := c.SetPermLevelReported("auto"); r == "" {
		t.Fatal("回合内应拒绝")
	}
	close(release)
	waitIdle(t, c)
	if got := c.PermLevelSnapshot(); got != "" {
		t.Errorf("snapshot after turn = %q, want 空", got)
	}
	if status, _ := c.SetPermLevelReported("auto"); status != StepApplied {
		t.Error("回合结束后应恢复可切换")
	}
}

// 回合内窗口同步 → Rejected；回合外 → Applied。
func TestStepGuardContextWindow(t *testing.T) {
	rec := newAttentionRecorder()
	c := New(Options{Sink: rec})
	_, release := startBlockedTurn(c)
	status, reason := c.SetContextWindowReported(64000)
	if status != StepRejected || !strings.Contains(reason, "窗口") {
		t.Fatalf("mid-turn: status=%q reason=%q", status, reason)
	}
	close(release)
	waitIdle(t, c)
	if status, _ := c.SetContextWindowReported(64000); status != StepApplied {
		t.Error("idle: want applied")
	}
}

// 兼容面：无回报的 SetPermLevel 回合内静默拒绝（不 panic、不换级）。
func TestStepGuardLegacySetterSafeMidTurn(t *testing.T) {
	rec := newAttentionRecorder()
	c := New(Options{Sink: rec})
	_, release := startBlockedTurn(c)
	defer close(release)
	c.SetPermLevel("yolo") // 不 panic
	if c.PermLevel() == "yolo" {
		t.Error("legacy setter 换级成功，快照被撕裂")
	}
}

func waitIdle(t *testing.T, c *Controller) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !c.Running() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("controller did not go idle")
}
