# 全仓审计第 40 批 · FE2-02 续刀（CostLibraryView 三内联 Modal 上提为受控展示件）· 2026-10-03

> 接续 [round-41（批次三十九）](round-41-p1-batch39.md)。批 34 留下的「弹窗上提」设计型续刀：删除确认/分类新建重命名/价格历史三个内联 antd Modal 上提 `cost_library_modals.tsx`。零逻辑改动，**不抬版本**。
> 快照：开工时 HEAD = `85643c50`（批次三十九）；工作树干净。

---

## 一、切面与形态

- 批 34 时「5 弹窗留设计型」的实情盘点：CostEntry/Import/Compare 三弹窗**本就独立文件**（memoryhub/），真正内联的是三个 antd Modal（~76 行 JSX）。本批只上提这三个；「弹窗上提」候选就此关账。
- **受控展示件形态**（批 34 展示件同配方）：状态（deleteName/catModal/catName/history*）与处理器（handleDelete/saveCategory/openHistory）留主件，props 穿线六处——`deleteName→name`（open 由子件 `!!name` 自算）、`historyRows→rows`、`historyName→name`、`setCatName→onNameChange`、`saveCategory→onSave`、`pathById.get(...) ?? "—"` 主件预解析为 `parentPath` 字符串（短路次序不变，纯查询无行为差）。
- 弹窗文件依赖极瘦：antd Modal/Input + icons Clock + PriceHistory/CostCategory 两类型；测试对三弹窗零触达（grep 证），无 re-export 负担。

## 二、落地

- `cost_library_modals.tsx`（115）：DeleteCostModal/CategoryModal/PriceHistoryModal 三件。
- CostLibraryView.tsx 729→669：三块 JSX 原位换三件用法；import 重算——antd 去 Modal/Input（剩 Popconfirm/message）、icons 去 Clock（唯一消费点随行）、新增 modals 导入行。

## 三、门禁

- tsc -b 绿 / CostLibraryView 金样 13 例绿 / **全量 vitest 435 文件 3762 例全绿（计数五批持平）** / 改动两文件 eslint 0 错 0 警。

## 四、余量与下一批

- CostLibraryView 族累计 1167→669+455（parts）+115（modals），主件剩数据装载/分类树/批量操作编排；弹窗上提候选关账，该族无既定续刀（进一步拆=装载逻辑分域，可套 hook 分域配方，收益中等）。
- 下一批候选：对账地图其他活池或 controller 族同型推广（methods_office.ts 状态面/App.tsx 源级锁解开后的拆分）；coupling ~20 与零散死码等拍板。
