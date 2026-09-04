/**
 * WorkbenchPrefsPanel.test.tsx — 设置中心「办公工作台偏好」卡（v4.65 欠账收口）
 *
 * 覆盖：四项自动展开/自动切换偏好逐项渲染；默认值一个不改（三项开、产物关）；
 * 切换即时写入既有 localStorage 键（消费方既有约定不变）；损坏存储值回落默认。
 */
import { fireEvent, render, screen, within } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'
import { LocaleProvider } from '../../gaea/lib/i18n'

import WorkbenchPrefsPanel from './WorkbenchPrefsPanel'

const KEYS = {
  subagent: 'gaea.subagentAutoOpen',
  tasks: 'gaea.tasks.autoOpenSubagent',
  browser: 'gaea.browserAutoOpen',
  deliverable: 'gaea.deliverableAutoOpen',
} as const

// 面板文案经 useT 读字典，测试需包 LocaleProvider（zh 为默认语言，同 SettingsPage.test 先例）
const wrap = (node: React.ReactNode) => <LocaleProvider>{node}</LocaleProvider>

// 行定位：标签文本 → 左侧文本容器 → 行容器（行内含 Switch）
const rowOf = (label: string) => {
  const el = screen.getByText(label)
  return el.parentElement!.parentElement!
}
const switchOf = (label: string) => within(rowOf(label)).getByRole('switch') as HTMLButtonElement

beforeEach(() => {
  window.localStorage.clear()
  Object.defineProperty(navigator, 'language', { value: 'zh-CN', configurable: true })
})

describe('WorkbenchPrefsPanel 办公工作台偏好卡', () => {
  it('渲染卡片标题与四项偏好（含一句话说明）', () => {
    render(wrap(<WorkbenchPrefsPanel />))
    expect(screen.getByText('办公工作台偏好')).toBeTruthy()
    for (const label of ['新子代理自动展开', '新任务自动切任务视图', '浏览器自动弹出', '产物自动弹出']) {
      expect(screen.getByText(label)).toBeTruthy()
    }
    // 说明文案诚实标注生效时机（分工面板为下挂载同步，其余即时生效）
    expect(screen.getByText(/分工面板内胶囊在面板重开后同步/)).toBeTruthy()
    // 三条「即时生效」说明 + 卡片「即时生效」徽章
    expect(screen.getAllByText(/即时生效/).length).toBeGreaterThanOrEqual(4)
  })

  it('未设置时开关状态 = 既有默认值（三项开、产物关，一个不改）', () => {
    render(wrap(<WorkbenchPrefsPanel />))
    expect(switchOf('新子代理自动展开').getAttribute('aria-checked')).toBe('true')
    expect(switchOf('新任务自动切任务视图').getAttribute('aria-checked')).toBe('true')
    expect(switchOf('浏览器自动弹出').getAttribute('aria-checked')).toBe('true')
    expect(switchOf('产物自动弹出').getAttribute('aria-checked')).toBe('false')
  })

  it('切换即时写入既有 localStorage 键（"1"/"0" 约定不变）', () => {
    render(wrap(<WorkbenchPrefsPanel />))
    fireEvent.click(switchOf('新子代理自动展开'))
    expect(window.localStorage.getItem(KEYS.subagent)).toBe('0')
    fireEvent.click(switchOf('新任务自动切任务视图'))
    expect(window.localStorage.getItem(KEYS.tasks)).toBe('0')
    fireEvent.click(switchOf('浏览器自动弹出'))
    expect(window.localStorage.getItem(KEYS.browser)).toBe('0')
    // 产物默认关 → 开
    fireEvent.click(switchOf('产物自动弹出'))
    expect(window.localStorage.getItem(KEYS.deliverable)).toBe('1')
  })

  it('损坏存储值回落既有默认（三项开、产物关），不崩面板', () => {
    window.localStorage.setItem(KEYS.subagent, 'garbage!!')
    window.localStorage.setItem(KEYS.tasks, '{broken json')
    window.localStorage.setItem(KEYS.browser, '??')
    window.localStorage.setItem(KEYS.deliverable, 'not-a-bool')
    render(wrap(<WorkbenchPrefsPanel />))
    expect(switchOf('新子代理自动展开').getAttribute('aria-checked')).toBe('true')
    expect(switchOf('新任务自动切任务视图').getAttribute('aria-checked')).toBe('true')
    expect(switchOf('浏览器自动弹出').getAttribute('aria-checked')).toBe('true')
    expect(switchOf('产物自动弹出').getAttribute('aria-checked')).toBe('false')
  })

  it('已存储的关闭值如实显示为关（与面板内胶囊读同一键）', () => {
    window.localStorage.setItem(KEYS.subagent, '0')
    window.localStorage.setItem(KEYS.deliverable, '1')
    render(wrap(<WorkbenchPrefsPanel />))
    expect(switchOf('新子代理自动展开').getAttribute('aria-checked')).toBe('false')
    expect(switchOf('产物自动弹出').getAttribute('aria-checked')).toBe('true')
  })
})
