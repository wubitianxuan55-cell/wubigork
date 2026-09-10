package memory

import (
	"testing"
)

// StripDanglingCitations 语义（v4.211 回复定稿闸）：悬空键整处剥离（含紧邻
// 前导空白），命中键原样保留；**不 Touch**（触达职责在回合收尾，两路分离）。
func TestStripDanglingCitations(t *testing.T) {
	gdb := graphTestDB(t)
	store := SQLiteStoreFor(gdb, t.TempDir(), t.TempDir())
	if _, err := store.Save(Memory{Name: "cost-rule", Body: "b"}); err != nil {
		t.Fatalf("save: %v", err)
	}

	// 命中键（大小写不敏感）保留 + 悬空键剥离（含前导空格），返回剥离清单。
	in := "按 [MEM:cost-rule] 汇总，参考 [MEM:ghost] 与 [MEM:Cost-Rule]。"
	cleaned, stripped := store.StripDanglingCitations(in, "work")
	if cleaned != "按 [MEM:cost-rule] 汇总，参考 与 [MEM:Cost-Rule]。" {
		t.Fatalf("cleaned = %q", cleaned)
	}
	if len(stripped) != 1 || stripped[0] != "ghost" {
		t.Fatalf("stripped = %v, want [ghost]", stripped)
	}

	// 全命中：原样返回，nil 清单。
	cleaned, stripped = store.StripDanglingCitations("见 [MEM:cost-rule]。", "work")
	if cleaned != "见 [MEM:cost-rule]。" || stripped != nil {
		t.Fatalf("全命中应原样：cleaned=%q stripped=%v", cleaned, stripped)
	}

	// 无引用键：快速路径原样返回。
	cleaned, stripped = store.StripDanglingCitations("普通文本 [链接](x) 不动", "work")
	if cleaned != "普通文本 [链接](x) 不动" || stripped != nil {
		t.Fatalf("无键应原样：cleaned=%q stripped=%v", cleaned, stripped)
	}

	// 空间隔离：work 会话里 play 键视为悬空剥离（读端隔离器同语义）。
	if _, err := store.Save(Memory{Name: "play-note", Body: "b", Space: "play"}); err != nil {
		t.Fatalf("save play: %v", err)
	}
	cleaned, stripped = store.StripDanglingCitations("玩 [MEM:play-note] 去了", "work")
	if cleaned != "玩 去了" || len(stripped) != 1 {
		t.Fatalf("跨空间应剥离：cleaned=%q stripped=%v", cleaned, stripped)
	}
	// 本空间命中不剥。
	cleaned, _ = store.StripDanglingCitations("玩 [MEM:play-note] 去了", "play")
	if cleaned != "玩 [MEM:play-note] 去了" {
		t.Fatalf("本空间不应剥离：%q", cleaned)
	}
}

// StripDanglingCitations 不触达：命中键的 last_used/事件日志都不动。
func TestStripDanglingCitationsDoesNotTouch(t *testing.T) {
	gdb := graphTestDB(t)
	store := SQLiteStoreFor(gdb, t.TempDir(), t.TempDir())
	if _, err := store.Save(Memory{Name: "kept", Body: "b"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	before, _ := store.Get("kept")
	if _, stripped := store.StripDanglingCitations("保留 [MEM:kept] 与 [MEM:nope]", "work"); len(stripped) != 1 {
		t.Fatalf("stripped = %v, want [nope]", stripped)
	}
	after, _ := store.Get("kept")
	if !before.LastUsedAt.Equal(after.LastUsedAt) {
		t.Fatalf("last_used_at 被改动：before=%v after=%v", before.LastUsedAt, after.LastUsedAt)
	}
	events, _ := (&EventLog{DB: gdb}).LoadEvents()
	for _, e := range events {
		if e.Op == OpTouch {
			t.Fatalf("定稿闸不应落 touch 事件：%+v", e)
		}
	}
}

// 零值 Store 防误剥：记忆未装配（零值构造）时整体跳过，引用键原样保留。
func TestStripDanglingCitationsZeroStoreNoop(t *testing.T) {
	var zero Store
	cleaned, stripped := zero.StripDanglingCitations("见 [MEM:cost-rule]。", "work")
	if cleaned != "见 [MEM:cost-rule]。" || stripped != nil {
		t.Fatalf("零值 Store 应原样：cleaned=%q stripped=%v", cleaned, stripped)
	}
}
