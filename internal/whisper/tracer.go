package whisper

import "time"

const traceRingMax = 100

var traceRing []TurnTrace

func LogTurn(trace TurnTrace) {
	now := time.Now()
	if trace.Timestamp == nil {
		trace.Timestamp = &now
	}
	traceRing = append(traceRing, trace)
	if len(traceRing) > traceRingMax {
		traceRing = traceRing[len(traceRing)-traceRingMax:]
	}
}

func TraceLatest(n int) []TurnTrace {
	if n <= 0 {
		return nil
	}
	start := len(traceRing) - n
	if start < 0 {
		start = 0
	}
	result := make([]TurnTrace, len(traceRing)-start)
	copy(result, traceRing[start:])
	return result
}

func TraceRing() []TurnTrace {
	result := make([]TurnTrace, len(traceRing))
	copy(result, traceRing)
	return result
}

func TraceCount() int { return len(traceRing) }

func PatchLatestTurnL5(turn int, toolCalls []string) {
	for i := len(traceRing) - 1; i >= 0; i-- {
		if traceRing[i].Turn != turn {
			continue
		}
		l5 := TurnTraceL5{ToolCalls: toolCalls}
		traceRing[i].L5 = &l5
		break
	}
}
