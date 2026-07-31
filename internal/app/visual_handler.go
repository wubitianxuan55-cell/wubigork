package app

import (
	"fmt"

	"github.com/gaea/gaea/internal/visual"
)

// ── 视觉叙事 API ────────────────────────────────────────────

// ExtractTimeline 提取故事时间线
func (a *App) ExtractTimeline() (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	tl, err := visual.ExtractTimeline(pm)
	if err != nil {
		return nil, err
	}

	var events []map[string]interface{}
	for _, ev := range tl.Events {
		events = append(events, map[string]interface{}{
			"chapter_num":   ev.ChapterNum,
			"title":         ev.Title,
			"summary":       ev.Summary,
			"emotion":       ev.Emotion,
			"characters":    ev.Characters,
			"pov":           ev.POV,
			"word_count":    ev.WordCount,
			"key_events":    ev.KeyEvents,
			"quality_score": ev.QualityScore,
		})
	}

	return map[string]interface{}{
		"events":        events,
		"total_words":   tl.TotalWords,
		"chapter_count": tl.ChapterCount,
		"pov_chars":     tl.POVChars,
	}, nil
}

// ExtractEmotionCurve 提取情绪曲线
func (a *App) ExtractEmotionCurve() ([]map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	points, err := visual.ExtractEmotionCurve(pm)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, p := range points {
		result = append(result, map[string]interface{}{
			"chapter_num": p.ChapterNum,
			"label":       p.Label,
			"emotion":     p.Emotion,
			"tension":     p.Tension,
			"valence":     p.Valence,
			"word_count":  p.WordCount,
		})
	}
	return result, nil
}

// ExtractCharacterHeatmap 提取角色热力图
func (a *App) ExtractCharacterHeatmap() (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	cells, charNames, maxChapter, err := visual.ExtractCharacterHeatmap(pm)
	if err != nil {
		return nil, err
	}

	var cellMaps []map[string]interface{}
	for _, c := range cells {
		cellMaps = append(cellMaps, map[string]interface{}{
			"character_name": c.CharacterName,
			"chapter_num":    c.ChapterNum,
			"appears":        c.Appears,
			"mention_count":  c.MentionCount,
			"is_pov":         c.IsPOV,
		})
	}

	return map[string]interface{}{
		"cells":        cellMaps,
		"characters":   charNames,
		"chapter_count": maxChapter,
	}, nil
}

// GenerateDefaultCanvas 生成默认画布
func (a *App) GenerateDefaultCanvas() (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}

	data, err := visual.GenerateDefaultCanvas(pm)
	if err != nil {
		return nil, err
	}

	var cards []map[string]interface{}
	for _, c := range data.Cards {
		cards = append(cards, map[string]interface{}{
			"id":          c.ID,
			"type":        c.Type,
			"title":       c.Title,
			"content":     c.Content,
			"x":           c.X,
			"y":           c.Y,
			"width":       c.Width,
			"height":      c.Height,
			"color":       c.Color,
			"chapter_ref": c.ChapterRef,
		})
	}

	var edges []map[string]interface{}
	for _, e := range data.Edges {
		edges = append(edges, map[string]interface{}{
			"id":      e.ID,
			"from_id": e.FromID,
			"to_id":   e.ToID,
			"label":   e.Label,
			"color":   e.Color,
		})
	}

	return map[string]interface{}{
		"cards": cards,
		"edges": edges,
	}, nil
}
