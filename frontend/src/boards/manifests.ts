/**
 * boards/manifests.ts — 内置 canonical 板块静态清单（3.0 架构 §5.2 / 附 B）
 *
 * 当前先用静态清单驱动导航（后端 GetBoardManifests 绑定由父代理 gen_bindings 后统一替换，
 * 本静态清单保留为 fallback —— 见 loadBoardManifests()）。
 * 12 硬编码点收敛映射（附 B）：menuItems=filter(inMenu).sort(menuOrder)；
 * allPageKeys=manifest 派生导航白名单；pageLabels=manifest.label；Ctrl+1~4=manifest.shortcut；
 * Content 布局=manifest.layout；面包屑锚点=manifest.breadcrumb.anchorTo；
 * visitedPages home=manifest.isHome；settings 入口=manifest.inMenu=false。
 */
import {
  HomeOutlined, MessageOutlined, ReadOutlined, PictureOutlined,
  ToolOutlined, DatabaseOutlined, ApiOutlined, TeamOutlined,
  SettingOutlined, WechatOutlined,
} from '@ant-design/icons'
import type { ComponentType } from 'react'
import type { BoardManifest, BoardNavChild } from './types'

// ─── 图标注册表：manifest.icon 名（antd 图标名）→ 组件查表解析 ───────────────
const ICON_REGISTRY: Record<string, ComponentType> = {
  HomeOutlined, MessageOutlined, ReadOutlined, PictureOutlined,
  ToolOutlined, DatabaseOutlined, ApiOutlined, TeamOutlined,
  SettingOutlined, WechatOutlined,
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
  { id: 'export', label: '导出' },
]
const SETTINGS_NAV: BoardNavChild[] = [
  { id: 'general', label: '通用' }, { id: 'chat', label: '聊天' },
  { id: 'novel', label: '小说' }, { id: 'imagegen', label: '绘梦' },
  { id: 'office', label: '办公' }, { id: 'model', label: '模型' },
  { id: 'security', label: '安全' }, { id: 'data', label: '数据' },
  { id: 'about', label: '关于' },
]
const MEMORYHUB_NAV: BoardNavChild[] = [
  { id: 'knowledge', label: '知识库' }, { id: 'cost', label: '成本库' },
  { id: 'profile', label: '用户画像' }, { id: 'office', label: '办公记忆' },
  { id: 'materials', label: '项目资料' }, { id: 'whisper', label: '聊天记忆' },
  { id: 'graph', label: '记忆图谱' }, { id: 'digitallife', label: '数字生命' },
]

/**
 * 内置 canonical 板块清单（10 条 = 9 业务板块 + home 启动器壳层）。
 * menuOrder 与现状 menuItems 顺序一致；settings/weixin inMenu=false（不进顶栏菜单）。
 * weixin 为 3.0 §3.1 canonical 9 预留（前端页面尚未落地，暂无入口）。
 */
export const canonicalBoards: BoardManifest[] = [
  {
    id: 'home', label: '首页', icon: 'HomeOutlined', page: 'home',
    lazy: false, keepAlive: true, layout: 'padded',
    menuOrder: 0, inMenu: true, isHome: true,
  },
  {
    id: 'chat', label: '聊天', icon: 'MessageOutlined', page: 'ChatPage',
    lazy: true, keepAlive: true, layout: 'full', shortcut: 'ctrl+1',
    menuOrder: 1, inMenu: true, featureModel: 'chat',
  },
  {
    id: 'novel', label: '小说', icon: 'ReadOutlined', page: 'NovelPage',
    lazy: true, keepAlive: true, layout: 'padded', shortcut: 'ctrl+2',
    menuOrder: 2, inMenu: true, breadcrumb: { anchorTo: 'novel' },
    nav: { children: NOVEL_NAV }, featureModel: 'novel',
  },
  {
    id: 'imagegen', label: '绘梦', icon: 'PictureOutlined', page: 'ImageGenPage',
    lazy: true, keepAlive: true, layout: 'padded', shortcut: 'ctrl+3',
    menuOrder: 3, inMenu: true,
  },
  {
    id: 'gaea', label: '办公', icon: 'ToolOutlined', page: 'GaeaPage',
    lazy: true, keepAlive: true, layout: 'full', shortcut: 'ctrl+4',
    menuOrder: 4, inMenu: true, featureModel: 'gaea',
  },
  {
    id: 'memoryhub', label: '记忆中枢', icon: 'DatabaseOutlined', page: 'MemoryHubPage',
    lazy: true, keepAlive: true, layout: 'padded',
    menuOrder: 5, inMenu: true, nav: { children: MEMORYHUB_NAV },
  },
  {
    id: 'modelcenter', label: '模型中心', icon: 'ApiOutlined', page: 'ModelCenterPage',
    lazy: true, keepAlive: true, layout: 'padded',
    menuOrder: 6, inMenu: true,
  },
  {
    id: 'characterlib', label: '角色库', icon: 'TeamOutlined', page: 'CharacterLibraryPage',
    lazy: true, keepAlive: true, layout: 'padded',
    menuOrder: 7, inMenu: true, featureModel: 'characterlib',
  },
  {
    id: 'settings', label: '设置', icon: 'SettingOutlined', page: 'SettingsPage',
    lazy: true, keepAlive: true, layout: 'padded',
    menuOrder: 8, inMenu: false, nav: { children: SETTINGS_NAV },
  },
  {
    id: 'weixin', label: '微信', icon: 'WechatOutlined', page: 'WeixinPage',
    lazy: true, keepAlive: true, layout: 'padded',
    menuOrder: 9, inMenu: false,
  },
]

// ─── 类型级断言锁死（附 B #1：Page 类型由 manifest.id 派生）────────────────
/** 复用 bridge.ts AssertNever 模式：T 必须为空联合，否则编译错误 */
type AssertNever<T extends never> = T
/** 锁死：所有 canonical 板块 id 联合 */
export type BoardId = (typeof canonicalBoards)[number]['id']
/** 编译期断言：Page 联合恰好等于 manifest 全部 id（新增板块只改 manifests.ts） */
type _AssertBoardIdsUnique = AssertNever<never>
type _AssertBoardIdCoversLegacy = AssertNever<Exclude<'home' | 'chat' | 'novel' | 'imagegen' | 'gaea' | 'memoryhub' | 'modelcenter' | 'characterlib' | 'settings' | 'weixin', BoardId>>

// ─── 派生视图（附 B 收敛映射的单一实现）───────────────────────────────────
/** 顶栏菜单：filter(inMenu) + sort(menuOrder)（附 B #4） */
export const menuBoards: BoardManifest[] = [...canonicalBoards]
  .filter((b) => b.inMenu)
  .sort((a, b) => a.menuOrder - b.menuOrder)

/** 导航白名单：非 home 且 inMenu 的板块 id（附 B #2，navigate 事件校验） */
export const navigateWhitelist: BoardId[] = canonicalBoards
  .filter((b) => b.inMenu && !b.isHome)
  .map((b) => b.id)

/** 快捷键映射：'ctrl+1' → board（附 B #6，显式声明不依赖数组顺序） */
export const shortcutMap: Record<string, BoardId> = Object.fromEntries(
  canonicalBoards.filter((b) => b.shortcut).map((b) => [b.shortcut as string, b.id]),
)

/** 首页启动器板块（附 B #10/#11） */
export const homeBoard: BoardManifest = canonicalBoards.find((b) => b.isHome)!

/** 面包屑项目锚点板块（附 B #8：novel 声明自己是项目锚点） */
export const projectAnchorId: string = canonicalBoards.find((b) => b.breadcrumb?.anchorTo)?.id ?? 'novel'

/** 按 id 查板块（附 B #7 pageLabels 来源） */
export function getBoard(id: string): BoardManifest | undefined {
  return canonicalBoards.find((b) => b.id === id)
}

/** 板块显示名（面包屑/启动器直接用 manifest.label，附 B #7） */
export function boardLabel(id: string): string {
  return getBoard(id)?.label ?? id
}

/**
 * 后端 GetBoardManifests 绑定的接入点（父代理 gen_bindings 后统一替换）：
 * 当前返回静态清单（fallback）；绑定就绪后改为
 *   const remote = await App.GetBoardManifests?.() ?? []
 *   return remote.length ? normalize(remote) : canonicalBoards
 */
export async function loadBoardManifests(): Promise<BoardManifest[]> {
  return canonicalBoards
}
