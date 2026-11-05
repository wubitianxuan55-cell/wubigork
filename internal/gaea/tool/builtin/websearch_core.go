package builtin

// websearch_core.go — 统一检索核（众神殿 2026-11-05 网络搜索：刀1 编排与状态
// 契约 + 刀2 短窗合并/主源保留窗/覆盖门）。办公 web_search 工具与聊天板块
// adapter（internal/app/chat_search_adapter.go）共用同一条链：办公经 Execute
// （JSON 数组输出契约零变化），聊天经 RunSearch（结构化结果+状态+诊断）。
//
// 刀2 编排语义（替换刀1「首胜单槽」——基线实测 bing 672ms 先胜 5 条全离题 vs
// tavily 1122ms 5 条全正题被单槽丢弃）：
//   1. 主源保留窗：主源类引擎（local-searxng/tavily/brave，engine_order 中首个
//      可用者）在保留窗内就绪 → 以其结果为主、短窗合并 keyless 独有结果；
//      超窗/失败 → keyless 首胜+短窗兜底，状态强制 DEGRADED(primary-timeout|
//      primary-error)，禁止静默 bing-only 标 OK。
//   2. 短窗合并：首胜后开 searchMergeWindow 收齐已完成引擎，URL 规范化去重
//      （www./尾斜杠/utm_*），重复条目并入 Source（"a+b"）；单引擎行为等价。
//   3. 覆盖门 v0（websearch_coverage.go）：剥日期/数字/泛词后核心实体命中
//      title+snippet，不过 → DEGRADED(coverage-*)，不冒充可核验结果。
//   4. 每引擎 queried/won/elapsed 埋点：SearchAttempt（恒记）+ Result.Source。
//
// 窗口为 var（测试可缩短）；数值按 30 集 A 臂 elapsed 分布校准，聊天 8s 硬上限
// 在 adapter 层，本层只守办公 20s 总截止；per-engine 15s 不动。

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"
)

// SearchState 统一检索核顶层状态（三层失败契约第一层：返回值枚举）。
type SearchState string

const (
	SearchStateOK        SearchState = "ok"
	SearchStateDegraded  SearchState = "degraded"
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

// 刀2 编排窗口（var：测试可缩短；数值按基线校准）。
var (
	// searchPrimaryWindow 主源保留窗：主源在此窗内就绪则以其为主。
	// 校准（2026-11-05 30 集 A 臂实测）：tavily p50=2.4s / p90=2.9s / p95=3.2s →
	// 3.5s 覆盖 >95% 主源请求；聊天最坏 3.5s 窗 + keyless 首胜（bing p95≈1.1s）
	// ≈ 4.6s < adapter 8s 硬上限。基线档案 docs/web-search-baseline-30q-20261105.json。
	searchPrimaryWindow = 3500 * time.Millisecond
	// searchMergeWindow 首胜/主源就绪后的合并短窗：收齐已完成或快完成引擎。
	searchMergeWindow = 1200 * time.Millisecond
)

// searchPrimaryKinds 主源类别 kind 集合（var：测试注入桩 kind）。keyless 引擎
// （public-searxng/bing/ddg）永不为主源。
var searchPrimaryKinds = map[string]bool{
	SearchEngineKindLocalSearXNG: true,
	SearchEngineKindTavily:       true,
	SearchEngineKindBrave:        true,
}

func isPrimaryKind(kind string) bool { return searchPrimaryKinds[kind] }

// SearchAttempt 一次引擎尝试的结局诊断（统一核 → 消费方的可观测契约；
// queried/elapsed/outcome 逐引擎埋点，won 由 Result.Source 归因）。
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

// taggedResults 引擎产出随附引擎名（合并归因 + 主源识别）。
type taggedResults struct {
	name    string
	results []SearchResult
}

// SearchOutcome 统一检索核的结构化产出（刀2 起 RunSearch 返回）。
type SearchOutcome struct {
	Results        []SearchResult
	Attempts       []SearchAttempt
	State          SearchState
	DegradedReason string // primary-timeout / primary-error / coverage-*（OK/EMPTY/ALL_FAILED 时为空）
	Diagnostics    string // ALL_FAILED 时的全败诊断（含零配置指引）
}

// RunSearch 统一检索核：主源保留窗+短窗合并+覆盖门。err 非 nil 当且仅当无
// 可用结果（State==empty|all_failed，消息=Diagnostics；与办公工具刀1 前的
// 「空/全败即 error」输出契约一致）；有结果时 State 仍可为 degraded（降级标注）。
func RunSearch(ctx context.Context, query string, topK int) (*SearchOutcome, error) {
	return webSearch{}.runSearch(ctx, query, topK)
}

// ClassifySearchState 按结果与逐引擎诊断归顶层状态（结果缺失场景：EMPTY vs
// ALL_FAILED）；OK/DEGRADED 的终判在 runSearch 编排层（依赖主源与覆盖门上下文）。
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

// searchEngineRunner 引擎及其 kind（主源识别按 kind，不按引擎 Name）。
type searchEngineRunner struct {
	kind   string
	engine SearchEngine
}

// buildEngineRunners 按 config 驱动的 kind 顺序经注册表构建引擎（可用性过滤）。
// 顺序来源：[search] engine_order（SetSearchEngineOrder 注入），未配置时用默认序。
// 刀2 起顺序语义扩展：还决定合并时 keyless 结果的追加权重，以及主源选择
// （主源类 kind 中顺序首位且可用者）。
func (webSearch) buildEngineRunners() []searchEngineRunner {
	var runners []searchEngineRunner
	cfg := searchCfg // may be nil

	for _, kind := range searchEngineOrder() {
		var ecfg SearchEngineConfig
		switch kind {
		case SearchEngineKindLocalSearXNG:
			if cfg != nil {
				ecfg.BaseURL = cfg.LocalSearXNGURL
			}
		case SearchEngineKindTavily:
			if cfg != nil {
				ecfg.APIKey = cfg.TavilyKey()
			}
		case SearchEngineKindBrave:
			if cfg != nil {
				ecfg.APIKey = cfg.BraveKey()
			}
		}
		eng, err := NewSearchEngine(kind, ecfg)
		if err != nil {
			// 未知 kind（config 写了未注册引擎）：跳过并在诊断中提示，不中断搜索。
			slog.Warn("websearch: 跳过不可用搜索引擎", "kind", kind, "error", err)
			continue
		}
		if !eng.Available() {
			continue // 未配置凭据/URL 的引擎不参与扇出（等价旧分支）
		}
		runners = append(runners, searchEngineRunner{kind: kind, engine: eng})
	}
	return runners
}

// buildEngines 引擎列表视图（既有注册表测试消费；编排走 buildEngineRunners）。
func (ws webSearch) buildEngines() []SearchEngine {
	runners := ws.buildEngineRunners()
	out := make([]SearchEngine, 0, len(runners))
	for _, r := range runners {
		out = append(out, r.engine)
	}
	return out
}

// searchEngineOrder 返回生效的引擎顺序：config 注入序优先，否则默认序。
func searchEngineOrder() []string {
	if len(searchEngineOrderCfg) > 0 {
		return searchEngineOrderCfg
	}
	return defaultSearchEngineOrder
}

// runSearch 编排：引擎扇出 → 主源保留窗（或 keyless 首胜）→ 短窗合并 →
// 覆盖门 → 状态归集。
func (ws webSearch) runSearch(ctx context.Context, query string, topK int) (*SearchOutcome, error) {
	runners := ws.buildEngineRunners()

	// 域名策略受限时（[search] allow_domains/deny_domains），向引擎请求更多结果再按
	// 策略过滤，避免“前几条被拒 → 结果为空”（unsloth web_access_policy over-fetch）。
	requested := topK
	if searchPolicyRestricted() {
		requested = topK * searchPolicyOverfetch
	}

	attemptCh := make(chan SearchAttempt, len(runners))
	resultCh := make(chan taggedResults, len(runners))

	ctx, cancel := context.WithTimeout(ctx, webSearchTotalLimit)
	defer cancel()

	for _, r := range runners {
		r := r
		go func() {
			defer func() {
				if rec := recover(); rec != nil {
					slog.Error("websearch: engine goroutine panic recovered", "engine", r.engine.Name(), "panic", rec)
					attemptCh <- SearchAttempt{Name: r.engine.Name(), Outcome: attemptError, Err: fmt.Sprintf("engine panic: %v", rec)}
				}
			}()
			start := time.Now()
			raw, err := r.engine.Search(ctx, query, requested)
			elapsed := time.Since(start)
			if err != nil {
				attemptCh <- SearchAttempt{Name: r.engine.Name(), Outcome: attemptError, Err: err.Error(), ElapsedMS: elapsed.Milliseconds()}
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
				attemptCh <- SearchAttempt{Name: r.engine.Name(), Outcome: outcome, ElapsedMS: elapsed.Milliseconds()}
				return
			}
			// 胜者先发结果、后发尝试（程序序）：主循环收齐全部尝试时，已发结果
			// 必然已在缓冲（均非阻塞缓冲写）。
			resultCh <- taggedResults{name: r.engine.Name(), results: results}
			attemptCh <- SearchAttempt{Name: r.engine.Name(), Outcome: attemptOK, ElapsedMS: elapsed.Milliseconds()}
		}()
	}

	out := &SearchOutcome{Attempts: make([]SearchAttempt, 0, len(runners))}

	primaryName := ""
	for _, r := range runners {
		if isPrimaryKind(r.kind) {
			primaryName = r.engine.Name()
			break
		}
	}

	var extras []taggedResults   // keyless 产出（按到达序，合并时按引擎序加权）
	var primaryResults []SearchResult
	primaryDegraded := "" // ""=主源正常参与；否则 primary-timeout / primary-error

	if primaryName == "" {
		// 零主源（keyless 三件套）：首胜开短窗；单引擎时窗内无他源可收，行为等价旧首胜。
		first := waitFirstResult(out, len(runners), attemptCh, resultCh, ctx)
		if first == nil {
			return allFailedOutcome(out, attemptCh, "")
		}
		extras = append(extras, *first)
		extras = append(extras, collectMergeWindow(out, attemptCh, resultCh, ctx, nil)...)
	} else {
		// 主源保留窗：等主源就绪（结果到达 / 失败尝试 / 超窗）。
		timer := time.NewTimer(searchPrimaryWindow)
		primaryResolved := false
		for !primaryResolved {
			select {
			case tr := <-resultCh:
				if tr.name == primaryName {
					primaryResults = tr.results
					primaryResolved = true
				} else {
					extras = append(extras, tr) // keyless 结果先到：留着合并
				}
			case a := <-attemptCh:
				out.Attempts = append(out.Attempts, a)
				if a.Name != primaryName {
					continue
				}
				if a.Outcome != attemptOK {
					primaryDegraded = "primary-error"
					primaryResolved = true
					break
				}
				// 主源 ok 尝试到达：result 必已入缓冲（result 先发的程序序），非阻塞补收配对。
				select {
				case tr := <-resultCh:
					if tr.name == primaryName {
						primaryResults = tr.results
					} else {
						extras = append(extras, tr)
					}
				default:
				}
				primaryResolved = true
			case <-timer.C:
				primaryDegraded = "primary-timeout"
				primaryResolved = true
			case <-ctx.Done():
				return allFailedOutcome(out, attemptCh, "")
			}
		}
		timer.Stop()

		if primaryResults != nil {
			// 主源在保留窗内就绪：以其为主，短窗合并 keyless 独有结果。
			extras = append(extras, collectMergeWindow(out, attemptCh, resultCh, ctx, nil)...)
		} else {
			// 主源超窗/失败：keyless 首胜+短窗兜底。超窗时主源无尝试记录，
			// 补一条合成诊断（保留窗超时），让「配了主源但没生效」可见。
			if primaryTimedOut := primaryDegraded == "primary-timeout"; primaryTimedOut {
				out.Attempts = append(out.Attempts, SearchAttempt{
					Name: primaryName, Outcome: attemptError,
					Err: fmt.Sprintf("主源保留窗(%v)超时", searchPrimaryWindow),
				})
			}
			if len(extras) == 0 {
				first := waitFirstResult(out, len(runners), attemptCh, resultCh, ctx)
				if first == nil {
					return allFailedOutcome(out, attemptCh, primaryDegraded)
				}
				extras = append(extras, *first)
			}
			extras = append(extras, collectMergeWindow(out, attemptCh, resultCh, ctx, nil)...)
		}
	}

	// 收尾诊断：非阻塞收齐已到账尝试（迟到者不等待）。
	out.Attempts = append(out.Attempts, drainAttempts(attemptCh, len(runners)-len(out.Attempts))...)

	merged := mergeSearchResults(primaryResults, extras, topK)
	out.Results = merged

	if len(merged) == 0 {
		return allFailedOutcome(out, attemptCh, primaryDegraded)
	}

	switch {
	case primaryName != "" && primaryResults == nil:
		// 配置了主源但主源结果全缺席：强制 DEGRADED，禁止静默 bing-only 标 OK。
		out.State = SearchStateDegraded
		out.DegradedReason = primaryDegraded
	default:
		pass, reason := coverageGate(query, merged)
		if pass {
			out.State = SearchStateOK
		} else {
			out.State = SearchStateDegraded
			out.DegradedReason = reason
		}
	}
	return out, nil
}

// waitFirstResult 等首个引擎结果；全部引擎落账仍无结果或截止 → nil（尝试恒记）。
func waitFirstResult(out *SearchOutcome, total int, attemptCh chan SearchAttempt, resultCh chan taggedResults, ctx context.Context) *taggedResults {
	for len(out.Attempts) < total {
		select {
		case tr := <-resultCh:
			return &tr
		case a := <-attemptCh:
			out.Attempts = append(out.Attempts, a)
		case <-ctx.Done():
			return nil
		}
	}
	return nil
}

// collectMergeWindow 合并短窗：窗内或全部引擎落账前持续收结果与尝试。
// stop 用于提前结束（当前无消费方，保留给刀3 动态窗）。
func collectMergeWindow(out *SearchOutcome, attemptCh chan SearchAttempt, resultCh chan taggedResults, ctx context.Context, stop <-chan struct{}) []taggedResults {
	timer := time.NewTimer(searchMergeWindow)
	defer timer.Stop()
	var extras []taggedResults
	for len(out.Attempts) < cap(attemptCh) {
		select {
		case tr := <-resultCh:
			extras = append(extras, tr)
		case a := <-attemptCh:
			out.Attempts = append(out.Attempts, a)
		case <-timer.C:
			return extras
		case <-ctx.Done():
			return extras
		case <-stop:
			return extras
		}
	}
	return extras
}

// allFailedOutcome 无可用结果收口：EMPTY（有引擎真完成过检索）或 ALL_FAILED
// （全败/截止）。两者都附逐引擎诊断并以 err 收口（办公「空/全败即 error」契约；
// primaryDegraded 非空时在诊断头部点名主源缺位）。
func allFailedOutcome(out *SearchOutcome, attemptCh chan SearchAttempt, primaryDegraded string) (*SearchOutcome, error) {
	out.Attempts = append(out.Attempts, drainAttempts(attemptCh, cap(attemptCh)-len(out.Attempts))...)
	out.State = ClassifySearchState(nil, out.Attempts)
	diag := ""
	if primaryDegraded != "" {
		diag = fmt.Sprintf("主源缺位（%s）。", primaryDegraded)
	}
	out.Diagnostics = diag + buildAllFailedDiag(out.Attempts)
	return out, fmt.Errorf("%s", out.Diagnostics)
}

// mergeSearchResults 主源优先 + 引擎到达序的短窗合并：URL 规范化去重（www./
// 尾斜杠/utm_* 归一），重复条目保留先到者并把来源并入 Source（"a+b"），截断 topK。
func mergeSearchResults(primary []SearchResult, extras []taggedResults, topK int) []SearchResult {
	seen := make(map[string]int)
	out := make([]SearchResult, 0, topK)
	addAll := func(rs []SearchResult) {
		for _, r := range rs {
			key := normalizeSearchURL(r.URL)
			if key == "" {
				continue
			}
			if idx, dup := seen[key]; dup {
				if r.Source != "" && !strings.Contains(out[idx].Source, r.Source) {
					out[idx].Source += "+" + r.Source
				}
				continue
			}
			if len(out) >= topK {
				return
			}
			seen[key] = len(out)
			out = append(out, r)
		}
	}
	addAll(primary)
	for _, tr := range extras {
		addAll(tr.results)
	}
	return out
}

// normalizeSearchURL 跨引擎 URL 归一键：小写 host、去 www. 前缀、去尾斜杠、
// 剥 utm_*/from/spm 跟踪参数。Bing ck/a 跳转链接的展开仍是台账项（无实网样本）。
func normalizeSearchURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return strings.ToLower(raw)
	}
	host := strings.ToLower(u.Host)
	host = strings.TrimPrefix(host, "www.")
	path := strings.TrimRight(u.Path, "/")
	if u.RawQuery == "" {
		return host + path
	}
	q := u.Query()
	changed := false
	for k := range q {
		lk := strings.ToLower(k)
		if strings.HasPrefix(lk, "utm_") || lk == "from" || lk == "spm" {
			q.Del(k)
			changed = true
		}
	}
	if !changed {
		return host + path + "?" + u.RawQuery
	}
	if enc := q.Encode(); enc != "" {
		return host + path + "?" + enc
	}
	return host + path
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
