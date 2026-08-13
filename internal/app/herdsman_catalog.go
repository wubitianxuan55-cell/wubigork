package app

// Herdsman 模型库联动：通过 herdsman.exe 的 skill CLI（RPC 到运行中的 Herdsman
// 桌面进程）拉取完整模型目录（90 个已知模型：能力/安装状态/量化/变体/大小），
// 供模型中心「模型库」分类展示。HTTP /v1/models 只有 id+status，目录必须走 CLI。

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// HerdsmanCatalogModel 是模型目录的单条记录（字段对齐 skill models list 输出）。
type HerdsmanCatalogModel struct {
	Name             string   `json:"name"`
	DisplayName      string   `json:"display_name"`
	Type             string   `json:"type"`
	Runtime          string   `json:"runtime"`
	InferenceEngines []string `json:"inference_engines,omitempty"`
	Capabilities     []string `json:"capabilities,omitempty"`
	Installed        bool     `json:"installed"`
	Running          bool     `json:"running"`
	Status           string   `json:"status"`
	RunStatus        string   `json:"run_status,omitempty"`
	Quantization     string   `json:"quantization,omitempty"`
	ParameterCount   float64  `json:"parameter_count,omitempty"`
	ActiveParams     float64  `json:"active_parameters,omitempty"`
	IsMoE            bool     `json:"is_moe,omitempty"`
	FileSize         int64    `json:"file_size,omitempty"`
	Variants         []string `json:"llama_cpp_variants,omitempty"`
	// Hint 是本机实测/受控测评给出的用途建议（依据
	// docs/2026-08-12-herdsman-models-evaluation-report.md 与
	// docs/2026-08-13-herdsman-tts-ocr-comparison.md），前端模型库卡片展示。
	Hint string `json:"hint,omitempty"`
}

// HerdsmanCatalog 是「模型库」视图的完整载荷。
type HerdsmanCatalog struct {
	Models    []HerdsmanCatalogModel `json:"models"`
	Total     int                    `json:"total"`
	Installed int                    `json:"installed"`
	Running   int                    `json:"running"`
	Source    string                 `json:"source"`
	Error     string                 `json:"error,omitempty"`
}

// herdsmanRawModel 对齐 CLI 原始 JSON 结构。
type herdsmanRawModel struct {
	Name             string            `json:"name"`
	DisplayName      map[string]string `json:"display_name"`
	Type             string            `json:"type"`
	Runtime          string            `json:"runtime"`
	InferenceEngines []string          `json:"inference_engines"`
	Capabilities     []string          `json:"capabilities"`
	Installed        bool              `json:"installed"`
	Running          bool              `json:"running"`
	Status           string            `json:"status"`
	RunStatus        string            `json:"run_status"`
	Quantization     string            `json:"quantization"`
	ParameterCount   float64           `json:"parameter_count"`
	ActiveParams     float64           `json:"active_parameters"`
	IsMoE            bool              `json:"is_moe"`
	FileSize         int64             `json:"file_size"`
	Variants         []string          `json:"llama_cpp_variants"`
}

type herdsmanCLIResponse struct {
	OK     bool               `json:"ok"`
	Result []herdsmanRawModel `json:"result"`
	Error  string             `json:"error,omitempty"`
}

// parseHerdsmanModelList 解析 CLI JSON：优先展示名取中文，回退英文/模型名。
func parseHerdsmanModelList(data []byte) ([]HerdsmanCatalogModel, error) {
	// 兼容 UTF-8 BOM（例如经 PowerShell 管道重定向保存的 CLI 输出）。
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	var resp herdsmanCLIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("解析 herdsman 模型目录失败: %w", err)
	}
	if !resp.OK {
		msg := strings.TrimSpace(resp.Error)
		if msg == "" {
			msg = "herdsman 返回未知错误"
		}
		return nil, errors.New(msg)
	}
	out := make([]HerdsmanCatalogModel, 0, len(resp.Result))
	for _, m := range resp.Result {
		display := m.Name
		if zh := m.DisplayName["zh"]; zh != "" {
			display = zh
		} else if en := m.DisplayName["en"]; en != "" {
			display = en
		}
		out = append(out, HerdsmanCatalogModel{
			Name:             m.Name,
			DisplayName:      display,
			Type:             m.Type,
			Runtime:          m.Runtime,
			InferenceEngines: m.InferenceEngines,
			Capabilities:     m.Capabilities,
			Installed:        m.Installed,
			Running:          m.Running,
			Status:           m.Status,
			RunStatus:        m.RunStatus,
			Quantization:     m.Quantization,
			ParameterCount:   m.ParameterCount,
			ActiveParams:     m.ActiveParams,
			IsMoE:            m.IsMoE,
			FileSize:         m.FileSize,
			Variants:         m.Variants,
		})
	}
	for i := range out {
		out[i].Hint = herdsmanModelHint(out[i].Name, out[i].Capabilities)
	}
	// 排序：运行中 → 已安装 → 其余，组内按显示名。
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := out[i].Running, out[j].Running
		if ri != rj {
			return ri
		}
		ii, ij := out[i].Installed, out[j].Installed
		if ii != ij {
			return ii
		}
		return strings.ToLower(out[i].DisplayName) < strings.ToLower(out[j].DisplayName)
	})
	return out, nil
}

// herdsmanExePath 定位 herdsman.exe：优先 HERDSMAN_EXE（不校验存在性，由调用方
// 给出可读错误），回退默认安装路径，最后回退 PATH（与 internal/herdsman 探测
// 契约对齐——herdsman.exe 只在 PATH 上的安装同样可用）。
func herdsmanExePath() string {
	if p := strings.TrimSpace(os.Getenv("HERDSMAN_EXE")); p != "" {
		return p
	}
	candidates := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "starwave", "Herdsman", "herdsman.exe"),
		`C:\Program Files\starwave\Herdsman\herdsman.exe`,
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c
		}
	}
	return herdsmanExeInPath()
}

// herdsmanExeInPath 在 PATH 中查找 herdsman.exe（独立函数便于确定性测试）。
func herdsmanExeInPath() string {
	if exe, err := exec.LookPath("herdsman.exe"); err == nil {
		return exe
	}
	return ""
}

// herdsmanCLI 执行 herdsman.exe skill 命令（测试可替换；默认 25s 超时）。
var herdsmanCLI = func(args ...string) ([]byte, error) {
	return runHerdsmanCLI(25*time.Second, args...)
}

// HerdsmanModelCatalog 拉取 Herdsman 完整模型目录（模型中心「模型库」数据源）。
func (a *App) HerdsmanModelCatalog() (HerdsmanCatalog, error) {
	catalog := HerdsmanCatalog{Source: "herdsman-cli"}
	out, err := herdsmanCLI("skill", "models", "list", "--json")
	if err != nil {
		catalog.Error = err.Error()
		return catalog, err
	}
	models, err := parseHerdsmanModelList(out)
	if err != nil {
		catalog.Error = err.Error()
		return catalog, err
	}
	catalog.Models = models
	catalog.Total = len(models)
	for _, m := range models {
		if m.Installed {
			catalog.Installed++
		}
		if m.Running {
			catalog.Running++
		}
	}
	return catalog, nil
}

// herdsmanModelHint 按模型名/能力给出用途建议。依据：
// - docs/2026-08-12-herdsman-models-evaluation-report.md（120 组受控对照测评）
// - docs/2026-08-13-herdsman-tts-ocr-comparison.md（TTS/OCR 本机实测）
// 关键词按序匹配，命中即返回，未命中返回空（前端不显示提示）。
func herdsmanModelHint(name string, capabilities []string) string {
	n := strings.ToLower(name)
	has := func(key string) bool {
		if strings.Contains(n, key) {
			return true
		}
		for _, c := range capabilities {
			if strings.Contains(strings.ToLower(c), key) {
				return true
			}
		}
		return false
	}
	switch {
	case has("hauhaucs"):
		return "日常对话/识图首选：约 65 tok/s 最快，识图表格理解最准，正文输出最充分"
	case has("lynnstyle"):
		return "复杂任务/可审计推理：唯一稳定输出独立推理链；思考模式请给足 token（≥4096）"
	case has("hermes"):
		return "长篇/角色化输出：篇幅扎实；不建议开思考模式（推理会混入正文）"
	case has("qwen3-embedding") || has("bge-m3") || has("embedding"):
		return "本地语义向量（embedding），驱动语义召回与文件索引"
	case has("qwen3-reranker") || has("bge-reranker") || has("rerank"):
		return "本地精排（rerank），成本库/知识库检索重排序"
	case has("paddleocr") || has("ocr"):
		return "快速 OCR（约 90ms）：中文混合场景有错字，失败自动回退 OvisOCR2"
	case has("mineru") || has("document-parse") || has("parse"):
		return "文档结构解析（PDF/图片/Office → Markdown）；中文按字加空格，适合 OCR 兜底"
	case has("voxcpm2"):
		return "本地 TTS：冷启动约 50 秒；不支持预设音色（用描述语而非 voice 名）"
	case has("qwen3-tts"):
		return "本地 TTS（声音设计/克隆）：需先安装，未安装时语音回退 voxcpm2"
	case has("edge-tts"):
		return "云端 TTS：本机服务端曾返回空音频，使用前先测试连接"
	case has("zimage") || has("text-to-image") || has("image-to-image") || has("image-generation"):
		return "本地文生图（19GB）：绘梦板块 herdsman 后端"
	case has("hy-mt") || has("hunyuan-mt") || has("translation"):
		return "本地翻译：translate_text 工具优先使用"
	case has("sherpa") || has("speech-to-text"):
		return "实时语音识别（流式 ASR）"
	case has("gemma"):
		return "通用对话备选：12B 参数，资源占用低"
	}
	return ""
}
