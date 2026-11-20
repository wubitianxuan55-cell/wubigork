package computer

// W4 按 app 授权的接线（电脑操控施工单）：给 permission.Subject 供
// window_id → process:<exe 全路径> 解析。permission 包零依赖本包（钩子
// 注入，避免反向依赖）；解析失败（窗口已亡/枚举失败）返回空串，审批退回
// 裸工具规则（现状语义）。每次解析走一次真实 EnumWindows（毫秒级，只发生
// 在电脑族工具的闸门检查上）。
import "github.com/gaea/gaea/internal/gaea/permission"

func init() {
	permission.WindowSubjectResolver = func(windowID int) string {
		wins, err := ListWindows()
		if err != nil {
			return ""
		}
		for _, w := range wins {
			if w.ID == windowID {
				return w.App
			}
		}
		return ""
	}
}
