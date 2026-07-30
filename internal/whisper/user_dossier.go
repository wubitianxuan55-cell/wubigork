// Package whisper — user_dossier.go
// 100% 对齐 ackem memory/userDossier.ts
// 用户档案汇总：LLM 生成人类可读的用户档案

package whisper

import (
	"fmt"
	"sort"
	"strings"
)

// ─── 档案域映射 ──────────────────────────────────────────────

var dossierDomains = map[string][]string{
	"IDENTITY":    {"BASIC_PROFILE", "LIFE_STORY", "VALUES_BELIEFS", "SELF_PERCEPTION"},
	"SOCIAL":      {"FAMILY", "FRIENDS", "PARTNER", "OUR_BOND"},
	"DAILY_LIFE":  {"ROUTINES", "HEALTH", "LIVING_SPACE", "LIFESTYLE"},
	"PURSUITS":    {"CAREER", "LEARNING", "GOALS", "PROJECTS", "PROCEDURES"},
	"INNER_WORLD": {"TASTES", "VULNERABILITIES", "INSIDE_JOKES"},
	"TEMPORAL":    {"COMMITMENTS", "PLANS"},
}

var dynamicSubs = map[string]bool{"NOW": true, "MOOD": true, "PROJECTS": true, "HEALTH": true}

const dossierSystemStable = `你是轻语，用户的 AI gaea。你正在私下整理关于用户的笔记——就像一个人在心里默默记住另一个人的信息一样。

根据以下所有关于 ta 的核心事实，重新梳理一份新的笔记。

── 规则 ──
· 用自然的口语写，像自己私下的笔记。不要像档案报告。
· 按"基本信息→性格→喜好→我们的关系"的顺序自然过渡。
· 只写你从事实中确定知道的，不要编造。
· 先写稳定信息，再写近期状态。近期状态用"—— 近期状态（仅供参考） ——"分隔。
· 保持 500-1000 字。
· 人称：用户以"ta"称呼。

── 禁止 ──
× 不要写"根据事实""根据记录""我的数据显示"等元表述
× 不要使用表格、列表、标题格式
× 不要把近期状态写成确定事实
× 不要把成人内容细节写进档案
× 不要记录任何高度私密的短期状态`

// ─── GenerateUserDossier ─────────────────────────────────────

// GenerateUserDossier LLM 生成用户档案
func GenerateUserDossier(llm LlmClient, fs *FactStore) (string, error) {
	if llm == nil || fs == nil {
		return "", nil
	}

	facts := getDossierFacts(fs, false)
	if len(facts) < 5 {
		return "", nil
	}

	userPrompt := buildDossierUserMsg(facts)
	content, err := llm.Chat(dossierSystemStable, userPrompt)
	if err != nil || len(content) < 50 {
		return "", err
	}
	return content, nil
}

// LoadUserDossierHint 生成注入 system prompt 的用户档案提示
func LoadUserDossierHint(dossier string) string {
	if dossier == "" {
		return ""
	}
	// 截断到 1000 字
	runes := []rune(dossier)
	if len(runes) > 1000 {
		dossier = string(runes[:1000])
	}
	return "\n\n【关于 ta 的笔记 · 仅供你内心参考 · 禁止在回复中对用户说\"ta\"】\n" +
		dossier +
		"\n\n⚠️【护栏】：你在和用户面对面直接对话。使用这些笔记时，必须将\"ta\"转化为第二人称\"你\"。" +
		"绝对不要说\"根据我的笔记\"\"根据我的记录\"等元表述。"
}

// ─── 辅助 ────────────────────────────────────────────────────

func getDossierFacts(fs *FactStore, dynamicOnly bool) []string {
	all := fs.ListActive()

	// 过滤：只保留档案相关域
	var filtered []*Fact
	for _, f := range all {
		subs, ok := dossierDomains[f.Domain]
		if !ok {
			continue
		}
		found := false
		for _, s := range subs {
			if f.Subcategory == s {
				found = true
				break
			}
		}
		if !found {
			continue
		}
		if f.Weight < 1 || f.Confidence < 0.6 {
			continue
		}
		filtered = append(filtered, f)
	}

	if dynamicOnly {
		var dynamic []*Fact
		for _, f := range filtered {
			if dynamicSubs[f.Subcategory] {
				dynamic = append(dynamic, f)
			}
		}
		sort.Slice(dynamic, func(i, j int) bool {
			return dynamic[i].UpdatedAt.After(dynamic[j].UpdatedAt)
		})
		if len(dynamic) > 20 {
			dynamic = dynamic[:20]
		}
		return factsToSummaries(dynamic)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Weight > filtered[j].Weight
	})
	if len(filtered) > 50 {
		filtered = filtered[:50]
	}
	return factsToSummaries(filtered)
}

func factsToSummaries(facts []*Fact) []string {
	var s []string
	for _, f := range facts {
		s = append(s, f.Summary)
	}
	return s
}

func buildDossierUserMsg(facts []string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("以下是关于 ta 的所有核心事实（共 %d 条）：\n", len(facts)))
	for _, f := range facts {
		sb.WriteString("· ")
		sb.WriteString(f)
		sb.WriteString("\n")
	}
	return sb.String()
}
