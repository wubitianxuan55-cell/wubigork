/**
 * boards/space.ts — 双空间壳 S2.1 空间模型
 * （docs/gaea-space-shell-design.md §4.2）
 *
 * 壳层视图空间（currentSpace）与 manifest 板块归属（space）分离：
 *   - ShellSpace 决定导航/首页/事件面（本地持久化，见 appStore）；
 *   - BoardSpace 是板块的静态归属：shared = 两空间均可达；
 *     independent = 独立窗口（用户拍板：不并入工位、不共享工具面）——
 *     两空间导航/首页均不出现，壳层单独入口可达。v4.439 编程板块删除后
 *     该分面暂无成员（机制保留，供后续独立窗口类板块复用）。
 * 缺省语义：manifest 无 space 字段 → work（旧数据按 work 兼容回填，
 * 与阶段 1 S1.1 旧数据回填 work 同语义）；home 壳层恒 shared。
 */
import type { BoardManifest } from './types'
import type { DictKey } from '../gaea/locales/en'

/** 壳层视图空间 */
export type ShellSpace = 'work' | 'play'

/** manifest 板块空间归属 */
export type BoardSpace = ShellSpace | 'shared' | 'independent'

/**
 * 壳层空间静态枚举（导航切换器/搜索 scope 共用标签）。
 * label/title 为 zh 兜底（纯数据层），labelKey/titleKey 供组件 i18n 渲染
 * （S2.2 i18n 全铺第一刀：新壳 chrome 全部走字典）。
 */
export const SHELL_SPACES: {
  id: ShellSpace
  label: string
  title: string
  labelKey: DictKey
  titleKey: DictKey
}[] = [
  { id: 'work', label: '工位', title: '工位（办公/造价/记忆）——工作空间', labelKey: 'shell.space.work', titleKey: 'shell.space.workTitle' },
  { id: 'play', label: '乐园', title: '乐园（轻语/小说/绘梦）——娱乐空间', labelKey: 'shell.space.play', titleKey: 'shell.space.playTitle' },
]

/** 类型守卫：仅 work/play 是合法壳层空间（localStorage 读取/后端回包用） */
export function isShellSpace(v: unknown): v is ShellSpace {
  return v === 'work' || v === 'play'
}

/** 板块空间归属解析：manifest.space 优先；缺省 home→shared、其余→work */
export function boardSpace(b: BoardManifest): BoardSpace {
  if (b.space) return b.space
  return b.isHome ? 'shared' : 'work'
}

/** 独立窗口板块（两空间导航/首页均不出现） */
export function isIndependentBoard(b: BoardManifest): boolean {
  return boardSpace(b) === 'independent'
}

// 一级导航合一（2026-10-26 用户拍板）：isBoardReachableInSpace /
// filterBoardsForSpace / pruneVisitedForSpace 随 rail 分域退役——壳层空间
// 降为纯数据面概念（会话空间/事件分域/任务标签仍用 ShellSpace 与 boardSpace），
// 导航/keepAlive/快捷键不再按空间过滤或剪枝。
