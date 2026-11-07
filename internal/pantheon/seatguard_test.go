package pantheon

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// 2026-11-06 卡死/缓存两刀的行为锁：
//   - DSH 席位命令行必须显式带只读与审批策略，续接时带 --session-id；
//   - 引擎自报的 step_end usage 要能解析成可核对账目（缓存命中率有分母）；
//   - 工具预算与无进展看门狗必须真的取消调用，且不能被「空结果自动重试」吃掉。

func TestDSHSeatArgsDeclareReadOnlyAndApproval(t *testing.T) {
	plain := strings.Join(dshSeatArgs(""), " ")
	for _, want := range []string{
		"--profile headless", "--json", "--permission-mode read-only", "--approval-policy never",
	} {
		if !strings.Contains(plain, want) {
			t.Fatalf("DSH 席位命令行缺少 %q：%s", want, plain)
		}
	}
	if strings.Contains(plain, "--session-id") {
		t.Fatalf("首次调用不得带 --session-id（未知 ID 是引擎硬错误）：%s", plain)
	}
	if !strings.HasSuffix(plain, " -") {
		t.Fatalf("任务必须从 stdin 读（最后一个参数为 -）：%s", plain)
	}

	resumed := strings.Join(dshSeatArgs("session-abc"), " ")
	if !strings.Contains(resumed, "--session-id session-abc") {
		t.Fatalf("续接调用必须带引擎会话 ID：%s", resumed)
	}
}

func TestParseSeatRunStatsReadsSessionAndUsage(t *testing.T) {
	raw := strings.Join([]string{
		`{"type":"session","sessionId":"session-1f77","cwd":"C:\\AI\\wubigrok"}`,
		`{"type":"status","phase":"turn_start","turn":1}`,
		`{"type":"status","phase":"step_end","turn":1,"step":1,"usage":{"inputTokens":606,"outputTokens":79,"cacheReadTokens":5632}}`,
		`{"type":"thinking","text":""}`,
		`{"type":"status","phase":"step_end","turn":1,"step":2,"usage":{"inputTokens":419,"outputTokens":47,"cacheReadTokens":6272}}`,
	}, "\n")
	usage := parseSeatRunStats(raw)
	if usage.EngineSessionID != "session-1f77" {
		t.Fatalf("引擎会话 ID = %q", usage.EngineSessionID)
	}
	if usage.Steps != 2 {
		t.Fatalf("步数 = %d, want 2", usage.Steps)
	}
	if usage.InputTokens != 1025 || usage.CacheReadTokens != 11904 || usage.OutputTokens != 126 {
		t.Fatalf("token 账目 = in %d / cache %d / out %d",
			usage.InputTokens, usage.CacheReadTokens, usage.OutputTokens)
	}
	// 命中率=11904/(11904+1025)≈92.1%，不是「每步各自平均」。
	if hit := usage.CacheHitPercent(); hit < 91 || hit > 93 {
		t.Fatalf("缓存命中率 = %.1f%%, want ~92%%", hit)
	}

	// 引擎没上报时保持 -1（未上报），不能用 0 冒充「0% 命中」。
	if empty := parseSeatRunStats("纯文本陈词"); empty.CacheHitPercent() != -1 || empty.Steps != 0 {
		t.Fatalf("无账目时不应编造命中率：%+v", empty)
	}
}

func TestSeatGuardStopsWhenNoTextEverStarts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// 工具/thinking 一直有活性，但始终不写正文：这正是「一轮 87 次工具调用、
	// 六分钟不开口」的实弹形态，必须由开局静默闸终止。
	guard := newSeatGuard(cancel, 100, time.Minute, 40*time.Millisecond)
	defer guard.Close()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		guard.NoteTool("t1") // 有活性，但不算正文
		if errors.Is(guard.Err(), errSeatNoFirstText) {
			if ctx.Err() == nil {
				t.Fatal("开局静默闸报错时必须已取消调用")
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("开局静默闸没有触发")
}

func TestParseSeatRunStatsReadsZCodeResult(t *testing.T) {
	// ZCode --output-format stream-json 的收尾行（2026-11-06 实弹形状）。
	raw := `{"type":"result","sessionId":"sess_e054867d-70a6-46e7-9e93-c39dcba925c1",` +
		`"response":"gaea 是…","usage":{"source":"provider","modelRequestCount":2,` +
		`"inputTokens":67412,"outputTokens":309,"totalTokens":67721,"cacheReadTokens":37504}}`
	usage := parseSeatRunStats(raw)
	if usage.EngineSessionID != "sess_e054867d-70a6-46e7-9e93-c39dcba925c1" {
		t.Fatalf("引擎会话 ID = %q", usage.EngineSessionID)
	}
	if usage.Steps != 2 {
		t.Fatalf("步数（模型请求数）= %d, want 2", usage.Steps)
	}
	if usage.InputTokens != 29908 || usage.CacheReadTokens != 37504 || usage.OutputTokens != 309 {
		t.Fatalf("token 账目 = 未命中 %d / 命中 %d / 输出 %d",
			usage.InputTokens, usage.CacheReadTokens, usage.OutputTokens)
	}
	if hit := usage.CacheHitPercent(); hit < 55 || hit > 56 {
		t.Fatalf("缓存命中率 = %.1f%%, want ~55.6%%（inputTokens 含缓存，不能重复计入分母）", hit)
	}
}

func TestParseSeatRunStatsReadsCodexThreadStarted(t *testing.T) {
	// codex exec --json 的会话 ID 载体（2026-11-07 A5② 探针实锤形状）。
	raw := `{"type":"thread.started","thread_id":"01a1b609-a539-74a0-aee5-f3e152643663"}` + "\n" +
		`{"type":"item.completed","item":{"id":"item_2","type":"agent_message","text":"好的"}}` + "\n" +
		`{"type":"turn.completed","usage":{"input_tokens":14087,"cached_input_tokens":12800,"output_tokens":2}}`
	usage := parseSeatRunStats(raw)
	if usage.EngineSessionID != "01a1b609-a539-74a0-aee5-f3e152643663" {
		t.Fatalf("codex thread id 未被采集: %q", usage.EngineSessionID)
	}
	// codex token 账目暂无同形载体（input_tokens 蛇形命名未接），保持零值
	// 而不是估算——「未上报」与「真的 0」是两回事。
	if usage.InputTokens != 0 || usage.Steps != 0 {
		t.Fatalf("codex token 账目应保持零值: %+v", usage)
	}
}

func TestZCodeSeatResumesOnlyWithKnownSession(t *testing.T) {
	resolve := func() (string, string, error) { return "node.exe", "zcode.cjs", nil }
	plain, err := zcodeCommandSpec(resolve, "议题", `C:\work`, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(plain.Args, " "), "--resume") {
		t.Fatal("首次调用不应带 --resume（未知会话 ID 是引擎硬错误）")
	}
	resumed, err := zcodeCommandSpec(resolve, "议题", `C:\work`, nil, "sess_abc")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(resumed.Args, " "), "--resume sess_abc") {
		t.Fatalf("续接调用必须带引擎会话 ID：%v", resumed.Args)
	}
}

func TestSeatGuardStopsOnToolBudget(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	guard := newSeatGuard(cancel, 2, time.Minute, time.Minute)
	defer guard.Close()

	guard.NoteTool("t1")
	guard.NoteTool("t1") // 同一工具的 started/completed 只算一次
	if err := guard.Err(); err != nil {
		t.Fatalf("预算内不应报错：%v", err)
	}
	guard.NoteTool("t2")
	guard.NoteTool("t3") // 第 3 次 > 上限 2
	err := guard.Err()
	if !errors.Is(err, errSeatToolBudget) {
		t.Fatalf("err = %v, want errSeatToolBudget", err)
	}
	if ctx.Err() == nil {
		t.Fatal("超预算必须取消整次调用")
	}
}

func TestSeatGuardStopsOnNoProgress(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	guard := newSeatGuard(cancel, 100, 30*time.Millisecond, time.Minute)
	defer guard.Close()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if errors.Is(guard.Err(), errSeatStalled) {
			if ctx.Err() == nil {
				t.Fatal("看门狗报错时必须已取消调用")
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("看门狗没有在无进展时触发")
}

func TestSeatGuardActivityKeepsWatchdogAlive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	guard := newSeatGuard(cancel, 100, 60*time.Millisecond, time.Minute)
	defer guard.Close()

	// 每 20ms 喂一次活性：总时长超过 stall，但不是「无进展」。
	for i := 0; i < 5; i++ {
		time.Sleep(20 * time.Millisecond)
		guard.NoteActivity()
	}
	if err := guard.Err(); err != nil {
		t.Fatalf("持续有事件时不应判卡死：%v", err)
	}
	if ctx.Err() != nil {
		t.Fatal("持续有事件时不应取消调用")
	}
}

// dshSeatFake 模拟 DSH headless 的事件流：先报 session，再报 step_end usage，
// 最后给一段正文。第二次起按命令行里是否带 --session-id 报不同会话。
type dshSeatFake struct {
	mu    sync.Mutex
	specs []commandSpec
}

func (r *dshSeatFake) Run(_ context.Context, spec commandSpec) (string, error) {
	r.mu.Lock()
	r.specs = append(r.specs, spec)
	r.mu.Unlock()
	id := "session-first"
	if strings.Contains(strings.Join(spec.Args, " "), "--session-id") {
		id = "session-resumed"
	}
	return strings.Join([]string{
		`{"type":"session","sessionId":"` + id + `","cwd":"x"}`,
		`{"type":"status","phase":"step_end","turn":1,"step":1,"usage":{"inputTokens":100,"outputTokens":10,"cacheReadTokens":900}}`,
		`{"type":"text","text":"陈词正文：同意"} `,
	}, "\n"), nil
}

func newDSHFakeAdapter(runner *dshSeatFake) agentAdapter {
	status := AgentStatus{
		ID: "dsh", Name: "DeepSeek Harness", Kind: "fake", Available: true,
		Capabilities: []string{"session-resume"},
	}
	build := func(prompt string, cwd string, resumeID string) (commandSpec, error) {
		args := []string{"--profile", "headless", "--json"}
		if resumeID != "" {
			args = append(args, "--session-id", resumeID)
		}
		// Stdin 必须携带提示词：增量形态断言要按 Stdin 区分讨论/汇总调用
		// （真实 DSH adapter 同样走 Stdin）。
		return commandSpec{Executable: "dsh", Args: args, Dir: cwd, Stdin: clampPrompt(prompt)}, nil
	}
	return agentAdapter{
		id: "dsh", name: status.Name, kind: status.Kind,
		probe: func(context.Context) AgentStatus { return status },
		build: func(prompt, cwd string, _ []string) (commandSpec, error) {
			return build(prompt, cwd, "")
		},
		buildResume: func(prompt, cwd string, _ []string, resumeID string) (commandSpec, error) {
			return build(prompt, cwd, resumeID)
		},
	}
}

func waitSessionSettled(t *testing.T, m *Manager, id string) *Session {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		session, err := m.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		switch session.Status {
		case StatusCompleted, StatusFailed, StatusCancelled:
			return session
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("议事没有在 5s 内收敛")
	return nil
}

func TestDSHSeatReusesEngineSession(t *testing.T) {
	oldBudget, oldStall := seatToolBudget, seatStallLimit
	seatToolBudget, seatStallLimit = 100, time.Minute
	t.Cleanup(func() { seatToolBudget, seatStallLimit = oldBudget, oldStall })

	runner := &dshSeatFake{}
	m := newTestManager(t, runner, newDSHFakeAdapter(runner))
	session, err := m.Start(StartRequest{
		Topic: "缓存命中验证", AgentIDs: []string{"dsh"},
		Rounds: 1, Protocol: ProtocolIndependent,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitSessionSettled(t, m, session.ID)
	if done.Status != StatusCompleted {
		t.Fatalf("status = %s, error = %s", done.Status, done.Error)
	}

	runner.mu.Lock()
	specs := append([]commandSpec(nil), runner.specs...)
	runner.mu.Unlock()
	if len(specs) < 2 {
		t.Fatalf("至少应有开场和汇总两次调用，实际 %d", len(specs))
	}
	if strings.Contains(strings.Join(specs[0].Args, " "), "--session-id") {
		t.Fatal("首次调用不应续接，否则引擎会拒绝未知会话")
	}
	resumed := 0
	for _, spec := range specs[1:] {
		if strings.Contains(strings.Join(spec.Args, " "), "--session-id session-first") {
			resumed++
		}
	}
	if resumed != len(specs)-1 {
		t.Fatalf("后续调用都应续接同一引擎会话，实际 %d/%d", resumed, len(specs)-1)
	}

	// 席位账目落进 Contribution：会话 ID + 步数 + 缓存命中可核对。
	var contribution *Contribution
	for i := range done.Contributions {
		if done.Contributions[i].Usage != nil {
			contribution = &done.Contributions[i]
			break
		}
	}
	if contribution == nil {
		t.Fatal("Contribution 缺少引擎账目（无法核对是否真的调用 DSH）")
	}
	if contribution.Usage.EngineSessionID == "" || contribution.Usage.Steps != 1 {
		t.Fatalf("账目 = %+v", *contribution.Usage)
	}
	if hit := contribution.Usage.CacheHitPercent(); hit < 89 || hit > 91 {
		t.Fatalf("缓存命中率 = %.1f%%, want 90%%", hit)
	}

	// 引擎会话 ID 同时落回 Participant，供续跑/界面核对。
	for _, participant := range done.Participants {
		if participant.AgentID == "dsh" && participant.EngineSessionID == "" {
			t.Fatal("Participant 没有记住引擎会话 ID")
		}
	}
}

func TestDSHDiscussionRoundUsesDeltaPrompt(t *testing.T) {
	oldBudget, oldStall := seatToolBudget, seatStallLimit
	seatToolBudget, seatStallLimit = 100, time.Minute
	t.Cleanup(func() { seatToolBudget, seatStallLimit = oldBudget, oldStall })

	runner := &dshSeatFake{}
	m := newTestManager(t, runner, newDSHFakeAdapter(runner))
	session, err := m.Start(StartRequest{
		Topic: "增量形态验证", AgentIDs: []string{"dsh"},
		Rounds: 2, Protocol: ProtocolIndependent,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitSessionSettled(t, m, session.ID)
	if done.Status != StatusCompleted {
		t.Fatalf("status = %s, error = %s", done.Status, done.Error)
	}
	runner.mu.Lock()
	specs := append([]commandSpec(nil), runner.specs...)
	runner.mu.Unlock()
	var discussion, synthesis *commandSpec
	for i := range specs {
		switch {
		case strings.Contains(specs[i].Stdin, "补充论证"):
			discussion = &specs[i]
		case strings.Contains(specs[i].Stdin, "协助整理"):
			synthesis = &specs[i]
		}
	}
	if discussion == nil || synthesis == nil {
		t.Fatalf("缺讨论或汇总调用，specs = %d", len(specs))
	}
	// A5①：讨论轮走增量形态——引擎侧会话活跃，往轮梯度不再重发。
	if strings.Contains(discussion.Stdin, "立场卡节选") || strings.Contains(discussion.Stdin, "归档发言") {
		t.Fatalf("delta 讨论提示词不得带梯度：%q", discussion.Stdin)
	}
	if !strings.Contains(discussion.Stdin, "引擎侧历史") || !strings.Contains(discussion.Stdin, "自你上次发言以来没有新增发言或插话") {
		t.Fatalf("delta 讨论提示词缺增量形态标记：%q", discussion.Stdin)
	}
	// 汇总保持梯度形态（CLI 席的引擎会话只见过梯度视图，汇总要全量记录在场）。
	if !strings.Contains(synthesis.Stdin, "多轮讨论记录") || !strings.Contains(synthesis.Stdin, "立场卡节选") {
		t.Fatalf("汇总提示词应保持梯度形态：%q", synthesis.Stdin)
	}
}

// toolSpamRunner 模拟「一席连开 5 次工具」的自主循环。
type toolSpamRunner struct {
	mu    sync.Mutex
	tools int
}

func (r *toolSpamRunner) Run(context.Context, commandSpec) (string, error) {
	return "", nil
}

func (r *toolSpamRunner) RunStreamWithTools(
	_ context.Context,
	_ commandSpec,
	_ func(string),
	onTool func(ToolTrace),
) (string, error) {
	r.mu.Lock()
	r.tools = 0
	r.mu.Unlock()
	for i := 0; i < 5; i++ {
		r.mu.Lock()
		r.tools++
		r.mu.Unlock()
		onTool(ToolTrace{ID: fmt.Sprintf("cmd-%d", i), Name: "pwsh", Status: "started"})
		onTool(ToolTrace{ID: fmt.Sprintf("cmd-%d", i), Name: "pwsh", Status: "completed"})
	}
	return "", nil
}

func TestSeatToolBudgetTerminatesRunawaySeat(t *testing.T) {
	oldBudget, oldStall := seatToolBudget, seatStallLimit
	seatToolBudget, seatStallLimit = 2, time.Minute
	t.Cleanup(func() { seatToolBudget, seatStallLimit = oldBudget, oldStall })

	runner := &toolSpamRunner{}
	adapter := agentAdapter{
		id: "dsh", name: "DeepSeek Harness", kind: "fake",
		probe: func(context.Context) AgentStatus {
			return AgentStatus{ID: "dsh", Name: "DeepSeek Harness", Available: true}
		},
		build: func(_ string, cwd string, _ []string) (commandSpec, error) {
			return commandSpec{Executable: "dsh", Dir: cwd}, nil
		},
	}
	m := newTestManager(t, runner, adapter)
	session, err := m.Start(StartRequest{
		Topic: " runaway ", AgentIDs: []string{"dsh"}, Rounds: 1, Protocol: ProtocolIndependent,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := waitSessionSettled(t, m, session.ID)
	if done.Status == StatusCompleted {
		t.Fatal("超预算的席位不应产出完成态议事")
	}
	budgetErrors, retries := 0, 0
	for _, event := range done.Events {
		if event.Kind == EventAgentError && strings.Contains(event.Error, "工具调用超出议事预算") {
			budgetErrors++
		}
		if event.Kind == EventAgentRetry {
			retries++
		}
	}
	if budgetErrors == 0 {
		t.Fatalf("没有记录「预算耗尽」错误：%+v", done.Warnings)
	}
	if retries != 0 {
		t.Fatalf("预算失败不得触发自动重试（实际 %d 次）", retries)
	}
}
