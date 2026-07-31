package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/config"
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
	PlannerTemperature  float64 `json:"plannerTemperature"`
	SubagentTemperature float64 `json:"subagentTemperature"`
	Effort              string  `json:"effort"`
	PlannerEffort       string  `json:"plannerEffort"`
	SubagentEffort      string  `json:"subagentEffort"`
}

// SettingsView 是设置面板完整负载。
type SettingsView struct {
	DefaultModel   string            `json:"defaultModel"`
	PlannerModel   string            `json:"plannerModel"`
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
	cfg, err := config.Load()
	if err != nil {
		cfg = config.Default()
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
	view.Agent = AgentView{Temperature: cfg.Agent.Temperature, MaxSteps: cfg.Agent.MaxSteps, SystemPrompt: cfg.Agent.SystemPrompt, Effort: cfg.Agent.Effort, PlannerEffort: cfg.Agent.PlannerEffort, SubagentEffort: cfg.Agent.SubagentEffort}
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
		return errNotSupported
	}
	return a.engineMgr.SetDefaultModel(engine, model)
}

// GaeaSaveProvider/GaeaDeleteProvider/GaeaLoginProvider/GaeaLogoutProvider/GaeaSetProviderKey
// 引擎增删改由 gaea 模型中心管理，办公板块不直接操作。
func (a *App) GaeaSaveProvider(p ProviderView) error            { return errNotSupported }
func (a *App) GaeaDeleteProvider(name string) error             { return errNotSupported }
func (a *App) GaeaLoginProvider(name string) error              { return errNotSupported }
func (a *App) GaeaLogoutProvider(name string) error             { return errNotSupported }
func (a *App) GaeaSetProviderKey(apiKeyEnv, value string) error { return errNotSupported }

// GaeaSetPermissionMode/GaeaAddPermissionRule/GaeaRemovePermissionRule/GaeaSetSandbox
// 权限与沙箱由办公引擎配置注入时确定，不支持运行时修改。
func (a *App) GaeaSetPermissionMode(mode string) error          { return errNotSupported }
func (a *App) GaeaAddPermissionRule(list, rule string) error    { return errNotSupported }
func (a *App) GaeaRemovePermissionRule(list, rule string) error { return errNotSupported }
func (a *App) GaeaSetSandbox(bash string, network bool, workspaceRoot string, allowWrite []string) error {
	return errNotSupported
}

// GaeaSetAgentParams 等 agent 参数不支持运行时热更。
func (a *App) GaeaSetAgentParams(temperature float64, maxSteps int, systemPrompt string) error {
	return errNotSupported
}
func (a *App) GaeaSetPlannerTemperature(temp float64) error         { return errNotSupported }
func (a *App) GaeaSetSubagentTemperature(temp float64) error        { return errNotSupported }
func (a *App) GaeaSetEffort(effort string) error                    { return errNotSupported }
func (a *App) GaeaSetPlannerEffort(effort string) error             { return errNotSupported }
func (a *App) GaeaSetSubagentEffort(effort string) error            { return errNotSupported }
func (a *App) GaeaSetSubagentModel(ref string) error                { return errNotSupported }
func (a *App) GaeaSetSubagentModelForSkill(skill, ref string) error { return errNotSupported }
func (a *App) GaeaSetPlannerModel(ref string) error                 { return errNotSupported }

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

// gaeaCwd 返回办公引擎工作目录。
func gaeaCwd() string {
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
	return errNotSupported
}
func (a *App) GaeaRevealWorkspacePath(rel string) error {
	return errNotSupported
}

// GaeaWorkspaceChanges 办公板块不追踪工作区变更，返回空。
func (a *App) GaeaWorkspaceChanges() []WorkspaceChangeView { return []WorkspaceChangeView{} }

// GaeaPickFiles 使用系统文件对话框选择文件。
func (a *App) GaeaPickFiles() []FilePickResult { return []FilePickResult{} }

// FilePickResult 是文件选择结果。
type FilePickResult struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// GaeaSavePastedImage 保存粘贴图片到工作区 .gaea/uploads。
func (a *App) GaeaSavePastedImage(dataURL string) (string, error) {
	return "", errNotSupported
}

// GaeaSaveAttachmentFile 保存附件文件。
func (a *App) GaeaSaveAttachmentFile(fileName, base64Data string) (string, error) {
	return "", errNotSupported
}

// GaeaAttachmentDataURL 读取附件为 dataURL。
func (a *App) GaeaAttachmentDataURL(path string) (string, error) {
	return "", errNotSupported
}

// ── 工作区切换 / MCP / 更新 / 其他 ────────────────────────────────

// GaeaListWorkspaces 返回工作区列表（gaea 单工作区：当前目录）。
func (a *App) GaeaListWorkspaces() []WorkspaceView { return []WorkspaceView{{Path: gaeaCwd()}} }

// WorkspaceView 是工作区条目。
type WorkspaceView struct {
	Path string `json:"path"`
}

// GaeaPickWorkspace 选择并切换工作区（gaea 办公板块固定当前目录）。
func (a *App) GaeaPickWorkspace() string { return "" }
func (a *App) GaeaSwitchWorkspace(path string) string {
	return gaeaCwd()
}

// GaeaAddMCPServer 添加 MCP 服务器（真实生效）。
func (a *App) GaeaAddMCPServer(input MCPServerInput) (int, error) {
	c := gaeaCtrl()
	if c == nil {
		return 0, errors.New("办公引擎未初始化")
	}
	entry := config.PluginEntry{Name: input.Name, Command: input.Command, Args: input.Args, Env: input.Env}
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
func (a *App) GaeaApplyUpdate() error                { return errNotSupported }
func (a *App) GaeaOpenDownloadPage() error           { return errNotSupported }

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
	return "", errNotSupported
}
func (a *App) GaeaAcceptSkillSuggestion(candidate interface{}) (string, error) {
	return "", errNotSupported
}
