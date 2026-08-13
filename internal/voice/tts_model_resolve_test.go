// Package voice — ResolveHerdsmanTTSModel 纯函数测试
package voice

import "testing"

// TestResolveHerdsmanTTSModel_ConfiguredInstalled configured 已装 → 直接返回配置值，不走回退
func TestResolveHerdsmanTTSModel_ConfiguredInstalled(t *testing.T) {
	model, usedFallback, fromInstalled := ResolveHerdsmanTTSModel("qwen3-tts-customvoice", []string{"voxcpm2", "qwen3-tts-customvoice", "edge-tts"})
	if model != "qwen3-tts-customvoice" {
		t.Errorf("model = %q, want qwen3-tts-customvoice", model)
	}
	if usedFallback {
		t.Error("已装命中不应标记回退")
	}
	if fromInstalled {
		t.Error("已装命中不应标记 resolvedFromInstalled")
	}
}

// TestResolveHerdsmanTTSModel_ConfiguredNotInstalledPicksVoxcpm2
// configured 未装（本机实测 qwen3-tts-customvoice 为 0MB）→ 按优先级选已装 voxcpm2
func TestResolveHerdsmanTTSModel_ConfiguredNotInstalledPicksVoxcpm2(t *testing.T) {
	model, usedFallback, fromInstalled := ResolveHerdsmanTTSModel("qwen3-tts-customvoice", []string{"voxcpm2", "edge-tts"})
	if model != "voxcpm2" {
		t.Errorf("model = %q, want voxcpm2", model)
	}
	if !usedFallback {
		t.Error("配置值未安装，应标记回退")
	}
	if !fromInstalled {
		t.Error("结果来自已装列表，应标记 resolvedFromInstalled")
	}
}

// TestResolveHerdsmanTTSModel_AllNotInstalledFallsBackDefault
// 全部未命中（installed 里没有任何 TTS 模型）→ 回退 configured（此处即默认 qwen3-tts-customvoice）
func TestResolveHerdsmanTTSModel_AllNotInstalledFallsBackDefault(t *testing.T) {
	model, usedFallback, fromInstalled := ResolveHerdsmanTTSModel("qwen3-tts-customvoice", []string{"qwen3-8b", "sherpa-onnx-zh-14m"})
	if model != "qwen3-tts-customvoice" {
		t.Errorf("model = %q, want 回退 qwen3-tts-customvoice", model)
	}
	if !usedFallback {
		t.Error("全部未命中应标记回退")
	}
	if fromInstalled {
		t.Error("全部未命中时结果不应来自已装列表")
	}
}

// TestResolveHerdsmanTTSModel_EmptyConfiguredConfigured 空配置 → 用默认值；默认未装则按优先级选已装
func TestResolveHerdsmanTTSModel_EmptyConfigured(t *testing.T) {
	// 默认值本身已装
	model, usedFallback, fromInstalled := ResolveHerdsmanTTSModel("", []string{"qwen3-tts-customvoice"})
	if model != "qwen3-tts-customvoice" || usedFallback || fromInstalled {
		t.Errorf("空配置+默认已装: got (%q,%v,%v), want (qwen3-tts-customvoice,false,false)", model, usedFallback, fromInstalled)
	}
	// 默认值未装 → 优先级选 voxcpm2
	model, usedFallback, fromInstalled = ResolveHerdsmanTTSModel("", []string{"voxcpm2", "edge-tts"})
	if model != "voxcpm2" || !usedFallback || !fromInstalled {
		t.Errorf("空配置+默认未装: got (%q,%v,%v), want (voxcpm2,true,true)", model, usedFallback, fromInstalled)
	}
}

// TestResolveHerdsmanTTSModel_EmptyInstalled 空 installed（拿不到已装列表）→ 等价于原逻辑
func TestResolveHerdsmanTTSModel_EmptyInstalled(t *testing.T) {
	// 空切片
	model, usedFallback, fromInstalled := ResolveHerdsmanTTSModel("qwen3-tts-customvoice", []string{})
	if model != "qwen3-tts-customvoice" || usedFallback || fromInstalled {
		t.Errorf("空列表: got (%q,%v,%v), want (qwen3-tts-customvoice,false,false)", model, usedFallback, fromInstalled)
	}
	// nil 切片
	model, usedFallback, fromInstalled = ResolveHerdsmanTTSModel("qwen3-tts-customvoice", nil)
	if model != "qwen3-tts-customvoice" || usedFallback || fromInstalled {
		t.Errorf("nil 列表: got (%q,%v,%v), want (qwen3-tts-customvoice,false,false)", model, usedFallback, fromInstalled)
	}
	// 空配置 + 空列表 → 默认值
	model, usedFallback, fromInstalled = ResolveHerdsmanTTSModel("", nil)
	if model != "qwen3-tts-customvoice" || usedFallback || fromInstalled {
		t.Errorf("空配置+空列表: got (%q,%v,%v), want (qwen3-tts-customvoice,false,false)", model, usedFallback, fromInstalled)
	}
}

// TestResolveHerdsmanTTSModel_CaseInsensitive 大小写差异匹配
func TestResolveHerdsmanTTSModel_CaseInsensitive(t *testing.T) {
	model, usedFallback, fromInstalled := ResolveHerdsmanTTSModel("voxcpm2", []string{"VoxCPM2", "Edge-TTS"})
	if model != "voxcpm2" {
		t.Errorf("model = %q, want voxcpm2（大小写不敏感命中配置值）", model)
	}
	if usedFallback || fromInstalled {
		t.Errorf("大小写命中配置值不应回退: (%v,%v)", usedFallback, fromInstalled)
	}

	// 优先级匹配也大小写不敏感：configured 未装，installed 里是大写 VoxCPM2
	model, usedFallback, fromInstalled = ResolveHerdsmanTTSModel("qwen3-tts-customvoice", []string{"VOXCPM2"})
	if model != "VOXCPM2" || !usedFallback || !fromInstalled {
		t.Errorf("优先级大小写匹配: got (%q,%v,%v), want (VOXCPM2,true,true)", model, usedFallback, fromInstalled)
	}
}

// TestResolveHerdsmanTTSModel_AffixDifference 前后缀差异匹配（installed 含前缀/后缀变体）
func TestResolveHerdsmanTTSModel_AffixDifference(t *testing.T) {
	// 后缀差异：configured 命中带后缀的已装模型
	model, usedFallback, fromInstalled := ResolveHerdsmanTTSModel("voxcpm2", []string{"voxcpm2-int8"})
	if model != "voxcpm2" || usedFallback || fromInstalled {
		t.Errorf("后缀差异命中配置值: got (%q,%v,%v), want (voxcpm2,false,false)", model, usedFallback, fromInstalled)
	}
	// 前缀差异：configured 未装，优先级命中带前缀的已装模型
	model, usedFallback, fromInstalled = ResolveHerdsmanTTSModel("qwen3-tts-customvoice", []string{"herdsman-voxcpm2"})
	if model != "herdsman-voxcpm2" || !usedFallback || !fromInstalled {
		t.Errorf("前缀差异优先级命中: got (%q,%v,%v), want (herdsman-voxcpm2,true,true)", model, usedFallback, fromInstalled)
	}
	// configured 本身是变体：configured 含 installed → 也算包含匹配
	model, usedFallback, fromInstalled = ResolveHerdsmanTTSModel("qwen3-tts-customvoice-int8", []string{"qwen3-tts-customvoice"})
	if model != "qwen3-tts-customvoice-int8" || usedFallback || fromInstalled {
		t.Errorf("configured 为变体: got (%q,%v,%v), want (qwen3-tts-customvoice-int8,false,false)", model, usedFallback, fromInstalled)
	}
}

// TestResolveHerdsmanTTSModel_PriorityOrder 优先级顺序：voxcpm2 必须第一（本机唯一可用）
func TestResolveHerdsmanTTSModel_PriorityOrder(t *testing.T) {
	// voxcpm2 与 edge-tts 都已装 → 必须选 voxcpm2
	model, _, _ := ResolveHerdsmanTTSModel("qwen3-tts-customvoice", []string{"edge-tts", "voxcpm2"})
	if model != "voxcpm2" {
		t.Errorf("model = %q, want voxcpm2（优先级第一）", model)
	}
	// 仅 cosyvoice 已装（无 voxcpm2）→ 兜底选 cosyvoice
	model, _, _ = ResolveHerdsmanTTSModel("qwen3-tts-customvoice", []string{"cosyvoice"})
	if model != "cosyvoice" {
		t.Errorf("model = %q, want cosyvoice（优先级兜底）", model)
	}
}
