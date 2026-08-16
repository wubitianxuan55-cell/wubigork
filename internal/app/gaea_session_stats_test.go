package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent"
	"github.com/gaea/gaea/internal/gaea/agent/session"
	gaeaConfig "github.com/gaea/gaea/internal/gaea/config"
	"github.com/gaea/gaea/internal/gaea/event"
	"github.com/gaea/gaea/internal/gaea/provider"
)

// writeEventSessionWithUsage 写一个事件日志会话：用户回合 + 两次 usage 事件
// （main + subagent 各一，带 pricing），返回会话路径。
func writeEventSessionWithUsage(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	s := agent.NewSession("you are gaea")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "写一份周报"})
	path := filepath.Join(dir, "stats.jsonl")
	if err := s.Save(path); err != nil {
		t.Fatalf("保存会话: %v", err)
	}

	lp := session.LogPathFor(path)
	w, err := session.OpenLog(lp, path)
	if err != nil {
		t.Fatalf("OpenLog: %v", err)
	}
	defer w.Close()

	// main 调用：prompt 100 = hit 40 + miss 60；completion 50。
	// 输入价 2/MTok、输出 8/MTok、命中 0.5/MTok
	if _, err := w.Append(usageEntryKind(t, event.UsageSourceMain, provider.Pricing{Input: 2, Output: 8, CacheHit: 0.5, Currency: "usd"},
		provider.Usage{PromptTokens: 100, CompletionTokens: 50, CacheHitTokens: 40, CacheMissTokens: 60, TotalTokens: 150})); err != nil {
		t.Fatalf("append main usage: %v", err)
	}
	// subagent 调用：prompt 300 全 miss；completion 100
	if _, err := w.Append(usageEntryKind(t, event.UsageSourceSubagent, provider.Pricing{Input: 2, Output: 8, Currency: "usd"},
		provider.Usage{PromptTokens: 300, CompletionTokens: 100, CacheMissTokens: 300, TotalTokens: 400})); err != nil {
		t.Fatalf("append subagent usage: %v", err)
	}
	return path
}

// usageEntryKind 由 EntryFromEvent 生成 usage 日志条目，返回 (kind, payload)
// 供 LogWriter.Append 落盘（与真实落盘载荷同源：usagePayloadFromEvent 折叠
// pricing 为无损 payload）。
func usageEntryKind(t *testing.T, source string, pricing provider.Pricing, u provider.Usage) (string, any) {
	t.Helper()
	le, err := session.EntryFromEvent(event.Event{
		Kind:        event.Usage,
		Usage:       &u,
		UsageSource: source,
		Pricing:     &pricing,
	}, 1)
	if err != nil {
		t.Fatalf("EntryFromEvent: %v", err)
	}
	return le.Kind, le.Payload
}

func TestGaeaSessionStats(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()

	ws := t.TempDir()
	sessionDir := gaeaConfig.WorkspaceSessionDir(ws)
	path := writeEventSessionWithUsage(t, sessionDir)

	a := &App{}
	got := a.GaeaSessionStats(path)
	if !got.Available {
		t.Fatalf("事件日志会话应 Available=true")
	}
	st := got.Stats
	if st.PromptTokens != 400 || st.CompletionTokens != 150 || st.TotalTokens != 550 {
		t.Errorf("token 统计错误: prompt=%d completion=%d total=%d, want 400/150/550", st.PromptTokens, st.CompletionTokens, st.TotalTokens)
	}
	if st.CacheHitTokens != 40 || st.CacheMissTokens != 360 {
		t.Errorf("缓存统计错误: hit=%d miss=%d, want 40/360", st.CacheHitTokens, st.CacheMissTokens)
	}
	if st.UsageCount != 2 {
		t.Errorf("UsageCount=%d, want 2", st.UsageCount)
	}
	// 成本按 DeriveStats 口径 = (hit*price + miss*input + completion*output)/1e6
	// main: (40*0.5 + 60*2 + 50*8)/1e6 = 0.00054；subagent: (300*2 + 100*8)/1e6 = 0.0014
	wantCost := (40*0.5+60*2+50*8)/1e6 + (300*2+100*8)/1e6
	if st.Cost < wantCost-1e-9 || st.Cost > wantCost+1e-9 {
		t.Errorf("Cost=%.10f, want %.10f", st.Cost, wantCost)
	}
	// MainCost 只含 main 调用成本
	wantMain := (40*0.5 + 60*2 + 50*8) / 1e6
	if st.MainCost < wantMain-1e-9 || st.MainCost > wantMain+1e-9 {
		t.Errorf("MainCost=%.10f, want %.10f", st.MainCost, wantMain)
	}
	wantSub := (300*2 + 100*8) / 1e6
	if st.SubagentCost < wantSub-1e-9 || st.SubagentCost > wantSub+1e-9 {
		t.Errorf("SubagentCost=%.10f, want %.10f", st.SubagentCost, wantSub)
	}
}

func TestGaeaSessionStats_Unavailable(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()

	a := &App{}

	// 空路径
	if v := a.GaeaSessionStats(""); v.Available {
		t.Errorf("空路径应 Available=false")
	}
	// 非法路径（不在会话目录下）
	if v := a.GaeaSessionStats(filepath.Join(t.TempDir(), "outside.jsonl")); v.Available {
		t.Errorf("非法路径应 Available=false")
	}
	// legacy 会话（有 jsonl 无事件日志）
	ws := t.TempDir()
	sessionDir := gaeaConfig.WorkspaceSessionDir(ws)
	s := agent.NewSession("you are gaea")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "hi"})
	legacyPath := filepath.Join(sessionDir, "legacy.jsonl")
	if err := s.Save(legacyPath); err != nil {
		t.Fatalf("保存 legacy 会话: %v", err)
	}
	if v := a.GaeaSessionStats(legacyPath); v.Available {
		t.Errorf("legacy 会话（无日志）应 Available=false")
	}
}

// TestGaeaSessionStats_EmptyLog 验证事件日志存在但无 usage 事件时统计全零但
// 仍 Available（有日志即视为可派生，前端可展示「0 调用」）。
func TestGaeaSessionStats_EmptyLog(t *testing.T) {
	restore := workspaceTestIsolate(t)
	defer restore()

	ws := t.TempDir()
	sessionDir := gaeaConfig.WorkspaceSessionDir(ws)
	s := agent.NewSession("you are gaea")
	s.Add(provider.Message{Role: provider.RoleUser, Content: "hi"})
	path := filepath.Join(sessionDir, "empty.jsonl")
	if err := s.Save(path); err != nil {
		t.Fatalf("保存会话: %v", err)
	}
	// 仅迁移出日志（无 usage 事件）
	lp := session.LogPathFor(path)
	w, err := session.OpenLog(lp, path)
	if err != nil {
		t.Fatalf("OpenLog: %v", err)
	}
	_ = w.Close()

	a := &App{}
	v := a.GaeaSessionStats(path)
	if !v.Available {
		t.Errorf("有日志应 Available=true")
	}
	if v.Stats.UsageCount != 0 || v.Stats.TotalTokens != 0 || v.Stats.Cost != 0 {
		t.Errorf("空日志统计应全零: %+v", v.Stats)
	}
}
