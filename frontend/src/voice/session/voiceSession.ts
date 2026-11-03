/**
 * voice/session/voiceSession.ts — Voice V2 会话状态机（M2，纯函数零 DOM）。
 *
 * UI 状态由协议事件直接驱动（V2 方案 §10.3），禁止用音量阈值猜 AI 是否在说话。
 * 相位迁移以 M1 实测事件序为准：
 *   speech_started → LISTENING
 *   speech_stopped → LISTENING（软停，等提交）
 *   transcription.completed → 轮次提交，转写定稿
 *   response.created → THINKING
 *   首个 output_audio.delta → SPEAKING
 *   response.done(completed) → IDLE
 *   response.done(cancelled/turn_detected) → INTERRUPTED（随后 speech_started → LISTENING）
 *   response.done(cancelled/client_cancelled) → IDLE（客户端主动停）
 */

import { isKnownServerEvent, responseIdOf, responseStatusOf, type ServerEvent } from '../protocol/events'

export type VoicePhase = 'idle' | 'listening' | 'thinking' | 'speaking' | 'interrupted' | 'error'

export interface ResponseLedger {
  id: string
  status: string
  reason: string
  /** 本轮输入协议 item_id（sidecar 请求头与提交回执共用，重复话术仍可区分） */
  userItemId: string
  /** 本轮定稿用户转写（response.created 时从未配对队列取首项） */
  userTranscript: string
  /** 本轮助手文本（终稿 transcript.done 权威替换） */
  assistantText: string
  /** transcript delta 累积字符数 */
  textChars: number
  /** 下行 PCM 字节数（base64 解码后） */
  audioBytes: number
  /** 终态后仍到达的旧代际字节数（泄漏计数） */
  staleBytesAfterDone: number
  closed: boolean
}

export interface VoiceSessionState {
  phase: VoicePhase
  sessionId: string
  /** 活动输入项的增量转写（delta 累积，可被 completed 替换） */
  partialTranscript: string
  /** 每轮用户转写定稿（latest 在末尾） */
  committedTranscripts: string[]
  /**
   * 尚未与 response 配对的用户转写。旧轮 response.done 迟到时仍从自己的
   * ledger 取用户文本，不再用「全局最后一句」猜配对。
   */
  pendingUserTranscripts: PendingUserTranscript[]
  /** 活动 response 的助手文本（transcript delta 累积） */
  assistantText: string
  responses: Record<string, ResponseLedger>
  responseOrder: string[]
  activeResponseId: string
  speechStartCount: number
  speechStopCount: number
  unknownEventCount: number
  errors: string[]
}

export interface PendingUserTranscript {
  text: string
  itemId: string
}

export function initialVoiceSession(): VoiceSessionState {
  return {
    phase: 'idle',
    sessionId: '',
    partialTranscript: '',
    committedTranscripts: [],
    pendingUserTranscripts: [],
    assistantText: '',
    responses: {},
    responseOrder: [],
    activeResponseId: '',
    speechStartCount: 0,
    speechStopCount: 0,
    unknownEventCount: 0,
    errors: [],
  }
}

function ledger(state: VoiceSessionState, id: string): ResponseLedger {
  let led = state.responses[id]
  if (!led) {
    led = {
      id, status: '', reason: '', userItemId: '', userTranscript: '', assistantText: '',
      textChars: 0, audioBytes: 0, staleBytesAfterDone: 0, closed: false,
    }
    state.responses[id] = led
    state.responseOrder.push(id)
  }
  return led
}

/** 单事件归约（返回新状态；实现上做浅拷贝顶层 + 就地更新 ledger 副本） */
export function reduceVoiceSession(state: VoiceSessionState, ev: ServerEvent, audioDeltaBytes = 0): VoiceSessionState {
  const next: VoiceSessionState = { ...state, responses: { ...state.responses } }

  if (!isKnownServerEvent(ev)) {
    next.unknownEventCount += 1
    return next
  }

  switch (ev.type) {
    case 'session.created':
    case 'session.updated': {
      const sid = typeof ev.session?.id === 'string' ? ev.session.id : ''
      if (sid) next.sessionId = sid
      return next
    }
    case 'input_audio_buffer.speech_started':
      next.speechStartCount += 1
      next.phase = 'listening'
      return next
    case 'input_audio_buffer.speech_stopped':
      next.speechStopCount += 1
      // 软停不迁相位：等转写定稿/提交（M1 实测该事件在提交期才上 wire）
      return next
    case 'conversation.item.input_audio_transcription.delta':
      next.partialTranscript += typeof ev.delta === 'string' ? ev.delta : ''
      return next
    case 'conversation.item.input_audio_transcription.completed': {
      const transcript = typeof ev.transcript === 'string' ? ev.transcript : ''
      next.committedTranscripts = [...next.committedTranscripts, transcript]
      next.pendingUserTranscripts = [...next.pendingUserTranscripts, {
        text: transcript,
        itemId: typeof ev.item_id === 'string' ? ev.item_id : '',
      }]
      next.partialTranscript = ''
      return next
    }
    case 'response.created': {
      const id = responseIdOf(ev)
      const led = ledger(next, id)
      led.status = 'in_progress'
      const pending = next.pendingUserTranscripts[0]
      led.userTranscript = pending?.text ?? ''
      led.userItemId = pending?.itemId ?? ''
      if (next.pendingUserTranscripts.length > 0) {
        next.pendingUserTranscripts = next.pendingUserTranscripts.slice(1)
      }
      next.activeResponseId = id
      next.assistantText = ''
      next.phase = 'thinking'
      return next
    }
    case 'response.output_audio_transcript.delta': {
      const id = responseIdOf(ev)
      const led = ledger(next, id)
      if (typeof ev.delta === 'string') {
        led.textChars += ev.delta.length
        if (!led.closed) {
          led.assistantText += ev.delta
          if (id === next.activeResponseId) next.assistantText += ev.delta
        }
      }
      return next
    }
    case 'response.output_audio_transcript.done': {
      const id = responseIdOf(ev)
      const led = ledger(next, id)
      if (typeof ev.transcript === 'string') led.assistantText = ev.transcript
      if (id === next.activeResponseId && typeof ev.transcript === 'string') {
        // 终稿权威替换累积（协议语义：done.transcript 为全量）
        next.assistantText = ev.transcript
      }
      return next
    }
    case 'response.output_audio.delta': {
      const id = responseIdOf(ev)
      const led = ledger(next, id)
      led.audioBytes += audioDeltaBytes
      if (!led.closed && id === next.activeResponseId) {
        next.phase = 'speaking'
      } else if (led.closed) {
        // 终态后的迟到帧：不可能播，只记账（M1 断言恒为 0）
        led.staleBytesAfterDone += audioDeltaBytes
      }
      return next
    }
    case 'response.output_audio.done':
      return next
    case 'response.done': {
      const id = responseIdOf(ev)
      const led = ledger(next, id)
      // 重复 done 不得再次触发写回；事件计数仍由原始流保留。
      if (led.closed) return next
      const { status, reason } = responseStatusOf(ev)
      led.status = status
      led.reason = reason
      led.closed = true
      if (id === next.activeResponseId) {
        if (status === 'cancelled' && reason === 'turn_detected') {
          next.phase = 'interrupted' // 随后 speech_started 会接管 → listening
        } else if (status === 'completed') {
          next.phase = 'idle'
        } else {
          // cancelled/client_cancelled、incomplete、failed：回空闲，错误态留给 error 事件
          next.phase = status === 'failed' ? 'error' : 'idle'
          if (status === 'failed') next.errors = [...next.errors, `response ${id} failed${reason ? ` (${reason})` : ''}`]
        }
      }
      return next
    }
    case 'error': {
      const msg = typeof ev.error?.message === 'string' ? ev.error.message : JSON.stringify(ev.error)
      next.errors = [...next.errors, msg]
      return next
    }
    default:
      return next
  }
}

// ───────────────────── 事件序不变式（无模型回放断言面） ─────────────────────

/**
 * 对一段服务端事件流做有序性检查，返回违规清单（空 = 有序）。
 * 这是 M2「不需要真实模型即可断言事件有序性」的兑现面：
 * - session.created 若存在必须居首；
 * - speech_started 必须先于对应 speech_stopped（计数配对）；
 * - response X 的任何 delta/done 不得早于 response.created(X)；
 * - response.done(X) 之后的 X 帧视为迟到泄漏（M1 门禁要求 0）。
 */
export function orderingViolations(events: ServerEvent[]): string[] {
  const bad: string[] = []
  const seenCreated = new Set<string>()
  const closedAt = new Map<string, number>()
  let speechOpen = false
  let firstRealEvent = true

  events.forEach((ev, i) => {
    const label = `#${i} ${ev.type}`
    if (firstRealEvent && ev.type !== 'session.created' && ev.type !== 'session.updated') {
      // session.created 缺席不算违规（允许直接回放中段），但若存在必须第一
    }
    firstRealEvent = false

    if (i === 0 && ev.type !== 'session.created' && events.some((e) => e.type === 'session.created')) {
      bad.push(`${label}: session.created 未居首`)
    }

    switch (ev.type) {
      case 'input_audio_buffer.speech_started':
        if (speechOpen) bad.push(`${label}: speech_started 出现在上一段 speech 未 stopped 期间`)
        speechOpen = true
        break
      case 'input_audio_buffer.speech_stopped':
        if (!speechOpen) bad.push(`${label}: speech_stopped 无配对 speech_started`)
        speechOpen = false
        break
      case 'response.created': {
        const id = responseIdOf(ev)
        if (!id) bad.push(`${label}: response.created 缺 id`)
        else if (seenCreated.has(id)) bad.push(`${label}: response ${id} 重复 created`)
        else seenCreated.add(id)
        break
      }
      case 'response.output_audio.delta':
      case 'response.output_audio_transcript.delta':
      case 'response.output_audio.done':
      case 'response.output_audio_transcript.done': {
        const id = responseIdOf(ev)
        if (!id) bad.push(`${label}: 缺 response_id`)
        else if (!seenCreated.has(id)) bad.push(`${label}: response ${id} 未 created 即出帧`)
        else if (closedAt.has(id)) bad.push(`${label}: response ${id} 终态后迟到帧（泄漏）`)
        break
      }
      case 'response.done': {
        const id = responseIdOf(ev)
        if (!id) bad.push(`${label}: response.done 缺 id`)
        else if (!seenCreated.has(id)) bad.push(`${label}: response ${id} 未 created 即 done`)
        else if (closedAt.has(id)) bad.push(`${label}: response ${id} 重复 done`)
        else closedAt.set(id, i)
        break
      }
      default:
        break
    }
  })

  if (speechOpen) bad.push('事件流结尾 speech 段未闭合（无 speech_stopped）')
  return bad
}
