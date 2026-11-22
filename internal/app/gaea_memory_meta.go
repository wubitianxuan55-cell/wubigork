package app

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

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
	// Source 候选来源（Nebula 蒸馏 P3，additive）：fuzzy=词法 textsim 臂；
	// vector=语义向量余弦臂（≥0.85，embedder 不可用时该臂静默缺席）。
	Source string `json:"source,omitempty"`
}

// GaeaMemoryDuplicates 返回办公记忆中疑似重复的事实对（相似度 ≥ min，
// 默认 0.55），按相似度降序；每对建议保留较早创建/名称靠前的一项。
//
// 与 memory.DistillMergeCandidates 是两套口径（审计 AP5-06 现场钉死，非同源
// 复制，行为冻结中）：本函数=模糊相似度（textsim 对 描述+正文/标题 ≥ 阈值
// 即报，人审建议面板，不设上限、不分区 space、保留方向=名称靠前）；
// DistillMergeCandidates=确定性检测（归一化同名异写/同类型同描述逐字相等，
// 仅同 space 内成对，上限 8，保留方向=UpdatedAt 较新），其「宁漏勿误」纪律
// 刻意拒绝模糊相似度——误合并记忆是不可逆伤害，故两套输出必然有交集之外
// 的互斥区（近似文面板报/蒸馏不报；同名异写两套都报但保留方向可相反）。
// 对同输入的差异已由 gaea_memory_meta_compare_test.go 钉成机器可见事实；
// 是否把本函数收敛为 DistillMergeCandidates 的映射（改绑定输出）或从绑定面
// 移除（改绑定面）在拍板池，未拍板前维持现状。
func (a *App) GaeaMemoryDuplicates(min float64) []MemoryDuplicateView {
	if min <= 0 {
		min = 0.55
	}
	facts := a.hubOfficeStore().List()
	sort.Slice(facts, func(i, j int) bool { return facts[i].Name < facts[j].Name })
	seen := map[string]bool{}
	var out []MemoryDuplicateView
	add := func(keep, keepTitle, dup, dupTitle string, score float64, source string) {
		lo, hi := keep, dup
		if lo > hi {
			lo, hi = hi, lo
		}
		key := lo + "\x00" + hi
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, MemoryDuplicateView{
			Keep: keep, KeepTitle: keepTitle,
			Dup: dup, DupTitle: dupTitle,
			Score: score, Source: source,
		})
	}
	for i := 0; i < len(facts); i++ {
		for j := i + 1; j < len(facts); j++ {
			score := memoryFactSimilarity(facts[i], facts[j])
			if score >= min {
				add(facts[i].Name, facts[i].Title, facts[j].Name, facts[j].Title, score, "fuzzy")
			}
		}
	}
	// 向量臂（Nebula 蒸馏 P3）：语义余弦 ≥0.85 的同义对（词法臂抓不到的换
	// 说法重复）。embedder/索引不可用或规模超限→该臂缺席（不报错不假装）。
	for _, p := range a.vectorConsolidatePairs() {
		ka, kOK := factsByName(facts, p.A)
		kb, k2OK := factsByName(facts, p.B)
		if !kOK || !k2OK {
			continue
		}
		add(ka.Name, ka.Title, kb.Name, kb.Title, p.Score, "vector")
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

// vectorConsolidatePairs 向量臂装配：office 向量 Ensure（与星云召回同缓存
// 口径）→ VectorsOf 快照 → memory.VectorSimilarPairs（0.85 / 20000 对 / 每名 8）。
func (a *App) vectorConsolidatePairs() []memory.SimilarPair {
	// nil core（裸 App/测试构造）诚实缺席：向量臂是增强项，不得绑架词法面
	// （engineMgr 走 core 内嵌指针，nil core 即崩——app 测试语义管线地雷）。
	if a.core == nil {
		return nil
	}
	e := a.localSearchEmbedder()
	if e == nil {
		return nil
	}
	st := a.hubSemanticStore()
	if st == nil || !st.Available() {
		return nil
	}
	mems := a.hubOfficeStore().List()
	if len(mems) == 0 {
		return nil
	}
	// O(n²) 面板动作规模护栏（800²=32 万对量级，秒内）；超限如实缺席。
	if len(mems) > 800 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := a.ensureOfficeVectorIndex(ctx, e, st, mems); err != nil {
		slog.Warn("向量巩固臂索引失败（该臂缺席）", "error", err)
		return nil
	}
	vecs, err := st.VectorsOf("office")
	if err != nil {
		slog.Warn("向量巩固臂取数失败（该臂缺席）", "error", err)
		return nil
	}
	names := make([]string, 0, len(mems))
	for _, m := range mems {
		names = append(names, m.Name)
	}
	return memory.VectorSimilarPairs(names, vecs, memory.VectorConsolidateMinCosine, memory.VectorConsolidateMaxPairs, memory.VectorConsolidatePerNameCap)
}

// factsByName 按名取事实（未命中返回零值+false）。
func factsByName(facts []memory.Memory, name string) (memory.Memory, bool) {
	for _, m := range facts {
		if m.Name == name {
			return m, true
		}
	}
	return memory.Memory{}, false
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
