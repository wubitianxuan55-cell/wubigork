package app

// 阶段七 7.2-1 演示式录制（docs/gaea-stage7-plan-2026-09.md §2）：把当前
// 会话的多轮回放（含工具事件）经 LLM 蒸馏成结构化技能草稿，用户审阅修订后
// 沉淀入库。与既有两条沉淀通道的关系——GaeaCaptureSkill=单轮 task/solution
// 手动沉淀；gaea_memory_suggestions=procedural 记忆聚类候选（7.2-2 的前身）；
// 本通道=会话级演示录制（对齐 Anthropic Record a Skill / OpenAI Record & Replay
// 的已验证形态），差异在「多轮过程 → 结构化草稿」而非「单轮结果直存」。
// LLM 只产草稿，落盘必经用户审阅确认（GaeaSkillDraftSave），作者是上帝。

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/gaea/gaea/internal/ai"
	"github.com/gaea/gaea/internal/gaea/skill"
	"github.com/gaea/gaea/internal/util"
)

// SkillDraft 是一个可编辑的技能草稿（LLM 蒸馏产出，用户可改后保存）。
type SkillDraft struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Scenario    string   `json:"scenario"`
	Steps       []string `json:"steps"`
	Cautions    []string `json:"cautions"`
}

// SkillRecordResult 是会话录制草稿的完整返回（草稿+回放摘要+SKILL.md 预览）。
type SkillRecordResult struct {
	Draft   SkillDraft `json:"draft"`
	Replay  string     `json:"replay"`
	Preview string     `json:"preview"`
}

const (
	skillReplayMaxTurns  = 60  // 参与回放的最大消息条数（超出取末段——最近的任务过程优先）
	skillReplayTurnMax   = 600 // 单条消息 rune 截断
	skillReplayToolMax   = 200 // 工具条目（名+参数或结果）rune 截断
	skillDraftDescMax    = 120
	skillDraftScenMax    = 400
	skillDraftStepsMax   = 15
	skillDraftStepMax    = 400
	skillDraftCautionMax = 8
)

// buildSkillReplay 把会话历史压成回放文本：用户/助手动静按条截断，工具调用
// 与结果合并为一行摘要，只保留末段 skillReplayMaxTurns 条。纯函数、确定性。
func buildSkillReplay(msgs []HistoryMessage) string {
	if len(msgs) > skillReplayMaxTurns {
		msgs = msgs[len(msgs)-skillReplayMaxTurns:]
	}
	var b strings.Builder
	for _, m := range msgs {
		switch m.Role {
		case "user":
			b.WriteString("【用户】" + truncRunes(strings.TrimSpace(m.Content), skillReplayTurnMax) + "\n")
		case "assistant":
			if s := strings.TrimSpace(m.Content); s != "" {
				b.WriteString("【助手】" + truncRunes(s, skillReplayTurnMax) + "\n")
			}
		case "tool":
			line := strings.TrimSpace(m.ToolName + " " + m.ToolArgs)
			if line != "" {
				b.WriteString("【工具】" + truncRunes(line, skillReplayToolMax) + "\n")
			}
		case "tool_result":
			if s := strings.TrimSpace(m.ToolOutput); s != "" {
				b.WriteString("【工具结果】" + truncRunes(s, skillReplayToolMax) + "\n")
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func truncRunes(s string, max int) string {
	if r := []rune(s); len(r) > max {
		return string(r[:max]) + "…"
	}
	return s
}

// skillDraftNameSanitizer 把 LLM 给出的名字压成合法技能标识（小写、非法字符
// 转 -、去连续 -）；压缩后仍不合法则报错（重试/用户修改兜底）。
var skillDraftNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeSkillDraftName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = skillDraftNameSanitizer.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-._")
	if r := []rune(name); len(r) > 64 {
		name = string(r[:64])
	}
	return name
}

// parseSkillDraft 校验并归一化草稿：名字清洗、条目裁剪上限、空步骤断然拒绝。
func parseSkillDraft(d *SkillDraft) error {
	d.Name = sanitizeSkillDraftName(d.Name)
	if !skill.IsValidName(d.Name) {
		return fmt.Errorf("技能名不合法：%q（需字母开头，仅字母/数字/_/-/.，1-64 字符）", d.Name)
	}
	d.Description = truncRunes(strings.TrimSpace(d.Description), skillDraftDescMax)
	d.Scenario = truncRunes(strings.TrimSpace(d.Scenario), skillDraftScenMax)
	d.Steps = clampNonEmpty(d.Steps, skillDraftStepsMax, skillDraftStepMax)
	d.Cautions = clampNonEmpty(d.Cautions, skillDraftCautionMax, skillDraftStepMax)
	if len(d.Steps) == 0 {
		return fmt.Errorf("技能草稿缺少操作步骤")
	}
	return nil
}

func clampNonEmpty(in []string, countMax, lenMax int) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, truncRunes(s, lenMax))
		if len(out) >= countMax {
			break
		}
	}
	return out
}

// renderSkillDraftBody 渲染草稿为技能正文（与 GaeaCaptureSkill 模板同构：
// 适用场景/操作步骤/调用方式，另加注意事项段）。
func renderSkillDraftBody(d SkillDraft) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s 技能\n\n## 适用场景\n%s\n\n## 操作步骤\n", d.Name, d.Scenario)
	for i, s := range d.Steps {
		fmt.Fprintf(&b, "%d. %s\n", i+1, s)
	}
	if len(d.Cautions) > 0 {
		b.WriteString("\n## 注意事项\n")
		for _, c := range d.Cautions {
			b.WriteString("- " + c + "\n")
		}
	}
	fmt.Fprintf(&b, "\n## 调用方式\n- 对话中直接描述任务即可命中本技能，也可用 /%s 显式调用\n- 按操作步骤执行，完成后按场景要求验证产出\n", d.Name)
	return b.String()
}

// GaeaSkillDraftFromSession 把当前会话回放蒸馏为技能草稿（LLM 结构化 +
// RetryJSON 格式重试；办公功能级路由，本地优先）。草稿只回不落盘——
// 落盘必经 GaeaSkillDraftSave（用户审阅后）。
func (a *App) GaeaSkillDraftFromSession() (SkillRecordResult, error) {
	if gaeaCtrl() == nil {
		return SkillRecordResult{}, fmt.Errorf("办公引擎未初始化")
	}
	msgs := a.GaeaHistory()
	replay := buildSkillReplay(msgs)
	if replay == "" {
		return SkillRecordResult{}, fmt.Errorf("当前会话为空——先完成一次任务再录制技能")
	}
	if a.eng == nil {
		return SkillRecordResult{}, fmt.Errorf("提示词引擎未初始化")
	}
	tmpl := a.eng.Get("skill-from-session")
	if tmpl == nil {
		return SkillRecordResult{}, fmt.Errorf("缺少 skill-from-session 模板文件")
	}
	if a.client == nil {
		return SkillRecordResult{}, fmt.Errorf("ai client unavailable")
	}
	engID, model, _ := a.routeOfficeLocal("office")
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	caller := func(ctx context.Context, sys, usr string) (string, error) {
		return a.client.ChatSimpleStreamWithOptions(ctx, model, sys, usr,
			ai.ChatSimpleOptions{EngineID: engID, Temperature: 0.2, MaxTokens: 1500, TimeoutMinutes: 3})
	}
	system := tmpl.BuildSystemPrompt("")
	user := tmpl.BuildUserPrompt(map[string]string{"replay": replay})
	jsonStr, err := util.RetryJSON(ctx, caller, system, user, 2)
	if err != nil {
		return SkillRecordResult{}, fmt.Errorf("蒸馏技能草稿失败: %w", err)
	}
	var draft SkillDraft
	if err := json.Unmarshal([]byte(jsonStr), &draft); err != nil {
		return SkillRecordResult{}, fmt.Errorf("解析技能草稿失败: %w", err)
	}
	if err := parseSkillDraft(&draft); err != nil {
		return SkillRecordResult{}, err
	}
	return SkillRecordResult{
		Draft:   draft,
		Replay:  replay,
		Preview: skill.RenderSkillFile(draft.Name, draft.Description, renderSkillDraftBody(draft)),
	}, nil
}

// GaeaSkillDraftSave 保存（通常已审阅修订的）技能草稿：渲染→落盘→镜像→
// 热加载，与 GaeaCaptureSkill 同一通道（saveSkillFile）。
func (a *App) GaeaSkillDraftSave(d SkillDraft) (SkillCaptureResult, error) {
	if err := parseSkillDraft(&d); err != nil {
		return SkillCaptureResult{}, err
	}
	if strings.TrimSpace(d.Scenario) == "" {
		return SkillCaptureResult{}, fmt.Errorf("缺少适用场景")
	}
	return a.saveSkillFile(d.Name, d.Description, renderSkillDraftBody(d))
}
