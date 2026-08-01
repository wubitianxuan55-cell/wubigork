import React, { useState, useCallback, useEffect, useRef, useMemo } from 'react'
import { Input, Button, Avatar, Typography, Tooltip, Card, Modal, Tag, message, Tabs, Select, Popconfirm } from 'antd'
import {
  SendOutlined, UserOutlined, CopyOutlined, CheckOutlined,
  HeartOutlined, SwapOutlined, SoundOutlined, DeleteOutlined,
  PlusOutlined, MessageOutlined, ApiOutlined, ClearOutlined,
  SearchOutlined, GlobalOutlined, StarFilled, EditOutlined,
  SettingOutlined, MenuFoldOutlined, MenuUnfoldOutlined, CloseOutlined,
  AudioOutlined, StopOutlined, RobotOutlined,
} from '@ant-design/icons'
import * as App from '../../wailsjs/go/app/App'
import { C } from '../utils/theme'
import VoiceChatOrb from '../components/VoiceChatOrb'
import { useVoiceChat } from '../hooks/useVoiceChat'
import { VOICE_LAUNCH_FLAG } from '../components/ModuleLauncher'
import { WhisperEmotionPanel } from '../components/WhisperEmotionPanel'
import { MarkdownContent, mdStyles } from '../components/MarkdownContent'
import { ParticleFlow } from '../components/ParticleFlow'
import { SoundWaveOverlay } from '../components/SoundWaveOverlay'
import VoiceSettingsPanel from '../components/VoiceSettingsPanel'
import WhisperSettingsPanel from '../components/WhisperSettingsPanel'
import WhisperTracePanel from '../components/WhisperTracePanel'
import WhisperDesirePanel from '../components/WhisperDesirePanel'
import WhisperMemoryModal from '../components/WhisperMemoryModal'
import AssistantManagerModal from '../components/AssistantManagerModal'
import { CompanionAvatar } from '../components/CompanionAvatar'
import '../whisper-theme.css'
interface Personality {

  id: string; label: string; gender: string; tags?: string[]
  dims: { T: number; I: number; S: number; O: number; R: number }
  voiceGuide?: string; requiresAdult18?: boolean
}
interface Message { id: string; role: 'user' | 'assistant'; content: string; streaming?: boolean; timestamp?: number }
interface Topic { id: string; title: string; messages: Message[]; createdAt: number }

const STORAGE_KEY = 'gaea_whisper_topics'
const LEGACY_STORAGE_KEY = 'wubigrok_whisper_topics'
const PERSONALITY_KEY = 'gaea_whisper_personality'
const LEGACY_PERSONALITY_KEY = 'wubigrok_whisper_personality'
const ACTIVE_TOPIC_KEY = 'gaea_whisper_active_topic'
const COMPANION_SETTINGS_KEY = 'gaea_whisper_companion_settings'
const LEGACY_COMPANION_SETTINGS_KEY = 'wubigrok_whisper_companion_settings'
let msgId = 0
function nextMsgId() { msgId++; return `wm_${msgId}_${Date.now()}` }
function genTopicId() { return `wt_${Date.now()}_${Math.random().toString(36).slice(2, 8)}` }
function loadTopics(): Topic[] {
  try { const r = localStorage.getItem(STORAGE_KEY) ?? localStorage.getItem(LEGACY_STORAGE_KEY); if (r) { const p = JSON.parse(r); if (Array.isArray(p) && p.length > 0) return p } } catch (_) {}
  return [createTopic('新对话')]
}
function saveTopics(t: Topic[]) { try { localStorage.setItem(STORAGE_KEY, JSON.stringify(t)) } catch (_) {} }
function createTopic(title: string): Topic { return { id: genTopicId(), title, messages: [], createdAt: Date.now() } }

// ─── 记忆精确分类映射（对齐后端 memory_taxonomy.go 6 domain）───
const DOMAIN_LABELS: Record<string, string> = {
  IDENTITY: '🪪 身份', SOCIAL: '💕 社交', DAILY_LIFE: '🏠 日常',
  PURSUITS: '🎯 追求', INNER_WORLD: '🧘 内心', TEMPORAL: '⏰ 时间',
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

const WhisperPage: React.FC = () => {
  const initTopics = loadTopics()
  const [topics, setTopics] = useState<Topic[]>(initTopics)
  const [activeId, setActiveId] = useState<string>(() => initTopics[0]?.id || '')
  const [editingTitle, setEditingTitle] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')
  const [personalities, setPersonalities] = useState<Personality[]>([])
  const [activePersonality, setActivePersonality] = useState<string>(() => {
    try { return (localStorage.getItem(PERSONALITY_KEY) ?? localStorage.getItem(LEGACY_PERSONALITY_KEY)) || 'deredere' } catch { return 'deredere' }
  })
  const [personalityOpen, setPersonalityOpen] = useState(false)
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [streamText, setStreamText] = useState('')
  const [copiedId, setCopiedId] = useState<string | null>(null)
  const [speakingId, setSpeakingId] = useState<string | null>(null)
  const [emotion, setEmotion] = useState('')
  const [stage, setStage] = useState(''); const [trust, setTrust] = useState(50)
  const [aff, setAff] = useState(0); const [sec, setSec] = useState(0); const [aro, setAro] = useState(0); const [dom, setDom] = useState(0)
  const [rifts, setRifts] = useState(0); const [totalTurns, setTotalTurns] = useState(0)
  const [engineName, setEngineName] = useState(''); const [modelName, setModelName] = useState('')
  const [showSettings, setShowSettings] = useState(false)
  const [showVoiceSettings, setShowVoiceSettings] = useState(false)
  const [adultMode, setAdultMode] = useState(false)
  const [sidebarTab, setSidebarTab] = useState<string>('status') // 侧边栏标签页
  const [collapsedMemoryGroups, setCollapsedMemoryGroups] = useState<Set<string>>(new Set())
  const [showMemoryPage, setShowMemoryPage] = useState(false)
  const [memorySearch, setMemorySearch] = useState('')
  const [facts, setFacts] = useState<any[]>([])
  const [sharedEvents, setSharedEvents] = useState(0)
  const [searchEnabled, setSearchEnabled] = useState(true) // 上网搜索开关
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false) // 侧边栏折叠
  const [desireSlots, setDesireSlots] = useState<any[]>([])
  const [traces, setTraces] = useState<any[]>([])
  const listRef = useRef<HTMLDivElement>(null); const inputRef = useRef<any>(null); const hasInitRef = useRef(false)

  // ── 语音对话（轻语板块承载语音能力；对话目标 = 平台 AI 助手 gaea）──
  const [voiceOpen, setVoiceOpen] = useState(false)
  const [voiceUserText, setVoiceUserText] = useState('')
  const [voiceReply, setVoiceReply] = useState('')
  const voiceTargetSetRef = useRef(false)
  const { state: voice, start: startVoice, stop: stopVoice, interrupt } = useVoiceChat({
    onTranscript: (t) => { setVoiceUserText(t); setVoiceReply('') },
    onReply: (t) => setVoiceReply(t),
  })

  const openVoice = useCallback(async () => {
    setVoiceOpen(true)
    setVoiceUserText('')
    setVoiceReply('')
    try {
      if (!voiceTargetSetRef.current) {
        await App.VoiceSetChatTarget('gaea')
        voiceTargetSetRef.current = true
      }
    } catch (err: any) {
      console.warn('[whisper] 语音对话目标切换失败:', err)
    }
    await startVoice()
  }, [startVoice])

  const closeVoice = useCallback(() => {
    stopVoice()
    setVoiceOpen(false)
    setVoiceUserText('')
    setVoiceReply('')
  }, [stopVoice])

  // 首页语音入口：进入轻语板块自动启动语音对话
  useEffect(() => {
    let flag = false
    try { flag = sessionStorage.getItem(VOICE_LAUNCH_FLAG) === '1' } catch (_) {}
    if (flag) {
      try { sessionStorage.removeItem(VOICE_LAUNCH_FLAG) } catch (_) {}
      openVoice()
    }
  }, [openVoice])

  const activeTopic = topics.find(t => t.id === activeId)
  const messages = activeTopic?.messages ?? []
  const currentPersonality = personalities.find(p => p.id === activePersonality)
  const streamingMsg = messages.find(m => m.streaming)
  const hasMessages = messages.length > 0
  const emoColors: Record<string,string> = {SWEET_ATTACHMENT:"#f472b6",SHY_HEARTBEAT:"#fb7185",TSUNDERE:"#f59e0b",HURT_GRIEVANCE:"#a78bfa",ANGRY_ATTACK:"#ef4444",COLD_DETACHED:"#94a3b8",FEARFUL_OBEDIENT:"#c084fc",QUIET_FOND:"#fbbf24",CALM_RATIONAL:"#60a5fa"}
  const emoColor = emoColors[emotion] || "#e85388"
  const topicList = useMemo(() => topics.map(({ id, title, createdAt }) => ({ id, title, createdAt })), [topics])

  // gaea名称：优先从 localStorage 读取自定义名称，fallback 到人格 label
  const companionName = useMemo(() => {
    try {
      const raw = localStorage.getItem(COMPANION_SETTINGS_KEY) ?? localStorage.getItem(LEGACY_COMPANION_SETTINGS_KEY)
      if (raw) {
        const parsed = JSON.parse(raw)
        if (parsed?.companionName) return parsed.companionName
      }
    } catch (_) {}
    return currentPersonality?.label || '温柔'
  }, [currentPersonality])

  useEffect(() => { localStorage.setItem(ACTIVE_TOPIC_KEY, activeId) }, [activeId])
  useEffect(() => { if (listRef.current) listRef.current.scrollTop = listRef.current.scrollHeight }, [messages, streamText])
  useEffect(() => { App.WhisperGetPersonalities().then(setPersonalities).catch(() => {}) }, [])
  useEffect(() => {
    if (hasInitRef.current || messages.length > 0 || personalities.length === 0) return
    hasInitRef.current = true
    const p = personalities.find(p => p.id === activePersonality)
    if (p) {
      setTopics(prev => prev.map(t => t.id === activeId ? { ...t, messages: [{ id: nextMsgId(), role: 'assistant' as const, timestamp: Date.now(), content: `你好呀~ 我是「${p.label}」💫\n想和我聊什么呢？` }] } : t))
    }
  }, [personalities, activePersonality, activeId, messages.length])

  const handleSend = useCallback(async () => {
    const text = input.trim(); if (!text || loading) return
    message.info('发送中…')
    console.log('[whisper] handleSend start:', text.slice(0, 50))
    setInput('')
    const um: Message = { id: nextMsgId(), role: 'user', content: text, timestamp: Date.now() }
    const am: Message = { id: nextMsgId(), role: 'assistant', content: '...', streaming: true, timestamp: Date.now() }
    setTopics(prev => {
      const next = prev.map(t => t.id === activeId ? { ...t, messages: [...t.messages, um, am] } : t)
      saveTopics(next)
      return next
    })
    setLoading(true)
    try {
      const chatFn = searchEnabled ? App.WhisperChatWithSearch : App.WhisperChat
      const res = await chatFn(text, activePersonality) as any
      const reply = (typeof res?.reply === 'string') ? res.reply : (typeof res === 'string' ? res : '')
      console.log('[whisper] WhisperChat reply len:', reply.length)
      message.success('收到回复: ' + reply.length + '字')
      if (res?.emotion) setEmotion(res.emotion); if (res?.stage) setStage(res.stage)
      if (typeof res?.trust === 'number') setTrust(Math.round(res.trust))
      if (typeof res?.aff === 'number') setAff(Math.round(res.aff))
      if (typeof res?.sec === 'number') setSec(Math.round(res.sec))
      if (typeof res?.aro === 'number') setAro(Math.round(res.aro))
      if (typeof res?.dom === 'number') setDom(Math.round(res.dom))
      if (typeof res?.rifts === 'number') setRifts(res.rifts)
      if (typeof res?.totalTurns === 'number') setTotalTurns(res.totalTurns)
      if (res?.desireSlots) setDesireSlots(res.desireSlots)
      if (res?.trace) setTraces(prev => [...prev, res.trace])
      if (res?.facts) setFacts(res.facts)
      if (typeof res?.sharedEvents === 'number') setSharedEvents(res.sharedEvents)
      am.content = reply || '(空回复)'; am.streaming = false
      setTopics(prev => {
        const next = prev.map(t => t.id === activeId ? { ...t, messages: t.messages.map(m => m.id === am.id ? { ...am } : m), title: t.title === '新对话' ? text.slice(0, 20) + (text.length > 20 ? '…' : '') : t.title } : t)
        saveTopics(next)
        return next
      })
    } catch (err: any) {
      message.error('发送失败: ' + (err?.message || String(err)))
      console.error('[whisper] WhisperChat FAILED:', err)
      am.content = '❌ ' + (err?.message || String(err)); am.streaming = false
      setTopics(prev => prev.map(t => t.id === activeId ? { ...t, messages: t.messages.map(m => m.id === am.id ? { ...am } : m) } : t))
    } finally {
      setLoading(false)
    }
  }, [input, loading, activeId, activePersonality])

  const handleKeyDown = (e: React.KeyboardEvent) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend() } }
  const handleCopy = async (content: string, id: string) => {
    try { await navigator.clipboard.writeText(content) } catch { const ta = document.createElement('textarea'); ta.value = content; document.body.appendChild(ta); ta.select(); document.execCommand('copy'); document.body.removeChild(ta) }
    setCopiedId(id); setTimeout(() => setCopiedId(null), 2000)
  }
  const handleSpeak = async (content: string, id: string) => {
    if (speakingId) return; setSpeakingId(id)
    try {
      const result = await (App as any).TTSSpeakBase64(content)
      if (result?.base64) { const b = atob(result.base64); const bytes = new Uint8Array(b.length); for (let i = 0; i < b.length; i++) bytes[i] = b.charCodeAt(i); const a = new Audio(URL.createObjectURL(new Blob([bytes], { type: result.mimeType || 'audio/mp3' }))); a.onended = () => setSpeakingId(null); a.onerror = () => { setSpeakingId(null); message.error('播放失败') }; await a.play(); return }
      message.warning('TTS 未返回音频数据')
    } catch (err: any) { message.error(`朗读失败: ${typeof err === 'string' ? err : err?.message || '未知错误'}`) }
    setSpeakingId(null)
  }

  const handleCreateTopic = useCallback(() => { const t = createTopic('新对话'); setTopics(prev => [...prev, t]); setActiveId(t.id); setEmotion(''); setStage(''); setTrust(50); setAff(0); setSec(0); setAro(0); setDom(0); setRifts(0); setTotalTurns(0) }, [])
  const handleDeleteTopic = useCallback((id: string) => { setTopics(prev => { const next = prev.filter(t => t.id !== id); if (next.length === 0) { const fb = createTopic('新对话'); setActiveId(fb.id); setEmotion(''); return [fb] } if (id === activeId) { const idx = prev.findIndex(t => t.id === id); setActiveId(next[Math.min(idx, next.length - 1)].id); setEmotion('') } return next }) }, [activeId])
  const handleRenameTopic = useCallback((id: string) => { setEditingTitle(id); setEditValue(topics.find(t => t.id === id)?.title || '') }, [topics])
  const handleRenameConfirm = useCallback((id: string) => { if (editValue.trim()) { setTopics(prev => prev.map(t => t.id === id ? { ...t, title: editValue.trim() } : t)) }; setEditingTitle(null) }, [editValue])
  const handleClearMessages = useCallback(async () => {
    setTopics(prev => prev.map(t => t.id === activeId ? { ...t, messages: [] } : t))
    setEmotion(''); setStage(''); setTrust(50); setAff(0); setSec(0); setAro(0); setDom(0); setRifts(0); setTotalTurns(0)
    try { await (App as any).WhisperClearSession(activePersonality) } catch (_) {}
  }, [activeId, activePersonality])

  const handleSwitchPersonality = useCallback(async (id: string) => {
    // 清除旧人格的后端会话
    try { await (App as any).WhisperClearSession(activePersonality) } catch (_) {}
    setActivePersonality(id); localStorage.setItem(PERSONALITY_KEY, id); setPersonalityOpen(false)
    // 重置前端状态
    setEmotion(''); setStage(''); setTrust(50); setAff(0); setSec(0); setAro(0); setDom(0); setRifts(0); setTotalTurns(0)
    // 添加切换提示消息
    const p = personalities.find(p => p.id === id)
    if (p) {
      setTopics(prev => prev.map(t => t.id === activeId ? {
        ...t,
        messages: [...t.messages, { id: nextMsgId(), role: 'assistant' as const, timestamp: Date.now(), content: `（已切换为「${p.label}」人格）\n想和我聊点什么呢？` }]
      } : t))
    }
  }, [activePersonality, personalities, activeId])

  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'row', minHeight: 0, position: 'relative' }}>
      {/* 左侧话题侧边栏 */}
      <div style={{ width: 200, minWidth: 200, borderRight: `1px solid ${C('color-border')}`, background: C('color-bg-elevated'), display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        <div style={{ padding: '10px 12px', borderBottom: `1px solid ${C('color-border')}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Typography.Text strong style={{ fontSize: 12, color: C('color-text') }}>对话</Typography.Text>
          <Button type="text" size="small" icon={<PlusOutlined />} onClick={handleCreateTopic} style={{ color: C('color-text-secondary'), width: 24, height: 24, padding: 0, borderRadius: 6 }} />
        </div>
        <div style={{ flex: 1, overflow: 'auto', padding: '4px 6px' }}>
          {topicList.map(t => (
            <div key={t.id} onClick={() => setActiveId(t.id)} onDoubleClick={() => handleRenameTopic(t.id)}
              style={{ padding: '7px 10px', borderRadius: 8, cursor: 'pointer', marginBottom: 2, background: t.id === activeId ? `${C('color-primary')}12` : 'transparent', display: 'flex', alignItems: 'center', gap: 6, fontSize: 12, color: t.id === activeId ? C('color-primary') : C('color-text') }}>
              <MessageOutlined style={{ fontSize: 11, opacity: 0.5 }} />
              {editingTitle === t.id ? (
                <input value={editValue} onChange={e => setEditValue(e.target.value)} onBlur={() => handleRenameConfirm(t.id)} onKeyDown={e => { if (e.key === 'Enter') handleRenameConfirm(t.id) }} autoFocus
                  style={{ flex: 1, border: 'none', outline: 'none', background: 'transparent', fontSize: 12, color: C('color-text'), padding: 0 }} onClick={e => e.stopPropagation()} />
              ) : (
                <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{t.title}</span>
              )}
              {t.id === activeId && topics.length > 1 && (
                <Button type="text" size="small" icon={<DeleteOutlined />} onClick={e => { e.stopPropagation(); handleDeleteTopic(t.id) }}
                  style={{ color: C('color-text-secondary'), opacity: 0.3, width: 18, height: 18, padding: 0, minWidth: 18, fontSize: 10, borderRadius: 4 }} />
              )}
            </div>
          ))}
        </div>
        <div style={{ padding: '8px 12px', borderTop: `1px solid ${C('color-border')}`, fontSize: 10, color: C('color-text-secondary'), opacity: 0.6 }}>
          <ApiOutlined style={{ marginRight: 4 }} />{engineName || '默认引擎'}{modelName && <span> / {modelName}</span>}
        </div>
      </div>

      {/* 右侧主聊天区 */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0, overflow: 'hidden', background: C('color-bg-container'), position: 'relative' }}>
        <style>{mdStyles}</style>
        <ParticleFlow aro={aro} />
        <SoundWaveOverlay active={speakingId !== null} aff={aff} aro={aro} />
        {/* 设置按钮 */}
        <div style={{ position: 'absolute', top: 8, right: 8, zIndex: 10, display: 'flex', gap: 8 }}>
          <Tooltip title="gaea设置">
            <Button type="text" size="small" icon={<SettingOutlined />} onClick={() => setShowSettings(true)}
              style={{ color: C('color-text-secondary'), opacity: 0.5, width: 28, height: 28, padding: 0 }} />
          </Tooltip>
          <Tooltip title="语音设置">
            <Button type="text" size="small" icon={<SoundOutlined />} onClick={() => setShowVoiceSettings(true)}
              style={{ color: C('color-text-secondary'), opacity: 0.5, width: 28, height: 28, padding: 0 }} />
          </Tooltip>
          <Tooltip title={voiceOpen ? '结束语音对话' : '语音对话（gaea）'}>
            <Button type="text" size="small" icon={<AudioOutlined />}
              onClick={() => voiceOpen ? closeVoice() : openVoice()}
              style={{ color: voiceOpen ? '#e85388' : C('color-text-secondary'), opacity: voiceOpen ? 1 : 0.5, width: 28, height: 28, padding: 0 }} />
          </Tooltip>
          {hasMessages && (
            <Tooltip title="清空当前对话"><Button type="text" size="small" icon={<ClearOutlined />} onClick={handleClearMessages} style={{ color: C('color-text-secondary'), opacity: 0.4, width: 28, height: 28, padding: 0 }} /></Tooltip>
          )}
        </div>

        {/* 人格信息头 */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '8px 16px', borderBottom: `1px solid ${C('color-border')}`, flexShrink: 0, background: C('color-bg-elevated') }}>
          <CompanionAvatar size={48} state={speakingId ? 'speaking' : loading ? 'thinking' : 'idle'} emotionColor={emoColor} />
          <div style={{ flex: 1 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
              <Typography.Text strong style={{ fontSize: 13, color: C('color-text') }}>{companionName}</Typography.Text>
              <Button type="text" size="small" icon={<SwapOutlined />} onClick={() => setPersonalityOpen(true)} style={{ color: C('color-text-secondary'), width: 22, height: 22, padding: 0 }} />
            </div>
          </div>
        </div>


        {/* 引擎状态条 */}
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '3px 16px', borderBottom: `1px solid ${C('color-border')}`, flexShrink: 0, fontSize: 10, color: C('color-text-secondary'), background: C('color-bg-elevated'), flexWrap: 'wrap' }}>
            <span>💭{emotion}</span><span style={{ opacity: 0.2 }}>|</span>
            <span>🤝{stage === 'STRANGER' ? '初识' : stage === 'FAMILIAR' ? '熟悉' : stage === 'INTIMATE' ? '亲密' : stage}</span><span style={{ opacity: 0.2 }}>|</span>
            <span>💚{trust}</span><span style={{ opacity: 0.2 }}>|</span>
            {rifts > 0 && <><span style={{ opacity: 0.2 }}>|</span><span>💔{rifts}</span></>}
            <span style={{ marginLeft: 'auto', display: 'flex', alignItems: 'center', gap: 4 }}>
              <Tooltip title={searchEnabled ? '上网搜索已开启（自动检测搜索意图）' : '上网搜索已关闭'}>
                <Button type="text" size="small"
                  icon={<GlobalOutlined style={{ color: searchEnabled ? '#52c41a' : C('color-text-secondary') }} />}
                  onClick={() => setSearchEnabled(!searchEnabled)}
                  style={{ padding: '0 2px', height: 18, fontSize: 12, opacity: searchEnabled ? 1 : 0.5 }} />
              </Tooltip>
              <span style={{ opacity: 0.4 }}>#{totalTurns}</span>
            </span>
          </div>

        {/* 消息区 */}
        <div ref={listRef} style={{ flex: 1, minHeight: 0, overflow: 'auto', padding: hasMessages ? '20px 0 150px' : '0' }}>
          {!hasMessages ? (
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%', padding: 32 }}>
              <CompanionAvatar size={120} state="idle" emotionColor={emoColor} />
              <div style={{ height: 16 }} />
              <Typography.Text style={{ color: C('color-text'), fontSize: 20, fontWeight: 700, marginBottom: 4 }}>轻语</Typography.Text>
              <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 13, marginBottom: 20 }}>选择一位AIgaea，开始对话 💫</Typography.Text>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: 8, maxWidth: 460, width: '100%', marginBottom: 20 }}>\n                {[{ icon: '💬', label: '聊聊今天', desc: '分享你的日常' }, { icon: '💭', label: '倾诉心情', desc: '说说心里话' }, { icon: '🌐', label: '上网查询', desc: '搜最新资讯' }, { icon: '🎵', label: '分享兴趣', desc: '聊聊你喜欢的东西' }, { icon: '🌙', label: '晚安问候', desc: '睡前聊一会儿' }].map(s => (
                  <div key={s.label} onClick={() => { setInput(s.label); inputRef.current?.focus() }}
                    style={{ padding: '10px 12px', borderRadius: 12, background: C('color-bg-elevated'), border: `1px solid ${C('color-border')}`, cursor: 'pointer', transition: 'all 0.15s', userSelect: 'none' }}
                    onMouseEnter={e => { e.currentTarget.style.borderColor = '#e85388'; e.currentTarget.style.transform = 'translateY(-2px)' }}
                    onMouseLeave={e => { e.currentTarget.style.borderColor = C('color-border'); e.currentTarget.style.transform = 'translateY(0)' }}>
                    <div style={{ fontSize: 18, marginBottom: 4 }}>{s.icon}</div><div style={{ color: C('color-text'), fontSize: 12, fontWeight: 500 }}>{s.label}</div><div style={{ color: C('color-text-secondary'), fontSize: 10 }}>{s.desc}</div>
                  </div>
                ))}
              </div>
              <Button type="primary" icon={<SwapOutlined />} onClick={() => setPersonalityOpen(true)}
                style={{ borderRadius: 20, padding: '4px 22px', height: 38, fontSize: 13, background: 'linear-gradient(135deg, #e85388, #a855f7)', border: 'none' }}>选择gaea人格</Button>
            </div>
          ) : (
            <div style={{ maxWidth: 'var(--whisper-chat-max-width)', margin: '0 auto', padding: '0 16px' }}>
              {messages.map(msg => {
                const isUser = msg.role === 'user'; const isStreaming = msg.streaming && msg === streamingMsg; const displayContent = isStreaming ? streamText : msg.content
                return (
                  <div key={msg.id} className={isUser ? 'whisper-msg-user' : `whisper-msg-her${isStreaming ? ' streaming' : ''}`}>
                    {!isUser && (
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 2 }}>
                        <div className={`whisper-light-core${(trust || 0) > 70 ? ' double-pulse' : ''}`} />
                        <span style={{ fontSize: 11, color: 'var(--whisper-ink-muted)', fontWeight: 500 }}>
                          {currentPersonality?.label || 'gaea'}
                        </span>
                      </div>
                    )}
                    <div style={{ color: isUser ? 'var(--whisper-ink-muted)' : 'var(--whisper-ink)', lineHeight: 1.75, fontSize: 13, wordBreak: 'break-word' }}>
                      {isUser ? displayContent : <MarkdownContent source={displayContent} className="md-content" />}
                      {isStreaming && <span className="cursor-blink" />}
                    </div>
                    {msg.content && !msg.streaming && (
                      <div style={{ display: 'flex', gap: 2, marginTop: 3 }}>
                        <Tooltip title={copiedId === msg.id ? '已复制' : '复制'}><Button type="text" size="small" icon={copiedId === msg.id ? <CheckOutlined style={{ color: '#52c41a' }} /> : <CopyOutlined />} onClick={() => handleCopy(msg.content, msg.id)} style={{ color: 'var(--whisper-ink-muted)', opacity: 0.4, fontSize: 11, padding: '0 4px', height: 20 }} /></Tooltip>
                        {!isUser && <Tooltip title={speakingId === msg.id ? '朗读中…' : '朗读'}><Button type="text" size="small" icon={<SoundOutlined />} loading={speakingId === msg.id} onClick={() => handleSpeak(msg.content, msg.id)} style={{ color: 'var(--whisper-ink-muted)', opacity: 0.4, fontSize: 11, padding: '0 4px', height: 20 }} /></Tooltip>}
                      </div>
                    )}
                  </div>
                )
              })}
              {loading && !streamText && !streamingMsg && (
                <div className="whisper-msg-her">
                  <div className="whisper-typing-indicator">
                    <span className="whisper-typing-dot" />
                    <span className="whisper-typing-dot" />
                    <span className="whisper-typing-dot" />
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        {/* 输入框 */}
        <div style={{ position: 'absolute', bottom: 0, left: 0, right: 0, display: 'flex', justifyContent: 'center', padding: '0 16px 16px', pointerEvents: 'none' }}>
          <div style={{ width: '100%', maxWidth: 660, display: 'flex', alignItems: 'flex-end', gap: 6, padding: '6px 10px', background: C('color-bg-container'), border: `1px solid ${C('color-border')}`, borderRadius: 16, boxShadow: '0 6px 24px rgba(0,0,0,0.06)', pointerEvents: 'auto' }}>
            <Input.TextArea ref={inputRef} value={input} onChange={e => setInput(e.target.value)} onKeyDown={handleKeyDown} placeholder="输入消息，Enter 发送 / Shift+Enter 换行" disabled={loading} autoSize={{ minRows: 1, maxRows: 5 }} className="chat-input-textarea" style={{ flex: 1, background: 'transparent', border: 'none', color: C('color-text'), borderRadius: 0, resize: 'none', fontSize: 13, lineHeight: 1.5, padding: '4px 2px', boxShadow: 'none' }} />
            <Tooltip title="发送 (Enter)"><Button type="primary" icon={<SendOutlined />} onClick={handleSend} loading={loading} disabled={!input.trim()} style={{ background: input.trim() ? 'linear-gradient(135deg, #e85388, #a855f7)' : C('color-border'), borderColor: 'transparent', borderRadius: 10, width: 34, height: 34, minWidth: 34, padding: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: input.trim() ? '0 2px 8px rgba(232,83,136,0.4)' : 'none', flexShrink: 0, transition: 'transform 0.15s', transform: 'scale(1)' }} onMouseEnter={e=>e.currentTarget.style.transform="scale(1.05)"} onMouseLeave={e=>e.currentTarget.style.transform="scale(1)"} /></Tooltip>
          </div>
        </div>
      </div>

      {/* 右侧侧边栏 — 情感/渴望/追踪/记忆 — 毛玻璃风格 */}
      <aside className="whisper-glass" style={{
          width: sidebarCollapsed ? 0 : 280, minWidth: sidebarCollapsed ? 0 : 280,
          overflow: 'hidden', flexShrink: 0,
          display: 'flex', flexDirection: 'column',
          borderLeft: sidebarCollapsed ? 'none' : '1px solid var(--whisper-glass-border)',
          transition: 'width 0.25s ease, min-width 0.25s ease',
        }}>
          {!sidebarCollapsed && (
            <Tabs
              activeKey={sidebarTab}
              onChange={setSidebarTab}
              size="small"
              tabBarStyle={{ margin: '0 10px', borderBottom: `1px solid ${C('color-border')}` }}
              style={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 }}
              items={[
                {
                  key: 'status',
                  label: <span style={{ fontSize: 11, display: 'flex', alignItems: 'center', gap: 3 }}>🫀 状态</span>,
                  children: (
                    <div style={{ flex: 1, overflow: 'auto', minHeight: 0 }}>
                      <WhisperEmotionPanel
                        emotion={emotion} stage={stage} trust={trust} rifts={rifts}
                        aff={aff} sec={sec} aro={aro} dom={dom}
                        T={currentPersonality?.dims?.T ?? 50} I={currentPersonality?.dims?.I ?? 50}
                        S={currentPersonality?.dims?.S ?? 50} O={currentPersonality?.dims?.O ?? 50}
                        R={currentPersonality?.dims?.R ?? 50}
                        totalTurns={totalTurns}
                        personalityLabel={currentPersonality?.label || '温柔'}
                      />
                      <WhisperDesirePanel
                        desireStack={{ slots: desireSlots }}
                        sharedEventsCount={sharedEvents}
                      />
                    </div>
                  ),
                },
                {
                  key: 'memory',
                  label: <span style={{ fontSize: 11, display: 'flex', alignItems: 'center', gap: 3 }}>🧠 记忆 {facts.length > 0 && <Tag style={{ fontSize: 9, margin: 0, padding: '0 4px', lineHeight: '14px' }}>{facts.length}</Tag>}</span>,
                  children: (() => {
                    // 过滤 + 按 domain 分组
                    const filteredFacts = facts.filter((f: any) =>
                      !memorySearch ||
                      f.subject?.toLowerCase().includes(memorySearch.toLowerCase()) ||
                      f.summary?.toLowerCase().includes(memorySearch.toLowerCase())
                    )
                    const grouped = DOMAIN_ORDER.map(d => ({
                      domain: d,
                      label: DOMAIN_LABELS[d] || d,
                      facts: filteredFacts.filter((f: any) => f.domain === d || (f.domain === d.toLowerCase())),
                    })).filter(g => g.facts.length > 0)

                    const toggleGroup = (d: string) => setCollapsedMemoryGroups(prev => {
                      const next = new Set(prev); next.has(d) ? next.delete(d) : next.add(d); return next
                    })

                    return (
                      <div style={{ display: 'flex', flexDirection: 'column', height: '100%', gap: 4, padding: '0 2px' }}>
                        <Input prefix={<SearchOutlined />} size="small" placeholder="搜索记忆…"
                          value={memorySearch} onChange={e => setMemorySearch(e.target.value)}
                          allowClear style={{ borderRadius: 8, fontSize: 11 }} />
                        <div style={{ flex: 1, overflow: 'auto', minHeight: 0 }}>
                          {facts.length === 0 ? (
                            <div style={{ textAlign: 'center', padding: 24, color: C('color-text-secondary'), fontSize: 12 }}>
                              还没有记忆，多聊聊吧 💫
                            </div>
                          ) : filteredFacts.length === 0 ? (
                            <div style={{ textAlign: 'center', padding: 24, color: C('color-text-secondary'), fontSize: 12 }}>无匹配</div>
                          ) : (
                            grouped.map(g => {
                              const collapsed = collapsedMemoryGroups.has(g.domain)
                              const coreInGroup = g.facts.filter((f: any) => f.tier === 'core').length
                              return (
                                <div key={g.domain} style={{ marginBottom: 2 }}>
                                  {/* 分组标题 */}
                                  <div
                                    onClick={() => toggleGroup(g.domain)}
                                    style={{
                                      display: 'flex', alignItems: 'center', gap: 4, padding: '6px 6px',
                                      borderRadius: 8, cursor: 'pointer',
                                      background: collapsed ? 'transparent' : `${C('color-bg-elevated')}80`,
                                      transition: 'background 150ms',
                                      userSelect: 'none',
                                    }}
                                  >
                                    <span style={{ fontSize: 10, transition: 'transform 200ms', transform: collapsed ? 'rotate(-90deg)' : 'rotate(0deg)' }}>▼</span>
                                    <span style={{ fontSize: 12, fontWeight: 600, color: C('color-text'), flex: 1 }}>{g.label}</span>
                                    <Tag style={{ fontSize: 9, margin: 0, padding: '0 5px', lineHeight: '16px', background: 'transparent', border: '1px solid rgba(255,255,255,0.08)', color: C('color-text-secondary') }}>
                                      {g.facts.length}
                                    </Tag>
                                    {coreInGroup > 0 && (
                                      <span style={{ fontSize: 9, color: '#faad14' }}>
                                        <StarFilled style={{ fontSize: 8 }} />{coreInGroup}
                                      </span>
                                    )}
                                  </div>
                                  {/* 分组内容 */}
                                  {!collapsed && g.facts.map((f: any) => (
                                    <div key={f.id}
                                      onClick={() => setShowMemoryPage(true)}
                                      style={{
                                        padding: '6px 8px 6px 20px', margin: '1px 0', borderRadius: 8, cursor: 'pointer',
                                        background: f.tier === 'core' ? `${C('color-primary')}06` : 'transparent',
                                        borderLeft: f.tier === 'core' ? `2px solid #faad14` : '2px solid transparent',
                                        transition: 'background 150ms',
                                      }}
                                      onMouseEnter={e => { if (f.tier !== 'core') e.currentTarget.style.background = `${C('color-bg-elevated')}40` }}
                                      onMouseLeave={e => { if (f.tier !== 'core') e.currentTarget.style.background = 'transparent' }}
                                    >
                                      <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                                        {f.tier === 'core' && <StarFilled style={{ color: '#faad14', fontSize: 9 }} />}
                                        <span style={{ fontSize: 11, fontWeight: 600, color: C('color-text'), flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                                          {f.subject}
                                        </span>
                                        <span style={{ fontSize: 9, color: C('color-text-secondary'), opacity: 0.45, flexShrink: 0 }}>
                                          {SUB_LABELS[f.subcategory] || f.subcategory || ''}
                                        </span>
                                      </div>
                                      <div style={{ fontSize: 9, color: C('color-text-secondary'), marginTop: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', opacity: 0.7 }}>
                                        {f.summary?.slice(0, 50)}{f.summary?.length > 50 ? '…' : ''}
                                      </div>
                                      <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 2 }}>
                                        <span style={{ fontSize: 8, color: C('color-text-secondary'), opacity: 0.5 }}>W{f.weight?.toFixed?.(1) ?? '—'}</span>
                                        {f.emotionalContext?.valence != null && (
                                          <span style={{ fontSize: 8, color: f.emotionalContext.valence > 0.2 ? '#52c41a' : f.emotionalContext.valence < -0.2 ? '#ff4d4f' : '#8c8c8c' }}>
                                            {f.emotionalContext.valence > 0.2 ? '😊' : f.emotionalContext.valence < -0.2 ? '😔' : '😐'}
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
                      </div>
                    )
                  })(),
                },
                {
                  key: 'trace',
                  label: <span style={{ fontSize: 11, display: 'flex', alignItems: 'center', gap: 3 }}>📊 追踪</span>,
                  children: <WhisperTracePanel traces={traces} currentTurn={totalTurns} />,
                },
              ]}
            />
          )}
        </aside>

        {/* 侧边栏折叠按钮 */}
        <div style={{
          position: 'absolute', right: sidebarCollapsed ? 0 : 282, top: '50%',
          transform: 'translateY(-50%)', zIndex: 15,
          transition: 'right 0.25s ease',
        }}>
          <Tooltip title={sidebarCollapsed ? '展开侧边栏' : '折叠侧边栏'}>
            <Button type="text" size="small"
              onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
              icon={sidebarCollapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
              style={{
                width: 24, height: 48, borderRadius: '6px 0 0 6px',
                background: 'var(--whisper-glass-bg)', backdropFilter: 'blur(10px)',
                border: '1px solid var(--whisper-glass-border)', borderRight: 'none',
                color: C('color-text-secondary'), fontSize: 12,
                display: 'flex', alignItems: 'center', justifyContent: 'center',
              }} />
          </Tooltip>
        </div>
      {/* 设置弹窗 */}
      <Modal
        open={showSettings}
        onCancel={() => setShowSettings(false)}
        footer={null}
        width={520}
        centered
        destroyOnClose
      >
        <WhisperSettingsPanel
          activePersonality={activePersonality}
          personalities={personalities}
          adultMode={adultMode}
          engineID={engineName}
          onPersonalityChange={(id) => handleSwitchPersonality(id)}
          onAdultModeChange={async (v) => { setAdultMode(v); try { await (App as any).WhisperSetAdultMode?.(activePersonality, v) } catch (_) {} }}
          onClearSession={handleClearMessages}
        />
      </Modal>

      {/* 语音设置弹窗 */}
      <Modal
        title="语音设置"
        open={showVoiceSettings}
        onCancel={() => setShowVoiceSettings(false)}
        footer={null}
        width={480}
        centered
        destroyOnClose
      >
        <VoiceSettingsPanel />
      </Modal>


      {/* 记忆管理弹窗 */}
      <Modal title={null} open={showMemoryPage} onCancel={() => setShowMemoryPage(false)}
        footer={null} width={720} centered bodyStyle={{ maxHeight: '70vh', overflow: 'auto' }}>
        <WhisperMemoryModal facts={facts} personalityID={activePersonality}
          onFactsChange={setFacts} />
      </Modal>

      {/* 虚拟助手管理中心 */}
      <AssistantManagerModal
        open={personalityOpen}
        activePersonality={activePersonality}
        adultMode={adultMode}
        onClose={() => setPersonalityOpen(false)}
        onSwitchPersonality={(id) => handleSwitchPersonality(id)}
      />

      {/* 语音对话浮层（轻语板块承载语音管道，对话目标 = gaea） */}
      {voiceOpen && (
        <div style={{
          position: 'fixed', inset: 0, zIndex: 200,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          background: 'rgba(0,0,0,0.35)', backdropFilter: 'blur(6px)',
        }} onClick={() => closeVoice()}>
          <div
            className="md-glass-strong"
            onClick={e => e.stopPropagation()}
            style={{
              width: 440, maxWidth: '92vw', borderRadius: 24,
              padding: '20px 26px 22px',
              display: 'flex', flexDirection: 'column', alignItems: 'center',
              boxShadow: '0 20px 60px rgba(0,0,0,0.35), 0 0 40px color-mix(in srgb, var(--gaea-glow) 20%, transparent)',
              animation: 'launcherFadeUp 0.3s cubic-bezier(0.16, 1, 0.3, 1)',
            }}
          >
            {/* 标题行 */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, width: '100%', marginBottom: 2 }}>
              <span className="live-dot" />
              <Typography.Text strong style={{ fontSize: 14, color: 'var(--md-sys-color-text)', letterSpacing: '0.04em' }}>
                语音对话
              </Typography.Text>
              <span style={{
                fontSize: 10, padding: '1px 7px', borderRadius: 9,
                background: 'color-mix(in srgb, var(--gaea-glow) 14%, transparent)',
                color: 'var(--gaea-glow)', border: '1px solid color-mix(in srgb, var(--gaea-glow) 30%, transparent)',
                fontWeight: 500, letterSpacing: '0.06em',
              }}>
                平台 AI 助手 gaea
              </span>
              <div style={{ flex: 1 }} />
              <Button type="text" size="small" icon={<CloseOutlined />} onClick={() => closeVoice()}
                style={{ color: 'var(--md-sys-color-text-secondary)', width: 26, height: 26, padding: 0 }} />
            </div>

            {/* 语言粒子球 */}
            <VoiceChatOrb
              volume={voice.volume}
              listening={voice.listening}
              speaking={voice.speaking}
              aiSpeaking={voice.aiSpeaking}
              transcript={voice.transcript}
              size={252}
            />

            {/* 状态行 */}
            <div style={{ minHeight: 22, marginTop: 2, fontSize: 12, fontWeight: 500, letterSpacing: '0.05em',
              color: voice.aiSpeaking ? '#64b5f6' : voice.listening ? '#ff8a65' : 'var(--md-sys-color-text-secondary)' }}>
              {voice.aiSpeaking ? 'AI 回复中…' : voice.listening ? '正在聆听…' : '待机（请说话）'}
            </div>

            {/* 对话文本 */}
            <div style={{ width: '100%', minHeight: 66, maxHeight: 140, overflowY: 'auto', marginTop: 8,
              display: 'flex', flexDirection: 'column', gap: 8 }}>
              {voiceUserText && (
                <div style={{ alignSelf: 'flex-end', maxWidth: '85%', padding: '7px 12px', borderRadius: 14,
                  background: 'linear-gradient(135deg, color-mix(in srgb, var(--gaea-glow) 22%, transparent), color-mix(in srgb, var(--gaea-glow) 10%, transparent))',
                  border: '1px solid color-mix(in srgb, var(--gaea-glow) 32%, transparent)',
                  color: 'var(--md-sys-color-text)', fontSize: 13, lineHeight: 1.55 }}>
                  {voiceUserText}
                </div>
              )}
              {voiceReply && (
                <div style={{ alignSelf: 'flex-start', maxWidth: '88%', padding: '7px 12px', borderRadius: 14,
                  background: 'var(--md-sys-color-surface-container-high)',
                  border: '1px solid var(--md-sys-color-outline-variant)',
                  color: 'var(--md-sys-color-text)', fontSize: 13, lineHeight: 1.6, wordBreak: 'break-word' }}>
                  <RobotOutlined style={{ marginRight: 6, color: 'var(--gaea-glow)' }} />{voiceReply}
                </div>
              )}
              {!voiceUserText && !voiceReply && (
                <div style={{ textAlign: 'center', color: 'var(--md-sys-color-text-secondary)', fontSize: 12, opacity: 0.7, padding: '14px 0' }}>
                  语言粒子汇聚成声 —— 说话即可与 gaea 对话
                </div>
              )}
            </div>

            {voice.error && (
              <Typography.Text style={{ color: '#fb7185', fontSize: 12, marginTop: 4 }}>{voice.error}</Typography.Text>
            )}

            {/* 控制区 */}
            <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginTop: 14 }}>
              {voice.aiSpeaking && (
                <Button shape="round" icon={<StopOutlined />} onClick={interrupt}
                  style={{ border: '1px solid color-mix(in srgb, #fb7185 45%, transparent)', color: '#fb7185',
                    background: 'color-mix(in srgb, #fb7185 10%, transparent)', fontSize: 13 }}>
                  打断回复
                </Button>
              )}
              <Button type="primary" danger icon={<StopOutlined />} onClick={() => closeVoice()}
                style={{ borderRadius: 22, padding: '4px 24px', height: 40, fontSize: 13 }}>
                结束语音对话
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

export default WhisperPage
