package memory

// ── 晨报预载块纯函数测试（v4.16 刀④）──────────────────────────────
// 覆盖：确定性排序（max(UpdatedAt,LastUsedAt) 降序 + 高频优先）、预算截断
// （整块 ≤ maxRunes、UTF-8 多字节边界不切开）、默认预算、空输入/无可渲染
// 条目返回空串。

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func preloadMem(name, desc string, updated, lastUsed time.Time) Memory {
	return Memory{Name: name, Description: desc, UpdatedAt: updated, LastUsedAt: lastUsed}
}

// TestBuildMorningPreloadBlock_Empty 空输入/无可渲染条目 → 空串。
func TestBuildMorningPreloadBlock_Empty(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	if got := BuildMorningPreloadBlock(nil, now, 0); got != "" {
		t.Fatalf("nil 输入 = %q, want 空串", got)
	}
	if got := BuildMorningPreloadBlock([]Memory{}, now, 600); got != "" {
		t.Fatalf("空列表 = %q, want 空串", got)
	}
	// 名称与内容全空 → 无可渲染条目 → 空串
	blank := []Memory{{Name: "", Title: "", Description: "", Body: ""}}
	if got := BuildMorningPreloadBlock(blank, now, 600); got != "" {
		t.Fatalf("空条目列表 = %q, want 空串", got)
	}
}

// TestBuildMorningPreloadBlock_SortAndRender 排序口径与渲染形态：
// max(UpdatedAt,LastUsedAt) 降序（高频优先），行格式「- 名称：摘要」。
func TestBuildMorningPreloadBlock_SortAndRender(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	old := now.Add(-72 * time.Hour)
	block := BuildMorningPreloadBlock([]Memory{
		preloadMem("stale", "旧条目", old, time.Time{}),
		preloadMem("fresh", "新条目", now.Add(-1*time.Hour), time.Time{}),
	}, now, 0)
	if !strings.Contains(block, "【工作记忆晨报】") {
		t.Fatalf("缺块头: %q", block)
	}
	// 块头后第一行应为近期条目（max(UpdatedAt,LastUsedAt) 降序）
	lines := strings.Split(strings.TrimSpace(block), "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[1]) != "- fresh：新条目" {
		t.Fatalf("近期条目应在块头后首位: %q", block)
	}
	if !strings.Contains(block, "fresh：新条目") || !strings.Contains(block, "stale：旧条目") {
		t.Fatalf("渲染行缺失: %q", block)
	}

	// 高频优先：LastUsedAt 更新者（max 排序键更晚）排在 UpdatedAt 更新者之前
	hot := preloadMem("hot", "高频条目", old, now.Add(-30*time.Minute))
	cold := preloadMem("cold", "低频条目", now.Add(-1*time.Hour), time.Time{})
	got := BuildMorningPreloadBlock([]Memory{cold, hot}, now, 0)
	gotLines := strings.Split(strings.TrimSpace(got), "\n")
	if len(gotLines) < 2 || strings.TrimSpace(gotLines[1]) != "- hot：高频条目" {
		t.Fatalf("LastUsedAt 更新者应优先: %q", got)
	}
}

// TestBuildMorningPreloadBlock_Budget 预算纪律：整块 ≤ maxRunes；整行放不下
// 时按 UTF-8 字符边界截断，输出恒为合法 UTF-8（不切开多字节字符）。
func TestBuildMorningPreloadBlock_Budget(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	mems := []Memory{
		preloadMem("alpha", strings.Repeat("甲", 200), now.Add(-1*time.Hour), time.Time{}),
		preloadMem("beta", strings.Repeat("乙", 200), now.Add(-2*time.Hour), time.Time{}),
		preloadMem("gamma", strings.Repeat("丙", 200), now.Add(-3*time.Hour), time.Time{}),
	}
	for _, budget := range []int{1, 8, 9, 10, 30, 100, 500, 600, 700} {
		block := BuildMorningPreloadBlock(mems, now, budget)
		if !utf8.ValidString(block) {
			t.Fatalf("budget=%d 输出非法 UTF-8: %q", budget, block)
		}
		if n := utf8.RuneCountInString(block); n > budget {
			t.Fatalf("budget=%d 超预算: %d rune > %d（%q）", budget, n, budget, block)
		}
	}
}

// TestBuildMorningPreloadBlock_DefaultBudget maxRunes<=0 → 默认 600 rune：
// 记忆再多也不挤爆注入预算（对齐 profileBudget 精神）。
func TestBuildMorningPreloadBlock_DefaultBudget(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	mems := make([]Memory, 0, 30)
	for i := 0; i < 30; i++ {
		mems = append(mems, preloadMem(
			"fact-"+strings.Repeat(string(rune('a'+i%26)), 3),
			strings.Repeat("字", 120),
			now.Add(-time.Duration(i)*time.Hour), time.Time{}))
	}
	for _, maxRunes := range []int{0, -1} {
		block := BuildMorningPreloadBlock(mems, now, maxRunes)
		if n := utf8.RuneCountInString(block); n > DefaultMorningPreloadBudget {
			t.Fatalf("maxRunes=%d 应回退默认预算: %d rune > %d", maxRunes, n, DefaultMorningPreloadBudget)
		}
		if block == "" {
			t.Fatalf("maxRunes=%d 有记忆却返回空串", maxRunes)
		}
	}
}

// TestBuildMorningPreloadBlock_HeaderOnly 预算连块头都放不下 → 空串（诚实
// 不注入残缺块）；预算只够块头无注入行 → 空串。
func TestBuildMorningPreloadBlock_HeaderOnly(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	mems := []Memory{preloadMem("alpha", "描述", now.Add(-1*time.Hour), time.Time{})}
	if got := BuildMorningPreloadBlock(mems, now, 3); got != "" {
		t.Fatalf("预算 3 连块头都放不下，应返回空串: %q", got)
	}
	if got := BuildMorningPreloadBlock(mems, now, 8); got != "" {
		t.Fatalf("预算 8 只够块头，应返回空串（无注入行）: %q", got)
	}
}
