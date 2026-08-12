// Package whisper — web_search.go
// 聊天/轻语联网搜索：Bing（国内可用）优先 → DuckDuckGo Lite 兜底。
// 返回带标题、链接、摘要的结果文本，供 LLM 参考回答。

package whisper

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/netclient"
)

// httpClient 共享 HTTP 客户端：自动模式跟随系统/环境代理，
// 这样开启 VPN 梯子（系统代理或 TUN）时聊天搜索同样能出网。
var httpClient = func() *http.Client {
	c, err := netclient.NewHTTPClient(netclient.ProxySpec{Mode: netclient.ModeAuto}, netclient.TransportOptions{})
	if err != nil {
		return netclient.NewSimpleClient(12 * time.Second)
	}
	c.Timeout = 12 * time.Second
	return c
}()

// userAgent 浏览器 UA（避免被反爬拦截）
const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"

// maxResults 单次搜索最多返回条数
const maxResults = 5

// WebSearch 执行 Web 搜索：Bing 优先（国内直连可用），失败降级到 DuckDuckGo Lite。
func WebSearch(query string) (string, error) {
	// 口语化提问先清洗成关键词查询，避免 Bing 按整句话检索命中无关词条
	// （例如「最近几天天气怎样」原样搜索会返回“最近”的百科/歌曲）。
	query = cleanSearchQuery(query)
	if result, err := searchBing(query); err == nil && result != "" && !strings.Contains(result, "暂无结果") {
		return result, nil
	}
	if result, err := searchDDG(query); err == nil && result != "" && !strings.Contains(result, "暂无结果") {
		return result, nil
	}
	return fmt.Sprintf("搜索「%s」暂无结果", query), nil
}

// cleanSearchQuery 从口语化提问中提取适合搜索引擎的关键词：
// 去掉触发前缀（帮我搜索/查一下…）、疑问词（什么是/如何/为什么…）、
// 结尾语气词（怎样/怎么样/吗/呢…）与礼貌用语（请/麻烦/帮我），
// 再归一化空白与标点。清洗后为空时回退原文。
func cleanSearchQuery(msg string) string {
	s := strings.TrimSpace(msg)

	// 去掉开头的“帮/请/查”类触发前缀
	for _, p := range []string{
		"帮我搜索一下", "帮我搜一下", "帮我查一下", "请帮我查一下", "搜索一下",
		"帮我搜索", "帮我查", "帮我搜", "上网查一下", "上网查",
		"搜一下", "查一下", "查一查", "查查", "搜索", "查询",
	} {
		if strings.HasPrefix(s, p) {
			s = strings.TrimSpace(s[len(p):])
			break
		}
	}

	// 去掉开头的疑问词（什么是X → X）
	for _, p := range []string{"什么是", "啥是", "怎么样", "怎样", "如何", "为什么", "怎么"} {
		if strings.HasPrefix(s, p) {
			s = strings.TrimSpace(s[len(p):])
			break
		}
	}

	// 去掉结尾的疑问/口语后缀（X怎么样 → X）
	for _, suf := range []string{
		"怎么样了", "怎么样呢", "怎么样", "怎样", "怎么办", "怎么弄",
		"是什么", "是啥", "在哪里", "在哪儿", "什么时候", "多少钱",
		"有没有", "有哪些", "多少", "吗", "呢", "啊", "呀", "吧",
	} {
		for strings.HasSuffix(s, suf) {
			s = strings.TrimSpace(strings.TrimSuffix(s, suf))
		}
	}

	// 去掉常见礼貌词/冗余词（注意不能用单字“请”，会误删“申请/请假”等）
	for _, w := range []string{"麻烦", "帮我", "我想知道", "告诉我", "介绍一下"} {
		s = strings.ReplaceAll(s, w, "")
	}

	// 归一化空白并去掉首尾标点
	s = strings.Join(strings.Fields(s), " ")
	s = strings.Trim(s, " ，。？！、,.?!:：;；\"'“”‘’()（）")
	if s == "" {
		return msg
	}
	return s
}

// ─── 引擎 1：Bing ───

func searchBing(query string) (string, error) {
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
	results := extractBingResults(body)
	if len(results) == 0 {
		return fmt.Sprintf("搜索「%s」暂无结果", query), nil
	}
	return formatResults(query, results), nil
}

var (
	bingBlockRe = regexp.MustCompile(`<li class="b_algo[^"]*"[^>]*>([\s\S]*?)</li>`)
	bingTitleRe = regexp.MustCompile(`<h2[^>]*>\s*<a[^>]*href="([^"]+)"[^>]*>([\s\S]*?)</a>`)
	bingSnipRe  = regexp.MustCompile(`class="b_caption"[^>]*>[\s\S]*?<p[^>]*>([\s\S]*?)</p>`)
)

type webResult struct {
	Title   string
	URL     string
	Snippet string
}

func extractBingResults(body string) []webResult {
	var results []webResult
	for _, m := range bingBlockRe.FindAllStringSubmatch(body, -1) {
		if len(m) < 2 {
			continue
		}
		block := m[1]
		title, href := "", ""
		if tm := bingTitleRe.FindStringSubmatch(block); len(tm) >= 3 {
			href = strings.TrimSpace(tm[1])
			title = cleanHTML(tm[2])
		}
		if title == "" || href == "" || strings.HasPrefix(href, "javascript:") {
			continue
		}
		snippet := ""
		if sm := bingSnipRe.FindStringSubmatch(block); len(sm) >= 2 {
			snippet = cleanHTML(sm[1])
		}
		results = append(results, webResult{Title: title, URL: href, Snippet: snippet})
		if len(results) >= maxResults {
			break
		}
	}
	return results
}

// ─── 引擎 2：DuckDuckGo Lite ───

func searchDDG(query string) (string, error) {
	u := "https://lite.duckduckgo.com/lite/?" + url.Values{"q": {query}}.Encode()
	body, err := fetchURL(u)
	if err != nil {
		return "", err
	}
	results := extractDDGResults(body)
	if len(results) == 0 {
		return fmt.Sprintf("搜索「%s」暂无结果", query), nil
	}
	return formatResults(query, results), nil
}

var (
	ddgLinkRe = regexp.MustCompile(`<a[^>]*class="result-link"[^>]*href="([^"]+)"[^>]*>([\s\S]*?)</a>`)
	ddgSnipRe = regexp.MustCompile(`class="result-snippet">(.*?)</td>`)
)

func extractDDGResults(body string) []webResult {
	var links []webResult
	for _, m := range ddgLinkRe.FindAllStringSubmatch(body, -1) {
		if len(m) < 3 {
			continue
		}
		href := strings.TrimSpace(m[1])
		title := cleanHTML(m[2])
		if href == "" || title == "" {
			continue
		}
		links = append(links, webResult{Title: title, URL: href})
	}
	snippets := ddgSnipRe.FindAllStringSubmatch(body, -1)
	for i := range links {
		if i < len(snippets) && len(snippets[i]) >= 2 {
			links[i].Snippet = cleanHTML(snippets[i][1])
		}
	}
	if len(links) > maxResults {
		links = links[:maxResults]
	}
	return links
}

// formatResults 输出为带标题/链接/摘要的编号列表，便于 LLM 引用。
func formatResults(query string, results []webResult) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("搜索「%s」结果：\n", query))
	for i, r := range results {
		b.WriteString(fmt.Sprintf("%d. %s\n   %s\n   %s\n", i+1, r.Title, r.URL, r.Snippet))
	}
	return strings.TrimSpace(b.String())
}

// ─── HTTP 抓取 ───

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

// ─── HTML 清洗 ───

func cleanHTML(s string) string {
	s = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	// 压缩多余空白
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
