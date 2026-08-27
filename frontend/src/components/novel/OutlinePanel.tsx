import React from 'react'
import { Button, Tooltip } from 'antd'
import {
  BookOutlined, UnorderedListOutlined,
  MenuFoldOutlined, MenuUnfoldOutlined,
} from '@ant-design/icons'
import { C, STATUS_COLORS, STATUS_LABELS } from '../../utils/theme'
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
            const statusColor = STATUS_COLORS[n.status as keyof typeof STATUS_COLORS] || STATUS_COLORS.planned
            const statusLabel = STATUS_LABELS[n.status as keyof typeof STATUS_LABELS] || '未写'
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
      width: collapsed ? 40 : '100%', flexShrink: 0, overflow: 'hidden',
      display: 'flex', flexDirection: 'column',
    }}>
      <div className="novel-zone-head" style={{
        padding: collapsed ? '0 8px' : '0 12px',
      }}>
        {!collapsed && (
          <span className="novel-zone-title" style={{ fontSize: 12 }}>
            <UnorderedListOutlined />大纲
          </span>
        )}
        <div style={{ flex: 1 }} />
        <Tooltip title={collapsed ? '展开大纲' : '收起大纲'}>
          <Button type="text" size="small" className="novel-outline-toggle"
            icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
            onClick={onToggleCollapse}
            aria-label={collapsed ? '展开大纲' : '收起大纲'}
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
              <span style={{ fontSize: 9, color: STATUS_COLORS.planned, display: 'flex', alignItems: 'center', gap: 3 }}>
                <span className="novel-dot" style={{ background: STATUS_COLORS.planned }} />{STATUS_LABELS.planned}
              </span>
              <span style={{ fontSize: 9, color: STATUS_COLORS.writing, display: 'flex', alignItems: 'center', gap: 3 }}>
                <span className="novel-dot" style={{ background: STATUS_COLORS.writing }} />{STATUS_LABELS.writing}
              </span>
              <span style={{ fontSize: 9, color: STATUS_COLORS.done, display: 'flex', alignItems: 'center', gap: 3 }}>
                <span className="novel-dot" style={{ background: STATUS_COLORS.done }} />{STATUS_LABELS.done}
              </span>
              <span style={{ fontSize: 9, color: STATUS_COLORS.abandoned, display: 'flex', alignItems: 'center', gap: 3 }}>
                <span className="novel-dot" style={{ background: STATUS_COLORS.abandoned }} />{STATUS_LABELS.abandoned}
              </span>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

export default OutlinePanel
