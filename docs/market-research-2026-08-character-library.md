# 市场调研：统一角色库（小说角色 × 轻语角色），2026-08

> 目的：为 gaea「角色库」板块提供设计依据，并明确两个下游场景——小说创作如何使用角色、聊天陪伴如何使用角色。来源均为 2026 年公开资料，侧重点在角色资产的一体化管理。

## 1. 行业共识：角色卡不是静态档案，而是 AI 生成时的"实时引用源"

- Sudowrite Characters：角色卡 = 结构化画像（人格一段话 + 对话样本 + 具体外貌 + 背景 + 关系映射），写作时由 Write 引擎**自动引用**，而非作者手动翻阅角色圣经。跨章节上下文可达 2 万字 / 25 章联动；"对话样本（3-5 条）教 AI 说话节奏，比特质列表有效得多"；卡片应随角色弧线持续更新（[Sudowrite](https://sudowrite.com/blog/how-to-develop-fictional-characters-with-ai-sudowrites-characters-feature/)）
- QMAI（青幕）：一致性检查作为专项能力——审查时先抽取章节出场角色，再逐个对照记忆库的「角色光环 / 人物状态 / 角色认知状态」（[QMAI](https://github.com/Mochocyang/QMAI)）
- lore-weave：每个章节自动把角色、地点、事件、关系抽取进知识图谱；小说世界观可 NPC 化，让读者"走进自己写的世界"（[lore-weave](https://github.com/letuhao/lore-weave)）
- CharacterArc / aiAIfiction / Hnovel：长篇工作台把项目设定、角色关系、角色状态、大纲、正文、质量审计放在同一空间，角色状态管理是长篇可持续写作的骨架
- 读者侧证据：USC Narrative Intelligence Lab 指出读者在声音跑调的 3 段内即察觉角色不一致；2023 Written Word Media 调研中"角色声音不一致"是读者弃书原因第 2 名（Sudowrite 引述）

## 2. 聊天/陪伴侧：角色定义 = 人格 + 说话方式 + 行为规则 + 记忆状态

- Character.AI：Character Definition 定义"如何思考、说话、行为"——人格、背景、怪癖、说话模式、行为规则、情感逻辑；平台提供模板与完整示例，记忆工具在用户侧（固定记忆 + 自动记忆）（[Character.AI 创作指南](https://support.character.ai/hc/en-us/articles/50608794517915)）
- NeurIPS 2025 四象限人格分类：AI 陪伴、游戏 NPC、功能机器人目标不同，人设系统应按使用场景分型（[Four-Quadrant Taxonomy](https://neurips.cc/virtual/2025/loc/mexico-city/129932)）
- eros-engine（开源陪伴引擎）：一个 persona 跨数千轮保持一致——持久记忆 + 关系模型 + 决策引擎，强调"陪伴 = 一人格 × 多会话 × 关系演进"（[eros-engine](https://github.com/etherfunlab/eros-engine)）
- character-sim：MBTI/OCEAN 人格科学 + 分层记忆 + 实时特质演化，角色在对话中逐步细化（[character-sim](https://github.com/hugoloubser/character-sim)）
- Ginger：独立角色卡编辑器（.png/.charx/.json），支持**聊天中实时编辑角色**——角色管理不应与聊天隔离（[Ginger](https://github.com/DominaeDev/ginger)）

## 3. gaea 角色资产现状盘点

| 资产 | 载体 | 字段/能力 | 消费方 |
|---|---|---|---|
| 小说角色 | 项目 `characters.json`（`types.CharacterFile`） | id/name/role_type/gender/age/personality/background/appearance/figure/motivation/arc/status/portrait_url；关系 + 组织 | 小说 Agent 经 `PromptView` 注入（剥离开 portrait base64）；角色对话/试演 |
| 轻语人格预设 | `internal/whisper/personality.go`（16 种） | 五维 T/I/S/O/R + gender + tags + voiceGuide + requiresAdult18 | 轻语 Orchestrator（记忆/情绪/信任/轮次） |
| 轻语虚拟助手 | assistant 存储 | 名称 + 绑定人格 + 微信通道 + 剧照 + 启停 | 微信/聊天切换人格 |
| 会话记忆 | hermes.db | facts / emotion / trust / desire / trace | Orchestrator 跨轮一致性 |

现状问题：两套角色资产各自为政，schema 不同、入口分散（小说页 + 聊天全屏面板），互导靠手工按钮（小说→轻语、轻语→小说），缺少统一资产视图。

## 4. 设计决策：角色库 = 统一资产中心（本轮落地）

- 导航：独立板块「角色库」，位于模型中心之后
- 小说角色 Tab：完整 CRUD + 筛选 + AI 生成 + 剧照 + AI 补全 + 导入轻语（复用现有组件，数据仍落项目）
- 轻语角色 Tab：人格预设只读浏览（五维雷达 + 标签 + 设为当前）+ 虚拟助手完整管理（原全屏面板页面化）
- 双向互导保留为显式动作；切换人格通过 `gaea-persona-changed` 事件与聊天板块联动

## 5. 后续：小说如何使用角色（调研 → 增强清单）

1. **对话样本字段（voice samples）**：Sudowrite 证明 3-5 条对话样本 > 特质列表。小说角色卡新增 `dialogue_samples`，章节/对话生成注入；角色试演（ChatCharacter）的产出可沉淀回写
2. **角色状态快照注入**：QMAI 的「人物状态 / 认知状态」。小说注入不应只给初始设定，而应带"当前弧线阶段 + 状态变更"（身份/关系/认知），让 Agent 写的是"此刻的角色"
3. **一致性审计**：基于角色库对照（性格、说话方式、外貌、认知状态）对成稿做角色一致性检查，发现即回改——现有 Analyze/质量审计可扩展
4. **关系图谱消费**：角色库卡片展示关系网络，生成时按"在场角色"注入关系（lore-weave 思路，在场角色才注入，节省上下文）
5. **跨项目资产复用**：角色库作为资产中心后，新项目可从库中导入既有角色（克隆到项目 characters.json）

## 6. 后续：聊天如何使用角色（调研 → 增强清单）

1. **角色定义结构化**：Character.AI 五要素落地——轻语角色在 voiceGuide 之外补"行为规则 / 情感逻辑"结构化字段（可用 YAML/JSON 块），Orchestrator 注入
2. **人格实时编辑**：Ginger 思路——角色库编辑即时生效，下一次对话即用新设定（当前已是即时读配置，无需重启）
3. **dims 随关系演化**：character-sim 思路——五维静态预设 → 随 trust/emotion 微调（如信任高时 S 上升），在情绪面板体现"关系中的角色"
4. **人格记忆隔离**：已有 hermes.db 按会话/人格隔离，角色库"设为当前"后会话无缝衔接（本轮已打通事件）
5. **小说角色 → 聊天角色的 voiceGuide 自动生成**：导入时用角色卡 personality/background 自动生成说话方式，而非裸用角色名

## 7. 落地优先级建议

- 本轮（已实现）：统一入口 + 双 Tab 管理 + 互导 + 人格联动
- 下一轮优先：小说角色卡 `dialogue_samples` + 弧线状态注入（小说一致性收益最大）
- 其次：轻语角色定义结构化（行为规则/情感逻辑）、dims 演化、跨项目角色导入

## 8. 结论

角色库的本质是「角色资产的单一事实源」：小说侧把它当"AI 实时引用的角色圣经"，聊天侧把它当"人格与角色的管理面板"，两侧通过同一资产视图互通（导入/导出/设为当前）。本轮先把入口与治理统一，后续再按上表增强消费侧的注入与一致性能力。
