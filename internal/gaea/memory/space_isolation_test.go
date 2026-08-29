package memory

// S1.2 记忆空间隔离器 · B 步测试（docs/gaea-memory-isolation-design.md §方案 B）：
// GetInSpace/TouchInSpace（空=旧行为、跨空间不命中）、space_id SELECT 回填、
// InSpace 视图 / Load Space 选项（空=不过滤零变化）、remember 工具 ctx 空间
// 盖章（play→play、缺省→work）。

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/db"
)

func newSpaceTestStore(t *testing.T) Store {
	t.Helper()
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(dir) })
	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	if _, err := s.Save(Memory{Name: "fact-work", Description: "工位事实", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save(Memory{Name: "fact-play", Space: "play", Description: "乐园事实", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	return s
}

// GetInSpace/TouchInSpace：空 space = 旧行为（全空间），非空仅命中该空间，
// 跨空间键不命中不 Touch。
func TestGetInSpaceAndTouchInSpace(t *testing.T) {
	s := newSpaceTestStore(t)

	// 空 space = 旧行为：两空间都能读
	for _, name := range []string{"fact-work", "fact-play"} {
		if _, ok := s.GetInSpace(name, ""); !ok {
			t.Fatalf("GetInSpace(%q, \"\") 应命中（旧行为全空间）", name)
		}
	}
	// 本空间命中、跨空间不命中
	if m, ok := s.GetInSpace("fact-play", "play"); !ok || m.Space != "play" {
		t.Fatalf("GetInSpace(play) = %+v ok=%v, want fact-play/space=play", m, ok)
	}
	if _, ok := s.GetInSpace("fact-play", "work"); ok {
		t.Fatal("跨空间键 GetInSpace(work) 不应命中")
	}
	if _, ok := s.GetInSpace("fact-work", "work"); !ok {
		t.Fatal("GetInSpace(work) 应命中 fact-work")
	}

	// TouchInSpace 跨空间不触达：fact-play 的 last_used_at 在 work 触达下不变
	before, _ := s.Get("fact-play")
	if err := s.TouchInSpace("fact-play", "work"); err != nil {
		t.Fatalf("TouchInSpace(work): %v", err)
	}
	mid, _ := s.Get("fact-play")
	if !mid.LastUsedAt.Equal(before.LastUsedAt) {
		t.Fatalf("跨空间 Touch 不应改行: %v -> %v", before.LastUsedAt, mid.LastUsedAt)
	}
	// 本空间触达生效（秒级时间戳，等待 1.1s 保证可比）
	time.Sleep(1100 * time.Millisecond)
	if err := s.TouchInSpace("fact-play", "play"); err != nil {
		t.Fatalf("TouchInSpace(play): %v", err)
	}
	after, _ := s.Get("fact-play")
	if !after.LastUsedAt.After(mid.LastUsedAt) {
		t.Fatalf("本空间 Touch 应更新 last_used_at: %v -> %v", mid.LastUsedAt, after.LastUsedAt)
	}
	// Touch（无空间谓词）行为不变
	if err := s.Touch("fact-work"); err != nil {
		t.Fatalf("Touch: %v", err)
	}
}

// listInSpace SELECT 回填 space_id：List/ListInSpace 返回的 Memory.Space 非
// 空（S1.1 时读取端不回填）。
func TestListInSpaceBackfillsSpaceColumn(t *testing.T) {
	s := newSpaceTestStore(t)
	all := s.List()
	byName := map[string]string{}
	for _, m := range all {
		byName[m.Name] = m.Space
	}
	if byName["fact-work"] != "work" {
		t.Fatalf("List 回填 Space = %q, want work", byName["fact-work"])
	}
	if byName["fact-play"] != "play" {
		t.Fatalf("List 回填 Space = %q, want play", byName["fact-play"])
	}
	// Get 亦回填
	if m, ok := s.Get("fact-play"); !ok || m.Space != "play" {
		t.Fatalf("Get 回填 Space = %+v, want play", m)
	}
}

// InSpace 视图 / Load Space 选项：Space 非空时 Set 的 Store 只在该空间读
//（List/Index/Get 收窄）；空 = 不过滤（既有调用零行为变化）。
func TestLoadSpaceOptionScopesReads(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(dir) })
	cwd := "/Users/me/proj"
	s := SQLiteStoreFor(gdb, dir, cwd)
	if _, err := s.Save(Memory{Name: "fact-work", Description: "工位", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Save(Memory{Name: "fact-play", Space: "play", Description: "乐园", Body: "b"}); err != nil {
		t.Fatal(err)
	}

	// 旧行为：不传 Space → 全量
	all := Load(Options{UserDir: dir, CWD: cwd, DB: gdb})
	if got := len(all.Store.List()); got != 2 {
		t.Fatalf("缺省 List = %d 条, want 2（零行为变化）", got)
	}
	if !strings.Contains(all.Index, "fact-play") || !strings.Contains(all.Index, "fact-work") {
		t.Fatalf("缺省 Index 应含两空间条目: %s", all.Index)
	}

	// play 视图：只读 play
	play := Load(Options{UserDir: dir, CWD: cwd, DB: gdb, Space: "play"})
	if got := play.Store.List(); len(got) != 1 || got[0].Name != "fact-play" {
		t.Fatalf("play 视图 List = %+v, want [fact-play]", got)
	}
	if strings.Contains(play.Index, "fact-work") || !strings.Contains(play.Index, "fact-play") {
		t.Fatalf("play 视图 Index 应只含 play 条目: %s", play.Index)
	}
	if _, ok := play.Store.Get("fact-work"); ok {
		t.Fatal("play 视图 Get(fact-work) 不应命中")
	}
	if _, ok := play.Store.Get("fact-play"); !ok {
		t.Fatal("play 视图 Get(fact-play) 应命中")
	}
	// work 视图对称
	work := Load(Options{UserDir: dir, CWD: cwd, DB: gdb, Space: "work"})
	if got := work.Store.List(); len(got) != 1 || got[0].Name != "fact-work" {
		t.Fatalf("work 视图 List = %+v, want [fact-work]", got)
	}
	// file 后端目录隔离：Space 选项不改变 file 后端行为（等价全量）
	fs := StoreFor(dir, cwd)
	if got := fs.InSpace("play").List(); len(got) != len(fs.List()) {
		t.Fatalf("file 后端 InSpace 应与全量等价: %d vs %d", len(got), len(fs.List()))
	}
}

// remember 工具 ctx 空间盖章（A 步写侧）：play ctx → 落 play，缺省 → 落 work；
// 修复「play 会话记忆默认落 work」写侧泄漏。Profile 路径（type=user）走主脑
// 画像表不受影响，此处验证 facts 路径。
func TestRememberToolStampsSpaceFromContext(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(dir) })
	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	tool := rememberTool{store: s}

	// play ctx：经 memory.WithSpace 盖章（agent 链路由 executeOne 注入同款值）
	args, _ := json.Marshal(map[string]string{
		"name": "play-pref", "description": "乐园偏好", "body": "喜欢像素游戏", "type": "project",
	})
	if _, err := tool.Execute(WithSpace(context.Background(), "play"), args); err != nil {
		t.Fatalf("Execute(play): %v", err)
	}
	// 缺省 ctx（无标注 → work）
	args2, _ := json.Marshal(map[string]string{
		"name": "work-pref", "description": "工位偏好", "body": "先对科目再汇总", "type": "project",
	})
	if _, err := tool.Execute(context.Background(), args2); err != nil {
		t.Fatalf("Execute(default): %v", err)
	}

	play := namesOf(s.ListInSpace("play"))
	work := namesOf(s.ListInSpace("work"))
	if len(play) != 1 || !play["play-pref"] {
		t.Fatalf("play ctx 记忆应落 play: %v", play)
	}
	if len(work) != 1 || !work["work-pref"] {
		t.Fatalf("缺省 ctx 记忆应落 work: %v", work)
	}

	// SpaceFromContext 语义：缺省/非法回退 work，play 透传
	if got := SpaceFromContext(context.Background()); got != "work" {
		t.Fatalf("SpaceFromContext(Background) = %q, want work", got)
	}
	if got := SpaceFromContext(WithSpace(context.Background(), "bogus")); got != "work" {
		t.Fatalf("SpaceFromContext(bogus) = %q, want work（Normalize 兜底）", got)
	}
}

// memory_get 工具空间限定：本空间可读并 Touch，跨空间键报未找到。
func TestMemoryGetToolSpaceScoped(t *testing.T) {
	dir := t.TempDir()
	gdb := db.GetDatabase(dir)
	if gdb == nil {
		t.Fatal("GetDatabase nil")
	}
	t.Cleanup(func() { db.CloseDatabase(dir) })
	s := SQLiteStoreFor(gdb, dir, "/Users/me/proj")
	if _, err := s.Save(Memory{Name: "play-fact", Space: "play", Description: "d", Body: "乐园正文"}); err != nil {
		t.Fatal(err)
	}
	tool := NewMemoryGetTool(s)
	args, _ := json.Marshal(map[string]string{"name": "play-fact"})

	// play ctx 可读
	out, err := tool.Execute(WithSpace(context.Background(), "play"), args)
	if err != nil || !strings.Contains(out, "乐园正文") {
		t.Fatalf("play ctx 读取: out=%q err=%v", out, err)
	}
	// work ctx 跨空间不可读
	if _, err := tool.Execute(WithSpace(context.Background(), "work"), args); err == nil {
		t.Fatal("work ctx 读取 play 记忆应报未找到（隔离红线）")
	}
	// 缺省 ctx（work）同样不可读 play 记忆
	if _, err := tool.Execute(context.Background(), args); err == nil {
		t.Fatal("缺省 ctx 读取 play 记忆应报未找到")
	}
}
