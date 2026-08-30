/**
 * SecurityPanel.test.tsx — 全局离线模式开关（v4.8.1）
 *
 * 加载回填 GetOfflineMode；切换调用 SetOfflineMode 且传新值；
 * 保存失败回滚到旧值并提示（不静默）。
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

// SecurityPanel 走裸 window.go.app.App（AppFacade 宽类型），测试直接注入 window.go
const setOfflineMode = vi.fn()
const getOfflineMode = vi.fn()
const getOfficeLocal = vi.fn()
const getSensitiveLocal = vi.fn()
const herdsmanSecurityCheck = vi.fn()

type GoStub = {
  GetOfflineMode?: () => Promise<boolean>
  SetOfflineMode?: (v: boolean) => Promise<void>
  GetOfficeLocal?: () => Promise<boolean>
  GetSensitiveLocal?: () => Promise<boolean>
  HerdsmanSecurityCheck?: () => Promise<unknown>
}

const setGo = (stub: GoStub) => {
  ;(window as unknown as { go?: { app?: { App?: GoStub } } }).go = { app: { App: stub } }
}

import SecurityPanel from './SecurityPanel'

beforeEach(() => {
  vi.clearAllMocks()
  getOfflineMode.mockResolvedValue(false)
  getOfficeLocal.mockResolvedValue(true)
  getSensitiveLocal.mockResolvedValue(true)
  herdsmanSecurityCheck.mockResolvedValue(undefined)
  setOfflineMode.mockResolvedValue(undefined)
  setGo({ GetOfflineMode: getOfflineMode, SetOfflineMode: setOfflineMode, GetOfficeLocal: getOfficeLocal, GetSensitiveLocal: getSensitiveLocal, HerdsmanSecurityCheck: herdsmanSecurityCheck })
})

describe('SecurityPanel 全局离线模式（v4.8.1）', () => {
  it('加载回填后端开关状态', async () => {
    getOfflineMode.mockResolvedValue(true)
    render(<SecurityPanel />)
    await waitFor(() => {
      expect(getOfflineMode).toHaveBeenCalled()
    })
    // 回填 true → 开关文案为「仅本地（离线）」
    expect(await screen.findByText('仅本地（离线）')).toBeTruthy()
  })

  it('切换开关调用 SetOfflineMode 且传新值', async () => {
    render(<SecurityPanel />)
    // 等待初始加载完成（默认 false → 文案「关闭（默认）」）
    await screen.findByText('关闭（默认）')
    // 面板里第四个 Switch 是离线模式（前三个：敏感域/办公/离线——离线是第 3 个）
    const switches = document.querySelectorAll('button[role="switch"]')
    fireEvent.click(switches[2])
    await waitFor(() => {
      expect(setOfflineMode).toHaveBeenCalledWith(true)
    })
  })

  it('保存失败回滚旧值并提示（不静默）', async () => {
    setOfflineMode.mockRejectedValue(new Error('写盘失败'))
    render(<SecurityPanel />)
    await screen.findByText('关闭（默认）')
    const switches = document.querySelectorAll('button[role="switch"]')
    fireEvent.click(switches[2])
    // 失败后回滚：开关文案回到「关闭（默认）」且出现错误提示
    expect(await screen.findByText('写盘失败')).toBeTruthy()
    expect(screen.getByText('关闭（默认）')).toBeTruthy()
  })
})
