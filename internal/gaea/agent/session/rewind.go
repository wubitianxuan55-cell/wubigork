package session

// rewind.go — 会话回退/分叉的事件日志支持（3.0 Step 1「日志即真相」的破坏性
// 派生操作）。日志本身保持 append-only；回退/分叉是用户显式放弃后续历史的
// 操作，由本文件提供「从日志定位回合边界」与「截断日志」两个原子能力，
// controller 层据此重建会话并接管。
//
// 对齐前端契约：CheckpointMeta{turn, prompt, files, time}（gaea_ui.go）——
// turn 0 起、prompt = 该轮用户消息、files = 该轮写过的文件、time = 回合起始。

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gaea/gaea/internal/gaea/fileutil"
)

// TurnRange 是日志中一个用户回合的消息边界（供 Checkpoints 与 Rewind 定位截断点）。
type TurnRange struct {
	Turn     int      // 0 起
	FirstSeq int64    // 该轮 user_message 事件的 seq
	LastSeq  int64    // 该轮最后一条事件（下一轮 user_message 之前）的 seq
	Prompt   string   // 该轮用户消息
	Time     int64    // 回合起始时间（unix 秒；无 turn_started 时取 user_message 的 ts）
	Files    []string // 该轮写过的文件（write 类工具 args.path，去重保序）
}

// writeToolNames 是写类工具的 args.path 提取白名单（与前端 lib/changes.ts
// 的 WRITE_TOOL_NAMES 对齐：写/改文件 → 该轮产物，回退点展示用）。
var writeToolNames = map[string]bool{
	"write_file": true, "edit_file": true, "edit_lines": true, "multi_edit": true,
}

// UserTurnRanges 从事件日志条目派生每个用户回合的边界。
// 无 user_message 条目时返回空。LastSeq 供 RewindLog 截断；FirstSeq 供回退
// 到「该轮之前」时计算 keepSeq（FirstSeq-1）。
func UserTurnRanges(entries []LogEntry) []TurnRange {
	var ranges []TurnRange
	cur := -1 // 当前回合在 ranges 的下标；-1 = 尚未开始
	for _, e := range entries {
		switch e.Kind {
		case KindUserMessage:
			var p userLogPayload
			_ = json.Unmarshal(e.Payload, &p)
			if cur >= 0 {
				// 上一回合的 LastSeq = 本回合 user_message 的前一条
				ranges[cur].LastSeq = e.Seq - 1
			}
			ranges = append(ranges, TurnRange{
				Turn:     len(ranges),
				FirstSeq: e.Seq,
				LastSeq:  e.Seq,
				Prompt:   p.Content,
				Time:     e.Ts,
			})
			cur = len(ranges) - 1
		case "turn_started":
			if cur >= 0 && ranges[cur].Time == 0 {
				ranges[cur].Time = e.Ts
			}
		case "tool_dispatch", KindToolCall:
			if cur < 0 {
				continue
			}
			var p toolCallLogPayload
			_ = json.Unmarshal(e.Payload, &p)
			if !writeToolNames[p.Name] || p.Args == "" {
				continue
			}
			var arg struct {
				Path string `json:"path"`
			}
			if err := json.Unmarshal([]byte(p.Args), &arg); err != nil || arg.Path == "" {
				continue
			}
			files := ranges[cur].Files
			dup := false
			for _, f := range files {
				if f == arg.Path {
					dup = true
					break
				}
			}
			if !dup {
				ranges[cur].Files = append(files, arg.Path)
			}
		}
	}
	// 收尾：最后一个回合的 LastSeq = 日志最后一条 seq（无后续 user_message 触发）。
	if n := len(ranges); n > 0 && len(entries) > 0 {
		ranges[n-1].LastSeq = entries[len(entries)-1].Seq
	}
	return ranges
}

// RewindLog 把事件日志截断为保留 seq <= keepSeq 的条目（原子重写）。
// 用途：回退到第 N 轮之前 → keepSeq = 第 N 轮 FirstSeq-1；回退到初始 → 0
// （清空日志，保留空文件以免 OpenLog 的 torn-tail 修复误判）。
// 注意：调用方负责确保没有活跃 LogWriter 正在写该日志（回合间 sink 已关闭）。
func RewindLog(logPath string, keepSeq int64) error {
	if logPath == "" {
		return fmt.Errorf("empty log path")
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("rewind: log not found: %s", logPath)
		}
		return fmt.Errorf("rewind: read log: %w", err)
	}
	entries, err := parseLogBytes(data)
	if err != nil {
		return fmt.Errorf("rewind: parse log: %w", err)
	}
	var buf bytes.Buffer
	kept := 0
	for _, e := range entries {
		if e.Seq > keepSeq {
			continue
		}
		buf.Write(formatLogLine(e.Seq, e.Ts, e.Kind, e.Payload))
		kept++
	}
	if kept == 0 {
		// 空日志：直接清空文件（sink 下次 OpenLog 会重新从 0 起 seq）。
		return fileutil.AtomicWrite(logPath, nil, 0o644)
	}
	return fileutil.AtomicWrite(logPath, buf.Bytes(), 0o644)
}
