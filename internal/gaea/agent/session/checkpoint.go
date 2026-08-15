package session

// 3.0 Step 1: 检查点。压缩发生时写 <id>.gaea-checkpoint.json
// （压缩后消息投影 + 最后消费的 log seq）；恢复 = checkpoint + 从 seq 后重放。
// 模型调用前 flush 检查点（fail-closed）：压缩后、新一轮模型调用前落盘，
// 断电/崩溃后日志仍可回放到检查点处。

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/gaea/gaea/internal/gaea/fileutil"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// Checkpoint 是压缩后落盘的会话快照：投影消息 + 最后消费的 log seq。
type Checkpoint struct {
	// Seq 是 checkpoint 覆盖到的最后一条日志 seq；恢复时从 seq+1 起重放。
	Seq int64 `json:"seq"`
	// Messages 是压缩后的消息投影（system + digest + tail）。
	Messages []provider.Message `json:"messages"`
	// Rewrite 是压缩/折叠重写版本（透传，便于对账）。
	Rewrite int   `json:"rewrite,omitempty"`
	Created int64 `json:"createdAt,omitempty"`
}

// WriteCheckpoint 原子写入检查点（写临时文件再 rename，崩溃不残留半截）。
func WriteCheckpoint(path string, seq int64, msgs []provider.Message) error {
	if path == "" {
		return errors.New("empty checkpoint path")
	}
	cp := Checkpoint{Seq: seq, Messages: msgs, Created: time.Now().Unix()}
	b, err := json.Marshal(cp)
	if err != nil {
		return fmt.Errorf("marshal checkpoint: %w", err)
	}
	return fileutil.AtomicWrite(path, b, 0o644)
}

// WriteCheckpointFull 带重写版本写检查点（压缩协议用）。
func WriteCheckpointFull(path string, cp Checkpoint) error {
	if path == "" {
		return errors.New("empty checkpoint path")
	}
	cp.Created = time.Now().Unix()
	b, err := json.Marshal(cp)
	if err != nil {
		return fmt.Errorf("marshal checkpoint: %w", err)
	}
	return fileutil.AtomicWrite(path, b, 0o644)
}

// ReadCheckpoint 读回检查点。文件不存在时返回 nil, nil（无检查点 = 从日志头重放）。
// 损坏的检查点同样按「无检查点」处理（防御：检查点只是加速恢复，不应阻塞）。
func ReadCheckpoint(path string) (*Checkpoint, error) {
	if path == "" {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cp Checkpoint
	if err := json.Unmarshal(b, &cp); err != nil {
		return nil, nil
	}
	return &cp, nil
}

// Restore 恢复会话：checkpoint 消息 + 日志中 seq 之后的条目重放投影。
// 流程：
//  1. 读取 checkpoint（无则跳过）；
//  2. 修复 torn-tail（截断最后不完整行）；
//  3. 取 checkpoint.Seq 之后的完整条目，BalanceEntries 补合成 closers；
//  4. 投影增量并追加到 checkpoint 消息之后。
//
// 返回恢复后的消息、最后消费的 log seq 与错误。
func Restore(checkpointPath, logPath string) ([]provider.Message, int64, error) {
	var msgs []provider.Message
	base := int64(0)
	if cp, err := ReadCheckpoint(checkpointPath); err != nil {
		return nil, 0, err
	} else if cp != nil {
		msgs = append(msgs, cp.Messages...)
		base = cp.Seq
	}

	entries, err := ReadLogRepaired(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 日志不存在：只有 checkpoint（或两者皆无）。checkpoint 本身就是
			// 完整投影，可直接作为恢复结果。
			return msgs, base, nil
		}
		return nil, 0, err
	}
	var tail []LogEntry
	for _, e := range entries {
		if e.Seq > base {
			tail = append(tail, e)
		}
	}
	// 返回的游标是「最后消费的真实 log seq」，合成 closer 只存在于重放流中，
	// 不计入游标（否则下一次 checkpoint 会覆盖一条并不存在的行）。
	last := base
	if len(tail) > 0 {
		last = tail[len(tail)-1].Seq
	}
	tail = BalanceEntries(tail)
	msgs = append(msgs, ProjectMessages(tail)...)
	return msgs, last, nil
}

// FlushCheckpoint 是「模型调用前 flush 检查点（fail-closed）」的落盘动作：
// 把当前消息投影 + 已消费 seq 写入检查点。事件日志模式下，调用方在每轮
// 模型调用前调用它；失败返回错误（fail-closed），由调用方决定中止还是降级。
func FlushCheckpoint(checkpointPath string, msgs []provider.Message, consumedSeq int64) error {
	return WriteCheckpoint(checkpointPath, consumedSeq, msgs)
}
