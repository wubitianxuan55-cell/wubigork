import React, { useEffect, useRef } from 'react'
import { Drawer, Typography } from 'antd'
import type { SrcRef } from './ParseSummaryCards'

const { Text } = Typography

export const SourcePreviewDrawer: React.FC<{
  open: boolean
  files?: { name?: string; markdown?: string }[]
  source?: SrcRef | null
  onClose: () => void
}> = ({ open, files, source, onClose }) => {
  const bodyRef = useRef<HTMLDivElement>(null)
  const file = (files || []).find(f => f.name === source?.fileName) || (files || [])[0]
  useEffect(() => {
    if (!open || !bodyRef.current || !source?.snippet) return
    const idx = file?.markdown?.indexOf(source.snippet)
    if (idx != null && idx >= 0 && bodyRef.current.firstChild) {
      const walker = document.createTreeWalker(bodyRef.current, NodeFilter.SHOW_TEXT)
      let node: Node | null
      let offset = 0
      while ((node = walker.nextNode())) {
        const len = node.textContent?.length || 0
        if (offset + len >= idx) {
          const range = document.createRange()
          range.setStart(node, idx - offset)
          range.setEnd(node, Math.min(idx - offset + (source.snippet?.length || 0), len))
          const sel = window.getSelection()
          sel?.removeAllRanges()
          sel?.addRange(range)
          node.parentElement?.scrollIntoView({ block: 'center' })
          break
        }
        offset += len
      }
    }
  }, [open, source, file])
  return (
    <Drawer title="招标文件原文" width={560} open={open} onClose={onClose}>
      <Text type="secondary" style={{ fontSize: 12 }}>{file?.name || '未选择文件'}</Text>
      <div ref={bodyRef} style={{ marginTop: 8, whiteSpace: 'pre-wrap', fontSize: 13, lineHeight: 1.8 }}>
        {file?.markdown || '（无原文内容）'}
      </div>
    </Drawer>
  )
}
