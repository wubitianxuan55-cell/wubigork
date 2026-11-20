// ChatPage（T6-10.1 巨型组件拆分后的编排层，行为零变化）
// 职责：状态编排 + 跨 hook/组件装配；真实流订阅（useChatStream）、话题状态机
// （useChatTopics）、语音集成（useChatVoice）与纯展示组件拆分见各产物文件。
import React, { useState, useCallback, useEffect, useRef, useMemo } from 'react'
import { createPortal } from 'react-dom'
import { Typography, message, Input } from 'antd'
import { DownOutlined } from '@ant-design/icons'
import { app } from '../gaea/lib/bridge'
import { b64ToBytes } from '../gaea/lib/bytes'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { copyToClipboard } from '../gaea/lib/clipboard'
import { useFollowScroll } from '../utils/followScroll'
import { useBoardActive } from '../lib/boardActive'
import { C } from '../utils/theme'
import { shouldSubmitOnEnter } from '../utils/chatComposer'
import { useAppStore } from '../stores/appStore'
import ChatTopicSidebar, { type Topic as SidebarTopic } from '../components/ChatTopicSidebar'
import { mdStyles } from '../components/MarkdownContent'
import { ParticleFlow } from '../components/ParticleFlow'
import { SoundWaveOverlay } from '../components/SoundWaveOverlay'
import { useChatTopics } from '../hooks/useChatTopics'
import { useChatStream } from '../hooks/useChatStream'
import { ChatModeBar } from '../components/chat/ChatModeBar'
import { ChatPersonaBar } from '../components/chat/ChatPersonaBar'
import { MessageList } from '../components/chat/MessageList'
import { WelcomeScreen } from '../components/chat/WelcomeScreen'
import { ChatComposer, QUICK_REPLIES } from '../components/chat/ChatComposer'
const EMPTY_REPLIES: Array<{ label: string; text: string }> = []
import { ChatInspector } from '../components/chat/ChatInspector'
import VoiceConsoleV2 from '../components/chat/VoiceConsoleV2'
import { SearchPanel, type SourcePanelData } from '../components/chat/SearchPanel'
import {
  STREAM_INACTIVITY_TIMEOUT_MS, STREAM_HARD_TIMEOUT_MS,
  CHAT_ENGINE_START_TIMEOUT_MS, EMO_COLORS,
} from './chat/constants'
import { navigateToCharacterLib, loadCompanionName, toUpdatedAt, loadPersonality } from './chat/utils'
import { sortByUpdatedAtDesc } from '../utils/chatTopics'
import { btOf, dbIdOf } from './chat/branches'
import type { ChatMsg, Personality } from './chat/types'
import '../chat-board.css'
import '../chat-workspace.css'
import '../chat-signal-redesign.css'
import { GenuiScopeProvider } from '../genui/scope'
import { GenuiActionProvider } from '../genui/GenuiActionContext'

// T6-3.1：流式对话无帧超时（30s 无任何帧即视为失败）。测试从 ChatPage 导入
// （vitest fake timers 推进同一阈值），故在此再导出。
export {
  STREAM_INACTIVITY_TIMEOUT_MS, STREAM_HARD_TIMEOUT_MS,
  CHAT_ENGINE_START_TIMEOUT_MS,
}

function initialCollapsed(key: string): boolean {
  try {
    const saved = localStorage.getItem(key)
    if (saved !== null) return saved === '1'
  } catch {
    // localStorage 不可用时按视口给出一致初值。
  }
  return typeof window !== 'undefined' && window.innerWidth < 900
}

const ChatPage: React.FC = () => {
  const chatActive = useBoardActive('chat')
  // ── 消息区状态（页面持有；流/话题/语音三个 hook 注入更新） ──
  const [messages, setMessages] = useState<ChatMsg[]>([])
  const updateMessage = useCallback((key: string, patch: Partial<ChatMsg>) => {
    setMessages(prev => prev.map(m => m.key === key ? { ...m, ...patch } : m))
  }, [])

  // ── 话题状态机（含模式/情绪元数据/初始化） ──
  const [personalities, setPersonalities] = useState<Personality[]>([])
  const topicsApi = useChatTopics({ setMessages, setPersonalities })
  const {
    topics, setTopics, activeId, activeIdRef, mode, modeRef,
    emotion, setEmotion, aff, setAff, aro, setAro,
    initializing, resetPersonaMeta, selectTopic: selectTopicData,
    createTopic, deleteTopic, renameTopic, switchMode, finalizeTopicAfterSend, invalidateLoads,
    hasOlder, loadingOlder, loadOlder, prepend,
  } = topicsApi

  // G2 草稿按会话保留（chat-demo drafts.ts 语义对齐）：input 由「按话题键的
  // 草稿表」派生——切会话互不覆盖，流式 setInput('') 落在发送话题（activeIdRef
  // 当时值）。内存态不持久化（与会话列表同生命周期，重开应用按现状）。
  const [drafts, setDrafts] = useState<Record<string, string>>({})
  const input = drafts[activeId ?? ''] ?? ''
  const setInput = useCallback((v: string) => {
    setDrafts(prev => ({ ...prev, [activeIdRef.current ?? '']: v }))
  }, [activeIdRef])
  const [activePersonality, setActivePersonality] = useState<string>(() => loadPersonality())
  // 联网开关：默认关 + localStorage 持久化（2026-11-11 根治「重挂载静默回开」：
  // 旧 useState(true) 无持久化，切话题/重挂载后按钮状态与实际请求脱节，
  // 工具 schema 无感注入每轮请求——聊天页/轻语两链均受累）。
  const [searchEnabled, setSearchEnabled] = useState(() => {
    try { return localStorage.getItem('gaea.chatSearchEnabled') === '1' } catch { return false }
  })
  const [thinking, setThinking] = useState(false)
  const [forceSearch, setForceSearch] = useState(false)

  const [voiceConsoleOpen, setVoiceConsoleOpen] = useState(false)
  // Voice V2 档（M4 产品接入刀）：v2=实时 sidecar（VoiceConsoleV2）；读取一次，
  // 设置面板切档下次打开语音台生效（V1 全保留，回退=切回 v1）。
  const [copiedId, setCopiedId] = useState<string | null>(null)
  const [speakingId, setSpeakingId] = useState<string | null>(null)
  // v4.3d 朗读情绪：手动选择（'' = 跟随会话最近一轮情绪）；朗读时作为
  // TTSParams.Emotion 透传，无情绪回退无参数版（零值参数语义等价）。
  const [speakEmotion, setSpeakEmotion] = useState('')

  // ── 顶栏模式条宿主（T6-10.2）：ChatModeBar 移入 MainLayout 的 v3-strip，
  // 经 portal 渲染。宿主 DOM 在首帧提交后才存在，故挂载后查找一次即可
  // （MainLayout 恒挂载该容器，仅按板块切换显隐，节点不会重建）。
  const [modeBarHost, setModeBarHost] = useState<HTMLElement | null>(null)
  useEffect(() => {
    setModeBarHost(document.getElementById('v3-chatmode-host'))
  }, [])

  // 左侧会话栏折叠态（本地持久化，随面板一起折叠悬浮绑定模型卡）
  const [sidebarCollapsed, setSidebarCollapsed] = useState<boolean>(
    () => initialCollapsed('gaea.chatSidebarCollapsed'),
  )
  const toggleChatSidebar = useCallback(() => {
    setSidebarCollapsed((c) => {
      const next = !c
      try { localStorage.setItem('gaea.chatSidebarCollapsed', next ? '1' : '0') } catch { /* ignore */ }
      return next
    })
  }, [])

  // 右侧上下文/人格 inspector 折叠态（本地持久化，随面板折叠为窄条）
  const [inspectorCollapsed, setInspectorCollapsed] = useState<boolean>(
    () => initialCollapsed('gaea.chatInspectorCollapsed'),
  )
  const voiceInspectorRestoreRef = useRef<boolean | null>(null)

  // 语音模式暂时收起右侧上下文面板，把宽度让给声场；左侧会话索引继续可用，
  // 以便直接切换话题。退出时恢复原状态，不写入 localStorage。
  useEffect(() => {
    if (voiceConsoleOpen) {
      if (voiceInspectorRestoreRef.current === null) {
        voiceInspectorRestoreRef.current = inspectorCollapsed
      }
      setInspectorCollapsed(true)
      return
    }
    const restore = voiceInspectorRestoreRef.current
    if (restore === null) return
    voiceInspectorRestoreRef.current = null
    setInspectorCollapsed(restore)
  }, [voiceConsoleOpen, inspectorCollapsed])

  // The voice room owns the full content area while active. Keep app-level
  // chrome available, but release the status rail space for the conversation.
  useEffect(() => {
    const root = document.documentElement
    root.classList.toggle('gaea-voice-room-open', voiceConsoleOpen)
    return () => root.classList.remove('gaea-voice-room-open')
  }, [voiceConsoleOpen])

  useEffect(() => {
    const collapseForCompactWindow = () => {
      if (window.innerWidth >= 900) return
      setSidebarCollapsed(true)
      setInspectorCollapsed(true)
    }
    collapseForCompactWindow()
    window.addEventListener('resize', collapseForCompactWindow)
    return () => window.removeEventListener('resize', collapseForCompactWindow)
  }, [])
  // v4.365 性能轮：传给 memo 子组件的回调全部 useCallback 稳定（内联箭头使 memo 恒失效）
  const handleToggleSearch = useCallback(() => setSearchEnabled(v => {
    const next = !v
    try { localStorage.setItem('gaea.chatSearchEnabled', next ? '1' : '0') } catch { /* ignore */ }
    return next
  }), [])
  const handleToggleForceSearch = useCallback(() => setForceSearch(v => !v), [])
  const handleToggleThinking = useCallback(() => setThinking(v => !v), [])
  // 语音设置入口：直达聊天组的「朗读引擎」任务页签。
  const handleOpenVoiceSettings = useCallback(
    () => useAppStore.getState().openSettings('chat', 'speech-engine'),
    [],
  )
  const toggleChatInspector = useCallback(() => {
    setInspectorCollapsed((c) => {
      const next = !c
      try { localStorage.setItem('gaea.chatInspectorCollapsed', next ? '1' : '0') } catch { /* ignore */ }
      return next
    })
  }, [])

  // 角色音色预设同步（FE 角色音色线）：人格/模式变化即同步后端
  // personalityPresetID——朗读（TTSSpeak*）按当前人格解析音色预设。
  // P0a-A1（众神殿定稿）：同步改共享串行队列。旧实现 fire-and-forget 吞错
  // （.catch(()=>{})），注入侧人格只有尽力而无屏障；现在开台时把「这一笔」
  // 入队并等待该 Promise，写入被拒则拒绝开台（磁盘落盘失败仅后端告警，不在此列）。
  const modeForPreset = mode === 'plain' ? 'plain' : activePersonality
  const personaSyncChainRef = useRef<Promise<void>>(Promise.resolve())
  const enqueuePersonaSync = useCallback((presetId: string) => {
    const job = personaSyncChainRef.current.then(async () => {
      // Promise.resolve 包一层：mock 桩可能返回 undefined，防挂载期 TypeError
      await Promise.resolve(app.VoiceApplySettings({ personalityPresetId: presetId }))
    })
    personaSyncChainRef.current = job
    return job
  }, [])
  useEffect(() => {
    // 后台路径失败不打断输入主链；开台屏障会重新入队并以可见报错拦截
    enqueuePersonaSync(modeForPreset).catch(() => {})
  }, [modeForPreset, enqueuePersonaSync])

  const inputRef = useRef<React.ComponentRef<typeof Input.TextArea>>(null)
  // 智能滚动（v4.514 对齐办公板 v4.512 手势语义）：跟随清除权唯一归滚轮
  // 上滚——布局扰动（滚动锚定/夹紧的程序性 scroll 事件）杀不死跟随；
  // 贴底恢复；暂停态（上翻读历史）由「回到底部」按钮/滚回底部恢复。
  const { ref: listRef, atBottom, onScroll: onListScroll, onWheel: onListWheel, follow, rearm, scrollToBottom } = useFollowScroll<HTMLDivElement>()

  // 发送参数快照（doSend 在异步流期间读取，避免闭包过期）
  const searchEnabledRef = useRef(searchEnabled)
  searchEnabledRef.current = searchEnabled
  const thinkingRef = useRef(thinking)
  thinkingRef.current = thinking
  const forceSearchRef = useRef(forceSearch)
  forceSearchRef.current = forceSearch

  // ── 流式发送（plain/persona 同一真实流协议，逻辑在 hook 内） ──
  // A 批分支轮收口：终态后以 db 权威重载（回填 bt/兄弟版本/新行正式位次）。
  const reloadActiveTopic = useCallback(async () => {
    const id = activeIdRef.current
    if (!id) return
    await topicsApi.loadTopic(id, topicsApi.topicsRef.current)
    setBranchAnchor(null)
  }, [activeIdRef, topicsApi])
  const streamApi = useChatStream({
    updateMessage,
    setMessages,
    setInput,
    setEmotion,
    setAff,
    setAro,
    finalizeTopicAfterSend,
    reloadAfterBranch: reloadActiveTopic,
  })
  const { sending, streamKey, streamText, streamReasoning, streamStatus, send, stop, cancelTyping } = streamApi

  // ── A 批消息分支树（编辑提问/重新生成/版本切换） ──
  // branchAnchor：分支流乐观行插入锚（db_<parentID>），流中视图挂 mid-tree；
  // 终态 reload 后清零（reloadActiveTopic 内置）。
  const [branchAnchor, setBranchAnchor] = useState<string | null>(null)

  // D 批来源面板：单入口开合；切话题/新建即关（chat-demo 同款纪律）。
  const [sourcePanel, setSourcePanel] = useState<SourcePanelData | null>(null)
  const closeSourcePanel = useCallback(() => setSourcePanel(null), [])

  const handleEditSubmit = useCallback((msgKey: string, text: string) => {
    const row = messages.find(m => m.key === msgKey)
    if (!row || sending) return
    const parent = btOf(row)?.p ?? 0
    if (parent <= 0) {
      message.warning('首条提问暂不支持编辑')
      return
    }
    setBranchAnchor(`db_${parent}`)
    void send({
      text,
      mode: modeRef.current,
      active: activeIdRef.current,
      search: searchEnabledRef.current,
      thinking: thinkingRef.current,
      force: forceSearchRef.current,
      branchParentMsgID: parent,
    })
  }, [messages, sending, send, activeIdRef, modeRef, searchEnabledRef, thinkingRef, forceSearchRef])

  const handleRegenerate = useCallback((msgKey: string) => {
    const row = messages.find(m => m.key === msgKey)
    if (!row || sending) return
    const qid = btOf(row)?.p ?? 0
    if (qid <= 0) return
    setBranchAnchor(`db_${qid}`)
    void send({
      text: '',
      mode: modeRef.current,
      active: activeIdRef.current,
      search: searchEnabledRef.current,
      thinking: thinkingRef.current,
      force: forceSearchRef.current,
      branchRegenerateOf: qid,
    })
  }, [messages, sending, send, activeIdRef, modeRef, searchEnabledRef, thinkingRef, forceSearchRef])

  const handleSelectVersion = useCallback((msgId: number) => {
    const topicId = activeIdRef.current
    if (!topicId) return
    void app.ChatSelectVersion(topicId, msgId)
      .then(() => reloadActiveTopic())
      .catch((err: unknown) => message.error(`切换版本失败：${err instanceof Error ? err.message : String(err)}`))
  }, [activeIdRef, reloadActiveTopic])

  // A 批续写：已停止回答行 prefill 续写（chat-demo continueReply 子集）。
  // 独立事件订阅（协议复用 chat-stream:<runID>）：delta 限流拼进原行，
  // 终态以 db 权威 reload（Go 侧已拼接落库、extra.stopped 保留）。
  const [contBusy, setContBusy] = useState(false)
  const handleContinue = useCallback((msgKey: string) => {
    const row = messages.find(m => m.key === msgKey)
    const id = row ? dbIdOf(row.key) : null
    const topicId = activeIdRef.current
    if (!row || id === null || !topicId || contBusy || sending) return
    setContBusy(true)
    void (async () => {
      let settled = false
      let unsub: (() => void) | null = null
      let buf = ''
      let firstChunk = true
      let inactivity: ReturnType<typeof setTimeout> | null = null
      let hard: ReturnType<typeof setTimeout> | null = null
      const clearTimers = () => {
        if (inactivity !== null) clearTimeout(inactivity)
        if (hard !== null) clearTimeout(hard)
        inactivity = null; hard = null
      }
      const finish = () => {
        if (settled) return
        settled = true
        clearTimers()
        if (unsub !== null) { const f = unsub; unsub = null; f() }
        setContBusy(false)
      }
      try {
        const runID = await app.ChatContinue(topicId, id)
        if (settled) return
        const armInactivity = () => {
          if (inactivity !== null) clearTimeout(inactivity)
          inactivity = setTimeout(() => {
            void app.ChatStreamCancel(runID).catch(() => {})
            finish()
            void reloadActiveTopic()
          }, STREAM_INACTIVITY_TIMEOUT_MS)
        }
        unsub = EventsOn(`chat-stream:${runID}`, (payload: { type?: string; content?: string; error?: string }) => {
          if (settled) return
          const p = payload || {}
          if (p.type === 'delta') {
            armInactivity()
            buf += p.content || ''
            if (buf) {
              const chunk = buf
              buf = ''
              const sep = firstChunk ? '\n\n' : ''
              firstChunk = false
              setMessages(prev => prev.map(m => m.key === msgKey ? { ...m, content: m.content + sep + chunk } : m))
            }
          } else if (p.type === 'done' || p.type === 'cancelled') {
            finish()
            void reloadActiveTopic()
          } else if (p.type === 'error') {
            finish()
            message.error(`续写失败：${p.error || '未知错误'}`)
            void reloadActiveTopic()
          }
        })
        armInactivity()
        hard = setTimeout(() => {
          void app.ChatStreamCancel(runID).catch(() => {})
          finish()
          void reloadActiveTopic()
        }, STREAM_HARD_TIMEOUT_MS)
      } catch (err: unknown) {
        message.error(`续写失败：${err instanceof Error ? err.message : String(err)}`)
        setContBusy(false)
      }
    })()
  }, [messages, contBusy, sending, activeIdRef, reloadActiveTopic])

  // ── 语音台（V2 唯一；V1 whisper 链已退役 2026-11-05 用户拍板） ──
  useEffect(() => {
    if (chatActive) return
    setVoiceConsoleOpen(false) // V2 控制台卸载即断开（组件内自治）
  }, [chatActive])

  const openVoiceConsole = useCallback(async () => {
    if (!activeIdRef.current) {
      message.warning('当前会话尚未准备好，请稍后再试')
      return
    }
    // P0a-A1 开台屏障：捕获 {topicID, personalityID}，把这一笔人格同步入共
    // 享队列并等待该 Promise（不是等旧队尾——旧队尾可能还没把本次写入入队），
    // 成功后才挂载语音台；同步失败拒绝开台并可见报错。
    const topicID = activeIdRef.current
    try {
      await enqueuePersonaSync(modeForPreset)
    } catch (err) {
      message.error(`语音人格同步失败，已取消开启语音台：${err instanceof Error ? err.message : String(err)}`)
      return
    }
    if (activeIdRef.current !== topicID) return // 等待期间已切题，放弃开台
    setVoiceConsoleOpen(true) // V2 自管会话（VoiceConsoleV2 挂载即连接）
  }, [activeIdRef, enqueuePersonaSync, modeForPreset])

  const closeVoiceConsole = useCallback(() => {
    setVoiceConsoleOpen(false)
  }, [])
  const handleOpenVoiceConsole = useCallback(() => {
    void openVoiceConsole()
  }, [openVoiceConsole])
  const handleCloseVoiceConsole = useCallback(() => {
    closeVoiceConsole()
  }, [closeVoiceConsole])

  // ── 派生展示值 ──
  const currentPersonality = personalities.find(p => p.id === activePersonality)
  const companionName = useMemo(
    () => loadCompanionName(currentPersonality?.label || 'gaea'),
    [currentPersonality])
  // V2 人格音色（2026-11-01 实弹「声色没有固定」根修）：角色卡 TTSVoiceID
  // （breeze 克隆 id）→ session.update.audio.output.voice；无卡/无字段落服务端缺省。
  const [v2VoiceId, setV2VoiceId] = useState('')
  useEffect(() => {
    const pid = currentPersonality?.id
    if (!pid || pid === 'plain') { setV2VoiceId(''); return }
    void app.CharacterGet(pid).then((r) => {
      const ch = (r as { character?: { ttsVoiceId?: string } }).character
      setV2VoiceId(String(ch?.ttsVoiceId || ''))
    }).catch(() => setV2VoiceId(''))
  }, [currentPersonality?.id])
  // V2 人格指令：当前人格的 label/voiceGuide 折叠成 sidecar session instructions
  // （完整人设/记忆接 gaea 内部端点是下一刀；这里保底人格名与语气导引不丢）。
  const v2Instructions = useMemo(() => {
    const persona = currentPersonality?.label
      ? `你正在扮演「${currentPersonality.label}」这个角色。`
      : '你是 gaea 的语音对话助手。'
    const guide = currentPersonality?.voiceGuide ? `${currentPersonality.voiceGuide}` : ''
    return `${persona}${guide} 用口语化中文回答，每轮不超过两句话，不要列表和 Markdown，不要输出思考过程。`
  }, [currentPersonality])
  const emoColor = EMO_COLORS[emotion] || 'var(--gaea-glow, var(--md-sys-color-primary))'
  const personaLabel = currentPersonality?.label || '角色'
  const hasMessages = messages.length > 0
  const activeTopic = topics.find((topic) => topic.id === activeId)
  const workspaceTitle = voiceConsoleOpen
    ? '语音对话'
    : activeTopic?.title || (mode === 'plain' ? '新对话' : companionName)
  const workspaceStatus = voiceConsoleOpen
    ? '语音会话中'
    : sending
      ? '正在生成'
      : hasMessages
        ? `${messages.length} 条消息`
        : '等待开始'
  // 朗读生效情绪：手动选择优先，其次当前会话最近一轮情绪（whisper L2 标签）
  const effectiveSpeakEmotion = speakEmotion || emotion

  // ── 滚动跟随（新消息/流式文本变化：贴底保持吸底；暂停态自愈按钮显隐） ──
  useEffect(() => {
    follow()
  }, [messages, streamText, follow])

  // ── 朗读资源释放（T6-3.3：卸载时 revokeObjectURL） ──
  const speakUrlRef = useRef<string | null>(null)
  const revokeSpeakUrl = useCallback(() => {
    if (speakUrlRef.current) {
      URL.revokeObjectURL(speakUrlRef.current)
      speakUrlRef.current = null
    }
  }, [])
  useEffect(() => () => { revokeSpeakUrl() }, [revokeSpeakUrl])

  // ── 话题操作（滚动/聚焦包装；数据逻辑在 useChatTopics） ──
  // 选择话题 → 中止上一话题的模拟打字流 + 加载消息 + 恢复人格元数据
  const selectTopic = useCallback(async (id: string) => {
    closeVoiceConsole()
    closeSourcePanel()
    // T6-3.3：切话题即中止上一话题的模拟打字流（避免过期 setStreamText 继续跑）
    cancelTyping()
    rearm()
    await selectTopicData(id)
    inputRef.current?.focus?.()
  }, [cancelTyping, closeVoiceConsole, closeSourcePanel, selectTopicData, rearm])

  const handleCreate = useCallback(async () => {
    closeVoiceConsole()
    closeSourcePanel()
    rearm()
    await createTopic()
    inputRef.current?.focus?.()
  }, [closeVoiceConsole, closeSourcePanel, createTopic, rearm])

  const handleDelete = useCallback(async (id: string) => {
    if (id === activeIdRef.current) closeVoiceConsole()
    rearm()
    await deleteTopic(id)
    // G2：删除会话连带丢弃其草稿
    setDrafts(prev => {
      if (!(id in prev)) return prev
      const next = { ...prev }
      delete next[id]
      return next
    })
    inputRef.current?.focus?.()
  }, [activeIdRef, closeVoiceConsole, deleteTopic, rearm])

  const handleSwitchPersonality = useCallback(async (id: string) => {
    closeVoiceConsole()
    try { await app.WhisperClearSession(activePersonality) } catch (_) {}
    setActivePersonality(id)
    await switchMode(id)
  }, [activePersonality, closeVoiceConsole, switchMode])

  const handleSwitchPlain = useCallback(() => {
    closeVoiceConsole()
    void switchMode('plain')
  }, [closeVoiceConsole, switchMode])

  const handleSwitchPersona = useCallback(() => {
    closeVoiceConsole()
    if (mode === 'plain') void switchMode(activePersonality)
  }, [activePersonality, closeVoiceConsole, mode, switchMode])

  // E 档：设置面板导入替换历史后刷新（topics + 当前话题消息）
  useEffect(() => {
    const onReplaced = () => {
      invalidateLoads()
      void (async () => {
        const list = (await app.ChatTopicsList().catch(() => [])) || []
        const sorted = sortByUpdatedAtDesc(list, (t) => toUpdatedAt(t))
        setTopics(sorted)
        const active = activeIdRef.current
        if (active) await topicsApi.loadTopic(active, sorted)
      })()
    }
    window.addEventListener('gaea-chat-history-replaced', onReplaced)
    return () => window.removeEventListener('gaea-chat-history-replaced', onReplaced)
  }, [invalidateLoads, topicsApi])

  // 角色库切换人格 → 聊天板块联动
  useEffect(() => {
    const onPersona = (e: Event) => {
      const id = (e as CustomEvent).detail?.id
      if (!id) return
      closeVoiceConsole()
      setActivePersonality(id)
      switchMode(id)
    }
    window.addEventListener('gaea-persona-changed', onPersona)
    return () => window.removeEventListener('gaea-persona-changed', onPersona)
  }, [closeVoiceConsole, switchMode])

  // ── 发送管道 ──
  const doSend = useCallback((text: string, retryKey?: string) => {
    rearm()
    return send({
      text, retryKey,
      mode: modeRef.current,
      active: activeIdRef.current,
      search: searchEnabledRef.current,
      thinking: thinkingRef.current,
      force: forceSearchRef.current,
    })
  }, [send, modeRef, activeIdRef, searchEnabledRef, thinkingRef, forceSearchRef, rearm])

  const handleSend = useCallback(() => { doSend(input) }, [input, doSend])
  // GenUI action：聊天没有运行中 steer → 仅在本回合结束后发送；payload 折成
  // 可读文本进话题历史（模型按提示词规则解析 [UI 操作] 前缀）。
  const genuiChatAction = useCallback((action: string, payload: Record<string, unknown>) => {
    if (sending) return
    const extra = Object.entries(payload)
      .map(([k, v]) => `${k}=${String(v).slice(0, 160)}`)
      .join(' · ')
    const text = `[UI 操作] ${action}${extra ? `（${extra}）` : ''}`.slice(0, 500)
    if (!activeIdRef.current || !text) return
    void send({
      text,
      mode: modeRef.current,
      active: activeIdRef.current,
      search: searchEnabledRef.current,
      thinking: thinkingRef.current,
      force: forceSearchRef.current,
    })
  }, [sending, send, activeIdRef, modeRef])
  const handleFillInput = useCallback((label: string) => { setInput(label); inputRef.current?.focus?.() }, [])
  const handleRetry = useCallback((msgKey: string) => {
    const idx = messages.findIndex(m => m.key === msgKey)
    if (idx < 0) return
    const userMsg = messages.slice(0, idx).reverse().find(m => m.role === 'user')
    if (userMsg) doSend(userMsg.content, msgKey)
  }, [messages, doSend])

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (shouldSubmitOnEnter(e.key, e.shiftKey, e.nativeEvent.isComposing)) { e.preventDefault(); handleSend() }
  }

  const handleCopy = async (content: string, id: string) => {
    // v4.513 收敛进剪贴板单源：成功才翻转复制态，双路失败可见提示。
    if (await copyToClipboard(content)) {
      setCopiedId(id); setTimeout(() => setCopiedId(null), 2000)
    } else {
      message.warning('复制失败：剪贴板不可用')
    }
  }

  const handleSpeak = async (content: string, id: string) => {
    if (speakingId) return; setSpeakingId(id)
    try {
      // v4.3d：朗读携带当前生效情绪（TTSParams.Emotion 透传，后端按标签映射
      // 语速/音高/风格）；无情绪时回退无参数版 TTSSpeakBase64（引擎默认）。
      const result = effectiveSpeakEmotion
        ? await app.TTSSpeakBase64WithParams(content, { Speed: 0, Pitch: 0, Style: '', Emotion: effectiveSpeakEmotion })
        : await app.TTSSpeakBase64(content)
      if (result?.base64) {
        const bytes = b64ToBytes(result.base64 as string)
        // T6-3.3：blob URL 登记到 ref，播放结束/失败/卸载时 revokeObjectURL
        speakUrlRef.current = URL.createObjectURL(new Blob([bytes], { type: (result.mimeType as string) || 'audio/mp3' }))
        const audio = new Audio(speakUrlRef.current)
        audio.onended = () => { revokeSpeakUrl(); setSpeakingId(null) }
        audio.onerror = () => { revokeSpeakUrl(); setSpeakingId(null); message.error('播放失败') }
        await audio.play()
        return
      }
      message.warning('TTS 未返回音频数据')
    } catch (err: unknown) { message.error(`朗读失败：${typeof err === 'string' ? err : (err instanceof Error ? err.message : '未知错误')}`) }
    finally {
      // 播放结束/失败后释放（play() 已开始加载资源，revoke 不影响播放）
      revokeSpeakUrl()
      setSpeakingId(null)
    }
  }

  const handleClearMessages = useCallback(async () => {
    rearm()
    invalidateLoads() // v4.354：失效在途话题载入，防旧响应把清空后的视图重新填满
    const snapshot = messages // v4.361：清空失败时恢复（此前吞错后消息在切会话回来时「复活」）
    setMessages([]); resetPersonaMeta()
    if (activeIdRef.current) {
      try {
        await app.ChatTopicClear(activeIdRef.current)
      } catch (_) {
        setMessages(snapshot)
        message.error('清空失败，请重试')
        return
      }
    }
    setTopics(prev => prev.map(t => t.id === activeIdRef.current ? { ...t, preview: '' } : t))
    if (modeRef.current !== 'plain') {
      try { await app.WhisperClearSession(modeRef.current) } catch (_) {}
    }
  }, [resetPersonaMeta, activeIdRef, modeRef, setTopics, invalidateLoads, messages, rearm])

  const handleExport = useCallback(async () => {
    if (!activeIdRef.current) return
    try {
      const path: string = await app.ChatTopicExportMarkdown(activeIdRef.current)
      message.success(`已导出会话：${path}`)
      try { await navigator.clipboard.writeText(path) } catch (_) {}
    } catch (err: unknown) {
      message.error(`导出失败：${err instanceof Error ? err.message : String(err)}`)
    }
  }, [activeIdRef])

  // v4.350：memo 化——streamText 每 chunk（角色模式≈70 renders/秒）都触发本页
  // 重渲染，topicList 内联重建（含 O(T×P) personalities.find）使侧栏 props 恒新，
  // 数百会话×每行 Tooltip 全量重渲染。
  const topicList: SidebarTopic[] = useMemo(() => topics.map(t => ({
    id: t.id, title: t.title, createdAt: new Date(t.created_at || 0).getTime() || Date.now(),
    updatedAt: toUpdatedAt(t),
    mode: t.mode, modeLabel: t.mode === 'plain' ? '' : (personalities.find(p => p.id === t.mode)?.label || '角色'),
    preview: t.preview || '',
  })), [topics, personalities])

  // 顶栏模式切换条（T6-10.2）：渲染进全局轨道条宿主；行为与原先完全一致。
  const modeBar = (
    <ChatModeBar
      variant="strip"
      mode={mode}
      personaLabel={personaLabel}
      currentPersonalityLabel={currentPersonality?.label}
      personaPickerActiveId={mode !== 'plain' ? mode : activePersonality}
      searchEnabled={searchEnabled}
      onToggleSearch={handleToggleSearch}
      onNavigateLib={navigateToCharacterLib}
      onSwitchPlain={handleSwitchPlain}
      onSwitchPersona={handleSwitchPersona}
      onSwitchPersonality={handleSwitchPersonality}
      onOpenVoiceSettings={handleOpenVoiceSettings}
      speakEmotion={speakEmotion}
      sessionEmotion={emotion}
      onChangeSpeakEmotion={setSpeakEmotion}
      hasMessages={hasMessages}
      onExport={handleExport}
      onClear={handleClearMessages}
    />
  )

  return (
    <div
      className={`chat-board chat-cockpit chat-signal-redesign${voiceConsoleOpen ? ' chat-voice-mode' : ''}`}
      style={{ flex: 1, minHeight: 0, position: 'relative' }}
    >
      <a className="chat-skip-link" href={voiceConsoleOpen ? '#voice-workspace' : '#chat-workspace'}>
        {voiceConsoleOpen ? '跳到语音控制' : '跳到对话内容'}
      </a>
      {!voiceConsoleOpen && (
        <>
          <ChatTopicSidebar
            topics={topicList}
            activeId={activeId}
            onSelect={selectTopic}
            onCreate={handleCreate}
            onDelete={handleDelete}
            onRename={renameTopic}
            collapsed={sidebarCollapsed}
            onToggle={toggleChatSidebar}
          />
          <div className="v3-split-v" aria-hidden="true" />
        </>
      )}

      <main className={`chat-main v3-zone${voiceConsoleOpen ? ' chat-voice-room-main' : ''}`}>
        <style>{mdStyles}</style>
        {voiceConsoleOpen ? (
          <VoiceConsoleV2
            companionName={companionName}
            instructions={v2Instructions}
            voiceId={v2VoiceId}
            variant="full"
            topicTitle={topics.find(t => t.id === activeId)?.title}
            onClose={handleCloseVoiceConsole}
            onOpenSettings={handleOpenVoiceSettings}
            setMessages={setMessages}
            getActiveId={() => activeIdRef.current}
            finalizeTopicAfterSend={finalizeTopicAfterSend}
          />
        ) : (
          <>
            {mode !== 'plain' && <ParticleFlow aro={aro} />}
            <SoundWaveOverlay active={speakingId !== null} aff={aff} aro={aro} />
            <header className="chat-workspace-head">
              <div className="chat-workspace-identity">
                <span className="chat-workspace-kicker">当前会话</span>
                <div className="chat-workspace-title-row">
                  <h1 id="chat-workspace" tabIndex={-1}>{workspaceTitle}</h1>
                  <span className="chat-workspace-status" aria-live="polite">{workspaceStatus}</span>
                </div>
              </div>
              <div className="chat-workspace-mode" title="会话记录保存在本机">
                <span className="chat-workspace-mode-dot" aria-hidden />
                本机保存
              </div>
            </header>

            {/* 人格状态条（临场感：头像常驻 + 名字；状态/记忆归角色库） */}
            {mode !== 'plain' && hasMessages && (
              <ChatPersonaBar
                companionName={companionName}
                emoColor={emoColor}
                speaking={speakingId !== null}
                thinking={sending}
                activeId={mode !== 'plain' ? mode : activePersonality}
                onSelect={handleSwitchPersonality}
                onManage={navigateToCharacterLib}
              />
            )}

            {/* 消息区 */}
            <div ref={listRef} className="chat-message-scroll" role="log" aria-live="polite" aria-busy={sending} aria-label="对话消息"
              onScroll={onListScroll}
              onWheel={onListWheel}
              style={{ flex: 1, minHeight: 0, overflow: 'auto', padding: 0, position: 'relative' }}>
              {initializing ? (
                <div className="chat-empty">
                  <span className="typing-dots"><span className="typing-dot" /><span className="typing-dot" /><span className="typing-dot" /></span>
                  <Typography.Text style={{ color: C('color-text-secondary'), fontSize: 12, marginTop: 14 }}>正在载入会话…</Typography.Text>
                </div>
              ) : !hasMessages ? (
                <WelcomeScreen
                  mode={mode}
                  personaLabel={personaLabel}
                  companionName={companionName}
                  emoColor={emoColor}
                  activePersonality={activePersonality}
                  onSwitchPersonality={handleSwitchPersonality}
                  onNavigateLib={navigateToCharacterLib}
                  onFillInput={handleFillInput}
                  onOpenVoiceConsole={handleOpenVoiceConsole}
                />
              ) : (
                <GenuiScopeProvider scope={activeId ? { scope: 'chat', sessionKey: activeId } : null}>
                  <GenuiActionProvider onAction={genuiChatAction}>
                    <MessageList
                      hasOlder={hasOlder}
                      loadingOlder={loadingOlder}
                      onLoadOlder={loadOlder}
                      prepend={prepend}
                      messages={messages}
                      streamKey={streamKey}
                      streamText={streamText}
                      streamReasoning={streamReasoning}
                      streamStatus={streamStatus}
                      mode={mode}
                      companionName={companionName}
                      copiedId={copiedId}
                      speakingId={speakingId}
                      onCopy={handleCopy}
                      onSpeak={handleSpeak}
                      onRetry={handleRetry}
                      onSelectVersion={handleSelectVersion}
                      onEditSubmit={handleEditSubmit}
                      onRegenerate={handleRegenerate}
                      branchAnchorKey={branchAnchor}
                      onOpenSources={(refs, queries) => setSourcePanel({ refs, queries })}
                      onContinue={handleContinue}
                      extraBusy={contBusy}
                    />
                  </GenuiActionProvider>
                </GenuiScopeProvider>
              )}
            </div>

            {!atBottom && hasMessages && (
              <button className="chat-scroll-bottom" onClick={scrollToBottom} aria-label="回到底部">
                <DownOutlined /> 回到底部
              </button>
            )}

            {/* D 批：搜索来源抽屉（覆盖式浮层，不挤消息布局） */}
            <SearchPanel data={sourcePanel} onClose={closeSourcePanel} />

            {/* 输入岛 */}
            <ChatComposer
              mode={mode}
              input={input}
              onInputChange={setInput}
              onKeyDown={handleKeyDown}
              inputRef={inputRef}
              voiceOpen={voiceConsoleOpen}
              onToggleVoice={handleOpenVoiceConsole}
              sending={sending}
              forceSearch={forceSearch}
              onToggleForceSearch={handleToggleForceSearch}
              thinking={thinking}
              onToggleThinking={handleToggleThinking}
              onSend={handleSend}
              onStop={stop}
              onFillInput={handleFillInput}
            />
          </>
        )}
      </main>

      {!voiceConsoleOpen && (
        <>
          <div className="v3-split-v" aria-hidden="true" />
          <ChatInspector
            mode={mode}
            personalities={personalities}
            activePersonality={activePersonality}
            companionName={companionName}
            emoColor={emoColor}
            messages={messages}
            speaking={speakingId !== null}
            thinking={sending}
            quickReplies={mode !== 'plain' ? QUICK_REPLIES : EMPTY_REPLIES}
            collapsed={inspectorCollapsed}
            onToggle={toggleChatInspector}
            onFillInput={handleFillInput}
            onSwitchPersonality={handleSwitchPersonality}
            onExport={handleExport}
            onClear={handleClearMessages}
            onOpenVoiceSettings={handleOpenVoiceSettings}
            onNavigateLib={navigateToCharacterLib}
          />
        </>
      )}

      {/* 模型状态统一由顶栏轨道条展示（3.0 定制：移除左下角悬浮模型卡） */}

      {/* 顶栏模式切换条（T6-10.2：portal 进 MainLayout 的 v3-strip 宿主） */}
      {modeBarHost !== null && !voiceConsoleOpen && createPortal(modeBar, modeBarHost)}

    </div>
  )
}

export default ChatPage
