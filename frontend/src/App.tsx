import React, { useEffect, useMemo } from 'react'
import { ConfigProvider, theme } from 'antd'
import MainLayout from './layouts/MainLayout'
import BootSplash from './components/BootSplash'
import BloomSvgDefs from './components/BloomSvgDefs'
import { useAppStore, getThemeTokens, FONT_OPTIONS } from './stores/appStore'
import { initBridge } from './gaea/lib/bridge'
import { initRuntimePolyfill } from './api/runtimePolyfill'
import { LocaleProvider } from './gaea/lib/i18n'

// 在模块作用域最早时机初始化桥接层
// 无论 Wails 原生还是移动端 HTTP，都能确保 window.go.app.App 可用
initBridge()
initRuntimePolyfill()

import { ensureLightContrast } from './lib/accent'
import { subscribeWailsEvent } from './gaea/lib/wailsEvents'
import { handleImageGenPressureEvent } from './utils/imagegenPressure'
import { SYNTAX_PALETTE } from './theme/syntaxPalette'
import { useThemeAutoRotation } from './hooks/useThemeAutoRotation'

/** hex 颜色 → 'r,g,b' 字符串（用于 --accent-rgb 覆盖） */
function hexToRgb(hex: string): string {
  const m = /^#?([0-9a-f]{6})$/i.exec(hex.trim())
  if (!m) return ''
  const n = parseInt(m[1], 16)
  return `${(n >> 16) & 255},${(n >> 8) & 255},${n & 255}`
}

const App: React.FC = () => {
  useThemeAutoRotation()
  const baseTheme = useAppStore((s) => s.baseTheme)
  const darkMode = useAppStore((s) => s.darkMode)
  const density = useAppStore((s) => s.density)
  const motion = useAppStore((s) => s.motion)
  const surfaceStyle = useAppStore((s) => s.surfaceStyle)
  const lightAmbienceStyle = useAppStore((s) => s.lightAmbienceStyle)
  const darkAmbienceStyle = useAppStore((s) => s.darkAmbienceStyle)
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
  // 切亮色主题后若压浅底对比不足，全站 accent 文字会不可读——渲染
  // 时自动深化到 WCAG AA；暗态与用户存储的原始色均不变（v4.274）。
  // useMemo：无 accent 时保持 tokens 同一引用，避免 useEffect 每次渲染重复写 CSS 变量
  const effTokens = useMemo(() => {
    if (!accentColor) return tokens
    const eff = darkMode ? accentColor : ensureLightContrast(accentColor)
    return {
      ...tokens,
      glow: eff,
      colorPrimary: eff,
      accentReadable: eff,
      accentRgb: hexToRgb(eff) || tokens.accentRgb,
    }
  }, [tokens, accentColor, darkMode])

  // 图像生成内存压力预检（v4.409）：后端 comfyui 提交口命中低内存即发
  // imagegen:pressure——App 级全局订阅一次（2 分钟节流在 handler 内），
  // 覆盖绘梦/角色库/sin 全部提交口，卸载只摘自己不碰他人监听。
  useEffect(() => {
    if (!window.runtime?.EventsOn) return
    return subscribeWailsEvent(window.runtime, 'imagegen:pressure', handleImageGenPressureEvent)
  }, [])

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
    set('--gaea-ambience-base', effTokens.bodyBackground)
    set('--md-sys-color-surface', effTokens.surface)
    set('--md-sys-color-on-surface', effTokens.onSurface)
    set('--md-sys-color-surface-variant', effTokens.surfaceVariant)
    set('--md-sys-color-on-surface-variant', effTokens.onSurfaceVariant)
    // surface-container-low：M3 tonal 阶介于 surface 与 container 之间（v4.350
    // 补档——此前全仓零定义，SubagentThread 输入区/VersionTimeline 对比底等
    // 消费点整条属性 invalid）。不逐主题加值，注入处对既有两档插值。
    set('--md-sys-color-surface-container-low', effTokens.surfaceContainerLow)
    set('--md-sys-color-surface-container', effTokens.surfaceContainer)
    set('--md-sys-color-surface-container-high', effTokens.surfaceContainerHigh)
    set('--md-sys-color-surface-container-highest', effTokens.surfaceContainerHighest)
    set('--gaea-surface-tint-1', effTokens.surfaceTint1)
    set('--gaea-surface-tint-2', effTokens.surfaceTint2)
    set('--gaea-surface-tint-3', effTokens.surfaceTint3)
    set('--md-sys-color-surface-dim', effTokens.surfaceDim)
    set('--md-sys-color-surface-bright', effTokens.surfaceBright)

    // M3 Outline
    set('--md-sys-color-outline', effTokens.outline)
    set('--md-sys-color-outline-variant', effTokens.outlineVariant)

    // Background (Ant Design 兼容)
    set('--md-sys-color-bg-container', effTokens.colorBgContainer)
    set('--md-sys-color-bg-layout', effTokens.colorBgLayout)
    // 壳层侧栏底（§7.6 shellSidebar 派生）：侧栏比画布暗一档后退（暗态）/暖灰
    // 压灰（亮态）；Sidebar aside 与 redesign.css rail 消费。
    set('--md-sys-color-shell-sidebar', effTokens.shellSidebar)

    // Text
    set('--md-sys-color-text', effTokens.colorText)
    set('--md-sys-color-text-secondary', effTokens.colorTextSecondary)
    // tertiary 文本档由 ThemeTokens 统一下发，避免跨主题硬编码漂移。
    // 三个变量名保留（25 个消费点零改动），值统一由 colorTextTertiary 下发。
    set('--md-sys-color-text-tertiary', effTokens.colorTextTertiary)
    // 幽灵令牌补定义（v4.350）：--v3-fg-soft/--v3-line-soft/--color-text-tertiary
    // 此前全仓零定义，所有消费点恒走 fallback 亮色值——暗主题下 11~12px 次要
    // 文字/边框对比不足（gray-500 暗底约 3.9:1 < AA）。
    set('--v3-fg-soft', effTokens.colorTextTertiary)
    set('--v3-line-soft', effTokens.colorBorder)
    set('--color-text-tertiary', effTokens.colorTextTertiary)
    set('--md-sys-color-border', effTokens.colorBorder)

    // Semantic
    set('--md-sys-color-success', effTokens.colorSuccess)
    set('--md-sys-color-warning', effTokens.colorWarning)
    set('--md-sys-color-destructive', effTokens.colorDestructive)
    set('--md-sys-color-info', effTokens.colorInfo)
    set('--gaea-accent-readable', effTokens.accentReadable)
    set('--gaea-state-success-primary', effTokens.colorSuccess)
    set('--gaea-state-success-secondary', effTokens.colorSuccess)
    set('--gaea-state-success-tertiary', effTokens.stateSuccessTertiary)
    set('--gaea-state-warning-primary', effTokens.colorWarning)
    set('--gaea-state-warning-secondary', effTokens.colorWarning)
    set('--gaea-state-warning-tertiary', effTokens.stateWarningTertiary)
    set('--gaea-state-warning-label', effTokens.stateWarningLabel)
    set('--gaea-state-danger-primary', effTokens.colorDestructive)
    set('--gaea-state-danger-secondary', effTokens.stateDangerSecondary)
    set('--gaea-state-danger-hover', effTokens.stateDangerHoverBg)
    set('--gaea-business-tertiary', effTokens.businessTertiary)
    set('--gaea-label-secondary', effTokens.labelSecondary)
    set('--gaea-label-tertiary', effTokens.labelTertiary)
    set('--gaea-label-caption', effTokens.labelCaption)
    set('--gaea-label-dimmed', effTokens.labelDimmed)
    set('--gaea-border-1', effTokens.border1)
    set('--gaea-border-2', effTokens.border2)
    set('--gaea-border-3', effTokens.border3)
    set('--gaea-border-4', effTokens.border4)
    set('--gaea-bloom-hairline', effTokens.bloomHairline)
    set('--gaea-bloom-hairline-strong', effTokens.bloomHairlineStrong)
    set('--gaea-bloom-glow', effTokens.bloomGlow)
    set('--gaea-bloom-code-bg', effTokens.bloomCodeBg)
    set('--gaea-sidebar-nav-active', effTokens.sidebarNavActive)
    set('--gaea-sidebar-nav-active-accent', effTokens.sidebarNavActiveAccent)
    set('--gaea-sidebar-nav-hover', effTokens.sidebarNavHover)
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
    set('--gaea-ambience-rgb', effTokens.ambienceRgb)
    set('--gaea-motion-1', effTokens.motion1)
    set('--gaea-motion-2', effTokens.motion2)
    set('--gaea-motion-3', effTokens.motion3)

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
    set('--color-info', effTokens.colorInfo)
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
    // 主题预设下发（v10「天玄」）：页面级星穹底纹与描线按预设开关，
    // 各页面以 data-theme 作用域隔离，其余预设零影响。
    root.dataset.theme = baseTheme
    root.dataset.gaeaMode = darkMode ? 'dark' : 'light'
    root.dataset.gaeaSurface = surfaceStyle
    root.dataset.gaeaAmbienceLight = lightAmbienceStyle
    root.dataset.gaeaAmbienceDark = darkAmbienceStyle
    // 原生控件明暗（v4.351）：滚动条轨道角/表单控件随主题（此前全仓无声明，
    // 暗主题下原生控件残留亮色渲染）
    root.style.colorScheme = darkMode ? 'dark' : 'light'
    const favicon = document.getElementById('gaea-favicon') as HTMLLinkElement | null
    if (favicon) favicon.href = darkMode ? '/favicon-dark.svg' : '/favicon-light.svg'
    // 首帧脚本只读取一次显示模式；运行时切换主题时必须同步这个静态标记，
    // 否则浅色启动后切暗，index.html 的浅色页面底仍会保留，露出白色底板。
    root.toggleAttribute('data-bs-light', !darkMode)

    for (const [scheme, palette] of Object.entries(SYNTAX_PALETTE)) {
      for (const [token, color] of Object.entries(palette)) {
        set(`--syntax-${scheme}-${token}`, color)
      }
    }
  }, [
    effTokens,
    darkMode,
    baseTheme,
    surfaceStyle,
    lightAmbienceStyle,
    darkAmbienceStyle,
  ])

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
        <BloomSvgDefs />
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
