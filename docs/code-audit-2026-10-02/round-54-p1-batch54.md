# 全仓审计第 54 批 · A7 fsync 决策落地（AtomicWriteSync 变体 + 两站点收编单源）· 2026-10-03

> 接续 [round-53（批次五十三）](round-53-p1-batch53.md)。拍板执行 A7（按建议②的保守解读）：**AtomicWrite 增断电级变体 `AtomicWriteSync`，默认 AtomicWrite 行为零变化**；两处显式 Sync 站点收编单源。零行为变化申报（唯一变化=错误文案合并），**不抬版本**。
> 快照：开工时 HEAD = `54a2006d`（批次五十三）；工作树干净。

---

## 一、决策与落地

- **决策口径**（brief 建议②的保守解读）：fsync 能力进 fileutil 单源但**默认关**——「全量开」的性能影响需真机基准数据，未经基准不引入默认行为变化；需要断电级 durability 的站点显式选 `AtomicWriteSync`。
- **`fileutil/atomic.go`**：`AtomicWrite` / `AtomicWriteSync` 双入口收拢到私有 `atomicWrite(path, data, perm, sync bool)`；Sync 在 write 后 close 前执行（与两站点原时序一致）。
- **两站点收编**（各 −20 行手搓）：
  - `gaea_benchmark.go`（Herdsman 基准报告）→ `AtomicWriteSync(path, md, 0o600)`；权限 0600 与 CreateTemp 历史口径一致。
  - `gaea_ui_extra.go`（WriteFile 编辑主链）→ `AtomicWriteSync(abs, content, 0o600)`；失败自清理临时件语义与原 defer 等价。
  - 原两处「刻意不并入」拍板注记改写为「已收编」。
- **测试**：`TestAtomicWriteSyncSmoke`——Sync 分支全平台执行覆盖（Windows 也走），权限断言仅 POSIX；文件内容断言防退化。
- **预核修正**：审计 D21 点名的「saveConfigFile 第二份 CreateTemp+Sync+Chmod」实测已不存在——config `Save` 早已走 AtomicWrite（移动靶复核再证）。今日全仓 fsync 写路径仅剩两处且均已收编。

## 二、门禁

- gofmt 0 差 / `go build ./...` 绿 / vet 绿 / fileutil 全套绿（含新烟测）/ **app 全包 136s 绿**。

## 三、fsync 决策终态

- AtomicWrite（默认，无 fsync）：会话/配置/记忆等高频持久化主链——性能优先。
- AtomicWriteSync（显式）：基准报告、编辑器写盘等低频高价值站点——断电级 durability。
- 「全量默认开 fsync」若未来要做，需先补真机写路径基准（登记观察池：fsync 基准测试）。

## 四、余量与下一批

- 拍板池余：**A2**（AP4-01 三套任务身份统一——需先预核 DAG/任务调度器/收件箱的身份语义，独立批）、**B1/B2**（新功能立项）。
- 下一批候选：A2 预核+小刀，或 WhisperGraphPanel 真机复走，或按需发版（44 批本地未推）。
