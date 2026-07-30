// Package whisper — web_search.go
// 简化版 Web 搜索：DuckDuckGo Lite HTML 抓取
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

// WebSearch 执行 Web 搜索（DuckDuckGo Lite）
func WebSearch(query string) (string, error) {
	u := "https://lite.duckduckgo.com/lite/?"
	u += url.Values{"q": {query}}.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", err
	}

	html := string(body)
	snippets := extractDuckDuckGoSnippets(html)
	if len(snippets) == 0 {
		return fmt.Sprintf("搜索「%s」暂无结果", query), nil
	}

	result := fmt.Sprintf("搜索「%s」结果：\n", query)
	for i, s := range snippets {
		if i >= 5 {
			break
		}
		result += fmt.Sprintf("%d. %s\n", i+1, s)
	}
	return result, nil
}

// extractDuckDuckGoSnippets 从 DuckDuckGo Lite HTML 提取摘要
func extractDuckDuckGoSnippets(html string) []string {
	// 匹配 class="result-snippet" 内容
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

// cleanHTML 去除简单 HTML 标签
func cleanHTML(s string) string {
	s = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	return s
}
