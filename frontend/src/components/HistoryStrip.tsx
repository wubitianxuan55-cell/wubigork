import React from 'react'
import { Typography, Button } from 'antd'
import { DeleteOutlined } from '@ant-design/icons'
import { C } from '../utils/theme'
import type { GenResult } from './ResultGallery'

interface Props {
  history: GenResult[]
  onSelect: (index: number) => void
  onClear: () => void
}

const HistoryStrip: React.FC<Props> = ({ history, onSelect, onClear }) => {
  if (history.length === 0) return (
    <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11, textAlign: 'center', display: 'block', padding: 20 }}>
      暂无历史
    </Typography.Text>
  )

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 6, flexShrink: 0 }}>
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>
          📜 ({history.length})
        </Typography.Text>
        <Button
          type="text" size="small" icon={<DeleteOutlined />} onClick={onClear}
          style={{ color: C('color-text-secondary'), fontSize: 11, padding: '0 2px' }}
        />
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 6, overflowY: 'auto', flex: 1 }}>
        {history.map((h, i) => (
          <div
            key={i} onClick={() => onSelect(i)}
            style={{
              width: '100%', aspectRatio: '1', flexShrink: 0, cursor: 'pointer',
              borderRadius: 'var(--radius-sm)', overflow: 'hidden',
              border: '1px solid var(--border-subtle)',
            }}
          >
            <img src={h.image} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
          </div>
        ))}
      </div>
    </div>
  )
}

export default HistoryStrip
