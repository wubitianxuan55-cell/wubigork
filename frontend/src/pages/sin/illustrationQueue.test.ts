// sin/illustrationQueue.test.ts — 插图生成串行队列：一次只跑一张、位次可查、
// 失败不阻断后续（进度归属可信的前提）。
import { beforeEach, describe, expect, it } from 'vitest'
import {
  __resetIllustrationQueueForTest, enqueueIllustration, illustrationQueueSnapshot,
  subscribeIllustrationQueue,
} from './illustrationQueue'

const tick = () => new Promise((r) => setTimeout(r, 0))

beforeEach(() => {
  __resetIllustrationQueueForTest()
})

describe('illustrationQueue', () => {
  it('串行执行：第二个任务在第一个结束前不启动', async () => {
    const started: string[] = []
    let releaseA: (v: string) => void = () => {}
    const a = enqueueIllustration(() => {
      started.push('a')
      return new Promise<string>((res) => { releaseA = res })
    })
    const b = enqueueIllustration(async () => {
      started.push('b')
      return 'b-done'
    })
    await tick()
    expect(started).toEqual(['a'])           // b 还在排队
    expect(illustrationQueueSnapshot().waiting).toBe(1)
    expect(illustrationQueueSnapshot().positionOf(b.token)).toBe(1) // b = 下一个
    releaseA('a-done')
    await expect(a.promise).resolves.toBe('a-done')
    await expect(b.promise).resolves.toBe('b-done')
    expect(started).toEqual(['a', 'b'])      // b 在 a 结束后才启动
    expect(illustrationQueueSnapshot().running).toBe(false)
    expect(illustrationQueueSnapshot().waiting).toBe(0)
  })

  it('失败不阻断队列：第一个抛错，第二个照常跑', async () => {
    const a = enqueueIllustration(async () => { throw new Error('boom') })
    const b = enqueueIllustration(async () => 'ok')
    await expect(a.promise).rejects.toThrow('boom')
    await expect(b.promise).resolves.toBe('ok')
    expect(illustrationQueueSnapshot().running).toBe(false)
  })

  it('订阅者在入队/开跑/收尾时收到通知（用于刷新排队位次）', async () => {
    let calls = 0
    const unsub = subscribeIllustrationQueue(() => { calls += 1 })
    const { promise } = enqueueIllustration(async () => 'x')
    await promise
    await tick() // 让 finally 里的收尾通知跑完（resolve 先于 finally）
    unsub()
    expect(calls).toBeGreaterThanOrEqual(2)
  })
})
