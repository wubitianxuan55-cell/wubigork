package app

// office 工具族 — P1a 结构化写入口（2026-11-09 众神殿 pan_1794234593182 裁决
// action-3：四类写升 agent 工具同走 Journal；从零生成走 write_paths 声明入口）。
//
//   - office_generate：从零生成的唯一声明入口。包装 GaeaExportDeliverable（模板
//     链 create_docx.py 等，不重写生成器）+ output_path 声明；实现 WritePathResolver
//     （DAG write_paths 声明面可被 WriteScopePolicy 逐调用包含校验；bash 等无
//     resolver 写工具在声明面出局=agent/task.go:138-140）；不实现 PersistWriteTool
//     （FilterRegistry 不摘新入口，裁决验收线）。
//   - docx_apply / docx_accept / pptx_apply / xlsx_apply：四类写升 agent 工具，
//     薄包装 App 绑定方法——Journal 落账在 App 方法内部单点 appendOfficeEvidence，
//     agent 工具层不重复记账（不建第四本账，裁决 issue-3 收口）。docx_accept 仅
//     accept=true（接受修订的不可逆整理；拒绝路径=回滚语义留 UI，不暴露给 agent，
//     裁决 Grok-R4 口径）。
//
// 装配点=gaeaExtraTools（唯一注册表四入口之一，boot.Build reg.Add 同账）。

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/wspath"
)

// officeResolveWritePath 把工具参数里的目标路径归一到绝对路径（WritePathResolver
// 与 App 方法同判据：相对=ResolveRelWithin，绝对=原样交 WriteScopePolicy 校验）。
func officeResolveWritePath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("缺少目标文件路径")
	}
	if filepath.IsAbs(raw) {
		return filepath.Clean(raw), nil
	}
	p, err := wspath.ResolveRelWithin(gaeaCwd(), raw)
	if err != nil {
		return "", fmt.Errorf("非法工作区相对路径: %s", raw)
	}
	return p, nil
}

// officeWritePathsFromArgs 通用 WritePathResolver 实现：取 args 中 path/output_path。
func officeWritePathsFromArgs(args json.RawMessage, keys ...string) ([]string, error) {
	var fields struct {
		Path       string `json:"path"`
		OutputPath string `json:"output_path"`
	}
	if len(args) > 0 {
		if err := json.Unmarshal(args, &fields); err != nil {
			return nil, fmt.Errorf("参数无效: %w", err)
		}
	}
	for _, k := range keys {
		raw := fields.Path
		if k == "output_path" {
			raw = fields.OutputPath
		}
		if strings.TrimSpace(raw) == "" {
			continue
		}
		p, err := officeResolveWritePath(raw)
		if err != nil {
			return nil, err
		}
		return []string{p}, nil
	}
	return nil, fmt.Errorf("缺少写路径参数（%s）", strings.Join(keys, "/"))
}

// ── office_generate：从零生成声明入口 ─────────────────────────────────────

type officeGenerateTool struct{ a *App }

func (t officeGenerateTool) Name() string        { return "office_generate" }
func (t officeGenerateTool) ReadOnly() bool      { return false }
func (t officeGenerateTool) Description() string { return officeGenerateDescription }

const officeGenerateDescription = "从零生成 Office 交付物（docx/pptx/xlsx/md/pdf）：受控 Markdown 经模板链排版落盘。" +
	"output_path 为必填声明写路径（工作区相对路径，落点与声明逐字节一致）；生成后自动登记证据卡（可复核/回滚）。" +
	"编辑既有文档请用 docx_apply/pptx_apply/xlsx_apply，不要用本工具覆盖。"

func (t officeGenerateTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type":"object",
  "required":["markdown","format","output_path"],
  "properties":{
    "markdown":{"type":"string","description":"受控 Markdown 全文（# 标题/段落/表格）"},
    "format":{"type":"string","enum":["docx","pptx","xlsx","md","pdf"],"description":"交付格式"},
    "output_path":{"type":"string","description":"声明写路径（工作区相对路径，如 报告/季度总结.docx）；扩展名须与 format 一致"},
    "title":{"type":"string","description":"文档标题（缺省取 Markdown 首个标题）"},
    "template":{"type":"string","enum":["通用","公文","报告","合同"],"description":"docx 模板，缺省 通用"},
    "cover":{"type":"boolean","description":"docx 生成封面"},
    "toc":{"type":"boolean","description":"docx 生成目录"},
    "header":{"type":"string","description":"docx 页眉"},
    "footer":{"type":"string","description":"docx 页脚"}
  }
}`)
}

func (t officeGenerateTool) WritePaths(args json.RawMessage) ([]string, error) {
	return officeWritePathsFromArgs(args, "output_path")
}

func (t officeGenerateTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in ExportDeliverableInput
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if strings.TrimSpace(in.OutputPath) == "" {
		return "", fmt.Errorf("缺少 output_path（声明写路径）：durable 交付物必须声明落点，不接受隐式 ExportsDir 命名")
	}
	in.Format = strings.ToLower(strings.TrimPrefix(in.Format, "."))
	res, err := t.a.GaeaExportDeliverable(in)
	if err != nil {
		return "", err
	}
	// Journal 证据卡：新文件无基线快照（Verifier 通道 B 记 n/a=诚实缺省，不冒充
	// 通过）；声明路径与实际落点逐字节一致是本工具的验收线。
	appendOfficeEvidence("office_generate", res.Path,
		"从零生成（Markdown 模板链，无基线快照）",
		fmt.Sprintf("format=%s size=%d bytes；落点=%s", res.Format, res.Size, res.Path),
		"", "")
	return fmt.Sprintf("已生成 %s：%s（%d bytes）。声明路径与实际落点一致；可用 doc_render 预览，交付面板可复核。", res.Format, res.Path, res.Size), nil
}

// ── 四类写：薄包装 App 绑定方法 ──────────────────────────────────────────

// officeDocxApplyTool docx_apply：修订制替换（w:del+w:ins）写入既有 docx。
type officeDocxApplyTool struct{ a *App }

func (t officeDocxApplyTool) Name() string        { return "docx_apply" }
func (t officeDocxApplyTool) ReadOnly() bool      { return false }
func (t officeDocxApplyTool) Description() string { return officeDocxApplyDescription }

const officeDocxApplyDescription = "对既有 .docx 应用修订制文字替换（选中文本→替换文本，Word 修订可见、可拒绝可接受）。" +
	"落盘前自动做基线快照并登记证据卡，可在交付面板复核或一键回滚。只改文字内容，不动版式。"

func (t officeDocxApplyTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type":"object",
  "required":["path","selected_text","replacement"],
  "properties":{
    "path":{"type":"string","description":"目标 .docx 工作区相对路径"},
    "selected_text":{"type":"string","description":"要替换的原文（须在文档中唯一且逐字存在）"},
    "replacement":{"type":"string","description":"替换后的新文本"}
  }
}`)
}

func (t officeDocxApplyTool) WritePaths(args json.RawMessage) ([]string, error) {
	return officeWritePathsFromArgs(args, "path")
}

func (t officeDocxApplyTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Path        string `json:"path"`
		Selected    string `json:"selected_text"`
		Replacement string `json:"replacement"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if strings.TrimSpace(in.Selected) == "" {
		return "", fmt.Errorf("selected_text 为空")
	}
	if _, err := t.a.GaeaDocxApplyEdit(in.Path, in.Selected, in.Replacement); err != nil {
		return "", err
	}
	return fmt.Sprintf("docx 修订已写入 %s（修订制写入，证据卡已落账，可复核/回滚）。", in.Path), nil
}

// officeDocxAcceptTool docx_accept：接受全部待处理修订（仅 accept=true；拒绝路径
// 是回滚语义，留 UI 不暴露）。
type officeDocxAcceptTool struct{ a *App }

func (t officeDocxAcceptTool) Name() string        { return "docx_accept" }
func (t officeDocxAcceptTool) ReadOnly() bool      { return false }
func (t officeDocxAcceptTool) Description() string { return officeDocxAcceptDescription }

const officeDocxAcceptDescription = "接受目标 .docx 的全部待处理修订（清除修订标记的不可逆整理）。" +
	"落盘前自动基线快照并登记证据卡，可复核可回滚。仅接受：拒绝/单条挑选请让用户在界面操作。"

func (t officeDocxAcceptTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type":"object",
  "required":["path"],
  "properties":{
    "path":{"type":"string","description":"目标 .docx 工作区相对路径"}
  }
}`)
}

func (t officeDocxAcceptTool) WritePaths(args json.RawMessage) ([]string, error) {
	return officeWritePathsFromArgs(args, "path")
}

func (t officeDocxAcceptTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if _, err := t.a.GaeaDocxAcceptChanges(in.Path, true); err != nil {
		return "", err
	}
	return fmt.Sprintf("docx 修订已全部接受并清除标记：%s（证据卡已落账，可复核/回滚）。", in.Path), nil
}

// officePptxApplyTool pptx_apply：页码+目标文本→替换文本（run 级 rPr 原字节继承）。
type officePptxApplyTool struct{ a *App }

func (t officePptxApplyTool) Name() string        { return "pptx_apply" }
func (t officePptxApplyTool) ReadOnly() bool      { return false }
func (t officePptxApplyTool) Description() string { return officePptxApplyDescription }

const officePptxApplyDescription = "对既有 .pptx 指定页应用文字替换（页码+目标文本→替换文本，样式原样继承）。" +
	"落盘前自动基线快照并登记证据卡，可复核可回滚。"

func (t officePptxApplyTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type":"object",
  "required":["path","slide","target","replacement"],
  "properties":{
    "path":{"type":"string","description":"目标 .pptx 工作区相对路径"},
    "slide":{"type":"integer","minimum":1,"description":"页码（从 1 起）"},
    "target":{"type":"string","description":"该页内要替换的原文（逐字存在）"},
    "replacement":{"type":"string","description":"替换后的新文本"}
  }
}`)
}

func (t officePptxApplyTool) WritePaths(args json.RawMessage) ([]string, error) {
	return officeWritePathsFromArgs(args, "path")
}

func (t officePptxApplyTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Path        string `json:"path"`
		Slide       int    `json:"slide"`
		Target      string `json:"target"`
		Replacement string `json:"replacement"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	if _, err := t.a.GaeaPptxApplyEdit(in.Path, in.Slide, in.Target, in.Replacement); err != nil {
		return "", err
	}
	return fmt.Sprintf("pptx 第 %d 页替换已写入 %s（证据卡已落账，可复核/回滚）。", in.Slide, in.Path), nil
}

// officeXlsxApplyTool xlsx_apply：应用操作集 JSON（set 公式/值、find/replace 等，
// 与 UI Plan→Apply 同一 ops 契约），应用后自动公式重算。
type officeXlsxApplyTool struct{ a *App }

func (t officeXlsxApplyTool) Name() string        { return "xlsx_apply" }
func (t officeXlsxApplyTool) ReadOnly() bool      { return false }
func (t officeXlsxApplyTool) Description() string { return officeXlsxApplyDescription }

const officeXlsxApplyDescription = "对既有 .xlsx 应用操作集（JSON 数组，与界面 Plan→Apply 同一契约：set_value/set_formula/find_replace 等；" +
	"每项含 sheet/target/type/value，可选 find/replace）。应用后自动公式重算，落盘前基线快照+证据卡，可复核可回滚。" +
	"复杂改动先用 xlsx 工具规划操作集，再整体应用，不要逐格散改。"

func (t officeXlsxApplyTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type":"object",
  "required":["path","ops_json"],
  "properties":{
    "path":{"type":"string","description":"目标 .xlsx 工作区相对路径"},
    "ops_json":{"type":"string","description":"操作集 JSON 数组，如 [{\"sheet\":\"Sheet1\",\"target\":\"B2\",\"type\":\"set_value\",\"value\":42}]"}
  }
}`)
}

func (t officeXlsxApplyTool) WritePaths(args json.RawMessage) ([]string, error) {
	return officeWritePathsFromArgs(args, "path")
}

func (t officeXlsxApplyTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Path    string `json:"path"`
		OpsJSON string `json:"ops_json"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	res, err := t.a.GaeaXlsxApplyEdit(in.Path, in.OpsJSON)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("xlsx 已应用 %d 个操作：%s（%s；证据卡已落账，可复核/回滚。）", res.Applied, in.Path, res.Summary), nil
}

var _ tool.Tool = officeGenerateTool{}
var _ tool.Tool = officeDocxApplyTool{}
var _ tool.Tool = officeDocxAcceptTool{}
var _ tool.Tool = officePptxApplyTool{}
var _ tool.Tool = officeXlsxApplyTool{}
var _ tool.WritePathResolver = officeGenerateTool{}
var _ tool.WritePathResolver = officeDocxApplyTool{}
var _ tool.WritePathResolver = officeDocxAcceptTool{}
var _ tool.WritePathResolver = officePptxApplyTool{}
var _ tool.WritePathResolver = officeXlsxApplyTool{}
