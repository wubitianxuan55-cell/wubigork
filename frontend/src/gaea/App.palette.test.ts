import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

/**
 * App.palette.test.ts — 办公域「进度计划」命令面板入口（v4.168.0 刀1b）回归锁。
 * App 是巨组件（App.export.test.ts 先例：不做整页渲染），以源级断言锁定：
 * cmd-schedule 的组/标题/关键词/图标齐备、run 派发 NAVIGATE(page=schedule)
 * （与 ScheduleFileCard.tsx:67 同一事件通道；emitFrontendEvent 为 src/events.ts
 * 模块级函数，不参与 useMemo 依赖数组）。
 */
const src = readFileSync(join(process.cwd(), 'src/gaea/App.tsx'), 'utf8')

describe('办公域「进度计划」命令面板入口（刀1b）', () => {
  it('cmd-schedule 命令项存在：组/标题/图标/关键词齐备', () => {
    const cmdLine = src.split('\n').find((l) => l.includes('id: "cmd-schedule"'))
    expect(cmdLine).toBeTruthy()
    expect(cmdLine).toContain('t("palette.group.commands") ?? "命令"')
    expect(cmdLine).toContain('t("schedCard.tag")')
    expect(cmdLine).toContain('<LineChart size={15} />')
    expect(cmdLine).toContain('compact: true')
    for (const kw of ['"schedule"', '"进度"', '"计划"', '"工作台"']) {
      expect(cmdLine).toContain(kw)
    }
  })

  it('cmd-schedule 的 run 经 emitFrontendEvent 派发 NAVIGATE(page=schedule)', () => {
    const cmdLine = src.split('\n').find((l) => l.includes('id: "cmd-schedule"'))
    expect(cmdLine).toBeTruthy()
    expect(cmdLine).toContain('emitFrontendEvent(FRONTEND_EVENTS.NAVIGATE, { page: "schedule" })')
  })

  it('App 已从 ../events 引入 emitFrontendEvent/FRONTEND_EVENTS（模块级，无需 dep）', () => {
    expect(src).toMatch(/import \{ emitFrontendEvent, FRONTEND_EVENTS \} from "\.\.\/events"/)
  })
})