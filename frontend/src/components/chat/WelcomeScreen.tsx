// ChatPage 拆分产物：欢迎屏（普通/角色双形态，行为零变化，T6-10.1）
import React from 'react'
import { Button } from 'antd'
import {
  MessageOutlined, SearchOutlined, EditOutlined, BulbOutlined,
  TranslationOutlined, BookOutlined, HeartOutlined, GlobalOutlined,
  StarFilled, ThunderboltOutlined, SettingOutlined, SwapOutlined,
} from '@ant-design/icons'
import { C } from '../../utils/theme'
import { CompanionAvatar } from '../CompanionAvatar'
import VoiceChatOrb from '../VoiceChatOrb'
import PersonaPicker from '../PersonaPicker'
import { SuggestionCard } from './SuggestionCard'
import type { VoiceChatState } from '../../hooks/useVoiceChat'

const PLAIN_SUGGESTIONS = [
  { icon: <MessageOutlined />, label: '随便聊聊', desc: '和 AI 畅聊任何话题' },
  { icon: <SearchOutlined />, label: '帮我查资料', desc: '快速搜索和整理信息' },
  { icon: <EditOutlined />, label: '写篇文章', desc: '博客、报告、文案随时生成' },
  { icon: <BulbOutlined />, label: '头脑风暴', desc: '一起碰撞灵感火花' },
  { icon: <TranslationOutlined />, label: '翻译内容', desc: '多语言互译，保持原意' },
  { icon: <BookOutlined />, label: '解释概念', desc: '深入浅出地讲解知识点' },
]

const PERSONA_SUGGESTIONS = [
  { icon: <HeartOutlined />, label: '聊聊天', desc: '分享你的日常' },
  { icon: <SearchOutlined />, label: '倾诉心情', desc: '说说心里话' },
  { icon: <GlobalOutlined />, label: '上网查问', desc: '搜最新资讯' },
  { icon: <StarFilled />, label: '分享兴趣', desc: '聊聊你喜欢的东西' },
  { icon: <ThunderboltOutlined />, label: '晚安问候', desc: '睡前聊一会儿' },
]

export interface WelcomeScreenProps {
  mode: string
  personaLabel: string
  companionName: string
  voice: VoiceChatState
  emoColor: string
  activePersonality: string
  onSwitchPersonality: (id: string) => void
  onNavigateLib: () => void
  onFillInput: (label: string) => void
  onSuggestion: (label: string) => void
}

export const WelcomeScreen: React.FC<WelcomeScreenProps> = ({
  mode, personaLabel, companionName, voice, emoColor,
  activePersonality, onSwitchPersonality, onNavigateLib, onFillInput, onSuggestion,
}) => {
  if (mode !== 'plain') {
    return (
      <div className="chat-welcome">
        <div className="chat-welcome-frame" aria-hidden="true">
          <span className="chat-wel-corner chat-wel-tl" />
          <span className="chat-wel-corner chat-wel-tr" />
          <span className="chat-wel-corner chat-wel-bl" />
          <span className="chat-wel-corner chat-wel-br" />
        </div>

        <span className="chat-wel-kicker">// COMPANION · {personaLabel}</span>

        <div className="chat-wel-orb chat-wel-orb-sm">
          <span className="chat-wel-ring chat-wel-ring-1" aria-hidden="true" />
          <span className="chat-wel-ring chat-wel-ring-2" aria-hidden="true" />
          <CompanionAvatar
            size={146}
            state={voice.aiSpeaking ? 'speaking' : voice.listening ? 'listening' : 'idle'}
            emotionColor={emoColor}
          />
        </div>

        <h2>{companionName}</h2>
        <p>我是{personaLabel}，今天想聊点什么？</p>

        <div className="chat-wel-telemetry">
          <span className="chat-wel-dot" />
          BOND <b>ACTIVE</b>
          <span className="chat-wel-sep" />
          VOICE <b>{voice.listening ? 'LISTEN' : voice.aiSpeaking ? 'SPEAK' : 'STANDBY'}</b>
          <span className="chat-wel-sep" />
          INPUT <b>READY</b>
        </div>

        <PersonaPicker activeId={activePersonality}
          onSelect={onSwitchPersonality} onManage={onNavigateLib}>
          <Button type="primary" icon={<SwapOutlined />} style={{ marginTop: 14, marginBottom: 16, borderRadius: 20, padding: '4px 22px', height: 36, fontSize: 13 }}>
            选择角色
          </Button>
        </PersonaPicker>
        <div className="chat-suggestion-grid">
          {PERSONA_SUGGESTIONS.map(s => (
            <SuggestionCard key={s.label} s={s} onClick={() => onFillInput(s.label)} />
          ))}
        </div>
        <div style={{ marginTop: 10 }}>
          <Button type="link" size="small" icon={<SettingOutlined />} onClick={onNavigateLib}
            style={{ color: C('color-text-secondary'), fontSize: 11.5 }}>
            去角色库管理角色
          </Button>
        </div>
      </div>
    )
  }
  return (
    <div className="chat-welcome">
      <div className="chat-welcome-frame" aria-hidden="true">
        <span className="chat-wel-corner chat-wel-tl" />
        <span className="chat-wel-corner chat-wel-tr" />
        <span className="chat-wel-corner chat-wel-bl" />
        <span className="chat-wel-corner chat-wel-br" />
      </div>

      <span className="chat-wel-kicker">// GAEA CORE · 语音就绪</span>

      <div className="chat-wel-orb">
        <span className="chat-wel-ring chat-wel-ring-1" aria-hidden="true" />
        <span className="chat-wel-ring chat-wel-ring-2" aria-hidden="true" />
        <VoiceChatOrb
          volume={voice.volume}
          listening={voice.listening}
          speaking={voice.speaking}
          aiSpeaking={voice.aiSpeaking}
          transcript={voice.transcript}
          size={188}
        />
      </div>

      <h2>gaea AI</h2>
      <p>你的智能 AI 助手：聊天、写作、翻译、学习，随时待命 —— 说话即可开始对话</p>

      <div className="chat-wel-telemetry">
        <span className="chat-wel-dot" />
        VOICE <b>{voice.listening ? 'LISTEN' : voice.aiSpeaking ? 'SPEAK' : 'STANDBY'}</b>
        <span className="chat-wel-sep" />
        CORE <b>ONLINE</b>
        <span className="chat-wel-sep" />
        INPUT <b>READY</b>
      </div>

      <div className="chat-suggestion-grid">
        {PLAIN_SUGGESTIONS.map(s => (
          <SuggestionCard key={s.label} s={s} onClick={() => onSuggestion(s.label)} />
        ))}
      </div>
    </div>
  )
}
