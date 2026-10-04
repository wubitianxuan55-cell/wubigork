# Go 审计 · 小说业务域

审计范围：`internal/` 下 47 个小说业务域包 + 仓库根 `shared/`（`shared/` 仅 TS，无 Go 文件）。
只读审计，未修改任何仓库文件（唯一写入：本报告）。
工具链：Go 1.26.6；证据来自全仓 token 级扫描（`C:\AI\wubigrok\.tmp\audit-sym\`，临时产物，不在仓库内）。

**方法学（重要）**：死代码判定不能只靠"符号名在别处没出现"——本仓注释里大量出现符号名，且短名（`New`/`Set`/`Sheet`/`Path`）会误命中。本次采用：
1. 用 AST 风格正则抽取 48 个范围包全部 **1776 个顶层声明**（func 834 / method 393 / type 411 / var·const 138）；
2. 对所有 1825 个仓库 `.go` 文件（排除 `clones/ node_modules/ .tmp* backups/ releases/ dist/`）做 **token 级精确扫描**（按标识符边界，前缀桶索引），得到 14930 条 `文件→符号` 二元组；
3. 对每个声明再逐行判定 **本文件内是否被引用**（排除声明行与 `//` 注释行），从而区分"真死"与"仅包内使用"。

---

## 范围与文件统计（每个包 KB / LOC / 函数数 / 真死 LOC）

非测试文件 147 个 / 测试文件 139 个；非测试 1255.4 KB、测试 951.1 KB；非测试总行数 37578；顶层函数 1229。
"真死"列 = 全仓（含本文件、含测试）**零引用**的顶层声明及其行数。

| 包 | 文件 | 测试 | KB | 测试KB | LOC | 函数 | 真死N | 真死LOC |
|---|---|---|---|---|---|---|---|---|
| schedule | 14 | 17 | 149.9 | 182.8 | 4447 | 123 | 0 | 0 |
| types | 13 | 6 | 98.8 | 46.6 | 2247 | 34 | 4 | 0 |
| office | 11 | 10 | 98.7 | 51.7 | 3382 | 89 | 0 | 0 |
| booksource | 12 | 3 | 75.9 | 66.0 | 2420 | 70 | 0 | 0 |
| docmd | 8 | 16 | 73.4 | 67.9 | 2354 | 80 | 0 | 0 |
| config | 9 | 4 | 68.5 | 36.8 | 1777 | 44 | 5 | 21 |
| novelstyle | 8 | 5 | 58.5 | 32.4 | 1857 | 72 | 0 | 0 |
| project | 7 | 6 | 53.8 | 25.4 | 1566 | 91 | 0 | 0 |
| characterlib | 5 | 4 | 49.1 | 37.9 | 1457 | 56 | 3 | 28 |
| novelreview | 4 | 2 | 42.8 | 17.7 | 1234 | 46 | 0 | 0 |
| novelcontext | 1 | 4 | 34.8 | 22.3 | 1037 | 38 | 0 | 0 |
| bookimport | 3 | 4 | 32.5 | 25.1 | 1010 | 36 | 0 | 0 |
| herdsman | 3 | 3 | 26.4 | 30.3 | 747 | 23 | 0 | 0 |
| promptstore | 2 | 2 | 25.5 | 30.4 | 614 | 18 | 0 | 0 |
| character | 1 | 2 | 24.8 | 11.6 | 835 | 30 | 1 | 3 |
| graph | 3 | 2 | 24.5 | 13.5 | 797 | 28 | 3 | 19 |
| export | 3 | 2 | 23.9 | 24.8 | 759 | 21 | 0 | 0 |
| outline | 1 | 3 | 22.4 | 19.2 | 676 | 24 | 0 | 0 |
| realtime | 3 | 3 | 20.3 | 23.5 | 561 | 23 | 0 | 0 |
| intent | 2 | 4 | 15.4 | 12.0 | 365 | 6 | 0 | 0 |
| rewrite | 2 | 2 | 14.9 | 14.7 | 410 | 10 | 0 | 0 |
| chat | 2 | 2 | 14.6 | 8.8 | 497 | 21 | 0 | 0 |
| novelgate | 2 | 3 | 14.1 | 14.4 | 360 | 12 | 0 | 0 |
| characterstate | 1 | 1 | 14.0 | 8.4 | 386 | 9 | 0 | 0 |
| httpbridge | 2 | 2 | 13.8 | 9.9 | 442 | 23 | 0 | 0 |
| prompt | 1 | 3 | 11.5 | 18.1 | 342 | 13 | 0 | 0 |
| narrative | 2 | 1 | 11.4 | 8.1 | 357 | 13 | 0 | 0 |
| visual | 1 | 1 | 11.1 | 1.0 | 388 | 8 | 0 | 0 |
| stats | 2 | 1 | 10.5 | 5.3 | 362 | 6 | 0 | 0 |
| chapter | 1 | 0 | 8.6 | 0.0 | 221 | 9 | 0 | 0 |
| assistant | 1 | 2 | 8.4 | 7.8 | 281 | 11 | 0 | 0 |
| maturecraft | 1 | 1 | 8.2 | 3.6 | 146 | 7 | 0 | 0 |
| worldview | 1 | 1 | 8.1 | 1.2 | 251 | 12 | 0 | 0 |
| screen | 2 | 1 | 8.1 | 4.0 | 271 | 13 | 1 | 0 |
| skilldistill | 1 | 1 | 8.0 | 10.6 | 274 | 13 | 0 | 0 |
| memory | 1 | 2 | 8.0 | 9.6 | 299 | 10 | 0 | 0 |
| ocr | 1 | 2 | 7.8 | 4.1 | 264 | 8 | 2 | 37 |
| style | 1 | 2 | 7.4 | 5.5 | 217 | 9 | 1 | 3 |
| snapshot | 1 | 1 | 7.1 | 3.3 | 294 | 12 | 0 | 0 |
| taskinbox | 1 | 1 | 7.0 | 12.1 | 180 | 9 | 0 | 0 |
| scene | 1 | 1 | 6.8 | 3.3 | 278 | 16 | 1 | 7 |
| routesuggest | 1 | 1 | 6.4 | 6.3 | 183 | 6 | 0 | 0 |
| context | 1 | 1 | 6.1 | 2.4 | 231 | 10 | 1 | 0 |
| skill | 1 | 2 | 4.4 | 4.7 | 190 | 7 | 0 | 0 |
| search | 1 | 0 | 3.7 | 0.0 | 151 | 5 | 0 | 0 |
| noveltext | 1 | 1 | 3.0 | 3.3 | 79 | 2 | 0 | 0 |
| skillstats | 1 | 1 | 2.5 | 2.7 | 82 | 3 | 0 | 0 |
| **合计** | **147** | **139** | **1255.4** | **951.1** | **37578** | **1229** | **19** | **102** |

`shared/`：`shared/package.json` + `shared/types/index.ts`，**无 Go 文件**，不适用。

---

## 整包死代码清单

**无。** 48 个范围内包（含 7 个子包 `office/{crosslink,docxedit,ooxml,pptxedit,standard,xlsxedit,xlsxpreview}`）全部被外部 import，逐个确认：

| 包 | 外部 import 方（非本包） |
|---|---|
| bookimport | app×8, booksource |
| booksource | app×7 |
| chapter | app(app.go, writing_state.go, scene_illustration_v2_test.go) |
| character | app×18, analysis |
| characterlib | app×19, character(无) |
| characterstate | character, analysis |
| chat | app×11 |
| config | 110 个文件（app/ai/auth/...） |
| context | app/context_handler.go |
| docmd | app×7, gaea/{wssearch,fileindex,tool/builtin,control,largefile,knowledgeimport} |
| export | app/{handler_search_export,platform_handler}.go |
| graph | app×2, novelcontext |
| herdsman | app×4 |
| httpbridge | main.go, app×7 |
| intent | app×4 |
| maturecraft | app×11, character, outline |
| narrative | app/novel_state_handler.go |
| novelcontext | app×3 |
| novelgate | app×7 |
| novelreview | app/novel_review_handler.go, novelgate(granularity_test) |
| novelstyle | app×13 |
| noveltext | novelreview×2, novelgate |
| ocr | app/gaea_ocr.go |
| office(+7 子包) | app×12；子包 importers = 1~5 |
| outline | app×3, writing_state |
| project | 97 个文件 |
| prompt | 38 个文件 |
| promptstore | app×4 |
| realtime | voice×2, app×2 |
| rewrite | app×2 |
| routesuggest | app×2 |
| scene | app×2, project×2 |
| schedule | app×4, gaea/control, gaea/tool/builtin |
| screen | gaea/tool/builtin/screenshot.go, app/intent_router.go |
| search | app/bindings_core.go, handler_search_export.go |
| skill | app×5 |
| skilldistill | app/gaea_skill_distill.go |
| skillstats | app×2 |
| snapshot | project/project_stores.go, app(读侧守卫测试) |
| stats | app×2 |
| style | novelcontext, app/platform_handler.go |
| taskinbox | app×3 |
| types | 145 个文件 |
| visual | app/visual_handler.go |
| worldview | app×2, writing_state |
| assistant | characterlib, app×9 |
| memory | app/context_handler.go |

`internal/booksource/rules/`（仅 `rule-template.json`、`websearch-engines.json`）与 `internal/schedule/testdata/` 为数据目录，非包。

**结论：本域不需要"整包删除"，收益全部来自符号级清理。**

---

## Top 20 发现（按 可删行数 × 安全度 排序）

> **置信度口径**
> - `high`：全仓 token 扫描 + grep 双重确认零引用（或结构性证据），可安全删/改。
> - `medium`：零引用已确认，但存在"经 json tag / 接口 / 反射使用"的可能，删前看一眼调用面。
> - `low`：字段/常量级，可能只靠序列化或配置存活，**必须人工确认**。

1. **[P1]** `internal/schedule/cpm.go:148` `computeCpmFull`（+ `ComputeCpmPlan`/`ComputeCpmCal`/`ComputeCpmPlanCal`/`forwardBound`/`backwardBound`/`topoOrder`/`effDur`/`freeFloatPart`/`joinNames`） — D1/D6 — LOC≈256 — confidence:high
   - 证据：`rg -u -F computeCpmFull` 仅 `cpm.go` 命中，且只被同文件 `ComputeCpm` 委托；`ComputeCpmPlan`/`ComputeCpmCal`/`ComputeCpmPlanCal` 三个"双入口"包装在全仓零调用（token 扫描确认）。任务描述按 12 个 op 注册表拆分后，`computeCpmFull` 仍为单函数 256 行、`ComputeCpm` 62 处引用全部走快路径。
   - 建议：保留 `computeCpmFull`（它是唯一实现，`ComputeCpm` 需要它），但**删除 3 个零调用的 `ComputeCpmPlan/Cal/PlanCal` 包装**（≈17 行）并把 `computeCpmFull` 拆为「拓扑 → 正推 → 逆推 → 时差」4 段（每段 ≤60 行）。**不要**删 `computeCpmFull` 本体。

2. **[P1]** `internal/office/docxedit/docxedit.go:470` `rebuildParagraph` 与 `internal/office/pptxedit/pptxedit.go:380` `rebuildParagraph` — D4 — LOC≈250 — confidence:high
   - 证据：逐 6 行滑窗在全仓比对，`docxedit` 与 `pptxedit` 有 **11 个 6 行块完全相同**（`groupByRun := map[[2]int]*runGroup{}`、`for _, seg := range p.segs`、`key := [2]int{seg.runTagStart, seg.runTagEnd}`、`var out bytes.Buffer`、`pos := p.start`、`insInserted := false`、`runRunes := []rune(runText)`、`var delFrom, delTo int`、`var affected []*runSpan`、`cursor := 0` …）。函数长度 140 / 124 行。
   - 建议：把 run 分段与变更区间计算抽到 `internal/office/ooxml`（该包已被两者 import）作为 `BuildRunGroups`/`ApplyReplacement`；两处 `rebuildParagraph` 只保留各自 XML 方言（w: 与 a:）差异。可省 ≈120 行，且消灭"改一处忘一处"风险。

3. **[P1]** `internal/office/xlsxedit/xlsxedit.go:330` `applyOne` — D6 — LOC≈253 — confidence:high
   - 证据：单函数 253 行；同仓 `internal/schedule/ops.go:110` 已有 `var opHandlers = map[string]func(...)` 注册表范式的成功先例（注释自述"原 applyOne 430 行巨型 switch 按 op 拆为独立函数"）。
   - 建议：照 `schedule/ops.go` 范式拆为 `apply<Op>` 12 个函数 + `opHandlers` 表（该目录已有 700+ 行文件，拆后 `applyOne` ≤15 行）。

4. **[P1]** `internal/config/config.go:27` `func Load() *Config` — D6/D2 — LOC≈449 — confidence:high
   - 证据：braces 精确测量 449 行（27–475），为全仓最长单函数；文件 `config.go` 477 行中 94% 是这一个函数。内部含多段"if 未设置 → 填默认"重复骨架。
   - 建议：按域拆为 `loadAI/loadVoice/loadNovel/loadUI/loadPaths` 小函数，`Load()` 只做装配与 `ApplyNovelsDirWalkthrough(cfg)` 收口（≈30 行）。**这是本域最高收益的结构性改动**，但需覆盖 `config_test.go` 全量回归。

5. **[P1]** `internal/types/interfaces.go`（全文 240 行） — D5 — LOC≈120 — confidence:medium
   - 证据：`TemplateParser`/`TemplateResolver`/`TemplateValidator`/`TemplateEngine`/`TemplateRenderer`/`TemplateRenderResult`/`TemplateSource`/`TemplateSourceRank`/`TemplateWarning`/`TemplateConstraintsDef`/`InputSectionDef`/`OutputFormatDef`/`SectionPriority`/`ContextSection`/`SectionSortKey`/`ContextProvider`/`ContextBuilder`/`ContextAssembler`/`LintCode`/`LintSeverity`/`ForeshadowLinter`/`ChapterNumOf` 等 **22 个导出类型/函数**全仓零引用（`otherFiles==0`）。`SectionPriority` 6 次、`LintCode` 11 次出现**全在 interfaces.go 自身**。
   - 证据补充：`rg -u -F 'TemplateParser' --glob '*.go' .` 仅命中本文件；`SortContextSections` 仅被 `internal/types/compat_test.go` 引用（测试钉一个没人用的导出函数）。
   - 建议：整段删除"模板引擎/上下文装配"抽象层（疑似早期设计残留，实际由 `internal/prompt` + `internal/novelcontext` 承担）。**删前需确认**没有外部消费者（本域外已确认零）。

6. **[P1]** `internal/schedule/ops.go:110` `opHandlers` 注册表中的 12 个 `applyXxx` — D1（局部） — LOC≈60 — confidence:high
   - 证据：`applyUpsertTask`/`applyPatchTask`/`applyRemoveTask`/`applySetLinks`/`applySetMeta`/`applyAutoChain`/`applySetBaseline`/`applyClearBaseline`/`applyUpsertResource`/`applyPatchResource`/`applyRemoveResource`/`applySetAssignments` 全部仅出现在「注册表字面量 1 行 + 声明」两处——**唯一入口是 `opHandlers` 表**，因此它们本身不是死代码；但**没有测试直接调用**（`ops_dispatch_test.go` 只校验键集）。
   - 建议：不算死代码，但**12 个函数共 ≈560 行完全无调用方覆盖**；加 `TestApplyOpsEachKind` 表驱动（每 op 一条最小 Project）即可把 `ops.go` 591 行的回改风险锁住。**这是"最该补测"而不是"最该删"**。

7. **[P1]** `internal/docmd/pdf.go:666-720` 一组 `is*` 判定残留 — D1/D2 — LOC≈60 — confidence:high
   - 证据：`isPDFKeyword`(652)、`isPDFNoiseLine`(675)、`isXrefEntry`(692)、`isDigitsOrHex`(707)、`isTextStreamBody`(644)、`isPDFNameChar`(173)、`isDigitsOrHex`、`hasControlByte`(393)、`hasReplacementRune`(384)、`toUint16BE`(376)、`findDictStart`(560)、`findEndstream`(583)、`flateDecompress`(601)、`alphaWordCount`(748)、`meaningfulText`(727) —— `rg -u -F <name> --glob '*.go' .` 每个都只命中 `pdf.go` 自身定义行，token 扫描确认 `otherFiles==0` 且本文件内除声明外无引用。
   - 建议：逐个确认后在 `pdf.go` 内删除（pdf.go 640 行 → ≈500 行）；若为"两遍解析"预留则加 `// 保留：X 路径预留` 注释说明，否则下次审计仍会命中。

8. **[P1]** `internal/office/xlsxpreview/condfmt.go:48-119` `cfRuleXML`/`condFmtXML`/`worksheetCfXML`/`cfFontXML`/`cfPatternFillXML`/`cfFillXML`/`cfDxfXML`/`cfDxfsXML`/`cfStyleSheetXML`/`cfWorkbookSheet`/`cfWorkbookXML`/`cfRelXML`/`cfRelsXML` — D1 — LOC≈75 — confidence:medium
   - 证据：13 个类型中 `cfRuleXML` 只被同文件 `condFmtXML` 的字段引用，而 `condFmtXML` 自身零引用；`rg -u -F cfRuleXML internal/office/xlsxpreview/` 仅 2 行（定义 + 字段声明）。
   - 建议：整组删除（条件格式 XML 序列化模型未被使用）。**中置信**原因：可能用于 `xml.Marshal` 动态路径，删前确认 `condfmt.go` 的导出函数是否真的不产生这些结构。

9. **[P1]** `internal/httpbridge/bridge.go` 多个内部句柄零调用 — D1 — LOC≈60 — confidence:high
   - 证据：`handleRPC`(159)、`subscribe`(285)、`unsubscribe`(296)、`tokenOK`(118)、`newEventHub`(281)、`handleStream`(323)、`cors`(259)、`writeRPC`(253)、`hostAllowlist`(17)、`requestHost`(33) 全部 `otherFiles==0`；其中 `subscribe`/`unsubscribe` 是 `eventHub` 的方法，`eventHub` 类型本身也零外部引用（`bridge.go:274`）。
   - 建议：`bridge.go` 332 行 + `hostguard.go`（另 40 行）疑似整体过时（`httpbridge` 仍被 main.go 使用，但只是 `ServeWithToken` 一层）。**先确认 `main.go` 实际起的是哪条路径**，再把未用的 RPC/stream/eventHub 分支删掉。

10. **[P1]** `internal/skill/skill.go:108-190` `parseSkillFile` — D3/D6 — LOC≈81 — confidence:high
    - 证据：81 行；本文件内 line 126 出现**重复注释 `// 找到第二个 ---`（连续两行完全一致，D3 遗留）**；`scan()` 调用它以填充 `l.skills`。
    - 建议：删掉重复注释（1 行）；`parseSkillFile` 拆出 `parseFrontmatter` + `parseAppliesTo` 两段（各 ≈25 行）。

11. **[P1]** `internal/novelcontext/novelcontext.go` 822 行单文件 38 函数 — D6 — LOC≈822（拆分目标 ≤400） — confidence:high
    - 证据：本域最大非测试文件（34.8 KB / 822 行）；`Render(maxRunes)` 单函数 106 行；同文件内 `buildSetting`/`buildStyle`/`buildThread`/`buildForeshadows`/`buildTimeAnchor`/`buildSceneChars`/`buildPOVMask`/`synthesizeChapterScene`/`formatScene` 共 9 个 builder 全部只在本文件被调用（`otherFiles==0`）。
    - 建议：按 `scene_bible.go`（结构 + Render）/`builders.go`（build* 系列）/`pov.go`（POV 掩码与可见性）/`fallback.go`（synthesize* + infer*）四文件切分，纯搬运不改逻辑。

12. **[P1]** `internal/character/character.go:146-250` `GeneratePortrait` 105 行 — D6 — LOC≈105（拆分） — confidence:high
    - 证据：函数跨度 146–250；唯一调用方 `internal/app/character_handler.go`。
    - 建议：拆 `buildNovelPortraitPrompt`（已存在，23 行）+ 远端调用 + 落盘三段。

13. **[P2]** `internal/characterstate/characterstate.go:75` `ApplyChapterDiff` 136 行 — D6 — LOC≈136 — confidence:high
    - 证据：75–210；同文件内 `applyCareerInc`/`applyOrgMemberChange`/`cascadeSurvival`/`clamp100`/`findBidirectionalRel`/`appendRelNote`（合计 ≈167 行）都是它的内联子步骤且零外部引用。
    - 建议：正是天然的拆分点——`ApplyChapterDiff` 只保留 diff 遍历，子步骤已是独立函数，只需把 switch 改为显式调用序列。

14. **[P2]** `internal/novelstyle/score.go` 10 个 `ruleXxx` 各自独立但零外部引用 — D5 — LOC≈60 — confidence:medium
    - 证据：`ruleUnmotivatedMetaphor`(136)/`ruleRegisterBreak`(154)/`ruleEmotionDirect`(178)/`ruleConnectiveDense`(194)/`ruleFourCharConsecutive`(218)/`ruleSentenceUniform`(238)/`ruleAdjAdvDensity`(264)/`rulePunctuation`(282)/`ruleAIBlacklist`(325) 全部 `otherFiles==0`，只在 `score.go` 内被 `ScoreText`/`ScoreTextNoRef` 调用（包内使用，非死代码）。
    - 建议：不是死代码，但 9 个同签名函数应改为 `var rules = []ruleFn{...}` 表（`schedule/ops.go` 同范式），并把权重从散落的魔数提到表内。可省 ≈15 行、显著提升可测性。

15. **[P2]** `internal/project/story_spine_store.go:51` `ValidateStorySpine` 96 行 — D1 — LOC≈96 — confidence:medium
    - 证据：`rg -u -F ValidateStorySpine --glob '*.go' .` **仅 1 个文件**（定义文件自身），token 扫描 `otherFiles==0`；同文件 `StorySpinePath`(16)/`WriteStorySpine`(39) 亦零外部引用。
    - 建议：`story_spine_store.go` 疑似半成品（只有写与校验，无读入口）。**要么接入调用面，要么整文件删（≈147 行）**——请产品/主控确认剧情脊柱是否在路线图上。

16. **[P2]** `internal/project/project.go:137` `LoadContext` 53 行 + `internal/ocr/herdsman.go:84-93` 双 `Recognize*` — D1 — LOC≈67 — confidence:high
    - 证据：`LoadContext` 全仓仅定义行；`RecognizeImageBytes`/`RecognizeImageBase64` 全仓仅 `herdsman.go` 自身（80/89 行互相调用，无外部入口）。
    - 建议：删除（`ocr.Client` 的外部入口只有 `RecognizeFile`？请与调用方确认后一并清理 `internal/ocr` 未用导出）。

17. **[P2]** `internal/style/profile.go:204` `Profile.ExportProfile` — D1/重复实现 — LOC≈24（含 `buildStyleGuide` 37 行） — confidence:high
    - 证据：`rg -u -F ExportProfile --glob '*.go' .` 仅 `profile.go:203-204`（注释 + 定义）。同时 `buildStyleGuide`(176) 亦零引用。
    - 建议：`internal/style` 与 `internal/novelstyle` 存在职责重叠（见"重复实现"节）。若确认 `style` 只服务 `internal/app/platform_handler.go` 一处，可整体合并进 `novelstyle`。

18. **[P2]** `internal/config/config_prefs.go` 一批只在文件内使用的 setter — D1 — LOC≈15 — confidence:low
    - 证据：`SetReadScreenSummary`(43)/`SetReadScreenKeepLast`(57)/`SetIntentsLLMTimeoutMS`(115) `otherFiles==0` 且本文件内除声明外无引用（token + grep 双确认）。但 `config/` 是 **序列化重灾区**：`Get*` 系列虽"零外部引用"却可能是前端字符串调用。
    - 建议：**先查 `frontend/src` 是否按方法名字符串调用**（本次已 grep 全部候选方法名，`frontend/src` 命中 0），再删。标注 low 是因为 `config` 字段/方法是 TOML/JSON 面，删 getter 可能让前端某个未在当前版本启用的面板失联。

19. **[P2]** `internal/intent/intent.go` 6 个正则 var 零引用 — D1 — LOC≈6 — confidence:low
    - 证据：`reNavStrong`(68)/`reStatus`(80)/`reScreenOrdinal`(127)/`reScreenPrimary`(128)/`reScreenSecondary`(129)/`boardAliases`(40) 在 token 扫描中 `otherFiles==0`；但 `rg -n` 明确显示它们在本文件被 `MatchString`/`range` 使用（例如 `reNavStrong` 用于 248 行、`boardAliases` 用于 57 行）。
    - 建议：**这是误报，不要删。** 列入此表是为了记录"token 扫描在短名/同文件场景下会给出 `otherFiles==0`，必须再看本文件使用"的方法学结论。

20. **[P2]** `internal/config/walkthrough.go:24,50` `IsWalkthrough` / `ApplyNovelsDirWalkthrough` — D2（可见性/接线） — LOC≈0（不建议删） — confidence:high
    - 证据：`rg -u -F ApplyNovelsDirWalkthrough .` 命中 `walkthrough.go` 定义 + `config.go:472` 调用点（`Load()` 末步）；`IsWalkthrough` 仅 `walkthrough.go` 自身 3 处使用。二者均为**有效代码**。
    - 建议：**不要动。** 这是 v4.441.1 的走查沙箱隔离（第三数据面），删掉会让沙箱写真实书架。此处记录是为了纠正自动分析的假阳性。

---

## 全量发现表

（每条都经 token 扫描 + `rg` 双确认；"仅包内"= 有引用但只在定义文件/同包内）

| 文件:行 | 符号 | 类型 | LOC | 置信度 | 证据 | 建议 |
|---|---|---|---|---|---|---|
| internal/schedule/cpm.go:148 | computeCpmFull | D6 | 256 | high | `rg -F computeCpmFull` 仅 cpm.go；被 ComputeCpm 委托 | 拆 4 段 |
| internal/schedule/cpm.go:131 | ComputeCpmPlan | D1/D5 | 5 | high | 全仓零引用（token 扫描） | 删 |
| internal/schedule/cpm.go:139 | ComputeCpmCal | D1/D5 | 5 | high | 同上 | 删 |
| internal/schedule/cpm.go:144 | ComputeCpmPlanCal | D1/D5 | 7 | high | 同上 | 删 |
| internal/schedule/cpm.go:418 | CriticalChain | D1 | 24 | high | `rg -F CriticalChain` 仅 cpm.go 注释+定义 | 删或接入叙事用途 |
| internal/schedule/cpm.go:405 | joinNames | D1 | 10 | high | 仅本文件 | 删 |
| internal/schedule/index.go:237 | adoptIndex | D1 | 48 | high | otherFiles==0，本文件仅自身 | 删 |
| internal/schedule/mpp.go:79 | mppVersionOfFormat | D1 | 11 | high | 同上 | 删 |
| internal/schedule/mpp.go:450 | intInSlice | D1 | 8 | high | 同上 | 删 |
| internal/schedule/xlsx.go:648 | sheetNameOr | D1 | 7 | high | 同上 | 删 |
| internal/schedule/cost.go:47 | assignmentCost | D1 | 24 | high | 同上 | 删 |
| internal/office/xlsxpreview/xlsxpreview.go:153 | renderSheet | 仅包内 | 118 | high | 同文件 96 行调用；无外部入口 | 拆 |
| internal/office/xlsxpreview/xlsxpreview.go:77/87/272/297 | renderGuarded/renderXlsx/cellTypeName/hasStyle | 仅包内 | 42 | high | 仅本文件 | 并入拆分段 |
| internal/office/xlsxpreview/xlsxpreview.go:40 | Freeze | D1 | 6 | medium | otherFiles==0（DTO，可能仅序列化） | 确认后删 |
| internal/office/xlsxpreview/xlsxpreview.go:46/55 | Cell/CellStyle | 仅包内 | 10 | medium | 同文件使用 | 保留 |
| internal/office/xlsxedit/xlsxedit.go:330 | applyOne | D6 | 253 | high | 精确测量；ops.go 已有注册表先例 | 拆注册表 |
| internal/office/xlsxedit/xlsxedit.go:586 | mergeStyle | 仅包内 | 65 | high | 仅 applyOne 调用 | 保留（拆后可测） |
| internal/office/xlsxedit/xlsxedit.go:199/250/98 | diffOps/opCells/trimCells | 仅包内 | 113 | high | 仅本文件 | 保留 |
| internal/office/xlsxedit/xlsxedit.go:683 | cellRefRe | D1 | 2 | low | otherFiles==0，同文件仅声明 | 人工确认 |
| internal/office/xlsxedit/xlsxedit.go:753 | findRecalcScript | 仅包内 | 18 | high | 仅 Recalc 调用路径 | 保留 |
| internal/office/docxedit/docxedit.go:470 | rebuildParagraph | D4/D6 | 140 | high | 与 pptxedit 11 个 6 行块相同 | 抽公共实现 |
| internal/office/docxedit/docxedit.go:353 | parseParagraphs | 仅包内 | 107 | high | 仅本文件 | 与 pptx 合并 |
| internal/office/docxedit/docxedit.go:308-338 | changeAuthorRe/changeAuthor/unwrapInner/renameDelText/spanCoversLink | D1 | 40 | high | otherFiles==0；`spanCoversLink` 等仅声明 | 删未用正则+helper |
| internal/office/docxedit/docxedit.go:22/615 | wmlNS/widRe | D1 | 4 | low | 同文件零引用 | 人工确认 |
| internal/office/pptxedit/pptxedit.go:380 | rebuildParagraph | D4/D6 | 124 | high | 同 docxedit | 抽公共实现 |
| internal/office/pptxedit/pptxedit.go:263 | parseSlideParagraphs | 仅包内 | 110 | high | 仅本文件 | 保留（拆） |
| internal/office/pptxedit/pptxedit.go:139 | relsNS | D1 | 2 | low | 同文件零引用 | 人工确认 |
| internal/office/xlsxpreview/condfmt.go:48-119 | cfRuleXML 等 13 个 XML 模型 | D1 | 75 | medium | 仅本文件互相引用，无外部消费者 | 整组删 |
| internal/office/standard/registry.go:13/25/40/74 | Checker/NewRegistry/Names/registrySummary | D1 | 44 | medium | otherFiles==0（接口 + 实现，可能经接口调用） | 确认后删 |
| internal/office/standard/redhead.go:92/102/122 | hasSalutation/fixHint/redHeadSummary | D1 | 34 | high | `rg -F` 仅本文件 | 删 |
| internal/office/standard/costtable.go:48 | costFixHint | D1 | 17 | high | 仅本文件 | 删 |
| internal/office/crosslink/crosslink.go:180 | dirOf | D1 | 7 | high | 仅本文件 | 删 |
| internal/config/config.go:27 | Load | D6 | 449 | high | 精确测量 27–475 | 按域拆 |
| internal/config/config.go:518 | ResolveResourceDirForTest | D1 | 3 | high | `rg -F` 仅 config.go:3 注释 + 517 注释 + 518 定义 | 删 |
| internal/config/config_realtime.go:7/14/22 | GetRealtimeProvider/GetRealtimeModel/GetRealtimeAPIKey | 仅包内 | 18 | low | token 扫描 otherFiles==0，但 `internal/app/voice_handler.go:147-152` 实际调用 | **误报，勿删** |
| internal/config/config_features.go:7-84 | Get/SetFeatureModel(Enabled) | 仅包内 | 105 | low | `internal/analysis/analysis.go:314`、`app/bindings_core.go:39-43` 调用 | **误报，勿删** |
| internal/config/config_prefs.go:43/57/115 | SetReadScreenSummary/SetReadScreenKeepLast/SetIntentsLLMTimeoutMS | D1 | 15 | low | otherFiles==0 且本文件零使用 | grep frontend 后删 |
| internal/config/config_prefs.go:36/50/79 | GetReadScreenSummary/GetReadScreenKeepLast/GetIntentsLLMTimeoutMS | 仅包内 | 15 | low | 同文件被 setter 对称使用 | 与 setter 同进同出 |
| internal/config/config.go:479/536 | resolveResourceDir/dirExists | 仅包内 | 43 | high | 仅 config.go 内 | 保留 |
| internal/config/walkthrough.go:24/50 | IsWalkthrough/ApplyNovelsDirWalkthrough | D2 | 0 | high | config.go:472 调用点 | **不要动（安全隔离）** |
| internal/docmd/pdf.go:652/675/692/707 | isPDFKeyword/isPDFNoiseLine/isXrefEntry/isDigitsOrHex | D1 | 57 | high | `rg -F` 各仅 1 文件 | 删 |
| internal/docmd/pdf.go:376/384/393/644 | toUint16BE/hasReplacementRune/hasControlByte/isTextStreamBody | D1 | 32 | high | 同上 | 删 |
| internal/docmd/pdf.go:560/583/601 | findDictStart/findEndstream/flateDecompress | D1 | 27 | high | 同上 | 删或补调用面 |
| internal/docmd/pdf.go:727/748 | meaningfulText/alphaWordCount | D1 | 27 | high | 同上 | 删 |
| internal/docmd/ocr.go:360/409 | ExtractImage（两个接收者） | D1 | 22 | high | otherFiles==0 | 删 |
| internal/docmd/office.go:190/249/279/302/316 | extractDocxTable/extractCellText/findWt/isSelfClosing/extractAttr | D1 | 83 | high | otherFiles==0，本文件零调用 | 删 |
| internal/docmd/markitdown.go:18/70/81 | markItDownTimeout/markItDownCmd/markItDownError | D1 | 12 | high | 同上 | 删 |
| internal/docmd/render.go:22 | RenderPDFPages | 仅包内 | 5 | medium | 包内使用 | 保留 |
| internal/project/story_spine_store.go:16/39/51 | StorySpinePath/WriteStorySpine/ValidateStorySpine | D1 | 102 | medium | 三者 otherFiles==0；`ValidateStorySpine` `rg` 仅 1 文件 | 接入或整文件删 |
| internal/project/project.go:137 | LoadContext | D1 | 53 | high | `rg -u -F` 全仓仅定义行 | 删 |
| internal/project/project_stores.go:106/171/274 | ChapterMemoriesPath/RewriteChapterDir/ChapterAnnotationsPath | D1 | 9 | high | otherFiles==0 且本文件零调用 | 删 |
| internal/project/project_chapters.go:69 | ChapterSummaryPath | D1 | 3 | high | 同上 | 删 |
| internal/project/project_migrate.go:158 | finalizeV4Migration | 仅包内 | 8 | high | 仅 MigrateV3ToV4 调用 | 保留 |
| internal/project/project_chapters.go:130/236/273 | ReadAllChapterSummaries/maxChapterNumMatching/chapterHasBodyContent | 仅包内 | 19 | high | 同包内使用 | 保留 |
| internal/types/interfaces.go:28-348 | InputSectionDef/OutputFormatDef/TemplateConstraintsDef/TemplateSource/TemplateSourceRank/TemplateParser/TemplateResolver/TemplateValidator/TemplateWarning/TemplateEngine/TemplateRenderer/SectionPriority/ContextSection/SectionSortKey/ContextProvider/ContextBuilder/ContextAssembler/LintCode/LintSeverity/ForeshadowLinter/ChapterNumOf | D1/D5 | 240 | medium | 22 个导出全零外部引用；`rg -F TemplateParser` 仅本文件 | 整段删（确认后） |
| internal/types/types.go:289 | SnapshotChain | D1 | 2 | medium | `rg -u -F SnapshotChain --glob '*.go' .` 仅 types.go（另有序号测试噪声） | 删 |
| internal/types/types.go:204/297 | ForeshadowChange/ContextPriority | D1 | 5 | medium | otherFiles==0 | 删 |
| internal/types/character_state.go:60/75 | CareerDefinition/CareerStage | D1 | 7 | medium | otherFiles==0 | 删 |
| internal/types/foreshadow_v2.go:225 | ForeshadowAmbiguity | D1 | 3 | medium | otherFiles==0 | 删 |
| internal/types/plot_v2.go:25/44 | ChapterPlanEndingType/RewriteMode | D1 | 10 | medium | 出现次数全在本文件 | 确认后删 |
| internal/graph/entity.go:199/204/215 | AddRelation/GetRelations/AllEntityNames | D1 | 19 | high | `rg -u -F GetRelations --glob '*.go'` 仅 entity.go:203 注释 + 204 定义 | 删 |
| internal/graph/entity.go:29 | EntityRelation | D1 | 6 | medium | otherFiles==0（GetRelations 返回类型，随其删） | 随方法同删 |
| internal/graph/linker.go:44 | ParseLinksWithContext | D1 | 4 | high | `rg` 仅 linker.go | 删 |
| internal/graph/linker.go:114 | GetAllEntities | D1 | 7 | medium | 仅 linker.go | 删 |
| internal/graph/consistency.go:149-385 | checkCharacterAttributes/detectAttributeConflicts/extractEvidence/checkCharacterStatus/checkTimeline | D2 | 260 | medium | 全部 otherFiles==0 且本文件零调用（`CheckConsistency` 已不调它们） | 删未接管分支 |
| internal/characterlib/portrait.go:280/288/296 | DataDir/PortraitsDir/PortraitFilePath | D1 | 28 | high | `rg -u -F PortraitsDir\|PortraitFilePath --glob '*.go'` 仅 portrait.go 注释+定义 | 删 |
| internal/characterlib/portrait.go:55/106/123/154 | saveRemoteImage/writePortraitBytes/extFromDataURL/extFromPath | 仅包内 | 72 | high | 同包内使用 | 保留 |
| internal/characterlib/import.go:17/37/61 | descField/normalizeFieldKey/ImportFieldConflict | D1 | 20 | medium | otherFiles==0（field 可能仅反射） | 确认后删 |
| internal/characterlib/db.go:20/113 | GetDatabase/CloseDatabase | 仅包内 | 40 | high | 仅本包 | 保留 |
| internal/character/character.go:146 | GeneratePortrait | D6 | 105 | high | 唯一调用方 character_handler.go | 拆三段 |
| internal/character/character.go:528 | BatchGenerate | D1 | 3 | high | `rg -F BatchGenerate --glob '*.go'` 仅 character.go:526 注释 + 528 定义 | 删（注释自述"兼容旧接口"） |
| internal/character/character.go:254/281/667/695 | buildNovelPortraitPrompt/savePortraitToProject/extractCharacterUpdates/mergeCharacters | 仅包内 | 136 | high | 同文件内调用 | 保留 |
| internal/character/character.go:739 | SetCharacterCareerRequest | D1 | 4 | medium | otherFiles==0（请求 DTO） | 删 |
| internal/graph — | — | — | — | — | 见上 | — |
| internal/scene/scene.go:118 | UpdateMeta | D1 | 7 | high | `rg -u -F UpdateMeta --glob '*.go'` 仅 scene.go:117 注释 + 118 定义 | 删 |
| internal/scene/scene.go:28/241 | ensureDir/readMeta | 仅包内 | 12 | high | 同文件使用 | 保留 |
| internal/style/profile.go:204/176 | ExportProfile/buildStyleGuide | D1 | 27 | high | `rg -u -F ExportProfile --glob '*.go'` 仅 profile.go 2 行 | 删（或并入 novelstyle） |
| internal/style/profile.go:41/49 | Analyzer/NewAnalyzer | 仅包内 | 5 | medium | 仅本包 | 保留 |
| internal/stats/dashboard.go:234 | computeAchievements | 仅包内 | 44 | high | 仅 GatherDashboard 调用 | 保留 |
| internal/stats/dashboard.go:19/50/57 | DashboardData/ChapterWordCount/Achievement | D1 | 15 | medium | otherFiles==0（DTO，经 json 出前端） | **勿删（序列化面）** |
| internal/novelreview/rubric.go:21 | RubricMeta | D1 | 3 | medium | otherFiles==0 | 确认后删 |
| internal/novelreview/rubric.go:80/94/114 | dimensionIDs/newRubricSnapshot/validateRubric | 仅包内 | 15 | high | 仅本文件 | 保留 |
| internal/novelreview/review.go:180-386 | dimLength/dimOpeningHook/openingSignals/dimEndingHook/dimTrailer/dimEmotionDensity/dimPowerDensity | 仅包内 | 271 | high | 全部经 review.go 内 dimension 表注册调用 | 保留（改规则表） |
| internal/novelreview/review.go:162/170 | severityOf/labelOf | 仅包内 | 12 | high | 同上 | 保留 |
| internal/novelreview/util.go:21/131 | countNonSpaceRunes/countRune | 仅包内 | 18 | high | 同上 | 保留 |
| internal/novelgate/plan_gate.go:33 | chapterNum（参数） | D7 | 1 | high | 代码内自述 `_ = chapterNum // 保留章号入参` | **需人工确认**签名对齐后删参数 |
| internal/novelgate/plan_gate.go:106/130/138/148/159/174 | duplicateEventProblems/duplicateEventMessage/containsNormalizedEvent/meaningfulEvents/countMeaningfulEvents/normalizePlanEvent | 仅包内 | 72 | high | 仅 PlanContractIssues 调用 | 保留 |
| internal/booksource/rule.go:361 | LoadFile | 仅包内 | 7 | high | 被同文件 LoadDir 调用 | 保留 |
| internal/booksource/rule.go:392 | rejectUnsupportedFields | 仅包内 | 20 | high | 仅本文件 | 保留 |
| internal/booksource/fetch.go:26 | systemSleeper | 仅包内 | 13 | high | 仅本文件 | 保留 |
| internal/booksource/fetch.go:142 | cfTitles | D1 | 2 | low | otherFiles==0 且本文件零调用 | 人工确认 |
| internal/booksource/guess.go:66 | tocIndexFromURL | D1 | 3 | high | otherFiles==0 | 删 |
| internal/booksource/toc.go:109 | maxTocGuessPages | D1 | 3 | low | 同上 | 人工确认 |
| internal/booksource/clean.go:28 | removeFilterTags | 仅包内 | 3 | high | 仅本文件 | 保留 |
| internal/booksource/clean.go:41 | voidElements | D1 | 3 | low | 同文件零引用 | 人工确认 |
| internal/booksource/search.go:154 | searchLink | D1 | 3 | high | otherFiles==0 且本文件零调用 | 删 |
| internal/booksource/search.go:272 | SourceError | D1 | 6 | medium | otherFiles==0（error 类型） | 确认后删 |
| internal/booksource/websearch.go:24 | indexHtmlTail | D1 | 5 | low | 同文件零引用 | 人工确认 |
| internal/booksource/chapter.go:97/167 | nextChapterPage/fetchChapters | 仅包内 | 76 | high | DownloadChapters 链内 | 保留（DownloadChapters 本身零外部引用 → 确认后整链删） |
| internal/booksource/chapter.go:129-159 | DownloadOptions/DownloadReport/DownloadChapters | D1 | 30 | high | 三者均 otherFiles==0 | 确认后整链删 |
| internal/booksource/assemble.go:72 | firstNonBlank | 仅包内 | 8 | high | 同文件使用 | 保留 |
| internal/booksource/cache.go:10 | ttlItem | 仅包内 | 4 | high | TTLCache 内部 | 保留 |
| internal/bookimport/reconstruct.go:215/241/386 | JSONHint/ExpectedKind/characterListField | D1 | 31 | high | otherFiles==0 且本文件零调用 | 删 |
| internal/bookimport/reconstruct.go:31 | OutlineCharacter | D1 | 7 | medium | otherFiles==0 | 确认后删 |
| internal/bookimport/skeleton.go:73 | Segment | D1 | 5 | medium | otherFiles==0 | 确认后删 |
| internal/bookimport/skeleton.go:61/83 | SegmentSize/AggregateSkeleton | 仅包内 | 22 | high | 被 `internal/app/novel_import_ai.go` 间接使用链内 | 保留 |
| internal/bookimport/parse.go:152/245/251/272/313/386 | decodeUTF16BOM/isStrongHeading/isWeakHeading/assembleChapters/windowSplit/buildWarnings | 仅包内 | 145 | high | 仅 Parse 调用链 | 保留 |
| internal/bookimport/reconstruct.go:330/360/414/425 | strField/stringListField/clampTargetWords/firstSentence | 仅包内 | 62 | high | 仅本文件 | 保留 |
| internal/bookimport/reconstruct.go:76 | fallbackScenes | D1 | 2 | low | 同文件零引用 | 人工确认 |
| internal/ocr/herdsman.go:84/93 | RecognizeImageBytes/RecognizeImageBase64 | D1 | 37 | high | `rg -u -F` 仅 herdsman.go 80/83/84/89/92/93（互相调用，无外部入口） | 删 |
| internal/ocr/herdsman.go:137/143 | ParsePage/ParseResult | D1 | 5 | medium | otherFiles==0 | 确认后删 |
| internal/ocr/herdsman.go:242 | normalizeBaseURL | 仅包内 | 8 | high | 仅本文件 | 保留 |
| internal/ocr/herdsman.go:125 | ParseOptions | 仅包内 | 3 | high | ParseDocument 入参 | 保留 |
| internal/herdsman/probe.go:151/192/212/238/250/271/299 | probeConfig/probeCLI/probeDataFiles/checkDir/checkJSON/checkJSONL/probeAPI | 仅包内 | 163 | high | 全部由 Probe 编排 | 保留 |
| internal/herdsman/probe.go:73/85 | CLIStatus/FileStatus | D1 | 11 | medium | otherFiles==0（DTO，经 json 出前端） | **勿删（序列化面）** |
| internal/herdsman/health.go:141/147/176/235 | modelUsable/isKnownCapability/pingModelsAPI/buildSummary | 仅包内 | 43 | high | 仅 HealthCheck 链 | 保留 |
| internal/herdsman/lancheck.go:134/148/160/168 | lanSplitKeyValue/lanParseBool/lanStripComment/lanBuildGuidance | 仅包内 | 33 | high | 仅 CheckLanExposure 链 | 保留 |
| internal/httpbridge/bridge.go:118/159/253/259/281/285/296/323 | tokenOK/handleRPC/writeRPC/cors/newEventHub/subscribe/unsubscribe/handleStream | D1 | 137 | medium | 全部 otherFiles==0；`eventHub` 类型(274)亦零外部引用 | 确认 main.go 实际路径后删整组 |
| internal/httpbridge/hostguard.go:17/33 | hostAllowlist/requestHost | D1 | 24 | medium | 同上 | 同组删 |
| internal/prompt/prompt.go:96/274 | loadEmbedded/loadTemplate | 仅包内 | 30 | high | 仅本文件 Engine 装配 | 保留 |
| internal/prompt/prompt.go:245 | buildSection | 仅包内 | 28 | high | 仅本文件 | 保留 |
| internal/promptstore/bundle.go:46 | BundleExportStats | D1 | 3 | medium | otherFiles==0 | 确认后删 |
| internal/promptstore/bundle.go:242/253/264 | upsertImported/hasErrorIssue/summarizeIssues | 仅包内 | 25 | high | 仅 ImportBundle 链 | 保留 |
| internal/promptstore/promptstore.go:164 | maskDoubleBrace | 仅包内 | 21 | high | 仅 Validate 链 | 保留 |
| internal/promptstore/promptstore.go:110/119 | fieldText/templateTextFields | 仅包内 | 25 | high | 仅本文件 | 保留 |
| internal/promptstore/promptstore.go:66 | legacyPlaceholderRe | D1 | 3 | low | 同文件零引用 | 人工确认 |
| internal/memory/memory.go:283/292 | formatMemory/formatScore | D1 | 16 | high | otherFiles==0 且本文件零调用 | 删 |
| internal/memory/memory.go:131/151 | summaryChapterNums/leadingNumber | 仅包内 | 25 | high | 仅本文件 | 保留 |
| internal/memory/memory.go:258 | InjectIntoContext | 仅包内 | 24 | high | 唯一调用方 `app/context_handler.go` | 保留 |
| internal/context/engine.go:227 | formatLorebookEntry | 仅包内 | 5 | high | 仅 BuildFullContext 链 | 保留 |
| internal/context/engine.go:28 | BudgetSection | D1 | 4 | medium | otherFiles==0 | 确认后删 |
| internal/screen/screen_windows.go:221/230 | getDC/createCompatibleDC | 仅包内 | 8 | high | 仅 CaptureArea 链 | 保留 |
| internal/screen/screen_windows.go:76/94 | bitmapInfo/monitorInfo | D1 | 6 | medium | otherFiles==0（syscall 结构体，**可能被 syscall 反射使用**） | **勿删** |
| internal/skill/skill.go:110 | parseSkillFile | 仅包内 | 81 | high | 同文件 43 行调用 | 拆 + 删重复注释 |
| internal/skill/skill.go:124/126 | 重复注释 `// 找到第二个 ---` | D3 | 1 | high | 两行完全相同 | 删 1 行 |
| internal/skillstats/skillstats.go:77 | statLess | 仅包内 | 6 | high | 仅排序调用 | 保留 |
| internal/skilldistill/distill.go:137/243 | patternKey/firstAt | 仅包内 | 11 | high | 仅本文件 | 保留 |
| internal/snapshot/snapshot.go:44/49/289 | snapDir/ensureSnapDir/splitLines | 仅包内 | 12 | high | 仅本文件 | 保留 |
| internal/scene/scene.go:180 | Stitch | 仅包内 | 5 | medium | 包内使用 | 保留 |
| internal/routesuggest/suggest.go:127/165 | eligible/buildSuggestion | 仅包内 | 22 | high | 仅 Suggest 链 | 保留 |
| internal/routesuggest/suggest.go:19/22/69 | ScoreGapThreshold/MinSamples/costEpsilon | D1 | 6 | low | 同文件零引用 | 人工确认 |
| internal/rewrite/engine.go:146 | pickSuggestions | 仅包内 | 12 | high | 仅 Rewrite 链 | 保留 |
| internal/rewrite/partial.go:186 | normalizePartial | 仅包内 | 18 | high | 仅 engine.go 调用 | 保留 |
| internal/realtime/openai.go:158/171/199 | sessionUpdate/sessionUpdateVoice/inputAudioAppend | 仅包内 | 13 | medium | 仅本文件（协议帧结构体） | 保留 |
| internal/realtime/openai.go:374 | isOpen | 仅包内 | 3 | high | 仅本文件 | 保留 |
| internal/realtime/provider.go:87 | SessionFactory | D1 | 4 | medium | otherFiles==0（接口） | 确认后删 |
| internal/narrative/journal.go:17/20 | journalName/stateName | D1 | 4 | low | 同文件零引用 | 人工确认 |
| internal/maturecraft/maturecraft.go:70 | craftHeader | 仅包内 | 3 | high | 同文件 91 行调用 | 保留 |
| internal/chapter/chapter.go:138/147 | ChapterReviewResult/ReviewChapter | 仅包内 | 29 | high | 被 `app/novel_gate_handler.go` 调用 | 保留 |
| internal/chapter/chapter.go:212/218 | novelModelName/novelEngineName | D4 | 8 | high | 与 character/outline 各一份完全同名同实现 | 抽公共 helper |
| internal/character/character.go:716/722 | novelModelName/novelEngineName | D4 | 8 | high | 同上 | 抽公共 helper |
| internal/outline/outline.go:39/45 | novelModelName/novelEngineName | D4 | 8 | high | 同上 | 抽公共 helper |
| internal/analysis/analysis.go:100 | novelModelName 内联重复 | D4 | — | high | 同逻辑第 4 份 | 统一走 helper |
| internal/export/html.go:27 | DefaultTemplates | 仅包内 | 34 | high | 唯一调用方 `app/platform_handler.go` | 保留 |
| internal/export/html.go:261/282/299 | webNovelCSS/printCSS/minimalCSS | D1 | 6 | low | 同文件零引用（**可能是模板字符串**） | 人工确认 |
| internal/export/export.go:323 | escapeHTML | 仅包内 | 7 | high | 仅本文件 | 保留 |
| internal/export/export.go:203 | findCoverImage | 仅包内 | 3 | high | 仅本文件 | 保留 |
| internal/export/docx.go:70 | contentLines | 仅包内 | 9 | high | 同文件 37/45 行调用 | 保留 |
| internal/visual/visual.go:372 | emotionToColor | 仅包内 | 17 | high | 仅本文件 | 保留 |
| internal/visual/visual.go:43/135 | readSummaryForScan/scoreByKeywords | 仅包内 | 14 | high | 仅 ExtractCharacterHeatmap 链 | 保留 |
| internal/visual/visual.go:117/207/284/298/307/18 | EmotionPoint/CharacterHeatmapCell/CanvasCard/CanvasEdge/CanvasData/TimelineEvent | D1 | 39 | medium | otherFiles==0（**前端消费的 DTO**，经 json 序列化） | **勿删** |
| internal/worldview/worldview.go:86/104/113/125/143 | SaveSection/SaveAllSections/GetCurrent/GetSections/ChatWithAutoSave | 仅包内 | 55 | high | 全部由 `app/worldview_handler.go` 调用 | 保留 |
| internal/worldview/worldview.go:49/50 | 重复分节注释 `// ── 保存 ──` | D3 | 1 | high | 相邻两行完全相同 | 删 1 行 |
| internal/outline/outline.go:355 | 重复分节注释 `// ── CRUD ──` | D3 | 1 | high | 相邻两行完全相同 | 删 1 行 |
| internal/chapter/chapter.go:33 | 重复分节注释 `// ── 辅助函数 ──` | D3 | 1 | high | 相邻两行完全相同 | 删 1 行 |
| internal/project/story_spine_store.go:51 | ValidateStorySpine | D6 | 96 | medium | 同上（单函数 96 行无测试） | 接入或删 |
| internal/characterlib/store.go:639 | Store 639 行 | D6 | 639 | high | 本包最大文件，含 `ImportProjectCharacters` 92 行 | 按 store/portrait/import 已分文件，再拆 `store.go` 的 SQL 助手 |
| internal/narrative/journal.go | Journal 未接入 | D2 | — | medium | `internal/narrative` 仅 `app/novel_state_handler.go` 使用，`journal.go` 的 `marshalSnapshot`/`journalPath` 零外部引用 | 确认 Journal 是否在路线图 |
| internal/intent/intent.go:40/68/71/80/127/128/129 | boardAliases/reNavStrong/reNavHome/reStatus/reScreen* | D1（假阳性） | 6 | low | token 扫描 otherFiles==0，但 `rg` 显示本文件 57/248/270 行实际使用 | **误报，勿删** |
| internal/novelstyle/*.go | ruleXxx 系列 9 个 | D5 | 60 | medium | 仅 score.go 内规则循环调用 | 改规则表 |
| internal/office/docxedit/docxedit.go:481 / office/pptxedit/pptxedit.go:390 | groupByRun 块 | D4 | 12 | high | 6 行滑窗比对完全相同（共 11 块） | 抽公共 |
| internal/characterstate/characterstate.go:75 | ApplyChapterDiff | D6 | 136 | high | 精确测量 75–210 | 拆分（子步骤已独立） |
| internal/schedule/project.go:75 | Validate | D6 | 120 | high | 精确测量 | 拆 5 闸 |
| internal/schedule/xlsx.go:430 | ImportXlsx | D6 | 216 | high | 精确测量 | 拆表头匹配/行解析/自定义列 |
| internal/schedule/mpp.go:93/348 | mppParseFromStreams/mppBuildProject | D6 | 206 | high | 精确测量 | 拆 |
| internal/docmd/pdf.go:227 | extractPDFText | D6 | 113 | high | 精确测量 | 拆 |
| internal/novelgate/gate.go:73 | ChapterQualityIssues | D6 | 71 | high | 精确测量 | 拆规则 |
| internal/booksource/rule.go:183 | Rule.Validate | D6 | 153 | high | 精确测量 | 拆 |
| internal/booksource/websearch.go:197 | discover | D6 | 118 | high | 精确测量 | 拆 |
| internal/outline/outline.go:129 | Agent.Continue | D6 | 150 | high | 精确测量 | 拆 |
| internal/export/html.go:63 | ExportHTML | D6 | 107 | high | 精确测量 | 拆 |
| internal/office/crosslink/crosslink.go:24 | ExtractChartData | D6 | 90 | high | 精确测量 | 拆 |
| internal/office/xlsxedit/xlsxedit.go:586 | applyOne 内 mergeStyle | D6 | 65 | high | 精确测量 | 拆 |

---

## 疑似重复实现的包对（含收敛建议与风险）

### 1. `internal/office/docxedit` ↔ `internal/office/pptxedit`（最高价值，风险低）
- **证据**：6 行滑窗全仓比对，两者有 **11 个完全相同的 6 行块**；`rebuildParagraph` 140 vs 124 行、`parseParagraphs` 107 vs `parseSlideParagraphs` 110 行、两侧各有一份 `xmlTokenRange`（同 17 行）与 `textSeg`（同 10 行）。
- **收敛建议**：把「run 分段（groupByRun）→ 变更区间（affected/delFrom/delTo）→ 重建」三段下沉到 `internal/office/ooxml`（**该包已被两者 import**，无新循环依赖）。保留各自的命名空间常量（`wmlNS`/`relsNS`）与标签方言。
- **风险**：中。docx 有 w:del/w:ins 修订语义（`renameDelText`/`parseChangeSpans`），pptx 无；下沉时必须把"是否支持修订"作为参数显式化，否则会悄悄改变 pptx 行为。**先补 pptx 的金样测试再动**。

### 2. `internal/novelstyle` ↔ `internal/style`（职责重叠）
- **证据**：`novelstyle` 45 个声明零外部引用（大量 `ruleXxx`/`countXxx` 只在包内），`style` 仅被 `internal/app/platform_handler.go` 与 `internal/novelcontext` 使用；`style.ExportProfile`/`buildStyleGuide` 零引用。两者都做"风格画像"（`novelstyle.Fingerprint` ↔ `style.Profile`）。
- **收敛建议**：保留 `novelstyle` 作为唯一风格引擎（它服务 13 个 app 文件，且已有 `ScoreTextNoRef`/`DeSlopRewriteEx` 等生产入口）；把 `internal/style` 的 `Profile`/`Analyzer` 并入 `novelstyle` 的 fingerprint 子模块，或降为 `novelstyle` 的适配层。
- **风险**：中高。`style` 的 `Profile` 参与落盘（`project.StyleProfilePath`），合并要同步 `project` 的文件路径与 `style/profile_io_test.go`。

### 3. `internal/prompt` ↔ `internal/promptstore` ↔ `internal/skill`（三者都在"装配提示词"）
- **证据**：`prompt` 提供模板引擎与 `Engine`；`promptstore` 提供 `Validate`/`ImportBundle`/`SameTemplate`（都 import `prompt`）；`skill` 独立实现 `SKILL.md` 解析 + `InjectSkill`，功能上等价于"从磁盘装载模板片段并拼进 system prompt"（`prompt.LoadDir` 亦扫描目录）。
- **收敛建议**：`skill` 改为消费 `promptstore` 的模板模型（`skill.Skill` → `prompt.Template`），`InjectSkill` 由 `prompt.Engine.Render` 承担；`promptstore` 保持"校验/导入/导出"职责。
- **风险**：中。`skill` 的 `applies_to` 语义（按场景过滤）在 `prompt.Template` 里没有对应字段，需要先扩展模板模型，否则会丢功能。

### 4. `internal/outline` ↔ `internal/rewrite` ↔ `internal/novelstyle`
- **证据**：`outline.Agent` 与 `rewrite.Engine` 都做"章节级 LLM 重写"（`outline.ExpandNode`/`Continue` ↔ `rewrite.Rewrite`），且都各自维护 `novelModelName/novelEngineName`；`novelstyle.DeSlopRewriteEx` 亦做重写（去 AI 味）。
- **收敛建议**：抽 `novelLLM` 小包提供 `(model, engine)` 解析与流式调用（现在是 **4 份重复**：`chapter`、`character`、`outline`、`analysis`），三处重写只保留各自 prompt 与后处理。
- **风险**：低（纯 helper 提取）。**这是最该先做的一条重复实现收敛**——4 份 `novelModelName` 实现完全一致（各 3 行 + 3 行注释）。

### 5. `internal/types` ↔ 各业务包内重复定义
- **证据**：`types` 411 个声明，其中 22 个零引用；同时 `novelcontext.SceneBible`/`SceneChar`、`graph.EntityRelation`、`stats.DashboardData`/`ChapterWordCount`/`Achievement`、`visual.EmotionPoint`/`TimelineEvent`/`CanvasCard`/`CanvasEdge`/`CanvasData`/`CharacterHeatmapCell`、`herdsman.CLIStatus`/`FileStatus` 等 **DTO 全部零跨包引用**。
- **收敛建议**：**先不要删**——它们大概率是前端 JSON 契约（详见"不建议动的地方"）。真正该动的是 `types/interfaces.go` 的模板/上下文抽象层（22 个零引用类型，见 Top5）。
- **风险**：低（本域内已确认零消费者）。

---

## 不建议动的地方

| 位置 | 为什么不要动 |
|---|---|
| `internal/config/config_features.go`、`config_realtime.go` 全部 Get/Set | token 扫描显示"零外部引用"，但 `rg -u` 实证 `internal/analysis/analysis.go:314`、`internal/app/bindings_core.go:39-43`、`internal/app/voice_handler.go:147-152` **正在调用**。这是自动分析的假阳性。 |
| `internal/config/walkthrough.go:24,50` `IsWalkthrough`/`ApplyNovelsDirWalkthrough` | 走查沙箱第三数据面隔离（v4.441.1）。`config.Load()` 末步调用；删掉会让沙箱壳写进用户真实书架（`CHANGELOG.md:44` 记录了这次事故与修复）。 |
| `internal/intent/intent.go` 的 `reXxx` 正则与 `boardAliases` | `otherFiles==0` 但同文件内 `MatchString`/`range` 大量使用。**短名 + 同文件使用是 token 扫描的盲区。** |
| `internal/stats/dashboard.go`、`internal/visual/visual.go`、`internal/herdsman/probe.go`、`internal/screen/`、`internal/ocr/herdsman.go` 的 DTO 类型 | 全部带 `json` tag，是**前端契约**（Wails 绑定经 `encoding/json` 出栈）。字段名不出现于 Go 代码 ≠ 无人使用。删了会静默破坏前端面板。 |
| `internal/types/**` 的 DTO 结构体（`SnapshotChain`/`ForeshadowChange`/`CareerDefinition`/`ContextPriority` 等） | 同上；且 `internal/types` 被 145 个文件 import，任何删改都可能影响 `internal/gaea/**`（本次未审计）。 |
| `internal/schedule/ops.go` 的 12 个 `applyXxx` | 它们经 `opHandlers` 表（`ops.go:110`）分派，是活代码；"零测试直接调用"≠ 死代码。**该补测试，不该删。** |
| `internal/booksource/rules/*.json`、`internal/schedule/testdata/*.json` | 数据文件（规则模板/测试金样），`go:embed` 或运行期读取，非代码。 |
| 所有 `_test.go` | 测试面（951.1 KB）不在本次"死代码"口径内；测试引用不计入"外部引用"，但很多导出函数**只被测试使用**——删之前必须看测试（本次已对全部候选排除 `_test.go` 引用）。 |
| `internal/office/ooxml`、`office/xlsxedit` | 被 app 多处 import（xlsxedit 5 处），且 `applyOne` 属结构问题不是死代码。 |

---

## 附：本次审计的方法学坑（供后续审计复用）

1. `rg` 默认遵守 `.gitignore`，而仓库 `.gitignore` 含 `.tmp*`；若在命令里显式写了 `.tmp` 相关路径或用了 `--no-ignore-parent`，扫描面会**静默缩小 3.5 倍**（1825 → 6346 Go 文件里的另一个方向）。**必须显式** `-u` 或 `--no-ignore`。
2. PowerShell 的 `Sort-Object -Unique` **忽略大小写**，用它给符号去重会合并 `Foo`/`foo`，导致正则 alternation 漏匹配整行其他符号（本次一度造成"`applyUpsertTask` 被误判为死代码"）。**必须用 `HashSet[string]` 默认比较器**。
3. 单条巨型正则（1600+ 分支 alternation）会**静默丢匹配**（ripgrep 与 .NET 都观察到）。改用「前缀桶 + 逐 token 查表」可同时获得正确性与速度（1825 文件 / 5 秒）。
4. `otherFiles==0` 只说明"其它文件不出现该符号"，**必须再判本文件内是否被引用**（排除声明行与 `//` 注释行）才能叫死代码。574 → 19 的收敛就来自这一步。
5. 2026-10-02 的 `docs/code-audit-2026-10-02/` 已有一份机器扫描（`machine/scan.json`、`gofunc.json`、`appendix-A-mechanical.md`）。本次结论与其中"god-func"榜单一致（`config.Load` 449 / `computeCpmFull` 256 / `applyOne` 253），可作为交叉验证。
