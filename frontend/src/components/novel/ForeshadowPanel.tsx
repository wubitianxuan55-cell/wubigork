// ForeshadowPanel.tsx — 伏笔登记表面板（v4.3f 闭环版 + t1-P4 调度可视面）
// 登记→埋设→回收闭环：GetForeshadows 展示 + SaveForeshadows 全量写回。
// ①「登记伏笔」表单（类别/描述/埋设章节/是否长线，manual_ 前缀 ID）；
// ② 每条状态流转按钮（planted→hinted→revealed，revealed 可回退）；
// ③ 删除（confirm）；④ 描述可编辑；⑤「一致性体检」（LintForeshadows）：
//    概要统计 + findings 直显（severity Tag + 说明 + 条目描述 + 章节引用）；
// ⑥ t1-P4：后端统计行（分状态+超期）、紧急度 Badge（后端投影不落库，D7 前端不自算）、
//    生命周期清理入口（删本章/重分析前清理/项目重置，手动条目后端护栏保护）、
//    上次分析同步结果（跳过原因可见，D3 不静默）。
// 纯逻辑（ID 生成/状态机/载荷收窄）抽在 foreshadowLogic.ts。
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  Alert, Button, Checkbox, Empty, Input, InputNumber, message, Popconfirm, Select, Spin, Tag, Tooltip,
} from 'antd'
import {
  CheckCircleOutlined, ClearOutlined, DeleteOutlined, EditOutlined, FlagOutlined, PlusOutlined, ReloadOutlined, SafetyCertificateOutlined,
} from '@ant-design/icons'
import { app } from '../../gaea/lib/bridge'
import type { ForeshadowLintReport, ForeshadowStatsReport, ForeshadowSyncResult } from '../../gaea/lib/bridge/novel'
import type { ForeshadowItemData, ForeshadowStatus } from '../../types'
import {
  advanceForeshadowStatus,
  buildManualForeshadow,
  foreshadowFlowLabel,
  formatPlantedIn,
  normalizeForeshadowItems,
  stripForeshadowUrgency,
} from './foreshadowLogic'

const STATUS_META: Record<ForeshadowStatus, { label: string; color: string }> = {
  planted: { label: '已埋设', color: 'blue' },
  hinted: { label: '已暗示', color: 'gold' },
  revealed: { label: '已回收', color: 'green' },
}

/** 体检 severity → Tag 色/文案（high=红「重」/ medium=gold「中」/ low=default「轻」，
 *  未知档透传原文显示；口径对齐 bridge/novel.ts ForeshadowLintFinding）。 */
const LINT_SEVERITY_META: Record<string, { color: string; label: string }> = {
  high: { color: 'red', label: '重' },
  medium: { color: 'gold', label: '中' },
  low: { color: 'default', label: '轻' },
}

const CATEGORY_LABELS: Record<string, string> = {
  character: '角色',
  plot: '剧情',
  world: '世界观',
  relationship: '关系',
}

const CATEGORY_OPTIONS = Object.entries(CATEGORY_LABELS).map(([value, label]) => ({ value, label }))

/** 紧急度 Badge（后端投影 level 0-3；阈值在后端，前端不自算 D7）。 */
const URGENCY_META: Record<number, { color: string; label: string }> = {
  3: { color: 'red', label: '已超期' },
  2: { color: 'orange', label: '急需回收' },
  1: { color: 'gold', label: '需关注' },
}

/** 回收时机四值中文（tooltip 用）。 */
const RESOLVE_STATUS_LABELS: Record<string, string> = {
  must_resolve_now: '本章必须回收',
  overdue: '已超期',
  not_yet: '未到回收时机',
  no_plan: '未填计划回收章',
}

interface ForeshadowPanelProps {
  /** 未打开项目时仅展示空态引导，不触发加载 */
  disabled?: boolean
}

const ForeshadowPanel: React.FC<ForeshadowPanelProps> = ({ disabled }) => {
  const [items, setItems] = useState<ForeshadowItemData[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const loadToken = useRef(0)

  // 登记表单
  const [formOpen, setFormOpen] = useState(false)
  const [category, setCategory] = useState('plot')
  const [description, setDescription] = useState('')
  const [chapterNum, setChapterNum] = useState(1)
  const [isLongTerm, setIsLongTerm] = useState(false)
  // 行内描述编辑
  const [editing, setEditing] = useState<{ id: string; desc: string } | null>(null)
  // 一致性体检（LintForeshadows 结果直显；保留上次报告直至下次体检）
  const [linting, setLinting] = useState(false)
  const [lintReport, setLintReport] = useState<ForeshadowLintReport | null>(null)
  // t1-P4：后端统计 + 上次分析同步（都随 load 拉取；失败静默降级）
  const [beStats, setBeStats] = useState<ForeshadowStatsReport | null>(null)
  const [lastSync, setLastSync] = useState<ForeshadowSyncResult | null>(null)
  // 生命周期清理工具（展开态 + 共享章号）
  const [cleanOpen, setCleanOpen] = useState(false)
  const [cleanChapter, setCleanChapter] = useState(1)
  const [onlyAnalysis, setOnlyAnalysis] = useState(true)

  const load = useCallback(async () => {
    const token = ++loadToken.current
    if (disabled) {
      setItems([])
      setBeStats(null)
      setLastSync(null)
      setLoading(false)
      setError('')
      return
    }
    setLoading(true)
    setError('')
    try {
      const [res, stat] = await Promise.all([
        app.GetForeshadows(),
        // 统计/同步为增量绑定（t1-P4）：旧桥接/局部 mock 缺失时静默降级
        typeof app.GetForeshadowStats === 'function'
          ? app.GetForeshadowStats(0).catch(() => null)
          : Promise.resolve(null),
      ])
      if (token !== loadToken.current) return
      setItems(normalizeForeshadowItems(res))
      setBeStats(stat)
      setLastSync(null)
      if (typeof app.GetLastForeshadowSync === 'function') {
        void app.GetLastForeshadowSync()
          .then((r) => { if (token === loadToken.current) setLastSync(r) })
          .catch(() => { if (token === loadToken.current) setLastSync(null) }) // 尚未分析=不展示
      }
    } catch (err: unknown) {
      if (token !== loadToken.current) return
      setError(err instanceof Error ? err.message : '伏笔加载失败')
    } finally {
      if (token === loadToken.current) setLoading(false)
    }
  }, [disabled])

  useEffect(() => { void load() }, [load])

  // 一致性体检：拉报告直显（Go 侧字段可能 omitempty/findings 可能为 null，渲染处 ?. ?? 防御）
  const runLint = useCallback(async () => {
    setLinting(true)
    try {
      const report = await app.LintForeshadows()
      setLintReport(report)
    } catch (err: unknown) {
      message.error(`一致性体检失败：${err instanceof Error ? err.message : '未知错误'}`)
    } finally {
      setLinting(false)
    }
  }, [])

  // persist 全量写回（乐观更新 + 失败回滚提示）
  const persist = useCallback(async (prev: ForeshadowItemData[], next: ForeshadowItemData[]) => {
    setItems(next)
    try {
      // A5：urgency 是运行时投影不落库，写回载荷剥离
      await app.SaveForeshadows(JSON.stringify(stripForeshadowUrgency(next)))
      return true
    } catch (err: unknown) {
      setItems(prev)
      message.error(`伏笔保存失败，已回滚：${err instanceof Error ? err.message : '未知错误'}`)
      return false
    }
  }, [])

  // 生命周期清理（t1-P3 绑定，t1-P4 面板消费）：后端护栏=手动条目只重置不删除。
  // 全部经 Popconfirm 二次确认；结果 toast + 重载登记表。
  const runClean = useCallback(async (kind: 'cleanChapter' | 'deleteChapter' | 'resetProject') => {
    try {
      if (kind === 'resetProject') {
        const res = await app.ClearProjectForeshadowsForReset()
        message.success(`项目重置完成：删除分析条目 ${res.deleted ?? 0} 条，重置手动条目 ${res.resetManual ?? 0} 条`)
      } else {
        const file = formatPlantedIn(cleanChapter)
        if (kind === 'cleanChapter') {
          const res = await app.CleanChapterAnalysisForeshadows(file)
          message.success(`已清理第 ${cleanChapter} 章（${file}）：删除 ${res.deleted ?? 0} 条，回退回收 ${res.rolledBack ?? 0} 条`)
        } else {
          const res = await app.DeleteChapterForeshadows(file, onlyAnalysis)
          message.success(`已删除第 ${cleanChapter} 章（${file}）伏笔 ${res.deleted ?? 0} 条${onlyAnalysis ? '（仅分析来源）' : ''}`)
        }
      }
      await load()
    } catch (err: unknown) {
      message.error(`清理失败：${err instanceof Error ? err.message : '未知错误'}`)
    }
  }, [cleanChapter, onlyAnalysis, load])

  const register = () => {
    const desc = description.trim()
    if (!desc) {
      message.warning('请填写伏笔描述')
      return
    }
    const entry = buildManualForeshadow({ category, description: desc, chapterNum, isLongTerm })
    void persist(items, [...items, entry])
    setDescription('')
    setIsLongTerm(false)
    setFormOpen(false)
    message.success('伏笔已登记')
  }

  const flow = (id: string) => {
    const target = items.find((it) => it.id === id)
    if (!target) return
    const nextStatus = advanceForeshadowStatus(target.status)
    void persist(items, items.map((it) => (it.id !== id ? it : {
      ...it,
      status: nextStatus,
      revealed_in: nextStatus === 'revealed' ? (it.revealed_in ?? it.planted_in) : undefined,
    })))
  }

  const remove = (id: string) => {
    void persist(items, items.filter((it) => it.id !== id))
  }

  const saveEdit = () => {
    if (!editing) return
    const desc = editing.desc.trim()
    if (!desc) {
      message.warning('描述不能为空')
      return
    }
    const id = editing.id
    setEditing(null)
    void persist(items, items.map((it) => (it.id !== id ? it : { ...it, description: desc })))
  }

  const stats = useMemo(() => {
    const total = items.length
    const revealed = items.filter((i) => i.status === 'revealed').length
    const hinted = items.filter((i) => i.status === 'hinted').length
    const planted = items.filter((i) => i.status === 'planted').length
    const rate = total === 0 ? 0 : Math.round((revealed / total) * 100)
    return { total, revealed, hinted, planted, rate }
  }, [items])

  // 体检发现（Go 侧 findings 可能为 null → ?? [] 防御）
  const lintFindings = lintReport?.findings ?? []

  const flowLegend = (
    <div className="fs-flow-legend">
      <span>埋设</span><span className="fs-arrow">→</span>
      <span>暗示</span><span className="fs-arrow">→</span>
      <span>回收</span>
    </div>
  )

  return (
    <div className="novel-panel fs-panel" style={{ flex: 1, minWidth: 0, minHeight: 0, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
      <div className="novel-panel-head" style={{ flexWrap: 'wrap', rowGap: 4 }}>
        <span className="novel-panel-title"><FlagOutlined />伏笔登记</span>
        <div style={{ flex: 1 }} />
        <span className="novel-setting-meta">
          回收率 {stats.rate}%（{stats.revealed}/{stats.total}）
        </span>
        <Button size="small" icon={<PlusOutlined />} disabled={disabled} onClick={() => setFormOpen((o) => !o)}>
          登记伏笔
        </Button>
        <Button
          size="small" icon={<SafetyCertificateOutlined />} loading={linting} disabled={disabled}
          onClick={() => void runLint()}
        >
          一致性体检
        </Button>
        <Button
          size="small" icon={<ClearOutlined />} disabled={disabled}
          onClick={() => setCleanOpen((o) => !o)}
        >
          清理
        </Button>
        <Button size="small" icon={<ReloadOutlined />} onClick={() => void load()} loading={loading} disabled={disabled}>
          刷新
        </Button>
      </div>
      <div className="novel-setting-body" style={{ padding: 8 }}>
        {disabled ? (
          <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="请先在「书架」打开一部小说项目" style={{ margin: 'auto' }} />
        ) : loading ? (
          <div style={{ margin: 'auto' }}><Spin size="small" /></div>
        ) : error ? (
          <Alert
            type="error" showIcon style={{ width: '100%' }}
            message="伏笔加载失败"
            description={error}
            action={<Button size="small" icon={<ReloadOutlined />} onClick={() => void load()}>重试</Button>}
          />
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0 }}>
            {/* ① 手工登记表单 */}
            {formOpen && (
              <div className="fs-form" style={{ display: 'flex', flexDirection: 'column', gap: 6, marginBottom: 8, flexShrink: 0 }}>
                <div style={{ display: 'flex', gap: 6, alignItems: 'center', flexWrap: 'wrap' }}>
                  <Select size="small" style={{ width: 96 }} value={category} onChange={setCategory} options={CATEGORY_OPTIONS} />
                  <span className="novel-setting-meta">埋设章节</span>
                  <InputNumber size="small" min={1} max={9999} style={{ width: 72 }} value={chapterNum} onChange={(v) => setChapterNum(Number(v) || 1)} />
                  <Checkbox checked={isLongTerm} onChange={(e) => setIsLongTerm(e.target.checked)} style={{ fontSize: 12 }}>
                    长线伏笔
                  </Checkbox>
                </div>
                <Input.TextArea
                  size="small" rows={2} maxLength={500}
                  placeholder="伏笔描述（如：主角左臂旧伤的来历）"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                />
                <div style={{ display: 'flex', gap: 6, justifyContent: 'flex-end' }}>
                  <Button size="small" type="primary" icon={<FlagOutlined />} onClick={register}>登记</Button>
                  <Button size="small" onClick={() => setFormOpen(false)}>取消</Button>
                </div>
              </div>
            )}
            {/* ⑤ 一致性体检报告（直显从简：概要行 + findings 列表，Go 侧字段 omitempty → ?. ?? 防御） */}
            {lintReport && (
              <div
                className="fs-lint"
                style={{
                  flexShrink: 0, marginBottom: 8, padding: '6px 8px', borderRadius: 6,
                  border: '1px solid var(--color-border, var(--md-sys-color-outline-variant))',
                  display: 'flex', flexDirection: 'column', gap: 6,
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
                  <span style={{ fontSize: 12, fontWeight: 600 }}><SafetyCertificateOutlined />体检报告</span>
                  <div style={{ flex: 1 }} />
                  <span className="novel-setting-meta">
                    全书 {lintReport.totalChapters ?? 0} 章 · 登记 {lintReport.items ?? 0} 条 · 已埋 {lintReport.planted ?? 0} / 暗示 {lintReport.hinted ?? 0} / 回收 {lintReport.revealed ?? 0}（长线 {lintReport.longTerm ?? 0}）
                  </span>
                </div>
                {lintFindings.length === 0 ? (
                  <div style={{ fontSize: 12, color: 'var(--color-success, #52c41a)', display: 'flex', alignItems: 'center', gap: 4 }}>
                    <CheckCircleOutlined />未发现一致性问题
                  </div>
                ) : (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                    {lintFindings.map((f, i) => {
                      const sev = LINT_SEVERITY_META[f.severity] ?? { color: 'default', label: f.severity }
                      return (
                        <div key={`${f.foreshadowId}-${i}`} style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                          <div style={{ display: 'flex', alignItems: 'baseline', gap: 6, flexWrap: 'wrap' }}>
                            <Tag style={{ marginInlineEnd: 0, fontSize: 11 }} color={sev.color}>{sev.label}</Tag>
                            <span style={{ fontSize: 12 }}>{f.message}</span>
                            {f.chapter && <span className="novel-setting-meta">→ {f.chapter}</span>}
                          </div>
                          {f.itemDesc && <span className="novel-setting-meta">条目：{f.itemDesc}</span>}
                        </div>
                      )
                    })}
                  </div>
                )}
              </div>
            )}
            {/* ⑥ 生命周期清理工具（后端护栏：手动条目只重置不删除）+ 上次同步结果 */}
            {cleanOpen && (
              <div
                className="fs-clean"
                style={{
                  flexShrink: 0, marginBottom: 8, padding: '6px 8px', borderRadius: 6,
                  border: '1px solid var(--color-border, var(--md-sys-color-outline-variant))',
                  display: 'flex', flexDirection: 'column', gap: 6,
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
                  <span style={{ fontSize: 12, fontWeight: 600 }}><ClearOutlined />生命周期清理</span>
                  <span className="novel-setting-meta">手动登记条目永不批量删除（仅分析来源可删）</span>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexWrap: 'wrap' }}>
                  <span className="novel-setting-meta">章节</span>
                  <InputNumber size="small" min={1} max={9999} style={{ width: 72 }} value={cleanChapter} onChange={(v) => setCleanChapter(Number(v) || 1)} />
                  <Popconfirm
                    title={`重分析前清理第 ${cleanChapter} 章？`}
                    description="删除该章分析来源伏笔，回退该章回收记录（手动条目保留）"
                    okText="清理" cancelText="取消"
                    onConfirm={() => void runClean('cleanChapter')}
                  >
                    <Button size="small">重分析前清理</Button>
                  </Popconfirm>
                  <Popconfirm
                    title={`删除第 ${cleanChapter} 章伏笔？`}
                    description={onlyAnalysis ? '仅删除分析来源条目' : '将连同手动条目一起删除'}
                    okText="删除" cancelText="取消"
                    onConfirm={() => void runClean('deleteChapter')}
                  >
                    <Button size="small" danger>删除本章伏笔</Button>
                  </Popconfirm>
                  <Checkbox checked={onlyAnalysis} onChange={(e) => setOnlyAnalysis(e.target.checked)} style={{ fontSize: 12 }}>
                    只删分析来源
                  </Checkbox>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  <Popconfirm
                    title="重置整个项目的伏笔登记表？"
                    description="删除全部分析来源条目；手动条目重置为待规划（保留内容）"
                    okText="重置" cancelText="取消"
                    onConfirm={() => void runClean('resetProject')}
                  >
                    <Button size="small" danger>项目重置</Button>
                  </Popconfirm>
                </div>
              </div>
            )}
            {/* ⑥ 上次分析同步结果（常显；跳过原因可见 D3 不静默） */}
            {lastSync && (
              <div
                style={{
                  flexShrink: 0, marginBottom: 8, padding: '4px 8px', borderRadius: 6, fontSize: 12,
                  border: '1px dashed var(--color-border, var(--md-sys-color-outline-variant))',
                  display: 'flex', flexDirection: 'column', gap: 2,
                }}
              >
                <span className="novel-setting-meta">
                  上次分析同步：回收 {lastSync.resolvedCount ?? 0} · 新埋 {lastSync.createdCount ?? 0} · 内容匹配 {lastSync.matchedByContent ?? 0} · 跳过 {lastSync.skippedResolveCount ?? 0}
                </span>
                {(lastSync.skippedReasons ?? []).slice(0, 2).map((r, i) => (
                  <span key={i} className="novel-setting-meta">· {r.message}</span>
                ))}
                {(lastSync.skippedReasons?.length ?? 0) > 2 && (
                  <span className="novel-setting-meta">· ……其余 {lastSync.skippedReasons!.length - 2} 条略</span>
                )}
              </div>
            )}
            {items.length === 0 ? (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={
                  <span>
                    还没有伏笔登记。生成章节并分析后会自动登记，也可点「登记伏笔」手工记录。
                  </span>
                }
                style={{ margin: 'auto' }}
              />
            ) : (
              <div className="fs-list" style={{ flex: 1, minHeight: 0, overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 6 }}>
                <div className="fs-stats-row">
                  {flowLegend}
                  <div style={{ flex: 1 }} />
                  {/* t1-P4：优先后端统计口径（含待规划/部分/废弃/长线）；失败降级前端计数 */}
                  {beStats ? (
                    <span className="novel-setting-meta">
                      埋设 {beStats.planted ?? 0} · 暗示 {beStats.hinted ?? 0} · 回收 {beStats.resolved ?? 0} · 部分 {beStats.partiallyResolved ?? 0} · 待规划 {beStats.pending ?? 0} · 废弃 {beStats.abandoned ?? 0} · 长线 {beStats.longTermCount ?? 0}
                    </span>
                  ) : (
                    <span className="novel-setting-meta">埋设 {stats.planted} · 暗示 {stats.hinted} · 回收 {stats.revealed}</span>
                  )}
                  {(beStats?.overdueCount ?? 0) > 0 && (
                    <Tooltip title={`有 ${beStats!.overdueCount} 条伏笔已过计划回收章，正以硬约束进入章节生成上下文`}>
                      <Tag style={{ marginInlineEnd: 0, fontSize: 11 }} color="red">超期 {beStats!.overdueCount}</Tag>
                    </Tooltip>
                  )}
                </div>
                {items.map((it) => (
                  <div key={it.id} className="fs-item">
                    <div className="fs-item-head">
                      <Tag style={{ marginInlineEnd: 6, fontSize: 11 }} color="default">{CATEGORY_LABELS[it.category] || it.category}</Tag>
                      {editing?.id === it.id ? (
                        <Input.TextArea
                          size="small" rows={2} autoFocus style={{ flex: 1, minWidth: 0 }}
                          value={editing.desc}
                          onChange={(e) => setEditing({ id: it.id, desc: e.target.value })}
                        />
                      ) : (
                        <span className="fs-item-desc">{it.description}</span>
                      )}
                      <div style={{ flex: 1 }} />
                      <Tooltip title={`章节：${it.planted_in}`}>
                        <span className="novel-setting-meta">{it.planted_in}</span>
                      </Tooltip>
                      <Tag style={{ marginInlineEnd: 0, fontSize: 11 }} color={STATUS_META[it.status].color}>
                        {STATUS_META[it.status].label}
                      </Tag>
                      {/* t1-P4：紧急度 Badge（后端运行时投影；阈值后端算 D7） */}
                      {it.urgency && it.urgency.level > 0 && (
                        <Tooltip title={`${RESOLVE_STATUS_LABELS[it.urgency.resolveStatus] ?? it.urgency.resolveStatus}${it.urgency.overdueChapters ? `·已超 ${it.urgency.overdueChapters} 章` : it.urgency.remainingChapters > 0 ? `·还有 ${it.urgency.remainingChapters} 章` : ''}`}>
                          <Tag style={{ marginInlineEnd: 0, fontSize: 11 }} color={URGENCY_META[it.urgency.level]?.color ?? 'default'}>
                            {URGENCY_META[it.urgency.level]?.label ?? `紧急度${it.urgency.level}`}
                          </Tag>
                        </Tooltip>
                      )}
                    </div>
                    <div className="fs-item-foot" style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                      {it.is_long_term && <Tag style={{ fontSize: 10, marginInlineEnd: 0 }} color="purple">长期伏笔</Tag>}
                      {it.status === 'revealed' && it.revealed_in && (
                        <span className="novel-setting-meta">回收于 {it.revealed_in}</span>
                      )}
                      <div style={{ flex: 1 }} />
                      {editing?.id === it.id ? (
                        <>
                          <Button size="small" type="link" style={{ padding: 0 }} onClick={saveEdit}>保存</Button>
                          <Button size="small" type="link" style={{ padding: 0 }} onClick={() => setEditing(null)}>取消</Button>
                        </>
                      ) : (
                        <>
                          {/* ② 状态流转：planted→hinted→revealed，revealed 可回退 */}
                          <Button size="small" type="link" style={{ padding: 0 }} onClick={() => flow(it.id)}>
                            {foreshadowFlowLabel(it.status)}
                          </Button>
                          {/* ④ 描述编辑 */}
                          <Button
                            size="small" type="link" icon={<EditOutlined />}
                            aria-label={`编辑伏笔：${it.description}`}
                            onClick={() => setEditing({ id: it.id, desc: it.description })}
                          />
                          {/* ③ 删除（confirm） */}
                          <Popconfirm title="删除该伏笔？" okText="删除" cancelText="取消" onConfirm={() => remove(it.id)}>
                            <Button
                              size="small" type="link" danger icon={<DeleteOutlined />}
                              aria-label={`删除伏笔：${it.description}`}
                            />
                          </Popconfirm>
                        </>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

export default ForeshadowPanel
