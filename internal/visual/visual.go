package visual

import (
	"fmt"
	"sort"
	"strings"

	"github.com/wubigork/wubigork/internal/project"
)

// ── 时间线数据 ───────────────────────────────────────────────

// TimelineEvent 时间线上的一个事件
type TimelineEvent struct {
	ChapterNum  int      `json:"chapter_num"`
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	Emotion     string   `json:"emotion"`
	Characters  []string `json:"characters"`
	POV         string   `json:"pov"`        // 推测 POV 角色
	WordCount   int      `json:"word_count"`
	KeyEvents   []string `json:"key_events"`
	QualityScore int     `json:"quality_score"`
}

// Timeline 完整时间线
type Timeline struct {
	Events     []TimelineEvent `json:"events"`
	TotalWords int             `json:"total_words"`
	ChapterCount int           `json:"chapter_count"`
	POVChars   []string        `json:"pov_chars"` // 所有 POV 角色
}

// ExtractTimeline 从项目提取时间线数据
func ExtractTimeline(pm *project.Manager) (*Timeline, error) {
	var events []TimelineEvent
	povSet := make(map[string]bool)
	totalWords := 0

	for chapterNum := 1; ; chapterNum++ {
		summary, err := pm.ReadChapterSummary(chapterNum)
		if err != nil {
			break
		}
		if summary == nil {
			continue
		}

		content, err := pm.ReadChapter(chapterNum)
		wordCount := 0
		if err == nil {
			wordCount = len([]rune(content))
		}
		totalWords += wordCount

		// 推测 POV（从出场角色中取第一个主角类型）
		pov := ""
		if len(summary.CharactersAppeared) > 0 {
			pov = summary.CharactersAppeared[0]
			povSet[pov] = true
		}

		events = append(events, TimelineEvent{
			ChapterNum:   chapterNum,
			Title:        summary.Title,
			Summary:      summary.Summary,
			Emotion:      summary.EmotionTone,
			Characters:   summary.CharactersAppeared,
			POV:          pov,
			WordCount:    wordCount,
			KeyEvents:    summary.KeyEvents,
			QualityScore: summary.QualityEstimate,
		})
	}

	var povChars []string
	for p := range povSet {
		povChars = append(povChars, p)
	}
	sort.Strings(povChars)

	return &Timeline{
		Events:       events,
		TotalWords:   totalWords,
		ChapterCount: len(events),
		POVChars:     povChars,
	}, nil
}

// ── 情绪曲线 ─────────────────────────────────────────────────

// EmotionPoint 情绪曲线上的一个点
type EmotionPoint struct {
	ChapterNum int     `json:"chapter_num"`
	Label      string  `json:"label"`      // 章节标题
	Emotion    string  `json:"emotion"`    // 原始情感标签
	Tension    float64 `json:"tension"`    // 紧张度 0-10
	Valence    float64 `json:"valence"`    // 正负情感 -5~+5
	WordCount  int     `json:"word_count"`
}

// extractEmotionValue 将情感标签转为数值
func extractEmotionValue(emotion string) (tension, valence float64) {
	lower := strings.ToLower(emotion)

	// 紧张度评分
	tensionKeywords := map[string]float64{
		"紧张": 8, "悬疑": 7, "恐惧": 9, "战斗": 8, "冲突": 7,
		"危机": 9, "高潮": 10, "追逐": 7, "对决": 8, "审判": 6,
		"悲伤": 5, "愤怒": 7, "绝望": 9, "焦虑": 6,
		"平静": 2, "温馨": 2, "日常": 1, "轻松": 1, "幽默": 1,
		"浪漫": 3, "感人": 4, "希望": 3,
	}

	tension = 5 // 默认中性
	for kw, val := range tensionKeywords {
		if strings.Contains(lower, kw) {
			tension = val
			break
		}
	}

	// 情感正负值 (-5=极度负面, 0=中性, +5=极度正面)
	valenceKeywords := map[string]float64{
		"恐惧": -4, "绝望": -5, "悲伤": -3, "愤怒": -2, "焦虑": -2,
		"紧张": -1, "悬疑": -1, "冲突": -1, "危机": -3, "战斗": -1,
		"温馨": 4, "浪漫": 4, "希望": 3, "感人": 3, "幽默": 3,
		"轻松": 2, "日常": 1, "平静": 1, "高兴": 5,
	}

	valence = 0
	for kw, val := range valenceKeywords {
		if strings.Contains(lower, kw) {
			valence = val
			break
		}
	}

	return
}

// ExtractEmotionCurve 从项目提取情绪曲线数据
func ExtractEmotionCurve(pm *project.Manager) ([]EmotionPoint, error) {
	var points []EmotionPoint

	for chapterNum := 1; ; chapterNum++ {
		summary, err := pm.ReadChapterSummary(chapterNum)
		if err != nil {
			break
		}
		if summary == nil {
			continue
		}

		content, err := pm.ReadChapter(chapterNum)
		wordCount := 0
		if err == nil {
			wordCount = len([]rune(content))
		}

		tension, valence := extractEmotionValue(summary.EmotionTone)

		points = append(points, EmotionPoint{
			ChapterNum: chapterNum,
			Label:      summary.Title,
			Emotion:    summary.EmotionTone,
			Tension:    tension,
			Valence:    valence,
			WordCount:  wordCount,
		})
	}

	return points, nil
}

// ── 角色弧光热力图 ──────────────────────────────────────────

// CharacterHeatmapCell 热力图的一个单元格
type CharacterHeatmapCell struct {
	CharacterName string `json:"character_name"`
	ChapterNum    int    `json:"chapter_num"`
	Appears       bool   `json:"appears"`       // 是否出场
	MentionCount  int    `json:"mention_count"` // 提及次数
	IsPOV         bool   `json:"is_pov"`        // 是否为主视角
}

// ExtractCharacterHeatmap 提取角色出场矩阵
func ExtractCharacterHeatmap(pm *project.Manager) ([]CharacterHeatmapCell, []string, int, error) {
	// 获取所有角色
	chars, err := pm.ReadCharacters()
	if err != nil || chars == nil {
		return nil, nil, 0, err
	}

	// 收集章节信息
	type chapterInfo struct {
		characters []string
		content    string
	}
	var chapters []chapterInfo
	maxChapter := 0

	for chapterNum := 1; ; chapterNum++ {
		summary, err := pm.ReadChapterSummary(chapterNum)
		if err != nil {
			break
		}
		content, _ := pm.ReadChapter(chapterNum)
		if summary != nil {
			chapters = append(chapters, chapterInfo{
				characters: summary.CharactersAppeared,
				content:    content,
			})
			maxChapter = chapterNum
		}
	}

	// 构建热力图
	var cells []CharacterHeatmapCell
	charNames := make([]string, 0, len(chars.Characters))

	for _, ch := range chars.Characters {
		charNames = append(charNames, ch.Name)

		for chIdx, chInfo := range chapters {
			chNum := chIdx + 1
			appears := false
			mentionCount := 0

			for _, appeared := range chInfo.characters {
				if appeared == ch.Name {
					appears = true
				}
			}

			if chInfo.content != "" {
				mentionCount = strings.Count(chInfo.content, ch.Name)
			}

			cells = append(cells, CharacterHeatmapCell{
				CharacterName: ch.Name,
				ChapterNum:    chNum,
				Appears:       appears,
				MentionCount:  mentionCount,
				IsPOV:         chIdx == 0 && appears, // 简化：第一章出场且为主角类型
			})
		}
	}

	return cells, charNames, maxChapter, nil
}

// ── Canvas 数据 ──────────────────────────────────────────────

// CanvasCard 画布上的一张卡片
type CanvasCard struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"` // scene / character / location / note
	Title     string  `json:"title"`
	Content   string  `json:"content"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Width     float64 `json:"width"`
	Height    float64 `json:"height"`
	Color     string  `json:"color"`
	ChapterRef int    `json:"chapter_ref,omitempty"`
}

// CanvasEdge 画布上的一条连线
type CanvasEdge struct {
	ID     string `json:"id"`
	FromID string `json:"from_id"`
	ToID   string `json:"to_id"`
	Label  string `json:"label"`
	Color  string `json:"color"`
}

// CanvasData 画布完整数据
type CanvasData struct {
	Cards []CanvasCard `json:"cards"`
	Edges []CanvasEdge `json:"edges"`
}

// GenerateDefaultCanvas 为项目生成默认画布（场景卡片按章节排列）
func GenerateDefaultCanvas(pm *project.Manager) (*CanvasData, error) {
	data := &CanvasData{}

	x := 20.0
	y := 20.0
	cols := 0

	for chapterNum := 1; ; chapterNum++ {
		summary, err := pm.ReadChapterSummary(chapterNum)
		if err != nil {
			break
		}
		if summary == nil {
			continue
		}

		emotionColor := emotionToColor(summary.EmotionTone)
		qualityLabel := ""
		if summary.QualityEstimate > 0 {
			qualityLabel = fmt.Sprintf(" [评分:%d]", summary.QualityEstimate)
		}

		data.Cards = append(data.Cards, CanvasCard{
			ID:         fmt.Sprintf("ch-%03d", chapterNum),
			Type:       "scene",
			Title:      summary.Title,
			Content:    summary.Summary + qualityLabel,
			X:          x,
			Y:          y,
			Width:      200,
			Height:     120,
			Color:      emotionColor,
			ChapterRef: chapterNum,
		})

		// 网格布局
		cols++
		x += 220
		if cols >= 4 {
			cols = 0
			x = 20
			y += 140
		}

		// 添加边（前序章节→当前章节）
		if chapterNum > 1 {
			data.Edges = append(data.Edges, CanvasEdge{
				ID:     fmt.Sprintf("edge-%03d", chapterNum),
				FromID: fmt.Sprintf("ch-%03d", chapterNum-1),
				ToID:   fmt.Sprintf("ch-%03d", chapterNum),
				Label:  "→",
				Color:  "#64748b",
			})
		}
	}

	return data, nil
}

func emotionToColor(emotion string) string {
	lower := strings.ToLower(emotion)
	switch {
	case strings.Contains(lower, "紧张"), strings.Contains(lower, "悬疑"), strings.Contains(lower, "恐惧"):
		return "#f87171" // red
	case strings.Contains(lower, "温馨"), strings.Contains(lower, "浪漫"), strings.Contains(lower, "希望"):
		return "#4ade80" // green
	case strings.Contains(lower, "悲伤"), strings.Contains(lower, "绝望"):
		return "#60a5fa" // blue
	case strings.Contains(lower, "愤怒"), strings.Contains(lower, "战斗"):
		return "#f59e0b" // amber
	case strings.Contains(lower, "平静"), strings.Contains(lower, "日常"):
		return "#9ca3af" // gray
	default:
		return "#c084fc" // purple
	}
}
