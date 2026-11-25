package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/BurntSushi/toml"

	gaeaAgent "github.com/gaea/gaea/internal/gaea/agent"
	gaeaBoot "github.com/gaea/gaea/internal/gaea/boot"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/fileutil"
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
	// nodeRunner 是「全新子代理运行」执行器（6.3 文件流水线节点）：boot 用
	// taskTool.RunNew 管道组装，经 OnNodeRunnerReady 交给这里；语义同 followUp
	// （随 controller 重建更新）。dagMu 串行化 run 档读改写，dagCancels 是
	// runID→cancel 的在途执行登记（终态级联 + 重启中断懒清扫判据）。
	nodeRunner gaeaAgent.SubagentRunRunner
	dagMu      sync.Mutex
	dagCancels sync.Map
	// wire 是 gaea 事件流 → 前端 gaea-event 的转发层（v4.26 对话流式重造）：
	// wire seq 打点 + phase 节流的唯一状态点，随进程存活（不随 controller
	// 重建重置——设置变更重建引擎不打断当前会话，seq 回退会破坏前端断号检测）。
	wire *gaeaEventForwarder
	// runMode 是当前 controller 装配所用运行模式（2026-11-21 众神殿终案，
	// pan_1795202763988）：由 gaeaBuildController 在持锁调用下写入。与
	// cfg.OfficeRunMode()（新会话默认档）不等时，GaeaNewSession 先重建引擎
	// 再建会话——「会话创建时定档、当前会话不中途切换」的 DSH 同构语义。
	runMode string
	// toolFace/groupSig/cuEnabled 是当前装配所用工具面档（lean|standard）、组
	// 覆盖签名与电脑操控注册闸（2026-11-21 工具面双轴批，pan_1795224680124）：
	// GaeaNewSession 判定 (runMode, toolFace, groupSig, cuEnabled) 四元组不等
	// 时先重建再建会话。
	toolFace  string
	groupSig  string
	cuEnabled bool
}

// gaeaLoadConfig 加载办公引擎配置：内置默认 + 用户持久化文件（若有），
// 再注入 bridge provider（kind=gaea 走 gaea 模型中心）。返回可直接修改的配置。
// 持久化文件解析失败时隔离损坏件并按默认继续（对齐壳层 T6-9.4 范式，
// 2026-10-06 断电事故根修落地）——损坏的 config.toml 绝不能把办公引擎
// 锁死在 GaeaInit 启动失败上。
func gaeaLoadConfig() (*gaeaConfig.Config, error) {
	cfg := gaeaConfig.Default()
	if p := gaeaConfig.UserConfigPath(); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			if _, err := toml.Decode(string(b), cfg); err != nil {
				if backup, berr := fileutil.BackupCorrupt(p, b); berr != nil {
					slog.Error("gaea: 持久化配置损坏且备份失败，按默认继续", "path", p, "err", err, "backup_error", berr)
				} else {
					slog.Warn("gaea: 持久化配置解析失败，已备份损坏件并按默认继续", "path", p, "backup", backup, "err", err)
				}
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("gaea: 读取持久化配置 %s: %w", p, err)
		}
	}
	// GAEA_WALKTHROUGH=1 走查沙箱：必须在用户 toml 合并后覆盖（v4.418，与 config.Load 同调一助手）
	gaeaConfig.ApplyWalkthroughOverride(cfg)
	cfg.DefaultModel = "gaea"
	cfg.Providers = []gaeaConfig.ProviderEntry{{
		Name:  "gaea",
		Kind:  "wubigrok", // 内部 provider 注册名（bridge provider）
		Model: "",
		// 上下文窗口按实际绑定模型的能力取保守值（此前写 1M 导致自动压缩
		// 阈值高达 80 万 token，办公会话膨胀到十几万也从不压缩，模型首字
		// 极慢、看起来像“没流式输出”）。256k 下 80% 阈值≈204k，超限自动压缩。
		// fallback only: Build applies the routed model's real ContextLength.
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
	// 办公功能级模型绑定（feature=gaea）构造期注入（2026-11-15 刀1 评审补强）：
	// boot.Build 同步窗内全部 provider 按当前绑定解析——每次构建前注入办公自己的
	// 绑定，杜绝图文引擎构建后 bridge 全局槽残留图文绑定被办公重建误取（双引擎
	// 隔离）。启停语义与 FeatureModelBar 一致：停用=清绑定回落全局活跃引擎。
	if a.cfg != nil {
		eng, mdl := a.cfg.GetFeatureModel("gaea")
		if !a.cfg.GetFeatureModelEnabled("gaea") {
			eng, mdl = "", ""
		}
		bridge.SetFeature(eng, mdl)
	}
	// 办公 bridge provider 原先固定为 256k，导致 DeepSeek 的 1M 模型也过早
	// 触发压缩。装配前按办公实际路由模型读取目录能力；未知才保留 256k。
	if ga.cfg != nil && len(ga.cfg.Providers) > 0 {
		p := &ga.cfg.Providers[0]
		p.ContextWindow = a.gaeaRoutedContextWindow(p.ContextWindow)
	}
	// 事件转发：gaea 事件流 → 前端 gaea-event 回调。经转发层统一打点 wire
	// seq 并对 phase 节流（v4.26 对话流式重造，见 gaeaEventForwarder）。
	sink := event.FuncSink(func(e event.Event) {
		a.emitGaeaEvent(e)
		// auto-dream 做梦机制已删除（2026-11-20 用户拍板：轮次后台自动提炼
		// 记忆建议质量差且干扰正常记忆运作——remember 工具+面板手写+显式
		// 接受才是正口）。
		if e.Kind == event.TurnDone && e.Err == nil {
			// 会话自动命名（dsh session-title 蒸馏）：首个成功回合后给无标题
			// 会话起名（办公本地路由，单飞）。
			a.maybeAutoTitleAfterTurn()
			// COM 档回合收尾（dsh settle 蒸馏，波3）：静默文档自动保存关闭 +
			// card 收尾——无 office_* 活动时零 helper 命令（office_com_settle.go）。
			a.maybeOfficeComSettle()
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
	opts := gaeaBoot.Options{
		Model: "gaea",
		// T11 Guardian：[guardian] enabled 时注入动作级自动评审（评审走办公
		// 本地优先路由）；未启用传 nil=审批纯人工（现状逐字节）。
		Guardian: newGuardianReviewer(a, ga.cfg, "office"),
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
		// 6.3 文件流水线：全新子代理执行器交给 ga 保存（DAG 节点执行器调用）。
		OnNodeRunnerReady: func(runner gaeaAgent.SubagentRunRunner) {
			ga.followUpMu.Lock()
			ga.nodeRunner = runner
			ga.followUpMu.Unlock()
		},
		// 7.2-2 判据②：read_skill/run_skill 工具级调用计数（包级函数无状态）。
		OnSkillUse:      recordSkillUse,
		ReadPathAllowed: isPickedFile,
		RequireKey:      false,
		Sink:            sink,
		MaxSteps:        0,
		SessionDir:      gaeaConfig.WorkspaceSessionDir(gaeaCwd(), space),
		// 工作空间根：基础工具（read/write/bash/ls）相对路径基于它，
		// 而非进程 cwd（否则办公 agent 在启动目录而非用户工作空间操作）。
		Cwd: gaeaCwd(),
		// 注入需要应用服务的工具（生图/画图复用模型中心模型与图片后端）。
		// 3.0 Step 3d #6：专业工具（ocr 等）经 gaeaSpecialistTools 集中注册，
		// 展开进 ExtraTools，避免装配列表与定义漂移。
		ExtraTools: gaeaExtraTools(a),
	}
	// 运行模式（下面 ToolHide/ModeLine/技能面同用）：记忆读端也按它收窄——会话只
	// 看到「通用 + 本模式」的记忆（2026-11-25 办公工具梳理批·记忆按模式隔离）。
	runMode := normalizeOfficeRunMode(ga.cfg.OfficeRunMode())
	opts.MemoryMode = runMode
	// 运行模式×工具面双轴（2026-11-21 pan_1795202763988 + pan_1795224680124
	// 终案）：组合器（office_runmode.go）按 run_mode→tool_face→组覆盖确定性
	// 汇总 Hide 名单与 ModeLine；技能门窄 deny 跟随最终启用组（schedule-edit/
	// cad-takeoff），配合 SkillFilterKeepsInstall 保住 install_skill。
	// ga.cfg 为 nil（部分测试路径）时归一 standard/lean=最小面。
	face := normalizeOfficeToolFace(ga.cfg.OfficeToolFace())
	overrides := ga.cfg.OfficeToolGroupOverrides()
	hide := composeToolHide(runMode, face, overrides)
	opts.ToolHide = hide
	opts.ModeLine = composeModeLine(runMode, face, hide)
	opts.SkillExtras = officeRunModeSkillExtras(runMode)
	// 模式专属身份提示词（2026-11-21 用户拍板，wenxue 身份分板同构）：造价/
	// 方案档顶替全局 agent.system_prompt 作 L1 基底（配置键 system_prompt_cost/
	// system_prompt_proposal 可覆盖内置默认）；standard 空=现状逐字节。
	opts.SystemPrompt = officeModeSystemPrompt(runMode, ga.cfg)
	if sf := officeRunModeSkillFilter(hide); sf != nil {
		opts.SkillFilter = sf
		opts.SkillFilterKeepsInstall = true
	}
	ga.runMode = runMode
	ga.toolFace = face
	ga.groupSig = toolGroupOverridesSig(overrides)
	if ga.cfg != nil {
		ga.cuEnabled = ga.cfg.ComputerUse.Enabled // 直接字段访问须判 nil（ga.cfg 可空）
	}
	ctrl, err := gaeaBoot.Build(a.ctx, opts)
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

// gaeaRoutedContextWindow resolves the window for the model the office agent
// will actually use. Feature binding wins; otherwise follow the active global
// engine. Missing/zero model capability keeps the caller's fallback.
func (a *App) gaeaRoutedContextWindow(fallback int) int {
	if fallback <= 0 {
		fallback = 256_000
	}
	if a == nil || a.core == nil || a.engineMgr == nil {
		return fallback
	}
	if a.cfg != nil && a.cfg.GetFeatureModelEnabled("gaea") {
		if eng, model := a.cfg.GetFeatureModel("gaea"); eng != "" && model != "" {
			return a.engineModelContextWindow(eng, model, fallback)
		}
	}
	return a.engineModelContextWindow(a.GetActiveEngine(), a.GetActiveModel(), fallback)
}

func (a *App) engineModelContextWindow(engineID, model string, fallback int) int {
	if a == nil || a.engineMgr == nil || engineID == "" {
		return fallback
	}
	eng, ok := a.engineMgr.GetEngine(engineID)
	if !ok {
		return fallback
	}
	if model == "" {
		model = eng.DefaultModel
	}
	for _, m := range eng.Models {
		if m.ID == model && m.ContextLength > 0 {
			return m.ContextLength
		}
	}
	// 模型目录尚未刷新/被裁剪时，DeepSeek v4 仍按官方 1M 处理，避免再次
	// 静默回落到办公默认的 256k。
	modelID := strings.ToLower(model)
	if engineID == "deepseek" && (strings.Contains(modelID, "v4") ||
		strings.HasPrefix(modelID, "deepseek-flash") ||
		strings.HasPrefix(modelID, "deepseek-pro")) {
		return 1_000_000
	}
	return fallback
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

	// 1. 注入模型中心客户端（bridge provider 的底层）。功能级绑定改在
	// gaeaBuildController 构造期注入（双引擎各表各的绑定，见彼处注释）。
	bridge.SetClient(a.clientRef())
	if eng, model := a.cfg.GetFeatureModel("gaea"); eng != "" {
		slog.Info("办公模型绑定生效", "engine", eng, "model", model)
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
	// 图文板块（wenxue）专属功能绑定：变更即时重建图文引擎（绑定注入在
	// Build 位点，见 wenxueBuildController）。
	if feature == "wenxue" {
		a.applyWenxueFeatureModel()
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
	// 图文启停同语义：停用=清绑定回落全局活跃引擎（Build 位点读 enabled）。
	if feature == "wenxue" {
		a.applyWenxueFeatureModel()
	}
	return nil
}

// applyOfficeFeatureModel 注入办公 bridge 的 feature 模型，并在办公引擎已初始化时重建
// controller。办公尚未打开（ga.ctrl == nil）时仅更新注入，后续 GaeaInit 会再按最新配置注入。
func (a *App) applyOfficeFeatureModel(engine, model string) {
	// 绑定注入已收敛到 gaeaBuildController 构造期（双引擎各表各的绑定）；
	// 此处只负责重建生效。
	contextChanged := false
	contextWindow := 0
	ga.mu.Lock()
	if ga.ctrl != nil && ga.cfg != nil {
		if err := a.gaeaRebuildLocked(); err != nil {
			slog.Warn("办公模型绑定变更后重建引擎失败", "error", err)
		} else if len(ga.cfg.Providers) > 0 {
			contextChanged = true
			contextWindow = ga.cfg.Providers[0].ContextWindow
		}
	}
	ga.mu.Unlock()
	if contextChanged {
		// 重建完成后才广播。core.feature-model-changed 早于本次重建，不能让
		// UI 在那个时点读到旧 256k controller；事件也必须在解锁后发出。
		a.emit("office-context-changed", map[string]interface{}{"window": contextWindow})
	}
	// 2026-11-15 刀1：图文独立引擎跟随（同全局模型注入面；未初始化跳过）。
	// 不可持 ga.mu 调用（wx 构建经全局 loader 读 ga.cfg，同 goroutine 重入
	// 即自死锁；2026-11-16 审查批1 修正——原 defer 把本调用关在临界区内，
	// 与本注释矛盾）。
	a.wenxueReloadIfInit()
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
		// v4.414.1：静默 no-op 改诚实报错——引擎未初始化时调用方（UI 新会话/
		// 走查脚本）拿不到「后端会话未切换」的信号，后续消息会写进自动恢复的
		// 旧会话（真机走查实录：走查脚本先 NewSession 后 Send，全部落进用户
		// 上一个会话）。前端 catch 有可见提示路径，行为收紧无害。
		return errors.New("办公引擎未初始化：先发送一条消息完成引擎装配，再新建会话")
	}
	// 运行闸（照 GaeaSubagentFollowUp 先例）：run loop 只在 idle 时换 session
	//（agent.go 的调用方约定），回合中换会话=无锁直读 a.session 指针与
	// SetSession 交错+后续消息落旧 session 而路径已换新。绑定层自守，防
	// httpbridge 绕过前端 busy 闸。
	if ga.ctrl.Running() {
		return errors.New("回合正在运行，请先停止当前回合再开启新会话")
	}
	// 会话创建时定档定面（2026-11-21 pan_1795202763988 + pan_1795224680124
	// 终案，DSH 同构）：配置的 (run_mode, tool_face, 组覆盖) ≠ 当前引擎装配
	// 三元组时先重建引擎，再建会话——「切换=登记默认值，新会话生效；当前
	// 会话不中途切换」。重建在此刻才发生：切换动作本身只持久化配置。
	// ga.cfg=nil（部分测试/未装配路径）：无配置可比，跳过判面重建（旧行为——
	// 建会话不触发引擎构建），直接建会话。
	if ga.cfg != nil {
		cfgMode := normalizeOfficeRunMode(ga.cfg.OfficeRunMode())
		cfgFace := normalizeOfficeToolFace(ga.cfg.OfficeToolFace())
		cfgSig := toolGroupOverridesSig(ga.cfg.OfficeToolGroupOverrides())
		cfgCU := ga.cfg.ComputerUse.Enabled
		if ga.runMode != cfgMode || ga.toolFace != cfgFace || ga.groupSig != cfgSig || ga.cuEnabled != cfgCU {
			if err := a.gaeaRebuildLocked(); err != nil {
				return fmt.Errorf("运行模式/工具面切换重建失败（保持当前面）: %w", err)
			}
		}
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
		if len(e.Tool.Sources) > 0 {
			t["sources"] = e.Tool.Sources
		}
		m["tool"] = t
	case event.Notice:
		m["text"] = e.Text
		m["level"] = gaeaLevelName(e.Level)
	case event.Phase:
		m["text"] = e.Text
	case event.TurnStarted:
		// U5 触发源徽标（dsh ui 蒸馏收尾刀）：回合发起者随线事件下发。
		// user 是缺省形态不下发（老前端/普通轮线格式逐字节不变）；
		// wake/subagent 非用户回合，前端 WorkHeader 据此画归属徽标。
		if e.Trigger != "" && e.Trigger != event.TriggerUser {
			m["trigger"] = e.Trigger
		}
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
		if e.Approval.Source != "" {
			am["source"] = e.Approval.Source
		}
		m["approval"] = am
	case event.ApprovalResolved:
		// 注意力事务结果侧：持久层账本事件。透传线格式便于将来的卡片状态
		// 对账/补拉；当前前端 reducer 对未知 kind 无操作，不产生 UI 变化。
		m["resolved"] = map[string]interface{}{
			"id": e.Resolved.ID, "tool": e.Resolved.Tool, "subject": e.Resolved.Subject,
			"source": e.Resolved.Source, "decision": e.Resolved.Decision,
			"outcome": e.Resolved.Outcome, "latencyMs": e.Resolved.LatencyMS,
		}
	case event.AskResolved:
		m["askResolved"] = map[string]interface{}{
			"id": e.AskResolved.ID, "outcome": e.AskResolved.Outcome,
			"answered": e.AskResolved.Answered, "latencyMs": e.AskResolved.LatencyMS,
		}
	case event.ApprovalAuto:
		// 自动放行账本（没弹卡、由机器决定）：同样只落持久层，前端 reducer
		// 无操作；线格式单列是为了将来做「谁替人批了什么」的审计视图。
		m["auto"] = map[string]interface{}{
			"tool": e.Auto.Tool, "subject": e.Auto.Subject,
			"source": e.Auto.Source, "decided": e.Auto.Decided,
		}
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
		event.ApprovalResolved: "approval_resolved", event.AskResolved: "ask_resolved",
		event.ApprovalAuto: "approval_auto",
		event.TurnDone:     "turn_done", event.CompactionStarted: "compaction_started",
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

// gaeaExtraTools 办公 agent ExtraTools 装配单点（P0 注册表地图：唯一注册表
// 四入口之一，boot.Build reg.Add 同账；新增应用层工具只改这一处）。P1a
// office 工具族（office_generate + 四类写）同点装配，不新开装配源
// （2026-11-09 众神殿 pan_1794234593182 裁决 action-3）。
// 媒体根=gaeaCwd 现状（nil 缝）；图文引擎的媒体根分档版见 wenxueBaseTools。
func gaeaExtraTools(a *App) []tool.Tool {
	return extraToolsWithMediaRoot(a, nil)
}

// extraToolsWithMediaRoot gaeaExtraTools 的媒体根注入版（2026-11-15 众神殿
// 首刀媒体分档，pan_1794754879290）：mediaRoot 非 nil 时 image_gen/ocr 锚该根
// （仅图文引擎用）；nil=gaeaCwd() 办公现状逐字节。新增应用层工具仍只改这一处。
func extraToolsWithMediaRoot(a *App, mediaRoot func() string) []tool.Tool {
	tools := []tool.Tool{
		imageGenTool{a: a, mediaRoot: mediaRoot},
		diagramTool{a: a},
		routineLLMTool{a: a},
		translateTextTool{a: a},
		factAddTool{},
		factListTool{},
		factClearTool{},
		// 6.3 文件流水线：规划工具 dag_plan 已于 2026-11-25 从模型工具面下线
		// （68 个真实办公会话零调用；办公工具梳理批）。运行面（任务中心→文件
		// 流水线）与模板实例化保留，实现留作测试夹具。
		// P1a 结构化写入口：office_generate（从零生成声明入口）+ 四类写升
		// agent 工具（docx_apply/docx_accept/pptx_apply/xlsx_apply）。全部走
		// appendOfficeEvidence 单点 Journal；实现 WritePathResolver、不标
		// PersistWrite（FilterRegistry 不摘，write_paths 声明面可用）。
		officeGenerateTool{a: a},
		officeDocxApplyTool{a: a},
		officeDocxAcceptTool{a: a},
		officePptxApplyTool{a: a},
		officeXlsxApplyTool{a: a},
		// 波2 扩面：xlsx_profile 只读列概况（规划输入；改文件仍走 xlsx_apply）。
		xlsxProfileTool{a: a},
		// P1-2 质量门禁（灵犀蒸馏）：xlsx_recalc 暴露 LibreOffice 重算链 +
		// xlsx_audit 只读复核（错误值/自引用/未重算）——交付门=零公式错误。
		xlsxRecalcTool{a: a},
		xlsxAuditTool{a: a},
		// P4 持久任务中心：批量转 PDF 提交型工具（断窗续跑/可取消/可重试）。
		officeBatchTool{a: a},
		// 运行模式专用面（2026-11-21 众神殿 pan_1795202763988 终案，五件）：
		// 造价三件（cost_project 读/cost_project_apply 写+PersistWrite/
		// cost_audit 机检，锚 workcost 权威面）+ 方案两件（longdoc_manifest
		// 清单机检/longdoc_build 组装交付）。模式可见性由 gaeaBuildController
		// 的 ToolHide 缝按档收放（standard 全藏/cost 藏方案族/proposal 藏造价
		// 族）；spacetags 静态表与 registrymap 契约行同批（三处同批纪律）。
		costProjectTool{a: a},
		costProjectApplyTool{a: a},
		costAuditTool{a: a},
		// 造价库维护（2026-11-25 造价数据库推倒重建）：导入五表工作簿、定额与
		// 工料机增删改调价、批量调价、删除——造价模式的唯一库维护入口。
		costLibraryTool{a: a},
		longdocManifestTool{a: a},
		longdocBuildTool{a: a},
		longdocVerifyTool{a: a},
	}
	// dsh-office COM 实时协同档（波1）：本机装了 Office/WPS 才注册，无则
	// 整族不出现、无头管线照旧（docs/gaea-dsh-office-distill-research-2026-11.md）。
	if officeComAvailable() {
		tools = append(tools,
			officeComOpenTool{},
			officeComStatusTool{},
			officeComReadTool{},
			officeComEditTool{},
			officeComRenderTool{},
			officeComSaveTool{},
			officeComCloseTool{},
			officeComHelpTool{},
			// P0-3 选区上下文：读用户此刻选中的内容 + 前后文（划词即改）。
			officeComSelectionTool{},
			// P0-2 受控宏逃生舱（A 档）：宏白名单展开成既有 ops，复用同一条
			// COM 执行路径；不给任意脚本（docs/gaea-lingxi-optimize-plan-2026-11.md）。
			officeMacroTool{},
		)
	}
	return append(tools, specialistToolsWithMediaRoot(a, mediaRoot)...)
}
