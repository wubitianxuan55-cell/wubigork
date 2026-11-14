/**
 * schedule/hitMenu.ts — 命中区 + 上下文 → 菜单条目（ZPert8 W5 剩余项「右键按命中类型追加」）
 *
 * 逆向依据（`进度计划/gaea-schedule-zpert8-uiux-re-20261114.md` §15.3）：
 * 斑马 `FUN_14053e5e0(doc, hitCtx, hitType)` 用 `switch (hitType)` 往**同一个** vector 里
 * `push_back` 菜单项，每项带「启用条件」——条件从文档状态直接读。菜单条目是**数据**，
 * 不是写死的一整份 QMenu 顺序；命中类型不同，追加的条目不同。
 *
 * 移植口径（`...uiux-port-plan-20261114.md` §W5）：「右键菜单按命中类型追加，不写死整份
 * QMenu 顺序」。所以本模块只做一件纯事：
 *
 *     menuForHit(region, ctx) → HitMenuItem[]
 *
 * 不依赖 React / antd 类型（输出是普通对象数组，渲染层自己映射成 `MenuProps['items']`），
 * 因此可以脱离视图单测「哪一项在什么上下文下禁用、为什么禁用」。
 *
 * 禁用原因**复用** `commandRegistry.scheduleCommandDisabledReason`（同一句文案是全仓
 * 唯一出口，功能区、快捷键、命令面板与右键菜单不许各说一套）；注册表没有对应命令的
 * 条目（解组/格式刷/文字颜色/列操作/分隔条/状态栏）在本文件里给出同样口吻的原因，
 * 并且**诚实禁用**——没有实现的条目就写明未接入，不做假动作。
 */
import { scheduleCommandDisabledReason, type ScheduleCommandAvailability } from './commandRegistry'
import type { HitRegion } from './hitRegion'

/** 命中区 + 上下文 */
export interface HitMenuContext {
  /** 当前选中行（右键先选中命中行，ZPert §16.2 命中对象 → 选中 + 对象菜单） */
  hasSelection: boolean
  /** 选中行是分组行（level 0） */
  selectedIsGroup: boolean
  /** 选中行在任务表里的序号（-1=无选中） */
  selectedIndex: number
  /** 任务表总行数 */
  taskCount: number
  /** 选中工作是否已是里程碑（决定「设为/取消里程碑」文案） */
  selectedIsMilestone?: boolean
  /** 撤销/重做是否可用（沿用命令注册表口径，菜单本身不出这两项，留给功能区） */
  canUndo?: boolean
  canRedo?: boolean
  /** 本视图剪贴板（复制出的 TSV 行）是否有内容 */
  hasClipboard?: boolean
  /**
   * 选中分组行之前是否还存在别的分组行：`store.demoteTask` 要求「降级后要有归属」，
   * 首个分区降级无处归属，菜单必须先禁用而不是点了才报错。
   */
  groupBeforeSelected?: boolean
  /** 选中行是否落在某个分组内（含选中即组头） */
  hasGroupContext?: boolean
  /** 该组当前是否已折叠（决定「折叠该组 / 展开该组」谁可用） */
  selectedGroupCollapsed?: boolean
  /** 列头命中的具体列（key/label）；null=没落到具体列 */
  column?: { key: string; label: string } | null
}

/** 菜单条目分组（渲染层据此插分隔线；也用于用例断言“同一区固定序”） */
export type HitMenuGroup = '结构' | '任务' | '画布' | '列' | '分隔条' | '状态栏'

/** 一条菜单项描述：key=动作键（同 `commandRegistry` 命令 id 空间）+ 文案 + 启用条件 */
export interface HitMenuItem {
  key: string
  label: string
  group: HitMenuGroup
  enabled: boolean
  /** 禁用原因（同一句文案给 title，用户知道为什么点不动） */
  disabledReason?: string
  /** 危险动作（删除），渲染层走 danger 样式 */
  danger?: boolean
  /** 子菜单（文字颜色色板等）：父项只作分组标题，动作在子项 */
  children?: { key: string; label: string }[]
}

/**
 * 文字颜色色板（斑马「格式 → 字体颜色」的右键等价物，`hitMenu` 不让用户跳功能区）：
 * 键 = `text-color.<色名>`，动作由 GanttView 落到 `SchedTaskStyle.color`。
 * 色值属**内容数据**（用户给任务自选的标注色），不是 UI 主题色，故允许直接写。
 */
export const HIT_TEXT_COLORS: readonly { key: string; label: string; color: string }[] = [
  { key: 'text-color.red', label: '红', color: '#d4380d' }, // hex-exempt：任务标注色
  { key: 'text-color.orange', label: '橙', color: '#d46b08' }, // hex-exempt
  { key: 'text-color.green', label: '绿', color: '#237804' }, // hex-exempt
  { key: 'text-color.blue', label: '蓝', color: '#0958d9' }, // hex-exempt
  { key: 'text-color.purple', label: '紫', color: '#722ed1' }, // hex-exempt
  { key: 'text-color.default', label: '默认（跟随主题）', color: '' }, // 空=清色
]

/** 不可隐藏列（行号/名称）：列头「隐藏该列」对它们禁用（与 `ganttCols.GANTT_FIXED_KEYS` 同口径） */
const FIXED_COL_KEYS: readonly string[] = ['no', 'name']

/** 结构操作 12 项固定序（行头 / 单元格共用；值=动作键 + label + 对应命令 id） */
export const HIT_STRUCTURE_ITEMS: readonly { key: string; label: string; command?: string }[] = [
  { key: 'add-work', label: '添加工作', command: 'task.add' },
  { key: 'promote', label: '升级', command: 'outline.promote' },
  { key: 'demote', label: '降级', command: 'outline.demote' },
  { key: 'up', label: '上移', command: 'outline.up' },
  { key: 'down', label: '下移', command: 'outline.down' },
  { key: 'group', label: '成组', command: 'task.group' },
  { key: 'ungroup', label: '解组' },
  { key: 'copy', label: '复制', command: 'edit.copy' },
  { key: 'paste', label: '粘贴', command: 'edit.paste' },
  { key: 'brush', label: '格式刷' },
  { key: 'delete', label: '删除', command: 'edit.delete' },
  { key: 'text-color', label: '文字颜色' },
]

/** 12 项动作键序（用例与渲染层共用的稳定契约） */
export const HIT_STRUCTURE_KEYS: readonly string[] = HIT_STRUCTURE_ITEMS.map((item) => item.key)

/** 上下文 → 命令注册表的可用性入参（一份口径，别在菜单里另造一套判断） */
function availabilityOf(ctx: HitMenuContext): ScheduleCommandAvailability {
  return {
    hasSelection: ctx.hasSelection,
    selectedIsGroup: ctx.selectedIsGroup,
    selectedIndex: ctx.selectedIndex,
    taskCount: ctx.taskCount,
    canUndo: ctx.canUndo ?? false,
    canRedo: ctx.canRedo ?? false,
  }
}

/** 注册表原因（无原因=可用） */
function fromRegistry(key: string, ctx: HitMenuContext, label: string, command: string): HitMenuItem {
  const reason = scheduleCommandDisabledReason(command, availabilityOf(ctx))
  return reason
    ? { key, label, group: '结构', enabled: false, disabledReason: reason }
    : { key, label, group: '结构', enabled: true }
}

/** 「只有叶行（工作）」类条目的统一原因（与注册表 LEAF_SELECTION_COMMANDS 同一句） */
function leafOnlyReason(ctx: HitMenuContext): string | undefined {
  if (!ctx.hasSelection) return '请先选中一条工作'
  if (ctx.selectedIsGroup) return '分组行不支持该操作，请选中一条工作'
  return undefined
}

/**
 * 结构操作 12 项（行头/单元格）：固定序，逐项按上下文给 enabled + 原因。
 * 尾部再追加两项**行级**任务动作（里程碑/任务检查器）——斑马的对象菜单同样是
 * 「公共结构项 + 该对象专有项」，这里保留既有右键能力（老用例与老用户零回归）。
 */
export function structureMenu(ctx: HitMenuContext): HitMenuItem[] {
  const items: HitMenuItem[] = HIT_STRUCTURE_ITEMS.map((item) => {
    switch (item.key) {
      case 'demote': {
        // 注册表只答“有没有选中分组”；「首个分区无处归属」是 store 的落库前提，
        // 菜单先拦下来，避免点了才弹失败。
        const base = fromRegistry(item.key, ctx, item.label, 'outline.demote')
        if (!base.enabled) return base
        if (!ctx.groupBeforeSelected) {
          return { ...base, enabled: false, disabledReason: '降级后无处归属（前面没有分组行）' }
        }
        return base
      }
      case 'ungroup': {
        if (!ctx.hasSelection) {
          return { key: item.key, label: item.label, group: '结构', enabled: false, disabledReason: '请先选中一条任务或分组' }
        }
        if (!ctx.selectedIsGroup) {
          return { key: item.key, label: item.label, group: '结构', enabled: false, disabledReason: '只有分组行可以解组（工作行没有下级）' }
        }
        return { key: item.key, label: item.label, group: '结构', enabled: true }
      }
      case 'paste': {
        if (!ctx.hasClipboard) {
          return {
            key: item.key,
            label: item.label,
            group: '结构',
            enabled: false,
            disabledReason: '剪贴板没有可粘贴的行',
          }
        }
        return { key: item.key, label: item.label, group: '结构', enabled: true }
      }
      case 'brush': {
        const reason = leafOnlyReason(ctx)
        return reason
          ? { key: item.key, label: item.label, group: '结构', enabled: false, disabledReason: reason }
          : { key: item.key, label: item.label, group: '结构', enabled: true }
      }
      case 'text-color': {
        const reason = leafOnlyReason(ctx)
        return {
          key: item.key,
          label: item.label,
          group: '结构',
          enabled: !reason,
          ...(reason ? { disabledReason: reason } : {}),
          children: HIT_TEXT_COLORS.map((c) => ({ key: c.key, label: c.label })),
        }
      }
      case 'delete':
        return { ...fromRegistry(item.key, ctx, item.label, 'edit.delete'), danger: true }
      default:
        return item.command
          ? fromRegistry(item.key, ctx, item.label, item.command)
          : { key: item.key, label: item.label, group: '结构', enabled: true }
    }
  })
  // 行级专有项：分组行不给（分组不是工作对象）
  const leafReason = leafOnlyReason(ctx)
  items.push({
    key: 'milestone',
    label: ctx.selectedIsMilestone ? '取消里程碑' : '设为里程碑',
    group: '任务',
    enabled: !leafReason,
    ...(leafReason ? { disabledReason: leafReason } : {}),
  })
  const inspectorReason = scheduleCommandDisabledReason('task.inspector', availabilityOf(ctx))
  items.push({
    key: 'inspector',
    label: '任务检查器',
    group: '任务',
    enabled: !inspectorReason,
    ...(inspectorReason ? { disabledReason: inspectorReason } : {}),
  })
  return items
}

/**
 * 画布（横道图区/时间轴区）：对象级项与图面级项混合。
 * 无实现的三项（分区标记、图面样式浮层）诚实禁用并写明出口，不做假动作。
 */
export function canvasMenu(ctx: HitMenuContext): HitMenuItem[] {
  const hasPrev = ctx.hasSelection && ctx.selectedIndex > 0
  const groupReason = ctx.hasGroupContext
    ? undefined
    : '选中行不在分组内（先在行头选中分组或组内工作）'
  const collapseReason = groupReason ?? (ctx.selectedGroupCollapsed ? '该组已折叠' : undefined)
  const expandReason = groupReason ?? (ctx.selectedGroupCollapsed ? undefined : '该组已展开')
  const selectReason = ctx.hasSelection ? undefined : '请先选中一行作为插入参照'
  return [
    { key: 'canvas.add-work', label: '在此处添加工作', group: '画布', enabled: true },
    {
      key: 'canvas.insert-pre',
      label: '插入前置工作',
      group: '画布',
      enabled: hasPrev,
      ...(selectReason
        ? { disabledReason: selectReason }
        : hasPrev
          ? {}
          : { disabledReason: '选中行已是表首，没有前置位置' }),
    },
    {
      key: 'canvas.insert-post',
      label: '插入后置工作',
      group: '画布',
      enabled: ctx.hasSelection,
      ...(selectReason ? { disabledReason: selectReason } : {}),
    },
    {
      key: 'canvas.collapse-group',
      label: '折叠该组',
      group: '画布',
      enabled: !collapseReason,
      ...(collapseReason ? { disabledReason: collapseReason } : {}),
    },
    {
      key: 'canvas.expand-group',
      label: '展开该组',
      group: '画布',
      enabled: !expandReason,
      ...(expandReason ? { disabledReason: expandReason } : {}),
    },
    {
      key: 'canvas.partition',
      label: '显示为分区',
      group: '画布',
      enabled: false,
      disabledReason: '分区由首层分组自动推导，横道图暂不支持手动标记',
    },
    {
      key: 'canvas.unpartition',
      label: '取消分区',
      group: '画布',
      enabled: false,
      disabledReason: '分区由首层分组自动推导，横道图暂不支持手动标记',
    },
    { key: 'canvas.fit', label: '全览整计划', group: '画布', enabled: true },
    {
      key: 'canvas.style',
      label: '表格与横道样式…',
      group: '画布',
      enabled: false,
      disabledReason: '图面样式面板在功能区「显示」，横道视图内不重复挂浮层',
    },
  ]
}

/** 列头：插入列 / 隐藏该列 / 列设置… / 列宽自适应（可用的只有「隐藏该列」） */
export function columnHeaderMenu(ctx: HitMenuContext): HitMenuItem[] {
  const col = ctx.column ?? null
  const hideReason = !col
    ? '未命中具体列'
    : FIXED_COL_KEYS.includes(col.key)
      ? '行号/名称列固定不隐藏（表格最小可用集）'
      : undefined
  return [
    {
      key: 'column.insert',
      label: '插入列',
      group: '列',
      enabled: false,
      disabledReason: '列集由视图预设与列菜单定义，列头暂不支持直接插列',
    },
    {
      key: 'column.hide',
      label: '隐藏该列',
      group: '列',
      enabled: !hideReason,
      ...(hideReason ? { disabledReason: hideReason } : {}),
    },
    {
      key: 'column.settings',
      label: '列设置…',
      group: '列',
      enabled: false,
      disabledReason: '列设置在工具栏「列」菜单（右键不展开该浮层）',
    },
    {
      key: 'column.autofit',
      label: '列宽自适应',
      group: '列',
      enabled: true,
    },
  ]
}

/** 分隔条：两项都是真动作（复位全列宽 / 收纳到行号+名称） */
export function splitterMenu(): HitMenuItem[] {
  return [
    { key: 'splitter.reset', label: '复位表格宽度', group: '分隔条', enabled: true },
    { key: 'splitter.collapse', label: '收纳表格（只留行号与名称）', group: '分隔条', enabled: true },
  ]
}

/**
 * 状态栏：横道视图里没有状态栏节点（它在进度页底部），这里只保留**说明性**条目并
 * 如实禁用 —— 不假装能改统计口径。
 */
export function statusBarMenu(): HitMenuItem[] {
  return [
    {
      key: 'status.metric-help',
      label: '显示统计口径说明',
      group: '状态栏',
      enabled: false,
      disabledReason: '状态栏在进度页底部，横道视图内只读展示',
    },
    {
      key: 'status.copy-metrics',
      label: '复制统计值',
      group: '状态栏',
      enabled: false,
      disabledReason: '状态栏不属横道视图，此处无处理器（可选中单元格复制）',
    },
  ]
}

/**
 * 唯一入口：命中区 + 上下文 → 菜单条目。
 * 未知区域 / null（真空白处）→ 空数组（渲染层据此不弹菜单，等价斑马 §16.2 的
 * “未命中走空白菜单”里“空白处确实没有对象动作”的那一支）。
 */
export function menuForHit(region: HitRegion | null | undefined, ctx: HitMenuContext): HitMenuItem[] {
  switch (region) {
    case 'row-header':
    case 'cell':
      return structureMenu(ctx)
    case 'canvas':
      return canvasMenu(ctx)
    case 'column-header':
      return columnHeaderMenu(ctx)
    case 'splitter':
      return splitterMenu()
    case 'status-bar':
      return statusBarMenu()
    default:
      return []
  }
}
