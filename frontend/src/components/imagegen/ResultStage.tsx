import React from 'react'
import { Button, Typography } from 'antd'
import {
  PictureOutlined, ExpandOutlined, DownloadOutlined, SyncOutlined,
  DeleteOutlined, ReloadOutlined, AppstoreOutlined,
} from '@ant-design/icons'
import { C } from '../../utils/theme'
import type { GenResult } from '../ResultGallery'

interface Props {
  results: GenResult[]
  generating: boolean
  error?: string
  onPreview: (index: number) => void
  onDownload: (index: number) => void
  onReuse: (index: number) => void
  onDelete?: (index: number) => void
  onRetry?: () => void
  onOpenTemplatePicker?: () => void
}

const getAspect = (size: string) => {
  if (size === '576x1024') return '9 / 16'
  if (size === '1024x576') return '16 / 9'
  if (size === '1280x544') return '21 / 9'
  if (size === '1024x768' || size === '768x1024') return '4 / 3'
  return '1 / 1'
}

export const ResultStage: React.FC<Props> = ({
  results, generating, error, onPreview, onDownload, onReuse, onDelete, onRetry, onOpenTemplatePicker,
}) => {
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
          border: '1px solid rgba(248,113,113,0.35)',
          background: 'rgba(248,113,113,0.08)', padding: '14px 16px',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
            <span style={{ color: '#f87171', fontSize: 14 }}><ReloadOutlined /></span>
            <span style={{ color: '#f87171', fontSize: 13, fontWeight: 600 }}>生成失败</span>
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
        gap: 12, paddingTop: 110,
      }}>
        <div style={{
          width: 56, height: 56, borderRadius: 18, display: 'flex', alignItems: 'center', justifyContent: 'center',
          background: 'rgba(var(--accent-rgb), 0.12)', color: 'var(--color-primary)', fontSize: 24,
          border: '1px solid rgba(var(--accent-rgb), 0.2)',
        }}>
          <PictureOutlined />
        </div>
        <div style={{ textAlign: 'center' }}>
          <div style={{ color: 'var(--color-text)', fontSize: 15, fontWeight: 600, marginBottom: 4 }}>
            输入描述，开始创作
          </div>
          <div style={{ color: C('color-text-secondary'), fontSize: 12, maxWidth: 280, lineHeight: 1.6 }}>
            在左侧写下想要的画面，选择模型与参数后点击生成
          </div>
        </div>
        {onOpenTemplatePicker && (
          <Button icon={<AppstoreOutlined />} onClick={onOpenTemplatePicker}
            style={{ borderRadius: 999, fontSize: 12 }}>浏览模板</Button>
        )}
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
          <img
            src={r.image}
            alt=""
            style={{ width: '100%', display: 'block', aspectRatio: getAspect(r.size), objectFit: 'cover' }}
          />
          <div style={{
            position: 'absolute', top: 6, right: 6, background: 'rgba(0,0,0,0.55)',
            borderRadius: 999, padding: '2px 8px', fontSize: 10, color: '#fff', backdropFilter: 'blur(4px)',
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
      color: '#fff', fontSize: 11, borderRadius: 999, padding: '4px 8px', cursor: 'pointer',
      fontFamily: 'inherit', pointerEvents: 'auto', backdropFilter: 'blur(4px)',
    }}
  >
    {icon}
    {label}
  </button>
)
