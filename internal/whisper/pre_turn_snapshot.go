package whisper

// PreTurnSnapshot 是 PreLLMTurn 的完整进程内回滚点。除持久化 State 外，
// 情绪涌现/成人模式/节奏计数器也属于回合状态；V2 生成前预推进后若上游模型
// 失败或被客户端取消，必须一并归还，否则下一轮会被这轮幽灵状态污染。
type PreTurnSnapshot struct {
	State                      FullState
	RecentEventTypes           []string
	ConsecutiveMeaningfulCount int
	ConsecutiveVulnerableCount int
	AdultStateStr              string
	AdultBudget                float64
	AdultLockTurns             int
	AdultConsecutiveVulnerable int
	AdultLastRejectedTurn      int
	Rhythm                     RhythmCounters
	MemoryGraphRestored        bool
}

func (o *Orchestrator) SnapshotPreTurn() PreTurnSnapshot {
	return PreTurnSnapshot{
		State:                      CloneFullState(o.State),
		RecentEventTypes:           append([]string(nil), o.recentEventTypes...),
		ConsecutiveMeaningfulCount: o.consecutiveMeaningfulCount,
		ConsecutiveVulnerableCount: o.consecutiveVulnerableCount,
		AdultStateStr:              o.adultStateStr,
		AdultBudget:                o.adultBudget,
		AdultLockTurns:             o.adultLockTurns,
		AdultConsecutiveVulnerable: o.adultConsecutiveVulnerable,
		AdultLastRejectedTurn:      o.adultLastRejectedTurn,
		Rhythm:                     o.rhythm,
		MemoryGraphRestored:        o.memoryGraphRestored,
	}
}

func (o *Orchestrator) RestorePreTurn(s PreTurnSnapshot) {
	o.State = CloneFullState(s.State)
	o.recentEventTypes = append([]string(nil), s.RecentEventTypes...)
	o.consecutiveMeaningfulCount = s.ConsecutiveMeaningfulCount
	o.consecutiveVulnerableCount = s.ConsecutiveVulnerableCount
	o.adultStateStr = s.AdultStateStr
	o.adultBudget = s.AdultBudget
	o.adultLockTurns = s.AdultLockTurns
	o.adultConsecutiveVulnerable = s.AdultConsecutiveVulnerable
	o.adultLastRejectedTurn = s.AdultLastRejectedTurn
	o.rhythm = s.Rhythm
	o.memoryGraphRestored = s.MemoryGraphRestored
}
