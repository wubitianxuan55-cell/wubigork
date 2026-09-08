import React, { useMemo, useEffect, useCallback } from 'react'
import { diffLines } from '../../../gaea/lib/diff'

/**
 * DiffReview — 内联 Diff 审查组件
 *
 * 显示 original 与 revised 之间的行级差异
 * 绿色 = 新增行, 红色 = 删除行, 白色 = 相同行
 *
 * 支持 ⌘Y 接受 / ⌘N 拒绝 快捷键
 *
 * Props:
 *   original — 原始文本
 *   revised — AI 编辑后的文本
 *   onAccept — 接受回调
 *   onReject — 拒绝回调
 */
interface DiffReviewProps {
  original: string
  revised: string
  onAccept: () => void
  onReject: () => void
}

interface DiffLine {
  type: 'same' | 'add' | 'del'
  content: string
  lineNum: number
}

/**
 * 统一 LCS（gaea/lib/diff.ts，W3 收口刀）映射到本地 DiffLine：ctx→same、
 * add/del 直传。lineNum 口径沿旧实现：same/add=新侧行号、del=旧侧行号
 * （现有 UI 不渲染 lineNum，仅保持导出类型契约不劣化）。空串=0 行边界：
 * lib 的 split('\n') 会把 '' 拆成 ['']，这里按旧口径滤掉该幽灵行，避免
 * 空原文/新文多显示一行空删除/新增（显示不劣化）。
 */
function computeDiff(oldText: string, newText: string): DiffLine[] {
  const oldEmpty = oldText === ''
  const newEmpty = newText === ''
  const result: DiffLine[] = []
  let oi = 0 // 旧侧行游标（del/ctx 推进）
  let ni = 0 // 新侧行游标（add/ctx 推进）
  for (const row of diffLines(oldText, newText)) {
    if (row.type === 'ctx') {
      result.push({ type: 'same', content: row.text, lineNum: ni + 1 })
      oi += 1
      ni += 1
    } else if (row.type === 'del') {
      if (oldEmpty && row.text === '') continue
      result.push({ type: 'del', content: row.text, lineNum: oi + 1 })
      oi += 1
    } else {
      if (newEmpty && row.text === '') continue
      result.push({ type: 'add', content: row.text, lineNum: ni + 1 })
      ni += 1
    }
  }
  return result
}

const DiffReview: React.FC<DiffReviewProps> = ({ original, revised, onAccept, onReject }) => {
  const diffLines = useMemo(() => computeDiff(original, revised), [original, revised])

  // 快捷键
  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'y') {
        e.preventDefault()
        onAccept()
      }
      if ((e.metaKey || e.ctrlKey) && e.key === 'n') {
        e.preventDefault()
        onReject()
      }
    },
    [onAccept, onReject],
  )

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  // 统计
  const stats = useMemo(() => {
    const adds = diffLines.filter(l => l.type === 'add').length
    const dels = diffLines.filter(l => l.type === 'del').length
    return { adds, dels, total: diffLines.length }
  }, [diffLines])

  return (
    <div>
      {/* 统计条 */}
      <div
        style={{
          display: 'flex',
          gap: 12,
          marginBottom: 12,
          fontSize: 12,
          color: 'var(--color-text-secondary)',
        }}
      >
        <span style={{ color: 'var(--color-success)' }}>+{stats.adds} 新增</span>
        <span style={{ color: 'var(--color-destructive)' }}>-{stats.dels} 删除</span>
        <span>{stats.total} 行</span>
        <span style={{ marginLeft: 'auto', opacity: 0.6 }}>
          ⌘Y 接受 &nbsp; ⌘N 拒绝
        </span>
      </div>

      {/* Diff 视图 */}
      <div
        style={{
          background: 'var(--bg-deep)',
          borderRadius: 'var(--radius-md)',
          border: '1px solid var(--border-subtle)',
          maxHeight: 360,
          overflow: 'auto',
          fontFamily: "'JetBrains Mono', 'Source Code Pro', 'Fira Code', 'Consolas', monospace",
          fontSize: 13,
          lineHeight: 1.7,
        }}
      >
        {diffLines.map((line, i) => {
          let bg = 'transparent'
          let prefix = '  '
          if (line.type === 'add') {
            bg = 'rgba(34, 197, 94, 0.1)'
            prefix = '+ '
          } else if (line.type === 'del') {
            bg = 'rgba(239, 68, 68, 0.1)'
            prefix = '- '
          }

          return (
            <div
              key={i}
              style={{
                padding: '1px 12px',
                background: bg,
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-all',
                color:
                  line.type === 'add'
                    ? 'var(--color-success)'
                    : line.type === 'del'
                      ? 'var(--color-destructive)'
                      : 'var(--color-text)',
              }}
            >
              <span style={{ opacity: 0.4, marginRight: 8, userSelect: 'none' }}>
                {prefix}
              </span>
              {line.content || '\u00A0'}
            </div>
          )
        })}

        {diffLines.length === 0 && (
          <div style={{ padding: 12, color: 'var(--color-text-secondary)', textAlign: 'center' }}>
            无差异
          </div>
        )}
      </div>
    </div>
  )
}

export default DiffReview
export type { DiffReviewProps, DiffLine }
