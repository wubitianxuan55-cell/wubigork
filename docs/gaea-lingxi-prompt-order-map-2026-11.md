# 灵犀↔gaea 提示词装配顺序对照（batch2 B3，2026-11-13）

> 输入：`re-lingxi/out/PROMPT-skeleton.md`（130 节点序+四规律，re-distill §26.1）、
> gaea 实码（`internal/gaea/boot/sysprompt.go buildSystemPrompt` + `boot.go` 544~555）。
> 结论先行：**四规律中 gaea 天然满足三条，两条不满足项经性质分析判定不调序**；
> SingleModelPrompt 逐字节红线不动（被既有测试锚定，波3.5 先例）。调整与否留
> A/B 实验设计（§3），证据到手前不动结构。

## 1. gaea 实际装配序（实码锚点）

**缓存稳定前缀**（`buildSystemPrompt`，跨会话不变）：

| # | 段 | 锚点 |
|---|---|---|
| 1 | 主系统提示词（agent_loop/工具纪律主体，单文件整块） | `cfg.ResolveSystemPrompt()` |
| 2 | 部署级 PromptPrefix（最前置） | sysprompt.go:49 |
| 3 | outputstyle 输出风格 | :53 |
| 4 | LanguagePolicy 语言策略 | :55 |
| 5 | GenUI OfficePointer | :56 |
| 6 | memory.Compose（画像 ProfileBlock 600rune+记忆索引） | :70 |
| 7 | 晨报预载块（work） | :79 |
| 8 | 项目本体 ProjectBrief（work） | :89 |
| 9 | 固化正文 PinnedFactsBlock | :98 |
| 10 | 常用资料 pins.Block | :104 |
| 11 | **技能目录 skill.ApplyIndex（4000 rune 预算）** | :139 |

**运行时尾段**（`boot.Build`，逐会话附加）：

| # | 段 | 锚点 |
|---|---|---|
| 12 | SingleModelPromptFor(space)——空间路径纪律 | boot.go:544 |
| 13 | deliveryPromptSection 交付与收尾 | :548 |
| 14 | officeLivePromptSection（条件：COM 档） | :555 |

## 2. 灵犀四规律逐条对账

| 规律（灵犀 §26.1） | 灵犀证据 | gaea 现状 | 判定 |
|---|---|---|---|
| ①场景开关前置（group_chat/im_scene 最前） | 位置即优先级 | PromptPrefix 最前✓；但 space 段（12）在尾 | **不满足，不调**：gaea 的 space 是 exports/work **路径根纪律**而非会话场景模式，放尾部=紧邻工具调用点语义更贴；性质不同构，不硬套 |
| ②language→format→tool_call_style 先于 agent_loop（先钉形态） | §26.1 | LanguagePolicy/outputstyle（4/3）在主提示词（1）**之后**拼接——但拼接序≠注意力序，且 gaea 主提示词是一整块不可拆 | **部分满足，不动**：主提示词单文件整块是结构底座（与灵犀 130 节点全模板化最大的不同），拆它超出本批红线 |
| ③记忆插在循环之后（先干活再讲记住什么） | memory_system 紧跟 agent_loop | memory.Compose（6）在主提示词（1）之后 ✓；晨报/项目本体/固化（7~9）依次在后 | **天然满足** ✓ |
| ④system_reminder=动态插入位（33 处） | 段库统计 | 运行时尾段（12~14）+会话内 system-reminder 注入即同构 | **满足** ✓ |
| 附：skills 目录位置 | 灵犀=第 2 位（最前） | gaea=第 11 位（稳定前缀末） | **不满足，不调**：gaea 技能=run_skill 工具化调用+4000 rune 目录索引，非灵犀式顶层开关；前置会把索引插进主纪律与语言策略之间，无 A/B 证据不动 |

## 3. A/B 实验设计（若要动，先跑这个）

1. 变体 A（现状）vs 变体 B（技能目录前移至 #3，语言策略之前）。
2. 各跑 10 条真实任务（5 办公+5 通用），对比：技能调用率（run_skill 触发数）、
   「没有该技能」类误答数、交付质量人工评分。
3. 判据：B 组技能调用率提升 ≥20% 且无误答回归才落结构改动；否则维持现状归档。

## 4. 红线与回滚

- SingleModelPrompt 常量逐字节不动（boot_test 系锚定）。
- 缓存稳定前缀边界（#1~#11 vs #12~#14）不跨越：任何调序只在前缀内部或尾段内部进行。
- 回滚=单提交 revert，无迁移面。
