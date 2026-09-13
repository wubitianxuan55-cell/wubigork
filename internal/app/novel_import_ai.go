package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/bookimport"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/prompt"
	"github.com/gaea/gaea/internal/types"
)

// ── 拆书导入 P1：大纲反推（AI）与落地（规格 docs/distill/02-book-import.md §8.2 / §8.3 首刀）──
//
// 口径：① 反推**只读**（预览载荷，零落库）；② 落库走 NovelOutlineReconstructApply，
// 按章号与既有大纲节点合并（**幂等**：同一载荷连跑两次结果一致）；
// ③ AI 不可用/解析失败逐级降级到规则兜底并如实回报（aiUsed=false + warnings），不静默失败。

const (
	reconstructBatchSize   = 5
	reconstructMaxChapters = 200
	reconstructSampleChaps = 3
	reconstructSampleRunes = 2000
	reconstructMaxAttempts = 3
	reconstructTimeout     = 10 * time.Minute
)

// OutlineReconstructItem 反推结果条目（前端预览与落库共用载荷）。
// 短篇 = 单章条目（chapterNumber）；中/长篇 = 聚合骨架条目（chapterFrom/chapterTo
// 标注跨度，oh-story T4 篇幅路由），落库时新建卷级参考节点。
type OutlineReconstructItem struct {
	ChapterNumber int      `json:"chapterNumber"`
	ChapterFrom   int      `json:"chapterFrom,omitempty"` // 聚合骨架：起始章（>0 = 骨架条目）
	ChapterTo     int      `json:"chapterTo,omitempty"`   // 聚合骨架：结束章
	Title         string   `json:"title"`
	Summary       string   `json:"summary"`
	Scenes        []string `json:"scenes,omitempty"`
	Characters    []string `json:"characters,omitempty"` // 角色/组织名（落库仅作展示，角色 ID 匹配另刀）
	KeyPoints     []string `json:"keyPoints,omitempty"`
	Emotion       string   `json:"emotion,omitempty"`
	Goal          string   `json:"goal,omitempty"`

	// matchedCharacterIDs 应用期由角色名匹配填入（不序列化，预览不可见；
	// 角色库为空时为空）。见 matchCharacterIDs。
	matchedCharacterIDs []string `json:"-"`
}

// OutlineReconstructPreview 反推预览载荷（零落库）。
type OutlineReconstructPreview struct {
	AIUsed               bool                     `json:"aiUsed"`
	ProjectTitle         string                   `json:"projectTitle"`
	Description          string                   `json:"description,omitempty"`
	Theme                string                   `json:"theme,omitempty"`
	Genre                string                   `json:"genre,omitempty"`
	NarrativePerspective string                   `json:"narrativePerspective,omitempty"`
	TargetWords          int                      `json:"targetWords,omitempty"`
	Items                []OutlineReconstructItem `json:"items"`
	Tier                 string                   `json:"tier,omitempty"`       // 篇幅档位 short|mid|long（T4 篇幅路由）
	SegmentSize          int                      `json:"segmentSize,omitempty"` // >0 = items 已按每 N 章聚合
	Warnings             []string                 `json:"warnings,omitempty"`
}

// NovelOutlineReconstruct 对**当前打开的工程**跑大纲反推：立项信息（Stage1）+
// 分批章节大纲（batchSize=5），返回预览载荷，**不写任何文件**。
func (a *writingState) NovelOutlineReconstruct() (OutlineReconstructPreview, error) {
	pm := a.getPM()
	if pm == nil {
		return OutlineReconstructPreview{}, fmt.Errorf("请先打开项目")
	}
	chapters := reconstructProjectChapters(pm)
	if len(chapters) == 0 {
		return OutlineReconstructPreview{}, fmt.Errorf("这本书还没有已写章节，无法反推大纲")
	}
	title := ""
	if pm.Meta != nil {
		title = pm.Meta.Title
	}
	ctx, cancel := context.WithTimeout(context.Background(), reconstructTimeout)
	defer cancel()

	// ── 篇幅路由（oh-story T4）：按读到的总字数/章数选骨架粒度 ──
	totalWords := 0
	for _, ch := range chapters {
		totalWords += utf8.RuneCountInString(ch.Content)
	}
	tier := bookimport.RouteTier(totalWords, len(chapters))
	segSize := bookimport.SegmentSize(tier)

	preview := OutlineReconstructPreview{
		ProjectTitle: title,
		Tier:         string(tier),
		SegmentSize:  segSize,
		Items:        []OutlineReconstructItem{},
		Warnings:     []string{},
	}
	// ── Stage1：立项反推（失败降级规则兜底）──
	suggestion := bookimport.FallbackProjectSuggestion(title, chapters)
	if raw, err := a.reconstructStage1(ctx, title, chapters); err != nil {
		preview.Warnings = append(preview.Warnings, "立项反推失败，已用规则兜底："+err.Error())
	} else {
		suggestion = bookimport.NormalizeProjectSuggestion(raw, suggestion)
		preview.AIUsed = true
	}
	preview.Description = suggestion.Description
	preview.Theme = suggestion.Theme
	preview.Genre = suggestion.Genre
	preview.NarrativePerspective = suggestion.NarrativePerspective
	preview.TargetWords = suggestion.TargetWords

	// ── 分批章节大纲（每批独立降级）──
	var structs []bookimport.OutlineStructure
	for start := 0; start < len(chapters); start += reconstructBatchSize {
		end := start + reconstructBatchSize
		if end > len(chapters) {
			end = len(chapters)
		}
		batch := chapters[start:end]
		batchStructs, err := a.reconstructBatch(ctx, suggestion, batch, start, len(chapters))
		if err != nil {
			preview.Warnings = append(preview.Warnings,
				fmt.Sprintf("第 %d-%d 章反推失败，已用规则兜底：%v", start+1, end, err))
			batchStructs = fallbackBatch(batch, start+1)
		} else {
			preview.AIUsed = true
		}
		structs = append(structs, batchStructs...)
	}
	if segSize > 0 {
		// 中/长篇：章级条目聚合为骨架节点（卷级粗纲），预览/落库以骨架为准。
		for _, seg := range bookimport.AggregateSkeleton(structs, segSize) {
			preview.Items = append(preview.Items, OutlineReconstructItem{
				ChapterNumber: seg.ChapterFrom,
				ChapterFrom:   seg.ChapterFrom,
				ChapterTo:     seg.ChapterTo,
				Title:         seg.Title,
				Summary:       seg.Summary,
			})
		}
	} else {
		for _, st := range structs {
			preview.Items = append(preview.Items, toPreviewItem(st))
		}
	}
	return preview, nil
}

// matchCharacterIDs 角色名→角色库 ID 匹配（v4.290，拆书线欠账：反推条目带
// 角色名，大纲节点 Characters 字段收角色 ID）。规则：精确名优先，双向包含
// 兜底（反推给「林晚」、库里有「林晚儿」）；去重保序。
// 匹配不到的名字静默跳过（角色库是用户资产，不因反推编造角色）。
func matchCharacterIDs(names []string, chars []types.Character) []string {
	if len(names) == 0 || len(chars) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var ids []string
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		for _, c := range chars {
			if seen[c.ID] || c.Name == "" {
				continue
			}
			if c.Name == name || strings.Contains(c.Name, name) || strings.Contains(name, c.Name) {
				ids = append(ids, c.ID)
				seen[c.ID] = true
				break
			}
		}
	}
	return ids
}

// NovelOutlineReconstructApply 把预览载荷**合并**到当前工程的大纲：
// 章级条目（短篇）按章号命中既有节点写 summary/scenes/key_points/emotion
// （角色名留预览，不写角色 ID 字段——那是角色库的活），**幂等**，不新建/不删除
// 节点、不碰章节正文。
// 骨架条目（中/长篇，chapterFrom>0，T4 篇幅路由）：新建卷级参考节点
// （seg-NNN，planned，根级）；节点由反推功能持有——重复应用时先移除旧的
// seg-* 再落新节点，保持可重复执行不堆积。
func (a *writingState) NovelOutlineReconstructApply(itemsJSON string) (int, error) {
	pm := a.getPM()
	if pm == nil {
		return 0, fmt.Errorf("请先打开项目")
	}
	var items []OutlineReconstructItem
	if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil {
		return 0, fmt.Errorf("解析反推结果失败: %w", err)
	}
	if len(items) == 0 {
		return 0, fmt.Errorf("反推结果为空，未做任何修改")
	}
	byNum := make(map[int]OutlineReconstructItem, len(items))
	var segItems []OutlineReconstructItem
	for _, it := range items {
		if it.ChapterFrom > 0 && it.ChapterTo > 0 {
			segItems = append(segItems, it) // 骨架条目：走新建卷级节点分支
			continue
		}
		if it.ChapterNumber > 0 {
			byNum[it.ChapterNumber] = it
		}
	}
	// 角色名→角色库 ID 匹配（v4.290，拆书线欠账）：库为空/读失败时无匹配，
	// 不影响其余合并；角色库是用户资产，匹配不到的名字不编造角色。
	if cf, cerr := pm.ReadCharacters(); cerr == nil && cf != nil {
		for num, it := range byNum {
			if ids := matchCharacterIDs(it.Characters, cf.Characters); len(ids) > 0 {
				it.matchedCharacterIDs = ids
				byNum[num] = it
			}
		}
	}
	outline, err := pm.ReadOutlines()
	if err != nil || outline == nil {
		return 0, fmt.Errorf("读取大纲失败: %w", err)
	}
	// 骨架节点由反推持有：重复应用先移除旧 seg-*（幂等不堆积）。
	keptNodes := outline.Nodes[:0:0]
	maxOrder := 0
	for _, n := range outline.Nodes {
		if strings.HasPrefix(n.ID, "seg-") {
			continue
		}
		if n.OrderIndex > maxOrder {
			maxOrder = n.OrderIndex
		}
		keptNodes = append(keptNodes, n)
	}
	outline.Nodes = keptNodes
	for k, seg := range segItems {
		outline.Nodes = append(outline.Nodes, types.OutlineNode{
			ID:         fmt.Sprintf("seg-%03d", k+1),
			Title:      seg.Title,
			Summary:    seg.Summary,
			OrderIndex: maxOrder + 1 + k,
			Status:     types.OutlinePlanned,
		})
	}
	created := len(segItems)
	updated := 0
	for i := range outline.Nodes {
		node := &outline.Nodes[i]
		num := types.ChapterNumOf(node.ChapterFile)
		if num <= 0 {
			num = node.OrderIndex
		}
		it, ok := byNum[num]
		if !ok {
			continue
		}
		if it.Summary != "" {
			node.Summary = it.Summary
		}
		if len(it.Scenes) > 0 {
			node.SceneIdeas = it.Scenes
		}
		if len(it.KeyPoints) > 0 {
			node.KeyPoints = it.KeyPoints
		}
		if len(it.matchedCharacterIDs) > 0 {
			// 合并去重：既有出场角色 ID 保留在前，新增匹配追加在后。
			have := map[string]bool{}
			for _, id := range node.Characters {
				have[id] = true
			}
			for _, id := range it.matchedCharacterIDs {
				if !have[id] {
					node.Characters = append(node.Characters, id)
					have[id] = true
				}
			}
		}
		if it.Emotion != "" {
			node.Emotion = it.Emotion
		}
		updated++
	}
	if updated == 0 && created == 0 {
		return 0, fmt.Errorf("没有匹配到任何章节节点（反推结果与当前工程对不上）")
	}
	if err := pm.WriteOutlines(outline); err != nil {
		return 0, fmt.Errorf("写大纲失败: %w", err)
	}
	return updated + created, nil
}

// ── 内部实现 ──────────────────────────────────────────────────

// reconstructProjectChapters 读当前工程章节（标题取大纲节点，正文走 Stitch 口径）。
func reconstructProjectChapters(pm *project.Manager) []bookimport.ParsedChapter {
	titleByFile := map[string]string{}
	if outline, err := pm.ReadOutlines(); err == nil && outline != nil {
		for _, n := range outline.Nodes {
			if n.ChapterFile != "" && strings.TrimSpace(n.Title) != "" {
				titleByFile[n.ChapterFile] = n.Title
			}
		}
	}
	out := make([]bookimport.ParsedChapter, 0, reconstructMaxChapters)
	for i := 1; i <= reconstructMaxChapters; i++ {
		content, err := pm.ReadChapterAsStitch(i)
		if err != nil {
			break
		}
		if strings.TrimSpace(content) == "" {
			continue
		}
		file := fmt.Sprintf("%03d.md", i)
		title := titleByFile[file]
		if title == "" {
			title = fmt.Sprintf("第%d章", i)
		}
		out = append(out, bookimport.ParsedChapter{Title: title, Content: content})
	}
	return out
}

// reconstructStage1 立项反推（模板 book-import-project；期望 object）。
func (a *writingState) reconstructStage1(ctx context.Context, title string, chapters []bookimport.ParsedChapter) (map[string]any, error) {
	tmpl := a.reconstructTemplate("book-import-project")
	if tmpl == nil {
		return nil, fmt.Errorf("模板缺失：book-import-project")
	}
	user := tmpl.BuildUserPrompt(map[string]string{
		"title":        title,
		"sampled_text": bookimport.SampledText(chapters, reconstructSampleChaps, reconstructSampleRunes),
	})
	raw, err := bookimport.CallJSON(ctx, user, bookimport.ExpectObject, reconstructMaxAttempts,
		func(c context.Context, p string) (string, error) { return a.reconstructChat(c, tmpl.System, p) })
	if err != nil {
		return nil, err
	}
	obj, _ := raw.(map[string]any)
	return obj, nil
}

// reconstructBatch 单批章节大纲反推（模板 book-import-outline；期望 array）。
func (a *writingState) reconstructBatch(ctx context.Context, s bookimport.ProjectSuggestion,
	batch []bookimport.ParsedChapter, startIdx, total int) ([]bookimport.OutlineStructure, error) {
	tmpl := a.reconstructTemplate("book-import-outline")
	if tmpl == nil {
		return nil, fmt.Errorf("模板缺失：book-import-outline")
	}
	user := tmpl.BuildUserPrompt(map[string]string{
		"project_title":         s.Title,
		"genre":                 s.Genre,
		"theme":                 s.Theme,
		"narrative_perspective": s.NarrativePerspective,
		"batch_range":           fmt.Sprintf("第 %d-%d 章（全书共 %d 章）", startIdx+1, startIdx+len(batch), total),
		"expected_count":        fmt.Sprintf("%d", len(batch)),
		"chapters_text":         bookimport.BatchText(batch),
	})
	raw, err := bookimport.CallJSON(ctx, user, bookimport.ExpectArray, reconstructMaxAttempts,
		func(c context.Context, p string) (string, error) { return a.reconstructChat(c, tmpl.System, p) })
	if err != nil {
		return nil, err
	}
	return bookimport.NormalizeOutlineBatch(raw, batch, startIdx+1), nil
}

// reconstructChat 小说域模型调用（与重写/续写同路由：routeModel("novel")）。
func (a *writingState) reconstructChat(ctx context.Context, system, user string) (string, error) {
	if a.client == nil {
		return "", fmt.Errorf("模型服务不可用，请先配置小说功能模型")
	}
	eng, model, _ := a.routeModel("novel")
	if model == "" {
		return "", fmt.Errorf("未找到可用模型（可能离线）")
	}
	return a.client.ChatSimpleStreamWithOptions(ctx, model, system, user, ai.ChatSimpleOptions{
		EngineID: eng, Temperature: 0.3, MaxTokens: 8192,
	})
}

func (a *writingState) reconstructTemplate(name string) *prompt.Template {
	if a.eng == nil {
		return nil
	}
	if t := a.eng.Get(name); t != nil {
		return t
	}
	return nil
}

func fallbackBatch(batch []bookimport.ParsedChapter, startNumber int) []bookimport.OutlineStructure {
	out := make([]bookimport.OutlineStructure, 0, len(batch))
	for i, ch := range batch {
		out = append(out, bookimport.FallbackStructure(ch, startNumber+i))
	}
	return out
}

func toPreviewItem(st bookimport.OutlineStructure) OutlineReconstructItem {
	item := OutlineReconstructItem{
		ChapterNumber: st.ChapterNumber,
		Title:         st.Title,
		Summary:       st.Summary,
		Scenes:        st.Scenes,
		KeyPoints:     st.KeyPoints,
		Emotion:       st.Emotion,
		Goal:          st.Goal,
	}
	for _, c := range st.Characters {
		item.Characters = append(item.Characters, c.Name)
	}
	return item
}
