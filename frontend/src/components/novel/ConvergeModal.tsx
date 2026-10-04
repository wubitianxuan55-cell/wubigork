// ConvergeModal.tsx — 质量收敛修补（长篇刀4：体检→定向修补→复检→三态判停）。
// 预览（dry-run 判据状态）→ 执行（converge-stream 逐轮轨迹）→ 终态如实
// （达标 / 无改善已回滚 / 轮次上限+剩余项）。每轮版本库留痕可恢复。
import { useCallback, useEffect, useRef, useState } from 'react'
import { Alert, Button, InputNumber, Modal, Space, Spin, Tag, Typography, message } from 'antd'
import { subscribeWailsEvent } from '../../gaea/lib/wailsEvents'
import { app } from '../../gaea/lib/bridge'

interface PreviewView {
  tasteScore?: number; targetTaste?: number; converged?: boolean; planRounds?: number
  s1s2?: Array<{ code?: string; severity?: string; message?: string }>
  s3?: Array<{ message?: string }>
}
type RoundLog = { round: number; before?: number; after?: number; applied?: number }

export default function ConvergeModal({ open, chapterNum, onClose, onFinished }: {
  open: boolean
  chapterNum: number | null
  onClose: () => void
  /** 终态回调（正文可能已变：converged/stopped 都刷新章节） */
  onFinished?: () => void
}) {
  const [preview, setPreview] = useState<PreviewView | null>(null)
  const [loading, setLoading] = useState(false)
  const [running, setRunning] = useState(false)
  const [target, setTarget] = useState(40)
  const [rounds, setRounds] = useState<RoundLog[]>([])
  const [finalMsg, setFinalMsg] = useState<{ kind: 'ok' | 'warn' | 'info'; text: string } | null>(null)
  const finishedRef = useRef(false)

  const loadPreview = useCallback(async (num: number) => {
    setLoading(true); setPreview(null); setFinalMsg(null); setRounds([]); finishedRef.current = false
    try {
      const p = (await app.NovelChapterConvergePreview(num, 0)) as PreviewView | null
      setPreview(p ?? {})
      if (p?.targetTaste) setTarget(p.targetTaste)
    } catch (e) {
      message.error(e instanceof Error ? e.message : '预检失败')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (open && chapterNum != null) void loadPreview(chapterNum)
  }, [open, chapterNum, loadPreview])

  // 流事件订阅（挂载期常驻；终态收口）
  useEffect(() => {
    const rt = window.runtime
    if (!rt?.EventsOn) return
    let detach: (() => void) | null = null
    try {
      detach = subscribeWailsEvent(rt, 'converge-stream', (payload: unknown) => {
        const raw = (payload as { detail?: unknown } | null)?.detail ?? payload
        const ev = (raw ?? {}) as { type?: string; chapterNum?: number; round?: number; before?: number; after?: number; applied?: number; message?: string; error?: string; rounds?: number; taste?: number; remainingS3?: number }
        if (typeof ev.chapterNum === 'number' && chapterNum != null && ev.chapterNum !== chapterNum) return
        if (ev.type === 'round-done') {
          setRounds((prev) => [...prev, { round: ev.round ?? 0, before: ev.before, after: ev.after, applied: ev.applied }])
        } else if (ev.type === 'converged') {
          setRunning(false)
          setFinalMsg({ kind: 'ok', text: `已收敛（${ev.rounds ?? '?'} 轮，AI 味降至 ${ev.taste ?? '?'} 分${ev.remainingS3 ? `，剩 ${ev.remainingS3} 项 S3 提示不阻断` : ''}）` })
          if (!finishedRef.current) { finishedRef.current = true; onFinished?.() }
        } else if (ev.type === 'stopped') {
          setRunning(false)
          setFinalMsg({ kind: 'warn', text: ev.message || '已停止' })
          if (!finishedRef.current) { finishedRef.current = true; onFinished?.() }
        } else if (ev.type === 'error') {
          setRunning(false)
          setFinalMsg({ kind: 'warn', text: ev.message || ev.error || '修补失败' })
        }
      })
    } catch { detach = null }
    return () => { if (detach) detach() }
  }, [chapterNum, onFinished])

  const start = async () => {
    if (chapterNum == null || running) return
    setRunning(true); setRounds([]); setFinalMsg(null)
    try {
      const res = (await app.NovelChapterConverge(chapterNum, 0, target)) as { alreadyConverged?: boolean } | null
      if (res?.alreadyConverged) {
        setRunning(false)
        setFinalMsg({ kind: 'info', text: '本章已达标（零轮直答，未发起修补）' })
      }
    } catch (e) {
      setRunning(false)
      message.error(e instanceof Error ? e.message : '启动失败')
    }
  }

  return (
    <Modal
      open={open}
      title={`收敛修补（第 ${chapterNum ?? '—'} 章：体检 → 定向修补 → 复检）`}
      onCancel={() => { if (!running) onClose() }}
      footer={(
        <Space>
          <Button size="small" disabled={running} onClick={onClose}>关闭</Button>
          <Button size="small" disabled={running} onClick={() => chapterNum != null && void loadPreview(chapterNum)}>重新预检</Button>
          <Button size="small" type="primary" loading={running} disabled={loading || preview?.converged === true}
            onClick={() => void start()} data-testid="converge-run">
            {preview?.converged === true ? '已达标' : '开始修补'}
          </Button>
        </Space>
      )}
    >
      {loading ? <div style={{ textAlign: 'center', padding: '24px 0' }}><Spin /></div> : (
        <div data-testid="converge-body">
          {preview && (
            <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap', marginBottom: 8 }}>
              <Tag color={(preview.tasteScore ?? 0) > (preview.targetTaste ?? 40) ? 'red' : 'green'}>
                AI 味 {preview.tasteScore ?? '—'} / 目标 ≤{preview.targetTaste ?? 40}
              </Tag>
              <Tag color={((preview.s1s2?.length) ?? 0) > 0 ? 'orange' : 'green'}>
                S1/S2 信号 {(preview.s1s2?.length) ?? 0} 项
              </Tag>
              <Space size={4}>
                <Typography.Text style={{ fontSize: 12 }}>目标分</Typography.Text>
                <InputNumber size="small" min={0} max={100} value={target} onChange={(v) => setTarget(Number(v) || 40)} disabled={running} />
                <Typography.Text style={{ fontSize: 12 }}>（计划 ≤{preview.planRounds ?? 2} 轮，每轮留痕可恢复）</Typography.Text>
              </Space>
            </div>
          )}
          {(preview?.s1s2?.length ?? 0) > 0 && (
            <Alert
              style={{ marginBottom: 8 }} type="warning" showIcon
              message={<span>{(preview?.s1s2 ?? []).slice(0, 4).map((is, i) => <div key={i} style={{ fontSize: 12 }}>· {is.message}</div>)}
                {(preview?.s1s2?.length ?? 0) > 4 ? <div style={{ fontSize: 12 }}>… 共 {preview?.s1s2?.length} 项</div> : null}</span>}
            />
          )}
          {preview?.converged === true && (
            <Alert type="success" showIcon message="本章已达标：AI 味与确定性信号都过线，无需修补。" />
          )}
          {rounds.length > 0 && (
            <div style={{ marginTop: 8 }} data-testid="converge-rounds">
              {rounds.map((r) => (
                <div key={r.round} style={{ fontSize: 12, color: 'var(--color-text-secondary)', margin: '2px 0' }}>
                  第 {r.round} 轮：AI 味 {r.before ?? '?'} → {r.after ?? '?'}，改写 {r.applied ?? 0} 句
                </div>
              ))}
            </div>
          )}
          {running && <div style={{ marginTop: 8, fontSize: 12, display: 'flex', gap: 8, alignItems: 'center' }}><Spin size="small" /> 修补中（每轮完成即落盘，未改善会自动回滚）…</div>}
          {finalMsg && (
            <Alert
              style={{ marginTop: 8 }} showIcon
              type={finalMsg.kind === 'ok' ? 'success' : finalMsg.kind === 'warn' ? 'warning' : 'info'}
              message={finalMsg.text}
            />
          )}
        </div>
      )}
    </Modal>
  )
}
