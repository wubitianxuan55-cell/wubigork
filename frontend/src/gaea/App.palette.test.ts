import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

/**
 * App.palette.test.ts — 办公域「进度计划/CAD 图纸」命令面板入口回归锁。
 * v4.168.0 刀1b 立锁（源级断言，App 巨组件不整页渲染，App.export.test.ts 先例）；
 * 2026-10-25 工具化改口径（ToolSheet 显式命令项）；2026-11-14 用户拍板 CAD 图纸/
 * 进度计划回归右栏一级 Tab——显式 cmd-schedule/cmd-tool-cad 退役，两入口由
 * SIDEBAR_REGISTRY 循环统一派生（cmd-cad/cmd-schedule 自带未启用标注与点击即
 * 启用），深处组件 OPEN_TOOL 单通道与首页 tile 冷启补开改走 openToolTab。
 */
const src = readFileSync(join(process.cwd(), 'src/gaea/App.tsx'), 'utf8')

describe('办公域「进度计划/CAD 图纸」命令面板入口（2026-11-14 右栏一级 Tab）', () => {
  it('命令面板经 SIDEBAR_REGISTRY 循环派生右栏命令项（cad/schedule 同权，不再单列）', () => {
    expect(src).toMatch(/for \(const reg of SIDEBAR_REGISTRY\)/)
    expect(src).toContain('`cmd-${reg.id}`')
    // ToolSheet 时代的显式命令项已退役
    expect(src).not.toContain('id: "cmd-schedule"')
    expect(src).not.toContain('id: "cmd-tool-cad"')
  })

  it('派生命令项语义=未启用先启用再开视图 tab（可选卡显式检索=用户意图）', () => {
    // 循环体截取锚定缩进闭合行（模板字面量含 `}`，不能按首个大括号切）
    const loopBody = src.split('for (const reg of SIDEBAR_REGISTRY) {')[1]?.split('\n    }\n')[0] ?? ''
    expect(loopBody).toContain('setPaneEnabled(reg.id, true)')
    expect(loopBody).toContain('openPaneView(reg.id)')
    expect(loopBody).toContain('未启用')
  })

  it('openToolTab 统一落点：停用先启用 + openPaneView（ToolSheet 退役）', () => {
    expect(src).toMatch(/const openToolTab = useCallback/)
    const body = src.split('const openToolTab = useCallback')[1]?.split('}, [openPaneView')[0] ?? ''
    expect(body).toContain('setPaneEnabled(tool, true)')
    expect(body).toContain('openPaneView(tool)')
    expect(src).not.toContain('setActiveTool')
    // ToolSheet 组件本体已删：App 不再有 import/JSX 引用（史注里的「ToolSheet」
    // 字样属注释，不算接线残留）
    expect(src).not.toMatch(/import \{ ToolSheet/)
    expect(src).not.toMatch(/<ToolSheet/)
  })

  it('App 已从 ../events 引入 FRONTEND_EVENTS + OPEN_TOOL_FLAG（§7.5 刀6 冷启补开用）', () => {
    expect(src).toMatch(/import \{ FRONTEND_EVENTS, OPEN_TOOL_FLAG \} from "\.\.\/events"/)
  })

  it('深处组件打开工具面板的单通道存在：OPEN_TOOL 监听 → openToolTab', () => {
    expect(src).toMatch(/FRONTEND_EVENTS\.OPEN_TOOL/)
    const listenerBody = src.split('const onOpenTool = (ev: Event)')[1]?.split('};')[0] ?? ''
    expect(listenerBody).toContain('openToolTab(tool)')
  })

  it('首页工具 tile 的冷启补开双通道：挂载效应消费 OPEN_TOOL_FLAG 标记（读后即清）+ 监听顺带清残留（§7.5 刀6）', () => {
    // 挂载效应：读标记 → removeItem → openToolTab（办公 App 首次懒加载时事件已错过）
    expect(src).toMatch(/sessionStorage\.getItem\(OPEN_TOOL_FLAG\)/)
    expect(src).toMatch(/sessionStorage\.removeItem\(OPEN_TOOL_FLAG\);/)
    expect(src).toMatch(/openToolTab\(pending\)/)
    // 已挂载场景：事件监听里也清标记，杜绝已消费标记残留到下次冷启误开
    const listenerBody = src.split('const onOpenTool')[1]?.split('}, [openToolTab]);')[0] ?? ''
    expect(listenerBody).toContain('sessionStorage.removeItem(OPEN_TOOL_FLAG)')
  })

  it('右栏上下文注入 onAsk：CAD 问图纸/进度问计划经面板直发对话（send）', () => {
    expect(src).toMatch(/onAsk: send,/)
  })
})
