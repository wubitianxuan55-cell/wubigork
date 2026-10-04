# 前端审计 · genui + 工程配置/死代码工具链

**审计范围**：`frontend/src/genui/`（19 文件 / 129,514 B）、前端工程配置与死代码工具链
（`package.json`、`knip.json`、`eslint.config.js`、`eslint-rules/`、`tsconfig*.json`、`vite.config.ts`、
`index.html`、`pnpm-workspace.yaml`）、仓库根 `wails.json` / `.golangci.yml` / `go.mod`（依赖层面）、
前端根磁盘垃圾文件评估。**只读审计，未修改/删除/新建除本报告与 `.audit/` 下临时脚本外的任何文件。**

## 审计时点与工作树状态（重要）

| 项 | 值 |
|---|---|
| 审计时点 | 2026-10-04 |
| `frontend/src/genui/renderNode.tsx` | **mtime 2026-10-04 9:45:30，git 状态 ` M`（会话外有并发编辑）** |
| 该文件行数/字节 | 1098 行 / 36,279 B（Node `readFileSync` 口径） |
| 其余 genui 文件 | 最新 mtime 2026-10-03，git 干净 |
| `tsconfig.app.json` | mtime 2026-10-04 9:46:07，git ` M`（同批并发编辑） |

> ⚠️ **并发改动告警**：审计过程中 `renderNode.tsx` 被会话外进程改动了一次
> （`seriesColor(chart, s, i)` → `seriesColor(s, i)`，3 处调用同步更新，−1 参数）。
> 本报告所有 `renderNode.tsx` 行号以**上述 1098 行版本**为准，`LastWriteTime` 为审计时点；
> 该文件正在被编辑，行号可能漂移。另 `git status` 显示 `frontend/src/gaea/components/TrendChart.tsx`
> 已由该并发会话 `D`（staged 删除）——见下节。

**证据工具**（均只读，脚本落 `C:\AI\wubigrok\.audit\`）：
`reach-scan2.mjs`（1199 文件导入图）、`dep-scan.mjs`（依赖 importers）、`export-scan.mjs`（导出外部引用）、
`css-scan.mjs`（genui CSS 选择器消费面）；`tsc -p tsconfig.app.json --noEmit` 与
`eslint src/genui`（JSON 输出）均为本机实跑。

---

## 零被 import 文件复核结论（TrendChart.tsx 判定 + 全前端零引用清单）

### TrendChart.tsx 判定：**确认零引用（dead code），判定成立，confidence: high**

四路独立证据（`C:\AI\wubigrok` 全仓，排除 `node_modules/`、`clones/`、`dist/`）：

| # | 检查 | 命令 / 模式 | 结果 |
|---|---|---|---|
| 1 | 符号级 grep | 全仓 `TrendChart` | 21 命中，**无一处是 `./TrendChart` 形式的 import**：全部为 ① 前缀同名组件 `RequestsTrendChart`/`TokenTrendChart`（`pages/modelcenter/charts.tsx` 内联实现）② `ContextTrendChart`（`ContextView.tsx` 内联实现）③ docs/releases 文字提及 |
| 2 | 路径级导入图 | `reach-scan2.mjs`：解析 1199 个 ts/tsx/css 的静态 import / `export … from` / 动态 `import()` / `require()` / `import.meta.glob` / CSS `@import`，3,628 条解析边 | `gaea/components/TrendChart.tsx` 为**唯一**「零入边且非 knip entry」文件 |
| 3 | 动态引用 | 全 `frontend/src` grep `import.meta.glob` | **0 命中**（无 glob 动态装载可藏身） |
| 4 | 字符串路径 | `TrendChart` 无出现在字符串/template 中（21 命中已逐条目检） | 无 |

> **交叉印证**：并发会话已把该文件 `git rm --cached` 式暂存删除（`git status` 显示 `D frontend/src/gaea/components/TrendChart.tsx`），
> 与本判定方向一致。**最终删除动作已由他方完成，本审计不再建议重复处置。**
> LOC≈59（含 4 行文件头注释；`TrendPoint` 接口 5 行 + `TrendChart` 组件 49 行）。

### 全前端零引用清单

| 分类 | 数量 | 明细 |
|---|---|---|
| `frontend/src` 文件总数（ts/tsx/css） | **1,199** | 与既有独立扫描一致 |
| knip entry 且零入边（**不算死码**） | 437 | `**/*.test.*` 测试文件 434（含 `gaea/lib/mock/contract.test.ts`）+ `main.tsx` + `test/setup.ts`（= 438 零入边总数 − 1 条真零引用） |
| **零入边且非 entry（真零引用）** | **1** | `gaea/components/TrendChart.tsx` |
| 零入边的全局声明文件（**按使用判定为活**，非死码） | 1 | `types/wails.d.ts`（297 行，`declare global { interface Window { go?: { app?: { App: AppAPI } } … } }` + `export interface RuntimeAPI`；由 `tsconfig.app.json` 的 `include: ["src"]` 自动纳入编译，**无需被 import**；`tsc -p tsconfig.app.json --noEmit` 退出码 0。**注意**：本轮仅独立验证了「零入边 + 全局声明形态 + 编译通过」三点，**未逐字段统计 `ProjectCard`/`FilePickResult`/`RuntimeAPI` 的消费点**——「活」这一判定沿用既有扫描结论 + 文件头注释自陈（「为 `window.go.app.App.*` 与 `window.runtime.*` 提供类型安全，消除全站 `@ts-ignore`」，`wails.d.ts:1-9`） |

**结论**：真·零引用文件 = **1 个（TrendChart.tsx）**，且已被并发会话删除；
`types/wails.d.ts` 是 live 全局声明，**不可删**。
`genui/` 内部 **0 个**零引用文件（19/19 均被 import 或为 knip entry）。

---

## 未使用 npm 依赖清单（包名 | 证据 | 建议）

方法：`dep-scan.mjs` 扫 `frontend/src` + `frontend/scripts` + `vite.config.ts` + `eslint.config.js` +
`index.html`（共 1,204 文件），提取全部 `import … from` / `export … from` / 裸导入 / 动态 `import()` /
`require()` / `import.meta.glob()` / html `src|href`；再按包名归并。

| 包名 | 证据 | 建议 |
|---|---|---|
| **无** | 56 个声明包中 54 个直接命中 import；余 2 个经人工复核为**误报**（见下） | **无可删依赖** |

**两个「疑似幽灵依赖」的复核结论（均判活，不要删）**：

| 包名 | 表面证据 | 真实使用证据 | 判定 |
|---|---|---|---|
| `katex@0.16.47` | 全 `frontend/src` 无 `from "katex"` 命中 | `src/gaea/lib/mathText.ts:13`：`link.href = new URL("katex/dist/katex.min.css", import.meta.url).href` —— **Vite 可解析的相对 URL 引用**（官方 kaTeX CSS 按需注入路径，v4.179.0 已按此显式化）；`rehype-katex@7` 的 peer 依赖亦是它 | **活，保留** |
| `@types/node@^22` | 无 `from "@types/node"`（类型包本就不会被 import） | `vite.config.ts:4` `import os from 'node:os'` + `scripts/schedule/paths.test.ts` `node:fs/promises`；knip 已 `ignoreDependencies: ["@types/node"]` | **活，保留（knip 已正确豁免）** |

**devDependencies 未被 `import` 但属合法构建期工具**（不算幽灵依赖，逐条给出真实消费点）：
`vite`（`vite build`/`vitest` 的 peer + `vitest/config` 入口）、`tailwindcss`（`@tailwindcss/vite` 插件 + CSS `@import`）、
`typescript`（`tsc -b`）、`jsdom`（`vite.config.ts:46` `environment: 'jsdom'`）、
`@types/react` / `@types/react-dom` / `@types/d3-force`（tsc 编译期类型解析）。

---

## 磁盘垃圾/日志文件清单（路径 | 大小 | .gitignore/CI 引用情况 | 建议）

判定口径：**是否有任何脚本/CI 读取**（grep 全仓 `*.ps1`/`*.mjs`/`*.yml`/`*.bat`/`*.js` 引用该文件名）
+ **是否被 .gitignore 覆盖**（`git check-ignore -v` 实测）。**本节只评估，未删任何文件。**

| 路径 | 大小 | .gitignore | 脚本/CI 读取？ | 判定 |
|---|---|---|---|---|
| `frontend/vitest-report.json` | 1,256,138 B | ✅ `.gitignore:22`（专用行） | 仅 `.github/workflows/ci.yml:153,160` 在 **CI runner 上现场生成并现场读**；本地这份是残留（mtime 2026-09-27） | **可删（磁盘垃圾）**；再跑 `pnpm test -- --reporter=json` 会重建。属「CI 产物」，删了零风险 |
| `frontend/vitest-p4b.log` | 424,652 B | ✅ `.gitignore:3`（`frontend/.gitignore` `*.log`） | ❌ 零引用（grep `vitest-p4` 全仓仅中 1 条 docs，非脚本读取） | **可删** |
| `frontend/vitest-batch1.log` | 408,350 B | ✅ `*.log` | ❌ 零引用 | **可删** |
| `frontend/vitest-p4d.log` | 396,448 B | ✅ `*.log` | ❌ 零引用 | **可删** |
| `frontend/vitest-p4e.log` | 320,640 B | ✅ `*.log` | ❌ 零引用 | **可删** |
| `frontend/vitest-p4f.log` | 317,704 B | ✅ `*.log` | ❌ 零引用 | **可删** |
| `frontend/vitest-batch1-r2.log` | 315,982 B | ✅ `*.log` | ❌ 零引用 | **可删** |
| `frontend/vitest-run3.log` | 312,818 B | ✅ `*.log` | ❌ 零引用 | **可删** |
| `frontend/vitest-batch3.log` | 312,764 B | ✅ `*.log` | ❌ 零引用 | **可删** |
| `frontend/vitest-batch2.log` | 311,582 B | ✅ `*.log` | ❌ 零引用 | **可删** |
| `frontend/vitest-p4.log` | 311,396 B | ✅ `*.log` | ❌ 零引用 | **可删** |
| `frontend/vitest-p4c.log` | 311,272 B | ✅ `*.log` | ❌ 零引用 | **可删** |
| `frontend/vite-dev.log` | 709 B | ✅ `*.log` | ❌ 零引用（`vite-dev` 全仓 0 脚本命中） | **可删** |
| `frontend/vite-dev-v3.log` | 225 B | ✅ `*.log` | ❌ 零引用 | **可删** |
| `frontend/vite-dev.err.log` | **0 B** | ✅ `*.log` | ❌ 零引用 | **可删（空文件）** |
| `frontend/C?AIwubigrok.tmpci-v4401.log` | 209 B | ✅ `*.log` | ❌ 零引用（`tmpci-v4401` 全仓 0 命中） | **可删**；文件名含 **U+FFFD 替换字符**（非法/损坏路径名，`C` 与 `AIwubigrok` 之间夹控制字符），Windows 下删除需用通配 `C?AIwubigrok.tmpci-v4401.log` 或 `\\?\` 前缀路径。**典型「CWD 被拼成文件名」的脚本事故残留** |
| `frontend/package.json.md5` | 32 B | ✅ `.gitignore:20`（专用行） | ❌ **零引用**（全仓 grep `package.json.md5` 只命中 `.gitignore` 自己的 1 行） | **可删**；且**内容已过期**：文件记 `3e3df6eefae637a75e99b38ae89d9124`，实测 `package.json` 现行 MD5 = `43cca3d3f5b63054d0713e0a7d4c5253`（不匹配）。无消费者 + 已失真 = 纯残留 |
| `frontend/known-flaky.txt` | 700 B | ❌ **未被 ignore（已入库/受跟踪）** | ✅ **被读取**：`frontend/scripts/classify-vitest-failures.mjs:28` `const DEFAULT_LIST = 'known-flaky.txt'`；`ci.yml:160` 显式传参 `known-flaky.txt` | **必须保留（活配置，勿动）** |
| `frontend/pnpm-lock.yaml` | 231,090 B | ❌ 受跟踪 | ✅ CI `--frozen-lockfile`（`ci.yml:127,129`） | **必须保留（唯一 lockfile）** |
| `frontend/pnpm-workspace.yaml` | 31 B | ❌ 受跟踪 | ✅ pnpm 自动读取（`allowBuilds: esbuild: true`） | **必须保留** |

**小计**：可删磁盘垃圾 **17 个文件 / 5,000,921 B ≈ 4.77 MB**（其中 `vitest-report.json` 占 1.20 MB）。

**归类总结**
- **源码可删**：1 条 —— `frontend/src/gaea/components/TrendChart.tsx`（已被并发会话删除）。
- **磁盘垃圾可删**：17 条（上表）。
- **被 CI 脚本读取、必须保留**：`known-flaky.txt`（+ `scripts/go-known-flaky.txt` 后端对应物）。
- **已被 .gitignore**：除 `known-flaky.txt` / `pnpm-lock.yaml` / `pnpm-workspace.yaml` 外，其余 14 个 .log 与
  `vitest-report.json`、`package.json.md5` **全部**被 ignore（`git check-ignore -v` 逐条实测）。
- ⚠️ **`scripts/clean-tmp.ps1` 不会碰这些文件**：它 `$tmp = Join-Path $root '.tmp'`，全部 `Get-ChildItem $tmp`
  均在 `.tmp/` 内操作（第 18/43/53 行），模式 `*.log` 只作用于 `.tmp/` 内部。
  因此 frontend 根目录的日志**不会被自动清理**，但也**不会因别的脚本误删**——清理必须显式针对 frontend 根。

---

## Top 20 发现

> 约定：`[类型]` 为 D1…D8；`LOC≈n` 为涉及行数；路径均为相对 `C:\AI\wubigrok`。

1. **[P1]** `frontend/src/genui/parse.ts:124-205` `parsePartialGenuiSpec` + `collectPartialCandidates` + `MAX_PARTIAL_REPAIR_ATTEMPTS` + `partialAttemptsLimit` — **D1 死代码** — LOC≈58 — confidence:**high**
   - 证据：全仓 grep `parsePartialGenuiSpec` 命中 4 处 = `parse.ts` 定义 + `index.ts:44` barrel 再导出 + `parse.test.ts:5,74,84` **测试**；生产侧 0 命中。`collectPartialCandidates` 唯一调用点 `parse.ts:193`（在 `parsePartialGenuiSpec` 内）。生产流式路径实际走 `markdownFence.tsx:42` → `parseGenuiFenceBody`（完整闭合体）与 `Markdown.tsx:543-546`（同样只用 `splitGenuiFences` + `parseGenuiFenceBody`，且 `if (seg.kind !== "fence" || !seg.closed) continue;`）。`MAX_PARTIAL_REPAIR_ATTEMPTS` 唯一消费者是 `parse.ts:126` **自身**（`const partialAttemptsLimit = MAX_PARTIAL_REPAIR_ATTEMPTS;`）。
   - 建议：要么删除该「流式前缀解析」子系统（58 行 + 2 个测试块）并同步清 `index.ts` 再导出，要么在 Go/前端文档里补一句「预留 API」并加 `@internal` 标注避免下轮重复报。**medium 置信度部分**：若产品规划里明确有「未闭合围栏增量渲染」需求，则应保留但补生产调用点——目前是**能力已建、入口未接**。

2. **[P1]** `frontend/src/genui/index.ts:1-69`（整文件 barrel，69 行）— **D1 死代码 / D5 无用包装** — LOC≈69 — confidence:**high**
   - 证据：全仓 import 图（`reach-scan2.mjs`）显示 `genui/index.ts` 的**入边只有 1 条**：`gaea/lib/genuiPanel.ts:4` `import type { GenuiSpec } from "../../genui";`——**只取一个类型**。而 barrel 再导出的 30+ 个符号中，生产代码全部**绕过 barrel 直连内部模块**：`scope`（3 文件）、`markdownFence`（4 文件）、`parse`（2 文件）、`GenuiBlock`、`GenuiActionContext`、`fingerprint`、`spec`。零生产消费者的 barrel 导出含：`repairGenuiSpec`、`repairSingleComponent`、`parsePartialGenuiSpec`、`stripTrailingCommas`、`MAX_PARTIAL_REPAIR_ATTEMPTS`、`splitGenuiFences`、`parseGenuiFenceBody`、`loadBlockState`、`saveBlockState`、`resetInteractionStore`、`fingerprint`、`genuiStateKey`、`GENUI_ACTION_DEBOUNCE_MS`、`renderNode`、`GenuiScopeProvider`、`useGenuiScope`、`isGenuiFenceLang`、`GenuiMarkdownFence`。
   - 雪上加霜：`frontend/knip.json:15` **显式 `ignore: ["src/genui/index.ts"]`** —— 唯一的死代码检测器被配置成对该文件闭眼，所以「unused exports」永远不会在这个 barrel 上报警。
   - 建议：删 barrel，`genuiPanel.ts:4` 改 `import type { GenuiSpec } from "../../genui/spec"`（与同文件第 5-6 行已有的 `spec`/`fingerprint` 直连风格一致）；同步删 `knip.json` 的 `ignore` 第 1 项。**净减 69 行 + 消灭 1 处抑制面。**

3. **[P1]** `frontend/wails.json:5-7` `frontend:install: "npm install"` / `frontend:build: "npm run build"` / `frontend:dev:watcher: "npm run dev"` — **D6 屎山（配置漂移）** — LOC≈3 — confidence:**high**
   - 证据：`frontend/pnpm-lock.yaml` 是唯一 lockfile（`ci.yml:117-118` 注释自陈「v4.371 起唯一 lockfile 是 pnpm-lock.yaml：npm ci 结构性跑不通」），CI 用 `pnpm/action-setup@v4` + `pnpm install --frozen-lockfile`（`ci.yml:119-129`）；`package.json` 无 `packageManager` 字段，frontend 根**无 `package-lock.json`**（实测 `Test-Path` False）。`wails build`/`wails dev` 若真去跑 `npm install`，会在 pnpm 树上重解析依赖并**新生成 package-lock.json**，与「唯一 lockfile」口径直接冲突。另 `frontend:dev:serverUrl` 端口与 `vite.config.ts:66` `port: 5173` 一致，无问题。
   - 建议：改为 `pnpm install --frozen-lockfile` / `pnpm run build` / `pnpm run dev`（与 `ci.yml` 同源）。**低风险、纯配置**。

4. **[P1]** `frontend/src/genui/renderNode.tsx:840` `SubmitNode` 的 `enabled` 表达式 — **D2 冗余/不可达分支** — LOC≈4 — confidence:**high**
   - 证据：`(graded ? answeredAll : (node.action !== undefined && api.hasAction) || (node.action === undefined && groups.length === 0 && api.hasAction))`。第二个析取项的 `node.action === undefined` 与第一个的 `node.action !== undefined` 互斥 ⇒ 化简后其贡献仅为 `api.hasAction`，整体等价于 `api !== null && !api.locked && (graded ? answeredAll : api.hasAction)`。且 `answeredAll` 在 `graded === false` 时仍被求值（第 835-836 行），属白算。另：`SubmitNode` 从不提及 `api.fields`，而 `gradeNow` 却把 `{ ...api.fields }` 塞进 payload（第 853 行）——字段收集与禁用条件之间无耦合。
   - 建议：化简为 `graded ? answeredAll : api.hasAction`，可读性与等价性同时改善（等价性可由现有 `GenuiBlock.test.tsx` 判卷/交卷用例回归）。

5. **[P1]** `frontend/src/genui/renderNode.tsx:91-100 + 412-416` `GridNode` 双重 clamp + 单分支三元 — **D2 冗余分支** — LOC≈11 — confidence:**high**
   - 证据：`GridNode` 用 `Math.max(1, Math.min(12, node.cols))`，而 `node.cols` 早已被 `guard.ts:200` `int(o.cols, 1, MAX.maxGridCols /* =12 */)` clamp 过一次（guard 输出是 renderNode 的唯一输入，见 `renderNode.tsx:1-2` 文件头「本文件只消费 guard 修复后的节点」）。同一文件第 412 行：`const groupW = hasSeries ? plotW / Math.max(1, n) : plotW / Math.max(1, n);` —— **两个分支字面完全相同**。硬编码 `12` 亦与 `GENUI_LIMITS.maxGridCols` 双口径（`limits.ts:13` 为单一真相源）。
   - 建议：`GridNode` 直接 `style={{ gridTemplateColumns: \`repeat(${node.cols}, minmax(0, 1fr))\` }}`（或改引 `GENUI_LIMITS.maxGridCols`）；`groupW` 去掉三元。

6. **[P1]** `frontend/src/genui/renderNode.tsx:598-604` `INPUT_TYPES` 集合与 `guard.ts:140` 重复 — **D4 重复实现 / D8 重复类型定义** — LOC≈3+3 — confidence:**high**
   - 证据：同一 3 元集合在 genui 内出现 **3 份**：① `spec.ts:32` `export type InputType = "text" | "email" | "password"`（类型层）；② `guard.ts:140` `const INPUT_TYPES = new Set([...])`（守卫白名单）；③ `renderNode.tsx:598` `const INPUT_TYPES = new Set([...])`（渲染层兜底）。渲染层这份是**死逻辑**：guard 已在 `guard.ts:495-497` 用 `isTone(o.inputType, INPUT_TYPES)` 过滤，非白名单值到不了 `node.inputType`，故第 604 行 `INPUT_TYPES.has(...) ? ... : "text"` 的 `"text"` 兜底分支不可达。
   - **附带 D1**：`spec.ts:30` `export type TextSize` 与 `spec.ts:32` `export type InputType` **全仓零外部引用**（`export-scan.mjs` 对 1,173 文件 / 666 个含导出文件全量扫描：两名字在其他文件出现次数 = 0）。`guard.ts` 用 `TEXT_SIZES` 内联字面量集合（`guard.ts:142`）与 `GenuiText["size"] as` 断言来消费，从不 import 这两个类型名。
   - 建议：渲染层删 `INPUT_TYPES` 与三元，直接用 `node.inputType ?? "text"`；`guard.ts:142` 的 `TEXT_SIZES` 改由 `spec.ts` 导出常量（单一真相源），或直接标注 `TextSize`/`InputType` 为「供外部宿主/未来扩展」并加 `@public` 注释。

7. **[P1]** `frontend/src/genui/blocks/state.ts:15,24,29-32 + GenuiBlock.tsx:61,103,113,126` `round` 字段 — **D1 死代码（未接线 API）** — LOC≈6 — confidence:**high**
   - 证据：`round` 的注释自称「重做轮次：radio/quiz 以此为 key 重挂」（`state.ts:15`）。全仓 grep `api.round` / `round,` 命中：`GenuiBlock.tsx` 内 `useState`(61) / `api` 对象字面量(113) / `useMemo` 依赖数组(126) ——**纯自我引用**；`renderNode.tsx` 零命中（`QuizNode` 用的是自己的本地 `picked/revealed` state，`RadioNode` 用 `local`）。故「重挂」机制从未实现，`reset()` 里 `setRound(r => r + 1)`（`GenuiBlock.tsx:103`）只触发一次无消费者的重渲染。
   - 建议：要么在 `renderNode.tsx` 给 radio/quiz 加 `key={api?.round}` 兑现注释语义（真修复），要么删 `round` 字段与 `setRound`（−6 行 + 缩 `useMemo` 依赖）。

8. **[P1]** `frontend/src/genui/blocks/state.ts:22, `GenuiBlock.tsx:98,119,126`` `BlockApi.isSecret` — **D1 死代码** — LOC≈5 — confidence:**high**
   - 证据：全仓 grep `\.isSecret(` = **0 命中**（仅 `blocks/state.ts` 的接口声明与 `GenuiBlock.tsx` 的 `useCallback` 定义/对象字面量/依赖数组）。密码剥离逻辑**不消费它**：`GenuiBlock.tsx:132-134` 持久化过滤直接用 `secretFields.has(id)`，`renderNode.tsx` 的 `InputNode` 用本地 `const isSecret = type === "password"`（第 605 行，同名但**不同源**，易误读）。
   - 建议：删 `isSecret`（含 `useCallback` 与依赖项）；或让持久化过滤改走 `api.isSecret` 以统一真相源。当前形态是「API 面多一个没人用的钩子 + 两处同义命名混淆」。

9. **[P2]** `frontend/src/genui/blocks/state.ts:5` `QuestionMeta.label` — **D7 无用字段** — LOC≈1 — confidence:**high**
   - 证据：唯一写入点 `renderNode.tsx:717-722` 传 `label: node.options[0] ?? ""`（即**第一个选项文本**，语义可疑）；全仓读取点 = 0（`renderNode.tsx:864` 只读 `meta.answer`，`:880` 读 `meta.explanation`）。
   - 建议：删除字段（顺带消除 `node.options[0]` 这个「标签=第一选项」的错误语义）；若判卷结果面板想显示题目标签，应传 `node.label ?? node.question`。

10. **[P3]** `frontend/src/genui/styles.css:106` `.gui-text-muted,` 死选择器半截（+ 533/538 契约保留项）— **D1 死代码（no-op 级）** — LOC≈1 — confidence:**high**
    - 证据：`css-scan.mjs` 对 119 个 `.gui-*` 类做「类名是否在 `src/**/*.{ts,tsx,mjs}` 字面出现」全量比对，输出 **13 个零命中**；再**逐条剔除模板字符串动态生成族**（`` `gui-tone-${tone}` ``／`` `gui-btn-${tone}` ``／`` `gui-text-${size}` ``，由 `guard.ts:137-142` 常量闭合，`.gui-tone-info/success/error/accent/warn/danger`、`.gui-btn-primary/ghost/disabled/small/full`、`.gui-text-h1/h2/h3/caption/muted` 因此**全部可达，不可删**）后，真·零消费者**只剩 1 条**：
      ① `.gui-text-muted,`（第 106 行）—— 该行是**逗号选择器** `.gui-text-muted, .gui-muted { color: var(--gui-text-dim); }`：前半段无人生成（全仓 grep `gui-text-muted` 仅命中 `styles.css:106` 自身，`src` 的 ts/tsx/mjs 中 0 命中），但**整条规则实际生效**（`.gui-muted` 被大量使用）⇒ **删除该行是 no-op，规则不失效。**
      ② 契约保留项（非死码）：`.gui-btn-danger` / `.gui-btn-success`（533/538）—— `BUTTON_TONES`（`guard.ts:139`）允许 danger/success，`renderNode.tsx:581` 会产出 `` `gui-btn-${tone}` ``，故**可渲染**；只是全仓（含 harness 文档/测试/夹具）零使用这两种 tone。**删除会缩小 `ButtonTone` 契约表达力 ⇒ 保留 + 注释。**
    - 建议：删 `styles.css:106` 的 `.gui-text-muted,` 半截（纯 no-op 清理）；`.gui-btn-danger/success` 保留但在 `spec.ts:31` 旁加注「契约保留、当前无 harness 使用」。**本条优先级低、收益仅 1 行**——列出是为了让下轮扫描不再重复报这 13 条。

11. **[P2]** `frontend/src/genui/guard.ts` 6 处「算了不用」的无效校验分支 — **D2 无效校验分支** — LOC≈6 — confidence:**high**
    - 证据（逐条）：
      ① `guard.ts:181` `...(optStr(o.size, 16) !== undefined && TEXT_SIZES.has(o.size as string) ? { size: ... } : {})` —— `optStr` 结果被丢弃，真实判据是 `o.size` 的原始值；`16` 这个 cap 完全无效（与 `TEXT_SIZES` 实际最长值 `caption`=7 / `muted`=5 相去甚远，像从别的字段复制来的）。
      ② `guard.ts:279` `const value = str(o.value)`（`progress`）—— `value` 未使用（同分支下面另有 `const value = num(o.value, 0, 100)`，名字重复但作用域不同，易误读为「字符串校验通过才算数」）。
      ③ `guard.ts:521` `const label = str(o.label, 200); if (label === undefined) return null;`（`checkbox`/`switch`）… **实际返回对象里 `label` 缺失**：`:522-529` 构造的对象**没有 `label` 字段**（只有 `type`/`checked`/`action`），而 `GenuiCheckbox`/`GenuiSwitch` 的 `label` 是**必填**（`spec.ts:181,187`）。→ 这不是死分支，而是 **guard 丢弃必填字段的真 bug**（`ToggleNode` 渲染 `node.label` 会得到 `undefined`）。
      ④⑤⑥ `guard.ts:369` `color(o.color)`、`:409` `color(s.color)`、`:495` `isTone(o.inputType, INPUT_TYPES)` —— 三者结果只用于 `!== undefined` 判定，随后 `as` 断言回写原始值；功能等价但**双重解析**（解析一次判空、再一次断言透传）。
    - 建议：①③ 必须修（① 改为 `const size = optStr(o.size, 16); ... size !== undefined && TEXT_SIZES.has(size)`；③ 补 `label` 到返回对象）；② 删无用 `value`；④⑤⑥ 改为 `const v = color(...); ... v !== undefined ? {color: v} : {}`。

12. **[P2]** `frontend/src/genui/guard.ts:510-512, 537-539` 三次 clamp 冗余 — **D2 冗余分支** — LOC≈6 — confidence:**high**
    - 证据：`...(optInt(o.selected, -1, options.length - 1) !== undefined && (o.selected as number) >= 0 ? { selected: Math.round(o.selected as number) } : {})`。`optInt` 内部已 `num(...)` clamp 且 `Math.round`（`guard.ts:85-89`），外层再 `Math.round` + `>= 0` 复检属重复；且两处（`select`、`radio`）**逐字重复**（D4）。
    - 建议：抽 `function selIdx(v: unknown, n: number)` 单源。

13. **[P2]** `frontend/src/genui/renderNode.tsx:935` `AccordionReal` 内联类型字面量 — **D8 重复类型定义** — LOC≈2 — confidence:**high**
    - 证据：`{ node: { type: "accordion"; items: { title: string; items: GenuiNode[] }[] }; depth: number }` 与 `spec.ts:231-234` `GenuiAccordion` **结构完全相同**，而 `renderNode.tsx:5-38` 的 import 清单里**恰好没有 `GenuiAccordion`**（其余 27 个节点类型都 import 了）——典型的「忘了加 import 就地重写一份」。函数名 `AccordionReal` 的 `Real` 后缀也暗示曾与别的实现并存。
    - 建议：`import { type GenuiAccordion }` 并用 `{ node }: { node: GenuiAccordion; depth: number }`，顺带改名 `AccordionNode` 与同族 `TabsNode`/`CardNode` 对齐。

14. **[P2]** `frontend/tsconfig.app.json:21-22` `noUnusedLocals: false` + `noUnusedParameters: false` — **D7（工具链关闭）** — LOC≈2 — confidence:**high**
    - 证据：`tsconfig.node.json:18-19` 同两项为 **`true`**（且 `erasableSyntaxOnly: true`），`app` 配置刻意关掉。这正是本轮 D1/D7 类发现（未用 import、未用参数、死 API 字段）**无法被 `tsc` 拦下**的结构性原因；目前只能靠 eslint `@typescript-eslint/no-unused-vars`（`eslint.config.js:72-76`）且**已降为 `warn`**，而 CI 只跑 `pnpm lint`（`ci.yml:131`）——`eslint` 默认对 warn **退出码 0**（本轮 `eslint src/genui` 实跑：0 error / 0 warning / exit 0）。
    - 建议：先开 `noUnusedLocals` 试跑（存量可能不小，建议「只对 `src/genui/**` 开」的渐进路径：单独一个 `tsconfig.genui.json` 或在 CI 加 `tsc --noUnusedLocals` 仅对 genui 的窄档）。
    - **附带证据**：本轮 `renderNode.tsx` 的 `seriesColor(chart, …)` 未用首参（`ChartNode` 不消费 `node` 参数）**恰好**在审计期间被并发会话删掉，说明这类问题确实存在且正在被手工发现——工具链本可自动拦下。

15. **[P2]** `frontend/src/genui/index.ts:40,46` 再导出内部助手 — **D5 过度抽象/无用包装** — LOC≈4 — confidence:**medium**
    - 证据：`repairSingleComponent`（guard 内部用：`guard.ts:685` 调 `repairGenuiSpec` 的 `single` 分支）与 `MAX_PARTIAL_REPAIR_ATTEMPTS`（`parse.ts:126` 自用）被抬到 barrel 公开面，但生产零消费（见发现 1、2）。`stripTrailingCommas` 同理（生产只经 `parseGenuiFenceBody` 内部调用，`parse.ts:111,197`）。
    - 建议：与发现 2 合并处置（删 barrel 后自然消失）。

16. **[P2]** `frontend/knip.json:20-22` `ignoreDependencies: ["@types/node"]` 为唯一豁免 —— **D1（工具链空转）** — LOC≈3 — confidence:**medium**
    - 证据：knip **未安装**（`node_modules/.bin/knip.cmd` 实测不存在，`package.json` devDependencies 无 knip），而 `knip.json` 头部仍引 schema `knip@5`、`package.json` 无 `knip` script、`ci.yml` 无 knip step。⇒ 死代码工具链实际处于「**有配置、无执行者**」状态：`$schema` 指向 v5 而运行时应是当前版；`ignoreExportsUsedInFile: true` 会掩盖「导出后文件内又用」的真死导出。
    - 建议：二选一——① 接回 CI（`pnpm add -D knip` + `pnpm knip` step，且**先删 `ignore` 里的 `src/genui/index.ts`**，否则发现 2 永不被报）；② 明确落档为「历史遗留、不接 CI」并把 schema 版本对齐，避免下轮审计重复计入。**注意 v4.179.0 CHANGELOG 自陈「knip 首扫三发现」——说明它曾被临时用过，之后被移除，配置留下来了。**

17. **[P2]** `frontend/package.json:8,11` `build: "tsc -b && vite build"` / `lint: "eslint ."` 与 CI 口径不一致 — **D6 屎山（门禁密度）** — LOC≈2 — confidence:**medium**
    - 证据：`ci.yml:131,133` 只跑 `pnpm lint` + `pnpm build`，**不跑 `pnpm test`**（测试走 `ci.yml:149-164` 单独 step 且带 flaky 分类器）；而 `knip.json:11-12,15` 把 `src/**/*.test.ts(x)` 声明为 entry。⇒ 死代码工具链（若启用）会以测试文件为根，测试引用的「仅供测试」导出会被判活——这解释了为何 `parse.ts` 的 partial 解析族（仅测试消费）在 knip 口径下**不会**报死。这是**口径设计**而非缺陷，但必须写进判断依据，否则下轮审计会重报。
    - 建议：在 `knip.json` 旁加注释或在审计基线文档登记「测试入口 ⇒ partial 解析族按设计保留」，让「仅测试消费」成为**显式决策**而非静默豁免。

18. **[P3]** `frontend/eslint.config.js:17` `.eslint-out.json` 在 `globalIgnores` —— **D1 死配置** — LOC≈1 — confidence:**medium**
    - 证据：`frontend/.eslint-out.json` 实测**不存在**；全仓 grep `.eslint-out.json` 唯一命中是 `eslint.config.js:17` 自己（无脚本生成该文件，`scripts/ci.ps1`/`ci.yml` 均无此输出重定向）。
    - 建议：删除该 ignore 项（或补一个生成它的脚本，若本意是本地诊断）。

19. **[P3]** `frontend/src/genui/` 内 3 处 `eslint-disable` 抑制 — **D7（需人工确认）** — LOC≈3 — confidence:**medium**
    - 证据：`renderNode.tsx:3` `/* eslint-disable react-refresh/only-export-components */`（文件级）、`scope.tsx:2` 与 `GenuiActionContext.tsx:1`（同规则文件级）；另有 3 处行内 `// eslint-disable-next-line react-hooks/exhaustive-deps`（`renderNode.tsx:608,725`、`GenuiBlock.tsx` 无）。前 3 处在 `eslint.config.js:82` 该规则**已降为 `warn` 且 `allowConstantExport: true`** ⇒ 文件级 disable 属**冗余抑制**（规则本就不阻塞）。
    - 建议：删 3 处文件级 disable，保留 2 处 `exhaustive-deps`（那 2 处有明确理由注释「仅挂载注册一次」）。需人工确认是否有未写下的历史原因。

20. **[P3]** `frontend/src/genui/parse.ts:137` `collectPartialCandidates` 返回值的 `scannedChars` 字段无消费者 — **D1/D7** — LOC≈2 — confidence:**medium**
    - 证据：返回 `{ candidates, scannedChars: scanned }`；全仓唯一解构点是 `parse.ts:193` `const { candidates } = collectPartialCandidates(text)`（丢弃 `scannedChars`）与 `parse.test.ts:66`（同样只取 `candidates`）。字段本身可能用于「预算/性能诊断」，但**无任何读取点**。
    - 建议：随发现 1 一并处置（保留则补消费者，如遇超长围栏时上报扫描字符数）。

---

## 全量发现表

| 文件:行 | 符号 | 类型 | LOC | 置信度 | 证据 | 建议 |
|---|---|---|---|---|---|---|
| `src/genui/parse.ts:124-205` | `parsePartialGenuiSpec` / `collectPartialCandidates` / `MAX_PARTIAL_REPAIR_ATTEMPTS` / `partialAttemptsLimit` | D1 | 58 | high | 全仓唯一消费者 = `parse.test.ts:5,74,84`；生产流式走 `markdownFence.tsx:42` / `Markdown.tsx:543-546` 的 `parseGenuiFenceBody`；`MAX_PARTIAL_REPAIR_ATTEMPTS` 只被 `parse.ts:126` 自用 | 删（或补生产入口 + 标 `@internal`） |
| `src/genui/parse.ts:137` | `PartialCandidate.scannedChars` / 返回值 `scannedChars` | D1/D7 | 2 | medium | `const { candidates } = …` 丢字段；`parse.test.ts:66` 同 | 随上条处置 |
| `src/genui/index.ts:1-69` | barrel 全文件 | D1/D5 | 69 | high | 唯一入边 `genuiPanel.ts:4` 仅取 `GenuiSpec` 类型；其余 30+ 再导出零生产消费者 | 删 barrel，改直连 `spec` |
| `frontend/knip.json:15` | `ignore: ["src/genui/index.ts"]` | D1 | 1 | high | 唯一死代码检测器对该文件闭眼 | 随 barrel 一并删 |
| `frontend/knip.json:20-22` | `ignoreDependencies: ["@types/node"]` | D1 | 3 | medium | knip 未安装（`node_modules/.bin/knip.cmd` 不存在）、无 script、CI 无 step；`$schema` 锁 knip@5 | 接回 CI 或落档废弃 |
| `frontend/knip.json:11-12,15` | `src/**/*.test.ts(x)` 作 entry | D1（口径） | 2 | high | 使「仅测试消费」的导出一律判活 | 显式登记口径 |
| `src/genui/guard.ts:181` | `optStr(o.size, 16)` 结果丢弃 | D2 | 1 | high | 判据实为 `o.size` 原值；`16` cap 无意义 | 用 `optStr` 返回值 |
| `src/genui/guard.ts:279` | `const value = str(o.value)` 未使用（progress） | D2 | 1 | high | 同分支另有 `num(o.value,0,100)` | 删 |
| `src/genui/guard.ts:521-529` | `checkbox`/`switch` 的 `label` 校验后被丢弃 | **D2/bug** | 8 | high | 校验 `str(o.label,200)` 通过后构造的对象**不含 `label`**，而 `spec.ts:181,187` `label` 必填 ⇒ `ToggleNode`（`renderNode.tsx:705`）渲染 `undefined` | **补 `label` 进返回对象** |
| `src/genui/guard.ts:369,409,495` | `color()` / `isTone()` 双重解析 | D2 | 3 | medium | 结果只做 `!== undefined` 判定，随后 `as` 回写原值 | 用解析后的值 |
| `src/genui/guard.ts:510-512,537-539` | `select`/`radio` selected clamp 复检 | D2/D4 | 6 | high | `optInt` 已 clamp+round；两处逐字重复 | 抽 `selIdx()` 单源 |
| `src/genui/guard.ts:140` + `renderNode.tsx:598` | `INPUT_TYPES` × 2 | D4 | 6 | high | 与 `spec.ts:32` `InputType` 构成 3 份真相 | 单源导出 |
| `src/genui/guard.ts:142` + `spec.ts:30` | `TEXT_SIZES` vs `TextSize` | D4/D8 | 2 | high | 类型名零外部引用（export-scan：0 命中） | 合并为 `spec.ts` 常量 |
| `src/genui/guard.ts:137-142` | `BADGE_TONES`/`CALL_TONES` 与 `renderNode.tsx:131,285` 同名集合 | D4 | 6 | medium | 守卫白名单与渲染白名单各自维护同 4 元集合 | 抽 `spec.ts` 常量共享 |
| `src/genui/guard.ts:53` + `:61,67,122,156,161,172,200,258,261,276,281,291,301,316,324,348,351,391,404,426,443,455,463,468,485,498,513,526,540,564,576,585,586,590,604,618,637,648,673,678,680,681` | 别名 `MAX` 替代 `GENUI_LIMITS` | D5 | ~45 处 | low | `const MAX = GENUI_LIMITS`；直接引原名更可读、grep `GENUI_LIMITS.max*` 时不会漏 | 可选：改回原名（纯风格，**不建议单独立项**） |
| `src/genui/spec.ts:30` | `export type TextSize` | D1 | 1 | high | 全仓外部出现 0 次（export-scan 1,173 文件全量） | 删或标 `@public` |
| `src/genui/spec.ts:32` | `export type InputType` | D1 | 1 | high | 同上 | 同上 |
| `src/genui/spec.ts:64-69,250-251` | `GenuiDivider` / `GenuiSpacer` 被 `guard.ts` import/使用但 barrel 未再导出 | D7/D8 | 4 | medium | `index.ts` 的 type 导出清单（2-39 行）**不含**这两个类型；宿主无法从公开面拿到 | 补进 barrel 或确认「内部专用」 |
| `src/genui/renderNode.tsx:935` | `AccordionReal` 内联对象类型 | D8 | 2 | high | 与 `spec.ts:231-234` `GenuiAccordion` 同构；import 清单独缺该项 | 改用 `GenuiAccordion` |
| `src/genui/renderNode.tsx:840` | `SubmitNode.enabled` 三元 | D2 | 4 | high | 分支互斥 ⇒ 等价 `api.hasAction` | 化简 |
| `src/genui/renderNode.tsx:412` | `groupW` 三元 | D2 | 1 | high | 两分支字面相同 | 删三元 |
| `src/genui/renderNode.tsx:95` | `Math.max(1, Math.min(12, node.cols))` | D2 | 1 | high | guard 已 clamp；硬编码 12 与 `GENUI_LIMITS.maxGridCols` 双口径 | 直接用 `node.cols` |
| `src/genui/renderNode.tsx:598-604` | `INPUT_TYPES` + `type` 兜底三元 | D1/D4 | 7 | high | guard 已过滤，`"text"` 兜底分支不可达 | 删 |
| `src/genui/blocks/state.ts:15,24` + `GenuiBlock.tsx:61,103,113,126` | `round` / `setRound` | D1 | 6 | high | 注释声称 radio/quiz 以它为 key 重挂，`renderNode.tsx` 零消费 | 兑现语义或删 |
| `src/genui/blocks/state.ts:22` + `GenuiBlock.tsx:98,119,126` | `isSecret` | D1 | 5 | high | 全仓 `\.isSecret\(` = 0；持久化过滤直用 `secretFields.has` | 删或统一 |
| `src/genui/blocks/state.ts:5` | `QuestionMeta.label` | D7 | 1 | high | 写入 `renderNode.tsx:719`（=第一选项文本，语义错），读取 0 | 删 |
| `src/genui/styles.css:106` | `.gui-text-muted,`（逗号选择器前半截） | D1（no-op） | 1 | high | 全仓 grep `gui-text-muted` 仅命中 CSS 自身；但同规则的 `.gui-muted`（107 行）被 `gui-text-${size}` 与 `gui-muted` class 消费 ⇒ 删该行是 no-op | 删该行（低优先） |
| `src/genui/styles.css:533,538` | `.gui-btn-danger` / `.gui-btn-success` | D1（契约保留，非死码） | 10 | medium | `BUTTON_TONES`（`guard.ts:139`）+ `gui-btn-${tone}`（`renderNode.tsx:581`）使两者**可渲染**；仅缺 harness 使用 | **保留** + 注释 |
| `src/genui/styles.css:1` | 「只消费全局令牌」注释 vs 内联 hex 兜底 | D3（文档漂移） | 1 | medium | `styles.css:11-13` `--color-success, #34d399` / `--color-warning, #f59e0b` / `--color-destructive, #ef4444` 三个 raw hex，与第 1 行自述冲突（未走 `hex-exempt`，因 `eslint-rules/no-raw-hex.js:9` 只覆盖 JS/TS `Literal`，不覆盖 CSS） | 改注释为「令牌 + 兜底」，或把兜底值也 token 化 |
| `src/genui/` 全目录 | `TODO`/`FIXME`/`XXX`/`HACK` | D7 | 0 | high | grep `TODO|FIXME|XXX|HACK|待办|废弃|deprecated` 于 `src/genui` = **0 命中** | 无需处置（**好消息**） |
| `src/genui/` 全目录 | 注释掉的代码块 | D3 | 0 | high | 正则 `^\s*//\s*(const|let|function|return|if|import|export|type|…)` 于 `src/genui` = **0 命中**；3 处疑似命中经目检全为**用法示例注释**（`limits.ts:3` 生成命令、`markdownFence.tsx:4-5` 调用样例） | 无需处置（**好消息**） |
| `src/genui/renderNode.tsx:3` / `scope.tsx:2` / `GenuiActionContext.tsx:1` | 文件级 `eslint-disable react-refresh/only-export-components` | D7 | 3 | medium | 该规则在 `eslint.config.js:82` 已降为 `warn` 且 `allowConstantExport: true` ⇒ 抑制冗余 | 删 3 处 |
| `src/genui/renderNode.tsx:608,725` | 行内 `eslint-disable-next-line react-hooks/exhaustive-deps` | D7 | 2 | medium | 均有理由注释（`仅挂载注册一次`）；但 `:606-609` 的 `[]` 依赖 + `isSecret`/`api` 闭包捕获有陈旧闭包风险 | **需人工确认**（建议改依赖数组或 `useRef`） |
| `src/genui/guard.test.ts` | 覆盖盲区 | D7 | — | high | 仅覆盖 text/stat/progress/button/avatar/table/quiz/divider/unknown；`radio`/`select`/`slider`/`checkbox`/`switch`/`textarea`/`submit`/`tabs`/`accordion`/`chart`/`diff`/`json`/`code`/`list`/`timeline`/`steps`/`copy`/`badge`/`keyvalue`/`card`/`grid`/`row`/`col`/`spacer` 共 24 型**无 guard 级用例** | 补表驱动用例（尤其发现 11 的 `label` bug 正落在此盲区） |
| `src/genui/limits.ts:1-4` | `DO NOT EDIT` 生成文件 | — | 56 | high | 头部指向 `GAEA_GENUI_LIMITS_UPDATE=1 go test -run TestGenuiLimitsSync`；`spec.ts:299` 消费 `GENUI_GO_NODE_TYPES` | **不要手工改**（漂移会被 Go 侧逐字节校验拦下） |
| `frontend/wails.json:5-7` | `npm install` / `npm run build` / `npm run dev` | D6 | 3 | high | 唯一 lockfile 是 pnpm（`ci.yml:117-129`）；frontend 根无 `package-lock.json`（实测） | 改 pnpm |
| `frontend/package.json:8` | `build: "tsc -b && vite build"` | D6 | 1 | medium | `tsc -b` 走 `tsconfig.json` 引用双工程；`app` 关 unused 检查（见发现 14） | 见发现 14 |
| `frontend/package.json:11` | `lint: "eslint ."` | D6 | 1 | medium | CI 只 `pnpm lint`；warn 不阻塞 ⇒ `no-unused-vars`/`ban-ts-comment` 等 5 条 warn 无门禁效力 | 若要求门禁，改 `--max-warnings` |
| `frontend/package.json:1-78` | 声明依赖 | — | 78 | high | dep-scan 1,204 文件：**零未使用依赖**（`katex`/`@types/node` 为误报已复核） | 无需处置（**好消息**） |
| `frontend/eslint.config.js:17` | `'.eslint-out.json'` ignore | D1 | 1 | medium | 文件不存在、无脚本生成（全仓 grep 唯一命中即本行） | 删 |
| `frontend/eslint.config.js:38-66` | `exemptFiles` 17 项 | D7 | 29 | medium | 逐项需与真实文件核对（本轮仅核 `genui`，该目录不在名单内） | **需人工确认**（跨域项） |
| `frontend/tsconfig.app.json:21-22` | `noUnusedLocals:false` / `noUnusedParameters:false` | D7 | 2 | high | `tsconfig.node.json:18-19` 同为 `true`；关掉即本轮 D1/D7 无法被 tsc 拦 | 渐进开启 |
| `frontend/tsconfig.app.json:6` | `"lib": ["ES2023","DOM"]` 缺 `DOM.Iterable` | D7 | 1 | low | `renderNode.tsx:317` `[...node.name.trim()]` 与 `:62` `new Set(...)` 展开依赖迭代器；当前 `types:["vite/client"]` 下 tsc 仍 exit 0 ⇒ 仅潜在风险 | 可选 |
| `go.mod:7-25` | 17 个 direct require | — | 19 | high | 1825 个 `.go` 文件逐包 grep：**全部有 import 命中**（`goquery` 11 / `excelize` 27 / `toml` 11 / `x/text` 10 / `sqlite` 9 / `websocket` 8 / `wails` 6 / `x/net` 6 / `gooxml` 5 / `x/sys` 4 / `go-epub` 3 / `x/image` 2 / 其余各 1） | **无需处置**（`go build ./...` 在 CI backend job 第 22 行即做兜底） |
| `.golangci.yml:5-12` | `unused` linter 已启用 | — | 1 | high | 第 11 行 `- unused`；`ci.yml:79-82` 钉 `v2.14.0` | **不要动**（Go 侧死码防线） |
| `frontend/vite-dev*.log` × 3 | — | 磁盘垃圾 | — | high | `git check-ignore -v` → `frontend/.gitignore:3:*.log` | 可删 |
| `frontend/vitest-*.log` × 11 | — | 磁盘垃圾 | — | high | 同上；`vitest-p4*`/`vitest-batch*`/`vitest-run3` 全仓零脚本引用 | 可删 |
| `frontend/C?AIwubigrok.tmpci-v4401.log` | — | 磁盘垃圾 | — | high | 209 B；文件名含 U+FFFD；`tmpci-v4401` 全仓零引用；`*.log` ignored | 可删（注意非法字符路径） |
| `frontend/package.json.md5` | — | 磁盘垃圾 + 失真 | — | high | 记 `3e3df6ee…`，实测 `43cca3d3…` **不匹配**；零消费者（grep 仅命中 `.gitignore:20`） | 可删 |
| `frontend/vitest-report.json` | — | 磁盘垃圾 | — | high | `.gitignore:22` 专用行；仅 CI runner 现场生成/读取（`ci.yml:153,160`）；本地这份 mtime 2026-09-27 | 可删 |
| `frontend/known-flaky.txt` | — | **必须保留** | 10 | high | 未被 ignore（受跟踪）；`frontend/scripts/classify-vitest-failures.mjs:28` `DEFAULT_LIST='known-flaky.txt'` + `ci.yml:160` 显式传参 | **不要删** |
| `scripts/clean-tmp.ps1:18,43,53` | `$tmp = .tmp` 作用域 | — | 3 | high | 全部 `Get-ChildItem $tmp`；**不会**触及 frontend 根日志 | 结论：根目录日志需人工清理 |

**合计：52 条发现**（含 4 条「无需处置」的好消息项、3 条「需人工确认」项、1 条真 bug）。

---

## 依赖/配置精简建议（含 go.mod 未使用依赖结论）

### 1. npm 依赖：**零可删**

`dep-scan.mjs` 覆盖 `frontend/src` + `frontend/scripts` + `vite.config.ts` + `eslint.config.js` + `index.html`
（1,204 文件，含静态/动态/裸/`require`/`import.meta.glob`/CSS `@import`/html `src|href` 七类提取）：

| 指标 | 值 |
|---|---|
| `dependencies` 声明 | 44 |
| `devDependencies` 声明 | 20 |
| 直接命中 import | 54 |
| 需人工复核 | 2（`katex`、`@types/node`）→ **均判活** |
| **可删** | **0** |

> 唯一「看起来像幽灵依赖」的 `katex` 走 `new URL("katex/dist/katex.min.css", import.meta.url)`（`mathText.ts:13`），
> 这是 Vite 官方支持的资产解析形态，静态 import 扫描器必然漏报。**若未来引入 knip，务必把 `katex` 加进
> `ignoreDependencies`，否则会误报并诱导错误删除**（删掉会让数学公式失去样式）。

### 2. go.mod：**零未使用直接依赖**

17 个 direct require 在 1,825 个 `.go` 文件（排除 `clones/`）中**全部**有 import 命中，
最低 1 次（`cascadia`、`fsnotify`、`go-runewidth`、`mscfb`、`go-qrcode`）。
补充两道结构性保证：① Go 编译器**本身拒绝**「声明于 require 但全模块未 import」的直接依赖（`go build ./...`
在 `ci.yml:22` 每 push 必跑）；② `.golangci.yml:11` 启用 `unused` linter（v2.14.0 钉版，`ci.yml:79-82`）。
⇒ **go.mod 无需精简**；若要做「间接依赖瘦身」，须跑 `go mod tidy` 并核对 `go.sum` 差异，**不属本次只读审计范围**。

### 3. 配置层可落地精简（按收益/风险排序）

| # | 动作 | 文件 | 收益 | 风险 |
|---|---|---|---|---|
| 1 | 删 `npm`→`pnpm` 漂移 | `wails.json:5-7` | 消除「双 lockfile 幻觉」与 `wails dev/build` 爆改依赖树 | 低（与 `ci.yml` 同口径） |
| 2 | 删 genui barrel + knip ignore | `genui/index.ts`、`knip.json:15` | −69 行、−1 处检测抑制面、类型 import 路径与同文件其余 3 条 import 风格统一 | 低（唯一消费者改 1 行） |
| 3 | 删「流式前缀解析」子系统 | `genui/parse.ts:124-205` + `index.ts` 再导出 + `parse.test.ts:41-89` 三块 | −58 行生产码、−2 处测试 | **中**（需产品确认无「未闭合围栏增量渲染」规划） |
| 4 | 删 17 个磁盘垃圾 | `frontend/` 根 | 回收 **≈4.77 MB**、消除 `package.json.md5` 失真误导 | 极低（全部 gitignored 且零脚本引用） |
| 5 | 修 guard 的 `checkbox`/`switch` `label` 丢失 | `genui/guard.ts:520-527` | **修一个真 bug**（`ToggleNode` 当前渲染无标签） | 低（补字段是超集，不动契约） |
| 6 | 清 3 处冗余 eslint 文件级 disable + `.eslint-out.json` ignore | `renderNode.tsx:3`、`scope.tsx:2`、`GenuiActionContext.tsx:1`、`eslint.config.js:17` | 减噪音，让真抑制（`exhaustive-deps` × 2）显形 | 极低 |
| 7 | knip 决策落档 | `knip.json` | 消除「有配置无执行者」的审计重复劳动 | 低 |
| 8 | `noUnusedLocals` 渐进开启（先 genui 窄档） | `tsconfig.app.json:21` | 让本轮 10+ 条 D1/D7 未来自动拦下 | **中**（存量可能多，建议窄档起步） |

**建议不动**：`limits.ts`（Go 生成 + 逐字节漂移校验）、`vite.config.ts` 的 maxWorkers/testTimeout 调参注释
（有实测数据支撑）、`eslint.config.js` 的 `exemptFiles` 名单（跨域，本轮未核）、`generated` 类文件、
`pnpm-lock.yaml` / `pnpm-workspace.yaml`。

---

## 不建议动的地方（说明为什么）

| 位置 | 为什么别动 |
|---|---|
| `frontend/src/genui/limits.ts:1-56` | `DO NOT EDIT MANUALLY` 生成物，单一真相源在 `internal/gaea/genui/limits.go`；Go 侧 `TestGenuiLimitsSync` **逐字节**校验，手改必红（`limits.ts:4`）。要改常量请改 Go 再跑 `GAEA_GENUI_LIMITS_UPDATE=1 go test -run TestGenuiLimitsSync ./internal/gaea/genui/`。 |
| `frontend/src/genui/styles.css` 里 `` `gui-tone-${…}` `` / `` `gui-btn-${…}` `` / `` `gui-text-${…}` `` 的 CSS 规则（`.gui-tone-info/success/warning/error/accent/danger/warn`、`.gui-btn-primary/ghost/disabled/small/full`、`.gui-text-h1/h2/h3/caption/muted`…） | 静态扫描器（含 knip/purgecss 类工具）会判「未使用」（本轮 `css-scan.mjs` 一次就输出 **13 条**假阳性），实际由 `renderNode.tsx:60,135,289,581` 的**模板字符串动态拼接**生成，取值域由 `guard.ts:137-142` 常量闭合 ⇒ **全部可达**。**任何基于字符串匹配的 CSS/类名死码清理都会误删它们。** 真阳性只有 1 条半截选择器（`.gui-text-muted,`）。 |
| `frontend/src/genui/{spec.ts 的类型声明, guard.ts 的 switch 分支}` | 这是**契约面**：新增/改名组件只改 Go 侧 `nodeTypes` 再生成，`spec.ts:292-299` 注释已落档（审计 X1-07/FE6-03 收敛）。删「看起来没人用」的节点类型（如 `divider`/`spacer`/`json`/`diff`）会破坏 Go 侧白名单对齐与模型输出面。**发现 19（`GenuiDivider`/`GenuiSpacer` 未进 barrel）应「补导出」而非「删类型」。** |
| `frontend/src/genui/interaction.ts` 的 LRU / `clearBlockStatesForSession` 前缀匹配 | 有明确审计出处（2026-09 #7）与专门测试（`interaction.test.ts` 覆盖 `s1` vs `s10`、`s1extra` 不误伤）；`MAX_BLOCKS` 已 FE6-19 改为 `GENUI_LIMITS.maxNodes` 同源。 |
| `frontend/known-flaky.txt`、`scripts/go-known-flaky.txt` | CI 门禁判据（分类器默认清单 + `ci.yml:160` 显式传参）。当前两者均为**空清单**（仅表头）且自陈原因（vite.config worker 上限 / testTimeout 30s 收口后不再需要重试）——删文件会让分类器 `EACCES/ENOENT`。 |
| `scripts/clean-tmp.ps1` | 只作用于 `.tmp/`（第 18/43/53 行），**不会**误删 frontend 根日志；且其存在理由有实测（v4.359 `.tmp` 2.4 GB → vitest worker 间歇失败）。**想清理 frontend 根日志必须写新脚本或手工删，不能冀望它。** |
| `frontend/eslint.config.js` 的 `globalIgnores` 与 5 条降级 warn | 每条都有落档理由（`eslint.config.js:9-12`「硬错误清零、存量风格降 warn」+ 逐条注释）；`@typescript-eslint/no-explicit-any: 'error'`（`T6-10.2` any 全仓清零）是硬门禁，勿动。 |
| `frontend/tsconfig.node.json` | `noUnusedLocals/Parameters: true` + `erasableSyntaxOnly: true` 已是严格档，只 include `vite.config.ts`；勿与 `tsconfig.app.json` 合并（会互相污染 include 面）。 |
| `frontend/vite.config.ts` 的 `maxWorkers` / `testTimeout` / `hookTimeout` 注释 | 注释里是**实测数据**（本机 32 逻辑核、单文件 import 2.1 s、重组件 15~20 s 假红），是「为什么是这个值」的唯一依据；删注释会造成下一个人重新调参。 |
| `frontend/src/genui/renderNode.tsx`（当前） | ⚠️ **文件在审计期间被会话外进程编辑**（` M`，1098 行）。**不要在并发的编辑会话附近做批量重构**——本轮就有 `seriesColor` 参数被并发删除，若审计方也在改会冲突。 |
| `frontend/src/types/wails.d.ts` | 全局声明文件（297 行，`declare global`），零 import 是**正确形态**，由 `tsconfig.app.json` 的 `include: ["src"]` 自动纳入；`tsc --noEmit` exit 0 证明其被全仓消费。**「零 import」不等于死码。** |
| `frontend/src/gaea/components/TrendChart.tsx` | 已由并发会话删除（`git status` `D`）。**本审计确认零引用成立**，无需再次处置；若回滚恢复，再按发现 1 类处理。 |

---

## 附：审计中确立的关键口径（供下轮复用，避免重复劳动）

1. **`reach-scan2.mjs` 是可信的零引用判定器**：1199 文件 / 3,628 边 / 唯一非 entry 零入边 = `TrendChart.tsx`。
   Windows 盘符路径**不能用 `path.posix.resolve`**（会产出 `/AI/wubigrok/C:/…` 假路径），必须手工 posix 归一化。
2. **knip entry 口径决定「死导出」判定**：`src/**/*.test.ts(x)` 是 entry ⇒ 「仅测试消费」的导出（`parsePartialGenuiSpec` 族）
   在 knip 语义下**不是死码**。本报告仍把它列为 D1，但标注了「预留 API 可能性」——下轮请沿用这个双标注。
3. **`export-scan.mjs` 只信「名字在定义文件外零出现」**：全前端 413 条，绝大多数是 `src/gaea/lib/types/*` 与 `api/*` 的
   契约类型（有意公开），**不要按数量下结论**；本轮只在 `genui` 域内采信（2 条，均高置信）。
4. **CSS 类名零命中的必须剔除模板字符串族**后再判，否则 13/119 全是假阳性（本轮真阳性仅 1 条）。
5. 所有命令均为只读；`tsc --noEmit` 与 `eslint --format json` 的退出码与计数已记录（均 0 error）。
