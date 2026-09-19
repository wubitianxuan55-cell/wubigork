/**
 * outlineStore — 大纲数据共享层（zustand slice）
 * 所有页面共享同一份大纲数据，写操作后自动刷新
 */
import { create } from 'zustand'
import type { OutlineNode } from '../types'
import { sortNodes } from '../utils/outline'
import { loadOutlines as fetchOutlines } from '../components/novel/api/outlines'

/** 大纲刷新结果（后端 ApplyBranch/续写等返回的动态载荷最小消费面） */
interface RefreshOutlinesResult {
  outlines?: { nodes?: OutlineNode[] }
  nodes?: OutlineNode[]
}

interface OutlineState {
  outlines: OutlineNode[]
  storyThread: string
  loading: boolean
  /** T7-4：加载三态 {data, loading, error} 的 error 档；null 表示无错误 */
  error: string | null

  /** 从后端加载大纲 + 故事主线（失败置 error，可再次调用重试） */
  loadOutlines: () => Promise<void>
  /** 用 API 返回结果刷新大纲（用于 ApplyBranch 等操作后） */
  refreshOutlines: (result: RefreshOutlinesResult) => void
  /** 直接设置大纲数组 */
  setOutlines: (nodes: OutlineNode[]) => void
  /** 设置故事主线 */
  setStoryThread: (value: string) => void
}

// v4.354：载入序号（模块级）——快速换书时旧书 fetchOutlines 慢响应覆盖新书
// 大纲树，用户按 A 树章号读写 B 书章节（错误正文/写错章）
let loadSeq = 0

export const useOutlineStore = create<OutlineState>((set) => ({
  outlines: [],
  storyThread: '',
  loading: false,
  error: null,

  loadOutlines: async () => {
    const seq = ++loadSeq
    set({ loading: true, error: null })
    try {
      const data = await fetchOutlines()
      if (seq !== loadSeq) return
      // fetchOutlines 内部吞错并返回 null：null 同样视为加载失败（否则
      // 前端会呈现“加载完成但大纲为空”的假空态，无法区分失败与真空）。
      if (!data) {
        set({ loading: false, error: '加载大纲失败：后端未返回数据' })
        return
      }
      if (data.nodes) set({ outlines: sortNodes(data.nodes) })
      if (data.story_thread !== undefined) set({ storyThread: data.story_thread })
      set({ loading: false, error: null })
    } catch (err) {
      set({ loading: false, error: err instanceof Error ? err.message : String(err) })
    }
  },

  refreshOutlines: (result: RefreshOutlinesResult) => {
    if (result?.outlines?.nodes) {
      set({ outlines: sortNodes(result.outlines.nodes) })
    } else if (result?.nodes) {
      set({ outlines: sortNodes(result.nodes) })
    }
  },

  setOutlines: (nodes: OutlineNode[]) => set({ outlines: nodes }),
  setStoryThread: (value: string) => set({ storyThread: value }),
}))
