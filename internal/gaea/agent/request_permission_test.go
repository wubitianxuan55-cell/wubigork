package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// fakePermRequester 捕获 request_permission 工具的申请并按预设返回。
type fakePermRequester struct {
	gotTool    string
	gotSubject string
	gotReason  string
	granted    bool
	decision   string
	err        error
	called     int
}

func (f *fakePermRequester) RequestPermission(ctx context.Context, tool, subject, reason string) (bool, string, error) {
	f.called++
	f.gotTool, f.gotSubject, f.gotReason = tool, subject, reason
	return f.granted, f.decision, f.err
}

func runRequestPermission(t *testing.T, ctx context.Context, args string, f *fakePermRequester) (string, error) {
	t.Helper()
	if f != nil {
		ctx = WithPermissionRequester(ctx, f)
	}
	return NewRequestPermissionTool().Execute(ctx, json.RawMessage(args))
}

// TestRequestPermissionToolForwardsAndFormats：参数解析与转发 + 授予结果文本。
func TestRequestPermissionToolForwardsAndFormats(t *testing.T) {
	for _, tc := range []struct {
		name       string
		granted    bool
		decision   string
		wantSubstr []string
	}{
		{"allow_session", true, "allow_session", []string{"allow_session", `bash(go build*)" is in effect for this session`}},
		{"allow_once", true, "allow_once", []string{"allow_once", "in effect for this session"}},
		{"persist_allow", true, "persist_allow", []string{"persist_allow", "policy file", `bash(go build*)`}},
		{"deny", false, "deny", []string{"DENIED", "deny", "Do not repeat"}},
		{"abort", false, "abort", []string{"abort", "not granted"}},
		{"timeout", false, "timeout", []string{"timeout", "not granted"}},
		{"refused_hardask", false, "refused_hardask", []string{"refused_hardask", "Do not repeat"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakePermRequester{granted: tc.granted, decision: tc.decision}
			out, err := runRequestPermission(t, context.Background(),
				`{"tool":"bash","subject":"go build*","reason":"需要跑构建验证改动"}`, f)
			if err != nil {
				t.Fatalf("Execute err = %v", err)
			}
			if f.called != 1 || f.gotTool != "bash" || f.gotSubject != "go build*" || f.gotReason != "需要跑构建验证改动" {
				t.Fatalf("转发参数不符: called=%d %+v", f.called, f)
			}
			for _, want := range tc.wantSubstr {
				if !strings.Contains(out, want) {
					t.Fatalf("结果缺少 %q:\n%s", want, out)
				}
			}
		})
	}
}

// TestRequestPermissionToolHeadless：headless（无 requester）返回非交互降级
// 结果，绝不阻塞自治运行，也绝不静默授予。
func TestRequestPermissionToolHeadless(t *testing.T) {
	out, err := runRequestPermission(t, context.Background(),
		`{"tool":"bash","subject":"go build*","reason":"构建"}`, nil)
	if err != nil {
		t.Fatalf("Execute err = %v", err)
	}
	if !strings.Contains(out, "Never-Ask") || !strings.Contains(out, "NOT applied") {
		t.Fatalf("headless 结果应为 Never-Ask 降级且声明未授予:\n%s", out)
	}
}

// TestRequestPermissionToolValidation：tool 与 reason 必填——理由缺失的申请
// 直接退回，绝不把无理由的申请递给用户。
func TestRequestPermissionToolValidation(t *testing.T) {
	f := &fakePermRequester{}
	if _, err := runRequestPermission(t, context.Background(), `{"tool":"","reason":"r"}`, f); err == nil {
		t.Fatal("缺 tool 应返回错误")
	}
	if _, err := runRequestPermission(t, context.Background(), `{"tool":"bash","reason":"   "}`, f); err == nil {
		t.Fatal("缺 reason 应返回错误")
	}
	if _, err := runRequestPermission(t, context.Background(), `{"tool":"bash"}`, f); err == nil {
		t.Fatal("缺 reason 应返回错误")
	}
	if f.called != 0 {
		t.Fatalf("校验失败不得触达 requester，called=%d", f.called)
	}
	if _, err := runRequestPermission(t, context.Background(), `not-json`, f); err == nil {
		t.Fatal("非法 JSON 应返回错误")
	}
}

// TestRequestPermissionToolRequesterError：requester 报错（ctx 取消）向上透传。
func TestRequestPermissionToolRequesterError(t *testing.T) {
	f := &fakePermRequester{err: errors.New("context canceled")}
	if _, err := runRequestPermission(t, context.Background(),
		`{"tool":"bash","reason":"构建"}`, f); err == nil {
		t.Fatal("requester 错误应向上透传")
	}
}

// TestRequestPermissionToolWholeToolRule：subject 为空时规则串为裸工具名。
func TestRequestPermissionToolWholeToolRule(t *testing.T) {
	f := &fakePermRequester{granted: true, decision: "allow_session"}
	out, err := runRequestPermission(t, context.Background(),
		`{"tool":"web_fetch","reason":"需要抓取文档页"}`, f)
	if err != nil {
		t.Fatalf("Execute err = %v", err)
	}
	if f.gotSubject != "" {
		t.Fatalf("subject 应原样为空，got %q", f.gotSubject)
	}
	if !strings.Contains(out, `"web_fetch" is in effect`) {
		t.Fatalf("整工具规则串渲染不符:\n%s", out)
	}
}
