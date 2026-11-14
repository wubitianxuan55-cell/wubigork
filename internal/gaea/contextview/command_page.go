package contextview

import (
	"sort"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// CommandPageAt 从完整会话日志推导命令历史，返回 beforeSeq 之前最近的一页。
// FoldTimeline 的 400 条窗口只约束常规面板；本函数用于显式“加载更早”，
// 单页默认 200、硬上限 1000。
func CommandPageAt(entries []session.LogEntry, beforeSeq int64, limit int) CommandPage {
	if limit <= 0 {
		limit = 200
	}
	if limit > 1000 {
		limit = 1000
	}
	f := &folding{}
	for _, entry := range entries {
		f.apply(entry)
	}
	f.killOpenCmds(0)
	records := append([]CommandRecord(nil), f.commands...)
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].Seq == records[j].Seq {
			return records[i].Time < records[j].Time
		}
		return records[i].Seq < records[j].Seq
	})
	total := len(records)
	end := total
	if beforeSeq > 0 {
		end = sort.Search(total, func(i int) bool {
			return records[i].Seq >= beforeSeq
		})
	}
	start := end - limit
	if start < 0 {
		start = 0
	}
	page := append([]CommandRecord(nil), records[start:end]...)
	if page == nil {
		page = []CommandRecord{}
	}
	return CommandPage{
		Commands: page,
		AtStart:  start == 0,
		Total:    total,
	}
}
