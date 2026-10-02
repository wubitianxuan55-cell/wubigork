package trajectory

import (
	"runtime/debug"
	"testing"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// 审计 P0 GA5-03（第二轮·线D）：FoldTrajectory 是对**任意**磁盘日志文件
// 调用的纯函数（app/gaea_ui_contextview.go:204 直接喂读盘产物），残缺/迁移/
// 被截断的日志不得把看板打成 panic。
//
// 已证触发形态（internal/gaea/trajectory/fold.go）：
//   - 首条即 usage        → applyUsage:332 → assistantRecord:441 返回 nil → :333 解引用
//   - 首条即 text 增量     → apply:79     → assistantRecord:441 返回 nil → :80  解引用
//   - 首条即 reasoning     → apply:74     → 同上                              → :75
//   - assistant_message（迁移/投影产物）缺回合边界 → applyAssistantMessage:138/142
//   - 整段缺 turn_started  → 上述任一形态叠加（日志被截断/投影器只写增量）
//
// 契约（本条 P0 的验收口径）：
//  1. 不 panic；
//  2. 不凭空造轮（无 turn_started 就没有 Turns 条目）；
//  3. 如实降级、不丢信息——无回合的 assistant 增量落进轮间区段
//     BetweenTurns（与 applyHeader:231-235 / applyCompaction:359-363 /
//     applyAsk / applyApproval / applySubagentMessage 的既有分流同款），
//     连续多条无回合增量（推理→正文→usage）合并成同一条轮间记录。

// noPanic 把 panic 转成测试失败并打印触发栈（红阶段直接给出 file:line）。
func noPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("FoldTrajectory 对残缺日志 panic（P0 GA5-03）: %v\n%s", r, debug.Stack())
		}
	}()
	fn()
}

func betweenKinds(tl Trajectory) []string {
	kinds := make([]string, 0, len(tl.BetweenTurns))
	for _, r := range tl.BetweenTurns {
		kinds = append(kinds, r.Kind)
	}
	return kinds
}

func TestFoldTrajectoryPartialLogsDegradeNotPanic(t *testing.T) {
	cases := []struct {
		name        string
		entries     []session.LogEntry
		wantBetween []string
	}{
		{
			name: "首条即 usage",
			entries: []session.LogEntry{
				entry(1, "usage", map[string]any{"promptTokens": 1200, "completionTokens": 40}),
			},
			wantBetween: []string{"assistant"},
		},
		{
			name: "首条即 text 增量",
			entries: []session.LogEntry{
				entry(1, "text", map[string]any{"text": "半截回复"}),
				entry(2, "usage", map[string]any{"promptTokens": 10, "completionTokens": 2}),
			},
			wantBetween: []string{"assistant"},
		},
		{
			name: "首条即 reasoning 增量",
			entries: []session.LogEntry{
				entry(1, "reasoning", map[string]any{"text": "先看一眼"}),
			},
			wantBetween: []string{"assistant"},
		},
		{
			name: "缺 turn_started（日志中段被截断）",
			entries: []session.LogEntry{
				entry(1, "request_header", map[string]any{"system": "sys"}),
				entry(2, "reasoning", map[string]any{"text": "先看一眼"}),
				entry(3, "message", map[string]any{"text": "结论"}),
				entry(4, "usage", map[string]any{"promptTokens": 33, "completionTokens": 4}),
				entry(5, "turn_done", map[string]any{"err": ""}),
			},
			wantBetween: []string{"header", "assistant"},
		},
		{
			name: "迁移/投影产物 assistant_message 缺回合边界",
			entries: []session.LogEntry{
				entry(1, "assistant_message", map[string]any{
					"text":      "你好",
					"reasoning": "嗯",
					"tool_calls": []any{
						map[string]any{"id": "c1", "name": "read_file", "args": `{"path":"x"}`},
					},
				}),
			},
			wantBetween: []string{"assistant"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var tl Trajectory
			noPanic(t, func() { tl = FoldTrajectory(tc.entries) })
			if !tl.Ok {
				t.Fatalf("Ok = false，残缺日志也应返回可用的降级快照: %+v", tl)
			}
			if len(tl.Turns) != 0 {
				t.Fatalf("无 turn_started 不应凭空造轮: %+v", tl.Turns)
			}
			if len(tl.BetweenTurns) != len(tc.wantBetween) {
				t.Fatalf("BetweenTurns kinds = %v, want %v", betweenKinds(tl), tc.wantBetween)
			}
			for i, want := range tc.wantBetween {
				if tl.BetweenTurns[i].Kind != want {
					t.Fatalf("BetweenTurns[%d].Kind = %q, want %q（kinds=%v）",
						i, tl.BetweenTurns[i].Kind, want, betweenKinds(tl))
				}
			}
			// 降级不是丢弃：无回合的 assistant 记录必须带非 nil 的 Assistant
			// 子结构（前端按 rec.assistant.usage / .text 消费会崩在 nil 上）。
			for i, r := range tl.BetweenTurns {
				if r.Kind == "assistant" && r.Assistant == nil {
					t.Fatalf("BetweenTurns[%d] assistant 子结构为 nil: %+v", i, r)
				}
			}
			// Turns 非 nil（Go nil 切片序列化成 JSON null，前端按数组消费会崩）。
			if tl.Turns == nil {
				t.Fatalf("Turns 为 nil（应为空切片）")
			}
		})
	}
}

// TestFoldTrajectoryPartialLogsKeepPayload 降级必须保住信息：usage 的用量、
// text 的正文、reasoning 的推理都要落在轮间 assistant 记录上（而不是被静默
// 丢弃或只留空壳）。
func TestFoldTrajectoryPartialLogsKeepPayload(t *testing.T) {
	t.Run("首条即 usage 保住用量", func(t *testing.T) {
		var tl Trajectory
		noPanic(t, func() {
			tl = FoldTrajectory([]session.LogEntry{
				entry(1, "usage", map[string]any{
					"promptTokens": 1200, "completionTokens": 40,
					"cacheHitTokens": 1000, "cacheMissTokens": 200, "reasoningTokens": 7,
				}),
			})
		})
		if len(tl.BetweenTurns) != 1 {
			t.Fatalf("BetweenTurns = %+v, want 1 条 assistant", tl.BetweenTurns)
		}
		u := tl.BetweenTurns[0].Assistant.Usage
		if u == nil || u.PromptTokens != 1200 || u.CompletionTokens != 40 ||
			u.CacheHitTokens != 1000 || u.CacheMissTokens != 200 || u.ReasoningTokens != 7 {
			t.Fatalf("usage 未如实降级: %+v", u)
		}
	})

	t.Run("缺 turn_started 的推理+正文+用量合并成一条", func(t *testing.T) {
		var tl Trajectory
		noPanic(t, func() {
			tl = FoldTrajectory([]session.LogEntry{
				entry(1, "reasoning", map[string]any{"text": "先看一眼"}),
				entry(2, "message", map[string]any{"text": "结论如下"}),
				entry(3, "usage", map[string]any{"promptTokens": 33, "completionTokens": 4}),
			})
		})
		if len(tl.BetweenTurns) != 1 {
			t.Fatalf("连续无回合增量应合并为 1 条: %+v", tl.BetweenTurns)
		}
		asst := tl.BetweenTurns[0].Assistant
		if asst == nil || asst.Reasoning != "先看一眼" || asst.Text != "结论如下" ||
			asst.Usage == nil || asst.Usage.PromptTokens != 33 {
			t.Fatalf("降级记录内容不对: %+v", asst)
		}
	})

	t.Run("两段被截断的尾部不互相覆盖", func(t *testing.T) {
		var tl Trajectory
		noPanic(t, func() {
			tl = FoldTrajectory([]session.LogEntry{
				entry(1, "turn_started", map[string]any{}),
				entry(2, "user_message", map[string]any{"content": "a"}),
				entry(3, "turn_done", map[string]any{"err": ""}),
				entry(4, "usage", map[string]any{"promptTokens": 10, "completionTokens": 1}),
				entry(5, "turn_started", map[string]any{}),
				entry(6, "user_message", map[string]any{"content": "b"}),
				entry(7, "turn_done", map[string]any{"err": ""}),
				entry(8, "usage", map[string]any{"promptTokens": 99, "completionTokens": 9}),
			})
		})
		if len(tl.BetweenTurns) != 2 {
			t.Fatalf("两段被截断的尾部应各成一条轮间记录: %+v", tl.BetweenTurns)
		}
		if tl.BetweenTurns[0].Assistant.Usage.PromptTokens != 10 ||
			tl.BetweenTurns[1].Assistant.Usage.PromptTokens != 99 {
			t.Fatalf("两段用量串台: %+v / %+v", tl.BetweenTurns[0].Assistant.Usage, tl.BetweenTurns[1].Assistant.Usage)
		}
	})
}

// TestFoldRecordDurationsSurviveSliceGrowth 同族收口（同文件「从 helper 取出
// 指针再解引用」类）：f.cur.Records 每次增长都会重新分配底层数组，早先取出的
// &Records[i] 指针随即失效——经失效指针写 Record **自身**字段（DurationMs）
// 会静默写进废弃数组。改用「下标 + 取值时重算」后必须写进最终记录。
//
// 改前探针证据（同样输入）：toolByID[t1]=0x…4150 ≠ &Records[3]=0x…4558，
// 该失效指针对上的 durationMs=2000，而最终 records[3].durationMs=0；
// assistant records[2].durationMs 同样应为 5000 却为 0（usage 迟到时指针已失效）。
// 注：Tool/Assistant 子结构是堆上指针，内容写在旧代码里侥幸可见——本条只锁
// Record 级字段（DurationMs）。
func TestFoldRecordDurationsSurviveSliceGrowth(t *testing.T) {
	tl := FoldTrajectory([]session.LogEntry{
		entry(1, "turn_started", map[string]any{}),
		entry(2, "user_message", map[string]any{"content": "u"}),
		entry(3, "request_header", map[string]any{"system": "sys"}),
		entry(4, "reasoning", map[string]any{"text": "r1"}),                   // assistant 记录（cap4 数组）
		entry(5, "tool_dispatch", map[string]any{"id": "t1", "name": "bash"}), // 填满 cap4
		entry(6, "tool_dispatch", map[string]any{"id": "t2", "name": "bash"}), // 增长到 cap8 → 早先指针失效
		entry(7, "tool_result", map[string]any{"id": "t1", "name": "bash", "output": "out1"}),
		entry(8, "usage", map[string]any{"promptTokens": 10, "completionTokens": 1}),
	})
	if len(tl.Turns) != 1 {
		t.Fatalf("turns = %d, want 1", len(tl.Turns))
	}
	var asst *Record
	for i := range tl.Turns[0].Records {
		if tl.Turns[0].Records[i].Kind == "assistant" {
			asst = &tl.Turns[0].Records[i]
		}
	}
	if asst == nil || asst.Assistant == nil || asst.Assistant.Usage == nil {
		t.Fatalf("assistant 记录缺失/不完整: %+v", asst)
	}
	if asst.DurationMs != 5000 {
		t.Fatalf("assistant DurationMs = %d, want 5000（header ts=…003 → usage ts=…008）", asst.DurationMs)
	}
	var tool *Record
	for i := range tl.Turns[0].Records {
		if tl.Turns[0].Records[i].Tool != nil && tl.Turns[0].Records[i].Tool.ID == "t1" {
			tool = &tl.Turns[0].Records[i]
		}
	}
	if tool == nil || tool.Tool.Output != "out1" || tool.Tool.Status != "ok" {
		t.Fatalf("t1 记录未合并结果: %+v", tool)
	}
	if tool.DurationMs != 2000 {
		t.Fatalf("t1 DurationMs = %d, want 2000（dispatch ts=…005 → result ts=…007）", tool.DurationMs)
	}
}

// TestFoldTrajectoryTruncatedTailAfterTurnDone 日志尾部被截断（turn_done 之后
// 残着 usage/text）不得与轮内记录串台：轮内记录保持原样，尾部增量落轮间。
func TestFoldTrajectoryTruncatedTailAfterTurnDone(t *testing.T) {
	var tl Trajectory
	noPanic(t, func() {
		tl = FoldTrajectory([]session.LogEntry{
			entry(1, "turn_started", map[string]any{}),
			entry(2, "user_message", map[string]any{"content": "hi"}),
			entry(3, "usage", map[string]any{"promptTokens": 10, "completionTokens": 1}),
			entry(4, "turn_done", map[string]any{"err": ""}),
			entry(5, "text", map[string]any{"text": "轮后残片"}),
			entry(6, "usage", map[string]any{"promptTokens": 99, "completionTokens": 9}),
		})
	})
	if len(tl.Turns) != 1 {
		t.Fatalf("turns = %d, want 1", len(tl.Turns))
	}
	var inTurn *AssistantRec
	for i := range tl.Turns[0].Records {
		if tl.Turns[0].Records[i].Kind == "assistant" {
			inTurn = tl.Turns[0].Records[i].Assistant
		}
	}
	if inTurn == nil || inTurn.Usage == nil || inTurn.Usage.PromptTokens != 10 {
		t.Fatalf("轮内 assistant 记录被尾部残片污染: %+v", inTurn)
	}
	if len(tl.BetweenTurns) != 1 || tl.BetweenTurns[0].Kind != "assistant" {
		t.Fatalf("尾部残片应落轮间: %+v", tl.BetweenTurns)
	}
	tail := tl.BetweenTurns[0].Assistant
	if tail == nil || tail.Text != "轮后残片" || tail.Usage == nil || tail.Usage.PromptTokens != 99 {
		t.Fatalf("轮间尾部记录内容不对: %+v", tail)
	}
}
