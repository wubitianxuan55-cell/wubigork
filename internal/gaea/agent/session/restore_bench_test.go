package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ─── projection-cache 收益评估基准（v4.386，dsh ⑪ session-projection-cache）───
//
// 回答一个问题：会话冷启动恢复（Restore）在真实规模的事件日志上有多贵？
// 有 checkpoint（每轮模型调用前 flush 的既有机制）时尾差投影有多小？
// 若恢复成本可忽略（<几百 ms），上游的 projection-cache（投影单元按 session
// 落盘）在 gaea 无立项价值——checkpoint 机制已是等价物；若成本显著再立项。
//
// 运行：go test ./internal/gaea/agent/session/ -bench BenchmarkRestore -benchtime 3x

// benchLogMB 生成约 targetMB 大小的合成事件日志（user/assistant 交替，
// 单条正文 ~1.5KB，贴近真实会话密度），并在倒数第 100 条处落 checkpoint
// （模拟「每轮模型调用前 flush」的既有纪律——恢复只投影尾差）。
func benchLogMB(b *testing.B, dir string, targetMB int, withCheckpoint bool) (logPath, cpPath string, entries int) {
	logPath = filepath.Join(dir, "bench-sess.gaea-log.jsonl")
	cpPath = filepath.Join(dir, "bench-sess.gaea-ckpt.json")

	msgText := strings.Repeat("基准正文内容——包含中文与 ascii mixed content，覆盖代码片段 `go test ./...` 与路径 C:\\work\\file.go。", 12)
	target := targetMB * 1024 * 1024
	entries = 0
	var sb strings.Builder
	written := 0
	for written < target {
		entries++
		for _, kind := range []string{"user_message", "assistant_message"} {
			var payload []byte
			if kind == "user_message" {
				payload, _ = json.Marshal(map[string]string{"content": msgText})
			} else {
				payload, _ = json.Marshal(map[string]string{"id": fmt.Sprintf("a%d", entries), "text": msgText})
			}
			line, _ := json.Marshal(map[string]any{
				"seq": entries, "ts": 1758500000 + entries, "kind": kind, "payload": payload,
			})
			sb.Write(line)
			sb.WriteByte('\n')
			written += len(line)
		}
	}
	if err := os.WriteFile(logPath, []byte(sb.String()), 0o644); err != nil {
		b.Fatal(err)
	}

	if withCheckpoint {
		// checkpoint 落在倒数第 100 条（每轮 flush 的既有形态：尾差≈一轮工具链）。
		cutoff := entries - 100
		msgs := make([]providerMessage, 0, cutoff)
		for i := 1; i <= cutoff; i++ {
			role, text := "user", msgText
			if i%2 == 0 {
				role, text = "assistant", msgText
			}
			msgs = append(msgs, providerMessage{Role: role, Content: text})
		}
		cp := map[string]any{"seq": cutoff, "space": "work", "messages": msgs}
		raw, _ := json.Marshal(cp)
		if err := os.WriteFile(cpPath, raw, 0o644); err != nil {
			b.Fatal(err)
		}
	}
	return logPath, cpPath, entries
}

// providerMessage/providerMessage 影子类型：bench 内避免导入 provider（该包
// 只需 Role/Content 字段形状；json 序列化后与真实 checkpoint 同构）。
type providerMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func BenchmarkRestoreSynthetic(b *testing.B) {
	for _, mb := range []int{2, 10, 50} {
		for _, withCP := range []bool{true, false} {
			name := fmt.Sprintf("log-%dMB-checkpoint-%v", mb, withCP)
			b.Run(name, func(b *testing.B) {
				dir := b.TempDir()
				logPath, cpPath, entries := benchLogMB(b, dir, mb, withCP)
				b.ReportMetric(float64(mb), "log-MB")
				b.ReportMetric(float64(entries), "entries")
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					msgs, _, err := Restore(cpPath, logPath)
					if err != nil {
						b.Fatal(err)
					}
					if len(msgs) == 0 {
						b.Fatal("empty restore")
					}
				}
			})
		}
	}
}
