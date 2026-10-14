package app

// ── 伏笔到期预警 测试（阶段八 8.6）──────────────────────────
//
// 出口判据「超期伏笔有可见预警」的服务端面：判定复用 types.UrgencyLevel 单源
// （不重复实现——测的是分桶/口径/空态，不是阈值）。

import (
	"testing"

	"github.com/gaea/gaea/internal/types"
)

func TestForeshadowDue_Buckets(t *testing.T) {
	a, pm := newFinalizeTestApp(t)

	if err := pm.WriteForeshadows(&types.ForeshadowFile{Items: []types.Foreshadow{
		{ID: "due1", Title: "本章必须回收", Status: types.ForeshadowPlanted, PlantedIn: "001.md", TargetResolveIn: "005.md"},
		{ID: "over1", Title: "已超期三章", Status: types.ForeshadowPlanted, PlantedIn: "001.md", TargetResolveIn: "002.md"},
		{ID: "soon1", Title: "临近需关注", Status: types.ForeshadowHinted, PlantedIn: "001.md", TargetResolveIn: "008.md"},
		{ID: "far1", Title: "远期不预警", Status: types.ForeshadowPlanted, PlantedIn: "001.md", TargetResolveIn: "050.md"},
		{ID: "done1", Title: "已回收不预警", Status: types.ForeshadowRevealed, PlantedIn: "001.md", TargetResolveIn: "002.md", RevealedIn: "002.md"},
		{ID: "no1", Title: "无计划不猜", Status: types.ForeshadowPlanted, PlantedIn: "001.md"},
	}}); err != nil {
		t.Fatal(err)
	}

	rep, err := a.ForeshadowDue(5)
	if err != nil {
		t.Fatalf("预警: %v", err)
	}
	if rep.ChapterNum != 5 {
		t.Fatalf("章号回显: %d", rep.ChapterNum)
	}
	// Urgent = 本章必须回收（urgency2）+ 已超期（urgency3）
	if len(rep.Urgent) != 2 {
		t.Fatalf("Urgent 应 2 条: %+v", rep.Urgent)
	}
	overdueOK := false
	for _, it := range rep.Urgent {
		if it.ID == "over1" {
			if it.Urgency != 3 || it.OverdueBy != 3 {
				t.Fatalf("超期条目口径: %+v", it)
			}
			overdueOK = true
		}
	}
	if !overdueOK {
		t.Fatalf("超期条目应在 Urgent: %+v", rep.Urgent)
	}
	// Soon = 临近（target 8 - current 5 = 3 ≤ RemindBefore 5）
	if len(rep.Soon) != 1 || rep.Soon[0].ID != "soon1" {
		t.Fatalf("Soon 应只有 soon1: %+v", rep.Soon)
	}
}

func TestForeshadowDue_EmptyAndFallback(t *testing.T) {
	a, pm := newFinalizeTestApp(t)

	// 空登记=空报告正常态（不报错）
	rep, err := a.ForeshadowDue(5)
	if err != nil || len(rep.Urgent) != 0 || len(rep.Soon) != 0 {
		t.Fatalf("空登记应空报告: %+v err=%v", rep, err)
	}

	// chapterNum<=0 回退全书已写章数
	if err := pm.WriteChapter(1, "第一章。"); err != nil {
		t.Fatal(err)
	}
	if err := pm.WriteChapter(2, "第二章。"); err != nil {
		t.Fatal(err)
	}
	if err := pm.WriteForeshadows(&types.ForeshadowFile{Items: []types.Foreshadow{
		{ID: "over1", Title: "超期", Status: types.ForeshadowPlanted, PlantedIn: "001.md", TargetResolveIn: "001.md"},
	}}); err != nil {
		t.Fatal(err)
	}
	rep, err = a.ForeshadowDue(0)
	if err != nil {
		t.Fatal(err)
	}
	if rep.ChapterNum != 2 {
		t.Fatalf("应回退已写章数 2: %d", rep.ChapterNum)
	}
	if len(rep.Urgent) != 1 || rep.Urgent[0].OverdueBy != 1 {
		t.Fatalf("按回退章数判超期: %+v", rep.Urgent)
	}
}
