// Package builtin — U1 蒸馏契约：office 域工具错误码。
// 形态 `Error [CODE]: message`，模型按 code 路由恢复路径，不解析自然语言
// （蒸馏自 dsh-univer-office 的错误路由口径；范围仅 office 域工具，不全局推广，
// 见 docs/gaea-dsh-univer-office-distill-plan-2026-09.md 拍板项 4）。
package builtin

import "fmt"

func codedError(code, format string, args ...any) error {
	return fmt.Errorf("Error [%s]: %s", code, fmt.Sprintf(format, args...))
}
