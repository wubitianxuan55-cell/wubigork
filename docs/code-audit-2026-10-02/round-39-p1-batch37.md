# 全仓审计第 37 批 · FE3-11 续刀二（hook 分域配方：会话管理族十一件出仓）· 2026-10-03

> 接续 [round-38（批次三十六）](round-38-p1-batch36.md)。hook 分域配方第二刀：会话管理族 11 回调出仓 `controller_actions_session.ts`。同配方同纪律——原位搬移零逻辑改动，**不抬版本**。
> 快照：开工时 HEAD = `58a4bbf3`（批次三十六）；工作树干净。

---

## 一、域选取与跨族依赖

- 余五族中按纪要候选选**会话管理族**：newSession/listSessions/listProjectSessions/fetchSessionStats/resumeSession/archiveSession/unarchiveSession/pinSession/deleteSession/renameSession/refreshMeta 共 11 件，域内聚（全部围绕会话生命周期）。
- **跨族依赖显式化**：newSession/resumeSession 依赖 refreshFactBase（会话装载族）、resumeSession 依赖 loadItemsFoldedFirst（会话装载族）→ 子 hook 签名收 `(dispatch, loadItemsFoldedFirst, refreshFactBase)` 三参，useCallback 依赖数组逐字保留（listSessions/listProjectSessions=[]、refreshMeta/archiveSession 等 [dispatch]、newSession [dispatch, refreshFactBase]、resumeSession [dispatch, loadItemsFoldedFirst, refreshFactBase, fetchSessionStats]）。
- **域内件不注入**：fetchSessionStats 被域内 resumeSession 消费，同时在返回对象导出——依赖数组里的 fetchSessionStats 在子 hook 作用域内自然闭合，无需上提。
- 防环复查：本族只消费桥接错误两助手 failWrite/logBridgeError，二者已在 `controller_actions_memory.ts` 导出（批 36）→ session 文件单向导入 memory 文件，controller 单向导入两者，无环。

## 二、落地

- `controller_actions_session.ts`（67）：`useSessionActions` 十一件 + 返回对象。
- `controller.ts`（376→328）：会话块原位换解构一行（调用点在 loadItemsFoldedFirst/refreshFactBase 定义之后，注入参数先定义后使用）；import 重算——新增 actions 导入行，types 清单去 SessionMeta/ProjectGroup（随块走后无消费），HistoryMessage 仍被 loadItemsFoldedFirst 用故保留。
- 顺带修批 36 残留：`controller_actions_memory.ts` 里 useMemoryActions 头上错挂的 isFinalAnswerRendered 注释（该函数本体留在 controller.ts）换成正确的一句话注释。纯注释零行为。

## 三、门禁

- tsc -b 绿 / store.resync+store.t74+contract+store.invalidate 60 例绿 / **全量 vitest 435 文件 3762 例全绿（与批 36 计数持平）** / 改动三文件 eslint 0 错 0 警（controller.ts 15 条存量警告为搬移前既有）。

## 四、余量与下一批

- FE3-11 累计 884→328+484+67+54，**余四族**：会话装载族（4 件，喂 ensureEventsBound，动它须连 useEffect 依赖一起核）、轮次控制族（send/steer/cancel/approve/answerQuestion/setPermLevel 6 件）、工作区+模型族（pickWorkspace/switchWorkspace/compact/setModel 4 件）、回退重生成族（rewind/regenerate 2 件，regenerate 依赖 rewind+send 跨域注入）。
- 下一批候选：FE3-11 续刀三（轮次控制族 6 件，体量与内聚与首刀相当）或对账地图其他活池；coupling ~20 与零散死码等拍板。
