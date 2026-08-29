# gaea 3.0 UI 革命性重设计 ——「星枢 Constellation OS」(v2 定稿)

> 状态：**定稿 v2**（2026-08-15，用户指令「革命性重设计整个 UI，适配 V3.0，面板布局可调整」）
> 前置：v1 设计系统（`docs/2026-08-15-gaea3-ui-design-system.md` + `design-system/gaea/MASTER.md` v1，
> 12 主题令牌、玻璃拟态、aurora）**全部保留为基底**；本 v2 在其上做**壳层革命 + 板块工作台革命**。
> 机器可读 source of truth：`design-system/gaea/MASTER.md`（v2 已更新）。
> 本文档是各板块子代理的统一改造指令（必读）。

---

## 0. 一句话概念

**从「10 个页面的窗口应用」革命为「一艘个人 AI 驾驶舱」**：
左侧**指挥轨道**（Command Rail）导航所有模块；顶部**轨道条**（Orbit Strip）承载
AI 状态与模型；底部**遥测轨道**（Telemetry Rail）实时显示引擎与资源；中央是各板块的
**3 分区可调工作台**。视觉语言升级为 **Luminous Glass 2.0**（玻璃 + 1px 渐变高光线 + 柔光）。

- 导航革命：顶栏横向菜单 → **左侧竖向指挥轨道**（icon dock，hover 展开标签，Ctrl+1~9 / ⌘K 优先）。
- 布局革命：每板块统一为「侧栏 zone + 主区 zone + inspector zone」的可调工作台。
- 状态革命：AI/引擎/资源状态从「角落里的小字」→ **底部实时遥测轨道**（面积 sparkline + 数字）。
- 视觉革命：Luminous Glass 2.0——轨道/卡片带顶部 1px 高光线、激活项带光条 + 光晕 orb。

---

## 1. 壳层规格（MainLayout 全部重做，由主代理实现）

### 1.1 指挥轨道 Command Rail（左，宽 48px → hover 展开 176px）
- 结构：`v3-rail`（flex column）：
  - 顶部：品牌 sigil（`/favicon.svg`，`--gaea-glow` drop-shadow）；
  - 中部：模块 dock —— `home` 恒首位，其余按 manifest.menuOrder（沿用 `getActiveMenuBoards`）；
    激活项 = `--color-primary-container` 底 + 左缘 3px 主色光条 + 8px 光晕 orb（呼吸 2s）；
  - 底部：设置、主题、折叠按钮。
- 每项 `aria-label` + tooltip；键盘：方向键/Enter（或沿用 Ctrl+1~9 与 ⌘K）。
- 图标继续用 `resolveBoardIcon`；hover 展开时显示 `b.label` 标签。

### 1.2 轨道条 Orbit Strip（顶部，高 38px）
- 左：面包屑（项目锚点 → 当前板块，沿用现有 breadcrumb 数据，等宽小字）；
- 中：模型 pill —— `--color-primary-container` 胶囊 + 呼吸灯（`--gaea-glow`）+ 模型名，
  点击打开模型切换（可先沿用现有 Tag → 升级为按钮样式）；AI 流式时 pill 呼吸加快；
- 右：搜索（⌘K，沿用 SearchModal）、4 主题色点（键盘可达）、明暗切换、设置。
- 玻璃底 + 底部 1px 高光线 `--v3-edge`。

### 1.3 遥测轨道 Telemetry Rail（底部，30px 折叠 / 140px 展开）
- 数据沿用现有 StatusBar 轮询（`GetModelMonitor` 3s）：engines + stats(cpu/mem/vram/gpu)。
- 折叠态：引擎 pods（本地已启用引擎徽标，点击 ⚡ 启停入口）＋ CPU/内存/GPU 当前值数字。
- 展开态：CPU/MEM/GPU 三条实时**面积 sparkline**（SVG 自绘，缓冲 ≤60s，`--v3-telemetry`
  主色、20% 填充、当前值数字 + 趋势文字；不只靠颜色）+ 项目写作进度（沿用 Progress）+ 模型负载条。
- 超载预警沿用现有 notification 节流逻辑；`prefers-reduced-motion` 下 sparkline 静态。

### 1.4 内容区
- 保留现有 pageRegistry / keepAlive / visitedPages / navigate 事件 / 快捷键机制，仅改布局容器。
- 板块区 = `v3-zone` 容器；padded 板块内容保持 16px 呼吸，full 板块（chat/gaea）铺满。

---

## 2. 板块工作台规格（子代理逐板块改造）

统一模式：**侧栏 zone + 主区 zone + inspector zone**，分区用玻璃面板（`v3-panel`）+ 分隔线
（`v3-split`）视觉；拖拽/缩放沿用各板块已有机制（无则 CSS resize / 不做拖动，只做布局）。

### 2.1 home 首页 →「任务指挥中心 Mission Control」
- 现状：`ModuleLauncher` 玻璃 HUD（orb + 卡片 + 遥测小字）。
- 目标：**Bento 网格** —— 顶部一排 AI 状态卡（活跃模型、引擎数、资源占用、今日统计）；
  中部模块卡片网格（保留现有模块卡数据/跳转，卡片带 aurora 水印 + 进入箭头微交互）；
  底部最近会话 / 记忆脉搏条。轨道 orb 保留但收进左上（不挡内容）。

### 2.2 chat 聊天 →「对话驾驶舱」
- 现状：`ChatPage` 左侧话题栏 + 消息流（右侧无 inspector）。
- 目标：**3 分区**：左 = 话题/会话栏（240px，`v3-panel`）；中 = 对话流（`v3-zone`，74ch 阅读宽度
  居中）；右 = **新增上下文/人格 inspector**（280px，可折叠；人格卡片、当前模型、上下文摘要、
  快捷建议）。流式发光与 Composer 玻璃化保留并升级为 v3 高光线。

### 2.3 novel 小说 →「世界构建工作台」
- 现状：`NovelPage` 顶部分类 tab + 各 tab 内容。
- 目标：**3 分区**：左 = 书架/大纲树栏；中 = 主视图（章节编辑器/角色卡/设定卡）；右 =
  章节树/属性 inspector（可折叠）。子导航保留（书架/设定/角色/创作/阅读/导出），但视觉收敛
  为轨道式细条（非大按钮）。编辑器区沿用 GhostText/流式，包一层 v3-card。

### 2.4 imagegen 绘梦 →「画廊工作台」
- 现状：`ImageGenPage` 顶部模式 nav + 左控制台(300px) + 中央画布，历史/队列在别处。
- 目标：**3 分区**：左 = 控制台 zone（300px，折叠式高级参数）；中 = 画布 zone（瀑布/单图预览）；
  右 = 新增历史/任务队列 inspector（280px，沿用 HistoryRail/TaskCenter 数据）。模式 nav 保留
  但收敛为轨道条细 tab。生成进度条升级为 v3 高光。

### 2.5 gaea 办公 →「生产力主战场」（最大板块，拆 2 个子代理）
- shell 子代理：`gaea/App.tsx` 布局（会话栏 | 过程卡流 | 工具/交付物 3 分区）、`Sidebar.tsx`、
  `styles.css`/`redesign.css`/`tailwind.css` 视觉升级为 v3（高光线、柔光、分区线）。
- chat 子代理：`Transcript.tsx`/`Message.tsx`/`Composer.tsx` 及 composer/*、StreamingIndicator、
  reasoning 展示 —— 消息层级、推理面板、Composer 聚焦光晕按 v3 规范升级（不改 props/行为）。

### 2.6 memoryhub 记忆中枢 →「记忆图谱舰桥」
- 现状：顶部 8 分类 tab + 列表/图谱双视图 + hub 科幻 bg（grid/scanline）。
- 目标：**左分类轨道**（竖向图标+标签，替代顶部横排）+ 主区视图 + 右详情 inspector。
  hub-grid/scanline 保留为氛围层（降透明度，避免抢戏）；图谱视图（GraphView）保持沉浸。

### 2.7 modelcenter 模型中心 →「引擎控制台」
- 现状：`ModelCenterPage` 头部 + 分类 tab + KPI/引擎卡/统计面板。
- 目标：**3 分区**：左 = 分类导航 + 绑定状态；中 = KPI 卡行 + 引擎卡网格；右 = 统计/图表
  inspector。`--mc-*` 命名空间已在 Wave 3 收敛，延续全局令牌；KPI 卡升级 v3-card 高光线。

### 2.8 characterlib 角色库 →「角色档案库」
- 现状：头部 + 工具栏 + 档案卡网格 + 详情抽屉。
- 目标：左 = 搜索/筛选栏（v3-panel）；中 = 档案卡网格（v3-card 高光线，进入箭头微交互）；
  右 = 详情 inspector（可折叠，替代/补充抽屉）。头部收敛（板块名已在轨道）。

### 2.9 settings 设置 →「控制室」
- 现状：头部 + 分类磁贴 + 表单区。
- 目标：左 = 分类导航（竖向，替代磁贴）；中 = 表单区（v3-zone）；右 =（空，可隐藏）。
  表单控件沿用 antd + v3 令牌；头部收敛为细条。

### 2.10 knowledge 知识库 →「知识舰桥」
- 现状：`KnowledgePage` 包 `KnowledgePanel`（variant=page）。
- 目标：左 = 知识分类/列表栏；中 = 详情/编辑；右 = 检索结果/引用 inspector。
  搜索框聚焦光晕按 v3 规范。

---

## 3. 实现红线（所有子代理必须遵守）

1. **文件边界**：只改自己板块目录下的文件；**禁止触碰**：
   `src/App.tsx`、`src/main.tsx`、`src/layouts/MainLayout.tsx`、`src/index.css`、
   `src/stores/appStore.ts`、`src/boards/*`、`src/wailsjsCompat.ts`、`src/gaea/lib/bridge.ts`、
   `src/events.ts`、`wailsjs/`、Go 文件。若确有跨板块共享组件需要调整，先记入报告不要改。
2. **API 不变**：组件 props / 对外导出 / 事件名 / 后端绑定调用一律不变（改样式与布局容器，
   不改行为与契约）；测试若因视觉类 class 变化断言失败，仅当断言的是已被替换的视觉结构
   时才可同步更新测试。
3. **令牌纪律**：零硬编码 hex；v3 派生令牌（`--v3-*`）已由主代理在 `src/v3/foundation.css`
   定义，直接消费；`--md-sys-*`/`--gaea-*`/`--color-*` 沿用。
4. **可访问性**：焦点环可见、aria-label 齐全、`cursor-pointer`、reduced-motion 降级
   （动画用 class 包裹在 `@media (prefers-reduced-motion: reduce)` 或 `.ui-reduced-motion` 下关停）。
5. **图标**：antd icons only，禁 emoji 图标；装饰图标 `aria-hidden`。
6. **自检**：改完跑 `npx tsc --noEmit`（全项目只读检查，必须通过，不得新增报错）+
   `npx vitest run <本板块测试文件>`；不要跑 `npm run build`（最终由主代理统一构建）。
7. **报告**：结束时输出：改动文件清单、布局结构变化说明、自检结果、遗留问题。

---

## 4. 交付物

- `design-system/gaea/MASTER.md`（v2，已完成）
- 本文档（v2 规格，已完成）
- `frontend/src/v3/foundation.css`（v3 令牌 + 壳层/分区/卡片原语，主代理）
- `MainLayout.tsx` 新壳层（主代理）
- 10 个板块工作台改造（子代理，按本文件 §2）
- 版本号：`wails.json` productVersion → 3.0.0；`frontend/package.json` → 3.0.0（主代理收尾）
