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
	maxAttachments         = 4
	maxAttachmentBytes     = 20 << 20
)

var consensusMarker = regexp.MustCompile(`(?im)^\s*(?:[-*>]\s*)*(?:共识状态|consensus)\s*[:：]\s*(一致|同意|仍有分歧|不同意|yes|no)`)

var (
	errAgentSkipped = errors.New("agent skipped")
	errAgentRetry   = errors.New("agent retry requested")
)

var attachmentMIME = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
	".gif":  "image/gif",
	".bmp":  "image/bmp",
}

// NormalizeAttachments validates the small, image-only attachment surface used
// by council discussions. It is safe to call more than once.
func NormalizeAttachments(items []Attachment) ([]Attachment, error) {
	if len(items) > maxAttachments {
		return nil, fmt.Errorf("最多添加 %d 张图片", maxAttachments)
	}
	out := make([]Attachment, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		path := strings.TrimSpace(item.Path)
		if path == "" {
			return nil, errors.New("图片附件路径不能为空")
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("解析图片路径失败: %w", err)
		}
		abs = filepath.Clean(abs)
		key := abs
		if filepath.Separator == '\\' {
			key = strings.ToLower(key)
		}
		if seen[key] {
			continue
		}
		info, err := os.Stat(abs)
		if err != nil {
			return nil, fmt.Errorf("图片不可访问 %s: %w", filepath.Base(abs), err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("图片附件不是普通文件: %s", filepath.Base(abs))
		}
		if info.Size() <= 0 {
			return nil, fmt.Errorf("图片为空: %s", filepath.Base(abs))
		}
		if info.Size() > maxAttachmentBytes {
			return nil, fmt.Errorf("图片 %s 超过 %dMB", filepath.Base(abs), maxAttachmentBytes>>20)
		}
		mime := attachmentMIME[strings.ToLower(filepath.Ext(abs))]
		if mime == "" {
			return nil, fmt.Errorf("仅支持 PNG/JPG/WEBP/GIF/BMP 图片: %s", filepath.Base(abs))
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = filepath.Base(abs)
		}
		seen[key] = true
		out = append(out, Attachment{Name: name, Path: abs, MIME: mime, Size: info.Size()})
	}
	return out, nil
}

func attachmentPaths(items []Attachment) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Path) != "" {
			out = append(out, item.Path)
		}
	}
	return out
}

func supportsImageAttachment(status AgentStatus) bool {
	for _, capability := range status.Capabilities {
		switch strings.ToLower(strings.TrimSpace(capability)) {
		case "image-input", "image-path-tools", "shared-image-summary":
			return true
		}
	}
	return false
}

type Manager struct {
	root    string
	workDir string

	mu        sync.Mutex
	store     *sessionStore
	runner    commandRunner
	overrides map[string]string
	dshKey    string
	adapters  map[string]agentAdapter
	grok      GrokBackend
	cancels   map[string]context.CancelFunc
	// agentCalls 会话×席位双键槽：独立开场第 1 轮并行时多席同时持有活动调用，
	// skip/retry/中止判定按 (sessionID, agentID) 定位（2026-11-03 并行刀）。
	agentCalls map[string]map[string]*activeAgentCall
	runs       map[string]*Session
	wg         sync.WaitGroup
	emit       func(Event)
	deltaSeq   int64
	// authorityID 每次进程启动重生成：席位观测的序号只在同一 authority 内可比，
	// 换进程后必须从头计数（§2.2）。
	authorityID string
}

type activeAgentCall struct {
	sessionID string
	CallRef
	agentID string
	cancel  context.CancelFunc
	skipped bool
	retry   bool
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
		root:        root,
		workDir:     root,
		store:       store,
		runner:      runner,
		overrides:   overrides,
		adapters:    buildAdapters(overrides, nil, ""),
		cancels:     map[string]context.CancelFunc{},
		agentCalls:  map[string]map[string]*activeAgentCall{},
		runs:        map[string]*Session{},
		emit:        emit,
		authorityID: fmt.Sprintf("pantheon:%d:%d", os.Getpid(), time.Now().UnixNano()),
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
			// waiting_input 是"停着等人"，不是"被中断"：重启不替人解开未答的
			// 问题或没处理的决策门（M14），所以它原样保持阻塞，等显式处理。
			continue
		}
		session, err := m.store.get(summary.ID)
		if err != nil {
			return recovered, err
		}
		now := time.Now().UnixMilli()
		reason := "上次运行被应用中断，已保留此前产生的发言与事件"
		// 重启不是失败：没有人说过这场议事输了，只是没人看见收声（O-D / M3）。
		// 于是会话落 interrupted、在跑的席位落 finished_unverified，
		// 而不是伪造一个 failed 让用户以为模型给了否定结论。
		session.Status = StatusInterrupted
		session.Error = reason
		session.UpdatedAt = now
		for i := range session.Participants {
			p := &session.Participants[i]
			if p.Status != SeatRunning {
				continue
			}
			if err := m.observeSeatLocked(session, p.AgentID, seatObservation{
				Origin:   OriginRecovery,
				Status:   SeatFinishedUnverified,
				Evidence: "应用重启，未观测到收声",
				At:       now,
			}); err != nil {
				session.Warnings = append(session.Warnings, fmt.Sprintf("%s 状态未能更新：%s", p.Name, err.Error()))
			}
			p.FinishedAt = now
			p.Error = "应用重启，本次调用被中断"
		}
		session.Warnings = append(session.Warnings, reason)
		m.appendEventLocked(session, Event{
			Phase: session.Phase,
			Kind:  EventError,
			Text:  "议事因应用重启而中断：未观测到收声的席位标为未验证，可从断点续跑",
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
	m.adapters = buildAdapters(m.overrides, backend, m.dshKey)
}

// SetDSHAPIKey 注入 DeepSeek 凭据给 dsh 席位（DEEPSEEK_API_KEY 走 DSH 官方
// 环境变量通道；其凭据服务无存档时陈词会 MISSING_CREDENTIAL，实弹逮住）。
func (m *Manager) SetDSHAPIKey(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dshKey = key
	m.adapters = buildAdapters(m.overrides, m.grok, key)
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
	m.adapters = buildAdapters(overrides, m.grok, m.dshKey)
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
	attachments, err := NormalizeAttachments(req.Attachments)
	if err != nil {
		return nil, err
	}
	req.Attachments = attachments
	protocol, err := NormalizeProtocol(req.Protocol)
	if err != nil {
		return nil, err
	}
	req.Protocol = protocol
	req.ParentSessionID = strings.TrimSpace(req.ParentSessionID)
	normalizeRoundSettings(&req)
	// 材料快照先于历史论点注入：快照只钉用户设题材料，续议立场卡是派生
	// 内容（全文已随父会话落盘），不入 manifest。
	inputManifest := buildInputManifest(req)
	if req.ParentSessionID != "" {
		parent, err := m.store.get(req.ParentSessionID)
		if err != nil {
			return nil, fmt.Errorf("续议的父会话不可用：%w", err)
		}
		req.Evidence = append(req.Evidence, parentStanceEvidence(parent)...)
	}
	// ReadOnly 是会话级声明性标记（无逻辑消费，真实机制见 agentAdapter.runMode
	// 逐席入档）。2026-11-03 起 codex 单席沙箱放宽 workspace-write+network
	// （用户拍板全员联网查资料），写面由陈词纪律约束不写盘——本标记不再等于
	// 「全部席位硬只读」，对外展示以 runMode 为准。
	req.ReadOnly = true
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
	// 圆桌接力仍要求至少两席；独立开场允许单席深答（对照基线之一，
	// 裁决第二刀：单 Agent 路径此前既跑不出来也无法打标签）。
	if len(selectedIDs) < 2 && req.Protocol != ProtocolIndependent {
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
	if len(available) < 2 && req.Protocol != ProtocolIndependent {
		return nil, errors.New("可用 Agent 少于两个，无法开始讨论")
	}
	if len(attachments) > 0 {
		for _, id := range selectedIDs {
			status := byID[id]
			if status.Available && !supportsImageAttachment(status) {
				return nil, fmt.Errorf("%s 不支持图片附件，请移除图片或取消该席", status.Name)
			}
		}
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
		ID:            newSessionID(now),
		Title:         req.Title,
		Status:        StatusRunning,
		Phase:         PhasePreparing,
		CreatedAt:     now,
		UpdatedAt:     now,
		Request:       req,
		Participants:  participants,
		Events:        []Event{},
		Contributions: []Contribution{},
		InputManifest: inputManifest,
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
	m.runs[session.ID] = session
	m.wg.Add(1)
	m.mu.Unlock()
	go func() {
		defer m.wg.Done()
		defer func() {
			m.mu.Lock()
			delete(m.cancels, session.ID)
			delete(m.runs, session.ID)
			m.mu.Unlock()
			cancel()
		}()
		m.run(runCtx, session, available, chairman, 1, nil)
	}()
	return cloneSession(session), nil
}

// Deliberation path labels (2026-10-31 裁决第二刀)：对照实验可表达——
// 每场议事必须记录自己走的是哪条路径，比较才有对象。
const (
	// ProtocolCouncil 圆桌接力：同轮前序发言对后位可见（原始协议，可显式选用）。
	ProtocolCouncil = "council"
	// ProtocolIndependent 独立开场（2026-10-31 用户拍板升为默认）：首轮各抒己见，
	// 不注入同轮前序，避免首位发言框定范围；第 2 轮起共享前轮信息开始议论。
	// 配合单轮即得「独立多答」基线。用户指定顺序与首位轮换保留。
	ProtocolIndependent = "independent"
)

func NormalizeProtocol(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		// 默认独立开场：首位锚定是实弹观察到的真问题，新会话默认各抒己见。
		// 历史存档无 protocol 字段，展示层按圆桌接力渲染，不篡改史实。
		return ProtocolIndependent, nil
	}
	switch trimmed {
	case ProtocolCouncil, ProtocolIndependent:
		return trimmed, nil
	}
	return "", fmt.Errorf("未知议事路径 %q", trimmed)
}

func ProtocolLabel(protocol string) string {
	if protocol == ProtocolIndependent {
		return "独立开场"
	}
	return "圆桌接力"
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
	// 单轮放行（裁决第二刀：允许单轮）；0 仍取默认两轮。
	if req.Rounds < 1 {
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

func (m *Manager) beginAgentCall(
	ctx context.Context,
	session *Session,
	status AgentStatus,
	phase string,
	round int,
) (context.Context, *activeAgentCall) {
	callCtx, cancel := context.WithCancel(ctx)
	callID := seatCallID(session.ID, status.ID, phase, round)
	attempt := 1
	retryOf := ""
	m.mu.Lock()
	for _, event := range session.Events {
		if event.CallID != callID {
			continue
		}
		if event.Kind == EventAgentStart && event.AttemptID != "" {
			attempt++
		}
		if event.Kind == EventAgentRetry && event.AttemptID != "" {
			retryOf = event.AttemptID
		}
	}
	m.mu.Unlock()
	call := &activeAgentCall{
		sessionID: session.ID,
		CallRef: CallRef{
			CallID:         callID,
			AttemptID:      newAttemptID(time.Now().UnixMilli()),
			RetryOfAttempt: retryOf,
			Attempt:        attempt,
		},
		agentID: status.ID,
		cancel:  cancel,
	}
	m.recordAgentStarted(session, status, phase, round, call)
	return callCtx, call
}

func (m *Manager) endAgentCall(sessionID string, call *activeAgentCall) {
	if call == nil {
		return
	}
	m.mu.Lock()
	if slot := m.agentCalls[call.sessionID]; slot[call.agentID] == call {
		delete(slot, call.agentID)
		if len(slot) == 0 {
			delete(m.agentCalls, call.sessionID)
		}
	}
	m.mu.Unlock()
	call.cancel()
}

func (m *Manager) agentCallSkipped(call *activeAgentCall) bool {
	if call == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return call.skipped
}

func (m *Manager) agentCallRetry(call *activeAgentCall) bool {
	if call == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return call.retry
}

// agentCallAborted 报告该次调用是否已被跳过、要求重试或不再是当前 Attempt。
// 取消只杀进程，管道里已缓冲的输出仍会继续流入；这些残影必须在发射前丢弃。
func (m *Manager) agentCallAborted(call *activeAgentCall) bool {
	if call == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return call.skipped || call.retry || m.agentCalls[call.sessionID][call.agentID] != call
}

// SkipAgent cancels only the active model/CLI call. The council loop continues
// with the next participant, preserving all prior contributions.
func (m *Manager) SkipAgent(id, agentID string) error {
	id = strings.TrimSpace(id)
	agentID = strings.ToLower(strings.TrimSpace(agentID))
	m.mu.Lock()
	// 并行开场下多席同时活动（2026-11-03）：按席位定位槽，不再假设全场单调用。
	call := m.agentCalls[id][agentID]
	if call == nil {
		m.mu.Unlock()
		return fmt.Errorf("%s 当前没有可跳过的调用", agentID)
	}
	if call.skipped {
		m.mu.Unlock()
		return nil
	}
	session := m.runs[id]
	if session == nil {
		m.mu.Unlock()
		call.cancel()
		return errors.New("议事运行状态不可用")
	}
	if session.Status != StatusRunning {
		m.mu.Unlock()
		return errors.New("议事已不在运行")
	}
	call.skipped = true
	now := time.Now().UnixMilli()
	name := agentID
	for i := range session.Participants {
		participant := &session.Participants[i]
		if participant.AgentID != agentID {
			continue
		}
		name = participant.Name
		if err := m.observeSeatLocked(session, agentID, seatObservation{
			Origin:   OriginUser,
			Status:   SeatSkipped,
			Evidence: "用户跳过当前调用",
			CallRef:  call.CallRef,
			At:       now,
		}); err != nil {
			session.Warnings = append(session.Warnings, fmt.Sprintf("%s 无法标记为已跳过：%s", name, err.Error()))
			continue
		}
		participant.FinishedAt = now
		if participant.StartedAt > 0 && now > participant.StartedAt {
			participant.DurationMs += now - participant.StartedAt
		}
		participant.Error = "用户跳过当前调用"
	}
	session.UpdatedAt = now
	event := m.appendEventLocked(session, Event{
		CallRef: call.CallRef,
		Phase:   session.Phase,
		Round:   session.CurrentRound,
		AgentID: agentID,
		Kind:    EventAgentSkipped,
		Text:    name + " 当前调用已跳过，继续下一位 Agent",
		At:      now,
	})
	saveErr := m.store.save(session)
	m.mu.Unlock()
	call.cancel()
	m.emitSafe(event)
	return saveErr
}

// RetryAgent aborts only the current call and lets the council loop invoke the
// same participant again with the existing history and round position.
func (m *Manager) RetryAgent(id, agentID string) error {
	id = strings.TrimSpace(id)
	agentID = strings.ToLower(strings.TrimSpace(agentID))
	m.mu.Lock()
	call := m.agentCalls[id][agentID]
	if call == nil {
		m.mu.Unlock()
		return fmt.Errorf("%s 当前没有可重试的调用", agentID)
	}
	if call.skipped {
		m.mu.Unlock()
		return errors.New("当前 Agent 已跳过")
	}
	if call.retry {
		m.mu.Unlock()
		return nil
	}
	session := m.runs[id]
	if session == nil {
		m.mu.Unlock()
		call.cancel()
		return errors.New("议事运行状态不可用")
	}
	if session.Status != StatusRunning {
		m.mu.Unlock()
		return errors.New("议事已不在运行")
	}
	// 重试要自己声明运行方式（M17）：沿用上一次等于假设它没变，可它可能已经换了
	// 只读机制。声明不出来就拒绝，让人看见，而不是拿旧口径悄悄再跑一遍。
	runMode := strings.TrimSpace(m.adapters[agentID].runMode)
	if runMode == "" {
		m.mu.Unlock()
		return fmt.Errorf("%s 没有声明运行方式，重试不能沿用上一次的口径", agentID)
	}
	call.retry = true
	now := time.Now().UnixMilli()
	name := agentID
	for i := range session.Participants {
		participant := &session.Participants[i]
		if participant.AgentID != agentID {
			continue
		}
		name = participant.Name
		// 重试给席位发一个新的 attempt，旧 attempt 的终态不参与转移判定。
		m.rebindSeatLocked(session, agentID, SeatRetrying, "用户重试当前调用，保留此前发言")
		participant.FinishedAt = 0
		participant.Error = ""
	}
	placement := "运行方式：" + runMode
	if workspace := strings.TrimSpace(session.Request.Workspace); workspace != "" {
		placement = fmt.Sprintf("运行方式：%s；工作空间：%s", runMode, workspace)
	}
	session.UpdatedAt = now
	event := m.appendEventLocked(session, Event{
		CallRef: call.CallRef,
		Phase:   session.Phase,
		Round:   session.CurrentRound,
		AgentID: agentID,
		Kind:    EventAgentRetry,
		Text:    name + " 正在重新调用，保留此前发言（" + placement + "）",
		At:      now,
	})
	saveErr := m.store.save(session)
	m.mu.Unlock()
	call.cancel()
	m.emitSafe(event)
	return saveErr
}

// Restart starts a fresh run from an existing council brief. A live run is
// cancelled first, but Restart does not wait for it to unwind: the UI needs a
// recovery path even when an external harness ignores cancellation until its
// own hard timeout. The old record remains available for audit.
func (m *Manager) Restart(id string) (*Session, error) {
	m.mu.Lock()
	session, err := m.store.get(id)
	cancel := m.cancels[id]
	m.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if cancel != nil {
		cancel()
	}
	req := session.Request
	req.AgentIDs = append([]string(nil), req.AgentIDs...)
	req.Evidence = append([]EvidenceItem(nil), req.Evidence...)
	req.Attachments = append([]Attachment(nil), req.Attachments...)
	return m.Start(req)
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

// ResultPath returns the chair's final Markdown deliverable, generating it on
// demand for completed sessions created before automatic result documents.
func (m *Manager) ResultPath(id string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, err := m.store.get(id)
	if err != nil {
		return "", err
	}
	if session.Status != StatusCompleted {
		return "", errors.New("讨论尚未完成，暂无最终结果文档")
	}
	if session.ResultPath != "" {
		if info, err := os.Stat(session.ResultPath); err == nil && info.Mode().IsRegular() {
			return session.ResultPath, nil
		}
	}
	path, err := m.store.writeResult(session)
	if err != nil {
		return "", err
	}
	session.ResultPath = path
	if err := m.store.save(session); err != nil {
		return "", err
	}
	return path, nil
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
	session.SignOff = &SignOff{
		Decision: decision,
		Note:     strings.TrimSpace(note),
		At:       now,
		// 缺口也进哈希：同样的小结，三席齐全与两席缺一不该签出同一个指纹
		// （M15 把"谁没交出结论"算作结案事实的一部分）。
		VerdictSHA256: signoffFingerprint(session.Verdict, session.Gaps),
		Gaps:          append([]string(nil), session.Gaps...),
	}
	session.UpdatedAt = now
	label := map[string]string{
		"accepted": "确认讨论小结", "needs_revision": "标记为需继续讨论", "rejected": "暂不采纳讨论小结",
	}[decision]
	event := m.appendEventLocked(session, Event{
		Phase: session.Phase, Kind: EventSignOff, Text: label, At: now,
	})
	// 重渲染对外交付物：签注后 results 文档必须携带用户确认与其绑定的小结
	// 版本，否则签注（session.json）与转发件是两份真相（2026-10-31 裁决第三刀）。
	if session.Status == StatusCompleted {
		if path, err := m.store.writeResult(session); err == nil {
			session.ResultPath = path
		} else {
			session.Warnings = append(session.Warnings, "最终结果文档重渲染失败："+err.Error())
		}
	}
	if err := m.store.save(session); err != nil {
		m.mu.Unlock()
		return nil, err
	}
	out := cloneSession(session)
	m.mu.Unlock()
	m.emitSafe(event)
	return out, nil
}

// Resume 继续一场被中断（失败/终止）的议事：已有贡献原样保留，从中断处的
// 下一轮接着跑，最后照常汇总。中途保存随时可续（2026-10-31 实弹反馈）；
// 崩溃后启动恢复把 running 标为 failed，本方法就是它的「恢复讨论」入口。
func (m *Manager) Resume(id string) (*Session, error) {
	m.mu.Lock()
	session, err := m.store.get(id)
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	if session.Status == StatusRunning {
		m.mu.Unlock()
		return nil, errors.New("讨论仍在进行，无需恢复")
	}
	if session.Status == StatusCompleted {
		m.mu.Unlock()
		return nil, errors.New("讨论已完成；续议请使用「基于小结续议」")
	}
	if len(session.Contributions) == 0 {
		m.mu.Unlock()
		return nil, errors.New("该场没有可续的历史发言；请使用「重开整场」")
	}
	// 未答的问题和没处理的决策门不是"中断"，是"在等人"：续跑不会替人解开，
	// 直接拒绝比让 run 立刻又停回去更诚实（M14）。
	if pending := pendingQuestionsLocked(session); len(pending) > 0 {
		m.mu.Unlock()
		return nil, fmt.Errorf("还有 %d 个问题没有回答，请先回答再继续议事", len(pending))
	}
	if hasOpenGateLocked(session) {
		m.mu.Unlock()
		return nil, errors.New("有一道决策门没被处理，请先放行或放弃")
	}
	request := session.Request
	resumeRound := session.CurrentRound + 1
	history := contributionsForResume(session)
	m.mu.Unlock()

	probeCtx, cancelProbe := context.WithTimeout(context.Background(), 15*time.Second)
	statuses := m.Agents(probeCtx)
	cancelProbe()
	byID := map[string]AgentStatus{}
	for _, status := range statuses {
		byID[status.ID] = status
	}
	available := map[string]AgentStatus{}
	for _, agentID := range uniqueNonEmpty(request.AgentIDs) {
		if status, ok := byID[agentID]; ok && status.Available {
			available[agentID] = status
		}
	}
	if len(available) == 0 {
		return nil, errors.New("没有可用 Agent，无法恢复讨论")
	}
	chairman := strings.TrimSpace(request.ChairmanAgentID)
	if _, ok := available[chairman]; !ok {
		chairman = pickChairman(request.AgentIDs, available)
	}

	m.mu.Lock()
	session, err = m.store.get(id)
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	if session.Status == StatusRunning {
		m.mu.Unlock()
		return nil, errors.New("讨论已在进行中")
	}
	if resumeRound < 1 {
		resumeRound = 1
	}
	// 重开是新的 run，不是会话状态转移：终态会话允许被用户续跑，
	// 所以这里不走 sessionTransitions，而按"新建 attempt"处理（M12 具名例外）。
	session.Status = StatusRunning
	session.Phase = PhasePreparing
	session.Error = ""
	session.UpdatedAt = time.Now().UnixMilli()
	for i := range session.Participants {
		p := &session.Participants[i]
		switch p.Status {
		case SeatFailed, SeatFinishedUnverified, SeatSkipped, SeatCancelled, SeatUnavailable, SeatNeedsInput:
			// 旧 attempt 的终态不拦住新 attempt 的起步：抬 incarnation 后回到待发言。
			m.rebindSeatLocked(session, p.AgentID, SeatWaiting, "整场重开，席位回到待发言")
			p.Error = ""
			p.FinishedAt = 0
		}
	}
	event := m.appendEventLocked(session, Event{
		Phase: PhasePreparing,
		Kind:  EventPhase,
		Text: fmt.Sprintf(
			"从中断处恢复：保留 %d 条已有发言，从第 %d 轮继续",
			len(history), resumeRound,
		),
		At: session.UpdatedAt,
	})
	if err := m.store.save(session); err != nil {
		m.mu.Unlock()
		return nil, err
	}
	runCtx, cancel := context.WithCancel(context.Background())
	m.cancels[session.ID] = cancel
	m.runs[session.ID] = session
	m.wg.Add(1)
	m.mu.Unlock()
	go func() {
		defer m.wg.Done()
		defer func() {
			m.mu.Lock()
			delete(m.cancels, session.ID)
			delete(m.runs, session.ID)
			m.mu.Unlock()
			cancel()
		}()
		m.run(runCtx, session, available, chairman, resumeRound, history)
	}()
	out := cloneSession(session)
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

// run 执行议事循环。startRound/history 支持中断恢复：恢复时保留既有贡献、
// 从中断处的下一轮继续（2026-10-31 实弹反馈「中途保存、临时变故的保存」）。
func (m *Manager) run(
	ctx context.Context,
	session *Session,
	available map[string]AgentStatus,
	chairman string,
	startRound int,
	history []Contribution,
) {
	order := orderedAgentIDs(session.Request.AgentIDs, available)
	if history == nil {
		history = make([]Contribution, 0, len(order)*session.Request.Rounds)
	}
	successful := map[string]AgentStatus{}
	for _, item := range history {
		if status, ok := available[item.AgentID]; ok {
			successful[item.AgentID] = status
		}
	}
	actualRounds := startRound - 1
	if actualRounds < 0 {
		actualRounds = 0
	}
	consensusRound := 0
	clippedInputs := make([]string, 0)

	for round := startRound; round <= session.Request.Rounds; round++ {
		// 停车点一：轮次之间。门没解、问题没答之前不进新一轮，宁可停着，
		// 也不让界面看起来还在推进（Orca 把这条叫"别伪造活性"）。
		if m.paused(session) {
			m.notePaused(session)
			return
		}
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
		// 独立开场第 1 轮并行（2026-11-03 用户拍板）：每位视同首位发言、输入互
		// 不依赖（openingHistory=nil），串行纯属等待不存在的依赖；并行=最慢一席
		// 的时间，单场省 40%~70%。圆桌接力第 1 轮与所有协议的第 2 轮起保持串行
		// ——后位要看前位发言，接力与吸收是轮内语义。
		parallelOpening := round == 1 && session.Request.Protocol == ProtocolIndependent && len(roundOrder) > 1

		// runSeat 执行一席并入库：返回 (贡献, prompt 是否超限, 是否成功)。
		// 入库（appendContribution）自带锁，并行下安全；clipped 记账由调用方
		// 单线程收集，避免闭包写共享切片。
		runSeat := func(agentID string, position int) (Contribution, bool, bool) {
			status := available[agentID]
			files := m.transcriptFor(session)
			var prompt string
			if round == 1 {
				openingHistory := history
				if session.Request.Protocol == ProtocolIndependent {
					// 独立开场基线：第 1 轮不注入同轮前序，每位视同首位发言。
					openingHistory = nil
				}
				prompt = openingPrompt(session.Request, status, openingHistory, position+1, len(roundOrder), files)
			} else {
				prompt = discussionPrompt(session.Request, status, history, round, position+1, len(roundOrder), files)
			}
			clipped := promptExceedsBudget(prompt)
			out, duration, callRef, err := m.runAgentWithRetry(ctx, session, status, prompt, phase, round)
			if err != nil {
				return Contribution{}, clipped, false
			}
			consensus := ""
			if session.Request.UntilConsensus {
				out, consensus = extractConsensusSignal(out)
			}
			contribution := Contribution{
				CallRef:    callRef,
				ID:         fmt.Sprintf("%s-%s-r%d-%d", session.ID, status.ID, round, time.Now().UnixNano()),
				Phase:      phase,
				Round:      round,
				AgentID:    status.ID,
				AgentName:  status.Name,
				Content:    out,
				Consensus:  consensus,
				RunMode:    m.adapter(status.ID).runMode,
				CreatedAt:  time.Now().UnixMilli(),
				DurationMs: duration.Milliseconds(),
			}
			m.appendContribution(session, contribution)
			return contribution, clipped, true
		}

		if parallelOpening {
			if ctx.Err() != nil {
				m.finishCancelled(session)
				return
			}
			if m.paused(session) {
				m.notePaused(session)
				return
			}
			type seatOutcome struct {
				contribution Contribution
				clipped      bool
				ok           bool
			}
			outcomes := make([]seatOutcome, len(roundOrder))
			var openingWG sync.WaitGroup
			for position, agentID := range roundOrder {
				if ctx.Err() != nil {
					break
				}
				openingWG.Add(1)
				go func(position int, seatID string) {
					defer openingWG.Done()
					contribution, clipped, ok := runSeat(seatID, position)
					outcomes[position] = seatOutcome{contribution, clipped, ok}
				}(position, agentID)
			}
			openingWG.Wait()
			for position, outcome := range outcomes {
				if outcome.clipped {
					clippedInputs = append(
						clippedInputs,
						fmt.Sprintf("第%d轮·%s", round, roundOrder[position]),
					)
				}
				if !outcome.ok {
					continue
				}
				history = append(history, outcome.contribution)
				roundContributions = append(roundContributions, outcome.contribution)
				successful[roundOrder[position]] = available[roundOrder[position]]
			}
		} else {
			for position, agentID := range roundOrder {
				if ctx.Err() != nil {
					m.finishCancelled(session)
					return
				}
				// 停车点二：席位之间。已经在跑的那一席不打断（外部进程没法拦腰
				// 插话），但下一个席位绝不启动。
				if m.paused(session) {
					m.notePaused(session)
					return
				}
				contribution, clipped, ok := runSeat(agentID, position)
				if clipped {
					clippedInputs = append(
						clippedInputs,
						fmt.Sprintf("第%d轮·%s", round, agentID),
					)
				}
				if !ok {
					continue
				}
				history = append(history, contribution)
				roundContributions = append(roundContributions, contribution)
				successful[agentID] = available[agentID]
			}
		}
		if session.Request.UntilConsensus &&
			round >= 2 &&
			roundReachedConsensus(roundContributions, len(order)) {
			consensusRound = round
			m.recordConsensus(session, round)
			break
		}
		// 用户要求每轮之间停一次：本轮已收声，就在此处开门，等人放行。
		if session.Request.GateEachRound && round < session.Request.Rounds {
			if m.openGateInSession(session, fmt.Sprintf("第 %d 轮结束，按设置停下等你确认", round)) {
				return
			}
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
		m.addWarning(session, consensusShortfallWarning(
			session.Request.Rounds, order, history, actualRounds, clippedInputs,
		))
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
		out, duration, callRef, err := m.runAgentWithRetry(
			ctx,
			session,
			summarizerStatus,
			synthesisPrompt(session.Request, summarizerStatus, history, m.transcriptFor(session)),
			PhaseSynthesis,
			0,
		)
		if err == nil && strings.TrimSpace(out) != "" {
			verdict = parseVerdict(out, summarizerStatus.Name, duration)
			m.appendContribution(session, Contribution{
				CallRef:    callRef,
				ID:         fmt.Sprintf("%s-%s-synthesis-%d", session.ID, summarizerStatus.ID, time.Now().UnixNano()),
				Phase:      PhaseSynthesis,
				AgentID:    summarizerStatus.ID,
				AgentName:  summarizerStatus.Name,
				AnonLabel:  "协作汇总",
				Content:    out,
				RunMode:    m.adapter(summarizerStatus.ID).runMode,
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
	session.Gaps = seatGaps(session)
	session.Phase = PhaseCompleted
	session.Status = StatusCompleted
	session.UpdatedAt = time.Now().UnixMilli()
	m.appendEventLocked(session, Event{
		Phase: PhaseCompleted, Kind: EventVerdict, Text: verdict.Recommendation,
		At: time.Now().UnixMilli(),
	})
	completeText := "讨论小结已形成，等待用户确认"
	if len(session.Gaps) > 0 {
		completeText = fmt.Sprintf("讨论小结已形成，但有 %d 席没交出可用结论：%s",
			len(session.Gaps), strings.Join(session.Gaps, "、"))
	}
	m.appendEventLocked(session, Event{
		Phase: PhaseCompleted, Kind: EventComplete, Text: completeText,
		At: time.Now().UnixMilli(),
	})
	if path, err := m.store.writeResult(session); err == nil {
		session.ResultPath = path
	} else {
		session.Warnings = append(session.Warnings, "最终结果文档生成失败："+err.Error())
	}
	_ = m.store.save(session)
	events := cloneEvents(m.lastEventsLocked(session, 2))
	m.mu.Unlock()
	for _, event := range events {
		m.emitSafe(event)
	}
}

func (m *Manager) runAgentWithRetry(
	ctx context.Context,
	session *Session,
	status AgentStatus,
	prompt string,
	phase string,
	round int,
) (string, time.Duration, CallRef, error) {
	for {
		text, duration, callRef, err := m.runOne(ctx, session, status, prompt, phase, round)
		if errors.Is(err, errAgentRetry) && ctx.Err() == nil {
			continue
		}
		return text, duration, callRef, err
	}
}

func (m *Manager) runOne(ctx context.Context, session *Session, status AgentStatus, prompt, phase string, round int) (string, time.Duration, CallRef, error) {
	adapter := m.adapter(status.ID)
	if adapter.id == "" {
		return "", 0, CallRef{}, fmt.Errorf("未找到 Agent Adapter %s", status.ID)
	}
	images := attachmentPaths(session.Request.Attachments)
	if adapter.invoke != nil {
		callCtx, call := m.beginAgentCall(ctx, session, status, phase, round)
		defer m.endAgentCall(session.ID, call)
		text, duration, err := m.runModelSeat(
			callCtx,
			session,
			status,
			call,
			adapter.invoke,
			adapter.stream,
			adapter.streamWithTools,
			prompt,
			images,
			phase,
			round,
		)
		return text, duration, call.CallRef, err
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
	callCtx, call := m.beginAgentCall(ctx, session, status, phase, round)
	defer m.endAgentCall(session.ID, call)
	spec, err := adapter.build(prompt, workDir, images)
	if err != nil {
		m.recordAgentError(session, status, phase, round, call, err)
		return "", 0, call.CallRef, err
	}
	runCtx, cancel := context.WithTimeout(callCtx, defaultAgentLimit)
	defer cancel()
	start := time.Now()
	onDelta := func(chunk string) {
		if m.agentCallAborted(call) {
			return
		}
		m.recordAgentDelta(session, status, phase, round, call, chunk)
	}
	onTool := func(trace ToolTrace) {
		if m.agentCallAborted(call) {
			return
		}
		m.recordToolTrace(session, status, phase, round, call, trace)
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
	if m.agentCallSkipped(call) {
		return "", duration, call.CallRef, errAgentSkipped
	}
	if m.agentCallRetry(call) {
		return "", duration, call.CallRef, errAgentRetry
	}
	if err != nil {
		m.recordAgentError(session, status, phase, round, call, err)
		return "", duration, call.CallRef, err
	}
	text := strings.TrimSpace(parseAgentOutput(raw))
	if m.agentCallSkipped(call) {
		return "", duration, call.CallRef, errAgentSkipped
	}
	if m.agentCallRetry(call) {
		return "", duration, call.CallRef, errAgentRetry
	}
	if text == "" {
		err = errors.New("Agent 返回空结果")
		m.recordAgentError(session, status, phase, round, call, err)
		return "", duration, call.CallRef, err
	}
	text = trimRunes(text, maxAgentOutput)
	m.recordAgentOutput(session, status, phase, round, call, text, duration)
	return text, duration, call.CallRef, nil
}

// runModelSeat 模型席位执行：无 cwd、无 CLI 包装；提示词即全部输入，输出是
// 模型纯文本（不经 parseAgentOutput 的 CLI JSONL 清洗——它可能把合法 JSON
// 讨论小结误当噪音剥离）。
func (m *Manager) runModelSeat(
	ctx context.Context,
	session *Session,
	status AgentStatus,
	call *activeAgentCall,
	invoke func(context.Context, string) (string, error),
	stream func(context.Context, string, func(string)) (string, error),
	streamWithTools func(
		context.Context,
		string,
		string,
		[]string,
		func(string),
		func(ToolTrace),
	) (string, error),
	prompt string,
	images []string,
	phase string,
	round int,
) (string, time.Duration, error) {
	runCtx, cancel := context.WithTimeout(ctx, defaultAgentLimit)
	defer cancel()
	start := time.Now()
	var text string
	var err error
	onDelta := func(chunk string) {
		if m.agentCallAborted(call) {
			return
		}
		m.recordAgentDelta(session, status, phase, round, call, chunk)
	}
	onTool := func(trace ToolTrace) {
		if m.agentCallAborted(call) {
			return
		}
		m.recordToolTrace(session, status, phase, round, call, trace)
	}
	if streamWithTools != nil {
		// 归档发言全文路径注入 readPaths 白名单：提示词只带索引行与预览，
		// grok 席需要引用往轮论证细节时按路径自读（2026-11-03 论证文件化）。
		readPaths := make([]string, 0, len(images)+len(session.Contributions))
		readPaths = append(readPaths, images...)
		if t := m.transcriptFor(session); t != nil {
			readPaths = append(readPaths, t.archiveContributions(session.Contributions)...)
		}
		text, err = streamWithTools(
			runCtx,
			clampPrompt(prompt),
			session.Request.Workspace,
			readPaths,
			onDelta,
			onTool,
		)
	} else if stream != nil {
		text, err = stream(runCtx, clampPrompt(prompt), onDelta)
	} else {
		text, err = invoke(runCtx, clampPrompt(prompt))
	}
	duration := time.Since(start)
	if m.agentCallSkipped(call) {
		return "", duration, errAgentSkipped
	}
	if m.agentCallRetry(call) {
		return "", duration, errAgentRetry
	}
	if err != nil {
		m.recordAgentError(session, status, phase, round, call, err)
		return "", duration, err
	}
	text = strings.TrimSpace(text)
	if m.agentCallSkipped(call) {
		return "", duration, errAgentSkipped
	}
	if m.agentCallRetry(call) {
		return "", duration, errAgentRetry
	}
	if text == "" {
		err = errors.New("模型返回空结果")
		m.recordAgentError(session, status, phase, round, call, err)
		return "", duration, err
	}
	text = trimRunes(text, maxAgentOutput)
	m.recordAgentOutput(session, status, phase, round, call, text, duration)
	return text, duration, nil
}

func (m *Manager) adapter(id string) agentAdapter {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.adapters[id]
}

// transcriptFor 返回本会话的归档句柄；构建失败（仅非法 ID，实际不可达）返回
// nil，提示词构建降级为内联节选，绝不因分发辅助故障挡议事。
func (m *Manager) transcriptFor(session *Session) *transcript {
	t, err := newTranscript(m.root, session.ID)
	if err != nil {
		return nil
	}
	return &t
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

func (m *Manager) recordAgentStarted(
	session *Session,
	status AgentStatus,
	phase string,
	round int,
	call *activeAgentCall,
) {
	text := fmt.Sprintf("%s 开始%s", status.Name, phaseLabel(phase))
	if round > 0 {
		text = fmt.Sprintf("%s 开始第 %d 轮发言", status.Name, round)
	}
	m.mu.Lock()
	now := time.Now().UnixMilli()
	session.UpdatedAt = now
	if err := m.observeSeatLocked(session, status.ID, seatObservation{
		Origin:   OriginLauncher,
		Status:   SeatRunning,
		Evidence: text,
		CallRef:  call.CallRef,
		At:       now,
	}); err != nil {
		// 状态没更新就照实记一笔：宁可让用户看到"这条状态没写进去"，
		// 也不要在界面上假装它开始了（M3）。
		session.Warnings = append(session.Warnings, fmt.Sprintf("%s 状态未能更新：%s", status.Name, err.Error()))
	}
	for i := range session.Participants {
		if session.Participants[i].AgentID == status.ID {
			session.Participants[i].StartedAt = now
			session.Participants[i].FinishedAt = 0
			session.Participants[i].Error = ""
		}
	}
	event := Event{
		CallRef: call.CallRef,
		Phase:   phase, Round: round, AgentID: status.ID,
		Kind: EventAgentStart, Text: text, At: time.Now().UnixMilli(),
	}
	event = m.appendEventLocked(session, event)
	if m.agentCalls[session.ID] == nil {
		m.agentCalls[session.ID] = map[string]*activeAgentCall{}
	}
	m.agentCalls[session.ID][status.ID] = call
	_ = m.store.save(session)
	m.mu.Unlock()
	m.emitSafe(event)
}

func (m *Manager) recordAgentDelta(
	session *Session,
	status AgentStatus,
	phase string,
	round int,
	call *activeAgentCall,
	text string,
) {
	if text == "" {
		return
	}
	now := time.Now().UnixMilli()
	// 增量是"这个席位还在动"的新鲜证据（M10）：只刷新观测时间，不改状态、
	// 不落盘——每来一段增量就写一次 session.json 会把磁盘写穿。
	m.mu.Lock()
	_ = m.observeSeatLocked(session, status.ID, seatObservation{
		Origin:  OriginEvent,
		Status:  SeatRunning,
		CallRef: call.CallRef,
		At:      now,
	})
	m.mu.Unlock()
	m.emitSafe(Event{
		CallRef:   call.CallRef,
		Seq:       -atomic.AddInt64(&m.deltaSeq, 1),
		SessionID: session.ID,
		Phase:     phase,
		Round:     round,
		AgentID:   status.ID,
		Kind:      EventAgentDelta,
		Text:      text,
		At:        now,
	})
}

func (m *Manager) recordToolTrace(
	session *Session,
	status AgentStatus,
	phase string,
	round int,
	call *activeAgentCall,
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
		CallRef:    call.CallRef,
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

func (m *Manager) recordAgentOutput(
	session *Session,
	status AgentStatus,
	phase string,
	round int,
	call *activeAgentCall,
	text string,
	duration time.Duration,
) {
	m.mu.Lock()
	now := time.Now().UnixMilli()
	session.UpdatedAt = now
	for i := range session.Participants {
		if session.Participants[i].AgentID == status.ID {
			if err := m.observeSeatLocked(session, status.ID, seatObservation{
				Origin:   OriginAdapter,
				Status:   SeatFinished,
				Evidence: fmt.Sprintf("调用返回，用时 %.1fs", duration.Seconds()),
				CallRef:  call.CallRef,
				At:       now,
			}); err != nil {
				session.Warnings = append(session.Warnings, fmt.Sprintf("%s 状态未能更新：%s", status.Name, err.Error()))
			}
			session.Participants[i].FinishedAt = now
			session.Participants[i].DurationMs += duration.Milliseconds()
			session.Participants[i].CallRef = call.CallRef
		}
	}
	event := Event{
		CallRef: call.CallRef,
		Phase:   phase, Round: round, AgentID: status.ID, Kind: EventAgentOutput,
		Text: text, At: now, DurationMs: duration.Milliseconds(),
	}
	event = m.appendEventLocked(session, event)
	_ = m.store.save(session)
	m.mu.Unlock()
	m.emitSafe(event)
}

func (m *Manager) recordAgentError(
	session *Session,
	status AgentStatus,
	phase string,
	round int,
	call *activeAgentCall,
	err error,
) {
	m.mu.Lock()
	now := time.Now().UnixMilli()
	session.UpdatedAt = now
	for i := range session.Participants {
		if session.Participants[i].AgentID == status.ID {
			if observeErr := m.observeSeatLocked(session, status.ID, seatObservation{
				Origin:   OriginProcess,
				Status:   SeatFailed,
				Evidence: shortEvidence(err.Error()),
				CallRef:  call.CallRef,
				At:       now,
			}); observeErr != nil {
				session.Warnings = append(session.Warnings, fmt.Sprintf("%s 状态未能更新：%s", status.Name, observeErr.Error()))
			}
			session.Participants[i].FinishedAt = now
			session.Participants[i].CallRef = call.CallRef
			session.Participants[i].Error = err.Error()
		}
	}
	event := Event{
		CallRef: call.CallRef,
		Phase:   phase, Round: round, AgentID: status.ID, Kind: EventAgentError,
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
	// 取消是对整场议事下的命令：还在跑的席位必须一起收口，
	// 否则界面会一直显示"某个席位仍在发言"。
	for i := range session.Participants {
		p := &session.Participants[i]
		switch p.Status {
		case SeatRunning, SeatRetrying, SeatNeedsInput:
			_ = m.observeSeatLocked(session, p.AgentID, seatObservation{
				Origin:   OriginUser,
				Status:   SeatCancelled,
				Evidence: "用户终止了本次议事",
				At:       session.UpdatedAt,
			})
			p.FinishedAt = session.UpdatedAt
		}
	}
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

// consensusShortfallWarning separates substantive disagreement from
// infrastructure states (absent seats, clipped prompts, undeclared signals)
// so a stop without consensus stays auditable instead of collapsing into a
// blanket "still not unanimous" that masks failures (2026-10-31 裁决第一刀).
func consensusShortfallWarning(
	rounds int,
	order []string,
	history []Contribution,
	finalRound int,
	clipped []string,
) string {
	signals := make(map[string]string)
	for _, item := range history {
		if item.Round == finalRound {
			signals[item.AgentID] = item.Consensus
		}
	}
	disagreed := make([]string, 0, len(order))
	undeclared := make([]string, 0, len(order))
	absent := make([]string, 0, len(order))
	for _, id := range order {
		signal, ok := signals[id]
		switch {
		case !ok:
			absent = append(absent, id)
		case signal == "no":
			disagreed = append(disagreed, id)
		case signal == "":
			undeclared = append(undeclared, id)
		}
	}
	parts := make([]string, 0, 4)
	if len(disagreed) > 0 {
		parts = append(parts, fmt.Sprintf("实质分歧 %d 席（%s）", len(disagreed), strings.Join(disagreed, "、")))
	} else {
		parts = append(parts, "末轮没有席位声明仍有分歧")
	}
	if len(undeclared) > 0 {
		parts = append(parts, fmt.Sprintf("末轮 %d 席未声明共识信号（%s）", len(undeclared), strings.Join(undeclared, "、")))
	}
	if len(absent) > 0 {
		parts = append(parts, fmt.Sprintf("末轮缺席 %d 席（%s）——缺席是运行状态，不构成实质分歧", len(absent), strings.Join(absent, "、")))
	}
	if len(clipped) > 0 {
		parts = append(parts, fmt.Sprintf("提示词超限截断 %d 次（%s）——截断可能掩盖真实立场", len(clipped), strings.Join(clipped, "、")))
	}
	return fmt.Sprintf("已达到 %d 轮安全上限，未达成全员一致：%s", rounds, strings.Join(parts, "；"))
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

func seatCallID(sessionID, agentID, phase string, round int) string {
	return fmt.Sprintf("%s/%s/%s/%d", sessionID, agentID, phase, round)
}

func newAttemptID(now int64) string {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("att_%d", now)
	}
	return fmt.Sprintf("att_%d_%s", now, hex.EncodeToString(buf))
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
