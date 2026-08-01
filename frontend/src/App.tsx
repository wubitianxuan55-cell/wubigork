import React, { useEffect } from 'react'
import { ConfigProvider, theme } from 'antd'
import MainLayout from './layouts/MainLayout'
import { useAppStore, getThemeTokens } from './stores/appStore'
import { initBridge } from './api/bridge'
import { initRuntimePolyfill } from './api/runtimePolyfill'

// 在模块作用域最早时机初始化桥接层
// 无论 Wails 原生还是移动端 HTTP，都能确保 window.go.app.App 可用
initBridge()
initRuntimePolyfill()

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

    // ── 未来感扩展令牌 ──
    set('--gaea-glow', tokens.glow)
    set('--gaea-glass-bg', tokens.glassBg)
    set('--gaea-aurora-bg', tokens.auroraBg)

    // ═══ Backward-compat shims (old var names → new M3 tokens) ═══
    set('--accent-rgb', tokens.accentRgb)
    set('--color-primary', tokens.colorPrimary)
    set('--color-text', tokens.colorText)
    set('--color-text-secondary', tokens.colorTextSecondary)
    set('--color-border', tokens.colorBorder)
    set('--color-success', tokens.colorSuccess)
    set('--color-warning', tokens.colorWarning)
    set('--color-bg-container', tokens.colorBgContainer)
    set('--color-bg-layout', tokens.colorBgLayout)
    set('--bg-glass', tokens.surfaceVariant)
    set('--bg-elevated', tokens.surfaceContainer)
    set('--bg-deep', tokens.surfaceDim)
    set('--bg-base', tokens.surface)
    set('--border-subtle', tokens.outlineVariant)
    set('--shadow-sm', tokens.elevation1)
    set('--shadow-md', tokens.elevation2)
    set('--shadow-lg', tokens.elevation3)
    set('--shadow-glow', tokens.elevation5)
    set('--radius-sm', tokens.radiusSm)
    set('--radius-md', tokens.radiusMd)
    set('--radius-lg', tokens.radiusLg)
    set('--radius-xl', tokens.radiusXl)
    set('--transition-fast', tokens.transitionFast)
    set('--transition-normal', tokens.transitionNormal)
    set('--transition-slow', tokens.transitionSlow)
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
