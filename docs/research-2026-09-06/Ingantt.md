# Ingantt 调研原始稿

> 调研日期：2026-09-06。定位：**MS Project 替代品 + AI**（跨平台：Windows/macOS/iOS/Android/Web/Google Workspace 插件），「Describe your project in plain words. AI generates a complete schedule with tasks, dependencies, durations, and a work breakdown structure.」——与 gaea（含 mspdi 互导）几乎是同生态位的 AI 化样本，刀5 的最直接对标。

## 来源 URL
- 官网：https://ingantt.com/ ；中文页：https://www.ingantt.com/zh
- Google Workspace Marketplace：https://workspace.google.com/marketplace/app/ai_project_planning_gantt_chart_ingantt/286119906331
- Microsoft Store：https://apps.microsoft.com/detail/9nhq26qv09f6
- Apple App Store：https://apps.apple.com/ua/app/gantt-chart-ai-planner-ingantt/id6450835363
- Google Play：https://play.google.com/store/apps/details?id=com.ingantt_development.ingantt
- 官方 AI demo 视频：https://www.youtube.com/watch?v=sOjhGy5BJso

## 能力清单（AI 能做什么）
- **自然语言→完整计划**：日常语言描述项目 → AI 生成含**任务、依赖关系、工期、WBS** 的完整日程（四要素齐备，非仅标题）。
- 「Use it as-is, tweak it, or start from scratch」：生成物可直接用/可继续改/可从零手编——三种路径并列。
- 内置行业模板（建筑、软件、市场营销等）。
- 非确定性引擎能力（甘特图、成本、进度跟踪、关键路径等）为常规功能，AI 之外独立存在。
- **未查到**：对话式二次调整（如「压 10 天」）、AI 分析报告、生成前 diff 预览的细节口径；微软商店称 AI 助于「避免漏项、时间估算更准」属营销口径。

## 交互形态
- 单次生成式：描述→生成→进入常规甘特编辑器；app 内嵌 AI（非聊天面板长期驻留的口径）。
- 多端+插件分发（Google Workspace 内直接用）。

## 数据流（读什么/写什么）
- 读：用户的项目文字描述（+所选模板）。
- 写：生成计划落入本地 app 数据（甘特/WBS/成本）。
- MS Project 互导（mspdi）为其常规能力口径（作为 MS Project alternative），AI 生成与 mspdi 导出可组合。

## 可蒸馏点（对刀4/刀5）
1. **生成四要素口径（tasks+dependencies+durations+WBS）是市场已验证的最低完整集**：刀5 的 NL 生成至少要含这四项才有「完整计划」的心智；资源/日历可作为 v2 增量（Ingantt 也未把资源放进 AI 口径）。
2. **「as-is / tweak / from scratch」三态文案**直接可用作 gaea 生成确认页的三个按钮：整体接受 / 微调 / 全部手改——把「人工确认」产品化成一句话。
3. **AI 生成物落进确定性编辑器（甘特+CPM）而非聊天流**：证明「AI 生成→引擎接管」的产品形态在 MS Project 生态被接受，正对应 gaea 的 AI→schedule_apply→引擎重算。
4. 生成时机营销点「避免漏项、估算更准」：AI 的卖点不是「替你思考」而是「补全+查漏」——gaea 生成功能的文案口径可对齐（面向资料员/施工员的补全工具）。

## 不可取点
1. 独立评测稀缺（reddit 用户普遍表述偏好手动控制 AI 生成计划），信任建立主要靠应用商店自家文案——说明「生成即全盘接受」的用户阻力真实存在，diff 确认不可省。
2. 对话式调整与 AI 分析**未查到**：一次性生成范式，生成后 AI 退场；gaea 有 agent 常驻+workspace 纪律，可超越该形态。
3. 无建筑垂直口径（无工作日历/定额/季节施工的公开说明），模板只是「建筑模板」名称级。
