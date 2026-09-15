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
// v9 案头重设计追加断言：书斋 masthead 身份区（h1.w-mast-title 文书台 / w-mast-lede /
// w-seal 印章 / w-mast-side 内空间 chip + w-mast-pill 就绪徽记）、能力目录改案牌
// 网格（w-plaques / w-plaque 按钮，行式 w-index-item 退役）、脉息面板四节
// （w-vitals 内 4×w-vital，aria-label 用 zh 精确文案）；命令台 w-cmd 保留而
// v8 刊头行 w-deck-head 删除；闲庭版式不受 v9 影响。

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

describe('书斋 v9 案头版式', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('masthead 身份区——文书台大标 + 印章装饰 + 就绪徽记（v8 刊头行 w-deck-head 退役）', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="work" onSwitchSpace={vi.fn()} />))
    // 大标：h1.w-mast-title = home.title「文书台」
    const title = screen.getByText('文书台')
    expect(title.tagName.toLowerCase()).toBe('h1')
    expect(title.classList.contains('w-mast-title')).toBe(true)
    // 题辞：w-mast-lede 承载 home.sub 长文案（锁存在 + 非空，不锁全串避免实现侧标点级脆断）
    const lede = document.querySelector('.w-mast-lede')
    expect(lede).toBeTruthy()
    expect((lede?.textContent ?? '').length).toBeGreaterThan(0)
    // 印章：纯装饰（aria-hidden），内容为空间名首字「书」
    const seal = document.querySelector('.w-seal')
    expect(seal).toBeTruthy()
    expect(seal?.getAttribute('aria-hidden')).toBe('true')
    expect(seal?.textContent).toBe('书')
    // 侧翼：既有空间 chip 落位 w-mast-side，就绪徽记 = home.pill（zh.ts 精确串）
    const side = document.querySelector('.w-mast-side')
    expect(side).toBeTruthy()
    expect(side?.querySelector('[data-testid="ml-space-chip"]')).toBeTruthy()
    expect(screen.getByText('GAEA 已就绪 · 本地 AI 创作中枢')).toBeTruthy()
    // v8 刊头行删除；命令台本体保留
    expect(document.querySelector('.w-deck-head')).toBeNull()
    expect(document.querySelector('.w-cmd')).toBeTruthy()
  })

  it('能力目录为案牌网格，行式索引 w-index-item 不再出现', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="work" onSwitchSpace={vi.fn()} />))
    expect(document.querySelector('.w-plaques')).toBeTruthy()
    // fallback 清单派生：静态 canonicalBoards 在 work 空间可得 6 模块
    // （gaea/cost/memoryhub/weixin/modelcenter/settings）；旗舰是否单列由实现定
    // → 宽松断言只锁「非空 + 全部为按钮」。
    const plaques = document.querySelectorAll('.w-plaque')
    expect(plaques.length).toBeGreaterThan(0)
    plaques.forEach((p) => expect(p.tagName.toLowerCase()).toBe('button'))
    expect(document.querySelector('.w-index-item')).toBeNull()
  })

  it('脉息面板四节齐备（写作/内核/会话/记忆，aria-label 对齐 zh 精确文案）', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="work" onSwitchSpace={vi.fn()} />))
    expect(document.querySelector('.w-vitals')).toBeTruthy()
    const vitals = document.querySelectorAll('.w-vital')
    expect(vitals.length).toBe(4)
    // zh.ts 精确键值：shell.launcher.statWriting=项目写作进度（非简称「写作进度」）、
    // home.kernel=内核状态、shell.launcher.sessions=最近会话、
    // shell.launcher.memoryPulse=记忆脉搏
    const labels = Array.from(vitals).map((el) => el.getAttribute('aria-label'))
    expect(labels).toContain('项目写作进度')
    expect(labels).toContain('内核状态')
    expect(labels).toContain('最近会话')
    expect(labels).toContain('记忆脉搏')
  })

  it('闲庭不受 v9 影响', () => {
    render(wrap(<ModuleLauncher onNavigate={vi.fn()} space="play" onSwitchSpace={vi.fn()} />))
    expect(document.querySelector('.w-masthead')).toBeNull()
    expect(document.querySelector('.w-plaques')).toBeNull()
    expect(document.querySelector('.garden-banner')).toBeTruthy()
  })
})
