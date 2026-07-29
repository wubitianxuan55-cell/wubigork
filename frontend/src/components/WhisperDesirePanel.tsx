// WhisperDesirePanel.tsx — 欲望栈 + 重逢状态展示（对齐 Ackem EmotionPanel）
// 增强：折叠展示、dormant 态、dismiss 按钮

import React, { useState } from 'react'

interface Desire {
  id: string
  topic: string
  category: string
  urgency: number
  status: string
}

interface ReunionInfo {
  gapHours: number
  timePhrase: string
  moodPhrase: string
}

interface Props {
  desireStack?: { slots: (Desire | null)[] }
  reunion?: ReunionInfo | null
  sharedEventsCount?: number
  onDismissDesire?: (id: string) => void
}

const CATEGORY_LABELS: Record<string, string> = {
  concern: '关心', curiosity: '好奇', share: '分享', tease: '捉弄', suggest: '建议',
}
const CATEGORY_EMOJI: Record<string, string> = {
  concern: '💛', curiosity: '🔍', share: '💬', tease: '😏', suggest: '💡',
}

function DesireBar({ desire, onDismiss }: { desire: Desire; onDismiss?: (id: string) => void }) {
  const pct = Math.min(100, (desire.urgency / 10) * 100)
  const emoji = CATEGORY_EMOJI[desire.category] || '✨'
  const label = CATEGORY_LABELS[desire.category] || desire.category
  const isDormant = desire.status === 'dormant'

  return (
    <div style={{
      display: 'flex', alignItems: 'center', gap: 6, padding: '3px 0',
      opacity: isDormant ? 0.45 : 1, transition: 'opacity 0.3s',
      fontSize: 12,
    }}>
      <span style={{ width: 20, textAlign: 'center' }}>{emoji}</span>
      <span style={{ width: 36, color: '#999', fontSize: 11 }}>{label}</span>
      <span style={{ flex: 1, color: '#ccc', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
        {desire.topic}
      </span>
      <div style={{
        width: 48, height: 6, background: 'rgba(255,255,255,0.06)',
        borderRadius: 3, overflow: 'hidden', flexShrink: 0,
      }}>
        <div style={{
          height: '100%', borderRadius: 3, width: `${pct}%`,
          background: pct > 70 ? '#f59e0b' : pct > 40 ? '#d97706' : '#52525b',
          transition: 'width 0.5s',
        }} />
      </div>
      {onDismiss && (
        <button
          onClick={e => { e.stopPropagation(); onDismiss(desire.id) }}
          style={{
            background: 'none', border: 'none', color: '#666', cursor: 'pointer',
            fontSize: 14, padding: '0 2px', lineHeight: 1,
          }}
          title="忽略"
        >×</button>
      )}
    </div>
  )
}

export default function WhisperDesirePanel({ desireStack, reunion, sharedEventsCount, onDismissDesire }: Props) {
  const [collapsed, setCollapsed] = useState(false)
  const allSlots = desireStack?.slots ?? []
  const activeDesires = allSlots.filter((d): d is Desire => d !== null && d.status === 'active')
  const dormantDesires = allSlots.filter((d): d is Desire => d !== null && d.status === 'dormant')

  return (
    <div>
      {/* 重逢状态 */}
      {reunion && reunion.gapHours > 0 && (
        <div style={{
          background: 'rgba(180,83,9,0.15)', border: '1px solid rgba(180,83,9,0.25)',
          borderRadius: 8, padding: '8px 10px', marginBottom: 10,
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12 }}>
            <span style={{ color: '#fbbf24' }}>🕐</span>
            <span style={{ color: '#fcd34d', fontWeight: 600 }}>{reunion.timePhrase}没见了</span>
          </div>
          <p style={{ color: '#888', fontSize: 11, margin: '4px 0 0' }}>{reunion.moodPhrase}</p>
        </div>
      )}

      {/* 共享事件计数 */}
      {sharedEventsCount !== undefined && sharedEventsCount > 0 && (
        <div style={{ color: '#888', fontSize: 11, display: 'flex', alignItems: 'center', gap: 4, marginBottom: 8 }}>
          <span>📝 共同经历 {sharedEventsCount} 件事</span>
        </div>
      )}

      {/* 欲望栈 — 折叠 */}
      <details open={!collapsed} onToggle={e => setCollapsed(!(e.currentTarget as HTMLDetailsElement).open)}>
        <summary style={{
          cursor: 'pointer', color: '#888', fontSize: 11, display: 'flex',
          alignItems: 'center', gap: 4, userSelect: 'none', marginBottom: 4,
        }}>
          <span>💭 心里想着 ({activeDesires.length})</span>
        </summary>

        {activeDesires.length > 0 ? (
          <div style={{ marginBottom: 4 }}>
            {activeDesires.slice(0, 5).map(d => (
              <DesireBar key={d.id} desire={d} onDismiss={onDismissDesire} />
            ))}
          </div>
        ) : (
          <div style={{ color: '#666', fontSize: 11, fontStyle: 'italic', padding: '4px 0' }}>
            随遇而安
          </div>
        )}

        {/* dormant 欲望 */}
        {dormantDesires.length > 0 && (
          <div style={{ marginTop: 4 }}>
            <div style={{ color: '#555', fontSize: 10, marginBottom: 2 }}>💤 休眠 ({dormantDesires.length})</div>
            {dormantDesires.slice(0, 3).map(d => (
              <DesireBar key={d.id} desire={d} />
            ))}
          </div>
        )}
      </details>
    </div>
  )
}
