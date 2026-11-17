package permission

import (
	"strings"
	"sync"
)

// argv 前缀规则（codex 蒸馏 T10，execpolicy prefix_rule 同构）：命令类工具
// 的规则粒度问题——glob 是字符串级匹配，`bash(git status*)` 会把
// `git status; rm -rf ~` 一起放行（无法区分「后面跟的是参数还是另一条命令」）。
// argv 前缀匹配天然没有这个问题：规则只匹配「每个 shell 段的 token 前缀都
// 命中」的命令。
//
// 纪律（照抄上游，v1 收敛）：
//   - 首 token 建索引（热路径只比对同程序下的几条规则）；
//   - 只做 Single token（Alts 多选一先不做，可读性优先）；
//   - 匹配语义按「整条命令的所有段」判定——任一段不命中即不匹配，
//     这是防串接提权的核心。

// ArgvRule 是一条 argv 前缀规则：First 是命令首 token，Rest 是后续 token
// 前缀（全部精确匹配）。
type ArgvRule struct {
	First string
	Rest  []string
}

// ParseArgvRuleText 解析配置/落盘一行式：`ToolName prefix ["tok",...]`。
// 返回工具名与规则；ok=false 表示格式不合法。
func ParseArgvRuleText(s string) (string, ArgvRule, bool) {
	i := strings.Index(s, " prefix [")
	if i <= 0 {
		return "", ArgvRule{}, false
	}
	tool := strings.TrimSpace(s[:i])
	if tool == "" || strings.ContainsAny(tool, " \t()") {
		return "", ArgvRule{}, false
	}
	r, ok := ParseArgvRule(s[i+1:])
	if !ok {
		return "", ArgvRule{}, false
	}
	return tool, r, true
}

// ArgvRuleText 渲染带工具名的一行式（落盘用）。
func ArgvRuleText(tool string, r ArgvRule) string {
	return tool + " " + r.String()
}

// ParseArgvRule 解析一行式规则文本：`prefix ["git","status","--porcelain"]`。
// ok=false 表示格式不合法（调用方告警并丢弃，不静默装一条匹配不到的规则）。
func ParseArgvRule(s string) (ArgvRule, bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "prefix ") {
		return ArgvRule{}, false
	}
	list := strings.TrimSpace(strings.TrimPrefix(s, "prefix "))
	if !strings.HasPrefix(list, "[") || !strings.HasSuffix(list, "]") {
		return ArgvRule{}, false
	}
	inner := list[1 : len(list)-1]
	var toks []string
	for _, part := range strings.Split(inner, ",") {
		tok := strings.Trim(strings.TrimSpace(part), `"`)
		if tok == "" {
			return ArgvRule{}, false
		}
		toks = append(toks, tok)
	}
	if len(toks) == 0 {
		return ArgvRule{}, false
	}
	return ArgvRule{First: toks[0], Rest: toks[1:]}, true
}

// String 渲染回一行式（落盘/日志用）。
func (r ArgvRule) String() string {
	quoted := make([]string, 0, 1+len(r.Rest))
	quote := func(t string) string { return `"` + t + `"` }
	quoted = append(quoted, quote(r.First))
	for _, t := range r.Rest {
		quoted = append(quoted, quote(t))
	}
	return "prefix [" + strings.Join(quoted, ",") + "]"
}

// ArgvRuleFromCommand 从一条命令串构造规则（「记住这条命令」的生成端）：
// 取 shell 分段后第一段的前 maxTokens 个 token。命令含 glob 元字符（* ? [）
// 或无法安全切分时 ok=false——glob 语义的 subject 继续走既有 glob/精确规则
// 路径，不猜。
func ArgvRuleFromCommand(command string, maxTokens int) (ArgvRule, bool) {
	if strings.ContainsAny(command, "*?[") {
		return ArgvRule{}, false
	}
	segs := splitShellSegments(command)
	if len(segs) == 0 {
		return ArgvRule{}, false
	}
	toks := tokenizeArgv(segs[0])
	if len(toks) == 0 {
		return ArgvRule{}, false
	}
	if maxTokens <= 0 || maxTokens > len(toks) {
		maxTokens = len(toks)
	}
	if toks[0] == "" {
		return ArgvRule{}, false
	}
	return ArgvRule{First: toks[0], Rest: append([]string(nil), toks[1:maxTokens]...)}, true
}

// Matches 报告整条命令串是否命中该规则：按 shell 分段，每段的 token 前缀都
// 必须命中——`git status; rm -rf x` 的第二段不命中 ["git","status"] 规则，
// 这正是它比 glob 安全的地方。
func (r ArgvRule) Matches(command string) bool {
	segs := splitShellSegments(command)
	if len(segs) == 0 {
		return false
	}
	for _, seg := range segs {
		toks := tokenizeArgv(seg)
		if len(toks) == 0 || toks[0] != r.First {
			return false
		}
		for i, want := range r.Rest {
			if i+1 >= len(toks) || toks[i+1] != want {
				return false
			}
		}
	}
	return true
}

// ArgvRuleSet 是带首 token 索引的规则集（并发安全）。
type ArgvRuleSet struct {
	mu    sync.RWMutex
	rules map[string][]ArgvRule // 首 token → 规则组
}

// NewArgvRuleSet 构造空集。
func NewArgvRuleSet() *ArgvRuleSet {
	return &ArgvRuleSet{rules: map[string][]ArgvRule{}}
}

// Add 追加一条规则。
func (s *ArgvRuleSet) Add(r ArgvRule) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.rules == nil {
		s.rules = map[string][]ArgvRule{}
	}
	s.rules[r.First] = append(s.rules[r.First], r)
}

// Clear 清空会话级规则。保留同一个集合指针，让已经注入 Gate 的引用同步失效
// ——/new 轮换时不能只换 Controller 字段，否则旧 Gate 仍会命中旧授权。
func (s *ArgvRuleSet) Clear() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.rules = map[string][]ArgvRule{}
	s.mu.Unlock()
}

// Len 返回规则总数（测试与诊断用）。
func (s *ArgvRuleSet) Len() int {
	if s == nil {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, rs := range s.rules {
		n += len(rs)
	}
	return n
}

// Allowed 报告命令是否被某条规则放行。首 token 索引：只比对同程序下的
// 几条规则，不扫全表。
func (s *ArgvRuleSet) Allowed(command string) bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.rules) == 0 {
		return false
	}
	segs := splitShellSegments(command)
	if len(segs) == 0 {
		return false
	}
	first := tokenizeArgv(segs[0])
	if len(first) == 0 {
		return false
	}
	for _, r := range s.rules[first[0]] {
		if r.Matches(command) {
			return true
		}
	}
	return false
}

// splitShellSegments 把命令串按 shell 串接符分段（; && || | 与换行）——
// 串接后的每一段都是独立命令，规则必须对每段分别成立。
func splitShellSegments(cmd string) []string {
	var out []string
	for _, seg := range strings.FieldsFunc(cmd, func(r rune) bool {
		return r == ';' || r == '|' || r == '\n' || r == '&'
	}) {
		seg = strings.TrimSpace(seg)
		if seg != "" {
			out = append(out, seg)
		}
	}
	return out
}

// tokenizeArgv 把单个命令段切成 token（空白分隔，shell 引号剥除一层；
// 引号不配对按字面处理——保守，宁可不匹配不误放行）。
func tokenizeArgv(seg string) []string {
	var out []string
	var cur strings.Builder
	var quote rune
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for _, r := range seg {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
				continue
			}
			cur.WriteRune(r)
		case r == '"' || r == '\'':
			quote = r
		case r == ' ' || r == '\t':
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}
