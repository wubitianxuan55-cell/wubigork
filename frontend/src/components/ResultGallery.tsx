import React from 'react'
import { Typography, Spin, Empty } from 'antd'
import { ExpandOutlined, DownloadOutlined, SyncOutlined, DeleteOutlined } from '@ant-design/icons'
import { C } from '../utils/theme'

export interface GenResult {
  image: string
  seed: number
  time: number
  prompt: string
  negative?: string
  model: string
  size: string
  style?: string
}

interface Props {
  results: GenResult[]
  generating: boolean
  onPreview: (index: number) => void
  onDownload: (index: number) => void
  onReuse: (index: number) => void
  onDelete?: (index: number) => void
}

const ResultGallery: React.FC<Props> = ({ results, generating, onPreview, onDownload, onReuse, onDelete }) => {
  if (generating && results.length === 0) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: 300, flexDirection: 'column', gap: 16 }}>
        <Spin size="large" />
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 13 }}>AI 正在绘制中...</Typography.Text>
      </div>
    )
  }

  if (results.length === 0) {
    return (
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: 300 }}>
        <Empty description={<span style={{ color: C('color-text-secondary') }}>输入描述，点击生成</span>} />
      </div>
    )
  }

  const getAspect = (size: string) => {
    if (size === '576x1024') return '9 / 16'
    if (size === '1024x576') return '16 / 9'
    return '1 / 1'
  }

  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
      {results.map((r, i) => (
        <div
          key={i}
          style={{
            position: 'relative',
            borderRadius: 'var(--radius-md)',
            overflow: 'hidden',
            border: '1px solid var(--border-subtle)',
            background: 'var(--bg-elevated)',
            cursor: 'pointer',
          }}
          onClick={() => onPreview(i)}
        >
          <img
            src={r.image}
            alt=""
            style={{ width: '100%', display: 'block', aspectRatio: getAspect(r.size), objectFit: 'cover' }}
          />
          <div
            style={{
              position: 'absolute', top: 6, right: 6,
              background: 'rgba(0,0,0,0.6)', borderRadius: 'var(--radius-sm)',
              padding: '1px 6px', fontSize: 10, color: '#fff',
            }}
          >
            {r.time}s
          </div>
          <div
            style={{
              position: 'absolute', bottom: 0, left: 0, right: 0,
              background: 'rgba(0,0,0,0.5)', padding: '4px 8px',
              display: 'flex', gap: 8, justifyContent: 'center', opacity: 0.85,
            }}
          >
            <ExpandOutlined style={{ color: '#fff', fontSize: 12 }} onClick={(e) => { e.stopPropagation(); onPreview(i) }} />
            <DownloadOutlined style={{ color: '#fff', fontSize: 12 }} onClick={(e) => { e.stopPropagation(); onDownload(i) }} />
            <SyncOutlined style={{ color: '#fff', fontSize: 12 }} onClick={(e) => { e.stopPropagation(); onReuse(i) }} />
            {onDelete && (
              <DeleteOutlined style={{ color: '#fff', fontSize: 12 }} onClick={(e) => { e.stopPropagation(); onDelete(i) }} />
            )}
          </div>
        </div>
      ))}
    </div>
  )
}

export default ResultGallery
