package app

// 长篇刀6测试：上下文编译（setting 预算裁剪 / 文风合并去重 / 世界观相关性 /
// Inventory 清单）。规格 进度计划/gaea-novel-ctx-compiler-20260930.md。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/types"
)

// TestJoinStyleSections 文风合并矩阵：双空零注入 / 单 style / 单 digest /
// 双有单标题（去重验收：只出现一个文风标题）。
func TestJoinStyleSections(t *testing.T) {
	if got := joinStyleSections("", ""); got != "" {
		t.Fatalf("双空应零注入：%q", got)
	}
	onlyStyle := joinStyleSections("短句为主。", "")
	if !strings.Contains(onlyStyle, "短句为主") || strings.Contains(onlyStyle, "成稿学习") {
		t.Fatalf("单 style 应只有偏好：%q", onlyStyle)
	}
	onlyDigest := joinStyleSections("", "- 四字格克制")
	if !strings.Contains(onlyDigest, "四字格克制") || strings.Contains(onlyDigest, "显式偏好") {
		t.Fatalf("单 digest 应只有学习指令：%q", onlyDigest)
	}
	both := joinStyleSections("短句为主。", "- 四字格克制")
	for _, want := range []string{"文风与表达", "显式偏好", "成稿学习", "短句为主", "四字格克制"} {
		if !strings.Contains(both, want) {
			t.Fatalf("合并区段应含 %q：%q", want, both)
		}
	}
	// 去重验收：标题唯一（「## 文风与表达」只出现一次）
	if n := strings.Count(both, "## 文风与表达"); n != 1 {
		t.Fatalf("合并后应只有一个文风标题，得到 %d 处：%q", n, both)
	}
}

// TestWorldviewRelevance 相关性排序：hint 命中的维度条目前移（稳定），零命中原序。
func TestWorldviewRelevance(t *testing.T) {
	pm := newContextTestProject(t)
	// 直接写结构化 sections（ToMarkdown 输出 "## 标题"——绕开 md 迁移的 legacy 包装）
	if err := pm.WriteWorldviewFile(&types.WorldviewFile{Sections: []types.WorldviewSection{
		{ID: "era", Title: "时代背景", Content: "架空中世纪，城邦制。", Order: 1},
		{ID: "power", Title: "修炼体系", Content: "灵气分九阶。", Order: 2},
		{ID: "factions", Title: "势力格局", Content: "三门两阁相争。", Order: 3},
	}}); err != nil {
		t.Fatal(err)
	}
	// 零命中：原序（时代背景在前）
	got0 := buildWorldviewSection(pm, "")
	if strings.Index(got0, "时代背景") > strings.Index(got0, "修炼体系") {
		t.Fatalf("零命中应保持原序：%s", got0)
	}
	// 命中「修炼体系」（hint 含该词）：修炼条目前移
	got1 := buildWorldviewSection(pm, "本章计划：主角突破修炼体系第二阶")
	if strings.Index(got1, "修炼体系") > strings.Index(got1, "时代背景") {
		t.Fatalf("命中维度应前移：%s", got1)
	}
	// 前移≠丢弃：未命中维度仍在
	if !strings.Contains(got1, "时代背景") || !strings.Contains(got1, "势力格局") {
		t.Fatalf("预算内未命中维度不得丢弃：%s", got1)
	}
}

// TestSettingBudgetTrim setting 槽入预算：超长设定截断+提示（HTTP body 断言）。
func TestSettingBudgetTrim(t *testing.T) {
	a, pm, requests := newChapterGateLLMAppReply(t, "正文。")
	a.setPM(pm)
	if err := pm.WriteOutlines(&types.OutlineFile{Nodes: []types.OutlineNode{
		{ID: "n1", OrderIndex: 1, Title: "第一章", Summary: "开头", KeyPoints: []string{"引入"}},
	}}); err != nil {
		t.Fatal(err)
	}
	longSetting := strings.Repeat("世界观设定条目，用来撑长度。", 400) // >> 3000 rune
	if _, err := a.CreateChapterWithOverride(longSetting, "", "写第一章", 1, "", "", 10, 0.8, true); err != nil {
		t.Fatalf("CreateChapter: %v", err)
	}
	// Windows unlinkat 竞态（在册）：生成协程尾步（自动门/摘要/SceneRefs）仍在写盘时
	// TempDir 清理会撞 directory not empty——等登记表与协程组清空再返回。
	waitGensDone(t, a)
	select {
	case body := <-requests:
		if !strings.Contains(string(body), "设定过长已截断") {
			t.Fatal("超长 setting 应被截断并附提示（预算外堆料收口）")
		}
		if !strings.Contains(string(body), "世界观设定条目") {
			t.Fatal("截断版仍应保留设定内容前段")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("未捕获生成请求")
	}
}

// TestNovelContextInventory 清单：含各功能区段与合计行；与实际注入同源。
func TestNovelContextInventory(t *testing.T) {
	a := newGateEmptyApp()
	pm := newContextTestProject(t)
	a.setPM(pm)
	_ = os.Remove(filepath.Join(pm.Dir, "worldview.json"))
	if err := os.WriteFile(filepath.Join(pm.Dir, "worldview.md"), []byte("## 时代背景\n架空中世纪。"), 0644); err != nil {
		t.Fatal(err)
	}
	items, err := a.NovelContextInventory(1)
	if err != nil {
		t.Fatalf("清单: %v", err)
	}
	names := map[string]bool{}
	for _, it := range items {
		if n, ok := it["name"].(string); ok {
			names[n] = true
		}
	}
	for _, want := range []string{"setting（小说设定）", "世界观要点", "角色摘要", "合计（不含 setting 槽）"} {
		if !names[want] {
			t.Fatalf("清单应含 %q：%v", want, names)
		}
	}
}
