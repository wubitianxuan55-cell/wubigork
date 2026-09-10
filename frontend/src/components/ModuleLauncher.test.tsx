import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import type { ReactElement } from 'react'
import ModuleLauncher from './ModuleLauncher'
import { LocaleProvider } from '../gaea/lib/i18n'

// ModuleLauncher 双首页（v4.182）：空间切换器迁首页顶栏 + 书斋/闲庭两变体版式。
// 断言焦点：切换器（SHELL_SPACES 驱动、aria-pressed、点击回调）与两空间
// 版式分化（书斋=文档流水面板；闲庭=旗舰横幅画廊+无遥测右舷）。
// 数据 hooks（遥测/会话/记忆）后端未就绪时静默兜底，无需 mock；语音 hook 整体
// mock（真实实现依赖 Wails EventsOn）。

const bridgeMocks = vi.hoisted(() => ({
  app: {
    ListSessions: vi.fn(async () => []),
    MemoryHubOverview: vi.fn(async () => ({})),
    VoiceApplySettings: vi.fn(async () => ({})),
    VoiceChatText: vi.fn(async () => ({})),
  },
}))

vi.mock('../gaea/lib/bridge', () => ({ app: bridgeMocks.app }))
vi.mock('../hooks/useVoiceChat', () => ({
  useVoiceChat: () => ({
    state: { active: false, listening: false, aiSpeaking: false, error: null },
    start: vi.fn(),
    stop: vi.fn(),
    interrupt: vi.fn(),
  }),
}))
vi.mock('../gaea/components/MorningBriefCard', () => ({ default: () => null }))

const wrap = (ui: ReactElement) => {
  localStorage.setItem('gaea-lang', 'zh')
  return <LocaleProvider>{ui}</LocaleProvider>
}

describe('ModuleLauncher 双空间首页（v4.182 书斋/闲庭）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('顶栏渲染空间切换器（SHELL_SPACES 驱动：当前空间 aria-pressed，点击另一空间回调 onSwitchSpace）', () => {
    const onSwitch = vi.fn()
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="work" onSwitchSpace={onSwitch} />))
    const work = screen.getByTestId('ml-space-work')
    const play = screen.getByTestId('ml-space-play')
    expect(work.textContent).toBe('书斋')
    expect(play.textContent).toBe('闲庭')
    expect(work.getAttribute('aria-pressed')).toBe('true')
    expect(play.getAttribute('aria-pressed')).toBe('false')
    fireEvent.click(play)
    expect(onSwitch).toHaveBeenCalledWith('play')
    // 当前空间按钮点击不重复触发
    fireEvent.click(work)
    expect(onSwitch).toHaveBeenCalledTimes(1)
  })

  it('1B 前置：切换钮与空间 chip 的 title 说明「仅导航，不影响办公引擎空间」', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="work" onSwitchSpace={vi.fn()} />))
    const hint = '不影响办公引擎空间'
    expect(screen.getByTestId('ml-space-work').getAttribute('title')).toContain(hint)
    expect(screen.getByTestId('ml-space-play').getAttribute('title')).toContain(hint)
    expect(screen.getByTestId('ml-space-chip').getAttribute('title')).toContain(hint)
  })

  it('书斋（work）：文档流水面板为主角 + 空间 chip；闲庭板块卡不出现', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="work" onSwitchSpace={vi.fn()} />))
    expect(screen.getByTestId('desk-recent-docs')).toBeTruthy()
    expect(screen.getByTestId('ml-space-chip').textContent).toBe('书斋')
    // 闲庭版式元素不在书斋出现
    expect(screen.queryByTestId('garden-banner-impl')).toBeNull()
  })

  it('闲庭（play）：全幅画廊——旗舰横幅 + 激活态，书斋文档流水不出现', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="play" onSwitchSpace={vi.fn()} />))
    const play = screen.getByTestId('ml-space-play')
    expect(play.getAttribute('aria-pressed')).toBe('true')
    // 闲庭版式：会客厅旗舰横幅在画廊渲染；书斋专有元素（文档流水/chip）不渲染
    expect(document.querySelector('.garden-banner')).toBeTruthy()
    expect(screen.queryByTestId('desk-recent-docs')).toBeNull()
    expect(screen.queryByTestId('ml-space-chip')).toBeNull()
  })
})
