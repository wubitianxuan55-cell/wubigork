package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"unsafe"

	"github.com/gaea/gaea/internal/gaea/provider"
)

// ── 上下文编译摘要（v4.212，阶段五 5.2 首刀）──────────────────────────
//
// CacheShape 只覆盖请求头（system+tools）；这里补齐消息历史这一半：把每条
// 消息规约成 canonical 字节串取摘要，得到「逐消息哈希链」，相邻两次请求
// 一比即得前缀稳定判定——
//
//	Stable       上一请求消息序列是本请求的无改写前缀（RewriteCount==0）
//	PrefixBytes  稳定公共前缀的字节数
//	AppendCount  稳定前缀之后的新增消息数
//	RewriteCount 上一请求中被改写/移除的消息数（压缩/rewind/重排 >0）
//
// 判定随 RequestHeader 事件落会话日志（dump 可回放），与 usage 的
// CacheHitTokens 配对即「相邻两轮前缀字节级稳定 + 缓存命中可证」的完整
// 证据链（5.2 出口判据）。纯函数、无 IO、无时钟——确定性可测。

// MsgDigest 是一条消息的编译摘要。
type MsgDigest struct {
	Hash  string // canonical 形态的 SHA256 前 16 hex
	Bytes int    // canonical 字节数
}

// digestMessage 把消息规约为 canonical 字节串并取摘要。分隔符用长度前缀
// 思维（role 分隔 + 内容长度声明），杜绝「A|BC == AB|C」的拼接歧义。
func digestMessage(m provider.Message) MsgDigest {
	var b []byte
	b = append(b, m.Role...)
	b = append(b, 0x1f)
	b = append(b, strconv.Itoa(len(m.Content))...)
	b = append(b, 0x1f)
	b = append(b, m.Content...)
	// tool_calls（助手消息携带工具调用时参与摘要；nil/空不影响）。
	if len(m.ToolCalls) > 0 {
		b = append(b, 0x1f)
		for _, tc := range m.ToolCalls {
			b = append(b, tc.ID...)
			b = append(b, 0x1e)
			b = append(b, tc.Name...)
			b = append(b, 0x1e)
			b = append(b, tc.Arguments...)
			b = append(b, 0x1d)
		}
	}
	sum := sha256.Sum256(b)
	return MsgDigest{Hash: hex.EncodeToString(sum[:8]), Bytes: len(b)}
}

// DigestMessages 摘要整条消息序列。
func DigestMessages(msgs []provider.Message) []MsgDigest {
	out := make([]MsgDigest, len(msgs))
	for i, m := range msgs {
		out[i] = digestMessage(m)
	}
	return out
}

// ── 增量摘要缓存（刀A v4.245）──────────────────────────────────────
//
// DigestMessages 每次请求对全部历史消息重哈希（每步 O(会话总字节)）。相邻
// 请求间消息序列通常只追加不改写——DigestCache 以指针级身份核对前缀，未变
// 消息复用上一请求摘要，仅对新增/改写消息重哈希。
//
// 身份判定的正确性依据：Go 字符串不可变，数据指针+长度相等 ⇒ 字节内容相等
// （两份存活字符串不可能部分重叠）；切片同理（同一底层数组 ⇒ 同元素）。
// ToolCallID 等不在 digestMessage 输入里的字段不参与身份。身份不符只意味着
// 「须重哈希」，绝不据此判定内容不同——保守方向，前缀稳定证据链只会重算、
// 不会错判。

// DigestCache 是相邻请求间的逐消息摘要缓存（AgentRunner 每实例一个，
// 非并发安全——stream 链单调用方）。零值可用。
type DigestCache struct {
	msgs    []provider.Message
	digests []MsgDigest
	hits    int // 最近一次 Digest 复用的前缀消息数（测试/观测用）
}

// Digest 摘要消息序列：与 DigestMessages 逐元素等价，前缀身份相符的
// 部分复用上次结果。
func (c *DigestCache) Digest(msgs []provider.Message) []MsgDigest {
	k := 0
	for k < len(c.msgs) && k < len(msgs) && sameDigestIdentity(c.msgs[k], msgs[k]) {
		k++
	}
	c.hits = k
	out := make([]MsgDigest, len(msgs))
	copy(out, c.digests[:k])
	for i := k; i < len(msgs); i++ {
		out[i] = digestMessage(msgs[i])
	}
	c.msgs = msgs
	c.digests = out
	return out
}

// sameDigestIdentity 报告两条消息的摘要相关输入是否同一份数据。
func sameDigestIdentity(a, b provider.Message) bool {
	if a.Role != b.Role || !sameStringIdentity(a.Content, b.Content) {
		return false
	}
	if len(a.ToolCalls) != len(b.ToolCalls) {
		return false
	}
	if len(a.ToolCalls) == 0 {
		return true
	}
	return unsafe.SliceData(a.ToolCalls) == unsafe.SliceData(b.ToolCalls)
}

// sameStringIdentity 报告两个字符串是否同一份数据（指针+长度）。
func sameStringIdentity(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	return unsafe.StringData(a) == unsafe.StringData(b)
}

// SequenceHash 是整条序列的整体摘要（相邻请求等值即整体未变）。
func SequenceHash(ds []MsgDigest) string {
	var b []byte
	for _, d := range ds {
		b = append(b, d.Hash...)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8])
}

// CompileDiff 是相邻两请求的前缀稳定判定（5.2 出口判据的承担者）。
type CompileDiff struct {
	Hash         string
	PrevHash     string
	Stable       bool
	PrefixBytes  int64
	AppendCount  int
	RewriteCount int
}

// DiffMessages 比较 prev → curr 两条消息序列。消息级判定即字节级：消息在
// 请求里是离散对象，prev[:k] 逐条相等且位置相同 ⇒ 序列化后的 JSON 数组
// 前缀逐字节稳定。
func DiffMessages(prev, curr []MsgDigest) CompileDiff {
	d := CompileDiff{
		Hash:     SequenceHash(curr),
		PrevHash: SequenceHash(prev),
	}
	k := 0
	for k < len(prev) && k < len(curr) && prev[k].Hash == curr[k].Hash {
		d.PrefixBytes += int64(curr[k].Bytes)
		k++
	}
	d.RewriteCount = len(prev) - k
	d.AppendCount = len(curr) - k
	d.Stable = d.RewriteCount == 0 && len(prev) > 0
	return d
}
