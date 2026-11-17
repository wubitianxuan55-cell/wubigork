# 资产 10 · 执行语义施工级：双缓冲、双闸门、终端生命周期

> 源码：`core/src/unified_exec/`（`process_manager.rs` 78 KB / `process.rs` 28 KB /
> `head_tail_buffer.rs` 6.1 KB / `stdin_approval.rs` 12.9 KB / `mod.rs` 8.6 KB）。
> 对照：gaea `internal/gaea/proc/`（`proc.go` 336 B + 各平台 kill/hide/priority）+ `agent/batch_executor.go`。
> 常量证据：`LATE_NETWORK_DENIAL_GRACE_PERIOD=100ms`、`MAX_STDIN_APPROVAL_BYTES=8000`、
> `POST_EXIT_CLOSE_WAIT_CAP=50ms`、`INTERRUPT="\u{3}"`（已在 `hotspot-10` 记录）。

---

## 一、两个缓冲：这是「不重复输出」的关键

```rust
/// Output pending model polling and the retained completion transcript.
/// Append both under the same lock so cancellation cannot split an update.
pub(crate) struct OutputBuffers<const MAX_BYTES: usize> {
    pub(super) pending: HeadTailBuffer<MAX_BYTES>,      // 等模型来取（消费式）
    pub(super) transcript: HeadTailBuffer<MAX_BYTES>,   // 累积完整记录（最终结果用）
}
```

| 缓冲 | 用途 | 语义 |
|---|---|---|
| `pending` | 等下一次轮询取走 | **消费式**：取走即清空，所以 `write_stdin` 续取不会重复 |
| `transcript` | 最终结果 / 回放 | **累积式**：到上限后按头尾保留裁剪 |

两条注释值得抄：
1. **两个缓冲在同一把锁里一起追加**——否则取消时可能只写进一个，出现「模型看到过但最终记录里没有」；
2. **`pending` 与 `transcript` 都要有上限**，不能只截 `pending`（否则长命令的 transcript 会吃满内存）。

## 二、头尾保留 + 省略标记（输出截断的正确姿势）

```rust
/// 保留稳定前缀（head）与后缀（tail），中间超限就丢。
/// 对称：50% 给 head，50% 给 tail。
struct HeadTailBuffer<const MAX_BYTES> {
    head: Vec<u8>, tail: VecDeque<u8>, omitted_bytes: usize,
}
// 省略标记格式：
format!("... {omitted_bytes} bytes omitted ...")
```

**为什么不是简单截断尾部**：命令输出的**头部**常是「命令回显/环境头」，**尾部**是「错误与结果」，
两头都重要；中间往往是重复的进度条。丢中间 + 明确写「省略了多少字节」，模型才不会以为输出完整。

## 三、双闸门：时间先到先返回

```rust
let deadline = start + Duration::from_millis(yield_time_ms);
// collect_output_until_deadline 在 deadline 到点时结束本次收集
// 另有 POST_EXIT_CLOSE_WAIT_CAP = 50ms：进程退出后最多再等 50ms 收尾
```

- `yield_time_ms` 是**本次调用的时间预算**，到点就返回当前增量（不是结束进程）；
- `max_output_tokens` 是**输出量预算**，与时间预算**谁先到谁生效**；
- 进程退出后还有 `POST_EXIT_CLOSE_WAIT_CAP = 50ms` 的上限等待管道关闭，避免「进程已退但管道还被后代持有」时卡死。

`OutputDrainPolicy` 区分两种收尾策略：
- `BoundedAfterExit`：本地执行（后代可能继承管道 → 有界等待）；
- `WaitForOutputClosure`：远端前台输出可能仍在传输 → 等到关闭信号。

## 四、终端生命周期

```rust
pub(crate) struct TerminalSession {
    pub process_id: i32,          // 键
    process: Arc<UnifiedExecProcess>,
    call_id: String,
    pub environment_id: String,
}
pub(crate) struct ProcessStore {
    processes: HashMap<i32, ProcessEntry>,
    reserved_process_ids: HashSet<i32>,   // 测试用确定性 id
}
```

要点：
1. **终端以 `process_id` 为键**，不是 UUID——查表快、日志好读；
2. `collection_finished`（CancellationToken）表示「初次 exec 调用已返回或被取消，可以开始后台 drain」——
   **在此之前不许后台抢读**，否则会出现「前台和后台抢同一段输出」；
3. `chunk_id` 是 **6 位十六进制随机串**，给每段输出打标，便于去重与排障；
4. `max_write_stdin_yield_time_ms` 有下限（`MIN_EMPTY_YIELD_TIME_MS`），防止调用方传 0 导致空转。

## 五、stdin 审批

`MAX_STDIN_APPROVAL_BYTES = 8000`：向交互式命令写 stdin 前，若内容超过 8 KB，
需要走「审批」而不是直接写入（防止把大段内容灌进 `vim`/`python -i` 之类的交互进程）。
`INTERRUPT = "\u{3}"`（Ctrl-C）是中断的统一写法。

---

## 六、gaea 施工方案

gaea 现在 `proc` 只有进程/管道/信号包装，`batch_executor` 是「跑完取结果」。要拿到「边跑边看」需要：

1. **新增 `proc/session.go`**：
   ```go
   type Session struct {
       PID      int
       CallID   string
       EnvID    string
       pending  *HeadTail      // 消费式
       transcript *HeadTail    // 累积式
       done     chan struct{}
   }
   func (s *Session) Drain(maxBytes int) []byte      // 取走 pending
   func (s *Session) Transcript() []byte
   ```
2. **新增 `proc/headtail.go`**：50/50 头尾保留 + `... N bytes omitted ...` 标记（纯函数，易测）；
3. **执行入口加双闸门**：`Run(ctx, argv, yieldMS, maxTokens)` → 到点返回增量 + `running=true`；
   续取走 `Write(stdin, yieldMS)`；
4. **进程退出后的收尾上限**（对应 50 ms）：给一个保守值（例如 100 ms），防止后代持有管道导致永久等待；
5. **`chunk_id` 打标**：6 位十六进制即可，用于日志关联。

**不改的东西**：现有 `kill_windows.go` / `hide_windows.go` / `priority_windows.go` 的平台逻辑保留，
新会话层只是它们的上层。

## 七、验收（可断言）

1. 同一长命令连续三次 `Drain` → 三段内容**互不重叠**且拼接等于 `Transcript()` 前 N 字节；
2. 输出超过缓冲上限 → 只有一个省略标记，且标记里的字节数 = 实际丢弃量；
3. `yieldMS=100` 的调用在 ≈100 ms 返回（允许 ±30%），且进程仍在运行；
4. 进程退出后 100 ms 内必定拿到收尾输出（不再无限等管道）；
5. `Write` 到已退出终端 → 明确错误（不是静默成功）；
6. 交互式命令下写入超 8 KB 的 stdin → 触发审批路径而不是直接写入。

## 八、不建议照搬

- **`OutputDrainPolicy::WaitForOutputClosure`**（远端传输语义）：gaea 没有远端执行器；
- **`reserved_process_ids`（确定性 id）**：那是给测试用的，gaea 用真实 pid + 单调计数器即可；
- **`exec-server` 那条远端执行路径**：整块跳过，本地 PTY 够用。
