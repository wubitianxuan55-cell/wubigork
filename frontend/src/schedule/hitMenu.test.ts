/**
 * hitMenu.test.ts — 命中区 → 菜单条目的纯逻辑用例
 *
 * 口径来源：`进度计划/gaea-schedule-zpert8-uiux-port-plan-20261114.md` §W5
 * 「右键菜单按命中类型追加，不写死整份 QMenu 顺序」；逆向侧是 uiux-re §15.3
 * `FUN_14053e5e0(doc, hitCtx, hitType)` 的 `switch (hitType)` 追加器。
 * 这里只断言**数据**：哪些项出现、什么顺序、什么上下文禁用、原因文案是否与
 * 命令注册表同一口径（不碰 DOM，不碰 antd）。
 */
import { describe, expect, it } from 'vitest'
import {
  HIT_STRUCTURE_ITEMS,
  HIT_STRUCTURE_KEYS,
  canvasMenu,
  columnHeaderMenu,
  menuForHit,
  splitterMenu,
  statusBarMenu,
  structureMenu,
  type HitMenuContext,
  type HitMenuItem,
} from './hitMenu'
import { scheduleCommand } from './commandRegistry'

/** 缺省上下文：有选中、选中叶行、表里 3 行、选中首个 */
function ctx(patch: Partial<HitMenuContext> = {}): HitMenuContext {
  return {
    hasSelection: true,
    selectedIsGroup: false,
    selectedIndex: 0,
    taskCount: 3,
    hasClipboard: false,
    groupBeforeSelected: false,
    hasGroupContext: false,
    selectedGroupCollapsed: false,
    column: null,
    ...patch,
  }
}

function byKey(items: HitMenuItem[], key: string): HitMenuItem {
  const found = items.find((item) => item.key === key)
  if (!found) throw new Error(`菜单无 ${key}`)
  return found
}

describe('结构操作 12 项（行头 / 单元格）', () => {
  it('固定序 12 项：添加工作→升级→降级→上移→下移→成组→解组→复制→粘贴→格式刷→删除→文字颜色', () => {
    const items = structureMenu(ctx())
    expect(items.slice(0, 12).map((item) => item.key)).toEqual([...HIT_STRUCTURE_KEYS])
    expect(items.slice(0, 12).map((item) => item.label)).toEqual([
      '添加工作',
      '升级',
      '降级',
      '上移',
      '下移',
      '成组',
      '解组',
      '复制',
      '粘贴',
      '格式刷',
      '删除',
      '文字颜色',
    ])
    // 首项恒为「添加工作」（斑马对象菜单里“加工作”永远在最前）
    expect(items[0].key).toBe('add-work')
    expect(items[0].enabled).toBe(true)
  })

  it('每一项的动作键都在命令注册表里有落地口径（不许出现野键）', () => {
    for (const spec of HIT_STRUCTURE_ITEMS) {
      if (!spec.command) continue
      expect(scheduleCommand(spec.command), `${spec.key} → ${spec.command}`).toBeDefined()
    }
    // 无注册表命令的三项：现在都是真动作（解组/格式刷/文字颜色）
    // 解组只对分组行可用；格式刷与文字颜色只对叶行可用——上下文不同，分开断言
    const groupItems = structureMenu(ctx({ selectedIsGroup: true, groupBeforeSelected: true }))
    expect(byKey(groupItems, 'ungroup').enabled).toBe(true)
    expect(byKey(groupItems, 'ungroup').disabledReason).toBeUndefined()
    const leafItems = structureMenu(ctx())
    expect(byKey(leafItems, 'ungroup').enabled).toBe(false)
    expect(byKey(leafItems, 'ungroup').disabledReason).toContain('只有分组行可以解组')
    for (const key of ['brush', 'text-color']) {
      expect(byKey(leafItems, key).enabled, `${key} 应为真动作`).toBe(true)
      expect(byKey(leafItems, key).disabledReason).toBeUndefined()
    }
    // 文字颜色是色板子菜单（6 项，含「默认」清色项）
    const palette = byKey(leafItems, 'text-color').children ?? []
    expect(palette.map((c) => c.key)).toEqual([
      'text-color.red', 'text-color.orange', 'text-color.green',
      'text-color.blue', 'text-color.purple', 'text-color.default',
    ])
  })

  it('无选中：升级/降级/上移/下移/复制/删除 全禁用，原因沿用命令注册表原话', () => {
    const items = structureMenu(ctx({ hasSelection: false, selectedIndex: -1 }))
    expect(byKey(items, 'promote').disabledReason).toBe('请先选中一条工作')
    expect(byKey(items, 'demote').disabledReason).toBe('请先选中一个分组')
    expect(byKey(items, 'up').disabledReason).toBe('请先选中一条任务或分组')
    expect(byKey(items, 'down').disabledReason).toBe('请先选中一条任务或分组')
    expect(byKey(items, 'copy').disabledReason).toBe('请先选中一条任务或分组')
    expect(byKey(items, 'delete').disabledReason).toBe('请先选中一条任务或分组')
    // 不需要选中也能用的两项
    expect(byKey(items, 'add-work').enabled).toBe(true)
    expect(byKey(items, 'group').enabled).toBe(true)
  })

  it('分组行：升级禁用（分组行不支持叶行操作），降级取决于“前面还有没有分组”', () => {
    const noPrev = structureMenu(ctx({ selectedIsGroup: true }))
    expect(byKey(noPrev, 'promote').enabled).toBe(false)
    expect(byKey(noPrev, 'promote').disabledReason).toBe('分组行不支持该操作，请选中一条工作')
    expect(byKey(noPrev, 'demote').enabled).toBe(false)
    expect(byKey(noPrev, 'demote').disabledReason).toBe('降级后无处归属（前面没有分组行）')
    // 前面有分组 → 降级可用（与 store.demoteTask 的归属前提一致）
    const withPrev = structureMenu(ctx({ selectedIsGroup: true, groupBeforeSelected: true }))
    expect(byKey(withPrev, 'demote').enabled).toBe(true)
    expect(byKey(withPrev, 'demote').disabledReason).toBeUndefined()
  })

  it('首行「上移」禁用、末行「下移」禁用（原因=命令注册表的表首/表尾口径）', () => {
    const first = structureMenu(ctx({ selectedIndex: 0, taskCount: 3 }))
    expect(byKey(first, 'up').enabled).toBe(false)
    expect(byKey(first, 'up').disabledReason).toBe('选中项已经在表首')
    expect(byKey(first, 'down').enabled).toBe(true)
    const last = structureMenu(ctx({ selectedIndex: 2, taskCount: 3 }))
    expect(byKey(last, 'down').enabled).toBe(false)
    expect(byKey(last, 'down').disabledReason).toBe('选中项已经在表尾')
    expect(byKey(last, 'up').enabled).toBe(true)
  })

  it('剪贴板：空 → 粘贴禁用（与 store.pasteRows 同一句话）；有内容 → 可用', () => {
    expect(byKey(structureMenu(ctx()), 'paste').disabledReason).toBe('剪贴板没有可粘贴的行')
    expect(byKey(structureMenu(ctx({ hasClipboard: true })), 'paste').enabled).toBe(true)
  })

  it('删除是危险项（danger）；行级专有项跟在 12 项之后且随里程碑状态换文案', () => {
    const items = structureMenu(ctx({ selectedIsMilestone: true }))
    expect(byKey(items, 'delete').danger).toBe(true)
    expect(items.length).toBe(14) // 12 结构 + 里程碑 + 任务检查器
    expect(items[12].key).toBe('milestone')
    expect(items[12].label).toBe('取消里程碑')
    expect(items[13].key).toBe('inspector')
    expect(byKey(structureMenu(ctx()), 'milestone').label).toBe('设为里程碑')
  })

  it('分组行不给里程碑/任务检查器（分组不是工作对象）', () => {
    const items = structureMenu(ctx({ selectedIsGroup: true, groupBeforeSelected: true }))
    expect(byKey(items, 'milestone').enabled).toBe(false)
    expect(byKey(items, 'milestone').disabledReason).toBe('分组行不支持该操作，请选中一条工作')
    expect(byKey(items, 'inspector').enabled).toBe(false)
  })
})

describe('画布 / 列头 / 分隔条 / 状态栏 分区差异', () => {
  it('画布项：无选中时插入类禁用并说明原因；全览可用；分区两项诚实禁用', () => {
    const items = canvasMenu(ctx({ hasSelection: false, selectedIndex: -1 }))
    expect(byKey(items, 'canvas.insert-pre').disabledReason).toBe('请先选中一行作为插入参照')
    expect(byKey(items, 'canvas.insert-post').disabledReason).toBe('请先选中一行作为插入参照')
    expect(byKey(items, 'canvas.add-work').enabled).toBe(true)
    expect(byKey(items, 'canvas.fit').enabled).toBe(true)
    expect(byKey(items, 'canvas.partition').enabled).toBe(false)
    expect(byKey(items, 'canvas.partition').disabledReason).toBe('分区由首层分组自动推导，横道图暂不支持手动标记')
    expect(byKey(items, 'canvas.style').enabled).toBe(false)
    // 画布菜单不含结构 12 项
    expect(items.map((item) => item.key)).not.toContain('add-work')
    expect(items.every((item) => item.group === '画布')).toBe(true)
  })

  it('画布项：选中行是表首时「插入前置工作」禁用；有前置位则可用', () => {
    expect(byKey(canvasMenu(ctx({ selectedIndex: 0 })), 'canvas.insert-pre').disabledReason).toBe(
      '选中行已是表首，没有前置位置',
    )
    expect(byKey(canvasMenu(ctx({ selectedIndex: 1 })), 'canvas.insert-pre').enabled).toBe(true)
  })

  it('画布项：折叠/展开该组按“是否在组内 + 该组是否已折叠”互补', () => {
    const outside = canvasMenu(ctx({ hasGroupContext: false }))
    expect(byKey(outside, 'canvas.collapse-group').disabledReason).toBe(
      '选中行不在分组内（先在行头选中分组或组内工作）',
    )
    expect(byKey(outside, 'canvas.expand-group').enabled).toBe(false)
    const open = canvasMenu(ctx({ hasGroupContext: true, selectedGroupCollapsed: false }))
    expect(byKey(open, 'canvas.collapse-group').enabled).toBe(true)
    expect(byKey(open, 'canvas.expand-group').disabledReason).toBe('该组已展开')
    const closed = canvasMenu(ctx({ hasGroupContext: true, selectedGroupCollapsed: true }))
    expect(byKey(closed, 'canvas.collapse-group').disabledReason).toBe('该组已折叠')
    expect(byKey(closed, 'canvas.expand-group').enabled).toBe(true)
  })

  it('列头项：插入列/列设置/列宽自适应诚实禁用；隐藏该列只在落到可藏列时可用', () => {
    const none = columnHeaderMenu(ctx({ column: null }))
    expect(none.map((item) => item.key)).toEqual(['column.insert', 'column.hide', 'column.settings', 'column.autofit'])
    expect(byKey(none, 'column.hide').disabledReason).toBe('未命中具体列')
    expect(byKey(none, 'column.insert').enabled).toBe(false)
    const dur = columnHeaderMenu(ctx({ column: { key: 'dur', label: '工期(工日)' } }))
    expect(byKey(dur, 'column.hide').enabled).toBe(true)
    const fixed = columnHeaderMenu(ctx({ column: { key: 'name', label: '工作名称' } }))
    expect(byKey(fixed, 'column.hide').disabledReason).toBe('行号/名称列固定不隐藏（表格最小可用集）')
  })

  it('分隔条项：复位/收纳两项都是真动作，顺序固定', () => {
    const items = splitterMenu()
    expect(items.map((item) => item.key)).toEqual(['splitter.reset', 'splitter.collapse'])
    expect(items.every((item) => item.enabled && item.group === '分隔条')).toBe(true)
  })

  it('状态栏项：只有说明性条目且如实禁用（不假装能改统计口径）', () => {
    const items = statusBarMenu()
    expect(items.map((item) => item.key)).toEqual(['status.metric-help', 'status.copy-metrics'])
    expect(items.every((item) => !item.enabled && item.disabledReason)).toBe(true)
  })
})

describe('menuForHit 分发', () => {
  it('行头与单元格同构（同一份结构 12 项）；画布/列头/分隔条/状态栏各不相同', () => {
    const cell = menuForHit('cell', ctx())
    const rowHeader = menuForHit('row-header', ctx())
    expect(rowHeader.map((item) => item.key)).toEqual(cell.map((item) => item.key))
    const canvas = menuForHit('canvas', ctx()).map((item) => item.key)
    const column = menuForHit('column-header', ctx()).map((item) => item.key)
    const split = menuForHit('splitter', ctx()).map((item) => item.key)
    const status = menuForHit('status-bar', ctx()).map((item) => item.key)
    expect(new Set([canvas[0], column[0], split[0], status[0]]).size).toBe(4)
  })

  it('未知命中区 / null（真空白处）→ 空数组（不兜底成某个默认菜单）', () => {
    expect(menuForHit(null, ctx())).toEqual([])
    expect(menuForHit(undefined, ctx())).toEqual([])
    // 运行期可能收到脏值（属性是外部可改的）：一律当“没有菜单”
    expect(menuForHit('nope' as unknown as 'cell', ctx())).toEqual([])
  })
})
