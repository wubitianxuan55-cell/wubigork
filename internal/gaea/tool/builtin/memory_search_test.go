package builtin

// 审计 P0#7（源 GA3-02）回归锁 · builtin 侧：memory_search 的索引解析不再读
// 无同步的包级变量，而是「ctx 上会话控制器实例索引（主路径，空间硬隔离）→
// 进程级空间分槽兜底（无控制器时，读写同锁）」。

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/spaces"
)

// searchIndexWith 造一份只含单条事实的检索索引（文件后端：Save 落盘后需重载
// 才进索引，Load 每次重建一份新索引，换入后不再原地改写）。
func searchIndexWith(t *testing.T, name, desc, body string) *memory.SearchIndex {
	t.Helper()
	opts := memory.Options{UserDir: t.TempDir(), CWD: t.TempDir()}
	set := memory.Load(opts)
	if _, err := set.Store.Save(memory.Memory{
		Name: name, Type: memory.TypeProject, Kind: memory.KindSemantic,
		Description: desc, Body: body,
	}); err != nil {
		t.Fatalf("save %s: %v", name, err)
	}
	idx := memory.Load(opts).Search
	if idx == nil {
		t.Fatalf("索引为空（%s 未进索引）", name)
	}
	return idx
}

// runSearchOn 在给定 ctx 上执行一次检索。
func runSearchOn(t *testing.T, ctx context.Context, query string) string {
	t.Helper()
	args, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		t.Fatal(err)
	}
	out, err := memorySearch{}.Execute(ctx, args)
	if err != nil {
		t.Fatalf("memory_search: %v", err)
	}
	return out
}

// stubIndexSource 冒充会话控制器：同时满足 memory.SessionSaver 与
// memorySearchIndexer（生产链路里两者都是 *control.Controller）。
type stubIndexSource struct {
	idx   *memory.SearchIndex
	space string
}

func (s stubIndexSource) SaveSession(memory.Memory) string { return "" }

func (s stubIndexSource) MemorySearchIndexForSpace(space string) *memory.SearchIndex {
	if s.space != "" && s.space != space {
		return nil
	}
	return s.idx
}

// stubPromoter 只满足 memory.SessionFactPromoter + memorySearchIndexer（覆盖
// 「SessionSaver 槽位不是控制器、Promoter 槽位才是」的兜底查找链）。
type stubPromoter struct{ idx *memory.SearchIndex }

func (s stubPromoter) PromoteSessionFacts() (int, error) { return 0, nil }

func (s stubPromoter) MemorySearchIndexForSpace(string) *memory.SearchIndex { return s.idx }

// TestMemorySearchIndexPrefersSessionInstanceOverGlobal：ctx 上有会话控制器时
// 必须用它实例持有的索引，进程级槽位（boot 装配/其它空间写入）不得介入。
func TestMemorySearchIndexPrefersSessionInstanceOverGlobal(t *testing.T) {
	instIdx := searchIndexWith(t, "session-fact", "session rule", "session account ledger")
	globalIdx := searchIndexWith(t, "global-slot-fact", "global slot rule", "global slot account ledger")
	SetMemorySearchIndexForSpace(spaces.SpaceWork, globalIdx) // 兜底槽被写成别的索引

	ctx := memory.WithSpace(context.Background(), spaces.SpaceWork)
	ctx = memory.WithSessionSaver(ctx, stubIndexSource{idx: instIdx, space: spaces.SpaceWork})
	got := runSearchOn(t, ctx, "account")
	if !strings.Contains(got, "session-fact") {
		t.Fatalf("ctx 上有控制器时未用实例索引:\n%s", got)
	}
	if strings.Contains(got, "global-slot-fact") {
		t.Fatalf("ctx 上有控制器时仍读到了进程级兜底槽:\n%s", got)
	}

	// SessionSaver 槽位不是控制器、Promoter 槽位是 → 第二个槽位兜底生效
	ctx = memory.WithSpace(context.Background(), spaces.SpaceWork)
	ctx = memory.WithPromoter(ctx, stubPromoter{idx: instIdx})
	got = runSearchOn(t, ctx, "account")
	if !strings.Contains(got, "session-fact") || strings.Contains(got, "global-slot-fact") {
		t.Fatalf("Promoter 槽位的实例索引未生效:\n%s", got)
	}
}

// TestMemorySearchIndexSpaceSlotsAreIsolated：ctx 上没有控制器时按空间分槽兜底
// ——一个空间的刷新不得覆盖另一个空间的槽位（修复前两者共用一个包级变量）。
func TestMemorySearchIndexSpaceSlotsAreIsolated(t *testing.T) {
	workIdx := searchIndexWith(t, "slot-work-fact", "slot work rule", "slot work account ledger")
	playIdx := searchIndexWith(t, "slot-play-fact", "slot play rule", "slot play account ledger")

	// 连带清一次未标注槽（同包其它用例可能留过），确保本用例只看空间槽
	SetMemorySearchIndex(nil)
	SetMemorySearchIndexForSpace(spaces.SpaceWork, workIdx)
	SetMemorySearchIndexForSpace(spaces.SpacePlay, playIdx)
	// 模拟「play 会话又刷新了一次」：不得影响 work 的解析
	SetMemorySearchIndexForSpace(spaces.SpacePlay, playIdx)

	workOut := runSearchOn(t, memory.WithSpace(context.Background(), spaces.SpaceWork), "account")
	if !strings.Contains(workOut, "slot-work-fact") || strings.Contains(workOut, "slot-play-fact") {
		t.Fatalf("work 空间槽被 play 刷新串味:\n%s", workOut)
	}
	playOut := runSearchOn(t, memory.WithSpace(context.Background(), spaces.SpacePlay), "account")
	if !strings.Contains(playOut, "slot-play-fact") || strings.Contains(playOut, "slot-work-fact") {
		t.Fatalf("play 空间槽被 work 刷新串味:\n%s", playOut)
	}

	// 反向：work 再刷新，play 的解析不变
	SetMemorySearchIndexForSpace(spaces.SpaceWork, workIdx)
	playOut = runSearchOn(t, memory.WithSpace(context.Background(), spaces.SpacePlay), "account")
	if !strings.Contains(playOut, "slot-play-fact") || strings.Contains(playOut, "slot-work-fact") {
		t.Fatalf("work 刷新后 play 空间槽被串味:\n%s", playOut)
	}

	// 清场：本用例写入的槽位不留给后续用例
	SetMemorySearchIndexForSpace(spaces.SpaceWork, nil)
	SetMemorySearchIndexForSpace(spaces.SpacePlay, nil)
}

// TestMemorySearchIndexLegacySetterIsLastResortFallback：boot 的旧入口（无空间
// 信息）落未标注槽，只在没有空间槽时兜底——签名不变以免改 boot。
func TestMemorySearchIndexLegacySetterIsLastResortFallback(t *testing.T) {
	legacyIdx := searchIndexWith(t, "legacy-fact", "legacy rule", "legacy account ledger")
	SetMemorySearchIndexForSpace(spaces.SpaceWork, nil)
	SetMemorySearchIndexForSpace(spaces.SpacePlay, nil)
	SetMemorySearchIndex(legacyIdx)
	t.Cleanup(func() { SetMemorySearchIndex(nil) })

	// 无空间章 → 缺省 work；空间槽为空 → 落未标注槽
	got := runSearchOn(t, context.Background(), "account")
	if !strings.Contains(got, "legacy-fact") {
		t.Fatalf("未标注槽兜底失效:\n%s", got)
	}

	// 空间槽一旦写入，优先于未标注槽
	spaceIdx := searchIndexWith(t, "space-priority-fact", "space rule", "space account ledger")
	SetMemorySearchIndexForSpace(spaces.SpaceWork, spaceIdx)
	t.Cleanup(func() { SetMemorySearchIndexForSpace(spaces.SpaceWork, nil) })
	got = runSearchOn(t, memory.WithSpace(context.Background(), spaces.SpaceWork), "account")
	if !strings.Contains(got, "space-priority-fact") {
		t.Fatalf("空间槽未优先于未标注槽:\n%s", got)
	}
}

// TestMemorySearchIndexConcurrentSlotsNoRace 是并发 smoke：空间槽写点与检索读点
// 并发（-count=5 稳定）。本机无 gcc 跑不了 -race，该用例与「读写点全部同锁」
// 的静态论证（memoryIndexMu 覆盖全部槽位读写）互为替代证据；race 门需 Actions。
func TestMemorySearchIndexConcurrentSlotsNoRace(t *testing.T) {
	workIdx := searchIndexWith(t, "race-work-fact", "race work rule", "race work account ledger")
	playIdx := searchIndexWith(t, "race-play-fact", "race play rule", "race play account ledger")
	SetMemorySearchIndexForSpace(spaces.SpaceWork, workIdx)
	SetMemorySearchIndexForSpace(spaces.SpacePlay, playIdx)
	t.Cleanup(func() {
		SetMemorySearchIndexForSpace(spaces.SpaceWork, nil)
		SetMemorySearchIndexForSpace(spaces.SpacePlay, nil)
	})

	done := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
				}
				SetMemorySearchIndexForSpace(spaces.SpaceWork, workIdx)
				SetMemorySearchIndexForSpace(spaces.SpacePlay, playIdx)
				SetMemorySearchIndex(workIdx)
			}
		}()
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
				}
				got := runSearchOn(t, memory.WithSpace(context.Background(), spaces.SpaceWork), "account")
				if !strings.Contains(got, "race-work-fact") || strings.Contains(got, "race-play-fact") {
					t.Errorf("并发下 work 空间解析串味:\n%s", got)
					return
				}
			}
		}()
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
				}
				got := runSearchOn(t, memory.WithSpace(context.Background(), spaces.SpacePlay), "account")
				if !strings.Contains(got, "race-play-fact") || strings.Contains(got, "race-work-fact") {
					t.Errorf("并发下 play 空间解析串味:\n%s", got)
					return
				}
			}
		}()
	}
	// 让并发窗口跑满 ~200 轮读，然后收束
	for i := 0; i < 200; i++ {
		runSearchOn(t, memory.WithSpace(context.Background(), spaces.SpaceWork), "account")
	}
	close(done)
	wg.Wait()
}
