package app

// S1.2 记忆空间隔离器 · A 步测试（docs/gaea-memory-isolation-design.md §方案 A）：
// dream 指纹键含 space（同内容 work/play 各算一次非 no-op）、play dream notes
// 不写 work AGENTS.md（丢弃仅落 facts）。

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/spaces"
)

// S1.2 A：指纹键含会话空间——同内容跨空间不共用指纹（play 会话内容与 work
// 相同时不得误命中 work 指纹跳过提炼）；space=""（mode=off）**字节级**退化
// 为纯内容哈希 sha256(input)（mode=off 三态回退锚点 9）。
func TestDreamInputHashSpaceKeyed(t *testing.T) {
	in := dreamInput([]provider.Message{
		{Role: provider.RoleUser, Content: "帮我整理这份成本测算表"},
		{Role: provider.RoleAssistant, Content: "已完成，公式与汇总如下……（超过 100 字的实质内容省略）"},
	})
	hWork := dreamInputHash("work", in)
	hPlay := dreamInputHash("play", in)
	hOff := dreamInputHash("", in)
	if hWork != dreamInputHash("work", in) {
		t.Fatal("同空间同内容指纹应一致（no-op 判定保留）")
	}
	if hWork == hPlay {
		t.Fatal("同内容跨空间指纹应不同（防跨空间 no-op 误判）")
	}
	if hWork == hOff || hPlay == hOff {
		t.Fatal("off（纯内容指纹）与分区空间指纹应不同")
	}
	// off 字节级等价旧行为：纯内容 sha256
	sum := sha256.Sum256([]byte(in))
	if hOff != hex.EncodeToString(sum[:]) {
		t.Fatal("off 指纹应等于 sha256(input)（旧行为字节级一致）")
	}
	// 内容变化 → 指纹变化（同空间）
	in2 := in + "追加一列环比"
	if dreamInputHash("work", in2) == hWork {
		t.Fatal("内容变化后指纹应变化")
	}
}

// recordingNotesWriter 记录 QuickAdd 调用（*control.Controller 同形状）。
type recordingNotesWriter struct {
	calls []struct {
		scope memory.Scope
		note  string
	}
}

func (w *recordingNotesWriter) QuickAdd(scope memory.Scope, note string) (string, error) {
	w.calls = append(w.calls, struct {
		scope memory.Scope
		note  string
	}{scope, note})
	return "AGENTS.md", nil
}

// S1.2 A：dream notes 空间分流——work 行为逐字不变（QuickAdd 落 work 项目
// 说明 AGENTS.md）；play 会话 dream 不得写 work AGENTS.md（验收红线），play
// 侧丢弃 notes 仅落 facts（QuickAdd 不被调用 = 不产生 work 侧写入）。
func TestDreamWriteNotesSpaceRouting(t *testing.T) {
	notes := []dreamNote{
		{Scope: "project", Note: "成本测算常用口径：税前"},
		{Scope: "", Note: "  "}, // 空白笔记跳过
		{Scope: "user", Note: "用户偏好简报"},
	}

	// work：逐条 QuickAdd，scope 映射不变
	w := &recordingNotesWriter{}
	if got := dreamWriteNotes(w, spaces.SpaceWork, notes); got != 2 {
		t.Fatalf("work notes 写入数 = %d, want 2", got)
	}
	if len(w.calls) != 2 {
		t.Fatalf("work QuickAdd 调用 = %d, want 2", len(w.calls))
	}
	if w.calls[0].scope != memory.ScopeProject || w.calls[0].note != "成本测算常用口径：税前" {
		t.Fatalf("work 第一条 = %+v", w.calls[0])
	}
	if w.calls[1].scope != memory.ScopeUser {
		t.Fatalf("work 第二条 scope = %v", w.calls[1].scope)
	}

	// play：全部丢弃，QuickAdd 零调用（不写 work AGENTS.md）
	wp := &recordingNotesWriter{}
	if got := dreamWriteNotes(wp, spaces.SpacePlay, notes); got != 0 {
		t.Fatalf("play notes 写入数 = %d, want 0（丢弃仅落 facts）", got)
	}
	if len(wp.calls) != 0 {
		t.Fatalf("play 不得调用 QuickAdd（work AGENTS.md 属工位说明）: %+v", wp.calls)
	}

	// ""（mode=off 平铺形态）：与旧行为一致（走 QuickAdd，无空间维度）
	wo := &recordingNotesWriter{}
	if got := dreamWriteNotes(wo, "", notes); got != 2 || len(wo.calls) != 2 {
		t.Fatalf("off 形态 notes 应走 QuickAdd（旧行为）: got=%d calls=%d", got, len(wo.calls))
	}
}
