package memory

import (
	"math"
	"sort"
)

// ── 向量巩固聚类（Nebula Memory 蒸馏 P3）─────────────────────────────────
//
// Nebula 的 consolidate：相似 episode（tf-idf 余弦 ≥0.85）聚成 2..12 的簇 →
// fact 摘要；候选对上限 20000 防 O(n²)。gaea 映射：不自动写任何记忆（评审制
// 红线），而是把「语义向量 ≥0.85 的疑似同义对」报进既有 duplicates 评审面板
// （词法臂 textsim 之外的第二来源）——人审 accept，复用既有 Merge 流。
// 簇不显式返回：对即簇的传递闭包投影，面板按对渲染（2..12 簇界由每名
// perNameCap 自然收敛，巨簇不刷屏）。

// VectorConsolidateMinCosine 是向量臂成对阈值（蒸馏值）：Nebula consolidate
// 同款 0.85——「语义同义」的保守下限，宁漏勿误（误合并记忆不可逆）。
const VectorConsolidateMinCosine = 0.85

// VectorConsolidateMaxPairs 是向量臂候选对上限（蒸馏值 20000）：早停护栏。
const VectorConsolidateMaxPairs = 20000

// VectorConsolidatePerNameCap 是单条记忆最多参与的对数（巨簇收敛护栏，8）。
const VectorConsolidatePerNameCap = 8

// SimilarPair 是一对语义相似的记忆（Score=余弦 ∈ [0,1]）。
type SimilarPair struct {
	A     string
	B     string
	Score float64
}

// VectorSimilarPairs 纯函数：names 两两余弦 ≥minCosine 的对（Score 降序 →
// 字典序，确定性）。vecs 缺向量的名字跳过；单名参与对数超 perNameCap 后不再
// 产出（巨簇收敛）；候选达 maxPairs 早停。调用方守 O(n²) 规模（面板动作，
// 建议 ≤800 条；超限由调用方如实降级）。
func VectorSimilarPairs(names []string, vecs map[string][]float32, minCosine float64, maxPairs, perNameCap int) []SimilarPair {
	if minCosine <= 0 {
		minCosine = VectorConsolidateMinCosine
	}
	if maxPairs <= 0 {
		maxPairs = VectorConsolidateMaxPairs
	}
	if perNameCap <= 0 {
		perNameCap = VectorConsolidatePerNameCap
	}
	sorted := make([]string, 0, len(names))
	for _, n := range names {
		if n != "" && len(vecs[n]) > 0 {
			sorted = append(sorted, n)
		}
	}
	sort.Strings(sorted)
	// 单位化一次（余弦=点积），零向量跳过。
	units := make([]struct {
		name string
		v    []float32
	}, 0, len(sorted))
	for _, n := range sorted {
		u := unitVec(vecs[n])
		if u != nil {
			units = append(units, struct {
				name string
				v    []float32
			}{n, u})
		}
	}
	counts := make(map[string]int, len(units))
	out := make([]SimilarPair, 0, 64)
	for i := 0; i < len(units); i++ {
		if counts[units[i].name] >= perNameCap {
			continue
		}
		for j := i + 1; j < len(units); j++ {
			if counts[units[j].name] >= perNameCap {
				continue
			}
			cos := dotUnit(units[i].v, units[j].v)
			if cos < minCosine {
				continue
			}
			out = append(out, SimilarPair{A: units[i].name, B: units[j].name, Score: cos})
			counts[units[i].name]++
			counts[units[j].name]++
			if len(out) >= maxPairs {
				sortSimilarPairs(out)
				return out
			}
		}
	}
	sortSimilarPairs(out)
	return out
}

func sortSimilarPairs(out []SimilarPair) {
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		if out[i].A != out[j].A {
			return out[i].A < out[j].A
		}
		return out[i].B < out[j].B
	})
}

// unitVec 返回单位化副本；零向量/空向量返回 nil。
func unitVec(v []float32) []float32 {
	norm := 0.0
	for _, x := range v {
		norm += float64(x) * float64(x)
	}
	if norm <= 0 {
		return nil
	}
	inv := 1 / math.Sqrt(norm)
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = float32(float64(x) * inv)
	}
	return out
}

// dotUnit 两个已单位化向量的点积（余弦），下限夹 0。
func dotUnit(a, b []float32) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	sum := 0.0
	for i := 0; i < n; i++ {
		sum += float64(a[i]) * float64(b[i])
	}
	if sum < 0 {
		return 0
	}
	return sum
}
