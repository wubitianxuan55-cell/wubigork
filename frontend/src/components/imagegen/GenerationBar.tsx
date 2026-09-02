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
  comfyProgress?: { status: string; elapsed: number; percent?: number; node?: string }
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

  // 百炼改图（dashscope）支持图生图：页面级 needsComfy 门禁未感知该云端引擎时，
  // 这里按 mode+backend 复核放行，避免 dashscope+img2img 被误判为需 ComfyUI。
  const modeBlocked = needsComfy && !(mode === 'img2img' && backend === 'dashscope')

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

  // ComfyUI 实时进度（percent >= 0 时显示确定进度条；其余后端/未知显示不定态光带）
  const hasRealProgress = generating && backend === 'comfyui' && (comfyProgress?.percent ?? -1) >= 0
  const percent = hasRealProgress ? Math.min(100, Math.max(0, comfyProgress!.percent!)) : -1
  const nodeLabel = comfyProgress?.node ? (COMFY_NODE_LABELS[comfyProgress.node] || comfyProgress.node) : ''

  return (
    <div className={generating ? 'ig-bottom-bar is-busy' : 'ig-bottom-bar'}>
      {generating && (
        <div className="ig-progress" aria-live="polite">
          <div className="ig-progress-track">
            <div
              className={hasRealProgress ? 'ig-progress-fill' : 'ig-progress-fill is-indeterminate'}
              style={hasRealProgress ? { width: `${percent}%` } : undefined}
            />
          </div>
          <div className="ig-progress-meta">
            {hasRealProgress
              ? <span>全部: {percent}%</span>
              : <span>生成中…</span>}
            {nodeLabel && <span>当前节点: {nodeLabel}</span>}
          </div>
        </div>
      )}
      <div
        className="ig-bottom-row"
        style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 12, minWidth: 0, width: '100%' }}
      >
        <div
          className="ig-bottom-hint"
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
          {modeBlocked && (
            <span style={{ fontSize: 11, color: 'var(--md-sys-color-warning)' }}>
              {mode === 't2v' ? '文生视频需切换至 ComfyUI' : '图生图需切换至 ComfyUI / Herdsman / 百炼改图'}
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
            disabled={modeBlocked}
            onClick={onGenerate}
            className="ig-gen-button"
            aria-busy={generating}
            title={modeBlocked ? (mode === 't2v' ? '文生视频需切换至 ComfyUI 本地后端' : '图生图需切换至 ComfyUI / Herdsman / 百炼改图 后端') : undefined}
          >
            {generating ? <LoadingOutlined /> : <ThunderboltOutlined />}
            {label}
          </button>
        </div>
      </div>
    </div>
  )
}

/** ComfyUI 节点 class_type → 中文阶段名（未知节点回退原始名） */
const COMFY_NODE_LABELS: Record<string, string> = {
  CheckpointLoaderSimple: '加载模型',
  CheckpointLoader: '加载模型',
  UNETLoader: '加载模型',
  UnetLoaderGGUF: '加载模型',
  CLIPLoader: '加载模型',
  CLIPLoaderGGUF: '加载模型',
  DualCLIPLoader: '加载模型',
  LoraLoader: '加载 LoRA',
  LoraLoaderModelOnly: '加载 LoRA',
  EmptyLatentImage: '初始化画布',
  LatentFromImage: '初始化画布',
  KSampler: '采样中',
  KSamplerAdvanced: '采样中',
  SamplerCustom: '采样中',
  VAEDecode: '解码中',
  VAEDecodeTiled: '解码中',
  VAEEncode: '编码中',
  LoadImage: '读取参考图',
  SaveImage: '保存图片',
  SaveAnimatedWEBP: '保存动画',
  LTXVideo: '生成视频',
  LTXV: '生成视频',
  LTXVConditioning: '视频条件',
  ZImagePowerNodes: '图像处理',
}
