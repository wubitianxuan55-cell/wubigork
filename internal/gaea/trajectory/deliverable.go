package trajectory

import (
	"encoding/json"
	"sort"

	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/evidence"
)

// 权威产物登记表（v4.24 提升项 C1，docs/gaea-office-upgrade-plan-2026-09.md）：
// 从事件日志（会话事实源）折叠出「写类工具落盘登记」，前端只读该表，
// 替代正文扩展名白名单等启发式（漏登 agent 写出的非常规扩展名文件）。
// 与 FoldTrajectory/FoldTimeline 同构：纯函数 + 单遍折叠 + 黄金测试。

// maxDeliverables 是登记表保留上限（浏览视图防爆炸）：按 updatedAt 倒序
// 保留最近 200 条；Total 仍返回去重后的完整登记数（前端可提示「还有 N 条」）。
const maxDeliverables = 200

// DeliverableEntry 是登记表中的一条产物记录。
type DeliverableEntry struct {
	Path      string `json:"path"`      // 产物路径（工具参数原样，trim 后）
	Tool      string `json:"tool"`      // 最近一次写入该路径的工具
	Turn      int    `json:"turn"`      // 最近一次写入的来源轮次（1-based；0=轮外）
	UpdatedAt int64  `json:"updatedAt"` // 最近一次写入时间（日志 ts，unix 秒）
	Touches   int    `json:"touches"`   // 该路径累计写入次数
}

// DeliverableRegistry 是会话的权威产物登记表。Entries 按 updatedAt 倒序、
// 上限 maxDeliverables 条；Total = 去重后完整登记数（可能 > len(Entries)）。
type DeliverableRegistry struct {
	Available bool               `json:"available"`
	Entries   []DeliverableEntry `json:"entries"`
	Total     int                `json:"total"`
}

// FoldDeliverables 把会话日志条目折叠为权威产物登记表。纯函数：同输入必同输出。
// 口径：
//   - 只登记 evidence.IsDeliverableTool 的工具（写类 8 种 + 生成/导出类 3 种）；
//   - 路径从工具调用参数权威提取（evidence.ExtractDeliverablePaths，与证据链
//     extractPaths / agent extractFilePath 同源对齐，不另造一套）；
//   - 按 path 去重保留最近一次（tool/turn/updatedAt 刷新，touches 累计）；
//   - 登记时机 = 工具派发（dispatch）：写盘中的产物立刻可见（前端轮询实时）；
//     失败调用不回剔——登记的是「尝试落盘」的产物路径，文件是否存在由预览/
//     文件树判断（与 contextview 文件活动时间线同口径）。
func FoldDeliverables(entries []session.LogEntry) DeliverableRegistry {
	f := &deliverableFolding{byPath: map[string]*DeliverableEntry{}}
	for _, e := range entries {
		f.apply(e)
	}
	out := DeliverableRegistry{Available: true, Entries: f.ordered(), Total: len(f.byPath)}
	// Go 的 nil 切片会序列化成 JSON null，前端按数组消费会崩——统一兜底空切片。
	if out.Entries == nil {
		out.Entries = []DeliverableEntry{}
	}
	return out
}

type deliverableFolding struct {
	turn   int
	byPath map[string]*DeliverableEntry
}

func (f *deliverableFolding) apply(e session.LogEntry) {
	switch e.Kind {
	case "turn_started":
		f.turn++
	case "tool_dispatch":
		var p struct {
			Name    string `json:"name"`
			Args    string `json:"args"`
			Partial bool   `json:"partial,omitempty"`
		}
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return
		}
		if p.Partial {
			return
		}
		f.record(p.Name, p.Args, e.Ts)
	case "assistant_message":
		// 迁移/投影产物（ToLogEntries）把工具调用内嵌在 assistant_message 里，
		// 与运行期 tool_dispatch 同一归并面（trajectory fold 同款处理）。
		var p struct {
			ToolCalls []struct {
				Name string `json:"name"`
				Args string `json:"args"`
			} `json:"tool_calls,omitempty"`
		}
		if err := json.Unmarshal(e.Payload, &p); err != nil {
			return
		}
		for _, tc := range p.ToolCalls {
			f.record(tc.Name, tc.Args, e.Ts)
		}
	}
}

// record 把一次产物工具调用登记进表：按 path 去重，保留最近一次的
// tool/turn/updatedAt，touches 累计。
func (f *deliverableFolding) record(tool, args string, ts int64) {
	if tool == "" || !evidence.IsDeliverableTool(tool) {
		return
	}
	for _, path := range evidence.ExtractDeliverablePaths(tool, json.RawMessage(args)) {
		if rec, ok := f.byPath[path]; ok {
			rec.Tool = tool
			rec.Turn = f.turn
			rec.UpdatedAt = ts
			rec.Touches++
			continue
		}
		f.byPath[path] = &DeliverableEntry{Path: path, Tool: tool, Turn: f.turn, UpdatedAt: ts, Touches: 1}
	}
}

// ordered 返回按 updatedAt 倒序（同刻按 path 字典序，保证纯函数稳定输出）
// 的登记切片，超出 maxDeliverables 截断（保留最近）。
func (f *deliverableFolding) ordered() []DeliverableEntry {
	out := make([]DeliverableEntry, 0, len(f.byPath))
	for _, rec := range f.byPath {
		out = append(out, *rec)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt != out[j].UpdatedAt {
			return out[i].UpdatedAt > out[j].UpdatedAt
		}
		return out[i].Path < out[j].Path
	})
	if len(out) > maxDeliverables {
		out = out[:maxDeliverables]
	}
	return out
}
