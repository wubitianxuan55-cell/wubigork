// Package whisper — web_search.go
// 多引擎 Web 搜索：DuckDuckGo Lite → Bing → 本地降级
// 补齐 agent_loop_runner.go + agent_tool_batch.go 的占位实现

package whisper

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// httpClient 共享 HTTP 客户端（复用连接）
var httpClient = &http.Client{Timeout: 12 * time.Second}

// userAgent 浏览器 UA（避免被反爬）
const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"

// maxResults 单次搜索最多返回条数
const maxResults = 5

// ─── 主入口：多引擎 fallback ──────────────────────────────────

// WebSearch 执行 Web 搜索，DDG 优先，失败降级到 Bing
func WebSearch(query string) (string, error) {
	// 引擎 1：DuckDuckGo Lite（国际通用，隐私友好）
	result, err := searchDDG(query)
	if err == nil && result != "" && !strings.Contains(result, "暂无结果") {
		return result, nil
	}

	// 引擎 2：Bing（国内可用性更好）
	result, err = searchBing(query)
	if err == nil && result != "" && !strings.Contains(result, "暂无结果") {
		return result, nil
	}

	// 降级：返回占位
	return fmt.Sprintf("搜索「%s」暂无结果", query), nil
}

// ─── 引擎 1：DuckDuckGo Lite ──────────────────────────────────

func searchDDG(query string) (string, error) {
	u := "https://lite.duckduckgo.com/lite/?" + url.Values{"q": {query}}.Encode()
	body, err := fetchURL(u)
	if err != nil {
		return "", err
	}

	snippets := extractDDGSnippets(body)
	if len(snippets) == 0 {
		return fmt.Sprintf("搜索「%s」暂无结果", query), nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("搜索「%s」结果：\n", query))
	for i, s := range snippets {
		if i >= maxResults {
			break
		}
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, s))
	}
	return b.String(), nil
}

func extractDDGSnippets(html string) []string {
	re := regexp.MustCompile(`class="result-snippet">(.*?)</td>`)
	matches := re.FindAllStringSubmatch(html, -1)

	var snippets []string
	for _, m := range matches {
		if len(m) >= 2 {
			text := cleanHTML(m[1])
			text = strings.TrimSpace(text)
			if text != "" && len(text) > 5 {
				snippets = append(snippets, text)
			}
		}
	}
	return snippets
}

// ─── 引擎 2：Bing ─────────────────────────────────────────────

func searchBing(query string) (string, error) {
	// 使用 cn.bing.com（国内用户首选）
	u := "https://cn.bing.com/search?" + url.Values{"q": {query}, "count": {"10"}}.Encode()
	body, err := fetchURL(u)
	if err != nil {
		// 回退国际版
		u = "https://www.bing.com/search?" + url.Values{"q": {query}, "count": {"10"}}.Encode()
		body, err = fetchURL(u)
		if err != nil {
			return "", err
		}
	}

	snippets := extractBingSnippets(body)
	if len(snippets) == 0 {
		return fmt.Sprintf("搜索「%s」暂无结果", query), nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("搜索「%s」结果：\n", query))
	for i, s := range snippets {
		if i >= maxResults {
			break
		}
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, s))
	}
	return b.String(), nil
}

func extractBingSnippets(html string) []string {
	// 策略 1：匹配 b_algo 结果块中的摘要段落
	// Bing 结果格式多样，使用多个正则 fallback
	var snippets []string

	// 模式 1：b_caption 内的 <p> 文本
	re1 := regexp.MustCompile(`class="b_caption"[^>]*>[\s\S]*?<p[^>]*>([\s\S]*?)</p>`)
	matches1 := re1.FindAllStringSubmatch(html, -1)
	for _, m := range matches1 {
		if len(m) >= 2 {
			text := cleanHTML(m[1])
			text = strings.TrimSpace(text)
			if text != "" && len(text) > 10 {
				snippets = append(snippets, text)
			}
		}
	}
	if len(snippets) > 0 {
		return snippets
	}

	// 模式 2：b_algo 块内的任意可见文本
	re2 := regexp.MustCompile(`<li class="b_algo"[^>]*>[\s\S]*?</li>`)
	blocks := re2.FindAllString(html, -1)
	for _, block := range blocks {
		text := cleanHTML(block)
		text = strings.TrimSpace(text)
		// 提取标题后的摘要文本（去掉标题本身）
		if idx := strings.Index(text, "·"); idx > 0 && idx < len(text)-1 {
			text = strings.TrimSpace(text[idx+1:])
		}
		if text != "" && len(text) > 10 {
			snippets = append(snippets, text)
		}
	}

	return snippets
}

// ─── HTTP 抓取 ────────────────────────────────────────────────

func fetchURL(rawURL string) (string, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// ─── HTML 清洗 ─────────────────────────────────────────────────

func cleanHTML(s string) string {
	s = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&#39;", "'")
	// 压缩多余空白
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
