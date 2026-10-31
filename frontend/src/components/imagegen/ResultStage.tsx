import React from 'react'
import { Button, Typography } from 'antd'
import {
  PictureOutlined, ExpandOutlined, DownloadOutlined, SyncOutlined,
  DeleteOutlined, ReloadOutlined, AppstoreOutlined, VideoCameraOutlined,
  EditOutlined,
  HighlightOutlined,
  ScissorOutlined,
} from '@ant-design/icons'
import { C } from '../../utils/theme'
import { mediaIsVideo } from './media'
import type { GenResult } from './types'

interface Props {
  results: GenResult[]
  generating: boolean
  error?: string
  mode?: 'txt2img' | 'img2img' | 't2v'
  initImage?: string
  onPreview: (index: number) => void
  onDownload: (index: number) => void
  onReuse: (index: number) => void
  onDelete?: (index: number) => void
  onRetry?: () => void
  onOpenTemplatePicker?: () => void
  /** 把该张结果作为参考图发起图生图（不传则不渲染「改图」动作） */
  onEditImage?: (index: number) => void
  /** 指令编辑（刀 C）：原图+人话指令语义改图（云端 /images/edits）。 */
  onInstructEdit?: (index: number) => void
  /** 抠图（刀 E，T3 收官）：涂选保留主体 → 透明底 PNG 导出。 */
  onCutout?: (index: number) => void
}

const getAspect = (size: string) => {
  const m = /^(\d+)x(\d+)$/.exec(size || '')
  if (m) return `${m[1]} / ${m[2]}`
  return '1 / 1'
}

export const ResultStage: React.FC<Props> = ({
  results, generating, error, mode = 'txt2img', initImage, onPreview, onDownload, onReuse, onDelete, onRetry, onOpenTemplatePicker, onEditImage, onInstructEdit, onCutout,
}) => {
  const emptyTitle = mode === 't2v' ? '一句话，生成第一段镜头。'
    : mode === 'img2img' ? '让参考图，长出新的画面。'
    : '让第一幅画，从一句话开始。'
  const emptyDesc = mode === 't2v'
    ? '写下主体、动作与镜头运动；本地 ComfyUI + LTX-Video 会把描述变成动态画面。'
    : mode === 'img2img'
      ? '先放入参考图，再描述想保留和改变的部分。重绘幅度决定它离原图有多远。'
      : '在左侧写下你脑海里的画面，再选定模型与画幅。绘梦会把描述变成一张可继续编辑的作品。'
  const emptyCues = mode === 't2v'
    ? ['主体动作', '镜头运动', '环境光线']
    : mode === 'img2img'
      ? ['保留构图', '改变内容', '目标风格']
      : ['主体细节', '环境光线', '镜头风格']

  // 生成中且无结果：骨架屏
  if (generating && results.length === 0) {
    return (
      <div className="img-stage-enter" style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: 12 }}>
          {[0, 1, 2, 3].map((i) => (
            <div key={i} style={{
              aspectRatio: '1 / 1', borderRadius: 'var(--radius-md)',
              background: 'linear-gradient(110deg, var(--bg-elevated) 30%, color-mix(in srgb, var(--color-text) 6%, transparent) 50%, var(--bg-elevated) 70%)',
              backgroundSize: '200% 100%',
              animation: 'imgStageShimmer 1.4s linear infinite',
            }} />
          ))}
        </div>
        <div style={{ textAlign: 'center', marginTop: 4 }}>
          <span style={{ color: C('color-text-secondary'), fontSize: 12 }}>AI 正在绘制中，请稍候...</span>
        </div>
      </div>
    )
  }

  // 错误横幅
  if (error && !generating) {
    return (
      <div className="img-stage-enter" style={{
        display: 'flex', flexDirection: 'column', gap: 12, alignItems: 'center', paddingTop: 80,
      }}>
        <div style={{
          maxWidth: 460, width: '100%', borderRadius: 'var(--radius-md)',
          border: '1px solid color-mix(in srgb, var(--color-destructive) 35%, transparent)',
          background: 'color-mix(in srgb, var(--color-destructive) 8%, transparent)', padding: '14px 16px',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
            <span style={{ color: 'var(--color-destructive)', fontSize: 14 }}><ReloadOutlined /></span>
            <span style={{ color: 'var(--color-destructive)', fontSize: 13, fontWeight: 600 }}>生成失败</span>
          </div>
          <Typography.Paragraph style={{ color: C('color-text-secondary'), fontSize: 12, margin: '0 0 10px' }}>
            {error}
          </Typography.Paragraph>
          {onRetry && (
            <Button size="small" icon={<ReloadOutlined />} onClick={onRetry}
              style={{ borderRadius: 999, fontSize: 12 }}>重试</Button>
          )}
        </div>
      </div>
    )
  }

  // 空态
  if (results.length === 0) {
    return (
      <div className="img-stage-enter ig-empty-stage">
        <span className="ig-empty-index" aria-hidden>01 / Canvas</span>
        <div className="ig-empty-art" aria-hidden={!initImage}>
          {mode === 'img2img' && initImage ? (
            <>
              <img src={initImage} alt="图生图参考图" />
              <span className="ig-empty-ref-label">参考图已就位</span>
            </>
          ) : (
            <>
              <span className="ig-empty-frame ig-empty-frame--back" />
              <span className="ig-empty-frame ig-empty-frame--front">
                {mode === 't2v' ? <VideoCameraOutlined /> : <PictureOutlined />}
              </span>
              <span className="ig-empty-orbit" />
            </>
          )}
        </div>
        <div className="ig-empty-copy">
          <span className="ig-empty-eyebrow">
            {mode === 't2v' ? 'Motion study' : mode === 'img2img' ? 'Reference study' : 'Blank canvas'}
          </span>
          <h2>{emptyTitle}</h2>
          <p>{emptyDesc}</p>
          <div className="ig-empty-cues" aria-label="描述建议">
            {emptyCues.map((cue, index) => (
              <span key={cue}>
                <b>{String(index + 1).padStart(2, '0')}</b>
                {cue}
              </span>
            ))}
          </div>
          {onOpenTemplatePicker && (
            <Button
              type="primary"
              icon={<AppstoreOutlined />}
              onClick={onOpenTemplatePicker}
              className="ig-empty-template"
            >
              从模板开始
            </Button>
          )}
        </div>
      </div>
    )
  }

  // 单张结果：铺满中央画布（object-fit contain 完整展示，不裁剪）
  if (results.length === 1) {
    const r = results[0]
    return (
      <div
        className="img-stage-enter"
        style={{ flex: 1, minHeight: 0, position: 'relative', display: 'flex' }}
      >
        <div
          className="img-card"
          onClick={() => onPreview(0)}
          style={{
            position: 'relative', flex: 1, minHeight: 0, borderRadius: 'var(--radius-md)',
            overflow: 'hidden', border: '1px solid var(--md-sys-color-outline-variant)',
            background: '#000', cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', // hex-exempt 图片覆盖层 chrome
          }}
        >
          {mediaIsVideo(r.image) ? (
            <video
              src={r.image}
              controls
              autoPlay
              loop
              muted
              playsInline
              style={{ width: '100%', height: '100%', display: 'block', objectFit: 'contain' }}
            />
          ) : (
            <img
              src={r.image}
              alt={r.prompt ? `绘梦生成：${r.prompt}` : '绘梦生成图片'}
              style={{ width: '100%', height: '100%', display: 'block', objectFit: 'contain' }}
            />
          )}
          {r.kind === 'video' && (
            <span style={{
              position: 'absolute', top: 10, left: 10, display: 'inline-flex', alignItems: 'center', gap: 4,
              background: 'rgba(0,0,0,0.55)', borderRadius: 999, padding: '3px 10px', fontSize: 11, color: '#fff', // hex-exempt 图片覆盖层 chrome
              backdropFilter: 'blur(4px)',
            }}>
              <VideoCameraOutlined style={{ fontSize: 11 }} /> 视频
            </span>
          )}
          <div style={{
            position: 'absolute', top: 10, right: 10, background: 'rgba(0,0,0,0.55)',
            borderRadius: 999, padding: '3px 10px', fontSize: 11, color: '#fff', backdropFilter: 'blur(4px)', // hex-exempt 图片覆盖层 chrome
          }}>
            {r.time}s
          </div>
          <div style={{
            position: 'absolute', bottom: 12, left: 0, right: 0,
            display: 'flex', gap: 6, justifyContent: 'center',
            opacity: 0, transition: 'opacity 0.2s ease', pointerEvents: 'none',
          }} className="img-card-actions">
            <Action icon={<ExpandOutlined />} label="预览" onClick={(e) => { e.stopPropagation(); onPreview(0) }} />
            <Action icon={<DownloadOutlined />} label="下载" onClick={(e) => { e.stopPropagation(); onDownload(0) }} />
            <Action icon={<SyncOutlined />} label="复用" onClick={(e) => { e.stopPropagation(); onReuse(0) }} />
            {onEditImage && !mediaIsVideo(r.image) && (
              <Action icon={<EditOutlined />} label="改图" onClick={(e) => { e.stopPropagation(); onEditImage(0) }} />
            )}
            {onInstructEdit && !mediaIsVideo(r.image) && (
              <Action icon={<HighlightOutlined />} label="指令编辑" onClick={(e) => { e.stopPropagation(); onInstructEdit(0) }} />
            )}
            {onCutout && !mediaIsVideo(r.image) && (
              <Action icon={<ScissorOutlined />} label="抠图" onClick={(e) => { e.stopPropagation(); onCutout(0) }} />
            )}
            {onDelete && (
              <Action icon={<DeleteOutlined />} label="删除" onClick={(e) => { e.stopPropagation(); onDelete(0) }} />
            )}
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="img-stage-enter" style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: 12 }}>
      {results.map((r, i) => (
        <div
          key={i}
          className="img-card"
          style={{
            position: 'relative', borderRadius: 'var(--radius-md)', overflow: 'hidden',
            border: '1px solid var(--md-sys-color-outline-variant)',
            background: 'var(--bg-elevated)', cursor: 'pointer',
          }}
          onClick={() => onPreview(i)}
        >
          {/* T6-4.2：按实际媒体类型选择元素（t2v 输出动画 webp 用 <img> 播，video/* 才用 <video>） */}
          {mediaIsVideo(r.image) ? (
            <video
              src={r.image}
              controls
              autoPlay
              loop
              muted
              playsInline
              style={{ width: '100%', display: 'block', aspectRatio: getAspect(r.size), objectFit: 'cover', background: '#000' }} // hex-exempt 图片覆盖层 chrome
            />
          ) : (
            <img
              src={r.image}
              alt={r.prompt ? `绘梦生成：${r.prompt}` : '绘梦生成图片'}
              style={{ width: '100%', display: 'block', aspectRatio: getAspect(r.size), objectFit: 'cover' }}
            />
          )}
          {r.kind === 'video' && (
            <span style={{
              position: 'absolute', top: 6, left: 6, display: 'inline-flex', alignItems: 'center', gap: 4,
              background: 'rgba(0,0,0,0.55)', borderRadius: 999, padding: '2px 8px', fontSize: 10, color: '#fff', // hex-exempt 图片覆盖层 chrome
              backdropFilter: 'blur(4px)',
            }}>
              <VideoCameraOutlined style={{ fontSize: 10 }} /> 视频
            </span>
          )}
          <div style={{
            position: 'absolute', top: 6, right: 6, background: 'rgba(0,0,0,0.55)',
            borderRadius: 999, padding: '2px 8px', fontSize: 10, color: '#fff', backdropFilter: 'blur(4px)', // hex-exempt 图片覆盖层 chrome
          }}>
            {r.time}s
          </div>
          <div style={{
            position: 'absolute', bottom: 0, left: 0, right: 0,
            background: 'linear-gradient(transparent, rgba(0,0,0,0.72))',
            padding: '14px 8px 6px', display: 'flex', gap: 4, justifyContent: 'center',
            opacity: 0, transition: 'opacity 0.2s ease', pointerEvents: 'none',
          }} className="img-card-actions">
            <Action icon={<ExpandOutlined />} label="预览" onClick={(e) => { e.stopPropagation(); onPreview(i) }} />
            <Action icon={<DownloadOutlined />} label="下载" onClick={(e) => { e.stopPropagation(); onDownload(i) }} />
            <Action icon={<SyncOutlined />} label="复用" onClick={(e) => { e.stopPropagation(); onReuse(i) }} />
            {onEditImage && !mediaIsVideo(r.image) && (
              <Action icon={<EditOutlined />} label="改图" onClick={(e) => { e.stopPropagation(); onEditImage(i) }} />
            )}
            {onInstructEdit && !mediaIsVideo(r.image) && (
              <Action icon={<HighlightOutlined />} label="指令编辑" onClick={(e) => { e.stopPropagation(); onInstructEdit(i) }} />
            )}
            {onCutout && !mediaIsVideo(r.image) && (
              <Action icon={<ScissorOutlined />} label="抠图" onClick={(e) => { e.stopPropagation(); onCutout(i) }} />
            )}
            {onDelete && (
              <Action icon={<DeleteOutlined />} label="删除" onClick={(e) => { e.stopPropagation(); onDelete(i) }} />
            )}
          </div>
        </div>
      ))}
    </div>
  )
}

const Action: React.FC<{ icon: React.ReactNode; label: string; onClick: (e: React.MouseEvent) => void }> = ({
  icon, label, onClick,
}) => (
  <button
    type="button"
    onClick={onClick}
    title={label}
    className="img-picker-btn"
    style={{
      display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 4,
      border: '1px solid rgba(255,255,255,0.18)', background: 'rgba(0,0,0,0.42)',
      color: '#fff', fontSize: 11, borderRadius: 999, padding: '4px 8px', cursor: 'pointer', // hex-exempt 图片覆盖层 chrome
      fontFamily: 'inherit', pointerEvents: 'auto', backdropFilter: 'blur(4px)',
    }}
  >
    {icon}
    {label}
  </button>
)
