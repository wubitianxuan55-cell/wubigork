package app

// wenxue_engine.go — 图文板块（wenxue）独立会话引擎（2026-11-15 刀1，案A）。
//
// 仿 pantheon 议席自建注册表先例：图文板块懒建独立 controller（boot.Build
// 全套装配），与办公引擎（ga）物理并存、互不触碰：
//   · Cwd=当前小说工作空间（C:\AI\tuwen/<书>）——会话内 read_file/write_file/
//     bash/pins/exports 全部锚定小说目录（收口调研档 §1.4 的 Cwd 地基坑）；
//   · SessionDir=<小说>/.gaea/sessions（平铺，与既有图文会话布局一致）；
//   · 工具装配带 ToolFilter=wenxueToolFilter（办公域工具物理不装）；
//   · 事件走 gaea-wenxue-event / gaea-wenxue-subagent-text 独立通道（前端
//     图文树订阅面同步切，防双引擎事件串台）；
//   · 办公域 TurnDone 钩子（做梦/自动命名/COM 收尾）不进图文引擎。
// 办公 gaeaRuntime/控制器/重建链零改动（红线）；办公设置变更经
// wenxueReloadIfInit 让图文引擎跟随重建（配置是全局共享面）。

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	gaeaAgent "github.com/gaea/gaea/internal/gaea/agent"
	gaeaBoot "github.com/gaea/gaea/internal/gaea/boot"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/control"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider/bridge"
)

// wx 是图文板块的引擎运行时（与办公 ga 同构的最小面：无任务中心/无 DAG/
// 无追问外的宿主钩子）。wire 与办公各持一份——seq 打点/phase 节流按引擎独立，
// 断号检测互不干扰。
var wx = &wenxueRuntime{wire: newGaeaEventForwarder()}

type wenxueRuntime struct {
	mu sync.Mutex
	// reloadMu 序列化重建（2026-11-16 实弹死锁根修）：构建阶段绝不持 mu——
	// boot.Build 会同步回调 OnFollowUpReady，其写锁家族与 wx.mu 交叉时同
	// goroutine 重入 sync.Mutex 即永久挂死（此前切书/热重载整板块冻结）。
	// 重建串行化还保证「后交换者不会 Close 掉先交换者的现役控制器」。
	reloadMu sync.Mutex
	ctrl     *control.Controller
	wire     *gaeaEventForwarder
	// sends 是本进程正在执行 WenxueSend 的请求数（含引擎预检/冷启动等待）。
	// 前端 30s watchdog 会以 WenxueRunning 校准本地 running；预检期间 ctrl.Send
	// 尚未开始、Running() 仍为 false，若不把在途发送计入，会把正在冷启动的回合
	// 误判为结束（2026-11-24 文学板块 Strata 约 28s 后假停止）。
	sends int
	// followUp 子代理追问执行器（OnFollowUpReady 交给这里；语义同办公
	// ga.followUp+followUpMu：构建回调写入与追问读取各持独立锁，不得复用
	// mu——Build 位点回调与 mu 上的重建互斥是死锁源）。
	followUp   gaeaAgent.SubagentFollowUpRunner
	followUpMu sync.Mutex
}

// beginWenxueSend/endWenxueSend 标记一段发送（含最长 300s 的本地引擎冷启动
// 预检）为在途。WenxueRunning 据此让前端 watchdog 在 turn_started 到达前
// 保持“运行中”语义。
func beginWenxueSend() func() {
	wx.mu.Lock()
	wx.sends++
	wx.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			wx.mu.Lock()
			if wx.sends > 0 {
				wx.sends--
			}
			wx.mu.Unlock()
		})
	}
}

// wenxueCtrl 返回当前图文控制器（未初始化返回 nil）。
func wenxueCtrl() *control.Controller {
	wx.mu.Lock()
	defer wx.mu.Unlock()
	return wx.ctrl
}

// wenxueBrainDirOverride 测试缝（非空=顶替 MemoryUserDir 推导，防测试误写
// 真实图文脑——os.UserConfigDir 在 Windows 读 %AppData%，不受 USERPROFILE
// 重定向影响）。生产恒空。
var wenxueBrainDirOverride string

// wenxueBrainDir 图文记忆脑目录（2026-11-16 记忆左右脑批）：
// <MemoryUserDir>/wenxue-brain——与办公默认脑物理隔离（独立 docs/画像/
// remember 画像表/会话检索/引用消毒全随目录；DB 路径随目录推导分池）。
// 基目录不可得（os.UserConfigDir 失败）返回空=不 scope，回退默认脑
// （宁共享不建野目录）。
func wenxueBrainDir() string {
	if wenxueBrainDirOverride != "" {
		return wenxueBrainDirOverride
	}
	base := gaeaConfig.MemoryUserDir()
	if base == "" {
		return ""
	}
	return filepath.Join(base, "wenxue-brain")
}

// wenxueSystemPrompt 解析图文板块身份提示词：配置文件键优先（wenxue_system_
// prompt_file，读失败硬报错——用户显式配置的档位缺档不许静默吞），其次
// wenxue_system_prompt，双空 = 代码默认 DefaultWenxueSystemPrompt。ga.cfg
// 未就绪（测试/早期构建）恒不回落办公人格，直接用板块默认。
func wenxueSystemPrompt() (string, error) {
	if ga.cfg == nil {
		return gaeaConfig.DefaultWenxueSystemPrompt, nil
	}
	return ga.cfg.ResolveWenxueSystemPrompt()
}

// 图文事件通道：主对话流 + 子代理流式增量（与办公通道物理分线）。
const (
	wenxueEventChannel        = "gaea-wenxue-event"
	wenxueSubagentTextChannel = "gaea-wenxue-subagent-text"
	wenxueReadyEvent          = "gaea-wenxue-ready"
)

// WenxueEngineInit 初始化图文引擎（幂等）。模型走 gaea 模型中心（与办公同一
// bridge provider 注入），工作空间锚定当前小说目录。
func (a *App) WenxueEngineInit() error {
	wx.mu.Lock()
	ready := wx.ctrl != nil
	wx.mu.Unlock()
	if ready {
		return nil
	}

	// 模型中心客户端注入（bridge provider 全局；幂等）。功能级绑定改在
	// wenxueBuildController 构造期注入（feature=wenxue，见彼处注释）。
	bridge.SetClient(a.clientRef())

	// 配置加载：ga.cfg 未就绪（用户直进图文板块）时先补办公配置加载与
	// 全局 loader——这是配置装载（引擎未构建），不触碰办公控制器。
	ga.mu.Lock()
	if ga.cfg == nil {
		cfg, err := gaeaLoadConfig()
		if err != nil {
			ga.mu.Unlock()
			return err
		}
		ga.cfg = cfg
		// loader 无锁读 ga.cfg（2026-11-16 审查批1 根修：旧实现闭包内取 ga.mu
		// ——任何「持 ga.mu 调 config.Load」路径（applyOfficeFeatureModel/
		// gaeaApplyCfg 的 wx 跟随重建、办公 gaeaRebuildLocked 自身的
		// boot.Build→config.Load）同 goroutine 重入即整应用死锁。与办公 loader
		// 同口径：指针替换在持锁下进行，读取方只会拿到完整可用配置）。
		gaeaConfig.SetLoader(func() (*gaeaConfig.Config, error) {
			if ga.cfg == nil {
				return gaeaConfig.Default(), nil
			}
			return ga.cfg, nil
		})
	}
	ga.mu.Unlock()

	// 任务模板库落到小说工作空间（与办公同机制、不同根；幂等不覆盖用户文件）。
	// 图文 v1 不预置办公八条模板（刀4 换小说模板时再落）——这里只建目录语义，
	// ensureTaskTemplateCommands 按需调用交给刀4，先不挂。

	ctrl, err := a.wenxueBuildController()
	if err != nil {
		return err
	}
	wx.mu.Lock()
	wx.ctrl = ctrl
	wx.mu.Unlock()
	// 恢复当前小说最近一次会话（仅首次初始化；仅在已选定小说时）。
	a.wenxueResumeLastSession(ctrl)
	a.emit(wenxueReadyEvent, map[string]interface{}{"kind": "ready"})
	return nil
}

// wenxueBuildController 构建图文控制器（不持 wx.mu，调用方负责）。
// 骨架与办公 gaeaBuildController 同构；差异点逐条注释。
func (a *App) wenxueBuildController() (*control.Controller, error) {
	// 功能级模型绑定（feature=wenxue）构造期注入：boot.Build 是同步窗，窗内
	// 构造的全部 provider（主 agent+子代理）按当前绑定解析——每次构建前注入
	// 自己的绑定，杜绝与办公引擎共享 bridge 全局槽的交叉污染（双引擎隔离，
	// 2026-11-15 刀1 评审补强）。回落链：wenxue 绑定 → gaea 绑定 → 全局活跃；
	// wenxue 功能停用 = 清绑定回落全局活跃。
	if a.cfg != nil {
		eng, mdl := a.cfg.GetFeatureModel("wenxue")
		if !a.cfg.GetFeatureModelEnabled("wenxue") {
			eng, mdl = "", ""
		} else if eng == "" && mdl == "" {
			eng, mdl = a.cfg.GetFeatureModel("gaea")
		}
		bridge.SetFeature(eng, mdl)
	}
	// 事件转发：图文事件流 → gaea-wenxue-event 独立通道。办公域 TurnDone
	// 钩子（做梦/自动命名/COM 收尾）有意不挂——图文会话不触发办公记忆整理。
	sink := event.FuncSink(func(e event.Event) {
		if m := gaeaSubagentTextPayload(e); m != nil {
			a.emit(wenxueSubagentTextChannel, m)
			return
		}
		if m := wx.wire.payload(e); m != nil {
			a.emit(wenxueEventChannel, m)
		}
	})
	// 工作空间锚：当前小说目录（未选书=图文根本身，会话被 NewSession 拒绝，
	// 引擎仍可构建保证 Meta/面板可用）。
	ws := wenxueWorkspace()
	// 书籍信息契约投影同步（2026-11-17 审查批2 刀3）：档位行=事实源，
	// 恒守线行/执行契约节=投影；建书/切书/热重载重建都经此处幂等校正，
	// 会话注入（pinned）发生在 Build 之后，吃到的是校正后的文件。
	wenxueSyncBookInfoContract(ws)
	// 板块身份（2026-11-16 图文身份分板）：图文装配自己的身份提示词，不再
	// 吃办公 system_prompt（本机实锤：办公人格=土壤修复工程助手，图文会话
	// 开场自称「土壤修复办公专用AI助手」）。配置双键自定义（wenxue_system_
	// prompt / wenxue_system_prompt_file，文件键优先），空 = 代码默认
	// DefaultWenxueSystemPrompt。
	sysPrompt, err := wenxueSystemPrompt()
	if err != nil {
		return nil, err
	}
	ctrl, err := gaeaBoot.Build(a.ctx, gaeaBoot.Options{
		Model: "gaea",
		// T11 Guardian：图文委员会有真实工具审批面，不能只在办公引擎接
		// 自动评审；feature=wenxue 让评审调用跟随图文自己的模型绑定。
		Guardian: newGuardianReviewer(a, ga.cfg, "wenxue"),
		// 身份覆写：图文人格（见 wenxueSystemPrompt）。
		SystemPrompt: sysPrompt,
		// 账目标签分列（2026-11-17 审查批5 刀1）：wx 全部 provider（主 agent+
		// 子代理）用量标 "wenxue"，与办公 "gaea" 桶分列（PerFeature 分桶）。
		Feature: "wenxue",
		// 执行纪律板块缝（2026-11-18 众神殿 pan_1795007052552 v1.0）：图文
		// executor 换 WenxueSingleModelPrompt 独立常量+跳过 officeLive 段，
		// 不再吃办公执行后缀（此前文学最终串含办公事实底座/落盘/签退全段）。
		// 中立交付段照常单源追加。
		ExecBoard: "wenxue",
		// 空间锚（2026-11-18 众神殿余项·play 装配洞根修）：图文引擎恒 work
		// ——全局 session.space=play 时若跟随，AllowsSpace 会把基础读写工具
		// 与全部 wenxue_*（SpaceTag 恒 work）滤成零写工具。锚定 work 后工具/
		// 记忆读端按 work 装配（文学脑本就独立分池，SessionDir 已显式平铺，
		// Journal 已硬编码书内 .gaea/work）。
		SpaceAnchor: "work",
		EmitSubagentText: func(ref, text string) {
			a.emit(wenxueSubagentTextChannel, map[string]interface{}{
				"kind": "subagent_text", "text": text, "subagentRef": ref,
			})
		},
		OnFollowUpReady: func(runner gaeaAgent.SubagentFollowUpRunner) {
			// 独立 followUpMu（办公 ga 同款）：Build 位点同步回调，写锁不得
			// 复用 wx.mu——重建持 wx.mu 跨 Build 时会自死锁（2026-11-16）。
			wx.followUpMu.Lock()
			wx.followUp = runner
			wx.followUpMu.Unlock()
		},
		// OnNodeRunnerReady 不挂：DAG 文件流水线是办公域（图文右栏无该面）。
		OnSkillUse:      recordSkillUse,
		ReadPathAllowed: isPickedFile,
		RequireKey:      false,
		Sink:            sink,
		MaxSteps:        0,
		// 会话目录平铺在小说工作空间（与既有图文会话布局逐字节一致，
		// WenxueListSessions 读取端同口径）。
		SessionDir: gaeaConfig.WorkspaceSessionDir(ws, ""),
		Cwd:        ws,
		// 写入根显式接管=小说工作空间（2026-11-16 write_file 越界 bug 根修）：
		// cfg.Sandbox.WorkspaceRoot 是办公板块持久化的全局共享值（本机实锤
		// workspace_root=D:\新工作\2630合并沧州），WriteRootsAt 让它优先于
		// Cwd——图文会话写小说目录被拒。接管后 write_file/bash 只锚小说目录，
		// 不吃办公配置。
		WriteRoots: []string{ws},
		// 额外读边界同步接管（2026-11-16 同族审计）：ReadRootsAt 含办公持久化
		// 目录——不接管则图文代理可读整个办公工作区（跨板块数据泄漏）。
		// workspace 本身恒可读，拾取文件走 ReadPathAllowed 回调，域规范走
		// read_skill 按需加载。
		ReadRoots: []string{ws},
		// 办公域工具物理不装（ToolFilter，案A 拍板）；wenxue_* 专属四族只进
		// 本引擎（办公 ExtraTools 不含——办公会话零出现）。space 装配过滤照常。
		// 通用面走媒体根分档版（wenxueBaseTools）：image_gen/ocr 锚小说目录。
		ExtraTools: append(wenxueBaseTools(a), wenxueExtraTools(a)...),
		ToolFilter: wenxueToolFilter,
		// 技能面（刀2）：allowlist 子集 + 内嵌三件（索引/slash/run_skill/
		// read_skill/能力抽屉五面同源收口）。
		SkillFilter: wenxueSkillFilter,
		SkillExtras: wenxueSkillExtras(),
		// 记忆左右脑（2026-11-16）：图文脑独立目录（docs/画像/auto-memory/
		// remember 画像表/会话检索/引用消毒全随目录；DB 路径随目录推导分池）
		// ——与办公脑物理隔离。办公引擎不传=默认脑现状逐字节。基目录取不到
		// 时不 scope（回退默认，宁共享不建野目录）。
		MemoryUserDir: wenxueBrainDir(),
	})
	if err != nil {
		return nil, fmt.Errorf("图文引擎初始化失败: %w", err)
	}
	if ga.cfg != nil {
		ctrl.SetLogFormat(ga.cfg.EffectiveLogFormat())
		// 会话空间自描述与办公同口径（novel 会话历史上按 work 折算；路径
		// 仍是唯一真相源，见 controller_session.writeSpaceForLocked）。
		ctrl.SetSpace(ga.cfg.EffectiveSessionSpace())
	}
	ctrl.EnableInteractiveApproval()
	return ctrl, nil
}

// wenxueResumeLastSession 恢复当前小说最近一次有内容的会话（仅首次初始化）。
// 体与办公 resumeLastSession 同构（ListSessions 最近一份 → ResumeFromDisk →
// 中断摘要注入 → 上下文读数种子），目录锚换小说工作空间。
func (a *App) wenxueResumeLastSession(ctrl *control.Controller) {
	ws := wenxueWorkspace()
	if ws == wenxueWorkspaceRoot() {
		return // 未选书：图文根不承载会话
	}
	dir := gaeaConfig.WorkspaceSessionDir(ws, "")
	infos, err := gaeaAgent.ListSessions(dir)
	if err != nil || len(infos) == 0 {
		return
	}
	loaded, err := ctrl.ResumeFromDisk(infos[0].Path)
	if err != nil {
		slog.Warn("图文引擎自动恢复最近会话失败", "path", wenxueRedactPath(infos[0].Path), "error", err)
		return
	}
	injectInterruption(loaded, infos[0].Path)
	ctrl.SeedContextUsage()
	slog.Info("图文引擎已自动恢复最近会话", "path", wenxueRedactPath(infos[0].Path), "messages", len(loaded.Messages))
}

// wenxueReloadIfInit 图文引擎已初始化时按当前配置重建（办公设置变更/切书后
// 跟随）。reloadMu 串行化整段（并发重建会把现役控制器 Close 掉）；构建阶段
// 不持 wx.mu——boot.Build 同步回调 OnFollowUpReady（独立 followUpMu 写入），
// 持 wx.mu 跨 Build 即同 goroutine 重入自死锁（2026-11-16 实弹：切书/热重载
// 整板块冻结，点击无响应）。独立加锁，不持 ga.mu 调用（boot 读全局 loader
// 闭包内部会取 ga.mu，与 ga.mu 无死锁面）。
func (a *App) wenxueReloadIfInit() {
	wx.reloadMu.Lock()
	defer wx.reloadMu.Unlock()
	wx.mu.Lock()
	initialized := wx.ctrl != nil
	wx.mu.Unlock()
	if !initialized {
		return
	}
	newCtrl, err := a.wenxueBuildController()
	if err != nil {
		slog.Warn("图文引擎跟随重建失败（保持旧引擎）", "error", err)
		return
	}
	wx.mu.Lock()
	old := wx.ctrl
	wx.ctrl = newCtrl
	wx.wire.reset() // 配置重建=会话面重置，seq 归零防断号
	wx.mu.Unlock()
	if old != nil {
		old.Close()
	}
	a.emit(wenxueReadyEvent, map[string]interface{}{"kind": "reloaded"})
}

// WenxueReload 热加载图文引擎（对齐 GaeaReload 语义）：重读持久化配置并重建。
// 未初始化时先完成初始化。失败保持旧引擎。reloadMu 串行化 + 构建不持 wx.mu
// （同 wenxueReloadIfInit 死锁根修，2026-11-16）。
func (a *App) WenxueReload() (GaeaReloadResult, error) {
	wx.reloadMu.Lock()
	defer wx.reloadMu.Unlock()
	wx.mu.Lock()
	initialized := wx.ctrl != nil
	wx.mu.Unlock()
	if !initialized {
		if err := a.WenxueEngineInit(); err != nil {
			return GaeaReloadResult{}, err
		}
	}
	ga.mu.Lock()
	cfg, err := gaeaLoadConfig()
	if err != nil {
		ga.mu.Unlock()
		return GaeaReloadResult{}, fmt.Errorf("图文热加载配置失败: %w", err)
	}
	ga.cfg = cfg
	ga.mu.Unlock()

	newCtrl, err := a.wenxueBuildController()
	if err != nil {
		return GaeaReloadResult{}, err
	}
	wx.mu.Lock()
	old := wx.ctrl
	wx.ctrl = newCtrl
	wx.wire.reset()
	wx.mu.Unlock()
	if old != nil {
		old.Close()
	}
	res := GaeaReloadResult{Tools: len(newCtrl.Tools()), Skills: len(newCtrl.Skills())}
	a.emit(wenxueReadyEvent, map[string]interface{}{"kind": "reloaded"})
	return res, nil
}

// applyWenxueFeatureModel 图文功能绑定变更后跟随重建（绑定注入在 Build 位点，
// 重建即取新值；未初始化跳过——首次 WenxueEngineInit 自然取最新配置）。
func (a *App) applyWenxueFeatureModel() {
	a.wenxueReloadIfInit()
}

// wenxueRequireCtrl 取图文控制器并保证引擎已初始化（发送/切会话等入口共用）。
func (a *App) wenxueRequireCtrl() (*control.Controller, error) {
	if err := a.WenxueEngineInit(); err != nil {
		return nil, err
	}
	c := wenxueCtrl()
	if c == nil {
		return nil, errors.New("图文引擎未初始化")
	}
	return c, nil
}
