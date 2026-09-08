# Wrike AI（Work Intelligence + AI Agents）调研原始稿

> 调研日期：2026-09-06。定位：最早把「ML 风险预测」产品化的工作管理平台之一（2021 年即推出 AI Project Risk Prediction），2025 年叠加 LLM 层与 AI agents。与 gaea 最相关的是其「风险分级+基线计划+情景推演」叙事。

## 来源 URL
- 风险预测发布文：https://www.wrike.com/blog/wrike-work-intelligence-ai-project-risk-prediction/
- 官方帮助（AI Project Risk Prediction）：https://help.wrike.com/hc/en-us/articles/360055046934-AI-Project-Risk-Prediction
- Wrike AI 平台页：https://www.wrike.com/ai/
- 发布通稿（Business Wire）：https://www.businesswire.com/news/home/20211216005958/en/Wrike-Releases-New-Capabilities-To-Automatically-Predict-Project-Success-Makes-Recommendations-To-Save-Time-Optimize-Resources
- 企业 AI 用例（含基线计划生成/情景测试/风险打分）：https://www.wrike.com/project-management-guide/enterprise-ai-use-cases/ ；https://www.wrike.com/blog/ai-for-project-management/
- 2025-11 更新（内置+自定义 AI agents）：https://www.youtube.com/watch?v=mvOeyIY0-Dg

## 能力清单（AI 能做什么）
- **AI Project Risk Prediction**（ML）：分析项目数据，输出「低/中/高」延期风险评级，提前告警 PM。输出是分档评级而非数值预测。
- **Smart Selectors / Smart Search**（2021 同批）：自然语言筛选与搜索。
- **基线计划生成**：用 AI agents 生成 baseline project plan（用例指南口径）。
- **情景测试（scenario tests）**：对计划跑情景并给 schedule/scope 风险打分——与 gaea 的「压 10 天重算 diff」最接近的官方叙事。
- **自动化+AI agents**（2025-11 起）：内置 agents 与自定义 agents 做风险报告、流程编排。
- **未查到**：公开的 CPM 级计算细节（关键路径/总时差由引擎算还是 AI 算无口径）、对话式调整的确认流。

## 交互形态
- 嵌入式：项目详情页内风险徽章/评级。
- 自然语言：Smart Search 检索。
- Agent/自动化：规则+agents 后台跑。

## 数据流（读什么/写什么）
- 读：历史项目+实时项目数据（进度、负载）用于 ML 训练与预测。
- 写：风险评级展示、告警；agents 生成报告/计划草案。写操作确认流未查到。

## 可蒸馏点（对刀4/刀5）
1. **风险输出用「低/中/高」三档而非裸概率**：对非数据背景的建筑 PM 友好；schedule_analyze 的风险提示可按「延误风险低/中/高+依据（关键路径含 X、浮动仅 N 天）」输出。
2. **「基线计划+情景测试」是风险预测的前置**：gaea 已有快照/Journal 纪律，可把「AI 调整前的 baseline」与「调整后情景」对齐成 Wrike 式叙事——调整即跑一个情景。
3. ML 风险预测吃「历史项目+实时数据」：gaea 短期无跨项目数据，但可先用确定性信号（总时差消耗速率、关键路径变化次数）做规则版风险分级，口径与 Wrike 一致。
4. 2021→2025 的演进路径（先 ML 预测后 LLM agents）说明：**风险/分析类 AI 先落地、生成/调整类后落地**，与刀4(analyze 工具)→刀5(生成/调整)的顺序吻合。

## 不可取点
1. 风险评级黑箱：公开资料未解释评级依据，用户无法校准信任——gaea 的 AI 解读必须附 CPM 引擎的确定性依据（时差/路径），避免黑箱评级。
2. 「scenario tests」只见宣传口径，无产品文档细节，落地成色存疑。
3. Work Intelligence 是闭源专有层，能力与 Wrike 数据模型强耦合，无法借鉴实现只能借鉴口径。
