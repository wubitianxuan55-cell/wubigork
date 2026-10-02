# 第四轮修复记录（P1 批次二：并发续）· 2026-10-02

> **承接**：round-3 批次一（12aeaf6d）。同口径继续 concurrency 类。
> **验证**：`go build` 全绿、`go vet ./...` 0、whisper/app/plugin 包定向测试全绿。

## 已落地（6 条）

### AP5-03 · 追问单飞令牌随工作结束释放
`followUpClaims.Delete` 从绑定方法 defer（受理即释放）移入工作 goroutine 的 defer——单飞单元与工作单元对齐，同 ref 第二枪在工作期间仍被拒。

### AP6-09 · clawbot 先登记再 Start
`weixinServers[ast.ID]` 先入表再 `Start()`（Start 内部已起 pollLoop/notifyStart，后登记窗口内 stopAssistantWx/Shutdown 遍历不到 = 孤儿轮询继续收消息）；失败路径摘除登记并 `Stop()`（幂等，未启动安全）。

### AP7-09 · ComfyUI 拉起/恢复等待接取消
`ensureComfyUIRunning`/`recoverComfyUI` 加 ctx 参数，轮询 sleep 换 `sleepCtx`（timer+select）——生成取消（用户点取消/切图）即刻放弃等待，不再替已取消的请求白等最长 120s。两处调用点传 `genCtx`。

### IN4-02 · 情绪涌现双状态源删除死的那支
包级 `recentEventTypes/consecutive*Count` 三变量、6 个 Push*/Get* 函数、`ResetEmergenceTracking` 及 orchestrator.go 调用点全部删除——真实状态只在 Orchestrator 实例字段（PreLLMTurn 只写实例字段，全局支零生产写入）。纯函数 `CountMeaningfulInRecent`（有测试消费）保留。

### AP6-04 · 提醒回推移出锁外
`tickReminders` 重构：锁内只取到期快照 → 锁外推送（apiPost 最长 20s，持锁推送卡死列表/新增/删除绑定）→ 按 id 回写结果 → 成功失败都落盘（原实现只有 pushed>0 才落盘，失败计数在本次运行内永不落）。快照-回写窗口内被删改的条目按 id 找不到即丢弃，不复活。

### GA4-10 · stdio notify 受 ctx 约束
`notify` 的写入放入独立 goroutine + select ctx；新增 `writeMu` 串行化全部 stdin 写入——被 ctx 弃单的 write goroutine 最多存活到子进程退出，期间后续写入排队、JSON-RPC 帧不撕裂（原实现对端不读时管道写满永久阻塞）。

## 本批未动（留池）

AP8-01（client 锁收敛，30+ 读点全改，面宽单独立刀）、AP6-03（wx_agent 失败回滚，需 oplog 设计）、IN1-07（project Manager 互斥下沉，需防嵌套死锁设计）、GA1-08（StartInheriting 值继承，需 custom Context 设计）、AP5-04（审计自标待证）、AP6-08、IN4-07 余项、FE2-08/FE7-11（前端批合并处理）、GA1-05/06、GA4-10 已清。
