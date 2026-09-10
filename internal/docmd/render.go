package docmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gaea/gaea/internal/gaea/proc"
)

// RenderPDFPages 用 poppler pdftoppm 把 PDF 逐页渲染为 PNG（v4.6 Verifier
// 通道 B 真视觉 diff 的渲染侧）。返回按页序排列的 PNG 路径（页码由 pdftoppm
// 文件名后缀推导，避免目录读取顺序依赖）。
//
// dpi<=0 时用默认 100（视觉 diff 用低分辨率即可，像素对比按归一化网格采样，
// 高分辨率只增加耗时不增加判定精度）。pdftoppm 不可用返回错误，调用方按
// 「渲染降级 warn」处理（与 OCR 路径同一探测源 findPdftoppm）。
func RenderPDFPages(pdfPath, prefix string, dpi int) ([]string, error) {
	pdftoppmPath := findPdftoppm()
	if pdftoppmPath == "" {
		return nil, fmt.Errorf("视觉 diff 需要 poppler 渲染（pdftoppm），但未找到。" +
			"请安装 poppler：https://poppler.freedesktop.org")
	}
	if dpi <= 0 {
		dpi = 100
	}
	args := []string{"-png", "-r", strconv.Itoa(dpi), pdfPath, prefix}
	cmd := exec.Command(pdftoppmPath, args...)
	proc.HideWindow(cmd) // Windows: 防止弹出 cmd 黑框
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("pdftoppm 执行失败: %w\n输出: %s", err, string(out))
	}

	dir := filepath.Dir(prefix)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取渲染目录失败: %w", err)
	}
	type pageFile struct {
		num int
		abs string
	}
	var pages []pageFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".png") {
			continue
		}
		name := strings.ToLower(e.Name())
		// pdftoppm 输出形如 <prefix>-1.png / -10.png；从尾部数字段解析页码。
		stem := strings.TrimSuffix(name, ".png")
		idx := strings.LastIndexByte(stem, '-')
		if idx < 0 || idx == len(stem)-1 {
			continue
		}
		if n, perr := strconv.Atoi(stem[idx+1:]); perr == nil && n > 0 {
			pages = append(pages, pageFile{num: n, abs: filepath.Join(dir, e.Name())})
		}
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("pdftoppm 未生成页面图片")
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].num < pages[j].num })
	out := make([]string, 0, len(pages))
	for _, p := range pages {
		out = append(out, p.abs)
	}
	return out, nil
}
