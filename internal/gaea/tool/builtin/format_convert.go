// Package builtin — 文档格式转换工具（docx/xlsx/pdf → Markdown）。
package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/docmd"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() { tool.RegisterBuiltin(formatConvert{}) }

type formatConvert struct{ workDir string }

func (formatConvert) Name() string { return "format_convert" }

func (formatConvert) Description() string {
	return "文档格式转换：将 docx/xlsx/pptx/pdf 文件转换为 Markdown。docx→md 保留标题层级和表格；xlsx→md 生成表格；pptx→md 提取每页文本与备注。PDF 优先经本机 WPS 转换（需已登录 WPS 会员）：文字型 PDF 表格成真表格、图片保留；扫描件走 WPS 在线 OCR（174 页约 1 分钟），照片版 PDF 也能出真表格。WPS 不可用或失败时退回本地提取 / 本地 OCR（本地 OCR 一次最多 30 页，超限报错——用 pages 分段）。preserve_images=true 时保留内嵌图片到 Markdown 同目录的 -assets 子目录并改相对引用（必须同时给 output）；录方案库时建议开启。回执附标题/表格/图片计数，三项全 0 说明产物不可用。耗时数秒到数分钟，取决于页数（233 页经 WPS 约 1 分半）；不消耗主模型 token。"
}

func (formatConvert) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "path":{"type":"string","description":"源文件路径（支持 .docx/.xlsx/.pptx/.pdf）"},
  "output":{"type":"string","description":"输出 Markdown 文件路径（可选，不指定则返回文本）"},
  "pages":{"type":"string","description":"PDF页码范围，如\"1-5\"或\"1,3,5\"（仅PDF有效）"},
  "preserve_images":{"type":"boolean","description":"保留 DOCX/PPTX 内嵌图片：要求同时给 output，图片写入同名 -assets 目录，Markdown 改相对引用。录方案库时建议开启"}
},
"required":["path"]
}`)
}

// ReadOnly is false because the optional output path writes Markdown to disk.
func (formatConvert) ReadOnly() bool { return false }

func (f formatConvert) ReadTarget(args json.RawMessage) (string, bool) {
	var p fcInput
	if err := json.Unmarshal(args, &p); err != nil || strings.TrimSpace(p.Path) == "" {
		return "", false
	}
	return resolveIn(f.workDir, p.Path), true
}

func (f formatConvert) WritePaths(args json.RawMessage) ([]string, error) {
	var p fcInput
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.Output) == "" {
		return nil, nil
	}
	out := resolveIn(f.workDir, p.Output)
	if p.PreserveImages {
		return []string{out, formatConvertAssetDir(out)}, nil
	}
	return []string{out}, nil
}

func (formatConvert) CompactDescription() string     { return compactDesc["format_convert"] }
func (formatConvert) CompactSchema() json.RawMessage { return compactSchema["format_convert"] }

type fcInput struct {
	Path           string `json:"path"`
	Output         string `json:"output,omitempty"`
	Pages          string `json:"pages,omitempty"`
	PreserveImages bool   `json:"preserve_images,omitempty"`
}

// convertPreservingImages 转换 seam（测试打桩；单测不得真调 WPS / OCR）。
var convertPreservingImages = docmd.ConvertPreservingImages

// fcOutcome 一次转换的结果（后台 goroutine 经管道回传）。
type fcOutcome struct {
	md        string
	total     int
	truncated bool
	images    int
	err       error
}

func (f formatConvert) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p fcInput
	if err := json.Unmarshal(args, &p); err != nil {
		return "", codedError("FORMAT_INVALID_ARGS", "参数无效: %v", err)
	}
	if p.Path == "" {
		return "", codedError("FORMAT_INVALID_ARGS", "path 不能为空")
	}
	if p.PreserveImages && strings.TrimSpace(p.Output) == "" {
		return "", codedError("FORMAT_INVALID_ARGS", "preserve_images=true 需要同时给 output 路径")
	}
	p.Path = resolveIn(f.workDir, p.Path)
	if p.Output != "" {
		p.Output = resolveIn(f.workDir, p.Output)
	}

	// 转换链不看取消信号（pdftoppm / wpscli / OCR 引擎请求都是整段同步的）：用户按
	// 停止时立即返回，让本轮收尾——否则 executeBatch 死等这个工具，下一条消息一直
	// 「排队中」（2026-11-26 实弹：174 页扫描件 OCR 堵到重启）。转换在后台跑完即弃，
	// 结果仍进 docmd 的进程内缓存，重试同一份可秒回；落盘在下面，取消后不会写文件。
	ch := make(chan fcOutcome, 1)
	go func() {
		md, total, truncated, images, err := convertPreservingImages(p.Path, p.Pages, docmd.DefaultMaxPDFPages)
		ch <- fcOutcome{md: md, total: total, truncated: truncated, images: images, err: err}
	}()
	var r fcOutcome
	select {
	case r = <-ch:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	md, total, truncated, images, err := r.md, r.total, r.truncated, r.images, r.err
	if err != nil {
		// U1 错误码：模型按 code 路由恢复（换路径/如实告知不支持），不解析散文
		switch {
		case strings.Contains(err.Error(), "不支持的文件格式"):
			return "", codedError("FORMAT_UNSUPPORTED", "%v", err)
		case errors.Is(err, os.ErrNotExist) || os.IsNotExist(err):
			return "", codedError("FORMAT_SOURCE_MISSING", "源文件不存在: %s（%v）", p.Path, err)
		case strings.Contains(err.Error(), "文字层无法解码"):
			// 自写提取只认得到乱码（子集字体缺字体映射），本机 WPS 与 OCR 都没接住：
			// 如实报失败，不把乱码落盘（2026-11-24 实弹：233 页报告转出 17 万替换字符却报成功）。
			return "", codedError("FORMAT_TEXT_UNDECODABLE", "%v", err)
		default:
			return "", codedError("FORMAT_CONVERT_FAILED", "%v", err)
		}
	}
	if truncated {
		md += fmt.Sprintf("\n\n> 转换已截断：PDF 共 %d 页，本次仅处理前 %d 页。可指定 pages 参数分段转换，或使用 summarize_file 获取全文摘要。",
			total, docmd.DefaultMaxPDFPages)
	}
	md = fmt.Sprintf("# 文档转换: %s\n\n%s\n\n---\n*由 gaea format_convert 转换*", filepath.Base(p.Path), md)

	if p.Output != "" {
		// 自动创建输出文件的父目录，与 write_file 行为一致；否则输出路径
		// 指向不存在的目录时 os.WriteFile 会失败，导致“没有生成文件”。
		if dir := filepath.Dir(p.Output); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return "", codedError("FORMAT_OUTPUT_WRITE_FAILED", "创建输出目录失败: %v", err)
			}
		}
		assetDir := ""
		if p.PreserveImages {
			assetDir = formatConvertAssetDir(p.Output)
			var externalized int
			md, externalized, err = docmd.ExternalizeDataImages(md, filepath.Dir(p.Output), assetDir)
			if err != nil {
				return "", codedError("FORMAT_OUTPUT_WRITE_FAILED", "保存文档图片失败: %v", err)
			}
			images = externalized
		}
		if err := os.WriteFile(p.Output, []byte(md), 0o644); err != nil {
			return "", codedError("FORMAT_OUTPUT_WRITE_FAILED", "写入输出文件失败: %v", err)
		}
		// 回执带结构计数：调用方一眼判断产物可用（乱码/空转换的特征是三项全 0）。
		st := docmd.CountStructure(md)
		detail := fmt.Sprintf("已转换并保存为 %s（%d 字）", p.Output, utf8.RuneCountInString(md))
		detail += fmt.Sprintf("；标题 %d / 表格 %d / 图片 %d", st.Headings, st.Tables, st.Images)
		switch {
		case images > 0 && p.PreserveImages:
			detail += fmt.Sprintf("（保留图片 %d 张到 %s）", images, assetDir)
		case images > 0:
			detail += "（图片以 data URI 内嵌在 Markdown 里；要落成图片文件请设 preserve_images=true）"
		case p.PreserveImages && strings.EqualFold(filepath.Ext(p.Path), ".pdf"):
			detail += "（未抽取到图片：该 PDF 没有内嵌图，或本机没有 WPS、只做了纯文本提取）"
		}
		return tool.WrapText(detail), nil
	}
	// 不落盘时图片没法交付：去掉 data URI 再回（几百页的 PDF 经 WPS 转换带 108 张图，
	// 原样回给模型是几 MB 的 base64），并在回执里说明怎么要图片。
	if strings.Contains(md, "data:image/") {
		n := strings.Count(md, "](data:image/")
		md = strings.TrimSpace(docmd.StripDataImages(md)) +
			fmt.Sprintf("\n\n---\n*内嵌图片 %d 张未随文本返回；需要图片请给 output 与 preserve_images=true 落成文件*", n)
	}
	return tool.WrapText(md), nil
}

func formatConvertAssetDir(output string) string {
	ext := filepath.Ext(output)
	stem := strings.TrimSuffix(filepath.Base(output), ext)
	return filepath.Join(filepath.Dir(output), stem+"-assets")
}
