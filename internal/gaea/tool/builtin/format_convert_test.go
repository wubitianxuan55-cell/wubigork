package builtin

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/docmd"
)

// TestFormatConvertWritesOutputFile verifies that format_convert generates the
// output file even when its parent directory does not exist yet.
func TestFormatConvertWritesOutputFile(t *testing.T) {
	docx := writeMinimalDocx(t)
	out := filepath.Join(t.TempDir(), "nested", "deep", "converted.md")

	args, err := json.Marshal(map[string]string{
		"path":   docx,
		"output": out,
	})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	res, err := (formatConvert{}).Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(res, "已转换并保存为") {
		t.Errorf("unexpected result: %s", res)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("output file not generated: %v", err)
	}
	if !strings.HasPrefix(string(data), "# 文档转换:") {
		t.Errorf("unexpected output content: %.120s", data)
	}
}

func TestFormatConvertPreservesDocxImages(t *testing.T) {
	const png = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Y9Wl9sAAAAASUVORK5CYII="
	dir := t.TempDir()
	imagePath := filepath.Join(dir, "figure.png")
	raw, err := base64.StdEncoding.DecodeString(png)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(imagePath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	docx := filepath.Join(dir, "with-image.docx")
	script := `
from docx import Document
from docx.shared import Inches
d = Document()
d.add_heading("第一章 工艺", level=1)
d.add_paragraph("工艺流程图如下：")
d.add_picture(r"` + imagePath + `", width=Inches(0.25))
d.save(r"` + docx + `")
`
	if out, err := exec.Command("python", "-c", script).CombinedOutput(); err != nil {
		t.Skipf("python-docx 不可用，跳过: %v %s", err, out)
	}
	outPath := filepath.Join(dir, "converted", "方案.md")
	args, err := json.Marshal(map[string]any{
		"path": docx, "output": outPath, "preserve_images": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := (formatConvert{}).Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(res, "保留图片 1 张") {
		t.Fatalf("回执未说明图片: %s", res)
	}
	md, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(md), "data:image/") || !strings.Contains(string(md), "方案-assets/") {
		t.Fatalf("Markdown 图片未外置: %.800s", md)
	}
	entries, err := os.ReadDir(filepath.Join(filepath.Dir(outPath), "方案-assets"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("sidecar 图片: %v %+v", err, entries)
	}
}

func TestFormatConvertPreserveImagesWritePaths(t *testing.T) {
	dir := t.TempDir()
	args, err := json.Marshal(map[string]any{
		"path": "source.docx", "output": "exports/方案.md", "preserve_images": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	paths, err := (formatConvert{workDir: dir}).WritePaths(args)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 || paths[1] != filepath.Join(dir, "exports", "方案-assets") {
		t.Fatalf("write paths=%+v", paths)
	}
}

// writeMinimalDocx builds a tiny but structurally valid docx (word/document.xml
// only) so the built-in docx parser has something to read; markitdown being
// present or absent doesn't affect the assertions above.
func writeMinimalDocx(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sample.docx")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	docXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>项目周报</w:t></w:r></w:p>
  </w:body>
</w:document>`
	if _, err := w.Write([]byte(docXML)); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("write docx: %v", err)
	}
	return path
}

// TestFormatConvertNoOutputStripsDataImages 不给 output 时图片没法随文本交付：
// 回执里不能带 data URI（几百页的 PDF 经 WPS 转换有上百张图，原样回是几 MB 的
// base64），并说明怎么要图片。
func TestFormatConvertNoOutputStripsDataImages(t *testing.T) {
	const png = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9Y9Wl9sAAAAASUVORK5CYII="
	dir := t.TempDir()
	imagePath := filepath.Join(dir, "figure.png")
	raw, err := base64.StdEncoding.DecodeString(png)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(imagePath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	docx := filepath.Join(dir, "with-image.docx")
	script := `
from docx import Document
from docx.shared import Inches
d = Document()
d.add_heading("第一章 工艺", level=1)
d.add_paragraph("工艺流程图如下：")
d.add_picture(r"` + imagePath + `", width=Inches(0.25))
d.save(r"` + docx + `")
`
	if out, err := exec.Command("python", "-c", script).CombinedOutput(); err != nil {
		t.Skipf("python-docx 不可用，跳过: %v %s", err, out)
	}
	args, err := json.Marshal(map[string]any{"path": docx})
	if err != nil {
		t.Fatal(err)
	}
	res, err := (formatConvert{}).Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(res, "data:image/") {
		t.Fatalf("不落盘时不得把 data URI 回给模型: %.600s", res)
	}
	if !strings.Contains(res, "工艺流程图如下") || !strings.Contains(res, "内嵌图片 1 张未随文本返回") {
		t.Fatalf("正文应保留并说明图片去向: %.600s", res)
	}
}

// 用户按停止：转换还没完也要立即返回（ctx.Err），让本轮收尾——否则 executeBatch 死等
// 这个工具，下一条消息一直「排队中」（2026-11-26 实弹：174 页扫描件 OCR 堵到重启）。
// 取消后不落盘；后台那次转换跑完即弃。
func TestFormatConvert_CancelReturnsImmediatelyAndWritesNothing(t *testing.T) {
	orig := convertPreservingImages
	release := make(chan struct{})
	started := make(chan struct{})
	convertPreservingImages = func(_, _ string, _ int) (string, int, bool, int, error) {
		close(started)
		<-release
		return "# 慢转换的结果", 1, false, 0, nil
	}
	t.Cleanup(func() { convertPreservingImages = orig })

	out := filepath.Join(t.TempDir(), "converted.md")
	args, err := json.Marshal(map[string]string{"path": "a.pdf", "output": out})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := (formatConvert{}).Execute(ctx, args)
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("转换没有启动")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("取消后应返回 context.Canceled，得到 %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("取消后应立即返回，不该等转换结束")
	}
	close(release)
	// 给后台那次转换一点时间跑完，确认它没有补写文件。
	time.Sleep(200 * time.Millisecond)
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("取消后不得落盘，Stat err=%v", err)
	}
}

// ── to=xlsx：PDF / 图片 → Excel ─────────────────────────────────────

// excelCall 打桩记下的一次 to=xlsx 转换。
type excelCall struct {
	src      string
	opt      docmd.ExcelOptions
	prodUsed bool // save 是否被调用（产物有没有走到落盘这一步）
}

// stubExcelConverter 把 to=xlsx 的转换 seam 换成桩：记下参数，往临时目录写一个假产物并
// 经 save 落盘，再返回 rec；convErr 非空时不产出、直接报错。单测不得真调 WPS。
func stubExcelConverter(t *testing.T, rec docmd.ExcelReceipt, convErr error) *[]excelCall {
	t.Helper()
	calls := &[]excelCall{}
	orig := convertToExcel
	convertToExcel = func(src string, opt docmd.ExcelOptions, save func(string) error) (docmd.ExcelReceipt, error) {
		*calls = append(*calls, excelCall{src: src, opt: opt})
		if convErr != nil {
			return docmd.ExcelReceipt{}, convErr
		}
		prod := filepath.Join(t.TempDir(), "out.xlsx")
		if err := os.WriteFile(prod, []byte("fake xlsx"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := save(prod); err != nil {
			return docmd.ExcelReceipt{}, err
		}
		(*calls)[len(*calls)-1].prodUsed = true
		return rec, nil
	}
	t.Cleanup(func() { convertToExcel = orig })
	return calls
}

// 参数不对时在拉起 WPS 之前就报错，错误码可路由。
func TestFormatConvertToExcel_ArgumentValidation(t *testing.T) {
	calls := stubExcelConverter(t, docmd.ExcelReceipt{Cells: 1}, nil)
	dir := t.TempDir()
	docx := filepath.Join(dir, "x.docx")
	if err := os.WriteFile(docx, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, wantCode, wantMsg string
		args                    map[string]any
	}{
		{"to 取值无效", "FORMAT_INVALID_ARGS", "to 取值无效", map[string]any{"path": "a.pdf", "to": "csv"}},
		{"to=xlsx 必须给 output", "FORMAT_INVALID_ARGS", "需要同时给 output", map[string]any{"path": "a.pdf", "to": "xlsx"}},
		{"output 必须是 .xlsx", "FORMAT_INVALID_ARGS", "必须是 .xlsx", map[string]any{"path": "a.pdf", "to": "xlsx", "output": "a.csv"}},
		{"preserve_images 与 to=xlsx 互斥", "FORMAT_INVALID_ARGS", "preserve_images",
			map[string]any{"path": "a.pdf", "to": "xlsx", "output": "a.xlsx", "preserve_images": true}},
		{"源文件不存在", "FORMAT_SOURCE_MISSING", "源文件不存在", map[string]any{"path": "missing.pdf", "to": "xlsx", "output": "a.xlsx"}},
		{"不支持的输入", "FORMAT_UNSUPPORTED", "to=xlsx 支持", map[string]any{"path": docx, "to": "xlsx", "output": "a.xlsx"}},
	}
	for _, c := range cases {
		args, err := json.Marshal(c.args)
		if err != nil {
			t.Fatal(err)
		}
		res, err := (formatConvert{workDir: dir}).Execute(context.Background(), args)
		if err == nil || !strings.HasPrefix(err.Error(), "Error ["+c.wantCode+"]: ") || !strings.Contains(err.Error(), c.wantMsg) {
			t.Fatalf("%s：应报 [%s] 含「%s」，得到 res=%q err=%v", c.name, c.wantCode, c.wantMsg, res, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "a.xlsx")); !os.IsNotExist(err) {
		t.Fatalf("参数不对时不得产出文件，Stat err=%v", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("参数不对时不应进入转换：%+v", *calls)
	}
}

// 正常路径：参数原样到转换 seam（ai_fix 默认开）、嵌套目录自动创建、产物落盘、回执带
// 结构计数与抽检提示；ai_fix=false、图片目录各自落到参数与回执措辞上。
func TestFormatConvertToExcel_WritesFileAndReceipt(t *testing.T) {
	rec := docmd.ExcelReceipt{Sheets: 2, Rows: 30, MaxCols: 9, Cells: 120, Numbers: 40, NumberText: 12,
		Suspects: []string{"Sheet2!6", "Sheet2!9"}, SuspectCount: 2, Pages: 2, OnlineOCR: true, Elapsed: 83 * time.Second}
	calls := stubExcelConverter(t, rec, nil)
	dir := t.TempDir()
	pdf := filepath.Join(dir, "报价单.pdf")
	if err := os.WriteFile(pdf, []byte("%PDF-1.4"), 0o644); err != nil {
		t.Fatal(err)
	}
	args, err := json.Marshal(map[string]any{
		"path": pdf, "to": "xlsx", "output": "exports/报价单.xlsx", "pages": "3-4", "per_page_sheet": true, "scanned": "true",
	})
	if err != nil {
		t.Fatal(err)
	}
	res, err := (formatConvert{workDir: dir}).Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := filepath.Join(dir, "exports", "报价单.xlsx")
	if data, err := os.ReadFile(out); err != nil || string(data) != "fake xlsx" {
		t.Fatalf("产物没有落到 output（嵌套目录应自动创建）：data=%q err=%v", data, err)
	}
	if len(*calls) != 1 {
		t.Fatalf("转换次数=%d，want 1", len(*calls))
	}
	got := (*calls)[0]
	if got.src != pdf || !got.prodUsed || got.opt != (docmd.ExcelOptions{Pages: "3-4", Scanned: "true", AIFix: true, PerPageSheet: true}) {
		t.Fatalf("转换 seam 收到的参数不对：%+v", got)
	}
	// 回执是 JSON 信封：路径里的反斜杠是转义过的
	for _, want := range []string{
		"exports\\\\报价单.xlsx（2 页，耗时 1m23s）", "工作表 2 个，共 30 行、最宽 9 列", "非空格子 120（数字格 40",
		"另有 12 格数字以文本保存", "经 WPS 在线识别（AI 修复开）", "疑似缺格 2 行（Sheet2!6、Sheet2!9）", "对照原件核对",
	} {
		if !strings.Contains(res, want) {
			t.Fatalf("回执缺 %q：\n%s", want, res)
		}
	}

	// ai_fix=false：显式关掉
	args, _ = json.Marshal(map[string]any{"path": pdf, "to": "xlsx", "output": "b.xlsx", "ai_fix": false})
	if res, err = (formatConvert{workDir: dir}).Execute(context.Background(), args); err != nil {
		t.Fatalf("Execute(ai_fix=false): %v", err)
	}
	if (*calls)[1].opt.AIFix || !strings.Contains(res, "未开 AI 修复") {
		t.Fatalf("ai_fix=false 应关掉 AI 修复并在回执里说明：opt=%+v\n%s", (*calls)[1].opt, res)
	}

	// 图片目录：回执里的「页」是图片张数
	imgs := filepath.Join(dir, "photos")
	for _, n := range []string{"a.jpg", "b.png"} {
		if err := os.MkdirAll(imgs, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(imgs, n), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	args, _ = json.Marshal(map[string]any{"path": imgs, "to": "xlsx", "output": "c.xlsx"})
	if res, err = (formatConvert{workDir: dir}).Execute(context.Background(), args); err != nil {
		t.Fatalf("Execute(图片目录): %v", err)
	}
	if (*calls)[2].src != imgs || !strings.Contains(res, "2 张图片") {
		t.Fatalf("图片目录应原样交给转换、回执按张数说：src=%q\n%s", (*calls)[2].src, res)
	}
}

// 用户按停止：立即返回，产物不落盘（落盘在 save 里、先看取消信号）。
func TestFormatConvertToExcel_CancelReturnsImmediatelyAndWritesNothing(t *testing.T) {
	orig := convertToExcel
	release := make(chan struct{})
	started := make(chan struct{})
	convertToExcel = func(_ string, _ docmd.ExcelOptions, save func(string) error) (docmd.ExcelReceipt, error) {
		close(started)
		<-release
		prod := filepath.Join(t.TempDir(), "out.xlsx")
		if err := os.WriteFile(prod, []byte("late"), 0o644); err != nil {
			return docmd.ExcelReceipt{}, err
		}
		if err := save(prod); err != nil {
			return docmd.ExcelReceipt{}, err
		}
		return docmd.ExcelReceipt{Cells: 1}, nil
	}
	t.Cleanup(func() { convertToExcel = orig })

	dir := t.TempDir()
	pdf := filepath.Join(dir, "a.pdf")
	if err := os.WriteFile(pdf, []byte("%PDF-1.4"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "清单.xlsx")
	args, err := json.Marshal(map[string]string{"path": pdf, "to": "xlsx", "output": out})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := (formatConvert{}).Execute(ctx, args)
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("转换没有启动")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("取消后应返回 context.Canceled，得到 %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("取消后应立即返回，不该等转换结束")
	}
	close(release)
	time.Sleep(200 * time.Millisecond)
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("取消后不得落盘，Stat err=%v", err)
	}
}

// 转换失败、产物写不进目标位置：各自的错误码；目标位置是个目录时模拟「被占用」。
func TestFormatConvertToExcel_ErrorCodes(t *testing.T) {
	dir := t.TempDir()
	pdf := filepath.Join(dir, "a.pdf")
	if err := os.WriteFile(pdf, []byte("%PDF-1.4"), 0o644); err != nil {
		t.Fatal(err)
	}
	args, err := json.Marshal(map[string]string{"path": pdf, "to": "xlsx", "output": filepath.Join(dir, "清单.xlsx")})
	if err != nil {
		t.Fatal(err)
	}
	stubExcelConverter(t, docmd.ExcelReceipt{}, errors.New("本机未找到 WPS（wpscli）：转 Excel 需要 WPS Office（登录会员）"))
	if _, err := (formatConvert{}).Execute(context.Background(), args); err == nil ||
		!strings.HasPrefix(err.Error(), "Error [FORMAT_CONVERT_FAILED]: ") || !strings.Contains(err.Error(), "未找到 WPS") {
		t.Fatalf("转换失败应透传原因，得到 err=%v", err)
	}

	// 产物写不进目标位置：目标被一个同名目录占着，模拟「文件被 Excel / WPS 占用」
	origExcel := convertToExcel
	convertToExcel = func(_ string, _ docmd.ExcelOptions, save func(string) error) (docmd.ExcelReceipt, error) {
		prod := filepath.Join(t.TempDir(), "out.xlsx")
		if err := os.WriteFile(prod, []byte("x"), 0o644); err != nil {
			return docmd.ExcelReceipt{}, err
		}
		if err := save(prod); err != nil {
			return docmd.ExcelReceipt{}, err
		}
		return docmd.ExcelReceipt{Cells: 1}, nil
	}
	t.Cleanup(func() { convertToExcel = origExcel })
	locked := filepath.Join(dir, "locked.xlsx")
	if err := os.Mkdir(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	args, _ = json.Marshal(map[string]string{"path": pdf, "to": "xlsx", "output": locked})
	if _, err := (formatConvert{}).Execute(context.Background(), args); err == nil ||
		!strings.HasPrefix(err.Error(), "Error [FORMAT_OUTPUT_WRITE_FAILED]: ") || !strings.Contains(err.Error(), "先关闭它") {
		t.Fatalf("产物写不进目标位置应报输出写入失败并给出处置建议，得到 err=%v", err)
	}
}

func TestFormatConvertToExcel_WritePaths(t *testing.T) {
	dir := t.TempDir()
	args, err := json.Marshal(map[string]any{
		"path": "清单.pdf", "to": "xlsx", "output": "exports/清单.xlsx", "preserve_images": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	paths, err := (formatConvert{workDir: dir}).WritePaths(args)
	if err != nil {
		t.Fatal(err)
	}
	// Excel 产物只有一个文件：即使误传 preserve_images 也不登记图片 sidecar
	if len(paths) != 1 || paths[0] != filepath.Join(dir, "exports", "清单.xlsx") {
		t.Fatalf("write paths=%+v", paths)
	}
}
