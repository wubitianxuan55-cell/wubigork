# docs/archive · 归档说明

> 归档时间：2026（v3.7.0 之后，长期规划定稿时）；**最近一次复核：2026-09-10**（顶层 53 项逐条登记，此前表格只用通配描述、实际漏登 14 项）；**2026-11-24 更新权威指向**（用户明示当前没有长期规划，`.gaea/AGENTS.md` 同日清理）。
> 原则：**历史调研、已落地计划、被后续结论纠正/取代的文档一律归档**，防止后续会话把过时结论当"当前事实"读取。归档不删除——git 历史与本部目录均可查。

## 权威文档（后续会话请以这些为准）

**完整索引与状态行见 `docs/README.md`**（2026-09-10 重建，45 份顶层文档全覆盖）。本节只列最高频的四类：

- `docs/archive/gaea-nextgen-roadmap-2026.md` —— 全域路线图（2026-09 制定）。**2026-11-24：用户明示当前没有长期规划，本档不再是权威**，仅作历史参考。
- `docs/archive/gaea-slim-masterplan-2026-09.md` + `docs/archive/gaea-convergence-plan-2026-09.md` —— 当时的两层规划（瘦身/收敛）。**2026-11-24：已过时**，不再称活跃。
- `.gaea/AGENTS.md`（长期记忆：现行规则与决策；2026-11-24 清理后已在 32KB 注入预算内）与 `.gaea/progress.md`（各线进度与包记录）、`.gaea/todos.md`（开放任务）。
- 政策与评测：`docs/ADULT_MODE.md` / `DREAM_WRITE_POLICY.md` / `MEMORY_ARCHITECTURE.md` / `evaluation-set.md` / `retrieval-eval-set.md`；环境与架构基准：`docs/archive/2026-08-14-sandbox-environment-notes.md` / `docs/archive/2026-08-15-gaea3-architecture-design.md`。

## 子目录

| 目录 | 内容（文件数） | 归档原因 |
|---|---|---|
| `market-research-2026-08/` | 14 份 2026-08 板块调研 | 结论已被 2026-09 全板块新调研取代/纠正（如"灵犀=阿里/通义"已被纠正为 WPS 灵犀） |
| `superpowers-plans/` `superpowers-specs/` | 历史执行计划与规格（40 + 15） | 已落地或已被取代（方案编写板块已删除、移动端已冻结、UI 已重设计多版） |
| `gaea3-review/` `gaea3-vision-research/` | 3.0 时代审查与愿景调研（11 + 7） | 只读参考结论已消化；愿景已由长期规划 §10 版本重定义取代 |
| `gaea2/` | 2.x 设计文档（2） | 已被 3.0 架构取代 |
| `audits/` | 历史审查（.gaea/reviews）、phase7-candidates、data_backup_review、herdsman API 文档（8） | 已过时；最新执行审计见本目录顶层 `audit-2026-08-30-v4-execution-review.md` |
| `research-2026-08-31/` | 08-31 模块制调研原始稿（8） | 合成版见 `market-research-2026-08-31.md`，结论已随 v4.12/v4.13 交付 |
| `research-2026-09-01/` `-01b/` `-02/` `-02b/` `-03c/` `-05/` `-05b/` `-06/` | 2026-09 各线调研原始稿（4/2/3/3/2/3/1/15） | 结论已合成进 docs/ 各权威文档并随版本落地（v4.25~v4.126）；原始稿只留来源与证据（`-06` 15 份对应 v4.113.0 进度计划线） |

## 顶层遗留文件（逐条登记）

### 版本 / 进度磁带（大件，检索入口）

| 文件 | 内容 | 说明 |
|---|---|---|
| `agents-version-history-2026-09.md` | `.gaea/AGENTS.md` 版本段磁带 | **覆盖 v4.173–v4.146 + v4.49 及更早（至 v3.0.6）**；2026-09-09 首迁 + 2026-09-10 二迁（AGENTS 超工作区指令预算 65536 B）。**v4.50–v4.145 不在本件**，该区间只在 `CHANGELOG.md` / `releases/` |
| `progress-history-2026-09.md` | `.gaea/progress.md` 历史磁带 | v4.170.0 及之前的逐版发布记录（2026-09-09 迁出，progress.md 由此瘦身为「最近发布速览」） |
| `agents-cleanup-2026-11-24.md` | `.gaea/AGENTS.md` 2026-11-24 清理时移出与改写段落的原文（文首专项入口、11-12~11-14 状态速览、长期规划、本地 TTS 引擎等） | 起因=文件 40KB 超过 32KB 注入预算、尾部被截断，且用户明示当前没有长期规划；**不作为现行依据**，取代者=`.gaea/AGENTS.md` 现行段落与 `.gaea/progress.md` |

### 执行审计与规划（结论已被后续取代，但仍是承接依据）

| 文件 | 归档原因 |
|---|---|
| `audit-2026-08-30-v4-execution-review.md` | v4.x「承诺 vs 代码」对照（裁决=最小版执行）——曾被 AGENTS「长期规划」段摘要；该段已于 2026-11-24 移出（见 `agents-cleanup-2026-11-24.md`），本档仅作历史参考（原路径写作 `docs/`，实际在本目录，引用时用 `docs/archive/…`） |
| `gaea-u4-render-evidence-inventory-2026-09.md` | U4 渲染证据清单（U 系蒸馏已收官 v4.97~v4.109） |
| `gaea-v41-evidence-chain-design.md` / `gaea-v415-auto-route-design.md` / `gaea-v42-cost-ai-design.md` / `gaea-v43-play-deepen-design.md` / `gaea-v482-realtime-s2-design.md` | v4.1~v4.8 期专题设计（证据链/自动路由/成本 AI/乐园深化/Realtime S2），均已落地或被现行设计取代 |
| `better-sidebar-port-2026-09-03-worktree.md` / `dsh-better-sidebar-go-port-plan-2026-09.md` | 侧边栏整包 Go 化方案与 worktree 记录——**路线未采纳**，被滚动蒸馏规划取代 |
| `2026-08-27-dsh-context-go-port.md` | dsh-context 移植记录（结论已消化） |
| `2026-08-office-cloud-local-architecture-review.md` / `2026-08-office-knowledge-stack-optimization-summary.md` / `2026-08-16-office-file-interaction-research.md` | 2026-08 办公架构/知识栈/文件交互调研，结论已随版本落地 |
| `2026-08-19-cost-db-reorganize-report.md` / `2026-08-19-cost-library-redesign-zaojia.md` | 造价库重组与改版报告（现行为见 `docs/archive/gaea-cost-domain-survey-2026-09.md`） |
| `gaea-schedule-multi-project-design-2026-09.md` / `gaea-schedule-resource-cost-design-2026-09.md` / `gaea-schedule-aoa-manual-layout-design-2026-09.md` | 进度计划三份已实施设计（v4.112~v4.143 交付）；代码注释仍可能引用为设计出处 |

### 3.0 时代与更早

| 文件 | 归档原因 |
|---|---|
| `2026-08-15-gaea3-vision-roadmap.md` / `2026-08-15-gaea3-ui-constellation-os.md` / `2026-08-15-gaea3-ui-design-system.md` | 3.0 愿景与 UI 设计语言——已被 v4 双空间重设计与长期规划 §10 取代 |
| `2026-08-09-voxcpm2-integration.md` / `2026-08-09-cosyvoice2-llm-gguf-speed-optimization.md` | 本地 TTS 历程（VoxCPM2 已于 v2.6.9 移除）；**当前语音引擎现状见 `.gaea/AGENTS.md`「本地语音引擎」段（CosyVoice2 亦已于 2026-10 退役）** |
| `2026-08-12-herdsman-models-evaluation-report.md` / `2026-08-13-herdsman-tts-ocr-comparison.md` | herdsman 模型测评与 TTS/OCR 对比（时点结论，现以 herdsman 实际探测为准） |

### 2026-09 市场调研（合成版 + 原始稿目录见上表）

`market-research-2026-08-31.md`（模块制复扫合成，v4.11.0 基线）· `market-research-2026-09-01.md` / `market-research-2026-09-01b.md` / `market-research-2026-09-02.md` / `market-research-2026-09-02b.md` / `market-research-2026-09-03c.md` / `market-research-2026-09-03d.md` / `market-research-2026-09-05.md` / `market-research-2026-09-06.md`（各线合成版）——结论均已进入 docs/ 权威文档并随版本落地。


## 2026-11-24 清理移入（41 份）

用户令「许多记载的文档已经过期可以清理」。原在 `docs/` 顶层；移入原因与取代者如下。状态为移入当时所述，**不作为现行依据**。

| 文件 | 归档原因 | 取代者 / 现状 |
|---|---|---|
| `gaea-nextgen-roadmap-2026.md` | 规划已收官 / 未立项 | 原称「全域路线图（唯一权威）」；用户 2026-11-24 明示当前没有长期规划。现行决策=`.gaea/AGENTS.md`「在册决策与边界」 |
| `gaea-longterm-plan-2026-10.md` | 规划已收官 / 未立项 | 阶段八~十一已于 2026-10-15 全清 |
| `gaea-v5-case-os-proposal-2026-10.md` | 规划已收官 / 未立项 | V5 立项建议，未立项（当前没有长期规划）；如需重启从本档起步 |
| `gaea-v5-zcode-debate-2026-10-29.md` | 规划已收官 / 未立项 | V5 辩论止于 2026-10-29，未立项 |
| `gaea-pantheon-v2-roadmap-2026-11-03.md` | 规划已收官 / 未立项 | 众神殿 V2 协议已落地；现状见 `.gaea/progress.md` 众神殿各条 |
| `gaea-pantheon-market-research-2026-10-29.md` | 规划已收官 / 未立项 | 圆桌讨论 V2 已实现 |
| `gaea-competitive-landscape-2026.md` | 规划已收官 / 未立项 | 竞品格局基线 2026-08-29，已过时效 |
| `gaea-competitor-survey-2026-10.md` | 规划已收官 / 未立项 | 阶段八立项证据档；阶段八已收官 |
| `gaea-whisper-deadcode-survey-20260925.md` | 规划已收官 / 未立项 | 2026-09-25 入拍板池后未排刀，过期 |
| `gaea-image-domain-longterm-plan-2026.md` | 规划已收官 / 未立项 | T0 契约已落地（v4.98.0），T1+ 未启动；当前没有长期规划 |
| `gaea-computer-use-exec-brief-2026-11.md` | 规划已收官 / 未立项 | 电脑操控八刀已于 2026-11-20 落地装机 |
| `gaea-computer-use-landing-research-2026-11.md` | 规划已收官 / 未立项 | 同上（施工单的调研依据） |
| `gaea-ai-workbench-ui-2026-10.md` | 规划已收官 / 未立项 | 刀1~7 已随 v4.527.0 发版收官 |
| `gaea-board-slim-reorg-2026-10.md` | 规划已收官 / 未立项 | 纪要；其中小说质检部分随小说板块 2026-11-15 退役 |
| `gaea-dsh-distill-plan-01-2026-11.md` | 规划已收官 / 未立项 | 蒸馏计划 01，首批 D1–D7 已于 2026-11-18 收官 |
| `gaea-dsh-better-sidebar-long-term-distill-plan-2026.md` | 规划已收官 / 未立项 | 自述为历史台账；现状见 `docs/gaea-better-sidebar-go-iteration-2026-11.md` |
| `gaea-tianxuan-research-2026-10.md` | 引擎已退役 | tianxuan 引擎于 2026-11-05 退役，35B 线由自管 llama 接手 |
| `gaea-tianxuan-strata-deepdive-2026-10-18.md` | 引擎已退役 | 同上 |
| `gaea-moe4all-engine-assessment-2026-10.md` | 引擎已退役 | MoE4All 后改名 tianxuan，于 2026-11-05 退役 |
| `gaea-zcode-distill-2026-10.md` | 线已闭环的过程档 | ZCode 蒸馏线 v1 已随 v4.525.0 收官 |
| `zcode-office-distill-2026-10.md` | 线已闭环的过程档 | 同上 |
| `zcode-office-distill-analysis-2026-10.md` | 线已闭环的过程档 | 同上 |
| `zcode-office-decision-memo-2026-10.md` | 线已闭环的过程档 | 同上 |
| `zcode-dialogue-2026-10-24.md` | 线已闭环的过程档 | 同上（对话记录） |
| `zcode-distill-dialogue-2026-10.md` | 线已闭环的过程档 | 同上（对话记录） |
| `zcode-handoff-2026-10-24.md` | 线已闭环的过程档 | 同上（交接档） |
| `gaea-dsh-021-core-distill-2026-10.md` | 线已闭环的过程档 | dsh 0.2.1 时代蒸馏档，刀池已清；上游现为 0.2.2-rc.5 |
| `gaea-dsh-022-distill-2026-11.md` | 线已闭环的过程档 | 同上 |
| `gaea-dsh-subagent-distill-2026-10.md` | 线已闭环的过程档 | 同上（子代理板块） |
| `gaea-dsh-subagent-deep-distill-2026-11.md` | 线已闭环的过程档 | 同上 |
| `gaea-dsh-subagent-ui-distill-2026-11.md` | 线已闭环的过程档 | 同上（子代理 UI） |
| `gaea-t0-effect-audit-2026-11-08.md` | 线已闭环的过程档 | 众神殿 T0 effect 机械审计证据，结论已入 progress |
| `gaea-chat-probe-baseline-20261106.md` | 线已闭环的过程档 | 聊天人格实弹对照报告（2026-11-06），结论已入 progress |
| `gaea-chat-probe-armb-20261106.md` | 线已闭环的过程档 | 同上 |
| `gaea-chat-probe-delivery-contract-20261106.md` | 线已闭环的过程档 | 同上 |
| `gaea-web-search-pantheon-digest-20261105.md` | 线已闭环的过程档 | 论证资料整理；网络搜索三刀已收官 |
| `gaea-contextview-upgrade-research-2026-11.md` | 线已闭环的过程档 | 上下文界面 1:1 复刻已随 v4.542.0 发版 |
| `gaea-contextview-complete-parity-2026-11.md` | 线已闭环的过程档 | 同上 |
| `gaea-novel-revolution-2026.md` | 功能已退役 | 小说板块于 2026-11-15 整体删除 |
| `gaea-novel-skill-distill-survey-2026-10.md` | 功能已退役 | 同上 |
| `gaea-reasonix-2x-distill-2026-10.md` | 线已闭环的过程档 | Reasonix 2.x A/B 档已全清并随 v4.524.0 发版 |

### 此前只登记在 `docs/README.md`、实际早已在本目录的 43 份（2026-11-24 补登）

归档时索引行没有跟着迁过来。下表「内容」取自原 `docs/README.md` 登记行（最后一份原只在索引的散文段里提到）。

| 文件 | 内容（原登记摘要） |
|---|---|
| `gaea-next-stage-plan-2026-09.md` | 已过时（2026-11-24 标注，已归档）——原「下一阶段规划大纲（活跃指导）」，阶段五/六已收官 |
| `gaea-stage7-plan-2026-09.md` | 已收官（出口判据 2026-09-17 复核全满足；已归档）——原「阶段七规划（平台收口与复利层，2026-09-15 拍板启动）」 |
| `gaea-market-survey-2026-09.md` | 市场调研增量刷新（v4.240 落地） |
| `gaea-platform-market-survey-2026-09-15.md` | 平台级市场调研（阶段七前置，2026-09-15） |
| `gaea-mcp-hub-reversal-review-2026-09.md` | MCP 双向枢纽翻案评估（拍板池条2，建议维持排除） |
| `gaea-backend-perf-survey-2026-09.md` | Go 后端性能热点普查（2026-09-12，交拍板池刀路 A-E） |
| `gaea-priceband-datasource-research-2026-09.md` | 价格带数据源调研（拍板池候选4，v4.275 结案建议） |
| `gaea-memory-graph-51-design-2026-09.md` | 记忆语义图谱 5.1 设计（阶段五域内基线） |
| `gaea-memory-injection-eval-design-2026-09.md` | 记忆注入质量评测设计（市场调研候选2，v4.241 落地） |
| `gaea-office-dag-63-design-2026-09.md` | 办公多文件 DAG 6.3 设计（阶段六域内基线） |
| `gaea-outlook-longterm-plan-2026-09.md` | 长期规划（已收官） |
| `gaea-optimization-direction-2026-09.md` | 2026-09-26 全仓审计衍生的优化方向候选池（2026-11-24 起随「当前没有长期规划」失效，仅存档） |
| `gaea-office-upgrade-plan-2026-09.md` | 主体已落地（分期 v4.23–v4.31） |
| `gaea-office-mindmap-base-design-2026-09.md` | M1/B1/B2 看板/M2 画布编辑（v4.108.0）已落地 |
| `gaea-pptx-edit-design-2026-09.md` | 刀1 v4.109 / 刀2+刀3 v4.156 已交付 |
| `gaea-character-domain-survey-2026-09.md` | 阶段四读档（结论=一套资产 + 项目工作副本，非三套人） |
| `gaea-dream-studio-nextgen-2026-09.md` | 材料汇编+下一代草案，不承诺版本（§0.5 已被图域 longterm-plan 吸收） |
| `gaea-image-domain-t0-contract-design-2026-09.md` | 已落地随 v4.98.0 |
| `gaea-sin-board-design-2026-09.md` | 闲庭·原罪板块（v4.256.0） |
| `gaea-reasonix-v138-distill-2026-09.md` | Reasonix 真身蒸馏（esengine/DeepSeek-Reasonix v1.15→v1.38） |
| `gaea-dsh-016-distill-2026-09.md` | dsh（deepseek-harness TS monorepo）蒸馏（原误题 Reasonix，已勘误更名） |
| `gaea-dsh-univer-office-distill-plan-2026-09.md` | 已蒸馏收官（U 系落地 v4.97~v4.109；Univer 零引入） |
| `gaea-dsh-genui-distill-plan-2026-09.md` | 已收官（P0–P5 全部发布，止于 v4.97.0） |
| `gaea-unsloth-modelhub-distill-plan-2026-09.md` | 已退役（2026-10-18 modelhub 引擎整体删除：引擎/Key/MH4 预热随 Unsloth 退役下线；CU1/CU2 等 ComfyUI 侧蒸馏成果保留）。本文保留作历史蒸馏记录 |
| `gaea-schedule-gap-vs-project-2026-09.md` | 收官总账 |
| `gaea-schedule-diff-confirm-design-2026-09.md` | 已收官（四刀 v4.146~149） |
| `gaea-schedule-dual-duration-design-2026-09.md` | 已收官（四刀 v4.150~153 + 搭接放开 v4.155） |
| `gaea-schedule-projectlibre-distill-2026-09.md` | 机制供给参考（蒸馏成果已全部落地） |
| `gaea-schedule-standards-digest-2026-09.md` | JGJ/T 121-2015 规范权威底稿（缺陷清单已修，v4.129/130） |
| `gaea-slim-baseline-2026-09.md` | P0 基线证据层 |
| `gaea-slim-knife2-audit-2026-09.md` | 审计证据 |
| `gaea-slim-p2-dualspace-2026-09.md` | 审计证据（P2 双空间并列） |
| `webview2-shell-audit-2026-09.md` | 审计完成（P0×2/P1×7 + 刀序 A–D） |
| `gaea-office-hub-decouple-audit-2026-09.md` | 已收刀（v4.197：office/docmd → internal/docmd，内核内边清零） |
| `gaea-session-binding-audit-2026-09.md` | 无参绑定会话语义审计收官（v4.338.0） |
| `gaea-sin-booksource-distill-2026-09.md` | 原罪·书源引擎蒸馏规格（so-novel 机制重推导；AGPL 红线=不搬代码与规则 JSON） |
| `gaea-sin-booksearch-distill-2026-09.md` | 原罪·书源检索蒸馏规格（并行线在制品） |
| `gaea-novel-ohstory-distill-2026-09.md` | 小说板块·oh-story-claudecode 蒸馏规格（MIT：机制重推导 + 知识资产带许可收录） |
| `gaea-novel-booksource-import-2026-09.md` | 小说板块·书源取书→拆书导入接通刀路（用户拍板 2026-09-13：两板块不合并改接通；2026-10-27 修订：两板块导航级合并，原罪收编为小说第六页签） |
| `gaea-longform-novel-system-2026-09.md` | 长篇小说创作系统方向（2026-09-28 立项，用户指令「解决如何写一个优秀的长篇小说」） |
| `gaea-longform-craft-mechanics-2026-09.md` | 长篇创作机制调研交付物（2026-09-28，方法论→可机械化的 14 机制 M1–M14 + 25 条「优秀长篇」机械判据（22 条可确定性自动化）+ 故事层/场景层/文本层三层架构 + 主流工具对应层做法与短板） |
| `gaea-getting-started-30min-2026-09.md` | 30 分钟上手（2026-09 定稿） |
| `gaea-genui-memoryfence-audit-2026-09.md` | 审计存档；6/7 项 v4.101 收口，resume 槽位口径=唯一开放项 |

## 归档纪律

1. **归档不删除**：文件移入本目录并在本 README 登记一行；不写「等」「若干」这类无法核对的量词。
2. **权威指向现行**：本目录任何结论被现行文档纠正后，登记行必须注明取代者（否则后续会话会把旧结论当现状）。
3. **磁带类大件单独说明覆盖面**（含缺口区间），避免「以为全都有」。
