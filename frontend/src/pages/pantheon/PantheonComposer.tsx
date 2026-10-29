import React, { useEffect, useMemo, useRef, useState } from 'react'
import {
  ArrowDownOutlined,
  ArrowUpOutlined,
  CommentOutlined,
  DownOutlined,
  FolderOpenOutlined,
  FolderOutlined,
  PaperClipOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  SendOutlined,
} from '@ant-design/icons'
import { Alert, Button, Input, InputNumber, Select, Tooltip } from 'antd'
import { pantheonApi } from './api'
import type {
  PantheonAgentStatus,
  PantheonComposerSeed,
  PantheonStartRequest,
  PantheonWorkspace,
} from './types'
import { AGENT_ACCENTS } from './visuals'

interface Props {
  agents: PantheonAgentStatus[]
  busy: boolean
  error: string
  seed?: PantheonComposerSeed | null
  onStart: (request: PantheonStartRequest) => Promise<void>
  onRefreshAgents: () => void
}

const MANUAL_ROUNDS_MAX = 20
const CONSENSUS_SAFE_LIMIT = 12

function workspaceName(path: string): string {
  if (!path) return ''
  const parts = path.split(/[/\\]/).filter(Boolean)
  return parts[parts.length - 1] || path
}

function seedEvidenceText(seed?: PantheonComposerSeed | null): string {
  return (seed?.evidence || [])
    .map((item) => item.name ? `${item.name}\n${item.content}` : item.content)
    .join('\n\n---\n\n')
    .trim()
}

function agentToolLabel(agent: PantheonAgentStatus): string {
  if (agent.id === 'grok') return '办公工具'
  if (agent.id === 'zcode') return '计划工具'
  return '只读工具'
}

function agentToolDetail(agent: PantheonAgentStatus): string {
  if (agent.id === 'grok') return 'Gaea 办公工具；只读轨迹实时可见'
  if (agent.id === 'zcode') return 'Plan 模式工具；轨迹实时可见'
  return '只读工具；轨迹实时可见'
}

// 对话式开议：主输入即议题，其余全部渐进披露（材料/背景/成功标准/标题按需
// 展开），席位压成一行胶囊。字段与提交契约和旧表单版完全一致。
export const PantheonComposer: React.FC<Props> = ({
  agents,
  busy,
  error,
  seed,
  onStart,
  onRefreshAgents,
}) => {
  const initialEvidence = seedEvidenceText(seed)
  const [title, setTitle] = useState(seed?.title || '')
  const [topic, setTopic] = useState(seed?.topic || '')
  const [context, setContext] = useState(seed?.context || '')
  const [criteria, setCriteria] = useState(seed?.successCriteria || '')
  const [evidence, setEvidence] = useState(initialEvidence)
  const [selected, setSelected] = useState<string[]>(seed?.agentIds || [])
  const [chairman, setChairman] = useState(seed?.chairmanAgentId || '')
  const [rounds, setRounds] = useState(
    Math.max(2, Math.min(MANUAL_ROUNDS_MAX, seed?.rounds || 2)),
  )
  const [untilConsensus, setUntilConsensus] = useState(Boolean(seed?.untilConsensus))
  const [showEvidence, setShowEvidence] = useState(Boolean(initialEvidence))
  const [showAdvanced, setShowAdvanced] = useState(Boolean(
    seed?.title || seed?.context || seed?.successCriteria,
  ))
  const [workspaces, setWorkspaces] = useState<PantheonWorkspace[]>([])
  const [workspace, setWorkspace] = useState(seed?.workspace || '')
  const [workspaceBusy, setWorkspaceBusy] = useState(false)
  const [workspaceError, setWorkspaceError] = useState('')
  const initialized = useRef(Boolean(seed?.agentIds?.length))

  const available = useMemo(() => agents.filter((agent) => agent.available), [agents])

  useEffect(() => {
    if (available.length === 0) return
    if (!initialized.current && available.length > 0) {
      initialized.current = true
      setSelected(available.slice(0, 3).map((agent) => agent.id))
    }
    const allowed = new Set(available.map((agent) => agent.id))
    setSelected((current) => current.filter((id) => allowed.has(id)))
  }, [available])

  useEffect(() => {
    if (selected.length === 0) {
      if (chairman) setChairman('')
      return
    }
    // 汇总人默认优先 Grok；仅负责整理，不拥有最终决定权。
    if (!selected.includes(chairman)) {
      setChairman(selected.includes('grok') ? 'grok' : selected[0])
    }
  }, [selected, chairman])

  useEffect(() => {
    let cancelled = false
    void pantheonApi.PantheonWorkspaces()
      .then((list) => {
        if (cancelled) return
        const seeded = seed?.workspace && !list.some((item) => item.path === seed.workspace)
          ? [{
              path: seed.workspace,
              name: workspaceName(seed.workspace),
              current: false,
              recent: true,
            }, ...list]
          : list
        setWorkspaces(seeded)
        setWorkspace((current) =>
          current || seeded.find((item) => item.current)?.path || seeded[0]?.path || '',
        )
      })
      .catch((err) => {
        if (!cancelled) {
          setWorkspaceError(err instanceof Error ? err.message : String(err))
        }
      })
    return () => { cancelled = true }
  }, [seed?.workspace])

  const pickWorkspace = async () => {
    setWorkspaceBusy(true)
    setWorkspaceError('')
    try {
      const path = await pantheonApi.PantheonPickWorkspace()
      if (!path) return
      setWorkspace(path)
      setWorkspaces((current) => {
        if (current.some((item) => item.path === path)) return current
        return [
          ...current,
          { path, name: workspaceName(path), current: false, recent: true },
        ]
      })
    } catch (err) {
      setWorkspaceError(`选择工作空间失败：${err instanceof Error ? err.message : String(err)}`)
    } finally {
      setWorkspaceBusy(false)
    }
  }

  const toggleAgent = (id: string) => {
    setSelected((current) =>
      current.includes(id) ? current.filter((candidate) => candidate !== id) : [...current, id],
    )
  }

  const moveAgent = (index: number, delta: -1 | 1) => {
    setSelected((current) => {
      const target = index + delta
      if (target < 0 || target >= current.length) return current
      const next = [...current]
      const moved = next[index]
      next[index] = next[target]
      next[target] = moved
      return next
    })
  }

  const startBlockReason = !workspace
    ? '请先选择讨论范围'
    : selected.length < 2
      ? '至少选择两位参与者'
      : !topic.trim()
        ? '写下议题后即可开始'
        : ''
  const canStart = !startBlockReason && !busy

  const start = async () => {
    if (!canStart) return
    const roundLimit = untilConsensus
      ? Math.max(2, Math.min(MANUAL_ROUNDS_MAX, seed?.maxRounds || CONSENSUS_SAFE_LIMIT))
      : rounds
    await onStart({
      title: title.trim(),
      topic: topic.trim(),
      context: context.trim(),
      successCriteria: criteria.trim(),
      workspace,
      evidence: evidence.trim()
        ? [{ name: '议事材料', content: evidence.trim() }]
        : [],
      agentIds: selected,
      chairmanAgentId: chairman,
      rounds: roundLimit,
      untilConsensus,
      maxRounds: untilConsensus ? roundLimit : undefined,
      readOnly: true,
    })
  }

  return (
    <section className="pantheon-composer" data-testid="pantheon-composer" aria-labelledby="pantheon-compose-title">
      <header className="pantheon-composer-head">
        <div>
          <span className="pantheon-eyebrow">发起圆桌讨论</span>
          <h2 id="pantheon-compose-title">有序表达，多轮补强。</h2>
          <p>参与者按顺序表达，多轮补充证据、澄清分歧，并在最后协作整理。</p>
        </div>
        <div className="pantheon-composer-mode">
          <SafetyCertificateOutlined />
          <span><strong>只读运行</strong><small>不改动工作区</small></span>
        </div>
      </header>

      {error && <Alert type="error" showIcon message={error} className="pantheon-alert" />}

      <section className="pantheon-workspace-scope" aria-labelledby="pantheon-workspace-title">
        <div className="pantheon-workspace-head">
          <div className="pantheon-workspace-title">
            <span className="pantheon-workspace-icon" aria-hidden>
              <FolderOpenOutlined />
            </span>
            <span>
              <span className="pantheon-eyebrow">讨论范围</span>
              <strong id="pantheon-workspace-title">
                {workspace ? workspaceName(workspace) : '选择工作空间'}
              </strong>
              <small title={workspace}>{workspace || '将本次议题限定在一个项目目录内'}</small>
            </span>
          </div>
          <Button
            type="text"
            size="small"
            icon={<FolderOpenOutlined />}
            loading={workspaceBusy}
            onClick={() => void pickWorkspace()}
            disabled={busy}
            data-testid="pantheon-pick-workspace"
          >
            选择文件夹
          </Button>
        </div>
        <div className="pantheon-workspace-options" role="group" aria-label="工作空间">
          {workspaces.length === 0 && <span className="pantheon-muted">正在读取工作空间…</span>}
          {workspaces.map((item, index) => {
            const active = workspace === item.path
            return (
              <button
                key={item.path}
                type="button"
                data-testid={`pantheon-workspace-${item.name || index}`}
                aria-pressed={active}
                className={`pantheon-workspace-option${active ? ' is-selected' : ''}`}
                onClick={() => setWorkspace(item.path)}
                disabled={busy}
                title={item.path}
              >
                <FolderOutlined aria-hidden />
                <span>
                  <strong>{item.name}</strong>
                  <small>{item.current ? '当前工作空间' : item.recent ? '最近使用' : '工作空间'}</small>
                </span>
              </button>
            )
          })}
        </div>
        {workspaceError && <span className="pantheon-workspace-error">{workspaceError}</span>}
      </section>

      <section className="pantheon-seat-section" aria-labelledby="pantheon-seat-title">
        <div className="pantheon-seat-section-head">
          <div>
            <h3 id="pantheon-seat-title">选择参与者</h3>
            <p>至少两位；Grok 走 Gaea 办公工具，Codex / DSH 只读，ZCode 为计划模式。</p>
          </div>
          <Tooltip title="重新检测参与者">
            <Button
              type="text"
              size="small"
              icon={<ReloadOutlined />}
              onClick={onRefreshAgents}
              disabled={busy}
              aria-label="重新检测"
            >
              重新检测
            </Button>
          </Tooltip>
        </div>
        <div className="pantheon-seat-row" role="group" aria-label="讨论参与者">
          {agents.length === 0 && <span className="pantheon-muted">正在检测本机参与者…</span>}
          {agents.map((agent) => {
            const active = selected.includes(agent.id)
            return (
              <Tooltip
                key={agent.id}
                title={agent.available
                  ? [
                      agent.executable || agent.version || agent.kind,
                      agentToolDetail(agent),
                      agent.capabilities?.join(' · '),
                    ].filter(Boolean).join(' · ')
                  : agent.reason || '不可用'}
              >
                <button
                  type="button"
                  data-testid={`pantheon-agent-${agent.id}`}
                  aria-pressed={active}
                  className={`pantheon-seat-chip${active ? ' is-selected' : ''}${agent.available ? '' : ' is-disabled'}`}
                  disabled={!agent.available || busy}
                  onClick={() => toggleAgent(agent.id)}
                  style={{ '--agent-accent': AGENT_ACCENTS[agent.id] || 'var(--pantheon-accent)' } as React.CSSProperties}
                >
                  <span className="pantheon-seat-monogram" aria-hidden>
                    {agent.name.slice(0, 1).toUpperCase()}
                  </span>
                  <span className="pantheon-seat-chip-copy">
                    <strong>{agent.name}</strong>
                    <small>
                      {chairman === agent.id && active
                        ? `负责汇总 · ${agentToolLabel(agent)}`
                        : agent.available
                          ? agentToolLabel(agent)
                          : '未就绪'}
                    </small>
                  </span>
                  {active && (
                    <span className="pantheon-seat-order" aria-label={`第 ${selected.indexOf(agent.id) + 1} 位`}>
                      {String(selected.indexOf(agent.id) + 1).padStart(2, '0')}
                    </span>
                  )}
                </button>
              </Tooltip>
            )
          })}
        </div>
      </section>

      <section className="pantheon-speaking-order" aria-labelledby="pantheon-order-title">
        <div className="pantheon-speaking-order-head">
          <div>
            <h3 id="pantheon-order-title">发言顺序</h3>
            <p>后续轮次自动轮换首位，避免固定顺序影响讨论公平。</p>
          </div>
          <span>{selected.length} 位参与者</span>
        </div>
        <ol>
          {selected.map((id, index) => {
            const agent = agents.find((item) => item.id === id)
            if (!agent) return null
            return (
              <li key={id} data-testid={`pantheon-order-${id}`}>
                <span className="pantheon-order-number">{String(index + 1).padStart(2, '0')}</span>
                <span className="pantheon-order-copy">
                  <strong>{agent.name}</strong>
                  <small>{index === 0 ? '本轮首位' : `本轮第 ${index + 1} 位`}</small>
                </span>
                <span className="pantheon-order-actions">
                  <Button
                    type="text"
                    size="small"
                    icon={<ArrowUpOutlined />}
                    aria-label={`${agent.name} 前移`}
                    disabled={busy || index === 0}
                    onClick={() => moveAgent(index, -1)}
                  />
                  <Button
                    type="text"
                    size="small"
                    icon={<ArrowDownOutlined />}
                    aria-label={`${agent.name} 后移`}
                    disabled={busy || index === selected.length - 1}
                    onClick={() => moveAgent(index, 1)}
                  />
                </span>
              </li>
            )
          })}
        </ol>
      </section>

      <section className="pantheon-topic-desk" aria-labelledby="pantheon-topic-title">
        <div className="pantheon-topic-desk-head">
          <div>
            <span className="pantheon-eyebrow">讨论议题</span>
            <h3 id="pantheon-topic-title">这次想共同探讨什么？</h3>
          </div>
          <span className="pantheon-topic-count">{topic.trim().length > 0 ? `${topic.trim().length} 字` : '必填'}</span>
        </div>
        <Input.TextArea
          data-testid="pantheon-topic"
          className="pantheon-topic-input"
          value={topic}
          onChange={(event) => setTopic(event.target.value)}
          autoSize={{ minRows: 5, maxRows: 14 }}
          placeholder="写下现状、候选方案、已有证据和希望讨论清楚的开放问题。"
          disabled={busy}
        />
        <div className="pantheon-topic-hint">
          <span>建议写清问题背景、可选路径、已知证据和成功标准。</span>
          <span>{selected.length} 位参与</span>
        </div>
      </section>

      <div className="pantheon-chatbar">
        <Button
          size="small"
          type={showEvidence ? 'primary' : 'text'}
          ghost={showEvidence}
          icon={<PaperClipOutlined />}
          data-testid="pantheon-toggle-evidence"
          onClick={() => setShowEvidence((value) => !value)}
          disabled={busy}
        >
          材料
        </Button>
        <Select
          size="small"
          data-testid="pantheon-chairman"
          value={chairman || undefined}
          placeholder="汇总人"
          options={available
            .filter((agent) => selected.includes(agent.id))
            .map((agent) => ({ label: `${agent.name} 负责汇总`, value: agent.id }))}
          onChange={setChairman}
          disabled={busy || selected.length === 0}
          className="pantheon-chatbar-select"
          suffixIcon={<CommentOutlined />}
        />
        <div className="pantheon-round-control" aria-label="讨论轮次设置">
          <div className="pantheon-round-mode" role="group" aria-label="讨论轮次模式">
            <button
              type="button"
              aria-pressed={!untilConsensus}
              data-testid="pantheon-round-manual"
              onClick={() => setUntilConsensus(false)}
              disabled={busy}
            >
              手动
            </button>
            <button
              type="button"
              aria-pressed={untilConsensus}
              data-testid="pantheon-round-auto"
              onClick={() => setUntilConsensus(true)}
              disabled={busy}
            >
              不限
            </button>
          </div>
          {untilConsensus ? (
            <Tooltip title={`全员意见一致即结束；无共识时最多讨论 ${CONSENSUS_SAFE_LIMIT} 轮，可随时终止`}>
              <span className="pantheon-round-auto-label" data-testid="pantheon-round-auto-label">
                至共识 · ≤{CONSENSUS_SAFE_LIMIT} 轮
              </span>
            </Tooltip>
          ) : (
            <>
              <InputNumber
                size="small"
                min={2}
                max={MANUAL_ROUNDS_MAX}
                precision={0}
                value={rounds}
                onChange={(value) => setRounds(
                  Math.max(2, Math.min(MANUAL_ROUNDS_MAX, Number(value) || 2)),
                )}
                disabled={busy}
                aria-label="手动讨论轮次"
                data-testid="pantheon-rounds"
                className="pantheon-round-input"
              />
              <span className="pantheon-round-unit">轮</span>
            </>
          )}
        </div>
        <Button
          size="small"
          type="text"
          data-testid="pantheon-toggle-advanced"
          onClick={() => setShowAdvanced((value) => !value)}
          disabled={busy}
        >
          补充设置 <DownOutlined className={showAdvanced ? 'is-open' : ''} />
        </Button>
        <span className="pantheon-start-spacer" />
        <span
          className={`pantheon-selection-note${startBlockReason ? ' is-blocked' : ''}`}
          aria-live="polite"
        >
          {startBlockReason || '只读运行，不改动工作区'}
        </span>
        <Button
          type="primary"
          size="large"
          data-testid="pantheon-start"
          icon={<SendOutlined />}
          disabled={!canStart}
          loading={busy}
          onClick={() => void start()}
        >
          开始讨论
        </Button>
      </div>

      {showEvidence && (
        <label className="pantheon-field pantheon-field--wide pantheon-disclose">
          <span>讨论材料 <small>粘贴同一份事实基线，让参与者建立在共同证据上</small></span>
          <Input.TextArea
            data-testid="pantheon-evidence"
            value={evidence}
            onChange={(event) => setEvidence(event.target.value)}
            autoSize={{ minRows: 5, maxRows: 14 }}
            placeholder="数据、调研结论、原文片段或已有方案。首版不自动读取工作区文件。"
            disabled={busy}
          />
        </label>
      )}

      {showAdvanced && (
        <div className="pantheon-advanced pantheon-disclose">
          <label className="pantheon-field">
            <span>议题名称 <small>可留空，自动截取议题首行</small></span>
            <Input
              data-testid="pantheon-title"
              value={title}
              maxLength={80}
              onChange={(event) => setTitle(event.target.value)}
              placeholder="例：Gaea 是否应采用事项、事务与证据主轴"
              disabled={busy}
            />
          </label>
          <label className="pantheon-field">
            <span>背景与约束</span>
            <Input.TextArea
              data-testid="pantheon-context"
              value={context}
              onChange={(event) => setContext(event.target.value)}
              autoSize={{ minRows: 3, maxRows: 9 }}
              placeholder="时间、预算、权限、不可违反的红线。"
              disabled={busy}
            />
          </label>
          <label className="pantheon-field">
            <span>成功标准</span>
            <Input.TextArea
              data-testid="pantheon-criteria"
              value={criteria}
              onChange={(event) => setCriteria(event.target.value)}
              autoSize={{ minRows: 3, maxRows: 9 }}
              placeholder="什么算一个可用结论？例如：必须列出反例、成本与适用条件。"
              disabled={busy}
            />
          </label>
        </div>
      )}
    </section>
  )
}
