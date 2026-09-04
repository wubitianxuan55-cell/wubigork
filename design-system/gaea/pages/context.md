# gaea 板块蓝图 · context 上下文标签页「上下文驾驶舱」

> 覆盖 Master 的通用分区规则在「上下文」标签页的落地。v4.67.0 落地
> （ContextView.tsx 重构），功能零删减：单列 8 卡堆叠 → 驾驶舱三区布局。

## 视觉性格：观测仪表（Observability Cockpit）

上下文标签页回答三个问题：**窗口里现在有什么（构成）→ 如何演化到这里
（过程）→ 每一类内容的具体内容（明细）**。它是开发者遥测面板而非内容页：
数据密度优先、等宽字体承载数字、六分类语义色（system 蓝/tools 橙/user 绿/
inject 紫/assistant 深蓝/tool 青）是全页唯一的强色，装饰色全部退后。
沿用星枢语言：`v3-panel` 玻璃容器、Luminous Glass 2.0、`--gaea-glow` 只给
「运行中」状态；ui-ux-pro-max dense dial（8/10）= 卡内 8-12px 间距、区块
12px 栏距；Subtle motion（150-200ms opacity/transform，禁大面积位移动画）。

## 布局（三区）

### ① 顶部总览条（ContextHeader，常驻）
- **左：窗口仪表**——「已用 / 窗口」大数字（等宽）+ 六分类**分段堆叠条**
  （吸收原 CurrentComposition 的构成条；段 hover 显分类名+token 数；
  段色 = 六分类语义色）+ 分类图例 chips（可点=高亮构成条对应段）。
- **右：统计 chips 行**——原 StatsBoard 9 项全部保留为紧凑 pill（轮次/
  步数/注入/压缩/修剪/工具调用/图片/缓存命中/成本），超出换行；数字等宽。
- 刷新按钮 + 运行中呼吸灯保留在条右端。

### ② 主区双栏（flex；<1100px 回落单列，辅栏 tab 横排到主栏下方）
- **左主栏（过程轴，min-w-0 flex-1）**：
  1. **趋势图**（原 ContextTrendChart，每请求一柱、六分类堆叠、事件标记、
     点选请求）——点选后柱高亮；
  2. **步骤详情**（原 StepDetail）紧贴趋势图下方——master-detail 就地联动，
     选哪根柱详情就在哪，不再跨屏滚动；未选中时显示最新一步；
  3. **上下文事件流**（原 EventsList：压缩/注入/修剪时间线）。
- **右辅栏（inspector，w-80 缩回 xl:w-72；tab 切换，渐进披露）**：
  - tab「浏览器」= 原 ContextBrowserCard（活跃/归档双页签+六分类过滤
    chips+节点行展开，交互不变）；
  - tab「文件」= 原 FileActivityCard（动作徽标时间线，倒序 40 条）；
  - tab「Agent」= 原 AgentNetworkCard（子代理树拓扑）。
  tab 记忆本地选择（localStorage gaea.context.inspectorTab）。

### ③ 空态 / 错误态
- 空态：虚线卡 + 一句话（沿用现有文案键）；错误条保留常驻顶部。

## 数据与纪律
- 数据源不变：`app.ContextView()` 单轮询 + useLiveReload(running)；所有
  原 props/state 语义保持；i18n 键全部复用（新增仅布局性键，如 tab 标签
  若需新键则三语同步）。
- 红线：八块信息一条不删——只重组（统计→chips、构成→分段条、浏览器/
  文件/Agent→inspector tab）；六分类语义色 hex 豁免沿用 CAT_COLORS 单源。
- 窄栏适配：<1100px 双栏回落；inspector 三 tab 在窄栏转为横向胶囊条。
