# 第七轮修复记录（P1 批次五：并发收尾）· 2026-10-02

> **承接**：批次一~四（12aeaf6d / 9986bb9d / 532f5444 / 3bdd1922）。
> **验证**：`go build` 全绿、`go vet ./...` 0、`internal/app` 全包 + agent 族 + visual 测试全绿、gofmt 干净。

## 已落地（4 条 + 1 条复核深挖出的真竞态）

### AP3-04 · 两个设置绑定补 ga.mu
`GaeaSetMemoryBrief` / `GaeaSetMorningPreload` 裸调 `gaeaRebuildLocked`（注释明写「调用方必须已持有 ga.mu」）补锁。**修法偏离审计建议并留档**：不改走 `gaeaApplyCfg`——那两个开关持久化在 `appconfig`（~/.gaea_config.json），`gaeaApplyCfg` 写的是 gaea.toml，照抄会改变持久化落点。

### AP4-05 · 续跑受理显式复位 skipped
`GaeaDagRun` 在选待跑集之前把 skipped 归一 pending（清 error）并落盘——失败级联置 skipped 的节点在「续跑=重试」语义下获得显式复位动作，状态机与派生一致（此前只是恰好落进待跑集）。

### IN3-08 · OCR 四层降级链收集真实错误
`GaeaOCRText` 逐层 `errors.Join`——此前只判 err==nil 从不保留 err，四层全挂时只回最后一层的错误，排障看不到前三层为何失败。层内文案标注通道（指定引擎/herdsman ocr/herdsman parse/本地 OCR）。

### AP7-07 · 视觉逐章扫描区分「没有」与「损坏」
四份复制循环（实际 3+1 处）收敛进 `readSummaryForScan`：文件不存在 → 收循环；解析/读取失败 → slog.Warn + 跳过该章继续——此前一律 break，中间某章 summary.json 损坏即静默截断全库统计。与 project 包 readChapterSummariesWithFiles 口径对齐。

### GA1-06 · 会话裸写竞态（复核深挖后定性修正）
审计指的「stream 零拷贝读」**符合** Session 锁纪律（run-loop 内串行读豁免，`Save` 已走锁内 Snapshot）——该半条证伪。但复核发现**真竞态**：run loop 自己的三处绕锁写——grace-round 截尾 ×2（`agent_run.go:288/351` 裸缩 Messages 切片头）与 `MergeRuntimePrompt`（就地改 `Messages[0].Content`）——与每秒落盘 ticker 的 `Save`（RLock）构成写读数据竞争。修复：Session 新增 `DropLast()` / `MergeIntoSystem()` 锁内方法，三处全部改道。

## 复核证伪（不修，留档）

- **GA1-05**（asker/permReq 非原子读写）：三个 setter 的唯一生产调用点在 `EnableInteractiveApproval`（gaeaBuildController 内、引擎构建期），先于任何 run loop / pre-exec goroutine 创建，happens-before 链成立，注释契约在生产路径为真。若未来出现运行中重挂 asker 的需求再原子化。
- **GA1-06 的 stream 裸读半条**：见上，读侧纪律本就豁免 run-loop。

## 并发/吞错类收尾状态

至本批，P1 concurrency（23）+ error-swallow（9）共 32 条全部处置完毕：修复 20、随 P0 闭环 2（AP1-03、#4 内）、证伪 3（GA4-05、GA1-05、GA1-06 读半条）、留池需设计 4（AP7-08 core.cfg 快照、AP6-03 PreLLMTurn oplog、AP5-04 待证上游、AP6-08 已修）。后续批次转向 duplication/consistency/god-file 类。
