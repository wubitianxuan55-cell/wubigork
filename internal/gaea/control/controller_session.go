package control

// controller_session.go — 会话与上下文管理族（P5 破防专项拆自 controller.go：
// 新建/恢复会话、会话工具态复位、日志格式与 space 应用、checkpoint 落盘、
// 上下文用量与压缩比——搬移零功能变更）。

import (
	"context"
	"fmt"
	"strings"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/session"
	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// NewSession snapshots the current conversation, rotates to a fresh file, and
// resets the executor to a clean session carrying the same system prompt. It
// ends the old session and starts the new one for lifecycle hooks.
func (c *Controller) NewSession() error {
	if c.executor == nil {
		return nil
	}
	if err := c.Snapshot(); err != nil {
		return err
	}
	c.hooks.SessionEnd(context.Background())
	if c.sessionDir != "" {
		c.mu.Lock()
		c.sessionPath = agent.NewSessionPath(c.sessionDir, c.label)
		c.mu.Unlock()
	}
	ns := agent.NewSession(c.systemPrompt)
	c.mu.Lock()
	f := c.logFormat
	c.mu.Unlock()
	ns.SetLogFormat(f)
	// S2：新会话空间自描述按新路径归属折算（sessionDir 已由宿主按配置分区）。
	ns.SetSpace(c.writeSpaceFor(c.SessionPath()))
	c.executor.SetSession(ns)
	c.applyLogFormat(f)
	c.applySpace(c.space)
	// Reset V3.0 TCCA state so the new session starts clean.
	if c.ctxMgr != nil {
		c.ctxMgr.Flow().ReplaceMessages(nil)
	}
	// v4.378：清空持有会话级状态的工具（bash 状态锚），防 cwd/env 跨会话泄漏。
	c.resetSessionToolState()
	c.mu.Lock()
	c.startedOnce = true // NewSession fires SessionStart itself; don't re-fire on the next turn
	c.mu.Unlock()
	c.hooks.SessionStart(context.Background())
	return nil
}

// resetSessionToolState clears per-session mutable state on tools that opt in
// via tool.SessionStateResetter (currently the bash shell anchor). Nil-safe and
// a no-op for tools that don't implement it.
func (c *Controller) resetSessionToolState() {
	for _, t := range c.Tools() {
		if r, ok := t.(tool.SessionStateResetter); ok {
			r.ResetSessionState()
		}
	}
}

// RewindScope selects what a Rewind restores.
// Resume seeds the session from a loaded transcript and pins the active file to
// its path so auto-save keeps appending there.
func (c *Controller) Resume(s *agent.Session, path string) {
	if c.executor != nil {
		c.executor.SetSession(s)
	}
	c.mu.Lock()
	c.sessionPath = path
	f := c.logFormat
	c.mu.Unlock()
	// 3.0 Step 1：恢复的会话继承当前持久化格式（事件日志模式下 Save 双写、
	// 后续回合落用户消息 + flush 检查点都依赖该标记）。
	s.SetLogFormat(f)
	// S2：恢复会话的空间自描述按「路径归属」折算（play 分区会话即使当前
	// 配置为 work 也写 play，保证日志/检查点与目录归属一致、可通过恢复校验）。
	s.SetSpace(c.writeSpaceFor(path))
	if c.executor != nil {
		c.executor.SetCheckpointFlusher(c.flushCheckpoint)
	}
	// v4.378：换会话即清 bash 状态锚（恢复的是对话历史，shell 锚定态从不
	// 持久化——磁盘上没有它的真相，重置是唯一诚实态）。
	c.resetSessionToolState()
}

// ResumeFromDisk 从磁盘恢复会话并接管为当前会话。事件日志模式下：
// DetectLegacy → 旧格式先迁移 → Restore（checkpoint 消息 + log tail 重放），
// 无日志时回退 legacy Load；legacy 模式保持原 LoadSession + Resume 行为。
// 返回恢复后的会话（已注入 c.Resume）。
func (c *Controller) ResumeFromDisk(path string) (*agent.Session, error) {
	var (
		s   *agent.Session
		err error
	)
	if !c.EventMode() {
		s, err = agent.LoadSession(path)
		if err != nil {
			return nil, err
		}
	} else {
		// 旧格式会话（无事件日志而有 <id>.jsonl）先迁移，旧文件保留。
		// S2：迁移条目按会话目录归属携带空间自描述（play 分区迁移必须带
		// play，否则恢复校验按空间穿越拒绝）。
		if legacy, legacyPath, derr := session.DetectLegacy(path); derr != nil {
			return nil, derr
		} else if legacy {
			if _, merr := session.MigrateLegacyToLog(session.LogPathFor(path), legacyPath, c.writeSpaceFor(path)); merr != nil {
				return nil, fmt.Errorf("resume: migrate legacy session: %w", merr)
			}
		}
		s, err = session.LoadWithFormat(path, "event")
		if err != nil {
			return nil, err
		}
	}
	c.Resume(s, path)
	return s, nil
}

// EventMode 报告控制器是否处于事件日志模式（session.log_format="event"）。
func (c *Controller) EventMode() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return strings.EqualFold(c.logFormat, "event")
}

// SetLogFormat 设置会话持久化格式（"legacy"/""=旧行为，"event"=事件日志），
// 并同步注入当前会话与检查点 flush 钩子。boot.Build 后由宿主（app）按配置
// 注入；缺省（空）时所有事件日志接线均为 no-op，行为与改造前一致。
func (c *Controller) SetLogFormat(f string) { c.applyLogFormat(f) }

// applyLogFormat 把持久化格式写入控制器并传播到当前会话与 executor 的
// 检查点 flush 钩子（fail-closed：模型调用前由 executor.FlushCheckpointFailClosed
// 触发，失败中止回合）。
func (c *Controller) applyLogFormat(f string) {
	c.mu.Lock()
	c.logFormat = f
	exec := c.executor
	c.mu.Unlock()
	if exec != nil {
		exec.Session().SetLogFormat(f)
		exec.SetCheckpointFlusher(c.flushCheckpoint)
	}
}

// SetSpace 设置会话空间的配置生效值（S2 双空间）："work"/"play"=分区空间，
// ""=space.mode=off。与 SetLogFormat 同点注入（boot.Build 后由宿主按配置调用）。
func (c *Controller) SetSpace(space string) { c.applySpace(space) }

// applySpace 把空间配置生效值写入控制器并按「路径归属」折算后传播到当前
// 会话（仿 applyLogFormat 的注入点）。
func (c *Controller) applySpace(space string) {
	c.mu.Lock()
	c.space = space
	// 空间切换使在途记忆快照过期（刀B v4.246）：在途 Load 完成后按代号被
	// 拒绝换入，下次写路径以新空间重载（与改前一致：applySpace 本就不刷新）。
	if c.mem != nil {
		c.memGen++
	}
	path := c.sessionPath
	exec := c.executor
	spaceForPath := c.writeSpaceForLocked(path)
	c.mu.Unlock()
	if exec != nil {
		exec.Session().SetSpace(spaceForPath)
	}
}

// writeSpaceForLocked 计算会话路径对应的写入侧空间自描述值（日志行/检查点/
// 分支 meta 用）。已持有 c.mu（或初始化单线程上下文）时调用。规则：
//   - play 分区目录 → 恒 "play"：即使 space.mode=off 也保持自描述，否则
//     off 时代恢复 play 存量会话会写入无 space 的行，混合空间日志将被恢复
//     校验拒绝（fail-closed 防穿越）；
//   - work 分区/平铺目录 → space.mode=on 时 "work"；off 时 ""（日志不写
//     space 字段，与旧行为逐字节一致；读端空值降级 work，行为等价）。
//
// 路径归属是唯一真相源（session.SpaceForPath）：配置空间只决定「off 与否」，
// 不直接决定写入值——恢复的 play 会话在 work 配置下仍按目录归属写 play。
func (c *Controller) writeSpaceForLocked(path string) string {
	sp := session.SpaceForPath(path)
	if sp == spaces.SpacePlay {
		return spaces.SpacePlay
	}
	if c.space == "" {
		return "" // space.mode=off：平铺日志不写 space 字段（旧行为形态）
	}
	return spaces.SpaceWork
}

// writeSpaceFor 是 writeSpaceForLocked 的自带锁版本（无锁上下文调用）。
func (c *Controller) writeSpaceFor(path string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writeSpaceForLocked(path)
}

// SessionSpace 返回当前会话的写入侧空间自描述值（事件日志 sink 的懒解析
// 来源；""=space.mode=off 平铺日志不写 space 字段）。
func (c *Controller) SessionSpace() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.writeSpaceForLocked(c.sessionPath)
}

// flushCheckpoint 把当前会话消息投影 + 已消费的 log seq 写入检查点
// （事件日志模式）。无会话路径 / 无内容 / 尚无日志条目时不写（幂等）。
// 返回错误供 fail-closed 调用方（模型调用前）中止回合。
func (c *Controller) flushCheckpoint() error {
	c.mu.Lock()
	path := c.sessionPath
	exec := c.executor
	spaceForPath := c.writeSpaceForLocked(path)
	c.mu.Unlock()
	if path == "" || exec == nil {
		return nil
	}
	s := exec.Session()
	if !s.IsEventMode() || !s.HasContent() {
		return nil
	}
	logPath := session.LogPathFor(path)
	if logPath == "" {
		return nil
	}
	seq := session.LastLogSeq(logPath)
	if seq == 0 {
		return nil // 尚无日志条目（本会话还没有任何事件），无可固化内容
	}
	// S2：检查点携带空间自描述做对账（与日志行同源）。
	return session.WriteCheckpoint(session.CheckpointPathFor(path), seq, s.Snapshot(), spaceForPath)
}

// logUserMessage 把本回合用户消息追加到事件日志（「模型可见必入日志」：
// 运行期 user_message 由控制器在模型调用前落盘）。日志缺失时自动创建
// （旧格式会话先迁移）。失败返回错误（fail-closed）。
func (c *Controller) logUserMessage(input string) error {
	path := c.SessionPath()
	if path == "" {
		return nil
	}
	logPath := session.LogPathFor(path)
	if logPath == "" {
		return nil
	}
	// S2：用户消息行携带空间自描述（与日志所在目录归属一致）。
	if _, err := session.AppendUserMessage(logPath, path, c.writeSpaceFor(path), input); err != nil {
		return fmt.Errorf("append user message to event log: %w", err)
	}
	return nil
}

// Snapshot writes the executor's conversation to the active session file. No-op
// when persistence is unavailable or the session has never been used (no user
// interaction). Called after every turn so a crash loses at most one in-flight
// prompt.
func (c *Controller) Snapshot() error {
	c.mu.Lock()
	path := c.sessionPath
	c.mu.Unlock()
	if c.executor == nil || path == "" {
		return nil
	}
	s := c.executor.Session()
	if !s.HasContent() {
		return nil
	}
	if err := s.Save(path); err != nil {
		return err
	}
	if err := agent.TouchBranchMeta(path); err != nil {
		return err
	}
	// 3.0 Step 1 事件日志模式：回合结束后 flush 检查点（含压缩后的消息
	// 投影 + 已消费 log seq），断电/崩溃后可由 checkpoint + log tail 恢复。
	if s.IsEventMode() {
		if err := c.flushCheckpoint(); err != nil {
			return err
		}
	}
	return nil
}

// SetSessionPath pins where auto-save lands (a fresh session file minted by the
// caller when no resume path applies).
func (c *Controller) SetSessionPath(p string) {
	c.mu.Lock()
	c.sessionPath = p
	f := c.logFormat
	spaceForPath := c.writeSpaceForLocked(p)
	exec := c.executor
	c.mu.Unlock()
	if exec != nil {
		exec.Session().SetLogFormat(f)
		// S2：空间自描述随落点路径归属折算注入当前会话。
		exec.Session().SetSpace(spaceForPath)
		exec.SetCheckpointFlusher(c.flushCheckpoint)
	}
}

// SessionDir reports the directory new session files land in ("" disables
// persistence), so the caller can decide whether to mint a path.
func (c *Controller) SessionDir() string { return c.sessionDir }

// SessionPath reports the file the current conversation auto-saves to ("" when
// persistence is disabled), so a history view can mark the active session.
func (c *Controller) SessionPath() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sessionPath
}
