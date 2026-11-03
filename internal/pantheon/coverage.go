package pantheon

import (
	"fmt"
)

// seatRoundCoverage 按 CallID 重放事件流，现算某轮各席位的覆盖状态
// （2026-11-03 V2 P0 事件投影）：started / succeeded / failed / skipped /
// not-started。它是纯函数，不落第二份字段——席位 Participant.Status 在多轮
// 场里是冻结的（seatstate 终态自环+对象跨轮复用），读它会谎报「全员在场」；
// 覆盖、部分完成与签注缺口都必须从事件重放获得（pan_1793681319746 r8/r9
// 四席代码链共识）。
func seatRoundCoverage(session *Session, round int) map[string]string {
	agents := uniqueNonEmpty(session.Request.AgentIDs)
	coverage := make(map[string]string, len(agents))
	for _, id := range agents {
		coverage[id] = "not-started"
	}
	// 该轮每席取「最后一个有结论的事件」：error/skipped/output 之后的
	// started 属于重试的新 attempt，以最新 attempt 的结论为准。
	lastStart := make(map[string]string, len(agents)) // agentID -> 最新 attempt 的 CallID
	for _, event := range session.Events {
		if event.Round != round {
			continue
		}
		switch event.Kind {
		case EventAgentStart:
			if event.AgentID != "" {
				coverage[event.AgentID] = "started"
				lastStart[event.AgentID] = event.CallID
			}
		case EventAgentOutput:
			if event.AgentID != "" {
				coverage[event.AgentID] = "succeeded"
			}
		case EventAgentError:
			if event.AgentID != "" {
				coverage[event.AgentID] = "failed"
			}
		case EventAgentSkipped:
			if event.AgentID != "" {
				coverage[event.AgentID] = "skipped"
			}
		}
	}
	_ = lastStart // CallID 现算保留口径：输出/错误事件当前不带 CallID 过滤，按 agent+round 终态即可
	return coverage
}

// coverageGaps 把末轮覆盖投影转成签注缺口清单（与 seatGaps 的席位名口径一致，
// 但读的是事件真相而不是冻结状态）。
func coverageGaps(session *Session, round int) []string {
	reasons := map[string]string{
		"failed":      "末轮调用失败",
		"skipped":     "末轮被跳过",
		"not-started": "末轮未启动",
	}
	names := make(map[string]string, len(session.Participants))
	for _, participant := range session.Participants {
		names[participant.AgentID] = participant.Name
	}
	coverage := seatRoundCoverage(session, round)
	var gaps []string
	for _, id := range uniqueNonEmpty(session.Request.AgentIDs) {
		reason, bad := reasons[coverage[id]]
		if !bad {
			continue
		}
		name := names[id]
		if name == "" {
			name = id
		}
		gaps = append(gaps, fmt.Sprintf("%s（%s）", name, reason))
	}
	return gaps
}
