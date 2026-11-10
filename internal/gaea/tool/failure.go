package tool

// Failure 结构化失败（B2 失败即值，dsh-ops 蒸馏 2026-11-10）：
//   - 工具在 Go error 通道返回 *Failure，executeOne 收口点按其字段渲染信封
//     （code/retryable/effect_state/next_action），而不是退化为通用 exec_error
//     首行文本；
//   - EffectState 是副作用类工具的诚实失败语义核心：none=失败发生在任何
//     副作用之前（CAS 原子拒绝、参数校验）；unknown=副作用可能已部分发生
//     （超时被杀、ctx 中断、非零退出）。unknown 时 NextAction 必须明示
//     「先核实现状再重试」，防模型盲重试二次破坏（上游 file.js:113 原型）；
//   - Error() 原样透传底层文案：REVISION_CONFLICT/CREDENTIAL_DENIED 等既有
//     前缀文本是兼容层（测试与字符串消费方在册），结构化只做加法。
//
// 不抄边界（distill 报告 §4）：不引入 phase 字段（gaea 无相位概念）；不动
// (string, error) 工具契约——Failure 就是 error 的一种。

import (
	"errors"
	"strings"
)

// 副作用状态。none/unknown 两态为主（partial 与 unknown 对模型的下一步动作
// 完全等价，不加）；applied（D3）=副作用已发生但流程报错——「已生效，验证
// 而非重试」（如 move 复制成功删源失败、上游发布后 fsync 失败）。
const (
	// EffectNone 失败先于任何副作用（或工具明示无副作用）。
	EffectNone = "none"
	// EffectUnknown 副作用可能已部分发生，重试前必须先核实现状。
	EffectUnknown = "unknown"
	// EffectApplied 副作用已发生（文件已落盘/命令已产出），失败在其后——
	// 重试会二次生效，必须先验证现状再决定清理或继续。
	EffectApplied = "applied"
)

// Failure 是工具错误的结构化形态。Code 为空时按 exec_error 渲染；Subcode
// 为同 code 下的具体阶段/成因（直接回答「重试有没有用」）；ExitCode/
// HasExitCode 供 bash 族把退出码带进信封 data（exit code 只活在文案里是
// 蒸馏报告实证的缺口）。
type Failure struct {
	Code        string
	Subcode     string
	Err         error
	Retryable   bool
	EffectState string // EffectNone / EffectUnknown / EffectApplied
	NextAction  string
	ExitCode    int
	HasExitCode bool
}

func (f *Failure) Error() string {
	if f == nil || f.Err == nil {
		if f != nil && f.Code != "" {
			return f.Code
		}
		return "tool failure"
	}
	return f.Err.Error()
}

func (f *Failure) Unwrap() error { return f.Err }

// AsFailure 报告 err 链上是否携带结构化失败（errors.As 语义，兼容包装）。
func AsFailure(err error) (*Failure, bool) {
	var f *Failure
	if errors.As(err, &f) {
		return f, true
	}
	return nil, false
}

// DefaultRetryable 按信封码给出「原样重试是否可能成功」缺省值。修正参数
// 类（validation_error/conflict）不算 retryable——重试前必须换输入；但
// conflict 的换输入路径在 NextAction 里给了指引。
func DefaultRetryable(code string) bool {
	switch code {
	case CodeTimeout, CodeExecError:
		return true
	default:
		return false
	}
}

// DefaultNextAction 按码+副作用状态给出下一步短句（token 纪律：一句以内）。
// exec_error 的动作分叉取决于 effect_state，调用方须传入正确状态。
func DefaultNextAction(code, effectState string) string {
	switch effectState {
	case EffectApplied:
		return "副作用已生效：先验证现状再决定清理或继续，勿盲目重试"
	case EffectUnknown:
		switch code {
		case CodeTimeout:
			return "超时被杀，副作用状态未知：先检查后置条件再决定重试"
		default:
			return "副作用可能已部分发生：先核实现状再决定重试"
		}
	}
	switch code {
	case CodeTimeout:
		return "可重试"
	case CodeConflict:
		return "文件已被外部修改：重读拿最新 revision 后再改"
	case CodeValidationError:
		return "修正参数后重试"
	case CodeDenied:
		return "勿原样重试：换路径或请用户授权"
	case CodeNotFound:
		return "确认路径是否存在（或需先创建）"
	case CodeExecError:
		return "可重试"
	default:
		return ""
	}
}

// WrapFailure 把结构化失败渲染为信封 JSON（executeOne 收口点专用）。
// errMsg 传首行文案（与 WrapError 的 error 字段口径一致）。
func WrapFailure(f *Failure, errMsg string, data any) string {
	code := f.Code
	if code == "" {
		code = CodeExecError
	}
	effect := f.EffectState
	next := f.NextAction
	if next == "" {
		next = DefaultNextAction(code, effect)
	}
	return mustMarshal(ToolEnvelope{
		OK:          false,
		Success:     false,
		Code:        code,
		Subcode:     f.Subcode,
		Error:       strings.TrimSpace(errMsg),
		Data:        data,
		Retryable:   f.Retryable,
		EffectState: effect,
		NextAction:  next,
	})
}
