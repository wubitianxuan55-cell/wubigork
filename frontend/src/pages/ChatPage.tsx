import React, { useState, useCallback, useEffect, useRef, useMemo } from 'react'
import { Input, Button, Typography, Tooltip, Modal, Tabs, Tag, message, Space } from 'antd'
import {
  SendOutlined, RobotOutlined, CopyOutlined, CheckOutlined,
  SoundOutlined, ReloadOutlined, CloseCircleOutlined,
  GlobalOutlined, SettingOutlined, ClearOutlined, MenuFoldOutlined, MenuUnfoldOutlined,
  AudioOutlined, StopOutlined, HeartOutlined, MessageOutlined, SearchOutlined,
  EditOutlined, BulbOutlined, BookOutlined, TranslationOutlined, StarFilled,
  ThunderboltOutlined, InboxOutlined,
} from '@ant-design/icons'
import * as App from '../../wailsjs/go/app/App'
import { C } from '../utils/theme'
import FeatureModelBar from '../components/FeatureModelBar'
import ChatTopicSidebar, { type Topic as SidebarTopic } from '../components/ChatTopicSidebar'
import ChatMarkdown from '../components/ChatMarkdown'
import { MarkdownContent, mdStyles } from '../components/MarkdownContent'
import { CompanionAvatar } from '../components/CompanionAvatar'
import { WhisperEmotionPanel } from '../components/WhisperEmotionPanel'
import WhisperDesirePanel from '../components/WhisperDesirePanel'
import WhisperTracePanel from '../components/WhisperTracePanel'
import WhisperMemoryModal from '../components/WhisperMemoryModal'
import VoiceSettingsPanel from '../components/VoiceSettingsPanel'
import { ParticleFlow } from '../components/ParticleFlow'
import { SoundWaveOverlay } from '../components/SoundWaveOverlay'
import { useVoiceChat } from '../hooks/useVoiceChat'
import { VOICE_LAUNCH_FLAG } from '../components/ModuleLauncher'
import { requestPersonaEnter, consumePersonaEnter } from '../utils/chatNav'
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

// ── 记忆分类映射（对齐后端 memory_taxonomy.go 6 domain） ──
const DOMAIN_LABELS: Record<string, string> = {
  IDENTITY: '身份', SOCIAL: '社交', DAILY_LIFE: '日常',
  PURSUITS: '追求', INNER_WORLD: '内心', TEMPORAL: '时间',
}
const DOMAIN_ORDER = ['IDENTITY', 'SOCIAL', 'DAILY_LIFE', 'PURSUITS', 'INNER_WORLD', 'TEMPORAL']
const SUB_LABELS: Record<string, string> = {
  BASIC_PROFILE: '基本信息', LIFE_STORY: '人生故事', VALUES_BELIEFS: '价值观', SELF_PERCEPTION: '自我认知',
  OUR_BOND: '我们的羁绊', FAMILY: '家庭', FRIENDS: '朋友', PARTNER: '伴侣',
  ROUTINES: '日常习惯', HEALTH: '健康', LIVING_SPACE: '居住', LIFESTYLE: '生活方式',
  CAREER: '职业', LEARNING: '学习', GOALS: '目标', PROJECTS: '项目', PROCEDURES: '流程',
  MOOD: '情绪', TASTES: '品味', VULNERABILITIES: '脆弱面', INSIDE_JOKES: '内部梗',
  NOW: '当下', COMMITMENTS: '承诺', PLANS: '计划', WORLD: '世界观',
}

const EMO_COLORS: Record<string, string> = {
  SWEET_ATTACHMENT: '#f472b6', SHY_HEARTBEAT: '#fb7185', TSUNDERE: '#f59e0b',
  HURT_GRIEVANCE: '#a78bfa', ANGRY_ATTACK: '#ef4444', COLD_DETACHED: '#94a3b8',
  FEARFUL_OBEDIENT: '#c084fc', QUIET_FOND: '#fbbf24', CALM_RATIONAL: '#60a5fa',
}
const EMO_LABELS: Record<string, string> = {
  SWEET_ATTACHMENT: '甜蜜依恋', SHY_HEARTBEAT: '害羞心动', TSUNDERE: '傲娇',
  HURT_GRIEVANCE: '委屈受伤', ANGRY_ATTACK: '愤怒反击', COLD_DETACHED: '冷淡疏离',
  FEARFUL_OBEDIENT: '不安顺从', QUIET_FOND: '安静的喜欢', CALM_RATIONAL: '平静理性',
}
const STAGE_LABELS: Record<string, string> = {
  INTIMATE: '亲密', FAMILIAR: '熟悉', STRANGER: '初识',
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

/** 右侧记忆抽屉（人格模式） */
const MemoryDrawer: React.FC<{
  facts: any[]; personalityLabel: string
  search: string; onSearch: (v: string) => void
  collapsed: Set<string>; onToggle: (d: string) => void
  onOpenPage: () => void
}> = ({ facts, personalityLabel, search, onSearch, collapsed, onToggle, onOpenPage }) => {
  const filtered = facts.filter((f: any) =>
    !search ||
    String(f.subject || '').toLowerCase().includes(search.toLowerCase()) ||
    String(f.summary || '').toLowerCase().includes(search.toLowerCase()))
  const grouped = DOMAIN_ORDER.map(d => ({
    domain: d,
    label: DOMAIN_LABELS[d] || d,
    facts: filtered.filter((f: any) => f.domain === d || f.domain === d.toLowerCase()),
  })).filter(g => g.facts.length > 0)

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', gap: 4, padding: '2px 8px 8px' }}>
      <Input
        prefix={<SearchOutlined />} size="small" placeholder="搜索记忆"
        value={search} onChange={e => onSearch(e.target.value)} allowClear
        style={{ borderRadius: 8, fontSize: 11 }}
      />
      <div style={{ flex: 1, overflow: 'auto', minHeight: 0 }}>
        {facts.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 26, color: C('color-text-secondary'), fontSize: 12, lineHeight: 1.7 }}>
            <InboxOutlined style={{ fontSize: 20, opacity: 0.5, marginBottom: 8, display: 'block' }} />
            还没有记忆，多聊几句吧
          </div>
        ) : filtered.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 26, color: C('color-text-secondary'), fontSize: 12 }}>无匹配</div>
        ) : (
          grouped.map(g => {
            const isCollapsed = collapsed.has(g.domain)
            const coreCount = g.facts.filter((f: any) => f.tier === 'core').length
            return (
              <div key={g.domain} style={{ marginBottom: 2 }}>
                <div
                  role="button" tabIndex={0}
                  onClick={() => onToggle(g.domain)}
                  onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onToggle(g.domain) } }}
                  style={{
                    display: 'flex', alignItems: 'center', gap: 4, padding: '6px',
                    borderRadius: 8, cursor: 'pointer', userSelect: 'none',
                    background: isCollapsed ? 'transparent' : `${C('bg-elevated')}80`,
                    transition: 'background 150ms',
                  }}
                >
                  <span style={{ fontSize: 10, transition: 'transform 200ms', transform: isCollapsed ? 'rotate(-90deg)' : 'rotate(0deg)' }}>▼</span>
                  <span style={{ fontSize: 12, fontWeight: 600, color: C('color-text'), flex: 1 }}>{g.label}</span>
                  <Tag style={{ fontSize: 9, margin: 0, padding: '0 5px', lineHeight: '16px', background: 'transparent', border: '1px solid var(--md-sys-color-outline-variant)', color: C('color-text-secondary') }}>
                    {g.facts.length}
                  </Tag>
                  {coreCount > 0 && <StarFilled style={{ fontSize: 9, color: '#faad14' }} />}
                </div>
                {!isCollapsed && g.facts.map((f: any) => (
                  <div
                    key={f.id}
                    role="button" tabIndex={0}
                    onClick={onOpenPage}
                    onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onOpenPage() } }}
                    style={{
                      padding: '6px 8px 6px 18px', margin: '1px 0', borderRadius: 8, cursor: 'pointer',
                      background: f.tier === 'core' ? `${C('color-primary')}06` : 'transparent',
                      borderLeft: f.tier === 'core' ? `2px solid #faad14` : '2px solid transparent',
                      transition: 'background 150ms',
                    }}
                  >
                    <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                      {f.tier === 'core' && <StarFilled style={{ color: '#faad14', fontSize: 9 }} />}
                      <span style={{ fontSize: 11, fontWeight: 600, color: C('color-text'), flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {f.subject}
                      </span>
                      <span style={{ fontSize: 9, color: C('color-text-secondary'), opacity: 0.55, flexShrink: 0 }}>
                        {SUB_LABELS[f.subcategory] || f.subcategory || ''}
                      </span>
                    </div>
                    <div style={{ fontSize: 9, color: C('color-text-secondary'), marginTop: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', opacity: 0.7 }}>
                      {String(f.summary || '').slice(0, 50)}{f.summary?.length > 50 ? '…' : ''}
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 2 }}>
                      <span style={{ fontSize: 8, color: C('color-text-secondary'), opacity: 0.55 }}>W{f.weight?.toFixed?.(1) ?? '–'}</span>
                      {f.emotionalContext?.valence != null && (
                        <span style={{ fontSize: 8, color: f.emotionalContext.valence > 0.2 ? '#52c41a' : f.emotionalContext.valence < -0.2 ? '#ff4d4f' : '#8c8c8c' }}>
                          {f.emotionalContext.valence > 0.2 ? '正' : f.emotionalContext.valence < -0.2 ? '负' : '平'}
                        </span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )
          })
        )}
      </div>
      <Typography.Text style={{ fontSize: 10, color: C('color-text-secondary'), opacity: 0.6, textAlign: 'center', paddingBottom: 4 }}>
        {personalityLabel} · 只读记忆
      </Typography.Text>
    </div>
  )
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
  const [stage, setStage] = useState('')
  const [trust, setTrust] = useState(50)
  const [rifts, setRifts] = useState(0)
  const [aff, setAff] = useState(0); const [sec, setSec] = useState(0); const [aro, setAro] = useState(0); const [dom, setDom] = useState(0)
  const [totalTurns, setTotalTurns] = useState(0)
  const [facts, setFacts] = useState<any[]>([])
  const [traces, setTraces] = useState<any[]>([])
  const [desireSlots, setDesireSlots] = useState<any[]>([])
  const [sharedEvents, setSharedEvents] = useState(0)
  const [searchEnabled, setSearchEnabled] = useState(true)

  const [drawerOpen, setDrawerOpen] = useState(false)
  const [drawerTab, setDrawerTab] = useState('status')
  const [memorySearch, setMemorySearch] = useState('')
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(new Set())
  const [showMemoryPage, setShowMemoryPage] = useState(false)
  const [showVoiceSettings, setShowVoiceSettings] = useState(false)
  const [copiedId, setCopiedId] = useState<string | null>(null)
  const [speakingId, setSpeakingId] = useState<string | null>(null)
  const [voiceOn, setVoiceOn] = useState(false)

  const listRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<any>(null)
  const initRef = useRef(false)
  const pendingPersonaRef = useRef(false)
  const topicsRef = useRef<any[]>([])
  topicsRef.current = topics
  const modeRef = useRef(mode)
  modeRef.current = mode
  const activeIdRef = useRef(activeId)
  activeIdRef.current = activeId

  const currentPersonality = personalities.find(p => p.id === activePersonality)
  const companionName = useMemo(
    () => loadCompanionName(currentPersonality?.label || 'gaea'),
    [currentPersonality])
  const emoColor = EMO_COLORS[emotion] || 'var(--gaea-glow, #2dd4bf)'
  const personaLabel = currentPersonality?.label || '轻语'

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
    try { await (App as any).VoiceApplySettings?.({ personalityPresetId: modeRef.current !== 'plain' ? modeRef.current : activePersonality }) } catch (_) {}
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
    setEmotion(''); setStage(''); setTrust(50); setRifts(0)
    setAff(0); setSec(0); setAro(0); setDom(0); setTotalTurns(0)
    setFacts([]); setTraces([]); setDesireSlots([]); setSharedEvents(0)
  }, [])

  const loadFacts = useCallback(async (personaId: string) => {
    try { const f = await App.WhisperGetFacts(personaId); setFacts(Array.isArray(f) ? f : []) } catch (_) {}
  }, [])

  // ── 初始化：话题列表 + 旧数据迁移 + 人格列表 + 首页轻语入口 ──
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
        loadFacts(topicMode)
        const last = [...list].reverse().find(m => m.role === 'assistant' && m.extra)
        if (last?.extra) {
          if (last.extra.emotion) setEmotion(last.extra.emotion)
          if (last.extra.stage) setStage(last.extra.stage)
          if (typeof last.extra.trust === 'number') setTrust(Math.round(last.extra.trust))
          if (typeof last.extra.totalTurns === 'number') setTotalTurns(last.extra.totalTurns)
        }
      }
    } catch (_) { setMessages([]) }
  }, [loadFacts, resetPersonaMeta])

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
      loadFacts(next)
      setDrawerOpen(true)
    }
    if (activeIdRef.current) {
      try { await App.ChatTopicSetMode(activeIdRef.current, next) } catch (_) {}
      setTopics(prev => prev.map(t => t.id === activeIdRef.current ? { ...t, mode: next } : t))
    }
  }, [loadFacts, resetPersonaMeta])

  // 首页「轻语」卡片 → persona 模式（事件可能早于会话加载完成）
  useEffect(() => {
    const enterPersona = () => {
      if (topics.length > 0 && activeId) {
        switchMode(activePersonality)
        setDrawerOpen(true)
      } else {
        pendingPersonaRef.current = true
      }
    }
    if (consumePersonaEnter()) enterPersona()
    window.addEventListener('gaea-chat-persona-enter', enterPersona)
    return () => window.removeEventListener('gaea-chat-persona-enter', enterPersona)
  }, [topics.length, activeId, activePersonality, switchMode])

  // 会话加载完成后补执行轻语入口请求（避免被话题模式覆盖）
  useEffect(() => {
    if (!pendingPersonaRef.current || topics.length === 0 || !activeId) return
    pendingPersonaRef.current = false
    switchMode(activePersonality)
    setDrawerOpen(true)
  }, [topics, activeId, activePersonality, switchMode])

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
      loadFacts(id)
      switchMode(id)
    }
    window.addEventListener('gaea-persona-changed', onPersona)
    return () => window.removeEventListener('gaea-persona-changed', onPersona)
  }, [loadFacts, switchMode])

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
      if (typeof res.trust === 'number') extra.trust = res.trust
      if (res.stage) extra.stage = res.stage
      if (typeof res.totalTurns === 'number') extra.totalTurns = res.totalTurns
      updateMessage(am.key, { content: reply, streaming: false, extra })
      if (res.emotion) setEmotion(res.emotion)
      if (res.stage) setStage(res.stage)
      if (typeof res.trust === 'number') setTrust(Math.round(res.trust))
      if (typeof res.aff === 'number') setAff(Math.round(res.aff))
      if (typeof res.sec === 'number') setSec(Math.round(res.sec))
      if (typeof res.aro === 'number') setAro(Math.round(res.aro))
      if (typeof res.dom === 'number') setDom(Math.round(res.dom))
      if (typeof res.rifts === 'number') setRifts(res.rifts)
      if (typeof res.totalTurns === 'number') setTotalTurns(res.totalTurns)
      if (res.desireSlots) setDesireSlots(res.desireSlots)
      if (res.trace) setTraces(prev => [...prev, res.trace])
      if (res.facts) setFacts(res.facts)
      if (typeof res.sharedEvents === 'number') setSharedEvents(res.sharedEvents)
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
    mode: t.mode, modeLabel: t.mode === 'plain' ? '' : (personalities.find(p => p.id === t.mode)?.label || '轻语'),
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
              <HeartOutlined style={{ fontSize: 12 }} /> 轻语 · {mode !== 'plain' ? personaLabel : (currentPersonality?.label || '人格')}
            </button>
          </div>
          <div style={{ flex: 1 }} />
          {mode !== 'plain' ? (
            <Space size={2}>
              <Tooltip title={searchEnabled ? '联网搜索已开启（自动检测搜索意图）' : '联网搜索已关闭'}>
                <Button type="text" size="small" icon={<GlobalOutlined style={{ color: searchEnabled ? '#52c41a' : C('color-text-secondary') }} />}
                  onClick={() => setSearchEnabled(!searchEnabled)} style={{ padding: '0 4px', height: 24, opacity: searchEnabled ? 1 : 0.5 }} />
              </Tooltip>
              <Tooltip title="虚拟助手管理">
                <Button type="text" size="small" icon={<SettingOutlined />} onClick={navigateToCharacterLib}
                  style={{ color: C('color-text-secondary'), height: 24 }} />
              </Tooltip>
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
              <Tooltip title={drawerOpen ? '折叠记忆抽屉' : '展开记忆抽屉'}>
                <Button type="text" size="small" icon={drawerOpen ? <MenuFoldOutlined /> : <MenuUnfoldOutlined />}
                  onClick={() => setDrawerOpen(!drawerOpen)} style={{ color: C('color-text-secondary'), height: 24 }} />
              </Tooltip>
            </Space>
          ) : (
            <Tooltip title="清空当前对话">
              <Button type="text" size="small" icon={<ClearOutlined />} onClick={handleClearMessages}
                style={{ color: C('color-text-secondary'), height: 24, opacity: hasMessages ? 1 : 0.35 }} />
            </Tooltip>
          )}
        </div>

        {/* 人格状态条（临场感：头像常驻 + 只读情绪/信任/轮次） */}
        {mode !== 'plain' && (
          <div className="chat-persona-bar">
            <CompanionAvatar size={46} state={speakingId ? 'speaking' : sending ? 'thinking' : 'idle'} emotionColor={emoColor} />
            <div style={{ flex: 1, minWidth: 0 }}>
              <div className="chat-persona-meta">
                <Typography.Text strong style={{ fontSize: 14, color: C('color-text') }}>{companionName}</Typography.Text>
                <span className="chat-chip" style={{ color: 'var(--gaea-glow)', borderColor: 'color-mix(in srgb, var(--gaea-glow) 30%, transparent)' }}>
                  <span style={{ width: 6, height: 6, borderRadius: '50%', background: 'var(--gaea-glow)' }} />
                  AI 陪伴
                </span>
                <span className="chat-chip">{STAGE_LABELS[stage] || '初识'}</span>
                {emotion && (
                  <span className="chat-chip" style={{ color: emoColor, borderColor: `${emoColor}44`, background: `${emoColor}14` }}>
                    {EMO_LABELS[emotion] || emotion}
                  </span>
                )}
              </div>
              <div className="chat-trust-track">
                <span style={{ fontSize: 10, color: C('color-text-secondary'), flexShrink: 0 }}>信任</span>
                <div className="chat-trust-bar"><div className="chat-trust-fill" style={{ width: `${Math.min(100, Math.max(0, trust))}%` }} /></div>
                <span style={{ fontSize: 10, fontWeight: 600, color: 'var(--gaea-glow)', flexShrink: 0 }}>{trust}</span>
                {rifts > 0 && <span style={{ fontSize: 10, color: 'var(--whisper-rift)', flexShrink: 0 }}>裂痕 {rifts}</span>}
                <span style={{ fontSize: 10, color: C('color-text-secondary'), flexShrink: 0, marginLeft: 2 }}>对话 {totalTurns} 轮</span>
              </div>
            </div>
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
              <div className="chat-empty">
                <CompanionAvatar size={112} state="idle" emotionColor={emoColor} />
                <h2>{companionName}</h2>
                <p>我是{personaLabel}，今天想聊点什么？</p>
                <div className="chat-suggestion-grid">
                  {PERSONA_SUGGESTIONS.map(s => (
                    <div key={s.label} className="chat-suggestion-card" onClick={() => handleFillInput(s.label)}>
                      <div style={{ fontSize: 17, marginBottom: 6, color: 'var(--gaea-glow)' }}>{s.icon}</div>
                      <div style={{ color: C('color-text'), fontSize: 12.5, fontWeight: 500, marginBottom: 2 }}>{s.label}</div>
                      <div style={{ color: C('color-text-secondary'), fontSize: 11, lineHeight: 1.4 }}>{s.desc}</div>
                    </div>
                  ))}
                </div>
                <Button type="primary" onClick={navigateToCharacterLib} style={{ marginTop: 22, borderRadius: 20, padding: '4px 22px', height: 38, fontSize: 13 }}>
                  虚拟助手管理
                </Button>
              </div>
            ) : (
              <div className="chat-empty">
                <div className="chat-empty-orb"><RobotOutlined style={{ fontSize: 40 }} /></div>
                <h2>gaea AI</h2>
                <p>你的智能 AI 助手：聊天、写作、翻译、学习，随时待命</p>
                <div className="chat-suggestion-grid">
                  {PLAIN_SUGGESTIONS.map(s => (
                    <div key={s.label} className="chat-suggestion-card" onClick={() => handleSuggestion(s.label)}>
                      <div style={{ fontSize: 17, marginBottom: 6, color: 'var(--gaea-glow)' }}>{s.icon}</div>
                      <div style={{ color: C('color-text'), fontSize: 12.5, fontWeight: 500, marginBottom: 2 }}>{s.label}</div>
                      <div style={{ color: C('color-text-secondary'), fontSize: 11, lineHeight: 1.4 }}>{s.desc}</div>
                    </div>
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

      {/* 右侧记忆抽屉（人格模式） */}
      {mode !== 'plain' && (
        <aside className="chat-drawer gaea-glass-shell" style={{ width: drawerOpen ? 300 : 0, minWidth: drawerOpen ? 300 : 0, display: drawerOpen ? 'flex' : 'none' }}>
          {drawerOpen && (
            <Tabs
              activeKey={drawerTab}
              onChange={setDrawerTab}
              size="small"
              tabBarStyle={{ margin: '0 10px', borderBottom: `1px solid ${C('color-border')}` }}
              style={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 }}
              items={[
                {
                  key: 'status',
                  label: <span style={{ fontSize: 11 }}>状态</span>,
                  children: (
                    <div style={{ flex: 1, overflow: 'auto', minHeight: 0 }}>
                      <WhisperEmotionPanel
                        emotion={emotion} stage={stage} trust={trust} rifts={rifts}
                        aff={aff} sec={sec} aro={aro} dom={dom}
                        T={currentPersonality?.dims?.T ?? 50} I={currentPersonality?.dims?.I ?? 50}
                        S={currentPersonality?.dims?.S ?? 50} O={currentPersonality?.dims?.O ?? 50}
                        R={currentPersonality?.dims?.R ?? 50}
                        totalTurns={totalTurns}
                        personalityLabel={personaLabel}
                      />
                      <WhisperDesirePanel desireStack={{ slots: desireSlots }} sharedEventsCount={sharedEvents} />
                    </div>
                  ),
                },
                {
                  key: 'memory',
                  label: <span style={{ fontSize: 11 }}>记忆 {facts.length > 0 && <Tag style={{ fontSize: 9, margin: 0, padding: '0 4px', lineHeight: '14px' }}>{facts.length}</Tag>}</span>,
                  children: (
                    <MemoryDrawer
                      facts={facts} personalityLabel={personaLabel}
                      search={memorySearch} onSearch={setMemorySearch}
                      collapsed={collapsedGroups} onToggle={(d) => setCollapsedGroups(prev => { const next = new Set(prev); next.has(d) ? next.delete(d) : next.add(d); return next })}
                      onOpenPage={() => setShowMemoryPage(true)}
                    />
                  ),
                },
                {
                  key: 'trace',
                  label: <span style={{ fontSize: 11 }}>追踪</span>,
                  children: <WhisperTracePanel traces={traces} currentTurn={totalTurns} />,
                },
              ]}
            />
          )}
        </aside>
      )}

      {/* 绑定模型条（聊天板块统一入口；whisper 为 chat 别名） */}
      <div style={{ position: 'absolute', left: 12, bottom: 12, zIndex: 50 }}>
        <FeatureModelBar feature="chat" label="聊天" />
      </div>

      <Modal title="语音设置" open={showVoiceSettings} onCancel={() => setShowVoiceSettings(false)} footer={null} width={480} centered destroyOnClose>
        <VoiceSettingsPanel />
      </Modal>

      <Modal title={null} open={showMemoryPage} onCancel={() => setShowMemoryPage(false)} footer={null} width={720} centered bodyStyle={{ maxHeight: '70vh', overflow: 'auto' }}>
        <WhisperMemoryModal facts={facts} personalityID={mode !== 'plain' ? mode : activePersonality} onFactsChange={setFacts} />
      </Modal>
    </div>
  )
}

export default ChatPage
