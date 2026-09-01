package app

// v4.28 B2「pptx 最小交互」——pptx 的「读」侧第一刀：
//   1. GaeaPptxOutline：python-pptx 原生解析 slides/shapes/text_frame 生成
//      结构化大纲（零像素、毫秒级），供前端大纲卡与「针对第 N 页修改」指令；
//   2. previewPptx：soffice→PDF 懒渲染 + poppler 逐页缩略（PNG 落盘缓存），
//      供 GaeaPreview 的 .pptx 分支（gaea_preview.go）复用。
// 错误一律结构化（Available=false / Kind="error"），绝不 panic——pptx 预览是
// 增强能力，任何一环缺失（python / python-pptx / soffice / poppler）都降级
// 而非中断。真编辑 pptx（python-pptx 写回）为远期项。

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/proc"
)

// PptxSlideOutline 是单页大纲：Index 为 1-based 页码（与逐页预览的页锚点、
// 「针对第 N 页修改」指令里的页码一致）。
type PptxSlideOutline struct {
	Index      int      `json:"index"`
	Title      string   `json:"title"`
	Texts      []string `json:"texts"`     // 正文文本框（标题除外，单条截断）
	ShapeCount int      `json:"shapeCount"` // 该页 shape 总数（含图片等非文本）
}

// PptxOutlineView 是 GaeaPptxOutline 的返回：失败结构化（Available=false +
// Error），前端降级为纯逐页预览 + 诚实提示。
type PptxOutlineView struct {
	Available bool               `json:"available"`
	Error     string             `json:"error,omitempty"`
	Slides    []PptxSlideOutline `json:"slides"`
}

const (
	// pptxOutlineTextLimit 每条正文文本的字符上限（按 rune 截断，附省略号），
	// 防大文本框把大纲负载撑爆。
	pptxOutlineTextLimit = 200
	// pptxOutlineTextsMax 每页收录的正文文本框条数上限（防异常文件万级
	// 文本框卡死前端渲染；超出丢弃不报错）。
	pptxOutlineTextsMax = 40
	// pptxThumbDPI 逐页缩略渲染 DPI：只做「看得见版式」的缩略，低分辨率省负载。
	pptxThumbDPI = 64
	// pptxMaxPreviewPages 逐页缩略页数上限：演示文稿场景通常远小于此；
	// 超出部分不缩略（Truncated=true 由前端明示），完整内容仍可外部打开。
	pptxMaxPreviewPages = 60
	// pptxCacheTTL 缓存清理阈值：超过未再命中的缓存产物直接删除（防
	// .gaea/cache/pptx-preview 随文件版本无界增长）。
	pptxCacheTTL = 7 * 24 * time.Hour
	// pptxOutlineHint 是 PreviewResult.Hint 的取值：提示前端可另行拉取大纲卡。
	pptxOutlineHint = "outline"
	// pptxOutlineTimeout 大纲解析超时（秒）：python-pptx 原生读远快于此，
	// 防御异常文件把绑定调用挂死。
	pptxOutlineTimeout = 60
)

// GaeaPptxOutline 读取 pptx 结构化大纲（python-pptx，零渲染）。路径解析与
// GaeaPreview 同款（resolvePreviewPath：裸文件名回退常见输出目录，Join/Clean
// 防穿越）；python 调用镜像 exportPptx 的既有方式（PATH 上的 python + 超时 +
// stderr 透出，见 runProcess/runPython），脚本以临时 .py 落盘注入（避免 -c
// 的命令行长度/引号陷阱），结果 JSON 走 stdout。
func (a *App) GaeaPptxOutline(rel string) PptxOutlineView {
	view := PptxOutlineView{Slides: []PptxSlideOutline{}}
	path, _ := resolvePreviewPath(rel)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		view.Error = "文件不存在"
		return view
	}
	if ext := strings.ToLower(filepath.Ext(path)); ext != ".pptx" {
		view.Error = fmt.Sprintf("仅支持 .pptx 大纲解析（收到 %s；.ppt 旧格式请先另存为 .pptx）", ext)
		return view
	}
	scriptPath, cleanup, err := writePptxOutlineScript()
	if err != nil {
		view.Error = "大纲脚本落盘失败: " + err.Error()
		return view
	}
	defer cleanup()
	out, err := runPythonOut([]string{scriptPath, path}, pptxOutlineTimeout)
	if err != nil {
		view.Error = "大纲解析失败: " + err.Error()
		return view
	}
	return parsePptxOutline(out)
}

// pptxOutlinePy 内置大纲脚本：stdout 输出 JSON（默认 ensure_ascii，纯 ASCII
// 管道输出，规避 Windows 控制台编码差异）；python-pptx 缺失/文件打不开也走
// JSON 结构化错误（退出码 0），只有解释器级故障才非零退出 + stderr（由
// runPythonOut 透出）。标题取标题占位符，正文取其余 text_frame 文本框。
const pptxOutlinePy = `import json
import sys

try:
    from pptx import Presentation
except Exception as e:
    print(json.dumps({"error": "python-pptx 不可用（%s）；可执行 pip install python-pptx 后重试" % e}))
    sys.exit(0)

try:
    prs = Presentation(sys.argv[1])
except Exception as e:
    print(json.dumps({"error": "无法打开 pptx：%s" % e}))
    sys.exit(0)

slides = []
for i, slide in enumerate(prs.slides, start=1):
    title = ""
    try:
        t = slide.shapes.title
        if t is not None and t.has_text_frame:
            title = (t.text or "").strip()
    except Exception:
        title = ""
    texts = []
    shapes = 0
    seen_text = False
    for sh in slide.shapes:
        shapes += 1
        try:
            if not sh.has_text_frame:
                continue
            txt = "\n".join(p.text for p in sh.text_frame.paragraphs if p.text).strip()
        except Exception:
            continue
        if not txt:
            continue
        # 无标题占位符的版式（gaea create_pptx.py 用空白版式 + 文本框，标题
        # 即首个文本框）：首个单行短文本框视作标题，不再重复计入正文。
        if not title and not seen_text and "\n" not in txt and len(txt) <= 80:
            title = txt
            seen_text = True
            continue
        seen_text = True
        if txt == title:
            continue
        texts.append(txt)
    slides.append({"index": i, "title": title, "texts": texts, "shapeCount": shapes})

print(json.dumps({"slides": slides}))
`

// writePptxOutlineScript 把内置脚本写到系统临时目录（exportPptx 的技能脚本
// 走 .gaea/skills 定位；大纲脚本随二进制内置，无需用户侧安装）。
func writePptxOutlineScript() (path string, cleanup func(), err error) {
	tmp, err := os.CreateTemp("", "gaea-pptx-outline-*.py")
	if err != nil {
		return "", nil, err
	}
	if _, werr := tmp.WriteString(pptxOutlinePy); werr != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", nil, werr
	}
	if cerr := tmp.Close(); cerr != nil {
		os.Remove(tmp.Name())
		return "", nil, cerr
	}
	return tmp.Name(), func() { os.Remove(tmp.Name()) }, nil
}

// runPythonOut 是 runPython 的带 stdout 版本（gaea_export.go runProcess 同款
// 口径：PATH 上的 python 解释器、超时杀进程、stderr 摘要透出、Windows 隐藏
// 窗口）；stdout 供脚本回传 JSON。
func runPythonOut(args []string, timeoutSec int) (string, error) {
	cmd := exec.Command("python", args...)
	proc.HideWindow(cmd) // Windows: 防止弹出 cmd 黑框
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		return "", err
	}
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				return "", fmt.Errorf("%v（%s）", err, truncateStr(stderr.String(), 1500))
			}
			return "", err
		}
		return stdout.String(), nil
	case <-time.After(time.Duration(timeoutSec) * time.Second):
		cmd.Process.Kill()
		<-done
		return "", fmt.Errorf("超时（%d 秒）", timeoutSec)
	}
}

// parsePptxOutline 解析脚本 stdout JSON（{"error"} 或 {"slides":[...]}）：
// 纯解析层，无 IO，可独立单测（CI 无 python 也可锁行为）。文本按 rune 截断、
// 条数封顶、剔除空白条；Texts 恒非 nil（JSON null 会硌前端）。
func parsePptxOutline(out string) PptxOutlineView {
	view := PptxOutlineView{Slides: []PptxSlideOutline{}}
	var raw struct {
		Error  string             `json:"error"`
		Slides []PptxSlideOutline `json:"slides"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &raw); err != nil {
		view.Error = "大纲输出解析失败: " + truncateStr(err.Error(), 200)
		return view
	}
	if raw.Error != "" {
		view.Error = raw.Error
		return view
	}
	for _, s := range raw.Slides {
		texts := make([]string, 0, len(s.Texts))
		for i, t := range s.Texts {
			if i >= pptxOutlineTextsMax {
				break
			}
			t = strings.TrimSpace(t)
			if t == "" {
				continue
			}
			texts = append(texts, truncateRunesEllipsis(t, pptxOutlineTextLimit))
		}
		view.Slides = append(view.Slides, PptxSlideOutline{
			Index:      s.Index,
			Title:      strings.TrimSpace(s.Title),
			Texts:      texts,
			ShapeCount: s.ShapeCount,
		})
	}
	view.Available = true
	return view
}

// truncateRunesEllipsis 按 rune 截断（UTF-8 安全），截断处附省略号。
func truncateRunesEllipsis(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// ── .pptx 预览：soffice→PDF（缓存）+ poppler 逐页缩略 ────────────────────

// PreviewPage 是 pptx 逐页预览的单页缩略（Page 为 1-based 页码，与大纲
// Index 同序；DataURL 为该页 PNG）。
type PreviewPage struct {
	Page    int    `json:"page"`
	DataURL string `json:"dataUrl"`
}

// previewPptx 是 GaeaPreview 的 .pptx 分支（gaea_preview.go 转接进来）：
//   soffice→PDF（verifyConvertToPdf seam，产物落 .gaea/cache/pptx-preview/
//   <hash>.pdf，二次预览命中缓存）→ poppler pdftoppm 低 DPI 逐页 PNG（同目录
//   <hash>-pages/，同样缓存）→ Pages 随 PreviewResult 回传，前端纵向铺页 +
//   大纲卡页锚点滚动。pdftoppm 不可用/渲染失败时回退整本 PDF dataUrl（交给
//   WebView 内嵌 PDF 查看器）；soffice 不可用/转换失败 → Kind=error 带原因。
//   Hint="outline" 提示前端另行拉取 GaeaPptxOutline 大纲卡。
func previewPptx(path string, info os.FileInfo, base PreviewResult) PreviewResult {
	base.Kind = "pdf"
	base.Hint = pptxOutlineHint
	cacheDir := pptxPreviewCacheDir()
	sweepPptxPreviewCache(cacheDir)
	key := pptxPreviewCacheKey(path, info)
	pdfPath := filepath.Join(cacheDir, key+".pdf")
	if !fileExists(pdfPath) {
		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			base.Kind = "error"
			base.Error = err.Error()
			return base
		}
		// 复用 Verifier 通道 B 的转换 seam（soffice 无头；包级变量可注入，
		// 测试不依赖真实 LibreOffice）
		if err := verifyConvertToPdf(path, pdfPath); err != nil {
			base.Kind = "error"
			base.Error = err.Error()
			return base
		}
	}
	pages, total, perr := pptxPageThumbs(pdfPath, filepath.Join(cacheDir, key+"-pages"))
	if perr == nil && len(pages) > 0 {
		base.Pages = pages
		base.TotalPages = total
		base.Truncated = total > len(pages)
		return base
	}
	// 回退：逐页缩略不可得（无 poppler / 渲染失败）→ 整本 PDF dataUrl，
	// 与 docx 全量 dataUrl 同口径（WebView 内嵌 PDF 查看器渲染，无页锚点）。
	b, err := os.ReadFile(pdfPath)
	if err != nil {
		base.Kind = "error"
		base.Error = err.Error()
		return base
	}
	base.DataURL = "data:application/pdf;base64," + base64.StdEncoding.EncodeToString(b)
	return base
}

// pptxPreviewCacheDir pptx 预览缓存根目录（.gaea/cache/pptx-preview，与
// .gaea/cache/identity.hash 同属运行时缓存区，不进 exports 产物区）。
func pptxPreviewCacheDir() string {
	return filepath.Join(gaeaCwd(), ".gaea", "cache", "pptx-preview")
}

// pptxPreviewCacheKey 由「绝对路径+大小+mtime(纳秒)」生成缓存键：同文件二次
// 预览直接命中；文件一变（大小或修改时间）立即失配重转。
func pptxPreviewCacheKey(path string, info os.FileInfo) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d\x00%d",
		strings.ToLower(filepath.ToSlash(path)), info.Size(), info.ModTime().UnixNano())))
	return hex.EncodeToString(sum[:])[:16]
}

// pptxPageThumbs 渲染（或复用缓存目录中的）逐页 PNG 缩略。返回缩略页（按页
// 序、封顶 pptxMaxPreviewPages）与实际页数（供 Truncated 判定）。pdftoppm
// 一次渲染全本，页数上限只裁回传负载（超大 deck 多渲染的部分仅弃用）。
func pptxPageThumbs(pdfPath, pagesDir string) ([]PreviewPage, int, error) {
	pages, err := readPptxPageThumbs(pagesDir)
	if err != nil {
		return nil, 0, err
	}
	if len(pages) == 0 {
		if err := os.MkdirAll(pagesDir, 0o755); err != nil {
			return nil, 0, err
		}
		// verifyRenderPages = docmd.RenderPDFPages（poppler pdftoppm，通道 B
		// 渲染 seam）：不可用时返回错误 → 调用方回退整本 PDF dataUrl
		if _, err := verifyRenderPages(pdfPath, filepath.Join(pagesDir, "p"), pptxThumbDPI); err != nil {
			return nil, 0, err
		}
		pages, err = readPptxPageThumbs(pagesDir)
		if err != nil {
			return nil, 0, err
		}
	}
	total := len(pages)
	if total > pptxMaxPreviewPages {
		pages = pages[:pptxMaxPreviewPages]
	}
	return pages, total, nil
}

// readPptxPageThumbs 读取缓存目录里的逐页 PNG（p-<n>.png，pdftoppm 命名），
// 按页序升序组装 dataUrl；目录不存在/无页图返回空切片（触发懒渲染）。
func readPptxPageThumbs(pagesDir string) ([]PreviewPage, error) {
	entries, err := os.ReadDir(pagesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	type pageFile struct {
		num  int
		path string
	}
	var pages []pageFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".png") {
			continue
		}
		stem := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		idx := strings.LastIndexByte(stem, '-')
		if idx < 0 || idx == len(stem)-1 {
			continue
		}
		n, perr := strconv.Atoi(stem[idx+1:])
		if perr != nil || n <= 0 {
			continue
		}
		pages = append(pages, pageFile{num: n, path: filepath.Join(pagesDir, e.Name())})
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].num < pages[j].num })
	out := make([]PreviewPage, 0, len(pages))
	for _, p := range pages {
		b, rerr := os.ReadFile(p.path)
		if rerr != nil {
			return nil, rerr
		}
		out = append(out, PreviewPage{Page: p.num, DataURL: pngDataURL(b)})
	}
	return out, nil
}

// pngDataURL 把 PNG 字节包成 dataUrl（<img src> 直接可用）。
func pngDataURL(b []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b)
}

// sweepPptxPreviewCache 清理超过 TTL 未命中的缓存产物（含页图子目录，尽力
// 而为，错误静默）——防缓存目录随文件版本无界增长。
func sweepPptxPreviewCache(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-pptxCacheTTL)
	for _, e := range entries {
		info, ierr := e.Info()
		if ierr != nil || info.ModTime().After(cutoff) {
			continue
		}
		p := filepath.Join(dir, e.Name())
		if e.IsDir() {
			_ = os.RemoveAll(p)
			continue
		}
		_ = os.Remove(p)
	}
}
