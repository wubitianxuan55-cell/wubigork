# Smartsheet AI 调研原始稿

> 调研日期：2026-09-06。定位：表格型工作管理平台，AI 主打「text-to-X」（自然语言→公式/图表/描述），2025-2026 转向「AI-first Intelligent Work Management」并接 MCP。

## 来源 URL
- 官方 AI 平台页：https://www.smartsheet.com/platform/features/ai
- 官方帮助总览：https://help.smartsheet.com/2483169/AI-tools
- 公式/文本生成帮助：https://help.smartsheet.com/2483096/AI-Formulas-text-generator
- 内容中心文章：https://www.smartsheet.com/content-center/ai-tools-generate-formulas-text-and-summaries
- AI-first 平台发布报道：https://uctoday.com/smartsheet-ai-revolution-intelligent-work
- 学习路径：https://help.smartsheet.com/learning-track/ai-tools

## 能力清单（AI 能做什么）
- **AI-powered project setup**：引导式新手建 workspace（按角色/目标/用例推荐工具并帮建第一个工作区）。
- **Generate formulas**：自然语言→可用公式（text-to-formula）。
- **Analyze data**：打字即出实时图表/指标，可直接加到 dashboard；AI dashboard builder「生成已填充数据的起点」。
- **Generate text & summaries**：任务摘要、文案、情感识别、翻译。
- **Intelligent form fill**：语音输入自动映射到表单字段。
- **CLI Power Tools（agents）**：三个 Claude Code agents，「scan for bottlenecks, reassign work, clone project structures」——瓶颈扫描+改派+克隆项目结构。
- **MCP Server**：连 Claude/ChatGPT/Gemini/Copilot，agents 读实时项目数据并执行：更新状态、生成报告、创建计划、批量行更新、管理列、改派任务。
- **未查到**：依赖/关键路径/工期排程类 AI 能力（Smartsheet 有甘特视图，AI 层未涉及）。

## 交互形态
- 嵌入式（sheet 内公式/文本生成）、引导式（project setup）、提示词驱动（图表生成）、外部 agent（MCP/CLI，无 GUI 对话）。

## 数据流（读什么/写什么）
- 读：sheet 数据、表单语音输入、（MCP）实时项目数据。
- 写：公式入 sheet、图表/指标入 dashboard、文本摘要；（MCP agents）状态更新、批量行更新、列管理、任务改派、创建计划。
- **确认门槛：官方页面全程未描述任何 preview/approve 步骤**；最接近的是生成后可「Edit/refine/customize」（dashboard builder）与 Smart Columns 自定义指令。

## 可蒸馏点（对刀4/刀5）
1. **text-to-formula 是「AI 生成确定性表达式」的成熟范式**：AI 写公式→引擎求值，出错即报错。gaea 完全同构：AI 写结构化计划 JSON→CPM 引擎求值→错误（环/不可行）回传 AI 修复。这是刀5 的技术形态背书。
2. **「clone project structures」+「bulk row updates」**作为 agent 的高频安全动作集合，提示 schedule_apply 应内置批量原语（批量建任务/批量改搭接）而非仅单条。
3. MCP Server 的动作清单（更新状态/生成报告/创建计划/批量更新/管理列/改派）可直接当作 gaea agent 工具集的能力清单对照表。
4. 「AI dashboard builder 生成已填充起点，随后 Edit/refine」= 生成物可继续编辑的口径，与「Use it as-is, tweak it」（Ingantt）一致。

## 不可取点
1. 全线无 preview/确认门槛，agent 可批量改行——表格域侥幸，进度计划域高危。
2. project setup 只面向 trial 用户引导，非通用生产力功能。
3. 「scan for bottlenecks」无口径说明（基于什么信号判定瓶颈），黑箱。
