// WhisperPersonalityModal.tsx — 轻语人格选择弹窗 v3
// Liquid Glass 风格 · 环形雷达图 · 预览+确认交互
import React, { useState, useMemo } from 'react'
import { Modal, Card, Tag, Tooltip, Typography, message } from 'antd'
import { CheckCircleFilled, LockOutlined, ThunderboltOutlined, UserOutlined, TeamOutlined, FireOutlined, ExperimentOutlined } from '@ant-design/icons'
import TisorRadar from './TisorRadar'
import PersonalityPreview from './PersonalityPreview'

const { Text } = Typography

export interface PersonalityPreset {
  id: string; label: string; gender: string; tags?: string[]
  dims: { T: number; I: number; S: number; O: number; R: number }
  voiceGuide?: string; requiresAdult18?: boolean
  hiddenPersona?: { T: number; I: number; S: number; O: number; R: number }
}

interface Props {
  open: boolean
  personalities: PersonalityPreset[]
  activePersonality: string
  adultMode: boolean
  onClose: () => void
  onSwitch: (id: string) => void
}

// ─── 常量 ────────────────────────────────────────────────

const DIM_LABELS: Record<string, string> = { T: 'Tender', I: 'Active', S: 'Sub', O: 'Unique', R: 'Reserved' }
const DIM_COLORS: Record<string, string> = { T: '#f472b6', I: '#fb923c', S: '#4ade80', O: '#a78bfa', R: '#60a5fa' }
const KEYS = ['T', 'I', 'S', 'O', 'R'] as const

const GROUPS = [
  { key: 'core', label: 'Core', icon: <ThunderboltOutlined />, ids: ['gaea'] },
  { key: 'female', label: 'Female', icon: <UserOutlined />, ids: ['tsundere','yandere','oneesan','genki','kuudere','deredere','shitakiri','bokke','ice_queen','girl_next_door'] },
  { key: 'male', label: 'Male', icon: <TeamOutlined />, ids: ['ceo_dom','gentle_warmth','puppy','iceberg','schemer','loyal_knight','bad_boy','artistic','innocent_boy','boy_next_door'] },
  { key: 'ds', label: 'D/s', icon: <FireOutlined />, ids: ['submissive','dominatrix','loyal_pup','tamer'] },
  { key: 'special', label: 'Special', icon: <ExperimentOutlined />, ids: ['mommy','mesugaki','gap_moe_f','daddy','gap_moe_m'] },
]

const TAG_COLORS: Record<string, string> = {
  maternal: 'magenta', nurturing: 'magenta', bratty: 'orange',
  'provoke-submit': 'volcano', 'dual-persona': 'purple', paternal: 'blue',
}

const TAG_ICONS: Record<string, string> = {
  maternal: '♡', nurturing: '♡', bratty: '◇', 'dual-persona': '⬡',
  'provoke-submit': '◆', paternal: '♢',
}

// ─── 微缩雷达图（卡片内） ──────────────────────────────────

function MiniRadar({ dims, color = '#e85388' }: { dims: PersonalityPreset['dims']; color?: string }) {
  return <TisorRadar dims={dims} size={56} color={color} showLabels={false} />
}

// ─── 卡片组件 ──────────────────────────────────────────────

function PersonalityCard({ p, isActive, locked, hasHidden, onClick }: {
  p: PersonalityPreset; isActive: boolean; locked: boolean; hasHidden: boolean
  onClick: () => void
}) {
  const [hovered, setHovered] = useState(false)

  return (
    <div
      onClick={() => { if (locked) { message.warning('Requires adult mode'); return }; onClick() }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        position: 'relative', overflow: 'hidden',
        cursor: locked ? 'not-allowed' : 'pointer',
        opacity: locked ? 0.4 : 1,
        borderRadius: 14,
        background: isActive
          ? 'linear-gradient(135deg, rgba(232,83,136,0.12), rgba(168,85,247,0.08))'
          : 'rgba(255,255,255,0.03)',
        border: isActive
          ? '1.5px solid rgba(232,83,136,0.35)'
          : '1px solid rgba(255,255,255,0.06)',
        boxShadow: isActive
          ? '0 0 20px rgba(232,83,136,0.1), inset 0 1px 0 rgba(255,255,255,0.04)'
          : hovered ? '0 4px 20px rgba(0,0,0,0.2), inset 0 1px 0 rgba(255,255,255,0.03)' : 'none',
        backdropFilter: 'blur(12px)',
        transition: 'all 250ms cubic-bezier(0.16,1,0.3,1)',
        transform: hovered && !locked ? 'translateY(-2px) scale(1.01)' : 'none',
        padding: '12px 14px',
        display: 'flex', gap: 12, alignItems: 'center',
      }}
    >
      {/* 虹彩光泽层 */}
      {hovered && !locked && !isActive && (
        <div style={{
          position: 'absolute', inset: 0, borderRadius: 14, pointerEvents: 'none',
          background: 'linear-gradient(135deg, rgba(232,83,136,0.06), rgba(168,85,247,0.04), rgba(96,165,250,0.03))',
          transition: 'opacity 300ms ease',
        }} />
      )}

      {/* 左侧 — 微型雷达图 */}
      <div style={{ flexShrink: 0 }}>
        <MiniRadar dims={p.dims} color={isActive ? '#e85388' : '#888'} />
      </div>

      {/* 右侧 — 信息 */}
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4 }}>
          <Text strong style={{
            fontSize: 13, color: isActive ? '#e85388' : 'rgba(255,255,255,0.85)',
            transition: 'color 200ms',
          }}>
            {p.label}
          </Text>
          {isActive && <CheckCircleFilled style={{ color: '#e85388', fontSize: 13 }} />}
          {locked && <LockOutlined style={{ color: 'rgba(255,255,255,0.3)', fontSize: 11 }} />}
          {hasHidden && (
            <Tooltip title="Has hidden persona">
              <ThunderboltOutlined style={{ color: '#a78bfa', fontSize: 11 }} />
            </Tooltip>
          )}
        </div>

        {/* 标签 */}
        {p.tags && p.tags.length > 0 && (
          <div style={{ display: 'flex', gap: 3, flexWrap: 'wrap' }}>
            {p.tags.map(t => (
              <span key={t} style={{
                fontSize: 9, padding: '1px 6px', borderRadius: 4,
                background: TAG_COLORS[t]
                  ? `rgba(${t === 'magenta' ? '235,47,150' : t === 'orange' ? '250,140,22' : t === 'volcano' ? '216,72,45' : t === 'purple' ? '114,46,209' : '37,99,235'}, 0.15)`
                  : 'rgba(255,255,255,0.06)',
                color: TAG_COLORS[t]
                  ? (t === 'magenta' ? '#f472b6' : t === 'orange' ? '#fb923c' : t === 'volcano' ? '#f87171' : t === 'purple' ? '#a78bfa' : '#60a5fa')
                  : 'rgba(255,255,255,0.4)',
                whiteSpace: 'nowrap',
              }}>
                {TAG_ICONS[t] || ''} {t}
              </span>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

// ═══════════════════════════════════════════════════════════
// 主组件
// ═══════════════════════════════════════════════════════════

export default function WhisperPersonalityModal({ open, personalities, activePersonality, adultMode, onClose, onSwitch }: Props) {
  const [previewId, setPreviewId] = useState<string | null>(null)

  // 关闭时重置预览状态
  const handleClose = () => {
    setPreviewId(null)
    onClose()
  }

  // 分组数据
  const grouped = useMemo(() =>
    GROUPS.map(g => ({
      ...g,
      items: g.ids.map(id => personalities.find(p => p.id === id)).filter(Boolean) as PersonalityPreset[],
    })).filter(g => g.items.length > 0),
    [personalities],
  )

  // 当前预览的人格
  const previewPersonality = previewId ? personalities.find(p => p.id === previewId) : null

  return (
    <Modal
      title={null}
      open={open}
      onCancel={handleClose}
      footer={null}
      width={860}
      centered
      styles={{
        body: {
          padding: 0,
          maxHeight: '72vh',
          overflow: 'auto',
          background: 'linear-gradient(180deg, #0d0d14 0%, #111119 100%)',
        },
      }}
      style={{
        background: 'linear-gradient(180deg, #0f0f18 0%, #13131e 100%)',
        border: '1px solid rgba(255,255,255,0.06)',
        borderRadius: 18,
        boxShadow: '0 24px 80px rgba(0,0,0,0.5), 0 0 0 1px rgba(255,255,255,0.04)',
        overflow: 'hidden',
      }}
    >
      {/* ─── 标题栏 ──────────────────────────────────────── */}
      <div style={{
        padding: '16px 24px 12px',
        borderBottom: '1px solid rgba(255,255,255,0.05)',
        background: 'rgba(255,255,255,0.015)',
        backdropFilter: 'blur(20px)',
        display: 'flex', alignItems: 'center', gap: 12,
      }}>
        <span style={{
          fontSize: 24, width: 40, height: 40, borderRadius: 12,
          background: 'linear-gradient(135deg, rgba(232,83,136,0.15), rgba(168,85,247,0.1))',
          border: '1px solid rgba(232,83,136,0.2)',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }}>
          <span style={{ fontSize: 16 }}>⚡</span>
        </span>
        <div>
          <div style={{ fontSize: 16, fontWeight: 700, color: '#fff', lineHeight: 1.3 }}>
            Select gaea Personality
          </div>
          <Text style={{ fontSize: 11, color: 'rgba(255,255,255,0.35)' }}>
            {personalities.length} personalities · TISOR five-dimensional system
          </Text>
        </div>
      </div>

      {/* ─── 内容区 ──────────────────────────────────────── */}
      <div style={{ padding: previewPersonality ? 0 : '20px 24px' }}>
        {previewPersonality ? (
          /* 预览视图 */
          <PersonalityPreview
            personality={previewPersonality}
            isActive={activePersonality === previewPersonality.id}
            adultMode={adultMode}
            onConfirm={() => {
              if (!previewPersonality.requiresAdult18 || adultMode) {
                onSwitch(previewPersonality.id)
                setPreviewId(null)
              }
            }}
            onBack={() => setPreviewId(null)}
          />
        ) : (
          /* 网格视图 */
          <>
            {grouped.map((group, gi) => (
              <div key={group.key} style={{ marginBottom: gi < grouped.length - 1 ? 24 : 0 }}>
                {/* 分组标题 */}
                <div style={{
                  display: 'flex', alignItems: 'center', gap: 8,
                  marginBottom: 10, padding: '6px 0',
                  borderBottom: '1px solid rgba(255,255,255,0.04)',
                }}>
                  <span style={{ color: 'rgba(255,255,255,0.35)', fontSize: 13 }}>
                    {group.icon}
                  </span>
                  <Text strong style={{ fontSize: 12, color: 'rgba(255,255,255,0.55)', letterSpacing: '0.5px', textTransform: 'uppercase' }}>
                    {group.label}
                  </Text>
                  <span style={{
                    fontSize: 10, padding: '1px 8px', borderRadius: 10,
                    background: 'rgba(255,255,255,0.04)', color: 'rgba(255,255,255,0.3)',
                    marginLeft: 4,
                  }}>
                    {group.items.length}
                  </span>
                </div>

                {/* 卡片网格 */}
                <div style={{
                  display: 'grid',
                  gridTemplateColumns: 'repeat(auto-fill, minmax(240px, 1fr))',
                  gap: 10,
                }}>
                  {group.items.map((p, ci) => {
                    const isActive = activePersonality === p.id
                    const locked = Boolean(p.requiresAdult18 && !adultMode)
                    const hasHidden = p.tags?.includes('dual-persona')
                    return (
                      <div key={p.id} style={{
                        animation: `fadeInStagger 350ms ease both`,
                        animationDelay: `${(gi * 4 + ci) * 40}ms`,
                      }}>
                        <PersonalityCard
                          p={p}
                          isActive={isActive}
                          locked={locked}
                          hasHidden={!!hasHidden}
                          onClick={() => setPreviewId(p.id)}
                        />
                      </div>
                    )
                  })}
                </div>
              </div>
            ))}
          </>
        )}
      </div>

      {/* ─── 页脚 TISOR 图例 ──────────────────────────────── */}
      {!previewPersonality && (
        <div style={{
          margin: '16px 24px 16px', padding: '10px 16px',
          borderRadius: 12,
          background: 'rgba(255,255,255,0.02)',
          border: '1px solid rgba(255,255,255,0.04)',
          display: 'flex', gap: 20, flexWrap: 'wrap', alignItems: 'center',
        }}>
          <Text style={{ fontSize: 10, color: 'rgba(255,255,255,0.3)', fontWeight: 600, letterSpacing: '1px' }}>
            TISOR
          </Text>
          {KEYS.map(k => (
            <span key={k} style={{ display: 'flex', alignItems: 'center', gap: 5 }}>
              <span style={{
                width: 7, height: 7, borderRadius: 2,
                background: DIM_COLORS[k], display: 'inline-block',
              }} />
              <Text style={{ fontSize: 10, color: 'rgba(255,255,255,0.35)' }}>
                {k} = {DIM_LABELS[k]}
              </Text>
            </span>
          ))}
        </div>
      )}

      {/* ─── 入场动画 CSS ────────────────────────────────── */}
      <style>{`
        @keyframes fadeInStagger {
          from { opacity: 0; transform: translateY(8px) scale(0.97); }
          to   { opacity: 1; transform: translateY(0) scale(1); }
        }
      `}</style>
    </Modal>
  )
}
