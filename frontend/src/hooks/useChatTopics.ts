// ChatPage 拆分产物：话题/模式/人格元数据状态机（行为零变化，T6-10.1）。
// 原 ChatPage 内 loadTopic / 初始化 / selectTopic / create/delete/rename /
// switchMode / finalizeTopicAfterSend / emotion-aff-aro 全部原样搬入，
// 仅把需要外部 state（messages/personalities）的地方改为注入回调。
import { useCallback, useEffect, useRef, useState } from 'react'
import { message } from 'antd'
import * as App from '../wailsjsCompat'
import { sortByUpdatedAtDesc, autoTopicTitle } from '../utils/chatTopics'
import { ACTIVE_TOPIC_KEY, PERSONALITY_KEY } from '../pages/chat/constants'
import { toUpdatedAt, loadPersonality, migrateLegacyTopics, logFrontendError, parseExtra } from '../pages/chat/utils'
import type { ChatMsg, Personality } from '../pages/chat/types'
import type { chat, whisper } from '../../wailsjs/go/models'

export interface UseChatTopicsOptions {
  setMessages: React.Dispatch<React.SetStateAction<ChatMsg[]>>
  setPersonalities: React.Dispatch<React.SetStateAction<Personality[]>>
}

export function useChatTopics({ setMessages, setPersonalities }: UseChatTopicsOptions) {
  const [topics, setTopics] = useState<chat.Topic[]>([])
  const [activeId, setActiveId] = useState<string>('')
  const [mode, setMode] = useState<string>('plain') // 'plain' | personaID
  // 人格元数据（只读展示，不操纵）
  const [emotion, setEmotion] = useState('')
  const [aff, setAff] = useState(0); const [aro, setAro] = useState(0)
  const [initializing, setInitializing] = useState(true)

  const activeIdRef = useRef('')
  activeIdRef.current = activeId
  const modeRef = useRef('plain')
  modeRef.current = mode
  const topicsRef = useRef<chat.Topic[]>([])
  topicsRef.current = topics
  const initRef = useRef(false)
  // 话题载入序号：快速切换话题时丢弃过期响应，避免旧话题消息覆盖当前视图。
  const topicLoadSeqRef = useRef(0)

  const resetPersonaMeta = useCallback(() => {
    setEmotion(''); setAff(0); setAro(0)
  }, [])

  // 载入话题消息并恢复模式/情绪元数据（初始进入与切换话题共用）。
  const loadTopic = useCallback(async (id: string, list: chat.Topic[]) => {
    const seq = ++topicLoadSeqRef.current
    let ms: chat.Message[] = []
    try {
      ms = (await App.ChatMessagesList(id)) || []
    } catch (err: unknown) {
      // T6-3.2：消息列表读取失败不再静默——记录后按空消息继续（不打断页面功能）
      logFrontendError('话题消息读取失败: ' + (err instanceof Error ? err.message : String(err)))
    }
    if (seq !== topicLoadSeqRef.current) return
    const loaded: ChatMsg[] = ms.map((m: chat.Message) => ({
      key: `db_${m.id}`, role: m.role === 'user' ? 'user' : 'assistant',
      content: m.content || '', createdAt: m.created_at || '', extra: parseExtra(m.extra),
    }))
    setMessages(loaded)
    const topic = list.find(t => t.id === id)
    const topicMode = topic?.mode || 'plain'
    if (topicMode !== modeRef.current) setMode(topicMode)
    resetPersonaMeta()
    if (topicMode !== 'plain') {
      const last = [...loaded].reverse().find(m => m.role === 'assistant' && m.extra)
      const lastEmotion = last?.extra?.emotion
      if (typeof lastEmotion === 'string') setEmotion(lastEmotion)
    }
  }, [resetPersonaMeta, setMessages])

  // ── 初始化：话题列表 + 旧数据迁移 + 人格列表 + 首页角色入口 ──
  useEffect(() => {
    if (initRef.current) return
    initRef.current = true
    ;(async () => {
      let list: chat.Topic[] = []
      const errText = (err: unknown) => err instanceof Error ? err.message : String(err)
      try { list = (await App.ChatTopicsList()) || [] } catch (err: unknown) {
        // T6-3.2：话题列表读取失败不再静默——记录后按空列表继续初始化
        logFrontendError('话题列表读取失败: ' + errText(err))
      }
      if (list.length === 0) {
        const imported = await migrateLegacyTopics()
        try { list = (await App.ChatTopicsList()) || [] } catch (err: unknown) {
          logFrontendError('话题列表读取失败（迁移后）: ' + errText(err))
        }
        if (!imported && list.length === 0) {
          try { await App.ChatTopicCreate('新对话', 'plain') } catch (err: unknown) {
            logFrontendError('话题创建失败: ' + errText(err))
          }
          try { list = (await App.ChatTopicsList()) || [] } catch (err: unknown) {
            logFrontendError('话题列表读取失败（创建后）: ' + errText(err))
          }
        }
      }
      // 最近活跃优先：默认把最新会话排到顶部。
      list = sortByUpdatedAtDesc(list, (t) => toUpdatedAt(t))
      setTopics(list)
      let first: chat.Topic | undefined = list[0]
      try {
        const last = localStorage.getItem(ACTIVE_TOPIC_KEY)
        if (last) first = list.find(t => t.id === last) || first
      } catch (_) {}
      if (first) {
        setActiveId(first.id)
        await loadTopic(first.id, list)
      }
      setInitializing(false)
    })()
    try {
      App.WhisperGetPersonalities().then((ps: whisper.PersonalityPreset[]) => setPersonalities(ps || [])).catch((err: unknown) => {
        logFrontendError('人格列表读取失败: ' + (err instanceof Error ? err.message : String(err)))
      })
    } catch (_) {}
  }, [loadTopic, setPersonalities])

  // 选择话题（数据部分）：切换 activeId + 恢复模式/情绪元数据；外部负责
  // 中止模拟打字流 / 滚动定位 / 输入框聚焦（见 ChatPage.selectTopic 包装）。
  const selectTopic = useCallback(async (id: string) => {
    if (!id || id === activeIdRef.current) return
    setActiveId(id)
    try { localStorage.setItem(ACTIVE_TOPIC_KEY, id) } catch (_) {}
    await loadTopic(id, topicsRef.current)
  }, [loadTopic])

  const createTopic = useCallback(async () => {
    try {
      const t = await App.ChatTopicCreate('新对话', modeRef.current)
      setTopics(prev => [t, ...prev])
      setActiveId(t.id)
      try { localStorage.setItem(ACTIVE_TOPIC_KEY, t.id) } catch (_) {}
      setMessages([])
      resetPersonaMeta()
      return t
    } catch (err: unknown) {
      message.error(`创建话题失败：${err instanceof Error ? err.message : String(err)}`)
      return null
    }
  }, [resetPersonaMeta, setMessages])

  const deleteTopic = useCallback(async (id: string) => {
    try { await App.ChatTopicDelete(id) } catch (err: unknown) { message.error(`删除失败：${err instanceof Error ? err.message : String(err)}`); return }
    const remaining = topicsRef.current.filter(t => t.id !== id)
    if (remaining.length === 0) {
      await createTopic()
      return
    }
    setTopics(remaining)
    if (id === activeIdRef.current) {
      const idx = topicsRef.current.findIndex(t => t.id === id)
      const next = remaining[Math.min(idx, remaining.length - 1)]
      await selectTopic(next.id)
    }
  }, [createTopic, selectTopic])

  const renameTopic = useCallback(async (id: string, title: string) => {
    try { await App.ChatTopicRename(id, title) } catch (err: unknown) {
      message.error(`重命名失败：${err instanceof Error ? err.message : String(err)}`)
      return
    }
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

  // 自动命名 + 最近活跃置顶 + 侧栏预览同步（发送成功后的统一收尾）。
  const finalizeTopicAfterSend = useCallback(async (topicId: string, firstUserText: string) => {
    const topic = topicsRef.current.find(t => t.id === topicId)
    const title = topic?.title === '新对话'
      ? autoTopicTitle(firstUserText)
      : undefined
    if (title) {
      try { await App.ChatTopicRename(topicId, title) } catch (_) {}
    }
    setTopics(prev => {
      const item = prev.find(t => t.id === topicId)
      if (!item) return prev
      const updated = { ...item, updated_at: new Date().toISOString(), preview: item.preview || firstUserText }
      if (title) updated.title = title
      return [updated, ...prev.filter(t => t.id !== topicId)]
    })
  }, [])

  return {
    topics, setTopics, activeId, activeIdRef, mode, modeRef, topicsRef,
    emotion, setEmotion, aff, setAff, aro, setAro,
    initializing, resetPersonaMeta, loadTopic, selectTopic,
    createTopic, deleteTopic, renameTopic, switchMode, finalizeTopicAfterSend,
  }
}
