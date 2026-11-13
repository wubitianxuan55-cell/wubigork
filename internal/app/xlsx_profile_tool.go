package app

// xlsx_profile — 波2 扩面的只读列概况工具（dsh-office office_read 的 Excel
// 列概况语义移植进无头管线，见 internal/office/xlsxedit/profile.go）。
//
// 定位：概况是规划输入——模型对 >60 行的大表先用本工具了解每列（取值种类、
// min/median/max、低基数取值、可疑项），再规划 xlsx_apply 的清洗操作集；
// 改文件一律走 xlsx_apply（Journal 证据卡 + 基线快照 + 公式重算链）。
// 只读：不实现 WritePathResolver，读目标经 ReadTargeter 交 AccessPolicy 校验
// （路径解析同 officeComResolvePath 模式：相对=ResolveRelWithin，绝对=原样）。

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/gaea/wspath"
	"github.com/gaea/gaea/internal/office/xlsxedit"
)

type xlsxProfileTool struct{ a *App }

func (t xlsxProfileTool) Name() string   { return "xlsx_profile" }
func (t xlsxProfileTool) ReadOnly() bool { return true }

func (t xlsxProfileTool) Description() string { return xlsxProfileDescription }

const xlsxProfileDescription = "对既有 .xlsx 做列概况（只读，不改文件）：每列取值种类计数（数值/文本/日期/空白）、" +
	"数值列 min/median/max、低基数列各取值+次数、可疑项（首尾空格/写成文本的数字和日期/类型混杂/离群值/空行/重复行）。" +
	"概况是规划输入：>60 行的大表先用本工具看每列再规划清洗操作，不要逐行读大表；改文件用 xlsx_apply。"

func (t xlsxProfileTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type":"object",
  "required":["path"],
  "properties":{
    "path":{"type":"string","description":"目标 .xlsx 工作区相对路径"},
    "sheet":{"type":"string","description":"工作表名，缺省第一个表"},
    "range":{"type":"string","description":"区块 A1:D99，缺省整表 used range（首行按表头）"}
  }
}`)
}

// xlsxProfileResolvePath 读路径单点收口（与 officeComResolvePath 同模式，
// 只读版：相对=ResolveRelWithin；绝对=Clean 原样交上层）。不可解析返回错误。
func xlsxProfileResolvePath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("缺少文件路径")
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

func (t xlsxProfileTool) ReadTarget(args json.RawMessage) (string, bool) {
	var in struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", false
	}
	p, err := xlsxProfileResolvePath(in.Path)
	if err != nil {
		return "", false
	}
	return p, true
}

func (t xlsxProfileTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Path  string `json:"path"`
		Sheet string `json:"sheet"`
		Range string `json:"range"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("参数无效: %w", err)
	}
	path, err := xlsxProfileResolvePath(in.Path)
	if err != nil {
		return "", err
	}
	if _, statErr := osStat(path); statErr != nil {
		return "", fmt.Errorf("文件不存在：%s", in.Path)
	}
	text, err := xlsxedit.Profile(path, in.Sheet, in.Range)
	if err != nil {
		return "", err
	}
	return tool.WrapText(text), nil
}

var _ tool.Tool = xlsxProfileTool{}
var _ tool.ReadTargeter = xlsxProfileTool{}
