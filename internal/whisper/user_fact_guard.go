// Package whisper — user_fact_guard.go
// 100% 对齐 ackem memory/userFactGuard.ts
// 用户事实抽取守卫：只从用户自述写入档案，问句/gaea自述不得污染用户 BASIC_PROFILE

package whisper

import (
	"regexp"
	"strings"
)

// questionToCompanionREs 问句→gaea检测正则
var questionToCompanionREs = []*regexp.Regexp{
	regexp.MustCompile(`^你(?:是|叫|谁|名字|生日|多大|几岁|哪年)`),
	regexp.MustCompile(`^请问?你(?:的)?(?:生日|名字|是谁)`),
	regexp.MustCompile(`你(?:是|叫)什么`),
	regexp.MustCompile(`是谁[啊呀吗呢]?[？?]?$`),
	regexp.MustCompile(`什么时候[啊呀吗呢]?[？?]?$`),
	regexp.MustCompile(`多大[了]?[啊呀吗呢]?[？?]?$`),
}

var interrogativeNameRE = regexp.MustCompile(`^[谁什么啥哪怎么为何几个]+$`)
var refusalNameRE = regexp.MustCompile(`^(随便|不想|不说|保密|不告诉你|无可奉告)`)

// IsQuestionToCompanion 判断是否为对gaea的提问
func IsQuestionToCompanion(msg string) bool {
	t := strings.TrimSpace(msg)
	if t == "" {
		return false
	}
	if strings.HasSuffix(t, "？") || strings.HasSuffix(t, "?") {
		return true
	}
	for _, re := range questionToCompanionREs {
		if re.MatchString(t) {
			return true
		}
	}
	return false
}

// UserMsgClaimsSelfBirthday 用户消息是否自述生日
func UserMsgClaimsSelfBirthday(msg string) bool {
	return regexp.MustCompile(`(?:^|[^你])我(?:本人)?(?:的)?生日(?:是|在)?`).MatchString(msg) ||
		regexp.MustCompile(`\bmy birthday\b`).MatchString(strings.ToLower(msg))
}

// UserMsgClaimsSelfName 用户消息是否自述姓名
func UserMsgClaimsSelfName(msg string) bool {
	return regexp.MustCompile(`(?:我(?:叫|是|名字)|叫我|你可以叫我|大家都叫我|名字[是叫])`).MatchString(msg)
}

// IsValidExtractedUserName 校验抽取的用户名是否合法
func IsValidExtractedUserName(name, userMsg string) bool {
	n := strings.TrimSpace(name)
	if n == "" || len([]rune(n)) > 10 {
		return false
	}
	if interrogativeNameRE.MatchString(n) {
		return false
	}
	if regexp.MustCompile(`^[谁什么啥你他她]`).MatchString(n) {
		return false
	}
	if refusalNameRE.MatchString(n) {
		return false
	}
	if IsQuestionToCompanion(userMsg) {
		return false
	}
	return true
}

// GuardableFact 可守卫事实接口
type GuardableFact struct {
	Domain      string `json:"domain,omitempty"`
	Subcategory string `json:"subcategory"`
	Subject     string `json:"subject"`
	Summary     string `json:"summary"`
}

// FilterExtractedUserFacts LLM/规则抽取后二次过滤：用户档案只接受用户自述
func FilterExtractedUserFacts(facts []GuardableFact, userMsg string) []GuardableFact {
	questionTurn := IsQuestionToCompanion(userMsg)

	var result []GuardableFact
	for _, f := range facts {
		if f.Subcategory == "NOTE" {
			result = append(result, f)
			continue
		}
		if f.Subcategory == "OUR_BOND" && strings.HasPrefix(f.Subject, "Ackem回复") {
			result = append(result, f)
			continue
		}

		if questionTurn && f.Subcategory == "BASIC_PROFILE" {
			continue
		}

		if f.Subcategory == "BASIC_PROFILE" {
			if f.Subject == "用户生日" && !UserMsgClaimsSelfBirthday(userMsg) {
				continue
			}
			if (f.Subject == "用户姓名" || f.Subject == "用户昵称") && !UserMsgClaimsSelfName(userMsg) {
				continue
			}
		}

		result = append(result, f)
	}
	return result
}

// FilterExtractedUserMemoryFacts MemoryFact 版本的用户事实守卫
func FilterExtractedUserMemoryFacts(facts []*MemoryFact, userMsg string) []*MemoryFact {
	questionTurn := IsQuestionToCompanion(userMsg)

	var result []*MemoryFact
	for _, f := range facts {
		if f.Subcategory == "NOTE" {
			result = append(result, f)
			continue
		}
		if f.Subcategory == "OUR_BOND" && strings.HasPrefix(f.Subject, "Ackem回复") {
			result = append(result, f)
			continue
		}

		if questionTurn && f.Subcategory == "BASIC_PROFILE" {
			continue
		}

		if f.Subcategory == "BASIC_PROFILE" {
			if f.Subject == "用户生日" && !UserMsgClaimsSelfBirthday(userMsg) {
				continue
			}
			if (f.Subject == "用户姓名" || f.Subject == "用户昵称") && !UserMsgClaimsSelfName(userMsg) {
				continue
			}
		}

		result = append(result, f)
	}
	return result
}
