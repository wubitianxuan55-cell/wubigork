// OfficePanel.test.tsx — v4.362 读取失败防覆盖：读失败不得让默认草稿一键
// 覆盖真实引擎配置（保存禁用+横幅+重试恢复）。
import React from 'react'
import { describe, expect, it, vi, beforeEach, beforeAll } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

const settingsMocks = vi.hoisted(() => ({
  gaeaSettings: vi.fn(),
}))

vi.mock('../../api/settings', () => ({
  gaeaSettings: (...args: unknown[]) => settingsMocks.gaeaSettings(...args),
}))

const bridgeMocks = vi.hoisted(() => ({
  SaveSettings: vi.fn(),
}))

vi.mock('../../gaea/lib/bridge', () => ({
  app: {
    SaveSettings: bridgeMocks.SaveSettings,
    Reload: vi.fn().mockResolvedValue({ tools: 0, skills: 0 }),
    DocumentLint: vi.fn(),
  },
}))

import OfficePanel from './OfficePanel'
import { LocaleProvider } from '../../gaea/lib/i18n'

const wrap = (node: React.ReactNode) => <LocaleProvider>{node}</LocaleProvider>

// antd disabled Button 的 accessible name 在 jsdom 下不稳定——按文字直查
const saveButton = () =>
  Array.from(document.querySelectorAll('button')).find((b) => b.textContent?.includes('保存')) as HTMLButtonElement

const okSettings = {
  defaultModel: 'xai/grok-4.20',
  subagentModel: '',
  agent: { systemPrompt: '你是办公助手', temperature: 0.4, maxSteps: 24, effort: 'high' },
  permissions: { mode: 'allow', allow: ['fs:*'], ask: [], deny: [] },
  sandbox: { bash: 'enforce', network: true, workspaceRoot: 'C:/ws' },
  providers: [],
}

describe('OfficePanel 读取失败防覆盖（v4.362）', () => {
  beforeAll(async () => {
    const { loadLocale } = await import('../../gaea/lib/i18n')
    await loadLocale('zh')
  })

  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.setItem('gaea-lang', 'zh')
    bridgeMocks.SaveSettings.mockResolvedValue(undefined)
  })

  it('读取成功：正常渲染，保存可用', async () => {
    settingsMocks.gaeaSettings.mockResolvedValue(okSettings)
    render(wrap(<OfficePanel />))
    await waitFor(() => expect(settingsMocks.gaeaSettings).toHaveBeenCalled())
    await waitFor(() => {
      expect(screen.queryByTestId('office-settings-load-failed')).toBeNull()
    })
    expect(saveButton().disabled).toBe(false)
  })

  it('读取失败：横幅+保存禁用，SaveSettings 不被调用；重试成功后恢复', async () => {
    settingsMocks.gaeaSettings.mockRejectedValueOnce(new Error('config unreadable'))
    render(wrap(<OfficePanel />))

    const banner = await screen.findByTestId('office-settings-load-failed')
    expect(banner.textContent).toContain('读取失败')
    expect(saveButton().disabled).toBe(true)

    // 重试成功后横幅消失、保存恢复
    settingsMocks.gaeaSettings.mockResolvedValue(okSettings)
    const retryBtn = banner.querySelector('button')
    expect(retryBtn).toBeTruthy()
    fireEvent.click(retryBtn!)
    await waitFor(() => expect(screen.queryByTestId('office-settings-load-failed')).toBeNull())
    expect(saveButton().disabled).toBe(false)

    // 恢复后保存走真实草稿（含重试拉回的 systemPrompt）
    bridgeMocks.SaveSettings.mockResolvedValue(undefined)
    fireEvent.click(screen.getByRole('button', { name: /保存/ }))
    await waitFor(() => expect(bridgeMocks.SaveSettings).toHaveBeenCalledTimes(1))
    expect((bridgeMocks.SaveSettings.mock.calls[0][0] as { agent: { systemPrompt: string } }).agent.systemPrompt).toBe('你是办公助手')
  })

  // 波3 dsh-office COM 实时协同档：office_com 配置段经既有 Settings/SaveSettings
  // 通道读写（约定键 suite/silent/follow/typing/render_width）——读取回显 +
  // 保存负载携带该段。
  it('office_com 实时编辑节：读取回显草稿，保存负载携带约定键', async () => {
    settingsMocks.gaeaSettings.mockResolvedValue({
      ...okSettings,
      office_com: { suite: 'wps', follow: false, typing: false, silent: true, render_width: 1600 },
    })
    render(wrap(<OfficePanel />))
    await waitFor(() => expect(settingsMocks.gaeaSettings).toHaveBeenCalled())

    // 套件回显 wps；实时编辑节三开关回显（follow=false、typing=false、silent=true）
    expect(screen.getByText('wps（金山 WPS）')).toBeTruthy()
    const switches = Array.from(document.querySelectorAll('button[role="switch"]'))
    // 开关序：沙箱 network（true）→ follow（false）→ typing（false）→ silent（true）
    expect(switches.length).toBe(4)
    expect(switches[1]?.getAttribute('aria-checked')).toBe('false')
    expect(switches[2]?.getAttribute('aria-checked')).toBe('false')
    expect(switches[3]?.getAttribute('aria-checked')).toBe('true')

    bridgeMocks.SaveSettings.mockResolvedValue(undefined)
    fireEvent.click(saveButton())
    await waitFor(() => expect(bridgeMocks.SaveSettings).toHaveBeenCalledTimes(1))
    expect((bridgeMocks.SaveSettings.mock.calls[0][0] as { office_com?: unknown }).office_com).toEqual({
      suite: 'wps', follow: false, typing: false, silent: true, render_width: 1600,
    })
  })
})
