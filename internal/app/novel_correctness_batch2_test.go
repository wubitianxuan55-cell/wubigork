package app

// 小说板块优化批 2 · 线5（Go 后端正确性）测试：
//
//	G1  生成收尾的大纲回写改为「读-改-写 + 按 nodeID 定点合并」，生成期间的
//	    并发编辑（新增/删除节点、改其它章状态与摘要）不被旧快照回滚。
//	N8  建章节点落盘失败必须中止本次生成，不得把没写进 outline.json 的章
//	    当成已建好。
//	M5  取消残稿文件名唯一化（同秒两次取消各留一份）。
//	G8  去味 before 分口径：生成链与手动路径共用 deSlopRewriteWithin。

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/analysis"
	"github.com/gaea/gaea/internal/bookimport"
	"github.com/gaea/gaea/internal/booksource"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/novelstyle"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/types"
)

// beginChapterGen 把「生成开始时的旧大纲快照」读进内存，返回该快照指针。
// 与 createChapter 的 of 同语义：内容取自磁盘，之后不再自动同步。
func beginChapterGen(t *testing.T, pm *project.Manager) *types.OutlineFile {
	t.Helper()
	of, err := pm.ReadOutlines()
	if err != nil || of == nil {
		t.Fatalf("读大纲: %v", err)
	}
	return of
}

// outlineNode 从磁盘读回大纲并按 ID 取节点（不存在返回 nil）。
func outlineNode(t *testing.T, pm *project.Manager, id string) *types.OutlineNode {
	t.Helper()
	of, err := pm.ReadOutlines()
	if err != nil || of == nil {
		t.Fatalf("读大纲: %v", err)
	}
	return findOutlineNodeByID(of.Nodes, id)
}

// TestMergeChapterWriteBack_KeepsConcurrentEdits 生成期间作者对**别的节点**的
// 并发编辑（改摘要 / 改状态 / 改要点 / 新增节点 / 删除节点）必须全部保留，
// 只有目标节点的摘要与状态被本次生成推进（G1 核心用例）。
func TestMergeChapterWriteBack_KeepsConcurrentEdits(t *testing.T) {
	pm := newGateProject(t)
	if err := pm.WriteOutlines(&types.OutlineFile{StoryThread: "主线", Nodes: []types.OutlineNode{
		{ID: "n1", Title: "第1章", OrderIndex: 1, Summary: "旧摘要1", Status: types.OutlinePlanned},
		{ID: "n2", Title: "第2章", OrderIndex: 2, Summary: "旧摘要2", Status: types.OutlinePlanned},
		{ID: "n3", Title: "第3章", OrderIndex: 3, Summary: "旧摘要3", Status: types.OutlineWriting},
		{ID: "n9", Title: "将被删除", OrderIndex: 9, Summary: "待删", Status: types.OutlinePlanned},
	}}); err != nil {
		t.Fatalf("预置大纲: %v", err)
	}
	// 生成开始：这一刻的快照。此后磁盘上的编辑都与它无关。
	of := beginChapterGen(t, pm)

	// 生成期间（数分钟）作者在别的页面做的事，全部写同一份 outline.json：
	// 改第 2 章摘要与要点、把第 3 章状态退回 planned、新增第 4 章、删掉 n9。
	latest, err := pm.ReadOutlines()
	if err != nil {
		t.Fatalf("重读大纲: %v", err)
	}
	findOutlineNodeByID(latest.Nodes, "n2").Summary = "生成期间改的新摘要2"
	findOutlineNodeByID(latest.Nodes, "n2").KeyPoints = []string{"作者刚加的关键点"}
	findOutlineNodeByID(latest.Nodes, "n3").Status = types.OutlinePlanned
	latest.Nodes = append(latest.Nodes, types.OutlineNode{
		ID: "n4", Title: "第4章", OrderIndex: 4, Summary: "生成期间新增", Status: types.OutlinePlanned,
	})
	kept := latest.Nodes[:0]
	for _, n := range latest.Nodes {
		if n.ID != "n9" {
			kept = append(kept, n)
		}
	}
	latest.Nodes = kept
	if err := pm.WriteOutlines(latest); err != nil {
		t.Fatalf("写回并期编辑: %v", err)
	}

	// 生成收尾：只应把 n1 的摘要与状态应用到**最新**快照。
	a := newGateEmptyApp()
	a.mergeChapterWriteBack(pm, "n1", "本次生成摘要1", types.OutlineDone)

	got := outlineNode(t, pm, "n1")
	if got == nil || got.Summary != "本次生成摘要1" || got.Status != types.OutlineDone {
		t.Fatalf("目标节点应被本次生成推进: %+v", got)
	}
	if n := outlineNode(t, pm, "n2"); n.Summary != "生成期间改的新摘要2" || len(n.KeyPoints) != 1 {
		t.Fatalf("生成期间对别的节点的摘要/要点编辑被回滚: %+v", n)
	}
	if n := outlineNode(t, pm, "n3"); n.Status != types.OutlinePlanned {
		t.Fatalf("生成期间对别的节点状态的回退被覆盖: %+v", n)
	}
	if n := outlineNode(t, pm, "n4"); n == nil || n.Summary != "生成期间新增" {
		t.Fatalf("生成期间新增的节点丢失: %+v", n)
	}
	if n := outlineNode(t, pm, "n9"); n != nil {
		t.Fatalf("生成期间删除的节点被旧快照复活: %+v", n)
	}
	// 旧快照本身不该被当作写回源——写回完成后它仍是生成开始时的样子。
	if of.Nodes[0].Summary != "旧摘要1" {
		t.Fatalf("旧快照应保持生成开始时的内容（写回源必须是重读结果）: %+v", of.Nodes[0])
	}
}

// TestMergeChapterWriteBack_EmptySummaryKeepsOld 空/纯空白摘要不得覆盖既有摘要
// （否则写前硬闸的 outline_summary_empty 会反噬同一章），状态仍推进到 done。
func TestMergeChapterWriteBack_EmptySummaryKeepsOld(t *testing.T) {
	pm := newGateProject(t)
	if err := pm.WriteOutlines(&types.OutlineFile{Nodes: []types.OutlineNode{
		{ID: "n1", OrderIndex: 1, Summary: "既有摘要", Status: types.OutlineWriting},
	}}); err != nil {
		t.Fatalf("预置大纲: %v", err)
	}

	newGateEmptyApp().mergeChapterWriteBack(pm, "n1", "   ", types.OutlineDone)

	got := outlineNode(t, pm, "n1")
	if got.Summary != "既有摘要" {
		t.Fatalf("空白摘要不得覆盖既有摘要: %+v", got)
	}
	if got.Status != types.OutlineDone {
		t.Fatalf("状态仍应推进到 done: %+v", got)
	}
}

// TestMergeChapterWriteBack_MissingNodeNotRevived 目标节点在生成期间被并发删除
// 时：不加回、其余节点不动（宁可不写，不复活已删节点）。
func TestMergeChapterWriteBack_MissingNodeNotRevived(t *testing.T) {
	pm := newGateProject(t)
	// 生成结束时磁盘上只剩 n2（n1 已被作者删掉）。
	if err := pm.WriteOutlines(&types.OutlineFile{Nodes: []types.OutlineNode{
		{ID: "n2", OrderIndex: 2, Summary: "二", Status: types.OutlinePlanned},
	}}); err != nil {
		t.Fatalf("预置大纲: %v", err)
	}

	newGateEmptyApp().mergeChapterWriteBack(pm, "n1", "本次生成摘要", types.OutlineDone)

	of, err := pm.ReadOutlines()
	if err != nil {
		t.Fatalf("读大纲: %v", err)
	}
	if len(of.Nodes) != 1 || of.Nodes[0].ID != "n2" {
		t.Fatalf("被删的目标节点不得被加回: %+v", of.Nodes)
	}
	if of.Nodes[0].Summary != "二" || of.Nodes[0].Status != types.OutlinePlanned {
		t.Fatalf("其余节点不应被本次回写触碰: %+v", of.Nodes[0])
	}
}

// TestEnsureChapterNode_WriteFailureAborts N8：节点落盘失败必须返回 error
// （调用方 createChapter 据此中止本次生成），不得静默把章节当成已建好。
// 夹具：把 outline.json 换成同名目录——原子写的改名覆盖必然失败。
func TestEnsureChapterNode_WriteFailureAborts(t *testing.T) {
	pm := newGateProject(t)
	of := beginChapterGen(t, pm)

	outlinePath := filepath.Join(pm.Dir, "outline.json")
	if err := os.Remove(outlinePath); err != nil {
		t.Fatalf("删 outline.json: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(outlinePath, "block"), 0o755); err != nil {
		t.Fatalf("造写盘失败夹具: %v", err)
	}

	targetNum, nodeID, branch, err := newGateEmptyApp().ensureChapterNode(pm, of, 1, "")
	if err == nil {
		t.Fatalf("节点落盘失败必须返回 error（否则生成会继续跑在没写进大纲的章上）")
	}
	if nodeID != "" || targetNum != 0 || branch != "" {
		t.Fatalf("失败时不应返回可用定号结果: num=%d id=%q branch=%q", targetNum, nodeID, branch)
	}
}

// TestEnsureChapterNode_CreatePersists 反向守卫：正常盘况下新建节点确实落盘、
// 返回 nodeID（避免上面的 0 值返回被当成「反正不写也对」）。
func TestEnsureChapterNode_CreatePersists(t *testing.T) {
	pm := newGateProject(t)
	of := beginChapterGen(t, pm)

	targetNum, nodeID, branch, err := newGateEmptyApp().ensureChapterNode(pm, of, 3, "")
	if err != nil {
		t.Fatalf("正常盘况不应报错: %v", err)
	}
	if targetNum != 3 || nodeID == "" || branch != "" {
		t.Fatalf("定号/建节点结果异常: num=%d id=%q branch=%q", targetNum, nodeID, branch)
	}
	stored := outlineNode(t, pm, nodeID)
	if stored == nil || stored.OrderIndex != 3 || stored.Status != types.OutlineWriting {
		t.Fatalf("新节点应已落盘: %+v", stored)
	}
}

// TestCancelPartialSidecar_UniqueFileNames M5：同刻两次取消各留一份残稿，
// 后一份不覆盖前一份。
func TestCancelPartialSidecar_UniqueFileNames(t *testing.T) {
	pm := newGateProject(t)

	first, err := writeCancelledPartialSidecar(pm, 1, "", "第一份残稿。")
	if err != nil {
		t.Fatalf("第一次落残稿: %v", err)
	}
	second, err := writeCancelledPartialSidecar(pm, 1, "", "第二份残稿。")
	if err != nil {
		t.Fatalf("第二次落残稿: %v", err)
	}
	if first == second {
		t.Fatalf("两次取消不得落到同一残稿文件: %s", first)
	}
	if n := len(partialSidecars(t, pm.Dir, "001")); n != 2 {
		t.Fatalf("应有两份残稿侧车，得到 %d", n)
	}
	for path, want := range map[string]string{first: "第一份残稿。", second: "第二份残稿。"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("读残稿 %s: %v", path, err)
		}
		if string(data) != want {
			t.Fatalf("残稿 %s 内容被覆盖: %q", filepath.Base(path), data)
		}
	}
}

// TestDeSlopRewriteWithin_MatchesManualPathBeforeScore G8：生成链与手动去味路径
// 共用同一 helper，同文本 before/after 分一致；且与「传 nil 让内核自算 before」
// 的旧生成链口径不同（旧口径不含白名单豁免，before 被抬高 → after<before 闸偏松）。
func TestDeSlopRewriteWithin_MatchesManualPathBeforeScore(t *testing.T) {
	text := "她的眸光流转，带着几分笑意，一步步走来。"
	wl := []string{"眸光流转"}

	// 生成链（新）：helper 内先算 before 分并应用白名单。
	_, repNew, err := deSlopRewriteWithin(text, wl)
	if err != nil {
		t.Fatalf("deSlopRewriteWithin: %v", err)
	}

	// 手动去味路径的现有算法（先 ScoreTextNoRef + ApplyWhitelist 再传入）。
	score, err := novelstyle.ScoreTextNoRef(text)
	if err != nil {
		t.Fatalf("打分: %v", err)
	}
	novelstyle.ApplyWhitelist(score, text, wl)
	_, repManual, err := novelstyle.DeSlopRewriteEx(text, score, wl)
	if err != nil {
		t.Fatalf("手动路径改写: %v", err)
	}
	if repNew.BeforeScore != repManual.BeforeScore {
		t.Fatalf("两路径 before 分应一致: 生成链=%d 手动=%d", repNew.BeforeScore, repManual.BeforeScore)
	}
	if repNew.AfterScore != repManual.AfterScore {
		t.Fatalf("两路径 after 分应一致: 生成链=%d 手动=%d", repNew.AfterScore, repManual.AfterScore)
	}

	// 旧生成链口径（nil → 内核自算不豁免的 before）确实不同：本断言即回归钉。
	_, repNil, err := novelstyle.DeSlopRewriteEx(text, nil, wl)
	if err != nil {
		t.Fatalf("nil before 改写: %v", err)
	}
	if repNil.BeforeScore <= repNew.BeforeScore {
		t.Fatalf("旧 nil 口径的 before 应更高（未豁免白名单），得到 nil=%d 新=%d",
			repNil.BeforeScore, repNew.BeforeScore)
	}
}

// TestAppendProjectChapters_ReducedOutlineNoOverwrite G2：大纲被「续写」整体
// 替换成 5 个卷节点（OrderIndex 重新 1..5）而磁盘已有 1..8 章时，补下不得写进
// 已有的 6..8，必须顺延到 9 起；返回总量按实际最大章号报。
func TestAppendProjectChapters_ReducedOutlineNoOverwrite(t *testing.T) {
	dir := t.TempDir()
	base, err := createImportedProject(dir, "补下基准", "未分类", "默认", []importChapter{
		{Title: "第1章", Content: "一"}, {Title: "第2章", Content: "二"},
		{Title: "第3章", Content: "三"}, {Title: "第4章", Content: "四"},
		{Title: "第5章", Content: "五"}, {Title: "第6章", Content: "六"},
		{Title: "第7章", Content: "七"}, {Title: "第8章", Content: "八"},
	}, bookimport.Report{SplitStrategy: "booksource"})
	if err != nil {
		t.Fatalf("建基线项目: %v", err)
	}

	// 「续写大纲」把章节节点整体替换成 5 个卷节点：顶层 max(OrderIndex) 退回 5。
	reduced := &types.OutlineFile{Nodes: []types.OutlineNode{}}
	for i := 1; i <= 5; i++ {
		reduced.Nodes = append(reduced.Nodes, types.OutlineNode{
			ID:         "vol-" + strconv.Itoa(i),
			Title:      "第" + strconv.Itoa(i) + "卷",
			OrderIndex: i,
			Status:     types.OutlinePlanned,
		})
	}
	pm, err := project.Open(base.Path)
	if err != nil {
		t.Fatalf("打开项目: %v", err)
	}
	if err := pm.WriteOutlines(reduced); err != nil {
		t.Fatalf("写回缩减大纲: %v", err)
	}
	if err := pm.Close(); err != nil {
		t.Fatalf("关闭项目: %v", err)
	}

	res, err := appendProjectChapters(base.Path, booksource.DownloadReport{
		Chapters: []booksource.ChapterText{
			{Title: "补下甲", Order: 1, Paragraphs: []string{"补下甲正文"}},
			{Title: "补下乙", Order: 2, Paragraphs: []string{"补下乙正文"}},
		},
	})
	if err != nil {
		t.Fatalf("补下: %v", err)
	}
	if res.Appended != 2 {
		t.Fatalf("应追加 2 章，得到 %d", res.Appended)
	}
	if res.TotalChapters != 10 {
		t.Fatalf("总量应为追加后的实际最大章号 10，得到 %d", res.TotalChapters)
	}

	pm2, err := project.Open(base.Path)
	if err != nil {
		t.Fatalf("重开项目: %v", err)
	}
	defer pm2.Close()
	for num, want := range map[int]string{
		6: "六", 7: "七", 8: "八", 9: "补下甲正文", 10: "补下乙正文",
	} {
		got, rerr := pm2.ReadChapter(num)
		if rerr != nil {
			t.Fatalf("读第 %d 章: %v", num, rerr)
		}
		if got != want {
			t.Fatalf("第 %d 章被改写：want %q got %q", num, want, got)
		}
	}
	of, err := pm2.ReadOutlines()
	if err != nil {
		t.Fatalf("读大纲: %v", err)
	}
	if len(of.Nodes) != 7 { // 5 个卷节点 + 2 个新章节点
		t.Fatalf("大纲节点数应为 7（卷节点保留 + 新增 2 章），得到 %d: %+v", len(of.Nodes), of.Nodes)
	}
	for i := 1; i <= 5; i++ {
		if of.Nodes[i-1].ID != "vol-"+strconv.Itoa(i) {
			t.Fatalf("既有卷节点不得被动: %+v", of.Nodes[i-1])
		}
	}
}

// TestAppendProjectChapters_OutlineWithHoleSkipsOccupied 大纲章号与磁盘章号都
// 不连续（大纲有 001/003、磁盘有 002）时，补下必须落到下一个真正空的章号 004。
func TestAppendProjectChapters_OutlineWithHoleSkipsOccupied(t *testing.T) {
	dir := t.TempDir()
	base, err := createImportedProject(dir, "空洞基准", "未分类", "默认", []importChapter{
		{Title: "第1章", Content: "一"}, {Title: "第2章", Content: "二"},
	}, bookimport.Report{SplitStrategy: "booksource"})
	if err != nil {
		t.Fatalf("建基线项目: %v", err)
	}
	pm, err := project.Open(base.Path)
	if err != nil {
		t.Fatalf("打开项目: %v", err)
	}
	if err := pm.WriteOutlines(&types.OutlineFile{Nodes: []types.OutlineNode{
		{ID: "imp-001", OrderIndex: 1, Title: "第1章", Status: types.OutlineDone},
		{ID: "imp-003", OrderIndex: 3, Title: "第3章", Status: types.OutlineDone},
	}}); err != nil {
		t.Fatalf("写空洞大纲: %v", err)
	}
	if err := pm.WriteChapter(3, "磁盘上的第三章"); err != nil {
		t.Fatalf("预置第3章: %v", err)
	}
	if err := pm.Close(); err != nil {
		t.Fatalf("关闭项目: %v", err)
	}

	res, err := appendProjectChapters(base.Path, booksource.DownloadReport{
		Chapters: []booksource.ChapterText{
			{Title: "补下", Order: 1, Paragraphs: []string{"补下正文"}},
		},
	})
	if err != nil {
		t.Fatalf("补下: %v", err)
	}
	if res.Appended != 1 || res.TotalChapters != 4 {
		t.Fatalf("应追加到第 4 章: %+v", res)
	}

	pm2, err := project.Open(base.Path)
	if err != nil {
		t.Fatalf("重开项目: %v", err)
	}
	defer pm2.Close()
	if got, _ := pm2.ReadChapter(2); got != "二" {
		t.Fatalf("既有第 2 章被覆盖: %q", got)
	}
	if got, _ := pm2.ReadChapter(3); got != "磁盘上的第三章" {
		t.Fatalf("既有第 3 章被覆盖: %q", got)
	}
	if got, _ := pm2.ReadChapter(4); got != "补下正文" {
		t.Fatalf("补下应落到第 4 章: %q", got)
	}
}

// TestAppendChapterBaseline_TakesMaxOfOutlineAndDisk G2 的基准本身：递归大纲
// 最大章号与 chapters/ 磁盘已有最大章号取大。旧实现只取大纲**顶层**
// max(OrderIndex)，在大纲被「续写」重排（顶层退回 1..5）后基准退回 5 → 覆盖
// 磁盘已有正稿。
func TestAppendChapterBaseline_TakesMaxOfOutlineAndDisk(t *testing.T) {
	pm := newGateProject(t)
	if err := pm.WriteChapter(6, "磁盘上的第六章"); err != nil {
		t.Fatalf("预置第六章: %v", err)
	}
	// 嵌套大纲：章节点藏在卷节点的 Children 里（顶层 OrderIndex 只有 1）。
	of := &types.OutlineFile{Nodes: []types.OutlineNode{
		{ID: "vol-1", OrderIndex: 1, Title: "卷一", Children: []types.OutlineNode{
			{ID: "ch-005", OrderIndex: 5, ChapterFile: "005.md", Title: "第5章"},
		}},
		{ID: "vol-2", OrderIndex: 2, Title: "卷二"},
	}}
	if err := pm.WriteOutlines(of); err != nil {
		t.Fatalf("预置嵌套大纲: %v", err)
	}

	got, err := appendChapterBaseline(pm, of)
	if err != nil {
		t.Fatalf("appendChapterBaseline: %v", err)
	}
	if got != 6 {
		t.Fatalf("基准应为 max(递归大纲 5, 磁盘 6) = 6，得到 %d", got)
	}

	// 磁盘更高：补下必须从 7 起。
	if n := nextFreeChapterNum(pm, of, got); n != 7 {
		t.Fatalf("下一个空章号应为 7，得到 %d", n)
	}

	// 磁盘侧还能读出「只有场景、没有 blob」的 v4 章（否则会写进该章）。
	if err := pm.MigrateV3ToV4(); err != nil {
		t.Fatalf("迁移 v4: %v", err)
	}
	sm := pm.SceneManager(9)
	sc, err := sm.Create("opening", "开场")
	if err != nil {
		t.Fatalf("建场景: %v", err)
	}
	sc.Content = "第九章场景正文。"
	if err := sm.Write(sc); err != nil {
		t.Fatalf("写场景: %v", err)
	}
	diskMax, err := pm.MaxChapterBodyNum()
	if err != nil {
		t.Fatalf("MaxChapterBodyNum: %v", err)
	}
	if diskMax < 9 {
		t.Fatalf("只有场景承载正文的 v4 章也应计入磁盘最大章号，得到 %d", diskMax)
	}
}

// TestAppendChapterBaseline_IgnoresBranchOnlyFile 分支正稿 NNNa.md 不计入主线
// 章号基准（避免把分支章号当主线续编起点）。
func TestAppendChapterBaseline_IgnoresBranchOnlyFile(t *testing.T) {
	pm := newGateProject(t)
	if err := pm.WriteChapterBranch(7, "a", "分支正文"); err != nil {
		t.Fatalf("写分支章: %v", err)
	}
	of := &types.OutlineFile{Nodes: []types.OutlineNode{{ID: "n1", OrderIndex: 2}}}
	got, err := appendChapterBaseline(pm, of)
	if err != nil {
		t.Fatalf("appendChapterBaseline: %v", err)
	}
	if got != 2 {
		t.Fatalf("分支章不应抬高主线基准（应为大纲的 2），得到 %d", got)
	}
}

// TestBuildAutoGateReport_NoSyncResultWhenAnalysisFails G3 消费端口径：分析路
// 未成立时报告里的 foreshadowSync 必须保持 nil，绝不拿上一轮 lastSync 的计数
// 冒充本轮结果（chapter-gate 报告只报「本轮真的发生了什么」）。
//
// 注：persisted/errors 两个字段的正向映射见 novel_book_health.go 的 map 构造；
// 失败侧语义（errors 非空 ⇒ SyncPersisted=false）由 internal/analysis 的
// TestSyncForeshadows_WriteFailureReported 直接钉住。
func TestBuildAutoGateReport_NoSyncResultWhenAnalysisFails(t *testing.T) {
	a := newFingerprintTestApp(t)
	a.analysisAgent = analysis.New(ai.NewClient(&config.Config{}), a.getPM(),
		&config.Config{}, prompt.NewEngine("../../prompts"))
	mustWriteChapter(t, a, 1, "干净样本。")

	// 先造一轮「落盘失败」的同步：lastSync 已带错误（defer 登记，G3）。
	pmDir := a.getPM().Dir
	path := filepath.Join(pmDir, "foreshadows.json")
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("清理 foreshadows.json: %v", err)
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("造写盘失败夹具 %q: %v", path, err)
	}
	res := a.analysisAgent.SyncForeshadows(1, []types.ForeshadowHit{
		{Type: "planted", Content: "落盘失败的伏笔"},
	})
	if len(res.Errors) == 0 || res.SyncPersisted() {
		t.Fatalf("夹具应造出「落盘失败」的一轮: %+v", res)
	}

	pm := a.getPM()
	content, err := pm.ReadChapter(1)
	if err != nil {
		t.Fatalf("读章: %v", err)
	}
	// 无可用引擎 → Analyze 失败 → analysisDone=false，且不得带上一轮的同步结果。
	report := a.buildAutoGateReport(pm, 1, content, "")
	if report["analysisDone"] != false {
		t.Fatalf("无引擎时分析路应降级 false: %+v", report)
	}
	if report["foreshadowSync"] != nil {
		t.Fatalf("分析未成立时不得带同步结果（尤其不得拿上一轮计数冒充）: %+v", report["foreshadowSync"])
	}
}

// TestEnginesSave_NoFixedTempNameLeftovers G6：引擎规则保存改走 AtomicWrite
// （随机名临时文件），成功后不留固定名 .tmp，且并发保存最终文件是完整合法 JSON
// （旧实现的固定名 tmp 会互写/互删）。
func TestEnginesSave_NoFixedTempNameLeftovers(t *testing.T) {
	dir := t.TempDir()
	good := `[{"name":"bing","url":"https://www.bing.com/search?q=%s","result":".b_algo","title":"h2 a"}]`
	if _, err := enginesSave(dir, good); err != nil {
		t.Fatalf("首次保存: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读目录: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("不应残留固定名临时文件: %s", e.Name())
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = enginesSave(dir, `[{"name":"e`+strconv.Itoa(i)+
				`","url":"https://x`+strconv.Itoa(i)+`.test/s?q=%s","result":".r","title":"a"}]`)
		}(i)
	}
	wg.Wait()

	got, err := enginesGet(dir)
	if err != nil {
		t.Fatalf("并发保存后应可读回完整规则: %v", err)
	}
	if len(got.Rules) != 1 || got.Rules[0].Name == "" {
		t.Fatalf("回读规则异常: %+v", got.Rules)
	}
}
