package analysis

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/types"
)

// newSyncTestAgent 构造带临时项目的分析 Agent（syncForeshadows 不依赖 LLM，client 可为 nil）。
func newSyncTestAgent(t *testing.T) *Agent {
	t.Helper()
	pm, err := project.Create(filepath.Join(t.TempDir(), "novel"), "测试", "玄幻", "", "")
	if err != nil {
		t.Fatalf("创建项目: %v", err)
	}
	return New(nil, pm, &config.Config{}, nil)
}

// seedForeshadows 写入初始伏笔文件（失败即终止）。
func seedForeshadows(t *testing.T, a *Agent, items ...types.Foreshadow) {
	t.Helper()
	if err := a.pm.WriteForeshadows(&types.ForeshadowFile{Items: items}); err != nil {
		t.Fatalf("写伏笔文件: %v", err)
	}
}

// readForeshadowItems 读回伏笔条目（失败即终止）。
func readForeshadowItems(t *testing.T, a *Agent) []types.Foreshadow {
	t.Helper()
	ff, err := a.pm.ReadForeshadows()
	if err != nil {
		t.Fatalf("读伏笔文件: %v", err)
	}
	return ff.Items
}

// findByID 按 ID 查找条目。
func findByID(items []types.Foreshadow, id string) *types.Foreshadow {
	for i := range items {
		if items[i].ID == id {
			return &items[i]
		}
	}
	return nil
}

// TestSyncForeshadows_MergesByIDAndKeepsManual 覆盖核心合并语义：
// AI 结果更新既有 ID 的状态、追加新 ID，manual_ 手工条目原样保留（仅可被描述匹配推进状态）。
func TestSyncForeshadows_MergesByIDAndKeepsManual(t *testing.T) {
	a := newSyncTestAgent(t)
	manual := types.Foreshadow{
		ID: "manual_1725000000000", Category: "plot", Description: "神秘铜匣的钥匙",
		PlantedIn: "001.md", Status: types.ForeshadowPlanted, IsLongTerm: true,
	}
	aiA := types.Foreshadow{
		ID: "plot_001_abc", Category: "character", Description: "主角左臂旧伤",
		PlantedIn: "001.md", Status: types.ForeshadowPlanted,
	}
	seedForeshadows(t, a, manual, aiA)

	result := &AnalysisResult{Foreshadows: []ForeshadowAction{
		// 新伏笔：第 2 章新埋设
		{Category: "world", Action: "planted", Description: "星门在月圆之夜开启"},
		// 既有 AI 条目：按 StableID 推进到 hinted
		{Action: "hinted", StableID: "plot_001_abc", Description: "左臂隐隐作痛"},
		// 手工条目：按描述匹配被 AI 回收（期望：状态推进但 ID/长线标记保留）
		{Action: "revealed", Description: "神秘铜匣的钥匙"},
	}}
	a.syncForeshadows(2, result)

	items := readForeshadowItems(t, a)
	if len(items) != 3 {
		t.Fatalf("应有 3 条伏笔（2 既有 + 1 新增），实际 %d: %+v", len(items), items)
	}

	// 手工条目：ID 与 is_long_term 原样保留，仅状态被 AI 的 revealed 推进
	got := findByID(items, "manual_1725000000000")
	if got == nil {
		t.Fatalf("手工条目被 syncForeshadows 冲掉: %+v", items)
	}
	if got.Status != types.ForeshadowRevealed || got.RevealedIn != "002.md" {
		t.Fatalf("手工条目应被 AI 按描述推进到 revealed/002.md: %+v", got)
	}
	if !got.IsLongTerm || got.PlantedIn != "001.md" {
		t.Fatalf("手工条目其他字段不应被改写: %+v", got)
	}

	// 既有 AI 条目：按 StableID 更新到 hinted
	got = findByID(items, "plot_001_abc")
	if got == nil || got.Status != types.ForeshadowHinted {
		t.Fatalf("既有 AI 条目应更新为 hinted: %+v", got)
	}

	// 新增条目：planted 在 002.md，stable ID
	got = findByID(items, GenerateStableID("world", "002.md", "星门在月圆之夜开启"))
	if got == nil || got.Status != types.ForeshadowPlanted || got.PlantedIn != "002.md" {
		t.Fatalf("新增伏笔缺失或字段错误: %+v", got)
	}
}

// TestSyncForeshadows_ReplayNoDuplicateNoReset 同章同内容重复分析（同一 planted 动作
// 重放）不应产生重复条目，也不应重置既有条目状态。
func TestSyncForeshadows_ReplayNoDuplicateNoReset(t *testing.T) {
	a := newSyncTestAgent(t)
	stableID := GenerateStableID("plot", "003.md", "铜匣出现")
	result := &AnalysisResult{Foreshadows: []ForeshadowAction{
		{Category: "plot", Action: "planted", Description: "铜匣出现"},
	}}

	a.syncForeshadows(3, result)
	// 模拟已有推进：手工把该条目标成 hinted，再次重放同一 planted 动作
	items := readForeshadowItems(t, a)
	if len(items) != 1 {
		t.Fatalf("首次同步后应只有 1 条，实际 %d", len(items))
	}
	items[0].Status = types.ForeshadowHinted
	seedForeshadows(t, a, items...)

	a.syncForeshadows(3, result)

	items = readForeshadowItems(t, a)
	if len(items) != 1 {
		t.Fatalf("重放 planted 不应产生重复条目，实际 %d: %+v", len(items), items)
	}
	if items[0].ID != stableID || items[0].Status != types.ForeshadowHinted {
		t.Fatalf("重放不应重置既有条目: %+v", items[0])
	}
}

// TestSyncForeshadows_CorruptFileNotOverwritten 伏笔文件损坏（读取失败）时放弃同步，
// 不得用空数据覆盖既有文件。
func TestSyncForeshadows_CorruptFileNotOverwritten(t *testing.T) {
	a := newSyncTestAgent(t)
	path := filepath.Join(a.pm.Dir, "foreshadows.json")
	corrupt := []byte("{ not valid json !!")
	if err := os.WriteFile(path, corrupt, 0644); err != nil {
		t.Fatalf("写损坏文件: %v", err)
	}

	a.syncForeshadows(1, &AnalysisResult{Foreshadows: []ForeshadowAction{
		{Category: "plot", Action: "planted", Description: "不应落盘"},
	}})

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读回损坏文件: %v", err)
	}
	if string(after) != string(corrupt) {
		t.Fatalf("损坏文件应原样保留（不被覆盖）: %q", after)
	}
}

// TestSyncForeshadows_FreshProject 新项目（文件不存在）首次同步正常建档。
func TestSyncForeshadows_FreshProject(t *testing.T) {
	a := newSyncTestAgent(t)
	a.syncForeshadows(1, &AnalysisResult{Foreshadows: []ForeshadowAction{
		{Category: "character", Action: "planted", Description: "主角的旧玉佩"},
	}})

	items := readForeshadowItems(t, a)
	if len(items) != 1 || items[0].Status != types.ForeshadowPlanted || items[0].PlantedIn != "001.md" {
		t.Fatalf("新项目首次同步应登记 1 条 planted: %+v", items)
	}
}
