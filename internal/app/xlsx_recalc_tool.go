package app

// xlsx_recalc — 灵犀 optimize plan P1-2 的公式重算工具：把既有 LibreOffice
// 重算链（xlsxedit.Recalc，.gaea/skills/xlsx/scripts/recalc.py 宏）暴露成
// agent 工具面。原 GaeaXlsxRecalc 是前端「重算」钮用的（返回预览投影），
// 本工具给 agent 返回重算报告文本。
//
// 写面：重算会用缓存值改写文件（公式本身不动），因此实现 WritePathResolver
// 声明 path；不落证据卡——刷新的是派生数据（公式结果缓存），公式与用户数据
// 不变，审计面在 xlsx_audit。

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

type xlsxRecalcTool struct{ a *App }

func (t xlsxRecalcTool) Name() string   { return "xlsx_recalc" }
func (t xlsxRecalcTool) ReadOnly() bool { return false }

func (t xlsxRecalcTool) Description() string {
	return "用 LibreOffice 重算 .xlsx 里全部公式并写回缓存值（公式本身不改，刷新的是结果缓存）。" +
		"openpyxl/excelize 生成的公式没有缓存值（Excel/WPS 打开时才显示结果），交付前必须重算；" +
		"重算同时报出公式错误计数（Err:522=循环引用，Err:508=配对缺失等）。" +
		"交付门：重算后用 xlsx_audit 复核，零错误才算交付。依赖本机 LibreOffice——未安装时报错如实说明。"
}

func (t xlsxRecalcTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type":"object",
  "required":["path"],
  "properties":{
    "path":{"type":"string","description":"目标 .xlsx 工作区相对路径"}
  }
}`)
}

func (t xlsxRecalcTool) WritePaths(args json.RawMessage) ([]string, error) {
	return officeWritePathsFromArgs(args, "path")
}

func (t xlsxRecalcTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("xlsx_recalc 参数解析失败: %w", err)
	}
	if strings.TrimSpace(in.Path) == "" {
		return "", fmt.Errorf("缺少参数 path（目标 .xlsx 工作区相对路径）")
	}
	path := in.Path
	if !filepath.IsAbs(path) {
		p, perr := wspath.ResolveRelWithin(gaeaCwd(), path)
		if perr != nil {
			return "", fmt.Errorf("非法工作区相对路径: %s", path)
		}
		path = p
	}
	if !xlsxedit.AuditFileExists(path) {
		return "", fmt.Errorf("文件不存在：%s", in.Path)
	}
	rep, err := xlsxedit.Recalc(path, gaeaCwd())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "recalc.py") ||
			strings.Contains(err.Error(), "重算失败") {
			return "", fmt.Errorf("公式重算不可用：%v（需要本机 LibreOffice 与 .gaea/skills/xlsx/scripts/recalc.py）", err)
		}
		return "", fmt.Errorf("公式重算失败：%w", err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "重算完成：%d 个公式，%d 个错误", rep.TotalFormulas, rep.TotalErrors)
	if len(rep.ErrorSummary) > 0 {
		b.WriteString("（错误分布：")
		first := true
		for k, v := range rep.ErrorSummary {
			if !first {
				b.WriteString("，")
			}
			first = false
			fmt.Fprintf(&b, "%s=%v", k, v)
		}
		b.WriteString("）")
	}
	if rep.TotalErrors > 0 {
		b.WriteString("\n有公式错误未达交付门：用 xlsx_audit 看具体单元格，修复后再次重算复核；" +
			"Err:522=循环引用，Err:508=括号/配对缺失，Err:507=除零。")
	} else {
		b.WriteString("\n零错误，达到交付门（可再跑 xlsx_audit 复核未重算清单）。")
	}
	return b.String(), nil
}

var _ tool.Tool = xlsxRecalcTool{}
var _ tool.WritePathResolver = xlsxRecalcTool{}
