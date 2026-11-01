/**
 * voice/session/voiceSession.test.ts — 无模型回放断言（M2 退出条件）。
 * 用 M1 实录三场景 + 两条违序负例驱动纯状态机与有序性检查。
 */
import { describe, expect, it } from 'vitest'
import {
  initialVoiceSession,
  orderingViolations,
  reduceVoiceSession,
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
    expect(st.partialTranscript).toBe('')
  })

  it('助手文本=transcript done 全量（非 delta 拼接漂移）', () => {
    expect(st.assistantText).toBe('你好呀，我是你的语音助手gaea。很高兴跟你聊天，有什么想聊的尽管说！')
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
