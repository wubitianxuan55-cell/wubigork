// PersonaPicker.tsx — 聊天内角色选择器
// 聊天只做「选角色」：这里只展示角色库中可聊天的角色并切换，管理与编辑一律去角色库。
import React, { useCallback, useEffect, useState } from 'react'
import { Popover, Input, Tag, Typography, Button, Empty } from 'antd'
import { SearchOutlined, SwapOutlined, TeamOutlined, UserOutlined } from '@ant-design/icons'
import * as App from '../../src/wailsjsCompat'
import type { characterlib } from '../../wailsjs/go/models'
import { C } from '../utils/theme'

type LibraryCharacter = characterlib.Character

const KIND_LABELS: Record<string, { label: string; color: string }> = {
  builtin: { label: '内置', color: 'gold' },
  custom: { label: '自定义', color: 'green' },
  assistant: { label: '助手', color: 'geekblue' },
}

interface Props {
  children: React.ReactNode
  activeId: string
  onSelect: (id: string) => void
  onManage: () => void
  placement?: 'bottomLeft' | 'bottomRight' | 'topLeft' | 'topRight'
}

const PersonaPicker: React.FC<Props> = ({ children, activeId, onSelect, onManage, placement }) => {
  const [open, setOpen] = useState(false)
  const [items, setItems] = useState<LibraryCharacter[]>([])
  const [query, setQuery] = useState('')

  const load = useCallback(async () => {
    try {
      const res = await App.CharacterList('', '', true, 1, 500)
      const data = res as unknown as { items?: LibraryCharacter[] }
      setItems(data.items || [])
    } catch (_) { setItems([]) }
  }, [])

  useEffect(() => {
    if (open) {
      setQuery('')
      load()
    }
  }, [open, load])

  const filtered = items.filter(c => {
    if (!query.trim()) return true
    const q = query.trim().toLowerCase()
    return c.name.toLowerCase().includes(q) ||
      (c.tags || []).some(t => t.toLowerCase().includes(q))
  })

  const content = (
    <div style={{ width: 300 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
        <Typography.Text strong style={{ color: C('color-text'), fontSize: 13 }}>选择角色</Typography.Text>
        <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 10.5 }}>
          共 {items.length} 个可聊天角色
        </Typography.Text>
      </div>
      <Input size="small" allowClear prefix={<SearchOutlined style={{ color: C('color-text-secondary') }} />}
        placeholder="搜索名称 / 标签" value={query} onChange={e => setQuery(e.target.value)}
        autoFocus style={{ marginBottom: 8 }} />
      <div style={{ maxHeight: 320, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 4 }}>
        {filtered.length === 0 ? (
          <Empty description="没有可聊天角色" image={Empty.PRESENTED_IMAGE_SIMPLE} style={{ margin: '24px 0' }} />
        ) : filtered.map(c => {
          const km = KIND_LABELS[c.kind] || KIND_LABELS.custom
          const active = c.id === activeId
          return (
            <div key={c.id}
              onClick={() => { onSelect(c.id); setOpen(false) }}
              style={{
                display: 'flex', alignItems: 'center', gap: 8, padding: '7px 8px', borderRadius: 10, cursor: 'pointer',
                background: active ? 'rgba(244,114,182,0.12)' : 'transparent',
                border: active ? '1px solid rgba(244,114,182,0.35)' : '1px solid transparent',
              }}
              onMouseEnter={e => { if (!active) e.currentTarget.style.background = 'rgba(255,255,255,0.05)' }}
              onMouseLeave={e => { if (!active) e.currentTarget.style.background = 'transparent' }}
            >
              {c.portraitUrl
                ? <img src={c.portraitUrl} alt={c.name} style={{ width: 32, height: 32, borderRadius: 8, objectFit: 'cover', flexShrink: 0 }} />
                : <div style={{ width: 32, height: 32, borderRadius: 8, background: 'rgba(244,114,182,0.12)', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                    <UserOutlined style={{ fontSize: 14, color: '#f472b6' }} />
                  </div>}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
                  <span style={{ fontSize: 12.5, fontWeight: 600, color: C('color-text'), whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{c.name}</span>
                  <Tag color={km.color} style={{ fontSize: 8.5, margin: 0, lineHeight: '13px' }}>{km.label}</Tag>
                </div>
                {(c.tags || []).length > 0 && (
                  <div style={{ fontSize: 10, color: C('color-text-secondary'), whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                    {c.tags!.slice(0, 3).join(' · ')}
                  </div>
                )}
              </div>
              {active
                ? <Tag color="pink" style={{ margin: 0, fontSize: 9, lineHeight: '14px' }}>当前</Tag>
                : <SwapOutlined style={{ color: C('color-text-secondary'), fontSize: 12 }} />}
            </div>
          )
        })}
      </div>
      <div style={{ borderTop: '1px solid var(--border-subtle)', marginTop: 8, paddingTop: 8, display: 'flex', justifyContent: 'flex-end' }}>
        <Button size="small" type="text" icon={<TeamOutlined />}
          onClick={() => { setOpen(false); onManage() }}
          style={{ fontSize: 11.5, color: 'var(--gaea-glow)' }}>去角色库管理角色</Button>
      </div>
    </div>
  )

  return (
    <Popover open={open} onOpenChange={setOpen} trigger="click" placement={placement || 'bottomLeft'}
      content={content} arrow={false} styles={{ body: { padding: 10 } }}>
      {children}
    </Popover>
  )
}

export default PersonaPicker
