// WhisperTracePanel.tsx — 对话追踪面板（对齐 Ackem TracePanel）
// 增强：四列网格布局、MiniBar四色、信任/aff着色、轮数选择

import React, { useState } from 'react'

interface TraceEntry {
  turn: number
  l0: { type: string; intensity: number; sincerity?: number }
  l1: { trust: number; rifts: number; stage: string; atmosphere?: string }
  l2: { aff: number; sec: number; aro: number; dom: number; label: string }
  l3: { silent: boolean; tierBChars: number; factsUsed?: number; embeddingHits?: number }
  l4: { wrote: boolean }
  l5?: { toolCalls?: string[] }
  ms?: { total?: number }
  timestamp?: string
}

interface Props {
  traces: TraceEntry[]
  currentTurn: number
}

const STAGE_LABEL: Record<string, string> = {
  STRANGER: '初识', FAMILIAR: '熟悉', INTIMATE: '亲密',
}

const EVENT_LABEL: Record<string, string> = {
  praise: '赞美', tease: '调侃', casual_chat: '闲聊', cold: '冷淡',
  hurtful: '伤害', apology: '道歉', vulnerable: '脆弱', question: '提问',
  extreme_redline: '红线',
}

// 信任颜色（绿→黄→红）
function trustColor(t: number): string {
  if (t >= 70) return '#4ade80'
  if (t >= 40) return '#facc15'
  return '#f87171'
}

// MiniBar 四色：indigo/green/amber/pink
function MiniBar2({ value, color, max }: { value: number; color: string; max: number }) {
  const pct = Math.max(0, Math.min(100, ((value + max) / (2 * max)) * 100))
  return (
    <div style={{ width: '100%', height: 4, background: 'rgba(255,255,255,0.06)', borderRadius: 2, overflow: 'hidden' }}>
      <div style={{ height: '100%', borderRadius: 2, width: `${pct}%`, background: color, transition: 'width 0.3s' }} />
    </div>
  )
}

function TraceRow({ entry, expanded, onToggle }: {
  entry: TraceEntry; expanded: boolean; onToggle: () => void
}) {
  const stageLabel = STAGE_LABEL[entry.l1.stage] || entry.l1.stage
  const eventLabel = EVENT_LABEL[entry.l0.type] || entry.l0.type
  const tColor = trustColor(entry.l1.trust)

  return (
    <div style={{ borderBottom: '1px solid rgba(255,255,255,0.05)' }}>
      <button
        onClick={onToggle}
        style={{
          width: '100%', display: 'flex', alignItems: 'center', gap: 6,
          padding: '6px 8px', background: 'none', border: 'none',
          cursor: 'pointer', textAlign: 'left', fontSize: 12, color: '#999',
        }}
        onMouseEnter={e => (e.currentTarget.style.background = 'rgba(255,255,255,0.04)')}
        onMouseLeave={e => (e.currentTarget.style.background = 'none')}
      >
        <span style={{ color: '#666', width: 22, flexShrink: 0 }}>#{entry.turn}</span>
        <span style={{ width: 44, flexShrink: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {eventLabel}
        </span>
        {entry.l3.silent && <span style={{ color: '#f59e0b', fontSize: 11 }}>🤫</span>}
        <span style={{ color: tColor, fontWeight: 600, marginLeft: 'auto', marginRight: 8 }}>
          {Math.round(entry.l1.trust)}
        </span>
        <span style={{ color: '#888', fontSize: 10 }}>{stageLabel}</span>
      </button>

      {expanded && (
        <div style={{ padding: '8px 12px 10px', display: 'flex', flexDirection: 'column', gap: 8, fontSize: 11, color: '#888' }}>
          {/* L0: 事件类型 */}
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 6 }}>
            <span>类型: <b style={{ color: '#ccc' }}>{eventLabel}</b></span>
            <span>强度: {entry.l0.intensity}</span>
            <span>真诚: {entry.l0.sincerity?.toFixed(1) ?? '-'}</span>
          </div>

          {/* L1: 关系 — 四列网格 */}
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr 1fr', gap: 6 }}>
            <span style={{ color: tColor }}>信任: {Math.round(entry.l1.trust)}</span>
            <span>裂痕: {entry.l1.rifts}</span>
            <span>阶段: {stageLabel}</span>
            <span>气氛: {entry.l1.atmosphere || 'neutral'}</span>
          </div>

          {/* L2: 情绪 — 四列网格 + MiniBar */}
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr 1fr', gap: 4 }}>
            {[
              { label: 'aff', value: entry.l2.aff, color: '#818cf8' },
              { label: 'sec', value: entry.l2.sec, color: '#4ade80' },
              { label: 'aro', value: entry.l2.aro, color: '#f59e0b' },
              { label: 'dom', value: entry.l2.dom, color: '#f472b6' },
            ].map(d => (
              <div key={d.label}>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 2 }}>
                  <span>{d.label}</span><span>{d.value.toFixed(0)}</span>
                </div>
                <MiniBar2 value={d.value} color={d.color} max={100} />
              </div>
            ))}
          </div>
          <div style={{ color: '#aaa' }}>情绪: <b style={{ color: '#fbbf24' }}>{entry.l2.label}</b></div>

          {/* L3 */}
          <div style={{ display: 'flex', gap: 12 }}>
            <span>TierB: {entry.l3.tierBChars}字</span>
            <span>事实: {entry.l3.factsUsed || 0}</span>
            <span>命中: {entry.l3.embeddingHits || 0}</span>
            {entry.l3.silent && <span style={{ color: '#f59e0b' }}>🤫 沉默</span>}
          </div>

          {/* L4/L5 */}
          {(entry.l4?.wrote || entry.l5?.toolCalls?.length) && (
            <div style={{ display: 'flex', gap: 12 }}>
              {entry.l4?.wrote && <span style={{ color: '#4ade80' }}>✍️ 已写入</span>}
              {entry.l5?.toolCalls?.map((tc, i) => (
                <span key={i} style={{ color: '#818cf8' }}>🔧 {tc}</span>
              ))}
            </div>
          )}

          {/* 耗时 */}
          {entry.ms?.total != null && (
            <span style={{ color: '#666', fontSize: 10 }}>⏱ {entry.ms.total}ms</span>
          )}
        </div>
      )}
    </div>
  )
}

export default function WhisperTracePanel({ traces }: Props) {
  const [expandedTurn, setExpandedTurn] = useState<number | null>(null)
  const [count, setCount] = useState(20)

  if (traces.length === 0) {
    return (
      <div style={{ color: '#777', fontSize: 12, padding: 16, textAlign: 'center' }}>
        暂无追踪记录 — 开始对话后将显示每轮的心理变化
      </div>
    )
  }

  const recent = traces.slice(-count).reverse()

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{
        display: 'flex', alignItems: 'center', gap: 8, padding: '8px 12px',
        borderBottom: '1px solid rgba(255,255,255,0.08)', fontSize: 12, color: '#999',
      }}>
        <span>📊 对话追踪</span>
        <select
          value={count}
          onChange={e => setCount(Number(e.target.value))}
          style={{
            marginLeft: 'auto', fontSize: 11, background: 'rgba(255,255,255,0.06)',
            border: '1px solid rgba(255,255,255,0.1)', borderRadius: 4,
            color: '#aaa', padding: '2px 6px',
          }}
        >
          {[10, 20, 50, 100].map(n => <option key={n} value={n}>最近 {n} 轮</option>)}
        </select>
        <span style={{ color: '#666' }}>共{traces.length}轮</span>
      </div>
      <div style={{ flex: 1, overflow: 'auto' }}>
        {recent.map(entry => (
          <TraceRow
            key={entry.turn}
            entry={entry}
            expanded={expandedTurn === entry.turn}
            onToggle={() => setExpandedTurn(expandedTurn === entry.turn ? null : entry.turn)}
          />
        ))}
      </div>
    </div>
  )
}
