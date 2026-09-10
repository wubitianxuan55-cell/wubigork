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
