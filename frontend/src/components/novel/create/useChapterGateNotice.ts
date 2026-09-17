// useChapterGateNotice.ts — 生成后自动门事件通知（v4.331，规格
// 进度计划/gaea-gate-notice-race-20260917.md）：订阅 chapter-gate 通道
// （v4.326 后端 emit，此前零前端消费），把异步自动门结果轻通知给作者
// （antd message.info，key 防叠，6s 自动消失，零弹窗打扰）。
// 通道模式同 useChapterStream（window.runtime.EventsOn/EventsOff——浏览器
// mock 无通道静默跳过）。
import { useEffect } from 'react'
import { message } from 'antd'

export const CHAPTER_GATE_CHANNEL = 'chapter-gate'

/** chapter-gate report 载荷窄化（Go runAutoGateAfterGeneration 的 map）。 */
interface ChapterGateReport {
  type?: string
  chapterNum?: number
  branch?: string
  outlineIssues?: number
  qualityIssues?: number
  aiTasteScore?: number | null
  analysisDone?: boolean
}

/** 体检结果拼装（纯函数供测试）：空体检显示「完成」。 */
export function gateNoticeText(r: ChapterGateReport): string {
  const label = `第 ${r.chapterNum ?? '?'} 章${r.branch ? r.branch : ''}`
  const parts = [
    (r.outlineIssues ?? 0) > 0 ? `契约 ${r.outlineIssues} 项` : '',
    (r.qualityIssues ?? 0) > 0 ? `质量 ${r.qualityIssues} 项` : '',
    r.aiTasteScore != null ? `AI 味 ${r.aiTasteScore}` : '',
    r.analysisDone ? '分析已同步' : '',
  ].filter(Boolean)
  return parts.length > 0 ? `${label} 生成后自动体检：${parts.join(' · ')}` : `${label} 生成后自动体检完成`
}

/** 订阅 chapter-gate：组件挂载期监听，卸载退订（EventsOff 只清本通道）。 */
export function useChapterGateNotice(): void {
  useEffect(() => {
    const handler = (payload: unknown) => {
      // 兼容 CustomEvent 包装（event.detail）与 Wails 直传负载
      const raw = (payload as { detail?: unknown } | null)?.detail ?? payload
      const r = raw as ChapterGateReport | null
      if (!r || r.type !== 'report') return
      message.info({ key: 'chapter-gate', content: gateNoticeText(r), duration: 6 })
    }
    try {
      window.runtime?.EventsOn?.(CHAPTER_GATE_CHANNEL, handler as (data: unknown) => void)
    } catch {
      // 非 Wails 环境无通道（浏览器 mock）：静默跳过
      return
    }
    return () => {
      try {
        window.runtime?.EventsOff?.(CHAPTER_GATE_CHANNEL)
      } catch {
        // 退订失败无害（通道不存在/已卸载）
      }
    }
  }, [])
}
