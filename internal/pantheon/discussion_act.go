package pantheon

import (
	"regexp"
	"strings"
)

var (
	discussionActMarker   = regexp.MustCompile(`(?im)^\s*(?:[-*>]\s*)*(?:动作|讨论动作|发言动作|discussion action|discussion act)\s*[:：]\s*(.+)$`)
	actTargetMarker       = regexp.MustCompile(`(?im)^\s*(?:[-*>]\s*)*(?:目标贡献|贡献目标|target contribution)\s*[:：]\s*(.*)$`)
	actTargetIssueMarker  = regexp.MustCompile(`(?im)^\s*(?:[-*>]\s*)*(?:目标分歧|目标台账|issue target|target issue)\s*[:：]\s*(.*)$`)
	challengedClaimMarker = regexp.MustCompile(`(?im)^\s*(?:[-*>]\s*)*(?:被挑战命题|挑战命题|challenged claim|claim under challenge)\s*[:：]\s*(.*)$`)
	actEvidenceMarker     = regexp.MustCompile(`(?im)^\s*(?:[-*>]\s*)*(?:证据句柄|动作证据|action evidence)\s*[:：]\s*(.*)$`)
	actResponseMarker     = regexp.MustCompile(`(?im)^\s*(?:[-*>]\s*)*(?:回应结论|回应结果|response verdict)\s*[:：]\s*(.*)$`)
)

// discussionActDetails is the small, provider-neutral subset of a discussion
// action that the host can trust. Raw prose remains authoritative; these fields
// only make relationships and warnings machine-readable.
type discussionActDetails struct {
	Text                 string
	RawAction            string
	DiscussionAct        string
	TargetContributionID string
	TargetIssueID        string
	ChallengedClaim      string
	EvidenceHandle       string
	ResponseVerdict      string
}

func extractDiscussionActDetails(text string) discussionActDetails {
	lines := strings.Split(text, "\n")
	clean := make([]string, 0, len(lines))
	result := discussionActDetails{}
	for _, line := range lines {
		if match := discussionActMarker.FindStringSubmatch(line); len(match) > 1 {
			result.RawAction = strings.TrimSpace(match[1])
			result.DiscussionAct = normalizeDiscussionAct(result.RawAction)
			continue
		}
		if match := actTargetMarker.FindStringSubmatch(line); len(match) > 1 {
			result.TargetContributionID = cleanProtocolID(match[1])
			continue
		}
		if match := actTargetIssueMarker.FindStringSubmatch(line); len(match) > 1 {
			result.TargetIssueID = cleanProtocolID(match[1])
			continue
		}
		if match := challengedClaimMarker.FindStringSubmatch(line); len(match) > 1 {
			result.ChallengedClaim = trimRunes(strings.TrimSpace(match[1]), 600)
			continue
		}
		if match := actEvidenceMarker.FindStringSubmatch(line); len(match) > 1 {
			result.EvidenceHandle = trimRunes(strings.TrimSpace(match[1]), 1200)
			continue
		}
		if match := actResponseMarker.FindStringSubmatch(line); len(match) > 1 {
			result.ResponseVerdict = normalizeResponseVerdict(match[1])
			continue
		}
		clean = append(clean, line)
	}
	result.Text = strings.TrimSpace(strings.Join(clean, "\n"))
	return result
}

func normalizeDiscussionAct(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.Trim(value, "`'\"。.;；:：()（）[]【】")
	switch value {
	case "主张", "提出", "claim":
		return DiscussionActClaim
	case "质疑", "质询", "挑战", "challenge":
		return DiscussionActChallenge
	case "回应", "回答", "respond", "reply":
		return DiscussionActRespond
	case "修正", "修订", "revise", "revision":
		return DiscussionActRevise
	case "补强", "支持", "reinforce", "support":
		return DiscussionActReinforce
	case "拓宽", "扩展", "broaden", "expand":
		return DiscussionActBroaden
	default:
		return ""
	}
}

func normalizeResponseVerdict(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.Trim(value, "`'\"。.;；:：()（）[]【】")
	switch value {
	case "接受", "同意", "accept", "accepted":
		return ResponseAccept
	case "部分接受", "有条件接受", "partial", "partially accept":
		return ResponsePartiallyAccept
	case "拒绝", "不接受", "reject", "rejected":
		return ResponseReject
	default:
		return ""
	}
}

func cleanProtocolID(raw string) string {
	value := strings.TrimSpace(raw)
	value = strings.Trim(value, "`'\"。.;；,，")
	if fields := strings.Fields(value); len(fields) > 0 {
		value = strings.Trim(fields[0], "`'\"。.;；,，")
	}
	return value
}

func discussionActNeedsTarget(act string) bool {
	switch act {
	case DiscussionActChallenge, DiscussionActRespond, DiscussionActRevise:
		return true
	default:
		return false
	}
}

func discussionActLabel(act string) string {
	switch act {
	case DiscussionActClaim:
		return "主张"
	case DiscussionActChallenge:
		return "质疑"
	case DiscussionActRespond:
		return "回应"
	case DiscussionActRevise:
		return "修正"
	case DiscussionActReinforce:
		return "补强"
	case DiscussionActBroaden:
		return "拓宽"
	default:
		return act
	}
}

func responseVerdictLabel(verdict string) string {
	switch verdict {
	case ResponseAccept:
		return "接受"
	case ResponsePartiallyAccept:
		return "部分接受"
	case ResponseReject:
		return "拒绝"
	default:
		return verdict
	}
}
