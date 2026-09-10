/**
 * StrategySection.test.tsx — 模型中心「空间策略」（总闸画面）分区测试
 *
 * 断言焦点：双空间卡渲染与当前生效徽标、gaea 覆写三态（未配置/解析/坏引用
 * 说人话）、play 护栏行、space.mode=off 警示条、读取失败诚实错误卡+重试。
 * bridge app facade 按既有 mock factory 惯例打桩。
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { StrategySection } from './StrategySection'

const mocks = vi.hoisted(() => ({
  GaeaSpaceProfiles: vi.fn(),
  GaeaSpaceActive: vi.fn(),
}))

vi.mock('../../gaea/lib/bridge', () => ({
  app: {
    GaeaSpaceProfiles: mocks.GaeaSpaceProfiles,
    GaeaSpaceActive: mocks.GaeaSpaceActive,
  },
}))

const workView = {
  space: 'work', gaea: '', gaeaResolved: '', gaeaOk: false, models: {},
  permMode: 'ask', permHardAskCount: 0, permHardAskBySpace: false,
  guardrailsOn: false, modeOn: true,
}
const playView = {
  space: 'play', gaea: 'glm/glm-5.3', gaeaResolved: 'glm · glm-5.3', gaeaOk: true,
  models: { novel: 'glm/glm-5.3-air' },
  permMode: 'allow', permHardAskCount: 0, permHardAskBySpace: true,
  guardrailsOn: true, modeOn: true,
}

describe('模型中心「空间策略」分区（总闸画面）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.GaeaSpaceActive.mockResolvedValue({ space: 'play', modeOn: true })
  })

  it('渲染双空间卡：当前生效徽标 + gaea 覆写解析显示 + 其余域 chips + play 护栏行', async () => {
    mocks.GaeaSpaceProfiles.mockResolvedValue([workView, playView])
    render(<StrategySection />)
    await waitFor(() => expect(screen.getByTestId('mc-strategy-work')).toBeTruthy())
    // play=当前生效；work 未生效
    const play = screen.getByTestId('mc-strategy-play')
    expect(play.textContent).toContain('当前生效')
    expect(screen.getByTestId('mc-strategy-work').textContent).toContain('未生效')
    // gaea 三态：work 未配置（现状），play 解析显示 ref → resolved
    expect(screen.getByTestId('mc-strategy-work-gaea').textContent).toContain('未配置（维持现状模型）')
    expect(screen.getByTestId('mc-strategy-play-gaea').textContent).toContain('glm/glm-5.3')
    expect(screen.getByTestId('mc-strategy-play-gaea').textContent).toContain('glm · glm-5.3')
    // 其余功能域覆写 chips + play 护栏
    expect(play.textContent).toContain('小说：glm/glm-5.3-air')
    expect(play.textContent).toContain('生效（钳制中）')
    expect(play.textContent).toContain('按空间配置（0 项）')
    // work 无护栏行（仅 play 显示）
    expect(screen.getByTestId('mc-strategy-work').textContent).not.toContain('内容护栏')
  })

  it('坏引用说人话：gaeaOk=false 且非空 → 显「无法解析，现状模型继续生效」', async () => {
    mocks.GaeaSpaceProfiles.mockResolvedValue([{
      ...workView, gaea: 'nope/none',
    }])
    render(<StrategySection />)
    await waitFor(() => expect(screen.getByTestId('mc-strategy-work-gaea').textContent)
      .toContain('无法解析，现状模型继续生效'))
  })

  it('space.mode=off → 警示条（策略不生效，全域回退 work 现状）', async () => {
    mocks.GaeaSpaceProfiles.mockResolvedValue([{ ...workView, modeOn: false }])
    render(<StrategySection />)
    await waitFor(() => expect(screen.getByText(/space.mode=off/)).toBeTruthy())
  })

  it('读取失败 → 诚实错误卡，重试成功后恢复', async () => {
    mocks.GaeaSpaceProfiles.mockRejectedValueOnce(new Error('绑定不可用（旧后端）'))
    render(<StrategySection />)
    await waitFor(() => expect(screen.getByTestId('mc-strategy-error')).toBeTruthy())
    expect(screen.getByTestId('mc-strategy-error').textContent).toContain('绑定不可用（旧后端）')

    // 重试成功恢复双空间卡
    mocks.GaeaSpaceProfiles.mockResolvedValue([workView, playView])
    fireEvent.click(screen.getByText('重试'))
    await waitFor(() => expect(screen.getByTestId('mc-strategy-play')).toBeTruthy())
  })
})
