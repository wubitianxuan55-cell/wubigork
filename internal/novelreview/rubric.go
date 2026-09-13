package novelreview

// 质量 rubric 数据资产（蒸馏自 oh-story-claudecode，MIT；证据与刀路见
// docs/gaea-novel-ohstory-distill-2026-09.md §2/§4）。
//
// 纪律与 internal/novelstyle 词表同款（v4.225 用户拍板红线：领域规则表不写死在
// Go 代码里）——**阈值 / 平台档位 / 判定词表是数据资产**（rubric.json go:embed），
// 引擎是通用机制留码；运行时可被覆盖文件**整体替换**，惯例路径
// <cwd>/.gaea/skills/novel-review/rubric.json（app 层接线 ensureNovelReviewRubric），
// 改门槛不改代码不发版（想增条目把默认表抄进覆盖文件再改）。

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"sync/atomic"
)

// RubricMeta 数据资产出处（面板/文档可展示）。
type RubricMeta struct {
	Source  string `json:"source"`
	Spec    string `json:"spec"`
	Version string `json:"version"`
}

// Markers 判定词表（知识资产：冲突/情绪/悬念/爽点/金手指/预告收尾）。
type Markers struct {
	Conflict       []string `json:"conflict"`
	Emotion        []string `json:"emotion"`
	Suspense       []string `json:"suspense"`
	Power          []string `json:"power"`
	Lever          []string `json:"lever_mention"`
	TrailerEnding  []string `json:"trailerEnding"`
	TrailerSummary []string `json:"trailerSummary"`
}

// Platform 一个平台档位（阈值 + 维度严重度映射；form=chapter 按章评审、story 按整篇口径）。
type Platform struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Form  string `json:"form"` // chapter | story

	// 篇幅口径：form=chapter 用 chapterWordsMin/Max；form=story 用 wordsMin/Max。
	ChapterWordsMin int `json:"chapterWordsMin,omitempty"`
	ChapterWordsMax int `json:"chapterWordsMax,omitempty"`
	WordsMin        int `json:"wordsMin,omitempty"`
	WordsMax        int `json:"wordsMax,omitempty"`

	LongParagraphChars    int     `json:"longParagraphChars"`
	OpeningHookParagraphs int     `json:"openingHookParagraphs"`
	DialogRatioMin        float64 `json:"dialogRatioMin"`
	DialogRatioMax        float64 `json:"dialogRatioMax"`
	EmotionPerKilo        float64 `json:"emotionPerKilo"`
	EmotionGapWarn        int     `json:"emotionGapWarn"`
	EmotionGapFail        int     `json:"emotionGapFail"`
	EllipsisPerKiloWarn   float64 `json:"ellipsisPerKiloWarn"`
	ParaUniformRatioWarn  float64 `json:"paraUniformRatioWarn"`
	PowerPer3k            float64 `json:"powerPer3k,omitempty"`
	LeverRequired         bool    `json:"leverRequired,omitempty"`
	FirstPerson           bool    `json:"firstPerson,omitempty"`
	NoBlankLines          bool    `json:"noBlankLines,omitempty"`

	// Severity 维度 → 严重度（S1 打回 / S2 顾虑 / S3 局部 / S4 建议）。
	Severity map[string]string `json:"severity"`
}

// Rubric 全部档位 + 词表（可整体替换的数据资产）。
type Rubric struct {
	Meta       RubricMeta `json:"meta"`
	Advisories []string   `json:"advisories"`
	Markers    Markers    `json:"markers"`
	Platforms  []Platform `json:"platforms"`
}

// severityValues 合法严重度档（与 oh-story quality-rubric.md 的 S1~S4 同口径）。
var severityValues = map[string]bool{"S1": true, "S2": true, "S3": true, "S4": true}

// dimensionIDs 引擎实现的维度 ID 全集（数据资产只能引用这些键）。
func dimensionIDs() []string {
	return []string{
		"length_band", "opening_freshness", "hook_expectation", "trailer_ending", "emotion_curve",
		"plot_loop", "format_readability", "dialogue_ratio", "punctuation_rhythm", "dash_usage",
		"format_hygiene", "perspective_consistency", "protagonist_presence", "lever_mention", "stated_length_accuracy",
	}
}

//go:embed rubric.json
var rubricJSON []byte

// rubric 数据资产快照（原子换出，与 novelstyle 词表同款）。
var rubric = newRubricSnapshot()

func newRubricSnapshot() *atomic.Pointer[Rubric] {
	p := &atomic.Pointer[Rubric]{}
	p.Store(mustLoadEmbeddedRubric())
	return p
}

func mustLoadEmbeddedRubric() *Rubric {
	r := &Rubric{}
	if err := json.Unmarshal(rubricJSON, r); err != nil {
		// 内置资产随构建走，损坏属构建事故——fail-fast 暴露，不带病运行。
		panic(fmt.Sprintf("novelreview 内置 rubric 损坏: %v", err))
	}
	if err := validateRubric(r); err != nil {
		panic(fmt.Sprintf("novelreview 内置 rubric 非法: %v", err))
	}
	return r
}

// validateRubric 数据资产自检：档位齐备、维度键与严重度合法、词表非空。
// 覆盖文件解析后同样过这道闸——坏资产宁可报错也不静默降级成「什么都查不出」。
func validateRubric(r *Rubric) error {
	if r == nil {
		return fmt.Errorf("rubric 为空")
	}
	if len(r.Platforms) == 0 {
		return fmt.Errorf("platforms 为空")
	}
	valid := map[string]bool{}
	for _, id := range dimensionIDs() {
		valid[id] = true
	}
	seen := map[string]bool{}
	for _, p := range r.Platforms {
		if p.ID == "" || p.Label == "" {
			return fmt.Errorf("档位缺 id/label")
		}
		if seen[p.ID] {
			return fmt.Errorf("档位 id 重复: %s", p.ID)
		}
		seen[p.ID] = true
		if p.Form != "chapter" && p.Form != "story" {
			return fmt.Errorf("档位 %s form 非法: %q", p.ID, p.Form)
		}
		if p.Form == "chapter" && (p.ChapterWordsMin <= 0 || p.ChapterWordsMax < p.ChapterWordsMin) {
			return fmt.Errorf("档位 %s 章节字数区间非法", p.ID)
		}
		if p.Form == "story" && (p.WordsMin <= 0 || p.WordsMax < p.WordsMin) {
			return fmt.Errorf("档位 %s 整篇字数区间非法", p.ID)
		}
		if p.LongParagraphChars <= 0 || p.OpeningHookParagraphs <= 0 {
			return fmt.Errorf("档位 %s 缺长段/开篇段阈值", p.ID)
		}
		for k, v := range p.Severity {
			if !valid[k] {
				return fmt.Errorf("档位 %s 引用了未知维度 %q", p.ID, k)
			}
			if !severityValues[v] {
				return fmt.Errorf("档位 %s 维度 %s 严重度非法: %q", p.ID, k, v)
			}
		}
	}
	if len(r.Markers.Conflict) == 0 || len(r.Markers.Emotion) == 0 || len(r.Markers.Suspense) == 0 {
		return fmt.Errorf("判定词表缺失（conflict/emotion/suspense 均不可为空）")
	}
	return nil
}

func currentRubric() *Rubric { return rubric.Load() }

// LoadRubricFile 用覆盖文件整体替换 rubric（不与默认表合并——语义可预测）。
// 文件不存在返回 nil（无覆盖=内置默认）；解析或校验失败**返回错误**且不换出，
// 调用方决定是报错还是静默保默认。
func LoadRubricFile(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	r := &Rubric{}
	if err := json.Unmarshal(b, r); err != nil {
		return fmt.Errorf("rubric 覆盖文件解析失败 %s: %w", path, err)
	}
	if err := validateRubric(r); err != nil {
		return fmt.Errorf("rubric 覆盖文件非法 %s: %w", path, err)
	}
	rubric.Store(r)
	return nil
}

// PlatformByID 取档位（id 为空或未知时回落到 general；无 general 时取第一个）。
func PlatformByID(id string) (Platform, error) {
	r := currentRubric()
	if id != "" {
		for _, p := range r.Platforms {
			if p.ID == id {
				return p, nil
			}
		}
	}
	for _, p := range r.Platforms {
		if p.ID == "general" {
			return p, nil
		}
	}
	if len(r.Platforms) == 0 {
		return Platform{}, fmt.Errorf("rubric 无可用档位")
	}
	return r.Platforms[0], nil
}

// PlatformIDs 返回全部档位 id（前端选择器/错误提示用）。
func PlatformIDs() []string {
	r := currentRubric()
	out := make([]string, 0, len(r.Platforms))
	for _, p := range r.Platforms {
		out = append(out, p.ID)
	}
	return out
}

// PlatformLabels 返回 id → 中文档位名（前端选择器直显）。
func PlatformLabels() map[string]string {
	r := currentRubric()
	out := make(map[string]string, len(r.Platforms))
	for _, p := range r.Platforms {
		out[p.ID] = p.Label
	}
	return out
}
