package app

import (
	"fmt"
	"github.com/gaea/gaea/internal/docmd"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/wspath"
)

// ── PDF 导出：LibreOffice 无头转换 ──────────────────────────
// soffice --headless --convert-to pdf 与 recalc.py 共用同一 LibreOffice
// 依赖底座；转换使用独立 UserInstallation profile，避免与重算/用户正在
// 打开的 LibreOffice 实例发生 profile 锁冲突。

// ConvertPdfResult 是转换结果（PDF 落 .gaea/exports/，工作区相对路径）。
type ConvertPdfResult struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Source string `json:"source"` // 源文件相对路径
}

// pdfDirectExts 是 LibreOffice 原生可读、可直接转 PDF 的扩展名。
// md/markdown 不在其列（Writer 不读 Markdown），走 docx 中转链。
var pdfDirectExts = map[string]bool{
	".docx": true, ".xlsx": true, ".pptx": true, ".odt": true,
	".html": true, ".htm": true, ".txt": true, ".csv": true,
}

// GaeaConvertToPdf 把工作区文档转换为 PDF（LibreOffice 无头转换），
// PDF 落 .gaea/exports/ 并返回相对路径。md/markdown 先经 create_docx.py
// 出 docx 再转 PDF（「任何文档 → PDF」统一出口）。
func (a *App) GaeaConvertToPdf(rel string) (ConvertPdfResult, error) {
	if rel == "" {
		return ConvertPdfResult{}, fmt.Errorf("缺少文件路径")
	}
	path := rel
	if !filepath.IsAbs(rel) {
		// v4.363：相对路径拒 .. 穿越；审计 2026-10-02 AP5-09：判据改调唯一原语
		// wspath.ResolveRelWithin（Clean + 包含性检查，与全仓同类站点同源）。
		p, err := wspath.ResolveRelWithin(gaeaCwd(), rel)
		if err != nil {
			return ConvertPdfResult{}, fmt.Errorf("非法工作区相对路径: %s", rel)
		}
		path = p
	}
	// v4.363：读侧收口——本绑定是「读任意路径喂外部转换进程」的原语，此前
	// 绝对路径直通无约束（v4.358 审计挂池）。统一门 = 工作区/数据根白名单
	// ∪ 文件对话框登记（附件 chip 的工作区外素材正是经 GaeaPickFiles 进来、
	// 登记时已入表），fail-closed 对齐 GaeaReadFileB64。写侧 exports 目录
	// 与文件名全服务端拼装，无越界面。
	if !a.withinReadRoots(path) && !isPickedFile(path) {
		return ConvertPdfResult{}, fmt.Errorf("文件不在可读范围（工作区/数据根），且未经文件对话框选取: %s", rel)
	}
	if _, err := os.Stat(path); err != nil {
		return ConvertPdfResult{}, fmt.Errorf("文件不存在：%s", rel)
	}
	ext := strings.ToLower(filepath.Ext(path))

	src := path
	tmpDir := ""
	switch ext {
	case ".md", ".markdown":
		// Markdown → docx（复用交付模板链路）→ PDF
		raw, err := os.ReadFile(path)
		if err != nil {
			return ConvertPdfResult{}, err
		}
		var err2 error
		tmpDir, err2 = os.MkdirTemp("", "gaea-pdf-*")
		if err2 != nil {
			return ConvertPdfResult{}, err2
		}
		base := safeDeliverableName(strings.TrimSuffix(filepath.Base(path), ext))
		docxPath := filepath.Join(tmpDir, base+".docx")
		in := ExportDeliverableInput{
			Markdown: string(raw),
			Format:   "docx",
			Title:    base,
			Template: "通用",
			Footer:   "第 {page} 页",
		}
		if err := exportDocx(in, base, docxPath); err != nil {
			_ = os.RemoveAll(tmpDir)
			return ConvertPdfResult{}, err
		}
		src = docxPath
	case ".doc":
		return ConvertPdfResult{}, fmt.Errorf("暂不支持 .doc 转 PDF（请先另存为 .docx）")
	default:
		if !pdfDirectExts[ext] {
			return ConvertPdfResult{}, fmt.Errorf(
				"暂不支持将 %s 转换为 PDF（支持 docx/xlsx/pptx/odt/html/txt/csv/md）", ext)
		}
	}
	if tmpDir != "" {
		defer func() { _ = os.RemoveAll(tmpDir) }()
	}

	// S4 产物路径分区：work 恒 .gaea/exports（现状不动），play 落 .gaea/play/exports。
	exportsDir := spaces.ExportsDir(gaeaCwd(), gaeaEffectiveSpace())
	if err := os.MkdirAll(exportsDir, 0o755); err != nil {
		return ConvertPdfResult{}, err
	}
	srcBase := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	outPath := filepath.Join(exportsDir, safeDeliverableName(srcBase)+"-"+time.Now().Format("20060102-150405")+".pdf")
	if err := convertToPdfFile(src, outPath); err != nil {
		return ConvertPdfResult{}, err
	}

	info, err := os.Stat(outPath)
	if err != nil {
		return ConvertPdfResult{}, err
	}
	outRel, _ := filepath.Rel(gaeaCwd(), outPath)
	srcRel, _ := filepath.Rel(gaeaCwd(), path)
	return ConvertPdfResult{
		Path:   filepath.ToSlash(outRel),
		Name:   filepath.Base(outPath),
		Size:   info.Size(),
		Source: filepath.ToSlash(srcRel),
	}, nil
}

// markdownToPdf 受控 Markdown → docx（临时）→ soffice PDF，
// 供 GaeaExportDeliverable 的 pdf 格式复用交付模板链路。
func markdownToPdf(in ExportDeliverableInput, title, outPath string) error {
	tmpDir, err := os.MkdirTemp("", "gaea-pdf-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()
	docxPath := filepath.Join(tmpDir, safeDeliverableName(title)+".docx")
	if err := exportDocx(in, title, docxPath); err != nil {
		return err
	}
	return convertToPdfFile(docxPath, outPath)
}

// F2 蒸馏刀：四原语单源搬家至 docmd（doc_render 只读渲染工具与导出共用），
// 此处保留同名薄委托——调用面（gaea_export 的 python 进程等）零改动。
func convertToPdfFile(src, outPath string) error { return docmd.ConvertToPdfFile(src, outPath) }

func findSoffice() string { return docmd.FindSoffice() }

func runProcess(exe string, args []string, timeoutSec int) error {
	return docmd.RunProcess(exe, args, timeoutSec)
}
