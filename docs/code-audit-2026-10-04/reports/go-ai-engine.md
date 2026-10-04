# Go 审计 · ai/modelengine/channels/provider

审计对象：Gaea（`github.com/gaea/gaea`，Go 1.26 + Wails v2 桌面应用）
审计日期：2026-10-03 | 只读审计（未修改、未新建任何仓库文件，除本报告）
审计方法：AST 解析（`go/parser`，临时工具建在 `%TEMP%`，不落仓库）+ **逐符号 ripgrep 引用计数**（589 个顶层符号，每个符号单独一次精确 `\b符号\b` 检索）

> **方法论警告（重要）**：审计中途发现，用「589 个符号拼成一条 `\b(a|b|c|…)\b` 大正则」做批量检索会**漏匹配**（实测 `setGLMCatalogPath` 在 `glm_catalog.go:413` 的调用点被漏掉，因为 Rust regex 的 leftmost-first 交替在长名字上前缀抢先）。
> 因此本报告**全部结论以「每符号单独 grep」的复核数据为准**，批量结果只用于筛候选。凡是"0 命中"的结论，都已用单符号 grep 复验过。

---

## 范围与文件统计

| 目录 | 文件数 | 字节 | 行数 | 非测试文件 | 非测试行数 |
|---|---:|---:|---:|---:|---:|
| `internal/ai/` | 35 | 348,330 | 7,937 | 18 | 3,105 |
| `internal/modelengine/` | 26 | 302,446 | 6,852 | 16 | 3,856 |
| `internal/channels/` | 15 | 185,714 | 4,139 | 8 | 2,328 |
| `internal/gaea/provider/` | 15 | 93,913 | 2,484 | 8 | 1,105 |
| `internal/netclient/` | 8 | 38,908 | 1,233 | 5 | 605 |
| `internal/analysis/` | 13 | 106,388 | 2,420 | 7 | 1,314 |
| `internal/auth/` | 7 | 48,955 | 1,257 | 4 | 707 |
| **合计** | **119** | **1,124,654** | **26,322** | **66** | **13,020** |

- 顶层声明：**452 个函数/方法**、**120 个类型**、589 个可检索符号（含 const/var）。
- 排除目录（未纳入任何检索）：`clones/`、`node_modules/`、`frontend/`、`.tmp*/`、`backups/`、`releases/`、`dist/`。
  - 例外：为判定"字符串驱动/前端调用"风险，对 `frontend/src` **只做过字符串字面量反向检索**（不视为代码引用）。
- `reasonix.toml`（77 行）已审，**结论：它不是 Gaea 的配置**，见 F-50。

### 全局性结论（先说结论，避免误伤）

1. **D3（注释掉的代码）在本域基本不存在** —— 这是本次审计最重要的"否定结论"。对 7 个目录做 4 组注释代码启发式（`// if|for|return|case|switch|func|…`、`// x := y`、`// )`、`// foo(...)`）+ `/* */` 块注释扫描，命中 **5 条全部是文档注释/中文词语误报**，无一条是注释掉的代码块。`TODO/FIXME/XXX/HACK` 在非测试代码中 **0 命中**。→ **本域没有"注释掉旧模型名/旧端点"的问题**，不必安排清理批次。
2. **没有"实现了接口但从未注册"的废弃适配器**。`ImageBackend` 接口的 3 个实现全部在 `init()` 自注册：`image_openai.go:45`（openai）、`image_comfyui.go:57`（comfyui）、`image_glm.go:50`（glm）。`provider.Provider` 注册表在生产侧只有 `bridge/bridge.go:176` 一个 kind。
3. **ComfyUI 的模型分派用的是 map 而不是巨型 if/else 链**（`image_comfyui.go` 的 `txt2imgWorkflows`/`img2imgWorkflows` + `lookupTxt2imgBuilder`）。D6 的"模型名分支链"在 `internal/ai` **不成立**，不要照单去改。
4. 真正的 D6 屎山集中在 3 个文件：`client.go`(1160 行)、`image_comfyui.go`(914 行)、`weixin/clawbot.go`(893 行)；真正的 D4 重灾区是 **HTTP 鉴权头 / 重试退避 / SSRF 判据 / GLM 错误体解析** 四处。

---

## Top 20 发现（按 可删行数 × 安全度 排序）

1. **[P0] `internal/gaea/provider/retry.go`:15-146 `BackoffStrategy`/`DefaultBackoff`/`RateLimitBackoff`/`Duration`/`Sleep`/`RetryPolicy`/`ParseRetryAfter`/`IsRetryableStatus`/`IsTransientNetErr`** — **D1** — LOC≈136（整文件） — confidence:**high**
   - 证据：`rg -n '\b(DefaultBackoff|RateLimitBackoff|ParseRetryAfter|IsRetryableStatus|IsTransientNetErr|RetryPolicy)\b' --glob '*.go'` → 30 命中，**全部**落在 `internal/gaea/provider/retry.go`（定义处 8 行）与 `internal/gaea/provider/retry_test.go`（自测 22 行）；`BackoffStrategy` 另用 `rg '\bBackoffStrategy\b'` → 仅 `retry.go`+`retry_test.go`。生产代码（agent/bridge/app/modelengine）**零调用**。`rg '\bRetryPolicy\b'` → 仅 `retry.go:84` 与 2 处 docs。
   - 建议：**这是"要么接线、要么删除"的典型**，不要直接删——见"不建议动的地方"第 3 条。两条路线：(a) 让 `ai/client.go` 的退避、`modelengine/health_probe.go` 的重试、`channels/weixin` 的退避统一改用本文件（它的 API 是最完整的：`BackoffStrategy.Duration/Sleep` + `ParseRetryAfter` + `IsRetryableStatus/IsTransientNetErr`），删掉各处的私有实现；(b) 确认不接线则整文件（含 `retry_test.go`）删除，回收 136+196 行。**单删实现保留测试会直接编译失败，必须成对处理。**

2. **[P0] `internal/gaea/provider/provider.go`:66-211 `SanitizeToolPairing`+`isToolResultOrBridge`+`extractToolResults`+`pairToolResults`+`idDistinct`+`interruptedToolResult`** — **D1** — LOC≈120 — confidence:**high**（"无生产调用方"），**medium**（"可安全删除"）
   - 证据：`rg -n 'SanitizeToolPairing|interruptedToolResult|pairToolResults|extractToolResults|isToolResultOrBridge|idDistinct' --glob '*.go'` → 35 命中，全部在 `provider.go`（定义 6 处 + 内部互调）与三个测试文件（`pairing_probe_test.go:18,45`、`provider_test.go:36,52,71,90`、`sanitize_upgrade_test.go:21,60,99,147,172`）。**没有任何生产调用方**（`bridge/bridge.go`、`internal/gaea/agent/*` 均无）。
   - 建议：这套"修复 tool_call/tool_result 配对"的护栏按设计应在把历史发给模型前调用（`sanitize_upgrade_test.go` 自述对照 `Kun model-history-repair.ts`）。**"没被接线"很可能是一个真 bug 而不是垃圾**：请先判定是否应该在 `bridge.Chat` 入口调用；若确认不需要，则 6 个符号 + 3 个测试文件一次性删除。

3. **[P1] `internal/modelengine/engine_models.go`:67-68、73-74、95-96、101-102 四处不可达分支** — **D2** — LOC≈4（+ 退化为可简化的链） — confidence:**high**
   - 证据：同文件 `:40-43` `if engine.Type == EngineGLM { return m.glmCatalogModels(), nil }`、`:44-49` `if engine.Type == EngineModelHub { return m.fetchModelHubModels(...) }` **函数头就提前返回**；因此 `:67 else if engine.Type == EngineGLM && m.glmKey != ""`、`:73 else if engine.Type == EngineModelHub && …`、`:95 else if engine.Type == EngineGLM`、`:101 else if engine.Type == EngineModelHub` **四个分支恒为假**。
   - 建议：删除这 4 个分支；顺手把 `fetchModels` 的鉴权链/401 文案链收敛（见下一条）。

4. **[P0] `internal/modelengine/engine_models.go`:63-81 + 89-106 同一个 `engine.Type` 七分支链写了两遍** — **D4/D6** — LOC≈40（可缩到 ≈12） — confidence:**high**
   - 证据：`:63-81` 是 6 个 `else if engine.Type == X && m.xKey != ""` 全部执行**同一句** `req.Header.Set("Authorization", "Bearer "+<key>)`；`:89-106` 是同一个 `engine.Type` 顺序再走一遍，每个分支产出一句 401 文案。
   - 建议：抽 `func (m *Manager) engineKey(t EngineType) string`（其实 `CustomEngineKey` 已是先例）+ `var engine401Hint = map[EngineType]string{…}`，两条链各压成 3-4 行。纯等价重构，风险低。

5. **[P0] SSRF 判据被复制三份，且副本漏了已修的 bug** — **D4** — LOC≈45 — confidence:**high**
   - 证据：
     - 正本 `internal/netclient/ssrf.go:14-25`（`cgnatRange`+`mustCIDR`）+ `:33-46`（`BlockedSensitiveIP`/`BlockedInternalIP`）+ `:53-89`（`GuardedClient`）。
     - 副本 A `internal/channels/weixin/media_download.go:36-46`（`cgnatRange`+`mustCIDR` **逐字节相同**）+ `:51-60`（`blockedMediaIP` ≡ `BlockedInternalIP` + 一个 `mediaAllowLoopback` 测试开关）+ `:65-86`（`mediaTransport` ≡ `GuardedClient`）。
     - 副本 B `internal/gaea/plugin/ssrf.go:41-60`（`cgnatRangePlugin`+`mustCIDR`）。
     - `internal/gaea/tool/builtin/webfetch.go:74` 注释自己写明"判据已提升为 `netclient.BlockedSensitiveIP`"，即 netclient 是**公认的收敛点**，但另两处没跟。
   - **副本 A 已经落后于正本**：`ssrf.go:73-87` 有"按已校验 IP 集合逐个回退拨号"的修复（注释标 `审计 P1 IN2-04`），而 `media_download.go:82` 仍写死 `dialer.DialContext(..., net.JoinHostPort(ips[0].IP.String(), port))` —— 单个目标连接失败即整体失败。**这是"复制实现导致修复不齐"的实证**。
   - 建议：`media_download.go` 删 `cgnatRange`/`mustCIDR`/`blockedMediaIP`/`mediaTransport`（≈45 行），改调 `netclient.BlockedInternalIP` + `netclient.GuardedClient`，保留 `mediaAllowLoopback` 作为回环豁免参数；`plugin/ssrf.go` 同理（注意它有独立的 plugin 侧开关，需一次等价性比对）。**收敛后副本 A 自动获得 IN2-04 修复。**

6. **[P1] `internal/ai/client.go`:66-211 `client.go` 全文件 1160 行 / `parseStreamEvents` 159 行 / `chatOnce` 126 行 / `ChatStream` 109 行 / `doStreamRequest` 82 行** — **D6** — LOC≈0（重构） — confidence:**high**
   - 证据：AST `go/parser` 统计，见"屎山清单"表。`rg -c '^func ' internal/ai/client.go` → 60 个函数塞在 1 个文件里。
   - 建议：按职责拆成 `client.go`（路由/引擎选择）+ `client_stream.go`（`doStreamRequest`/`parseStreamEvents`/`idleTimeoutBody`）+ `client_retry.go`（退避 + `isTransferableChatError`）+ `client_usage.go`（`recordUsage`/`cacheSplitForUsage`）。纯文件拆分，无行为变更，是最安全的第一刀。

7. **[P1] `internal/channels/weixin/clawbot.go`:455/893 行、`handle` 126 行、`pollLoop` 97 行** — **D6** — LOC≈0（重构） — confidence:**high**
   - 证据：AST：`Server.handle` L531-656（126 行，控制嵌套 5 层）、`pollLoop` L433-529（97 行）、`processInboundFile` L779、`invokeFileHandler` L799、`invokeInboundImageHook` L760 … `clawbot.go` 单文件 18 个方法。
   - 建议：把 `handle` 的消息类型分派抽成 `dispatch(msg)` + 每种 item 一个 `handleXxx`（`itemElem`/`imageItem`/`fileItem` 已是有类型），`pollLoop` 的退避与停止判断抽 `nextPollDelay`。**这是纯结构重构，风险中；建议单独批次 + 现有 `clawbot_test.go`(544 行) 兜底。**

8. **[P0] HTTP `Bearer` 头 + `Content-Type: application/json` 在 6 个包手写 14+ 次** — **D4** — LOC≈0（收敛） — confidence:**high**
   - 证据（生产代码，测试不计）：
     `internal/ai/client.go:729-733`、`internal/ai/image_openai.go:85-87,152-154,281`、`internal/ai/image_glm.go:93-94`、
     `internal/modelengine/engine_models.go:64,66,68,70,72,74,79,240-241`、`engine_modelhub.go:62,192-193,247`、`health_probe.go:202`、
     `internal/channels/weixin/clawbot.go:1062,1086-1087`、`weixin/qrlogin.go:48`、`weixin/media_upload.go:209`、
     （域外同形：`internal/tts/xai.go:103-104`、`internal/gaea/vision/vision.go:191`、`internal/gaea/tool/builtin/websearch_engine.go:251-252`）
   - 建议：在 `netclient` 加 `func SetJSONBearer(req *http.Request, key string)`（key 为空时**不设** Authorization —— 这正是 `engine_models.go:77` 与 `engine_custom_test.go:320` 依赖的语义），全域替换。`image_comfyui.go:578` 只设 Content-Type（本地无鉴权）→ 用 `SetJSON(req)` 变体。**注意 weixin 需要额外的 `AuthorizationType`/`X-WECHAT-UIN`/`iLink-*` 四个私有头，不要在收敛时把它们弄丢。**

9. **[P1] `internal/ai/client.go`:208 + `:759/:778/:780` 状态码先格式化成字符串、再用正则解析回来** — **D2/D6** — LOC≈20 — confidence:**high**
   - 证据：`var transferableStatusRe = regexp.MustCompile(`API 错误 \(HTTP (\d{3})\)`)`（`:208`）用于 `isTransferableChatError`（`:215-227`）反推状态码；而该字符串正是 `:759`、`:778`、`:780` 三处 `fmt.Errorf("API 错误 (HTTP %d): …")` 拼出来的。
   - 建议：定义 `type HTTPStatusError struct{ Code int; Body string }`，三处改为返回它，`isTransferableChatError` 改 `errors.As` 取 `Code`；正则与字符串契约一起删掉（`transferableStatusRe` 回收 1 行 + 消除隐式契约）。

10. **[P1] GLM/智谱错误体解析实现了两遍** — **D4** — LOC≈31 — confidence:**high**
    - 证据：`internal/modelengine/engine_models.go:262-275 zhipuErrorMessage`（12 行）与 `internal/ai/image_glm.go:139-159 zhipuErrorBodyMessage`（19 行）；两者注释均自称"解析智谱错误体 `{"error":{"code","message"}}`（官方形态）"，调用点分别在 `engine_models.go:251`（glmPing）与 `image_glm.go:107`。
    - 建议：留一份（建议放 `modelengine` 或新的 `internal/ai/zhipuerror`），另一处调用；`modelengine` 不依赖 `ai`（`ai` 依赖 `modelengine`），所以正确方向是 **`ai` 复用 `modelengine` 的导出版**，或把两者都提升到一个零依赖小包。**先确认依赖方向再动手**（`manage`→ 若反向会引入 import cycle）。

11. **[P1] `internal/ai/copilot.go`:20 `GhostComplete`(77 行) / `:245 GenerateBeats`(43 行) / `:292 GenerateProseFromBeat`(86 行) 生产零调用** — **D1** — LOC≈206 — confidence:**medium**
    - 证据：`rg -n 'GhostComplete|GenerateBeats|GenerateProseFromBeat' --glob '*.go'` → 仅 `internal/ai/copilot.go` 的定义与注释；`internal/app/` 0 命中（对照：`CmdKEdit`/`OfficeEditText`/`XlsxEditOps` 都有 `internal/app/copilot_handler.go:24`、`gaea_docx_edit.go:35`、`gaea_xlsx_edit.go:79` 的真实调用方）。`rg` 前端 → **只有** `frontend/src/types/wails.d.ts:221,224,225` 三行类型声明，**没有任何 .ts/.tsx 调用点**。
    - 风险与建议：`wails.d.ts` 是**生成的绑定声明**，且这三行的签名（`GhostComplete(currentText, styleProfile)`）与 `ai.Client` 的真实签名（`GhostComplete(ctx, model, currentText, styleProfile)`）**根本对不上**，说明它是历史遗留声明，不能作为"存活"证据。但因为它含这三个字符串，**判死前请人工确认 wails.d.ts 的重生成时机**（若某条构建路径仍按它生成，删 Go 侧会破坏前端类型检查）。建议：连同 `wails.d.ts` 三行一起删，并跑一次绑定一致性测试 `internal/app/bindings_completeness_test.go`。

12. **[P1] `internal/ai/client.go`:1326 `ListModels`(33 行) 生产零调用** — **D1** — LOC≈33 — confidence:**high**
    - 证据：`rg -n '\.ListModels\(' --glob '*.go'` → 0 命中；`rg -n '\bListModels\b'` → 仅 `client.go:1326` 定义 + 4 处 docs/CHANGELOG 历史记录；前端 0 命中。
    - 建议：直接删。模型列表能力已由 `modelengine.Manager.fetchModels`（`engine_models.go:39`）承担，`ai.Client.ListModels` 是被取代的旧路径。

13. **[P1] `internal/ai/image_outpaint.go`:93 `OutpaintToSquare`(14 行) 生产零调用** — **D1** — LOC≈14 — confidence:**high**
    - 证据：`rg -n '\bOutpaintToSquare\b'` → 仅 `image_outpaint.go:91-106`（定义+注释）与 `image_outpaint_test.go:106,108,113,118`；前端 0 命中。对照 `ComposeOutpaint` 有真实调用方 `internal/app/image_handler.go:588`。
    - 建议：删函数 + 其 1 个测试。注意 `image_handler.go:584-587` 是**直接取用户传入的 `p.Expand`**，不经 `OutpaintToSquare`，所以它确实是"自动补正方形"的废弃旁路，而非被内联替代。

14. **[P1] `internal/modelengine/engine_models.go`:120-143 过滤-压缩两趟循环** — **D2/D6** — LOC≈10（可省） — confidence:**high**
    - 证据：`:115-132` 先 `models := make([]ModelInfo, len(result.Data))` 再用 `continue` 跳过后按 `models[i]` 赋值**留下空洞**；`:134-143` 再为 opencode 两个引擎做第二趟压缩。
    - 建议：改成 `models := make([]ModelInfo, 0, len(result.Data))` + `append`，删掉第二趟压缩（10 行）与"过滤后可能有空洞"的注释。

15. **[P1] `internal/channels/weixin/clawbot.go`:212 `HasILink`(1 行) 生产零调用** — **D1** — LOC≈1 — confidence:**high**
    - 证据：`rg -n '\bHasILink\b' --glob '*.go'` → 唯一命中 `clawbot.go:212`（定义）；前端 0 命中；`Server` 的其它方法（`Start`/`Stop`/`IsRunning`/`AssistantID`/`SessionExpired`/`Send`/`Push`）都有 `internal/app/*` 调用方，说明 `Server` 本身活着，只是这一个方法没人用。**它不是任何接口方法**（全仓无 `HasILink()` 的接口签名）。
    - 建议：直接删（1 行）。同类可删的还有 `internal/channels/weixin/clawbot.go:208-213` 一组单行 getter 中的这一个。

16. **[P2] `internal/gaea/provider/provider.go`:310-311 `AuthError.KeySource` / `HasKey` 两个字段从未被读写** — **D1** — LOC≈2 — confidence:**high**
    - 证据：逐字段引用扫描（333 个字段名，`\b字段名\b` 单独检索）→ `KeySource defs=provider.go:307 goNT=1 goT=0`（唯一"引用"就是字段声明行本身 `provider.go:310`）、`HasKey defs=provider.go:307 goNT=1 goT=0`（`provider.go:311`）。**没有任何赋值点，也没有任何读取点**（`provider_test.go` 只读 `AuthError` 的其它字段）。
    - 建议：删 2 个字段。若曾计划让 UI 展示"Key 来自哪个配置源"，请先把赋值点补上，否则字段是死的。

17. **[P2] `internal/ai/types.go`:67-68 `ChatRequest.FrequencyPenalty` / `PresencePenalty`，`:190 `ImageGenerationRequest.ResponseFormat` 三个字段从未被赋值** — **D1** — LOC≈3 — confidence:**high**
    - 证据：`rg -n 'FrequencyPenalty|PresencePenalty|ResponseFormat' --glob '*.go'` → 仅 `types.go:67`、`types.go:68`、`types.go:190` 三行声明（`HasKey` 那类子串误报已用词边界排除）。三者都带 `omitempty`，所以序列化上空值不发，删掉**线上 wire 形状不变**。
    - 建议：删 3 个字段。若想保留可调参能力，至少要让前端/调用方真的能传 —— 目前是"看起来能配、实际恒为 0"的假接口。

18. **[P2] `internal/modelengine/catalog_models.go`:240 `engineCatalogInfo`(3 行) 只有测试用** — **D1** — LOC≈3 — confidence:**high**
    - 证据：`rg -n 'engineCatalogInfo'` → `catalog_models_test.go:40,42,57,58,66,99,149` 共 7 处，**生产 0 处**。对照同族的 `engineCatalogPrice` 有生产调用方 `modelprice.go:134`、`enrichCatalogMeta` 有 `engine_models.go:163`。
    - 建议：把用例改成直接断言 `engineCatalogPrice`/`enrichCatalogMeta` 的可见行为，然后删 `engineCatalogInfo`。若它只是测试探针，移动到 `export_test.go` 并改成未导出测试专用函数。

19. **[P2] `internal/gaea/provider/llm.go`:51 `errNilProvider` / `provider.go`:394 `contextOverflowMarkers` / `schema_canonicalize.go`:80 `setLikeSchemaArrays` / `image_comfyui_workflows.go`:272 `preferredKontextResolutions` 等"只用一次"的包级常量** — **D5/D7** — LOC≈1-6 each — confidence:**medium**
    - 证据：单符号检索各得 `goNT=1`，且这唯一引用都在**定义文件内部**（`llm.go:58`、`provider.go:412`、`schema_canonicalize.go:93`、`image_comfyui_workflows.go:289`）。
    - 建议：**不是死代码**（确实被用），但这类"单点常量/表"可以直接内联到使用处，减少顶层符号与阅读跳转。**低优先，纯口味问题；不要为此开批次。**

20. **[P2] `internal/ai/client.go`:193-203 `failoverTarget` 用 `for` + 无条件 `return` 表达"取第一个"** — **D2** — LOC≈3 — confidence:**high**
    - 证据：`:200-203`
      ```go
      for _, cand := range c.engineMgr.FailoverCandidates(failedEngineID) {
          return cand.ID, cand.DefaultModel, true
      }
      return "", "", false
      ```
    - 建议：改 `if cs := c.engineMgr.FailoverCandidates(id); len(cs) > 0 { return cs[0].ID, cs[0].DefaultModel, true }`，语义逐字节等价（`FailoverCandidates` 已保证顺序，见 `health_probe.go:258`）。

---

## 全量发现表

> LOC 为 AST 精确统计（`go/parser`，函数体起止行）。置信度含义：**high** = 逐符号 `\b名称\b` 检索 + 接口/字符串双查已完成；**medium** = 检索通过但存在字符串驱动或绑定生成的残余风险；**low** = 需人工拍板。

| 文件:行 | 符号 | 类型 | LOC | 置信度 | 证据 | 建议 |
|---|---|---|---:|---|---|---|
| `internal/gaea/provider/retry.go:15-146` | `BackoffStrategy` 等 9 个符号（整文件） | D1 | ≈136 | high | `rg '\b(DefaultBackoff\|RateLimitBackoff\|ParseRetryAfter\|IsRetryableStatus\|IsTransientNetErr\|RetryPolicy)\b' --glob '*.go'` → 30 命中全在 `retry.go` + `retry_test.go` | 接线或整文件（含测试）删除 |
| `internal/gaea/provider/retry.go:31` | `DefaultBackoff` | D1 | 16 | high | 仅 `retry_test.go:21,28,35,68` | 同上 |
| `internal/gaea/provider/retry.go:41` | `RateLimitBackoff` | D1 | 8 | high | 仅 `retry_test.go:42,49` | 同上 |
| `internal/gaea/provider/retry.go:52` | `BackoffStrategy.Duration` | D1 | 16 | high | 仅 `retry_test.go:11,14` | 同上 |
| `internal/gaea/provider/retry.go:70` | `BackoffStrategy.Sleep` | D1 | 12 | high | 仅 `retry_test.go:77-93` | 同上 |
| `internal/gaea/provider/retry.go:84` | `RetryPolicy` | D1 | 13 | high | 仅 `retry.go:89`（自身字段）+ 2 处 docs | 同上 |
| `internal/gaea/provider/retry.go:101` | `ParseRetryAfter` | D1 | 26 | high | 仅 `retry_test.go:127,136` | 同上（注意 `Retry-After` 解析是有用能力，接线价值高） |
| `internal/gaea/provider/retry.go:129` | `IsRetryableStatus` | D1 | 5 | high | 仅 `retry_test.go:160` | 同上 |
| `internal/gaea/provider/retry.go:138` | `IsTransientNetErr` | D1 | 9 | high | 仅 `retry_test.go:179` | 同上；语义与 `ai/client.go:215 isTransferableChatError` 重复 |
| `internal/gaea/provider/provider.go:84` | `SanitizeToolPairing` | D1 | 53 | high/medium | `rg` 35 命中全在 `provider.go`+3 个测试；生产 0 | 先判断是否该在 `bridge.Chat` 接线，否则连测试一起删 |
| `internal/gaea/provider/provider.go:141` | `isToolResultOrBridge` | D1 | 10 | high | 唯一调用 `provider.go:111`（`SanitizeToolPairing` 内） | 随上条 |
| `internal/gaea/provider/provider.go:153` | `extractToolResults` | D1 | 9 | high | 唯一调用 `provider.go:121` | 随上条 |
| `internal/gaea/provider/provider.go:169` | `pairToolResults` | D1 | 27 | high | 唯一调用 `provider.go:123` | 随上条 |
| `internal/gaea/provider/provider.go:199` | `idDistinct` | D1 | 13 | high | 唯一调用 `provider.go:171` | 随上条 |
| `internal/gaea/provider/provider.go:71` | `interruptedToolResult`（const） | D1 | 1 | high | 仅 `provider.go:180,191`（`pairToolResults` 内） | 随上条 |
| `internal/gaea/provider/provider.go:310` | `AuthError.KeySource` | D1 | 1 | high | 字段扫描：唯一提及=声明行本身；无赋值/读取 | 删字段（或补齐赋值点） |
| `internal/gaea/provider/provider.go:311` | `AuthError.HasKey` | D1 | 1 | high | 同上 | 删字段 |
| `internal/ai/client.go:1326` | `ListModels` | D1 | 33 | high | `rg '\.ListModels\('` → 0；前端 0 | 删 |
| `internal/ai/copilot.go:20` | `GhostComplete` | D1 | 77 | medium | Go 侧仅定义；前端仅 `wails.d.ts:221`（签名不匹配=陈旧） | 连 `wails.d.ts` 一起删，跑绑定一致性测试 |
| `internal/ai/copilot.go:245` | `GenerateBeats` | D1 | 43 | medium | 同上（`wails.d.ts:224`） | 同上 |
| `internal/ai/copilot.go:292` | `GenerateProseFromBeat` | D1 | 86 | medium | 同上（`wails.d.ts:225`） | 同上 |
| `internal/ai/image_outpaint.go:93` | `OutpaintToSquare` | D1 | 14 | high | 仅 `image_outpaint_test.go:108,113,118` | 删函数+测试；`ComposeOutpaint` 保留（有 `image_handler.go:588` 调用） |
| `internal/channels/weixin/clawbot.go:212` | `HasILink` | D1 | 1 | high | 全仓唯一命中=定义行；非接口方法；前端 0 | 删 |
| `internal/modelengine/catalog_models.go:240` | `engineCatalogInfo` | D1 | 3 | high | 仅 `catalog_models_test.go` 7 处 | 改测试断言后删，或移入 `export_test.go` |
| `internal/modelengine/catalog_models.go:82` | `newModelCatalog` | D1/D5 | 5 | high | 仅 `catalog_models_test.go:221,237,247`；注释自述"测试隔离用" | **有意保留的测试缝**；建议移入 `export_test.go` 明确意图，不删 |
| `internal/ai/types.go:67` | `ChatRequest.FrequencyPenalty` | D1 | 1 | high | 全仓仅 `types.go:67` | 删（`omitempty`，wire 不变） |
| `internal/ai/types.go:68` | `ChatRequest.PresencePenalty` | D1 | 1 | high | 全仓仅 `types.go:68` | 删 |
| `internal/ai/types.go:190` | `ImageGenerationRequest.ResponseFormat` | D1 | 1 | high | 全仓仅 `types.go:190` | 删 |
| `internal/modelengine/engine_models.go:67-68` | `EngineGLM` 鉴权分支 | D2 | 2 | high | `:40-43` 已提前 return GLM → 恒假 | 删分支 |
| `internal/modelengine/engine_models.go:73-74` | `EngineModelHub` 鉴权分支 | D2 | 2 | high | `:44-49` 已提前 return ModelHub → 恒假 | 删分支 |
| `internal/modelengine/engine_models.go:95-96` | `EngineGLM` 401 文案分支 | D2 | 2 | high | 同 `:40-43` | 删分支 |
| `internal/modelengine/engine_models.go:101-102` | `EngineModelHub` 401 文案分支 | D2 | 2 | high | 同 `:44-49` | 删分支 |
| `internal/modelengine/engine_models.go:63-81` | `fetchModels` 七分支鉴权链 | D4/D6 | 19 | high | 6 条 `else if` 全执行同一句 `Header.Set("Authorization", …)` | 抽 `engineKey(EngineType) string` |
| `internal/modelengine/engine_models.go:89-106` | `fetchModels` 七分支 401 文案链 | D4/D6 | 18 | high | 与上条同序同条件，第二次 | 改 `map[EngineType]string` |
| `internal/modelengine/engine_models.go:115-143` | 过滤-压缩两趟循环 | D2 | 29 | high | `make(…, len())` + `continue` 留空洞 + 二次压缩 | 改 `append`，删第二趟 |
| `internal/ai/client.go:208` | `transferableStatusRe` + `:759/:778/:780` | D2/D6 | 20 | high | 错误先 `Sprintf("API 错误 (HTTP %d)…")`，再正则解析回来 | 改 typed `HTTPStatusError` + `errors.As` |
| `internal/ai/client.go:193-203` | `failoverTarget` 的 for-return | D2 | 3 | high | 循环体唯一语句是无条件 `return` | 改 `len()>0` 取首个 |
| `internal/channels/weixin/media_download.go:36-46` | `cgnatRange` + `mustCIDR` | D4 | 11 | high | 与 `netclient/ssrf.go:14-25` 逐字节相同 | 删，改用 netclient |
| `internal/channels/weixin/media_download.go:51-60` | `blockedMediaIP` | D4 | 10 | high | ≡ `netclient.BlockedInternalIP`（多一个测试开关） | 收敛到 netclient + 豁免参数 |
| `internal/channels/weixin/media_download.go:65-86` | `mediaTransport` | D4 | 22 | high | ≡ `netclient.GuardedClient`，**且未包含 `ssrf.go:73-87` 的 IN2-04 多 IP 回退修复**（`:82` 仍 `ips[0]`） | 删，改用 `GuardedClient`（顺带拿到修复） |
| `internal/gaea/plugin/ssrf.go:41-60` | `cgnatRangePlugin` + `mustCIDR` | D4 | 20 | medium | 第三份 SSRF CIDR 副本；有独立 plugin 开关，需先等价性比对 | 收敛（域外，仅登记） |
| `internal/modelengine/engine_models.go:264` | `zhipuErrorMessage` | D4 | 12 | high | 与 `ai/image_glm.go:141` 同名近似、同注释、同解析目标 | 二选一，注意 import 方向 |
| `internal/ai/image_glm.go:141` | `zhipuErrorBodyMessage` | D4 | 19 | high | 同上 | 同上 |
| `internal/ai/client.go:729-733` 等 14 处 | `Bearer`+`Content-Type` 手写 | D4 | ≈0 | high | 见 Top 8 的站点清单 | 抽 `netclient.SetJSONBearer` |
| `internal/ai/client.go:710-791` | `doStreamRequest` 私有退避循环 | D4 | 82 | high | `for attempt := 0; attempt <= len(backoff)` + `time.After`，与 `retry.go`/`health_probe.go`/`clawbot.go` 各自重写 | 收敛到一个 backoff helper |
| `internal/modelengine/health_probe.go:106/130/185` | `probeRound`/`probeOne`/`probeEndpoint` | D4 | 21+51+31 | medium | 自带探测重试与错误脱敏（`sanitizeProbeError`），与 `retry.go` 语义重叠 | 保持（探测有独立超时/脱敏需求），只复用 backoff |
| `internal/modelengine/modelprice.go:120` | `estimatePrice` → `glmCatalogPrice`/`engineCatalogPrice`/`modelPricing` | D4 | 24 | medium | 三层查价，注释自述为分层设计；但 `glmCatalogPrice:369` 与 `modelPricing:28`+`modelprice.go:130` 各自实现"最长前缀优先"匹配 | 抽 `matchLongestPrefix(table, id)` 单点，不动分层 |
| `internal/ai/types.go:28` vs `internal/gaea/provider/provider.go:52` | `ChatToolFunctionSpec` ≡ `ToolSchema` | D8 | 4 | high | AST 结构指纹完全相同：`{Description:string, Name:string, Parameters:json.RawMessage}` | 二选一（`bridge.go:212` 已在做转换，可让 bridge 承担映射） |
| `internal/ai/image_glm.go:32` vs `internal/ai/image_openai.go:25` | `GLMImageBackend` ≡ `OpenAIImageBackend`（字段集） | D8 | 0 | medium | AST 指纹相同 `{apiKey, baseURL, httpClient}` | 可提公共 `openAICompatBackend` 基结构；但两者行为分化（GLM 有专属错误体/审核分支），**收益低、风险中，不建议本批动** |
| `internal/ai/image_backend.go:16-22` vs `:43-49` | `ImageBackendKind*` vs `ImageBackendType*`（`comfyui`/`glm` 同值两套常量） | D8/D5 | 0 | high | `:45 ImageBackendTypeComfyUI="comfyui"` == `ImageBackendKindComfyUI`；`:48 ImageBackendTypeGLM="glm"` == `ImageBackendKindGLM`（后者定义在 `image_glm.go:25`） | **注释已自述（IN2-07）"需配置迁移拍板"→ 保留**，仅在报告登记 |
| `internal/gaea/provider/provider.go:325-360` | `Factory`/`registry`/`Register`/`Kinds` | D5 | 36 | medium | 生产侧仅 `bridge/bridge.go:176` 注册 1 个 kind；`Kinds()` 仅用于 `provider.go:342` 自己的报错与测试 | **保留**（扩展缝 + 测试大量注册 mock kind）；不为单一实现做重构 |
| `internal/ai/client.go:89-94` | `chatBackoffOrDefault` | D5 | 6 | medium | 纯取值转发；`NewClient:103` 恒把 `chatRetryBackoff` 设为非空 → `defaultStreamRetryBackoff` 回退分支生产不可达 | 可内联；或保留作测试注入口（测试确实注入短间隔），**建议保留** |
| `internal/analysis/analysis_v2.go:116` | `deriveLegacyAnalysis` | D5 | 64 | high | 唯一调用 `analysis_v2.go` 内（`AnalyzeChapter` 绑定兼容层）；前端依赖旧 wire 键 | **保留**（有明确日落条件：前端迁 V2 后删）；建议在注释里写明迁移触发条件 |
| `internal/ai/copilot.go:99/152/194` | `CmdKEdit`/`OfficeEditText`/`XlsxEditOps` | — | 107/42/40 | high | 有真实调用方：`app/copilot_handler.go:24`、`gaea_docx_edit.go:35`、`gaea_xlsx_edit.go:79` | **不是死代码，勿动**（与 Top 11 的三兄弟形成对照） |
| `internal/ai/image_comfyui_workflows.go:144` | `buildQwenProfileImg2Img` 的 `width,height` 形参 | D7 | 0 | high | AST 形参使用分析：`width`/`height` 在函数体内 0 次出现（10 参函数） | 删这两个形参 + 更新 1 处调用点（纯签名清理；注意是否与接口/函数类型绑定） |
| `internal/channels/weixin/clawbot.go:1060` | `apiPost` 缺 `ctx` 形参，内部 `context.Background()` | D7 | 0 | medium | `:1068 context.WithTimeout(context.Background(), timeout)` → 调用方无法取消 | 加 `ctx` 首参，调用方传链上 ctx（`notifyStart`/`notifyStop`/`getUpdates` 等） |
| `internal/netclient/netclient.go:172/231/240` | `defaultTransport`/`environmentProxyFunc`/`autoProxyFunc` | D4 | 25 | medium | 三条各自从环境推导代理：`http.ProxyFromEnvironment`(172)、`httpproxy.FromEnvironment()`(232/241)；`environmentProxyFunc` 仅被 `:200` 调用 | 抽 `envProxyFunc()` 单点；注意 `defaultTransport` 走 stdlib 语义，合并需确认等价 |
| `internal/modelengine/glm_catalog.go:369` | `glmCatalogPrice` | D4 | 30 | medium | 与 `catalog_models.go:245 engineCatalogPrice`、`modelprice.go:28 modelPricing` 三处查价并存（有生产调用 `modelprice.go:130`） | 见上"matchLongestPrefix" |
| `internal/ai/client.go`（整文件） | 60 个函数 / 1160 行 | D6 | 0 | high | AST 统计；最大函数 `parseStreamEvents` 159 行、`chatOnce` 126 行 | 按 4 个文件拆分 |
| `internal/ai/image_comfyui.go`（整文件） | 914 行 / `GenerateImage` 196 行 / `pollComfyProgress` 140 行 / `checkHistory` 102 行（嵌套 8） | D6 | 0 | high | AST：`GenerateImage` L204-399；`pollComfyProgress` L739-878；`checkHistory` L893-994 | 拆 `GenerateImage` 为 校验/构建/提交/轮询/收集 五段 |
| `internal/channels/weixin/clawbot.go`（整文件） | 893 行 / `handle` 126 行（嵌套 5） / `pollLoop` 97 行 | D6 | 0 | high | AST：`handle` L531-656、`pollLoop` L433-529 | 按消息类型抽 `handleXxx` |
| `internal/ai/image_comfyui_workflows.go`（整文件） | 296 行 / 15 个 `buildXxxWorkflow` | D6/D5 | 0 | medium | AST：`buildQwenProfileImg2Img`(25)/`buildZetaImg2Img`(9)/`injectLoraNodes`(19)… | **保留**：一模型一函数是清晰的，别合并 |
| `internal/analysis/story_memory.go:43` | `ExtractStoryMemories` | D6 | 149 | high | AST LOC=149 | 抽提示词构建 + 解析两段 |
| `internal/analysis/foreshadow_sync.go:222` | `syncPlant` | D6 | 101 | high | AST LOC=101 | 中优先 |
| `internal/auth/oauth.go:59` | `DoLogin` | D6 | 135 | high | AST LOC=135；唯一调用 `internal/app/auth_handler.go:61`、`app.go:793` | 抽 `buildAuthURL`(已存在)/`startCallbackServer` |
| `internal/modelengine/stats.go:375` | `statsRecorder.summary` | D6 | 102 | high | AST LOC=102 | 中优先 |
| `internal/modelengine/engine.go:163` | `NewManager` | D6 | 121 | high | AST LOC=121 | 中优先 |
| `internal/modelengine/engine_connect.go:100` | `BuildChatURL`（控制嵌套 7） | D6 | 37 | high | AST maxNesting=7 | 用早返平铺 |
| `internal/gaea/provider/schema_canonicalize.go:14/85` | `compressSchema`/`canonicalizeSchemaValue`（嵌套 5） | D6 | 40/37 | high | AST maxNesting=5 | 递归函数，**仅登记不动** |
| `reasonix.toml:45-49,63-66` | `[skills]` / `[statusline]` 空 section | D1/D3 | 0 | high | 两个 section 内**全部是注释**，无有效键 | 删空 section（见 F-50） |
| `reasonix.toml:52` | `[permissions] allow` 30+ 条一次性规则 | D7 | 0 | high | 含硬编码绝对路径、`git clone hermes-agent`、`nvidia-smi`、`pip list`、`ls docs/superpowers/specs/` 等一次性命令，及 `Bash(cd C:\AI\wubigrok && go build ./... 2>&1)` | 需人工确认；建议只保留通用规则，删一次性条目 |
| `internal/gaea/provider/llm.go:51`、`provider.go:394`、`schema_canonicalize.go:80`、`image_comfyui_workflows.go:272` | `errNilProvider`/`contextOverflowMarkers`/`setLikeSchemaArrays`/`preferredKontextResolutions` | D5/D7 | 1-6 | medium | 各自 `goNT=1` 且唯一引用在定义文件内 | 可内联；**低优先，不建议单独开批** |

---

## 屎山清单（结构性重构建议，含风险）

### A. 超长函数（AST 精确 LOC，非测试）

| 函数 | 位置 | LOC | 控制嵌套 | 风险 |
|---|---|---:|---:|---|
| `ComfyUIBackend.GenerateImage` | `internal/ai/image_comfyui.go:204` | **196** | 4 | 中（5 个后端分支 + 轮询 + 超时收割交织） |
| `Client.parseStreamEvents` | `internal/ai/client.go:795` | **159** | 4 | **高**（SSE 解析 + tool_call 拼装 + usage 统计 + panic 补发 + 信号量释放全在一处） |
| `ExtractStoryMemories` | `internal/analysis/story_memory.go:43` | 149 | 3 | 中 |
| `ComfyUIBackend.pollComfyProgress` | `internal/ai/image_comfyui.go:739` | 140 | 4 | 中 |
| `DoLogin` | `internal/auth/oauth.go:59` | 135 | 4 | 中（浏览器回调 + PKCE + 换 token） |
| `Manager.fetchModels` | `internal/modelengine/engine_models.go:39` | 128 | **9** | **高**（嵌套 9 层是全域最高） |
| `Client.chatOnce` | `internal/ai/client.go:463` | 126 | 4 | 高 |
| `Server.handle` | `internal/channels/weixin/clawbot.go:531` | 126 | 5 | 中 |
| `NewManager` | `internal/modelengine/engine.go:163` | 121 | 3 | 低 |
| `Client.ChatStream` | `internal/ai/client.go:593` | 109 | 4 | 高 |
| `statsRecorder.summary` | `internal/modelengine/stats.go:375` | 102 | 3 | 低 |
| `ComfyUIBackend.checkHistory` | `internal/ai/image_comfyui.go:893` | 102 | **8** | 中 |
| `syncPlant` | `internal/analysis/foreshadow_sync.go:222` | 101 | 3 | 低 |

### B. 最大文件（非测试行数）

| 文件 | 行 | 判断 |
|---|---:|---|
| `internal/ai/client.go` | 1160 | **应拆**（60 个函数） |
| `internal/ai/image_comfyui.go` | 914 | **应拆**（ComfyUI HTTP + 工作流 + 轮询 + 预热 + LoRA） |
| `internal/channels/weixin/clawbot.go` | 893 | **应拆**（协议客户端 + 消息分派 + 媒体 + 通知） |
| `internal/modelengine/stats.go` | 479 | 可拆（记账 + 聚合 + 落盘 + 汇率） |
| `internal/gaea/provider/provider.go` | 363 | 拆后（去掉死代码）≈240 |
| `internal/channels/weixin/media_download.go` | 344 | 收敛 SSRF 后 ≈300 |

### C. 建议的重构顺序（按风险从低到高，每步独立可回滚）

1. **纯删除批**（零行为风险）：Top 1 若走"删除路线"、12、13、15、16、17、18。
2. **等价替换批**（行为逐字节不变，有测试兜底）：`engine_models.go` 4 个死分支 + 两条七分支链收敛 + 两趟循环改 `append`；`client.go` `failoverTarget`；`client.go` typed error 替代正则反解。
3. **D4 收敛批**（收益最大，影响面也最大）：`netclient.SetJSONBearer` 统一 14 处鉴权头；`media_download.go` 的 SSRF 四件套改调 netclient（**顺带修掉 IN2-04 缺失**）。
4. **文件拆分批**（无行为变更）：`client.go` → 4 文件；`image_comfyui.go` → 2-3 文件；`clawbot.go` → 2 文件。
5. **函数级重写批**（有行为风险，必须单独批次 + 现有测试）：`parseStreamEvents`、`fetchModels`、`GenerateImage`、`handle`。

**风险提示**：
- 第 3 步动 `weixin` 的 SSRF 会改变**测试可达性**（`mediaAllowLoopback` 是 `httptest` 127.0.0.1 能通的唯一原因），`media_download_test.go`(189 行)+`media_file_test.go`(295 行) 是回归网，替换时必须保留回环豁免开关。
- 第 2 步的 `engine_models.go` 七分支链收敛时，**空 Key 不设 Authorization 头**是本地无鉴权服务的依赖语义（`engine_custom_test.go:320` 断言"空 Key 不应带 Authorization 头"），helper 必须保留该行为。
- 第 5 步 `parseStreamEvents` 承担 `releaseSem`（信号量释放）与 panic 路径补发错误块，重写时极易引入并发泄漏；**建议只做提取不做逻辑改动**。

---

## 不建议动的地方（说明为什么）

1. **`func (f *fileItem) UnmarshalJSON` / `func (t *imageItem) UnmarshalJSON`（`internal/channels/weixin/clawbot.go:348/400`）** —— 逐符号检索显示 `goNT=0`、`goT=0`，**看起来像死代码，实际是 `encoding/json` 的 `Unmarshaler` 接口实现，由 `json.Unmarshal` 通过反射调用**。删了会静默改变 JSON 解析行为（这两处正是 `itemElem`/`imageItem` 兼容老协议形态的入口）。同类：`internal/gaea/provider/provider.go:379 StreamInterruptedError.Unwrap`（`errors.Is/As` 依赖）、`provider.go:315/372 AuthError.Error`/`StreamInterruptedError.Error`。
2. **`internal/netclient/sysproxy/system_windows.go:31-41` 的 `dwAccessType`/`dwReserved`/`lpvReserved`** —— 字段扫描把它们列为"从未被读写"，但它们是 **Windows `WINHTTP_AUTOPROXY_OPTIONS`/`IE_PROXY_CONFIG` 的内存布局占位字段**，必须按 ABI 保留原样，否则不安全的指针/反射调用会读错偏移。**绝对不要删**。
3. **`internal/gaea/provider/retry.go` 整文件** —— 虽然生产零引用，但它是一套**完整且已测试**的重试工具箱，而域内目前有 **4 套各自手写的退避**（`ai/client.go:710`、`health_probe.go`、`clawbot.go:1101 sleepOrStop`、`catalog_remote.go`）。**正确动作多半是"接线"而不是"删除"**。删它等于把唯一一份写好的实现扔掉，再继续维护 4 份私有版本。**建议主控决定方向后一次性执行。**
4. **`internal/gaea/provider/provider.go:84 SanitizeToolPairing` 及其 5 个 helper** —— 同上：它是"发给模型前修复 tool_call/tool_result 配对"的护栏，测试覆盖充分（3 个测试文件、30+ 断言），**没有被生产调用很可能是漏接线**。删代码前请确认是否应在 `bridge.Chat` 入口调用。
5. **`internal/modelengine/catalog_models.go:82 newModelCatalog`** —— 唯一调用方是测试，但它是**有意的测试隔离缝**（注释明写"测试隔离用：坏 JSON/版本不符"）。建议移入 `export_test.go` 表达意图，**不要删**。
6. **`internal/ai/image_backend.go:43-49 ImageBackendType*` 与 `:16-22 ImageBackendKind*` 两套常量** —— 注释自己写明：类型名会落盘到配置、进前端、进绑定返回值，统一会**破坏配置文件字段值**（IN2-07），需配置迁移拍板。**本批不要动**，仅登记。
7. **ComfyUI 的模型名分派（`image_comfyui.go` 的 `txt2imgWorkflows`/`img2imgWorkflows` map 键、`lookupTxt2imgBuilder`）** —— 模型名（`krea2`、`flux`、`z-image-turbo`…）是**字符串驱动**的（前端/配置传入），map 分派本身是良好设计。**不要因为"这些字符串在 Go 里只出现一两次"就判死。**
8. **`internal/analysis/analysis_v2.go:116 deriveLegacyAnalysis`** —— 有生产调用，是 `AnalyzeChapter` 绑定返回键的兼容层（注释"前端零改动 t4-C1"）。删它会破坏前端。**保留**，建议加注日落条件。
9. **`internal/ai/copilot.go` 的 `CmdKEdit`/`OfficeEditText`/`XlsxEditOps`** —— 与 Top 11 的三个"死"函数同文件，但**这三个都有真实调用方**（`app/copilot_handler.go:24`、`gaea_docx_edit.go:35`、`gaea_xlsx_edit.go:79`）。清理时不要整文件处理。
10. **`internal/gaea/provider/provider.go:325-360` 注册表（`Factory`/`Register`/`Kinds`）** —— 生产只注册 1 个 kind，看着像过度抽象，但**测试大量 `provider.Register` mock kind**（`boot_test.go:38,94,170`、`space_assembly_test.go:50,142`、`llm_seam_test.go:67` 等），是刻意的扩展缝。**保留。**
11. **`internal/gaea/provider/schema_canonicalize.go` 的递归规范化（`compressSchema`/`canonicalizeSchemaValue`，嵌套 5）** —— 深度受 `compressSchemaMaxDepth` 常量约束，是**有界递归**，不是失控嵌套。**登记不动。**
12. **`internal/ai/image_comfyui_workflows.go` 的 15 个 `buildXxxWorkflow`** —— 一模型一构建器，函数都很短（3-25 行），可读性良好；**不要合并成带 switch 的巨型函数**（反而是当前设计在避免 D6）。

---

## F-50 · `reasonix.toml`（专项）

**核心结论：`reasonix.toml` 不是 Gaea 的配置，而是 DSH/Reasonix agent harness 的配置文件。** 证据：文件第 1-3 行自述 `# Reasonix configuration.` / `# Resolution order: flag > ./reasonix.toml > ~/AppData/Roaming/reasonix/config.toml > built-in defaults.` / `# Secrets come from the environment via api_key_env; never put keys here.`；Gaea 全仓检索 `rg -n 'reasonix\.toml' --glob '!clones/**'` → **唯一命中是 `reasonix.toml:2` 自身的注释**，没有任何 Go 代码读取它。

因此"配置项是否指向已不存在的代码/字段"这个问题的答案是：**Gaea 侧无此耦合，不存在指向失效代码的配置项**。剩下的都是 harness 配置自身的卫生问题：

| 位置 | 问题 | 类型 | 建议 |
|---|---|---|---|
| `reasonix.toml:45-49` | `[skills]` section 内**全部被注释**，无有效键 → 空 section | D1 | 删 section（或取消注释其一） |
| `reasonix.toml:63-66` | `[statusline]` section 内**只有注释** → 空 section | D1 | 同上 |
| `reasonix.toml:52` | `permissions.allow` 30+ 条，绝大多数是**一次性历史命令**：`git clone https://github.com/NousResearch/hermes-agent.git`、`nvidia-smi`、`python --version && pip list`、`ls docs/superpowers/specs/`、一次性 `git log -- 三个文件` 等；且大量硬编码绝对路径（`C:\AI\wubigrok`、`/c/AI/wubigrok`、`D:/AI/hermes-agent`） | D7 | **需人工确认**；建议只留通用规则（`Bash(curl -s:*)`、`explore`、`Edit`、`research`、`run_skill`），删一次性条目 |
| `reasonix.toml:52` | 其中一条 `Bash(cd C:\AI\wubigrok && go build ./... 2>&1)` **在白名单里放行了 `go build ./...`** | 风险提示 | 本次审计被明确要求不要跑 `go build ./...`（有人在跑）。这条 allow 规则正是"能跑起来"的通道，建议主控知悉 |
| `reasonix.toml:7,10-12,16-17,21-22,24-29,40-43,46-49,54-59,64-66,68-77` | 大量注释掉的**示例配置**（`# system_prompt_file`、`# [lsp.servers.go]`、`# [[plugins]]`…） | D3 | **不是**废弃代码，是配置模板文档。**保留**（删了反而降低可用性） |
| `reasonix.toml:5-6` | `config_version = 3`、`default_model = "deepseek-flash"` | — | 无法核验：本机 **DSH checkout 路径不存在**（`C:\Users\wubi\AppData\Local\Programs\DeepSeek Harness\resources\app.asar\dsh` 未找到，`app.asar` 为打包文件无法 grep）。**列为未验证项** |

---

## 未覆盖 / 验证边界（供主控判断置信度）

1. **`reasonix.toml` 键名与 harness 代码的对照无法完成**：任务给定的 DSH checkout 目录在本机不存在（已确认 `resources/` 下只有 `app.asar`、`app.asar.unpacked`、`runtime`，无 `dsh/` 展开目录）。凡涉及"配置项是否指向已消失的 harness 字段"的判断，本报告只做了 Gaea 侧反向检索。
2. **`internal/gaea/plugin/ssrf.go`（副本 B）在本次审计范围之外**，仅作为 D4 收尾的旁证登记，未做完整等价性分析。
3. **Wails 绑定面**：`internal/ai` 与 `internal/channels/weixin` 的内部方法**不直接绑定**（`internal/app/bindings_manifest.go` 绑定的是 `App` 门面），已用 `rg 'ai\.NewClient'`、`bindings_novel.go` 交叉确认。但 `frontend/src/types/wails.d.ts` 是**生成物**，其重生成时机未能从仓库内确定 —— 这是 Top 11（copilot 三函数）置信度只能给 `medium` 的唯一原因。
4. **`go build`/`go vet`/测试未运行**（任务禁止）。因此"删除后仍能编译"属于**静态推断**：所有 D1 条目都已检查是否存在同包内的未导出互调（例如 `pairToolResults` 只被 `SanitizeToolPairing` 调用，两者必须同批处理）。
5. 逐符号检索覆盖 **589 个顶层符号**；结构体字段另做 **333 个字段名**的独立检索。**包级 `blockvar`（2783 条，含 struct field）中的局部常量块未逐一检索**，可能仍有少量遗漏的死常量。

---

*报告结束。本次审计未修改、未删除、未新建任何仓库文件（`.audit/reports/go-ai-engine.md` 为任务指定交付物）。*
