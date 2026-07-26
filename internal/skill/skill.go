package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Skill 单个写作指导 Skill
type Skill struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	AppliesTo   []string `json:"applies_to"`
	Version     string   `json:"version"`
	Body        string   `json:"body"`
	Path        string   `json:"path"`
}

// Loader Skill 加载器
type Loader struct {
	skills map[string]*Skill // name → Skill
}

// NewLoader 创建加载器，扫描指定目录
func NewLoader(globalDir string) *Loader {
	l := &Loader{skills: make(map[string]*Skill)}
	l.scan(globalDir)
	return l
}

// scan 扫描直接子目录，解析 SKILL.md 文件
func (l *Loader) scan(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillFile := filepath.Join(dir, entry.Name(), "SKILL.md")
		skill, err := parseSkillFile(skillFile)
		if err != nil {
			continue
		}
		skill.Path = skillFile
		l.skills[skill.Name] = skill
	}
}

// Get 按名称获取 Skill
func (l *Loader) Get(name string) *Skill {
	return l.skills[name]
}

// List 列出所有 Skill 的名称和描述
func (l *Loader) List() []Skill {
	var result []Skill
	for _, s := range l.skills {
		result = append(result, Skill{
			Name:        s.Name,
			Description: s.Description,
			AppliesTo:   s.AppliesTo,
			Version:     s.Version,
		})
	}
	return result
}

// FilterByAppliesTo 按适用范围过滤
func (l *Loader) FilterByAppliesTo(target string) []*Skill {
	var result []*Skill
	for _, s := range l.skills {
		for _, a := range s.AppliesTo {
			if a == target {
				result = append(result, s)
				break
			}
		}
	}
	return result
}

// InjectSkill 将 Skill 内容追加到 system prompt 后
func (l *Loader) InjectSkill(basePrompt string, skillNames ...string) string {
	if len(skillNames) == 0 {
		return basePrompt
	}

	var sb strings.Builder
	sb.WriteString(basePrompt)

	for _, name := range skillNames {
		s := l.Get(name)
		if s == nil {
			continue
		}
		sb.WriteString("\n\n---\n## 写作指导: ")
		sb.WriteString(s.Name)
		sb.WriteString("\n")
		sb.WriteString(s.Body)
	}

	return sb.String()
}

// ── 内部解析 ────────────────────────────────────────────────

func parseSkillFile(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	content = strings.ReplaceAll(content, "\r\n", "\n") // Windows 换行兼容

	// 解析 YAML frontmatter (--- ... ---)
	if !strings.HasPrefix(content, "---\n") {
		return nil, fmt.Errorf("SKILL.md 缺少 YAML frontmatter")
	}

	// 找到第二个 ---

	// 找到第二个 ---
	rest := content[4:] // 跳过第一个 "---\n"
	endIdx := strings.Index(rest, "\n---")
	if endIdx == -1 {
		return nil, fmt.Errorf("SKILL.md YAML frontmatter 未闭合")
	}

	frontmatter := rest[:endIdx]
	body := rest[endIdx+4:] // 跳过 "\n---"

	skill := &Skill{Body: strings.TrimSpace(body)}

	// 简单逐行解析 YAML
	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, "\"'")

		switch key {
		case "name":
			skill.Name = val
		case "description":
			skill.Description = val
		case "version":
			skill.Version = val
		case "applies_to":
			// 解析 YAML 列表: [chapter, outline]
			val = strings.Trim(val, "[]")
			for _, item := range strings.Split(val, ",") {
				item = strings.TrimSpace(item)
				if item != "" {
					skill.AppliesTo = append(skill.AppliesTo, item)
				}
			}
		}
	}

	if skill.Name == "" {
		return nil, fmt.Errorf("SKILL.md 缺少 name 字段")
	}

	return skill, nil
}
