# 全仓审计第 24 批 · god-func 大拆第三刀（AP2-01 runSinStream 三段拆）· 2026-10-03

> 接续 [round-25（批次二十三）](round-25-p1-batch23.md)。**单线主代理直做**：AP2-01 `internal/app/sin_handler.go` `runSinStream` 265 行串六职责 → 「装配 / 循环 / 收尾」三段拆（审计修法落地）。
> 本批**不抬版本**、不动 CHANGELOG；交付物 = 本文件 + `.gaea/progress.md` 一条。
> 快照：开工时 HEAD = `341da99d`（README 对账小提交）；工作树干净。

---

## 〇、预核与刀深判定

- 审计指认复核成立（行号已漂移：现 268–532）：六职责=历史窗口装配/附件展开/play 护栏钳制/最多 5 轮工具循环/取消·失败·空回复三条收尾分支+落库回写/插图产物回写/成本估算与事件下发；循环内可变态 8 个（messages/reply/reasoning/usage/trace/round/useTools/budget/nudged）。
- **刀深判定**：非 switch 型，批 22/23 的表驱动配方不适用；按审计建议「先搬行不改语义」三段拆。与表驱动化的关键差异=循环内两条 inline 收尾出口（整轮失败零正文→emit+仅用户落库+return）——**abort 哨兵上提**到分派层执行，帧序与原实现逐帧一致。
- **测试网盘点**：sin 族 20+ 用例全走 `SinStream` 公共入口（goroutine+事件捕获+落库断言）：工具循环五态（执行+trace/降级/轮次上限/收尾兜底/预算闸）、取消两态（部分保留/零正文仅用户落库）、错误两态（部分保留/纯失败仅用户落库）、并发闸、回写定位。runSinStream/sinStreamRound 无测试直调（签名可动但保持）。

## 一、落地

- `runSinStream` 收窄为分派器（~25 行）：panic recover 双 defer 原样 → `buildSinTurn` → `runSinRounds` → abort 处理（error 帧+仅用户消息落库，注释随行）→ `persistSinTurn`。
- `sinTurnPlan`（装配产物：messages/opts/tools/schemas）+ `buildSinTurn`（原 277–332 行搬，含全部注释；无失败出口——历史读取失败 Warn 后按无历史继续）。
- `sinRoundOutcome`（循环产物：reply/reasoning/usage/trace + abort 哨兵）+ `runSinRounds`（原 333–435 行搬：轮次上限/收尾 nudge/取消与中途错误正文并入/工具降级/收尾轮拒执接回/工具轮执行；**可变状态圈死在函数内**，`messages` 切片循环后调用方不再读，本函数自有）。
- `persistSinTurn`（原 437–531 行搬：空回复兜底/零正文取消/extra 序列化/落库透传/回写定位/插图产物/成本/done 帧）。
- 预核漏项自查：`buildSinTurn` 首版漏 `eng` 参数（opts.EngineID），编译期抓到即补——拆分签名由数据流推导，编译器是第一道守卫。

## 二、等价证明与门禁

- **测试零编辑**：sin 族 20+ 用例（`-run TestSin` 4.9s）+ `go test ./internal/app/` 全包（117s）全绿——装配/循环/收尾三段的全部行为分支（含五条收尾出口）由既有测试网背书。
- **反向变异一组（主代理亲手）**：abort 分支落库 topicID 加 `TEMP-MUTATE` 后缀 → `TestSinStreamPlainErrorSavesUserMessage` 红（「失败轮应只落用户指令: []」）→ 还原绿 → `grep TEMP-MUTATE|REVERSE-TEST` 全仓 0 残留（批 14 教训收线纪律）。
- 前台 ci：golangci 0 issues / `go test ./...` 全绿 / **vitest 435 文件 3762 例全绿**（顺带证批 23 的 2 红确为环境瞬时）；尾段卫生守卫拦下 `releases/README.md` 源码包计数漂移（11 vs 12，e48c1c56 归档漏更新，与本批无关）——单修 `341da99d` 后 `check-docs.mjs` 复绿（源码包 12 个/合计 181.0 MiB 对账）。

## 三、god-func 存量状态

- 大拆三刀收官试水：AP2-05（表驱动 63 键）/ IN3-04（表驱动 12 op）/ AP2-01（三段拆+哨兵上提）——**switch 型用表驱动、状态圈死型用三段拆**；两种形态的等价证明都=「契约/测试先行→搬行→零编辑复跑绿+反向变异抽查」。
- god-func 簇余量：对账地图剩余为 god-file 大拆 ~26 + coupling ~20（拍板）+ 零散死码（等 109 绑定删除拍板）。
