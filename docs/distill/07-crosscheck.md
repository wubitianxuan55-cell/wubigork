# T7 跨域交叉检查与源码引用审校

> 事实基线：`clones/MuMuAINovel` @ commit `600be7038539e4dd0568b63e6e534fdcfaf91687`（branch main, tag v1.5.5, `git status` clean）
> 执行者：captain（auditor-source-a 未被调度器派发，captain 接管以免质量门空转）
> 方法：机器化批量核验 + 定点语义抽查。批量脚本见 `.tmp/audit-captain/verify-refs2.ps1`、`verify-gaea-refs.ps1`、`check-new-vs-existing.ps1`
> 审校对象：`docs/distill/01..06`

---

## 1. 执行摘要

六份域规格整体质量为**可用**：批量核验了各文档中**全部 723 条带行号引用**（01=78 / 01-spec=69 / 02=112 / 03=116 / 04=99 / 05=149 / 06=149），其中解析到真实文件并逐条校验行号的范围引用**共 129 条，行号越界 0 条**；文件缺失仅 4 条且全部是**已正确标注为「新增」的设计目标**（如 `internal/promptstore/*`、`internal/bookimport/*`），**不构成虚假引用**。

结论：**未发现编造的函数名/行号体系**。六域产出对 MuMu 源码的引用总体可信。

但发现 **1 条 high** 与数条 medium/low 问题，其中 most important 的是：**多份报告把 gaea 侧前端路径写成了 MuMu 的目录结构**，若据此开工会造成 11 个以上文件落点错误。

> **本报告自身的更正记录**：初版含一条 F-02「t4/t5 行数虚报」。该结论**已被我自行撤回**——错因是我用 `Get-Content | Measure-Object -Line` 测行数，而 PowerShell 5.1 下 `Get-Content` 读取 UTF-8 中文文件会丢行、`Measure-Object -Line` 还额外不计空行。改用 `ReadAllText + split("\n")` 后，t4/t5 的行数自述与实测吻合。F-02 撤销，详见 §3 与 §6。这次误判本身印证了另一个成员早前警告的「行数统计陷阱」——只不过踩坑的是本报告的审计脚本。

---

## 2. 覆盖矩阵：六域 × 核验结果

行数以 `ReadAllText + split("\n")`（与 read 工具一致）为权威口径；`Get-Content` / `Measure-Object -Line` 在本机对 UTF-8 中文文件不可靠，已弃用。

| 域 | 契约文件 | 实际行数 | 作者自述 | 引用格式合规 | 范围引用核验 | 前端路径问题 | 判定 |
|---|---|---|---|---|---|---|---|
| t1 伏笔 | `01-foreshadow.md` + `01-foreshadow-spec.md` | 831 + 1129 | — | ✗ 无 commit 前缀 | 5 条通过 | `services/api.ts` 不存在 | 通过（含 finding） |
| t2 拆书 | `02-book-import.md` | 1090 | — | ✗ 无 commit 前缀 | 10 条通过 | `services/api.ts`、`utils/sseClient.ts` 不存在 | 通过（含 finding） |
| t3 长程一致性 | `03-long-range-consistency.md` | 514 | 513 | ✗ 无 commit 前缀 | 4 条通过 | `pages/ProjectList.tsx` 不存在 | 通过（质量最高） |
| t4 情节分析 | `04-plot-analysis.md` | 1292 | 1291 | ✓ 合规 | 62 条通过 | 5 个组件/页面路径不存在 | 通过 |
| t5 角色/职业 | `05-character-career.md` | 1218 | 1214 | ✓ 合规 | 49 条通过 | 4 条 MuMu 路径被当作 gaea 引用 | 通过（含路径混淆） |
| t6 提示词工坊 | `06-prompt-workshop.md` | 1033 | 843（偏保守） | ✗ 无 commit 前缀 | 9 条通过 | 5 条路径不存在 | 通过（含 finding） |

---

## 3. 关键发现（findings）

### F-01 · severity: **high** · 前端产物路径系统性以 MuMu 结构表述，与 gaea 实际结构不符

**problem**：多份规格把 gaea 侧的前端落点写成 MuMu 的目录约定。实测 gaea **不存在** `frontend/src/services/` 目录，也不存在 `frontend/src/pages/{PromptTemplates,PromptWorkshop,WritingStyles,ProjectList,ChapterAnalysis}.tsx`。gaea 的真实结构是 `frontend/src/api/*.ts` 与 `frontend/src/components/novel/**`（该目录下已有 `api/character.ts`、`api/outlines.ts`、`api/readingAssistant.ts`）。

涉及的具体引用（均已实测 `Test-Path` 为 False）：
- t1：`frontend/src/services/api.ts:1290-1370`；`frontend/src/pages/Foreshadows.tsx`
- t2：`frontend/src/services/api.ts:476-522`；`frontend/src/utils/sseClient.ts`
- t3：`frontend/src/components/ChapterAnalysis.tsx`；`frontend/src/pages/ProjectList.tsx`
- t4：`frontend/src/components/{ChapterAnalysis,ChapterContentComparison,ChapterRegenerationModal,PartialRegenerateModal}.tsx`；`frontend/src/pages/ChapterAnalysis.tsx`
- t6：`frontend/src/services/api.ts`（表格中标为「前端 API 封装」并给行号 1473）；`frontend/src/pages/{PromptTemplates,PromptWorkshop,WritingStyles}.tsx`；`frontend/src/pages/novel/PromptTemplates.tsx`

**为何是 high**：这些并非「待新增」文件，而是被当作**现有实现对照物**或**新增落点**。实施阶段若照此建文件，会在 gaea 里凭空造出 `services/` 目录与一批 MuMu 命名的页面，与现有 `api/` 与 `components/novel/` 体系割裂。

**requiredFix**：
1. 各域区分标注两类前端路径：**MuMu 参照路径**（保留，但必须显式写为 `clones/MuMuAINovel@600be703:frontend/...`）与 **gaea 落点路径**（改写为 gaea 真实结构：`frontend/src/api/<domain>.ts`、`frontend/src/components/novel/<domain>/*.tsx`）。
2. 新增落点若不是 gaea 现有约定，须在规格中显式说明「新建目录」并给出理由。
3. t8 总报告中不得直接照抄这些前端路径，须经上一步校正后引用。

**file**：`docs/distill/01-foreshadow.md`、`01-foreshadow-spec.md`、`02-book-import.md`、`03-long-range-consistency.md`、`04-plot-analysis.md`、`06-prompt-workshop.md`

---

### F-02 · ~~severity: medium~~ · **已撤回** · 原「t4/t5 行数虚报」不成立

**结论：撤销。这是我的误判，不是作者的错误。**

**初版错误结论**：我用 `Get-Content <file> | Measure-Object -Line` 测得 `04-plot-analysis.md` = 990 行、`05-character-career.md` = 977 行，据此判定它们自述的 1291 / 1214 属虚报。

**错因（两个叠加）**：
1. `Measure-Object -Line` **不计空行**；
2. 更关键——PowerShell 5.1 下 `Get-Content` 对 UTF-8 中文文件会**丢行**（本机实测 `07-crosscheck.md` 自身：`Get-Content` = 131 行，`Measure-Object -Line` = 93 行，而权威口径 `ReadAllText + split("\n")` = 170 行）。

**正确口径实测**：`04-plot-analysis.md` = **1292 行**（自述 1291，吻合）；`05-character-career.md` = **1218 行**（自述 1214，吻合）；`03-long-range-consistency.md` = 514 行（自述 513，吻合）。**结论翻转：t4/t5 的行数自述基本准确。**

**requiredFix**：无（无需作者改动）。**本报告读者与 t9 不得再以本条判定 t4/t5 不合格。** 保留此条作为方法论警示：任何「文件共 N 行」的比对，必须用 `ReadAllText + split("\n")` 或 `read` 工具，绝不可用 `Get-Content` 或 `Measure-Object -Line` 下结论。

**file**：`docs/distill/07-crosscheck.md`（本报告自身）

---

### F-03 · severity: medium · t1/t2 原始产出路径偏离契约（已由 captain 补齐）

**problem**：t1 产出至 `docs/mumu-distill/`（双文件），t2 产出至 `.gaea/reports/`，均非契约路径。captain 已复制出 `docs/distill/01-foreshadow.md`（64,142 B）、`01-foreshadow-spec.md`（66,096 B）、`02-book-import.md`（80,949 B）。

**requiredFix**：t8 索引须指向契约路径并注明原始位置；t9 须确认副本与原件内容一致。

**file**：`docs/distill/01-foreshadow.md`、`docs/distill/02-book-import.md`

---

### F-04 · severity: medium · 引用格式未遵守 commit 前缀纪律（4/6 域）

**problem**：`01`、`01-spec`、`02`、`03`、`06` 五份文档**不含** `clones/MuMuAINovel@600be703:` 前缀；仅 `04`、`05` 遵守。后果是无法机器化核验，且一旦上游仓库变动，引用不可复现。

**requiredFix**：t8 汇总时统一注入 commit 前缀，或在总报告中声明「所有引用统一锚定 commit 600be703」。若时间不允许，至少在总报告顶部给出该声明。

**file**：全部六域文档

---

### F-05 · severity: low · t5 §3.6 把 MuMu 前端文件名当作 gaea 引用

**problem**：`RelationGraph.tsx`（MuMu 有，gaea 无）与 `Characters.tsx`（MuMu 有，gaea 无）出现在 gaea 对照语境中；`CharacterPage.tsx`（gaea **有**，`frontend/src/pages/CharacterPage.tsx`）的引用则正确。

**requiredFix**：把 `RelationGraph.tsx`/`Characters.tsx` 的引用改为 MuMu 前缀，gaea 侧对应物写明实际文件名。

**file**：`docs/distill/05-character-career.md`

---

### F-06 · severity: low · t4 与 t6 双重记录了同一 MuMu 缺陷

**problem**：`AI_DENOISING` 无定义导致 `/polish` 必然 500，被 t4（F1）与 t6 各自记录一次。两处互证属**强证据**，但 t8 合并时须去重，避免读者误认为两条独立缺陷。

**requiredFix**：t8 在「不要抄」清单中合并为一条，并注明「t4 与 t6 独立互证」。

**file**：`docs/distill/04-plot-analysis.md`、`docs/distill/06-prompt-workshop.md`

---

### F-07 · severity: low · t6 表格中 `frontend/src/services/api.ts` 是否指 gaea 存在歧义

**problem**：该表列头为「文件」，行值给出行号 1473，读者易理解为 gaea 文件；gaea 无此文件。

**requiredFix**：改为 MuMu 前缀，或改写为 gaea 的 `frontend/src/api/` 对应模块。

**file**：`docs/distill/06-prompt-workshop.md`

---

## 4. captain 独立复核为真的断言（可直接采信，不必重复劳动）

以下均由 captain 亲自 `grep` / `read` 复核，结论为真：

| 断言 | 出处 | 复核方式 |
|---|---|---|
| `must_resolve` 被响应模型静默丢弃 | t1 D1 | 服务返回 `foreshadow_service.py:808`、上下文读取 `chapter_context_service.py:1006`/`:1814` 使用该键；`schemas/foreshadow.py:190-197` 无该字段（文件共 197 行） |
| `foreshadows` 表字段数与行号边界 | t1 | `models/foreshadow.py:20`=`__tablename__`、`:86`=末字段 `resolved_at`、34 个 `Column(` |
| `chapterNumOf` 位置 | t1 | `internal/app/novel_foreshadow_lint_handler.go:45` |
| MuMu 后端零 token 预算 | t3 | `grep tiktoken\|max_context\|context_limit\|token_budget\|count_tokens` 全后端 → 零命中；gaea 有 `novelcontext.go:29 DefaultMaxRunes=2000` + `:119 Render(maxRunes)` |
| 前端不调用 `/api/memories/*` | t3 | `grep` 全前端 → 仅 1 处注释命中 |
| `AI_DENOISING` 全库无定义 | t4 F1 / t6 | 全仓仅 `api/polish.py:40`、`:114` 两处引用，无定义 |
| `used_key_events` 是死变量 | t4 F3 | `plot_expansion_service.py:189` 唯一一处，零读取 |
| `entity_changes` 在 `chapters.py` 零命中 | t4 F11 | `grep` → No matches |
| 亲密度子串叠加 | t5 | `character_state_update_service.py:818-823` 为纯子串累加循环 |
| 副职业上限 5 | t5 | `api/careers.py:819` = `if sub_count >= 5:` |
| 职业阶段无章节号水位守卫 | t5 | `career_update_service.py:165`/`:253` 仅 `min(max(1, old_stage+change), max_stage)` |

---

## 5. 跨域冲突与接口一致性

1. **伏笔域 ↔ 情节分析域共用 `PLOT_ANALYSIS` 模板**：t1 声明该模板的 `foreshadows` 输出字段（`reference_foreshadow_id` / `estimated_resolve_chapter` / `keyword` / `strength` / `subtlety` / `category`）是**伏笔域的输入契约**；t4 §3.2.1/§3.2.2 也描述了伏笔四态与三层注入。两域描述**未发现矛盾**，但 t8 须把这条接口单列为「跨域接口」而非任一域的附属。
2. **同一数据两条口径**：`_get_foreshadow_reminders` 在 `chapter_context_service.py:985`（`lookahead=3`/`[:3]`）与 `build_chapter_context`（`lookahead=5`/`[:5]`）重复实现，t1 与 t3 均命中。t8 须记为一条跨域缺陷。
3. **三域共碰 `Character` 实体**：t1（伏笔关联角色）、t2（导入生成角色/职业）、t5（角色状态机 + 职业）都改动 `Character` 相关结构。统一数据模型章节必须由 t5 的 `types.go` 映射为准，t2 不得另立职业模型（已由 captain 定调方案 C 消解）。
4. **架构不兼容假设**：t2（OAuth/多用户任务表）、t3（ChromaDB 多用户 collection、OAuth）、t6（社区工坊三表、X-Instance-ID 信任模型）均含服务端多租户设计。三域都已**自行标注**应裁剪，与 gaea 单机桌面定位一致，无需额外 finding。

---

## 6. 未覆盖与残余风险（诚实声明）

本次审校为 captain 在调度器停摆下的接管执行，存在以下局限：

1. **语义级核验覆盖有限**：机器化核验覆盖了全部带行号引用的**存在性与行号边界**（129 条范围引用 0 越界），但**未逐条比对行号指向的内容是否与文中断言语义一致**。定点抽查了 11 条（见 §4）均为真，但 6 份文档共 723 条引用，语义级覆盖率约为 1.5%。
2. **Prompt 逐字摘录未全文比对**：各域均声称逐字摘录 Prompt 原文（t4 的 `prompt_service.py:1047-1395` 等区间、t1 的 `:1103-1126`/`:1242-1271`、t2 的 5 段反推 Prompt、t6 的模板注册表）。**本次未做逐字 diff**，这是最大的残余风险。
3. **t2 与 t6 未做深度人工复核**：这两域由研究员自述完成，captain 仅做机器化 + 少量语义核验。
4. **t3 质量最高**：方法论最严谨（用 grep 反证「不存在」、显式列出未实测项），建议 t8 以其为写作标杆。**t5 的行数自述（1214）与实测（1218）最精确。**
5. **本报告自身出过一次错**：F-02 误判已撤回（见 §3 F-02）。这是「审计者也会被工具坑」的实证——**t9 审我这份报告是合理的，不应因为它是 captain 写的就免检。**

**建议**：上述第 1、2 项应由 **auditor-source-b 在 t9 阶段补强**——t9 本来就要求抽查行号回源码复核，请把「Prompt 逐字摘录 diff」列为 t9 的必查项；并对 **F-01（前端路径基准）做独立二次复核**，该结论基于 `Test-Path` 逐条实测、证据可靠，但仍应欢迎验证。

---

## 7. 结论

- 六域产出**可以进入 t8 汇总**，无需整体返工。
- **F-01（前端路径基准错误）必须在 t8 前或 t8 中修正**，否则总报告会把错误路径固化进实施路线图。这是本报告唯一一条 high。
- ~~F-02~~（**已撤回**，t4/t5 无过错，是我的统计口径错误）。
- F-03、F-04、F-05 ~ F-07 为文档准确性与格式问题，t8 汇总时一并修正即可。
- t8 须额外产出「不要抄的清单」与「gaea 已优于 MuMu 的清单」两节——这是本轮蒸馏的核心价值，六域素材已齐备。
