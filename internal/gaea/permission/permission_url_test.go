package permission

// browser_navigate 的 url 键接入 Subject 匹配（subjectKeys 追加 "url"）：
// 细粒度规则 browser_navigate(<glob>) 可按 URL 精确放行/拦截。

import (
	"context"
	"encoding/json"
	"testing"
)

// TestSubjectURLKey url 键参与 subject 提取（排在最后，不遮蔽既有键）。
func TestSubjectURLKey(t *testing.T) {
	cases := []struct {
		args string
		want string
	}{
		{`{"url":"https://example.com/page"}`, "https://example.com/page"},
		{`{"url":"http://127.0.0.1:8080/"}`, "http://127.0.0.1:8080/"},
		{`{"url":""}`, ""},                        // 空串不算 subject
		{`{"path":"/a","url":"https://x"}`, "/a"}, // 既有键优先，url 不遮蔽
	}
	for _, c := range cases {
		if got := Subject(json.RawMessage(c.args)); got != c.want {
			t.Errorf("Subject(%q) = %q, want %q", c.args, got, c.want)
		}
	}
}

// TestPolicyDecideNavigateURL 按 URL 的 allow/deny/fallback 全链。
func TestPolicyDecideNavigateURL(t *testing.T) {
	p := New("ask",
		[]string{"browser_navigate(https://trusted.example.com*)"},
		nil,
		[]string{"browser_navigate(*evil.example.com*)"},
	)

	cases := []struct {
		name string
		url  string
		want Decision
	}{
		{"allow 规则按 URL 命中", "https://trusted.example.com/a/b", Allow},
		{"deny 规则优先于 allow", "https://evil.example.com/x", Deny},
		{"未匹配 URL 落入 mode(ask)", "https://unknown.example.com/", Ask},
		{"无 url 键落入 mode(ask)", "", Ask},
	}
	for _, c := range cases {
		args := json.RawMessage(`{}`)
		if c.url != "" {
			args = json.RawMessage(`{"url":"` + c.url + `"}`)
		}
		if got := p.Decide("browser_navigate", false, args); got != c.want {
			t.Errorf("%s: Decide = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestGateNavigateURLRemember 会话记忆把带 url 的放行固化为窄规则。
func TestGateNavigateURLRemember(t *testing.T) {
	var remembered string
	ap := &stubApprover{allow: true, remember: true}
	g := NewGate(New("ask", nil, nil, nil), ap)
	g.OnRemember = func(rule string) { remembered = rule }

	allow, _, err := g.Check(context.Background(), "browser_navigate",
		json.RawMessage(`{"url":"https://docs.example.com/report"}`), false)
	if err != nil || !allow {
		t.Fatalf("approved call = (%v,%v), want allow", allow, err)
	}
	if want := "browser_navigate(https://docs.example.com/report)"; remembered != want {
		t.Errorf("remembered = %q, want %q", remembered, want)
	}
}
