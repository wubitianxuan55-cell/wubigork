/**
 * outlineStore — 大纲数据共享层（zustand slice）
 * 所有页面共享同一份大纲数据，写操作后自动刷新
 */
import { create } from 'zustand'
import type { OutlineNode } from '../types'
import { sortNodes } from '../utils/outline'
import { loadOutlines as fetchOutlines } from '../api/outlines'

interface OutlineState {
  outlines: OutlineNode[]
  storyThread: string
  loading: boolean

  /** 从后端加载大纲 + 故事主线 */
  loadOutlines: () => Promise<void>
  /** 用 API 返回结果刷新大纲（用于 ApplyBranch 等操作后） */
  refreshOutlines: (result: any) => void
  /** 直接设置大纲数组 */
  setOutlines: (nodes: OutlineNode[]) => void
  /** 设置故事主线 */
  setStoryThread: (value: string) => void
}

export const useOutlineStore = create<OutlineState>((set) => ({
  outlines: [],
  storyThread: '',
  loading: false,

  loadOutlines: async () => {
    set({ loading: true })
    const data = await fetchOutlines()
    if (data?.nodes) set({ outlines: sortNodes(data.nodes) })
    if (data?.story_thread !== undefined) set({ storyThread: data.story_thread })
    set({ loading: false })
  },

  refreshOutlines: (result: any) => {
    if (result?.outlines?.nodes) {
      set({ outlines: sortNodes(result.outlines.nodes) })
    } else if (result?.nodes) {
      set({ outlines: sortNodes(result.nodes) })
    }
  },

  setOutlines: (nodes: OutlineNode[]) => set({ outlines: nodes }),
  setStoryThread: (value: string) => set({ storyThread: value }),
}))
