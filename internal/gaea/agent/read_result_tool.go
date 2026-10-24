package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// ── read_subagent_result（B2，Reasonix 2.x subagent_result 蒸馏，v4.524）──
//
// 病根：子代理最终回答只有 spawn 当时内联进 task 结果的一份——父模型回合
// 结束后它随上下文滚动被压缩/丢弃，ref 的唯一消费口是 continue_from（重新
// 起 LLM 循环=重复付费）或人工开 UI 抄。上游的场景：并行聚合没送达（重启/
// 崩溃打断）时，完成态 child 留盘可按 ref 取回，避免为同一工作付两次推理费。
//
// 取回语义对齐上游 read()：终态才可读（completed=最终回答；failed=部分转录
// 且显式标注「非结果」，防把 partial 当完成——上游 partialTranscript 同款），
// 取转录里最后一条非空 assistant 消息，按 UTF-8 字节分页（默认 12KB、上限
// 24KB，页尾给 next_offset）。
//
// 刻意差异（留档）：①只读 sa_——mt_ 模型工具的结果在调用时已内联返回，无
// 「取回」场景；②不做血统校验——store 目录即作用域（dirFn 跟随当前会话），
// ref 不可枚举，与 interrupt_agent/continue_from 同一威胁模型；③假 running
// **不改写 meta**——上游 CleanupStaleRunning 靠会话租约授权破坏性改写，gaea
// 无租约（桌面多窗可同开同一工作区），读端判活给诚实文案即可。
const (
	readResultDefaultLimit = 12 * 1024
	readResultMaxLimit     = 24 * 1024

	// staleRunningAfter：running/queued meta 的 UpdatedAt 由 1s 进度 ticker
	// 刷新（起跑后）；超过该阈值且本进程无在跑句柄 = 大概率崩溃/强退遗留
	// 的假 running（排队中无 ticker，但排队期 subRuns 句柄在，归不到这里）。
	staleRunningAfter = 30 * time.Second
)

// ReadSubagentResultTool 按 ref 分页读取子代理运行的最终回答。
type ReadSubagentResultTool struct {
	store *SubagentStore
}

// NewReadSubagentResultTool 装配 read_subagent_result（boot 注册点调用，共享
// lazy store）。
func NewReadSubagentResultTool(store *SubagentStore) *ReadSubagentResultTool {
	return &ReadSubagentResultTool{store: store}
}

func (t *ReadSubagentResultTool) Name() string { return "read_subagent_result" }

func (t *ReadSubagentResultTool) Description() string {
	return "Read a persisted sub-agent run's final answer by sa_… ref (byte-paged). Use it when the inline task result has been dropped from context, or after a restart interrupted result delivery — a completed run can be recovered without re-running it."
}

func (t *ReadSubagentResultTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "ref":{"type":"string","description":"The sa_… subagent reference (from the task result's 'Subagent reference:' line or subagent_list)."},
  "offset_bytes":{"type":"integer","minimum":0,"description":"Byte offset into the answer for pagination. Omit for the first page."},
  "limit_bytes":{"type":"integer","minimum":1,"description":"Page size in bytes (default 12288, max 24576)."}
},
"required":["ref"]
}`)
}

// ReadOnly：纯读转录与 meta 侧车，无状态变更——折叠降噪适用。
func (t *ReadSubagentResultTool) ReadOnly() bool { return true }

func (t *ReadSubagentResultTool) CompactDescription() string {
	// CompactDescriptor 无条件替换 provider 面描述（模型只看这里）。
	return "按ref分页读取子代理运行(sa_…)的最终回答:task结果已滚出上下文或重启打断送达时取回,completed才能读;不必重跑(省一次推理)"
}

func (t *ReadSubagentResultTool) CompactSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"ref":{"type":"string"},"offset_bytes":{"type":"integer"},"limit_bytes":{"type":"integer"}},"required":["ref"]}`)
}

type readResultArgs struct {
	Ref         string `json:"ref"`
	OffsetBytes int64  `json:"offset_bytes"`
	LimitBytes  int64  `json:"limit_bytes"`
}

func (t *ReadSubagentResultTool) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var p readResultArgs
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid read_subagent_result arguments: %w", err)
	}
	ref := strings.TrimSpace(p.Ref)
	if ref == "" {
		return "", fmt.Errorf("ref is required")
	}
	if IsModelToolRef(ref) {
		// mt_ 的产出在调用当时已内联返回，转录只是过程记录。
		return "This is a model-tool run (mt_…); its output was returned inline at call time — there is no persisted final answer to read.", nil
	}
	if !IsSubagentRef(ref) {
		return "", fmt.Errorf("invalid subagent reference %q: must be an sa_… ref (see subagent_list)", ref)
	}

	meta, err := t.store.loadMeta(ref)
	if err != nil {
		// 不在当前 store 目录：换会话/换工作区后 ref 不可达是常态——诚实
		// 报错即可，重试无意义。
		return fmt.Sprintf("No subagent %s in this session's store — the ref may belong to another session or workspace (check subagent_list for what is reachable here).", ref), nil
	}

	switch meta.Status {
	case SubagentRunning, SubagentQueued:
		if _, live := subRuns.Load(ref); live {
			return fmt.Sprintf("Subagent %s is still %s in this process — wait for it to finish, or interrupt_agent if it should stop.", ref, meta.Status), nil
		}
		if time.Since(meta.UpdatedAt) > staleRunningAfter {
			return fmt.Sprintf("Subagent %s is marked %s but has no live handle in this process (last update %s ago) — it was most likely interrupted by a restart. Its partial transcript can be resumed with task continue_from.", ref, meta.Status, time.Since(meta.UpdatedAt).Truncate(time.Second)), nil
		}
		return fmt.Sprintf("Subagent %s is %s — wait for it to finish before reading its result.", ref, meta.Status), nil
	case SubagentCompleted, SubagentFailed:
		// 终态：继续走取回。
	default:
		return fmt.Sprintf("Subagent %s has unknown status %q; refusing to read.", ref, meta.Status), nil
	}

	path := t.store.transcriptPathUnlocked(ref)
	if path == "" {
		return fmt.Sprintf("No subagent %s in this session's store — the ref may belong to another session or workspace.", ref), nil
	}
	sess, err := session.Load(path)
	if err != nil {
		state := "failed"
		if meta.Status == SubagentCompleted {
			state = "completed"
		}
		return fmt.Sprintf("Subagent %s is %s but its transcript is missing or unreadable (%v).", ref, state, err), nil
	}
	answer := extractLastAssistantMessage(sess.Messages)
	if strings.TrimSpace(answer) == "" {
		if meta.Status == SubagentFailed {
			return fmt.Sprintf("Subagent %s failed and left no final answer in its transcript.", ref), nil
		}
		return fmt.Sprintf("Subagent %s completed but its transcript has no final answer text.", ref), nil
	}

	offset := p.OffsetBytes
	if offset < 0 {
		offset = 0
	}
	limit := p.LimitBytes
	if limit <= 0 {
		limit = readResultDefaultLimit
	}
	if limit > readResultMaxLimit {
		limit = readResultMaxLimit
	}
	data := []byte(answer)
	total := int64(len(data))
	if offset >= total {
		return fmt.Sprintf("offset_bytes=%d is beyond the end of the answer (%d bytes total) — the full answer was shown in earlier pages.", offset, total), nil
	}
	// 页界按 rune 对齐：起始向后落到 rune 起点（上游按字节切，多字节字符
	// 会被劈开——gaea 上下文以 CJK 为主，劈字符会产生乱码噪音）。
	if offset > 0 {
		for offset < total && !utf8.RuneStart(data[offset]) {
			offset++
		}
	}
	end := offset + limit
	if end > total {
		end = total
	} else {
		for end < total && !utf8.RuneStart(data[end]) {
			end++
		}
	}

	var sb strings.Builder
	if meta.Status == SubagentFailed {
		// 上游 partialTranscript 同款显式标注：partial 不是结果，依赖它的
		// 下游不得当完成态消费。
		sb.WriteString("PARTIAL RESULT — this run FAILED; the text below is the transcript tail, not a verified result.\n")
	}
	fmt.Fprintf(&sb, "<subagent-result ref=%q status=%q bytes=\"%d/%d\">\n", ref, meta.Status, end-offset, total)
	sb.Write(data[offset:end])
	sb.WriteString("\n</subagent-result>")
	if end < total {
		fmt.Fprintf(&sb, "\n(continued: call again with offset_bytes=%d; %d bytes remain)", end, total-end)
	}
	return sb.String(), nil
}
