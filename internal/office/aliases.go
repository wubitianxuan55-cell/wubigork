// Package office — 办公引擎门面。内核能力（桌面 agent 文件读写 / 会话持久化 /
// 会话任务态 / 变更留痕 journal）已下沉 internal/core，本包仅保留绑定面所需类型
// 的等价别名：bindings_office.go 由 scripts/gen_bindings 生成且签名受
// TestBindingsCompleteness 保护，故经类型别名保持 office.ExecResult /
// office.AgentJobState 恒定（别名 = 同一类型，JSON/反射零变化）。
package office

import "github.com/gaea/gaea/internal/core"

// ExecResult 桌面 agent 执行结果（别名，绑定面签名不变）。
type ExecResult = core.ExecResult

// AgentJobState 任务状态（别名，绑定面签名不变）。
type AgentJobState = core.AgentJobState