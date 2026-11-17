# 资产 03 · 记忆管线：两阶段 + 背压门 + 基线差分

> 源码：`codex-rs/memories/{read,write}/`（write 侧 20 个文件：phase1 28.7 KB / phase2 23.7 KB /
> rollout_input 15.2 KB / runtime 16 KB / storage 8.5 KB / workspace 8.6 KB / guard / workspace_lock / extensions/prune）
> 与 `codex-rs/ext/memories/`。
> 对照对象：gaea `internal/gaea/memory/`（60+ 文件：store/sqlite/file_backend/decay/activation/graph/citations/distill/morning/recall/search/injection_eval）。

---

## 一、codex 的形态：把「记忆整理」当成离线批处理任务

入口 `start.rs::start_memories_startup_task` —— **异步启动任务**，且有三道前置门：

1. **会话类型门**：临时会话（ephemeral）、子智能体会话**不跑**；只有「有资格的根会话」才跑；
2. **特性门**：feature flag 关着就不跑；
3. **背压门**（`guard.rs`）：`rate_limits_ok()` —— 后端限流时不跑。理由是「限额已经紧张，别把配额花在整理记忆上」。

### 1.1 阶段一：抽取（phase1）

- 输入由 `rollout_input.rs` 从 rollout 文件**反向扫描**（`ReverseJsonlScanner`）取最近证据，
  再**按来源（provenance）重排成时间序**；
- 人发的消息**保留问题原文**，图片/音频等媒体**替换成占位符**（不把二进制塞进记忆）；
- 分块时**保持证据顺序**，每块受同一套 token 预算约束；
- 输出契约 `StageOneOutput { raw_memory, rollout_summary, rollout_slug }`，
  反序列化用 `#[serde(deny_unknown_fields)]`（版本化契约，v2 明确「不解码 raw memory」）；
- 写入前先过 **`redact_secrets`** 脱敏 + `TruncationPolicy` 截断。

### 1.2 阶段二：合并（phase2）

固定顺序，每一步都有对应防护：

```
MemoryWorkspaceLock::try_acquire()          // 跨进程独占，拿不到就 WouldBlock 放弃
prune_old_extension_resources()             // 扩展资源按 RETENTION_DAYS 清理
prepare_memory_workspace()                  // 准备基线
build_consolidation_prompt_for_version()    // 版本化合并提示词
spawn 合并 agent → validate_consolidation_artifacts_for_version()
memory_workspace_diff()                     // 基线 → 现在的差分
write_workspace_diff()                      // 差分落盘（可审）
sync_rollout_summaries_from_memories()
rebuild_raw_memories_file_from_memories()
```

### 1.3 锁的关键细节（`workspace_lock.rs` 注释原文意思）

锁文件打开在 **memory root 之外**（`memory_root.with_extension("lock")`），
原因写得很直白：**基线重置/记忆重置会替换 workspace 里的 inode，若锁在 workspace 内，会被连根替换掉，
另一个进程仍以为持有锁**。`try_acquire` 返回 `WouldBlock` 时**放弃本次整理**，不排队、不阻塞会话。

---

## 二、与 gaea 对照（先说实话：gaea 有更成熟的部分）

| 维度 | codex | gaea 现状 | 判断 |
|---|---|---|---|
| 记忆形态 | 文件 + workspace 基线/差分 | SQLite + file_backend + eventlog + 图存储 | **gaea 更强**（结构化、可图查询） |
| 衰减/生命周期 | 未见（依赖重写覆盖） | `decay.go::DecayScore` + `LifecycleOf` | **gaea 更强** |
| 引用与使用反馈 | `read/citations.rs`、`read/usage.rs` | `citations.go` + `injection_eval.go` | **gaea 更强**（有注入效果评估） |
| 整理触发 | 启动时异步、三道门 | `morning.go`（确定性晨报，零 LLM）+ `distill.go`（合并候选） | 两种哲学，可并存 |
| **抽取前脱敏** | `redact_secrets` + 截断策略 | 无（0 处命中） | **gaea 缺** |
| **背压门** | 限流即跳过整理 | 无（0 处命中） | **gaea 缺** |
| **合并写入的基线差分** | baseline + diff + 产物校验 | 无（0 处命中） | **gaea 缺** |
| **跨进程锁的 inode 语义** | 锁文件放在 root 之外，reset 不影响 | 有锁，但未见「放 root 之外」的理由 | 需核对 |
| 扩展资源保留期 | `RETENTION_DAYS` 定期清理 | 无对应概念 | 可选 |

---

## 三、值得搬的四条（按性价比）

### M1 抽取前脱敏（最该先做）

- **做法**：任何「写进记忆 / 写进长期事实」的路径，落库前先过一遍密钥形态识别
  （`sk-…`、`ghp_…`、`AKIA…`、`-----BEGIN … PRIVATE KEY-----`、`eyJ…` JWT、`postgres://user:pass@`），
  命中即替换成 `<redacted:kind>`，并**保留命中计数**（用于审计，不保留原文）。
- **落点**：`memory/distill.go` 与 `memory/remember.go` 的统一入口（若有多个写入口，抽一个 `sanitize()`）。
- **验收**：构造 12 条真实形态的假密钥 → 写库后按原文 grep 全表 **零命中**；计数与类型分布可查；
  普通长文本（小说段落、代码块）不被误伤（准备 5 条含 `key`/`token` 字样但不含密钥的文本）。

### M2 背压门（防「半夜整理记忆把配额烧光」）

- **做法**：整理任务启动前查一次「当前是否处于限流/配额告警」；是则跳过本次并记一条事件
  （`skipped: rate_limited`），不重排、不阻塞。
- **落点**：`memory/morning.go` / `distill.go` 的调度入口。
- **验收**：模拟限流 → 整理任务不执行且事件可查；限流解除后下一次调度正常执行。

### M3 合并写入的基线差分（可审）

- **做法**：整理前把当前记忆集合的快照当基线 → LLM 产出合并结果 → 计算「新增/修改/删除」计数的差分 →
  差分与结果一并落库；差分超过阈值（例如删除 > 10% 或修改 > 30%）时**拒绝应用**并要求人工确认。
- **落点**：`memory/distill.go`（已有 `MergeCandidate`）+ `memory/eventlog.go`（差分入事件流）。
- **验收**：一次整理产生可读差分（新增 N / 修改 M / 删除 K）；超阈值时结果被拦下且事件标红；
  正常整理不改变既有记忆的 ID 与引用关系。

### M4 跨进程锁的位置语义（核对项）

- **做法**：若 gaea 的记忆锁文件位于记忆目录内部，改成**同级目录之外**（例如 `<root>/../.gaea-memory.lock`），
  理由：重置/归档会替换目录 inode，锁会跟着失效。
- **落点**：`memory/store.go` 或 `memory/file_backend.go` 的锁初始化处。
- **验收**：两个进程并发整理，一个拿锁一个放弃；在整理过程中触发「重置记忆」不会让锁提前失效。

---

## 四、不建议照搬的

- **两阶段 LLM 合并**本身：gaea 的 `morning.go` 是**确定性零 LLM** 的，这是优势（可离线、可复现、零成本）。
  codex 用 LLM 是因为它要跨会话归并长历史；gaea 已有 `distill.go` 的候选合并，按需再加 LLM 即可。
- **文件 workspace 作为唯一存储**：gaea 的 SQLite + 图存储更强，不必退回文件制。
- **版本化抽取契约（v1/v2 并存 + deny_unknown_fields）**：只有在「记忆 schema 会演进」时才值得，
  现在上等于提前背债。

---

## 五、一句话结论

codex 记忆管线里真正值得抄的是**四个防护**（脱敏、背压、基线差分、锁的位置语义），
而不是它的两阶段 LLM 结构；gaea 在数据结构与效果评估上已经领先，缺的是「写入前后的护栏」。
