package builtin

// websearch_core.go — 统一检索核（众神殿 2026-11-05 网络搜索刀1：编排与状态契约）。
// 办公 web_search 工具与聊天板块 adapter（internal/app/chat_search_adapter.go）
// 共用同一条引擎编排链：办公经 Execute（JSON 数组输出契约零变化），聊天经
// RunSearch（结构化结果 + 逐引擎诊断）。
//
// 刀1 语义：编排保持「首胜」现状；新增逐引擎 SearchAttempt 诊断，把「策略过滤
// 后空」从引擎失败里拆分出来（原先与「无结果」一起被当作 engineError{"no
// results"} 混入失败诊断，deny_domains 用户会看到误导性失败）。ClassifySearchState
// 归出顶层状态（三层失败契约第一层）；刀2 在本函数内改短窗合并+主源保留窗，
// DEGRADED 状态届时启用，本批只预留常量位。

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// SearchState 统一检索核顶层状态（三层失败契约第一层：返回值枚举）。
type SearchState string

const (
	SearchStateOK        SearchState = "ok"
	SearchStateDegraded  SearchState = "degraded" // 刀2 主源保留窗启用（primary-timeout|primary-error），本批不产出
	SearchStateEmpty     SearchState = "empty"
	SearchStateAllFailed SearchState = "all_failed"
)

// 逐引擎尝试结局（SearchAttempt.Outcome）。「filtered-empty」是刀1 的关键拆分：
// 引擎返回了结果但被域名 allow/deny 策略滤空——检索本身发生了，不是引擎失败。
const (
	attemptOK            = "ok"
	attemptEmpty         = "empty"
	attemptFilteredEmpty = "filtered-empty"
	attemptError         = "error"
)

// SearchAttempt 一次引擎尝试的结局诊断（统一核 → 消费方的可观测契约）。
type SearchAttempt struct {
	Name      string `json:"name"`                // 引擎名（bing / tavily / …）
	Outcome   string `json:"outcome"`             // ok / empty / filtered-empty / error
	Err       string `json:"err,omitempty"`       // error 结局的原因摘要
	ElapsedMS int64  `json:"elapsedMs,omitempty"` // 该引擎耗时
}

// attemptDiagText 诊断文本：错误给原因，滤空与真空结果分开表述（刀1 拆分落字面）。
func attemptDiagText(a SearchAttempt) string {
	switch a.Outcome {
	case attemptEmpty:
		return "无结果"
	case attemptFilteredEmpty:
		return "策略过滤后无可用结果"
	default:
		if a.Err != "" {
			return a.Err
		}
		return "失败"
	}
}

// RunSearch 统一检索核：引擎扇出编排 + 逐引擎诊断。零 key 可用引擎集、域名
// 策略 over-fetch、首胜语义均与刀1 前 Execute 一致（办公工具行为零变化）。
// err 非 nil 表示无可用的成功结果（消费方按 ClassifySearchState 归状态）。
func RunSearch(ctx context.Context, query string, topK int) ([]SearchResult, []SearchAttempt, error) {
	return webSearch{}.runSearch(ctx, query, topK)
}

// ClassifySearchState 按结果与逐引擎诊断归顶层状态：有结果=OK；无结果但至少
// 一个引擎真正完成检索（含策略过滤后空——检索发生了，只是没有可用产出）=EMPTY；
// 全部传输/解析失败或零引擎=ALL_FAILED。
func ClassifySearchState(results []SearchResult, attempts []SearchAttempt) SearchState {
	if len(results) > 0 {
		return SearchStateOK
	}
	for _, a := range attempts {
		if a.Outcome == attemptEmpty || a.Outcome == attemptFilteredEmpty {
			return SearchStateEmpty
		}
	}
	return SearchStateAllFailed
}

// runSearch 引擎扇出编排：每个引擎一个 goroutine，首胜即返；每次尝试的结局
// 进 attemptCh（胜者也记，便于诊断）；全部落空时收集诊断并以 err 收口。
func (ws webSearch) runSearch(ctx context.Context, query string, topK int) ([]SearchResult, []SearchAttempt, error) {
	engines := ws.buildEngines()

	// 域名策略受限时（[search] allow_domains/deny_domains），向引擎请求更多结果再按
	// 策略过滤，避免“前几条被拒 → 结果为空”（unsloth web_access_policy over-fetch）。
	requested := topK
	if searchPolicyRestricted() {
		requested = topK * searchPolicyOverfetch
	}

	resultCh := make(chan []SearchResult, 1)
	attemptCh := make(chan SearchAttempt, len(engines))

	ctx, cancel := context.WithTimeout(ctx, webSearchTotalLimit)
	defer cancel()

	for _, eng := range engines {
		eng := eng
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("websearch: engine goroutine panic recovered", "engine", eng.Name(), "panic", r)
					attemptCh <- SearchAttempt{Name: eng.Name(), Outcome: attemptError, Err: fmt.Sprintf("engine panic: %v", r)}
				}
			}()
			start := time.Now()
			raw, err := eng.Search(ctx, query, requested)
			elapsed := time.Since(start)
			if err != nil {
				attemptCh <- SearchAttempt{Name: eng.Name(), Outcome: attemptError, Err: err.Error(), ElapsedMS: elapsed.Milliseconds()}
				return
			}
			// 应用域名 allow/deny 策略并截断到 topK；原始非空而过滤后为空记
			// filtered-empty（策略滤空≠引擎失败，刀1 拆分），原始即空记 empty。
			results := filterSearchResults(raw, topK)
			if len(results) == 0 {
				outcome := attemptEmpty
				if len(raw) > 0 {
					outcome = attemptFilteredEmpty
				}
				attemptCh <- SearchAttempt{Name: eng.Name(), Outcome: outcome, ElapsedMS: elapsed.Milliseconds()}
				return
			}
			// 胜者先发结果、后发尝试（程序序）：主循环收齐全部尝试时，胜者
			// 结果必然已入缓冲，收口处的非阻塞读取不会错过它。
			select {
			case resultCh <- results:
			default:
				// another engine already won, discard
			}
			attemptCh <- SearchAttempt{Name: eng.Name(), Outcome: attemptOK, ElapsedMS: elapsed.Milliseconds()}
		}()
	}

	attempts := make([]SearchAttempt, 0, len(engines))
	// 收集循环：每个引擎恰好发一条尝试；胜者额外先发一条结果。收到全部尝试后
	// 胜者的结果必然已入缓冲（均非阻塞缓冲写）。
	for len(attempts) < len(engines) {
		select {
		case results := <-resultCh:
			// 首胜即返：诊断只收已到账的尝试，迟到者随首胜语义丢弃（刀1 现状；
			// 刀2 改短窗合并时在此收齐窗口内的尝试）。
			won := drainAttempts(attemptCh, len(engines)-len(attempts))
			return results, append(attempts, won...), nil
		case a := <-attemptCh:
			attempts = append(attempts, a)
		case <-ctx.Done():
			// Timeout — drain any remaining attempts that arrive quickly.
			attempts = append(attempts, SearchAttempt{Name: "timeout", Outcome: attemptError, Err: ctx.Err().Error()})
			for j := len(attempts); j < len(engines); j++ {
				select {
				case a := <-attemptCh:
					attempts = append(attempts, a)
				case <-time.After(100 * time.Millisecond):
				}
			}
			return nil, attempts, ctx.Err()
		}
	}
	// 全部尝试已到账：胜者结果（如有）已在缓冲里，非阻塞取一次。
	select {
	case results := <-resultCh:
		return results, attempts, nil
	default:
		return nil, attempts, fmt.Errorf("all %d engines failed", len(engines))
	}
}

// drainAttempts 非阻塞收齐已到账的尝试（至多 max 条，不等待迟到者）。
func drainAttempts(ch chan SearchAttempt, max int) []SearchAttempt {
	out := make([]SearchAttempt, 0, max)
	for i := 0; i < max; i++ {
		select {
		case a := <-ch:
			out = append(out, a)
		default:
			return out
		}
	}
	return out
}

// buildAllFailedDiag 全败诊断：逐引擎结局 + 零配置指引（文案与指引位置均为
// 刀1 前现状，办公工具输出契约零变化；指引只在 [search] 三键全空时附上）。
func buildAllFailedDiag(attempts []SearchAttempt) string {
	var diag strings.Builder
	diag.WriteString("所有搜索引擎失败：")
	for _, f := range attempts {
		fmt.Fprintf(&diag, "\n  • %s (%dms): %s", f.Name, f.ElapsedMS, attemptDiagText(f))
	}
	if searchCfg == nil || (searchCfg.TavilyAPIKeyEnv == "" && searchCfg.BraveAPIKeyEnv == "" && searchCfg.LocalSearXNGURL == "") {
		diag.WriteString("\n\n💡 提示：配置搜索 API 可大幅提高成功率：")
		diag.WriteString("\n  1. Tavily（免费 1000次/月）：注册 tavily.com → 设环境变量 TAVILY_API_KEY")
		diag.WriteString("\n  2. Brave Search（免费 2000次/月）：注册 api.search.brave.com → 设环境变量 BRAVE_API_KEY")
		diag.WriteString("\n  3. 自建 SearXNG：docker run -d -p 8080:8080 searxng/searxng")
		diag.WriteString("\n  然后在 gaea.toml 中配置 [search] 节。")
	}
	return diag.String()
}
