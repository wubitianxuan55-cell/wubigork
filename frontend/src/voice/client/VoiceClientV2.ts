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
  /** 连接生命周期 */
  onOpen?: () => void
  onClose?: () => void
  /** response.done 终态回执（轮完成/取消的协议级事件；写回聊天等消费方用） */
  onResponseDone?: (info: { responseId: string; status: string; reason: string }) => void
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
    t.onerror = () => {
      /* 错误面交给 onclose；避免未处理回调炸桩 */
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
    const bytes = ev.type === 'response.output_audio.delta' ? this.decodeAudioBytes(ev.delta) : 0
    switch (ev.type) {
      case 'response.created': {
        const id = this.responseId(ev)
        if (id) this.playback.open(id)
        break
      }
      case 'response.output_audio.delta': {
        const id = this.responseId(ev)
        const pcm = this.decodeAudioPcm(ev.delta)
        if (id && pcm) this.playback.append(id, pcm)
        break
      }
      case 'response.done': {
        const id = this.responseId(ev)
        if (id) {
          this.playback.markClosed(id)
          const { status, reason } = responseStatusOf(ev)
          // 非完成态（打断/取消/失败/截断）：未播块立即冲刷，不等下一代际
          if (status !== 'completed') this.playback.flush()
          this.opts.onResponseDone?.({ responseId: id, status, reason })
        }
        break
      }
      case 'input_audio_buffer.speech_started': {
        // barge-in 兜底：服务端若先发 speech_started 再补 done，也不留旧音频
        if (this.state.phase === 'speaking') this.playback.flush()
        break
      }
      default:
        break
    }
    this.state = reduceVoiceSession(this.state, ev, bytes)
    this.opts.onState?.(this.state)
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
