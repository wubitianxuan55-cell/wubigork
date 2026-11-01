/**
 * voice/session/fixtures.ts — M1 实测事件流的规范化回放夹具（M2）。
 *
 * 事件序照抄 2026-11-01 M1 Spike 三场景实录（.tmp/voice_v2 各场景 report_*.json）：
 * turn_final / cancel1 / bargein1；response id、转写文本沿用实测，
 * 助手文本与音频 delta 帧数为可辨识的仿真值（顺序语义与实录一致）。
 * 另附两条「违序流」用于有序性断言的负例。
 */

import type { ServerEvent } from '../protocol/events'

/** 一次 response 的完整闭合序列（audio N 帧示意；帧体为短 base64 PCM） */
function responseLifecycle(id: string, opts: {
  audioFrames: number
  transcriptDeltas: string[]
  status: 'completed' | 'cancelled' | 'incomplete' | 'failed'
  reason?: string
}): ServerEvent[] {
  const evs: ServerEvent[] = [
    { type: 'response.created', response: { id, status: 'in_progress' } },
    { type: 'response.output_item.added', response_id: id, output_index: 0, item: { type: 'message', role: 'assistant' } },
    { type: 'response.content_part.added', response_id: id, item_id: `item_${id}` },
  ]
  for (const d of opts.transcriptDeltas) {
    evs.push({ type: 'response.output_audio_transcript.delta', response_id: id, item_id: `item_${id}`, delta: d })
  }
  const pcm = 'AAAAggAAAMABAAAA' // 8 字节 PCM16 示意帧
  for (let i = 0; i < opts.audioFrames; i++) {
    evs.push({ type: 'response.output_audio.delta', response_id: id, item_id: `item_${id}`, delta: pcm })
  }
  evs.push({ type: 'response.output_audio.done', response_id: id, item_id: `item_${id}` })
  evs.push({
    type: 'response.output_audio_transcript.done',
    response_id: id,
    item_id: `item_${id}`,
    transcript: opts.transcriptDeltas.join(''),
  })
  evs.push({ type: 'response.content_part.done', response_id: id, item_id: `item_${id}` })
  evs.push({ type: 'response.output_item.done', response_id: id, output_index: 0, item: { type: 'message' } })
  evs.push({
    type: 'response.done',
    response: { id, status: opts.status, ...(opts.reason ? { status_details: { reason: opts.reason } } : {}) },
  })
  return evs
}

function speechTurn(itemSeq: number, transcript: string): ServerEvent[] {
  return [
    { type: 'input_audio_buffer.speech_started', audio_start_ms: itemSeq * 1000 },
    { type: 'input_audio_buffer.speech_stopped', audio_end_ms: itemSeq * 1000 + 2600 },
    { type: 'input_audio_buffer.committed', item_id: `item_user_${itemSeq}` },
    { type: 'conversation.item.created', item: { id: `item_user_${itemSeq}`, role: 'user' } },
    { type: 'conversation.item.input_audio_transcription.completed', item_id: `item_user_${itemSeq}`, transcript },
  ]
}

const SESSION_HEAD: ServerEvent[] = [
  { type: 'session.created', session: { id: 'session_77ae68b153084e48aa148f1a06b17cb7', type: 'realtime' } },
  { type: 'session.updated', session: { id: 'session_77ae68b153084e48aa148f1a06b17cb7', type: 'realtime' } },
]

/** 场景一：单轮完整（turn_final 实录：133 音频帧、全句转写、completed） */
export const FIXTURE_TURN: ServerEvent[] = [
  ...SESSION_HEAD,
  ...speechTurn(1, '你好请用两句话介绍一下你自己'),
  ...responseLifecycle('resp_turn_1', {
    audioFrames: 12,
    transcriptDeltas: ['你好呀，', '我是你的语音助手gaea。', '很高兴跟你聊天，有什么想聊的尽管说！'],
    status: 'completed',
  }),
]

/** 场景二：客户端显式取消（cancel1 实录：1 帧后 response.cancel → cancelled/client_cancelled） */
export const FIXTURE_CANCEL: ServerEvent[] = [
  ...SESSION_HEAD,
  ...speechTurn(1, '你好请用两句话介绍一下你自己'),
  ...responseLifecycle('resp_cancel_1', {
    audioFrames: 1,
    transcriptDeltas: ['你好'],
    status: 'cancelled',
    reason: 'client_cancelled',
  }),
]

/** 场景三：barge-in（bargein1 实录：resp1 播报中被插话 → 完整闭合 → 新轮 completed） */
export const FIXTURE_BARGEIN: ServerEvent[] = [
  ...SESSION_HEAD,
  ...speechTurn(1, '你好请用两句话介绍一下你自己'),
  ...responseLifecycle('resp_barge_1', {
    audioFrames: 3,
    transcriptDeltas: ['你好呀，我是你的语音助手'],
    status: 'cancelled',
    reason: 'turn_detected',
  }),
  // 实测序：旧 response 完整闭合在前，插话 speech_started 在后
  ...speechTurn(2, '等等先别说了我想问明天天气怎么样'),
  ...responseLifecycle('resp_barge_2', {
    audioFrames: 6,
    transcriptDeltas: ['好的，', '明天天气我需要查一下才能告诉你。'],
    status: 'completed',
  }),
]

/** 负例一：response 未 created 即出帧 */
export const FIXTURE_BAD_ORPHAN_DELTA: ServerEvent[] = [
  ...SESSION_HEAD,
  { type: 'input_audio_buffer.speech_started' },
  { type: 'response.output_audio.delta', response_id: 'resp_ghost', delta: 'AAAA' },
]

/** 负例二：终态后迟到帧（泄漏）+ speech 段未闭合 */
export const FIXTURE_BAD_LATE_FRAME: ServerEvent[] = [
  ...SESSION_HEAD,
  { type: 'input_audio_buffer.speech_started' },
  { type: 'input_audio_buffer.speech_stopped' },
  ...responseLifecycle('resp_late_1', { audioFrames: 1, transcriptDeltas: ['嗯'], status: 'completed' }),
  { type: 'response.output_audio.delta', response_id: 'resp_late_1', delta: 'AAAA' },
  { type: 'input_audio_buffer.speech_started' }, // 结尾未闭合
]
