// useChapterGateNotice.ts — 生成后自动门事件通知（v4.331，规格
// 进度计划/gaea-gate-notice-race-20260917.md）：订阅 chapter-gate 通道
// （v4.326 后端 emit，此前零前端消费），把异步自动门结果轻通知给作者
// （antd message.info，key 防叠，6s 自动消失，零弹窗打扰）。
// 订阅/退订统一走 gaea/lib/wailsEvents.subscribeWailsEvent（v4.421.0 起：此前
// 卸载时裸 EventsOff(CHAPTER_GATE_CHANNEL) 会把同通道别人的监听一起注销，
// 违反该模块开篇记的事故纪律）；浏览器 mock 无通道静默跳过。
// v4.336：通知可点击——onOpen(chapterNum) 跳章节分析面板（观察池头名收口）。
import { useEffect, useRef } from 'react'
import { message } from 'antd'
import { subscribeWailsEvent } from '../../../gaea/lib/wailsEvents'

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

/** 订阅 chapter-gate：组件挂载期监听，卸载退订（只摘本 hook 注册的那一个
 *  监听者，同通道其他消费者不受影响——subscribeWailsEvent 纪律）。
 *  onOpen：点击通知回调（传报告章号；无回调=纯通知不跳转，既有行为零变化）。 */
export function useChapterGateNotice(onOpen?: (chapterNum: number) => void): void {
  // ref 保最新回调（防过期闭包——[] 依赖订阅一次，回调可能引用异步态）
  const onOpenRef = useRef(onOpen)
  onOpenRef.current = onOpen
  useEffect(() => {
    const rt = window.runtime
    if (!rt?.EventsOn) return // 非 Wails 环境无通道（浏览器 mock）：静默跳过
    const handler = (payload: unknown) => {
      // 兼容 CustomEvent 包装（event.detail）与 Wails 直传负载
      const raw = (payload as { detail?: unknown } | null)?.detail ?? payload
      const r = raw as ChapterGateReport | null
      if (!r || r.type !== 'report') return
      message.info({
        key: 'chapter-gate',
        content: gateNoticeText(r),
        duration: 6,
        onClick: () => {
          if (r.chapterNum && r.chapterNum > 0) onOpenRef.current?.(r.chapterNum)
        },
      })
    }
    try {
      return subscribeWailsEvent(rt, CHAPTER_GATE_CHANNEL, handler)
    } catch {
      // 订阅失败：无监听可退（不抛给 React 渲染链）
      return undefined
    }
  }, [])
}
