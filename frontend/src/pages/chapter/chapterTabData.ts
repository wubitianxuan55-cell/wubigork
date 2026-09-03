/**
 * chapterTabData — 章节 Tab 数据的构造与关闭守卫纯函数（自 ChapterPage 原样搬移）：
 * - createTabData：新建 Tab 时的空白数据骨架；
 * - needsCloseConfirm：关闭前是否弹确认（有未保存修改）。
 */
import type { OutlineNode, ChapterTabData } from '../../types'

/** 创建空的 ChapterTabData */
export function createTabData(node: OutlineNode): ChapterTabData {
  return {
    node, chapterNum: node.order_index || 0,
    scenes: [''], summary: '', keyEvents: [],
    emotionTone: '', saved: false, generating: false,
    streamSpeed: 0, messages: [], targetWords: 3000,
    skillName: '', retryStatus: null,
  }
}

/** 关闭前是否需要确认：非 saved 且任一场景有内容（未保存修改会丢） */
export function needsCloseConfirm(tab: ChapterTabData | null | undefined): boolean {
  return !!tab && !tab.saved && tab.scenes.some((s) => s.trim())
}
