package booksource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/gaea/gaea/internal/bookimport"
)

// ── 抓取纪律（规格 §2.3：抖动 / 线性退避重试 / 并发钳制 / 每请求超时）───────

// Sleeper 间隔等待，可注入替换（测试零真实睡眠；ctx 取消即返回）。
type Sleeper func(ctx context.Context, d time.Duration) error

func systemSleeper(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// Request 一次抓取的完整描述（搜索的 POST 表单 / cookies 也在同一通道）。
type Request struct {
	Method  string            // get（默认）| post
	URL     string            // 必填
	Form    url.Values        // post 表单
	Cookies map[string]string // 规则 search.cookies
}

// Page 抓取结果：原始字节（解码交给引擎，统一走 bookimport 五级编码链）。
type Page struct {
	URL  string
	Body []byte
}

// Fetcher 网络通道，app 层注入（生产走 netclient 代理语义；测试注入假站）。
type Fetcher interface {
	Fetch(ctx context.Context, req Request) (*Page, error)
}

// HTTPFetcher 生产实现：随机 UA + Referer=host + 每请求独立超时。
type HTTPFetcher struct {
	Client  *http.Client  // 缺省 http.DefaultClient 的浅改造（app 层会注入带代理的）
	Timeout time.Duration // 每请求；缺省 DefaultTimeout 秒
}

// 常见桌面/移动 UA 池（自备的通用清单，非上游拷贝）。
var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:127.0) Gecko/20100101 Firefox/127.0",
	"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1",
}

func (f *HTTPFetcher) Fetch(ctx context.Context, req Request) (*Page, error) {
	client := f.Client
	if client == nil {
		client = http.DefaultClient
	}
	timeout := f.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout * time.Second
	}
	method := strings.ToUpper(req.Method)
	if method == "" {
		method = http.MethodGet
	}
	var body io.Reader
	if method == http.MethodPost && len(req.Form) > 0 {
		body = strings.NewReader(req.Form.Encode())
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, req.URL, body)
	if err != nil {
		return nil, err
	}
	if method == http.MethodPost {
		httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	httpReq.Header.Set("User-Agent", userAgents[rand.IntN(len(userAgents))]) //nolint:gosec // UA 池随机，非安全用途
	if u, err := url.Parse(req.URL); err == nil && u.Host != "" {
		httpReq.Header.Set("Referer", u.Scheme+"://"+u.Host+"/")
	}
	if len(req.Cookies) > 0 {
		pairs := make([]string, 0, len(req.Cookies))
		for k, v := range req.Cookies {
			pairs = append(pairs, k+"="+v)
		}
		httpReq.Header.Set("Cookie", strings.Join(pairs, "; "))
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	resp, err := client.Do(httpReq.WithContext(cctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// 重定向跟随后的最终 URL（重定向解包依赖，规格书源搜索 §2.1）
	finalURL := req.URL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	return &Page{URL: finalURL, Body: raw}, nil
}

// ── 引擎 ────────────────────────────────────────────────────────────────

// ErrCloudflare 页面命中 CF 真人验证标题集合（只检测报错，不绕过——规格 §4 D6）。
var ErrCloudflare = errors.New("页面命中 Cloudflare 真人验证，已停止（本引擎不提供绕过）")

// ErrEmptyContent 正文抽空：限流/风控的典型信号，如实报错不静默（规格 §2.7）。
var ErrEmptyContent = errors.New("正文内容为空（可能被限流）")

var cfTitles = map[string]bool{
	"Just a moment...":                       true,
	"403 Forbidden":                          true,
	"Attention Required":                     true,
	"Checking your browser before accessing": true,
}

// crawler 抓取纪律载体（抖动/退避/并发/CF 检测），Engine 与 WebSearcher 共用。
type crawler struct {
	cfg   CrawlConfig
	fetch Fetcher
	sleep Sleeper
	rand  *rand.Rand
}

func newCrawler(cfg CrawlConfig, opt Options) *crawler {
	if opt.Fetcher == nil {
		// 缺省生产通道（v4.283.1 走查补刀）：绑定层零注入不得 panic——
		// v4.283.0 实机走查抓到 NovelBookSourceSearch 空指针（HTTPFetcher
		// 的 Client==nil 回落已内建，这里只补「没注入就没有通道」这一层）。
		opt.Fetcher = &HTTPFetcher{}
	}
	if opt.Sleeper == nil {
		opt.Sleeper = systemSleeper
	}
	if opt.Rand == nil {
		opt.Rand = rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 1)) //nolint:gosec
	}
	return &crawler{cfg: cfg, fetch: opt.Fetcher, sleep: opt.Sleeper, rand: opt.Rand}
}

// jitter 请求间隔随机抖动 [min,max]。
func (c *crawler) jitter() time.Duration {
	return time.Duration(c.randBetween(c.cfg.MinIntervalMs, c.cfg.MaxIntervalMs)) * time.Millisecond
}

// retryJitter 重试间隔抖动 × 尝试次数（线性放大，书源规格 §2.3）。
func (c *crawler) retryJitter(attempt int) time.Duration {
	base := c.randBetween(c.cfg.RetryMinMs, c.cfg.RetryMaxMs)
	return time.Duration(base*attempt) * time.Millisecond
}

func (c *crawler) randBetween(min, max int) int {
	if max <= min {
		return min
	}
	return min + c.rand.IntN(max-min+1)
}

// pacedFetch 带纪律的抓取：先抖动再请求；失败按「重试抖动 × 尝试次数」退避，
// ctx 取消后零调用（不睡眠不请求）。
func (c *crawler) pacedFetch(ctx context.Context, req Request) (*goquery.Document, string, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", err
	}
	if err := c.sleep(ctx, c.jitter()); err != nil {
		return nil, "", err
	}
	page, err := c.fetch.Fetch(ctx, req)
	for attempt := 1; page == nil && err != nil && attempt <= c.cfg.MaxRetries; attempt++ {
		if ctx.Err() != nil {
			return nil, "", ctx.Err()
		}
		if errors.Is(err, ErrCloudflare) {
			return nil, "", err
		}
		if err := c.sleep(ctx, c.retryJitter(attempt)); err != nil {
			return nil, "", err
		}
		page, err = c.fetch.Fetch(ctx, req)
	}
	if err != nil {
		return nil, "", err
	}
	text, _ := bookimport.Decode(page.Body)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(text))
	if err != nil {
		return nil, "", err
	}
	title := strings.TrimSpace(doc.Find("title").First().Text())
	if cfTitles[title] {
		return nil, page.URL, ErrCloudflare
	}
	return doc, page.URL, nil
}

// runBounded 有界并发执行 fn(i)，i ∈ [0,n)；ctx 取消后未起跑的槽位记 ctx 错误。
// fn 内 panic 转该槽位 error（外部内容+用户规则解析面，单章异常不放大为进程崩溃）。
func (c *crawler) runBounded(ctx context.Context, n, limit int, fn func(i int) error) []error {
	if limit <= 0 {
		limit = c.cfg.Concurrency
	}
	if limit > n {
		limit = n
	}
	errs := make([]error, n)
	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				errs[i] = ctx.Err()
				return
			}
			errs[i] = func() (err error) {
				defer func() {
					if r := recover(); r != nil {
						err = fmt.Errorf("章节处理 panic: %v", r)
					}
				}()
				return fn(i)
			}()
		}(i)
	}
	wg.Wait()
	return errs
}

// Engine 单书源引擎：规则 + 注入的通道/时钟/随机源。
type Engine struct {
	*crawler
	rule *Rule
}

// Options 引擎依赖注入（零值可用：系统 sleeper、默认抓取参数）。
type Options struct {
	Fetcher Fetcher
	Sleeper Sleeper
	Rand    *rand.Rand
}

func New(rule *Rule, opt Options) *Engine {
	return &Engine{crawler: newCrawler(rule.crawlConfig(), opt), rule: rule}
}

// Rule 只读访问规则（前端展示书源名等）。
func (e *Engine) Rule() *Rule { return e.rule }
