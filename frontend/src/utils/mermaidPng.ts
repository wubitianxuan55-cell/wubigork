/**
 * 把 Mermaid 代码渲染为 PNG data URL（绘梦「流程图/框架图」模式使用）。
 * 渲染在浏览器本地完成，中文使用系统字体，清晰无乱码。
 */
import mermaid from 'mermaid'

let mermaidReady = false

function ensureMermaid() {
  if (mermaidReady) return
  mermaidReady = true
  mermaid.initialize({
    startOnLoad: false,
    securityLevel: 'loose',
    theme: 'default',
    fontFamily: 'system-ui, "Microsoft YaHei", "PingFang SC", "Noto Sans CJK SC", sans-serif',
    themeVariables: { fontFamily: 'system-ui, "Microsoft YaHei", "PingFang SC", sans-serif' },
  })
}

export interface RenderedPng {
  dataUrl: string
  width: number
  height: number
}

/** 渲染 Mermaid 代码并导出 PNG；失败返回 null。 */
export async function renderMermaidToPng(code: string): Promise<RenderedPng | null> {
  ensureMermaid()
  const id = `ig-mermaid-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
  let svg = ''
  try {
    const r = await mermaid.render(id, code)
    svg = r.svg
  } catch (e) {
    console.error('[mermaidPng] render failed:', e)
    return null
  }

  const holder = document.createElement('div')
  holder.style.position = 'absolute'
  holder.style.left = '-99999px'
  holder.style.top = '0'
  document.body.appendChild(holder)
  holder.innerHTML = svg
  const el = holder.querySelector('svg')
  if (!el) {
    holder.remove()
    return null
  }

  const vb = el.getAttribute('viewBox')?.split(/\s+/).map(Number)
  const rect = el.getBoundingClientRect()
  const w = vb?.[2] || rect.width || 800
  const h = vb?.[3] || rect.height || 600
  const scale = 2
  const canvas = document.createElement('canvas')
  canvas.width = Math.max(1, Math.round(w * scale))
  canvas.height = Math.max(1, Math.round(h * scale))
  const ctx = canvas.getContext('2d')
  if (!ctx) {
    holder.remove()
    return null
  }
  ctx.fillStyle = '#ffffff'
  ctx.fillRect(0, 0, canvas.width, canvas.height)

  const xml = new XMLSerializer().serializeToString(el)
  holder.remove()

  const img = new Image()
  return new Promise((resolve) => {
    img.onload = () => {
      ctx.drawImage(img, 0, 0, canvas.width, canvas.height)
      resolve({ dataUrl: canvas.toDataURL('image/png'), width: canvas.width, height: canvas.height })
    }
    img.onerror = () => resolve(null)
    img.src = 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(xml)
  })
}
