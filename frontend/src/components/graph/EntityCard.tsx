import React from 'react'
import { Typography, Tag, Empty } from 'antd'
import {
  UserOutlined, EnvironmentOutlined, BankOutlined,
  ToolOutlined, HistoryOutlined, BulbOutlined,
} from '@ant-design/icons'

/**
 * EntityCard — 实体详情卡片
 *
 * 显示角色/地点/组织/物品/事件/概念的详细信息
 *
 * Props:
 *   entity — 实体数据
 */
interface EntityData {
  id: string
  name: string
  type: string
  description?: string
  properties?: Record<string, string>
  chapter_refs?: string[]
}

interface EntityCardProps {
  entity: EntityData | null
}

const TYPE_ICONS: Record<string, React.ReactNode> = {
  character: <UserOutlined />,
  location: <EnvironmentOutlined />,
  organization: <BankOutlined />,
  item: <ToolOutlined />,
  event: <HistoryOutlined />,
  concept: <BulbOutlined />,
}

const TYPE_LABELS: Record<string, string> = {
  character: '角色',
  location: '地点',
  organization: '组织',
  item: '物品',
  event: '事件',
  concept: '概念',
}

const TYPE_COLORS: Record<string, string> = {
  character: '#4ade80',
  location: '#f59e0b',
  organization: '#60a5fa',
  item: '#c084fc',
  event: '#f87171',
  concept: '#9ca3af',
}

// 属性标签中文映射
const PROPERTY_LABELS: Record<string, string> = {
  role_type: '角色类型',
  gender: '性别',
  age: '年龄',
  status: '状态',
  appearance: '外貌',
  personality: '性格',
  motivation: '动机',
  arc: '角色弧光',
  type: '类型',
  power_level: '实力等级',
  location: '位置',
  motto: '格言',
}

const EntityCard: React.FC<EntityCardProps> = ({ entity }) => {
  if (!entity) {
    return <Empty description="选择节点查看详情" />
  }

  const icon = TYPE_ICONS[entity.type] || <BulbOutlined />
  const label = TYPE_LABELS[entity.type] || entity.type
  const color = TYPE_COLORS[entity.type] || '#9ca3af'

  // 过滤有意义的属性
  const meaningfulProps: [string, string][] = []
  if (entity.properties) {
    for (const [key, val] of Object.entries(entity.properties)) {
      if (val && val.trim() && key !== 'id' && key !== 'name') {
        const label = PROPERTY_LABELS[key] || key
        meaningfulProps.push([label, val])
      }
    }
  }

  return (
    <div
      style={{
        background: 'var(--bg-glass)',
        borderRadius: 'var(--radius-lg)',
        border: '1px solid var(--border-subtle)',
        borderLeft: `3px solid ${color}`,
        padding: 16,
      }}
    >
      {/* 头部 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 12 }}>
        <span style={{ color, fontSize: 18 }}>{icon}</span>
        <Typography.Title level={5} style={{ margin: 0, fontSize: 15 }}>
          {entity.name}
        </Typography.Title>
        <Tag color={color} style={{ fontSize: 10, margin: 0 }}>
          {label}
        </Tag>
      </div>

      {/* 描述 */}
      {entity.description && (
        <Typography.Paragraph
          style={{ fontSize: 12, color: 'var(--color-text-secondary)', marginBottom: 12 }}
          ellipsis={{ rows: 3 }}
        >
          {entity.description}
        </Typography.Paragraph>
      )}

      {/* 属性列表 */}
      {meaningfulProps.length > 0 && (
        <div style={{ marginBottom: 12 }}>
          {meaningfulProps.map(([key, val]) => (
            <div
              key={key}
              style={{
                display: 'flex',
                gap: 8,
                padding: '3px 0',
                fontSize: 12,
                borderBottom: '1px solid var(--border-subtle)',
              }}
            >
              <Typography.Text
                type="secondary"
                style={{ fontSize: 11, width: 56, flexShrink: 0 }}
              >
                {key}
              </Typography.Text>
              <Typography.Text style={{ fontSize: 12 }}>{val}</Typography.Text>
            </div>
          ))}
        </div>
      )}

      {/* 出场章节 */}
      {entity.chapter_refs && entity.chapter_refs.length > 0 && (
        <div>
          <Typography.Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 4 }}>
            出场章节:
          </Typography.Text>
          <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
            {entity.chapter_refs.map(ref => (
              <Tag key={ref} style={{ fontSize: 10, padding: '0 4px', margin: 0 }}>
                {ref}
              </Tag>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

export default EntityCard
export type { EntityCardProps, EntityData }
