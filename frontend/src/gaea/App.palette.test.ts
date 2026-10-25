import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

/**
 * App.palette.test.ts — 办公域「进度计划」命令面板入口回归锁。
 * v4.168.0 刀1b 立锁（源级断言，App 巨组件不整页渲染，App.export.test.ts 先例）；
 * 2026-10-25 工具化改口径：run 从派发 NAVIGATE(page=schedule) 跳板块改为
 * setActiveTool("schedule") 开 ToolSheet 工具面板（不离开对话；
 * docs/gaea-ai-workbench-ui-2026-10.md §7.4——进度是动词不是房间；完整编制
 * 工作台经工具面板头部「完整工作台」深链接，板块路由保留）。
 */
const src = readFileSync(join(process.cwd(), 'src/gaea/App.tsx'), 'utf8')

describe('办公域「进度计划」命令面板入口', () => {
  it('cmd-schedule 命令项存在：组/标题/图标/关键词齐备', () => {
    const cmdLine = src.split('\n').find((l) => l.includes('id: "cmd-schedule"'))
    expect(cmdLine).toBeTruthy()
    expect(cmdLine).toContain('t("palette.group.commands") ?? "命令"')
    expect(cmdLine).toContain('t("schedCard.tag")')
    expect(cmdLine).toContain('<LineChart size={15} />')
    expect(cmdLine).toContain('compact: true')
    for (const kw of ['"schedule"', '"进度"', '"计划"', '"工具"']) {
      expect(cmdLine).toContain(kw)
    }
  })

  it('cmd-schedule 的 run 开进度工具面板：setActiveTool("schedule")，不再跳板块', () => {
    const cmdLine = src.split('\n').find((l) => l.includes('id: "cmd-schedule"'))
    expect(cmdLine).toBeTruthy()
    expect(cmdLine).toContain('setActiveTool("schedule")')
    expect(cmdLine).not.toContain('NAVIGATE')
  })

  it('App 已从 ../events 引入 FRONTEND_EVENTS（OPEN_TOOL 监听用，模块级，无需 dep）', () => {
    expect(src).toMatch(/import \{ FRONTEND_EVENTS \} from "\.\.\/events"/)
  })

  it('深处组件打开工具面板的单通道存在：OPEN_TOOL 监听 → setActiveTool', () => {
    expect(src).toMatch(/FRONTEND_EVENTS\.OPEN_TOOL/)
    expect(src).toMatch(/activeTool, setActiveTool\] = useState<"cad" \| "schedule" \| null>/)
  })
})
