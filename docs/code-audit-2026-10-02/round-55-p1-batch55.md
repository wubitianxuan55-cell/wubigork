# 全仓审计第 55 批 · A2① 任务身份语义统一（对照表+三包格式钉，零行为变化）· 2026-10-03

> 接续 [round-54（批次五十四）](round-54-p1-batch54.md)。拍板执行 A2①（建议②「先统一任务身份语义，小刀可验」）：**预核实证三套是三种域后，以对照表+格式钉收口语义口径——不合并枚举，零行为变化。不抬版本。**
> 快照：开工时 HEAD = `245a7ebe`（批次五十四）；工作树干净。

---

## 一、预核实证（审计 TOP 榜 #9 的「三套编排」身份面）

| 系统 | ID 格式 | 生成动因 | 状态词汇 | 取消/重试/续跑 |
|---|---|---|---|---|
| 任务调度器（gaea/tasks） | `tsk_`+32hex（rand 失败退 `tsk_<nanos>`） | 长寿命 store，碰撞域大 | queued/running/stopping/succeeded/failed/cancelled | 取消两段式（C1 stopping 中间态）；重试=re-enqueue |
| DAG 流水线（gaea/dag） | `dag_<yyyymmdd_hhmmss>_<seq>` | 每 run 一个 `<id>.json`，时间戳 slug=文件名字典序即时间序 | 节点 pending/hold/running/done/failed/skipped/accepted；run 派生 draft/running/failed/ready/accepted | 取消=级联终止；重试=单节点重跑；续跑=已完成集跳过 |
| 任务收件箱（app/task_inbox） | `ti-`+12hex | 单文件 JSON 数组建议队列，短 id 足够；rand 失败如实报错 | pending/doing/done/abandoned | 无执行语义（仅 CanTransition 状态迁移） |

**判定：三套是三种域（执行引擎/流水线编排/建议队列），词汇差异=语义差异——审计「合并入口」原 fix 维持拒绝，统一落为口径单源。** 跨系统引用走既有命名空间（DAG 节点 Ref→`sa_` 子代理运行、收件箱 Session→会话档），无新增耦合。

## 二、落地

- `gaea/tasks/tasks.go`：Status 上方落**任务身份三系统对照表**（权威注释，三包互引）。
- **三包格式钉**：`TestTaskIDFormatPin`（tsk_+32hex）/`TestDagNewIDFormatPin`（dag_时间戳序） /`TestTaskInboxIDFormatPin`（ti-+12hex）——ID 格式是对照表的机器锚，漂移即红（dag 格式漂移会破坏「文件名字典序=时间序」的隐含存储契约，注释写明）。
- dag/inbox 两生成站点补互引注释（失败语义差异：tasks nanos 兜底 vs inbox 如实报错——队列条目可重试生成）。

## 三、门禁

- gofmt 0 差 / `go build ./...` 绿 / 三包定向绿（tasks/dag/app）。

## 四、余量与下一批

- **A2① 关账**（「合并入口」维持拒绝——等真实痛点再立项）。拍板池余：B1/B2（新功能立项）。
- 观察池：fsync 全量基准、WhisperGraphPanel 真机复走。
- 下一批候选：按需发版（45 批本地未推，tag+源码包归档+README 版本表随发版走）或 WhisperGraphPanel 真机复走。
