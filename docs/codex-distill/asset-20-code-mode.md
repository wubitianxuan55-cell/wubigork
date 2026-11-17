# 资产 20 · code-mode（V8 脚本编排）：判定不做，但记下沙箱硬化清单

> 源码：`code-mode/`、`code-mode-protocol/`、`code-mode-host/`、`code-mode-runtime/`
> 与 `core/src/tools/code_mode/{mod.rs, execute_handler.rs, wait_handler.rs, delegate.rs}`。
> 二进制侧：本机 `codex.exe` 旁就带 `codex-code-mode-host.exe`（**78 MB**）。

---

## 一、它解决什么问题

模型不直接发工具调用，而是写一段 **JS「cell」**，在 V8 isolate 里执行；
JS 里可以调用一组宿主回调（`tool` / `notify` / `store` / `load` / `set_timeout` / `image` /
`generated_image` / `audio` / `exit` / `yield_control`），**一次工具调用因此能编排成百次工具调用**。

对外工具只有两个：`code_mode`（执行 cell）与 `code_mode_wait`（等长任务）；
内部有 cell id、会话注册表、gRPC 传输、重连、截止时间。

## 二、为什么 gaea 不做

| 维度 | codex 依赖 | gaea 现实 |
|---|---|---|
| 运行时 | 内嵌 V8 isolate + 独立宿主进程（78 MB 的 `codex-code-mode-host.exe`） | Go 侧要么上 goja（纯 Go 但性能/兼容有限），要么再起一个进程 |
| 传输 | gRPC + 会话注册 + 重连 | 进程内调用，没有这一层 |
| 前提条件 | **模型能写出正确的 JS** | gaea 以本地小模型为主，长脚本正确率低，失败成本高（一个语法错误整段白跑） |
| 已有的替代 | — | `agent/batch_executor.go` 已能做工具批处理与并发 |

**结论：能力面不匹配，投入产出不成立。**

## 三、三条值得借的工程实践（与 V8 无关）

### 3.1 沙箱硬化：默认删除危险全局，只注入白名单

```rust
// code-mode-runtime/src/globals.rs
delete_global(scope, global, "console")?;
delete_global(scope, global, "Atomics")?;
delete_global(scope, global, "SharedArrayBuffer")?;
delete_global(scope, global, "WebAssembly")?;
// 然后只安装白名单回调
install: tool / notify / store / load / set_timeout / image / generated_image / audio / exit / yield_control
```

**这条是通用安全纪律**：任何「执行用户/模型提供的代码」的场景，默认**删掉**能力（不只是不暴露），
再按需注入白名单。`SharedArrayBuffer`（共享内存）、`Atomics`（底层并发）、`WebAssembly`（可执行任意机器码）
是典型的「一旦留下就等于绕过所有上层限制」。

gaea 若将来做「表达式钩子」或「用户脚本」，第一件事就该是这张删除清单。

### 3.2 每次调用独立超时 + 取消优先

```rust
const TRANSPORT_TIMEOUT: Duration = Duration::from_secs(60);
// 每请求另有 runtime_timeout
tokio::select! {
    biased;
    _ = session.stopped.cancelled() => return Err("session closed"),
    result = enforce(operation, runtime_timeout, request) => result,
}
```

- **`biased` + 取消分支在前**：会话被停时立刻返回，不等当前请求跑完；
- 超时分两层：传输层 60 s 兜底，业务层 `runtime_timeout` 按 cell 类型给不同上限。

### 3.3 错误信息有硬上限

```rust
const MAX_ERROR_BYTES: usize = 512;
```

脚本执行失败时的错误文本截断到 512 字节——防止一个巨大的堆栈把上下文撑爆，
同时保证「错误仍可读」。这条与 asset-11 的 `... N bytes omitted ...` 是同一个纪律。

## 四、什么情况下才值得重新评估

三个条件**同时**满足时再考虑：

1. gaea 的主力模型能稳定写出 30 行以上的正确脚本（可通过评测集验证）；
2. 出现「一次请求里要串几十次工具调用」的真实场景（例如批量整理上百个文档）；
3. 已有 `batch_executor` 的并发方案被证明不够（例如需要在中间结果上做条件分支）。

届时的最小形态不是 V8，而是 **Go 侧受限表达式**（如 goja + §3.1 的删除清单），
并且必须保留 §3.2 / §3.3 两条。

## 五、不做清单

| 不做 | 理由 |
|---|---|
| `code-mode-host`（独立进程 + gRPC） | 为进程隔离与远端会话设计，gaea 进程内即可 |
| `code-mode-protocol`（JSON schema 43 KB + payload 17 KB） | 协议层，无对应场景 |
| cell 会话注册 / 重连（`session_registry` / `reconnect`） | 与 gRPC 生命周期绑定 |
| V8 绑定（`v8_init` / `module_loader` / `timers`） | 换成 goja 或直接不做 |

## 六、一句话

code-mode 是「用代码代替往返」的思路，架构代价很大（78 MB 宿主进程 + gRPC + V8）；
对 gaea 现在不做，但那张**危险全局删除清单**和**取消优先 + 错误 512 字节上限**是两块钱都能用的经验。
