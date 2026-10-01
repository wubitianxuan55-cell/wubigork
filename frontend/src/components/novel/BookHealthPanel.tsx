// BookHealthPanel.tsx — 全书体检面板（GenerationGate 闭环收口，规格
// 进度计划/gaea-gen-gate-closure-20260916.md）：触发式全量编译，纯确定性
// 零 LLM，作者点按跑完出报告（无常驻无定时）。消费 RunBookHealthCheck：
// 头部聚合卡（总章/契约问题章/质量问题章/最差 AI 味/V2 覆盖/伏笔 findings）
// + 逐章表（章号/字数/契约/质量/AI 味，越线红标）+ 伏笔 findings 列表。
// v4.340：情感曲线区（t7 观察池）——按需逐章拉分析 V2 的情感弧线强度，
// 纯 SVG 折线（零新依赖）；未分析章诚实跳过并计数。
import { softTextStyle } from '../../utils/uiStyles'
import React, { useCallback, useEffect, useRef, useState } from 'react'
import { Alert, Button, Empty, Modal, Spin, Table, Tag, Typography, message } from 'antd'
import { app } from '../../gaea/lib/bridge'
import { useAppStore } from '../../stores/appStore'
import type { BookHealthReportView, ChapterAnalysisV2View } from '../../gaea/lib/bridge/novel'


/** 逐章体检行（JSX 泛型不认索引访问，抽别名）。 */
type HealthRow = BookHealthReportView['chapters'][number]

/** 情感曲线数据点（章号 + 强度 0-10 + 主导情绪）。 */
interface CurvePoint { num: number; intensity: number; emotion: string }

/** 评测快照视图（v4.446；Go 侧 omitempty 字段可能缺省，消费方 ?. 与 ?? 防御）。 */
interface EvalSnapView {
  chapters?: number; chars?: number
  taste?: { mean?: number; max?: number; p90?: number; worstChapter?: number }
  quality?: { s1?: number; s2?: number; s3?: number }
  foreshadow?: { items?: number; recall?: number; findings?: number }
  tension?: { mean?: number; p90?: number; swing?: number; covered?: number }
  contextRunes?: number
}
/** 对比行（dir: better|worse|flat）。 */
interface EvalCmpItem { metric?: string; from?: number; to?: number; delta?: number; dir?: string }
interface EvalCmpView { stale?: boolean; items?: EvalCmpItem[] }

/** 情感曲线 SVG：x=章号线性、y=强度 0-10；相邻点直连（未分析章的间隙以长段如实呈现）。 */
function EmotionCurveSvg({ pts }: { pts: CurvePoint[] }) {
  const W = 640, H = 150, PAD_L = 30, PAD_R = 16, PAD_T = 12, PAD_B = 24
  const nums = pts.map(p => p.num)
  const minN = Math.min(...nums)
  const maxN = Math.max(...nums)
  const x = (n: number) =>
    maxN === minN ? PAD_L + (W - PAD_L - PAD_R) / 2 : PAD_L + ((n - minN) / (maxN - minN)) * (W - PAD_L - PAD_R)
  const y = (v: number) => PAD_T + (1 - v / 10) * (H - PAD_T - PAD_B)
  const linePts = pts.map(p => `${x(p.num).toFixed(1)},${y(p.intensity).toFixed(1)}`).join(' ')
  return (
    <svg data-testid="health-curve-svg" viewBox={`0 0 ${W} ${H}`} width="100%" height={H} role="img"
      aria-label={`情感曲线：${pts.length} 章强度折线`}>
      {[0, 5, 10].map(v => (
        <g key={v}>
          <line x1={PAD_L} x2={W - PAD_R} y1={y(v)} y2={y(v)} stroke="currentColor" strokeOpacity={0.15} strokeDasharray={v === 5 ? '4 4' : undefined} />
          <text x={PAD_L - 6} y={y(v) + 3} textAnchor="end" fontSize={9} fill="currentColor" fillOpacity={0.5}>{v}</text>
        </g>
      ))}
      <polyline points={linePts} fill="none" stroke="var(--color-primary, var(--md-sys-color-primary, #2563eb))" strokeWidth={1.8} strokeLinejoin="round" />
      {pts.map(p => (
        <circle key={p.num} cx={x(p.num)} cy={y(p.intensity)} r={3} data-testid="health-curve-point"
          fill="var(--color-primary, var(--md-sys-color-primary, #2563eb))">
          <title>{`第 ${p.num} 章 · ${p.emotion || '—'}（强度 ${p.intensity}）`}</title>
        </circle>
      ))}
      <text x={PAD_L} y={H - 6} fontSize={10} fill="currentColor" fillOpacity={0.6}>第 {minN} 章</text>
      <text x={W - PAD_R} y={H - 6} textAnchor="end" fontSize={10} fill="currentColor" fillOpacity={0.6}>第 {maxN} 章</text>
    </svg>
  )
}

export default function BookHealthPanel({ open, onClose }: {
  open: boolean
  onClose: () => void
}) {
  const [loading, setLoading] = useState(false)
  const [report, setReport] = useState<BookHealthReportView | null>(null)
  // v4.425：失败与「确实没有结果」必须分开——此前 catch 里 setReport(null)，弹窗只剩
  // Empty「没有体检结果」，把一次失败说成「无数据」且没有任何重试入口（B10）。
  const [error, setError] = useState('')
  const [curveLoading, setCurveLoading] = useState(false)
  const [curve, setCurve] = useState<CurvePoint[] | null>(null)
  const [curveSkipped, setCurveSkipped] = useState(0)
  // v4.446 评测基线区：快照/设基线/对比，全部按需点按（零 LLM 纯确定性聚合）。
  const [evalSnap, setEvalSnap] = useState<EvalSnapView | null>(null)
  const [evalCmp, setEvalCmp] = useState<EvalCmpView | null>(null)
  const [evalBusy, setEvalBusy] = useState<'' | 'snap' | 'base' | 'cmp'>('')
  // 情感曲线在途代际（观察池#3）：重跑体检/切书自增作废旧循环——逐章 await
  // 无守卫时，旧循环完成后会把上一本书/上一轮的曲线 setCurve 进新上下文。
  const curveSeqRef = useRef(0)
  // 弹窗由 CreatePage 控制开关、面板常驻挂载：报告属该书的数据，切书必须失效，
  // 否则整屏显示的是上一本的体检结论（弹窗若开着更明显）。
  const projectPath = useAppStore((s) => s.projectPath)

  const run = useCallback(async () => {
    setLoading(true)
    setError('')
    curveSeqRef.current++ // 作废在途曲线循环（章数可能变化，旧循环结果不得再落）
    setCurveLoading(false) // 被作废的循环不再收尾，这里代收（否则按钮永远转圈）
    setCurve(null) // 重跑体检后曲线失效（章数可能变化），需重新生成
    setCurveSkipped(0)
    try {
      setReport(await app.RunBookHealthCheck())
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      message.error(`全书体检失败：${msg}`)
      setReport(null)
      setError(msg)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (open) void run()
  }, [open, run])

  // 切书即清报告态（含曲线与失败原因）：不额外发请求，重新打开弹窗时 effect 自然重拉。
  useEffect(() => {
    curveSeqRef.current++ // 作废在途曲线循环：切书后旧书的点不得写进新上下文
    setReport(null)
    setError('')
    setCurve(null)
    setCurveSkipped(0)
    setCurveLoading(false) // 被作废的循环不再收尾，这里代收
    setLoading(false)
    setEvalSnap(null) // 评测基线区同属该书数据，切书一并失效（v4.446）
    setEvalCmp(null)
  }, [projectPath])

  // 情感曲线：逐章读分析 V2（缺档/无情感弧线跳过并计数；后端缺档只报
  // 「尚未分析」，不触发重建，循环安全）。体检触发式口径一致：按钮按需生成。
  const buildCurve = useCallback(async () => {
    if (!report) return
    const seq = ++curveSeqRef.current
    setCurveLoading(true)
    try {
      const pts: CurvePoint[] = []
      let skipped = 0
      for (const row of report.chapters ?? []) {
        try {
          const v2 = (await app.NovelChapterAnalysisV2(row.chapterNum)) as ChapterAnalysisV2View | null
          if (curveSeqRef.current !== seq) return // 已被重跑/切书作废：丢弃迟到结果
          const it = v2?.result?.emotional_arc?.intensity
          if (typeof it === 'number' && Number.isFinite(it)) {
            pts.push({
              num: row.chapterNum,
              intensity: Math.max(0, Math.min(10, it)),
              emotion: String(v2?.result?.emotional_arc?.primary_emotion ?? ''),
            })
          } else {
            skipped++
          }
        } catch {
          skipped++
        }
      }
      if (curveSeqRef.current !== seq) return // 迟到的整轮结果不落（同上）
      setCurve(pts)
      setCurveSkipped(skipped)
    } finally {
      // 只有最新代际才收 loading：被作废的旧循环若在这里 setCurveLoading(false)，
      // 会把新一轮（已在跑）的转圈提前关掉。
      if (curveSeqRef.current === seq) setCurveLoading(false)
    }
  }, [report])

  const fs = report?.foreshadow
  const findings = fs?.findings ?? []

  // v4.446 评测基线区动作：快照 / 设基线 / 对比（零 LLM，点按触发）。
  const runEvalSnap = useCallback(async () => {
    setEvalBusy('snap')
    try {
      setEvalSnap((await app.NovelEvalSnapshot(false)) as EvalSnapView)
      setEvalCmp(null)
    } catch (e) {
      message.error(`快照失败：${e instanceof Error ? e.message : String(e)}`)
    } finally {
      setEvalBusy('')
    }
  }, [])
  const runEvalBase = useCallback(async () => {
    setEvalBusy('base')
    try {
      await app.NovelEvalBaselineSet()
      message.success('已设为基线（eval/baseline.json）')
      setEvalSnap((await app.NovelEvalSnapshot(false)) as EvalSnapView)
      setEvalCmp(null)
    } catch (e) {
      message.error(`设基线失败：${e instanceof Error ? e.message : String(e)}`)
    } finally {
      setEvalBusy('')
    }
  }, [])
  const runEvalCmp = useCallback(async () => {
    setEvalBusy('cmp')
    try {
      setEvalCmp((await app.NovelEvalCompare()) as EvalCmpView)
    } catch (e) {
      message.error(`对比失败：${e instanceof Error ? e.message : String(e)}`)
    } finally {
      setEvalBusy('')
    }
  }, [])

  return (
    <Modal open={open} title="全书体检（确定性 · 零模型调用）" onCancel={onClose} width={780} destroyOnHidden
      footer={[
        <Button key="rerun" size="small" loading={loading} onClick={() => void run()}>重新体检</Button>,
        <Button key="close" size="small" onClick={onClose}>关闭</Button>,
      ]}>
      {loading ? (
        <div style={{ padding: '48px 0', textAlign: 'center' }}><Spin /><div style={{ ...softTextStyle, marginTop: 8 }}>全量编译中（逐章写前契约 / 写后质量 / AI 味 / 伏笔登记）…</div></div>
      ) : error ? (
        <Alert
          type="error" showIcon data-testid="book-health-error"
          message="全书体检失败"
          description={error}
          action={<Button size="small" danger onClick={() => void run()}>重试</Button>}
          style={{ margin: '24px 0' }}
        />
      ) : !report ? (
        <Empty description="没有体检结果" style={{ padding: '32px 0' }} />
      ) : (
        <div data-testid="book-health-body" style={{ maxHeight: '66vh', overflowY: 'auto' }}>
          {/* 聚合卡 */}
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 12 }}>
            <Tag style={{ marginRight: 0 }}>共 {report.totalChapters} 章</Tag>
            <Tag color={report.contractIssueChapters ? 'orange' : undefined} style={{ marginRight: 0 }}>契约问题章 {report.contractIssueChapters}</Tag>
            <Tag color={report.qualityIssueChapters ? 'orange' : undefined} style={{ marginRight: 0 }}>质量问题章 {report.qualityIssueChapters}</Tag>
            {report.worstAiTaste && (
              <Tag color="red" style={{ marginRight: 0 }}>最差 AI 味 {report.worstAiTaste.aiTasteScore}（第 {report.worstAiTaste.chapterNum} 章）</Tag>
            )}
            <Tag style={{ marginRight: 0 }}>分析覆盖 {report.analyzedChapters}/{report.totalChapters}</Tag>
            {fs && <Tag style={{ marginRight: 0 }}>伏笔 {fs.items} 条{findings.length ? ` · ${findings.length} 项告警` : ''}</Tag>}
          </div>
          {report.worstAiTaste && report.worstAiTaste.aiTasteScore >= 60 && (
            <Alert type="warning" showIcon style={{ marginBottom: 12 }}
              message={`第 ${report.worstAiTaste.chapterNum} 章 AI 味 ${report.worstAiTaste.aiTasteScore} 分`}
              description="建议用「一键去味 / 高级去味」处理后复检（60 分以上视为明显 AI 味）。" />
          )}
          {/* 逐章表 */}
          <Table<HealthRow> size="small" rowKey="chapterNum"
            dataSource={report.chapters ?? []} pagination={false}
            locale={{ emptyText: '还没有已写章节' }}
            columns={[
              { title: '章', dataIndex: 'chapterNum', width: 56 },
              { title: '字数', dataIndex: 'words', width: 80, render: (v: number) => <span style={{ fontVariantNumeric: 'tabular-nums' }}>{v}</span> },
              {
                title: '写前契约', width: 100, render: (_v, row) =>
                  row.outlineIssues > 0
                    ? <span style={{ color: row.outlineErrors > 0 ? 'var(--color-error, var(--md-sys-color-error, #cf1322))' : undefined }}>{row.outlineIssues} 项{row.outlineErrors > 0 ? `（${row.outlineErrors} 阻断级）` : ''}</span>
                    : <span style={softTextStyle}>—</span>,
              },
              {
                title: '写后质量', width: 100, render: (_v, row) =>
                  row.qualityIssues > 0
                    ? <span style={{ color: row.qualityErrors > 0 ? 'var(--color-error, var(--md-sys-color-error, #cf1322))' : undefined }}>{row.qualityIssues} 项{row.qualityErrors > 0 ? `（${row.qualityErrors} 阻断级）` : ''}</span>
                    : <span style={softTextStyle}>—</span>,
              },
              {
                title: 'AI 味', dataIndex: 'aiTasteScore', width: 80, render: (v: number) =>
                  v < 0 ? <span style={softTextStyle}>—</span> : (
                    <span style={{ color: v >= 60 ? 'var(--color-error, var(--md-sys-color-error, #cf1322))' : undefined, fontVariantNumeric: 'tabular-nums' }}>{v}</span>
                  ),
              },
            ]} />
          {/* 情感曲线（t7 观察池）：按需逐章拉分析 V2，SVG 折线 */}
          <div style={{ marginTop: 14 }} data-testid="health-curve-section">
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
              <span style={{ fontSize: 13, fontWeight: 600 }}>情感曲线</span>
              {!curveLoading && (
                <Button size="small" data-testid="health-curve-run" onClick={() => void buildCurve()}>
                  {curve === null ? '生成曲线' : '刷新'}
                </Button>
              )}
              {curveLoading && <Spin size="small" />}
              <span style={softTextStyle}>数据源：逐章分析 V2 的情感弧线强度（0-10）</span>
            </div>
            {curveLoading ? (
              <div style={softTextStyle}>逐章读取分析 V2（跳过未分析章）…</div>
            ) : curve !== null ? (
              curve.length >= 2 ? (
                <>
                  <EmotionCurveSvg pts={curve} />
                  {curveSkipped > 0 && (
                    <div style={softTextStyle}>{curveSkipped} 章未分析或无情感弧线，已跳过。</div>
                  )}
                </>
              ) : (
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  可绘制的章不足（需 ≥2 章有情感弧线数据；本次拿到 {curve.length} 章{curveSkipped ? `，${curveSkipped} 章跳过` : ''}）。
                </Typography.Text>
              )
            ) : (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                点击「生成曲线」逐章读取情感弧线（仅读盘，不触发分析）。
              </Typography.Text>
            )}
          </div>
          {/* 评测基线区（v4.446）：确定性指标快照/基线/对比，零 LLM 点按触发 */}
          <div style={{ marginTop: 14 }} data-testid="health-eval-section">
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
              <span style={{ fontSize: 13, fontWeight: 600 }}>评测基线（确定性指标）</span>
              <Button size="small" data-testid="health-eval-snap" loading={evalBusy === 'snap'} onClick={() => void runEvalSnap()}>
                {evalSnap ? '刷新快照' : '生成快照'}
              </Button>
              <Button size="small" loading={evalBusy === 'base'} disabled={evalBusy !== ''} onClick={() => void runEvalBase()}>
                设为基线
              </Button>
              <Button size="small" data-testid="health-eval-cmp" loading={evalBusy === 'cmp'} disabled={evalBusy !== ''} onClick={() => void runEvalCmp()}>
                与基线对比
              </Button>
            </div>
            {evalSnap && (
              <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginBottom: 6 }}>
                <Tag style={{ marginRight: 0 }}>章节 {evalSnap.chapters ?? 0}</Tag>
                <Tag style={{ marginRight: 0 }}>字数 {(evalSnap.chars ?? 0).toLocaleString()}</Tag>
                <Tag style={{ marginRight: 0 }}>AI 味均值 {evalSnap.taste?.mean ?? 0}（P90 {evalSnap.taste?.p90 ?? 0}）</Tag>
                <Tag style={{ marginRight: 0 }}>S1 {evalSnap.quality?.s1 ?? 0} · S2 {evalSnap.quality?.s2 ?? 0} · S3 {evalSnap.quality?.s3 ?? 0}</Tag>
                <Tag style={{ marginRight: 0 }}>伏笔回收率 {evalSnap.foreshadow?.recall ?? 0}</Tag>
                {evalSnap.tension && (evalSnap.tension.covered ?? 0) > 0 && (
                  <Tag style={{ marginRight: 0 }}>张力均值 {evalSnap.tension.mean} · 极差 {evalSnap.tension.swing}（覆盖 {evalSnap.tension.covered} 章）</Tag>
                )}
                <Tag style={{ marginRight: 0 }}>上下文合计 {evalSnap.contextRunes ?? 0} rune</Tag>
              </div>
            )}
            {evalCmp && (
              evalCmp.stale ? (
                <Alert type="warning" showIcon style={{ marginBottom: 6 }}
                  message="prompts 已变更，基线过期——重设基线后再对比（旧基线不可比）。" />
              ) : (
                <Table<EvalCmpItem> size="small" rowKey="metric"
                  dataSource={evalCmp.items ?? []} pagination={false}
                  columns={[
                    { title: '指标', dataIndex: 'metric', width: 150 },
                    { title: '基线', dataIndex: 'from', width: 90, render: (v: number) => <span style={{ fontVariantNumeric: 'tabular-nums' }}>{v}</span> },
                    { title: '当前', dataIndex: 'to', width: 90, render: (v: number) => <span style={{ fontVariantNumeric: 'tabular-nums' }}>{v}</span> },
                    { title: 'Δ', dataIndex: 'delta', width: 80, render: (v: number) => <span style={{ fontVariantNumeric: 'tabular-nums' }}>{v}</span> },
                    {
                      title: '方向', dataIndex: 'dir', width: 80,
                      render: (v: string) => (
                        <span style={{ color: v === 'worse' ? 'var(--color-error, var(--md-sys-color-error, #cf1322))' : v === 'better' ? 'var(--color-success)' : undefined }}>
                          {v === 'worse' ? '↓ 恶化' : v === 'better' ? '↑ 改善' : '— 持平'}
                        </span>
                      ),
                    },
                  ]} />
              )
            )}
            <Typography.Text type="secondary" style={{ fontSize: 12, display: 'block', marginTop: 4 }}>
              快照=零 LLM 确定性聚合（AI 味/质量闸/伏笔回收/张力/上下文合计）；基线落 eval/baseline.json，
              对比给出逐指标方向（张力均值/波动下降=节奏塌）。
            </Typography.Text>
          </div>
          {/* 伏笔 findings */}
          {findings.length > 0 && (
            <div style={{ marginTop: 14 }}>
              <div style={{ fontSize: 13, fontWeight: 600, marginBottom: 6 }}>伏笔登记告警（{findings.length}）</div>
              {findings.map((f, i) => (
                <div key={i} style={{ padding: '3px 0', fontSize: 12.5, display: 'flex', gap: 6 }}>
                  <Tag color="purple" style={{ marginRight: 0, flexShrink: 0 }}>{String(f.code ?? 'lint')}</Tag>
                  <span>{String(f.message ?? '')}</span>
                </div>
              ))}
            </div>
          )}
          <Typography.Text type="secondary" style={{ fontSize: 12, display: 'block', marginTop: 12 }}>
            体检为纯确定性检查（写前契约齐备性 / 段落句式标点 / AI 味高频词 / 伏笔登记自洽），零模型调用；
            逐章 AI 深检（情节分析/审阅/一致性）在章节页「章节体检」单章触发。
          </Typography.Text>
        </div>
      )}
    </Modal>
  )
}
