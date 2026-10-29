import type { PantheonBindings } from '../bridge/pantheon'
import type {
  PantheonAgentStatus,
  PantheonContribution,
  PantheonEvent,
  PantheonSession,
  PantheonSessionSummary,
  PantheonStartRequest,
  PantheonWorkspace,
} from '../types/pantheon'

const now = () => Date.now()

function summary(session: PantheonSession): PantheonSessionSummary {
  return {
    id: session.id,
    title: session.title,
    workspace: session.request.workspace,
    status: session.status,
    phase: session.phase,
    createdAt: session.createdAt,
    updatedAt: session.updatedAt,
    agentCount: session.participants.length,
    verdictSummary: session.verdict?.recommendation,
    error: session.error,
  }
}

function exportSession(session: PantheonSession): string {
  const verdict = session.verdict
  return [
    `# 众神殿议事录 · ${session.title}`,
    '',
    session.request.workspace ? `- 工作空间：${session.request.workspace}` : '',
    session.request.untilConsensus
      ? `- 讨论模式：不限轮次，至全员一致（上限 ${session.request.maxRounds || session.request.rounds} 轮）`
      : `- 讨论轮数：${session.request.rounds}`,
    `- ID：${session.id}`,
    `- 状态：${session.status}`,
    `- 议题：${session.request.topic}`,
    '',
    '## 讨论小结',
    '',
    verdict?.recommendation || '未形成自动小结。',
    '',
    verdict?.summary || '',
    '',
    ...(verdict?.risks || []).map((item) => `- 风险：${item}`),
    '',
    '> 本纪要是多轮讨论整理，不等同于用户确认。',
  ].join('\n')
}

export function buildPantheon(): PantheonBindings {
  let agents: PantheonAgentStatus[] = [
    {
      id: 'grok',
      name: 'Grok',
      kind: 'Gaea 办公席位 · xAI',
      available: true,
      version: 'grok-4 (mock)',
      capabilities: ['chairman', 'gaea-model', 'gaea-office-tools', 'read-only-tools', 'tool-trajectory'],
    },
    { id: 'codex', name: 'Codex', kind: 'OpenAI Codex CLI', available: true, version: 'mock', capabilities: ['read-only'] },
    { id: 'zcode', name: 'ZCode', kind: 'ZCode CLI', available: true, version: 'mock', capabilities: ['plan-mode'] },
    { id: 'dsh', name: 'DeepSeek Harness', kind: 'DSH headless', available: true, version: 'mock', capabilities: ['read-only'] },
  ]
  const workspaces: PantheonWorkspace[] = [
    { path: 'C:/AI/wubigrok', name: 'wubigrok', current: true },
    { path: 'C:/AI/gaea-research', name: 'gaea-research', current: false, recent: true },
  ]
  const sessions = new Map<string, PantheonSession>()

  const makeSession = (request: PantheonStartRequest): PantheonSession => {
    const stamp = now()
    const id = `pan_mock_${stamp}`
    const participants = request.agentIds.map((agentID) => ({
      agentId: agentID,
      name: agents.find((agent) => agent.id === agentID)?.name || agentID,
      role: 'participant',
      status: 'waiting',
    }))
    const events: PantheonEvent[] = [{
      seq: 1,
      sessionId: id,
      phase: 'preparing',
      kind: 'session_started',
      text: '众神殿圆桌讨论开始',
      at: stamp,
    }]
    return {
      id,
      title: request.title || request.topic.split('\n')[0].slice(0, 48),
      status: 'running',
      phase: 'preparing',
      createdAt: stamp,
      updatedAt: stamp,
      request,
      participants,
      events,
      contributions: [],
    }
  }

  const addRoundEvents = (session: PantheonSession, round: number, phase: string) => {
    const order = session.participants.map((participant) => participant.agentId)
    const offset = (round - 1) % Math.max(order.length, 1)
    const roundOrder = [...order.slice(offset), ...order.slice(0, offset)]
    roundOrder.forEach((agentID, index) => {
      if (session.events.some(
        (event) => event.kind === 'agent_output'
          && event.agentId === agentID
          && event.round === round,
      )) return
      const participant = session.participants.find((item) => item.agentId === agentID)
      if (!participant) return
      {
        const toolID = `${session.id}-${agentID}-r${round}-tool`
        const toolName = agentID === 'grok'
          ? 'semantic_search'
          : agentID === 'zcode'
            ? 'Glob'
            : agentID === 'dsh'
              ? 'pwsh'
              : '命令'
        const toolInput = agentID === 'grok'
          ? '检索与议题相关的办公资料与知识条目'
          : agentID === 'zcode'
            ? '寻找与议题相关的项目文件'
            : agentID === 'dsh'
              ? '读取项目目录与关键配置'
              : '检索相关代码与文档'
        session.events.push({
          seq: session.events.length + 1,
          sessionId: session.id,
          phase,
          round,
          agentId: agentID,
          kind: 'tool_started',
          text: `${participant.name} 正在使用 ${toolName}`,
          toolId: toolID,
          toolName,
          toolStatus: 'started',
          toolInput,
          at: now(),
        })
        session.events.push({
          seq: session.events.length + 1,
          sessionId: session.id,
          phase,
          round,
          agentId: agentID,
          kind: 'tool_result',
          text: `${participant.name} 已完成 ${toolName}`,
          toolId: toolID,
          toolName,
          toolStatus: 'completed',
          toolInput,
          toolOutput: '已读取 3 个相关文件，未修改工作区。',
          at: now(),
          durationMs: 460 + index * 120,
        })
      }
      session.events.push({
        seq: session.events.length + 1,
        sessionId: session.id,
        phase,
        round,
        agentId: agentID,
        kind: 'agent_output',
        text: phase === 'opening'
          ? `我的初步判断是先从可中止的垂直切片开始。主要依据是成本边界可控；当前最大的不确定是长期复用数据。\n协作请求：请补充失败条件和验证指标。\n置信度：${index === 0 ? '高' : '中'}。`
          : '我吸收了前文对成本与失败边界的补充，建议先记录单 Agent 基线，再加入多席讨论，避免把形式收益误当成质量收益。\n保留意见：多轮讨论是否带来稳定增益仍需验证。',
        at: now(),
        durationMs: 1800 + index * 260,
      })
    })
  }

  const advance = (session: PantheonSession) => {
    if (session.status !== 'running') return
    const elapsed = now() - session.createdAt
    if (elapsed < 500) return
    if (elapsed < 1100) {
      session.phase = 'opening'
      session.currentRound = 1
      addRoundEvents(session, 1, 'opening')
      session.participants.forEach((participant, index) => {
        participant.status = index === 0 ? 'running' : 'waiting'
      })
      return
    }
    if (elapsed < 1700) {
      addRoundEvents(session, 1, 'opening')
      session.phase = 'discussion'
      session.currentRound = 2
      addRoundEvents(session, 2, 'discussion')
      session.participants.forEach((participant) => { participant.status = 'finished' })
      return
    }
    if (elapsed < 2200) {
      session.phase = 'synthesis'
      session.participants.forEach((participant) => {
        participant.status = participant.agentId === session.request.chairmanAgentId ? 'running' : 'finished'
      })
      return
    }
    const totalRounds = session.request.untilConsensus
      ? 2
      : Math.max(2, Math.min(20, session.request.rounds || 2))
    const contributions: PantheonContribution[] = []
    const order = session.participants.map((participant) => participant.agentId)
    for (let round = 1; round <= totalRounds; round++) {
      const offset = (round - 1) % Math.max(order.length, 1)
      const roundOrder = [...order.slice(offset), ...order.slice(0, offset)]
      roundOrder.forEach((agentID, index) => {
        const participant = session.participants.find((item) => item.agentId === agentID)
        if (!participant) return
        contributions.push({
          id: `${session.id}-${agentID}-round-${round}`,
          phase: round === 1 ? 'opening' : 'discussion',
          round,
          agentId: agentID,
          agentName: participant.name,
          content: round === 1
            ? `我的理解：建议先用可中止的垂直切片验证方向。\n主要依据：改造成本与失败边界可控，但缺少长期复用数据。\n协作请求：请帮助补充失败条件和验证指标。\n置信度：${index === 0 ? '高' : '中'}。`
            : '我吸收了前文的成本边界建议，补充一个验证策略：先记录单 Agent 基线，再加入多席讨论，避免把形式收益误当成质量收益。\n仍待验证：多轮讨论是否会带来可观增益。',
          consensus: round > 1 ? 'yes' : undefined,
          createdAt: session.createdAt + 500 + round * 400 + index * 80,
        })
      })
    }
    const facilitator = session.participants.find(
      (participant) => participant.agentId === session.request.chairmanAgentId,
    ) || session.participants[0]
    if (facilitator) {
      contributions.push({
        id: `${session.id}-${facilitator.agentId}-synthesis`,
        phase: 'synthesis',
        agentId: facilitator.agentId,
        agentName: facilitator.name,
        anonLabel: '协作汇总',
        content: '各参与者都支持先做可中止的只读验证；主要分歧集中在验证周期和扩展条件。',
        createdAt: now(),
      })
    }
    session.contributions = contributions
    if (session.request.untilConsensus) {
      session.events.push({
        seq: session.events.length + 1,
        sessionId: session.id,
        phase: 'discussion',
        round: 2,
        kind: 'consensus',
        text: '第 2 轮全员意见一致，提前进入协作汇总',
        at: now(),
      })
    }
    session.events.push(
      {
        seq: session.events.length + 1,
        sessionId: session.id,
        phase: 'completed',
        kind: 'verdict',
        text: '先做只读 MVP，通过接入门与产品门后再扩展。',
        at: now(),
      },
      {
        seq: session.events.length + 2,
        sessionId: session.id,
        phase: 'completed',
        kind: 'completed',
        text: '讨论小结已形成，等待用户确认',
        at: now(),
      },
    )
    session.status = 'completed'
    session.phase = 'completed'
    session.updatedAt = now()
    session.verdict = {
      summary: session.request.untilConsensus
        ? '各参与者通过多轮补充论证，最终对当前综合方向达成一致。'
        : '各参与者通过多轮补充论证，认可先做可中止的只读验证，同时对讨论顺序、验证周期和扩展条件保留了不同判断。',
      recommendation: '先完成小范围只读试点，并记录单 Agent 与多席讨论的对照数据，再根据实际增益决定是否扩大。',
      conditions: ['只开放 Codex / ZCode / DSH', '不授予工作区写入权限', '一席失败不阻塞其他 Agent'],
      disagreements: session.request.untilConsensus
        ? []
        : ['验证周期是否应超过两轮仍缺少一致意见'],
      risks: ['CLI 版本漂移', '多 Agent 调用成本高于单 Agent', '模型共识不等于事实正确'],
      evidenceNotes: ['竞品验证了议事形态，但没有证明本项目用户会持续复用'],
      chairman: session.request.chairmanAgentId,
      partial: false,
      createdAt: now(),
    }
  }

  return {
    PantheonAgents: async () => agents,
    PantheonWorkspaces: async () => structuredClone(workspaces),
    PantheonPickWorkspace: async () => {
      const picked: PantheonWorkspace = {
        path: 'C:/AI/council-workspace',
        name: 'council-workspace',
        current: false,
        recent: true,
      }
      if (!workspaces.some((workspace) => workspace.path === picked.path)) {
        workspaces.push(picked)
      }
      return picked.path
    },
    PantheonList: async () => [...sessions.values()]
      .map(summary)
      .sort((a, b) => b.updatedAt - a.updatedAt),
    PantheonGet: async (id) => {
      const session = sessions.get(id)
      if (!session) throw new Error('议事不存在')
      advance(session)
      return structuredClone(session)
    },
    PantheonStart: async (request) => {
      const session = makeSession({
        ...request,
        workspace: request.workspace || workspaces[0]?.path,
        readOnly: true,
      })
      sessions.set(session.id, session)
      return structuredClone(session)
    },
    PantheonCancel: async (id) => {
      const session = sessions.get(id)
      if (!session) throw new Error('议事不存在')
      session.status = 'cancelled'
      session.phase = 'cancelled'
      session.error = '用户终止了本次议事'
      session.updatedAt = now()
    },
    PantheonDelete: async (id) => {
      if (!sessions.delete(id)) throw new Error('议事不存在')
    },
    PantheonExport: async (id) => {
      const session = sessions.get(id)
      if (!session) throw new Error('议事不存在')
      return exportSession(session)
    },
    PantheonSignOff: async (id, decision, note) => {
      const session = sessions.get(id)
      if (!session) throw new Error('议事不存在')
      if (session.status === 'running') throw new Error('议事仍在运行')
      session.signoff = { decision, note, at: now() }
      session.updatedAt = now()
      return structuredClone(session)
    },
    PantheonSetExecutable: async (id, path) => {
      if (id === 'grok') {
        throw new Error('Grok 由 Gaea 办公 AgentRunner 驱动，无需配置可执行路径')
      }
      agents = agents.map((agent) => agent.id === id
        ? { ...agent, available: Boolean(path), executable: path, reason: path ? '' : '未配置路径' }
        : agent)
      return agents
    },
  }
}
