import React, { useMemo, useEffect, useCallback } from 'react'

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

/** 简单的行级 diff 算法（前端版，用于纯展示） */
function computeDiff(oldText: string, newText: string): DiffLine[] {
  const oldLines = oldText ? oldText.split('\n') : []
  const newLines = newText ? newText.split('\n') : []
  const result: DiffLine[] = []

  let oi = 0
  let ni = 0

  while (oi < oldLines.length && ni < newLines.length) {
    if (oldLines[oi] === newLines[ni]) {
      result.push({ type: 'same', content: newLines[ni], lineNum: ni + 1 })
      oi++
      ni++
    } else {
      // 前向搜索
      let found = false
      for (let look = ni + 1; look < ni + 4 && look < newLines.length; look++) {
        if (newLines[look] === oldLines[oi]) {
          for (; ni < look; ni++) {
            result.push({ type: 'add', content: newLines[ni], lineNum: ni + 1 })
          }
          found = true
          break
        }
      }
      if (!found) {
        result.push({ type: 'del', content: oldLines[oi], lineNum: oi + 1 })
        oi++
        if (ni < newLines.length) {
          result.push({ type: 'add', content: newLines[ni], lineNum: ni + 1 })
          ni++
        }
      }
    }
  }

  // 剩余行
  while (oi < oldLines.length) {
    result.push({ type: 'del', content: oldLines[oi], lineNum: oi + 1 })
    oi++
  }
  while (ni < newLines.length) {
    result.push({ type: 'add', content: newLines[ni], lineNum: ni + 1 })
    ni++
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
