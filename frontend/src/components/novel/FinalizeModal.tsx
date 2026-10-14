// FinalizeModal.tsx — 章节定稿回写弹窗（阶段八 8.4）
// 定稿 = 作者宣布本章正文为最终版。AI 基于当前正文提取三件套草案（阶段合账/
// 角色状态/伏笔状态），作者勾选确认后才落库——建议制红线：确认前零写盘。
// 分支章只标记定稿（登记表/合账/状态卡主线口径，Go 侧 Skipped 申报回显）。
import React, { useCallback, useEffect, useState } from 'react'
import { Alert, Button, Checkbox, Input, Modal, Spin, Tag, Typography, message } from 'antd'
import { app } from '../../gaea/lib/bridge'
import { errText } from '../../utils/errText'

const { Text } = Typography

/** 角色状态草案条目（Go finalizeCharDraft 镜像）。 */
interface CharDraft {
  id: string
  name: string
  field: string
  text: string
  note?: string
}

/** 伏笔草案条目（Go types.ForeshadowHit 镜像，取渲染所需字段）。 */
interface FSDraft {
  title?: string
  content?: string
  type?: string
  category?: string
}

interface StageRecapDraft {
  stage: number
  start: number
  end: number
  text: string
}

/** 草案（Go FinalizeChapterDraft 返回 map 的前端投影）。 */
interface FinalizeDraft {
  chapterNum: number
  branch: string
  finalized: boolean
  finalization?: Record<string, unknown>
  characters: CharDraft[]
  foreshadows: FSDraft[]
  stageRecap?: StageRecapDraft
  warnings: string[]
}

/** 落库回执（Go FinalizeChapterApply 返回 map 的前端投影）。 */
interface FinalizeReceipt {
  finalized: boolean
  applied?: {
    stageRecap?: { start: number; end: number; text: string } | null
    characters?: number
    foreshadows?: { plantedCount: number; resolvedCount: number; errors?: string[] } | null
  } | null
  errors?: string[]
  skipped?: string
}

interface FinalizeModalProps {
  open: boolean
  chapterNum: number | null
  branch: string
  onClose: () => void
  /** 定稿成功回调（父层刷新定稿徽标/状态）。 */
  onFinalized: () => void
}

const LIST_STYLE: React.CSSProperties = {
  maxHeight: 180, overflowY: 'auto', padding: '4px 8px',
  border: '1px solid var(--color-border, var(--md-sys-color-outline-variant))', borderRadius: 6,
}

const FinalizeModal: React.FC<FinalizeModalProps> = ({ open, chapterNum, branch, onClose, onFinalized }) => {
  const [loading, setLoading] = useState(false)
  const [loadError, setLoadError] = useState('')
  const [draft, setDraft] = useState<FinalizeDraft | null>(null)
  // 勾选与可编辑文本（建议制的「作者确认」载体；逐条勾选=逐条落库范围）
  const [recapOn, setRecapOn] = useState(true)
  const [recapText, setRecapText] = useState('')
  const [charChecks, setCharChecks] = useState<boolean[]>([])
  const [charTexts, setCharTexts] = useState<string[]>([])
  const [fsChecks, setFsChecks] = useState<boolean[]>([])
  const [applying, setApplying] = useState(false)
  const [receipt, setReceipt] = useState<FinalizeReceipt | null>(null)

  const load = useCallback(async (num: number) => {
    setLoading(true); setLoadError(''); setReceipt(null); setDraft(null)
    try {
      const raw = (await app.FinalizeChapterDraft(num, branch)) as Record<string, unknown>
      const d: FinalizeDraft = {
        chapterNum: num,
        branch: branch,
        finalized: raw.finalized === true,
        finalization: raw.finalization as Record<string, unknown> | undefined,
        characters: (raw.characters as CharDraft[]) || [],
        foreshadows: (raw.foreshadows as FSDraft[]) || [],
        stageRecap: raw.stageRecap as StageRecapDraft | undefined,
        warnings: (raw.warnings as string[]) || [],
      }
      setDraft(d)
      setRecapOn(!!d.stageRecap)
      setRecapText(d.stageRecap?.text || '')
      setCharChecks(d.characters.map(() => true))
      setCharTexts(d.characters.map((c) => c.text))
      setFsChecks(d.foreshadows.map(() => true))
    } catch (err) {
      setLoadError(errText(err, '定稿草案加载失败'))
    } finally {
      setLoading(false)
    }
  }, [branch])

  useEffect(() => {
    if (!open || chapterNum == null || chapterNum < 1) return
    void load(chapterNum)
  }, [open, chapterNum, load])

  const apply = useCallback(async () => {
    if (chapterNum == null || !draft) return
    setApplying(true)
    try {
      const payload: Record<string, unknown> = { note: '' }
      if (draft.stageRecap && recapOn && recapText.trim()) {
        payload.stage_recap = { stage: draft.stageRecap.stage, start: draft.stageRecap.start, end: draft.stageRecap.end, text: recapText.trim() }
      }
      payload.characters = draft.characters
        .map((c, i) => ({ ...c, text: (charTexts[i] || '').trim() }))
        .filter((_, i) => charChecks[i] && charTexts[i]?.trim())
      payload.foreshadows = draft.foreshadows.filter((_, i) => fsChecks[i])
      const res = (await app.FinalizeChapterApply(chapterNum, branch, JSON.stringify(payload))) as unknown as FinalizeReceipt
      setReceipt(res)
      onFinalized()
      message.success('已定稿')
    } catch (err) {
      message.error(errText(err, '定稿落库失败'))
    } finally {
      setApplying(false)
    }
  }, [chapterNum, branch, draft, recapOn, recapText, charChecks, charTexts, fsChecks, onFinalized])

  const nothingSelected =
    !draft || (!draft.stageRecap && draft.characters.length === 0 && draft.foreshadows.length === 0)

  const renderReceipt = () => {
    if (!receipt) return null
    const a = receipt.applied
    const lines: string[] = []
    if (a?.stageRecap) lines.push(`阶段合账（第 ${a.stageRecap.start}~${a.stageRecap.end} 章）已落盘缓存`)
    if (typeof a?.characters === 'number' && a.characters > 0) lines.push(`角色状态 ${a.characters} 条已写入状态卡`)
    if (a?.foreshadows) lines.push(`伏笔同步：埋设 ${a.foreshadows.plantedCount} 条、回收 ${a.foreshadows.resolvedCount} 条`)
    if (receipt.skipped) lines.push(receipt.skipped)
    if (lines.length === 0) lines.push('本章已标记定稿（未选择任何回写项）')
    return (
      <div data-testid="finalize-receipt">
        <Alert type={receipt.errors?.length ? 'warning' : 'success'} showIcon
          message="定稿完成" description={
            <div>
              {lines.map((l, i) => <div key={i}>{l}</div>)}
              {receipt.errors?.map((e, i) => <div key={`e${i}`} style={{ color: 'var(--color-danger, #cf1322)' }}>留痕：{e}</div>)}
            </div>
          } />
        <div style={{ marginTop: 12, textAlign: 'right' }}>
          <Button onClick={onClose}>关闭</Button>
        </div>
      </div>
    )
  }

  return (
    <Modal
      title={`定稿本章（第 ${chapterNum ?? '?'}${branch ? branch : ''} 章）`}
      open={open}
      onCancel={onClose}
      width={640}
      footer={null}
      destroyOnClose
    >
      {loading ? (
        <div style={{ textAlign: 'center', padding: 32 }}><Spin tip="正在基于当前正文提取三件套草案（不落库）…" /></div>
      ) : loadError ? (
        <Alert type="error" showIcon message={loadError} />
      ) : receipt ? (
        renderReceipt()
      ) : draft ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <Alert type="info" showIcon
            message="定稿 = 本章正文定为最终版"
            description="以下三件套由 AI 基于当前正文提取（**尚未落库**）。勾选确认后才写入：定稿后下一章生成将读到最新状态。重新定稿会覆盖上一批回写。" />
          {draft.finalized && (
            <Alert type="warning" showIcon message="本章已定稿过——本次确认将重新提取并覆盖上一批回写（幂等，不重复堆积）" />
          )}
          {draft.warnings.map((w, i) => (
            <Alert key={i} type="warning" showIcon message={w} />
          ))}

          {draft.stageRecap && (
            <div data-testid="finalize-recap-section">
              <Checkbox checked={recapOn} onChange={(e) => setRecapOn(e.target.checked)}>阶段合账（第 {draft.stageRecap.start}~{draft.stageRecap.end} 章）——落盘缓存，阶段开篇章生成不再重烧</Checkbox>
              <Input.TextArea
                data-testid="finalize-recap-text"
                value={recapText}
                onChange={(e) => setRecapText(e.target.value)}
                disabled={!recapOn}
                autoSize={{ minRows: 3, maxRows: 8 }}
                style={{ marginTop: 6 }}
              />
            </div>
          )}

          {draft.characters.length > 0 && (
            <div data-testid="finalize-chars-section">
              <Text strong>角色状态变化（写入状态卡「状态演进」）</Text>
              <div style={{ ...LIST_STYLE, marginTop: 6 }}>
                {draft.characters.map((c, i) => (
                  <div key={c.id} style={{ display: 'flex', gap: 6, alignItems: 'flex-start', padding: '4px 0' }}>
                    <Checkbox checked={charChecks[i]} onChange={(e) => setCharChecks((p) => p.map((v, j) => (j === i ? e.target.checked : v)))} />
                    <Tag>{c.name}</Tag>
                    <Input size="small" value={charTexts[i]} onChange={(e) => setCharTexts((p) => p.map((v, j) => (j === i ? e.target.value : v)))} style={{ flex: 1 }} />
                  </div>
                ))}
              </div>
            </div>
          )}

          {draft.foreshadows.length > 0 && (
            <div data-testid="finalize-fs-section">
              <Text strong>伏笔状态变化（同步伏笔登记表）</Text>
              <div style={{ ...LIST_STYLE, marginTop: 6 }}>
                {draft.foreshadows.map((f, i) => (
                  <div key={i} style={{ display: 'flex', gap: 6, alignItems: 'flex-start', padding: '4px 0' }}>
                    <Checkbox checked={fsChecks[i]} onChange={(e) => setFsChecks((p) => p.map((v, j) => (j === i ? e.target.checked : v)))} />
                    <Tag color={f.type === 'resolved' ? 'green' : 'blue'}>{f.type === 'resolved' ? '回收' : '埋设'}</Tag>
                    <span style={{ fontSize: 13 }}>{f.title || f.content}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {nothingSelected && (
            <Alert type="info" message="本章未提取到可回写的变化，定稿仅做标记。" />
          )}

          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 8, marginTop: 4 }}>
            <Button onClick={onClose}>取消</Button>
            <Button type="primary" data-testid="finalize-confirm" loading={applying} onClick={apply}>
              确认定稿
            </Button>
          </div>
        </div>
      ) : null}
    </Modal>
  )
}

export default FinalizeModal
