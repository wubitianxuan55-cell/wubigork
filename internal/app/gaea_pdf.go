package app

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/proc"
	"github.com/gaea/gaea/internal/gaea/spaces"
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
		path = filepath.Join(gaeaCwd(), rel)
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
			os.RemoveAll(tmpDir)
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
		defer os.RemoveAll(tmpDir)
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
	defer os.RemoveAll(tmpDir)
	docxPath := filepath.Join(tmpDir, safeDeliverableName(title)+".docx")
	if err := exportDocx(in, title, docxPath); err != nil {
		return err
	}
	return convertToPdfFile(docxPath, outPath)
}

// convertToPdfFile 用 soffice 把单个文档转成 outPath 指定的 PDF。
// soffice 以源文件名命名输出，所以先转进临时目录再搬到 outPath
//（临时目录与工作区可能跨盘，rename 失败时回退复制）。
func convertToPdfFile(src, outPath string) error {
	soffice := findSoffice()
	if soffice == "" {
		return fmt.Errorf("未找到 LibreOffice（soffice），请安装 LibreOffice 后重试")
	}
	tmpOut, err := os.MkdirTemp("", "gaea-pdf-out-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpOut)
	profile, err := os.MkdirTemp("", "gaea-soffice-profile-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(profile)
	// 独立 profile：与 recalc/用户已开的 LibreOffice 互不抢锁
	profileURL := "file:///" + strings.ReplaceAll(filepath.ToSlash(profile), " ", "%20")

	args := []string{
		"-env:UserInstallation=" + profileURL,
		"--headless", "--norestore", "--nolockcheck", "--nologo",
		"--convert-to", "pdf", "--outdir", tmpOut, src,
	}
	if err := runProcess(soffice, args, 180); err != nil {
		return fmt.Errorf("PDF 转换失败: %w", err)
	}
	produced := filepath.Join(tmpOut, strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))+".pdf")
	if _, err := os.Stat(produced); err != nil {
		return fmt.Errorf("PDF 未生成（源文件可能无法被 LibreOffice 打开）")
	}
	return moveOrCopyFile(produced, outPath)
}

// findSoffice 定位 soffice：PATH 优先，再补常见安装位置
//（Windows 下 PATH 通常没有 soffice）。
func findSoffice() string {
	if p, err := exec.LookPath("soffice"); err == nil {
		return p
	}
	var cands []string
	switch runtime.GOOS {
	case "windows":
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			cands = append(cands, filepath.Join(local, "Programs", "LibreOffice", "program", "soffice.exe"))
		}
		for _, pf := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)")} {
			if pf != "" {
				cands = append(cands, filepath.Join(pf, "LibreOffice", "program", "soffice.exe"))
			}
		}
	case "darwin":
		cands = append(cands, "/Applications/LibreOffice.app/Contents/MacOS/soffice")
	default:
		cands = append(cands, "/usr/lib/libreoffice/program/soffice", "/opt/libreoffice/program/soffice")
	}
	for _, c := range cands {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c
		}
	}
	return ""
}

// moveOrCopyFile 搬运文件：同盘 rename，跨盘回退复制（临时目录可能在
// 系统盘而工作区在数据盘）。
func moveOrCopyFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// runProcess 带超时地运行外部命令（stderr 合并进错误信息）。
// runPython 的通用化版本，供 soffice 等非 python 外部工具复用。
func runProcess(exe string, args []string, timeoutSec int) error {
	cmd := exec.Command(exe, args...)
	proc.HideWindow(cmd) // Windows: 防止弹出 cmd 黑框
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				return fmt.Errorf("%v（%s）", err, truncateStr(stderr.String(), 1500))
			}
			return err
		}
		return nil
	case <-time.After(time.Duration(timeoutSec) * time.Second):
		cmd.Process.Kill()
		<-done
		return fmt.Errorf("超时（%d 秒）", timeoutSec)
	}
}
