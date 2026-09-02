package app

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// newForeshadowTestApp 构造带临时小说项目的测试 App（不依赖 LLM）。
func newForeshadowTestApp(t *testing.T) *App {
	t.Helper()
	a := newCharacterLibTestApp(t)
	pm, err := project.Create(filepath.Join(t.TempDir(), "novel"), "测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	a.setPM(pm)
	return a
}

// mustSaveForeshadows 调 SaveForeshadows（失败即终止）。
func mustSaveForeshadows(t *testing.T, a *App, items []types.Foreshadow) {
	t.Helper()
	payload, err := json.Marshal(items)
	if err != nil {
		t.Fatalf("序列化伏笔: %v", err)
	}
	if err := a.SaveForeshadows(string(payload)); err != nil {
		t.Fatalf("SaveForeshadows: %v", err)
	}
}

// getForeshadowItems 走 GetForeshadows 绑定读回条目（失败即终止）。
func getForeshadowItems(t *testing.T, a *App) []types.Foreshadow {
	t.Helper()
	res := a.GetForeshadows()
	if res == nil {
		t.Fatal("GetForeshadows 返回 nil")
	}
	items, ok := res["items"].([]types.Foreshadow)
	if !ok {
		t.Fatalf("GetForeshadows items 类型异常: %T", res["items"])
	}
	return items
}

// TestSaveForeshadows_RoundTrip 写读回环：保存→GetForeshadows 读回字段一致；
// 再改状态重写→读回为新状态。空 ID 条目由后端兜底补 manual_ 前缀 ID。
func TestSaveForeshadows_RoundTrip(t *testing.T) {
	a := newForeshadowTestApp(t)

	mustSaveForeshadows(t, a, []types.Foreshadow{
		{ID: "manual_1725000000000", Category: "plot", Description: "神秘铜匣",
			PlantedIn: "001.md", Status: types.ForeshadowPlanted, IsLongTerm: true},
		{ID: "plot_001_abc", Category: "character", Description: "左臂旧伤",
			PlantedIn: "001.md", Status: types.ForeshadowRevealed, RevealedIn: "005.md"},
		{Category: "world", Description: "无 ID 手工条目", PlantedIn: "002.md",
			Status: types.ForeshadowPlanted}, // 空 ID → 后端补 manual_ 前缀
	})

	items := getForeshadowItems(t, a)
	if len(items) != 3 {
		t.Fatalf("应读回 3 条，实际 %d: %+v", len(items), items)
	}
	manual := findByAppID(items, "manual_1725000000000")
	if manual == nil || manual.Description != "神秘铜匣" || manual.Status != types.ForeshadowPlanted ||
		!manual.IsLongTerm || manual.PlantedIn != "001.md" {
		t.Fatalf("手工条目回环不一致: %+v", manual)
	}
	ai := findByAppID(items, "plot_001_abc")
	if ai == nil || ai.Status != types.ForeshadowRevealed || ai.RevealedIn != "005.md" {
		t.Fatalf("AI 条目回环不一致: %+v", ai)
	}

	// 空 ID 兜底：manual_ 前缀
	filled := 0
	for _, it := range items {
		if strings.HasPrefix(it.ID, "manual_") {
			filled++
		}
	}
	if filled != 2 {
		t.Fatalf("应有 2 条 manual_ 前缀条目（1 显式 + 1 空 ID 兜底），实际 %d: %+v", filled, items)
	}

	// 状态流转写回：planted → hinted
	manual.Status = types.ForeshadowHinted
	mustSaveForeshadows(t, a, items)
	items = getForeshadowItems(t, a)
	if got := findByAppID(items, "manual_1725000000000"); got == nil || got.Status != types.ForeshadowHinted {
		t.Fatalf("状态流转写回失败: %+v", got)
	}
}

// TestSaveForeshadows_RejectsInvalidStatus 非法 Status 拒收，且原文件不被破坏。
func TestSaveForeshadows_RejectsInvalidStatus(t *testing.T) {
	a := newForeshadowTestApp(t)
	mustSaveForeshadows(t, a, []types.Foreshadow{
		{ID: "manual_1", Category: "plot", Description: "铜匣", PlantedIn: "001.md",
			Status: types.ForeshadowPlanted},
	})

	for _, bad := range []types.ForeshadowStatus{"done", ""} {
		err := a.SaveForeshadows(`[{"id":"manual_2","category":"plot","description":"坏状态","planted_in":"002.md","status":"` + string(bad) + `"}]`)
		if err == nil || !strings.Contains(err.Error(), "状态非法") {
			t.Fatalf("非法状态 %q 应拒收: %v", bad, err)
		}
	}

	// 拒收后原文件不受影响（全量替换语义不落半截数据）
	items := getForeshadowItems(t, a)
	if len(items) != 1 || items[0].ID != "manual_1" {
		t.Fatalf("拒收后原文件应保持不变: %+v", items)
	}
}

// TestSaveForeshadows_RejectsDuplicateID 全量写回中 ID 重复应拒收。
func TestSaveForeshadows_RejectsDuplicateID(t *testing.T) {
	a := newForeshadowTestApp(t)
	err := a.SaveForeshadows(`[{"id":"manual_1","category":"plot","description":"A","planted_in":"001.md","status":"planted"},
		{"id":"manual_1","category":"plot","description":"B","planted_in":"001.md","status":"planted"}]`)
	if err == nil || !strings.Contains(err.Error(), "ID 重复") {
		t.Fatalf("重复 ID 应拒收: %v", err)
	}
}

// TestSaveForeshadows_RequiresProject 未打开项目时报错。
func TestSaveForeshadows_RequiresProject(t *testing.T) {
	a := newCharacterLibTestApp(t)
	if err := a.SaveForeshadows(`[]`); err == nil || !strings.Contains(err.Error(), "请先打开项目") {
		t.Fatalf("无项目应报错: %v", err)
	}
}

// TestSaveForeshadows_ClearAll 空数组全量清空（items 落盘为 [] 而非 null）。
func TestSaveForeshadows_ClearAll(t *testing.T) {
	a := newForeshadowTestApp(t)
	mustSaveForeshadows(t, a, []types.Foreshadow{
		{ID: "manual_1", Category: "plot", Description: "铜匣", PlantedIn: "001.md",
			Status: types.ForeshadowPlanted},
	})
	mustSaveForeshadows(t, a, []types.Foreshadow{})
	items := getForeshadowItems(t, a)
	if len(items) != 0 {
		t.Fatalf("清空写回后应为 0 条: %+v", items)
	}
}

// findByAppID 在 app 包测试内按 ID 查找条目。
func findByAppID(items []types.Foreshadow, id string) *types.Foreshadow {
	for i := range items {
		if items[i].ID == id {
			return &items[i]
		}
	}
	return nil
}
