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
	"github.com/gaea/gaea/internal/channels/weixin"
	"github.com/gaea/gaea/internal/chapter"
	"github.com/gaea/gaea/internal/character"
	"github.com/gaea/gaea/internal/characterlib"
	"github.com/gaea/gaea/internal/chat"
	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/httpbridge"
	"github.com/gaea/gaea/internal/modelengine"
	officedb "github.com/gaea/gaea/internal/office/db"
	"github.com/gaea/gaea/internal/office/proposal"
	"github.com/gaea/gaea/internal/outline"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/skill"
	"github.com/gaea/gaea/internal/voice"
	"github.com/gaea/gaea/internal/whisper"
	"github.com/gaea/gaea/internal/worldview"
)

// core 是所有子服务共享的基础依赖（ctx/client/cfg/engineMgr 等）。
// 通过指针嵌入到 App 与各子服务，保证同一实例、方法体零改动。
type core struct {
	ctx    context.Context
	cfg    *config.Config
	client *ai.Client

	// 模型引擎管理器
	engineMgr *modelengine.Manager

	// 统一聊天会话存储（聊天/轻语合并：话题 + 消息）
	chatStore *chat.Store

	// 全局统一角色库（小说 × 聊天共享的角色资产）
	charLib *characterlib.Store

	// 前端静态资源
	distFS fs.FS
}

// writingState 是小说写作域状态（章节/角色/大纲/世界观/分析/图谱）。
type writingState struct {
	*core

	// app 反向引用：跨域调用（媒体/轻语）经 App 协调。
	app *App

	mu sync.RWMutex // 保护 pm 的并发读写

	// 项目管理
	pm *project.Manager

	// 共享 prompt engine（单例）
	eng *prompt.Engine

	// 子代理
	worldviewAgent *worldview.Agent
	characterAgent *character.Agent
	outlineAgent   *outline.Agent
	chapterAgent   *chapter.Agent
	analysisAgent  *analysis.Agent

	// Skill
	skillLoader *skill.Loader
}

// mediaState 是媒体域状态（图像生成/TTS/语音/ComfyUI 进程）。
type mediaState struct {
	*core

	// app 反向引用：跨域调用（写作/轻语）经 App 协调。
	app *App

	// TTS 语音朗读（模型中心选择）
	activeTTSEngine string
	activeTTSModel  string

	// ASR 语音识别（模型中心选择）
	activeASREngine string
	activeASRModel  string

	// 语音管道
	voiceManager *voice.Manager

	// ComfyUI 进程管理
	comfyUICancel context.CancelFunc
	comfyUICmd    *exec.Cmd
}

// whisperState 是轻语域状态（AI 人格/虚拟助手/微信通道）。
type whisperState struct {
	*core

	// app 反向引用：跨域调用（媒体）经 App 协调。
	app *App

	// 轻语模块数据根目录（SQLite 持久化）
	whisperDataRoot string

	// 虚拟助手管理器
	assistantMgr *assistant.Manager
	// 微信通道（多实例：assistantID → Server）
	weixinServers map[string]*weixin.Server
	weixinMu      sync.Mutex
}

// officeState 是办公域状态（方案编写模块）。
type officeState struct {
	*core

	// app 反向引用：跨域调用经 App 协调。
	app *App

	proposalSvc *proposal.Service
	batchCancel context.CancelFunc
}

// App Wails 应用实例 — 聚合各域子服务，方法经嵌入提升到 App 供前端绑定。
// Wails 要求绑定单对象：所有子服务方法通过 Go 嵌入自动提升，前端调用不变。
type App struct {
	*core
	*writingState
	*mediaState
	*whisperState
	*officeState

	brain *BrainStore

	modules *ModuleRegistry
}

// emit 统一事件发射 — 发送到 Wails 前端。定义在 core 上，
// 子服务内嵌 core 后直接可用（App 经嵌入也获得该方法）。
func (c *core) emit(eventName string, data map[string]interface{}) {
	// 本地 HTTP 桥接订阅（网页/移动端调试）：无 Wails 上下文也发布。
	httpbridge.Publish(eventName, data)
	if c.ctx == nil {
		return
	}
	// Wails runtime 要求 ctx 携带生命周期值（"events"）；非 Wails 上下文（测试等）
	// 直接调用会 log.Fatalf 退出进程，这里预检后跳过发射。
	if c.ctx.Value("events") == nil {
		return
	}
	runtime.EventsEmit(c.ctx, eventName, data)
}

// New 创建 App 实例
func New() *App {
	cfg := config.Load()
	c := &core{cfg: cfg, client: ai.NewClient(cfg)}
	a := &App{core: c}
	a.writingState = &writingState{
		core: c,
		app:  a,
		eng:  prompt.NewEngine(filepath.Join(cfg.ResourceDir, "prompts")),
		mu:   sync.RWMutex{},
	}
	a.mediaState = &mediaState{core: c, app: a}
	a.whisperState = &whisperState{
		core:            c,
		app:             a,
		whisperDataRoot: filepath.Join(cfg.ResourceDir, "whisper_data"),
		weixinServers:   map[string]*weixin.Server{},
	}
	a.officeState = &officeState{core: c, app: a}
	return a
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
	if err := a.engineMgr.LoadState(filepath.Join(a.whisperDataRoot, "engines.json")); err != nil {
		slog.Warn("加载引擎状态失败（回退预置默认）", "error", err)
	}
	if tok, err := auth.NewTokenStore(a.cfg.TokenStorePath).Load(); err == nil && tok != nil && !tok.IsExpired() {
		a.engineMgr.UpdateXAIKey(tok.AccessToken)
	}
	a.configureClient()
	a.initImageBackend()

	// 恢复模型中心语音模型选择（持久化自 ~/.gaea_config.json）
	a.activeASREngine = a.cfg.ActiveASREngine
	a.activeASRModel = a.cfg.ActiveASRModel
	a.activeTTSEngine = a.cfg.ActiveTTSEngine
	a.activeTTSModel = a.cfg.ActiveTTSModel

	a.initVoice()
	a.initWeixin()

	// 全局角色库：内置人格 + 助手全部种子化为统一角色
	a.charLib = characterlib.NewStore(filepath.Join(a.whisperDataRoot, "characterlib"))
	if a.charLib != nil {
		if err := a.charLib.EnsureBuiltins(whisper.PersonalityPresets); err != nil {
			slog.Error("角色库种子化内置角色失败", "error", err)
		}
		if err := a.charLib.EnsureAssistants(a.assistantMgr.List(), whisper.PersonalityPresets); err != nil {
			slog.Error("角色库同步助手失败", "error", err)
		}
	}

	// 初始化方案编写模块
	a.proposalSvc = proposal.NewService(a.whisperDataRoot, a.client)
	// 统一聊天会话存储（chat.db）
	a.chatStore = chat.NewStore(filepath.Join(a.whisperDataRoot, "chat"))
	a.initBrain()
	a.initModules()

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
			} else {
				// 模型列表就绪后重配 ASR（用户选择/自动扫描 STT 模型）
				a.applyASRClient()
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
	if err := officedb.CloseDatabase(filepath.Join(a.whisperDataRoot, "office")); err != nil {
		slog.Error("关闭 office.db 失败", "error", err)
	}
	if err := chat.CloseDatabase(filepath.Join(a.whisperDataRoot, "chat")); err != nil {
		slog.Error("关闭 chat.db 失败", "error", err)
	}
	if err := characterlib.CloseDatabase(filepath.Join(a.whisperDataRoot, "characterlib")); err != nil {
		slog.Error("关闭 characterlib.db 失败", "error", err)
	}
}

// getPM/setPM/closePM/initAgents/restoreImageBackend 已迁移到
// writing_state.go（writingState 域）。App 经嵌入提升，Startup 调用不变。
// initWeixin/startAssistantWx/stopAssistantWx 已迁移到 whisper_state.go。

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
