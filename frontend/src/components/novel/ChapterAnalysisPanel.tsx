// ChapterAnalysisPanel.tsx — 章节分析 V2 面板（t7 前端接线收官刀：分析 V2
// 消费 + 标注高亮，规格 进度计划/gaea-analysis-v2-panel-t7-20260916.md）。
// analysis-v2.json 九维此前零消费、AnalyzeChapter 绑定困在 Legacy 面零入口
// ——本面板闭环：打开拉 NovelChapterAnalysisV2；缺档空态给「分析本章」按钮
// （AnalyzeChapter 慢路径 loading，落盘后重拉 V2+标注）；标注区双模式
// （列表 / 只读高亮视图，rune 偏移→code-unit 换算后按段 mark，不动编辑器）。
// 直调 app.* 同 RewriteModal 先例；视图类型从 bridge/novel.ts 导入。
import { softTextStyle } from '../../utils/uiStyles'
import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { Alert, Button, Empty, Modal, Segmented, Select, Spin, Tag, Typography, message } from 'antd'
import { app } from '../../gaea/lib/bridge'
import type { ChapterAnalysisV2View, ChapterAnnotation } from '../../gaea/lib/bridge/novel'
import { buildAnnSegments } from './create/annotationMarks'

const cmpThStyle: React.CSSProperties = { textAlign: 'left', fontWeight: 500, padding: '2px 10px 2px 0', color: 'var(--v3-fg-soft, #6b7280)', fontSize: 11.5 }
const cmpTdStyle: React.CSSProperties = { padding: '2px 10px 2px 0', borderTop: '1px dashed rgba(0,0,0,0.10)', fontSize: 12.5 }
const labelTextStyle: React.CSSProperties = { fontSize: 12, color: 'var(--v3-fg-soft, #6b7280)', marginBottom: 4 }

/** 标注 type → 徽标/高亮配色（Q6：Tag 走 antd 色名，mark 用软底 rgba）。 */
const TYPE_VIEWS: Record<string, { label: string; color?: string; bg: string }> = {
  hook: { label: '钩子', color: 'blue', bg: 'rgba(59,130,246,0.22)' },
  foreshadow: { label: '伏笔', color: 'purple', bg: 'rgba(168,85,247,0.22)' },
  plot_point: { label: '情节', color: 'green', bg: 'rgba(34,197,94,0.22)' },
  conflict: { label: '冲突', color: 'red', bg: 'rgba(239,68,68,0.22)' },
  character: { label: '角色', color: 'orange', bg: 'rgba(249,115,22,0.22)' },
  suggestion: { label: '建议', bg: 'rgba(0,0,0,0.10)' },
}

const PACING_LABELS: Record<string, string> = { slow: '舒缓', moderate: '适中', fast: '紧凑', varied: '多变' }

// 高亮段构建收敛到 create/annotationMarks.ts（v4.344 起：与编辑器持久 overlay 同一实现）

/** 小节标题（带计数；空节由调用方隐藏）。 */
function Section({ title, count, children }: { title: string; count: number; children: React.ReactNode }) {
  if (count <= 0 && !React.Children.count(children)) return null
  return (
    <div style={{ marginBottom: 12 }}>
      <div style={{ ...labelTextStyle, fontWeight: 600 }}>{title}{count > 0 ? `（${count}）` : ''}</div>
      {children}
    </div>
  )
}

export default function ChapterAnalysisPanel({ open, onClose, chapterNum, content, onLocate, chapterOptions }: {
  open: boolean
  onClose: () => void
  /** 当前章号（0/null=无激活章）。 */
  chapterNum: number | null
  /** 章正文（CreatePage 编辑器态；高亮视图数据源，rune 换算基准）。 */
  content: string
  /** 可选：把标注定位到正文编辑器光标（CreatePage 传入；仅面板章=编辑章时
   *  提供——gate 跳转他章时编辑器内容不一致，不提供即隐藏入口）。 */
  onLocate?: (ann: ChapterAnnotation) => void
  /** 可选：可对比的章号清单（t7 观察池「多章对比」最小形态——章际对比）。
   *  提供且可选章 ≥1 时渲染对比区；缺省隐藏。 */
  chapterOptions?: number[]
}) {
  const [loading, setLoading] = useState(false)
  const [analyzing, setAnalyzing] = useState(false)
  const [data, setData] = useState<ChapterAnalysisV2View | null>(null)
  const [missing, setMissing] = useState(false)
  const [anns, setAnns] = useState<ChapterAnnotation[]>([])
  const [annMode, setAnnMode] = useState<'list' | 'highlight'>('list')
  // 章际对比（t7 多章对比最小形态）：选定对比章拉其 V2，四维+情感强度并排差值。
  const [cmpNum, setCmpNum] = useState<number | null>(null)
  const [cmpV2, setCmpV2] = useState<ChapterAnalysisV2View | null>(null)
  const [cmpMissing, setCmpMissing] = useState(false)
  const [cmpLoading, setCmpLoading] = useState(false)

  const load = useCallback(async (num: number) => {
    setLoading(true)
    setMissing(false)
    setData(null)
    try {
      setData(await app.NovelChapterAnalysisV2(num))
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      if (msg.includes('尚未分析')) setMissing(true)
      else message.error(`读取分析失败：${msg}`)
    } finally {
      setLoading(false)
    }
    try {
      setAnns((await app.NovelChapterAnnotations(num)) ?? [])
    } catch {
      setAnns([]) // 标注缺档是正常态（空列表；重建由后端按需做）
    }
  }, [])

  useEffect(() => {
    if (open && chapterNum && chapterNum > 0) {
      setAnnMode('list')
      setCmpNum(null)
      setCmpV2(null)
      setCmpMissing(false)
      void load(chapterNum)
    }
    if (open && !chapterNum) {
      setData(null)
      setMissing(false)
      setAnns([])
    }
  }, [open, chapterNum, load])

  /** 加载对比章的 V2（尚未分析→行内提示；其他错误→toast）。 */
  const cmpLoad = async (num: number) => {
    setCmpNum(num)
    setCmpLoading(true)
    setCmpV2(null)
    setCmpMissing(false)
    try {
      setCmpV2(await app.NovelChapterAnalysisV2(num))
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      if (msg.includes('尚未分析')) setCmpMissing(true)
      else message.error(`读取第 ${num} 章分析失败：${msg}`)
    } finally {
      setCmpLoading(false)
    }
  }

  const runAnalysis = async () => {
    if (!chapterNum) return
    setAnalyzing(true)
    try {
      await app.AnalyzeChapter(chapterNum)
      message.success('分析完成')
      await load(chapterNum)
    } catch (e) {
      message.error(`分析失败：${e instanceof Error ? e.message : String(e)}`)
    } finally {
      setAnalyzing(false)
    }
  }

  const segments = useMemo(() => buildAnnSegments(content ?? '', anns), [content, anns])

  /** 列表行点击：切到高亮视图并滚动到对应锚点（未命中段不跳）。 */
  const jumpTo = (ann: ChapterAnnotation) => {
    const idx = segments.findIndex(s => s.ann === ann)
    if (idx < 0) return
    setAnnMode('highlight')
    requestAnimationFrame(() => {
      document.getElementById(`ann-mark-${idx}`)?.scrollIntoView({ block: 'center' })
    })
  }

  const r = data?.result
  const scores = r?.scores
  const meta = data
    ? [metaAt(data.analyzed_at), data.engine, data.model].filter(Boolean).join(' · ')
    : ''

  return (
    <Modal open={open} title={`章节分析（第 ${chapterNum || '—'} 章）`} onCancel={onClose} footer={null} width={860} destroyOnHidden>
      {chapterNum ? null : (
        <Empty description="先在左侧选择一章" style={{ padding: '32px 0' }} />
      )}
      {chapterNum && loading ? (
        <div style={{ padding: '32px 0', textAlign: 'center' }}><Spin /></div>
      ) : chapterNum && missing ? (
        <div data-testid="analysis-empty">
          <Alert type="info" showIcon message={`第 ${chapterNum} 章尚未分析`}
            description="运行「分析本章」后，这里会展示九维分析（钩子/伏笔/冲突/情感/角色/组织/情节/场景/节奏），并在正文中高亮标注锚点。分析同时会同步伏笔库与记忆回填。" />
          <Button type="primary" size="small" style={{ marginTop: 12 }} loading={analyzing}
            data-testid="analysis-run-btn" onClick={() => void runAnalysis()}>分析本章</Button>
        </div>
      ) : chapterNum && r ? (
        <div style={{ maxHeight: '68vh', overflowY: 'auto', paddingRight: 4 }} data-testid="analysis-body">
          {/* 头部：总分 + 三维 + 评分理由 */}
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 14, marginBottom: 4 }}>
            <span style={{ fontSize: 30, fontWeight: 650, fontVariantNumeric: 'tabular-nums' }}>
              {(scores?.overall ?? 0).toFixed(1)}
            </span>
            <span style={softTextStyle}>综合分</span>
            {scores && [
              ['节奏', scores.pacing], ['代入', scores.engagement], ['连贯', scores.coherence],
            ].map(([label, v]) => (
              <Tag key={String(label)} style={{ marginRight: 0 }}>{String(label)} {Number(v ?? 0).toFixed(1)}</Tag>
            ))}
            {analyzing && <Button size="small" loading={analyzing} onClick={() => void runAnalysis()}>重新分析</Button>}
          </div>
          {scores?.score_justification && (
            <Typography.Paragraph type="secondary" style={{ fontSize: 12.5, marginBottom: 8 }}>{scores.score_justification}</Typography.Paragraph>
          )}
          {/* chips：节奏/阶段/文白比 + meta */}
          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', alignItems: 'center', marginBottom: 12 }}>
            {r.pacing && <Tag color="geekblue" style={{ marginRight: 0 }}>节奏 {PACING_LABELS[r.pacing] ?? r.pacing}</Tag>}
            {r.plot_stage && <Tag color="cyan" style={{ marginRight: 0 }}>{r.plot_stage}</Tag>}
            {r.dialogue_ratio != null && <Tag style={{ marginRight: 0 }}>对话 {Math.round(r.dialogue_ratio * 100)}%</Tag>}
            {r.description_ratio != null && <Tag style={{ marginRight: 0 }}>叙述 {Math.round(r.description_ratio * 100)}%</Tag>}
            {meta && <span style={softTextStyle}>{meta}</span>}
          </div>

          {/* 章际对比（t7 多章对比最小形态）：选定对比章拉其 V2，四维+情感强度差值。
              四维 Δ 上=绿（更好）；情感强度中性不判好坏。 */}
          {(chapterOptions ?? []).filter(n => n !== chapterNum).length > 0 && (
            <div style={{ marginBottom: 12 }} data-testid="analysis-compare">
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 6 }}>
                <span style={{ fontSize: 13, fontWeight: 600 }}>章际对比</span>
                <Select size="small" style={{ minWidth: 128 }} placeholder="选择对比章"
                  data-testid="analysis-compare-select"
                  value={cmpNum ?? undefined}
                  onChange={(v) => void cmpLoad(Number(v))}
                  options={(chapterOptions ?? []).filter(n => n !== chapterNum).map(n => ({ value: n, label: `第 ${n} 章` }))} />
                {cmpLoading && <Spin size="small" />}
              </div>
              {cmpNum && cmpV2?.result?.scores && scores && (() => {
                const cs = cmpV2.result.scores
                const rows: Array<{ label: string; mine?: number; other?: number; neutral?: boolean }> = [
                  { label: '综合', mine: scores.overall, other: cs.overall },
                  { label: '节奏', mine: scores.pacing, other: cs.pacing },
                  { label: '代入', mine: scores.engagement, other: cs.engagement },
                  { label: '连贯', mine: scores.coherence, other: cs.coherence },
                  { label: '情感强度', mine: r?.emotional_arc?.intensity, other: cmpV2.result?.emotional_arc?.intensity, neutral: true },
                ]
                return (
                  <table style={{ borderCollapse: 'collapse' }} data-testid="analysis-compare-table">
                    <thead>
                      <tr>
                        <th style={cmpThStyle} />
                        <th style={cmpThStyle}>本章</th>
                        <th style={cmpThStyle}>第 {cmpNum} 章</th>
                        <th style={cmpThStyle}>Δ</th>
                      </tr>
                    </thead>
                    <tbody>
                      {rows.map(row => {
                        const mine = typeof row.mine === 'number' ? row.mine : null
                        const other = typeof row.other === 'number' ? row.other : null
                        const d = mine != null && other != null ? other - mine : null
                        return (
                          <tr key={row.label}>
                            <td style={cmpTdStyle}>{row.label}</td>
                            <td style={{ ...cmpTdStyle, fontVariantNumeric: 'tabular-nums' }}>{mine != null ? mine.toFixed(1) : '—'}</td>
                            <td style={{ ...cmpTdStyle, fontVariantNumeric: 'tabular-nums' }}>{other != null ? other.toFixed(1) : '—'}</td>
                            <td style={{
                              ...cmpTdStyle,
                              fontVariantNumeric: 'tabular-nums',
                              color: d == null || row.neutral || d === 0 ? undefined
                                : d > 0 ? 'var(--color-success, var(--md-sys-color-success, #2e7d32))'
                                  : 'var(--color-error, var(--md-sys-color-error, #cf1322))',
                            }}>
                              {d == null ? '—' : `${d > 0 ? '+' : ''}${d.toFixed(1)}`}
                            </td>
                          </tr>
                        )
                      })}
                    </tbody>
                  </table>
                )
              })()}
              {cmpNum && cmpV2 && !cmpMissing && !cmpV2.result?.scores && (
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>对比章分析无评分数据。</Typography.Text>
              )}
              {cmpNum && cmpMissing && (
                <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                  第 {cmpNum} 章尚未分析——先在章节页运行分析后再对比。
                </Typography.Text>
              )}
            </div>
          )}

          <Section title="钩子" count={r.hooks?.length ?? 0}>
            {(r.hooks ?? []).map((h, i) => (
              <div key={i} style={{ padding: '3px 0', fontSize: 12.5 }}>
                <Tag style={{ marginRight: 4 }}>{h.type || '钩子'}{h.strength ? ` ${h.strength}` : ''}</Tag>
                {h.position && <span style={softTextStyle}>[{h.position}] </span>}
                {h.content}
              </div>
            ))}
          </Section>

          <Section title="伏笔（本章埋设/回收）" count={r.foreshadows?.length ?? 0}>
            {(r.foreshadows ?? []).map((f, i) => (
              <div key={i} style={{ padding: '3px 0', fontSize: 12.5 }}>
                <Tag color={f.type === 'resolved' ? 'green' : 'purple'} style={{ marginRight: 4 }}>
                  {f.type === 'resolved' ? '回收' : '埋设'}{f.strength ? ` ${f.strength}` : ''}
                </Tag>
                {f.title && <b style={{ marginRight: 4 }}>{f.title}</b>}
                {f.content}
                {f.is_long_term && <span style={softTextStyle}>（长线）</span>}
                {f.estimated_resolve_chapter ? <span style={softTextStyle}>→ 预计第 {f.estimated_resolve_chapter} 章</span> : null}
              </div>
            ))}
          </Section>

          {r.conflict && (r.conflict.description || r.conflict.level) && (
            <Section title="冲突" count={r.conflict.types?.length ?? 0}>
              <div style={{ fontSize: 12.5 }}>
                <Tag color="red" style={{ marginRight: 4}}>烈度 {r.conflict.level ?? '—'}</Tag>
                {(r.conflict.types ?? []).map(tp => <Tag key={tp} style={{ marginRight: 4 }}>{tp}</Tag>)}
                {r.conflict.resolution_progress != null && (
                  <span style={softTextStyle}>化解 {Math.round(r.conflict.resolution_progress * 100)}%</span>
                )}
                {r.conflict.description && <div style={{ marginTop: 4 }}>{r.conflict.description}</div>}
              </div>
            </Section>
          )}

          {r.emotional_arc?.primary_emotion && (
            <Section title="情感" count={1}>
              <div style={{ fontSize: 12.5 }}>
                <Tag color="magenta" style={{ marginRight: 4 }}>{r.emotional_arc.primary_emotion}{r.emotional_arc.intensity ? ` ${r.emotional_arc.intensity}` : ''}</Tag>
                {r.emotional_arc.curve && <span>{r.emotional_arc.curve}</span>}
                {(r.emotional_arc.secondary_emotions ?? []).length > 0 && (
                  <span style={softTextStyle}>（次级：{(r.emotional_arc.secondary_emotions ?? []).join('、')}）</span>
                )}
              </div>
            </Section>
          )}

          <Section title="角色变化" count={r.character_states?.length ?? 0}>
            {(r.character_states ?? []).map((c, i) => (
              <div key={i} style={{ padding: '3px 0', fontSize: 12.5 }}>
                <b>{c.name || '—'}</b>
                {c.old_state && c.new_state && <span>：{c.old_state} → {c.new_state}</span>}
                {c.psychological_change && <span style={softTextStyle}>（{c.psychological_change}）</span>}
                {c.survival_status && c.survival_status !== 'active' && (
                  <Tag color="volcano" style={{ marginLeft: 4 }}>{c.survival_status}</Tag>
                )}
              </div>
            ))}
          </Section>

          <Section title="组织变化" count={r.organization_states?.length ?? 0}>
            {(r.organization_states ?? []).map((o, i) => (
              <div key={i} style={{ padding: '3px 0', fontSize: 12.5 }}>
                <b>{o.org_name}</b>
                {o.destroyed && <Tag color="red" style={{ marginLeft: 4 }}>覆灭</Tag>}
                {o.power_value != null && <Tag style={{ marginLeft: 4 }}>势力 {o.power_value}</Tag>}
                {(o.member_changes ?? []).map((m, j) => (
                  <div key={j} style={softTextStyle}>{m.character_name} · {m.change_type}{m.position ? ` · ${m.position}` : ''}</div>
                ))}
              </div>
            ))}
          </Section>

          <Section title="情节推进" count={r.plot_points?.length ?? 0}>
            {(r.plot_points ?? []).map((p, i) => (
              <div key={i} style={{ padding: '3px 0', fontSize: 12.5 }}>
                <Tag color="green" style={{ marginRight: 4 }}>{p.type || '情节'}</Tag>
                {p.content}
                {p.importance != null && <span style={softTextStyle}>（权重 {Math.round(p.importance * 100)}%）</span>}
              </div>
            ))}
          </Section>

          <Section title="场景" count={r.scenes?.length ?? 0}>
            {(r.scenes ?? []).map((s, i) => (
              <Tag key={i} style={{ marginRight: 4 }}>{[s.location, s.atmosphere, s.duration].filter(Boolean).join(' · ')}</Tag>
            ))}
          </Section>

          <Section title="改进建议" count={r.suggestions?.length ?? 0}>
            {(r.suggestions ?? []).map((sg, i) => (
              <div key={i} style={{ padding: '3px 0', fontSize: 12.5 }}>{i + 1}. {sg}</div>
            ))}
          </Section>

          {r.summary && (
            <Section title="本章小结" count={0}>
              <Typography.Paragraph style={{ fontSize: 12.5, marginBottom: 0 }}>{r.summary}</Typography.Paragraph>
            </Section>
          )}

          {/* 标注区：列表 | 高亮视图 */}
          <div style={{ marginTop: 16, paddingTop: 12, borderTop: '1px dashed rgba(0,0,0,0.12)' }} data-testid="analysis-annotations">
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 8 }}>
              <span style={{ fontSize: 13, fontWeight: 600 }}>正文标注（{anns.length}）</span>
              <Segmented size="small" value={annMode} onChange={v => setAnnMode(v as 'list' | 'highlight')}
                options={[{ label: '列表', value: 'list' }, { label: '高亮视图', value: 'highlight' }]} />
            </div>
            {anns.length === 0 ? (
              <Typography.Text type="secondary" style={{ fontSize: 12 }}>
                无标注（分析未命中可锚定的关键词，或该章尚未分析）。
              </Typography.Text>
            ) : annMode === 'list' ? (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                {anns.map((a, i) => {
                  const v = TYPE_VIEWS[a.type]
                  const anchored = segments.some(s => s.ann === a)
                  return (
                    <div key={i} data-testid="analysis-ann-row"
                      onClick={() => jumpTo(a)}
                      style={{ display: 'flex', gap: 6, alignItems: 'flex-start', cursor: anchored ? 'pointer' : 'default', padding: '3px 4px', borderRadius: 4 }}>
                      <Tag color={v?.color} style={{ marginRight: 0, flexShrink: 0 }}>{v?.label ?? a.type}</Tag>
                      <span style={{ fontSize: 12.5 }}>
                        {a.title && <b style={{ marginRight: 4 }}>{a.title}</b>}
                        {a.content}
                      </span>
                      {a.importance ? <span style={softTextStyle}>{Math.round(a.importance * 100)}%</span> : null}
                      {anchored && onLocate && (
                        <Button size="small" type="link" data-testid="analysis-ann-locate"
                          style={{ fontSize: 11, padding: 0, height: 'auto', flexShrink: 0 }}
                          onClick={(e) => { e.stopPropagation(); onLocate(a) }}>
                          编辑器定位
                        </Button>
                      )}
                      {anchored && <span style={softTextStyle}>↩高亮</span>}
                    </div>
                  )
                })}
              </div>
            ) : (
              <div data-testid="analysis-highlight-view"
                style={{ position: 'relative', maxHeight: 360, overflowY: 'auto', background: 'rgba(0,0,0,0.03)', borderRadius: 6, padding: '10px 12px', fontSize: 12.5, lineHeight: 1.9, whiteSpace: 'pre-wrap' }}>
                {(() => {
                  const text = content ?? ''
                  const parts: React.ReactNode[] = []
                  let last = 0
                  segments.forEach((s, i) => {
                    if (s.start > last) parts.push(<span key={`t${i}`}>{text.slice(last, s.start)}</span>)
                    const v = TYPE_VIEWS[s.ann.type]
                    parts.push(
                      <mark id={`ann-mark-${i}`} key={`m${i}`} data-testid="analysis-highlight"
                        style={{ background: v?.bg ?? 'rgba(0,0,0,0.10)', borderRadius: 2, padding: '0 1px', color: 'inherit' }}
                        title={`${s.ann.title || ''} ${s.ann.content}`.trim()}>
                        {text.slice(s.start, s.end)}
                      </mark>,
                    )
                    last = s.end
                  })
                  parts.push(<span key="tail">{text.slice(last)}</span>)
                  return parts
                })()}
              </div>
            )}
          </div>
        </div>
      ) : chapterNum ? (
        <Empty description="没有可展示的分析结果" style={{ padding: '32px 0' }} />
      ) : null}
    </Modal>
  )
}

/** meta 时间展示（ISO → 本地短格式；解析失败原样）。 */
function metaAt(iso?: string): string {
  if (!iso) return ''
  try {
    const d = new Date(iso)
    return Number.isNaN(d.getTime()) ? iso : d.toLocaleString()
  } catch {
    return iso
  }
}
