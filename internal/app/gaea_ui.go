package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/session"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/factbase"
	"github.com/gaea/gaea/internal/gaea/fileutil"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// ── gaeaW 原生 UI 绑定（前端 gaea/lib/bridge.ts 适配层映射短名 → Gaea*）──
// 类型与实现对齐 gaeaW desktop/app_*.go，保证办公板块 UI 无改动可用。

// HistoryMessage 是一轮对话消息（历史面板/恢复会话用）。
type HistoryMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	// 工具事件还原（Kun 可观察性）：恢复会话后过程卡与「变更」面板仍可见。
	// dispatch 条目 Role="tool"（携带 name/args/id），结果条目 Role="tool_result"
	// 携带同 id 的 output，前端按 id 合并为完成态工具卡片。
	ToolName   string `json:"toolName,omitempty"`
	ToolArgs   string `json:"toolArgs,omitempty"`
	ToolID     string `json:"toolId,omitempty"`
	ToolOutput string `json:"toolOutput,omitempty"`
}

// SessionMeta 是一个已保存会话（历史面板列表项）。
type SessionMeta struct {
	Path    string `json:"path"`
	Preview string `json:"preview"`
	Title   string `json:"title,omitempty"`
	Turns   int    `json:"turns"`
	ModTime int64  `json:"modTime"`
	Current bool   `json:"current"`
	Pinned  bool   `json:"pinned"`
	// Archived 为 true 表示会话在 <sessions>/archive/ 下（可恢复，不参与列表排序）。
	Archived bool `json:"archived"`
	// Interrupted 为 true 表示上次会话因崩溃/进程被杀未正常结束（state 文件
	// 残留 running=true），前端据此展示「未完成」徽标。
	Interrupted bool `json:"interrupted"`
}

// CheckpointMeta 是一个可回退点（用户回合）。
type CheckpointMeta struct {
	Turn   int      `json:"turn"`
	Prompt string   `json:"prompt"`
	Files  []string `json:"files"`
	Time   int64    `json:"time"`
}

var errActiveSession = errors.New("can't delete the session you're in — start a new one first")

// gaeaCtrl 返回当前办公控制器（未初始化返回 nil）。
func gaeaCtrl() *control.Controller {
	ga.mu.Lock()
	defer ga.mu.Unlock()
	return ga.ctrl
}

// ── 会话 ─────────────────────────────────────────────────────────

// GaeaApprove 审批/拒绝一个待批工具调用；abort=true 表示「拒绝并终止本轮」。
func (a *App) GaeaApprove(id string, allow, session, abort bool) {
	if c := gaeaCtrl(); c != nil {
		c.Approve(id, allow, session, abort)
	}
}

// GaeaAnswer 回答结构化提问。
func (a *App) GaeaAnswer(id string, answers []event.AskAnswer) {
	if c := gaeaCtrl(); c != nil {
		c.AnswerQuestion(id, answers)
	}
}

// GaeaHistory 返回当前会话的对话历史。
func (a *App) GaeaHistory() []HistoryMessage {
	out := []HistoryMessage{}
	c := gaeaCtrl()
	if c == nil {
		return out
	}
	for _, m := range c.History() {
		switch m.Role {
		case provider.RoleTool:
			out = append(out, HistoryMessage{
				Role:       "tool_result",
				Content:    m.Content,
				ToolName:   m.Name,
				ToolID:     m.ToolCallID,
				ToolOutput: m.Content,
			})
		default:
			out = append(out, HistoryMessage{Role: string(m.Role), Content: m.Content})
			// assistant 消息携带的工具调用：逐个还原为 dispatch 条目，
			// 让恢复后的过程卡与变更面板和实时会话一致。
			if m.Role == provider.RoleAssistant {
				for _, tc := range m.ToolCalls {
					out = append(out, HistoryMessage{
						Role:     "tool",
						ToolName: tc.Name,
						ToolArgs: tc.Arguments,
						ToolID:   tc.ID,
					})
				}
			}
		}
	}
	return out
}

const pinnedFileName = ".pinned.json"

func pinnedPath(dir string) string { return filepath.Join(dir, pinnedFileName) }

// loadPinned 读取工作区会话目录的置顶注册表（base name → true）。
func loadPinned(dir string) map[string]bool {
	m := map[string]bool{}
	b, err := os.ReadFile(pinnedPath(dir))
	if err != nil {
		return m
	}
	if err := json.Unmarshal(b, &m); err != nil {
		// 注册表文件损坏时按空处理并记录，避免静默丢失用户置顶
		slog.Warn("置顶注册表解析失败（按空处理）", "path", pinnedPath(dir), "error", err)
	}
	return m
}

func savePinned(dir string, m map[string]bool) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return saveAtomically(dir, ".pinned.*.tmp", pinnedPath(dir), m)
}

// listSessionsForDir 构建一个工作区会话目录的 SessionMeta 列表。
// includeNewFallback 为 true 时，把当前未落盘的会话补为「(新会话)」条目；
// 仅当前工作区需要该回退，历史项目目录不需要。排序：当前会话 → 置顶 → 最近使用。
func (a *App) listSessionsForDir(dir string, cur string, includeNewFallback bool) []SessionMeta {
	infos, err := agent.ListSessions(dir)
	if err != nil {
		return []SessionMeta{}
	}
	titles := loadSessionTitles(dir)
	pinned := loadPinned(dir)
	out := make([]SessionMeta, 0, len(infos)+1)
	curFound := false
	for _, s := range infos {
		if s.Path == cur {
			curFound = true
		}
		base := filepath.Base(s.Path)
		// 中断标记：state 文件残留 running=true 说明上次会话未正常结束。
		// 状态文件很小，会话数在 50 以内全读没有开销问题。
		st, _ := session.LoadState(session.StatePath(s.Path))
		out = append(out, SessionMeta{
			Path:            s.Path,
			Preview:         s.Preview,
			Title:           titles[base],
			Turns:           s.Turns,
			ModTime:         s.ModTime.UnixMilli(),
			Current:         s.Path == cur,
			Pinned:          pinned[base],
			Interrupted:     st.Running,
		})
	}
	if cur != "" && !curFound && includeNewFallback {
		out = append(out, SessionMeta{Path: cur, Preview: "(新会话)", ModTime: time.Now().UnixMilli(), Current: true, Pinned: true})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Current != out[j].Current {
			return out[i].Current
		}
		if out[i].Pinned != out[j].Pinned {
			return out[i].Pinned
		}
		return out[i].ModTime > out[j].ModTime
	})
	return out
}

// archivedSessionsForDir 列出 <dir>/archive 下的已归档会话（新→旧），
// 供侧边栏「已归档」分组展示与恢复。
func (a *App) archivedSessionsForDir(dir string) []SessionMeta {
	infos, err := agent.ListArchivedSessions(dir)
	if err != nil {
		return []SessionMeta{}
	}
	titles := loadSessionTitles(dir)
	out := make([]SessionMeta, 0, len(infos))
	for _, s := range infos {
		st, _ := session.LoadState(session.StatePath(s.Path))
		out = append(out, SessionMeta{
			Path:            s.Path,
			Preview:         s.Preview,
			Title:           titles[filepath.Base(s.Path)],
			Turns:           s.Turns,
			ModTime:         s.ModTime.UnixMilli(),
			Archived:        true,
			Interrupted:     st.Running,
		})
	}
	return out
}

// GaeaListSessions 返回当前工作区的已保存会话（新→旧），标记当前会话。
// 会话写入统一在 gaeaCwd()/.gaea/sessions（见 gaeaBuildController），
// 这里必须用同一路径读取，否则从不同目录启动时历史会"消失"。
func (a *App) GaeaListSessions() []SessionMeta {
	cur := ""
	if c := gaeaCtrl(); c != nil {
		cur = c.SessionPath()
	}
	return a.listSessionsForDir(gaeaConfig.WorkspaceSessionDir(gaeaCwd()), cur, true)
}

// ProjectGroup 是侧边栏「项目」分组：一个工作区 + 它的会话列表。
type ProjectGroup struct {
	Path     string        `json:"path"`
	Name     string        `json:"name"`
	Current  bool          `json:"current"`
	Sessions []SessionMeta `json:"sessions"`
	Archived []SessionMeta `json:"archived"`
	ModTime  int64         `json:"modTime"`
}

// maxProjectSessionsPerGroup 防止超大项目把侧边栏请求撑爆；
// 前端在分组内提供「显示更多」展开剩余会话（不重复请求）。
const maxProjectSessionsPerGroup = 50

// GaeaListProjectSessions 按项目聚合会话（Codex/Kun 风）：当前工作区在前，
// 其余为最近打开过的工作区（仅包含仍存在且有会话的）。供侧边栏「项目」视图。
func (a *App) GaeaListProjectSessions() []ProjectGroup {
	cur := gaeaCwd()
	roots := []string{cur}
	seen := map[string]bool{cur: true}
	for _, p := range gaeaConfig.LoadRecentWorkspaces() {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		roots = append(roots, p)
	}
	curSession := ""
	if c := gaeaCtrl(); c != nil {
		curSession = c.SessionPath()
	}
	groups := make([]ProjectGroup, 0, len(roots))
	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}
		sessions := a.listSessionsForDir(gaeaConfig.WorkspaceSessionDir(root), curSession, root == cur)
		archived := a.archivedSessionsForDir(gaeaConfig.WorkspaceSessionDir(root))
		if root != cur && len(sessions) == 0 && len(archived) == 0 {
			continue
		}
		if len(sessions) > maxProjectSessionsPerGroup {
			sessions = sessions[:maxProjectSessionsPerGroup]
		}
		if len(archived) > maxProjectSessionsPerGroup {
			archived = archived[:maxProjectSessionsPerGroup]
		}
		mod := int64(0)
		for _, s := range sessions {
			if s.ModTime > mod {
				mod = s.ModTime
			}
		}
		for _, s := range archived {
			if s.ModTime > mod {
				mod = s.ModTime
			}
		}
		if root == cur && mod == 0 {
			mod = time.Now().UnixMilli()
		}
		groups = append(groups, ProjectGroup{
			Path:     root,
			Name:     filepath.Base(root),
			Current:  root == cur,
			Sessions: sessions,
			Archived: archived,
			ModTime:  mod,
		})
	}
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].Current != groups[j].Current {
			return groups[i].Current
		}
		return groups[i].ModTime > groups[j].ModTime
	})
	return groups
}

// GaeaDeleteSession 删除已保存会话（拒绝删除当前会话）。
func (a *App) GaeaDeleteSession(path string) error {
	if c := gaeaCtrl(); c != nil && c.SessionPath() == path {
		return errActiveSession
	}
	dir := sessionDirForPath(path)
	if dir == "" {
		return fmt.Errorf("非法会话路径: %s", path)
	}
	return deleteSessionFile(dir, path)
}

// GaeaArchiveSession 归档已保存会话（移动至 <sessions>/archive/，可恢复）。
// 拒绝归档当前会话——当前会话由「新建会话」管理生命周期。
func (a *App) GaeaArchiveSession(path string) error {
	if c := gaeaCtrl(); c != nil && c.SessionPath() == path {
		return errActiveSession
	}
	dir := sessionDirForPath(path)
	if dir == "" || filepath.Base(dir) != "sessions" {
		return fmt.Errorf("非法会话路径: %s", path)
	}
	if err := agent.ArchiveSession(path); err != nil {
		return err
	}
	// 归档后清除置顶标记，恢复时重新置顶由用户决定
	pinned := loadPinned(dir)
	base := filepath.Base(path)
	if pinned[base] {
		delete(pinned, base)
		_ = savePinned(dir, pinned)
	}
	return nil
}

// GaeaUnarchiveSession 把归档会话移回会话目录，返回恢复后的活动路径。
func (a *App) GaeaUnarchiveSession(path string) (string, error) {
	dir := sessionDirForPath(path)
	if dir == "" || filepath.Base(dir) != "archive" {
		return "", fmt.Errorf("非法归档路径: %s", path)
	}
	if err := agent.UnarchiveSession(path); err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(dir), filepath.Base(path)), nil
}

// GaeaPinSession 置顶/取消置顶一个活动会话（持久化到 .pinned.json）。
func (a *App) GaeaPinSession(path string, pinned bool) error {
	dir := sessionDirForPath(path)
	if dir == "" || filepath.Base(dir) != "sessions" {
		return fmt.Errorf("仅活动会话可置顶: %s", path)
	}
	m := loadPinned(dir)
	base := filepath.Base(path)
	if pinned {
		m[base] = true
	} else {
		delete(m, base)
	}
	return savePinned(dir, m)
}

// GaeaRenameSession 设置会话自定义名称（空清除）。
func (a *App) GaeaRenameSession(path, title string) error {
	dir := sessionDirForPath(path)
	if dir == "" {
		return fmt.Errorf("非法会话路径: %s", path)
	}
	return setSessionTitle(dir, path, title)
}

// sessionDirForPath 从绝对会话路径反推其所属会话目录，仅接受
// <root>/.gaea/sessions/<file> 或 <root>/.gaea/sessions/archive/<file>
// 形态，防止删除/重命名/归档逃出会话目录。
func sessionDirForPath(sessionPath string) string {
	dir := filepath.Dir(filepath.Clean(sessionPath))
	if filepath.Base(dir) == "archive" {
		// <root>/.gaea/sessions/archive/<file>
		if filepath.Base(filepath.Dir(dir)) != "sessions" ||
			filepath.Base(filepath.Dir(filepath.Dir(dir))) != ".gaea" {
			return ""
		}
		return dir
	}
	if filepath.Base(dir) != "sessions" || filepath.Base(filepath.Dir(dir)) != ".gaea" {
		return ""
	}
	return dir
}

// SessionStatsView 是会话级 token/成本派生统计（3.0 Step 1 事件日志重放）。
// Available=false 表示该会话无事件日志（legacy 会话或路径非法），前端不展示
// 历史统计块（避免全 0 误导）。
type SessionStatsView struct {
	Available bool               `json:"available"`
	Stats     session.TokenStats `json:"stats"`
}

// GaeaSessionStats 从会话事件日志派生 token/成本统计（DeriveStats 重放 usage
// 事件）。恢复会话后前端调用它回填统计面板，修复「恢复的长会话成本展示不全」
// （评审 03-office-frontend.md 缺陷 11：StatsPanel 只累计本次窗口内 usage 事件，
// 不回溯历史）。路径经 sessionDirForPath 校验（防穿越），读失败/无日志返回
// Available=false，不阻塞恢复流程。
func (a *App) GaeaSessionStats(path string) SessionStatsView {
	if path == "" || sessionDirForPath(path) == "" {
		return SessionStatsView{}
	}
	lp := session.LogPathFor(path)
	if lp == "" {
		return SessionStatsView{}
	}
	entries, err := session.ReadLogRepaired(lp)
	if err != nil {
		// 无日志文件（legacy 会话）或读取失败：不视为统计为 0，标记不可用。
		return SessionStatsView{}
	}
	return SessionStatsView{Available: true, Stats: session.DeriveStats(entries)}
}

// GaeaResumeSession 快照当前会话并加载目标会话继续，返回其消息。
func (a *App) GaeaResumeSession(path string) ([]HistoryMessage, error) {	// 引擎未初始化时先初始化（幂等），避免重启后首次点击会话报"引擎未初始化"
	if err := a.GaeaInit(); err != nil {
		return nil, err
	}
	c := gaeaCtrl()
	if c == nil {
		return nil, errors.New("办公引擎未初始化")
	}
	// 3.0 Step 1 事件日志模式：恢复走 Restore（先 DetectLegacy 迁移，再
	// checkpoint + log tail 重放）；legacy 模式保持原 LoadSession + Resume。
	_ = c.Snapshot() // 切换前持久化当前会话（事件日志模式含检查点）
	loaded, err := c.ResumeFromDisk(path)
	if err != nil {
		return nil, err
	}
	// 中断恢复：上次进程崩溃/被杀时 state 残留 running=true，向消息末尾
	// 追加一条 system 摘要让模型先总结进度再继续，并清除标记避免重复提示。
	// 事件日志模式下该摘要同步写入事件日志（「模型可见必入日志」）。
	injectInterruption(loaded, path)
	c.SeedContextUsage() // 恢复后上下文读数立即反映该会话，而非 0
	return a.GaeaHistory(), nil
}

// injectInterruption 检查会话的中断状态（state 文件残留 running=true）。
// 若上次会话未完成，则在消息末尾追加一条中断摘要 system 消息，并清除
// state 文件；非中断会话不注入、不触碰 state 文件。
func injectInterruption(s *agent.Session, path string) {
	stPath := session.StatePath(path)
	st, err := session.LoadState(stPath)
	if err != nil || !st.Running {
		return
	}
	loc := st.Summary
	if loc == "" {
		loc = lastUserPreview(s)
	}
	msg := provider.Message{
		Role:    provider.RoleSystem,
		Content: interruptionMessage(loc),
	}
	s.Messages = append(s.Messages, msg)
	// 3.0 Step 1 事件日志模式：注入的中断摘要也写入事件日志（重放恢复不丢失；
	// 写入失败仅告警，不阻断恢复——legacy 镜像仍保留该消息）。
	if s.IsEventMode() {
		if lp := session.LogPathFor(path); lp != "" {
			if _, err := session.AppendSystemMessage(lp, path, msg.Content); err != nil {
				slog.Warn("gaea: 中断摘要写入事件日志失败", "path", path, "error", err)
			}
		}
	}
	_ = session.ClearState(stPath)
}

// interruptionMessage 构造恢复中断会话时注入的 system 消息。loc 为中断位置
// （state 摘要或最后用户消息预览），为空时省略引号部分。
func interruptionMessage(loc string) string {
	loc = strings.TrimSpace(loc)
	if loc == "" {
		return "上次会话中断。请先简要总结已完成进度与当前状态，再继续原任务。"
	}
	return "上次会话在「" + loc + "」处中断。请先简要总结已完成进度与当前状态，再继续原任务。"
}

// lastUserPreview 返回会话中最后一条用户消息的文本（截断到 80 字符），
// 作为 state 摘要为空时中断位置的回退。
func lastUserPreview(s *agent.Session) string {
	msgs := s.Snapshot()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == provider.RoleUser && strings.TrimSpace(msgs[i].Content) != "" {
			t := strings.TrimSpace(msgs[i].Content)
			if r := []rune(t); len(r) > 80 {
				return string(r[:77]) + "…"
			}
			return t
		}
	}
	return ""
}

// resumeLastSession 自动恢复会话目录中最近一次有内容的会话。
// 仅在办公引擎首次初始化时调用；返回最近会话路径（无可恢复会话时为空）。
func (a *App) resumeLastSession(ctrl *control.Controller) string {
	dir := gaeaConfig.WorkspaceSessionDir(gaeaCwd())
	infos, err := agent.ListSessions(dir)
	if err != nil || len(infos) == 0 {
		return ""
	}
	// 3.0 Step 1 事件日志模式：自动恢复同样走 Restore（DetectLegacy 迁移 →
	// checkpoint + log tail）；legacy 模式保持原 LoadSession + Resume。
	loaded, err := ctrl.ResumeFromDisk(infos[0].Path)
	if err != nil {
		slog.Warn("gaea: 自动恢复最近会话失败", "path", infos[0].Path, "error", err)
		return ""
	}
	// 崩溃后重启的自动恢复同样注入中断摘要并清除 state 标记，
	// 避免「未完成」徽标残留在当前正在对话的会话上。
	injectInterruption(loaded, infos[0].Path)
	ctrl.SeedContextUsage()
	slog.Info("gaea: 已自动恢复最近会话", "path", infos[0].Path, "messages", len(loaded.Messages))
	return infos[0].Path
}

// GaeaCheckpoints 列出会话可回退点（每个用户回合一个：turn/prompt/files/time）。
// 事件日志模式从日志派生；legacy 模式从内存消息派生。引擎未初始化返回空。
func (a *App) GaeaCheckpoints() []CheckpointMeta {
	c := gaeaCtrl()
	if c == nil {
		return nil
	}
	ranges := c.Checkpoints()
	out := make([]CheckpointMeta, 0, len(ranges))
	for _, r := range ranges {
		out = append(out, CheckpointMeta{Turn: r.Turn, Prompt: r.Prompt, Files: r.Files, Time: r.Time})
	}
	return out
}

// GaeaRewind 回退到指定回合。scope 语义（与前端 rewind 菜单对齐）：
//   - conversation / both：回退会话（both 下文件无独立版本系统，文件随会话
//     产物自然撤销，故与 conversation 同行为）；
//   - fork：从该回合分叉新会话；
//   - code / summ-from / summ-upto：明确报错（文件快照 / 摘要协议未支持）。
func (a *App) GaeaRewind(turn int, scope string) error {
	c := gaeaCtrl()
	if c == nil {
		return errors.New("办公引擎未初始化")
	}
	switch scope {
	case "fork":
		if _, err := c.Fork(turn); err != nil {
			return err
		}
		return nil
	case "code":
		return errors.New("文件级回退需要文件版本快照，暂未支持；可用「仅对话」回退会话")
	case "summ-from", "summ-upto":
		return errors.New("摘要回退暂未支持")
	default: // conversation / both
		return c.Rewind(turn)
	}
}

// GaeaFork 从指定回合分叉出新会话并接管为当前会话。
func (a *App) GaeaFork(turn int) error {
	c := gaeaCtrl()
	if c == nil {
		return errors.New("办公引擎未初始化")
	}
	_, err := c.Fork(turn)
	return err
}

// GaeaSummarizeFrom/GaeaSummarizeUpTo 摘要回退暂未支持（压缩协议需 LLM 摘要
// 服务，留给后续）；保留签名返回明确错误，避免前端误以为成功。
func (a *App) GaeaSummarizeFrom(turn int) error { return errSummarizeUnsupported }
func (a *App) GaeaSummarizeUpTo(turn int) error { return errSummarizeUnsupported }

var errSummarizeUnsupported = errors.New("摘要回退暂未支持")

// ── 会话辅助（照搬 gaeaW desktop/sessions.go）──────────────────────

const sessionTitlesFile = ".titles.json"

func sessionTitlesPath(dir string) string { return filepath.Join(dir, sessionTitlesFile) }

func loadSessionTitles(dir string) map[string]string {
	m := map[string]string{}
	b, err := os.ReadFile(sessionTitlesPath(dir))
	if err != nil {
		return m
	}
	if err := json.Unmarshal(b, &m); err != nil {
		slog.Warn("会话标题注册表解析失败（按空处理）", "path", sessionTitlesPath(dir), "error", err)
	}
	return m
}

func saveAtomically(dir, pattern, path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return fileutil.AtomicWrite(path, b, 0o644)
}

func setSessionTitle(dir, sessionPath, title string) error {
	m := loadSessionTitles(dir)
	key := filepath.Base(sessionPath)
	if strings.TrimSpace(title) == "" {
		delete(m, key)
	} else {
		m[key] = strings.TrimSpace(title)
	}
	return saveAtomically(dir, ".titles.*.tmp", sessionTitlesPath(dir), m)
}

func deleteSessionFile(dir, sessionPath string) error {
	// 防御：仅允许删除会话目录内的文件，防止路径逃逸。
	if filepath.Dir(filepath.Clean(sessionPath)) != filepath.Clean(dir) {
		return fmt.Errorf("session outside session dir: %s", sessionPath)
	}
	key := filepath.Base(sessionPath)
	if err := os.Remove(sessionPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	// 事实底座与会话同生共死：删除会话时顺带清理 <session>-facts.json。
	if factsPath := factbase.PathFor(sessionPath); factsPath != "" {
		_ = os.Remove(factsPath)
	}
	// 注册表（标题 / 置顶）统一放在会话目录；归档子目录删除时向上取父目录。
	registryDir := dir
	if filepath.Base(registryDir) == "archive" {
		registryDir = filepath.Dir(registryDir)
	}
	titles := loadSessionTitles(registryDir)
	if _, ok := titles[key]; ok {
		delete(titles, key)
		if err := saveAtomically(registryDir, ".titles.*.tmp", sessionTitlesPath(registryDir), titles); err != nil {
			return err
		}
	}
	pinned := loadPinned(registryDir)
	if pinned[key] {
		delete(pinned, key)
		if err := savePinned(registryDir, pinned); err != nil {
			return err
		}
	}
	return nil
}
