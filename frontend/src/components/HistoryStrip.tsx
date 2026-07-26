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
  if (history.length === 0) return null

  return (
    <div style={{ marginTop: 16 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8 }}>
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>
          📜 本次会话历史 ({history.length})
        </Typography.Text>
        <Button
          type="text" size="small" icon={<DeleteOutlined />} onClick={onClear}
          style={{ color: C('color-text-secondary'), fontSize: 11 }}
        >
          清空
        </Button>
      </div>
      <div style={{ display: 'flex', gap: 8, overflowX: 'auto', paddingBottom: 4 }}>
        {history.map((h, i) => (
          <div
            key={i} onClick={() => onSelect(i)}
            style={{
              width: 72, height: 72, flexShrink: 0, cursor: 'pointer',
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
