import React, { useEffect, useState } from 'react'
import { Z_INDEX } from '../utils/zIndex'

interface Props {
  open: boolean
  onClose: () => void
  children: React.ReactNode
  title?: string
  height?: string  // 默认 '60vh'
}

const MobileSheet: React.FC<Props> = ({
  open, onClose, children, title, height = '60vh'
}) => {
  const [visible, setVisible] = useState(false)
  const [animating, setAnimating] = useState(false)

  useEffect(() => {
    if (open) {
      setVisible(true)
      requestAnimationFrame(() => setAnimating(true))
    } else {
      setAnimating(false)
      const timer = setTimeout(() => setVisible(false), 300)
      return () => clearTimeout(timer)
    }
  }, [open])

  if (!visible) return null

  return (
    <div
      style={{
        position: 'fixed', inset: 0, zIndex: Z_INDEX.SHEET_BACKDROP,
        display: 'flex', alignItems: 'flex-end',
      }}
    >
      <div
        onClick={onClose}
        style={{
          position: 'absolute', inset: 0,
          background: 'rgba(0,0,0,0)',
          transition: 'background 300ms cubic-bezier(0.2, 0, 0, 1)',
          ...(animating ? { background: 'rgba(0,0,0,0.4)' } : {}),
        }}
      />
      <div
        style={{
          position: 'relative',
          width: '100%',
          maxHeight: height,
          background: 'var(--md-sys-color-surface-container-high)',
          borderTopLeftRadius: 28,
          borderTopRightRadius: 28,
          boxShadow: 'var(--md-sys-elevation-5)',
          display: 'flex',
          flexDirection: 'column',
          transform: animating ? 'translateY(0)' : 'translateY(100%)',
          transition: 'transform 300ms cubic-bezier(0.2, 0, 0, 1)',
          paddingBottom: 'env(safe-area-inset-bottom, 0px)',
        }}
      >
        {/* 拖拽手柄 */}
        <div style={{
          display: 'flex', justifyContent: 'center', padding: '8px 0',
        }}>
          <div style={{
            width: 32, height: 4, borderRadius: 2,
            background: 'var(--md-sys-color-on-surface-variant)',
            opacity: 0.4,
          }} />
        </div>

        {title && (
          <div style={{
            padding: '0 24px 12px', fontSize: 16, fontWeight: 600,
            color: 'var(--md-sys-color-on-surface)',
          }}>
            {title}
          </div>
        )}

        <div style={{ flex: 1, overflow: 'auto', padding: '0 16px 16px' }}>
          {children}
        </div>
      </div>
    </div>
  )
}

export default MobileSheet
