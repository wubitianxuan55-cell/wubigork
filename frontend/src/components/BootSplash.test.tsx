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

function startVoice(container: HTMLElement) {
  fireEvent(container.querySelector('audio')!, new Event('playing'))
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

  it('接管静态启动屏并展示地核生成启动序列', () => {
    const staticSplash = document.createElement('div')
    staticSplash.id = 'boot-splash'
    document.body.appendChild(staticSplash)

    renderSplash()

    expect(document.getElementById('boot-splash')).toBeNull()
    expect(screen.getByTestId('boot-splash')).not.toBeNull()
    expect(screen.getByRole('heading', { name: 'GAEA' })).not.toBeNull()
    expect(screen.getByText('正在启动 · 连接智能世界…')).not.toBeNull()
    expect(screen.getByRole('button', { name: /跳过启动.*ESC/i })).not.toBeNull()
    expect(document.querySelector('.bs-original-scene--earth')).not.toBeNull()
    expect(document.querySelector('.bs-original-scene--gaia')).not.toBeNull()
    expect(document.querySelectorAll('.bs-original-scene img')).toHaveLength(2)
    expect(screen.getByRole('progressbar').getAttribute('aria-valuenow')).toBe('68')
    expect(document.querySelector('canvas')).toBeNull()
  })

  it('Escape 立即结束启动序列', () => {
    renderSplash()

    act(() => {
      fireEvent.keyDown(window, { key: 'Escape' })
    })

    expect(screen.getByTestId('boot-splash').classList.contains('is-leaving')).toBe(true)
    act(() => {
      vi.advanceTimersByTime(420)
    })
    expect(screen.queryByTestId('boot-splash')).toBeNull()
  })

  it('启动过程中自动从天空场景过渡到人间场景', () => {
    const { container } = renderSplash()
    const splash = screen.getByTestId('boot-splash')

    expect(splash.getAttribute('data-scene')).toBe('earth')
    expect(screen.getByText('YOUR INTELLIGENT WORK ASSISTANT')).not.toBeNull()

    startVoice(container)
    act(() => {
      vi.advanceTimersByTime(1596)
    })

    expect(splash.getAttribute('data-scene')).toBe('gaia')
    expect(screen.getByRole('progressbar').getAttribute('aria-valuenow')).toBe('72')
    expect(screen.getByText('EMPOWER YOUR INTELLIGENCE')).not.toBeNull()
  })

  it('跳过按钮提供可见且可点击的退出路径', () => {
    renderSplash()

    fireEvent.click(screen.getByRole('button', { name: /跳过启动.*ESC/i }))
    expect(screen.getByTestId('boot-splash').classList.contains('is-leaving')).toBe(true)

    act(() => {
      vi.advanceTimersByTime(420)
    })
    expect(screen.queryByTestId('boot-splash')).toBeNull()
  })

  it('声音入口可见，首次点击会尝试启动启动音效', () => {
    const play = vi.spyOn(window.HTMLMediaElement.prototype, 'play')
      .mockReturnValue(undefined as unknown as Promise<void>)
    const { container } = renderSplash()
    const voice = container.querySelector('audio')!
    Object.defineProperty(voice, 'readyState', { configurable: true, value: 2 })

    const sound = screen.getByRole('button', { name: '开启启动音效' })
    expect(sound.textContent).toContain('PLAY')

    fireEvent.click(sound)
    expect(play).toHaveBeenCalledTimes(1)
    expect(sound.getAttribute('aria-label')).toContain('静音')
  })

  it('正常启动在计时结束后自动卸载', () => {
    const { container } = renderSplash()
    startVoice(container)

    act(() => {
      vi.advanceTimersByTime(4300)
    })
    expect(screen.getByTestId('boot-splash').classList.contains('is-leaving')).toBe(true)

    act(() => {
      vi.advanceTimersByTime(520)
    })
    expect(screen.queryByTestId('boot-splash')).toBeNull()
  })

  it('reduced-motion 只关闭动画，不跳过完整启动时序', () => {
    setReducedMotion(true)
    const { container } = renderSplash()
    startVoice(container)

    act(() => {
      vi.advanceTimersByTime(760)
    })
    expect(screen.getByTestId('boot-splash')).not.toBeNull()
    expect(screen.getByTestId('boot-splash').classList.contains('is-leaving')).toBe(false)

    act(() => {
      vi.advanceTimersByTime(836)
    })
    expect(screen.getByText('正在启动 · 连接智能世界…')).not.toBeNull()

    act(() => {
      fireEvent(container.querySelector('audio')!, new Event('ended'))
    })
    expect(screen.getByTestId('boot-splash').classList.contains('is-leaving')).toBe(true)
    act(() => {
      vi.advanceTimersByTime(180)
    })
    expect(screen.queryByTestId('boot-splash')).toBeNull()
  })

  it('按两段语音推进状态文案', () => {
    const { container } = renderSplash()
    startVoice(container)

    act(() => {
      vi.advanceTimersByTime(1596)
    })
    expect(screen.getByText('正在启动 · 连接智能世界…')).not.toBeNull()
    expect(screen.getAllByRole('heading', { name: 'GAEA' }).length).toBe(1)
  })

  it('语音自然结束即收束，不等待兜底计时器', () => {
    const { container } = renderSplash()
    const audio = container.querySelector('audio')
    expect(audio?.getAttribute('src')).toBe('/boot-gaea-voice-selected.wav')

    act(() => {
      fireEvent(audio!, new Event('ended'))
    })
    expect(screen.getByTestId('boot-splash').classList.contains('is-leaving')).toBe(true)
  })
})
