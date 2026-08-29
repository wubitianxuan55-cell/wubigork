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

// ResolveCitations 解析最终回复中的引用键：只保留真实存在的记忆（未知键静默
// 丢弃——模型可能幻觉出不存在的键，不应报错也不应触达），并对每条命中的记忆
// Touch（更新 last_used_at）。返回命中的记忆名（按出现顺序）。
//
// S1.2 B 读端隔离器：space 非空时解析限定本空间——Get/Touch 走 GetInSpace/
// TouchInSpace，跨空间键等同未知键静默不命中不 Touch（工位回合不触达乐园
// 记忆，反之亦然；验收红线）；space 为空 = 旧行为（全空间，既有调用语义）。
func (s *Set) ResolveCitations(text, space string) []string {
	if s == nil || text == "" {
		return nil
	}
	names := ExtractCitationNames(text)
	if len(names) == 0 {
		return nil
	}
	resolved := make([]string, 0, len(names))
	for _, name := range names {
		if _, ok := s.Store.GetInSpace(name, space); !ok {
			continue
		}
		if err := s.Store.TouchInSpace(name, space); err == nil {
			resolved = append(resolved, name)
		}
	}
	if len(resolved) == 0 {
		return nil
	}
	return resolved
}
