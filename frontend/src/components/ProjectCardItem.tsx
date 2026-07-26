import React from 'react'
import { Typography, Button, Space, Tag, Popconfirm } from 'antd'
import {
  BookOutlined, FileTextOutlined, ClockCircleOutlined, DeleteOutlined,
} from '@ant-design/icons'
import type { ProjectCard } from '../stores/appStore'
import { C } from '../utils/theme'
import { formatRelativeTime } from '../utils/time'

interface ProjectCardItemProps {
  card: ProjectCard
  isActive: boolean
  isHero: boolean
  isMobile?: boolean
  onOpen: (card: ProjectCard) => void
  onDelete: (card: ProjectCard) => void
}

/** ProjectCardItem — Bento grid 项目卡片 */
const ProjectCardItem: React.FC<ProjectCardItemProps> = ({
  card, isActive, isHero, onOpen, onDelete,
}) => (
  <div
    key={card.path}
    className="glass-card"
    onClick={() => onOpen(card)}
    style={{
      gridColumn: isHero ? 'span 2' : 'span 1',
      gridRow: isHero ? 'span 2' : 'span 1',
      cursor: 'pointer',
      padding: '20px 24px',
      display: 'flex', flexDirection: 'column', justifyContent: 'space-between',
      borderRadius: 'var(--radius-xl)',
      border: isActive ? '1px solid var(--color-primary)' : '1px solid var(--border-subtle)',
      boxShadow: isActive ? 'var(--shadow-glow)' : 'var(--shadow-md)',
      transition: 'all var(--transition-normal)',
      position: 'relative',
      overflow: 'hidden',
    }}
    onMouseEnter={(e) => {
      e.currentTarget.style.transform = 'translateY(-2px)'
      e.currentTarget.style.boxShadow = 'var(--shadow-lg)'
    }}
    onMouseLeave={(e) => {
      e.currentTarget.style.transform = 'translateY(0)'
      e.currentTarget.style.boxShadow = isActive ? 'var(--shadow-glow)' : 'var(--shadow-md)'
    }}
  >
    {/* 装饰光晕 */}
    {isActive && (
      <div style={{
        position: 'absolute', top: -40, right: -40,
        width: 120, height: 120, borderRadius: '50%',
        background: `radial-gradient(circle, rgba(var(--accent-rgb), 0.12) 0%, transparent 70%)`,
        pointerEvents: 'none',
      }} />
    )}

    {/* 标题行 */}
    <div>
      <Typography.Title level={isHero ? 4 : 5} style={{
        color: C('color-text'), marginBottom: 8, marginTop: 0,
        display: 'flex', alignItems: 'center', gap: 8,
      }}>
        <BookOutlined style={{
          color: isActive ? C('color-primary') : C('color-text-secondary'),
          fontSize: isHero ? 20 : 16,
        }} />
        {card.title}
        {isActive && (
          <Tag color="green" style={{ marginLeft: 4, fontSize: 10, lineHeight: '18px' }}>
            已打开
          </Tag>
        )}
      </Typography.Title>

      {/* 标签 */}
      <Space size={4} wrap style={{ marginBottom: 12 }}>
        {card.genre && card.genre !== '未分类' && (
          <Tag color="#60a5fa" style={{ fontSize: 11, borderRadius: 6 }}>{card.genre}</Tag>
        )}
        {card.style && card.style !== '默认' && (
          <Tag color="#c084fc" style={{ fontSize: 11, borderRadius: 6 }}>{card.style}</Tag>
        )}
      </Space>

      {/* 统计信息 */}
      <Space direction="vertical" size={4} style={{ width: '100%' }}>
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12 }}>
          <FileTextOutlined style={{ marginRight: 6 }} />
          {card.chapter_count > 0
            ? `${card.word_count.toLocaleString()} 字 · ${card.chapter_count} 章`
            : '尚未开始写作'}
        </Typography.Text>
        {isHero && card.word_count > 0 && (
          <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, lineHeight: 1.8 }}>
            {card.chapter_count > 0 && `平均 ${Math.round(card.word_count / Math.max(card.chapter_count, 1)).toLocaleString()} 字/章`}
          </Typography.Text>
        )}
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 11 }}>
          <ClockCircleOutlined style={{ marginRight: 6 }} />
          {formatRelativeTime(card.last_opened_at)}
        </Typography.Text>
      </Space>
    </div>

    {/* 操作区 */}
    <div style={{ marginTop: 12, display: 'flex', justifyContent: 'flex-end' }}>
      <Popconfirm
        key="delete"
        title="确定删除？"
        description={`「${card.title}」的所有数据将被永久删除`}
        onConfirm={(e) => { e?.stopPropagation(); onDelete(card) }}
        onCancel={(e) => e?.stopPropagation()}
        okText="删除"
        cancelText="取消"
        okButtonProps={{ danger: true }}
      >
        <Button
          type="text" size="small" danger
          icon={<DeleteOutlined />}
          onClick={(e) => e.stopPropagation()}
        >
          删除
        </Button>
      </Popconfirm>
    </div>
  </div>
)

export default ProjectCardItem
