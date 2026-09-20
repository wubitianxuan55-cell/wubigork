// sin/useSinStory.ts — 原罪故事状态机（闲庭 · 图文故事创作）。
//
// 职责：故事列表 / 当前故事消息 / 流式续写 / 插图回写 的编排。文本与插图都
// 走后端既有能力（SinStream 走功能级路由 sin，SinIllustrate 走绘梦图像后端），
// 本 hook 只做前端状态收敛，不做内容加工。
//
// 流式纪律与聊天板块一致（useChatStream 同范式）：先拿 runID 再订阅
// `sin-stream:<runID>`；无帧超时按失败收尾（长文生成，阈值放长到 90s）；
// 卸载/切故事必收尾，不留悬挂订阅与定时器。

import { useCallback, useEffect, useRef, useState } from 'react'
import { app } from '../../gaea/lib/bridge'
import { subscribe, sinStreamChannel } from '../../events'
import { parseIllustrations, parseReasoning, parseTools, suggestStoryTitle, toToolViews } from './storyText'
import type { SinMessageView, SinStoryView, SinStreamPayload, SinToolTraceView } from './types'

/** 无帧超时：故事单次输出可长达数千字，比聊天（30s）放宽到 90s。 */
export const SIN_STREAM_SILENCE_TIMEOUT_MS = 90_000

function errText(err: unknown, fallback: string): string {
  return (err instanceof Error && err.message) || fallback
}

function nowStr(): string {
  const d = new Date()
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
}

let msgSeq = 0
function nextKey(prefix: string): string {
  msgSeq += 1
  return `${prefix}_${Date.now()}_${msgSeq}`
}

/** Go chat.Topic → 视图（缺字段诚实留空，不编造）。 */
function toStoryView(raw: Record<string, unknown>): SinStoryView {
  return {
    id: typeof raw.id === 'string' ? raw.id : '',
    title: typeof raw.title === 'string' && raw.title ? raw.title : '新故事',
    createdAt: typeof raw.created_at === 'string' ? raw.created_at : '',
    updatedAt: typeof raw.updated_at === 'string' ? raw.updated_at : '',
    preview: typeof raw.preview === 'string' ? raw.preview : '',
  }
}

/** Go chat.Message → 视图（extra 里的插图映射/reasoning 解析出来）。 */
function toMessageView(raw: Record<string, unknown>): SinMessageView {
  const id = typeof raw.id === 'number' ? raw.id : 0
  return {
    key: `db_${id}`,
    messageId: id,
    role: raw.role === 'user' ? 'user' : 'assistant',
    content: typeof raw.content === 'string' ? raw.content : '',
    illustrations: parseIllustrations(raw.extra),
    reasoning: parseReasoning(raw.extra),
    tools: toToolViews(parseTools(raw.extra)),
    createdAt: typeof raw.created_at === 'string' ? raw.created_at : '',
  }
}

/**
 * 结果帧合并到哪一行：按 id 精确匹配；id 缺失时回退「最后一个同名运行中行」
 * （后端 id 恒有，这里只是兜底）。找不到 → -1（调用方补一行，别丢帧）。
 */
function findToolRow(tools: SinToolTraceView[], p: SinStreamPayload): number {
  const id = p.id ?? ''
  if (id) {
    const byId = tools.findIndex((t) => t.id === id)
    if (byId >= 0) return byId
  }
  for (let i = tools.length - 1; i >= 0; i -= 1) {
    if (tools[i].name === (p.name ?? '') && tools[i].status === 'running') return i
  }
  return -1
}

export interface UseSinStoryResult {
  stories: SinStoryView[]
  activeId: string
  activeStory: SinStoryView | undefined
  messages: SinMessageView[]
  initializing: boolean
  sending: boolean
  notice: string
  clearNotice: () => void
  showNotice: (message: string) => void
  selectStory: (id: string) => Promise<void>
  createStory: () => Promise<void>
  renameStory: (id: string, title: string) => Promise<void>
  deleteStory: (id: string) => Promise<void>
  clearStory: (id: string) => Promise<void>
  send: (text: string) => Promise<void>
  /**
   * 停止当前生成（Composer 停止按钮）：后端中止 + 本地立即解封输入。
   * 返回 undefined = 不回填输入框（原罪的指令已在发出时消费，无草稿可还）。
   */
  cancel: () => string | undefined
  setIllustration: (messageKey: string, cueKey: string, path: string) => void
  /** 从后端重读当前故事消息（画廊「重新生成」回写后刷新画廊与流内插图）。 */
  reloadMessages: () => Promise<void>
}

export function useSinStory(): UseSinStoryResult {
  const [stories, setStories] = useState<SinStoryView[]>([])
  const [activeId, setActiveId] = useState('')
  const [messages, setMessages] = useState<SinMessageView[]>([])
  const [initializing, setInitializing] = useState(true)
  const [sending, setSending] = useState(false)
  const [notice, setNotice] = useState('')

  const activeIdRef = useRef('')
  activeIdRef.current = activeId
  const storiesRef = useRef<SinStoryView[]>([])
  storiesRef.current = stories
  // 流收尾句柄（done/error/超时/卸载四路径收敛，sending 必复位）
  const finishRef = useRef<((ok: boolean) => void) | null>(null)
  const cleanupRef = useRef<(() => void) | null>(null)
  // 在流的助手消息 key（取消时把「正在写」标记收掉）
  const streamingKeyRef = useRef('')
  // v4.354：消息载入序号——快速切故事时旧故事 SinMessages 慢响应会覆盖当前
  // 视图（同 useChatTopics.loadTopic 守卫范式），此后发送会把两故事内容同屏串台
  const loadSeqRef = useRef(0)

  const clearNotice = useCallback(() => setNotice(''), [])
  const showNotice = useCallback((message: string) => setNotice(message), [])

  const loadMessages = useCallback(async (id: string) => {
    const seq = ++loadSeqRef.current
    if (!id) {
      setMessages([])
      return
    }
    try {
      const ms = await app.SinMessages(id)
      if (seq !== loadSeqRef.current) return
      setMessages((ms || []).map((m) => toMessageView(m as unknown as Record<string, unknown>)))
    } catch (err) {
      if (seq !== loadSeqRef.current) return
      setNotice(errText(err, '故事内容读取失败'))
      setMessages([])
    }
  }, [])

  const refreshStories = useCallback(async (): Promise<SinStoryView[]> => {
    try {
      const list = await app.SinTopicsList()
      return (list || []).map((t) => toStoryView(t as unknown as Record<string, unknown>))
    } catch (err) {
      setNotice(errText(err, '故事列表读取失败'))
      return []
    }
  }, [])

  // ── 初始化：故事列表 + 自动选中（无故事则建一个空故事） ──
  useEffect(() => {
    let live = true
    ;(async () => {
      let list = await refreshStories()
      if (!live) return
      if (list.length === 0) {
        try {
          await app.SinTopicCreate('新故事')
          list = await refreshStories()
        } catch (err) {
          if (live) setNotice(errText(err, '新建故事失败'))
        }
        if (!live) return
      }
      setStories(list)
      const first = list[0]
      if (first) {
        setActiveId(first.id)
        await loadMessages(first.id)
      }
      if (live) setInitializing(false)
    })()
    return () => {
      live = false
      finishRef.current?.(false)
      finishRef.current = null
      cleanupRef.current?.()
      cleanupRef.current = null
    }
    // 初始化只跑一次（故事列表后续由各操作自己刷新）
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const selectStory = useCallback(async (id: string) => {
    if (!id || id === activeIdRef.current) return
    // 切故事 = 中止在途流（其消息已不在视图内，继续写只会串台）
    finishRef.current?.(false)
    finishRef.current = null
    cleanupRef.current?.()
    cleanupRef.current = null
    setSending(false)
    setActiveId(id)
    activeIdRef.current = id
    await loadMessages(id)
  }, [loadMessages])

  const createStory = useCallback(async () => {
    try {
      loadSeqRef.current++ // 失效在途故事消息载入（v4.354）
      const t = await app.SinTopicCreate('新故事')
      const list = await refreshStories()
      setStories(list)
      const id = typeof (t as { id?: unknown })?.id === 'string' ? String((t as { id: string }).id) : list[0]?.id ?? ''
      if (id) {
        setActiveId(id)
        activeIdRef.current = id
        setMessages([])
      }
    } catch (err) {
      setNotice(errText(err, '新建故事失败'))
    }
  }, [refreshStories])

  const renameStory = useCallback(async (id: string, title: string) => {
    try {
      await app.SinTopicRename(id, title)
      setStories(await refreshStories())
    } catch (err) {
      setNotice(errText(err, '重命名失败'))
    }
  }, [refreshStories])

  const deleteStory = useCallback(async (id: string) => {
    try {
      await app.SinTopicDelete(id)
      const list = await refreshStories()
      setStories(list)
      if (id === activeIdRef.current) {
        const next = list[0]
        setActiveId(next?.id ?? '')
        activeIdRef.current = next?.id ?? ''
        await loadMessages(next?.id ?? '')
      }
    } catch (err) {
      setNotice(errText(err, '删除失败'))
    }
  }, [loadMessages, refreshStories])

  const clearStory = useCallback(async (id: string) => {
    try {
      await app.SinTopicClear(id)
      if (id === activeIdRef.current) setMessages([])
    } catch (err) {
      setNotice(errText(err, '清空失败'))
    }
  }, [])

  const setIllustration = useCallback((messageKey: string, cueKey: string, path: string) => {
    setMessages((prev) => prev.map((m) => (
      m.key === messageKey ? { ...m, illustrations: { ...m.illustrations, [cueKey]: path } } : m
    )))
  }, [])

  const reloadMessages = useCallback(async () => {
    await loadMessages(activeIdRef.current)
  }, [loadMessages])

  const send = useCallback(async (text: string) => {
    const content = (text ?? '').trim()
    let storyId = activeIdRef.current
    if (!content || sending) return
    // 没有故事时先建一个（首次进来直接开写不该被拦住）
    if (!storyId) {
      try {
        const t = await app.SinTopicCreate('新故事')
        storyId = typeof (t as { id?: unknown })?.id === 'string' ? String((t as { id: string }).id) : ''
        if (!storyId) throw new Error('新建故事失败')
        setStories(await refreshStories())
        setActiveId(storyId)
        activeIdRef.current = storyId
      } catch (err) {
        setNotice(errText(err, '新建故事失败'))
        return
      }
    }

    // 首轮用户消息即故事标题建议（仅当仍是默认名）
    const story = storiesRef.current.find((s) => s.id === storyId)
    if (story && (story.title === '新故事' || story.title.trim() === '')) {
      const title = suggestStoryTitle(content)
      // v4.362：自动命名失败可见化（原故事名停留「新故事」无原因）。
      void app.SinTopicRename(storyId, title).then(() => refreshStories().then(setStories)).catch((err: unknown) => {
        setNotice(`自动命名失败：${err instanceof Error ? err.message : String(err)}`)
      })
    }

    const userKey = nextKey('sin_user')
    const asstKey = nextKey('sin_asst')
    streamingKeyRef.current = asstKey
    setMessages((prev) => [
      ...prev,
      { key: userKey, messageId: 0, role: 'user', content, illustrations: {}, createdAt: nowStr() },
      { key: asstKey, messageId: 0, role: 'assistant', content: '', streaming: true, illustrations: {}, createdAt: nowStr() },
    ])
    setSending(true)
    setNotice('')

    const patch = (key: string, p: Partial<SinMessageView>) => {
      setMessages((prev) => prev.map((m) => (m.key === key ? { ...m, ...p } : m)))
    }

    // 过程帧折叠（v4.262）：dispatch 先到追加运行中行、result 后到按 id 合并终态；
    // 丢帧导致 result 先到时如实补一行（不静默吞掉模型确实调用过的事实）。
    const applyToolFrame = (p: SinStreamPayload) => {
      setMessages((prev) => prev.map((m) => {
        if (m.key !== asstKey) return m
        const tools = m.tools ?? []
        if (p.type === 'tool_dispatch') {
          const view: SinToolTraceView = {
            id: p.id ?? '', name: p.name ?? '', args: p.args ?? '',
            read_only: p.read_only === true, status: 'running',
          }
          return { ...m, tools: [...tools, view] }
        }
        const idx = findToolRow(tools, p)
        const done: SinToolTraceView = idx >= 0
          ? {
            ...tools[idx],
            output: p.output ?? tools[idx].output,
            error: p.error ?? '',
            elapsed_ms: typeof p.elapsed_ms === 'number' ? p.elapsed_ms : tools[idx].elapsed_ms,
            status: p.error ? 'failed' : 'done',
          }
          : {
            id: p.id ?? '', name: p.name ?? '', args: '',
            output: p.output ?? '', error: p.error ?? '',
            elapsed_ms: typeof p.elapsed_ms === 'number' ? p.elapsed_ms : 0,
            status: p.error ? 'failed' : 'done',
          }
        const next = tools.slice()
        if (idx >= 0) next[idx] = done
        else next.push(done)
        return { ...m, tools: next }
      }))
    }

    await new Promise<void>((resolve) => {
      let settled = false
      // v4.262：静默超时改为「每一帧重置」——工具循环下这一轮要跑多次模型调用
      // （查询→抓取→收尾写作），一次性 90 秒会把正常的长回合误判成超时
      // （真机走查：正文已落库 827 字，前端却报了「请求超时」）。
      let timer: ReturnType<typeof setTimeout> | undefined
      const armSilenceTimer = () => {
        if (timer) clearTimeout(timer)
        timer = setTimeout(() => {
          patch(asstKey, {
            streaming: false,
            error: true,
            content: `请求超时：${SIN_STREAM_SILENCE_TIMEOUT_MS / 1000} 秒内未收到回复，请重试`,
          })
          finish()
        }, SIN_STREAM_SILENCE_TIMEOUT_MS)
      }
      const finish = () => {
        if (settled) return
        settled = true
        if (timer) clearTimeout(timer)
        if (finishRef.current === finish) finishRef.current = null
        resolve()
      }
      finishRef.current = finish
      armSilenceTimer()

      app.SinStream(storyId, content)
        .then((runID: string) => {
          if (!runID) {
            patch(asstKey, { streaming: false, error: true, content: '请求失败：未取得流通道' })
            finish()
            return
          }
          // 先订阅后收帧：runID 一到立即注册（与后端 emit 无异步间隙）
          const unsub = subscribe(sinStreamChannel(runID), (raw) => {
            if (settled) return
            armSilenceTimer() // 收到任何一帧都算「还活着」
            const p = (raw || {}) as SinStreamPayload
            if (p.type === 'delta') {
              setMessages((prev) => prev.map((m) => (
                m.key === asstKey ? { ...m, content: m.content + (p.content || '') } : m
              )))
              return
            }
            if (p.type === 'reasoning') {
              setMessages((prev) => prev.map((m) => (
                m.key === asstKey ? { ...m, reasoning: (m.reasoning ?? '') + (p.content || '') } : m
              )))
              return
            }
            if (p.type === 'tool_dispatch' || p.type === 'tool_result') {
              applyToolFrame(p)
              return
            }
            if (p.type === 'notice') {
              // 降级提示（如「当前模型不支持工具调用，已按纯写作继续」）：如实弹出，
              // 不写进故事正文。
              setNotice(p.message || '')
              return
            }
            if (p.type === 'done') {
              const reply = typeof p.reply === 'string' ? p.reply : ''
              const messageId = typeof p.message_id === 'number' ? p.message_id : 0
              const doneTools = Array.isArray(p.tools) && p.tools.length > 0 ? toToolViews(p.tools) : undefined
              patch(asstKey, {
                // 落库后的消息 id 是插图回写的定位键，命中后 key 与 db 行对齐
                key: messageId > 0 ? `db_${messageId}` : asstKey,
                messageId,
                content: reply,
                streaming: false,
                reasoning: typeof p.reasoning === 'string' ? p.reasoning : '',
                ...(doneTools ? { tools: doneTools } : {}),
              })
              finish()
              return
            }
            if (p.type === 'error') {
              patch(asstKey, {
                streaming: false,
                error: true,
                content: `请求失败：${p.error || '未知错误'}`,
              })
              finish()
            }
          })
          cleanupRef.current = () => {
            unsub()
            finish()
          }
        })
        .catch((err: unknown) => {
          patch(asstKey, { streaming: false, error: true, content: `请求失败：${errText(err, '未知错误')}` })
          finish()
        })
    })

    cleanupRef.current?.()
    cleanupRef.current = null
    // 无论是否卸载/取消，都必须收尾（v4.258 修复：历史实现用 deadRef 跳过收尾，
    // 在 StrictMode 双挂载/空间剪枝重挂后 sending 会永久卡 true，输入框被锁死）。
    setSending(false)
    streamingKeyRef.current = ''
    setStories(await refreshStories())
  }, [refreshStories, sending])

  // 停止：先让后端中止（保留已生成部分落库），再本地立刻解封输入
  const cancel = useCallback((): string | undefined => {
    const topicID = activeIdRef.current
    if (topicID) {
      app.SinCancel(topicID).catch(() => { /* 无在途流/取消失败：本地照常收尾 */ })
    }
    const key = streamingKeyRef.current
    if (key) {
      setMessages((prev) => prev.map((m) => (
        m.key === key
          ? { ...m, streaming: false, content: m.content || '（已停止）' }
          : m
      )))
    }
    finishRef.current?.(false)
    finishRef.current = null
    cleanupRef.current?.()
    cleanupRef.current = null
    streamingKeyRef.current = ''
    setSending(false)
    return undefined
  }, [])

  const activeStory = stories.find((s) => s.id === activeId)

  return {
    stories, activeId, activeStory, messages, initializing, sending, notice, clearNotice, showNotice,
    selectStory, createStory, renameStory, deleteStory, clearStory, send, setIllustration,
    reloadMessages,
    cancel,
  }
}
