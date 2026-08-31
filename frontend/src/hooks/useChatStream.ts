// ChatPage 拆分产物：流式/模拟打字发送状态机（行为零变化，T6-10.1）。
// 原 ChatPage.doSend 的 plain 流订阅（T6-3.1：先订阅后收帧、30s 无帧超时、
// done/error/启动失败终态、卸载收尾）+ 角色模式模拟打字（T6-3.3：切话题/卸载
// 取消循环）原样搬入；外部仅注入消息更新回调与最终态回调。
import { useCallback, useEffect, useRef, useState } from 'react'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import * as App from '../wailsjsCompat'
import { STREAM_SILENCE_TIMEOUT_MS } from '../pages/chat/constants'
import { nextMsgKey, nowStr } from '../pages/chat/utils'
import type { ChatMsg } from '../pages/chat/types'
import type { AnsweredByInfo } from '../components/chat/AnsweredByLine'

/** chat-stream:<runID> 事件的动态载荷（最小消费面） */
interface ChatStreamPayload {
  type?: string
  content?: string
  reply?: string
  reasoning?: string
  error?: string
  /** v4.15 可选：实际回答的引擎/模型/来源/费用；旧事件无此字段 → 静默跳过 */
  answered_by?: AnsweredByInfo
}

export interface UseChatStreamOptions {
  updateMessage: (key: string, patch: Partial<ChatMsg>) => void
  setMessages: React.Dispatch<React.SetStateAction<ChatMsg[]>>
  setInput: React.Dispatch<React.SetStateAction<string>>
  setEmotion: (v: string) => void
  setAff: (v: number) => void
  setAro: (v: number) => void
  finalizeTopicAfterSend: (topicId: string, firstUserText: string) => Promise<void>
}

export interface ChatSendParams {
  text: string
  mode: string
  active: string
  search: boolean
  thinking: boolean
  force: boolean
  retryKey?: string
}

export function useChatStream(opts: UseChatStreamOptions) {
  const { updateMessage, setMessages, setInput, setEmotion, setAff, setAro, finalizeTopicAfterSend } = opts
  const [sending, setSending] = useState(false)
  const [streamKey, setStreamKey] = useState<string | null>(null)
  const [streamText, setStreamText] = useState('')
  // T6-3.3：模拟打字循环取消标志（切话题/卸载即置 true 中止循环，新一轮发送复位）。
  const typingCancelRef = useRef(false)
  // 真实流式订阅的清理函数：卸载时取消监听，避免泄漏。
  const streamCleanupRef = useRef<(() => void) | null>(null)
  // T6-3.1：当前流式对话的终态回调（done/error/超时/卸载时调用一次，
  // 保证挂起的流 Promise 必然收尾、finally 必执行、sending 四路径均可复位）。
  const streamFinishRef = useRef<((ok: boolean) => void) | null>(null)

  // 卸载收尾：取消流式订阅 + 收尾挂起的流 Promise（sending 复位路径完整）、
  // 中止模拟打字循环。
  useEffect(() => () => {
    // T6-3.1：卸载时触发流终态回调——挂起的流 Promise 得以 resolve，
    // finally 必执行（组件已卸载，setState 无副作用），不留悬挂计时器/监听。
    streamFinishRef.current?.(false)
    streamFinishRef.current = null
    streamCleanupRef.current?.()
    streamCleanupRef.current = null
    // T6-3.3：卸载中止模拟打字循环
    typingCancelRef.current = true
  }, [])

  const cancelTyping = useCallback(() => { typingCancelRef.current = true }, [])
  const resetTyping = useCallback(() => { typingCancelRef.current = false }, [])

  const send = useCallback(async (params: ChatSendParams) => {
    const { text, mode, active, search, thinking, force, retryKey } = params
    const trimmed = text.trim()
    if (!trimmed || sending || !active) return
    // T6-3.3：新一轮发送复位模拟打字取消标志（旧循环已被切话题/卸载置 true）
    typingCancelRef.current = false
    setInput(''); setSending(true)
    const um: ChatMsg = { key: nextMsgKey(), role: 'user', content: trimmed, createdAt: nowStr() }
    const am: ChatMsg = { key: nextMsgKey(), role: 'assistant', content: '', streaming: true, createdAt: nowStr() }
    setMessages(prev => {
      if (!retryKey) return [...prev, um, am]
      // 重试：失败消息未落库，其前置用户消息同样未落库，一并移除保持与 DB 一致
      const errIdx = prev.findIndex(m => m.key === retryKey)
      const drop = new Set<string>([retryKey])
      if (errIdx >= 0) {
        const userMsg = prev.slice(0, errIdx).reverse().find(m => m.role === 'user')
        if (userMsg) drop.add(userMsg.key)
      }
      return [...prev.filter(m => !drop.has(m.key)), um, am]
    })
    setStreamKey(am.key); setStreamText('')

    // 普通对话：后端逐块流式下发；角色模式继续走整段返回（保留原有模拟打字流）。
    if (mode === 'plain') {
      let unsub: (() => void) | null = null
      let reasoningAcc = ''
      const cleanup = () => { if (unsub) { const f = unsub; unsub = null; f() } }
      streamCleanupRef.current = cleanup
      try {
        // T6-3.1：流 Promise 自带「无帧超时 + 终态兜底」。runID 一到立即在
        // 同一同步块注册精确频道监听（与 binding 解析零异步间隙，先订阅后收帧）；
        // 首帧若在注册前发出（后端 goroutine 竞态——Wails JS 事件按事件名精确
        // 匹配，runID 未知时无法提前订阅），由超时兜底：30s 无任何帧按失败展示，
        // sending 必复位、finally 必执行。
        const ok = await new Promise<boolean>((resolve) => {
          let settled = false
          const finish = (okVal: boolean) => {
            if (settled) return
            settled = true
            clearTimeout(timer)
            if (streamFinishRef.current === finish) streamFinishRef.current = null
            resolve(okVal)
          }
          streamFinishRef.current = finish
          // finish 仅在 timer 初始化之后被调用（事件/超时回调），闭包引用无 TDZ 问题
          const timer = setTimeout(() => {
            setStreamText(''); setStreamKey(null)
            updateMessage(am.key, { content: `请求超时：${STREAM_SILENCE_TIMEOUT_MS / 1000} 秒内未收到回复，请重试`, streaming: false, error: true })
            finish(false)
          }, STREAM_SILENCE_TIMEOUT_MS)
          App.ChatStreamPlain(active, trimmed, search, thinking, force)
            .then((runID: string) => {
              // 订阅注册紧跟 runID 解析：同一微任务内完成，先订阅后收帧，首帧不丢
              unsub = EventsOn(`chat-stream:${runID}`, (payload: ChatStreamPayload) => {
                if (settled) return
                const p = payload || {}
                if (p.type === 'delta') {
                  setStreamText(prev => prev + (p.content || ''))
                } else if (p.type === 'reasoning') {
                  reasoningAcc += p.content || ''
                } else if (p.type === 'done') {
                  const reply = typeof p.reply === 'string' ? p.reply : ''
                  const reasoning = typeof p.reasoning === 'string' ? p.reasoning : reasoningAcc
                  // v4.15：answered_by 为可选字段（旧事件缺失 → 静默跳过，不写入 extra）
                  const extra: Record<string, unknown> = {}
                  const ab = p.answered_by
                  if (ab && typeof ab === 'object' && typeof ab.engine === 'string' && typeof ab.model === 'string') {
                    extra.answered_by = ab
                  }
                  setStreamText(''); setStreamKey(null)
                  updateMessage(am.key, { content: reply, streaming: false, reasoning, extra })
                  finish(true)
                } else if (p.type === 'error') {
                  setStreamText(''); setStreamKey(null)
                  updateMessage(am.key, { content: `请求失败：${p.error || '未知错误'}`, streaming: false, error: true })
                  finish(false)
                }
              })
            })
            .catch((err: unknown) => {
              // 启动失败（binding 拒绝）：直接按失败收尾，与事件错误终态一致
              setStreamText(''); setStreamKey(null)
              updateMessage(am.key, { content: `请求失败：${err instanceof Error ? err.message : String(err)}`, streaming: false, error: true })
              finish(false)
            })
        })
        if (ok) await finalizeTopicAfterSend(active, trimmed)
      } catch (err: unknown) {
        setStreamText(''); setStreamKey(null)
        updateMessage(am.key, { content: `请求失败：${err instanceof Error ? err.message : String(err)}`, streaming: false, error: true })
      } finally {
        cleanup()
        if (streamCleanupRef.current === cleanup) streamCleanupRef.current = null
        setSending(false)
      }
      return
    }

    // 角色模式：整段返回 + 前端模拟打字流。
    try {
      const res = await App.ChatSend(active, trimmed, mode, search, thinking, force)
      const reply = typeof res?.reply === 'string' ? res.reply : ''
      const reasoning = typeof res?.reasoning === 'string' ? res.reasoning : ''
      const reduced = typeof window !== 'undefined' && !!window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
      if (!reduced && reply.length > 40) {
        const step = Math.max(2, Math.round(reply.length / 180))
        for (let i = 0; i <= reply.length; i += step) {
          // T6-3.3：切话题/卸载即中止模拟打字流（避免过期 setStreamText 持续写入）
          if (typingCancelRef.current) break
          setStreamText(reply.slice(0, i))
          await new Promise(r => setTimeout(r, 14))
        }
      }
      // T6-3.3：循环被取消（话题已切换/组件已卸载）→ 不再更新消息与最终态，
      // 由 finally 复位 sending；旧消息 key 已随话题切换离开消息列表，更新无意义。
      if (typingCancelRef.current) return
      setStreamText(''); setStreamKey(null)
      const extra: Record<string, unknown> = {}
      if (res.emotion) extra.emotion = res.emotion
      if (reasoning) extra.reasoning = reasoning
      // v4.15：ChatSend 返回可选字段 answered_by（旧后端无此字段 → 静默跳过）
      const ab = res.answered_by
      if (ab && typeof ab === 'object' && typeof ab.engine === 'string' && typeof ab.model === 'string') {
        extra.answered_by = ab
      }
      updateMessage(am.key, { content: reply, streaming: false, reasoning, extra })
      if (res.emotion) setEmotion(res.emotion)
      if (typeof res.aff === 'number') setAff(Math.round(res.aff))
      if (typeof res.aro === 'number') setAro(Math.round(res.aro))
      await finalizeTopicAfterSend(active, trimmed)
    } catch (err: unknown) {
      setStreamText(''); setStreamKey(null)
      updateMessage(am.key, { content: `请求失败：${err instanceof Error ? err.message : String(err)}`, streaming: false, error: true })
    } finally { setSending(false) }
  }, [sending, updateMessage, setMessages, setInput, setEmotion, setAff, setAro, finalizeTopicAfterSend])

  return { sending, streamKey, streamText, send, cancelTyping, resetTyping }
}
