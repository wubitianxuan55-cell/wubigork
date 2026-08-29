# wubigork 移动端远程操控 — 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use `subagent-driven-development` (recommended) or `executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 wubigork 桌面端启动 HTTP 服务，手机通过局域网浏览器远程操控，同时全站从 Layered Glass 迁移到 Material Design 3 设计语言。

**Architecture:** 一套 React 代码库通过 Container Queries 适配桌面和移动端；Go 新增 `internal/mobile/` 包提供 HTTP 服务 + 二维码；M3 主题通过 Ant Design ConfigProvider token 驱动，不改组件源码。

**Tech Stack:** React 19 + TypeScript 6 + Ant Design 6 + Zustand 5 + Vite 8 + Go 1.26 + Wails v2

## Global Constraints

- 不引入新的 npm 依赖（涟漪 CSS only，二维码 Go 端生成）
- 不引入新的 Go 依赖（`net/http` 标准库即可，`embed` 已在用）
- 不修改 Go 核心业务逻辑 — `internal/mobile/` 纯 HTTP 薄适配层
- 不 fork 前端项目 — 同一套 `dist/` 供 Wails embed 和 HTTP FileServer 共用
- 触控区最小 44×44px
- Container Query 为主，Media Query 为辅
- CSS 变量命名使用 `--md-sys-color-*` M3 标准

---

### Task 1: M3 Tonal Palette 生成器

**Files:**
- Create: `frontend/src/theme/m3-palette.ts`
- Create: `frontend/src/theme/` directory

**Interfaces:**
- Produces: `generateTonalPalette(seedHex: string): string[]` — 返回 13 级色调 hex 数组 [0,10,20,...,95,99,100]
- Produces: `sourceColorFromHex(hex: string): number` — hex → HCT 色相
- Produces: `hexFromHct(hue: number, chroma: number, tone: number): string` — HCT → hex

- [ ] **Step 1: 创建 `frontend/src/theme/` 目录和 `m3-palette.ts`**

```typescript
// frontend/src/theme/m3-palette.ts
// Material Design 3 Tonal Palette 轻量实现
// 参考: https://m3.material.io/styles/color/the-color-system/key-colors-tones

/** sRGB 线性化 */
function linearize(c: number): number {
  const s = c / 255
  return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
}

/** 反线性化 */
function delinearize(v: number): number {
  const s = v <= 0.0031308 ? v * 12.92 : 1.055 * v ** (1 / 2.4) - 0.055
  return Math.round(s * 255)
}

/** RGB → 相对亮度 (Y from XYZ) */
function luminance(r: number, g: number, b: number): number {
  return 0.2126 * linearize(r) + 0.7152 * linearize(g) + 0.0722 * linearize(b)
}

/**
 * 混合两个色调：用 tone 在纯黑(0)和纯白(100)之间插值
 * 然后叠加 hue/chroma 分量
 */
function blendTone(hexColor: string, tone: number): string {
  // 简化版：用 luminance 调整实现 M3 tonal palette
  // 完整 HCT 颜色空间实现过于复杂，这里用 CAM16 简化近似
  const r = parseInt(hexColor.slice(1, 3), 16)
  const g = parseInt(hexColor.slice(3, 5), 16)
  const b = parseInt(hexColor.slice(5, 7), 16)

  const targetLum = tone / 100 // 目标亮度 0-1
  const srcLum = luminance(r, g, b)

  if (Math.abs(srcLum - targetLum) < 0.001) {
    return hexColor
  }

  // 通过混合黑白来逼近目标亮度
  let lo = 0, hi = 1, mix = 0.5
  for (let i = 0; i < 20; i++) {
    mix = (lo + hi) / 2
    const mixedR = Math.round(r * (1 - mix) + (tone > 50 ? 255 : 0) * mix)
    const mixedG = Math.round(g * (1 - mix) + (tone > 50 ? 255 : 0) * mix)
    const mixedB = Math.round(b * (1 - mix) + (tone > 50 ? 255 : 0) * mix)
    const mixedLum = luminance(mixedR, mixedG, mixedB)

    if (mixedLum < targetLum) lo = mix
    else hi = mix
  }

  const finalR = Math.round(r * (1 - mix) + (tone > 50 ? 255 : 0) * mix)
  const finalG = Math.round(g * (1 - mix) + (tone > 50 ? 255 : 0) * mix)
  const finalB = Math.round(b * (1 - mix) + (tone > 50 ? 255 : 0) * mix)

  const clamp = (v: number) => Math.min(255, Math.max(0, v))
  return `#${clamp(finalR).toString(16).padStart(2, '0')}${clamp(finalG).toString(16).padStart(2, '0')}${clamp(finalB).toString(16).padStart(2, '0')}`
}

/** M3 标准色调级别 */
const TONAL_STOPS = [0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 95, 99, 100]

/**
 * 从种子 hex 色生成完整 Tonal Palette
 * 返回 13 个 hex 色，对应 TONAL_STOPS 的每个级别
 */
export function generateTonalPalette(seedHex: string): string[] {
  const sanitized = seedHex.replace(/^#/, '')
  if (!/^[0-9a-fA-F]{6}$/.test(sanitized)) {
    throw new Error(`Invalid seed color: ${seedHex}`)
  }
  const hex = `#${sanitized.toLowerCase()}`

  return TONAL_STOPS.map((tone) => blendTone(hex, tone))
}

/**
 * 获取特定 tone 级别的颜色
 */
export function tonalColor(seedHex: string, tone: number): string {
  return blendTone(seedHex, tone)
}

/**
 * M3 默认色调映射（Material Design 3 角色 → tone 值）
 */
export const M3_TONE_MAP = {
  primary: 40,
  onPrimary: 100,
  primaryContainer: 90,
  onPrimaryContainer: 10,
  secondary: 40,
  onSecondary: 100,
  secondaryContainer: 90,
  onSecondaryContainer: 10,
  tertiary: 40,
  onTertiary: 100,
  tertiaryContainer: 90,
  onTertiaryContainer: 10,
  error: 40,
  onError: 100,
  errorContainer: 90,
  onErrorContainer: 10,
  surface: 6,        // dark: tone 6, light: tone 98
  onSurface: 90,     // dark: tone 90, light: tone 10
  surfaceVariant: 30,
  onSurfaceVariant: 80,
  outline: 60,
  outlineVariant: 30,
  surfaceContainer: 8,
  surfaceContainerHigh: 12,
  surfaceContainerHighest: 16,
  inverseSurface: 90,
  inverseOnSurface: 20,
  inversePrimary: 80,
  surfaceDim: 6,
  surfaceBright: 24,
} as const

/** 暗色主题 tone 映射覆盖 */
export const DARK_TONES: Partial<typeof M3_TONE_MAP> = {
  surface: 6,
  onSurface: 90,
  surfaceContainer: 8,
  surfaceContainerHigh: 12,
  surfaceContainerHighest: 16,
  surfaceDim: 6,
  surfaceBright: 24,
}

/** 亮色主题 tone 映射覆盖 */
export const LIGHT_TONES: Partial<typeof M3_TONE_MAP> = {
  surface: 98,
  onSurface: 10,
  surfaceContainer: 94,
  surfaceContainerHigh: 92,
  surfaceContainerHighest: 90,
  surfaceDim: 86,
  surfaceBright: 98,
}
```

- [ ] **Step 2: 类型检查**

```bash
cd frontend && npx tsc --noEmit src/theme/m3-palette.ts
```

- [ ] **Step 3: 提交**

```bash
git add frontend/src/theme/m3-palette.ts
git commit -m "feat: add M3 tonal palette generator"
```

---

### Task 2: 重写 ThemeTokens 接口和主题预设

**Files:**
- Modify: `frontend/src/stores/appStore.ts`

**Interfaces:**
- Consumes: `generateTonalPalette`, `M3_TONE_MAP`, `DARK_TONES`, `LIGHT_TONES` from `../theme/m3-palette`
- Produces: 新的 `ThemeTokens` 接口 (M3 命名); 保持 `getThemeTokens(base, darkMode)` 签名不变; 保持 `useAppStore` 不变
- Produces: `ThemePreset` 枚举不变

- [ ] **Step 1: 重写 ThemeTokens 接口及主题预设**

在 `frontend/src/stores/appStore.ts` 中，找到 `ThemeTokens` 接口（约第 35 行），替换为：

```typescript
// M3 Design Token 命名体系
export interface ThemeTokens {
  // Primary
  colorPrimary: string
  onPrimary: string
  primaryContainer: string
  onPrimaryContainer: string
  // Surface
  surface: string
  onSurface: string
  surfaceVariant: string
  onSurfaceVariant: string
  surfaceContainer: string
  surfaceContainerHigh: string
  surfaceContainerHighest: string
  surfaceDim: string
  surfaceBright: string
  // Outline
  outline: string
  outlineVariant: string
  // Background (兼容旧 Layout 容器)
  colorBgContainer: string
  colorBgLayout: string
  // Text
  colorText: string
  colorTextSecondary: string
  colorBorder: string
  colorSuccess: string
  colorWarning: string
  // Elevation shadows (M3 5 级)
  elevation1: string
  elevation2: string
  elevation3: string
  elevation4: string
  elevation5: string
  // Radius (M3 — 更大圆角)
  radiusSm: string
  radiusMd: string
  radiusLg: string
  radiusXl: string
  // Transitions (M3 标准缓动)
  transitionFast: string
  transitionNormal: string
  transitionSlow: string
  // 兼容旧代码的 accentRgb
  accentRgb: string
}
```

找到 `sharedDarkTokens`（约第 64 行），替换为：

```typescript
const sharedDarkTokens = {
  elevation1: '0 1px 2px rgba(0,0,0,0.3), 0 1px 3px rgba(0,0,0,0.15)',
  elevation2: '0 1px 2px rgba(0,0,0,0.3), 0 2px 6px rgba(0,0,0,0.15)',
  elevation3: '0 4px 8px rgba(0,0,0,0.35), 0 1px 4px rgba(0,0,0,0.15)',
  elevation4: '0 6px 12px rgba(0,0,0,0.4), 0 2px 6px rgba(0,0,0,0.15)',
  elevation5: '0 8px 24px rgba(0,0,0,0.45), 0 4px 12px rgba(0,0,0,0.2)',
  outline: 'rgba(255,255,255,0.12)',
  outlineVariant: 'rgba(255,255,255,0.06)',
  radiusSm: '8px',
  radiusMd: '12px',
  radiusLg: '16px',
  radiusXl: '28px',
  transitionFast: '200ms cubic-bezier(0.2, 0, 0, 1)',
  transitionNormal: '300ms cubic-bezier(0.2, 0, 0, 1)',
  transitionSlow: '400ms cubic-bezier(0.2, 0, 0, 1)',
}
```

找到 `sharedLightTokens`（约第 82 行），替换为：

```typescript
const sharedLightTokens = {
  elevation1: '0 1px 2px rgba(0,0,0,0.08), 0 1px 3px rgba(0,0,0,0.04)',
  elevation2: '0 1px 2px rgba(0,0,0,0.08), 0 2px 6px rgba(0,0,0,0.04)',
  elevation3: '0 4px 8px rgba(0,0,0,0.12), 0 1px 4px rgba(0,0,0,0.04)',
  elevation4: '0 6px 12px rgba(0,0,0,0.15), 0 2px 6px rgba(0,0,0,0.04)',
  elevation5: '0 8px 24px rgba(0,0,0,0.18), 0 4px 12px rgba(0,0,0,0.08)',
  outline: 'rgba(0,0,0,0.12)',
  outlineVariant: 'rgba(0,0,0,0.06)',
  radiusSm: '8px',
  radiusMd: '12px',
  radiusLg: '16px',
  radiusXl: '28px',
  transitionFast: '200ms cubic-bezier(0.2, 0, 0, 1)',
  transitionNormal: '300ms cubic-bezier(0.2, 0, 0, 1)',
  transitionSlow: '400ms cubic-bezier(0.2, 0, 0, 1)',
}
```

找到每个 `themePresets` 中的预设（如 `nightGreen`），将它们改为使用 M3 palette 和新的 token 结构。种子色保持不变，但每个预设现在包含完整的 surface 系列 token。例如 `nightGreen`：

```typescript
import { generateTonalPalette, DARK_TONES, LIGHT_TONES } from '../theme/m3-palette'

function makeDarkTokens(seedHex: string, accentRgb: string): Partial<ThemeTokens> {
  const palette = generateTonalPalette(seedHex)
  return {
    colorPrimary: palette[8],           // tone 80
    onPrimary: palette[2],              // tone 20
    primaryContainer: palette[4],       // tone 40
    onPrimaryContainer: palette[9],     // tone 90
    surface: palette[0],                // tone 0 (dark)
    onSurface: palette[9],              // tone 90
    surfaceVariant: palette[3],         // tone 30
    onSurfaceVariant: palette[8],       // tone 80
    surfaceContainer: palette[1],       // tone 10
    surfaceContainerHigh: palette[2],   // tone 20
    surfaceContainerHighest: palette[3],// tone 30
    surfaceDim: palette[0],             // tone 0
    surfaceBright: palette[3],          // tone 30
    outline: 'rgba(255,255,255,0.12)',
    outlineVariant: 'rgba(255,255,255,0.06)',
    accentRgb,
    colorBgContainer: palette[1],       // tone 10 (用于 Ant Design)
    colorBgLayout: palette[0],          // tone 0
    colorText: palette[9],              // tone 90
    colorTextSecondary: palette[7],     // tone 70
    colorBorder: 'rgba(255,255,255,0.08)',
    colorSuccess: '#4ade80',
    colorWarning: '#f59e0b',
    ...sharedDarkTokens,
  }
}
```

同样为亮色主题创建 `makeLightTokens` 函数，使用 `LIGHT_TONES` 的 tone 值。

找到 `getThemeTokens` 函数（约第 251 行），将其改为使用新的工厂函数：

```typescript
const darkFn: Record<ThemePreset, () => Partial<ThemeTokens>> = {
  nightGreen: () => makeDarkTokens('#4caf50', '76, 175, 80'),
  starPurple: () => makeDarkTokens('#7c4dff', '124, 77, 255'),
  midnightInk: () => makeDarkTokens('#6750a4', '103, 80, 164'),
  warmAmber: () => makeDarkTokens('#ff9800', '255, 152, 0'),
  minimalGray: () => makeDarkTokens('#64748b', '100, 116, 139'),
}

const lightFn: Record<ThemePreset, () => Partial<ThemeTokens>> = {
  nightGreen: () => makeLightTokens('#4caf50', '76, 175, 80'),
  starPurple: () => makeLightTokens('#7c4dff', '124, 77, 255'),
  midnightInk: () => makeLightTokens('#6750a4', '103, 80, 164'),
  warmAmber: () => makeLightTokens('#ff9800', '255, 152, 0'),
  minimalGray: () => makeLightTokens('#64748b', '100, 116, 139'),
}

export function getThemeTokens(base: ThemePreset, darkMode: boolean): ThemeTokens {
  const fn = darkMode ? darkFn[base] : lightFn[base]
  return fn() as ThemeTokens
}
```

> **注意**: 删除原有的 `themePresets` 和 `lightPresets` 大对象，替换为工厂函数。`ThemePreset` 枚举保持 `'nightGreen' | 'starPurple' | 'midnightInk' | 'warmAmber' | 'minimalGray'` 不变。

- [ ] **Step 2: 类型检查**

```bash
cd frontend && npx tsc --noEmit
```

修复所有类型错误。

- [ ] **Step 3: 提交**

```bash
git add frontend/src/stores/appStore.ts
git commit -m "feat: rewrite ThemeTokens to M3 naming with tonal palette"
```

---

### Task 3: 更新 App.tsx ConfigProvider 和 CSS 变量同步

**Files:**
- Modify: `frontend/src/App.tsx`

**Interfaces:**
- Consumes: 新的 `ThemeTokens` 接口 (含 M3 surface/outline/elevation tokens)
- Produces: 不变 — `<ConfigProvider>` + `<MainLayout />`

- [ ] **Step 1: 重写 CSS 变量注入和 ConfigProvider**

替换 `frontend/src/App.tsx` 全部内容：

```typescript
import React, { useEffect } from 'react'
import { ConfigProvider, theme } from 'antd'
import MainLayout from './layouts/MainLayout'
import { useAppStore, getThemeTokens } from './stores/appStore'

const App: React.FC = () => {
  const baseTheme = useAppStore((s) => s.baseTheme)
  const darkMode = useAppStore((s) => s.darkMode)
  const tokens = getThemeTokens(baseTheme, darkMode)

  // 同步 M3 CSS 变量到 :root
  useEffect(() => {
    const root = document.documentElement
    const set = (k: string, v: string) => root.style.setProperty(k, v)

    // M3 Primary
    set('--md-sys-color-primary', tokens.colorPrimary)
    set('--md-sys-color-on-primary', tokens.onPrimary)
    set('--md-sys-color-primary-container', tokens.primaryContainer)
    set('--md-sys-color-on-primary-container', tokens.onPrimaryContainer)

    // M3 Surface
    set('--md-sys-color-surface', tokens.surface)
    set('--md-sys-color-on-surface', tokens.onSurface)
    set('--md-sys-color-surface-variant', tokens.surfaceVariant)
    set('--md-sys-color-on-surface-variant', tokens.onSurfaceVariant)
    set('--md-sys-color-surface-container', tokens.surfaceContainer)
    set('--md-sys-color-surface-container-high', tokens.surfaceContainerHigh)
    set('--md-sys-color-surface-container-highest', tokens.surfaceContainerHighest)
    set('--md-sys-color-surface-dim', tokens.surfaceDim)

    // M3 Outline
    set('--md-sys-color-outline', tokens.outline)
    set('--md-sys-color-outline-variant', tokens.outlineVariant)

    // Background (Ant Design 兼容)
    set('--md-sys-color-bg-container', tokens.colorBgContainer)
    set('--md-sys-color-bg-layout', tokens.colorBgLayout)

    // Text
    set('--md-sys-color-text', tokens.colorText)
    set('--md-sys-color-text-secondary', tokens.colorTextSecondary)
    set('--md-sys-color-border', tokens.colorBorder)

    // Semantic
    set('--md-sys-color-success', tokens.colorSuccess)
    set('--md-sys-color-warning', tokens.colorWarning)

    // Elevation
    set('--md-sys-elevation-1', tokens.elevation1)
    set('--md-sys-elevation-2', tokens.elevation2)
    set('--md-sys-elevation-3', tokens.elevation3)
    set('--md-sys-elevation-4', tokens.elevation4)
    set('--md-sys-elevation-5', tokens.elevation5)

    // Radius
    set('--md-sys-radius-sm', tokens.radiusSm)
    set('--md-sys-radius-md', tokens.radiusMd)
    set('--md-sys-radius-lg', tokens.radiusLg)
    set('--md-sys-radius-xl', tokens.radiusXl)

    // Transition
    set('--md-sys-transition-fast', tokens.transitionFast)
    set('--md-sys-transition-normal', tokens.transitionNormal)
    set('--md-sys-transition-slow', tokens.transitionSlow)
  }, [tokens])

  return (
    <ConfigProvider
      theme={{
        algorithm: darkMode ? theme.darkAlgorithm : theme.defaultAlgorithm,
        token: {
          colorPrimary: tokens.colorPrimary,
          colorBgContainer: tokens.colorBgContainer,
          colorBgLayout: tokens.colorBgLayout,
          colorText: tokens.colorText,
          colorTextSecondary: tokens.colorTextSecondary,
          colorBorder: tokens.colorBorder,
          borderRadius: 16,              // M3 默认更大圆角
          borderRadiusLG: 20,
          borderRadiusSM: 12,
          fontFamily: `system-ui, 'Segoe UI', Roboto, 'Helvetica Neue', sans-serif`,
          fontSize: 14,
          controlHeight: 36,
          lineHeight: 1.5,
        },
      }}
    >
      <MainLayout />
    </ConfigProvider>
  )
}

export default App
```

- [ ] **Step 2: 验证编译**

```bash
cd frontend && npx tsc --noEmit
```

- [ ] **Step 3: 提交**

```bash
git add frontend/src/App.tsx
git commit -m "feat: sync M3 CSS variables and update ConfigProvider tokens"
```

---

### Task 4: 重写 index.css — M3 Surface 全局样式

**Files:**
- Modify: `frontend/src/index.css`

**Interfaces:**
- Consumes: M3 CSS 变量 (`--md-sys-*`)
- Produces: 全局 M3 样式类，无 Layered Glass 遗留

- [ ] **Step 1: 完全重写 index.css**

替换 `frontend/src/index.css` 全部内容：

```css
/* wubigork — Material Design 3 全局样式 */

#root {
  width: 100%;
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
}

body {
  margin: 0;
  font-family: system-ui, 'Segoe UI', Roboto, 'Helvetica Neue', sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  background: var(--md-sys-color-surface);
  color: var(--md-sys-color-on-surface);
}

/* ═══ 滚动条 ═══ */
::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb {
  background: var(--md-sys-color-outline-variant);
  border-radius: 3px;
}
::-webkit-scrollbar-thumb:hover { background: var(--md-sys-color-outline); }

/* 移动端隐藏滚动条 */
@media (max-width: 899px) {
  ::-webkit-scrollbar { display: none; }
  * { scrollbar-width: none; }
}

/* ═══ M3 Surface 工具类 ═══ */
.md-surface {
  background: var(--md-sys-color-surface);
  color: var(--md-sys-color-on-surface);
  border-radius: var(--md-sys-radius-md);
}
.md-surface-container {
  background: var(--md-sys-color-surface-container);
  border-radius: var(--md-sys-radius-lg);
}
.md-surface-container-high {
  background: var(--md-sys-color-surface-container-high);
  border-radius: var(--md-sys-radius-lg);
}
.md-surface-container-highest {
  background: var(--md-sys-color-surface-container-highest);
  border-radius: var(--md-sys-radius-lg);
}

/* M3 Elevation 工具类 */
.md-elevation-1 { box-shadow: var(--md-sys-elevation-1); }
.md-elevation-2 { box-shadow: var(--md-sys-elevation-2); }
.md-elevation-3 { box-shadow: var(--md-sys-elevation-3); }
.md-elevation-4 { box-shadow: var(--md-sys-elevation-4); }
.md-elevation-5 { box-shadow: var(--md-sys-elevation-5); }

/* M3 可点击卡片（带 hover lift + state layer） */
.md-card {
  background: var(--md-sys-color-surface-container);
  border-radius: var(--md-sys-radius-xl);
  box-shadow: var(--md-sys-elevation-1);
  cursor: pointer;
  transition: box-shadow var(--md-sys-transition-normal),
              transform var(--md-sys-transition-normal);
  position: relative;
}
.md-card:hover {
  box-shadow: var(--md-sys-elevation-2);
  transform: translateY(-1px);
}
.md-card:active {
  box-shadow: var(--md-sys-elevation-1);
  transform: translateY(0);
}
/* 触摸设备跳过 hover 动画 */
@media (hover: none) {
  .md-card:hover { box-shadow: var(--md-sys-elevation-1); transform: none; }
}

/* ═══ M3 涟漪效果 (CSS only) ═══ */
.md-ripple {
  position: relative;
  overflow: hidden;
}
.md-ripple::after {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at center,
    var(--md-sys-color-on-surface) 0%,
    transparent 70%
  );
  opacity: 0;
  transition: opacity 400ms;
}
.md-ripple:active::after {
  opacity: 0.1;
}

/* ═══ 分区标题 ═══ */
.section-title {
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.05em;
  color: var(--md-sys-color-on-surface-variant);
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--md-sys-color-outline-variant);
}

/* ═══ 写作编辑器 ═══ */
.writing-textarea {
  font-family: 'Georgia', 'Noto Serif SC', 'Source Han Serif SC', 'STSong', 'SimSun', serif !important;
  background: var(--md-sys-color-surface-container) !important;
  border-radius: var(--md-sys-radius-md) !important;
  border: 1px solid var(--md-sys-color-outline-variant) !important;
  color: var(--md-sys-color-on-surface) !important;
  padding: 20px 24px !important;
  font-size: 16px !important;
  line-height: 2 !important;
  resize: vertical;
  min-height: 240px;
  letter-spacing: 0.02em;
}
.writing-textarea::placeholder {
  color: var(--md-sys-color-on-surface-variant);
  opacity: 0.5;
  font-style: italic;
}
.writing-textarea:focus {
  border-color: var(--md-sys-color-primary) !important;
  box-shadow: 0 0 0 3px rgba(from var(--md-sys-color-primary) r g b / 0.15) !important;
}

/* ═══ 主题色发光按钮 ═══ */
.btn-primary-glow {
  background: var(--md-sys-color-primary) !important;
  border-color: var(--md-sys-color-primary) !important;
  box-shadow: 0 2px 8px rgba(from var(--md-sys-color-primary) r g b / 0.3);
  transition: box-shadow var(--md-sys-transition-slow);
}
.btn-primary-glow:hover {
  box-shadow: 0 4px 16px rgba(from var(--md-sys-color-primary) r g b / 0.4);
}

/* ═══ 动画 ═══ */
@keyframes slideInRight {
  from { transform: translateX(100%); opacity: 0; }
  to { transform: translateX(0); opacity: 1; }
}
@keyframes pulseCursor {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}
@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}
@keyframes pageEnter {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}

.writing-cursor::after {
  content: '▌';
  color: var(--md-sys-color-primary);
  animation: pulseCursor 0.8s ease-in-out infinite;
}
.cursor-blink {
  display: inline-block;
  width: 2px; height: 1.1em;
  background: var(--md-sys-color-primary);
  vertical-align: text-bottom;
  margin-left: 1px;
  border-radius: 1px;
  animation: blink 1s step-end infinite;
}
.page-transition-enter {
  animation: pageEnter 0.25s cubic-bezier(0.2, 0, 0, 1);
}

/* ═══ M3 Navigation (Ant Design 覆盖) ═══ */
.ant-menu-horizontal {
  border-bottom: none !important;
  background: transparent !important;
}
.ant-menu-horizontal .ant-menu-item {
  border-radius: 28px !important;  /* M3 pill shape */
  margin: 0 2px !important;
  padding: 0 16px !important;
  height: 36px !important;
  line-height: 36px !important;
  transition: background var(--md-sys-transition-fast);
  color: var(--md-sys-color-on-surface-variant) !important;
}
.ant-menu-horizontal .ant-menu-item:hover {
  background: var(--md-sys-color-surface-container-high) !important;
  color: var(--md-sys-color-on-surface) !important;
}
.ant-menu-horizontal .ant-menu-item-selected {
  background: var(--md-sys-color-primary-container) !important;
  color: var(--md-sys-color-on-primary-container) !important;
}
.ant-menu-horizontal .ant-menu-item-selected::after {
  display: none !important;
}

/* ═══ Chapter Tabs ═══ */
.chapter-tabs .ant-tabs-nav { margin-bottom: 0 !important; }
.chapter-tabs .ant-tabs-tab {
  border-radius: 20px !important;
  padding: 6px 14px !important;
  margin: 4px 2px !important;
  background: transparent !important;
  color: var(--md-sys-color-on-surface-variant) !important;
  border: none !important;
  transition: all var(--md-sys-transition-fast) !important;
  font-size: 12px !important;
}
.chapter-tabs .ant-tabs-tab-active {
  background: var(--md-sys-color-primary) !important;
  color: var(--md-sys-color-on-primary) !important;
}
.chapter-tabs .ant-tabs-tab-active .ant-tabs-tab-btn { color: var(--md-sys-color-on-primary) !important; }
.chapter-tabs .ant-tabs-tab-remove { color: var(--md-sys-color-on-surface-variant) !important; font-size: 10px !important; }

/* ═══ M3 Modal (Ant Design 覆盖) ═══ */
.ant-modal-content {
  background: var(--md-sys-color-surface-container-high) !important;
  border-radius: var(--md-sys-radius-xl) !important;
  box-shadow: var(--md-sys-elevation-5) !important;
  border: none !important;
}
.ant-modal-header {
  background: transparent !important;
  border-bottom: 1px solid var(--md-sys-color-outline-variant) !important;
  border-radius: var(--md-sys-radius-xl) var(--md-sys-radius-xl) 0 0 !important;
}
.ant-modal-footer {
  border-top: 1px solid var(--md-sys-color-outline-variant) !important;
}
.ant-modal-close { color: var(--md-sys-color-on-surface-variant) !important; }
.ant-modal-mask {
  background: rgba(0, 0, 0, 0.5) !important;
  backdrop-filter: none !important;
}

/* ═══ Bento Grid 响应式 ═══ */
.bento-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  grid-auto-rows: minmax(160px, auto);
  gap: 16px;
}
@media (max-width: 1200px) { .bento-grid { grid-template-columns: repeat(3, 1fr); } }
@media (max-width: 768px)  { .bento-grid { grid-template-columns: repeat(2, 1fr); } }
@media (max-width: 480px)  { .bento-grid { grid-template-columns: 1fr; } }

/* ═══ 触控区最小 44px ═══ */
@media (max-width: 899px) {
  button, .ant-btn, .ant-menu-item, .ant-tabs-tab,
  [role="button"], a[href], .clickable {
    min-height: 44px;
    min-width: 44px;
  }
}

/* ═══ reduced-motion ═══ */
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
```

- [ ] **Step 2: 验证构建**

```bash
cd frontend && npx tsc --noEmit
```

- [ ] **Step 3: 提交**

```bash
git add frontend/src/index.css
git commit -m "feat: replace Layered Glass with M3 Surface global styles"
```

---

### Task 5: 更新 MainLayout.tsx 为 M3 视觉风格

**Files:**
- Modify: `frontend/src/layouts/MainLayout.tsx`

**Interfaces:**
- Consumes: M3 CSS 变量
- Produces: 不变 — 仍输出完整的 `<Layout>` 结构，但视觉改为 M3

- [ ] **Step 1: 替换 MainLayout 中的 CSS 变量引用和样式**

在 `MainLayout.tsx` 中，做以下替换（批量处理）：

- 所有 `var(--color-primary)` → `var(--md-sys-color-primary)`
- 所有 `var(--color-text)` → `var(--md-sys-color-text)`
- 所有 `var(--color-text-secondary)` → `var(--md-sys-color-text-secondary)`
- 所有 `var(--color-border)` → `var(--md-sys-color-border)` 或 `var(--md-sys-color-outline-variant)`
- 所有 `var(--bg-glass)` → `var(--md-sys-color-surface-container)`
- 所有 `var(--bg-elevated)` → `var(--md-sys-color-surface-container-high)`
- 所有 `var(--bg-deep)` → `var(--md-sys-color-surface-dim)`
- 所有 `var(--bg-base)` → `var(--md-sys-color-surface)`
- 所有 `var(--border-subtle)` → `var(--md-sys-color-outline-variant)`
- 所有 `var(--shadow-sm)` → `var(--md-sys-elevation-1)`
- 所有 `var(--shadow-md)` → `var(--md-sys-elevation-2)`
- 所有 `var(--shadow-lg)` → `var(--md-sys-elevation-3)`
- 所有 `var(--shadow-glow)` → `0 2px 8px rgba(from var(--md-sys-color-primary) r g b / 0.3)`
- 所有 `var(--radius-sm)` → `var(--md-sys-radius-sm)`
- 所有 `var(--radius-md)` → `var(--md-sys-radius-md)`
- 所有 `var(--radius-lg)` → `var(--md-sys-radius-lg)`
- 所有 `var(--radius-xl)` → `var(--md-sys-radius-xl)`
- 所有 `var(--transition-fast)` → `var(--md-sys-transition-fast)`
- 所有 `var(--transition-normal)` → `var(--md-sys-transition-normal)`
- 所有 `var(--transition-slow)` → `var(--md-sys-transition-slow)`
- 所有 `var(--accent-rgb)` 使用保持不变（兼容旧 rgab() 用法）

同时移除所有 `backdropFilter: 'blur(...)'` 和 `WebkitBackdropFilter` 属性 — M3 不使用毛玻璃。

Header 的 `textShadow` 移除（M3 无文字发光）。

- [ ] **Step 2: 验证构建**

```bash
cd frontend && npx tsc --noEmit
```

- [ ] **Step 3: 提交**

```bash
git add frontend/src/layouts/MainLayout.tsx
git commit -m "refactor: migrate MainLayout to M3 design tokens"
```

---

### Task 6: 创建 useMediaQuery Hook

**Files:**
- Create: `frontend/src/hooks/useMediaQuery.ts`
- Create: `frontend/src/hooks/` directory

**Interfaces:**
- Consumes: 无
- Produces: `useMediaQuery(query: string): boolean` — 监听 CSS Media Query，返回匹配结果

- [ ] **Step 1: 实现 hook**

```typescript
// frontend/src/hooks/useMediaQuery.ts
import { useState, useEffect } from 'react'

/** 响应式断点查询 hook — 移动优先 */
export function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() => {
    if (typeof window === 'undefined') return false
    return window.matchMedia(query).matches
  })

  useEffect(() => {
    const mq = window.matchMedia(query)
    const handler = (e: MediaQueryListEvent) => setMatches(e.matches)

    mq.addEventListener('change', handler)
    return () => mq.removeEventListener('change', handler)
  }, [query])

  return matches
}

/** 预设断点 */
export const BREAKPOINTS = {
  compact:  '(max-width: 599px)',
  medium:   '(min-width: 600px) and (max-width: 899px)',
  mobile:   '(max-width: 899px)',   // compact + medium
  expanded: '(min-width: 900px)',   // 桌面端
} as const

/** 便捷 hook */
export function useIsMobile(): boolean {
  return useMediaQuery(BREAKPOINTS.mobile)
}

export function useIsCompact(): boolean {
  return useMediaQuery(BREAKPOINTS.compact)
}
```

- [ ] **Step 2: 类型检查**

```bash
cd frontend && npx tsc --noEmit
```

- [ ] **Step 3: 提交**

```bash
git add frontend/src/hooks/useMediaQuery.ts
git commit -m "feat: add useMediaQuery hook with preset breakpoints"
```

---

### Task 7: 创建移动端导航组件 (MobileTabBar, MobileDrawer, AppBar)

**Files:**
- Create: `frontend/src/components/MobileTabBar.tsx`
- Create: `frontend/src/components/MobileDrawer.tsx`
- Create: `frontend/src/components/AppBar.tsx`

**Interfaces:**
- Consumes: 无外部依赖
- Produces:
  - `MobileTabBar`: props `{ currentPage: Page, onNavigate: (page: Page) => void }`
  - `MobileDrawer`: props `{ open: boolean, onClose: () => void, currentPage: Page, onNavigate: (page: Page) => void }`
  - `AppBar`: props `{ title: string, onMenuClick: () => void }`

- [ ] **Step 1: 创建 MobileTabBar**

```typescript
// frontend/src/components/MobileTabBar.tsx
import React from 'react'
import {
  HomeOutlined, UnorderedListOutlined, BookOutlined,
  UserOutlined, GlobalOutlined,
} from '@ant-design/icons'

type Page = 'home' | 'worldview' | 'character' | 'outline' | 'chapter' | 'canvas' | 'imagegen' | 'export' | 'settings'

const tabs: { key: Page; icon: React.ReactNode; label: string }[] = [
  { key: 'home', icon: <HomeOutlined />, label: '项目' },
  { key: 'outline', icon: <UnorderedListOutlined />, label: '大纲' },
  { key: 'chapter', icon: <BookOutlined />, label: '写作' },
  { key: 'character', icon: <UserOutlined />, label: '角色' },
  { key: 'worldview', icon: <GlobalOutlined />, label: '世界' },
]

const tabBarStyle: React.CSSProperties = {
  position: 'fixed',
  bottom: 0,
  left: 0,
  right: 0,
  height: 64,
  display: 'flex',
  justifyContent: 'space-around',
  alignItems: 'center',
  background: 'var(--md-sys-color-surface-container)',
  borderTop: '1px solid var(--md-sys-color-outline-variant)',
  paddingBottom: 'env(safe-area-inset-bottom, 0px)',
  zIndex: 100,
}

const tabItemStyle: React.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  alignItems: 'center',
  justifyContent: 'center',
  gap: 2,
  flex: 1,
  height: '100%',
  cursor: 'pointer',
  border: 'none',
  background: 'transparent',
  padding: 0,
  transition: 'color 200ms cubic-bezier(0.2, 0, 0, 1)',
}

interface Props {
  currentPage: Page
  onNavigate: (page: Page) => void
}

const MobileTabBar: React.FC<Props> = ({ currentPage, onNavigate }) => (
  <nav style={tabBarStyle}>
    {tabs.map((t) => {
      const active = t.key === currentPage
      return (
        <button
          key={t.key}
          style={{
            ...tabItemStyle,
            color: active
              ? 'var(--md-sys-color-primary)'
              : 'var(--md-sys-color-on-surface-variant)',
          }}
          onClick={() => onNavigate(t.key)}
        >
          <span style={{ fontSize: 20 }}>{t.icon}</span>
          <span style={{ fontSize: 10, fontWeight: active ? 600 : 400 }}>
            {t.label}
          </span>
        </button>
      )
    })}
  </nav>
)

export default MobileTabBar
```

- [ ] **Step 2: 创建 MobileDrawer**

```typescript
// frontend/src/components/MobileDrawer.tsx
import React from 'react'
import {
  AimOutlined, PictureOutlined, ExportOutlined,
  SettingOutlined, BarChartOutlined,
} from '@ant-design/icons'

type Page = 'home' | 'worldview' | 'character' | 'outline' | 'chapter' | 'canvas' | 'imagegen' | 'export' | 'settings'

const secondaryItems: { key: Page; icon: React.ReactNode; label: string }[] = [
  { key: 'canvas', icon: <AimOutlined />, label: '画布' },
  { key: 'imagegen', icon: <PictureOutlined />, label: '绘梦' },
  { key: 'home', icon: <BarChartOutlined />, label: '分析' },
  { key: 'export', icon: <ExportOutlined />, label: '导出' },
  { key: 'settings', icon: <SettingOutlined />, label: '设置' },
]

const overlayStyle: React.CSSProperties = {
  position: 'fixed',
  inset: 0,
  zIndex: 200,
  display: 'flex',
}

const scrimStyle: React.CSSProperties = {
  position: 'absolute',
  inset: 0,
  background: 'rgba(0,0,0,0.4)',
}

const drawerStyle: React.CSSProperties = {
  position: 'absolute',
  top: 0,
  right: 0,
  width: 280,
  maxWidth: '80vw',
  height: '100%',
  background: 'var(--md-sys-color-surface-container)',
  boxShadow: 'var(--md-sys-elevation-5)',
  display: 'flex',
  flexDirection: 'column',
  overflow: 'hidden',
}

const itemStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 16,
  padding: '16px 24px',
  minHeight: 56,
  fontSize: 14,
  border: 'none',
  background: 'transparent',
  cursor: 'pointer',
  width: '100%',
  color: 'var(--md-sys-color-on-surface-variant)',
}

interface Props {
  open: boolean
  onClose: () => void
  currentPage: Page
  onNavigate: (page: Page) => void
}

const MobileDrawer: React.FC<Props> = ({ open, onClose, currentPage, onNavigate }) => {
  if (!open) return null

  return (
    <div style={overlayStyle} onClick={onClose}>
      <div style={scrimStyle} />
      <div style={drawerStyle} onClick={(e) => e.stopPropagation()}>
        <div style={{
          padding: '16px 24px',
          borderBottom: '1px solid var(--md-sys-color-outline-variant)',
          display: 'flex',
          justifyContent: 'flex-end',
        }}>
          <button onClick={onClose} style={{
            background: 'none', border: 'none', fontSize: 20,
            cursor: 'pointer', color: 'var(--md-sys-color-on-surface-variant)',
            padding: 4, minHeight: 44, minWidth: 44,
          }}>✕</button>
        </div>
        {secondaryItems.map((item) => {
          const active = item.key === currentPage
          return (
            <button
              key={item.key}
              style={{
                ...itemStyle,
                background: active ? 'var(--md-sys-color-primary-container)' : 'transparent',
                color: active
                  ? 'var(--md-sys-color-on-primary-container)'
                  : 'var(--md-sys-color-on-surface-variant)',
              }}
              onClick={() => { onNavigate(item.key); onClose() }}
            >
              <span style={{ fontSize: 20 }}>{item.icon}</span>
              {item.label}
            </button>
          )
        })}
      </div>
    </div>
  )
}

export default MobileDrawer
```

- [ ] **Step 3: 创建 AppBar**

```typescript
// frontend/src/components/AppBar.tsx
import React from 'react'
import { MenuOutlined } from '@ant-design/icons'

const appBarStyle: React.CSSProperties = {
  position: 'sticky',
  top: 0,
  zIndex: 99,
  display: 'flex',
  alignItems: 'center',
  height: 56,
  padding: '0 16px',
  background: 'var(--md-sys-color-surface)',
  borderBottom: '1px solid var(--md-sys-color-outline-variant)',
}

interface Props {
  title: string
  onMenuClick: () => void
}

const AppBar: React.FC<Props> = ({ title, onMenuClick }) => (
  <header style={appBarStyle}>
    <button
      onClick={onMenuClick}
      style={{
        background: 'none', border: 'none', cursor: 'pointer',
        marginRight: 16, fontSize: 20,
        color: 'var(--md-sys-color-on-surface-variant)',
        minWidth: 44, minHeight: 44, padding: 0,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
      }}
    >
      <MenuOutlined />
    </button>
    <span style={{
      fontSize: 18, fontWeight: 600,
      color: 'var(--md-sys-color-on-surface)',
    }}>
      {title}
    </span>
  </header>
)

export default AppBar
```

- [ ] **Step 4: 类型检查**

```bash
cd frontend && npx tsc --noEmit
```

- [ ] **Step 5: 提交**

```bash
git add frontend/src/components/MobileTabBar.tsx frontend/src/components/MobileDrawer.tsx frontend/src/components/AppBar.tsx
git commit -m "feat: add mobile navigation components (TabBar, Drawer, AppBar)"
```

---

### Task 8: 创建通用移动端组件 (MobileSheet, LongPressable)

**Files:**
- Create: `frontend/src/components/MobileSheet.tsx`
- Create: `frontend/src/components/LongPressable.tsx`

**Interfaces:**
- Produces:
  - `MobileSheet`: props `{ open: boolean, onClose: () => void, children: React.ReactNode, title?: string, height?: string }`
  - `LongPressable`: props `{ onLongPress: () => void, children: React.ReactNode, delay?: number }` + 转发所有 div props

- [ ] **Step 1: 创建 MobileSheet (底部滑出面板)**

```typescript
// frontend/src/components/MobileSheet.tsx
import React, { useEffect, useRef, useState } from 'react'

interface Props {
  open: boolean
  onClose: () => void
  children: React.ReactNode
  title?: string
  height?: string  // 默认 '60vh'
}

const MobileSheet: React.FC<Props> = ({
  open, onClose, children, title, height = '60vh'
}) => {
  const [visible, setVisible] = useState(false)
  const [animating, setAnimating] = useState(false)

  useEffect(() => {
    if (open) {
      setVisible(true)
      requestAnimationFrame(() => setAnimating(true))
    } else {
      setAnimating(false)
      const timer = setTimeout(() => setVisible(false), 300)
      return () => clearTimeout(timer)
    }
  }, [open])

  if (!visible) return null

  return (
    <div
      style={{
        position: 'fixed', inset: 0, zIndex: 200,
        display: 'flex', alignItems: 'flex-end',
      }}
    >
      <div
        onClick={onClose}
        style={{
          position: 'absolute', inset: 0,
          background: 'rgba(0,0,0,0)',
          transition: 'background 300ms cubic-bezier(0.2, 0, 0, 1)',
          ...(animating ? { background: 'rgba(0,0,0,0.4)' } : {}),
        }}
      />
      <div
        style={{
          position: 'relative',
          width: '100%',
          maxHeight: height,
          background: 'var(--md-sys-color-surface-container-high)',
          borderTopLeftRadius: 28,
          borderTopRightRadius: 28,
          boxShadow: 'var(--md-sys-elevation-5)',
          display: 'flex',
          flexDirection: 'column',
          transform: animating ? 'translateY(0)' : 'translateY(100%)',
          transition: 'transform 300ms cubic-bezier(0.2, 0, 0, 1)',
          paddingBottom: 'env(safe-area-inset-bottom, 0px)',
        }}
      >
        {/* 拖拽手柄 */}
        <div style={{
          display: 'flex', justifyContent: 'center', padding: '8px 0',
        }}>
          <div style={{
            width: 32, height: 4, borderRadius: 2,
            background: 'var(--md-sys-color-on-surface-variant)',
            opacity: 0.4,
          }} />
        </div>

        {title && (
          <div style={{
            padding: '0 24px 12px', fontSize: 16, fontWeight: 600,
            color: 'var(--md-sys-color-on-surface)',
          }}>
            {title}
          </div>
        )}

        <div style={{ flex: 1, overflow: 'auto', padding: '0 16px 16px' }}>
          {children}
        </div>
      </div>
    </div>
  )
}

export default MobileSheet
```

- [ ] **Step 2: 创建 LongPressable (长按检测)**

```typescript
// frontend/src/components/LongPressable.tsx
import React, { useRef, useCallback } from 'react'

interface Props extends React.HTMLAttributes<HTMLDivElement> {
  onLongPress: () => void
  children: React.ReactNode
  delay?: number  // 默认 500ms
}

const LongPressable: React.FC<Props> = ({
  onLongPress, children, delay = 500, ...divProps
}) => {
  const timerRef = useRef<ReturnType<typeof setTimeout>>()
  const movedRef = useRef(false)

  const clear = useCallback(() => {
    if (timerRef.current) {
      clearTimeout(timerRef.current)
      timerRef.current = undefined
    }
  }, [])

  return (
    <div
      {...divProps}
      onTouchStart={(e) => {
        movedRef.current = false
        timerRef.current = setTimeout(() => {
          if (!movedRef.current) {
            // 震动反馈
            if (navigator.vibrate) navigator.vibrate(10)
            onLongPress()
          }
        }, delay)
      }}
      onTouchMove={() => { movedRef.current = true }}
      onTouchEnd={clear}
      onTouchCancel={clear}
      onContextMenu={(e) => {
        // 桌面端右键不触发长按
      }}
    >
      {children}
    </div>
  )
}

export default LongPressable
```

- [ ] **Step 3: 类型检查 + 提交**

```bash
cd frontend && npx tsc --noEmit
git add frontend/src/components/MobileSheet.tsx frontend/src/components/LongPressable.tsx
git commit -m "feat: add MobileSheet and LongPressable components"
```

---

### Task 9: MainLayout 响应式双导航

**Files:**
- Modify: `frontend/src/layouts/MainLayout.tsx`

**Interfaces:**
- Consumes: `useIsMobile`, `useMediaQuery` from `../hooks/useMediaQuery`
- Consumes: `MobileTabBar`, `MobileDrawer`, `AppBar` from `../components/`
- Produces: 桌面端保持 Sidebar，移动端显示 AppBar + TabBar + Drawer

- [ ] **Step 1: 在 MainLayout.tsx 顶部添加 import**

在现有 import 块后添加：

```typescript
import { useIsMobile } from '../hooks/useMediaQuery'
import MobileTabBar from '../components/MobileTabBar'
import MobileDrawer from '../components/MobileDrawer'
import AppBar from '../components/AppBar'
```

在组件内部，`const { loggedIn, ... }` 之后添加：

```typescript
const isMobile = useIsMobile()
const [drawerOpen, setDrawerOpen] = useState(false)
```

- [ ] **Step 2: 修改 Header — 桌面端显示顶栏，移动端显示 AppBar**

将 `<Header>` 块用条件包裹：

```tsx
{!isMobile && (
  <Header style={{ /* 现有样式 */ }}>
    {/* 现有顶栏内容 — 保持不变 */}
  </Header>
)}

{isMobile && (
  <AppBar
    title={pageLabels[page]}
    onMenuClick={() => setDrawerOpen(true)}
  />
)}
```

- [ ] **Step 3: 移动端内容区添加底部间距**

在 `<Content>` 的 style 中添加条件 paddingBottom：

```tsx
<Content style={{
  padding: '8px 16px 16px',
  paddingBottom: isMobile ? '72px' : '16px',  // 为 TabBar 留空间
  background: 'var(--md-sys-color-bg-layout)',
  overflow: 'auto', flex: 1,
}}>
```

- [ ] **Step 4: 在 return 末尾（Footer 之前）添加移动端导航**

```tsx
{/* 移动端底部 TabBar */}
{isMobile && (
  <MobileTabBar currentPage={page} onNavigate={setPage} />
)}

{/* 移动端 Drawer */}
{isMobile && (
  <MobileDrawer
    open={drawerOpen}
    onClose={() => setDrawerOpen(false)}
    currentPage={page}
    onNavigate={setPage}
  />
)}
```

- [ ] **Step 5: 类型检查**

```bash
cd frontend && npx tsc --noEmit
```

修复所有类型错误（确保 `Page` 类型在新增的组件中也一致）。

- [ ] **Step 6: 提交**

```bash
git add frontend/src/layouts/MainLayout.tsx
git commit -m "feat: add responsive dual navigation (desktop sidebar + mobile tab bar/drawer)"
```

---

### Task 10: 创建 Go `internal/mobile/` 包

**Files:**
- Create: `internal/mobile/qrcode.go`
- Create: `internal/mobile/handlers.go`
- Create: `internal/mobile/server.go`
- Modify: `internal/app/app.go` — 新增 `mobileServer` 字段和 3 个 Wails 绑定方法
- Modify: `main.go` — 传入 `dist` embed.FS 给 App

**Interfaces:**
- Consumes: `embed.FS` (dist), `app.App` 现有方法
- Produces: `StartMobileServer`, `StopMobileServer`, `GetMobileServerStatus` — Wails 绑定

- [ ] **Step 1: 创建二维码生成**

```go
// internal/mobile/qrcode.go
package mobile

import (
	"bytes"
	"image"
	"image/png"
	"net"

	"encoding/base64"
)

// GetLANIP 返回第一个非 loopback 的局域网 IPv4 地址
func GetLANIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipNet.IP.To4()
			if ip4 == nil || ip4.IsLoopback() {
				continue
			}
			return ip4.String()
		}
	}
	return "127.0.0.1"
}

// GenerateQRCodePNG 生成包含 text 的二维码 PNG 的 base64 字符串
// 使用简单版本：绘制一个占位二维码图像 (8x8 模块的简化 QR)
// 生产可使用 go-qrcode 库，这里先用 canvas 占位避免依赖
func GenerateQRCodePNG(text string) (string, error) {
	size := 256
	moduleSize := size / 29 // QR version 3: 29×29 modules
	img := image.NewGray(image.Rect(0, 0, size, size))

	// 简化：生成带定位图案的占位二维码
	// 三个角定位图案 (7×7)
	drawFinder := func(x, y int) {
		for i := 0; i < 7; i++ {
			for j := 0; j < 7; j++ {
				px := x*moduleSize + i*moduleSize
				py := y*moduleSize + j*moduleSize
				outline := i == 0 || i == 6 || j == 0 || j == 6
				inner := i >= 2 && i <= 4 && j >= 2 && j <= 4
				if outline || inner {
					for di := 0; di < moduleSize; di++ {
						for dj := 0; dj < moduleSize; dj++ {
							img.SetGray(px+di, py+dj, image.Gray{Y: 0})
						}
					}
				}
			}
		}
	}
	drawFinder(0, 0)
	drawFinder(22, 0)
	drawFinder(0, 22)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
```

> **注意**: 上述二维码是简化的占位版本（仅定位图案）。真正的 QR 生成需要数据编码，建议后续用 `image/draw` 加文字叠加，或考虑引入 `rsc.io/qr`。当前阶段以实现功能框架为主。

- [ ] **Step 2: 创建 HTTP handlers**

```go
// internal/mobile/handlers.go
package mobile

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/wubigork/wubigork/internal/app"
)

type Handlers struct {
	App    *app.App
	DistFS interface{} // embed.FS，保持类型灵活
}

func (h *Handlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", h.handleHealth)
	mux.HandleFunc("/api/qrcode", h.handleQRCode)
	mux.HandleFunc("/api/projects", h.handleProjects)
}

func (h *Handlers) handleHealth(w http.ResponseWriter, r *http.Request) {
	ip := GetLANIP()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"ip":     ip,
	})
}

func (h *Handlers) handleQRCode(w http.ResponseWriter, r *http.Request) {
	ip := GetLANIP()
	url := "http://" + ip + ":8080"

	b64, err := GenerateQRCodePNG(url)
	if err != nil {
		http.Error(w, "QR generation failed", 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"url":    url,
		"qrcode": b64,
	})
}

func (h *Handlers) handleProjects(w http.ResponseWriter, r *http.Request) {
	// 暂时返回占位 — 后续 Phase 对接实际 API
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "projects endpoint ready",
	})
}

// SPAHandler 返回单页应用 fallback handler
func SPAHandler(distFS interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 尝试提供静态文件
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// 尝试打开文件
		// 由于 embed.FS 类型在包间传递复杂，这里用接口方式
		// 实际实现见 server.go
		_ = path
		http.ServeFile(w, r, filepath.Join("dist", "index.html"))
	}
}
```

- [ ] **Step 3: 创建 HTTP 服务器**

```go
// internal/mobile/server.go
package mobile

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
)

type Server struct {
	mu     sync.Mutex
	http   *http.Server
	port   int
	distFS interface{} // embed.FS
	running bool
}

func NewServer(port int, distFS interface{}) *Server {
	return &Server{port: port, distFS: distFS}
}

func (s *Server) Start(ip string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server already running")
	}

	mux := http.NewServeMux()

	// API 路由
	handlers := &Handlers{DistFS: s.distFS}
	handlers.Register(mux)

	// 静态文件 + SPA fallback
	// 简化处理：所有非 /api/ 请求返回 index.html（让 React Router 处理）
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
			// API 路由已在上面注册，这里不会到达
			http.NotFound(w, r)
			return
		}
		// SPA fallback — 由 Wails asset server 处理
		// 在 HTTP 模式下，使用 net/http FileServer
		fmt.Fprintf(w, `<!doctype html><html lang="zh-CN">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>wubigork</title></head>
<body><div id="root"></div><script type="module" src="http://%s:5173/src/main.tsx"></script></body></html>`, ip)
	})

	s.http = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	s.running = true

	go func() {
		log.Printf("[mobile] HTTP server starting on http://%s:%d", ip, s.port)
		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[mobile] HTTP server error: %v", err)
		}
	}()

	return nil
}

func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || s.http == nil {
		return nil
	}

	s.running = false
	return s.http.Shutdown(context.Background())
}

func (s *Server) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
```

- [ ] **Step 4: 集成到 App 结构体**

在 `internal/app/app.go` 的 `App` struct 中添加字段：

```go
import "github.com/wubigork/wubigork/internal/mobile"

type App struct {
	// ... 现有字段 ...
	mobileServer *mobile.Server  // 新增
	distFS       interface{}     // 新增: embed.FS 引用
}
```

在 `Startup` 方法中初始化：

```go
func (app *App) Startup(ctx context.Context) {
	app.ctx = ctx
	// ... 现有代码 ...
	app.mobileServer = mobile.NewServer(8080, app.distFS)
}
```

在 `app.go` 末尾添加 3 个 Wails 绑定方法：

```go
// StartMobileServer 启动移动端 HTTP 服务
func (app *App) StartMobileServer(port int) (map[string]interface{}, error) {
	if port <= 0 {
		port = 8080
	}
	ip := mobile.GetLANIP()
	if err := app.mobileServer.Start(ip); err != nil {
		return nil, err
	}
	b64, _ := mobile.GenerateQRCodePNG(fmt.Sprintf("http://%s:%d", ip, port))
	return map[string]interface{}{
		"ip":   ip,
		"port": port,
		"url":  fmt.Sprintf("http://%s:%d", ip, port),
		"qrcode": b64,
	}, nil
}

// StopMobileServer 停止移动端 HTTP 服务
func (app *App) StopMobileServer() error {
	return app.mobileServer.Stop()
}

// GetMobileServerStatus 获取移动端服务状态
func (app *App) GetMobileServerStatus() map[string]interface{} {
	ip := mobile.GetLANIP()
	return map[string]interface{}{
		"running": app.mobileServer.IsRunning(),
		"ip":      ip,
		"port":    8080,
	}
}
```

- [ ] **Step 5: 修改 main.go 传入 dist embed.FS**

在 `main.go` 中，将 `assets` embed.FS 传给 App：

```go
func main() {
	application := app.New()
	application.SetDistFS(assets)  // 新增方法
	// ... 其余不变 ...
}
```

在 `internal/app/app.go` 中添加 `SetDistFS` 方法：

```go
func (app *App) SetDistFS(fs interface{}) {
	app.distFS = fs
	if app.mobileServer != nil {
		app.mobileServer = mobile.NewServer(8080, fs)
	}
}
```

- [ ] **Step 6: Go 编译验证**

```bash
cd D:/AI/wubigork && go build -o build/bin/wubigork-mobile-test.exe .
```

- [ ] **Step 7: 提交**

```bash
git add internal/mobile/ internal/app/app.go main.go
git commit -m "feat: add mobile HTTP server with QR code and Wails bindings"
```

---

### Task 11: 设置页移动端访问开关

**Files:**
- Modify: `frontend/src/pages/SettingsPage.tsx`

**Interfaces:**
- Consumes: Wails Go bindings `StartMobileServer`, `StopMobileServer`, `GetMobileServerStatus`
- Produces: UI 开关 + 二维码面板

> **由于 SettingsPage.tsx 可能较大，此处描述关键改动，实际编辑时读取完整文件后精确操作。**

- [ ] **Step 1: 在 SettingsPage 中添加移动端 panel**

读取 `frontend/src/pages/SettingsPage.tsx` 完整内容，在合适位置（如主题设置之后）添加：

```tsx
{/* 移动端远程访问 */}
<Card title="移动端远程访问" style={{ marginBottom: 16 }}>
  <Space direction="vertical" style={{ width: '100%' }}>
    <Switch
      checked={mobileRunning}
      onChange={async (checked) => {
        try {
          if (checked) {
            const result = await window.go.app.App.StartMobileServer(8080)
            setMobileInfo(result)
            setMobileRunning(true)
          } else {
            await window.go.app.App.StopMobileServer()
            setMobileRunning(false)
            setMobileInfo(null)
          }
        } catch (err) {
          console.error('Mobile server toggle failed:', err)
        }
      }}
      checkedChildren="已启用"
      unCheckedChildren="已关闭"
    />

    {mobileRunning && mobileInfo && (
      <div style={{ textAlign: 'center', marginTop: 16 }}>
        <img
          src={mobileInfo.qrcode}
          alt="QR Code"
          style={{ width: 200, height: 200, borderRadius: 12 }}
        />
        <div style={{ marginTop: 8, color: 'var(--md-sys-color-on-surface-variant)', fontSize: 13 }}>
          {mobileInfo.url}
        </div>
        <Typography.Text type="secondary" style={{ fontSize: 11, display: 'block', marginTop: 4 }}>
          ⚠ 请确保手机和电脑在同一局域网
        </Typography.Text>
      </div>
    )}
  </Space>
</Card>
```

需要在组件顶部添加 state：

```typescript
const [mobileRunning, setMobileRunning] = useState(false)
const [mobileInfo, setMobileInfo] = useState<{ip:string, port:number, url:string, qrcode:string} | null>(null)
```

并在 `useEffect` 中检查初始状态：

```typescript
useEffect(() => {
  // 检查移动端服务初始状态
  window.go?.app?.App?.GetMobileServerStatus().then((s: any) => {
    if (s?.running) setMobileRunning(true)
  }).catch(() => {})
}, [])
```

- [ ] **Step 2: 类型检查 + 构建**

```bash
cd frontend && npx tsc --noEmit
```

- [ ] **Step 3: 提交**

```bash
git add frontend/src/pages/SettingsPage.tsx
git commit -m "feat: add mobile server toggle with QR code in SettingsPage"
```

---

### Task 12: 页面响应式适配 — HomePage

**Files:**
- Modify: `frontend/src/pages/HomePage.tsx`

**Interfaces:**
- Consumes: `useIsMobile` from `../hooks/useMediaQuery`
- Produces: 桌面端 Bento Grid，移动端单列卡片

- [ ] **Step 1: 读取 HomePage.tsx 并做适配**

在 `HomePage.tsx` 顶部添加：

```typescript
import { useIsMobile } from '../hooks/useMediaQuery'
```

在组件内添加：

```typescript
const isMobile = useIsMobile()
```

修改 Bento Grid 容器样式，在移动端改为单列：

```tsx
<div className="bento-grid" style={isMobile ? {
  gridTemplateColumns: '1fr',
  gap: 12,
  gridAutoRows: 'auto',
} : undefined}>
```

移动端卡片间距缩小、字号微调。将 `.glass-card` 替换为 `.md-card md-elevation-1`。

- [ ] **Step 2: 类型检查**

```bash
cd frontend && npx tsc --noEmit
```

- [ ] **Step 3: 提交**

```bash
git add frontend/src/pages/HomePage.tsx
git commit -m "feat: make HomePage responsive for mobile"
```

---

### Task 13: 页面响应式适配 — ChapterPage

**Files:**
- Modify: `frontend/src/pages/ChapterPage.tsx`

**Interfaces:**
- Consumes: `useIsMobile`, `MobileSheet` from `../components/MobileSheet`
- Produces: 桌面端 SplitPane (左编辑/右 AI)，移动端全屏编辑器 + 底部 Sheet

**策略**：ChapterPage 是最大的文件 (52KB)，不重写全部逻辑——只在最外层加条件渲染，将 AI 面板从右侧栏移入 MobileSheet。

- [ ] **Step 1: 读取 ChapterPage.tsx 并找到 SplitPane 相关代码**

添加 import：

```typescript
import { useIsMobile } from '../hooks/useMediaQuery'
import MobileSheet from '../components/MobileSheet'
```

组件内添加：

```typescript
const isMobile = useIsMobile()
const [aiSheetOpen, setAiSheetOpen] = useState(false)
```

找到桌面端左右分栏结构，移动端改为：

```tsx
{isMobile ? (
  // 移动端: 全屏编辑器
  <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
    {/* 编辑器区域 — 占满全屏 */}
    <div style={{ flex: 1, overflow: 'auto', padding: 8 }}>
      {/* 原有编辑器内容 */}
    </div>

    {/* 底部 AI 触发按钮 */}
    <button
      onClick={() => setAiSheetOpen(true)}
      style={{
        position: 'fixed', bottom: 72, right: 16,
        width: 48, height: 48, borderRadius: '50%',
        background: 'var(--md-sys-color-primary)',
        color: 'var(--md-sys-color-on-primary)',
        border: 'none', boxShadow: 'var(--md-sys-elevation-3)',
        fontSize: 20, cursor: 'pointer',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        zIndex: 50,
      }}
    >
      ✦
    </button>

    {/* AI 面板 Sheet */}
    <MobileSheet
      open={aiSheetOpen}
      onClose={() => setAiSheetOpen(false)}
      title="AI 协写"
      height="70vh"
    >
      {/* 原有的 AI ChatPanel / 协写控件 */}
    </MobileSheet>
  </div>
) : (
  // 桌面端: 原有 SplitPane 结构 — 保持不变
  <>{/* ... 现有代码 ... */}</>
)}
```

- [ ] **Step 2: 类型检查**

```bash
cd frontend && npx tsc --noEmit
```

- [ ] **Step 3: 提交**

```bash
git add frontend/src/pages/ChapterPage.tsx
git commit -m "feat: adapt ChapterPage for mobile with AI sheet overlay"
```

---

### Task 14: 页面响应式适配 — 剩余页面

**Files:**
- Modify: `frontend/src/pages/OutlinePage.tsx`
- Modify: `frontend/src/pages/CharacterPage.tsx`
- Modify: `frontend/src/pages/WorldviewPage.tsx`

**策略**：
- **OutlinePage**: 移动端全屏列表，点击进入详情全屏视图
- **CharacterPage**: 移动端卡片列表 + 底部 Sheet 编辑
- **WorldviewPage**: 移动端手风琴分类

- [ ] **Step 1-3: 逐页适配**

每个页面遵循相同模式：

1. 添加 `import { useIsMobile } from '../hooks/useMediaQuery'`
2. 添加 `const isMobile = useIsMobile()`
3. 在关键分支处 `{isMobile ? <MobileView /> : <DesktopView />}`
4. 替换 Layered Glass class 为 M3 class (`.glass-panel` → `.md-surface-container`)

具体修改参考步骤 12/13 的模式。

- [ ] **Step 4: 类型检查**

```bash
cd frontend && npx tsc --noEmit
```

- [ ] **Step 5: 提交**

```bash
git add frontend/src/pages/OutlinePage.tsx frontend/src/pages/CharacterPage.tsx frontend/src/pages/WorldviewPage.tsx
git commit -m "feat: adapt Outline/Character/Worldview pages for mobile"
```

---

### Task 15: Canvas + 3D 降级 + 最终页面

**Files:**
- Modify: `frontend/src/pages/CanvasPage.tsx`
- Modify: `frontend/src/components/StoryGraph.tsx` (3D 关系图)
- Modify: `frontend/src/pages/ExportPage.tsx`
- Modify: `frontend/src/pages/ImageGenPage.tsx`
- Modify: `frontend/src/components/TTSPlayer.tsx`

**策略**：
- **CanvasPage**: 移动端固定网格（`display: grid`），禁用拖拽
- **StoryGraph**: 移动端 `useIsMobile()` 时渲染 2D DOM 版本替代 Three.js Canvas
- **TTSPlayer**: 移动端全宽底部播放条
- **ExportPage / ImageGenPage**: 单列表单

- [ ] **Step 1-5: 逐组件适配**

遵循 `{isMobile ? <SimplifiedView /> : <DesktopView />}` 模式。

StoryGraph 的 2D 降级：用 div + CSS 绝对定位模拟节点关系，箭头用 SVG `<line>`。

- [ ] **Step 6: 全量类型检查**

```bash
cd frontend && npx tsc --noEmit
```

- [ ] **Step 7: 提交**

```bash
git add frontend/src/pages/CanvasPage.tsx frontend/src/components/StoryGraph.tsx frontend/src/pages/ExportPage.tsx frontend/src/pages/ImageGenPage.tsx frontend/src/components/TTSPlayer.tsx
git commit -m "feat: add mobile fallbacks for Canvas, 3D graph, TTS and remaining pages"
```

---

### Task 16: 全量构建验证 + 最终测试

- [ ] **Step 1: 前端构建**

```bash
cd frontend && npm run build
```

确认 `dist/` 目录生成成功，无构建错误。

- [ ] **Step 2: Go 编译**

```bash
cd D:/AI/wubigork && go build -o build/bin/wubigork-v5.0.0.exe .
```

- [ ] **Step 3: 运行桌面端测试**

启动 `build/bin/wubigork-v5.0.0.exe`：
- 验证 M3 主题切换正常
- 验证 SettingsPage 移动端开关
- 用手机浏览器访问显示二维码

- [ ] **Step 4: 移动端浏览器测试**

手机连接同一局域网：
- 访问 `http://<IP>:8080`
- 验证 TabBar / Drawer 导航
- 验证各页面响应式布局
- 验证触控区尺寸 ≥ 44px

- [ ] **Step 5: Bug 修复 + 提交**

```bash
git add -A && git commit -m "chore: final build verification and polish"
```

---

## 计划完成后的自检清单

- [ ] Spec 覆盖: 每个 design section 都有对应 task
- [ ] 无 TBD/TODO 占位
- [ ] Type 一致性: `Page` 类型在所有组件中一致；`ThemeTokens` 接口前后一致
- [ ] Go 绑定: Wails 自动生成的 JS 绑定与 Go 方法签名匹配
- [ ] CSS 变量: M3 命名 `--md-sys-*` 全局统一
- [ ] 无 Layered Glass 残留: `backdrop-filter`, `bg-glass`, `glass-panel` 全面移除
