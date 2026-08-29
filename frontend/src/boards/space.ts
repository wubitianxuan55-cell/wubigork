/**
 * boards/space.ts — 双空间壳 S2.1 空间模型
 * （docs/gaea-space-shell-design.md §4.2）
 *
 * 壳层视图空间（currentSpace）与 manifest 板块归属（space）分离：
 *   - ShellSpace 决定导航/首页/事件面（本地持久化，见 appStore）；
 *   - BoardSpace 是板块的静态归属：shared = 两空间均可达；
 *     independent = 独立窗口（编程 DSH，用户拍板：不并入工位、不共享工具面）——
 *     两空间导航/首页均不出现，壳层单独入口可达。
 * 缺省语义：manifest 无 space 字段 → work（旧数据按 work 兼容回填，
 * 与阶段 1 S1.1 旧数据回填 work 同语义）；home 壳层恒 shared。
 */
import type { BoardManifest } from './types'

/** 壳层视图空间 */
export type ShellSpace = 'work' | 'play'

/** manifest 板块空间归属 */
export type BoardSpace = ShellSpace | 'shared' | 'independent'

/** 壳层空间静态枚举（导航切换器/搜索 scope 共用标签） */
export const SHELL_SPACES: { id: ShellSpace; label: string; title: string }[] = [
  { id: 'work', label: '工位', title: '工位（办公/造价/记忆）——工作空间' },
  { id: 'play', label: '乐园', title: '乐园（轻语/小说/绘梦）——娱乐空间' },
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

/** 板块在指定壳层空间是否可达：shared 或归属等于当前空间；independent 恒不可达 */
export function isBoardReachableInSpace(b: BoardManifest, space: ShellSpace): boolean {
  const s = boardSpace(b)
  if (s === 'independent') return false
  return s === 'shared' || s === space
}

/** 清单按空间过滤（导航/启动器/快捷键共用；保持原顺序） */
export function filterBoardsForSpace(list: BoardManifest[], space: ShellSpace): BoardManifest[] {
  return list.filter((b) => isBoardReachableInSpace(b, space))
}
