# docs/ 文档索引

> 2026-09-09 整理立此索引（同日状态大清算：已完成文档的状态行全部修正为终态，原始调研目录归档）。**归档件在 `docs/archive/`**（已实施设计、一次性调研/审计、版本历史存档与 8 个 research-2026-09-* 原始稿目录）；发布物全文在 `releases/`；版本动态速览在 `.gaea/AGENTS.md`。**引用某文档前先看其头部状态行——状态与 git log 逐版核对过。**

## 总纲（权威排序依据）

| 文档 | 一句话 |
|---|---|
| gaea-nextgen-roadmap-2026.md | 全域路线图（权威；§16 调研回填后近期落地项见 v4.88/4.89） |
| gaea-competitive-landscape-2026.md | 竞品格局与差异化定位（参考基线 2026-08-29，注意时效） |
| gaea-cost-domain-survey-2026-09.md | 造价域现状基线（file:line；**校正 roadmap §15 过时欠账描述**：AI 组价 v4.2 已在产）+ 造价刀路池 |

## 域方案

| 文档 | 状态 |
|---|---|
| gaea-office-upgrade-plan-2026-09.md | ✅ 主体已落地（分期 v4.23–v4.31 全数发布）；「远期」行（终端 tab/双工作台/侧边对话等）维持拍板门控 |
| gaea-office-mindmap-base-design-2026-09.md | 🔄 活跃：M1/B1/B2 看板/M2 画布编辑（v4.108.0）均已落地；**B2 后半（字段面板/画廊）待拍板** |
| gaea-pptx-edit-design-2026-09.md | ✅ 刀1 v4.109 / 刀2+刀3 v4.156 已交付（三件套编辑闭环）；刀4 待反馈；真机走查挂池 |
| gaea-novel-revolution-2026.md | 🔄 刀1–6 已落地（v4.77 批次）；GenerationGate 闭环/刀7 续/刀8 未落地；兼材料汇编 |
| gaea-dream-studio-nextgen-2026-09.md | 材料汇编+下一代草案，不承诺版本（§0.5 已被图域 longterm-plan 吸收） |
| gaea-image-domain-longterm-plan-2026.md | 长期路线：T0 契约已落地（v4.98.0）；T1+ 未启动 |
| gaea-image-domain-t0-contract-design-2026-09.md | ✅ 已落地随 v4.98.0 |

## 蒸馏规划（取道不取器）

| 文档 | 状态 |
|---|---|
| gaea-dsh-univer-office-distill-plan-2026-09.md | ✅ 已蒸馏收官（U 系落地 v4.97~v4.109；Univer 零引入） |
| gaea-dsh-genui-distill-plan-2026-09.md | ✅ 已收官（P0–P5 全部发布，止于 v4.97.0） |
| gaea-dsh-better-sidebar-long-term-distill-plan-2026.md | 🔄 滚动权威：阶段一/二/二.5/三(3a/3b)已全销账；3c 与阶段四/五维持「拍板后/若做」 |
| gaea-unsloth-modelhub-distill-plan-2026-09.md | ✅ 主体已落地（v4.102~106）；仅真机补验池观察项（CU1 收益/CU3 /free） |

> dsh-better-sidebar-go-port-plan-2026-09.md（整包 Go 化方案）已归档至 docs/archive/——路线未采纳，被滚动蒸馏规划取代。

## 进度计划域（16 项差距已全清收官；观察池=真机项）

| 文档 | 状态 |
|---|---|
| gaea-schedule-gap-vs-project-2026-09.md | **收官总账**：16 项立账→销项版本对照（v4.132~139） |
| gaea-schedule-diff-confirm-design-2026-09.md | ✅ 已收官（四刀 v4.146~149） |
| gaea-schedule-dual-duration-design-2026-09.md | ✅ 已收官（四刀 v4.150~153 + 搭接放开 v4.155） |
| gaea-schedule-projectlibre-distill-2026-09.md | 机制供给参考（蒸馏成果已全部落地） |
| gaea-schedule-standards-digest-2026-09.md | JGJ/T 121-2015 规范权威底稿（缺陷清单已修，v4.129/130） |

> 已实施设计（多工程/资源成本/AOA 手动布局）与 2026-09 市场调研合成版已移入 docs/archive/。

## 已收官历史设计（留原位：代码注释仍引用为设计出处）

gaea-space-shell-design.md（S2.1，v3.9.0）· gaea-space-dimension-design.md（v3.8.0）· gaea-space-assembly-design.md（v3.8.0）· gaea-memory-isolation-design.md（S1.2，v3.8.0）· gaea-page-migration-design.md（P1，v3.9.0；挂账项以 roadmap 为权威）· gaea-edit-tools-design.md（五工具在产，S0.6 起）· gaea-genui-memoryfence-audit-2026-09.md（审计存档；6/7 项 v4.101 收口，resume 槽位口径=唯一开放项）

## 政策 / 约定 / 数据集

ADULT_MODE.md · DREAM_WRITE_POLICY.md · MEMORY_ARCHITECTURE.md · evaluation-set.md · retrieval-eval-set.md（12 条查询集，代码运行时直接解析）· ilink-non-text-protocol.md · 2026-08-14-sandbox-environment-notes.md · 2026-08-15-gaea3-architecture-design.md（历史基准）
