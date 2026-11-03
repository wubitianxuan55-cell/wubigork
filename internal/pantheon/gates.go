package pantheon

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// 决策门与提问（O-E，Orca M9/M14 的圆桌版）。
//
// 三条边界决定这一刀的形状：
//   - 门与提问都不是注解，它们改状态：等的时候整场是 waiting_input，
//     而不是"running 但没人在干活"——后者正是 Orca 反复修的假活着；
//   - 门只认显式处理：没被 resolve/dismiss 的门跨重启仍然挡路，重启不会
//     替人把门解开（M14 "Only explicit human gate resolution may unblock"）；
//   - 回答只解除阻塞，不偷偷续跑：席位背后的进程在提问那一刻就已经结束，
//     续跑要有可用席位、要由人再点一次，否则就是伪造执行。

// hasOpenGateLocked 从事件历史推导"当前是否有一道没被处理的决策门"。
// 门不存独立字段：会话本身是唯一真相，重启后从事件重放即可判断阻塞，
// 不会出现"字段说没门、事件里还挂着一条"的两份真相。调用方必须持有 m.mu。
func hasOpenGateLocked(session *Session) bool {
	open := false
	for _, event := range session.Events {
		if event.Kind != EventGate || event.Gate == nil {
			continue
		}
		switch event.Gate.Action {
		case GateActionOpen:
			open = true
		case GateActionResolve, GateActionDismiss:
			open = false
		}
	}
	return open
}

// pendingQuestionsLocked 列出还没有对应回答的事件级提问。同一个
// QuestionID 只允许被回答一次（M9 的 exactly-once），所以判定只需要看
// answer 事件里出现过哪些 ID。调用方必须持有 m.mu。
func pendingQuestionsLocked(session *Session) []QuestionPayload {
	answered := make(map[string]bool, len(session.Events))
	for _, event := range session.Events {
		if event.Kind == EventAnswer && event.Answer != nil && event.Answer.QuestionID != "" {
			answered[event.Answer.QuestionID] = true
		}
	}
	var pending []QuestionPayload
	for _, event := range session.Events {
		if event.Kind != EventQuestion || event.Question == nil {
			continue
		}
		if answered[event.Question.QuestionID] {
			continue
		}
		pending = append(pending, *event.Question)
	}
	return pending
}

// seatGaps 列出结案时不能算成功的席位，并给出人能读的原因（§4.2）。
// 缺口必须具名：断线的第五席留在缺口里，不能悄悄算成弃权。
func seatGaps(session *Session) []string {
	var gaps []string
	for _, participant := range session.Participants {
		reason := ""
		switch participant.Status {
		case SeatFinishedUnverified:
			reason = "未验证返回"
		case SeatUnavailable:
			reason = "不可用"
		case SeatSkipped:
			reason = "已跳过"
		case SeatCancelled:
			reason = "已取消"
		default:
			continue
		}
		gaps = append(gaps, fmt.Sprintf("%s（%s）", participant.Name, reason))
	}
	return gaps
}

// notePaused 在循环真的停下时留一条可读的痕迹：用户要知道它停在哪一轮，
// 而不是发现整场忽然不动了。run 调用后立刻返回，所以只会留一条。
func (m *Manager) notePaused(session *Session) {
	m.mu.Lock()
	phase := session.Phase
	round := session.CurrentRound
	m.mu.Unlock()
	m.emitEvent(session, Event{
		Phase: phase,
		Round: round,
		Kind:  EventPhase,
		Text:  "议事已暂停：等待未答的问题或决策门被处理",
	})
}

// openGateInSession 是循环内部用的开门：会话已经在手上，不再按 id 重读，
// 但同样只认 running 且当前没有未处理的门。返回 false 表示没开成。
func (m *Manager) openGateInSession(session *Session, reason string) bool {
	m.mu.Lock()
	if session.Status != StatusRunning || hasOpenGateLocked(session) {
		m.mu.Unlock()
		return false
	}
	now := time.Now().UnixMilli()
	session.Status = StatusWaitingInput
	session.UpdatedAt = now
	event := m.appendEventLocked(session, Event{
		Phase: session.Phase,
		Round: session.CurrentRound,
		Kind:  EventGate,
		Text:  reason,
		Gate:  &GatePayload{Action: GateActionOpen, Reason: reason},
		At:    now,
	})
	if err := m.store.save(session); err != nil {
		m.mu.Unlock()
		return false
	}
	m.mu.Unlock()
	m.emitSafe(event)
	return true
}

// AskSeat 让一席向用户提问：席位进入 waiting_input，整场随之停下等回答。
//
// 拒绝在席位正在发言时插问，理由很实在：席位背后是外部进程，我们没有办法
// 在它输出到一半时把问题塞进去，假装能塞就是假装能打断（M7 的同一条约束）。
func (m *Manager) AskSeat(id, agentID, question string, options []string) (*Session, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return nil, errors.New("提问内容不能为空")
	}
	m.mu.Lock()
	session, err := m.store.get(id)
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	if session.Status != StatusRunning {
		m.mu.Unlock()
		return nil, errors.New("议事不在运行中，无法提问")
	}
	if call, ok := m.agentCalls[session.ID]; ok && call != nil && call.agentID == agentID {
		m.mu.Unlock()
		return nil, errors.New("该席正在发言：外部进程无法中途插问，请等这一席收声后再问")
	}
	if err := SessionTransitionError(session.Status, StatusWaitingInput); err != nil {
		m.mu.Unlock()
		return nil, err
	}
	now := time.Now().UnixMilli()
	payload := &QuestionPayload{
		QuestionID: fmt.Sprintf("%s-q%d", session.ID, len(session.Events)+1),
		Options:    cleanStrings(options),
		AskedBy:    agentID,
		At:         now,
	}
	if err := m.observeSeatLocked(session, agentID, seatObservation{
		Origin:   OriginEvent,
		Status:   SeatNeedsInput,
		Evidence: "向用户提问：" + shortEvidence(question),
		At:       now,
	}); err != nil {
		m.mu.Unlock()
		return nil, err
	}
	session.Status = StatusWaitingInput
	session.UpdatedAt = now
	event := m.appendEventLocked(session, Event{
		Phase:    session.Phase,
		AgentID:  agentID,
		Kind:     EventQuestion,
		Text:     question,
		Question: payload,
		At:       now,
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

// AnswerSeat 把一条回答绑到指定提问上。回答只解除这一席的阻塞，不触发续跑：
// 续跑是一次新的、用户可见的动作（Resume / 解门），不是回答的副作用。
func (m *Manager) AnswerSeat(id, questionID, body string) (*Session, error) {
	questionID = strings.TrimSpace(questionID)
	body = strings.TrimSpace(body)
	if questionID == "" {
		return nil, errors.New("必须指明回答的是哪个提问")
	}
	if body == "" {
		return nil, errors.New("回答内容不能为空")
	}
	m.mu.Lock()
	session, err := m.store.get(id)
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	pending := pendingQuestionsLocked(session)
	var target *QuestionPayload
	for index := range pending {
		if pending[index].QuestionID == questionID {
			target = &pending[index]
			break
		}
	}
	if target == nil {
		m.mu.Unlock()
		return nil, errors.New("该提问不存在或已经回答")
	}
	remaining := false
	for _, question := range pending {
		if question.QuestionID != questionID && question.AskedBy == target.AskedBy {
			remaining = true
			break
		}
	}
	now := time.Now().UnixMilli()
	if !remaining && target.AskedBy != "" {
		// 这一席的问题答完了：它的旧 attempt 早已结束，下次续跑是新 attempt，
		// 所以走控制通道而不是伪造一次 needs_input → running 的"复活"。
		m.rebindSeatLocked(session, target.AskedBy, SeatWaiting, "问题已回答，等待用户继续议事")
	}
	session.UpdatedAt = now
	event := m.appendEventLocked(session, Event{
		Phase:   session.Phase,
		AgentID: target.AskedBy,
		Kind:    EventAnswer,
		Text:    body,
		Answer: &AnswerPayload{
			QuestionID: questionID,
			Body:       body,
			By:         "user",
			At:         now,
		},
		At: now,
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

// AddUserNote 记录用户在讨论进行中的评论（2026-11-03 拍板）。不打断当前发言
// ——外部进程无法拦腰插话，与 AskSeat 是同一条 M7 约束——评论以贡献身份入档，
// 下一席开始构建的提示词会全文注入【用户插话】（prompts.go 分发层永不截断）。
func (m *Manager) AddUserNote(id, content string) (*Session, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("评论内容不能为空")
	}
	m.mu.Lock()
	session, err := m.store.get(id)
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	if session.Status != StatusRunning && session.Status != StatusWaitingInput {
		m.mu.Unlock()
		return nil, errors.New("讨论已结束，无法继续评论")
	}
	now := time.Now().UnixMilli()
	contribution := Contribution{
		ID:        fmt.Sprintf("%s-user-%d", session.ID, time.Now().UnixNano()),
		Phase:     PhaseUserNote,
		Round:     session.CurrentRound,
		AgentID:   "user",
		AgentName: "用户",
		Content:   content,
		CreatedAt: now,
	}
	session.Contributions = append(session.Contributions, contribution)
	session.UpdatedAt = now
	event := m.appendEventLocked(session, Event{
		Phase:   session.Phase,
		Round:   session.CurrentRound,
		AgentID: "user",
		Kind:    EventUserNote,
		Text:    content,
		At:      now,
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

// OpenGate 在轮次之间开一道决策门：整场立刻转为 waiting_input，跑着的循环
// 会在下一个安全点（席位之间/轮次之间）停下，不会有人已经在跑的那一席被拦腰砍。
func (m *Manager) OpenGate(id, reason string) (*Session, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "用户请求在轮次之间停一下"
	}
	m.mu.Lock()
	session, err := m.store.get(id)
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	if session.Status != StatusRunning {
		m.mu.Unlock()
		return nil, errors.New("只有进行中的议事才能开决策门")
	}
	if hasOpenGateLocked(session) {
		m.mu.Unlock()
		return nil, errors.New("已经有一道决策门没被处理")
	}
	if err := SessionTransitionError(session.Status, StatusWaitingInput); err != nil {
		m.mu.Unlock()
		return nil, err
	}
	now := time.Now().UnixMilli()
	session.Status = StatusWaitingInput
	session.UpdatedAt = now
	event := m.appendEventLocked(session, Event{
		Phase: session.Phase,
		Kind:  EventGate,
		Text:  reason,
		Gate:  &GatePayload{Action: GateActionOpen, Reason: reason},
		At:    now,
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

// ResolveGate 处理一道决策门：
//   - resolve：放行，从下一轮接着跑；
//   - dismiss：放弃后续轮次，直接用已有发言收尾（理由必填，因为这是唯一一次
//     让缺口更大的动作，M14 的 dismiss 必须留下人能读的原因）。
func (m *Manager) ResolveGate(id, action, reason string) (*Session, error) {
	action = strings.ToLower(strings.TrimSpace(action))
	switch action {
	case GateActionResolve:
	case GateActionDismiss:
		if strings.TrimSpace(reason) == "" {
			return nil, errors.New("放弃决策门必须写明理由")
		}
	default:
		return nil, errors.New("决策门只支持 resolve / dismiss")
	}

	m.mu.Lock()
	session, err := m.store.get(id)
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	if !hasOpenGateLocked(session) {
		m.mu.Unlock()
		return nil, errors.New("当前没有待处理的决策门")
	}
	request := session.Request
	startRound := session.CurrentRound + 1
	history := contributionsForResume(session)
	m.mu.Unlock()

	// 探活会等 15 秒，绝不能抱着 m.mu 做。
	available, chairman, err := m.probeSeats(request)
	if err != nil {
		return nil, err
	}
	if action == GateActionDismiss {
		// 跳到最后一轮之后：循环体不执行，直接走汇总。
		startRound = request.Rounds + 1
	}
	if startRound < 1 {
		startRound = 1
	}

	m.mu.Lock()
	session, err = m.store.get(id)
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	if !hasOpenGateLocked(session) {
		m.mu.Unlock()
		return nil, errors.New("决策门已被处理")
	}
	now := time.Now().UnixMilli()
	label := "放行决策门，议事继续"
	if action == GateActionDismiss {
		label = "放弃决策门：" + strings.TrimSpace(reason)
	}
	event := m.appendEventLocked(session, Event{
		Phase: session.Phase,
		Kind:  EventGate,
		Text:  label,
		Gate:  &GatePayload{Action: action, Reason: strings.TrimSpace(reason)},
		At:    now,
	})
	session.Status = StatusRunning
	session.Error = ""
	session.UpdatedAt = now
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
		m.run(runCtx, session, available, chairman, startRound, history)
	}()
	out := cloneSession(session)
	m.emitSafe(event)
	return out, nil
}

// pausedLocked 报告"现在就该停下"：要么有一道没解的门，要么整场在等回答。
// 循环只在这两个条件上停，别的状态（比如某席失败）不改变整场的推进节奏。
func (m *Manager) pausedLocked(session *Session) bool {
	return session.Status == StatusWaitingInput || hasOpenGateLocked(session)
}

// paused 是 pausedLocked 的不持锁版本，供 run 循环在每一步之间探一下。
func (m *Manager) paused(session *Session) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pausedLocked(session)
}

// probeSeats 探一次可用席位并挑主持人（与 Resume 相同的口径：只认
// Available 的席，主持人不可用时按顺序退让）。
func (m *Manager) probeSeats(request StartRequest) (map[string]AgentStatus, string, error) {
	probeCtx, cancelProbe := context.WithTimeout(context.Background(), 15*time.Second)
	statuses := m.Agents(probeCtx)
	cancelProbe()
	byID := make(map[string]AgentStatus, len(statuses))
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
		return nil, "", errors.New("没有可用 Agent，无法继续议事")
	}
	chairman := strings.TrimSpace(request.ChairmanAgentID)
	if _, ok := available[chairman]; !ok {
		chairman = pickChairman(request.AgentIDs, available)
	}
	return available, chairman, nil
}

// contributionsForResume 取"可以带进下一轮"的历史：汇总发言不重复喂回去，
// 否则续跑会把上一版小结当成新一轮的发言。
func contributionsForResume(session *Session) []Contribution {
	history := make([]Contribution, 0, len(session.Contributions))
	for _, contribution := range session.Contributions {
		if contribution.Phase == PhaseSynthesis {
			continue
		}
		history = append(history, contribution)
	}
	return history
}
