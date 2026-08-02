// Package whisper — archive_exporter.go
// 对齐 ackem memory/archiveExporter.ts
// P3 记忆归档导出：按领域/子类分目录生成 Markdown 档案

package whisper

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ArchiveFile 单个归档文件（相对路径 + Markdown 内容）
type ArchiveFile struct {
	Path    string
	Content string
}

// archiveDomainOrder 领域展示顺序
var archiveDomainOrder = []string{
	DomainIdentity,
	DomainSocial,
	DomainDailyLife,
	DomainPursuits,
	DomainInnerWorld,
	DomainTemporal,
	"KNOWLEDGE",
}

// archiveFileHeader 归档文件头部
const archiveFileHeader = "> Hermes 记忆归档 · 自动生成，勿手工编辑"

// ─── BuildArchive ─────────────────────────────────────────────

// BuildArchive 构建归档文件集（内存 FactStore 版）
func BuildArchive(fs *FactStore, eps *EpisodicStore) []ArchiveFile {
	if fs == nil {
		fs = NewFactStore()
	}
	return BuildArchiveFromFacts(fs.ListActive(), eps)
}

// BuildArchiveFromFacts 构建归档文件集：README 索引 + 每个领域/子类一个 Markdown 文件
// 输入为记忆事实列表（hermes.db 或内存 FactStore），退役事实不入档
func BuildArchiveFromFacts(active []*Fact, eps *EpisodicStore) []ArchiveFile {
	// 过滤非活跃事实
	var live []*Fact
	for _, f := range active {
		if f == nil || !f.IsActive() {
			continue
		}
		live = append(live, f)
	}
	active = live

	// 分组：domain → subcategory → facts
	groups := map[string]map[string][]*Fact{}
	for _, f := range active {
		if groups[f.Domain] == nil {
			groups[f.Domain] = map[string][]*Fact{}
		}
		groups[f.Domain][f.Subcategory] = append(groups[f.Domain][f.Subcategory], f)
	}

	// 排序：领域按固定顺序，子类按字典序，事实按创建时间
	domains := archiveDomainOrder
	for _, d := range domains {
		if groups[d] == nil {
			continue
		}
		for sub := range groups[d] {
			facts := groups[d][sub]
			sort.Slice(facts, func(i, j int) bool {
				return facts[i].CreatedAt.Before(facts[j].CreatedAt)
			})
		}
	}
	// 补充未列出的领域（按字典序）
	var extraDomains []string
	for d := range groups {
		found := false
		for _, known := range domains {
			if d == known {
				found = true
				break
			}
		}
		if !found {
			extraDomains = append(extraDomains, d)
		}
	}
	sort.Strings(extraDomains)
	domains = append(domains, extraDomains...)

	var files []ArchiveFile
	coreCount := 0
	for _, d := range domains {
		subs := groups[d]
		if subs == nil {
			continue
		}
		var subNames []string
		for s := range subs {
			subNames = append(subNames, s)
		}
		sort.Strings(subNames)
		for _, s := range subNames {
			facts := subs[s]
			for _, f := range facts {
				if f.IsCore() {
					coreCount++
				}
			}
			files = append(files, ArchiveFile{
				Path:    filepath.ToSlash(filepath.Join(d, s+".md")),
				Content: buildSubcategoryMarkdown(d, s, facts),
			})
		}
	}

	files = append([]ArchiveFile{{
		Path:    "README.md",
		Content: buildReadmeMarkdown(active, eps, coreCount),
	}}, files...)

	return files
}

// ─── WriteArchive ─────────────────────────────────────────────

// WriteArchive 将归档写入目录（内存 FactStore 版），返回写入文件数
func WriteArchive(fs *FactStore, eps *EpisodicStore, dir string) (int, error) {
	if fs == nil {
		fs = NewFactStore()
	}
	return WriteArchiveFromFacts(fs.ListActive(), eps, dir)
}

// WriteArchiveFromFacts 将归档写入目录（事实列表版），返回写入文件数
func WriteArchiveFromFacts(facts []*Fact, eps *EpisodicStore, dir string) (int, error) {
	files := BuildArchiveFromFacts(facts, eps)
	for _, f := range files {
		path := filepath.Join(dir, filepath.FromSlash(f.Path))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return 0, fmt.Errorf("archive: 创建目录失败 %s: %w", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(f.Content), 0o644); err != nil {
			return 0, fmt.Errorf("archive: 写入失败 %s: %w", path, err)
		}
	}
	return len(files), nil
}

// ─── 内容生成 ─────────────────────────────────────────────────

// buildSubcategoryMarkdown 生成单个子类档案
func buildSubcategoryMarkdown(domain, subcategory string, facts []*Fact) string {
	var sb strings.Builder
	sb.WriteString("# " + displayDomain(domain) + " / " + displaySubcategory(subcategory) + "\n\n")
	sb.WriteString(archiveFileHeader + "\n\n")
	sb.WriteString(fmt.Sprintf("> 事实数: %d\n\n", len(facts)))
	sb.WriteString("## 条目\n\n")

	for i, f := range facts {
		sb.WriteString(fmt.Sprintf("%d. **%s** — %s\n", i+1, f.Subject, f.Summary))
		sb.WriteString(fmt.Sprintf("   - 权重 %.1f · 置信度 %.1f · 相关度 %.1f\n",
			f.Weight, f.Confidence, f.SelfRelevance))
		sb.WriteString(fmt.Sprintf("   - 创建: %s · 更新: %s\n",
			formatArchiveTime(f.CreatedAt), formatArchiveTime(f.UpdatedAt)))
		var tags []string
		if f.IsCore() {
			tags = append(tags, "core")
		}
		if f.Tier == "core" {
			tags = append(tags, "核心")
		}
		if f.PrivacyLevel != "" && f.PrivacyLevel != "normal" {
			tags = append(tags, "隐私:"+f.PrivacyLevel)
		}
		if len(tags) > 0 {
			sb.WriteString("   - 标签: " + strings.Join(tags, " · ") + "\n")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// buildReadmeMarkdown 生成归档总览
func buildReadmeMarkdown(active []*Fact, eps *EpisodicStore, coreCount int) string {
	var sb strings.Builder
	sb.WriteString("# Hermes 记忆归档\n\n")
	sb.WriteString(archiveFileHeader + "\n\n")
	sb.WriteString(fmt.Sprintf("> 生成时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	sb.WriteString("## 统计\n\n")
	sb.WriteString(fmt.Sprintf("- 事实总数: %d", len(active)))
	if coreCount > 0 {
		sb.WriteString(fmt.Sprintf("（core: %d）", coreCount))
	}
	sb.WriteString("\n")
	if eps != nil {
		sb.WriteString(fmt.Sprintf("- 情节记忆: %d\n", eps.Count()))
	}

	// 领域分布
	domainCount := map[string]int{}
	subCount := map[string]int{}
	for _, f := range active {
		domainCount[f.Domain]++
		if domainCount[f.Domain] == 1 {
			subCount[f.Domain] = 0
		}
	}
	seen := map[string]bool{}
	for _, f := range active {
		key := f.Domain + "/" + f.Subcategory
		if !seen[key] {
			seen[key] = true
			subCount[f.Domain]++
		}
	}
	sb.WriteString("\n## 领域分布\n\n")
	sb.WriteString("| 领域 | 子类数 | 事实数 |\n|---|---|---|\n")
	for _, d := range archiveDomainOrder {
		if domainCount[d] == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("| %s | %d | %d |\n", displayDomain(d), subCount[d], domainCount[d]))
	}
	var extra []string
	for d := range domainCount {
		found := false
		for _, known := range archiveDomainOrder {
			if d == known {
				found = true
				break
			}
		}
		if !found {
			extra = append(extra, d)
		}
	}
	sort.Strings(extra)
	for _, d := range extra {
		sb.WriteString(fmt.Sprintf("| %s | %d | %d |\n", displayDomain(d), subCount[d], domainCount[d]))
	}
	return sb.String()
}

// formatArchiveTime 归档时间格式
func formatArchiveTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04")
}

// displayDomain 领域展示名（未知领域原样输出）
func displayDomain(d string) string {
	if l, ok := domainLabels[d]; ok {
		return l
	}
	return d
}

// displaySubcategory 子类展示名（未知子类原样输出）
func displaySubcategory(s string) string {
	if l, ok := subcatLabels[s]; ok {
		return l
	}
	return s
}
