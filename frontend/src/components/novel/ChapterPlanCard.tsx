// ChapterPlanCard.tsx — 章节计划卡（「章节计划闭环」刀1 线D：前端计划卡与硬闸提示）。
//
// 规格：docs/gaea-longform-novel-system-2026-09.md §7.2（绑定）/§7.5（分线）/§7.6（验收）。
// 契约以 Go 侧为准：types.ChapterPlan 七字段（internal/types/plot_v2.go:11-22，json tag 逐字
// 对齐）、types.PlanGateReport / PlanDeviation / PlanProblem（internal/types/plan_v1.go:20-69）。
//
// 绑定面（线C 实现、主代理收口时生成 wailsjs + bridge 类型）：
//   NovelChapterPlanGet(chapterNum) → ChapterPlan | null
//   NovelChapterPlanSave(chapterNum, planJSON) → void（校验失败 reject，错误中文可读）
//   NovelChapterPlanPropose(chapterNum, direction) → ChapterPlan（**草案不落盘**；
//     direction=创作方向（分支意向/剧情要求），空串=既有口径，v4.450.0 三点打通）
//   NovelChapterPlanDeviation(chapterNum) → PlanDeviation
//   NovelChapterGatePrecheck(chapterNum) → PlanGateReport（硬闸唯一判据来源）
// 生成物在收口阶段才产出 → 本组件自建本地 interface 并 `app as unknown as ...` 收窄，
// 不 import wailsjs（线D 足迹纪律：不碰 bridge/**、wailsjs/**、bindingNames.ts）。
//
// 纪律：
//   1) 打开/切章一律 seq 守卫（先例 BookSearchModal.pickSeqRef / PromptWorkshopPanel.selectSeqRef
//      / ChapterAnalysisPanel.cmpSeqRef）——迟到的旧章计划写进新章卡片，作者会拿 A 章计划
//      审批落盘到 B 章；偏差报告同源。
//   2) AI 只产出草案：Propose 成功进入**编辑态**，落盘必须作者点「保存计划」。
//   3) 保存失败如实显示后端错误（含被点名的重复事件）且**不丢表单**。
//   4) 不用 EventsOff、不写 localStorage（本组件零持久化）。
import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Alert, Button, Empty, Input, InputNumber, Select, Space, Spin, Tag, message } from 'antd'
import { app } from '../../gaea/lib/bridge'

// ── 契约类型（本地副本；字段名/JSON 键以 Go json tag 为准）─────────────────

/** 章节计划（Go types.ChapterPlan，internal/types/plot_v2.go:11-22）。 */
export interface ChapterPlan {
  /** Go: sub_index —— 不在七字段编辑面内，读回时原样保留（保存不截断大纲序）。 */
  sub_index?: number
  /** Go: title —— 同上，仅透传。 */
  title?: string
  /** Go: plot_summary（200-300 字）。 */
  plot_summary: string
  /** Go: key_events（2-4 条，跨章不得重复）。 */
  key_events: string[]
  /** Go: character_focus。 */
  character_focus: string[]
  /** Go: emotional_tone。 */
  emotional_tone: string
  /** Go: narrative_goal。 */
  narrative_goal: string
  /** Go: conflict_type。 */
  conflict_type: string
  /** Go: ending_type（悬念|冲突升级|情节转折|情感收尾|自然过渡）。 */
  ending_type: string
  /** Go: estimated_words,omitempty。 */
  estimated_words?: number
}

/** 计划/大纲问题条（Go types.PlanProblem）。 */
export interface PlanProblem {
  code: string
  /** S1|S2 阻断；S3|S4 仅提示。 */
  severity: string
  message: string
  evidence?: string
}

/** 生成前预检报告（Go types.PlanGateReport；硬闸唯一判据来源）。 */
export interface PlanGateReport {
  chapterNum: number
  allowed: boolean
  hasPlan: boolean
  /** 缺失齐备性字段的中文标签（后端给中文，前端原样展示）。 */
  missing: string[]
  planProblems: PlanProblem[]
  outlineIssues: PlanProblem[]
  blocking: boolean
}

/** 计划 vs 实际偏差报告（Go types.PlanDeviation；缺失/未分析均为正常态）。 */
export interface PlanDeviation {
  chapterNum: number
  hasPlan: boolean
  analyzed: boolean
  missingEvents: string[]
  endingMismatch: boolean
  plannedEnding: string
  actualEnding: string
  emotionDrift: boolean
  plannedEmotion: string
  actualEmotion: string
  duplicateEvents: string[]
  summary: string
  /** 下一章计划建议：**纯建议，不落盘**（审批制）。 */
  nextSuggestion: string
}

/** 结尾类型枚举（Go types.ChapterPlanEndingType 五值，逐字对齐）。 */
export const ENDING_TYPES = ['悬念', '冲突升级', '情节转折', '情感收尾', '自然过渡'] as const

// ── 绑定访问（本地 interface 收窄，不依赖 wails 生成物）───────────────────

interface PlanBridge {
  NovelChapterPlanGet(chapterNum: number): Promise<unknown>
  NovelChapterPlanSave(chapterNum: number, planJSON: string): Promise<unknown>
  NovelChapterPlanPropose(chapterNum: number, direction: string): Promise<unknown>
  NovelChapterPlanDeviation(chapterNum: number): Promise<unknown>
  NovelChapterGatePrecheck(chapterNum: number): Promise<unknown>
}

const PLAN_BINDINGS = [
  'NovelChapterPlanGet',
  'NovelChapterPlanSave',
  'NovelChapterPlanPropose',
  'NovelChapterPlanDeviation',
  'NovelChapterGatePrecheck',
] as const

/**
 * 取计划绑定面。任一绑定缺失（dev mock 未实现 / 收口前绑定面未生成）即抛**可读中文**
 * 错误——dev mock 诚实降级，绝不假装成功（§7.6-3）。
 */
function planBridge(): PlanBridge {
  const rec = app as unknown as Record<string, unknown>
  const missing = PLAN_BINDINGS.filter((name) => typeof rec[name] !== 'function')
  if (missing.length > 0) {
    throw new Error(`章节计划接口未就绪（缺少绑定：${missing.join('、')}）`)
  }
  return rec as unknown as PlanBridge
}

// ── 防御式收窄（后端结构体负载不可信；畸形一律降级为空/中文提示）──────────

function toText(value: unknown): string {
  return typeof value === 'string' ? value : ''
}

function toTextList(value: unknown): string[] {
  return Array.isArray(value) ? value.filter((x): x is string => typeof x === 'string') : []
}

function toNumValue(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

/** 收窄为 ChapterPlan；非对象（含 null = 未制定计划）返回 null。 */
function toChapterPlan(value: unknown): ChapterPlan | null {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) return null
  const rec = value as Record<string, unknown>
  const plan: ChapterPlan = {
    plot_summary: toText(rec.plot_summary),
    key_events: toTextList(rec.key_events),
    character_focus: toTextList(rec.character_focus),
    emotional_tone: toText(rec.emotional_tone),
    narrative_goal: toText(rec.narrative_goal),
    conflict_type: toText(rec.conflict_type),
    ending_type: toText(rec.ending_type),
  }
  const sub = toNumValue(rec.sub_index)
  if (sub !== undefined) plan.sub_index = sub
  const title = toText(rec.title)
  if (title) plan.title = title
  const words = toNumValue(rec.estimated_words)
  if (words !== undefined) plan.estimated_words = words
  return plan
}

function toPlanProblems(value: unknown): PlanProblem[] {
  if (!Array.isArray(value)) return []
  return value
    .filter((x): x is Record<string, unknown> => typeof x === 'object' && x !== null)
    .map((x) => {
      const item: PlanProblem = {
        code: toText(x.code),
        severity: toText(x.severity),
        message: toText(x.message),
      }
      const evidence = toText(x.evidence)
      if (evidence) item.evidence = evidence
      return item
    })
}

/** 收窄为 PlanGateReport；非对象返回 null（调用方按「预检不可用」诚实降级）。 */
function toGateReport(value: unknown, chapterNum: number): PlanGateReport | null {
  if (typeof value !== 'object' || value === null) return null
  const rec = value as Record<string, unknown>
  return {
    chapterNum: toNumValue(rec.chapterNum) ?? chapterNum,
    allowed: rec.allowed === true,
    hasPlan: rec.hasPlan === true,
    missing: toTextList(rec.missing),
    planProblems: toPlanProblems(rec.planProblems),
    outlineIssues: toPlanProblems(rec.outlineIssues),
    blocking: rec.blocking === true,
  }
}

/** 收窄为 PlanDeviation；非对象返回 null（缺失/未分析都是正常态）。 */
function toDeviation(value: unknown, chapterNum: number): PlanDeviation | null {
  if (typeof value !== 'object' || value === null) return null
  const rec = value as Record<string, unknown>
  return {
    chapterNum: toNumValue(rec.chapterNum) ?? chapterNum,
    hasPlan: rec.hasPlan === true,
    analyzed: rec.analyzed === true,
    missingEvents: toTextList(rec.missingEvents),
    endingMismatch: rec.endingMismatch === true,
    plannedEnding: toText(rec.plannedEnding),
    actualEnding: toText(rec.actualEnding),
    emotionDrift: rec.emotionDrift === true,
    plannedEmotion: toText(rec.plannedEmotion),
    actualEmotion: toText(rec.actualEmotion),
    duplicateEvents: toTextList(rec.duplicateEvents),
    summary: toText(rec.summary),
    nextSuggestion: toText(rec.nextSuggestion),
  }
}

function errText(err: unknown): string {
  return err instanceof Error ? err.message : String(err ?? '未知错误')
}

// ── 编辑草稿（文本域一行一条，保存时拆行；表单与后端行数组互转）──────────

interface PlanDraft {
  narrative_goal: string
  plot_summary: string
  key_events_text: string
  character_focus_text: string
  emotional_tone: string
  conflict_type: string
  ending_type: string
  estimated_words: number | null
}

const EMPTY_DRAFT: PlanDraft = {
  narrative_goal: '', plot_summary: '', key_events_text: '', character_focus_text: '',
  emotional_tone: '', conflict_type: '', ending_type: '', estimated_words: null,
}

function splitLines(text: string): string[] {
  return text.split('\n').map((s) => s.trim()).filter((s) => s !== '')
}

function draftOf(plan: ChapterPlan | null): PlanDraft {
  if (plan === null) return EMPTY_DRAFT
  return {
    narrative_goal: plan.narrative_goal,
    plot_summary: plan.plot_summary,
    key_events_text: plan.key_events.join('\n'),
    character_focus_text: plan.character_focus.join('\n'),
    emotional_tone: plan.emotional_tone,
    conflict_type: plan.conflict_type,
    ending_type: plan.ending_type,
    estimated_words: plan.estimated_words ?? null,
  }
}

// ── 小组件（模块级，非导出）──────────────────────────────────────────────

const softLabel: React.CSSProperties = { fontSize: 12, color: 'var(--color-text-secondary)', marginBottom: 2 }

const Field: React.FC<{ label: string; children: React.ReactNode }> = ({ label, children }) => (
  <div>
    <div style={softLabel}>{label}</div>
    {children}
  </div>
)

const Row: React.FC<{ label: string; children: React.ReactNode }> = ({ label, children }) => (
  <div style={{ display: 'flex', gap: 6, alignItems: 'baseline' }}>
    <span style={{ ...softLabel, marginBottom: 0, flexShrink: 0 }}>{label}</span>
    <span style={{ minWidth: 0 }}>{children}</span>
  </div>
)

const Dash: React.FC = () => <span style={{ color: 'var(--color-text-secondary)' }}>—</span>

export interface ChapterPlanCardProps {
  /** 当前章号；null/≤0 = 未选章（空态「先在左侧选择章节」）。 */
  chapterNum: number | null
  /** 硬闸横幅「先补章节计划」动作：由父级决定如何把作者引到补计划处。 */
  onNeedPlan?: () => void
  /** 计划落盘成功回调（父级刷新计划卡/大纲状态）。 */
  onPlanSaved?: () => void
  /** 生成中禁用计划动作。 */
  disabled?: boolean
  /**
   * 作者创作方向（v4.450.0 三点打通）：来自剧情分支意向或剧情要求，随 Propose
   * 下发；非空时卡片顶部展示来源横幅，作者知道草案将服从这条方向。空/未传=无方向。
   */
  direction?: string
  /**
   * 「生成本章」出口（v4.450.0 三点打通）：计划落盘后一键回到生成——父级以计划
   * 情节摘要为剧情要求发起生成（后端生成时会按章号注入本计划）。不传=不出口。
   */
  onGenerate?: (chapterNum: number, plan: ChapterPlan) => void
}

/**
 * 章节计划卡（受控、自包含）：显示/编辑/生成草案/审批落盘 + 写后偏差报告。
 * 硬闸横幅只在预检不允许生成（或未制定计划）时出现——硬闸只拦「没有抓手」，
 * 不评判计划质量。
 */
const ChapterPlanCard: React.FC<ChapterPlanCardProps> = ({
  chapterNum, onNeedPlan, onPlanSaved, disabled = false, direction = '', onGenerate,
}) => {
  // 切章/刷新共用的序号守卫：seq 不等的迟到响应一律丢弃（见文件头纪律 1）。
  const seqRef = useRef(0)
  /** 最新章号（异步回调判定「是否已切章」，避免旧章结果写进新章卡片）。 */
  const numRef = useRef<number | null>(null)
  const [loading, setLoading] = useState(false)
  const [plan, setPlan] = useState<ChapterPlan | null>(null)
  const [gate, setGate] = useState<PlanGateReport | null>(null)
  const [deviation, setDeviation] = useState<PlanDeviation | null>(null)
  const [loadError, setLoadError] = useState('')
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState<PlanDraft>(EMPTY_DRAFT)
  const [proposing, setProposing] = useState(false)
  const [saving, setSaving] = useState(false)
  const [proposeError, setProposeError] = useState('')
  const [saveError, setSaveError] = useState('')
  const [notice, setNotice] = useState('')

  const num = chapterNum !== null && chapterNum > 0 ? chapterNum : null
  numRef.current = num

  /** 拉计划 + 预检（同一 seq）；偏差报告随后非阻断拉取。 */
  const load = useCallback(async (target: number) => {
    const seq = ++seqRef.current
    setLoading(true); setLoadError(''); setNotice('')
    setProposeError(''); setSaveError('')
    setPlan(null); setGate(null); setDeviation(null); setEditing(false)
    try {
      const bridge = planBridge()
      const [rawPlan, rawGate] = await Promise.all([
        bridge.NovelChapterPlanGet(target),
        bridge.NovelChapterGatePrecheck(target),
      ])
      if (seq !== seqRef.current) return // 迟到响应：已被更新的切章/刷新取代
      const next = toChapterPlan(rawPlan)
      setPlan(next)
      setDraft(draftOf(next))
      setGate(toGateReport(rawGate, target))
    } catch (err: unknown) {
      if (seq !== seqRef.current) return
      setLoadError(errText(err))
    } finally {
      if (seq === seqRef.current) setLoading(false)
    }
    // 偏差报告是写后回写产物：缺失/未分析都是正常态，失败静默降级（不打扰作者）。
    try {
      const raw = await planBridge().NovelChapterPlanDeviation(target)
      if (seq !== seqRef.current) return
      setDeviation(toDeviation(raw, target))
    } catch {
      if (seq === seqRef.current) setDeviation(null)
    }
  }, [])

  useEffect(() => {
    if (num === null) {
      seqRef.current += 1 // 失效在途响应：旧章结果不得落到「未选章」空态
      setLoading(false); setPlan(null); setGate(null); setDeviation(null)
      setEditing(false); setLoadError(''); setProposeError(''); setSaveError(''); setNotice('')
      return
    }
    setEditing(false); setProposeError(''); setSaveError('')
    void load(num)
  }, [num, load])

  /** 保存成功后复拉预检：硬闸横幅随审批结果实时消失（不做乐观假设）。 */
  const refreshGate = useCallback(async (target: number) => {
    if (numRef.current !== target) return // 已切章：旧章预检不得污染新章卡片
    const seq = ++seqRef.current
    try {
      const raw = await planBridge().NovelChapterGatePrecheck(target)
      if (seq !== seqRef.current || numRef.current !== target) return
      setGate(toGateReport(raw, target))
    } catch {
      // 预检复拉失败不打扰：卡片主体已按保存结果更新，下次打开重拉
    }
  }, [])

  /** 生成 AI 草案：成功进入**编辑态**（不落盘——作者是上帝）。创作方向随请求下发。 */
  const proposePlan = async () => {
    const target = num
    if (target === null) return
    setProposing(true); setProposeError(''); setSaveError(''); setNotice('')
    try {
      const raw = await planBridge().NovelChapterPlanPropose(target, direction.trim())
      const next = toChapterPlan(raw)
      if (next === null) throw new Error('模型未返回可用的计划草案，可改用手写计划')
      if (numRef.current !== target) return
      setPlan(next)
      setDraft(draftOf(next))
      setEditing(true)
      setNotice('AI 已生成计划草案（未落盘）——审阅修改后点「保存计划」才写入 plans.json')
    } catch (err: unknown) {
      if (numRef.current === target) setProposeError(errText(err))
    } finally {
      setProposing(false)
    }
  }

  /** 组装落盘载荷：json 键逐字对齐 Go（plot_summary/key_events/…），保留未编辑字段。 */
  const buildPayload = (): ChapterPlan => {
    const payload: ChapterPlan = {
      plot_summary: draft.plot_summary.trim(),
      key_events: splitLines(draft.key_events_text),
      character_focus: splitLines(draft.character_focus_text),
      emotional_tone: draft.emotional_tone.trim(),
      narrative_goal: draft.narrative_goal.trim(),
      conflict_type: draft.conflict_type.trim(),
      ending_type: draft.ending_type,
    }
    if (draft.estimated_words !== null) payload.estimated_words = draft.estimated_words
    if (plan?.sub_index !== undefined) payload.sub_index = plan.sub_index
    if (plan?.title) payload.title = plan.title
    return payload
  }

  /** 审批落盘：失败如实显示后端错误（含被点名的重复事件）且不丢表单。 */
  const savePlan = async () => {
    const target = num
    if (target === null) return
    setSaving(true); setSaveError(''); setNotice('')
    try {
      const payload = buildPayload()
      await planBridge().NovelChapterPlanSave(target, JSON.stringify(payload))
      if (numRef.current !== target) return // 已切章：本次保存结果不回写新章卡片
      setPlan(payload)
      setDraft(draftOf(payload))
      setEditing(false)
      setNotice('计划已保存')
      message.success(`第 ${target} 章计划已保存`)
      onPlanSaved?.()
      void refreshGate(target)
    } catch (err: unknown) {
      if (numRef.current !== target) return
      setSaveError(errText(err))
    } finally {
      setSaving(false)
    }
  }

  const startEdit = () => {
    setDraft(draftOf(plan)); setSaveError(''); setProposeError(''); setNotice(''); setEditing(true)
  }

  const startManual = () => {
    setDraft(draftOf(plan)); setSaveError(''); setProposeError('')
    setNotice('手写模式：填写后点「保存计划」落盘（后端会做齐备性与跨章去重校验）')
    setEditing(true)
  }

  const cancelEdit = () => { setEditing(false); setSaveError(''); setNotice('') }

  const setField = <K extends keyof PlanDraft>(key: K, value: PlanDraft[K]) => {
    setDraft((prev) => ({ ...prev, [key]: value }))
  }

  // ── 渲染 ────────────────────────────────────────────────────────────
  if (num === null) {
    return (
      <div data-testid="chapter-plan-card" style={{ padding: 4 }}>
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="先在左侧选择章节" />
      </div>
    )
  }

  const hasPlan = gate !== null ? gate.hasPlan : plan !== null
  const blocking = gate?.blocking === true
  const missing = gate?.missing ?? []
  const problems = [...(gate?.planProblems ?? []), ...(gate?.outlineIssues ?? [])]
  const showBanner = !hasPlan || (gate !== null && !gate.allowed)
  const showDeviation = deviation !== null && deviation.analyzed && deviation.hasPlan

  return (
    <div
      data-testid="chapter-plan-card"
      style={{
        border: '1px solid var(--border-subtle, var(--color-border))',
        borderRadius: 'var(--radius-md, 6px)',
        background: 'var(--bg-elevated, transparent)',
        padding: 12, fontSize: 13,
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8 }}>
        <b>章节计划</b>
        <span style={{ fontSize: 12, color: 'var(--color-text-secondary)' }}>第 {num} 章</span>
        {gate !== null && (hasPlan
          ? <Tag color="green">已制定</Tag>
          : <Tag color={blocking ? 'red' : 'orange'}>未制定</Tag>)}
        <div style={{ flex: 1 }} />
        {!loading && !editing && hasPlan && plan !== null && onGenerate && (
          <Button
            size="small" type="primary" disabled={disabled}
            data-testid="plan-generate-chapter"
            onClick={() => onGenerate(num, plan)}
          >
            生成本章
          </Button>
        )}
        {!loading && !editing && hasPlan && (
          <>
            <Button size="small" disabled={disabled} onClick={startEdit}>编辑计划</Button>
            <Button size="small" disabled={disabled} loading={proposing} onClick={() => void proposePlan()}>
              重新生成草案
            </Button>
          </>
        )}
      </div>

      {direction.trim() !== '' && (
        <Alert
          data-testid="plan-direction"
          type="info" showIcon style={{ marginBottom: 8 }}
          message="本章创作方向（来自分支/剧情要求）"
          description={<div style={{ fontSize: 12 }}>{direction.trim()}</div>}
        />
      )}

      {loading ? (
        <div style={{ display: 'flex', gap: 8, alignItems: 'center', color: 'var(--color-text-secondary)' }}>
          <Spin size="small" />
          <span>正在读取第 {num} 章计划…</span>
        </div>
      ) : (
        <>
          {loadError !== '' && (
            <Alert
              data-testid="plan-load-error"
              type="warning" showIcon style={{ marginBottom: 8 }}
              message="章节计划读取失败" description={loadError}
            />
          )}

          {/* 硬闸横幅：未制定计划（blocking 红）/ 计划未过预检 */}
          {showBanner && (
            <Alert
              data-testid="plan-gate-banner"
              type={blocking ? 'error' : 'warning'}
              showIcon
              style={{ marginBottom: 8 }}
              message={hasPlan ? '章节计划未过预检（会阻断生成）' : '本章尚未制定计划（硬闸）'}
              description={(
                <div style={{ fontSize: 12 }}>
                  {missing.length > 0 && <div>缺失：{missing.join('、')}</div>}
                  {problems.length > 0 && (
                    <ul style={{ margin: '4px 0 0 16px', padding: 0 }}>
                      {problems.map((p, i) => (
                        <li key={`${p.code}-${i}`}>
                          [{p.severity}] {p.message}{p.evidence ? `（${p.evidence}）` : ''}
                        </li>
                      ))}
                    </ul>
                  )}
                  <div style={{ marginTop: 4 }}>
                    {onGenerate
                      ? '补完并保存计划后，点右上角「生成本章」开始写正文（也可在弹窗里显式跳过硬闸）。'
                      : '先补章节计划，再回生成入口（也可在弹窗里显式跳过硬闸）。'}
                  </div>
                </div>
              )}
              action={onNeedPlan
                ? <Button size="small" danger={blocking} onClick={onNeedPlan}>先补章节计划</Button>
                : undefined}
            />
          )}

          {notice !== '' && <Alert type="info" showIcon style={{ marginBottom: 8 }} message={notice} />}
          {proposeError !== '' && (
            <Alert
              data-testid="plan-propose-error"
              type="error" showIcon style={{ marginBottom: 8 }}
              message="生成计划草案失败" description={proposeError}
            />
          )}
          {saveError !== '' && (
            <Alert
              data-testid="plan-save-error"
              type="error" showIcon style={{ marginBottom: 8 }}
              message="保存计划被拒绝（后端校验未通过）" description={saveError}
            />
          )}

          {/* 编辑 / 手写表单 */}
          {editing ? (
            <div style={{ display: 'grid', gap: 8 }}>
              <Field label="叙事目标（这一章为什么存在）">
                <Input
                  data-testid="plan-goal-input"
                  size="small" value={draft.narrative_goal} placeholder="一句话说明本章意图"
                  onChange={(e) => setField('narrative_goal', e.target.value)}
                />
              </Field>
              <Field label="情节摘要">
                <Input.TextArea
                  data-testid="plan-summary-input"
                  rows={3} value={draft.plot_summary} placeholder="200-300 字本章剧情走向"
                  onChange={(e) => setField('plot_summary', e.target.value)}
                />
              </Field>
              <Field label="关键事件（每行一条，至少 2 条；跨章不得重复）">
                <Input.TextArea
                  data-testid="plan-key-events-input"
                  rows={3} value={draft.key_events_text} placeholder={'第一件事\n第二件事'}
                  onChange={(e) => setField('key_events_text', e.target.value)}
                />
              </Field>
              <Field label="角色焦点（每行一个，至少 1 个）">
                <Input.TextArea
                  data-testid="plan-focus-input"
                  rows={2} value={draft.character_focus_text} placeholder={'主角名\n关键配角名'}
                  onChange={(e) => setField('character_focus_text', e.target.value)}
                />
              </Field>
              <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap' }}>
                <Field label="情绪基调">
                  <Input
                    data-testid="plan-emotion-input"
                    size="small" style={{ width: 160 }} value={draft.emotional_tone}
                    onChange={(e) => setField('emotional_tone', e.target.value)}
                  />
                </Field>
                <Field label="冲突类型">
                  <Input
                    data-testid="plan-conflict-input"
                    size="small" style={{ width: 160 }} value={draft.conflict_type}
                    onChange={(e) => setField('conflict_type', e.target.value)}
                  />
                </Field>
                <Field label="结局类型">
                  <Select
                    data-testid="plan-ending-select"
                    size="small" style={{ width: 160 }} placeholder="选择结尾类型"
                    value={draft.ending_type === '' ? undefined : draft.ending_type}
                    options={ENDING_TYPES.map((v) => ({ value: v, label: v }))}
                    onChange={(v: string | undefined) => setField('ending_type', v ?? '')}
                  />
                </Field>
                <Field label="预估字数（可选）">
                  <InputNumber
                    data-testid="plan-words-input"
                    size="small" style={{ width: 140 }} min={0} max={50000} step={500}
                    value={draft.estimated_words}
                    onChange={(v) => setField('estimated_words', typeof v === 'number' ? v : null)}
                  />
                </Field>
              </div>
              <Space size={8}>
                <Button type="primary" size="small" loading={saving} disabled={disabled} onClick={() => void savePlan()}>
                  保存计划
                </Button>
                <Button size="small" disabled={saving} onClick={cancelEdit}>取消编辑</Button>
              </Space>
            </div>
          ) : hasPlan && plan !== null ? (
            /* 已制定计划：七字段 */
            <div style={{ display: 'grid', gap: 6 }}>
              <Row label="叙事目标">{plan.narrative_goal || <Dash />}</Row>
              <Row label="情节摘要">{plan.plot_summary || <Dash />}</Row>
              <Row label="关键事件">
                {plan.key_events.length > 0 ? (
                  <ol data-testid="plan-key-events" style={{ margin: 0, paddingLeft: 18 }}>
                    {plan.key_events.map((e, i) => <li key={`${e}-${i}`}>{e}</li>)}
                  </ol>
                ) : <Dash />}
              </Row>
              <Row label="角色焦点">
                {plan.character_focus.length > 0
                  ? plan.character_focus.map((c) => <Tag key={c} style={{ marginInlineEnd: 4 }}>{c}</Tag>)
                  : <Dash />}
              </Row>
              <Row label="情绪基调">{plan.emotional_tone ? <Tag color="blue">{plan.emotional_tone}</Tag> : <Dash />}</Row>
              <Row label="冲突类型">{plan.conflict_type ? <Tag color="purple">{plan.conflict_type}</Tag> : <Dash />}</Row>
              <Row label="结局类型">{plan.ending_type ? <Tag color="gold">{plan.ending_type}</Tag> : <Dash />}</Row>
              {plan.estimated_words !== undefined && (
                <Row label="预估字数">{plan.estimated_words.toLocaleString()} 字</Row>
              )}
            </div>
          ) : (
            /* 未制定计划：草案入口 + 手写入口 */
            <div style={{ display: 'grid', gap: 8 }}>
              <div style={{ fontSize: 12, color: 'var(--color-text-secondary)' }}>
                AI 只产出草案，落盘一律由你点「保存计划」审批（作者是上帝）。
              </div>
              <Space size={8} wrap>
                <Button type="primary" size="small" disabled={disabled} loading={proposing} onClick={() => void proposePlan()}>
                  生成计划草案
                </Button>
                <Button size="small" disabled={disabled} onClick={startManual}>手写计划</Button>
              </Space>
            </div>
          )}

          {/* 写后偏差报告：有分析结果才显示；建议只展示，不提供一键落盘 */}
          {showDeviation && deviation !== null && (
            <div
              data-testid="plan-deviation"
              style={{ marginTop: 10, paddingTop: 8, borderTop: '1px dashed var(--border-subtle, var(--color-border))' }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                <b>计划偏差（写后回写）</b>
                <Tag color="cyan">第 {deviation.chapterNum} 章</Tag>
              </div>
              {deviation.summary !== '' && <div style={{ marginTop: 4 }}>{deviation.summary}</div>}
              {deviation.missingEvents.length > 0 && (
                <div style={{ marginTop: 4 }}>
                  <div style={softLabel}>未达成的关键事件</div>
                  <ul data-testid="plan-deviation-missing" style={{ margin: 0, paddingLeft: 18 }}>
                    {deviation.missingEvents.map((e, i) => <li key={`${e}-${i}`}>{e}</li>)}
                  </ul>
                </div>
              )}
              <Space size={6} wrap style={{ marginTop: 4 }}>
                {deviation.endingMismatch && (
                  <Tag color="red">结尾类型不符：计划「{deviation.plannedEnding || '—'}」→ 实际「{deviation.actualEnding || '—'}」</Tag>
                )}
                {deviation.emotionDrift && (
                  <Tag color="orange">情绪走向漂移：计划「{deviation.plannedEmotion || '—'}」→ 实际「{deviation.actualEmotion || '—'}」</Tag>
                )}
                {deviation.duplicateEvents.length > 0 && (
                  <Tag color="volcano">与其它章重复 {deviation.duplicateEvents.length} 条：{deviation.duplicateEvents.join('、')}</Tag>
                )}
              </Space>
              {deviation.nextSuggestion !== '' && (
                <div data-testid="plan-next-suggestion" style={{ marginTop: 6 }}>
                  <div style={softLabel}>下一章建议（仅建议，不落盘）</div>
                  <div>{deviation.nextSuggestion}</div>
                </div>
              )}
            </div>
          )}
        </>
      )}
    </div>
  )
}

export default ChapterPlanCard
