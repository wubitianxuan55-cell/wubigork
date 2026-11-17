# 资产 05 · Hooks：14 个事件 + 4 种执行器 + 运行观测

> 源码：`codex-rs/hooks/`（40 个实现文件：`schema.rs` 48 KB / `engine/discovery.rs` 73 KB /
> `engine/dispatcher.rs` 23 KB / `engine/command_runner.rs` 17.6 KB / `engine/output_parser.rs` 21 KB /
> `events/*` 10 个事件模块）与 `core/src/hook_{runtime,environment_executor,mcp_executor}.rs`。
> 对照对象：gaea `internal/gaea/hook/`（4 个实现文件：`hook.go` 16 KB / `runner.go` 13 KB / `trust.go` 2 KB）。
> 二进制侧证据：RPC `hook/started`、`hook/completed`、`HooksList` 均在 `evidence/rpc-ClientRequest.txt` 与
> 方法名清单中出现。

---

## 一、事件面：14 个（gaea 10 个）

```rust
pub enum HookEventName {
    PreToolUse, PermissionRequest, PostToolUse,
    PreCompact, PostCompact,                       // ← gaea 只有 PreCompact
    SessionStart, SessionEnd,
    UserPromptSubmit,
    SubagentStart, SubagentStop,                   // ← gaea 只有 SubagentStop
    Stop,
    Interrupt,                                     // ← gaea 无
    PreModelRequest, PostModelRequest,             // ← gaea 只有 PostLLMCall
}
```

**对照 gaea 现状**：gaea 有 `PermissionRequest / PreToolUse / PostToolUse / UserPromptSubmit / Stop /
PostLLMCall / SessionStart / SessionEnd / SubagentStop / Notification / PreCompact`（11 个），
缺 5 个：`PostCompact`、`SubagentStart`、`Interrupt`、`PreModelRequest`（post 有，pre 无）。

**为什么这 5 个重要**

| 缺的事件 | 能做什么 |
|---|---|
| `PostCompact` | 压缩后重新注入项目约定 / 重置缓存标记（gaea 压缩后没有钩子可用） |
| `SubagentStart` | 给子智能体注入额外纪律（现在只能靠父会话的 prompt 继承） |
| `Interrupt` | 用户中断时落盘现场、停止后台任务（现在是散落在各处的清理逻辑） |
| `PreModelRequest` | 请求前做最后一道「剔除敏感内容 / 补上下文」 |

## 二、执行器：4 种，不是 1 种

```rust
pub enum HookHandlerType { Command, McpTool, Prompt, Agent }
pub enum HookExecutionMode { Sync, Async }
pub enum HookScope { Thread, Turn }
```

| 类型 | 语义 | gaea 现状 |
|---|---|---|
| `Command` | 跑一个可执行文件，读 stdout / 退出码 | ✅ 有（`runner.go` 的 `Spawner`） |
| `McpTool` | 调一个 MCP 工具（`core/hook_mcp_executor.rs`） | ❌ 无 |
| `Prompt` | 直接注入一段提示词片段（不起进程） | ❌ 无 |
| `Agent` | 起一个子智能体做检查（重量级） | ❌ 无 |

`HookExecutionMode::{Sync, Async}`：同步者阻塞当前步骤（可阻断），异步者只旁路记录；
`HookScope::{Thread, Turn}`：决定钩子生命周期是整线程还是单回合。

**`Prompt` 类型是最该先做的**：它不起进程、不依赖外部环境、天然安全，
却能把「每次请求前提醒模型某条纪律」从写死 prompt 变成可配置——gaea 现在这类需求只能改代码。

## 三、运行观测：每次 hook 有结构化结果

`codex_protocol::protocol` 里与 hook 相关的类型：`HookRunStatus`、`HookRunSummary`、
`HookCompletedEvent`、`HookHandlerType`、`HookExecutionMode`、`HookScope`；
对外有 RPC `hook/started` / `hook/completed` 与 `HooksList`（列出已加载钩子）。

**含义**：钩子不是「黑箱脚本」，而是**可观测的一等对象**——每次执行有状态、耗时可查、能被 UI 展示。
gaea 的 hook 目前只有 `Notify` 回调，没有可回放的结构化记录。

## 四、安全：插件钩子要验指纹

`engine/plugin_trust.rs` 4.3 KB + `engine/plugin_fingerprint.rs` 3.4 KB：
**来自插件的钩子必须通过信任校验与指纹比对**才会执行（防止插件更新后偷偷改钩子）。

gaea 的 `trust.go` 只有项目管理器级别的 `IsTrusted(projectRoot, homeDir)`，
**没有「这条钩子来自哪个插件、插件指纹是否变过」这一层**——一旦将来支持插件带钩子，这就是缺口。

## 五、输出处理：解析 + 溢出

- `engine/output_parser.rs` 21 KB：把命令输出解析成**结构化决策**（允许/拒绝/询问 + 注入内容），
  而不是只看退出码；
- `output_spill.rs` 4.7 KB：钩子输出过大时**落盘并给模型一个引用**，不把几 MB 塞进上下文。

## 六、兼容性（说明为什么它这么大）

`engine/mod.rs` 里引擎叫 `ClaudeHooksEngine`，配置读取走 `engine/discovery.rs`（73 KB）——
**codex 的 hooks 配置格式与 Claude Code 兼容**（`.claude/settings.json` 那套 schema）。
这是它有 48 KB schema 文件的原因。gaea 不需要背这个包袱。

---

## 七、给 gaea 的三档落地

**A 档（1–2 天，纯增量）**
1. 事件表补到 14 个（至少补 `PostCompact`、`SubagentStart`、`Interrupt`、`PreModelRequest`）；
2. 把「事件 → 默认超时 / 是否阻断 / 执行器种类」从 switch 改成**一张数据表**驱动
   （gaea 现在 `defaultTimeout` 与 `IsBlocking` 是两处 switch，加事件容易漏）；
3. 实现 `Prompt` 执行器（无进程、无 IO，最安全）。

**B 档（3–5 天）**
4. 加 `McpTool` 执行器：复用 `internal/gaea/plugin` 的 MCP 客户端（已有 stdio/http 双传输）；
5. 加 `Sync/Async` 分流：同步者参与阻断判定，异步者只记录不阻塞。

**C 档（按需，先验证需求）**
6. `Agent` 执行器（起子智能体做检查）——成本高，只在「需要独立视角复核」的场景做；
7. 插件指纹信任——等插件真能带钩子时再做。

## 八、验收（A 档即可断言）

- 新增 `PostCompact` 钩子在压缩后触发一次，且能拿到压缩前后的 token 数；
- 事件表与 `IsBlocking`/`defaultTimeout` 同源（加事件只需改一处，单测断言两表键集一致）；
- `Prompt` 执行器的输出出现在下一轮上下文里，且**不产生进程**（用 `os.Getpid` 或进程计数断言）；
- 钩子执行有结构化记录（事件名、耗时、结果、执行器类型），可从事件流里回放。

## 九、不建议照搬

- **Claude Code 配置兼容层**（48 KB schema + 73 KB discovery）：gaea 有自己的设置格式，兼容别家 schema 是纯负担；
- **`legacy_notify.rs`**：历史兼容代码，没有对应场景；
- **`Agent` 执行器**：在单用户桌面应用里，「起子智能体来检查一次工具调用」的性价比通常低于直接加规则。
