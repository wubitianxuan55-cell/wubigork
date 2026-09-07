/**
 * exportArtifact.ts — 图面导出 DOM 管线（v4.132.0 刀D1）
 *
 * SVG 字符串 → PNG / JPEG / 打印 / 下载。构建器（ganttExport.ts）纯函数出
 * SVG 字符串；本模块只做浏览器编码：Blob URL → Image → canvas → toBlob，
 * 失败逐层抛错由调用方提示。打印走隐藏 iframe（WebView2 打开系统打印，
 * 「另存为 PDF」即得 PDF 文件）。
 */
import { jpegBytesToPdf, bytesToBlob } from './imgPdf'

/** SVG 字符串尺寸（构建器写在 svg 根节点 width/height 属性上的数字） */
function svgSize(svg: string): { w: number; h: number } {
  const el = new DOMParser().parseFromString(svg, 'image/svg+xml').documentElement
  const w = Number(el.getAttribute('width'))
  const h = Number(el.getAttribute('height'))
  if (!Number.isFinite(w) || !Number.isFinite(h) || w <= 0 || h <= 0) throw new Error('SVG 尺寸缺失')
  return { w, h }
}

/** 注入 viewBox（预览用：有 viewBox 的 SVG 可被 CSS 等比缩放进容器） */
export function svgWithViewBox(svg: string): string {
  const tag = /<svg[^>]*>/i.exec(svg)?.[0]
  if (!tag || /viewBox=/i.test(tag)) return svg
  const w = /width="([\d.]+)"/i.exec(tag)
  const h = /height="([\d.]+)"/i.exec(tag)
  if (!w || !h) return svg
  return svg.replace(tag, tag.replace(/height="([\d.]+)"/i, `viewBox="0 0 ${w[1]} ${h[1]}" height="$1"`))
}

/** SVG 字符串 → Image（Blob URL 加载；加载完成后即回收 URL） */
function svgToImage(svg: string): Promise<HTMLImageElement> {
  const url = URL.createObjectURL(new Blob([svg], { type: 'image/svg+xml;charset=utf-8' }))
  return new Promise<HTMLImageElement>((resolve, reject) => {
    const img = new Image()
    img.onload = () => resolve(img)
    img.onerror = () => reject(new Error('图面渲染失败'))
    img.src = url
  }).finally(() => URL.revokeObjectURL(url))
}

/** SVG → canvas（scale 倍超采样；白底） */
async function svgToCanvas(svg: string, scale: number): Promise<HTMLCanvasElement> {
  const { w, h } = svgSize(svg)
  const img = await svgToImage(svg)
  const canvas = document.createElement('canvas')
  canvas.width = Math.round(w * scale)
  canvas.height = Math.round(h * scale)
  const ctx = canvas.getContext('2d')
  if (!ctx) throw new Error('画布不可用')
  ctx.fillStyle = 'white'
  ctx.fillRect(0, 0, canvas.width, canvas.height)
  ctx.drawImage(img, 0, 0, canvas.width, canvas.height)
  return canvas
}

function canvasToBlob(canvas: HTMLCanvasElement, type: string, quality?: number): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (b) => (b ? resolve(b) : reject(new Error('编码失败'))),
      type,
      quality,
    )
  })
}

/** 图面 SVG → PNG Blob（scale=2 超采样，打印清晰度） */
export async function svgToPngBlob(svg: string, scale = 2): Promise<Blob> {
  return canvasToBlob(await svgToCanvas(svg, scale), 'image/png')
}

/** 图面 SVG → PDF Blob（canvas JPEG 直嵌单页） */
export async function svgToPdfBlob(svg: string, scale = 2): Promise<Blob> {
  const canvas = await svgToCanvas(svg, scale)
  const dataUrl = canvas.toDataURL('image/jpeg', 0.92)
  const b64 = dataUrl.slice(dataUrl.indexOf(',') + 1)
  const bin = atob(b64)
  const jpeg = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) jpeg[i] = bin.charCodeAt(i)
  return bytesToBlob(jpegBytesToPdf(jpeg, canvas.width, canvas.height), 'application/pdf')
}

/** 触发浏览器下载（与导出 XML 同一 <a download> 机制） */
export function downloadBlob(blob: Blob, name: string): void {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = name
  a.click()
  URL.revokeObjectURL(url)
}

/** 图面打印：隐藏 iframe 载入 SVG，唤起系统打印（另存为 PDF 可得 PDF） */
export function printSvg(svg: string): void {
  const frame = document.createElement('iframe')
  frame.style.position = 'fixed'
  frame.style.right = '0'
  frame.style.bottom = '0'
  frame.style.width = '0'
  frame.style.height = '0'
  frame.style.border = '0'
  frame.onload = () => {
    try {
      frame.contentWindow?.focus()
      frame.contentWindow?.print()
    } finally {
      window.setTimeout(() => frame.remove(), 5000)
    }
  }
  document.body.appendChild(frame)
  frame.srcdoc = `<!doctype html><html><head><meta charset="utf-8"><style>@page{margin:8mm}body{margin:0}svg{max-width:100%;height:auto}</style></head><body>${svg}</body></html>`
}
