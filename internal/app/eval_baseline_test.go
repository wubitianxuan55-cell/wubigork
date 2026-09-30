package app

// 长篇刀7测试：评测基线（确定性快照/聚合造数/基线对比 Δ 方向/stale）。
// 规格 进度计划/gaea-novel-eval-baseline-20260930.md。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// evalTestProject 带三章样本的项目（第三章 AI 味显著偏重）+ 两条伏笔（一回收一埋设）。
func evalTestProject(t *testing.T) (*App, *project.Manager) {
	t.Helper()
	a := newGateEmptyApp()
	pm := newGateProject(t)
	a.setPM(pm)
	good := "他推门进来，雨还没停。她没抬头，手里的针线也没停。灯芯短了一截，屋里暗下来。"
	heavy := strings.Repeat("他感到无比的震惊。此外，值得注意的是，这一切仿佛命运的安排。", 10)
	for i, body := range []string{good, good, heavy} {
		if err := pm.WriteChapter(i+1, body); err != nil {
			t.Fatalf("写章 %d: %v", i+1, err)
		}
	}
	if err := pm.WriteForeshadows(&types.ForeshadowFile{SchemaVersion: 2, Items: []types.Foreshadow{
		{ID: "f1", Category: "plot", Description: "已回收的伏笔", PlantedIn: "001.md", Status: types.ForeshadowRevealed},
		{ID: "f2", Category: "plot", Description: "仍埋设的伏笔", PlantedIn: "002.md", Status: types.ForeshadowPlanted},
	}}); err != nil {
		t.Fatalf("写伏笔: %v", err)
	}
	return a, pm
}

// TestEvalSnapshotDeterministic 同一状态两次产出逐字节相同（body 无时钟）。
func TestEvalSnapshotDeterministic(t *testing.T) {
	a, _ := evalTestProject(t)
	b1 := a.buildEvalSnapshot(a.getPM())
	b2 := a.buildEvalSnapshot(a.getPM())
	j1, _ := json.Marshal(b1)
	j2, _ := json.Marshal(b2)
	if string(j1) != string(j2) {
		t.Fatalf("快照必须确定性（除时间戳外逐字节相同）：\n%s\n%s", j1, j2)
	}
}

// TestEvalSnapshotAggregation 聚合正确：最差章=AI 味最重的第 3 章；回收率=1/2；
// 伏笔计数与造数一致。
func TestEvalSnapshotAggregation(t *testing.T) {
	a, pm := evalTestProject(t)
	body := a.buildEvalSnapshot(pm)
	if body.Chapters != 3 {
		t.Fatalf("章数应 3：%d", body.Chapters)
	}
	if body.Taste.Worst != 3 {
		t.Fatalf("最差章应是第 3 章（AI 味最重）：%d", body.Taste.Worst)
	}
	if body.Foreshadow.Items != 2 || body.Foreshadow.Revealed != 1 {
		t.Fatalf("伏笔计数应 items=2 revealed=1：%+v", body.Foreshadow)
	}
	if body.Foreshadow.Recall != 0.5 {
		t.Fatalf("回收率应 0.5：%v", body.Foreshadow.Recall)
	}
	if body.PromptSetHash == "" || body.PromptSetHash == "unknown" {
		t.Fatalf("promptSetHash 应非空：%q", body.PromptSetHash)
	}
}

// TestEvalBaselineAndCompare 基线落盘→状态变化→Δ 方向正确（AI 味均值 worse=变差）；
// baseline hash 篡改 → stale。
func TestEvalBaselineAndCompare(t *testing.T) {
	a, pm := evalTestProject(t)
	if _, err := a.NovelEvalBaselineSet(); err != nil {
		t.Fatalf("设基线: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pm.Dir, "eval", "baseline.json")); err != nil {
		t.Fatalf("基线应落盘: %v", err)
	}
	// 无对比基线时报错（新项目）
	a2 := newGateEmptyApp()
	pm2 := newGateProject(t)
	a2.setPM(pm2)
	if _, err := a2.NovelEvalCompare(); err == nil || !strings.Contains(err.Error(), "基线") {
		t.Fatalf("无基线应报错：%v", err)
	}

	// 状态变化：把好章换成重 AI 味文本 → AI 味均值上升=worse
	if err := pm.WriteChapter(1, strings.Repeat("他感到无比的震惊。此外，这一切仿佛命运的安排。", 10)); err != nil {
		t.Fatal(err)
	}
	res, err := a.NovelEvalCompare()
	if err != nil {
		t.Fatalf("对比: %v", err)
	}
	if res["stale"] != false {
		t.Fatalf("同 prompts 不应 stale：%v", res["stale"])
	}
	items := res["items"].([]map[string]interface{})
	meanItem := map[string]interface{}{}
	for _, it := range items {
		if it["metric"] == "AI 味均值" {
			meanItem = it
		}
	}
	if meanItem["dir"] != "worse" {
		t.Fatalf("AI 味均值上升应为 worse：%v", meanItem)
	}

	// 篡改 baseline 的 promptSetHash → stale=true（禁止直接对比）
	basePath := filepath.Join(pm.Dir, "eval", "baseline.json")
	raw, _ := os.ReadFile(basePath)
	var bj map[string]interface{}
	_ = json.Unmarshal(raw, &bj)
	bj["prompt_set_hash"] = "tampered"
	raw2, _ := json.Marshal(bj)
	if err := os.WriteFile(basePath, raw2, 0644); err != nil {
		t.Fatal(err)
	}
	res2, err := a.NovelEvalCompare()
	if err != nil {
		t.Fatalf("对比2: %v", err)
	}
	if res2["stale"] != true {
		t.Fatalf("hash 不一致必须 stale：%v", res2["stale"])
	}
}

// TestEvalSnapshotPersist persist=true 落 eval/snapshots/；零章项目空快照不炸。
func TestEvalSnapshotPersist(t *testing.T) {
	a, pm := evalTestProject(t)
	res, err := a.NovelEvalSnapshot(true)
	if err != nil {
		t.Fatalf("快照: %v", err)
	}
	name, _ := res["savedAs"].(string)
	if name == "" {
		t.Fatal("persist 应返回文件名")
	}
	if _, err := os.Stat(filepath.Join(pm.Dir, "eval", "snapshots", name)); err != nil {
		t.Fatalf("快照应落盘: %v", err)
	}
	// 零章项目：空快照（chapters=0）可用不炸
	a2 := newGateEmptyApp()
	pm2 := newGateProject(t)
	a2.setPM(pm2)
	res2, err := a2.NovelEvalSnapshot(false)
	if err != nil || res2["chapters"] != 0 {
		t.Fatalf("零章项目应空快照：%v err=%v", res2, err)
	}
}
