/**
 * boards/manifests.ts — 板块 manifest 数据源（3.0 架构 §5.2 / 附 B）
 *
 * 数据源 seam（§5.3 前端侧）：提供者 = CoreB.GetBoardManifests()（经 wailsjsCompat
 * 直调，见 gaea/lib/bridge.ts LegacySurfaceNames 注记）；消费者 = 菜单/白名单/快捷键/
 * 布局派生视图（MainLayout 经 subscribeBoards 订阅）。加载失败/未就绪时 fail-closed
 * 回退内置静态 canonicalBoards，壳层永远可用。
 * 板块差集归一（normalizeManifests）：后端清单为准 + 前端 home 壳层补位——
 *   · knowledge：后端 D7 独立板块（前端静态清单无）→ 并入；
 *   · home：后端无壳层 → 补前端静态壳层（isHome 唯一）；
 *   · weixin：后端 page=""（无前端页面）→ 以后端为准（静态 WeixinPage 未注册不覆盖）。
 * 12 硬编码点收敛映射（附 B）：menuItems=filter(inMenu).sort(menuOrder)；
 * allPageKeys=manifest 派生导航白名单；pageLabels=manifest.label；Ctrl+1~4=manifest.shortcut；
 * Content 布局=manifest.layout；面包屑锚点=manifest.breadcrumb.anchorTo；
 * visitedPages home=manifest.isHome；settings 入口=manifest.inMenu=false。
 */
import {
  HomeOutlined, MessageOutlined, ReadOutlined, PictureOutlined,
  ToolOutlined, DatabaseOutlined, ApiOutlined, TeamOutlined,
  SettingOutlined, WechatOutlined, BookOutlined, CodeOutlined, AccountBookOutlined,
} from '@ant-design/icons'
import type { ComponentType } from 'react'
import type { BoardManifest, BoardNavChild } from './types'
import { GetBoardManifests } from '../wailsjsCompat'
import { filterBoardsForSpace, isIndependentBoard, type ShellSpace } from './space'

// ─── 图标注册表：manifest.icon 名（antd 图标名）→ 组件查表解析 ───────────────
const ICON_REGISTRY: Record<string, ComponentType> = {
  HomeOutlined, MessageOutlined, ReadOutlined, PictureOutlined,
  ToolOutlined, DatabaseOutlined, ApiOutlined, TeamOutlined,
  SettingOutlined, WechatOutlined, BookOutlined, CodeOutlined, AccountBookOutlined,
}

/** 查表解析 antd 图标；未知图标名返回 null（不抛错，渲染时退化） */
export function resolveBoardIcon(name: string): ComponentType | null {
  return ICON_REGISTRY[name] ?? null
}

// ─── 板块内子导航（与各页面实际 tab/分类一致）─────────────────────────────
const NOVEL_NAV: BoardNavChild[] = [
  { id: 'home', label: '书架' },
  { id: 'novelsetting', label: '设定' },
  { id: 'character', label: '角色' },
  { id: 'create', label: '创作' },
  { id: 'chapter', label: '阅读' },
]
const SETTINGS_NAV: BoardNavChild[] = [
  { id: 'general', label: '通用' }, { id: 'chat', label: '聊天' },
  { id: 'novel', label: '小说' }, { id: 'imagegen', label: '绘梦' },
  { id: 'office', label: '办公' }, { id: 'model', label: '模型' },
  { id: 'security', label: '安全' }, { id: 'data', label: '数据' },
  { id: 'about', label: '关于' },
]
const MEMORYHUB_NAV: BoardNavChild[] = [
  { id: 'knowledge', label: '知识库' },
  { id: 'profile', label: '用户画像' }, { id: 'office', label: '办公记忆' },
  { id: 'materials', label: '项目资料' }, { id: 'whisper', label: '聊天记忆' },
  { id: 'graph', label: '记忆图谱' }, { id: 'digitallife', label: '数字生命' },
]
const COST_NAV: BoardNavChild[] = [
  { id: 'overview', label: '概览' },
  { id: 'entries', label: '成本条目' },
  { id: 'sources', label: '价格源' },
  { id: 'repository', label: '价格仓库' },
]

/**
 * 内置 canonical 板块清单（11 条 = 10 业务板块 + home 启动器壳层）。
 * menuOrder 与现状 menuItems 顺序一致；settings/weixin inMenu=false（不进顶栏菜单）。
 * weixin 为 3.0 §3.1 canonical 9 预留（前端页面尚未落地，暂无入口）。
 */
export const canonicalBoards: BoardManifest[] = [
  {
    id: 'home', label: '首页', icon: 'HomeOutlined', page: 'home',
    lazy: false, keepAlive: true, layout: 'padded',
    menuOrder: 0, inMenu: true, isHome: true,
    space: 'shared',
  },
  {
    id: 'chat', label: '聊天', icon: 'MessageOutlined', page: 'ChatPage',
    lazy: true, keepAlive: true, layout: 'full', shortcut: 'ctrl+1',
    menuOrder: 1, inMenu: true, featureModel: 'chat', space: 'shared',
  },
  {
    id: 'novel', label: '小说', icon: 'ReadOutlined', page: 'NovelPage',
    lazy: true, keepAlive: true, layout: 'padded', shortcut: 'ctrl+2',
    menuOrder: 2, inMenu: true, breadcrumb: { anchorTo: 'novel' },
    nav: { children: NOVEL_NAV }, featureModel: 'novel', space: 'play',
  },
  {
    id: 'imagegen', label: '绘梦', icon: 'PictureOutlined', page: 'ImageGenPage',
    lazy: true, keepAlive: true, layout: 'padded', shortcut: 'ctrl+3',
    menuOrder: 3, inMenu: true, space: 'play',
  },
  {
    id: 'gaea', label: '办公', icon: 'ToolOutlined', page: 'GaeaPage',
    lazy: true, keepAlive: true, layout: 'full', shortcut: 'ctrl+4',
    menuOrder: 4, inMenu: true, featureModel: 'gaea', space: 'work',
  },
  {
    id: 'cost', label: '造价数据库', icon: 'AccountBookOutlined', page: 'CostLibraryPage',
    lazy: true, keepAlive: true, layout: 'padded',
    menuOrder: 5, inMenu: true, nav: { children: COST_NAV }, featureModel: 'cost', space: 'work',
  },
  {
    id: 'code', label: '编程', icon: 'CodeOutlined', page: 'ProgrammingPage',
    lazy: true, keepAlive: true, layout: 'full', // 桌面内嵌 Harness Web 工作台（全出血）
    menuOrder: 6, inMenu: true, space: 'independent', // 独立 DSH 窗口（用户拍板：不并入工位/乐园）
  },
  {
    id: 'memoryhub', label: '记忆中枢', icon: 'DatabaseOutlined', page: 'MemoryHubPage',
    lazy: true, keepAlive: true, layout: 'padded',
    menuOrder: 7, inMenu: true, nav: { children: MEMORYHUB_NAV }, space: 'work',
  },
  {
    id: 'modelcenter', label: '模型中心', icon: 'ApiOutlined', page: 'ModelCenterPage',
    lazy: true, keepAlive: true, layout: 'padded',
    menuOrder: 8, inMenu: true, space: 'shared',
  },
  {
    id: 'characterlib', label: '角色库', icon: 'TeamOutlined', page: 'CharacterLibraryPage',
    lazy: true, keepAlive: true, layout: 'padded',
    menuOrder: 9, inMenu: true, featureModel: 'characterlib', space: 'play',
  },
  {
    id: 'settings', label: '设置', icon: 'SettingOutlined', page: 'SettingsPage',
    lazy: true, keepAlive: true, layout: 'padded',
    menuOrder: 10, inMenu: false, nav: { children: SETTINGS_NAV }, space: 'shared',
  },
  {
    id: 'weixin', label: '微信', icon: 'WechatOutlined', page: 'WeixinPage',
    lazy: true, keepAlive: true, layout: 'padded',
    menuOrder: 11, inMenu: false, space: 'work',
  },
]

// ─── 类型级断言锁死（附 B #1：Page 类型由 manifest.id 派生）────────────────
/** 复用 bridge.ts AssertNever 模式：T 必须为空联合，否则编译错误 */
type AssertNever<T extends never> = T
/** 锁死：所有 canonical 板块 id 联合 */
export type BoardId = (typeof canonicalBoards)[number]['id']
/** 编译期断言：Page 联合恰好等于 manifest 全部 id（新增板块只改 manifests.ts） */
type _AssertBoardIdsUnique = AssertNever<never>
type _AssertBoardIdCoversLegacy = AssertNever<Exclude<'home' | 'chat' | 'novel' | 'imagegen' | 'gaea' | 'cost' | 'code' | 'memoryhub' | 'modelcenter' | 'characterlib' | 'settings' | 'weixin', BoardId>>

// ─── 派生视图通用实现（列表参数化：静态 canonicalBoards 与活动清单共用）──────
/** 顶栏菜单：filter(inMenu) + sort(menuOrder)（附 B #4） */
export function deriveMenuBoards(list: BoardManifest[]): BoardManifest[] {
  return [...list]
    .filter((b) => b.inMenu)
    .sort((a, b) => (a.menuOrder ?? Number.MAX_SAFE_INTEGER) - (b.menuOrder ?? Number.MAX_SAFE_INTEGER))
}

/** 导航白名单：非 home 且 inMenu 的板块 id（附 B #2，navigate 事件校验） */
export function deriveNavigateWhitelist(list: BoardManifest[]): string[] {
  return list.filter((b) => b.inMenu && !b.isHome).map((b) => b.id)
}

/** 快捷键映射：'ctrl+1' → board（附 B #6，显式声明不依赖数组顺序） */
export function deriveShortcutMap(list: BoardManifest[]): Record<string, string> {
  return Object.fromEntries(list.filter((b) => b.shortcut).map((b) => [b.shortcut as string, b.id]))
}

/** 首页启动器板块（附 B #10/#11） */
export function deriveHomeBoard(list: BoardManifest[]): BoardManifest | undefined {
  return list.find((b) => b.isHome)
}

/** 面包屑项目锚点板块（附 B #8：声明 breadcrumb.anchorTo 的板块，novel 是项目锚点） */
export function deriveProjectAnchorId(list: BoardManifest[]): string {
  return list.find((b) => b.breadcrumb?.anchorTo)?.id ?? 'novel'
}

/** 按 id 查板块（附 B #7 pageLabels 来源） */
export function deriveBoard(list: BoardManifest[], id: string): BoardManifest | undefined {
  return list.find((b) => b.id === id)
}

/** 板块显示名（面包屑/启动器直接用 manifest.label，附 B #7） */
export function deriveBoardLabel(list: BoardManifest[], id: string): string {
  return deriveBoard(list, id)?.label ?? id
}

// ─── 静态派生视图（fallback 基线，附 B 像素回归测试锁定）─────────────────
export const menuBoards: BoardManifest[] = deriveMenuBoards(canonicalBoards)
export const navigateWhitelist: BoardId[] = deriveNavigateWhitelist(canonicalBoards)
export const shortcutMap: Record<string, BoardId> = deriveShortcutMap(canonicalBoards)
export const homeBoard: BoardManifest = deriveHomeBoard(canonicalBoards)!
export const projectAnchorId: string = deriveProjectAnchorId(canonicalBoards)
export function getBoard(id: string): BoardManifest | undefined {
  return deriveBoard(canonicalBoards, id)
}
export function boardLabel(id: string): string {
  return deriveBoardLabel(canonicalBoards, id)
}

// ─── 后端 GetBoardManifests 接线（§5.3 seam 前端侧）──────────────────────
// 提供者 = CoreB.GetBoardManifests()（经 wailsjsCompat 直调）；消费者 = 下方
// getActive* 派生视图（MainLayout 订阅）。加载前活动清单 = 静态 canonicalBoards。

/** 后端 manifest 原始形态（wailsjs board.Manifest 字段子集，结构兼容） */
export interface RemoteBoardManifest {
  id: string
  label?: string
  icon?: string
  page?: string
  lazy?: boolean
  keepAlive?: boolean
  layout?: string
  shortcut?: string
  menuOrder?: number
  inMenu?: boolean
  space?: 'work' | 'play' | 'shared'
  breadcrumb?: { anchorTo?: string }
  isHome?: boolean
  nav?: { children?: BoardNavChild[] }
  featureModel?: string
}

/**
 * 板块差集归一（merge 语义 = 后端清单 + 前端 home 壳层）：
 *  - 重叠 id：后端字段优先，缺失字段回填前端静态（icon/page/layout 兜底）；
 *  - knowledge：后端 D7 独立板块**过滤不并入一级导航**——知识库已并入记忆中枢
 *    （MemoryHubPage「知识库」分类），一级导航不再单列避免重复入口（3.0 定制）；
 *  - home：后端无 isHome 板块时补前端静态壳层，menuOrder=0 恒首位（差集 #2）；
 *  - weixin：后端 page=""（无前端页面）保留空串，静态 WeixinPage 不覆盖（差集 #3）；
 *  - 空输入返回 []（回退决策在 loadBoardManifests 层）。
 */
export function normalizeManifests(remote: RemoteBoardManifest[]): BoardManifest[] {
  const staticById = new Map(canonicalBoards.map((b) => [b.id, b]))
  const out: BoardManifest[] = []
  for (const r of remote ?? []) {
    if (!r || typeof r.id !== 'string' || r.id === '') continue
    // 3.0 定制：knowledge 并入记忆中枢，一级导航不单列（后端清单照常返回，仅导航侧过滤）
    if (r.id === 'knowledge') continue
    const base = staticById.get(r.id)
    out.push({
      id: r.id,
      label: r.label ?? base?.label ?? r.id,
      icon: r.icon ?? base?.icon ?? '',
      page: r.page ?? base?.page ?? '',
      lazy: r.lazy ?? base?.lazy ?? false,
      keepAlive: r.keepAlive ?? base?.keepAlive ?? true,
      layout: r.layout === 'full' || r.layout === 'padded' ? r.layout : (base?.layout ?? 'padded'),
      shortcut: r.shortcut ?? base?.shortcut,
      menuOrder: r.menuOrder ?? base?.menuOrder,
      inMenu: r.inMenu ?? base?.inMenu ?? false,
      space: r.space ?? base?.space,
      breadcrumb: r.breadcrumb ? { anchorTo: r.breadcrumb.anchorTo } : base?.breadcrumb,
      isHome: r.isHome ?? base?.isHome ?? false,
      nav: r.nav ? { children: r.nav.children ?? [] } : base?.nav,
      featureModel: r.featureModel ?? base?.featureModel,
    })
  }
  if (out.length > 0 && !out.some((b) => b.isHome)) {
    out.push({ ...homeBoard })
  }
  // 稳定排序：menuOrder 升序（undefined 视为无穷大；home=0 恒首位）
  out.sort((a, b) => (a.menuOrder ?? Number.MAX_SAFE_INTEGER) - (b.menuOrder ?? Number.MAX_SAFE_INTEGER))
  return out
}

// ── 活动清单状态（加载成功后被合并清单替换；失败保持静态 fallback）──────
let activeBoards: BoardManifest[] = canonicalBoards
const boardListeners = new Set<() => void>()

/** 订阅活动板块清单变化（MainLayout 重渲染用）；返回退订函数 */
export function subscribeBoards(cb: () => void): () => void {
  boardListeners.add(cb)
  return () => { boardListeners.delete(cb) }
}

function notifyBoardsChanged(): void {
  for (const cb of boardListeners) cb()
}

/** 当前生效板块清单（后端 + home 壳层合并；未加载 = 静态 canonicalBoards） */
export function getActiveBoards(): BoardManifest[] {
  return activeBoards
}

// ── 活动派生视图（消费者：MainLayout 菜单/白名单/快捷键/布局/面包屑）────
export function getActiveMenuBoards(): BoardManifest[] { return deriveMenuBoards(activeBoards) }
/** S2.1：按壳层空间过滤后的菜单（shared + 当前空间板块） */
export function getActiveMenuBoardsForSpace(space: ShellSpace): BoardManifest[] {
  return deriveMenuBoards(filterBoardsForSpace(activeBoards, space))
}
/** S2.1：独立窗口板块（编程 DSH）——两空间导航/首页均不出现，壳层单独入口 */
export function getActiveIndependentBoards(): BoardManifest[] {
  return deriveMenuBoards(activeBoards.filter(isIndependentBoard))
}
export function getActiveNavigateWhitelist(): string[] { return deriveNavigateWhitelist(activeBoards) }
export function getActiveShortcutMap(): Record<string, string> { return deriveShortcutMap(activeBoards) }
export function getActiveHomeBoard(): BoardManifest { return deriveHomeBoard(activeBoards) ?? homeBoard }
export function getActiveProjectAnchorId(): string { return deriveProjectAnchorId(activeBoards) }
export function getActiveBoard(id: string): BoardManifest | undefined { return deriveBoard(activeBoards, id) }
export function activeBoardLabel(id: string): string { return deriveBoardLabel(activeBoards, id) }

/** 测试隔离：恢复静态 fallback 基线（vitest beforeEach，与 clearPageRegistry 同范式） */
export function resetActiveBoardsForTest(): void {
  activeBoards = canonicalBoards
  notifyBoardsChanged()
}

/**
 * 后端 GetBoardManifests 绑定的接入点（§5.3 seam 提供者调用）：
 *  1. 优先 CoreB.GetBoardManifests()（经 wailsjsCompat 直调）；
 *  2. 成功 → normalizeManifests 合并（后端清单 + home 壳层）并替换活动清单；
 *  3. 失败/空/未就绪（浏览器 dev mock 无 window.go）→ fail-closed 回退静态
 *     canonicalBoards，壳层保持可用。
 * 返回生效清单；消费方经 subscribeBoards 感知变化。
 */
export async function loadBoardManifests(): Promise<BoardManifest[]> {
  let merged: BoardManifest[] = []
  try {
    const remote = await GetBoardManifests()
    if (Array.isArray(remote)) {
      merged = normalizeManifests(remote)
    }
  } catch {
    merged = []
  }
  activeBoards = merged.length ? merged : canonicalBoards
  notifyBoardsChanged()
  return activeBoards
}
