# 全仓审计第 38 批 · FE3-11 续刀三（hook 分域配方三连排：轮次 6 + 工作区/模型 4 + 回退重生成 2）· 2026-10-03

> 接续 [round-39（批次三十七）](round-39-p1-batch37.md)。配方已两刀验证，本批一次连排三族（先例=批 25/26 双刀批）；**会话装载族单留末刀**（喂 ensureEventsBound+useEffect 依赖，动它须连绑定接线一起核，单独走）。零逻辑改动，**不抬版本**。
> 快照：开工时 HEAD = `71c8d3a9`（批次三十七）；工作树干净。

---

## 一、三刀切分与依赖注入

- **轮次控制族**（`controller_actions_turn.ts` 54）：send/steer/cancel/approve/answerQuestion/setPermLevel 六件。cancel 读运行态——store 以 `{ getState: () => ControllerState }` **结构面注入**，不回导 useStore（防 controller→actions→controller 环；类型面用结构子集，参数注解诚实于实际消费）。
- **工作区/模型族**（`controller_actions_workspace.ts` 54）：pickWorkspace/switchWorkspace + compact/setModel 四件同仓（件数不足以各立文件，文件头注明）；跨族依赖 refreshFactBase 注入。
- **回退重生成族**（`controller_actions_rewind.ts` 49）：rewind/regenerate 两件；regenerate 编排「先截断成功、再重发」——**跨 hook 产物链**：useController 先解构 turn 族 send，再注入 `useRewindActions(dispatch, loadItemsFoldedFirst, store, send)`；useCallback 依赖数组逐字保留（rewind [dispatch, loadItemsFoldedFirst]、regenerate [store, rewind, send]）。
- **hook 调用序即原位序**：turn 解构行落在原 send 块位置（session 之前）→ workspace 落在原 pickWorkspace 块位置（memory 之前）→ rewind 落在原 rewind 块位置（return 之前）；send 定义先于 rewind 注入点，先定义后使用。

## 二、落地

- controller.ts 328→220：三块原位换三行解构；import 重算——eventSync 去 resetEventSync、memory 三助手去 errText/failWrite（logBridgeError 留守）、types 去 QuestionAnswer；新增三行 actions 导入。
- 组装为 Edit 原位替换（无 python 插值，中文注释零转义风险）。

## 三、门禁

- tsc -b 绿 / store.resync+store.t74+contract+store.invalidate 60 例绿 / **全量 vitest 435 文件 3762 例全绿（计数与批 36/37 持平）** / 三个新文件 eslint 0 错 0 警（controller.ts 15 条存量警告为搬移前既有）。
- **搬行零编辑证明法复用**：git diff 118 删行逐行回查三个新文件，除 3 行有意 import/类型清单重算外 115 行全逐字节命中。

## 四、余量与下一批

- FE3-11 累计 884→220+484+54+67+54+54+49：**只剩会话装载族末刀**（loadItemsFoldedFirst/loadSessionData/reconcileFinalAnswer/refreshFactBase 4 件）——动它须连 ensureEventsBound 注入与 useEffect 依赖数组一起核；reconcileFinalAnswer 消费 isFinalAnswerRendered（模块级纯函数，随走须查测试导入面再定 re-export）。末刀后 controller.ts 即纯接线（~130 行）。
- 下一批候选：FE3-11 末刀（装载族，本批配方直接适用但须过 useEffect 依赖）或 CostLibraryView 5 弹窗上提或对账地图其他活池；coupling ~20 与零散死码等拍板。
