# Design System Master File — gaea 3.0「星枢 Constellation OS」(v2)

> **LOGIC:** When building a specific page, first check `design-system/gaea/pages/[page-name].md`.
> If that file exists, its rules **override** this Master file.
> If not, strictly follow the rules below.
>
> v2（2026-08-15，用户指令「革命性重设计」）：从 v1「智能玻璃 · 有温度的仪表」
> **升级为「星枢 Constellation OS」——个人 AI 驾驶舱**。壳层由「顶栏菜单」革命为
> 「左侧指挥轨道 + 顶部轨道条 + 底部遥测轨道」；每板块统一为 3 分区可调工作台。
> 完整规格见 `docs/2026-08-15-gaea3-ui-constellation-os.md`。

---

**Project:** gaea
**Version:** 3.0 Constellation OS (v2)
**Generated:** 2026-08-15（ui-ux-pro-max skill → gaea 革命性定制 v2）

---

## Global Rules

### 0. 设计理念（Design Thesis）

> gaea 3.0 = 本地优先的个人 AI 智能体平台。v2 视觉语言：
> **「星枢驾驶舱」** —— 应用不再是「10 个页面」，而是一艘以 AI 内核为中心的座舱：
> 左侧**指挥轨道**（模块导航 dock）、顶部**轨道条**（AI 状态与模型）、底部**遥测轨道**
> （引擎与资源实时遥测）、中央**板块工作台**（3 分区可调）。氛围层保留 aurora 深空，
> 容器层升级为 **Luminous Glass 2.0**（玻璃 + 1px 渐变高光线 + 柔光），
> 信息层用实底卡。12 套主题（6 色系 × 明暗）令牌体系完整保留——换主题 = 换令牌表。

三条原则：
1. **座舱分层**：氛围层（aurora）→ 轨道层（rail/strip/telemetry，玻璃+高光）→ 工作台层
   （3 分区 zone）→ 信息层（实底卡）。玻璃只做容器，文字永远落在高对比表面。
2. **命令优先、键盘可达**：导航第一入口是左侧指挥轨道 + Ctrl+1~9 + ⌘K；一切操作
   有可见焦点环与 aria 标签。
3. **AI 可见、可感知**：模型 pill、流式呼吸灯、遥测 sparkline、过程卡——AI 状态
   在轨道条/遥测轨道上一眼可读，与静态内容明确区分。

### Color Palette（令牌化，跟随 12 主题；示例 nightJade 暗）

| Role | Token | Example (nightJade dark) | Usage |
|------|-------|--------------------------|-------|
| Primary | `--color-primary` | `#2dd4bf` | 主按钮/激活/链接/轨道激活 |
| On Primary | `--color-on-primary` | `#042f2e` | 主色上的文字 |
| Primary Container | `--color-primary-container` | `#134e4a` | 徽标/高亮容器 |
| Surface | `--color-surface` | `#0a1014` | 页面底 |
| Surface Container | `--color-surface-container` | `#0f1a20` | 卡片/面板 |
| Surface Container High | `--color-surface-container-high` | `#14242e` | 悬浮 |
| Surface Container Highest | `--color-surface-container-highest` | `#1b2f3a` | 弹窗顶层 |
| Text | `--color-text` | `#e2e8f0` | 正文 |
| Text Secondary | `--color-text-secondary` | `#94a3b8` | 辅助文字 |
| Border | `--color-border` | `rgba(255,255,255,0.06)` | 描边 |
| Success / Warning / Destructive | `--color-success/-warning/-destructive` | `#34d399/#f59e0b/#ef4444` | 语义状态 |
| Accent RGB | `--accent-rgb` | `45,212,191` | rgba 发光/水印 |
| Glow | `--gaea-glow` | `#5eead4` | AI 状态发光 |
| Glass BG | `--gaea-glass-bg` | `rgba(15,26,32,0.62)` | 玻璃面板 |
| Aurora BG | `--gaea-aurora-bg` | radial 渐变 | 氛围背景 |

**v2 新增令牌（Luminous Glass 2.0，由现有令牌派生，禁止硬编码 hex）**

| Role | Token | 派生方式 | Usage |
|------|-------|----------|-------|
| 高光线 | `--v3-edge` | `linear-gradient(180deg, color-mix(in srgb, var(--color-text) 14%, transparent), transparent 60%)` | 轨道/卡片顶部 1px 高光 |
| 柔光 | `--v3-glow-soft` | `0 0 0 1px color-mix(in srgb, var(--gaea-glow) 12%, transparent), 0 6px 28px color-mix(in srgb, var(--gaea-glow) 10%, transparent)` | 悬浮/激活柔光 |
| 轨道底 | `--v3-rail-bg` | `color-mix(in srgb, var(--gaea-glass-bg) 78%, var(--color-surface))` | 指挥轨道背景 |
| 分区线 | `--v3-split` | `1px solid color-mix(in srgb, var(--color-border) 70%, transparent)` | 分区分隔 |
| 遥测色 | `--v3-telemetry` | `var(--gaea-glow)` | 遥测 sparkline 主色 |

**Color Notes:** 组件零硬编码色值；正文对比度 ≥4.5:1、次要 ≥3:1；状态三重传达。

### Typography

- **Body:** 14px 默认（用户可调 12-20），line-height 1.5+；`FONT_OPTIONS`：system /
  微软雅黑 / 思源黑体 / 宋体衬线 / 等宽；中文推荐增强 = Noto Sans SC。
- **Heading:** 16/18/22px 阶梯；轨道条/遥测数据走等宽栈（`font-variant-numeric: tabular-nums`）。
- 轨道图标配 10-12px 标签；标题字号阶梯 16/18/22。

### Spacing / Radius / Elevation / Motion

| Token | Value | Usage |
|-------|-------|-------|
| `--radius-sm/md/lg/xl` | 8/12/16/28px | 按钮/卡/面板/弹窗 |
| `--elevation-1..5` | 5 级 | 卡→悬浮→弹窗→抽屉→模态 |
| 间距 | 4/8/12/16/24/32px | 卡内 16、分区 24、轨道 8 |
| `--transition-fast/normal/slow` | 200/300/400ms | hover 200、轨道展开 300、整页 400 |

---

## Shell Specs（星枢壳层）

### Command Rail（左侧指挥轨道）
```css
.v3-rail {
  width: 48px; /* hover/聚焦展开至 176px，300ms */
  background: var(--v3-rail-bg);
  backdrop-filter: blur(24px) saturate(150%);
  border-right: var(--v3-split);
  display: flex; flex-direction: column;
}
```
- 顶部：品牌 sigil（`/favicon.svg`，`--gaea-glow` drop-shadow）。
- 中部：模块 dock（home 恒首位）。激活项 = `--color-primary-container` 底 +
  左缘 3px 主色光条 + 光晕 orb（8px 圆点，`--gaea-glow` 呼吸 2s，reduced-motion 静态）。
- 底部：设置 / 主题 / 折叠按钮。
- 每项含 `aria-label` + tooltip；hover 展开时显示标签。

### Orbit Strip（顶部轨道条，38px）
- 左：面包屑（项目锚点 → 板块），等宽小字；
- 中：模型 pill（`--color-primary-container` 胶囊，呼吸灯 + 模型名，点击弹模型切换）+ AI 状态；
- 右：搜索（⌘K）、主题色点（4 色块，键盘可达）、明暗切换、设置。
- 玻璃底 + 底部 `--v3-edge` 高光线。

### Telemetry Rail（底部遥测轨道）
- 折叠态 30px：引擎 pods（已启用本地引擎徽标，⚡ 一键启停）+ 当前值（CPU/MEM/GPU 数字）。
- 展开态 140px：实时面积 sparkline（≤60s 缓冲，SVG 无新依赖，`--v3-telemetry` 主色、
  20% 透明度填充、当前值文字 + 数字，不只靠颜色）+ 项目写作进度环 + 模型负载条。
- 超载（CPU/GPU>85%、内存>90%）→ 徽标变 warning 色 + notification（沿用现有 60s 节流）。
- `prefers-reduced-motion`：sparkline 静态（不逐帧刷新）。

### 板块工作台（3 分区可调）
- 每板块 = `v3-zone`（主区）+ 侧区（左/右/底部 inspector），分区用 `v3-panel` 玻璃容器，
  分隔条视觉 `v3-split`（各板块沿用/新增现有拖拽逻辑或 CSS resize）。
- 板块入口统一：无重复页面头（板块名已在轨道/面包屑），信息层实底卡不叠玻璃。

---

## Component Specs

### Luminous Card（v3-card）
```css
.v3-card {
  background: var(--color-surface-container);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: inset 0 1px 0 color-mix(in srgb, var(--color-text) 6%, transparent);
  /* hover: --elevation-2 + --v3-glow-soft + translateY ≤2px（reduced-motion 跳过） */
}
```

### Glass Panel（v3-panel，轨道与容器基座）
- `--gaea-glass-bg` + `backdrop-filter: blur(24px) saturate(150%)` + `--v3-edge` 顶部高光 +
  `--radius-lg`。信息层用实底卡，不叠玻璃。

### Buttons / Inputs / Status / Icons
- 沿用 v1 规范：primary/secondary/danger；focus ring 可见；错误就近 + 色/图标/文；
  antd icons only、禁 emoji 图标、装饰 aria-hidden、纯图标按钮带 aria-label；
  AI 流式 = glow 呼吸（2s）+ 流式光标；loading = skeleton/spinner + 文案。

---

## Style Guidelines

**Style:** 星枢驾驶舱 Constellation OS（Luminous Glass 2.0 + 轨道布局）
**Keywords:** 座舱、轨道、遥测、深空 aurora、玻璃 + 1px 高光线、命令优先、键盘可达、
主题即身份、本地优先质感
**Key Effects:** 轨道激活光条/光晕 orb、玻璃高光线、sparkline 遥测、150-300ms 微交互
（位移 ≤2px，compositor-only）、GSAP 入场（reduced-motion 降级）

---

## Anti-Patterns（Do NOT Use）

- ❌ 组件内硬编码色值（必须走令牌，v3 派生令牌也禁止手写 hex）
- ❌ Emoji 当图标
- ❌ 只靠颜色传达状态（sparkline 必须有数字/文字）
- ❌ 无焦点环（键盘用户）；命令轨道键盘不可达
- ❌ 忽略 `prefers-reduced-motion`
- ❌ hover 动画改布局属性（width/height/margin）——轨道展开用 transform/自定义属性
- ❌ 玻璃上叠玻璃（信息层必须实底）
- ❌ 正文对比度 < 4.5:1
- ❌ 板块内重复大标题（板块名已在轨道/面包屑）

---

## Pre-Delivery Checklist

- [ ] 正文对比度 ≥4.5:1 / 次要 ≥3:1（12 主题抽查）
- [ ] 键盘全可达：rail 方向键/Enter 导航、⌘K、焦点环可见
- [ ] reduced-motion 降级（orb 静态、sparkline 静态、入场淡入）
- [ ] 无 emoji 图标、图标集一致（antd）
- [ ] 可点元素 cursor-pointer、hover ≤2px
- [ ] 状态三重传达（色/图标/文）
- [ ] 明暗双模式正常（12 套主题）
- [ ] 无硬编码色值（含 v3 派生令牌）
- [ ] 遥测数值与引擎状态真实联动（非假数据）
