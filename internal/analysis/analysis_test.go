package analysis

import (
	"testing"
)

func TestGenerateStableID_Deterministic(t *testing.T) {
	id1 := GenerateStableID("character", "chapters/001.md", "主角登场")
	id2 := GenerateStableID("character", "chapters/001.md", "主角登场")
	if id1 != id2 {
		t.Fatalf("相同输入应产生相同 ID:\n  id1=%q\n  id2=%q", id1, id2)
	}
}

func TestGenerateStableID_DifferentDescription(t *testing.T) {
	id1 := GenerateStableID("character", "chapters/001.md", "主角登场")
	id2 := GenerateStableID("character", "chapters/001.md", "反派现身")
	if id1 == id2 {
		t.Fatal("不同 description 应产生不同 ID")
	}
}

func TestGenerateStableID_DifferentCategory(t *testing.T) {
	id1 := GenerateStableID("character", "chapters/001.md", "伏笔")
	id2 := GenerateStableID("plot", "chapters/001.md", "伏笔")
	if id1 == id2 {
		t.Fatal("不同 category 应产生不同 ID")
	}
}

func TestGenerateStableID_HasCategoryPrefix(t *testing.T) {
	id := GenerateStableID("world", "chapters/005.md", "")
	if len(id) < 5 {
		t.Fatalf("ID 过短: %q", id)
	}
	// 应以 category_ 开头
	if id[:6] != "world_" {
		t.Fatalf("ID 应以 category 前缀开头: %q", id)
	}
}

func TestGenerateStableID_NoCollisionCloseInputs(t *testing.T) {
	// 非常相似的输入也不应碰撞
	id1 := GenerateStableID("character", "chapters/001.md", "A")
	id2 := GenerateStableID("character", "chapters/001.md", "a")
	if id1 == id2 {
		t.Fatal("大小写不同不应碰撞")
	}
}
