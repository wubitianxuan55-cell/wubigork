package app

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/auth"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ── 设置面板视图（对齐 gaeaW desktop/settings_app.go）────────────────

// ProviderView 是设置面板的引擎条目（gaea 侧映射模型中心引擎）。
type ProviderView struct {
	Name          string   `json:"name"`
	Kind          string   `json:"kind"`
	BaseURL       string   `json:"baseUrl"`
	Models        []string `json:"models"`
	Default       string   `json:"default"`
	APIKeyEnv     string   `json:"apiKeyEnv"`
	KeySet        bool     `json:"keySet"`
	BalanceURL    string   `json:"balanceUrl"`
	ContextWindow int      `json:"contextWindow"`
	OAuthKind     string   `json:"oauthKind"`
	OAuthReady    bool     `json:"oauthReady"`
}

// PermissionsView 是权限设置。
type PermissionsView struct {
	Mode  string   `json:"mode"`
	Allow []string `json:"allow"`
	Ask   []string `json:"ask"`
	Deny  []string `json:"deny"`
}

// SandboxView 是沙箱设置。
type SandboxView struct {
	Bash          string   `json:"bash"`
	Network       bool     `json:"network"`
	WorkspaceRoot string   `json:"workspaceRoot"`
	AllowWrite    []string `json:"allowWrite"`
}

// AgentView 是 agent 参数设置。
type AgentView struct {
	Temperature         float64 `json:"temperature"`
	MaxSteps            int     `json:"maxSteps"`
	SystemPrompt        string  `json:"systemPrompt"`
	SubagentTemperature float64 `json:"subagentTemperature"`
	Effort              string  `json:"effort"`
	SubagentEffort      string  `json:"subagentEffort"`
}

// SettingsView 是设置面板完整负载。
type SettingsView struct {
	DefaultModel   string            `json:"defaultModel"`
	SubagentModel  string            `json:"subagentModel"`
	SubagentModels map[string]string `json:"subagentModels"`
	SubagentSkills []string          `json:"subagentSkills"`
	Providers      []ProviderView    `json:"providers"`
	Permissions    PermissionsView   `json:"permissions"`
	Sandbox        SandboxView       `json:"sandbox"`
	Agent          AgentView         `json:"agent"`
	ConfigPath     string            `json:"configPath"`
	ProviderKinds  []string          `json:"providerKinds"`
	Bypass         bool              `json:"bypass"`
}

// GaeaSettings 返回设置面板数据（引擎来自模型中心，agent 参数来自 gaea 配置）。
func (a *App) GaeaSettings() SettingsView {
	ga.mu.Lock()
	cfg := ga.cfg
	ga.mu.Unlock()
	if cfg == nil {
		var err error
		cfg, err = gaeaLoadConfig()
		if err != nil {
			cfg = gaeaConfig.Default()
		}
	}
	view := SettingsView{
		Providers:      []ProviderView{},
		SubagentModels: map[string]string{},
		SubagentSkills: []string{},
		ProviderKinds:  []string{"wubigrok"}, // 内部 provider 注册名（bridge provider）
		DefaultModel:   a.GaeaModel(),
	}
	if a.engineMgr != nil {
		for _, eng := range a.engineMgr.GetEngines() {
			if !eng.Enabled {
				continue
			}
			pv := ProviderView{Name: eng.ID, Kind: string(eng.Type), BaseURL: eng.BaseURL, Default: eng.DefaultModel}
			if eng.DefaultModel != "" {
				pv.Models = append(pv.Models, eng.DefaultModel)
			}
			pv.KeySet = eng.APIKey != ""
			view.Providers = append(view.Providers, pv)
		}
	}
	view.Permissions = PermissionsView{Mode: cfg.Permissions.Mode, Allow: cfg.Permissions.Allow, Ask: cfg.Permissions.Ask, Deny: cfg.Permissions.Deny}
	view.Sandbox = SandboxView{Bash: cfg.Sandbox.Bash, Network: cfg.Sandbox.Network, WorkspaceRoot: cfg.WriteRoots()[0], AllowWrite: cfg.WriteRoots()}
	view.Agent = AgentView{
		Temperature:         cfg.Agent.Temperature,
		MaxSteps:            cfg.Agent.MaxSteps,
		SystemPrompt:        cfg.Agent.SystemPrompt,
		Effort:              cfg.Agent.Effort,
		SubagentEffort:      cfg.Agent.SubagentEffort,
		SubagentTemperature: cfg.Agent.SubagentTemperature,
	}
	view.SubagentModel = cfg.Agent.SubagentModel
	view.SubagentModels = cfg.Agent.SubagentModels
	view.ConfigPath = gaeaConfig.UserConfigPath()
	if c := gaeaCtrl(); c != nil {
		view.Bypass = c.PermLevel() != "ask"
	}
	return view
}

// GaeaSetDefaultModel 设置模型中心引擎的默认模型。
func (a *App) GaeaSetDefaultModel(ref string) error {
	engine, model := ref, ""
	if i := strings.IndexByte(ref, '/'); i > 0 {
		engine, model = ref[:i], ref[i+1:]
	}
	if model == "" || a.engineMgr == nil {
		return fmt.Errorf("无法设置默认模型: 引擎或模型无效（ref=%q）", ref)
	}
	return a.engineMgr.SetDefaultModel(engine, model)
}

// gaeaApplyCfg 修改当前办公引擎配置并持久化，然后重建 controller 使变更生效。
// 权限/沙箱/Agent 参数等引擎级设置统一走此通道：改 ga.cfg → Save → Rebuild。
func (a *App) gaeaApplyCfg(mutate func(cfg *gaeaConfig.Config)) error {
	if err := a.GaeaInit(); err != nil {
		return err
	}
	ga.mu.Lock()
	defer ga.mu.Unlock()
	if ga.cfg == nil {
		return errors.New("gaea: 办公引擎配置未初始化")
	}
	mutate(ga.cfg)
	if err := gaeaConfig.Save(ga.cfg); err != nil {
		return fmt.Errorf("gaea: 保存配置失败: %w", err)
	}
	return a.gaeaRebuildLocked()
}

// GaeaSaveSettings 保存办公引擎设置（模型 / Agent 参数 / 权限 / 沙箱），
// 经 gaeaApplyCfg 通道：改 ga.cfg → Save → 重建 controller 即时生效。
// 空字段/零值跳过，不覆盖已配置项；Network 布尔显式写入。
func (a *App) GaeaSaveSettings(view SettingsView) error {
	return a.gaeaApplyCfg(func(cfg *gaeaConfig.Config) {
		if view.DefaultModel != "" {
			cfg.DefaultModel = view.DefaultModel
		}
		if view.SubagentModel != "" {
			cfg.Agent.SubagentModel = view.SubagentModel
		}
		if view.Agent.SystemPrompt != "" {
			cfg.Agent.SystemPrompt = view.Agent.SystemPrompt
		}
		if view.Agent.MaxSteps > 0 {
			cfg.Agent.MaxSteps = view.Agent.MaxSteps
		}
		if view.Agent.Temperature > 0 {
			cfg.Agent.Temperature = view.Agent.Temperature
		}
		if view.Agent.SubagentTemperature > 0 {
			cfg.Agent.SubagentTemperature = view.Agent.SubagentTemperature
		}
		if view.Agent.Effort != "" {
			cfg.Agent.Effort = view.Agent.Effort
		}
		if view.Agent.SubagentEffort != "" {
			cfg.Agent.SubagentEffort = view.Agent.SubagentEffort
		}
		if view.Permissions.Mode != "" {
			cfg.Permissions.Mode = view.Permissions.Mode
		}
		if view.Permissions.Allow != nil {
			cfg.Permissions.Allow = view.Permissions.Allow
		}
		if view.Permissions.Ask != nil {
			cfg.Permissions.Ask = view.Permissions.Ask
		}
		if view.Permissions.Deny != nil {
			cfg.Permissions.Deny = view.Permissions.Deny
		}
		if view.Sandbox.Bash != "" {
			cfg.Sandbox.Bash = view.Sandbox.Bash
		}
		cfg.Sandbox.Network = view.Sandbox.Network
		if view.Sandbox.WorkspaceRoot != "" {
			cfg.Sandbox.WorkspaceRoot = view.Sandbox.WorkspaceRoot
		}
	})
}

// GaeaSaveProvider 保存模型中心引擎配置（更新已有引擎）。
// 模型中心引擎为固定四大引擎（xai/ollama/herdsman/deepseek），不支持新增。
func (a *App) GaeaSaveProvider(p ProviderView) error {
	if a.engineMgr == nil {
		return errors.New("gaea: 模型中心未初始化")
	}
	cfg := modelengine.EngineConfig{
		ID:           p.Name,
		Name:         p.Name,
		Type:         modelengine.EngineType(p.Kind),
		BaseURL:      p.BaseURL,
		DefaultModel: p.Default,
		Enabled:      true,
	}
	return a.engineMgr.SaveEngine(cfg)
}

// GaeaDeleteProvider 删除模型中心引擎。模型中心引擎固定，不支持删除。
func (a *App) GaeaDeleteProvider(name string) error {
	return fmt.Errorf("模型中心引擎 %s 为内置固定引擎，不支持删除", name)
}

// GaeaLoginProvider 触发 xAI OAuth 登录（办公板块 provider 面板入口）。
// 仅 xai 引擎支持 OAuth 登录，其余引擎走 API Key 配置。
func (a *App) GaeaLoginProvider(name string) error {
	if name != "xai" && name != "xAI (Grok)" {
		return fmt.Errorf("引擎 %s 不支持 OAuth 登录，请直接配置 API Key", name)
	}
	return a.Login()
}

// GaeaLogoutProvider 登出 xAI（清除本地 token）。
func (a *App) GaeaLogoutProvider(name string) error {
	if name != "xai" && name != "xAI (Grok)" {
		return fmt.Errorf("引擎 %s 不支持 OAuth 登出", name)
	}
	if a.cfg == nil || a.cfg.TokenStorePath == "" {
		return errors.New("gaea: token 存储路径未配置")
	}
	store := auth.NewTokenStore(a.cfg.TokenStorePath)
	if err := store.Delete(); err != nil {
		return fmt.Errorf("清除 xAI token 失败: %w", err)
	}
	if a.engineMgr != nil {
		a.engineMgr.UpdateXAIKey("")
	}
	a.client = ai.NewClient(a.cfg)
	a.configureClient()
	return nil
}

// GaeaSetProviderKey 设置引擎 API Key（按引擎映射到模型中心的 key 槽位）。
func (a *App) GaeaSetProviderKey(apiKeyEnv, value string) error {
	if a.engineMgr == nil {
		return errors.New("gaea: 模型中心未初始化")
	}
	env := strings.ToLower(apiKeyEnv)
	switch {
	case strings.Contains(env, "xai") || strings.Contains(env, "grok"):
		a.engineMgr.UpdateXAIKey(value)
		return nil
	case strings.Contains(env, "deepseek") || strings.Contains(env, "deep"):
		a.engineMgr.UpdateDeepseekKey(value)
		return nil
	default:
		return fmt.Errorf("不支持的引擎 Key 环境变量 %q", apiKeyEnv)
	}
}

// GaeaSetPermissionMode 设置权限模式（ask/auto/yolo），持久化并重建。
func (a *App) GaeaSetPermissionMode(mode string) error {
	return a.gaeaApplyCfg(func(cfg *gaeaConfig.Config) {
		cfg.Permissions.Mode = mode
	})
}

// GaeaAddPermissionRule 追加权限规则（allow/ask/deny 列表）。
func (a *App) GaeaAddPermissionRule(list, rule string) error {
	if rule == "" {
		return errors.New("规则不能为空")
	}
	return a.gaeaApplyCfg(func(cfg *gaeaConfig.Config) {
		perms := &cfg.Permissions
		switch list {
		case "allow":
			perms.Allow = appendUnique(perms.Allow, rule)
		case "ask":
			perms.Ask = appendUnique(perms.Ask, rule)
		case "deny":
			perms.Deny = appendUnique(perms.Deny, rule)
		}
	})
}

// GaeaRemovePermissionRule 移除权限规则。
func (a *App) GaeaRemovePermissionRule(list, rule string) error {
	return a.gaeaApplyCfg(func(cfg *gaeaConfig.Config) {
		perms := &cfg.Permissions
		switch list {
		case "allow":
			perms.Allow = removeString(perms.Allow, rule)
		case "ask":
			perms.Ask = removeString(perms.Ask, rule)
		case "deny":
			perms.Deny = removeString(perms.Deny, rule)
		}
	})
}

// GaeaSetSandbox 设置沙箱参数（bash 模式/网络/写入根），持久化并重建。
func (a *App) GaeaSetSandbox(bash string, network bool, workspaceRoot string, allowWrite []string) error {
	return a.gaeaApplyCfg(func(cfg *gaeaConfig.Config) {
		cfg.Sandbox.Bash = bash
		cfg.Sandbox.Network = network
		if workspaceRoot != "" {
			cfg.Sandbox.WorkspaceRoot = workspaceRoot
		}
		if allowWrite != nil {
			cfg.Sandbox.AllowWrite = allowWrite
		}
	})
}

// GaeaSetAgentParams 设置 agent 核心参数（温度/最大步数/系统提示词），持久化并重建。
func (a *App) GaeaSetAgentParams(temperature float64, maxSteps int, systemPrompt string) error {
	return a.gaeaApplyCfg(func(cfg *gaeaConfig.Config) {
		cfg.Agent.Temperature = temperature
		cfg.Agent.MaxSteps = maxSteps
		if systemPrompt != "" {
			cfg.Agent.SystemPrompt = systemPrompt
		}
	})
}

// GaeaSetSubagentEffort 设置子代理推理强度。
func (a *App) GaeaSetSubagentEffort(effort string) error {
	return a.gaeaApplyCfg(func(cfg *gaeaConfig.Config) { cfg.Agent.SubagentEffort = effort })
}
// GaeaSetSubagentModelForSkill 设置指定技能的子代理模型。
// GaeaSetSubagentModelForSkill 设置指定技能的子代理模型。
func (a *App) GaeaSetSubagentModelForSkill(skill, ref string) error {
	return a.gaeaApplyCfg(func(cfg *gaeaConfig.Config) {
		if cfg.Agent.SubagentModels == nil {
			cfg.Agent.SubagentModels = map[string]string{}
		}
		cfg.Agent.SubagentModels[skill] = ref
	})
}

// appendUnique 追加元素（去重）。

// appendUnique 追加元素（去重）。
func appendUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

// removeString 移除元素。
func removeString(list []string, v string) []string {
	out := list[:0]
	for _, x := range list {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}

// GaeaSetPermLevel 设置权限级别（ask/auto/yolo），真实生效。
func (a *App) GaeaSetPermLevel(level string) error {
	if c := gaeaCtrl(); c != nil {
		c.SetPermLevel(level)
	}
	return nil
}

// GaeaAgentMode/GaeaPermLevel 返回模式。
func (a *App) GaeaAgentMode() string { return "" }
func (a *App) GaeaPermLevel() string {
	if c := gaeaCtrl(); c != nil {
		return c.PermLevel()
	}
	return "ask"
}

// ── 工作区文件（对齐 gaeaW desktop/app_workspace.go）────────────────

// DirEntry 是工作区目录条目。
type DirEntry struct {
	Name  string `json:"name"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size"`
}

// FilePreview 是文件预览。
type FilePreview struct {
	Path     string `json:"path"`
	Markdown string `json:"markdown"`
	Size     int64  `json:"size"`
}

// WorkspaceChangeView 是会话期间被修改的文件。
type WorkspaceChangeView struct {
	Path    string `json:"path"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
}

// gaeaCwd 返回办公引擎工作目录：优先当前工作空间配置，回退进程启动目录。
func gaeaCwd() string {
	if ga.cfg != nil && ga.cfg.Workspace != "" {
		return ga.cfg.Workspace
	}
	cwd, _ := os.Getwd()
	return cwd
}

// GaeaListDir 列出工作区相对路径下的目录。
func (a *App) GaeaListDir(rel string) []DirEntry {
	out := []DirEntry{}
	root := gaeaCwd()
	dir := root
	if rel != "" {
		dir = filepath.Join(root, rel)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		out = append(out, DirEntry{Name: e.Name(), IsDir: e.IsDir(), Size: size})
	}
	return out
}

// GaeaReadFile 读取工作区相对路径的文件文本。
func (a *App) GaeaReadFile(rel string) FilePreview {
	path := filepath.Join(gaeaCwd(), rel)
	b, err := os.ReadFile(path)
	if err != nil {
		return FilePreview{Path: rel}
	}
	return FilePreview{Path: rel, Markdown: string(b), Size: int64(len(b))}
}

// GaeaOpenWorkspacePath/GaeaRevealWorkspacePath 在文件管理器中打开/定位。
func (a *App) GaeaOpenWorkspacePath(rel string) error {
	return exec.Command("explorer", filepath.Join(gaeaCwd(), rel)).Start()
}
func (a *App) GaeaRevealWorkspacePath(rel string) error {
	return exec.Command("explorer", "/select,", filepath.Join(gaeaCwd(), rel)).Start()
}

// GaeaWorkspaceChanges 办公板块不追踪工作区变更，返回空。
func (a *App) GaeaWorkspaceChanges() []WorkspaceChangeView { return []WorkspaceChangeView{} }

// GaeaPickDirectory 使用系统目录对话框选择导出目录（记忆归档导出用）。
func (a *App) GaeaPickDirectory() string {
	if a.ctx == nil {
		return ""
	}
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "选择导出目录",
		DefaultDirectory: gaeaCwd(),
	})
	if err != nil {
		return ""
	}
	return dir
}

// GaeaPickFiles 使用系统文件对话框选择文件。
func (a *App) GaeaPickFiles() []FilePickResult {
	if a.ctx == nil {
		return []FilePickResult{}
	}
	files, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "选择文件",
		DefaultDirectory: gaeaCwd(),
	})
	if err != nil {
		return []FilePickResult{}
	}
	out := make([]FilePickResult, 0, len(files))
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		out = append(out, FilePickResult{Path: f, Name: filepath.Base(f), Size: info.Size()})
	}
	return out
}

// FilePickResult 是文件选择结果。
type FilePickResult struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// GaeaSavePastedImage 保存粘贴图片到工作区 .gaea/uploads。
func (a *App) GaeaSavePastedImage(dataURL string) (string, error) {
	dir := filepath.Join(gaeaCwd(), ".gaea", "uploads")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	// dataURL 形如 "data:image/png;base64,xxxx"
	comma := strings.Index(dataURL, ",")
	if comma < 0 {
		return "", errors.New("无效的图片 dataURL")
	}
	mime := strings.TrimSuffix(strings.TrimPrefix(dataURL[:comma], "data:"), ";base64")
	ext := ".png"
	switch mime {
	case "image/jpeg":
		ext = ".jpg"
	case "image/webp":
		ext = ".webp"
	case "image/gif":
		ext = ".gif"
	}
	b, err := base64.StdEncoding.DecodeString(dataURL[comma+1:])
	if err != nil {
		return "", fmt.Errorf("图片解码失败: %w", err)
	}
	name := fmt.Sprintf("paste-%d%s", time.Now().UnixNano(), ext)
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// GaeaSaveAttachmentFile 保存附件文件到工作区 .gaea/uploads。
func (a *App) GaeaSaveAttachmentFile(fileName, base64Data string) (string, error) {
	dir := filepath.Join(gaeaCwd(), ".gaea", "uploads")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	b, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return "", fmt.Errorf("附件解码失败: %w", err)
	}
	name := fmt.Sprintf("attach-%d-%s", time.Now().UnixNano(), filepath.Base(fileName))
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// GaeaAttachmentDataURL 读取附件为 dataURL。
func (a *App) GaeaAttachmentDataURL(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	mime := "application/octet-stream"
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		mime = "image/png"
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".gif":
		mime = "image/gif"
	case ".webp":
		mime = "image/webp"
	case ".pdf":
		mime = "application/pdf"
	case ".txt":
		mime = "text/plain"
	case ".md":
		mime = "text/markdown"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(b), nil
}

// ── 工作区切换 / MCP / 更新 / 其他 ────────────────────────────────

// GaeaListWorkspaces 返回工作区列表（gaea 单工作区：当前工作空间）。
func (a *App) GaeaListWorkspaces() []WorkspaceView {
	cwd := gaeaCwd()
	return []WorkspaceView{{Path: cwd, Name: filepath.Base(cwd), Current: true}}
}

// WorkspaceView 是工作区条目。
type WorkspaceView struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Current bool   `json:"current"`
}

// GaeaPickWorkspace 弹出系统目录对话框选择/新建工作空间并切换。
// 用户取消或对话框失败返回空串（前端 no-op）。
func (a *App) GaeaPickWorkspace() string {
	if a.ctx == nil {
		return ""
	}
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "选择工作空间（可在对话框内新建文件夹）",
		DefaultDirectory: gaeaCwd(),
	})
	if err != nil || dir == "" {
		return ""
	}
	return a.GaeaSwitchWorkspace(dir)
}

// GaeaSwitchWorkspace 切换并持久化工作空间；无效路径保持当前工作空间。
func (a *App) GaeaSwitchWorkspace(path string) string {
	if path == "" {
		return gaeaCwd()
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return gaeaCwd()
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return gaeaCwd()
	}
	return a.switchWorkspace(abs)
}

// switchWorkspace 切换并持久化工作空间，随后重建办公引擎使会话目录生效。
func (a *App) switchWorkspace(abs string) string {
	ga.mu.Lock()
	defer ga.mu.Unlock()
	if err := a.persistWorkspaceLocked(abs); err != nil {
		slog.Error("保存工作空间失败", "error", err)
		return gaeaCwd()
	}
	// 重建办公引擎使会话目录跟随新工作空间。失败仅记日志（工作空间已持久化，
	// 下次启动生效），绝不让引擎重建问题阻塞工作空间切换。
	func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("重建办公引擎 panic recovered（工作空间已切换）", "panic", r)
			}
		}()
		if err := a.gaeaRebuildLocked(); err != nil {
			slog.Error("重建办公引擎失败（工作空间已持久化）", "error", err)
		}
	}()
	return abs
}

// persistWorkspaceLocked 持久化工作空间路径到内存配置与用户配置文件。
// 调用方必须已持有 ga.mu。
func (a *App) persistWorkspaceLocked(abs string) error {
	if ga.cfg == nil {
		cfg, err := gaeaLoadConfig()
		if err != nil {
			return err
		}
		ga.cfg = cfg
	}
	ga.cfg.Workspace = abs
	return gaeaConfig.Save(ga.cfg)
}

// GaeaAddMCPServer 添加 MCP 服务器（真实生效）。
func (a *App) GaeaAddMCPServer(input MCPServerInput) (int, error) {
	c := gaeaCtrl()
	if c == nil {
		return 0, errors.New("办公引擎未初始化")
	}
	entry := gaeaConfig.PluginEntry{Name: input.Name, Command: input.Command, Args: input.Args, Env: input.Env}
	if input.URL != "" {
		entry.URL = input.URL
	}
	return c.AddMCPServer(entry)
}

// MCPServerInput 是抽屉的添加服务器表单。
type MCPServerInput struct {
	Name      string            `json:"name"`
	Transport string            `json:"transport"`
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	URL       string            `json:"url"`
	Env       map[string]string `json:"env"`
}

// GaeaRemoveMCPServer 移除 MCP 服务器。
func (a *App) GaeaRemoveMCPServer(name string) error {
	if c := gaeaCtrl(); c != nil {
		_, err := c.RemoveMCPServer(name)
		return err
	}
	return nil
}

// GaeaRetryMCPServer 重连已配置的 MCP 服务器。
func (a *App) GaeaRetryMCPServer(name string) error {
	if c := gaeaCtrl(); c != nil {
		_, err := c.ConnectConfiguredMCPServer(name)
		return err
	}
	return nil
}

// GaeaSetMCPServerEnabled 启用/停用 MCP 服务器。
func (a *App) GaeaSetMCPServerEnabled(name string, enabled bool) error {
	c := gaeaCtrl()
	if c == nil {
		return nil
	}
	if enabled {
		_, err := c.ConnectConfiguredMCPServer(name)
		return err
	}
	c.DisconnectMCPServer(name)
	return nil
}

// GaeaSelectTab/GaeaTabMeta 办公板块为单标签，返回空。
func (a *App) GaeaSelectTab(tabID string) error { return nil }
func (a *App) GaeaTabMeta() []TabMeta           { return []TabMeta{} }

// TabMeta 是标签元信息。
type TabMeta struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// GaeaCheckUpdate/GaeaApplyUpdate/GaeaOpenDownloadPage 更新由 gaea 自身管理。
func (a *App) GaeaCheckUpdate() (*UpdateInfo, error) { return nil, nil }
func (a *App) GaeaApplyUpdate() error {
	return errors.New("桌面版无自动更新机制，请从发布渠道获取新版本")
}
func (a *App) GaeaOpenDownloadPage() error {
	return errors.New("桌面版无自动更新机制，请从发布渠道获取新版本")
}

// UpdateInfo 是更新信息。
type UpdateInfo struct {
	Version string `json:"version"`
	Notes   string `json:"notes"`
}

// GaeaSaveWindowState 窗口状态由 gaea 主窗口管理。
func (a *App) GaeaSaveWindowState(state map[string]interface{}) error { return nil }

// GaeaMemorySuggestions 返回记忆建议（办公板块不提供，返回空）。
func (a *App) GaeaMemorySuggestions() MemorySuggestionsView { return MemorySuggestionsView{} }

// MemorySuggestionsView 是记忆建议负载。
type MemorySuggestionsView struct {
	Facts  []interface{} `json:"facts"`
	Skills []interface{} `json:"skills"`
}

func (a *App) GaeaAcceptMemorySuggestion(candidate interface{}) (string, error) {
	return "", errors.New("办公板块不提供记忆建议")
}
func (a *App) GaeaAcceptSkillSuggestion(candidate interface{}) (string, error) {
	return "", errors.New("办公板块不提供技能建议")
}
