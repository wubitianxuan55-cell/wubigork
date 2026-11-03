/**
 * voice/protocol/events.ts — Voice V2 Realtime 协议类型固化（M2）。
 *
 * 形状以 M1 Spike 实测为准（huggingface/speech-to-speech@6feb527 的
 * OpenAI Realtime GA 兼容面；详见 进度计划/gaea-voice-v2-m1-spike-report-20261101.md）。
 * 未知/扩展事件必须可容忍通过（gaea 扩展 soft_ended/reopened/superseded/
 * latency_trace 尚未由运行时发射，先保证解码不丢事件）。
 */

// ─────────────────────────── 共享形状 ───────────────────────────

/** server_vad 轮次检测配置（session.update.audio.input.turn_detection） */
export interface ServerVadConfig {
  type: 'server_vad'
  threshold?: number
  interrupt_response?: boolean
  create_response?: boolean
}

/** 会话更新（client → server，深合并语义） */
export interface SessionUpdate {
  type?: 'realtime'
  instructions?: string
  audio?: {
    input?: { turn_detection?: ServerVadConfig; format?: unknown }
    output?: { voice?: string; format?: unknown }
  }
}

/** 服务端会话快照（session.created / session.updated 回执） */
export interface SessionSnapshot {
  type?: string
  id?: string
  instructions?: string
  audio?: unknown
  [key: string]: unknown
}

export type ResponseStatus = 'in_progress' | 'completed' | 'cancelled' | 'incomplete' | 'failed'

export interface ResponseInfo {
  id: string
  status: ResponseStatus
  status_details?: { reason?: string } & Record<string, unknown>
  [key: string]: unknown
}

// ─────────────────────────── Client → Server ───────────────────────────

export type ClientEvent =
  | { type: 'session.update'; session: SessionUpdate }
  /** base64 PCM16@16k 单声道音频帧 */
  | { type: 'input_audio_buffer.append'; audio: string }
  | { type: 'input_audio_buffer.commit' }
  | { type: 'input_audio_buffer.clear' }
  | { type: 'conversation.item.create'; item: Record<string, unknown> }
  | { type: 'response.create'; response?: Record<string, unknown>; instructions?: string }
  | { type: 'response.cancel' }

// ─────────────────────────── Server → Client（已知事件） ───────────────────────────

export type KnownServerEvent =
  | { type: 'session.created'; session: SessionSnapshot; event_id?: string }
  | { type: 'session.updated'; session: SessionSnapshot; event_id?: string }
  | { type: 'input_audio_buffer.speech_started'; audio_start_ms?: number; item_id?: string; event_id?: string }
  | { type: 'input_audio_buffer.speech_stopped'; audio_end_ms?: number; item_id?: string; event_id?: string }
  | { type: 'input_audio_buffer.committed'; item_id?: string; event_id?: string }
  | { type: 'conversation.item.created'; item?: Record<string, unknown>; event_id?: string }
  | { type: 'conversation.item.input_audio_transcription.delta'; item_id: string; delta: string; event_id?: string }
  | { type: 'conversation.item.input_audio_transcription.completed'; item_id: string; transcript: string; event_id?: string }
  | { type: 'response.created'; response: ResponseInfo; event_id?: string }
  | { type: 'response.output_item.added'; response_id: string; output_index?: number; item?: Record<string, unknown>; event_id?: string }
  | { type: 'response.output_item.done'; response_id: string; output_index?: number; item?: Record<string, unknown>; event_id?: string }
  | { type: 'response.content_part.added'; response_id: string; item_id?: string; event_id?: string }
  | { type: 'response.content_part.done'; response_id: string; item_id?: string; event_id?: string }
  | { type: 'response.output_audio_transcript.delta'; response_id: string; item_id?: string; delta: string; event_id?: string }
  | { type: 'response.output_audio_transcript.done'; response_id: string; item_id?: string; transcript: string; event_id?: string }
  /** base64 PCM16 下行音频帧（管线重采样后 16k；音频量大时应升级 binary frame） */
  | { type: 'response.output_audio.delta'; response_id: string; item_id?: string; delta: string; event_id?: string }
  | { type: 'response.output_audio.done'; response_id: string; item_id?: string; event_id?: string }
  | { type: 'response.done'; response: ResponseInfo; event_id?: string }
  | { type: 'error'; error: { message?: string; [key: string]: unknown }; event_id?: string }

/** 未知/扩展事件：type 原样保留，字段不丢 */
export interface UnknownServerEvent {
  type: string
  [key: string]: unknown
}

export type ServerEvent = KnownServerEvent | UnknownServerEvent

const KNOWN_SERVER_TYPES: ReadonlySet<string> = new Set([
  'session.created',
  'session.updated',
  'input_audio_buffer.speech_started',
  'input_audio_buffer.speech_stopped',
  'input_audio_buffer.committed',
  'conversation.item.created',
  'conversation.item.input_audio_transcription.delta',
  'conversation.item.input_audio_transcription.completed',
  'response.created',
  'response.output_item.added',
  'response.output_item.done',
  'response.content_part.added',
  'response.content_part.done',
  'response.output_audio_transcript.delta',
  'response.output_audio_transcript.done',
  'response.output_audio.delta',
  'response.output_audio.done',
  'response.done',
  'error',
])

export function isKnownServerEvent(ev: ServerEvent): ev is KnownServerEvent {
  return KNOWN_SERVER_TYPES.has(ev.type)
}

export type DecodeResult = { ok: true; event: ServerEvent } | { ok: false; error: string }

/** 线缆 JSON → 事件。非对象/非字符串 type 拒收，其余宽容。 */
export function decodeServerEvent(raw: string): DecodeResult {
  let parsed: unknown
  try {
    parsed = JSON.parse(raw)
  } catch {
    return { ok: false, error: 'invalid json' }
  }
  if (typeof parsed !== 'object' || parsed === null) {
    return { ok: false, error: 'event is not an object' }
  }
  const ev = parsed as Record<string, unknown>
  if (typeof ev.type !== 'string' || ev.type.length === 0) {
    return { ok: false, error: 'missing event type' }
  }
  return { ok: true, event: ev as ServerEvent }
}

/** 事件 → 线缆 JSON（ClientEvent 均为纯数据，直接序列化） */
export function encodeClientEvent(ev: ClientEvent): string {
  return JSON.stringify(ev)
}

// ─────────────────────────── 事件取值助手（宽容读取） ───────────────────────────

/** response.done / response.created 的 response.id（缺失返回 ''） */
export function responseIdOf(ev: ServerEvent): string {
  if ('response' in ev && ev.response && typeof ev.response === 'object') {
    const id = (ev.response as ResponseInfo).id
    if (typeof id === 'string') return id
  }
  if ('response_id' in ev && typeof ev.response_id === 'string') return ev.response_id
  return ''
}

/** response.done 的终态与原因（"cancelled/turn_detected" 形态用于展示） */
export function responseStatusOf(ev: ServerEvent): { status: string; reason: string } {
  if ('response' in ev && ev.response && typeof ev.response === 'object') {
    const r = ev.response as ResponseInfo
    const reason = r.status_details?.reason ?? ''
    return { status: typeof r.status === 'string' ? r.status : '', reason: typeof reason === 'string' ? reason : '' }
  }
  return { status: '', reason: '' }
}
