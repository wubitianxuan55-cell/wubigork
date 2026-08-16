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
    app
      .AttachmentDataURL(src)
      .then((u) => { if (live) setUrl(u) })
      .catch(() => { if (live) { setUrl(undefined); setFailed(true) } })
    return () => { live = false }
  }, [src])

  if (!url) {
    // 剧照不可用（本地文件缺失 / 远程 URL 已过期）：显示占位首字，避免裂图
    if (!failed) return null
    return (
      <div
        aria-label={alt || '剧照缺失'}
        title={alt || '剧照缺失'}
        style={{
          width: '100%',
          height: '100%',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: 44,
          fontWeight: 700,
          color: 'var(--gaea-glow)',
          opacity: 0.7,
          background:
            'radial-gradient(130% 90% at 50% 0%, color-mix(in srgb, var(--gaea-glow) 18%, transparent), transparent 58%), linear-gradient(155deg, color-mix(in srgb, var(--gaea-glow) 8%, transparent), transparent 55%), var(--md-sys-color-surface-container-high)',
        }}
      >
        {alt?.trim().slice(0, 1) || '?'}
      </div>
    )
  }
  return (
    <img
      src={url}
      alt={alt ?? ''}
      className={className}
      style={style}
      onError={() => { setUrl(undefined); setFailed(true) }}
    />
  )
}
