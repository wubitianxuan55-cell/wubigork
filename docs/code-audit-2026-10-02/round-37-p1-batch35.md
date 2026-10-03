# 全仓审计第 35 批 · FE2-04 CostProjectsView 拆分（明细件+样式常量出仓）· 2026-10-03

> 接续 [round-36（批次三十四）](round-36-p1-batch34.md)。前端配方第六刀：CostProjectsView.tsx 895 行的明细件层（ItemRow/EntryPicker/ProjectForm+Field/SnapshotTable/小工具四件）出仓 `cost_projects_parts.tsx`，主组件留项目列表/详情编排。主代理直做，**不抬版本**。
> 快照：开工时 HEAD = `93db2bfd`（批次三十四）；工作树干净。

---

## 一、预核与切口

- FE2-04 复核（895 行）：两层缝——①明细件层（:594-895，302 行：可编辑明细行 ItemRow/成本库单价搜索下拉 EntryPicker〔自带 debounce〕/ProjectForm+Field/SnapshotTable/StatusBadge/fmtTime/slug/emptyProject，八件全部为主段消费）；②主组件（:35-592，558 行：项目列表→详情→版本快照→沉淀闭环编排）。
- 外部消费核查：CostInquiryPanel/ComposeModal/spaceBindings 仅注释提及；测试只导主件——**parts 无需 re-export**。
- 五个共享样式常量（fmtPrice/fieldCls/ghostBtn/solidBtn/iconBtn，Tailwind token 类）**双侧共用** → 全部落 parts 单源导出，主文件导入。

## 二、落地与编译期迭代

- `cost_projects_parts.tsx`（~330）+ `CostProjectsView.tsx`（~570）：行数对账闭合（561 主体 + 302 明细件 + 头/缝合 = 895）。
- import 重算：parts 带 Search+X 图标（**X 为单字母、首轮正则扫描双侧漏报——grep -w 精确定位后归 parts**）/message/costReadErrorText/useDebouncedValue/app/四类型；主文件去 ReactNode/CostSummary/Search/X/message/costReadErrorText/useDebouncedValue。
- 编译期迭代两次：①共享样式常量五件双侧共用——**从原文件拷字节插值（批 34 手编转义教训执行）**，iconBtn 跨行值被行号切片切断→按位置拼接修复（CRLF/LF 混排下禁锚点串匹配，用 find 定位拼接）；②常量插到 import 语句之间（语法合法但排版劣化）→ 移到导出语句之后。

## 三、门禁

- tsc 绿 / CostProjectsView 6 例绿 / 前台 ci **exit 0**（vitest 435 文件 3762 例全绿）/ golangci 0 issues；提交后树干净+HEAD 可编译。

## 四、余量与下一批

- FE2-04 部分收敛（895→570+~318）：主组件 558 行仍是状态+弹窗编排（设计型同族）。
- memoryhub god-file 族现状：CostLibraryView 1167→726+455、CostProjectsView 895→~570+~318。剩余设计型：弹窗/hook 分域（useController 36 回调、CostLibraryView 5 弹窗上提、CostProjectsView 编排）。
- 下一批候选：设计型配方刀（hook 分域/弹窗上提）或转到对账地图其他活池；coupling ~20 与零散死码等拍板。
