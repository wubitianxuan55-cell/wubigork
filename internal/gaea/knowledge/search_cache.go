package knowledge

import (
	"sync"

	gaesearch "github.com/gaea/gaea/internal/gaea/search"
)

// tfidfCache 缓存 Search 用的 TF-IDF 索引，避免每次查询对全部候选重建
// （对齐 memory 包"索引建一次、写路径失效"的做法）。
//
// 单槽设计：同一过滤签名+候选指纹的重复查询直接命中；Save/Delete 经
// invalidate bump 代数失效；候选内容指纹兜底捕获绕过 Store 的带外改动
// （文件后端每次 ReadAll 直读磁盘，旧实现总能看到这类改动，不能因缓存丢失）。
// 交替使用不同过滤条件时退化为每次重建 —— 与旧实现成本相同，不会更差。
type tfidfCache struct {
	mu     sync.Mutex
	gen    uint64 // 写路径失效代数：Save/Delete 各 bump 一次
	builds uint64 // 索引实际构建次数（测试可观测）
	sig    string // 缓存槽：过滤条件签名
	genAt  uint64 // 缓存槽构建时的 gen
	fp     uint64 // 缓存槽构建时的候选内容指纹
	idx    *gaesearch.TfidfIndex
}

// invalidate bumps the generation so the next Search rebuilds the index.
func (c *tfidfCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gen++
}

// get 返回仍然有效的缓存索引；过期或未缓存时返回 nil。
func (c *tfidfCache) get(sig string, fp uint64) *gaesearch.TfidfIndex {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.idx != nil && c.gen == c.genAt && c.sig == sig && c.fp == fp {
		return c.idx
	}
	return nil
}

// put 存入新构建的索引。并发 Search 可能用相同内容互相覆盖 —— 良性竞争，
// Build 对相同 docs 是确定性的，索引等价。
func (c *tfidfCache) put(sig string, fp uint64, idx *gaesearch.TfidfIndex) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sig, c.fp, c.idx = sig, fp, idx
	c.genAt = c.gen
	c.builds++
}

// buildsCount 返回索引构建次数（测试断言用，无计时断言的脆弱性）。
func (c *tfidfCache) buildsCount() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.builds
}

// searchFilterSig 把 Filter 渲染成缓存键。长度前缀消除字段边界歧义；
// 即便残留碰撞也无害 —— 指纹校验会在候选集实际不同时强制重建。
func searchFilterSig(f Filter) string {
	return lenPrefix(f.Category) + lenPrefix(f.Phase) + lenPrefix(f.Tag) + lenPrefix(f.Status)
}

func lenPrefix(s string) string {
	return string(rune(len(s))) + s
}

// candidatesFingerprint 对索引的真实输入（候选集及其顺序、每条的
// Name/Title/Category/Body —— 即 Build 消费的 docs 内容）做 FNV-1a 指纹。
// O(字节数)、零分配，相对逐条分词建 bigram 表的开销可忽略。
// 注意 Phase/Tags/Status 不参与索引，故不入指纹：它们变化时 TF-IDF 部分
// 仍然有效（关键词打分每次都基于最新候选重算）。
func candidatesFingerprint(candidates []Entry) uint64 {
	const (
		offset64 = uint64(14695981039346656037)
		prime64  = uint64(1099511628211)
	)
	h := offset64
	mix := func(b byte) { h ^= uint64(b); h *= prime64 }
	mixInt := func(n int) {
		for i := 0; i < 8; i++ {
			mix(byte(n))
			n >>= 8
		}
	}
	mixStr := func(s string) {
		mixInt(len(s))
		for i := 0; i < len(s); i++ {
			mix(s[i])
		}
	}
	mixInt(len(candidates))
	for _, e := range candidates {
		mixStr(e.Name)
		mixStr(e.Title)
		mixStr(e.Category)
		mixStr(e.Body)
	}
	return h
}

// tfidfIndexFor 返回给定候选集的 TF-IDF 索引：缓存有效则复用，否则构建并
// 缓存。docs 构造与旧的逐查询内联实现逐字节一致，因此分数与排序零变化。
func (s *Store) tfidfIndexFor(candidates []Entry, filter Filter) *gaesearch.TfidfIndex {
	sig := searchFilterSig(filter)
	fp := candidatesFingerprint(candidates)
	if idx := s.tfidf.get(sig, fp); idx != nil {
		return idx
	}
	docs := make([]gaesearch.Doc, 0, len(candidates))
	for _, e := range candidates {
		docs = append(docs, gaesearch.Doc{ID: e.Name, Text: e.Title + " " + e.Category + " " + e.Body})
	}
	idx := gaesearch.NewTfidfIndex()
	idx.Build(docs)
	s.tfidf.put(sig, fp, idx)
	return idx
}
