package docmd

import "sync"

// ocr_progress.go — OCR 进度钩子（2026-11-18 办公 OCR 专属大卡）。
//
// Why：OCR 在对话流里此前全程静默——agent `ocr` 工具只有 dispatch→result 两
// 点、@附件 OCR 腿完全无声、扫描件 PDF 转换分钟级只有预览面板可见（
// preview_progress 专用通道）。前端要一张转录流大卡（页进度/引擎/状态/耗时），
// 需要底层把进度拱出来。
//
// How：包级单点钩子（对齐 ocr 进程内缓存的包级形态）——App 层启动注册一次，
// 所有 OCR 入口（OCRImageText / ocrPDFRange / ocrCached）统一经 emitOCRProgress
// 上报。钩子异常不反噬识别主链（进度面故障不能弄坏 OCR 本身）；钩子内不得
// 回调 docmd OCR（死锁面自负）——纯转发契约。

// OCR 进度阶段词汇（前端 OCRCard 按 stage 渲染）：
//
//	engine-start 引擎拉起/选型（冷启动可达分钟级，这是进度价值最大的阶段）
//	running      识别中（PDF 页粒度 done/total；图片单发）
//	cached       缓存命中（同文件重复 OCR 秒回）
//	done         完成（chars=提取 rune 数）
//	failed       失败（err=原因）
const (
	OCRStageEngineStart = "engine-start"
	OCRStageRunning     = "running"
	OCRStageCached      = "cached"
	OCRStageDone        = "done"
	OCRStageFailed      = "failed"
)

// OCRProgressEvent 一次 OCR 作业的进度快照。Path 是作业键（前端按 path 去重
// 原地更新同一张卡）；Engine 是当前引擎 kind（uocr2，或测试替身），选型前
// 为空；Done/Total 是页粒度进度（图片作业恒 Total=1）。
type OCRProgressEvent struct {
	Path   string `json:"path"`
	Engine string `json:"engine,omitempty"`
	Stage  string `json:"stage"`
	Done   int    `json:"done,omitempty"`
	Total  int    `json:"total,omitempty"`
	Chars  int    `json:"chars,omitempty"`
	Err    string `json:"err,omitempty"`
}

var (
	ocrProgressMu sync.RWMutex
	ocrProgressFn func(OCRProgressEvent)
)

// SetOCRProgressHook 注册 OCR 进度钩子（App 启动单点注册；nil=摘除）。
func SetOCRProgressHook(fn func(OCRProgressEvent)) {
	ocrProgressMu.Lock()
	ocrProgressFn = fn
	ocrProgressMu.Unlock()
}

// emitOCRProgress 钩子转发：无钩子 no-op；钩子 panic 被 recover 吞掉——
// 进度面故障绝不反噬 OCR 主链。
func emitOCRProgress(e OCRProgressEvent) {
	ocrProgressMu.RLock()
	fn := ocrProgressFn
	ocrProgressMu.RUnlock()
	if fn == nil {
		return
	}
	defer func() { _ = recover() }()
	fn(e)
}

// emitOCRDone 图片作业完成态便捷封装（chars=提取 rune 数，调用方已按 rune 计）。
func emitOCRDone(path, engine string, chars int) {
	emitOCRProgress(OCRProgressEvent{Path: path, Engine: engine, Stage: OCRStageDone, Total: 1, Chars: chars})
}
