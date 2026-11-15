// SinReadingPane.tsx — 原罪阅读页签（刀5 rehost）：
//   · 章节导航（拍板 2026-11-15：**唯一锚点=chapters/ 文件夹**——清单读
//     outline.json 的 ChapterFile 节点，只列有定稿文件的章）；
//   · 正文阅读 = 小说板块 ChapterPage 整件复用（吃全局项目 store——书打开
//     时经 openNovelProject 落位）；点击导航派发 novel:open-chapter
//     （ChapterPage 既有事件入口，active 门控语义原样）；
//   · 落章联动：AppBoard 在 turn_done 时 bump chapterTick → 重拉大纲清单。
import React, { useCallback, useEffect } from 'react'
import { useOutlineStore } from '../../stores/outlineStore'
import { sortNodes } from '../../utils/outline'
import type { OutlineNode } from '../../types'
import ChapterPage from '../../pages/ChapterPage'
import { app } from '../../gaea/lib/bridge'

export interface SinReadingPaneProps {
  /** 页签激活（透传 ChapterPage 的窗口级监听门控）。 */
  active: boolean
  /** 落章联动信号（回合结束/外部刷新递增）。 */
  chapterTick: number
  /** 是否已打开书（未打开给引导空态）。 */
  hasBook: boolean
}

interface ChapterRow {
  node: OutlineNode
  num: number
  title: string
}

function chapterNumOf(file: string): number {
  const m = /^(\d{3})\.md$/.exec(file)
  return m ? Number(m[1]) : Number.MAX_SAFE_INTEGER
}

const SinReadingPane: React.FC<SinReadingPaneProps> = ({ active, chapterTick, hasBook }) => {
  const outlines = useOutlineStore((s) => s.outlines)
  const loadOutlines = useOutlineStore((s) => s.loadOutlines)
  const [reloadError, setReloadError] = React.useState('')

  // 大纲清单：挂载 + 落章信号时重拉（文件夹锚点数据源）
  useEffect(() => {
    if (!hasBook) return
    loadOutlines().catch((err: unknown) => setReloadError(err instanceof Error ? err.message : String(err)))
  }, [hasBook, chapterTick, loadOutlines])

  // 章节导航行：只列带定稿文件的章（ChapterFile 非空），按 OrderIndex 排
  const chapters: ChapterRow[] = React.useMemo(() => {
    const sorted = sortNodes(outlines)
    const rows: ChapterRow[] = []
    const walk = (nodes: OutlineNode[]) => {
      for (const n of nodes) {
        if (n.chapter_file) {
          rows.push({ node: n, num: chapterNumOf(n.chapter_file), title: n.title || `第${rows.length + 1}回` })
        }
        if (n.children?.length) walk(n.children)
      }
    }
    walk(sorted)
    return rows
  }, [outlines])

  const openChapter = useCallback((node: OutlineNode) => {
    if (!active) return
    window.dispatchEvent(new CustomEvent('novel:open-chapter', { detail: { node } }))
  }, [active])

  if (!hasBook) {
    return (
      <div className="sin-reading-empty" data-testid="sin-reading-empty">
        先在书架选择一本书——定稿章节以书目录 chapters/ 文件夹为锚
      </div>
    )
  }

  return (
    <div className="sin-reading" data-testid="sin-reading-pane">
      <nav className="sin-reading-nav" aria-label="章节导航（定稿文件夹锚点）">
        <div className="sin-reading-nav-head">
          <span>定稿章节</span>
          <button
            type="button"
            className="sin-reading-refresh"
            onClick={() => {
              setReloadError('')
              void app.SinEngineListSessions().catch(() => {})
              loadOutlines().catch((err: unknown) => setReloadError(err instanceof Error ? err.message : String(err)))
            }}
            title="重拉章节清单"
          >
            刷新
          </button>
        </div>
        <div className="sin-reading-nav-list">
          {chapters.map((row) => (
            <button
              key={row.node.id}
              type="button"
              className="sin-reading-nav-item"
              onClick={() => openChapter(row.node)}
              title={`${row.node.chapter_file} · 点击在阅读区打开`}
            >
              <span className="sin-reading-nav-num">第{row.num}回</span>
              <span className="sin-reading-nav-title">{row.title}</span>
            </button>
          ))}
          {chapters.length === 0 && (
            <div className="sin-reading-nav-empty">
              {reloadError ? `章节清单读取失败：${reloadError}` : '本书还没有定稿章节——对话里用 @@章节|标题@@ 标记创作，回合结束自动落盘'}
            </div>
          )}
        </div>
      </nav>
      <div className="sin-reading-body">
        <ChapterPage active={active} />
      </div>
    </div>
  )
}

export default SinReadingPane
