package agent

// wait_subagents_test.go — P1-7 回归钉：①TouchAction 进度回写（running 期刷、
// 终态不刷）②subagent_list last: 行③wait sa_ 分流（注入缝；unknown 引用如实）
// ④bgjobs wait 混合 job ids+sa_ 引用分流（builtin 侧）。

import (
	"context"
	"strings"
	"testing"
	"time"
)

func newWaitTestStore(t *testing.T) *SubagentStore {
	t.Helper()
	return NewSubagentStore(t.TempDir())
}

func TestTouchActionProgress(t *testing.T) {
	store := newWaitTestStore(t)
	run := &SubagentRun{Ref: "sa_prog1", Session: nil, Title: "编第3章"}
	if err := store.MarkRunning(run); err != nil {
		t.Fatal(err)
	}
	store.TouchAction("sa_prog1", "write_file 第3章.md")
	meta, err := store.loadMeta("sa_prog1")
	if err != nil {
		t.Fatal(err)
	}
	if meta.LastAction != "write_file 第3章.md" || meta.LastActionAt.IsZero() {
		t.Fatalf("进度未回写: %+v", meta)
	}
	// 终态后不再刷（防 stale 写回）。
	if err := store.SaveCompleted(&SubagentRun{Ref: "sa_prog1", Title: "编第3章"}); err != nil {
		t.Fatal(err)
	}
	store.TouchAction("sa_prog1", "bash 渲染")
	meta2, _ := store.loadMeta("sa_prog1")
	if meta2.LastAction != "write_file 第3章.md" || meta2.Status != SubagentCompleted {
		t.Fatalf("终态后进度不得刷新: %+v", meta2)
	}
}

func TestWaitSubagentsTerminalAndUnknown(t *testing.T) {
	old := subWaitStore
	store := newWaitTestStore(t)
	subWaitStore = store
	defer func() { subWaitStore = old }()
	store.MarkRunning(&SubagentRun{Ref: "sa_w1", Title: "任务一"})
	store.SaveCompleted(&SubagentRun{Ref: "sa_w1", Title: "任务一"})
	res := WaitSubagents(context.Background(), []string{"sa_w1", "sa_gone"}, time.Second)
	if len(res) != 2 {
		t.Fatalf("应返回 2 条: %+v", res)
	}
	byRef := map[string]SubagentWaitResult{}
	for _, r := range res {
		byRef[r.Ref] = r
	}
	if byRef["sa_w1"].Status != "completed" || !strings.Contains(byRef["sa_w1"].Output, "自动回投") {
		t.Fatalf("终态行缺取结果指引: %+v", byRef["sa_w1"])
	}
	if byRef["sa_gone"].Status != "unknown" {
		t.Fatalf("未知引用应 unknown: %+v", byRef["sa_gone"])
	}
}
