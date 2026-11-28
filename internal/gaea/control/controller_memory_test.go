package control

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/memory"
)

func TestRefreshMemorySpaceIsolation(t *testing.T) {
	userDir := t.TempDir()
	gdb := db.GetDatabase(userDir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(userDir) })
	c := New(Options{Sink: event.FuncSink(func(event.Event) {})})
	c.mem = memory.Load(memory.Options{
		CWD:     t.TempDir(),
		UserDir: userDir,
		DB:      gdb,
	})
	if _, err := c.mem.Store.Save(memory.Memory{
		Name: "work-fact", Space: "work", Type: memory.TypeProject,
		Kind: memory.KindSemantic, Description: "工位事实", Body: "预算口径",
	}); err != nil {
		t.Fatalf("save work-fact: %v", err)
	}
	if _, err := c.mem.Store.Save(memory.Memory{
		Name: "play-fact", Space: "play", Type: memory.TypeProject,
		Kind: memory.KindSemantic, Description: "乐园事实", Body: "游戏偏好",
	}); err != nil {
		t.Fatalf("save play-fact: %v", err)
	}

	names := func(ms []memory.Memory) []string {
		out := make([]string, 0, len(ms))
		for _, m := range ms {
			out = append(out, m.Name)
		}
		return out
	}

	// work 会话：刷新后读端只含 work（play 事实被隔离）
	c.space = "work"
	c.refreshMemory()
	got := names(c.Memory().Store.List())
	if len(got) != 1 || got[0] != "work-fact" {
		t.Fatalf("work 会话 List = %v, want [work-fact]", got)
	}
	// 逐轮注入同样收窄：play 关键词命中的事实不注入 work 会话
	if block := c.Memory().RecallBlock("游戏偏好 乐园", 0); strings.Contains(block, "play-fact") {
		t.Fatalf("work 会话 RecallBlock 泄露 play 事实: %q", block)
	}

	// play 会话：刷新后只含 play
	c.space = "play"
	c.refreshMemory()
	got = names(c.Memory().Store.List())
	if len(got) != 1 || got[0] != "play-fact" {
		t.Fatalf("play 会话 List = %v, want [play-fact]", got)
	}

	// space.mode=off（""）：不过滤，两空间都可见（旧行为）
	c.space = ""
	c.refreshMemory()
	got = names(c.Memory().Store.List())
	if len(got) != 2 {
		t.Fatalf("mode=off List = %v, want [work-fact play-fact]", got)
	}
}

// v4.377 清污：DreamAutoWritten 只圈 auto_dream 审计写过的现存事实；
// ForgetFacts 仅允许删该交集（绑定面不可删任意记忆），交集外静默跳过，
// 删除后落 dream_purge 审计。

// 众神殿 2026-11-28 论证 action-4：中途重载换 Set 时固化块发射面携带——去重
// 依据跟 boot 前缀快照走，不按新 List 重算（否则重载后去重依据丢失，已发射
// 条目重复注入；或会话中途新固化条目被误判「已在前缀」失去召回正文）。
func TestRefreshMemoryCarriesPinnedFactsState(t *testing.T) {
	userDir := t.TempDir()
	gdb := db.GetDatabase(userDir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(userDir) })
	c := New(Options{Sink: event.FuncSink(func(event.Event) {})})
	c.mem = memory.Load(memory.Options{CWD: t.TempDir(), UserDir: userDir, DB: gdb})
	if _, err := c.mem.Store.Save(memory.Memory{
		Name: "emit-me", Space: "work", Type: memory.TypeProject,
		Kind: memory.KindSemantic, Description: "已入块固化", Body: "固化正文 预算口径",
		Pinned: true,
	}); err != nil {
		t.Fatalf("save emit-me: %v", err)
	}
	// boot 形态：固化块发射面写在装配 Set 上（emit-me 正文已完整入块）
	c.mem.SetPinnedFactsState(memory.PinnedFactsState{Emitted: []string{"emit-me"}})
	if block := c.mem.RecallBlock("预算口径", 0); strings.Contains(block, "[MEM:emit-me]") {
		t.Fatalf("装配后去重未生效: %s", block)
	}
	// 中途重载：发射面必须携带到新 Set，去重依据不丢
	c.refreshMemory()
	got := c.Memory().PinnedFactsSnapshot()
	if len(got.Emitted) != 1 || got.Emitted[0] != "emit-me" {
		t.Fatalf("重载后发射面丢失: %+v", got)
	}
	if block := c.Memory().RecallBlock("预算口径", 0); strings.Contains(block, "[MEM:emit-me]") {
		t.Fatalf("重载后去重依据丢失，已发射条目重复注入: %s", block)
	}
}
