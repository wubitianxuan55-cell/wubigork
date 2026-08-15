/**
 * boards/pageRegistry.ts — PageRegistry 集中注册表（3.0 架构 §5.2 / 附 B #3/#5）
 *
 * pageComponents 替代：Record<pageKey, LazyComponent>，在 main.tsx 集中 registerPage。
 * MainLayout 渲染时按 manifest.page 查表解析；与旧 pageComponents 并行一个版本（过渡期）。
 */
import type { ComponentType, LazyExoticComponent } from 'react'

/** 注册条目：统一 lazy 包装后的组件（LazyExoticComponent）或普通组件 */
export type RegisteredPage = LazyExoticComponent<ComponentType> | ComponentType

const registry = new Map<string, RegisteredPage>()

/** 注册页面组件（key 与 manifest.page 对应，如 'ChatPage'） */
export function registerPage(key: string, component: RegisteredPage): void {
  registry.set(key, component)
}

/** 按 manifest.page 查表；未注册返回 undefined（调用方走旧 pageComponents fallback） */
export function getPageComponent(key: string): RegisteredPage | undefined {
  return registry.get(key)
}

/** 已注册页面 key 列表（测试/诊断用） */
export function listRegisteredPages(): string[] {
  return [...registry.keys()]
}

/** 清空注册表（测试隔离用） */
export function clearPageRegistry(): void {
  registry.clear()
}
