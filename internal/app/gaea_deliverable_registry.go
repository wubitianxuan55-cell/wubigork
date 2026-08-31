package app

import (
	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/trajectory"
)

// DeliverableRegistryView 是会话的权威产物登记表视图（v4.24 提升项 C1）：
// 后端从事件日志折叠出「写类工具落盘登记」（路径/来源轮次/工具/时间/次数），
// 前端只读该表，替代正文扩展名白名单启发式（漏登 agent 写出的非常规
// 扩展名文件）。Entries 按 updatedAt 倒序、上限 200 条；Total 为去重后
// 完整登记数（可能 > len(Entries)，前端可提示「还有 N 条未列出」）。
type DeliverableRegistryView struct {
	Available bool                          `json:"available"` // false = 无事件日志（legacy 未迁移/路径非法）
	Entries   []trajectory.DeliverableEntry `json:"entries"`
	Total     int                           `json:"total"`
}

// GaeaDeliverableRegistry 返回指定会话的权威产物登记表（折叠口径见
// trajectory.FoldDeliverables）。防御式风格与 GaeaSessionStats 一致：
//   - sessionPath 经 sessionDirForPath 校验（防穿越，仅接受会话目录族）；
//   - 数据源直接读事件日志（不回退 legacy 投影）：无日志（legacy 会话）或
//     读取失败返回 Available=false（前端显示空态，不报错）——恢复会话时
//     ResumeSession 会自动迁移出事件日志，此后登记表即生效。
func (a *App) GaeaDeliverableRegistry(sessionPath string) DeliverableRegistryView {
	if sessionPath == "" || sessionDirForPath(sessionPath) == "" {
		return DeliverableRegistryView{}
	}
	lp := session.LogPathFor(sessionPath)
	if lp == "" {
		return DeliverableRegistryView{}
	}
	entries, err := session.ReadLogRepaired(lp)
	if err != nil {
		// 无日志文件（legacy 会话）或读取失败：Available=false，不报错。
		return DeliverableRegistryView{}
	}
	reg := trajectory.FoldDeliverables(entries)
	return DeliverableRegistryView{Available: true, Entries: reg.Entries, Total: reg.Total}
}
