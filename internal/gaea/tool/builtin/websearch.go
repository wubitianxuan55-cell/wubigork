package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	nethtml "golang.org/x/net/html"

	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/netclient"
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

// --- search engine seam（定义 / 提供者 / 消费者） ---
// 3.0 Step 3d #1：websearch 6 引擎硬编码扇出收敛为注册表 + config 可配序。
// 范式见 internal/gaea/provider/provider.go 与 internal/ai/image_backend.go
// 的 Register/New/Kinds；消费者（webSearch.Execute → buildEngines）只依赖
// SearchEngine 接口与 config 驱动的 kind 顺序，不硬编码引擎实现。

// SearchEngineKind 各引擎注册 kind（稳定标识，config 按此排顺序）。
const (
	SearchEngineKindLocalSearXNG  = "local-searxng"
	SearchEngineKindTavily        = "tavily"
	SearchEngineKindBrave         = "brave"
	SearchEngineKindPublicSearXNG = "public-searxng"
	SearchEngineKindBing          = "bing"
	SearchEngineKindDuckDuckGo    = "duckduckgo-lite"
)

// defaultSearchEngineOrder 与改造前 buildEngines 的硬编码优先级一致：
// local SearXNG → Tavily → Brave → public SearXNG → Bing → DDG。
// config（[search] engine_order）未配置时使用此默认序，行为零变化。
var defaultSearchEngineOrder = []string{
	SearchEngineKindLocalSearXNG,
	SearchEngineKindTavily,
	SearchEngineKindBrave,
	SearchEngineKindPublicSearXNG,
	SearchEngineKindBing,
	SearchEngineKindDuckDuckGo,
}

// SearchEngine abstracts a single search backend.
type SearchEngine interface {
	// Name returns a human-readable label for error messages.
	Name() string
	// Available reports whether this engine is configured and ready.
	Available() bool
	// Search executes a search and returns results (never nil on success).
	Search(ctx context.Context, query string, limit int) ([]SearchResult, error)
}

// SearchEngineConfig 是引擎实例化入参（注册表 New 用）。
// 各 kind 按需读取字段：local-searxng 用 BaseURL；tavily/brave 用 APIKey。
type SearchEngineConfig struct {
	BaseURL string
	APIKey  string
}

// SearchEngineFactory 按实例配置构建搜索引擎（kind → 实例）。
type SearchEngineFactory func(cfg SearchEngineConfig) (SearchEngine, error)

// searchEngineRegistry kind → 工厂注册表。各实现 init() 自注册；
// 互斥注册，重复即 panic（编译期接线错误）。
var searchEngineRegistry = map[string]SearchEngineFactory{}

// RegisterSearchEngine 注册搜索引擎 kind（如 "tavily" / "bing"）。
// kind 为空或重复注册直接 panic。
func RegisterSearchEngine(kind string, factory SearchEngineFactory) {
	if kind == "" {
		panic("builtin: search engine kind must not be empty")
	}
	if _, dup := searchEngineRegistry[kind]; dup {
		panic("builtin: duplicate search engine kind " + kind)
	}
	searchEngineRegistry[kind] = factory
}

// NewSearchEngine 按 kind 经注册表构建引擎；未知 kind 返回错误
// （fail-closed，附已注册 kind 列表）。
func NewSearchEngine(kind string, cfg SearchEngineConfig) (SearchEngine, error) {
	factory, ok := searchEngineRegistry[kind]
	if !ok {
		return nil, fmt.Errorf("builtin: unknown search engine kind %q (registered: %v)", kind, SearchEngineKinds())
	}
	eng, err := factory(cfg)
	if err != nil {
		return nil, err
	}
	if eng == nil {
		return nil, fmt.Errorf("builtin: search engine factory %q returned nil", kind)
	}
	return eng, nil
}

// SearchEngineKinds 返回已注册引擎 kind 列表（排序，供诊断/校验）。
func SearchEngineKinds() []string {
	out := make([]string, 0, len(searchEngineRegistry))
	for k := range searchEngineRegistry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

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

// engineError records a failed engine attempt for diagnostics.
type engineError struct {
	name    string
	err     error
	elapsed time.Duration
}

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

	engines := ws.buildEngines()

	// 域名策略受限时（[search] allow_domains/deny_domains），向引擎请求更多结果再按
	// 策略过滤，避免“前几条被拒 → 结果为空”（unsloth web_access_policy over-fetch）。
	requested := p.TopK
	if searchPolicyRestricted() {
		requested = p.TopK * searchPolicyOverfetch
	}

	// Parallel execution: every engine fires in its own goroutine.
	// First success wins; failures are collected for diagnostics.
	resultCh := make(chan []SearchResult, 1)
	errCh := make(chan engineError, len(engines))

	ctx, cancel := context.WithTimeout(ctx, webSearchTotalLimit)
	defer cancel()

	for _, eng := range engines {
		eng := eng
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("websearch: engine goroutine panic recovered", "engine", eng.Name(), "panic", r)
					errCh <- engineError{name: eng.Name(), err: fmt.Errorf("engine panic: %v", r), elapsed: 0}
				}
			}()
			start := time.Now()
			results, err := eng.Search(ctx, p.Query, requested)
			elapsed := time.Since(start)
			if err != nil {
				errCh <- engineError{name: eng.Name(), err: err, elapsed: elapsed}
				return
			}
			// 应用域名 allow/deny 策略并截断到 topK；过滤后为空视作该引擎无可用结果。
			results = filterSearchResults(results, p.TopK)
			if len(results) == 0 {
				errCh <- engineError{name: eng.Name(), err: fmt.Errorf("no results"), elapsed: elapsed}
				return
			}
			select {
			case resultCh <- results:
			default:
				// another engine already won, discard
			}
		}()
	}

	// Collect: first result wins, or accumulate all failures.
	var failures []engineError
	for i := 0; i < len(engines); i++ {
		select {
		case results := <-resultCh:
			return formatResults(results), nil
		case fe := <-errCh:
			failures = append(failures, fe)
		case <-ctx.Done():
			// Timeout — drain any remaining errors that arrive quickly.
			failures = append(failures, engineError{name: "timeout", err: ctx.Err()})
			for j := i + 1; j < len(engines); j++ {
				select {
				case fe := <-errCh:
					failures = append(failures, fe)
				case <-time.After(100 * time.Millisecond):
				}
			}
			i = len(engines) // break outer loop
		}
	}

	// All engines failed — build detailed diagnostic.
	var diag strings.Builder
	diag.WriteString("所有搜索引擎失败：")
	for _, f := range failures {
		fmt.Fprintf(&diag, "\n  • %s (%v): %v", f.name, f.elapsed.Round(time.Millisecond), f.err)
	}
	if searchCfg == nil || (searchCfg.TavilyAPIKeyEnv == "" && searchCfg.BraveAPIKeyEnv == "" && searchCfg.LocalSearXNGURL == "") {
		diag.WriteString("\n\n💡 提示：配置搜索 API 可大幅提高成功率：")
		diag.WriteString("\n  1. Tavily（免费 1000次/月）：注册 tavily.com → 设环境变量 TAVILY_API_KEY")
		diag.WriteString("\n  2. Brave Search（免费 2000次/月）：注册 api.search.brave.com → 设环境变量 BRAVE_API_KEY")
		diag.WriteString("\n  3. 自建 SearXNG：docker run -d -p 8080:8080 searxng/searxng")
		diag.WriteString("\n  然后在 gaea.toml 中配置 [search] 节。")
	}
	return "", fmt.Errorf("%s", diag.String())
}

// buildEngines 按 config 驱动的 kind 顺序经注册表构建引擎（可用性过滤）。
// 顺序来源：[search] engine_order（SetSearchEngineOrder 注入），未配置时用
// defaultSearchEngineOrder（与改造前硬编码优先级一致：local SearXNG → Tavily →
// Brave → public SearXNG → Bing → DDG）。每个 kind 经 NewSearchEngine 构建后
// 用 Available() 过滤（等价于旧逻辑里"有 key/有 URL 才加入"的分支）。
func (webSearch) buildEngines() []SearchEngine {
	var engines []SearchEngine
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
		engines = append(engines, eng)
	}
	return engines
}

// searchEngineOrder 返回生效的引擎顺序：config 注入序优先，否则默认序。
func searchEngineOrder() []string {
	if len(searchEngineOrderCfg) > 0 {
		return searchEngineOrderCfg
	}
	return defaultSearchEngineOrder
}

// --- HTTP client ---

// searchHTTPClient returns an HTTP client with SSRF protection and the same
// proxy behaviour as web_fetch (auto/env/custom/off from the gaea network
// config). Without this, search engines behind a proxy all time out and the
// tool reports "所有搜索引擎失败".
func searchHTTPClient() *http.Client {
	timeout := webSearchTimeout
	if searchCfg != nil {
		timeout = searchCfg.SearchTimeout()
	}
	return ssrfGuardedClient(timeout, searchProxyURLFor)
}

// searchProxyURLFor resolves the configured network proxy for a search request.
// An empty result means direct connection (no proxy applies).
func searchProxyURLFor(req *http.Request) (string, error) {
	pf, err := netclient.ProxyFunc(searchProxy)
	if err != nil {
		return "", err
	}
	if pf == nil {
		return "", nil
	}
	u, err := pf(req)
	if err != nil || u == nil {
		return "", err
	}
	return u.String(), nil
}

// --- Local SearXNG Engine ---

type localSearxNGEngine struct{ baseURL string }

func (e *localSearxNGEngine) Name() string    { return "local-searxng" }
func (e *localSearxNGEngine) Available() bool { return e.baseURL != "" }
func (e *localSearxNGEngine) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	return trySearXNG(ctx, e.baseURL, query, limit)
}

// --- Public SearXNG Engine ---

// publicSearxNGInstances — publicly accessible SearXNG instances returning JSON.
var publicSearxNGInstances = []string{
	"https://searx.be",
	"https://search.sapti.me",
	"https://searx.dresden.network",
	"https://search.bus-hit.me",
	"https://searx.tuxcloud.net",
	"https://search.ipv6s.net",
}

type publicSearxNGEngine struct{}

func (e *publicSearxNGEngine) Name() string    { return "public-searxng" }
func (e *publicSearxNGEngine) Available() bool { return true }
func (e *publicSearxNGEngine) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	var lastErr error
	for _, baseURL := range publicSearxNGInstances {
		results, err := trySearXNG(ctx, baseURL, query, limit)
		if err == nil && len(results) > 0 {
			return results, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// --- Bing Web Search Engine (keyless HTML fallback) ---

type bingEngine struct{}

func (e *bingEngine) Name() string    { return "bing" }
func (e *bingEngine) Available() bool { return true }
func (e *bingEngine) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	searchURL := fmt.Sprintf("https://www.bing.com/search?q=%s&count=%d&setlang=zh-CN&mkt=zh-CN",
		url.QueryEscape(query), limit)
	body, err := doSearchRequest(ctx, searchHTTPClient(), searchURL, webSearchMaxRetries)
	if err != nil {
		return nil, err
	}
	return parseBingResults(body, limit)
}

// parseBingResults extracts organic results from a Bing SERP: <li class="b_algo">
// blocks with the title in <h2><a> and the snippet inside <div class="b_caption">.
func parseBingResults(body []byte, limit int) ([]SearchResult, error) {
	doc, err := nethtml.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse bing response: %w", err)
	}
	var results []SearchResult
	var walk func(*nethtml.Node)
	walk = func(n *nethtml.Node) {
		if len(results) >= limit {
			return
		}
		if n.Type == nethtml.ElementNode && n.Data == "li" && hasClass(n, "b_algo") {
			if r, ok := bingResultFromBlock(n); ok {
				results = append(results, r)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if len(results) == 0 {
		return nil, fmt.Errorf("no bing results parsed")
	}
	return results, nil
}

func bingResultFromBlock(li *nethtml.Node) (SearchResult, bool) {
	var title, href, snippet string
	var walk func(*nethtml.Node)
	walk = func(n *nethtml.Node) {
		if href != "" && snippet != "" {
			return
		}
		if n.Type == nethtml.ElementNode && n.Data == "h2" && href == "" {
			if a := firstChildElement(n, "a"); a != nil {
				href = strings.TrimSpace(attrValue(a, "href"))
				title = strings.TrimSpace(nodeText(a))
			}
		}
		if n.Type == nethtml.ElementNode && n.Data == "div" && hasClass(n, "b_caption") && snippet == "" {
			if p := firstDescendantElement(n, "p"); p != nil {
				snippet = strings.TrimSpace(nodeText(p))
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(li)
	if title == "" || href == "" || strings.HasPrefix(href, "javascript:") {
		return SearchResult{}, false
	}
	return SearchResult{Title: title, URL: href, Snippet: truncate(snippet, 300), Source: "bing"}, true
}

// --- DuckDuckGo Lite Engine (keyless HTML fallback) ---

type duckDuckGoLiteEngine struct{}

func (e *duckDuckGoLiteEngine) Name() string    { return "duckduckgo-lite" }
func (e *duckDuckGoLiteEngine) Available() bool { return true }
func (e *duckDuckGoLiteEngine) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	searchURL := "https://lite.duckduckgo.com/lite/?q=" + url.QueryEscape(query)
	body, err := doSearchRequest(ctx, searchHTTPClient(), searchURL, 0)
	if err != nil {
		return nil, err
	}
	return parseDDGLiteResults(body, limit)
}

// parseDDGLiteResults extracts results from DuckDuckGo Lite: <a class="result-link">
// entries with snippets in <td class="result-snippet"> cells.
func parseDDGLiteResults(body []byte, limit int) ([]SearchResult, error) {
	doc, err := nethtml.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse duckduckgo response: %w", err)
	}
	var results []SearchResult
	var snippets []string
	var walk func(*nethtml.Node)
	walk = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode && n.Data == "a" && hasClass(n, "result-link") && len(results) < limit {
			href := strings.TrimSpace(attrValue(n, "href"))
			title := strings.TrimSpace(nodeText(n))
			if href != "" && title != "" {
				results = append(results, SearchResult{Title: title, URL: href, Source: "duckduckgo"})
			}
		}
		if n.Type == nethtml.ElementNode && n.Data == "td" && hasClass(n, "result-snippet") {
			snippets = append(snippets, strings.TrimSpace(nodeText(n)))
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if len(results) == 0 {
		return nil, fmt.Errorf("no duckduckgo results parsed")
	}
	for i := range results {
		if i < len(snippets) {
			results[i].Snippet = truncate(snippets[i], 300)
		}
	}
	return results, nil
}

// --- HTML helpers ---

func hasClass(n *nethtml.Node, class string) bool {
	for _, a := range n.Attr {
		if a.Key != "class" {
			continue
		}
		for _, c := range strings.Fields(a.Val) {
			if c == class {
				return true
			}
		}
	}
	return false
}

func attrValue(n *nethtml.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func nodeText(n *nethtml.Node) string {
	var sb strings.Builder
	var walk func(*nethtml.Node)
	walk = func(m *nethtml.Node) {
		if m.Type == nethtml.TextNode {
			sb.WriteString(m.Data)
		}
		for c := m.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}

func firstChildElement(n *nethtml.Node, tag string) *nethtml.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == nethtml.ElementNode && c.Data == tag {
			return c
		}
	}
	return nil
}

func firstDescendantElement(n *nethtml.Node, tag string) *nethtml.Node {
	if n.Type == nethtml.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if r := firstDescendantElement(c, tag); r != nil {
			return r
		}
	}
	return nil
}

// --- Tavily Search API Engine ---

type tavilyEngine struct{ apiKey string }

func (e *tavilyEngine) Name() string    { return "tavily" }
func (e *tavilyEngine) Available() bool { return e.apiKey != "" }

type tavilyRequest struct {
	Query         string `json:"query"`
	SearchDepth   string `json:"search_depth,omitempty"`
	MaxResults    int    `json:"max_results,omitempty"`
	IncludeAnswer bool   `json:"include_answer,omitempty"`
}

type tavilyResponse struct {
	Results []struct {
		Title   string  `json:"title"`
		URL     string  `json:"url"`
		Content string  `json:"content"`
		Score   float64 `json:"score"`
	} `json:"results"`
	Answer string `json:"answer,omitempty"`
}

func (e *tavilyEngine) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	body, err := json.Marshal(tavilyRequest{
		Query:      query,
		MaxResults: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.tavily.com/search", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+e.apiKey)
	httpReq.Header.Set("User-Agent", "gaea/1.0")

	resp, err := searchHTTPClient().Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, webSearchMaxRead))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var tr tavilyResponse
	if err := json.Unmarshal(respBody, &tr); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// Tavily
	results := make([]SearchResult, 0, limit)
	for _, r := range tr.Results {
		if len(results) >= limit {
			break
		}
		results = append(results, SearchResult{
			Title:   strings.TrimSpace(r.Title),
			URL:     strings.TrimSpace(r.URL),
			Snippet: truncate(r.Content, 300),
			Source:  "tavily",
		})
	}
	return results, nil
}

// --- Brave Search API Engine ---

type braveEngine struct{ apiKey string }

func (e *braveEngine) Name() string    { return "brave" }
func (e *braveEngine) Available() bool { return e.apiKey != "" }

type braveResponse struct {
	Web struct {
		Results []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
		} `json:"results"`
	} `json:"web"`
}

func (e *braveEngine) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	searchURL := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?q=%s&count=%d",
		url.QueryEscape(query), limit)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Accept-Encoding", "gzip")
	httpReq.Header.Set("X-Subscription-Token", e.apiKey)
	httpReq.Header.Set("User-Agent", "gaea/1.0")

	resp, err := searchHTTPClient().Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, webSearchMaxRead))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var br braveResponse
	if err := json.Unmarshal(respBody, &br); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// Brave
	results := make([]SearchResult, 0, limit)
	for _, r := range br.Web.Results {
		if len(results) >= limit {
			break
		}
		results = append(results, SearchResult{
			Title:   strings.TrimSpace(r.Title),
			URL:     strings.TrimSpace(r.URL),
			Snippet: truncate(r.Description, 300),
			Source:  "brave",
		})
	}
	return results, nil
}

// --- shared SearXNG implementation ---

func trySearXNG(ctx context.Context, baseURL, query string, limit int) ([]SearchResult, error) {
	searchURL := fmt.Sprintf("%s/search?%s", strings.TrimRight(baseURL, "/"),
		"q="+url.QueryEscape(query)+"&format=json&language=zh-CN&safesearch=1")

	client := searchHTTPClient()
	body, err := doSearchRequest(ctx, client, searchURL, webSearchMaxRetries)
	if err != nil {
		return nil, err
	}

	var resp searxNGResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse SearXNG response: %w", err)
	}

	// SearXNG
	results := make([]SearchResult, 0, limit)
	for _, r := range resp.Results {
		if len(results) >= limit {
			break
		}
		results = append(results, SearchResult{
			Title:   strings.TrimSpace(r.Title),
			URL:     strings.TrimSpace(r.URL),
			Snippet: truncate(r.Content, 300),
			Source:  "searxng",
		})
	}
	return results, nil
}

type searxNGResponse struct {
	Results []struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Content string `json:"content"`
	} `json:"results"`
}

// --- shared HTTP ---

func doSearchRequest(ctx context.Context, client *http.Client, urlStr string, maxRetries int) ([]byte, error) {
	timeout := webSearchTimeout
	if searchCfg != nil {
		timeout = searchCfg.SearchTimeout()
	}
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(1<<(attempt-1)) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		reqCtx, cancel := context.WithTimeout(ctx, timeout)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, urlStr, nil)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0 Safari/537.36")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/json;q=0.9,*/*;q=0.8")

		resp, err := client.Do(req)
		if err != nil {
			cancel()
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			resp.Body.Close()
			cancel()
			lastErr = fmt.Errorf("search engine returned %d", resp.StatusCode)
			break // rate-limited / overloaded: no point retrying, try next engine
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, webSearchMaxRead))
		resp.Body.Close()
		cancel()
		if err != nil {
			lastErr = fmt.Errorf("read body: %w", err)
			continue
		}
		return body, nil
	}
	return nil, fmt.Errorf("search failed after %d retries: %w", maxRetries+1, lastErr)
}

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

func truncate(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Source  string `json:"source"`
}
