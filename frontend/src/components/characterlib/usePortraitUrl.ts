// usePortraitUrl.ts — 本地文件路径经 AttachmentDataURL 转 data URL 的通用钩子。
// 独立文件（v4.361）：react-refresh 要求组件文件只导出组件；hook 供
// PortraitImg / 头像包装组件共用。
import { useEffect, useState } from 'react'
import { app } from '../../gaea/lib/bridge'

// 本地路径→URL 的成功结果缓存：Popover/网格一次渲染几十上百个头像时，
// 避免每次挂载都重发 AttachmentDataURL 绑定调用；只缓存成功，失败允许重试。
// v4.391：缓存值由 base64 data URL 改为 blob: object URL——data URL 是 MB 级
// 字符串驻留 JS 堆（长会话多图场景堆压力可观），blob URL 把字节迁到浏览器侧
// 存储、Map 里只留短句柄；LRU 上限防无界增长，淘汰延迟 revoke（仍挂载的 img
// 在窗口期内重挂载会重拉新条目，直接 revoke 会裂图）。
const dataUrlCache = new Map<string, string>()

const CACHE_LIMIT = 96
const REVOKE_DELAY_MS = 120_000

// data URL → object URL；环境无 createObjectURL/atob（jsdom 等）原样返回
// data URL，行为与旧版一致。
function toObjectUrl(dataUrl: string): string {
  if (typeof URL === 'undefined' || typeof URL.createObjectURL !== 'function' ||
      typeof atob !== 'function') return dataUrl
  try {
    const comma = dataUrl.indexOf(',')
    if (!dataUrl.startsWith('data:') || comma < 0) return dataUrl
    const meta = dataUrl.slice(5, comma)
    const mime = meta.endsWith(';base64') ? meta.slice(0, -7) : meta
    const bin = atob(dataUrl.slice(comma + 1))
    const bytes = new Uint8Array(bin.length)
    for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
    return URL.createObjectURL(
      new Blob([bytes], { type: mime || 'application/octet-stream' }))
  } catch {
    return dataUrl
  }
}

// Map 迭代序即 LRU 序：命中时 delete+set 刷新 recency，超限淘汰最旧。
function cachePut(path: string, url: string) {
  dataUrlCache.delete(path)
  dataUrlCache.set(path, url)
  while (dataUrlCache.size > CACHE_LIMIT) {
    const oldest = dataUrlCache.keys().next().value
    if (oldest === undefined) break
    const stale = dataUrlCache.get(oldest)
    dataUrlCache.delete(oldest)
    if (stale?.startsWith('blob:') && typeof URL !== 'undefined' &&
        typeof URL.revokeObjectURL === 'function') {
      setTimeout(() => { try { URL.revokeObjectURL(stale) } catch { /* 已释放 */ } }, REVOKE_DELAY_MS)
    }
  }
}

/** 测试专用：清空缓存（含撤销挂起的 revoke 计时器由 vitest 自行 fake 处理）。 */
export function __clearAttachmentUrlCacheForTest() {
  dataUrlCache.clear()
}

// usePortraitUrl：本地文件路径经 AttachmentDataURL 转 data URL 的通用钩子。
// 远程 http(s)/data URL 直接透传。WebView2 拒绝 <img src="file://…">，所有
// 角色剧照上墙必须走本通道（v4.361 清账：PersonaPicker/绘梦/青鸟残留直连点）。
export function usePortraitUrl(src?: string): { url?: string; failed: boolean } {
  const [url, setUrl] = useState<string | undefined>(undefined)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let live = true
    setFailed(false)
    if (!src) {
      setUrl(undefined)
      return
    }
    if (/^(https?:|data:)/i.test(src)) {
      setUrl(src)
      return
    }
    const cached = dataUrlCache.get(src)
    if (cached) {
      cachePut(src, cached) // 命中也走 put：刷新 LRU recency
      setUrl(cached)
      return
    }
    app
      .AttachmentDataURL(src)
      .then((u) => {
        if (!live) return
        const objectUrl = toObjectUrl(u)
        cachePut(src, objectUrl)
        setUrl(objectUrl)
      })
      .catch(() => { if (live) { setUrl(undefined); setFailed(true) } })
    return () => { live = false }
  }, [src])

  return { url, failed }
}

// getCachedAttachmentDataURL：非组件场景/既有 useEffect 直调点的缓存入口——
// 与 usePortraitUrl 共用同一份成功缓存（v4.365 性能轮：办公消息内联图/缩略图/
// inspector 此前每次挂载都重发绑定调用重读 base64）。
export function getCachedAttachmentDataURL(src: string): Promise<string> {
  const cached = dataUrlCache.get(src)
  if (cached) {
    cachePut(src, cached) // 命中也走 put：刷新 LRU recency
    return Promise.resolve(cached)
  }
  // 防御：测试 mock/旧运行时可能缺此方法（原调用点用可选链容忍）——统一
  // 以 reject 表达不可用，消费方既有 .catch 静默/占位路径照常工作。
  if (typeof app.AttachmentDataURL !== 'function') {
    return Promise.reject(new Error('AttachmentDataURL unavailable'))
  }
  return app.AttachmentDataURL(src).then((u) => {
    const objectUrl = toObjectUrl(u)
    cachePut(src, objectUrl)
    return objectUrl
  })
}
