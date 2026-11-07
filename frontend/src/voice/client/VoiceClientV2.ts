/**
 * voice/client/VoiceClientV2.ts — Realtime 客户端（M2，transport 无关）。
 *
 * 职责边界（V2 方案 §5.2 frontend/src/voice/realtime-client）：
 * - 唯一有序事件入口：transport 收到的 JSON 统一 decodeServerEvent；
 * - 会话状态：reduceVoiceSession 纯归约，UI 只读快照；
 * - 播放调度：response.created→queue.open、audio.delta→queue.append、
 *   response.done→queue.markClosed、barge-in（speech_started 且在播）→queue.flush；
 * - 发送面：session.update（连接即发）、麦克风帧 base64、response.cancel。
 * transport 可注入（WebSocket / 回放桩），无模型即可整链测试。
 */

import {
  decodeServerEvent,
  encodeClientEvent,
  responseStatusOf,
  type ClientEvent,
  type ServerEvent,
} from '../protocol/events'
import { initialVoiceSession, reduceVoiceSession, type VoiceSessionState } from '../session/voiceSession'
import { PlaybackQueue } from '../playback/pcmQueue'

/** transport 契约：浏览器 WebSocket 满足；测试用回放桩 */
export interface VoiceTransport {
  send(data: string): void
  close(): void
  onopen: (() => void) | null
  onclose: (() => void) | null
  onerror: ((err: unknown) => void) | null
  onmessage: ((data: string) => void) | null
}

export interface VoiceClientOptions {
  transport: VoiceTransport
  instructions?: string
  /** 输出音色（OpenAI Realtime session.audio.output.voice；breeze 克隆 id） */
  voice?: string
  vadThreshold?: number
  interruptResponse?: boolean
  /** 状态快照变化回调（UI 订阅点） */
  onState?: (state: VoiceSessionState) => void
  /** 回放完成回执（带 response id，替代全局布尔） */
  onPlaybackDone?: (responseId: string) => void
  /** 下行音频实时电平（0..1，播放 PCM 块 RMS；0=静默。说话相位声纹驱动源） */
  onPlayLevel?: (level: number) => void
  /** 连接生命周期 */
  onOpen?: () => void
  onClose?: () => void
  onError?: (error: unknown) => void
  /** response.done 终态回执（轮完成/取消的协议级事件；写回聊天等消费方用） */
  onResponseDone?: (info: {
    responseId: string
    status: string
    reason: string
    user: string
    itemId: string
    assistant: string
  }) => void
}

export class VoiceClientV2 {
  readonly playback = new PlaybackQueue()
  private state: VoiceSessionState = initialVoiceSession()
  private opts: VoiceClientOptions
  private disposed = false

  constructor(opts: VoiceClientOptions) {
    this.opts = opts
    this.playback.onDone((id) => opts.onPlaybackDone?.(id))
    const t = opts.transport
    t.onopen = () => {
      this.sendEvent({
        type: 'session.update',
        session: {
          type: 'realtime',
          ...(opts.instructions ? { instructions: opts.instructions } : {}),
          audio: {
            input: {
              turn_detection: {
                type: 'server_vad',
                ...(opts.vadThreshold !== undefined ? { threshold: opts.vadThreshold } : {}),
                ...(opts.interruptResponse !== undefined ? { interrupt_response: opts.interruptResponse } : {}),
              },
            },
            // 人格音色（breeze 克隆 id；空=服务端 YAML 缺省）——声色固定根修
            ...(opts.voice ? { output: { voice: opts.voice } } : {}),
          },
        },
      })
      opts.onOpen?.()
    }
    t.onclose = () => {
      opts.onClose?.()
    }
    t.onerror = (error) => {
      opts.onError?.(error)
    }
    t.onmessage = (data: string) => {
      const decoded = decodeServerEvent(data)
      if (!decoded.ok) {
        this.pushError(`decode: ${decoded.error}`)
        return
      }
      this.handleServerEvent(decoded.event)
    }
  }

  /** 事件唯一入口（transport 与 mock 回放共用） */
  handleServerEvent(ev: ServerEvent): void {
    const responseId = this.responseId(ev)
    // WebSocket 重连/服务端重试可能重复 done。终态只消费一次，避免聊天双写。
    if (ev.type === 'response.done' && responseId && this.state.responses[responseId]?.closed) {
      return
    }
    const bytes = ev.type === 'response.output_audio.delta' ? this.decodeAudioBytes(ev.delta) : 0
    switch (ev.type) {
      case 'response.created': {
        const id = responseId
        if (id) this.playback.open(id)
        break
      }
      case 'response.output_audio.delta': {
        const id = responseId
        const pcm = this.decodeAudioPcm(ev.delta)
        if (id && pcm) {
          this.playback.append(id, pcm)
          // 说话相位声纹源：播放块 RMS（×3 刻度对齐采集侧 0..1 观感）
          let sum = 0
          for (let i = 0; i < pcm.length; i++) {
            const s = pcm[i] / 32768
            sum += s * s
          }
          const rms = Math.sqrt(sum / Math.max(pcm.length, 1)) * 3
          this.opts.onPlayLevel?.(Math.min(rms, 1))
        }
        break
      }
      case 'response.done': {
        const id = responseId
        this.opts.onPlayLevel?.(0)
        if (id) {
          this.playback.markClosed(id)
          const { status } = responseStatusOf(ev)
          // 仅取消态冲刷未播块（审计 P3：上游 done(cancelled) 才 clear，
          // audio_client.py:536-539；incomplete/failed 的已发音频依然有效可播
          // ——此前 status!==completed 一律 flush=截断响应被二次截断）
          if (status === 'cancelled') this.playback.flush()
        }
        break
      }
      case 'input_audio_buffer.speech_started': {
        // barge-in 兜底：不能只看 phase——response.done 可能已先把相位置 idle，
        // 但播放器仍有未播 PCM。此时用户开口必须立即冲刷旧代际。
        // 勿打断模式（interrupt_response=false，审计 P1）：服务端刻意不取消响应
        // （audio.py:361-367），客户端掐播=断续——冲刷只在打断开启时执行。
        if (this.opts.interruptResponse !== false && this.playback.stats.pendingChunkCount > 0) {
          this.playback.flush()
        }
        break
      }
      default:
        break
    }
    this.state = reduceVoiceSession(this.state, ev, bytes)
    this.opts.onState?.(this.state)
    if (ev.type === 'response.done' && responseId) {
      const { status, reason } = responseStatusOf(ev)
      const led = this.state.responses[responseId]
      this.opts.onResponseDone?.({
        responseId,
        status,
        reason,
        user: led?.userTranscript ?? '',
        itemId: led?.userItemId ?? '',
        assistant: led?.assistantText ?? '',
      })
    }
  }

  /** 麦克风帧（PCM16@16k 单声道）→ base64 append */
  sendAudioChunk(pcm: Int16Array): void {
    this.sendEvent({ type: 'input_audio_buffer.append', audio: pcmToBase64(pcm) })
  }

  /** 显式取消（客户端语义：cancelled/client_cancelled） */
  cancel(): void {
    this.sendEvent({ type: 'response.cancel' })
    this.playback.flush()
  }

  /**
   * 运行时改监听阈值（session.update）。
   * M5 刀一回执朗读的回声门：TTS 朗读扬声器出声期间拉高阈值（0.9≈禁听）
   * 防残余回声触发假插话（11-01 教训：0.6 被回声触发），朗读结束回落监听值。
   */
  updateVADThreshold(threshold: number): void {
    this.sendEvent({
      type: 'session.update',
      session: {
        type: 'realtime',
        audio: { input: { turn_detection: { type: 'server_vad', threshold } } },
      },
    })
  }

  /**
   * 运行时改语音打断开关（session.update；上游 M13 interrupt_response_enabled
   * 会话级旋钮产品化——关=回复完整播完不被语音打断，真打断走 UI 打断键）。
   * 值同步进 opts：重连/新会话 onopen 的 session.update 沿用当前值。
   */
  setInterruptResponse(enabled: boolean): void {
    this.opts.interruptResponse = enabled
    this.sendEvent({
      type: 'session.update',
      session: {
        type: 'realtime',
        audio: { input: { turn_detection: { type: 'server_vad', interrupt_response: enabled } } },
      },
    })
  }

  get snapshot(): VoiceSessionState {
    return this.state
  }

  dispose(): void {
    this.disposed = true
    this.opts.transport.close()
  }

  private sendEvent(ev: ClientEvent): void {
    if (this.disposed) return
    this.opts.transport.send(encodeClientEvent(ev))
  }

  private pushError(msg: string): void {
    this.state = { ...this.state, errors: [...this.state.errors, msg] }
    this.opts.onState?.(this.state)
  }

  private responseId(ev: ServerEvent): string {
    if ('response' in ev && ev.response && typeof ev.response === 'object') {
      const id = (ev.response as { id?: unknown }).id
      if (typeof id === 'string') return id
    }
    if ('response_id' in ev && typeof ev.response_id === 'string') return ev.response_id
    return ''
  }

  private decodeAudioBytes(b64: unknown): number {
    const pcm = this.decodeAudioPcm(b64)
    return pcm ? pcm.byteLength : 0
  }

  private decodeAudioPcm(b64: unknown): Int16Array | null {
    if (typeof b64 !== 'string' || b64.length === 0) return null
    try {
      const bin = atob(b64)
      const buf = new Uint8Array(bin.length)
      for (let i = 0; i < bin.length; i++) buf[i] = bin.charCodeAt(i)
      const aligned = buf.byteLength - (buf.byteLength % 2)
      return new Int16Array(buf.buffer, 0, aligned / 2)
    } catch {
      return null
    }
  }
}

export function pcmToBase64(pcm: Int16Array): string {
  const bytes = new Uint8Array(pcm.buffer, pcm.byteOffset, pcm.byteLength)
  let bin = ''
  for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i])
  return btoa(bin)
}
