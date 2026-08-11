import React, { useEffect, useState } from 'react'
import Lightbox from '../../Lightbox'
import { app } from '../../../gaea/lib/bridge'

interface PortraitLightboxProps {
  imageUrl: string
  onClose: () => void
}

/** PortraitLightbox — 角色剧照全屏查看（包装 Lightbox，消除空回调）；
 *  本地文件路径先经 AttachmentDataURL 读取（剧照已单独落盘）。 */
const PortraitLightbox: React.FC<PortraitLightboxProps> = ({ imageUrl, onClose }) => {
  const [resolved, setResolved] = useState(imageUrl)

  useEffect(() => {
    let live = true
    if (/^(https?:|data:)/i.test(imageUrl)) {
      setResolved(imageUrl)
      return
    }
    app
      .AttachmentDataURL(imageUrl)
      .then((u) => { if (live) setResolved(u) })
      .catch(() => { if (live) setResolved('') })
    return () => { live = false }
  }, [imageUrl])

  if (!resolved) return null
  return (
    <Lightbox
      singleImage={resolved}
      results={[]} index={0} characters={[]}
      onClose={onClose}
      onIndexChange={() => {}}
      onDownload={() => {}}
      onReuse={() => {}}
      onSetPortrait={() => {}}
    />
  )
}

export default PortraitLightbox
