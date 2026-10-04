# Go 审计 · gaea 核心

> 审计对象：`internal/gaea/`（排除 `agent/`、`tool/`、`memory/`）、`internal/core/`、`internal/util/`、`main.go`、`main_test.go`
> 仓库 HEAD：`a745999a chore: 源码包归档 v4.455.0`；审计期间 `git status` 仅 `.audit/` 未跟踪，工作树与 HEAD 一致（无并发改动）。
> 只读审计，未修改任何仓库文件（本报告除外）。未运行 `go build`。

## 范围与文件统计

| 项 | 值 |
|---|---|
| 范围内 .go 文件 | **410**（其中测试文件 225） |
| 范围内总行数 | **68,571**（含测试） |
| `internal/gaea` 纳入包数 | **58**（排除 agent/tool/memory 后） |
| `internal/core` | 1 包 / 8 文件（fs.go、jobs.go、modes.go、routing.go、sessionmode.go、types.go + 测试） |
| `internal/util` | 1 包 / 2 文件（util.go 333 行、util_test.go） |
| 根目录 | main.go（76 行）、main_test.go（36 行） |
| 取证语料 | 仓内全部 **1,825** 个 .go（含 internal/app、agent、tool、memory 等作为引用方）+ `frontend/src`、`frontend/wailsjs` **1,201** 个文件（Wails 字符串调用面） |
| 被统计的范围内定义 | 4,136（生产非测试 2,541） |

### 方法（每条发现都可复现）
1. 抽符号定义 → `rg -o -w -f names.txt`（范围内全部符号名，3,575 个）跑全仓 .go + 前端，得到每符号总出现次数；`refs = 总次数 − 定义处数`。
2. 逐命中分类：**代码/注释/字符串字面量/测试文件**四分（字符串命中视为反射风险，注释命中不计入真实引用）。
3. **D1 判定式**：`生产代码引用 = 0 且 测试引用 = 0 且 前端命中 = 0 且 字符串命中 = 0`。
4. Wails 风险：对每个候选再做一次 **不带词界** 的子串检索（`rg -F`）覆盖 `frontend/src`、`frontend/wailsjs`、`internal/app`、`scripts`。
5. 计数口径：`refs` 是**标识符名**级计数——同名多处定义（`Range`、`percentile`、`Preview`、`round4`、`Store`…）共享同一计数，故 D4/D8 条目里的 refs 只当“热度”参考；**D1 的零引用判定是按定义位置逐命中分类**得到的，不受同名干扰（例如 `DetectConflict` 与 `DetectConflicts`、`SessionFacts` 与 `PromoteSessionFactsTool` 已分别核对）。

### 结论速览（诚实版）
- **D1 死代码**：范围内**只有 14 个**真正零调用符号（约 149 行）。这个仓刚经过「512 条审计 50 批清账 + 零调用绑定删除 744→635」，所以存量已经很干净；剩下的 14 条是那轮清账的**漏网项**。
- **D3 注释掉的代码**：**0 处**（全范围 0 个 `/* */` 遗留块；`//`+Go 关键字连续 ≥2 行的只有 1 处，且是中文说明注释，非代码）。这一项无需动手。
- **D2 不可达/空错误处理**：**0 个真缺陷**。103 个“return 后还有语句”的粗筛候选在人工收紧后为 0（多行 return 续行，非不可达）；「函数只返回 error 却 `if err != nil { return nil }`」为 0；`if true`/常量条件为 0；相邻重复语句 1（测试文件）。
- **D4/D8 重复**：8 组真实重复（含 2 组跨包 17 行完全相同、1 组 14 行相同的双实现），4 组重复类型定义（其中 1 组应保留，见“不建议动”）。
- **D6 屎山**：6 个 >200 行函数、30 个 ≥80 行函数，`boot.Build` 559 行是全仓第一大；另有 1 类 ~20 处复制的 payload 解码样板。
- **D7**：真实未用参数 7 处；TODO/FIXME 全范围**只有 1 条**（在测试文件里）。

## Top 20 发现（按可删行数×安全度排序）

1. [P0] `internal/gaea/context/manager.go:132` `(*ContextManager).ActiveTools` — D1 — LOC≈3 — confidence:high
   - 证据：`rg -n -w ActiveTools` 全仓 .go 仅 1 命中 = 该定义行本身；`rg -F ActiveTools` 在 `frontend/src`、`frontend/wailsjs`、`internal/app`、`scripts` 0 命中。refs=0，无文档注释。
   - 建议：直接删方法（`provider.ToolSchema` 过滤能力已由 `identity.FilteredSchemas` 在别处调用）。
2. [P0] `internal/gaea/cache/compiler.go:53` `(*Compiler).WithInstructions` — D1 — LOC≈7 — confidence:high
   - 证据：仅 3 命中：`compiler.go:43`（注释“Call WithInstructions to add custom instructions”）、`:49`（doc）、`:53`（定义）。前端/`internal/app` 0 命中。
   - 建议：删方法 + 顺手改 `:43` 注释（否则注释指向不存在的 API）。
3. [P0] `internal/gaea/control/controller.go:1012` `(*Controller).HookRunner` — D1 — LOC≈1 — confidence:high
   - 证据：仅 `:1010` doc + `:1012` 定义；前端 0 命中；`hook.Runner` 只在本包内使用。
   - 建议：删 1 行方法（连同 doc）。
4. [P0] `internal/gaea/control/controller.go:867` `(*Controller).TCCAStats` — D1 — LOC≈25 — confidence:high
   - 证据：仅 `:865` doc + `:867` 定义；`rg -F TCCAStats` 前端/`internal/app`/scripts 0 命中。
   - 建议：删方法（`TCCAReport` 同批删，见第 5 条），`cache.Metrics` 数据仍可经别处取。
5. [P0] `internal/gaea/control/controller.go:895` `(*Controller).TCCAReport` — D1 — LOC≈6 — confidence:high
   - 证据：仅 `:893` doc + `:895` 定义；前端 0 命中。
   - 建议：与第 4 条成对删除。
6. [P0] `internal/gaea/cache/runtime.go:181` `(*RuntimeLayer).DetectConflict` — D1 — LOC≈15 — confidence:high
   - 证据：仅 `:178` doc + `:181` 定义。注意干扰项：`internal/app/memory_hub.go:168` 是 `ps.DetectConflicts(...)`（另一个方法，复数），`rg -F DetectConflict` 的子串命中即来自它，**不是**本方法。
   - 建议：删方法；若冲突检测确需保留，应改由调用方直接用 `DiffChildEdits`（同文件另一方法）。
7. [P0] `internal/gaea/cache/runtime.go:200` `(*RuntimeLayer).MergeChildEdits` — D1 — LOC≈26 — confidence:high
   - 证据：仅 `:197` doc + `:200` 定义；无测试引用。
   - 建议：删方法（fork/子代理合并路径当前未接线）。
8. [P0] `internal/gaea/control/controller_mcp.go:82` `(*Controller).ConfiguredMCPNames` — D1 — LOC≈11 — confidence:high
   - 证据：仅 `:80` doc + `:82` 定义；前端 0 命中。
   - 建议：删方法；`DisconnectedMCPNames` 只调用它，可同批删。
9. [P0] `internal/gaea/control/controller_mcp.go:96` `(*Controller).DisconnectedMCPNames` — D1 — LOC≈19 — confidence:high
   - 证据：仅 `:94` doc + `:96` 定义（其内部调用 `ConfiguredMCPNames`，属“死方法互相取暖”）。
   - 建议：与第 8 条一并删。
10. [P0] `internal/gaea/control/controller_memory.go:329` `(*Controller).SessionRemember` — D1 — LOC≈5 — confidence:high
    - 证据：仅 `:326` doc + `:329` 定义；前端 0 命中。
    - 建议：删方法（会话记忆写入仍有 `QueueMemory` 等在用路径）。
11. [P0] `internal/gaea/control/controller_memory.go:493` `(*Controller).SessionFacts` — D1 — LOC≈7 — confidence:high
    - 证据：仅 `:492` doc + `:493` 定义。干扰项：`internal/app/gaea_tools_diag_test.go:32` 的 `memory.NewPromoteSessionFactsTool()` 是子串命中，非本方法。
    - 建议：删方法，或（若前端记忆面板需要）改为 `internal/app` 直读 memory 并补绑定。
12. [P0] `internal/gaea/control/refs.go:112` `(*Controller).HasRefs` — D1 — LOC≈3 — confidence:high
    - 证据：仅 `:110` doc + `:112` 定义（`detectRefs` 另有真实调用者）。
    - 建议：删方法。
13. [P1] `internal/gaea/provider/provider.go:379` `(*StreamInterruptedError).Unwrap` — D1 — LOC≈6 — confidence:medium
    - 证据：`rg -n -w Unwrap` 全仓 .go 仅 1 命中 = 定义行；前端 0 命中。**但**它是 Go 错误链约定方法（`errors.Is/As` 会隐式使用），不是普通死代码。
    - 建议：先确认无 `fmt.Errorf("%w", streamErr)` 包装后再走 `errors.Is`；不确定就保留。不要自动化删除。
14. [P1] `internal/gaea/cost/cost.go:347/354` `SaveTx` + `saveTx` — D4/D5 — LOC≈0（勿删） — confidence:high
    - 证据：`SaveTx` refs=1（1 个外部调用方），`saveTx` refs=1（只被 `SaveTx` 调）。看似重复，实为**公开校验层 + 私有实现**的正当分层。
    - 建议：**不动**（此处列出是为了防止别的批量脚本按“只调用一次”误删 `saveTx`）。
15. [P0] `internal/gaea/costref/costref.go:422` + `internal/gaea/costinquiry/costinquiry.go:507` `round4` ×2 — D4 — LOC≈6 — confidence:high
    - 证据：两份函数体规范化后**完全相同**（`math.Round(v*1e4)/1e4` vs `math.Round(v*10000)/10000`），`round4` 该名字全仓 refs=12（两处定义共享该计数）；`cost.Round2` 已是既有单源（`cost/matchindex.go:133`，注释 `pricefeed.go:336` 记“审计 GA6-05 收敛”）。
    - 建议：round2 收敛那一轮漏了 round4；在 `cost` 增加 `Round4`（或 `RoundN`）单源，两处改调，删 2 个本地函数。
16. [P0] `internal/gaea/cost/cost.go:660` + `internal/gaea/knowledge/sqlite.go:180` `parseTagsJSON` ×2 — D4 — LOC≈8 — confidence:high
    - 证据：两函数体规范化后逐字相同（含 `raw == "[]"` 早退与 `_ = json.Unmarshal`），`parseTagsJSON` 该名字全仓 refs=5（两处定义共享该计数）。
    - 建议：抽到 `internal/util`（或 `strutil`）作为 `ParseJSONStringSlice`，删 1 份。
17. [P0] `internal/gaea/costimport/costimport.go:1075` `hasHan` + `internal/gaea/knowledge/search.go:212` `isCJStr` — D4 — LOC≈8 — confidence:high
    - 证据：两函数体规范化后相同（`unicode.Is(unicode.Han, r)` 循环），refs 各 2。
    - 建议：抽 `strutil.HasHan` 单源，删 1 份。
18. [P0] `internal/gaea/config/render.go:299` `sortedSpaceProfileKeys` + `internal/gaea/contextview/imgrefs.go:130` `sortedKeys` — D4 — LOC≈8 — confidence:high
    - 证据：8 行函数体规范化后相同（`make([]string,0,len(m))` + `sort.Strings`），refs 分别 2 / 7。
    - 建议：抽 `strutil.SortedKeys`（泛型版更好），删 1 份。
19. [P0] `internal/gaea/context/store.go:39` `(*MemoryStore).Range` + `internal/gaea/context/store_file.go:50` `(*FileStore).Range` — D4 — LOC≈14 — confidence:high
    - 证据：14 行函数体规范化后完全相同（含 `start<0`/`end>len` 夹紧与 `start>=end` 提前返回）；`Range` 该名字全仓 refs=56（多类型共享该计数）。
    - 建议：抽包内 `func sliceRange(msgs []Message, start, end int) ([]Message, error)`，两个方法体各留 1 行转调。
20. [P1] `internal/gaea/cost/priceband.go:146` `percentile` + `internal/gaea/costref/costref.go:444` `percentile` — D4 — LOC≈17 — confidence:high
    - 证据：17 行函数体规范化后相同（线性插值分位数）；该名字全仓 refs=16（两处定义共享该计数）。
    - 建议：抽到 `cost`（`Percentile` 导出）或公共数值工具；删 1 份。注意 `costref` 已 import `cost`（`round2` 收敛先例），依赖方向安全。

## 全量发现表

> 排序：D1 → D4/D8 → D5 → D7 → D6 → D2/D3。LOC 为“可删/可收敛行数”估算。

| 文件:行 | 符号 | 类型 | LOC | 置信度 | 证据 | 建议 |
|---|---|---|---|---|---|---|
| internal/gaea/context/manager.go:132 | `(*ContextManager).ActiveTools` | D1 | 3 | high | `rg -w ActiveTools` 全仓 1 命中（=定义）；前端/`internal/app` 0 命中 | 删除方法与 doc |
| internal/gaea/cache/compiler.go:53 | `(*Compiler).WithInstructions` | D1 | 7 | high | 仅 43/49 注释 + 53 定义；前端 0 命中 | 删除并修 `:43` 注释 |
| internal/gaea/cache/runtime.go:181 | `(*RuntimeLayer).DetectConflict` | D1 | 15 | high | 仅 178 doc + 181 定义；`memory_hub.go:168` 的 `DetectConflicts` 是另一方法 | 删除 |
| internal/gaea/cache/runtime.go:200 | `(*RuntimeLayer).MergeChildEdits` | D1 | 26 | high | 仅 197 doc + 200 定义；测试 0 命中 | 删除 |
| internal/gaea/control/controller.go:867 | `(*Controller).TCCAStats` | D1 | 25 | high | 仅 865 doc + 867 定义；前端 0 命中 | 删除 |
| internal/gaea/control/controller.go:895 | `(*Controller).TCCAReport` | D1 | 6 | high | 仅 893 doc + 895 定义；前端 0 命中 | 删除 |
| internal/gaea/control/controller.go:1012 | `(*Controller).HookRunner` | D1 | 1 | high | 仅 1010 doc + 1012 定义 | 删除 |
| internal/gaea/control/controller_mcp.go:82 | `(*Controller).ConfiguredMCPNames` | D1 | 11 | high | 仅 80 doc + 82 定义；仅有 DisconnectedMCPNames 内调 | 与下一行成对删 |
| internal/gaea/control/controller_mcp.go:96 | `(*Controller).DisconnectedMCPNames` | D1 | 19 | high | 仅 94 doc + 96 定义 | 删除 |
| internal/gaea/control/controller_memory.go:329 | `(*Controller).SessionRemember` | D1 | 5 | high | 仅 326 doc + 329 定义；前端 0 命中 | 删除 |
| internal/gaea/control/controller_memory.go:493 | `(*Controller).SessionFacts` | D1 | 7 | high | 仅 492 doc + 493 定义；app 测试中的 `PromoteSessionFactsTool` 为子串误命中 | 删除（或改由 app 直读 memory） |
| internal/gaea/control/refs.go:112 | `(*Controller).HasRefs` | D1 | 3 | high | 仅 110 doc + 112 定义；`detectRefs` 另有真实调用者 | 删除 |
| internal/gaea/provider/provider.go:379 | `(*StreamInterruptedError).Unwrap` | D1 | 6 | medium | `rg -w Unwrap` 全仓 1 命中（=定义）；但属 `errors.Is/As` 约定方法 | 人工确认无 `%w` 包装链后再删；默认保留 |
| internal/gaea/provider/retry.go:84 | `RetryPolicy`（type） | D1 | 8 | low | 仅 83 doc + 84 定义；`docs/archive/gaea3-review/05-engine-core.md:351` 与 `docs/gaea-dsh-016-*.md:43` 提到它（文档引用不算代码引用） | 导出类型 + 文档承诺，需人工拍板；删则同步改 2 处文档 |
| internal/gaea/context/manager.go:90 | `(*ContextManager).Fork(mode, taskPrompt)` 的 `taskPrompt` | D7 | 1 | high | 函数体 18 行内 `taskPrompt` 0 次出现（grep `\btaskPrompt\b` 仅在签名） | 删参数（3 个调用点同步）或接上真实用途 |
| internal/gaea/control/controller.go:463 | `(*Controller).runTurnWithRaw(ctx,input,raw)` 的 `raw` | D7 | 1 | high | 463–560 行内 `raw` 只出现在 528 行注释；调用点 `:443`/测试都传了值 | 核对是否是漏接线（raw 提示词未使用）；确认后删参数 |
| internal/gaea/control/controller.go:838 | `(*Controller).Compact(ctx, instructions)` | D7/D5 | 7 | high | 两参数在 838–844 体内 0 引用；注释自称“exists for API compatibility”；前端命中 16（Wails 绑定） | 保留方法签名（绑定面），体内可仅留 notice；参数名改 `_` |
| internal/gaea/control/dream.go:310 | `(*Controller).Dream(ctx)` 的 `ctx` | D7 | 1 | medium | 310–343 体内无 `ctx`（无取消传播） | 接上 `ctx` 取消，或改 `_` 并在 doc 说明 |
| internal/gaea/control/dream.go:347 | `(*Controller).Distill(ctx)` 的 `ctx` | D7 | 1 | medium | 347–364 体内无 `ctx` | 同上 |
| internal/gaea/pricefeed/pricefeed.go:127 | `decodeHTML(body, contentType)` 的 `contentType` | D7 | 1 | high | 127–134 体内无 `contentType`（只用 `utf8.Valid` + 内容探测） | 删参数 |
| internal/gaea/sessionquery/tool.go:40 | `(searchTool).Execute(ctx,args)` 的 `ctx` | D7 | 1 | low | 体内无 `ctx`；工具接口要求 ctx 形参 | 保留（接口约定），可用 `_` |
| internal/gaea/wspath/skipdirs_test.go:12 | `// TODO(主代理收线)：...` | D7 | 1 | medium | 全范围唯一 TODO/FIXME（`-cmatch` 且注释上下文） | 需人工确认：属跨包待办，交接给 `internal/app` 线 |
| internal/gaea/costref/costref.go:422 | `round4` | D4 | 3 | high | 与 `costinquiry/costinquiry.go:507 round4` 规范化函数体逐字相同；`round4` 该名字全仓 refs=12（两处定义共享计数） | 收敛到 `cost.Round4` 单源 |
| internal/gaea/costinquiry/costinquiry.go:507 | `round4` | D4 | 3 | high | 同上 | 删除，改调单源 |
| internal/gaea/cost/cost.go:660 | `parseTagsJSON` | D4 | 8 | high | 与 `knowledge/sqlite.go:180` 同体；该名字全仓 refs=5（两处定义共享） | 抽 `util.ParseJSONStringSlice` |
| internal/gaea/knowledge/sqlite.go:180 | `parseTagsJSON` | D4 | 8 | high | 同上 | 删除，改调单源 |
| internal/gaea/costimport/costimport.go:1075 | `hasHan` | D4 | 8 | high | 与 `knowledge/search.go:212 isCJStr` 同体；refs 各 2 | 抽 `strutil.HasHan` |
| internal/gaea/knowledge/search.go:212 | `isCJStr` | D4 | 8 | high | 同上 | 删除，改调单源 |
| internal/gaea/config/render.go:299 | `sortedSpaceProfileKeys` | D4 | 8 | high | 与 `contextview/imgrefs.go:130 sortedKeys` 同体（`make`+`sort.Strings`） | 抽 `strutil.SortedKeys` |
| internal/gaea/contextview/imgrefs.go:130 | `sortedKeys` | D4 | 8 | high | 同上 | 删除，改调单源 |
| internal/gaea/context/store.go:39 | `(*MemoryStore).Range` | D4 | 14 | high | 与 `context/store_file.go:50 (*FileStore).Range` 14 行同体；`Range` 该名字全仓 refs=56（多类型共享计数） | 抽包内 `sliceRange` |
| internal/gaea/context/store_file.go:50 | `(*FileStore).Range` | D4 | 14 | high | 同上 | 删除实现，转调共享函数 |
| internal/gaea/cost/priceband.go:146 | `percentile` | D4 | 17 | high | 与 `costref/costref.go:444 percentile` 17 行同体；该名字全仓 refs=16（两处定义共享计数） | 抽 `cost.Percentile` 单源 |
| internal/gaea/plugin/procattr_windows.go:13 | `hideProcessWindow` | D4 | 7 | medium | 与 `proc/hide_windows.go:12 hideWindow` 7 行同体（`SysProcAttr.HideWindow` + `createNoWindow`）；两包各有 `_other` 空实现 | 合并到 `proc`（检查 `plugin` 是否已 import `proc`，避免新依赖环） |
| internal/gaea/proc/hide_windows.go:12 | `hideWindow` | D4 | 7 | medium | 同上；`proc/proc.go:9 HideWindow` 已是导出包装 | 保留 `proc` 为单源 |
| internal/gaea/contextview/fold.go:235,302,329,351,366,579 | `if err := json.Unmarshal(e.Payload,&p); err != nil { return }` ×6 | D4 | 12 | high | 同型样板 6 处；`contextview/fold_timing.go:64,78` 再 2 处；`trajectory/fold.go:150,194,211,263,301,340,364,386,411,437` 共 10 处；`trajectory/deliverable.go:74,90` 2 处 → 全范围 ~20 处 | 抽泛型 `payloadOf[T](e) (T,bool)`（Go 1.26 泛型），每处省 ~2 行并统一“坏 payload 静默跳过”语义 |
| internal/gaea/costimport/costimport.go:60 | `Preview` | D8 | 0 | medium | 与 `knowledgeimport/knowledgeimport.go:50 Preview` **字段签名完全相同**（6 字段、同序同名）；`Preview` 该名字全仓 refs=218、前端 feRefs=553（两处定义共享该计数，JSON 契约直连前端） | 仅建议**共用字段结构**（如提出 `importpreview.Preview`）并显式保留各自 json tag；不要盲合，前端契约需同步 |
| internal/gaea/retrieval/embed.go:20 | `Embedder` | D8 | 10 | medium | 与 `retrieval/rerank.go:27 Reranker` 字段布局完全一致（BaseURL/Model/client/mu/checkedAt/available） | 抽共享内嵌结构 `localHTTPClient`（含 client/mu/checkedAt/available + 健康检查），两个类型各留 3 行；语义不同，勿并为同一类型 |
| internal/gaea/browser/cdp.go:33 | `cdpError` | D8 | 2 | low | 与 `plugin/plugin.go:665 rpcError` 同为 2 字段错误体（分属两协议域） | 不建议合并（协议不同）；仅登记为已知重复 |
| internal/gaea/skill/skill.go:77 | `Options` / `Store` | D8 | 0 | low | 5 字段同名同型、仅可见性不同（导出配置 vs 私有状态） | **不建议动**：标准 opts→resolved 模式，非重复缺陷 |
| internal/gaea/control/controller_memory.go:600 | `slugifyName` | D5 | 3 | high | 单语句 `return strutil.TitleSlug(name)`，refs=1 | 内联到唯一调用点，删函数 |
| internal/gaea/trajectory/agentnet.go:166 | `aggregateTool` | D5 | 3 | high | 单语句 `aggregateToolStats(node, t)`，refs=1 | 内联到唯一调用点 |
| internal/gaea/costimport/costimport.go:1232 | `round3` | D5 | 3 | high | 单语句 `math.Round(v*1000)/1000`，refs=1；与 round2/round4 家族同型 | 并入 `cost.RoundN` 家族 |
| internal/gaea/pricefeed/pricefeed.go:338 | `round2` | D5 | 1 | high | 单语句转发 `cost.Round2(v)`（注释即“审计 GA6-05 收敛”） | 直接调 `cost.Round2`，删包装 |
| internal/gaea/permission/bash_readonly.go:105 | `containsShellSyntax` | D5 | 3 | high | 单语句 `strings.ContainsAny(...)||strings.Contains(cmd,"$(")`，refs=1 | 内联到唯一调用点 |
| internal/gaea/retrieval/semantic.go:33 | `formatPrice` | D5 | 3 | high | 单语句链式 `TrimRight(FormatFloat)`，refs=1（前端另有同名 TS 函数，无绑定关系） | 内联或移入 `strutil` |
| internal/core/jobs.go:21 | `NewJobManager` | D5 | 3 | high | refs=1，唯一调用点 `internal/app/office_handler.go:8`；仅做 make(map) + 赋值 | 可内联（保留亦可，绑定面经 `OfficeGetJobState`） |
| internal/core/sessionmode.go:11 | `NewSessionModeStore` | D5 | 3 | high | refs=1（`internal/app/office_handler.go:9`） | 同上 |
| internal/gaea/config/config.go:713 | `(e PluginEntry).ShouldAutoStart` | D5 | 3 | medium | refs=1，单语句 `e.AutoStart == nil \|\| *e.AutoStart`；导出但前端 0 命中 | 可内联；导出符号建议保留（配置语义自解释） |
| internal/util/util.go:235 | `util.Max` / `util.Min` | D5/D4 | 14 | medium | 名字 refs：`Max`=77、`Min`=63（手写两数比较）；Go 1.26 内建 `max`/`min` 已可替代 | 分批替换为内建（77+63 个调用点）→ 删 14 行；建议单独立项，不与本次小改混跑 |
| internal/util/util.go:263 | `MustMarshal` / `MustMarshalCompact` | D4 | 15 | medium | 两个 package 级函数共享同一 `panic("util.Must...")` 结构（Indent vs 紧凑），名字 refs 14/10 | 抽 `mustJSON(v, indent bool)`，删 ~7 行 |
| internal/util/util.go:254 | `TruncateRef` | D1-adjacent | 8 | low | 名字 refs=4，但 `util.TruncateRef` 限定名 0 命中（调用方经别名/本地转发） | 需人工确认调用方；勿盲删 |
| internal/gaea/archive/archive.go:169 | `(*Store).ListRecentSessions` | D1-test-only | 66 | high | 生产代码 0 引用，仅 `archive_test.go:169` 调用 | 若不打算接回产品面：方法 + 测试一并删（-66/-? 行） |
| internal/gaea/cache/runtime.go:235 | `(*RuntimeLayer).UpdateExecution` | D1-test-only | 5 | high | 仅 `runtime_test.go:112` 调用 | 同上一并评估 |
| internal/gaea/cache/runtime.go:253 | `(*RuntimeLayer).AppendHint` | D1-test-only | 8 | high | 仅 `runtime_test.go:58` 调用 | 同上 |
| internal/gaea/context/flow.go:108 | `(*FlowLayer).SetDetailDir` | D1-test-only | 3 | high | 仅 `flow_test.go:226` 调用 | 同上 |
| internal/gaea/context/flow.go:212 | `(*FlowLayer).SetCompactPolicy` | D1-test-only | 1 | high | 仅 `flow_test.go:214` 调用 | 同上 |
| internal/core/modes.go:52 | `ClearAllPersistedModes` | D1-test-only | 1 | medium | 仅 `internal/core/modes_test.go:61` 调用；导出且属 office 域 | 需 `internal/app`/office 线确认后再删 |
| internal/gaea/event/event.go:106 | `UsageSourceMain` / `UsageSourceExecutor` | D1-test-only | 2 | medium | `UsageSourceMain` 仅被 `internal/app/gaea_session_stats_test.go:38` 引用；`UsageSourceExecutor` 0 生产引用（同类 var 块） | 常量块整体人工确认（可能前端按字符串取值） |
| internal/gaea/boot/boot.go:108 | `Build` | D6 | 559 | high | 全仓最长函数；含 ≥30 段 `// S1.3-A / V10.22 / v4.221 / GA4-06 …` 分节注释 | 按分节拆 8 个 helper（模型/空间解析、事件日志 sink、provider+effort、sysprompt、plugins+host、permission、hooks、tools+task+cleanup），各返回局部 struct；目标 `Build` ≤80 行 |
| internal/gaea/control/controller_submit.go:39 | `(*Controller).dispatchSlash` | D6 | 268 | high | 该函数体内 `case` 20 个（`39–306`），每 case 各写一遍参数解析+回执 | 改表驱动 `map[string]slashHandler`（handler 各自小文件），主分发 ≤30 行 |
| internal/gaea/skill/builtins.go:18 | `builtinSkills` | D6 | 239 | high | 6 个内建 skill 定义内联长文本（`blocks=6`） | prompt 正文改 `//go:embed` 的 .md（`prompts/` 已有目录），函数降到 ≤40 行 |
| internal/util/util.go:107 | `extractBalancedJSON` | D6 | 227 | high | 单函数 227 行；同文件 `isPreferredCandidate` 197 行；两者通过 `scanBalanced`(159)/`extractFromJSONFence`(193) 耦合 | 拆成 `jsonscan` 扫描器（配平）+ `jsonprefer` 评分规则表；`ExtractJSON` 保持对外签名 |
| internal/util/util.go:137 | `isPreferredCandidate` | D6 | 197 | high | 同上；函数内是长串启发式优先级判断 | 抽“候选特征 → 权重”表驱动，或按 4 类（对象/数组/工具调用/文本）拆子函数 |
| internal/gaea/config/render.go:20 | `RenderTOML` | D6 | 213 | high | 单函数 213 行；`render_preserve.go:338 mergeOwnedSections` 110 行与之配套 | 渲染拆为「段收集 → 保序合并 → 文本输出」三步；`mergeOwnedSections` 独立成 `merge.go` |
| internal/gaea/cost/cost.go:511 | `(*Store).Search` | D6 | 148 | high | 148 行；同文件 `Categories` 98、`SaveCategory` 92、`saveTx` 长 SQL | `cost` 包按 `store_tx.go / store_query.go / store_category.go` 拆文件；SQL 常量集中 |
| internal/gaea/cost/repair.go:35 | `legacyCategoryTarget` | D6 | 136 | medium | 136 行纯迁移映射；`repair.go:299 RepairCategoryPaths` 108 行 | 映射表数据化（map 或内嵌 JSON），函数降到 ≤40 行 |
| internal/gaea/boot/sysprompt.go:43 | `buildSystemPrompt` | D6 | 131 | medium | 131 行装配；`boot/space_assembly_test.go` 已按分节断言 | 按「身份/项目画像/技能索引/空间定制」拆 4 段 helper，测试可保持 |
| internal/gaea/backup/backup.go:119 | `(*Plan).Create` | D6 | 129 | medium | 129 行 + 同文件 `ApplyPending` 116 行 | 拆 zip 写入（`addFileToZip` 已有）与清单生成两段 |
| internal/gaea/control/controller_approval.go:233 | `(*Controller).promptApproval` | D6 | 125 | medium | 125 行，含前端交互+超时+拒绝分支 | 拆 `approvalRequest` 构造 / 等待 / 回写三步 |
| internal/gaea/trajectory/agentnet.go:12 | `FoldAgentNetwork` | D6 | 118 | medium | 118 行折叠聚合 | 与 `trajectory/fold.go` 共用折叠框架，按事件类型分派 |
| internal/gaea/control/dream.go:49 | `(*Controller).dreamText` | D6 | 113 | low | 113 行；同文件 `distillText` 89 行 | 两函数共用「取会话→摘要→产文本」骨架，可抽公共前置 |
| internal/gaea/control/controller.go:463 | `(*Controller).runTurnWithRaw` | D6+D7 | 86 | medium | 86 行；且 `raw` 参数未用（见上） | 拆「会话状态标记 / ContextManager 首轮 / executor 执行」三段 |
| internal/gaea/config/render_preserve.go:338 | `mergeOwnedSections` | D6 | 110 | medium | 110 行保序合并状态机 | 独立的 `merge.go` + 表驱动指令集 |
| internal/gaea/wssearch/wssearch.go:218 | `Search` | D6 | 108 | medium | 108 行；同包 `runeCount`(144) 是与 util 重复的小工具 | 拆「过滤/打分/截断」三段；`runeCount` 并入 `strutil` |
| internal/gaea/filewatch/filewatch.go:162 | `(*Watcher).loop` | D6 | 109 | low | 109 行（含平台分支） | 可拆事件去抖与派发；风险中等，建议最后动 |
| internal/gaea/tasks/tasks.go:934 | `(*Manager).execute` | D6 | 91 | low | 91 行；同文件 `mupdateProgress`(1108)/`mupdateResult`(1136)/`mrequeue`(1167)/`mmustGet`(1202) 均为 ≤5 行私有小方法 | 建议：把 4 个小方法合并进调用点或统一 `mutate(id, fn)`；`execute` 拆状态机步骤 |
| internal/gaea/costinquiry/scan.go:41 | `(*Store).ScanAnomalies` | D6 | 141 | medium | 141 行；阈值常量 `spreadWatch/spreadAlarm/jumpWatchPct/jumpAlarmPct`（:27-30）只在函数内用 | 阈值改入配置/参数；函数拆「取数/判异/落库」 |
| internal/gaea/control/controller.go:838 | `(*Controller).Compact` | D2 | 7 | high | 函数体：`if c.executor == nil { return nil }` + `notice(...)` + `return nil`，参数全未用 | 不是缺陷（V5.0 起自动截断），但应把 doc 里的“API compatibility”写明或加 `// Deprecated:` |
| internal/gaea/contextview/fold.go:235 等 | `json.Unmarshal` 后直接 `return` 的样板 | D2 | 0 | high | 20 处同型；均为“坏 payload 静默跳过”，非不可达代码 | 语义正确，归入上面的泛型 helper 收敛即可 |
| internal/gaea/archive/archive.go:64,70 | `if err != nil { return }` | D2 | 0 | high | 所在函数无 error 返回值（best-effort 写入） | 保持；不要“补错误处理” |
| internal/gaea/costinquiry/costinquiry.go:98-99 | `_, _ = gdb.Exec(createTableSQL/createIndexSQL)` | D2 | 0 | medium | 建表错误被显式丢弃 | 建议至少 `slog.Warn`（不是死代码） |
| internal/gaea/bm25/cache_test.go:36 | 相邻重复语句 | D2 | 1 | low | 全范围唯一 `dup-stmt` 命中，位于测试 | 忽略 |
| （全范围） | 注释掉的代码块 / `/* */` 遗留块 | D3 | 0 | high | `rg '^\s*//\s*(func \|if .*\{$\|return \|for .*\{$\|var \|const \|type \|\}$)'` 在范围内仅 3 处散文注释；`/\*[\s\S]*?\*/` 在 `internal/core`、`internal/util` 0 命中；连续 ≥2 行“代码样”注释仅 `controller_submit.go:303`（中文说明） | 无需动作 |

## 屎山清单（结构性重构建议，含风险）

1. **`boot.Build` 559 行（`internal/gaea/boot/boot.go:108`）** —— 全仓第一。它按注释已有 ~30 个分节（`S1.3-A 空间模型`、`v4.61 子代理 transcript 落盘`、`3.0 Step 1 事件日志`、`S1.5-A 权限按空间`、`GA4-06 hook 与 bash 同源`…）。
   - 方案：`type buildState struct{...}` + 8 个 helper（`resolveModels` / `newEventSink` / `newProviderWithEffort` / `assembleSysPrompt` / `assemblePlugins` / `assemblePermissions` / `assembleHooks` / `assembleTools`），`Build` 只做编排 + `defer cleanup`。
   - 风险：中。`boot` 是装配中心，`space_assembly_test.go`、`pinned_facts_assembly_test.go` 覆盖了部分分节，可作为重构护栏；**顺序敏感项**（effort 必须先于 NewProvider、sink 必须共享同一同步实例）必须在提取时保持调用序，建议逐节搬迁 + 每步跑包内测试。
2. **`dispatchSlash` 268 行 / 20 个 case（`internal/gaea/control/controller_submit.go:39`）** —— 巨型 if/else 链的 switch 变体。
   - 方案：`var slashHandlers = map[string]func(c *Controller, args []string) error{...}`，每个命令一个文件/函数；`/model`、`/skill`、`/hooks`、`/mcp` 的补全项已各有 `xxxArgItems` helper（`slash.go:63/104/119/138`），正好与 handler 同处。
   - 风险：低-中。纯分发重构，行为测试（`controller_submit` 相关测试）可覆盖。
3. **`internal/util/util.go` 的 JSON 抽取双巨函数（227 + 197 行）** —— 全仓 JSON 抽取的唯一入口（`ExtractJSON` refs=46），因此**不能删，只能拆**。
   - 方案：`internal/util/jsonscan`（配平扫描：`scanBalanced`/`extractBalancedJSON`）+ 评分规则表（`isPreferredCandidate` 的优先级改为有序规则切片）；`ExtractJSON` 签名不变，46 个调用点零改动。
   - 风险：中-高（已被 46 处依赖，且 `util_test.go` 有 7 个用例）。建议先补「多候选选择」的表驱动测试再动。
4. **~20 处重复的“payload 解码 → 失败即 return”样板**（`contextview/fold.go`、`contextview/fold_timing.go`、`trajectory/fold.go`、`trajectory/deliverable.go`）。
   - 方案：Go 1.26 泛型 `func payloadOf[T any](e Event) (T, bool)`，集中“坏 payload 静默跳过”策略（现在是 20 份隐式约定，将来改策略要改 20 处）。
   - 风险：低。是本次性价比最高的结构化收敛。
5. **`cost` 系列 6 包的同构体**（`cost/costproject/costref/coststage/costinquiry/pricefeed` 各有 `Store` + `Open(gdb)` + `requireDB()` + `roundN`）。`Store` 类型名在仓内出现 642 次（多包同名），`round4`/`percentile`/`parseTagsJSON` 都在包间重复。
   - 方案：抽 `internal/gaea/coststore`（`Base` 内嵌：db 句柄 + `requireDB` + `Percentile`/`RoundN`），各包 `Store` 内嵌 `coststore.Base`。
   - 风险：中-高（涉及 SQL schema 与前端契约），建议独立批次 + 全量回归，不在小改里做。
6. **`skill/builtins.go:18 builtinSkills` 239 行内联 prompt** —— 与 `prompts/` 目录、`go:embed`（main.go 已有 `embed.FS` 先例）重复劳动。
   - 方案：正文移 `prompts/builtin/<name>.md` + `go:embed`，函数只留元数据表。
   - 风险：低（测试 `skill_template_kind_test.go` 等按 kind 断言，不受文本位置影响）。
7. **`tasks.Manager` 的 4 个子 5 行私有方法**（`mupdateProgress:1108`、`mupdateResult:1136`、`mrequeue:1167`、`mmustGet:1202`）：命名风格与 Go 惯例不符（`m` + 小写动词连写），且都是「加锁 → 取 map → 改字段」同型。
   - 方案：`func (m *Manager) mutate(id string, fn func(*task) ) bool` 单点加锁，删 4 个包装。
   - 风险：低-中（并发语义必须逐行核对加锁边界）。

## 不建议动的地方（说明为什么）

1. **Wails 绑定面方法**：`(*Controller).Compact`（前端命中 16）、`SetSubagentModel`（7）、`Fork`（7）、`formatPrice`（前端 17，但那是 TS 同名函数）等，`refs` 低≠死。判定必须看 `feRefs` 列；本报告所有 D1 判定都已过 `frontend/src` + `frontend/wailsjs` 的**不加词界**子串检索。`internal/app/bindings_completeness_test.go` 用 `reflect` 枚举 `App` 导出方法并与前端绑定表比对，**新增/删除导出方法可能直接让该测试红**——所以 14 条 D1 全在 `internal/gaea` 内部类型上（不是 `App` 方法），删它们不会动绑定表；但若要顺带删 `App` 上的转发方法，必须同批更新前端绑定表。
2. **`internal/core` 整包**：初看像“孤儿包”（`execReadText`/`execWebSearch` 等 refs=1），实际由 `internal/app/office_handler.go:8-25` 直接使用（`officecore.Execute/IsTask/NewJobManager/NewSessionModeStore/GetState`）并经 `OfficeExecute`/`OfficeIsTask`/`OfficeListFolder`/`OfficeReadFile`/`OfficeGetJobState` 暴露给前端。**不要从本域视角判它死**。
3. **`util.EstimateTokens` vs `util.EstimateTokensLen`**：`util.go:295-299` 明确写了「三家族服务不同面，跨型不许顺手对齐」（语言学口径 vs bytes÷4 floor vs bytes÷4 ceil）。这是**有意分歧**，不要合并。
4. **`util.Truncate`**：`util.go:228-232` 注明“实现已下沉到 `internal/gaea/strutil`（X1-06 单一源），此处只保留导出签名”。保留 shim。
5. **`cost.Store.SaveTx` / `saveTx`**：公开校验层 + 私有实现，不是重复。
6. **`skill.Options` / `Store`**：导出 opts 与私有 resolved 状态的标准镜像，字段同名是特性不是缺陷。
7. **tag/反射驱动结构**：`config.*`（toml 标签）、`costimport.Preview`/`knowledgeimport.Preview`（JSON 直连前端，`feRefs` 551+）、`event.*`、`trajectory/fold` 的 payload 结构、`internal/core/types.go` 的 `AgentJobState/AgentTaskResult`——字段级“grep 不到”不能证明未使用，需按 `json:"..."` 反查前端。
8. **`provider.RetryPolicy`**：虽然代码零引用，但它是导出类型且被 `docs/archive/gaea3-review/05-engine-core.md:351` 与 `docs/gaea-dsh-016-*.md:43` 描述为对外重试策略 API。删或不删需产品拍板（本次标 low 置信度）。
9. **`internal/gaea/fileutil/encoding`、`internal/gaea/proc` 的平台对偶文件**（`hide_windows.go` / `hide_other.go`、`procattr_windows.go` / `procattr_other.go`）：`_other` 只有 1 行空实现是**构建标签必需**，不是“无用包装”。
10. **`internal/gaea/plugin/procattr_windows.go` 与 `internal/gaea/proc/hide_windows.go` 的合并**：需先确认 `plugin` → `proc` 的依赖方向（避免环），否则保持现状更安全。

## 附：本次审计的坑（供主控参考）
- 用「符号名出现次数」判死时，**Go 文档注释会把死函数救活**（`// Foo does ...`）。必须把注释/字符串命中单独分类，否则 14 条里会漏掉 12 条。
- 用「带词界」的 grep 验 Wails 风险会**漏掉子串调用形态**，且会**误报子串**：`DetectConflict` 被 `DetectConflicts` 命中、`SessionFacts` 被 `PromoteSessionFactsTool` 命中。两向都要做（`-w` 与 `-F` 各一次）。
- `git status` 干净 + HEAD 未变，说明审计期间**没有并发写入**；若后续看到符号“消失”，先怀疑自己的抽取/展示拼接（本次一度把 `recv` 与 `name` 字段拼成了 `cmActiveTools` 这种不存在的符号名，已修正）。
