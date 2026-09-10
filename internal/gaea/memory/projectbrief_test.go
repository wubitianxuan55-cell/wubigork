package memory

import (
	"strings"
	"testing"
	"time"
)

// 项目本体注入块（6.2）：固化优先 + [MEM:] 引用键 + 决策来源归因 + 预算截断，
// 纯函数确定性。
func TestBuildProjectBrief(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	fresh := now.Add(-24 * time.Hour)
	mems := []Memory{
		{Name: "naming-rule", Title: "命名规范", Description: "交付文件用 日期_主题 命名", Type: TypeProject, UpdatedAt: fresh},
		kouJing(),
		{Name: "casual", Description: "闲聊偏好", Type: TypeUser, Pinned: true, UpdatedAt: fresh},
		{Name: "episodic-old", Description: "一次经历", Type: TypeProject, Kind: KindEpisodic, LastUsedAt: now.Add(-200 * 24 * time.Hour)},
	}
	block := BuildProjectBrief(mems, now, 0)
	if block == "" {
		t.Fatal("有候选时不应为空")
	}
	// 固化条入选（pinned 全收，含 user 型）。
	if !strings.Contains(block, "[MEM:casual]") || !strings.Contains(block, "固化") {
		t.Fatalf("固化条应入选并带标记:\n%s", block)
	}
	// project/feedback 决策入选；user 非固化不进（评分通道只收 project/feedback）。
	if !strings.Contains(block, "[MEM:naming-rule]") {
		t.Fatalf("project 决策应入选:\n%s", block)
	}
	if strings.Contains(block, "[MEM:kou-jing]") == false && strings.Contains(block, "[MEM:episodic-old]") == false {
		t.Fatalf("project/feedback 至少其一入选:\n%s", block)
	}
	// 引用键格式（citations 正则可解析）。
	if got := ExtractCitationNames(block); len(got) < 2 {
		t.Fatalf("块内引用键应可解析: %v", got)
	}
	// 决策来源归因（SourceSession → 「依据 … 会话」）。
	if !strings.Contains(block, "依据 session-20260801-abc 会话 · turn 3") {
		t.Fatalf("决策来源归因缺失:\n%s", block)
	}
}

func kouJing() Memory {
	return Memory{Name: "kou-jing", Description: "口径：建筑面积按新清单", Type: TypeFeedback,
		UpdatedAt:     time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		LastUsedAt:    time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC),
		SourceSession: "session-20260801-abc.jsonl", SourceMessage: "turn 3"}
}

// 预算截断：预算极小时返回空（一条都放不下）或不完整块——不截半行。
func TestBuildProjectBriefBudget(t *testing.T) {
	now := time.Now()
	mems := []Memory{
		{Name: "a", Description: "短决策一", Type: TypeProject, UpdatedAt: now},
		{Name: "b", Description: "短决策二", Type: TypeProject, UpdatedAt: now},
	}
	tiny := BuildProjectBrief(mems, now, 10)
	if tiny != "" {
		t.Fatalf("极小预算应零注入: %s", tiny)
	}
	// 无候选：空串（零注入，前缀不变）。
	if got := BuildProjectBrief([]Memory{{Name: "x", Description: "d", Type: TypeUser}}, now, 0); got != "" {
		t.Fatalf("无 project/feedback/pinned 候选应为空: %s", got)
	}
	// 确定性：两次渲染逐字节一致。
	a := BuildProjectBrief(mems, now, 0)
	b := BuildProjectBrief(mems, now, 0)
	if a != b {
		t.Fatal("同一输入两次渲染不一致")
	}
}
