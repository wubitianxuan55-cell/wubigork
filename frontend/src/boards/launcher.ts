/**
 * boards/launcher.ts — 首页启动器模块派生（3.0 架构 §5.2 附 B #10/#11）
 *
 * ModuleLauncher 的清单化数据源：把「非 home 且可达（inMenu 或 settings 隐式入口）
 * 的板块 → 启动器卡片」抽成纯函数，输入为任意 BoardManifest 列表（静态
 * canonicalBoards 或后端合并活动清单），输出 LauncherModule[]。icon 为图标注册表
 * 名（字符串），渲染处经 resolveBoardIcon 查表解析（未知 → Thunderbolt 兜底）。
 * desc 是 UI 文案层（manifest 契约不含），经 LAUNCHER_DESC 按板块 id 覆盖，
 * 缺失时兜底 = manifest.label。
 */
import type { BoardManifest } from './types'

/** 启动器卡片模块（icon 为图标注册表名，渲染处 resolveBoardIcon 解析） */
export interface LauncherModule {
  key: string
  name: string
  desc: string
  icon: string
}

// 卡片描述为 UI 文案（manifest 契约不含 desc），按板块 id 本地维护；
// 名称/图标/顺序全部由 manifest 派生（3.0 §5.2，顺带补 memoryhub/characterlib 缺失入口）。
export const LAUNCHER_DESC: Record<string, string> = {
  chat: '与 AI 对话，激发灵感',
  novel: '世界观、角色与大纲创作',
  imagegen: 'AI 图像生成工作台',
  gaea: '通用办公工作台',
  memoryhub: '知识/成本/画像跨板块记忆沉淀',
  modelcenter: '模型引擎管理与配置',
  characterlib: '角色档案与跨板块角色管理',
  settings: '应用偏好与主题外观',
}

/**
 * 启动器模块清单 = 非 home 且可达（inMenu 或 settings 隐式入口）的板块，
 * 按 menuOrder 升序（undefined 视为无穷大，与 manifests.ts 其它派生视图一致；
 * filter 已产出新数组，sort 不污染入参）。desc 取 descMap 覆盖文案，缺失兜底
 * = manifest.label；icon 字段透传图标注册表名（渲染处解析）。
 */
export function deriveLauncherModules(list: BoardManifest[], descMap: Record<string, string>): LauncherModule[] {
  return list
    .filter((b) => !b.isHome && (b.inMenu || b.id === 'settings'))
    .sort((a, b) => (a.menuOrder ?? Number.MAX_SAFE_INTEGER) - (b.menuOrder ?? Number.MAX_SAFE_INTEGER))
    .map((b) => ({
      key: b.id,
      name: b.label,
      desc: descMap[b.id] ?? b.label,
      icon: b.icon,
    }))
}
