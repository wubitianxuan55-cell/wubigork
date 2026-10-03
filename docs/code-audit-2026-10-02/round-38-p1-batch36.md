# 全仓审计第 36 批 · FE3-11 续刀（hook 分域配方首刀：记忆/画像族 + 桥接错误三助手出仓）· 2026-10-03

> 接续 [round-37（批次三十五）](round-37-p1-batch35.md)。**hook 分域配方首刀**：controller.ts 的记忆/画像族八回调 + 桥接错误三助手出仓 `controller_actions_memory.ts`。与组件拆分同策略——先切一小块内聚组立配方，不一次吞 36 回调。主代理直做，**不抬版本**。
> 快照：开工时 HEAD = `51b81793`（批次三十五）；工作树干净。

---

## 一、预核与配方设计

- useController 36 回调盘点出六域：会话装载族（喂 ensureEventsBound deps）/轮次控制/会话管理/工作区/记忆画像/回退重生成。**首刀选记忆/画像族**：8 件、多为 [dispatch] 单行、域内聚。
- **跨族依赖显式化**：clearFactBase 依赖 refreshFactBase（会话装载族）→ 子 hook 签名收 `(dispatch, refreshFactBase)` 两参，useCallback 依赖数组逐字保留（fetchMemory=[]、其余 [dispatch]、clearFactBase [dispatch, refreshFactBase]）。
- **防环关键决策**：logBridgeError/errText/failWrite 三助手被两侧消费（failWrite 走动作族、errText 留守 compact、logBridgeError 走 ensureEventsBound）——三助手**随动作族出仓并导出**，controller.ts 单向导入。若留在 controller.ts 会形成 controller→actions→controller 环。
- hook 规则：useMemoryActions 在 useController 内**原位替换记忆块**（调用位置固定、无条件），hook 调用顺序逐渲染一致。

## 二、落地

- `controller_actions_memory.ts`（54）：三助手（补 export）+ `useMemoryActions(dispatch, refreshFactBase)` 八件 + 返回对象。
- `controller.ts`（414→376）：记忆块原位换解构一行；import 重算（MemoryView 随块走，新增 actions 导入）。
- 组装为 Edit/python 行号删补——**锚点教训复训**：跨批改写后头部 import 形态已变（多行 type import 块），锚点必须按当前文件实测取（首轮用旧文件形态断言失败即中止，未写盘）。

## 三、门禁

- tsc 绿 / store.resync+contract 53 例绿 / 前台 ci **exit 0**（vitest 435 文件 3762 例全绿）/ golangci 0 issues；提交后树干净+HEAD 可编译。

## 四、余量与下一批

- FE3-11 累计 884→376+484+54：**hook 分域配方已立**（子 hook 收参注入跨族依赖+依赖数组逐字保留+三助手随动作域走防环+原位替换保 hook 顺序），余五族同配方续刀即可。
- 下一批候选：FE3-11 续刀（会话管理族 13 件/轮次控制族/回退族）或对账地图其他活池；coupling ~20 与零散死码等拍板。
