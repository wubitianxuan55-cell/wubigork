// WhisperMemoryTimeline.tsx — 记忆时间线（对齐 Ackem MemoryTimeline）
// 增强：情绪上下文标签、子分类badge、👍/👎反馈、内联编辑

import React, { useState } from 'react'

interface MemoryFact {
  id: string
  domain: string
  subcategory?: string
  subject: string
  summary: string
  weight: number
  confidence: number
  createdAt: string
  tier?: string
  emotionalContext?: { valence: number; relStage: string; trust: number }
}

interface Props {
  facts: MemoryFact[]
  onDelete?: (id: string) => void
  onFeedback?: (id: string, type: 'thumbs_up' | 'thumbs_down') => void
  onUpdate?: (id: string, summary: string) => void
}

const DOMAIN_LABELS: Record<string, string> = {
  personal: '个人信息', preference: '偏好', relationship: '关系',
  shared_bond: '共同经历', health: '健康', work: '工作',
}

function valenceLabel(v: number): { text: string; color: string } {
  if (v > 0.3) return { text: '正面', color: '#4ade80' }
  if (v < -0.3) return { text: '负面', color: '#f87171' }
  return { text: '中性', color: 'var(--whisper-ink-muted)' }
}

function groupByDate(facts: MemoryFact[]): Map<string, MemoryFact[]> {
  const groups = new Map<string, MemoryFact[]>()
  for (const f of facts) {
    const date = f.createdAt.slice(0, 10)
    const arr = groups.get(date) || []
    arr.push(f)
    groups.set(date, arr)
  }
  return groups
}

function FactCard({ fact, onDelete, onFeedback, onUpdate }: {
  fact: MemoryFact; onDelete?: (id: string) => void
  onFeedback?: (id: string, type: 'thumbs_up' | 'thumbs_down') => void
  onUpdate?: (id: string, summary: string) => void
}) {
  const [expanded, setExpanded] = useState(false)
  const [editing, setEditing] = useState(false)
  const [editText, setEditText] = useState(fact.summary)
  const domainLabel = DOMAIN_LABELS[fact.domain] || fact.domain
  const coreStyle = fact.tier === 'core'
    ? { borderLeft: '2px solid var(--whisper-accent)' }
    : { borderLeft: '2px solid var(--whisper-glass-border)' }
  const eco = fact.emotionalContext
  const vLabel = eco ? valenceLabel(eco.valence) : null

  const handleSave = () => {
    if (editText.trim() && editText !== fact.summary) {
      onUpdate?.(fact.id, editText.trim())
    }
    setEditing(false)
  }

  return (
    <div style={{
      background: 'rgba(24,24,27,0.6)', borderRadius: '0 6px 6px 0',
      padding: '8px 10px', marginBottom: 4, ...coreStyle,
    }}>
      <div
        style={{ display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer' }}
        onClick={() => setExpanded(!expanded)}
      >
        <span style={{ color: 'var(--whisper-ink)', fontSize: 12, fontWeight: 500, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {fact.subject}
        </span>
        {fact.subcategory && (
          <span style={{
            background: 'rgba(251,191,36,0.15)', color: 'var(--whisper-accent)',
            borderRadius: 3, padding: '1px 6px', fontSize: 10,
          }}>{fact.subcategory}</span>
        )}
        <span style={{ color: 'var(--whisper-ink-muted)', fontSize: 10 }}>{domainLabel}</span>
      </div>

      {expanded && (
        <div style={{ marginTop: 6, display: 'flex', flexDirection: 'column', gap: 6 }}>
          {/* 情绪上下文 */}
          {eco && (
            <div style={{ display: 'flex', gap: 10, fontSize: 10, color: 'var(--whisper-ink-muted)' }}>
              {vLabel && <span style={{ color: vLabel.color }}>{vLabel.text}</span>}
              <span>信任 {eco.trust.toFixed(0)}</span>
              <span>{eco.relStage}</span>
            </div>
          )}

          {/* 摘要 — 可编辑 */}
          {editing ? (
            <div style={{ display: 'flex', gap: 4 }}>
              <input
                value={editText}
                onChange={e => setEditText(e.target.value)}
                onKeyDown={e => { if (e.key === 'Enter') handleSave(); if (e.key === 'Escape') setEditing(false) }}
                autoFocus
                style={{
                  flex: 1, background: 'rgba(255,255,255,0.06)', border: '1px solid rgba(255,255,255,0.1)',
                  borderRadius: 4, color: 'var(--whisper-ink)', fontSize: 12, padding: '4px 8px',
                }}
              />
              <button onClick={handleSave} style={{
                background: 'rgba(251,191,36,0.2)', border: 'none', borderRadius: 4,
                color: 'var(--whisper-accent)', fontSize: 11, padding: '4px 8px', cursor: 'pointer',
              }}>保存</button>
            </div>
          ) : (
            <p style={{ color: 'var(--whisper-ink-muted)', fontSize: 11, margin: 0 }}>{fact.summary}</p>
          )}

          {/* 元数据 */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 10, color: 'var(--whisper-ink-muted)', flexWrap: 'wrap' }}>
            <span>权重 {fact.weight.toFixed(1)}</span>
            <span>置信 {(fact.confidence * 100).toFixed(0)}%</span>
            {fact.tier === 'core' && <span style={{ color: 'var(--whisper-accent)' }}>★ 核心记忆</span>}

            {/* 反馈按钮 */}
            {onFeedback && (
              <span style={{ display: 'flex', gap: 4, marginLeft: 'auto' }}>
                <button onClick={e => { e.stopPropagation(); onFeedback(fact.id, 'thumbs_up') }}
                  style={{ background: 'none', border: 'none', color: 'var(--whisper-ink-muted)', cursor: 'pointer', fontSize: 13 }}
                  title="有用">👍</button>
                <button onClick={e => { e.stopPropagation(); onFeedback(fact.id, 'thumbs_down') }}
                  style={{ background: 'none', border: 'none', color: 'var(--whisper-ink-muted)', cursor: 'pointer', fontSize: 13 }}
                  title="无用">👎</button>
              </span>
            )}

            {/* 编辑 */}
            {onUpdate && !editing && (
              <button onClick={e => { e.stopPropagation(); setEditing(true); setEditText(fact.summary) }}
                style={{ background: 'none', border: 'none', color: 'var(--whisper-ink-muted)', cursor: 'pointer', fontSize: 13 }}
                title="编辑">✏️</button>
            )}

            {/* 删除 */}
            {onDelete && (
              <button onClick={e => { e.stopPropagation(); onDelete(fact.id) }}
                style={{ background: 'none', border: 'none', color: '#f87171', cursor: 'pointer', fontSize: 11 }}
              >删除</button>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

export default function WhisperMemoryTimeline({ facts, onDelete, onFeedback, onUpdate }: Props) {
  const groups = groupByDate(facts)

  if (facts.length === 0) {
    return (
      <div style={{ color: 'var(--whisper-ink-muted)', fontSize: 12, padding: 16, textAlign: 'center' }}>
        还没有关于你的记忆 — 多聊聊，我会记住的
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{
        display: 'flex', alignItems: 'center', gap: 8, padding: '8px 12px',
        borderBottom: '1px solid rgba(255,255,255,0.08)', fontSize: 12, color: 'var(--whisper-ink-muted)',
      }}>
        <span>🧠 记忆时间线</span>
        <span style={{ color: 'var(--whisper-ink-muted)', marginLeft: 'auto' }}>{facts.length} 条</span>
      </div>
      <div style={{ flex: 1, overflow: 'auto', padding: '6px 10px' }}>
        {Array.from(groups.entries()).map(([date, dateFacts]) => (
          <div key={date} style={{ marginBottom: 10 }}>
            <div style={{
              color: 'var(--whisper-ink-muted)', fontSize: 10, marginBottom: 4,
              position: 'sticky', top: 0, background: 'var(--whisper-surface)', padding: '2px 0',
            }}>
              {date}
            </div>
            {dateFacts.map(f => (
              <FactCard key={f.id} fact={f}
                onDelete={onDelete} onFeedback={onFeedback} onUpdate={onUpdate}
              />
            ))}
          </div>
        ))}
      </div>
    </div>
  )
}
