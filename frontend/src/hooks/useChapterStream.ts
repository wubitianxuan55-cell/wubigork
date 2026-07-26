import { useEffect } from 'react'
import { message } from 'antd'
import type { ChapterTabData, OutlineNode } from '../types'

/**
 * useChapterStream — 监听 Wails 后端流式生成事件，自动更新标签页状态
 * 提取自 ChapterPage，职责单一：管理 chapter-stream 事件生命周期
 */
export function useChapterStream(
  tabs: ChapterTabData[],
  setTabs: React.Dispatch<React.SetStateAction<ChapterTabData[]>>,
  activeKeyRef: React.MutableRefObject<string>,
  startTime: React.MutableRefObject<number>,
  modalTimerRef: React.MutableRefObject<number>,
  lastCompletedNode: React.MutableRefObject<OutlineNode | null>,
) {
  useEffect(() => {
    // @ts-ignore
    if (!window.runtime?.EventsOn) return
    // @ts-ignore
    window.runtime.EventsOn('chapter-stream', (ev: any) => {
      if (!ev?.type) return
      setTabs((prev) => {
        const key = activeKeyRef.current
        const idx = prev.findIndex((t) => t.node.id === key)
        if (idx < 0) return prev
        const copy = [...prev]
        const tab = { ...copy[idx] }
        copy[idx] = tab

        if (ev.type === 'chunk') {
          const retryMatch = ev.content?.match(/\[AI 正在根据审稿意见重写.*?评分 (\d+)\/10.*?目标.*?(\d+)\]/)
          if (retryMatch) {
            tab.retryStatus = { score: parseInt(retryMatch[1]), target: parseInt(retryMatch[2]) }
            return copy
          }
          tab.retryStatus = null
          const s = [...tab.scenes]
          s[s.length - 1] = (s[s.length - 1] || '') + ev.content
          tab.scenes = s
          if (startTime.current > 0) {
            const elapsed = (Date.now() - startTime.current) / 1000
            tab.streamSpeed = elapsed > 0 ? Math.round((ev.total || 0) / elapsed) : 0
          }
        } else if (ev.type === 'error') {
          tab.generating = false
          const s = [...tab.scenes]
          s[s.length - 1] = (s[s.length - 1] || '') + `\n\n❌ ${ev.error}`
          tab.scenes = s
        } else if (ev.type === 'done') {
          tab.generating = false
          tab.streamSpeed = 0
          if (ev.content) tab.scenes = [ev.content]
          if (ev.summary) {
            tab.summary = ev.summary.summary || ''
            tab.keyEvents = ev.summary.key_events || []
            tab.emotionTone = ev.summary.emotion_tone || ''
          }
          tab.saved = true
          message.success('已自动保存')
          if (tab.node) lastCompletedNode.current = tab.node
        }
        return copy
      })
    })
    return () => {
      try {
        // @ts-ignore
        window.runtime?.EventsOff?.('chapter-stream')
      } catch (e) { /* ignore */ }
      if (modalTimerRef.current) clearTimeout(modalTimerRef.current)
    }
  }, [])
}
