package memory

import (
	"regexp"
)

// 记忆脱敏（codex 蒸馏 T12 护栏一）：模型沉淀的事实可能夹带真实密钥，
// 落库前按四条正则替换为占位符。替换顺序先具体后宽泛（Bearer → sk- → AKIA
// → key=赋值），避免宽泛条把具体条的键名一起吃掉。

const (
	// redactedPlaceholder 是脱敏占位符：不记录原文，只留下「这里曾有密钥」。
	redactedPlaceholder = "[REDACTED_SECRET]"
)

var (
	// reBearer 匹配 HTTP 认证头形态（Bearer + token 体）。
	reBearer = regexp.MustCompile(`(?i)\bBearer[ \t]+[A-Za-z0-9._~+/-]{16,}=*`)
	// reOpenAI 匹配 sk- 开头的长密钥（含 sk-proj- 形态的连字符段）。
	reOpenAI = regexp.MustCompile(`sk-[A-Za-z0-9_-]{20,}`)
	// reAWS 匹配 AWS AccessKeyId 形态。
	reAWS = regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)
	// reAssign 匹配 key=值 / "token": "值" / 'secret': '值' 形态：保留键名、
	// 键侧引号、分隔符与值侧引号，只换值——记忆里仍能看出「这里有个 api_key
	// 字段」，但拿不到值。
	reAssign = regexp.MustCompile(`(?i)\b(api[_-]?key|token|secret|password)\b(["']?)(\s*[:=]\s*)(["']?)[^\s"']{8,}`)
)

// Sanitize 按四条正则脱敏，返回替换后的文本与替换次数（计数进日志，不记原文）。
func Sanitize(s string) (string, int) {
	n := 0
	for _, re := range []*regexp.Regexp{reBearer, reOpenAI, reAWS} {
		s = re.ReplaceAllStringFunc(s, func(string) string {
			n++
			return redactedPlaceholder
		})
	}
	s = reAssign.ReplaceAllStringFunc(s, func(m string) string {
		n++
		sub := reAssign.FindStringSubmatch(m)
		// 1=键名 2=键侧引号 3=分隔符 4=值侧引号。
		return sub[1] + sub[2] + sub[3] + sub[4] + redactedPlaceholder
	})
	return s, n
}

// sanitizeMemory 就地脱敏一条记忆的文本字段（Title/Description/Body 都是
// 模型产出文本），返回替换总次数。
func sanitizeMemory(m *Memory) int {
	total := 0
	s, n := Sanitize(m.Title)
	m.Title, total = s, total+n
	s, n = Sanitize(m.Description)
	m.Description, total = s, total+n
	s, n = Sanitize(m.Body)
	m.Body, total = s, total+n
	return total
}
