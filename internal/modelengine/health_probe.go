package modelengine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// ── 引擎健康巡检 + 故障转移候选（C 刀 v0）─────────────────────
//
// 健康巡检：周期性轻量探测云端引擎连通性，只探 enabled 且非本地
// （EngineType.IsLocal()==false）的引擎（xai/deepseek/glm/opencode-*/custom-*）；
// 本地引擎（ollama/herdsman/cosyvoice/modelhub）由保活机制（keep-warm）管理，巡检跳过。
//
// 探测口径：GET {BaseURL}/models（与 fetchModels 的 /models 端点同源拼法），
// 带按引擎的鉴权头（与 fetchModels/BuildChatURL 同一取 Key 链；custom 无 Key
// 不发 Authorization），单引擎超时 8s，HTTP 2xx = Connected。GLM 无 /models
// 端点，与 TestConnection 同源走 glmPing 最小 chat 探测。结果写引擎 Status
// （随 saveState 持久化）；状态相对上次变化时经 SetHealthNotifyFunc 注入的
// 回调通知 app 层 emit engine-health-changed（Manager 不直接依赖前端）。
//
// 连续失败计数只在内存（probeFails，重启清零不落盘）：连续 ≥3 次失败才在
// Status.Error 加「连续 N 次探测失败」前缀供前端展示严重度，成功即清零。
// 错误串经 sanitize：永不含 API Key（Authorization 头内容）——Status 会随
// 状态文件落盘并下发前端。

const (
	// healthProbeInitialDelay app 启动后首轮探测延迟。
	healthProbeInitialDelay = time.Minute
	// healthProbeInterval 巡检周期。
	healthProbeInterval = 10 * time.Minute
	// healthProbeTimeout 单引擎探测超时。
	healthProbeTimeout = 8 * time.Second
	// healthProbeFailThreshold 连续失败达到该次数才在 Status.Error 标注
	// 「连续 N 次探测失败」严重度前缀。
	healthProbeFailThreshold = 3
)

// SetHealthNotifyFunc 注入健康状态变化回调（app Startup 接线 → a.emit，
// 照 SetCustomEngineKeys 注入先例；Manager 不直接 emit）。nil 清除。
func (m *Manager) SetHealthNotifyFunc(fn func(id string, connected bool)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthNotify = fn
}

// StartHealthProbe 启动健康巡检循环（app Startup 调用，非绑定方法）：
// 首轮延迟 1 分钟，之后每 10 分钟一轮（time.Ticker）；ctx 取消或
// StopHealthProbe 退出。重复调用幂等（先停旧循环再起新循环，
// 照 StartGLMCatalogRemote 先例）。
func (m *Manager) StartHealthProbe(ctx context.Context) {
	m.mu.Lock()
	if m.healthStop != nil {
		close(m.healthStop)
	}
	stop := make(chan struct{})
	m.healthStop = stop
	if m.probeFails == nil {
		m.probeFails = make(map[string]int)
	}
	m.mu.Unlock()

	go func() {
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		case <-time.After(healthProbeInitialDelay):
		}
		m.probeRound(ctx)
		ticker := time.NewTicker(healthProbeInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-ticker.C:
				m.probeRound(ctx)
			}
		}
	}()
	slog.Info("引擎健康巡检已启动", "interval", healthProbeInterval.String())
}

// StopHealthProbe 停止健康巡检循环（app Shutdown 调用；未启动则空操作，幂等）。
func (m *Manager) StopHealthProbe() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.healthStop != nil {
		close(m.healthStop)
		m.healthStop = nil
	}
}

// probeRound 执行一轮巡检：快照本轮目标引擎（enabled 且非本地）后逐个探测
// （探测在锁外进行，不阻塞其他引擎操作）。
func (m *Manager) probeRound(ctx context.Context) {
	m.mu.RLock()
	targets := make([]EngineConfig, 0, len(m.engines))
	for _, id := range m.order {
		e, ok := m.engines[id]
		if !ok || !e.Enabled || e.Type.IsLocal() {
			continue
		}
		targets = append(targets, *e)
	}
	m.mu.RUnlock()

	for i := range targets {
		select {
		case <-ctx.Done():
			return
		default:
		}
		m.probeOne(ctx, targets[i])
	}
}

// probeOne 探测单个引擎并落 Status：2xx=Connected；失败计连续次数并在
// 达到阈值时加严重度前缀；状态相对上次变化时经回调通知 app 层。
func (m *Manager) probeOne(ctx context.Context, e EngineConfig) {
	pctx, cancel := context.WithTimeout(ctx, healthProbeTimeout)
	defer cancel()

	key := m.probeAuthKey(&e)
	start := time.Now()
	connected, modelCount, probeErr := m.probeEndpoint(pctx, &e)
	latency := time.Since(start).Milliseconds()

	errMsg := ""
	if probeErr != nil {
		errMsg = sanitizeProbeError(key, probeErr.Error())
	}

	m.mu.Lock()
	eng, ok := m.engines[e.ID]
	if !ok {
		// 巡检期间引擎被删除：丢弃本次结果
		m.mu.Unlock()
		return
	}
	prev := eng.Status
	status := EngineStatus{
		ID:          e.ID,
		Connected:   connected,
		ModelCount:  modelCount,
		LastChecked: time.Now().Format("2006-01-02 15:04:05"),
		LatencyMs:   latency,
	}
	if connected {
		delete(m.probeFails, e.ID) // 成功即清零
	} else {
		if m.probeFails == nil {
			m.probeFails = make(map[string]int)
		}
		m.probeFails[e.ID]++
		if n := m.probeFails[e.ID]; n >= healthProbeFailThreshold {
			errMsg = fmt.Sprintf("连续 %d 次探测失败：%s", n, errMsg)
		}
		status.Error = errMsg
	}
	changed := prev.Connected != status.Connected || prev.Error != status.Error
	eng.Status = status
	notify := m.healthNotify
	m.mu.Unlock()

	m.saveState()
	if changed && notify != nil {
		notify(e.ID, status.Connected)
	}
}

// probeEndpoint 轻量探测：GET {BaseURL}/models（GLM 走 glmPing），返回
// （连通性, 模型数, 错误）。2xx 即连通；ModelCount 顺带解析 /models JSON
// （坏 JSON 保持 0，不影响连通判定）。错误串不携带任何 Key。
func (m *Manager) probeEndpoint(ctx context.Context, e *EngineConfig) (connected bool, modelCount int, err error) {
	if e.Type == EngineGLM {
		// GLM 无 /models 端点：与 TestConnection 同源走最小 chat ping（只判连通）。
		if perr := m.glmPing(ctx, e); perr != nil {
			return false, 0, perr
		}
		return true, 0, nil
	}
	baseURL := strings.TrimRight(strings.TrimSpace(e.BaseURL), "/")
	if !validBaseURL(baseURL) {
		return false, 0, fmt.Errorf("引擎地址无效：需要 http:// 或 https:// 前缀")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return false, 0, fmt.Errorf("创建请求失败: %w", err)
	}
	if key := m.probeAuthKey(e); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return false, 0, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return false, 0, fmt.Errorf("HTTP %d: 健康探测失败", resp.StatusCode)
	}
	var list modelsListResponse
	_ = json.NewDecoder(resp.Body).Decode(&list) // 坏 JSON → count 0，2xx 仍判连通
	return true, len(list.Data), nil
}

// probeAuthKey 按引擎类型取探测用 API Key（与 fetchModels/BuildChatURL 同一
// 取 Key 链）；custom 引擎 Key 在内存 customKeys，空 Key 不带 Authorization。
func (m *Manager) probeAuthKey(e *EngineConfig) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	switch e.Type {
	case EngineXAI:
		return m.xaiKey
	case EngineDeepseek:
		return m.deepseekKey
	case EngineGLM:
		return m.glmKey
	case EngineOpencodeGo:
		return m.opencodeKey
	case EngineOpencodeZen:
		return m.opencodeZenKey
	case EngineCustom:
		return m.customKeys[e.ID]
	}
	return ""
}

// sanitizeProbeError 巡检错误串脱敏：Status.Error 会随状态文件落盘并下发
// 前端，任何 Key（Authorization 头内容）出现都替换为 ****。
func sanitizeProbeError(key, msg string) string {
	if key != "" {
		msg = strings.ReplaceAll(msg, "Bearer "+key, "Bearer ****")
		msg = strings.ReplaceAll(msg, key, "****")
	}
	return msg
}

// ── 故障转移候选（C 刀 v0，重试逻辑在 ai.Client）────────────────

// FailoverCandidates 返回故障转移候选引擎快照（按 m.order 稳定顺序，活跃
// 引擎排最前的语义已隐含在 order 内）：
//   - enabled 且最近 Status.Connected（巡检/测试连接成功）；
//   - default_model 非空且判型为 llm（排除 cosyvoice 等 TTS/图像引擎）；
//   - 排除 failedID（本次失败引擎）。
//
// ai.Client 取首个候选，用其 default_model 换新 URL/新 Key 重试一次。
func (m *Manager) FailoverCandidates(failedID string) []EngineConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []EngineConfig
	for _, id := range m.order {
		if id == failedID {
			continue
		}
		e, ok := m.engines[id]
		if !ok || !e.Enabled || !e.Status.Connected {
			continue
		}
		if e.DefaultModel == "" || ClassifyModelKind(e.Type, e.DefaultModel) != "llm" {
			continue
		}
		cfg := *e
		cfg.APIKey = ""
		out = append(out, cfg)
	}
	return out
}
