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

/** Makes the jsdom audio element look like a voice that is really playing. */
function mockPlayingVoice(container: HTMLElement, readPosition: () => number): HTMLAudioElement {
  const voice = container.querySelector('audio')!
  Object.defineProperty(voice, 'paused', { configurable: true, value: false })
  Object.defineProperty(voice, 'readyState', { configurable: true, value: 4 })
  Object.defineProperty(voice, 'duration', { configurable: true, value: 4 })
  let position = readPosition()
  Object.defineProperty(voice, 'currentTime', {
    configurable: true,
    get: () => position,
    set: (value: number) => {
      position = value
    },
  })
  return voice
}

const progress = () => screen.getByRole('progressbar').getAttribute('aria-valuenow')
const splash = () => screen.getByTestId('boot-splash')

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

  it('静态首屏等底片清晰后才撤下，然后展示两幕底片、新标与品牌文案', () => {
    const staticSplash = document.createElement('div')
    staticSplash.id = 'boot-splash'
    document.body.appendChild(staticSplash)

    const { container } = renderSplash()

    // The blurred first paint stays until the sharp plate is in, and the splash stays see-through underneath it.
    expect(document.getElementById('boot-splash')).not.toBeNull()
    expect(staticSplash.classList.contains('is-handing-off')).toBe(false)
    expect(splash().classList.contains('is-over-static')).toBe(true)

    fireEvent.load(screen.getByTestId('boot-poster-earth'))
    expect(staticSplash.classList.contains('is-handing-off')).toBe(true)
    expect(splash().classList.contains('is-over-static')).toBe(false)
    act(() => {
      vi.advanceTimersByTime(300)
    })
    expect(document.getElementById('boot-splash')).toBeNull()

    expect(splash()).not.toBeNull()
    expect(screen.getByTestId('boot-poster-earth').getAttribute('src')).toBe('/boot/plate-orbit.avif')
    expect(screen.getByTestId('boot-poster-gaia').getAttribute('src')).toBe('/boot/plate-valley.avif')
    expect(container.querySelector('canvas')).not.toBeNull()
    expect(screen.getByRole('heading', { name: 'GAEA' })).not.toBeNull()
    expect(screen.getByText('YOUR INTELLIGENT WORK ASSISTANT')).not.toBeNull()
    expect(screen.getByText('EMPOWER YOUR INTELLIGENCE')).not.toBeNull()
    expect(screen.getByText('正在启动 · 连接智能世界…')).not.toBeNull()
    expect(screen.getByRole('button', { name: /跳过启动.*ESC/i })).not.toBeNull()
    expect(progress()).toBe('0')
  })

  it('Escape 立即结束启动序列', () => {
    renderSplash()

    act(() => {
      fireEvent.keyDown(window, { key: 'Escape' })
    })

    expect(splash().classList.contains('is-leaving')).toBe(true)
    act(() => {
      vi.advanceTimersByTime(420)
    })
    expect(screen.queryByTestId('boot-splash')).toBeNull()
  })

  it('启动过程中自动从天空场景过渡到人间场景，进度沿 68 → 72 推进', () => {
    const { container } = renderSplash()
    expect(splash().getAttribute('data-scene')).toBe('earth')

    startVoice(container)
    act(() => {
      vi.advanceTimersByTime(1100)
    })
    expect(progress()).toBe('68')
    expect(splash().getAttribute('data-scene')).toBe('earth')

    act(() => {
      vi.advanceTimersByTime(520)
    })
    expect(splash().getAttribute('data-scene')).toBe('gaia')
    expect(progress()).toBe('70')

    act(() => {
      vi.advanceTimersByTime(684)
    })
    expect(progress()).toBe('72')
  })

  it('画面跟着语音自己的播放位置走：主线程再忙，换景也落在语音的第二句上', () => {
    const { container } = renderSplash()
    // The voice has been playing for a while before the main thread gets to run.
    const voice = mockPlayingVoice(container, () => 1.7)
    startVoice(container)
    act(() => {
      vi.advanceTimersByTime(64)
    })
    expect(splash().getAttribute('data-scene')).toBe('gaia')
    expect(progress()).toBe('70')

    // The voice ends: the close lines up with the last syllable, not with a late timer.
    voice.currentTime = 4
    act(() => {
      fireEvent(voice, new Event('ended'))
      vi.advanceTimersByTime(64)
    })
    expect(splash().classList.contains('is-leaving')).toBe(true)
    expect(Number(progress())).toBeGreaterThanOrEqual(94)
  })

  it('语音晚于宽限线起播时，画面与收束都跟着语音走，不被兜底计时器提前截断', () => {
    const { container } = renderSplash()
    // The fallback timeline has been running since 450ms; its guard would close at 4750ms.
    act(() => {
      vi.advanceTimersByTime(3000)
    })
    let position = 0.2
    const voice = mockPlayingVoice(container, () => position)
    startVoice(container)
    act(() => {
      vi.advanceTimersByTime(32)
    })
    // Wall time 3032ms, but the picture is where the voice is (0.2s → about a third of the
    // 68% beat; the wall clock would already be past 75%).
    expect(Number(progress())).toBeGreaterThan(20)
    expect(Number(progress())).toBeLessThan(50)

    // Past 4.9s of wall time: the old wall-clock guard would already have closed the splash.
    act(() => {
      vi.advanceTimersByTime(1900)
    })
    expect(splash().classList.contains('is-leaving')).toBe(false)

    position = 4
    act(() => {
      fireEvent(voice, new Event('ended'))
    })
    expect(splash().classList.contains('is-leaving')).toBe(true)
  })

  it('语音的 ended 事件丢失时，保护计时器在语音播完后 300ms 内收束', () => {
    const { container } = renderSplash()
    mockPlayingVoice(container, () => 3.9)
    startVoice(container)
    act(() => {
      vi.advanceTimersByTime(32 + 350)
    })
    expect(splash().classList.contains('is-leaving')).toBe(false)
    act(() => {
      vi.advanceTimersByTime(60)
    })
    expect(splash().classList.contains('is-leaving')).toBe(true)
  })

  it('语音没有自动播放时，宽限期后时间轴照常启动', () => {
    renderSplash()
    act(() => {
      vi.advanceTimersByTime(400)
    })
    expect(progress()).toBe('0')
    act(() => {
      vi.advanceTimersByTime(1150)
    })
    expect(progress()).toBe('68')
  })

  it('跳过按钮提供可见且可点击的退出路径', () => {
    renderSplash()

    fireEvent.click(screen.getByRole('button', { name: /跳过启动.*ESC/i }))
    expect(splash().classList.contains('is-leaving')).toBe(true)

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
    expect(splash().classList.contains('is-leaving')).toBe(true)
    expect(Number(progress())).toBeGreaterThanOrEqual(94)

    // The bar tops up to 100 while the splash hands over to the workspace.
    act(() => {
      vi.advanceTimersByTime(300)
    })
    expect(Number(progress())).toBeGreaterThan(94)
    expect(Number(progress())).toBeLessThanOrEqual(100)

    act(() => {
      vi.advanceTimersByTime(220)
    })
    expect(screen.queryByTestId('boot-splash')).toBeNull()
  })

  it('没有 WebGL2 时保持静态底片交叉淡变，不显示实时层', () => {
    renderSplash()

    expect(splash().classList.contains('is-live')).toBe(false)
    expect(splash().getAttribute('data-scene')).toBe('earth')
    expect(screen.getByTestId('boot-poster-gaia')).not.toBeNull()
  })

  it('reduced-motion 只关闭动画，不跳过完整启动时序', () => {
    setReducedMotion(true)
    const { container } = renderSplash()
    expect(splash().classList.contains('is-reduced')).toBe(true)
    startVoice(container)

    act(() => {
      vi.advanceTimersByTime(760)
    })
    expect(screen.getByTestId('boot-splash')).not.toBeNull()
    expect(splash().classList.contains('is-leaving')).toBe(false)

    act(() => {
      vi.advanceTimersByTime(860)
    })
    expect(screen.getByText('正在启动 · 连接智能世界…')).not.toBeNull()
    expect(splash().getAttribute('data-scene')).toBe('gaia')

    act(() => {
      fireEvent(container.querySelector('audio')!, new Event('ended'))
    })
    expect(splash().classList.contains('is-leaving')).toBe(true)
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
    expect(splash().classList.contains('is-leaving')).toBe(true)
  })
})
