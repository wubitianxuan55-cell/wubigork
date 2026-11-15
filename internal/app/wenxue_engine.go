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
	mu   sync.Mutex
	ctrl *control.Controller
	wire *gaeaEventForwarder
	// followUp 子代理追问执行器（OnFollowUpReady 交给这里；语义同办公 ga.followUp）。
	followUp gaeaAgent.SubagentFollowUpRunner
}

// wenxueCtrl 返回当前图文控制器（未初始化返回 nil）。
func wenxueCtrl() *control.Controller {
	wx.mu.Lock()
	defer wx.mu.Unlock()
	return wx.ctrl
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
		gaeaConfig.SetLoader(func() (*gaeaConfig.Config, error) {
			ga.mu.Lock()
			defer ga.mu.Unlock()
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
	ctrl, err := gaeaBoot.Build(a.ctx, gaeaBoot.Options{
		Model: "gaea",
		EmitSubagentText: func(ref, text string) {
			a.emit(wenxueSubagentTextChannel, map[string]interface{}{
				"kind": "subagent_text", "text": text, "subagentRef": ref,
			})
		},
		OnFollowUpReady: func(runner gaeaAgent.SubagentFollowUpRunner) {
			wx.mu.Lock()
			wx.followUp = runner
			wx.mu.Unlock()
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
		// 办公域工具物理不装（ToolFilter，案A 拍板）；wenxue_* 专属四族只进
		// 本引擎（办公 ExtraTools 不含——办公会话零出现）。space 装配过滤照常。
		ExtraTools: append(gaeaExtraTools(a), wenxueExtraTools(a)...),
		ToolFilter: wenxueToolFilter,
		// 晨报预载/项目本体是 work 空间办公会话注入段——图文一律关。
		MorningPreload: false,
		ProjectBrief:   false,
		// 技能面（刀2）：allowlist 子集 + 内嵌三件（索引/slash/run_skill/
		// read_skill/能力抽屉五面同源收口）。
		SkillFilter: wenxueSkillFilter,
		SkillExtras: wenxueSkillExtras(),
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
		slog.Warn("图文引擎自动恢复最近会话失败", "path", infos[0].Path, "error", err)
		return
	}
	injectInterruption(loaded, infos[0].Path)
	ctrl.SeedContextUsage()
	slog.Info("图文引擎已自动恢复最近会话", "path", infos[0].Path, "messages", len(loaded.Messages))
}

// wenxueReloadIfInit 图文引擎已初始化时按当前配置重建（办公设置变更后跟随）。
// 独立加锁，不持 ga.mu 调用（boot 读全局 loader，与 ga.mu 无死锁面）。
func (a *App) wenxueReloadIfInit() {
	wx.mu.Lock()
	initialized := wx.ctrl != nil
	wx.mu.Unlock()
	if !initialized {
		return
	}
	wx.mu.Lock()
	defer wx.mu.Unlock()
	newCtrl, err := a.wenxueBuildController()
	if err != nil {
		slog.Warn("图文引擎跟随重建失败（保持旧引擎）", "error", err)
		return
	}
	old := wx.ctrl
	wx.ctrl = newCtrl
	wx.wire.reset() // 配置重建=会话面重置，seq 归零防断号
	if old != nil {
		old.Close()
	}
	a.emit(wenxueReadyEvent, map[string]interface{}{"kind": "reloaded"})
}

// WenxueReload 热加载图文引擎（对齐 GaeaReload 语义）：重读持久化配置并重建。
// 未初始化时先完成初始化。失败保持旧引擎。
func (a *App) WenxueReload() (GaeaReloadResult, error) {
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

	wx.mu.Lock()
	defer wx.mu.Unlock()
	newCtrl, err := a.wenxueBuildController()
	if err != nil {
		return GaeaReloadResult{}, err
	}
	old := wx.ctrl
	wx.ctrl = newCtrl
	wx.wire.reset()
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
