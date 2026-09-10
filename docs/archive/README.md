# docs/archive · 归档说明

> 归档时间：2026（v3.7.0 之后，长期规划定稿时）；**最近一次复核：2026-09-10**（顶层 53 项逐条登记，此前表格只用通配描述、实际漏登 14 项）。
> 原则：**历史调研、已落地计划、被后续结论纠正/取代的文档一律归档**，防止后续会话把过时结论当"当前事实"读取。归档不删除——git 历史与本部目录均可查。

## 权威文档（后续会话请以这些为准）

**完整索引与状态行见 `docs/README.md`**（2026-09-10 重建，45 份顶层文档全覆盖）。本节只列最高频的四类：

- `docs/gaea-nextgen-roadmap-2026.md` —— 全域路线图（唯一权威）。
- `docs/gaea-slim-masterplan-2026-09.md` + `docs/gaea-convergence-plan-2026-09.md` —— 当前活跃的两层规划（瘦身/收敛）。
- `.gaea/AGENTS.md`（长期记忆：纪律 + 工装 + 版本速览）与 `.gaea/progress.md`（最近发布速览 + 开放任务）。
- 政策与评测：`docs/ADULT_MODE.md` / `DREAM_WRITE_POLICY.md` / `MEMORY_ARCHITECTURE.md` / `evaluation-set.md` / `retrieval-eval-set.md`；环境与架构基准：`docs/2026-08-14-sandbox-environment-notes.md` / `docs/2026-08-15-gaea3-architecture-design.md`。

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

### 执行审计与规划（结论已被后续取代，但仍是承接依据）

| 文件 | 归档原因 |
|---|---|
| `audit-2026-08-30-v4-execution-review.md` | v4.x「承诺 vs 代码」对照（裁决=最小版执行）——**已被 AGENTS「长期规划」段摘要并持续引用，原路径写作 `docs/`，实际在本目录**（引用时用 `docs/archive/…`） |
| `gaea-u4-render-evidence-inventory-2026-09.md` | U4 渲染证据清单（U 系蒸馏已收官 v4.97~v4.109） |
| `gaea-v41-evidence-chain-design.md` / `gaea-v415-auto-route-design.md` / `gaea-v42-cost-ai-design.md` / `gaea-v43-play-deepen-design.md` / `gaea-v482-realtime-s2-design.md` | v4.1~v4.8 期专题设计（证据链/自动路由/成本 AI/乐园深化/Realtime S2），均已落地或被现行设计取代 |
| `better-sidebar-port-2026-09-03-worktree.md` / `dsh-better-sidebar-go-port-plan-2026-09.md` | 侧边栏整包 Go 化方案与 worktree 记录——**路线未采纳**，被滚动蒸馏规划取代 |
| `2026-08-27-dsh-context-go-port.md` | dsh-context 移植记录（结论已消化） |
| `2026-08-office-cloud-local-architecture-review.md` / `2026-08-office-knowledge-stack-optimization-summary.md` / `2026-08-16-office-file-interaction-research.md` | 2026-08 办公架构/知识栈/文件交互调研，结论已随版本落地 |
| `2026-08-19-cost-db-reorganize-report.md` / `2026-08-19-cost-library-redesign-zaojia.md` | 造价库重组与改版报告（现行为见 `docs/gaea-cost-domain-survey-2026-09.md`） |
| `gaea-schedule-multi-project-design-2026-09.md` / `gaea-schedule-resource-cost-design-2026-09.md` / `gaea-schedule-aoa-manual-layout-design-2026-09.md` | 进度计划三份已实施设计（v4.112~v4.143 交付）；代码注释仍可能引用为设计出处 |

### 3.0 时代与更早

| 文件 | 归档原因 |
|---|---|
| `2026-08-15-gaea3-vision-roadmap.md` / `2026-08-15-gaea3-ui-constellation-os.md` / `2026-08-15-gaea3-ui-design-system.md` | 3.0 愿景与 UI 设计语言——已被 v4 双空间重设计与长期规划 §10 取代 |
| `2026-08-09-voxcpm2-integration.md` / `2026-08-09-cosyvoice2-llm-gguf-speed-optimization.md` | 本地 TTS 历程（VoxCPM2 已于 v2.6.9 移除）；**当前 TTS 结论见 `.gaea/AGENTS.md`「本地 TTS 引擎」段** |
| `2026-08-12-herdsman-models-evaluation-report.md` / `2026-08-13-herdsman-tts-ocr-comparison.md` | herdsman 模型测评与 TTS/OCR 对比（时点结论，现以 herdsman 实际探测为准） |

### 2026-09 市场调研（合成版 + 原始稿目录见上表）

`market-research-2026-08-31.md`（模块制复扫合成，v4.11.0 基线）· `market-research-2026-09-01.md` / `-01b` / `-02` / `-02b` / `-03c` / `-03d` / `-05` / `-06`（各线合成版）——结论均已进入 docs/ 权威文档并随版本落地。

## 归档纪律

1. **归档不删除**：文件移入本目录并在本 README 登记一行；不写「等」「若干」这类无法核对的量词。
2. **权威指向现行**：本目录任何结论被现行文档纠正后，登记行必须注明取代者（否则后续会话会把旧结论当现状）。
3. **磁带类大件单独说明覆盖面**（含缺口区间），避免「以为全都有」。
