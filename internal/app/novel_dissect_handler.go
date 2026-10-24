package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/fileutil"
	"github.com/gaea/gaea/internal/project"
	"github.com/gaea/gaea/internal/util"
)

// ── 对标拆解（oh-story story-long-analyze 机制重推导蒸馏）──────────────
//
// 学习导向的写法拆解，与章节分析（修改导向 analysis-v2）分面：拆「这本书
// 的写法哪里值得学」（付息/钩子/情绪曲线/信息推进/写法公式），不出修改建议。
// 产物逐章落 <项目根>/dissections/NNN.json（原子写；断点续跑=文件存在即已
// 拆解，force=false 直接返回存量不发 LLM）；聚合报告纯确定性零 LLM。
// 对标导入书即普通项目（ImportNovelBook 建档），打开后对本面板操作即可。

// novelDissectEntry 单章拆解条目（dissections/NNN.json 的文件结构）。
type novelDissectEntry struct {
	Chapter       int      `json:"chapter"`
	Payoff        string   `json:"payoff"`
	HookType      string   `json:"hook_type"`
	HookDetail    string   `json:"hook_detail"`
	EmotionArc    string   `json:"emotion_arc"`
	InfoPush      []string `json:"info_push"`
	CraftFormulas []string `json:"craft_formulas"`
	DissectedAt   string   `json:"dissected_at"`
}

// dissectHookTypes hook_type 合法枚举（与 novel-dissect.json 模板同一口径）。
var dissectHookTypes = map[string]bool{
	"危机": true, "揭示": true, "反转": true, "两难": true,
	"承诺": true, "意象": true, "留白": true, "无": true,
}

const (
	// dissectMaxInputRunes 章节全文进 prompt 的预算（rune 口径）。
	dissectMaxInputRunes = 12000
	// dissectMinChapterRunes 章节太短没有拆解价值（正文阈值，非硬内容判据）。
	dissectMinChapterRunes = 50
)

func dissectDir(pm *project.Manager) string { return filepath.Join(pm.Dir, "dissections") }

func dissectPath(pm *project.Manager, num int) string {
	return filepath.Join(dissectDir(pm), fmt.Sprintf("%03d.json", num))
}

// loadDissectEntry 读单章拆解；文件不存在返回 nil（未拆解），损坏返回错误。
func loadDissectEntry(pm *project.Manager, num int) (*novelDissectEntry, error) {
	data, err := os.ReadFile(dissectPath(pm, num))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var e novelDissectEntry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("拆解档损坏 %03d.json: %w", num, err)
	}
	return &e, nil
}

// NovelDissectChapter 拆解单章：LLM 提炼写法 → 归一 → 原子落盘。
// force=false 且已有拆解档时直接返回存量（断点续跑零 LLM）。
func (a *writingState) NovelDissectChapter(chapterNum int, force bool) (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	if chapterNum <= 0 {
		return nil, fmt.Errorf("章号不合法: %d", chapterNum)
	}
	if !force {
		if existing, err := loadDissectEntry(pm, chapterNum); err != nil {
			return nil, err
		} else if existing != nil {
			return entryToMap(existing), nil
		}
	}
	content, err := pm.ReadChapter(chapterNum)
	if err != nil {
		return nil, fmt.Errorf("读取第 %d 章失败: %w", chapterNum, err)
	}
	content = strings.TrimSpace(content)
	if len([]rune(content)) < dissectMinChapterRunes {
		return nil, fmt.Errorf("第 %d 章正文太短（%d 字），没有拆解价值", chapterNum, len([]rune(content)))
	}

	tmpl := a.eng.Get("novel-dissect")
	if tmpl == nil {
		return nil, fmt.Errorf("缺少 novel-dissect 模板文件")
	}
	systemPrompt := tmpl.BuildSystemPrompt("")
	userPrompt := tmpl.BuildUserPrompt(map[string]string{
		"chapter_num":  strconv.Itoa(chapterNum),
		"chapter_text": clampRunes(content, dissectMaxInputRunes),
	})

	eng, model, _ := a.routeModel("novel")
	opts := ai.ChatSimpleOptions{EngineID: eng, Feature: "novel"}
	applyChatSimpleMaxTokens(&opts, playGuardrails().MaxOutputTokens)
	reply, err := a.clientRef().ChatSimpleStreamWithOptions(a.ctx, model, systemPrompt, userPrompt, opts)
	if err != nil {
		return nil, fmt.Errorf("拆解第 %d 章失败: %w", chapterNum, err)
	}

	entry, err := parseNovelDissectReply(reply, chapterNum)
	if err != nil {
		return nil, err
	}
	if err := fileutil.AtomicWrite(dissectPath(pm, chapterNum), dissectJSON(entry), 0o644); err != nil {
		return nil, fmt.Errorf("保存拆解档失败: %w", err)
	}
	return entryToMap(entry), nil
}

// parseNovelDissectReply 提取并归一拆解 JSON（枚举落保守档、字段钳长）。
func parseNovelDissectReply(reply string, chapterNum int) (*novelDissectEntry, error) {
	jsonStr := util.ExtractJSON(reply)
	var raw struct {
		Payoff        string   `json:"payoff"`
		HookType      string   `json:"hook_type"`
		HookDetail    string   `json:"hook_detail"`
		EmotionArc    string   `json:"emotion_arc"`
		InfoPush      []string `json:"info_push"`
		CraftFormulas []string `json:"craft_formulas"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, fmt.Errorf("解析拆解 JSON 失败: %w", err)
	}
	if !dissectHookTypes[raw.HookType] {
		raw.HookType = "无" // 模型口径漂移时落保守档
	}
	entry := &novelDissectEntry{
		Chapter:       chapterNum,
		Payoff:        clampRunes(raw.Payoff, 200),
		HookType:      raw.HookType,
		HookDetail:    clampRunes(raw.HookDetail, 120),
		EmotionArc:    clampRunes(raw.EmotionArc, 120),
		InfoPush:      clampStringList(raw.InfoPush, 4, 120),
		CraftFormulas: clampStringList(raw.CraftFormulas, 3, 160),
		DissectedAt:   time.Now().Format(time.RFC3339),
	}
	return entry, nil
}

// clampStringList 逐条钳长并截断条数（保持顺序）。
func clampStringList(items []string, maxItems, maxRunes int) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		if len(out) >= maxItems {
			break
		}
		if s := clampRunes(it, maxRunes); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// NovelDissectList 已拆解章节清单（打开面板即拉；零 LLM）。
func (a *writingState) NovelDissectList() (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	entries, err := loadAllDissectEntries(pm)
	if err != nil {
		return nil, err
	}
	maps := make([]map[string]interface{}, 0, len(entries))
	for _, e := range entries {
		maps = append(maps, entryToMap(e))
	}
	return map[string]interface{}{"entries": maps}, nil
}

// NovelDissectReport 聚合报告（纯确定性零 LLM）：剧情线（付息链）/钩子类型
// 分布/情绪曲线/写法公式合并去重。
func (a *writingState) NovelDissectReport() (map[string]interface{}, error) {
	pm := a.getPM()
	if pm == nil {
		return nil, fmt.Errorf("请先打开项目")
	}
	entries, err := loadAllDissectEntries(pm)
	if err != nil {
		return nil, err
	}

	type timelineItem struct {
		Chapter  int    `json:"chapter"`
		Payoff   string `json:"payoff"`
		HookType string `json:"hook_type"`
	}
	type emotionItem struct {
		Chapter int    `json:"chapter"`
		Arc     string `json:"emotion_arc"`
	}
	timeline := make([]timelineItem, 0, len(entries))
	emotion := make([]emotionItem, 0, len(entries))
	hookDist := map[string]int{}
	formulas := []string{}
	seen := map[string]bool{}
	for _, e := range entries {
		timeline = append(timeline, timelineItem{Chapter: e.Chapter, Payoff: e.Payoff, HookType: e.HookType})
		emotion = append(emotion, emotionItem{Chapter: e.Chapter, Arc: e.EmotionArc})
		if e.HookType != "" && e.HookType != "无" {
			hookDist[e.HookType]++
		}
		for _, f := range e.CraftFormulas {
			if !seen[f] {
				seen[f] = true
				formulas = append(formulas, f)
			}
		}
	}
	return map[string]interface{}{
		"chapters_dissected": len(entries),
		"timeline":           timeline,
		"hook_dist":          hookDist,
		"emotion_arc":        emotion,
		"formulas":           formulas,
	}, nil
}

// loadAllDissectEntries 扫 dissections/ 全档并按章号升序。
func loadAllDissectEntries(pm *project.Manager) ([]*novelDissectEntry, error) {
	dir := dissectDir(pm)
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []*novelDissectEntry
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		var e novelDissectEntry
		data, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &e); err != nil {
			// 单档损坏跳过不阻断报告（诚实：跳过档可从清单缺失看出）
			continue
		}
		entries = append(entries, &e)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Chapter < entries[j].Chapter })
	return entries, nil
}

func entryToMap(e *novelDissectEntry) map[string]interface{} {
	return map[string]interface{}{
		"chapter":        e.Chapter,
		"payoff":         e.Payoff,
		"hook_type":      e.HookType,
		"hook_detail":    e.HookDetail,
		"emotion_arc":    e.EmotionArc,
		"info_push":      e.InfoPush,
		"craft_formulas": e.CraftFormulas,
		"dissected_at":   e.DissectedAt,
	}
}

// dissectJSON 序列化拆解档（自有静态结构，序列化不可达失败；空对象兜底不 panic）。
func dissectJSON(v interface{}) []byte {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return []byte("{}")
	}
	return data
}
