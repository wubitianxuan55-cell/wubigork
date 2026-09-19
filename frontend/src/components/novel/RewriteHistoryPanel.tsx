// RewriteHistoryPanel.tsx — 重写版本历史面板（t4-C3 余项：版本库前端消费）。
// v4.304 版本库的 List/Get 绑定此前零 UI 消费——跨会话找回历史版本用户到不了。
// 动作按后端状态机门控原样镜像（Apply: completed|discarded|applied 幂等；
// Discard: 仅 completed；Restore: 仅 applied），前端不重复裁决只禁用不可用项。
import React, { useCallback, useEffect, useState } from 'react'
import { Button, Collapse, Empty, Modal, Popconfirm, Spin, Tag, message } from 'antd'
import { app } from '../../gaea/lib/bridge'
import type { RewriteVersionIndex } from '../../gaea/lib/bridge/novel'

const MODE_LABELS: Record<string, string> = { whole: '整章', partial: '局部', deslop: '去味' }
const STATUS_LABELS: Record<string, string> = {
  pending: '排队中', running: '重写中', failed: '失败',
  completed: '已完成', applied: '已应用', discarded: '已放弃',
}
const STATUS_COLORS: Record<string, string> = {
  pending: 'default', running: 'gold', failed: 'red',
  completed: 'blue', applied: 'green', discarded: 'default',
}

interface VersionDetail {
  originalWordCount?: number
  newWordCount?: number
  similarity?: number
  beforeAIScore?: number
  afterAIScore?: number
  originalContent?: string
  newContent?: string
}

export default function RewriteHistoryPanel({ open, chapterNum, onClose, onApplied }: {
  open: boolean
  chapterNum: number | null
  onClose: () => void
  onApplied?: () => void
}) {
  const [rows, setRows] = useState<RewriteVersionIndex[]>([])
  const [loading, setLoading] = useState(false)
  const [expandedIds, setExpandedIds] = useState<Set<string>>(() => new Set())
  const [details, setDetails] = useState<Record<string, VersionDetail>>({})
  const [busyId, setBusyId] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    if (chapterNum == null) { setRows([]); return }
    setLoading(true)
    try {
      setRows(await app.NovelListRewriteVersions(chapterNum))
    } catch (e) {
      message.error(`重写历史加载失败：${e instanceof Error ? e.message : String(e)}`)
      setRows([])
    } finally {
      setLoading(false)
    }
  }, [chapterNum])

  useEffect(() => {
    if (open) { setExpandedIds(new Set()); setDetails({}); void refresh() }
  }, [open, refresh])

  const toggleDetail = async (id: string) => {
    const next = new Set(expandedIds)
    if (next.has(id)) { next.delete(id); setExpandedIds(next); return }
    next.add(id)
    setExpandedIds(next)
    if (!details[id] && chapterNum != null) {
      try {
        const v = await app.NovelGetRewriteVersion(chapterNum, id) as unknown as VersionDetail
        setDetails(d => ({ ...d, [id]: v }))
      } catch (e) {
        message.error(`版本详情加载失败：${e instanceof Error ? e.message : String(e)}`)
      }
    }
  }

  const apply = async (id: string) => {
    if (chapterNum == null) return
    setBusyId(id)
    try {
      await app.NovelApplyRewriteVersion(chapterNum, id)
      message.success('已应用该版本到正文')
      onApplied?.()
      await refresh()
    } catch (e) {
      message.error(e instanceof Error ? e.message : String(e))
    } finally {
      setBusyId(null)
    }
  }

  const discard = async (id: string) => {
    if (chapterNum == null) return
    setBusyId(id)
    try {
      await app.NovelDiscardRewriteVersion(chapterNum, id)
      message.success('已放弃该版本')
      await refresh()
    } catch (e) {
      message.error(e instanceof Error ? e.message : String(e))
    } finally {
      setBusyId(null)
    }
  }

  const restore = async (id: string) => {
    if (chapterNum == null) return
    setBusyId(id)
    try {
      await app.NovelRestoreRewriteVersion(chapterNum, id)
      message.success('已恢复原文')
      onApplied?.()
      await refresh()
    } catch (e) {
      message.error(e instanceof Error ? e.message : String(e))
    } finally {
      setBusyId(null)
    }
  }

  const fmtTime = (iso: string) => {
    try { return new Date(iso).toLocaleString() } catch { return iso }
  }
  const fmtPct = (v?: number) => (typeof v === 'number' && v > 0 ? `${Math.round(v)}%` : '')

  return (
    <Modal
      open={open && chapterNum != null}
      title={`重写历史 · 第 ${chapterNum ?? ''} 章`}
      onCancel={onClose}
      footer={null}
      width={720}
      destroyOnHidden
    >
      {loading ? (
        <div style={{ padding: '32px 0', textAlign: 'center' }}><Spin /></div>
      ) : rows.length === 0 ? (
        <Empty description="该章还没有重写版本（整章重写产生版本后在这里找回）" style={{ padding: '24px 0' }} />
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8, maxHeight: '62vh', overflowY: 'auto' }}>
          {rows.map(v => {
            const st = String(v.status || '')
            const expanded = expandedIds.has(v.id)
            const d = details[v.id]
            const canApply = st === 'completed' || st === 'discarded'
            const canDiscard = st === 'completed'
            const canRestore = st === 'applied'
            return (
              <div key={v.id} data-testid="rewrite-history-row"
                style={{ border: '1px solid var(--v3-line-soft, #e5e7eb)', borderRadius: 8, padding: '8px 12px', background: expanded ? 'color-mix(in srgb, var(--v3-fg-soft, #6b7280) 7%, transparent)' : undefined }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer' }}
                  onClick={() => void toggleDetail(v.id)}>
                  <span style={{ fontSize: 12, color: 'var(--v3-fg-soft, #6b7280)', fontVariantNumeric: 'tabular-nums' }}>{fmtTime(v.createdAt)}</span>
                  <Tag style={{ marginRight: 0 }}>{MODE_LABELS[String(v.mode)] ?? v.mode}</Tag>
                  <Tag color={STATUS_COLORS[st] ?? 'default'} style={{ marginRight: 0 }}>{STATUS_LABELS[st] ?? st}</Tag>
                  {fmtPct(v.similarity) && <span style={{ fontSize: 12, color: 'var(--v3-fg-soft, #6b7280)' }}>相似度 {fmtPct(v.similarity)}</span>}
                  <span style={{ flex: 1 }} />
                  <span style={{ fontSize: 12, color: 'var(--v3-fg-soft, #6b7280)' }}>{expanded ? '收起' : '详情'}</span>
                </div>
                {expanded && (
                  <div style={{ marginTop: 8 }}>
                    {d ? (
                      <>
                        <div data-testid="rewrite-history-metrics" style={{ display: 'flex', gap: 16, flexWrap: 'wrap', fontSize: 12, color: 'var(--v3-fg-soft, #6b7280)', marginBottom: 8 }}>
                          <span>原文 {d.originalWordCount ?? '-'} 字 → 新文 {d.newWordCount ?? '-'} 字</span>
                          {typeof d.similarity === 'number' && d.similarity > 0 && <span>相似度 {Math.round(d.similarity)}%</span>}
                          {(d.beforeAIScore || d.afterAIScore) ? <span>AI 味分 {d.beforeAIScore ?? '-'} → {d.afterAIScore ?? '-'}</span> : null}
                        </div>
                        <div data-testid="rewrite-history-new" style={{ maxHeight: 200, overflowY: 'auto', whiteSpace: 'pre-wrap', fontSize: 13, lineHeight: 1.7, padding: '8px 10px', borderRadius: 6, background: 'rgba(0,0,0,0.03)', marginBottom: 8 }}>
                          {d.newContent || '（无新内容）'}
                        </div>
                        <Collapse size="small" items={[{ key: 'orig', label: '原文快照（恢复原文将写回这段）', children: (
                          <div style={{ maxHeight: 200, overflowY: 'auto', whiteSpace: 'pre-wrap', fontSize: 13, lineHeight: 1.7 }}>{d.originalContent || '（无原文快照）'}</div>
                        ) }]} />
                      </>
                    ) : (
                      <div style={{ padding: '12px 0', textAlign: 'center' }}><Spin size="small" /></div>
                    )}
                    {(canApply || canDiscard || canRestore) && (
                      <div style={{ display: 'flex', gap: 8, marginTop: 8 }}>
                        {canApply && (
                          <Popconfirm title="应用将覆盖当前正文" description="覆盖前请确认已保存当前编辑。" okText="确定" cancelText="取消" onConfirm={() => void apply(v.id)} disabled={busyId === v.id}>
                            <Button size="small" type="primary" loading={busyId === v.id} data-testid="rewrite-history-apply">应用此版本</Button>
                          </Popconfirm>
                        )}
                        {canDiscard && (
                          <Button size="small" danger disabled={busyId === v.id} onClick={() => void discard(v.id)}>放弃版本</Button>
                        )}
                        {canRestore && (
                          <Popconfirm title="恢复原文" description="该版本应用前的原文快照将写回正文。" okText="确定" cancelText="取消" onConfirm={() => void restore(v.id)} disabled={busyId === v.id}>
                            <Button size="small" loading={busyId === v.id} data-testid="rewrite-history-restore">恢复原文</Button>
                          </Popconfirm>
                        )}
                      </div>
                    )}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}
    </Modal>
  )
}
