package app

// v4.126 刀1 本地模型使用：办公切换器从引擎粒度到模型粒度。
// GaeaModels 按引擎展开全部对话模型（本地引擎置前、非 llm 过滤、空清单回退
// "(默认)"）；GaeaSetModel 支持 "engine/model" 精确设默认模型。全部内存构造，
// 配置写入临时 USERPROFILE/HOME（SetActiveEngine 落盘走 config.Save）。

import (
	"testing"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

// modelsTestApp 构造带 client/engineMgr 的最小 App（配置落盘重定向临时目录）。
func modelsTestApp(t *testing.T) *App {
	t.Helper()
	home := t.TempDir()
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)
	cfg := &config.Config{}
	return &App{core: &core{cfg: cfg, engineMgr: modelengine.NewManager("", ""), client: ai.NewClient(cfg)}}
}

// TestGaeaModelsExpandsModels 按引擎展开对话模型：本地引擎置前、非 llm 过滤、
// 空清单回退 "(默认)"、Current=活跃引擎且为引擎默认模型。
func TestGaeaModelsExpandsModels(t *testing.T) {
	a := modelsTestApp(t)
	mgr := a.engineMgr
	// 先禁用全部预置引擎，再启用本用例关心的四个，保证精确断言序列。
	for _, eng := range mgr.GetEngines() {
		eng.Enabled = false
		if err := mgr.SaveEngine(eng); err != nil {
			t.Fatalf("SaveEngine disable %s: %v", eng.ID, err)
		}
	}
	setEngines := func() {
		// ollama（本地）：llm+tts+embedding，默认 q3 —— tts/embedding 不进切换器。
		if err := mgr.SaveEngine(modelengine.EngineConfig{
			ID: "ollama", Enabled: true, DefaultModel: "q3",
			Models: []modelengine.ModelInfo{
				{ID: "q3", Kind: "llm"},
				{ID: "q3-turbo", Kind: "image"}, // turbo 关键词判生图，不进聊天切换器
				{ID: "tts-x", Kind: "tts"},
				{ID: "bge-m3", Kind: "embedding"},
			},
		}); err != nil {
			t.Fatalf("SaveEngine ollama: %v", err)
		}
		// modelhub（本地）：模型带加载态与展示名。
		if err := mgr.SaveEngine(modelengine.EngineConfig{
			ID: "modelhub", Enabled: true, DefaultModel: "ollama-manifest:tiny",
			Models: []modelengine.ModelInfo{
				{ID: "ollama-manifest:tiny", Status: "running", Name: "Tinyrick Q6", Kind: "llm"},
				{ID: "ollama-manifest:big", Status: "stopped", Name: "Big UD", Kind: "llm"},
			},
		}); err != nil {
			t.Fatalf("SaveEngine modelhub: %v", err)
		}
		// xai（云端）：清单含 grok-tts 补充项，只应出现 llm 条目。
		if err := mgr.SaveEngine(modelengine.EngineConfig{
			ID: "xai", Enabled: true, DefaultModel: "grok-4.20",
			Models: []modelengine.ModelInfo{{ID: "grok-4.20", Kind: "llm"}, {ID: "grok-tts", Kind: "tts"}},
		}); err != nil {
			t.Fatalf("SaveEngine xai: %v", err)
		}
		// deepseek（云端）：空清单 → "(默认)" 回退。
		if err := mgr.SaveEngine(modelengine.EngineConfig{ID: "deepseek", Enabled: true}); err != nil {
			t.Fatalf("SaveEngine deepseek: %v", err)
		}
	}
	setEngines()

	a.client.SetActiveEngine("modelhub")
	got := a.GaeaModels()

	// 期望序列：本地引擎（ollama→modelhub 注册序）在前、云端在后。
	type want struct {
		ref     string
		current bool
	}
	wants := []want{
		{"ollama/q3", false},
		{"modelhub/ollama-manifest:tiny", true}, // 活跃引擎默认模型
		{"modelhub/ollama-manifest:big", false},
		{"xai/grok-4.20", false},
		{"deepseek/(默认)", false},
	}
	if len(got) != len(wants) {
		t.Fatalf("GaeaModels 长度 = %d, want %d：%+v", len(got), len(wants), got)
	}
	for i, w := range wants {
		if got[i].Ref != w.ref {
			t.Errorf("[%d] ref = %q, want %q", i, got[i].Ref, w.ref)
		}
		if got[i].Current != w.current {
			t.Errorf("[%d] %s current = %v, want %v", i, w.ref, got[i].Current, w.current)
		}
	}
	// 本地/加载态/展示名透传。
	if !got[0].Local {
		t.Error("ollama 条目应标记 Local")
	}
	if got[1].Status != "running" || got[1].Label != "Tinyrick Q6" {
		t.Errorf("modelhub 条目 status/label = %q/%q, want running/Tinyrick Q6", got[1].Status, got[1].Label)
	}
	if got[4].Model != "(默认)" {
		t.Errorf("空清单引擎应回退 (默认)，got %q", got[4].Model)
	}
}

// TestGaeaSetModelEngineModel ref 带 model 时先设引擎默认模型再切引擎；
// "(默认)" 占位只切引擎；模型不在清单如实报错不静默降级。
func TestGaeaSetModelEngineModel(t *testing.T) {
	a := modelsTestApp(t)
	mgr := a.engineMgr
	if err := mgr.SaveEngine(modelengine.EngineConfig{
		ID: "ollama", Enabled: true, DefaultModel: "q3",
		Models: []modelengine.ModelInfo{{ID: "q3", Kind: "llm"}, {ID: "q7", Kind: "llm"}},
	}); err != nil {
		t.Fatalf("SaveEngine: %v", err)
	}

	// 精确切换：默认模型与活跃引擎同时生效。
	if err := a.GaeaSetModel("ollama/q7"); err != nil {
		t.Fatalf("GaeaSetModel: %v", err)
	}
	if m, _ := mgr.GetDefaultModel("ollama"); m != "q7" {
		t.Errorf("默认模型 = %q, want q7", m)
	}
	if eng := a.GetActiveEngine(); eng != "ollama" {
		t.Errorf("活跃引擎 = %q, want ollama", eng)
	}

	// "(默认)" 占位：只切引擎，不动默认模型。
	if err := mgr.SaveEngine(modelengine.EngineConfig{ID: "deepseek", Enabled: true, DefaultModel: "ds-v4"}); err != nil {
		t.Fatalf("SaveEngine deepseek: %v", err)
	}
	if err := a.GaeaSetModel("deepseek/(默认)"); err != nil {
		t.Fatalf("GaeaSetModel (默认): %v", err)
	}
	if m, _ := mgr.GetDefaultModel("ollama"); m != "q7" {
		t.Errorf("(默认) 不应改 ollama 默认模型，got %q", m)
	}
	if eng := a.GetActiveEngine(); eng != "deepseek" {
		t.Errorf("活跃引擎 = %q, want deepseek", eng)
	}

	// 模型不在清单：报错且引擎不切换（fail-closed，防悄悄回落默认）。
	if err := a.GaeaSetModel("ollama/ghost"); err == nil {
		t.Error("清单外模型应报错")
	}
	if eng := a.GetActiveEngine(); eng != "deepseek" {
		t.Errorf("失败切换不应改变活跃引擎，got %q", eng)
	}
}
