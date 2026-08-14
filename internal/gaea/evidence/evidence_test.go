package evidence

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func TestReceiptFromToolCallBash(t *testing.T) {
	r := ReceiptFromToolCall("bash", json.RawMessage(`{"command":"  ls -la  "}`), true, false)
	if r.Command != "ls -la" {
		t.Errorf("Command = %q, want trimmed ls -la", r.Command)
	}
	if r.Write || r.Read {
		t.Errorf("bash receipt must be neither write nor read, got %+v", r)
	}
}

func TestReceiptFromToolCallWrite(t *testing.T) {
	r := ReceiptFromToolCall("write_file", json.RawMessage(`{"file_path":"src/a.go","content":"x"}`), true, false)
	if !r.Write {
		t.Error("write_file must set Write")
	}
	if len(r.Paths) != 1 || r.Paths[0] != "src/a.go" {
		t.Errorf("Paths = %v, want [src/a.go]", r.Paths)
	}

	r2 := ReceiptFromToolCall("edit_file", json.RawMessage(`{"paths":["a.go","b.go"]}`), true, false)
	if !r2.Write || len(r2.Paths) != 2 {
		t.Errorf("edit_file paths = %v, want 2", r2.Paths)
	}
}

func TestReceiptFromToolCallRead(t *testing.T) {
	r := ReceiptFromToolCall("read_file", json.RawMessage(`{"path":"src/a.go"}`), true, false)
	if !r.Read || r.Write {
		t.Errorf("read_file = %+v, want Read only", r)
	}

	// unknown tool + readOnly flag + paths → Read
	r2 := ReceiptFromToolCall("preview", json.RawMessage(`{"path":"doc.md"}`), true, true)
	if !r2.Read {
		t.Error("readOnly tool with paths must be marked Read")
	}
}

func TestReceiptFromToolCallCompleteStep(t *testing.T) {
	r := ReceiptFromToolCall("complete_step", json.RawMessage(`{"step":" 2 "}`), true, false)
	if r.Step != "2" {
		t.Errorf("Step = %q, want trimmed 2", r.Step)
	}
	r2 := ReceiptFromToolCall("complete_step", json.RawMessage(`{"step_index":3}`), true, false)
	if r2.Step != "3" {
		t.Errorf("step_index Step = %q, want 3", r2.Step)
	}
	r3 := ReceiptFromToolCall("complete_step", json.RawMessage(`{"step_index":0}`), true, false)
	if r3.Step != "" {
		t.Errorf("step_index 0 must be ignored, got %q", r3.Step)
	}
}

func TestReceiptFromToolCallTodoWrite(t *testing.T) {
	r := ReceiptFromToolCall("todo_write", json.RawMessage(`{"todos":[{"content":"  Fix bug  ","status":"completed"}]}`), true, false)
	if len(r.Todos) != 1 {
		t.Fatalf("Todos = %+v, want 1", r.Todos)
	}
	if r.Todos[0].Content != "Fix bug" || r.Todos[0].Status != "completed" {
		t.Errorf("Todos[0] = %+v, want trimmed content + completed", r.Todos[0])
	}
}

func TestLedgerHasSuccessfulCommand(t *testing.T) {
	l := NewLedger()
	l.Record(Receipt{ToolName: "bash", Command: "echo hi", Success: true})
	l.Record(Receipt{ToolName: "bash", Command: "echo fail", Success: false})
	if !l.HasSuccessfulCommand("echo hi") {
		t.Error("successful command must match")
	}
	if l.HasSuccessfulCommand("echo fail") {
		t.Error("failed command must not match")
	}
	if !l.HasSuccessfulCommand("  echo hi  ") {
		t.Error("query must be trimmed before matching")
	}
	if l.HasSuccessfulCommand("") {
		t.Error("empty command must not match")
	}
}

func TestLedgerSuccessfulCommandsDedup(t *testing.T) {
	l := NewLedger()
	l.Record(Receipt{ToolName: "bash", Command: "a", Success: true})
	l.Record(Receipt{ToolName: "bash", Command: "a", Success: true})
	l.Record(Receipt{ToolName: "bash", Command: "b", Success: true})
	l.Record(Receipt{ToolName: "bash", Command: "c", Success: false})
	got := l.SuccessfulCommands(10)
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SuccessfulCommands = %v, want %v (dedup, exclude failures)", got, want)
	}
	if got := l.SuccessfulCommands(1); !reflect.DeepEqual(got, []string{"a"}) {
		t.Errorf("SuccessfulCommands(1) = %v, want [a]", got)
	}
}

func TestLedgerHasSuccessfulWritePaths(t *testing.T) {
	l := NewLedger()
	l.Record(Receipt{ToolName: "write_file", Paths: []string{"src/sub/a.go"}, Success: true, Write: true})
	if !l.HasSuccessfulWrite([]string{"src/sub/a.go"}) {
		t.Error("written path must match")
	}
	if l.HasSuccessfulWrite([]string{"src/sub/b.go"}) {
		t.Error("unwritten path must not match")
	}
	// partial coverage fails
	l.Record(Receipt{ToolName: "write_file", Paths: []string{"src/sub/c.go"}, Success: true, Write: true})
	if l.HasSuccessfulWrite([]string{"src/sub/a.go", "src/sub/b.go"}) {
		t.Error("partial path coverage must fail")
	}
	// failed receipt never counts
	l.Record(Receipt{ToolName: "write_file", Paths: []string{"src/sub/d.go"}, Success: false, Write: true})
	if l.HasSuccessfulWrite([]string{"src/sub/d.go"}) {
		t.Error("failed write receipt must not count")
	}
}

func TestLedgerHasSuccessfulReadOrWrite(t *testing.T) {
	l := NewLedger()
	l.Record(Receipt{ToolName: "read_file", Paths: []string{"src/a.go"}, Success: true, Read: true})
	l.Record(Receipt{ToolName: "write_file", Paths: []string{"src/b.go"}, Success: true, Write: true})
	if !l.HasSuccessfulReadOrWrite([]string{"src/a.go"}) {
		t.Error("read path must satisfy read-or-write")
	}
	if !l.HasSuccessfulReadOrWrite([]string{"src/b.go"}) {
		t.Error("write path must satisfy read-or-write")
	}
	if l.HasSuccessfulReadOrWrite([]string{"src/c.go"}) {
		t.Error("untouched path must not satisfy read-or-write")
	}
}

func TestLedgerMatchLatestTodoStep(t *testing.T) {
	l := NewLedger()
	l.Record(Receipt{ToolName: "todo_write", Success: true, Todos: []TodoItem{{Content: "Step one"}, {Content: "Step two"}}})

	m, ok := l.MatchLatestTodoStep("1")
	if !ok || m.Index != 1 || m.Content != "Step one" {
		t.Errorf("MatchLatestTodoStep(1) = %+v, %v; want index 1", m, ok)
	}
	m, ok = l.MatchLatestTodoStep("step two")
	if !ok || m.Index != 2 || m.Content != "Step two" {
		t.Errorf("MatchLatestTodoStep(title) = %+v, %v; want index 2 case-insensitive", m, ok)
	}
	if m, ok := l.MatchLatestTodoStep("missing"); ok && m.Found {
		t.Error("unknown step must not match")
	}
	if _, ok := l.MatchLatestTodoStep(""); ok {
		t.Error("empty step must not match")
	}
}

func TestRecordCompleteStepAutoMatch(t *testing.T) {
	l := NewLedger()
	l.Record(Receipt{ToolName: "todo_write", Success: true, Todos: []TodoItem{{Content: "A"}, {Content: "B"}}})
	l.Record(Receipt{ToolName: "complete_step", Step: "2", Success: true})

	l.mu.Lock()
	last := l.receipts[len(l.receipts)-1]
	l.mu.Unlock()
	if last.TodoStep == nil {
		t.Fatal("complete_step receipt must auto-populate TodoStep")
	}
	if !last.TodoStep.Found || last.TodoStep.Index != 2 || last.TodoStep.Content != "B" {
		t.Errorf("auto TodoStep = %+v, want index 2 content B", last.TodoStep)
	}
}

func TestMatchStep(t *testing.T) {
	todos := []TodoItem{{Content: "One"}, {Content: "Two"}}
	m, ok := MatchStep("2", todos)
	if !ok || m.Index != 2 || m.Content != "Two" {
		t.Errorf("MatchStep(2) = %+v, %v; want index 2", m, ok)
	}
	m, ok = MatchStep("2.", todos)
	if !ok || m.Index != 2 {
		t.Errorf("MatchStep(2.) = %+v, %v; want index 2 (strip trailing dot)", m, ok)
	}
	m, ok = MatchStep("two", todos)
	if !ok || m.Index != 2 {
		t.Errorf("MatchStep(two) = %+v, %v; want case-insensitive index 2", m, ok)
	}
	if _, ok := MatchStep("3", todos); ok {
		t.Error("out-of-range index must not match")
	}
	if _, ok := MatchStep("zzz", todos); ok {
		t.Error("unknown title must not match")
	}
}

func TestUnverifiedCompletedTodos(t *testing.T) {
	// baseline all pending; current marks A completed without complete_step
	l := NewLedger()
	l.Record(Receipt{ToolName: "todo_write", Success: true, Todos: []TodoItem{{Content: "A", Status: "pending"}, {Content: "B", Status: "pending"}}})
	missing, baseline := l.UnverifiedCompletedTodos([]TodoItem{{Content: "A", Status: "completed"}, {Content: "B", Status: "pending"}})
	if !baseline {
		t.Fatal("hasBaseline must be true")
	}
	if len(missing) != 1 || missing[0].Index != 1 || missing[0].Content != "A" {
		t.Errorf("missing = %+v, want [A@1]", missing)
	}
}

func TestUnverifiedCompletedTodosCoveredByCompleteStep(t *testing.T) {
	l := NewLedger()
	l.Record(Receipt{ToolName: "todo_write", Success: true, Todos: []TodoItem{{Content: "A", Status: "pending"}, {Content: "B", Status: "pending"}}})
	l.Record(Receipt{ToolName: "complete_step", Step: "1", Success: true})
	missing, baseline := l.UnverifiedCompletedTodos([]TodoItem{{Content: "A", Status: "completed"}, {Content: "B", Status: "pending"}})
	if !baseline {
		t.Fatal("hasBaseline must be true")
	}
	if len(missing) != 0 {
		t.Errorf("complete_step-covered todo must not be missing, got %+v", missing)
	}
}

func TestUnverifiedCompletedTodosDifferentIndexNotCovered(t *testing.T) {
	l := NewLedger()
	l.Record(Receipt{ToolName: "todo_write", Success: true, Todos: []TodoItem{{Content: "A", Status: "pending"}, {Content: "B", Status: "pending"}}})
	l.Record(Receipt{ToolName: "complete_step", Step: "2", Success: true})
	missing, _ := l.UnverifiedCompletedTodos([]TodoItem{{Content: "A", Status: "completed"}, {Content: "B", Status: "pending"}})
	if len(missing) != 1 || missing[0].Index != 1 {
		t.Errorf("complete_step for B must not cover A, missing = %+v", missing)
	}
}

func TestUnverifiedCompletedTodosPreviousCompletedSkipped(t *testing.T) {
	l := NewLedger()
	l.Record(Receipt{ToolName: "todo_write", Success: true, Todos: []TodoItem{{Content: "A", Status: "completed"}, {Content: "B", Status: "pending"}}})
	missing, _ := l.UnverifiedCompletedTodos([]TodoItem{{Content: "A", Status: "completed"}, {Content: "B", Status: "completed"}})
	if len(missing) != 1 || missing[0].Index != 2 || missing[0].Content != "B" {
		t.Errorf("only newly-completed B should be missing, got %+v", missing)
	}
}

func TestUnverifiedCompletedTodosNoBaseline(t *testing.T) {
	l := NewLedger()
	missing, baseline := l.UnverifiedCompletedTodos([]TodoItem{{Content: "A", Status: "completed"}})
	if baseline {
		t.Error("no prior todo_write → hasBaseline must be false")
	}
	if len(missing) != 0 {
		t.Errorf("no baseline → missing must be empty, got %+v", missing)
	}
}

func TestIncompleteTodos(t *testing.T) {
	todos := []TodoItem{
		{Content: "Done", Status: "completed"},
		{Content: "WIP"},
		{Content: "Todo", Status: "pending"},
	}
	got := IncompleteTodos(todos)
	if len(got) != 2 {
		t.Fatalf("IncompleteTodos = %+v, want 2", got)
	}
	if got[0].Index != 2 || got[0].Status != "pending" {
		t.Errorf("got[0] = %+v, want index 2 pending", got[0])
	}
	if got[1].Index != 3 {
		t.Errorf("got[1] = %+v, want index 3", got[1])
	}
}

func TestNormalizePath(t *testing.T) {
	if got := normalizePath("  "); got != "" {
		t.Errorf("blank path = %q, want empty", got)
	}
	want := normalizePath("src/SubDir/File.go")
	got := normalizePath("src\\SubDir\\File.go")
	if got != want {
		t.Errorf("backslash normalizePath = %q, want %q (same form as slash input)", got, want)
	}
}

func TestNormalizePaths(t *testing.T) {
	got := normalizePaths([]string{"a.go", "", "  ", "b.go"})
	if len(got) != 2 || got[0] != "a.go" || got[1] != "b.go" {
		t.Errorf("normalizePaths = %v, want [a.go b.go] (blanks dropped)", got)
	}
}

func TestWithLedgerFromContext(t *testing.T) {
	l := NewLedger()
	ctx := WithLedger(context.Background(), l)
	got, ok := FromContext(ctx)
	if !ok || got != l {
		t.Errorf("FromContext = %v, %v; want ledger", got, ok)
	}
	// nil ledger leaves context untouched
	ctx2 := WithLedger(context.Background(), nil)
	if _, ok := FromContext(ctx2); ok {
		t.Error("nil ledger must not be retrievable")
	}
}

func TestLedgerResetAndStrict(t *testing.T) {
	l := NewLedger()
	l.SetStrictVerification(true)
	if !l.StrictVerification() {
		t.Error("strict verification must be on")
	}
	l.Record(Receipt{ToolName: "bash", Command: "echo hi", Success: true})
	if !l.HasSuccessfulCommand("echo hi") {
		t.Fatal("precondition failed")
	}
	l.Reset()
	if l.HasSuccessfulCommand("echo hi") {
		t.Error("Reset must clear receipts")
	}
	if !l.StrictVerification() {
		t.Error("Reset must keep the strict verification flag")
	}
}
