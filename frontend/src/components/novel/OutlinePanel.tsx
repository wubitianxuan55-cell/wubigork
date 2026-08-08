import React from 'react'
import { Typography, Button, Tooltip } from 'antd'
import {
  BookOutlined, UnorderedListOutlined,
  MenuFoldOutlined, MenuUnfoldOutlined,
} from '@ant-design/icons'
import { C } from '../../utils/theme'
import type { OutlineNode } from '../../types'

interface OutlinePanelProps {
  outlines: OutlineNode[]
  activeKey: string
  onSelectNode: (node: OutlineNode) => void
  collapsed: boolean
  onToggleCollapse: () => void
}

/** OutlinePanel — 左侧大纲面板 */
const OutlinePanel: React.FC<OutlinePanelProps> = ({
  outlines, activeKey, onSelectNode, collapsed, onToggleCollapse,
}) => {
  const renderOutlineCards = () => {
    const cards: React.ReactNode[] = []
    const walk = (nodes: OutlineNode[], depth: number) => {
      nodes.forEach((n) => {
        const isVolume = (n.children?.length || 0) > 0 || n.id.startsWith('vol_')
        const isSelected = n.id === activeKey
        if (isVolume) {
          cards.push(
            <div key={n.id} style={{
              padding: '6px 10px', marginTop: depth === 0 ? 0 : 4,
              background: 'color-mix(in srgb, var(--gaea-glow) 10%, transparent)',
              borderLeft: '3px solid var(--gaea-glow)', borderRadius: '0 4px 4px 0',
              fontSize: 12, fontWeight: 600, color: 'var(--gaea-glow)',
              display: 'flex', alignItems: 'center', gap: 6,
            }}>
              <BookOutlined style={{ fontSize: 11 }} />
              <span style={{ flex: 1 }}>{n.title}</span>
            </div>
          )
        } else {
            const statusColor =
              n.status === 'writing' ? '#f59e0b' :
              n.status === 'done' ? '#22c55e' :
              n.status === 'abandoned' ? '#374151' : '#6b7280'
            const statusLabel =
              n.status === 'writing' ? '草稿' :
              n.status === 'done' ? '定稿' :
              n.status === 'abandoned' ? '废弃' : '未写'
          cards.push(
            <div key={n.id} className="novel-outline-row" onClick={() => onSelectNode(n)} style={{
              padding: '6px 10px', paddingLeft: 10 + depth * 16,
              margin: '2px 0', cursor: 'pointer',
              background: isSelected ? 'color-mix(in srgb, var(--gaea-glow) 12%, transparent)' : 'transparent',
              borderRadius: '0 var(--radius-sm) var(--radius-sm) 0',
              borderLeft: isSelected ? '3px solid var(--gaea-glow)' : '3px solid transparent',
            }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                <span style={{ color: 'var(--gaea-glow)', fontSize: 10, fontWeight: 600 }}>
                  {n.order_index || '·'}
                </span>
                <span style={{ flex: 1, fontSize: 12 }}>{(n.title || '').trim() || `第${n.order_index || '?'}章`}</span>
                <span style={{
                  width: 7, height: 7, borderRadius: '50%',
                  background: statusColor, display: 'inline-block',
                  flexShrink: 0,
                }} />
                <span style={{ fontSize: 9, color: statusColor, flexShrink: 0 }}>
                  {statusLabel}
                </span>
              </div>
            </div>
          )
        }
        if (n.children?.length) walk(n.children, depth + 1)
      })
    }
    walk(outlines, 0)
    return cards
  }

  return (
    <div className="novel-panel" style={{
      width: collapsed ? 40 : 220, flexShrink: 0, overflow: 'hidden',
      display: 'flex', flexDirection: 'column', transition: 'width 0.2s',
    }}>
      <div style={{
        padding: collapsed ? '10px 8px' : '10px 14px',
        borderBottom: '1px solid ' + C('color-border'),
        display: 'flex', justifyContent: 'space-between', alignItems: 'center',
      }}>
        {!collapsed && (
          <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>
            <UnorderedListOutlined style={{ marginRight: 6 }} />大纲
          </Typography.Text>
        )}
        <Tooltip title={collapsed ? '展开大纲' : '收起大纲'}>
          <Button type="text" size="small"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={onToggleCollapse}
            style={{ color: C('color-text-secondary'), fontSize: 11, padding: collapsed ? '0' : undefined }}
          />
        </Tooltip>
      </div>
      {!collapsed && (
        <div style={{ flex: 1, overflow: 'auto', padding: 6, display: 'flex', flexDirection: 'column' }}>
          <div style={{ flex: 1 }}>
            {outlines.length === 0
              ? <div style={{ textAlign: 'center', color: C('color-text-secondary'), marginTop: 32, fontSize: 12 }}>暂无大纲</div>
              : renderOutlineCards()
            }
          </div>
          {outlines.length > 0 && (
            <div style={{
              padding: '6px 0 2px', borderTop: '1px solid ' + C('color-border'),
              display: 'flex', justifyContent: 'center', gap: 12, flexShrink: 0,
            }}>
              <span style={{ fontSize: 9, color: '#6b7280', display: 'flex', alignItems: 'center', gap: 3 }}>
                <span style={{ width: 6, height: 6, borderRadius: '50%', background: '#6b7280', display: 'inline-block' }} />未写
              </span>
              <span style={{ fontSize: 9, color: '#f59e0b', display: 'flex', alignItems: 'center', gap: 3 }}>
                <span style={{ width: 6, height: 6, borderRadius: '50%', background: '#f59e0b', display: 'inline-block' }} />草稿
              </span>
              <span style={{ fontSize: 9, color: '#22c55e', display: 'flex', alignItems: 'center', gap: 3 }}>
                <span style={{ width: 6, height: 6, borderRadius: '50%', background: '#22c55e', display: 'inline-block' }} />定稿
              </span>
              <span style={{ fontSize: 9, color: '#374151', display: 'flex', alignItems: 'center', gap: 3 }}>
                <span style={{ width: 6, height: 6, borderRadius: '50%', background: '#374151', display: 'inline-block' }} />废弃
              </span>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export default OutlinePanel
