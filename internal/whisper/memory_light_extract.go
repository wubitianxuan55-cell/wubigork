// Package whisper — memory_light_extract.go
// 100% 对齐 ackem memory/lightExtract/patterns.ts + index.ts
// 轻量规则提取引擎：正则匹配 → 即时事实草稿（毫秒级，无需 LLM）
//
// 8 类规则：
//   1. 自述生日  2. 家人生日  3. 姓名
//   4. 过敏      5. 喜好/厌恶  6. 职业
//   7. 宠物      8. 承诺/计划

package whisper

import (
	"regexp"
	"strconv"
	"strings"
)

// ─── 事实草稿类型 ──────────────────────────────────────────────

// FactDraftSource 草稿来源
type FactDraftSource string

const (
	SourceLightRule       FactDraftSource = "light_rule"
	SourceExplicitRemember FactDraftSource = "explicit_remember"
)

// FactDraft 轻量提取的事实草稿
type FactDraft struct {
	Domain      string   `json:"domain"`
	Subcategory string   `json:"subcategory"`
	Subject     string   `json:"subject"`
	Summary     string   `json:"summary"`
	Weight      float64  `json:"weight"`
	Confidence  float64  `json:"confidence"`
	Triggers    []string `json:"triggers,omitempty"`
	AgeMeta     *AgeMeta `json:"ageMeta,omitempty"`
	Source      FactDraftSource `json:"source"`
	RuleID      string   `json:"ruleId,omitempty"`
	FamilyScope string   `json:"familyScope,omitempty"`
}

// ─── 家人类系映射 ──────────────────────────────────────────────

type familyRelation struct {
	re      *regexp.Regexp
	label   string
	subject string
}

var familyRelations = []familyRelation{
	{regexp.MustCompile(`(?:妈妈|母亲|妈)`), "母亲", "用户母亲生日"},
	{regexp.MustCompile(`(?:爸爸|父亲|爸)`), "父亲", "用户父亲生日"},
	{regexp.MustCompile(`(?:奶奶|祖母)`), "奶奶", "用户奶奶生日"},
	{regexp.MustCompile(`(?:爷爷|祖父)`), "爷爷", "用户爷爷生日"},
	{regexp.MustCompile(`(?:外婆|姥姥|外祖母)`), "外婆", "用户外婆生日"},
	{regexp.MustCompile(`(?:外公|姥爷|外祖父)`), "外公", "用户外公生日"},
	{regexp.MustCompile(`(?:妹妹|姐姐|哥哥|弟弟|兄弟|姐妹)`), "兄弟姐妹", "用户兄弟姐妹生日"},
}

// ─── 日期解析 ──────────────────────────────────────────────────

var (
	reBirthdayZH = regexp.MustCompile(`(\d{1,2})\s*月\s*(\d{1,2})\s*日?`)
	reBirthdaySlash = regexp.MustCompile(`(\d{1,2})[/.](\d{1,2})`)
	reBirthdayEN    = regexp.MustCompile(`(?i)\b(jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[a-z]*\s+(\d{1,2})\b`)
)

var enMonths = map[string]int{
	"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
	"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
}

// parseBirthdayFromText 从文本解析生日日期
func parseBirthdayFromText(text string) (month, day int, ok bool) {
	if m := reBirthdayZH.FindStringSubmatch(text); m != nil {
		month, _ = strconv.Atoi(m[1])
		day, _ = strconv.Atoi(m[2])
		return clampMonthDay(month, day)
	}
	if m := reBirthdaySlash.FindStringSubmatch(text); m != nil {
		month, _ = strconv.Atoi(m[1])
		day, _ = strconv.Atoi(m[2])
		return clampMonthDay(month, day)
	}
	if m := reBirthdayEN.FindStringSubmatch(text); m != nil {
		key := strings.ToLower(m[1][:3])
		if mo, exists := enMonths[key]; exists {
			d, _ := strconv.Atoi(m[2])
			return clampMonthDay(mo, d)
		}
	}
	return 0, 0, false
}

func clampMonthDay(m, d int) (int, int, bool) {
	if m < 1 || m > 12 || d < 1 || d > 31 {
		return 0, 0, false
	}
	return m, d, true
}

// formatBirthdayMMDD 格式化月日为 MM-DD
func formatBirthdayMMDD(month, day int) string {
	return pad2(month) + "-" + pad2(day)
}

func pad2(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// calendarSuffix 检测农历/阳历标记
func calendarSuffix(text string) string {
	if strings.Contains(text, "阴历") || strings.Contains(text, "农历") {
		return "（阴历）"
	}
	if strings.Contains(text, "阳历") || strings.Contains(text, "公历") {
		return "（阳历）"
	}
	return ""
}

// ─── 正则规则引擎 ──────────────────────────────────────────────

var (
	reSelfBirthday = regexp.MustCompile(`(?:^|[^你])我(?:本人)?(?:的)?生日(?:是|在)?`)
	reMyBirthday   = regexp.MustCompile(`(?i)\bmy birthday\b`)
	reAllergy      = regexp.MustCompile(`过敏(?:了)?([一-鿿\w]{1,20})`)
	reDislike      = regexp.MustCompile(`(?:讨厌|不喜欢|不爱吃)([一-鿿\w]{1,12})`)
	reLike         = regexp.MustCompile(`(?:喜欢|爱吃|爱听)([一-鿿\w]{1,12})`)
	reJob1         = regexp.MustCompile(`(?:我是|职业是|做)([一-鿿\w]{2,16}(?:工程师|师|员|家|生|经理|开发|设计))`)
	reJob2         = regexp.MustCompile(`([一-鿿\w]{2,12})(?:专业|系)`)
	rePet1         = regexp.MustCompile(`(?:养了|有)(?:一?只?)([一-鿿\w]{1,8})(?:猫|狗|兔|鸟|宠物)`)
	rePet2         = regexp.MustCompile(`(?:猫|狗)叫([一-鿿\w]{1,8})`)
	reCommitment   = regexp.MustCompile(`(?:周末|下周|明天|别忘了|记得).{0,30}(?:一起|找我|见面|看电影|吃饭|提醒)`)
	reChineseName  = regexp.MustCompile(`(?:我叫|我是|叫我)([\p{Han}]{2,4})`)
)

// RunLightExtractRules 执行轻量规则提取
// 100% 对齐 ackem lightExtract/patterns.ts runLightExtractRules
func RunLightExtractRules(userMsg string) []FactDraft {
	text := strings.TrimSpace(userMsg)
	if text == "" {
		return nil
	}

	var drafts []FactDraft

	// 按标点分句
	parts := splitByPunct(text)
	if len(parts) == 0 {
		parts = []string{text}
	}

	for _, part := range parts {
		// 规则1: 自述生日
		if reSelfBirthday.MatchString(part) || reMyBirthday.MatchString(part) {
			if m, d, ok := parseBirthdayFromText(part); ok {
				mmdd := formatBirthdayMMDD(m, d)
				drafts = append(drafts, FactDraft{
					Domain:      "IDENTITY",
					Subcategory: "BASIC_PROFILE",
					Subject:     "用户生日",
					Summary:     "用户生日为" + strconv.Itoa(m) + "月" + strconv.Itoa(d) + "日" + calendarSuffix(text),
					Weight:      3,
					Confidence:  0.95,
					Triggers:    []string{"生日", "用户生日"},
					AgeMeta:     &AgeMeta{BirthdayMMDD: mmdd},
					Source:      SourceLightRule,
					RuleID:      "birthday_self",
				})
			}
		}

		// 规则2: 家人生日
		if strings.Contains(part, "生日") {
			if m, d, ok := parseBirthdayFromText(part); ok {
				for _, rel := range familyRelations {
					if rel.re.MatchString(part) {
						ruleID := "family_birthday_other"
						if rel.label == "母亲" {
							ruleID = "family_birthday_mom"
						} else if rel.label == "父亲" {
							ruleID = "family_birthday_dad"
						}
						mmdd := formatBirthdayMMDD(m, d)
						drafts = append(drafts, FactDraft{
							Domain:      "SOCIAL",
							Subcategory: "FAMILY",
							Subject:     rel.subject,
							Summary:     "用户" + rel.label + "生日为" + strconv.Itoa(m) + "月" + strconv.Itoa(d) + "日" + calendarSuffix(text),
							Weight:      2.5,
							Confidence:  0.95,
							Triggers:    []string{rel.label, "生日"},
							AgeMeta:     &AgeMeta{BirthdayMMDD: mmdd},
							Source:      SourceLightRule,
							RuleID:      ruleID,
							FamilyScope: "user",
						})
					}
				}
			}
		}
	}

	// 规则3: 姓名（全文本匹配）
	if m := reChineseName.FindStringSubmatch(text); m != nil {
		name := m[1]
		if isValidChineseName(name) {
			drafts = append(drafts, FactDraft{
				Domain:      "IDENTITY",
				Subcategory: "BASIC_PROFILE",
				Subject:     "用户姓名",
				Summary:     "用户叫" + name,
				Weight:      3,
				Confidence:  0.85,
				Triggers:    []string{name},
				Source:      SourceLightRule,
				RuleID:      "name_intro",
			})
		}
	}

	// 规则4: 过敏
	if strings.Contains(text, "过敏") {
		if m := reAllergy.FindStringSubmatch(text); m != nil {
			drafts = append(drafts, FactDraft{
				Domain:      "DAILY_LIFE",
				Subcategory: "HEALTH",
				Subject:     "用户过敏",
				Summary:     "用户对" + m[1] + "过敏",
				Weight:      2.5,
				Confidence:  0.9,
				Triggers:    []string{"过敏", m[1]},
				Source:      SourceLightRule,
				RuleID:      "allergy",
			})
		}
	}

	// 规则5: 厌恶
	if m := reDislike.FindStringSubmatch(text); m != nil {
		drafts = append(drafts, FactDraft{
			Domain:      "INNER_WORLD",
			Subcategory: "TASTES",
			Subject:     "用户偏好",
			Summary:     "用户不喜欢" + m[1],
			Weight:      1.5,
			Confidence:  0.85,
			Triggers:    []string{m[1]},
			Source:      SourceLightRule,
			RuleID:      "like_dislike",
		})
	}

	// 规则5b: 喜好
	if m := reLike.FindStringSubmatch(text); m != nil {
		drafts = append(drafts, FactDraft{
			Domain:      "INNER_WORLD",
			Subcategory: "TASTES",
			Subject:     "用户偏好",
			Summary:     "用户喜欢" + m[1],
			Weight:      1.5,
			Confidence:  0.85,
			Triggers:    []string{m[1]},
			Source:      SourceLightRule,
			RuleID:      "like_dislike",
		})
	}

	// 规则6: 职业
	if m := reJob1.FindStringSubmatch(text); m != nil {
		drafts = append(drafts, FactDraft{
			Domain:      "IDENTITY",
			Subcategory: "BASIC_PROFILE",
			Subject:     "用户职业",
			Summary:     "用户从事" + m[1] + "相关",
			Weight:      2,
			Confidence:  0.85,
			Triggers:    []string{m[1]},
			Source:      SourceLightRule,
			RuleID:      "major_job",
		})
	} else if m := reJob2.FindStringSubmatch(text); m != nil {
		drafts = append(drafts, FactDraft{
			Domain:      "IDENTITY",
			Subcategory: "BASIC_PROFILE",
			Subject:     "用户职业",
			Summary:     "用户从事" + m[1] + "相关",
			Weight:      2,
			Confidence:  0.85,
			Triggers:    []string{m[1]},
			Source:      SourceLightRule,
			RuleID:      "major_job",
		})
	}

	// 规则7: 宠物
	if m := rePet1.FindStringSubmatch(text); m != nil {
		drafts = append(drafts, FactDraft{
			Domain:      "DAILY_LIFE",
			Subcategory: "LIVING_SPACE",
			Subject:     "用户宠物",
			Summary:     "用户养了宠物" + m[1],
			Weight:      2,
			Confidence:  0.9,
			Triggers:    []string{m[1], "宠物"},
			Source:      SourceLightRule,
			RuleID:      "pet",
		})
	} else if m := rePet2.FindStringSubmatch(text); m != nil {
		drafts = append(drafts, FactDraft{
			Domain:      "DAILY_LIFE",
			Subcategory: "LIVING_SPACE",
			Subject:     "用户宠物",
			Summary:     "用户养了宠物" + m[1],
			Weight:      2,
			Confidence:  0.9,
			Triggers:    []string{m[1], "宠物"},
			Source:      SourceLightRule,
			RuleID:      "pet",
		})
	}

	// 规则8: 承诺/计划
	if reCommitment.MatchString(text) {
		summary := text
		if len([]rune(summary)) > 80 {
			summary = string([]rune(summary)[:80])
		}
		drafts = append(drafts, FactDraft{
			Domain:      "TEMPORAL",
			Subcategory: "COMMITMENTS",
			Subject:     "用户承诺",
			Summary:     "用户提及计划或承诺：" + summary,
			Weight:      2,
			Confidence:  0.8,
			Triggers:    []string{"计划", "承诺"},
			Source:      SourceLightRule,
			RuleID:      "commitment",
		})
	}

	return drafts
}

// ─── 辅助函数 ──────────────────────────────────────────────────

// splitByPunct 按中文标点分句
func splitByPunct(text string) []string {
	re := regexp.MustCompile(`[，,；;]`)
	parts := re.Split(text, -1)
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// isValidChineseName 简单校验中文名（2-4字，不含标点数字）
func isValidChineseName(name string) bool {
	runes := []rune(name)
	if len(runes) < 2 || len(runes) > 4 {
		return false
	}
	for _, r := range runes {
		if r < 0x4E00 || r > 0x9FFF {
			// 非汉字但允许「·」（如少数民族名）
			if r != '·' {
				return false
			}
		}
	}
	return true
}

// HasUserFamilyLightHits 检查是否命中家人相关规则
func HasUserFamilyLightHits(userMsg string) bool {
	drafts := RunLightExtractRules(userMsg)
	for _, d := range drafts {
		if d.FamilyScope == "user" || d.Subcategory == "FAMILY" {
			return true
		}
	}
	return false
}

// FactDraftsToRows 将 FactDraft 转为 ExtractedFactRow
func FactDraftsToRows(drafts []FactDraft) []ExtractedFactRow {
	rows := make([]ExtractedFactRow, 0, len(drafts))
	for _, d := range drafts {
		rows = append(rows, ExtractedFactRow{
			Domain:      d.Domain,
			Subcategory: d.Subcategory,
			Subject:     d.Subject,
			Summary:     d.Summary,
			Weight:      d.Weight,
			Confidence:  d.Confidence,
			Triggers:    d.Triggers,
			AgeMeta:     d.AgeMeta,
		})
	}
	return rows
}

// ExtractedFactRow 提取的事实行
type ExtractedFactRow struct {
	Domain      string   `json:"domain"`
	Subcategory string   `json:"subcategory"`
	Subject     string   `json:"subject"`
	Summary     string   `json:"summary"`
	Weight      float64  `json:"weight"`
	Confidence  float64  `json:"confidence"`
	Triggers    []string `json:"triggers,omitempty"`
	AgeMeta     *AgeMeta `json:"ageMeta,omitempty"`
}
