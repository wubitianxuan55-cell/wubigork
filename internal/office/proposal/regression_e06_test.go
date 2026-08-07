package proposal

import "testing"

// E06：招标解析合并/数组解析缺陷——分块合并去重不丢项、评分按名称去重。
func TestRegressionE06MergeParseResultsDedup(t *testing.T) {
	a := parseFileResult{
		Overview:        "项目概况",
		Qualification:   []parseItem{{Name: "资质", Content: "要求A", Quote: "q1"}},
		TechScoring:     []parseScoring{{Name: "技术", MaxScore: "50"}},
		KeyRequirements: []string{"核心要求"},
	}
	b := parseFileResult{
		TotalWords: 120000,
		Qualification: []parseItem{
			{Name: "资质", Content: "要求A", Quote: "q1"}, // 与 a 重复
			{Name: "业绩", Content: "要求B", Quote: "q2"},
		},
		TechScoring:     []parseScoring{{Name: "技术", MaxScore: "60"}, {Name: "价格", MaxScore: "30"}},
		KeyRequirements: []string{"核心要求", "工期要求"},
	}
	got := mergeParseResults(a, b)
	if got.TotalWords != 120000 {
		t.Fatalf("totalWords = %d", got.TotalWords)
	}
	if len(got.Qualification) != 2 {
		t.Fatalf("qualification = %+v", got.Qualification)
	}
	if len(got.TechScoring) != 2 {
		t.Fatalf("techScoring = %+v", got.TechScoring)
	}
	if len(got.KeyRequirements) != 2 {
		t.Fatalf("keyRequirements = %+v", got.KeyRequirements)
	}
}
