// BookHealthPanel.tsx — 全书体检面板（GenerationGate 闭环收口，规格
// 进度计划/gaea-gen-gate-closure-20260916.md）：触发式全量编译，纯确定性
// 零 LLM，作者点按跑完出报告（无常驻无定时）。消费 RunBookHealthCheck：
// 头部聚合卡（总章/契约问题章/质量问题章/最差 AI 味/V2 覆盖/伏笔 findings）
// + 逐章表（章号/字数/契约/质量/AI 味，越线红标）+ 伏笔 findings 列表。
// v4.340：情感曲线区（t7 观察池）——按需逐章拉分析 V2 的情感弧线强度，
// 纯 SVG 折线（零新依赖）；未分析章诚实跳过并计数。
import { softTextStyle } from '../../utils/uiStyles'
import React, { useCallback, useEffect, useState } from 'react'
import { Alert, Button, Empty, Modal, Spin, Table, Tag, Typography, message } from 'antd'
import { app } from '../../gaea/lib/bridge'
import type { BookHealthReportView, ChapterAnalysisV2View } from '../../gaea/lib/bridge/novel'


/** 逐章体检行（JSX 泛型不认索引访问，抽别名）。 */
type HealthRow = BookHealthReportView['chapters'][number]

/** 情感曲线数据点（章号 + 强度 0-10 + 主导情绪）。 */
interface CurvePoint { num: number; intensity: number; emotion: string }

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
  const [curveLoading, setCurveLoading] = useState(false)
  const [curve, setCurve] = useState<CurvePoint[] | null>(null)
  const [curveSkipped, setCurveSkipped] = useState(0)

  const run = useCallback(async () => {
    setLoading(true)
    setCurve(null) // 重跑体检后曲线失效（章数可能变化），需重新生成
    setCurveSkipped(0)
    try {
      setReport(await app.RunBookHealthCheck())
    } catch (e) {
      message.error(`全书体检失败：${e instanceof Error ? e.message : String(e)}`)
      setReport(null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (open) void run()
  }, [open, run])

  // 情感曲线：逐章读分析 V2（缺档/无情感弧线跳过并计数；后端缺档只报
  // 「尚未分析」，不触发重建，循环安全）。体检触发式口径一致：按钮按需生成。
  const buildCurve = useCallback(async () => {
    if (!report) return
    setCurveLoading(true)
    const pts: CurvePoint[] = []
    let skipped = 0
    for (const row of report.chapters ?? []) {
      try {
        const v2 = (await app.NovelChapterAnalysisV2(row.chapterNum)) as ChapterAnalysisV2View | null
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
    setCurve(pts)
    setCurveSkipped(skipped)
    setCurveLoading(false)
  }, [report])

  const fs = report?.foreshadow
  const findings = fs?.findings ?? []

  return (
    <Modal open={open} title="全书体检（确定性 · 零模型调用）" onCancel={onClose} width={780} destroyOnHidden
      footer={[
        <Button key="rerun" size="small" loading={loading} onClick={() => void run()}>重新体检</Button>,
        <Button key="close" size="small" onClick={onClose}>关闭</Button>,
      ]}>
      {loading ? (
        <div style={{ padding: '48px 0', textAlign: 'center' }}><Spin /><div style={{ ...softTextStyle, marginTop: 8 }}>全量编译中（逐章写前契约 / 写后质量 / AI 味 / 伏笔登记）…</div></div>
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
