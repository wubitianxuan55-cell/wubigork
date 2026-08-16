package app

import (
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/characterlib"
)

func TestMergeFill_OnlyFillsEmpty(t *testing.T) {
	cur := map[string]interface{}{
		"name":        "苏念",
		"personality": "清冷寡言，剑心通明",
		"tags":        []interface{}{"剑修", "女主"},
	}
	gen := map[string]interface{}{
		"name":        "另一个名字",
		"role_type":   "protagonist",
		"gender":      "female",
		"appearance":  "墨发如瀑，眉目如霜",
		"personality": "AI 想覆盖但不应生效",
		"tags":        []interface{}{"AI", "覆盖"},
	}
	mergeFill(cur, gen)
	if cur["personality"] != "清冷寡言，剑心通明" {
		t.Fatalf("已有 personality 不应被覆盖: %v", cur["personality"])
	}
	if cur["name"] != "苏念" {
		t.Fatalf("name 不应被修改: %v", cur["name"])
	}
	if cur["roleType"] != "protagonist" {
		t.Fatalf("role_type 应归一化为 roleType: %v", cur["roleType"])
	}
	if cur["appearance"] != "墨发如瀑，眉目如霜" {
		t.Fatalf("空缺 appearance 应被填充: %v", cur["appearance"])
	}
	if len(cur["tags"].([]interface{})) != 2 {
		t.Fatalf("已有 tags 不应被覆盖: %v", cur["tags"])
	}
}

func TestMergeFill_FillsEmptyTags(t *testing.T) {
	cur := map[string]interface{}{"name": "林晚", "tags": []interface{}{}}
	gen := map[string]interface{}{"tags": []interface{}{"剑修", "冷面"}}
	mergeFill(cur, gen)
	if len(cur["tags"].([]interface{})) != 2 {
		t.Fatalf("空 tags 应被填充: %v", cur["tags"])
	}
}

func TestMergeFill_SkipsEmptyGenerated(t *testing.T) {
	cur := map[string]interface{}{"name": "林晚"}
	gen := map[string]interface{}{"gender": "", "age": "  "}
	mergeFill(cur, gen)
	if _, ok := cur["gender"]; ok {
		t.Fatalf("空生成值不应写入: %v", cur)
	}
	if _, ok := cur["age"]; ok {
		t.Fatalf("空白生成值不应写入: %v", cur)
	}
}

func TestMergeFill_FillsDefaultDims(t *testing.T) {
	cur := map[string]interface{}{
		"name": "苏念",
		"dims": map[string]interface{}{"T": 50.0, "I": 50.0, "S": 50.0, "O": 50.0, "R": 50.0},
	}
	gen := map[string]interface{}{
		"dims": map[string]interface{}{"T": 85.0, "I": 40.0, "S": 20.0, "O": 70.0, "R": 60.0},
	}
	mergeFill(cur, gen)
	d, ok := cur["dims"].(map[string]interface{})
	if !ok || d["T"] != 85.0 || d["I"] != 40.0 {
		t.Fatalf("默认五维人格应被补齐: %v", cur["dims"])
	}
}

func TestMergeFill_KeepsCustomDims(t *testing.T) {
	cur := map[string]interface{}{
		"name": "苏念",
		"dims": map[string]interface{}{"T": 30.0, "I": 80.0, "S": 70.0, "O": 40.0, "R": 50.0},
	}
	gen := map[string]interface{}{
		"dims": map[string]interface{}{"T": 99.0, "I": 10.0, "S": 10.0, "O": 10.0, "R": 10.0},
	}
	mergeFill(cur, gen)
	d, _ := cur["dims"].(map[string]interface{})
	if d["T"] != 30.0 {
		t.Fatalf("已有五维人格不应被补齐覆盖: %v", cur["dims"])
	}
}

func TestMergeRandom_DimsTarget(t *testing.T) {
	cur := map[string]interface{}{
		"name":        "苏念",
		"personality": "清冷剑修，寡言重诺",
		"dims":        map[string]interface{}{"T": 50.0, "I": 50.0, "S": 50.0, "O": 50.0, "R": 50.0},
	}
	gen := map[string]interface{}{
		"dims": map[string]interface{}{"T": 20.0, "I": 30.0, "S": 40.0, "O": 50.0, "R": 80.0},
	}
	mergeRandom(cur, gen, []string{"dims"})
	d, _ := cur["dims"].(map[string]interface{})
	if d["R"] != 80.0 || d["T"] != 20.0 {
		t.Fatalf("dims 目标字段应被随机覆盖: %v", cur["dims"])
	}
	if cur["personality"] != "清冷剑修，寡言重诺" {
		t.Fatalf("非目标字段不应被修改: %v", cur["personality"])
	}
}

func TestParseDims_StringForms(t *testing.T) {
	for _, s := range []string{
		"85/40/20/70/60",
		"85,40,20,70,60",
		"T=85,I=40,S=20,O=70,R=60",
		"t=85 i=40 s=20 o=70 r=60",
	} {
		d, ok := parseDims(s)
		if !ok {
			t.Fatalf("parseDims(%q) 解析失败", s)
		}
		if d["T"] != 85.0 || d["R"] != 60.0 {
			t.Fatalf("parseDims(%q) = %v", s, d)
		}
	}
	if _, ok := parseDims("85/40/20/70"); ok {
		t.Fatalf("缺一个维度时应解析失败")
	}
}

func TestParseRandomFields_IncludesDims(t *testing.T) {
	got := parseRandomFields("dims")
	if len(got) != 1 || got[0] != "dims" {
		t.Fatalf("parseRandomFields(dims) = %v", got)
	}
	all := parseRandomFields("all")
	if !containsKey(all, "dims") {
		t.Fatalf("all 应包含 dims: %v", all)
	}
}

func TestBuildPortraitPrompt(t *testing.T) {
	c := characterlib.Character{
		Name:        "苏念",
		Gender:      "female",
		Age:         "23",
		Appearance:  "墨发如瀑，眉目如霜",
		Figure:      "身形修长",
		Personality: "清冷",
		Tags:        []string{"剑修", "女主"},
	}
	p := buildPortraitPrompt(c)
	for _, want := range []string{"苏念", "female", "年龄23", "墨发如瀑", "身形修长", "清冷", "剑修", "半身像"} {
		if !strings.Contains(p, want) {
			t.Fatalf("prompt 缺少 %q: %s", want, p)
		}
	}
}

func TestBuildPortraitPrompt_SkipsEmpty(t *testing.T) {
	p := buildPortraitPrompt(characterlib.Character{Name: "无名"})
	if strings.Contains(p, "外貌") || strings.Contains(p, "气质") || strings.Contains(p, "特征标签") {
		t.Fatalf("空字段不应出现在 prompt: %s", p)
	}
	if !strings.Contains(p, "无名") {
		t.Fatalf("prompt 应包含角色名: %s", p)
	}
}

func TestSummarizeExisting(t *testing.T) {
	cur := map[string]interface{}{
		"name":        "苏念",
		"personality": "清冷",
		"tags":        []interface{}{"剑修"},
	}
	s := summarizeExisting(cur)
	if !strings.Contains(s, "性格: 清冷") || !strings.Contains(s, "标签: 剑修") {
		t.Fatalf("summarizeExisting 输出不完整: %s", s)
	}
}
