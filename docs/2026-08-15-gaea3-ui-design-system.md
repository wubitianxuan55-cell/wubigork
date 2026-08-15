# gaea 3.0 UI 设计系统（定制版）

> 状态：**定稿 v1**（2026-08-15，ui-ux-pro-max skill 驱动 + gaea 现状定制）
> 定位：本文是 gaea 3.0「个人 AI 智能体平台」的视觉与交互规范；配套机器可读文件见
> `design-system/gaea/MASTER.md`（skill 持久化 source of truth）；各板块蓝图见
> `design-system/gaea/pages/*.md`（页面覆盖优先于 MASTER）。
> 与现状的关系：**沿用并统一**现有 12 套主题（6 色系 × 明暗，Material 3 风格令牌 +
> 玻璃拟态 + aurora 渐变），在此基底上建立**统一的设计语言与组件规范**——不是推翻
> 主题系统，而是让 10 个板块长成同一套设计语言。

---

## 0. 设计理念（Design Thesis）

> gaea 3.0 = **本地优先的个人 AI 智能体平台**。视觉语言的关键词：
> **「智能玻璃 · 有温度的仪表」** —— 深空渐变（aurora）承载氛围，玻璃拟态面板承载
> 内容，主题色（每套 6 色系）承载品牌，强调色（accentRgb）承载焦点与动作。

三条设计原则：
1. **氛围与内容分层**：aurora 背景（氛围层）→ glass 面板（容器层）→ 内容卡/文本
   （信息层）。玻璃只做容器，不做信息载体（文字永远落在高对比表面上）。
2. **主题即身份，令牌即契约**：所有颜色/圆角/阴影/动效走 CSS 变量令牌，组件零硬编码
   色值；换主题 = 换令牌表，代码不动。
3. **AI 可见、可感知**：AI 生成/流式/推理状态有专属视觉语言（发光脉冲、流式光标、
   过程卡），与静态内容可一眼区分。

---

## 1. 设计令牌（Design Tokens）

> 全部沿用现有 `appStore.ts ThemeTokens` + `index.css` 变量，新增/规范如下。

### 1.1 颜色（跟随 12 主题，示例 = nightJade）

| 角色 | 令牌 | 暗色值（示例） | 用途 |
|---|---|---|---|
| 主色 | `--color-primary` | `#2dd4bf` | 主按钮/激活/链接/选中 |
| 主色上文字 | `--color-on-primary` | `#042f2e` | 主色上的文字/图标 |
| 主容器 | `--color-primary-container` | `#134e4a` | 徽标/高亮容器 |
| 表面 | `--color-surface` | `#0a1014` | 页面底 |
| 表面容器 | `--color-surface-container` | `#0f1a20` | 卡片/面板 |
| 容器高 | `--color-surface-container-high` | `#14242e` | 悬浮/层级+1 |
| 容器最高 | `--color-surface-container-highest` | `#1b2f3a` | 弹窗/顶层 |
| 正文 | `--color-text` | `#e2e8f0` | 主要文字 |
| 次要文字 | `--color-text-secondary` | `#94a3b8` | 辅助/说明 |
| 描边 | `--color-border` | `rgba(255,255,255,0.06)` | 分隔/描边 |
| 成功 | `--color-success` | `#34d399` | 成功状态 |
| 警告 | `--color-warning` | `#f59e0b` | 警告状态 |
| 强调 RGB | `--accent-rgb` | `45,212,191` | rgba() 构建发光/水印 |
| 发光 | `--glow` | `#5eead4` | AI 状态发光 |
| 玻璃底 | `--glass-bg` | `rgba(15,26,32,0.62)` | 玻璃面板背景 |

**新增规范（落地 Wave 1 补充）**：
- `--color-destructive`（破坏性操作红，各主题统一 `#ef4444` 暗 / `#dc2626` 亮）
- `--focus-ring`（焦点环：`0 0 0 2px var(--color-bg-layout), 0 0 0 4px var(--color-primary)`）
- 对比度契约：正文 ≥ 4.5:1、次要文字 ≥ 3:1、主色上文字 ≥ 3:1（对照现有令牌逐主题核验）

### 1.2 圆角 / 阴影 / 间距 / 动效（现有令牌，规范用法）

| 令牌 | 值 | 规范用途 |
|---|---|---|
| `--radius-sm` 8px | 按钮/输入/徽标/小卡 |
| `--radius-md` 12px | 卡/列表项/工具卡 |
| `--radius-lg` 16px | 大面板/侧栏 |
| `--radius-xl` 28px | 弹窗/模态/大图卡 |
| `--elevation-1..5` | 悬浮层级：1 默认卡 / 2 悬浮 / 3 弹窗 / 4 抽屉 / 5 模态顶层 |
| 间距 | 4/8/12/16/24/32px 阶梯 | 卡内 16px、板块间 24px、布局级 32px |
| 动效 | 200/300/400ms `cubic-bezier(0.2,0,0,1)` | hover 150-250ms、展开 300ms、整页 400ms |

### 1.3 排版

- 字体：沿用 `FONT_OPTIONS`（system / 微软雅黑 / 思源黑体 / 宋体衬线 / 等宽）；默认系统。
- 正文：14px（`fontSize` 默认 14，可 12-20），行高 1.5+；标题：16/18/22px 阶梯；
  代码/数据：等宽字体栈。
- 中文场景以「思源黑体（Noto Sans SC）」为推荐增强（skill 中文配对建议），系统回退。

---

## 2. 组件规范（Component Specs）

> 目标：10 个板块共享一套视觉组件语义。落地 Wave 2 起逐步收敛。

### 2.1 玻璃面板（Glass Panel）—— 所有板块的容器基座
```css
.gp-panel {
  background: var(--glass-bg);
  backdrop-filter: blur(24px) saturate(150%);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}
```
- 用于：侧栏、会话列表、内容面板、属性栏。
- 规则：面板内不再叠玻璃，信息层用 `surface-container` 实底卡。

### 2.2 内容卡（Content Card）
- 背景 `var(--color-surface-container)`、圆角 `--radius-md`、描边 `--color-border`、
  悬浮 `--elevation-2` + 位移 ≤2px（respect reduced-motion）。

### 2.3 按钮
- 主按钮：`--color-primary` 底 + `--color-on-primary` 字，圆角 `--radius-sm`；
- 次级：透明 + 主色描边/文字；危险：destructive 红；
- 反馈：hover 位移 ≤1px + 透明度 0.9（150-200ms，skill 微交互 Subtle 档）；
- 禁用/加载态显式。

### 2.4 输入 / 表单
- 标签恒可见（不依赖 placeholder）；错误就近显示 + 颜色 + 图标双通道（不只靠颜色）；
- focus 显示 `--focus-ring`；`cursor-pointer` 于一切可点元素。

### 2.5 状态反馈
- 成功/警告/错误用语义色 + 图标 + 文字（不只靠颜色传达）；
- AI 流式：发光脉冲（`--glow` box-shadow 呼吸，2s 循环）+ 流式光标；
- 加载：骨架屏或 spinner + 文字说明；中断可恢复（对齐既有取消/重试模式）。

### 2.6 图标
- 全部 antd 图标（`@ant-design/icons`，现有）；禁 emoji 当图标（skill 反模式）；
- 装饰性图标 `aria-hidden`；纯图标按钮带 `aria-label`/tooltip。

---

## 3. 导航与壳层（3.0 §5.2 Manifest 配套视觉）

- 顶栏：玻璃（`blur(28px) saturate(160%)`）+ 主题色发光线；左：品牌 + 板块面包屑；
  中：功能模型指示；右：模型状态/语音/设置入口。
- 侧栏（chat/gaea 等）：`gp-panel`，激活项 = `--color-primary-container` 高亮 +
  主色左缘 3px 指示条；hover 项 = `--color-surface-container-high`。
- 首页启动器（ModuleLauncher）：维持现有玻璃 HUD，卡片 = 内容卡 + aurora 水印 +
  进入箭头微交互（右移 2px，200ms）。

---

## 4. 各板块设计语言速查（详见 pages/*.md）

| 板块 | 视觉性格 | 关键元素 |
|---|---|---|
| home 首页 | 中枢/仪式感 | 语音晶核 orb + 模块卡片 + HUD 遥测 |
| chat 聊天 | 沉浸对话 | 左侧会话玻璃栏 + 消息流 + 流式发光 |
| novel 小说 | 创作工作台 | 书架卡 + 编辑器 + 章节/角色分栏 |
| imagegen 绘梦 | 画廊工作台 | 历史瀑布 + 生成队列 + 参数面板 |
| gaea 办公 | 生产力主战场 | 会话栏 + 过程卡流 + 工具/交付物面板 |
| memoryhub 记忆中枢 | 图谱/知识 | 分类 Tab + 列表/图谱双视图 |
| modelcenter 模型中心 | 控制台 | KPI 卡 + 引擎卡 + 统计面板 |
| characterlib 角色库 | 档案库 | 档案卡网格 + 详情抽屉 |
| settings 设置 | 克制工具 | 分组导航 + 表单 |
| knowledge 知识库 | 文档库 | 面板 + 导入/检索 |

---

## 5. 可访问性与质量红线（Pre-Delivery Checklist）

- [ ] 正文对比度 ≥ 4.5:1；次要 ≥ 3:1（逐主题抽查）
- [ ] 键盘全可达：tab 顺序 = 视觉顺序，焦点环可见（`--focus-ring`）
- [ ] `prefers-reduced-motion`：动画降级为静态/淡入
- [ ] 无 emoji 图标；图标集一致（antd icons）
- [ ] 一切可点元素 `cursor-pointer`；hover 位移 ≤2px
- [ ] 颜色 + 图标 + 文字三重传达状态
- [ ] 明暗双模式均正常（12 套主题全验证）
- [ ] 无硬编码色值（组件内色值全部改令牌）

---

## 6. 实施路线（分 Wave，每 Wave 独立提交 + 门禁）

- **Wave 1（令牌层）**：index.css 补充 `--color-destructive`/`--focus-ring` 等缺失令牌；
  appStore 主题校验对比度；清点组件内硬编码色值清单。
- **Wave 2（壳层）**：MainLayout 顶栏/侧栏/首页启动器按 §3 收敛（玻璃面板 + 激活态 +
  focus 环）。
- **Wave 3+（板块）**：按 pages/*.md 蓝图逐板块改造，每板块一批次提交。
- 每 Wave 验收：tsc/eslint/vitest 全绿 + 12 主题冒烟（关键页面截屏对比）+ 无回归。
