import React from 'react'
import { Button, Typography } from 'antd'
import {
  PictureOutlined, ExpandOutlined, DownloadOutlined, SyncOutlined,
  DeleteOutlined, ReloadOutlined, AppstoreOutlined, VideoCameraOutlined,
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
}

const getAspect = (size: string) => {
  const m = /^(\d+)x(\d+)$/.exec(size || '')
  if (m) return `${m[1]} / ${m[2]}`
  return '1 / 1'
}

export const ResultStage: React.FC<Props> = ({
  results, generating, error, mode = 'txt2img', initImage, onPreview, onDownload, onReuse, onDelete, onRetry, onOpenTemplatePicker,
}) => {
  const emptyTitle = mode === 't2v' ? '输入描述，生成你的第一支 AI 视频'
    : mode === 'img2img' ? '上传参考图，开始重绘'
    : '输入描述，开始创作'
  const emptyDesc = mode === 't2v'
    ? '在左侧写下画面描述，选择分辨率与时长后点击生成（需 ComfyUI + LTX-Video）'
    : mode === 'img2img'
      ? '上传参考图并描述想要的改动，控制重绘幅度生成新画面'
      : '在左侧写下想要的画面，选择模型与参数后点击生成'

  // 生成中且无结果：骨架屏
  if (generating && results.length === 0) {
    return (
      <div className="img-stage-enter" style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: 12 }}>
          {[0, 1, 2, 3].map((i) => (
            <div key={i} style={{
              aspectRatio: '1 / 1', borderRadius: 'var(--radius-md)',
              background: 'linear-gradient(110deg, var(--bg-elevated) 30%, rgba(255,255,255,0.06) 50%, var(--bg-elevated) 70%)',
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
      <div className="img-stage-enter" style={{
        display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center',
        gap: 12, paddingTop: mode === 'img2img' && initImage ? 40 : 110,
      }}>
        {mode === 'img2img' && initImage && (
          <div style={{ position: 'relative' }}>
            <img
              src={initImage}
              alt="参考图"
              style={{ width: 160, height: 160, objectFit: 'cover', borderRadius: 14, border: '1px solid var(--border-subtle)', boxShadow: 'var(--shadow-sm)' }}
            />
            <span style={{
              position: 'absolute', left: '50%', bottom: -10, transform: 'translateX(-50%)',
              background: 'rgba(0,0,0,0.65)', color: '#fff', fontSize: 10, padding: '2px 10px', borderRadius: 999, // hex-exempt 图片覆盖层 chrome
              border: '1px solid var(--border-subtle)', whiteSpace: 'nowrap',
            }}>参考图已就绪</span>
          </div>
        )}
        <div style={{
          width: 56, height: 56, borderRadius: 18, display: 'flex', alignItems: 'center', justifyContent: 'center',
          background: 'rgba(var(--accent-rgb), 0.12)', color: 'var(--color-primary)', fontSize: 24,
          border: '1px solid rgba(var(--accent-rgb), 0.2)',
        }}>
          {mode === 't2v' ? <VideoCameraOutlined /> : <PictureOutlined />}
        </div>
        <div style={{ textAlign: 'center' }}>
          <div style={{ color: 'var(--color-text)', fontSize: 20, fontWeight: 650, marginBottom: 8 }}>{emptyTitle}</div>
          <div style={{ color: C('color-text-secondary'), fontSize: 12, maxWidth: 300, lineHeight: 1.6 }}>{emptyDesc}</div>
        </div>
        {onOpenTemplatePicker && (
          <Button type="primary" icon={<AppstoreOutlined />} onClick={onOpenTemplatePicker}
            style={{ borderRadius: 999, fontSize: 13, height: 36, paddingInline: 18 }}>浏览模板</Button>
        )}
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
              alt=""
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
              alt=""
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
