# asset-52 · 线程存储与状态库（B3 · 施工级）

> 证据根：`C:\AI\re\codex-0.165.0\oss\codex-0.165.0\codex-rs`
> 已读（正文级）：`thread-store/src/{lib.rs,types.rs,store.rs,thread_metadata_sync.rs,live_thread.rs,queue_store.rs}`、
> `thread-store/src/local/{mod.rs,live_writer.rs,read_thread.rs,list_threads.rs,search_threads.rs,revert_thread.rs,archive_thread.rs,history_revision.rs,thread_history.rs,update_thread_metadata.rs,rollout_lineage.rs,rollout_migration.rs}`、
> `thread-store/src/local/thread_history/{segment_paging.rs,read.rs}`、`thread-store/src/local/rollout_migration/{rollback_plan.rs,rollback_replay.rs}`、
> `state/src/{lib.rs,sqlite.rs,migrations.rs,paths.rs,audit.rs}`、`state/src/sqlite/busy_retry.rs`、
> `state/src/runtime/{reclamation.rs,backfill.rs,recovery.rs,threads.rs(节选)}`、`state/src/model/thread_metadata.rs(节选)`、
> `state/{migrations,thread_history_migrations}/*.sql`、`rollout/src/writer_lock.rs`；
> gaea 侧：`internal/chat/{db.go,store.go,branches.go}`、`internal/gaea/agent/session/{session.go,branch.go,checkpoint.go,save.go,log.go}`、
> `internal/gaea/control/{controller_session.go,controller.go(节选)}`、`docs/gaea-chat-branch-tree-design-2026-11.md`
> 只读对照，未改动 gaea 任何代码。

---

## 1. 一句话结论

codex 0.165.0 的线程存储是**「JSONL 为权威、SQLite 为可重建视图」的双层结构**：rollout JSONL 行文件承载唯一可重放历史，`state_5.sqlite` 只存元数据索引、`thread_history_1.sqlite` 只存 turn/item 投影；写入永远 JSONL 先落盘、投影后追且允许滞后（不允许超前），分页靠 `(thread_id, rollout_ordinal)` 键集游标 + 分段 lineage，删除/回退靠「新文件 + 指针 CAS」而非就地改写。整套设计的可抄点是**投影检查点（byte_offset + ordinal）+ 分段指针 + 三层锁 + 预算化页回收**这四件。

---

## 2. 机制地图（表格）

| 层 | 载体 | 谁写 | 谁读 | 权威性 | 证据 |
|---|---|---|---|---|---|
| 权威历史 | `sessions/**/rollout-*.jsonl`（行文件，append-only） | `RolloutRecorder`（`record_canonical_items` + `flush`） | `read_thread` / `load_history` / 投影器 | **唯一权威**，删了不可恢复 | `local/live_writer.rs:436-452`、`local/mod.rs:133-145` |
| 元数据索引 | `state_5.sqlite` → `threads`（+`thread_sections`/`projects`/`backfill_state`） | `update_thread_metadata` / `apply_rollout_items` | `list_threads`(过滤) / `read_thread`(快路径) | 可重建（缺失只掉性能与显示） | `sqlite.rs:43-48`、`migrations/0001_threads.sql:1-19` |
| 分页历史投影 | `thread_history_1.sqlite` → `thread_turns`/`thread_items`/`thread_history_projection_state` | `thread_history_materialization::materialize_to_sqlite` | `list_turns`/`list_items`/`list_timeline` | 可重建（注释原文 "SQLite is a rebuildable view"） | `thread_history_migrations/0001_thread_history.sql:1-38`、`live_writer.rs:440-441` |
| 队列 | `queue_1.sqlite` | `LocalQueueStore` | 前端待发消息 | 独立库 | `sqlite.rs:46`、`queue_store.rs:54-68` |
| 日志 | `logs_2.sqlite` | 批量 sink | 诊断 | 独立库，唯一开启页回收 | `sqlite.rs:90-99`、`log_db.rs:56-58` |
| 跨进程写锁 | `<codex_home>/thread-writer-locks/<thread_id>.lock` + `.coordination.lock` | `WriterLockCoordinator` | 所有写路径 | 进程存活即持有（OS advisory lock） | `rollout/src/writer_lock.rs:17-18,43-86` |

---

## 3. 数据模型与权威源（含一致性规则）

### 3.1 「双层」到底指什么

- **第一层：rollout JSONL**。注释写死语义：`Rollout JSONL files are the durable replay format and remain readable without SQLite`（`local/mod.rs:135-137`）。每行是一个 `RolloutLine { timestamp, ordinal, item }`，ordinal 由 canonicalizer 单调分配（`rollout_migration.rs:1013-1016` 返回 `next_ordinal`）。
- **第二层：SQLite**。当可用时是 `the queryable metadata index used by list/read paths for fast lookup`（`local/mod.rs:138-139`）。两个库分工：`state_5.sqlite` = 线程元数据；`thread_history_1.sqlite` = 分页历史投影（`sqlite.rs:47-48`）。
- **不是**主从复制关系：SQLite 没有任何反向写 JSONL 的路径。唯一从 DB 方向影响文件的是 `rollout_path` 指针（revert/归档）+ 名字索引（legacy 兼容）。

### 3.2 写入顺序（硬规则，可抄点）

```
append_items / persist / flush
  └─ write_and_project()
       ├─ [Legacy 模式] durable_write() 即完成
       └─ [Paginated 模式]
            ① durable_write(&recorder, op)   ← JSONL 先落盘（含 flush 屏障）
            ② materialize_to_sqlite(...)     ← 投影后追；Err 只 warn!，不返回错误
```
证据 `local/live_writer.rs:436-452`，注释原文：

> `SQLite is a rebuildable view. The flush barrier must win before projection starts so it can lag JSONL after failure, but can never get ahead of canonical history.`

这条"**允许滞后、禁止超前**"是整层设计的地基。反向守卫在投影端：`apply_projection` 在 `BEGIN IMMEDIATE` 事务内先读 `next_rollout_byte_offset`，若 `expected_offset != start_offset` 直接 `ThreadStoreError::Internal { "thread history projection ... is behind durable rollout" }`（`local/thread_history.rs:116-136`）。

### 3.3 投影检查点（第二层的游标）

`thread_history_projection_state(thread_id PK, next_rollout_byte_offset, next_rollout_ordinal)`（`thread_history_migrations/0001_thread_history.sql:34-38`）。它把「读到哪」记成**字节偏移 + ordinal 双游标**，二者与同一事务提交（`thread_history.rs:113-116`）。跳过不可投影的完整行时用 `SkippedOrdinalRange` 保证两游标仍指向同一 durable 前缀（`thread_history.rs:38-48`）。

### 3.4 冲突消解规则（五条，按优先级）

| # | 冲突 | 消解规则 | 证据 |
|---|---|---|---|
| R1 | SQLite 说 A 路径、调用方给 B 路径 | **显式路径赢**，且显式路径优先于陈旧 DB 行 | `live_writer.rs:82-100`；测试 `local/mod.rs:1024-1043`（SQLite 存 stale 路径，resume 仍用请求路径） |
| R2 | SQLite 元数据的 `rollout_path` 已失效 | 请求 history 时校验该路径的 `SessionMeta.id == thread_id`，不匹配即不信 DB | `read_thread.rs:139-152` |
| R3 | revert 与并发写竞争 | SQLite 侧 CAS：`UPDATE threads SET rollout_path=? WHERE id=? AND rollout_path=?`，`rows_affected!=1` 即冲突并删掉新文件 | `runtime/threads.rs:415-429` + `revert_thread.rs:135-150` |
| R4 | 迁移期间老进程继续追加 | publish 前比对源文件 `len` + `modified()`，变了就 `Conflict`「close older Codex processes and retry」 | `rollout_migration.rs:822-831` |
| R5 | SQLite 文件损坏 | 只搬该库及其 sidecar 到 `db-backups`，重建该库，不影响其它库 | `recovery.rs:96-119,130-139`；`RECOVERY` 常量 `recovery.rs:16` |

### 3.5 快照复用的一致性守卫

`history_revision` 是文件身份元组：`("local", canonical_path, len, modified, created, (dev,ino,ctime,ctime_nsec)@unix)` 的 JSON（`history_revision.rs:9-27`）。用法是**读前读后各取一次，前后一致才认为快照可用**（`read_thread.rs:297-304`）；`is_model_context_current` 也用它比对（`local/mod.rs:619-636`）。非 unix 平台 identity 恒 `None`——即 Windows 上守卫强度降为 `path+len+mtime+created`。

### 3.6 元数据「在哪里被观测」

append 出来的元数据**不在 store 内推导**：`ThreadMetadataSync` 在 live 层观测，再把 patch 通过 `record_thread_metadata` 显式写回（`thread_metadata_sync.rs:34-50`、`live_thread.rs:229-253`）。契约性测试钉死了这点：`raw_append_items_does_not_update_sqlite_metadata`（`local/mod.rs:953-987`）。节流常量 `THREAD_UPDATED_AT_TOUCH_INTERVAL = 5s`（`thread_metadata_sync.rs:28`）。

---

## 4. 读路径（列表/读取/搜索/分页：各自契约）

### 4.1 `read_thread` —— 三级回退

```
1) 若 !include_history 且 live writer 有未落盘 SessionMeta：
      直接用 writer 的 meta（"Missing JSONL is intentional"）          read_thread.rs:38-55
2) 若 SQLite 元数据存在 且（允许归档 或 未归档）且（不取 history 或 rollout 路径可校验）：
      用 SQLite 造 StoredThread；Legacy 模式再用 rollout 覆盖展示字段   read_thread.rs:62-108
3) 否则走 thread_rollout_resolver 解析路径，纯 rollout 构造             read_thread.rs:111-136
```
- **Paginated 模式禁止随线程返回全量 history**：`reject_paginated_history` → `error::reject_paginated_history_mode`（`read_thread.rs:214-219`）。理由注释在 `read_thread.rs:75-76`：`Paginated history may contain only a suffix, so its display metadata lives in SQLite.`
- 归档判据双通道：DB `archived_at` 或 路径在 `archived_sessions` 下（`read_thread.rs:64-68`）。
- `attach_history_if_requested` 用 revision 前后夹取（`read_thread.rs:284-306`）。

### 4.2 `list_threads` —— 两套引擎、两种游标

分支点 `list_threads.rs:28-30`：`sort_key == SectionPosition` 走 state DB，其余走 rollout 列表。

| 场景 | 引擎 | 游标形状 | 证据 |
|---|---|---|---|
| 默认（CreatedAt/UpdatedAt/RecencyAt） | `RolloutRecorder::list_threads*`（可 `use_state_db_only` 跳过 JSONL 修复扫描） | 不透明串，`codex_rollout::parse_cursor` 解析 | `list_threads.rs:31-39,268-328` |
| 任意有 `relation_filter`/`section`/`project_id` | `codex_rollout::state_db::list_threads_db` | 同上 | `list_threads.rs:244-266` |
| `SectionPosition` | `state_db.list_threads` / `list_threads_by_relation` | **`"{timestamp_millis}|{thread_id}"`** | `list_threads.rs:122-232`（拼装 `226-231`，解析 `140-167`） |

- keyset 语义：`Anchor { ts, id }`（`model/thread_metadata.rs:50-57`），SQL 生成 `column </> ts OR (column = ts AND id </> anchor_id)`（`runtime/threads.rs:1534-1569`），排序列固定带 `id` 次级键（`runtime/threads.rs:1610-1613`）。
- 溢出判定：取 `page_size + 1`，多出来的那条丢弃并作为 `next_anchor`（`runtime/threads.rs:523,542-551`）。
- **去重**：列表路径没有 SQL 级去重，靠 `stored_thread_from_rollout_item` 的 `filter_map` 丢弃不可解析项（`list_threads.rs:74-84`）——即"解析失败即少一条"，页大小可能小于 `page_size`。
- 二次补齐：名字（`resolve_thread_names`）、section_position/entered_at（`resolve_thread_section_metadata`）、`project_id` retain（`list_threads.rs:86-117`）。

### 4.3 `search_threads` —— ripgrep 预筛 + 分页合并

```
① search_rollout_matches(rg, codex_home, archived, term)      ← 外部 rg 进程，返回 (path, Option<snippet>)
② scan_page_size = (page_size*8).clamp(256, 2048)             ← 放大扫描页
③ loop: list_rollout_threads(scan_page_size) → 命中 map 取走一项
④ snippet 两档：rg 已给就用；否则 first_rollout_content_match_snippet 现读
⑤ matching_items.len() > page_size 即"还有更多"，截断后 next_cursor = 最后一条
```
证据：`search_threads.rs:77-93`（rg 预筛与空短路）、`:96`（扫描页放大）、`:117-153`（循环与退出条件）、`:130-140`（snippet 两档）、`:155-166`（溢出与游标）。

- **路径归一化**是去重键：`rollout_search_path` 把压缩 sibling 折回 plain、Windows 上再跑 `normalize_windows_device_path`，否则 resume 存的 namespace 前缀与 rg 返回的拼写对不上（`search_threads.rs:204-215`）。
- 游标构造分排序键：`RecencyAt` 用 `"{timestamp}|{thread_id}"`，`CreatedAt/UpdatedAt` 用纯 timestamp（`search_threads.rs:217-241`）。`SectionPosition` 在搜索里直接拒绝（`search_threads.rs:59-63`）。

### 4.4 `list_turns` / `list_items` —— 分页契约

- 入参先校验：`validate_thread_for_paginated_reads` + `validate_page_size`（`thread_history/read.rs:106-113`）；`page_size == 0` 报错（`segment_paging.rs:34-43`）。
- **双游标**：`next_cursor`（`include_anchor=false`，排他）与 `backwards_cursor`（`include_anchor=true`，含锚），只有存在下一页时才给 `next_cursor`（`segment_paging.rs:498-536`）。
- 游标形状（JSON，`serde(tag="kind", rename_all="camelCase")`）：

```json
{ "requestedThreadId": "<uuid>", "rolloutOrdinal": 12345,
  "includeAnchor": false, "scope": "turns" }
```
`scope ∈ { turns, itemsByCreatedAtOrdinal, itemsByUpdatedAtOrdinal }`（`thread_history/read.rs:30-45`）。解析时**同时校验 threadId 与 scope**，任一不符即 `invalid cursor`（`read.rs:276-290`）——这是防跨线程/跨 scope 串游标的硬关卡。
- 更新序排序的两个额外前置：必须给 `after_updated_at_ordinal`（`segment_paging.rs:220-239`），且**只允许单段 lineage**（`segment_paging.rs:215-219`，报错 "incremental item replay is not supported for forked threads"）。
- `item_ids` 精确查 / `turn_id` 过滤 / `INDEXED BY idx_thread_items_by_turn_page` 三处索引提示见 `segment_paging.rs:297-323`。

---

## 5. 历史分段与重放

### 5.1 段是什么

```rust
struct RolloutLineageSegment {          // rollout_lineage.rs:15-22
    rollout_id: ThreadId,               // 段所属 rollout 的不可变 ID（≠ 逻辑 thread_id）
    rollout_path: PathBuf,
    start_ordinal: u64,                 // 本段在合并序里的起始 ordinal
    end: Option<HistoryPosition>,       // None = 段尾即文件尾
}
struct RolloutLineage { segments: Vec<RolloutLineageSegment> }   // :29-31
```
`ThreadId`（逻辑线程，永久不变）与 `rollout_id`（物理文件身份）**是两个 ID**；`live_recorders` 里两个都存，注释明说 `Rollout projection rows are keyed by immutable rollout ID, not the stable thread ID`（`local/mod.rs:164-174`）。

### 5.2 段首如何定位

- 段链来源是 `SessionMeta.history_base: Option<HistoryPosition>`：父指针。`RolloutLineage` 是**唯一**跟这个指针的抽象（`rollout_lineage.rs:26-28` 原文）。
- 解析循环三态（`rollout_lineage.rs:91-122`）：有 next id → 按 id 找路径；有初始 path → 由 path 反推 rollout_id；都没有 → `resolve_current_including_archived`。
- 环检测：`if !seen.insert(rollout_id) { return Err(malformed_lineage(requested_thread_id, "cycle detected")) }`（`rollout_lineage.rs:123-125`）。
- `PlainForReference` 模式额外做「路径必须在 managed roots 内」的符号链接逃逸检查（`rollout_lineage.rs:130-156`）。

### 5.3 跨段读取的边界条件

| 边界 | 规则 | 证据 |
|---|---|---|
| 段内取值 | `rollout_ordinal >= start AND rollout_ordinal < end`（`end=None` 则无上界） | `segment_paging.rs:442-455` |
| 段遍历顺序 | Asc：`cursor_index..len`；Desc：`0..=cursor_index` 反转；无 cursor 时 Asc 从 0、Desc 从末段 | `segment_paging.rs:409-440` |
| 游标只作用所在段 | 其它段的 `segment_cursor = None`，即"新段从段首重新开始" | `segment_paging.rs:429-439` |
| 跨段同 turn 去重 | 对每个更晚段加 `NOT EXISTS (SELECT 1 FROM thread_turns newer WHERE newer.thread_id=<later> AND newer.turn_id=thread_turns.turn_id AND newer.rollout_ordinal >= later.start [AND < later.end])` → **同名 turn 以最晚段为准** | `segment_paging.rs:85-97` |
| 摘要列跨段取值 | `push_summary_item_join` 把 first_user/final_agent 的 join 也夹在本段 ordinal 区间内 | `segment_paging.rs:166-192` |
| 页预算跨段递减 | `remaining_limit(page_size, rows.len()) = page_size+1-row_count`，取到 0 即 `break` 停止后续段 | `segment_paging.rs:59-62,288-291,490-496` |
| 游标越界 | `segment_index_for_ordinal` 返回 None → `invalid_cursor("position outside thread lineage")` | `segment_paging.rs:415-421` |
| 单项锚点 | `ListItemsPosition::ItemAnchor` 必须伴随 `turn_id`，且仅允许 `CreatedAtOrdinal` 且无 update watermark | `segment_paging.rs:199-250` |

### 5.4 「重放」在两个地方，不是一件事

1. **冷恢复重放**（读侧）：`rollback_replay` 逐条镜像老版 cold resume 的**反向扫描**，从文件尾往前走，跳过被回滚的段，选出最新的「有 replacement_history 的 `Compacted`」作为模型上下文起点（`rollback_replay.rs:87-173`）。
2. **迁移重放**（写侧）：canonicalizer 两遍流式重写（见 §6.1）。

---

## 6. 迁移、回填、回滚

### 6.1 `rollout_migration` —— 触发、幂等、续跑

**触发点**
- 启动：`migrate_rollouts_on_startup`（`rollout_migration.rs:277-279`，实现 `rollout_migration/startup.rs`）。
- 手工/带进度：`migrate_rollouts` / `migrate_rollouts_with_progress`（`:282-308`）。
- 模式：`DryRun`（默认，只报 `Eligible`）| `Apply`（`rollout_migration.rs:112-120`）。

**单文件状态机**（`migrate_one_rollout`，`rollout_migration.rs:627-855`）
```
0. 有 state_db 则必须存在该 thread 的 SQLite 行，否则 MissingSqliteMetadata  (:636-650)
1. 先删旧投影 thread_history::delete_thread(thread_id)                  (:663-666)
2. 写 .pending journal（durable）                                        (:667-671)
3. 若压缩：先解压到 staged sibling                                       (:681-699)
4. 跳过 pre-header 前缀，找 ordinal=0 的 canonical SessionMeta            (:707-728)
5. 流式 canonicalize → staged JSONL（带 rollback plan 的重试见下）       (:741-784)
6. fsync staged（并回填源 mtime）                                        (:764-776)
7. 批量投影 staged → SQLite，然后**校验** next_byte_offset==len && next_ordinal==expected  (:787-806)
8. publish：源文件 len+mtime 未变才 rename 覆盖；再同步父目录            (:810-849)
9. finish_published_migration：提升 legacy 名字 → 删 journal → fsync 父目录 (:1099-1110)
```

**幂等性来源（三条叠加）**
| 来源 | 说明 | 证据 |
|---|---|---|
| ① `.pending` journal 文件 | 判据 `history_mode==Paginated && Apply && journal exists` ⇒ 认为"已发布但未收尾" | `rollout_migration.rs:496-502` |
| ② 投影检查点比对 | `recover_published_migration` 只在 `next_byte_offset != len` 时才删投影重做 | `:1060-1065` |
| ③ 重跑前先清投影 | `delete_thread(thread_id)` 幂等重置，投影是视图所以随便删 | `:663-666`、`:1089-1096` |
| ④ 名字提升幂等 | `promote_legacy_name` 可重复调用 | `:1105`、`:1112-1119` |

**失败如何续跑**
- 已发布未收尾 → `recover_published_migration`：重解压（如需）→ 补投影 → 删 staged/compressed/decompressed 残留 → 收尾 journal（`rollout_migration.rs:1019-1074`）。失败归类 `InterruptedMigrationRecoveryFailed`（`:530-533`）；撞锁则 `SkippedBusy`（`:521-529`）。
- 未发布失败 → `cleanup_failed_unpublished_migration`：确认源仍是 legacy（`history_mode != Paginated` 才清）→ 删投影 → 删 4 种 staged 残留 → 删 journal → fsync（`:1076-1097`）。
- 判据是"源文件是否已变 paginated"：**已变就不清**（`:1085-1087`），避免删掉刚发布成功的成果。

**两遍写（回滚标记的代价）**
第一遍遇到 `EventMsg::ThreadRolledBack` 立刻返回 `NeedsRollbackPlan` 并丢弃 staged（`:978-985`、`:884-886`）；第二遍先 `build_rollback_plan` 完整扫一遍源（`:857-872`），再带 plan 重写；结尾用 `parsed_record_index != plan.record_count()` 防"扫描期间源被改"（`:1002-1008`）。

**吞吐控制**：`PROJECTION_BATCH_BYTES = 256 KiB` 为限额与 yield 粒度（`rollout_migration.rs:79`、`:253-271`）；`max_mib_per_second` 可选，缺省不睡只 `yield_now`（`:127-129`、`:260-262`）。

### 6.2 `backfill` —— state 侧的单行租约

与 `rollout_migration` 完全是两件事：backfill 只把 **rollout 元数据灌进 `threads` 表**，游标是 **watermark 路径字符串**，不是字节偏移。

| 动作 | SQL 要点 | 证据 |
|---|---|---|
| 抢 worker slot | 单行 `UPDATE backfill_state SET status=Running WHERE id=1 AND status!='complete' AND (status!='running' OR updated_at <= now-lease)`；`rows_affected==1` 才算抢到 | `backfill.rs:24-45` |
| 进度 | `checkpoint_backfill(watermark)` 写 `last_watermark` + `updated_at` | `:65-80` |
| 完成 | `status=complete`，此后 `try_claim_backfill` 永远返回 false | `:83-104` + 测试 `:277-285` |
| 租约过期可接管 | 测试用 `updated_at=now-10000` 模拟旧租约，`lease_seconds=10` 抢回 | 测试 `:257-275` |

幂等来源：**单行 CAS**（`WHERE id = 1 AND ...`）+ `rows_affected` 判定，无外部状态。缺陷：lease 只靠 `updated_at`，长时间无 checkpoint 会被误判陈旧（同 `writer_lock` 的取舍相反，见 §8.2）。

### 6.3 `rollback_plan` / `rollback_replay` / `revert_thread` —— 三者语义差别

| 维度 | `rollback_plan` | `rollback_replay` | `revert_thread` |
|---|---|---|---|
| 回答的问题 | 「legacy 回滚后，**哪些记录还可见**」 | 「legacy 冷恢复会**从哪个 checkpoint 起点**重放」 | 「把这个线程的持久历史**截断到某 turn 之前**」 |
| 输入 | 源 JSONL 全量（按 `record_index`） | 源 JSONL 全量（按语义事件反向扫） | 现役 rollout + lineage |
| 输出 | `record_boundaries: Vec<Option<usize>>`、`boundary_alive`、`compacted_items` | `ModelReplayPlan { empty_replacement_history_compaction: Option<usize> }` | 一个**全新 rollout 文件** + SQLite 指针切换 |
| 谁删 | **它删**：`apply()` 返回 `None` 即该行不进新文件；`ThreadRolledBack` 标记行无条件丢弃 | **它不删**，只标出"需要保留一个空 checkpoint"以免模型上下文回退 | **它不删旧文件**：`Old rollouts stay intact`（`revert_thread.rs:15-18`） |
| 谁重放 | 它是重放的**过滤器**（第二遍写时逐条 `plan.apply(index, line)`） | 它是重放的**起点选择器**（镜像 cold resume） | 它**不重放**，只冻结前缀：`history_base` + `forked_from_ordinal_exclusive` |
| 关键证据 | `rollback_plan.rs:71-96`（`apply`）、`:99-115`（planner 状态）、`:1-8`（注释解释"回滚删的是逻辑指令轮，不是物理后缀"） | `rollback_replay.rs:1-6`（两个职责）、`:161-172`（空 checkpoint 的推导） | `revert_thread.rs:19-152` |

`rollback_plan` 的难点注释值得抄：`Legacy rollback removes logical instruction turns, not a physical suffix of the rollout file. Most records happen to be ordered that way, but late completion events can target an older surviving turn after a newer turn has started.`（`rollback_plan.rs:3-6`）——所以它维护的是**每条记录 → 指令边界**的所有权映射，而不是简单截断。

**revert 的落盘顺序**（可抄的安全切换）：
```
0. lifecycle 写锁 → writer 锁 → 断言无 live recorder → 文件写锁          revert_thread.rs:34-37
1. 读 SQLite 期望路径（CAS 的 expected）                                  :40-47
2. 校验：SessionMeta.id 对、history_mode==Paginated                       :52-70
3. 逐段 materialize（首段从压缩物化出可引用副本）+ 投影补齐                :82-104
4. history_base = paginated_fork::history_base_at_boundary(BeforeTurn)    :105-111
5. 新 rollout_id，建 replacement recorder，persist + shutdown             :121-133
6. replace_rollout_path_if_current(CAS)；失败则删新文件报 Conflict        :135-150
```

**真正删文件的只有 `delete_thread`**：先做引用检查（`ensure_no_external_references`），再删投影（`thread_history::delete_thread(store, rollout_id)`），再删磁盘文件（`delete_rollout_file` → `remove_file`），路径必须落在 `sessions` 根内（`delete_thread.rs:271,288,303-332`）。归档不是删：`archive_thread` 是 `std::fs::rename` 到 `archived_sessions`，失败批量回滚已移动项（`archive_thread.rs:118-148`）。

---

## 7. 状态库：schema / 事务 / 锁 / 页回收（常量表）

### 7.1 库清单（7 个，`RuntimeDbSpec`，`sqlite.rs:56-67,148-156`）

| 库 | 文件名 | `background_reclamation` | `recovery` | 证据 |
|---|---|---|---|---|
| state DB | `state_5.sqlite` | false | BackupAndRebuild | `sqlite.rs:80-88` |
| log DB | `logs_2.sqlite` | **true** | BackupAndRebuild | `sqlite.rs:90-99` |
| goals DB | `goals_1.sqlite` | false | BackupAndRebuild | `sqlite.rs:101-109` |
| memories DB | `memories_1.sqlite` | false | BackupAndRebuild | `sqlite.rs:111-119` |
| memories v2 DB | `memories_v2_1.sqlite` | false | BackupAndRebuild | `sqlite.rs:121-126` |
| queue DB | `queue_1.sqlite` | false | BackupAndRebuild | `sqlite.rs:128-136` |
| thread history DB | `thread_history_1.sqlite` | false | **Unavailable** | `sqlite.rs:138-146` |

注意两点设计取舍：① **只有 log DB 开页回收**，理由注释是 `Log transactions write before reading, so they already hold the writer lock`（`sqlite.rs:96`），即回收提交会打断"先读后升写"的事务（`SQLITE_BUSY_SNAPSHOT`，`sqlite.rs:63-64`）；② thread history 库损坏时**不走备份重建**（`RecoveryMode::Unavailable`），因为它整个是投影。

### 7.2 schema

`threads`（`migrations/0001_threads.sql:1-19`）：`id TEXT PK, rollout_path TEXT NOT NULL, created_at, updated_at, source, model_provider, cwd, title, sandbox_policy, approval_mode, tokens_used, has_user_event, archived, archived_at, git_sha, git_branch, git_origin_url`；索引 `idx_threads_{created_at,updated_at,archived,source,provider}`（`:21-25`）。

后续增量（只列与本资产相关的）：
- `recency_at`/`recency_at_ms` + **insert 后补种触发器**（`WHEN NEW.recency_at_ms = 0` 时回填，供老二进制插入新库用）+ 3 条排序索引 `idx_threads_recency_at_ms` / `idx_threads_archived_cwd_recency_at_ms` / `idx_threads_visible_recency_at_ms`（部分索引 `WHERE preview <> ''`）（`migrations/0039_threads_recency_at.sql:1-27`）。
- `section_position`/`section_entered_at_ms`（`0046_threads_section_order.sql:1-19`）。
- `thread_sections` 表（`0045_threads_section.sql:1`）、`projects`（`0049`）、`backfill_state`（`0008`）、`rollout_migration_state`（`0047`）、`thread_read_state`（`0061`）、`thread_attachments`（`0055`）。

分页投影三表（`thread_history_migrations/0001_thread_history.sql`）：
```sql
thread_turns(thread_id, turn_id, rollout_ordinal, status, error_json,
             started_at, completed_at, duration_ms,
             first_user_item_id, final_agent_item_id, PRIMARY KEY(thread_id,turn_id))
  + UNIQUE idx_thread_turns_page(thread_id, rollout_ordinal)
thread_items(thread_id, turn_id, item_id, rollout_ordinal, created_at_ms, item_json,
             PRIMARY KEY(thread_id, turn_id, item_id))
  + UNIQUE idx_thread_items_page(thread_id, rollout_ordinal)
  + idx_thread_items_by_turn_page(thread_id, turn_id, rollout_ordinal)
thread_history_projection_state(thread_id PK, next_rollout_byte_offset, next_rollout_ordinal)
```
`0004` 加 `updated_at_ordinal`（**故意非 unique**：`Keep this index non-unique so mixed-version writers can continue to persist history`，`0004_thread_items_updated_at_ordinal.sql`）；`0003` 给 turn 加 `rollout_byte_offset/rollout_end_ordinal/rollout_end_byte_offset`；`0006` 加部分索引 `idx_thread_turns_end_page ... WHERE rollout_end_ordinal IS NOT NULL`。

### 7.3 事务与重试

- 写入口统一 `SqlitePoolWriteExt::begin_immediate()` → `BEGIN IMMEDIATE`，**只重试准入，不重试已开始的事务或外部副作用**（`busy_retry.rs:23-40`）。
- 退避：`[100ms, 200ms]` 两档，注释 `SQLite already waits up to five seconds per lock; allow two more attempts`（`busy_retry.rs:132-151`）。
- `DEFAULT_BUSY_TIMEOUT = 5s`（`sqlite.rs:41`）。
- busy 判定收 4 个码：`SQLITE_BUSY / BUSY_RECOVERY / BUSY_SNAPSHOT / BUSY_TIMEOUT`（`busy_retry.rs:176-187`）。
- 单条 autocommit 写用 `execute_with_busy_retry`；RETURNING 写用 `fetch_one_write`（先 `begin_immediate`、解码行、再 commit，`busy_retry.rs:78-110`）。

### 7.4 常量表（生产值；`#[cfg(test)]` 里另注）

**页回收组（逐条，`state/src/runtime/reclamation.rs`）**

| 常量 / 门槛 | 生产值 | 语义 | 行 |
|---|---|---|---|
| `IDLE_INTERVAL` | **60 s** | 无事可做/空闲/错误的下一轮间隔 | `:21` |
| `ACTIVE_INTERVAL` | **50 ms** | 本轮有进展（Active/Interrupted）立即再来 | `:22` |
| `CONTENTION_INTERVAL` | **500 ms** | 撞锁（Contended）退让 | `:23` |
| `PASS_DURATION` | **100 ms** | 单 pass 合作式 deadline | `:24` |
| `MIN_FREE_BYTES` | **64 MiB** | 空闲页总字节下限，不到就 Idle | `:25` |
| `RESERVE_BYTES` | **16 MiB** | 保留储备，永不回收 | `:26` |
| `PASS_PAGES` | **1024 页** | 单 pass 最多回收页数 | `:27` |
| `BATCH_PAGES` | **64 页** | 单事务批量，失速时减半 | `:28` |
| 百分比门槛 | `free < pages / 4` | 空闲页占比不足 **25%** 即 Idle | `:324` |
| 模式门槛 | `journal != "wal"` 或 `auto_vacuum != 2` | 必须 WAL + **incremental** auto-vacuum | `:324` |
| 实际 reserve | `max(RESERVE_BYTES/page_size, pages/10)` | 页数口径，至少 10% | `:333` |
| 批量后休眠 | **5 ms** | 给前台写者让路 | `:370` |
| 失速减半下限 | `max(batch_pages/2, 1)` | 连续失速逐次减半 | `:175` |
| 失速退避 | `500ms * 2^(stalled-1)`，指数上限 2^7 | 500 ms → 60 s | `:177-179` |
| 维护连接 `busy_timeout` | **0** | 维护连接绝不等待前台 | `:248` |
| 维护连接 `cache_size` | **-16384**（16 MiB） | 大库实测降 CPU | `:253` |
| 维护连接 `wal_autocheckpoint` | **0** | 提交时不做 checkpoint I/O | `:251` |
| 维护连接 `synchronous` | `Normal` | 非 `Full` | `:249` |
| 选主锁文件 | `<sqlite_home>/.sqlite-maintenance.lock` | per-home，非 per-db | `:128` |

**其它常量（同属本资产）**

| 常量 | 生产值 | 出处 |
|---|---|---|
| `PROJECTION_BATCH_BYTES` | **256 KiB** | `thread-store/src/local/rollout_migration.rs:79` |
| `MAX_ROLLOUT_LINE_BYTES` | **16 MiB** | 同上 `:80` |
| `BATCH_SOURCE_BYTES` | **1 MiB** | `local/thread_history_materialization.rs:28` |
| `BATCH_STEPS` | **256** | 同上 `:29` |
| `LOG_QUEUE_CAPACITY` | **2048** | `state/src/log_db.rs:56` |
| `LOG_BATCH_SIZE` | **512** | `log_db.rs:57` |
| `LOG_FLUSH_INTERVAL` | **10 s** | `log_db.rs:58` |
| `DEFAULT_BUSY_TIMEOUT` | **5 s** | `state/src/sqlite.rs:41` |
| `MAX_QUEUE_ITEMS` | **100** | `state/src/lib.rs:117` |
| `MAX_THREAD_ATTACHMENT_PAYLOAD_BYTES` | **64 KiB** | `state/src/lib.rs:120` |
| `MAX_THREAD_ATTACHMENT_LIST_PAGE_SIZE` | **100** | `state/src/lib.rs:129` |
| `THREAD_UPDATED_AT_TOUCH_INTERVAL` | **5 s** | `thread-store/src/thread_metadata_sync.rs:28` |
| 搜索扫描页 | `(page_size*8).clamp(256, 2048)` | `local/search_threads.rs:96` |
| 路径解析重试 | **3 次** | `local/read_thread.rs:251` |
| 页大小上限 | `i64` 可表示即可（`page_size+1` 溢出即错） | `segment_paging.rs:34-43` |

### 7.5 页回收的四个非平凡机制

**(a) 选主**：`owners: BTreeMap<PathBuf, File>` 按 **home 目录**（不是库文件）缓存 owner file handle；某 home 第一次访问时 `try_ownership(home)` 建/开 `.sqlite-maintenance.lock` 并 `try_lock()`，`WouldBlock` 说明别的进程在回收 → 该库推迟 `IDLE_INTERVAL`（`reclamation.rs:87-98,122-137`）。一个进程内同一 home 只持一个 owner（`Entry::Vacant` 才尝试），因此**同 home 的多库串行访问**。

**(b) NFS 特判**：`reclaim()` 开头就判 NFS，是则直接返回 `Idle, pages=0`——**不建维护连接**（`reclamation.rs:221-233`）。两套探测分别对应 logs 目录与主 home（`sqlite.rs:220-227`，`is_nfs`/`logs_is_nfs` 各一个 `OnceCell` 只探一次）。

**(c) 失败退避**：`ReclamationState::retry_after`（`:168-192`）。唯一"失速"判据是 `Interrupted && pages == 0`（`:170-173`）——即"给了 deadline 却一页没回收"。持续失速时批量减半 + 指数退避；**一旦有进展，减小的批量保留不恢复**（注释 `Keep the smaller batch after progress; growing it would repeat the stall`，`:182`）。

**(d) 合作式中止与争用检测**：
- `set_progress_handler(1, ...)` 让 SQLite 每 1 个 VM op 检查一次 shutdown/deadline（`:305-313`）。
- 每批在 `BEGIN IMMEDIATE` 事务内重读 `freelist_count`，`free <= reserve` 即 `rollback` 转 Idle——注释说明了为什么必须在同一事务里读：`a foreground writer cannot consume the reserve between our free-page check and vacuum`（`:346-354`）。
- 每批后 `wal_checkpoint(PASSIVE)` 必须 `busy==0 && frames==done`，否则 `Contended` 并 break（`:365-368`、`:425-433`）。
- 每批后比对 `PRAGMA data_version`，变了说明有别的连接提交过 → `Contended` break（`:371-378`）。
- 错误码映射：`SQLITE_BUSY/LOCKED → Contended`，`SQLITE_INTERRUPT → Interrupted`，其余上抛（`:410-423`）。

---

## 8. 并发与失败路径

### 8.1 写锁是谁的（三层，缺一不可）

| 层 | 类型 | 保护什么 | 证据 |
|---|---|---|---|
| L1 in-process 写串行 | `tokio::sync::Mutex<()>`（`ThreadCoordination.writer`） | 同一线程的 append/persist/flush 顺序；fork 快照一致性 | `local/mod.rs:183-192,204-211` |
| L1' in-process 生命周期 | `tokio::sync::RwLock<()>`（`ThreadCoordination.lifecycle`） | fork 持**共享读**租约直到子引用落盘；delete/archive/unarchive 要**写**锁 | `local/mod.rs:186-191,213-229` |
| L2 跨进程文件锁 | `<home>/thread-writer-locks/<thread_id>.lock` 的 `File::try_lock()` | 同机多 Codex 进程同一线程互斥；`WouldBlock` → `ThreadStoreError::Conflict` | `rollout/src/writer_lock.rs:43-86`；映射 `local/mod.rs:374-389` |
| L3 SQLite 事务 | `BEGIN IMMEDIATE` | 投影推进、元数据 upsert、CAS、队列 | `busy_retry.rs:32-40` |

**锁序（硬约定）**：`lifecycle` 先于 `writer`（`local/mod.rs:190` 注释）。批量操作按 `thread_id` 字符串排序去重后统一加锁，避免 ABBA（`archive_thread.rs:24-40`）。`archive`/`delete` 的加锁顺序是：全部 lifecycle → 全部 writer（同时断言无 live recorder）→ 全部文件锁（`archive_thread.rs:28-41`）。

**`live_writer_locks` 永不主动回收**：`Keep per-thread locks after a writer goes idle. Removing one while another caller is about to acquire it could let two operations for the same thread run at once.`（`local/mod.rs:178-180`）——代价是 `HashMap` 随线程数单调增长，可抄点也是可抄的坑。

### 8.2 「锁可陈旧接管」的判据在哪个文件

`rollout/src/writer_lock.rs`，两条：

1. **判据本体**：`remove_stale_thread_locks()`（`:124-171`）。对 `thread-writer-locks/` 下每个 `<uuid>.lock`（非 uuid 文件名跳过）开文件并 `try_lock()`：
   - `Ok(())` ⇒ **此刻没有任何进程持有它** ⇒ 持锁期间 `remove_file` 删掉；
   - `WouldBlock` ⇒ 有活着的持有者 ⇒ 不动；
   - `Err(Error(_))` ⇒ 只 warn。
   
   之所以"能 try_lock 就说明陈旧"：锁是 OS advisory lock，进程死亡即释放；文件残留不代表锁残留。

2. **只在协调锁内执行，且一辈子只跑一次**：`acquire()` 先 `lock_coordination()`（`.coordination.lock`，阻塞式 `file.lock()`，`:108-122`），再 `cleanup_attempted.swap(true, Relaxed)` 保证单进程单次（`:45-49`）。

3. **释放端顺序**：`Drop for WriterLockGuard` 先 `drop(self.file.take())` 关文件，再 `remove_file`——注释 `Close the writer lock before deleting it so cleanup works on Windows too`（`:184-193`）。**Windows 上不先关闭就删不掉**，这条必须抄。

**对照：`backfill` 的"陈旧"是另一套语义**——按 `updated_at <= now - lease_seconds` 的时间租约（`backfill.rs:24-45`），会误判长任务。两者取舍相反：文件锁不误判但要求"能接触锁文件所在路径"，时间租约不要求但依赖心跳。

### 8.3 失败路径清单

| 失败点 | 行为 | 是否影响可用性 | 证据 |
|---|---|---|---|
| JSONL 落盘失败 | 返回错误（写路径唯一硬失败点） | 是 | `live_writer.rs:437,442,459-470` |
| 投影失败（JSONL 已成功） | `warn!("failed to project durable rollout")` 后继续 | 否（可重建） | `live_writer.rs:443-451` |
| shutdown 时投影失败 | 同上，但仍 `sync_materialized_rollout_path` | 否 | `live_writer.rs:216-226` |
| 投影超前 JSONL | `Internal` 错误，拒绝推进 | 是（数据不一致信号） | `thread_history.rs:132-136` |
| 投影 ordinal 不连续 | `Internal { expected ordinal X, got Y }` | 是 | `thread_history.rs:141-148` |
| 迁移 publish 撞改动 | `Conflict`，提示关闭老进程 | 否（重试） | `rollout_migration.rs:822-831` |
| 迁移撞维护锁 | `Conflict { "rollout compression or another migration is already running" }` | 否（跳过） | `rollout_migration.rs:334-341` |
| 迁移撞 writer 锁 | `SkippedBusy`（**不是** Failed） | 否 | `:436-444,560-566,618-624` |
| revert CAS 失败 | 删新文件 + `Conflict` | 否 | `revert_thread.rs:145-150` |
| 归档 rename 中途失败 | 逆序 restore 已移动项 | 否 | `archive_thread.rs:118-148` |
| 空 rollout（writer 建了文件还没写 meta） | 先抢 writer 锁再重读，避免"永久空跳过" | 否 | `rollout_migration.rs:427-451` |
| SQLite 损坏 | 只迁该库 + sidecar 到 `db-backups`，重建 | 否（其余库不受影响） | `recovery.rs:96-119` |
| 未落盘的 live 线程读不到 | `InvalidRequest { "no rollout found for thread id" }` | 符合契约 | `read_thread.rs:45-54` |

**pending metadata 的收尾规则**（容易漏）：`shutdown_thread` 只有在 rollout 文件**不存在**时才丢弃 staged patch（`live_writer.rs:233-246`）；文件存在则保留给首次成功 update 消费。

---

## 9. gaea 对照

> 注：任务书给的 `internal/gaea/chat/*.go` **在本仓库不存在**（`Test-Path internal\gaea\chat` = False）。实际对应实现有两条线：①聊天板块落库 `internal/chat/{db.go,store.go,branches.go}`（SQLite `chat.db`）；②agent 会话转录 `internal/gaea/agent/session/*`（JSONL + 事件日志 + 检查点）。下表按这两条线取证。

| # | 维度 | codex 0.165.0 | gaea | 判定 |
|---|---|---|---|---|
| G1 | **分支树 vs 线程分段** | 分段是**物理文件 lineage**：`history_base: HistoryPosition` 指父前缀，段以 `(rollout_id, start_ordinal, end)` 表达，读侧按 ordinal 区间裁 SQL（`rollout_lineage.rs:15-31`、`segment_paging.rs:442-455`） | 分支是**行内语义指针**：`extra.bt = {p, c}`，`c` 挂父消息，可见路径靠叶子回溯在**展示层**过滤，DB 里所有分支行都在（`internal/chat/branches.go:18-22`、设计文档 `:28-36`） | **口径不同**。gaea 分支不产生物理隔离，因此"分支各自的选择"能零迁移保留（`bt.c` 挂父），代价是每次装配历史都要算可见路径且 `hasMore` 按全集计数会翻出不可见行（设计文档 `:83-85` 自己记档了）。codex 的段有物理边界，跨段去重必须靠 `NOT EXISTS newer_turn` 这类昂贵谓词 |
| G2 | **分页 vs 全量** | 双游标（`next`/`backwards`）+ 复合 keyset + 段跨页 + `page_size+1` 溢出判定（`segment_paging.rs:498-536`、`runtime/threads.rs:523,542-551`） | 聊天侧有 keyset 但只有单列：`WHERE topic_id=? AND (?<=0 OR seq<?) ORDER BY seq DESC LIMIT ?`，`limit<=0 \|\| limit>1000` 回落 200（`internal/chat/store.go:373-408`）；`ListMessages` 全量拉取仍在用（`:410-416`）。会话侧**没有分页**：历史面板每次刷新对全部 `.jsonl` 全量解码，只靠 `size+mtime` 预览缓存（上限 64 条，超出整体清空）兜（`internal/gaea/agent/session/save.go:303-306,337-340`） | **gaea 缺**。单列 keyset 无法消除同 seq 竞态（`seq` 非唯一），且"全量解码 + 64 条缓存"在多 MB 工具输出的会话上必然退化——注释里已经写明这个现状 |
| G3 | **页回收 vs 无回收** | 全套：选主锁 + 预算（1024 页/100 ms）+ 64 MiB 门槛 + 16 MiB 储备 + 25% 占比 + 失速减半与指数退避 + NFS 跳过（`reclamation.rs:21-28,122-137,168-192,221-233,322-333`） | 只有关闭时 `PRAGMA wal_checkpoint(TRUNCATE)`（`internal/chat/db.go:58`），无 `incremental_vacuum`、无 leader、无预算、无 NFS 判据 | **gaea 缺**。长期运行 + 频繁分页读的 `chat.db`/索引库会留下 freelist；且 `_journal_mode=WAL` 下 `TRUNCATE` checkpoint 只在关库时执行一次（`db.go:34-35`） |
| G4 | **权威源口径** | 文件权威、DB 视图，允许 DB 滞后禁止超前（`live_writer.rs:440-441`、`thread_history.rs:132-136`） | **完全分离的两套**：聊天板块 DB 是唯一真相（无 JSONL 转录，`internal/chat/db.go:34-45`）；agent 会话是四件套文件：`<id>.jsonl`（消息）+ `<id>.gaea-log.jsonl`（事件日志 seq）+ `<id>.gaea-checkpoint.json`（seq + 投影消息）+ `<id>.meta`（分支元数据），恢复 = checkpoint + log tail（`session/log.go:194-217`、`checkpoint.go:36-46,81-121`） | **口径不同且更分散**。gaea 的 checkpoint 已有 `Seq` 游标（`checkpoint.go:21-32`），语义上等价于 codex 的投影检查点，但**缺字节偏移**，所以无法做"半行/跳过行"级别的续跑 |
| G5 | **写锁** | 三层：in-process Mutex + lifecycle RwLock + 跨进程文件锁 + SQLite IMMEDIATE（`local/mod.rs:176-230`、`writer_lock.rs`、`busy_retry.rs`） | 只有**进程内** `sync.Mutex`（`session/log.go:221-232` `LogWriter.mu`）；`chat.db` 靠 `SetMaxOpenConns(1)` + `_busy_timeout=5000` 串行（`internal/chat/db.go:40,35`） | **gaea 缺跨进程锁**。多实例/多进程同开同一会话目录时无互斥 |
| G6 | **迁移与回滚** | 迁移有 journal + 投影校验 + 发布 CAS；回滚三件套分工明确（§6.3） | 迁移是单向 legacy→event 日志（`controller_session.go:155-161` `MigrateLegacyToLog`），无 journal、无发布 CAS、无回滚计划；分叉只写 `BranchMeta.ParentID/ForkTurn/ForkMessageIndex` 三个整数（`session/branch.go:18-29`） | **gaea 缺**。`ForkTurn`/`ForkMessageIndex` 是**逻辑**切点，父会话文件若被压缩/重写会漂移（codex 用 `HistoryPosition{ordinal,byte}` 物理切点解决同一问题） |
| G7 | **已有且值得保留** | — | `checkpoint.Seq` + `ReadLogRepaired`（torn-tail 截断）+ `BalanceEntries` 合成 closers + `Space` 自描述对账 fail-closed（`checkpoint.go:68-121`、`log.go:353-391`） | **gaea 已有**：torn-tail 修复与空间穿越守卫是 codex 侧没有对应物的能力，反向可抄 |

**小结一句**：gaea 在「单进程、单会话、可容忍全量解码」的假设下自洽；codex 那套是为「多进程并发 + 十万级 turn + 需要按页交付」长出来的。要抄的是**检查点游标化**与**页回收**，不是整副骨架。

---

## 10. 施工卡（2–3 刀）

### 刀 A · 会话投影检查点：把 `Seq` 游标升级成 `(Seq, ByteOffset)` + 幂等投影表

**目标**：消灭"面板刷新全量解码 `.jsonl`"（`save.go:303-306` 自陈的现状）。

1. 新建 `<id>.gaea-projection.json`（或独立 `session_1.sqlite`）：
   `{ "nextByteOffset": u64, "nextSeq": i64, "logPath": "...", "logLen": u64, "logMtimeNs": i64 }`。
   **抄 codex 的双游标**：`next_rollout_byte_offset` + `next_rollout_ordinal`（`thread_history_migrations/0001_thread_history.sql:34-38`）。
2. 物化表照 `thread_turns`/`thread_items` 骨架：`(session_id, ordinal)` 复合主键 + `UNIQUE(session_id, ordinal)` 索引（同上 `:15-32`）。
3. 投影推进必须在**单事务内**先读游标、比对 `expected == start`、再写行、再更新游标；不等即返回错误（抄 `thread_history.rs:116-148`）。
4. 写路径顺序照抄 `live_writer.rs`：**`.jsonl`/`.gaea-log.jsonl` 先 flush，投影后追且允许失败**（`live_writer.rs:436-452`）。失败只 `slog.Warn`，下次从游标续跑。
5. 恢复路径：`Restore` 先按游标取 checkpoint，再投影 `Seq` 之后的 tail（保留现有 `checkpoint.go:81-121` 行为，只把"读全文件算 tail"换成"从 byteOffset 起开文件"）。

**验收**：杀掉进程在投影中途，重启后 `nextByteOffset` 严格等于最后一条**完整**行的结尾字节，且不重复计入任何 ordinal。

### 刀 B · 页回收：给 `chat.db` 与新增的 `session_1.sqlite` 上预算化回收

直接照 `reclamation.rs` 抄，注意三条硬约束：

1. **选主**：`<dataDir>/.gaea-maintenance.lock`，`try_lock` 抢（`reclamation.rs:122-137`）；`WouldBlock` 就让路 60 s。
2. **门槛**：`journal_mode==wal && auto_vacuum==2 && free*page_size >= 64MiB && free >= pages/4`（`:324`）。`auto_vacuum` 必须在**建库时**就设 `2`——存量库需要 `PRAGMA auto_vacuum=2; VACUUM;` 一次性迁移，这是最容易漏的一步。
3. **预算**：`PASS_PAGES=1024` / `BATCH_PAGES=64` / `PASS_DURATION=100ms`；每批 `BEGIN IMMEDIATE` 内重读 `freelist_count` 判储备，提交后 `wal_checkpoint(PASSIVE)` + `data_version` 双检（`:338-379`）；失速减半 + `500ms*2^(n-1)` 退避（`:168-192`）。
4. **NFS 跳过**，且**不要**给 `session_1.sqlite` 开 `background_reclamation`（沿用 codex 的判断：只有"先写后读"的事务安全，`sqlite.rs:96`）。

**验收**：见 §11 第 8–10 条。

### 刀 C · 跨进程写锁 + 陈旧接管（Windows 关文件再删）

1. 目录 `<sessionDir>/.writer-locks/`，文件名 `<branchID>.lock`（对齐 `writer_lock.rs:17`、`WRITER_LOCK_DIR`）。
2. `acquire`：先 `flock`/`LockFileEx` 协调锁 `.coordination.lock` → 一次性 `remove_stale_thread_locks` → 开/建 thread 锁 → `try_lock`，`WouldBlock` 映射成现有 `Conflict` 语义（`writer_lock.rs:43-86`）。
3. **陈旧判据 = "我能 try_lock 它"**，不是看 mtime、不是看 PID（`writer_lock.rs:149-160`）。
4. Windows 释放顺序：先关 `*os.File` 再 `os.Remove`（`writer_lock.rs:184-193`）。
5. 与刀 A 的锁序：`lifecycle`（fork 共享租约）先于 `writer`（`local/mod.rs:186-191`），Go 侧可用 `sync.RWMutex` 对应。

**不要**在本刀里做 codex 的 `lifecycle` 全套：gaea 没有 fork 共享前缀语义（分叉是逻辑切点），先只上 writer 锁即可。

---

## 11. 验收断言（≥8 条）

| # | 断言 | 判据来源（codex 行为） |
|---|---|---|
| A1 | **分页边界·不重不漏**：连续用 `next_cursor` 翻到底，所有行的 `rolloutOrdinal` 全局严格单调且无重复；`next_cursor == nil` 当且仅当最后一页行数 `< page_size` | `segment_paging.rs:504,517-530`（`rows.len() > page_size` 才给 next） |
| A2 | **分页边界·锚点语义**：`next_cursor` 排他（`>`/`<`），`backwards_cursor` 含锚（`>=`/`<=`）；用 backwards 翻回必须能取回该锚行 | `segment_paging.rs:463-473`、`498-516` |
| A3 | **分页边界·跨段不串**：cursor 的 `rolloutOrdinal` 落在 lineage 之外时必须报 `invalid cursor`（"position outside thread lineage"），不得静默返回空页 | `segment_paging.rs:415-421` |
| A4 | **游标防串**：把线程 A 的 cursor 喂给线程 B、或把 turns scope 的 cursor 喂给 items 接口，都必须报错而非降级 | `read.rs:284-289`（同时校验 threadId + scope） |
| A5 | **跨段去重**：构造 fork 场景，同一 `turn_id` 在父段与子段都存在且子段 ordinal 更大，`list_turns` 只能返回子段那条 | `segment_paging.rs:85-97` |
| A6 | **迁移幂等（重跑）**：对同一批 rollout 连续跑两次 `Apply`，第二次全部落 `AlreadyPaginated`（或 `Migrated` via recovery），投影 `(byteOffset, ordinal)` 与文件长度、ordinal 总数完全一致，且不产生重复 turn/item 行 | `rollout_migration.rs:503-548,787-806` |
| A7 | **迁移续跑**：在"已 rename 覆盖、journal 未删"处强杀；重启后必须走 `recover_published_migration` 路径补投影并清 staged，最终 `journal` 不存在、`history_mode == Paginated` | `rollout_migration.rs:496-534,1019-1074` |
| A8 | **迁移不误删**：在"未发布"阶段强杀后跑清理，源文件仍是 legacy 内容且可读（清理只删投影与 staged，不删源） | `rollout_migration.rs:1076-1097`（先确认 `history_mode != Paginated` 才清） |
| A9 | **回滚可重放**：取一个含 `ThreadRolledBack` 的 legacy rollout 迁移后，用老版 cold-resume 语义（反向扫 checkpoint）算出的模型上下文起点，与新文件的 `Compacted` 起点一致；若被回滚的正是那个 checkpoint 段，必须留下一个**空 replacement_history** 的 `Compacted` 占位 | `rollback_replay.rs:161-172`、`rollback_plan.rs:86-91` |
| A10 | **revert 不毁旧文件 + CAS**：revert 后旧 rollout 文件仍存在且内容逐字节不变；并发再 revert 一次必须 `Conflict` 且新产生的文件被删除 | `revert_thread.rs:15-18,135-150`、`runtime/threads.rs:415-429` |
| A11 | **回收选主**：同机起两个实例，同一 home 下同一时刻只有一个执行 `incremental_vacuum`（另一实例的 pass 报 `Idle` 且 `pages=0`）；owner 退出后另一实例在 ≤ 60 s 内接管 | `reclamation.rs:87-98,122-137` |
| A12 | **回收门槛与储备**：`free*page_size < 64 MiB` 或 `free < pages/4` 或 `auto_vacuum != 2` 或非 WAL 时，pass 必须 `Idle` 且 `pages == 0`；任何一次回收后剩余 free 页必须 `> reserve`（`max(16MiB/page_size, pages/10)`） | `reclamation.rs:322-333,346-354` |
| A13 | **回收退避**：连续两次 `Interrupted && pages==0` 后 `batch_pages` 必须减半且下一轮间隔 ≥ 500 ms；一旦有进展，`batch_pages` 保持减小后的值不回升 | `reclamation.rs:168-192` |
| A14 | **NFS 不连库**：把 sqlite home 指到 NFS 挂载，`reclaim` 必须直接返回 `Idle` 且**不创建**维护连接（可用连接计数/文件句柄断言） | `reclamation.rs:221-233`、`sqlite.rs:220-227` |
| A15 | **陈旧锁接管**：留一个孤儿 `.lock` 文件（无持有进程），新进程 `acquire` 必须成功且该文件被删除；若另有一进程真正持锁，`acquire` 必须 `WouldBlock`→`Conflict` 且**不删**文件 | `writer_lock.rs:65-79,149-168` |
| A16 | **Windows 释放顺序**：在 Windows 上反复 acquire/release 10k 次不得出现"文件被占用无法删除"，且 `thread-writer-locks/` 最终为空 | `writer_lock.rs:184-193` |

---

## 12. 不搬 / 风险

**不搬**

1. **`agent_jobs` 那套**（`0042_drop_agent_jobs.sql` 已删表）——同一个 crate 里留过 3 次"先建后删"（`0006_memories`→`0035_drop_memory_tables`、`0014_agent_jobs`→`0042_drop_agent_jobs`、`0029_thread_goals`→`0034_drop_thread_goals`）。**教训是别抄具体表**，抄"独立库 + 迁移版本化"这个纪律。
2. **`legacy` 与 `Paginated` 双模式并存**。codex 为兼容老 rollout 付了全框架代价：`reject_paginated_history`、`canonical_history_mode_from_rollout_items`、`stored_thread_from_meta_line` 三条并行路径（`read_thread.rs:214-219`、`types.rs:143-155`、`:532-560`）。gaea 目前只有一条路径，**不要**为了互操作先造双模式。
3. **`thread_attachments`/`projects`/`thread_sections`/`thread_section_order`**：与本资产的"线程存储"正交，属 UI 组织层，本期不碰（`state/src/runtime/thread_section_order.rs`、`thread_attachments.rs` 均未细读）。
4. **7 库拆分**。codex 拆 7 个 SQLite 是因为"单库损坏不该带走全部"（`recovery.rs:1-8` 注释）。gaea 现在 1 个 `chat.db`，拆库收益不抵迁移成本；先只加 1 个 `session_1.sqlite`。

**风险**

| 风险 | 说明 | 缓解 |
|---|---|---|
| R-1 双写顺序被写反 | 一旦投影先于 JSONL/日志提交，崩溃后 DB 声称有、文件没有，任何"从文件重建"都会丢数据 | 把顺序做成**不可绕过的单一函数**（抄 `write_and_project`），并在投影入口加 `expected == actual` 断言（`thread_history.rs:132-136`） |
| R-2 `auto_vacuum` 存量库迁移 | `auto_vacuum` 只能在建库或 `VACUUM` 时改；存量 `chat.db` 需一次性 `VACUUM`，大库会锁库 | 迁移放启动期后台任务 + 单次 journal，失败保持 `mode != 2` 自然退化为 Idle（不会坏，只是不回收） |
| R-3 `live_writer_locks` 无界增长 | codex 自己也承认不回收（`local/mod.rs:178-180`）；Go 侧若照抄 `map[threadID]*Mutex` 会随会话数增长 | 用 `sync.Map` + 引用计数，或接受并设上限告警 |
| R-4 跨平台文件锁差异 | `try_lock` 在 Windows 上是强制锁且需先关文件才能删；Linux 上 advisory | 抽一层 `lockfile` 接口，把"关→删"顺序封在 `Release()` 里（抄 `writer_lock.rs:184-193`） |
| R-5 `history_revision` 在 Windows 上变弱 | unix 有 `(dev,ino,ctime,ctime_nsec)`，非 unix 恒 `None`（`history_revision.rs:15-26`） | gaea 在 Windows 为主：改用 `(path, size, mtime, created)` + 内容首行 hash 兜底 |
| R-6 迁移"两遍写"的 CPU 代价 | 带回滚标记的 rollout 要读 3 遍（plan 一遍 + 写两遍，`rollout_migration.rs:857-899`） | 只在检测到 `ThreadRolledBack` 时进两遍路径（现状即如此），并保留 `max_mib_per_second` 限速 |
| R-7 页回收与前台写的 `SQLITE_BUSY_SNAPSHOT` | 回收提交会让"先读后升写"的事务拿到快照冲突 | 只对"先写后读"的库开启（`sqlite.rs:63-64,96`）；`session_1.sqlite` 的写路径必须保证先写 |

---

## 13. 证据校验（命令 + 未读清单）

### 13.1 复核命令（只读，PowerShell）

```powershell
$R = 'C:\AI\re\codex-0.165.0\oss\codex-0.165.0\codex-rs'
$G = 'C:\AI\wubigrok'

# 1) 权威源声明 + "允许滞后禁止超前" 那条注释
Select-String -Path "$R\thread-store\src\local\mod.rs"              -Pattern 'durable replay format|queryable metadata index'
Select-String -Path "$R\thread-store\src\local\live_writer.rs"      -Pattern 'rebuildable view|never get ahead'

# 2) 投影游标 schema 与"投影不得超前"守卫
Get-Content "$R\state\thread_history_migrations\0001_thread_history.sql"
Select-String -Path "$R\thread-store\src\local\thread_history.rs"   -Pattern 'is behind durable rollout|next_rollout_byte_offset'

# 3) 分页游标形状与校验
Select-String -Path "$R\thread-store\src\local\thread_history\read.rs" -Pattern 'enum CursorScope|struct HistoryCursor|requested_thread_id != requested_thread_id|scope != scope'

# 4) 分段结构与跨段去重谓词
Select-String -Path "$R\thread-store\src\local\rollout_lineage.rs" -Pattern 'struct RolloutLineageSegment|history_base|cycle detected'
Select-String -Path "$R\thread-store\src\local\thread_history\segment_paging.rs" -Pattern 'NOT EXISTS|push_segment_range|include_anchor'

# 5) 回滚三件套的分工
Get-Content "$R\thread-store\src\local\rollout_migration\rollback_plan.rs" -TotalCount 10
Get-Content "$R\thread-store\src\local\rollout_migration\rollback_replay.rs" -TotalCount 8

# 6) revert 的"旧文件不动 + 指针 CAS"
Select-String -Path "$R\thread-store\src\local\revert_thread.rs" -Pattern 'Old rollouts stay intact|replace_rollout_path_if_current'
Select-String -Path "$R\state\src\runtime\threads.rs"            -Pattern 'rollout_path = \? WHERE id = \? AND rollout_path = \?'

# 7) 页回收常量全组（应输出 8 行 const）
Select-String -Path "$R\state\src\runtime\reclamation.rs" -Pattern '^const '
Select-String -Path "$R\state\src\runtime\reclamation.rs" -Pattern 'wal|auto_vacuum|MIN_FREE_BYTES|RESERVE_BYTES / page_size'
Select-String -Path "$R\state\src\runtime\reclamation.rs" -Pattern 'sqlite-maintenance.lock|try_ownership|is_nfs'

# 8) 陈旧锁接管判据
Select-String -Path "$R\rollout\src\writer_lock.rs" -Pattern 'WRITER_LOCK_DIR|COORDINATION_LOCK_FILE|remove_stale_thread_locks|try_lock|Windows too'

# 9) gaea 侧对照（注意 internal/gaea/chat 不存在）
Test-Path "$G\internal\gaea\chat"          # 期望 False
Select-String -Path "$G\internal\chat\db.go"     -Pattern 'chat_messages|_journal_mode=WAL|wal_checkpoint|SetMaxOpenConns'
Select-String -Path "$G\internal\chat\store.go"  -Pattern 'ListMessagesPage|seq < \?'
Select-String -Path "$G\internal\gaea\agent\session\save.go" -Pattern '全量解码|上限 64'
Select-String -Path "$G\internal\gaea\agent\session\log.go"  -Pattern 'LogWriter|gaea-log'
```

> 上文每条 `file:line` 均已在写作过程中用 `Get-Content <file> | Select-Object -Skip <n-1> -First <k>` 逐行核对过原文（含 codex 的 18 处、gaea 的 10 处）。

### 13.2 未读 / 未细读清单（禁止据此猜测结论）

**thread-store（未读）**
- `src/in_memory.rs`、`src/error.rs`、`src/thread_attachments.rs`、`src/projects.rs`、`src/thread_sections.rs`
- `src/local/create_thread.rs`、`delete_thread.rs`（**仅 grep 到符号与行号**，未逐行读）、`helpers.rs`、`model_context.rs`、`move_thread_to_section.rs`、`paginated_fork.rs`、`pending_thread_metadata.rs`、`thread_rollout_resolver.rs`、`thread_sections.rs`、`unarchive_thread.rs`
- `src/local/thread_history/{realtime.rs,search.rs,turn_lookup.rs}`（仅读 `read.rs` 与 `segment_paging.rs`）
- `src/local/thread_history.rs` 仅读 1–150 行（共 580 行，`apply_change_set` 之后未读）
- `src/local/read_thread.rs` 仅读 1–560 行（共 1696 行）
- `src/local/list_threads.rs` 仅读 1–330 行（共 808 行）
- `src/local/update_thread_metadata.rs` 仅读 1–130 行（共 2475 行）——**"patch 逐字段 literal 应用"的细节未逐行读**
- `src/local/rollout_migration.rs` 读 ≈60%（未读 1120–1419 的收尾与测试段）
- `src/local/rollout_migration/{startup.rs, rollback.rs, canonicalizer.rs, line_parser.rs, publish.rs, subagent.rs, telemetry.rs}` ——**只通过调用点推断职责，未读实现**
- `src/local/rollout_migration/rollback_plan.rs` 仅读 1–200 行（共 613 行）
- `src/local/archive_thread.rs` 仅读 1–180 行（共 371 行）

**state（未读）**
- `src/extract.rs`（36 KB，`apply_rollout_item` 的真实实现——**元数据抽取规则未读**）
- `src/runtime/{memories.rs, goals.rs, queued_items.rs, thread_section_order.rs, logs.rs, logs_maintenance.rs, projects.rs, thread_attachments.rs, thread_read_state.rs, thread_sections.rs, thread_metadata.rs, rollout_migration.rs, remote_control.rs, guardian_feedback.rs, memory_versions.rs, memory_readiness.rs, external_agent_config_imports.rs}`
- `src/runtime/threads.rs` 3932 行中仅读约 400 行（`get_thread`/`replace_rollout_path_if_current`/`list_threads`/`list_threads_matching`/`push_thread_filters`/`push_thread_order_and_limit`/`ThreadFilterOptions`）
- `src/model/{backfill_state.rs, graph.rs, log.rs, memories.rs, project.rs, queued_item.rs, rollout_migration_state.rs, thread_attachment.rs, thread_goal.rs}`；`src/model/thread_metadata.rs` 仅读 1–120 行（共 910 行）
- `src/sqlite/validation.rs`（quick-check 管理器）、`src/sqlite.rs` 仅读 1–300 行（共 617 行，`open_read_write_pool` 之后未读）
- `src/{log_db.rs(仅常量), telemetry.rs, delegated_preview.rs, migrations.rs(已读全)}`

**rollout（未读）**
- `src/{recorder.rs, state_db.rs, list.rs, list_files.rs, maintenance.rs, compression.rs, metadata.rs, ordinal.rs, policy.rs, search.rs, session_index.rs, rollout_reference_index.rs}` ——`RolloutRecorder` 的内部缓冲/flush 语义、`list_threads` 的 SQL、`parse_cursor` 的编码格式均**未读实现**，本文件仅按调用点与返回类型描述

**gaea（未读/节选）**
- `internal/gaea/control/controller.go` 1186 行中仅读 650–729
- `internal/gaea/agent/session/{log.go(节选), save.go(节选), migrate.go, rewind.go, projection.go, sink.go, attention.go, derived.go, state.go, space.go, subagent_anchor.go}` 多数未读
- `internal/chat/store.go` 仅读 368–422 与 grep 结果；`branches.go` 仅读 1–60
- 未核实 `chat.db` 的真实表结构是否与 `SchemaV1` 字面一致（**按要求未打开任何真实数据库文件**）

**结论边界**：本文件对 `RolloutRecorder` 内部、`extract.rs` 的元数据抽取规则、`apply_change_set` 的写入细节**不作断言**；凡涉及之处均标注为"按调用契约"而非"按实现"。
