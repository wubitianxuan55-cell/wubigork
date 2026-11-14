package contextview

import "github.com/gaea/gaea/internal/gaea/agent/session"

// SurfaceAt 从日志重放到指定 request_header，返回该请求组装时的模型可见
// surface。只接受真实存在于日志中的 request_header seq；未知/非请求头 seq
// 返回 ok=false，调用方不得回退成当前 live surface。
func SurfaceAt(entries []session.LogEntry, seq, window int64) (SurfaceSnapshot, bool) {
	end := -1
	for i := range entries {
		if entries[i].Seq == seq && entries[i].Kind == session.KindRequestHeader {
			end = i
			break
		}
	}
	if end < 0 {
		return SurfaceSnapshot{}, false
	}
	tl := FoldTimeline(entries[:end+1], window, 0)
	snapshot := SurfaceSnapshot{
		Ok:           true,
		Seq:          seq,
		Nodes:        tl.Nodes,
		Archive:      tl.Archive,
		NodesDropped: tl.NodesDropped,
		NodesFloor:   tl.NodesFloor,
	}
	if snapshot.Nodes == nil {
		snapshot.Nodes = []SurfaceNode{}
	}
	if snapshot.Archive == nil {
		snapshot.Archive = []SurfaceNode{}
	}
	return snapshot, true
}
