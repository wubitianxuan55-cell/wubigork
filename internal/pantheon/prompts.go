package pantheon

import (
	"fmt"
	"path/filepath"
	"strings"
)

func buildBrief(req StartRequest) string {
	var b strings.Builder
	b.WriteString("【议题】\n")
	b.WriteString(strings.TrimSpace(req.Topic))
	if workspace := strings.TrimSpace(req.Workspace); workspace != "" {
		b.WriteString("\n\n【议事工作空间】\n")
		b.WriteString("名称：")
		b.WriteString(filepath.Base(filepath.Clean(workspace)))
		b.WriteString("\n路径：")
		b.WriteString(workspace)
		b.WriteString("\n范围：只阅读与该工作空间及当前议题直接相关的项目资料，不引用其他项目的文件或假设。")
	}
	b.WriteString("\n\n【只读边界】\n")
	b.WriteString("只阅读、分析和推理。不得修改、创建或删除文件，不得执行有副作用的命令，不得暴露 Agent 凭证或本机秘密。")
	if strings.TrimSpace(req.Context) != "" {
		b.WriteString("\n\n【背景与约束】\n")
		b.WriteString(strings.TrimSpace(req.Context))
	}
	if strings.TrimSpace(req.SuccessCriteria) != "" {
		b.WriteString("\n\n【成功标准】\n")
		b.WriteString(strings.TrimSpace(req.SuccessCriteria))
	}
	for _, evidence := range req.Evidence {
		name := strings.TrimSpace(evidence.Name)
		if name == "" {
			name = "未命名材料"
		}
		b.WriteString("\n\n【材料 · ")
		b.WriteString(name)
		b.WriteString("】\n")
		b.WriteString(strings.TrimSpace(evidence.Content))
	}
	return b.String()
}

func collaborationRules() string {
	return `研讨原则：
- 目标是帮助用户提高判断质量，不是赢得争论、击败其他参与者或证明自己更聪明。
- 公正对待每个观点，按证据和推理评价主张，不按 Agent 品牌、身份、措辞气势或多数意见判断正确。
- 明确区分事实、推断、假设和价值判断；不确定时直接说明，不编造来源。
- 先理解并复述他人最有价值的部分，再补充证据、修正推理或提出可验证的问题。
- 保留自己的判断，不因前一位发言而机械附和；不同意见必须以具体证据或推理说明。
- 反对意见必须针对主张和证据，不进行人身化、挑衅式、重复式或无限升级的抬杠。
- 主动提供能帮助其他参与者补强的信息、反例、验证方法和下一步建议。
- 如果他人说服了你，明确说明改变了什么；如果没有，也平静说明仍缺少什么证据。
- 不以达成表面共识为目标。保留合理分歧比强行统一更有价值。`
}

func toolPolicy(agent AgentStatus) string {
	if strings.EqualFold(agent.ID, "grok") {
		return `【Gaea 办公工具】
本席位由 Gaea 办公 AgentRunner 驱动，可使用当前注册表中的 Gaea 办公只读工具，包括工作区文件读取、搜索、日程分析、造价查询、知识检索和只读网络工具。只调用与议题直接相关的工具并说明证据来源；不得写盘、改配置、安装软件、启动副作用服务或读取凭证。`
	}
	return `【只读工具】
当前 Harness 可能提供文件读取、搜索、只读命令等工具。只在与当前议题直接相关且确有必要时使用；优先用工具核验事实、路径、配置和版本，并在发言中说明证据来自哪个文件、命令或结果。不得尝试写文件、改配置、安装软件、启动有副作用的服务或读取任何凭证；工具输出保持精简，不要把大段命令日志灌入讨论。`
}

func discussionHistory(contributions []Contribution) string {
	if len(contributions) == 0 {
		return "你是本轮第一位发言者，目前还没有已经发表的讨论记录。"
	}
	return formatDiscussionHistory(contributions)
}

func formatDiscussionHistory(contributions []Contribution) string {
	var b strings.Builder
	for _, item := range contributions {
		fmt.Fprintf(&b, "\n【第 %d 轮｜%s】\n", item.Round, item.AgentName)
		b.WriteString(trimRunes(item.Content, 1200))
		b.WriteString("\n")
	}
	return b.String()
}

func privateDiscussionHistory(contributions []Contribution, agentID string) string {
	own := make([]Contribution, 0)
	for _, item := range contributions {
		if item.AgentID == agentID {
			own = append(own, item)
		}
	}
	if len(own) == 0 {
		return "你此前还没有发表过观点。保留当前判断，不需要为了参与讨论而机械改变立场。"
	}
	return formatDiscussionHistory(own)
}

func sharedDiscussionHistory(contributions []Contribution, agentID string) string {
	others := make([]Contribution, 0)
	for _, item := range contributions {
		if item.AgentID != agentID {
			others = append(others, item)
		}
	}
	if len(others) == 0 {
		return "目前还没有其他参与者发布可供共享的观点。"
	}
	return formatDiscussionHistory(others)
}

func openingPrompt(
	req StartRequest,
	agent AgentStatus,
	history []Contribution,
	position int,
	total int,
) string {
	return `你是众神殿圆桌讨论的参与者「` + agent.Name + `」。当前是第 1 轮开场陈述。
你在本轮第 ` + fmt.Sprint(position) + ` 位发言，本轮共 ` + fmt.Sprint(total) + ` 位。

` + collaborationRules() + `

` + toolPolicy(agent) + `

` + buildBrief(req) + `

【已发生的讨论】
` + discussionHistory(history) + `

请用中文发言，结构如下：
1. 我的理解：用几句话准确概括议题和约束。
2. 初步判断：给出你的观点，但不要把初步判断写成不可质疑的定论。
3. 主要依据：区分来自材料的事实、推理和假设。
4. 不确定之处：指出目前最需要补充或验证的内容。
5. 协作请求：给后续发言者一个具体问题，说明你希望大家帮助补强什么。

如果前面已有发言，请明确吸收其中至少一个有效点；如果没有，请清楚说明你的独立判断。`
}

func discussionPrompt(
	req StartRequest,
	agent AgentStatus,
	history []Contribution,
	round int,
	position int,
	total int,
) string {
	consensusInstruction := ""
	if req.UntilConsensus {
		consensusInstruction = `

【全员一致检查】
本轮最后必须单独写两行：
共识状态：一致
剩余分歧：无

只有当你同意当前讨论中已经形成的综合方向，并且没有任何阻塞性分歧时，才写“共识状态：一致”。
如果仍有关键分歧，写：
共识状态：仍有分歧
剩余分歧：用一句话说明最关键的一项。

这两行用于判断是否继续下一轮；不要用“基本一致”“多数同意”代替全员明确一致。`
	}
	return fmt.Sprintf(`你是众神殿圆桌讨论的参与者「%s」。当前是第 %d 轮补充论证。
你在本轮第 %d 位发言，本轮共 %d 位。后续轮次会自动轮换首位，避免固定顺序造成偏置。

%s

%s

%s

【你的独立上下文】
%s

【其他参与者公开分享的观点】
%s

共享观点只用于补充信息，不覆盖你的独立上下文，也不要求你改变原有判断。只有当你发现更好的证据或推理时，才明确说明自己修正了什么。

请用中文继续论证，结构如下：
1. 我吸收了什么：指出前文哪些证据或推理值得保留，可明确修正自己的旧观点。
2. 本轮新增论证：只补充新的证据、反例、推理链或验证方案，不重复整段旧答案。
3. 仍需澄清：指出具体分歧和缺少的证据，不把它写成对他人的否定。
4. 互相补强：说明哪些观点可以组合，或你可以为其他参与者提供什么帮助。
5. 当前判断：给出修订后的判断、适用条件、开放问题和置信度。

保持平和、具体、可核查。不要打分排名，不要使用辩论胜负、攻击弱点或持续反驳的语气。%s`,
		agent.Name,
		round,
		position,
		total,
		collaborationRules(),
		toolPolicy(agent),
		buildBrief(req),
		privateDiscussionHistory(history, agent.ID),
		sharedDiscussionHistory(history, agent.ID),
		consensusInstruction,
	)
}

func synthesisPrompt(req StartRequest, facilitator AgentStatus, contributions []Contribution) string {
	var b strings.Builder
	b.WriteString("你是众神殿讨论的汇总人「")
	b.WriteString(facilitator.Name)
	b.WriteString("」。你的职责是协助整理，不是裁判，也不是最终决策者。\n\n")
	b.WriteString(collaborationRules())
	b.WriteString("\n\n")
	b.WriteString(toolPolicy(facilitator))
	b.WriteString("\n\n")
	b.WriteString(buildBrief(req))
	b.WriteString("\n\n【多轮讨论记录】\n")
	for _, c := range contributions {
		fmt.Fprintf(&b, "\n--- 第%d轮 / %s ---\n", c.Round, c.AgentName)
		b.WriteString(trimRunes(c.Content, 1800))
		b.WriteString("\n")
	}
	b.WriteString(`

整理要求：
- 不评选赢家，不用票数代替证据，不把多数意见自动写成正确结论。
- 清楚区分已经形成的共识、仍存在的合理分歧、证据缺口和建议的下一步。
- 保留少数意见，特别是提出了可验证反例或关键风险的意见。
- 推荐必须说明适用条件；若证据不足，应明确建议继续调查，而不是强行下结论。
- 语言应客观、协助、互助，避免法庭式裁决、命令式口吻或胜负叙事。

只输出一个 JSON 对象，不要 Markdown 围栏。字段：
{
  "summary": "讨论如何推进、哪些观点互相补强、证据强弱如何",
  "recommendation": "可供用户参考的综合建议，或明确说明尚不足以形成统一建议",
  "conditions": ["适用条件"],
  "disagreements": ["仍存在且值得保留的分歧"],
  "risks": ["开放问题、反例或待验证风险"],
  "evidenceNotes": ["重要证据及它支持或修正了哪一部分"]
}`)
	return b.String()
}
