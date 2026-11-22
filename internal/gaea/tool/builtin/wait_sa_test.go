package builtin

// wait_sa_test.go — P1-7 wait 分流回归钉：sa_/mt_ 引用走注入缝（未接线=旧行为
// 只认 job ids；接线后混合列表两边都等）。

import (
	"context"
	"encoding/json"
	"testing"
)

func TestWaitSplitsSubagentRefs(t *testing.T) {
	old := subagentWaiter
	defer func() { subagentWaiter = old }()
	subagentWaiter = func(ctx context.Context, refs []string, timeoutSec int) []SubagentWaitRow {
		if len(refs) != 2 || refs[0] != "sa_x1" || refs[1] != "mt_y2" {
			t.Fatalf("子代理引用应分流到注入缝，实得 %v", refs)
		}
		return []SubagentWaitRow{{Ref: refs[0], Status: "completed", Output: "完成——read_subagent_result"}}
	}
	out, err := waitJob{}.Execute(context.Background(), json.RawMessage(`{"job_ids":["sa_x1","mt_y2"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if !contains(out, "[sa_x1] completed") || !contains(out, "read_subagent_result") {
		t.Fatalf("wait 输出缺子代理结论: %s", out)
	}
	if contains(out, "No background jobs") {
		t.Fatalf("有子代理引用时不得报空: %s", out)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && stringContains(s, sub))
}

func stringContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
