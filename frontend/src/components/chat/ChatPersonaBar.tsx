// ChatPage 拆分产物：人格状态条（临场感：头像常驻 + 名字；状态/记忆归角色库，行为零变化，T6-10.1）
import React from 'react'
import { Button, Typography } from 'antd'
import { SwapOutlined } from '@ant-design/icons'
import { C } from '../../utils/theme'
import { CompanionAvatar } from '../CompanionAvatar'
import PersonaPicker from '../PersonaPicker'

export interface ChatPersonaBarProps {
  companionName: string
  emoColor: string
  speaking: boolean
  thinking: boolean
  activeId: string
  onSelect: (id: string) => void
  onManage: () => void
}

export const ChatPersonaBar: React.FC<ChatPersonaBarProps> = ({
  companionName, emoColor, speaking, thinking, activeId, onSelect, onManage,
}) => (
  <div className="chat-persona-bar">
    <CompanionAvatar size={46} state={speaking ? 'speaking' : thinking ? 'thinking' : 'idle'} emotionColor={emoColor} />
    <div style={{ flex: 1, minWidth: 0 }}>
      <div className="chat-persona-meta">
        <Typography.Text strong style={{ fontSize: 14, color: C('color-text') }}>{companionName}</Typography.Text>
        <span className="chat-chip" style={{ color: 'var(--gaea-glow)', borderColor: 'color-mix(in srgb, var(--gaea-glow) 30%, transparent)' }}>
          <span style={{ width: 6, height: 6, borderRadius: '50%', background: 'var(--gaea-glow)' }} />
          AI 陪伴
        </span>
      </div>
    </div>
    <PersonaPicker activeId={activeId}
      onSelect={onSelect} onManage={onManage}>
      <Button type="primary" size="small" icon={<SwapOutlined />}
        style={{ borderRadius: 16, height: 30, fontSize: 12, flexShrink: 0 }}>
        选择角色
      </Button>
    </PersonaPicker>
  </div>
)
