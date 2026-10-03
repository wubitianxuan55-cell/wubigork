package builtin

// websearch_engine.go — 搜索引擎 seam（批 25 GA2-01 文件拆分，自 websearch.go
// 原位搬移，零逻辑改动）：SearchEngine 接口 + kind 注册表 + 默认序 + 6 引擎
// 实现（local/public SearXNG、Bing、DDG Lite、Tavily、Brave）+ 引擎侧共享
// HTTP（searchHTTPClient/searchProxyURLFor/doSearchRequest）+ SearchResult
// 结果契约。工具本体与编排见 websearch.go，HTML 解析见 websearch_html.go。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/netclient"
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

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Source  string `json:"source"`
}
