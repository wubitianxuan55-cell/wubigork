package app

// runtime_monitor.go — 运行状况监测快照（「运行监测」板块数据源）。
//
// 信息结构参照 Strata Monitor 仪表盘（用户 2026-10-19 参考图），数据全部取
// gaea 自己的口径：
//   - system：本机 CPU/内存/显存（mediaState.GetSystemStats 既有采集；AMD
//     桌面在 ComfyUI 未运行时 GPU 遥测可能为空——如实为空，前端显示「无遥测」
//     而非编造数值）；
//   - engines：模型中心引擎表（启用/连通/延迟/默认模型 + herdsman 侧模型级
//     running 标记），复用 engineMgr 内存态，零额外探测；
//   - strata/tianxuan：引擎启用且配置了地址时代理其 /metrics（live 状态、
//     tok/s、槽位、最近请求、硬件分钟级序列——Strata 系引擎自带的遥测面，
//     天璇同款）；引擎停机/拉取失败为 null，不拖慢快照（1.5s 超时并发拉取）；
//   - recentCalls：gaea 侧全部引擎（含云端）最近调用记账（modelengine 环形
//     缓冲，30 条新→旧）；
//   - totals：聚合计账标量汇总（总调用/成败/Token/费用）。
//
// 前端 2s 轮询（页面不可见时 usePollingGate 治理停拉）。/metrics 原样透传
// （map 形状由 Strata 系服务端定义，前端按宽松契约取用——服务端字段演进不
// 编进 Go 结构体）。

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/modelengine"
)

// runtimeMetricsTimeout 引擎 /metrics 拉取超时：监测快照 2s 一拍，单引擎
// 遥测最多占 1.5s（停机引擎连接拒收立即返回，不占满）。
const runtimeMetricsTimeout = 1500 * time.Millisecond

// RuntimeMonitor 聚合运行监测快照（Wails 绑定挂 ModelB；零值 App 安全——
// 测试裸 &App{} 时 system/engines 给空壳不 panic）。
func (a *App) RuntimeMonitor() map[string]interface{} {
	out := map[string]interface{}{
		"time": time.Now().Format("2006-01-02 15:04:05"),
	}

	// 系统资源：mediaState 既有采集（CPU/内存 kernel32，GPU 在无 ComfyUI 时
	// 可能拿不到——如实透出）。
	if a != nil && a.mediaState != nil {
		out["system"] = a.mediaState.GetSystemStats()
	} else {
		out["system"] = map[string]interface{}{}
	}

	// 引擎表：内存态直读（GetEngines 已清 Key、补空清单）。App 指针嵌入
	// *core/*mediaState——零值 App（测试裸 &App{}）访问提升字段前必须判嵌入
	// 指针非 nil，否则 nil 解引用 panic。
	c := a.core
	engines := make([]map[string]interface{}, 0, 12)
	if c != nil && c.engineMgr != nil {
		for _, e := range c.engineMgr.GetEngines() {
			row := map[string]interface{}{
				"id":            e.ID,
				"name":          e.Name,
				"type":          string(e.Type),
				"local":         e.Type.IsLocal(),
				"enabled":       e.Enabled,
				"connected":     e.Status.Connected,
				"latency_ms":    e.Status.LatencyMs,
				"last_checked":  e.Status.LastChecked,
				"error":         e.Status.Error,
				"default_model": e.DefaultModel,
				"base_url":      e.BaseURL,
			}
			// 模型级清单：herdsman 侧带 running/stopped 状态（识图/语音等
			// 辅助模型是否在线一屏可见）；其余引擎清单只作展示。
			models := make([]map[string]interface{}, 0, len(e.Models))
			for _, m := range e.Models {
				models = append(models, map[string]interface{}{
					"id":     m.ID,
					"kind":   m.Kind,
					"status": m.Status,
				})
			}
			row["models"] = models
			engines = append(engines, row)
		}
	}
	out["engines"] = engines

	// Strata 系引擎 /metrics 代理：仅启用且有地址的本地引擎，停机/失败为 null。
	// 结果经 channel 回收后统一写入（两 goroutine 同写 map = 数据竞争）。
	if c != nil && c.engineMgr != nil {
		type metricsResult struct {
			id      string
			payload map[string]interface{}
		}
		results := make(chan metricsResult, 2)
		pending := 0
		for _, id := range []string{"strata", "tianxuan"} {
			eng, ok := c.engineMgr.GetEngine(id)
			if !ok || !eng.Enabled || !strings.HasPrefix(eng.BaseURL, "http") {
				out[id] = nil
				continue
			}
			pending++
			go func(id, base string) {
				results <- metricsResult{id: id, payload: fetchEngineMetrics(base)}
			}(id, eng.BaseURL)
		}
		for ; pending > 0; pending-- {
			r := <-results
			if r.payload == nil {
				out[r.id] = nil // 归一无类型 nil：带类型的 nil map 会让 Go 侧类型断言误判「有遥测」
			} else {
				out[r.id] = r.payload
			}
		}
	} else {
		out["strata"], out["tianxuan"] = nil, nil
	}

	// gaea 侧最近调用（全引擎，含云端）与聚合计账标量。
	if c != nil && c.engineMgr != nil {
		out["recent_calls"] = c.engineMgr.RecentCalls(30)
		sum := c.engineMgr.GetModelCallStats()
		out["totals"] = map[string]interface{}{
			"total_calls":   sum.TotalCalls,
			"success_calls": sum.SuccessCalls,
			"fail_calls":    sum.FailCalls,
			"total_tokens":  sum.TotalTokens,
			"input_tokens":  sum.InputTokens,
			"output_tokens": sum.OutputTokens,
			"total_cost":    sum.TotalCost,
			"since":         sum.Since,
		}
	} else {
		out["recent_calls"] = []modelengine.ModelCallUsage{}
		out["totals"] = map[string]interface{}{}
	}
	return out
}

// runtimeMetricsClient /metrics 拉取客户端缝（测试注入 httptest 地址仍走真实
// HTTP，超时可注短；生产固定 1.5s）。
var runtimeMetricsHTTP = &http.Client{Timeout: runtimeMetricsTimeout}

// fetchEngineMetrics 拉取 Strata 系引擎的 /metrics JSON（原样透传 map）。
// 引擎停机/非 200/解码失败返回 nil（调用方渲染为「引擎未运行」）。
func fetchEngineMetrics(baseURL string) map[string]interface{} {
	url := strings.TrimRight(baseURL, "/") + "/metrics"
	ctx, cancel := context.WithTimeout(context.Background(), runtimeMetricsTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	resp, err := runtimeMetricsHTTP.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	var m map[string]interface{}
	if json.Unmarshal(body, &m) != nil {
		return nil
	}
	return m
}
