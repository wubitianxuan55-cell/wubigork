package modelengine

import "testing"

// TestRemoveModel_RetiresAndClearsDefault 目录退役清理：RemoveModel 移除存量
// 模型条目；默认模型指向被删模型时一并清空（防悬空默认）；幂等 no-op 不落盘。
func TestRemoveModel_RetiresAndClearsDefault(t *testing.T) {
	m := NewManager("", "")
	m.SaveEngine(EngineConfig{
		ID:           "xai",
		Enabled:      true,
		DefaultModel: "grok-old-tts",
		Models: []ModelInfo{
			{ID: "grok-old-tts"},
			{ID: "grok-tts"},
		},
	})

	m.RemoveModel("xai", "grok-old-tts")

	e, ok := m.GetEngine("xai")
	if !ok {
		t.Fatal("引擎不应被删除")
	}
	for _, mdl := range e.Models {
		if mdl.ID == "grok-old-tts" {
			t.Error("存量模型应被移除")
		}
	}
	if len(e.Models) != 1 || e.Models[0].ID != "grok-tts" {
		t.Errorf("剩余模型 = %v, want 只剩 grok-tts", e.Models)
	}
	if e.DefaultModel == "grok-old-tts" {
		t.Error("默认模型指向被删模型时应被清空")
	}

	// 幂等：再删一次不 panic 不误删其它模型
	m.RemoveModel("xai", "grok-old-tts")
	e, _ = m.GetEngine("xai")
	if len(e.Models) != 1 {
		t.Errorf("重复删除后模型数 = %d, want 1", len(e.Models))
	}
	// 不存在的引擎/模型：静默 no-op
	m.RemoveModel("no-such", "x")
	m.RemoveModel("xai", "not-in-list")
}

// TestRemoveModel_KeepsOtherDefault 默认模型不指向被删模型时保持不动。
func TestRemoveModel_KeepsOtherDefault(t *testing.T) {
	m := NewManager("", "")
	m.SaveEngine(EngineConfig{
		ID:           "xai",
		Enabled:      true,
		DefaultModel: "grok-tts",
		Models: []ModelInfo{
			{ID: "grok-old-tts"},
			{ID: "grok-tts"},
		},
	})
	m.RemoveModel("xai", "grok-old-tts")
	e, _ := m.GetEngine("xai")
	if e.DefaultModel != "grok-tts" {
		t.Errorf("DefaultModel = %q, want 保持 grok-tts", e.DefaultModel)
	}
}
