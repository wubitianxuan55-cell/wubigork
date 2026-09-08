# 参考：AI 改计划的「预览-确认-回滚」设计模式原始笔记（非竞品稿）

> 调研日期：2026-09-06。性质：专题检索原始记录（任务问题 3：对话式调整的安全口径）。检索结论：该设计模式的成熟公开范式集中在**编码 copilot**领域（diff review），PM 工具普遍缺席（各家竞品稿已标注）。本文件只记录来源与原文要点，不做跨竞品综合。

## 来源与要点

1. **VS Code「Review and revert agent changes」**
   URL: https://code.visualstudio.com/docs/agents/run/review-code-edits
   要点：apply/merge 前逐文件 review 全部变更；可 revert agent 编辑；核对基线/目标分支。

2. **GitHub Copilot Edits（VS）**
   URL: https://learn.microsoft.com/en-us/visualstudio/ide/copilot-edits?view=visualstudio
   要点：编辑完成后给「所有变更的汇总视图」（batched diff summary）——先汇总后细看。

3. **VS Copilot Agent Mode**
   URL: https://learn.microsoft.com/en-us/visualstudio/ide/copilot-agent-mode?view=visualstudio
   要点：执行动作前请求确认（confirmation gate）。

4. **Xebia：Copilot Edits 重构**
   URL: https://xebia.com/blog/smarter-refactoring-starts-with-github-copilot-edits/
   要点：「You review the diff before accepting the changes, similar to reviewing a pull request」——PR review 隐喻作为信任机制。

5. **GitHub Blog：ask/edit/agent 三模式**
   URL: https://github.blog/ai-and-ml/github-copilot/copilot-ask-edit-and-agent-modes-what-they-do-and-when-to-use-them/
   要点：自主性分级（ask→edit→agent），按风险决定是否加人工确认门。

## 归纳出的四步安全口径（供刀5 diff 确认流对照，非新结论）
1. Preview the diff（先看将改什么）
2. Explicit confirmation gate（副作用动作显式确认）
3. Easy revert（一键回滚）
4. Familiar metaphors（用 PR/修订对比这类用户已熟的心智）

## 对「不可行时怎么表现」的直接证据
- 未检索到任何 PM/排程工具公开「资源冲突/日历冲突/环检测时 AI 如何回应」的产品口径（本次检索范围内）。
- 最接近的替代证据：斑马进度以「检查逻辑关系错漏/工期冲突」为非 AI 卖点（见斑马稿）；孔明以「工期冲突+逻辑错误检查+合规检查」为卖点（见孔明稿）；Onplana 宣称 NL 解析+关键路径+风险检测（搜索摘要，未深挖：https://ingantt.com 对比栏目提及）。
- Reddit 信号（r/projectmanagers，经 Ingantt 检索转引）：用户普遍偏好手动控制、对 AI 生成计划信任有限——支持「默认 diff 确认+可整体拒绝」。

## PM 域缺席的含义
对话式调整的 diff/快照/回滚在 PM SaaS 中无成熟公开范式可抄；gaea 的 Plan→Apply+快照+Journal 纪律若配「甘特图上的 diff 高亮 + 情景对比」有机会成为该空白的定义者。
