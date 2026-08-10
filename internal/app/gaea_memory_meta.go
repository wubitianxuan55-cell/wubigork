package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/gaea/memory"
	"github.com/gaea/gaea/internal/gaea/textsim"
)

// MemoryDuplicateView 是一组疑似重复的办公记忆事实（keep 为建议保留项）。
type MemoryDuplicateView struct {
	Keep      string  `json:"keep"`
	KeepTitle string  `json:"keepTitle"`
	Dup       string  `json:"dup"`
	DupTitle  string  `json:"dupTitle"`
	Score     float64 `json:"score"`
}

// GaeaMemoryDuplicates 返回办公记忆中疑似重复的事实对（相似度 ≥ min，
// 默认 0.55），按相似度降序；每对建议保留较早创建/名称靠前的一项。
func (a *App) GaeaMemoryDuplicates(min float64) []MemoryDuplicateView {
	if min <= 0 {
		min = 0.55
	}
	facts := a.hubOfficeStore().List()
	sort.Slice(facts, func(i, j int) bool { return facts[i].Name < facts[j].Name })
	var out []MemoryDuplicateView
	for i := 0; i < len(facts); i++ {
		for j := i + 1; j < len(facts); j++ {
			score := memoryFactSimilarity(facts[i], facts[j])
			if score >= min {
				out = append(out, MemoryDuplicateView{
					Keep: facts[i].Name, KeepTitle: facts[i].Title,
					Dup: facts[j].Name, DupTitle: facts[j].Title,
					Score: score,
				})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

// GaeaMemoryMerge 把 sourceNames 合并进 targetName：标签取并集，来源事实的
// 描述/正文追加到目标正文（带「合并自」标记），来源删除；返回目标名。
func (a *App) GaeaMemoryMerge(targetName string, sourceNames []string) (string, error) {
	store := a.hubOfficeStore()
	all := store.List()
	byName := make(map[string]memory.Memory, len(all))
	for _, m := range all {
		byName[m.Name] = m
	}
	target, ok := byName[targetName]
	if !ok {
		return "", fmt.Errorf("记忆事实不存在: %s", targetName)
	}
	tagSet := make(map[string]bool, len(target.Tags))
	for _, t := range target.Tags {
		tagSet[t] = true
	}
	var merged []string
	for _, sn := range sourceNames {
		if sn == "" || sn == targetName {
			continue
		}
		src, ok := byName[sn]
		if !ok {
			continue
		}
		for _, t := range src.Tags {
			tagSet[t] = true
		}
		merged = append(merged, fmt.Sprintf("- 合并自「%s」：%s", src.Title, strings.TrimSpace(src.Description)))
		if err := store.Delete(sn); err != nil {
			return "", err
		}
	}
	if len(merged) == 0 {
		return target.Name, nil
	}
	target.Tags = sortedMemoryTags(tagSet)
	if strings.TrimSpace(target.Body) != "" {
		target.Body += "\n"
	}
	target.Body += strings.Join(merged, "\n")
	if _, err := store.Save(target); err != nil {
		return "", err
	}
	return target.Name, nil
}

func memoryFactSimilarity(a, b memory.Memory) float64 {
	ta := strings.TrimSpace(a.Description + " " + a.Body)
	tb := strings.TrimSpace(b.Description + " " + b.Body)
	s := textsim.Similarity(ta, tb)
	// 名称本身高度相似也加分（自动做梦常产生同名变体）。
	if s < 0.99 {
		ns := textsim.Similarity(a.Title, b.Title)
		if ns > s {
			s = ns
		}
	}
	return s
}

func sortedMemoryTags(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for t := range set {
		if strings.TrimSpace(t) != "" {
			out = append(out, t)
		}
	}
	sort.Strings(out)
	return out
}
