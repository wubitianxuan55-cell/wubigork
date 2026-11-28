package modelengine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/netclient"
)

// ── 引擎健康巡检 + 故障转移候选（C 刀 v0）─────────────────────
//
// 健康巡检：周期性轻量探测云端引擎连通性，只探 enabled 且非本地
// （EngineType.IsLocal()==false）的引擎（xai/deepseek/glm/opencode-*/custom-*）；
// 本地引擎（llama/Strata 等）由保活机制（keep-warm）管理，巡检跳过。
//
// 例外是 Herdsman：它是本地的检索模型网关（向量 / 精排），「连通」只说明网关在——
// 2026-11-27 它的向量模型进程显卡设备丢失后，/models 照常列 running、进程不退，
// 语义检索静默失效了 20 小时。所以巡检与「测试连接」在连通之后，会对在运行的向量
// 模型与精排模型各发一次真实小请求；坏了的写进 Status.Error 报警示（连通性仍是 true：
// 聊天不受影响，不该触发故障转移）。
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
	// localProbeTimeout 本地检索模型一次真实小请求的超时。模型若刚被卸载，Herdsman 会
	// 按需重拉，本机实测冷启动 4~6 秒；与 HTTP 客户端自身的超时一致。
	localProbeTimeout = 15 * time.Second
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

// probeRound 执行一轮巡检：快照本轮目标引擎（enabled 的云端引擎 + Herdsman）后逐个
// 探测（探测在锁外进行，不阻塞其他引擎操作）。
func (m *Manager) probeRound(ctx context.Context) {
	m.mu.RLock()
	targets := make([]EngineConfig, 0, len(m.engines))
	for _, id := range m.order {
		e, ok := m.engines[id]
		if !ok || !e.Enabled {
			continue
		}
		// 本地引擎只有 Herdsman 进巡检（见文件头：它承载检索模型，连通不等于可用）。
		if e.Type.IsLocal() && e.Type != EngineHerdsman {
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
// 达到阈值时加严重度前缀；Herdsman 连通后再对在运行的向量 / 精排模型各发一次真实
// 小请求，坏了的写进 Error（连通性不变）。状态相对上次变化时经回调通知 app 层。
func (m *Manager) probeOne(ctx context.Context, e EngineConfig) {
	budget := healthProbeTimeout
	if e.Type == EngineHerdsman {
		budget += 2 * localProbeTimeout // 连通之外还要发两次真实小请求（可能撞上模型冷启动）
	}
	pctx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()

	key := m.probeAuthKey(&e)
	start := time.Now()
	connected, modelCount, probeErr := m.probeEndpoint(pctx, &e)
	latency := time.Since(start).Milliseconds() // 只算连通那次，真实小请求的耗时不混进来

	errMsg := ""
	if probeErr != nil {
		errMsg = sanitizeProbeError(key, probeErr.Error())
	} else if e.Type == EngineHerdsman {
		// 模型清单现取，不拿缓存里的旧状态——照旧清单去探，会把已经卸载的模型强行拉起来。
		if models, ferr := m.fetchModels(pctx, &e); ferr == nil {
			if problems := m.probeLocalRetrieval(ctx, &e, models); len(problems) > 0 {
				errMsg = strings.Join(problems, "；")
			}
		}
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
		// 连通但检索模型坏了：连通性照常为 true（聊天不受影响、不该触发故障转移），
		// 问题写进 Error 给界面报警示。
		status.Error = errMsg
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
	if !changed {
		return
	}
	// 检索模型的坏与好只在变化时记一行（巡检每 10 分钟一轮，不刷屏）。
	if connected && e.Type == EngineHerdsman {
		if status.Error != "" {
			slog.Warn("本地检索模型不可用", "engine", e.ID, "problems", status.Error)
		} else if prev.Error != "" {
			slog.Info("本地检索模型已恢复", "engine", e.ID)
		}
	}
	if notify != nil {
		notify(e.ID, status.Connected)
	}
}

// retrievalKinds 要发真实小请求的检索模型类别（固定顺序；问题清单按它排）。
var retrievalKinds = []struct{ Kind, Label string }{
	{"embedding", "向量"},
	{"rerank", "精排"},
}

// probeLocalRetrieval 本地检索模型的真实探测：对清单里「在运行」的向量模型与精排
// 模型各发一次小请求，各取清单里第一个在运行的。/models 说 running 只代表进程在；
// 设备丢失后进程照样活着，只有真发一次请求才知道。没在运行的模型不探——用到时
// 会按需拉起，探它反而会把没在用的模型强行载入。返回问题清单（空 = 都正常）。
func (m *Manager) probeLocalRetrieval(ctx context.Context, e *EngineConfig, models []ModelInfo) []string {
	var problems []string
	for _, k := range retrievalKinds {
		id := firstRunningOfKind(models, k.Kind)
		if id == "" {
			continue
		}
		if err := m.probeLocalModel(ctx, e, k.Kind, id); err != nil {
			problems = append(problems, fmt.Sprintf("%s模型 %s 不可用：%v", k.Label, id, err))
		}
	}
	return problems
}

// firstRunningOfKind 清单里第一个在运行的某类模型（没有 = 空串）。
func firstRunningOfKind(models []ModelInfo, kind string) string {
	for _, mo := range models {
		if mo.Kind == kind && strings.EqualFold(strings.TrimSpace(mo.Status), "running") {
			return mo.ID
		}
	}
	return ""
}

// probeLocalModel 对一个检索模型发一次最小请求：向量 = 两个字的文本，要求回来非空
// 向量；精排 = 两个短句，要求 2xx。
func (m *Manager) probeLocalModel(ctx context.Context, e *EngineConfig, kind, id string) error {
	pctx, cancel := context.WithTimeout(ctx, localProbeTimeout)
	defer cancel()

	var path string
	var payload any
	switch kind {
	case "embedding":
		path, payload = "/embeddings", map[string]any{"model": id, "input": "造价"}
	default:
		path, payload = "/rerank", map[string]any{"model": id, "query": "造价", "documents": []string{"清单定额", "天气预报"}}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	baseURL := strings.TrimRight(strings.TrimSpace(e.BaseURL), "/")
	req, err := http.NewRequestWithContext(pctx, http.MethodPost, baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer netclient.DrainAndClose(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, errorBodyText(resp.Body))
	}
	if kind == "embedding" {
		var out struct {
			Data []struct {
				Embedding []float64 `json:"embedding"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || len(out.Data) == 0 || len(out.Data[0].Embedding) == 0 {
			return fmt.Errorf("响应里没有向量")
		}
	}
	return nil
}

// errorBodyText 把 HTTP 错误体压成一句：是 JSON 就取 message（本地引擎的 500 体例：
// {"error":{"code":500,"message":"decode() failed: …"}}），否则原文；都限长。
func errorBodyText(r io.Reader) string {
	raw, _ := io.ReadAll(io.LimitReader(r, 2048))
	s := strings.TrimSpace(string(raw))
	var body struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(raw, &body) == nil {
		if body.Error != nil && body.Error.Message != "" {
			s = body.Error.Message
		} else if body.Message != "" {
			s = body.Message
		}
	}
	if rs := []rune(s); len(rs) > 200 {
		s = string(rs[:200]) + "…"
	}
	return s
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
	defer netclient.DrainAndClose(resp.Body)
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
//   - default_model 非空且判型为 llm（排除 breeze 等 TTS/图像引擎）；
//   - default_model 在引擎自己的模型清单内（悬空默认直发必 404
//     model_not_found——2026-10-19 真机实锤：herdsman 换装后旧默认退役，
//     转移重试报 Model not found 且配置类错误不再二次转移，办公整板不可用。
//     清单为空=从未刷新，无从校验，维持旧口径放行）；
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
		if len(e.Models) > 0 && !modelsContainID(e.Models, e.DefaultModel) {
			continue
		}
		cfg := *e
		cfg.APIKey = ""
		out = append(out, cfg)
	}
	return out
}
