# 全仓审计第 34 批 · FE2-02/FE1-05 CostLibraryView 拆分（三份渲染+分类树出仓）· 2026-10-03

> 接续 [round-35（批次三十三）](round-35-p1-batch33.md)。前端配方第五刀：CostLibraryView.tsx 1167 行的展示件层（分类树递归节点/列表/表格两视图/CostRow 行件/人材机条）出仓 `cost_library_parts.tsx`，主组件留状态+5 弹窗编排。主代理直做，**不抬版本**。
> 快照：开工时 HEAD = `64ec4f81`（批次三十三）；工作树干净。

---

## 一、预核与切口

- FE1-05/FE2-02 复核（1167 行）：天然两层——①展示件层（:727-1167，441 行：CategoryNode 递归树/props 接口四件/CostRow+ListView+TableRow+TableView 四个 React.memo 件/MiniCompositionBar，依赖极瘦：memo+8 图标+2 类型，无 antd/bridge/hooks）；②主组件（:32-726，695 行：状态+5 弹窗内联编排——**弹窗内联属设计型，本刀不动**）。
- 双向消费：测试从原路径导入 `{ CostLibraryView, CostRow, ListView }` → CostRow/ListView/TableRow/TableView 由主文件 re-export 保路径；`CategoryNode` 主段 :348 使用 → parts 补 export；`TableView` 原本**无** export（测试也没引）→ parts 补 export 供主用；`SortKey` 两段共用 → 落 parts 单源导出。
- genui 守卫族 grep 命中为 `maxTableRows` 假阳性——组件无外部守卫引用。

## 二、落地与一次自伤的抓取

- `cost_library_parts.tsx`（455）+ `CostLibraryView.tsx`（726）：行数对账闭合（17 旧头+1 SortKey+708 主体+441 展示件=1167）。
- **自伤一处被金样当场抓获**：STATUSES 常量从手写头漏掉后补插时，python 里手工 UTF-8 字节把「草稿」错编成「草案」（\xe7\xa8\xbf→\xe6\xa1\x88，一字之差）——CostLibraryView.test 13 例当场 1 红。**教训入册：python 字节级插值禁手编 UTF-8 转义（中文字符必须从源文件字节拷贝或走 Edit 工具），文案常量搬移后必须 diff 原文逐字节核对**。修复后 13 例全绿。
- 门禁：tsc 绿 / CostLibraryView 13 例绿 / 前台 ci **exit 0**（vitest 435 文件 3762 例全绿）/ golangci 0 issues；提交后树干净+HEAD 可编译。

## 三、余量与下一批

- FE2-02/FE1-05 部分收敛（1167→726+455，「三份渲染」出仓）；5 弹窗内联（状态+JSX 编排）属设计型，收敛需弹窗状态上提配方（同 hook 分域族）。
- 下一批候选：FE3-11 续刀（hook 分域）或 CostLibraryView 弹窗上提或 CostProjectsView 862；coupling ~20 与零散死码等拍板。
