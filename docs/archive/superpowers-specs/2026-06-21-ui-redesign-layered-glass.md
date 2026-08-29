# wubigork UI 重设计 — "Layered Glass"

**日期:** 2026-06-21
**风格:** 现代科技 (Modern Tech)
**设计语言:** Layered Glass — 多层磨砂玻璃叠加于深色渐变基底

---

## 1. 设计语言系统

### 1.1 色彩体系

保留现有 4 套主题色 (nightGreen / starPurple / warmAmber / minimalGray)，统一深色基底：

**CSS 自定义属性（新增）：**

```css
:root {
  /* 背景层级 */
  --bg-deep: #08080a;
  --bg-base: #0d0d12;
  --bg-elevated: rgba(255, 255, 255, 0.04);
  --bg-glass: rgba(255, 255, 255, 0.06);

  /* 边框 */
  --border-subtle: rgba(255, 255, 255, 0.06);
  --border-accent: rgba(var(--accent-rgb), 0.15);

  /* 阴影 */
  --shadow-sm: 0 1px 3px rgba(0, 0, 0, 0.4);
  --shadow-md: 0 4px 12px rgba(0, 0, 0, 0.5);
  --shadow-lg: 0 8px 24px rgba(0, 0, 0, 0.6);
  --shadow-glow: 0 0 20px rgba(var(--accent-rgb), 0.15);

  /* 圆角 */
  --radius-sm: 4px;
  --radius-md: 8px;
  --radius-lg: 12px;
  --radius-xl: 16px;

  /* 动画 */
  --transition-fast: 100ms ease-out;
  --transition-normal: 150ms ease-out;
  --transition-slow: 200ms ease-out;
}
```

**4 套主题色升级为渐变：**

| 主题 | 主色 | 渐变 |
|------|------|------|
| nightGreen | #4ade80 | `linear-gradient(135deg, #4ade80, #22c55e)` |
| starPurple | #a78bfa | `linear-gradient(135deg, #a78bfa, #8b5cf6)` |
| warmAmber | #f59e0b | `linear-gradient(135deg, #f59e0b, #d97706)` |
| minimalGray | #9ca3af | `linear-gradient(135deg, #9ca3af, #6b7280)` |

每套主题需设置对应的 `--accent-rgb`（用于 rgba 构建）。

### 1.2 间距系统（8dp）

`4, 8, 12, 16, 20, 24, 32, 48, 64`

全局 padding/gap 统一到这些值。

### 1.3 动画令牌

- 进入动画：150ms ease-out
- 退出动画：100ms ease-in
- 悬停过渡：200ms ease-out
- 弹簧动画：`cubic-bezier(0.16, 1, 0.3, 1)` 用于模态框/面板
- 尊重 `prefers-reduced-motion`

---

## 2. 布局外壳 (MainLayout)

### 2.1 顶栏

- 玻璃态：`backdrop-filter: blur(12px)` + `background: var(--bg-glass)`
- 底部边框：`1px solid var(--border-subtle)`
- 高度保持 48px
- 导航菜单：去掉 Ant Design 默认下划线指示器，改用半透明背景高亮 + 左侧 2px 主题色竖线
- 主题色点：保持 4 色圆形，尺寸略微放大到 18px
- 明暗切换：保持

### 2.2 底栏

- 玻璃态同顶栏
- 进度条加发光效果
- 信息项间距增大到 16px

### 2.3 XAI 控制台

- 去等宽字体 → 系统字体 11px
- 玻璃态浮层面板：`background: var(--bg-glass)` + `backdrop-filter: blur(12px)` + `border-radius: var(--radius-lg)`
- 浮动入场：`@keyframes slideInRight`（已有，保留）
- 关闭按钮改为半透明圆形按钮
- 状态指示改为彩色圆点替代 Tag

### 2.4 页面背景

从纯黑 → 深色渐变：
```css
background: linear-gradient(180deg, var(--bg-deep) 0%, var(--bg-base) 100%);
```

---

## 3. 页面重设计

### 3.1 HomePage（项目书架）

- 项目卡片：玻璃面板 + `border-radius: var(--radius-lg)` + `box-shadow: var(--shadow-sm)`
- 悬停：`transform: translateY(-2px)` + `box-shadow: var(--shadow-md)` + `transition: var(--transition-slow)`
- 卡片内容：书名 + 类型 Tag + 迷你进度条 + 字数统计
- 空状态：品牌 logo 水印（大号半透明 SVG） + "Ctrl+N 新建项目" 快捷键提示
- 新建项目按钮：主题色渐变 + `box-shadow: var(--shadow-glow)`
- 脑暴区域：玻璃态 Modal，背景模糊

### 3.2 ChapterPage（写作页）— 最核心

**左侧大纲树：**
- 玻璃态面板，`border-radius: var(--radius-lg)`
- 选中项：左侧 3px 主题色竖线 + 半透明主题色背景
- 叶子节点小圆点状态指示

**标签栏：**
- 自定义 pill 样式标签（不用 Ant Design 默认卡片风格）
- 激活态：主题色背景 + 白色文字
- 非激活态：透明 + 半透明文字
- 可横向滚动（已完成）
- 关闭按钮 + 未保存确认（已完成）

**编辑器区：**
- 内凹 inset shadow 制造"沉入页面"感
- 背景稍深于面板
- 光标颜色跟随主题

**工具栏：**
- 按钮统一样式：半透明背景 + 柔和边框 + 悬停发光
- 生成按钮：主题色渐变 + 发光阴影

### 3.3 全局卡片/面板样式

应用到 WorldviewPage, CharacterPage, OutlinePage, CanvasPage, SettingsPage, ExportPage：

- 所有卡片/面板：`background: var(--bg-glass)` + `backdrop-filter: blur(8px)` + `border: 1px solid var(--border-subtle)` + `border-radius: var(--radius-lg)` + `box-shadow: var(--shadow-sm)`
- 输入框：`background: rgba(255,255,255,0.05)` + `border: 1px solid var(--border-subtle)` + `border-radius: var(--radius-md)` + 聚焦时主题色边框
- 主按钮：主题色渐变 + `box-shadow: var(--shadow-glow)` + 悬停时亮度提升
- 次按钮：透明 + `border: 1px solid var(--border-subtle)` + 悬停时边框变亮
- 分区标题：小号大写 + `letter-spacing: 0.05em` + 底部细线分割
- 空状态：品牌水印

### 3.4 对话框/模态框

- 玻璃面板 + `backdrop-filter: blur(20px)`
- 圆角 `var(--radius-xl)`
- 入场：`cubic-bezier(0.16, 1, 0.3, 1)` 弹簧效果
- 遮罩：`rgba(0, 0, 0, 0.6)`

---

## 4. 技术实施策略

### 4.1 不引入新依赖

完全在现有技术栈内实现（React 19 + Ant Design 6 + CSS 变量 + 内联 style）。

### 4.2 实施顺序

1. **CSS 变量层** — 在 `appStore.ts` ThemeTokens 中扩展新的 CSS 变量，在 `App.tsx` 中同步到 `:root`
2. **MainLayout** — 重做顶栏、底栏、控制台
3. **HomePage** — 书架卡片 + 空状态
4. **ChapterPage** — 大纲树 + 标签栏 + 编辑器 + 工具栏
5. **其余 6 页** — 逐页应用全局卡片/面板/按钮样式
6. **模态框** — 统一对话框样式

### 4.3 不回退现有功能

所有功能保持不变，仅改变视觉呈现。

---

## 5. 预交付检查清单

- [ ] 4 套主题色均可正常切换，无颜色异常
- [ ] 明暗模式切换正常
- [ ] 所有交互元素有 `cursor: pointer`
- [ ] 悬停状态有 `transition: var(--transition-slow)`
- [ ] 文本对比度 ≥ 4.5:1
- [ ] 聚焦状态可见
- [ ] `prefers-reduced-motion` 被尊重
- [ ] TypeScript 编译无错误
