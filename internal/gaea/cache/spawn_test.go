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
	// 2026-11-23：subagent_writing 随 novel-agents 技能退役，内置模板 5→4。
	if len(templates) != 4 {
		t.Fatalf("expected 4 builtin templates, got %d", len(templates))
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
	if types["subagent_writing"] {
		t.Fatalf("subagent_writing 模板应已随 novel-agents 退役: %v", types)
	}
}
