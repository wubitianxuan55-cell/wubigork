package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gaea/gaea/internal/whisper"
)

// ── T6-5.3 异步记忆写入可观测：WriteErrors 计数 + 日志断言 ──────────────

// writeObsLlmStub 测试用 LlmClient：固定返回 reply/err，或按需 panic。
type writeObsLlmStub struct {
	reply    string
	err      error
	panicMsg string
}

func (m writeObsLlmStub) Chat(systemPrompt, userPrompt string) (string, error) {
	if m.panicMsg != "" {
		panic(m.panicMsg)
	}
	return m.reply, m.err
}

// waitWhisperWriteErrorCount 轮询等待 WriteErrors 计数达标（异步协程完成）。
func waitWhisperWriteErrorCount(t *testing.T, a *whisperState, want int64) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		got, last, _ := a.whisperWriteErrorStats()
		if got >= want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("等待 WriteErrors 计数 ≥%d 超时: got %d, last=%q", want, got, last)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// waitFactStoreCount 轮询等待事实库计数达标（正常路径写入完成信号）。
func waitFactStoreCount(t *testing.T, fs *whisper.FactStore, want int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if fs.Count() >= want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("等待 FactStore 计数 ≥%d 超时: got %d", want, fs.Count())
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestWhisperMemoryWrite_LLMFailureCounted 记忆写入协程 LLM 失败：
// WriteErrors 计数 ≥1 且错误日志出现，事实库不写入。
func TestWhisperMemoryWrite_LLMFailureCounted(t *testing.T) {
	a := &whisperState{core: &core{}}
	fs := whisper.NewFactStore()
	records := captureLogs(t, func() {
		whisper.EnqueueMemoryWrite(writeObsLlmStub{err: errors.New("llm down")}, whisper.MemoryWritePayload{
			SessionID: "obs-llm-fail", TurnIndex: 1,
			UserMsg: "我喜欢吃辣", AssistantText: "记住了",
			FactStore: fs, TotalTurns: 1,
		}, a.recordMemoryWriteError)
		whisper.DrainAllMemoryWriteJobs()
		waitWhisperWriteErrorCount(t, a, 1)
	})
	if got, last, _ := a.whisperWriteErrorStats(); got < 1 {
		t.Fatalf("WriteErrors 计数 = %d, want ≥1", got)
	} else if !strings.Contains(last, "llm_extract") {
		t.Errorf("最近错误摘要应含阶段 llm_extract, got %q", last)
	}
	if fs.Count() != 0 {
		t.Fatalf("LLM 失败后不应写入事实, got %d", fs.Count())
	}
	if !logContainsMsg(records, "LLM 事实抽取失败") {
		t.Fatalf("未记录 LLM 事实抽取失败日志, got %v", logMsgs(records))
	}
	if !logContainsMsg(records, "异步记忆写入失败") {
		t.Fatalf("未记录异步记忆写入失败日志, got %v", logMsgs(records))
	}
}

// TestWhisperMemoryWrite_JSONParseCounted 记忆写入协程 JSON 解析失败：
// WriteErrors 计数 ≥1 且错误日志出现。
func TestWhisperMemoryWrite_JSONParseCounted(t *testing.T) {
	a := &whisperState{core: &core{}}
	fs := whisper.NewFactStore()
	records := captureLogs(t, func() {
		whisper.EnqueueMemoryWrite(writeObsLlmStub{reply: "这不是 JSON 输出"}, whisper.MemoryWritePayload{
			SessionID: "obs-json-parse", TurnIndex: 2,
			UserMsg: "我喜欢吃辣", AssistantText: "记住了",
			FactStore: fs, TotalTurns: 1,
		}, a.recordMemoryWriteError)
		whisper.DrainAllMemoryWriteJobs()
		waitWhisperWriteErrorCount(t, a, 1)
	})
	if got, last, _ := a.whisperWriteErrorStats(); got < 1 {
		t.Fatalf("WriteErrors 计数 = %d, want ≥1", got)
	} else if !strings.Contains(last, "json_parse") {
		t.Errorf("最近错误摘要应含阶段 json_parse, got %q", last)
	}
	if fs.Count() != 0 {
		t.Fatalf("解析失败后不应写入事实, got %d", fs.Count())
	}
	if !logContainsMsg(records, "事实抽取结果 JSON 解析失败") {
		t.Fatalf("未记录 JSON 解析失败日志, got %v", logMsgs(records))
	}
	if !logContainsMsg(records, "异步记忆写入失败") {
		t.Fatalf("未记录异步记忆写入失败日志, got %v", logMsgs(records))
	}
}

// TestWhisperMemoryWrite_PanicCounted 记忆写入协程 panic 兜底：
// panic 被恢复、计入 WriteErrors 计数并记录日志（兜底行为保持不变）。
func TestWhisperMemoryWrite_PanicCounted(t *testing.T) {
	a := &whisperState{core: &core{}}
	records := captureLogs(t, func() {
		whisper.EnqueueMemoryWrite(writeObsLlmStub{panicMsg: "boom"}, whisper.MemoryWritePayload{
			SessionID: "obs-panic", TurnIndex: 1,
			UserMsg: "你好", AssistantText: "嗨",
			FactStore: whisper.NewFactStore(), TotalTurns: 1,
		}, a.recordMemoryWriteError)
		whisper.DrainAllMemoryWriteJobs()
		waitWhisperWriteErrorCount(t, a, 1)
	})
	if got, last, _ := a.whisperWriteErrorStats(); got < 1 {
		t.Fatalf("WriteErrors 计数 = %d, want ≥1", got)
	} else if !strings.Contains(last, "panic") {
		t.Errorf("最近错误摘要应含阶段 panic, got %q", last)
	}
	if !logContainsMsg(records, "memory write goroutine panic recovered") {
		t.Fatalf("未记录 panic 恢复日志, got %v", logMsgs(records))
	}
	if !logContainsMsg(records, "异步记忆写入失败") {
		t.Fatalf("未记录异步记忆写入失败日志, got %v", logMsgs(records))
	}
}

// TestWhisperMemoryWrite_NormalPathNoError 正常路径：LLM 抽取成功写库，
// WriteErrors 计数必须为 0（无错误计入）。
func TestWhisperMemoryWrite_NormalPathNoError(t *testing.T) {
	a := &whisperState{core: &core{}}
	fs := whisper.NewFactStore()
	records := captureLogs(t, func() {
		whisper.EnqueueMemoryWrite(writeObsLlmStub{reply: `{"facts":[{"domain":"user_profile","subcategory":"BASIC_PROFILE","subject":"宠物","summary":"养了一只猫","weight":0.8,"confidence":0.9,"selfRelevance":0.8}]}`}, whisper.MemoryWritePayload{
			SessionID: "obs-normal", TurnIndex: 1,
			UserMsg: "我家有只猫", AssistantText: "好可爱",
			FactStore: fs, TotalTurns: 1,
		}, a.recordMemoryWriteError)
		whisper.DrainAllMemoryWriteJobs()
		// 等事实写入完成（异步任务已跑完），再断言无错误计数
		waitFactStoreCount(t, fs, 1)
	})
	if got, last, _ := a.whisperWriteErrorStats(); got != 0 {
		t.Fatalf("正常路径 WriteErrors 计数 = %d, want 0 (last=%q)", got, last)
	}
	if fs.Count() != 1 {
		t.Fatalf("正常路径应写入 1 条事实, got %d", fs.Count())
	}
	if logContainsMsg(records, "异步记忆写入失败") {
		t.Fatalf("正常路径不应记录异步记忆写入失败日志, got %v", logMsgs(records))
	}
}

// TestWhisperPersist_DBFailureCounted 持久化协程落库失败（数据根被文件占位）：
// persistStateAsync 把错误计入 WriteErrors 计数并记录日志，不再 fire-and-forget 静默丢弃。
func TestWhisperPersist_DBFailureCounted(t *testing.T) {
	a := newChatServiceTestApp(t)
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("写占位文件: %v", err)
	}

	orch := whisper.NewOrchestrator("obs-persist", whisper.PersonalityPresets[0])
	orch.DataRoot = blocker
	orch.FactStore.Add(whisper.MemoryFact{
		Domain: "preference", Subcategory: "FOOD", Subject: "用户",
		Summary: "喜欢吃辣", Weight: 2, Confidence: 0.9,
	})
	orch.WM.Push(orch.SessionID, whisper.Exchange{
		TurnIndex: 1, UserText: "你好", AssistantText: "你好呀",
	})

	records := captureLogs(t, func() {
		a.persistStateAsync(orch)
	})
	if got, last, _ := a.whisperWriteErrorStats(); got < 1 {
		t.Fatalf("WriteErrors 计数 = %d, want ≥1", got)
	} else if !strings.Contains(last, "persist") {
		t.Errorf("最近错误摘要应含阶段 persist, got %q", last)
	}
	if !logContainsMsg(records, "异步记忆写入失败") {
		t.Fatalf("未记录异步记忆写入失败日志, got %v", logMsgs(records))
	}
	if !logContainsMsg(records, "落库失败") {
		t.Fatalf("未记录落库失败日志, got %v", logMsgs(records))
	}
}
