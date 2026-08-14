package whisper

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// ── slog 捕获测试基建（T6-1.3 吞错清理日志断言用）──────────────────

type captureHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r.Clone())
	return nil
}

func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(string) slog.Handler      { return h }

func (h *captureHandler) snapshot() []slog.Record {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]slog.Record(nil), h.records...)
}

func captureLogs(t *testing.T, fn func()) []slog.Record {
	t.Helper()
	h := &captureHandler{}
	orig := slog.Default()
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { slog.SetDefault(orig) })
	defer slog.SetDefault(orig)
	fn()
	return h.snapshot()
}

func logMsgs(records []slog.Record) []string {
	out := make([]string, 0, len(records))
	for _, r := range records {
		out = append(out, r.Message)
	}
	return out
}

func logContainsMsg(records []slog.Record, substr string) bool {
	for _, r := range records {
		if strings.Contains(r.Message, substr) {
			return true
		}
	}
	return false
}

// llmStub 测试用 LlmClient：固定返回 reply/err。
type llmStub struct {
	reply string
	err   error
}

func (m llmStub) Chat(systemPrompt, userPrompt string) (string, error) { return m.reply, m.err }

// ── memory_ingest 吞错 → 日志（T6-1.3）─────────────────────────────

// TestMemoryIngest_LLMFailureLogs LLM 事实抽取失败必须记录日志（不再静默 return）。
func TestMemoryIngest_LLMFailureLogs(t *testing.T) {
	fs := NewFactStore()
	p := NewMemoryIngestPipeline(llmStub{err: errors.New("llm down")})
	records := captureLogs(t, func() {
		p.AfterTurn(IngestTurnArgs{
			SessionID: "s1", TurnIndex: 3,
			UserMsg: "我喜欢吃辣", CompanionMsg: "记住了",
			FactStore: fs, TotalTurns: 0,
		})
	})
	if !logContainsMsg(records, "LLM 事实抽取失败") {
		t.Fatalf("未记录 LLM 事实抽取失败日志, got %v", logMsgs(records))
	}
	if fs.Count() != 0 {
		t.Fatalf("LLM 失败后不应写入任何事实, got %d", fs.Count())
	}
}

// TestMemoryIngest_ParseFailureLogs LLM 返回坏 JSON 时解析失败必须记录日志。
func TestMemoryIngest_ParseFailureLogs(t *testing.T) {
	fs := NewFactStore()
	p := NewMemoryIngestPipeline(llmStub{reply: "这不是 JSON 输出"})
	records := captureLogs(t, func() {
		p.AfterTurn(IngestTurnArgs{
			SessionID: "s1", TurnIndex: 4,
			UserMsg: "我喜欢吃辣", CompanionMsg: "记住了",
			FactStore: fs, TotalTurns: 0,
		})
	})
	if !logContainsMsg(records, "事实抽取结果 JSON 解析失败") {
		t.Fatalf("未记录解析失败日志, got %v", logMsgs(records))
	}
	if fs.Count() != 0 {
		t.Fatalf("解析失败后不应写入任何事实, got %d", fs.Count())
	}
}

// ── memory_consolidator 吞错 → 日志（T6-1.3）──────────────────────

// seedRawFacts 灌入 consolidationMinFacts 条互不重复的原始事实（Subcategory 各不相同，
// 避免 Jaccard 去重合并导致条数不足）。
func seedRawFacts(fs *FactStore) {
	for i := 0; i < consolidationMinFacts; i++ {
		fs.Add(MemoryFact{
			Domain: "user_profile", Subcategory: fmt.Sprintf("BASIC_PROFILE_%d", i), Subject: "用户",
			Summary: fmt.Sprintf("事实%d：用户喜欢不同的内容%d", i, i), Weight: 1, Confidence: 0.8,
		})
	}
}

// TestMemoryConsolidate_LLMFailureLogs 记忆整合 LLM 调用失败必须记录日志。
func TestMemoryConsolidate_LLMFailureLogs(t *testing.T) {
	fs := NewFactStore()
	seedRawFacts(fs)
	mc := NewMemoryConsolidator()
	records := captureLogs(t, func() {
		added := mc.Consolidate(fs, llmStub{err: errors.New("llm down")}, nil, "s1", 1)
		if added != 0 {
			t.Fatalf("LLM 失败时整合数应为 0, got %d", added)
		}
	})
	if !logContainsMsg(records, "记忆整合 LLM 调用失败") {
		t.Fatalf("未记录记忆整合 LLM 调用失败日志, got %v", logMsgs(records))
	}
}

// TestMemoryConsolidate_ParseFailureLogs 记忆整合结果坏 JSON（兜底提取也失败）
// 必须记录日志。
func TestMemoryConsolidate_ParseFailureLogs(t *testing.T) {
	fs := NewFactStore()
	seedRawFacts(fs)
	mc := NewMemoryConsolidator()
	records := captureLogs(t, func() {
		added := mc.Consolidate(fs, llmStub{reply: "纯文本回复，没有 JSON 结构"}, nil, "s1", 1)
		if added != 0 {
			t.Fatalf("解析失败时整合数应为 0, got %d", added)
		}
	})
	if !logContainsMsg(records, "记忆整合结果解析失败") {
		t.Fatalf("未记录记忆整合解析失败日志, got %v", logMsgs(records))
	}
}

// ── task_plan_store persist 吞错 → 日志（T6-1.3）──────────────────

// TestTaskPlanStore_PersistFailureLogs 数据根不可写时 Save/Clear 落盘失败
// 必须记录日志（不再静默丢弃）。
func TestTaskPlanStore_PersistFailureLogs(t *testing.T) {
	root := t.TempDir()

	// 场景 1：dataRoot 被普通文件占位 → Save 原子写（MkdirAll）失败
	blocker := filepath.Join(root, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("写占位文件: %v", err)
	}
	store := NewTaskPlanStoreWithDataRoot(blocker)
	plan, progress := sampleTaskPlan()
	records := captureLogs(t, func() {
		store.Save("sess-1", plan, progress)
	})
	if !logContainsMsg(records, "轻语任务计划落盘失败") {
		t.Fatalf("未记录 Save 落盘失败日志, got %v", logMsgs(records))
	}

	// 场景 2：落盘目标 task_plan.json 被非空目录占位 → Clear 的 os.Remove 失败
	// （Windows 上父路径为文件的 Remove 会映射成 IsNotExist，因此改用非空目录制造非 IsNotExist 错误）
	dir := filepath.Join(root, "dir")
	if err := os.MkdirAll(filepath.Join(dir, taskPlanFileName, "sub"), 0o755); err != nil {
		t.Fatalf("建非空目录占位: %v", err)
	}
	store2 := NewTaskPlanStoreWithDataRoot(dir)
	records2 := captureLogs(t, func() {
		store2.Clear("sess-2")
	})
	if !logContainsMsg(records2, "轻语任务计划清除落盘失败") {
		t.Fatalf("未记录 Clear 落盘失败日志, got %v", logMsgs(records2))
	}
}
