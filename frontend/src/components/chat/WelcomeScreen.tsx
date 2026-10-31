// ChatPage 拆分产物：欢迎屏（普通/角色双形态，行为零变化，T6-10.1）
import React from 'react'
import { Button } from 'antd'
import {
  MessageOutlined, SearchOutlined, EditOutlined, BulbOutlined,
  TranslationOutlined, BookOutlined, HeartOutlined, GlobalOutlined,
  StarFilled, ThunderboltOutlined, SettingOutlined, SwapOutlined,
  AudioOutlined,
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
  onOpenVoiceConsole?: () => void
}

/** 语音降级警示条（FE6-02）：麦克风不可用时必须可见——此前页面照旧显示
 *  「聆听中」而实际零采集，是对用户撒谎。文案属页面内容层（chat 等页面
 *  内容层保持 zh 单语，见 .gaea/AGENTS.md i18n 决策），故直接中文。
 *  样式照抄仓内既有诚实降级提示条（FilePreview/FilePreviewModal 同款琥珀条）。 */
const MIC_DEGRADED_TEXT = '麦克风不可用，未在采集音频，本回合走文本输入'

const MicDegradedNotice: React.FC = () => (
  <div
    role="alert"
    className="mb-2 px-3 py-2 rounded-md border border-amber-500/30 bg-amber-500/5 text-amber-500 text-[11px] leading-relaxed"
  >
    {MIC_DEGRADED_TEXT}
  </div>
)

export const WelcomeScreen: React.FC<WelcomeScreenProps> = ({
  mode, personaLabel, companionName, voice, emoColor,
  activePersonality, onSwitchPersonality, onNavigateLib, onFillInput, onSuggestion,
  onOpenVoiceConsole,
}) => {
  // 遥测行文案：降级时不得显示 LISTEN（麦克风不可用时宣称在听=同一条谎），
  // 与上方警示条口径一致；非降级时逐字零变化。
  const voiceStatus = voice.degraded === 'mic-unavailable'
    ? '不可用'
    : voice.listening ? '聆听中' : voice.aiSpeaking ? '回应中' : '待命'
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

        {voice.degraded === 'mic-unavailable' && <MicDegradedNotice />}

        <div className="chat-wel-orb chat-wel-orb-sm">
          <span className="chat-wel-ring chat-wel-ring-1" aria-hidden="true" />
          <span className="chat-wel-ring chat-wel-ring-2" aria-hidden="true" />
          <CompanionAvatar
            size={104}
            state={voice.aiSpeaking ? 'speaking' : voice.listening ? 'listening' : 'idle'}
            emotionColor={emoColor}
          />
        </div>

        <h2>{companionName}</h2>
        <p>以 {personaLabel} 的身份，今天想聊点什么？</p>

        <div className="chat-wel-telemetry">
          <span className="chat-wel-dot" />
          角色 <b>已就位</b>
          <span className="chat-wel-sep" />
          语音 <b>{voiceStatus}</b>
          <span className="chat-wel-sep" />
          输入 <b>就绪</b>
        </div>

        {onOpenVoiceConsole && (
          <button type="button" className="chat-welcome-voice-entry" onClick={onOpenVoiceConsole}>
            <AudioOutlined aria-hidden /> 进入语音对话
          </button>
        )}

        <PersonaPicker activeId={activePersonality}
          onSelect={onSwitchPersonality} onManage={onNavigateLib}>
          <Button type="default" icon={<SwapOutlined />} className="chat-wel-persona-button">
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

      <span className="chat-wel-kicker">GAEA CORE · 普通对话</span>

      {voice.degraded === 'mic-unavailable' && <MicDegradedNotice />}

      <div className="chat-wel-orb">
        <span className="chat-wel-ring chat-wel-ring-1" aria-hidden="true" />
        <span className="chat-wel-ring chat-wel-ring-2" aria-hidden="true" />
        <VoiceChatOrb
          volume={voice.volume}
          listening={voice.listening}
          speaking={voice.speaking}
          aiSpeaking={voice.aiSpeaking}
          transcript={voice.transcript}
          size={112}
          micUnavailable={voice.degraded === 'mic-unavailable'}
        />
      </div>

      <h2>今天想完成什么？</h2>
      <p>从一个具体问题开始：查资料、写文章、整理想法，或直接把判断交给我。</p>

      <div className="chat-wel-telemetry">
        <span className="chat-wel-dot" />
        语音 <b>{voiceStatus}</b>
        <span className="chat-wel-sep" />
        搜索 <b>可选</b>
        <span className="chat-wel-sep" />
        输入 <b>就绪</b>
      </div>

      {onOpenVoiceConsole && (
        <button type="button" className="chat-welcome-voice-entry" onClick={onOpenVoiceConsole}>
          <AudioOutlined aria-hidden /> 进入语音对话
        </button>
      )}

      <div className="chat-suggestion-grid">
        {PLAIN_SUGGESTIONS.map(s => (
          <SuggestionCard key={s.label} s={s} onClick={() => onSuggestion(s.label)} />
        ))}
      </div>
    </div>
  )
}
