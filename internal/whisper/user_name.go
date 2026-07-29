// Package whisper — user_name.go
// 100% 对齐 ackem memory/userName.ts
// 用户名字记忆：从事实库查询/解析用户姓名/昵称

package whisper

import (
	"regexp"
	"sort"
	"strings"
)

// ResolvePreferredName 取当前首选名字（按 weight 降序→updatedAt 降序）
func ResolvePreferredName(store *FactStore) string {
	var nameFacts []*Fact
	for _, f := range store.ListActive() {
		if f.Subcategory == "BASIC_PROFILE" && (f.Subject == "用户姓名" || f.Subject == "用户昵称") && f.Weight > 0 {
			nameFacts = append(nameFacts, f)
		}
	}
	if len(nameFacts) == 0 {
		return ""
	}
	sort.Slice(nameFacts, func(i, j int) bool {
		if nameFacts[i].Weight != nameFacts[j].Weight {
			return nameFacts[i].Weight > nameFacts[j].Weight
		}
		return nameFacts[i].UpdatedAt.After(nameFacts[j].UpdatedAt)
	})
	return cleanUserName(nameFacts[0].Summary)
}

// ResolveAllNames 取所有名字（按权重降序）
func ResolveAllNames(store *FactStore) []struct {
	Name    string
	Weight  float64
	Subject string
} {
	var result []struct {
		Name    string
		Weight  float64
		Subject string
	}
	for _, f := range store.ListActive() {
		if f.Subcategory == "BASIC_PROFILE" && (f.Subject == "用户姓名" || f.Subject == "用户昵称") {
			result = append(result, struct {
				Name    string
				Weight  float64
				Subject string
			}{Name: cleanUserName(f.Summary), Weight: f.Weight, Subject: f.Subject})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Weight > result[j].Weight })
	return result
}

// ShouldAskUserName 是否需要主动询问用户名字
func ShouldAskUserName(store *FactStore) bool {
	for _, f := range store.ListActive() {
		if f.Subcategory == "BASIC_PROFILE" && (f.Subject == "用户姓名" || f.Subject == "用户昵称") && f.Weight > 0 {
			return false
		}
	}
	return true
}

func cleanUserName(summary string) string {
	re := regexp.MustCompile(`^用户(叫|喜欢被叫|昵称是|的小名叫|的英文名是)`)
	return strings.TrimSpace(re.ReplaceAllString(summary, ""))
}

// ExtractNameByRegex 规则层提取名字
func ExtractNameByRegex(text string) (name string, confidence float64) {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`叫我([\p{Han}\w]{1,10})就好`),
		regexp.MustCompile(`你可以叫我([\p{Han}\w]{1,10})`),
		regexp.MustCompile(`大家都叫我([\p{Han}\w]{1,10})`),
		regexp.MustCompile(`我叫[叫是]?([\p{Han}\w]{1,10})`),
		regexp.MustCompile(`叫我([\p{Han}\w]{1,10})`),
		regexp.MustCompile(`我是([\p{Han}\w]{1,10})`),
		regexp.MustCompile(`名字[是叫]([\p{Han}\w]{1,10})`),
	}
	for _, re := range patterns {
		if m := re.FindStringSubmatch(text); len(m) > 1 {
			name := strings.TrimSpace(m[1])
			if IsValidExtractedUserName(name, text) {
				return name, 1.0
			}
		}
	}
	// 短文本回退
	trimmed := strings.TrimSpace(text)
	runes := []rune(trimmed)
	if len(runes) >= 1 && len(runes) <= 4 &&
		!regexp.MustCompile(`[，。！？?]`).MatchString(trimmed) &&
		!regexp.MustCompile(`[谁什么啥哪怎么为何几个你他她]`).MatchString(trimmed) {
		if IsValidExtractedUserName(trimmed, text) {
			return trimmed, 0.9
		}
	}
	return "", 0
}

// GetAskNamePrompt 人格化主动询问名字
func GetAskNamePrompt(personalityID string) string {
	prompts := map[string]string{
		"tsundere": "喂，我总不能一直叫你'你'吧？……你叫什么？才不是想知道呢。",
		"kuudere":  "……你叫什么？",
		"deredere": "对了，我还不知道你的名字呢。你希望我怎么称呼你？",
		"yandere":  "你……叫什么？我需要知道。",
		"genki":    "诶~我们聊了这么久，我还不知道你叫什么呢！告诉我嘛~",
	}
	if p, ok := prompts[personalityID]; ok {
		return p
	}
	return "对了，我还不知道你的名字呢。你希望我怎么称呼你？"
}

// BuildUserNameLine 日记注入用的用户名字行
func BuildUserNameLine(store *FactStore) string {
	preferred := ResolvePreferredName(store)
	if preferred == "" {
		return "你不知道用户的名字。用'ta'称呼。"
	}
	all := ResolveAllNames(store)
	if len(all) <= 1 {
		return "你知道用户的名字：" + preferred + "。你可以叫ta的名字，也可以用你人格风格的称呼。"
	}
	var others []string
	for _, n := range all[1:] {
		if n.Name != preferred {
			others = append(others, n.Name)
		}
	}
	if len(others) > 0 {
		return "你知道用户的名字：" + preferred + "（ta也用过这些名字：" + strings.Join(others, "、") + "，但更喜欢被叫" + preferred + "）。"
	}
	return "你知道用户的名字：" + preferred + "。你可以叫ta的名字，也可以用你人格风格的称呼。"
}
