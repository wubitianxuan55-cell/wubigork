// ChatPage 拆分产物：plain/persona 统一真实流式发送状态机。
// 原 ChatPage.doSend 的 plain 流订阅（T6-3.1：先订阅后收帧、30s 无帧超时、
// done/error/启动失败终态、卸载收尾）继续保留；角色模式已改为同一事件协议，
// 首个正文分片立即上屏，切话题/卸载直接取消后端 run。
import { useCallback, useEffect, useRef, useState } from 'react'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { app } from '../gaea/lib/bridge'
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
  /** 角色流完成帧附带的会话状态（plain 流不携带） */
  emotion?: string
  aff?: number
  aro?: number
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
  // 旧接口保留的取消标志（外部切话题/卸载调用取消后端流，新一轮发送复位）。
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
    typingCancelRef.current = true
    const rid = activeRunIDRef.current
    activeRunIDRef.current = null
    if (rid) void app.ChatStreamCancel(rid).catch(() => {})
  }, [])

  // P1-5：进行中流式回合的 runID（停止入口用；无流/角色模式为 null）。
  const activeRunIDRef = useRef<string | null>(null)
  const cancelTyping = useCallback(() => {
    typingCancelRef.current = true
    const rid = activeRunIDRef.current
    if (rid) {
      activeRunIDRef.current = null
      void app.ChatStreamCancel(rid).catch(() => {})
    }
  }, [])
  const resetTyping = useCallback(() => { typingCancelRef.current = false }, [])

  const send = useCallback(async (params: ChatSendParams) => {
    const { text, mode, active, search, thinking, force, retryKey } = params
    const trimmed = text.trim()
    if (!trimmed || sending || !active) return
    // T6-3.3：新一轮发送复位模拟打字取消标志（旧循环已被切话题/卸载置 true）
    typingCancelRef.current = false
    activeRunIDRef.current = null
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

    // 普通对话与角色对话统一走真实流：角色链路此前先等整段返回，再做前端模拟
    // 打字，长回复会额外增加数十秒可见延迟；现在首个模型分片即上屏。
    {
      let unsub: (() => void) | null = null
      let reasoningAcc = ''
      // v4.365 流式节流：每 chunk 一次 setState 使 ChatPage 以 chunk 频率（30~120/s）
      // 全页重渲染+每帧强制 reflow（吸底）。改为 buffer + rAF flush——一帧最多一刷，
      // 最终文本一致（done/error/超时路径 flush 前先取消挂起帧）。
      let deltaBuf = ''
      let deltaRaf = 0
      const cancelPendingDelta = () => {
        if (deltaRaf) { cancelAnimationFrame(deltaRaf); deltaRaf = 0 }
        deltaBuf = ''
      }
      const flushDelta = () => {
        deltaRaf = 0
        if (!deltaBuf) return
        const chunk = deltaBuf
        deltaBuf = ''
        setStreamText(prev => prev + chunk)
      }
      const cleanup = () => { cancelPendingDelta(); if (unsub) { const f = unsub; unsub = null; f() } }
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
            cancelPendingDelta()
            setStreamText(''); setStreamKey(null)
            updateMessage(am.key, { content: `请求超时：${STREAM_SILENCE_TIMEOUT_MS / 1000} 秒内未收到回复，请重试`, streaming: false, error: true })
            finish(false)
          }, STREAM_SILENCE_TIMEOUT_MS)
          const opening = mode === 'plain'
            ? app.ChatStreamPlain(active, trimmed, search, thinking, force)
            : app.ChatStreamPersona(active, trimmed, mode, search, thinking, force)
          opening
            .then((runID: string) => {
              activeRunIDRef.current = runID
              // 超时/卸载先于 binding resolve：runID 现在才到，直接撤销后端任务，
              // 不再为已结束的前端回合挂监听。
              if (settled) {
                activeRunIDRef.current = null
                void app.ChatStreamCancel(runID).catch(() => {})
                return
              }
              // 订阅注册紧跟 runID 解析：同一微任务内完成，先订阅后收帧，首帧不丢
              unsub = EventsOn(`chat-stream:${runID}`, (payload: ChatStreamPayload) => {
                if (settled) return
                const p = payload || {}
                if (p.type === 'delta') {
                  deltaBuf += (p.content || '')
                  if (!deltaRaf) deltaRaf = requestAnimationFrame(flushDelta)
                } else if (p.type === 'reasoning') {
                  reasoningAcc += p.content || ''
                } else if (p.type === 'done') {
                  const reply = typeof p.reply === 'string' ? p.reply : ''
                  const reasoning = typeof p.reasoning === 'string' ? p.reasoning : reasoningAcc
                  // answered_by 与角色状态随 done 帧下发；统一写入 extra，
                  // 刷新会话后指标仍可从消息记录恢复。
                  const extra: Record<string, unknown> = {}
                  const ab = p.answered_by
                  if (ab && typeof ab === 'object' && typeof ab.engine === 'string' && typeof ab.model === 'string') {
                    extra.answered_by = ab
                  }
                  if (typeof p.emotion === 'string' && p.emotion) extra.emotion = p.emotion
                  if (typeof p.reasoning === 'string' && p.reasoning) extra.reasoning = p.reasoning
                  cancelPendingDelta()
                  setStreamText(''); setStreamKey(null)
                  updateMessage(am.key, { content: reply, streaming: false, reasoning, extra })
                  if (typeof p.emotion === 'string' && p.emotion) setEmotion(p.emotion)
                  if (typeof p.aff === 'number') setAff(Math.round(p.aff))
                  if (typeof p.aro === 'number') setAro(Math.round(p.aro))
                  finish(true)
                } else if (p.type === 'cancelled') {
                  // P1-5 用户停止：已生成部分落库（后端已 appendChatExchange），
                  // 前端如实展示部分回复 + 停止标注（不按失败处理）。
                  cancelPendingDelta()
                  const partial = typeof p.reply === 'string' ? p.reply : ''
                  setStreamText(''); setStreamKey(null)
                  const extra: Record<string, unknown> = {}
                  if (p.answered_by && typeof p.answered_by === 'object') extra.answered_by = p.answered_by
                  updateMessage(am.key, {
                    content: partial ? partial + '\n\n（已停止生成）' : '（已停止生成，未收到内容）',
                    streaming: false, reasoning: p.reasoning || reasoningAcc, extra,
                  })
                  finish(false)
                } else if (p.type === 'error') {
                  cancelPendingDelta()
                  setStreamText(''); setStreamKey(null)
                  updateMessage(am.key, { content: `请求失败：${p.error || '未知错误'}`, streaming: false, error: true })
                  finish(false)
                }
              })
              // 用户在 runID 返回前已点停止：订阅建立后立即撤销后端任务，
              // cancelled 帧仍会走统一收尾。
              if (typingCancelRef.current) {
                void app.ChatStreamCancel(runID).catch(() => {})
              }
            })
            .catch((err: unknown) => {
              // 启动失败（binding 拒绝）：直接按失败收尾，与事件错误终态一致
              cancelPendingDelta()
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
        activeRunIDRef.current = null
        if (streamCleanupRef.current === cleanup) streamCleanupRef.current = null
        setSending(false)
      }
      return
    }
  }, [sending, updateMessage, setMessages, setInput, setEmotion, setAff, setAro, finalizeTopicAfterSend])

  /** P1-5 停止生成：plain/persona 流均调后端取消，cancelled 帧收尾并保留部分回复。 */
  const stop = useCallback(() => {
    if (typingCancelRef.current !== null && !sending) return
    cancelTyping()
  }, [sending, cancelTyping])

  return { sending, streamKey, streamText, send, stop, cancelTyping, resetTyping }
}
