// SearchPanel.test.tsx — 搜索主源设置卡测试（众神殿刀3）：读取可用性徽标、
// 保存回执、连通性实测三分支（ok/degraded/全败带指引）。
import React from 'react'
import { describe, expect, it, vi, beforeEach, beforeAll } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

const settingsMocks = vi.hoisted(() => ({
  webSearchSettings: vi.fn(),
  saveWebSearchSettings: vi.fn(),
  testWebSearchConnection: vi.fn(),
}))

vi.mock('../../api/settings', () => ({
  webSearchSettings: (...args: unknown[]) => settingsMocks.webSearchSettings(...args),
  saveWebSearchSettings: (...args: unknown[]) => settingsMocks.saveWebSearchSettings(...args),
  testWebSearchConnection: (...args: unknown[]) => settingsMocks.testWebSearchConnection(...args),
}))

import SearchPanel from './SearchPanel'
import { LocaleProvider } from '../../gaea/lib/i18n'

const wrap = (node: React.ReactNode) => <LocaleProvider>{node}</LocaleProvider>

const buttonByText = (text: string) =>
  Array.from(document.querySelectorAll('button')).find((b) => b.textContent?.includes(text)) as HTMLButtonElement

const okView = {
  local_searxng_url: '',
  tavily_api_key_env: 'TAVILY_API_KEY',
  brave_api_key_env: '',
  tavily_ready: true,
  brave_ready: false,
  searxng_ready: false,
  has_primary: true,
}

describe('SearchPanel 搜索主源设置卡（刀3）', () => {
  beforeAll(async () => {
    const { loadLocale } = await import('../../gaea/lib/i18n')
    await loadLocale('zh')
  })

  beforeEach(() => {
    vi.clearAllMocks()
    settingsMocks.webSearchSettings.mockResolvedValue(okView)
    settingsMocks.saveWebSearchSettings.mockResolvedValue(undefined)
  })

  it('读取后展示可用性徽标（主源已配置）', async () => {
    render(wrap(<SearchPanel />))
    await waitFor(() => expect(screen.getByText('主源已配置')).toBeTruthy())
    expect(screen.getByText('Tavily 已就绪')).toBeTruthy()
    expect(screen.getByText('Brave 未配置')).toBeTruthy()
    // 密钥不回显：占位提示已配置
    expect(screen.getByPlaceholderText('已配置（输入新 key 可替换；留空清除）')).toBeTruthy()
  })

  it('保存走 saveWebSearchSettings 并透传三入参', async () => {
    render(wrap(<SearchPanel />))
    await waitFor(() => expect(screen.getByText('主源已配置')).toBeTruthy())
    fireEvent.change(screen.getByPlaceholderText('http://localhost:8080'), { target: { value: 'http://localhost:9999' } })
    fireEvent.click(buttonByText('保存'))
    await waitFor(() => expect(settingsMocks.saveWebSearchSettings).toHaveBeenCalled())
    const args = settingsMocks.saveWebSearchSettings.mock.calls[0] as unknown[]
    expect(args[2]).toBe('http://localhost:9999')
  })

  it('测试连接 ok 态展示引擎与耗时', async () => {
    settingsMocks.testWebSearchConnection.mockResolvedValue({
      ok: true, state: 'ok', elapsed_ms: 1200,
      engines: [{ name: 'tavily', outcome: 'ok', elapsed_ms: 1200 }],
      sample: ['正题结果（tavily）'], error: '', hint: '',
    })
    render(wrap(<SearchPanel />))
    await waitFor(() => expect(screen.getByText('主源已配置')).toBeTruthy())
    fireEvent.click(buttonByText('测试连接'))
    await waitFor(() => expect(screen.getByText(/连通正常（ok，1200ms）/)).toBeTruthy())
    expect(screen.getByText(/tavily:ok\(1200ms\)/)).toBeTruthy()
  })

  it('测试连接 degraded 态给出警告级降级标注', async () => {
    settingsMocks.testWebSearchConnection.mockResolvedValue({
      ok: true, state: 'degraded', degraded_reason: 'primary-timeout', elapsed_ms: 4600,
      engines: [{ name: 'bing', outcome: 'ok', elapsed_ms: 700 }],
      sample: [], error: '', hint: '',
    })
    render(wrap(<SearchPanel />))
    await waitFor(() => expect(screen.getByText('主源已配置')).toBeTruthy())
    fireEvent.click(buttonByText('测试连接'))
    await waitFor(() => expect(screen.getByText(/连通正常（degraded：primary-timeout/)).toBeTruthy())
  })

  it('测试连接全败给出配置指引', async () => {
    settingsMocks.testWebSearchConnection.mockResolvedValue({
      ok: false, state: 'all_failed', elapsed_ms: 8000,
      engines: [{ name: 'bing', outcome: 'error', elapsed_ms: 8000 }],
      sample: [], error: '所有搜索引擎失败',
      hint: '在上方填入 Tavily（tavily.com 免费注册）/ Brave key 或自建 SearXNG 地址后保存重试；免费兜底（Bing）仍可用。',
    })
    render(wrap(<SearchPanel />))
    await waitFor(() => expect(screen.getByText('主源已配置')).toBeTruthy())
    fireEvent.click(buttonByText('测试连接'))
    await waitFor(() => expect(screen.getByText(/未获得可用结果/)).toBeTruthy())
    expect(screen.getByText(/在上方填入 Tavily/)).toBeTruthy()
  })

  it('读取失败展示错误横幅不静默', async () => {
    settingsMocks.webSearchSettings.mockRejectedValue(new Error('配置不可读'))
    render(wrap(<SearchPanel />))
    await waitFor(() => expect(screen.getByText(/读取搜索设置失败：配置不可读/)).toBeTruthy())
  })
})
