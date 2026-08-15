import React, { useEffect, useMemo } from 'react'
import { ConfigProvider, theme } from 'antd'
import MainLayout from './layouts/MainLayout'
import { useAppStore, getThemeTokens, FONT_OPTIONS } from './stores/appStore'
import { initBridge } from './gaea/lib/bridge'
import { initRuntimePolyfill } from './api/runtimePolyfill'

// 在模块作用域最早时机初始化桥接层
// 无论 Wails 原生还是移动端 HTTP，都能确保 window.go.app.App 可用
initBridge()
initRuntimePolyfill()

/** hex 颜色 → 'r,g,b' 字符串（用于 --accent-rgb 覆盖） */
function hexToRgb(hex: string): string {
  const m = /^#?([0-9a-f]{6})$/i.exec(hex.trim())
  if (!m) return ''
  const n = parseInt(m[1], 16)
  return `${(n >> 16) & 255},${(n >> 8) & 255},${n & 255}`
}

const App: React.FC = () => {
  const baseTheme = useAppStore((s) => s.baseTheme)
  const darkMode = useAppStore((s) => s.darkMode)
  const density = useAppStore((s) => s.density)
  const motion = useAppStore((s) => s.motion)
  const accentColor = useAppStore((s) => s.accentColor)
  const fontFamily = useAppStore((s) => s.fontFamily)
  const fontSize = useAppStore((s) => s.fontSize)
  const tokens = getThemeTokens(baseTheme, darkMode)

  // 字体设置：预设 key → 完整 font-family 值（默认系统）
  const effFontFamily = FONT_OPTIONS.find((o) => o.key === fontFamily)?.value ?? FONT_OPTIONS[0].value

  // 强调色自定义：覆盖主题默认 glow/primary（accentRgb 供 rgb() 使用）
  // useMemo：无 accent 时保持 tokens 同一引用，避免 useEffect 每次渲染重复写 CSS 变量
  const effTokens = useMemo(
    () => accentColor
      ? { ...tokens, glow: accentColor, colorPrimary: accentColor, accentRgb: hexToRgb(accentColor) || tokens.accentRgb }
      : tokens,
    [tokens, accentColor],
  )

  // 同步 M3 CSS 变量到 :root
  useEffect(() => {
    const root = document.documentElement
    const set = (k: string, v: string) => root.style.setProperty(k, v)

    // M3 Primary
    set('--md-sys-color-primary', effTokens.colorPrimary)
    set('--md-sys-color-on-primary', effTokens.onPrimary)
    set('--md-sys-color-primary-container', effTokens.primaryContainer)
    set('--md-sys-color-on-primary-container', effTokens.onPrimaryContainer)

    // M3 Surface
    set('--md-sys-color-surface', effTokens.surface)
    set('--md-sys-color-on-surface', effTokens.onSurface)
    set('--md-sys-color-surface-variant', effTokens.surfaceVariant)
    set('--md-sys-color-on-surface-variant', effTokens.onSurfaceVariant)
    set('--md-sys-color-surface-container', effTokens.surfaceContainer)
    set('--md-sys-color-surface-container-high', effTokens.surfaceContainerHigh)
    set('--md-sys-color-surface-container-highest', effTokens.surfaceContainerHighest)
    set('--md-sys-color-surface-dim', effTokens.surfaceDim)

    // M3 Outline
    set('--md-sys-color-outline', effTokens.outline)
    set('--md-sys-color-outline-variant', effTokens.outlineVariant)

    // Background (Ant Design 兼容)
    set('--md-sys-color-bg-container', effTokens.colorBgContainer)
    set('--md-sys-color-bg-layout', effTokens.colorBgLayout)

    // Text
    set('--md-sys-color-text', effTokens.colorText)
    set('--md-sys-color-text-secondary', effTokens.colorTextSecondary)
    set('--md-sys-color-border', effTokens.colorBorder)

    // Semantic
    set('--md-sys-color-success', effTokens.colorSuccess)
    set('--md-sys-color-warning', effTokens.colorWarning)
    set('--md-sys-color-destructive', effTokens.colorDestructive)

    // Elevation
    set('--md-sys-elevation-1', effTokens.elevation1)
    set('--md-sys-elevation-2', effTokens.elevation2)
    set('--md-sys-elevation-3', effTokens.elevation3)
    set('--md-sys-elevation-4', effTokens.elevation4)
    set('--md-sys-elevation-5', effTokens.elevation5)

    // Radius
    set('--md-sys-radius-sm', effTokens.radiusSm)
    set('--md-sys-radius-md', effTokens.radiusMd)
    set('--md-sys-radius-lg', effTokens.radiusLg)
    set('--md-sys-radius-xl', effTokens.radiusXl)

    // Transition
    set('--md-sys-transition-fast', effTokens.transitionFast)
    set('--md-sys-transition-normal', effTokens.transitionNormal)
    set('--md-sys-transition-slow', effTokens.transitionSlow)

    // ── 未来感扩展令牌 ──
    set('--gaea-glow', effTokens.glow)
    set('--gaea-glass-bg', effTokens.glassBg)
    set('--gaea-aurora-bg', effTokens.auroraBg)

    // ═══ Backward-compat shims (old var names → new M3 tokens) ═══
    set('--accent-rgb', effTokens.accentRgb)
    set('--color-primary', effTokens.colorPrimary)
    set('--color-text', effTokens.colorText)
    set('--color-text-secondary', effTokens.colorTextSecondary)
    set('--color-border', effTokens.colorBorder)
    set('--color-success', effTokens.colorSuccess)
    set('--color-warning', effTokens.colorWarning)
    set('--color-destructive', effTokens.colorDestructive)
    set('--color-bg-container', effTokens.colorBgContainer)
    set('--color-bg-layout', effTokens.colorBgLayout)
    set('--bg-glass', effTokens.surfaceVariant)
    set('--bg-elevated', effTokens.surfaceContainer)
    set('--bg-deep', effTokens.surfaceDim)
    set('--bg-base', effTokens.surface)
    set('--border-subtle', effTokens.outlineVariant)
    set('--shadow-sm', effTokens.elevation1)
    set('--shadow-md', effTokens.elevation2)
    set('--shadow-lg', effTokens.elevation3)
    set('--shadow-glow', effTokens.elevation5)
    set('--radius-sm', effTokens.radiusSm)
    set('--radius-md', effTokens.radiusMd)
    set('--radius-lg', effTokens.radiusLg)
    set('--radius-xl', effTokens.radiusXl)
    set('--transition-fast', effTokens.transitionFast)
    set('--transition-normal', effTokens.transitionNormal)
    set('--transition-slow', effTokens.transitionSlow)
  }, [effTokens])

  return (
    <div className={[
      density === 'compact' ? 'ui-compact' : '',
      motion === 'reduced' ? 'ui-reduced-motion' : '',
    ].filter(Boolean).join(' ')}>
      <ConfigProvider
        theme={{
          algorithm: darkMode ? theme.darkAlgorithm : theme.defaultAlgorithm,
          token: {
            colorPrimary: effTokens.colorPrimary,
            colorBgContainer: effTokens.colorBgContainer,
            colorBgLayout: effTokens.colorBgLayout,
            colorText: effTokens.colorText,
            colorTextSecondary: effTokens.colorTextSecondary,
            colorBorder: effTokens.colorBorder,
            colorError: effTokens.colorDestructive, // antd 危险语义对齐 gaea 令牌（删除/错误确认）
            borderRadius: density === 'compact' ? 12 : 16,       // M3 默认更大圆角
            borderRadiusLG: density === 'compact' ? 14 : 20,
            borderRadiusSM: density === 'compact' ? 8 : 12,
            fontFamily: effFontFamily,
            fontSize: fontSize,
            controlHeight: density === 'compact' ? 32 : 36,
            lineHeight: 1.5,
          },
        }}
      >
        <MainLayout />
      </ConfigProvider>
    </div>
  )
}

export default App
