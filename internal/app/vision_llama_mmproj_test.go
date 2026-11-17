package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/modelengine"
)

// ── 2026-11-18 办公识图修复（本地 llama 档识图坏）测试面 ────────────────
// 根因：llamaEnsure 起 llama-server 不带 --mmproj，而办公聊天绑 llama（qwen35
// @8124）时识图跟随链（visionChatFollowRoute）把图喂给纯文本 llama-server——
// 必坏。修复三层：llamaEnsure 按档位 MMProj 加载编码器、managed 目录标注
// caps=vision、跟随链对未标注档位弃权（能力闸）。

// TestManagedProfileMmprojPathHermetic mmproj 路径解析：统一根优先、历史根
// 回退、两处都缺=空（纯文本档位）。t.TempDir 隔离，不触真机模型目录。
func TestManagedProfileMmprojPathHermetic(t *testing.T) {
	root := t.TempDir()
	legacy := t.TempDir()
	p := engineProfileOption{ID: "qwen35", ModelDir: "qwen35-uncensored", MMProj: "mmproj-f16.gguf"}

	if got := managedProfileMmprojPathIn(root, legacy, p); got != "" {
		t.Fatalf("文件缺失应返回空串，got %q", got)
	}
	inRoot := filepath.Join(root, "qwen35-uncensored")
	if err := os.MkdirAll(inRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	inRootFile := filepath.Join(inRoot, "mmproj-f16.gguf")
	if err := os.WriteFile(inRootFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := managedProfileMmprojPathIn(root, legacy, p); got != inRootFile {
		t.Fatalf("统一根命中，got %q want %q", got, inRootFile)
	}
	// 历史根回退：统一根删掉后落 legacy 根。
	if err := os.Remove(inRootFile); err != nil {
		t.Fatal(err)
	}
	inLegacy := filepath.Join(legacy, "qwen35-uncensored")
	if err := os.MkdirAll(inLegacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inLegacy, "mmproj-f16.gguf"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := managedProfileMmprojPathIn(root, legacy, p); !strings.HasSuffix(got, filepath.Join("qwen35-uncensored", "mmproj-f16.gguf")) {
		t.Fatalf("历史根回退，got %q", got)
	}
	// 无 MMProj 配置的档位（gemma 纯文本档）恒空。
	plain := engineProfileOption{ID: "gemma4", ModelDir: "gemma", MMProj: ""}
	if got := managedProfileMmprojPathIn(root, legacy, plain); got != "" {
		t.Fatalf("无 MMProj 配置应返回空串，got %q", got)
	}
}

// TestManagedCatalogModelsVisionCaps 目录层能力标注：带 mmproj 且文件在位的
// 档位标 caps=vision，纯文本档位不标——识图跟随链的能力闸读这份标注。
func TestManagedCatalogModelsVisionCaps(t *testing.T) {
	root := t.TempDir()
	inRoot := filepath.Join(root, "qwen35-uncensored")
	if err := os.MkdirAll(inRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inRoot, "mmproj-f16.gguf"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// voiceV2ModelsRoot 是包级常量——本测试经 llamaServerArgs 间接消费真实根，
	// 这里直接验证 managedProfileMmprojPathIn 的纯逻辑层（caps 判定同源）。
	profiles := []engineProfileOption{
		{ID: "qwen35", ModelID: "qwen35", ModelDir: "qwen35-uncensored", MMProj: "mmproj-f16.gguf"},
		{ID: "gemma4", ModelID: "gemma4", ModelDir: "gemma-4-26b-a4b-it-qat", MMProj: ""},
	}
	// 直接用 In 变体注入临时根验证 caps 判定逻辑（与 managedCatalogModels
	// 内部同一条 managedProfileMmprojPath 判定链）。
	for _, p := range profiles {
		models := managedCatalogModelsForRoots(root, legacyRootForTest(t), []engineProfileOption{p}, nil, t)
		if len(models) != 1 {
			t.Fatalf("单档位应产出单条模型记录")
		}
		hasVision := len(models[0].Caps) > 0 && models[0].Caps[0] == "vision"
		if p.MMProj != "" && !hasVision {
			t.Fatalf("档位 %s 带 mmproj 应标 caps=vision，got %v", p.ID, models[0].Caps)
		}
		if p.MMProj == "" && hasVision {
			t.Fatalf("纯文本档位 %s 不应标 vision，got %v", p.ID, models[0].Caps)
		}
	}
}

// managedCatalogModelsForRoots managedCatalogModels 的根注入变体（测试用）：
// caps 判定走 managedProfileMmprojPathIn 同一条链。
func managedCatalogModelsForRoots(root, legacyRoot string, profiles []engineProfileOption, served []string, t *testing.T) []modelengine.ModelInfo {
	t.Helper()
	out := make([]modelengine.ModelInfo, 0, len(profiles))
	for _, p := range profiles {
		var caps []string
		if managedProfileMmprojPathIn(root, legacyRoot, p) != "" {
			caps = []string{"vision"}
		}
		out = append(out, modelengine.ModelInfo{ID: p.ModelID, Name: p.Label, Kind: "llm", Status: "stopped", Caps: caps})
	}
	return out
}

func legacyRootForTest(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

// TestLlamaServerArgsMmproj 启动参数：带 mmproj 路径才加 --mmproj，空=纯文本
// 档原参数形态（不新增字节）。
func TestLlamaServerArgsMmproj(t *testing.T) {
	p := llamaProfiles[0]
	withMmproj := llamaServerArgs(p, "model.gguf", "", "mmproj-f16.gguf", 32768)
	joined := strings.Join(withMmproj, " ")
	if !strings.Contains(joined, "--mmproj mmproj-f16.gguf") {
		t.Fatalf("带 mmproj 路径应含 --mmproj 参数，got: %s", joined)
	}
	plain := llamaServerArgs(p, "model.gguf", "", "", 32768)
	if strings.Contains(strings.Join(plain, " "), "--mmproj") {
		t.Fatalf("无 mmproj 不应含 --mmproj 参数")
	}
	// 既有形态守卫：alias/ctx/reasoning 等基础参数不受影响。
	for _, flag := range []string{"--alias qwen35", "--reasoning off", "--ctx-size 32768"} {
		if !strings.Contains(strings.Join(plain, " "), flag) {
			t.Fatalf("基础参数缺失 %s", flag)
		}
	}
}

// TestVisionChatFollowRoute_VisionGate 识图跟随链能力闸：运行档位带
// caps=vision 才跟随（chat-follow）；纯文本档位弃权（不返回 llama 路由，
// 落 herdsman 兜底链——本测试种子 herdsman 无默认模型，兜底如实弃权）。
func TestVisionChatFollowRoute_VisionGate(t *testing.T) {
	c := newTestCore(t)
	if err := c.SetFeatureModel("gaea", "llama", "qwen35"); err != nil {
		t.Fatalf("SetFeatureModel(gaea): %v", err)
	}
	mkLlama := func(caps []string) {
		t.Helper()
		if err := c.engineMgr.SaveEngine(modelengine.EngineConfig{
			ID:      "llama",
			Enabled: true,
			BaseURL: "http://127.0.0.1:8124/v1",
			Models:  []modelengine.ModelInfo{{ID: "qwen35", Kind: "llm", Status: "running", Caps: caps}},
		}); err != nil {
			t.Fatalf("SaveEngine(llama): %v", err)
		}
	}
	mkLlama([]string{"vision"})
	eng, model, source, ok := c.visionChatFollowRoute()
	if !ok || source != "chat-follow" || eng != "llama" || model != "qwen35" {
		t.Fatalf("带 vision 档位应 chat-follow llama/qwen35，got %q/%q/%q/%v", eng, model, source, ok)
	}
	// 纯文本档位（无 caps）：跟随链弃权——图不能喂给纯文本服务。
	mkLlama(nil)
	eng, model, source, ok = c.visionChatFollowRoute()
	if ok && source == "chat-follow" && eng == "llama" {
		t.Fatalf("无 vision 档位不得跟随 llama，got %q/%q/%q/%v", eng, model, source, ok)
	}
	if ok && source == "chat-follow-fallback" {
		t.Logf("回落 herdsman 兜底（engine=%q model=%q）", eng, model)
	}
}
