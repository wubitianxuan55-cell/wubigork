// ChatPage 拆分产物：消息流列表（行为零变化，T6-10.1）。
// 纯展示组件：渲染 user/assistant 气泡、流式光标、推理块、错误块与复制/朗读操作。
//
// T-perf 性能优化（保持对外 props 与渲染契约不变）：
//  1) 行级 memo：行渲染抽到 ChatRow，此处施加 React.memo 浅比较边界；配合
//     「稳定回调桥」（latest-ref）把 ChatPage 传入的 onCopy/onSpeak/onRetry
//     转为恒稳引用——流式 chunk 期间仅流式行重渲染，其余行全部命中缓存。
//  2) 尾部窗口：消息数超过 VIRTUALIZE_THRESHOLD 后仅渲染最近 WINDOW_INITIAL 条
//     （真实约束 DOM 节点规模）；用户翻到窗口顶部附近时按 WINDOW_GROW_STEP 向上
//     扩载更早消息（渐进式历史加载，Chromium 原生滚动锚定补偿上方插入的高度差）。
//
// 采用尾部窗口而非 react-window 的原因：react-window v2 的 List 根节点必须自身
// 是滚动容器（行绝对定位由其内部 scroll 事件驱动），而本页滚动容器是 ChatPage
// 的宿主 div（listRef / onScroll / 吸底跟随都挂在它上面，且该文件不在本次改动
// 白名单内）；此外聊天行高随 Markdown 内容动态变化，虚拟化的估算/测量抖动会
// 集中落在流式吸底这一最敏感路径上。尾部窗口保持滚动容器与 DOM 结构完全不变，
// 流式尾巴（最后一条消息）恒在渲染窗口内，吸底行为零改动。
import React, { useCallback, useEffect, useRef, useState } from 'react'
import { ChatRow } from './ChatRow'
import type { ChatMsg } from '../../pages/chat/types'

export interface MessageListProps {
  messages: ChatMsg[]
  streamKey: string | null
  streamText: string
  mode: string
  companionName: string
  copiedId: string | null
  speakingId: string | null
  onCopy: (content: string, id: string) => void
  onSpeak: (content: string, id: string) => void
  onRetry: (msgKey: string) => void
}

/** 消息数超过该阈值后启用尾部窗口渲染（≤ 阈值全量渲染，行为与原实现一致）。 */
export const VIRTUALIZE_THRESHOLD = 50
/** 尾部窗口初始大小：仅渲染最近 N 条消息 */
export const WINDOW_INITIAL = 80
/** 每次触发向上扩载时新增渲染的历史条数 */
export const WINDOW_GROW_STEP = 60
/** 滚动容器距顶部多少 px 内触发向上扩载 */
export const TOP_GROW_PX = 120

/** memo 边界：props 为原始值 + msg 对象引用 + 恒稳回调，默认浅比较即可生效。 */
const MemoChatRow = React.memo(ChatRow)

export const MessageList: React.FC<MessageListProps> = ({
  messages, streamKey, streamText, mode, companionName,
  copiedId, speakingId, onCopy, onSpeak, onRetry,
}) => {
  // ── 稳定回调桥：ChatPage 侧 handleCopy/handleSpeak 每次渲染都是新引用、
  // handleRetry 依赖 messages 数组，直接透传会让行 memo 全部失效。经 latest-ref
  // 转发为恒稳引用（useCallback 空依赖），行内调用时再取最新实现。
  const cbsRef = useRef({ onCopy, onSpeak, onRetry })
  useEffect(() => {
    cbsRef.current = { onCopy, onSpeak, onRetry }
  })
  const stableOnCopy = useCallback((content: string, id: string) => { cbsRef.current.onCopy(content, id) }, [])
  const stableOnSpeak = useCallback((content: string, id: string) => { cbsRef.current.onSpeak(content, id) }, [])
  const stableOnRetry = useCallback((msgKey: string) => { cbsRef.current.onRetry(msgKey) }, [])

  // ── 尾部窗口状态：以消息集合首键（anchor）对齐——集合被整体替换（切话题/
  // 清空/重试移除首条）时重置窗口，避免旧会话扩载出的窗口尺寸泄漏到新会话。
  const [win, setWin] = useState<{ anchor: string; size: number }>(
    () => ({ anchor: messages[0]?.key ?? '', size: WINDOW_INITIAL }))
  const firstKey = messages[0]?.key ?? ''
  if (win.anchor !== firstKey) {
    // 渲染期同步修正（React 官方「props 变化时重置 state」模式）：立即以新
    // anchor 重渲染，首帧即按新窗口渲染，不闪烁。
    setWin({ anchor: firstKey, size: WINDOW_INITIAL })
  }

  const virtualized = messages.length > VIRTUALIZE_THRESHOLD
  const shown = virtualized
    ? messages.slice(messages.length - Math.min(win.size, messages.length))
    : messages
  // shown 首行在完整消息数组中的下标：newGroup 需要对照窗口外的相邻行，
  // 保证「AI 名牌」分组判断与全量渲染完全一致。
  const startIdx = messages.length - shown.length

  // ── 向上扩载：滚动宿主是 ChatPage 的 listRef 容器（本组件根节点的父级）。
  // 仅超阈值时挂监听（≤ 阈值零开销）；用户接近顶部时扩大渲染窗口。jsdom 无
  // 布局（scrollTop 恒 0），测试经手动派发 scroll 事件驱动同一扩载路径。
  const flowRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (!virtualized) return
    const host = flowRef.current?.parentElement
    if (!host) return
    const onHostScroll = () => {
      if (host.scrollTop > TOP_GROW_PX) return
      setWin(prev => prev.size >= messages.length
        ? prev
        : { ...prev, size: Math.min(prev.size + WINDOW_GROW_STEP, messages.length) })
    }
    host.addEventListener('scroll', onHostScroll, { passive: true })
    return () => host.removeEventListener('scroll', onHostScroll)
  }, [virtualized, messages])

  return (
    <div ref={flowRef} className="chat-flow v3-reading">
      {shown.map((msg, idx) => {
        const isStreaming = !!msg.streaming && msg.key === streamKey
        const prev = messages[startIdx + idx - 1]
        return (
          <MemoChatRow
            key={msg.key}
            msg={msg}
            text={isStreaming ? streamText : msg.content}
            isStreaming={isStreaming}
            newGroup={!prev || prev.role !== msg.role}
            mode={mode}
            companionName={companionName}
            copied={copiedId === msg.key}
            speaking={speakingId === msg.key}
            onCopy={stableOnCopy}
            onSpeak={stableOnSpeak}
            onRetry={stableOnRetry}
          />
        )
      })}
    </div>
  )
}
