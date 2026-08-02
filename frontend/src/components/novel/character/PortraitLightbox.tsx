import React from 'react'
import Lightbox from '../../Lightbox'

interface PortraitLightboxProps {
  imageUrl: string
  onClose: () => void
}

/** PortraitLightbox — 角色剧照全屏查看（包装 Lightbox，消除空回调） */
const PortraitLightbox: React.FC<PortraitLightboxProps> = ({ imageUrl, onClose }) => (
  <Lightbox
    singleImage={imageUrl}
    results={[]} index={0} characters={[]}
    onClose={onClose}
    onIndexChange={() => {}}
    onDownload={() => {}}
    onReuse={() => {}}
    onSetPortrait={() => {}}
  />
)

export default PortraitLightbox
