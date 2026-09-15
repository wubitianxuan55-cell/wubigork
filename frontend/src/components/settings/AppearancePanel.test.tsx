/**
 * AppearancePanel.test.tsx — 外观设置面板新设计（主题下拉 + Segmented 行式切换）
 *
 * 测试目标：
 * - 主题下拉（antd Select）：labelRender 显示当前主题名（默认 nightJade →「暗夜青」）；
 *   打开后 optionRender 呈现主题名 + 描述；点击 option 写入 store 并持久化（gaea-theme）；
 *   悬停 option 触发实时预览徽章「预览中」。
 * - 显示模式 / 密度 / 动效三块行式切换（antd Segmented）：点击选项文字即调用
 *   对应 store action（setMode/setDensity/setMotion）并写入持久化键。
 *
 * store 单例重置与 localStorage 断言策略：
 * - useAppStore 是跨用例共享的 zustand 单例，beforeEach 必须 localStorage.clear()
 *   + useAppStore.setState(...) 显式把四项外观字段重置回默认值，避免用例间串扰。
 * - 每个交互用例同时断言 store 状态与 localStorage 持久化键值（gaea-theme /
 *   gaea-display-mode / gaea-density / gaea-motion），覆盖「写 store + 落盘」两步。
 *
 * 稳健处理（antd 在 jsdom 下的已知坑）：
 * - Select 下拉面板渲染在 body portal：用 screen.findByText 全局查询并等待出现；
 *   打开动作用 jsdom 下可靠的 fireEvent.mouseDown(.ant-select-selector)。
 * - Segmented 选项 label 常是 icon + 文字的组合 span：getByText 默认只比对元素
 *   直接子文本节点，这里用匹配函数只比对非空直接文本（等值，或以目标文案结尾
 *   以容忍 label 前的 icon 字符），容器节点天然不匹配，保证唯一命中；各面板的
 *   标题/描述文案均不以目标词结尾（已对照 zh.ts settings.appear.* 核查），
 *   故 endsWith 兜底不会误命中。
 */
import { fireEvent, render, screen, within } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'

import AppearancePanel, { DarkModePanel, DensityPanel, MotionPanel } from './AppearancePanel'
import { useAppStore } from '../../stores/appStore'
import { LocaleProvider } from '../../gaea/lib/i18n'

// S2.2b i18n：面板组件经 useT 读字典（zh 为默认语言），断言为中文文案：包 LocaleProvider
const wrap = (node: React.ReactNode) => <LocaleProvider>{node}</LocaleProvider>

/** 元素「直接子文本节点」拼接（与 getByText 默认取词口径一致），去首尾空白 */
const directText = (el: Element): string =>
  Array.from(el.childNodes)
    .filter((n) => n.nodeType === Node.TEXT_NODE)
    .map((n) => n.textContent ?? '')
    .join('')
    .trim()

/**
 * getByText/findByText 匹配函数：只比对元素的非空直接文本；
 * 「等值」覆盖纯文字 label，「以目标结尾」容忍 label 前的 icon 字符前缀。
 * 无直接文本的容器节点不匹配，避免命中多个祖先而报 multiple elements。
 */
const byDirectText = (text: string) => (_content: string, el: Element | null): boolean => {
  if (!el) return false
  const t = directText(el)
  return t.length > 0 && (t === text || t.endsWith(text))
}

/** jsdom 下打开 antd Select：对选择器 mousedown */
const openSelect = (container: HTMLElement) => {
  const selector = container.querySelector('.ant-select-selector')
  expect(selector).toBeTruthy()
  fireEvent.mouseDown(selector as HTMLElement)
}

beforeEach(() => {
  // zustand 单例跨用例共享：清持久化 + 显式重置四项外观字段到默认值
  localStorage.clear()
  useAppStore.setState({ baseTheme: 'nightJade', mode: 'dark', density: 'standard', motion: 'full' })
  // 面板文案经 useT 读字典（zh 为默认语言），断言为中文文案：固定 zh 语言
  Object.defineProperty(navigator, 'language', { value: 'zh-CN', configurable: true })
})

describe('AppearancePanel 外观设置重构：主题下拉 + Segmented', () => {
  it('渲染主题下拉并显示当前主题', () => {
    const { container } = render(wrap(<AppearancePanel />))
    const selector = container.querySelector('.ant-select-selector')
    expect(selector).toBeTruthy()
    // 默认主题 nightJade → 选择框内显示「暗夜青」（within 收敛到选择框，预览块同名不受影响）
    expect(within(selector as HTMLElement).getByText(byDirectText('暗夜青'))).toBeTruthy()
  })

  it('下拉选择主题写入 store 并持久化', async () => {
    const { container } = render(wrap(<AppearancePanel />))
    openSelect(container)
    // 下拉面板渲染在 body portal；「暗夜金」= nightAmber（仅出现在 option，无同名冲突）
    const amber = await screen.findByText(byDirectText('暗夜金'))
    fireEvent.click(amber)
    expect(useAppStore.getState().baseTheme).toBe('nightAmber')
    expect(localStorage.getItem('gaea-theme')).toBe('nightAmber')
  })

  it('悬停下拉选项出现预览中徽章', async () => {
    const { container } = render(wrap(<AppearancePanel />))
    openSelect(container)
    const violet = await screen.findByText(byDirectText('暗夜紫'))
    fireEvent.mouseEnter(violet)
    expect(await screen.findByText(byDirectText('预览中'))).toBeTruthy()
  })

  it('显示模式 Segmented 切换', () => {
    render(wrap(<DarkModePanel />))
    fireEvent.click(screen.getByText(byDirectText('亮色模式')))
    expect(useAppStore.getState().mode).toBe('light')
    expect(localStorage.getItem('gaea-display-mode')).toBe('light')
  })

  it('密度 Segmented 切换', () => {
    render(wrap(<DensityPanel />))
    fireEvent.click(screen.getByText(byDirectText('紧凑')))
    expect(useAppStore.getState().density).toBe('compact')
    expect(localStorage.getItem('gaea-density')).toBe('compact')
  })

  it('动效 Segmented 切换', () => {
    render(wrap(<MotionPanel />))
    fireEvent.click(screen.getByText(byDirectText('减弱动态')))
    expect(useAppStore.getState().motion).toBe('reduced')
    expect(localStorage.getItem('gaea-motion')).toBe('reduced')
  })
})
