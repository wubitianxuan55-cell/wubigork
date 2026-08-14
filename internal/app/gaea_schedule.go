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
	"time"

	"github.com/gaea/gaea/internal/config"
)

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

// GaeaModelSwitchEstimate 换模预计等待：
// 非 herdsman 引擎恒为 hot（引擎常驻，无需本机加载）；herdsman 读引擎
// DefaultModel，查模型目录定位 hot/cold/download；目录不可用 → unknown。
func (a *App) GaeaModelSwitchEstimate(engineID string) ModelSwitchEstimate {
	out := ModelSwitchEstimate{Engine: engineID}
	if engineID != "herdsman" {
		// 云端/常驻引擎无需本机加载，视为热切换。
		out.Status = "hot"
		out.WaitSeconds = 1
		out.Note = "引擎已就绪"
		return out
	}
	if a == nil || a.core == nil || a.engineMgr == nil {
		out.Status = "unknown"
		out.Note = "无法确认模型状态，切换后可能需等待"
		return out
	}
	eng, ok := a.engineMgr.GetEngine("herdsman")
	if !ok || eng.DefaultModel == "" {
		out.Status = "unknown"
		out.Note = "无法确认模型状态，切换后可能需等待"
		return out
	}
	model := eng.DefaultModel
	out.Model = model
	catalog, err := a.HerdsmanModelCatalog()
	if err != nil {
		out.Status = "unknown"
		out.Note = "无法确认模型状态，切换后可能需等待"
		return out
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
		out.Note = "引擎已就绪"
	case "cold":
		out.Note = "本地模型需冷启动，实测约 15-20 秒"
	case "download":
		out.Note = "模型未安装，需先下载（模型中心模型库可下载）"
	}
	return out
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
