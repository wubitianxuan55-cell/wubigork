package app

// wenxue_chat.go — 图文板块会话态绑定（2026-11-15 刀1，案A）。
//
// 图文独立引擎（wx）的会话态面：发送/插话/取消/审批/提问/历史/恢复/回退/
// 分叉/归档/置顶/改名、上下文与轨迹看板、事件补拉、命令与技能清单、记忆面板、
// 权限档位、子代理追问。方法体与办公对应件同构（路径无关 helper 复用），控制
// 器锚从 gaeaCtrl() 换 wenxueCtrl()、事件 wire 换 wx.wire、引擎初始化换
// WenxueEngineInit。办公对应件零改动（红线）。
//
// 刻意不 fork 的共享面（读写全局注册表/显式路径参数，无会话态）：
// Settings/SetModel/SetAgentParams（全局配置，经 wenxueReloadIfInit 跟随重建）、
// MCP 配置族、SkillDistill*/SkillStats、TaskInbox 族、SessionStats/
// DeliverableRegistry/SubagentRuns/SubagentTranscript（显式 sessionPath 路径
// 锚定，图文会话路径同样可读）。

import (
	"context"
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
	"github.com/gaea/gaea/internal/gaea/contextview"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/i18n"
	"github.com/gaea/gaea/internal/gaea/trajectory"
)

// ── 回合生命周期 ────────────────────────────────────────────────

// WenxueSend 提交图文对话（异步，事件经 gaea-wenxue-event）。未初始化时自动初始化。
// 每轮发送前过本地引擎预检（wenxue_engine_preflight.go）：引擎可能在任意
// 两轮之间被空闲收割器停机，预检不限于首轮；预检失败转 error 事件，绝不
// 静默落进 connection refused → 故障转移换引擎。
func (a *App) WenxueSend(input string) {
	wx.mu.Lock()
	needBoot := wx.ctrl == nil
	wx.mu.Unlock()
	if needBoot {
		a.emit(wenxueEventChannel, map[string]interface{}{"kind": "phase", "text": "正在启动图文引擎"})
	}
	if err := a.WenxueEngineInit(); err != nil {
		a.emit(wenxueEventChannel, map[string]interface{}{"kind": "error", "text": err.Error()})
		return
	}
	if err := a.wenxuePreflight(); err != nil {
		a.emit(wenxueEventChannel, map[string]interface{}{"kind": "error", "text": err.Error()})
		return
	}
	if c := wenxueCtrl(); c != nil {
		c.Send(input)
	}
}

// WenxueSteer 运行中插话（不打断工具执行；未运行走 Send 排队兜底）。
func (a *App) WenxueSteer(input string) {
	c, err := a.wenxueRequireCtrl()
	if err != nil {
		a.emit(wenxueEventChannel, map[string]interface{}{"kind": "error", "text": err.Error()})
		return
	}
	c.Steer(input)
}

// WenxueCancel 取消当前回合。
func (a *App) WenxueCancel() {
	if c := wenxueCtrl(); c != nil {
		c.Cancel()
	}
}

// WenxueRunning 报告图文引擎是否正在运行。
func (a *App) WenxueRunning() bool {
	wx.mu.Lock()
	defer wx.mu.Unlock()
	return wx.ctrl != nil && wx.ctrl.Running()
}

// WenxueApprove 以决策串答复待批工具调用。
func (a *App) WenxueApprove(id string, decision string) {
	if c := wenxueCtrl(); c != nil {
		c.Approve(id, decision)
	}
}

// WenxueAnswer 回答结构化提问。
func (a *App) WenxueAnswer(id string, answers []event.AskAnswer) {
	if c := wenxueCtrl(); c != nil {
		c.AnswerQuestion(id, answers)
	}
}

// ── 会话生命周期 ────────────────────────────────────────────────

// WenxueHistory 返回图文当前会话的对话历史（体与办公 GaeaHistory 同构）。
func (a *App) WenxueHistory() []HistoryMessage {
	out := []HistoryMessage{}
	c := wenxueCtrl()
	if c == nil {
		return out
	}
	msgs := c.History()
	out = buildHistoryMessages(msgs)
	entries, err := session.ReadEntriesFor(c.SessionPath())
	if err != nil {
		return out
	}
	anchors := session.ProjectSubagentAnchors(entries)
	if len(anchors) == 0 {
		return out
	}
	logOffset := len(msgs) - len(session.ProjectMessages(entries))
	return mergeSubagentAnchors(msgs, anchors, logOffset)
}

// WenxueListSessions 当前小说工作空间会话列表（工作空间面 fork 自
// wenxue_workspace.go，此处改锚 wx 控制器当前会话）。
func (a *App) WenxueListSessions() []SessionMeta {
	current := wenxueWorkspace()
	if current == wenxueWorkspaceRoot() {
		return []SessionMeta{}
	}
	curSession := ""
	if c := wenxueCtrl(); c != nil {
		curSession = c.SessionPath()
	}
	return a.listSessionsForDir(gaeaConfig.WorkspaceSessionDir(current, ""), curSession, true)
}

// WenxueListProjectSessions 以小说目录为项目分组（控制器锚换 wx）。
func (a *App) WenxueListProjectSessions() []ProjectGroup {
	current := wenxueWorkspace()
	curSession := ""
	if c := wenxueCtrl(); c != nil {
		curSession = c.SessionPath()
	}
	groups := make([]ProjectGroup, 0)
	for _, ws := range a.WenxueListWorkspaces() {
		sessions := a.listSessionsForDir(
			gaeaConfig.WorkspaceSessionDir(ws.Path, ""),
			curSession,
			ws.Current,
		)
		archived := a.archivedSessionsForDir(gaeaConfig.WorkspaceSessionDir(ws.Path, ""))
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
		if mod == 0 {
			if info, err := os.Stat(ws.Path); err == nil {
				mod = info.ModTime().UnixMilli()
			}
		}
		groups = append(groups, ProjectGroup{
			Path:     ws.Path,
			Name:     ws.Name,
			Current:  wenxueSamePath(ws.Path, current),
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

// WenxueResumeSession 快照当前会话并加载目标会话继续（体与办公同构）。
func (a *App) WenxueResumeSession(path string) ([]HistoryMessage, error) {
	c, err := a.wenxueRequireCtrl()
	if err != nil {
		return nil, err
	}
	if c.Running() {
		return nil, errors.New("回合正在运行，请先停止当前回合再切换会话")
	}
	_ = c.Snapshot()
	loaded, err := c.ResumeFromDisk(path)
	if err != nil {
		return nil, err
	}
	wx.wire.reset()
	injectInterruption(loaded, path)
	c.SeedContextUsage()
	return a.WenxueHistory(), nil
}

// WenxueDeleteSession 删除已保存会话（拒绝删除当前会话）。
func (a *App) WenxueDeleteSession(path string) error {
	if c := wenxueCtrl(); c != nil && c.SessionPath() == path {
		return errActiveSession
	}
	dir := sessionDirForPath(path)
	if dir == "" {
		return fmt.Errorf("非法会话路径: %s", path)
	}
	return deleteSessionFile(dir, path)
}

// WenxueArchiveSession 归档会话（拒绝归档当前会话；归档后清置顶）。
func (a *App) WenxueArchiveSession(path string) error {
	if c := wenxueCtrl(); c != nil && c.SessionPath() == path {
		return errActiveSession
	}
	dir := sessionDirForPath(path)
	if dir == "" || !isSessionSpaceBase(dir) {
		return fmt.Errorf("非法会话路径: %s", path)
	}
	if err := agent.ArchiveSession(path); err != nil {
		return err
	}
	pinned := loadPinned(dir)
	base := filepath.Base(path)
	if pinned[base] {
		delete(pinned, base)
		_ = savePinned(dir, pinned)
	}
	return nil
}

// WenxueUnarchiveSession 把归档会话移回会话目录，返回恢复后的活动路径。
func (a *App) WenxueUnarchiveSession(path string) (string, error) {
	dir := sessionDirForPath(path)
	if dir == "" || filepath.Base(dir) != "archive" {
		return "", fmt.Errorf("非法归档路径: %s", path)
	}
	if err := agent.UnarchiveSession(path); err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(dir), filepath.Base(path)), nil
}

// WenxuePinSession 置顶/取消置顶一个活动会话。
func (a *App) WenxuePinSession(path string, pinned bool) error {
	dir := sessionDirForPath(path)
	if dir == "" || !isSessionSpaceBase(dir) {
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

// WenxueRenameSession 设置会话自定义名称（空清除）。
func (a *App) WenxueRenameSession(path, title string) error {
	dir := sessionDirForPath(path)
	if dir == "" {
		return fmt.Errorf("非法会话路径: %s", path)
	}
	return setSessionTitle(dir, path, title)
}

// WenxueRewind 回退到指定回合（scope 语义与办公一致）。
func (a *App) WenxueRewind(turn int, scope string) error {
	c := wenxueCtrl()
	if c == nil {
		return errors.New("图文引擎未初始化")
	}
	switch scope {
	case "fork":
		if _, err := c.Fork(turn); err != nil {
			return err
		}
		wx.wire.reset()
		return nil
	case "code":
		return errors.New("文件级回退需要文件版本快照，暂未支持；可用「仅对话」回退会话")
	case "summ-from", "summ-upto":
		return errors.New("摘要回退暂未支持")
	default:
		return c.Rewind(turn)
	}
}

// WenxueFork 从指定回合分叉出新会话并接管为当前会话。
func (a *App) WenxueFork(turn int) error {
	c := wenxueCtrl()
	if c == nil {
		return errors.New("图文引擎未初始化")
	}
	if _, err := c.Fork(turn); err != nil {
		return err
	}
	wx.wire.reset()
	return nil
}

// ── 面板读数（体与办公对应件同构，控制器锚换图文）────────────────

// WenxueCommands 列出图文会话可用斜杠命令。
func (a *App) WenxueCommands() []CommandInfo {
	out := []CommandInfo{
		{Name: "new", Description: i18n.M.CmdNew, Kind: "builtin"},
		{Name: "compact", Description: i18n.M.CmdCompact, Kind: "builtin"},
		{Name: "model", Description: i18n.M.CmdModel, Kind: "builtin"},
		{Name: "memory", Description: i18n.M.CmdMemory, Kind: "builtin"},
		{Name: "context", Description: i18n.M.CmdContext, Kind: "builtin"},
		{Name: "mcp", Description: i18n.M.CmdMcp, Kind: "builtin"},
		{Name: "hooks", Description: i18n.M.CmdHooks, Kind: "builtin"},
		{Name: "skill", Description: i18n.M.CmdSkill, Kind: "builtin"},
	}
	c := wenxueCtrl()
	if c == nil {
		return out
	}
	for _, s := range c.Skills() {
		out = append(out, CommandInfo{Name: s.Name, Description: s.Description, Kind: "skill"})
	}
	for _, cmd := range c.Commands() {
		out = append(out, CommandInfo{Name: cmd.Name, Description: cmd.Description, Hint: cmd.ArgHint, Kind: "custom"})
	}
	if h := c.Host(); h != nil {
		for _, p := range h.Prompts() {
			out = append(out, CommandInfo{Name: p.Name, Description: p.Description, Kind: "mcp"})
		}
	}
	return out
}

// WenxueSlashArgs 补全管理类斜杠命令的参数。
func (a *App) WenxueSlashArgs(input string) SlashArgsResult {
	c := wenxueCtrl()
	if c == nil {
		return SlashArgsResult{Items: []SlashArgItem{}}
	}
	data := control.ArgData{Skills: c.Skills()}
	for _, m := range a.GaeaModels() {
		data.ModelRefs = append(data.ModelRefs, m.Ref)
	}
	if h := c.Host(); h != nil {
		data.ServerNames = h.ServerNames()
	}
	items, from := control.SlashArgItems(input, data)
	out := SlashArgsResult{Items: []SlashArgItem{}, From: from}
	for _, it := range items {
		out.Items = append(out.Items, SlashArgItem{Label: it.Label, Insert: it.Insert, Hint: it.Hint, Descend: it.Descend})
	}
	return out
}

// WenxueContext 返回上下文窗口仪表读数。
func (a *App) WenxueContext() ContextInfo {
	c := wenxueCtrl()
	if c == nil {
		return ContextInfo{}
	}
	used, window := c.ContextSnapshot()
	info := ContextInfo{Used: used, Window: window}
	if _, ratio, _ := c.CompactionTiers(); ratio > 0 && window > 0 {
		info.CompactRatio = ratio
		left := int(float64(window)*ratio) - used
		if left < 0 {
			left = 0
		}
		info.CompactLeft = left
	}
	return info
}

// WenxueBalance 查询当前引擎钱包余额（网络调用，失败返回不可用读数）。
func (a *App) WenxueBalance() BalanceInfo {
	c := wenxueCtrl()
	if c == nil {
		return BalanceInfo{}
	}
	b, err := c.Balance(context.Background())
	if err != nil {
		return BalanceInfo{Err: err.Error()}
	}
	if b == nil {
		return BalanceInfo{}
	}
	display := ""
	for _, inf := range b.Infos {
		display += inf.Currency + " " + inf.TotalBalance + " "
	}
	return BalanceInfo{Available: b.Available, Display: strings.TrimSpace(display)}
}

// WenxueJobs 返回图文引擎后台任务。
func (a *App) WenxueJobs() []JobView {
	out := []JobView{}
	c := wenxueCtrl()
	if c == nil {
		return out
	}
	for _, v := range c.Jobs() {
		out = append(out, JobView{ID: v.ID, Kind: v.Kind, Label: v.Label, Status: v.Status, StartedAt: v.StartedAt})
	}
	return out
}

// WenxueCapabilities 返回图文引擎的 MCP 服务器与技能列表（刀2 起技能面为
// 图文子集；本绑定如实反映引擎装配结果）。
func (a *App) WenxueCapabilities() CapabilitiesView {
	view := CapabilitiesView{Servers: []ServerView{}, Skills: []SkillView{}, Tools: []string{}}
	c := wenxueCtrl()
	if c == nil {
		return view
	}
	for _, t := range c.Tools() {
		view.Tools = append(view.Tools, t.Name())
	}
	if h := c.Host(); h != nil {
		for _, s := range h.Servers() {
			sv := ServerView{Name: s.Name, Transport: s.Transport, Status: "connected", Tools: s.Tools, Prompts: s.Prompts, Resources: s.Resources, ToolList: []ToolView{}}
			for _, t := range s.ToolList {
				sv.ToolList = append(sv.ToolList, ToolView{Name: t.Name, Description: t.Description})
			}
			view.Servers = append(view.Servers, sv)
		}
		for _, f := range h.Failures() {
			view.Servers = append(view.Servers, ServerView{Name: f.Name, Transport: f.Transport, Status: "failed", Error: f.Error})
		}
	}
	for _, s := range c.Skills() {
		view.Skills = append(view.Skills, SkillView{Name: s.Name, Description: s.Description, Scope: string(s.Scope), RunAs: string(s.RunAs)})
	}
	return view
}

// ── 记忆面板（ctrl 锚定记忆库，随图文 Cwd=小说目录天然隔离）────────

// WenxueMemory 返回图文会话记忆面板数据（体与办公 GaeaMemory 同构）。
// facts 挂图文脑（2026-11-16 面板恢复批修正：本绑定曾读 hubOfficeStore=混源
// 遗留，与 docs/scopes 的 ctrl 锚（图文脑）不同源；现全部锚 wenxue-brain）。
func (a *App) WenxueMemory() MemoryView {
	view := MemoryView{Docs: []MemoryDoc{}, Facts: []MemoryFact{}, Scopes: []MemoryScope{}}
	view.Available = wenxueBrainDir() != ""
	for _, f := range a.hubWenxueBrainStore().List() {
		view.Facts = append(view.Facts, MemoryFact{
			Name: f.Name, Title: f.Title, Description: f.Description,
			Type: string(f.Type), Body: f.Body,
			LastUsedAt:    fmtTimeOrEmpty(f.LastUsedAt),
			SourceSession: f.SourceSession,
			SourceMessage: f.SourceMessage,
			Pinned:        f.Pinned,
		})
	}
	if c := wenxueCtrl(); c != nil {
		if set := c.Memory(); set != nil {
			view.StoreDir = set.Store.Dir
			for _, d := range set.Docs {
				view.Docs = append(view.Docs, MemoryDoc{Path: d.Path, Scope: string(d.Scope), Body: d.Body})
			}
			for _, sc := range writableScopes {
				if p := set.DocPath(sc); p != "" {
					view.Scopes = append(view.Scopes, MemoryScope{Scope: string(sc), Path: p})
				}
			}
		}
	}
	view.Enabled = memoryEnabled()
	view.DreamMode = gaeaDreamMode()
	return view
}

// WenxueRemember/WenxueForget/WenxueUpdateFact/WenxueChangeFactType/WenxueSaveDoc
// 记忆面板写侧五件（体与办公对应件同构）。
func (a *App) WenxueRemember(scope, note string) (string, error) {
	if c := wenxueCtrl(); c != nil {
		return c.QuickAdd(parseScope(scope), note)
	}
	return "", nil
}

func (a *App) WenxueForget(name string) error {
	if c := wenxueCtrl(); c != nil {
		return c.ForgetMemory(name)
	}
	return nil
}

func (a *App) WenxueUpdateFact(name, body string) (string, error) {
	if c := wenxueCtrl(); c != nil {
		return c.UpdateFact(name, body)
	}
	return "", nil
}

func (a *App) WenxueChangeFactType(name, typ string) (string, error) {
	if c := wenxueCtrl(); c != nil {
		if err := c.ChangeFactType(name, typ); err != nil {
			return "", err
		}
	}
	return name, nil
}

func (a *App) WenxueSaveDoc(path, body string) (string, error) {
	if c := wenxueCtrl(); c != nil {
		return c.SaveDoc(path, body)
	}
	return "", nil
}

// WenxueMemorySuggestions 图文记忆建议卡（诚实空态）：图文引擎不跑自动做梦
// （办公域 TurnDone 钩子不进图文引擎，见 wenxue_engine.go），建议队列恒空；
// 记忆管理走记忆面板写侧五件与记忆中枢。遗留办公建议的接受/忽略走共享
// AcceptMemorySuggestion/DismissMemorySuggestion（前端未重指）。
func (a *App) WenxueMemorySuggestions() MemorySuggestionsView {
	return MemorySuggestionsView{
		Memories:    []MemorySuggestionView{},
		Skills:      []SkillSuggestionView{},
		GeneratedAt: time.Now().Format(time.RFC3339),
		Source:      "图文引擎不运行自动做梦（无待确认建议）",
	}
}

// ── 权限档位 / 事件补拉 / 子代理追问 ────────────────────────────

// WenxueSetPermLevel 设置图文会话权限级别（ask/auto/yolo，ctrl 级状态）。
func (a *App) WenxueSetPermLevel(level string) error {
	if c := wenxueCtrl(); c != nil {
		c.SetPermLevel(level)
	}
	return nil
}

// WenxuePermLevel 返回图文会话权限级别。
func (a *App) WenxuePermLevel() string {
	if c := wenxueCtrl(); c != nil {
		return c.PermLevel()
	}
	return "ask"
}

// WenxueResyncEvents 按前端最后收到的 wire seq 补拉图文当前会话对话视图
// （体与办公 GaeaResyncEvents 同构；wire 换 wx.wire）。
func (a *App) WenxueResyncEvents(afterSeq int) GaeaResyncResult {
	res := GaeaResyncResult{Seq: wx.wire.last(), Items: []GaeaResyncItem{}}
	c := wenxueCtrl()
	if c == nil {
		return res
	}
	entries, err := session.ReadEntriesFor(c.SessionPath())
	if err != nil {
		return res
	}
	res.Items = foldResyncItems(entriesAfter(entries, int64(afterSeq)))
	return res
}

// WenxueSubagentFollowUp 对图文会话的子代理运行发起追问。体与办公
// GaeaSubagentFollowUp 同构：runner/运行态读 wx，单飞 claims 复用全局表
// （ref 全局唯一），空间自描述与办公同配置口径。
func (a *App) WenxueSubagentFollowUp(sessionPath, ref, prompt string) (string, error) {
	_ = sessionPath // transcript 目录随 controller 解析；ref 全局唯一无需路由
	wx.mu.Lock()
	runner := wx.followUp
	running := wx.ctrl != nil && wx.ctrl.Running()
	wx.mu.Unlock()
	if runner == nil {
		return "", fmt.Errorf("子代理追问未接线（图文引擎尚未构建完成）")
	}
	if running {
		return "", fmt.Errorf("主对话回合正在运行，请等回合结束再追问")
	}
	claimKey := ref
	if _, loaded := followUpClaims.LoadOrStore(claimKey, true); loaded {
		return "", fmt.Errorf("该子代理已有追问正在运行")
	}
	if dir := sessionDirForPath(sessionPath); dir != "" && agent.ValidRunRef(ref) {
		store := agent.NewSubagentStore(filepath.Join(dir, "subagents"))
		_ = store.RecordFollowUpError(ref, "")
	}
	ctx := agent.WithSpace(context.Background(), gaeaSessionSpace())
	go func() {
		defer func() {
			r := recover()
			followUpClaims.Delete(claimKey)
			if r != nil {
				a.gaeaBackgroundPanicNotice("子代理追问", r, map[string]interface{}{"ref": ref})
			}
		}()
		if err := runner(ctx, ref, prompt); err != nil {
			slog.Warn("子代理追问失败", "ref", ref, "error", err)
		}
	}()
	return "follow-up started", nil
}

// ── 上下文/轨迹看板族：显式路径委托（办公实现本就接受 sessionPath）──
// 缺省解析换图文当前会话路径；显式路径原样透传（sessionDirForPath 守门不变）。

// wenxueSessionPathArg 返回图文看板族的路径参数：显式传入优先，缺省=图文
// 控制器当前会话（无引擎=原样透传空参，按办公同款空快照语义回落）。
func (a *App) wenxueSessionPathArg(args []string) []string {
	for _, p := range args {
		if strings.TrimSpace(p) != "" {
			return args
		}
	}
	if c := wenxueCtrl(); c != nil {
		if sp := c.SessionPath(); sp != "" {
			return []string{sp}
		}
	}
	return args
}

// WenxueContextView/WenxueContextNodeDetail/WenxueContextSurface/
// WenxueContextCommands/WenxueContextDna/WenxueContextInsights/WenxueTrajectory/
// WenxueAgentNetwork/WenxueFamilyEvents/WenxueSubagentContextView 看板族：
// 一行委托办公实现（显式路径语义），仅缺省解析换图文当前会话。

func (a *App) WenxueContextView(sessionPath ...string) (contextview.ContextTimeline, error) {
	return a.GaeaContextView(a.wenxueSessionPathArg(sessionPath)...)
}

func (a *App) WenxueContextNodeDetail(seq int64, sessionPath ...string) (contextview.NodeDetail, error) {
	return a.GaeaContextNodeDetail(seq, a.wenxueSessionPathArg(sessionPath)...)
}

func (a *App) WenxueContextSurface(seq int64, sessionPath ...string) (contextview.SurfaceSnapshot, error) {
	return a.GaeaContextSurface(seq, a.wenxueSessionPathArg(sessionPath)...)
}

func (a *App) WenxueContextCommands(beforeSeq int64, limit int, sessionPath ...string) (contextview.CommandPage, error) {
	return a.GaeaContextCommands(beforeSeq, limit, a.wenxueSessionPathArg(sessionPath)...)
}

func (a *App) WenxueContextDna(seqs []int64, sessionPath ...string) ([]contextview.DnaSample, error) {
	return a.GaeaContextDna(seqs, a.wenxueSessionPathArg(sessionPath)...)
}

func (a *App) WenxueContextInsights(rangeKey string, includeArchived bool) (contextview.ContextInsights, error) {
	// Insight 面聚合全会话区间（无会话路径参数），与引擎锚无关，直接委托。
	return a.GaeaContextInsights(rangeKey, includeArchived)
}

func (a *App) WenxueTrajectory(sessionPath ...string) (trajectory.Trajectory, error) {
	return a.GaeaTrajectory(a.wenxueSessionPathArg(sessionPath)...)
}

func (a *App) WenxueAgentNetwork(sessionPath ...string) (trajectory.AgentNetwork, error) {
	return a.GaeaAgentNetwork(a.wenxueSessionPathArg(sessionPath)...)
}

func (a *App) WenxueFamilyEvents(sessionPath ...string) (trajectory.FamilyEventStream, error) {
	return a.GaeaFamilyEvents(a.wenxueSessionPathArg(sessionPath)...)
}

func (a *App) WenxueSubagentContextView(sessionPath, ref string) (contextview.ContextTimeline, error) {
	if sessionPath == "" {
		if c := wenxueCtrl(); c != nil {
			sessionPath = c.SessionPath()
		}
	}
	return a.GaeaSubagentContextView(sessionPath, ref)
}
