package computer

import "errors"

// 电脑操控域错误哨兵（施工单 W1：errors.go）。
var (
	// ErrUnsupported 非 Windows 平台：全部能力不可用（computer_other.go）。
	ErrUnsupported = errors.New("computer use is only supported on Windows")
	// ErrStaleObservation 失效闸门：element_index/坐标/shotID 只属于产生它的
	// 那次观察，版次不匹配一律拒（语义照抄 codex「window changed; call
	// get_window_state …」，文案按 gaea 工具名自写）。
	ErrStaleObservation = errors.New("window changed; call computer_state before using an element index or coordinate")
	// ErrScreenshotIDRequired 坐标闸门：多区域截图时坐标动作必须携带本次
	// 观察的 screenshotId（P2 多区域生效；单区域路径 shotID 自动满足）。
	ErrScreenshotIDRequired = errors.New("coordinate input requires a screenshot from the latest computer_state observation")
	// ErrForbiddenKey 本地硬拒：键名含 Win/Meta/Cmd/Super/OS 的组合键
	//（asset-62 §8.3 不可谈判红线），先于任何策略/审批。
	ErrForbiddenKey = errors.New("forbidden key: chords containing Win/Meta/Cmd/Super/OS are hard-denied")
	// ErrWindowGone 目标窗口已关闭或不存在。
	ErrWindowGone = errors.New("window is gone; list windows again")
	// ErrCoordinatesDisabled DPI 上下文复原失败后的整体坐标禁用
	//（asset-62 §6「失败即禁用」语义，照抄）。
	ErrCoordinatesDisabled = errors.New("DPI context restoration failed and coordinate operations are disabled")
)

// errDpiRestoreFailed 是 ErrCoordinatesDisabled 的底层信号（withThreadDpi
// 复原失败时叠加返回；动作编排层据此置 coordinatesDisabled）。
var errDpiRestoreFailed = errors.New("DPI context restoration failed")
