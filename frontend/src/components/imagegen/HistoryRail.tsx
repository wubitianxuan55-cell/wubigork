import React from 'react'
import { Button, Typography } from 'antd'
import { DeleteOutlined, HistoryOutlined, MenuFoldOutlined, MenuUnfoldOutlined, PictureOutlined, VideoCameraOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'
import { mediaIsVideo } from './media'
import type { GenResult } from './types'

interface Props {
  history: GenResult[]
  selectedIndex: number
  onSelect: (index: number) => void
  onClear: () => void
  onRegenerateMeta?: (item: GenResult) => void
  collapsed?: boolean
  onToggleCollapse?: () => void
}

export const HistoryRail: React.FC<Props> = ({ history, selectedIndex, onSelect, onClear, onRegenerateMeta, collapsed = false, onToggleCollapse }) => (
  <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8, flexShrink: 0 }}>
      {!collapsed && (
        <Typography.Text style={{ fontSize: 11, fontWeight: 500, letterSpacing: '0.06em', color: C('color-text-secondary') }}>
          <HistoryOutlined style={{ marginRight: 5 }} />
          历史 {history.length > 0 ? `(${history.length})` : ''}
        </Typography.Text>
      )}
      {!collapsed && history.length > 0 && (
        <Button
          type="text" size="small" icon={<DeleteOutlined />}
          onClick={onClear}
          title="清空历史"
          style={{ color: C('color-text-secondary'), fontSize: 11, padding: '0 2px' }}
        />
      )}
      {onToggleCollapse && (
        <Button
          type="text" size="small"
          icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
          onClick={onToggleCollapse}
          title={collapsed ? '展开历史' : '收起历史'}
          style={{ color: C('color-text-secondary'), fontSize: 11, padding: '0 2px', marginLeft: collapsed ? 0 : 4 }}
        />
      )}
    </div>

    {collapsed ? (
      <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: C('color-text-secondary'), fontSize: 11 }}>
        {history.length}
      </div>
    ) : history.length === 0 ? (
      <Typography.Text style={{
        color: C('color-text-secondary'), fontSize: 11, textAlign: 'center', display: 'block', padding: '18px 8px',
      }}>
        生成的图片与视频会出现在这里
      </Typography.Text>
    ) : (
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 6, overflowY: 'auto', flex: 1 }}>
        {history.map((h, i) => {
          const selected = i === selectedIndex
          return (
            <div
              key={i}
              onClick={() => onSelect(i)}
              title={h.prompt?.slice(0, 40) || h.model}
              className="img-card"
              style={{
                width: '100%', aspectRatio: '1', flexShrink: 0, cursor: 'pointer',
                borderRadius: 'var(--radius-sm)', overflow: 'hidden',
                border: selected ? '2px solid var(--color-primary)' : '1px solid var(--border-subtle)',
                boxShadow: selected ? 'var(--shadow-sm)' : 'none',
              }}
            >
              {!h.image ? (
                <div style={{ width: '100%', height: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', color: C('color-text-secondary'), gap: 4, padding: 4, background: 'var(--bg-elevated)' }}>
                  <PictureOutlined style={{ fontSize: 16, opacity: 0.55 }} />
                  <Typography.Text style={{ fontSize: 9, color: C('color-text-secondary'), textAlign: 'center', lineHeight: 1.2 }}>
                    {h.model || '历史记录'}
                  </Typography.Text>
                  {onRegenerateMeta && (
                    <Button size="small" type="text"
                      onClick={(e) => { e.stopPropagation(); onRegenerateMeta(h) }}
                      style={{ fontSize: 9, padding: '0 3px', height: 18, color: 'var(--color-primary)' }}>
                      重新生成
                    </Button>
                  )}
                </div>
              ) : mediaIsVideo(h.image) ? (
                <div style={{ position: 'relative', width: '100%', height: '100%', background: '#000' }}> {/* hex-exempt 图片覆盖层 chrome */}
                  <video src={h.image} muted playsInline preload="metadata" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                  <span style={{
                    position: 'absolute', right: 3, bottom: 3, width: 16, height: 16, borderRadius: '50%',
                    background: 'rgba(0,0,0,0.65)', color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', // hex-exempt 图片覆盖层 chrome
                    fontSize: 9,
                  }}>
                    <VideoCameraOutlined />
                  </span>
                </div>
              ) : (
                <img src={h.image} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
              )}
            </div>
          )
        })}
      </div>
    )}
  </div>
)
