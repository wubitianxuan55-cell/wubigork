package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/BurntSushi/toml"

	appconfig "github.com/gaea/gaea/internal/config"
	gaeaBoot "github.com/gaea/gaea/internal/gaea/boot"
	gaeaAgent "github.com/gaea/gaea/internal/gaea/agent"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
	"github.com/gaea/gaea/internal/gaea/spaces"
	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/modelengine"
)

// ── gaea 工程办公板块（移植自 gaeaW）─────────────────────────────
// 独立板块：47 个工程工具 + 6 个技能 + 单模型 agent（规划+执行一体）。
// 模型走 gaea 模型中心（bridge provider），前端 UI + AI 双通道调用。

var ga = &gaeaRuntime{wire: newGaeaEventForwarder()}

type gaeaRuntime struct {
	mu   sync.Mutex
	ctrl *control.Controller
	// cfg 是当前生效的办公引擎配置。设置面板的写操作（Agent 参数/权限/沙箱）
	// 直接修改它并持久化到用户配置，随后重建 controller 使变更生效。
	cfg *gaeaConfig.Config
	// followUp 是子代理追问执行器（v4.64 Side Chat 式追问）：boot 用 taskTool
	// 的 continue_from 管道组装，经 OnFollowUpReady 交给这里；随 controller
	// 重建而更新（任务树/tab 状态经轮询自校正，无在途状态需要迁移）。
	followUp   gaeaAgent.SubagentFollowUpRunner
	followUpMu sync.Mutex
	// wire 是 gaea 事件流 → 前端 gaea-event 的转发层（v4.26 对话流式重造）：
	// wire seq 打点 + phase 节流的唯一状态点，随进程存活（不随 controller
	// 重建重置——设置变更重建引擎不打断当前会话，seq 回退会破坏前端断号检测）。
	wire *gaeaEventForwarder
}

// gaeaLoadConfig 加载办公引擎配置：内置默认 + 用户持久化文件（若有），
// 再注入 bridge provider（kind=gaea 走 gaea 模型中心）。返回可直接修改的配置。
func gaeaLoadConfig() (*gaeaConfig.Config, error) {
	cfg := gaeaConfig.Default()
	if p := gaeaConfig.UserConfigPath(); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			if _, err := toml.Decode(string(b), cfg); err != nil {
				return nil, fmt.Errorf("gaea: 解析持久化配置 %s: %w", p, err)
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("gaea: 读取持久化配置 %s: %w", p, err)
		}
	}
	cfg.DefaultModel = "gaea"
	cfg.Providers = []gaeaConfig.ProviderEntry{{
		Name:  "gaea",
		Kind:  "wubigrok", // 内部 provider 注册名（bridge provider）
		Model: "",
		// 上下文窗口按实际绑定模型的能力取保守值（此前写 1M 导致自动压缩
		// 阈值高达 80 万 token，办公会话膨胀到十几万也从不压缩，模型首字
		// 极慢、看起来像“没流式输出”）。256k 下 80% 阈值≈204k，超限自动压缩。
		ContextWindow: 256_000,
	}}
	// 全部工程工具注册（Enabled 为空 = 全部）。S1.3-B：装配期按当前空间过滤
	// ——boot.Build 读同一份配置的 EffectiveSessionSpace，在 addBuiltins/
	// ExtraTools/MCP spec 层物理过滤（work=办公/编辑/检索域，play=生图/轻语域，
	// shared 通用）；GaeaSpaceActivate 写 session.space 后下次引擎重建/重启
	// 生效，与既有会话目录分区语义一致（运行中引擎不受影响）。
	cfg.Tools.Enabled = nil
	// 关闭写文件/网络类工具的沙箱限制，避免办公工具被无谓拦截
	cfg.Sandbox.Bash = "off"
	// 写入根跟随工作空间：文件写入工具（write_file/edit_file）限制在
	// 当前工作空间内，而不是进程启动目录（WriteRoots 空时回退 os.Getwd）。
	if cfg.Workspace != "" {
		cfg.Sandbox.WorkspaceRoot = cfg.Workspace
	}
	return cfg, nil
}

// gaeaBuildController 用当前配置构建 controller（不持有 ga.mu，调用方负责）。
func (a *App) gaeaBuildController() (*control.Controller, error) {
	// 事件转发：gaea 事件流 → 前端 gaea-event 回调。经转发层统一打点 wire
	// seq 并对 phase 节流（v4.26 对话流式重造，见 gaeaEventForwarder）。
	sink := event.FuncSink(func(e event.Event) {
		a.emitGaeaEvent(e)
		// 自动做梦：轮次成功后后台整理记忆（单飞、有实质内容才跑）。
		// S1.2 A dream 空间化：TurnDone 事件槽不带上下文——此处取当前办公
		// 会话的空间（ctrl.SessionSpace()，缺省回退配置生效空间；mode=off 为
		// ""=无空间维度）一并传给做梦管线，dream 写入/指纹/notes 全链路按
		// 会话空间分流。
		if e.Kind == event.TurnDone && e.Err == nil {
			a.maybeDreamAfterTurn(gaeaSessionSpace())
		}
	})
	// 构建 controller（单模型 agent）
	//    SessionDir 必须指向工作区会话目录（cwd/.gaea/sessions[/<space>]），
	//    与 GaeaListSessions/GaeaResumeSession 的读取路径一致，否则历史面板
	//    永远看不到当前会话（会落到用户级 AppData/Roaming/gaea/sessions）。
	//    S2：目录按配置空间分区（space.mode=off 回平铺 ""），读取端对两个
	//    空间目录 + 平铺兜底各列一次。
	space := ""
	if ga.cfg != nil {
		space = ga.cfg.EffectiveSessionSpace()
	}
	ctrl, err := gaeaBoot.Build(a.ctx, gaeaBoot.Options{
		Model:      "gaea",
		// v4.64 Side Chat 式追问：文本增量走专用通道（gaea-subagent-text），
		// 追问执行器交给 ga 保存（GaeaSubagentFollowUp 绑定调用）。
		EmitSubagentText: func(ref, text string) {
			a.emit("gaea-subagent-text", map[string]interface{}{
				"kind": "subagent_text", "text": text, "subagentRef": ref,
			})
		},
		OnFollowUpReady: func(runner gaeaAgent.SubagentFollowUpRunner) {
			ga.followUpMu.Lock()
			ga.followUp = runner
			ga.followUpMu.Unlock()
		},
		RequireKey: false,
		Sink:       sink,
		MaxSteps:   0,
		SessionDir: gaeaConfig.WorkspaceSessionDir(gaeaCwd(), space),
		// 工作空间根：基础工具（read/write/bash/ls）相对路径基于它，
		// 而非进程 cwd（否则办公 agent 在启动目录而非用户工作空间操作）。
		Cwd: gaeaCwd(),
		// 注入需要应用服务的工具（生图/画图复用模型中心模型与图片后端）。
		// 3.0 Step 3d #6：专业工具（ocr 等）经 gaeaSpecialistTools 集中注册，
		// 展开进 ExtraTools，避免装配列表与定义漂移。
		ExtraTools: append([]tool.Tool{
			imageGenTool{a: a},
			diagramTool{a: a},
			routineLLMTool{a: a},
			translateTextTool{a: a},
			factAddTool{},
			factListTool{},
			factClearTool{},
		}, gaeaSpecialistTools(a)...),
		// 晨报预载（v4.16 刀④）：work 空间会话装配时把高频工作记忆预装配进
		// agent 上下文（零 LLM、预算受限、work 只读）。开关读 ~/.gaea_config.json
		// 的 morning_preload 键（默认开，仅 config 文件可控，无 UI 绑定）；
		// a.cfg 未就绪（测试/启动早期）时缺省开启，与配置默认值一致。
		MorningPreload: morningPreloadEnabled(a),
	})
	if err != nil {
		return nil, fmt.Errorf("gaea: 引擎初始化失败: %w", err)
	}
	// 3.0 Step 1 回退开关（session.log_format）：把配置的会话持久化格式注入
	// 控制器——event 模式下 Snapshot 双写、回合前落用户消息 + flush 检查点
	// （fail-closed）、Resume 走 Restore（checkpoint+tail）；缺省 event
	// （轨迹/上下文看板的数据源），显式 "legacy" 退回旧行为。
	// S2 双空间：同点注入空间配置生效值（""=space.mode=off，仿 logFormat 三件套）。
	if ga.cfg != nil {
		ctrl.SetLogFormat(ga.cfg.EffectiveLogFormat())
		ctrl.SetSpace(ga.cfg.EffectiveSessionSpace())
	}
	// 启用交互式审批：工具调用放行/拒绝、ask 结构化提问经前端确认，
	// 否则全部工具（含写文件/网络）自动放行且审批弹窗永不出现。
	ctrl.EnableInteractiveApproval()
	return ctrl, nil
}

// morningPreloadEnabled 返回晨报预载开关（v4.16 刀④）：读 App 配置的
// morning_preload 键（默认开）；a/a.cfg 未就绪（测试/启动早期）时缺省开启，
// 与配置键默认值一致。
func morningPreloadEnabled(a *App) bool {
	if a == nil || a.cfg == nil {
		return true
	}
	return a.cfg.GetMorningPreload()
}

// GaeaMorningPreload 返回晨报预载开关（~/.gaea_config.json 的 morning_preload
// 键，默认开）：work 空间新会话装配时把高频工作记忆确定性预装配进上下文。
func (a *App) GaeaMorningPreload() bool {
	if a == nil || a.cfg == nil {
		return true
	}
	return a.cfg.GetMorningPreload()
}

// GaeaSetMorningPreload 持久化晨报预载开关并重建办公引擎（新会话装配即时
// 生效）：写 ~/.gaea_config.json + 更新内存 + gaeaRebuildLocked。
func (a *App) GaeaSetMorningPreload(enabled bool) error {
	val := "0"
	if enabled {
		val = "1"
	}
	if err := appconfig.Save(appconfig.KeyMorningPreload, val); err != nil {
		return err
	}
	if a.cfg != nil {
		a.cfg.SetMorningPreload(enabled)
	}
	return a.gaeaRebuildLocked()
}

// gaeaRebuildLocked 用当前配置重建 controller（设置变更后生效），替换旧实例。
// 调用方必须已持有 ga.mu。
func (a *App) gaeaRebuildLocked() error {
	newCtrl, err := a.gaeaBuildController()
	if err != nil {
		return err
	}
	old := ga.ctrl
	ga.ctrl = newCtrl
	if old != nil {
		old.Close()
	}
	return nil
}

// GaeaInit 初始化办公引擎（幂等）。用 gaea 模型中心的默认引擎驱动。
func (a *App) GaeaInit() error {
	ga.mu.Lock()
	defer ga.mu.Unlock()
	if ga.ctrl != nil {
		return nil
	}

	// 1. 注入模型中心客户端（bridge provider 的底层）
	bridge.SetClient(a.client)
	// 注入办公功能级模型绑定（func_gaea_engine/model）：办公 agent 走指定
	// 引擎，而非全局活跃引擎——避免活跃引擎为 xai 时把其他模型名发到 xAI
	// 导致 404。未绑定则跟随全局活跃引擎。
	if eng, model := a.cfg.GetFeatureModel("gaea"); eng != "" {
		bridge.SetFeature(eng, model)
		slog.Info("办公模型绑定已注入", "engine", eng, "model", model)
	}

	// 2. 注入配置：持久化文件（用户设置）+ bridge provider
	cfg, err := gaeaLoadConfig()
	if err != nil {
		return err
	}
	ga.cfg = cfg
	// 把启动时的工作区记入最近列表（幂等），确保「项目」分组在首次启动
	// 也包含当前项目，而不仅仅在用户显式切换工作区之后才出现。
	gaeaConfig.TouchRecentWorkspace(gaeaCwd())
	// 任务模板库：把内置模板落盘为 .gaea/commands/*.md（幂等，不覆盖用户文件），
	// 使 / 菜单与 Submit 通过既有自定义命令管线直接解析模板。
	if err := ensureTaskTemplateCommands(gaeaCwd()); err != nil {
		slog.Warn("任务模板安装失败（不影响引擎启动）", "error", err)
	}
	// 文件落盘规范配套：过程/中间文件统一目录 .gaea/work/（脚本、OCR 页图、
	// 中间文本等），交付物 .gaea/exports/，避免与源文件混在工作空间根目录。
	// S4 双空间：work 侧现状目录恒建（兼容红线，不挪目录）；当前生效空间为
	// play 时追加创建 .gaea/play/{work,exports} 分区目录（space.mode=off 时
	// 生效空间为空，行为与改造前一致）。注意：GaeaInit 持有 ga.mu，空间取
	// 本地 cfg 计算而非 gaeaEffectiveSpace()（其内部加锁会死锁）。
	spaceDirs := []string{
		spaces.WorkDir(gaeaCwd(), spaces.SpaceWork),
		spaces.ExportsDir(gaeaCwd(), spaces.SpaceWork),
	}
	if cfg.EffectiveSessionSpace() == spaces.SpacePlay {
		spaceDirs = append(spaceDirs,
			spaces.WorkDir(gaeaCwd(), spaces.SpacePlay),
			spaces.ExportsDir(gaeaCwd(), spaces.SpacePlay))
	}
	for _, dir := range spaceDirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			slog.Warn("创建工作区 .gaea 子目录失败", "dir", dir, "error", err)
		}
	}
	// loader 无锁读 ga.cfg：ga.cfg 指针的替换在持锁下进行，读取方只会拿到
	// 一个完整可用的配置（旧指针在替换后不再被修改），不会与重建死锁。
	gaeaConfig.SetLoader(func() (*gaeaConfig.Config, error) {
		if ga.cfg == nil {
			return gaeaConfig.Default(), nil
		}
		return ga.cfg, nil
	})

	// 3. 构建 controller
	ctrl, err := a.gaeaBuildController()
	if err != nil {
		return err
	}
	ga.ctrl = ctrl
	// 重启后自动恢复最近一次会话（仅首次初始化；设置变更重建时不干预当前会话），
	// 避免用户重启桌面端后看到"上下文全部清零"的空对话。
	a.resumeLastSession(ctrl)
	// 通知前端办公引擎就绪（对应 gaea/lib/bridge.ts 的 onReady 监听 gaea-ready）
	a.emit("gaea-ready", map[string]interface{}{"kind": "ready"})
	return nil
}

// SetFeatureModel 是 core.SetFeatureModel 的 App 层覆盖：办公主 agent 的模型
// 由 bridge provider 在 GaeaInit 时注入并缓存，运行时改绑必须重新注入并重建
// controller，否则仍沿用旧模型（例如先前注入 deepseek，改绑 grok 后办公仍走 deepseek）。
func (a *App) SetFeatureModel(feature, engineID, modelName string) error {
	if err := a.core.SetFeatureModel(feature, engineID, modelName); err != nil {
		return err
	}
	if feature == "gaea" {
		a.applyOfficeFeatureModel(engineID, modelName)
	}
	return nil
}

// SetFeatureModelEnabled 同理：办公功能级启停需即时反映到 bridge provider；
// 停用清空注入回退全局，启用恢复功能绑定。
func (a *App) SetFeatureModelEnabled(feature string, enabled bool) error {
	if err := a.core.SetFeatureModelEnabled(feature, enabled); err != nil {
		return err
	}
	if feature == "gaea" {
		if enabled {
			eng, model := a.cfg.GetFeatureModel("gaea")
			a.applyOfficeFeatureModel(eng, model)
		} else {
			a.applyOfficeFeatureModel("", "")
		}
	}
	return nil
}

// applyOfficeFeatureModel 注入办公 bridge 的 feature 模型，并在办公引擎已初始化时重建
// controller。办公尚未打开（ga.ctrl == nil）时仅更新注入，后续 GaeaInit 会再按最新配置注入。
func (a *App) applyOfficeFeatureModel(engine, model string) {
	bridge.SetFeature(engine, model)
	ga.mu.Lock()
	defer ga.mu.Unlock()
	if ga.ctrl != nil && ga.cfg != nil {
		if err := a.gaeaRebuildLocked(); err != nil {
			slog.Warn("办公模型绑定变更后重建引擎失败", "error", err)
		}
	}
}

// GaeaReloadResult 热加载结果摘要：重建后生效的工具/技能数量，前端可据此提示。
type GaeaReloadResult struct {
	Tools  int `json:"tools"`
	Skills int `json:"skills"`
}

// GaeaReload 热加载办公引擎：重新读取磁盘上的持久化配置（Agent 参数/权限/
// 沙箱/技能路径/插件），并重建 controller 使变更立即生效——新增或修改的
// 技能/工具/插件无需重启桌面端即可被引擎感知。未初始化时先完成初始化。
// 成功后广播 gaea-ready（kind=reloaded），前端 store 会重新拉取数据。
// 失败时保持旧引擎继续运行，不替换任何状态。
func (a *App) GaeaReload() (GaeaReloadResult, error) {
	ga.mu.Lock()
	initialized := ga.ctrl != nil
	ga.mu.Unlock()
	if !initialized {
		if err := a.GaeaInit(); err != nil {
			return GaeaReloadResult{}, err
		}
	}

	ga.mu.Lock()
	defer ga.mu.Unlock()

	cfg, err := gaeaLoadConfig()
	if err != nil {
		return GaeaReloadResult{}, fmt.Errorf("gaea: 热加载配置失败: %w", err)
	}
	// 替换生效配置。gaeaConfig.SetLoader 的闭包实时读取 ga.cfg，boot.Build
	// 全程只读该指针，持锁替换后旧指针不再被修改，不会与重建死锁。
	ga.cfg = cfg
	if err := a.gaeaRebuildLocked(); err != nil {
		return GaeaReloadResult{}, err
	}

	res := GaeaReloadResult{
		Tools:  len(ga.ctrl.Tools()),
		Skills: len(ga.ctrl.Skills()),
	}
	a.emit("gaea-ready", map[string]interface{}{"kind": "reloaded"})
	return res, nil
}

// GaeaSend 提交对话（异步，事件经 gaea-event 回调）。未初始化时自动初始化。
func (a *App) GaeaSend(input string) {
	ga.mu.Lock()
	needBoot := ga.ctrl == nil
	ga.mu.Unlock()
	if needBoot {
		// v4.26 对话流式重造：引擎未初始化时 GaeaInit（模型中心注入/工具装配/
		// 会话自动恢复）可持续数秒，先发 phase「正在启动引擎」让首条消息发出
		// 后立即有反馈。此时控制器尚不存在、会话日志未建立，该事件仅走 wire
		// （不落盘），是唯一不带磁盘记录的 phase 来源。
		a.emitGaeaEvent(event.Event{Kind: event.Phase, Text: "正在启动引擎"})
	}
	if err := a.GaeaInit(); err != nil {
		a.emit("gaea-event", map[string]interface{}{"kind": "error", "text": err.Error()})
		return
	}
	ga.mu.Lock()
	ctrl := ga.ctrl
	ga.mu.Unlock()
	if ctrl != nil {
		ctrl.Send(input)
	}
}

// GaeaSteer 任务运行中插话调整（2026-08-28，对齐豆包工作「边跑边改」）：
// 把消息作为当前回合的补充指引注入 agent（不打断工具执行、不开新回合）；
// 未运行（无回合可插话）时走 GaeaSend 排队兜底。事件经 gaea-event 回调，
// 以 notice 轻量回显。
func (a *App) GaeaSteer(input string) {
	if err := a.GaeaInit(); err != nil {
		a.emit("gaea-event", map[string]interface{}{"kind": "error", "text": err.Error()})
		return
	}
	ga.mu.Lock()
	ctrl := ga.ctrl
	ga.mu.Unlock()
	if ctrl != nil {
		ctrl.Steer(input)
	}
}

// GaeaCancel 取消当前回合。
func (a *App) GaeaCancel() {
	ga.mu.Lock()
	defer ga.mu.Unlock()
	if ga.ctrl != nil {
		ga.ctrl.Cancel()
	}
}

// GaeaRunning 报告引擎是否正在运行。
func (a *App) GaeaRunning() bool {
	ga.mu.Lock()
	defer ga.mu.Unlock()
	return ga.ctrl != nil && ga.ctrl.Running()
}

// GaeaNewSession 开启新会话（清空上下文，保留记忆/技能）。
func (a *App) GaeaNewSession() error {
	ga.mu.Lock()
	defer ga.mu.Unlock()
	if ga.ctrl == nil {
		return nil
	}
	// 新会话 = 全新目标：清空 goal gate，避免上个会话的「持续工作到验收」
	// 目标残留到新会话（手动 /goal 在新会话同样需要重设）。
	ga.ctrl.SetGoal("")
	if err := ga.ctrl.NewSession(); err != nil {
		return err
	}
	// v4.26：会话切换归零 wire seq（per-会话语义）。前端在本动作后会重载
	// 历史 items 并重置 lastSeq，断号检测从新会话重新计数。
	ga.wire.reset()
	return nil
}

// GaeaModel 实时返回模型中心当前活跃的引擎与模型（engine/model 格式）。
func (a *App) GaeaModel() string {
	engine := a.GetActiveEngine()
	model := a.GetActiveModel()
	if model == "" {
		return engine
	}
	return engine + "/" + model
}

// GaeaEngines 返回模型中心全部引擎（办公板块切换用）。
func (a *App) GaeaEngines() []modelengine.EngineConfig {
	if a.engineMgr == nil {
		return nil
	}
	return a.engineMgr.GetEngines()
}

// GaeaSetEngine 切换办公板块使用的模型中心引擎（与全局活跃引擎联动）。
func (a *App) GaeaSetEngine(engineID string) error {
	return a.SetActiveEngine(engineID)
}

// GaeaTools 列出全部内置工程工具（UI 面板用）。
func (a *App) GaeaTools() []map[string]interface{} {
	out := make([]map[string]interface{}, 0)
	for _, t := range tool.Builtins() {
		out = append(out, map[string]interface{}{
			"name":        t.Name(),
			"description": t.Description(),
			"schema":      string(t.Schema()),
		})
	}
	return out
}

// GaeaSkills 列出工程技能模块。
func (a *App) GaeaSkills() []map[string]interface{} {
	ga.mu.Lock()
	defer ga.mu.Unlock()
	out := make([]map[string]interface{}, 0)
	if ga.ctrl == nil {
		return out
	}
	for _, s := range ga.ctrl.Skills() {
		out = append(out, map[string]interface{}{
			"name":        s.Name,
			"description": s.Description,
		})
	}
	return out
}

// GaeaCallTool 前端直接调用工具（UI 双通道）：name 工具名，argsJSON 参数 JSON。
func (a *App) GaeaCallTool(name, argsJSON string) (string, error) {
	t, ok := tool.LookupBuiltin(name)
	if !ok {
		return "", fmt.Errorf("gaea: 未知工具 %q", name)
	}
	return t.Execute(a.ctx, json.RawMessage(argsJSON))
}

// ── 事件转换 ────────────────────────────────────────────────────

// gaeaPhaseThrottleWindow 是 phase 事件的同阶段节流窗口（v4.26）：同一文案
// 200ms 内不重发——预处理/重试阶段的 phase 由多个发射点产生（控制器、agent
// 循环、Retrying 转译），窗口吸收同刻重复，又不至于吞掉真实的阶段推进。
const gaeaPhaseThrottleWindow = 200 * time.Millisecond

// gaeaEventForwarder 是 gaea 事件流 → 前端 gaea-event 的转发层（v4.26 对话
// 流式重造）。Why：Wails 事件流在密集到达时会丢件（前端 store 注释认账），
// 丢件不可在传输层根治，退而求其次给每条 payload 打单调 seq，让前端能检测
// 断号并调用 GaeaResyncEvents 用磁盘日志整体重建。本层职责：
//  1. wire seq 打点：per 会话单调递增（会话切换时 reset，见 GaeaNewSession /
//     GaeaResumeSession / GaeaFork），只在此处消费，不改磁盘日志格式（日志
//     在 sink 链上游 EventLogSink 已落盘，与本层 seq 无关）；
//  2. phase 节流：同一阶段（同文案）200ms 内不重发（只影响 wire；磁盘日志
//     保留全量，轨迹/恢复不受影响）；
//  3. 事件转译见 gaeaEventMap（Retrying/compaction → phase）。
//
// 不变量（v4.62.1 钉死）：凡经本层上 gaea-event 的事件必须是「已入账本」
// 的事件——seq 与磁盘日志 1:1 对应，缺口才可经 GaeaResyncEvents 补拉。
// wire-only 类事件（SubagentText）一律走专用通道（emitGaeaEvent 分道），
// 严禁进本层消费 seq。
type gaeaEventForwarder struct {
	seq atomic.Int64
	mu  sync.Mutex
	// phaseLast 记录各 phase 文案最近一次转发时刻（节流用，键=phase 文案）。
	phaseLast map[string]time.Time
}

func newGaeaEventForwarder() *gaeaEventForwarder {
	return &gaeaEventForwarder{phaseLast: map[string]time.Time{}}
}

// last 返回当前最新 wire seq（GaeaResyncEvents 的返回 seq）。
func (f *gaeaEventForwarder) last() int64 { return f.seq.Load() }

// next 领取下一个 wire seq（转发 payload 打点用）。
func (f *gaeaEventForwarder) next() int64 { return f.seq.Add(1) }

// reset 会话切换时归零 wire seq。
func (f *gaeaEventForwarder) reset() { f.seq.Store(0) }

// payload 把事件转译为前端 payload 并打点 seq；phase 命中节流时返回 nil
// （调用方跳过转发，seq 不消费——保证转发出去的 payload seq 无断号）。
func (f *gaeaEventForwarder) payload(e event.Event) map[string]interface{} {
	m := gaeaEventMap(e)
	if k, _ := m["kind"].(string); k == "phase" {
		text, _ := m["text"].(string)
		now := time.Now()
		f.mu.Lock()
		if last, ok := f.phaseLast[text]; ok && now.Sub(last) < gaeaPhaseThrottleWindow {
			f.mu.Unlock()
			return nil
		}
		f.phaseLast[text] = now
		f.mu.Unlock()
	}
	m["seq"] = f.next()
	return m
}

// emitGaeaEvent 把一条 gaea 事件经转发层发到前端（seq 打点 + phase 节流）。
// 事件此时已过 EventLogSink（「模型可见必入日志」在上游完成），本层不再落盘。
func (a *App) emitGaeaEvent(e event.Event) {
	// 子代理流式增量走专用通道（v4.62.1 回归修复）：gaea-event 的契约是
	// 「每条 payload 带会话内单调 seq，且与磁盘日志 1:1 对应，丢件可经
	// GaeaResyncEvents 从账本补拉」（v4.26 防线）。SubagentText 是装饰性
	// 实时流（wire-only、有意不落盘），走 gaea-event 会消费 seq 却永远无法
	// 补拉——transport 密集流丢一件即产生不可愈合缺口，防线反复整体重建
	// 对话视图（v4.62.0 回归：子代理运行中对话窗过程可见性被打断）。专用
	// 通道无 seq、有损无妨，由 SubagentThread 的快照 reconcile 兜底。
	if m := gaeaSubagentTextPayload(e); m != nil {
		a.emit(gaeaSubagentTextChannel, m)
		return
	}
	if m := ga.wire.payload(e); m != nil {
		a.emit("gaea-event", m)
	}
}

// gaeaSubagentTextChannel 是子代理流式增量的专用 wails 事件名（见
// emitGaeaEvent 的分道说明）。
const gaeaSubagentTextChannel = "gaea-subagent-text"

// gaeaSubagentTextPayload 把子代理流式增量映射为专用通道 payload；非该类
// 事件返回 nil（走 gaea-event 主通道）。独立成纯函数钉死路由回归：
// SubagentText 不得进入 gaea-event 的 seq 序列（gaeaSubagentTextPayload 命中
// 即 return，wire.payload 永远见不到它）。
func gaeaSubagentTextPayload(e event.Event) map[string]interface{} {
	if e.Kind != event.SubagentText {
		return nil
	}
	m := map[string]interface{}{"kind": "subagent_text", "text": e.Text}
	if e.SubagentRef != "" {
		m["subagentRef"] = e.SubagentRef
	}
	if e.ParentToolID != "" {
		m["parentId"] = e.ParentToolID
	}
	return m
}

// gaeaEventMap 把 gaea 事件流转换为 gaeaW WireEvent 兼容格式（前端 store 直接消费）。
func gaeaEventMap(e event.Event) map[string]interface{} {
	m := map[string]interface{}{"kind": gaeaKindName(e.Kind)}
	switch e.Kind {
	case event.Text, event.Reasoning:
		if e.Text != "" {
			m["text"] = e.Text
		}
		if e.Reasoning != "" {
			m["reasoning"] = e.Reasoning
		}
	case event.Message:
		m["text"] = e.Text
		if e.Reasoning != "" {
			m["reasoning"] = e.Reasoning
		}
	case event.ToolDispatch:
		m["tool"] = map[string]interface{}{
			"id": e.Tool.ID, "name": e.Tool.Name, "args": e.Tool.Args,
			"readOnly": e.Tool.ReadOnly, "partial": e.Tool.Partial,
			"parentId": e.Tool.ParentID,
		}
	case event.ToolResult:
		t := map[string]interface{}{
			"id": e.Tool.ID, "name": e.Tool.Name, "output": e.Tool.Output,
			"recoverable": e.Tool.Recoverable, "truncated": e.Tool.Truncated,
		}
		if e.Tool.Err != "" {
			t["err"] = e.Tool.Err
		}
		m["tool"] = t
	case event.Notice:
		m["text"] = e.Text
		m["level"] = gaeaLevelName(e.Level)
	case event.Phase:
		m["text"] = e.Text
	case event.TurnDone:
		if e.Err != nil {
			m["err"] = e.Err.Error()
		}
	case event.Usage:
		if e.Usage != nil {
			m["usage"] = map[string]interface{}{
				"promptTokens":           e.Usage.PromptTokens,
				"completionTokens":       e.Usage.CompletionTokens,
				"totalTokens":            e.Usage.TotalTokens,
				"cacheHitTokens":         e.Usage.CacheHitTokens,
				"cacheMissTokens":        e.Usage.CacheMissTokens,
				"reasoningTokens":        e.Usage.ReasoningTokens,
				"sessionCacheHitTokens":  e.SessionHit,
				"sessionCacheMissTokens": e.SessionMiss,
				"turn":                   e.Turn,
				"source":                 e.UsageSource,
			}
		}
	case event.ApprovalRequest:
		am := map[string]interface{}{"id": e.Approval.ID, "tool": e.Approval.Tool, "subject": e.Approval.Subject}
		// request_permission 权限升级申请：透传 Request 标记与 reason 原文
		// （前端审批卡据此换标题并必须展示理由）。字段缺省不下发，普通
		// 工具审批的线格式保持不变。
		if e.Approval.Reason != "" {
			am["reason"] = e.Approval.Reason
		}
		if e.Approval.Request {
			am["request"] = true
		}
		m["approval"] = am
	case event.AskRequest:
		qs := make([]map[string]interface{}, 0, len(e.Ask.Questions))
		for _, q := range e.Ask.Questions {
			opts := make([]map[string]interface{}, 0, len(q.Options))
			for _, o := range q.Options {
				opt := map[string]interface{}{"label": o.Label}
				if o.Description != "" {
					opt["description"] = o.Description
				}
				opts = append(opts, opt)
			}
			qq := map[string]interface{}{"id": q.ID, "prompt": q.Prompt, "options": opts, "multi": q.Multi}
			if q.Header != "" {
				qq["header"] = q.Header
			}
			qs = append(qs, qq)
		}
		askMap := map[string]interface{}{"id": e.Ask.ID, "questions": qs}
		m["ask"] = askMap
	case event.CompactionStarted:
		// v4.26 对话流式重造：compaction 转译为 phase（Why：前端 reducer 对
		// compaction_started/done 无 case、事件被整体丢弃——压缩期间对话窗
		// 静默；统一走 phase 后前端零改动即可见「正在压缩上下文…」）。仅
		// 转译 wire 形态，磁盘日志仍按 compaction_started/done 落（不改格式）。
		m["kind"] = "phase"
		m["text"] = "正在压缩上下文…"
	case event.CompactionDone:
		m["kind"] = "phase"
		m["text"] = "压缩完成"
	case event.Retrying:
		// v4.26：Retrying 转译为 phase（此前无映射落到 unknown 被前端丢弃，
		// 流式恢复重试期间用户完全无感）。n/m 取自事件自带的重试进度。
		m["kind"] = "phase"
		m["text"] = fmt.Sprintf("正在重试 (%d/%d)", e.RetryAttempt, e.RetryMax)
	case event.SubagentMessage:
		// v4.26：子代理完成回投（对标 Codex 2026-08 "Report completed
		// sub-agent activity on parent turns"）。text=最终答复全文。
		// v4.27.2 收口：wire 层转译为 kind="message" + subagentRef——前端
		// reducer 的 message case 早已支持该可选字段（整体替换语义，把「子代理」
		// 徽标打在独立气泡上），此前 kind=subagent_message 前端无消费整条被丢，
		// 回投特性实际未通。磁盘日志仍按 subagent_message 落（EventLogSink 在
		// sink 链上游，转译只影响 wire）；轨迹/补拉折叠各自消费原始 kind。
		m["kind"] = "message"
		m["text"] = e.Text
		if e.SubagentRef != "" {
			m["subagentRef"] = e.SubagentRef
		}
		if e.ParentToolID != "" {
			m["parentId"] = e.ParentToolID
		}
	case event.Steer:
		// 运行中插话：agent 已把该消息作为当前回合 guidance 消费，
		// 前端以轻量 notice 回显（不渲染成独立用户气泡）。
		m["text"] = e.Text
		m["level"] = "info"
	}
	return m
}

// gaeaKindName 事件类型名映射（对齐 gaeaW WireEvent.EventKind）。
func gaeaKindName(k event.Kind) string {
	names := map[event.Kind]string{
		event.TurnStarted: "turn_started", event.Reasoning: "reasoning", event.Text: "text",
		event.Message: "message", event.ToolDispatch: "tool_dispatch", event.ToolResult: "tool_result",
		event.Usage: "usage", event.Notice: "notice", event.Phase: "phase",
		event.ApprovalRequest: "approval_request", event.AskRequest: "ask_request",
		event.TurnDone: "turn_done", event.CompactionStarted: "compaction_started",
		event.CompactionDone: "compaction_done", event.Steer: "notice",
		event.SubagentMessage: "subagent_message",
	}
	if n, ok := names[k]; ok {
		return n
	}
	return "unknown"
}

// gaeaLevelName 通知级别名。

// gaeaLevelName 通知级别名。
func gaeaLevelName(l event.Level) string {
	switch l {
	case event.LevelWarn:
		return "warn"
	default:
		return "info"
	}
}
