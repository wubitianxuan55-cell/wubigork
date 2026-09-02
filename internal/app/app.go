package app

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
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
	"github.com/gaea/gaea/internal/gaea/filewatch"
	"github.com/gaea/gaea/internal/gaea/secure"
	"github.com/gaea/gaea/internal/gaea/tasks"
	"github.com/gaea/gaea/internal/httpbridge"
	"github.com/gaea/gaea/internal/intent"
	"github.com/gaea/gaea/internal/modelengine"
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

	// OCR 激活引擎 + 模型（空=自动选择 Herdsman PaddleOCR/MinerU）
	activeOCREngine string
	activeOCRModel  string

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

	// 章节生成互斥与取消（T6-7.2）：key = chapterGenKey(chapterNum, branch)，
	// 记录进行中的章节生成任务。同一章节（同一 NNN.md 目标文件）并发生成被拒绝，
	// 不同章节可并行；CancelCreateChapter 取出并调用取消函数中断流式生成。
	chapterGenMu      sync.Mutex
	chapterGenCancels map[string]context.CancelFunc

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
	activeTTSEngine     string
	activeTTSModel      string
	activeTTSVoice      string
	activePersonalityID string
	chatVoiceEngine     string
	chatVoiceModel      string

	// ASR 语音识别（模型中心选择）
	activeASREngine string
	activeASRModel  string

	// 语音管道
	voiceManager *voice.Manager

	// ComfyUI 进程管理
	comfyUICancel context.CancelFunc
	comfyUICmd    *exec.Cmd

	// 当前图片/视频生成任务（前端生成队列逐条提交，这里只保留取消句柄）
	imageGenMu      sync.Mutex
	imageGenCancel  context.CancelFunc
	imageGenRunning bool
	imageGenID      uint64

	// ComfyUI 任务实时状态（前端轮询 GetComfyUITaskProgress）
	comfyTaskMu      sync.RWMutex
	comfyTaskStatus  string
	comfyTaskElapsed int
	comfyTaskPercent int
	comfyTaskNode    string
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

	// T6-5.3 异步写可观测：记忆写入/持久化协程错误计数 + 最近错误摘要
	// （内部方法读取，不新增 Wails 绑定）。
	writeErrors whisperWriteErrors

	// v4.3c：主动关心定时推送（频控 + 作息尊重 + 特殊日期，play 空间）。
	// proactiveMu 保护配置与「当天已发生日祝福」标记；频控实例各自带锁。
	proactiveMu       sync.Mutex
	proactiveCfgInit  bool
	proactiveCfg      proactivePushCfg
	proactiveStop     chan struct{}
	proactiveOnce     sync.Once
	proactiveStopOnce sync.Once
	attentionMu       sync.Mutex
	attentionManagers map[string]*whisper.AttentionManager // personalityID → 每会话独立频控
	birthdayGreetedMu sync.Mutex
	birthdayGreeted   map[string]string // personalityID → YYYY-MM-DD（当天已发生日祝福）

	// v4.4：微信「离线代办」提醒（书房触点，weixin_reminder.go）。
	remindersMu       sync.Mutex
	reminders         []wxReminder
	remindersLoaded   bool
	reminderStop      chan struct{}
	reminderOnce      sync.Once
	wxPushFunc        wxPushFn // 回推实现（nil = 默认走 weixinServers[].Push；测试注入）
	weixinTaskCfgMu   sync.Mutex
	weixinTaskCfg     weixinTaskCfg
	weixinTaskCfgInit bool
}

// officeState 是办公域状态（桌面动作执行 + 价格源定时抓取 + 文件语义索引）。
type officeState struct {
	*core

	// app 反向引用：跨域调用经 App 协调。
	app *App

	// 价格源定时抓取调度（P1-⑥）：app 启动后按订阅频率检查到期源。
	priceMu       sync.Mutex
	priceCronStop chan struct{}
	priceCronOnce sync.Once
	priceStopOnce sync.Once

	// 文件语义索引自动维护：每 10 分钟增量重建（Ensure 内容感知）。
	fileIndexStop     chan struct{}
	fileIndexOnce     sync.Once
	fileIndexStopOnce sync.Once

	// 阶段 5 T5-1：通用任务调度器（持久任务表 + 进度事件 + 取消/重试/重启续跑）。
	tasks    *tasks.Manager
	taskOnce sync.Once

	// 阶段 5 T5-2：工作区实时文件监听（fsnotify 增量索引，失败回退轮询）。
	fileWatch *filewatch.Watcher

	// 阶段 5 T5-3a：本地模型保活（keep-warm）。每 5 分钟一轮，对 catalog 中
	// Running 的 herdsman 模型发轻量探针，防止空闲卸载/降温。开关
	// keep_warm_enabled 关闭时整轮跳过；只探不启（local_concurrency=1）。
	keepWarmStop     chan struct{}
	keepWarmOnce     sync.Once
	keepWarmStopOnce sync.Once

	// keepAliveAt 各模型最近一次成功探针时间（内存态，重启即失；供诊断/展示）。
	keepAliveMu sync.RWMutex
	keepAliveAt map[string]string

	// 阶段 5 T5-3b：启动自动预载（auto_preload）。Startup 后延迟 10s，按功能
	// 绑定 gaea→office→chat 预载一个已安装且未运行的 herdsman 模型。
	preloadOnce sync.Once
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

	// intentClassifierFn 意图 LLM 兜底分类 seam（v4.8，测试注入点；nil=内置 LLM 分类器）。
	intentClassifierFn func(text string) *intent.Intent
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
		whisperDataRoot: filepath.Join(config.DataRoot(), "whisper_data"),
		weixinServers:   map[string]*weixin.Server{},
	}
	a.officeState = &officeState{core: c, app: a}
	return a
}
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// 卡死诊断端口（仅本机）：即使 Wails 调用队列死锁，这个独立 HTTP
	// 服务仍可访问，用于抓取 Go 协程栈定位进程级死锁。
	startDebugServer()

	// 数据根迁移：旧版本把 whisper_data 放在 ResourceDir（exe 相对路径），
	// 桌面副本会因找不到 prompts 而分裂数据目录。新版本统一到用户级
	// DataRoot，首次启动时把旧目录内容搬过去（目标已存在则跳过）。
	a.migrateLegacyDataRoot()

	// P4-3 数据可迁移：应用待恢复数据（恢复前先备份当前数据；必须在打开任何数据库/日志前执行）
	a.applyPendingRestore()

	// 将 slog 输出到文件（GUI 应用无控制台）
	logFile, err := os.OpenFile(filepath.Join(a.whisperDataRoot, "gaea.log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err == nil {
		slog.SetDefault(slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{Level: slog.LevelInfo})))
		slog.Info("=== gaea startup ===")
	}
	// 创建 AI client（仅此一次；token 由 GetToken 懒加载）
	a.client = ai.NewClient(a.cfg)

	// 密钥保护：旧版明文一次性迁移为 DPAPI 密文，再解密供内存使用
	encryptSecretIfLegacy(config.KeyDeepseekAPIKey, &a.cfg.DeepseekAPIKey)
	encryptSecretIfLegacy(config.KeyGLMAPIKey, &a.cfg.GLMAPIKey)
	encryptSecretIfLegacy(config.KeyOpencodeGoAPIKey, &a.cfg.OpenCodeGoAPIKey)
	encryptSecretIfLegacy(config.KeyOpencodeZenAPIKey, &a.cfg.OpenCodeZenAPIKey)
	deepseekKey, _ := secure.DecryptString(a.cfg.DeepseekAPIKey)
	glmKey, _ := secure.DecryptString(a.cfg.GLMAPIKey)
	opencodeGoKey, _ := secure.DecryptString(a.cfg.OpenCodeGoAPIKey)
	opencodeZenKey, _ := secure.DecryptString(a.cfg.OpenCodeZenAPIKey)

	// 初始化模型引擎管理器，尝试恢复已保存的 xAI token
	a.engineMgr = modelengine.NewManager("", deepseekKey)
	a.engineMgr.UpdateGLMKey(glmKey)
	a.engineMgr.UpdateOpencodeKey(opencodeGoKey)
	a.engineMgr.UpdateOpencodeZenKey(opencodeZenKey)
	// A 刀自定义引擎：custom_engine_keys 密文解密为明文后注入 Manager（Key 只存
	// 内存；引擎本体由下方 LoadState 从状态文件恢复，Key 与引擎分离存储）。
	a.engineMgr.SetCustomEngineKeys(decryptCustomEngineKeys(a.cfg.CustomEngineKeys))
	if err := a.engineMgr.LoadState(filepath.Join(a.whisperDataRoot, "engines.json")); err != nil {
		slog.Warn("加载引擎状态失败（回退预置默认）", "error", err)
	}
	a.engineMgr.SetStatsPath(modelengine.StatsPathFor(filepath.Join(a.whisperDataRoot, "engines.json")))
	// T6-6.2 汇率配置：把 ~/.gaea_config.json 的 usd_cny_rate 注入统计折算
	// （config.Load 含磁盘 IO，只在启动/用户修改时注入一次，避免逐调用读取）。
	a.engineMgr.SetUsdCnyRate(a.cfg.UsdCnyRate)
	// GLM 目录覆盖文件（模型中心成本层）：启动注入一次，照 SetUsdCnyRate
	// 先例（非绑定方法）；空=只用内嵌目录。
	a.engineMgr.SetGLMCatalogPath(a.cfg.GLMCatalogPath)
	// B 刀 GLM 目录远程热更新：缓存文件 glm_catalog_remote.json 与 engines.json
	// 同目录（加载器按 mtime 感知，生效优先级 覆盖文件 > 远程缓存 > 内嵌）；
	// glm_catalog_url 非空时再启动异步拉取循环（10s 超时 / 24h 周期，ctx 随
	// app 生命周期）。远程目录仅影响展示与费用估算，不影响请求路由/alias/鉴权。
	a.engineMgr.StartGLMCatalogRemote(a.ctx, a.cfg.GLMCatalogURL,
		filepath.Join(a.whisperDataRoot, "glm_catalog_remote.json"))
	// C 刀：引擎健康巡检（首轮延迟 1 分钟 / 周期 10 分钟，只探 enabled 非本地
	// 引擎；ctx 随 app 生命周期，Shutdown 经 StopHealthProbe 停止）。状态相对
	// 上次变化时经回调 emit engine-health-changed（Manager 不直接 emit）。
	a.engineMgr.SetHealthNotifyFunc(func(id string, connected bool) {
		a.emit("engine-health-changed", map[string]interface{}{"id": id, "connected": connected})
	})
	a.engineMgr.StartHealthProbe(a.ctx)
	// 确保 xAI 引擎始终提供内置语音模型 grok-tts（TTS API 不返回在 /v1/models 列表）
	a.engineMgr.EnsureModel("xai", "grok-tts")
	// 确保本地 CosyVoice2 引擎提供语音模型（OpenAI 兼容 TTS 服务）
	a.engineMgr.EnsureModel("cosyvoice", "CosyVoice2-0.5B")
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
	a.activeTTSVoice = a.cfg.TTSVoice
	a.activeOCREngine = a.cfg.ActiveOCREngine
	a.activeOCRModel = a.cfg.ActiveOCRModel
	a.activePersonalityID = a.cfg.VoicePersonality
	a.chatVoiceEngine = a.cfg.FuncChatVoiceEngine
	a.chatVoiceModel = a.cfg.FuncChatVoiceModel

	a.initVoice()
	a.initWeixin()

	// 全局角色库：内置人格 + 助手全部种子化为统一角色
	a.charLib = characterlib.NewStore(filepath.Join(a.whisperDataRoot, "characterlib"))
	if n := a.charLib.MigratePortraitsToFiles(); n > 0 {
		slog.Info("角色库剧照已迁移为文件", "count", n)
	}
	// 远程剧照（xAI 临时图等会过期）本地化，避免角色卡裂图
	if n := a.charLib.MigrateRemotePortraits(); n > 0 {
		slog.Info("角色库远程剧照已本地化", "count", n)
	}
	if a.charLib != nil {
		if err := a.charLib.EnsureBuiltins(whisper.PersonalityPresets); err != nil {
			slog.Error("角色库种子化内置角色失败", "error", err)
		}
		if err := a.charLib.EnsureAssistants(a.assistantMgr.List(), whisper.PersonalityPresets); err != nil {
			slog.Error("角色库同步助手失败", "error", err)
		}
		// 助手记录同步本地化后的剧照路径（聊天人格头像与角色库一致）
		if a.assistantMgr != nil {
			for _, c := range a.charLib.ListChatEnabled() {
				if c.AssistantID == "" || c.PortraitURL == "" {
					continue
				}
				if ast := a.assistantMgr.Get(c.AssistantID); ast != nil && ast.PortraitURL != c.PortraitURL {
					ast.PortraitURL = c.PortraitURL
					if err := a.assistantMgr.Update(ast.ID, *ast); err != nil {
						slog.Warn("同步助手剧照失败", "assistantID", ast.ID, "error", err)
					}
				}
			}
		}
	}

	// 统一聊天会话存储（chat.db）
	a.chatStore = chat.NewStore(filepath.Join(a.whisperDataRoot, "chat"))
	a.initBrain()
	a.initModules()

	// 本地 TTS 服务保活：模型中心内置 CosyVoice2，gaea 启动即自动拉起（幂等，已就绪零开销）
	a.ensureLocalTTSService("cosyvoice")

	// 阶段 5 T5-1：通用任务调度器（价格抓取/文件索引等长任务统一异步化）。
	a.startTaskScheduler()
	// 阶段 5 T5-3a/b：本地模型调度纵深——保活（keep-warm）+ 启动自动预载。
	// 放在任务调度器之后装配（幂等；只探已 Running 模型 / 只预载一个，互不影响）。
	a.startKeepWarm()
	a.startAutoPreload()
	// 价格源定时抓取调度：启动后立即检查一次，之后每 30 分钟按订阅频率轮询。
	a.startPriceCron()
	// v4.3c：主动关心定时推送循环（轻语会客厅，play 空间）——按配置间隔评估
	// 已就绪轻语会话，门控/频控/作息/生日四信号齐全时推前端主动消息事件。
	a.startProactiveTicker()
	// 文件语义索引自动维护：启动后立即增量重建一次，之后每 10 分钟（实时监听
	// 可用时轮询仅作兜底，见 startFileWatch）。
	a.startFileIndexCron()
	// 阶段 5 T5-2：工作区实时文件监听（fsnotify 增量索引，秒级可搜）。
	a.startFileWatch()

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

// debugServerPort 诊断端口（仅本机回环）。
const debugServerPort = "127.0.0.1:18123"

// startDebugServer 启动独立诊断 HTTP 服务：
//
//	GET /healthz → "ok"（进程存活探针）
//	GET /stack   → 全部 goroutine 栈（死锁定位）
//
// 独立 goroutine + 独立端口，不经过 Wails IPC，卡死时仍可响应。
func startDebugServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("/stack", func(w http.ResponseWriter, _ *http.Request) {
		buf := make([]byte, 1<<20)
		n := goruntime.Stack(buf, true)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(buf[:n])
	})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("debug server panic recovered", "panic", r)
			}
		}()
		srv := &http.Server{Addr: debugServerPort, Handler: mux}
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Warn("诊断端口启动失败（不影响主程序）", "error", err)
		}
	}()
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
	case "glm":
		eng, ok := a.engineMgr.GetEngine("glm")
		key := a.engineMgr.GLMKey()
		if ok && eng.Enabled && key != "" {
			backend := ai.NewGLMImageBackend(eng.BaseURL, key)
			a.client.SetImageBackend(backend, "glm")
			slog.Info("图片后端: GLM", "url", eng.BaseURL)
		} else {
			a.client.SetImageBackend(nil, "xai")
			slog.Warn("图片后端: GLM 不可用（引擎未启用或 Key 未配置），回退 xAI")
		}
	default: // "xai" 或空
		a.client.SetImageBackend(nil, "xai")
		slog.Info("图片后端: xAI")
	}
}

// Shutdown Wails 关闭回调
func (a *App) Shutdown(ctx context.Context) {
	// 停止 GLM 目录远程拉取循环（B 刀；engineMgr 未初始化时为空操作兜底）。
	if a.engineMgr != nil {
		a.engineMgr.StopGLMCatalogRemote()
		// 停止引擎健康巡检循环（C 刀 v0）。
		a.engineMgr.StopHealthProbe()
	}
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
	// T7-1.1 ③：末轮落库——先 drain 异步记忆写入队列再持久化所有轻语会话，
	// 避免进程退出时末轮 LLM 抽取的记忆/状态丢失（H4）。须在关闭 DB 前执行。
	if a.whisperState != nil {
		a.whisperState.drainAndPersistAll()
		// v4.3c：停止主动关心定时推送循环。
		a.whisperState.proactiveStopOnce.Do(func() {
			if a.whisperState.proactiveStop != nil {
				close(a.whisperState.proactiveStop)
			}
		})
		// v4.4：停止微信提醒到点回推循环（reminderStop 由 Startup 同步创建，
		// 与 proactiveStop 同模式，此处无并发写）。
		if a.whisperState.reminderStop != nil {
			close(a.whisperState.reminderStop)
		}
	}
	if err := a.closePM(); err != nil {
		slog.Error("关闭项目失败", "error", err)
	}
	if err := chat.CloseDatabase(filepath.Join(a.whisperDataRoot, "chat")); err != nil {
		slog.Error("关闭 chat.db 失败", "error", err)
	}
	if err := characterlib.CloseDatabase(filepath.Join(a.whisperDataRoot, "characterlib")); err != nil {
		slog.Error("关闭 characterlib.db 失败", "error", err)
	}
	// 停止价格源定时抓取调度。
	a.officeState.priceStopOnce.Do(func() {
		if a.officeState.priceCronStop != nil {
			close(a.officeState.priceCronStop)
		}
	})
	a.officeState.fileIndexStopOnce.Do(func() {
		if a.officeState.fileIndexStop != nil {
			close(a.officeState.fileIndexStop)
		}
	})
	// 停止本地模型保活轮询（T5-3a）。
	a.officeState.keepWarmStopOnce.Do(func() {
		if a.officeState.keepWarmStop != nil {
			close(a.officeState.keepWarmStop)
		}
	})
	// 停止任务调度器（running 任务留待下次启动续跑）与文件监听。
	if a.officeState.tasks != nil {
		a.officeState.tasks.Close()
	}
	if w := a.officeState.fileWatch; w != nil {
		_ = w.Close()
		a.officeState.fileWatch = nil
	}
}

// getPM/setPM/closePM/initAgents/restoreImageBackend 已迁移到
// writing_state.go（writingState 域）。App 经嵌入提升，Startup 调用不变。
// initWeixin/startAssistantWx/stopAssistantWx 已迁移到 whisper_state.go。

// SetDistFS 设置前端静态资源 embed.FS（由 main.go 在启动前调用）
func (a *App) SetDistFS(fsys fs.FS) {
	a.distFS = fsys
}

// SetPromptFS 注入内置 prompt 模板（由 main.go 的 go:embed 在启动前调用）。
// 磁盘 prompts/ 仍优先（开发期可直接改模板生效），exe 单独分发（旁边没有
// prompts/ 目录，如桌面快捷副本）时由内置模板兜底，避免“缺少 xxx 模板文件”。
func (a *App) SetPromptFS(fsys fs.FS) {
	if a.writingState == nil {
		return
	}
	a.writingState.eng = prompt.NewEngineWithEmbedded(
		filepath.Join(a.cfg.ResourceDir, "prompts"), fsys)
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

// encryptSecretIfLegacy 将旧版明文密钥就地加密并落盘（一次性迁移）。
// 已带 "dpapi:" 前缀（密文）或为空时跳过，避免每次启动重复写配置。
func encryptSecretIfLegacy(key string, v *string) {
	if *v == "" || strings.HasPrefix(*v, "dpapi:") {
		return
	}
	enc, err := secure.EncryptString(*v)
	if err != nil || enc == *v {
		return
	}
	if err := config.Save(key, enc); err != nil {
		slog.Warn("迁移密钥为密文失败", "key", key, "error", err)
		return
	}
	*v = enc
}

// migrateLegacyDataRoot 把旧版 ResourceDir/whisper_data 中的数据迁移到
// 用户级 DataRoot/whisper_data（仅当目标不存在或为空时执行，幂等）。
func (a *App) migrateLegacyDataRoot() {
	newRoot := a.whisperDataRoot
	if newRoot == "" {
		return
	}
	// 目标已存在且有内容：不迁移（可能是新版本已在写入，避免覆盖）。
	if entries, err := os.ReadDir(newRoot); err == nil && len(entries) > 0 {
		return
	}
	legacy := filepath.Join(a.cfg.ResourceDir, "whisper_data")
	if legacy == newRoot || legacy == "" {
		return
	}
	if _, err := os.Stat(legacy); err != nil {
		return // 旧目录不存在，无需迁移
	}
	if err := os.MkdirAll(newRoot, 0o755); err != nil {
		slog.Warn("迁移旧数据目录失败：无法创建目标", "path", newRoot, "error", err)
		return
	}
	entries, err := os.ReadDir(legacy)
	if err != nil {
		slog.Warn("迁移旧数据目录失败：无法读取旧目录", "path", legacy, "error", err)
		return
	}
	moved := 0
	for _, e := range entries {
		src := filepath.Join(legacy, e.Name())
		dst := filepath.Join(newRoot, e.Name())
		if err := copyPath(src, dst); err != nil {
			slog.Warn("迁移旧数据失败", "src", src, "error", err)
			continue
		}
		moved++
	}
	if moved > 0 {
		slog.Info("旧数据目录已迁移", "from", legacy, "to", newRoot, "items", moved)
	}
}

// copyPath 复制文件或整个目录（用于数据根迁移）。
func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, 0o755); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyPath(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, info.Mode().Perm())
}
