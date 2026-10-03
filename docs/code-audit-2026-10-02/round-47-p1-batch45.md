# 全仓审计第 45 批 · NOT_MOCKED 续刀三（resync/浏览器观察/子代理族五件，45→25）· 2026-10-03

> 接续 [round-46（批次四十四）](round-46-p1-batch44.md)。NOT_MOCKED 续刀三：事件防线面 + 浏览器观察空态 + 子代理族四件（其中 promote/followup 接上既有 mock 会话族 fixtures）。dev mock 行为新增（申报），**不抬版本**。
> 快照：开工时 HEAD = `a9ac52c0`（批次四十四）；工作树干净。

---

## 一、五件与口径来源

- **ResyncEvents**（事件序号防线补拉面）：mock 会话非事件溯源（ChatStreamPlain 是组件内脚本流，不经 wire 事件总线）→ 快照恒空集 `{seq: 0, items: []}`；**Items 非 nil 是 Go 契约红线**（nil 序列化成 null，前端按数组消费会崩——GaeaResyncResult 注释原话），落 mock 注释。
- **GaeaBrowserObserve**：浏览器未运行是观察窗**正常态**（Go 不返回 error，前端空态渲染）→ mock 恒 `{available:false, …}` 空帧（形状对齐 BrowserPanel.tsx 的 UNAVAILABLE_VIEW 口径）。
- **GaeaSubagentContextView**：mock 无子代理持久档 → 恒空时间线（ok=true + 零值类目 + 空集，行形状与 core.ts ContextView 同源），空态渲染路径可走查。
- **PromoteSubagent / SubagentFollowUp**：接上既有 mock 会话族（methods_office.ts SubagentRuns 的 `/mock/sessions/a.jsonl` 两 ref 同源硬编码，注释互指）——提升返回新顶层会话路径、追问受理回 `"follow-up started（mock 受理不执行）"`（Go 受理即回 "follow-up started"、执行在后台，mock 无引擎如实注记）；非 sa_ 前缀拒绝文案逐字同 Go。

## 二、落地

- `mock/chat.ts`：Pick +ResyncEvents + 方法（History 旁）。
- `mock/core.ts`：Pick +2 名 + GaeaBrowserObserve/GaeaSubagentContextView 两方法（Trajectory 前，contextview 域聚拢）。
- `mock/office/methods_office.ts` + `mock/office/types.ts`：PromoteSubagent/SubagentFollowUp 两方法 + Pick +2 名（methods_office 本身是 Partial 组合无逐名清单，types.ts 是唯一清单面）。
- `mock/contract.test.ts`：NOT_MOCKED 删 5 条（30→25；余=novel AI 族 12 + ChatStreamCancel/DAG 文档封面 PPT 语音图谱绘图等其他域 13）。

## 三、门禁

- tsc -b 绿（frontend）/ contract+store 金样 60 例绿 / **全量 vitest 435 文件 3762 例全绿** / 改动五文件 eslint 0 错 0 警。

## 四、余量与下一批

- NOT_MOCKED 余 25 条：**可继续做真/做实的只剩 ChatStreamCancel**（mock 流加 runID+可取消）与 AI 空态件；其余 12 件 novel AI + 12 件其他域 AI 均为「浏览器 dev 无 LLM 内核」天然白名单——**NOT_MOCKED 池接近收敛**，剩余件下一刀收尾或维持白名单待拍板（AI 件补 mock 价值存疑：空响应反而误导走查）。
- 下一批候选：NOT_MOCKED 收尾刀（ChatStreamCancel+AI 空态件裁决）或对账地图其他活池；coupling ~20 与零散死码等拍板。
