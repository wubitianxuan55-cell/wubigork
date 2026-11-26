package app

// 文学子代理插话绑定（2026-11-26 文学同步批）的行为锁。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestWenxueSubagentSteer_TerminalRunUsesWenxueRunner 终态腿必须走图文引擎自己
// 的 runner（wx.followUp），不得被办公 runner（ga.followUp）接走——这正是此前
// 文学前端把 SubagentSteer 映射到办公绑定的病灶。
func TestWenxueSubagentSteer_TerminalRunUsesWenxueRunner(t *testing.T) {
	sessionPath, sessionDir := writeSubagentFixture(t)
	ref := "sa_20260902_100000_0000000004_d5d5d5d5"
	metaPath := filepath.Join(sessionDir, "subagents", ref+".meta.json")
	writeJSON(t, metaPath, map[string]interface{}{
		"ref": ref, "status": "completed", "title": "可追问的图文运行", "sessionPath": sessionPath,
		"followUpError": "上一枪的失败：provider 掉线",
		"createdAt":     time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano),
		"updatedAt":     time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano),
	})

	// 办公 runner：被调用即判负；图文 runner：把收到的 (ref, prompt) 送回测试。
	officeHit := make(chan string, 1)
	ga.mu.Lock()
	prevOffice := ga.followUp
	ga.followUp = func(_ context.Context, gotRef, _ string) error {
		select {
		case officeHit <- gotRef:
		default:
		}
		return nil
	}
	ga.mu.Unlock()
	t.Cleanup(func() {
		ga.mu.Lock()
		ga.followUp = prevOffice
		ga.mu.Unlock()
	})

	wxHit := make(chan [2]string, 1)
	wx.followUpMu.Lock()
	prevWx := wx.followUp
	wx.followUp = func(_ context.Context, gotRef, gotPrompt string) error {
		wxHit <- [2]string{gotRef, gotPrompt}
		return nil
	}
	wx.followUpMu.Unlock()
	t.Cleanup(func() {
		wx.followUpMu.Lock()
		wx.followUp = prevWx
		wx.followUpMu.Unlock()
		followUpClaims.Delete(ref)
	})

	a := &App{core: &core{}}
	got, err := a.WenxueSubagentSteer(sessionPath, ref, "追问：再展开第二点")
	if err != nil {
		t.Fatalf("WenxueSubagentSteer: %v", err)
	}
	if got != "follow-up started" {
		t.Fatalf("终态腿应返回 follow-up started，得 %q", got)
	}

	select {
	case hit := <-wxHit:
		if hit[0] != ref || hit[1] != "追问：再展开第二点" {
			t.Fatalf("图文 runner 收到的参数不符: %v", hit)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("图文 runner 未被调用")
	}
	select {
	case ref2 := <-officeHit:
		t.Fatalf("办公 runner 不应被调用（ref=%s）", ref2)
	default:
	}

	// 受理即同步清盘：返回时旧摘要已不在 meta 里（FollowUp 同款契约）。
	b, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read meta: %v", err)
	}
	if strings.Contains(string(b), "上一枪的失败") {
		t.Fatal("stale followUpError should be cleared synchronously on dispatch")
	}
}

// TestWenxueSubagentSteer_RejectsHonest 空内容/非法引用/未知引用如实拒绝，
// 绝不静默转续跑。
func TestWenxueSubagentSteer_RejectsHonest(t *testing.T) {
	sessionPath, _ := writeSubagentFixture(t)
	a := &App{core: &core{}}

	if _, err := a.WenxueSubagentSteer(sessionPath, "sa_20260902_100000_0000000004_d5d5d5d5", "  "); err == nil {
		t.Fatal("空插话内容应被拒绝")
	}
	if _, err := a.WenxueSubagentSteer(sessionPath, "not-a-ref", "插话"); err == nil {
		t.Fatal("非法子代理引用应被拒绝")
	}
	unknown := "sa_20260902_100000_0000000009_0a0a0a0a"
	if _, err := a.WenxueSubagentSteer(sessionPath, unknown, "插话"); err == nil || !strings.Contains(err.Error(), "不在运行") {
		t.Fatalf("未知引用应如实报「不在运行」，得 %v", err)
	}
}
