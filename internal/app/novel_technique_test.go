package app

import (
	"strings"
	"testing"
)

// N11 技法检索库（webnovel-writer 九表思想机制重推导蒸馏）测试：
// 账本完整性 / 检索计分确定性 / 注入区段 advisory 定位。

func TestNovelTechniqueLedgerIntegrity(t *testing.T) {
	if len(novelTechniques) < 20 {
		t.Fatalf("技法账本应不少于 20 条，实得 %d", len(novelTechniques))
	}
	seen := map[string]bool{}
	cats := map[string]int{}
	for _, tc := range novelTechniques {
		if tc.ID == "" || seen[tc.ID] {
			t.Fatalf("技法 ID 空或重复: %q", tc.ID)
		}
		seen[tc.ID] = true
		if tc.Category == "" {
			t.Fatalf("技法 %s 缺分类", tc.ID)
		}
		cats[tc.Category]++
		for _, kw := range tc.Keywords {
			if len([]rune(kw)) < 2 {
				t.Fatalf("技法 %s 关键词 %q 短于 2 rune（单字误伤面大，不许入库）", tc.ID, kw)
			}
		}
		if tc.Summary == "" || tc.Detail == "" || tc.Pitfall == "" {
			t.Fatalf("技法 %s 缺摘要/展开/毒点三件套", tc.ID)
		}
	}
	// 九类账本（webnovel-writer 九表思想）齐备
	wantCats := []string{"场景写法", "爽点节奏", "桥段套路", "人设关系", "写作技法", "命名规则", "裁决规则", "金手指", "题材调性"}
	if len(cats) != len(wantCats) {
		t.Fatalf("账本应恰含九类，实得 %d 类: %v", len(cats), cats)
	}
	for _, c := range wantCats {
		if cats[c] == 0 {
			t.Fatalf("缺分类 %s", c)
		}
	}
}

func TestTechniqueMatchScoreDeterministic(t *testing.T) {
	bt := novelTechnique{Keywords: []string{"打脸", "退婚"}, Category: "爽点节奏"}
	if got := techniqueMatchScore(bt, "本章主角被退婚后当众打脸"); got != 2 {
		t.Fatalf("双关键词命中应计 2 分，实得 %d", got)
	}
	if got := techniqueMatchScore(bt, "平淡的一章"); got != 0 {
		t.Fatalf("零命中应计 0 分，实得 %d", got)
	}
	// 分类名参与匹配：按分类名检索应命中该类全部条目
	score := techniqueMatchScore(bt, "爽点节奏")
	if score != 1 {
		t.Fatalf("分类名命中应计 1 分，实得 %d", score)
	}
	// 空文本零分（面板空查询走分类目录，不倒全量）
	if got := techniqueMatchScore(bt, "  "); got != 0 {
		t.Fatalf("空白文本应计 0 分，实得 %d", got)
	}
}

func TestSearchTechniquesOrderAndLimit(t *testing.T) {
	hits := searchTechniques("打脸", 5)
	if len(hits) == 0 {
		t.Fatal("「打脸」应至少命中一条")
	}
	for _, h := range hits {
		if !strings.Contains(strings.Join(h.Keywords, ","), "打脸") && h.Category != "打脸" {
			t.Fatalf("命中条目 %s 不含检索词", h.ID)
		}
	}
	// limit 截断
	if got := searchTechniques("写", 2); len(got) > 2 {
		t.Fatalf("limit=2 应至多返回 2 条，实得 %d", len(got))
	}
	// 空查询返回空
	if got := searchTechniques("", 10); got != nil {
		t.Fatal("空查询应返回空（初始态走分类目录）")
	}
}

func TestTechniqueCategoriesShape(t *testing.T) {
	cats := techniqueCategories()
	if len(cats) != 9 {
		t.Fatalf("分类目录应 9 类，实得 %d", len(cats))
	}
	total := 0
	for _, c := range cats {
		if c["category"].(string) == "" || c["count"].(int) <= 0 {
			t.Fatalf("分类条目形不合法: %v", c)
		}
		total += c["count"].(int)
	}
	if total != len(novelTechniques) {
		t.Fatalf("分类计数合计 %d ≠ 账本 %d 条", total, len(novelTechniques))
	}
}

func TestNovelTechniqueSearchBinding(t *testing.T) {
	a := &writingState{}
	res, err := a.NovelTechniqueSearch("伏笔")
	if err != nil {
		t.Fatalf("检索不应报错: %v", err)
	}
	if res["total"].(int) != len(novelTechniques) {
		t.Fatalf("total 应=账本条数")
	}
	if len(res["results"].([]novelTechnique)) == 0 {
		t.Fatal("「伏笔」应命中写作技法条目")
	}
	// 空查询：results 空 + categories 全量（面板初始态契约）
	empty, err := a.NovelTechniqueSearch("")
	if err != nil {
		t.Fatalf("空查询不应报错: %v", err)
	}
	if len(empty["results"].([]novelTechnique)) != 0 {
		t.Fatal("空查询 results 应为空")
	}
	if len(empty["categories"].([]map[string]interface{})) != 9 {
		t.Fatal("空查询应返回九类目录")
	}
}

func TestBuildTechniqueSection(t *testing.T) {
	// 命中：hint 含关键词 → advisory 区段带标题与毒点
	sec := buildTechniqueSection("本章计划：主角当众打脸退婚对手\n要点：情绪落地")
	if sec == "" {
		t.Fatal("命中关键词应产出技法区段")
	}
	if !strings.Contains(sec, "本章技法提示") || !strings.Contains(sec, "非创作约束") {
		t.Fatalf("区段应明示参考定位: %q", sec)
	}
	if !strings.Contains(sec, "毒点：") {
		t.Fatalf("条目应带毒点: %q", sec)
	}
	// 零命中 → 空串（不占额度，prompt 不出现空区段）
	if got := buildTechniqueSection("平平无奇"); got != "" {
		t.Fatalf("零命中应返回空串，实得 %q", got)
	}
	if got := buildTechniqueSection(""); got != "" {
		t.Fatalf("空 hint 应返回空串，实得 %q", got)
	}
	// 确定性：同输入两次逐字节一致
	if buildTechniqueSection("打脸 误会 伏笔") != buildTechniqueSection("打脸 误会 伏笔") {
		t.Fatal("同输入两次装配应逐字节一致")
	}
	// 预算护栏：至多 3 条
	sec = buildTechniqueSection("打脸 误会 伏笔 反派 对话 升级 取名 修仙 悬疑 金手指 追逐 宴会")
	if n := strings.Count(sec, "\n- "); n > 3 {
		t.Fatalf("注入条目应 ≤3，实得 %d", n)
	}
}
