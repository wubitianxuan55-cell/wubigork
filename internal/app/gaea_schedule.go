package app

// 阶段 5 T5-3a/b/c：本地模型调度纵深
//
//   - T5-3a 保活 keep-warm：每 5 分钟一轮，对 herdsman 模型目录中 Running 的模型
//     发轻量流式探针（POST /v1/chat/completions，max_tokens=8），防止空闲卸载/降温，
//     保持「说用就能用」。只探不启——herdsman model_scheduling.local_concurrency=1，
//     同一时间只服务一个模型，绝不主动 start。开关 keep_warm_enabled 关闭时整轮
//     跳过；探针失败只记日志，待下一轮 catalog 重新显示 Running 再探。
//   - T5-3b 启动自动预载：Startup 后延迟 10s，按功能绑定优先级 gaea→office→chat
//     取第一个 engine=="herdsman" 的模型，查 catalog 确认为「已安装且未运行」后
//     调 HerdsmanModelStart（--wait 等冷启动完成，后台 goroutine 不阻塞启动）。
//     只预载一个；未安装/已在跑/非 herdsman 都跳过。开关 auto_preload 关闭时跳过。
//   - T5-3c 换模预计等待：GaeaModelSwitchEstimate 返回切换目标模型的
//     hot/cold/download/unknown 状态与预计等待秒数，供前端切换前提示。

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gaea/gaea/internal/config"
	"github.com/gaea/gaea/internal/modelengine"
)

// ollamaProbeClient Ollama 原生 API 探测专用客户端（超时由请求 ctx 控制，
// 独立实例避免与引擎主客户端配置互相牵连）。
var ollamaProbeClient = &http.Client{}

// ─── T5-3a 保活 keep-warm ──────────────────────────────────

// keepWarmProbeTimeout 单模型探针超时（15s：足够首 token 返回，超时视为不可用）。
const keepWarmProbeTimeout = 15 * time.Second

// keepWarmInterval 保活轮询间隔（5 分钟）。
const keepWarmInterval = 5 * time.Minute

// keepWarmProbe 单模型轻量探针（可注入测试）。成功返回 nil。
// 走 OpenAI 兼容 /v1/chat/completions 流式接口，body 固定
// {model, messages:[{role:"user",content:"hi"}], max_tokens:8, stream:true}；
// HTTP 客户端复用 herdsmanBenchHTTP（可注入替身），超时由调用方 ctx 控制。
var keepWarmProbe = func(ctx context.Context, baseURL, model string) error {
	body, err := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": "hi"},
		},
		"max_tokens": 8,
		"stream":     true,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		v1Join(baseURL)+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := herdsmanBenchHTTP.Do(req)
	if err != nil {
		return fmt.Errorf("连接 Herdsman 失败（模型未运行/服务不可用）: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Herdsman 返回 HTTP %d", resp.StatusCode)
	}
	return nil
}

// startKeepWarm 启动本地模型保活轮询（幂等，Startup 装配；Shutdown 关闭 stop）。
// 立即跑一轮（引擎已就绪），之后每 5 分钟一轮。
func (a *App) startKeepWarm() {
	a.officeState.keepWarmOnce.Do(func() {
		stop := make(chan struct{})
		a.officeState.keepWarmStop = stop
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("keep-warm panic recovered", "panic", r)
				}
			}()
			for {
				statuses := a.keepWarmRound()
				if a.core != nil && len(statuses) > 0 {
					a.emit("keep-warm-status", map[string]interface{}{
						"at":     time.Now().Format(time.RFC3339),
						"models": statuses,
					})
				}
				select {
				case <-stop:
					return
				case <-time.After(keepWarmInterval):
				}
			}
		}()
	})
}

// keepWarmRound 执行一轮保活：取 catalog 中 Running 的模型逐一轻量探针，
// 返回 model → "ok"/"fail" 状态（开关关闭/目录不可用/未配置时返回空 = 整轮跳过）。
func (a *App) keepWarmRound() map[string]string {
	statuses := map[string]string{}
	if a == nil || a.core == nil || a.cfg == nil {
		return statuses
	}
	// 开关关闭：整轮跳过（不查目录、不发探针）。
	if !a.cfg.GetKeepWarm() {
		return statuses
	}
	if a.engineMgr == nil {
		return statuses
	}
	base := a.herdsmanBaseURL()
	if base == "" {
		slog.Warn("keep-warm: Herdsman 引擎未配置，跳过本轮")
		return statuses
	}
	catalog, err := a.HerdsmanModelCatalog()
	if err != nil {
		slog.Warn("keep-warm: 模型目录不可用，跳过本轮", "error", err)
		return statuses
	}
	for _, m := range catalog.Models {
		if !m.Running {
			continue // local_concurrency=1：只探已运行的模型，绝不主动 start
		}
		ctx, cancel := context.WithTimeout(context.Background(), keepWarmProbeTimeout)
		err := keepWarmProbe(ctx, base, m.Name)
		cancel()
		if err != nil {
			// 模型已卸载/服务不可用：跳过，待 catalog 重新显示 Running 再探。
			slog.Info("keep-warm: 模型探针失败（本轮跳过）", "model", m.Name, "error", err)
			statuses[m.Name] = "fail"
			continue
		}
		a.setLastKeepAlive(m.Name)
		statuses[m.Name] = "ok"
	}
	if len(statuses) > 0 {
		slog.Info("keep-warm: 本轮完成", "probed", len(statuses))
	}
	return statuses
}

// setLastKeepAlive 记录模型最近一次成功探针时间（内存态，重启即失）。
func (a *App) setLastKeepAlive(model string) {
	if a.officeState == nil {
		return
	}
	a.officeState.keepAliveMu.Lock()
	defer a.officeState.keepAliveMu.Unlock()
	if a.officeState.keepAliveAt == nil {
		a.officeState.keepAliveAt = map[string]string{}
	}
	a.officeState.keepAliveAt[model] = time.Now().Format(time.RFC3339)
}

// lastKeepAlive 返回模型最近一次成功探针时间（空 = 本轮之前从未成功）。
func (a *App) lastKeepAlive(model string) string {
	if a.officeState == nil {
		return ""
	}
	a.officeState.keepAliveMu.RLock()
	defer a.officeState.keepAliveMu.RUnlock()
	return a.officeState.keepAliveAt[model]
}

// ─── T5-3b 启动自动预载 ────────────────────────────────────

// preloadDelay 预载等待：Startup 后延迟 10s 执行（等引擎/模型列表就绪）。
const preloadDelay = 10 * time.Second

// preloadFeatureOrder 预载选择优先级：gaea → office → chat（whisper 并入 chat）。
var preloadFeatureOrder = []string{"gaea", "office", "chat"}

// preloadTarget 按优先级（gaea→office→chat）选第一个引擎为 herdsman 的功能绑定
// 模型。所有绑定都非 herdsman（或未绑定）时 ok=false。
func preloadTarget(cfg *config.Config) (model string, ok bool) {
	if cfg == nil {
		return "", false
	}
	for _, feature := range preloadFeatureOrder {
		eng, mdl := cfg.GetFeatureModel(feature)
		if eng == "herdsman" && mdl != "" {
			return mdl, true
		}
	}
	return "", false
}

// autoPreloadDecision 判断目标模型是否应预载（纯逻辑，便于单测）：
// 返回 (shouldStart, reason)；reason 供日志说明跳过原因。
func autoPreloadDecision(catalog HerdsmanCatalog, model string) (bool, string) {
	if model == "" {
		return false, "未绑定 herdsman 模型"
	}
	for _, m := range catalog.Models {
		if m.Name != model {
			continue
		}
		if m.Running {
			return false, "已在运行，无需预载"
		}
		if !m.Installed {
			return false, "模型未安装，需先下载"
		}
		return true, "已安装且未运行"
	}
	return false, "catalog 中未找到该模型"
}

// autoPreloadStartModel 预载启动动作（可注入测试；默认走 HerdsmanModelStart
// --wait，等冷启动完成）。在后台 goroutine 中调用，不阻塞启动流程。
var autoPreloadStartModel = func(a *App, model string) error {
	res, err := a.HerdsmanModelStart(model)
	if err != nil {
		return err
	}
	slog.Info("auto-preload: 预载完成", "model", model, "status", res.Status)
	return nil
}

// startAutoPreload 启动自动预载（幂等，Startup 装配）：延迟 preloadDelay 后
// 在后台执行一轮预载。
func (a *App) startAutoPreload() {
	a.officeState.preloadOnce.Do(func() {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("auto-preload panic recovered", "panic", r)
				}
			}()
			time.Sleep(preloadDelay)
			a.runAutoPreload()
			// MH4（蒸馏 unsloth §三/§四）：modelhub 常驻预热，与 herdsman 预载
			// 同一延迟节拍、同一总闸、显存互斥（见 runModelHubPreload）。
			a.runModelHubPreload()
		}()
	})
}

// runAutoPreload 执行一轮自动预载（可直接调用测试）：
// 开关关闭/无 herdsman 绑定/目录不可用/非「已安装且未运行」→ 跳过；
// 命中 → 后台 goroutine 调 HerdsmanModelStart 等冷启动完成（只预载一个）。
func (a *App) runAutoPreload() {
	if a == nil || a.core == nil || a.cfg == nil || a.engineMgr == nil {
		return
	}
	if !a.cfg.GetAutoPreload() {
		slog.Info("auto-preload: 开关关闭，跳过")
		return
	}
	model, ok := preloadTarget(a.cfg)
	if !ok {
		slog.Info("auto-preload: 无 herdsman 功能绑定模型，跳过")
		return
	}
	catalog, err := a.HerdsmanModelCatalog()
	if err != nil {
		slog.Warn("auto-preload: 模型目录不可用，跳过", "model", model, "error", err)
		return
	}
	shouldStart, reason := autoPreloadDecision(catalog, model)
	if !shouldStart {
		slog.Info("auto-preload: 跳过预载", "model", model, "reason", reason)
		return
	}
	slog.Info("auto-preload: 开始预载", "model", model)
	// 后台 goroutine：--wait 等冷启动完成，不阻塞启动流程。
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("auto-preload start panic recovered", "panic", r)
			}
		}()
		if err := autoPreloadStartModel(a, model); err != nil {
			slog.Warn("auto-preload: 预载失败", "model", model, "error", err)
		}
	}()
}

// ─── MH4 Model Hub 常驻预热（蒸馏 unsloth，规划 §三/§四） ───────────

// modelHubPreloadArmed 运行态武装位：仅真实 App 生命周期（Startup）置位。
// 按项目纪律禁用「cfg != nil 即运行态」的惯性闸——包内其他测试初始化全局
// 配置会让惯性闸恒真（imageHubRuntimeArmed 同款先例）。
var modelHubPreloadArmed atomic.Bool

// modelHubPreloadWaitTimeout 预热等待 Studio 冷加载完成的收敛上限（§七实测
// 冷加载 50s+、内存挤占可超 2min，给足 6 分钟；含 StartModelHubModel 本身）。
const modelHubPreloadWaitTimeout = 6 * time.Minute

// modelHubPreloadPollInterval 预热收敛轮询间隔（var 供测试注入缩短）。
var modelHubPreloadPollInterval = 5 * time.Second

// runModelHubPreload 一轮 modelhub 预热（后台，静默降级不阻塞启动）：
// 武装位 → auto_preload 总闸 → 活跃引擎跟随（只预热活跃引擎就是 modelhub 的
// 场景，规划 §四-4 单卡显存互斥）→ 引擎启用+钉选默认模型+Key 已配置 →
// herdsman 预载将在途时让路（两个大模型同入统一内存会互相挤占）→
// /v1/models 已含目标即跳过（force_reload 幂等语义未证实，从不重复 load，
// §四-2）→ StartModelHubModel → 轮询收敛到 loaded，超时只记日志。
func (a *App) runModelHubPreload() {
	if !modelHubPreloadArmed.Load() {
		return
	}
	if a == nil || a.cfg == nil || a.engineMgr == nil || a.client == nil {
		return
	}
	if !a.cfg.GetAutoPreload() {
		return // 与 herdsman 预载同一总闸：用户关掉启动预载时不该被 modelhub 绕过
	}
	if a.client.ActiveEngineID() != "modelhub" {
		return
	}
	engine, ok := a.engineMgr.GetEngine("modelhub")
	if !ok || !engine.Enabled || engine.DefaultModel == "" {
		return
	}
	if !a.engineMgr.ModelHubKeyConfigured() {
		return
	}
	// 显存互斥：herdsman 预载将启动大模型时让路（catalog 判定纯读，失败不阻塞让路逻辑）
	if model, ok := preloadTarget(a.cfg); ok {
		if catalog, err := a.HerdsmanModelCatalog(); err == nil {
			if shouldStart, reason := autoPreloadDecision(catalog, model); shouldStart {
				slog.Info("modelhub-preload: herdsman 预载将在途，避免显存互斥，跳过", "herdsman_model", model, "reason", reason)
				return
			}
		}
	}
	target := engine.DefaultModel
	ctx, cancel := context.WithTimeout(context.Background(), modelHubPreloadWaitTimeout)
	defer cancel()
	loaded, err := a.engineMgr.ModelHubModelLoaded(ctx, target)
	if err != nil {
		// Studio 未启动/不可达：静默降级（预热是尽力而为，绝不报错打扰）
		slog.Info("modelhub-preload: Studio 不可达，跳过预热", "error", err)
		return
	}
	if loaded {
		slog.Info("modelhub-preload: 默认模型已加载，跳过", "model", target)
		return
	}
	slog.Info("modelhub-preload: 开始预热", "model", target)
	if err := a.engineMgr.StartModelHubModel(ctx, target); err != nil {
		slog.Warn("modelhub-preload: 预热请求失败（静默降级）", "model", target, "error", err)
		return
	}
	for {
		select {
		case <-ctx.Done():
			slog.Warn("modelhub-preload: 等待加载收敛超时", "model", target)
			return
		case <-time.After(modelHubPreloadPollInterval):
		}
		if ok, err := a.engineMgr.ModelHubModelLoaded(ctx, target); err == nil && ok {
			slog.Info("modelhub-preload: 预热完成", "model", target)
			return
		}
	}
}

// ─── T5-3c 换模预计等待 ────────────────────────────────────

// ModelSwitchEstimate 换模预计等待（T5-3c）：切换前给用户一个
// 热切换/冷启动/需下载/未知 的状态与预计等待，避免盲目点击后干等。
type ModelSwitchEstimate struct {
	Engine      string `json:"engine"`
	Model       string `json:"model"`
	Status      string `json:"status"`       // "hot" | "cold" | "download" | "unknown"
	WaitSeconds int    `json:"wait_seconds"` // 预计等待秒数（unknown/download 为 0）
	Note        string `json:"note"`
}

// estimateModelSwitch 纯函数：按安装/运行状态估切换档位（便于单测全分支）。
//   - running → hot：已在运行，热切换（1s）；
//   - installed && !running → cold：需冷启动，实测约 15.2s，取整 20s 上浮；
//   - !installed → download：未安装，需先下载（等待时间未知，给 0 由前端引导）。
func estimateModelSwitch(installed, running bool) (status string, waitSeconds int) {
	if running {
		return "hot", 1
	}
	if installed {
		return "cold", 20
	}
	return "download", 0
}

// ollamaProbeTimeout Ollama 原生 API 探测超时（切换器交互路径，宁短勿卡）。
const ollamaProbeTimeout = 3 * time.Second

// estimateUnknown 填 unknown 档（共享文案，调用方按语境可覆盖）。
func estimateUnknown(out ModelSwitchEstimate, note string) ModelSwitchEstimate {
	out.Status = "unknown"
	out.WaitSeconds = 0
	out.Note = note
	return out
}

// GaeaModelSwitchEstimate 换模预计等待。v4.126 刀2 起按「目标模型」口径覆盖全部
// 本地引擎（旧版非 herdsman 恒 hot、herdsman 只查引擎默认模型——ollama/modelhub
// 未加载模型切过去明明要冷启动却报「引擎已就绪」）：云端引擎常驻恒 hot；
// 本地引擎查目标 model 的加载态——herdsman 读模型目录（Installed/Running）、
// ollama 读原生 /api/ps（已加载）与 /api/tags（已安装）、modelhub 探测 Studio
// /v1/models；服务不可达如实 unknown（宁未知勿假 hot）。model 为空或 "(默认)"
// 回退引擎默认模型（兼容旧调用）。
func (a *App) GaeaModelSwitchEstimate(engineID, model string) ModelSwitchEstimate {
	out := ModelSwitchEstimate{Engine: engineID, Model: model}
	switch engineID {
	case "herdsman", "ollama", "modelhub":
		// 本地引擎：往下按目标模型查加载态
	default:
		out.Status = "hot"
		out.WaitSeconds = 1
		out.Note = "引擎已就绪"
		return out
	}
	if a == nil || a.engineMgr == nil {
		return estimateUnknown(out, "无法确认模型状态，切换后可能需等待")
	}
	eng, ok := a.engineMgr.GetEngine(engineID)
	if !ok {
		return estimateUnknown(out, "无法确认模型状态，切换后可能需等待")
	}
	if model == "" || model == "(默认)" {
		model = eng.DefaultModel
		out.Model = model
	}
	if model == "" {
		return estimateUnknown(out, "引擎未设默认模型，无法预估；请选择具体模型")
	}
	switch engineID {
	case "herdsman":
		return a.estimateHerdsmanModelSwitch(model, out)
	case "ollama":
		return a.estimateOllamaModelSwitch(eng, model, out)
	default:
		return a.estimateModelHubModelSwitch(model, out)
	}
}

// estimateHerdsmanModelSwitch 按目标模型查 herdsman 目录（Installed/Running）估档位。
func (a *App) estimateHerdsmanModelSwitch(model string, out ModelSwitchEstimate) ModelSwitchEstimate {
	catalog, err := a.HerdsmanModelCatalog()
	if err != nil {
		return estimateUnknown(out, "无法确认模型状态，切换后可能需等待")
	}
	installed, running := false, false
	for _, m := range catalog.Models {
		if m.Name == model {
			installed, running = m.Installed, m.Running
			break
		}
	}
	status, wait := estimateModelSwitch(installed, running)
	out.Status, out.WaitSeconds = status, wait
	switch status {
	case "hot":
		out.Note = "模型已运行，可直接切换"
	case "cold":
		out.Note = "本地模型需冷启动，实测约 15-20 秒"
	case "download":
		out.Note = "模型未安装，需先下载（模型中心模型库可下载）"
	}
	return out
}

// estimateOllamaModelSwitch 按目标模型查 Ollama 原生 API 估档位：
// /api/ps=已加载（hot）、/api/tags=已安装（未加载→cold，加载时长随模型大小差异
// 大，不假报秒数）、都不在=未安装（download 引导 ollama pull）。/api/ps 不可达但
// /api/tags 可达时按已安装未加载保守估 cold；两路都不可达如实 unknown。
func (a *App) estimateOllamaModelSwitch(eng *modelengine.EngineConfig, model string, out ModelSwitchEstimate) ModelSwitchEstimate {
	root := ollamaNativeRoot(eng.BaseURL)
	ctx, cancel := context.WithTimeout(context.Background(), ollamaProbeTimeout)
	defer cancel()
	loaded, loadedErr := fetchOllamaModelNames(ctx, root, "/api/ps")
	installed, instErr := fetchOllamaModelNames(ctx, root, "/api/tags")
	if loadedErr != nil && instErr != nil {
		return estimateUnknown(out, "无法连接 Ollama，无法确认模型状态")
	}
	inLoaded := loadedErr == nil && ollamaHasModel(loaded, model)
	inInstalled := instErr == nil && ollamaHasModel(installed, model)
	switch {
	case inLoaded:
		out.Status, out.WaitSeconds, out.Note = "hot", 1, "模型已加载，可直接切换"
	case inInstalled:
		out.Status, out.Note = "cold", "模型已安装未加载，切换后首次对话需等待加载"
	case instErr == nil:
		// 安装清单可达且无此模型：未安装（加载列表是否可达不影响结论）。
		out.Status, out.Note = "download", "Ollama 未安装该模型，需先 ollama pull 下载"
	case loadedErr == nil:
		// 加载列表可达但模型不在其中、安装清单不可达：未加载属实，是否已安装
		// 未知，保守按需加载估。
		out.Status, out.Note = "cold", "模型已安装未加载，切换后首次对话需等待加载"
	}
	return out
}

// estimateModelHubModelSwitch 探测 Studio /v1/models 是否已含目标模型（已加载
// →hot；未加载→cold 不报秒数——大 GGUF 加载以分钟计，模型中心 Model Hub 卡
// 可一键加载）；探测失败如实 unknown。
func (a *App) estimateModelHubModelSwitch(model string, out ModelSwitchEstimate) ModelSwitchEstimate {
	ctx, cancel := context.WithTimeout(context.Background(), ollamaProbeTimeout)
	defer cancel()
	loaded, err := a.engineMgr.ModelHubModelLoaded(ctx, model)
	if err != nil {
		return estimateUnknown(out, "无法连接 Model Hub，无法确认模型状态")
	}
	if loaded {
		out.Status, out.WaitSeconds, out.Note = "hot", 1, "模型已加载，可直接切换"
		return out
	}
	out.Status, out.Note = "cold", "模型未加载，切换后首次对话需等待加载（模型中心 Model Hub 卡可一键加载）"
	return out
}

// ollamaNativeRoot 由 OpenAI 兼容 BaseURL（…/v1）推 Ollama 原生 API 根地址。
func ollamaNativeRoot(baseURL string) string {
	return strings.TrimSuffix(strings.TrimRight(strings.TrimSpace(baseURL), "/"), "/v1")
}

// fetchOllamaModelNames 拉 Ollama 原生 API 模型名集合，名字去掉 ":latest" 后缀
// 归一（/v1/models 与原生 API 同名，但显式 tag 与 latest 缺省写法需互通）。
func fetchOllamaModelNames(ctx context.Context, root, path string) (map[string]bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, root+path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := ollamaProbeClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, path)
	}
	var doc struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(doc.Models))
	for _, m := range doc.Models {
		set[strings.TrimSuffix(m.Name, ":latest")] = true
	}
	return set, nil
}

// ollamaHasModel 目标模型是否在集合中（":latest" 归一后比对）。
func ollamaHasModel(set map[string]bool, model string) bool {
	return set[strings.TrimSuffix(model, ":latest")]
}

// ─── 绑定：保活/预载开关（T5-3a/b，持久化到 ~/.gaea_config.json） ──

// GetKeepWarm 读取本地模型保活开关（默认开启）。
func (c *core) GetKeepWarm() bool {
	if c == nil || c.cfg == nil {
		return true // 配置缺失时按默认开启处理
	}
	return c.cfg.GetKeepWarm()
}

// SetKeepWarm 设置本地模型保活开关并持久化（keep_warm_enabled）。
func (c *core) SetKeepWarm(enabled bool) error {
	if c == nil || c.cfg == nil {
		return errors.New("配置未初始化")
	}
	c.cfg.SetKeepWarm(enabled)
	if err := config.Save(config.KeyKeepWarm, strconv.FormatBool(enabled)); err != nil {
		slog.Warn("保存保活开关失败", "error", err)
		return err
	}
	slog.Info("本地模型保活开关已更新", "enabled", enabled)
	return nil
}

// GetPreloadPlan 读取启动自动预载开关（默认开启）。
func (c *core) GetPreloadPlan() bool {
	if c == nil || c.cfg == nil {
		return true // 配置缺失时按默认开启处理
	}
	return c.cfg.GetAutoPreload()
}

// SetPreloadPlan 设置启动自动预载开关并持久化（auto_preload）。
func (c *core) SetPreloadPlan(enabled bool) error {
	if c == nil || c.cfg == nil {
		return errors.New("配置未初始化")
	}
	c.cfg.SetAutoPreload(enabled)
	if err := config.Save(config.KeyAutoPreload, strconv.FormatBool(enabled)); err != nil {
		slog.Warn("保存自动预载开关失败", "error", err)
		return err
	}
	slog.Info("启动自动预载开关已更新", "enabled", enabled)
	return nil
}
