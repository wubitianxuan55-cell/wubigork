/**
 * StrategySection.test.tsx — 模型中心「空间策略」（总闸画面）分区测试
 *
 * 断言焦点：双空间卡渲染与当前生效徽标、gaea 覆写三态（未配置/解析/坏引用
 * 说人话）、其余功能域六键常显可编、play 护栏行、space.mode=off 警示条、
 * 读取失败诚实错误卡+重试。bridge app facade 按既有 mock factory 惯例打桩。
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { StrategySection } from './StrategySection'

const mocks = vi.hoisted(() => ({
  GaeaSpaceProfiles: vi.fn(),
  GaeaSpaceActive: vi.fn(),
  GaeaSpaceProfileSet: vi.fn(),
}))

vi.mock('../../gaea/lib/bridge', () => ({
  app: {
    GaeaSpaceProfiles: mocks.GaeaSpaceProfiles,
    GaeaSpaceActive: mocks.GaeaSpaceActive,
    GaeaSpaceProfileSet: mocks.GaeaSpaceProfileSet,
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
    // 其余功能域常显：已配小说 + 空键「未配置」；play 护栏
    expect(screen.getByTestId('mc-strategy-play-novel').textContent).toContain('glm/glm-5.3-air')
    expect(screen.getByTestId('mc-strategy-work-chat').textContent).toContain('未配置（维持现状）')
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

  it('编辑闭环：点编辑带出当前值 → 保存调 GaeaSpaceProfileSet → 视图随返回刷新', async () => {
    mocks.GaeaSpaceProfiles.mockResolvedValue([workView, playView])
    mocks.GaeaSpaceProfileSet.mockResolvedValue([
      workView,
      { ...playView, gaea: 'prov/m2', gaeaResolved: 'prov · m2' },
    ])
    render(<StrategySection />)
    await waitFor(() => expect(screen.getByTestId('mc-strategy-play')).toBeTruthy())

    fireEvent.click(screen.getByTestId('mc-strategy-play-edit'))
    const input = screen.getByTestId('mc-strategy-play-input') as HTMLInputElement
    expect(input.value).toBe('glm/glm-5.3') // 带出当前值
    fireEvent.change(input, { target: { value: 'prov/m2' } })
    fireEvent.click(screen.getByTestId('mc-strategy-play-save'))

    await waitFor(() => expect(mocks.GaeaSpaceProfileSet).toHaveBeenCalledWith('play', 'gaea', 'prov/m2'))
    await waitFor(() => expect(screen.getByTestId('mc-strategy-play-gaea').textContent).toContain('prov · m2'))
    await waitFor(() => expect(screen.getByText('已写入，下次引擎重建/重启生效')).toBeTruthy())
  })

  it('写入失败说人话：后端校验错误原样 message.error，不吞', async () => {
    mocks.GaeaSpaceProfiles.mockResolvedValue([workView, playView])
    mocks.GaeaSpaceProfileSet.mockRejectedValue(new Error('无法解析 "nope/x"——已配置 provider: prov'))
    render(<StrategySection />)
    await waitFor(() => expect(screen.getByTestId('mc-strategy-work-edit')).toBeTruthy())

    fireEvent.click(screen.getByTestId('mc-strategy-work-edit'))
    fireEvent.change(screen.getByTestId('mc-strategy-work-input'), { target: { value: 'nope/x' } })
    fireEvent.click(screen.getByTestId('mc-strategy-work-save'))

    await waitFor(() => expect(screen.getByText(/无法解析 "nope.x"/)).toBeTruthy())
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

  it('其余功能域编辑闭环：点小说编辑带出当前值 → 保存调 Set(play, novel, …)', async () => {
    mocks.GaeaSpaceProfiles.mockResolvedValue([workView, playView])
    mocks.GaeaSpaceProfileSet.mockResolvedValue([
      workView,
      { ...playView, models: { novel: 'prov/air2' } },
    ])
    render(<StrategySection />)
    await waitFor(() => expect(screen.getByTestId('mc-strategy-play-novel-edit')).toBeTruthy())

    fireEvent.click(screen.getByTestId('mc-strategy-play-novel-edit'))
    const input = screen.getByTestId('mc-strategy-play-novel-input') as HTMLInputElement
    expect(input.value).toBe('glm/glm-5.3-air')
    fireEvent.change(input, { target: { value: 'prov/air2' } })
    fireEvent.click(screen.getByTestId('mc-strategy-play-novel-save'))

    await waitFor(() => expect(mocks.GaeaSpaceProfileSet).toHaveBeenCalledWith('play', 'novel', 'prov/air2'))
    await waitFor(() => expect(screen.getByTestId('mc-strategy-play-novel').textContent).toContain('prov/air2'))
  })

  it('其余功能域清除：空串保存 → Set(play, novel, "") 行回未配置', async () => {
    mocks.GaeaSpaceProfiles.mockResolvedValue([workView, playView])
    mocks.GaeaSpaceProfileSet.mockResolvedValue([
      workView,
      { ...playView, models: {} },
    ])
    render(<StrategySection />)
    await waitFor(() => expect(screen.getByTestId('mc-strategy-play-novel-edit')).toBeTruthy())

    fireEvent.click(screen.getByTestId('mc-strategy-play-novel-edit'))
    fireEvent.change(screen.getByTestId('mc-strategy-play-novel-input'), { target: { value: '' } })
    fireEvent.click(screen.getByTestId('mc-strategy-play-novel-save'))

    await waitFor(() => expect(mocks.GaeaSpaceProfileSet).toHaveBeenCalledWith('play', 'novel', ''))
    await waitFor(() => expect(screen.getByTestId('mc-strategy-play-novel').textContent).toContain('未配置（维持现状）'))
  })
})
