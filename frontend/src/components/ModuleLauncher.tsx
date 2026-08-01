import React, { useState, useCallback, useRef, useEffect } from 'react'
import {
  MessageOutlined, ReadOutlined, PictureOutlined, HeartOutlined,
  FileTextOutlined, ToolOutlined, ApiOutlined, SettingOutlined,
  ThunderboltOutlined, ArrowRightOutlined, AudioOutlined, SendOutlined,
  StopOutlined, RobotOutlined, UserOutlined,
} from '@ant-design/icons'
import { Typography, Input, Button, Tooltip } from 'antd'
import VoiceChatOrb from './VoiceChatOrb'
import { useVoiceChat } from '../hooks/useVoiceChat'
import * as App from '../../wailsjs/go/app/App'

/** 启动器可跳转的目标页（与 MainLayout 的 Page 类型保持一致的子集） */
export type LauncherTarget =
  | 'chat' | 'novel' | 'imagegen' | 'whisper' | 'office' | 'gaea' | 'modelcenter' | 'settings'

interface LauncherModule {
  key: LauncherTarget
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
  { key: 'whisper', name: '轻语', desc: '陪伴式 AI 心灵对话', icon: <HeartOutlined />, accent: '#fb7185' },
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

/** 单张霓虹玻璃悬浮卡片 */
const LauncherCard: React.FC<{ m: LauncherModule; idx: number; onOpen: () => void }> = ({ m, idx, onOpen }) => (
  <div
    role="button"
    tabIndex={0}
    aria-label={`进入${m.name}模块`}
    className="neon-card launcher-card"
    onClick={onOpen}
    onKeyDown={(e) => {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault()
        onOpen()
      }
    }}
    style={{
      animationDelay: `${0.08 + idx * 0.05}s, ${0.7 + (idx % 2) * 0.8}s`,
      background: 'var(--gaea-glass-bg, var(--md-sys-color-surface-container))',
      WebkitBackdropFilter: 'blur(14px) saturate(130%)',
      backdropFilter: 'blur(14px) saturate(130%)',
      border: '1px solid var(--md-sys-color-outline-variant)',
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
      <LauncherCard key={m.key} m={m} idx={i} onOpen={() => onNavigate(m.key)} />
    ))}
  </div>
)

/** 语言交互气泡（识别/回复消息） */
const ChatBubble: React.FC<{ role: 'user' | 'assistant'; text: string }> = ({ role, text }) => {
  const isUser = role === 'user'
  return (
    <div style={{ display: 'flex', justifyContent: isUser ? 'flex-end' : 'flex-start', gap: 8, alignItems: 'flex-start' }}>
      {!isUser && (
        <div style={{
          width: 26, height: 26, borderRadius: '50%', flexShrink: 0,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          background: 'color-mix(in srgb, var(--gaea-glow) 16%, transparent)',
          color: 'var(--gaea-glow)', fontSize: 14,
          boxShadow: '0 0 10px color-mix(in srgb, var(--gaea-glow) 30%, transparent)',
        }}>
          <RobotOutlined />
        </div>
      )}
      <div style={{
        maxWidth: '82%', padding: '7px 12px', borderRadius: 14, fontSize: 13, lineHeight: 1.55,
        color: 'var(--md-sys-color-text)',
        background: isUser
          ? 'linear-gradient(135deg, color-mix(in srgb, var(--gaea-glow) 22%, transparent), color-mix(in srgb, var(--gaea-glow) 10%, transparent))'
          : 'var(--md-sys-color-surface-container-high)',
        border: `1px solid ${isUser ? 'color-mix(in srgb, var(--gaea-glow) 32%, transparent)' : 'var(--md-sys-color-outline-variant)'}`,
        backdropFilter: 'blur(10px)',
        wordBreak: 'break-word',
      }}>
        {text}
      </div>
      {isUser && (
        <div style={{
          width: 26, height: 26, borderRadius: '50%', flexShrink: 0,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          background: 'var(--md-sys-color-surface-container-high)',
          color: 'var(--md-sys-color-text-secondary)', fontSize: 14,
        }}>
          <UserOutlined />
        </div>
      )}
    </div>
  )
}

/**
 * ModuleLauncher — AI 中枢首页（语言交互 + 模块启动器）。
 * 正中 = 语言粒子交互球（语音后端走轻语板块管道，对话直连默认平台 AI 助手 gaea）；
 * 两侧 = 霓虹玻璃悬浮卡片墙。整体三栏悬浮布局。
 */
const ModuleLauncher: React.FC<ModuleLauncherProps> = ({ onNavigate, activeModel }) => {
  const [userText, setUserText] = useState('')
  const [aiReply, setAiReply] = useState('')
  const [sending, setSending] = useState(false)
  const [input, setInput] = useState('')
  const inputRef = useRef<any>(null)

  // 语音对话（语音后端 = 轻语板块 voiceManager 管道；对话目标 = gaea 通用 AI）
  const handleTranscript = useCallback((text: string) => {
    setUserText(text)
  }, [])
  const handleReply = useCallback((text: string) => {
    setAiReply(text)
  }, [])
  const { state: voice, start, stop, interrupt } = useVoiceChat({ onTranscript: handleTranscript, onReply: handleReply })

  // 启动语音前切到 gaea 对话目标（与默认平台 AI 助手直接对话，无人格）
  const toggleVoice = useCallback(async () => {
    if (voice.active) { stop(); return }
    try {
      await App.VoiceSetChatTarget('gaea')
    } catch (err: any) {
      console.warn('[Launcher] 语音对话目标切换失败，回退轻语引擎:', err)
    }
    await start()
  }, [voice.active, start, stop])

  // 文字对话（语音不可用时补充通道）
  const handleSend = useCallback(async () => {
    const text = input.trim()
    if (!text || sending) return
    setInput('')
    setUserText(text)
    setSending(true)
    setAiReply('')
    try {
      const result = await App.ChatGeneral(text)
      const reply = (result as any)?.reply
      setAiReply(typeof reply === 'string' ? reply : '（无回复）')
    } catch (err: any) {
      setAiReply(`❌ 对话失败: ${err?.message || err}`)
    } finally {
      setSending(false)
    }
  }, [input, sending])

  const hasChat = !!userText || !!aiReply
  const voiceStateLabel = voice.aiSpeaking
    ? 'AI 回复中'
    : voice.listening
      ? '正在聆听'
      : voice.active
        ? '语音待命'
        : '待机'

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
        width: '100%', maxWidth: 1180, minHeight: 0, flex: 1,
      }}>
        {/* ═══ AI 中枢状态条 ═══ */}
        <div
          className="md-glass scanline-top"
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
                background: 'var(--md-sys-color-primary-container)',
                color: 'var(--md-sys-color-on-primary-container)',
                fontWeight: 500, letterSpacing: '0.04em',
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
              正中语音直连 gaea 助手 —— 轻语引擎驱动，灵感已就绪
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
              border: '1px solid var(--md-sys-color-outline-variant)',
              background: 'var(--md-sys-color-primary-container)',
              boxShadow: '0 0 14px color-mix(in srgb, var(--gaea-glow) 22%, transparent)',
              transition: 'box-shadow var(--md-sys-transition-normal), transform var(--md-sys-transition-normal)',
            }}
          >
            聊天面板 <ArrowRightOutlined style={{ fontSize: 11 }} />
          </span>
        </div>

        {/* ═══ 主区域：左卡片 | 语言交互中枢 | 右卡片 ═══ */}
        <div style={{
          flex: 1, minHeight: 0,
          display: 'grid',
          gridTemplateColumns: '1fr 1.55fr 1fr',
          gap: 20,
          alignItems: 'center',
        }}>
          <CardColumn list={leftModules} onNavigate={onNavigate} />

          {/* ── 正中：语言粒子交互中枢 ── */}
          <div
            className="md-glass-strong neon-card language-core"
            style={{
              position: 'relative',
              display: 'flex', flexDirection: 'column', alignItems: 'center',
              padding: '18px 22px 20px',
              borderRadius: 'var(--md-sys-radius-xl)',
              animation: 'launcherFadeUp 0.5s cubic-bezier(0.16, 1, 0.3, 1) 0.12s backwards',
              overflow: 'hidden',
            }}
          >
            {/* 标题行 */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
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
                直连 gaea
              </span>
            </div>

            {/* 语言粒子球 */}
            <VoiceChatOrb
              volume={voice.volume}
              listening={voice.listening}
              speaking={voice.speaking}
              aiSpeaking={voice.aiSpeaking}
              transcript={voice.transcript}
              size={286}
            />

            {/* 状态行 */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginTop: 6, minHeight: 26 }}>
              <span style={{
                fontSize: 12, fontWeight: 500, letterSpacing: '0.05em',
                color: voice.aiSpeaking ? '#64b5f6' : voice.listening ? '#ff8a65' : voice.active ? 'var(--gaea-glow)' : 'var(--md-sys-color-text-secondary)',
                textShadow: voice.active ? '0 0 10px color-mix(in srgb, var(--gaea-glow) 55%, transparent)' : 'none',
                transition: 'color 0.3s',
              }}>
                {voiceStateLabel}
              </span>
              <span style={{ width: 4, height: 4, borderRadius: '50%', background: 'var(--md-sys-color-outline)' }} />
              <span style={{ fontSize: 11, color: 'var(--md-sys-color-text-secondary)' }}>
                {voice.active ? '轻语引擎 · 自动识别' : '点击语音球下方按钮开始'}
              </span>
            </div>

            {/* 对话气泡区 */}
            <div style={{
              width: '100%', minHeight: 84, maxHeight: 150, overflowY: 'auto',
              display: 'flex', flexDirection: 'column', gap: 10, marginTop: 6,
              padding: '4px 2px',
            }}>
              {hasChat ? (
                <>
                  {userText && <ChatBubble role="user" text={userText} />}
                  {aiReply && <ChatBubble role="assistant" text={aiReply} />}
                  {sending && (
                    <div style={{ display: 'flex', gap: 8, alignItems: 'center', padding: '2px 6px' }}>
                      <span className="typing-dots"><span className="typing-dot" /><span className="typing-dot" /><span className="typing-dot" /></span>
                    </div>
                  )}
                </>
              ) : (
                <div style={{
                  flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center',
                  color: 'var(--md-sys-color-text-secondary)', fontSize: 12, opacity: 0.7,
                  textAlign: 'center', lineHeight: 1.7,
                }}>
                  语言粒子汇聚成声 ——<br />与默认平台 AI 助手 gaea 直接对话
                </div>
              )}
            </div>

            {voice.error && (
              <Typography.Text style={{ color: '#fb7185', fontSize: 12, marginTop: 4 }}>
                {voice.error}
              </Typography.Text>
            )}

            {/* 控制区：语音开关 + 打断 */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 14, marginTop: 12 }}>
              <Tooltip title={voice.active ? '结束语音对话' : '开始语音对话'}>
                <Button
                  shape="circle"
                  icon={voice.active ? <StopOutlined /> : <AudioOutlined />}
                  onClick={toggleVoice}
                  size="large"
                  style={{
                    width: 56, height: 56, fontSize: 22,
                    border: 'none',
                    color: voice.active ? '#fff' : 'var(--gaea-glow)',
                    background: voice.active
                      ? 'linear-gradient(135deg, #fb7185, #f43f5e)'
                      : 'color-mix(in srgb, var(--gaea-glow) 16%, transparent)',
                    boxShadow: voice.active
                      ? '0 0 26px rgba(244,63,94,0.55), inset 0 0 12px rgba(255,255,255,0.25)'
                      : '0 0 18px color-mix(in srgb, var(--gaea-glow) 30%, transparent)',
                    transition: 'box-shadow var(--md-sys-transition-normal), transform var(--md-sys-transition-normal), background var(--md-sys-transition-normal)',
                  }}
                  className={voice.active ? 'voice-btn-active' : ''}
                />
              </Tooltip>
              {voice.aiSpeaking && (
                <Button
                  shape="round"
                  icon={<StopOutlined />}
                  onClick={interrupt}
                  style={{
                    border: '1px solid color-mix(in srgb, #fb7185 45%, transparent)',
                    color: '#fb7185', background: 'color-mix(in srgb, #fb7185 10%, transparent)',
                    fontSize: 13,
                  }}
                >
                  打断回复
                </Button>
              )}
            </div>

            {/* 文字输入补充通道 */}
            <div style={{
              width: '100%', display: 'flex', gap: 8, marginTop: 12,
              alignItems: 'center',
            }}>
              <Input
                ref={inputRef}
                value={input}
                onChange={e => setInput(e.target.value)}
                onPressEnter={handleSend}
                placeholder="或输入文字，与 gaea 对话"
                disabled={voice.active || sending}
                variant="borderless"
                style={{
                  flex: 1, background: 'var(--md-sys-color-surface-container-high)',
                  borderRadius: 'var(--md-sys-radius-md)',
                  fontSize: 13, color: 'var(--md-sys-color-text)',
                }}
              />
              <Button
                type="primary"
                icon={<SendOutlined />}
                onClick={handleSend}
                loading={sending}
                disabled={(!input.trim() && !sending) || voice.active}
                style={{
                  background: input.trim() ? 'var(--gaea-glow)' : 'var(--md-sys-color-outline-variant)',
                  borderColor: 'transparent', borderRadius: 'var(--md-sys-radius-md)',
                  color: input.trim() ? '#042f2e' : 'var(--md-sys-color-text-secondary)',
                  boxShadow: input.trim() ? '0 0 14px color-mix(in srgb, var(--gaea-glow) 45%, transparent)' : 'none',
                  flexShrink: 0,
                }}
              />
            </div>
          </div>

          <CardColumn list={rightModules} onNavigate={onNavigate} />
        </div>
      </div>
    </div>
  )
}

export default ModuleLauncher
