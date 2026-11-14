/**
 * commandRegistry.ts — 进度板块命令注册表（ZPert W2）
 *
 * 逆向口径：斑马把命令键当稳定标识，功能区、快捷键、埋点和自定义入口都引用
 * 同一份元数据；handler 由各功能在运行期注册，不存在“命令 ID → 函数指针”总表。
 * gaea 这里只保留可搜索元数据与统一禁用原因，实际动作由 SchedulePage 注入。
 */

export type ScheduleCommandGroup = '文件' | '编辑' | '任务' | '大纲' | '视图' | '显示' | '帮助'

export interface ScheduleCommandSpec {
  id: string
  group: ScheduleCommandGroup
  label: string
  detail: string
  keywords: readonly string[]
  shortcuts?: readonly string[]
}

export interface ScheduleCommandAvailability {
  hasSelection: boolean
  selectedIsGroup: boolean
  selectedIndex: number
  taskCount: number
  canUndo: boolean
  canRedo: boolean
}

export interface RuntimeScheduleCommand {
  spec: ScheduleCommandSpec
  disabledReason?: string
  active?: boolean
  run: () => void | Promise<void>
}

export const SCHEDULE_COMMANDS: readonly ScheduleCommandSpec[] = [
  { id: 'file.manage', group: '文件', label: '工程管理', detail: '新建、打开、复制或删除进度工程', keywords: ['新建', '打开', '项目', '工程', 'new', 'open'], shortcuts: ['ctrl+n'] },
  { id: 'file.import-new', group: '文件', label: '导入为新工程', detail: '从 XML、Excel 或 MPP 建立独立工程', keywords: ['导入', 'xml', 'excel', 'mpp', 'import'] },
  { id: 'file.import-xml', group: '文件', label: '导入 XML 替换当前工程', detail: '读取 MS Project XML 并替换当前计划', keywords: ['导入', '替换', 'project', 'xml', 'import'] },
  { id: 'file.import-xlsx', group: '文件', label: '导入 Excel 替换当前工程', detail: '读取上报 Excel 并替换当前计划', keywords: ['导入', '替换', 'excel', 'xlsx', 'import'] },
  { id: 'file.import-mpp', group: '文件', label: '导入 MPP 替换当前工程', detail: '读取 MS Project MPP 并替换当前计划', keywords: ['导入', '替换', 'mpp', 'project', 'import'] },
  { id: 'file.export-xml', group: '文件', label: '导出 MS Project XML', detail: '导出可与 MS Project 互通的 XML', keywords: ['导出', '项目', 'project', 'xml', 'export'] },
  { id: 'file.export-xlsx', group: '文件', label: '导出 Excel', detail: '导出当前进度计划工作表', keywords: ['导出', '表格', 'excel', 'xlsx', 'export'] },
  { id: 'file.export-art', group: '文件', label: '图面导出与打印', detail: '导出 PNG、PDF 或打印当前图面', keywords: ['导出', '打印', 'pdf', 'png', 'print'], shortcuts: ['ctrl+p', 'ctrl+shift+p'] },

  { id: 'edit.undo', group: '编辑', label: '撤销', detail: '撤销最近一次计划修改', keywords: ['撤回', 'undo'], shortcuts: ['ctrl+z'] },
  { id: 'edit.redo', group: '编辑', label: '重做', detail: '恢复最近一次撤销', keywords: ['恢复', 'redo'], shortcuts: ['ctrl+y', 'ctrl+shift+z'] },
  { id: 'edit.copy', group: '编辑', label: '复制选中行', detail: '把名称、工期、起止和前置复制为 TSV', keywords: ['拷贝', 'copy'], shortcuts: ['ctrl+c'] },
  { id: 'edit.paste', group: '编辑', label: '粘贴任务行', detail: '把剪贴板中的 TSV 行追加为工作', keywords: ['粘贴', 'paste'], shortcuts: ['ctrl+v'] },
  { id: 'edit.find', group: '编辑', label: '查找工作', detail: '按名称过滤横道表格', keywords: ['搜索', '筛选', 'find', 'search'], shortcuts: ['ctrl+f'] },
  { id: 'edit.delete', group: '编辑', label: '删除选中项', detail: '删除选中工作或分组及其关系', keywords: ['移除', 'delete', 'remove'], shortcuts: ['delete'] },
  { id: 'edit.clear', group: '编辑', label: '清空全部任务与关系', detail: '清空当前工程内容，可通过撤销恢复', keywords: ['重置', 'clear'] },

  { id: 'task.add', group: '任务', label: '添加工作', detail: '在选中任务附近新增一条工作', keywords: ['新建', '插入', 'task', 'add'] },
  { id: 'task.group', group: '任务', label: '添加分组', detail: '新增一条分区／汇总行', keywords: ['分区', '汇总', 'group'] },
  { id: 'task.relation', group: '任务', label: '编辑前置关系', detail: '编辑选中工作的搭接类型与时距', keywords: ['搭接', '依赖', 'relation', 'predecessor'] },
  { id: 'task.milestone', group: '任务', label: '切换里程碑', detail: '把选中工作设为或取消里程碑', keywords: ['里程碑', 'milestone'] },
  { id: 'task.inspector', group: '任务', label: '打开任务检查器', detail: '检查日期来源、时差、预警与推演', keywords: ['检查', '时差', 'inspector'] },

  { id: 'outline.collapse', group: '大纲', label: '折叠选中分组', detail: '收起选中工作所在的大纲分组', keywords: ['收起', 'collapse'] },
  { id: 'outline.expand', group: '大纲', label: '展开全部大纲', detail: '展开所有分组与细分层', keywords: ['展开', 'expand'] },
  { id: 'outline.promote', group: '大纲', label: '升级为分组', detail: '把选中工作升级为分组行', keywords: ['升级', 'promote'], shortcuts: ['shift+alt+arrowleft'] },
  { id: 'outline.demote', group: '大纲', label: '降级为工作', detail: '把选中分组降级为工作行', keywords: ['降级', 'demote'], shortcuts: ['shift+alt+arrowright'] },
  { id: 'outline.up', group: '大纲', label: '上移一行', detail: '把选中项移动到前一行', keywords: ['移动', 'move', 'up'], shortcuts: ['alt+arrowup'] },
  { id: 'outline.down', group: '大纲', label: '下移一行', detail: '把选中项移动到后一行', keywords: ['移动', 'move', 'down'], shortcuts: ['alt+arrowdown'] },

  { id: 'view.gantt', group: '视图', label: '横道图', detail: '表格与时间轴工作台', keywords: ['甘特', 'gantt', 'timeline'] },
  { id: 'view.pdm', group: '视图', label: '单代号网络图', detail: '按工作节点查看逻辑关系', keywords: ['pdm', '节点', 'network'] },
  { id: 'view.aoa', group: '视图', label: '双代号网络图', detail: '保留当前双代号显示模式并打开网络图', keywords: ['aoa', '网络', 'network'] },
  { id: 'view.timescaled', group: '视图', label: '双代号时标网络图', detail: '横轴按最早时间映射的双代号网络图', keywords: ['aoa', '时标', 'network'] },
  { id: 'view.logic', group: '视图', label: '双代号逻辑网络图', detail: '横轴按拓扑深度排列的双代号网络图', keywords: ['aoa', '逻辑', 'network'] },
  { id: 'view.usage', group: '视图', label: '资源使用', detail: '逐工作日查看资源负载', keywords: ['资源', 'usage'] },
  { id: 'view.dcma', group: '视图', label: '质量体检', detail: '查看 DCMA 计划质量指标', keywords: ['dcma', '检查'] },
  { id: 'view.monte', group: '视图', label: '工期模拟', detail: '查看工期概率区间', keywords: ['蒙特卡洛', 'monte', '模拟'] },
  { id: 'view.evm', group: '视图', label: '挣值分析', detail: '查看进度与成本绩效', keywords: ['evm', '挣值'] },

  { id: 'display.dock', group: '显示', label: '切换右侧检查面板', detail: '显示或隐藏预警与检查发现', keywords: ['预警', '检查', 'dock', 'panel'] },
  { id: 'display.chat', group: '显示', label: '切换 AI 助手', detail: '展开或收起进度 AI 对话', keywords: ['ai', 'chat', '对话'] },
  { id: 'display.smart-layout', group: '显示', label: '智能布图', detail: '清除手动布点并回到自动排布', keywords: ['网络图', '布图', 'layout'] },

  { id: 'help.commands', group: '帮助', label: '命令与快捷键', detail: '搜索命令并查看当前可用快捷键', keywords: ['命令面板', '快捷键', 'command', 'palette', 'shortcut'], shortcuts: ['ctrl+k', 'ctrl+/', 'f1'] },
  { id: 'help.templates', group: '帮助', label: '项目模板', detail: '从当前计划生成项目模板', keywords: ['模板', 'template'] },
  { id: 'help.sample', group: '帮助', label: '载入示例工程', detail: '用示例计划快速体验各视图', keywords: ['示例', '演示', 'sample'] },
]

const COMMAND_BY_ID = new Map(SCHEDULE_COMMANDS.map((command) => [command.id, command]))

export function scheduleCommand(id: string): ScheduleCommandSpec | undefined {
  return COMMAND_BY_ID.get(id)
}

const SELECTION_COMMANDS = new Set([
  'edit.copy',
  'edit.delete',
  'outline.collapse',
])

const LEAF_SELECTION_COMMANDS = new Set([
  'task.relation',
  'task.milestone',
  'task.inspector',
  'outline.promote',
])

/** 命令可用性只有这一份；功能区、菜单、快捷键和命令面板共用同一文案。 */
export function scheduleCommandDisabledReason(
  id: string,
  availability: ScheduleCommandAvailability,
): string | undefined {
  if (id === 'edit.undo' && !availability.canUndo) return '没有可撤销的修改'
  if (id === 'edit.redo' && !availability.canRedo) return '没有可重做的修改'
  if (SELECTION_COMMANDS.has(id) && !availability.hasSelection) return '请先选中一条任务或分组'
  if (LEAF_SELECTION_COMMANDS.has(id) && !availability.hasSelection) return '请先选中一条工作'
  if (LEAF_SELECTION_COMMANDS.has(id) && availability.selectedIsGroup) return '分组行不支持该操作，请选中一条工作'
  if (id === 'outline.demote' && !availability.hasSelection) return '请先选中一个分组'
  if (id === 'outline.demote' && !availability.selectedIsGroup) return '只有分组行可以降级'
  if (id === 'outline.up' && !availability.hasSelection) return '请先选中一条任务或分组'
  if (id === 'outline.down' && !availability.hasSelection) return '请先选中一条任务或分组'
  if (id === 'outline.up' && availability.selectedIndex <= 0) return '选中项已经在表首'
  if (id === 'outline.down' && availability.selectedIndex >= availability.taskCount - 1) return '选中项已经在表尾'
  return undefined
}

function normalizedText(command: ScheduleCommandSpec): string {
  return [command.id, command.group, command.label, command.detail, ...command.keywords].join(' ').toLowerCase()
}

function commandScore(command: ScheduleCommandSpec, query: string): number {
  const label = command.label.toLowerCase()
  const id = command.id.toLowerCase()
  const haystack = normalizedText(command)
  if (label === query) return 0
  if (label.startsWith(query)) return 1
  if (id.startsWith(query)) return 2
  if (haystack.includes(query)) return 3
  return Number.POSITIVE_INFINITY
}

/** 稳定排序：命中优先级优先，同档保持注册表原序。 */
export function searchScheduleCommands(query: string): ScheduleCommandSpec[] {
  const q = query.trim().toLowerCase()
  if (!q) return [...SCHEDULE_COMMANDS]
  return SCHEDULE_COMMANDS
    .map((command) => ({ command, score: commandScore(command, q) }))
    .filter((entry) => Number.isFinite(entry.score))
    .sort((a, b) => a.score - b.score)
    .map((entry) => entry.command)
}

export interface ShortcutEventLike {
  key: string
  ctrlKey: boolean
  altKey: boolean
  shiftKey: boolean
  metaKey: boolean
}

function normalizedKey(key: string): string {
  if (key === ' ') return 'space'
  if (key === 'Esc') return 'escape'
  return key.toLowerCase()
}

export function shortcutFromEvent(event: ShortcutEventLike): string {
  const parts: string[] = []
  if (event.ctrlKey || event.metaKey) parts.push('ctrl')
  if (event.altKey) parts.push('alt')
  if (event.shiftKey) parts.push('shift')
  parts.push(normalizedKey(event.key))
  return parts.join('+')
}

export function formatShortcut(shortcut: string): string {
  return shortcut.split('+').map((part) => {
    if (part === 'ctrl') return 'Ctrl'
    if (part === 'alt') return 'Alt'
    if (part === 'shift') return 'Shift'
    if (part === 'arrowleft') return '←'
    if (part === 'arrowright') return '→'
    if (part === 'arrowup') return '↑'
    if (part === 'arrowdown') return '↓'
    if (part === 'delete') return 'Del'
    if (part === 'escape') return 'Esc'
    if (part.length === 1) return part.toUpperCase()
    return part.toUpperCase()
  }).join('+')
}

export function shortcutMatches(event: ShortcutEventLike, shortcut: string): boolean {
  const tokens = shortcut.toLowerCase().split('+')
  const key = tokens.pop() ?? ''
  const modifiers = new Set(tokens)
  return normalizedKey(event.key) === key
    && (modifiers.has('ctrl') === (event.ctrlKey || event.metaKey))
    && (modifiers.has('alt') === event.altKey)
    && (modifiers.has('shift') === event.shiftKey)
}

export function commandForShortcut(event: ShortcutEventLike): ScheduleCommandSpec | undefined {
  return SCHEDULE_COMMANDS.find((command) =>
    (command.shortcuts ?? []).some((shortcut) => shortcutMatches(event, shortcut)),
  )
}
