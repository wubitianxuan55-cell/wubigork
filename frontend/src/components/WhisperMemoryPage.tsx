// WhisperMemoryPage.tsx — 轻语记忆管理页
// 对齐 ackem MemoryPage + MemoryTimeline

import React, { useState, useMemo } from 'react'
import { Input, Button, Modal, message, Tag } from 'antd'
import { SearchOutlined, DeleteOutlined, EyeOutlined } from '@ant-design/icons'

interface MemoryFact {
  id: string; domain: string; subject: string; summary: string
  weight: number; confidence: number; tier?: string; createdAt: string
}

interface Props {
  facts: MemoryFact[]
  onDelete?: (id: string) => void
}

const DOMAIN_LABELS: Record<string, string> = {
  personal: '个人信息', preference: '偏好', relationship: '关系',
  shared_bond: '共同经历', health: '健康', work: '工作'
}

export default function WhisperMemoryPage({ facts, onDelete }: Props) {
  const [search, setSearch] = useState('')
  const [selectedFact, setSelectedFact] = useState<MemoryFact | null>(null)

  const filtered = useMemo(() => {
    if (!search.trim()) return facts
    const q = search.toLowerCase()
    return facts.filter(f =>
      f.subject.toLowerCase().includes(q) ||
      f.summary.toLowerCase().includes(q) ||
      (DOMAIN_LABELS[f.domain] || f.domain).toLowerCase().includes(q)
    )
  }, [facts, search])

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 12, height: '100%' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <span style={{ fontSize: 13, fontWeight: 600, color: 'var(--whisper-ink)' }}>🧠 记忆管理</span>
        <Tag style={{ marginLeft: 'auto' }}>{facts.length} 条</Tag>
      </div>

      <Input prefix={<SearchOutlined />} placeholder="搜索记忆…" size="small"
        value={search} onChange={e => setSearch(e.target.value)}
        style={{ background: 'var(--whisper-glass-bg)', border: '1px solid var(--whisper-glass-border)', borderRadius: 8 }} />

      <div style={{ flex: 1, overflow: 'auto' }}>
        {filtered.length === 0 ? (
          <div style={{ textAlign: 'center', color: 'var(--whisper-ink-muted)', padding: 24, fontSize: 12 }}>
            {facts.length === 0 ? '还没有关于你的记忆 — 多聊聊，我会记住的' : '未找到匹配的记忆'}
          </div>
        ) : (
          filtered.map(f => (
            <div key={f.id} style={{
              padding: '8px 10px', marginBottom: 6, borderRadius: 8,
              background: 'var(--whisper-surface)', border: '1px solid var(--whisper-glass-bg)',
              borderLeft: f.tier === 'core' ? '3px solid var(--whisper-accent)' : '3px solid var(--whisper-glass-border)'
            }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                <span style={{ fontSize: 12, fontWeight: 500, color: 'var(--whisper-ink)', flex: 1 }}>{f.subject}</span>
                <span style={{ fontSize: 10, color: 'var(--whisper-ink-muted)' }}>{DOMAIN_LABELS[f.domain] || f.domain}</span>
                {f.tier === 'core' && <Tag color="gold" style={{ fontSize: 9, margin: 0 }}>核心</Tag>}
                <Button type="text" size="small" icon={<EyeOutlined />}
                  onClick={() => setSelectedFact(f)} style={{ color: 'var(--whisper-ink-muted)' }} />
                {onDelete && (
                  <Button type="text" size="small" danger icon={<DeleteOutlined />}
                    onClick={() => { onDelete(f.id); message.success('已删除') }} />
                )}
              </div>
              <div style={{ fontSize: 11, color: 'var(--whisper-ink-muted)', marginTop: 4, lineHeight: 1.5 }}>
                {f.summary.slice(0, 120)}{f.summary.length > 120 ? '…' : ''}
              </div>
              <div style={{ fontSize: 10, color: 'var(--whisper-ink-muted)', marginTop: 4 }}>
                权重:{f.weight.toFixed(1)} · 置信:{(f.confidence * 100).toFixed(0)}% · {f.createdAt?.slice(0, 10)}
              </div>
            </div>
          ))
        )}
      </div>

      <Modal title={selectedFact?.subject} open={!!selectedFact}
        onCancel={() => setSelectedFact(null)} footer={null}>
        {selectedFact && (
          <div style={{ fontSize: 13, lineHeight: 1.8, color: 'var(--whisper-ink)' }}>
            <p>{selectedFact.summary}</p>
            <div style={{ color: 'var(--whisper-ink-muted)', fontSize: 12, marginTop: 12 }}>
              <p>领域：{DOMAIN_LABELS[selectedFact.domain] || selectedFact.domain}</p>
              <p>权重：{selectedFact.weight.toFixed(2)} · 置信度：{(selectedFact.confidence * 100).toFixed(0)}%</p>
              <p>等级：{selectedFact.tier || 'memory'} · 创建：{selectedFact.createdAt?.slice(0, 10)}</p>
            </div>
          </div>
        )}
      </Modal>
    </div>
  )
}
