package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const herdsmanCatalogFixture = `{
  "version": 1,
  "request_id": "test",
  "ok": true,
  "result": [
    {
      "name": "bge-m3",
      "display_name": {"en": "bge-m3", "zh": "bge-m3"},
      "type": "embedding",
      "runtime": "llama.cpp",
      "inference_engines": ["llama.cpp"],
      "installed": true,
      "running": true,
      "status": "installed",
      "run_status": "running",
      "quantization": "Q4_K_M",
      "parameter_count": 0.568,
      "file_size": 437778496,
      "llama_cpp_variants": ["standard"]
    },
    {
      "name": "Hunyuan-MT:7B",
      "display_name": {"en": "Hunyuan-MT 7B", "zh": "混元 MT 7B 翻译模型"},
      "type": "text-generation",
      "runtime": "llama.cpp",
      "inference_engines": ["llama.cpp"],
      "capabilities": ["translation"],
      "installed": false,
      "running": false,
      "status": "uninstalled",
      "run_status": "stopped",
      "quantization": "Q4_K_M",
      "parameter_count": 7,
      "llama_cpp_variants": ["standard", "phison-aicache"]
    },
    {
      "name": "Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-2",
      "display_name": {
        "en": "Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive",
        "zh": "Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive"
      },
      "type": "multimodal",
      "runtime": "llama.cpp",
      "inference_engines": ["llama.cpp"],
      "installed": true,
      "running": true,
      "status": "installed",
      "run_status": "running",
      "active_parameters": 3,
      "is_moe": true,
      "parameter_count": 35,
      "file_size": 23424536704,
      "llama_cpp_variants": ["standard"]
    },
    {
      "name": "zimage-turbo",
      "display_name": {"en": "Z-Image-Turbo", "zh": "Z-Image-Turbo"},
      "type": "image-generation",
      "runtime": "sd-cpp",
      "inference_engines": ["sd-cpp"],
      "capabilities": ["text-to-image", "image-to-image"],
      "installed": true,
      "running": false,
      "status": "installed",
      "run_status": "stopped",
      "quantization": "Q4_K",
      "file_size": 20027974026
    }
  ]
}`

func TestParseHerdsmanModelList(t *testing.T) {
	models, err := parseHerdsmanModelList([]byte(herdsmanCatalogFixture))
	if err != nil {
		t.Fatalf("parseHerdsmanModelList: %v", err)
	}
	if len(models) != 4 {
		t.Fatalf("len(models) = %d, want 4", len(models))
	}
	// 排序：运行中在前（bge-m3、Qwen3.6），随后已安装（zimage-turbo），最后未安装。
	if !models[0].Running || !models[1].Running {
		t.Errorf("前两项应为运行中，got %q/%q", models[0].Name, models[1].Name)
	}
	if models[2].Name != "zimage-turbo" || models[3].Name != "Hunyuan-MT:7B" {
		t.Errorf("后两项应为已安装(zimage-turbo)/未安装(Hunyuan-MT)，got %q/%q", models[2].Name, models[3].Name)
	}
	// 显示名中文回退。
	byName := map[string]HerdsmanCatalogModel{}
	for _, m := range models {
		byName[m.Name] = m
	}
	if got := byName["Hunyuan-MT:7B"].DisplayName; got != "混元 MT 7B 翻译模型" {
		t.Errorf("DisplayName(zh) = %q, want 混元 MT 7B 翻译模型", got)
	}
	if got := byName["Hunyuan-MT:7B"].Capabilities; len(got) != 1 || got[0] != "translation" {
		t.Errorf("Capabilities = %v, want [translation]", got)
	}
	if got := byName["Hunyuan-MT:7B"].Variants; len(got) != 2 {
		t.Errorf("Variants = %v, want 2 项", got)
	}
	if !byName["Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-2"].IsMoE {
		t.Errorf("Qwen3.6 应标记 is_moe")
	}
	if byName["zimage-turbo"].FileSize != 20027974026 {
		t.Errorf("zimage-turbo FileSize = %d", byName["zimage-turbo"].FileSize)
	}
	// 用途提示（H0-5：模型默认值/用途对齐测评结论）。
	if got := byName["Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-2"].Hint; !strings.Contains(got, "识图") {
		t.Errorf("HauhauCS Hint = %q, want 含「识图」的日常首选提示", got)
	}
	if got := byName["Hunyuan-MT:7B"].Hint; !strings.Contains(got, "翻译") {
		t.Errorf("Hunyuan-MT Hint = %q, want 翻译提示", got)
	}
	if got := byName["zimage-turbo"].Hint; !strings.Contains(got, "文生图") {
		t.Errorf("zimage-turbo Hint = %q, want 文生图提示", got)
	}
	if got := byName["bge-m3"].Hint; !strings.Contains(got, "向量") {
		t.Errorf("bge-m3 Hint = %q, want embedding 提示", got)
	}
}

// TestHerdsmanModelHint 覆盖各关键词分支与未命中回退。
func TestHerdsmanModelHint(t *testing.T) {
	cases := []struct {
		name string
		caps []string
		want string // 空 = 期望无提示
	}{
		{"Qwen3.6-35B-A3B-Uncensored-HauhauCS-Aggressive-Q4_K_P-2", nil, "识图"},
		{"Qwen3.6-35B-A3B-DSV4Pro-SFT-GPT56Sol-RL-Agent-LynnStyle-Q5-imatrix", nil, "推理"},
		{"Hermes3.6-35B-A3B-Uncensored-Genesis-V7-MTP-APEX", nil, "思考模式"},
		{"qwen3-embedding-4b", nil, "向量"},
		{"bge-reranker-v2-m3", nil, "精排"},
		{"paddleocr-ppocrv5-server", nil, "OCR"},
		{"mineru-pipeline-hybrid", nil, "解析"},
		{"voxcpm2", nil, "TTS"},
		{"qwen3-tts-customvoice", nil, "需先安装"},
		{"edge-tts", nil, "空音频"},
		{"zimage-turbo", nil, "文生图"},
		{"Hy-MT1.5_1.8B", nil, "翻译"},
		{"sherpa-onnx-streaming-zipformer-zh-14m", nil, "语音识别"},
		{"some-unknown-model", nil, ""},
		// 能力标签兜底：名称不含关键词时按 capabilities 命中。
		{"Qwen3.6-X", []string{"rerank"}, "精排"},
	}
	for _, c := range cases {
		got := herdsmanModelHint(c.name, c.caps)
		if c.want == "" {
			if got != "" {
				t.Errorf("herdsmanModelHint(%q) = %q, want 空", c.name, got)
			}
			continue
		}
		if !strings.Contains(got, c.want) {
			t.Errorf("herdsmanModelHint(%q) = %q, want 含 %q", c.name, got, c.want)
		}
	}
}

func TestParseHerdsmanModelList_ErrorAndInvalid(t *testing.T) {
	if _, err := parseHerdsmanModelList([]byte(`{"ok":false,"error":"模型服务未启动"}`)); err == nil {
		t.Error("ok:false 应返回错误")
	}
	if _, err := parseHerdsmanModelList([]byte(`not-json`)); err == nil {
		t.Error("非法 JSON 应返回错误")
	}
}

func TestHerdsmanExePath(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "herdsman.exe")
	if err := os.WriteFile(fake, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HERDSMAN_EXE", fake)
	if got := herdsmanExePath(); got != fake {
		t.Fatalf("herdsmanExePath() = %q, want %q", got, fake)
	}
	// 环境变量优先，即使路径不存在也原样返回（由调用方给出可读错误）。
	missing := filepath.Join(dir, "missing.exe")
	t.Setenv("HERDSMAN_EXE", missing)
	if got := herdsmanExePath(); got != missing {
		t.Fatalf("环境变量应优先，got %q", got)
	}
}

// TestHerdsmanExeInPath 验证 PATH 回退查找（独立函数，确定性测试；
// herdsmanExePath 的默认安装路径候选在开发机上可能真实存在，无法在测试中消除）。
func TestHerdsmanExeInPath(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "herdsman.exe")
	if err := os.WriteFile(fake, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if got := herdsmanExeInPath(); got != fake {
		t.Fatalf("PATH 回退失败: got %q, want %q", got, fake)
	}
	// PATH 无 herdsman.exe 时返回空。
	empty := t.TempDir()
	t.Setenv("PATH", empty)
	if got := herdsmanExeInPath(); got != "" {
		t.Fatalf("PATH 未命中应返回空，got %q", got)
	}
}

func TestHerdsmanModelCatalog(t *testing.T) {
	orig := herdsmanCLI
	defer func() { herdsmanCLI = orig }()

	herdsmanCLI = func(args ...string) ([]byte, error) {
		return []byte(herdsmanCatalogFixture), nil
	}
	a := &App{}
	catalog, err := a.HerdsmanModelCatalog()
	if err != nil {
		t.Fatalf("HerdsmanModelCatalog: %v", err)
	}
	if catalog.Total != 4 || catalog.Installed != 3 || catalog.Running != 2 {
		t.Fatalf("汇总错误: total=%d installed=%d running=%d", catalog.Total, catalog.Installed, catalog.Running)
	}
	if catalog.Source != "herdsman-cli" {
		t.Errorf("Source = %q", catalog.Source)
	}

	// CLI 失败：错误透出且不 panic。
	herdsmanCLI = func(args ...string) ([]byte, error) {
		return nil, errors.New("herdsman 未运行")
	}
	catalog, err = a.HerdsmanModelCatalog()
	if err == nil || !strings.Contains(catalog.Error, "herdsman 未运行") {
		t.Fatalf("CLI 失败应透出错误: err=%v catalog.Error=%q", err, catalog.Error)
	}
}
