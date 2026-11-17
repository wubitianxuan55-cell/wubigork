# asset-47 · 回合准入与优雅关闭（B1 · 施工级）

> 证据：`codex-rs/app-server/src/{turn_admission.rs, connection_rpc_gate.rs, connection_cleanup.rs,
> request_serialization.rs, thread_status.rs, message_processor.rs, error_code.rs, lib.rs}`（`rust-v0.165.0`）。
> gaea 侧只读对照：`internal/gaea/control/controller.go`、`controller_approval.go`、`internal/app/app.go`。
> 本文只出机制与施工卡，不含代码改动。

---

## 1. 一句话结论

codex 把"**还能不能接新活**"做成了一个**显式状态**：关闸（drain）与准入（admit）共用一把锁，
关闭条件是"在跑回合 + 已准入请求"**两个计数同时归零**；连接维度另有闸门（关闸后排队的不再启动、
已启动的跑完，且**换账号等于连接关闭**）。
gaea 现在是"**单回合 + 全局队列 + 顺序 teardown**"：没有 drain 态、没有两计数收敛、
没有连接维度的作废、也没有 per-thread 活跃标记广播。差距集中在**关闭窗口期的语义**。

---

## 2. 机制地图

| 关注点 | 文件 | 关键行 |
|---|---|---|
| 准入令牌 | `turn_admission.rs` | 18–31 结构；35–40 `begin_drain`；48–63 `admit/try_admit`；66–71 `TurnStartAdmission`；73–83 `Drop` |
| 拒绝理由 | `error_code.rs` | 11–15 `server_draining_error`（`-32600` + `data.reason="serverShuttingDown"`） |
| 关闭判定 | `lib.rs` | 263–284 `on_signal`；286–317 `update`（两计数收敛 + 日志去重）；1041–1044 订阅三个计数 |
| 连接闸门 | `connection_rpc_gate.rs` | 16–21 结构；32–46 `run`；48–50 `is_closed`；52–61 `close/shutdown` |
| 连接收尾 | `message_processor.rs` / `lib.rs` | 111 `CONNECTION_RPC_DRAIN_TIMEOUT=30s`；883–909 `connection_closed`；1196–1220 `ConnectionClosed` 处理；1221–1228 关闸后丢消息 |
| 每资源串行化 | `request_serialization.rs` | 25–110 队列键与 `SharedRead/Exclusive`；112–142 排队项；162–177 `discard_closed`；205–231 入队与起 drain 协程；233–307 drain 循环 |
| 状态广播 | `thread_status.rs` | 19–30 结构；47–58 guard 的 `Drop`；133–145 运行中回合数 + watch；147–221 四类注记；223–249 发布与去重；258–268 guard 释放；378–420 watcher 回收；447–469 状态派生；290–304 抢跑补偿 |

---

## 3. 机制（按代码顺序）

### 3.1 准入令牌：一个结构，两种用法

```rust
struct AdmissionState { closed: bool, active: usize }
// admit(): 同锁内 closed 检查 + active += 1 + watch 广播
// TurnPermit 的 Drop: active -= 1 + watch 广播
```

三个不那么显然的点：

1. **`begin_drain()` 与 `admit()` 取同一把 `Mutex`**——注释写明原因：*"The permit keeps shutdown from
   finishing while an earlier request is still preparing or submitting work."*
   即"已准入但还没真正开始跑"的请求**也算在飞**，否则关闭会在准备阶段把它截断。
2. **广播是 `watch::Sender<usize>`，不是轮询**：`send_replace(active)` 每次准入/释放都推一次，
   关闭判定侧订阅这个 channel 即可，不需要定时器。
3. **同一结构另有 `TurnStartAdmission` trait 实现**（返回 `Box<dyn Send>` 的 permit），
   给 core/扩展侧用——**准入点不止 app-server 一处**，扩展（如 skills 服务）也能参与"现在能不能起回合"的判断。

### 3.2 关闭判定：两个计数同时归零

```rust
// lib.rs:296
if self.forced || (running_turn_count == 0 && active_admissions == 0) { Finish }
```

- `on_signal` 第一次：`begin_drain()` + 记 `requested`；**第二次**（`Forceable`）才置 `forced` 强退。
- `update` 只有在计数**变化**时才打日志（`last_logged_running_turn_count`），避免每轮事件循环刷屏。
- 注意 `Finish` 的条件里 **`active_admissions` 单独存在**：即使当前没有回合在跑，
  只要还有"已准入未完成"的请求（比如正在恢复线程），也不许退出。

### 3.3 连接闸门：三态合并成一个 `is_closed()`

```rust
is_closed() = tasks.is_closed() || auth.as_ref().is_some_and(|auth| !auth.is_current())
```

- `close()`：只关"接受"（`accepting=false` + `tasks.close()`），**已拿 token 的 future 继续跑**。
- `shutdown()`：`close()` 再 `tasks.wait()`——真正的收尾等待。
- `run(future)`：先在同一把锁下检查 `accepting` 与 `is_closed()`，再拿 `TaskTracker` token，
  所以"检查—登记"之间没有窗口能让关闭漏过去。
- **换账号 = 连接关闭**：auth 所有权变化（`is_current()` 为假）直接让 `is_closed()` 成立，
  旧连接排队的请求不会在新身份下被执行。
- 连接断开时的顺序（`lib.rs:1196`）：`rpc_gate.close()` → 通知出站路由 → **spawn** `connection_closed()`
  进 `ConnectionCleanupTasks`（`JoinSet`）→ 单客户端模式下 stdio 关闭直接 `break`。
- `connection_closed()` 里 `rpc_gate.shutdown()` 包在 **30s** 超时里，超时只 `warn` 不阻塞退出。
- 入站消息在闸门关闭后**静默丢弃**（`continue`），不是回错误——因为对端已经不算合法连接了，
  回错也送不出去。

### 3.4 每资源串行化队列：Exclusive / SharedRead

- 队列键是**资源**（线程 id、MCP OAuth server 名等），不是连接——跨连接的同类写操作也互斥。
- 每个键一条 FIFO（`VecDeque` + `Notify`），第一次入队时 `spawn` 一个 `drain` 协程，队列空即退出并删键。
- `Exclusive`：逐个 `await`；`SharedRead`：整批塞进 `FuturesUnordered` 并发跑，
  且运行期间**新到的 SharedRead 会被吸收进来**（`changed.notified()` 分支）。
- 排队项自身不带"连接"字段，但**透过 `gate` 判断**：`discard_closed()` 把 gate 已关闭的排队项直接扔掉并唤醒队列。
- `enqueue_background()` 让 app 自己的后台工作（无连接）也进同一条队列，和 RPC 抢同一把资源锁。

### 3.5 状态广播：per-thread 活跃标记

状态是**派生**的，不是被写入的：

```rust
NotLoaded                      // is_loaded == false
Active { active_flags }        // running 或 有挂起审批/追问
SystemError                    // has_system_error
Idle                           // 其余
```
`active_flags` 只有两枚：`WaitingOnApproval`、`WaitingOnUserInput`，由两个 `u32` 计数驱动。

四条工程细节：

1. **guard 释放不能 await**：`ThreadWatchActiveGuard::drop` 里用 `handle.spawn(...)` 把
   "计数减一 + 发布" 挪出 drop 体（drop 里不能 await），并 `saturating_sub` 防下溢。
2. **发布去重**：`running_turn_count_tx.send_if_modified(|cur| ...)` 与 per-thread
   `sender.send_if_modified(|cur| ...)` 都是"值没变就不通知"——避免状态风暴。
3. **watcher 回收**：`sender.receiver_count() == 0` 时从 map 里删掉——没人看的状态不占内存。
4. **抢跑补偿**：`resolve_thread_status(status, has_in_progress_turn)`——真实回合事件可能先于
   watch 状态被观察循环读到，此时把 `Idle/NotLoaded` 提升为 `Active{[]}`，
   宁可报"在跑"也不报"空闲"（避免前端把正在跑的线程画成可编辑）。

### 3.6 准入分类：runtime change 与 turn work

`message_processor.rs:1006` 把请求分成两类：

| 类别 | 请求 | 准入时机 |
|---|---|---|
| runtime change | `ThreadStart/Fork/Resume/Revert/SettingsUpdate`、`TurnSettingsUpdate`、`ThreadDelete/Archive` | 收到时准入一次，**不复核** |
| turn work | `ThreadSample`、`TurnStart/Steer`、`ReviewStart`、`ThreadCompactStart`、`ThreadShellCommand`、`ThreadQueueStart`、`ThreadRealtimeStart` | 收到时准入 **+** 串行化队列里**再准入一次** |

二次准入的注释说明了原因：*"Runtime changes already admitted before drain finish normally.
Turn work still waiting in serialization must observe the newly closed gate."*
——即**排队越久的活越要重新看闸**，因为它在队列里待的这段时间，服务器可能已经开始 drain。

另外 `restore_daemon_threads`（`:798`）在 `admit()` 失败时直接 **`break`**：drain 期间不再继续恢复线程，
避免"越恢复越多在飞"。

---

## 4. 常量与上限（全部为生产值）

| 名称 | 值 | 位置 | 语义 |
|---|---|---|---|
| `CONNECTION_RPC_DRAIN_TIMEOUT` | **30s** | `message_processor.rs:111` | 单连接收尾等待上限，超时只 warn |
| 拒绝码 | `-32600` + `data.reason="serverShuttingDown"` | `error_code.rs:11` | 结构化 drain 拒绝，客户端可据此重连 |
| 强退条件 | 第二次 `Forceable` 信号 | `lib.rs:271` | 第二次信号不等计数归零 |
| JSON-RPC 其它码 | `-32600/-32601/-32602/-32603/-32001` | `error_code.rs:3–7` | `-32001` 为过载 |
| 状态 flags | `WaitingOnApproval` / `WaitingOnUserInput` | `thread_status.rs:452–458` | 计数 > 0 才出现 |

---

## 5. 状态机（回合 × 关闸 × 连接）

| 回合状态 | 未关闸 | 已关闸（drain 中） | 强制关闸 |
|---|---|---|---|
| 未准入 | 可准入（`active+1`） | 拒绝：`-32600 serverShuttingDown` | 拒绝 |
| 已准入未开始 | 正常跑 | **跑完**（permit 保持计数） | 随进程终止 |
| 在跑 | 正常 | 跑完，计数归零后允许退出 | 取消/终止 |
| 排队中（串行化） | 轮到再准入 | **二次准入失败 → 回错误，不执行** | 同上 |
| 连接已关闭的排队项 | `discard_closed` 丢弃 | 同左 | 同左 |

> 关键不变量：**`active_admissions` 只降不升（关闸后）**；`running_turn_count` 由 thread 状态派生，
> 两者都为 0 才 `Finish`。

---

## 6. 失败路径

1. **drain 期间收到 turn work**：返回 `-32600` + `reason`，客户端按"重连后再试"处理（文案里写明）。
2. **二次准入失败**：走的是同一错误，但**已经占了一个 permit**——`turn_admission` 变量被 `_turn_admission`
   持有到请求结束，所以不会提前把计数降掉。
3. **连接收尾超时**：30s 后 warn 并继续（不阻塞其它连接、不阻塞进程退出）。
4. **guard 释放竞态**：drop 里 spawn，若运行时已关闭则 spawn 失败——计数不会归零，
   但此时进程已在退出路径上，属于可接受残留（代码未特殊处理，属已知边界）。
5. **恢复线程中途关闸**：`admit()` 失败即 `break`，剩余线程不恢复（下一轮启动由 daemon 恢复快照负责）。

---

## 7. gaea 对照（只读实测）

| 维度 | codex | gaea 现状 | 差在哪 |
|---|---|---|---|
| 准入 | `admit()`/`begin_drain()` 同锁 + `active` 计数 | `tryStartTurn`（`controller.go:415–423`）只在 `c.mu` 下判 `c.running` | 只有"有没有回合在跑"，**没有 drain 态**，也没有"已准入未完成"计数 |
| 关闭 | 两计数归零 + 二次信号强退 + 30s 连接超时 | `App.Shutdown`（`app.go:686–740`）顺序 teardown；`Controller.Close`（`controller.go:1149–1162`）取消 jobs + cleanup | 无等待在飞回合、无超时、无强退语义；`Close` 后没有"拒收"标志 |
| 队列 | 按资源键 + `Exclusive/SharedRead` + 丢弃已断连接项 | `pendingSends` 全局 FIFO，上限 8（`controller.go:224/524–536`） | 无按资源互斥、无共享读批量、无"连接关闭即作废" |
| 状态 | per-thread 派生状态 + 两枚 active flag + watch 去重 + 抢跑补偿 | 事件流 `TurnDone`/`Notice` + `jobs.View`；审批用 `c.approvals` map + `promptMu`（`controller_approval.go:340–372`） | 无"等待审批/等待追问"作为**可查询状态**；无订阅者回收；无抢跑补偿 |

**结论**：gaea 的"单回合 + 队列"在单用户桌面场景下够用，真正缺的是**关闭窗口期**的三件事：
① 关闸后拒收（而不是继续接活）；② 等"在跑 + 已准入"归零再退；③ 连接/窗口断开的排队项作废。

---

## 8. 施工卡（三刀，Go 侧；默认关闭）

> 纪律：机制照抄、文案自写；新开关默认关闭，关时行为与现状逐字节等价；每刀先记基线测试数字。

### 刀 1 · 准入与 drain（约半天）

- 改动点：`internal/gaea/control/controller.go`（新增 `admit()/beginDrain()`、`draining`、`admitted`），
  `internal/app/app.go`（`Shutdown` 改为先 `BeginDrain` 再 teardown）。
- 语义：`Send/Submit` 先 `admit()`；失败回结构化错误（`code=draining`），**不静默丢**。
- 关闭：`BeginDrain(ctx, timeout)` → 等 `!running && admitted==0`；超时→强制 `cancel()` 并落一条 `TurnDone{Err}`。
- 禁区：不动 `pendingSends` 的既有排队语义；不动 hook 的 `SessionEnd` 触发时机。

### 刀 2 · 连接/窗口闸门（约半天）

- 改动点：绑定层为每个前端入口（桌面窗口 / HTTP-SSE 会话）维护 `gate`；`pendingSends` 项带上 gate 引用。
- 语义：连接关闭 → 丢弃该连接的排队项并唤醒队列；单窗口模式关闭即触发刀 1 的 drain。
- 禁区：多入口共享同一 `Controller` 时，**不得**让一个入口的关闭影响另一个入口的排队项（断言 A4）。

### 刀 3 · 状态广播（约半天）

- 改动点：为会话维护 `activeFlags`（`WaitingOnApproval` 由 `len(c.approvals)`、`WaitingOnUserInput` 由 `len(c.asks)` 驱动），
  暴露 `Status()` + 订阅通道；发布去重；无订阅者即回收。
- 抢跑补偿：`TurnStarted` 事件先到时，状态读取按"有在跑回合"处理。
- 禁区：不改变现有 `event.Event` 的类型与顺序（前端已依赖）。

---

## 9. 验收断言（Go 单测，逐条可执行）

| # | 断言 |
|---|---|
| A1 | `BeginDrain` 后调用 `Send`：返回 draining 错误，**不启动回合**（`running` 保持 false，无 `TurnStarted`） |
| A2 | 关闸前已 `admit` 的回合：用 channel 卡住回合体，`BeginDrain` 必须阻塞；放行后返回且只返回一次 |
| A3 | 关闸后超时：`BeginDrain(ctx, 50ms)` 返回超时错误，回合被取消，落一条 `TurnDone{Err}` |
| A4 | 连接 A 关闭：A 的排队项被丢弃；连接 B 的排队项照常执行（不得串扰） |
| A5 | 审批挂起期间 `Status()` 含 `WaitingOnApproval`；审批结束后回落（计数归零）；两次相同状态只发一次通知 |
| A6 | 二次强退：drain 中再调 `ForceDrain()`，跳出等待并取消在飞回合 |
| A7 | 无订阅者时状态 watcher 被回收（引用计数为 0 后 map 不再增长） |
| A8 | 准入门禁关闭（默认）：`Send` 行为与改动前**逐字节等价**（事件序列快照对比） |

---

## 10. 不搬 / 只借语义

- **不搬**：`TaskTracker`/`watch` 的 tokio 具体 API（Go 用 `sync.WaitGroup` + `chan struct{}` 表达同一语义）。
- **不借**：多连接鉴权所有权（`ConnectionAuth::is_current`）——gaea 单用户本地，只借"换身份即作废旧队列"的**原则**，
  落到"切换会话/工作区时作废旧队列"。
- **不借**：`enqueue_background`（gaea 后台任务是 jobs 体系，已有自己的完成注记队列）。

---

## 11. 风险

| # | 风险 | 缓解 |
|---|---|---|
| 1 | drain 等待把关闭拖长（用户点了关闭却不退） | 默认超时 3–5s，超时即强退并落日志；与 codex 的 30s 不同（那是连接级、非进程级） |
| 2 | `admitted` 计数泄漏导致永不 Finish | permit 用 defer 释放；A2/A3 覆盖异常路径 |
| 3 | 状态广播与前端现有事件序列打架 | 新增通道只加不改（A8 快照对比） |
| 4 | 多入口共享 Controller 时误伤 | A4 断言 + gate 只在"入口维度"生效 |
