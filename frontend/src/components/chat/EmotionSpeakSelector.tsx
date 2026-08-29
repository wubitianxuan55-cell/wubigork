// EmotionSpeakSelector.tsx — 朗读情绪选择（v4.3d 前端收尾）
//
// 轻量 Popover（对齐 PersonaPicker 交互形态）：选择朗读时携带的情绪标签，
// 并显示当前生效情绪（手动选择优先，其次会话最近一轮情绪，均无则「自动」）。
// 情绪清单/中文名/色板见 pages/chat/emotions.ts（与后端 EmotionVoiceMap 一致）。
import React, { useState } from 'react'
import { Popover, Tooltip } from 'antd'
import { SmileOutlined, CheckOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'
import { SPEAK_EMOTIONS, SPEAK_EMOTION_AUTO, emotionLabel, emotionColor } from '../../pages/chat/emotions'

export interface EmotionSpeakSelectorProps {
  /** 手动选择的情绪标签（'' = 跟随会话自动） */
  value: string
  /** 当前会话最近一轮情绪（whisper L2 标签，可能为空） */
  sessionEmotion: string
  onChange: (emotion: string) => void
}

export const EmotionSpeakSelector: React.FC<EmotionSpeakSelectorProps> = ({ value, sessionEmotion, onChange }) => {
  const [open, setOpen] = useState(false)
  // 生效情绪 = 手动选择优先，其次会话最近一轮情绪
  const effective = value || sessionEmotion || SPEAK_EMOTION_AUTO
  const effectiveLabel = effective ? emotionLabel(effective) || effective : ''
  const effectiveColor = effective ? emotionColor(effective) : undefined

  const pick = (emotion: string) => { onChange(emotion); setOpen(false) }

  const rowStyle: React.CSSProperties = {
    display: 'flex', alignItems: 'center', gap: 8, width: '100%', padding: '6px 8px',
    borderRadius: 8, cursor: 'pointer', background: 'transparent', border: 'none',
    fontSize: 12.5, color: C('color-text'), textAlign: 'left',
  }
  const activeStyle: React.CSSProperties = { ...rowStyle, background: 'rgba(244,114,182,0.12)', fontWeight: 600 }

  const content = (
    <div style={{ width: 216 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 6 }}>
        <span style={{ fontSize: 12, fontWeight: 600, color: C('color-text') }}>朗读情绪</span>
        {sessionEmotion && (
          <span style={{ fontSize: 10.5, color: C('color-text-secondary') }}>
            会话情绪：{emotionLabel(sessionEmotion) || sessionEmotion}
          </span>
        )}
      </div>
      <div role="button" tabIndex={0}
        onClick={() => pick(SPEAK_EMOTION_AUTO)}
        onKeyDown={e => { if (e.key === 'Enter') pick(SPEAK_EMOTION_AUTO) }}
        style={value === SPEAK_EMOTION_AUTO ? activeStyle : rowStyle}>
        <span>跟随会话自动</span>
        {value === SPEAK_EMOTION_AUTO && <CheckOutlined style={{ marginLeft: 'auto', color: '#f472b6', fontSize: 11 }} />} {/* hex-exempt 品牌识别色（轻语玫红，选中态） */}
      </div>
      {SPEAK_EMOTIONS.map(e => {
        const active = effective === e.value
        return (
          <div key={e.value} role="button" tabIndex={0}
            onClick={() => pick(e.value)}
            onKeyDown={ev => { if (ev.key === 'Enter') pick(e.value) }}
            style={active ? activeStyle : rowStyle}>
            <span style={{ width: 8, height: 8, borderRadius: '50%', background: e.color, display: 'inline-block', flexShrink: 0 }} />
            <span>{e.label}</span>
            {active && <CheckOutlined style={{ marginLeft: 'auto', color: '#f472b6', fontSize: 11 }} />} {/* hex-exempt 品牌识别色（轻语玫红，选中态） */}
          </div>
        )
      })}
      <div style={{ marginTop: 6, paddingTop: 6, borderTop: '1px solid var(--border-subtle)', fontSize: 10.5, color: C('color-text-secondary') }}>
        朗读回复时携带所选情绪参数
      </div>
    </div>
  )

  return (
    <Popover open={open} onOpenChange={setOpen} trigger="click" placement="bottomRight"
      content={content} arrow={false} styles={{ body: { padding: 10 } }}>
      <Tooltip title={effectiveLabel ? `朗读情绪：${effectiveLabel}` : '朗读情绪：跟随会话'}>
        <button
          type="button"
          aria-label="朗读情绪"
          style={{
            display: 'inline-flex', alignItems: 'center', gap: 4, height: 24, padding: '0 6px',
            background: 'transparent', border: 'none', borderRadius: 6, cursor: 'pointer',
            fontSize: 11, color: C('color-text-secondary'),
          }}>
          <SmileOutlined style={{ fontSize: 12 }} />
          {effectiveLabel ? (
            <>
              <span style={{ width: 8, height: 8, borderRadius: '50%', background: effectiveColor, display: 'inline-block' }} />
              {effectiveLabel}
            </>
          ) : (
            <span>自动</span>
          )}
        </button>
      </Tooltip>
    </Popover>
  )
}

export default EmotionSpeakSelector
