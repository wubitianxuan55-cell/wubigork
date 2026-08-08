import React from 'react'
import { Typography, Tag } from 'antd'
import { EditOutlined, UserOutlined } from '@ant-design/icons'
import { C, ROLE_COLORS as roleColors, ROLE_LABELS as roleLabels } from '../../../utils/theme'
import type { CharacterData } from '../../../types'

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
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onClick() } }}
      className="char-card"
      style={{ ['--char-side' as any]: sideColor }}
    >
      {/* 剧照区域 */}
      <div className="char-card-portrait">
        {ch.portrait_url ? (
          <>
            <img src={ch.portrait_url} alt={ch.name} />
            <div
              className="char-card-portrait-overlay"
              onClick={(e) => { e.stopPropagation(); onPortraitFullscreen(ch.portrait_url!) }}
              title="点击放大"
            />
          </>
        ) : (
          <div className="char-card-portrait-placeholder"><UserOutlined /></div>
        )}
      </div>

      {/* 信息区域 */}
      <div className="char-card-body">
        {/* 名称 */}
        <Typography.Text strong className="char-card-name" style={{ color: C('color-text') }}>
          {ch.name}
        </Typography.Text>

        {/* 标签行 */}
        <div className="char-card-tags">
          <Tag color={roleColor}>
            {roleLabels[ch.role_type] || ch.role_type}
          </Tag>
          {statusInfo && (
            <Tag color={statusInfo.color}>
              {statusInfo.label}
            </Tag>
          )}
        </div>

        {/* 性别 · 年龄 */}
        <Typography.Text className="char-card-meta" style={{ color: C('color-text-secondary') }}>
          {(ch.gender === 'male' ? '♂ 男' : ch.gender === 'female' ? '♀ 女' : ch.gender || '?')}
          {ch.age ? ` · ${ch.age}岁` : ''}
        </Typography.Text>

        {/* 分隔线 */}
        <div className="char-card-divider" />

        {/* 性格预览 */}
        <Typography.Text className="char-card-preview" style={{ color: C('color-text-secondary') }}>
          {preview || '暂无描述'}
        </Typography.Text>

        {/* 底部：关系数 + 编辑提示 */}
        <div className="char-card-footer">
          {relationCount > 0
            ? <span>🔗 {relationCount} 个关系</span>
            : <span>暂无关系</span>}
          <EditOutlined />
        </div>
      </div>
    </div>
  )
}

export default CharacterCard
