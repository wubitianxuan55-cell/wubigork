package app

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/types"
)

// ── v4.429 小说板块观察池清账（规格 进度计划/gaea-novel-watchpool-20260929.md）──
// 本文件收 app 侧五项：连接建立阶段取消错报 / 嵌套大纲节点两处递归 /
// 计划表并发保存不丢更新 / 缺章计数续扫。

// TestStreamCreateChapter_CancelDuringConnect G-A（观察池前端#7 的后端根因）：
// 用户在流开始前点停止——ctx 已取消时 ChatStream 返回 err，旧实现走
// emit error 分支（前端显示「生成失败」）；修复后必须落盘取消路径并发出
// cancelled 事件（连接未建立，空稿只发事件不写空文件）。
func TestStreamCreateChapter_CancelDuringConnect(t *testing.T) {
	snap := subscribeCreateChapterStream(t, "create-chapter-stream")
	// 假站流会挂起，但本用例 ctx 预先取消：ChatStream 建连即带取消返回错误，
	// 永远到不了流读取循环。
	a, pm, _ := newCreateChapterSuspendingApp(t, nil, "")

	ctx, cancel := context.WithCancel(a.ctx)
	cancel() // 连接建立阶段前已请求停止

	a.streamCreateChapter(ctx, pm, &types.OutlineFile{Nodes: []types.OutlineNode{
		{ID: "n1", OrderIndex: 1, Title: "第一章"},
	}}, "", "", "写个开头", 1, "", "sys", "user", 10, 1, 0.8, 1, "n1", "", false)

	ev := waitCancelledEvent(t, snap)
	if _, hasContent := ev["content"]; hasContent {
		t.Fatalf("空稿取消不应带 content，得到 %v", ev["content"])
	}
	for _, e := range snap() {
		if e["type"] == "error" {
			t.Fatalf("取消不得走 error 事件（旧缺陷形态），得到 error=%v", e["error"])
		}
	}
}

// TestGateOutlineIssuesNestedChildNode G-E/N13：分卷大纲的章节点在 Children 里，
// 写前契约检查必须递归命中（旧实现只扫顶层——分卷书的写前闸整体失明）。
func TestGateOutlineIssuesNestedChildNode(t *testing.T) {
	a := newGateEmptyApp()
	pm := newGateProject(t)
	a.setPM(pm)

	if err := pm.WriteOutlines(&types.OutlineFile{Nodes: []types.OutlineNode{
		{ID: "vol1", OrderIndex: 1, Title: "第一卷", Children: []types.OutlineNode{
			{ID: "ch2", OrderIndex: 2, ChapterFile: "002.md", Title: "", Summary: ""}, // 空 Title/Summary → S2 两条
		}},
	}}); err != nil {
		t.Fatalf("写大纲: %v", err)
	}

	issues := gateOutlineIssues(pm, 2)
	if len(issues) == 0 {
		t.Fatal("嵌套章节点应命中写前契约检查（旧实现顶层遍历返回空表）")
	}
	sawTitleEmpty := false
	for _, is := range issues {
		if is.Code == "outline_title_empty" {
			sawTitleEmpty = true
		}
	}
	if !sawTitleEmpty {
		t.Fatalf("空标题应报 outline_title_empty，得到 %+v", issues)
	}
}

// TestFindOutlineNodeByNumNested G-E/N13：按章号找主线大纲节点递归到卷下
// （章计划/意图注入的取节点路径；旧实现只扫顶层返回 nil）。
func TestFindOutlineNodeByNumNested(t *testing.T) {
	nodes := []types.OutlineNode{
		{ID: "vol1", OrderIndex: 0, Title: "第一卷", Children: []types.OutlineNode{
			{ID: "ch5", OrderIndex: 5, Branch: "", Title: "第五章"},
			{ID: "ch5a", OrderIndex: 5, Branch: "a", Title: "第五章分支"},
		}},
	}
	got := findOutlineNodeByNum(nodes, 5)
	if got == nil {
		t.Fatal("嵌套章节点应被找到（旧实现返回 nil）")
	}
	if got.ID != "ch5" {
		t.Fatalf("应命中主线节点 ch5（Branch==\"\" 优先），得到 %s", got.ID)
	}
	if got := findOutlineNodeByNum(nodes, 9); got != nil {
		t.Fatalf("不存在的章号应返回 nil，得到 %s", got.ID)
	}
}

// TestNovelChapterPlanSaveConcurrentNoLostUpdate G-F/N7：并发保存不同章的
// 计划，整表读-改-写必须串行化——两条都在（旧实现无锁，后写覆盖先写）。
func TestNovelChapterPlanSaveConcurrentNoLostUpdate(t *testing.T) {
	a, _ := newPlanTestApp(t)

	const workers = 8
	var wg sync.WaitGroup
	for i := 1; i <= workers; i++ {
		wg.Add(1)
		go func(num int) {
			defer wg.Done()
			p := planValidFixture(num)
			// 关键事件按章唯一化（跨章去重会把相同事件拒之门外，与本测无关）
			p.KeyEvents = []string{
				fmt.Sprintf("第%d章独有事件甲", num),
				fmt.Sprintf("第%d章独有事件乙", num),
			}
			if err := a.NovelChapterPlanSave(num, planMustJSON(t, p)); err != nil {
				t.Errorf("并发保存第 %d 章: %v", num, err)
			}
		}(i)
	}
	wg.Wait()

	for i := 1; i <= workers; i++ {
		got, err := a.NovelChapterPlanGet(i)
		if err != nil || got == nil {
			t.Fatalf("第 %d 章计划丢失: plan=%v err=%v（读-改-写并发覆盖）", i, got, err)
		}
	}
}

// TestCountWrittenChaptersSkipsGaps G-B/N9 同族：中间缺章时已写章数按实际
// 计数（旧实现缺口即停，1、2、4 只数出 2）。
func TestCountWrittenChaptersSkipsGaps(t *testing.T) {
	pm := newGateProject(t)
	for _, pair := range []struct {
		num  int
		body string
	}{{1, "第一章"}, {2, "第二章"}, {4, "第四章（第 3 章缺）"}} {
		if err := pm.WriteChapter(pair.num, pair.body); err != nil {
			t.Fatalf("写第 %d 章: %v", pair.num, err)
		}
	}
	if n := countWrittenChapters(pm); n != 3 {
		t.Fatalf("缺口后应数出 3 章，得到 %d", n)
	}
}
