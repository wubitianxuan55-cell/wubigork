# 资产 21 · 执行服务（exec-server）：能力发现与网络决策协议

> 源码：`exec-server/src/`（顶层单文件最大 `client.rs` 145 KB / `local_process.rs` 85 KB / `environment.rs` 81 KB /
> `remote.rs` 74 KB / `relay.rs` 68 KB / `rpc.rs` 51 KB / `local_file_system.rs` 49 KB / `fs_sandbox.rs` 41 KB）
> 与 `exec-server-protocol/src/{protocol.rs 85 KB, capabilities.rs, network_policy.rs}`。
> 二进制侧：本机 `codex.exe` 旁带 `codex-command-runner.exe`（8.7 MB）与 `codex-windows-sandbox-setup.exe`。

---

## 一、它是什么

把「执行」从 agent 进程里拆出去的**独立执行服务**：agent 通过 RPC 让 exec-server 去跑命令、读写文件、
做网络策略判定，exec-server 自己可以跑在本地子进程、也可以跑在远端（`remote.rs` / `relay.rs` / 噪声加密中继）。

**结论：整块不做**（独立进程 + RPC + 远端中继约 500 KB），但它的两个**协议设计**值得抄。

## 二、可借之一：网络决策三态 + 审计通知

```rust
pub const NETWORK_POLICY_REQUEST_METHOD:  &str = "network/policyRequest";
pub const NETWORK_POLICY_DECISION_METHOD: &str = "network/policyDecision";
pub const MAX_NETWORK_POLICY_HOST_BYTES:      usize = 253;   // DNS 名上限
pub const MAX_NETWORK_POLICY_PROCESS_ID_BYTES: usize = 256;
pub const MAX_NETWORK_POLICY_REASON_BYTES:     usize = 1024;

pub enum ExecServerNetworkPolicyDecision {
    Allow,
    Deny { reason: String },
    Ask   { reason: String },       // ★ 三态：除了放行/拒绝，还能「问用户」
}

pub struct NetworkPolicyDecisionNotification {
    process_id, timestamp, scope, decision, source, reason,
    protocol, host, port, method?, client?, policy_override: bool,
}
```

**三个要点**：

1. **`Ask` 是独立状态**——出网不是「允许/拒绝」二选一，还能「这条要问用户」。
   gaea 现在的出网策略若只有放行/拦截，遇到「用户明确要下载这个域名」就只能改配置。
2. **决策必带审计通知**：`source`（谁拍的板）、`scope`（作用域）、`policy_override`（是否被人工覆盖）、
   `method`/`client`（具体请求与方法）。**决策与审计是一体两面，不是两件事。**
3. **限额按真实约束取**：host 253 字节正好是 DNS 域名上限；reason 1024 字节防止解释文本撑爆日志。

## 三、可借之二：能力发现快照（一步内复用，不重复扫盘）

```rust
pub const CAPABILITIES_DISCOVER_V2_METHOD: &str = "capabilities/discoverV2";

pub struct ExecutorCapabilityDiscoveryV2Snapshot {
    environment_id: String,
    result: Result<Arc<DiscoverV2CapabilitiesResponse>, String>,
}
impl ExecutorCapabilityDiscoveryV2Snapshot {
    /// Compares discovery identity, not inventory contents
    pub fn same_discovery(&self, other: &Self) -> bool {
        self.environment_id == other.environment_id
            && match (&self.result, &other.result) {
                (Ok(p), Ok(c)) => Arc::ptr_eq(p, c),      // ★ 指针相等即「同一次发现」
                (Err(p), Err(c)) => p == c,
                _ => false,
            }
    }
}
```

**为什么用指针比较**：能力发现要扫插件目录、解析 manifest、收集技能——很贵。
一次模型步骤里可能有多次需要「当前能力清单」，用 `Arc::ptr_eq` 判断「是不是同一份快照」，
命中就直接复用，不比较内容（内容比较本身就很贵）。

返回结构也说明职责边界：`{plugins[{id: "<name>@<marketplace>", display_name, version, root, files, skills[]}],
skills[](独立技能), warnings[]}` —— **发现阶段只报事实，策略（谁能用）由调用方施加**。

## 四、可借之三：Debug 实现刻意省略敏感内容

```rust
impl fmt::Debug for ExecutorSkill {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        // Omit potentially sensitive descriptions and file contents.
        f.debug_struct("ExecutorSkill").field("name", &self.name).field("path", &self.path).finish()
    }
}
```

日志/调试输出默认只留 name 与 path，**技能描述与文件内容不进日志**。
gaea 的日志里如果会打印工具参数或文件片段，这条值得照做：给敏感结构体写自定义 `String()`，
而不是直接 `%+v`。

---

## 五、不做清单

| 不做 | 体量 | 理由 |
|---|---|---|
| 独立执行服务进程（`local_process.rs` / `environment.rs`） | ~166 KB | gaea 进程内执行，`proc` 包已就位 |
| RPC 层（`rpc.rs` / `client.rs` / `connection.rs`） | ~240 KB | 无进程边界，无需 RPC |
| 远端执行（`remote.rs` / `relay.rs` / `noise_relay/`） | ~140 KB | 无远端执行器 |
| `fs_sandbox.rs` / `sandbox_integrity/` | ~50 KB | gaea 用自己的 `sandbox` 包（seatbelt/bwrap/WSL2） |

## 六、gaea 落地建议（两小改）

1. `internal/netclient`：把出网决策从「允许/拒绝」扩成 **`Allow / Deny{reason} / Ask{reason}`**，
   并让每次决策产出一条审计记录（`source` / `scope` / `override` / `method` / `host`）；
   限额照抄（host 253 / reason 1024）。
2. `internal/gaea/tool`：能力发现（技能 + 插件 + 工具清单）加**一步内快照**，
   同一步内多次取用直接复用（用指针/版本号判断，不比较内容）。

## 七、验收

1. 同一步内连续三次取能力清单 → 只做一次实际扫描（用计数器断言）；
2. 出网被拒 → 有一条审计记录，含 reason 与 source；被人工覆盖 → `policy_override=true`；
3. 出网命中「需要询问」→ 返回 `Ask` 并触发审批，而不是直接拦截；
4. 日志里不出现技能描述或文件内容（用敏感样例字符串 grep 日志）。

## 八、一句话

exec-server 的**进程拆分**对 gaea 无用，但它的**网络决策三态 + 审计一体**与**能力发现快照**是两块钱都能用的协议设计。
