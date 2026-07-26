import React, { useEffect, useState, useCallback, useRef } from 'react'
import { Typography, Button, Space, Tag, Select } from 'antd'
import { DownloadOutlined, SyncOutlined, CloseOutlined, LeftOutlined, RightOutlined, UserOutlined, ZoomInOutlined } from '@ant-design/icons'
import type { GenResult } from './ResultGallery'
import { Z_INDEX } from '../utils/zIndex'

interface Character {
  id: string
  name: string
}

interface Props {
  results: GenResult[]
  index: number
  characters: Character[]
  /** 单张图片模式（剧照等），传此则忽略 results/index */
  singleImage?: string
  onClose: () => void
  onIndexChange: (i: number) => void
  onDownload: (i: number) => void
  onReuse: (i: number) => void
  onSetPortrait: (i: number, charID: string) => void
}

const Lightbox: React.FC<Props> = ({ results, index, characters, singleImage, onClose, onIndexChange, onDownload, onReuse, onSetPortrait }) => {
  const r = singleImage ? null : results[index]
  const imageSrc = singleImage || r?.image
  const isSingle = !!singleImage
  if (!imageSrc) return null

  // 缩放状态
  const [scale, setScale] = useState(1)
  const [position, setPosition] = useState({ x: 0, y: 0 })
  const [dragging, setDragging] = useState(false)
  const dragStart = useRef({ x: 0, y: 0 })
  const posStart = useRef({ x: 0, y: 0 })

  // 切换图片时重置缩放
  useEffect(() => { setScale(1); setPosition({ x: 0, y: 0 }) }, [imageSrc])

  // 键盘
  useEffect(() => {
    if (isSingle) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
      if (e.key === 'ArrowLeft' && index > 0) onIndexChange(index - 1)
      if (e.key === 'ArrowRight' && index < results.length - 1) onIndexChange(index + 1)
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [index, results.length, onClose, onIndexChange, isSingle])

  // 滚轮缩放
  const handleWheel = useCallback((e: React.WheelEvent) => {
    e.preventDefault()
    const delta = e.deltaY > 0 ? -0.15 : 0.15
    setScale((s) => Math.max(0.5, Math.min(5, s + delta)))
  }, [])

  // 拖拽平移（仅在放大时）
  const handleMouseDown = useCallback((e: React.MouseEvent) => {
    if (scale <= 1) return
    e.preventDefault()
    setDragging(true)
    dragStart.current = { x: e.clientX, y: e.clientY }
    posStart.current = { ...position }
  }, [scale, position])

  const handleMouseMove = useCallback((e: React.MouseEvent) => {
    if (!dragging) return
    const dx = e.clientX - dragStart.current.x
    const dy = e.clientY - dragStart.current.y
    setPosition({ x: posStart.current.x + dx, y: posStart.current.y + dy })
  }, [dragging])

  const handleMouseUp = useCallback(() => setDragging(false), [])

  // 双击重置
  const handleDoubleClick = useCallback(() => {
    setScale(1)
    setPosition({ x: 0, y: 0 })
  }, [])

  return (
    <div
      style={{
        position: 'fixed', inset: 0, zIndex: Z_INDEX.LIGHTBOX,
        background: 'rgba(0,0,0,0.92)',
        display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
        cursor: scale > 1 ? (dragging ? 'grabbing' : 'grab') : 'default',
      }}
      onClick={onClose}
    >
      {/* 关闭按钮 */}
      <div style={{ position: 'absolute', top: 16, right: 16, zIndex: 10 }}>
        <Button type="text" icon={<CloseOutlined />} onClick={onClose} style={{ color: '#fff', fontSize: 20 }} />
      </div>

      {/* 缩放提示 */}
      {!isSingle && scale > 1 && (
        <div style={{ position: 'absolute', top: 16, left: 16, zIndex: 10 }}>
          <Tag color="rgba(255,255,255,0.15)" style={{ color: '#ccc' }}>
            <ZoomInOutlined /> {Math.round(scale * 100)}%
          </Tag>
        </div>
      )}

      {/* 左右箭头 */}
      {!isSingle && index > 0 && (
        <Button type="text" icon={<LeftOutlined />}
          onClick={(e) => { e.stopPropagation(); onIndexChange(index - 1) }}
          style={{ position: 'absolute', left: 16, top: '50%', color: '#fff', fontSize: 24, zIndex: 10 }}
        />
      )}
      {!isSingle && index < results.length - 1 && (
        <Button type="text" icon={<RightOutlined />}
          onClick={(e) => { e.stopPropagation(); onIndexChange(index + 1) }}
          style={{ position: 'absolute', right: 16, top: '50%', color: '#fff', fontSize: 24, zIndex: 10 }}
        />
      )}

      {/* 大图区（可缩放拖拽） */}
      <div
        onClick={(e) => e.stopPropagation()}
        onWheel={handleWheel}
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseUp}
        onDoubleClick={(e) => { e.stopPropagation(); handleDoubleClick() }}
        style={{
          maxWidth: isSingle ? '95vw' : '90vw',
          maxHeight: isSingle ? '95vh' : '70vh',
          overflow: 'hidden',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          userSelect: 'none',
        }}
      >
        <img
          src={imageSrc} alt=""
          draggable={false}
          style={{
            maxWidth: '100%', maxHeight: isSingle ? '95vh' : '70vh',
            borderRadius: 8, objectFit: 'contain',
            transform: `scale(${scale}) translate(${position.x / scale}px, ${position.y / scale}px)`,
            transition: dragging ? 'none' : 'transform 0.15s ease-out',
          }}
        />
      </div>

      {/* 信息区（单图模式不显示） */}
      {!isSingle && r && (
        <div onClick={(e) => e.stopPropagation()} style={{ marginTop: 16, maxWidth: '90vw', textAlign: 'center' }}>
          <Typography.Text style={{ color: '#ccc', fontSize: 12, display: 'block', marginBottom: 8, maxHeight: 60, overflow: 'hidden' }}>
            {r.prompt}
          </Typography.Text>
          <Space size={8} wrap>
            <Tag color="blue">🎲 种子: {r.seed}</Tag>
            <Tag color="green">{r.model}</Tag>
            <Tag>{r.size}</Tag>
            <Tag>⏱ {r.time}s</Tag>
          </Space>
          <div style={{ marginTop: 12 }}>
            <Space>
              <Button icon={<DownloadOutlined />} onClick={() => onDownload(index)} ghost>下载</Button>
              <Button icon={<SyncOutlined />} onClick={() => onReuse(index)} ghost>重用参数</Button>
              {characters.length > 0 && (
                <Select
                  placeholder={<span><UserOutlined /> 设为剧照</span>}
                  style={{ width: 140 }}
                  size="small"
                  onChange={(charID: string) => onSetPortrait(index, charID)}
                  options={characters.map((c) => ({ label: c.name, value: c.id }))}
                  popupMatchSelectWidth={false}
                />
              )}
            </Space>
          </div>
        </div>
      )}
    </div>
  )
}

export default Lightbox
