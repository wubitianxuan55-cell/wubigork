// WhisperMemoryList.tsx — 角色记忆只读列表（按领域分组，供角色库「记忆」页使用）
import React, { useState } from 'react'
import { Input, Tag } from 'antd'
import { SearchOutlined, InboxOutlined, StarFilled } from '@ant-design/icons'
import { C } from '../utils/theme'

const DOMAIN_LABELS: Record<string, string> = {
  IDENTITY: '身份', SOCIAL: '社交', DAILY_LIFE: '日常',
  PURSUITS: '追求', INNER_WORLD: '内心', TEMPORAL: '时间',
}
const DOMAIN_ORDER = ['IDENTITY', 'SOCIAL', 'DAILY_LIFE', 'PURSUITS', 'INNER_WORLD', 'TEMPORAL']
const SUB_LABELS: Record<string, string> = {
  BASIC_PROFILE: '基本信息', LIFE_STORY: '人生故事', VALUES_BELIEFS: '价值观', SELF_PERCEPTION: '自我认知',
  OUR_BOND: '我们的羁绊', FAMILY: '家庭', FRIENDS: '朋友', PARTNER: '伴侣',
  ROUTINES: '日常习惯', HEALTH: '健康', LIVING_SPACE: '居住', LIFESTYLE: '生活方式',
  CAREER: '职业', LEARNING: '学习', GOALS: '目标', PROJECTS: '项目', PROCEDURES: '流程',
  MOOD: '情绪', TASTES: '品味', VULNERABILITIES: '脆弱面', INSIDE_JOKES: '内部梗',
  NOW: '当下', COMMITMENTS: '承诺', PLANS: '计划', WORLD: '世界观',
}

import type { MemoryFact } from './WhisperMemoryModal'

export type { MemoryFact }

interface Props {
  facts: MemoryFact[]
  onOpenManage: () => void
}

const WhisperMemoryList: React.FC<Props> = ({ facts, onOpenManage }) => {
  const [search, setSearch] = useState('')
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set())

  const filtered = facts.filter((f: MemoryFact) =>
    !search ||
    String(f.subject || '').toLowerCase().includes(search.toLowerCase()) ||
    String(f.summary || '').toLowerCase().includes(search.toLowerCase()))
  const grouped = DOMAIN_ORDER.map(d => ({
    domain: d,
    label: DOMAIN_LABELS[d] || d,
    facts: filtered.filter((f: MemoryFact) => f.domain === d || f.domain === d.toLowerCase()),
  })).filter(g => g.facts.length > 0)

  const toggle = (d: string) => setCollapsed(prev => {
    const next = new Set(prev)
    if (next.has(d)) {
      next.delete(d)
    } else {
      next.add(d)
    }
    return next
  })

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', gap: 4, padding: '2px 8px 8px' }}>
      <Input
        prefix={<SearchOutlined />} size="small" placeholder="搜索记忆"
        value={search} onChange={e => setSearch(e.target.value)} allowClear
        style={{ borderRadius: 8, fontSize: 11 }}
      />
      <div style={{ flex: 1, overflow: 'auto', minHeight: 0 }}>
        {facts.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 26, color: C('color-text-secondary'), fontSize: 12, lineHeight: 1.7 }}>
            <InboxOutlined style={{ fontSize: 20, opacity: 0.5, marginBottom: 8, display: 'block' }} />
            还没有记忆，多聊几句吧
          </div>
        ) : filtered.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 26, color: C('color-text-secondary'), fontSize: 12 }}>无匹配</div>
        ) : (
          grouped.map(g => {
            const isCollapsed = collapsed.has(g.domain)
            const coreCount = g.facts.filter((f: MemoryFact) => f.tier === 'core').length
            return (
              <div key={g.domain} style={{ marginBottom: 2 }}>
                <div
                  role="button" tabIndex={0}
                  onClick={() => toggle(g.domain)}
                  onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); toggle(g.domain) } }}
                  style={{
                    display: 'flex', alignItems: 'center', gap: 4, padding: '6px',
                    borderRadius: 8, cursor: 'pointer', userSelect: 'none',
                    background: isCollapsed ? 'transparent' : `${C('bg-elevated')}80`,
                    transition: 'background 150ms',
                  }}
                >
                  <span style={{ fontSize: 10, transition: 'transform 200ms', transform: isCollapsed ? 'rotate(-90deg)' : 'rotate(0deg)' }}>▼</span>
                  <span style={{ fontSize: 12, fontWeight: 600, color: C('color-text'), flex: 1 }}>{g.label}</span>
                  <Tag style={{ fontSize: 9, margin: 0, padding: '0 5px', lineHeight: '16px', background: 'transparent', border: '1px solid var(--md-sys-color-outline-variant)', color: C('color-text-secondary') }}>
                    {g.facts.length}
                  </Tag>
                  {coreCount > 0 && <StarFilled style={{ fontSize: 9, color: 'var(--color-warning)' }} />}
                </div>
                {!isCollapsed && g.facts.map((f: MemoryFact) => (
                  <div
                    key={f.id}
                    role="button" tabIndex={0}
                    onClick={onOpenManage}
                    onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onOpenManage() } }}
                    style={{
                      padding: '6px 8px 6px 18px', margin: '1px 0', borderRadius: 8, cursor: 'pointer',
                      background: f.tier === 'core' ? `${C('color-primary')}06` : 'transparent',
                      borderLeft: f.tier === 'core' ? `2px solid var(--color-warning)` : '2px solid transparent',
                      transition: 'background 150ms',
                    }}
                  >
                    <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                      {f.tier === 'core' && <StarFilled style={{ color: 'var(--color-warning)', fontSize: 9 }} />}
                      <span style={{ fontSize: 11, fontWeight: 600, color: C('color-text'), flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {f.subject}
                      </span>
                      <span style={{ fontSize: 9, color: C('color-text-secondary'), opacity: 0.55, flexShrink: 0 }}>
                        {(f.subcategory ? SUB_LABELS[f.subcategory] : undefined) || f.subcategory || ''}
                      </span>
                    </div>
                    <div style={{ fontSize: 9, color: C('color-text-secondary'), marginTop: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', opacity: 0.7 }}>
                      {String(f.summary || '').slice(0, 50)}{f.summary?.length > 50 ? '…' : ''}
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 2 }}>
                      <span style={{ fontSize: 8, color: C('color-text-secondary'), opacity: 0.55 }}>W{f.weight?.toFixed?.(1) ?? '–'}</span>
                      {f.emotionalContext?.valence != null && (
                        <span style={{ fontSize: 8, color: f.emotionalContext.valence > 0.2 ? 'var(--color-success)' : f.emotionalContext.valence < -0.2 ? 'var(--color-destructive)' : 'var(--color-text-secondary)' }}>
                          {f.emotionalContext.valence > 0.2 ? '正' : f.emotionalContext.valence < -0.2 ? '负' : '平'}
                        </span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )
          })
        )}
      </div>
    </div>
  )
}

export default WhisperMemoryList
