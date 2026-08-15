# Design System Master File — gaea 3.0（定制版）

> **LOGIC:** When building a specific page, first check `design-system/gaea/pages/[page-name].md`.
> If that file exists, its rules **override** this Master file.
> If not, strictly follow the rules below.
>
> 本文由 ui-ux-pro-max skill 生成初版（Cyberpunk/NFT 分类）后，经 gaea 现状定制覆盖：
> gaea 是「本地优先的个人 AI 智能体平台」（Wails 桌面 + React 18 + AntD 5 + 12 套 Material 3
> 风格主题），视觉语言 = **智能玻璃 · 有温度的仪表**（aurora 氛围 + 玻璃容器 + 主题身份）。
> 详细设计与实施见 docs/2026-08-15-gaea3-ui-design-system.md。

---

**Project:** gaea
**Version:** 3.0 定制版 v1
**Generated:** 2026-08-15（ui-ux-pro-max → gaea 定制）

---

## Global Rules

### Color Palette（令牌化，跟随 12 主题；示例 nightJade 暗）

| Role | Token | Example (nightJade dark) | Usage |
|------|-------|--------------------------|-------|
| Primary | `--color-primary` | `#2dd4bf` | 主按钮/激活/链接 |
| On Primary | `--color-on-primary` | `#042f2e` | 主色上的文字 |
| Primary Container | `--color-primary-container` | `#134e4a` | 徽标/高亮容器 |
| Surface | `--color-surface` | `#0a1014` | 页面底 |
| Surface Container | `--color-surface-container` | `#0f1a20` | 卡片/面板 |
| Surface Container High | `--color-surface-container-high` | `#14242e` | 悬浮 |
| Surface Container Highest | `--color-surface-container-highest` | `#1b2f3a` | 弹窗顶层 |
| Text | `--color-text` | `#e2e8f0` | 正文 |
| Text Secondary | `--color-text-secondary` | `#94a3b8` | 辅助文字 |
| Border | `--color-border` | `rgba(255,255,255,0.06)` | 描边 |
| Success | `--color-success` | `#34d399` | 成功 |
| Warning | `--color-warning` | `#f59e0b` | 警告 |
| Destructive | `--color-destructive` | `#ef4444` | 破坏性操作（Wave 1 补令牌） |
| Accent RGB | `--accent-rgb` | `45,212,191` | rgba 发光/水印 |
| Glow | `--gaea-glow` | `#5eead4` | AI 状态发光 |
| Glass BG | `--gaea-glass-bg` | `rgba(15,26,32,0.62)` | 玻璃面板 |
| Aurora BG | `--gaea-aurora-bg` | radial 渐变 | 氛围背景 |
| Destructive | `--color-destructive` | `#ef4444` 暗 / `#dc2626` 亮 | 破坏性操作（Wave 1 已落地，antd colorError 对齐） |

**Color Notes:** 主题即身份（6 色系 × 明暗）；组件零硬编码色值；正文对比度 ≥4.5:1。
**死令牌治理（Wave 2+）:** `--ds-*` 仅 2/14 定义（27 处消费失效）→ 定义或迁移；`--mc-*` 私有命名空间
125 处 → 迁移全局令牌；tailwind `--color-border` 与 legacy shim 同名 → 消歧；主题色值 3 处重复 → 单源派生。

### Typography

- **Body:** 14px 默认（用户可调 12-20），line-height 1.5+；`FONT_OPTIONS`：system / 微软雅黑 /
  思源黑体 / 宋体衬线 / 等宽；中文推荐增强 = Noto Sans SC（skill 中文配对）。
- **Heading:** 16/18/22px 阶梯；代码/数据走等宽栈。

### Spacing / Radius / Elevation / Motion（沿用现有令牌）

| Token | Value | Usage |
|-------|-------|-------|
| `--radius-sm/md/lg/xl` | 8/12/16/28px | 按钮/卡/面板/弹窗 |
| `--elevation-1..5` | 5 级 | 卡→悬浮→弹窗→抽屉→模态 |
| 间距 | 4/8/12/16/24/32px | 卡内 16、板块 24、布局 32 |
| `--transition-fast/normal/slow` | 200/300/400ms | hover 200、展开 300、整页 400 |

---

## Component Specs

### Glass Panel（容器基座，所有板块）
```css
.gp-panel {
  background: var(--glass-bg);
  backdrop-filter: blur(24px) saturate(150%);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}
```
面板内信息层用实底卡（`--color-surface-container`），不叠玻璃。

### Content Card
- bg `--color-surface-container`, radius `--radius-md`, border `--color-border`;
- hover: `--elevation-2` + translateY ≤2px（reduced-motion 跳过）。

### Buttons
- Primary: `--color-primary` bg + `--color-on-primary` text, radius sm;
- Secondary: transparent + primary border/text; Danger: destructive red;
- Feedback: hover y≤1px + opacity .9, 150-200ms; explicit disabled/loading.

### Inputs / Forms
- Labels always visible; errors near field (color + icon + text);
- Focus ring `--focus-ring`; `cursor-pointer` on all clickables.

### Status Feedback
- Semantic color + icon + text（not color-only）;
- AI streaming: glow pulse（2s 呼吸）+ 流式光标; loading: skeleton/spinner + label.

### Icons
- antd icons only; no emoji-as-icon; decorative `aria-hidden`; icon-only buttons labeled.

---

## Style Guidelines

**Style:** 智能玻璃 · 有温度的仪表（Smart Glass / Warm Instrument）
**Keywords:** aurora 深空渐变、玻璃拟态、主题即身份、AI 发光、克制动效、本地优先质感
**Best For:** 个人 AI 智能体平台、生产力工作台、多域合一桌面应用
**Key Effects:** aurora 渐变背景、backdrop-blur 玻璃面板、主题色 glow（AI 状态）、
150-300ms 微交互（位移 ≤2px，compositor-only）

### Page Pattern（每板块见 pages/*.md）
壳层（顶栏/侧栏/面包屑）→ 板块主视图（列表/画布/面板）→ 状态与反馈层。

---

## Anti-Patterns（Do NOT Use）

- ❌ 组件内硬编码色值（必须走令牌）
- ❌ Emoji 当图标
- ❌ 只靠颜色传达状态（无图标/文字）
- ❌ 无焦点环（键盘用户）
- ❌ 忽略 `prefers-reduced-motion`
- ❌ hover 动画改布局属性（width/height/margin）
- ❌ 玻璃上叠玻璃（信息层必须实底）
- ❌ 正文对比度 < 4.5:1（尤其浅色主题次要文字）

---

## Pre-Delivery Checklist

- [ ] 正文对比度 ≥4.5:1 / 次要 ≥3:1（12 主题抽查）
- [ ] 键盘全可达、焦点环可见
- [ ] reduced-motion 降级
- [ ] 无 emoji 图标、图标集一致
- [ ] 可点元素 cursor-pointer、hover ≤2px
- [ ] 状态三重传达（色/图标/文）
- [ ] 明暗双模式正常
- [ ] 无硬编码色值
