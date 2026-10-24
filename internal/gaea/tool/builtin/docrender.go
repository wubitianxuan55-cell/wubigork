package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/docmd"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/office/anchor"
)

func init() { tool.RegisterBuiltin(docRender{}) }

// docRender 只读渲染工具（F2 蒸馏刀，对话档 X2 裁决的交汇点）：把文档
// 渲染为页 PNG 供视觉验收/识图消费——渲染与评审分离、渲染工具只读化，
// 「只读评审员」不再为渲染弹权限卡（此前 bash 跑 soffice/pdftoppm 属
// 写操作默认 ask，评审弹卡产品不可接受）。
//
// 产物落工作区 .gaea/renders/：PDF 中间件按内容 revision 缓存（同
// revision 不重复 soffice），页 PNG 每次现渲（png 便宜，且 dpi/页区间
// 是调用参数）。目录只负责隔离，不入证据链（derived 档语义，父指针
// 归属属 F1 相位）。
type docRender struct{ workDir string }

func (docRender) Name() string { return "doc_render" }

func (docRender) Description() string {
	return "把文档渲染成页 PNG（只读，不修改原文件）：docx/pptx/xlsx/pdf 经 LibreOffice+poppler 渲染，返回 PNG 路径列表供 vision 工具/视觉验收使用。pages 支持 \"3\" 或 \"2-5\"（1-based 连续区间）。需要本机已装 LibreOffice 与 poppler，未装会如实报错。"
}

func (docRender) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"文档路径（相对工作区或绝对）：docx/pptx/xlsx/pdf"},
  "pages":{"type":"string","description":"可选页区间：\"3\" 或 \"2-5\"（1-based 含端点），缺省渲染全部页"},
  "dpi":{"type":"integer","description":"可选渲染分辨率，缺省 110（视觉验收够用，越高越慢）"}
},
"required":["path"]
}`)
}

// ReadOnly 渲染不落任何用户文件（产物在 .gaea/renders 工具自管目录），
// 权限面按只读放行——这是 F2 的存在理由。
func (docRender) ReadOnly() bool { return true }

func (d docRender) ReadTarget(args json.RawMessage) (string, bool) {
	var p struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(args, &p); err != nil || strings.TrimSpace(p.Path) == "" {
		return "", false
	}
	return resolveIn(d.workDir, p.Path), true
}

func (docRender) CompactDescription() string     { return compactDesc["doc_render"] }
func (docRender) CompactSchema() json.RawMessage { return compactSchema["doc_render"] }

var renderableExts = map[string]bool{
	".docx": true, ".pptx": true, ".xlsx": true, ".pdf": true,
	".doc": true, ".ppt": true, ".xls": true, ".odt": true, ".ods": true, ".odp": true,
}

var rePageSpan = regexp.MustCompile(`^(\d+)(?:-(\d+))?$`)

func (d docRender) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Path  string `json:"path"`
		Pages string `json:"pages"`
		DPI   int    `json:"dpi"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	src := resolveIn(d.workDir, p.Path)
	ext := strings.ToLower(filepath.Ext(src))
	if !renderableExts[ext] {
		return "", fmt.Errorf("不支持的格式 %q（支持 docx/pptx/xlsx/pdf 及 odf 系）", ext)
	}
	raw, err := os.ReadFile(src)
	if err != nil {
		return "", fmt.Errorf("读取文档失败: %w", err)
	}
	rev := anchor.Rev1(raw)

	first, last := 0, 0
	if sp := strings.TrimSpace(p.Pages); sp != "" {
		m := rePageSpan.FindStringSubmatch(sp)
		if m == nil {
			return "", fmt.Errorf("pages 只支持连续区间 \"3\" 或 \"2-5\"（1-based 含端点），不支持逗号列表")
		}
		first, _ = strconv.Atoi(m[1])
		last = first
		if m[2] != "" {
			last, _ = strconv.Atoi(m[2])
		}
		if first < 1 || last < first {
			return "", fmt.Errorf("pages 区间无效: %q", sp)
		}
	}

	rendersDir := filepath.Join(d.workDir, ".gaea", "renders")
	if err := os.MkdirAll(rendersDir, 0o755); err != nil {
		return "", fmt.Errorf("创建渲染目录失败: %w", err)
	}
	base := strings.TrimSuffix(filepath.Base(src), ext)
	stem := base + "-" + rev[:8]

	pdfPath := src
	if ext != ".pdf" {
		cached := filepath.Join(rendersDir, stem+".pdf")
		if _, err := os.Stat(cached); err != nil {
			if err := docmd.ConvertToPdfFile(src, cached); err != nil {
				return "", fmt.Errorf("文档转 PDF 失败（视觉验收需本机 LibreOffice）: %w", err)
			}
		}
		pdfPath = cached
	}

	dpi := p.DPI
	if dpi <= 0 {
		dpi = 110
	}
	pagesDir := filepath.Join(rendersDir, stem)
	if err := os.MkdirAll(pagesDir, 0o755); err != nil {
		return "", fmt.Errorf("创建页目录失败: %w", err)
	}
	pngs, err := docmd.RenderPDFPageRange(ctx, pdfPath, filepath.Join(pagesDir, "page"), dpi, first, last)
	if err != nil {
		return "", fmt.Errorf("渲染页 PNG 失败（需本机 poppler/pdftoppm）: %w", err)
	}
	if len(pngs) == 0 {
		return "", fmt.Errorf("渲染产出 0 页（文档可能为空或页区间超范围）")
	}
	out, _ := json.Marshal(map[string]any{
		"pdf":    pdfPath,
		"rev":    rev,
		"dpi":    dpi,
		"pages":  pngs,
		"count":  len(pngs),
		"vision": "用 vision 工具读取以上 PNG（image_path 逐张）",
	})
	return string(out), nil
}
