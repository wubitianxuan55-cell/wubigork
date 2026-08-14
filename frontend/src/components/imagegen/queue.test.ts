import { describe, expect, it } from 'vitest'
import { afterTaskStatus, markQueueCanceled, shouldSubmitNext } from './queue'
import type { GenTask, QueueEntry } from './types'

const task = (prompt: string): GenTask => ({
  prompt, negative: '', size: '1024x1024', customWidth: 1024, customHeight: 1024,
  model: 'krea2', seed: 1, count: 1, selectedLoras: [], mode: 'txt2img',
  initImage: '', denoise: 0.65, frames: 97, fps: 8,
})

const entry = (id: number, status: QueueEntry['status']): QueueEntry => ({
  id, task: task(`p${id}`), status,
})

describe('markQueueCanceled', () => {
  it('marks running and pending items canceled, leaves done/failed untouched', () => {
    const out = markQueueCanceled([
      entry(1, 'running'), entry(2, 'pending'), entry(3, 'done'), entry(4, 'failed'),
    ])
    expect(out.map((e) => `${e.id}:${e.status}`)).toEqual([
      '1:canceled', '2:canceled', '3:done', '4:failed',
    ])
  })

  it('is idempotent for already canceled items', () => {
    const out = markQueueCanceled([entry(1, 'canceled')])
    expect(out[0].status).toBe('canceled')
  })
})

describe('shouldSubmitNext（取消后队列停止提交）', () => {
  it('submits while there are pending items and cancel is not armed', () => {
    expect(shouldSubmitNext(1, false)).toBe(true)
    expect(shouldSubmitNext(3, false)).toBe(true)
  })

  it('stops submitting after cancel until a new round', () => {
    expect(shouldSubmitNext(3, true)).toBe(false)
    expect(shouldSubmitNext(1, true)).toBe(false)
    expect(shouldSubmitNext(0, false)).toBe(false)
  })
})

describe('mock 队列流程：取消后不再启动排队项', () => {
  it('interrupts the draining loop mid-flight and skips queued items', async () => {
    const tasks: string[] = ['a', 'b', 'c']
    const submitted: string[] = []
    let cancelArmed = false
    let releaseRun: () => void = () => {}
    const gate = new Promise<void>((res) => { releaseRun = res })
    const submit = async (t: string) => { submitted.push(t); if (t === 'a') await gate }

    // 模拟页面 processQueue 循环：条件与取消标记与页面一致
    const drain = async () => {
      while (shouldSubmitNext(tasks.length, cancelArmed)) {
        const next = tasks.shift()!
        await submit(next)
      }
    }

    const p = drain()
    await Promise.resolve() // 循环启动，进入 submit('a') 等待 gate
    cancelArmed = true      // 用户点击取消：本地取消标记立即生效
    releaseRun()
    await p

    expect(submitted).toEqual(['a']) // 只执行了进行中的 a，排队 b/c 不再启动
    expect(tasks).toEqual(['b', 'c'])
  })

  it('resumes submitting on a new round (cancelArmed cleared by user enqueue)', async () => {
    const tasks: string[] = ['x']
    const submitted: string[] = []
    let cancelArmed = false
    const drain = async () => {
      while (shouldSubmitNext(tasks.length, cancelArmed)) {
        const next = tasks.shift()!
        submitted.push(next)
      }
    }
    // 上一轮取消后保持 armed：取消同时清空待执行队列（页面 pendingRef.current = []），无提交
    cancelArmed = true
    tasks.length = 0
    await drain()
    expect(submitted).toEqual([])
    // 用户手动发起新一轮生成：enqueueTask 清除标记 → 队列恢复提交
    cancelArmed = false
    tasks.push('y')
    await drain()
    expect(submitted).toEqual(['y'])
  })
})

describe('afterTaskStatus', () => {
  it('reports canceled when cancel armed, otherwise by result', () => {
    expect(afterTaskStatus(true, false)).toBe('canceled')
    expect(afterTaskStatus(true, true)).toBe('canceled')
    expect(afterTaskStatus(false, true)).toBe('done')
    expect(afterTaskStatus(false, false)).toBe('failed')
  })
})
