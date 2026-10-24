package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// fmtSscanf 从页尾提示里取下一页偏移："(continued: call again with offset_bytes=N; M bytes remain)"。
func fmtSscanf(page string, next *int64) (int, error) {
	i := strings.Index(page, "offset_bytes=")
	if i < 0 {
		return 0, fmt.Errorf("no next-offset hint")
	}
	rest := page[i+len("offset_bytes="):]
	j := strings.Index(rest, ";")
	if j < 0 {
		return 0, fmt.Errorf("malformed next-offset hint")
	}
	n, err := strconv.ParseInt(strings.TrimSpace(rest[:j]), 10, 64)
	if err != nil {
		return 0, err
	}
	*next = n
	return 1, nil
}

// newResultStore 建一个落在临时目录的持久 store，并返回读工具。
func newResultStore(t *testing.T) (*SubagentStore, *ReadSubagentResultTool) {
	t.Helper()
	store := NewSubagentStore(t.TempDir())
	return store, NewReadSubagentResultTool(store)
}

func finishRun(t *testing.T, store *SubagentStore, run *SubagentRun, answer string, status SubagentStatus) {
	t.Helper()
	run.Session.Add(provider.Message{Role: provider.RoleUser, Content: "do the thing"})
	run.Session.Add(provider.Message{Role: provider.RoleAssistant, Content: answer})
	if status == SubagentCompleted {
		if err := store.SaveCompleted(run); err != nil {
			t.Fatalf("SaveCompleted: %v", err)
		}
		return
	}
	if err := store.SaveFailed(run); err != nil {
		t.Fatalf("SaveFailed: %v", err)
	}
}

// ageMeta 把 ref 的 meta.UpdatedAt 改写到过去（生产里假 running 的形态=
// 崩溃/强退后停更的侧车）。
func ageMeta(t *testing.T, store *SubagentStore, ref string) {
	t.Helper()
	b, err := os.ReadFile(store.metaPath(ref))
	if err != nil {
		t.Fatal(err)
	}
	var stale map[string]any
	if err := json.Unmarshal(b, &stale); err != nil {
		t.Fatal(err)
	}
	stale["updatedAt"] = time.Now().Add(-2 * staleRunningAfter).Format(time.RFC3339Nano)
	nb, err := json.Marshal(stale)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.metaPath(ref), nb, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReadSubagentResultCompleted(t *testing.T) {
	store, tool := newResultStore(t)
	run, err := store.PrepareFresh("sub sys", "work")
	if err != nil {
		t.Fatal(err)
	}
	finishRun(t, store, run, "the answer is 42", SubagentCompleted)

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"ref":"`+run.Ref+`"}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "the answer is 42") || !strings.Contains(out, `status="completed"`) {
		t.Fatalf("out = %q, want answer with completed status", out)
	}
	if strings.Contains(out, "offset_bytes=") {
		t.Fatalf("single-page read must not advertise a next page: %q", out)
	}
}

func TestReadSubagentResultPaging(t *testing.T) {
	store, tool := newResultStore(t)
	run, _ := store.PrepareFresh("sub sys", "work")
	long := strings.Repeat("汉", 9000) // 3 bytes/rune → 27000 bytes > default page
	finishRun(t, store, run, long, SubagentCompleted)

	// 沿 next_offset 链翻到收敛：总长 27000B、每页 12288B → 恰好 3 页；
	// 页界按 rune 对齐（12288 是 3 的倍数，此处自然对齐），拼接不得出现劈字乱码。
	var pages []string
	offset := int64(0)
	for i := 0; ; i++ {
		if i > 10 {
			t.Fatal("pagination did not converge")
		}
		page, err := tool.Execute(context.Background(), json.RawMessage(
			`{"ref":"`+run.Ref+`","offset_bytes":`+itoa(int(offset))+`}`))
		if err != nil {
			t.Fatal(err)
		}
		pages = append(pages, page)
		if !strings.Contains(page, "bytes remain") {
			break
		}
		var next int64
		if _, err := fmtSscanf(page, &next); err != nil {
			t.Fatalf("parse next offset from %q: %v", page[strings.Index(page, "(continued):"):], err)
		}
		if next <= offset {
			t.Fatalf("next offset must advance: %d -> %d", offset, next)
		}
		offset = next
	}
	if len(pages) != 3 {
		t.Fatalf("pages = %d, want 3", len(pages))
	}
	for _, page := range pages {
		if strings.Contains(page, "\uFFFD") {
			t.Fatal("page boundary split a multi-byte rune")
		}
	}

	over, err := tool.Execute(context.Background(), json.RawMessage(`{"ref":"`+run.Ref+`","offset_bytes":999999}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(over, "beyond the end") {
		t.Fatalf("out-of-range offset must be reported honestly: %q", over)
	}
}

func TestReadSubagentResultFailedIsPartial(t *testing.T) {
	store, tool := newResultStore(t)
	run, _ := store.PrepareFresh("sub sys", "work")
	finishRun(t, store, run, "got halfway", SubagentFailed)

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"ref":"`+run.Ref+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "PARTIAL RESULT") || !strings.Contains(out, "got halfway") {
		t.Fatalf("failed run read must be explicitly annotated as partial: %q", out)
	}
}

func TestReadSubagentResultLiveRunning(t *testing.T) {
	store, tool := newResultStore(t)
	run, _ := store.PrepareFresh("sub sys", "work")
	if err := store.MarkRunning(run); err != nil {
		t.Fatal(err)
	}
	subRuns.Store(run.Ref, &subRunHandle{cancel: func() {}})
	defer subRuns.Delete(run.Ref)

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"ref":"`+run.Ref+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "still running in this process") {
		t.Fatalf("live run must be reported as such: %q", out)
	}
}

func TestReadSubagentResultStaleRunning(t *testing.T) {
	store, tool := newResultStore(t)
	run, _ := store.PrepareFresh("sub sys", "work")
	if err := store.MarkRunning(run); err != nil {
		t.Fatal(err)
	}
	ageMeta(t, store, run.Ref) // 无 subRuns 句柄 + 停更 = 假 running

	out, err := tool.Execute(context.Background(), json.RawMessage(`{"ref":"`+run.Ref+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "interrupted by a restart") || !strings.Contains(out, "continue_from") {
		t.Fatalf("stale run must be reported as restart-interrupted: %q", out)
	}
}

func TestReadSubagentResultUnreachableRef(t *testing.T) {
	_, tool := newResultStore(t)
	out, err := tool.Execute(context.Background(), json.RawMessage(`{"ref":"sa_20990101_000000_00000000_dead"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "No subagent") {
		t.Fatalf("unknown ref must give the honest not-found message: %q", out)
	}
	out2, err := tool.Execute(context.Background(), json.RawMessage(`{"ref":"mt_20990101_000000_00000000_dead"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out2, "model-tool run") {
		t.Fatalf("mt_ ref must explain inline return: %q", out2)
	}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"ref":"../../etc/passwd"}`)); err == nil {
		t.Fatal("non-sa_ ref must be rejected")
	}
}

// TestSubagentSnapshotMeta 钉 B1 出发快照的一次写定语义：fresh 派发写实
// workspaceRoot/systemPromptHash/toolScope/model/effort；continue 保留原值
// 并记 resumedFrom。
func TestSubagentSnapshotMeta(t *testing.T) {
	store, _ := newResultStore(t)
	tt := &TaskTool{sysPrompt: "task sys prompt"}
	tt.SetSubagentIdentity("glm-5.3-flash", "high")

	run, err := store.PrepareFresh(tt.sysPrompt, "work")
	if err != nil {
		t.Fatal(err)
	}
	run.Model, run.Effort = tt.subModel, tt.subEffort
	run.ToolScope = sortedToolNames([]string{"zread", "aread", "mread"})
	finishRun(t, store, run, "done", SubagentCompleted)

	meta, err := store.loadMeta(run.Ref)
	if err != nil {
		t.Fatal(err)
	}
	if meta.WorkspaceRoot == "" || meta.SystemPromptHash == "" || meta.Model != "glm-5.3-flash" || meta.Effort != "high" {
		t.Fatalf("fresh snapshot fields missing: %+v", meta)
	}
	if got := meta.ToolScope; len(got) != 3 || got[0] != "aread" || got[2] != "zread" {
		t.Fatalf("ToolScope = %v, want sorted 3 names", got)
	}
	if meta.SystemPromptHash != sysPromptHash("task sys prompt") {
		t.Fatalf("SystemPromptHash mismatch: %q", meta.SystemPromptHash)
	}

	// continue：身份字段保原值，resumedFrom 记源 ref。
	cont, err := store.PrepareContinue(run.Ref, "work")
	if err != nil {
		t.Fatal(err)
	}
	cont.Model, cont.Effort = "other-model", "low" // 派发方换了身份也不得篡改出发快照
	if err := store.MarkRunning(cont); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveCompleted(cont); err != nil {
		t.Fatal(err)
	}
	meta2, err := store.loadMeta(run.Ref)
	if err != nil {
		t.Fatal(err)
	}
	if meta2.Model != "glm-5.3-flash" || meta2.Effort != "high" {
		t.Fatalf("continue must keep the original snapshot identity: %+v", meta2)
	}
	if meta2.ResumedFrom != run.Ref {
		t.Fatalf("ResumedFrom = %q, want %q", meta2.ResumedFrom, run.Ref)
	}
}

// TestSubagentListStaleAnnotation 钉 subagent_list 的假 running 标注。
func TestSubagentListStaleAnnotation(t *testing.T) {
	store, _ := newResultStore(t)
	run, _ := store.PrepareFresh("sub sys", "work")
	if err := store.MarkRunning(run); err != nil {
		t.Fatal(err)
	}
	ageMeta(t, store, run.Ref)

	list := NewSubagentListTool(store)
	out, err := list.Execute(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[stale: no live handle") {
		t.Fatalf("stale running entry must be annotated: %q", out)
	}
	if !strings.Contains(out, "read_subagent_result") {
		t.Fatalf("list hint should mention the new read tool: %q", out)
	}
}
