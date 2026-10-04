# 前端审计 · pages/hooks/utils/lib/types

> 仓库：`C:\AI\wubigrok`（Gaea，Go 1.26 + Wails v2 / React 18 + TS + Vite）
> 审计范围：`frontend/src/pages/`、`frontend/src/hooks/`、`frontend/src/utils/`、`frontend/src/lib/`、`frontend/src/types/`、`frontend/src/data/`、`frontend/src/test/`、`frontend/src/App.tsx`、`frontend/src/main.tsx`、`frontend/src/events.ts`
> 只读审计，未修改任何仓库文件（本报告除外）。基线：`tsc -b --noEmit` 通过（exit 0）；`eslint`（同范围）仅 2 条 warning（见 F-02）。
>
> **重要前提（避免误报）**：本报告用到的所有"死代码"判定，都是在**全仓库**（1000+ 个 `.ts/.tsx`，含被排除目录 `components/`、`gaea/`、`boards/`、`stores/`、`layouts/`、`api/`、`v3/`、`genui/`、`schedule/` 以及全部 `*.test.ts(x)`）范围内做的符号名 + import 图双向核对。`wailsjs/` 仅作为"仍在使用"的证据使用，未作为删除依据。

---

## 0. 结论摘要（先看这段）

1. **范围内没有任何整文件死代码。** 全仓 import 图（入口 = `src/main.tsx` + `src/test/setup.ts` + 全部 `*.test.ts(x)`）反查：183 个范围内文件全部可达；页面文件与 `boards/manifests.ts` 的 `page` 键、`main.tsx` 的 `registerPage(...)` 表 100% 对齐，没有未挂载页面、没有 `XxxOld/XxxV2/XxxLegacy/XxxCopy` 命名残留、没有任何 `lazy(() => import())` 落在死分支里。
2. **范围内没有任何"无人引用"的导出符号。** 99 个"零 import 命中"的导出经全仓符号核对后，全部至少被一处引用（多为同文件内引用）。
3. **真正可动手的只有三类**：① 4 个未使用的 `React` 导入 + 2 个彻底没被引用的类型声明 + 2 个多余的 type-only import（**编译器/lint 背书**）；② 74 个"只在本文件用、却写了 `export`"的符号（去掉 `export` 即可，零行为变化）；③ 14 份逐字重复的 `errText()` 抄本（最值得做的 D4 收敛）。
4. 范围内**没有** TODO/FIXME/HACK 注释，**没有** >10 行注释掉的代码块，**没有**空 `useEffect`，**没有**只赋值不读取的 state/ref。
5. 一个重要的**诊断盲区**：`frontend/tsconfig.app.json` 显式 `"noUnusedLocals": false, "noUnusedParameters": false`。"范围内死代码很少"这一结论在很大程度上是它造成的 —— 一次性开 `--noUnusedLocals` 跑 `tsc` 会立刻抛出 **38 条 TS6133**（范围内 4 条、范围外 34 条）。
6. 唯一的结构性债是 5 个 >800 行的文件和 20 个 >300 行的巨型组件函数（`CharacterPage` 单函数 1011 行），以及 `modelcenter/` 下"每个 hook 都自带一份 `errText` + 一组空 catch（全范围 80 处）"的样板复制。

---

## 范围与文件统计

| 目录 | .ts/.tsx 文件数 | CSS 文件数 | 体积（.ts/.tsx） |
|---|---|---|---|
| `src/pages/` | 142 | 5 | 1,631.9 KB |
| `src/hooks/` | 14 | 0 | 107.0 KB |
| `src/utils/` | 32 | 0 | 42.6 KB |
| `src/lib/` | 6 | 0 | 12.8 KB |
| `src/types/` | 2 | 0 | 19.0 KB |
| `src/data/` | 16 | 0 | 90.7 KB |
| `src/test/` | 1 | 0 | 3.8 KB |
| 根文件 `App.tsx` / `main.tsx` / `events.ts` | 3 | 0 | 19.7 KB |
| **合计（范围）** | **216** | **5** | **≈1,928 KB** |

> 注：`src/App.tsx` 实际位于 `frontend/src/App.tsx`（不在 `frontend/` 根），由 `src/main.tsx:5` `import App from './App.tsx'` 引入。任务书写的 `frontend/App.tsx` 不存在。

### 最大的 20 个文件（范围内，含 CSS）

| 字节 | 行数 | 路径 |
|---|---|---|
| 84,643 | 1403 | `src/pages/CreatePage.tsx` |
| 57,890 | 1117 | `src/pages/ChapterPage.tsx` |
| 53,614 | 844 | `src/pages/CreatePage.test.tsx` |
| 51,197 | 1012 | `src/pages/CharacterPage.tsx` |
| 48,632 | 987 | `src/pages/WeixinPage.tsx` |
| 43,174 | 679 | `src/pages/ChapterPage.test.tsx` |
| 41,065 | 746 | `src/pages/SchedulePage.tsx` |
| 37,389 | 1498 | `src/pages/modelcenter/modelcenter.css` |
| 31,846 | 501 | `src/pages/NovelSettingPage.test.tsx` |
| 30,782 | 628 | `src/pages/CostLibraryPage.tsx` |
| 29,323 | 562 | `src/data/imageTemplates.ts` |
| 29,186 | 583 | `src/pages/sin/SinSidePanel.tsx` |
| 28,752 | 623 | `src/pages/MemoryHubPage.tsx` |
| 28,288 | 798 | `src/pages/sin/sin.css` |
| 25,992 | 466 | `src/pages/ChatPage.test.tsx` |
| 25,803 | 430 | `src/pages/modelcenter/StatsSection.tsx` |
| 25,538 | 404 | `src/pages/WeixinPage.test.tsx` |
| 25,447 | 525 | `src/pages/sin/useSinStory.ts` |
| 24,252 | 489 | `src/pages/OriginalSinPage.tsx` |
| 23,920 | 496 | `src/pages/ImageGenPage.tsx` |

### 超长组件函数（>300 行，按函数体行数）

| 函数跨度 | 位置 | 符号 |
|---|---|---|
| 1011 | `src/pages/CharacterPage.tsx:84-1094` | `CharacterPage` |
| 711 | `src/pages/WeixinPage.tsx:140-850` | `WeixinPage` |
| 696 | `src/pages/SchedulePage.tsx:70-765` | `SchedulePage` |
| 516 | `src/pages/modelcenter/EngineSection.tsx:10-525` | `EngineSection` |
| 492 | `src/pages/OriginalSinPage.tsx:50-541` | `OriginalSinPage` |
| 490 | `src/pages/MemoryHubPage.tsx:160-649` | `MemoryHubPage` |
| 477 | `src/pages/sin/useSinStory.ts:124-600` | `useSinStory` |
| 475 | `src/pages/ChatPage.tsx:39-513` | `ChatPage` |
| 475 | `src/pages/ImageGenPage.tsx:55-529` | `ImageGenPage` |
| 423 | `src/pages/CharacterLibraryPage.tsx:41-463` | `CharacterLibraryPage` |
| 411 | `src/pages/modelcenter/StatsSection.tsx:30-440` | `StatsSection` |
| 381 | `src/pages/CostLibraryPage.tsx:109-489` | `CostLibraryPage` |
| 366 | `src/pages/modelcenter/hooks/useEngineState.ts:92-457` | `useEngineState` |
| 338 | `src/pages/sin/SinBookSourcePanel.tsx:41-378` | `SinBookSourcePanel` |
| 333 | `src/hooks/useImageGenConfig.ts:25-357` | `useImageGenConfig` |
| 331 | `src/pages/ModelCenterPage.tsx:49-379` | `ModelCenterPage` |
| 328 | `src/pages/modelcenter/HerdsmanCatalogSection.tsx:68-395` | `HerdsmanCatalogSection` |
| 323 | `src/pages/modelcenter/BindSection.tsx:37-359` | `BindSection` |
| 297 | `src/pages/modelcenter/BenchmarkSection.tsx:65-361` | `BenchmarkSection` |
| 223 | `src/hooks/useChatStream.ts:44-266` | `useChatStream` |

---

## 整文件死代码清单

**结论：范围内 0 个整文件死代码。**（这是本次审计最重要的"反向发现"，请不要依据直觉去删文件。）

逐一核对过的、**看起来像死代码但实际活着**的高危候选：

| 路径 | 为什么看着像死的 | 反证（仍在使用） |
|---|---|---|
| `src/pages/GaeaPage.tsx`（26 L） | 只有 26 行、只是转发 `gaea/App` | `src/main.tsx:19` `registerPage('GaeaPage', lazy(...))` + `src/boards/manifests.ts:99` `page: 'GaeaPage'` |
| `src/pages/HomePage.tsx`（514 L） | 不在 `main.tsx` 的 registerPage 表里 | `src/pages/NovelPage.tsx:37` `const HomePage = React.lazy(() => import('./HomePage'))`（小说板块子 pane） |
| `src/data/templates/*.ts`（13 个分片，共 1,154 L） | 无任何外部 import `templates/anime` 等路径 | `src/data/herdsmanTemplates.ts:6,13...` 逐个 import 后聚合 re-export |
| `src/data/herdsmanTemplates.ts`(35 L) | 只做聚合 | `src/data/imageTemplates.ts:3` + `src/data/imageTemplates.test.ts:5` 引入 |
| `src/lib/wailsApp.ts`（13 L） | 只转发一层 `window.go.app.App` | `src/components/SkillModal.tsx:37`、`src/stores/appStore.ts:441` 等（范围内 0 引用，范围外有） |
| `src/lib/pollingGate.ts`（15 L） | 只有 1 个 1 行函数 | `src/hooks/usePollingGate.ts:2` 引入；后者被 `pages/WeixinPage.tsx:15`、`pages/modelcenter/ResourceMonitor.tsx:14` 等 12 处引入 |
| `src/utils/zIndex.ts`（50 L） | 范围内 0 引用 | `src/components/Lightbox.tsx:6`、`src/components/novel/ChapterEditor.tsx:13`、`src/gaea/components/BrowserPanel.tsx:9` |
| `src/utils/uiStyles.ts`（9 L） | 范围内 0 引用 | `src/components/imagegen/CutoutModal.tsx:6` 等 8 个组件 |
| `src/utils/emotionColors.ts`（16 L） | 范围内 0 引用 | `src/components/EmotionStarMap.tsx:2`、`src/components/WhisperEmotionPanel.tsx:2` |
| `src/pages/chat/emotions.ts`（46 L） | 范围内无 page 引用 | `src/components/chat/EmotionSpeakSelector.tsx:10` |
| `src/pages/chat/{constants,types,utils}.ts` | chat 子目录，只有 `emotions.ts` 被外部用 | `src/pages/ChatPage.tsx:28`、`src/hooks/useChatTopics.ts` 等大量引用 |
| `src/types/wails.d.ts`（317 L） | 全局声明文件，无 import | 提供 `Window.go/app` + `RuntimeAPI` 全局类型；但内部有 2 个死类型（见 F-02/F-03） |

---

## Top 20 发现（按可删行数 × 安全度排序）

1. **[P0] `src/pages/chapter/errText.ts:5` `errText` — D4 重复实现 — LOC≈42 — confidence:high**
   - 证据：同一函数（3 行体 + 1 行签名）`return (err instanceof Error && err.message) || fallback` 逐字出现在 **14 个文件**：`pages/chapter/errText.ts:5`、`pages/chapter/ChapterIllustration.tsx:33`、`pages/modelcenter/hooks/useAttributionState.ts:21`、`pages/modelcenter/hooks/useBindState.ts:18`、`pages/modelcenter/hooks/useEngineState.ts:28`、`pages/modelcenter/hooks/useVoiceState.ts:25`、`pages/modelcenter/BenchmarkSection.tsx:9`、`pages/sin/SinBookSourcePanel.tsx:23`、`pages/sin/useSinCast.ts:27`、`pages/sin/useSinNotes.ts:22`、`pages/sin/useSinStory.ts:23`、`pages/CharacterLibraryPage.tsx:28`、`pages/CharacterPage.tsx:20`、`hooks/useImageGenConfig.ts:21`（`errText.ts` 本身 7 行）。其中 `pages/chapter/errText.ts` 已经是抽出后的共享副本，另外 13 处是抄本（13 行签名 + 13 行体 ≈ 39 行可删）。
   - 建议：把 `errText.ts` 上移到 `src/utils/errText.ts`（或已有 `src/utils/` 下新增一行式模块），13 个文件改成 `import { errText } from '../../utils/errText'`，删掉各自 3 行本地定义。零行为变化、纯机械替换。

2. **[P0] `src/types/wails.d.ts:137` `LorebookEntry` — D1 死代码 — LOC≈6 — confidence:high**
   - 证据：全仓 grep 仅 1 处命中，就是定义行本身；`AppAPI` 里 Lorebook 段（`wails.d.ts:291-292`）只有注释 `// ── Lorebook ──`，没有任何方法返回该类型。
   - 建议：删除该 interface（连带 `// ── Lorebook ──` 空段注释）。

3. **[P0] `src/types/wails.d.ts:126` `PlotBranch` — D1 死代码 — LOC≈10 — confidence:high**
   - 证据：全仓 grep 仅 2 处命中且都在同一文件（`wails.d.ts:125` 注释 + `:126` 定义）。`AppAPI` 无任何方法返回 `PlotBranch`（`ApplyBranch` 返回 `Promise<void>`，`wails.d.ts:289`）。
   - 建议：删除 interface + 上方注释；若后端仍有 `plot_branch_handler.go`，改为在消费点按需内联结构。

4. **[P0] `src/types/wails.d.ts:12` `TTSConfig, TTSStatus` type-only import — D1 死代码 — LOC≈2（+ `types/index.ts` 2 个定义 13 行） — confidence:high**
   - 证据：`eslint` 明确报 2 条 warning：`'TTSConfig' is defined but never used`、`'TTSStatus' is defined but never used`（`wails.d.ts:12`）。全仓 grep `TTSConfig` 仅命中 `types/index.ts:189` 定义 + `wails.d.ts:12` import；`TTSStatus` 仅命中 `types/index.ts:197` 定义 + `wails.d.ts:12` import。**这是范围内唯一被 lint 抓到的死代码，也是全范围 0 error 的唯一 2 条 warning。**
   - 建议：删 `wails.d.ts:12` 的两个 import 名 → 顺带删 `src/types/index.ts:189-200` 的 `TTSConfig`/`TTSStatus` 两个 interface（15 行）。删除前请确认后端 TTS 相关 handler 不再有前端消费者（当前确认为 0）。

5. **[P0] 74 个"只在本文件使用却 export"的符号 — D1/D5 过度暴露 — LOC≈74（纯关键字） — confidence:high**
   - 证据：对 99 个"零直接 importer"的范围内导出做全仓符号核对后，74 个的符号名**在定义文件之外 0 次出现**，且在本文件内**均有使用**（见下方全量表 F-01 到 F-74，字段 `extFiles=0`）。
   - 典型：`hooks/useChatStream.ts:24 UseChatStreamOptions`（本文件 `:44` 用作参数类型）、`pages/modelcenter/ui.tsx:55 EngineMark`（本文件 `:191` 使用）、`utils/readingProgress.ts:1 ReadingProgress`（本文件 3 处使用）、`pages/modelcenter/utils.tsx:8 ModelMeta`（本文件 8 处使用）。
   - 建议：逐个去掉 `export` 关键字。**零行为变化、零风险，是本次最安全的清理目标。** 唯一注意：`ModelMeta`/`ModelKind`/`CharacterStatus` 这类"其实希望被共享"的类型，去 `export` 后若后续要复用需重新导出——但这正是应该做的（要么共享要么私有）。
   - 例外（**不要**去 export）：`pages/modelcenter/resource.ts:21 ResourceMonitorData`、`utils/theme.ts:57 ROLE_COLORS`、`utils/readingSettings.ts:10-11 ReadingColumn/ReadingTheme`、`utils/readingAnnotations.ts:7 AnnotationColor`、`data/imageTemplates.ts:19 CustomTemplate`、`pages/modelcenter/utils.tsx:36 classifyModel`、`pages/sin/storyText.ts:213 SinGalleryItem`、`pages/modelcenter/ui.tsx:6 StatusTone`、`pages/modelcenter/context.ts:13/78/131`、`types/wails.d.ts:104/149/151` —— 这些被范围外文件引用（属"范围内定义、范围外消费"）。

6. **[P0] `src/utils/emotionColors.ts:4` `EMOTION_COLORS` 与 `src/pages/chat/constants.ts:22` `EMO_COLORS` — D4 重复实现 — LOC≈9 — confidence:high**
   - 证据：两份 key 完全相同的 9 色映射，值逐字一致：`SWEET_ATTACHMENT:'#f472b6'`、`SHY_HEARTBEAT:'#fb7185'`、`TSUNDERE:'#f59e0b'`、`HURT_GRIEVANCE:'#a78bfa'`、`ANGRY_ATTACK:'#ef4444'`、`COLD_DETACHED:'#94a3b8'`、`FEARFUL_OBEDIENT:'#c084fc'`、`QUIET_FOND:'#fbbf24'`、`CALM_RATIONAL:'#60a5fa'`（`emotionColors.ts:4-10` vs `chat/constants.ts:22-26`）。
   - 建议：`chat/constants.ts` 改为 `export { EMOTION_COLORS as EMO_COLORS } from '../../utils/emotionColors'`（或直接改消费点）。两文件注释都自称是"唯一来源"，实际互为副本——属于典型的"改一处必漏另一处"。

7. **[P1] `src/pages/modelcenter/*` + `src/pages/sin/*` + `src/hooks/*` 大量空 catch — D2 空操作错误处理 — LOC≈80 — confidence:high**
   - 证据：`catch (_) {}` / `.catch(() => {})` 在审计范围内共 **80 处**，密集区：`hooks/useVoiceChat.ts` 30 处（`:124,151,164,176,182,191,200,205,206,275,280,305,378,424,444,450,456,501,507,511,533,539,551,555,556,557,561`）、`pages/modelcenter/hooks/useEngineState.ts` 16 处（`:131,135,139,143,147,151,155,263,269,276,282,290,301,312,323,334,345,358,368,376,386`）、`hooks/useChatTopics.ts` 7 处（`:125,136,144,158,197,200,212`）、`pages/ModelCenterPage.tsx` 6 处（`:82,99,100,101,129,130`）、`pages/modelcenter/hooks/useImageState.ts:49,59`、`hooks/useVoiceState.ts:71,78`、`hooks/useBindState.ts:94`、`pages/CharacterLibraryPage.tsx:37`、`pages/ChatPage.tsx:208,323,332`、`pages/CreatePage.tsx:1158`、`hooks/useChatVoice.ts:47,55,57`、`hooks/useFeatureModel.ts:25,27,42,44`。
   - 建议：**不要**盲目加日志（会污染 gaea.log 的错误信号）。只在"用户可感知的失败"处补 `message.error`，其余保留但在行尾注明 `// 静默：失败不影响主流程（原因：…）`。`useVoiceChat.ts` 的 30 处是最大集中点，建议先只动这一处。

8. **[P1] `src/utils/scroll.ts:3` `isNearBottom` — D5 过度抽象 — LOC≈4 — confidence:high**
   - 证据：函数体就是 `return distanceFromBottom < threshold`（`scroll.ts:4`）。消费点仅 2 个：`pages/ChatPage.tsx:116`、`pages/OriginalSinPage.tsx:155`（均写 `isNearBottom(el.scrollHeight - el.scrollTop - el.clientHeight)`）。另有 `src/gaea/components/Transcript.tsx:25` 自带一份同名 `isNearBottom(el)` 实现（未复用本模块）。
   - 建议：要么内联到 2 个消费点并删文件（`utils/scroll.test.ts` 12 行测试一并删，LOC≈16），要么让 `Transcript.tsx` 复用它。当前形态是"包了一层的 `<` 运算 + 4 行测试"，属典型无用包装。

9. **[P1] `src/hooks/useImageGenHistory.ts:22` `RESTORE_LIMIT` — D1 死代码 — LOC≈3 — confidence:high**
   - 证据：全仓 grep 仅 2 处命中，均在定义文件内（`:22` 定义 + 1 处使用）；`RESTORE_LIMIT` 无任何 importer。它是被导出以便测试，但 `hooks/useImageGenHistory.test.ts:59-64` 并未使用它。
   - 建议：去掉 `export`（或若已无使用则整删）。同类：`hooks/useImageGenQueue.ts:16/39`、`pages/sin/sinPanelState.ts:12/13`、`pages/sin/useSinNotes.ts:18`、`pages/sin/useSinStory.ts:52`、`utils/readingSettings.ts:22`。

10. **[P1] `src/pages/modelcenter/modelPrefs.ts:6,16,24` `loadPinnedModels` / `persistPinnedModels` / `togglePinnedModel` — D5 只用一次的抽象 — LOC≈34 — confidence:high（整体）**
    - 证据：三个函数均为"零外部 importer + 仅本文件使用"。`:6 loadPinnedModels` 本文件 2 处（含 `useState(loadPinnedModels)`）、`:16 persistPinnedModels` 1 处、`:24 togglePinnedModel` 1 处。
    - 建议：把 `persistPinnedModels`/`togglePinnedModel` 内联进唯一的 `usePinnedModels()`（`modelPrefs.ts:32`），去 export。`loadPinnedModels` 保留文件级私有（`useState` 惰性初始化需要函数引用，不能内联）。

11. **[P1] `src/pages/sin/{storyText.ts,sinToolMeta.ts,sinPanelState.ts,...}` 中 15 个 `export` 收窄为私有 — D1/D5 — LOC≈15 — confidence:high**
    - 证据：`sinToolMeta.ts:38 parseToolArgs`、`sinToolMeta.ts` 所属文件自身 `:64` 使用；`storyText.ts:173 parseArtifacts`/`:224 cueOrder`/`:236 galleryCache`/`:238 gallerySignature`、`useSinStory.ts:34 nextKey`/`:40 toStoryView`/`:55 toMessageView`/`:76 findToolRow`、`useSinCast.ts:25 LIBRARY_PAGE_SIZE`/`:32 toCharacter`、`useSinNotes.ts:18 SIN_CONFLICT_PREFIX`、`SIN_MESSAGES_PAGE_SIZE`/`sinPanelState.ts:43 defaultSinPanelOpen` — 全部"本文件 1~3 处使用、全仓 0 处外部引用"。
    - 建议：批量去 `export`。注意 `sinThumbCache.ts:35 __resetSinThumbCacheForTest` 与 `sin/useSinStory.ts:21 SIN_STREAM_SILENCE_TIMEOUT_MS` **必须保留 export**（被 `*.test.tsx` 使用，见 `SinProcessCard.artifact.test.tsx`、`useSinStory.test.tsx`）。

12. **[P1] `src/pages/modelcenter/utils.tsx:377` `COSYVOICE_FALLBACK_VOICES` + `:392 localTTSFallbackVoices` + `:397 localTTSDefaultVoice` — D5 单点抽象 — LOC≈8 — confidence:medium**
    - 证据：`:377` 定义后仅在本文件 `:392` 被读；`:392`/`:397` 属"范围外消费"（被 `components/` 引用），但 `COSYVOICE_FALLBACK_VOICES` 只服务于这一条链。需人工确认是否有意作为可覆盖的配置出口（当前无 override 点）。
    - 建议：若确认无 override 需求，去 `export` 并考虑直接内联到 `localTTSFallbackVoices`。

13. **[P1] 14 个 `if (x instanceof Error && ...)` 型 errText 变体 — D4 — LOC≈12 — confidence:high**
    - 证据：另有 3 处"非 fallback 变体"：`hooks/useChatTopics.ts:90` `const errText = (err: unknown) => err instanceof Error ? err.message : String(err)`、`pages/chat/utils.ts:88` 同样的内联表达式、`pages/CharacterLibraryPage.tsx:81,129,154` 传 `String(err)` 当 fallback（语义与统一版不同，替换时需保留调用点语义）。
    - 建议：统一版 `errText(err, fallback)` 已覆盖这两类；`useChatTopics.ts:90` 是"无 fallback"版，可用 `errText(err, String(err))` 替代。替换时逐点核对 fallback 文案不变。

14. **[P1] `src/pages/sin/useSinStory.ts:23` `errText` 与 `:34 nextKey` / `:40 toStoryView` / `:55 toMessageView` / `:76 findToolRow` 四个私有 helper — D5/D6 — LOC≈60 — confidence:medium**
    - 证据：`useSinStory.ts` 单文件 601 行、`useSinStory` 函数体 477 行（`:124-600`）。4 个 helper 均只在文件内使用。
    - 建议：把 `toStoryView`/`toMessageView`/`findToolRow`/`nextKey` 抽到 `pages/sin/storyText.ts`（已是该目录的解析层）或新建 `sin/useSinStory.helpers.ts`，让 `useSinStory` 回到 300 行内。属重构，非删除。

15. **[P1] `src/pages/CreatePage.tsx:34-153` 一批只在本文件使用的声明 — D5 — LOC≈20 — confidence:high**
    - 证据：`WizardRequest`(:34)、`BRAINSTORM_MSG_KEY`(:35)、`GEN_PREFS_KEY`(:39)、`NovelGenPrefs`(:40)、`GEN_PREFS_DEFAULT`(:41)、`loadGenPrefs`(:44)、`EntityGraphNode`(:68)、`EntityGraphEdge`(:69)、`EntityGraph`(:70)、`GRAPH_GROUP_TOKENS`(:73)、`graphGroupColor`(:82)、`toEntityGraph`(:89)、`EntityGraphSvg`(:104)、`RailGroup`(:153) —— 全仓 0 处外部引用，均仅本文件（1403 行）内使用。
    - 建议：全部已有 `export`？**注意**：这些都不是 `export`（`CreatePage.tsx` 只 `export default`），所以**无需动作**。列在此处仅作为"不要把它们误判为可删"的记录 —— 它们是 CreatePage 的私有实现。

16. **[P1] `src/pages/CharacterPage.tsx:54-76` `CHAR_FILTER_KEY`/`CharFilterState`/`readCharFilters`/`writeCharFilters` — D4 localStorage 样板 — LOC≈25 — confidence:high**
    - 证据：这套"`KEY + projectPath` → `JSON.parse` → 失败回退默认"的读写样板在同一仓库至少重复 6 份：`pages/CharacterPage.tsx:63-77`、`utils/readingAnnotations.ts:28-48`、`utils/readingBookmarks.ts:19-39`、`utils/readingProgress.ts:9-26`、`pages/CharacterLibraryPage.tsx:37,70`、`pages/NovelPage.tsx:48-60,97-110`、`pages/NovelSettingPage.tsx:35-39,187`、`pages/ChatPage.tsx:80-103`、`pages/sin/sinPanelState.ts:26-40`、`pages/modelcenter/modelPrefs.ts:8-22`、`data/imageTemplates.ts:557-575`、`utils/bookSearchHistory.ts:10-27`。
    - 建议：在 `src/utils/` 新增 `localStore.ts`：`readJSON<T>(key, fallback)` / `writeJSON(key, v)` / `readFlag(key)` / `writeFlag(key, b)`（全部内置 try/catch）。逐点替换可省 80~120 行样板。**这是本次收益最高但需要最多回归测试的一项**——务必分批做，先动 `utils/` 内的 4 个（已有测试覆盖）。

17. **[P2] `src/pages/{CreatePage,ChapterPage,ChatPage,CostLibraryPage,SinBookSourcePanel,WeixinPage}.test.tsx` 内 30+ 候选未使用声明 — D1 测试内死声明 — LOC≈40 — confidence:medium**
    - 证据：这些名字在**包括自身文件**的全仓范围内仅出现 1 次（即声明处），说明连测试自己都没用：`CostLibraryPage.test.tsx:81 mockFull`、`ChapterPage.test.tsx:27 novelB`、`:462 clickAndAwaitConfirm`、`ChatPage.test.tsx:11 ChatTopicLike`/`:18 ChatMessageLike`/`:27 runtimeMock`/`:168 TOPIC_PLAIN`/`:169 LONG_REPLY`/`:179 renderChat`/`:223 sendMessage`、`SinBookSourcePanel.test.tsx:36/37/39`、`WeixinPage.test.tsx:71 selectChannel`、`CreatePage.test.tsx:142 runtimeOffs`、`modelcenter/*.test.tsx` 的多个 `renderXxx`/`mockXxx`。
    - **重要**：`tsc --noUnusedLocals` 诊断（见下两条）**并未**把这些报出来 —— 说明它们确实被间接使用（我漏判）**或** `noUnusedParameters` 语义差异。因此本条**不可直接删**。
    - 建议：**需人工确认**。建议开一次专项 lint（`eslint --rule '{"@typescript-eslint/no-unused-vars":"error"}'`）逐条核对，不要脚本直删。

18. **[P2] `frontend/tsconfig.app.json` 关闭了 `noUnusedLocals` / `noUnusedParameters` — D1（诊断盲区） — LOC≈? — confidence:high**
    - 证据：`frontend/tsconfig.app.json` 显式写有 `"noUnusedLocals": false, "noUnusedParameters": false`（同时 `strict: true`、`noFallthroughCasesInSwitch: true`）。这解释了为什么 `tsc -b --noEmit` 在存在未使用的局部声明时仍然 exit 0。
    - 已实测（只读一次性诊断，未改配置）：`tsc --noEmit --noUnusedLocals --noUnusedParameters -p tsconfig.app.json` 全仓报 **38 条 TS6133**。其中落在本次审计范围（`pages/hooks/utils/lib/types/data/test`）的 **4 条**见下一条；另有 34 条在范围外（`components/`、`gaea/`、`genui/` —— 请别的审计线认领）。
    - 建议：**先只做诊断**（不改配置的话可用 `tsc --noEmit --noUnusedLocals --noUnusedParameters` 一次性跑，或临时改后回滚），把输出作为"未使用局部声明"的权威清单；再决定是修缮代码还是保持配置。

19. **[P0] `tsc --noUnusedLocals` 在审计范围内实测出 4 条确定死声明 — D1 死代码 — LOC≈4 — confidence:high（编译器背书）**
    - 证据（TS6133 原文，全部落在审计范围）：
      - `src/pages/GaeaPage.tsx(1,1)`: `'React' is declared but its value is never read.`（`:1 import React from 'react'`；JSX 已走 `react-jsx` 自动运行时，无需显式 React）
      - `src/pages/modelcenter/BenchmarkSection.tsx(1,8)`: `'React' is declared but its value is never read.`
      - `src/pages/modelcenter/RetrievalEvalSection.tsx(1,8)`: `'React' is declared but its value is never read.`
      - `src/pages/sin/SinBookSourcePanel.test.tsx(7,1)`: `'React' is declared but its value is never read.`
      - 另有范围外同类 34 条，例如 `components/ModuleLauncher.tsx(29,32)`: `'UserOutlined' is declared but its value is never read.`、`gaea/components/memoryhub/CostProjectsView.tsx(12,10)`: `'EntryPicker' is declared but its value is never read.`、`src/components/novel/create/BranchWizardModal.tsx(41,9)`: `'prevChapter' is declared but its value is never read.`
    - 建议：删除这 4 个 `React` 默认 import（若文件内不再需要 React 命名空间）。**这是本次全范围唯一有编译器背书的死代码清单**，可以直接动手。范围外的 34 条请转交对应审计线。

20. **[P2] `src/pages/modelcenter/charts.tsx:20 niceMax` / `:29 trendXTicks`、`StatsSection.tsx:16 fmtCacheRate`、`HerdsmanCatalogSection.tsx:49/56/62 fmtParams/fmtMs/fmtTps`、`StrategySection.tsx:30 domainRef`、`RetrievalEvalSection.tsx:17 recallTone`、`AttributionSection.tsx:24 featureLabel`/`:42 buildGroups` — D1 私有未使用？ — LOC≈? — confidence:low**
    - 证据：这些符号在自身文件内出现 1 次，全仓仅 1 个文件。`tsc --noUnusedLocals` 诊断**并未**把它们报出来（见上一条的实测输出），说明它们实际上被使用（我的纯文本 grep 漏判），**可以确认它们是活代码**。
    - 建议：**不要动**。此条保留仅为记录"grep 判死不可靠、编译器才可靠"这一教训。

21. **[P2] `src/pages/modelcenter/modelcenter/context.ts:60` 与 `hooks/useBindState.ts:27` — D8 重复类型定义 — LOC≈2 — confidence:high**
    - 证据：`pages/modelcenter/context.ts:60 modelRoutes: Record<string, { engine: string; model: string; source: string }>` 与 `pages/modelcenter/hooks/useBindState.ts:27 modelRoutes: Record<string, { engine: string; model: string; source: string }>` 两处匿名内联类型逐字相同（另有 `BindSection.test.tsx:15` 的 `modelRoutes: {}` 与 `ModelCenterPage.failover.test.tsx:72` 的 mock）。
    - 建议：提取为命名类型（如 `ModelRoute = { engine: string; model: string; source: string }`）并在 `context.ts` 导出，两处引用。

---

## 全量发现表

> 类型：D1 死代码 / D2 不可达冗余分支 / D3 注释代码 / D4 重复实现 / D5 过度抽象 / D6 屎山 / D7 无用 props·state / D8 重复类型
> `extFiles` = 该符号名在 `frontend/src` 全仓（含 `*.test.tsx`，含被排除目录）中**除定义文件外**出现的文件数。`extFiles=0` 且 `inFile≥1` ⇒ 该 `export` 关键字是多余的。

### F-01 ~ F-74：只在本文件使用却 export 的符号（去掉 `export`，零风险）

| 文件:行 | 符号 | 类型 | LOC | 置信度 | 证据 | 建议 |
|---|---|---|---|---|---|---|
| src/data/imageTemplates.ts:39 | `CORE_CATEGORIES` | D1/D5 | 1 | high | extFiles=0；仅本文件 :548 展开进 `CATEGORIES` | 去 `export` |
| src/data/imageTemplates.ts:55 | `CORE_TEMPLATES` | D1/D5 | 1 | high | extFiles=0；仅本文件 :550 展开进 `TEMPLATES` | 去 `export` |
| src/events.ts:76 | `FrontendEventName` | D1/D5 | 1 | high | extFiles=0；仅本文件内使用 | 去 `export` |
| src/hooks/useChatStream.ts:24 | `UseChatStreamOptions` | D1/D5 | 3 | high | extFiles=0；本文件 :44 用作参数类型 | 去 `export` |
| src/hooks/useChatStream.ts:34 | `ChatSendParams` | D1/D5 | 6 | high | extFiles=0；本文件 :75 用作参数类型 | 去 `export` |
| src/hooks/useChatTopics.ts:14 | `UseChatTopicsOptions` | D1/D5 | 4 | high | extFiles=0；本文件 :19 用作参数类型 | 去 `export` |
| src/hooks/useChatVoice.ts:11 | `UseChatVoiceOptions` | D1/D5 | 5 | high | extFiles=0；本文件 :17 用作参数类型 | 去 `export` |
| src/hooks/useImageGenHistory.ts:12 | `UseImageGenHistoryOptions` | D1/D5 | 6 | high | extFiles=0；本文件 :24 用作参数类型 | 去 `export` |
| src/hooks/useImageGenHistory.ts:22 | `RESTORE_LIMIT` | D1 | 1 | high | extFiles=0；测试 `useImageGenHistory.test.ts` 未使用 | 去 `export` |
| src/hooks/useImageGenQueue.ts:16 | `ImageGenQueueConfig` | D1/D5 | 20 | high | extFiles=0；本文件 :39 内嵌引用 | 去 `export` |
| src/hooks/useImageGenQueue.ts:39 | `UseImageGenQueueOptions` | D1/D5 | 4 | high | extFiles=0；本文件 :45 用作参数类型 | 去 `export` |
| src/pages/chapter/editChrome.tsx:15 | `EditChromeProps` | D1/D5 | 20 | high | extFiles=0；本文件 :39 用于 `React.FC<>` | 去 `export` |
| src/pages/chapter/novelSearchUtils.ts:87 | `SnippetSeg` | D1/D5 | 3 | high | extFiles=0；本文件 2 处 | 去 `export` |
| src/pages/chapter/readingAnnotation.ts:13 | `AnnotationMatch` | D1/D5 | 3 | high | extFiles=0；本文件 2 处 | 去 `export` |
| src/pages/chapter/readingAskSession.ts:7 | `ReadingAskRole` | D1/D5 | 2 | high | extFiles=0；本文件 :16 引用 | 去 `export` |
| src/pages/chapter/readingBookmark.ts:15 | `BookmarkSnapshot` | D1/D5 | 4 | high | extFiles=0；本文件内使用 | 去 `export` |
| src/pages/chapter/readingChrome.tsx:20 | `ReadingChromeProps` | D1/D5 | 18 | high | extFiles=0；本文件用于组件签名 | 去 `export` |
| src/pages/chapter/readingHighlight.ts:18 | `TextNodeSpan` | D1/D5 | 4 | high | extFiles=0；本文件 2 处 | 去 `export` |
| src/pages/chapter/readingHighlight.ts:135 | `ReadingSelection` | D1/D5 | 6 | high | extFiles=0；本文件内使用 | 去 `export` |
| src/pages/chapter/readingPanel.tsx:14 | `ReadingPanelProps` | D1/D5 | 15 | high | extFiles=0；本文件用于 `React.FC<>` | 去 `export` |
| src/pages/chapter/searchHitAnnotation.ts:16 | `SearchHitAnchor` | D1/D5 | 8 | high | extFiles=0；本文件内使用 | 去 `export` |
| src/pages/chat/emotions.ts:16 | `SpeakEmotionOption` | D1/D5 | 8 | high | extFiles=0；本文件 :25 用作数组元素类型 | 去 `export` |
| src/pages/modelcenter/hooks/useAttributionState.ts:25 | `AttributionState` | D1/D5 | 20 | high | extFiles=0；本文件为 hook 返回类型 | 去 `export` |
| src/pages/modelcenter/hooks/useBindState.ts:22 | `BindState` | D1/D5 | 40 | high | extFiles=0；本文件为 hook 返回类型 | 去 `export` |
| src/pages/modelcenter/hooks/useEngineState.ts:37 | `EngineState` | D1/D5 | 40 | high | extFiles=0；本文件为 hook 返回类型 | 去 `export` |
| src/pages/modelcenter/hooks/useImageState.ts:12 | `ImageState` | D1/D5 | 30 | high | extFiles=0；本文件为 hook 返回类型 | 去 `export` |
| src/pages/modelcenter/hooks/useStatsState.ts:13 | `StatsState` | D1/D5 | 25 | high | extFiles=0；本文件为 hook 返回类型 | 去 `export` |
| src/pages/modelcenter/hooks/useVoiceState.ts:29 | `VoiceState` | D1/D5 | 45 | high | extFiles=0；本文件为 hook 返回类型 | 去 `export` |
| src/pages/modelcenter/modelPrefs.ts:6 | `loadPinnedModels` | D1/D5 | 8 | high | extFiles=0；本文件 2 处 | 去 `export` |
| src/pages/modelcenter/modelPrefs.ts:16 | `persistPinnedModels` | D1/D5 | 6 | high | extFiles=0；本文件 1 处 | 内联进 `usePinnedModels` |
| src/pages/modelcenter/modelPrefs.ts:24 | `togglePinnedModel` | D1/D5 | 4 | high | extFiles=0；本文件 1 处 | 内联进 `usePinnedModels` |
| src/pages/modelcenter/resource.ts:4 | `ResourceMonitorEngine` | D1/D5 | 6 | high | extFiles=0；本文件为 `ResourceMonitorData` 字段类型 | 去 `export` |
| src/pages/modelcenter/resource.ts:11 | `ResourceMonitorStats` | D1/D5 | 8 | high | extFiles=0；同上 | 去 `export` |
| src/pages/modelcenter/resource.ts:69 | `ResourceLevel` | D1/D5 | 8 | high | extFiles=0；本文件 `resourceLevel()` 返回类型 | 去 `export` |
| src/pages/modelcenter/ui.tsx:55 | `EngineMark` | D1/D5 | 4 | high | extFiles=0；本文件 :191 `<EngineMark>` | 去 `export` |
| src/pages/modelcenter/ui.tsx:60 | `StatusText` | D1/D5 | 6 | high | extFiles=0；本文件 :200 使用 | 去 `export` |
| src/pages/modelcenter/ui.tsx:134 | `ModelCardProps` | D1/D5 | 6 | high | extFiles=0；本文件用于 `ModelCard` | 去 `export` |
| src/pages/modelcenter/utils.tsx:8 | `ModelMeta` | D1/D5 | 25 | high | extFiles=0；本文件 8 处 | 去 `export`（或反向提为共享） |
| src/pages/modelcenter/utils.tsx:34 | `ModelKind` | D1/D5 | 2 | high | extFiles=0；本文件 4 处 | 去 `export` |
| src/pages/modelcenter/utils.tsx:151 | `engineColors` | D1/D5 | 2 | high | extFiles=0；本文件 2 处 | 去 `export` |
| src/pages/modelcenter/utils.tsx:154 | `engineLabels` | D1/D5 | 2 | high | extFiles=0；本文件 2 处 | 去 `export` |
| src/pages/modelcenter/utils.tsx:192 | `ModelAvailability` | D1/D5 | 2 | high | extFiles=0；本文件内使用 | 去 `export` |
| src/pages/modelcenter/utils.tsx:252 | `COMFY_MODEL_CHECKPOINT` | D1 | 1 | high | extFiles=0；本文件内使用 | 去 `export` |
| src/pages/modelcenter/utils.tsx:260 | `isQuantEfficientCheckpoint` | D1/D5 | 1 | high | extFiles=0；本文件 :265 使用 | 去 `export` |
| src/pages/modelcenter/utils.tsx:328 | `FeatureState` | D1/D5 | 2 | high | extFiles=0；本文件 2 处 | 去 `export` |
| src/pages/modelcenter/utils.tsx:377 | `COSYVOICE_FALLBACK_VOICES` | D1/D5 | 12 | medium | extFiles=0；仅本文件 :392 读取 | 去 `export` + 评估内联 |
| src/pages/sin/illustrationQueue.ts:10 | `IllustrationQueueSnapshot` | D1/D5 | 8 | high | extFiles=0；本文件为快照返回类型 | 去 `export` |
| src/pages/sin/sinBookDownload.ts:11 | `BookDownloadState` | D1/D5 | 4 | high | extFiles=0；本文件 3 处 | 去 `export` |
| src/pages/sin/sinBookDownload.ts:15 | `BookDownloadOutcome` | D1/D5 | 2 | high | extFiles=0；本文件 3 处 | 去 `export` |
| src/pages/sin/SinCastPanel.tsx:15 | `SinCastPanelProps` | D1/D5 | 12 | high | extFiles=0；本文件用于组件签名 | 去 `export` |
| src/pages/sin/SinCastPicker.tsx:13 | `SinCastPickerProps` | D1/D5 | 10 | high | extFiles=0；本文件用于组件签名 | 去 `export` |
| src/pages/sin/SinIllustration.tsx:32 | `SinIllustrationProps` | D1/D5 | 10 | high | extFiles=0；本文件用于组件签名 | 去 `export` |
| src/pages/sin/sinPanelState.ts:12 | `SIN_PANEL_MIN_WIDTH` | D1 | 1 | high | extFiles=0；本文件 2 处 | 去 `export` |
| src/pages/sin/sinPanelState.ts:13 | `SIN_PANEL_MAX_WIDTH` | D1 | 1 | high | extFiles=0；本文件 2 处 | 去 `export` |
| src/pages/sin/sinPanelState.ts:43 | `defaultSinPanelOpen` | D1/D5 | 4 | high | extFiles=0；本文件 :52 使用 | 去 `export` |
| src/pages/sin/SinProcessCard.tsx:16 | `SinProcessCardProps` | D1/D5 | 8 | high | extFiles=0；本文件用于组件签名 | 去 `export` |
| src/pages/sin/SinSidePanel.tsx:70 | `SinSidePanelProps` | D1/D5 | 12 | high | extFiles=0；本文件用于组件签名 | 去 `export` |
| src/pages/sin/sinToolMeta.ts:38 | `parseToolArgs` | D1/D5 | 12 | high | extFiles=0；本文件 :64 使用 | 去 `export` |
| src/pages/sin/StoryStream.tsx:15 | `StoryStreamProps` | D1/D5 | 10 | high | extFiles=0；本文件用于组件签名 | 去 `export` |
| src/pages/sin/useSinCast.ts:48 | `UseSinCastResult` | D1/D5 | 30 | high | extFiles=0；本文件为 hook 返回类型 | 去 `export` |
| src/pages/sin/useSinNotes.ts:18 | `SIN_CONFLICT_PREFIX` | D1 | 1 | high | extFiles=0；本文件内使用 | 去 `export` |
| src/pages/sin/useSinNotes.ts:26 | `UseSinNotesResult` | D1/D5 | 12 | high | extFiles=0；本文件为 hook 返回类型 | 去 `export` |
| src/pages/sin/useSinStory.ts:52 | `SIN_MESSAGES_PAGE_SIZE` | D1 | 1 | high | extFiles=0；本文件 3 处 | 去 `export` |
| src/pages/sin/useSinStory.ts:88 | `UseSinStoryResult` | D1/D5 | 35 | high | extFiles=0；本文件为 hook 返回类型 | 去 `export` |
| src/types/index.ts:44 | `CareerRefData` | D1/D5 | 5 | high | extFiles=0；仅 `CharacterData.sub_careers` 引用（同文件 :41） | 去 `export` |
| src/types/index.ts:79 | `ConsistencyIssueData` | D1/D5 | 4 | high | extFiles=0；仅 `ConsistencyReportData.issues`（同文件 :87） | 去 `export` |
| src/types/index.ts:116 | `ForeshadowUrgencyData` | D1/D5 | 6 | high | extFiles=0；仅 `ForeshadowItemData.urgency`（同文件 :111） | 去 `export` |
| src/types/index.ts:144 | `ConsistencyIssueSource` | D1/D5 | 1 | high | extFiles=0；仅 `ConsistencyDeepIssue.source`（同文件 :148） | 去 `export` |
| src/utils/characterStatus.ts:11 | `CharacterStatus` | D1/D5 | 1 | high | extFiles=0；本文件 4 处 | 去 `export` |
| src/utils/emotionColors.ts:4 | `EMOTION_COLORS` | D1/D4 | 7 | high | extFiles=0；本文件 :14 读取；与 `pages/chat/constants.ts:22 EMO_COLORS` 重复 | 去 `export` + 与 `EMO_COLORS` 合并（见发现 6） |
| src/utils/loraFilter.ts:9 | `LoraFamily` | D1/D5 | 1 | high | extFiles=0；本文件 2 处 | 去 `export` |
| src/utils/mermaidPng.ts:25 | `RenderedPng` | D1/D5 | 6 | high | extFiles=0；本文件 `renderMermaidToPng` 返回类型 | 去 `export` |
| src/utils/readingProgress.ts:1 | `ReadingProgress` | D1/D5 | 6 | high | extFiles=0；本文件 3 处 | 去 `export` |
| src/utils/readingSettings.ts:22 | `READING_SETTINGS_KEY` | D1 | 1 | high | extFiles=0；本文件 2 处 | 去 `export` |

### F-75 ~ F-110：其他类型

| 文件:行 | 符号 | 类型 | LOC | 置信度 | 证据 | 建议 |
|---|---|---|---|---|---|---|
| src/types/wails.d.ts:12 | `TTSConfig`,`TTSStatus`（import） | D1 | 2 | high | eslint 2 条 `no-unused-vars` warning（唯一 2 条）；全仓 grep 各仅命中定义 + 此 import | 删 import 名 + 删 `types/index.ts:189-200` 两个 interface |
| src/types/index.ts:189 | `TTSConfig` | D1 | 6 | high | 全仓 0 处消费（见上） | 删除（需确认后端无前端 TTS 配置消费） |
| src/types/index.ts:197 | `TTSStatus` | D1 | 4 | high | 全仓 0 处消费 | 删除 |
| src/types/wails.d.ts:126 | `PlotBranch` | D1 | 9 | high | 全仓仅本文件 2 处（注释 + 定义） | 删除 |
| src/types/wails.d.ts:137 | `LorebookEntry` | D1 | 6 | high | 全仓仅本文件 1 处（定义） | 删除 + 删 `:291 // ── Lorebook ──` 空段 |
| src/types/wails.d.ts:274 | `// ── 可视化 ──` | D3 | 2 | high | `:274` 与 `:291` 两个 section 注释下无任何方法/声明 | 删注释 |
| src/pages/chapter/errText.ts:5 | `errText`（14 份抄本） | D4 | 42 | high | 14 个文件逐字相同（列表见 Top-1），13 处为抄本 | 合并到 `src/utils/errText.ts` |
| frontend/tsconfig.app.json | `noUnusedLocals` / `noUnusedParameters` = false | D1 盲区 | 2 | high | `tsconfig.app.json` 显式关闭 → `tsc` 对未使用局部声明零告警，是本次"死代码稀少"假象的主因 | 先用 `tsc --noEmit --noUnusedLocals --noUnusedParameters` 做一次性诊断 |
| src/utils/emotionColors.ts:13 + src/pages/chat/emotions.ts:43 | `emotionColor`（同名不同语义） | D4/D8 | 8 | medium | `emotionColors.ts:13` 返回 hex（fallback `'#60a5fa'`）；`chat/emotions.ts:43` 返回 `string \| undefined`（从 `SPEAK_EMOTIONS` 查）。**语义不同，不能直接合并**，但同名易混 | 重命名其一（如 `chatEmotionColor`）或让后者复用前者 |
| src/utils/scroll.ts:3 | `isNearBottom` | D5 | 5 | high | 函数体 = `<` 比较；2 个消费点；`gaea/components/Transcript.tsx:25` 另有同名实现 | 内联删文件（含 `scroll.test.ts` 12 L） |
| src/pages/modelcenter/context.ts:60 + hooks/useBindState.ts:27 | `modelRoutes` 内联匿名类型 | D8 | 2 | high | 两处 `Record<string, { engine: string; model: string; source: string }>` 逐字重复 | 提为命名类型 `ModelRoute` |
| src/hooks/useVoiceChat.ts | 30 处 `catch (_) {}` / `.catch(() => {})` | D2 | 30 | high | `:124,151,164,176,182,191,200,205,206,275,280,305,378,424,444,450,456,501,507,511,533,539,551,555,556,557,561` | 仅对用户可感知项补错误提示，其余注明静默原因 |
| src/pages/modelcenter/hooks/useEngineState.ts | 9 处空 catch | D2 | 9 | high | `:131,135,139,143,147,151,155,263,269` | 同上一行 |
| src/pages/CreatePage.tsx | 1403 行 / 组件跨度 1300+ | D6 | — | high | 文件行数 1403；`RailGroup`(:153) 之后主体超长 | 按 wizard step 拆 3~4 个子组件 |
| src/pages/CharacterPage.tsx:84 | `CharacterPage` | D6 | 1011 | high | 单函数 84-1094 行 | 拆 `useCharacterData` / `CharacterList` / `CharacterDrawer` |
| src/pages/WeixinPage.tsx:140 | `WeixinPage` | D6 | 711 | high | 单函数 140-850 行；文件内已有 6 个子组件（`:25,31,853,878,944`）可继续外提 | 外提 `ChannelDetail`/`PersonaPickerPanel` 到独立文件 |
| src/pages/SchedulePage.tsx:70 | `SchedulePage` | D6 | 696 | high | 单函数 70-765 行 | 拆三视图（横道图/单代号/双代号） |
| src/pages/modelcenter/EngineSection.tsx:10 | `EngineSection` | D6 | 516 | high | 单函数 10-525 行 | 拆表单/列表/端点切换 |
| src/pages/sin/useSinStory.ts:124 | `useSinStory` | D6 | 477 | high | 单 hook 477 行 | 抽 `toStoryView`/`toMessageView`/`findToolRow`/`nextKey` |
| src/pages/modelcenter/StatsSection.tsx:21-26 | `OverviewCache`/`overviewCache`/`SideCache`/`sideCache` | D7/D2 | 12 | medium | 模块级缓存（本文件 1 次出现），需人工确认失效策略 | 确认缓存失效路径，或改用 `useMemo` |
| src/lib/pollingGate.ts:12 + src/hooks/usePollingGate.ts:15 | `isPageVisible` / `usePollingGate` | D5 | 5 | medium | `isPageVisible` 是 1 行；`usePollingGate` 是 4 行 `useSyncExternalStore` 包装。**被 12+ 处使用**，属合理分层，**不建议动** | 保留 |
| src/utils/text.ts:2 | `countTextChars` | D5 | 1 | low | 1 行包装 `Array.from(text).length`，但被 7 个文件使用（含 4 个范围外） | 保留（AST 上有可读性价值） |

---

## 重复小工具清单（含合并建议）

| # | 能力 | 副本（文件:行） | 副本数 | 是否逐字相同 | 合并建议 |
|---|---|---|---|---|---|
| 1 | `errText(err, fallback)` | `pages/chapter/errText.ts:5`、`pages/chapter/ChapterIllustration.tsx:33`、`pages/modelcenter/hooks/useAttributionState.ts:21`、`hooks/useBindState.ts:18`、`hooks/useEngineState.ts:28`、`hooks/useVoiceState.ts:25`、`pages/modelcenter/BenchmarkSection.tsx:9`、`pages/sin/SinBookSourcePanel.tsx:23`、`pages/sin/useSinCast.ts:27`、`pages/sin/useSinNotes.ts:22`、`pages/sin/useSinStory.ts:23`、`pages/CharacterLibraryPage.tsx:28`、`pages/CharacterPage.tsx:20`、`hooks/useImageGenConfig.ts:21` | **14** | ✅ 完全逐字相同 | 提升为 `src/utils/errText.ts`，14 处改 import。**最高优先** |
| 2 | 情绪→色 9 色表 | `utils/emotionColors.ts:4 EMOTION_COLORS`、`pages/chat/constants.ts:22 EMO_COLORS` | 2 | ✅ 键值全等 | 合并到 `emotionColors.ts`，`constants.ts` 改 re-export |
| 3 | `emotionColor(x)` | `utils/emotionColors.ts:13`（→hex）、`pages/chat/emotions.ts:43`（→`string\|undefined`） | 2 | ❌ 语义不同 | **不要合并**；重命名避免同名混淆 |
| 4 | `fmtTime` | `pages/modelcenter/BenchmarkSection.tsx:63`（`replace('T',' ').slice(0,16)`）、`pages/sin/SinBookSourcePanel.tsx:33`（`Date.parse` + 本地化） | 2 | ❌ 实现不同（一个纯字符串切、一个 Date.parse） | 统一到 `utils/format.ts` 的 `fmtDateTime`，输出格式需先对齐（**需人工确认**） |
| 5 | `fmtSize` | `pages/modelcenter/HerdsmanCatalogSection.tsx:40`（MB/GB/TB，返回 `''` 兜底）、`pages/sin/SinBookSourcePanel.tsx:27`（B/KB/MB） | 2 | ❌ 单位域不同 | 统一为 `fmtBytes(n)`（B/KB/MB/GB/TB 全阶梯），两处改用；注意 `HerdsmanCatalogSection` 的"0 返回空串"语义要保留 |
| 6 | `popupContainer = () => document.body` | `pages/modelcenter/BindSection.tsx:32`、`pages/modelcenter/ImageSection.tsx:9` | 2 | ✅ | 提到 `pages/modelcenter/ui.tsx` 导出一次 |
| 7 | `errText` 无 fallback 变体 | `hooks/useChatTopics.ts:90`、`pages/chat/utils.ts:88` | 2 | ✅ 语义相同 | 用统一版 `errText(err, String(err))` |
| 8 | localStorage JSON 读写 | `pages/CharacterPage.tsx:63/74`、`utils/readingAnnotations.ts:28/43`、`utils/readingBookmarks.ts:19/34`、`utils/readingProgress.ts:9/21`、`utils/readingSettings.ts:51/79`、`utils/bookSearchHistory.ts:10/25`、`pages/sin/sinPanelState.ts:26/34`、`pages/modelcenter/modelPrefs.ts:8/18`、`data/imageTemplates.ts:557/569`、`pages/NovelPage.tsx:48/58/97/103/110`、`pages/NovelSettingPage.tsx:35/187`、`pages/ChatPage.tsx:80/85/92/103`、`pages/CharacterLibraryPage.tsx:37/70` | **13 处文件** | 结构相同 | 新增 `utils/localStore.ts`（`readJSON`/`writeJSON`/`readFlag`/`writeFlag`），**分批替换**，先从已有测试的 `utils/` 四个开始 |
| 9 | 数字/百分比格式化 | `utils/…` 无统一模块：`modelcenter/utils.tsx:94 formatCtx`、`:102 fmtPriceNum`、`:110 formatPrice`、`:383 fmtCompact`、`:401 fmtCost`、`modelcenter/resource.ts:84 fmtGB`、`modelcenter/RetrievalEvalSection.tsx:21 fmtPct`、`modelcenter/charts.tsx:95 fmtAxis`、`sin/sinToolMeta.ts:53 fmtToolElapsed`、`HerdsmanCatalogSection.tsx:56 fmtMs`/`:62 fmtTps`、`CostLibraryPage.tsx:182 fmt`(Intl) | **12+** | 部分 | 新建 `pages/modelcenter/format.ts` 或 `utils/format.ts`，归拢 `fmtPct`/`fmtGB`/`fmtMs`/`fmtCompact`；`Intl.NumberFormat` 实例应模块级复用（`CostLibraryPage.tsx:182` 已是好范例） |
| 10 | "接近底部"判断 | `utils/scroll.ts:3`、`gaea/components/Transcript.tsx:25` | 2 | ❌ 参数不同 | 统一签名或内联 |
| 11 | `nowStr()` / `msgSeq` | `pages/chat/utils.ts:11 nextMsgKey`/`:12 nowStr`、`pages/sin/useSinStory.ts:28-ish msgSeq`/`:34 nextKey` | 2 组 | 结构相同 | 提到 `utils/time.ts` / `utils/id.ts`（`nextKey` 已是同构实现） |

---

## 屎山清单（结构性重构建议）

按风险从低到高：

1. **`src/pages/modelcenter/` 的 "每 hook 一份 errText + 一组空 catch" 样板**（`hooks/use{Attribution,Bind,Engine,Image,Stats,Voice}State.ts` 共 6 个文件，合计 1,197 行）。
   症状：6 个 hook 开头都是同一段 `errText`，每个 async 动作都是 `try {...} catch (err: unknown) { message.error(errText(err, 'XX失败')) }`。
   建议：先合并 `errText`（发现 1），再把"加载态 + 错误态 + message.error"收敛成一个 `useAsyncAction()`；**不要**把 6 个 hook 合成 1 个（它们对应独立后端门面）。

2. **`src/pages/CharacterPage.tsx` — 1012 行文件 / 1011 行单函数**。
   症状：单函数包含"项目内角色列表 + 全局角色库抽屉 + 职业编辑 + 关系生成 + 剧照 + 合并 + 回写预览"7 个职责，`useState`/`useEffect`/`useCallback` 密集（`:20-501` 全是 handler）。
   建议：拆 `useCharacters(projectPath)`（数据/加载/错误）+ `useCharacterCareers()`（职业）+ `CharacterDrawer`（抽屉 UI，已独立 CSS `character-page.css`）。目标：主组件 ≤300 行。

3. **`src/pages/CreatePage.tsx` — 1403 行**，其中 `:104 EntityGraphSvg`、`:153 RailGroup` 之后是超大 JSX + wizard 步骤。
   建议：把 6 个 step 各提一个文件（`create/steps/*.tsx`），把 `EntityGraphSvg` 提到 `create/EntityGraphSvg.tsx`（它已是独立可测单元）。

4. **`src/pages/WeixinPage.tsx` — 987 行 / 711 行单函数**，文件内已自建 6 个子组件（`WxPortraitAvatar:25`、`WxPortraitImg:31`、`PersonaRow:853`、`PersonaPickerPanel:878`、`ChannelDetail:944`）。
   建议：把已有子组件外提到 `pages/weixin/*.tsx`（零逻辑改动，纯剪切/粘贴 + import），主组件自然降到 ~400 行。

5. **`src/pages/SchedulePage.tsx` — 746 行 / 696 行单函数**，三视图（横道图 / 单代号 / 双代号）共处一函数。
   建议：三视图各一文件，共享 `schedule/` 下已有工具。

6. **`src/pages/sin/useSinStory.ts` — 601 行 / hook 477 行**，含 4 个私有 helper + `errText`。
   建议：helper 外提（发现 14）。

7. **`src/pages/modelcenter/modelcenter.css` — 1498 行**（范围内的最大 CSS）+ `sin/sin.css` 798 行 + `weixin-page.css` 648 行。CSS 未按组件切分，任一 Section 改动都需全文件搜索类名。
   建议：配合上面对 Section 文件的拆分，把 CSS 就近切到各 Section 模块（或至少用注释分节）。

8. **无巨型 `switch`**：范围内最大的 switch 是 `modelcenter/utils.tsx` 的 `kindOf`/`featureStateMeta`（查表式，非 switch）。此项**无问题**。

---

## 不建议动的地方（说明为什么）

| 位置 | 为什么不要动 |
|---|---|
| `src/pages/{GaeaPage,HomePage,NovelPage,NovelSettingPage,ChapterPage,CharacterPage}.tsx` | 全部可达：`HomePage`/`NovelSettingPage`/`CharacterPage`/`CreatePage`/`ChapterPage` 由 `NovelPage.tsx:37-41` 的 `React.lazy` 挂载；`GaeaPage` 由 `main.tsx:19` + `boards/manifests.ts:99` 挂载。改名前请先看 `boards/manifests.ts`。 |
| `src/pages/chat/{constants,types,utils}.ts`、`src/data/templates/*.ts`、`src/data/herdsmanTemplates.ts` | 子目录/分片文件，**由兄弟文件聚合使用**，grep 单个文件名得不到引用。删任一分片都会连带打包结果变化。 |
| `src/lib/{wailsApp,pollingGate}.ts`、`src/utils/{zIndex,uiStyles,emotionColors}.ts` | 范围内 0 引用，但**范围外（`components/`、`stores/`、`gaea/`）在用**。这些是"范围内定义、范围外消费"的共享层，不能按"范围内无人用"删除。 |
| `src/types/wails.d.ts` 的 `AppAPI` / `AppFacade` / `RuntimeAPI` | 提供 `window.go.app.App` 的全局类型，**无 import 也生效**（`declare global`）。删任何一个方法签名会让 `wailsjs/` 之外的直调点类型失衡。只删本报告点名的 3 个死类型（`LorebookEntry`/`PlotBranch`/`TTSConfig`+`TTSStatus`）。 |
| `src/test/setup.ts` | 通过 `vite.config.ts` 的 `setupFiles` 注册，**无 TS import**，grep 不到引用属正常。 |
| `src/events.ts` 的 channel 辅助函数（`sinStreamChannel:84`、`bookImportProgressChannel:89`、`sinBooksourceChannel:94`、`subscribeForSpace:122`、`emitFrontendEvent:137`） | 范围内 0 引用，但被 `pages/sin/*` 与范围外组件通过字符串事件名间接使用（Wails 事件通道）。**判死前必须跑运行时**，我标为 `confidence:low` 不做删除建议。 |
| `src/utils/outline.ts`、`src/utils/text.ts` | 被多文件（含范围外）使用，函数虽短但有明确语义（Unicode 码点计数、递归叶子提取）。**保留**。 |
| `src/pages/sin/sinThumbCache.ts:35 __resetSinThumbCacheForTest`、`src/pages/sin/useSinStory.ts:21 SIN_STREAM_SILENCE_TIMEOUT_MS` | 只被 `*.test.tsx` 使用——**测试引用算使用**，必须保留 `export`。 |
| `src/components/`、`src/gaea/`、`src/stores/`、`src/boards/`、`src/genui/` 等被排除目录 | 本次范围外。**它们是我判定"范围内符号仍活着"的主要依据来源**，若它们自身有死代码，需另开一次审计。 |
| `src/pages/chat/emotions.ts:43 emotionColor` 与 `src/utils/emotionColors.ts:13 emotionColor` | 同名但语义不同（一个 `string`、一个 `string \| undefined`）。**不要合并函数**，只做重命名或提升复用。 |
| `knip.json` | 项目已配置 knip（`frontend/knip.json`），但 `node_modules` 中没有安装 knip（`Test-Path node_modules\knip` → False）。**不要为了跑它去 `pnpm install`**（任务禁止）。 |

---

## 交付给主控的 5 条"先动手"（风险最低）

1. **删 4 个未使用的 `React` 默认 import**（`src/pages/GaeaPage.tsx:1`、`src/pages/modelcenter/BenchmarkSection.tsx:1`、`src/pages/modelcenter/RetrievalEvalSection.tsx:1`、`src/pages/sin/SinBookSourcePanel.test.tsx:7`）—— 编译器（`tsc --noUnusedLocals`，TS6133）直接背书，零风险。
2. **删 `src/types/index.ts:189-200` 的 `TTSConfig`/`TTSStatus` + `src/types/wails.d.ts:12` 的两个 import 名**（15 行，eslint 已经背书：全范围唯一 2 条 unused-vars warning）。
3. **删 `src/types/wails.d.ts:126 PlotBranch` 与 `:137 LorebookEntry`**（15 行，全仓 0 引用，且 `AppAPI` 无对应方法，Lorebook 段只剩空注释）。
4. **74 处去掉多余 `export` 关键字**（全量表 F-01~F-74，机械操作，零行为变化；用 `tsc -b --noEmit` + 全量 vitest 验证）。
5. **合并 14 份 `errText`**（42 行 → 1 个共享模块；纯 import 替换，有 `useEngineState.test.tsx:138` 等既有测试覆盖）。
6. （可选第 6 条，收益最大但需回归）**合并 `EMO_COLORS` → `EMOTION_COLORS`**（9 行 + 消除"改一处漏一处"风险；被 `EmotionStarMap`/`WhisperEmotionPanel`/`EmotionSpeakSelector`/`ChatPage` 覆盖）。

---

## 方法与可复现命令

```powershell
# 类型检查（基线：exit 0，无输出）
& node.exe pnpm.mjs exec tsc -b --noEmit          # workdir = C:\AI\wubigrok\frontend

# lint（结果：0 error / 2 warning，均在 src/types/wails.d.ts:12）
& node.exe pnpm.mjs exec eslint src/pages src/hooks src/utils src/lib src/types src/data src/test src/App.tsx src/main.tsx src/events.ts
```

- import 图构建：解析范围内 216 个文件的 `from '...'` / `import('...')` / 裸 `import '...'`，按 TS 解析规则（`.ts/.tsx/index.ts/index.tsx`、`@/` 别名）落到真实文件，从 `src/main.tsx` + `src/test/setup.ts` + 全部 `*.test.ts(x)` 做可达性 BFS：**可达 183/183（范围内）**。
- 符号级核对：对全部范围内 `export` 声明（`function/class/const/let/var/interface/type/enum` + `export {}` specifier）计算"全仓唯一出现文件集合"与"导入方集合"，`extFiles=0` 判定为多余导出；对同名跨文件（如 `emotionColor`）逐个人工读源码确认语义是否相同。
- 动态引用核对：对每个候选额外 grep 文件名（无扩展名）、组件名、字符串路径、`lazy(`、`import.meta.glob`、`registerPage(`、`boards/manifests.ts` 的 `page:` 键。范围内 `import.meta.glob` 命中数为 0。
- 未执行：`pnpm install`、dev server、任何写操作。
