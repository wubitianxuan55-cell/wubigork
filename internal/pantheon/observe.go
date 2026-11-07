package pantheon

import (
	"fmt"
	"strings"
	"time"
)

// 席位观测的出处词表（Orca agent-status-observation 的圆桌版）。
// 前六个是证据；后两个只重绘，不算证据（M10：快照与心跳不续命）。
const (
	OriginSnapshot  = "snapshot"
	OriginHeartbeat = "heartbeat"
)

// seatObservation 是一次"我们看见了什么"。
// 只有状态来源明确的观测才允许写进席位：调用方必须说明出处，否则无从判断
// 这条状态能不能被相信（§2.1 的两层新鲜度靠它区分）。
type seatObservation struct {
	Origin   string
	Status   string
	Evidence string
	CallRef  CallRef
	At       int64
}

// isFreshnessEvidence 判定一次观测是否构成新证据。
// 快照是重绘、心跳是进度显示：两者都不能把"安静的席位"续命（M10）。
func isFreshnessEvidence(origin string) bool {
	switch origin {
	case OriginSnapshot, OriginHeartbeat:
		return false
	}
	return true
}

// observeSeatLocked 是席位状态的唯一写入口（O-D）。
//
// 三条纪律：
//  1. 同态观测不是状态转移——只推进 revision 与证据时间，不查转移表；
//  2. 状态变化必须过 seatTransitions，非法转移返回错误且不落地（fail closed）；
//  3. 换进程后 revision 从 1 重新计数：旧 authority 的序号与新 authority 不可比，
//     继续累加会造出"状态倒退了"的假结论（§2.2）。
func (m *Manager) observeSeatLocked(session *Session, agentID string, obs seatObservation) error {
	if !isFreshnessEvidence(obs.Origin) {
		return nil
	}
	if obs.At == 0 {
		obs.At = time.Now().UnixMilli()
	}
	for i := range session.Participants {
		p := &session.Participants[i]
		if p.AgentID != agentID {
			continue
		}
		if p.Status != obs.Status {
			if err := SeatTransitionError(p.Status, obs.Status); err != nil {
				return err
			}
			p.Status = obs.Status
		}
		revision := p.Observed.Revision + 1
		if p.Observed.AuthorityID != m.authorityID {
			revision = 1
		}
		p.Observed = Observed{
			Origin:      obs.Origin,
			AuthorityID: m.authorityID,
			Incarnation: p.Observed.Incarnation,
			Revision:    revision,
			At:          obs.At,
		}
		if obs.Evidence != "" {
			p.Evidence = obs.Evidence
		}
		if obs.CallRef.CallID != "" {
			p.CallRef = obs.CallRef
		}
		return nil
	}
	return fmt.Errorf("席位不存在：%s", agentID)
}

// shortEvidence 把证据压到一行可读长度：席位卡片上只放得下一条短句，
// 完整错误仍在 Participant.Error 与事件里。
func shortEvidence(text string) string {
	const limit = 120
	text = strings.TrimSpace(text)
	if len([]rune(text)) <= limit {
		return text
	}
	return string([]rune(text)[:limit]) + "…"
}

// rebindSeatLocked 是一次控制动作，不是观测：重试、重开整场议事都会给席位
// 发一个新的 attempt，旧 attempt 的终态不该拦住新 attempt 的起步（M12 的
// 具名例外通道）。因此它直接置状态并抬 incarnation，绕过转移表。
// 调用方必须持有 m.mu。
// startSeatRoundLocked 是轮边界的具名控制动作（2026-11-03 V2 P0）：
// 新一轮启动席位属于宿主调度（OriginLauncher），不是用户动作也不是普通观测——
// observeSeatLocked 的转移表会拒 finished→running（单次 attempt 纪律，正确），
// 但跨轮复用的 Participant 需要显式的「新 attempt」语义：直接置 running 并抬
// Incarnation，与 rebindSeatLocked 同构但出处是启动器，不得冒充用户动作。
func (m *Manager) startSeatRoundLocked(session *Session, agentID, evidence string, ref CallRef) {
	for i := range session.Participants {
		p := &session.Participants[i]
		if p.AgentID != agentID {
			continue
		}
		p.Status = SeatRunning
		p.Observed = Observed{
			Origin:      OriginLauncher,
			AuthorityID: m.authorityID,
			Incarnation: p.Observed.Incarnation + 1,
			Revision:    1,
			At:          time.Now().UnixMilli(),
		}
		p.Evidence = evidence
		p.CallRef = ref
		return
	}
}

func (m *Manager) rebindSeatLocked(session *Session, agentID, status, evidence string) {
	m.rebindSeatLockedWithOrigin(session, agentID, status, evidence, OriginUser)
}

// rebindSeatLockedWithOrigin 与 rebindSeatLocked 同构，但允许控制动作声明
// 自己的出处：席位自述提问是事件证据（OriginEvent），不是用户动作。
func (m *Manager) rebindSeatLockedWithOrigin(session *Session, agentID, status, evidence, origin string) {
	if origin == "" {
		origin = OriginUser
	}
	for i := range session.Participants {
		p := &session.Participants[i]
		if p.AgentID != agentID {
			continue
		}
		p.Status = status
		p.Observed = Observed{
			Origin:      origin,
			AuthorityID: m.authorityID,
			Incarnation: p.Observed.Incarnation + 1,
			Revision:    1,
			At:          time.Now().UnixMilli(),
		}
		p.Evidence = evidence
		return
	}
}
