package builtin

import (
	"context"
	"testing"

	"github.com/gaea/gaea/internal/gaea/config"
)

// TestSearchPolicyRestricted_None 无配置/全空列表均不视为受限（行为零变化）。
func TestSearchPolicyRestricted_None(t *testing.T) {
	saveSearchGlobals(t)
	searchCfg = nil
	if searchPolicyRestricted() {
		t.Error("searchCfg=nil 不应受限")
	}
	SetSearchConfig(config.SearchConfig{})
	if searchPolicyRestricted() {
		t.Error("allow/deny 均空不应受限")
	}
}

// TestSearchPolicyRestricted_Configured allow 或 deny 任一非空即受限（unsloth over-fetch 前提）。
func TestSearchPolicyRestricted_Configured(t *testing.T) {
	saveSearchGlobals(t)
	SetSearchConfig(config.SearchConfig{AllowDomains: []string{"*.gov.cn"}})
	if !searchPolicyRestricted() {
		t.Error("allow 非空应受限")
	}
	SetSearchConfig(config.SearchConfig{DenyDomains: []string{"denied.com"}})
	if !searchPolicyRestricted() {
		t.Error("deny 非空应受限")
	}
}

// TestFilterSearchResults_NoPolicy 无策略时原样返回并截断到 limit。前 3 条全保留。
func TestFilterSearchResults_NoPolicy(t *testing.T) {
	saveSearchGlobals(t)
	SetSearchConfig(config.SearchConfig{})
	res := []SearchResult{
		{Title: "a", URL: "https://a.com/1"},
		{Title: "b", URL: "https://b.com/2"},
		{Title: "c", URL: "https://c.com/3"},
	}
	got := filterSearchResults(res, 3)
	if len(got) != 3 {
		t.Fatalf("want 3, got %d", len(got))
	}
	if got[0].URL != "https://a.com/1" || got[2].URL != "https://c.com/3" {
		t.Errorf("原样顺序被改动: %+v", got)
	}
	// 超过 limit 应截断
	got2 := filterSearchResults(res, 2)
	if len(got2) != 2 {
		t.Errorf("limit=2 应截断到2, got %d", len(got2))
	}
}

// TestFilterSearchResults_AllowDeny 受限时：deny 优先、allow 非空须匹配；丢弃被拒 url。
func TestFilterSearchResults_AllowDeny(t *testing.T) {
	saveSearchGlobals(t)
	SetSearchConfig(config.SearchConfig{
		AllowDomains: []string{"allowed.com"},
		DenyDomains:  []string{"denied.com"},
	})
	res := []SearchResult{
		{Title: "1", URL: "https://allowed.com/a"},
		{Title: "2", URL: "https://denied.com/b"},
		{Title: "3", URL: "https://allowed.com/c"},
		{Title: "4", URL: "https://other.com/d"},
		{Title: "5", URL: "https://allowed.com/e"},
	}
	got := filterSearchResults(res, 5)
	if len(got) != 3 {
		t.Fatalf("want 3 (allowed/a,c,e), got %d: %+v", len(got), got)
	}
	for _, r := range got {
		if r.URL == "https://denied.com/b" || r.URL == "https://other.com/d" {
			t.Errorf("被拒 url 未过滤: %+v", got)
		}
	}
	// limit 生效：只留前两条合规结果
	got2 := filterSearchResults(res, 2)
	if len(got2) != 2 || got2[0].URL != "https://allowed.com/a" || got2[1].URL != "https://allowed.com/c" {
		t.Errorf("limit=2 过滤结果: %+v", got2)
	}
}

// TestFilterSearchResults_InvalidURL 非法 url 应被跳过而非报错（宁缺勿错）。
func TestFilterSearchResults_InvalidURL(t *testing.T) {
	saveSearchGlobals(t)
	SetSearchConfig(config.SearchConfig{DenyDomains: []string{"blocked.com"}})
	res := []SearchResult{
		{Title: "bad", URL: "not a url"},
		{Title: "ok", URL: "https://ok.com/x"},
	}
	got := filterSearchResults(res, 5)
	if len(got) != 1 || got[0].URL != "https://ok.com/x" {
		t.Errorf("非法 url 应被跳过: %+v", got)
	}
}

// TestFetchSearchPage_BadURL url 模式对非法协议同步报错（不发起网络请求）。
func TestFetchSearchPage_BadURL(t *testing.T) {
	saveSearchGlobals(t)
	if _, err := fetchSearchPage(context.Background(), "ftp://example.com/x"); err == nil {
		t.Error("ftp:// 应被拒绝（仅 http/https）")
	}
	if _, err := fetchSearchPage(context.Background(), "not a url"); err == nil {
		t.Error("非 url 应被拒绝")
	}
}
