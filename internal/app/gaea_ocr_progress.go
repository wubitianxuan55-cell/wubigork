package app

import (
	"path/filepath"

	"github.com/gaea/gaea/internal/docmd"
)

// gaea_ocr_progress.go — OCR 进度钩子转发（2026-11-18 办公 OCR 专属大卡）。
//
// docmd OCR 进度（引擎拉起/页粒度/缓存命中/终态）经包级钩子转发 gaea-event，
// 前端转录流 OCRCard 大卡消费。docmd 是共享底层（对话 ocr 工具/@附件 OCR 腿/
// 扫描件转换/成本导入/文件预览共用），钩子在 App 启动单点注册（app.go，
// SetVisionRuntimeResolver 同区）；文件预览面板的 preview_progress 专用通道
// 不受影响（双通道并存——预览条在面板内、大卡在转录流，互不替代）。

// ocrProgressPayload OCR 进度事件的前端 payload（纯函数，测试面）：
// 单对象（非数组），name=basename 供大卡展示（免前端再算）。
func ocrProgressPayload(e docmd.OCRProgressEvent) map[string]interface{} {
	return map[string]interface{}{
		"kind": "ocr_progress",
		"ocr": map[string]interface{}{
			"path":   e.Path,
			"name":   filepath.Base(e.Path),
			"engine": e.Engine,
			"stage":  e.Stage,
			"done":   e.Done,
			"total":  e.Total,
			"chars":  e.Chars,
			"err":    e.Err,
		},
	}
}

// emitOCRProgress docmd.OCRProgressEvent → gaea-event ocr_progress。
func (a *App) emitOCRProgress(e docmd.OCRProgressEvent) {
	a.emit("gaea-event", ocrProgressPayload(e))
}
