// ModuleLauncher.tsx — AI 中枢首页（启动器）
// 顶栏状态 → 正中语音交互中枢 + 两侧模块卡片；单一主题强调色，跟随 --gaea-glow
import React, { useState, useCallback, useEffect } from 'react'
import {
  MessageOutlined, ReadOutlined, PictureOutlined,
  ToolOutlined, ApiOutlined, SettingOutlined,
  ThunderboltOutlined, ArrowRightOutlined, AudioOutlined,
  StopOutlined, RobotOutlined, UserOutlined,
} from '@ant-design/icons'
import { Tooltip } from 'antd'
import VoiceChatOrb from './VoiceChatOrb'
import { useVoiceChat } from '../hooks/useVoiceChat'
import * as App from '../../src/wailsjsCompat'
import './module-launcher.css'

/** 启动器可跳转的目标页（与 MainLayout 的 Page 类型保持一致的子集） */
export type LauncherTarget =
  | 'chat' | 'novel' | 'imagegen' | 'gaea' | 'modelcenter' | 'settings'

/** 语音入口信号（首页现在本页启动语音，该信号保留兼容旧入口） */
export const VOICE_LAUNCH_FLAG = 'gaea_voice_launch'

interface LauncherModule {
  key: LauncherTarget
  name: string
  desc: string
  icon: React.ReactNode
}

// 每张卡片使用主题强调色（不硬编码多色，跟随 6 套主题预设）
const modules: LauncherModule[] = [
  { key: 'chat', name: '聊天', desc: '与 AI 对话，激发灵感', icon: <MessageOutlined /> },
  { key: 'novel', name: '小说', desc: '世界观、角色与大纲创作', icon: <ReadOutlined /> },
  { key: 'imagegen', name: '绘梦', desc: 'AI 图像生成工作台', icon: <PictureOutlined /> },
  { key: 'gaea', name: '办公', desc: '通用办公工作台', icon: <ToolOutlined /> },
  { key: 'modelcenter', name: '模型中心', desc: '模型引擎管理与配置', icon: <ApiOutlined /> },
  { key: 'settings', name: '设置', desc: '应用偏好与主题外观', icon: <SettingOutlined /> },
]

// 已/右卡片列（正中语音交互，卡片分居两侧）
const leftModules = modules.slice(0, 3)
const rightModules = modules.slice(3)

interface ModuleLauncherProps {
  onNavigate: (target: LauncherTarget) => void
  /** 当前激活的 AI 模型名（顶栏已加装，传入提升真实感） */
  activeModel?: string
}

/** 单张玻璃档案卡 */
const LauncherCard: React.FC<{ m: LauncherModule; idx: number; onOpen: () => void }> = ({ m, idx, onOpen }) => (
  <div
    role="button"
    tabIndex={0}
    aria-label={`进入${m.name}模块`}
    className="ml-card"
    style={{ '--ml-i': idx } as React.CSSProperties}
    onClick={onOpen}
    onKeyDown={(e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault()
        onOpen()
      }
    }}
  >
    <div className="ml-card-icon">{m.icon}</div>
    <div className="ml-card-body">
      <div className="ml-card-name">
        {m.name}
        <ArrowRightOutlined className="ml-card-arrow" />
      </div>
      <div className="ml-card-desc">{m.desc}</div>
    </div>
  </div>
)

/** 卡片列（左右两侧） */
const CardColumn: React.FC<{ list: LauncherModule[]; onNavigate: (t: LauncherTarget) => void }> = ({ list, onNavigate }) => (
  <div className="ml-col">
    {list.map((m, i) => (
      <LauncherCard key={m.key} m={m} idx={i} onOpen={() => onNavigate(m.key)} />
    ))}
  </div>
)

/** 语音对话气泡 */
const ChatBubble: React.FC<{ role: 'user' | 'assistant'; text: string }> = ({ role, text }) => {
  const isUser = role === 'user'
  return (
    <div className={`ml-bubble-row ${isUser ? 'ml-bubble-user' : 'ml-bubble-ai'}`}>
      {!isUser && (
        <div className="ml-avatar ml-avatar-ai"><RobotOutlined /></div>
      )}
      <div className="ml-bubble">{text}</div>
      {isUser && (
        <div className="ml-avatar ml-avatar-user"><UserOutlined /></div>
      )}
    </div>
  )
}

/**
 * ModuleLauncher — AI 中枢首页（正中语言粒子语音交互 + 模块启动器）。
 * 点击「进入语音对话」直接在本页启动麦克风开始语音交互（不跳转）。
 * 语音后端走聊天 voiceManager 管道，对话人格与聊天板块保持一致（后端持久化）。
 */
const ModuleLauncher: React.FC<ModuleLauncherProps> = ({ onNavigate, activeModel }) => {
  // ── 语音交互（本页直启麦克风）──
  const [userText, setUserText] = useState('')
  const [aiReply, setAiReply] = useState('')

  // 粒子球尺寸随视口高度自适应（全局窗口变化实时响应，400 为上限）
  const [orbSize, setOrbSize] = useState(() =>
    Math.min(380, Math.max(240, (typeof window !== 'undefined' ? window.innerHeight : 800) - 480)))
  useEffect(() => {
    const onResize = () => setOrbSize(Math.min(380, Math.max(240, window.innerHeight - 480)))
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  }, [])
  const { state: voice, start, stop, interrupt } = useVoiceChat({
    onTranscript: (t) => { setUserText(t); setAiReply('') },
    onReply: (t) => setAiReply(t),
  })

  // 首页语音固定使用核心人格 gaea（与聊天板块解耦，不跟随聊天所选人格）
  const voicePersonaLabel = 'gaea'

  const toggleVoice = useCallback(async () => {
    if (voice.active) { stop(); return }
    // 首页语音始终使用 gaea
    try { await App.VoiceApplySettings?.({ personalityPresetId: 'gaea' }) } catch (_) {}
    setUserText('')
    setAiReply('')
    await start()
  }, [voice.active, start, stop])

  const voiceStateLabel = voice.aiSpeaking
    ? 'AI 回复中…'
    : voice.listening
      ? '正在聆听…'
      : voice.active
        ? '语音待命'
        : '待机'

  const statusClass = voice.aiSpeaking
    ? 'ml-status ml-status-speaking'
    : voice.listening
      ? 'ml-status ml-status-listening'
      : voice.active
        ? 'ml-status ml-status-active'
        : 'ml-status'

  const hasChat = !!userText || !!aiReply

  // 语音状态 → 中枢面板状态类（驱动 HUD 脉冲环 / 声谱 / 扫描线的强度与配色）
  const centerClass = [
    'ml-center',
    voice.aiSpeaking ? 'is-speaking' : voice.listening ? 'is-listening' : voice.active ? 'is-active' : '',
  ].filter(Boolean).join(' ')

  return (
    <div className="ml">
      <div className="ml-shell">
        {/* ── AI 中枢状态栏 ── */}
        <div className="ml-top">
          <ThunderboltOutlined className="ml-top-icon" />
          <div className="ml-top-main">
            <div className="ml-top-title">
              <span className="live-dot" />
              AI 中枢在线
              <span className="ml-chip">GAEA CORE</span>
              {activeModel && (
                <span className="ml-chip ml-chip-model" title={activeModel}>
                  {activeModel}
                </span>
              )}
            </div>
            <div className="ml-top-sub">
              正中语音晶核直启麦克风 —— 与「{voicePersonaLabel}」语音对话
            </div>
          </div>
          <span
            role="button"
            tabIndex={0}
            className="ml-chat-cta"
            onClick={() => onNavigate('chat')}
            onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onNavigate('chat') } }}
          >
            聊天板块 <ArrowRightOutlined style={{ fontSize: 11 }} />
          </span>
        </div>

        {/* ── 主区：左卡片 | 语音交互 | 右卡片 ── */}
        <div className="ml-main">
          <CardColumn list={leftModules} onNavigate={onNavigate} />

          {/* 正中：语音粒子交互 */}
          <div className={centerClass}>
            {/* 科幻 HUD 边框角标 */}
            <div className="ml-hud-frame" aria-hidden="true">
              <span className="ml-hud-corner ml-hud-tl" />
              <span className="ml-hud-corner ml-hud-tr" />
              <span className="ml-hud-corner ml-hud-bl" />
              <span className="ml-hud-corner ml-hud-br" />
            </div>

            <div className="ml-center-head">
              <span className="live-dot" />
              <span className="ml-center-title">语音交互中枢</span>
              <span className="ml-chip">{voicePersonaLabel}</span>
            </div>

            <div className="ml-orb-wrap">
              {/* 雷达脉冲环：待机慢速呼吸，聆听/回复加速扩散 */}
              <span className="ml-ring ml-ring-1" aria-hidden="true" />
              <span className="ml-ring ml-ring-2" aria-hidden="true" />
              <span className="ml-ring ml-ring-3" aria-hidden="true" />
              <VoiceChatOrb
                volume={voice.volume}
                listening={voice.listening}
                speaking={voice.speaking}
                aiSpeaking={voice.aiSpeaking}
                transcript={voice.transcript}
                size={orbSize}
              />
            </div>

            {/* 声谱均衡条：随语音状态律动 */}
            <div className="ml-eq" aria-hidden="true">
              {Array.from({ length: 9 }).map((_, i) => (
                <span key={i} className="ml-eq-bar" style={{ '--eq-i': i } as React.CSSProperties} />
              ))}
            </div>

            {/* HUD 遥测读数 */}
            <div className="ml-telemetry">
              <span className="ml-tele-dot" />
              CORE <b>GAEA-07</b>
              <span className="ml-tele-sep" />
              LINK <b>OK</b>
              <span className="ml-tele-sep" />
              VAD <b>{voice.active ? 'ON' : 'STBY'}</b>
              <span className="ml-tele-sep" />
              VOL <b>{voice.active ? `${Math.round(voice.volume * 100)}%` : '--'}</b>
            </div>

            <div className={statusClass}>
              {voiceStateLabel}
            </div>

            <div className="ml-bubbles">
              {hasChat ? (
                <>
                  {userText && <ChatBubble role="user" text={userText} />}
                  {aiReply && <ChatBubble role="assistant" text={aiReply} />}
                </>
              ) : (
                <div className="ml-bubble-empty">
                  语音晶核汇聚成声 —— 点击下方按钮开始语音对话
                </div>
              )}
            </div>

            {voice.error && (
              <div className="ml-voice-err">{voice.error}</div>
            )}

            <div className="ml-controls">
              {voice.active && voice.aiSpeaking && (
                <button className="ml-interrupt-btn" onClick={interrupt}>
                  <StopOutlined /> 打断回复
                </button>
              )}
              <Tooltip title={voice.active ? '结束语音对话' : '启动麦克风，开始语音交互'}>
                <button
                  className={`ml-start-btn ${voice.active ? 'ml-starting' : ''}`}
                  onClick={toggleVoice}
                  type="button"
                >
                  {voice.active ? <StopOutlined /> : <AudioOutlined />}
                  {voice.active ? '结束语音对话' : '进入语音对话'}
                </button>
              </Tooltip>
            </div>
          </div>

          <CardColumn list={rightModules} onNavigate={onNavigate} />
        </div>
      </div>
    </div>
  )
}

export default ModuleLauncher
