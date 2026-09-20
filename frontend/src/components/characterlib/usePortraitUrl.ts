// usePortraitUrl.ts — 本地文件路径经 AttachmentDataURL 转 data URL 的通用钩子。
// 独立文件（v4.361）：react-refresh 要求组件文件只导出组件；hook 供
// PortraitImg / 头像包装组件共用。
import { useEffect, useState } from 'react'
import { app } from '../../gaea/lib/bridge'

// 本地路径→data URL 的成功结果缓存：Popover/网格一次渲染几十上百个头像时，
// 避免每次挂载都重发 AttachmentDataURL 绑定调用；只缓存成功，失败允许重试。
const dataUrlCache = new Map<string, string>()

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
      setUrl(cached)
      return
    }
    app
      .AttachmentDataURL(src)
      .then((u) => {
        if (!live) return
        dataUrlCache.set(src, u)
        setUrl(u)
      })
      .catch(() => { if (live) { setUrl(undefined); setFailed(true) } })
    return () => { live = false }
  }, [src])

  return { url, failed }
}

