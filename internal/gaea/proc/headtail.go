package proc

import "fmt"

// 头尾保留缓冲（codex 蒸馏 T13，head_tail_buffer 同构）：不是简单截尾部——
// 输出的头部常是命令回显/环境头，尾部是错误与结果，两头都重要，中间往往是
// 重复的进度条。50/50 对称：超限丢中间，并明确写「省略了多少字节」，
// 模型才不会误以为输出完整。pending 与 transcript 都要有上限，不能只截一份
// （否则长命令的 transcript 吃满内存）。

// 省略标记格式（唯一、可断言）。
func omittedMarker(n int64) string {
	return fmt.Sprintf("... %d bytes omitted ...", n)
}

// HeadTail 是 50/50 头尾保留缓冲：head 存稳定前缀，tail 存最近后缀，
// 中间超限就丢并计数。零值不可用，经 NewHeadTail 构造。
type HeadTail struct {
	maxBytes int
	head     []byte
	tail     []byte
	headFull bool
	omitted  int64
	total    int64
}

// NewHeadTail 构造容量 maxBytes 的缓冲（<=0 视为 0=不保留，仅计数）。
func NewHeadTail(maxBytes int) *HeadTail {
	if maxBytes < 0 {
		maxBytes = 0
	}
	return &HeadTail{maxBytes: maxBytes}
}

// Write 追加字节：head 未满则进 head；满后进 tail（超限从前端挤掉并计省略）。
func (h *HeadTail) Write(p []byte) {
	h.total += int64(len(p))
	half := h.maxBytes / 2
	if !h.headFull {
		space := half - len(h.head)
		if space >= len(p) {
			h.head = append(h.head, p...)
			return
		}
		if space > 0 {
			h.head = append(h.head, p[:space]...)
			p = p[space:]
		}
		h.headFull = true
		if len(p) == 0 {
			return
		}
	}
	// tail 环：保留最近 half 字节，前端挤掉的计入省略。
	h.tail = append(h.tail, p...)
	if over := len(h.tail) - half; over > 0 {
		h.tail = h.tail[over:]
		h.omitted += int64(over)
	}
}

// OmittedBytes 返回被省略（丢弃）的字节数。
func (h *HeadTail) OmittedBytes() int64 { return h.omitted }

// Size 返回当前实际持有的字节数（head+tail，不含省略部分）。
func (h *HeadTail) Size() int { return len(h.head) + len(h.tail) }

// Transcript 渲染完整记录：head + 省略标记（仅当有省略）+ tail。
// 超限时标记恰好一个，字节数 = 实际丢弃量（可断言）。
func (h *HeadTail) Transcript() []byte {
	if h.omitted == 0 {
		out := make([]byte, 0, len(h.head)+len(h.tail))
		out = append(out, h.head...)
		out = append(out, h.tail...)
		return out
	}
	marker := []byte(omittedMarker(h.omitted))
	out := make([]byte, 0, len(h.head)+len(marker)+len(h.tail)+2)
	out = append(out, h.head...)
	out = append(out, '\n')
	out = append(out, marker...)
	out = append(out, '\n')
	out = append(out, h.tail...)
	return out
}
