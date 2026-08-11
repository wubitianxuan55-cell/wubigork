import { useEffect, useState } from 'react'
import { app } from '../../gaea/lib/bridge'

// PortraitImg 角色剧照渲染：远程 URL / data URL 直接用；
// 本地文件路径（剧照落盘后 characters.json 只存路径）通过
// AttachmentDataURL 读取，避免巨型 base64 塞进 JSON 导致界面卡死。
export function PortraitImg({
  src,
  alt,
  className,
  style,
}: {
  src?: string
  alt?: string
  className?: string
  style?: React.CSSProperties
}) {
  const [url, setUrl] = useState<string | undefined>(undefined)

  useEffect(() => {
    let live = true
    if (!src) {
      setUrl(undefined)
      return
    }
    if (/^(https?:|data:)/i.test(src)) {
      setUrl(src)
      return
    }
    app
      .AttachmentDataURL(src)
      .then((u) => { if (live) setUrl(u) })
      .catch(() => { if (live) setUrl(undefined) })
    return () => { live = false }
  }, [src])

  if (!url) return null
  return <img src={url} alt={alt ?? ''} className={className} style={style} />
}
