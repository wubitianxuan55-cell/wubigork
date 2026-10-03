# 全仓审计第 33 批 · FE3-11 controller.ts 第一刀（状态机纯函数层出仓）· 2026-10-03

> 接续 [round-34（批次三十二）](round-34-p1-batch32.md)。前端配方第四刀：controller.ts 884 行的**纯状态机层**（types/item id 取号与 slot 口径/历史重建/resync+resume 快照解析/applyEvent/reducer/initialState，~471 行零闭包）出仓 `controller_state.ts`；React 编排层（useStore/ensureEventsBound/useController/logBridgeError 族）留守。主代理直做，**不抬版本**。
> 快照：开工时 HEAD = `5346e60a`（批次三十二）；工作树干净。

---

## 一、预核与切口

- FE3-11 复核（884 行，审计时 963）：两层天然缝——①纯状态机层（:18-488，全 export 无闭包：ToolStatus/Item/ControllerState 类型、eventLogSeq/counterSlot/assistantSlot 取号、flushPendingUser/rebuildHistoryItems/parseResyncItems/resumeSnapshotItems、applyEvent、reducer、initialState）；②React 编排层（useStore 单例/ensureEventsBound 依赖注入式绑定/useController 280 行单 hook 36 回调/logBridgeError 族——**logBridgeError 触 app.LogFrontendError，属编排侧非纯函数**）。
- 判定延续：useController 36 回调的 hook 分域=设计型（依赖数组重排），不属本刀；本刀只做纯层出仓，**为未来 hook 分域先画清纯/编排边界**。
- `export * from "./controller_state"` 原路 re-export——外部消费（store.ts 聚合/contract 测试/组件）导入路径零改动。

## 二、落地与编译期迭代

- `controller_state.ts`（484）+ `controller.ts`（415）：行数对账闭合（471 纯层 + 395 编排 + 头/缝合线 = 884）。
- import 重算：纯层带 parseTodos+11 类型名（BalanceInfo/ContextInfo/FactBaseView/HistoryMessage/JobView/Meta/SessionStatsView/WireApproval/WireAsk/WireEvent/WireUsage）；**logBridgeError 的 app.LogFrontendError 依赖经扫描定位后判归编排侧**（防「纯函数偷偷摸桥」）；编排层补 `export *` + 值/类型双 import（applyEvent/reducer/initialState/resync 族 + Action/ControllerState/Item）。
- 编译期迭代三次（终审是编译器配方复训）：①漏 type import（ControllerState/Action/Item）→ 级联出测试文件 TS7006 implicit-any 假象（根因单一，修一處全消）；②`Action` 原为模块私有类型、编排层要用 → 补 `export`（拆分会把「原私有、跨层要用」的符号逼出 exported 决策——这是拆分的正向收益：纯/编排的契约被显式化）；③CRLF 行尾下 python 字节替换需按 \r\n 分支（wb 模式）。

## 三、门禁

- tsc 绿 / store.resync+store.t74+contract+mock-contract 12 文件 105 例绿 / 前台 ci **exit 0**（vitest 435 文件 3762 例全绿）/ golangci 0 issues；提交后树干净+HEAD 可编译。

## 四、余量与下一批

- FE3-11 部分收敛（884→415+484）：useController 280 行单 hook 36 回调仍在（hook 分域=设计型，需自定义 hook 抽取+依赖数组保全配方，单独立刀）。
- 下一批候选：FE3-11 续刀（hook 分域）或 CostLibraryView/CostProjectsView 组件族；coupling ~20 与零散死码等拍板。
