# 第五轮修复记录（P1 批次三：结构小刀）· 2026-10-02

> **承接**：批次一（12aeaf6d）、批次二（9986bb9d）。
> **验证**：`go build` 全绿、`go vet` 0（含 copylocks）、project/jobs/agent 三包 `go test -count=1` 全绿。

## 已落地（2 条）

### IN1-07 · project.Manager 互斥下沉
Manager 新增 `fileMu sync.Mutex`，`UpsertAnalysisV2` / `syncRewriteIndex` 两条「读 JSON→改→writeJSON」整表覆盖链全程持锁——并发时后写者用旧快照覆盖先写者丢更新。读路径不持锁（原子替换下读到旧或新，无半截文件）。
**范围说明**：审计 fix 文中的「app 侧 chapterPlanMu 随之删除」本轮不做——该锁护的是 app 层跨「读表→计划契约闸门→写回」的复合临界区，收编需 Manager 提供 `UpdateChapterPlans(fn)` 式回调 API，属独立小刀。

### GA1-08 · 后台任务 ctx 值继承
- `jobs.Manager.StartInheriting(caller, …)`：job ctx 用 `detachedValues`（Value 委托 caller、Done/Err/Deadline 留在 root）继承调用方 value 链——空间/trace/evidence/memory 队列随代传递，caller 被取消不连带后台任务。
- `task.go` 后台子代理派生改用 StartInheriting，删除只补了 space 一个字段的逐字段补注。
- 回归锁：`TestStartInheritingValuesNotCancellation`（值继承 + jobIDKey 不被 caller 链遮蔽 + caller 取消不连带）。

## 留池（需设计先行）

- **AP8-01**（a.client 无锁替换）：足迹 ~35 生产点 + 全部测试构造器，atomic.Pointer 改造独立一刀。
- **AP6-03**（wx_agent 失败整份回滚）：回滚防双计数 与 并发回合保护 相冲突，正确解是 PreLLMTurn 记录可逆增量（oplog/版本化）——whisper-core 设计刀。
- AP5-04（追问失败无前端信号，审计自标「是否被上游堵住待证」）、AP6-08（crypto/rand 吞错，小刀可并入下批）、FE2-08/FE7-11（前端批）。
