# 全仓审计第 31 批 · FE3-15 续刀（Git/任务/证据链+WriteFile/数据备份 → methods_ops.ts）· 2026-10-03

> 接续 [round-32（批次三十）](round-32-p1-batch30.md)。前端配方第二刀：四域连排块整域抽取，配方同批 30。主代理直做，**不抬版本**。
> 快照：开工时 HEAD = `10f8ee5c`（批次三十）；工作树干净。

---

## 一、落地

- `methods_ops.ts`（222）：Git 演示族（GaeaGit* 7 件）/任务族（TaskList/Cancel/Kill/Retry/Output，taskMock+监听器）/证据链（GaeaJournalList/VerifyRecord/RollbackRecord）+WriteFile/数据备份族（DataBackup* 6 件，批次三a）——四域连排块一次性出仓，`Pick<OfficeMethods, 22 键>` 注解+`...opsMethods` 原位展开。
- `methods_office.ts`（550）：import 重算（taskMock/mockTaskListeners 随块走；pinnedMock 留守——Pin 三件套在 106-119 行的同文件约束保持）。
- 行数对账闭合：519（剩余方法）+185（ops 块）+2（dag 尾）+27（旧头）=733。

## 二、一次真回归的抓取与根修（配方教训升级）

- **回归**：全量 vitest 22 红（DeliverablesPanel 证据链三步展开族）——`app.GaeaJournalList is not a function`，组合对象缺 ops 的全部 22 键。
- **根因**：组装时漏插 `...opsMethods` 展开行。**为什么 tsc 没抓**：组合字面量的注解是 `Partial<Omit<...>>`——Partial 让除 PinnedMaterials 外全部键可选，**缺键在类型层面合法**；contract 金样 52 例也绿（只锁部分绑定面）。**真正守卫=组件端到端测试**（DeliverablesPanel 直调证据链方法）。
- **配方三件套修订（进在册）**：①真 tsc（frontend 目录）；②contract 金样；③**组件端到端=组合完整性的唯一真守卫**——对 Partial 注解的组合导出，tsc/contract 对缺键双盲，扩域后必须补跑消费该域的组件测试；终值以后全量 ci 兜底（本轮 22 红被 ci 拦下，门禁未失守）。
- 修复后复跑：面板/契约 13 文件 130 例全绿；前台 ci **exit 0**（vitest 435 文件 3762 例全绿）；提交后树干净+HEAD 可编译。

## 三、FE3-15 余量与下一批

- methods_office.ts 累计 1029→550（DAG+ops 两域 ~545 行出仓）；剩余主体=文件浏览/预览/附件/图像/herdsman/对话框（相互共享 mockFileBodies/mockXlsxState 状态面，属「共享状态域」，继续拆需先理状态归属——单独立刀）。
- 下一批候选：FE6-07 ModuleLauncher（1038 行）或 FE3-11 controller.ts（963 行，闭包结构需预核）；coupling ~20 与零散死码等拍板。
