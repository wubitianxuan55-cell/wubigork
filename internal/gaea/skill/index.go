package skill

import (
	"fmt"
	"strings"
)

// IndexMaxChars caps the pinned skills-index block so it can't bloat the
// cache-stable system-prompt prefix; bodies never enter the prefix.
const IndexMaxChars = 4000

const missingDescPlaceholder = `(no description — frontmatter is missing a "description:" line; tell the user to add one)`

// indexHeader introduces the skills block in the system prompt: how to invoke a
// skill, and the inline-vs-subagent distinction. V8.22: changed from permissive
// "you can invoke" to prescriptive "you MUST consult before acting" to push the
// model toward skill usage (Reasonix pattern).
const indexHeader = "# Skills — playbooks you MUST consult before acting\n\n" +
	"技能是领域知识或工作流编排——使用技能比从零开始更高效、更正确。" +
	"在开始任何工作之前，先检查是否有匹配的技能。" +
	"各技能的 description 说明了何时必须使用它" +
	"（如 \"在任何创造性工作之前必须使用此技能\"）。\n\n" +
	"每个条目是内置或用户编写的 playbook。" +
	"调用方式: `run_skill({ name: \"<skill-name>\", arguments: \"<task>\" })` " +
	"— `name` 只需标识符（如 `\"explore\"`），不要含 `[🧬 subagent]` 标签。" +
	"标记 `[🧬 subagent]` 的技能启动隔离子代理——其工具调用和推理不会进入你的上下文，" +
	"仅返回最终结论；用于上下文繁重的工作（深度探索、多步研究），你只需结论。" +
	"未标记的技能为 inline：正文成为工具结果供你直接阅读执行。" +
	"用户也可通过 `/<name>` 调用技能。"

// ApplyIndex appends the skills index to basePrompt, or returns it unchanged
// when there are no skills. Only names + descriptions (+ a subagent tag) are
// listed; bodies load on demand via run_skill.
//
// 刀3 预算化渲染（Distilled from Reasonix skill catalog 二分压缩描述行）：
// 目录超预算时先二分收缩描述宽度让全部条目挤进预算——尾部技能被硬截断
// 意味着模型永远看不见它们；只有纯名行仍超预算的极端才退回硬截断。
func ApplyIndex(basePrompt string, skills []Skill) string {
	if len(skills) == 0 {
		return basePrompt
	}
	join := func(descWidth int) string {
		lines := make([]string, 0, len(skills))
		for _, sk := range skills {
			lines = append(lines, indexLineAt(sk, descWidth))
		}
		return strings.Join(lines, "\n")
	}
	runes := func(s string) int { return len([]rune(s)) }
	joined := join(maxDescWidth)
	if runes(joined) <= IndexMaxChars {
		return basePrompt + "\n\n" + indexHeader + "\n\n```\n" + joined + "\n```"
	}
	// 二分最大可容纳的描述宽度；width 0 = 纯名行。压缩态在清单尾附一行
	// 说明（一并计入预算），模型知道描述是缩写、细节走 run_skill。
	fits := func(width int) bool {
		s := join(width)
		if width < maxDescWidth {
			s += compressedIndexNote
		}
		return runes(s) <= IndexMaxChars
	}
	lo, hi, best := 0, maxDescWidth, -1
	for lo <= hi {
		mid := (lo + hi) / 2
		if fits(mid) {
			best = mid
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	if best >= 0 {
		joined = join(best) + compressedIndexNote
		return basePrompt + "\n\n" + indexHeader + "\n\n```\n" + joined + "\n```"
	}
	// 极端兜底：纯名行也放不下（超长技能名/技能数极多）——旧行为硬截断。
	if r := []rune(joined); len(r) > IndexMaxChars {
		joined = string(r[:IndexMaxChars]) + fmt.Sprintf("\n… (truncated %d chars)", len(r)-IndexMaxChars)
	}
	return basePrompt + "\n\n" + indexHeader + "\n\n```\n" + joined + "\n```"
}

// compressedIndexNote 是压缩态目录的尾注，让模型知道描述被缩写、可经
// run_skill 取全文。计入 IndexMaxChars 预算。
const compressedIndexNote = "\n(descriptions compressed to fit the index budget; run_skill loads full details)"

// maxDescWidth 是目录条目描述的单行宽度上限（rune）。
const maxDescWidth = 130

// indexLine renders one skill at full width (unsqueezed catalog).
func indexLine(sk Skill) string {
	return indexLineAt(sk, maxDescWidth)
}

// indexLineAt renders one skill as "- name [tag] — description" with the
// description clipped to width (and the per-line base cap) runes. The subagent
// tag goes after the name so a model copying the line into run_skill's `name`
// arg still yields a clean identifier. width 0 renders the bare name line.
func indexLineAt(sk Skill, width int) string {
	if width <= 0 {
		tag := ""
		if sk.RunAs == RunSubagent {
			tag = " [🧬 subagent]"
		} else if sk.RunAs == RunPipeline {
			tag = " [🔗 pipeline]"
		}
		return "- " + sk.Name + tag
	}
	desc := strings.TrimSpace(strings.ReplaceAll(sk.Description, "\n", " "))
	if desc == "" {
		desc = missingDescPlaceholder
	}
	tag := ""
	if sk.RunAs == RunSubagent {
		tag = " [🧬 subagent]"
	} else if sk.RunAs == RunPipeline {
		tag = " [🔗 pipeline]"
	}
	base := 130 - len([]rune(sk.Name)) - len([]rune(tag))
	if width < base {
		base = width
	}
	clipped := clipRunes(desc, base)
	if clipped == "" {
		return "- " + sk.Name + tag
	}
	return "- " + sk.Name + tag + " — " + clipped
}

// clipRunes truncates s to at most max runes (ellipsis included), never
// splitting a multi-byte rune.
func clipRunes(s string, max int) string {
	if max < 1 {
		max = 1
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max-1 < 1 {
		return string(r[:1])
	}
	return string(r[:max-1]) + "…"
}
