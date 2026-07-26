package analysis

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/wubigork/wubigork/internal/types"
	"github.com/wubigork/wubigork/internal/util"
)

// EvolutionResult 章节生成后的自我演化结果
type EvolutionResult struct {
	NewCharacters       []string               `json:"new_characters"` // 新出现的角色名
	NewLocations        []string               `json:"new_locations"`  // 新出现的地点
	NewConcepts         []string               `json:"new_concepts"`   // 新出现的概念/道具
	LorebookSuggestions []LorebookSuggestion   `json:"lorebook_suggestions"`
	ForeshadowChanges   []ForeshadowChangeNote `json:"foreshadow_changes"`
	CharacterUpdates    []CharacterUpdateNote  `json:"character_updates"`
	WorldviewUpdates    []WorldviewAppend      `json:"worldview_updates"`
}

// LorebookSuggestion 建议新增的 Lorebook 词条
type LorebookSuggestion struct {
	Key      string `json:"key"`
	Content  string `json:"content"`
	Category string `json:"category"`
	Reason   string `json:"reason"` // 为什么建议添加
}

// ForeshadowChangeNote 伏笔变化说明
type ForeshadowChangeNote struct {
	Description string `json:"description"`
	Action      string `json:"action"` // planted / hinted / revealed
}

// CharacterUpdateNote 角色更新说明
type CharacterUpdateNote struct {
	Name    string `json:"name"`
	OldNote string `json:"old_note"`
	NewNote string `json:"new_note"`
	Reason  string `json:"reason"`
}

// WorldviewAppend 世界观追加条目
type WorldviewAppend struct {
	Dimension string `json:"dimension"` // era/geography/factions/rules/culture/history
	Content   string `json:"content"`
	Reason    string `json:"reason"`
}

// EvolveAfterChapter 章节生成后触发自我演化
func (a *Agent) EvolveAfterChapter(chapterNum int, chapterContent string, summary *types.ChapterSummary) (*EvolutionResult, error) {
	result := &EvolutionResult{}

	// 1. 从章节摘要中提取新角色名
	if summary != nil {
		existingChars, err := a.pm.ReadCharacters()
		if err != nil {
			slog.Warn("EvolveAfterChapter: 读取角色失败", "error", err)
		}
		existingNames := make(map[string]bool)
		if existingChars != nil {
			for _, ch := range existingChars.Characters {
				existingNames[ch.Name] = true
			}
		}
		for _, name := range summary.CharactersAppeared {
			if !existingNames[name] {
				result.NewCharacters = append(result.NewCharacters, name)
			}
		}
	}

	// 2. 扫描章节内容提取地名/概念（简单关键词匹配）
	locations := extractLocations(chapterContent)
	existingLocs, err := a.pm.ReadLorebook()
	if err != nil {
		slog.Warn("EvolveAfterChapter: 读取Lorebook失败", "error", err)
	}
	existingKeys := make(map[string]bool)
	if existingLocs != nil {
		for _, e := range existingLocs.Entries {
			existingKeys[e.Key] = true
		}
	}
	for _, loc := range locations {
		if !existingKeys[loc] {
			result.NewLocations = append(result.NewLocations, loc)
			result.LorebookSuggestions = append(result.LorebookSuggestions, LorebookSuggestion{
				Key: loc, Content: fmt.Sprintf("（请补充%s的设定）", loc),
				Category: "location", Reason: fmt.Sprintf("第%d章提到的新地点", chapterNum),
			})
		}
	}

	// 3. 检查世界观是否有新维度需要追加
	wf, err := a.pm.ReadWorldviewFile()
	if err == nil && wf != nil {
		for i := range wf.Sections {
			sec := &wf.Sections[i]
			if sec.Content == "" {
				// 尝试从章节内容提取相关信息
				hint := extractWorldviewHint(chapterContent, sec.ID)
				if hint != "" {
					result.WorldviewUpdates = append(result.WorldviewUpdates, WorldviewAppend{
						Dimension: sec.ID, Content: hint,
						Reason: fmt.Sprintf("从第%d章自动推断", chapterNum),
					})
					// 自动写入
					sec.Content = hint + "\n（from ch." + fmt.Sprintf("%d", chapterNum) + "）"
				}
			}
		}
		if len(result.WorldviewUpdates) > 0 {
			if err := a.pm.WriteWorldviewFile(wf); err != nil {
				slog.Warn("演化：写入世界观失败", "error", err)
			}
		}
	}

	// 4. 记录伏笔变化（已有 syncForeshadows 处理，这里仅记录日志）
	ff, err := a.pm.ReadForeshadows()
	if err != nil {
		slog.Warn("EvolveAfterChapter: 读取伏笔失败", "error", err)
	}
	if ff != nil {
		for _, f := range ff.Items {
			if f.PlantedIn == fmt.Sprintf("%03d.md", chapterNum) {
				result.ForeshadowChanges = append(result.ForeshadowChanges, ForeshadowChangeNote{
					Description: f.Description, Action: "planted",
				})
			}
			if f.RevealedIn == fmt.Sprintf("%03d.md", chapterNum) {
				result.ForeshadowChanges = append(result.ForeshadowChanges, ForeshadowChangeNote{
					Description: f.Description, Action: "revealed",
				})
			}
		}
	}

	return result, nil
}

// extractLocations 简单提取章节中可能的地名（2-4个汉字+常见后缀）
func extractLocations(content string) []string {
	suffixes := []rune("城/镇/村/山/海/湖/谷/林/殿/阁/楼/院/宗/宫/寺/堂/国/界/域/府/")
	// 构建后缀集合
	suffixSet := make(map[rune]bool)
	for _, r := range suffixes {
		if r != '/' {
			suffixSet[r] = true
		}
	}

	var result []string
	runes := []rune(content)
	seen := make(map[string]bool)

	for i := 0; i < len(runes)-1; i++ {
		if !suffixSet[runes[i]] {
			continue
		}
		// look backwards 1-3 chars
		for start := i - 1; start >= i-3 && start >= 0; start-- {
			candidate := string(runes[start : i+1])
			if len([]rune(candidate)) >= 2 && !seen[candidate] {
				result = append(result, candidate)
				seen[candidate] = true
			}
		}
	}
	return result
}

// extractWorldviewHint 尝试从章节内容提取世界观维度提示
func extractWorldviewHint(content, dimensionID string) string {
	keywords := map[string][]string{
		"era":       {"时代", "年代", "古代", "现代", "未来", "王朝", "帝国"},
		"geography": {"山脉", "河流", "大陆", "平原", "沙漠", "海洋", "气候"},
		"factions":  {"势力", "门派", "家族", "联盟", "帝国", "王国", "组织"},
		"rules":     {"法则", "规则", "修炼", "魔法", "灵力", "境界", "能力"},
		"culture":   {"习俗", "节日", "礼仪", "传统", "服饰", "饮食", "语言"},
		"history":   {"历史", "传说", "战争", "灾难", "纪元", "遗迹", "灭世"},
	}
	words, ok := keywords[dimensionID]
	if !ok {
		return ""
	}
	for _, w := range words {
		if strings.Contains(content, w) {
			// 提取包含关键词的句子
			idx := strings.Index(content, w)
			start := util.Max(0, idx-30)
			end := util.Min(len(content), idx+len(w)+60)
			snippet := content[start:end]
			return "..." + snippet + "..."
		}
	}
	return ""
}
