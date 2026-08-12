import React from 'react'
import GaeaApp from '../gaea/App'
import { LocaleProvider } from '../gaea/lib/i18n'
import '../gaea/styles.css'
import '../gaea/tailwind.css'
import '../gaea/redesign.css'

// 办公板块：完整使用 gaeaW 原生版面（对话区 + 左侧 Sidebar + 右侧面板）。
// 后端绑定通过 gaea/lib/bridge.ts 适配层映射到 gaea 的 Gaea* 方法。
// LocaleProvider 必须包在 GaeaApp 外层：GaeaApp 顶层自身调用 useT()，
// Provider 若只放在其 return JSX 内盖不住组件自身，运行时会抛
// 'useI18n must be used within a LocaleProvider'。
function GaeaPage() {
  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column', position: 'relative' }}>
      <div style={{ flex: 1, minHeight: 0 }}>
        <LocaleProvider>
          <GaeaApp />
        </LocaleProvider>
      </div>
    </div>
  )
}

export default GaeaPage
