# Genspark（Super Agent）调研原始稿

> 调研日期：2026-09-06。定位：通用「super agent」全栈 AI 工作区（slides/docs/图片/视频/代码），非 PM 专用，但代表 agent 产品通用交互与工具编排口径。PM 排程专项功能未查到。

## 来源 URL
- Super Agent 官方帮助：https://www.genspark.ai/helpcenter/super-agent
- 官网：https://www.genspark.ai/
- 评测：https://skywork.ai/skypage/en/genspark-ai-super-agent-review/2036648238339690496 ；https://mpgone.com/manus-ai-vs-genspark-ai-the-battle-of-next-gen-super-agents/ ；https://www.usecarly.com/blog/genspark-vs-chatgpt-agent/ ；https://www.saner.ai/blogs/best-genspark-alternatives ；https://www.eesel.ai/blog/genspark-ai

## 能力清单（AI 能做什么）
- Super Agent：自主理解目标→拆子任务→自选工具（官方称 80+ 内置工具+多模型路由）完成日常任务；产出含 slides、文档、调研、代码、文件等。
- 与 PM 相关：可生成项目类交付物（计划文档/幻灯/表格），**未查到**甘特图/CPM/进度调整专项能力或案例。
- 2026 年营销口径：unlimited AI chat/image。

## 交互形态
- 单一聊天入口给目标，agent 全自主执行（「thinks, plans, and acts」）；用户不打断，产物以文件/页面形式回传。
- 评测普遍把 Genspark 归为「易用性优先」（vs Manus 的企业深度优先）。

## 数据流（读什么/写什么）
- 读：用户目标+web+其内建工具集上下文。
- 写：以「生成文件/页面」为主，**未查到**对第三方系统（PM 工具等）的结构化写回；无与 PM 工具的双向同步口径。

## 可蒸馏点（对刀4/刀5）
1. 「一次聊天→多步自主→文件产物」的交互天花板：通用 agent 做计划只能产文档，不能产可计算模型——反向印证 gaea 的差异化：**计划必须落在结构化模型+引擎上**，这正是刀4「计划文件落 workspace」的价值。
2. 80+ 工具按需路由的 agent 架构说明：工具粒度要小而可组合（schedule_get/apply/analyze 拆开是对的，不要合成一个「manage_schedule」）。
3. super agent 产品的「跑完给产物包」心智，用户预期是拿到「成品」——gaea 生成计划后应同样给「可直接打开的成品计划」而非聊天里的文字清单。

## 不可取点
1. 无领域校验：agent 自由发挥的产物（如计划表）无正确性背书，工程场景不可直接用。
2. 产物单向（生成文件），无回流编辑闭环——对比 Ingantt「use as-is or tweak」弱一档。
3. 营销口径（unlimited/80+ tools）与 PM 场景无公开可验证交叉，本次仅能确认其通用 agent 定位。
