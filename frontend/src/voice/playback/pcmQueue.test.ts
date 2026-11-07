/**
 * voice/playback/pcmQueue.test.ts — 代际闸/冲刷/回执语义（M2）。
 */
import { describe, expect, it } from 'vitest'
import { PlaybackQueue } from './pcmQueue'

function pcm(bytes: number): Int16Array {
  return new Int16Array(bytes / 2).fill(1)
}

describe('PlaybackQueue', () => {
  it('单一 response：open→入队→拉取→闭合→回执', () => {
    const q = new PlaybackQueue()
    q.open('r1')
    q.append('r1', pcm(100))
    q.append('r1', pcm(200))
    expect(q.pull()).toBeInstanceOf(Int16Array)
    expect(q.stats.playbackDone).toBe(false) // 还有一块未拉
    expect(q.pull()).toBeInstanceOf(Int16Array)
    expect(q.stats.playbackDone).toBe(false) // 未闭合
    q.markClosed('r1')
    expect(q.stats.playbackDone).toBe(true) // 闭合且排空 → 回执
    expect(q.stats).toMatchObject({ playedBytes: 300, droppedBytes: 0, currentResponseId: 'r1', closed: true })
  })

  it('闭合后仍有存量：排空后才回执（播放完成≠响应完成）', () => {
    const q = new PlaybackQueue()
    q.open('r1')
    q.append('r1', pcm(100))
    q.markClosed('r1')
    expect(q.stats.playbackDone).toBe(false) // 存量未播完
    q.pull()
    expect(q.stats.playbackDone).toBe(true)
  })

  it('open 新代际：旧存量保留排队（上游 FIFO 同位，2026-11-07 拍板「应该排队」）；回执逐响应按序', () => {
    const q = new PlaybackQueue()
    const done: string[] = []
    q.onDone((id) => done.push(id))
    q.open('r1')
    q.append('r1', pcm(100))
    q.open('r2') // 代际切换：r1 存量保留（不再作废）
    q.append('r2', pcm(200))
    q.markClosed('r1')
    q.markClosed('r2')
    expect(q.stats.pendingChunkCount).toBe(2)
    // FIFO 到达序：r1 先播完出回执，r2 随后
    q.pull()
    expect(done).toEqual(['r1'])
    q.pull()
    expect(done).toEqual(['r1', 'r2'])
    const s = q.stats
    expect(s.droppedBytes).toBe(0)
    expect(s.enqueuedBytes).toBe(300)
    expect(s.playbackDone).toBe(true) // 当前代际 r2 已回执
  })

  it('flush 冲刷跨响应存量；未闭合响应不回执（打断语义保留）', () => {
    const q = new PlaybackQueue()
    const done: string[] = []
    q.onDone((id) => done.push(id))
    q.open('r1')
    q.append('r1', pcm(100))
    q.markClosed('r1')
    q.open('r2')
    q.append('r2', pcm(200))
    const dropped = q.flush()
    expect(dropped).toBe(300)
    expect(done).toEqual(['r1']) // 已闭合的 r1 收口回执；r2 未闭合不回执
    expect(q.stats.pendingChunkCount).toBe(0)
  })

  it('barge-in 全序：flush（不回执）→ 旧代闭合（回执）→ 新代开启', () => {
    const q = new PlaybackQueue()
    q.open('r1')
    q.append('r1', pcm(100))
    q.pull() // 播了 100
    q.append('r1', pcm(300))
    const dropped = q.flush()
    expect(dropped).toBe(300)
    expect(q.stats.playbackDone).toBe(false) // 回执等 done，flush 单独不算
    q.markClosed('r1') // 服务端 response.done(cancelled/turn_detected) 到达
    expect(q.stats.playbackDone).toBe(true)
    q.open('r2') // 新代际重置回执
    expect(q.stats).toMatchObject({ currentResponseId: 'r2', playbackDone: false, droppedBytes: 300 })
  })

  it('未 open 直接 append：全部按迟到帧丢弃（防御）', () => {
    const q = new PlaybackQueue()
    q.append('rx', pcm(60))
    expect(q.stats.droppedBytes).toBe(60)
    expect(q.stats.currentResponseId).toBe('')
  })

  it('reset：清账并可开启全新代际', () => {
    const q = new PlaybackQueue()
    q.open('r1')
    q.append('r1', pcm(100))
    q.reset()
    expect(q.stats.currentResponseId).toBe('')
    q.open('r2')
    q.append('r2', pcm(40))
    expect(q.stats).toMatchObject({ currentResponseId: 'r2', enqueuedBytes: 40, droppedBytes: 0 })
  })

  it('drainTo：按到达序喂 sink', () => {
    const q = new PlaybackQueue()
    q.open('r1')
    q.append('r1', pcm(20))
    q.append('r1', pcm(40))
    const got: number[] = []
    q.drainTo((chunk) => got.push(chunk.byteLength))
    expect(got).toEqual([20, 40])
  })
})
