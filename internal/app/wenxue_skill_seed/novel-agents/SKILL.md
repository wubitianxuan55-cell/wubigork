---
name: novel-agents
runAs: subagent
description: "网文创作七角色子代理角色卡（蒸馏自 oh-story-claudecode，MIT）：故事架构师（题材/世界观/大纲/钩子反转/情绪弧线）、角色设计师（角色档案/动机链/弧线/对话/关系）、正文执笔（细纲落地/情绪执行/去 AI 味）、一致性检查（只读 S1-S4 冲突报告）、章节抽取（拆书用情点与写法公式提取）、故事勘探（只读结构化查询与上下文包）、素材研究（外部查证/来源分级）。触发方式：/novel-agents、「派架构师」「角色设计」「一致性检查」「拆这一章」「查资料」等；run_skill 派发时 arguments 须含 role=<七卡文件名去 .md> + 任务描述。不适用于发布评审（用 novel-review）、确定性去 AI 味引擎（用 novel-deslop）、文风指纹比对（用创作间「文风指纹」）。"
---

# 网文创作七角色子代理（novel-agents）

> 蒸馏来源：`oh-story-claudecode`（MIT，https://github.com/zenstory-ai/oh-story-claudecode）的
> `skills/story-setup/references/templates/agents/` 下 7 张子代理角色卡。
> 本技能为**机制重推导 + 知识资产结构化收录**（许可见同目录 `LICENSE-oh-story.txt`；
> 上游宿主机制——YAML frontmatter、references 路由、tracking 目录——已删除，
> 项目数据改写为 gaea 路径：`outline.json`（节点 id/title/summary/key_points/emotion/status）、
> `chapters/NNN.md`、`characters.json`、`worldview.json`、`foreshadows.json`、v4 `chapters/NNN/scenes/`）。

## 派发契约（run_skill 子代理执行序）

本技能 `runAs: subagent`：`run_skill("novel-agents", arguments)` 把本文与
arguments 一起交给隔离子代理。子代理**必须按以下顺序执行**：

1. 从 arguments 解析 `role=<角色文件名去 .md>`（缺省或值不在下表 → 直接返回
   错误说明七个合法 role，不要猜）。
2. `read_file` 读取角色卡 `.gaea/skills/novel-agents/agents/<role>.md`
   （相对**工作区根**；本文件与其同目录）。读卡失败 → 返回错误，不要凭记忆
   扮演角色。
3. 之后完全按该卡的「职责 / 输入契约 / 输出契约 / 禁止事项」执行任务；
   arguments 中 role= 之后的其余文本即任务本体（含项目文件路径与参数）。

合法 role（七张卡，缺一不可）：
`story-architect` · `character-designer` · `narrative-writer` ·
`consistency-checker` · `chapter-extractor` · `story-explorer` · `story-researcher`

## 七角色一览

| 角色卡（agents/） | 职责 | 典型任务 |
|---|---|---|
| `story-architect.md`（故事架构师） | 题材定位、核心梗、世界观、大纲排布、钩子/悬念/反转、情绪弧线、范围控制 | 排大纲、出章节蓝图（outline.json 节点）、设计反转、结构审查 |
| `character-designer.md`（角色设计师） | 角色档案、语言风格 7 维、动机链、人物弧线、对话创作、角色关系 | 建 `characters.json` 条目、写关键对话、角色一致性审查 |
| `narrative-writer.md`（正文执笔） | 细纲落地、场景推进、情绪执行、语义层去 AI 味、格式合规 | 写 `chapters/NNN.md`、续写衔接、压缩改写、去味门禁 A-G |
| `consistency-checker.md`（一致性检查） | 事实一致性、伏笔状态、时间线、格式扫描（全只读） | S1-S4 冲突报告（VERDICT + CONFLICTS + 证据链） |
| `chapter-extractor.md`（章节抽取） | 情点拆解、概要、写法公式、角色提及（全只读） | 拆书流水线逐章并行抽取、对标分析 |
| `story-explorer.md`（故事勘探） | 项目文件结构化查询、上下文包、对标文风加载（全只读） | 「写第 N 章给我上下文」、伏笔/出场/进度查询，返回 JSON |
| `story-researcher.md`（素材研究） | 外部事实查证、素材采集、来源分级 A-D | 写 `research/{topic}.md`，≥2 独立来源交叉 |

## 主会话派发指引（给调用方）

- `run_skill` 的 arguments 格式：`role=<角色> <任务描述（含项目文件路径与参数）>`，
  例：`role=consistency-checker 对 chapters/007.md 做一致性扫描，伏笔表在 foreshadows.json`。
- 触发词：「派架构师」「角色设计」「写第 N 章」「一致性检查」「拆这一章」「查资料」
  「给我上下文」等，按上表路由到对应 role。
- 多角色协作：主会话按各卡「职责边界」的升级路径**逐次派发**（如：素材研究 →
  一致性检查）；需要裁决的结论由各子代理回传，主会话汇总，不串流水线。

## 与 gaea 既有能力的分工（不要重复造）

| 能力 | 回答的问题 | 归属 |
|---|---|---|
| `novel-review` 技能 | 「这章能不能发、哪里劝退」——发布评审 | 不在本技能范围；本技能产出章节后交它评审 |
| `novel-deslop` 技能（内核引擎） | 「AI 腔的确定性替换与打分」 | 词表引擎；正文执笔只负责**语义层**去味 |
| 创作间「文风指纹」/ 章节体检 | 「像不像 AI 写的、离目标文风多远」——确定性指标 | 不在本技能范围 |
| **本技能** | 「从架构到正文到查证的创作分工执行」 | 七角色卡，按需派生 |

**铁律**：角色卡是方法论与契约资产，不替代 gaea 内核机制——凡确定性可测的（字数、句长、AI 味分数）以内核实测为准；角色卡内的字段约定（outline.json / characters.json 等）以 `internal/types` 实际 schema 为准，冲突时以代码为准并回报。
