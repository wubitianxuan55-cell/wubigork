package memory

// 做梦 2.0 第一刀「蒸馏真实合并」（路线图 T0：no-op 转真实合并）的确定性
// 检测层。自动做梦按 name upsert——同名会更新，但**近似重名/重复描述**的
// 事实会永久累积（「用户-时区」与「用户时区」并存）。本文件只做检测：
//   - 规则 A：归一化同名异写（连字符/下划线/点/大小写差异）；
//   - 规则 B：同 type+kind 且描述逐字相同（重复沉淀）。
// 合并执行在 control.DistillMerge（归档较旧条 + Touch 较新条，锁内重算校验）。
// 纪律：宁漏勿误——只报确定性重复，不做模糊相似度（误合并记忆是不可逆伤害）；
// 跨空间的同名/同描述**不**构成候选（双空间红线：合并绝不跨空间）。

import (
	"sort"
	"strings"
)

// MergeCandidate 是一条蒸馏合并建议：保留 Keep（较新），归档 Archive（较旧，
// 经 Store.Archive 可逆）。
type MergeCandidate struct {
	Keep    string
	Archive string
	Reason  string
}

// distillMaxMerges 候选上限（面板一屏可读；超量按保留名排序截断）。
const distillMaxMerges = 8

// DistillNormalizeName 记忆名归一：小写 + 去分隔符（- _ 空格 .）。
func DistillNormalizeName(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	return strings.NewReplacer("-", "", "_", "", " ", "", ".", "").Replace(n)
}

// DistillMergeCandidates 从记忆列表检测确定性重复。纯函数：同输入同输出
// （组内按 UpdatedAt 降序取保留条，稳定排序）。
func DistillMergeCandidates(ms []Memory) []MergeCandidate {
	type key struct{ norm, space string }
	bySlug := map[key][]Memory{}
	byTypeDesc := map[string][]Memory{}
	for _, m := range ms {
		if strings.TrimSpace(m.Name) == "" {
			continue
		}
		if norm := DistillNormalizeName(m.Name); norm != "" {
			bySlug[key{norm, m.Space}] = append(bySlug[key{norm, m.Space}], m)
		}
		if desc := strings.TrimSpace(m.Description); desc != "" {
			k := strings.ToLower(desc) + "\x00" + string(m.Type) + "/" + string(m.Kind) + "\x00" + m.Space
			byTypeDesc[k] = append(byTypeDesc[k], m)
		}
	}

	seen := map[string]bool{}
	var candidates []MergeCandidate
	pick := func(group []Memory, reason string) {
		if len(group) < 2 {
			return
		}
		sorted := append([]Memory(nil), group...)
		sort.SliceStable(sorted, func(i, j int) bool {
			return sorted[i].UpdatedAt.After(sorted[j].UpdatedAt)
		})
		keep := sorted[0]
		for _, old := range sorted[1:] {
			if seen[keep.Name] || seen[old.Name] {
				continue
			}
			seen[keep.Name] = true
			seen[old.Name] = true
			candidates = append(candidates, MergeCandidate{Keep: keep.Name, Archive: old.Name, Reason: reason})
		}
	}

	for _, group := range bySlug {
		pick(group, "同名异写（连字符/大小写差异，归一化后同名）")
	}
	for _, group := range byTypeDesc {
		pick(group, "同类型同描述（重复沉淀）")
	}

	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].Keep < candidates[j].Keep })
	if len(candidates) > distillMaxMerges {
		candidates = candidates[:distillMaxMerges]
	}
	return candidates
}
