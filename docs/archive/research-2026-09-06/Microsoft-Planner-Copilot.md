# Microsoft Planner Copilot（Copilot in Planner, preview）调研原始稿

> 调研日期：2026-09-06。定位：微软任务级计划工具（原 Tasks by Planner/To Do 合并）的 Copilot 能力，与 Project 系（Project for the web / Plan 3/5）共用许可体系。未查到 Project 桌面版或 Project for the web 上有独立的「AI 排程/CPM 级」Copilot 功能（本次检索范围内）。

## 来源 URL
- 入门指南（官方）：https://support.microsoft.com/en-us/planner/get-started-with-copilot-in-planner-preview
- FAQ（官方，信息量最大）：https://support.microsoft.com/en-us/planner/copilot/frequently-asked-questions-about-copilot-in-planner-preview
- 产品页：https://www.microsoft.com/en-us/microsoft-365/planner/microsoft-planner
- 许可与发布报道：https://learn.microsoft.com/en-sg/answers/questions/5910427/help-with-planner-reports-and-dashboards-to-pull-f ；https://www.tenforums.com/windows-10-news/212928-copilot-planner-preview-begins-roll-out-new-microsoft-planner.html
- 第三方解读：https://pktech.net/blog/creating-automatic-task-boards-with-copilot-in-microsoft-planner

## 能力清单（AI 能做什么）
- 生成类：从自然语言 prompt「生成完整计划（full plan），含 tasks、sub-tasks、buckets、goals」；MVP 阶段生成的任务**只有标题**（tasks are titled only）；可向已有计划「添加任务/桶/目标」。
- 问答类：对既有计划做基本问答，如「本周到期任务有哪些」。官方把 prompt 分为 Create / Understand / Edit / Ask 四类。
- 辅助类：Plan my day、Get a status report（建议 prompt 形式提供）。
- **未查到**：自动排程/依赖关系/工期计算/关键路径类能力；FAQ 明确未提 Copilot 会设置 due date（只有对既有到期任务的问答）。这印证 Planner Copilot 目前是「清单生成器」而非「排程引擎」。

## 交互形态
- 聊天式右侧边栏：从计划顶部菜单打开 Copilot，「Copilot pane opens to the right of the plan」。
- 顶部提供建议 prompt 按钮，底部「View Prompts」可浏览 prompt 库，也可在输入框自由输入。
- 仅在 Teams 内 Planner 应用（以及新 Planner web，官方称 coming soon）可用。
- 依赖企业搜索（enterprise search）在每次请求时后台自动检索用户有权访问的相关文件；聊天框输入「/」可手动附加文件，「always used as context」。

## 数据流（读什么/写什么）
- 读：当前计划内容 + 企业搜索命中的用户有权文件 + 手动附加文件 + M365 共享 Copilot 基础设施。
- 写：直接写入当前计划（任务/桶/目标）。FAQ 未描述任何 preview/apply 或确认工作流，即生成物大概率直接落计划。
- 权限：沿用 M365 权限体系（只能看到用户有权访问的内容）。

## 许可与限制
- 需要 Planner and Project Plan 3、Plan 5 或 Microsoft 365 Copilot 许可；preview 阶段最终定价未公布。
- 支持 8 种 prompt 语言，含简体中文。
- 生成任务仅标题、无工期/依赖，限制明确写在 FAQ。

## 可蒸馏点（对刀4/刀5）
1. **prompt 分类学「Create / Understand / Edit / Ask」**可直接映射到 schedule_get（Understand/Ask）、schedule_apply（Create/Edit）、schedule_analyze（Ask/报告）三件套的工具语义设计。
2. **右侧 Copilot 侧栏 + 建议 prompt + 自由输入**是 office 系用户最熟的对话形态；gaea 对话式调整（刀5）可复用该布局语言。
3. **「/」手动附加文件作为强上下文**（always used as context）值得借鉴：让用户显式投喂招标文件/Excel 工作清单给 schedule 生成。
4. 企业搜索「每次请求后台自动带上下文」的思路对应 gaea 的 workspace 检索（计划文件落 workspace 后，agent 天然可读）。

## 不可取点
1. **无 preview/apply 确认流**（FAQ 未描述），生成物直接进计划——对进度计划这种强结构数据风险高，刀5 必须做 diff 确认。
2. **只生成标题不生成工期/依赖**，所谓「full plan」实际是 WBS 骨架——宣传口径与排程深度不符；gaea 有确定性 CPM 引擎，应让 AI 产出结构化字段（工期/搭接/日历）由引擎裁决，而非停在标题层。
3. Teams/许可墙高（Plan 3/5 或 Copilot 许可），preview 长期不毕业；提示「AI 能力与核心数据能力解耦、别绑死在单一入口」。
