# 众神殿席位卡死根修与 DSH 前缀缓存 · 2026-11-06

> 用户报两件事：①众神殿的 Agent 经常卡死；②DSH 的命中缓存低下，并追问「是否是真的调用 DSH」。
> 本文是这两条的取证、根因与落地记录。所有数字都来自本机实弹，不来自推测。

## 1. 先回答「是不是真的调用 DSH」

**是，而且有可核对的账。** 2026-11-06 实弹会话
`pan_1793963799148_6aecf8f40f0a`（4 轮 4 席，耗时 83 分钟）里：

- DSH 席位四次陈词分别用时 **368s / 147s / 316s / 403s**，全部真实产出正文；
- 该场 DSH 的 `tool_started` 事件共 **209 条**：`pwsh` 104、`read` 69、`grep` 30、`glob` 5、`working_directory` 1；
- 逐轮工具数 **87 / 31 / 53 / 38**，即「一席一轮 87 次工具调用」；
- 事件流里每条都是 DSH headless 的真实协议（`session` / `status` / `tool_call` /
  `tool_result` / `step_end` / `final`），不是宿主伪造的占位文本。

调用链本身也是真的：`internal/pantheon/agents.go` 的 DSH 适配器解析
`resources/runtime/cli/bin/dsh.cmd`，以 `cmd.exe /d /c call dsh.cmd --profile headless …`
启动，prompt 走 stdin。手工复跑同一条命令可拿到 `{"type":"session","sessionId":"session-…"}`
与 `~/.dsh/sessions/--C-AI-wubigrok--/<id>/session.v4.jsonl.zstd`，会话库里有实体记录。

**为什么用户会怀疑「不是真的 DSH」**，有两个合理来源：

1. **同一个 DSH 席位每轮都新开进程、新开会话**，模型没有跨轮记忆，看起来像另一个更笨的 agent；
2. **凭据跟着工作目录走**：DSH headless 的 DeepSeek 凭据来自启动目录的 `.env` 或环境变量
   （`~/.dsh/.credentials.yaml` 里没有 key）。把议事工作区切到没有 `.env` 的目录、Gaea 又没配
   deepseek 引擎时，每一轮都在 1.6s 内以 `MISSING_CREDENTIAL` 结束——不是卡死，但一样交不出陈词。

## 2. 卡死的根因（四条，按证据强度排序）

### 根因 A：席位被当成「完整 agent」用，自主循环没有上限

圆桌只想要一段陈词，但外部 CLI 是自主循环：`prompts.go` 的只读工具段要求「优先用工具核验」，
还给了 pwsh 的中文编码纪律，DSH 便真的去把仓库翻一遍（87 次工具/轮）。
每一次工具调用都是一次完整 LLM 往返，单轮 6 分钟是「几十次往返」的必然结果。

### 根因 B：DSH 自己的工具超时是 30s，长命令就是白等

实弹抓到 `{"status":"error","error":{"name":"ToolTimeoutError","code":"TOOL_TIMEOUT"}}`
——工具 30s 超时后模型换路再试，每一步都是分钟级等待。宿主这边此前没有任何单步约束，
只有 `defaultAgentLimit = 8 分钟` 的总上限兜底。

### 根因 C：8 分钟总上限 + 「空结果自动重试」会把单席放大到 16 分钟

`runAgentWithRetry` 对空结果自动重试一次；一场 4 轮 4 席的议事里，任一席超时/空结果都
翻倍。历史会话 `pan_1793681319746_a952dab6337a` 跑了 **140 分钟**才被用户取消。

### 根因 D：工具事件逐条整份落盘，且在 `m.mu` 里做

`recordToolTrace` 每条工具事件都 `store.save(session)` = 整份 `session.json`（该场 2.4MB）
序列化 + fsync，**持锁**。209 次调用 = 418 次整份写盘；这段时间里取消按钮、插话、
快照读取都排在锁后面，界面表现为「点了没反应」。

## 3. 缓存为什么低

DSH 的前缀缓存本身是好的。同一会话连续两步实测：

| 调用 | input | cacheRead | 命中率 |
|---|---|---|---|
| 同会话第 1 步 | 606 | 5632 | 90.3% |
| 同会话第 2 步 | 419 | 6272 | 93.7% |
| **新进程第 1 次**（冷） | 6207 | 0 | 0% |
| **续接旧会话第 1 次** | 191 | 6016 | 96.9% |

问题在众神殿的调用方式：**每轮 `--profile headless --json -` 新开会话**，能命中的只有
system + 工具表这段固定前缀（约 6k token）；本轮议题、历史立场卡、材料、工具输出
全部按 miss 计费，命中率被压在 ~25%。续接同一会话后，前缀包含「系统+全部历史步」，
命中率直接抬到 96%+（本轮 live 测试实测 **96.7%，6016/6224**）。

## 4. 落地（四刀 + 前端）

### 刀 1：DSH 席位会话续接（提命中率，降重复探索）

- `Participant.EngineSessionID` 记住引擎会话；`agentAdapter.buildResume` 在后续轮次追加
  `--session-id <id>`（首次不带，引擎对未知 ID 是硬错误）。
- 引擎返回的 `sessionId` 从事件流解析后落回 Participant，随 `session.json` 持久化；
  续跑（Resume）沿用，重开（Restart）本来就是新会话。
- 失败的三种收敛：护栏杀掉的调用保留会话（会话没坏）；引擎报错且零正文则丢掉会话
  （避免一轮接一轮撞同一个坏会话）；有正文的失败保留会话。

### 刀 2：运行护栏（治卡死）

新增 `seatguard.go`，一次调用三条确定性上限：

| 闸 | 默认 | 触发后 |
|---|---|---|
| 工具预算（按工具 ID 去重） | 120 次 | 终止本轮，报「工具调用超出议事预算」 |
| 无进展看门狗（任何输出事件都喂活性） | 120s | 终止，报「长时间没有任何进展」 |
| 开局静默闸（工具/thinking 不算正文） | 5 分钟 | 终止，报「长时间没有开始输出正文」 |

三个错误都不是「空结果」，**不触发自动重试**（同样的行为重跑只会再撞一次墙）。
`commandSpec.Activity` 让 CLI 的 thinking/status 行也算「还在动」，长推理不会被误杀。

### 刀 3：命令行显式化只读与审批

`--permission-mode read-only --approval-policy never` 写进命令行，不再只靠
`DSH_PERMISSION_MODE` 环境变量。实弹验证：`write` 被 sandbox 拒绝
（`[sandbox: file access denied under read-only mode]`），无人值守子进程不再有「弹卡等人」的
可能。`runMode` 同步改成实际命令行，不再申报一个只有环境变量的说法。

### 刀 4：可核对账目 + 落盘节流 + 提示词纪律

- `parseSeatRunStats` 解析 `session` / `step_end` 事件，得到引擎会话 ID、步数、
  input/cacheRead/output token；`Contribution.Usage` 落库，前端显示
  「缓存命中 96% · 12 步」，hover 显示引擎会话 ID——**这就是「真调用」的凭据**。
- 工具事件落盘节流 1s（终态仍逐次落盘）：高频工具期从「每条一次整份写」
  降到「每秒最多一次」，锁占用与磁盘写放大随之下降。
- 提示词工具段加预算纪律：本轮 ≤8 次工具、禁止构建/测试/安装/全量扫描、
  单条命令要秒级返回、核不动的项写进置信度行而不是继续消耗轮次。

### 前端

`PantheonRecord` 的席位卡与共享时间线在「用时」后追加缓存命中率与步数；
`PantheonSeatUsage` 类型进 `lib/types/pantheon.ts`。未上报的引擎显示空，
不用 0% 冒充。

## 5. 验收

- `go test ./internal/pantheon/` 全绿（含新增 8 例：命令行旗标、账目解析、三条护栏、
  引擎会话续接、预算失败不重试）；
- 实弹 `TestLiveDSHSeatSessionReuse`（`PANTHEON_LIVE_SMOKE=1`）：
  会话 `session-23c1b237-…`，第二轮 **缓存命中 96.7%（6016/6224 prompt token）**；
- `go build ./...`、`go vet ./internal/pantheon/`、`gofmt` 干净；
- 前端 `tsc -b` 0 错，Pantheon 相关 vitest 63/63。

## 6. 未做 / 观察

- 工具预算仍是「明显失控」的硬闸，而不是日常预算：复盘 138 次实弹调用后，工具数与
  耗时几乎无关（≤40 次工具平均 211s，>60 次平均 242s），16 次会误杀 116/138 次正常
  核验，因此实际阈值校准为 120 次；日常 8 次以内由提示词纪律约束，时间由 120s
  无进展闸、5 分钟开局静默闸与 8 分钟总上限收口。
- DSH 的 `--session-id` 续接把历史工具输出也带进上下文，12 轮长会上下文会持续增长；
  DSH 自身有压缩，但「单场 12 轮 DSH 席位的成本曲线」尚未实测。
- codex/zcode 目前不上报 usage，前端显示「未上报」；要同样的缓存证据需要它们的
  等价事件（codex `--json` 有 token 字段，可另开一刀）。
- 工作区切换导致 DSH 凭据缺失时仍会失败一轮（现在错误文案带修复路径）；
  是否能自动回落「Gaea 已配置的 deepseek key」需要产品口径确认。
