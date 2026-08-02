import React from 'react'
import GaeaApp from '../gaea/App'
import { LocaleProvider } from '../gaea/lib/i18n'
import FeatureModelBar from '../components/FeatureModelBar'
import '../gaea/styles.css'
import '../gaea/tailwind.css'

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
      {/* 绑定模型卡（左下角浮动） */}
      <div style={{ position: 'absolute', left: 12, bottom: 12, zIndex: 50 }}>
        <FeatureModelBar feature="gaea" label="办公" />
      </div>
    </div>
  )
}

export default GaeaPage
