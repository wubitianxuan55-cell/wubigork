import React from 'react'
import GaeaApp from '../gaea/App'
import '../gaea/styles.css'
import '../gaea/tailwind.css'

// 办公板块：完整使用 gaeaW 原生版面（对话区 + 左侧 Sidebar + 右侧面板）。
// 后端绑定通过 gaea/lib/bridge.ts 适配层映射到 wubigrok 的 Gaea* 方法。
function GaeaPage() {
  return <GaeaApp />
}

export default GaeaPage
