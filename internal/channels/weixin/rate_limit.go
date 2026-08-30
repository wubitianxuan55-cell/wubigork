// Package weixin — 入站 per-peer 滑动窗口限频（v4.8 子项 d①）
package weixin

import (
	"sync"
	"time"
)

// rateLimiter 是 per-peer 滑动窗口限频器：每个发送者在 window 内最多放行
// limit 条消息，超限拒绝（handle 层发固定文案且不触发 LLM）。时钟可注入
// （clock 字段，测试推进时间用）；零值不可用，须经 newRateLimiter 构造。
// peers 记录在条目被访问时就地清理窗口外时间戳；gaea 为个人桌面单用户场景，
// 不做全局过期回收。
type rateLimiter struct {
	mu     sync.Mutex
	window time.Duration
	limit  int
	// clock 可替换的时间源（测试注入）；nil 时回退 time.Now。
	clock func() time.Time
	peers map[string][]time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		window: window,
		limit:  limit,
		clock:  time.Now,
		peers:  make(map[string][]time.Time),
	}
}

func (r *rateLimiter) now() time.Time {
	if r.clock != nil {
		return r.clock()
	}
	return time.Now()
}

// Allow 记录 peer 的一次到来并报告是否放行（滑动窗口：now-window 之外的
// 历史不计入）。
func (r *rateLimiter) Allow(peer string) bool {
	now := r.now()
	r.mu.Lock()
	defer r.mu.Unlock()
	hits := r.peers[peer]
	kept := hits[:0] // 复用底层数组就地过滤（写入索引恒 ≤ 读取索引，安全）
	for _, ts := range hits {
		if now.Sub(ts) < r.window {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= r.limit {
		r.peers[peer] = kept
		return false
	}
	r.peers[peer] = append(kept, now)
	return true
}
