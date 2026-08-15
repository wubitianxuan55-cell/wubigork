// ChatPage（T6-10.1 巨型组件拆分后的编排层，行为零变化）
// 职责：状态编排 + 跨 hook/组件装配；流订阅（useChatStream）、话题状态机
// （useChatTopics）、语音集成（useChatVoice）与纯展示组件拆分见各产物文件。
import React, { useState, useCallback, useEffect, useRef, useMemo } from 'react'
import { createPortal } from 'react-dom'
import { Typography, Modal, message, Input } from 'antd'
import { DownOutlined } from '@ant-design/icons'
import * as App from '../../src/wailsjsCompat'
import { C } from '../utils/theme'
import { shouldSubmitOnEnter } from '../utils/chatComposer'
import { isNearBottom } from '../utils/scroll'
import ChatTopicSidebar, { type Topic as SidebarTopic } from '../components/ChatTopicSidebar'
import { mdStyles } from '../components/MarkdownContent'
import { ParticleFlow } from '../components/ParticleFlow'
import { SoundWaveOverlay } from '../components/SoundWaveOverlay'
import VoiceSettingsPanel from '../components/VoiceSettingsPanel'
import { useChatTopics } from '../hooks/useChatTopics'
import { useChatStream } from '../hooks/useChatStream'
import { useChatVoice } from '../hooks/useChatVoice'
import { ChatModeBar } from '../components/chat/ChatModeBar'
import { ChatPersonaBar } from '../components/chat/ChatPersonaBar'
import { MessageList } from '../components/chat/MessageList'
import { WelcomeScreen } from '../components/chat/WelcomeScreen'
import { ChatComposer, QUICK_REPLIES } from '../components/chat/ChatComposer'
import { ChatInspector } from '../components/chat/ChatInspector'
import { STREAM_SILENCE_TIMEOUT_MS, EMO_COLORS } from './chat/constants'
import { navigateToCharacterLib, loadCompanionName, toUpdatedAt, loadPersonality } from './chat/utils'
import type { ChatMsg, Personality } from './chat/types'
import '../chat-board.css'

// T6-3.1：流式对话无帧超时（30s 无任何帧即视为失败）。测试从 ChatPage 导入
// （vitest fake timers 推进同一阈值），故在此再导出。
export { STREAM_SILENCE_TIMEOUT_MS }

const ChatPage: React.FC = () => {
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
    createTopic, deleteTopic, renameTopic, switchMode, finalizeTopicAfterSend,
  } = topicsApi

  const [input, setInput] = useState('')
  const [activePersonality, setActivePersonality] = useState<string>(() => loadPersonality())
  const [searchEnabled, setSearchEnabled] = useState(true)
  const [thinking, setThinking] = useState(false)
  const [forceSearch, setForceSearch] = useState(false)

  const [showVoiceSettings, setShowVoiceSettings] = useState(false)
  const [copiedId, setCopiedId] = useState<string | null>(null)
  const [speakingId, setSpeakingId] = useState<string | null>(null)

  // ── 顶栏模式条宿主（T6-10.2）：ChatModeBar 移入 MainLayout 的 v3-strip，
  // 经 portal 渲染。宿主 DOM 在首帧提交后才存在，故挂载后查找一次即可
  // （MainLayout 恒挂载该容器，仅按板块切换显隐，节点不会重建）。
  const [modeBarHost, setModeBarHost] = useState<HTMLElement | null>(null)
  useEffect(() => {
    setModeBarHost(document.getElementById('v3-chatmode-host'))
  }, [])

  // 左侧会话栏折叠态（本地持久化，随面板一起折叠悬浮绑定模型卡）
  const [sidebarCollapsed, setSidebarCollapsed] = useState<boolean>(() => {
    try { return localStorage.getItem('gaea.chatSidebarCollapsed') === '1' } catch { return false }
  })
  const toggleChatSidebar = useCallback(() => {
    setSidebarCollapsed((c) => {
      const next = !c
      try { localStorage.setItem('gaea.chatSidebarCollapsed', next ? '1' : '0') } catch { /* ignore */ }
      return next
    })
  }, [])

  // 右侧上下文/人格 inspector 折叠态（本地持久化，随面板折叠为窄条）
  const [inspectorCollapsed, setInspectorCollapsed] = useState<boolean>(() => {
    try { return localStorage.getItem('gaea.chatInspectorCollapsed') === '1' } catch { return false }
  })
  const toggleChatInspector = useCallback(() => {
    setInspectorCollapsed((c) => {
      const next = !c
      try { localStorage.setItem('gaea.chatInspectorCollapsed', next ? '1' : '0') } catch { /* ignore */ }
      return next
    })
  }, [])

  const listRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<React.ComponentRef<typeof Input.TextArea>>(null)
  // 智能滚动：用户上翻阅读时不强制吸底，靠近底部/新消息/切换话题才跟随。
  const [atBottom, setAtBottom] = useState(true)
  const stickToBottomRef = useRef(true)
  const onListScroll = useCallback(() => {
    const el = listRef.current
    if (!el) return
    const near = isNearBottom(el.scrollHeight - el.scrollTop - el.clientHeight)
    stickToBottomRef.current = near
    setAtBottom(near)
  }, [])
  const scrollToBottom = useCallback(() => {
    stickToBottomRef.current = true
    setAtBottom(true)
    if (listRef.current) listRef.current.scrollTop = listRef.current.scrollHeight
  }, [])

  // 发送参数快照（doSend 在异步流期间读取，避免闭包过期）
  const searchEnabledRef = useRef(searchEnabled)
  searchEnabledRef.current = searchEnabled
  const thinkingRef = useRef(thinking)
  thinkingRef.current = thinking
  const forceSearchRef = useRef(forceSearch)
  forceSearchRef.current = forceSearch

  // ── 流式发送（plain 流订阅 + 角色模拟打字，T6-3.1/T6-3.3 逻辑在 hook 内） ──
  const streamApi = useChatStream({
    updateMessage,
    setMessages,
    setInput,
    setEmotion,
    setAff,
    setAro,
    finalizeTopicAfterSend,
  })
  const { sending, streamKey, streamText, send, cancelTyping } = streamApi

  // ── 语音集成（T6-3.3 语音消息落库在 hook 内） ──
  const voiceApi = useChatVoice({
    setMessages,
    getActiveId: useCallback(() => activeIdRef.current, [activeIdRef]),
    getMode: useCallback(() => modeRef.current, [modeRef]),
  })
  const { voice, voiceOn, toggleVoice } = voiceApi

  // ── 派生展示值 ──
  const currentPersonality = personalities.find(p => p.id === activePersonality)
  const companionName = useMemo(
    () => loadCompanionName(currentPersonality?.label || 'gaea'),
    [currentPersonality])
  const emoColor = EMO_COLORS[emotion] || 'var(--gaea-glow, var(--md-sys-color-primary))'
  const personaLabel = currentPersonality?.label || '角色'
  const hasMessages = messages.length > 0

  // ── 滚动跟随（新消息/流式文本变化时若用户贴底则保持吸底） ──
  useEffect(() => {
    if (stickToBottomRef.current && listRef.current) {
      listRef.current.scrollTop = listRef.current.scrollHeight
    }
  }, [messages, streamText])

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
    // T6-3.3：切话题即中止上一话题的模拟打字流（避免过期 setStreamText 继续跑）
    cancelTyping()
    stickToBottomRef.current = true
    setAtBottom(true)
    await selectTopicData(id)
    inputRef.current?.focus?.()
  }, [cancelTyping, selectTopicData])

  const handleCreate = useCallback(async () => {
    stickToBottomRef.current = true
    setAtBottom(true)
    await createTopic()
    inputRef.current?.focus?.()
  }, [createTopic])

  const handleDelete = useCallback(async (id: string) => {
    stickToBottomRef.current = true
    setAtBottom(true)
    await deleteTopic(id)
    inputRef.current?.focus?.()
  }, [deleteTopic])

  const handleSwitchPersonality = useCallback(async (id: string) => {
    try { await App.WhisperClearSession(activePersonality) } catch (_) {}
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

  // ── 发送管道 ──
  const doSend = useCallback((text: string, retryKey?: string) => {
    stickToBottomRef.current = true
    setAtBottom(true)
    return send({
      text, retryKey,
      mode: modeRef.current,
      active: activeIdRef.current,
      search: searchEnabledRef.current,
      thinking: thinkingRef.current,
      force: forceSearchRef.current,
    })
  }, [send, modeRef, activeIdRef, searchEnabledRef, thinkingRef, forceSearchRef])

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
    if (shouldSubmitOnEnter(e.key, e.shiftKey, e.nativeEvent.isComposing)) { e.preventDefault(); handleSend() }
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
      const result = await App.TTSSpeakBase64(content)
      if (result?.base64) {
        const b = atob(result.base64); const bytes = new Uint8Array(b.length)
        for (let i = 0; i < b.length; i++) bytes[i] = b.charCodeAt(i)
        // T6-3.3：blob URL 登记到 ref，播放结束/失败/卸载时 revokeObjectURL
        speakUrlRef.current = URL.createObjectURL(new Blob([bytes], { type: result.mimeType || 'audio/mp3' }))
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
    stickToBottomRef.current = true
    setAtBottom(true)
    setMessages([]); resetPersonaMeta()
    if (activeIdRef.current) {
      try { await App.ChatTopicClear(activeIdRef.current) } catch (_) {}
    }
    setTopics(prev => prev.map(t => t.id === activeIdRef.current ? { ...t, preview: '' } : t))
    if (modeRef.current !== 'plain') {
      try { await App.WhisperClearSession(modeRef.current) } catch (_) {}
    }
  }, [resetPersonaMeta, activeIdRef, modeRef, setTopics])

  const handleExport = useCallback(async () => {
    if (!activeIdRef.current) return
    try {
      const path: string = await App.ChatTopicExportMarkdown(activeIdRef.current)
      message.success(`已导出会话：${path}`)
      try { await navigator.clipboard.writeText(path) } catch (_) {}
    } catch (err: unknown) {
      message.error(`导出失败：${err instanceof Error ? err.message : String(err)}`)
    }
  }, [activeIdRef])

  const topicList: SidebarTopic[] = topics.map(t => ({
    id: t.id, title: t.title, createdAt: new Date(t.created_at || 0).getTime() || Date.now(),
    updatedAt: toUpdatedAt(t),
    mode: t.mode, modeLabel: t.mode === 'plain' ? '' : (personalities.find(p => p.id === t.mode)?.label || '角色'),
    preview: t.preview || '',
  }))

  // 顶栏模式切换条（T6-10.2）：渲染进全局轨道条宿主；行为与原先完全一致。
  const modeBar = (
    <ChatModeBar
      variant="strip"
      mode={mode}
      personaLabel={personaLabel}
      currentPersonalityLabel={currentPersonality?.label}
      personaPickerActiveId={mode !== 'plain' ? mode : activePersonality}
      searchEnabled={searchEnabled}
      onToggleSearch={() => setSearchEnabled(!searchEnabled)}
      onNavigateLib={navigateToCharacterLib}
      onSwitchPlain={() => switchMode('plain')}
      onSwitchPersona={() => { if (mode === 'plain') switchMode(activePersonality) }}
      onSwitchPersonality={handleSwitchPersonality}
      onOpenVoiceSettings={() => setShowVoiceSettings(true)}
      hasMessages={hasMessages}
      onExport={handleExport}
      onClear={handleClearMessages}
    />
  )

  return (
    <div className="chat-board chat-cockpit" style={{ flex: 1, minHeight: 0, position: 'relative' }}>
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

      <main className="chat-main v3-zone">
        <style>{mdStyles}</style>
        {mode !== 'plain' && <ParticleFlow aro={aro} />}
        <SoundWaveOverlay active={speakingId !== null} aff={aff} aro={aro} />

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
        <div ref={listRef} role="log" aria-live="polite" aria-label="对话消息"
          onScroll={onListScroll}
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
              voice={voice}
              emoColor={emoColor}
              activePersonality={activePersonality}
              onSwitchPersonality={handleSwitchPersonality}
              onNavigateLib={navigateToCharacterLib}
              onFillInput={handleFillInput}
              onSuggestion={handleSuggestion}
            />
          ) : (
            <MessageList
              messages={messages}
              streamKey={streamKey}
              streamText={streamText}
              mode={mode}
              companionName={companionName}
              copiedId={copiedId}
              speakingId={speakingId}
              onCopy={handleCopy}
              onSpeak={handleSpeak}
              onRetry={handleRetry}
            />
          )}
        </div>

        {!atBottom && hasMessages && (
          <button className="chat-scroll-bottom" onClick={scrollToBottom} aria-label="回到底部">
            <DownOutlined /> 回到底部
          </button>
        )}

        {/* 输入岛 */}
        <ChatComposer
          mode={mode}
          input={input}
          onInputChange={setInput}
          onKeyDown={handleKeyDown}
          inputRef={inputRef}
          voiceOn={voiceOn}
          voiceTranscript={voice.transcript}
          onToggleVoice={toggleVoice}
          sending={sending}
          forceSearch={forceSearch}
          onToggleForceSearch={() => setForceSearch(v => !v)}
          thinking={thinking}
          onToggleThinking={() => setThinking(v => !v)}
          onSend={handleSend}
          onFillInput={handleFillInput}
        />
      </main>

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
        quickReplies={mode !== 'plain' ? QUICK_REPLIES : []}
        collapsed={inspectorCollapsed}
        onToggle={toggleChatInspector}
        onFillInput={handleFillInput}
        onSwitchPersonality={handleSwitchPersonality}
        onExport={handleExport}
        onClear={handleClearMessages}
        onOpenVoiceSettings={() => setShowVoiceSettings(true)}
        onNavigateLib={navigateToCharacterLib}
      />

      {/* 模型状态统一由顶栏轨道条展示（3.0 定制：移除左下角悬浮模型卡） */}

      <Modal title="语音设置" open={showVoiceSettings} onCancel={() => setShowVoiceSettings(false)} footer={null} width={480} centered
        destroyOnHidden transitionName="" maskTransitionName="">
        <VoiceSettingsPanel />
      </Modal>

      {/* 顶栏模式切换条（T6-10.2：portal 进 MainLayout 的 v3-strip 宿主） */}
      {modeBarHost !== null && createPortal(modeBar, modeBarHost)}

    </div>
  )
}

export default ChatPage
