import React from 'react'
import { Progress } from 'antd'
import { MenuOutlined } from '@ant-design/icons'
import { Z_INDEX } from '../utils/zIndex'

const appBarStyle: React.CSSProperties = {
  position: 'sticky',
  top: 0,
  zIndex: Z_INDEX.APP_BAR,
  display: 'flex',
  flexDirection: 'column',
  background: 'var(--md-sys-color-surface)',
  borderBottom: '1px solid var(--md-sys-color-outline-variant)',
}

const titleRowStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  height: 48,
  padding: '0 16px',
}

const statusRowStyle: React.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  height: 22,
  padding: '0 16px 4px',
  fontSize: 11,
  color: 'var(--md-sys-color-text-secondary)',
  gap: 8,
}

interface Props {
  title: string
  onMenuClick: () => void
  projectOpen?: boolean
  projectTitle?: string
  chapterCount?: number
  totalWords?: number
  progressPercent?: number
}

const AppBar: React.FC<Props> = ({
  title, onMenuClick,
  projectOpen, projectTitle,
  chapterCount, totalWords, progressPercent,
}) => (
  <header style={{
    ...appBarStyle,
    height: projectOpen ? 'auto' : 48,
  }}>
    <div style={titleRowStyle}>
      <button
        onClick={onMenuClick}
        style={{
          background: 'none', border: 'none', cursor: 'pointer',
          marginRight: 12, fontSize: 20,
          color: 'var(--md-sys-color-on-surface-variant)',
          minWidth: 44, minHeight: 44, padding: 0,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }}
      >
        <MenuOutlined />
      </button>
      {/* 品牌 favicon */}
      <img
        src="/favicon.svg"
        alt="gaea"
        style={{ width: 20, height: 20, marginRight: 8 }}
      />
      <span style={{
        fontSize: 18, fontWeight: 600,
        color: 'var(--md-sys-color-on-surface)',
      }}>
        {title}
      </span>
    </div>

    {/* 项目状态行 — 仅在有打开项目时显示 */}
    {projectOpen && (
      <div style={statusRowStyle}>
        {projectTitle && (
          <span style={{ fontWeight: 500, color: 'var(--md-sys-color-text)', maxWidth: 120, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {projectTitle}
          </span>
        )}
        {progressPercent !== undefined && (
          <>
            <span style={{ opacity: 0.5 }}>·</span>
            <Progress
              percent={progressPercent}
              size="small"
              showInfo={false}
              style={{ width: 60, margin: 0 }}
              strokeColor="var(--md-sys-color-primary)"
              trailColor="var(--md-sys-color-outline-variant)"
            />
            <span>{progressPercent}%</span>
          </>
        )}
        {chapterCount !== undefined && (
          <>
            <span style={{ opacity: 0.5 }}>·</span>
            <span>{chapterCount} 章</span>
          </>
        )}
        {totalWords !== undefined && (
          <>
            <span style={{ opacity: 0.5 }}>·</span>
            <span>{totalWords >= 10000 ? (totalWords / 10000).toFixed(1) + '万' : totalWords.toLocaleString()} 字</span>
          </>
        )}
      </div>
    )}
  </header>
)

export default AppBar
