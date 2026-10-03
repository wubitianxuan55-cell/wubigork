# 全仓审计第 44 批 · NOT_MOCKED 续刀（DAG 验收两件 + 轻语主动式频控对，45→30）· 2026-10-03

> 接续 [round-45（批次四十三）](round-45-p1-batch45.md)。NOT_MOCKED 续刀：**做真状态机的做真验收**（DAG 两件），纯状态面照抄 Go 校验（轻语频控两件）。dev mock 行为新增（申报），**不抬版本**。
> 快照：开工时 HEAD = `5f850361`（批次四十三）；工作树干净。

---

## 一、四件与形状来源

- **DagNodeApprove**（审批放行，危险操作分级审批的人拍板侧）：hold→pending + approved=true + error 清空；节点不存在/非 hold 拒绝（错误文案逐字同 Go `只审批待审批（hold）的节点（当前 %s）`）。mock 节点类型本就带 hold/approved/risk 全字段（types/office.ts 契约冻结面），映射表 `DagNodeApprove`/`DagAcceptAll` **早已在册**（本批才发现先前 grep head 截断漏看——映射零改动）。
- **DagAcceptAll**（一键验收，成品直出首刀）：run 内全部 done 节点一次置 accepted（AcceptedAt 同批时间戳）；无可验收返回明确信息**非错误**（Go 同口径）；产物计数按 outputs 长度如实统计，mock 无记忆回写面以「（mock 不落盘）」注记。
- **WhisperProactiveConfig/SetConfig**（轻语主动式频控对）：配置五字段（enabled/limitPerHour/intervalMin/quietStartHour/quietEndHour）+ 默认值同 `defaultProactivePushCfg`（3/30/-1/-1）；Set 支持**部分字段更新缺省保持原值**，校验逐条同 Go（limitPerHour ≥ 1「关闭请用 enabled=false」、intervalMin 10–120、时窗 -1 或 0–23、解析失败诚实报错）。
- 教训注记：**查映射表/清单类文件禁 `head -N` 截断取样**——已存在的东西会被看漏，白做一轮「补映射」计划。

## 二、落地

- `mock/office/methods_dag.ts`：Pick +2 名；DagNodeApprove/DagAcceptAll 两方法（错误文案与返回消息逐字对 Go）。
- `mock/office/types.ts`：OfficeMethods Pick DAG 行 +2 名。
- `mock/memory.ts`：模块级配置槽（默认值注释对 Go）+ 两方法（校验同款）。
- `mock/contract.test.ts`：NOT_MOCKED 删 4 条（45→30，novel AI 族 12 + 其他域 18）。

## 三、门禁

- tsc -b 绿（frontend）/ contract+store 金样 60 例绿 / **全量 vitest 435 文件 3762 例全绿** / 改动四文件 eslint 0 错 0 警。

## 四、余量与下一批

- NOT_MOCKED 余 30 条；可做真状态机的还有 GaeaResyncEvents（chat mock 事件序号面）与子代理族空态件；纯 AI 件（LLM 依赖）天然留白名单。
- 下一批候选：NOT_MOCKED 续刀三或对账地图其他活池；coupling ~20 与零散死码等拍板。
