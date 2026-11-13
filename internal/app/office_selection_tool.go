package app

// office_selection_tool.go — 选区上下文（P0-3，灵犀 <wps_docs_selection> 蒸馏）。
//
// 解决的问题：用户指着屏幕上选中的一段话说「改成更正式的说法」时，模型要么
// 让用户把文字贴进来，要么整篇 read 一遍再猜位置。灵犀的做法是在对话上下文
// 里注入结构化选区（文档 + 位置 + 原文 + 前后文），模型直接做最小改动。
//
// 本工具只读：读 helper 的 selection 回执并排版成人可读文本。写入仍走
// office_edit（配合 replace_text 的 find/expect 防错位）。

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
)

type officeComSelectionTool struct{}

func (officeComSelectionTool) Name() string { return "office_selection" }

// ReadOnly 恒 true：只读选区，不改文档。
func (officeComSelectionTool) ReadOnly() bool { return true }

func (officeComSelectionTool) Description() string {
	return "读取用户此刻在 Office/WPS 里选中的内容与前后文：Word 给选区文字+段落号+前后各 N 字；Excel 给区域地址+工作表+首格文本；PowerPoint 给页码+形状名+文字。" +
		"用户说「这段」「选中的这句话」而没把原文贴进来时先调这个，再按选区做最小改动（office_edit + replace_text 的 find/expect）。用户没选中时返回明确的未选中说明。"
}

func (officeComSelectionTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type":"object",
  "properties":{
    "doc":{"type":"string","description":"文档路径或窗口名；缺省=最近 open/read/edit 过的文档"},
    "app":{"type":"string","enum":["word","excel","ppt"],"description":"仅在 doc 无扩展名或要读另一应用的活动文档时需要"},
    "context":{"type":"integer","description":"Word：选区前后各取多少字符（缺省 200，最大 2000）"}
  }
}`)
}

// officeSelectionReply 是 helper selection 回执的形状（三应用共用一份，
// 未用字段留零值——比按应用分三份结构更能容忍 helper 侧加字段）。
type officeSelectionReply struct {
	App      string `json:"app"`
	Selected bool   `json:"selected"`
	Text     string `json:"text"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Para     int    `json:"para"`
	Before   string `json:"before"`
	After    string `json:"after"`
	Address  string `json:"address"`
	Sheet    string `json:"sheet"`
	Rows     int    `json:"rows"`
	Cols     int    `json:"cols"`
	Slide    int    `json:"slide"`
	Shape    string `json:"shape"`
	Shapes   int    `json:"shapes"`
	Error    string `json:"error"`
}

func (t officeComSelectionTool) ExecuteWithContext(ctx context.Context, tc tool.ToolContext, args json.RawMessage) (string, error) {
	return t.execute(ctx, args)
}

func (t officeComSelectionTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	return t.execute(ctx, args)
}

func (t officeComSelectionTool) execute(ctx context.Context, args json.RawMessage) (string, error) {
	var in map[string]any
	if strings.TrimSpace(string(args)) != "" {
		if err := json.Unmarshal(args, &in); err != nil {
			return "", fmt.Errorf("office_selection 参数解析失败: %w", err)
		}
	}
	if in == nil {
		in = map[string]any{}
	}
	doc, _ := in["doc"].(string)
	appNameArg, _ := in["app"].(string)
	kind, targetDoc, err := officeCom.target(appNameArg, doc)
	if err != nil {
		return "", err
	}
	delete(in, "doc")
	delete(in, "app")
	in["app"] = kind
	if targetDoc != "" && officeComAbsRe.MatchString(targetDoc) {
		in["doc"] = targetDoc
	}
	raw, err := officeCom.call(ctx, "selection", in, officeComTimeoutRead)
	if err != nil {
		return "", err
	}
	var sel officeSelectionReply
	if err := json.Unmarshal(raw, &sel); err != nil {
		return "", fmt.Errorf("office helper selection 回执解析失败: %w", err)
	}
	return formatSelectionReply(sel, kind), nil
}

// formatSelectionReply 把回执排版成给模型看的文本（纯函数，便于单测）。
func formatSelectionReply(sel officeSelectionReply, fallbackApp string) string {
	if sel.Error != "" {
		return fmt.Sprintf("读不到当前选区（%s）。让用户先在文档里选中要改的内容，或把原文贴进来。", sel.Error)
	}
	if !sel.Selected {
		return "当前没有选区：用户在文档里没有选中内容。需要最小改动时，请用户先选中要改的部分，或改用 office_read 定位。"
	}
	app := sel.App
	if app == "" {
		app = fallbackApp
	}
	var b strings.Builder
	switch app {
	case "excel":
		fmt.Fprintf(&b, "选区（Excel，工作表 %s，区域 %s", sel.Sheet, sel.Address)
		if sel.Rows > 0 && sel.Cols > 0 {
			fmt.Fprintf(&b, "，%d 行 × %d 列", sel.Rows, sel.Cols)
		}
		b.WriteString("）\n")
		if strings.TrimSpace(sel.Text) != "" {
			fmt.Fprintf(&b, "首格文本：%s\n", clipRunes(sel.Text, 300))
		}
		b.WriteString("改这一块用 office_edit 的 write_range / format_range，range 直接传给上面的区域。")
	case "ppt":
		fmt.Fprintf(&b, "选区（PowerPoint，第 %d 页", sel.Slide)
		if sel.Shape != "" {
			fmt.Fprintf(&b, "，形状「%s」", sel.Shape)
		}
		if sel.Shapes > 1 {
			fmt.Fprintf(&b, "，共选中 %d 个形状（正文取第一个）", sel.Shapes)
		}
		b.WriteString("）\n")
		if strings.TrimSpace(sel.Text) != "" {
			fmt.Fprintf(&b, "选中文字：%s\n", clipRunes(sel.Text, 600))
		}
		b.WriteString("改这一处用 office_edit 的 set_text / format_text，定位用上面的页码 + 形状名。")
	default: // word
		fmt.Fprintf(&b, "选区（Word，第 %d 段，字符 %d-%d，共 %d 字）\n", sel.Para, sel.Start, sel.End, len([]rune(sel.Text)))
		fmt.Fprintf(&b, "选中：%s\n", clipRunes(sel.Text, 600))
		if strings.TrimSpace(sel.Before) != "" {
			fmt.Fprintf(&b, "上文：…%s\n", clipRunes(sel.Before, 300))
		}
		if strings.TrimSpace(sel.After) != "" {
			fmt.Fprintf(&b, "下文：%s…\n", clipRunes(sel.After, 300))
		}
		b.WriteString("改这一处用 office_edit 的 replace_text：find 用上面的选中原文（可带上下文避免多处误替换），expect 校验段落号防错位。")
	}
	return b.String()
}

// clipRunes 按 rune 截断（不切裂多字节字符），用于选区文本预览。
func clipRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}
