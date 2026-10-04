# Gaea 第三轮全仓审计 · 收敛报告（2026-10-04）

> 方法：11 个并行只读审计子代理分域扫描（Go 6 域 + 前端 4 域）→ 主控逐条独立复核 → 只实施**双证据**项。
> 基线（改动前）：`go build` / `go vet` / `golangci-lint v2.14.0`（errcheck/govet/ineffassign/staticcheck/unused/misspell）全部 **0 issues**；前端 vitest **435 文件 / 3760 用例全绿**。
> 分域原始报告：本目录 `reports/`（10 份，共 ~450KB，含 500+ 条带 `file:line` 证据的发现）。

---

## 0. 最重要的结论：这不是「屎山重灾」，而是「清零后的漏网项」

前两轮（`docs/code-audit-2026-10-02/`，512 条）已把类型级死代码清空：
- `golangci-lint` 的 `unused`/`staticcheck` 在全仓 1825 个 `.go` 上**零告警**——即**包内**死代码已为 0；
- 前端 import 图反查（入口 `main.tsx` + `test/setup.ts` + 全部 `*.test.ts(x)`，1199 个文件）：**真·零引用文件只有 1 个**；
- 整包死代码：`internal/` 下 **0 个**（58 + 48 个包全部有外部 import 方）；
- 注释掉的代码块（D3）：**全域接近 0**（Go 侧 0 处、前端 genui 0 处、agent/tool 0 处）——不要安排「删注释代码」批次；
- TODO/FIXME：全仓 **9 条**。

因此本轮的价值集中在三类：**① 跨包语义级死代码（lint 看不见）② 逐字重复实现 ③ 巨型函数与守卫缺口**。

---

## 1. 已实施改动（净 −1240 行，66 文件）

| 批次 | 内容 | 净删 |
|---|---|---|
| 前端 · 未使用符号 | `tsc --noUnusedLocals/--noUnusedParameters` 权威清单 **38 条** TS6133（26 个多余 `React` 导入、`UserOutlined`、`waitFor`、`EntryPicker`/`slug`/`message`/`query`/`phase`/`chart` 等） | −30 行 + 29 文件 |
| 前端 · 死文件 | `gaea/components/TrendChart.tsx`（0 import）、`SubagentsPanel.tsx` + `.test.tsx`（唯一 import 是自测，已被 `TasksWorkbench` 取代） | −671 行 |
| 前端 · 死本地化键 | **26 键 × 3 语言 = 78 行**（官方 `scripts/slim-locale-deadkeys.mjs` 复核：`dead keys: 0`） | −78 行 |
| 前端 · 死类型 | `TTSConfig`/`TTSStatus`、`PlotBranch`/`LorebookEntry`（全仓仅定义处命中） | −30 行 |
| 前端 · 死 API | `BlockApi.isSecret`（实现+接口+Provider 注入，零消费者；`secretFields` 仍被持久化过滤消费故保留） | −5 行 |
| 前端 · 死分支 | `seriesColor` 未用参数 `chart`（含 3 处调用点）、`groupW` 两分支字面相同的三元、`BranchWizardModal.prevChapter`（含 CreatePage 调用点） | −10 行 |
| **前端 · 守卫** | **`tsconfig.app.json`: `noUnusedLocals`/`noUnusedParameters` 由 `false` → `true`** —— 这是本轮最有价值的防复发改动 | — |
| Go · 零调用方法 | 12 个（`controller.go` TCCAStats/TCCAReport/HookRunner、`controller_mcp.go` ConfiguredMCPNames/DisconnectedMCPNames、`controller_memory.go` SessionRemember/SessionFacts、`refs.go` HasRefs、`context/manager.go` ActiveTools、`cache/compiler.go` WithInstructions、`cache/runtime.go` DetectConflict/MergeChildEdits） | −149 行 |
| Go · agent | `tool_precheck.go` 的 `case "delete_range"` 死分支 + `precheckDeleteRange`（36 行）+ 6 个对应测试函数（98 行）——`delete_range` 从来不是内建工具名 | −135 行 |
| Go · app | `FileIndexStatus`、`ctxForeshadowLineLen`、`imageTierFreeFallback`/`imageTierRemoteHeavy`、`SetCostStageStoreForTest`/`ResetCostStageStoreForTest`（均全仓零引用） | −22 行 |
| Go · memory | `Set.EpisodicMatches`（0 调用；注释称「用于 few-shot 注入」，无任何接线） | −44 行 |
| **Go · 重复收敛** | `round4` 两份逐字抄本 → 新增 `cost.Round4` 单一源（对齐既有 `cost.Round2` 的 GA6-05 收敛先例） | −6 行 |
| 工程 | `wails.json` 由 `npm install/build/dev` → `pnpm install --frozen-lockfile / run`（CI 已是 pnpm，npm 无 lockfile 结构性跑不通） | 3 行 |
| 前端 · 死 mock | `ChatPage.test.tsx:90` 对**已退役幽灵模块** `../../src/wailsjsCompat`（v4.174 退役、文件不存在）的整段 `vi.mock` | −21 行 |
| **前端 · CSS 死类** | **−676 行**（11 个 CSS 文件）：`msg--*`/`msg__*`、`ds-*` 类选择器、`btn-primary/secondary`、`badge--*`、`context-menu*` 全族、`drawer*`、`mem-*`、`launcher-*`、`neon-*`、`void-card`、`bento-grid`、`hub-*`、`chat-trust-*`、`chat-drawer*` 等。**两路独立扫描交叉验证**（我的整词匹配扫描 ∩ 子代理逐类复核），排除 `--ds-*` 变量、`hljs-*`/`tok-*`/`ant-*` 库生成类、`is-*`/`v3-*`/`gui-*`/`sched-*` 动态拼接族 | −676 行 |
| 磁盘 | 删除 17 个陈旧构建日志/产物（`vitest-*.log` ×11、`vite-dev*.log` ×3、含非法字符的 tmpci 日志、`package.json.md5`、`vitest-report.json`）**4.77 MB**，全部已 `.gitignore` 覆盖、零脚本引用 | 4.77 MB |

**合计：78 个文件，+54 / −1991，净删 1937 行（前端 1573 / internal/gaea 346 / internal/app 18）+ 4.77 MB 磁盘垃圾。**

### 验证
- `go build ./...` ✅ / `go vet ./...` ✅ / `golangci-lint` **0 issues** ✅
- `go test ./... -count=1`：**唯一失败** `TestKillRunningKillsRegisteredProcess`（`internal/gaea/tasks`）→ 单独复跑 **2/2 全绿**，且该包 `git status` 零改动 ⇒ **环境性 flaky，非回归**
- 前端 `tsc -b`（新 strict 口径）✅ / `eslint` 0 error ✅ / `pnpm build` ✅ / vitest 全量复跑（见 §4）
- 门禁：`check-docs.mjs` OK（孤儿 0 / 悬空 0 / 指令预算内）✅ · `check-bindings-drift.ps1` OK（635 方法一致、在册死名单 0）✅ · 版本三处一致 4.455.0 ✅

---

## 2. 「看着像死代码、实则活」——本轮拦下的误报（重要）

审计子代理的初判有相当比例误报，**逐条复核后才动手**，以下是典型：

| 候选 | 误报原因 |
|---|---|
| `memory.GraphNeighbors`（83 行） | 设计文档 `docs/gaea-memory-graph-51-design-2026-09.md:13` 与 `releases/v4.210.0.md` 明列其为**已交付 API**（v4.210.0「记忆语义图谱」），只是「未出绑定」 |
| `provider.SanitizeToolPairing` + 5 helper（120 行） | 生产零调用但**有 3 个测试文件 14 处断言**——唯一写好的 tool 配对护栏，**该接线不该删** |
| `SubagentStore.CleanupStaleRunning`（34 行） | 启动清理 stale running 的**安全机制**；无人调用是**功能缺失**（崩溃后 running 记录永久残留），删了等于删安全网 |
| `BudgetStatus`/`SessionInfo`/`BranchInfo` 类型别名 | `agent/session_alias.go` 等是**包拆分兼容层的既定设计**（历史评审文档明写「兼容层保留」） |
| `office/standard` `hasSalutation`/`fixHint`、`docmd` `isPDFKeyword`/`isXrefEntry`、`docxedit` `unwrapInner`/`spanCoversLink` 等 | 子代理报「零引用」，实则**同文件内均有调用**（`rg` 大正则静默丢匹配所致） |
| `config.GetFeatureModel` 等 | token 扫描显示零外部引用，实为 `analysis.go:314`/`bindings_core.go:39`/`voice_handler.go:147` 在调用 |
| `genui/guard.ts:521` checkbox/switch「丢 label」 | 报为 P1 真 bug，实为**误报**：`guard.ts:524` 确实写入了 `label` |
| `frontend/knip.json` entry/ignore、`types/wails.d.ts`、`locales/zh-TW.ts` | 前者是权威白名单；`wails.d.ts` 是 live 全局声明；`zh-TW.ts` 走 `await import()` 动态加载 |
| `hide_window_windows.go` 的 15 个零引用字段 | Win32 内存布局，**绝不能删** |
| `.gui-*` CSS 死类 13 个 | 模板串动态生成（`gui-tone-${tone}`）的假阳性 |

## 3. 未实施、但建议处理的清单

### 3.1 结构性（收益最大，需单独批次 + 全量回归）
| 项 | 位置 | 说明 |
|---|---|---|
| `Load()` 单函数 **449 行** | `internal/config/config.go:27`（占该文件 94%） | 按域拆 `loadAI/loadVoice/loadNovel/loadUI/loadPaths`，`Load()` 只做装配 |
| `applyOne` 巨型 switch **253 行** | `internal/office/xlsxedit/xlsxedit.go:330` | 照 `internal/schedule/ops.go:110` 的 `opHandlers` 注册表范式拆为 12 个 `apply<Op>` |
| `rebuildParagraph` 双份 **140/124 行** | `office/docxedit:470` ⇄ `office/pptxedit:380` | 6 行滑窗比对 **11 个块逐字相同**，可抽到已被两者 import 的 `office/ooxml` |
| `boot.Build` **559 行**（全仓最长） | `internal/gaea/boot/boot.go` | — |
| `client.go` 1160 行 / `CreatePage.tsx` 1403 行 | `internal/ai/client.go`、`frontend/src/pages/CreatePage.tsx` | 20 个 >300 行前端组件函数（`CharacterPage` 单函数 1011 行） |

### 3.2 重复实现收敛（逐组低风险）
| 组 | 份数 | 位置 |
|---|---|---|
| `errText()` | **14 份逐字相同** | `pages/chapter/errText.ts:5` + 13 处抄本（`pages/modelcenter/hooks/*`、`pages/sin/*`、`hooks/useImageGenConfig.ts` …）⇒ 上移到 `utils/errText.ts`，省 ~42 行 |
| `parseTagsJSON` | 2 份 | `gaea/cost/cost.go:660` ⇄ `gaea/knowledge/sqlite.go:180` |
| `sortedKeys` | 3 份（签名各异） | `app/gaea_usage_overview.go:177`、`contextview/imgrefs.go:130`、`novelcontext.go:998` |
| `hasHan` / `isCJStr` | 2 份 | `costimport.go:1075` ⇄ `knowledge/search.go:212` |
| `EMOTION_COLORS` / `EMO_COLORS` | 2 份 9 色全等 | `utils/emotionColors.ts:4` ⇄ `pages/chat/constants.ts:22` |
| localStorage 读写样板 | 14 个文件 | `gaea/lib/*Prefs.ts` 系列 |
| 拖拽改宽手势清理块 | 4 份逐字相同 | `usePreviewPanel.ts:174`、`useWorkspaceLayout.ts:80`、`ResizableDrawer.tsx:128`、`useSidebar.ts:46` |
| 测试 fixture/桩 | 6 + 4 + 4 + 3 份 | `schedule/*.test.tsx` 任务 fixture、`sin/*.test.tsx` 的 `vi.mock(bridge)`、`novel/create/*.test.tsx` 的 `stubRuntime()` |
| 4 份 `novelModelName`/`novelEngineName` | 4 | `chapter:212`、`character:716`、`outline:39`、`analysis:100` |

### 3.3 待产品拍板（不要单方面删）
- `agent/compact.go` `SummarizeFrom`/`SummarizeUpTo`：有 Wails 绑定但实现是**拒绝桩**（`gaea_ui.go:725` `errSummarizeUnsupported`），前端按钮可能仍可见。
- `tool.SuspendPrefix`/`ResumePrefix` + `suspended` 字段 + `Add` 守卫：生产 0 调用 ⇒ 「MCP 会话级禁用」**文档承诺 > 实现**；要么接线要么删。
- `agent/reasoning_language.go` 语言偏好 6 函数 + 2 字段：`withTurnPreferences` 因此**恒为 no-op**（每回合仍白跑一遍字符串判断）。
- `SetMemoryQueue` 生产 0 调用 ⇒ `memQueue` 恒 nil ⇒ `execute_one.go:158` 的 `WithQueue` 与 `recall_reminder.go`（48 行）**生产不可达**。
- 「幽灵工具名」`delete_range`/`delete_symbol`/`notebook_edit`/`glob`：从未注册，导致 `boot.go:903/912/915` 三次 `HideUnlessOnly` **永久 no-op**，且名字硬编码在 agent 的 6 处 switch 名单里（`compress.go:384`、`tool_coherence.go:84`、`execute_one.go:559`、`agent_run.go:363`、`batch_executor.go:237`、`compact_summary.go:46`）。
- `internal/types/interfaces.go` 22 个导出类型（模板引擎/上下文装配抽象层）：全仓零引用，职责实际由 `prompt` + `novelcontext` 承担——疑似早期设计残留（~240 行）。
- `provider/retry.go` 整文件（146 行、9 符号）：只有 `retry_test.go` 引用；域内另有 4 套手写退避（`ai/client.go` 数组、`webfetch.go:322`、`websearch_engine.go:408`、`fileutil/atomic.go:76`、`tasks.go:1184`）与 3 份 SSRF 判据复制（其中 weixin 副本漏了 IN2-04 修复）。

### 3.4 补测优先于删除
- `internal/schedule/ops.go` 12 个 `applyXxx`（≈560 行）经 `opHandlers` 表分派是**活代码**，但**零调用方覆盖**（现有测试只校验键集）⇒ 建议补 `TestApplyOpsEachKind` 表驱动。

### 3.5 前端守卫缺口（本轮已修一半）
- 已修：`tsconfig.app.json` 开 `noUnusedLocals`/`noUnusedParameters`。
- 仍缺：`tsconfig.node.json` 已开而 app 侧此前未开（现补齐）；eslint 对未用变量只降 `warn`；`knip` 未安装（未使用导出/文件只能靠人工扫描）；`knip.json` 的 `ignore` 列表包含 `genui/index.ts` 等，属于**主动对检测器闭眼**。

---

## 4. 复现方式

```powershell
# Go
go build ./... ; go vet ./...
& "$env:USERPROFILE\go\bin\golangci-lint.exe" run --timeout=10m
go test ./... -count=1

# 前端
cd frontend
pnpm exec tsc -b --noEmit --force        # 现已含 noUnusedLocals/Parameters
pnpm run lint ; pnpm run build ; pnpm run test

# 门禁
node scripts/check-docs.mjs
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/check-bindings-drift.ps1
node scripts/slim-locale-deadkeys.mjs    # 期望 dead keys: 0
```

审计辅助脚本（只读、可复现）保留在 `.audit/`（本机 scratch，gitignore 不入库）：`dup-scan.mjs`（前端重复块 + import 可达性）、`fix-unused-imports.mjs`（TS6133 批量清理，dry-run 默认）、`fix-locale-deadkeys.mjs`、`fix-precheck-tests.mjs`。
