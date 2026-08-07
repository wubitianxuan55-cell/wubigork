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

func TestBuildPortraitPrompt(t *testing.T) {
	c := characterlib.Character{
		Name:       "苏念",
		Gender:     "female",
		Age:        "23",
		Appearance: "墨发如瀑，眉目如霜",
		Figure:     "身形修长",
		Personality: "清冷",
		Tags:       []string{"剑修", "女主"},
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
