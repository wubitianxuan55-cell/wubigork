// CreateNovelModal.test.tsx — 新建小说弹窗（标题/题材/文风是父层 HomePage 受控 state）。
// v4.421.0 收口两条：空标题不再「点 OK 静默无反应」（OK 禁用 + 提交前兜底提示）、
// onCreate 在途不可双击重复提交（本地门闩 + confirmLoading）。
// 坑复训：antd 两字按钮自动插空格（「创 建」）→ 按钮名一律正则。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'

const scanMocks = vi.hoisted(() => ({
  scan: vi.fn(),
}))

// 选题扫榜走 app.NovelScanAnalyze（bridge Proxy）；测试层直接替桩。
vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return { ...actual, app: { NovelScanAnalyze: scanMocks.scan } }
})

import CreateNovelModal from './CreateNovelModal'

type Props = Parameters<typeof CreateNovelModal>[0]

function renderModal(overrides: Partial<Props> = {}) {
  const props: Props = {
    open: true,
    onClose: vi.fn(),
    onCreate: vi.fn().mockResolvedValue(undefined),
    title: '风雪夜归',
    onTitleChange: vi.fn(),
    genre: [],
    onGenreChange: vi.fn(),
    style: [],
    onStyleChange: vi.fn(),
    mature: '',
    onMatureChange: vi.fn(),
    ...overrides,
  }
  const view = render(<CreateNovelModal {...props} />)
  return { props, view }
}

/** OK 按钮（文案在在途态变为「创建中…」，用宽正则抓同一个按钮）。 */
const okButton = () => screen.getByRole('button', { name: /创\s*建/ }) as HTMLButtonElement

describe('CreateNovelModal 新建小说（v4.421.0）', () => {
  it('空标题 / 纯空格标题：OK 按钮禁用，点击不触发 onCreate 也不关窗', () => {
    const { props, view } = renderModal({ title: '' })
    expect(okButton().disabled).toBe(true)

    // 纯空格同样算空标题
    view.rerender(<CreateNovelModal {...props} title="   " />)
    expect(okButton().disabled).toBe(true)

    fireEvent.click(okButton())
    expect(props.onCreate).not.toHaveBeenCalled()
    expect(props.onClose).not.toHaveBeenCalled()

    // 填上标题后恢复可点（props 受控：父层 setState 后回传新 title）
    view.rerender(<CreateNovelModal {...props} title="风雪夜归" />)
    expect(okButton().disabled).toBe(false)
  })

  it('onCreate 在途：OK 转 loading 且禁用，重复点击不重复创建；完成后恢复', async () => {
    let resolveCreate: () => void = () => {}
    const onCreate = vi.fn(() => new Promise<void>((r) => { resolveCreate = r }))
    renderModal({ title: '风雪夜归', onCreate })

    fireEvent.click(okButton())
    await waitFor(() => expect(onCreate).toHaveBeenCalledTimes(1))

    const busy = okButton()
    expect(busy.className).toContain('ant-btn-loading')
    expect(busy.disabled).toBe(true)
    fireEvent.click(busy)
    fireEvent.click(busy)
    await new Promise((r) => setTimeout(r, 0))
    expect(onCreate).toHaveBeenCalledTimes(1)

    await act(async () => { resolveCreate() })
    await waitFor(() => expect(okButton().className).not.toContain('ant-btn-loading'))
    expect(okButton().disabled).toBe(false)
  })

  it('有标题提交：onCreate 恰一次且不代关窗（关闭权在父层 HomePage）', async () => {
    const onCreate = vi.fn().mockResolvedValue(undefined)
    const { props } = renderModal({ onCreate })

    fireEvent.click(okButton())
    await waitFor(() => expect(onCreate).toHaveBeenCalledTimes(1))
    expect(props.onClose).not.toHaveBeenCalled()
  })
})

describe('CreateNovelModal 本书定位（v4.441 建档同址可选）', () => {
  it('选直白档：onMatureChange 收到 explicit', () => {
    const { props } = renderModal()
    fireEvent.click(screen.getByText('成人向 · 直白（亲密戏正面直书）'))
    expect(props.onMatureChange).toHaveBeenCalledWith('explicit')
  })
})

// ── 选题扫榜（oh-story story-long-scan 蒸馏批 N3）：粘贴→分析→报告→采用题材 ──

const scanResult = {
  market: { genre_distribution: '都市脑洞 8 本居首' },
  topics: [
    {
      direction: '都市规则怪谈', genre_mix: '都市+悬疑', audience: '男频 18-30',
      selling_point: '规则怪谈+直播流', why_hot: '假设：怪谈热延续',
      market_evidence: '粘贴文本中怪谈标签 20 本', differentiation: '主播视角',
      feasibility: '高', feasibility_reason: '样本足、趋势涨', direction_sample: 20,
      trend: '涨', risks: '同质化', validation: '先写黄金三章看追读',
      benchmarks: '', scope: '200 万字·起点',
    },
    {
      direction: '克苏鲁西幻', genre_mix: '奇幻+克苏鲁', audience: '男频',
      selling_point: '诡异氛围', why_hot: '假设', market_evidence: '内置知识·未实时校验',
      differentiation: '', feasibility: '中', feasibility_reason: '样本不足',
      direction_sample: 3, trend: '平', risks: '小众', validation: '',
      benchmarks: '', scope: '',
    },
  ],
  source: {
    kind: 'paste', platform: '起点', entry_count: 20, verified: false,
    disclaimer: '榜单为作者粘贴文本，未实时校验',
  },
}

const PASTE = '# 起点·风云榜\n' + Array.from({ length: 20 }, (_, i) => `${i + 1}. 《书${i}》：都市脑洞开局签到简介`).join('\n')

describe('CreateNovelModal 选题扫榜（oh-story 蒸馏批 N3）', () => {
  beforeEach(() => {
    scanMocks.scan.mockReset()
  })

  it('分析选题：报告渲染（可行性徽标/依据/免责行），采用题材回填勾选', async () => {
    scanMocks.scan.mockResolvedValueOnce(scanResult)
    const { props } = renderModal()

    fireEvent.click(screen.getByText(/扫榜找灵感/))
    fireEvent.change(screen.getByPlaceholderText(/粘贴榜单文本/), { target: { value: PASTE } })
    fireEvent.click(screen.getByRole('button', { name: /分析选题/ }))

    await waitFor(() => expect(scanMocks.scan).toHaveBeenCalledWith('', PASTE))
    expect(await screen.findByText('都市规则怪谈')).toBeTruthy()
    // 诚实纪律：来源免责行必须可见
    // 诚实纪律：来源免责行必须可见（全文精确匹配——正则会同时命中免责行与
    // 选题依据行「内置知识·未实时校验」两类叶子）
    expect(screen.getByText('榜单为作者粘贴文本，未实时校验（样本 20 条 · 起点）')).toBeTruthy()
    // 精确匹配：正则会连带命中包含 Tag 的祖先容器（多元素错误）
    expect(screen.getByText('可行性高')).toBeTruthy()
    expect(screen.getByText('可行性中')).toBeTruthy()

    // 采用题材：genre_mix 映射 GENRE_OPTIONS（都市+悬疑 都在册）；两条选题各有一枚，取第一条
    fireEvent.click(screen.getAllByRole('button', { name: /采用题材/ })[0])
    expect(props.onGenreChange).toHaveBeenCalledWith(['都市', '悬疑'])
  })

  it('分析失败：错误如实上屏，不出假报告', async () => {
    scanMocks.scan.mockRejectedValueOnce(new Error('扫榜分析失败: 模型未配置'))
    renderModal()

    fireEvent.click(screen.getByText(/扫榜找灵感/))
    fireEvent.change(screen.getByPlaceholderText(/粘贴榜单文本/), { target: { value: PASTE } })
    fireEvent.click(screen.getByRole('button', { name: /分析选题/ }))

    expect(await screen.findByText(/扫榜分析失败/)).toBeTruthy()
    expect(screen.queryByText('都市规则怪谈')).toBeNull()
  })

  it('未展开时不请求；粘贴过短时分析钮禁用', () => {
    renderModal()
    expect(scanMocks.scan).not.toHaveBeenCalled()
    fireEvent.click(screen.getByText(/扫榜找灵感/))
    fireEvent.change(screen.getByPlaceholderText(/粘贴榜单文本/), { target: { value: '太短' } })
    expect((screen.getByRole('button', { name: /分析选题/ }) as HTMLButtonElement).disabled).toBe(true)
    expect(scanMocks.scan).not.toHaveBeenCalled()
  })
})
