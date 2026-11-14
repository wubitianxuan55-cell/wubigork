# gaea 进度计划 · 「纯照斑马外观密度」审计（2026-11）

> **口径**：只要斑马（ZPert8）的专业**密度**，不要 v3 星枢的玻璃 / 高光线 / 光晕。
> 形状语言 = 实底 + 1px 细线 + 30px 行高 + 14px 表格字 + 控件 4px 圆角 + 零装饰光效。
>
> 真源：`frontend/src/schedule/sched-zebra.css`（A 线：令牌 + 交互态）、
> `schedule.css`（就地最小修补）、`gantt/inplace.css` / `hit.css`（既有局部样式表）。
> 守卫：`frontend/src/schedule/zebraDensity.test.ts`（本刀新增，并发线 C）
> ＋ `sched-zebra.css.test.ts`（A 线自守）。
> 反面参照：`frontend/src/v3/foundation.css`（`--v3-glow*` / `--v3-edge` /
> `backdrop-filter`，本轮一条不取，由守卫第 2 组断言封堵）。

## 1. 斑马证据 → gaea 落点

证据文件统一为 `进度计划/gaea-schedule-zpert8-uiux-re-20261114.md`（下称 re）。

| # | 斑马证据（re 行号 + 取值） | gaea 落点 | 守的是哪种漂移 |
|---|---|---|---|
| 1 | **行高 30**：line 841「行高 30px」、line 757 `defaultRowHeight` 逐行累加 | `gantt/ganttUtil.ts:18` `ROW_H = 30`；`sched-zebra.css:54` `--sched-row-h: 30px`；`:315` 行高消费 | CSS / TS 两处各写一份 → 滚到底累积半行错位（守卫第 3 组同值比对） |
| 2 | **表格 14px**：line 359 `QTableWidget{font-size:14px}` | `sched-zebra.css:52` `--sched-font-table: 14px`；`:301` 单元格、`:308` 表头格（覆盖 `schedule.css:946` 的 12px） | 12↔14 混档 → 表头与表体两代字号（守卫第 1 组令牌带值断言） |
| 3 | **表头底与网格线**：line 359 `QHeaderView::section{background:#F8FAFC;color:#4E5969;border-bottom:1px solid #E2E8F0}` | `sched-zebra.css:46-49` 四条令牌；`:305-311` 表头格；`:316` 行横线 | 网格线不走令牌 → 暗色档线压字；写死斑马 hex → 6 套主题 + 护眼/黑白预设全失效（§2.3） |
| 4 | **4px 圆角**：line 473 `border-radius: 4px`（office2016/2021 工具按钮） | `sched-zebra.css:33` `--sched-radius-sm: 4px`；`:342` 七类控件消费；`:320-333` 表格/画布统一归 0 | 圆角从按钮渗进网格 → 行首尾白缝、表头表体接缝错位（守卫第 5 组圆角纪律） |
| 5 | **零 padding 就地编辑**：line 364 `ZQtInplaceEdit{padding:0px}`、line 567（6 步流程第 2 步） | `gantt/inplace.css:40` `padding: 0`；`:54-65`（antd 输入体 padding 归零、行高 20） | 编辑框撑高行 → 开编辑瞬间行高由 30 变 32，整页抖一行 |
| 6 | **12 项右键密度**：line 112（`20097…20109` 逐项）、line 22、line 580-588（命中类型统一追加器） | `hitMenu.ts:74-86` 12 项逐字对应；`sched-zebra.css:358-364`（行高 `--sched-ctl-h` 24px、内边距 8px、4px 圆角，收掉 `hit.css:36` 的 28px） | 行高 28↔24 混档 → 菜单比功能区还松，密度感直接崩（守卫第 4 组交互态） |

## 2. 刻意偏离（偏离是决策，不是漏抄）

1. **双标尺保 40px（两行 × 20）**：斑马的双标尺（re:130/176 工程标尺 + 进度标尺）
   与表格表头是两套对齐方式——它的表头是独立 sticky 层。gaea 要求**表格首行与画布首行同基线**，
   标尺必须吃满 40。真源 `timescale.ts:368`（`GANTT_RULER_ROW_H = 20`）与 `:372`（×2）。
   守卫「双标尺仍是 40」把它钉成断言：改 40 之前先改本节。
2. **浮层保留 1 档软阴影**：`sched-zebra.css:39` `--sched-elev-2: 0 2px 8px rgba(0,0,0,.14)`，
   只给 `.sched-hit-menu-layer`（`:353`）与命令面板；常规面 `--sched-elev-1: none`。
   依据 re:361/370-371——斑马**新弹层**（`#ZDocFloatMenuPanel`）本身也带 8px 圆角 + 软阴影，
   「零光效」禁的是 v3 的发光/玻璃，不禁止弹出层的一档投影。
3. **表格 14 / 浮层 12 是斑马自己的两档**：re:359 表格 14px、re:362 浮层正文 12px。
   所以 `sched-zebra.css` 把表格与表头升到 14，而 `inplace.css` 的就地编辑、菜单、
   状态栏保持 12；双击开编辑时字号从 14 跳到 12 属预期（斑马同款两级），不是漏改。
   若要统一成一档，须同时改 re 证据与本表——先改文档再改代码。
4. **主题色仍走 gaea 令牌**：`--sched-head-bg / -ink / -grid-line` 写成
   `var(--md-sys-color-*, #hex)`（`sched-zebra.css:46-49`），斑马 hex 只作兜底段。
   不绑死 `#F8FAFC/#4E5969/#E2E8F0`：否则 6 套主题与 `display-preset.css`
   的护眼/黑白两档无法继续叠（亮度对比在 `display-preset.css:20-38` 已逐值核算）。

## 3. 验收

### 3.1 守卫命令

```bash
cd frontend
npx vitest run src/schedule/zebraDensity.test.ts src/schedule/sched-zebra.css.test.ts
npx tsc --noEmit -p tsconfig.app.json
npx eslint src/schedule/zebraDensity.test.ts
```

`zebraDensity.test.ts` 六组断言对应六类漂移：令牌缺失/错值（第 1）、光效回潮（第 2）、
CSS↔TS 行高分叉（第 3）、18 组交互态缺格（第 4）、大圆角渗入网格（第 5）、
降级单入口（第 6）。A 线文件缺失时抛带路径的错误，不会以 `undefined` 崩在半路。

### 3.2 人工走查（亮 / 暗 × 1440 / 1100 / 880，共 6 屏）

| 看什么 | 判据 |
|---|---|
| 行高 30 不抖动 | 滚到底再快速上下滚：横道条与表格行不错位；拖完分隔条仍对齐；双击开就地编辑，行高不变 |
| 表头文字对比 | 表头字取 `--sched-head-ink`；亮档白底、暗档深底都 ≥4.5:1；表头底与表体底有明显但不刺眼的差 |
| focus 环不撑行 | Tab 走功能区 → 表格行 → 状态栏：2px 环 + `outline-offset:-2px`，不产生横向滚动条、行高不变 |
| reduced-motion 无动画 | 系统「减弱动效」开 与 gaea 偏好开 两个入口分别验：hover 无过渡（`transition: none`） |
| 零光效 | 整页无玻璃模糊 / 发光边；DevTools 抽查 `backdrop-filter` 为 `none`、`text-shadow` 为 `none` |
| 窄屏 880 | 状态栏 24px 高不换行；右键浮层不出屏（`max-height:70vh` 生效） |

## 4. 后续刀（≤3）

1. ~~接线~~ **已收口**（主代理）：`SchedulePage.tsx` 已在 `tailwind.css` 之后引入
   `../schedule/sched-zebra.css`（顺序在 `display-preset` / `schedule` / `hit` 之后）。
2. ~~不可达的 disabled 规则~~ **已收口**（主代理）：`sched-zebra.css` 的 disabled 组已补
   `.sched-gantt-cell.sched-th[aria-disabled="true"]` 变体；B 线把 `aria-disabled` 挂到
   表头格/状态栏后即可通电。`:disabled` 变体保留（换成 button 时仍生效）。
3. ~~注释指错真源~~ **已收口**（主代理）：`uiAlignment.test.ts` 的常量已改名
   `DENSITY_SHEET = 'sched-zebra.css'`，`schedule.css` 头注释同步；全仓已无 `sched-v3.css`
   引用（`grep` 零命中）。
