/**
 * useFollowScroll — 聊天滚动跟随（v4.514：v4.512 手势语义移植轻语）
 *
 * 语义与办公板 Transcript（v4.512 流式滚动手势化）对齐：
 *  - stick（跟随）清除权唯一归滚轮上滚手势——旧实现「scroll 事件不贴底
 *    即清除」会把布局扰动（滚动锚定/夹紧产生的程序性 scroll 事件）误判
 *    为用户上翻，流式跟随被误杀（「卡住没输出假象」同类病灶）；
 *  - onScroll 只在贴底时恢复跟随；不贴底只刷新按钮态（atBottom）；
 *  - follow() 供内容变化效应调用：贴底跟随落底，暂停态自愈按钮显隐
 *    （上翻后内容增长不触发 scroll 事件，按钮态需随内容重算）；
 *  - rearm() 供切话题/发送/清空等「程序性回底」场景：恢复跟随（不立即
 *    滚动，随下一次内容变化落底）；
 *  - scrollToBottom()：「回到底部」按钮——恢复跟随并立即落底。
 */
import { useCallback, useRef, useState } from 'react'
import type { WheelEvent as ReactWheelEvent } from 'react'
import { isNearBottom } from './scroll'

export function useFollowScroll<T extends HTMLElement>() {
  const ref = useRef<T>(null)
  const stick = useRef(true)
  const [atBottom, setAtBottom] = useState(true)

  const onScroll = useCallback(() => {
    const el = ref.current
    if (!el) return
    const near = isNearBottom(el.scrollHeight - el.scrollTop - el.clientHeight)
    // 只恢复不清除：布局扰动（程序性 scroll 事件）杀不死跟随；清除权在 onWheel。
    if (near) stick.current = true
    setAtBottom(near)
  }, [])

  // 上滚=显式暂停跟随（流式生成时可上翻读历史）；仅容器真实可滚时清除，
  // 短会话（内容不足一屏）的无效上滚不产生暂停态。
  const onWheel = useCallback((e: ReactWheelEvent<T>) => {
    if (e.deltaY >= 0) return
    const el = ref.current
    if (el && el.scrollHeight - el.clientHeight > 0) stick.current = false
  }, [])

  const follow = useCallback(() => {
    const el = ref.current
    if (!el) return
    if (stick.current) {
      el.scrollTop = el.scrollHeight
      setAtBottom(true)
      return
    }
    // 暂停态按钮自愈：内容变化后重算贴底（无 scroll 事件可依赖）
    setAtBottom(isNearBottom(el.scrollHeight - el.scrollTop - el.clientHeight))
  }, [])

  const rearm = useCallback(() => {
    stick.current = true
    setAtBottom(true)
  }, [])

  const scrollToBottom = useCallback(() => {
    stick.current = true
    const el = ref.current
    if (el) el.scrollTop = el.scrollHeight
    setAtBottom(true)
  }, [])

  return { ref, atBottom, onScroll, onWheel, follow, rearm, scrollToBottom }
}
