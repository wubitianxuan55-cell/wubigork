import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { Modal } from 'antd'
import PantheonPage from './PantheonPage'
import type { PantheonAgentStatus, PantheonSession } from './pantheon/types'

const mocks = vi.hoisted(() => ({
  agents: vi.fn<() => Promise<PantheonAgentStatus[]>>(),
  workspaces: vi.fn(),
  pickWorkspace: vi.fn(),
  list: vi.fn(),
  get: vi.fn(),
  start: vi.fn(),
  cancel: vi.fn(),
  remove: vi.fn(),
  export: vi.fn(),
  signOff: vi.fn(),
  setExecutable: vi.fn(),
}))

vi.mock('../gaea/lib/bridge', () => ({
  app: {
    PantheonAgents: mocks.agents,
    PantheonWorkspaces: mocks.workspaces,
    PantheonPickWorkspace: mocks.pickWorkspace,
    PantheonList: mocks.list,
    PantheonGet: mocks.get,
    PantheonStart: mocks.start,
    PantheonCancel: mocks.cancel,
    PantheonDelete: mocks.remove,
    PantheonExport: mocks.export,
    PantheonSignOff: mocks.signOff,
    PantheonSetExecutable: mocks.setExecutable,
  },
}))

const AGENTS: PantheonAgentStatus[] = [
  { id: 'grok', name: 'Grok', kind: 'Gaea 模型 · xAI', available: true, version: 'grok-4' },
  { id: 'codex', name: 'Codex', kind: 'Codex CLI', available: true, version: '1.0' },
  { id: 'zcode', name: 'ZCode', kind: 'ZCode CLI', available: true, version: '0.17' },
  { id: 'dsh', name: 'DeepSeek Harness', kind: 'DSH headless', available: true, version: '0.2.1' },
]

function sessionFixture(overrides: Partial<PantheonSession> = {}): PantheonSession {
  const now = Date.now()
  return {
    id: 'pan_test',
    title: '是否采用只读议事',
    status: 'running',
    phase: 'opening',
    currentRound: 1,
    createdAt: now,
    updatedAt: now,
    request: {
      title: '是否采用只读议事',
      topic: '是否先采用只读议事验证价值？',
      context: '首版不能写工作区。',
      successCriteria: '给出风险与条件。',
      workspace: 'C:\\AI\\wubigrok',
      evidence: [{ name: '调研', content: '已有竞品验证。' }],
      agentIds: ['grok', 'codex', 'zcode', 'dsh'],
      chairmanAgentId: 'grok',
      rounds: 2,
      readOnly: true,
    },
    participants: AGENTS.map((agent) => ({
      agentId: agent.id,
      name: agent.name,
      role: 'participant',
      status: 'running',
    })),
    events: [{
      seq: 1,
      sessionId: 'pan_test',
      phase: 'opening',
      kind: 'phase',
      text: '开始开场陈述',
      at: now,
    }],
    contributions: [],
    ...overrides,
  }
}

describe('PantheonPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    window.localStorage.removeItem('gaea.pantheon.rail-collapsed')
    mocks.agents.mockResolvedValue(AGENTS)
    mocks.workspaces.mockResolvedValue([
      { path: 'C:\\AI\\wubigrok', name: 'wubigrok', current: true },
      { path: 'C:\\AI\\gaea-research', name: 'gaea-research', current: false, recent: true },
    ])
    mocks.pickWorkspace.mockResolvedValue('')
    mocks.list.mockResolvedValue([])
    mocks.get.mockResolvedValue(sessionFixture())
    mocks.start.mockResolvedValue(sessionFixture())
    mocks.cancel.mockResolvedValue(undefined)
    mocks.remove.mockResolvedValue(undefined)
    mocks.export.mockResolvedValue('# 议事录')
    mocks.signOff.mockImplementation(async (_id, decision) => ({
      ...sessionFixture({ status: 'completed', phase: 'completed' }),
      signoff: { decision, at: Date.now() },
    }))
    mocks.setExecutable.mockResolvedValue(AGENTS)
  })

  it('空态可发起讨论，并展示四个本机席位', async () => {
    render(<PantheonPage />)
    expect(await screen.findByRole('heading', { name: /在讨论中彼此补强/ })).toBeTruthy()
    fireEvent.click(screen.getByTestId('pantheon-welcome-start'))
    expect(await screen.findByTestId('pantheon-composer')).toBeTruthy()
    expect(await screen.findByTestId('pantheon-agent-grok')).toBeTruthy()
    expect(screen.getByTestId('pantheon-agent-grok').textContent).toContain('办公工具')
    expect(screen.getByTestId('pantheon-agent-codex')).toBeTruthy()
    expect(screen.getByTestId('pantheon-agent-zcode')).toBeTruthy()
    expect(screen.getByTestId('pantheon-agent-dsh')).toBeTruthy()
  })

  it('左侧讨论档案栏可以折叠、展开并记住选择', async () => {
    const { container } = render(<PantheonPage />)
    await screen.findByRole('heading', { name: /在讨论中彼此补强/ })
    const shell = container.querySelector('.pantheon-shell')
    expect(shell?.classList.contains('is-rail-collapsed')).toBe(false)

    fireEvent.click(screen.getByRole('button', { name: '折叠讨论档案' }))
    expect(shell?.classList.contains('is-rail-collapsed')).toBe(true)
    expect(window.localStorage.getItem('gaea.pantheon.rail-collapsed')).toBe('1')
    expect(screen.getByRole('button', { name: '展开讨论档案' })).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: '展开讨论档案' }))
    expect(shell?.classList.contains('is-rail-collapsed')).toBe(false)
    expect(window.localStorage.getItem('gaea.pantheon.rail-collapsed')).toBe('0')
  })

  it('提交议题时传递只读参与者与主席设置（Grok 入席则默认主持）', async () => {
    render(<PantheonPage />)
    await screen.findByRole('heading', { name: /在讨论中彼此补强/ })
    fireEvent.click(screen.getByTestId('pantheon-welcome-start'))
    // 默认只露主输入；材料与更多选项渐进披露
    expect(screen.queryByTestId('pantheon-title')).toBeNull()
    expect(screen.queryByTestId('pantheon-evidence')).toBeNull()
    fireEvent.click(screen.getByTestId('pantheon-toggle-evidence'))
    fireEvent.click(screen.getByTestId('pantheon-toggle-advanced'))
    fireEvent.click(await screen.findByTestId('pantheon-workspace-gaea-research'))
    fireEvent.click(await screen.findByRole('button', { name: 'Codex 前移' }))
    fireEvent.change(screen.getByTestId('pantheon-title'), { target: { value: '只读验证' } })
    fireEvent.change(screen.getByTestId('pantheon-topic'), {
      target: { value: '是否应该先做只读众神殿 MVP？' },
    })
    fireEvent.change(screen.getByTestId('pantheon-criteria'), {
      target: { value: '列出风险和适用条件' },
    })
    fireEvent.change(screen.getByTestId('pantheon-evidence'), {
      target: { value: '竞品已经验证议事形态。' },
    })

    const start = screen.getByTestId('pantheon-start')
    await waitFor(() => expect((start as HTMLButtonElement).disabled).toBe(false))
    fireEvent.click(start)

    await waitFor(() => expect(mocks.start).toHaveBeenCalledTimes(1))
    const request = mocks.start.mock.calls[0][0]
    expect(request).toMatchObject({
      title: '只读验证',
      topic: '是否应该先做只读众神殿 MVP？',
      successCriteria: '列出风险和适用条件',
      workspace: 'C:\\AI\\gaea-research',
      evidence: [{ name: '议事材料', content: '竞品已经验证议事形态。' }],
      chairmanAgentId: 'grok',
      rounds: 2,
      readOnly: true,
    })
    expect(request.agentIds).toEqual(['codex', 'grok', 'zcode'])
    expect(await screen.findByTestId('pantheon-record')).toBeTruthy()
  })

  it('讨论轮次可手动填写到两次以上', async () => {
    render(<PantheonPage />)
    fireEvent.click(await screen.findByTestId('pantheon-welcome-start'))
    fireEvent.change(await screen.findByTestId('pantheon-topic'), {
      target: { value: '需要七轮充分讨论的复杂议题' },
    })
    fireEvent.change(screen.getByRole('spinbutton', { name: '手动讨论轮次' }), {
      target: { value: '7' },
    })
    const start = screen.getByTestId('pantheon-start')
    await waitFor(() => expect((start as HTMLButtonElement).disabled).toBe(false))
    fireEvent.click(start)

    await waitFor(() => expect(mocks.start).toHaveBeenCalledTimes(1))
    expect(mocks.start.mock.calls[0][0]).toMatchObject({
      rounds: 7,
      untilConsensus: false,
    })
  })

  it('可切换为不限轮次，直到全员明确一致', async () => {
    render(<PantheonPage />)
    fireEvent.click(await screen.findByTestId('pantheon-welcome-start'))
    fireEvent.change(await screen.findByTestId('pantheon-topic'), {
      target: { value: '讨论到全员一致为止' },
    })
    fireEvent.click(screen.getByTestId('pantheon-round-auto'))
    expect(screen.getByTestId('pantheon-round-auto-label').textContent).toContain('至共识')
    const start = screen.getByTestId('pantheon-start')
    await waitFor(() => expect((start as HTMLButtonElement).disabled).toBe(false))
    fireEvent.click(start)

    await waitFor(() => expect(mocks.start).toHaveBeenCalledTimes(1))
    expect(mocks.start.mock.calls[0][0]).toMatchObject({
      rounds: 12,
      maxRounds: 12,
      untilConsensus: true,
    })
  })

  it('打开历史纪要并渲染综合建议、分歧与风险', async () => {
    const completed = sessionFixture({
      status: 'completed',
      phase: 'completed',
      verdict: {
        summary: '参与者支持只读试点。',
        recommendation: '先批准只读 MVP。',
        conditions: ['记录单 Agent 基线'],
        disagreements: ['扩展范围尚未一致'],
        risks: ['CLI 版本漂移'],
        evidenceNotes: ['竞品样本支持议事形态'],
        chairman: 'Codex',
        partial: false,
        createdAt: Date.now(),
      },
    })
    mocks.list.mockResolvedValue([{
      id: completed.id,
      title: completed.title,
      status: completed.status,
      phase: completed.phase,
      createdAt: completed.createdAt,
      updatedAt: completed.updatedAt,
      agentCount: completed.participants.length,
      verdictSummary: completed.verdict?.recommendation,
    }])
    mocks.get.mockResolvedValue(completed)

    const { container } = render(<PantheonPage />)
    fireEvent.click(await screen.findByTestId(`pantheon-session-${completed.id}`))
    expect(await screen.findByText('先批准只读 MVP。')).toBeTruthy()
    expect(screen.getByText('扩展范围尚未一致')).toBeTruthy()
    expect(screen.getByText('CLI 版本漂移')).toBeTruthy()
    expect(screen.getByText(/待你确认/)).toBeTruthy()
    const progress = screen.getByLabelText('多轮讨论进度')
    expect(progress.textContent).toContain('开场陈述')
    expect(progress.textContent).toContain('第 2 轮')
    expect(progress.textContent).toContain('协作汇总')
    expect(progress.textContent).not.toContain('匿名互评')
    expect(progress.textContent).not.toContain('定向质询')
    const verdict = container.querySelector('.pantheon-verdict')
    const transcript = container.querySelector('.pantheon-discussion-board')
    expect(verdict).toBeTruthy()
    expect(transcript).toBeTruthy()
    expect(verdict?.compareDocumentPosition(transcript as Node)).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING,
    )
  })

  it('用户可以确认最终讨论小结', async () => {
    const completed = sessionFixture({
      status: 'completed',
      phase: 'completed',
      verdict: {
        summary: '证据支持试点。',
        recommendation: '采纳只读试点。',
        chairman: 'Codex',
        partial: false,
        createdAt: Date.now(),
      },
    })
    mocks.list.mockResolvedValue([{
      id: completed.id,
      title: completed.title,
      status: completed.status,
      phase: completed.phase,
      createdAt: completed.createdAt,
      updatedAt: completed.updatedAt,
      agentCount: completed.participants.length,
      verdictSummary: completed.verdict?.recommendation,
    }])
    mocks.get.mockResolvedValue(completed)
    mocks.signOff.mockResolvedValue({
      ...completed,
      signoff: { decision: 'accepted', at: Date.now() },
    })
    const confirm = vi.spyOn(Modal, 'confirm').mockImplementation(((config: {
      onOk?: () => unknown
    }) => {
      void config.onOk?.()
      return { destroy: vi.fn(), update: vi.fn() }
    }) as typeof Modal.confirm)

    render(<PantheonPage />)
    fireEvent.click(await screen.findByTestId(`pantheon-session-${completed.id}`))
    fireEvent.click(await screen.findByRole('button', { name: '确认小结' }))
    await waitFor(() => expect(mocks.signOff).toHaveBeenCalledWith('pan_test', 'accepted', ''))
    expect(await screen.findByText('已确认小结')).toBeTruthy()
    confirm.mockRestore()
  })

  it('不限轮次纪要显示共识轨道与实际讨论模式', async () => {
    const completed = sessionFixture({
      status: 'completed',
      phase: 'completed',
      currentRound: 2,
      request: {
        ...sessionFixture().request,
        rounds: 12,
        maxRounds: 12,
        untilConsensus: true,
      },
      verdict: {
        summary: '第 2 轮已全员一致。',
        recommendation: '按当前方案推进。',
        chairman: 'Grok',
        partial: false,
        createdAt: Date.now(),
      },
    })
    mocks.list.mockResolvedValue([{
      id: completed.id,
      title: completed.title,
      status: completed.status,
      phase: completed.phase,
      createdAt: completed.createdAt,
      updatedAt: completed.updatedAt,
      agentCount: completed.participants.length,
      verdictSummary: completed.verdict?.recommendation,
    }])
    mocks.get.mockResolvedValue(completed)

    render(<PantheonPage />)
    fireEvent.click(await screen.findByTestId(`pantheon-session-${completed.id}`))
    const progress = await screen.findByLabelText('共识讨论进度')
    expect(progress.textContent).toContain('开场陈述')
    expect(progress.textContent).toContain('共识讨论')
    expect(progress.textContent).toContain('全员一致即结束')
    const settings = screen.getByRole('complementary', { name: '讨论纪要摘要' })
    expect(settings.textContent).toContain('不限（≤12 轮）')
    expect(screen.getByText('第 2 轮 · 至共识')).toBeTruthy()
  })

  it('可以基于已完成的小结续议，并继承议题、材料、席位与顺序', async () => {
    const completed = sessionFixture({
      status: 'completed',
      phase: 'completed',
      request: {
        ...sessionFixture().request,
        title: '续议验证',
        topic: '是否继续扩大只读试点范围？',
        context: '当前只在本机验证。',
        successCriteria: '明确扩大条件和停止条件。',
        evidence: [{ name: '上一轮数据', content: '两轮讨论均完成。' }],
        agentIds: ['codex', 'grok', 'zcode'],
        chairmanAgentId: 'grok',
        rounds: 3,
      },
      verdict: {
        summary: '可以先扩大一个工作空间。',
        recommendation: '再扩大一个工作空间，并保留人工复核。',
        disagreements: ['是否增加第四位参与者'],
        risks: ['长议题的上下文可能漂移'],
        chairman: 'Grok',
        partial: false,
        createdAt: Date.now(),
      },
    })
    mocks.list.mockResolvedValue([{
      id: completed.id,
      title: completed.title,
      workspace: completed.request.workspace,
      status: completed.status,
      phase: completed.phase,
      createdAt: completed.createdAt,
      updatedAt: completed.updatedAt,
      agentCount: completed.participants.length,
      verdictSummary: completed.verdict?.recommendation,
    }])
    mocks.get.mockResolvedValue(completed)

    render(<PantheonPage />)
    fireEvent.click(await screen.findByTestId(`pantheon-session-${completed.id}`))
    fireEvent.click(await screen.findByRole('button', { name: /基于小结续议/ }))

    expect(await screen.findByTestId('pantheon-composer')).toBeTruthy()
    expect((screen.getByTestId('pantheon-title') as HTMLInputElement).value).toContain('续议：')
    expect((screen.getByTestId('pantheon-topic') as HTMLTextAreaElement).value)
      .toContain('上一轮小结：再扩大一个工作空间')
    expect(screen.getByTestId('pantheon-evidence')).toBeTruthy()
    expect(screen.getByTestId('pantheon-workspace-wubigrok').getAttribute('aria-pressed')).toBe('true')
    expect(screen.getByTestId('pantheon-order-codex')).toBeTruthy()
    expect(screen.getByTestId('pantheon-order-grok')).toBeTruthy()
    expect(screen.getByTestId('pantheon-order-zcode')).toBeTruthy()
  })

  it('导出菜单可生成 JSON 完整记录', async () => {
    const completed = sessionFixture({
      status: 'completed',
      phase: 'completed',
      verdict: {
        summary: '可以导出复核。',
        recommendation: '保留完整事件与原始发言。',
        chairman: 'Grok',
        partial: false,
        createdAt: Date.now(),
      },
    })
    mocks.list.mockResolvedValue([{
      id: completed.id,
      title: completed.title,
      status: completed.status,
      phase: completed.phase,
      createdAt: completed.createdAt,
      updatedAt: completed.updatedAt,
      agentCount: completed.participants.length,
      verdictSummary: completed.verdict?.recommendation,
    }])
    mocks.get.mockResolvedValue(completed)

    let exportedBlob: Blob | undefined
    const createObjectURL = vi.fn((blob: Blob) => {
      exportedBlob = blob
      return 'blob:pantheon-json'
    })
    const revokeObjectURL = vi.fn()
    const originalCreate = URL.createObjectURL
    const originalRevoke = URL.revokeObjectURL
    Object.defineProperty(URL, 'createObjectURL', {
      configurable: true,
      writable: true,
      value: createObjectURL,
    })
    Object.defineProperty(URL, 'revokeObjectURL', {
      configurable: true,
      writable: true,
      value: revokeObjectURL,
    })
    const anchorClick = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    try {
      render(<PantheonPage />)
      fireEvent.click(await screen.findByTestId(`pantheon-session-${completed.id}`))
      fireEvent.click(await screen.findByRole('button', { name: /导出/ }))
      fireEvent.click(await screen.findByText('JSON 完整记录'))

      await waitFor(() => expect(createObjectURL).toHaveBeenCalledTimes(1))
      expect(exportedBlob?.type).toBe('application/json;charset=utf-8')
      expect(revokeObjectURL).toHaveBeenCalledWith('blob:pantheon-json')
      expect(anchorClick).toHaveBeenCalledTimes(1)
      expect(await screen.findByText('JSON 议事录已导出')).toBeTruthy()
    } finally {
      anchorClick.mockRestore()
      Object.defineProperty(URL, 'createObjectURL', {
        configurable: true,
        writable: true,
        value: originalCreate,
      })
      Object.defineProperty(URL, 'revokeObjectURL', {
        configurable: true,
        writable: true,
        value: originalRevoke,
      })
    }
  })

  it('讨论档案可以按关键字和状态筛选', async () => {
    const now = Date.now()
    mocks.list.mockResolvedValue([
      {
        id: 'pan_running',
        title: '正在验证模型接入',
        workspace: 'C:\\AI\\gaea-research',
        status: 'running',
        phase: 'opening',
        createdAt: now,
        updatedAt: now,
        agentCount: 3,
      },
      {
        id: 'pan_finished',
        title: '已完成合同风险复核',
        workspace: 'C:\\AI\\wubigrok',
        status: 'completed',
        phase: 'completed',
        createdAt: now - 1000,
        updatedAt: now - 1000,
        agentCount: 4,
        verdictSummary: '先补充责任边界。',
      },
    ])

    render(<PantheonPage />)
    expect(await screen.findByTestId('pantheon-session-pan_running')).toBeTruthy()
    expect(screen.getByTestId('pantheon-session-pan_finished')).toBeTruthy()

    fireEvent.change(screen.getByRole('textbox', { name: '搜索讨论档案' }), {
      target: { value: '合同' },
    })
    expect(screen.queryByTestId('pantheon-session-pan_running')).toBeNull()
    expect(screen.getByTestId('pantheon-session-pan_finished')).toBeTruthy()

    fireEvent.change(screen.getByRole('textbox', { name: '搜索讨论档案' }), {
      target: { value: '' },
    })
    fireEvent.click(screen.getByTestId('pantheon-filter-running'))
    expect(screen.getByTestId('pantheon-session-pan_running')).toBeTruthy()
    expect(screen.queryByTestId('pantheon-session-pan_finished')).toBeNull()
  })

  it('运行期按席位实时展示各席观点，汇总时间线不再摊全文', async () => {
    const now = Date.now()
    const running = sessionFixture({
      phase: 'opening',
      events: [
        {
          seq: 1, sessionId: 'pan_test', phase: 'opening', kind: 'phase',
          text: '开始开场陈述', at: now - 4000,
        },
        {
          seq: 2, sessionId: 'pan_test', phase: 'opening', round: 1, kind: 'agent_output',
          agentId: 'grok', text: '结论：先做只读验证，成本可控。', at: now - 2000, durationMs: 18200,
        },
        {
          seq: 3, sessionId: 'pan_test', phase: 'opening', round: 1, kind: 'agent_output',
          agentId: 'codex', text: '结论：验证可行，但必须设超时。', at: now - 1000, durationMs: 22400,
        },
      ],
      participants: [
        { agentId: 'grok', name: 'Grok', role: 'participant', status: 'finished' },
        { agentId: 'codex', name: 'Codex', role: 'participant', status: 'running' },
      ],
    })
    mocks.list.mockResolvedValue([{
      id: running.id,
      title: running.title,
      status: running.status,
      phase: running.phase,
      createdAt: running.createdAt,
      updatedAt: running.updatedAt,
      agentCount: running.participants.length,
    }])
    mocks.get.mockResolvedValue(running)

    render(<PantheonPage />)
    fireEvent.click(await screen.findByTestId('pantheon-session-pan_test'))

    const grokPane = await screen.findByTestId('pantheon-live-seat-grok')
    expect(grokPane.textContent).toContain('先做只读验证')
    expect(screen.getByTestId('pantheon-live-seat-codex').textContent).toContain('必须设超时')
    expect(screen.getByText('共享观点池')).toBeTruthy()
    expect(screen.getByTestId('pantheon-live-seat-zcode')).toBeTruthy()
    expect(screen.getByTestId('pantheon-live-seat-dsh').textContent).toContain('未参与本次讨论')
    expect(screen.getByLabelText('本轮发言顺序').textContent).toContain('Grok')
    const settings = screen.getByRole('complementary', { name: '讨论纪要摘要' })
    expect(settings.textContent).toContain('讨论范围')
    expect(settings.textContent).toContain('参与者')
    expect(settings.textContent).toContain('发言顺序')
    // 汇总时间线指向独立窗口，不重复摊全文
    expect(screen.getAllByText(/全文见参与者独立窗口/).length).toBe(2)
    // 主持席带皇冠
    expect(grokPane.querySelector('.pantheon-facilitator-mark')).toBeTruthy()
  })

  it('运行期实时展示只读工具轨迹并合并开始与结果', async () => {
    const now = Date.now()
    const running = sessionFixture({
      phase: 'opening',
      events: [
        {
          seq: 1,
          sessionId: 'pan_test',
          phase: 'opening',
          round: 1,
          agentId: 'codex',
          kind: 'tool_started',
          text: 'Codex 正在使用 命令',
          toolId: 'tool_1',
          toolName: '命令',
          toolStatus: 'started',
          toolInput: 'rg --files',
          at: now - 800,
        },
        {
          seq: 2,
          sessionId: 'pan_test',
          phase: 'opening',
          round: 1,
          agentId: 'codex',
          kind: 'tool_result',
          text: 'Codex 已完成 命令',
          toolId: 'tool_1',
          toolName: '命令',
          toolStatus: 'completed',
          toolInput: 'rg --files',
          toolOutput: 'README.md\nmain.go',
          at: now - 500,
        },
      ],
    })
    mocks.list.mockResolvedValue([{
      id: running.id,
      title: running.title,
      status: running.status,
      phase: running.phase,
      createdAt: running.createdAt,
      updatedAt: running.updatedAt,
      agentCount: running.participants.length,
    }])
    mocks.get.mockResolvedValue(running)

    render(<PantheonPage />)
    fireEvent.click(await screen.findByTestId('pantheon-session-pan_test'))
    const trajectory = await screen.findByTestId('pantheon-tool-trajectory')
    expect(trajectory.textContent).toContain('工具轨迹')
    expect(trajectory.textContent).toContain('1 次')
    const trace = screen.getByTestId('pantheon-tool-tool_1')
    expect(trace.textContent).toContain('Codex')
    expect(trace.textContent).toContain('命令')
    expect(trace.textContent).toContain('rg --files')
    expect(trace.textContent).toContain('结果预览')
    expect(trace.textContent).toContain('README.md')
  })

  it('可用 Agent 少于两个时禁止开始讨论', async () => {
    mocks.agents.mockResolvedValue([
      AGENTS[0],
      { ...AGENTS[1], available: false, reason: '未登录' },
      { ...AGENTS[2], available: false, reason: '未找到 CLI' },
    ])
    render(<PantheonPage />)
    fireEvent.click(await screen.findByTestId('pantheon-welcome-start'))
    fireEvent.change(await screen.findByTestId('pantheon-topic'), { target: { value: '测试议题' } })
    await waitFor(() => expect(mocks.agents).toHaveBeenCalled())
    const start = screen.getByTestId('pantheon-start') as HTMLButtonElement
    expect(start.disabled).toBe(true)
    expect(screen.getByTestId('pantheon-agent-zcode').getAttribute('aria-pressed')).toBe('false')
  })

  it('Agent 路径通过页面弹窗设置，不调用被禁用的 window.prompt', async () => {
    const prompt = vi.spyOn(window, 'prompt')
    render(<PantheonPage />)

    fireEvent.click(await screen.findByTitle('Codex 1.0'))
    const input = await screen.findByLabelText('Agent 可执行文件路径')
    fireEvent.change(input, { target: { value: 'C:\\Tools\\codex.exe' } })
    fireEvent.click(screen.getByRole('button', { name: '保存路径' }))

    await waitFor(() => {
      expect(mocks.setExecutable).toHaveBeenCalledWith('codex', 'C:\\Tools\\codex.exe')
    })
    expect(prompt).not.toHaveBeenCalled()
    prompt.mockRestore()
  })

  it('可为单场议事选择其他工作空间而不切换全局工作区', async () => {
    mocks.pickWorkspace.mockResolvedValue('C:\\AI\\council-workspace')
    render(<PantheonPage />)

    fireEvent.click(await screen.findByTestId('pantheon-welcome-start'))
    fireEvent.click(await screen.findByTestId('pantheon-pick-workspace'))
    expect(await screen.findByTestId('pantheon-workspace-council-workspace')).toBeTruthy()
    fireEvent.change(screen.getByTestId('pantheon-topic'), { target: { value: '限定项目范围的议题' } })

    const start = screen.getByTestId('pantheon-start')
    await waitFor(() => expect((start as HTMLButtonElement).disabled).toBe(false))
    fireEvent.click(start)

    await waitFor(() => expect(mocks.start).toHaveBeenCalledTimes(1))
    expect(mocks.start.mock.calls[0][0].workspace).toBe('C:\\AI\\council-workspace')
  })

  it('Grok 缺席时主持回落到首个可用席位', async () => {
    mocks.agents.mockResolvedValue([
      { ...AGENTS[0], available: false, reason: '离线模式下云端模型席位不可用' },
      AGENTS[1],
      AGENTS[2],
    ])
    render(<PantheonPage />)
    fireEvent.click(await screen.findByTestId('pantheon-welcome-start'))
    fireEvent.change(await screen.findByTestId('pantheon-topic'), { target: { value: '测试议题' } })
    const start = screen.getByTestId('pantheon-start')
    await waitFor(() => expect((start as HTMLButtonElement).disabled).toBe(false))
    fireEvent.click(start)
    await waitFor(() => expect(mocks.start).toHaveBeenCalledTimes(1))
    const request = mocks.start.mock.calls[0][0]
    expect(request.chairmanAgentId).toBe('codex')
    expect(request.agentIds).toEqual(['codex', 'zcode'])
  })
})
