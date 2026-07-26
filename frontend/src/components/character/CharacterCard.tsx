import React from 'react'
import { Typography, Tag, Space } from 'antd'
import { UserOutlined } from '@ant-design/icons'
import { C, ROLE_COLORS as roleColors, ROLE_LABELS as roleLabels } from '../../utils/theme'
import type { CharacterData } from '../../types'

export interface CharacterCardProps {
  character: CharacterData
  relationCount: number
  onClick: () => void
  onPortraitFullscreen: (url: string) => void
}

/** 角色类型 → 卡牌左侧边条颜色 */
const SIDE_COLORS: Record<string, string> = {
  protagonist: '#f59e0b',
  antagonist: '#ef4444',
  supporting: '#3b82f6',
  minor: '#6b7280',
}

/** 状态 → 中文标签 + 颜色 */
const STATUS_MAP: Record<string, { label: string; color: string }> = {
  Alive: { label: '存活', color: '#22c55e' },
  Dead: { label: '已故', color: '#ef4444' },
  Missing: { label: '失踪', color: '#f59e0b' },
  Transformed: { label: '变身', color: '#a855f7' },
}

const CharacterCard: React.FC<CharacterCardProps> = ({ character, relationCount, onClick, onPortraitFullscreen }) => {
  const ch = character
  const sideColor = SIDE_COLORS[ch.role_type] || '#6b7280'
  const statusInfo = STATUS_MAP[ch.status]
  const preview = ch.personality || ch.motivation || ''
  const roleColor = roleColors[ch.role_type] || 'var(--color-text-secondary)'

  return (
    <div
      onClick={onClick}
      style={{
        background: 'var(--bg-glass)',
        backdropFilter: 'blur(8px)', WebkitBackdropFilter: 'blur(8px)',
        border: '1px solid var(--border-subtle)',
        borderRadius: 'var(--radius-lg)',
        cursor: 'pointer',
        transition: 'box-shadow 0.25s, transform 0.25s',
        height: '100%',
        overflow: 'hidden',
        display: 'flex',
        flexDirection: 'column',
        borderLeft: `3px solid ${sideColor}`,
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.transform = 'translateY(-2px)'
        e.currentTarget.style.boxShadow = '0 8px 24px rgba(0,0,0,0.25)'
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.transform = 'translateY(0px)'
        e.currentTarget.style.boxShadow = 'none'
      }}
    >
      {/* 剧照区域 */}
      <div
        style={{
          height: 220,
          background: !ch.portrait_url
            ? `linear-gradient(135deg, ${sideColor}44, ${sideColor}22)`
            : 'none',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          position: 'relative', flexShrink: 0, overflow: 'hidden',
        }}
      >
        {ch.portrait_url ? (
          <>
            <img src={ch.portrait_url} alt={ch.name}
              style={{ width: '100%', height: '100%', objectFit: 'cover', objectPosition: 'top center', display: 'block' }} />
            <div
              onClick={(e) => { e.stopPropagation(); onPortraitFullscreen(ch.portrait_url!) }}
              style={{
                position: 'absolute', inset: 0,
                cursor: 'zoom-in',
                background: 'rgba(0,0,0,0)',
                transition: 'background 0.2s',
              }}
              title="点击放大"
              onMouseEnter={(e) => { (e.currentTarget as HTMLElement).style.background = 'rgba(0,0,0,0.15)' }}
              onMouseLeave={(e) => { (e.currentTarget as HTMLElement).style.background = 'rgba(0,0,0,0)' }}
            />
          </>
        ) : (
          <UserOutlined style={{ fontSize: 56, color: sideColor, opacity: 0.4 }} />
        )}
      </div>

      {/* 信息区域 */}
      <div style={{ padding: '8px 10px', flex: 1, display: 'flex', flexDirection: 'column', gap: 4 }}>
        {/* 名称 */}
        <Typography.Text strong style={{ color: C('color-text'), fontSize: 14, lineHeight: 1.3 }}>
          {ch.name}
        </Typography.Text>

        {/* 标签行 */}
        <Space size={4} wrap>
          <Tag color={roleColor} style={{ fontSize: 9, lineHeight: '16px', padding: '0 5px', margin: 0, borderRadius: 3 }}>
            {roleLabels[ch.role_type] || ch.role_type}
          </Tag>
          {statusInfo && (
            <Tag color={statusInfo.color} style={{ fontSize: 9, lineHeight: '16px', padding: '0 5px', margin: 0, borderRadius: 3 }}>
              {statusInfo.label}
            </Tag>
          )}
        </Space>

        {/* 性别 · 年龄 */}
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10 }}>
          {(ch.gender === 'male' ? '♂ 男' : ch.gender === 'female' ? '♀ 女' : ch.gender || '?')}
          {ch.age ? ` · ${ch.age}岁` : ''}
        </Typography.Text>

        {/* 分隔线 */}
        <div style={{ height: 1, background: 'var(--border-subtle)', margin: '2px 0' }} />

        {/* 性格预览 */}
        <Typography.Text style={{
          color: C('color-text-secondary'), fontSize: 10, lineHeight: 1.5,
          overflow: 'hidden', textOverflow: 'ellipsis',
          display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical',
          flex: 1,
        }}>
          {preview || '暂无描述'}
        </Typography.Text>

        {/* 关系数 */}
        {relationCount > 0 && (
          <Typography.Text style={{ color: '#60a5fa', fontSize: 9 }}>
            🔗 {relationCount} 个关系
          </Typography.Text>
        )}
      </div>
    </div>
  )
}

export default CharacterCard
