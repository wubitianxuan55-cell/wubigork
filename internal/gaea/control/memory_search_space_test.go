package control

// 审计 P0#7（源 GA3-02）回归锁：memory_search 的检索索引必须按「会话/控制器
// 实例」归属，两个空间（work/play）同开时，一方 refreshMemory 不能把另一方的
// 检索结果换掉（S1.2 双空间硬隔离红线）。
//
// 本文件只用「修复前就已存在」的 API（New/refreshMemory/tool.Execute）构例，
// 因此修复前即可编译并复现「最后写者赢」的串味。

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/gaea/db"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/tool"
	_ "github.com/gaea/gaea/internal/gaea/tool/builtin" // memory_search 自注册
)

// lookupMemorySearch 取注册表里的 memory_search 工具。
func lookupMemorySearch(t *testing.T) tool.Tool {
	t.Helper()
	tl, ok := tool.LookupBuiltin("memory_search")
	if !ok {
		t.Fatal("memory_search 未注册（builtin init 未生效）")
	}
	return tl
}

// searchCtx 构造一次工具调用 ctx：盖会话空间章 + 会话控制器章（生产链路里
// 控制器由 control.New 经 SetSessionSaver 装到 executor，execute_one 逐次
// 盖到工具 ctx）。
func searchCtx(c *Controller, space string) context.Context {
	ctx := memory.WithSpace(context.Background(), space)
	if c != nil {
		ctx = memory.WithSessionSaver(ctx, c)
	}
	return ctx
}

// executeMemorySearch 执行一次检索（不碰 testing.T，可在并发用例里调用）。
func executeMemorySearch(tl tool.Tool, c *Controller, space, query string) (string, error) {
	args, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		return "", err
	}
	return tl.Execute(searchCtx(c, space), args)
}

// newSearchSpaceController 建一个「已按空间收窄装载记忆」的控制器。userDir
// 共享（模拟 app 里 work/play 两空间共用同一 Hephaestus.db，读端靠 space_id
// 收窄），facts 按 space 落库。
func newSearchSpaceController(t *testing.T, gdb *sql.DB, userDir, space string, facts ...memory.Memory) *Controller {
	t.Helper()
	c := New(Options{Sink: event.FuncSink(func(event.Event) {})})
	c.space = space
	c.mem = memory.Load(memory.Options{CWD: t.TempDir(), UserDir: userDir, DB: gdb})
	for _, m := range facts {
		m.Space = space
		if _, err := c.mem.Store.Save(m); err != nil {
			t.Fatalf("save %s: %v", m.Name, err)
		}
	}
	c.refreshMemory()
	return c
}

// searchFacts 返回两个空间共用词 "account" 的命中样本。
func workSearchFact() memory.Memory {
	return memory.Memory{
		Name: "work-budget-fact", Type: memory.TypeProject, Kind: memory.KindSemantic,
		Description: "work budget rule", Body: "work budget account ledger",
	}
}

func playSearchFact() memory.Memory {
	return memory.Memory{
		Name: "play-hero-fact", Type: memory.TypeProject, Kind: memory.KindSemantic,
		Description: "play hero preference", Body: "play hero account ledger",
	}
}

// TestMemorySearchIndexNotOverwrittenByOtherSpaceController 是 P0#7 的主回归
// 锁：A（work）检索到的结果不得被 B（play）的刷新覆盖，反向亦然。
func TestMemorySearchIndexNotOverwrittenByOtherSpaceController(t *testing.T) {
	userDir := t.TempDir()
	gdb := db.GetDatabase(userDir)
	if gdb == nil {
		t.Fatal("db.GetDatabase 返回 nil")
	}
	t.Cleanup(func() { db.CloseDatabase(userDir) })

	work := newSearchSpaceController(t, gdb, userDir, spaces.SpaceWork, workSearchFact())
	play := newSearchSpaceController(t, gdb, userDir, spaces.SpacePlay, playSearchFact())
	tl := lookupMemorySearch(t)

	// 写序：work 先刷新，play 后刷新（修复前 play 覆盖包级全局索引）。
	work.refreshMemory()
	play.refreshMemory()

	got, err := executeMemorySearch(tl, work, spaces.SpaceWork, "account")
	if err != nil {
		t.Fatalf("work 检索: %v", err)
	}
	if !strings.Contains(got, "work-budget-fact") {
		t.Fatalf("work 会话检索不到本空间事实（被 play 索引覆盖）:\n%s", got)
	}
	if strings.Contains(got, "play-hero-fact") {
		t.Fatalf("work 会话检索到 play 空间事实（双空间红线破裂）:\n%s", got)
	}

	// 反向：work 再刷新不得把 play 会话的结果换掉。
	work.refreshMemory()
	got, err = executeMemorySearch(tl, play, spaces.SpacePlay, "account")
	if err != nil {
		t.Fatalf("play 检索: %v", err)
	}
	if !strings.Contains(got, "play-hero-fact") {
		t.Fatalf("play 会话检索不到本空间事实（被 work 索引覆盖）:\n%s", got)
	}
	if strings.Contains(got, "work-budget-fact") {
		t.Fatalf("play 会话检索到 work 空间事实（双空间红线破裂）:\n%s", got)
	}
}

// TestMemorySearchIndexConcurrentSpaceRefresh 是并发 smoke：两空间控制器并发
// 刷新 + 并发检索（-count=5 稳定）。本机无 gcc 跑不了 -race，本用例与
// memory_search.go 的「读写点全部同锁」静态论证互为替代证据。
func TestMemorySearchIndexConcurrentSpaceRefresh(t *testing.T) {
	userDir := t.TempDir()
	gdb := db.GetDatabase(userDir)
	if gdb == nil {
		t.Fatal("db.GetDatabase 返回 nil")
	}
	t.Cleanup(func() { db.CloseDatabase(userDir) })

	work := newSearchSpaceController(t, gdb, userDir, spaces.SpaceWork, workSearchFact())
	play := newSearchSpaceController(t, gdb, userDir, spaces.SpacePlay, playSearchFact())
	tl := lookupMemorySearch(t)

	const rounds = 6
	var wg sync.WaitGroup
	errCh := make(chan string, rounds*4)

	body := func(c *Controller, space, want, notWant string) {
		defer wg.Done()
		for i := 0; i < rounds; i++ {
			got, err := executeMemorySearch(tl, c, space, "account")
			if err != nil {
				errCh <- space + " 检索失败: " + err.Error()
				return
			}
			if !strings.Contains(got, want) {
				errCh <- space + " 检索被另一空间覆盖（缺 " + want + "）:\n" + got
				return
			}
			if strings.Contains(got, notWant) {
				errCh <- space + " 检索串到另一空间（含 " + notWant + "）:\n" + got
				return
			}
		}
	}

	for i := 0; i < 4; i++ {
		wg.Add(4)
		go func() { defer wg.Done(); work.refreshMemory() }()
		go func() { defer wg.Done(); play.refreshMemory() }()
		go body(work, spaces.SpaceWork, "work-budget-fact", "play-hero-fact")
		go body(play, spaces.SpacePlay, "play-hero-fact", "work-budget-fact")
	}
	wg.Wait()
	close(errCh)
	for msg := range errCh {
		t.Error(msg)
	}
}
