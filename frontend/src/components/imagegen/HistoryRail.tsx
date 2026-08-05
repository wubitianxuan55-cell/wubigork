import React from 'react'
import { Button, Typography } from 'antd'
import { DeleteOutlined, HistoryOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'
import type { GenResult } from './types'

interface Props {
  history: GenResult[]
  selectedIndex: number
  onSelect: (index: number) => void
  onClear: () => void
}

export const HistoryRail: React.FC<Props> = ({ history, selectedIndex, onSelect, onClear }) => (
  <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8, flexShrink: 0 }}>
      <Typography.Text style={{ fontSize: 11, fontWeight: 500, letterSpacing: '0.06em', color: C('color-text-secondary') }}>
        <HistoryOutlined style={{ marginRight: 5 }} />
        历史 {history.length > 0 ? `(${history.length})` : ''}
      </Typography.Text>
      {history.length > 0 && (
        <Button
          type="text" size="small" icon={<DeleteOutlined />}
          onClick={onClear}
          title="清空历史"
          style={{ color: C('color-text-secondary'), fontSize: 11, padding: '0 2px' }}
        />
      )}
    </div>

    {history.length === 0 ? (
      <Typography.Text style={{
        color: C('color-text-secondary'), fontSize: 11, textAlign: 'center', display: 'block', padding: '18px 8px',
      }}>
        生成的图片会出现在这里
      </Typography.Text>
    ) : (
      <div style={{ display: 'flex', flexDirection: 'column', gap: 6, overflowY: 'auto', flex: 1 }}>
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
              <img src={h.image} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
            </div>
          )
        })}
      </div>
    )}
  </div>
)
