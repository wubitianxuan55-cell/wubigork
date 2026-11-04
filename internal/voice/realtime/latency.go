// latency.go — 轮次延迟账 + 日志行（规格 §6；eot_analyze.py/false_interrupt_bound.py 解析格式）。
package realtime

import (
	"log/slog"
	"strconv"
	"strings"
	"time"
)

// latencyRec 单响应延迟账。
type latencyRec struct {
	sttS       float64
	llmS       float64
	ttsTTFA    float64
	e2eS       float64
	vadDecideS float64
	holdS      float64
	smart      string
}

// fmtS "n/a" 或 %.2f（规格 _fmt）。
func fmtS(v float64) string {
	if v < 0 {
		return "n/a"
	}
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// ChatMessage LLM 对话历史条目（handlers/history.py 同位：暂存代际制）。
type ChatMessage struct {
	Role        string `json:"role"`
	Content     string `json:"content"`
	respKey     string // 非空=某响应的暂存输出（可回滚）
	provisional bool   // true=未 finalize（取消即回滚）
	turnKey     string // user 条目的轮次键（同轮重开替换文本）
}

// slogLatencyLine 逐字对齐：
// "Turn {turn} rev={rev} latency: stt=...s llm=...s tts_ttfa=...s e2e=...s vad_decision=...s hold=...s smart_turn_status=... status=...[ tts=cut][ source=speech] response_key=..."
func slogLatencyLine(rs *responseState, rec latencyRec, status string) {
	rev := rs.rev
	var b strings.Builder
	b.WriteString("Turn ")
	b.WriteString(rs.turnID)
	b.WriteString(" rev=")
	b.WriteString(strconv.Itoa(rev))
	b.WriteString(" latency: stt=")
	b.WriteString(fmtS(rec.sttS))
	b.WriteString("s llm=")
	b.WriteString(fmtS(rec.llmS))
	b.WriteString("s tts_ttfa=")
	b.WriteString(fmtS(rec.ttsTTFA))
	b.WriteString("s e2e=")
	b.WriteString(fmtS(rec.e2eS))
	b.WriteString("s vad_decision=")
	b.WriteString(fmtS(rec.vadDecideS))
	b.WriteString("s hold=")
	b.WriteString(fmtS(rec.holdS))
	b.WriteString("s smart_turn_status=")
	if rec.smart == "" {
		b.WriteString("n/a")
	} else {
		b.WriteString(rec.smart)
	}
	b.WriteString(" status=")
	b.WriteString(status)
	if status == "cancelled" && rec.ttsTTFA > 0 {
		b.WriteString(" tts=cut")
	}
	b.WriteString(" source=speech")
	b.WriteString(" response_key=")
	b.WriteString(rs.key)
	slog.Info(b.String())
}

// spokenSplit 口语分句（流式批切，stream_batch_sentences=1 语义）。
// CJK 句读 + 中西标点；保留省略号/引号闭合；长度上限兜底（60 rune 强切）。
func spokenSplit(text string) []string {
	var out []string
	var cur strings.Builder
	runes := []rune(text)
	const hardLimit = 60
	flush := func() {
		if s := strings.TrimSpace(cur.String()); s != "" {
			out = append(out, s)
		}
		cur.Reset()
	}
	for i, r := range runes {
		cur.WriteRune(r)
		boundary := false
		switch r {
		case '。', '！', '？', '；', '…', '\n':
			boundary = true
		case '.', '!', '?', ';':
			// 西文句读：后随空白/结尾才切（避免 3.14/缩写撕裂）
			if i+1 >= len(runes) || runes[i+1] == ' ' || runes[i+1] == '"' {
				boundary = true
			}
		}
		if boundary || cur.Len() >= hardLimit {
			flush()
		}
	}
	flush()
	return out
}

var _ = time.Now
