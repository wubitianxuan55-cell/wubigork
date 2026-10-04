# Go 审计 · agent/tool/memory

审计对象：Gaea（`github.com/gaea/gaea`，Go 1.26，Wails v2 桌面应用）
审计范围：`internal/gaea/agent/`、`internal/gaea/tool/`、`internal/gaea/memory/`
审计方式：**只读**。未修改、未新建、未删除任何仓库源文件；仅生成本报告。
本报告未执行 `go build ./...`，未执行任何 git 写操作。

---

## 范围与文件统计

| 包 | .go 文件 | 非测试 | 测试 | 磁盘字节 | 说明 |
|---|---|---|---|---|---|
| `internal/gaea/agent` | 151 | 67 | 84 | 914,672 | 含子包 `budget/ cache/ render/ session/ testutil/ textutils/ toolguard/` |
| `internal/gaea/tool` | 83 | 45 | 38 | 552,580 | 含子包 `builtin/`（46 个内建工具） |
| `internal/gaea/memory` | 62 | 28 | 34 | 350,731 | SQLite / 文件双后端 |
| **合计** | **296** | **140** | **156** | **1,817,983** | |

统计口径与验证手段（全部可复现）：

- 引用计数：对全仓 **1825** 个 `.go` 文件做「注释 + 字符串字面量剥离」后的标识符计数，剥离后**仅在注释/文档里出现过的名字计为 0 引用**；再对每条结论用原文 grep 复核一次（两层交叉验证，防剥离误判）。
- 提取声明 **2332** 条（func / method / type / var / const）。
- Wails 绑定风险：仓库生成物 `frontend/wailsjs/**` 已全量 grep（这是唯一把 Go 方法名以字符串暴露给前端的地方，形如 `window['go']['app']['OfficeB']['GaeaSummarizeFrom']`）。**本范围内所有 32 个零引用符号在 `frontend/wailsjs` 中 0 命中**，唯一例外见 P0-2（绑定存在但实现是拒绝桩）。
- 注册表风险：`tool.RegisterBuiltin` 全仓仅出现在 `internal/gaea/tool/builtin/*.go`（47 处，对应 46 个工具名）；`builtins` map 只有 `LookupBuiltin`/`Builtins` 两个读取口。
- 接口实现风险：凡「无人调用」的方法，均先确认是否满足 `tool.Tool` / `ContextualTool` / `PersistWriteTool` / `ModelBackedTool` / `SpaceTaggedTool` / `SessionStateResetter` / `Backend` 等接口签名；**已据此排除全部接口方法**（例如 `ContextualTool` 在 `execute_one.go:176` 有断言，故 `TaskTool.ExecuteWithContext` 判为存活）。

### 两个重要的「阴性」结论（先排除误报面）

1. **没有「注册了却永不执行」的内建工具。** `reasonix.toml:32` 为 `enabled = []`，而 `boot.go:761` 对空列表的语义是「注册全部内建」，`builtin.AllowsSpace` 对 46 个工具全部放行；`internal/gaea/config/config.go:763-769` 的 13 项默认清单会被 `reasonix.toml` 覆盖。`spacetags.go` 的 **64 个键逐一核对，全部有真实 `Name()` 实现对应**（含桌面端 ExtraTools：`image_gen/ocr/semantic_search/fact_*/routine_llm/translate_text` 等），无一行为死键。agent 侧元工具（`ask`/`task`/`exit_plan_mode`/`request_permission`）虽不经 `RegisterBuiltin`，但都在 `boot.go:295/361/370/377` 被构造注册。
2. **D3（注释掉的代码块）在本范围内基本不存在。** 全量扫描 ≥2 行连续「像 Go 代码的 `//` 块」，仅 2 处，且都是**合法用法**：`agent/testutil/mock_provider.go:34-43` 是 godoc `Usage:` 示例；`agent/session/log_bench_test.go:46` 是基准测试内的说明。另有 1 处 3 行中文 `// 临时工作稿不参与对标`（`cost_indicators.go:82`）属正常行尾注释。**建议：本范围不要以「删注释代码」为主要收益。**

---

## Top 20 发现（按 可删行数 × 安全度 排序）

> LOC 为「函数体 + 紧邻文档注释」的可删行数估算（由括号配对精确测量，非目测）。

1. **[P0] `internal/gaea/memory/graphstore.go:175` `GraphNeighbors`** — D1 — LOC≈83 — confidence:high
   - 证据：grep `GraphNeighbors` 全仓 5 命中 = `graphstore.go:171`(文档注释)、`:175`(定义)、`graph_test.go:238/252/256`(测试)。**生产代码 0 调用**；`frontend/wailsjs` 0 命中。
   - 建议：整函数删除（含 83 行）；若图查询是未接线功能，移到独立提交并附后续接线计划，不要留在主文件里。同时删除 `graph_test.go` 3 处断言或改为跳过。

2. **[P0] `internal/gaea/agent/compact.go:353,380` `SummarizeFrom` + `SummarizeUpTo`** — D1 — LOC≈56 — confidence:high ⚠️Wails
   - 证据：grep 两者全仓仅命中定义行与各自文档注释（`compact.go:352-353`、`379-380`）。**Wails 绑定确实存在**（`frontend/wailsjs/go/app/OfficeB.d.ts:343/345`），但实现是拒绝桩：`internal/app/gaea_ui.go:723-726` 注释「摘要回退暂未支持」、函数体 `return errSummarizeUnsupported`，**从不调用 AgentRunner 的这两个方法**。
   - 建议：二选一——(a) 前端保留按钮则删除 `gaea_ui.go` 的桩 + wailsjs 重新生成，改由 `CompactNow` 承担；(b) 确认不再支持则删除这两个方法（56 行）。**注意这是前端可见行为，需产品确认后再动**，故列 P0 但在「不建议单方面删除」清单里也出现。

3. **[P0] `internal/gaea/agent/tool_precheck.go:23-24` + `:114-149`** `case "delete_range"` 分支 + `precheckDeleteRange` — D1/D2 — LOC≈37 — confidence:high
   - 证据：`delete_range` **不是任何内建工具名**（46 个 `RegisterBuiltin` 中无此项；`tool.RegisterBuiltin` 全仓仅存在于 `tool/builtin/`）；`precheckDeleteRange` 全仓 8 命中 = 定义 + `tool_precheck_more_test.go` 6 处 + 死分支 1 处。生产路径 `precheckTool` 只可能被真实工具名调用。
   - 建议：删 `case "delete_range"` 两行 + 整个 `precheckDeleteRange`（36 行）；同步清理 `tool_precheck_more_test.go`。

4. **[P0] `internal/gaea/memory/memory.go:441` `Set.EpisodicMatches`** — D1 — LOC≈43 — confidence:high
   - 证据：grep 3 命中 = `memory.go:438`(注释)、`:441`(定义)、`recall.go:135`(注释里提到它)。**0 调用**。
   - 建议：删除；若「情景记忆按标签命中」要保留语义，应在 `recall.go` 里显式接线，否则是纯死代码。

5. **[P0] `internal/gaea/agent/subagent_store.go:705` `SubagentStore.CleanupStaleRunning`** — D1 — LOC≈34 — confidence:high
   - 证据：grep 2 命中 = `:702`(注释)、`:705`(定义)。0 调用、0 测试、`frontend/wailsjs` 0 命中。
   - 建议：删除。若「启动时清理 stale running」是设计意图，应该在 `boot` 里调用——现在这个意图没有任何执行点，`running` 记录会永久残留（这是**功能缺失**，不只是死码，值得单独提 issue）。

6. **[P1] `internal/gaea/tool/tool.go:354` `Registry.PersistWriteNames`** — D1 — LOC≈30 — confidence:high
   - 证据：grep 全仓 4 处调用点**全在测试**（`tool/registry_test.go:171/184`、`agent/task_test.go:325`、`tool/registry_race_test.go:76`）。生产代码用的是 `tool.IsPersistWrite(t)` 直接判定（`agent/task.go:392`）。
   - 风险点：`tool.go:79-80` 文档写「Sub-agent registry filtering and approval gates derive their forbidden/always-ask sets from this」——**该描述与实现不符**（没有任何生产调用者）。删除方法的同时必须修正 `tool.go:174-191` 的 Registry 文档（该文档把 `PersistWriteNames` 当作锁序设计的理由之一）。
   - 建议：删方法 + 改文档；`registry_test.go` 的对应断言改为直接测 `IsPersistWrite`。

7. **[P1] `internal/gaea/agent/reasoning_language.go` 语言偏好整族 6 函数 + 2 个 context key** — D1/D2 — LOC≈49 — confidence:high
   - 证据：`SetResponseLanguage`(:201)、`SetReasoningLanguage`(:207)、`WithResponseLanguagePreference`(:134)、`ResponseLanguageFromContext`(:142)、`WithReasoningLanguagePreference`(:155)、`ReasoningLanguageFromContext`(:163) 六者全文引用仅「定义 + 自身文档注释」；`responseLanguageContextKey`/`reasoningLanguageContextKey`(:8-9) 只被这四个 From/With 函数使用。全仓 grep `ResponseLanguage|ReasoningLanguage` 在 `frontend/` **0 命中**（前端无设置入口）。
   - 连带证据：`a.responseLanguage` / `a.reasoningLanguage` 两个 `atomic.Value` 字段（`agent.go:484/490`）**除了这两个死 setter 外无任何写入点**，故 `agent_run.go:53` 调用的 `withTurnPreferences`(:178-197) 恒为 no-op（两个 Load 恒返回 nil → 语言恒 "auto" → `WithResponseLanguage/WithReasoningLanguage` 返回原串）。**整个「运行时语言偏好」特性是死的**，但每次 user turn 仍要白跑一遍字符串拼接判断。
   - 建议：删除 6 函数 + 2 个 key + 2 个字段 + `withTurnPreferences` 及其唯一调用点（合计 ≈70 行）；保留 `NormalizeReasoningLanguage` 等被 `Options` 初始化路径使用的部分（需二次确认，见 D4-3）。

8. **[P1] `internal/gaea/tool/tool.go:296,316` `SuspendPrefix` / `ResumePrefix`（+ `Add` 里的 suspended 守卫）** — D1/D2 — LOC≈27 — confidence:high
   - 证据：`SuspendPrefix` 生产 0 调用（仅 `registry_race_test.go:65/120`、`158`）；`ResumePrefix` 生产 0 调用（仅 `registry_race_test.go:70/158`）。对比：`RemovePrefix` **有**生产调用（`control/controller_mcp.go:140,170`）。
   - **连带死分支**：`tool.go:226-230` 的 `for prefix := range r.suspended { ... }` —— 生产代码里只有 `SuspendPrefix` 会写 `r.suspended`，因此该循环在生产中**永不迭代**，`Add` 的「幽灵名规则」实际不生效（这条规则在 `tool.go:209-214` 文档里被当作重要不变量描述）。
   - 建议：删除 `SuspendPrefix`/`ResumePrefix`/`suspended` 字段 + `Add` 守卫；或反过来——如果 MCP 会话级禁用是真实需求，则在 `controller_mcp.go` 接上 Suspend/Resume（现在是**文档承诺 > 实现**）。

9. **[P1] `internal/gaea/agent/compact.go:348` `CompactNow`** — D1 — LOC≈4 — confidence:high
   - 证据：生产 0 调用（命中仅 `compact.go:347` 注释 + 定义）；测试 8 处（`compact_test.go`、`overflow_compact_test.go`）。`frontend/wailsjs` 0 命中。
   - 建议：删除；若「手动 /compact」是产品功能，则应接到 App 绑定（当前没有绑定，功能不可达）。

10. **[P1] `internal/gaea/memory/memory.go:409` `Set.ProceduralBlock`** — D1 — LOC≈30 — confidence:high
    - 证据：grep 2 命中 = `:407`(注释)、`:409`(定义)。0 调用、0 测试。
    - 建议：删除。

11. **[P1] `internal/gaea/agent/session/derived.go:17` `DeriveTitle`** — D1 — LOC≈23 — confidence:high
    - 证据：生产 0 调用（`frontend/wailsjs` 0 命中），测试 4 处（`derived_test.go`）。文件头注释自证：「3.0 Step 1: 派生 API……**供绑定层后续接线**」——绑定层从未接线。
    - 建议：删除函数与 `TitlePreviewMax`（若 `DeriveStats` 仍存活则保留该文件其余部分；`DeriveStats` prod=1，存活）。

12. **[P1] `internal/gaea/agent/compile.go:59` `DigestMessages`** — D1/D2 — LOC≈8 — confidence:high
    - 证据：全仓 5 命中中 4 处是注释（`compile.go:58` 文档、`:69` 说明、`:87` 说明「与 DigestMessages 逐元素等价」），第 5 处是定义。生产实际使用 `DigestCache.Digest`（`agent_stream.go:44` 走的是 `Digest`）。
    - 建议：删除 `DigestMessages`，`compile_test.go` 改测 `DigestCache.Digest`（两者语义等价，注释已声明）。

13. **[P1] `internal/gaea/memory/store.go:401` `NewDeliverable`** — D1 — LOC≈15 — confidence:high
    - 证据：grep 2 命中 = `:397`(注释)、`:401`(定义)。0 调用、0 测试。
    - 建议：删除（注释自称「审计 P1 AP4-06」引入，但从未被使用）。

14. **[P1] `internal/gaea/agent/stop_gate.go:117` `verifyGate`** — D1/D4 — LOC≈11 — confidence:high
    - 证据：生产 0 调用（仅 `stop_gate_test.go:205/218/221`）。**且逻辑与活代码重复**：`taskGate()` 的 `stop_gate.go:34-41` 内联了完全相同的四步（查 flag → 置位 → `session.Add(provider.Message{RoleUser, stopGateOrchestrateVerifyNudge})` → return true）。
    - 建议：删除 `verifyGate`，把 `taskGate` 内联段改为调用它或反之——二选一，不要两份。

15. **[P1] `internal/gaea/agent/canonical_todo.go:59` `AgentRunner.CanonicalTodoState`** — D1 — LOC≈6 — confidence:high
    - 证据：grep 2 命中 = `:58`(注释)、`:59`(定义)。0 调用（含测试 0）。
    - 建议：删除。

16. **[P1] `internal/gaea/memory/citations.go:114` `Set.ResolveCitations`** — D1/D5 — LOC≈14 — confidence:high
    - 证据：定义体是**对 `ResolveCitationsDetailed` 的 3 行转发**（`:115` 一行调用 + nil 判断）；生产 0 调用，测试 7 处（`citations_test.go`）。文档（`:107-113`）若删需把空间语义说明并入 `ResolveCitationsDetailed` 的文档（否则丢失「space 非空 = 限定本空间」的关键契约说明）。
    - 建议：删除包装，测试直接用 `ResolveCitationsDetailed`；**先把文档搬过去**。

17. **[P1] `internal/gaea/tool/tool.go:240` `Registry.Hide`** — D1 — LOC≈7 — confidence:high
    - 证据：grep `.Hide(` 全仓 1 命中（即定义本身）；`tool.go:62/178` 是注释。生产只使用 `HideUnlessOnly`（`boot.go:903-915`）。
    - 建议：删除；`tool.go` 类文档里的「writers (Add/Hide/HideUnlessOnly...)」列表同步改成 `HideUnlessOnly`。

18. **[P1] `internal/gaea/agent/agent.go:664-668` `CacheBreakCount`（含错挂的文档注释）** — D1/D2 — LOC≈5 — confidence:high
    - 证据：grep 全仓 1 命中（定义）。**注释错挂已确认**：`:664-665` 写的是「ContextWindow returns the configured context-window size in tokens. 0 means compaction is disabled」，而真正的 `ContextWindow` 在 `:684` 且无文档。
    - 建议：删除 `CacheBreakCount`；把 `:664-665` 的两行注释移到 `:684` 之前（修 godoc 语义错误，这是检索会命中的坑）。

19. **[P1] `internal/gaea/agent/agent.go:493-500` `AgentRunner.SetActiveSchemas`** — D1 — LOC≈8 — confidence:high
    - 证据：全仓 4 命中 = `:427`(注释提及)、`:493-495`(自身文档)、`:496`(定义)、`task.go:403`（**同名的另一个死方法**，见下表）。注释谎称「Called by the controller after GoalRouter classification」——controller 中无此调用。
    - 建议：删除两个 `SetActiveSchemas`（AgentRunner 与 TaskTool 各一）；若目标路由需要动态工具子集，则应接线（当前 `activeSchemas` 只在 `stream` 内读，且永远为空）。

20. **[P1] `internal/gaea/tool/tool.go:267-269` `MCPNamePrefix`** — D1 — LOC≈3 — confidence:high
    - 证据：grep 2 命中 = `:267`(注释)、`:269`(定义)。生产代码里的 `"mcp__"` 前缀是就地字面量（`control/controller_mcp.go` 等），不经此常量。
    - 建议：删除常量；或反过来把散落的 `"mcp__"` 字面量统一到它——后者更符合原意（改名风险低，建议统一）。

---

## 全量发现表

置信度说明：high = 原文 grep + 剥离计数双重零命中，且已排除接口/Wails/注册表；medium = 命中面干净但涉及持久化格式或公共 API；low = 存在反射/外部消费者/产品语义不确定。

| 文件:行 | 符号 | 类型 | LOC | 置信度 | 证据 | 建议 |
|---|---|---|---|---|---|---|
| `memory/graphstore.go:175` | `GraphNeighbors` | D1 | 83 | high | 5 命中：`:171`注释、`:175`定义、`graph_test.go:238/252/256`；生产 0 调用 | 整函数删；测试改跳过 |
| `agent/compact.go:380` | `SummarizeUpTo` | D1 | 30 | high | 仅定义+注释；Wails 绑定 `OfficeB.d.ts:345` 指向拒绝桩 `gaea_ui.go:726` | 产品确认后删，或接回绑定 |
| `tool/tool.go:354` | `Registry.PersistWriteNames` | D1 | 30 | high | 4 处调用全在测试；生产用 `IsPersistWrite`(`agent/task.go:392`) | 删+改 `tool.go:174-191` 文档 |
| `tool/tool.go:296` | `SuspendPrefix` | D1 | 22 | high | 生产 0 调用（仅 `registry_race_test.go:65/120`） | 删；或接到 `controller_mcp.go` |
| `agent/subagent_store.go:705` | `CleanupStaleRunning` | D1 | 34 | high | 2 命中=注释+定义；0 调用 0 测试 | 删；或 boot 里调用（现为功能缺失） |
| `agent/tool_precheck.go:116` | `precheckDeleteRange` | D1 | 36 | high | 调用方只有死分支 `:23-24` + 测试 | 删函数+分支+测试 |
| `agent/compact.go:353` | `SummarizeFrom` | D1 | 26 | high | 仅定义+注释；绑定 `OfficeB.d.ts:343` 指向拒绝桩 | 同上 |
| `memory/memory.go:441` | `Set.EpisodicMatches` | D1 | 43 | high | 3 命中=2 注释+定义 | 删 |
| `memory/memory.go:409` | `Set.ProceduralBlock` | D1 | 30 | high | 2 命中=注释+定义 | 删 |
| `agent/session/derived.go:17` | `DeriveTitle` | D1 | 23 | high | 生产 0；文件头注释「供绑定层后续接线」，从未接线 | 删（`DeriveStats` 保留） |
| `agent/reasoning_language.go:134` | `WithResponseLanguagePreference` | D1 | 8 | high | 命中=注释+定义 | 删 |
| `agent/reasoning_language.go:142` | `ResponseLanguageFromContext` | D1 | 10 | high | 命中=注释+定义 | 删 |
| `agent/reasoning_language.go:155` | `WithReasoningLanguagePreference` | D1 | 9 | high | 命中=注释+定义 | 删 |
| `agent/reasoning_language.go:163` | `ReasoningLanguageFromContext` | D1 | 10 | high | 命中=注释+定义 | 删 |
| `agent/reasoning_language.go:201` | `SetResponseLanguage` | D1 | 5 | high | 命中=注释+定义；字段无其它写入点 | 删+删字段 |
| `agent/reasoning_language.go:207` | `SetReasoningLanguage` | D1 | 5 | high | 同上 | 删+删字段 |
| `agent/reasoning_language.go:8-9` | `responseLanguageContextKey` / `reasoningLanguageContextKey` | D1 | 2 | high | 仅被上述 4 个死函数使用 | 随族删除 |
| `agent/reasoning_language.go:178` | `AgentRunner.withTurnPreferences` | D2 | 20 | high | 唯一调用 `agent_run.go:53`；读取的两个 atomic 字段恒 nil → 恒 no-op | 随整族删除 |
| `agent/agent.go:484,490` | `responseLanguage` / `reasoningLanguage` 字段 | D1 | 12 | high | 写入点仅两个死 setter | 删（含 10 行文档块） |
| `agent/agent.go:666` | `CacheBreakCount` | D1 | 5 | high | 全仓 1 命中 | 删+把注释移到 `ContextWindow` |
| `agent/agent.go:496` | `AgentRunner.SetActiveSchemas` | D1 | 8 | high | 4 命中=2 注释+2 定义（含同名 TaskTool 方法） | 删两个 |
| `agent/agent.go:538` | `SetMemoryQueue` + 由此产生的 `memQueue` 恒 nil 链 | D1/D2 | 3 | high | setter 生产 0 调用（`agent.go:384` 注释提及）；`memQueue` 字段全仓写入点**仅此 setter**，故生产恒 nil ⇒ `execute_one.go:158` 的 `WithQueue` 注入永不执行、`recall_reminder.go:21` 的 `if a.memQueue == nil { return }` 恒提前返回 | 需产品确认「记忆队列」是否还要：要则接线，不要则删 setter + `memQueue` 字段 + `recall_reminder.go` |
| `agent/agent.go:568` | `SetSink` | D1 | 3 | high | `:563` 注释+定义 | 删 |
| `agent/agent.go:888` | `ProvName` | D1 | 2 | high | 命中=注释+定义 | 删 |
| `agent/agent.go:925` | `SteerConsumed` | D1 | 6 | high | `:924` 注释+定义；仅 `controller_send_test.go:268` | 删或接线（steer 消费状态无人读） |
| `agent/task.go:165,402` | `TaskTool.templatePrefix` 字段 + `SetTemplatePrefix` | D1 | 2 | high | `templatePrefix` 全仓仅 2 处：定义+唯一写入；**无读取点** | 删字段+setter（`AgentConfig.TemplatePrefix` 是另一路径，存活） |
| `agent/task.go:404` | `TaskTool.SubUsage` | D1 | 1 | high | `:916` 是注释「so SubUsage() reflects...」，无调用 | 删（或接成本统计） |
| `agent/task.go:403` | `TaskTool.SetActiveSchemas` | D1 | 1 | high | 同名字段无写入点 | 删 |
| `agent/task.go:126` | `IsSubagentMetaTool` | D1 | 12 | high | 2 命中=注释+定义；`SubagentMetaTools()` 是另一函数 | 删 |
| `agent/task.go:571` | `TaskTool.WithHooks` | D1 | 8 | high | 生产 0（仅 `task_retry_gate_test.go:53`） | 删 |
| `agent/canonical_todo.go:59` | `CanonicalTodoState` | D1 | 6 | high | 2 命中=注释+定义 | 删 |
| `agent/compile.go:59` | `DigestMessages` | D1 | 8 | high | 4 注释+定义；生产走 `DigestCache.Digest` | 删 |
| `agent/stop_gate.go:117` | `verifyGate` | D1/D4 | 11 | high | 生产 0；`taskGate:34-41` 内联同逻辑 | 删，统一到一处 |
| `agent/render/sink.go:286` | `DimText` | D1 | 2 | high | 2 命中=注释+定义 | 删（`writeDim` 是同用途活代码） |
| `agent/render/sink.go:54` | `Sink.SetShowReasoning` | D1 | 2 | high | 生产 0（仅 `textsink_test.go:52`） | 删 |
| `agent/session/session.go:137` | `ReplayFromLog` | D1 | 9 | high | 生产 0（仅 `session_log_test.go`） | 删 |
| `agent/textutils_alias.go:12` | `streamedRows` | D1 | 1 | high | 生产仅定义；测试 `width_test.go:28/29/39/40/47` | 删（测试直接用 `textutils.StreamedRows`） |
| `agent/session_alias.go:11` | `SessionInfo` 别名 | D1 | 2 | high | 2 命中=注释+定义 | 删 |
| `agent/budget_alias.go:7` | `BudgetStatus` 别名 | D1 | 1 | high | 全仓 1 命中 | 删 |
| `memory/store.go:287` | `ArchivedRetention` | D1/D4 | 3 | medium | 生产 0（测试 6 处）；与 `config.ArchivedRetentionDays=90`(`config.go:750`) 语义重复 | 删，测试改用 config 值 |
| `memory/store.go:401` | `NewDeliverable` | D1 | 15 | high | 2 命中=注释+定义 | 删 |
| `memory/citations.go:114` | `Set.ResolveCitations` | D1/D5 | 14 | high | 转发 `ResolveCitationsDetailed`；生产 0 | 先搬文档再删 |
| `tool/builtin/websearch_engine.go:223,225` | `tavilyRequest.SearchDepth` / `IncludeAnswer` | D1 | 2 | high | 结构体定义 2 行；字面量 `:239-242` 只设 Query/MaxResults，**从不赋值** | 删两字段（有 `omitempty`，报文不变） |
| `agent/session/branch.go:23` | `BranchMeta.ForkMessageIndex` | D1 | 1 | medium | 0 读 0 写；带 `json:"fork_message_index,omitempty"` 属持久化 sidecar 格式 | 删前确认无旧版 meta 依赖（现无读取者） |
| `agent/session/checkpoint.go:27` | `Checkpoint.Rewrite` | D1 | 1 | medium | 0 读 0 写；持久化结构 | 同上 |
| `tool/tool.go:40` | `ToolContext.MessageID` | D1 | 1 | medium | 0 读 0 写；公共 API 字段（供 embedder） | 若确认无外部使用则删，否则保留并注明 |
| `agent/testutil/mock_provider.go:130` | `MockProvider.LastRequest` | D1 | 4 | medium | 生产 0 测试 0；test-helper 公共面 | 删 |
| `agent/testutil/mock_provider.go:148` | `MockProvider.SetScript` | D1 | 8 | medium | 生产 0 测试 0 | 删 |
| `agent/testutil/mock_provider.go:180` | `Turn.UsageTurn` | D1 | 4 | medium | 生产 0 测试 0 | 删 |
| `agent/testutil/mock_provider.go:193` | `ErrorTurn` | D1 | 4 | medium | 生产 0 测试 0 | 删 |
| `agent/subagent_store.go:140` | `EphemeralSubagentRun` | D1 | 8 | high | 2 命中=注释+定义 | 删 |
| `agent/tool/envelope.go:79` | `ParseEnvelope` | D1 | 9 | medium | 生产 0（2 测试命中）；公共解析 API | 删或明确为外部 API 并加注释 |
| `tool/tool.go:226-230` | `Add` 内 `suspended` 守卫循环 | D2 | 5 | high | 唯一写入者 `SuspendPrefix` 生产 0 调用 ⇒ 生产中永不迭代 | 随 SuspendPrefix 一起删（或接上 MCP 会话禁用） |
| `tool/tool.go:316` | `ResumePrefix` | D1 | 6 | high | 生产 0（仅 `registry_race_test.go:70/158`） | 删 |
| `agent/agent.go:664-665` | 错挂文档（`ContextWindow` 的注释挂在 `CacheBreakCount` 上） | D2/D7 | 2 | high | `:684` 的 `ContextWindow` 无文档；godoc 检索会命中错误语义 | 把注释移回 `:684` |
| `agent/compress.go:384,385,387,388` | `toolCompressLimits` 中 `run_command`/`run_background`/`glob`/`search_files` 4 行 | D2 | 4 | high | 4 个名字都不是任何工具名（46 内建 + ExtraTools 全无）；MCP 名带 `mcp__` 前缀 | 删 4 行 |
| `agent/compress.go:407` | `case "directory_tree", "list_directory"` | D2 | 2 | high | 两者同样不是工具名（`:390`/`:395` 的 map 行同步删） | 删 case + 2 map 行 |
| `agent/tool_coherence.go:84` | `isReadTool` 的 `"glob"` 分支 | D2 | 1 | high | `glob` 无实现 | 删 1 个 case 值 |
| `agent/tool_coherence.go:93` / `agent/execute_one.go:559` / `agent/agent_run.go:363` / `agent/batch_executor.go:237` / `agent/compact_summary.go:46` | 同一「写工具名单」内联重复 5-6 次，且含不存在的 `delete_range`/`delete_symbol` | D4/D2 | 6 | high | 5 处 switch 名单逐字重复；两个名字从未注册（`RegisterBuiltin` 全仓无此项） | 收敛为 `writerToolNames` 一个集合 + `isWriteTool` 一处判定，删 2 个幽灵名 |
| `agent/execute_one.go:557` | `isFileWriter` ≡ `tool_coherence.go:91 isWriteTool` | D4 | 8 | high | 归一化函数体哈希完全相同（7 行 switch 逐字节同）；两者**都有**生产调用（`execute_one.go:106/236/328` 与 `tool_coherence.go:35/56/68`） | 合并为一个函数 |
| `agent/reasoning_language.go:14` | `NormalizeReasoningLanguage` ≡ `:27 NormalizeResponseLanguage` | D4 | 10 | high | 函数体哈希相同（行为逐字等价） | 合并为一个 `normalizeLanguage`，两处调用点改名 |
| `tool/builtin/cost_tools.go:225,298` | `costEmbedder` ≈ `costReranker` | D4 | 70 | high | 归一化后函数体完全相同（36/34 行） | 抽 `lazyRetrievalComponent` 通用实现 |
| `agent/session/log.go:181,194` | `LogPathFor` ≈ `CheckpointPathFor` | D4 | 22 | high | 归一化后函数体完全相同（11 行） | 抽 `pathFor(suffix)` |
| `tool/builtin/readfile.go:281` / `websearch_engine.go:89` | `NewMarkdownConverter` ≈ `NewSearchEngine` | D4 | 28 | high | 归一化后函数体完全相同（14 行，同一 lazy-singleton 模式） | 抽通用 once+override 包装 |
| `agent/agent_helpers.go:19` | `extractFilePath(name string, ...)` 的 `name` 参数 | D7 | 1 | high | 函数体从不引用 `name`（调用点 `compact_summary.go:47` 传入 `tc.Name`） | 删参数（1 个调用点） |
| `agent/detector.go:83` | `isToolMisuse(sig string, ...)` 的 `sig` | D7 | 1 | high | 函数体只用 `errors` | 删参数 |
| `agent/batch_executor.go:182` | `partitionToolCalls(r *tool.Registry, ...)` 的 `r` | D7 | 1 | high | 函数体只用 `calls`；调用点 `:98/171` | 删参数 |
| `agent/fold_ladder.go:52` | `slimFoldForSummary(a *AgentRunner, ...)` 的 `a` | D7 | 1 | high | 自由函数，函数体不用接收者 | 删参数（调用点 `compact.go:702/707`） |
| `memory/morning_preload.go:37` | `BuildMorningPreloadBlock(..., now time.Time, ...)` 的 `now` | D7 | 1 | low | 文档自证「now 用于锚定确定性输出（**预留**；当前渲染不依赖时间）」 | 需人工确认：保留则补 `_ = now` 或注释，删则改 8 处测试签名 |
| `agent/session/sink.go:38` | `NewEventLogSink(dir string, ...)` 的 `dir` | D7 | 2 | high | 构造体只存 `inner`；路径由 `SetPathSource` 注入；调用点 `boot.go:169` | 删参数（1 调用点） |
| `agent/recall_reminder.go:20` | `maybeRecallReminder` + `recallReminderNudge`(:11) + `recallReminderFired`(:312) | D2 | 52 | high | 调用点存在（`agent_run.go:100`）但首行 `if a.memQueue == nil { return }` **恒真**（见上行）；48 行函数 + 11 行常量 + 1 字段在生产不可达，仅 `recall_reminder_test.go` 靠手写结构体字面量 `a := &AgentRunner{memQueue: stubMemQueue{}}` 覆盖 | 与 `SetMemoryQueue` 一起决策：接线则保留，否则整块删 |
| `memory/store.go:135-330` | `Store` 门面：约 30 个方法全部 `return s.engine().X(...)` | D5 | ~60 | medium | 逐方法单行转发；`engine()` 返回 `Backend` 接口 | 不建议直接删（是空间收窄与双后端的接缝）；可考虑内嵌 `Backend` 减少样板 |
| `memory/space_view.go:14-77` | `spaceView`：20 个纯转发方法 | D5 | ~45 | medium | 全部 `return v.inner.X(...)`；仅 `Index/List/ListInSpace/Get/GetInSpace/Touch/TouchInSpace` 7 个有空间逻辑 | 同上（嵌入 `Backend` 可省 ~20 行样板，但读端隔离语义依赖覆盖，风险中） |
| `agent/budget_alias.go:6-20` | budget 子包再导出 3 type/3 const/3 var | D5 | 9 | medium | `BudgetStatus` 死（见上）；`BudgetOK/Warn/Block`、`ModelProfile`、`NewBudgetGate`、`DefaultModelProfiles`、`LookupModelProfile`、`BudgetGate` 有引用 | 仅删死的 `BudgetStatus`；其余保留 |
| `agent/session_alias.go:14-28` | session 子包 14 个再导出 var | D5 | 15 | low | 逐个计数未做（属「保留兼容层」既定设计） | 建议后续单独评估别名层存废，本次不动 |
| `agent/render_alias.go:11-28` | render 别名（`Renderer`/`TextSink`/`newStreamBatcher`/…） | D5 | 18 | medium | `newStreamBatcher` 有 1 生产调用（`agent_stream.go:93`）；注释称「kept for external consumers (cli)」但仓内无 cli 引用 | 低优先；先确认仓外消费者 |
| `agent/render/sink.go:225,250,286` 等 | `closeTextStream`/`usageLine`/`DimText` | D5/D1 | 2 | high | 仅 `DimText` 零引用；其余各 1 处生产调用 | 仅删 `DimText` |
| `agent/agent.go:152` vs `agent/budget/gate.go:35` | 同名类型 `Gate`（interface vs struct） | D8 | 0 | low | 跨包同名，非重复定义 | 不改（仅命名干扰） |
| `agent/agent_config.go:10` vs `memory/memory.go:31` | 同名 `Options` struct | D8 | 0 | low | 跨包正常命名 | 不改 |
| `tool/registry_test.go:10` vs `tool/builtin/spacetags_test.go:12` | 同名 `stubTool` | D8 | 0 | low | 两个不同包的测试桩 | 不改 |

补充阴性结论（避免误报，勿列为发现）：

- `tool.ContextualTool`、`PersistWriteTool`、`ModelBackedTool`、`SpaceTaggedTool`、`SessionStateResetter` **都有真实的类型断言/实现**（如 `execute_one.go:176`、`tool.go:98/116/132`、`tool/builtin/spacetags.go:111`、`boot.go:562`），不是「只有一个实现的空接口」。
- `ToolDispatcher.SetObserver`（`tool_dispatch.go:34`）**是活代码**：生产调用点 `agent.go:894`（`SetCtxMgr` 内 `a.dispatcher.SetObserver(m)`）。初判为死，原文 grep 后撤回——这是本次最容易误报的一类（同名短方法）。
- `cache/`、`budget/`、`textutils/`、`toolguard/`、`render/` 子包**都在生产路径上被使用**（`agent.go:11/744` 用 `cache.New`、`param_storm.go:148` 用 `cache.CanonicalizeValue`、`execute_one.go:133/199/207` 用 `toolCache.Get/Set/InvalidatePath`），无死子包。
- `Cache.Clear`（`cache/toolcache.go:122`）文档写「Called at the start of each turn」但**生产无调用**（`.Clear()` 在 agent 包内仅 `cache_test.go:89`）。此处保守起见**未列入发现表**：`Cache.Clear` 是 `Cache` 类型的自然公共方法，且「每回合清理」若被删还需确认 `agent_stream` 是否依赖 TTL 兜底——建议人工确认后再决定（属 D1，LOC≈7，confidence:medium）。
- 46 个内建工具的 `Execute` 方法**全部可达**（每文件 `init()` → `RegisterBuiltin` → `boot.addBuiltins`），无「注册了却永不执行」者。
- `agent/testutil/` 包被 `boot`、`app` 的测试导入（`boot_test.go` 等 6 个文件），包本身存活；死的是包内 4 个 helper（见上表）。

---

## 屎山清单（结构性重构建议，含风险）

量化基线（非测试文件；括号深度由大括号配对精确计算；`if` 为语句计数）：

| 函数 | 行数 | if | case | switch | for | 最大嵌套 |
|---|---|---|---|---|---|---|
| `agent/agent_run.go:23 runDirect` | **445** | **52** | 2 | 1 | 3 | 7 |
| `agent/execute_one.go:21 executeOne` | **338** | **55** | 8 | 3 | 0 | 5 |
| `tool/builtin/bash.go:86 bash.Execute` | **289** | 32 | 7 | 0 | 0 | 6 |
| `agent/agent_stream.go:29 stream` | 181 | 17 | 6 | 1 | 1 | **7** |
| `agent/compact_summary.go:14 BuildCompactSummary` | 176 | 23 | 3 | 2 | 10 | 6 |
| `tool/builtin/readfile.go:62 readFile.Execute` | 173 | 23 | 3 | 1 | 3 | 4 |
| `tool/builtin/webfetch.go:112 ssrfGuardedTransport` | 108 | 19 | 2 | 1 | 2 | **8** |
| `agent/ask.go:30 Schema` | — | — | — | — | — | **9** |

全范围 **≥60 行的函数共 45 个**。

按收益/风险排序的重构建议：

1. **`runDirect`（445 行 / 52 个 if）拆 5-6 段** — 风险中。它把「输入预处理 → 首轮消息装配 → 循环调度 → 停止闸门 → 结果落盘」压在一个函数里，最大嵌套 7 层。建议按阶段抽成 `prepareTurn/runLoop/finalize`，**先加表驱动测试锁行为**（仓内已有 `agent_run_test.go` 可扩）。这是本范围**最大**的可读性债务，但也是最容易改出回归的地方——建议单独 PR、不改逻辑。
2. **`executeOne`（338 行 / 55 个 if / 8 case）** — 风险中高。工具分派与「写工具冲突键」判定、缓存失效、precheck 交织在一起，且与 `batch_executor` / `tool_coherence` 共享同一份工具名单（见 D4 行）。建议先把**工具分类**收敛为一个表（`writer/reader/parallel-safe/global-conflict`），再从 `executeOne` 里把 precheck 与缓存维护抽成独立步骤。先做分类表这一步是**低风险高收益**的（同时消掉 5-6 处重复名单）。
3. **`bash.Execute`（289 行 / 32 if / 7 case）** — 风险中。参数解析、`output_format` 分支、超时、后台作业、shell 状态采纳混在一起（`shellstate.go` 已单独抽出状态，可继续沿这条缝切：`parseArgs` / `runForeground` / `runBackground`）。
4. **深嵌套**：`webfetch.ssrfGuardedTransport`（8 层）与 `ask.Schema`（9 层）——都是**纯数据/守卫**逻辑，用「提前 return + 小函数」可轻松降到 3-4 层，风险低，适合作为样板 PR。
5. **`tool/builtin/compact.go`（143 行）是全仓最大的重复体**：`compactDesc` + `compactSchema` 是 46 个工具描述/schema 的**第二份手工副本**。`tool/tool.go:56-72` 明确写了「compact 变体是模型唯一所见，完整 Description/Schema 作为派生子成为死输出」，并由 `compact_drift_test.go` 用「字段集子集」测试钉住。**风险高，不要直接删/自动生成**：建议把 `compactSchema` 改为从完整 Schema 做**字段裁剪**（唯一数据源），`compactDesc` 保持手写（它是给人看的短文案）。这能一次性消掉 ~46 份手工维护的 schema 副本。
6. **`memory/Store` + `spaceView` 双层门面**：约 50 个单行转发方法。风险中：`spaceView` 的读端空间隔离语义依赖「覆盖 7 个方法、其余透传」的精确性，改成嵌入 `Backend` 需逐个核对覆盖面，建议**低优先、单独 PR**。

---

## 不建议动的地方（说明为什么）

| 位置 | 为什么不动 |
|---|---|
| `tool/builtin/hide_window_windows.go:33-53` 中 15 个「零引用」字段（`PerProcessUserTimeLimit`、`IoInfo`、`Affinity`…） | Win32 `JOBOBJECT_*` 结构体**内存布局定义**，字段必须按 ABI 顺序存在以对齐 `syscall` 调用；「零引用」是正常的（只用于 `unsafe.Sizeof`/内存覆盖）。删了会静默破坏 Job Object 限制。 |
| `tool/tool.go:51 ContextualTool`、`:82 PersistWriteTool`、`:109 ModelBackedTool`、`:125 SpaceTaggedTool`、`:92 SessionStateResetter` | 都有真实断言/实现点（`execute_one.go:176`、`agent/task.go:392`、`boot.go:562`、`spacetags.go:111`、`boot` 会话切换路径）。是**刻意的能力标记（capability marker）**模式，删接口会连带删掉注册表驱动逻辑。 |
| `tool/builtin/compact.go` 的 `compactDesc`/`compactSchema` | 见上（D4 最大项）。它是模型实际看到的唯一 schema，且被 `compact_drift_test.go` 钉住；**只能改数据源，不能删**。 |
| `agent/reasoning_language.go:14/27 Normalize*` 的**两个名字本身** | 函数体可合并，但两个名字分别被 `Options` 初始化/`config` 侧语义引用；合并时**保留两个导出名做转发**，否则影响 `internal/gaea/config` 与 `boot`（跨包，不在本次审计范围）。 |
| `memory/store.go` 的 `Store` 门面全部方法 | `Store` 是「零值可用 + 空间收窄 + 双后端」的公共接缝（`boot`/`control`/`app` 大量使用），单行转发是刻意设计。 |
| `agent/testutil/mock_provider.go:34-43` 的 `Usage:` 注释块 | **godoc 使用示例**，不是注释掉的死代码。删了会让这个测试桩无从使用。 |
| `agent/session/log_bench_test.go:46` 的注释块 | 基准测试的说明性注释。 |
| `tool/builtin/*.go` 里 `Execute(ctx context.Context, ...)` 未使用 `ctx`（`ls.go:34`、`schedule_tools.go:82/408`、`screenshot.go:46`、`sidebar_open.go:62`） | `ctx` 是 `tool.Tool` 接口签名的一部分（`tool.go:25`），**不可删**。 |
| `memory/file_backend.go` 的 `Touch/Pin/Unpin/AppendFeedbackEvent` 返回「不支持」错误 | 实现 `Backend` 接口的必要桩，删了编译不过。 |
| `agent/compact.go:353/380 Summarize*`、`agent/agent.go:496 SetActiveSchemas`、`agent/compact.go:348 CompactNow`、`agent/task.go:404 SubUsage` | 虽然生产零调用，但**前端/Wails 或产品语义可能已承诺**（`Summarize*` 有 wailsjs 绑定；`CompactNow`/`SubUsage` 对应「手动压缩」「子代理用量」这类用户可见概念）。建议按「删除 or 接线」二选一**由主控/产品确认**，不要静默删除。 |
| `docs/code-audit-2026-10-02/**` 中的历史审计数据 | 不是本次范围；且其中「已发现」条目与本次结论高度重合（如 `SetTemplatePrefix`、`CacheBreakCount`、testutil 4 helper），说明这些问题**至少被报告过一次而未被处理**——建议在修复时同步更新该目录，避免第三轮重复发现。 |

---

## 附：本次审计使用的可复现命令（只读，均为临时脚本，未落仓库）

1. 符号清单 + 剥离注释/字符串后的全仓引用计数（1825 个 .go）→ `symbols2.json`
2. 原文 grep 复核每条零引用结论（区分生产/测试/wailsjs/注册表）
3. 函数边界括号配对 → 精确 LOC / 行数 / if-case-switch-for 计数 / 最大嵌套深度
4. 归一化函数体 MD5 → D4 重复实现（标识符与字符串归一）
5. 结构体字段零引用扫描（758 个字段 → 18 个候选，其中 15 个为 Win32 布局，已排除）
6. `spaceTags` 64 键 × 全仓 `Name()` 字面量交叉核对 → 0 死键
7. `RegisterBuiltin`（47 处）+ `frontend/wailsjs` 全量 grep → Wails/注册表风险面收口

**统计口径声明**：所有 LOC 为「函数体 + 紧邻 `//` 文档注释」行数，含空行；实际删除时按 gofmt 后重排可再省若干行。所有「生产 0 调用」均指「排除 `_test.go` 后全仓 0 命中」。
