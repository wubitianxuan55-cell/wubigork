// sin/illustrationQueue.ts — 原罪插图生成串行队列（模块级单例，按板块共用）。
//
// 为什么必须串行：图像后端（尤其 ComfyUI）一次只跑一个任务，进度上报也是
// 单任务维度——并发提交会让「百分比/当前节点」失去归属，用户看到两条进度条
// 同时跳同一个百分比。串行后：一次只有一张在跑（进度可信），其余显示「排队中
// · 前面还有 N 张」。
//
// 纪律：单个任务失败不阻断队列（后续照常跑）；订阅者只拿到快照，不暴露内部队列。

export interface IllustrationQueueSnapshot {
  /** 是否有任务在跑。 */
  running: boolean
  /** 等待中的任务数（不含在跑那一个）。 */
  waiting: number
  /** 本任务在队列中的位次（1 = 下一个跑；0 = 在跑或已结束）。 */
  positionOf: (token: number) => number
}

interface Job {
  token: number
  run: () => Promise<unknown>
  resolve: (v: unknown) => void
  reject: (e: unknown) => void
}

let seq = 0
let running = false
const queue: Job[] = []
const listeners = new Set<() => void>()

function notify() {
  for (const cb of listeners) cb()
}

function drain() {
  if (running) return
  const job = queue.shift()
  if (!job) {
    notify()
    return
  }
  running = true
  notify()
  job.run()
    .then((v) => job.resolve(v))
    .catch((e) => job.reject(e))
    .finally(() => {
      running = false
      drain()
    })
}

/**
 * 入队一个插图生成任务，返回 { promise, token }。
 * token 用于查询自己在队列中的位次（配合 subscribeIllustrationQueue 重渲染）。
 */
export function enqueueIllustration<T>(run: () => Promise<T>): { promise: Promise<T>; token: number } {
  const token = ++seq
  const promise = new Promise<T>((resolve, reject) => {
    queue.push({ token, run, resolve: resolve as (v: unknown) => void, reject })
  })
  drain()
  return { promise, token }
}

/** 订阅队列变化（在跑状态/等待数；组件用它重算自己的位次）。 */
export function subscribeIllustrationQueue(cb: () => void): () => void {
  listeners.add(cb)
  return () => { listeners.delete(cb) }
}

/** 队列快照：waiting = 等待数；positionOf(token) = 该任务位次（1 起，0 = 不在等待队列）。 */
export function illustrationQueueSnapshot(): IllustrationQueueSnapshot {
  const order = queue.map((j) => j.token)
  return {
    running,
    waiting: order.length,
    positionOf: (token: number) => {
      const idx = order.indexOf(token)
      return idx < 0 ? 0 : idx + 1
    },
  }
}

/** 测试隔离：清空队列状态（生产不使用）。 */
export function __resetIllustrationQueueForTest(): void {
  queue.length = 0
  running = false
  seq = 0
  listeners.clear()
}
