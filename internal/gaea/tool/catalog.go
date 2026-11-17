package tool

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// 目录差分（codex 蒸馏 T8 三层之二，top_level_tools 同构）：工具面省 token
// 靠「目录只发差分、检索才给全量、描述按预算裁」——预算（budget.go）只是兜底。
// 快照是 name → 定义哈希：哈希相同 = 零输出（即使内部重建），模型看到的
// 工具列表是累积状态，每次只追加变化。
//
// 三条纪律（照抄上游）：
//  1. 一个增量批次只加一条 developer 通知，不是每个工具一条；
//  2. 定义没变就不重发（catalog hashes are unchanged）；
//  3. 整命名空间移除时吞并其成员——删一个命名空间不产生 N 条删除通知。

// Catalog 是目录差分的状态侧：持有一份 name→hash 快照，Diff 与新快照比较。
type Catalog struct {
	hashes map[string]string
}

// NewCatalog 构造空目录（首个快照即全量 added）。
func NewCatalog() *Catalog {
	return &Catalog{hashes: map[string]string{}}
}

// CatalogHash 对定义 JSON 取哈希（canonical schema 即可，编码稳定）。
func CatalogHash(def json.RawMessage) string {
	sum := sha256.Sum256(def)
	return hex.EncodeToString(sum[:])
}

// Diff 比较新快照：返回 added/removed（各自稳定排序）与 changed。
// 调用方应把返回的 added schemas + 一条通知放进上下文；changed=false 时
// 工具部分上下文与上轮逐字节相同（前缀缓存友好）。
func (c *Catalog) Diff(next map[string]string) (added, removed []string, changed bool) {
	for name, h := range next {
		if old, ok := c.hashes[name]; !ok || old != h {
			added = append(added, name)
			changed = true
		}
	}
	for name := range c.hashes {
		if _, ok := next[name]; !ok {
			removed = append(removed, name)
			changed = true
		}
	}
	sort.Strings(added)
	sort.Strings(removed)
	if changed {
		c.hashes = next
	}
	return added, removed, changed
}

// CoalesceRemoved 把删除列表按命名空间吞并：某命名空间的全部成员都被删时
// 只报命名空间一条（纪律 3）。nsOf 返回工具的命名空间标识。
func CoalesceRemoved(removed []string, nsOf func(string) string) []string {
	if len(removed) == 0 || nsOf == nil {
		return removed
	}
	// 统计每个命名空间在删除列表里的成员数。
	nsCount := map[string]int{}
	for _, name := range removed {
		nsCount[nsOf(name)]++
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(removed))
	for _, name := range removed {
		ns := nsOf(name)
		if ns == "" || nsCount[ns] <= 1 {
			out = append(out, name)
			continue
		}
		// 多成员命名空间：首个成员位置放「ns/*」，其余吞并。
		if !seen["ns:"+ns] {
			seen["ns:"+ns] = true
			out = append(out, ns+"/*")
		}
	}
	sort.Strings(out)
	return out
}
