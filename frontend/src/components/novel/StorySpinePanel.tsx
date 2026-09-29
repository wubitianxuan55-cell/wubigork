// StorySpinePanel.tsx — 故事骨架面板（长篇刀3：整书层结构化持续状态）。
// 主题论证（控制理念/反论）+ 人物弧线水位 + 结构节拍位 + 支线开关表 + 未解问题池；
// 「AI 提炼骨架」提案填充表单（确认制：落盘必须走「保存」）；
// 「结构体检」七维度确定性报告（NovelStoryHealth，零 LLM）。
import React, { useCallback, useEffect, useState } from 'react'
import { Alert, Button, Empty, Input, Modal, Select, Space, Spin, Tag, Typography, message } from 'antd'
import { app } from '../../gaea/lib/bridge'

interface ArcBeatView { chapter: number; stage: string; note: string }
interface ArcView { character_id: string; name: string; want: string; need: string; misbelief: string; beats: ArcBeatView[] }
interface BeatView { id: string; name: string; chapter: number; status: string }
interface ThreadView { id: string; name: string; mice_type: string; open_chapter: number; last_advanced_chapter: number; status: string; close_chapter: number; note: string }
interface QuestionView { id: string; question: string; raised_chapter: number; status: string; answer_chapter: number }
interface SpineView {
  theme?: { controlling_idea?: string; counter_idea?: string }
  arcs?: ArcView[]
  beats?: BeatView[]
  threads?: ThreadView[]
  open_questions?: QuestionView[]
}
interface FindingView { code: string; severity: string; message: string; ref?: string }

const SEV_COLOR: Record<string, string> = { S1: 'red', S2: 'orange', S3: 'default' }
const BEAT_STATUS = [{ value: 'planned', label: '计划' }, { value: 'hit', label: '已达成' }]
const THREAD_STATUS = [
  { value: 'active', label: '活跃' }, { value: 'parked', label: '停放' },
  { value: 'closed', label: '已闭合' }, { value: 'abandoned', label: '已放弃' },
]
const MICE_TYPES = [
  { value: 'query', label: '悬念' }, { value: 'character', label: '人物' },
  { value: 'event', label: '事件' }, { value: 'place', label: '地点' }, { value: 'thing', label: '物件' },
]
const ARC_STAGES = [
  { value: 'setup', label: '确立' }, { value: 'provoke', label: '初次反证' },
  { value: 'escalate', label: '反证递进' }, { value: 'proof', label: '行动推翻' },
]
const uid = () => 'sp_' + Math.random().toString(36).slice(2, 9)

export default function StorySpinePanel({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [proposing, setProposing] = useState(false)
  const [checking, setChecking] = useState(false)
  const [spine, setSpine] = useState<SpineView>({})
  const [findings, setFindings] = useState<FindingView[] | null>(null)
  const [curChapter, setCurChapter] = useState(0)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const sp = (await app.NovelStorySpineGet()) as SpineView | null
      setSpine(sp ?? {})
    } catch (e) {
      message.error(`故事骨架读取失败：${e instanceof Error ? e.message : String(e)}`)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (open) { void load(); setFindings(null) }
  }, [open, load])

  const save = async () => {
    setSaving(true)
    try {
      await app.NovelStorySpineSave(JSON.stringify({ version: 1, ...spine }))
      message.success('故事骨架已保存（生成时注入切片）')
    } catch (e) {
      message.error(e instanceof Error ? e.message : '保存失败（校验拒绝会点名问题）')
    } finally {
      setSaving(false)
    }
  }

  const propose = async () => {
    setProposing(true)
    try {
      const p = (await app.NovelStorySpinePropose()) as Record<string, unknown> | null
      if (!p) return
      // 提案填充表单（确认制）：作者审后仍须点「保存」才落盘。
      setSpine((prev) => ({
        ...prev,
        theme: {
          controlling_idea: typeof p.controlling_idea === 'string' ? p.controlling_idea : prev.theme?.controlling_idea ?? '',
          counter_idea: typeof p.counter_idea === 'string' ? p.counter_idea : prev.theme?.counter_idea ?? '',
        },
        arcs: Array.isArray(p.arcs) ? (p.arcs as ArcView[]).map((a) => ({ ...a, beats: [] })) : prev.arcs ?? [],
        beats: Array.isArray(p.beats) ? (p.beats as BeatView[]).map((b) => ({ ...b, status: b.status || 'planned' })) : prev.beats ?? [],
        threads: Array.isArray(p.threads)
          ? (p.threads as ThreadView[]).map((t, i) => ({ ...t, id: t.id || uid() + i, last_advanced_chapter: t.last_advanced_chapter || t.open_chapter }))
          : prev.threads ?? [],
        open_questions: Array.isArray(p.questions)
          ? (p.questions as QuestionView[]).map((q, i) => ({ ...q, id: q.id || uid() + 'q' + i }))
          : prev.open_questions ?? [],
      }))
      message.info('AI 骨架已填入表单（未落盘）——核对后点「保存」生效')
    } catch (e) {
      message.error(e instanceof Error ? e.message : '提炼失败')
    } finally {
      setProposing(false)
    }
  }

  const check = async () => {
    setChecking(true)
    try {
      const r = (await app.NovelStoryHealth()) as { findings?: FindingView[]; current_chapter?: number } | null
      setFindings(r?.findings ?? [])
      setCurChapter(r?.current_chapter ?? 0)
    } catch (e) {
      message.error(e instanceof Error ? e.message : '结构体检失败')
    } finally {
      setChecking(false)
    }
  }

  const patch = (fn: (draft: SpineView) => SpineView) => setSpine((prev) => fn(structuredClone(prev)))

  const labelStyle: React.CSSProperties = { fontSize: 12, fontWeight: 600, margin: '10px 0 4px' }
  const rowStyle: React.CSSProperties = { display: 'flex', gap: 6, marginBottom: 6, flexWrap: 'wrap' }

  return (
    <Modal
      open={open}
      title="故事骨架（整书层：主题 / 弧线 / 节拍 / 支线 / 悬念）"
      onCancel={onClose}
      width={720}
      footer={(
        <Space>
          <Button size="small" onClick={onClose}>关闭</Button>
          <Button size="small" loading={proposing} onClick={() => void propose()} data-testid="spine-propose">AI 提炼骨架</Button>
          <Button size="small" loading={checking} onClick={() => void check()} data-testid="spine-check">结构体检</Button>
          <Button size="small" type="primary" loading={saving} onClick={() => void save()} data-testid="spine-save">保存骨架</Button>
        </Space>
      )}
    >
      {loading ? <div style={{ textAlign: 'center', padding: '32px 0' }}><Spin /></div> : (
        <div data-testid="spine-body">
          <div style={labelStyle}>主题论证（控制理念 + 反论最强陈述）</div>
          <div style={rowStyle}>
            <Input size="small" style={{ flex: 2 }} placeholder="控制理念：这本书在讲什么道理（正命题）" data-testid="spine-idea"
              value={spine.theme?.controlling_idea ?? ''}
              onChange={(e) => patch((d) => { d.theme = { ...d.theme, controlling_idea: e.target.value }; return d })} />
            <Input size="small" style={{ flex: 2 }} placeholder="反论最强陈述（对手/世界的最强反驳）"
              value={spine.theme?.counter_idea ?? ''}
              onChange={(e) => patch((d) => { d.theme = { ...d.theme, counter_idea: e.target.value }; return d })} />
          </div>

          <div style={labelStyle}>人物弧线（want / need / misbelief + 水位点）</div>
          {(spine.arcs ?? []).map((a, i) => (
            <div key={i} style={{ ...rowStyle, borderLeft: '2px solid var(--color-primary)', paddingLeft: 8 }}>
              <Input size="small" style={{ width: 90 }} placeholder="角色名" value={a.name}
                onChange={(e) => patch((d) => { d.arcs![i].name = e.target.value; return d })} />
              <Input size="small" style={{ width: 140 }} placeholder="want 表层欲望" value={a.want}
                onChange={(e) => patch((d) => { d.arcs![i].want = e.target.value; return d })} />
              <Input size="small" style={{ width: 140 }} placeholder="need 深层需要" value={a.need}
                onChange={(e) => patch((d) => { d.arcs![i].need = e.target.value; return d })} />
              <Input size="small" style={{ width: 160 }} placeholder="misbelief 错误信念" value={a.misbelief}
                onChange={(e) => patch((d) => { d.arcs![i].misbelief = e.target.value; return d })} />
              <Button size="small" danger onClick={() => patch((d) => { d.arcs!.splice(i, 1); return d })}>删</Button>
              <div style={{ display: 'flex', gap: 6, width: '100%', flexWrap: 'wrap' }}>
                {(a.beats ?? []).map((b, j) => (
                  <span key={j} style={{ display: 'inline-flex', gap: 4, fontSize: 12 }}>
                    <Input size="small" type="number" style={{ width: 64 }} value={b.chapter}
                      onChange={(e) => patch((d) => { d.arcs![i].beats[j].chapter = Number(e.target.value) || 0; return d })} />
                    <Select size="small" style={{ width: 100 }} value={b.stage || 'setup'} options={ARC_STAGES}
                      onChange={(v) => patch((d) => { d.arcs![i].beats[j].stage = v; return d })} />
                    <Input size="small" style={{ width: 180 }} placeholder="这一水位发生了什么" value={b.note}
                      onChange={(e) => patch((d) => { d.arcs![i].beats[j].note = e.target.value; return d })} />
                    <Button size="small" danger onClick={() => patch((d) => { d.arcs![i].beats.splice(j, 1); return d })}>×</Button>
                  </span>
                ))}
                <Button size="small" onClick={() => patch((d) => {
                  d.arcs![i].beats = [...(d.arcs![i].beats ?? []), { chapter: 1, stage: 'setup', note: '' }]
                  return d
                })}>+ 水位</Button>
              </div>
            </div>
          ))}
          <Button size="small" onClick={() => patch((d) => {
            d.arcs = [...(d.arcs ?? []), { character_id: '', name: '', want: '', need: '', misbelief: '', beats: [] }]
            return d
          })} data-testid="spine-add-arc">+ 弧线</Button>

          <div style={labelStyle}>结构节拍位</div>
          {(spine.beats ?? []).map((b, i) => (
            <div key={i} style={rowStyle}>
              <Input size="small" style={{ width: 120 }} placeholder="ID（如 midpoint）" value={b.id}
                onChange={(e) => patch((d) => { d.beats![i].id = e.target.value; return d })} />
              <Input size="small" style={{ width: 160 }} placeholder="名称（中点：价值翻转）" value={b.name}
                onChange={(e) => patch((d) => { d.beats![i].name = e.target.value; return d })} />
              <Input size="small" type="number" style={{ width: 76 }} placeholder="章号" value={b.chapter || ''}
                onChange={(e) => patch((d) => { d.beats![i].chapter = Number(e.target.value) || 0; return d })} />
              <Select size="small" style={{ width: 96 }} value={b.status || 'planned'} options={BEAT_STATUS}
                onChange={(v) => patch((d) => { d.beats![i].status = v; return d })} />
              <Button size="small" danger onClick={() => patch((d) => { d.beats!.splice(i, 1); return d })}>删</Button>
            </div>
          ))}
          <Button size="small" onClick={() => patch((d) => {
            d.beats = [...(d.beats ?? []), { id: '', name: '', chapter: 0, status: 'planned' }]
            return d
          })} data-testid="spine-add-beat">+ 节拍</Button>

          <div style={labelStyle}>支线开关表（MICE：开了没关=遗忘）</div>
          {(spine.threads ?? []).map((t, i) => (
            <div key={i} style={rowStyle}>
              <Input size="small" style={{ width: 130 }} placeholder="支线名" value={t.name}
                onChange={(e) => patch((d) => { d.threads![i].name = e.target.value; return d })} />
              <Select size="small" style={{ width: 84 }} allowClear placeholder="MICE" value={t.mice_type || undefined} options={MICE_TYPES}
                onChange={(v) => patch((d) => { d.threads![i].mice_type = v ?? ''; return d })} />
              <Input size="small" type="number" style={{ width: 70 }} placeholder="开启章" value={t.open_chapter || ''}
                onChange={(e) => patch((d) => { d.threads![i].open_chapter = Number(e.target.value) || 0; return d })} />
              <Input size="small" type="number" style={{ width: 70 }} placeholder="最后推进" value={t.last_advanced_chapter || ''}
                onChange={(e) => patch((d) => { d.threads![i].last_advanced_chapter = Number(e.target.value) || 0; return d })} />
              <Select size="small" style={{ width: 92 }} value={t.status || 'active'} options={THREAD_STATUS}
                onChange={(v) => patch((d) => { d.threads![i].status = v; return d })} />
              <Input size="small" style={{ width: 150 }} placeholder="一句话" value={t.note}
                onChange={(e) => patch((d) => { d.threads![i].note = e.target.value; return d })} />
              <Button size="small" danger onClick={() => patch((d) => { d.threads!.splice(i, 1); return d })}>删</Button>
            </div>
          ))}
          <Button size="small" onClick={() => patch((d) => {
            d.threads = [...(d.threads ?? []), { id: uid(), name: '', mice_type: '', open_chapter: 1, last_advanced_chapter: 1, status: 'active', close_chapter: 0, note: '' }]
            return d
          })} data-testid="spine-add-thread">+ 支线</Button>

          <div style={labelStyle}>未解问题池（读者视角的开放悬念）</div>
          {(spine.open_questions ?? []).map((q, i) => (
            <div key={i} style={rowStyle}>
              <Input size="small" style={{ flex: 2 }} placeholder="问题（如：凶手到底是谁？）" value={q.question}
                onChange={(e) => patch((d) => { d.open_questions![i].question = e.target.value; return d })} />
              <Input size="small" type="number" style={{ width: 70 }} placeholder="提出章" value={q.raised_chapter || ''}
                onChange={(e) => patch((d) => { d.open_questions![i].raised_chapter = Number(e.target.value) || 0; return d })} />
              <Select size="small" style={{ width: 92 }} value={q.status || 'open'}
                options={[{ value: 'open', label: '未解' }, { value: 'answered', label: '已答' }]}
                onChange={(v) => patch((d) => { d.open_questions![i].status = v; return d })} />
              <Button size="small" danger onClick={() => patch((d) => { d.open_questions!.splice(i, 1); return d })}>删</Button>
            </div>
          ))}
          <Button size="small" onClick={() => patch((d) => {
            d.open_questions = [...(d.open_questions ?? []), { id: uid(), question: '', raised_chapter: 1, status: 'open', answer_chapter: 0 }]
            return d
          })} data-testid="spine-add-question">+ 问题</Button>

          {findings !== null && (
            <div style={{ marginTop: 14 }} data-testid="spine-report">
              <Typography.Text style={{ fontSize: 12, fontWeight: 600 }}>
                结构体检（当前第 {curChapter} 章 · {findings.length === 0 ? '无发现' : `${findings.length} 项` }）
              </Typography.Text>
              {findings.length === 0 ? (
                <Alert style={{ marginTop: 6 }} type="success" showIcon message="纵向结构健康：无遗忘支线/超期悬念/滞后节拍。" />
              ) : (
                <div style={{ marginTop: 6 }}>
                  {findings.map((f, i) => (
                    <Alert
                      key={i} style={{ marginBottom: 6 }}
                      type={f.severity === 'S1' ? 'error' : f.severity === 'S2' ? 'warning' : 'info'}
                      message={<span><Tag color={SEV_COLOR[f.severity]} style={{ marginRight: 6 }}>{f.severity}</Tag>{f.ref ? `${f.ref}：` : ''}{f.message}</span>}
                    />
                  ))}
                </div>
              )}
            </div>
          )}
          {((spine.arcs?.length ?? 0) + (spine.beats?.length ?? 0) + (spine.threads?.length ?? 0) + (spine.open_questions?.length ?? 0) === 0
            && !spine.theme?.controlling_idea) && (
            <Empty
              style={{ marginTop: 14 }}
              description={<>还没有骨架——点「AI 提炼骨架」从故事主线生成初稿，或手动添加</>}
            />
          )}
        </div>
      )}
    </Modal>
  )
}
