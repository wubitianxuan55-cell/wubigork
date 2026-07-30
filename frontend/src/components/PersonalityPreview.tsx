// PersonalityPreview.tsx — 人格预览面板 (Liquid Glass 风格)
// 点击卡片后展示，含雷达图 + 详情 + 确认/取消
import React from 'react'
import { Button, Tag, Typography } from 'antd'
import { CheckOutlined, CloseOutlined, ThunderboltOutlined, LockOutlined } from '@ant-design/icons'
import TisorRadar from './TisorRadar'

const { Text } = Typography

export interface PersonalityPreset {
  id: string; label: string; gender: string; tags?: string[]
  dims: { T: number; I: number; S: number; O: number; R: number }
  voiceGuide?: string; requiresAdult18?: boolean
  hiddenPersona?: { T: number; I: number; S: number; O: number; R: number }
}

interface Props {
  personality: PersonalityPreset
  isActive: boolean
  adultMode: boolean
  onConfirm: () => void
  onBack: () => void
}

const FULL_LABELS: Record<string, string> = {
  T: 'Tender 温顺', I: 'Independent 独立', S: 'Sensitive 感性',
  O: 'Open 开放', R: 'Rational 理性',
}
const DIM_COLORS: Record<string, string> = {
  T: '#f472b6', I: '#fb923c', S: '#4ade80', O: '#a78bfa', R: '#60a5fa',
}
const KEYS = ['T', 'I', 'S', 'O', 'R'] as const
const TAG_COLORS: Record<string, string> = {
  maternal: 'magenta', nurturing: 'magenta', bratty: 'orange',
  'provoke-submit': 'volcano', 'dual-persona': 'purple', paternal: 'blue',
}

export default function PersonalityPreview({ personality, isActive, adultMode, onConfirm, onBack }: Props) {
  const p = personality
  const locked = Boolean(p.requiresAdult18 && !adultMode)
  const hasHidden = p.tags?.includes('dual-persona')

  return (
    <div style={{
      display: 'flex', flexDirection: 'row', gap: 32,
      padding: 20, animation: 'previewSlideIn 300ms cubic-bezier(0.16,1,0.3,1)',
    }}>
      {/* 左侧 — 雷达图 */}
      <div style={{
        flex: '0 0 auto', display: 'flex', flexDirection: 'column',
        alignItems: 'center', gap: 12,
      }}>
        <div style={{
          padding: 20, borderRadius: 20,
          background: 'rgba(255,255,255,0.03)', border: '1px solid rgba(255,255,255,0.06)',
          backdropFilter: 'blur(20px)', boxShadow: '0 8px 32px rgba(0,0,0,0.3)',
        }}>
          <TisorRadar dims={p.dims} size={180} color="#e85388" showLabels />
        </div>

        {/* 数值详情 */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 4, width: '100%', maxWidth: 180 }}>
          {KEYS.map(k => (
            <div key={k} style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span style={{ width: 10, height: 10, borderRadius: 3, background: DIM_COLORS[k], flexShrink: 0 }} />
              <Text style={{ fontSize: 10, color: 'rgba(255,255,255,0.5)', flex: 1, whiteSpace: 'nowrap' }}>
                {FULL_LABELS[k]}
              </Text>
              <Text strong style={{ fontSize: 11, color: DIM_COLORS[k], minWidth: 24, textAlign: 'right' }}>
                {p.dims[k]}
              </Text>
            </div>
          ))}
        </div>
      </div>

      {/* 右侧 — 详情 */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 14, justifyContent: 'center' }}>
        {/* 头部 */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
          <span style={{ fontSize: 24 }}>
            {p.gender === 'male' ? '🤵' : '👩'}
          </span>
          <div>
            <div style={{ fontSize: 22, fontWeight: 700, color: '#fff', lineHeight: 1.2 }}>
              {p.label}
            </div>
            {p.voiceGuide && (
              <Text style={{ fontSize: 12, color: 'rgba(255,255,255,0.45)', fontStyle: 'italic' }}>
                {p.voiceGuide}
              </Text>
            )}
          </div>
          {hasHidden && (
            <Tag icon={<ThunderboltOutlined />} color="purple" style={{ fontSize: 10, marginLeft: 'auto' }}>
              Dual Persona
            </Tag>
          )}
        </div>

        {/* 标签 */}
        {p.tags && p.tags.length > 0 && (
          <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
            {p.tags.map(t => (
              <Tag key={t} color={TAG_COLORS[t] || 'default'} style={{ fontSize: 10, margin: 0 }}>
                {t === 'dual-persona' ? '🎭 dual' : t === 'maternal' ? '🤱 maternal' : t === 'bratty' ? '😈 bratty' : t}
              </Tag>
            ))}
          </div>
        )}

        {/* 隐藏人格对比 */}
        {p.hiddenPersona && (
          <div style={{
            padding: 12, borderRadius: 12,
            background: 'rgba(168,85,247,0.06)', border: '1px solid rgba(168,85,247,0.15)',
          }}>
            <Text style={{ fontSize: 10, color: '#a78bfa', fontWeight: 600, display: 'block', marginBottom: 8 }}>
              🎭 Hidden Persona Contrast
            </Text>
            <div style={{ display: 'flex', gap: 24, alignItems: 'center' }}>
              <div style={{ textAlign: 'center' }}>
                <Text style={{ fontSize: 9, color: 'rgba(255,255,255,0.3)' }}>Surface</Text>
                <div style={{ fontSize: 10, color: 'rgba(255,255,255,0.6)', marginTop: 2 }}>
                  T{p.dims.T} I{p.dims.I} S{p.dims.S} O{p.dims.O} R{p.dims.R}
                </div>
              </div>
              <span style={{ color: '#a78bfa', fontSize: 14 }}>→</span>
              <div style={{ textAlign: 'center' }}>
                <Text style={{ fontSize: 9, color: '#a78bfa' }}>Hidden</Text>
                <div style={{ fontSize: 10, color: '#a78bfa', fontWeight: 600, marginTop: 2 }}>
                  T{p.hiddenPersona.T} I{p.hiddenPersona.I} S{p.hiddenPersona.S} O{p.hiddenPersona.O} R{p.hiddenPersona.R}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* 成人锁定提示 */}
        {locked && (
          <div style={{
            padding: 10, borderRadius: 10, display: 'flex', alignItems: 'center', gap: 8,
            background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.08)',
          }}>
            <LockOutlined style={{ color: '#f59e0b', fontSize: 16 }} />
            <Text style={{ fontSize: 11, color: '#f59e0b' }}>
              This personality requires Adult Mode to be enabled in settings.
            </Text>
          </div>
        )}

        {/* 当前激活提示 */}
        {isActive && (
          <div style={{
            padding: 10, borderRadius: 10, display: 'flex', alignItems: 'center', gap: 8,
            background: 'rgba(232,83,136,0.08)', border: '1px solid rgba(232,83,136,0.2)',
          }}>
            <CheckOutlined style={{ color: '#e85388', fontSize: 14 }} />
            <Text style={{ fontSize: 12, color: '#e85388', fontWeight: 500 }}>
              Currently active — select another to switch
            </Text>
          </div>
        )}

        {/* 操作按钮 */}
        <div style={{ display: 'flex', gap: 10, marginTop: 8 }}>
          <Button onClick={onBack} icon={<CloseOutlined />}
            style={{
              background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.08)',
              color: 'rgba(255,255,255,0.6)', borderRadius: 10, height: 38,
            }}>
            Back
          </Button>
          {!isActive && (
            <Button type="primary" onClick={onConfirm} icon={<CheckOutlined />}
              disabled={locked}
              style={{
                background: locked ? undefined : 'linear-gradient(135deg, #e85388, #c02660)',
                border: 'none', borderRadius: 10, height: 38, fontWeight: 600,
                boxShadow: locked ? undefined : '0 4px 16px rgba(232,83,136,0.3)',
              }}>
              {locked ? 'Locked' : `Switch to ${p.label}`}
            </Button>
          )}
          {isActive && (
            <Button type="primary" onClick={onBack} icon={<CheckOutlined />}
              style={{
                background: 'linear-gradient(135deg, #e85388, #c02660)',
                border: 'none', borderRadius: 10, height: 38, fontWeight: 600,
                boxShadow: '0 4px 16px rgba(232,83,136,0.3)', opacity: 0.6,
              }}>
              Already Active
            </Button>
          )}
        </div>
      </div>

      {/* 入场动画 */}
      <style>{`
        @keyframes previewSlideIn {
          from { opacity: 0; transform: translateY(12px) scale(0.98); }
          to   { opacity: 1; transform: translateY(0) scale(1); }
        }
      `}</style>
    </div>
  )
}
