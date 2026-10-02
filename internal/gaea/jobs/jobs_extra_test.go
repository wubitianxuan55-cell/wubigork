package jobs

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/gaea/event"
)

type typedNilJobSink struct{}

func (*typedNilJobSink) Emit(event.Event) {}

func TestNewManagerTreatsTypedNilSinkAsDiscard(t *testing.T) {
	var sink *typedNilJobSink
	m := NewManager(sink)
	defer m.Close()

	j := m.Start("bash", "typed nil sink", func(context.Context, io.Writer) (string, error) {
		return "done", nil
	})
	res := m.Wait(context.Background(), []string{j.ID}, 1000)
	if len(res) != 1 || res[0].Status != Done {
		t.Fatalf("job result = %+v, want one done job", res)
	}
}

// --- Wait with timeout ---

func TestWaitTimeout(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()

	j := m.Start("bash", "", func(ctx context.Context, _ io.Writer) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	})
	// Wait with a very short timeout — the job won't finish in time.
	res := m.Wait(context.Background(), []string{j.ID}, 1)
	if len(res) != 1 {
		t.Fatalf("want 1 result, got %d", len(res))
	}
	// Should still be running (timeout expired before completion).
	if res[0].Status != Running {
		t.Errorf("status = %q, want running", res[0].Status)
	}
	m.Kill(j.ID)
}

// --- Wait with empty ids waits for all running ---

func TestWaitAllRunning(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()

	// Jobs block until cancelled so they are still running when Wait resolves the
	// "all running" set — instant-returning jobs could finish first and be missed,
	// which is exactly the resolution this test must observe deterministically. A
	// short timeout returns the still-running snapshot.
	j1 := m.Start("bash", "", func(ctx context.Context, _ io.Writer) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	})
	j2 := m.Start("bash", "", func(ctx context.Context, _ io.Writer) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	})
	res := m.Wait(context.Background(), nil, 1)
	if len(res) != 2 {
		t.Fatalf("want 2 results, got %d", len(res))
	}
	ids := map[string]bool{res[0].ID: true, res[1].ID: true}
	if !ids[j1.ID] || !ids[j2.ID] {
		t.Errorf("results missing expected ids: %v", ids)
	}
	m.Kill(j1.ID)
	m.Kill(j2.ID)
}

// --- Output with unknown id ---

func TestOutputUnknownID(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()

	_, _, ok := m.Output("nonexistent-id")
	if ok {
		t.Error("Output for unknown id should return ok=false")
	}
}

// --- Kill with unknown id ---

func TestKillUnknownID(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()

	if m.Kill("nonexistent-id") {
		t.Error("Kill for unknown id should return false")
	}
}

// --- startedText ---

func TestStartedTextWithLabel(t *testing.T) {
	got := startedText("bash", "bash-1", "my-label")
	if got != "background bash started: bash-1 (my-label)" {
		t.Errorf("startedText = %q", got)
	}
}

func TestStartedTextWithoutLabel(t *testing.T) {
	got := startedText("task", "task-1", "")
	if got != "background task started: task-1" {
		t.Errorf("startedText = %q", got)
	}
}

// --- DrainCompletedNote with multiple jobs ---

func TestDrainMultiple(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()

	m.Start("bash", "", func(_ context.Context, _ io.Writer) (string, error) {
		return "", nil
	})
	m.Start("task", "label", func(_ context.Context, _ io.Writer) (string, error) {
		return "answer", nil
	})
	m.Wait(context.Background(), nil, 5)
	note := m.DrainCompletedNote()
	if note == "" {
		t.Fatal("drain should not be empty after 2 completions")
	}
}

// --- Close is idempotent ---

func TestCloseIdempotent(t *testing.T) {
	m := NewManager(event.Discard)
	m.Start("bash", "", func(_ context.Context, _ io.Writer) (string, error) {
		return "", nil
	})
	m.Close()
	m.Close() // should not panic
}

// --- Running with no jobs ---

func TestRunningEmpty(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()
	if r := m.Running(); len(r) != 0 {
		t.Errorf("Running() = %d, want 0", len(r))
	}
}

// --- Job with error sets Failed ---

func TestJobFailed(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()

	j := m.Start("bash", "", func(_ context.Context, _ io.Writer) (string, error) {
		return "", io.ErrUnexpectedEOF
	})
	res := m.Wait(context.Background(), []string{j.ID}, 5)
	if len(res) != 1 || res[0].Status != Failed {
		t.Fatalf("want Failed, got %+v", res)
	}
}

// --- Job with result and no error sets Done ---

func TestJobWithResult(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()

	j := m.Start("task", "", func(_ context.Context, _ io.Writer) (string, error) {
		return "final answer", nil
	})
	res := m.Wait(context.Background(), []string{j.ID}, 5)
	if len(res) != 1 || res[0].Status != Done {
		t.Fatalf("want Done, got %+v", res)
	}
	if res[0].Output != "final answer" {
		t.Errorf("output = %q, want \"final answer\"", res[0].Output)
	}
}

// --- Context injection ---

func TestWithManagerFromContext(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()

	ctx := WithManager(context.Background(), m)
	got, ok := FromContext(ctx)
	if !ok || got != m {
		t.Error("FromContext should return the manager")
	}
}

func TestFromContextEmpty(t *testing.T) {
	_, ok := FromContext(context.Background())
	if ok {
		t.Error("plain context should return ok=false")
	}
}

// --- Status constants ---

func TestStatusConstants(t *testing.T) {
	if Running != "running" {
		t.Errorf("Running = %q", Running)
	}
	if Done != "done" {
		t.Errorf("Done = %q", Done)
	}
	if Failed != "failed" {
		t.Errorf("Failed = %q", Failed)
	}
	if Killed != "killed" {
		t.Errorf("Killed = %q", Killed)
	}
}

// --- nowMs ---

func TestNowMs(t *testing.T) {
	before := time.Now().UnixMilli()
	got := nowMs()
	after := time.Now().UnixMilli()
	if got < before || got > after {
		t.Errorf("nowMs() = %d, not in [%d, %d]", got, before, after)
	}
}

// TestStartInheritingValuesNotCancellation 审计 P1 GA1-08：StartInheriting 的
// job ctx 继承调用方 value 链（空间等随代传递），但取消链留在 root——caller
// 被取消不连带后台 job。
func TestStartInheritingValuesNotCancellation(t *testing.T) {
	m := NewManager(event.Discard)
	defer m.Close()

	type probeKey struct{}
	caller, cancelCaller := context.WithCancel(context.Background())
	caller = context.WithValue(caller, probeKey{}, "inherited")

	got := make(chan string, 1)
	job := m.StartInheriting(caller, "task", "probe", func(ctx context.Context, _ io.Writer) (string, error) {
		got <- ctx.Value(probeKey{}).(string)
		<-ctx.Done() // 等 manager.Cancel：正常收尾路径
		return "done-after-cancel", nil
	})

	if v := <-got; v != "inherited" {
		t.Fatalf("job ctx 未继承调用方 value: %q", v)
	}
	// job ctx 的 jobIDKey 必须是自己的 ID（不被 caller 链里的父 job ID 遮蔽）。
	m.mu.Lock()
	jctx := m.jobs[job.ID].ctx
	m.mu.Unlock()
	if id, _ := JobIDFromContext(jctx); id != job.ID {
		t.Fatalf("job ctx 的 jobID = %q, want %q（caller 链不应遮蔽）", id, job.ID)
	}
	// caller 取消不得连带 job：取消后 job 仍在跑。
	cancelCaller()
	select {
	case <-job.done:
		t.Fatal("caller 取消连带终止了后台 job（取消链被继承）")
	case <-time.After(200 * time.Millisecond):
	}
	// 收尾：用 manager 自己的 cancel 终止 job。
	m.Kill(job.ID)
	<-job.done
}
