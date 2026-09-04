package jobs

import (
	"context"
	"io"
	"testing"
	"time"
)

// 等待 job 到达终态（测试辅助；超时 2s 判失败）。
func waitTerminal(t *testing.T, m *Manager, id string) Status {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if j := m.get(id); j != nil {
			j.mu.Lock()
			st := j.status
			j.mu.Unlock()
			if st != Running {
				return st
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("job %s 未在时限内到达终态", id)
	return ""
}

// parentContext 构造嵌有 job ID 的调用方 ctx（模拟后台工具 Execute ctx，
// 与 Start 注入 run ctx 的 jobIDKey 同键）。
func parentContext(id string) context.Context {
	return context.WithValue(context.Background(), jobIDKey{}, id)
}

// TestKillCascade 终止级联：父任务被杀 → 其派生的存活后代（含跨层）连带
// 取消并落 killed 终态；无父子关系的任务不受波及。
func TestKillCascade(t *testing.T) {
	m := NewManager(nil)
	defer m.Close()

	block := func(ctx context.Context, _ io.Writer) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	}

	parent := m.Start("task", "parent", block)
	child := m.StartIn(parentContext(parent.ID), "task", "child", block)
	grandchild := m.StartIn(parentContext(child.ID), "bash", "grandchild", block)
	unrelated := m.Start("task", "unrelated", block)

	if !m.Kill(parent.ID) {
		t.Fatal("Kill(parent) 应返回 true")
	}
	if st := waitTerminal(t, m, parent.ID); st != Killed {
		t.Fatalf("parent = %v, want killed", st)
	}
	if st := waitTerminal(t, m, child.ID); st != Killed {
		t.Fatalf("child = %v, want killed（级联）", st)
	}
	if st := waitTerminal(t, m, grandchild.ID); st != Killed {
		t.Fatalf("grandchild = %v, want killed（跨层级联）", st)
	}
	// 无关任务不受波及：短暂等待后仍为 Running，再显式收尾。
	time.Sleep(30 * time.Millisecond)
	if j := m.get(unrelated.ID); j != nil {
		j.mu.Lock()
		if j.status != Running {
			t.Fatalf("unrelated = %v, want 仍运行", j.status)
		}
		j.mu.Unlock()
	}
	m.Kill(unrelated.ID)
}

// TestKillChildDoesNotAffectParent 杀子任务不终结父任务（级联单向向下）。
func TestKillChildDoesNotAffectParent(t *testing.T) {
	m := NewManager(nil)
	defer m.Close()

	release := make(chan struct{})
	parent := m.Start("task", "parent", func(ctx context.Context, _ io.Writer) (string, error) {
		<-release
		return "done", nil
	})
	child := m.StartIn(parentContext(parent.ID), "bash", "child", func(ctx context.Context, _ io.Writer) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	})
	if !m.Kill(child.ID) {
		t.Fatal("Kill(child) 应返回 true")
	}
	if st := waitTerminal(t, m, child.ID); st != Killed {
		t.Fatalf("child = %v, want killed", st)
	}
	select {
	case <-parent.done:
		t.Fatal("杀子任务不应终结父任务")
	default:
	}
	close(release)
}

// TestStartInNoParent 主回合派生（无父 ctx）行为与 Start 一致。
func TestStartInNoParent(t *testing.T) {
	m := NewManager(nil)
	defer m.Close()
	done := make(chan struct{})
	j := m.StartIn(context.Background(), "task", "solo", func(ctx context.Context, _ io.Writer) (string, error) {
		close(done)
		return "ok", nil
	})
	<-done
	waitTerminal(t, m, j.ID)
	if j.ParentID != "" {
		t.Fatalf("ParentID = %q, want 空", j.ParentID)
	}
}
