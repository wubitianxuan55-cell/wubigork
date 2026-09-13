package app

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// 造一个已写 3 章 + 大纲的工程并设为当前工程。
func newReconstructFixture(t *testing.T) (*App, *project.Manager) {
	t.Helper()
	a := newCharacterLibTestApp(t)
	dir := filepath.Join(t.TempDir(), "反推夹具")
	pm, err := project.Create(dir, "反推夹具", "都市", "默认", "")
	if err != nil {
		t.Fatalf("建工程: %v", err)
	}
	nodes := make([]types.OutlineNode, 0, 3)
	for i := 1; i <= 3; i++ {
		if err := pm.WriteChapter(i, strings.Repeat("他推开门，雨还在下。", 20)); err != nil {
			t.Fatalf("写章节 %d: %v", i, err)
		}
		nodes = append(nodes, types.OutlineNode{
			ID:          "imp-" + string(rune('0'+i)),
			Title:       "第" + string(rune('0'+i)) + "章",
			OrderIndex:  i,
			ChapterFile: fmt.Sprintf("%03d.md", i),
			Status:      types.OutlineDone,
		})
	}
	if err := pm.WriteOutlines(&types.OutlineFile{Nodes: nodes}); err != nil {
		t.Fatalf("写大纲: %v", err)
	}
	a.setPM(pm)
	return a, pm
}

func TestNovelOutlineReconstruct_NoModelFallsBackToRules(t *testing.T) {
	a, _ := newReconstructFixture(t)
	preview, err := a.NovelOutlineReconstruct()
	if err != nil {
		t.Fatalf("反推失败: %v", err)
	}
	if preview.AIUsed {
		t.Fatal("无模型时 aiUsed 必须为 false（如实回报）")
	}
	if len(preview.Warnings) == 0 {
		t.Fatal("降级必须有告警说明")
	}
	if len(preview.Items) != 3 {
		t.Fatalf("应逐章返回规则结构: %d", len(preview.Items))
	}
	for i, it := range preview.Items {
		if it.ChapterNumber != i+1 || it.Summary == "" || it.Emotion == "" || len(it.Scenes) != 2 || len(it.KeyPoints) != 2 {
			t.Fatalf("第 %d 章兜底结构不完整: %+v", i+1, it)
		}
	}
	if preview.NarrativePerspective == "" || preview.TargetWords <= 0 || preview.ProjectTitle == "" {
		t.Fatalf("立项兜底信息不完整: %+v", preview)
	}
}

func TestNovelOutlineReconstructApply_IdempotentAndScoped(t *testing.T) {
	a, pm := newReconstructFixture(t)
	items := []OutlineReconstructItem{
		{ChapterNumber: 1, Title: "第1章", Summary: "第一章概要", Scenes: []string{"场景A"}, KeyPoints: []string{"要点A"}, Emotion: "紧张"},
		// 章号 9 不匹配任何节点 ⇒ 不应写入、也不应报错整单失败
		{ChapterNumber: 9, Title: "不存在的章", Summary: "不该出现"},
	}
	payload, _ := json.Marshal(items)

	if _, err := a.NovelOutlineReconstructApply(string(payload)); err != nil {
		_ = err // 允许「只命中 1 章」的成功路径返回 nil/err 皆可，下面断言落盘结果
	}
	outline, err := pm.ReadOutlines()
	if err != nil {
		t.Fatalf("读大纲: %v", err)
	}
	if outline.Nodes[0].Summary != "第一章概要" || outline.Nodes[0].Emotion != "紧张" ||
		len(outline.Nodes[0].SceneIdeas) != 1 || len(outline.Nodes[0].KeyPoints) != 1 {
		t.Fatalf("第 1 章未按预期写入: %+v", outline.Nodes[0])
	}
	if outline.Nodes[1].Summary != "" {
		t.Fatalf("未命中的章不应被改写: %+v", outline.Nodes[1])
	}
	// 幂等：同一载荷再跑一次，结果一字不差
	before, _ := json.Marshal(outline)
	if _, err := a.NovelOutlineReconstructApply(string(payload)); err != nil {
		t.Fatalf("第二次应用应成功（幂等）: %v", err)
	}
	after, _ := json.Marshal(mustOutlines(t, pm))
	if string(before) != string(after) {
		t.Fatalf("幂等性被破坏:\n前 %s\n后 %s", before, after)
	}
	// 空载荷 / 对不上的载荷必须显式报错，不静默成功
	if _, err := a.NovelOutlineReconstructApply("[]"); err == nil {
		t.Fatal("空载荷应报错")
	}
	if _, err := a.NovelOutlineReconstructApply(`[{"chapterNumber":42,"title":"x"}]`); err == nil {
		t.Fatal("全部对不上时应报错（不静默成功）")
	}
}

func mustOutlines(t *testing.T, pm *project.Manager) *types.OutlineFile {
	t.Helper()
	outline, err := pm.ReadOutlines()
	if err != nil {
		t.Fatalf("读大纲: %v", err)
	}
	return outline
}
