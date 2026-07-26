import React from 'react'
import { Button } from 'antd'
import { ThunderboltOutlined, LoadingOutlined } from '@ant-design/icons'
import { C } from '../utils/theme'

interface Props {
  generating: boolean
  count: number
  lastTime: number
  elapsed: number
  backend: string
  model: string
  onGenerate: () => void
}

const estimateTime = (backend: string, model: string, count: number) => {
  if (backend === 'xai') return count * 5
  if (model === 'z-image-turbo') return count * 20
  return count * 60
}

const GenButton: React.FC<Props> = ({ generating, count, lastTime, elapsed, backend, model, onGenerate }) => {
  const est = estimateTime(backend, model, count)

  return (
    <div style={{ marginTop: 12 }}>
      <Button
        type="primary"
        block
        size="large"
        icon={generating ? <LoadingOutlined /> : <ThunderboltOutlined />}
        onClick={onGenerate}
        loading={generating}
        style={{
          boxShadow: 'var(--shadow-glow)',
          borderRadius: 'var(--radius-md)',
          height: 44,
        }}
      >
        {generating ? '生成中...' : `生成 ${count} 张`}
      </Button>
      <div style={{ textAlign: 'center', marginTop: 4 }}>
        <span style={{ color: C('color-text-secondary'), fontSize: 10 }}>
          {generating
            ? `⏱️ ${elapsed}s`
            : lastTime > 0
              ? `上次 ${lastTime}s · 预计 ${est}s`
              : `预计 ~${est}s`}
        </span>
      </div>
    </div>
  )
}

export default GenButton
