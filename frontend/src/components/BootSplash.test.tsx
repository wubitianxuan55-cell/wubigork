import { act, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { LocaleProvider } from '../gaea/lib/i18n'
import BootSplash from './BootSplash'

function renderSplash() {
  return render(
    <LocaleProvider>
      <BootSplash />
    </LocaleProvider>,
  )
}

function setReducedMotion(matches: boolean) {
  vi.spyOn(window, 'matchMedia').mockImplementation((query: string) => ({
    matches,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  }))
}

describe('BootSplash', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    localStorage.clear()
    localStorage.setItem('gaea-lang', 'zh')
  })

  afterEach(() => {
    vi.clearAllTimers()
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('接管静态启动屏并展示大地创生启动序列', () => {
    const staticSplash = document.createElement('div')
    staticSplash.id = 'boot-splash'
    document.body.appendChild(staticSplash)

    renderSplash()

    expect(document.getElementById('boot-splash')).toBeNull()
    expect(screen.getByTestId('boot-splash')).not.toBeNull()
    expect(screen.getAllByText('REQUESTING ACCESS').length).toBeGreaterThan(0)
    expect(screen.getByRole('button', { name: /跳过启动.*ESC/i })).not.toBeNull()
  })

  it('Escape 立即结束启动序列', () => {
    renderSplash()

    act(() => {
      fireEvent.keyDown(window, { key: 'Escape' })
    })

    expect(screen.getByTestId('boot-splash').classList.contains('is-leaving')).toBe(true)
    act(() => {
      vi.advanceTimersByTime(440)
    })
    expect(screen.queryByTestId('boot-splash')).toBeNull()
  })

  it('跳过按钮提供可见且可点击的退出路径', () => {
    renderSplash()

    fireEvent.click(screen.getByRole('button', { name: /跳过启动.*ESC/i }))
    expect(screen.getByTestId('boot-splash').classList.contains('is-leaving')).toBe(true)

    act(() => {
      vi.advanceTimersByTime(440)
    })
    expect(screen.queryByTestId('boot-splash')).toBeNull()
  })

  it('正常启动在计时结束后自动卸载', () => {
    renderSplash()

    act(() => {
      vi.advanceTimersByTime(7400)
    })
    expect(screen.getByTestId('boot-splash').classList.contains('is-leaving')).toBe(true)

    act(() => {
      vi.advanceTimersByTime(920)
    })
    expect(screen.queryByTestId('boot-splash')).toBeNull()
  })

  it('reduced-motion 下直接进入就绪态并快速退出', () => {
    setReducedMotion(true)
    renderSplash()

    expect(screen.getAllByText('大地终端已就绪').length).toBeGreaterThan(0)
    act(() => {
      vi.advanceTimersByTime(760)
    })
    expect(screen.getByTestId('boot-splash').classList.contains('is-leaving')).toBe(true)
  })

  it('按权限验证到大地终端的两幕节奏推进', () => {
    renderSplash()

    act(() => {
      vi.advanceTimersByTime(900)
    })
    expect(screen.getByText('PERMISSION AUTHORIZED')).not.toBeNull()

    act(() => {
      vi.advanceTimersByTime(900)
    })
    expect(screen.getAllByRole('heading', { name: 'GAEA' }).length).toBeGreaterThan(0)
    expect(screen.getByText('TERRA INTELLIGENCE')).not.toBeNull()
  })
})
