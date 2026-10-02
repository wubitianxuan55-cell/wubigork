/**
 * paths.test.ts — FE7-09：缺省计划 rel 单源锁。
 *
 * 三处消费点（schedule/gschedSummary.ts、schedule/store.ts、gaea/lib/changes.ts）
 * 必须取到同一值：任一处退回自写字面量并发生漂移，或 paths.ts 被塞进 import
 * 重新制造模块环风险，本用例即 FAIL。
 */
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { SCHEDULE_DEFAULT_REL } from './paths'
import { SCHEDULE_FILE_PATH } from './gschedSummary'
import { DEFAULT_SCHEDULE_PATH } from './store'
import { extractChangedPaths } from '../gaea/lib/changes'

describe('schedule/paths 缺省计划 rel 单源（FE7-09）', () => {
  it('三处消费点取到同一值，且与 Go DefaultRelPath 同字面量', () => {
    // 消费点 1：gschedSummary（办公卡 isCurrentPlan / 页头状态栏回落值）
    expect(SCHEDULE_FILE_PATH).toBe(SCHEDULE_DEFAULT_REL)
    // 消费点 2：store（currentPath 初值）
    expect(DEFAULT_SCHEDULE_PATH).toBe(SCHEDULE_DEFAULT_REL)
    // 消费点 3：changes.ts 不导出常量——按行为断言：schedule_apply 未带显式
    // path 时，聚合出的目标 rel 必须正是缺省 rel。
    expect(extractChangedPaths('{}', 'schedule_apply')).toEqual([SCHEDULE_DEFAULT_REL])
    // 字面量锁：与 internal/schedule/project.go DefaultRelPath 同值（正斜杠 rel）。
    expect(SCHEDULE_DEFAULT_REL).toBe('进度计划/当前计划.gsched.json')
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
