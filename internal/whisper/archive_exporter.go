// Package whisper — archive_exporter.go
// 100% 对齐 ackem memory/archiveExporter.ts
// 记忆档案导出：FactStore → 按领域/子类分组的 Markdown 文件

package whisper

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ─── 中文标签 ──────────────────────────────────────────────────

// factGroup 记忆分组（领域+子类）
type factGroup struct {
	domain string
	subcat string
	facts  []MemoryFact
}

var archiveDomainZH = map[string]string{
	"IDENTITY":    "自我与身份",
	"SOCIAL":      "关系与社交",
	"DAILY_LIFE":  "日常生活",
	"PURSUITS":    "事业与成长",
	"INNER_WORLD": "内心世界",
	"TEMPORAL":    "当下与未来",
}

var archiveSubcatZH = map[string]string{
	"BASIC_PROFILE":   "基本信息",
	"LIFE_STORY":      "人生经历",
	"VALUES_BELIEFS":  "价值观与信念",
	"SELF_PERCEPTION": "自我认知",
	"OUR_BOND":        "我们的羁绊",
	"FAMILY":          "家庭",
	"FRIENDS":         "朋友",
	"PARTNER":         "伴侣",
	"ROUTINES":        "日常习惯",
	"HEALTH":          "身心健康",
	"LIVING_SPACE":    "居住环境",
	"LIFESTYLE":       "生活方式",
	"CAREER":          "事业与工作",
	"LEARNING":        "学习与技能",
	"GOALS":           "目标与梦想",
	"PROJECTS":        "项目与创作",
	"PROCEDURES":      "做事方式",
	"TASTES":          "喜好与品味",
	"COMMITMENTS":     "承诺与约定",
	"PLANS":           "近期计划",
	"MEMORIES":        "回忆",
}

// ─── 导出统计 ──────────────────────────────────────────────────

// ExportStats 导出统计
type ExportStats struct {
	FilesWritten    int `json:"filesWritten"`
	FactsExported   int `json:"factsExported"`
	EpisodesExported int `json:"episodesExported"`
	CoreCount       int `json:"coreCount"`
}

// ─── 导出入口 ──────────────────────────────────────────────────

// ExportMemoryArchive 导出记忆档案
// 100% 对齐 ackem archiveExporter.ts exportMemoryArchive
func ExportMemoryArchive(dataRoot string, fs *FactStore, episodes []string) ExportStats {
	archiveDir := filepath.Join(dataRoot, "memory", "archive")
	os.MkdirAll(archiveDir, 0755)

	allFacts := fs.ListActive()
	coreFacts := fs.SelectCoreFacts(9999)

	stats := ExportStats{
		CoreCount: len(coreFacts),
	}

	// 按领域→子类分组

	groups := make(map[string]*factGroup)
	for _, f := range allFacts {
		key := f.Domain + "::" + f.Subcategory
		if g, ok := groups[key]; ok {
			g.facts = append(g.facts, f.MemoryFact)
		} else {
			groups[key] = &factGroup{
				domain: f.Domain,
				subcat: f.Subcategory,
				facts:  []MemoryFact{f.MemoryFact},
			}
		}
	}

	// 按领域排序
	var sortedKeys []string
	for k := range groups {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	// 写入文件
	var domainFiles []string
	for _, key := range sortedKeys {
		g := groups[key]
		md := formatFactGroupMarkdown(g)
		if md == "" {
			continue
		}

		domainName := archiveDomainZH[g.domain]
		if domainName == "" {
			domainName = g.domain
		}
		subcatName := archiveSubcatZH[g.subcat]
		if subcatName == "" {
			subcatName = g.subcat
		}

		domainDir := filepath.Join(archiveDir, domainName)
		os.MkdirAll(domainDir, 0755)
		filename := filepath.Join(domainDir, subcatName+".md")
		os.WriteFile(filename, []byte(md), 0644)

		domainFiles = append(domainFiles, filename)
		stats.FilesWritten++
		stats.FactsExported += len(g.facts)
	}

	// 写入情节
	if len(episodes) > 0 {
		epFile := filepath.Join(archiveDir, "重要情节.md")
		epMd := formatEpisodesMarkdown(episodes)
		os.WriteFile(epFile, []byte(epMd), 0644)
		stats.FilesWritten++
		stats.EpisodesExported = len(episodes)
	}

	// 写入索引
	indexFile := filepath.Join(archiveDir, "_索引.md")
	indexMd := formatArchiveIndex(domainFiles, stats)
	os.WriteFile(indexFile, []byte(indexMd), 0644)
	stats.FilesWritten++

	return stats
}

// ─── Markdown 格式化 ───────────────────────────────────────────

func formatFactGroupMarkdown(g *factGroup) string {
	if len(g.facts) == 0 {
		return ""
	}

	domainName := archiveDomainZH[g.domain]
	if domainName == "" {
		domainName = g.domain
	}
	subcatName := archiveSubcatZH[g.subcat]
	if subcatName == "" {
		subcatName = g.subcat
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s · %s\n\n", domainName, subcatName))
	sb.WriteString(fmt.Sprintf("> 共 %d 条记忆\n\n", len(g.facts)))

	for _, f := range g.facts {
		core := ""
		if f.Tier == "core" {
			core = " ★"
		}
		summary := escapeMarkdown(f.Summary)
		conf := fmt.Sprintf("%.0f%%", f.Confidence*100)
		sb.WriteString(fmt.Sprintf("- [%s] %s%s\n", conf, summary, core))
		if f.UpdatedAt.After(time.Time{}) {
			sb.WriteString(fmt.Sprintf("  *%s*\n", f.UpdatedAt.Format("2006-01-02 15:04")))
		}
	}

	return sb.String()
}

func formatEpisodesMarkdown(episodes []string) string {
	var sb strings.Builder
	sb.WriteString("# 重要情节\n\n")
	for i, ep := range episodes {
		sb.WriteString(fmt.Sprintf("## 情节 %d\n\n%s\n\n", i+1, escapeMarkdown(ep)))
	}
	return sb.String()
}

func formatArchiveIndex(files []string, stats ExportStats) string {
	var sb strings.Builder
	sb.WriteString("# 记忆档案索引\n\n")
	sb.WriteString(fmt.Sprintf("- 生成时间：%s\n", time.Now().Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("- 总事实数：%d\n", stats.FactsExported))
	sb.WriteString(fmt.Sprintf("- 核心事实：%d\n", stats.CoreCount))
	sb.WriteString(fmt.Sprintf("- 情节数：%d\n", stats.EpisodesExported))
	sb.WriteString(fmt.Sprintf("- 文件数：%d\n\n", stats.FilesWritten))
	sb.WriteString("## 文件列表\n\n")
	for _, f := range files {
		rel, _ := filepath.Rel(filepath.Dir(f), f)
		sb.WriteString(fmt.Sprintf("- [%s](%s)\n", filepath.Base(f), rel))
	}
	return sb.String()
}

func escapeMarkdown(s string) string {
	s = strings.ReplaceAll(s, "*", "\\*")
	s = strings.ReplaceAll(s, "_", "\\_")
	s = strings.ReplaceAll(s, "#", "\\#")
	return s
}
