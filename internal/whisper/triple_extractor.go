// Package whisper — triple_extractor.go
// 100% 对齐 ackem memory/tripleExtractor.ts
// 启发式正则 + 结构化三元组提取（likes/dislikes/亲属/宠物/职业等）

package whisper

import (
	"regexp"
	"strings"
)

// TripleRow 三元组行
type TripleRow struct {
	Subject       string   `json:"subject"`
	Predicate     string   `json:"predicate"`
	Object        string   `json:"object"`
	Confidence    float64  `json:"confidence"`
	SourceFactIDs []string `json:"sourceFactIds"`
}

// attachEmotion 把事实情感快照落进三元组（v4.9 图谱情绪维度）。无快照时保持
// 空值（中性）；情绪标签由效价确定性派生（正面/负面/中性），情绪强度/效价原样
// 携带——情绪从此不在图外（审计 §C：「情绪活在图外 EmotionState」）。
func attachEmotion(t Triple, ec *EmotionalContext) Triple {
	if ec == nil {
		return t
	}
	t.EmotionalIntensity = ec.Intensity
	t.Valence = ec.Valence
	switch {
	case ec.Valence > 0.15:
		t.EmotionLabel = "正面"
	case ec.Valence < -0.15:
		t.EmotionLabel = "负面"
	default:
		t.EmotionLabel = "中性"
	}
	return t
}

// ─── 因果维度（v4.9，审计 §C「无因果维度」欠账收口）──────────────

// causalPatterns 确定性因果模式：中文因果连接词 → 「导致」谓词。
// 同一事实可命中多条（分别成边）；模式经测试锁定，避免误提取。
var causalPatterns = []struct {
	re        *regexp.Regexp
	predicate string
}{
	{regexp.MustCompile(`因为(.{1,24})[，,](?:所以)?(.{1,48})`), "导致"},
	{regexp.MustCompile(`由于(.{1,24})[，,](?:所以)?(.{1,48})`), "导致"},
	{regexp.MustCompile(`(.{1,24})(?:导致|引发|造成)(.{1,48})`), "导致"},
	{regexp.MustCompile(`(.{1,24})(?:让我|使我|令我)(.{1,48})`), "导致"},
}

// extractCausalTriples 从事实摘要提取因果三元组：{因, 导致, 果}，情绪经
// attachEmotion 随事实落图。无模式命中返回空。
func extractCausalTriples(f *Fact) []Triple {
	var out []Triple
	for _, p := range causalPatterns {
		for _, m := range p.re.FindAllStringSubmatch(f.Summary, -1) {
			if len(m) < 3 {
				continue
			}
			cause := strings.TrimSpace(m[1])
			effect := strings.TrimSpace(m[2])
			// 直接式（X导致Y）可能把「因为/由于」带进因侧，剥掉连接词
			cause = strings.TrimPrefix(cause, "因为")
			cause = strings.TrimPrefix(cause, "由于")
			cause = strings.TrimSpace(cause)
			if len([]rune(cause)) < 2 || len([]rune(effect)) < 2 {
				continue
			}
			out = append(out, attachEmotion(Triple{
				Subject: cause, Predicate: p.predicate, Object: effect,
				Confidence: f.Confidence, SourceFactIDs: []string{f.ID},
			}, f.EmotionalContext))
		}
	}
	return out
}

// 正则模式
var triplePatterns = []struct {
	re        *regexp.Regexp
	predicate string
}{
	{regexp.MustCompile(`(?:用户|他|她|我)?喜欢|爱好|热衷于`), "likes"},
	{regexp.MustCompile(`(?:用户|他|她|我)?讨厌|不喜欢|厌恶|反感|排斥`), "dislikes"},
	{regexp.MustCompile(`(?:用户|他|她|我)?在(.{1,12})(?:工作|上班|任职)`), "works_at"},
	{regexp.MustCompile(`(?:用户|他|她|我)?是(.{1,8})[职岗]`), "is_a"},
	{regexp.MustCompile(`(?:用户|他|她|我)?住在|居住.?在(.{1,12})`), "lives_in"},
	{regexp.MustCompile(`(?:用户|他|她|我)?来自(.{1,12})`), "from"},
	{regexp.MustCompile(`(?:用户|他|她|我)?养了|养着|有一只?(.{1,8})(?:猫|狗|宠物)`), "has_pet"},
	{regexp.MustCompile(`(?:用户|他|她|我)?去过(.{1,12})旅行|旅游`), "traveled_to"},
}

// ExtractTriples 从事实 subject+summary 中提取三元组
func ExtractTriples(subject, summary, factID string, subcategory string, birthdayMMDD string) []TripleRow {
	text := subject + " " + summary

	// 结构化提取
	results := extractStructuredTriples(subject, summary, factID, subcategory, birthdayMMDD)

	// 正则启发式
	cleanSubject := strings.ReplaceAll(subject, "用户", "用户")
	cleanSubject = strings.ReplaceAll(cleanSubject, "他", "用户")
	cleanSubject = strings.ReplaceAll(cleanSubject, "她", "用户")
	cleanSubject = strings.ReplaceAll(cleanSubject, "我", "用户")
	cleanSubject = truncateStr(cleanSubject, 30)

	for _, p := range triplePatterns {
		matches := p.re.FindAllStringSubmatch(text, -1)
		for _, m := range matches {
			obj := ""
			if len(m) > 1 && m[1] != "" {
				obj = m[1]
			} else {
				obj = m[0]
			}
			obj = strings.ReplaceAll(obj, "用户", "")
			obj = strings.ReplaceAll(obj, "他", "")
			obj = strings.ReplaceAll(obj, "她", "")
			obj = strings.ReplaceAll(obj, "我", "")
			obj = strings.TrimSpace(obj)
			if len([]rune(obj)) >= 1 && len([]rune(obj)) <= 20 {
				results = append(results, TripleRow{
					Subject: cleanSubject, Predicate: p.predicate, Object: obj,
					Confidence: 0.6, SourceFactIDs: []string{factID},
				})
			}
		}
	}

	return results
}

func extractStructuredTriples(subject, summary, factID, subcategory, birthdayMMDD string) []TripleRow {
	var results []TripleRow
	text := subject + " " + summary

	// 生日
	if birthdayMMDD != "" {
		results = append(results, TripleRow{
			Subject: "用户", Predicate: "has_birthday", Object: birthdayMMDD,
			Confidence: 0.95, SourceFactIDs: []string{factID},
		})
	}

	// 家属
	familyMap := []struct {
		re     *regexp.Regexp
		member string
	}{
		{regexp.MustCompile(`母亲|妈妈|妈`), "母亲"},
		{regexp.MustCompile(`父亲|爸爸|爸`), "父亲"},
		{regexp.MustCompile(`奶奶|祖母`), "奶奶"},
		{regexp.MustCompile(`爷爷|祖父`), "爷爷"},
	}

	if subcategory == "FAMILY" || strings.Contains(text, "生日") {
		for _, fm := range familyMap {
			if fm.re.MatchString(text) {
				results = append(results, TripleRow{
					Subject: "用户", Predicate: "family_member", Object: fm.member,
					Confidence: 0.9, SourceFactIDs: []string{factID},
				})
				// 提取家属生日
				birthRe := regexp.MustCompile(`(\d{1,2})月(\d{1,2})`)
				if m := birthRe.FindStringSubmatch(summary); len(m) >= 3 {
					mmdd := m[1]
					if len(mmdd) == 1 {
						mmdd = "0" + mmdd
					}
					dd := m[2]
					if len(dd) == 1 {
						dd = "0" + dd
					}
					results = append(results, TripleRow{
						Subject: fm.member, Predicate: "has_birthday", Object: mmdd + "-" + dd,
						Confidence: 0.85, SourceFactIDs: []string{factID},
					})
				}
			}
		}
	}

	// 宠物
	if subcategory == "LIVING_SPACE" && (strings.Contains(text, "宠物") || strings.Contains(text, "猫") || strings.Contains(text, "狗")) {
		petRe := regexp.MustCompile(`宠物([\p{Han}\w]{1,8})`)
		if m := petRe.FindStringSubmatch(summary); len(m) >= 2 {
			results = append(results, TripleRow{
				Subject: "用户", Predicate: "has_pet", Object: m[1],
				Confidence: 0.85, SourceFactIDs: []string{factID},
			})
		} else {
			altRe := regexp.MustCompile(`养了([\p{Han}\w]{1,8})`)
			if m := altRe.FindStringSubmatch(summary); len(m) >= 2 {
				results = append(results, TripleRow{
					Subject: "用户", Predicate: "has_pet", Object: m[1],
					Confidence: 0.8, SourceFactIDs: []string{factID},
				})
			}
		}
	}

	// 职业
	if subcategory == "BASIC_PROFILE" && strings.Contains(subject, "职业") {
		job := strings.ReplaceAll(summary, "用户从事", "")
		job = strings.ReplaceAll(job, "相关", "")
		job = strings.TrimSpace(job)
		if job != "" {
			results = append(results, TripleRow{
				Subject: "用户", Predicate: "is_a", Object: job,
				Confidence: 0.85, SourceFactIDs: []string{factID},
			})
		}
	}

	return results
}
