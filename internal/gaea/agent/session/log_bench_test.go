package session

// 刀A（v4.245）性能基线：回合边界写入器重开成本。长会话下日志含全部
// reasoning/text 增量，基线 OpenLog 每回合 = RepairLogFile 全量读 +
// countLogLines 全量读+逐行解析，随会话总字节线性放大；OpenLogResuming
// 续接形态在干净关闭且文件未变时 O(1)。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// benchSeedLog 预置一份约 target 字节的合法事件日志（每行 ~460B）。
func benchSeedLog(b *testing.B, logPath string, target int) {
	b.Helper()
	f, err := os.Create(logPath)
	if err != nil {
		b.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]string{"text": string(make([]byte, 400))})
	n := int64(0)
	written := 0
	for written < target {
		n++
		line, _ := json.Marshal(struct {
			Seq     int64           `json:"seq"`
			Ts      int64           `json:"ts"`
			Kind    string          `json:"kind"`
			Payload json.RawMessage `json:"payload"`
		}{n, 1, "text", payload})
		line = append(line, '\n')
		if _, err := f.Write(line); err != nil {
			b.Fatal(err)
		}
		written += len(line)
	}
	if err := f.Close(); err != nil {
		b.Fatal(err)
	}
}

// BenchmarkOpenLogTurnBoundary 度量「每条用户消息回合边界」的写入器重开税：
// openlog=基线 OpenLog（全量修复+计数，改前 sink 每回合的实际成本）；
// resuming=OpenLogResuming（干净关闭后凭续接点重开，改后 sink 路径）。
// 一轮模拟：打开→写一条事件→干净关闭（turn_done 语义）。
func BenchmarkOpenLogTurnBoundary(b *testing.B) {
	for _, size := range []int{256 << 10, 2 << 20} {
		b.Run(fmt.Sprintf("log=%dKB", size>>10), func(b *testing.B) {
			dir := b.TempDir()
			logPath := filepath.Join(dir, "bench.gaea-log.jsonl")
			benchSeedLog(b, logPath, size)

			b.Run("openlog", func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					w, err := OpenLog(logPath, "", "work")
					if err != nil {
						b.Fatal(err)
					}
					if _, err := w.AppendRaw("turn_started", json.RawMessage(`{}`)); err != nil {
						b.Fatal(err)
					}
					if err := w.Close(); err != nil {
						b.Fatal(err)
					}
				}
			})
			b.Run("resuming", func(b *testing.B) {
				b.ReportAllocs()
				// 首轮 seq=0 走全量回落（冷启动形态），此后每轮以「上轮干净
				// 关闭时的 seq + 当时文件大小」续接——与 sink 逐轮记续接点一致。
				seq := int64(0)
				resumeSize := int64(0)
				for i := 0; i < b.N; i++ {
					w, err := OpenLogResuming(logPath, "", "work", seq, resumeSize)
					if err != nil {
						b.Fatal(err)
					}
					if i > 0 && w.Seq() != seq {
						b.Fatalf("seq 断裂: got %d want %d", w.Seq(), seq)
					}
					if _, err := w.AppendRaw("turn_started", json.RawMessage(`{}`)); err != nil {
						b.Fatal(err)
					}
					seq = w.Seq()
					st, err := os.Stat(logPath)
					if err != nil {
						b.Fatal(err)
					}
					resumeSize = st.Size()
					if err := w.Close(); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}
