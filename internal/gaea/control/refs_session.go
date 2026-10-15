package control

// ── 跨会话 @引用（阶段十 10.3 session-reference，dsh 021 P2-B 取道）──────
//
// 原案（docs/gaea-dsh-021-core-distill-2026-10.md session-reference）：跨会话
// @提及 = host 文件发现 + 结构化引用请求 + 快照注入（label 取最新标题）。
// gaea 落地：@会话:<关键词> → 当前工作区会话目录按标题/文件名匹配（包含式，
// 大小写不敏感）→ 唯一命中才解析（多命中列出候选，宁缺勿误）→ 读事件日志
// 投影为紧凑快照（label 取 titles 注册表最新标题）。
//
// 与 @<路径> 的关系：@路径 引用的是文件本体；会话文件是原始事件日志 JSONL，
// 直接注入是乱象——本类注入的是**结构化投影**（user 全文 + assistant 截断），
// 头部带最新标题。显式「会话:」前缀避免与文件路径语义撞车。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gaea/gaea/internal/gaea/agent/session"
)

// sessionRefPrefix @引用会话的显式前缀（token 去掉 @ 后匹配）。
const sessionRefPrefix = "会话:"

// sessionRefPrefixEn 英文同义前缀。
const sessionRefPrefixEn = "session:"

// refSessionMaxBytes 会话快照注入上限（比文件引用紧——快照是摘要不是全文，
// 超长尾部以「…截断」说明，原文随时可重新引用）。
const refSessionMaxBytes = 32 * 1024

// sessionDirResolver 会话目录解析（var 以便测试注入；生产取控制器会话目录）。
var sessionDirResolver = func(c *Controller) string {
	if c == nil {
		return ""
	}
	return c.sessionDir
}

// looksLikeSessionRef token 是否为会话引用（显式前缀）。
func looksLikeSessionRef(token string) bool {
	return strings.HasPrefix(token, sessionRefPrefix) || strings.HasPrefix(token, sessionRefPrefixEn)
}

// sessionRefKey 提取关键词（去前导 @、去前缀、去空白）。
func sessionRefKey(token string) string {
	k := strings.TrimPrefix(token, "@")
	k = strings.TrimPrefix(k, sessionRefPrefix)
	k = strings.TrimPrefix(k, sessionRefPrefixEn)
	return strings.TrimSpace(k)
}

// loadSessionTitlesFor 读会话目录的标题注册表（.titles.json：{文件名: 标题}）。
// 文件缺失 = 空表正常态；损坏 = 空表（引用匹配是增强面，不因坏档报错）。
func loadSessionTitlesFor(dir string) map[string]string {
	out := map[string]string{}
	raw, err := os.ReadFile(filepath.Join(dir, ".titles.json"))
	if err != nil {
		return out
	}
	var m map[string]string
	if json.Unmarshal(raw, &m) != nil {
		return out
	}
	for k, v := range m {
		out[k] = strings.TrimSpace(v)
	}
	return out
}

// findSessionsByRef 按关键词匹配会话文件：标题包含（大小写不敏感）优先，
// 其次文件 base 名包含。返回唯一命中（文件路径+最新标题）或候选列表。
// matches>1 = 歧义（宁缺勿误：列出候选让作者精确化）；matches=0 = 未找到。
func findSessionsByRef(dir, key string) (path, title string, candidates []string, err error) {
	if strings.TrimSpace(key) == "" {
		return "", "", nil, fmt.Errorf("会话引用缺少关键词（@会话:<标题或文件名>）")
	}
	infos, err := session.List(dir)
	if err != nil {
		return "", "", nil, err
	}
	titles := loadSessionTitlesFor(dir)
	lowerKey := strings.ToLower(key)

	var titleHits, nameHits []session.Info
	for _, s := range infos {
		base := filepath.Base(s.Path)
		if t := titles[base]; t != "" && strings.Contains(strings.ToLower(t), lowerKey) {
			titleHits = append(titleHits, s)
			continue
		}
		if strings.Contains(strings.ToLower(base), lowerKey) {
			nameHits = append(nameHits, s)
		}
	}

	pick := func(list []session.Info) (string, string) {
		s := list[0]
		base := filepath.Base(s.Path)
		if t := titles[base]; t != "" {
			return s.Path, t
		}
		return s.Path, strings.TrimSuffix(base, ".jsonl")
	}

	switch {
	case len(titleHits) == 1:
		path, title := pick(titleHits)
		return path, title, nil, nil
	case len(titleHits) > 1:
		cands := make([]string, 0, len(titleHits))
		for _, s := range titleHits {
			base := filepath.Base(s.Path)
			if t := titles[base]; t != "" {
				cands = append(cands, t)
			}
		}
		return "", "", cands, nil
	case len(nameHits) == 1:
		path, title := pick(nameHits)
		return path, title, nil, nil
	case len(nameHits) > 1:
		cands := make([]string, 0, len(nameHits))
		for _, s := range nameHits {
			cands = append(cands, strings.TrimSuffix(filepath.Base(s.Path), ".jsonl"))
		}
		return "", "", cands, nil
	}
	return "", "", nil, fmt.Errorf("未找到匹配「%s」的会话", key)
}

// renderSessionSnapshot 把会话事件日志投影为紧凑快照：user 消息全文、
// assistant 文本截断（512 rune）、其余角色折叠为行计数；头部带最新标题。
// 尾部超 refSessionMaxBytes 截断并注明（原文随时可重新引用）。
func renderSessionSnapshot(path, title string) (string, error) {
	entries, err := session.ReadEntriesFor(path)
	if err != nil {
		return "", fmt.Errorf("读取会话失败: %w", err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "<referenced-session title=%q>\n", title)
	const asstCap = 512
	turns := 0
	for _, e := range entries {
		role, text, ok := sessionContentOf(e)
		if !ok {
			continue
		}
		if role == "user" {
			turns++
			fmt.Fprintf(&b, "\n[user] %s\n", text)
			continue
		}
		r := []rune(text)
		if len(r) > asstCap {
			text = string(r[:asstCap]) + "…（截断）"
		}
		fmt.Fprintf(&b, "\n[%s] %s\n", role, text)
	}
	// 尾部大小截断（rune 界）：超限砍尾并注明。
	out := b.String()
	if len(out) > refSessionMaxBytes {
		cut := refSessionMaxBytes
		for cut > 0 && !strings.HasPrefix(out[cut:], string(rune(0))) && (out[cut]&0xC0) == 0x80 {
			cut-- // 回退到 UTF-8 边界
		}
		out = out[:cut] + "\n…（快照超长，尾部截断；原文用 @会话 重新精确引用或打开会话查看）"
	}
	if turns == 0 {
		return "", fmt.Errorf("会话「%s」没有可投影的用户消息", title)
	}
	out += fmt.Sprintf("\n（该会话共 %d 条用户消息；以上为快照投影，非全文）\n", turns)
	return out, nil
}

// sessionContentOf 会话事件 → (role, text, ok)。口径与 sessionquery 的
// indexableContent 一致（KindUserMessage→user content / KindAssistantMessage→
// assistant text），此处只读重投影不建索引。
func sessionContentOf(e session.LogEntry) (string, string, bool) {
	switch e.Kind {
	case session.KindUserMessage:
		var p struct {
			Content string `json:"content"`
		}
		if json.Unmarshal(e.Payload, &p) != nil || strings.TrimSpace(p.Content) == "" {
			return "", "", false
		}
		return "user", strings.TrimSpace(p.Content), true
	case session.KindAssistantMessage:
		var p struct {
			Text string `json:"text"`
		}
		if json.Unmarshal(e.Payload, &p) != nil || strings.TrimSpace(p.Text) == "" {
			return "", "", false
		}
		return "assistant", strings.TrimSpace(p.Text), true
	}
	return "", "", false
}

// resolveSessionRef 解析一个 @会话:<关键词> 引用：匹配 → 快照注入。
// errs 收人可读错误（歧义列出候选）。返回注入块（空=未解析）。
func (c *Controller) resolveSessionRef(token string) (string, string) {
	key := sessionRefKey(token)
	dir := sessionDirResolver(c)
	if dir == "" {
		return "", "会话目录不可用"
	}
	path, title, cands, err := findSessionsByRef(dir, key)
	if err != nil {
		msg := err.Error()
		if len(cands) > 0 {
			msg += "；候选：" + strings.Join(cands, "、")
		}
		return "", msg
	}
	if path == "" {
		// 歧义（多命中）：findSessionsByRef 已把候选放进 cands，列出让作者精确化
		msg := fmt.Sprintf("匹配到 %d 个会话（歧义）", len(cands))
		if len(cands) > 0 {
			msg += "；候选：" + strings.Join(cands, "、")
		}
		return "", msg
	}
	block, err := renderSessionSnapshot(path, title)
	if err != nil {
		return "", err.Error()
	}
	return block, ""
}
