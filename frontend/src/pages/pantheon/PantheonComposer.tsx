import React, { useEffect, useMemo, useRef, useState } from 'react'
import {
  ArrowDownOutlined,
  ArrowUpOutlined,
  CloseOutlined,
  CommentOutlined,
  DownOutlined,
  FolderOpenOutlined,
  FolderOutlined,
  PaperClipOutlined,
  PictureOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
  SendOutlined,
} from '@ant-design/icons'
import { Alert, Button, Input, InputNumber, Select, Tooltip } from 'antd'
import { pantheonApi } from './api'
import { PantheonAgentAvatar } from './PantheonAgentAvatar'
import type {
  PantheonAgentStatus,
  PantheonAttachment,
  PantheonComposerSeed,
  PantheonStartRequest,
  PantheonWorkspace,
} from './types'
import { AGENT_ACCENTS, agentCapabilityLabel } from './visuals'

interface Props {
  agents: PantheonAgentStatus[]
  busy: boolean
  error: string
  seed?: PantheonComposerSeed | null
  onStart: (request: PantheonStartRequest) => Promise<void>
  onRefreshAgents: () => void
  /** 离线席位 chip 上的显式「设置路径」入口（2026-10-31 实弹拍板：切换
   * 席位永不弹窗，路径配置只走这个明确标注的小钮）。 */
  onConfigureAgent?: (agent: PantheonAgentStatus) => void
}

const MANUAL_ROUNDS_MAX = 20
const CONSENSUS_SAFE_LIMIT = 12
const MAX_IMAGE_ATTACHMENTS = 4
const MAX_IMAGE_BYTES = 20 * 1024 * 1024
const IMAGE_FILTERS = 'png,jpg,jpeg,webp,gif,bmp'

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

function attachmentMIME(path: string): string {
  const ext = path.slice(path.lastIndexOf('.') + 1).toLowerCase()
  if (ext === 'jpg' || ext === 'jpeg') return 'image/jpeg'
  if (ext === 'webp') return 'image/webp'
  if (ext === 'gif') return 'image/gif'
  if (ext === 'bmp') return 'image/bmp'
  return 'image/png'
}

function formatAttachmentSize(size: number): string {
  if (!Number.isFinite(size) || size <= 0) return '图片'
  if (size < 1024 * 1024) return `${Math.max(1, Math.round(size / 1024))}KB`
  return `${(size / (1024 * 1024)).toFixed(1)}MB`
}

// 议题工作台：左侧先定义问题，右侧以连续索引配置范围、席位与发言顺序；
// 材料与补充背景按需在主卷下展开。字段与提交契约保持不变。
export const PantheonComposer: React.FC<Props> = ({
  agents,
  busy,
  error,
  seed,
  onStart,
  onRefreshAgents,
  onConfigureAgent,
}) => {
  const initialEvidence = seedEvidenceText(seed)
  const [title, setTitle] = useState(seed?.title || '')
  const [topic, setTopic] = useState(seed?.topic || '')
  const [context, setContext] = useState(seed?.context || '')
  const [criteria, setCriteria] = useState(seed?.successCriteria || '')
  const [evidence, setEvidence] = useState(initialEvidence)
  const [attachments, setAttachments] = useState<PantheonAttachment[]>(seed?.attachments || [])
  const [attachmentError, setAttachmentError] = useState('')
  const [attachmentBusy, setAttachmentBusy] = useState(false)
  const [selected, setSelected] = useState<string[]>(seed?.agentIds || [])
  const [chairman, setChairman] = useState(seed?.chairmanAgentId || '')
  const [rounds, setRounds] = useState(
    Math.max(1, Math.min(MANUAL_ROUNDS_MAX, seed?.rounds || 2)),
  )
  const [protocol, setProtocol] = useState<'council' | 'independent'>(
    // 2026-10-31 用户拍板：默认独立开场——首轮各抒己见，第二轮起共享信息再议论。
    seed?.protocol === 'council' ? 'council' : 'independent',
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
  const [topicTouched, setTopicTouched] = useState(false)
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

  const pickImages = async () => {
    setAttachmentBusy(true)
    setAttachmentError('')
    try {
      const picked = await pantheonApi.PickFiles(IMAGE_FILTERS)
      const images = picked
        .filter((item) => {
          const ext = item.path.slice(item.path.lastIndexOf('.') + 1).toLowerCase()
          return item.type === 'image' || ['png', 'jpg', 'jpeg', 'webp', 'gif', 'bmp'].includes(ext)
        })
        .map<PantheonAttachment>((item) => ({
          name: item.name,
          path: item.path,
          mime: attachmentMIME(item.path),
          size: item.size || 0,
        }))
      if (images.length === 0) return
      const oversize = images.find((item) => item.size > MAX_IMAGE_BYTES)
      if (oversize) {
        setAttachmentError(`${oversize.name} 超过 20MB，未添加`)
        return
      }
      const seen = new Set(attachments.map((item) => item.path.toLowerCase()))
      const next = [...attachments]
      for (const image of images) {
        const key = image.path.toLowerCase()
        if (seen.has(key)) continue
        if (next.length >= MAX_IMAGE_ATTACHMENTS) break
        seen.add(key)
        next.push(image)
      }
      if (attachments.length + images.length > MAX_IMAGE_ATTACHMENTS) {
        setAttachmentError(`最多添加 ${MAX_IMAGE_ATTACHMENTS} 张图片，多出的未加入`)
      }
      setAttachments(next)
    } catch (err) {
      setAttachmentError(`选择图片失败：${err instanceof Error ? err.message : String(err)}`)
    } finally {
      setAttachmentBusy(false)
    }
  }

  const removeAttachment = (path: string) => {
    setAttachments((current) => current.filter((item) => item.path !== path))
    setAttachmentError('')
  }

  const minSeats = protocol === 'independent' ? 1 : 2
  const topicError = topicTouched && !topic.trim()
    ? '请先写下这次讨论需要回答的问题。'
    : ''
  const protocolLabel = protocol === 'council' ? '圆桌接力' : '独立开场'
  const seatCount = agents.length || 4
  const startBlockReason = !workspace
    ? '请先选择讨论范围'
    : selected.length < minSeats
      ? protocol === 'independent' ? '至少选择一位参与者' : '至少选择两位参与者'
      : !topic.trim()
        ? '写下议题后即可开始'
        : ''
  const canStart = !startBlockReason && !busy && !attachmentBusy

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
      attachments: attachments.map(({ name, path, mime, size }) => ({ name, path, mime, size })),
      agentIds: selected,
      chairmanAgentId: chairman,
      rounds: roundLimit,
      untilConsensus,
      maxRounds: untilConsensus ? roundLimit : undefined,
      readOnly: true,
      protocol,
      parentSessionId: seed?.parentSessionId,
    })
  }

  const extraFields = (
    <>
      {(attachments.length > 0 || attachmentError) && (
        <section className="pantheon-image-evidence" aria-label="图片附件">
          <div className="pantheon-image-evidence-head">
            <span>
              <PictureOutlined aria-hidden />
              <strong>图片附件</strong>
              <small>{attachments.length} / {MAX_IMAGE_ATTACHMENTS}</small>
            </span>
            <small>开始时会先做本地识图，摘要共享给所有席位</small>
          </div>
          {attachmentError && <Alert type="warning" showIcon message={attachmentError} />}
          {attachments.length > 0 && (
            <ul className="pantheon-image-evidence-list">
              {attachments.map((attachment) => (
                <li key={attachment.path} title={attachment.path}>
                  <span className="pantheon-image-evidence-icon" aria-hidden>
                    <PictureOutlined />
                  </span>
                  <span className="pantheon-image-evidence-copy">
                    <strong>{attachment.name}</strong>
                    <small>{attachment.mime} · {formatAttachmentSize(attachment.size)}</small>
                  </span>
                  <Button
                    type="text"
                    size="small"
                    icon={<CloseOutlined />}
                    aria-label={`移除图片 ${attachment.name}`}
                    disabled={busy}
                    onClick={() => removeAttachment(attachment.path)}
                  />
                </li>
              ))}
            </ul>
          )}
        </section>
      )}

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
    </>
  )

  return (
    <section
      className="pantheon-composer pantheon-composer--studio"
      data-testid="pantheon-composer"
      aria-labelledby="pantheon-compose-title"
    >
      <header className="pantheon-composer-head">
        <div className="pantheon-composer-head-copy">
          <span className="pantheon-hero-note">
            <SafetyCertificateOutlined aria-hidden />
            只读运行 · 不改动工作区
          </span>
          <span className="pantheon-composer-kicker">New council · 立题</span>
          <h2 id="pantheon-compose-title">
            把问题说清楚，<br />让判断有处落脚。
          </h2>
          <p>先写下真正需要裁决的问题，再选择工作范围、参与席位与讨论路径。</p>
        </div>
        <ol className="pantheon-composer-steps" aria-label="发起讨论步骤">
          <li className="is-current">
            <b>01</b>
            <span>立题</span>
          </li>
          <li>
            <b>02</b>
            <span>组席</span>
          </li>
          <li>
            <b>03</b>
            <span>开议</span>
          </li>
        </ol>
      </header>

      {error && <Alert type="error" showIcon message={error} className="pantheon-alert" />}

      <div className="pantheon-composer-studio">
        <div className="pantheon-composer-main">
          <section
            className="pantheon-compose-card pantheon-topic-sheet"
            data-testid="pantheon-compose-card"
            aria-labelledby="pantheon-topic-label"
          >
            <header className="pantheon-topic-sheet-head">
              <div>
                <span className="pantheon-eyebrow">议题底稿 · Question 01</span>
                <label id="pantheon-topic-label" htmlFor="pantheon-topic-field">
                  我们要一起判断什么？
                  <em>必填</em>
                </label>
              </div>
              <span className="pantheon-topic-sheet-mark" aria-hidden>?</span>
            </header>
            <Input.TextArea
              id="pantheon-topic-field"
              data-testid="pantheon-topic"
              className={`pantheon-topic-input${topicError ? ' has-error' : ''}`}
              value={topic}
              onChange={(event) => setTopic(event.target.value)}
              onBlur={() => setTopicTouched(true)}
              autoSize={{ minRows: 5, maxRows: 14 }}
              placeholder="描述现状、候选路径，以及你希望这群 Agent 真正帮助你判断的争议点。"
              disabled={busy}
              aria-describedby="pantheon-topic-help pantheon-topic-meta"
            />
            <p className="pantheon-topic-help" id="pantheon-topic-help">
              一句话也可以；写清目标、限制和已知证据，讨论会更聚焦。
            </p>
            {topicError && (
              <p className="pantheon-topic-error" id="pantheon-topic-error" role="alert">
                {topicError}
              </p>
            )}
            <div className="pantheon-compose-meta" id="pantheon-topic-meta">
              <span>{topic.trim().length > 0 ? `${topic.trim().length} 字` : '尚未填写'}</span>
              <span>{selected.length} 位参与</span>
              <span>{protocolLabel}</span>
            </div>
            <div className="pantheon-chatbar">
              <div className="pantheon-toolbar-options">
                <Button
                  size="small"
                  type={showEvidence ? 'primary' : 'text'}
                  ghost={showEvidence}
                  icon={<PaperClipOutlined />}
                  data-testid="pantheon-toggle-evidence"
                  aria-expanded={showEvidence}
                  onClick={() => setShowEvidence((value) => !value)}
                  disabled={busy}
                >
                  材料
                </Button>
                <Button
                  size="small"
                  type={attachments.length > 0 ? 'primary' : 'text'}
                  ghost={attachments.length > 0}
                  icon={<PictureOutlined />}
                  loading={attachmentBusy}
                  data-testid="pantheon-pick-images"
                  onClick={() => void pickImages()}
                  disabled={busy}
                >
                  图片{attachments.length > 0 ? ` ${attachments.length}` : ''}
                </Button>
                <Tooltip title="独立开场：第 1 轮各自独立作答；圆桌接力：同轮发言按顺序互相可见。">
                  <Select
                    size="small"
                    data-testid="pantheon-protocol"
                    value={protocol}
                    options={[
                      { label: '独立开场', value: 'independent' },
                      { label: '圆桌接力', value: 'council' },
                    ]}
                    onChange={(value) => setProtocol(value)}
                    disabled={busy}
                    className="pantheon-chatbar-select"
                    aria-label="议事路径"
                  />
                </Tooltip>
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
                  aria-label="汇总人"
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
                        min={1}
                        max={MANUAL_ROUNDS_MAX}
                        precision={0}
                        value={rounds}
                        onChange={(value) => setRounds(
                          Math.max(1, Math.min(MANUAL_ROUNDS_MAX, Number(value) || 2)),
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
                  aria-expanded={showAdvanced}
                  onClick={() => setShowAdvanced((value) => !value)}
                  disabled={busy}
                >
                  背景与标准 <DownOutlined className={showAdvanced ? 'is-open' : ''} />
                </Button>
              </div>
              <div className="pantheon-toolbar-launch">
                <span
                  className={`pantheon-selection-note${startBlockReason ? ' is-blocked' : ''}`}
                  aria-live="polite"
                >
                  {startBlockReason || '议题已就绪'}
                </span>
                <Tooltip title={canStart ? '开始讨论' : startBlockReason}>
                  <Button
                    type="primary"
                    data-testid="pantheon-start"
                    icon={<SendOutlined />}
                    iconPosition="end"
                    aria-label="开始讨论"
                    disabled={!canStart}
                    loading={busy}
                    onClick={() => void start()}
                    className="pantheon-start"
                  >
                    开始讨论
                  </Button>
                </Tooltip>
              </div>
            </div>
          </section>
          <div className="pantheon-composer-extra">
            {extraFields}
          </div>
        </div>

        <aside className="pantheon-composer-config" aria-label="议事配置">
          <header className="pantheon-config-rail-head">
            <div>
              <span className="pantheon-eyebrow">Council setup</span>
              <h3>议事配置</h3>
            </div>
            <span className="pantheon-config-ready">
              <i aria-hidden />
              {available.length} / {seatCount} 席可用
            </span>
          </header>

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
            <h3 id="pantheon-seat-title">参与席位</h3>
            <p>点击席位加入或退出本轮讨论。</p>
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
              <button
                key={agent.id}
                type="button"
                data-testid={`pantheon-agent-${agent.id}`}
                aria-pressed={active}
                className={`pantheon-seat-chip${active ? ' is-selected' : ''}${agent.available ? '' : ' is-disabled'}`}
                disabled={!agent.available || busy}
                onClick={() => toggleAgent(agent.id)}
                title={`${agent.name} · ${agentCapabilityLabel(agent.id)}`}
                style={{ '--agent-accent': AGENT_ACCENTS[agent.id] || 'var(--pantheon-accent)' } as React.CSSProperties}
              >
                <PantheonAgentAvatar
                  agentId={agent.id}
                  className="pantheon-seat-monogram"
                  status={agent.available ? 'online' : 'offline'}
                />
                <span className="pantheon-seat-chip-copy">
                  <strong>{agent.name}</strong>
                  <small>
                    {chairman === agent.id && active
                      ? `负责汇总 · ${agentCapabilityLabel(agent.id)}`
                      : agent.available
                        ? agentCapabilityLabel(agent.id)
                        : agent.reason || '未就绪'}
                  </small>
                  {!agent.available && onConfigureAgent && (
                    <span
                      role="button"
                      tabIndex={-1}
                      className="pantheon-seat-chip-config"
                      data-testid={`pantheon-agent-config-${agent.id}`}
                      aria-label={`设置 ${agent.name} 路径`}
                      onClick={(event) => {
                        event.stopPropagation()
                        onConfigureAgent(agent)
                      }}
                    >
                      设置路径
                    </span>
                  )}
                </span>
                {active && (
                  <span className="pantheon-seat-order" aria-label={`第 ${selected.indexOf(agent.id) + 1} 位`}>
                    {String(selected.indexOf(agent.id) + 1).padStart(2, '0')}
                  </span>
                )}
              </button>
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
              <li
                key={id}
                data-testid={`pantheon-order-${id}`}
                data-agent-id={id}
                style={{
                  '--agent-accent': AGENT_ACCENTS[id] || 'var(--pantheon-accent)',
                } as React.CSSProperties}
              >
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

        </aside>
      </div>
    </section>
  )
}
