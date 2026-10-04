# 前端审计 · gaea/components/schedule

> 审计对象：Gaea（Go 1.26 + Wails v2；React 18 + TS + Vite）
> 审计路径：`frontend/src/gaea`、`frontend/src/components`、`frontend/src/schedule`、`frontend/src/api`、`frontend/src/boards`、`frontend/src/layouts`、`frontend/src/stores`、`frontend/src/v3`
> 方式：**只读**。未修改、新建、删除任何仓库文件（本报告除外）。未跑 `git commit`，未起 dev server。
> 工具：自建 AST/词法级脚本（import 图、未用导出、未用 props、跨文件重复块、死 CSS 选择器、不可达语句）+ `eslint --format json`（938 文件）+ 人工核对。
> 路径约定：全部相对仓库根，例如 `frontend/src/gaea/components/TrendChart.tsx`。

---

## 范围与文件统计

| 目录 | 文件数 | ts/tsx | 其中 test | css | LOC |
|---|---:|---:|---:|---:|---:|
| `frontend/src/gaea` | 581 | 571 | 208 | 8 | 106,256 |
| `frontend/src/components` | 241 | 234 | 90 | 7 | 45,343 |
| `frontend/src/schedule` | 113 | 110 | 46 | 1 | 25,652 |
| `frontend/src/api` | 8 | 8 | 2 | 0 | 2,009 |
| `frontend/src/boards` | 8 | 8 | 3 | 0 | 1,249 |
| `frontend/src/layouts` | 2 | 2 | 1 | 0 | 901 |
| `frontend/src/stores` | 5 | 5 | 3 | 0 | 884 |
| `frontend/src/v3` | 1 | 0（仅 `foundation.css`） | 0 | 1 | 639 |
| **合计** | **959** | **938** | **353** | **17** | **182,933** |

**验证基线**（本次实跑，可复现）：
- `pnpm exec tsc -b --noEmit`（工作目录 `frontend`）→ **exit 0，无任何诊断输出**（增量缓存命中，未强制 `--force`）。
- `pnpm exec eslint src/gaea src/components src/schedule src/api src/boards src/layouts src/stores --format json` → **938 文件、仅 12 条 message（0 error）**：
  - `@typescript-eslint/no-unused-vars` × 6（见 D7 表）
  - `react-refresh/only-export-components` × 5（`launcher_parts.tsx:392`、`cost_projects_parts.tsx:14/26`）
  - `react-hooks/exhaustive-deps` × 1（`frontend/src/components/novel/editor/GhostText.tsx:87`）
  > 说明：`src/v3` 只有 `foundation.css`，eslint 会因无可 lint 文件报 "all files ignored"，属预期。
  > **结论：本域 lint 基线极干净，因此本次高价值发现集中在 eslint 覆盖不到的地方**（未用导出、死文件、未用 props、重复实现、死 CSS、巨型组件）。

### 最大的 20 个文件（按 LOC）

| # | 文件 | LOC |
|---:|---|---:|
| 1 | `frontend/src/schedule/ops_golden.fixture.json` | 4,481 |
| 2 | `frontend/src/gaea/locales/zh-TW.ts` | 1,685 |
| 3 | `frontend/src/gaea/locales/en.ts` | 1,683 |
| 4 | `frontend/src/gaea/locales/zh.ts` | 1,681 |
| 5 | `frontend/src/gaea/styles.css` | 1,499 |
| 6 | `frontend/src/components/module-launcher.css` | 1,450 |
| 7 | `frontend/src/schedule/schedule.css` | 1,112 |
| 8 | `frontend/src/gaea/components/Sidebar.tsx` | 1,066 |
| 9 | `frontend/src/gaea/App.tsx` | 1,041 |
| 10 | `frontend/src/gaea/components/Transcript.tsx` | 1,024 |
| 11 | `frontend/src/gaea/components/DocxPreview.tsx` | 1,021 |
| 12 | `frontend/src/gaea/lib/mock/novel.ts` | 870 |
| 13 | `frontend/src/components/characterlib/CharacterLibEditor.tsx` | 869 |
| 14 | `frontend/src/gaea/components/context/inspector.tsx` | 861 |
| 15 | `frontend/src/gaea/lib/spaceBindings.ts` | 835 |
| 16 | `frontend/src/gaea/components/MemoryPanel.tsx` | 830 |
| 17 | `frontend/src/components/characterlib/character-library.css` | 827 |
| 18 | `frontend/src/components/novel/ChapterEditor.tsx` | 822 |
| 19 | `frontend/src/layouts/MainLayout.tsx` | 812 |
| 20 | `frontend/src/gaea/components/XlsxPreview.tsx` | 809 |

> 注：`gaea/App.tsx` 不在本审计的「修改建议」范围（由他人负责），仅作为 import 证据来源引用。

---

## 整文件死代码清单（最高价值）

判定方法：对 `frontend/src` 全部 1,173 个 ts/tsx 建 import 图（含 `import x from`、`import('...')`、`require('...')`、`vi.mock('...')`、`import '...'`），再对每个候选逐个 grep：**文件名（去扩展名）**、**符号名字符串**、**`lazy(() => import(`**、**`import.meta.glob`**。

| 文件 | LOC | 引用情况（硬证据） | 置信度 | 结论 |
|---|---:|---|---|---|
| `frontend/src/gaea/components/SubagentsPanel.tsx` | **400** | 全仓唯一 import 为 `SubagentsPanel.test.tsx:4`。其余 6 处命中**全是注释**：`gaea/lib/agentNetworkStore.ts:5`、`gaea/hooks/useRunningBadge.ts:11`、`gaea/components/MergedPanel.tsx:10`、`gaea/components/AgentTree.tsx:72`、`gaea/components/BrowserPanel.tsx:151`、`gaea/components/TasksWorkbench.tsx:33`。无 `lazy`、无字符串注册、无 glob | **high** | **可删**（其继任者是 `TasksWorkbench.tsx`，后者已是 App 侧任务视图） |
| `frontend/src/gaea/components/SubagentsPanel.test.tsx` | **281** | 只测上面这个死组件 | **high** | 随组件删除 |
| `frontend/src/gaea/components/MorningBriefCard.tsx` | **82** | 全仓唯一 import 为 `MorningBriefCard.test.tsx:3`。`components/ModuleLauncher.tsx:372` 注释原文：「组件与数据管线保留（gaea/components/MorningBriefCard），仅撤书斋侧列入口」→ **入口已被撤掉，组件成孤儿** | **high**（无引用）/ **medium**（是否该删，需产品确认） | **建议删**，但先确认那句注释是否仍是现行决策 |
| `frontend/src/gaea/components/MorningBriefCard.test.tsx` | **75** | 同上 | high | 随组件删 |
| `frontend/src/gaea/components/morning-brief-card.css` | **98** | 仅被 `MorningBriefCard.tsx` import | high | 随组件删 |
| `frontend/src/gaea/components/TrendChart.tsx` | **59** | **全仓 0 文件 import**。grep `TrendChart` 的其余命中全部是**别的符号或散文**：`pages/modelcenter/charts.tsx` 的 `RequestsTrendChart`/`TokenTrendChart`、`gaea/components/ContextView.tsx:183` 的 `ContextTrendChart`、以及 `AgentRadial.tsx:25`/`AgentRadial.test.tsx:13` 的「TrendChart 同款/同思路」注释。无同名 test 文件（`glob **/TrendChart*` 只命中自身）、无 `lazy`、无 glob | **high** | **整文件可删** |

**整文件死代码合计：约 936 行**（SubagentsPanel 400 + 其测试 281 + MorningBriefCard 82 + 测试 75 + CSS 98 = 936；TrendChart 59 → 全量 **995 行**）。
另有 1 个「test-only 但**不可删**」的文件见「不建议动的地方」：`frontend/src/gaea/lib/mock/bindingSignatures.ts`（660 行）。

---

## Top 20 发现（按 可删行数 × 安全度 排序）

1. **[P0]** `frontend/src/gaea/components/SubagentsPanel.tsx:1` `SubagentsPanel` — **D1** — LOC≈400（+测试 281）— confidence:**high**
   - 证据：唯一 import = `SubagentsPanel.test.tsx:4`；`grep -r SubagentsPanel` 在 `app/`、`App.tsx`、`TasksWorkbench.tsx` 中 0 引用，命中的 6 处全为注释（`agentNetworkStore.ts:5`、`useRunningBadge.ts:11`、`MergedPanel.tsx:10`、`AgentTree.tsx:72`、`BrowserPanel.tsx:151`、`TasksWorkbench.tsx:33`）；无 lazy/字符串/glob 引用。
   - 建议：先删 `.tsx` + `.test.tsx`（`git rm` 前只做一次 `grep -rn "SubagentsPanel" frontend/` 复核）。若担心回归，可保留测试文件、把断言的 DOM 结构迁到 `TasksWorkbench.test.tsx`。

2. **[P0]** `frontend/src/gaea/components/MorningBriefCard.tsx:1` `MorningBriefCard`（默认导出）— **D1** — LOC≈82（+测试 75 + css 98）— confidence:**high**
   - 证据：`MorningBriefCard.test.tsx:3` 是唯一 import；`components/ModuleLauncher.tsx:372` 注释明写「仅撤书斋侧列入口」。
   - 建议：产品确认后整组（tsx+test+css）删除；若确认要保留，至少加 `// @dead-entry-removed` 标记并把它加入 knip 的 `ignore`，避免下次审计重复上报。

3. **[P0]** `frontend/src/gaea/components/TrendChart.tsx:1` `TrendChart` / `TrendPoint` — **D1** — LOC≈59 — confidence:**high**
   - 证据：全仓 0 import；无同名测试；`grep -rn "TrendChart"` 只命中同名前缀的**不同**组件（`RequestsTrendChart`/`TokenTrendChart`/`ContextTrendChart`）与 2 条散文注释。另外它导出的 `TrendPoint`（`:5`）与 `frontend/src/api/engines.ts:93` 的 `TrendPoint` 重复定义（D8）。
   - 建议：整文件删；`TrendPoint` 统一收口到 `api/engines.ts`。

4. **[P1]** `frontend/src/gaea/components/memoryhub/CostImportModal.tsx` ⇄ `frontend/src/gaea/components/memoryhub/KnowledgeImportModal.tsx` — **D4** — LOC≈86 重复 — confidence:**high**
   - 证据：跨文件重复块 8 段：`CostImportModal.tsx:24-28 / 38-41 / 53-59 / 69-72 / 156-170 / 179-181 / 242-252 / 348-353` ⇔ `KnowledgeImportModal.tsx:15-19 / 29-32 / 43-49 / 59-62 / 130-144 / 153-155 / 199-209 / 265-270`，逐行字节级相同（含按钮 className 长串 `inline-flex items-center gap-1 px-3 h-8 rounded-lg border border-border …`）。
   - 建议：抽 `<ImportModalShell title parseColumns runAI … />`，两个模态只保留列定义与保存回调。

5. **[P1]** `frontend/src/gaea/lib/agentNetworkStore.ts` ⇄ `frontend/src/gaea/lib/subagentRunsStore.ts` — **D4** — LOC≈120 — confidence:**medium-high**
   - 证据：两文件各自定义 `Subscription`（`:40`/`:39`）、`Poller`（`:48`/`:46`）、`syncTicker`，且 `.catch/.finally` 尾部逐行相同：`agentNetworkStore.ts:87-89 / 101-103` ⇔ `subagentRunsStore.ts:79-81 / 93-95`。
   - 建议：抽 `createPoller({ fetch, keyOf })` 泛型工厂（两个 store 的差异只有 `GaeaAgentNetwork()` 无参 / `GaeaSubagentRuns(path)` 有参）。

6. **[P1]** `frontend/src/gaea/components/FilePreview.tsx` ⇄ `frontend/src/gaea/components/FilePreviewModal.tsx` — **D4** — LOC≈30 — confidence:**high**
   - 证据：① `formatSize` 两处 5 行**完全相同**（`FilePreview.tsx:43-47` ⇔ `FilePreviewModal.tsx:24-28`）；② pptx 逐页 `<figure>/<img>` + 懒加载占位 25 行重复（`FilePreview.tsx:476-493` ⇔ `FilePreviewModal.tsx:273-290`）。
   - 建议：抽 `components/preview/` 共享 `PptxPages` 与 `formatSize`。

7. **[P1]** 拖拽改宽手势 4 份 — `frontend/src/gaea/components/ResizableDrawer.tsx:128-132`、`frontend/src/gaea/hooks/useSidebar.ts:46-50`、`frontend/src/gaea/app/useWorkspaceLayout.ts:80-83`、`frontend/src/gaea/app/usePreviewPanel.ts:174-177` — **D4** — LOC≈60 — confidence:**high**
   - 证据：4 组重复块（6 对两两相同），内容为 `saveXxxWidth/setResizing(false)/removeEventListener(pointermove|pointerup|pointercancel)/document.body.style.cursor=""` 同一序列。
   - 建议：抽 `usePointerResize({ save, setResizing })`。

8. **[P1]** 字节格式化函数 5 份 — `frontend/src/gaea/components/FilePreview.tsx:43`、`frontend/src/gaea/components/FilePreviewModal.tsx:24`（两份字节级相同）、`frontend/src/gaea/components/composer/ComposerAttachmentBar.tsx:28`、`frontend/src/components/imagegen/ModelDirectory.tsx:85`、`frontend/src/components/settings/DataPanel.tsx:33` — **D4** — LOC≈25 — confidence:**high**
   - 证据：见上（`formatSize`/`formatBytes`/`fmtSize` 三名一物；`DataPanel` 多一档 GB，`ModelDirectory` 只输出 GB）。
   - 建议：收口到既有工具层（`frontend/src/utils`，非本审计范围）暴露 `formatBytes(n, {maxUnit})`。

9. **[P1]** eslint 实锤的 6 处未使用声明 — **D7** — LOC≈8 — confidence:**high**
   - `frontend/src/components/ModuleLauncher.tsx:29` `UserOutlined`（unused import）
   - `frontend/src/components/novel/create/BranchWizardModal.tsx:41` `prevChapter`（死 prop，调用方仍在传）
   - `frontend/src/components/novel/editor/GhostText.test.tsx:8` `waitFor`（unused import）
   - `frontend/src/gaea/components/memoryhub/CostProjectsView.tsx:12` `EntryPicker` + `slug`（两个 unused import）
   - `frontend/src/gaea/components/memoryhub/cost_projects_parts.tsx:7` `message`（unused import）
   - 证据：`eslint --format json` 原始 message；`cost_projects_parts.tsx:5` 的文件头注释「EntryPicker（自带 debounce）…八件全部为主段消费」已成**陈旧注释**。
   - 建议：直接删这 6 处并同步修注释。

10. **[P1]** `frontend/src/gaea/styles.css` 死选择器 53 个规则块 / **323 行**（占该文件 1,499 行的 21.5%）— **D1(CSS)** — confidence:**high**
    - 证据：对 `gaea/styles.css` 提取的全部 class 逐个在 `frontend/src/**/*.ts,tsx` 全量搜索，零命中的有：`md-code`(465-472)、`msg--assistant`(587-589/622-627/1467-1469)、`msg--user`(602-606/607-613/614-619/1462-1465)、`msg__body`、`msg__text`、`ds-card`(727-732)、`ds-input`(733-750)、`ds-kbd`(751-759)、`btn-primary`(762-787)、`btn-secondary`(789-809)、`ds-code-inline`(820-827)、`ds-code-block`(828-837)、`ds-modal-overlay`(840-848)、`ds-modal`(849-857)、`badge--accent/success/danger/warning`(894-918)、`ds-button-pill`(920-943)、`empty-state__icon`(970-973)、`mem-suggestion`(1237-1242)、`mem-doc`(1243-1245)、`drawer--wide/__head/__title/__summary/__close`(1267-1305)、`context-menu` 全族(1351-1412)、`pwa-titlebar`(1428-1430)。
    - **反误报核验**：① 全 CSS 文件互查——这些类只在 `gaea/styles.css` + `gaea/redesign.css` 出现，无 `@extend`/被别的 CSS 引用；② 无动态拼接（grep `btn-${`/`ds-${`/`badge--${`/`msg-${` 全仓 0 命中，唯一动态类是 `genui/renderNode.tsx:581` 的 `gui-btn-${tone}`，与本表无关）；③ `index.html` 0 命中；④ 仍在使用的是**同名不同类**，勿误伤：`ds-chip`（`FilterChip.tsx:10`）、`badge--muted`/`badge--project`（`ArchivesSection.tsx:38`、`DocEditor.tsx:23`、`FactCard.tsx:100`、`OfficeMemoryLibrary.tsx:507`、`SuggestionCard.tsx:29`）、`empty-state`（`CapabilitiesPanel.tsx:112`）、`drawer-backdrop`（`CommandPalette.tsx:135`）、`--ds-shadow-*` 是 **CSS 变量**不是类。
    - 建议：先删 `context-menu` 全族与 `drawer__*`（前者 0 命中、后者已被 Tailwind 取代），其余按块小步删除；每删一块跑一次 `pnpm test`。

11. **[P1]** 巨型组件函数（D6，见屎山清单）：`gaea/App.tsx:90 App` 952 行、`gaea/components/MemoryPanel.tsx:24 MemoryPanel` 806 行、`gaea/components/XlsxPreview.tsx:24 XlsxPreview` 786 行、`gaea/lib/mock/novel.ts:126 buildNovel` 744 行、`gaea/components/DocxPreview.tsx:79 DocxPreview` 738 行 —— **D6** — LOC≈0（重构，非删除）— confidence:**high**

12. **[P1]** 重复类型定义 65 组（D8）—— 典型：`Props` **24 份同名接口**（`characterlib/CharacterLibEditor.tsx:91`、`characterlib/CharacterLibFilterBar.tsx:14`、`imagegen/GenerationBar.tsx:11`、`Lightbox.tsx:11`、`WhisperMemoryList.tsx:21` …）；`ToolItem` 4 份（`SubagentThread.tsx:10`、`ToolCard.tsx:22`、`ToolGroup.tsx:9`、`Transcript.tsx:27`）；`MonitorStats`/`MonitorEngine`/`ModelMonitor` 双份（`components/launcher_parts.tsx:18/27/33` ⇔ `layouts/MainLayout.tsx:87/96/102`，逐行重复）；`CharacterCardProps` 双份（`characterlib/CharacterCard.tsx:34` ⇔ `novel/character/CharacterCard.tsx:9`）；`ModelInfo`（`api/engines.ts:8` ⇔ `gaea/lib/types/capabilities.ts:27`）；`BackendInfo`（`api/image.ts:51` ⇔ `api/settings.ts:8`）；`DiffLine`（`novel/editor/DiffReview.tsx:23` ⇔ `schedule/ScheduleDiffCard.tsx:24`）；`MemoryFact`（`WhisperMemoryModal.tsx:8` ⇔ `gaea/lib/types/memory.ts:7`，后者是导出源）—— **D8** — LOC≈250（重复声明本身）— confidence:**high**

13. **[P1]** `frontend/src/gaea/lib/bridge/novel.ts` 单文件 23 个导出**外部零引用**（202 行声明）—— **D1/D5** — LOC≈202 — confidence:**high**
    - 证据：`AnalysisV2ResultView:142`、`BookHealthChapterView:177`、`ChapterPlanView:208`、`PlanProblemView:222`、`PlanDeviationView:241`、`PromptIssueView:491`、`PromptTemplateMetaView:497`、`PromptTemplateBody:509`、`PromptTemplateDetailView:522`、`PromptSaveResultView:528`、`PromptPreviewResultView:534`、`PromptBundleImportResultView:539`、`NovelOutlineReconstructTaskState:556`、`OutlineReconstructItem:566`、`OutlineReconstructPreview:577`、`ChapterReviewEvidence:599`、`ChapterReviewDimension:605`、`NovelBookSourceImportStart:653`、`NovelBookSourceFailedChapter:658`、`NovelBookSourceEnginesPayload:680`、`FingerprintIssue:21`、`ForeshadowCleanupResult:58`、`ForeshadowSyncSkipReason:80` —— 这些标识符在**其它任何 src 文件里 0 命中**（含测试），文件本身有 24 个 importer。
    - 建议：去掉 `export` 关键字即可（0 行删除、显著缩小桥接面）；确属跨层契约的少数几个移入 `gaea/lib/types/`。

14. **[P1]** 全局「导出面过宽」：**327 个导出符号无任何外部引用**（54 个是值/常量、273 个是类型），分布在 `gaea/lib/bridge/*`、`gaea/lib/officeTurnProjection.ts`（11 个）、`gaea/lib/paneTabs.ts`（4 个）、`api/engines.ts`（7 个）、`api/image.ts`（4 个）、`api/characterlib.ts`（5 个）、`schedule/applyDiff.ts`（7 个）、`schedule/monte.ts`（5 个）等 —— **D1/D5** — LOC≈0（删 `export` 关键字）— confidence:**high**

15. **[P2]** `frontend/src/gaea/lib/officeTurnProjection.ts:427-432` 冗余分支 — **D2** — LOC≈2 — confidence:**high**
    - 证据：
      ```
      427:      if (!event.ok) {
      428:        return { state: withSets(state, { intentPath: null }), action: { type: "none" } };
      429:      }
      430:      if (state.openedThisTurn.has(event.path)) {
      431:        return { state: withSets(state, { intentPath: null }), action: { type: "none" } };
      432:      }
      ```
      两个 `return` 表达式逐字符相同。
    - 建议：合并为 `if (!event.ok || state.openedThisTurn.has(event.path)) { … }`。

16. **[P2]** `frontend/src/gaea/icons.ts:97` 恒等式 — **D2** — LOC≈1 — confidence:**high**
    - 证据：`const fontSize = size !== undefined ? size : undefined;`（等价于 `const fontSize = size`），下一行 `:98` 才做真正的 `fontSize !== undefined` 判断。
    - 建议：删掉中间变量，`:98` 直接用 `size !== undefined ? { ...style, fontSize: size } : style`。

17. **[P2]** `frontend/src/gaea/lib/spaceBindings.ts:777` `unknownBindingWarnings()` — **D1** — LOC≈4 — confidence:**high**
    - 证据：`grep -rn unknownBindingWarnings frontend/src` 只命中定义本身（`:777`）；测试 `spaceBindings.test.ts:29/37` 只用了 `resetUnknownBindingWarningsForTest`。
    - 建议：删除该函数。

18. **[P2]** `frontend/src/schedule/schedule.css` 死选择器 → **经核验为误报，不删**（见「不建议动的地方」）。真正可删的是同目录 `frontend/src/gaea/styles.css`（见第 10 条）。

19. **[P2]** `frontend/src/gaea/components/capabilities/ServersSection.tsx` + `ToolsSection.tsx` + `SkillsSection.tsx` 与 `frontend/src/gaea/components/CapabilitiesPanel.tsx` 的 props 逐层透传链 —— **D5** —— 见重复/抽象清单。

20. **[P2]** `frontend/src/schedule/applyDiff.ts` 7 个导出类型全部无人外部引用（`DiffFieldChange:22`、`TaskModify:29`、`TaskChange:35`、`LinkChange:41`、`ResourceChange:48`、`AssignmentChange:55`、`ApplyDiffSummary:62`，合计 48 行）—— **D1/D8** — LOC≈48 — confidence:**high**

---

## 全量发现表

| 文件:行 | 符号 | 类型 | LOC | 置信度 | 证据 | 建议 |
|---|---|---|---:|---|---|---|
| `frontend/src/gaea/components/SubagentsPanel.tsx:1` | SubagentsPanel（整文件） | D1 | 400 | high | 唯一 import=`SubagentsPanel.test.tsx:4`；其余 6 处命中皆注释 | 删文件 |
| `frontend/src/gaea/components/SubagentsPanel.test.tsx:1` | 测试 | D1 | 281 | high | 仅测死组件 | 随组件删 |
| `frontend/src/gaea/components/MorningBriefCard.tsx:1` | MorningBriefCard | D1 | 82 | high | 唯一 import=`MorningBriefCard.test.tsx:3`；`ModuleLauncher.tsx:372` 注释「仅撤书斋侧列入口」 | 产品确认后删 |
| `frontend/src/gaea/components/MorningBriefCard.test.tsx:1` | 测试 | D1 | 75 | high | 同上 | 随组件删 |
| `frontend/src/gaea/components/morning-brief-card.css:1` | 样式 | D1 | 98 | high | 仅被已死组件 import | 随组件删 |
| `frontend/src/gaea/components/TrendChart.tsx:1` | TrendChart / TrendPoint | D1 | 59 | high | 全仓 0 import；无同名 test；`glob **/TrendChart*` 仅自身 | 整文件删 |
| `frontend/src/gaea/styles.css:1351-1412` | `.context-menu*` 全族 | D1(CSS) | 62 | high | ts/tsx 0 命中；CSS 互查 0 命中；无动态拼接 | 整族删 |
| `frontend/src/gaea/styles.css:1267-1305` | `.drawer--wide/__head/__title/__summary/__close` | D1(CSS) | 39 | high | 同上（`drawer-backdrop` 是另一类，仍用） | 整族删 |
| `frontend/src/gaea/styles.css:762-809` | `.btn-primary/.btn-secondary` 族 | D1(CSS) | 48 | high | 0 命中；仅 `gui-btn-primary`（genui）在用 | 整族删 |
| `frontend/src/gaea/styles.css:733-759` | `.ds-input/.ds-kbd` 族 | D1(CSS) | 27 | high | 0 命中；`--ds-shadow-*` 是变量不是类 | 删 |
| `frontend/src/gaea/styles.css:587-632,1462-1469` | `.msg--assistant/--user/__body/__text` | D1(CSS) | 32 | high | `Message.tsx:264` 用的是 Tailwind `group/msg`，非该类 | 删 |
| `frontend/src/gaea/styles.css:840-857` | `.ds-modal-overlay/.ds-modal` | D1(CSS) | 18 | high | 0 命中 | 删 |
| `frontend/src/gaea/styles.css:920-943` | `.ds-button-pill` 族 | D1(CSS) | 24 | high | 0 命中 | 删 |
| `frontend/src/gaea/styles.css:894-918` | `.badge--accent/success/danger/warning` | D1(CSS) | 20 | high | 0 命中；`badge--muted/project` 仍在用，勿误删 | 删 |
| `frontend/src/gaea/styles.css:1237-1245` | `.mem-suggestion/.mem-doc` | D1(CSS) | 9 | high | 0 命中 | 删 |
| `frontend/src/gaea/styles.css:727-732` | `.ds-card` | D1(CSS) | 6 | high | 0 命中 | 删 |
| `frontend/src/gaea/styles.css:820-837` | `.ds-code-inline/.ds-code-block` | D1(CSS) | 18 | high | 0 命中 | 删 |
| `frontend/src/gaea/styles.css:465-472` | `.md-code` | D1(CSS) | 8 | medium | 0 命中（Markdown 已走 highlight.js） | 删前再确认 |
| `frontend/src/gaea/styles.css:970-973` | `.empty-state__icon` | D1(CSS) | 4 | high | `.empty-state` 仍在用（`CapabilitiesPanel.tsx:112`），仅 `__icon` 子类死 | 只删子类 |
| `frontend/src/gaea/styles.css:1428-1430` | `.pwa-titlebar` | D1(CSS) | 3 | medium | ts/tsx/html 0 命中（可能由 Wails 外壳注入） | 删前确认外壳 |
| `frontend/src/gaea/lib/bridge/novel.ts:142` | AnalysisV2ResultView 等 23 个 | D1/D5 | 202 | high | 标识符在其它 src 文件 0 命中；文件自身有 24 importer | 去 `export` |
| `frontend/src/gaea/lib/officeTurnProjection.ts:146/163/365/381/471/554/566` | OfficeFileTurnView/OfficeTurnProjection/PreviewAutoFrontState/… | D1/D5 | 111 | high | 同上（11 个符号） | 去 `export` |
| `frontend/src/gaea/lib/paneTabs.ts:23/38/183/191` | PaneTab/PaneTabsState/PaneTabsSnapshot/sanitizePaneTabsSnapshot | D1/D5 | 79 | high | 同文件内自用，无外部引用 | 去 `export`（`sanitizePaneTabsSnapshot` 可内联） |
| `frontend/src/gaea/lib/diffRender.ts:117` | FOLD_KEEP | D1 | 36 | high | 全仓 0 外部引用 | 去 `export` |
| `frontend/src/gaea/lib/sanitize.ts:91` | mdHtmlSchema | D1 | 31 | high | 同文件自用（own=5） | 去 `export` |
| `frontend/src/gaea/lib/changes.ts:135` | READ_TOOL_NAMES | D1 | 19 | high | 同文件自用 | 去 `export` |
| `frontend/src/gaea/lib/mock/shared.ts:68/85` | initialPriceSourcesMock/initialPriceFetchMock | D1 | 24 | medium | 同文件自用；mock 目录被 knip 列为 entry | 去 `export`（勿删数据） |
| `frontend/src/gaea/lib/versionCompare.ts:147/154` | buildDocxDiff/buildXlsxDiff | D1 | 17 | high | 0 外部引用 | 去 `export` 或内联 |
| `frontend/src/gaea/lib/mindmap.ts:126/128/129/147` | MM_NODE_W/MM_GAP_X/MM_GAP_Y/MindLayout | D1 | 4 | high | 0 外部引用 | 去 `export` |
| `frontend/src/gaea/lib/gbase.ts:53-56` | GBASE_MAX_VIEWS/CONDITIONS/SORTS/RULES | D1 | 4 | high | 0 外部引用 | 去 `export` |
| `frontend/src/gaea/lib/gbase.ts:10/25/48/186/283` | GbaseFilterOp/GbaseFilterCondition/GbaseConfig/GbaseRecord/GbaseGroup | D1 | 40 | high | 仅同文件（`parseGbaseConfig`）使用 | 去 `export` |
| `frontend/src/gaea/lib/spaceBindings.ts:777` | unknownBindingWarnings | D1 | 4 | high | 全仓仅定义处（测试只用 reset 变体） | 删函数 |
| `frontend/src/gaea/lib/spaceBindings.ts:730` | IndependentBindingName | D1 | 1 | medium | 全仓仅定义处；与 Work/Play/Shared 三兄弟不对称 | 删或补消费点 |
| `frontend/src/gaea/lib/spaceBindings.ts:711/713/717/722/734` | AssertNever/_NoStrayFacet/_NoMissingFacet/NamesOf/_NoFacetOverlap | D1 | 14 | low | **编译期断言类型，故意不被引用** | **不要删**（见「不建议动」） |
| `frontend/src/gaea/lib/reasoningDisplay.ts:4/5` | STREAMING_REASONING_TAIL_CHARS/LINES | D1 | 2 | high | 0 外部引用 | 去 `export` |
| `frontend/src/gaea/lib/tableData.ts:55` | tableBlockToMarkdown | D1 | 7 | high | 0 外部引用 | 去 `export` |
| `frontend/src/gaea/lib/verifyDiff.ts:182` | colToNum | D1 | 5 | high | 0 外部引用 | 去 `export` |
| `frontend/src/gaea/lib/planDiff.ts:88` | normalizeWsPath | D1 | 3 | high | 0 外部引用 | 去 `export` |
| `frontend/src/gaea/lib/fileLinks.ts:13` | FILE_EXT_RE | D1 | 4 | high | 0 外部引用 | 去 `export` |
| `frontend/src/gaea/lib/workspaceTabs.ts:67` | LEGACY_TAB_ALIASES | D1 | 4 | high | 0 外部引用（legacy 遗留） | 确认无迁移需求后删 |
| `frontend/src/gaea/lib/eventSync.ts:78` | RESYNC_MIN_GAP | D1 | 2 | high | 0 外部引用 | 去 `export` |
| `frontend/src/gaea/lib/mdViewPref.ts:5` | MD_VIEW_KEY | D1 | 2 | high | 0 外部引用 | 去 `export` |
| `frontend/src/gaea/lib/layoutPreferences.ts:103` | PREVIEW_DEFAULT_WIDTH | D1 | 5 | high | 0 外部引用 | 去 `export` |
| `frontend/src/gaea/lib/pageLazy.ts:30` | LAZY_MOUNT_BUFFER | D1 | 5 | high | 0 外部引用 | 去 `export` |
| `frontend/src/gaea/lib/mindmapEdit.ts:37` | findMindParent | D1 | 8 | high | 0 外部引用 | 去 `export` |
| `frontend/src/gaea/lib/toolArgs.ts:35/44` | ToolArgHunk/ParsedWriteArgs | D1 | 16 | high | 0 外部引用 | 去 `export` |
| `frontend/src/gaea/lib/store/controller_state.ts:111` | flushPendingUser | D1 | 10 | high | 0 外部引用 | 去 `export` |
| `frontend/src/gaea/lib/xlsxCellDiff.ts:23/36` | XlsxCellChangeRow/XlsxSheetDiffState | D1 | 29 | high | 0 外部引用 | 去 `export` |
| `frontend/src/gaea/lib/docxAnnotationQueue.ts:70/106/151/169` | QueueAddResult/QueueItemPatch/QueueStats/ExcerptLocation | D1 | 28 | high | 0 外部引用 | 去 `export` |
| `frontend/src/gaea/lib/pptxTextDiff.ts:50/59/83/88/91` | PptxRow/PptxPageDiff/… | D1 | 25 | high | 0 外部引用 | 去 `export` |
| `frontend/src/gaea/lib/verifyArtifacts.ts:33/41/47` | VerifyPagePair/VerifyThumbCell/VerifyThumbRow | D1 | 17 | high | 0 外部引用 | 去 `export` |
| `frontend/src/gaea/lib/taskActivity.ts:31/46/114` | TaskCardActivity/TaskCardActivityProvider/SubagentRunLike | D1 | 20 | high | 0 外部引用 | 去 `export` |
| `frontend/src/gaea/lib/eventSync.ts:42/90/95` | EventSyncTracker/EventSeqOutcome/NoteEventSeqOpts | D1 | 14 | high | 0 外部引用 | 去 `export` |
| `frontend/src/gaea/components/PasteManager.tsx:11/12/19/23` | LONG_PASTE_MIN_CHARS/LINES/shouldFoldPaste/renderPastedBlock | D1 | 11 | high | `PasteManager.tsx` 只有 1 个 importer（`MemoryPanel`），4 个导出全无外部引用 | 去 `export` |
| `frontend/src/components/novel/ChapterPlanCard.tsx:94` | ENDING_TYPES | D1 | 11 | high | 0 外部引用 | 去 `export` |
| `frontend/src/components/novel/ChapterPlanCard.tsx:55/64/76/275` | PlanProblem/PlanGateReport/PlanDeviation/ChapterPlanCardProps | D1 | 53 | high | 0 外部引用 | 去 `export` |
| `frontend/src/components/novel/create/chapterStreamTypes.ts:19/27/34/53/64/69` | CreateChapter*Event/AiTasteIssue | D1 | 52 | high | 0 外部引用 | 去 `export` |
| `frontend/src/components/novel/create/annotationMarks.ts:6/17` | runeToCodeUnit/AnnSegment | D1 | 23 | high | 0 外部引用 | 去 `export` |
| `frontend/src/components/novel/consistencyDeep.ts:6/7/8/63` | DEEP_CHAPTER_MIN/MAX/DEFAULT/DeepIssueAnnotation | D1 | 16 | high | 0 外部引用 | 去 `export` |
| `frontend/src/components/chat/AnsweredByLine.tsx:43` | answeredByCostText | D1 | 6 | high | 0 外部引用 | 去 `export` |
| `frontend/src/components/chat/MessageList.tsx:26/52` | MessageListProps/TOP_GROW_PX | D1 | 21 | high | 0 外部引用 | 去 `export` |
| `frontend/src/components/imagegen/historyMeta.ts:20` | isInlineDataUrl | D1 | 3 | high | 0 外部引用 | 去 `export` |
| `frontend/src/components/imagegen/CustomTemplateModal.tsx:8` | TEMPLATE_SIZE_OPTIONS | D1 | 2 | high | 0 外部引用 | 去 `export` |
| `frontend/src/components/settings/*Panel.tsx` | 各 Panel 的 `Props` 接口 | D8 | 240 | high | 24 份同名 `Props` 接口（见 D8 清单） | 命名 `XxxProps` 并集中 |
| `frontend/src/gaea/components/knowledge/utils.tsx:7` | escapeRegExp | D1 | 3 | high | 0 外部引用 | 去 `export` |
| `frontend/src/boards/manifests.ts:198` | deriveBoardLabel | D1 | 3 | high | 0 外部引用 | 去 `export` 或内联 |
| `frontend/src/boards/manifests.ts:156/159/220` | BoardId/_AssertBoardIdCoversLegacy/RemoteBoardManifest | D1 | 22 | high（前两项 medium） | `_Assert*` 为编译期断言 | 只处理 RemoteBoardManifest |
| `frontend/src/boards/types.ts:9/19` | BoardLayout/BoardNav | D1 | 12 | high | 0 外部引用 | 去 `export` |
| `frontend/src/boards/pageRegistry.ts:10` | RegisteredPage | D1 | 2 | high | 0 外部引用 | 去 `export` |
| `frontend/src/stores/appStore.ts:18` | ThemePresetMeta | D1 | 7 | high | 0 外部引用 | 去 `export` |
| `frontend/src/stores/appStore.ts:37/41` | THEME_PRESET_COLORS/THEME_PRESET_LABELS | D1 | 56 | high | 0 外部引用（同文件自用） | 去 `export` |
| `frontend/src/api/engines.ts:68/87/178/219/237/328/334` | ModelUsageStats/EngineSubtotal/HerdsmanModelStat/SavingsView/BenchmarkSummary/RouteEndpoint/RouteSuggestionTarget | D1 | 62 | high | 0 外部引用；文件自身 27 importer | 去 `export` |
| `frontend/src/api/image.ts:71/115/248/395` | ComfyUIStatus/ComfyLorasResult/ChapterArtEntry/ComfyTaskProgress | D1 | 27 | high | 0 外部引用（文件 39 importer） | 去 `export` |
| `frontend/src/api/characterlib.ts:8/10/16/153/201` | ProjectCharacter/CharacterListResult/CharacterDetail/FillAllResult/ConsistencyScore | D1 | 28 | high | 0 外部引用 | 去 `export` |
| `frontend/src/schedule/applyDiff.ts:22/29/35/41/48/55/62` | DiffFieldChange/TaskModify/TaskChange/LinkChange/ResourceChange/AssignmentChange/ApplyDiffSummary | D1 | 48 | high | 0 外部引用 | 去 `export` |
| `frontend/src/schedule/monte.ts:16/18/46/55/82` | MONTE_DEFAULT_RUNS/UNCERTAINTY/MonteTaskRate/MonteResult/MonteOptions | D1 | 51 | high | 0 外部引用 | 去 `export` |
| `frontend/src/schedule/aoaLayout.ts:19/83/89/227` | AOA_PINS_MAX/SegLabel/EdgeGeom/AOA_BRIDGE_R | D1 | 29 | high | 0 外部引用 | 去 `export` |
| `frontend/src/schedule/aoa.ts:22/84` | AoaNode/AOA_ASPECT_MAX | D1 | 27 | high | 0 外部引用 | 去 `export` |
| `frontend/src/schedule/dcma.ts:33-41/56` | DCMA_* / DcmaCheck / DcmaReport | D1 | 40 | high | 0 外部引用 | 去 `export` |
| `frontend/src/schedule/evm.ts:16/22/40` | EvmNumber/EvmResult/EvmOptions | D1 | 28 | high | 0 外部引用 | 去 `export` |
| `frontend/src/schedule/baseline.ts` / `deadline.ts:13` / `cost.ts:41` / `customFields.ts:17` / `ganttLinks.ts:12` | DeadlineCheck/TaskCost/CustomFieldType/LinkPts 等 | D1 | 30 | high | 0 外部引用 | 去 `export` |
| `frontend/src/schedule/store.ts:45/48` | ScheduleView/ScheduleSyncState | D1 | 11 | high | 0 外部引用（文件 31 importer） | 去 `export` |
| `frontend/src/schedule/api.ts:19/25/55` | ScheduleLoadOk/ScheduleSaveOk/ScheduleProjectCreateOk | D1 | 15 | high | 0 外部引用 | 去 `export` |
| `frontend/src/schedule/layout.ts:7/13/18` | LayoutItem/LayoutEdge/LayoutPos | D1 | 13 | high | 0 外部引用 | 去 `export` |
| `frontend/src/schedule/ganttExport.ts:44/247` | GanttExportSvg/exportNotesLines | D1 | 14 | high | 0 外部引用 | 去 `export` |
| `frontend/src/schedule/gantt/rowDerive.ts:22/38` | SynGroupRow/DerivedGanttRow | D1 | 20 | high | 0 外部引用 | 去 `export` |
| `frontend/src/schedule/chatPrefs.ts:15/29/30` | ChatPrefs/CHAT_WIDTH_MIN/MAX | D1 | 13 | high | 0 外部引用 | 去 `export` |
| `frontend/src/schedule/types.ts:26` | DurationUnit | D1 | 25 | high | 0 外部引用（文件 87 importer） | 去 `export` |
| `frontend/src/schedule/cpm.ts:106` | CpmOptions | D1 | 10 | high | 0 外部引用 | 去 `export` |
| `frontend/src/schedule/ganttGroup.ts:28` | GanttGroupOptions | D1 | 10 | high | 0 外部引用 | 去 `export` |
| `frontend/src/schedule/gantt/GanttOverlay.tsx:10` / `GanttTimeline.tsx:17` / `useFitView.ts:20` | GanttOverlayProps/GanttTimelineProps/FitViewController | D1 | 32 | high | 0 外部引用 | 去 `export` |
| `frontend/src/components/ModuleLauncher.tsx:29` | UserOutlined | D7 | 1 | high | eslint `no-unused-vars` | 删 import |
| `frontend/src/components/novel/create/BranchWizardModal.tsx:41` | prevChapter | D7 | 1 | high | eslint `no-unused-vars`（调用方仍在传） | 删 prop + 调用点 |
| `frontend/src/components/novel/editor/GhostText.test.tsx:8` | waitFor | D7 | 1 | high | eslint `no-unused-vars` | 删 import |
| `frontend/src/gaea/components/memoryhub/CostProjectsView.tsx:12` | EntryPicker / slug | D7 | 2 | high | eslint `no-unused-vars` ×2；同文件 `:5` 注释称 EntryPicker 仍在消费（陈旧） | 删 import + 改注释 |
| `frontend/src/gaea/components/memoryhub/cost_projects_parts.tsx:7` | message | D7 | 1 | high | eslint `no-unused-vars` | 删 import |
| `frontend/src/schedule/MonteView.tsx:8` | cpm（解构为 `_cpm`） | D7 | 1 | medium | 组件体 0 引用；调用方仍传 `cpm` | 删 prop（或保留兼容） |
| `frontend/src/gaea/components/Welcome.tsx:177` | cwd（解构为 `_cwd`） | D7 | 1 | medium | 同上 | 删 prop |
| `frontend/src/gaea/lib/officeTurnProjection.ts:427-432` | 两个相同 return | D2 | 2 | high | 表达式逐字符相同 | 合并条件 |
| `frontend/src/gaea/icons.ts:97` | fontSize 恒等三元 | D2 | 1 | high | `size !== undefined ? size : undefined` | 内联到 `:98` |
| `frontend/src/components/novel/editor/GhostText.tsx:87` | useEffect cleanup 里的 `reqSeqRef.current` | D6 | 3 | high | eslint `react-hooks/exhaustive-deps` 唯一告警 | 进入 effect 先取 `const seq = reqSeqRef.current` |
| `frontend/src/components/characterlib/CharacterLibEditor.tsx:91` / 23 个文件 | `interface Props` × 24 | D8 | 240 | high | 同名接口 24 份（清单见 D8 节） | 改 `XxxProps` 或收口 types |
| `frontend/src/components/launcher_parts.tsx:18/27/33` ⇔ `frontend/src/layouts/MainLayout.tsx:87/96/102` | MonitorStats/MonitorEngine/ModelMonitor | D8/D4 | 30 | high | 跨文件重复块 11 行（`launcher_parts.tsx:19-22` ⇔ `MainLayout.tsx:88-91`） | 抽 `types/monitor.ts` |
| `frontend/src/components/characterlib/CharacterCard.tsx:34` ⇔ `frontend/src/components/novel/character/CharacterCard.tsx:9` | CharacterCardProps | D8 | 12 | high | 两个同名不同定义 | 统一 |
| `frontend/src/api/engines.ts:8` ⇔ `frontend/src/gaea/lib/types/capabilities.ts:27` | ModelInfo | D8 | 10 | high | 双定义 | 收口 `types/` |
| `frontend/src/api/image.ts:51` ⇔ `frontend/src/api/settings.ts:8` | BackendInfo | D8 | 10 | high | 双定义 | 收口 `api/types.ts` |
| `frontend/src/components/novel/editor/DiffReview.tsx:23` ⇔ `frontend/src/schedule/ScheduleDiffCard.tsx:24` | DiffLine | D8 | 8 | high | 双定义，字段不同 | 各自改名 |
| `frontend/src/components/WhisperMemoryModal.tsx:8` ⇔ `frontend/src/gaea/lib/types/memory.ts:7` | MemoryFact | D8 | 8 | high | 后者已是导出源 | 前者改为 import |
| `frontend/src/boards/manifests.ts:156` ⇔ `frontend/src/layouts/MainLayout.tsx:15` | BoardId | D8 | 6 | high | 双定义 | 收口 |
| `frontend/src/components/SecurityBanner.tsx:16` ⇔ `frontend/src/components/settings/SecurityPanel.tsx:15` | LanExposure | D8 | 6 | high | 双定义 | 收口 |
| `frontend/src/api/image.ts:286` ⇔ `frontend/src/components/imagegen/VisionTrial.tsx:21` | VisionCallResult | D8 | 6 | high | 双定义 | 收口 |
| `frontend/src/gaea/components/SubagentThread.tsx:10` / `ToolCard.tsx:22` / `ToolGroup.tsx:9` / `Transcript.tsx:27` | ToolItem ×4 | D8 | 24 | high | 4 份同名接口 | 收口 `types/` |
| `frontend/src/gaea/app/useWorkspaceLayout.ts:10` ⇔ `frontend/src/gaea/lib/workspaceTabs.ts:213` | WorkspacePanelPersistedState | D8 | 8 | high | 双定义 | 收口 |
| `frontend/src/gaea/components/memoryhub/CostNotesView.tsx:283-286` ⇔ `frontend/src/gaea/components/memoryhub/cost_projects_parts.tsx:244-247` | Field 组件 | D4 | 22 | high | 跨文件重复块（283-286 ⇔ 244-247、279-281 ⇔ 240-242） | 抽 `memoryhub/Field.tsx` |
| `frontend/src/gaea/components/DeliverableCards.tsx:239-254` ⇔ `frontend/src/gaea/components/deliverables/DeliverableRow.tsx:203-218` | 产物行渲染 | D4 | 23 | high | 重复块 23 行 | 收敛到 DeliverableRow |
| `frontend/src/gaea/components/memoryhub/PriceSourcesPanel.tsx:372-380` ⇔ `PriceSourcesRepository.tsx:103-111` | 列表项渲染 | D4 | 16 | high | 重复块 16 行 | 抽共享行组件 |
| `frontend/src/gaea/lib/remarkFileLinks.ts:13-24` ⇔ `frontend/src/gaea/lib/remarkMemCitations.ts:12-23` | remark 插件骨架 | D4 | 13 | high | 两组重复块（13/12 行） | 抽 `remarkHelper` |
| `frontend/src/gaea/components/BrowserPanel.tsx:222-232` ⇔ `DeliverablesPanel.tsx:389-399` ⇔ `SubagentsPanel.tsx:301-311` | 空态/错误态块 | D4 | 18×3 | medium | 三方逐行相同 | 抽 `<PanelEmpty>` |
| `frontend/src/schedule/gantt/GanttBars.tsx:100-104` ⇔ `GanttTable.tsx:131-135` | 行渲染 | D4 | 12 | medium | 重复块 12 行 | 抽共享行 |
| `frontend/src/components/novel/CreateNovelModal.tsx:94-97` ⇔ `ImportNovelModal.tsx:81-84` | 校验/表单片段 | D4 | 11 | medium | 重复块（另有 5 处 8 行级） | 抽共享表单壳 |
| `frontend/src/gaea/components/DocxPreview.tsx:276-278` ⇔ `PptxEditPanel.tsx:199-201` | 编辑态按钮条 | D4 | 10 | medium | 重复块 10 行 | 抽按钮条 |
| `frontend/src/schedule/PdmView.tsx:66-68` ⇔ `networkExport.ts:242-244` | 节点绘制 | D4 | 10 | low | 重复块 10 行 | 观察 |

---

## 疑似重复组件 / 重复逻辑（含收敛建议与风险）

### A. 复制粘贴出的「孪生模态」（最高优先级，收益/风险比最好）

**`memoryhub/CostImportModal.tsx`（380 行）⇄ `memoryhub/KnowledgeImportModal.tsx`（约 300 行）**

- 8 段跨文件重复块（最长 22 行），合计 ≈86 行**逐字节相同**（含超长 Tailwind className 串）。
- 风险：**低**。两者只差「导入到成本库 / 导入到知识库」的列定义与保存回调。
- 建议：`<ImportModalShell open path fileName onClose onImported parse onSubmit labels />`；先抽共享块，再逐步把列定义数据化。

**`imagegen/` 下的多个 `*Modal.tsx`**（`CutoutModal`、`InstructionEditModal`、`VariantChainModal`、`TemplatePickerModal`、`CustomTemplateModal`）以及 `settings/` 下的多个 `*Panel.tsx`：本次跨文件重复检测未捕捉到 ≥10 行的整块重复，但各自的 modal 外壳（遮罩 + 标题行 + 底部按钮条）与 `settings` 面板的标题/描述行是**同构样板**（`AppearancePanel.tsx` 内即手写了 3 次 `color:'var(--md-sys-color-text-secondary)'` + `background:'var(--md-sys-color-surface-variant)'` 的卡片头）。建议抽 `<ModalShell>` / `<SettingRow>` 两个原语，属**中等收益、低风险**。

### B. 双轮询 store（结构性重复）

**`gaea/lib/agentNetworkStore.ts`（约 130 行）⇄ `gaea/lib/subagentRunsStore.ts`（约 130 行）**

- 两侧各自定义 `Subscription`/`Poller` 类型、`syncTicker`、`.catch/.finally` 拉取尾（重复块 10 行 ×2）。
- 差异仅为「无参 `GaeaAgentNetwork()`」vs「有参 `GaeaSubagentRuns(path)`」。
- 建议：`createPoller<T>({ fetch, keyOf })` 工厂。
- 风险：**中**——两个 store 被 `TasksWorkbench` / `Sidebar` / `AgentTree` 消费，且注释显示有「不可见门控」「单在途收敛」等语义，重构需覆盖既有测试（`agentNetworkStore.test.ts`、`subagentRunsStore.test.ts`、`TasksWorkbench.test.tsx`）。

### C. 四处「拖拽改宽」手势

`ResizableDrawer.tsx:128-132`、`useSidebar.ts:46-50`、`useWorkspaceLayout.ts:80-83`、`usePreviewPanel.ts:174-177` 两两相同。
建议 `usePointerResize({ save, setResizing })`。风险：低（纯 DOM 手势，有既有测试）。

### D. 预览/格式化的三处小重复

- `formatSize` 两份字节级相同（`FilePreview.tsx:43` / `FilePreviewModal.tsx:24`）+ 语义相同的 `formatBytes`/`fmtSize` 三份。
- pptx 逐页渲染 25 行重复（`FilePreview.tsx:476-493` ⇔ `FilePreviewModal.tsx:273-290`）——说明 `FilePreview`（内联右栏）与 `FilePreviewModal`（弹窗）本就是同一功能的两套皮。
- 建议：先合 `formatSize`（1 行改动），再评估 `FilePreview` 是否可直接复用 `FilePreviewModal` 的内容体。

### E. D8 类型的「重灾区」

- `interface Props` **24 份**同名（`Lightbox.tsx:11`、`MarkdownContent.tsx:26`、`imagegen/*` 5 份、`characterlib/*` 4 份、`Whisper*` 4 份 …）。虽不冲突（各文件私有），但 24 处同名会持续误导 grep。
- `MonitorStats/MonitorEngine/ModelMonitor` 明文两套（`launcher_parts.tsx` ⇔ `MainLayout.tsx`），**且渲染片段也重复 11 行** → 建议直接删 `launcher_parts.tsx` 中的一份并 import `MainLayout` 或抽 `types/monitor.ts`。
- 建议优先级：先把 `MonitorStats` 族与 `ToolItem` 族收敛（真实跨文件冲突风险），`Props` 改名批次处理。

### F. `gaea/components/capabilities/` 的 props 透传链（D5）

`CapabilitiesPanel.tsx:23` → `ToolsSection.tsx:165 ToolsTabContent` → `ToolsSection.tsx:120 ToolGroup` → `ServersSection.tsx:31/76/173/313` → `SkillsSection.tsx:25/43/143`：`toolCounts`/`skillCounts`/`expanded`/`onToggle` 逐层原样透传 3~4 层，中间层不改变语义。
建议：`expanded/onToggle` 族用 context 或直接就地 useState；`toolCounts/skillCounts` 改由 `useCapabilitiesData` 在下游直接取。风险：中（涉及 4 个文件与大量测试）。

---

## 屎山清单（结构性重构建议）

### 1. 超长文件（>800 行，本域 10 个）

`gaea/components/Sidebar.tsx`(1066)、`gaea/App.tsx`(1041，非本域责任)、`gaea/components/Transcript.tsx`(1024)、`gaea/components/DocxPreview.tsx`(1021)、`gaea/lib/mock/novel.ts`(870)、`components/characterlib/CharacterLibEditor.tsx`(869)、`gaea/components/context/inspector.tsx`(861)、`gaea/lib/spaceBindings.ts`(835)、`gaea/components/MemoryPanel.tsx`(830)、`components/novel/ChapterEditor.tsx`(822)、`layouts/MainLayout.tsx`(812)、`gaea/components/XlsxPreview.tsx`(809)。
> 参照：`gaea/components/capabilities/*` 与 `gaea/components/context/*` 已是既有拆分产物（文件头注释均标注「T6-10.1 拆分产物，行为零变化」），说明团队已有拆分范式，可直接复用。

### 2. 超长函数（≥150 行，本域 99 个）

| 函数 | 行数 |
|---|---:|
| `gaea/App.tsx:90 App` | 952 |
| `gaea/components/MemoryPanel.tsx:24 MemoryPanel` | 806 |
| `gaea/components/XlsxPreview.tsx:24 XlsxPreview` | 786 |
| `gaea/lib/mock/novel.ts:126 buildNovel` | 744 |
| `gaea/components/DocxPreview.tsx:79 DocxPreview` | 738 |
| `gaea/components/KnowledgePanel.tsx:18 KnowledgePanel` | 731 |
| `gaea/components/CostLibraryView.tsx:36 CostLibraryView` | 633 |
| `gaea/components/DagPanel.tsx:60 DagPanel` | 599 |
| `gaea/components/memoryhub/OfficeMemoryLibrary.tsx:24` | 582 |
| `gaea/components/Sidebar.tsx:492 Sidebar` / `:136 SessionRow` | 574 / 355 |
| `gaea/components/memoryhub/CostProjectsView.tsx:14` | 557 |
| `gaea/components/FilePreview.tsx:54` | 546 |

建议：按「数据派生 → const 化」「子区块 → 子组件」「事件处理 → `useXxxHandlers`」三步切；`XlsxPreview`/`DocxPreview` 与既有 `xlsxpreview/*` 拆分子目录（`FormulaBar`/`SheetGrid`/`MiniChart` 已存在）保持一致。

### 3. 最深 JSX 嵌套（花括号深度）

`gaea/components/DagPanel.tsx:620`（深度 **11**）、`components/novel/ChapterAnalysisPanel.tsx:264`（9）、`gaea/components/Sidebar.tsx:666`（8）、`gaea/components/MemoryPanel.tsx:695`（8）、`gaea/components/xlsxpreview/SheetGrid.tsx:257`（8）、`components/novel/ChapterEditor.tsx:552`（8）、`gaea/components/deliverables/EvidenceSection.tsx:248`（8）。
建议：>`8` 的位置抽成局部小渲染函数（`renderRow(...)`）而非再下钻 JSX。

### 4. 巨型条件链 / switch

`gaea/lib/officeTurnProjection.ts`（`previewAutoFrontReduce` 51 行 switch + `writeResult` 分支）、`gaea/components/memoryhub/CostImportModal.tsx:23`（338 行组件内含多段解析/映射 switch）、`gaea/components/UpdateBanner.tsx`（switch 6 分支 → 4 个分支返回同一结构的 div，可表驱动）。建议：把 `switch (kind)` 改成 `Record<kind, (e) => Result>` 映射表（仓库已有此范式，如 `gaea/lib/changes.ts`）。

### 5. useEffect 依赖混乱（105 处 `[]` 空依赖体引用外部标识符）

本次只报**证实的 1 条**（eslint 的 `GhostText.tsx:87`），其余 104 处经抽样看多为「挂载一次性初始化 + ref 兜底」的刻意写法（例：`components/RelationGraph.tsx:109` 只看容器尺寸、`components/SkillModal.tsx:29` 只看 `mountedRef`），**不宜机械加依赖**。
建议：对 `components/launcher_parts.tsx:212`（163 行 effect）、`components/novel/ChapterEditor.tsx:67`（132 行 effect）、`components/novel/BookHealthPanel.tsx:121`（77 行 effect）三个超长 effect 做人工拆分——它们是真正的维护风险点。

### 6. 内联超长样式

`gaea/components/Composer.tsx:477`（单行 ≈600 字符的 className，含两处 `shadow-[inset_0_1px_0_color-mix(...)]` 重复）、`gaea/styles.css` 中大量 `color-mix(in srgb, …)` 内联。建议把 Composer 的长 className 提为模块常量（同文件已对 `HEAD_BTN` 这么做，可对齐）。

---

## 不建议动的地方（含原因）

1. **`frontend/src/gaea/lib/mock/bindingSignatures.ts`（661 行）——唯一 importer 是 `mock/contract.test.ts`，但不可删。**
   `frontend/knip.json` 的 `entry` 明确列出 `src/gaea/lib/mock/*.ts`、`src/gaea/lib/mock/office/*.ts`；`contract.test.ts:133` 断言 `Object.keys(bindingSignatures).length === bindingSignatureCount`。它是「前端 mock × Go 绑定签名」的契约真值源，属**刻意的 test-only 入口**。
2. **`frontend/src/gaea/icons.ts`、`frontend/src/gaea/lib/bindingNames.ts`、`frontend/src/gaea/lib/bridge/drift.ts`。**
   三者出现在 `knip.json` 的 `ignore` 白名单里，且 `icons.ts:94 wrap()` 是逐个手写 `wrap(XxxOutlined)` 的导出——grep 到的「未用导出」多为动态/契约消费。**不要按未用导出清理。**
3. **`frontend/src/gaea/lib/spaceBindings.ts:711/713/717/722/734`**（`AssertNever`/`_NoStrayFacet`/`_NoMissingFacet`/`NamesOf`/`_NoFacetOverlap`）。
   这些类型别名**故意不被引用**：TS 在别名声明处即检查泛型约束（`T extends never`），因此「声明即断言」。`spaceBindings.test.ts:4/50/51` 的注释明确记录这是实验钉死的编译期防线。删掉 = 掉一层漂移保护。
4. **`frontend/src/gaea/locales/zh-TW.ts` / `en.ts` / `zh.ts`。**
   `zh-TW.ts` 的 import 图入度为 0，但 `gaea/lib/i18n.tsx:39` 用 `await import("../locales/zh-TW")` 动态加载（注释明确说明「模板拼接会因无法静态解析而构建失败」）。**动态 chunk，勿删。** 三份字典还需保证键集一致（`zh-TW.ts:1` 注释：「鍵集與 en.ts 必須完全一致」）。
5. **`frontend/src/schedule/schedule.css` 的 `.sched-group-c0..c5`。**
   本次脚本判为「死选择器」是**误报**：`schedule/gantt/GanttBars.tsx:69` / `GanttTable.tsx` 用 `` `sched-group-c${d.synSeq % 6}` `` 动态拼接。同理 `.sched-base-shifted/added/removed` 需先 grep `sched-base-` 模板再定夺——**按本次结论不要动**。
6. **`frontend/src/genui/**`、`frontend/src/pages/**`、`frontend/src/hooks/**`、`frontend/src/utils/**`、`frontend/src/types/**`、`frontend/src/data/**`、`frontend/src/test/**`。**
   不在本审计范围，未做结论；其上文只作为「谁在用」的证据来源。
7. **`frontend/wailsjs/**`。**
   自动生成的 Go→JS 绑定。本次只把它当作「某符号仍被 Rust/Go 侧使用」的反向证据；**任何删除决策不得以它为据**（也正因如此，`api/*.ts` 里被 Go 侧消费的类型需谨慎）。
8. **`gaea/lib/mock/*` 的巨型 fixture（`mock/novel.ts` 870 行、`mock/cost.ts` 747 行、`mock/bindingSignatures.ts` 661 行）与其 `buildXxx()` 巨型函数。**
   它们**故意**是「一文件一域的大字面量 + 一个大 builder」，拆成小文件反而破坏「同域 mock 单文件可读」的既有约定；`knip.json` 已把它们设为 entry。**只建议接口层收敛，不建议拆文件。**
9. **Tailwind 类名手写重复（如 `className="flex items-center gap-1 ..."` 长串）。**
   这类重复由 Tailwind 设计使然，抽 `<Row>` 之类封装会让调试期 DevTools 不可读。**本次不列为发现**（只在同一 className 串跨文件复制粘贴时才算 D4，见重复清单）。
10. **`eslint.config.js` 中 `local/no-raw-hex` 的 15 个 `exemptFiles` 白名单**（`RelationGraph.tsx`、`EmotionStarMap.tsx`、`ganttExport.ts`、`appStore.ts` 等）。
   它们是图表/主题令牌的合法硬编码色，删除白名单会立刻 lint 失败。**不要动。**
11. **`frontend/src/gaea/components/MergedPanel.tsx`（13 行纯布局壳）。**
    看着像「只透传 props 的包装」（D5），但它的注释明确记载了用户拍板的分区语义（主区 flex-1 / 次区 max-h-45%）；**是设计决策，不是过度抽象。**
12. **`frontend/src/schedule/*View.tsx` 的单一 importer（全部由 `pages/SchedulePage.tsx` 引入）。**
    188 个「只有一个生产 importer」的文件里大部分是这种「页面级唯一消费」，属正常分层；**不要按单 importer 判死。**（真正异常的是「唯一 importer 是它自己的 test」——本域仅 3 个，已列在整文件死代码清单。）

---

## 附录：本次审计执行的全部命令与可复现口径

```
# 类型检查（工作目录 C:\AI\wubigrok\frontend）
pnpm exec tsc -b --noEmit
# lint（本域 8 个目录，排除无 ts 的 src/v3）
pnpm exec eslint src/gaea src/components src/schedule src/api src/boards src/layouts src/stores --format json --output-file <tmp>\eslint.json
# 结果：938 文件 / 12 message / 0 error
```
自建分析（脚本均落在系统 `%TEMP%`，**未写入仓库**）：
1. import 图：`from '…'` / `import('…')` / `require('…')` / `vi.mock('…')` / `import '…'` 五种形态，共解析 3,636 条边。
2. 未用导出：对 8 个目录 938 个文件抽取 `export function|const|let|var|class|enum|type|interface` 与 `export {…}`，与全 `src` 标识符索引（含 353 个测试文件）求差 → **327 个候选**。
3. 未用 props：按「参数解构 + 花括号配平 + 只剥离单双引号字符串（保留模板串 `${}` 表达式）」提取组件体，逐 prop 计数 → 2 个 `_` 前缀的刻意未用（均非缺陷）。
4. 跨文件重复块：归一化行（去注释/空行/压缩空白）后做 8 行滚动哈希，跨文件比对并合并相邻窗口 → **314 个窗口命中、42 个 ≥10 行块**。
5. 死 CSS：提取选择器中全部 class 名，在 938 个 ts/tsx 全量搜索，并与**其它 CSS 文件互查**、检查**动态拼接**与 `index.html` → 仅 `gaea/styles.css` 的 53 块可靠（`schedule.css` 的 group 类被证伪）。
6. 不可达/冗余：`return/throw` 后续同缩进语句、相邻同表达式 `return`、恒真三元与恒真判断 → 逐条人工复核，**剔除 8 处误报**后保留 2 条。
