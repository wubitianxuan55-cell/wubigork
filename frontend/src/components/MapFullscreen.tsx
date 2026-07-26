import React from 'react'
import { Button } from 'antd'
import { Z_INDEX } from '../utils/zIndex'

interface MapFullscreenProps {
  worldMapImage: string
  onClose: () => void
}

/** MapFullscreen — 世界地图全屏覆盖层（仅 AI 图片） */
const MapFullscreen: React.FC<MapFullscreenProps> = ({ worldMapImage, onClose }) => (
  <div
    onClick={onClose}
    style={{
      position: 'fixed', inset: 0, zIndex: Z_INDEX.FULLSCREEN_OVERLAY,
      background: 'rgba(0,0,0,0.92)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      cursor: 'zoom-out',
    }}
  >
    {worldMapImage ? (
      <img src={worldMapImage} alt="世界地图"
        onClick={(e) => e.stopPropagation()}
        style={{ maxWidth: '95vw', maxHeight: '95vh', objectFit: 'contain', borderRadius: 8 }} />
    ) : (
      <div style={{ color: '#666', fontSize: 14 }}>
        暂无 AI 生成的地图图片
      </div>
    )}
    <Button
      onClick={onClose}
      style={{
        position: 'absolute', top: 16, right: 16,
        background: 'rgba(0,0,0,0.6)', border: 'none', color: '#fff',
        borderRadius: 'var(--radius-md)', fontSize: 18,
      }}
    >✕</Button>
  </div>
)

export default MapFullscreen
