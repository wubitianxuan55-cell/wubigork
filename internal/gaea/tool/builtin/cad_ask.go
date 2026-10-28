package builtin

// cad_ask.go — CAD 图纸识读 agent 工具（2026-10-28 早实弹翻车根修：用户在对话里
// 让 AI 读 PDF 版 CAD 图纸，AI 读不了——附件识图只认图片（PDF 归 file 型不走视觉）、
// agent 无 CAD 工具可自己调、doc_render 吃不了 DWG/DXF 原生格式且 PDF 渲染 DPI
// 不受控。本工具把「CAD 看图」工具面板同款管线一步带给模型）。
//
// 管线（与 CadPanel 同口径）：校验 → DWG/DXF 原生解析（全图框默认宽渲图回填
// passRate 质量闸，缺省选质量最好图框，6000px 加宽渲图喂视觉）；PDF 按页渲染
// （ask 档 DPI 高于面板默认档，选页后整页喂视觉）。回答须附图中依据。
// 红线（docs/gaea-cad-tool-master-plan-2026-10.md）：三格式只读零写入；渲染走
// 每次调用临时目录（defer 清理，无持久副作用——UI 侧 cadToolCache 缓存不适用
// 于 agent 单次问答形态）。
import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gaea/gaea/internal/gaea/cad"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/vision"
)

func init() { tool.RegisterBuiltin(cadAsk{}) }

const (
	// cadAskWidth DWG/DXF 问图纸加宽渲染宽度（与 CadPanel ASK_WIDTH 同源：真实
	// 图幅 3000px 首屏够看，6000px 小字可读——识图喂图红线）。
	cadAskWidth = 6000
	// cadAskPDFDpi PDF 问图纸渲染 DPI（默认档 110 是「可读与耗时折中」；ask 档
	// 近两倍——对话里问图纸就是为了读字，清晰度优先，页数≤50 有界可控）。
	cadAskPDFDpi = 200
)

type cadAsk struct{ workDir string }

func (cadAsk) Name() string { return "cad_ask" }

func (cadAsk) Description() string {
	return "读取 CAD 图纸（DWG/DXF/PDF）并用视觉模型回答关于图纸的问题：自动解析图纸（DWG/DXF 原生解析按图框切分；PDF 按页），选质量最好的图框高清渲染后交给视觉模型，回答须附图中依据。多图框图纸返回图框清单（名称+清晰度），可用 sheet 参数换图框再问。用户发来/提到 CAD 图纸、DWG、DXF、PDF 图纸并问「图里有什么/读一下这张图/尺寸/数量/构件/房间布局」等问题时用本工具；PDF 图纸不是普通文档，读图纸不要用 read_file 或 format_convert。冷启动（视觉模型未加载）约 20 秒+，属正常。"
}

func (cadAsk) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"图纸文件路径（DWG/DXF/PDF，相对工作区或绝对路径）"},
  "question":{"type":"string","description":"关于图纸的问题，例如「一层有哪些房间」「统计门的数量」「图签信息是什么」"},
  "sheet":{"type":"integer","description":"可选：图框/页序号（1 基）；缺省自动选文字清晰度最好的图框（PDF=第 1 页）。多图框图纸可换图框再问"}
},
"required":["path","question"]
}`)
}

func (cadAsk) ReadOnly() bool { return true }

func (v cadAsk) ReadTarget(args json.RawMessage) (string, bool) {
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &p); err != nil || strings.TrimSpace(p.Path) == "" {
		return "", false
	}
	return resolveIn(v.workDir, p.Path), true
}

// ModelBacked 声明本工具调用本地视觉模型（同 vision 工具：桌面端按「变相子代理」
// 开 mt_ 运行记录，与子代理同一会话 UI）。
func (cadAsk) ModelBacked() bool { return true }

func (cadAsk) CompactDescription() string     { return compactDesc["cad_ask"] }
func (cadAsk) CompactSchema() json.RawMessage { return compactSchema["cad_ask"] }

type cadSheetInfo struct {
	Sheet    int     `json:"sheet"`
	Name     string  `json:"name"`
	PassRate float64 `json:"passRate"`
}

func (v cadAsk) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path     string `json:"path"`
		Question string `json:"question"`
		Sheet    int    `json:"sheet"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if strings.TrimSpace(p.Path) == "" {
		return "", fmt.Errorf("path 不能为空")
	}
	if strings.TrimSpace(p.Question) == "" {
		return "", fmt.Errorf("question 不能为空")
	}
	path := resolveIn(v.workDir, p.Path)
	kind, _, err := cad.ValidateInput(path)
	if err != nil {
		return "", err
	}
	// 每次调用独立临时目录：渲染产物生命周期=本次问答（defer 清理——识别失败也要清）。
	tmpDir, err := os.MkdirTemp("", "gaea-cad-ask-")
	if err != nil {
		return "", fmt.Errorf("创建渲染临时目录失败: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	var sheetIdx int
	var sheets []cadSheetInfo
	var pngPath, sheetName string
	switch kind {
	case cad.KindPDF:
		// PDF：按页渲染（无图框口径，passRate 恒 -1；缺省第 1 页，sheet 显式选页）。
		res, err := cad.OpenPDFContext(ctx, path, tmpDir, cadAskPDFDpi, 0)
		if err != nil {
			return "", fmt.Errorf("PDF 渲染失败: %w", err)
		}
		if len(res.Sheets) == 0 {
			return mustJSON(map[string]any{"ok": false, "path": path, "error": "PDF 未渲染出任何页"})
		}
		sheetIdx = p.Sheet
		if sheetIdx == 0 {
			sheetIdx = 1
		}
		if sheetIdx > len(res.Sheets) {
			return mustJSON(map[string]any{"ok": false, "path": path, "error": fmt.Sprintf("sheet=%d 越界：本 PDF 共 %d 页", sheetIdx, len(res.Sheets))})
		}
		for i, s := range res.Sheets {
			sheets = append(sheets, cadSheetInfo{Sheet: i + 1, Name: s.Name, PassRate: s.PassRate})
		}
		pngPath, sheetName = res.Sheets[sheetIdx-1].PNG, res.Sheets[sheetIdx-1].Name
	default:
		// DWG/DXF：原生解析 → 全图框默认宽渲图（passRate 回填）→ 选优 → 加宽渲图。
		d, err := cad.OpenContext(ctx, path)
		if err != nil {
			return "", err
		}
		outs, err := d.RenderSheets(tmpDir, 0)
		if err != nil {
			return "", fmt.Errorf("图框渲染失败: %w", err)
		}
		if len(outs) == 0 {
			return mustJSON(map[string]any{"ok": false, "path": path, "error": "图纸中未检测到图框"})
		}
		sheetIdx = p.Sheet
		if sheetIdx == 0 {
			bestRate := -2.0
			for i, o := range outs {
				if o.Sheet.PassRate > bestRate {
					bestRate = o.Sheet.PassRate
					sheetIdx = i + 1
				}
			}
		}
		if sheetIdx > len(outs) {
			return mustJSON(map[string]any{"ok": false, "path": path, "error": fmt.Sprintf("sheet=%d 越界：本图共 %d 个图框", sheetIdx, len(outs))})
		}
		for i, o := range outs {
			sheets = append(sheets, cadSheetInfo{Sheet: i + 1, Name: o.Sheet.Name, PassRate: o.Sheet.PassRate})
		}
		sheetName = outs[sheetIdx-1].Sheet.Name
		if pngPath, err = d.RenderSheet(sheetIdx-1, tmpDir, cadAskWidth); err != nil {
			return "", fmt.Errorf("高清渲染失败（图框 %d）: %w", sheetIdx, err)
		}
	}
	prompt := fmt.Sprintf(
		"请依据这张 CAD 图纸的渲染图回答问题。图纸文件：%s（图框 %d/%d：%s）。问题：%s。"+
			"要求：图上文字按原文引用；数值与结论说明图中依据；图中没有依据时明确说明，不要猜测。",
		path, sheetIdx, len(sheets), sheetName, strings.TrimSpace(p.Question))
	answer, err := vision.RecognizeImage(ctx, pngPath, prompt)
	if err != nil {
		return "", err
	}
	return mustJSON(map[string]any{
		"ok":     true,
		"path":   path,
		"kind":   kind,
		"sheet":  sheetIdx,
		"sheets": sheets,
		"answer": strings.TrimSpace(answer),
	})
}
