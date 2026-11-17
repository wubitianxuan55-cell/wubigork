package docmd

import "testing"

// TestSetOCRProgressHook 钩子注册与事件送达（OCR 大卡契约）：
// 注册后 emitOCRProgress 送达注册函数；摘除（nil）后静默 no-op。
func TestSetOCRProgressHook(t *testing.T) {
	var got []OCRProgressEvent
	SetOCRProgressHook(func(e OCRProgressEvent) { got = append(got, e) })
	emitOCRProgress(OCRProgressEvent{Path: "a.png", Stage: OCRStageEngineStart, Total: 1})
	emitOCRProgress(OCRProgressEvent{Path: "a.png", Stage: OCRStageDone, Total: 1, Chars: 42})
	if len(got) != 2 {
		t.Fatalf("钩子应收到 2 个事件，got %d", len(got))
	}
	if got[0].Stage != OCRStageEngineStart || got[1].Stage != OCRStageDone || got[1].Chars != 42 {
		t.Fatalf("事件字段错: %+v", got)
	}
	SetOCRProgressHook(nil)
	emitOCRProgress(OCRProgressEvent{Path: "b", Stage: OCRStageRunning})
	if len(got) != 2 {
		t.Fatalf("摘除后不应再送达")
	}
}

// TestOCRProgressHookPanicSafe 钩子 panic 不反噬 OCR 主链（recover 契约）。
func TestOCRProgressHookPanicSafe(t *testing.T) {
	SetOCRProgressHook(func(OCRProgressEvent) { panic("hook boom") })
	t.Cleanup(func() { SetOCRProgressHook(nil) })
	emitOCRProgress(OCRProgressEvent{Path: "x", Stage: OCRStageDone})
	// 走到这里 = panic 未逃逸
}
