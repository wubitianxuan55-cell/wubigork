package app

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/i18n"
	"github.com/gaea/gaea/internal/gaea/knowledge"
	"github.com/gaea/gaea/internal/gaea/memory"
)

// ── 元信息 / 状态（对齐 gaeaW desktop/app_meta.go）──────────────────

// ContextInfo 是提示词 vs 上下文窗口的仪表读数。
type ContextInfo struct {
	Used   int `json:"used"`
	Window int `json:"window"`
}

// BalanceInfo 是状态栏的钱包余额读数。
type BalanceInfo struct {
	Available bool   `json:"available"`
	Display   string `json:"display"`
	Err       string `json:"err,omitempty"`
}

// JobView 是一个后台任务（状态栏指示器）。
type JobView struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Label     string `json:"label"`
	Status    string `json:"status"`
	StartedAt int64  `json:"startedAt"`
}

// Meta 描述会话（前端头部与状态行）。
type Meta struct {
	Label         string `json:"label"`
	SubagentLabel string `json:"subagentLabel,omitempty"`
	Ready         bool   `json:"ready"`
	StartupErr    string `json:"startupErr,omitempty"`
	EventChannel  string `json:"eventChannel"`
	Cwd           string `json:"cwd"`
	Bypass        bool   `json:"bypass"`
	PermLevel     string `json:"permLevel"`
}

// CommandInfo 描述一个斜杠命令（composer 的 "/" 菜单）。
type CommandInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Hint        string `json:"hint,omitempty"`
	Kind        string `json:"kind"`
}

// SlashArgItem 是斜杠命令参数补全项。
type SlashArgItem struct {
	Label   string `json:"label"`
	Insert  string `json:"insert"`
	Hint    string `json:"hint"`
	Descend bool   `json:"descend"`
}

// SlashArgsResult 携带补全项与当前 token 起始偏移。
type SlashArgsResult struct {
	Items []SlashArgItem `json:"items"`
	From  int            `json:"from"`
}

// ModelInfo 是模型切换器的一个可选项。
type ModelInfo struct {
	Ref      string `json:"ref"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Current  bool   `json:"current"`
}

// GaeaMeta 返回会话元信息。控制器未初始化时懒初始化（首次调用即就绪），
// 避免办公板块一直停留在"正在连接智能体"加载态。
func (a *App) GaeaMeta() Meta {
	c := gaeaCtrl()
	cwd := gaeaCwd()
	if c == nil {
		if err := a.GaeaInit(); err != nil {
			return Meta{
				Label:        "Hephaestus",
				Ready:        false,
				StartupErr:   err.Error(),
				EventChannel: "gaea-event",
				Cwd:          cwd,
			}
		}
		c = gaeaCtrl()
	}
	perm := "ask"
	if c != nil {
		perm = c.PermLevel()
	}
	return Meta{
		Label:        "Hephaestus",
		Ready:        c != nil,
		EventChannel: "gaea-event",
		Cwd:          cwd,
		Bypass:       c != nil && c.PermLevel() != "ask",
		PermLevel:    perm,
	}
}

// GaeaModels 返回模型中心全部引擎/模型（切换器选项）。
func (a *App) GaeaModels() []ModelInfo {
	out := []ModelInfo{}
	if a.engineMgr == nil {
		return out
	}
	active := a.client.ActiveEngineID()
	for _, eng := range a.engineMgr.GetEngines() {
		if !eng.Enabled {
			continue
		}
		model := eng.DefaultModel
		if model == "" {
			model = "(默认)"
		}
		ref := eng.ID + "/" + model
		out = append(out, ModelInfo{Ref: ref, Provider: eng.ID, Model: model, Current: eng.ID == active})
	}
	return out
}

// GaeaSetModel 切换模型：接受 "engine/model" 或 "engine"，按引擎切换。
func (a *App) GaeaSetModel(name string) error {
	engine := name
	if i := strings.Index(name, "/"); i > 0 {
		engine = name[:i]
	}
	return a.SetActiveEngine(engine)
}

// GaeaCommands 列出可用斜杠命令（内置 + 技能 + 自定义 + MCP 提示）。
func (a *App) GaeaCommands() []CommandInfo {
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
	c := gaeaCtrl()
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

// GaeaSlashArgs 补全管理类斜杠命令的参数。
func (a *App) GaeaSlashArgs(input string) SlashArgsResult {
	c := gaeaCtrl()
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

// GaeaContext 返回上下文窗口仪表读数。
func (a *App) GaeaContext() ContextInfo {
	c := gaeaCtrl()
	if c == nil {
		return ContextInfo{}
	}
	used, window := c.ContextSnapshot()
	return ContextInfo{Used: used, Window: window}
}

// GaeaTCCAReport 返回 TCCA 缓存指标 JSON 字符串。
func (a *App) GaeaTCCAReport() string {
	c := gaeaCtrl()
	if c == nil {
		return "{}"
	}
	b, err := json.Marshal(c.TCCAReport())
	if err != nil {
		return "{}"
	}
	return string(b)
}

// GaeaBalance 查询当前引擎钱包余额（网络调用，失败返回不可用读数）。
func (a *App) GaeaBalance() BalanceInfo {
	c := gaeaCtrl()
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

// GaeaJobs 返回后台运行任务。
func (a *App) GaeaJobs() []JobView {
	out := []JobView{}
	c := gaeaCtrl()
	if c == nil {
		return out
	}
	for _, v := range c.Jobs() {
		out = append(out, JobView{ID: v.ID, Kind: v.Kind, Label: v.Label, Status: v.Status, StartedAt: v.StartedAt})
	}
	return out
}

// ── 记忆 / 知识 ──────────────────────────────────────────────────

// MemoryDoc 是面板的一个文档记忆文件。
type MemoryDoc struct {
	Path  string `json:"path"`
	Scope string `json:"scope"`
	Body  string `json:"body"`
}

// MemoryFact 是一个已保存的自动记忆。
type MemoryFact struct {
	Name          string `json:"name"`
	Title         string `json:"title,omitempty"`
	Description   string `json:"description"`
	Type          string `json:"type"`
	Body          string `json:"body"`
	LastUsedAt    string `json:"lastUsedAt,omitempty"`    // 最近使用（RFC3339，溯源/高频展示）
	SourceSession string `json:"sourceSession,omitempty"` // 沉淀来源会话
	SourceMessage string `json:"sourceMessage,omitempty"` // 沉淀来源消息/轮次
}

// MemoryScope 是一个可写快捷添加目标。
type MemoryScope struct {
	Scope string `json:"scope"`
	Path  string `json:"path"`
}

// MemoryView 是记忆面板的完整负载。
type MemoryView struct {
	Docs      []MemoryDoc   `json:"docs"`
	Facts     []MemoryFact  `json:"facts"`
	Scopes    []MemoryScope `json:"scopes"`
	StoreDir  string        `json:"storeDir"`
	Available bool          `json:"available"`
	Enabled   bool          `json:"enabled"` // 记忆开关（当前生效值）
}

var writableScopes = []memory.Scope{memory.ScopeUser, memory.ScopeProject, memory.ScopeLocal}

// GaeaMemory 返回记忆面板数据。
func (a *App) GaeaMemory() MemoryView {
	view := MemoryView{Docs: []MemoryDoc{}, Facts: []MemoryFact{}, Scopes: []MemoryScope{}}
	c := gaeaCtrl()
	if c == nil {
		return view
	}
	set := c.Memory()
	if set == nil {
		return view
	}
	view.StoreDir = set.Store.Dir
	view.Available = true
	for _, d := range set.Docs {
		view.Docs = append(view.Docs, MemoryDoc{Path: d.Path, Scope: string(d.Scope), Body: d.Body})
	}
	for _, f := range set.Store.List() {
		view.Facts = append(view.Facts, MemoryFact{
			Name: f.Name, Title: f.Title, Description: f.Description,
			Type: string(f.Type), Body: f.Body,
			LastUsedAt:    fmtTimeOrEmpty(f.LastUsedAt),
			SourceSession: f.SourceSession,
			SourceMessage: f.SourceMessage,
		})
	}
	for _, sc := range writableScopes {
		if p := set.DocPath(sc); p != "" {
			view.Scopes = append(view.Scopes, MemoryScope{Scope: string(sc), Path: p})
		}
	}
	view.Enabled = memoryEnabled()
	return view
}

// memoryEnabled 返回当前生效的记忆开关值（配置读取失败时默认开启）。
func memoryEnabled() bool {
	ga.mu.Lock()
	cfg := ga.cfg
	ga.mu.Unlock()
	if cfg == nil {
		if c, err := gaeaLoadConfig(); err == nil {
			cfg = c
		}
	}
	if cfg == nil {
		return true
	}
	return cfg.Memory.Enabled
}

// fmtTimeOrEmpty 把 time.Time 格式化为 RFC3339（零值空串）。
func fmtTimeOrEmpty(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// GaeaRemember 快捷添加一条记忆。
func (a *App) GaeaRemember(scope, note string) (string, error) {
	if c := gaeaCtrl(); c != nil {
		return c.QuickAdd(parseScope(scope), note)
	}
	return "", nil
}

// GaeaForget 按名称删除自动记忆。
func (a *App) GaeaForget(name string) error {
	if c := gaeaCtrl(); c != nil {
		return c.ForgetMemory(name)
	}
	return nil
}

// GaeaUpdateFact 覆盖一条事实的内容。
func (a *App) GaeaUpdateFact(name, body string) (string, error) {
	if c := gaeaCtrl(); c != nil {
		return c.UpdateFact(name, body)
	}
	return "", nil
}

// GaeaChangeFactType 改变事实类型。
func (a *App) GaeaChangeFactType(name, typ string) (string, error) {
	if c := gaeaCtrl(); c != nil {
		if err := c.ChangeFactType(name, typ); err != nil {
			return "", err
		}
	}
	return name, nil
}

// GaeaSaveDoc 覆盖记忆文档。
func (a *App) GaeaSaveDoc(path, body string) (string, error) {
	if c := gaeaCtrl(); c != nil {
		return c.SaveDoc(path, body)
	}
	return "", nil
}

// GaeaSetMemoryEnabled 设置办公记忆开关（记忆可控性）：关闭后系统提示词与
// 逐轮上下文不再注入画像/规则/事实（磁盘记忆保留，面板仍可管理），持久化并
// 重建办公引擎立即生效。
func (a *App) GaeaSetMemoryEnabled(enabled bool) error {
	return a.gaeaApplyCfg(func(cfg *gaeaConfig.Config) {
		cfg.Memory.Enabled = enabled
	})
}

// parseScope 映射前端作用域 id 到 memory.Scope。
func parseScope(s string) memory.Scope {
	switch memory.Scope(s) {
	case memory.ScopeUser:
		return memory.ScopeUser
	case memory.ScopeLocal:
		return memory.ScopeLocal
	default:
		return memory.ScopeProject
	}
}

// KnowledgeSummary 是知识条目的轻量视图。
type KnowledgeSummary struct {
	Name      string    `json:"name"`
	Title     string    `json:"title"`
	Category  string    `json:"category"`
	Tags      []string  `json:"tags"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// KnowledgeEntry 是包含正文的完整知识条目。
type KnowledgeEntry struct {
	Name       string    `json:"name"`
	Title      string    `json:"title"`
	Category   string    `json:"category"`
	Phase      string    `json:"phase"`
	Discipline string    `json:"discipline"`
	Tags       []string  `json:"tags"`
	Status     string    `json:"status"`
	Version    int       `json:"version"`
	Author     string    `json:"author"`
	Reviewer   string    `json:"reviewer"`
	Source     string    `json:"source"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// GaeaKnowledgeList 返回知识条目摘要列表。
func (a *App) GaeaKnowledgeList() []KnowledgeSummary {
	store, err := knowledge.Global().Store()
	if err != nil {
		return []KnowledgeSummary{}
	}
	list := store.List()
	out := make([]KnowledgeSummary, 0, len(list))
	for _, s := range list {
		out = append(out, KnowledgeSummary{Name: s.Name, Title: s.Title, Category: s.Category, Tags: s.Tags, Status: s.Status, UpdatedAt: s.UpdatedAt})
	}
	return out
}

// GaeaKnowledgeSearch 全文检索知识库（标题/分类/标签/正文），返回匹配条目摘要。
// 空 query 等价于 List；category/phase/status 为 "all" 或空时不过滤。
func (a *App) GaeaKnowledgeSearch(query, category, phase, status string) []KnowledgeSummary {
	store, err := knowledge.Global().Store()
	if err != nil {
		return []KnowledgeSummary{}
	}
	filter := knowledge.Filter{Category: category, Phase: phase, Status: status}
	if filter.Category == "all" {
		filter.Category = ""
	}
	if filter.Phase == "all" {
		filter.Phase = ""
	}
	if filter.Status == "all" {
		filter.Status = ""
	}
	results := knowledge.Search(store, query, filter)
	// 语义召回：关键词召回不足（<3）时用本地 bge-m3 补召回。
	if len(results) < 3 && strings.TrimSpace(query) != "" {
		if sem := a.semanticKnowledgeRecall(query, results, 10); len(sem) > 0 {
			results = sem
		}
	}
	// 本地语义精排（bge-reranker-v2-m3），失败自动回退。
	if reranked := a.rerankKnowledgeResults(query, results, 20); len(reranked) > 0 {
		results = reranked
	} else if len(results) > 20 {
		results = results[:20]
	}
	out := make([]KnowledgeSummary, 0, len(results))
	for _, e := range results {
		out = append(out, KnowledgeSummary{Name: e.Name, Title: e.Title, Category: e.Category, Tags: e.Tags, Status: e.Status, UpdatedAt: e.UpdatedAt})
	}
	return out
}

// GaeaKnowledgeGet 返回单条知识条目（未找到返回 nil）。
func (a *App) GaeaKnowledgeGet(name string) *KnowledgeEntry {
	store, err := knowledge.Global().Store()
	if err != nil {
		return nil
	}
	e, err := store.Get(name)
	if err != nil || e == nil {
		return nil
	}
	return &KnowledgeEntry{Name: e.Name, Title: e.Title, Category: e.Category, Phase: e.Phase, Discipline: e.Discipline, Tags: e.Tags, Status: e.Status, Version: e.Version, Author: e.Author, Reviewer: e.Reviewer, Source: e.Source, Body: e.Body, CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt}
}

// GaeaKnowledgeSave 保存知识条目。
func (a *App) GaeaKnowledgeSave(e KnowledgeEntry) error {
	store, err := knowledge.Global().Store()
	if err != nil {
		return err
	}
	return saveKnowledgeVersioned(store, knowledge.Entry{Name: e.Name, Title: e.Title, Category: e.Category, Phase: e.Phase, Discipline: e.Discipline, Tags: e.Tags, Status: e.Status, Version: e.Version, Author: e.Author, Reviewer: e.Reviewer, Source: e.Source, Body: e.Body, CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt})
}

// GaeaKnowledgeDelete 删除知识条目。
func (a *App) GaeaKnowledgeDelete(name string) error {
	store, err := knowledge.Global().Store()
	if err != nil {
		return err
	}
	return store.Delete(name)
}

// ── 能力 / 插件 / 技能 ────────────────────────────────────────────

// CapabilitiesView 是 MCP 服务器与技能抽屉的负载。
type CapabilitiesView struct {
	Servers []ServerView `json:"servers"`
	Skills  []SkillView  `json:"skills"`
}

// ServerView 是一个 MCP 服务器条目。
type ServerView struct {
	Name      string     `json:"name"`
	Transport string     `json:"transport"`
	Status    string     `json:"status"`
	Tools     int        `json:"tools"`
	Prompts   int        `json:"prompts"`
	Resources int        `json:"resources"`
	Error     string     `json:"error,omitempty"`
	ToolList  []ToolView `json:"toolList,omitempty"`
}

// ToolView 是服务器工具条目。
type ToolView struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// SkillView 是一个技能条目。
type SkillView struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Scope       string `json:"scope"`
	RunAs       string `json:"runAs"`
}

// GaeaCapabilities 返回 MCP 服务器与技能列表。
func (a *App) GaeaCapabilities() CapabilitiesView {
	view := CapabilitiesView{Servers: []ServerView{}, Skills: []SkillView{}}
	c := gaeaCtrl()
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

// GaeaVersion 返回办公板块版本。
func (a *App) GaeaVersion() string { return "1.0.0" }
