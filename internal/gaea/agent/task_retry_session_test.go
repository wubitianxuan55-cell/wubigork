package agent

// GA1-09 回归钉子：retry_until 的重试必须续用同一会话——失败历史逐尝试累积，
// 子代理第二次请求必须能看到第一次的 assistant 回复。改前 run==nil（ephemeral
// 模式，无 transcript store）时每次重试都是全新会话，与 runSubWithRetrySession
// 的文档语义相反（死分支 `run == nil && subSession != nil` 恒不可达）。

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/gaea/gaea/internal/gaea/provider"
	"github.com/gaea/gaea/internal/gaea/tool"
)

// flakyCheckBash 前 failTimes 次 Execute 返回失败（check 未通过 → 触发重试），
// 之后放行成功——模拟「check 命令首败后过」的 retry_until 场景。命令文本经
// fakeTool 原样记录供断言。
type flakyCheckBash struct {
	fakeTool
	failTimes int
	calls     int
}

func (b *flakyCheckBash) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	b.calls++
	if b.calls <= b.failTimes {
		return "FAIL: 1 test broken", errors.New("exit status 1")
	}
	return b.fakeTool.Execute(ctx, args)
}

// TestRetryUntilSameSessionAccumulatesHistory run==nil（ephemeral）路径：首试
// check 失败、二试通过。断言第二次子代理请求包含第一次的 assistant 回复——
// 同会话累积的直接观测。回归（改坏能红）：若恢复「每次重试新会话」，第二条
// 请求只剩 [system, user 重试提示]，找不到首试回复。
func TestRetryUntilSameSessionAccumulatesHistory(t *testing.T) {
	rp := newRecordingScripted([][]provider.Chunk{
		{{Type: provider.ChunkText, Text: "attempt one answer"}, {Type: provider.ChunkDone}},
		{{Type: provider.ChunkText, Text: "attempt two answer"}, {Type: provider.ChunkDone}},
	})
	bash := &flakyCheckBash{fakeTool: fakeTool{name: "bash", readOnly: false}, failTimes: 1}
	parentReg := tool.NewRegistry()
	parentReg.Add(bash)
	task := NewTaskTool(rp, nil, parentReg, 20, 0, 0.0, "", "sys", nil)
	parentReg.Add(task) // 无 transcripts store ⇒ prepareRun 返回 run==nil（ephemeral）

	out, err := task.Execute(context.Background(),
		[]byte(`{"prompt":"fix the build","retry_until":{"check":"go test ./...","max_retries":3}}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "attempt two answer") {
		t.Fatalf("out = %q, want the second attempt's answer", out)
	}
	if bash.calls != 2 {
		t.Fatalf("check command ran %d times, want exactly 2 (fail then pass)", bash.calls)
	}
	if len(rp.reqs) != 2 {
		t.Fatalf("sub-agent made %d provider requests, want 2 (one per attempt)", len(rp.reqs))
	}

	// 首试请求形状：[system "sys", user 任务 prompt]。
	first := rp.reqs[0]
	if len(first.Messages) != 2 ||
		first.Messages[0].Content != "sys" ||
		lastUser(first) != "fix the build" {
		t.Fatalf("first request messages = %+v, want [sys, task prompt]", first.Messages)
	}

	// 核心断言：第二次请求包含第一次的 assistant 回复（同会话累积）+ 重试提示。
	second := rp.reqs[1]
	sawFirstAnswer := false
	for _, m := range second.Messages {
		if m.Role == provider.RoleAssistant && m.Content == "attempt one answer" {
			sawFirstAnswer = true
		}
	}
	if !sawFirstAnswer {
		t.Fatalf("second request must contain the first attempt's assistant reply (same-session accumulation), got messages:\n%+v", second.Messages)
	}
	if lu := lastUser(second); !strings.Contains(lu, "Previous attempt failed the verification") || !strings.Contains(lu, "go test ./...") {
		t.Fatalf("second request's last user message = %q, want the retry nudge with check output", lu)
	}
	if len(second.Messages) != 4 {
		t.Fatalf("second request carries %d messages, want 4 ([sys, user, assistant, retry user]) — seed re-injection or a session reset would change this", len(second.Messages))
	}
}
