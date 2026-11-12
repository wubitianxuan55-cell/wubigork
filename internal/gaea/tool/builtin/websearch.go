package builtin

// websearch.go — web_search 工具本体（批 25 GA2-01 文件拆分：原 921 行单文件
// 三关注点分家，本文件留「工具」——注册/Schema/Execute 编排/结果格式化与
// 域名策略过滤/url 直抓。引擎 seam 与 6 引擎实现见 websearch_engine.go，
// Bing/DDG SERP 的 HTML 解析见 websearch_html.go。均为同包原位搬移，零逻辑改动）。

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/gaea/strutil"
	"github.com/gaea/gaea/internal/gaea/tool"
)

func init() {
	tool.RegisterBuiltin(webSearch{})
	// 6 个搜索引擎自注册（3.0 Step 3d #1：6 引擎硬编码扇出 → 引擎注册表 + config 可配序）。
	// kind 列表见 SearchEngineKinds()；各引擎在注册表互斥注册，重复即 panic。
	RegisterSearchEngine(SearchEngineKindLocalSearXNG, func(cfg SearchEngineConfig) (SearchEngine, error) {
		return &localSearxNGEngine{baseURL: cfg.BaseURL}, nil
	})
	RegisterSearchEngine(SearchEngineKindTavily, func(cfg SearchEngineConfig) (SearchEngine, error) {
		return &tavilyEngine{apiKey: cfg.APIKey}, nil
	})
	RegisterSearchEngine(SearchEngineKindBrave, func(cfg SearchEngineConfig) (SearchEngine, error) {
		return &braveEngine{apiKey: cfg.APIKey}, nil
	})
	RegisterSearchEngine(SearchEngineKindPublicSearXNG, func(cfg SearchEngineConfig) (SearchEngine, error) {
		return &publicSearxNGEngine{}, nil
	})
	RegisterSearchEngine(SearchEngineKindBing, func(cfg SearchEngineConfig) (SearchEngine, error) {
		return &bingEngine{}, nil
	})
	RegisterSearchEngine(SearchEngineKindDuckDuckGo, func(cfg SearchEngineConfig) (SearchEngine, error) {
		return &duckDuckGoLiteEngine{}, nil
	})
}

type webSearch struct{}

const (
	webSearchTimeout    = 15 * time.Second // per-engine HTTP timeout
	webSearchMaxRetries = 1                // retries per engine: 0, 1 (= 1 retry)
	webSearchMaxRead    = 512 << 10        // 512 KB
	webSearchTotalLimit = 20 * time.Second // total execution deadline
	// searchPolicyOverfetch 域名策略受限时向引擎请求的倍数（unsloth web_access_policy
	// 的 over-fetch 行为）：先多抓再按 allow/deny 过滤，避免“前几条被拒 → 结果为空”。
	searchPolicyOverfetch = 3
)

// --- webSearch tool implementation ---

func (webSearch) Name() string { return "web_search" }

func (webSearch) Description() string {
	return "搜索公开网页（通过 SearXNG / Tavily / Brave Search / Bing / DuckDuckGo）。返回结构化 JSON 数组，每项含 title/url/snippet/source 字段，支持引用追踪。传 url 参数可跳过搜索、直接抓取该页面的完整文本（拿摘要后要正文/引用的场景）。当答案的正确性依赖于当前状态时使用——任何随时间变化的内容（事件、价格、发布版本、现实世界的状态）。先搜索再回答；常青问题不需要此工具。检索结果为实时抓取，按“今天”而非模型训练截止日作答。"
}

func (webSearch) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "query":{"type":"string","description":"自然语言搜索词"},
  "url":{"type":"string","description":"可选。传 URL 则不再搜索，直接抓取该页面的完整文本（先用 query 搜索得到摘要后，拿感兴趣结果目标的 url 再调本工具取全文）。"},
  "topK":{"type":"integer","description":"返回结果数（默认5，最多10）","minimum":1,"maximum":10}
}
}`)
}

func (webSearch) ReadOnly() bool { return true }

func (webSearch) CompactDescription() string     { return compactDesc["web_search"] }
func (webSearch) CompactSchema() json.RawMessage { return compactSchema["web_search"] }

func (ws webSearch) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		Query string `json:"query"`
		URL   string `json:"url"`
		TopK  int    `json:"topK"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	p.Query = strings.TrimSpace(p.Query)
	p.URL = strings.TrimSpace(p.URL)

	// unsloth 搜索工具的 url 模式：不再搜索，直接抓取目标页面的完整文本
	// （复用 web_fetch 的 SSRF 防护 + 域名 allow/deny 策略 + HTML→文本抽取），
	// 让“搜索→取全文→引用”收敛为一次工具调用。
	if p.URL != "" {
		return fetchSearchPage(ctx, p.URL)
	}
	if p.Query == "" {
		return "", fmt.Errorf("query is required (or set url to fetch a page directly)")
	}
	if p.TopK <= 0 {
		p.TopK = 5
	}
	if p.TopK > 10 {
		p.TopK = 10
	}

	// 编排委托统一检索核（众神殿刀1：聊天 adapter 与本工具共用一条链；刀2
	// 短窗合并+主源保留窗+覆盖门见 websearch_core.go/websearch_coverage.go）；
	// 输出契约零变化：成功=JSON 数组，全败=逐引擎诊断+零配置指引文本。
	outcome, err := ws.runSearch(ctx, p.Query, p.TopK)
	if err != nil {
		return "", err // err 消息即全败诊断（含零配置指引）
	}
	return formatResults(outcome.Results), nil
}

// buildEngines/buildEngineRunners/searchEngineOrder 见 websearch_core.go
// （刀2 起编排与主源识别按 kind，统一在核文件）。

// --- formatting ---

func formatResults(results []SearchResult) string {
	var out strings.Builder
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(results)
	return strings.TrimSpace(out.String())
}

// --- helpers ---

// searchPolicyRestricted 报告当前 [search] 域名策略（allow/deny）是否会约束
// web_search 的结果 url（对齐 unsloth web_access_policy 的语义：allow 列表非空
// 或 deny 列表非空即受限，需要过度抓取再过滤）。
func searchPolicyRestricted() bool {
	if searchCfg == nil {
		return false
	}
	return len(searchCfg.AllowDomains) > 0 || len(searchCfg.DenyDomains) > 0
}

// filterSearchResults 对搜索结果逐条应用域名 allow/deny 策略（checkDomainPolicy，
// 与 web_fetch 同一套 [search] enable/deny 域名单），并按 limit 截断。策略未配置
// 时原样返回（行为零变化）；受限时丢弃被拒 url，保留合规结果直到填满 limit
// （配合 searchPolicyOverfetch 过度抓取，对齐 unsloth 行为）。
func filterSearchResults(results []SearchResult, limit int) []SearchResult {
	if limit < 0 {
		limit = 0
	}
	if !searchPolicyRestricted() {
		if len(results) > limit {
			return results[:limit]
		}
		return results
	}
	out := make([]SearchResult, 0, limit)
	for _, r := range results {
		if len(out) >= limit {
			break
		}
		u, err := url.Parse(r.URL)
		if err != nil {
			continue
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			continue
		}
		host := u.Hostname()
		if host == "" {
			continue
		}
		if err := checkDomainPolicy(host); err != nil {
			continue
		}
		out = append(out, r)
	}
	return out
}

// fetchSearchPage 抓取单个 URL 的完整文本，供 web_search 在 url 模式下直接取全文
// （unsloth 搜索工具的 direct URL fetch 模式）。复用 web_fetch 的 SSRF 防护、域名
// allow/deny 策略与 HTML→文本抽取（同包 doFetch），并注入当前日期钉定，让模型按
// “今天的检索”而非训练截止日作答。
func fetchSearchPage(ctx context.Context, rawURL string) (string, error) {
	text, err := doFetch(ctx, rawURL, searchHTTPClient())
	if err != nil {
		return "", err
	}
	date := time.Now().Format("2006-01-02")
	return fmt.Sprintf("[web_search · url=%s · as of %s]\n%s", rawURL, date, text), nil
}

// truncate 先 TrimSpace 再按 rune 截断到 maxLen 个 rune，不补后缀。
//
// GA2-10 修正：原实现按**字节**切（`s[:maxLen]`），中文摘要被切在 rune 中间，
// JSON 编码后变成 U+FFFD。核心切片口径见 strutil.TruncateRunes；本包装保留
// 站点的 TrimSpace 前置政策。保留函数名是因为同包内 knowledge_search.go:254
// 与 websearch_live_test.go 也在调用它，而这两个文件不在本批「线 4」的文件
// 归属清单内（不动归属外文件），故只把实现下沉、不改名。
func truncate(s string, maxLen int) string {
	return strutil.TruncateRunes(strings.TrimSpace(s), maxLen)
}
