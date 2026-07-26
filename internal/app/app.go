package app

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"

	"log/slog"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/wubigork/wubigork/internal/ai"
	"github.com/wubigork/wubigork/internal/analysis"
	"github.com/wubigork/wubigork/internal/auth"
	"github.com/wubigork/wubigork/internal/chapter"
	"github.com/wubigork/wubigork/internal/character"
	"github.com/wubigork/wubigork/internal/config"
	"github.com/wubigork/wubigork/internal/modelengine"
	"github.com/wubigork/wubigork/internal/outline"
	"github.com/wubigork/wubigork/internal/project"
	"github.com/wubigork/wubigork/internal/prompt"
	"github.com/wubigork/wubigork/internal/skill"
	"github.com/wubigork/wubigork/internal/tts"
	"github.com/wubigork/wubigork/internal/worldview"
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
	ttsClient *tts.Client

	// ComfyUI 进程管理
	comfyUICancel context.CancelFunc

	// Ghost Text 取消控制器
	ghostCancel context.CancelFunc

	// 前端静态资源
	distFS fs.FS
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
		cfg: cfg,
		eng: prompt.NewEngine(filepath.Join(cfg.ResourceDir, "prompts")),
	}
}
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// 创建 AI client（仅此一次；token 由 GetToken 懒加载）
	a.client = ai.NewClient(a.cfg)

	// 初始化模型引擎管理器，尝试恢复已保存的 xAI token
	a.engineMgr = modelengine.NewManager("")
	if tok, err := auth.NewTokenStore(a.cfg.TokenStorePath).Load(); err == nil && tok != nil && !tok.IsExpired() {
		a.engineMgr.UpdateXAIKey(tok.AccessToken)
		// 后台自动刷新 xAI 模型列表
		go func() {
			if _, err := a.engineMgr.RefreshModels(context.Background(), "xai"); err != nil {
				slog.Warn("启动时刷新xAI模型列表失败", "error", err)
			} else {
				slog.Info("xAI模型列表已自动刷新")
			}
		}()
	}
	a.configureClient()
	a.initImageBackend()

	// 自动启动 TTS 服务（后台加载，不阻塞）
	go a.autoStartTTS()
}

// initImageBackend 根据配置初始化图片生成后端
func (a *App) initImageBackend() {
	if a.cfg.ImageBackend == "comfyui" && a.cfg.ComfyUIURL != "" {
		backend := ai.NewComfyUIBackend(a.cfg.ComfyUIURL)
		a.client.SetImageBackend(backend)
		slog.Info("图片后端: ComfyUI", "url", a.cfg.ComfyUIURL)
	} else {
		slog.Info("图片后端: xAI", "backend", a.cfg.ImageBackend)
	}
}

// Shutdown Wails 关闭回调
func (a *App) Shutdown(ctx context.Context) {
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
}

// SetDistFS 设置前端静态资源 embed.FS（由 main.go 在启动前调用）
func (a *App) SetDistFS(fsys fs.FS) {
	a.distFS = fsys
}

func CLILogin() {
	cfg := config.Load()
	fmt.Println("🚀 wubigork — 小说 AI Agent")
	fmt.Println("============================")

	store := auth.NewTokenStore(cfg.TokenStorePath)
	tok, _ := store.Load()
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
