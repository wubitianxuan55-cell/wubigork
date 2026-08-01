package app

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/analysis"
	"github.com/gaea/gaea/internal/assistant"
	"github.com/gaea/gaea/internal/auth"
	"github.com/gaea/gaea/internal/chapter"
	"github.com/gaea/gaea/internal/character"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
	"github.com/gaea/gaea/internal/outline"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/skill"
	"github.com/gaea/gaea/internal/voice"
	"github.com/gaea/gaea/internal/channels/weixin"
	"github.com/gaea/gaea/internal/office/proposal"
	"github.com/gaea/gaea/internal/worldview"
)

// App Wails 应用实例 — 管理所有 Agent 和生命周期
type App struct {
	ctx    context.Context
	cfg    *config.Config
	client *ai.Client

	mu sync.RWMutex // 保护 pm 的并发读写

	// 项目管理
	pm *project.Manager

	// 共享 prompt engine（单例）
	eng *prompt.Engine

	// 模型引擎管理器
	engineMgr *modelengine.Manager

	// 子代理
	worldviewAgent *worldview.Agent
	characterAgent *character.Agent
	outlineAgent   *outline.Agent
	chapterAgent   *chapter.Agent
	analysisAgent  *analysis.Agent

	// Skill
	skillLoader *skill.Loader
	// TTS 语音朗读
	activeTTSEngine string
	activeTTSModel  string

	// 语音管道
	voiceManager *voice.Manager

	// ComfyUI 进程管理
	comfyUICancel context.CancelFunc
	comfyUICmd    *exec.Cmd

	// Ghost Text 取消控制器
	ghostCancel context.CancelFunc

	// 前端静态资源
	distFS fs.FS

	// 轻语模块数据根目录（SQLite 持久化）
	whisperDataRoot string

	// 虚拟助手管理器
	assistantMgr *assistant.Manager
	// 微信通道（多实例：assistantID → Server）
	weixinServers map[string]*weixin.Server
	weixinMu      sync.Mutex

	// 方案编写模块
	proposalSvc *proposal.Service
}

// emit 统一事件发射 — 发送到 Wails 前端
func (a *App) emit(eventName string, data map[string]interface{}) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, eventName, data)
	}
}

// New 创建 App 实例
func New() *App {
	cfg := config.Load()
	return &App{
		cfg:             cfg,
		eng:             prompt.NewEngine(filepath.Join(cfg.ResourceDir, "prompts")),
		whisperDataRoot: filepath.Join(cfg.ResourceDir, "whisper_data"),
	}
}
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// 将 slog 输出到文件（GUI 应用无控制台）
	logFile, err := os.OpenFile(filepath.Join(a.whisperDataRoot, "gaea.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		slog.SetDefault(slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: slog.LevelInfo})))
		slog.Info("=== gaea startup ===")
	}
	// 创建 AI client（仅此一次；token 由 GetToken 懒加载）
	a.client = ai.NewClient(a.cfg)

	// 初始化模型引擎管理器，尝试恢复已保存的 xAI token
	a.engineMgr = modelengine.NewManager("", a.cfg.DeepseekAPIKey)
	if tok, err := auth.NewTokenStore(a.cfg.TokenStorePath).Load(); err == nil && tok != nil && !tok.IsExpired() {
		a.engineMgr.UpdateXAIKey(tok.AccessToken)
	}
	a.configureClient()
	a.initImageBackend()
	a.initVoice()
	a.initWeixin()

	// 初始化方案编写模块
	a.proposalSvc = proposal.NewService(a.whisperDataRoot, a.client)

	// 后台刷新所有引擎模型列表
	for _, eid := range []string{"xai", "herdsman", "ollama", "deepseek"} {
		go func(id string) {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("app: refresh models goroutine panic recovered", "engine", id, "panic", r)
				}
			}()
			eng, ok := a.engineMgr.GetEngine(id)
			if !ok || !eng.Enabled {
				return
			}
			if _, err := a.engineMgr.RefreshModels(context.Background(), id); err != nil {
				slog.Warn("刷新"+id+"模型列表失败", "error", err)
			}
		}(eid)
	}
}

// initImageBackend 根据配置初始化图片生成后端
func (a *App) initImageBackend() {
	switch a.cfg.ImageBackend {
	case "comfyui":
		if a.cfg.ComfyUIURL != "" {
			backend := ai.NewComfyUIBackend(a.cfg.ComfyUIURL)
			a.client.SetImageBackend(backend, "comfyui")
			slog.Info("图片后端: ComfyUI", "url", a.cfg.ComfyUIURL)
		}
	case "herdsman":
		eng, ok := a.engineMgr.GetEngine("herdsman")
		if ok && eng.Enabled {
			backend := ai.NewOpenAIImageBackend(eng.BaseURL, eng.APIKey)
			a.client.SetImageBackend(backend, "herdsman")
			slog.Info("图片后端: Herdsman", "url", eng.BaseURL)
		}
	case "ollama":
		eng, ok := a.engineMgr.GetEngine("ollama")
		if ok && eng.Enabled {
			backend := ai.NewOpenAIImageBackend(eng.BaseURL, eng.APIKey)
			a.client.SetImageBackend(backend, "ollama")
			slog.Info("图片后端: Ollama", "url", eng.BaseURL)
		}
	default: // "xai" 或空
		a.client.SetImageBackend(nil, "xai")
		slog.Info("图片后端: xAI")
	}
}

// Shutdown Wails 关闭回调
func (a *App) Shutdown(ctx context.Context) {
	if a.voiceManager != nil {
		a.voiceManager.Stop()
	}
	if a.weixinServers != nil {
		a.weixinMu.Lock()
		for id, srv := range a.weixinServers {
			srv.Stop()
			delete(a.weixinServers, id)
		}
		a.weixinMu.Unlock()
	}
	if err := a.closePM(); err != nil {
		slog.Error("关闭项目失败", "error", err)
	}
}

// ── 并发安全访问器 ────────────────────────────────────────────

// getPM 以读锁获取当前项目（调用方用完即释放引用）
func (a *App) getPM() *project.Manager {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.pm
}

// requirePM 以读锁获取当前项目，未打开时返回错误
func (a *App) requirePM() (*project.Manager, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	return pm, nil
}

// setPM 以写锁设置当前项目
func (a *App) setPM(pm *project.Manager) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pm = pm
}

// closePM 以写锁关闭并清空当前项目
func (a *App) closePM() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.pm == nil {
		return nil
	}
	err := a.pm.Close()
	a.pm = nil
	return err
}

func (a *App) initAgents() {
	a.worldviewAgent = worldview.New(a.client, a.pm, a.cfg, a.eng)
	a.characterAgent = character.New(a.client, a.pm, a.cfg, a.eng)
	a.outlineAgent = outline.New(a.client, a.pm, a.cfg, a.eng)
	a.chapterAgent = chapter.New(a.client, a.pm, a.cfg, a.eng)
	a.analysisAgent = analysis.New(a.client, a.pm, a.cfg, a.eng)
	a.skillLoader = skill.NewLoader(filepath.Join(a.cfg.ResourceDir, "skills"))

	// 恢复上次保存的图像后端配置
	a.restoreImageBackend()
}

// restoreImageBackend 从配置恢复图像后端（应用重启后自动恢复）
func (a *App) restoreImageBackend() {
	if a.client == nil {
		return
	}
	switch a.cfg.ImageBackend {
	case "comfyui":
		if a.cfg.ComfyUIURL != "" {
			a.client.SetImageBackend(ai.NewComfyUIBackend(a.cfg.ComfyUIURL), "comfyui")
			slog.Info("已恢复 ComfyUI 图像后端", "url", a.cfg.ComfyUIURL, "model", a.cfg.ImageModel)
		}
	case "herdsman":
		if a.engineMgr != nil {
			if eng, ok := a.engineMgr.GetEngine("herdsman"); ok && eng.Enabled {
				a.client.SetImageBackend(ai.NewOpenAIImageBackend(eng.BaseURL, eng.APIKey), "herdsman")
				slog.Info("已恢复 Herdsman 图像后端")
			}
		}
	case "ollama":
		if a.engineMgr != nil {
			if eng, ok := a.engineMgr.GetEngine("ollama"); ok && eng.Enabled {
				a.client.SetImageBackend(ai.NewOpenAIImageBackend(eng.BaseURL, eng.APIKey), "ollama")
				slog.Info("已恢复 Ollama 图像后端")
			}
		}
	// xai 不需要恢复（默认就是 xai fallback）
	}
}
// initWeixin 加载助手并启动各自微信通道
func (a *App) initWeixin() {
	var err error
	a.assistantMgr, err = assistant.Load(a.whisperDataRoot)
	if err != nil {
		slog.Error("[assistant] 加载失败，重试", "err", err)
		if a.assistantMgr, err = assistant.Load(a.whisperDataRoot); err != nil {
			slog.Error("[assistant] 重试加载仍失败，使用空管理器", "err", err)
			a.assistantMgr = assistant.NewEmpty(a.whisperDataRoot)
		}
	}
	a.weixinServers = make(map[string]*weixin.Server)

	// 首次启动：创建默认助手
	if len(a.assistantMgr.List()) == 0 {
		defaultAst := assistant.Assistant{
			ID:            "default",
			Name:          "轻语",
			PersonalityID: "deredere",
			Enabled:       true,
		}
		if token := os.Getenv("WXCLAW_TOKEN"); token != "" {
			defaultAst.WxToken = token
		}
		a.assistantMgr.Add(defaultAst)
	}

	for _, ast := range a.assistantMgr.Enabled() {
		a.startAssistantWx(ast)
	}
}

func (a *App) startAssistantWx(ast assistant.Assistant) {
	cfg := weixin.Config{
		ILinkURL:      "https://ilinkai.weixin.qq.com",
		BotToken:      ast.WxToken,
		AssistantID:   ast.ID,
		PersonalityID: ast.PersonalityID,
	}
	srv := weixin.New(cfg, func(userMsg, fromUser string) (string, error) {
		result, err := a.WhisperChatWithSearch(userMsg, ast.PersonalityID)
		if err != nil { return "", err }
		reply, _ := result["reply"].(string)
		if reply == "" { reply = "（思考中…）" }
		return reply, nil
	})
	if err := srv.Start(); err != nil {
		slog.Error("[assistant] 微信启动失败", "assistant", ast.ID, "err", err)
		return
	}
	a.weixinMu.Lock()
	a.weixinServers[ast.ID] = srv
	a.weixinMu.Unlock()
}

func (a *App) stopAssistantWx(id string) {
	a.weixinMu.Lock()
	srv, ok := a.weixinServers[id]
	if ok { delete(a.weixinServers, id) }
	a.weixinMu.Unlock()
	if ok { srv.Stop() }
}

// SetDistFS 设置前端静态资源 embed.FS（由 main.go 在启动前调用）
func (a *App) SetDistFS(fsys fs.FS) {
	a.distFS = fsys
}

func CLILogin() {
	cfg := config.Load()
	fmt.Println("🚀 gaea — 多功能 AI 助手")
	fmt.Println("============================")

	store := auth.NewTokenStore(cfg.TokenStorePath)
	tok, err := store.Load()
	if err != nil {
		fmt.Printf("⚠️ 读取登录状态失败（将重新登录）: %v\n", err)
		tok = nil
	}
	if tok != nil && !tok.IsExpired() {
		fmt.Println("✅ 已登录，无需重复操作")
		return
	}

	result, err := auth.DoLogin(cfg)
	if err != nil {
		fmt.Printf("❌ 登录失败: %v\n", err)
		return
	}

	if err := store.Save(result.Token); err != nil {
		fmt.Printf("⚠️ 保存 token 失败: %v\n", err)
	}
	fmt.Println("✅ 已成功登录 xAI，可以开始写作！")
}
