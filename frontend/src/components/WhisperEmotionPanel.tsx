import React, { useEffect, useRef } from 'react'
import { Typography } from 'antd'
import { EmotionStarMap } from './EmotionStarMap'
import { C } from '../utils/theme'

// ─── LightCore — 菱形生命信号（对齐 ackem 呼吸+双闪动画） ────

const LightCore: React.FC<{ trust: number }> = ({ trust }) => {
  const ref = useRef<HTMLDivElement>(null)
  const active = trust > 70

  // 双闪动画
  useEffect(() => {
    if (!active || !ref.current) return
    const el = ref.current
    const trigger = () => {
      el.style.transition = 'none'
      el.style.opacity = '1'; el.style.transform = 'rotate(45deg) scale(1.3)'
      el.style.boxShadow = '0 0 16px var(--whisper-accent, #e85388), 0 0 32px var(--whisper-accent, #e85388)'
      requestAnimationFrame(() => {
        el.style.transition = 'all 400ms ease-out'
        el.style.opacity = '0.4'; el.style.transform = 'rotate(45deg) scale(1)'
        el.style.boxShadow = '0 0 8px var(--whisper-accent, #e85388), 0 0 16px var(--whisper-accent, #e85388)'
      })
    }
    const id = window.setInterval(() => { if (Math.random() < 0.08) trigger() }, 30000)
    return () => clearInterval(id)
  }, [active])

  return (
    <div
      ref={ref}
      style={{
        width: 8, height: 8, transform: 'rotate(45deg)',
        background: active ? 'var(--whisper-accent, #e85388)' : C('color-text-secondary'),
        boxShadow: active ? '0 0 8px var(--whisper-accent, #e85388), 0 0 16px var(--whisper-accent, #e85388)' : 'none',
        transition: 'all 0.5s',
        flexShrink: 0, margin: '0 4px',
        animation: active ? 'whisperCoreBreathe 4s ease-in-out infinite' : 'none',
        cursor: 'default',
      }}
      title={active ? '生命信号很强' : '生命信号'}
      onMouseEnter={e => { e.currentTarget.style.width = '12px'; e.currentTarget.style.height = '12px'; e.currentTarget.style.opacity = '0.85' }}
      onMouseLeave={e => { e.currentTarget.style.width = '8px'; e.currentTarget.style.height = '8px'; e.currentTarget.style.opacity = '1' }}
    />
  )
}

// ─── TrustGlowBar — 信任光柱（对齐 ackem 6px 完全圆角） ──────

const TrustGlowBar: React.FC<{ trust: number; rifts: number; stage: string }> = ({ trust, rifts, stage }) => {
  const pct = Math.min(100, Math.max(0, trust))
  const stageZh = stage === 'INTIMATE' ? '亲密' : stage === 'FAMILIAR' ? '熟悉' : '初识'
  return (
    <div style={{ marginTop: 12 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12, color: C('color-text-secondary'), marginBottom: 4 }}>
        <span>信任</span>
        <span style={{ color: 'var(--whisper-accent, #e85388)', fontWeight: 600, fontSize: 12 }}>{trust.toFixed(0)}</span>
      </div>
      <div style={{ height: 6, borderRadius: 999, background: C('color-bg-elevated'), overflow: 'hidden' }}>
        <div style={{
          height: '100%', width: `${pct}%`, borderRadius: 999,
          background: 'var(--whisper-accent, #e85388)',
          boxShadow: '0 0 6px var(--whisper-accent, #e85388)',
          transition: 'width 700ms cubic-bezier(0.4,0,0.2,1)',
        }} />
      </div>
      <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 11, color: C('color-text-secondary'), marginTop: 4 }}>
        <span>阶段 · {stageZh}</span>
        <span>裂痕 {rifts}</span>
      </div>
    </div>
  )
}

// ─── TISOR 五维条（对齐 ackem 6px 高度） ─────────────────────

const TisorBars: React.FC<{ T: number; I: number; S: number; O: number; R: number }> = ({ T, I, S, O, R }) => {
  const dims = [
    { label: '温柔', val: T },
    { label: '主动', val: I },
    { label: '敏感', val: S },
    { label: '开放', val: O },
    { label: '理性', val: R },
  ]
  return (
    <div style={{ marginTop: 12 }}>
      <Typography.Text style={{ fontSize: 10, color: C('color-text-secondary'), textTransform: 'uppercase', letterSpacing: 1, fontWeight: 600 }}>人格 TISOR</Typography.Text>
      <div style={{ marginTop: 6 }}>
        {dims.map(d => (
          <div key={d.label} style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 5 }}>
            <span style={{ width: 32, fontSize: 11, color: C('color-text-secondary') }}>{d.label}</span>
            <div style={{ flex: 1, height: 6, borderRadius: 999, background: C('color-bg-elevated'), overflow: 'hidden' }}>
              <div style={{ height: '100%', width: `${d.val}%`, borderRadius: 999, background: 'var(--whisper-accent, #e85388)', opacity: 0.7, transition: 'width 700ms cubic-bezier(0.4,0,0.2,1)' }} />
            </div>
            <span style={{ width: 32, textAlign: 'right', fontSize: 11, color: C('color-text-secondary'), fontVariantNumeric: 'tabular-nums' }}>{d.val.toFixed(0)}</span>
          </div>
        ))}
      </div>
    </div>
  )
}

// ─── 轻语情感面板 ─────────────────────────────────────────────

export interface EmotionPanelProps {
  emotion: string; stage: string
  trust: number; rifts: number
  aff: number; sec: number; aro: number; dom: number
  T: number; I: number; S: number; O: number; R: number
  totalTurns: number; personalityLabel: string
}

const labelZH: Record<string, string> = {
  SWEET_ATTACHMENT: '甜蜜依恋', SHY_HEARTBEAT: '害羞心动', TSUNDERE: '傲娇',
  HURT_GRIEVANCE: '委屈受伤', ANGRY_ATTACK: '愤怒反击', COLD_DETACHED: '冷淡疏离',
  FEARFUL_OBEDIENT: '不安顺从', QUIET_FOND: '安静的喜欢', CALM_RATIONAL: '平静理性',
}

const emotionColorMap: Record<string, string> = {
  SWEET_ATTACHMENT: '#f472b6', SHY_HEARTBEAT: '#fb7185', TSUNDERE: '#f59e0b',
  HURT_GRIEVANCE: '#a78bfa', ANGRY_ATTACK: '#ef4444', COLD_DETACHED: '#94a3b8',
  FEARFUL_OBEDIENT: '#c084fc', QUIET_FOND: '#fbbf24', CALM_RATIONAL: '#60a5fa',
}

export const WhisperEmotionPanel: React.FC<EmotionPanelProps> = (props) => {
  const { emotion, stage, trust, rifts, aff, sec, aro, dom, T, I, S, O, R, totalTurns, personalityLabel } = props
  const hasData = emotion !== ''
  const moodHint = aff > 20 ? '心情很好' : aff < -15 ? '有些低落' : '气氛平稳'
  const emoColor = emotionColorMap[emotion] || '#60a5fa'

  // 用户六维（对齐 ackem — 当前使用模拟数据，后续接后端 API）
  const userSixDims = hasData ? {
    E: Math.round(40 + aff * 0.3 + sec * 0.1),
    A: Math.round(50 + aff * 0.4),
    D: Math.round(45 + dom * 0.2),
    P: Math.round(35 + dom * 0.3),
    N: Math.round(55 + sec * 0.2 + aff * 0.1),
    O: Math.round(50 + aro * 0.2),
  } : null

  const dimLabels: Record<string, string> = { E: '表达欲', A: '依恋', D: '直接', P: '权力', N: '情感', O: '开放' }

  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 12, height: '100%', overflow: 'auto' }}>
      <style>{`
        @keyframes whisperCoreBreathe {
          0%, 100% { opacity: 0.4; transform: rotate(45deg) scale(1); }
          50% { opacity: 1; transform: rotate(45deg) scale(1.08); }
        }
      `}</style>
      <div style={{ '--whisper-accent': emoColor } as React.CSSProperties} />

      {/* 头部 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <LightCore trust={hasData ? trust : 0} />
          <Typography.Text strong style={{ fontSize: 13, color: C('color-text'), fontFamily: 'inherit' }}>
            {hasData ? (labelZH[emotion] || emotion) : '等待连接'}
          </Typography.Text>
        </div>
      </div>

      {/* 星图 */}
      <div style={{ width: '100%', maxWidth: 200, margin: '0 auto' }}>
        <EmotionStarMap
          aff={hasData ? aff : 0} sec={hasData ? sec : 0}
          aro={hasData ? aro : 0} dom={hasData ? dom : 0}
          primaryLabel={hasData ? emotion : 'CALM_RATIONAL'} size={200}
        />
      </div>

      {/* 信任光柱 */}
      <TrustGlowBar trust={hasData ? trust : 50} rifts={hasData ? rifts : 0} stage={hasData ? stage : 'STRANGER'} />

      {/* 状态摘要 */}
      <Typography.Text style={{ fontSize: 11, color: C('color-text-secondary'), textAlign: 'center' }}>
        {hasData ? `聊了 ${totalTurns} 轮 · ${moodHint}` : '开始对话后显示实时状态'}
      </Typography.Text>

      {/* TISOR */}
      <div style={{ borderTop: `1px solid ${C('color-border')}`, paddingTop: 10 }}>
        <TisorBars T={T} I={I} S={S} O={O} R={R} />
      </div>

      {/* 用户六维（对齐 ackem EmotionPanel:243-263） */}
      {userSixDims && (
        <div style={{ borderTop: `1px solid ${C('color-border')}`, paddingTop: 10 }}>
          <Typography.Text style={{ fontSize: 10, color: C('color-text-secondary'), textTransform: 'uppercase', letterSpacing: 1, fontWeight: 600 }}>
            主人六维
          </Typography.Text>
          <div style={{ marginTop: 6 }}>
            {(Object.keys(dimLabels) as Array<keyof typeof dimLabels>).map(dim => (
              <div key={dim} style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 5 }}>
                <span style={{ width: 32, fontSize: 11, color: C('color-text-secondary') }}>{dimLabels[dim]}</span>
                <div style={{ flex: 1, height: 6, borderRadius: 999, background: C('color-bg-elevated'), overflow: 'hidden' }}>
                  <div style={{
                    height: '100%', width: `${Math.min(100, Math.max(0, (userSixDims as any)[dim]))}%`,
                    borderRadius: 999, background: '#60a5fa', opacity: 0.6,
                    transition: 'width 700ms cubic-bezier(0.4,0,0.2,1)',
                  }} />
                </div>
                <span style={{ width: 28, textAlign: 'right', fontSize: 11, color: C('color-text-secondary') }}>{(userSixDims as any)[dim]}</span>
              </div>
            ))}
          </div>
          <Typography.Text style={{ fontSize: 9, color: C('color-text-secondary'), opacity: 0.4 }}>
            来源：对话推断
          </Typography.Text>
        </div>
      )}

      {/* 人格标签 */}
      <div style={{ textAlign: 'center', marginTop: 'auto', paddingTop: 8, borderTop: `1px solid ${C('color-border')}` }}>
        <Typography.Text style={{ fontSize: 10, color: C('color-text-secondary'), opacity: 0.5 }}>
          {personalityLabel}型伴侣
        </Typography.Text>
      </div>
    </div>
  )
}
