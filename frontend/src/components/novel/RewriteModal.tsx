// RewriteModal.tsx — 驱动式整章重写弹窗（t4-C3 前端消费刀）
// 表单（来源：分析建议勾选/自定义指令/重点方向/保留元素/目标字数）→ 运行 →
// 结果（相似度与篇幅统计 + 新全文 + 应用/放弃）；应用后可恢复原文。
// 生成后不自动落章（对齐后端语义）：应用 = 写回正文并刷新编辑器。
import React, { useCallback, useEffect, useMemo, useState } from 'react'
import { Button, Checkbox, Input, InputNumber, Modal, Spin, Tag, message } from 'antd'
import { app } from '../../gaea/lib/bridge'
import type { NovelRewriteResult } from '../../gaea/lib/bridge/novel'

const FOCUS_OPTIONS = [
  { value: 'pacing', label: '节奏' },
  { value: 'emotion', label: '情感' },
  { value: 'description', label: '描写' },
  { value: 'dialogue', label: '对话' },
  { value: 'conflict', label: '冲突' },
]

interface RewriteModalProps {
  open: boolean
  chapterNum: number | null
  onClose: () => void
  /** 应用/恢复成功后回调（父层刷新章节正文） */
  onApplied: () => void
}

const RewriteModal: React.FC<RewriteModalProps> = ({ open, chapterNum, onClose, onApplied }) => {
  const [suggestions, setSuggestions] = useState<string[]>([])
  const [selected, setSelected] = useState<number[]>([])
  const [customInstr, setCustomInstr] = useState('')
  const [focusAreas, setFocusAreas] = useState<string[]>([])
  const [preserveStructure, setPreserveStructure] = useState(true)
  const [preserveTraits, setPreserveTraits] = useState(true)
  const [targetWords, setTargetWords] = useState(3000)
  const [running, setRunning] = useState(false)
  const [result, setResult] = useState<NovelRewriteResult | null>(null)
  const [applied, setApplied] = useState(false)
  const [busyAction, setBusyAction] = useState(false)

  // 打开时拉该章分析建议（无分析 → 空数组，来源锁自定义指令）
  useEffect(() => {
    if (!open || chapterNum == null) return
    setSelected([])
    setResult(null)
    setApplied(false)
    setSuggestions([])
    void app.NovelChapterSuggestions(chapterNum)
      .then(setSuggestions)
      .catch(() => setSuggestions([]))
  }, [open, chapterNum])

  const toggleSuggestion = (idx: number) => {
    setSelected((prev) => (prev.includes(idx) ? prev.filter((i) => i !== idx) : [...prev, idx]))
  }

  const canRun = useMemo(() => {
    if (running) return false
    const hasInstr = customInstr.trim() !== ''
    const hasIdx = selected.length > 0
    if (suggestions.length > 0 && hasIdx) return true
    return hasInstr
  }, [running, customInstr, selected, suggestions.length])

  const run = useCallback(async () => {
    if (chapterNum == null) return
    const hasIdx = selected.length > 0
    const hasInstr = customInstr.trim() !== ''
    const source = hasIdx && hasInstr ? 'mixed' : hasIdx ? 'analysis_suggestions' : 'custom'
    const req = {
      source,
      suggestion_indices: hasIdx ? selected : undefined,
      custom_instructions: hasInstr ? customInstr.trim() : undefined,
      focus_areas: focusAreas,
      preserve_elements: {
        preserve_structure: preserveStructure,
        preserve_character_traits: preserveTraits,
      },
      target_word_count: targetWords,
    }
    setRunning(true)
    try {
      const r = await app.NovelChapterRewrite(chapterNum, JSON.stringify(req))
      setResult(r)
      setApplied(false)
    } catch (err: unknown) {
      message.error(`重写失败：${err instanceof Error ? err.message : '未知错误'}`)
    } finally {
      setRunning(false)
    }
  }, [chapterNum, selected, customInstr, focusAreas, preserveStructure, preserveTraits, targetWords])

  const apply = useCallback(async () => {
    if (!result) return
    setBusyAction(true)
    try {
      await app.NovelApplyRewriteVersion(chapterNum!, result.versionId)
      setApplied(true)
      message.success('已应用重写版本并写回正文')
      onApplied()
    } catch (err: unknown) {
      message.error(`应用失败：${err instanceof Error ? err.message : '未知错误'}`)
    } finally {
      setBusyAction(false)
    }
  }, [chapterNum, result, onApplied])

  const discard = useCallback(async () => {
    if (!result) return
    setBusyAction(true)
    try {
      await app.NovelDiscardRewriteVersion(chapterNum!, result.versionId)
      message.success('已丢弃该版本（快照保留，可在版本列表再应用）')
      onClose()
    } catch (err: unknown) {
      message.error(`丢弃失败：${err instanceof Error ? err.message : '未知错误'}`)
    } finally {
      setBusyAction(false)
    }
  }, [chapterNum, result, onClose])

  const restore = useCallback(async () => {
    if (!result) return
    setBusyAction(true)
    try {
      await app.NovelRestoreRewriteVersion(chapterNum!, result.versionId)
      setApplied(false)
      message.success('已恢复原文')
      onApplied()
    } catch (err: unknown) {
      message.error(`恢复失败：${err instanceof Error ? err.message : '未知错误'}`)
    } finally {
      setBusyAction(false)
    }
  }, [chapterNum, result, onApplied])

  return (
    <Modal
      title={`整章重写 · 第 ${chapterNum ?? ''} 章`}
      open={open}
      onCancel={onClose}
      width={720}
      footer={null}
      destroyOnClose
    >
      {running ? (
        <div style={{ textAlign: 'center', padding: '48px 0' }} data-testid="rewrite-running">
          <Spin size="large" />
          <div style={{ marginTop: 12 }}>正在按修改指令重写整章（可能需要 1-3 分钟）…</div>
        </div>
      ) : result ? (
        <div data-testid="rewrite-result">
          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap', marginBottom: 8 }}>
            <Tag color="blue">相似度 {result.similarity?.toFixed?.(1) ?? result.similarity}%</Tag>
            <Tag>{result.originalWordCount} → {result.newWordCount} 字（{result.change >= 0 ? '+' : ''}{result.change}）</Tag>
            {applied && <Tag color="green">已应用</Tag>}
          </div>
          <div
            style={{
              maxHeight: 320, overflowY: 'auto', padding: 8, fontSize: 13, lineHeight: 1.7,
              border: '1px solid var(--color-border, var(--md-sys-color-outline-variant))', borderRadius: 6,
              whiteSpace: 'pre-wrap',
            }}
          >
            {result.newContent}
          </div>
          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 12 }}>
            {!applied ? (
              <>
                <Button danger disabled={busyAction} onClick={() => void discard()}>放弃版本</Button>
                <Button type="primary" loading={busyAction} onClick={() => void apply()}>应用并写回正文</Button>
              </>
            ) : (
              <Button loading={busyAction} onClick={() => void restore()}>恢复原文</Button>
            )}
          </div>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
          <div>
            <div style={{ fontSize: 12, marginBottom: 4 }}>分析建议（来自该章最近一次分析）</div>
            {suggestions.length === 0 ? (
              <div style={{ fontSize: 12, color: 'var(--color-text-tertiary, #999)' }}>
                该章暂无分析结果，可先用自定义指令重写；建议先执行章节分析。
              </div>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                {suggestions.map((s, i) => (
                  <Checkbox key={i} checked={selected.includes(i)} onChange={() => toggleSuggestion(i)}>
                    <span style={{ fontSize: 12 }}>{i + 1}. {s}</span>
                  </Checkbox>
                ))}
              </div>
            )}
          </div>
          <div>
            <div style={{ fontSize: 12, marginBottom: 4 }}>自定义修改要求</div>
            <Input.TextArea
              rows={2} value={customInstr} onChange={(e) => setCustomInstr(e.target.value)}
              placeholder="如：收紧中段节奏，结尾悬念更利落"
            />
          </div>
          <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', alignItems: 'center' }}>
            <span style={{ fontSize: 12 }}>重点方向：</span>
            <Checkbox.Group options={FOCUS_OPTIONS} value={focusAreas} onChange={(v) => setFocusAreas(v as string[])} />
          </div>
          <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap' }}>
            <Checkbox checked={preserveStructure} onChange={(e) => setPreserveStructure(e.target.checked)}>
              保持整体结构与情节框架
            </Checkbox>
            <Checkbox checked={preserveTraits} onChange={(e) => setPreserveTraits(e.target.checked)}>
              保持角色性格与行为模式
            </Checkbox>
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <span style={{ fontSize: 12 }}>目标字数</span>
            <InputNumber size="small" min={500} max={10000} step={500} style={{ width: 96 }}
              value={targetWords} onChange={(v) => setTargetWords(Number(v) || 3000)} />
          </div>
          <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
            <Button
              type="primary" disabled={!canRun}
              onClick={() => void run()}
            >
              开始重写
            </Button>
          </div>
        </div>
      )}
    </Modal>
  )
}

export default RewriteModal
