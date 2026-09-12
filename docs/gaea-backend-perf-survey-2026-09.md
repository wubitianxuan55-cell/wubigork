# Go 后端性能与资源热点普查（2026-09-12）

> 起因：用户提出「优化 gaea 的后端」。本档为普查证据档，候选一律交拍板池（gaea-next-stage-plan §5），不替用户排刀。
> 方法：全 internal/ 扫描（16.2 万行非测试 Go）+ 抽查核实前 6 项（均属实，file:line 引用为核实后原文）。`go vet ./...` 全净。与既有 docs 条目零重叠（唯一性能挂账均为前端：slim-baseline P4 / AOA 拖拽帧率）。

## 高（每回合/每请求固定成本，随数据量线性放大）

1. **回合边界事件日志全量重读** — `internal/gaea/agent/session/sink.go:119-124`：`turn_done` 落盘后关闭写入器、下事件懒重开；`OpenLog`（`session/log.go:226-259`）每次执行 `RepairLogFile`（全量 ReadFile）+ `countLogLines`（`log.go:398-406` 全量 ReadFile + 逐行 JSON 解析）。影响：每条用户消息 = 2 次全量读 + 全量解析整份 `.gaea-log.jsonl`（含全部 reasoning/text 增量），长会话每回合 O(会话总字节)。
2. **remember 走全局锁内全库重载** — `internal/gaea/control/controller_memory.go:218-223, 241-252`：模型每次 `remember`/quickadd/forget → `QueueMemory` 持控制器全局 `c.mu` 执行 `refreshMemoryLocked` = 完整 `memory.Load`（SELECT 全部 facts 含 body + 全量分词建倒排 + discoverDocs 读所有 AGENTS.md）。影响：一次记忆写入以「全库重载+全局锁」阻塞 Send/Cancel/事件等全部控制器操作。
3. **cost_search 每查询重建 BM25 索引** — `internal/gaea/cost/cost.go:370-374`：每次 `Search` 用 `bm25.NewRanker(docs)` 从零建倒排；专为避免此事写的 `bm25.Cache`（`internal/gaea/bm25/bm25.go`）全仓无生产消费方。另 `cost.go:322` 每行相关子查询 `(SELECT COUNT(*) FROM cost_entry_components ...)` 随库规模放大。
4. **cost 语义工具每次新建 HTTP 客户端** — `internal/gaea/tool/builtin/cost_tools.go:194-221, 257-286`：`costEmbedder()/costReranker()` 每次工具调用 new `retrieval.Embedder/Reranker`（各自新建 `http.Client`+transport），60s `Available()` 探测缓存按实例存、永不命中。影响：每次语义召回/精排多做一次 `/models` 探测且 keep-alive 全失效。
5. **contextview 每次刷新全量重解析** — `internal/app/gaea_ui_contextview.go:33-41`：`GaeaContextView` 每调 `session.ReadEntriesFor` 全量读会话日志 + `fold.go` 逐条 `json.Unmarshal` 重折叠（`GaeaTrajectory`/`GaeaContextNodeDetail` 同模式）。影响：前端每次刷新看板 = 全量 I/O+解析+折叠，会话越长 UI 越卡。
6. **Hephaestus.db 单连接串行化所有读写** — `internal/gaea/db/database.go:51`：`SetMaxOpenConns(1)` 注释写「串行写入最佳实践」但把读也串到单连接。长查询/写事务期间全部读写互阻塞；仓内他处已留该模式死锁注脚（`internal/characterlib/portrait.go:244`、`internal/whisper/db/repos/fts.go:25`）。

## 中（面板刷新/批量写/无界增长）

7. **会话列表预览全量解析** — `internal/gaea/agent/session/save.go:266-292`：`previewSession` 对每个 `.jsonl` 解码全部消息（含多 MB 工具输出）只为取首条预览+轮次数。会话面板每次刷新 = O(所有会话总字节)。
8. **记忆索引搬运全量 body** — `internal/gaea/memory/sqlite.go:282-289` + `memory/search.go:49-87`：`listInSpace`/`Index()` SELECT 每条 fact 完整 body（注入索引只用 title/description），`BuildSearchIndex` 再全量分词 body。会话装配与每次记忆写后重载都搬运+分词整个记忆库正文。
9. **并行子代理收尾串行落盘** — `internal/gaea/agent/subagent_store.go:115, 241-257` + `session/save.go:25-39`：`Session.Save` 每次全量重写 transcript JSONL，N 路并行子代理 `SaveCompleted` 全串行在同一把 `SubagentStore.mu` 上。
10. **wssearch 文本缓存无界** — `internal/gaea/wssearch/wssearch.go:86`：全局 `textCache` 每文件最多 20 万字符，只增不删无上限无过期，长会话/大工作区内存无界增长。
11. **批量清理逐条 DELETE 无事务** — `internal/gaea/memory/sqlite.go:426-432`（`CleanupArchived`）、`internal/gaea/semantic/semantic.go:270-274`（`Stale`）。N 行 = N 次独立写事务（WAL 下 N 次 commit/fsync）。
12. **记忆 Save 未包事务** — `internal/gaea/memory/sqlite.go:71-98`：facts UPSERT 与 `memory_events.AppendEvent` 拆两条独立语句；dream 批量写（`controller_memory.go:309-331` 循环 Save）成倍放大。
13. **grep 工具单线程全量扫** — `internal/gaea/tool/builtin/grep.go:103-131`：单 goroutine `filepath.WalkDir` 串行 ReadFile+编码转换再逐行匹配，无 worker 池、大文件不跳过。
14. **每步调用全量消息摘要** — `internal/gaea/agent/compile.go:34-63`：`DigestMessages` 每次把全部消息正文 append 进 buffer 再 SHA256（`stream()` 每步都调，`agent_stream.go:32`）。前缀稳定时本可增量只哈希新增消息。
15. **bm25.Cache 持锁建索引** — `internal/gaea/bm25/bm25.go:41-50`：`Get` 持全局锁执行 `build()`，慢 build 阻塞其他 key 并发取用；map 无淘汰。
16. **图谱 linker 循环内编译正则** — `internal/graph/linker.go:129-142`：`FindUnlinkedMentions` 对每个实体每次调用 `regexp.MustCompile` 两次，实体多时 O(实体数) 重编译。
17. **归档分页退化为全表扫描** — `internal/gaea/memory/file_backend.go:231-241`：`ListArchivedPaged` 返回一页（≤200 条）先 `ListArchived()` 全量 ReadDir+逐文件解析。

## 低（顺手修，单点影响小）

18. `internal/app/platform_handler.go:72-73` 循环 `result += string(r)` O(n²) 拼接，换 `strings.Builder`。
19. `internal/gaea/retrieval/semantic.go:22-31` `SemanticRecall` 每查询全量重 embedding 候选库——已被持久化向量路径取代、现无生产调用方，属遗留危险实现，建议删除或修复防误接线。
20. `internal/app/memory_hub.go:427`、`internal/app/gaea_export.go:323`、`internal/app/shelf.go:371` 函数体内 `regexp.MustCompile`，UI 交互路径重复编译，提为包级变量即可。

## Benchmark 现状

全仓仅 6 个 Benchmark（docmd 大文件转换 ×3、whisper ×3），与 gaea/ai 热路径无关；`internal/gaea`、`internal/ai`、`internal/app` 零 benchmark。首批立项应先立 benchmark 锁基线再动刀（#1/#3/#5/#7/#8 均可直接立）。

## 建议刀路（待拍板，不替用户排期）

- **刀 A（每回合税）**：#1 事件日志 seq 增量化（RepairLogFile/countLogLines 只跑会话首开或改增量校验）+ #14 DigestMessages 增量哈希。零外部行为变化，benchmark 锁基线。
- **刀 B（记忆写路径）**：#2 QueueMemory 改后台刷新/增量索引 + #12 Save 包事务 + #8 索引不 SELECT body。动全局锁语义，须真机走查。
- **刀 C（SQLite 口径）**：#6 读写连接分离（WAL 下读写不互阻）+ #11/#12 批量事务化。风险最高（仓内留有死锁注脚），单独一刀+回归。
- **刀 D（cost 检索）**：#3 接通 bm25.Cache（本来就为此写的）+ #4 Embedder/Reranker 单例化 + 子查询消除。
- **刀 E（面板/杂项）**：#5 contextview 缓存/增量 + #7 预览采样 + #10 缓存上限 + #16/#17/#18/#20 顺手项。
