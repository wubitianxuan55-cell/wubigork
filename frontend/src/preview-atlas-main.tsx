// 预览专用入口（不入库）：单独挂 CodeAtlasPanel，供 UI 目检明暗两态。
// 关键：md-sys 令牌由 App.tsx 运行时注入 :root，不挂 App 的预览壳必须复刻
// 同一注入（getThemeTokens 同源），否则 var(--md-sys-*) 全 invalid——SVG 在
// 暗底上画黑=整片隐形（2026-11-10 首版预览翻车根因）。
import { StrictMode, useEffect, useState } from 'react'
import { createRoot } from 'react-dom/client'
import { ConfigProvider, theme as antdTheme, Button } from 'antd'
import './index.css'
import './v3/foundation.css'
import CodeAtlasPanel from './components/settings/CodeAtlasPanel'
import { LocaleProvider } from './gaea/lib/i18n'
import { getThemeTokens, useAppStore } from './stores/appStore'

localStorage.setItem('gaea-lang', 'zh')

function applyTheme(darkMode: boolean) {
  const st = useAppStore.getState()
  const tk = getThemeTokens(st.baseTheme, darkMode)
  const root = document.documentElement
  const set = (k: string, v: string) => root.style.setProperty(k, v)
  set('--md-sys-color-primary', tk.colorPrimary)
  set('--md-sys-color-info', darkMode ? '#38bdf8' : '#0369a1') // hex-exempt：与 App.tsx 同值
  set('--md-sys-color-success', tk.colorSuccess)
  set('--md-sys-color-warning', tk.colorWarning)
  set('--md-sys-color-error', tk.colorDestructive)
  set('--md-sys-color-surface', tk.surface)
  set('--md-sys-color-surface-container', tk.surfaceContainer)
  set('--md-sys-color-surface-container-high', tk.surfaceContainerHigh)
  set('--md-sys-color-outline', tk.outline)
  set('--md-sys-color-outline-variant', tk.outlineVariant)
  set('--md-sys-color-text', tk.colorText)
  set('--md-sys-color-text-secondary', tk.colorTextSecondary)
  set('--md-sys-color-text-tertiary', tk.colorTextTertiary)
  set('--md-sys-radius-md', tk.radiusMd)
  set('--md-sys-radius-lg', tk.radiusLg)
  set('--gaea-glow', tk.glow)
  root.dataset.hl = darkMode ? 'dark' : 'light'
  root.style.background = tk.colorBgLayout
  root.style.color = tk.colorText
}

const Preview: React.FC = () => {
  const [dark, setDark] = useState(true)
  useEffect(() => { applyTheme(dark) }, [dark])
  return (
    <ConfigProvider theme={{ algorithm: dark ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm }}>
      <LocaleProvider>
        <div style={{ padding: 20, maxWidth: 1080, margin: '0 auto' }}>
          <Button
            size="small"
            style={{ position: 'fixed', top: 10, right: 12, zIndex: 99 }}
            onClick={() => setDark((v) => !v)}
          >
            切换明暗
          </Button>
          <CodeAtlasPanel />
        </div>
      </LocaleProvider>
    </ConfigProvider>
  )
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <Preview />
  </StrictMode>,
)
