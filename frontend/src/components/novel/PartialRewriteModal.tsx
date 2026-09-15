// PartialRewriteModal.tsx — 选段局部重写弹窗（t4-C3 余项前端消费刀）
// 表单（选段预览 + 指令 + 长度模式四档）→ 运行（禁关闭）→ 结果（新选段 + 统计 +
// 应用/放弃）；只重写选中片段，前后文原样拼接，版本库照常留全文快照（应用/放弃
// 走与整章重写相同的版本三绑定）。选区偏移为 rune 口径（EditorPanel 已换算）。
import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { Button, Input, InputNumber, Modal, Radio, Spin, Tag, message } from 'antd'
import { app } from '../../gaea/lib/bridge'
import type { NovelRewriteResult } from '../../gaea/lib/bridge/novel'
import { countTextChars } from '../../utils/text'

/** 长度模式四档（对齐后端 rewrite.LengthMode*）：similar 缺省 */
type LengthMode = 'similar' | 'expand' | 'condense' | 'custom'

const LENGTH_MODE_OPTIONS: Array<{ value: LengthMode; label: string }> = [
  { value: 'similar', label: '相近' },
  { value: 'expand', label: '扩展' },
  { value: 'condense', label: '精简' },
  { value: 'custom', label: '自定义' },
]

/** 选段（EditorPanel onPartialRewrite 回调载荷，rune 偏移） */
interface PartialRewriteSelection {
  start: number
  end: number
  text: string
}

/** 局部重写结果：whole 键集 + partial 增量键（B 线契约，增量键 omitempty 防御） */
interface PartialRewriteResult extends NovelRewriteResult {
  mode?: string
  selectedWordCount?: number
  newSelectedWordCount?: number
  lengthMode?: string
  startPos?: number
  endPos?: number
}

interface PartialRewriteModalProps {
  open: boolean
  chapterNum: number | null
  selection: PartialRewriteSelection | null
  onClose: () => void
  /** 应用成功后回调（父层刷新章节正文） */
  onApplied: () => void
}

const PREVIEW_STYLE: React.CSSProperties = {
  maxHeight: 200, overflowY: 'auto', padding: 8, fontSize: 13, lineHeight: 1.7,
  border: '1px solid var(--color-border, var(--md-sys-color-outline-variant))', borderRadius: 6,
  whiteSpace: 'pre-wrap',
}

const PartialRewriteModal: React.FC<PartialRewriteModalProps> = ({ open, chapterNum, selection, onClose, onApplied }) => {
  const [instr, setInstr] = useState('')
  const [lengthMode, setLengthMode] = useState<LengthMode>('similar')
  const [targetWords, setTargetWords] = useState(500)
  const [running, setRunning] = useState(false)
  const [result, setResult] = useState<PartialRewriteResult | null>(null)
  const [busyAction, setBusyAction] = useState(false)

  // 打开/切换章节时重置表单与结果（对齐 RewriteModal 挂法）
  useEffect(() => {
    if (!open) return
    setInstr('')
    setLengthMode('similar')
    setTargetWords(500)
    setResult(null)
  }, [open, chapterNum])

  const run = useCallback(async () => {
    if (chapterNum == null || !selection || instr.trim() === '') return
    const req = {
      mode: 'partial',
      source: 'custom',
      custom_instructions: instr.trim(),
      start_pos: selection.start,
      end_pos: selection.end,
      selected_text: selection.text,
      length_mode: lengthMode,
      ...(lengthMode === 'custom' ? { target_word_count: targetWords } : {}),
    }
    setRunning(true)
    try {
      const r = await app.NovelChapterRewrite(chapterNum, JSON.stringify(req))
      setResult(r)
    } catch (err: unknown) {
      message.error(`局部重写失败：${err instanceof Error ? err.message : '未知错误'}`)
    } finally {
      setRunning(false)
    }
  }, [chapterNum, selection, instr, lengthMode, targetWords])

  const apply = useCallback(async () => {
    if (!result || chapterNum == null) return
    setBusyAction(true)
    try {
      await app.NovelApplyRewriteVersion(chapterNum, result.versionId)
      message.success('已应用局部重写并写回正文')
      onApplied()
      onClose()
    } catch (err: unknown) {
      message.error(`应用失败：${err instanceof Error ? err.message : '未知错误'}`)
    } finally {
      setBusyAction(false)
    }
  }, [chapterNum, result, onApplied, onClose])

  const discard = useCallback(async () => {
    if (!result || chapterNum == null) return
    setBusyAction(true)
    try {
      await app.NovelDiscardRewriteVersion(chapterNum, result.versionId)
      message.success('已丢弃该版本（快照保留，可在版本列表再应用）')
      onClose()
    } catch (err: unknown) {
      message.error(`丢弃失败：${err instanceof Error ? err.message : '未知错误'}`)
    } finally {
      setBusyAction(false)
    }
  }, [chapterNum, result, onClose])

  // 新选段不在响应键集里：按契约 newContent = 前文 + 新选段 + 后文（rune 拼接），
  // startPos 为原选段起点，新选段 rune 长度 = newSelectedWordCount，据此切出预览。
  const newSelectedText = useMemo(() => {
    if (!result || !selection) return ''
    const runes = Array.from(result.newContent || '')
    const start = typeof result.startPos === 'number' ? result.startPos : selection.start
    const len = typeof result.newSelectedWordCount === 'number'
      ? result.newSelectedWordCount
      : runes.length - start
    return runes.slice(start, Math.min(runes.length, start + len)).join('')
  }, [result, selection])

  return (
    <Modal
      title={`局部重写 · 第 ${chapterNum ?? ''} 章`}
      open={open}
      onCancel={onClose}
      width={640}
      footer={null}
      destroyOnClose
      closable={!running}
      maskClosable={!running}
      keyboard={!running}
    >
      <div data-testid="partial-rewrite-modal">
        {running ? (
          <div style={{ textAlign: 'center', padding: '48px 0' }} data-testid="partial-rewrite-running">
            <Spin size="large" />
            <div style={{ marginTop: 12 }}>正在重写选中段落（选段外一字不动）…</div>
          </div>
        ) : result ? (
          <div data-testid="partial-rewrite-result">
            <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginBottom: 8 }}>
              <Tag color="blue">相似度 {result.similarity?.toFixed?.(1) ?? result.similarity}%</Tag>
              <Tag>{`选段 ${result.selectedWordCount ?? '—'} 字 → 新 ${result.newSelectedWordCount ?? '—'} 字`}</Tag>
            </div>
            <div style={PREVIEW_STYLE}>{newSelectedText}</div>
            <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 12 }}>
              <Button danger disabled={busyAction} onClick={() => void discard()}>放弃版本</Button>
              <Button type="primary" data-testid="partial-rewrite-apply" loading={busyAction} onClick={() => void apply()}>
                应用并写回
              </Button>
            </div>
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            <div>
              <div style={{ fontSize: 12, marginBottom: 4 }}>
                {`选中段落（${countTextChars(selection?.text ?? '')} 字，只重写这段）`}
              </div>
              <div style={PREVIEW_STYLE}>{selection?.text}</div>
            </div>
            <div>
              <div style={{ fontSize: 12, marginBottom: 4 }}>修改指令（必填）</div>
              <Input.TextArea
                rows={2} value={instr} maxLength={1000} showCount
                onChange={(e) => setInstr(e.target.value)}
                placeholder="如：把这段对话改得更锋利，删掉解释性旁白"
              />
            </div>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
              <span style={{ fontSize: 12 }}>长度模式</span>
              <Radio.Group
                value={lengthMode}
                onChange={(e) => setLengthMode(e.target.value as LengthMode)}
                options={LENGTH_MODE_OPTIONS}
              />
              {lengthMode === 'custom' && (
                <>
                  <InputNumber size="small" min={50} max={10000} step={50} style={{ width: 96 }}
                    value={targetWords} onChange={(v) => setTargetWords(Number(v) || 500)} />
                  <span style={{ fontSize: 12, color: 'var(--color-text-tertiary, #999)' }}>目标字数（50-10000）</span>
                </>
              )}
            </div>
            <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
              <Button
                type="primary" data-testid="partial-rewrite-submit"
                disabled={instr.trim() === ''}
                onClick={() => void run()}
              >
                开始重写
              </Button>
            </div>
          </div>
        )}
      </div>
    </Modal>
  )
}

export default PartialRewriteModal
