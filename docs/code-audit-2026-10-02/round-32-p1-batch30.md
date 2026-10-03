# 全仓审计第 30 批 · 前端拆分配方试水（FE3-15 methods_office.ts：DAG 域整域抽取）· 2026-10-03

> 接续 [round-31（批次二十九）](round-31-p1-batch29.md)。前端配方第一刀：DAG 域（mock 态+推进器+Dag* 方法 11 件）从 methods_office.ts 整域抽取至 methods_dag.ts，**组合导出原位展开保持键序与运行时绑定**。主代理直做，**不抬版本**。
> 快照：开工时 HEAD = `17bef745`（批次二十九）；工作树干净。

---

## 〇、预核与配方要点

- FE3-15 复核：methods_office.ts 1029 行 = DAG 状态机前言（:23-208，自包含：仅 types+全局 setTimeout/Date）+ 81 方法巨型对象字面量（:210-1030，方法排布历史混排）。
- **this 约束前置核查**：`this.PinnedMaterials()` 仅 WorkspaceSearch/SummarizeFile 两处（文件头注明三件套同文件约束）——Dag 块无 `this.`，可安全整域搬移。
- **组合形态**：`export const dagMethods: Pick<OfficeMethods, 11 键>` + 组合侧原位 `...dagMethods`（展开点=原 Dag 块位置）→ **键序逐位不变**；组合字面量保留完整类型注解（批 17 教训：spread 组合必须挂在显式注解上，避免超重载盲区）。

## 一、落地

- `methods_dag.ts`（323）：DagMockState/DagTplMockState 单例+预置「月度经营报告」链/模板、cloneRun/dagSnapshot/dagDerive/dagOutputsOf/dagRunNode/dagAdvance/DAG_STEP_MS + dagMethods（DagList/DagGet/DagRun/DagNodeRun/DagNodeSteer/DagNodeAccept/DagCancel/DagTemplateList/DagTemplateSave/DagTemplateNew/DagTemplateDelete）。
- `methods_office.ts`（732）：其余 70 方法+组合展开；import 重算（DagRunView/DagTemplateView 随域走，新增 dagMethods import）。

## 二、等价证明（前端配方三件套，进在册）

1. **tsc -b 全量类型检查**（真 tsc；**踩坑：从仓库根跑 npx tsc 会命中假 tsc 包打印 "This is not the tsc command" 且 exit 0 假绿——必须 cd frontend**）。
2. **契约金样**：mock-contract 全族+contract.test 共 52 例绿（744 绑定面锁）。
3. **端到端金样**：DagPanel/TasksWorkbench 32 例绿——Dag 方法经组件直调，spread 组合后的运行时绑定与 mock 态行为逐项验证。
4. 行数对账闭合：704+186+114+5 seam+13 wrapper+2 export+2 头尾=1054 两文件合计；前台 ci **exit 0**（vitest 435 文件 3762 例全绿）。
5. 过程两处编译期纠错（配方自证「终审是编译器」）：①组合侧漏带原 `export const officeMethods…= {` 开头（方法体悬空）；②Pick 类型实参末位多逗号。

## 三、FE3-15 余量与前端配方下一步

- FE3-15 **部分收敛**（DAG 域 ~323 行出仓，methods_office 1029→732；巨型字面量仍在，域分组全量重构待续刀——把 70 方法按域分组模块化需重排序，风险高一档，按「先立配方、小步验证」节奏单独立刀）。
- 前端配方三件套在册：**真 tsc（frontend 目录）→ contract 金样（52 例）→ 组件端到端金样**，外加键序不变原则。
- 下一批候选：FE3-15 续刀（方法域分组）或 FE6-07 ModuleLauncher（1038 行，首页两套变体+子组件+轮询）；coupling ~20 与零散死码等拍板。
