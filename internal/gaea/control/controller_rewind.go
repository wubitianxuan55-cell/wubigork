package control

// controller_rewind.go — 会话回退/分叉（办公板块回退链路的后端实现）。
// 前端 rewind 菜单（both/conversation/fork/summ-*）久已存在但后端绑定一直
// 空实现（gaea_ui.go「办公引擎无 checkpoint 系统」注释已过时——3.0 Step 1
// 的事件日志 + 检查点机制就是回退的真相源）。本文件让会话回退真正可用：
//
//   - Rewind：截断事件日志到目标轮之前 + 删除检查点（全量重放）+ 重建会话接管；
//   - Fork：从目标轮分叉新会话文件（写日志 + 镜像 + 分支 meta 记录父分支）；
//   - Checkpoints：从日志派生可回退点列表（turn/prompt/files/time）。
//
// 事件日志模式下三者均基于「日志即真相」；legacy 模式 Rewind 直接截断消息。

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// cutIndexForTurn 返回「保留到第 turn 轮之前」的消息截断下标：
// 第 turn 个 user 消息的索引（turn 0 起）；不存在返回 -1。
// 保留 [0, idx) 即删除第 turn 轮及其后的所有消息（含 system 前缀）。
func cutIndexForTurn(msgs []provider.Message, turn int) int {
	if turn < 0 {
		return -1
	}
	seen := -1
	for i, m := range msgs {
		if m.Role == provider.RoleUser {
			seen++
			if seen == turn {
				return i
			}
		}
	}
	return -1
}

// Checkpoints 列出当前会话的可回退点（每个用户回合一个）。
// 事件日志模式从日志派生；legacy 模式从内存消息派生（无文件信息）。
func (c *Controller) Checkpoints() []session.TurnRange {
	c.mu.Lock()
	path := c.sessionPath
	exec := c.executor
	c.mu.Unlock()
	if path == "" || exec == nil {
		return nil
	}
	if c.EventMode() {
		entries, err := session.ReadLogRepaired(session.LogPathFor(path))
		if err != nil {
			return nil
		}
		return session.UserTurnRanges(entries)
	}
	msgs := exec.Session().Snapshot()
	var out []session.TurnRange
	turn := 0
	for _, m := range msgs {
		if m.Role == provider.RoleUser {
			out = append(out, session.TurnRange{Turn: turn, Prompt: m.Content})
			turn++
		}
	}
	return out
}

// Rewind 把当前会话回退到第 turn 轮之前（保留 turn < N 的所有消息）。
// 事件日志模式：截断日志到该轮 user_message 前一条 → 删除检查点（强制从
// 截断后的日志全量重放，避免检查点覆盖被删轮次）→ LoadWithFormat 重建 →
// Resume 接管。legacy 模式：内存消息 Truncate + Save。
// 要求 turn 存在且会话非运行中（调用方保证：回退按钮仅在非运行期可点）。
func (c *Controller) Rewind(turn int) error {
	c.mu.Lock()
	path := c.sessionPath
	exec := c.executor
	c.mu.Unlock()
	if path == "" || exec == nil {
		return errors.New("no active session")
	}
	s := exec.Session()
	msgs := s.Snapshot()
	cut := cutIndexForTurn(msgs, turn)
	if cut < 0 {
		return fmt.Errorf("rewind: turn %d not found (会话只有 %d 个回合)", turn, countTurns(msgs))
	}
	if !c.EventMode() {
		s.Truncate(cut)
		if err := s.Save(path); err != nil {
			return fmt.Errorf("rewind: save legacy session: %w", err)
		}
		return nil
	}
	logPath := session.LogPathFor(path)
	entries, err := session.ReadLogRepaired(logPath)
	if err != nil {
		return fmt.Errorf("rewind: read log: %w", err)
	}
	ranges := session.UserTurnRanges(entries)
	if turn >= len(ranges) {
		return fmt.Errorf("rewind: turn %d not found in log (共 %d 个回合)", turn, len(ranges))
	}
	keepSeq := ranges[turn].FirstSeq - 1 // 该轮 user_message 之前
	if err := session.RewindLog(logPath, keepSeq); err != nil {
		return fmt.Errorf("rewind: truncate log: %w", err)
	}
	// 检查点若覆盖了被删轮次会泄漏历史，直接删除（下次 flush 检查点重建）。
	_ = os.Remove(session.CheckpointPathFor(path))
	restored, err := session.LoadWithFormat(path, "event")
	if err != nil {
		return fmt.Errorf("rewind: restore session: %w", err)
	}
	// 同步镜像 .jsonl（事件模式 Save 只写镜像，日志已截断不动），
	// 让列表/历史读取与回退后的会话一致。
	if err := restored.Save(path); err != nil {
		return fmt.Errorf("rewind: save mirror: %w", err)
	}
	c.Resume(restored, path)
	c.SeedContextUsage()
	return nil
}

// Fork 从第 turn 轮之前分叉出一个新会话文件并接管为当前会话。
// 新会话 = 原会话保留 [0, turn 前) 的消息，写入新 .jsonl（含事件日志），
// 并落分支 meta（ParentID/ForkTurn 记录父分支，分支树可导航）。
// 返回新会话路径。
func (c *Controller) Fork(turn int) (string, error) {
	c.mu.Lock()
	path := c.sessionPath
	exec := c.executor
	label := c.label
	sys := c.systemPrompt
	f := c.logFormat
	c.mu.Unlock()
	if path == "" || exec == nil {
		return "", errors.New("no active session")
	}
	msgs := exec.Session().Snapshot()
	cut := cutIndexForTurn(msgs, turn)
	if cut < 0 {
		return "", fmt.Errorf("fork: turn %d not found (会话只有 %d 个回合)", turn, countTurns(msgs))
	}
	// 分支文件落在父会话同目录（sessionDir 可能未在控制构建时注入）。
	dir := filepath.Dir(path)
	if dir == "." {
		return "", errors.New("fork: 无法确定会话目录")
	}
	newPath := agent.NewSessionPath(dir, label+"-fork")
	forkMsgs := append([]provider.Message(nil), msgs[:cut]...)
	ns := agent.NewSession(sys)
	ns.SetLogFormat(f)
	ns.Replace(forkMsgs)
	if err := ns.Save(newPath); err != nil {
		return "", fmt.Errorf("fork: save branch: %w", err)
	}
	// 分支 meta：记录父分支与会话关系（ParentID = 父会话文件名）。
	meta := agent.BranchMeta{
		ID:        agent.BranchID(newPath),
		Name:      fmt.Sprintf("分叉 · 第 %d 轮", turn),
		ParentID:  agent.BranchID(path),
		ForkTurn:  turn,
		CreatedAt: time.Now().UTC(),
	}
	if err := agent.SaveBranchMeta(newPath, meta); err != nil {
		return "", fmt.Errorf("fork: save branch meta: %w", err)
	}
	c.Resume(ns, newPath)
	c.SeedContextUsage()
	return newPath, nil
}

func countTurns(msgs []provider.Message) int {
	n := 0
	for _, m := range msgs {
		if m.Role == provider.RoleUser {
			n++
		}
	}
	return n
}
