package pantheon

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultManualRounds    = 2
	maxManualRounds        = 20
	defaultConsensusLimit  = 12
	maxConsensusRoundLimit = 20
)

var consensusMarker = regexp.MustCompile(`(?im)^\s*(?:[-*>]\s*)*(?:共识状态|consensus)\s*[:：]\s*(一致|同意|仍有分歧|不同意|yes|no)`)

type Manager struct {
	root    string
	workDir string

	mu        sync.Mutex
	store     *sessionStore
	runner    commandRunner
	overrides map[string]string
	adapters  map[string]agentAdapter
	grok      GrokBackend
	cancels   map[string]context.CancelFunc
	wg        sync.WaitGroup
	emit      func(Event)
	deltaSeq  int64
}

func NewManager(root string, runner commandRunner, emit func(Event)) (*Manager, error) {
	store, err := newSessionStore(root)
	if err != nil {
		return nil, err
	}
	overrides, err := store.loadOverrides()
	if err != nil {
		return nil, err
	}
	if runner == nil {
		runner = osCommandRunner{}
	}
	return &Manager{
		root:      root,
		workDir:   root,
		store:     store,
		runner:    runner,
		overrides: overrides,
		adapters:  buildAdapters(overrides, nil),
		cancels:   map[string]context.CancelFunc{},
		emit:      emit,
	}, nil
}

// RecoverInterrupted closes sessions left in the running state by a previous
// process. A council goroutine cannot survive an app restart, so presenting
// those records as active would leave "cancel" buttons that can never succeed.
// Partial contributions and events remain intact for later review.
func (m *Manager) RecoverInterrupted() (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	summaries, err := m.store.list()
	if err != nil {
		return 0, err
	}
	recovered := 0
	for _, summary := range summaries {
		if summary.Status != StatusRunning {
			continue
		}
		session, err := m.store.get(summary.ID)
		if err != nil {
			return recovered, err
		}
		now := time.Now().UnixMilli()
		reason := "上次运行被应用中断，已保留此前产生的发言与事件"
		session.Status = StatusFailed
		session.Phase = PhaseFailed
		session.Error = reason
		session.UpdatedAt = now
		for i := range session.Participants {
			if session.Participants[i].Status == "running" {
				session.Participants[i].Status = "failed"
				session.Participants[i].FinishedAt = now
				session.Participants[i].Error = "应用重启，本次调用被中断"
			}
		}
		session.Warnings = append(session.Warnings, reason)
		m.appendEventLocked(session, Event{
			Phase: PhaseFailed,
			Kind:  EventError,
			Text:  "议事因应用重启而中断",
			Error: reason,
			At:    now,
		})
		if err := m.store.save(session); err != nil {
			return recovered, err
		}
		recovered++
	}
	return recovered, nil
}

// SetGrokBackend 注入 Grok 模型席位后端（app 层装配时调用一次；传 nil 移除）。
func (m *Manager) SetGrokBackend(backend GrokBackend) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.grok = backend
	m.adapters = buildAdapters(m.overrides, backend)
}

func (m *Manager) DataRoot() string { return m.root }

// SetWorkDir changes the read-only directory exposed to external harnesses.
// Gaea uses its active office workspace so agent-native project configuration
// and credential lookup keep working without Gaea reading secret files.
func (m *Manager) SetWorkDir(dir string) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return
	}
	m.mu.Lock()
	m.workDir = dir
	m.mu.Unlock()
}

func (m *Manager) Agents(ctx context.Context) []AgentStatus {
	m.mu.Lock()
	adapters := make([]agentAdapter, 0, len(m.adapters))
	ids := make([]string, 0, len(m.adapters))
	for id := range m.adapters {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		priority := func(id string) int {
			switch id {
			case "grok":
				return 0
			case "codex":
				return 1
			case "zcode":
				return 2
			case "dsh":
				return 3
			default:
				return 10
			}
		}
		pi, pj := priority(ids[i]), priority(ids[j])
		if pi == pj {
			return ids[i] < ids[j]
		}
		return pi < pj
	})
	for _, id := range ids {
		adapters = append(adapters, m.adapters[id])
	}
	m.mu.Unlock()

	out := make([]AgentStatus, 0, len(adapters))
	for _, adapter := range adapters {
		probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		status := adapter.probe(probeCtx)
		cancel()
		out = append(out, status)
	}
	return out
}

func (m *Manager) SetExecutable(id, path string) ([]AgentStatus, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	path = strings.TrimSpace(path)
	m.mu.Lock()
	if _, ok := m.adapters[id]; !ok {
		m.mu.Unlock()
		return nil, fmt.Errorf("未知 Agent %q", id)
	}
	if m.adapters[id].invoke != nil {
		m.mu.Unlock()
		return nil, errors.New("该席位由 Gaea 办公 AgentRunner 驱动，无需配置可执行路径")
	}
	if path != "" {
		if !filepath.IsAbs(path) {
			m.mu.Unlock()
			return nil, errors.New("Agent 路径必须是绝对路径")
		}
		if !executableExists(path) {
			m.mu.Unlock()
			return nil, errors.New("Agent 路径不存在")
		}
		m.overrides[id] = path
	} else {
		delete(m.overrides, id)
	}
	overrides := cloneStringMap(m.overrides)
	if err := m.store.saveOverrides(overrides); err != nil {
		m.mu.Unlock()
		return nil, err
	}
	m.adapters = buildAdapters(overrides, m.grok)
	m.mu.Unlock()
	return m.Agents(context.Background()), nil
}

func (m *Manager) List() ([]SessionSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.store.list()
}

func (m *Manager) Get(id string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.store.get(id)
}

func (m *Manager) Start(req StartRequest) (*Session, error) {
	req.Topic = strings.TrimSpace(req.Topic)
	if req.Topic == "" {
		return nil, errors.New("议题不能为空")
	}
	workspace, err := m.resolveWorkspace(req.Workspace)
	if err != nil {
		return nil, err
	}
	req.Workspace = workspace
	normalizeRoundSettings(&req)
	req.ReadOnly = true // first release has no workspace-write debate mode
	if strings.TrimSpace(req.Title) == "" {
		req.Title = titleFromTopic(req.Topic)
	}

	probeCtx, cancelProbe := context.WithTimeout(context.Background(), 15*time.Second)
	statuses := m.Agents(probeCtx)
	cancelProbe()
	byID := map[string]AgentStatus{}
	for _, status := range statuses {
		byID[status.ID] = status
	}

	selectedIDs := uniqueNonEmpty(req.AgentIDs)
	if len(selectedIDs) == 0 {
		for _, id := range []string{"grok", "codex", "zcode", "dsh"} {
			if status, ok := byID[id]; ok && status.Available {
				selectedIDs = append(selectedIDs, id)
			}
		}
	}
	if len(selectedIDs) < 2 {
		return nil, errors.New("众神殿至少需要选择两个 Agent")
	}
	for _, id := range selectedIDs {
		if _, ok := byID[id]; !ok {
			return nil, fmt.Errorf("未知 Agent %q", id)
		}
	}

	participants := make([]Participant, 0, len(selectedIDs))
	available := map[string]AgentStatus{}
	for _, id := range selectedIDs {
		status := byID[id]
		p := Participant{AgentID: id, Name: status.Name, Role: "participant", Status: "waiting"}
		if !status.Available {
			p.Status = "unavailable"
			p.Error = status.Reason
		} else {
			available[id] = status
		}
		participants = append(participants, p)
	}
	if len(available) < 2 {
		return nil, errors.New("可用 Agent 少于两个，无法开始讨论")
	}

	chairman := strings.TrimSpace(req.ChairmanAgentID)
	if chairman == "" {
		chairman = pickChairman(selectedIDs, available)
	}
	if _, ok := available[chairman]; !ok {
		chairman = pickChairman(selectedIDs, available)
	}
	req.ChairmanAgentID = chairman

	now := time.Now().UnixMilli()
	session := &Session{
		ID:           newSessionID(now),
		Title:        req.Title,
		Status:       StatusRunning,
		Phase:        PhasePreparing,
		CreatedAt:    now,
		UpdatedAt:    now,
		Request:      req,
		Participants: participants,
		Events:       []Event{},
	}
	for _, p := range participants {
		if p.Status == "unavailable" {
			session.Warnings = append(session.Warnings, fmt.Sprintf("%s 不可用：%s", p.Name, p.Error))
		}
	}
	if err := m.store.save(session); err != nil {
		return nil, err
	}
	m.emitEvent(session, Event{Phase: PhasePreparing, Kind: EventSessionStart, Text: "众神殿圆桌讨论开始"})

	runCtx, cancel := context.WithCancel(context.Background())
	m.mu.Lock()
	m.cancels[session.ID] = cancel
	m.wg.Add(1)
	m.mu.Unlock()
	go func() {
		defer m.wg.Done()
		defer func() {
			m.mu.Lock()
			delete(m.cancels, session.ID)
			m.mu.Unlock()
			cancel()
		}()
		m.run(runCtx, session, available, chairman)
	}()
	return cloneSession(session), nil
}

func normalizeRoundSettings(req *StartRequest) {
	if req == nil {
		return
	}
	if req.UntilConsensus {
		if req.MaxRounds < 2 {
			req.MaxRounds = defaultConsensusLimit
		}
		if req.MaxRounds > maxConsensusRoundLimit {
			req.MaxRounds = maxConsensusRoundLimit
		}
		req.Rounds = req.MaxRounds
		return
	}
	req.MaxRounds = 0
	if req.Rounds < 2 {
		req.Rounds = defaultManualRounds
	}
	if req.Rounds > maxManualRounds {
		req.Rounds = maxManualRounds
	}
}

func (m *Manager) resolveWorkspace(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		m.mu.Lock()
		raw = m.workDir
		m.mu.Unlock()
	}
	if raw == "" {
		raw = m.root
	}
	abs, err := filepath.Abs(raw)
	if err != nil {
		return "", fmt.Errorf("解析议事工作空间失败: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("议事工作空间不可访问: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("议事工作空间不是目录: %s", abs)
	}
	return filepath.Clean(abs), nil
}

func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	cancel, ok := m.cancels[id]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("议事 %s 未在运行", id)
	}
	cancel()
	return nil
}

func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	_, running := m.cancels[id]
	m.mu.Unlock()
	if running {
		return errors.New("议事仍在运行，请先终止")
	}
	return m.store.delete(id)
}

func (m *Manager) Export(id string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, err := m.store.get(id)
	if err != nil {
		return "", err
	}
	return ExportMarkdown(session), nil
}

func (m *Manager) SignOff(id, decision, note string) (*Session, error) {
	decision = strings.TrimSpace(strings.ToLower(decision))
	switch decision {
	case "accepted", "needs_revision", "rejected":
	default:
		return nil, errors.New("讨论小结状态仅支持 accepted / needs_revision / rejected")
	}
	m.mu.Lock()
	session, err := m.store.get(id)
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	if session.Status == StatusRunning {
		m.mu.Unlock()
		return nil, errors.New("讨论仍在进行，暂不能确认小结")
	}
	now := time.Now().UnixMilli()
	session.SignOff = &SignOff{Decision: decision, Note: strings.TrimSpace(note), At: now}
	session.UpdatedAt = now
	label := map[string]string{
		"accepted": "确认讨论小结", "needs_revision": "标记为需继续讨论", "rejected": "暂不采纳讨论小结",
	}[decision]
	event := m.appendEventLocked(session, Event{
		Phase: session.Phase, Kind: EventSignOff, Text: label, At: now,
	})
	if err := m.store.save(session); err != nil {
		m.mu.Unlock()
		return nil, err
	}
	out := cloneSession(session)
	m.mu.Unlock()
	m.emitSafe(event)
	return out, nil
}

func (m *Manager) Close() {
	m.mu.Lock()
	for _, cancel := range m.cancels {
		cancel()
	}
	m.mu.Unlock()
	m.wg.Wait()
}

func (m *Manager) run(ctx context.Context, session *Session, available map[string]AgentStatus, chairman string) {
	order := orderedAgentIDs(session.Request.AgentIDs, available)
	history := make([]Contribution, 0, len(order)*session.Request.Rounds)
	successful := map[string]AgentStatus{}
	actualRounds := 0
	consensusRound := 0

	for round := 1; round <= session.Request.Rounds; round++ {
		actualRounds = round
		phase := PhaseDiscussion
		phaseText := fmt.Sprintf("第 %d 轮补充论证：承接前文，补充证据并澄清分歧", round)
		if round == 1 {
			phase = PhaseOpening
			phaseText = "按既定顺序开始开场陈述"
		}
		m.updateRoundPhase(session, phase, round, phaseText)
		roundOrder := rotateAgentIDs(order, round-1)
		roundContributions := make([]Contribution, 0, len(roundOrder))
		for position, agentID := range roundOrder {
			if ctx.Err() != nil {
				m.finishCancelled(session)
				return
			}
			status := available[agentID]
			var prompt string
			if round == 1 {
				prompt = openingPrompt(session.Request, status, history, position+1, len(roundOrder))
			} else {
				prompt = discussionPrompt(session.Request, status, history, round, position+1, len(roundOrder))
			}
			out, duration, err := m.runOne(ctx, session, status, prompt, phase, round)
			if err != nil {
				continue
			}
			consensus := ""
			if session.Request.UntilConsensus {
				out, consensus = extractConsensusSignal(out)
			}
			contribution := Contribution{
				ID:         fmt.Sprintf("%s-%s-r%d-%d", session.ID, status.ID, round, time.Now().UnixNano()),
				Phase:      phase,
				Round:      round,
				AgentID:    status.ID,
				AgentName:  status.Name,
				Content:    out,
				Consensus:  consensus,
				CreatedAt:  time.Now().UnixMilli(),
				DurationMs: duration.Milliseconds(),
			}
			history = append(history, contribution)
			roundContributions = append(roundContributions, contribution)
			m.appendContribution(session, contribution)
			successful[status.ID] = status
		}
		if session.Request.UntilConsensus &&
			round >= 2 &&
			roundReachedConsensus(roundContributions, len(order)) {
			consensusRound = round
			m.recordConsensus(session, round)
			break
		}
	}
	if ctx.Err() != nil {
		m.finishCancelled(session)
		return
	}
	if len(history) == 0 {
		m.finishFailed(session, "所有参与者均未返回有效发言")
		return
	}
	if len(successful) < 2 {
		m.addWarning(session, "形成有效发言的参与者少于两个，讨论小结仅供单人参考")
	}
	if session.Request.UntilConsensus &&
		consensusRound == 0 &&
		actualRounds >= session.Request.Rounds {
		m.addWarning(
			session,
			fmt.Sprintf("已达到 %d 轮安全上限，仍未全员一致；以下保留各席当前判断与未解分歧", session.Request.Rounds),
		)
	}

	m.updateRoundPhase(session, PhaseSynthesis, actualRounds, "汇总人正在整理共识、分歧与开放问题")
	summarizerStatus, ok := successful[chairman]
	if !ok {
		for _, id := range order {
			if status, exists := successful[id]; exists {
				summarizerStatus = status
				ok = true
				break
			}
		}
	}
	var verdict *Verdict
	if ok {
		out, duration, err := m.runOne(
			ctx,
			session,
			summarizerStatus,
			synthesisPrompt(session.Request, summarizerStatus, history),
			PhaseSynthesis,
			0,
		)
		if err == nil && strings.TrimSpace(out) != "" {
			verdict = parseVerdict(out, summarizerStatus.Name, duration)
			m.appendContribution(session, Contribution{
				ID:         fmt.Sprintf("%s-%s-synthesis-%d", session.ID, summarizerStatus.ID, time.Now().UnixNano()),
				Phase:      PhaseSynthesis,
				AgentID:    summarizerStatus.ID,
				AgentName:  summarizerStatus.Name,
				AnonLabel:  "协作汇总",
				Content:    out,
				CreatedAt:  time.Now().UnixMilli(),
				DurationMs: duration.Milliseconds(),
			})
		} else if err != nil && ctx.Err() == nil {
			m.addWarning(session, fmt.Sprintf("%s 协作汇总失败：%v", summarizerStatus.Name, err))
		}
	}
	if ctx.Err() != nil {
		m.finishCancelled(session)
		return
	}
	if verdict == nil {
		verdict = fallbackVerdict(history)
		m.addWarning(session, "未能形成自动小结，已保留原始发言供人工整理")
	}
	verdict.Partial = verdict.Partial || len(successful) < len(uniqueNonEmpty(session.Request.AgentIDs))

	m.mu.Lock()
	session.Verdict = verdict
	session.Phase = PhaseCompleted
	session.Status = StatusCompleted
	session.UpdatedAt = time.Now().UnixMilli()
	m.appendEventLocked(session, Event{
		Phase: PhaseCompleted, Kind: EventVerdict, Text: verdict.Recommendation,
		At: time.Now().UnixMilli(),
	})
	m.appendEventLocked(session, Event{
		Phase: PhaseCompleted, Kind: EventComplete, Text: "讨论小结已形成，等待用户确认",
		At: time.Now().UnixMilli(),
	})
	_ = m.store.save(session)
	events := cloneEvents(m.lastEventsLocked(session, 2))
	m.mu.Unlock()
	for _, event := range events {
		m.emitSafe(event)
	}
}

func (m *Manager) runOne(ctx context.Context, session *Session, status AgentStatus, prompt, phase string, round int) (string, time.Duration, error) {
	adapter := m.adapter(status.ID)
	if adapter.id == "" {
		return "", 0, fmt.Errorf("未找到 Agent Adapter %s", status.ID)
	}
	if adapter.invoke != nil {
		return m.runModelSeat(
			ctx,
			session,
			status,
			adapter.invoke,
			adapter.stream,
			adapter.streamWithTools,
			prompt,
			phase,
			round,
		)
	}
	m.mu.Lock()
	workDir := session.Request.Workspace
	if workDir == "" {
		workDir = m.workDir
	}
	m.mu.Unlock()
	if workDir == "" {
		workDir = m.root
	}
	spec, err := adapter.build(prompt, workDir)
	if err != nil {
		m.recordAgentError(session, status, phase, err)
		return "", 0, err
	}
	m.recordAgentStarted(session, status, phase, round)
	runCtx, cancel := context.WithTimeout(ctx, defaultAgentLimit)
	defer cancel()
	start := time.Now()
	onDelta := func(chunk string) {
		m.recordAgentDelta(session, status, phase, round, chunk)
	}
	onTool := func(trace ToolTrace) {
		m.recordToolTrace(session, status, phase, round, trace)
	}
	var raw string
	if runner, ok := m.runner.(toolStreamingCommandRunner); ok {
		raw, err = runner.RunStreamWithTools(runCtx, spec, onDelta, onTool)
	} else if runner, ok := m.runner.(streamingCommandRunner); ok {
		raw, err = runner.RunStream(runCtx, spec, onDelta)
	} else {
		raw, err = m.runner.Run(runCtx, spec)
	}
	duration := time.Since(start)
	if err != nil {
		m.recordAgentError(session, status, phase, err)
		return "", duration, err
	}
	text := strings.TrimSpace(parseAgentOutput(raw))
	if text == "" {
		err = errors.New("Agent 返回空结果")
		m.recordAgentError(session, status, phase, err)
		return "", duration, err
	}
	text = trimRunes(text, maxAgentOutput)
	m.recordAgentOutput(session, status, phase, round, text, duration)
	return text, duration, nil
}

// runModelSeat 模型席位执行：无 cwd、无 CLI 包装；提示词即全部输入，输出是
// 模型纯文本（不经 parseAgentOutput 的 CLI JSONL 清洗——它可能把合法 JSON
// 讨论小结误当噪音剥离）。
func (m *Manager) runModelSeat(
	ctx context.Context,
	session *Session,
	status AgentStatus,
	invoke func(context.Context, string) (string, error),
	stream func(context.Context, string, func(string)) (string, error),
	streamWithTools func(
		context.Context,
		string,
		string,
		func(string),
		func(ToolTrace),
	) (string, error),
	prompt, phase string,
	round int,
) (string, time.Duration, error) {
	m.recordAgentStarted(session, status, phase, round)
	runCtx, cancel := context.WithTimeout(ctx, defaultAgentLimit)
	defer cancel()
	start := time.Now()
	var text string
	var err error
	onTool := func(trace ToolTrace) {
		m.recordToolTrace(session, status, phase, round, trace)
	}
	if streamWithTools != nil {
		text, err = streamWithTools(
			runCtx,
			clampPrompt(prompt),
			session.Request.Workspace,
			func(chunk string) {
				m.recordAgentDelta(session, status, phase, round, chunk)
			},
			onTool,
		)
	} else if stream != nil {
		text, err = stream(runCtx, clampPrompt(prompt), func(chunk string) {
			m.recordAgentDelta(session, status, phase, round, chunk)
		})
	} else {
		text, err = invoke(runCtx, clampPrompt(prompt))
	}
	duration := time.Since(start)
	if err != nil {
		m.recordAgentError(session, status, phase, err)
		return "", duration, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		err = errors.New("模型返回空结果")
		m.recordAgentError(session, status, phase, err)
		return "", duration, err
	}
	text = trimRunes(text, maxAgentOutput)
	m.recordAgentOutput(session, status, phase, round, text, duration)
	return text, duration, nil
}

func (m *Manager) adapter(id string) agentAdapter {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.adapters[id]
}

func (m *Manager) updateRoundPhase(session *Session, phase string, round int, text string) {
	m.mu.Lock()
	session.Phase = phase
	session.CurrentRound = round
	session.UpdatedAt = time.Now().UnixMilli()
	event := Event{Phase: phase, Kind: EventPhase, Text: text, At: time.Now().UnixMilli()}
	event = m.appendEventLocked(session, event)
	_ = m.store.save(session)
	m.mu.Unlock()
	m.emitSafe(event)
}

func (m *Manager) appendContribution(session *Session, contribution Contribution) {
	m.mu.Lock()
	session.Contributions = append(session.Contributions, contribution)
	session.UpdatedAt = time.Now().UnixMilli()
	_ = m.store.save(session)
	m.mu.Unlock()
}

func (m *Manager) recordAgentStarted(session *Session, status AgentStatus, phase string, round int) {
	text := fmt.Sprintf("%s 开始%s", status.Name, phaseLabel(phase))
	if round > 0 {
		text = fmt.Sprintf("%s 开始第 %d 轮发言", status.Name, round)
	}
	m.mu.Lock()
	session.UpdatedAt = time.Now().UnixMilli()
	for i := range session.Participants {
		if session.Participants[i].AgentID == status.ID {
			session.Participants[i].Status = "running"
			session.Participants[i].StartedAt = time.Now().UnixMilli()
		}
	}
	event := Event{Phase: phase, Round: round, AgentID: status.ID, Kind: EventAgentStart, Text: text, At: time.Now().UnixMilli()}
	event = m.appendEventLocked(session, event)
	_ = m.store.save(session)
	m.mu.Unlock()
	m.emitSafe(event)
}

func (m *Manager) recordAgentDelta(session *Session, status AgentStatus, phase string, round int, text string) {
	if text == "" {
		return
	}
	m.emitSafe(Event{
		Seq:       -atomic.AddInt64(&m.deltaSeq, 1),
		SessionID: session.ID,
		Phase:     phase,
		Round:     round,
		AgentID:   status.ID,
		Kind:      EventAgentDelta,
		Text:      text,
		At:        time.Now().UnixMilli(),
	})
}

func (m *Manager) recordToolTrace(
	session *Session,
	status AgentStatus,
	phase string,
	round int,
	trace ToolTrace,
) {
	eventKind := EventToolResult
	if trace.Status == "started" {
		eventKind = EventToolStart
	}
	name := strings.TrimSpace(trace.Name)
	if name == "" {
		name = "工具"
	}
	toolID := strings.TrimSpace(trace.ID)
	if toolID == "" {
		toolID = fmt.Sprintf("%s-%s-%d", status.ID, name, time.Now().UnixNano())
	}
	text := fmt.Sprintf("%s 正在使用 %s", status.Name, name)
	if trace.Status == "completed" {
		text = fmt.Sprintf("%s 已完成 %s", status.Name, name)
	} else if trace.Status == "failed" {
		text = fmt.Sprintf("%s 的 %s 执行失败", status.Name, name)
	}
	m.mu.Lock()
	now := time.Now().UnixMilli()
	session.UpdatedAt = now
	event := Event{
		Phase:      phase,
		Round:      round,
		AgentID:    status.ID,
		Kind:       eventKind,
		Text:       text,
		ToolID:     toolID,
		ToolName:   name,
		ToolStatus: trace.Status,
		ToolInput:  trace.Input,
		ToolOutput: trace.Output,
		Error:      trace.Error,
		At:         now,
	}
	event = m.appendEventLocked(session, event)
	_ = m.store.save(session)
	m.mu.Unlock()
	m.emitSafe(event)
}

func (m *Manager) recordAgentOutput(session *Session, status AgentStatus, phase string, round int, text string, duration time.Duration) {
	m.mu.Lock()
	now := time.Now().UnixMilli()
	session.UpdatedAt = now
	for i := range session.Participants {
		if session.Participants[i].AgentID == status.ID {
			session.Participants[i].Status = "finished"
			session.Participants[i].FinishedAt = now
			session.Participants[i].DurationMs += duration.Milliseconds()
		}
	}
	event := Event{
		Phase: phase, Round: round, AgentID: status.ID, Kind: EventAgentOutput,
		Text: text, At: now, DurationMs: duration.Milliseconds(),
	}
	event = m.appendEventLocked(session, event)
	_ = m.store.save(session)
	m.mu.Unlock()
	m.emitSafe(event)
}

func (m *Manager) recordAgentError(session *Session, status AgentStatus, phase string, err error) {
	m.mu.Lock()
	now := time.Now().UnixMilli()
	session.UpdatedAt = now
	for i := range session.Participants {
		if session.Participants[i].AgentID == status.ID {
			session.Participants[i].Status = "failed"
			session.Participants[i].FinishedAt = now
			session.Participants[i].Error = err.Error()
		}
	}
	event := Event{
		Phase: phase, AgentID: status.ID, Kind: EventAgentError,
		Text: status.Name + " 调用失败", Error: err.Error(), At: now,
	}
	event = m.appendEventLocked(session, event)
	_ = m.store.save(session)
	m.mu.Unlock()
	m.emitSafe(event)
}

func (m *Manager) finishCancelled(session *Session) {
	m.mu.Lock()
	session.Status = StatusCancelled
	session.Phase = PhaseCancelled
	session.Error = "用户终止了本次议事"
	session.UpdatedAt = time.Now().UnixMilli()
	event := Event{Phase: PhaseCancelled, Kind: EventComplete, Text: session.Error, At: session.UpdatedAt}
	event = m.appendEventLocked(session, event)
	_ = m.store.save(session)
	m.mu.Unlock()
	m.emitSafe(event)
}

func (m *Manager) finishFailed(session *Session, reason string) {
	m.mu.Lock()
	session.Status = StatusFailed
	session.Phase = PhaseFailed
	session.Error = reason
	session.UpdatedAt = time.Now().UnixMilli()
	event := Event{Phase: PhaseFailed, Kind: EventError, Error: reason, Text: "议事失败", At: session.UpdatedAt}
	event = m.appendEventLocked(session, event)
	_ = m.store.save(session)
	m.mu.Unlock()
	m.emitSafe(event)
}

func (m *Manager) addWarning(session *Session, warning string) {
	warning = strings.TrimSpace(warning)
	if warning == "" {
		return
	}
	m.mu.Lock()
	session.Warnings = append(session.Warnings, warning)
	session.UpdatedAt = time.Now().UnixMilli()
	event := Event{Phase: session.Phase, Kind: "warning", Text: warning, At: session.UpdatedAt}
	event = m.appendEventLocked(session, event)
	_ = m.store.save(session)
	m.mu.Unlock()
	m.emitSafe(event)
}

func (m *Manager) recordConsensus(session *Session, round int) {
	m.mu.Lock()
	session.UpdatedAt = time.Now().UnixMilli()
	event := Event{
		Phase: session.Phase,
		Round: round,
		Kind:  EventConsensus,
		Text:  fmt.Sprintf("第 %d 轮全员意见一致，提前进入协作汇总", round),
		At:    session.UpdatedAt,
	}
	event = m.appendEventLocked(session, event)
	_ = m.store.save(session)
	m.mu.Unlock()
	m.emitSafe(event)
}

func (m *Manager) emitEvent(session *Session, event Event) {
	m.mu.Lock()
	if event.At == 0 {
		event.At = time.Now().UnixMilli()
	}
	event = m.appendEventLocked(session, event)
	_ = m.store.save(session)
	m.mu.Unlock()
	m.emitSafe(event)
}

func (m *Manager) appendEventLocked(session *Session, event Event) Event {
	event.SessionID = session.ID
	event.Seq = int64(len(session.Events) + 1)
	if event.At == 0 {
		event.At = time.Now().UnixMilli()
	}
	session.Events = append(session.Events, event)
	return event
}

func (m *Manager) lastEventsLocked(session *Session, count int) []Event {
	if count <= 0 || len(session.Events) == 0 {
		return nil
	}
	start := len(session.Events) - count
	if start < 0 {
		start = 0
	}
	return session.Events[start:]
}

func (m *Manager) emitSafe(event Event) {
	if m.emit == nil {
		return
	}
	defer func() { _ = recover() }()
	m.emit(event)
}

func parseVerdict(raw, chairman string, _ time.Duration) *Verdict {
	trimmed := strings.TrimSpace(raw)
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end > start {
		var parsed struct {
			Summary        string   `json:"summary"`
			Recommendation string   `json:"recommendation"`
			Conditions     []string `json:"conditions"`
			Disagreements  []string `json:"disagreements"`
			Risks          []string `json:"risks"`
			EvidenceNotes  []string `json:"evidenceNotes"`
		}
		if json.Unmarshal([]byte(trimmed[start:end+1]), &parsed) == nil {
			if strings.TrimSpace(parsed.Recommendation) != "" || strings.TrimSpace(parsed.Summary) != "" {
				return &Verdict{
					Summary: parsed.Summary, Recommendation: parsed.Recommendation,
					Conditions:    cleanStrings(parsed.Conditions),
					Disagreements: cleanStrings(parsed.Disagreements),
					Risks:         cleanStrings(parsed.Risks),
					EvidenceNotes: cleanStrings(parsed.EvidenceNotes),
					Chairman:      chairman, CreatedAt: time.Now().UnixMilli(),
				}
			}
		}
	}
	recommendation := firstParagraph(trimmed)
	return &Verdict{
		Summary: trimmed, Recommendation: recommendation,
		Chairman: chairman, Partial: true, CreatedAt: time.Now().UnixMilli(),
	}
}

func extractConsensusSignal(text string) (string, string) {
	lines := strings.Split(text, "\n")
	clean := make([]string, 0, len(lines))
	signal := ""
	for _, line := range lines {
		match := consensusMarker.FindStringSubmatch(line)
		if len(match) > 1 {
			switch strings.ToLower(strings.TrimSpace(match[1])) {
			case "一致", "同意", "yes":
				signal = "yes"
			case "仍有分歧", "不同意", "no":
				signal = "no"
			}
			continue
		}
		clean = append(clean, line)
	}
	return strings.TrimSpace(strings.Join(clean, "\n")), signal
}

func roundReachedConsensus(contributions []Contribution, expected int) bool {
	if expected < 2 || len(contributions) != expected {
		return false
	}
	seen := make(map[string]bool, expected)
	for _, contribution := range contributions {
		if contribution.Consensus != "yes" || seen[contribution.AgentID] {
			return false
		}
		seen[contribution.AgentID] = true
	}
	return true
}

func fallbackVerdict(history []Contribution) *Verdict {
	return &Verdict{
		Summary:        "未能形成结构化讨论小结。以下保留各轮原始发言，不应视为已取得共识。",
		Recommendation: "暂未形成自动综合建议，请结合各方发言人工整理。",
		Disagreements:  []string{fmt.Sprintf("保留了 %d 条讨论发言", len(history))},
		Chairman:       "未取得",
		Partial:        true,
		CreatedAt:      time.Now().UnixMilli(),
	}
}

func phaseLabel(phase string) string {
	switch phase {
	case PhaseOpening:
		return "开场陈述"
	case PhaseDiscussion:
		return "补充论证"
	case PhaseSynthesis:
		return "协作汇总"
	case PhaseIndependent:
		return "开场陈述"
	case PhaseReview:
		return "互相澄清"
	case PhaseChallenge:
		return "补充论证"
	case PhaseVerdict:
		return "协作汇总"
	default:
		return phase
	}
}

func cleanStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func firstParagraph(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.Index(s, "\n\n"); idx > 0 {
		s = s[:idx]
	}
	return trimRunes(s, 1200)
}

func titleFromTopic(topic string) string {
	line := strings.TrimSpace(strings.Split(topic, "\n")[0])
	return trimRunes(line, 48)
}

func newSessionID(now int64) string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("pan_%d", now)
	}
	return fmt.Sprintf("pan_%d_%s", now, hex.EncodeToString(buf))
}

func uniqueNonEmpty(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, id := range in {
		id = strings.ToLower(strings.TrimSpace(id))
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func orderedAgentIDs(requested []string, available map[string]AgentStatus) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(available))
	for _, id := range requested {
		if _, ok := available[id]; !ok || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	remaining := make([]string, 0, len(available)-len(out))
	for id := range available {
		if !seen[id] {
			remaining = append(remaining, id)
		}
	}
	sort.Strings(remaining)
	return append(out, remaining...)
}

func rotateAgentIDs(order []string, offset int) []string {
	if len(order) == 0 {
		return nil
	}
	offset %= len(order)
	out := make([]string, 0, len(order))
	out = append(out, order[offset:]...)
	out = append(out, order[:offset]...)
	return out
}

// pickChairman 默认汇总人：从参会席位中选择，Grok 在席且可用时优先，
// 仅用于整理共识与开放问题，不赋予最终裁决权。
func pickChairman(selectedIDs []string, available map[string]AgentStatus) string {
	if _, ok := available["grok"]; ok {
		for _, id := range selectedIDs {
			if id == "grok" {
				return "grok"
			}
		}
	}
	for _, id := range selectedIDs {
		if _, ok := available[id]; ok {
			return id
		}
	}
	return selectedIDs[0]
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneSession(in *Session) *Session {
	if in == nil {
		return nil
	}
	data, err := json.Marshal(in)
	if err != nil {
		return in
	}
	var out Session
	if json.Unmarshal(data, &out) == nil {
		return &out
	}
	return in
}

func cloneEvents(in []Event) []Event {
	out := make([]Event, len(in))
	copy(out, in)
	return out
}
