import React from 'react'
import { Button, Tag } from 'antd'
import {
  ThunderboltOutlined, CloseCircleOutlined, LoadingOutlined,
  ClockCircleOutlined, DatabaseOutlined,
} from '@ant-design/icons'
import { C } from '../../utils/theme'
import { estimateImageTime } from './ui'
import type { ImageMode } from './types'

interface Props {
  mode: ImageMode
  backend: string
  model: string
  count: number
  frames: number
  fps: number
  generating: boolean
  elapsed: number
  lastTime: number
  pendingCount: number
  queueTotal: number
  comfyProgress?: { status: string; elapsed: number }
  needsComfy: boolean
  onGenerate: () => void
  onCancel: () => void
}

export const GenerationBar: React.FC<Props> = ({
  mode, backend, model, count, frames, fps,
  generating, elapsed, lastTime, pendingCount, queueTotal,
  comfyProgress, needsComfy, onGenerate, onCancel,
}) => {
  const est = estimateImageTime(backend, model, count, mode, frames, fps)

  let hint: string
  if (generating) {
    hint = `已用时 ${elapsed}s`
  } else if (backend === 'comfyui' && comfyProgress?.status === 'running') {
    hint = `ComfyUI 执行中 · ${comfyProgress.elapsed}s`
  } else if (backend === 'comfyui' && comfyProgress?.status) {
    hint = 'ComfyUI 排队中'
  } else if (lastTime > 0) {
    hint = `上次 ${lastTime}s · 预计约 ${est}s`
  } else {
    hint = `预计约 ${est}s`
  }

  const label = generating
    ? pendingCount > 0 ? `生成中 · 队列 ${pendingCount}` : '生成中'
    : mode === 't2v' ? '生成视频' : `生成 ${count} 张`

  return (
    <div className={generating ? 'ig-bottom-bar is-busy' : 'ig-bottom-bar'}>
      <div
        className="ig-bottom-hint"
        aria-live="polite"
        style={{ display: 'flex', alignItems: 'center', gap: 10, minWidth: 0, flexWrap: 'wrap' }}
      >
        <span className="ig-bottom-hint-main" style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
          {generating
            ? <LoadingOutlined style={{ color: 'var(--color-primary)' }} />
            : <ClockCircleOutlined style={{ color: C('color-text-secondary') }} />}
          <span>{hint}</span>
        </span>
        {queueTotal > 0 && (
          <Tag icon={<DatabaseOutlined />} color="processing" style={{ marginInlineEnd: 0, borderRadius: 999, fontSize: 11 }}>
            队列 {queueTotal}
          </Tag>
        )}
        {needsComfy && (
          <span style={{ fontSize: 11, color: 'var(--md-sys-color-warning)' }}>
            {mode === 't2v' ? '文生视频需切换至 ComfyUI' : '图生图需切换至 ComfyUI / Herdsman'}
          </span>
        )}
      </div>
      <div className="ig-bottom-actions" style={{ display: 'flex', alignItems: 'center', gap: 8, flexShrink: 0 }}>
        {(generating || pendingCount > 0) && (
          <Button
            size="middle"
            icon={<CloseCircleOutlined />}
            onClick={onCancel}
            style={{ borderRadius: 999, fontSize: 12 }}
          >
            取消
          </Button>
        )}
        <button
          type="button"
          disabled={needsComfy}
          onClick={onGenerate}
          className="ig-gen-button"
          aria-busy={generating}
          title={needsComfy ? (mode === 't2v' ? '文生视频需切换至 ComfyUI 本地后端' : '图生图需切换至 ComfyUI / Herdsman 本地后端') : undefined}
        >
          {generating ? <LoadingOutlined /> : <ThunderboltOutlined />}
          {label}
        </button>
      </div>
    </div>
  )
}
