package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/tool"
)

// 实测反馈⑥：bash 空参数错误必须报实收参数面——此前连续 3 次
// "command is required" 空转，第一次就应暴露「0 键」vs「键名错」两种形态。
func TestBash_EmptyCommandReportsReceivedKeys(t *testing.T) {
	b := bash{}
	cases := []struct {
		name string
		args string
		want string
	}{
		{"空对象", `{}`, "0 个参数键"},
		{"键名错", `{"cmd":"echo hi"}`, "1 个参数键（cmd）"},
		{"command 空串", `{"command":"","output_format":"json"}`, "2 个参数键"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := b.Execute(context.Background(), json.RawMessage(tc.args))
			if err == nil {
				t.Fatal("空 command 必须报错")
			}
			f, ok := err.(*tool.Failure)
			if !ok || f.Code != tool.CodeValidationError {
				t.Fatalf("应为 CodeValidationError，实得 %v", err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("错误应含 %q（实收参数面），实得: %v", tc.want, err)
			}
			if !strings.Contains(err.Error(), "command is required") {
				t.Fatalf("保留语义主干，实得: %v", err)
			}
			// 2026-11-21 反馈 P1-8 补刀：原始参数体必须回显——键数定位不了
			// 「参数在到达工具前就丢了」的传输面丢参（回显走 %q 转义口径）。
			wantEcho := fmt.Sprintf("原始参数回显（%d 字节）：%q", len(tc.args), tc.args)
			if !strings.Contains(err.Error(), wantEcho) {
				t.Fatalf("错误应回显原始参数体 %q，实得: %v", wantEcho, err)
			}
		})
	}
}
