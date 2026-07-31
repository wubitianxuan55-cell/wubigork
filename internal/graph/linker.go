package graph

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/gaea/gaea/internal/project"
)

// ── 双向链接 ─────────────────────────────────────────────────

// Link 一个 [[wiki-link]] 引用
type Link struct {
	Target     string `json:"target"`     // 链接目标实体名
	SourceFile string `json:"source_file"` // 来源文件路径
	LineNumber int    `json:"line_number"` // 行号（1-based）
	Context    string `json:"context"`     // 周围文本（20字上下文）
}

// BacklinkIndex 反向链接索引：实体名 → 所有引用它的位置
type BacklinkIndex map[string][]Link

// wikiLinkRE 匹配 [[任意文本]]
var wikiLinkRE = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// ParseLinks 从文本中提取所有 [[wiki-link]]
func ParseLinks(content string) []string {
	matches := wikiLinkRE.FindAllStringSubmatch(content, -1)
	seen := make(map[string]bool)
	var result []string
	for _, m := range matches {
		target := strings.TrimSpace(m[1])
		if target != "" && !seen[target] {
			seen[target] = true
			result = append(result, target)
		}
	}
	return result
}

// ParseLinksWithContext 从文本中提取 [[wiki-link]] 及上下文
func ParseLinksWithContext(content string, sourceFile string) []Link {
	lines := strings.Split(content, "\n")
	var links []Link

	for lineNum, line := range lines {
		matches := wikiLinkRE.FindAllStringSubmatchIndex(line, -1)
		for _, m := range matches {
			if len(m) < 4 {
				continue
			}
			target := strings.TrimSpace(line[m[2]:m[3]])
			if target == "" {
				continue
			}
			// 提取上下文（周围20字符）
			ctxStart := m[0] - 10
			if ctxStart < 0 {
				ctxStart = 0
			}
			ctxEnd := m[1] + 10
			if ctxEnd > len(line) {
				ctxEnd = len(line)
			}
			context := line[ctxStart:ctxEnd]

			links = append(links, Link{
				Target:     target,
				SourceFile: sourceFile,
				LineNumber: lineNum + 1,
				Context:    context,
			})
		}
	}
	return links
}

// BuildBacklinkIndex 扫描项目中所有章节，构建反向链接索引
func BuildBacklinkIndex(pm *project.Manager) (BacklinkIndex, error) {
	index := make(BacklinkIndex)

	// 扫描章节
	for chapterNum := 1; ; chapterNum++ {
		content, err := pm.ReadChapter(chapterNum)
		if err != nil {
			break // 章节不存在
		}

		sourceFile := fmt.Sprintf("chapters/%03d.md", chapterNum)
		links := ParseLinksWithContext(content, sourceFile)
		for _, link := range links {
			index[link.Target] = append(index[link.Target], link)
		}
	}

	// 按行号排序
	for key := range index {
		sort.Slice(index[key], func(i, j int) bool {
			return index[key][i].LineNumber < index[key][j].LineNumber
		})
	}

	return index, nil
}

// GetBacklinks 获取某个实体的所有反向链接
func (idx BacklinkIndex) GetBacklinks(entityName string) []Link {
	return idx[entityName]
}

// GetAllEntities 获取索引中所有实体名
func (idx BacklinkIndex) GetAllEntities() []string {
	var entities []string
	for k := range idx {
		entities = append(entities, k)
	}
	sort.Strings(entities)
	return entities
}

// FindUnlinkedMentions 在文本中查找未被 [[ ]] 包裹的实体名
// entities: 已知实体名列表
func FindUnlinkedMentions(content string, entities []string) []string {
	var result []string

	for _, entity := range entities {
		// 检查是否已被 [[entity]] 链接
		linkedPattern := `\[\[` + regexp.QuoteMeta(entity) + `\]\]`
		linkedRE := regexp.MustCompile(linkedPattern)

		// 移除所有已链接的实例，检查剩余裸文本
		cleaned := linkedRE.ReplaceAllString(content, "")

		// 在清理后的文本中查找裸实体名
		barePattern := regexp.QuoteMeta(entity)
		bareRE := regexp.MustCompile(barePattern)
		if bareRE.MatchString(cleaned) {
			result = append(result, entity)
		}
	}
	return result
}
