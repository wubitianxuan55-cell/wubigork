package novelstyle

// 模式级判定资产（oh-story T2 内核消费，规格 docs/gaea-novel-ohstory-distill-2026-09.md
// T2）：默认表从 patterns.json go:embed，运行时可被惯例目录
// .gaea/skills/novel-deslop/patterns.json 整体替换（同 words.json 纪律）。
// 边界与 gates.json 一致：内核只装「同句高置信」（门禁 B）与「解释腔标记」
// （门禁 G）这两个可机械判定的模式族；跨段复核制、语义裁决（删了丢不丢锚点）
// 是模型按技能执行的事，内核不下结论。

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sync/atomic"
)

// patternTables 可替换的模式集合。
type patternTables struct {
	// NegationFlipHigh 否定铺垫后肯定翻转（同句=高置信，门禁 B）。
	NegationFlipHigh []string `json:"negationFlipHigh"`
	// ExplanatoryMarkers 解释腔/上帝感标记（门禁 G；advisory——承担锚点则保留）。
	ExplanatoryMarkers []string `json:"explanatoryMarkers"`
	// TrailerEndings 章末升华/预告腔标记（门禁 F；advisory，只在章末窗口判定——
	// 同样的词出现在章中可能是正常叙述，位置才是这类 AI 味的判据）。
	TrailerEndings []string `json:"trailerEndings"`
}

// compiledPatterns 编译后的模式（装载期编译：坏正则 fail-closed 不换出）。
type compiledPatterns struct {
	neg     []*regexp.Regexp
	expl    []*regexp.Regexp
	trailer []*regexp.Regexp
}

//go:embed patterns.json
var patternsJSON []byte

var patterns = newPatternsSnapshot()

func newPatternsSnapshot() *atomic.Pointer[compiledPatterns] {
	p := &atomic.Pointer[compiledPatterns]{}
	p.Store(mustLoadEmbeddedPatterns())
	return p
}

func mustLoadEmbeddedPatterns() *compiledPatterns {
	t := &patternTables{}
	if err := json.Unmarshal(patternsJSON, t); err != nil {
		panic(fmt.Sprintf("novelstyle 内置模式表损坏: %v", err))
	}
	c, err := compilePatterns(t)
	if err != nil {
		panic(fmt.Sprintf("novelstyle 内置模式表损坏: %v", err))
	}
	return c
}

func compilePatterns(t *patternTables) (*compiledPatterns, error) {
	c := &compiledPatterns{}
	for _, p := range t.NegationFlipHigh {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("否定翻转正则不可编译 %q: %w", p, err)
		}
		c.neg = append(c.neg, re)
	}
	for _, p := range t.ExplanatoryMarkers {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("解释腔标记正则不可编译 %q: %w", p, err)
		}
		c.expl = append(c.expl, re)
	}
	for _, p := range t.TrailerEndings {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("章末升华正则不可编译 %q: %w", p, err)
		}
		c.trailer = append(c.trailer, re)
	}
	return c, nil
}

func currentPatterns() *compiledPatterns { return patterns.Load() }

// LoadPatternsFile 用覆盖文件整体替换模式表（两族全量替换，不合并——语义可
// 预测）。任何一条正则不可编译都整表拒绝（fail-closed 不换出）。文件不存在
// 返回 nil（无覆盖=内置默认）。
func LoadPatternsFile(path string) error {
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
	t := &patternTables{}
	if err := json.Unmarshal(b, t); err != nil {
		return fmt.Errorf("模式覆盖文件解析失败 %s: %w", path, err)
	}
	if len(t.NegationFlipHigh) == 0 && len(t.ExplanatoryMarkers) == 0 && len(t.TrailerEndings) == 0 {
		return fmt.Errorf("模式覆盖文件无有效内容 %s", path)
	}
	c, err := compilePatterns(t)
	if err != nil {
		return fmt.Errorf("模式覆盖文件校验失败 %s: %w", path, err)
	}
	patterns.Store(c)
	return nil
}

// explanatoryMarkerCap advisory 标记的计分上限：解释腔命中常见于正常叙述，
// 超量计分会把低置信提示放大成主要扣分（gates.json G：advisory）。
const explanatoryMarkerCap = 5

// trailerEndingWindow 章末升华判定的窗口（章末最后 N 个 rune）：升华/预告腔
// 的判据是位置（收尾用总结陈词代替具体收束），同样的话出现在章中不算。
const trailerEndingWindow = 240

// trailerEndingCap 章末升华 advisory 计分上限：收尾只发生一次，命中即提示，
// 不因同段多个标记词重复放大。
const trailerEndingCap = 1

// ruleTrailerEnding 规则 12（门禁 F，advisory）：章末升华/预告腔。
// 只在章末 trailerEndingWindow 窗口内匹配：大段抒情总结（命运/宿命/齿轮式
// 陈词、「才刚刚开始」式预告）代替安静具体收束。内核分不清「有功能的收束
// 陈词」与「无功能升华」，一律 S3 提示，删不删由作者/语义裁决。
func ruleTrailerEnding(text string, runes []rune) []TasteIssue {
	pats := currentPatterns().trailer
	if len(pats) == 0 {
		return nil
	}
	n := len(runes)
	w := trailerEndingWindow
	if n < w {
		w = n
	}
	tailStart := n - w
	tailRunes := runes[tailStart:]
	tailText := string(tailRunes)
	var issues []TasteIssue
	for _, re := range pats {
		for _, loc := range re.FindAllStringIndex(tailText, -1) {
			if len(issues) >= trailerEndingCap {
				return issues
			}
			start := tailStart + byteToRune(tailRunes, loc[0])
			end := tailStart + byteToRune(tailRunes, loc[1])
			if end > n || start >= end {
				continue
			}
			issues = append(issues, TasteIssue{
				Start: start, End: end,
				Reason:     "章末升华/预告腔：结尾用抒情总结陈词收束（advisory：安静细节或动作收尾，悬念优先于升华）",
				Severity:   "S3",
				Suggestion: "删总结陈词，落在一个具体动作、细节或对话上；确需留余韵时用可感的物象，不用命运/齿轮式概括。",
			})
		}
	}
	return issues
}

// byteToRune 把字节偏移换算成 rune 偏移（TasteIssue 的 span 口径，rune 下标）。
func byteToRune(runes []rune, byteIdx int) int {
	b := 0
	for i, r := range runes {
		if b >= byteIdx {
			_ = r
			return i
		}
		b += len(string(r))
	}
	return len(runes)
}

// ruleNegationFlip 规则 10（门禁 B，同句高置信）：否定铺垫后肯定翻转。
// 「不是 A，而是 B」「没有 X 没有 Y，只是 Z」一类决策栏式句式。
func ruleNegationFlip(text string, runes []rune) []TasteIssue {
	var issues []TasteIssue
	for _, re := range currentPatterns().neg {
		for _, loc := range re.FindAllStringIndex(text, -1) {
			start, end := byteToRune(runes, loc[0]), byteToRune(runes, loc[1])
			if end > len(runes) || start >= end {
				continue
			}
			issues = append(issues, TasteIssue{
				Start: start, End: end,
				Reason:     "句式套路：否定铺垫后肯定翻转（不是A，而是B）",
				Severity:   "S2",
				Suggestion: "直接写肯定项，或把对比改成动作承担。",
			})
		}
	}
	return issues
}

// ruleExplanatoryMarkers 规则 11（门禁 G，advisory）：解释腔/上帝感标记。
// 内核分不清「无功能解释」与「承担锚点的伏笔」，一律 S4 级提示并封顶，
// 删不删由作者/技能按语义裁决。
func ruleExplanatoryMarkers(text string, runes []rune) []TasteIssue {
	var issues []TasteIssue
	for _, re := range currentPatterns().expl {
		for _, loc := range re.FindAllStringIndex(text, -1) {
			if len(issues) >= explanatoryMarkerCap {
				return issues
			}
			start, end := byteToRune(runes, loc[0]), byteToRune(runes, loc[1])
			if end > len(runes) || start >= end {
				continue
			}
			issues = append(issues, TasteIssue{
				Start: start, End: end,
				Reason:     "解释腔标记（advisory：无功能解释可删，承担锚点/情绪则保留）",
				Severity:   "S4",
				Suggestion: "删无功能的解释；确需保留信息时压成角色白话或场内载体（手机/公告/屏幕）。",
			})
		}
	}
	return issues
}
