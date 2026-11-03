package pantheon

import "fmt"

// seatTransitions 是席位状态的显式转移图（§4.1，Orca M12 的可搬部分）。
//
// 四条纪律写死在表里，而不是散在调用点的 if 里：
//   - 终态自环：finished / failed / skipped / unavailable / cancelled 到自身
//     合法（幂等重放），其余状态到自身一律不合法——重复应用同一条观测不是
//     一次状态变化，不该被当成新证据；
//   - finished_unverified 只有一个出口 finished，且只允许"同一 attempt 的陈词
//     被接受"时走（M12 的唯一例外升级通道）；它到 failed 不合法，因为没人
//     看见不等于失败；
//   - needs_input 到 finished 不合法：回答到达才回 running，问题没答完不许
//     自动续、更不许直接判完成；
//   - 表里没有的转移就是禁止：用"表里没有"表达不允许，不另建一张黑名单。
var seatTransitions = map[string][]string{
	SeatWaiting: {SeatRunning, SeatSkipped, SeatUnavailable},
	// running 到 skipped：用户中途跳过某席位，这一席没说完也不算说完。
	SeatRunning:    {SeatFinished, SeatFinishedUnverified, SeatFailed, SeatNeedsInput, SeatCancelled, SeatSkipped},
	SeatNeedsInput: {SeatRunning, SeatSkipped, SeatCancelled},
	// 唯一的"升级"：未验证完成 → 已验证完成。
	SeatFinishedUnverified: {SeatFinished},
	// retrying 到 cancelled：重试期间用户终止整场议事，席位要跟着收口。
	SeatRetrying:    {SeatRunning, SeatFailed, SeatSkipped, SeatCancelled},
	SeatFinished:           {SeatFinished},
	SeatFailed:             {SeatFailed},
	SeatSkipped:            {SeatSkipped},
	SeatUnavailable:        {SeatUnavailable},
	SeatCancelled:          {SeatCancelled},
}

// sessionTransitions 是议事会话状态的显式转移图（§4.1）。
//
// waiting_input 是"有未答问题"：只能被回答推进回 running，或被用户取消、
// 或被重启记为 interrupted；它到 completed 不合法——问题没答完不许结案。
// interrupted 是重启的落点，属于终态：被中断的议事不会被偷偷续上，恢复是
// 一次新的、用户可见的动作，不是状态倒退。
var sessionTransitions = map[string][]string{
	StatusRunning:      {StatusWaitingInput, StatusCompleted, StatusFailed, StatusCancelled, StatusInterrupted},
	StatusWaitingInput: {StatusRunning, StatusCancelled, StatusInterrupted},
	StatusCompleted:    {StatusCompleted},
	StatusFailed:       {StatusFailed},
	StatusCancelled:    {StatusCancelled},
	StatusInterrupted:  {StatusInterrupted},
}

// CanTransitionSeat 报告席位状态能否从 from 走到 to。from == to 只在表里显式
// 登记过（终态自环）时才为真；未知状态一律不允许（fail closed）。
func CanTransitionSeat(from, to string) bool {
	for _, allowed := range seatTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// SeatTransitionError 返回拒绝一次席位状态转移的理由，可转移时返回 nil。
// 这个函数存在的意义是让步调用带上人能读的原因，而不是让写入方各写一句。
func SeatTransitionError(from, to string) error {
	if CanTransitionSeat(from, to) {
		return nil
	}
	if from == to {
		return fmt.Errorf("席位状态不能从 %s 变为 %s：只有终态允许自环", from, to)
	}
	return fmt.Errorf("席位状态不能从 %s 变为 %s", from, to)
}

// CanTransitionSession 报告会话状态能否从 from 走到 to，规则与席位表一致：
// 表里没有就是不允许，终态只自环。
func CanTransitionSession(from, to string) bool {
	for _, allowed := range sessionTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// SessionTransitionError 返回拒绝一次会话状态转移的理由，可转移时返回 nil。
func SessionTransitionError(from, to string) error {
	if CanTransitionSession(from, to) {
		return nil
	}
	if from == to {
		return fmt.Errorf("议事状态不能从 %s 变为 %s：只有终态允许自环", from, to)
	}
	return fmt.Errorf("议事状态不能从 %s 变为 %s", from, to)
}
