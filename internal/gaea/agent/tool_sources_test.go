package agent

import "testing"

func TestToolSourcesWebSearch(t *testing.T) {
	out := `[
	  {"title":"Go 1.24","url":"https://go.dev/doc/go1.24","snippet":"release notes","source":"bing"},
	  {"title":"重复同一页","url":"https://go.dev/doc/go1.24/","snippet":"dup","source":"tavily"},
	  {"title":"bad","url":"javascript:alert(1)","source":"x"}
	]`
	got := toolSources("web_search", `{"query":"go 1.24"}`, out)
	if len(got) != 1 {
		t.Fatalf("sources = %+v, want one normalized source", got)
	}
	if got[0].Title != "Go 1.24" || got[0].URL != "https://go.dev/doc/go1.24" || got[0].Source != "bing" {
		t.Fatalf("source = %+v", got[0])
	}
}

func TestToolSourcesURLModes(t *testing.T) {
	for _, name := range []string{"web_search", "web_fetch"} {
		got := toolSources(name, `{"url":"https://example.com/a"}`, "page body")
		if len(got) != 1 || got[0].URL != "https://example.com/a" {
			t.Fatalf("%s sources = %+v", name, got)
		}
	}
	if got := toolSources("web_fetch", `{"url":"file:///etc/passwd"}`, "x"); len(got) != 0 {
		t.Fatalf("non-http source should be rejected: %+v", got)
	}
	if got := successfulToolSources("web_fetch", `{"url":"https://example.com/a"}`, "network failed", "network failed"); len(got) != 0 {
		t.Fatalf("failed fetch must not claim evidence: %+v", got)
	}
}
