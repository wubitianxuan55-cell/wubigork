package cache

import (
	"testing"
)

func TestRegisterAndLookupSpawnTemplate(t *testing.T) {
	// 确保测试间隔离
	spawnTemplates.items = nil

	st := SpawnTemplate{Kind: "test_kind", Prefix: "test prefix", Description: "test"}
	RegisterSpawnTemplate(st)

	got, ok := LookupSpawnTemplate("test_kind")
	if !ok {
		t.Fatal("LookupSpawnTemplate should find registered template")
	}
	if got.Prefix != "test prefix" {
		t.Fatalf("wrong prefix: got %q, want %q", got.Prefix, "test prefix")
	}
	if got.Description != "test" {
		t.Fatalf("wrong description: got %q, want %q", got.Description, "test")
	}

	// 未注册的 kind
	_, ok = LookupSpawnTemplate("nonexistent")
	if ok {
		t.Fatal("LookupSpawnTemplate should not find unregistered kind")
	}
}

func TestRegisterSpawnTemplate_EmptyKindOrPrefix(t *testing.T) {
	spawnTemplates.items = nil
	RegisterSpawnTemplate(SpawnTemplate{Kind: "", Prefix: "x", Description: ""})
	RegisterSpawnTemplate(SpawnTemplate{Kind: "k", Prefix: "", Description: ""})
	if len(spawnTemplates.items) != 0 {
		t.Fatal("should not register empty kind or prefix")
	}
}

func TestBuiltinSpawnTemplates(t *testing.T) {
	templates := BuiltinSpawnTemplates()
	if len(templates) != 5 {
		t.Fatalf("expected 5 builtin templates, got %d", len(templates))
	}
	types := make(map[string]bool)
	for _, st := range templates {
		if st.Prefix == "" {
			t.Fatalf("template %q has empty prefix", st.Kind)
		}
		if types[string(st.Kind)] {
			t.Fatalf("duplicate template kind: %s", st.Kind)
		}
		types[string(st.Kind)] = true
	}
	// v4.336：写作域角色卡模板（novel-agents 七角色子代理共享前缀缓存）
	if !types["subagent_writing"] {
		t.Fatalf("missing subagent_writing template: %v", types)
	}
}
