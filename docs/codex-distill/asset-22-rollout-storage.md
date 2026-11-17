# 资产 22 · 会话记录存储：冷数据压缩与写入协调

> 源码：`rollout/src/{compression.rs 53 KB, writer_lock.rs 7 KB, recorder.rs 81 KB, seekable_reader.rs}`。
> 对照：gaea `internal/gaea/db/{database.go, schema.go 41 KB}`（单 SQLite 库）+ `sessionquery` + `trajectory`。
> 现状核对：gaea 侧 `zstd` / `gzip` / `compress` 在会话存储路径上 **0 处命中**——目前没有冷数据压缩。

---

## 一、codex 的做法：文件冷压缩 + 派生指标

```rust
const COMPRESSED_SUFFIX: &str = ".zst";
const COMPRESSION_LEVEL: i32 = 3;
const MIN_ROLLOUT_AGE: Duration = Duration::from_secs(7 * 24 * 60 * 60);   // ★ 只压 7 天以上的冷会话
const RUN_MARKER_FILE_NAME: &str = "rollout-compression.lock";
const RUN_MARKER_STALE_AFTER: Duration = Duration::from_secs(6 * 60 * 60); // 6h 未更新视为陈旧
const TEMP_FILE_STALE_AFTER: Duration = RUN_MARKER_STALE_AFTER;
const WORKER_MAX_RUNTIME: Duration = Duration::from_secs(5 * 60 * 60);
const MAX_CONCURRENT_COMPRESSION_JOBS: usize = 2;
const MAX_METADATA_WARNINGS_PER_RUN: usize = 5;
const TEMP_SUFFIX: &str = ".tmp";
const MAX_NOT_FOUND_RETRIES: usize = 3;
const OPEN_ROLLOUT_LINE_READER_RETRY_DELAY: Duration = Duration::from_millis(50);
```

四条设计纪律：

1. **只压冷数据**（`MIN_ROLLOUT_AGE = 7 天`）：热会话还在追加，压了反而要解压才能写；
2. **单实例锁 + 陈旧标记**：锁文件名固定，`6 小时`没更新就认为上一个进程死了，允许接管——
   避免「上次崩在压缩中途，此后再也不压」；
3. **并发与运行时长双上限**（2 个任务、单 worker 5 小时）：后台压缩不能吃满 CPU 也不能永远跑；
4. **指标齐全**：`file.source_bytes` / `compressed_bytes` / `compression_ratio` / `duration_ms`，
   加 run 级计数——没有压缩率就打不出「这功能有没有用」。

另外 `materialize_rollout_for_append_blocking(path)`：要往**已压缩**的会话追加时，
先解压成临时明文再写。这是文件方案特有的麻烦。

## 二、写入协调（`writer_lock.rs`）

```rust
const WRITER_LOCK_DIR: &str = "thread-writer-locks";
const COORDINATION_LOCK_FILE: &str = ".coordination.lock";
```

- 每个 thread 一个锁文件，放在专门目录；
- **锁文件的创建/删除本身**走一把全局协调锁，避免「同时创建同一把锁」的竞态。

这条对 gaea 有用：多窗口/多进程同时打开同一会话时，需要「每个会话一个写者」的保证，
而锁的**创建过程**也要串行化。

## 三、gaea 的差异（存储形态不同，方案要改）

| 维度 | codex | gaea |
|---|---|---|
| 明细存储 | 每会话一个 rollout 文件（JSONL） | 单 SQLite 库（`db.DatabasePath`，schema.go 41 KB） |
| 压缩单元 | 文件 → `.zst` | 无文件可压；对应的是**冷行归档** |
| 追加语义 | 文件追加，压缩后需 materialize | SQLite 行插入，无此问题 |
| 已有能力 | 全套指标与锁 | 有 `WithTransaction`，无归档压缩 |

**所以不能照搬「压文件」，要改造成「归档冷行」**：

```
1) 选冷数据：updated_at < now - 7d 的会话（与 codex 的 MIN_ROLLOUT_AGE 对齐）
2) 导出：把该会话的消息/事件按时间序导出为 JSONL → zstd(level=3) 写成 <session>.jsonl.zst
3) 校验：解压回读，比对条数与首尾 id（不一致就放弃删除）
4) 删除：事务内删行
5) 维护：定期 VACUUM（或 PRAGMA auto_vacuum）回收空间
6) 读取：查询命中已归档会话时，透明解压回读（UI 上标注「已归档」）
```

**锁与并行上限照抄**：锁文件 `archive.lock` + 6 小时陈旧接管 + 最多 2 个归档任务 + 单任务 5 小时上限。

## 四、实施建议（三步，1.5–2 天）

1. **步一（半天）**：`db/archive.go` —— 冷会话导出 + zstd 压缩 + 回读校验（纯函数，易测）；
2. **步二（半天）**：归档调度 —— 锁文件（含陈旧接管）、并发上限、运行时长上限、事件记录
   （`archived: N sessions, saved X KB`）；
3. **步三（半天）**：读取路径 —— `sessionquery` 命中归档会话时透明解压；UI 标注「已归档」。

**开关**：`storage.archive.enabled` 默认关；关时不产生任何 `.zst` 文件，行为与现状等价。

## 五、验收（可断言）

1. 归档一个 10 MB 会话 → 生成 `.zst`，解压回读条数与首尾 id 与归档前一致；
2. 归档后查询该会话 → 内容完整返回，UI 标「已归档」；
3. 伪造一个 6 小时前更新的锁文件 → 下次归档能接管并完成（陈旧标记生效）；
4. 同时触发两次归档 → 只有一个在跑（并发上限）；
5. 归档中途杀进程 → 不留下半成品（`.tmp` 先写后原子改名）；
6. 归档后库文件体积下降（`PRAGMA page_count * page_size` 断言）。

## 六、不建议照搬

- **`materialize_rollout_for_append_blocking`**：SQLite 行插入没有这个问题；
- **文件级 `rollout-compression.lock` 与 `.tmp` 命名细节**：意图照抄，实现按 Go 的 `os.Rename` 原子性写；
- **`recorder.rs`（81 KB）的写入管线**：codex 要处理「同会话多进程追加 + 崩溃恢复」，
  gaea 是单进程 + SQLite 事务，不需要那套。

## 七、一句话

这套的价值不在压缩算法（就是 zstd level 3），而在**四条运维纪律**：
只压冷数据、锁要能陈旧接管、并发与时长双上限、压缩率必须打点。
把它套到 gaea 的 SQLite 上，就是「冷会话归档 → 校验 → 删行 → VACUUM」。
