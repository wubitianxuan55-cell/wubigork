package promptstore

// t6-C2 模板包导入导出表驱动测试（规格 进度计划/gaea-prompt-bundle-t6c2-20260916.md §6）。
// 矩阵：三态四分支 + gaea 三闸（invalid/unknown/duplicate）+ 停用覆盖行导出
// 保真 + BuildBundle 统计 + SameTemplate 稳定性 + ContentHash 口径。

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/prompt"
)

func tpl(name, system string) prompt.Template {
	return prompt.Template{Name: name, System: system, Task: "写正文"}
}

// mapFixture 固定基线表闭包（nil 值=键不存在）。
func mapFixture(m map[string]prompt.Template) func(string) *prompt.Template {
	return func(k string) *prompt.Template {
		if t, ok := m[k]; ok {
			cp := t
			return &cp
		}
		return nil
	}
}

func setFixture(keys ...string) func(string) bool {
	set := make(map[string]bool, len(keys))
	for _, k := range keys {
		set[k] = true
	}
	return func(k string) bool { return set[k] }
}

func TestBundle_ContentHash(t *testing.T) {
	if got := ContentHash("  hello  "); got != ContentHash("hello") {
		t.Fatalf("哈希应忽略首尾空白: %s vs %s", got, ContentHash("hello"))
	}
	if len(ContentHash("hello")) != 16 {
		t.Fatalf("哈希长度=16（MuMu 同款截断），得到 %d", len(ContentHash("hello")))
	}
	if ContentHash("a") == ContentHash("b") {
		t.Fatal("不同内容哈希碰撞（sha256 前 16 hex 不可能）")
	}
}

func TestBundle_SameTemplate(t *testing.T) {
	a := tpl("k", "同一句")
	b := tpl("k", "同一句")
	if !SameTemplate(a, b) {
		t.Fatal("同构模板应判定相同")
	}
	// Inputs 是 map：键序不影响 canonical 序列化。
	a2 := prompt.Template{Name: "k", System: "s", Task: "t", Inputs: map[string]prompt.InputDef{
		"zeta":  {Priority: "P0"},
		"alpha": {Priority: "P1"},
	}}
	b2 := prompt.Template{Name: "k", System: "s", Task: "t", Inputs: map[string]prompt.InputDef{
		"alpha": {Priority: "P1"},
		"zeta":  {Priority: "P0"},
	}}
	if !SameTemplate(a2, b2) {
		t.Fatal("map 键序不同的同构模板应判定相同（canonical JSON 稳定）")
	}
	if SameTemplate(a, tpl("k", "另一句")) {
		t.Fatal("正文不同应判定不同")
	}
}

func TestBundle_BuildBundle(t *testing.T) {
	base := map[string]prompt.Template{
		"chapter-summary": tpl("chapter-summary", "基线S"),
		"create-chapter":  tpl("create-chapter", "基线C"),
	}
	entries := []Override{
		{Key: "create-chapter", Content: tpl("create-chapter", "覆盖C"), IsActive: false, Version: 2}, // 停用行也要保真
	}
	b := BuildBundle([]string{"chapter-summary", "create-chapter"}, entries, mapFixture(base), 1234)
	if b.Version != 1 || b.ExportedAt != 1234 {
		t.Fatalf("包头不对: %+v", b)
	}
	if b.Statistics.Total != 2 || b.Statistics.SystemDefault != 1 || b.Statistics.Customized != 1 {
		t.Fatalf("统计不对: %+v", b.Statistics)
	}
	sum := b.Templates[0]
	if sum.IsCustomized || sum.Content.System != "基线S" || sum.SystemContentHash == "" {
		t.Fatalf("内置行应取基线内容+哈希: %+v", sum)
	}
	cre := b.Templates[1]
	if !cre.IsCustomized || cre.Content.System != "覆盖C" || cre.IsActive || cre.Version != 2 {
		t.Fatalf("覆盖行应保真（含停用态与版本快照）: %+v", cre)
	}
}

// 三态矩阵：MuMu §6.4 四行 + gaea 三闸，逐分支钉死（规格 §3 ImportBundle 表）。
func TestBundle_ImportThreeState(t *testing.T) {
	base := map[string]prompt.Template{
		"k-same":   tpl("k-same", "与包一致"),
		"k-drift":  tpl("k-drift", "升级后的基线"),
		"k-custom": tpl("k-custom", "基线"),
	}
	engineHas := setFixture("k-same", "k-drift", "k-custom", "k-custom2", "k-bad")

	t.Run("false+同基线→删覆盖行回落内置", func(t *testing.T) {
		entries := []Override{{Key: "k-same", Content: tpl("k-same", "旧覆盖"), IsActive: true, Version: 5}}
		merged, res := ImportBundle(ExportBundle{Templates: []BundleTemplate{
			{Key: "k-same", Content: tpl("k-same", "与包一致"), IsActive: true},
		}}, entries, engineHas, mapFixture(base), 99)
		if len(merged) != 0 {
			t.Fatalf("同基线行应删掉本地覆盖: %+v", merged)
		}
		if !res.Applied || res.Statistics.KeptSystemDefault != 1 {
			t.Fatalf("统计不对: %+v", res.Statistics)
		}
		if res.Outcomes[0].Action != ActionKeptSystemDefault {
			t.Fatalf("action 不对: %+v", res.Outcomes[0])
		}
	})

	t.Run("false+同基线且本地无覆盖行→幂等不落盘", func(t *testing.T) {
		_, res := ImportBundle(ExportBundle{Templates: []BundleTemplate{
			{Key: "k-same", Content: tpl("k-same", "与包一致"), IsActive: true},
		}}, nil, engineHas, mapFixture(base), 99)
		if res.Applied {
			t.Fatal("无变更不应标记 Applied")
		}
		if res.Statistics.KeptSystemDefault != 1 {
			t.Fatalf("统计不对: %+v", res.Statistics)
		}
	})

	t.Run("false+基线不同→转自定义（内置升级对账）", func(t *testing.T) {
		merged, res := ImportBundle(ExportBundle{Templates: []BundleTemplate{
			{Key: "k-drift", Content: tpl("k-drift", "旧版内置快照"), IsActive: true},
		}}, nil, engineHas, mapFixture(base), 99)
		if len(merged) != 1 || merged[0].Key != "k-drift" || merged[0].Content.System != "旧版内置快照" {
			t.Fatalf("应转自定义行: %+v", merged)
		}
		if merged[0].Version != 1 || merged[0].CreatedAt != 99 {
			t.Fatalf("版本/时间应由 Upsert 裁定: %+v", merged[0])
		}
		if !res.Applied || res.Statistics.ConvertedToCustom != 1 {
			t.Fatalf("统计不对: %+v", res.Statistics)
		}
	})

	t.Run("true→直接写行（含停用态如实）", func(t *testing.T) {
		merged, res := ImportBundle(ExportBundle{Templates: []BundleTemplate{
			{Key: "k-custom", Content: tpl("k-custom", "包里的自定义"), IsActive: false, IsCustomized: true, Version: 9},
		}}, nil, engineHas, mapFixture(base), 99)
		if len(merged) != 1 || merged[0].IsActive {
			t.Fatalf("应如实写入停用态: %+v", merged)
		}
		if merged[0].Version != 1 {
			t.Fatalf("包内 version=9 不可信（防倒退），本地首写应=1: %+v", merged[0])
		}
		if !res.Applied || res.Statistics.CreatedOrUpdate != 1 {
			t.Fatalf("统计不对: %+v", res.Statistics)
		}
	})

	t.Run("未知键不建行", func(t *testing.T) {
		merged, res := ImportBundle(ExportBundle{Templates: []BundleTemplate{
			{Key: "k-ghost", Content: tpl("k-ghost", "x"), IsCustomized: true},
		}}, nil, engineHas, mapFixture(base), 99)
		if len(merged) != 0 {
			t.Fatalf("未知键不应建行: %+v", merged)
		}
		if res.Statistics.SkippedUnknown != 1 || res.Applied {
			t.Fatalf("统计不对: %+v", res.Statistics)
		}
	})

	t.Run("error 级校验跳行不阻断整包", func(t *testing.T) {
		bad := prompt.Template{Name: "k-bad", System: "  ", Task: "写正文"} // empty-system → error
		good := tpl("k-custom", "好行")
		merged, res := ImportBundle(ExportBundle{Templates: []BundleTemplate{
			{Key: "k-bad", Content: bad, IsCustomized: true},
			{Key: "k-custom", Content: good, IsCustomized: true},
		}}, nil, engineHas, mapFixture(base), 99)
		if len(merged) != 1 || merged[0].Key != "k-custom" {
			t.Fatalf("坏行跳过、好行照进: %+v", merged)
		}
		if res.Statistics.SkippedInvalid != 1 || res.Statistics.CreatedOrUpdate != 1 || !res.Applied {
			t.Fatalf("统计不对: %+v", res.Statistics)
		}
		if !strings.Contains(res.Outcomes[0].Reason, "校验未通过") {
			t.Fatalf("跳行 reason 应带校验摘要: %+v", res.Outcomes[0])
		}
	})

	t.Run("包内重复键首见生效", func(t *testing.T) {
		merged, res := ImportBundle(ExportBundle{Templates: []BundleTemplate{
			{Key: "k-custom", Content: tpl("k-custom", "第一份"), IsCustomized: true},
			{Key: "k-custom", Content: tpl("k-custom", "第二份"), IsCustomized: true},
		}}, nil, engineHas, mapFixture(base), 99)
		if len(merged) != 1 || merged[0].Content.System != "第一份" {
			t.Fatalf("首见应生效: %+v", merged)
		}
		if res.Statistics.SkippedDuplicate != 1 {
			t.Fatalf("统计不对: %+v", res.Statistics)
		}
	})

	t.Run("空键跳过", func(t *testing.T) {
		merged, res := ImportBundle(ExportBundle{Templates: []BundleTemplate{
			{Key: "  ", Content: tpl("", "x"), IsCustomized: true},
		}}, nil, engineHas, mapFixture(base), 99)
		if len(merged) != 0 || res.Statistics.SkippedInvalid != 1 {
			t.Fatalf("空键应跳: %+v / %+v", merged, res.Statistics)
		}
	})

	t.Run("入参覆盖表不被改写+结果恒非 nil", func(t *testing.T) {
		entries := []Override{{Key: "k-custom", Content: tpl("k-custom", "本地旧"), IsActive: true, Version: 3, CreatedAt: 1}}
		_, res := ImportBundle(ExportBundle{Templates: []BundleTemplate{}}, entries, engineHas, mapFixture(base), 99)
		if entries[0].Content.System != "本地旧" || entries[0].Version != 3 {
			t.Fatalf("入参切片不应被改写: %+v", entries[0])
		}
		if res.Outcomes == nil || res.Applied {
			t.Fatalf("空包: outcomes 恒非 nil / Applied=false: %+v", res)
		}
		if res.Statistics.Total != 0 {
			t.Fatalf("空包 Total=0: %+v", res.Statistics)
		}
	})
}

// 往返回归：BuildBundle 导出 → 清空状态 → ImportBundle 导入 → 覆盖层回魂
// （线 B round-trip 的纯函数内核）。
func TestBundle_RoundTrip(t *testing.T) {
	base := map[string]prompt.Template{
		"a": tpl("a", "基线A"),
		"b": tpl("b", "基线B"),
	}
	entries := []Override{
		{Key: "b", Category: "chapter", Description: "改过的", Content: tpl("b", "覆盖B"), IsActive: true, Version: 4, CreatedAt: 11, UpdatedAt: 22},
	}
	bundle := BuildBundle([]string{"a", "b"}, entries, mapFixture(base), 1000)
	raw, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	var back ExportBundle
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	merged, res := ImportBundle(back, nil, setFixture("a", "b"), mapFixture(base), 2000)
	if len(merged) != 1 || merged[0].Key != "b" || merged[0].Content.System != "覆盖B" || !merged[0].IsActive {
		t.Fatalf("覆盖行应回魂: %+v", merged)
	}
	if merged[0].Category != "chapter" || merged[0].Description != "改过的" {
		t.Fatalf("元数据应随行带回: %+v", merged[0])
	}
	if res.Statistics.KeptSystemDefault != 1 || res.Statistics.CreatedOrUpdate != 1 || !res.Applied {
		t.Fatalf("a 行同基线回落内置、b 行写回覆盖: %+v", res.Statistics)
	}
}
