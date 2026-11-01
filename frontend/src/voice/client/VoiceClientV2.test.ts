/**
 * voice/client/VoiceClientV2.test.ts — 回放桩 transport 上的整链测试（M2）。
 * 不需要真实模型/网络：实录夹具直接灌 handleServerEvent。
 */
import { describe, expect, it } from 'vitest'
import { VoiceClientV2, pcmToBase64, type VoiceTransport } from './VoiceClientV2'
import { FIXTURE_BARGEIN, FIXTURE_TURN } from '../session/fixtures'
import type { ServerEvent } from '../protocol/events'

class ReplayTransport implements VoiceTransport {
  sent: string[] = []
  onopen: (() => void) | null = null
  onclose: (() => void) | null = null
  onerror: ((err: unknown) => void) | null = null
  onmessage: ((data: string) => void) | null = null
  send(data: string): void {
    this.sent.push(data)
  }
  close(): void {
    this.onclose?.()
  }
  /** 逐事件驱动客户端（模拟服务端线缆输出） */
  emit(ev: ServerEvent): void {
    this.onmessage?.(JSON.stringify(ev))
  }
}

function makeClient() {
  const transport = new ReplayTransport()
  const playbackDone: string[] = []
  const states: string[] = []
  const client = new VoiceClientV2({
    transport,
    instructions: '你是语音助手',
    onState: (s) => states.push(s.phase),
    onPlaybackDone: (id) => playbackDone.push(id),
  })
  return { transport, client, playbackDone, states }
}

describe('VoiceClientV2（回放桩）', () => {
  it('连接即发 session.update（server_vad + 指令）', () => {
    const { transport } = makeClient()
    transport.onopen?.()
    expect(transport.sent).toHaveLength(1)
    const ev = JSON.parse(transport.sent[0])
    expect(ev.type).toBe('session.update')
    expect(ev.session.audio.input.turn_detection.type).toBe('server_vad')
    expect(ev.session.instructions).toBe('你是语音助手')
  })

  it('turn 实录：状态到 idle、播放队列闭合回执带 response id', () => {
    const { client, transport, playbackDone } = makeClient()
    for (const ev of FIXTURE_TURN) transport.emit(ev)
    const s = client.snapshot
    expect(s.phase).toBe('idle')
    expect(s.committedTranscripts).toEqual(['你好请用两句话介绍一下你自己'])
    // 夹具 12 帧全入队；未拉取 → 回执等排空，此处仅验证闭合语义
    const st = client.playback.stats
    expect(st.currentResponseId).toBe('resp_turn_1')
    expect(st.closed).toBe(true)
    expect(playbackDone).toEqual([]) // 存量未播 → 尚无回执
    client.playback.drainTo(() => {})
    expect(playbackDone).toEqual(['resp_turn_1'])
  })

  it('bargein 实录：done(cancelled) 即冲刷未播块，回执带旧 id，新代际不复活', () => {
    const { client, transport, playbackDone } = makeClient()
    for (const ev of FIXTURE_BARGEIN) {
      transport.emit(ev)
    }
    const s = client.snapshot
    expect(s.responseOrder).toEqual(['resp_barge_1', 'resp_barge_2'])
    expect(s.responses['resp_barge_1'].status).toBe('cancelled')
    // resp_barge_1 的 3 帧（36B）在其 done(cancelled/turn_detected) 时被冲刷；
    // 旧代际回执在冲刷瞬间触发（不等 open(r2)/drain）
    expect(client.playback.stats.droppedBytes).toBe(36)
    expect(playbackDone).toEqual(['resp_barge_1'])
    expect(client.playback.stats.currentResponseId).toBe('resp_barge_2')
    client.playback.drainTo(() => {})
    expect(playbackDone).toEqual(['resp_barge_1', 'resp_barge_2'])
  })

  it('cancel()：发 response.cancel 并本地冲刷', () => {
    const { client, transport } = makeClient()
    transport.onopen?.()
    transport.sent.length = 0
    client.cancel()
    expect(JSON.parse(transport.sent[0]).type).toBe('response.cancel')
  })

  it('麦克风帧 base64 往返无损', () => {
    const pcm = new Int16Array([0, 1, -1, 32767, -32768, 128])
    const b64 = pcmToBase64(pcm)
    const bin = atob(b64)
    const back = new Int16Array(bin.length / 2)
    for (let i = 0; i < back.length; i++) {
      back[i] = bin.charCodeAt(i * 2) | (bin.charCodeAt(i * 2 + 1) << 8)
    }
    expect([...back]).toEqual([...pcm])
  })

  it('坏 JSON 不炸会话、入错误账', () => {
    const { client, transport } = makeClient()
    transport.onmessage?.('{oops')
    expect(client.snapshot.errors).toHaveLength(1)
    expect(client.snapshot.phase).toBe('idle')
  })
})
