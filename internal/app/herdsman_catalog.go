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

// herdsmanExePath 定位 herdsman.exe：优先 HERDSMAN_EXE，回退默认安装路径。
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
