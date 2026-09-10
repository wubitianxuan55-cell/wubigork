package memory

import (
	"regexp"
	"strings"
)

// 记忆引用可追溯（蒸馏 codex memories/read 的 citation 闭环）：注入块每行带
// 稳定引用键 [MEM:<name>]，模型采纳某条记忆时在句末标注同名引用；回合结束后
// 程序化解析回传文本，被引用的记忆 Touch（更新 last_used_at，高频排序同源），
// 前端把引用键渲染成可点击徽标弹层展示记忆详情与沉淀来源——办公用户可验证
// 「你引用的资料是不是真的」。

// citationRe 匹配模型回传文本中的引用键：[MEM:cost-rule] / [mem:cost-rule]
// （键大小写不敏感，代码侧归一为小写）。name 与记忆 Name 同构（kebab-case
// slug），避免误匹配普通 Markdown 方括号文本。
var citationRe = regexp.MustCompile(`(?i)\[MEM:([a-z0-9][a-z0-9-]*)\]`)

// ExtractCitationNames 按出现顺序提取文本中的引用键并去重。
func ExtractCitationNames(text string) []string {
	if text == "" {
		return nil
	}
	matches := citationRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		name := strings.ToLower(m[1])
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

// CitationResult 是一次引用解析的两侧结果：命中的键 Touch 触达；悬空键
// （幻觉/已删）不触达、不进图，但留痕可查——「悬空拒写」的可观测侧。
type CitationResult struct {
	Resolved []string
	Dangling []string
}

// StripDanglingCitations 是回复定稿闸（5.1 出口判据「[MEM:] 引用在回复发出前
// 全部解析到节点」的执行侧）：文本里解析不到的 [MEM:] 键整处剥离（含紧邻的
// 一个前导空白），命中的键原样保留。**不 Touch**——触达仍是回合收尾
// ResolveCitations 的职责（两条路径分离：定稿闸只校验改写，触达带留痕）；
// 被剥离键由调用方落 dangling cite 事件（AppendCiteEvents），可观测性不丢。
// space 语义同 GetInSpace：非空限定本空间，跨空间键视为悬空。纯函数：不改
// 存储状态，只读存在性。
func (s Store) StripDanglingCitations(text, space string) (string, []string) {
	names := ExtractCitationNames(text)
	if len(names) == 0 {
		return text, nil
	}
	if s.backend == nil && s.Dir == "" && s.GlobalDir == "" {
		// 零值/禁用 Store：存在性判定恒失败，会误剥正常引用键——整体跳过
		//（记忆关闭/未装配场景，boot 闭包侧也不注入）。
		return text, nil
	}
	var dangling []string
	for _, name := range names {
		if _, ok := s.GetInSpace(name, space); !ok {
			dangling = append(dangling, name)
		}
	}
	if len(dangling) == 0 {
		return text, nil
	}
	cleaned := text
	for _, name := range dangling {
		re := regexp.MustCompile(`(?i)\s?\[MEM:` + name + `\]`)
		cleaned = re.ReplaceAllString(cleaned, "")
	}
	return cleaned, dangling
}

// ResolveCitationsDetailed 是 ResolveCitations 的全量版：命中照旧 Touch（更新
// last_used_at），悬空键返回而非静默丢——调用方（controller 回合收尾）拿去落
// cite 事件。空间语义与 ResolveCitations 一致：space 非空时限定本空间，跨空间
// 键等同未知键。
func (s *Set) ResolveCitationsDetailed(text, space string) CitationResult {
	if s == nil || text == "" {
		return CitationResult{}
	}
	names := ExtractCitationNames(text)
	if len(names) == 0 {
		return CitationResult{}
	}
	var out CitationResult
	for _, name := range names {
		if _, ok := s.Store.GetInSpace(name, space); !ok {
			out.Dangling = append(out.Dangling, name)
			continue
		}
		if err := s.Store.TouchInSpace(name, space); err == nil {
			out.Resolved = append(out.Resolved, name)
		}
	}
	return out
}

// ResolveCitations 解析最终回复中的引用键：只保留真实存在的记忆（未知键静默
// 丢弃——模型可能幻觉出不存在的键，不应报错也不应触达），并对每条命中的记忆
// Touch（更新 last_used_at）。返回命中的记忆名（按出现顺序）。
//
// S1.2 B 读端隔离器：space 非空时解析限定本空间——Get/Touch 走 GetInSpace/
// TouchInSpace，跨空间键等同未知键静默不命中不 Touch（工位回合不触达乐园
// 记忆，反之亦然；验收红线）；space 为空 = 旧行为（全空间，既有调用语义）。
func (s *Set) ResolveCitations(text, space string) []string {
	r := s.ResolveCitationsDetailed(text, space)
	if len(r.Resolved) == 0 {
		return nil
	}
	return r.Resolved
}
