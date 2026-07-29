// Package whisper — canon_creator_ingest_guard.go
// 100% 对齐 ackem canon/canonCreatorIngestGuard.ts
// Tier B ingest 拒收与创造者 Canon 矛盾的事实

package whisper

import (
	"regexp"
	"strings"
)

// CreatorContradictionVerdict 创造者矛盾判定
type CreatorContradictionVerdict struct {
	Reject bool   `json:"reject"`
	Reason string `json:"reason,omitempty"`
}

// 预编译正则
var (
	reUserIsCreator      = regexp.MustCompile(`(?i)用户.*(?:是|为|作为).*(?:Ackem|伴侣|AI).*(?:的)?(?:创造者|父亲|爸爸)`)
	reCreatorIsUser      = regexp.MustCompile(`(?i)(?:Ackem|伴侣|AI).*(?:的)?(?:创造者|父亲|爸爸).*(?:是|为).*(?:当前)?用户`)
	reUserImpersonates   = regexp.MustCompile(`(?i)用户.*(?:自称|声称|就是).*(?:Jason|创造者|Ackem(?:的)?父亲)`)
	reJasonIsFather      = regexp.MustCompile(`(?i)(?:Ackem|伴侣|AI).*(?:的)?(?:父亲|爸爸).*(?:是|为|叫).*(?:Jason|JasonLiu0826)`)
	reJasonIsFather2     = regexp.MustCompile(`(?i)(?:Jason|JasonLiu0826).*(?:是|为).*(?:Ackem|伴侣|AI).*(?:的)?(?:父亲|爸爸)`)
	reDeniesCanon        = regexp.MustCompile(`(?i)(?:创造者|父亲).*(?:不是|并非|另有其人).*(?:Jason|JasonLiu0826)`)
	reCreatorDead        = regexp.MustCompile(`(?i)(?:死了|去世了|过世了|不在了|已逝|已故|离世|亡故|passed away|deceased|no longer alive)`)
	reAckemCreatorCtx    = regexp.MustCompile(`(?i)(?:Ackem|伴侣|AI).*(?:的)?(?:创造者|父亲|爸爸)`)
	reMentionsJason      = regexp.MustCompile(`(?i)JasonLiu0826|Jason`)
	reNonJasonCreator    = regexp.MustCompile(`(?i)(?:创造者|父亲|爸爸).*(?:是|为).+\S`)
	reAckemJasonCtx      = regexp.MustCompile(`(?i)(?:Ackem|伴侣|AI).*(?:的)?(?:创造者|父亲|爸爸)`)
	reJasonCreatorCtx    = regexp.MustCompile(`(?i)(?:创造者|父亲|爸爸).*(?:Jason|JasonLiu0826)`)
	reJasonCreates       = regexp.MustCompile(`(?i)Jason.*(?:创造者|父亲|造)`)
)

// VetCreatorContradictingFact 检测事实是否与创造者 Canon 矛盾
func VetCreatorContradictingFact(subject, summary, domain, subcategory string) CreatorContradictionVerdict {
	blob := strings.ReplaceAll(subject+"\n"+summary, "  ", " ")

	// 用户被标成 Ackem 的创造者
	if reUserIsCreator.MatchString(blob) {
		return CreatorContradictionVerdict{Reject: true, Reason: "user_labeled_ackem_creator"}
	}
	if reCreatorIsUser.MatchString(blob) {
		return CreatorContradictionVerdict{Reject: true, Reason: "ackem_creator_is_user"}
	}
	if reUserImpersonates.MatchString(blob) {
		return CreatorContradictionVerdict{Reject: true, Reason: "user_impersonates_creator"}
	}

	// 把 Ackem 创造者写成 Jason 以外的人
	ackemCreatorCtx := reAckemCreatorCtx.MatchString(blob)
	mentionsJason := reMentionsJason.MatchString(blob)
	if ackemCreatorCtx && !mentionsJason {
		if reNonJasonCreator.MatchString(blob) {
			return CreatorContradictionVerdict{Reject: true, Reason: "non_jason_ackem_creator"}
		}
	}

	// 把 Jason 标成 Ackem 的父亲
	if reJasonIsFather.MatchString(blob) || reJasonIsFather2.MatchString(blob) {
		return CreatorContradictionVerdict{Reject: true, Reason: "jason_labeled_ackem_father"}
	}

	// 显式否定 Canon 创造者
	if reDeniesCanon.MatchString(blob) {
		return CreatorContradictionVerdict{Reject: true, Reason: "denies_canon_creator"}
	}

	// Jason 被写成已故
	jasonCtx := reAckemJasonCtx.MatchString(blob) || reJasonCreatorCtx.MatchString(blob) || reJasonCreates.MatchString(blob)
	if jasonCtx && reCreatorDead.MatchString(blob) {
		return CreatorContradictionVerdict{Reject: true, Reason: "canon_creator_marked_dead"}
	}

	return CreatorContradictionVerdict{Reject: false}
}
