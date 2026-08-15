/**
 * boards/types.ts — 板块 Manifest 类型定义（3.0 架构 §5.2 / 附 B）
 *
 * 字段与后端 GetBoardManifests() []BoardManifest 契约一致（Go 侧 internal/app/board/，
 * gen_bindings 生成后由父代理统一接线；当前前端先用静态清单驱动，静态清单保留为 fallback）。
 */

/** 内容区布局：full = chat/gaea 全出血（零 padding + 玻璃底 + 隐藏滚动）；padded = 默认 padding */
export type BoardLayout = 'full' | 'padded'

/** 板块内子导航项（NovelPage 6 tab / SettingsPage 9 分类 / MemoryHubPage 8 库等） */
export interface BoardNavChild {
  id: string
  label: string
  page?: string
  icon?: string
}

export interface BoardNav {
  children: BoardNavChild[]
}

/**
 * 板块 Manifest（导航白名单 / 菜单 / 快捷键 / 布局 / 面包屑 / 启动器全部由它派生）。
 * 约定：
 *  - label 同时服务菜单、面包屑、启动器显示名（单一来源）；
 *  - icon 为 antd 图标名（'MessageOutlined' 等），由图标注册表查表解析；
 *  - page 为 PageRegistry 中注册的页面组件 key（替代 lazy import + pageComponents 两处登记）；
 *  - isHome 仅一块（首页启动器，壳层渲染 ModuleLauncher）；
 *  - inMenu=false 不进顶栏菜单（settings/weixin 走隐式入口）。
 */
export interface BoardManifest {
  /** 板块稳定 id（导航白名单键，现 allPageKeys 的替代） */
  id: string
  /** 菜单/面包屑/启动器显示名 */
  label: string
  /** antd 图标名（图标注册表查表解析） */
  icon: string
  /** 页面组件 key，在 PageRegistry 中查找（替代 lazy import + pageComponents 两处登记） */
  page: string
  /** PageRegistry 统一 lazy 包装 */
  lazy: boolean
  /** visitedPages display:none 保活策略（默认 true） */
  keepAlive?: boolean
  /** full = chat/gaea 全出血；padded = 默认 padding */
  layout?: BoardLayout
  /** ctrl+1~4 显式声明（不依赖数组顺序） */
  shortcut?: string
  /** 菜单顺序 */
  menuOrder?: number
  /** 是否进菜单（settings 不进菜单，右上角按钮） */
  inMenu?: boolean
  /** 面包屑"项目名→novel"锚点语义 */
  breadcrumb?: { anchorTo?: string }
  /** 首页特判分支的替代 */
  isHome?: boolean
  /** 板块内子导航（NovelPage 6 tab/SettingsPage 9 分类/MemoryHubPage 8 库等） */
  nav?: BoardNav
  /** FeatureModelBar 的 feature 键（"事实上的板块注册表"，gaea/office 二义需消歧） */
  featureModel?: string
}
