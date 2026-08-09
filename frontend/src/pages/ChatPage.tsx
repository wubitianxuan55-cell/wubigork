import React, { useState, useCallback, useEffect, useRef, useMemo } from 'react'
import { Input, Button, Typography, Tooltip, Modal, message, Space } from 'antd'
import {
  SendOutlined, RobotOutlined, CopyOutlined, CheckOutlined,
  SoundOutlined, ReloadOutlined, CloseCircleOutlined,
  GlobalOutlined, SettingOutlined, ClearOutlined,
  AudioOutlined, StopOutlined, HeartOutlined, MessageOutlined, SearchOutlined,
  EditOutlined, BulbOutlined, BookOutlined, TranslationOutlined, StarFilled,
  ThunderboltOutlined, SwapOutlined,
} from '@ant-design/icons'
import * as App from '../../wailsjs/go/app/App'
import { C } from '../utils/theme'
import FeatureModelBar from '../components/FeatureModelBar'
import ChatTopicSidebar, { type Topic as SidebarTopic } from '../components/ChatTopicSidebar'
import ChatMarkdown from '../components/ChatMarkdown'
import { MarkdownContent, mdStyles } from '../components/MarkdownContent'
import { CompanionAvatar } from '../components/CompanionAvatar'
import VoiceChatOrb from '../components/VoiceChatOrb'
import VoiceSettingsPanel from '../components/VoiceSettingsPanel'
import PersonaPicker from '../components/PersonaPicker'
import { ParticleFlow } from '../components/ParticleFlow'
import { SoundWaveOverlay } from '../components/SoundWaveOverlay'
import { useVoiceChat } from '../hooks/useVoiceChat'
import { VOICE_LAUNCH_FLAG } from '../components/ModuleLauncher'
import '../chat-board.css'

interface Personality {
  id: string; label: string; gender: string; tags?: string[]
  dims: { T: number; I: number; S: number; O: number; R: number }
  voiceGuide?: string; requiresAdult18?: boolean
}

interface ChatMsg {
  key: string
  role: 'user' | 'assistant'
  content: string
  createdAt: string
  streaming?: boolean
  error?: boolean
  extra?: Record<string, any>
}

interface LegacyMsg { id?: string; role: string; content: string; timestamp?: number }
interface LegacyTopic { id?: string; title?: string; messages?: LegacyMsg[]; createdAt?: number }

// ── 存储键（旧 localStorage 话题导入 chat.db 后清理） ──
const STORAGE_KEY = 'gaea_chat_topics'
const LEGACY_STORAGE_KEY = 'wubigrok_chat_topics'
const WHISPER_TOPICS_KEY = 'gaea_whisper_topics'
const LEGACY_WHISPER_TOPICS_KEY = 'wubigrok_whisper_topics'
const PERSONALITY_KEY = 'gaea_whisper_personality'
const LEGACY_PERSONALITY_KEY = 'wubigrok_whisper_personality'
const COMPANION_SETTINGS_KEY = 'gaea_whisper_companion_settings'
const LEGACY_COMPANION_SETTINGS_KEY = 'wubigrok_whisper_companion_settings'
const ACTIVE_TOPIC_KEY = 'gaea_chat_active_topic'

// ── 快捷情绪回复（人格模式输入区 chips） ──
const QUICK_REPLIES = [
  { label: '抱抱我', text: '能抱抱我吗，今天有点累' },
  { label: '晚安', text: '晚安，做个好梦' },
  { label: '有点低落', text: '今天心情不太好，陪我聊聊' },
  { label: '分享开心事', text: '告诉你一件开心的事' },
  { label: '深入聊聊', text: '我们来深入聊聊这个话题吧' },
]

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

/** 欢迎屏建议卡：键盘可达 + 焦点可见 */
const SuggestionCard: React.FC<{ s: { icon: React.ReactNode; label: string; desc: string }; onClick: () => void }> = ({ s, onClick }) => (
  <div
    role="button"
    tabIndex={0}
    className="chat-suggestion-card"
    onClick={onClick}
    onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onClick() } }}
  >
    <div className="chat-suggestion-icon">{s.icon}</div>
    <div className="chat-suggestion-label">{s.label}</div>
    <div className="chat-suggestion-desc">{s.desc}</div>
  </div>
)

const EMO_COLORS: Record<string, string> = {
  SWEET_ATTACHMENT: '#f472b6', SHY_HEARTBEAT: '#fb7185', TSUNDERE: '#f59e0b',
  HURT_GRIEVANCE: '#a78bfa', ANGRY_ATTACK: '#ef4444', COLD_DETACHED: '#94a3b8',
  FEARFUL_OBEDIENT: '#c084fc', QUIET_FOND: '#fbbf24', CALM_RATIONAL: '#60a5fa',
}
let msgSeq = 0
function nextMsgKey(): string { msgSeq++; return `m_${msgSeq}_${Date.now()}` }
function nowStr(): string { return new Date().toISOString() }
function navigateToCharacterLib(): void {
  window.dispatchEvent(new CustomEvent('navigate', { detail: { page: 'characterlib' } }))
}
function parseExtra(raw?: string): Record<string, any> | undefined {
  if (!raw) return undefined
  try { const o = JSON.parse(raw); return typeof o === 'object' && o ? o : undefined } catch (_) { return undefined }
}

function loadPersonality(): string {
  try {
    return (localStorage.getItem(PERSONALITY_KEY) ?? localStorage.getItem(LEGACY_PERSONALITY_KEY)) || 'gaea'
  } catch (_) { return 'gaea' }
}

function loadCompanionName(personalityLabel: string): string {
  try {
    const raw = localStorage.getItem(COMPANION_SETTINGS_KEY) ?? localStorage.getItem(LEGACY_COMPANION_SETTINGS_KEY)
    if (raw) {
      const parsed = JSON.parse(raw)
      if (parsed?.companionName) return parsed.companionName
    }
  } catch (_) {}
  return personalityLabel || 'gaea'
}

/** 旧 localStorage 话题 → chat.db（一次性；成功后清理本地键） */
async function migrateLegacyTopics(): Promise<boolean> {
  const buckets: Array<{ title: string; mode: string; messages: LegacyMsg[] }> = []
  const chatRaw = localStorage.getItem(STORAGE_KEY) ?? localStorage.getItem(LEGACY_STORAGE_KEY)
  if (chatRaw) {
    try {
      const p = JSON.parse(chatRaw)
      if (Array.isArray(p)) p.forEach((t: LegacyTopic) => buckets.push({ title: t.title || '新对话', mode: 'plain', messages: t.messages || [] }))
    } catch (_) {}
  }
  const whisperRaw = localStorage.getItem(WHISPER_TOPICS_KEY) ?? localStorage.getItem(LEGACY_WHISPER_TOPICS_KEY)
  if (whisperRaw) {
    try {
      const p = JSON.parse(whisperRaw)
      if (Array.isArray(p)) p.forEach((t: LegacyTopic) => buckets.push({ title: t.title || '新对话', mode: loadPersonality(), messages: t.messages || [] }))
    } catch (_) {}
  }
  if (buckets.length === 0) return false
  for (const t of buckets) {
    const msgs = (t.messages || [])
      .filter(m => m.role === 'user' || m.role === 'assistant')
      .map(m => ({ Role: m.role, Content: typeof m.content === 'string' ? m.content : '', Extra: '' }))
    try { await App.ChatImportTopic(t.title, t.mode, msgs as any) } catch (_) {}
  }
  try {
    localStorage.removeItem(STORAGE_KEY); localStorage.removeItem(LEGACY_STORAGE_KEY)
    localStorage.removeItem(WHISPER_TOPICS_KEY); localStorage.removeItem(LEGACY_WHISPER_TOPICS_KEY)
  } catch (_) {}
  return true
}

const ChatPage: React.FC = () => {
  const [topics, setTopics] = useState<any[]>([])
  const [activeId, setActiveId] = useState<string>('')
  const [messages, setMessages] = useState<ChatMsg[]>([])
  const [initializing, setInitializing] = useState(true)
  const [sending, setSending] = useState(false)
  const [streamKey, setStreamKey] = useState<string | null>(null)
  const [streamText, setStreamText] = useState('')
  const [input, setInput] = useState('')
  const [mode, setMode] = useState<string>('plain') // 'plain' | personaID
  const [personalities, setPersonalities] = useState<Personality[]>([])
  const [activePersonality, setActivePersonality] = useState<string>(() => loadPersonality())
  // 人格元数据（只读展示，不操纵）
  const [emotion, setEmotion] = useState('')
  const [aff, setAff] = useState(0); const [aro, setAro] = useState(0)
  const [searchEnabled, setSearchEnabled] = useState(true)

  const [showVoiceSettings, setShowVoiceSettings] = useState(false)
  const [copiedId, setCopiedId] = useState<string | null>(null)
  const [speakingId, setSpeakingId] = useState<string | null>(null)
  const [voiceOn, setVoiceOn] = useState(false)

  const listRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<any>(null)
  const initRef = useRef(false)
  const topicsRef = useRef<any[]>([])
  topicsRef.current = topics
  const modeRef = useRef(mode)
  modeRef.current = mode

  // 语音角色跟随聊天模式：plain → 普通对话，其余 → 对应人格（后端持久化，首页语音保持一致）
  useEffect(() => {
    try { (App as any).VoiceApplySettings?.({ personalityPresetId: modeRef.current }) } catch (_) {}
  }, [mode])
  const activeIdRef = useRef(activeId)
  activeIdRef.current = activeId

  const currentPersonality = personalities.find(p => p.id === activePersonality)
  const companionName = useMemo(
    () => loadCompanionName(currentPersonality?.label || 'gaea'),
    [currentPersonality])
  const emoColor = EMO_COLORS[emotion] || 'var(--gaea-glow, #2dd4bf)'
  const personaLabel = currentPersonality?.label || '角色'

  // ── 语音对话（文本类：说话 → 识别文本进入聊天区） ──
  const onVoiceTranscript = useCallback((t: string) => {
    const text = (t || '').trim(); if (!text) return
    setMessages(prev => [...prev, { key: nextMsgKey(), role: 'user', content: text, createdAt: nowStr() }])
  }, [])
  const onVoiceReply = useCallback((t: string) => {
    const text = (t || '').trim(); if (!text) return
    setMessages(prev => [...prev, { key: nextMsgKey(), role: 'assistant', content: text, createdAt: nowStr() }])
  }, [])
  const { state: voice, start: startVoice, stop: stopVoice } = useVoiceChat({ onTranscript: onVoiceTranscript, onReply: onVoiceReply })

  const toggleVoice = useCallback(async () => {
    if (voiceOn) { stopVoice(); setVoiceOn(false); return }
    try { await (App as any).VoiceApplySettings?.({ personalityPresetId: modeRef.current }) } catch (_) {}
    setVoiceOn(true)
    await startVoice()
  }, [voiceOn, activePersonality, startVoice, stopVoice])

  // 首页语音入口兼容：进入聊天板块自动开启收听
  useEffect(() => {
    let flag = false
    try { flag = sessionStorage.getItem(VOICE_LAUNCH_FLAG) === '1' } catch (_) {}
    if (flag) {
      try { sessionStorage.removeItem(VOICE_LAUNCH_FLAG) } catch (_) {}
      toggleVoice()
    }
  }, [toggleVoice])

  const resetPersonaMeta = useCallback(() => {
    setEmotion(''); setAff(0); setAro(0)
  }, [])

  // ── 初始化：话题列表 + 旧数据迁移 + 人格列表 + 首页角色入口 ──
  useEffect(() => {
    if (initRef.current) return
    initRef.current = true
    ;(async () => {
      let list: any[] = []
      try { list = (await App.ChatTopicsList()) || [] } catch (_) {}
      if (list.length === 0) {
        const imported = await migrateLegacyTopics()
        try { list = (await App.ChatTopicsList()) || [] } catch (_) {}
        if (!imported && list.length === 0) {
          try { await App.ChatTopicCreate('新对话', 'plain') } catch (_) {}
          try { list = (await App.ChatTopicsList()) || [] } catch (_) {}
        }
      }
      setTopics(list)
      let first: any = list[0]
      try {
        const last = localStorage.getItem(ACTIVE_TOPIC_KEY)
        if (last) first = list.find(t => t.id === last) || first
      } catch (_) {}
      if (first) setActiveId(first.id)
      setInitializing(false)
    })()
    try { App.WhisperGetPersonalities().then((ps: any) => setPersonalities(ps || [])).catch(() => {}) } catch (_) {}
  }, [])

  // 选择话题 → 加载消息 + 恢复人格元数据
  const selectTopic = useCallback(async (id: string) => {
    if (!id || id === activeIdRef.current) return
    setActiveId(id)
    try { localStorage.setItem(ACTIVE_TOPIC_KEY, id) } catch (_) {}
    try {
      const ms = (await App.ChatMessagesList(id)) || []
      const list: ChatMsg[] = ms.map((m: any) => ({
        key: `db_${m.id}`, role: m.role === 'user' ? 'user' : 'assistant',
        content: m.content || '', createdAt: m.created_at || '', extra: parseExtra(m.extra),
      }))
      setMessages(list)
      const topic = topicsRef.current.find(t => t.id === id)
      const topicMode = topic?.mode || 'plain'
      if (topicMode !== modeRef.current) setMode(topicMode)
      resetPersonaMeta()
      if (topicMode !== 'plain') {
        const last = [...list].reverse().find(m => m.role === 'assistant' && m.extra)
        if (last?.extra) {
          if (last.extra.emotion) setEmotion(last.extra.emotion)
        }
      }
    } catch (_) { setMessages([]) }
  }, [resetPersonaMeta])

  useEffect(() => {
    if (listRef.current) listRef.current.scrollTop = listRef.current.scrollHeight
  }, [messages, streamText])

  const handleCreate = useCallback(async () => {
    try {
      const t = await App.ChatTopicCreate('新对话', modeRef.current)
      setTopics(prev => [...prev, t])
      setActiveId(t.id)
      try { localStorage.setItem(ACTIVE_TOPIC_KEY, t.id) } catch (_) {}
      setMessages([])
      resetPersonaMeta()
      inputRef.current?.focus?.()
    } catch (err: any) {
      message.error(`创建话题失败：${err?.message || err}`)
    }
  }, [resetPersonaMeta])

  const handleDelete = useCallback(async (id: string) => {
    try { await App.ChatTopicDelete(id) } catch (err: any) { message.error(`删除失败：${err?.message || err}`); return }
    const remaining = topicsRef.current.filter(t => t.id !== id)
    if (remaining.length === 0) {
      await handleCreate()
      return
    }
    setTopics(remaining)
    if (id === activeIdRef.current) {
      const idx = topicsRef.current.findIndex(t => t.id === id)
      const next = remaining[Math.min(idx, remaining.length - 1)]
      selectTopic(next.id)
    }
  }, [handleCreate, selectTopic])

  const handleRename = useCallback(async (id: string, title: string) => {
    try { await App.ChatTopicRename(id, title) } catch (_) {}
    setTopics(prev => prev.map(t => t.id === id ? { ...t, title } : t))
  }, [])

  const switchMode = useCallback(async (next: string) => {
    if (next === modeRef.current) return
    setMode(next)
    if (next === 'plain') {
      resetPersonaMeta()
    } else {
      try { localStorage.setItem(PERSONALITY_KEY, next) } catch (_) {}
    }
    if (activeIdRef.current) {
      try { await App.ChatTopicSetMode(activeIdRef.current, next) } catch (_) {}
      setTopics(prev => prev.map(t => t.id === activeIdRef.current ? { ...t, mode: next } : t))
    }
  }, [resetPersonaMeta])

  const handleSwitchPersonality = useCallback(async (id: string) => {
    try { await (App as any).WhisperClearSession(activePersonality) } catch (_) {}
    setActivePersonality(id)
    await switchMode(id)
  }, [activePersonality, switchMode])

  // 角色库切换人格 → 聊天板块联动
  useEffect(() => {
    const onPersona = (e: Event) => {
      const id = (e as CustomEvent).detail?.id
      if (!id) return
      setActivePersonality(id)
      switchMode(id)
    }
    window.addEventListener('gaea-persona-changed', onPersona)
    return () => window.removeEventListener('gaea-persona-changed', onPersona)
  }, [switchMode])

  const updateMessage = useCallback((key: string, patch: Partial<ChatMsg>) => {
    setMessages(prev => prev.map(m => m.key === key ? { ...m, ...patch } : m))
  }, [])

  const doSend = useCallback(async (text: string, retryKey?: string) => {
    const trimmed = text.trim()
    if (!trimmed || sending || !activeIdRef.current) return
    setInput(''); setSending(true)
    const um: ChatMsg = { key: nextMsgKey(), role: 'user', content: trimmed, createdAt: nowStr() }
    const am: ChatMsg = { key: nextMsgKey(), role: 'assistant', content: '', streaming: true, createdAt: nowStr() }
    setMessages(prev => {
      if (!retryKey) return [...prev, um, am]
      // 重试：失败消息未落库，其前置用户消息同样未落库，一并移除保持与 DB 一致
      const errIdx = prev.findIndex(m => m.key === retryKey)
      let drop = new Set<string>([retryKey])
      if (errIdx >= 0) {
        const userMsg = prev.slice(0, errIdx).reverse().find(m => m.role === 'user')
        if (userMsg) drop.add(userMsg.key)
      }
      return [...prev.filter(m => !drop.has(m.key)), um, am]
    })
    setStreamKey(am.key); setStreamText('')
    const active = activeIdRef.current
    const curMode = modeRef.current
    try {
      const res: any = await App.ChatSend(active, trimmed, curMode)
      const reply = typeof res?.reply === 'string' ? res.reply : ''
      const reduced = typeof window !== 'undefined' && !!window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
      if (!reduced && reply.length > 40) {
        const step = Math.max(2, Math.round(reply.length / 180))
        for (let i = 0; i <= reply.length; i += step) {
          setStreamText(reply.slice(0, i))
          await new Promise(r => setTimeout(r, 14))
        }
      }
      setStreamText(''); setStreamKey(null)
      const extra: Record<string, any> = {}
      if (res.emotion) extra.emotion = res.emotion
      updateMessage(am.key, { content: reply, streaming: false, extra })
      if (res.emotion) setEmotion(res.emotion)
      if (typeof res.aff === 'number') setAff(Math.round(res.aff))
      if (typeof res.aro === 'number') setAro(Math.round(res.aro))
      // 自动命名
      const topic = topicsRef.current.find(t => t.id === active)
      if (topic?.title === '新对话') {
        const title = trimmed.slice(0, 20) + (trimmed.length > 20 ? '…' : '')
        try { await App.ChatTopicRename(active, title) } catch (_) {}
        setTopics(prev => prev.map(t => t.id === active ? { ...t, title } : t))
      }
    } catch (err: any) {
      setStreamText(''); setStreamKey(null)
      updateMessage(am.key, { content: `请求失败：${err?.message || String(err)}`, streaming: false, error: true })
    } finally { setSending(false) }
  }, [sending, updateMessage])

  const handleSend = useCallback(() => { doSend(input) }, [input, doSend])
  const handleSuggestion = useCallback((label: string) => { doSend(label) }, [doSend])
  const handleFillInput = useCallback((label: string) => { setInput(label); inputRef.current?.focus?.() }, [])
  const handleRetry = useCallback((msgKey: string) => {
    const idx = messages.findIndex(m => m.key === msgKey)
    if (idx < 0) return
    const userMsg = messages.slice(0, idx).reverse().find(m => m.role === 'user')
    if (userMsg) doSend(userMsg.content, msgKey)
  }, [messages, doSend])

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend() }
  }

  const handleCopy = async (content: string, id: string) => {
    try { await navigator.clipboard.writeText(content) } catch {
      const ta = document.createElement('textarea'); ta.value = content; document.body.appendChild(ta); ta.select()
      document.execCommand('copy'); document.body.removeChild(ta)
    }
    setCopiedId(id); setTimeout(() => setCopiedId(null), 2000)
  }

  const handleSpeak = async (content: string, id: string) => {
    if (speakingId) return; setSpeakingId(id)
    try {
      const result: any = await App.TTSSpeakBase64(content)
      if (result?.base64) {
        const b = atob(result.base64); const bytes = new Uint8Array(b.length)
        for (let i = 0; i < b.length; i++) bytes[i] = b.charCodeAt(i)
        const audio = new Audio(URL.createObjectURL(new Blob([bytes], { type: result.mimeType || 'audio/mp3' })))
        audio.onended = () => setSpeakingId(null)
        audio.onerror = () => { setSpeakingId(null); message.error('播放失败') }
        await audio.play()
        return
      }
      message.warning('TTS 未返回音频数据')
    } catch (err: any) { message.error(`朗读失败：${typeof err === 'string' ? err : err?.message || '未知错误'}`) }
    setSpeakingId(null)
  }

  const handleClearMessages = useCallback(async () => {
    setMessages([]); resetPersonaMeta()
    if (activeIdRef.current) {
      try { await App.ChatTopicClear(activeIdRef.current) } catch (_) {}
    }
    if (modeRef.current !== 'plain') {
      try { await (App as any).WhisperClearSession(modeRef.current) } catch (_) {}
    }
  }, [resetPersonaMeta])

  const hasMessages = messages.length > 0

  const topicList: SidebarTopic[] = topics.map(t => ({
    id: t.id, title: t.title, createdAt: new Date(t.created_at || 0).getTime() || Date.now(),
    mode: t.mode, modeLabel: t.mode === 'plain' ? '' : (personalities.find(p => p.id === t.mode)?.label || '角色'),
    preview: t.preview || '',
  }))

  return (
    <div className="chat-board" style={{ flex: 1, display: 'flex', flexDirection: 'row', minHeight: 0, position: 'relative' }}>
      <ChatTopicSidebar
        topics={topicList}
        activeId={activeId}
        onSelect={selectTopic}
        onCreate={handleCreate}
        onDelete={handleDelete}
        onRename={handleRename}
      />

      <main style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0, minHeight: 0, overflow: 'hidden', position: 'relative', background: 'transparent' }}>
        <style>{mdStyles}</style>
        {mode !== 'plain' && <ParticleFlow aro={aro} />}
        <SoundWaveOverlay active={speakingId !== null} aff={aff} aro={aro} />

        {/* 模式切换条 */}
        <div className="chat-mode-bar">
          <div className="chat-mode-seg" role="tablist" aria-label="对话模式">
            <button role="tab" aria-selected={mode === 'plain'} className={mode === 'plain' ? 'active' : ''}
              onClick={() => switchMode('plain')}>
              <MessageOutlined style={{ fontSize: 12 }} /> 普通对话
            </button>
            <button role="tab" aria-selected={mode !== 'plain'} className={mode !== 'plain' ? 'active' : ''}
              onClick={() => { if (mode === 'plain') switchMode(activePersonality) }}>
              <HeartOutlined style={{ fontSize: 12 }} /> 角色 · {mode !== 'plain' ? personaLabel : (currentPersonality?.label || '人格')}
            </button>
          </div>
          <div style={{ flex: 1 }} />
          {mode !== 'plain' ? (
            <Space size={2}>
              <Tooltip title={searchEnabled ? '联网搜索已开启（自动检测搜索意图）' : '联网搜索已关闭'}>
                <Button type="text" size="small" icon={<GlobalOutlined style={{ color: searchEnabled ? '#52c41a' : C('color-text-secondary') }} />}
                  onClick={() => setSearchEnabled(!searchEnabled)} style={{ padding: '0 4px', height: 24, opacity: searchEnabled ? 1 : 0.5 }} />
              </Tooltip>
              <Tooltip title="角色库管理">
                <Button type="text" size="small" icon={<SettingOutlined />} onClick={navigateToCharacterLib}
                  style={{ color: C('color-text-secondary'), height: 24 }} />
              </Tooltip>
              <PersonaPicker activeId={mode !== 'plain' ? mode : activePersonality}
                onSelect={handleSwitchPersonality} onManage={navigateToCharacterLib}>
                <Tooltip title="切换角色">
                  <Button type="text" size="small" icon={<SwapOutlined />}
                    style={{ color: C('color-text-secondary'), height: 24 }} />
                </Tooltip>
              </PersonaPicker>
              <Tooltip title="语音设置">
                <Button type="text" size="small" icon={<SoundOutlined />} onClick={() => setShowVoiceSettings(true)}
                  style={{ color: C('color-text-secondary'), height: 24 }} />
              </Tooltip>
              {hasMessages && (
                <Tooltip title="清空当前对话">
                  <Button type="text" size="small" icon={<ClearOutlined />} onClick={handleClearMessages}
                    style={{ color: C('color-text-secondary'), height: 24 }} />
                </Tooltip>
              )}
            </Space>
          ) : (
            <Tooltip title="清空当前对话">
              <Button type="text" size="small" icon={<ClearOutlined />} onClick={handleClearMessages}
                style={{ color: C('color-text-secondary'), height: 24, opacity: hasMessages ? 1 : 0.35 }} />
            </Tooltip>
          )}
        </div>

        {/* 人格状态条（临场感：头像常驻 + 名字；状态/记忆归角色库） */}
        {mode !== 'plain' && hasMessages && (
          <div className="chat-persona-bar">
            <CompanionAvatar size={46} state={speakingId ? 'speaking' : sending ? 'thinking' : 'idle'} emotionColor={emoColor} />
            <div style={{ flex: 1, minWidth: 0 }}>
              <div className="chat-persona-meta">
                <Typography.Text strong style={{ fontSize: 14, color: C('color-text') }}>{companionName}</Typography.Text>
                <span className="chat-chip" style={{ color: 'var(--gaea-glow)', borderColor: 'color-mix(in srgb, var(--gaea-glow) 30%, transparent)' }}>
                  <span style={{ width: 6, height: 6, borderRadius: '50%', background: 'var(--gaea-glow)' }} />
                  AI 陪伴
                </span>
              </div>
            </div>
            <PersonaPicker activeId={mode !== 'plain' ? mode : activePersonality}
              onSelect={handleSwitchPersonality} onManage={navigateToCharacterLib}>
              <Button type="primary" size="small" icon={<SwapOutlined />}
                style={{ borderRadius: 16, height: 30, fontSize: 12, flexShrink: 0 }}>
                选择角色
              </Button>
            </PersonaPicker>
          </div>
        )}

        {/* 消息区 */}
        <div ref={listRef} role="log" aria-live="polite" aria-label="对话消息"
          style={{ flex: 1, minHeight: 0, overflow: 'auto', padding: 0, position: 'relative' }}>
          {initializing ? (
            <div className="chat-empty">
              <span className="typing-dots"><span className="typing-dot" /><span className="typing-dot" /><span className="typing-dot" /></span>
              <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, marginTop: 14 }}>正在载入会话…</Typography.Text>
            </div>
          ) : !hasMessages ? (
            mode !== 'plain' ? (
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
                  onSelect={handleSwitchPersonality} onManage={navigateToCharacterLib}>
                  <Button type="primary" icon={<SwapOutlined />} style={{ marginTop: 14, marginBottom: 16, borderRadius: 20, padding: '4px 22px', height: 36, fontSize: 13 }}>
                    选择角色
                  </Button>
                </PersonaPicker>
                <div className="chat-suggestion-grid">
                  {PERSONA_SUGGESTIONS.map(s => (
                    <SuggestionCard key={s.label} s={s} onClick={() => handleFillInput(s.label)} />
                  ))}
                </div>
                <div style={{ marginTop: 10 }}>
                  <Button type="link" size="small" icon={<SettingOutlined />} onClick={navigateToCharacterLib}
                    style={{ color: C('color-text-secondary'), fontSize: 11.5 }}>
                    去角色库管理角色
                  </Button>
                </div>
              </div>
            ) : (
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
                    <SuggestionCard key={s.label} s={s} onClick={() => handleSuggestion(s.label)} />
                  ))}
                </div>
              </div>
            )
          ) : (
            <div className="chat-flow">
              {messages.map((msg, idx) => {
                const isUser = msg.role === 'user'
                const isStreaming = msg.streaming && msg.key === streamKey
                const display = isStreaming ? streamText : msg.content
                const prev = messages[idx - 1]
                const newGroup = !prev || prev.role !== msg.role
                return isUser ? (
                  <div key={msg.key} className="chat-row chat-row-user">
                    <div className="chat-user-capsule">{display}</div>
                    {msg.content && !msg.streaming && (
                      <div className="chat-msg-actions">
                        <Tooltip title={copiedId === msg.key ? '已复制' : '复制'}>
                          <Button type="text" size="small" icon={copiedId === msg.key ? <CheckOutlined style={{ color: '#52c41a' }} /> : <CopyOutlined />}
                            onClick={() => handleCopy(msg.content, msg.key)} style={{ color: C('color-text-secondary'), fontSize: 12, padding: '0 4px', height: 22 }} />
                        </Tooltip>
                      </div>
                    )}
                  </div>
                ) : (
                  <div key={msg.key} className="chat-row chat-row-assistant">
                    {newGroup && (
                      <div className="chat-assistant-name">
                        <span className="ai-dot" />
                        {mode === 'plain' ? 'gaea AI 助手' : `${companionName} · AI`}
                      </div>
                    )}
                    {msg.error ? (
                      <div className="chat-error-block">
                        <CloseCircleOutlined style={{ marginTop: 2, flexShrink: 0 }} />
                        <div style={{ flex: 1, minWidth: 0 }}>
                          {display}
                          <div style={{ marginTop: 8 }}>
                            <Button size="small" icon={<ReloadOutlined />} onClick={() => handleRetry(msg.key)}
                              style={{ fontSize: 12, height: 26, borderRadius: 8 }}>
                              重试
                            </Button>
                          </div>
                        </div>
                      </div>
                    ) : (
                      <div className="chat-assistant-text">
                        {isStreaming ? (
                          <span className="chat-streaming">
                            {display ? <><MarkdownContent source={display} className="md-content" /><span className="cursor-blink" /></> : <span className="typing-dots"><span className="typing-dot" /><span className="typing-dot" /><span className="typing-dot" /></span>}
                          </span>
                        ) : (
                          mode === 'plain'
                            ? <ChatMarkdown text={display} />
                            : <MarkdownContent source={display} className="md-content" />
                        )}
                      </div>
                    )}
                    {msg.content && !msg.streaming && (
                      <div className="chat-msg-actions">
                        <Tooltip title={copiedId === msg.key ? '已复制' : '复制'}>
                          <Button type="text" size="small" icon={copiedId === msg.key ? <CheckOutlined style={{ color: '#52c41a' }} /> : <CopyOutlined />}
                            onClick={() => handleCopy(msg.content, msg.key)} style={{ color: C('color-text-secondary'), fontSize: 12, padding: '0 4px', height: 22 }} />
                        </Tooltip>
                        {!msg.error && (
                          <Tooltip title={speakingId === msg.key ? '朗读中…' : '朗读'}>
                            <Button type="text" size="small" icon={<SoundOutlined />} loading={speakingId === msg.key}
                              onClick={() => handleSpeak(msg.content, msg.key)} style={{ color: C('color-text-secondary'), fontSize: 12, padding: '0 4px', height: 22 }} />
                          </Tooltip>
                        )}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </div>

        {/* 输入岛 */}
        <div className="chat-composer-wrap">
          {mode !== 'plain' && (
            <div className="chat-quick-replies">
              {QUICK_REPLIES.map(q => (
                <button key={q.label} className="chat-quick-chip" onClick={() => handleFillInput(q.text)}>{q.label}</button>
              ))}
            </div>
          )}
          <div className="gaea-glass-shell chat-composer">
            <Tooltip title={voiceOn ? '结束聆听' : '语音输入（说话识别为文本对话）'}>
              <Button type="text" icon={voiceOn ? <StopOutlined /> : <AudioOutlined />}
                onClick={toggleVoice}
                style={{ color: voiceOn ? 'var(--gaea-glow)' : C('color-text-secondary'), borderRadius: 10, width: 36, height: 36, minWidth: 36, padding: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', background: voiceOn ? 'color-mix(in srgb, var(--gaea-glow) 12%, transparent)' : 'transparent', flexShrink: 0, fontSize: 15 }} />
            </Tooltip>
            <Input.TextArea
              ref={inputRef}
              value={input}
              onChange={e => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={voiceOn ? (voice.transcript || '正在聆听…请说话') : '输入消息，Enter 发送 / Shift+Enter 换行'}
              disabled={sending || voiceOn}
              autoSize={{ minRows: 1, maxRows: 6 }}
              className="chat-input-textarea"
              style={{ flex: 1, background: 'transparent', border: 'none', color: C('color-text'), borderRadius: 0, resize: 'none', fontSize: 14, lineHeight: 1.6, padding: '6px 2px', boxShadow: 'none' }}
            />
            <Tooltip title="发送 (Enter)">
              <Button type="primary" icon={<SendOutlined />} onClick={handleSend} loading={sending} disabled={!input.trim() || voiceOn}
                style={{ background: input.trim() ? 'var(--md-sys-color-primary)' : C('color-border'), borderColor: 'transparent', borderRadius: 14, width: 40, height: 40, minWidth: 40, padding: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: input.trim() ? '0 0 16px color-mix(in srgb, var(--gaea-glow) 40%, transparent)' : 'none', flexShrink: 0 }} />
            </Tooltip>
          </div>
        </div>
      </main>

      {/* 绑定模型条（聊天板块统一入口；whisper 为 chat 别名） */}
      <div style={{ position: 'absolute', left: 12, bottom: 12, zIndex: 50 }}>
        <FeatureModelBar feature="chat" label="聊天" />
      </div>

      <Modal title="语音设置" open={showVoiceSettings} onCancel={() => setShowVoiceSettings(false)} footer={null} width={480} centered destroyOnClose>
        <VoiceSettingsPanel />
      </Modal>

    </div>
  )
}

export default ChatPage
