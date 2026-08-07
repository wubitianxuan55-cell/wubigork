import React, { useState, useCallback, useEffect } from 'react'
import {
  MessageOutlined, ReadOutlined, PictureOutlined, HeartOutlined,
  FileTextOutlined, ToolOutlined, ApiOutlined, SettingOutlined,
  ThunderboltOutlined, ArrowRightOutlined, AudioOutlined,
  StopOutlined, RobotOutlined, UserOutlined,
} from '@ant-design/icons'
import { Typography, Button, Tooltip } from 'antd'
import VoiceChatOrb from './VoiceChatOrb'
import { useVoiceChat } from '../hooks/useVoiceChat'
import * as App from '../../wailsjs/go/app/App'
import { requestPersonaEnter } from '../utils/chatNav'

/** 启动器可跳转的目标页（与 MainLayout 的 Page 类型保持一致的子集） */
export type LauncherTarget =
  | 'chat' | 'novel' | 'imagegen' | 'office' | 'gaea' | 'modelcenter' | 'settings'

/** 轻语板块语音入口信号（首页现在本页启动语音，该信号保留兼容旧入口） */
export const VOICE_LAUNCH_FLAG = 'gaea_voice_launch'

interface LauncherModule {
  key: LauncherTarget | 'whisper'
  name: string
  desc: string
  icon: React.ReactNode
  accent: string
}

// 每张卡片独立主题色（中亮度色，亮/暗主题下均保证对比度）
const modules: LauncherModule[] = [
  { key: 'chat', name: '聊天', desc: '与 AI 对话，激发灵感', icon: <MessageOutlined />, accent: '#60a5fa' },
  { key: 'novel', name: '小说', desc: '世界观、角色与大纲创作', icon: <ReadOutlined />, accent: '#a78bfa' },
  { key: 'imagegen', name: '绘梦', desc: 'AI 图像生成工作台', icon: <PictureOutlined />, accent: '#f472b6' },
  { key: 'whisper', name: '轻语', desc: '陪伴式 AI 对话（人格模式）', icon: <HeartOutlined />, accent: '#fb7185' },
  { key: 'office', name: '方案编写', desc: '投标方案六标签页编写', icon: <FileTextOutlined />, accent: '#f59e0b' },
  { key: 'gaea', name: '办公', desc: 'gaea 办公套件与文档', icon: <ToolOutlined />, accent: '#2dd4bf' },
  { key: 'modelcenter', name: '模型中心', desc: '模型引擎管理与配置', icon: <ApiOutlined />, accent: '#34d399' },
  { key: 'settings', name: '设置', desc: '应用偏好与主题外观', icon: <SettingOutlined />, accent: '#94a3b8' },
]

// 左/右卡片列（正中语音交互，卡片分居两侧）
const leftModules = modules.slice(0, 4)
const rightModules = modules.slice(4)

interface ModuleLauncherProps {
  onNavigate: (target: LauncherTarget) => void
  /** 当前激活的 AI 模型名（顶栏已加载，传入提升真实感） */
  activeModel?: string
}

/** 单张透明虚空悬浮卡片 */
const LauncherCard: React.FC<{ m: LauncherModule; idx: number; onOpen: () => void }> = ({ m, idx, onOpen }) => (
  <div
    role="button"
    tabIndex={0}
    aria-label={`进入${m.name}模块`}
    className="void-card neon-card launcher-card"
    onClick={onOpen}
    onKeyDown={(e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault()
        onOpen()
      }
    }}
    style={{
      animationDelay: `${0.08 + idx * 0.05}s, ${0.7 + (idx % 2) * 0.8}s`,
      background: 'rgba(255, 255, 255, 0.04)',
      WebkitBackdropFilter: 'blur(10px) saturate(120%)',
      backdropFilter: 'blur(10px) saturate(120%)',
      border: '1px solid rgba(255, 255, 255, 0.1)',
    }}
  >
    <div style={{
      padding: 18,
      display: 'flex', flexDirection: 'column', gap: 10,
      height: '100%',
    }}>
      <div style={{
        width: 42, height: 42, borderRadius: 'var(--md-sys-radius-lg)',
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        fontSize: 21, color: m.accent,
        background: `color-mix(in srgb, ${m.accent} 14%, transparent)`,
        boxShadow: `0 0 16px color-mix(in srgb, ${m.accent} 35%, transparent)`,
        transition: 'box-shadow var(--md-sys-transition-normal), transform var(--md-sys-transition-normal)',
      }} className="launcher-icon">
        {m.icon}
      </div>
      <div>
        <div style={{
          fontSize: 15, fontWeight: 600,
          color: 'var(--md-sys-color-text)', marginBottom: 2,
          display: 'flex', alignItems: 'center', gap: 6,
        }}>
          {m.name}
          <ArrowRightOutlined
            style={{ fontSize: 10, opacity: 0, transform: 'translateX(-4px)', color: m.accent,
              transition: 'opacity 0.2s, transform 0.2s' }}
            className="launcher-arrow"
          />
        </div>
        <div style={{
          fontSize: 12, lineHeight: 1.5,
          color: 'var(--md-sys-color-text-secondary)',
        }}>
          {m.desc}
        </div>
      </div>
    </div>
  </div>
)

/** 卡片列（左右两侧，垂直悬浮排列） */
const CardColumn: React.FC<{ list: LauncherModule[]; onNavigate: (t: LauncherTarget) => void }> = ({ list, onNavigate }) => (
  <div style={{ display: 'flex', flexDirection: 'column', gap: 16, justifyContent: 'center' }}>
    {list.map((m, i) => (
      <LauncherCard key={m.key} m={m} idx={i} onOpen={() => {
        if (m.key === 'whisper') {
          requestPersonaEnter()
          onNavigate('chat')
          return
        }
        onNavigate(m.key)
      }} />
    ))}
  </div>
)

/** 语音对话气泡 */
const ChatBubble: React.FC<{ role: 'user' | 'assistant'; text: string }> = ({ role, text }) => {
  const isUser = role === 'user'
  return (
    <div style={{ display: 'flex', justifyContent: isUser ? 'flex-end' : 'flex-start', gap: 8, alignItems: 'flex-start' }}>
      {!isUser && (
        <div style={{
          width: 24, height: 24, borderRadius: '50%', flexShrink: 0,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          background: 'color-mix(in srgb, var(--gaea-glow) 16%, transparent)',
          color: 'var(--gaea-glow)', fontSize: 13,
          boxShadow: '0 0 10px color-mix(in srgb, var(--gaea-glow) 30%, transparent)',
        }}>
          <RobotOutlined />
        </div>
      )}
      <div style={{
        maxWidth: '82%', padding: '6px 11px', borderRadius: 13, fontSize: 12.5, lineHeight: 1.55,
        color: 'var(--md-sys-color-text)',
        background: isUser
          ? 'linear-gradient(135deg, color-mix(in srgb, var(--gaea-glow) 22%, transparent), color-mix(in srgb, var(--gaea-glow) 10%, transparent))'
          : 'rgba(255, 255, 255, 0.06)',
        border: `1px solid ${isUser ? 'color-mix(in srgb, var(--gaea-glow) 32%, transparent)' : 'rgba(255, 255, 255, 0.12)'}`,
        backdropFilter: 'blur(10px)',
        wordBreak: 'break-word',
      }}>
        {text}
      </div>
      {isUser && (
        <div style={{
          width: 24, height: 24, borderRadius: '50%', flexShrink: 0,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          background: 'rgba(255, 255, 255, 0.08)',
          color: 'var(--md-sys-color-text-secondary)', fontSize: 13,
        }}>
          <UserOutlined />
        </div>
      )}
    </div>
  )
}

/**
 * ModuleLauncher — AI 中枢首页（正中语言粒子语音交互 + 模块启动器）。
 * 点击「进入语音对话」直接在本页启动麦克风开始语音交互（不跳转）：
 * 语音后端走轻语板块 voiceManager 管道，对话人格 = 核心助手 gaea（大地女神）。
 */
const ModuleLauncher: React.FC<ModuleLauncherProps> = ({ onNavigate, activeModel }) => {
  // ── 语音交互（本页直启麦克风）──
  const [userText, setUserText] = useState('')
  const [aiReply, setAiReply] = useState('')

  // 粒子球尺寸随视口高度自适应（全屏/窗口变化实时响应，400 为上限）
  const [orbSize, setOrbSize] = useState(() =>
    Math.min(400, Math.max(260, (typeof window !== 'undefined' ? window.innerHeight : 800) - 460)))
  useEffect(() => {
    const onResize = () => setOrbSize(Math.min(400, Math.max(260, window.innerHeight - 460)))
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  }, [])
  const { state: voice, start, stop, interrupt } = useVoiceChat({
    onTranscript: (t) => { setUserText(t); setAiReply('') },
    onReply: (t) => setAiReply(t),
  })

  const toggleVoice = useCallback(async () => {
    if (voice.active) { stop(); return }
    // 语音人格锁定为核心助手 gaea（大地女神）；后端走轻语对话管道
    try { await (App as any).VoiceApplySettings?.({ personalityPresetId: 'gaea' }) } catch (_) {}
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

  const hasChat = !!userText || !!aiReply

  return (
    <div style={{
      flex: 1,
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      padding: '16px 24px 28px',
      position: 'relative', zIndex: 1,
      minHeight: 0,
    }}>
      <div style={{
        display: 'flex', flexDirection: 'column', gap: 16,
        width: '100%', maxWidth: 'min(1280px, 100%)', minHeight: 0, flex: 1,
      }}>
        {/* ═══ AI 中枢状态条 ═══ */}
        <div
          className="void-card scanline-top"
          style={{
            position: 'relative',
            display: 'flex', alignItems: 'center', gap: 14,
            padding: '10px 18px', borderRadius: 'var(--md-sys-radius-lg)',
            animation: 'launcherFadeUp 0.4s cubic-bezier(0.16, 1, 0.3, 1) backwards',
            flexShrink: 0,
          }}
        >
          <ThunderboltOutlined className="neon-glow-icon" style={{ fontSize: 18, color: 'var(--gaea-glow)' }} />
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
              <span className="live-dot" />
              <Typography.Text strong style={{ fontSize: 14, color: 'var(--md-sys-color-text)' }}>
                AI 中枢在线
              </Typography.Text>
              <span style={{
                fontSize: 11, padding: '1px 8px', borderRadius: 10,
                background: 'color-mix(in srgb, var(--gaea-glow) 14%, transparent)',
                color: 'var(--gaea-glow)',
                fontWeight: 500, letterSpacing: '0.04em',
                border: '1px solid color-mix(in srgb, var(--gaea-glow) 30%, transparent)',
              }}>
                GAEA CORE
              </span>
              {activeModel && (
                <span style={{
                  fontSize: 11, padding: '1px 8px', borderRadius: 10,
                  background: 'color-mix(in srgb, var(--gaea-glow) 14%, transparent)',
                  color: 'var(--gaea-glow)',
                  fontWeight: 500,
                  border: '1px solid color-mix(in srgb, var(--gaea-glow) 30%, transparent)',
                  maxWidth: 140, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                }}>
                  {activeModel}
                </span>
              )}
            </div>
            <Typography.Text style={{ fontSize: 12, color: 'var(--md-sys-color-text-secondary)' }}>
              正中语言粒子直启麦克风 —— 与核心助手 gaea（大地女神）语音对话
            </Typography.Text>
          </div>
          <span
            role="button" tabIndex={0}
            onClick={() => onNavigate('chat')}
            onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onNavigate('chat') } }}
            style={{
              display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer',
              fontSize: 13, fontWeight: 500, whiteSpace: 'nowrap',
              color: 'var(--gaea-glow)',
              padding: '6px 14px', borderRadius: 'var(--md-sys-radius-md)',
              border: '1px solid color-mix(in srgb, var(--gaea-glow) 30%, transparent)',
              background: 'color-mix(in srgb, var(--gaea-glow) 12%, transparent)',
              boxShadow: '0 0 14px color-mix(in srgb, var(--gaea-glow) 22%, transparent)',
              transition: 'box-shadow var(--md-sys-transition-normal), transform var(--md-sys-transition-normal)',
            }}
          >
            聊天面板 <ArrowRightOutlined style={{ fontSize: 11 }} />
          </span>
        </div>

        {/* ═══ 主区域：左卡片 | 语言交互 | 右卡片 ═══ */}
        <div style={{
          flex: 1, minHeight: 0,
          display: 'grid',
          gridTemplateColumns: 'minmax(200px, 1fr) minmax(360px, 1.6fr) minmax(200px, 1fr)',
          gap: 20,
          alignItems: 'center',
        }}>
          <CardColumn list={leftModules} onNavigate={onNavigate} />

          {/* ── 正中：语言粒子语音交互 ── */}
          <div
            className="void-card neon-card language-core"
            style={{
              position: 'relative',
              display: 'flex', flexDirection: 'column', alignItems: 'center',
              padding: '14px 22px 18px',
              borderRadius: 'var(--md-sys-radius-xl)',
              animation: 'launcherFadeUp 0.5s cubic-bezier(0.16, 1, 0.3, 1) 0.12s backwards',
              overflow: 'hidden',
            }}
          >
            {/* 标题行 */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 2 }}>
              <span className="live-dot" />
              <Typography.Text strong style={{ fontSize: 15, color: 'var(--md-sys-color-text)', letterSpacing: '0.04em' }}>
                语言交互中枢
              </Typography.Text>
              <span style={{
                fontSize: 10, padding: '1px 7px', borderRadius: 9,
                background: 'color-mix(in srgb, var(--gaea-glow) 14%, transparent)',
                color: 'var(--gaea-glow)',
                border: '1px solid color-mix(in srgb, var(--gaea-glow) 30%, transparent)',
                fontWeight: 500, letterSpacing: '0.06em',
              }}>
                gaea · 大地女神
              </span>
            </div>

            {/* 语言粒子球（语音状态驱动，聆听/回复动态） */}
            <VoiceChatOrb
              volume={voice.volume}
              listening={voice.listening}
              speaking={voice.speaking}
              aiSpeaking={voice.aiSpeaking}
              transcript={voice.transcript}
              size={orbSize}
            />

            {/* 状态行 */}
            <div style={{ minHeight: 24, marginTop: 2, fontSize: 12.5, fontWeight: 500, letterSpacing: '0.05em',
              color: voice.aiSpeaking ? '#64b5f6' : voice.listening ? '#ff8a65' : voice.active ? 'var(--gaea-glow)' : 'var(--md-sys-color-text-secondary)',
              textShadow: voice.active ? '0 0 10px color-mix(in srgb, var(--gaea-glow) 45%, transparent)' : 'none',
              transition: 'color 0.3s' }}>
              {voiceStateLabel}
            </div>

            {/* 对话气泡区 */}
            <div style={{
              width: '100%', minHeight: 62, maxHeight: 112, overflowY: 'auto',
              display: 'flex', flexDirection: 'column', gap: 8, marginTop: 6,
              padding: '2px 2px',
            }}>
              {hasChat ? (
                <>
                  {userText && <ChatBubble role="user" text={userText} />}
                  {aiReply && <ChatBubble role="assistant" text={aiReply} />}
                </>
              ) : (
                <div style={{
                  flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center',
                  color: 'var(--md-sys-color-text-secondary)', fontSize: 12, opacity: 0.75,
                  textAlign: 'center', lineHeight: 1.7,
                }}>
                  语言粒子汇聚成声 —— 点击下方按钮开始语音对话
                </div>
              )}
            </div>

            {voice.error && (
              <Typography.Text style={{ color: '#fb7185', fontSize: 12, marginTop: 4 }}>
                {voice.error}
              </Typography.Text>
            )}

            {/* 控制区 */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginTop: 12 }}>
              {voice.active && voice.aiSpeaking && (
                <Button shape="round" icon={<StopOutlined />} onClick={interrupt}
                  style={{ border: '1px solid color-mix(in srgb, #fb7185 45%, transparent)', color: '#fb7185',
                    background: 'color-mix(in srgb, #fb7185 10%, transparent)', fontSize: 13 }}>
                  打断回复
                </Button>
              )}
              <Tooltip title={voice.active ? '结束语音对话' : '启动麦克风，开始语音交互'}>
                <Button
                  type="primary"
                  icon={voice.active ? <StopOutlined /> : <AudioOutlined />}
                  size="large"
                  onClick={toggleVoice}
                  style={{
                    height: 46, padding: '0 26px', borderRadius: 23,
                    fontSize: 14.5, fontWeight: 600, letterSpacing: '0.04em',
                    border: 'none',
                    background: voice.active
                      ? 'linear-gradient(135deg, #fb7185, #f43f5e)'
                      : 'linear-gradient(135deg, var(--gaea-glow), color-mix(in srgb, var(--gaea-glow) 55%, #8b5cf6))',
                    color: voice.active ? '#fff' : '#042f2e',
                    boxShadow: voice.active
                      ? '0 0 24px rgba(244, 63, 94, 0.5)'
                      : '0 0 26px color-mix(in srgb, var(--gaea-glow) 45%, transparent)',
                    transition: 'box-shadow var(--md-sys-transition-normal), transform var(--md-sys-transition-normal), background var(--md-sys-transition-normal)',
                  }}
                  className={voice.active ? 'voice-btn-active' : ''}
                >
                  {voice.active ? '结束语音对话' : '进入语音对话'}
                </Button>
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
