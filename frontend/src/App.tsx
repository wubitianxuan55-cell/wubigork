import React, { useEffect, useMemo } from 'react'
import { ConfigProvider, theme } from 'antd'
import MainLayout from './layouts/MainLayout'
import BootSplash from './components/BootSplash'
import { useAppStore, getThemeTokens, FONT_OPTIONS } from './stores/appStore'
import { initBridge } from './gaea/lib/bridge'
import { initRuntimePolyfill } from './api/runtimePolyfill'
import { LocaleProvider } from './gaea/lib/i18n'

// 在模块作用域最早时机初始化桥接层
// 无论 Wails 原生还是移动端 HTTP，都能确保 window.go.app.App 可用
initBridge()
initRuntimePolyfill()

import { ensureLightContrast } from './lib/accent'

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
  // 令牌对象必须 memo：getThemeTokens 每次调用返回**新对象**，若留在渲染体内，
  // 下面的 effTokens useMemo 与主题 effect 会因依赖恒变而每次渲染都重跑
  // （≈45 个 CSS 变量重复 setProperty，v4.349 前实测如此）。
  const tokens = useMemo(() => getThemeTokens(baseTheme, darkMode), [baseTheme, darkMode])

  // 字体设置：预设 key → 完整 font-family 值（默认系统）
  const effFontFamily = FONT_OPTIONS.find((o) => o.key === fontFamily)?.value ?? FONT_OPTIONS[0].value

  // 强调色自定义：覆盖主题默认 glow/primary（accentRgb 供 rgb() 使用）。
  // 亮态下对自定义强调色做对比度保障（ensureLightContrast）：暗色调亮的强调色
  // （如 #1dd7bf）切亮色主题后压浅底对比仅 ~1.5，全站 accent 文字看不清——渲染
  // 时自动深化到 WCAG AA；暗态与用户存储的原始色均不变（v4.274）。
  // useMemo：无 accent 时保持 tokens 同一引用，避免 useEffect 每次渲染重复写 CSS 变量
  const effTokens = useMemo(() => {
    if (!accentColor) return tokens
    const eff = darkMode ? accentColor : ensureLightContrast(accentColor)
    return { ...tokens, glow: eff, colorPrimary: eff, accentRgb: hexToRgb(eff) || tokens.accentRgb }
  }, [tokens, accentColor, darkMode])

  // 同步 M3 CSS 变量到 :root
  useEffect(() => {
    const root = document.documentElement
    const set = (k: string, v: string) => root.style.setProperty(k, v)

    // M3 Primary
    set('--md-sys-color-primary', effTokens.colorPrimary)
    set('--md-sys-color-on-primary', effTokens.onPrimary)
    set('--md-sys-color-primary-container', effTokens.primaryContainer)
    set('--md-sys-color-on-primary-container', effTokens.onPrimaryContainer)

    // M3 Tertiary（v4.183：暖伴侣色，闲庭首页母题与书斋冷色区分）
    set('--md-sys-color-tertiary-container', effTokens.tertiaryContainer)
    set('--md-sys-color-on-tertiary-container', effTokens.onTertiaryContainer)

    // M3 Surface
    set('--md-sys-color-surface', effTokens.surface)
    set('--md-sys-color-on-surface', effTokens.onSurface)
    set('--md-sys-color-surface-variant', effTokens.surfaceVariant)
    set('--md-sys-color-on-surface-variant', effTokens.onSurfaceVariant)
    // surface-container-low：M3 tonal 阶介于 surface 与 container 之间（v4.350
    // 补档——此前全仓零定义，SubagentThread 输入区/VersionTimeline 对比底等
    // 消费点整条属性 invalid）。不逐主题加值，注入处对既有两档插值。
    set('--md-sys-color-surface-container-low', `color-mix(in srgb, ${effTokens.surface} 55%, ${effTokens.surfaceContainer})`)
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
    set('--md-sys-color-text-tertiary', darkMode ? '#8b93a0' : '#6b7280') // hex-exempt
    // 幽灵令牌补定义（v4.350）：--v3-fg-soft/--v3-line-soft/--color-text-tertiary
    // 此前全仓零定义，所有消费点恒走 fallback 亮色值——暗主题下 11~12px 次要
    // 文字/边框对比不足（gray-500 暗底约 3.9:1 < AA）。明暗分档，hex-exempt 同上。
    set('--v3-fg-soft', darkMode ? '#8b93a0' : '#6b7280') // hex-exempt
    set('--v3-line-soft', darkMode ? 'rgba(255,255,255,0.10)' : '#e5e7eb') // hex-exempt
    set('--color-text-tertiary', darkMode ? '#8b93a0' : '#6b7280') // hex-exempt
    set('--md-sys-color-border', effTokens.colorBorder)

    // Semantic
    set('--md-sys-color-success', effTokens.colorSuccess)
    set('--md-sys-color-warning', effTokens.colorWarning)
    set('--md-sys-color-destructive', effTokens.colorDestructive)
    // -info 明暗分档（v4.350）：此前仅 gaea/tailwind.css 定义 --color-info 且转发的
    // --md-sys-color-info 本身无注入源，gaea 板块外消费（novel 风格指纹/CreatePage
    // 图表分组）整条 invalid；亮档取 sky-700（白底 4.6:1），暗档 sky-400。hex-exempt：
    // 主题库无 info 字段，锁定值对齐 v4.349 lightFn 下沉色先例。
    set('--md-sys-color-info', darkMode ? '#38bdf8' : '#0369a1') // hex-exempt
    // -error 语义别名（v4.350 补定义）：gaea 四组件 26 处消费 --md-sys-color-error
    // 但全仓从未定义（无 fallback 整条属性 invalid→失败态红点/红字/红框全消失），
    // 与 destructive 同值双名，12 主题自动正确。
    set('--md-sys-color-error', effTokens.colorDestructive)

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
    set('--color-primary-container', effTokens.primaryContainer)
    set('--color-on-primary', effTokens.onPrimary)
    set('--color-on-primary-container', effTokens.onPrimaryContainer)
    set('--color-surface', effTokens.surface)
    set('--color-text', effTokens.colorText)
    set('--color-text-secondary', effTokens.colorTextSecondary)
    set('--color-border', effTokens.colorBorder)
    set('--color-success', effTokens.colorSuccess)
    set('--color-warning', effTokens.colorWarning)
    set('--color-destructive', effTokens.colorDestructive)
    set('--color-info', darkMode ? '#38bdf8' : '#0369a1') // hex-exempt
    set('--color-bg-container', effTokens.colorBgContainer)
    set('--color-bg-layout', effTokens.colorBgLayout)
    // 3.0「星枢」v3 层消费的容器色 shim（v3-card / v3-rail-item.is-active / v3-model-pill 等）
    set('--color-surface-container', effTokens.surfaceContainer)
    set('--color-surface-container-high', effTokens.surfaceContainerHigh)
    set('--color-surface-container-highest', effTokens.surfaceContainerHighest)
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

    // 代码高亮明暗挂钩（gaea 板块 .hljs-* 调色板切换，色值在 gaea/styles.css
    // :root[data-hl="light"]；刻意内联不走 gaea/lib 导入——那会把消毒链拖进主入口）
    document.documentElement.dataset.hl = darkMode ? 'dark' : 'light'
    // 原生控件明暗（v4.351）：滚动条轨道角/表单控件随主题（此前全仓无声明，
    // 暗主题下原生控件残留亮色渲染）
    root.style.colorScheme = darkMode ? 'dark' : 'light'
  }, [effTokens, darkMode])

  // antd 主题对象同样 memo：ConfigProvider 收到新对象就会重跑主题算法
  // （token 合并 + 组件级派生）。此前内联字面量每次渲染都是新引用（v4.349）。
  const antdTheme = useMemo(() => ({
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
  }), [darkMode, effTokens, density, effFontFamily, fontSize])

  return (
    <div className={[
      density === 'compact' ? 'ui-compact' : '',
      motion === 'reduced' ? 'ui-reduced-motion' : '',
    ].filter(Boolean).join(' ')}>
      <ConfigProvider theme={antdTheme}>
        {/* S2.2 i18n 全铺：LocaleProvider 提到根级，壳层/页面共用 gaea 字典 */}
        <LocaleProvider>
          {/* 启动动画：覆盖首帧，播完自卸（reduced-motion 降级） */}
          <BootSplash />
          <MainLayout />
        </LocaleProvider>
      </ConfigProvider>
    </div>
  )
}

export default App
