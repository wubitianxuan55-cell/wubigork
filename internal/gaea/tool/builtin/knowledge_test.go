package builtin

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestKnowledgeAddAndSearch(t *testing.T) {
	// Use a temp dir as HOME to isolate from real knowledge base.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	ka := knowledgeAdd{}
	result, err := ka.Execute(nil, toJSON(t, map[string]interface{}{
		"title":    "测试知识条目",
		"category": "经验总结",
		"body":     "这是一条测试知识条目，用于验证知识库功能。",
	}))
	if err != nil {
		t.Fatalf("knowledgeAdd failed: %v", err)
	}
	if !strings.Contains(result, "已保存") {
		t.Errorf("expected success message, got: %s", result)
	}

	// Now search for it.
	ks := knowledgeSearch{}
	searchResult, err := ks.Execute(nil, toJSON(t, map[string]interface{}{
		"query": "测试知识条目",
	}))
	if err != nil {
		t.Fatalf("knowledgeSearch failed: %v", err)
	}
	if !strings.Contains(searchResult, "测试知识条目") {
		t.Errorf("expected to find added entry, got: %s", searchResult)
	}
}

func TestKnowledgeSearchOverview(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	// Add one entry first.
	ka := knowledgeAdd{}
	_, err := ka.Execute(nil, toJSON(t, map[string]interface{}{
		"title":    "概览测试",
		"category": "规范标准",
		"body":     "概览测试正文",
	}))
	if err != nil {
		t.Fatal(err)
	}

	// Search with empty query should return overview.
	ks := knowledgeSearch{}
	result, err := ks.Execute(nil, toJSON(t, map[string]interface{}{}))
	if err != nil {
		t.Fatalf("knowledgeSearch overview failed: %v", err)
	}
	if !strings.Contains(result, "知识库概览") {
		t.Errorf("expected 知识库概览, got: %s", result)
	}
	if !strings.Contains(result, "规范标准") {
		t.Errorf("expected 规范标准 category, got: %s", result)
	}
}

func TestKnowledgeAddGeneratesUniqueName(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	ka := knowledgeAdd{}
	// Add the same title twice.
	r1, err := ka.Execute(nil, toJSON(t, map[string]interface{}{
		"title":    "相同标题",
		"category": "其他",
		"body":     "第一次",
	}))
	if err != nil {
		t.Fatal(err)
	}
	r2, err := ka.Execute(nil, toJSON(t, map[string]interface{}{
		"title":    "相同标题",
		"category": "其他",
		"body":     "第二次",
	}))
	if err != nil {
		t.Fatal(err)
	}
	// The file paths should be different.
	if r1 == r2 {
		t.Error("expected different file paths for duplicate titles")
	}
}

func TestKnowledgeSearchByCategory(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())

	ka := knowledgeAdd{}
	ka.Execute(nil, toJSON(t, map[string]interface{}{
		"title": "案例A", "category": "工程案例", "body": "案例A正文",
	}))
	ka.Execute(nil, toJSON(t, map[string]interface{}{
		"title": "规范B", "category": "规范标准", "body": "规范B正文",
	}))

	ks := knowledgeSearch{}
	result, err := ks.Execute(nil, toJSON(t, map[string]interface{}{
		"category": "工程案例",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result, "案例A") {
		t.Errorf("expected 案例A in filtered results, got: %s", result)
	}
	if strings.Contains(result, "规范B") {
		t.Errorf("did not expect 规范B in filtered results")
	}
}

func toJSON(t *testing.T, m map[string]interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return b
}
