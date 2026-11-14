import { describe, expect, it } from 'vitest'
import {
  SCHEDULE_COMMANDS,
  commandForShortcut,
  formatShortcut,
  scheduleCommand,
  scheduleCommandDisabledReason,
  searchScheduleCommands,
  shortcutFromEvent,
} from './commandRegistry'

const KEY = { ctrlKey: false, altKey: false, shiftKey: false, metaKey: false }

describe('schedule command registry', () => {
  it('命令 ID 稳定唯一，且关键命令可反查', () => {
    expect(SCHEDULE_COMMANDS).toHaveLength(44)
    expect(new Set(SCHEDULE_COMMANDS.map((command) => command.id)).size).toBe(SCHEDULE_COMMANDS.length)
    expect(scheduleCommand('edit.find')?.label).toBe('查找工作')
    expect(scheduleCommand('edit.column-filter')?.label).toBe('列筛选')
    expect(scheduleCommand('missing')).toBeUndefined()
  })

  it('搜索按标签前缀优先，并覆盖中英文关键词', () => {
    expect(searchScheduleCommands('查找')[0]?.id).toBe('edit.find')
    expect(searchScheduleCommands('gantt')[0]?.id).toBe('view.gantt')
    expect(searchScheduleCommands('不存在')).toEqual([])
  })

  it('快捷键规范化支持 Ctrl/Meta、方向键和大小写', () => {
    expect(shortcutFromEvent({ ...KEY, ctrlKey: true, key: 'K' })).toBe('ctrl+k')
    expect(shortcutFromEvent({ ...KEY, metaKey: true, shiftKey: true, key: 'ArrowLeft' })).toBe('ctrl+shift+arrowleft')
    expect(commandForShortcut({ ...KEY, key: 'Delete' })?.id).toBe('edit.delete')
    expect(commandForShortcut({ ...KEY, altKey: true, shiftKey: true, key: 'ArrowRight' })?.id).toBe('outline.demote')
    expect(formatShortcut('ctrl+shift+arrowleft')).toBe('Ctrl+Shift+←')
  })

  it('同一可用性函数产出统一禁用原因', () => {
    const base = {
      hasSelection: false,
      selectedIsGroup: false,
      selectedIndex: -1,
      taskCount: 3,
      canUndo: false,
      canRedo: false,
    }
    expect(scheduleCommandDisabledReason('edit.undo', base)).toBe('没有可撤销的修改')
    expect(scheduleCommandDisabledReason('task.milestone', base)).toBe('请先选中一条工作')
    expect(scheduleCommandDisabledReason('task.milestone', {
      ...base, hasSelection: true, selectedIsGroup: true, selectedIndex: 0,
    })).toContain('分组行不支持')
    expect(scheduleCommandDisabledReason('outline.down', {
      ...base, hasSelection: true, selectedIndex: 2,
    })).toBe('选中项已经在表尾')
  })
})
