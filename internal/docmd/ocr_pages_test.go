package docmd

// ocr_pages_test.go — T7-3：OCR 逐页失败跳过（全败才报错）与页范围过滤。全部为纯单测：
// ocrPageLoop 直接注入 fake 单页 OCR 函数。（OvisOCR2 单页识别的截断检测用例随
// 2026-11-26 退役删除；uocr2 的截断检测见 ocr_uocr2_llama_test.go。）

import (
	"strings"
	"testing"
)

// TestOCRPageLoopSkipSinglePageFailures 单页失败跳过继续：第 2 页失败，
// 第 1/3 页成功 → 返回成功文本、失败页码 [2]、无错误；progress total 为实际
// OCR 页数（3 页全参与）。
func TestOCRPageLoopSkipSinglePageFailures(t *testing.T) {
	files := []string{"p1.png", "p2.png", "p3.png"}
	var gotProgress []int
	texts, total, failed, err := ocrPageLoop(files, 1, "", func(pageNum int, pngPath string) (string, error) {
		if pageNum == 2 {
			return "", &fakeOCRError{page: pageNum}
		}
		return "page-" + pngPath, nil
	}, func(done, totalN int) {
		gotProgress = append(gotProgress, totalN)
	})
	if err != nil {
		t.Fatalf("单页失败不应整体报错: %v", err)
	}
	if len(texts) != 2 || !strings.Contains(texts[0], "p1.png") || !strings.Contains(texts[1], "p3.png") {
		t.Fatalf("应返回成功页文本: %+v", texts)
	}
	if total != 3 {
		t.Fatalf("totalOCR 应为 3（实际页数）: %d", total)
	}
	if len(failed) != 1 || failed[0] != 2 {
		t.Fatalf("失败页应为 [2]: %+v", failed)
	}
	if len(gotProgress) != 4 || gotProgress[0] != 3 || gotProgress[3] != 3 {
		t.Fatalf("progress 应含起步事件+逐页三次，total 恒为 3: %+v", gotProgress)
	}
}

// TestOCRPageLoopAllPagesFail 全部页失败 → 返回错误（含失败页码），不得静默成功。
func TestOCRPageLoopAllPagesFail(t *testing.T) {
	files := []string{"p1.png", "p2.png"}
	_, _, _, err := ocrPageLoop(files, 1, "", func(pageNum int, pngPath string) (string, error) {
		return "", &fakeOCRError{page: pageNum}
	}, nil)
	if err == nil {
		t.Fatal("全部页失败应报错")
	}
	if !strings.Contains(err.Error(), "全部 2 页失败") || !strings.Contains(err.Error(), "1") {
		t.Fatalf("错误应说明全部失败与失败页码: %v", err)
	}
}

// TestOCRPageLoopEmptyTextIsFailure 单页返回空文本视为失败（跳过），
// 但只要有其他页成功就不报错。
func TestOCRPageLoopEmptyTextIsFailure(t *testing.T) {
	files := []string{"p1.png", "p2.png"}
	texts, total, failed, err := ocrPageLoop(files, 1, "", func(pageNum int, pngPath string) (string, error) {
		if pageNum == 2 {
			return "   ", nil // 空文本 → 失败
		}
		return "ok", nil
	}, nil)
	if err != nil {
		t.Fatalf("不应报错: %v", err)
	}
	if len(texts) != 1 || total != 2 || len(failed) != 1 || failed[0] != 2 {
		t.Fatalf("空文本页应计入失败: texts=%+v total=%d failed=%+v", texts, total, failed)
	}
}

// TestOCRPageLoopPageRangeTotal 页范围过滤后 totalOCR 按实际页数计（非渲染页数）。
func TestOCRPageLoopPageRangeTotal(t *testing.T) {
	files := []string{"p1.png", "p2.png", "p3.png"} // 渲染了 3 页
	var totals []int
	texts, total, _, err := ocrPageLoop(files, 1, "1", func(pageNum int, pngPath string) (string, error) {
		return "text", nil
	}, func(done, totalN int) {
		totals = append(totals, totalN)
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if total != 1 || len(texts) != 1 {
		t.Fatalf("pages=1 时应只 OCR 1 页: total=%d texts=%d", total, len(texts))
	}
	if len(totals) != 2 || totals[0] != 1 || totals[1] != 1 {
		t.Fatalf("progress 应为起步事件+1 页，total 恒为 1: %+v", totals)
	}
}

// fakeOCRError 模拟单页 OCR 引擎错误。
type fakeOCRError struct{ page int }

func (e *fakeOCRError) Error() string { return "engine failed on page" }
