/**
 * useChapterStream — 章节生成流式事件订阅 hook（T6-7.2 / T6-7.5）。
 *
 * 旧实现把 EventsOn/EventsOff 与 (data as any) switch 全部内联在 CreatePage 的
 * startGeneration 里，三路收尾（done/error/cancelled）都要手动 EventsOff，
 * 卸载时也无清理保证。本 hook 收敛为 attach/detach 生命周期：
 *  - attach(handlers) 先退订再注册监听，负载经 parseCreateChapterEvent 收敛为
 *    判别联合后按 type 分发（畸形/未知负载忽略，不静默抛错）；
 *  - detach() 退订监听——v4.421.0 起走 subscribeWailsEvent 返回的「只摘除自己」
 *    清理函数（裸 EventsOff(channel) 会连坐同通道别人的监听，见
 *    gaea/lib/wailsEvents.ts 事故纪律）；生成终态与组件卸载共用；
 *  - 组件卸载自动 detach，杜绝悬挂监听。
 */
import { useCallback, useEffect, useRef } from 'react'
import { subscribeWailsEvent } from '../../../gaea/lib/wailsEvents'
import { parseCreateChapterEvent, type CreateChapterStreamEvent } from './chapterStreamTypes'

export const CREATE_CHAPTER_STREAM_CHANNEL = 'create-chapter-stream'

export interface ChapterStreamHandlers {
  /** 收到校验通过后的流式事件（判别联合，按 ev.type 分发） */
  onEvent: (event: CreateChapterStreamEvent) => void
}

/**
 * 订阅/退订 create-chapter-stream 流式事件。
 * 返回 { attach, detach }：attach 注册监听（重复调用会先退订再注册，
 * 兼容旧实现 startGeneration 前的防御性退订）；detach 立即退订本 hook 自己
 * 那一个监听（同通道其他监听者不受影响）。
 */
export function useChapterStream() {
  const detachRef = useRef<(() => void) | null>(null)

  const detach = useCallback(() => {
    if (detachRef.current) {
      detachRef.current()
      detachRef.current = null
    }
  }, [])

  const attach = useCallback((handlers: ChapterStreamHandlers) => {
    detach()
    const rt = window.runtime
    if (!rt?.EventsOn) return // 浏览器 mock 无事件通道：无监听可挂
    const handler = (payload: unknown) => {
      // 兼容 CustomEvent 包装（event.detail）与 Wails 直传负载
      const raw = (payload as { detail?: unknown } | null)?.detail ?? payload
      const event = parseCreateChapterEvent(raw)
      if (event) handlers.onEvent(event)
    }
    try {
      // 退订 = 本模块返回的「只摘除自己」清理函数（绝不 EventsOff 全清该通道）
      detachRef.current = subscribeWailsEvent(rt, CREATE_CHAPTER_STREAM_CHANNEL, handler)
    } catch {
      // 订阅失败不阻塞生成流程（事件通道仅在 Wails 环境存在）
      detachRef.current = null
    }
  }, [detach])

  // 组件卸载自动退订，杜绝悬挂监听
  useEffect(() => () => detach(), [detach])

  return { attach, detach }
}
