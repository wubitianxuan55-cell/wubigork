# gaea 进度板块 · v3 对齐审计（2026-11）

> 并发线 C 交付：不写产品代码，只把「对齐」钉成可回归的断言与可交接的清单。
> 守卫：`frontend/src/schedule/uiAlignment.test.ts`（21 例，纯文本断言，无 DOM/React）。
> 口径来源：`进度计划/gaea-schedule-zpert8-uiux-port-plan-20261114.md` §2/§6、
> `frontend/src/v3/foundation.css`、`.gaea/AGENTS.md`「执行纪律」。
> 样式真源（6+1）：`schedule.css`、`display-preset.css`、`gantt/inplace.css`、
> `gantt/ruler.css`、`hit.css`、`network-display.css`、新增 `sched-v3.css`。

## 1. 对齐了什么

| 块 | 落点 | 口径与选择器 |
|---|---|---|
| 令牌桥接 | `sched-v3.css` §0 `:root` | `--sched-radius-sm/md/lg` ← `--md-sys-radius-*`；`--sched-elev-1/2` ← `--md-sys-elevation-*`（回退用 `color-mix`，不落裸 `rgba`）；`--sched-edge` ← `--v3-edge`；`--sched-hairline` ← `--v3-split`；`--sched-hover-bg / active-bg / focus-ring` ← `--color-text / --color-primary / --gaea-glow`；`--sched-motion` 140ms 与 `--md-sys-transition-fast` 同档 |
| 容器原语 | §1 | `.sched-command-deck`、`.sched-ribbon-panel`、`.sched-workspace`、`.sched-dock`、`.sched-docbar`、`.sched-statusbar`、`.sched-export-preview`、`.sched-command-palette-modal .ant-modal-content` 统一 radius-lg + hairline + elev-1 + 顶部 1px 高光线（`::before` 走 `--sched-edge`）；玻璃只留 `.sched-command-deck` 一档；`.sched-gantt-frame / -tablepane / -canvaspane`、`.sched-network-table-inner` 保持实底卡（信息层不做玻璃） |
| 交互态 | §2（2.1–2.17） | 序列固定 hover → active → selected → disabled → focus-visible；焦点环统一 `--sched-focus-ring`（`--gaea-glow` 70%）+ `outline-offset:2px`；禁用保留可见（.45）却掐掉 hover 反馈。覆盖 `.sched-menu-item`、`.sched-ribbon-tabbar .sched-menu-item`、`.sched-ribbon-command`、`.sched-command-search-trigger`、`.sched-command-palette-item`、`.sched-doc-tab`、`.sched-doc-add`、`.sched-dock-item`、`.sched-colmenu-item`（含 `:has(input:focus-visible)`）、`.sched-gantt-row`、`.sched-gantt-cell.sched-th`、`.sched-gantt-split`、`.sched-aoa-node-wrap`、`.sched-pdm-node`、`.sched-statusbar`、`.sched-hit-menu-layer`、`.sched-inplace-filter-btn` |
| 键盘 / aria | `gantt/GanttTable.tsx`、`gantt/GanttRowShell.tsx`、`pages/schedule/ScheduleCommandPalette.tsx`、`pages/SchedulePage.tsx` | 表格行 roving tabindex（选中行 `0`、余 `-1`，`ArrowUp/Down` 移动）；行外壳接可选 `tabIndex/onKeyDown`（旧调用零改动）；表头列筛 `aria-haspopup="dialog"` + `aria-expanded` 随 Popover 真开合、`Esc` 收起并把焦点还给按钮；命令面板走 combobox 模式（`aria-controls` + `aria-activedescendant`）+ `role="listbox/option"` + `aria-selected/aria-disabled`，选项 `tabIndex=-1` 不占 Tab 位；功能区 `role="tablist"/"tab"` + `aria-selected`；状态栏 `role="status"` + `aria-live="polite"`，尾部同步区 `aria-live="off"`（不让逐字变化打断读屏） |
| 动效降级 | §3 | 双入口（`@media (prefers-reduced-motion: reduce)` 与 `.ui-reduced-motion`）同一份 17 类清单 `transition: none !important`，`:active` 位移同关；`schedule.css` 原 `.sched-chat-running-dot` 的 `animation:none` 两入口一并保留 |

## 2. 没对齐什么（诚实清单）

1. **语义色不在 v3 令牌内**：`--sched-critical/normal/link/group/deadline/front` 仍是 `schedule.css`
   `:root` 的 12 处 hex 定义（暗/亮两档 × 6）。斑马关键红/工作蓝属「领域语义色」，v3
   只提供 `--color-*` 系统位，硬塞会丢掉「关键路径红」的专用语义——这是边界，不是漏改。
2. **规则层仍有 12 处裸 hex**（`schedule.css`）：`.sched-drag-tip`（`#1f2937`/`#fff`）、
   `.sched-aoa-drag-tip`（`#fff`）、`.sched-aoa-mode-on` 与其 `:hover`（`#fff`）、
   `.sched-export-preview-body svg`（`#fff`）、`.sched-group-c0..c5::before`（六条分组色）。
   A 线按「新文件不动老文件」的并发纪律没碰，需另刀（见 §4）。
3. **导出件是纸面固定色**：`exportPalette.ts` / `ganttExport.ts` 的独立打印调色板不跟主题
   （`ganttExport.ts` 就在 eslint `no-raw-hex` 豁免名单里，属设计而非漂移）。
4. **画布是 SVG，交互态只到容器**：`.sched-aoa-node-wrap` / `.sched-pdm-node` 有焦点环，
   但横道条、依赖线、箭线本身不参与 CSS 交互态。
5. **焦点环已写、元素却不可聚焦（两处，需 B 线补 tabIndex）**：`.sched-gantt-split`
   （`GanttView.tsx:949` 只有 `onMouseDown/onDoubleClick`）与 `.sched-statusbar`
   （`SchedulePage.tsx:1587` 只有 `role=status`）都没有 `tabIndex`，A 的 `:focus-visible`
   当前不可达——分隔条的键盘调宽、状态栏的焦点反馈都还差一步。这是 A/B 交界处最容易漏的一条。
6. **合成组头行不可键盘折叠**：`.sched-gantt-row.sched-group-row`（`GanttTable.tsx:366`）
   `tabIndex={-1}` 且仅 `onClick`，无 `Enter/Space`、无 `aria-expanded`；任务行有键盘路径，
   组行没有。
7. **功能区 tabs 缺键盘模式**：`role="tab"` 按钮（`SchedulePage.tsx:807`）无 roving tabindex、
   无 `ArrowLeft/Right`，WAI-ARIA tabs 的键盘契约只落实了一半（Tab 能到，方向键不切页签）。
8. **reduced-motion 残留**：`.sched-resize-handle`（`schedule.css:1076`，拖拽改工期的手柄）
   的 `transition` 不在 §3 的 17 类清单内。
9. **命令面板条目的焦点环属防御性写法**：面板走 `aria-activedescendant`，DOM 焦点始终在
   搜索框，`.sched-command-palette-item:focus-visible` 实际不会被命中（保留无害）。
10. **右键浮层含 antd Menu**：`.sched-hit-menu-layer` 只管容器外观，菜单项键盘/焦点由 antd
    自身实现，板块不接管（浮层定位与层级是自管的）。

## 3. 怎么验收

守卫（与 CI 同口径，不跑全量）：

```bash
cd frontend
npx vitest run src/schedule/uiAlignment.test.ts   # 21 例：真源齐全 / 10 个焦点环 / 零裸 hex / 双入口降级 / 覆盖成对 / 状态栏活区
npx tsc --noEmit -p tsconfig.app.json
npx eslint src/schedule/uiAlignment.test.ts
```

守卫守的是这些漂移：样式真源文件被改名或合并（清单不同步）、十个交互钩子少焦点环、
`sched-v3.css` 混进裸 hex（或检测器本身坏掉——已加判据自检）、只挂一个降级入口、
降级块指向根本不存在的动效规则（假覆盖）、状态栏 aria 语义半落地（`role` 与 `aria-live` 只来一个）。

人工走查（亮 / 暗 × 1440 / 1100 / 880 六格）：

1. 键盘 Tab 全链：页签 → 功能区命令 → 命令面板（`Ctrl+K`，上下键 + Enter）→ 表格行
   （`ArrowUp/Down`）→ 表头列筛（`Enter` 开、`Esc` 收并回焦）→ 文档条 → 右舷条目；
   全程焦点环可见、不被 `overflow:hidden` 裁切。
2. `Esc` 归位：命令面板、列筛 Popover、右键浮层三处关闭后焦点回到出发元素。
3. reduced-motion：系统开启减弱动效 + 应用内开关各验一次，页面无过渡、无脉冲动画。
4. 主题切换：三档显示预设（常规/护眼/黑白）× 亮暗，横道条与依赖线可分辨，关键色不被预设吞掉。
5. 窄窗 880：命令台与右舷无横向溢出，长标签走省略号而非撑破容器。

## 4. 后续刀（三条）

1. **`schedule.css` 规则层 12 处裸 hex 令牌化**：分组色落 `--sched-group-c0..c5`，拖影/预览底
   改 `color-mix`；同刀把前锋线/竣工线纳入 `display-preset`（黑白档目前仍是彩色线），
   并给 `.sched-resize-handle` 补降级归零。
2. **A/B 交界补齐**：给 `.sched-gantt-split`、`.sched-statusbar` 加 `tabIndex`（分隔条补
   `ArrowLeft/Right` 调宽），给功能区 `role=tab` 加 roving tabindex + 方向键——三处都是
   「规则/语义已就位、入口还差一行」的便宜活。
3. **组头行键盘折叠**：`GanttTable.tsx` 合成组头行补 `Enter/Space` + `aria-expanded`，
   与任务行的 roving 序列打通（组行当前被排除在 `taskRowSeq` 外，需先定「组行是否参与寻址」）。
