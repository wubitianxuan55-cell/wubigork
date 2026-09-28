// events-discipline.test.ts — wails 事件订阅纪律守卫（source-guard 风格，同
// perf-guards.test.ts / App.tokens.test.ts）。
//
// Why（事故纪律，勿删）：`EventsOff(channel)` 只传通道名时会注销该通道上的**全部**
// 监听者。v4.61 给 SubagentThread 加订阅后，它卸载时的 EventsOff 把主对话 store 的
// 订阅连带炸掉——对话标签页从此收不到任何实时事件，直到重开应用（v4.62.2 根修，
// 见 `gaea/lib/wailsEvents.ts` 头注）。仓规因此定死两条：
//   ① 任何订阅都必须经 `subscribeWailsEvent` 唯一入口（它用 EventsOn 的返回值
//      「只摘自己」），不得直接摸 `window.runtime`；
//   ② **绝不允许** `EventsOff('chan')` 这种「整通道全清」形态。
//
// 为什么要有本守卫：v4.421 台账写「三处裸 EventsOff 已改 subscribeWailsEvent」，
// 读者会以为已清零，实际当时仍存 2 处（TTSPlayer / CharacterLibraryPage）——
// 声明与代码脱节正是本条守卫要防的复发形态。功能测试抓不到这类「别人被连坐」
// 的缺陷，只能靠静态扫源码。
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const SRC = resolve(__dirname)

/** 允许调用 EventsOff 的唯一文件：runtimePolyfill 是 EventsOff 的实现本身。 */
const ALLOW_FILES = new Set(['api/runtimePolyfill.ts'])

function walk(dir: string): string[] {
  const out: string[] = []
  for (const name of readdirSync(dir)) {
    const full = join(dir, name)
    if (statSync(full).isDirectory()) out.push(...walk(full))
    else out.push(full)
  }
  return out
}

const prodSources = walk(SRC)
  .filter((f) => /\.(ts|tsx)$/.test(f))
  .filter((f) => !/\.test\.(ts|tsx)$/.test(f))
  .filter((f) => !ALLOW_FILES.has(relative(SRC, f).replace(/\\/g, '/')))

/** 单参 EventsOff('chan')（含 `EventsOff?.('chan')`）= 整通道全清。 */
const BARE_EVENTS_OFF = /EventsOff\s*\??\.\s*\(\s*['"`][^'"`]+['"`]\s*\)/g

/** 订阅入口收敛：生产源码里除 polyfill/唯一入口外，不得直接调 runtime 的 EventsOn。 */
const RAW_EVENTS_ON = /\bruntime\s*\??\.\s*EventsOn\s*\(/g

const ALLOW_EVENTS_ON = new Set([
  'api/runtimePolyfill.ts',
  'gaea/lib/wailsEvents.ts', // 唯一入口本身
])

describe('wails 事件订阅纪律（source-guard）', () => {
  it('生产源码零「整通道全清」EventsOff(chan)（含可选链形态）', () => {
    const hits: string[] = []
    for (const f of prodSources) {
      const src = readFileSync(f, 'utf8')
      const m = src.match(BARE_EVENTS_OFF)
      if (m) hits.push(`${relative(SRC, f).replace(/\\/g, '/')} → ${m.join(' / ')}`)
    }
    expect(hits).toEqual([])
  })

  it('生产源码里 EventsOn 只出现在唯一入口与 polyfill（不得直接摸 window.runtime）', () => {
    const hits: string[] = []
    for (const f of prodSources) {
      const rel = relative(SRC, f).replace(/\\/g, '/')
      if (ALLOW_EVENTS_ON.has(rel)) continue
      const src = readFileSync(f, 'utf8')
      if (RAW_EVENTS_ON.test(src)) hits.push(rel)
      RAW_EVENTS_ON.lastIndex = 0
    }
    expect(hits).toEqual([])
  })
})
