// Package whisper — findings_merge.go
// 100% 对齐 ackem desktop-agent/investigation/findingsMerge.ts
// 发现合并：source优先级去重 + 名称联合匹配 + 置信度择优

package whisper

import "strings"

// ─── Source 优先级 ──────────────────────────────────────────────

// sourceRank 返回来源的优先级（越高越可靠）
func sourceRank(source string) int {
	switch strings.ToLower(source) {
	case "steam_common", "steam":
		return 10
	case "epic_manifest", "epic":
		return 9
	case "gog":
		return 8
	case "battle_net":
		return 7
	case "program_files", "program_files_x86":
		return 6
	case "start_menu":
		return 5
	case "local_programs":
		return 4
	case "desktop", "shortcut":
		return 3
	case "downloads":
		return 2
	case "heuristic":
		return 1
	default:
		return 0
	}
}

// ─── 增强合并 ──────────────────────────────────────────────────

// MergeFindingsEnhanced 增强版发现合并：source优先级 + 名称去重
// 100% 对齐 ackem findingsMerge.ts mergeGameFindings
func MergeFindingsEnhanced(findings []InvestigationFinding) []InvestigationFinding {
	if len(findings) <= 1 {
		return findings
	}

	// 按名称分组
	groups := groupByName(findings)

	var merged []InvestigationFinding
	for _, group := range groups {
		if len(group) == 1 {
			merged = append(merged, group[0])
			continue
		}
		// 多源冲突 → 选最高优先级
		best := pickBest(group)
		merged = append(merged, best)
	}

	return merged
}

// groupByName 按规范化名称分组
func groupByName(findings []InvestigationFinding) map[string][]InvestigationFinding {
	groups := make(map[string][]InvestigationFinding)
	for _, f := range findings {
		key := normalizeName(f.Name)
		if key == "" {
			key = strings.ToLower(f.Path)
		}
		if key == "" {
			continue
		}
		groups[key] = append(groups[key], f)
	}
	return groups
}

// normalizeName 规范化名称用于分组匹配
func normalizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	// 去掉版本后缀 " (2023)", " v.1.0" 等
	name = strings.TrimSuffix(name, " edition")
	name = strings.TrimSuffix(name, " version")
	// 去掉常见后缀
	suffixes := []string{"™", "®", "©", "：", ":"}
	for _, s := range suffixes {
		name = strings.TrimSuffix(name, s)
	}
	name = strings.TrimSpace(name)
	return name
}

// pickBest 从多个同源发现中选最优
func pickBest(group []InvestigationFinding) InvestigationFinding {
	if len(group) == 0 {
		return InvestigationFinding{}
	}
	if len(group) == 1 {
		return group[0]
	}

	best := group[0]
	bestRank := sourceRank(best.Source)
	bestNameLen := len([]rune(best.Name))

	for _, f := range group[1:] {
		rank := sourceRank(f.Source)
		nameLen := len([]rune(f.Name))

		// 更可靠的来源优先
		if rank > bestRank {
			best = f
			bestRank = rank
			bestNameLen = nameLen
			continue
		}
		// 同来源 → 名称更详细的优先
		if rank == bestRank && nameLen > bestNameLen {
			best = f
			bestNameLen = nameLen
			continue
		}
	}

	return best
}

// ─── 统计信息 ──────────────────────────────────────────────────

// MergeStats 合并统计
type MergeStats struct {
	TotalRaw     int `json:"totalRaw"`
	TotalMerged  int `json:"totalMerged"`
	Conflicts    int `json:"conflicts"`
	SourceCounts map[string]int `json:"sourceCounts"`
}

// ComputeMergeStats 计算合并统计
func ComputeMergeStats(raw, merged []InvestigationFinding) MergeStats {
	stats := MergeStats{
		TotalRaw:     len(raw),
		TotalMerged:  len(merged),
		SourceCounts: make(map[string]int),
	}

	// 统计冲突数 = 原始数 - 合并后（粗略）
	conflicts := len(raw) - len(merged)
	if conflicts > 0 {
		stats.Conflicts = conflicts
	}

	// 统计各来源数量
	for _, f := range merged {
		stats.SourceCounts[f.Source]++
	}

	return stats
}
