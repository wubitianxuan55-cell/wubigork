/**
 * paths.test.ts — FE7-09：缺省计划 rel 单源锁。
 *
 * 五处消费点必须取到同一值：
 *   1. schedule/gschedSummary.ts  SCHEDULE_FILE_PATH（办公卡 isCurrentPlan / 页头状态栏回落）
 *   2. schedule/store.ts          DEFAULT_SCHEDULE_PATH（currentPath 初值）
 *   3. gaea/lib/changes.ts        行为口径（schedule_apply 缺省 path 的聚合目标）
 *   4. gaea/components/ScheduleApplyRollback.tsx  渲染口径（Journal target 匹配用的回滚目标）
 *   5. gaea/lib/mock/office/schedule.ts           走查态当前工程指针初值
 * 任一处退回自写字面量并发生漂移，或 paths.ts 被塞进 import 重新制造模块环风险，
 * 本用例即 FAIL。
 */
import { createElement } from 'react'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, render, screen } from '@testing-library/react'
import { SCHEDULE_DEFAULT_REL } from './paths'
import { SCHEDULE_FILE_PATH } from './gschedSummary'
import { DEFAULT_SCHEDULE_PATH } from './store'
import { extractChangedPaths } from '../gaea/lib/changes'
import { scheduleMethods } from '../gaea/lib/mock/office/schedule'
import { ScheduleApplyRollback } from '../gaea/components/ScheduleApplyRollback'
import { ToastProvider } from '../gaea/components/Toast'

// 消费点 4 需要 Journal 下行绑定；其余消费点不碰桥接，故只给最小面
// （bridge/proxy 的 app 是转发代理，本文件不测转发本身）。
const mocks = vi.hoisted(() => ({
  journalList: vi.fn(async (_limit: number): Promise<unknown[]> => []),
}))
vi.mock('../gaea/lib/bridge', () => ({
  app: { GaeaJournalList: (limit: number) => mocks.journalList(limit) },
  onEvent: () => () => {},
  onReady: (cb: () => void) => {
    cb()
    return () => {}
  },
}))

afterEach(() => {
  cleanup()
  mocks.journalList.mockReset()
  mocks.journalList.mockResolvedValue([])
})

// 两个历史副本（批次十三前各自自写字面量）；源码级锁：只许 import，不许再写值。
const MIRROR_FILES = [
  ['gaea/components/ScheduleApplyRollback.tsx', '../../schedule/paths'],
  ['gaea/lib/mock/office/schedule.ts', '../../../../schedule/paths'],
] as const

describe('schedule/paths 缺省计划 rel 单源（FE7-09）', () => {
  it('五处消费点取到同一值，且与 Go DefaultRelPath 同字面量', async () => {
    // 消费点 1：gschedSummary（办公卡 isCurrentPlan / 页头状态栏回落值）
    expect(SCHEDULE_FILE_PATH).toBe(SCHEDULE_DEFAULT_REL)
    // 消费点 2：store（currentPath 初值）
    expect(DEFAULT_SCHEDULE_PATH).toBe(SCHEDULE_DEFAULT_REL)
    // 消费点 3：changes.ts 不导出常量——按行为断言：schedule_apply 未带显式
    // path 时，聚合出的目标 rel 必须正是缺省 rel。
    expect(extractChangedPaths('{}', 'schedule_apply')).toEqual([SCHEDULE_DEFAULT_REL])
    // 消费点 5：mock 走查态当前指针（板块 Load/Projects 的 current 来源）。
    await expect(scheduleMethods.ScheduleLoad()).resolves.toMatchObject({ path: SCHEDULE_DEFAULT_REL })
    // 消费点 4：组件在 args 无显式 path 时按缺省 rel 匹配 Journal target——
    // 用「值确实等于缺省 rel 的记录」驱动渲染：组件若漂移成别的字面量，
    // target 对不上记录 → 整行不渲染 → 本断言超时 FAIL。
    mocks.journalList.mockResolvedValue([
      {
        id: 'rec-1', sessionId: 's', space: 'work', turn: 3,
        tool: 'schedule_apply', target: SCHEDULE_DEFAULT_REL,
        beforeSummary: '', afterSummary: '', baselinePath: '/ws/.journal/b1.json',
      },
    ])
    render(
      createElement(ToastProvider, null, createElement(ScheduleApplyRollback, { args: '{}' })),
    )
    expect(await screen.findByTestId('sched-apply-rollback')).toBeTruthy()
    // 字面量锁：与 internal/schedule/project.go DefaultRelPath 同值（正斜杠 rel）。
    expect(SCHEDULE_DEFAULT_REL).toBe('进度计划/当前计划.gsched.json')
  })

  it('第五/第四份副本已并单源：两个消费文件只 import，不再自写 rel 字面量', async () => {
    for (const [file, specifier] of MIRROR_FILES) {
      const src = await readFile(resolve(process.cwd(), 'src', file), 'utf8')
      // 必须从零依赖 paths.ts 取值（转引不产生模块边）
      expect(src).toContain(`import { SCHEDULE_DEFAULT_REL } from "${specifier}"`)
      // 且源码里不得再出现该 rel 字面量——退回自写副本即红
      expect(src).not.toContain('进度计划/当前计划.gsched.json')
    }
  })

  it('paths.ts 零依赖：模块内不出现任何 import（转引它不产生模块边）', async () => {
    // 零依赖是本单源方案成立的前提：一旦 paths 反向 import（尤其 schedule/store
    // 或 gschedSummary），转引方就会重新踩上模块环 + TDZ。此处按源码断言。
    // vitest 里 import.meta.url 不是 file: 协议（Vite dev server 转译），按仓库
    // 约定以 frontend 为 cwd 定位源码。
    const src = await readFile(resolve(process.cwd(), 'src/schedule/paths.ts'), 'utf8')
    expect(src).not.toMatch(/^\s*import[\s{*'"]/m)
    expect(src).not.toMatch(/\brequire\s*\(/)
  })
})
