/**
 * voice/session/voiceSession.test.ts — 无模型回放断言（M2 退出条件）。
 * 用 M1 实录三场景 + 两条违序负例驱动纯状态机与有序性检查。
 */
import { describe, expect, it } from 'vitest'
import {
  initialVoiceSession,
  orderingViolations,
  reduceVoiceSession,
  voiceTranscriptMessages,
  type VoiceSessionState,
} from './voiceSession'
import {
  FIXTURE_BAD_LATE_FRAME,
  FIXTURE_BAD_ORPHAN_DELTA,
  FIXTURE_BARGEIN,
  FIXTURE_CANCEL,
  FIXTURE_TURN,
} from './fixtures'
import type { ServerEvent } from '../protocol/events'

const PCM_FRAME_BYTES = 12 // 夹具示意帧解码后字节数（'AAAAggAAAMABAAAA' base64 → 12B）

function replay(events: ServerEvent[]): VoiceSessionState {
  let st = initialVoiceSession()
  for (const ev of events) {
    const bytes = ev.type === 'response.output_audio.delta' ? PCM_FRAME_BYTES : 0
    st = reduceVoiceSession(st, ev, bytes)
  }
  return st
}

describe('orderingViolations（实录三场景必须零违规）', () => {
  it('turn：有序', () => expect(orderingViolations(FIXTURE_TURN)).toEqual([]))
  it('cancel：有序', () => expect(orderingViolations(FIXTURE_CANCEL)).toEqual([]))
  it('bargein：有序（旧 response 完整闭合先于插话 speech_started）', () =>
    expect(orderingViolations(FIXTURE_BARGEIN)).toEqual([]))

  it('负例：孤儿帧被逮', () => {
    const bad = orderingViolations(FIXTURE_BAD_ORPHAN_DELTA)
    expect(bad.some((v) => v.includes('resp_ghost'))).toBe(true)
  })

  it('负例：终态后迟到帧 + speech 未闭合被逮', () => {
    const bad = orderingViolations(FIXTURE_BAD_LATE_FRAME)
    expect(bad.some((v) => v.includes('迟到帧'))).toBe(true)
    expect(bad.some((v) => v.includes('未闭合'))).toBe(true)
  })
})

describe('reduceVoiceSession（turn）', () => {
  const st = replay(FIXTURE_TURN)

  it('终点相位 idle、单一 completed response', () => {
    expect(st.phase).toBe('idle')
    expect(st.responseOrder).toEqual(['resp_turn_1'])
    expect(st.responses['resp_turn_1'].status).toBe('completed')
    expect(st.responses['resp_turn_1'].closed).toBe(true)
  })

  it('用户转写定稿唯一且权威', () => {
    expect(st.committedTranscripts).toEqual(['你好请用两句话介绍一下你自己'])
    expect(st.responses['resp_turn_1'].userTranscript).toBe('你好请用两句话介绍一下你自己')
    expect(st.responses['resp_turn_1'].userItemId).toBe('item_user_1')
    expect(st.pendingUserTranscripts).toEqual([])
    expect(st.partialTranscript).toBe('')
  })

  it('助手文本=transcript done 全量（非 delta 拼接漂移）', () => {
    expect(st.assistantText).toBe('你好呀，我是你的语音助手gaea。很高兴跟你聊天，有什么想聊的尽管说！')
    expect(st.responses['resp_turn_1'].assistantText).toBe(st.assistantText)
  })

  it('音频与文本账目对齐（12 帧 × 12B；三段 delta 字符数）', () => {
    expect(st.responses['resp_turn_1'].audioBytes).toBe(12 * PCM_FRAME_BYTES)
    expect(st.responses['resp_turn_1'].textChars).toBe(
      '你好呀，'.length + '我是你的语音助手gaea。'.length + '很高兴跟你聊天，有什么想聊的尽管说！'.length,
    )
  })

  it('零泄漏、零未知事件、零错误', () => {
    expect(st.responses['resp_turn_1'].staleBytesAfterDone).toBe(0)
    expect(st.unknownEventCount).toBe(0)
    expect(st.errors).toEqual([])
  })

  it('相位全程经历 listening→thinking→speaking→idle', () => {
    const phases: string[] = []
    let s = initialVoiceSession()
    for (const ev of FIXTURE_TURN) {
      s = reduceVoiceSession(s, ev, ev.type === 'response.output_audio.delta' ? PCM_FRAME_BYTES : 0)
      if (phases[phases.length - 1] !== s.phase) phases.push(s.phase)
    }
    expect(phases).toEqual(['idle', 'listening', 'thinking', 'speaking', 'idle'])
  })
})

describe('reduceVoiceSession（cancel）', () => {
  const st = replay(FIXTURE_CANCEL)

  it('cancelled/client_cancelled → 回 idle', () => {
    expect(st.phase).toBe('idle')
    expect(st.responses['resp_cancel_1'].status).toBe('cancelled')
    expect(st.responses['resp_cancel_1'].reason).toBe('client_cancelled')
  })
})

describe('reduceVoiceSession（bargein）', () => {
  const st = replay(FIXTURE_BARGEIN)

  it('两轮账目齐全：旧轮 cancelled/turn_detected、新轮 completed', () => {
    expect(st.responseOrder).toEqual(['resp_barge_1', 'resp_barge_2'])
    expect(st.responses['resp_barge_1'].status).toBe('cancelled')
    expect(st.responses['resp_barge_1'].reason).toBe('turn_detected')
    expect(st.responses['resp_barge_2'].status).toBe('completed')
    expect(st.phase).toBe('idle')
  })

  it('两轮用户转写均入账', () => {
    expect(st.committedTranscripts).toEqual([
      '你好请用两句话介绍一下你自己',
      '等等先别说了我想问明天天气怎么样',
    ])
  })

  it('打断路径相位含 interrupted（随后被 speech_started 接管为 listening）', () => {
    const phases: string[] = []
    let s = initialVoiceSession()
    for (const ev of FIXTURE_BARGEIN) {
      s = reduceVoiceSession(s, ev, ev.type === 'response.output_audio.delta' ? PCM_FRAME_BYTES : 0)
      if (phases[phases.length - 1] !== s.phase) phases.push(s.phase)
    }
    expect(phases).toEqual(['idle', 'listening', 'thinking', 'speaking', 'interrupted', 'listening', 'thinking', 'speaking', 'idle'])
  })

  it('终态后旧代际音频帧只记账不参与播放态', () => {
    let s = replay(FIXTURE_BARGEIN)
    s = reduceVoiceSession(s, { type: 'response.output_audio.delta', response_id: 'resp_barge_1', delta: 'AAAA' }, 4)
    expect(s.responses['resp_barge_1'].staleBytesAfterDone).toBe(4)
    expect(s.phase).toBe('idle') // 不因迟到帧复活 speaking
  })
})

describe('response 与用户转写精确配对', () => {
  it('多轮定稿按用户、助手交替展开，而不是先列全部用户再列最后回答', () => {
    let s = initialVoiceSession()
    for (const turn of [
      { itemId: 'u1', responseId: 'r1', user: '第一轮问题', assistant: '第一轮回答' },
      { itemId: 'u2', responseId: 'r2', user: '第二轮问题', assistant: '第二轮回答' },
    ]) {
      s = reduceVoiceSession(s, {
        type: 'conversation.item.input_audio_transcription.completed',
        item_id: turn.itemId,
        transcript: turn.user,
      })
      s = reduceVoiceSession(s, {
        type: 'response.created',
        response: { id: turn.responseId, status: 'in_progress' },
      })
      s = reduceVoiceSession(s, {
        type: 'response.output_audio_transcript.done',
        response_id: turn.responseId,
        transcript: turn.assistant,
      })
      s = reduceVoiceSession(s, {
        type: 'response.done',
        response: { id: turn.responseId, status: 'completed' },
      })
    }

    expect(voiceTranscriptMessages(s).map(({ role, text }) => [role, text])).toEqual([
      ['user', '第一轮问题'],
      ['assistant', '第一轮回答'],
      ['user', '第二轮问题'],
      ['assistant', '第二轮回答'],
    ])
  })

  it('旧 response.done 迟到时仍取旧轮用户文本，不串到下一句', () => {
    let s = initialVoiceSession()
    s = reduceVoiceSession(s, {
      type: 'conversation.item.input_audio_transcription.completed',
      item_id: 'u1',
      transcript: '第一轮问题',
    })
    s = reduceVoiceSession(s, { type: 'response.created', response: { id: 'r1', status: 'in_progress' } })
    s = reduceVoiceSession(s, {
      type: 'conversation.item.input_audio_transcription.completed',
      item_id: 'u2',
      transcript: '第二轮问题',
    })
    s = reduceVoiceSession(s, {
      type: 'response.output_audio_transcript.delta',
      response_id: 'r1',
      delta: '第一轮回答',
    })
    s = reduceVoiceSession(s, {
      type: 'response.done',
      response: { id: 'r1', status: 'completed' },
    })

    expect(s.responses.r1.userTranscript).toBe('第一轮问题')
    expect(s.responses.r1.assistantText).toBe('第一轮回答')
    expect(s.pendingUserTranscripts).toEqual([{ text: '第二轮问题', itemId: 'u2' }])
    expect(s.activeResponseId).toBe('r1')
  })

  it('相同用户文本仍按 item_id 保持轮次归属', () => {
    let s = initialVoiceSession()
    for (const id of ['u1', 'u2']) {
      s = reduceVoiceSession(s, {
        type: 'conversation.item.input_audio_transcription.completed',
        item_id: id,
        transcript: '再说一遍',
      })
      s = reduceVoiceSession(s, { type: 'response.created', response: { id: `r-${id}`, status: 'in_progress' } })
    }
    expect(s.responses['r-u1'].userItemId).toBe('u1')
    expect(s.responses['r-u2'].userItemId).toBe('u2')
  })

  it('重复 response.done 不重复闭合或累积错误', () => {
    let s = replay(FIXTURE_TURN)
    const done = FIXTURE_TURN[FIXTURE_TURN.length - 1]
    s = reduceVoiceSession(s, done)
    expect(s.responses.resp_turn_1.status).toBe('completed')
    expect(s.responses.resp_turn_1.closed).toBe(true)
    expect(s.errors).toEqual([])
  })
})
