# Devin（Cognition）调研原始稿

> 调研日期：2026-09-06。定位：自主编码 agent（「AI software engineer」），2025-2026 的标志性进展是进入企业人力序列（Goldman Sachs「AI 员工 #1」）。对 gaea 的价值不在编码而在其「团队内 agent 成员」的组织化口径与集成方式。

## 来源 URL
- 官网：https://devin.ai/
- Cognition 官网：https://cognition.com/
- Goldman Sachs 案例（IBM Think 报道）：https://www.ibm.com/think/news/goldman-sachs-first-ai-employee-devin
- 生产环境实战分析：https://www.sitepoint.com/devin-ai-engineers-production-realities/
- 技术综述：https://www.zenml.io/llmops-database/autonomous-software-development-agent-for-production-code-generation
- 商业评估：https://www.baytechconsulting.com/blog/devin-ai-unveiled-should-your-business-hire-the-worlds-first-ai-software-engineer

## 能力清单（AI 能做什么）
- 接自然语言任务→规划→写代码→自测→交付生产代码，「plans, writes, tests, and ships production code inside your existing workflows」。
- 并行多 agent（parallel cloud agents）同时开工。
- 集成面向协作工具（Slack/Linear/GitHub 类），任务从协作工具进入，agent 在隔离环境执行，产出 PR/报告回流。
- **未查到**：PM/排程专项功能（Devin 不是 PM 工具）。

## 交互形态
- 对话派活（Slack/Linear issue/网页 session），异步执行，人类按 PR review 节奏验收——**「派活-执行-验收」三段式**而非实时陪伴。
- 组织化叙事：agent 是「员工/团队成员」（enterprise "hired" Devin），有账号、有任务队列、有产出物。

## 数据流（读什么/写什么）
- 读：任务描述、代码库、协作工具上下文。
- 写：代码/PR/评论；通过集成在协作工具里回写状态。
- 验收门槛：人类 code review（PR diff）是事实上的确认关口——与编码工具通用模式一致。

## 可蒸馏点（对刀4/刀5）
1. **「派活-执行-验收（diff review）」的组织化分工**：gaea 刀5 的对话式调整可类比为「人类 PM 派任务给 AI 计划工程师」，AI 在计划文件副本上执行，人类验收 diff 后 apply——与 Plan→Apply+快照纪律同构。
2. **并行 agent（多 session 隔离环境）**：多个调整情景（如「压 10 天」vs「加资源」）可并行试算再比较——前提是计划模型有快照分支能力（刀4 的快照设计正好支撑）。
3. 「AI 员工」叙事（有身份/有队列/有交付物）对建筑企业的采购语言友好，gaea 面向企业市场可借该话术。
4. 集成以协作工具为入口（Slack/Linear）而非内嵌 panel：提示 gaea 的 agent 入口可以同时是聊天面板与外部 IM。

## 不可取点
1. 生产现实报道（sitepoint）显示自主 agent 需大量监督，完全自治不成熟——映射到进度计划：AI 全自动改计划不可信，human-in-the-loop 必须默认开启。
2. Devin 验收靠 code review，要求用户有专业判断力；进度计划域用户（施工员/资料员）判断力参差，gaea 需要把「验收」降维成可读的风险/差异说明而非裸 diff。
3. 与 PM 场景无直接功能交叉，本文件仅提供组织化/交互口径参考。
