package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/docmd"
)

// stubConvert 给转换 seam 打桩：按页规格返回可预测的正文，并记下每次调用的规格。
func stubConvert(t *testing.T, total int) (calls *[]string) {
	t.Helper()
	orig := convertPreservingImagesReport
	var seen []string
	calls = &seen
	convertPreservingImagesReport = func(_, pages string, maxPages int) (string, int, bool, int, *docmd.DocxRebuildReport, error) {
		seen = append(seen, pages)
		eff, truncated, err := capForTest(pages, maxPages, total)
		if err != nil {
			return "", total, false, 0, nil, err
		}
		return "正文[" + eff + "]", total, truncated, 0, nil, nil
	}
	t.Cleanup(func() { convertPreservingImagesReport = orig })
	return calls
}

// capForTest 与 docmd.capPageSpec 同口径的桩内复刻（空规格超限 → 前 maxPages 页）。
func capForTest(pages string, maxPages, total int) (string, bool, error) {
	if pages == "" {
		if total > maxPages {
			return "1-500", true, nil
		}
		return "", false, nil
	}
	return pages, false, nil
}

func pdfPath(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "包1技术标V5.pdf")
	if err := os.WriteFile(p, []byte("%PDF-1.4\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// 2026-11-27 AI 反馈 #6：676 页扫描件只转了前 500 页，提示只在 md 尾部夹一行小字、
// 回执只报成功与字数。现在回执与文件头尾都醒目标出缺了哪几页与续转办法。
func TestFormatConvert_TruncationIsLoud(t *testing.T) {
	stubConvert(t, 676)
	out := filepath.Join(t.TempDir(), "out", "book.md")
	args, _ := json.Marshal(map[string]string{"path": pdfPath(t), "output": out})
	res, err := (formatConvert{}).Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("转换应成功: %v", err)
	}
	for _, want := range []string{"页数：共 676 页", "只转换了第 1–500 页，缺第 501–676 页", "pages=501-676", "all_pages=true"} {
		if !strings.Contains(res, want) {
			t.Errorf("回执应含 %q：%s", want, res)
		}
	}
	data, _ := os.ReadFile(out)
	md := string(data)
	head := md[:min(len(md), 600)]
	tail := md[max(0, len(md)-500):]
	if !strings.Contains(head, "本文档不完整") || !strings.Contains(head, "第 501–676 页缺失") {
		t.Errorf("文件头应有醒目的不完整告警：%s", head)
	}
	if !strings.Contains(tail, "[!WARNING]") || !strings.Contains(tail, "转换在第 500 页之后截断") {
		t.Errorf("文件尾（截断处）应有告警块：%s", tail)
	}
}

// pages=501-676 这类后半段区间照常转换，回执写明本次区间。
func TestFormatConvert_LaterPageRange(t *testing.T) {
	calls := stubConvert(t, 676)
	out := filepath.Join(t.TempDir(), "part2.md")
	args, _ := json.Marshal(map[string]string{"path": pdfPath(t), "output": out, "pages": "501-676"})
	res, err := (formatConvert{}).Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("后半段区间应可转换: %v", err)
	}
	if len(*calls) != 1 || (*calls)[0] != "501-676" {
		t.Fatalf("应原样把区间交给转换链：%v", *calls)
	}
	if !strings.Contains(res, "页数：共 676 页，本次 pages=501-676") {
		t.Errorf("回执应写明本次区间：%s", res)
	}
	if data, _ := os.ReadFile(out); strings.Contains(string(data), "本文档不完整") {
		t.Error("显式区间不是截断，文件里不该有不完整告警")
	}
}

// all_pages=true：超过单次上限时自动分段转完再拼接，回执汇总各段页码。
func TestFormatConvert_AllPagesSegments(t *testing.T) {
	calls := stubConvert(t, 676)
	out := filepath.Join(t.TempDir(), "whole.md")
	args, _ := json.Marshal(map[string]any{"path": pdfPath(t), "output": out, "all_pages": true})
	res, err := (formatConvert{}).Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("all_pages 应成功: %v", err)
	}
	if strings.Join(*calls, "|") != "|501-676" {
		t.Fatalf("应先整份试一次、再从 501 起续转：%v", *calls)
	}
	if !strings.Contains(res, "共 676 页，分 2 段转换（1-500、501-676），已全部转换") {
		t.Errorf("回执应汇总各段页码：%s", res)
	}
	data, _ := os.ReadFile(out)
	md := string(data)
	if !strings.Contains(md, "已全部转换") || !strings.Contains(md, "正文[1-500]") || !strings.Contains(md, "正文[501-676]") ||
		strings.Index(md, "正文[1-500]") > strings.Index(md, "正文[501-676]") {
		t.Errorf("各段正文应按页序拼接：%.300s", md)
	}
	if strings.Contains(md, "本文档不完整") {
		t.Error("整份转完不该有不完整告警")
	}
}

// all_pages 的参数校验：须给 output、与 pages 互斥、只对 PDF→md。
func TestFormatConvert_AllPagesValidation(t *testing.T) {
	run := func(m map[string]any) error {
		args, _ := json.Marshal(m)
		_, err := (formatConvert{}).Execute(context.Background(), args)
		return err
	}
	p := pdfPath(t)
	for name, m := range map[string]map[string]any{
		"no output":     {"path": p, "all_pages": true},
		"with pages":    {"path": p, "output": "x.md", "all_pages": true, "pages": "1-3"},
		"to=docx":       {"path": p, "output": "x.docx", "all_pages": true, "to": "docx"},
		"non-pdf input": {"path": filepath.Join(t.TempDir(), "a.docx"), "output": "x.md", "all_pages": true},
	} {
		if err := run(m); err == nil {
			t.Errorf("%s：all_pages 应被参数校验拒绝", name)
		}
	}
	// 续转中途失败：错误里带上已转出的段数，不留下半份「成功」文件。
	stubConvert(t, 1200)
	orig := convertPreservingImagesReport
	convertPreservingImagesReport = func(_, pages string, _ int) (string, int, bool, int, *docmd.DocxRebuildReport, error) {
		if pages == "1001-1200" {
			return "", 1200, false, 0, nil, errors.New("WPS 超时")
		}
		return "正文[" + pages + "]", 1200, pages == "", 0, nil, nil
	}
	t.Cleanup(func() { convertPreservingImagesReport = orig })
	out := filepath.Join(t.TempDir(), "whole.md")
	args, _ := json.Marshal(map[string]any{"path": p, "output": out, "all_pages": true})
	_, err := (formatConvert{}).Execute(context.Background(), args)
	if err == nil || !strings.Contains(err.Error(), "第 1001-1200 页续转失败") || !strings.Contains(err.Error(), "前 2 段已转出") {
		t.Fatalf("续转失败应点名段落与已完成段数：%v", err)
	}
	if _, serr := os.Stat(out); !os.IsNotExist(serr) {
		t.Error("失败时不应留下半份成功文件")
	}
}
