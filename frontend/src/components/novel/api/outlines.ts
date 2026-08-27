/**
 * 大纲数据 API
 * 封装所有大纲相关的后端调用，消除页面间重复代码
 */

import type { OutlineNode } from '../../../types'

export interface OutlineData {
  nodes: OutlineNode[]
  story_thread?: string
}

/** 加载大纲列表和故事主线 */
export async function loadOutlines(): Promise<OutlineData | null> {
  try {
    const data = await window.go.app.App.GetOutlines()
    return data || null
  } catch (err) {
    console.error('[API] loadOutlines failed:', err)
    return null
  }
}
