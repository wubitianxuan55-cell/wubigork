// turns.go — SpeculativeTurnTracker Go 移植。
//
// 语义逐字对齐 sidecar pipeline/speculative_turns.py（上游 6feb527），
// 移植规格：进度计划/gaea-voice-sidecar-port-spec-20261104.md §3。
package realtime

import (
	"strconv"
	"sync"
	"time"
)

// TurnPhase 轮次相位。
type TurnPhase int32

const (
	PhaseNone TurnPhase = iota
	PhaseListening
	PhaseSoftEnded
	PhaseAnswering
	PhaseClosed
)

func (p TurnPhase) String() string {
	switch p {
	case PhaseListening:
		return "listening"
	case PhaseSoftEnded:
		return "soft_ended"
	case PhaseAnswering:
		return "answering"
	case PhaseClosed:
		return "closed"
	}
	return "none"
}

// GateAction gate() 判决。
type GateAction int32

const (
	GateAccept GateAction = iota
	GateDrop
	GateHold
)

func (a GateAction) String() string {
	switch a {
	case GateAccept:
		return "accept"
	case GateDrop:
		return "drop"
	case GateHold:
		return "hold"
	}
	return "?"
}

// TurnGate gate 判决结果；Hold 时 Deadline 为等待上界。
type TurnGate struct {
	Action   GateAction
	Deadline time.Time
}

const (
	pendingReopenHoldS  = 2 * time.Second
	defaultUnansweredMs = 7000
)

// turnRef current 引用（seq 单调递增，同 id 不同 rev 复用 seq）。
type turnRef struct {
	seq int
	id  string
	rev int
}

// turnKey committed/closed 集合键——生命周期内 (id,rev) 唯一。
type turnKey struct {
	id  string
	rev int
}

func keyOf(id string, rev int) turnKey { return turnKey{id: id, rev: rev} }

type pendingReopen struct {
	turnID   string
	baseRev  int
	candRev  int
	deadline time.Time
}

type reopenGrace struct {
	turnID   string
	rev      int
	deadline time.Time
}

// SpeculativeTurns 推测轮次追踪器（单管线一实例；方法并发安全）。
type SpeculativeTurns struct {
	mu     sync.Mutex
	notify chan struct{} // 关闭+重建广播

	seq           int
	cur           *turnRef
	committed     map[turnKey]bool
	closedCurrent *turnKey
	pending       *pendingReopen
	grace         *reopenGrace
	phase         TurnPhase
	lastFinalMs   int
	hasLastFinal  bool
	procDeadline  time.Time
	hasProcDline  bool
	unansweredMs  int
	waitObserver  func(turnID string, rev int, started, ended time.Time)
}

// NewSpeculativeTurns 构造（unanswered 上限缺省 7000ms 音频时钟）。
func NewSpeculativeTurns() *SpeculativeTurns {
	return &SpeculativeTurns{
		notify:       make(chan struct{}),
		committed:    map[turnKey]bool{},
		unansweredMs: defaultUnansweredMs,
	}
}

// SetWaitObserver 挂观察器（smart_turn_wait 区间统计用）。
func (t *SpeculativeTurns) SetWaitObserver(fn func(turnID string, rev int, started, ended time.Time)) {
	t.mu.Lock()
	t.waitObserver = fn
	t.mu.Unlock()
}

// ConfigureReopen 配置未答重开上限（音频 ms）。
func (t *SpeculativeTurns) ConfigureReopen(ms int) {
	t.mu.Lock()
	if ms < 0 {
		ms = 0
	}
	t.unansweredMs = ms
	t.mu.Unlock()
}

func (t *SpeculativeTurns) broadcastLocked() {
	close(t.notify)
	t.notify = make(chan struct{})
}

// Phase 当前相位。
func (t *SpeculativeTurns) Phase() TurnPhase {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.phase
}

// CurrentTurn 返回 (turnID, revision)；无 current 时两者皆空串/-1。
func (t *SpeculativeTurns) CurrentTurn() (string, int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cur == nil {
		return "", -1
	}
	return t.cur.id, t.cur.rev
}

// StartTurn 开新轮（turn_{n}，rev=0），清全部悬挂态。
func (t *SpeculativeTurns) StartTurn() (string, int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.startTurnLocked()
}

func (t *SpeculativeTurns) startTurnLocked() (string, int) {
	t.seq++
	t.cur = &turnRef{seq: t.seq, id: "turn_" + strconv.Itoa(t.seq), rev: 0}
	t.closedCurrent = nil
	t.phase = PhaseListening
	t.hasLastFinal = false
	t.hasProcDline = false
	t.pending = nil
	t.grace = nil
	t.broadcastLocked()
	return t.cur.id, 0
}

// blocksLocked 提交/关闭中的轮次不可重开。
func (t *SpeculativeTurns) blocksLocked(id string, rev int) bool {
	if t.committed[keyOf(id, rev)] {
		return true
	}
	if t.closedCurrent != nil && *t.closedCurrent == keyOf(id, rev) {
		return true
	}
	return false
}

// canReopenLocked（§3 can_reopen）：SOFT_ENDED + 未阻塞 + 音频时钟上限。
func (t *SpeculativeTurns) canReopenLocked(audioMs int) bool {
	return t.cur != nil &&
		t.phase == PhaseSoftEnded &&
		t.hasLastFinal &&
		!t.blocksLocked(t.cur.id, t.cur.rev) &&
		max0(audioMs-t.lastFinalMs) <= t.unansweredMs
}

func max0(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

// CanReopen 导出查询（VAD 延迟确认用）。
func (t *SpeculativeTurns) CanReopen(audioMs int) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.canReopenLocked(audioMs)
}

// SpeechCandidateStarted 起重开候选（持有输出 ≤2s）；已有候选幂等返回 true。
func (t *SpeculativeTurns) SpeechCandidateStarted(audioMs int) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.pending != nil {
		return true
	}
	if !t.canReopenLocked(audioMs) {
		return false
	}
	return t.beginReopenCandidateLocked(t.cur.id, t.cur.rev)
}

// SpeechCandidateCancelled 取消候选。
func (t *SpeculativeTurns) SpeechCandidateCancelled() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.pending != nil {
		t.cancelReopenCandidateLocked(t.pending.turnID)
	}
}

// HasSpeechCandidate 是否持有重开候选。
func (t *SpeculativeTurns) HasSpeechCandidate() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.pending != nil
}

// SpeechStarted VAD 确认语音起点：续 LISTENING 原轮 / 确认重开 rev+1 / 开新轮。
func (t *SpeculativeTurns) SpeechStarted(audioMs int) (turnID string, rev int, reopened bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cur != nil && t.phase == PhaseListening {
		return t.cur.id, t.cur.rev, false
	}
	if t.pending == nil && t.canReopenLocked(audioMs) && t.cur != nil {
		t.beginReopenCandidateLocked(t.cur.id, t.cur.rev)
	}
	if t.pending != nil {
		pending := *t.pending
		if t.confirmReopenCandidateLocked(pending.turnID, pending.baseRev, pending.candRev) {
			return pending.turnID, pending.candRev, true
		}
	}
	id, rev := t.startTurnLocked()
	return id, rev, false
}

// SegmentFinalized VAD soft-end：SOFT_ENDED + 记音频游标 + 处理死线 + 重开宽限。
func (t *SpeculativeTurns) SegmentFinalized(audioMs int, outputHoldMs int, processingDelayMs int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cur == nil || t.blocksLocked(t.cur.id, t.cur.rev) {
		return
	}
	t.phase = PhaseSoftEnded
	t.lastFinalMs = audioMs
	t.hasLastFinal = true
	if processingDelayMs > 0 {
		t.procDeadline = time.Now().Add(time.Duration(processingDelayMs) * time.Millisecond)
		t.hasProcDline = true
	} else {
		t.hasProcDline = false
	}
	t.startReopenGraceLocked(t.cur.id, t.cur.rev, time.Duration(max0(outputHoldMs))*time.Millisecond)
	t.broadcastLocked()
}

// SegmentDiscarded 确认过的语音没有可用段：cancel 候选；LISTENING→SOFT_ENDED。
func (t *SpeculativeTurns) SegmentDiscarded() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.pending != nil {
		t.cancelReopenCandidateLocked(t.pending.turnID)
	}
	if t.phase == PhaseListening {
		t.phase = PhaseSoftEnded
	}
	t.broadcastLocked()
}

// ProcessingDeadline current 轮的 smart-turn 处理死线（非 current → 无）。
func (t *SpeculativeTurns) ProcessingDeadline(turnID string, rev int) (time.Time, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if turnID == "" || rev < 0 || t.cur == nil || t.cur.id != turnID || t.cur.rev != rev {
		return time.Time{}, false
	}
	return t.procDeadline, t.hasProcDline
}

// IsLatest (turn,rev) 是否 current 或 committed。
func (t *SpeculativeTurns) IsLatest(turnID string, rev int) bool {
	if turnID == "" || rev < 0 {
		return true
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.isLatestLocked(turnID, rev)
}

func (t *SpeculativeTurns) isLatestLocked(turnID string, rev int) bool {
	if t.cur != nil && t.cur.id == turnID && t.cur.rev == rev {
		return true
	}
	return t.committed[keyOf(turnID, rev)]
}

// Gate 输出/提交闸门（§3 gate）。holdUntil 零值=无额外等待。
func (t *SpeculativeTurns) Gate(turnID string, rev int, holdForGrace bool, holdUntil time.Time, commit bool) TurnGate {
	if turnID == "" || rev < 0 {
		return TurnGate{Action: GateAccept}
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.gateLocked(turnID, rev, holdForGrace, holdUntil, commit)
}

func (t *SpeculativeTurns) gateLocked(turnID string, rev int, holdForGrace bool, holdUntil time.Time, commit bool) TurnGate {
	if !t.isLatestLocked(turnID, rev) {
		return TurnGate{Action: GateDrop}
	}
	if !t.committed[keyOf(turnID, rev)] {
		now := time.Now()
		var minD time.Time
		consider := func(d time.Time) {
			if d.After(now) && (minD.IsZero() || d.Before(minD)) {
				minD = d
			}
		}
		if !holdUntil.IsZero() {
			consider(holdUntil)
		}
		if t.pending != nil && t.pending.turnID == turnID && t.pending.baseRev == rev && t.pending.deadline.After(now) {
			consider(t.pending.deadline)
		}
		if holdForGrace && t.grace != nil && t.grace.turnID == turnID && t.grace.rev == rev &&
			t.cur != nil && t.cur.id == turnID && t.cur.rev == rev && t.grace.deadline.After(now) {
			consider(t.grace.deadline)
		}
		if !minD.IsZero() {
			return TurnGate{Action: GateHold, Deadline: minD}
		}
	}
	if commit {
		t.commitLocked(turnID, rev)
	}
	return TurnGate{Action: GateAccept}
}

// WaitForGate 阻塞到非 HOLD（绝不从事件循环调用）。返回是否 ACCEPT。
func (t *SpeculativeTurns) WaitForGate(turnID string, rev int, holdForGrace bool, holdUntil time.Time) bool {
	if turnID == "" || rev < 0 {
		return true
	}
	for {
		t.mu.Lock()
		g := t.gateLocked(turnID, rev, holdForGrace, holdUntil, false)
		if g.Action != GateHold {
			t.mu.Unlock()
			return g.Action == GateAccept
		}
		deadline := g.Deadline
		waitingForGrace := holdUntil.IsZero() && !t.pendingHoldLocked(turnID, rev)
		ch := t.notify
		observer := t.waitObserver
		t.mu.Unlock()

		waitStarted := time.Now()
		timer := time.NewTimer(time.Until(deadline))
		select {
		case <-ch:
		case <-timer.C:
		}
		timer.Stop()
		if waitingForGrace && observer != nil {
			observer(turnID, rev, waitStarted, time.Now())
		}
	}
}

func (t *SpeculativeTurns) pendingHoldLocked(turnID string, rev int) bool {
	return t.pending != nil && t.pending.turnID == turnID && t.pending.baseRev == rev && t.pending.deadline.After(time.Now())
}

// Close response 终态：从 committed 移除；仍 current 则 CLOSED（不可重开，回答有效）。
func (t *SpeculativeTurns) Close(turnID string, rev int) {
	if turnID == "" || rev < 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	k := keyOf(turnID, rev)
	if !t.committed[k] {
		return
	}
	delete(t.committed, k)
	if t.cur != nil && t.cur.id == turnID && t.cur.rev == rev {
		t.closedCurrent = &k
		t.phase = PhaseClosed
		t.hasProcDline = false
	}
	t.broadcastLocked()
}

// IsCurrentTurn turn 是否 current（任意 revision）。
func (t *SpeculativeTurns) IsCurrentTurn(turnID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cur != nil && t.cur.id == turnID
}

// IsCommitted turn 是否已提交（含已关闭的 current）；rev<0 时只看 turn。
func (t *SpeculativeTurns) IsCommitted(turnID string, rev int) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if rev >= 0 && t.committed[keyOf(turnID, rev)] {
		return true
	}
	if rev < 0 {
		for k := range t.committed {
			if k.id == turnID {
				return true
			}
		}
	}
	if t.closedCurrent != nil && t.closedCurrent.id == turnID && (rev < 0 || t.closedCurrent.rev == rev) {
		return true
	}
	return false
}

func (t *SpeculativeTurns) beginReopenCandidateLocked(turnID string, rev int) bool {
	if turnID == "" || rev < 0 || t.cur == nil || t.cur.id != turnID || t.cur.rev != rev || t.blocksLocked(turnID, rev) {
		return false
	}
	if t.pending != nil {
		if t.pending.turnID == turnID && t.pending.baseRev == rev {
			return true // 幂等
		}
		return false
	}
	t.pending = &pendingReopen{
		turnID:   turnID,
		baseRev:  rev,
		candRev:  rev + 1,
		deadline: time.Now().Add(pendingReopenHoldS),
	}
	t.broadcastLocked()
	return true
}

func (t *SpeculativeTurns) confirmReopenCandidateLocked(turnID string, baseRev, candRev int) bool {
	if turnID == "" || t.pending == nil ||
		t.pending.turnID != turnID || t.pending.baseRev != baseRev || t.pending.candRev != candRev {
		return false
	}
	if t.cur == nil || t.cur.id != turnID || t.cur.rev != baseRev || t.blocksLocked(turnID, baseRev) {
		t.pending = nil
		t.broadcastLocked()
		return false
	}
	t.cur = &turnRef{seq: t.cur.seq, id: turnID, rev: candRev}
	t.phase = PhaseListening
	t.hasProcDline = false
	t.pending = nil
	t.grace = nil
	t.broadcastLocked()
	return true
}

func (t *SpeculativeTurns) cancelReopenCandidateLocked(turnID string) {
	if t.pending != nil && t.pending.turnID == turnID {
		t.pending = nil
		t.broadcastLocked()
	}
}

func (t *SpeculativeTurns) commitLocked(turnID string, rev int) bool {
	if t.committed[keyOf(turnID, rev)] {
		return true
	}
	if t.cur == nil || t.cur.id != turnID || t.cur.rev != rev {
		return false
	}
	t.committed[keyOf(turnID, rev)] = true
	t.phase = PhaseAnswering
	t.hasProcDline = false
	if t.grace != nil && t.grace.turnID == turnID && t.grace.rev == rev {
		t.grace = nil
	}
	t.broadcastLocked()
	return true
}

func (t *SpeculativeTurns) startReopenGraceLocked(turnID string, rev int, grace time.Duration) {
	if turnID == "" || rev < 0 || grace <= 0 {
		return
	}
	if t.cur == nil || t.cur.id != turnID || t.cur.rev != rev || t.blocksLocked(turnID, rev) {
		return
	}
	deadline := time.Now().Add(grace)
	if t.grace != nil && t.grace.turnID == turnID && t.grace.rev == rev && !deadline.After(t.grace.deadline) {
		return // 仅延长
	}
	t.grace = &reopenGrace{turnID: turnID, rev: rev, deadline: deadline}
	t.broadcastLocked()
}

// Reset 清全部（新会话）。
func (t *SpeculativeTurns) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.seq = 0
	t.cur = nil
	t.committed = map[turnKey]bool{}
	t.closedCurrent = nil
	t.pending = nil
	t.grace = nil
	t.phase = PhaseNone
	t.hasLastFinal = false
	t.hasProcDline = false
	t.broadcastLocked()
}
