import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { Modal } from 'antd'
import PantheonPage from './PantheonPage'
import type { PantheonAgentStatus, PantheonSession } from './pantheon/types'

const mocks = vi.hoisted(() => ({
  agents: vi.fn<() => Promise<PantheonAgentStatus[]>>(),
  pickFiles: vi.fn(),
  workspaces: vi.fn(),
  pickWorkspace: vi.fn(),
  list: vi.fn(),
  get: vi.fn(),
  start: vi.fn(),
  cancel: vi.fn(),
  skip: vi.fn(),
  retryAgent: vi.fn(),
  restart: vi.fn(),
  remove: vi.fn(),
  export: vi.fn(),
  openResult: vi.fn(),
  signOff: vi.fn(),
  setExecutable: vi.fn(),
  saveExportBlob: vi.fn<(blob: Blob, name: string) => Promise<boolean>>(),
}))

vi.mock('../gaea/lib/bridge', () => ({
  app: {
    PantheonAgents: mocks.agents,
    PickFiles: mocks.pickFiles,
    PantheonWorkspaces: mocks.workspaces,
    PantheonPickWorkspace: mocks.pickWorkspace,
    PantheonList: mocks.list,
    PantheonGet: mocks.get,
    PantheonStart: mocks.start,
    PantheonCancel: mocks.cancel,
    PantheonSkipAgent: mocks.skip,
    PantheonRetryAgent: mocks.retryAgent,
    PantheonRestart: mocks.restart,
    PantheonDelete: mocks.remove,
    PantheonExport: mocks.export,
    PantheonOpenResult: mocks.openResult,
    PantheonSignOff: mocks.signOff,
    PantheonSetExecutable: mocks.setExecutable,
  },
}))

vi.mock('../gaea/lib/saveFile', () => ({
  saveExportBlob: mocks.saveExportBlob,
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
    window.localStorage.removeItem('gaea.pantheon.inspector-collapsed')
    mocks.agents.mockResolvedValue(AGENTS)
    mocks.pickFiles.mockResolvedValue([])
    mocks.workspaces.mockResolvedValue([
      { path: 'C:\\AI\\wubigrok', name: 'wubigrok', current: true },
      { path: 'C:\\AI\\gaea-research', name: 'gaea-research', current: false, recent: true },
    ])
    mocks.pickWorkspace.mockResolvedValue('')
    mocks.list.mockResolvedValue([])
    mocks.get.mockResolvedValue(sessionFixture())
    mocks.start.mockResolvedValue(sessionFixture())
    mocks.cancel.mockResolvedValue(undefined)
    mocks.skip.mockResolvedValue(undefined)
    mocks.retryAgent.mockResolvedValue(undefined)
    mocks.restart.mockResolvedValue(sessionFixture({ id: 'pan_restarted' }))
    mocks.remove.mockResolvedValue(undefined)
    mocks.export.mockResolvedValue('# 议事录')
    mocks.openResult.mockResolvedValue('C:\\AI\\wubigrok\\.gaea\\pantheon\\results\\pan_test.md')
    mocks.signOff.mockImplementation(async (_id, decision) => ({
      ...sessionFixture({ status: 'completed', phase: 'completed' }),
      signoff: { decision, at: Date.now() },
    }))
    mocks.setExecutable.mockResolvedValue(AGENTS)
  })

  it('空态可发起讨论，并展示四个本机席位', async () => {
    render(<PantheonPage />)
    expect(await screen.findByRole('heading', { name: /在讨论中彼此补强/ })).toBeTruthy()
    expect(document.querySelectorAll('.pantheon-seat-card [data-avatar-agent] img').length).toBe(4)
    expect(screen.getByTestId('pantheon-agent-mini-dsh').textContent).toContain('DSH')
    expect(screen.getByTestId('pantheon-agent-mini-dsh').textContent).toContain('就绪')
    fireEvent.click(screen.getByTestId('pantheon-welcome-start'))
    expect(await screen.findByTestId('pantheon-composer')).toBeTruthy()
    expect(await screen.findByTestId('pantheon-agent-grok')).toBeTruthy()
    expect(screen.getByTestId('pantheon-agent-grok').textContent).toContain('联网 · 识图')
    expect(
      screen.getByTestId('pantheon-agent-grok').querySelector('[data-avatar-agent="grok"] img'),
    ).toBeTruthy()
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

  it('议题输入在失焦后给出就近错误，并在补全后解除开始拦截', async () => {
    render(<PantheonPage />)
    fireEvent.click(await screen.findByTestId('pantheon-welcome-start'))
    const topic = await screen.findByTestId('pantheon-topic')
    const start = screen.getByTestId('pantheon-start') as HTMLButtonElement

    fireEvent.blur(topic)
    expect((await screen.findByRole('alert')).textContent).toContain(
      '请先写下这次讨论需要回答的问题。',
    )
    expect(start.disabled).toBe(true)

    fireEvent.change(topic, { target: { value: '重新设计发起讨论页' } })
    await waitFor(() => expect(screen.queryByRole('alert')).toBeNull())
    await waitFor(() => expect(start.disabled).toBe(false))
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
      protocol: 'independent', // 2026-10-31 拍板：默认独立开场（首轮各抒己见）
    })
    expect(request.agentIds).toEqual(['codex', 'grok', 'zcode'])
    expect(await screen.findByTestId('pantheon-record')).toBeTruthy()
  })

  it('独立开场路径：允许单席单轮并把协议写入请求', async () => {
    render(<PantheonPage />)
    await screen.findByRole('heading', { name: /在讨论中彼此补强/ })
    fireEvent.click(screen.getByTestId('pantheon-welcome-start'))
    fireEvent.click(await screen.findByTestId('pantheon-workspace-gaea-research'))
    // 切换议事路径为独立开场（配方同 SchedulePage：mouseDown 打 .ant-select-selector，
    // 实等 .ant-select-item-option-content 渲染后点选项）
    fireEvent.mouseDown(
      screen.getByTestId('pantheon-protocol').querySelector('.ant-select-selector')!,
    )
    const option = await screen.findByText('独立开场', {
      selector: '.ant-select-item-option-content',
    })
    fireEvent.click(option.closest('.ant-select-item-option')!)
    // 只留 grok 一席（独立开场允许单席；圆桌接力则要求两位）
    fireEvent.click(screen.getByTestId('pantheon-agent-codex'))
    fireEvent.click(screen.getByTestId('pantheon-agent-zcode'))
    fireEvent.change(screen.getByTestId('pantheon-topic'), {
      target: { value: '单席独立深答对照' },
    })
    // 单轮放行：轮数最小值降为 1
    fireEvent.change(screen.getByRole('spinbutton', { name: '手动讨论轮次' }), {
      target: { value: '1' },
    })
    const start = screen.getByTestId('pantheon-start')
    await waitFor(() => expect((start as HTMLButtonElement).disabled).toBe(false))
    fireEvent.click(start)
    await waitFor(() => expect(mocks.start).toHaveBeenCalledTimes(1))
    const request = mocks.start.mock.calls[0][0]
    expect(request.protocol).toBe('independent')
    expect(request.agentIds).toEqual(['grok'])
    expect(request.rounds).toBe(1)
    expect(await screen.findByTestId('pantheon-record')).toBeTruthy()
  })

  it('图片附件进入议事请求，并显示共享识图说明', async () => {
    mocks.pickFiles.mockResolvedValue([
      {
        path: 'C:\\AI\\wubigrok\\docs\\diagram.png',
        name: 'diagram.png',
        size: 2048,
        type: 'image',
      },
    ])
    render(<PantheonPage />)
    fireEvent.click(await screen.findByTestId('pantheon-welcome-start'))
    fireEvent.change(await screen.findByTestId('pantheon-topic'), {
      target: { value: '结合架构图判断当前方案是否可实施。' },
    })
    fireEvent.click(screen.getByTestId('pantheon-pick-images'))

    expect(await screen.findByText('diagram.png')).toBeTruthy()
    expect(screen.getByText(/摘要共享给所有席位/)).toBeTruthy()
    const start = screen.getByTestId('pantheon-start')
    await waitFor(() => expect((start as HTMLButtonElement).disabled).toBe(false))
    fireEvent.click(start)

    await waitFor(() => expect(mocks.start).toHaveBeenCalledTimes(1))
    expect(mocks.start.mock.calls[0][0]).toMatchObject({
      attachments: [{
        name: 'diagram.png',
        path: 'C:\\AI\\wubigrok\\docs\\diagram.png',
        mime: 'image/png',
        size: 2048,
      }],
    })
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
    const transcript = container.querySelector('.pantheon-shared-dialogue')
    expect(verdict).toBeTruthy()
    expect(transcript).toBeTruthy()
    expect(transcript?.contains(verdict as Node)).toBe(true)
    expect(container.querySelector('.pantheon-shared-stream')?.compareDocumentPosition(verdict as Node)).toBe(
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

  it('完成态可直接打开主持人整理的最终文档', async () => {
    const completed = sessionFixture({
      status: 'completed',
      phase: 'completed',
      resultPath: 'C:\\AI\\wubigrok\\.gaea\\pantheon\\results\\pan_test.md',
      verdict: {
        summary: '全席完成综合。',
        recommendation: '形成最终交付文档。',
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
    fireEvent.click(await screen.findByTestId('pantheon-session-pan_test'))
    fireEvent.click(await screen.findByRole('button', { name: /打开最终文档/ }))

    await waitFor(() => expect(mocks.openResult).toHaveBeenCalledWith('pan_test'))
    expect(await screen.findByText('已打开主持人整理的最终文档')).toBeTruthy()
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
    fireEvent.click(screen.getByTestId('pantheon-meta-toggle'))
    const settings = screen.getByTestId('pantheon-meta')
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
    // 裁决第三刀：续议保留父会话谱系（承接以简报为准，不冒充继承全文）
    const start = screen.getByTestId('pantheon-start')
    await waitFor(() => expect((start as HTMLButtonElement).disabled).toBe(false))
    fireEvent.click(start)
    await waitFor(() => expect(mocks.start).toHaveBeenCalled())
    const continuation = mocks.start.mock.calls.find(
      (call) => (call[0] as { parentSessionId?: string }).parentSessionId,
    )
    expect(continuation?.[0]).toMatchObject({ parentSessionId: completed.id })
  })

  const openCompletedSession = async () => {
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
    render(<PantheonPage />)
    fireEvent.click(await screen.findByTestId(`pantheon-session-${completed.id}`))
    const trigger = await screen.findByRole('button', { name: /导出/ })
    fireEvent.click(trigger)
    // jsdom 下 rc-trigger 挂 portal 与 RTL waitFor 轮询相克，须实等 portal 出现
    await waitFor(() => {
      expect(document.body.querySelector('.ant-dropdown')).toBeTruthy()
    })
    fireEvent.click(screen.getByText('JSON 完整记录'))
  }

  it('导出菜单可生成 JSON 完整记录，经 saveExportBlob 双门落盘', async () => {
    mocks.saveExportBlob.mockResolvedValue(true)
    await openCompletedSession()
    await waitFor(() => expect(mocks.saveExportBlob).toHaveBeenCalledTimes(1))
    const [blob, name] = mocks.saveExportBlob.mock.calls[0]
    expect(blob?.type).toBe('application/json;charset=utf-8')
    expect(name).toBe('是否采用只读议事.json')
    expect(await screen.findByText('JSON 议事录已导出')).toBeTruthy()
  })

  it('导出被用户取消（另存为返回取消）提示已取消而不报错', async () => {
    mocks.saveExportBlob.mockResolvedValue(false)
    await openCompletedSession()
    await waitFor(() => expect(mocks.saveExportBlob).toHaveBeenCalledTimes(1))
    expect(await screen.findByText('已取消导出')).toBeTruthy()
    expect(screen.queryByText(/导出 JSON 失败/)).toBeNull()
    expect(screen.queryByText('JSON 议事录已导出')).toBeNull()
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

  it('左侧讨论档案可直接删除，且不会误打开纪要', async () => {
    const now = Date.now()
    mocks.list.mockResolvedValue([{
      id: 'pan_finished',
      title: '可删除的历史议题',
      status: 'completed',
      phase: 'completed',
      createdAt: now - 1000,
      updatedAt: now - 1000,
      agentCount: 3,
      verdictSummary: '已形成小结。',
    }])
    const confirm = vi.spyOn(Modal, 'confirm').mockImplementation(((config: {
      onOk?: () => unknown
    }) => {
      void config.onOk?.()
      return { destroy: vi.fn(), update: vi.fn() }
    }) as typeof Modal.confirm)

    render(<PantheonPage />)
    const remove = await screen.findByTestId('pantheon-session-delete-pan_finished')
    expect((remove as HTMLButtonElement).disabled).toBe(false)
    fireEvent.click(remove)

    await waitFor(() => expect(mocks.remove).toHaveBeenCalledWith('pan_finished'))
    expect(mocks.get).not.toHaveBeenCalled()
    confirm.mockRestore()
  })

  it('运行超过两分钟无新输出时只重试当前 Agent', async () => {
    const staleAt = Date.now() - 130_000
    const running = sessionFixture({
      createdAt: staleAt,
      updatedAt: staleAt,
      participants: [
        {
          agentId: 'grok',
          name: 'Grok',
          role: 'participant',
          status: 'running',
          startedAt: staleAt,
        },
        { agentId: 'codex', name: 'Codex', role: 'participant', status: 'waiting' },
      ],
      events: [{
        seq: 1,
        sessionId: 'pan_test',
        phase: 'opening',
        round: 1,
        agentId: 'grok',
        kind: 'agent_started',
        text: 'Grok 开始第 1 轮发言',
        at: staleAt,
      }],
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
    expect(await screen.findByTestId('pantheon-stall-notice')).toBeTruthy()
    fireEvent.click(screen.getByTestId('pantheon-retry-agent'))

    await waitFor(() => expect(mocks.retryAgent).toHaveBeenCalledWith('pan_test', 'grok'))
    expect(await screen.findByText('已重试当前 Agent，将保留已有发言并继续本轮')).toBeTruthy()
    expect(mocks.restart).not.toHaveBeenCalled()
  })

  it('卡死时可直接跳过当前 Agent 并继续下一位', async () => {
    const running = sessionFixture({
      participants: [
        { agentId: 'grok', name: 'Grok', role: 'participant', status: 'running' },
        { agentId: 'codex', name: 'Codex', role: 'participant', status: 'waiting' },
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
    fireEvent.click(await screen.findByTestId('pantheon-skip-current'))

    await waitFor(() => expect(mocks.skip).toHaveBeenCalledWith('pan_test', 'grok'))
    expect(await screen.findByText('已跳过当前 Agent，正在进入下一位')).toBeTruthy()
  })

  it('共享线程按顺序展示各 Agent 观点，点击拓扑节点查看独立上下文', async () => {
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

    const stream = await screen.findByTestId('pantheon-stream')
    expect(stream.textContent).toContain('开始开场陈述')
    const grokPosition = stream.textContent?.indexOf('先做只读验证') ?? -1
    const codexPosition = stream.textContent?.indexOf('必须设超时') ?? -1
    expect(grokPosition).toBeGreaterThanOrEqual(0)
    expect(codexPosition).toBeGreaterThan(grokPosition)
    expect(stream.querySelectorAll('.pantheon-shared-entry.is-agent').length).toBe(2)
    expect(document.querySelector('.pantheon-agent-windows')).toBeNull()
    expect(screen.queryByTestId('pantheon-agent-detail')).toBeNull()
    expect(screen.queryByTestId('pantheon-meta')).toBeNull()

    const grokNode = screen.getByTestId('pantheon-topology-node-grok')
    const codexNode = screen.getByTestId('pantheon-topology-node-codex')
    expect(grokNode.className).toContain('is-finished')
    expect(codexNode.className).toContain('is-running')
    expect(grokNode.getAttribute('aria-expanded')).toBe('false')

    fireEvent.click(grokNode)
    const grokDetail = await screen.findByTestId('pantheon-agent-detail')
    expect(grokNode.getAttribute('aria-expanded')).toBe('true')
    expect(grokDetail.textContent).toContain('先做只读验证')
    expect(grokDetail.textContent).not.toContain('必须设超时')
    expect(grokDetail.textContent).toContain('推理过程与工具调用不会写入中央共享讨论')

    fireEvent.click(screen.getByTestId('pantheon-agent-detail-close'))
    expect(screen.queryByTestId('pantheon-agent-detail')).toBeNull()
    fireEvent.click(codexNode)
    expect((await screen.findByTestId('pantheon-agent-detail')).textContent).toContain('必须设超时')

    fireEvent.click(screen.getByTestId('pantheon-meta-toggle'))
    const meta = screen.getByTestId('pantheon-meta')
    expect(meta.textContent).toContain('讨论范围')
    expect(meta.textContent).toContain('参与者')
    expect(meta.textContent).toContain('开始开场陈述')
  })

  it('四席拓扑节点共用身份色，详情面板继承所选 Agent 的颜色', async () => {
    const now = Date.now()
    const completed = sessionFixture({
      status: 'completed',
      phase: 'completed',
      participants: AGENTS.map((agent) => ({
        agentId: agent.id,
        name: agent.name,
        role: 'participant',
        status: 'finished',
      })),
      contributions: AGENTS.map((agent, index) => ({
        id: `contribution_${agent.id}`,
        phase: 'opening',
        round: 1,
        agentId: agent.id,
        agentName: agent.name,
        content: `${agent.name} 的独立观点`,
        createdAt: now + index,
      })),
      verdict: {
        summary: '四种判断均已保留。',
        recommendation: '按席位颜色复核各自观点。',
        chairman: 'Grok',
        partial: false,
        createdAt: now,
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
    await screen.findByTestId('pantheon-topology-node-grok')

    const accents = AGENTS.map((agent) => {
      const node = screen.getByTestId(`pantheon-topology-node-${agent.id}`) as HTMLElement
      const accent = node.style.getPropertyValue('--agent-accent')

      expect(node.getAttribute('data-agent-id')).toBe(agent.id)
      expect(node.querySelector(`[data-avatar-agent="${agent.id}"] img`)).toBeTruthy()
      expect(node.querySelector('[data-avatar-status="finished"]')).toBeTruthy()
      expect(accent).toBeTruthy()
      expect(node.textContent).toContain(agent.name)
      return accent
    })

    expect(new Set(accents).size).toBe(4)
    fireEvent.click(screen.getByTestId('pantheon-topology-node-grok'))
    const detail = await screen.findByTestId('pantheon-agent-detail')
    expect(detail.style.getPropertyValue('--agent-accent')).toBe(accents[0])
    expect(detail.querySelector('[data-avatar-agent="grok"] img')).toBeTruthy()
    expect(detail.textContent).toContain('联网 · 识图')
    expect(detail.textContent).toContain('Grok 的独立观点')
  })

  it('工具调用不进入共享线程，只在对应 Agent 详情的工具页展示', async () => {
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
    const stream = await screen.findByTestId('pantheon-stream')
    expect(stream.textContent).not.toContain('rg --files')
    expect(stream.textContent).not.toContain('README.md')
    expect(stream.querySelectorAll('[data-testid^="pantheon-tool-"]').length).toBe(0)
    expect(screen.queryByTestId('pantheon-tool-tool_1')).toBeNull()

    const codexNode = screen.getByTestId('pantheon-topology-node-codex')
    expect(codexNode.textContent).toContain('工具 1')
    fireEvent.click(codexNode)
    fireEvent.click(screen.getByTestId('pantheon-agent-tab-trace'))
    const trace = await screen.findByTestId('pantheon-tool-tool_1')
    expect(trace.textContent).toContain('命令')
    expect(trace.textContent).toContain('rg --files')
    expect(trace.textContent).toContain('README.md')
  })

  it('当前发言只在拓扑节点与共享主线程状态中高亮', async () => {
    const running = sessionFixture({
      participants: sessionFixture().participants.map((participant) => ({
        ...participant,
        status: participant.agentId === 'codex' ? 'running' : 'waiting',
      })),
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

    expect(await screen.findByTestId('pantheon-stream')).toBeTruthy()
    const codexNode = await screen.findByTestId('pantheon-topology-node-codex')
    expect(codexNode.className).toContain('is-running')
    expect(codexNode.getAttribute('aria-current')).toBe('true')
    expect(codexNode.querySelector('.pantheon-live-dot')).toBeTruthy()
    expect(codexNode.textContent).toContain('正在发言')

    const grokNode = screen.getByTestId('pantheon-topology-node-grok')
    expect(grokNode.className).toContain('is-waiting')
    expect(grokNode.className).not.toContain('is-running')
    expect(grokNode.getAttribute('aria-current')).toBeNull()
    expect(screen.getByTestId('pantheon-stream').textContent).toContain('Codex正在开场陈述')
    expect(document.querySelector('.pantheon-speaking-queue')).toBeNull()
    expect(document.querySelector('.pantheon-seat-list li.is-speaking')).toBeNull()
  })

  it('Agent 详情可关闭，并在 Escape 后把焦点还给拓扑节点', async () => {
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
    const codexNode = await screen.findByTestId('pantheon-topology-node-codex')
    fireEvent.click(codexNode)
    expect((await screen.findByTestId('pantheon-agent-detail')).textContent).toContain('必须设超时')
    fireEvent.click(screen.getByTestId('pantheon-agent-detail-close'))
    expect(screen.queryByTestId('pantheon-agent-detail')).toBeNull()

    fireEvent.click(codexNode)
    expect(await screen.findByTestId('pantheon-agent-detail')).toBeTruthy()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByTestId('pantheon-agent-detail')).toBeNull()
  })

  it('讨论信息默认收起，按需展开且不再占用常驻右栏', async () => {
    const running = sessionFixture()
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

    const toggle = await screen.findByTestId('pantheon-meta-toggle')
    expect(toggle.getAttribute('aria-expanded')).toBe('false')
    expect(screen.queryByRole('complementary', { name: '讨论纪要摘要' })).toBeNull()
    fireEvent.click(toggle)
    expect(toggle.getAttribute('aria-expanded')).toBe('true')
    const settings = screen.getByTestId('pantheon-meta')
    expect(settings.textContent).toContain('讨论范围')
    expect(settings.textContent).toContain('参与者')
    expect(document.querySelector('.pantheon-inspector')).toBeNull()
  })

  it('已结束会话的讨论信息保留公开进度时间线', async () => {
    const completed = sessionFixture({
      status: 'completed',
      phase: 'completed',
      verdict: {
        summary: '时间线随纪要归档。',
        recommendation: '先批准只读 MVP。',
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
    fireEvent.click(await screen.findByTestId('pantheon-meta-toggle'))
    const settings = screen.getByTestId('pantheon-meta')
    expect(settings.textContent).toContain('公开进度')
    expect(settings.textContent).toContain('开始开场陈述')
  })

  it('圆桌接力少于两席禁止开始；默认独立开场允许单席', async () => {
    mocks.agents.mockResolvedValue([
      AGENTS[0],
      { ...AGENTS[1], available: false, reason: '未登录' },
      { ...AGENTS[2], available: false, reason: '未找到 CLI' },
    ])
    render(<PantheonPage />)
    fireEvent.click(await screen.findByTestId('pantheon-welcome-start'))
    fireEvent.click(await screen.findByTestId('pantheon-workspace-wubigrok'))
    fireEvent.change(await screen.findByTestId('pantheon-topic'), { target: { value: '测试议题' } })
    await waitFor(() => expect(mocks.agents).toHaveBeenCalled())
    const start = screen.getByTestId('pantheon-start') as HTMLButtonElement
    expect(screen.getByTestId('pantheon-agent-zcode').getAttribute('aria-pressed')).toBe('false')
    // 默认独立开场（2026-10-31 拍板）：单席深答放行
    await waitFor(() => expect(start.disabled).toBe(false))
    // 切回圆桌接力：单席重新被拦
    fireEvent.mouseDown(
      screen.getByTestId('pantheon-protocol').querySelector('.ant-select-selector')!,
    )
    const councilOption = await screen.findByText('圆桌接力', {
      selector: '.ant-select-item-option-content',
    })
    fireEvent.click(councilOption.closest('.ant-select-item-option')!)
    await waitFor(() => expect(start.disabled).toBe(true))
  })

  it('Agent 路径只经 Composer 离线席位的显式「设置路径」配置；左栏卡永不弹窗', async () => {
    const prompt = vi.spyOn(window, 'prompt')
    // 2026-10-31 实弹拍板：切换席位永不弹窗——左栏卡全部纯状态展示，
    // 路径配置只走 Composer 离线席位 chip 上明确标注的「设置路径」小钮。
    const agents = AGENTS.map((agent) => (
      agent.id === 'codex' ? { ...agent, available: false, reason: '未找到 codex 命令' } : agent
    ))
    mocks.agents.mockResolvedValue(agents)
    render(<PantheonPage />)

    fireEvent.click(await screen.findByTestId('pantheon-agent-mini-codex'))
    expect(screen.queryByLabelText('Agent 可执行文件路径')).toBeNull()
    fireEvent.click(await screen.findByTestId('pantheon-welcome-start'))
    fireEvent.click(await screen.findByTestId('pantheon-agent-config-codex'))
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
