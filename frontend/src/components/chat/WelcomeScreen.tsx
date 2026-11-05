// ChatPage 拆分产物：欢迎屏（普通/角色双形态，行为零变化，T6-10.1）
import React from 'react'
import { Button } from 'antd'
import {
  MessageOutlined, SearchOutlined, EditOutlined, BulbOutlined,
  TranslationOutlined, BookOutlined, HeartOutlined, GlobalOutlined,
  StarFilled, ThunderboltOutlined, SettingOutlined, SwapOutlined,
  AudioOutlined,
} from '@ant-design/icons'
import { CompanionAvatar } from '../CompanionAvatar'
import PersonaPicker from '../PersonaPicker'
import { SuggestionCard } from './SuggestionCard'

const PLAIN_SUGGESTIONS = [
  { icon: <MessageOutlined />, label: '说清一个问题', desc: '从一个具体问题开始，边走边想', prompt: '我想把一个问题说清楚。先听我讲背景，再帮我梳理关键点：' },
  { icon: <SearchOutlined />, label: '查找资料', desc: '联网检索，整理来源与结论', prompt: '帮我查找并核验以下资料，整理来源与结论：' },
  { icon: <EditOutlined />, label: '起草内容', desc: '文章、报告、邮件，先出可用初稿', prompt: '帮我起草一份内容。先向我确认必要背景，再给出可用初稿：' },
  { icon: <BulbOutlined />, label: '梳理思路', desc: '把零散想法整理成清晰结构', prompt: '我有一些零散想法，帮我整理成清晰结构：' },
  { icon: <TranslationOutlined />, label: '翻译与改写', desc: '保留语气，跨语言准确表达', prompt: '帮我翻译并润色以下内容，保留原意和语气：' },
  { icon: <BookOutlined />, label: '解释概念', desc: '从直觉例子讲到关键细节', prompt: '请用一个具体例子解释这个概念，再补充关键细节：' },
]

const PERSONA_SUGGESTIONS = [
  { icon: <HeartOutlined />, label: '聊聊今天', desc: '分享一件刚刚发生的小事', prompt: '今天发生了一件小事，我想和你说说：' },
  { icon: <SearchOutlined />, label: '帮我理一理', desc: '把复杂的感受慢慢说清楚', prompt: '我现在有些乱，陪我慢慢理一理：' },
  { icon: <GlobalOutlined />, label: '看看新消息', desc: '查一查最近发生的事', prompt: '帮我查一下最近的相关消息，并说明来源：' },
  { icon: <StarFilled />, label: '聊点喜欢的', desc: '从电影、音乐或一本书开始', prompt: '我最近喜欢的东西是：' },
  { icon: <ThunderboltOutlined />, label: '陪我收尾', desc: '整理今天，准备休息', prompt: '帮我回顾一下今天，然后准备休息：' },
]

export interface WelcomeScreenProps {
  mode: string
  personaLabel: string
  companionName: string
  emoColor: string
  activePersonality: string
  onSwitchPersonality: (id: string) => void
  onNavigateLib: () => void
  onFillInput: (label: string) => void
  onOpenVoiceConsole?: () => void
}


export const WelcomeScreen: React.FC<WelcomeScreenProps> = ({
  mode, personaLabel, companionName, emoColor,
  activePersonality, onSwitchPersonality, onNavigateLib, onFillInput,
  onOpenVoiceConsole,
}) => {
  const personaMode = mode !== 'plain'
  const suggestions = personaMode ? PERSONA_SUGGESTIONS : PLAIN_SUGGESTIONS
  const title = personaMode ? `和 ${companionName} 聊聊` : '今天想完成什么？'
  const description = personaMode
    ? `以 ${personaLabel} 的身份继续对话。文字和声音都可以，内容只保存在本机。`
    : '写、查、想，从一句话开始。文字与实时语音共用同一条会话。'

  return (
    <div className={`chat-welcome chat-welcome--signal${personaMode ? ' is-persona' : ''}`}>
      <section className="chat-welcome-intro" aria-labelledby="chat-welcome-title">
        <div className="chat-welcome-overline">
          <span className="chat-wel-kicker">{personaMode ? `角色对话 · ${personaLabel}` : '普通对话'}</span>
          <span className="chat-welcome-local">
            <span className="chat-welcome-local-dot" aria-hidden="true" />
            本机保存
          </span>
        </div>

        <div className="chat-welcome-identity">
          <div className="chat-welcome-avatar" aria-hidden="true">
            <CompanionAvatar
              size={personaMode ? 68 : 60}
              state="idle"
              emotionColor={emoColor}
            />
          </div>
          <div className="chat-welcome-copy">
            <h2 id="chat-welcome-title">{title}</h2>
            <p>{description}</p>
          </div>
        </div>

        {onOpenVoiceConsole && (
          <button
            type="button"
            className="chat-welcome-voice-card"
            onClick={onOpenVoiceConsole}
            aria-label={personaMode ? `与 ${companionName} 开始实时语音对话` : '开始实时语音对话'}
          >
            <span className="chat-welcome-voice-icon" aria-hidden="true">
              <AudioOutlined />
            </span>
            <span className="chat-welcome-voice-copy">
              <strong>打开实时语音</strong>
              <span>实时转写与回答会自动写回当前会话</span>
            </span>
            <span className="chat-welcome-voice-live" aria-hidden="true">
              <i /><i /><i />
            </span>
            <span className="chat-welcome-voice-arrow" aria-hidden="true">→</span>
          </button>
        )}

        <div className="chat-welcome-tools">
          <PersonaPicker
            activeId={activePersonality}
            onSelect={onSwitchPersonality}
            onManage={onNavigateLib}
          >
            <Button type="default" icon={<SwapOutlined />} className="chat-wel-persona-button">
              {personaMode ? '切换角色' : '选择角色'}
            </Button>
          </PersonaPicker>
          <button type="button" className="chat-welcome-library-link" onClick={onNavigateLib}>
            <SettingOutlined aria-hidden="true" />
            角色库
          </button>
        </div>
      </section>

      <section className="chat-welcome-prompts" aria-labelledby="chat-welcome-prompts-title">
        <div className="chat-welcome-section-head">
          <span id="chat-welcome-prompts-title">可以这样开始</span>
          <span>{suggestions.length} 个建议</span>
        </div>
        <div className="chat-suggestion-grid">
          {suggestions.map(s => (
            <SuggestionCard
              key={s.label}
              s={s}
              onClick={() => onFillInput(s.prompt)}
            />
          ))}
        </div>
      </section>
    </div>
  )
}
