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
	"strings"
	"time"

	nethtml "golang.org/x/net/html"

	"github.com/gaea/gaea/internal/gaea/tool"
	"github.com/gaea/gaea/internal/netclient"
)

func init() { tool.RegisterBuiltin(webSearch{}) }

type webSearch struct{}

const (
	webSearchTimeout    = 15 * time.Second // per-engine HTTP timeout
	webSearchMaxRetries = 1                // retries per engine: 0, 1 (= 1 retry)
	webSearchMaxRead    = 512 << 10        // 512 KB
	webSearchTotalLimit = 20 * time.Second // total execution deadline
)

// --- search engine interface ---

// searchEngine abstracts a single search backend.
type searchEngine interface {
	// Name returns a human-readable label for error messages.
	Name() string
	// Available reports whether this engine is configured and ready.
	Available() bool
	// Search executes a search and returns results (never nil on success).
	Search(ctx context.Context, query string, limit int) ([]searchResult, error)
}

// --- webSearch tool implementation ---

func (webSearch) Name() string { return "web_search" }

func (webSearch) Description() string {
	return "搜索公开网页（通过 SearXNG / Tavily / Brave Search / Bing / DuckDuckGo）。返回结构化 JSON 数组，每项含 title/url/snippet/source 字段，支持引用追踪。当答案的正确性依赖于当前状态时使用——任何随时间变化的内容（事件、价格、发布版本、现实世界的状态）。先搜索再回答；常青问题不需要此工具。"
}

func (webSearch) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "query":{"type":"string","description":"自然语言搜索词"},
  "topK":{"type":"integer","description":"返回结果数（默认5，最多10）","minimum":1,"maximum":10}
},
"required":["query"]
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
		TopK  int    `json:"topK"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if p.Query == "" {
		return "", fmt.Errorf("query is required")
	}
	if p.TopK <= 0 {
		p.TopK = 5
	}
	if p.TopK > 10 {
		p.TopK = 10
	}

	engines := ws.buildEngines()

	// Parallel execution: every engine fires in its own goroutine.
	// First success wins; failures are collected for diagnostics.
	resultCh := make(chan []searchResult, 1)
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
			results, err := eng.Search(ctx, p.Query, p.TopK)
			elapsed := time.Since(start)
			if err != nil {
				errCh <- engineError{name: eng.Name(), err: err, elapsed: elapsed}
				return
			}
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

// buildEngines returns engines in priority order: local SearXNG → Tavily → Brave → public SearXNG.
func (webSearch) buildEngines() []searchEngine {
	var engines []searchEngine
	cfg := searchCfg // may be nil

	// 1. Local SearXNG (fastest, private)
	if cfg != nil && cfg.LocalSearXNGURL != "" {
		engines = append(engines, &localSearxNGEngine{baseURL: cfg.LocalSearXNGURL})
	}

	// 2. Tavily Search API
	if cfg != nil && cfg.TavilyKey() != "" {
		engines = append(engines, &tavilyEngine{apiKey: cfg.TavilyKey()})
	}

	// 3. Brave Search API
	if cfg != nil && cfg.BraveKey() != "" {
		engines = append(engines, &braveEngine{apiKey: cfg.BraveKey()})
	}

	// 4. Public SearXNG instances (always available as fallback)
	engines = append(engines, &publicSearxNGEngine{})

	// 5. Bing web search (keyless; reachable without a proxy in mainland China)
	engines = append(engines, &bingEngine{})

	// 6. DuckDuckGo Lite (keyless; reliable in most regions)
	engines = append(engines, &duckDuckGoLiteEngine{})

	return engines
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
func (e *localSearxNGEngine) Search(ctx context.Context, query string, limit int) ([]searchResult, error) {
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
func (e *publicSearxNGEngine) Search(ctx context.Context, query string, limit int) ([]searchResult, error) {
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
func (e *bingEngine) Search(ctx context.Context, query string, limit int) ([]searchResult, error) {
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
func parseBingResults(body []byte, limit int) ([]searchResult, error) {
	doc, err := nethtml.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse bing response: %w", err)
	}
	var results []searchResult
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

func bingResultFromBlock(li *nethtml.Node) (searchResult, bool) {
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
		return searchResult{}, false
	}
	return searchResult{Title: title, URL: href, Snippet: truncate(snippet, 300), Source: "bing"}, true
}

// --- DuckDuckGo Lite Engine (keyless HTML fallback) ---

type duckDuckGoLiteEngine struct{}

func (e *duckDuckGoLiteEngine) Name() string    { return "duckduckgo-lite" }
func (e *duckDuckGoLiteEngine) Available() bool { return true }
func (e *duckDuckGoLiteEngine) Search(ctx context.Context, query string, limit int) ([]searchResult, error) {
	searchURL := "https://lite.duckduckgo.com/lite/?q=" + url.QueryEscape(query)
	body, err := doSearchRequest(ctx, searchHTTPClient(), searchURL, 0)
	if err != nil {
		return nil, err
	}
	return parseDDGLiteResults(body, limit)
}

// parseDDGLiteResults extracts results from DuckDuckGo Lite: <a class="result-link">
// entries with snippets in <td class="result-snippet"> cells.
func parseDDGLiteResults(body []byte, limit int) ([]searchResult, error) {
	doc, err := nethtml.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("parse duckduckgo response: %w", err)
	}
	var results []searchResult
	var snippets []string
	var walk func(*nethtml.Node)
	walk = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode && n.Data == "a" && hasClass(n, "result-link") && len(results) < limit {
			href := strings.TrimSpace(attrValue(n, "href"))
			title := strings.TrimSpace(nodeText(n))
			if href != "" && title != "" {
				results = append(results, searchResult{Title: title, URL: href, Source: "duckduckgo"})
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

func (e *tavilyEngine) Search(ctx context.Context, query string, limit int) ([]searchResult, error) {
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
	results := make([]searchResult, 0, limit)
	for _, r := range tr.Results {
		if len(results) >= limit {
			break
		}
		results = append(results, searchResult{
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

func (e *braveEngine) Search(ctx context.Context, query string, limit int) ([]searchResult, error) {
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
	results := make([]searchResult, 0, limit)
	for _, r := range br.Web.Results {
		if len(results) >= limit {
			break
		}
		results = append(results, searchResult{
			Title:   strings.TrimSpace(r.Title),
			URL:     strings.TrimSpace(r.URL),
			Snippet: truncate(r.Description, 300),
			Source:  "brave",
		})
	}
	return results, nil
}

// --- shared SearXNG implementation ---

func trySearXNG(ctx context.Context, baseURL, query string, limit int) ([]searchResult, error) {
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
	results := make([]searchResult, 0, limit)
	for _, r := range resp.Results {
		if len(results) >= limit {
			break
		}
		results = append(results, searchResult{
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

func formatResults(results []searchResult) string {
	var out strings.Builder
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(results)
	return strings.TrimSpace(out.String())
}

// --- helpers ---

func truncate(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

type searchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Source  string `json:"source"`
}
