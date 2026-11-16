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
//  3) 绘制层虚拟化（v4.383，v4.365 池②收口的第二形态）：超阈值会话根节点挂
//     chat-flow-cv，行级 content-visibility:auto——离屏行跳过 layout/paint，
//     窗口扩载到全量历史后滚动依旧顺滑（React 渲染层约束由窗口管，绘制成本
//     由浏览器原生管，两层互不干扰）。
//
// 采用尾部窗口而非 react-window 的原因：react-window v2 的 List 根节点必须自身
// 是滚动容器（行绝对定位由其内部 scroll 事件驱动），而本页滚动容器是 ChatPage
// 的宿主 div（listRef / onScroll / 吸底跟随都挂在它上面，且该文件不在本次改动
// 白名单内）；此外聊天行高随 Markdown 内容动态变化，虚拟化的估算/测量抖动会
// 集中落在流式吸底这一最敏感路径上。尾部窗口保持滚动容器与 DOM 结构完全不变，
// 流式尾巴（最后一条消息）恒在渲染窗口内，吸底行为零改动。
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { ChatRow } from './ChatRow'
import { buildVisiblePath, buildVersions } from '../../pages/chat/branches'
import type { ChatSearchReference } from './searchCardModel'

import type { ChatMsg } from '../../pages/chat/types'

export interface MessageListProps {
  messages: ChatMsg[]
  /** v4.370 历史分页：是否还有更早历史（true 且窗口到达数据头时触发 onLoadOlder） */
  hasOlder?: boolean
  loadingOlder?: boolean
  onLoadOlder?: () => void
  /** 头部追加通知：{id 递增, count 新增条数}——区分「分页追加」与「切话题整体替换」 */
  prepend?: { id: number; count: number } | null
  streamKey: string | null
  streamText: string
  /** 思考流实时文本（reasoning 帧节流flush；流式行专属） */
  streamReasoning?: string
  streamStatus?: string
  mode: string
  companionName: string
  copiedId: string | null
  speakingId: string | null
  onCopy: (content: string, id: string) => void
  onSpeak: (content: string, id: string) => void
  onRetry: (msgKey: string) => void
  /** A 批分支树：切换到兄弟版本（msg db id） */
  onSelectVersion?: (msgId: number) => void
  /** A 批编辑提问：提交某 db 用户行的新文本 */
  onEditSubmit?: (msgKey: string, text: string) => void
  /** A 批重新生成：某 db 回答行下挂新版本 */
  onRegenerate?: (msgKey: string) => void
  /** A 批分支流锚点：编辑/重生成的乐观行插入该 db 行 key 之后并截断旧链 */
  branchAnchorKey?: string | null
  /** D 批：打开全量搜索来源面板（SearchCard「查看全部来源」入口） */
  onOpenSources?: (refs: ChatSearchReference[], queries: string[]) => void
  /** A 批续写：对已停止回答行发起 prefill 续写 */
  onContinue?: (msgKey: string) => void
  /** A 批续写占线（续写流期间行级 busy，与 streamKey 并列） */
  extraBusy?: boolean
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
  messages, streamKey, streamText, streamReasoning, streamStatus, mode, companionName,
  copiedId, speakingId, onCopy, onSpeak, onRetry,
  onSelectVersion, onEditSubmit, onRegenerate, branchAnchorKey, onOpenSources,
  onContinue, extraBusy,
  hasOlder, loadingOlder, onLoadOlder, prepend,
}) => {
  // ── A 批分支树：可见路径（叶子回溯 + 分支锚点插入）与兄弟版本表。
  // 线形话题（无 bt）两表皆直通，渲染契约零变化。
  const visible = useMemo(
    () => buildVisiblePath(messages, branchAnchorKey),
    [messages, branchAnchorKey],
  )
  const versions = useMemo(() => buildVersions(messages), [messages])
  // ── 稳定回调桥：ChatPage 侧 handleCopy/handleSpeak 每次渲染都是新引用、
  // handleRetry 依赖 messages 数组，直接透传会让行 memo 全部失效。经 latest-ref
  // 转发为恒稳引用（useCallback 空依赖），行内调用时再取最新实现。
  const cbsRef = useRef({ onCopy, onSpeak, onRetry, onSelectVersion, onEditSubmit, onRegenerate, onOpenSources, onContinue })
  useEffect(() => {
    cbsRef.current = { onCopy, onSpeak, onRetry, onSelectVersion, onEditSubmit, onRegenerate, onOpenSources, onContinue }
  })
  const stableOnCopy = useCallback((content: string, id: string) => { cbsRef.current.onCopy(content, id) }, [])
  const stableOnSpeak = useCallback((content: string, id: string) => { cbsRef.current.onSpeak(content, id) }, [])
  const stableOnRetry = useCallback((msgKey: string) => { cbsRef.current.onRetry(msgKey) }, [])
  const stableOnSelectVersion = useCallback((msgId: number) => { cbsRef.current.onSelectVersion?.(msgId) }, [])
  const stableOnEditSubmit = useCallback((msgKey: string, text: string) => { cbsRef.current.onEditSubmit?.(msgKey, text) }, [])
  const stableOnRegenerate = useCallback((msgKey: string) => { cbsRef.current.onRegenerate?.(msgKey) }, [])
  const stableOnOpenSources = useCallback((refs: ChatSearchReference[], queries: string[]) => { cbsRef.current.onOpenSources?.(refs, queries) }, [])
  const stableOnContinue = useCallback((msgKey: string) => { cbsRef.current.onContinue?.(msgKey) }, [])

  // ── 尾部窗口状态：以消息集合首键（anchor）对齐——集合被整体替换（切话题/
  // 清空/重试移除首条）时重置窗口，避免旧会话扩载出的窗口尺寸泄漏到新会话。
  const [win, setWin] = useState<{ anchor: string; size: number }>(
    () => ({ anchor: messages[0]?.key ?? '', size: WINDOW_INITIAL }))
  const winRef = useRef(win)
  winRef.current = win
  const firstKey = visible[0]?.key ?? ''
  // v4.370：头部追加（分页加载更早历史）时窗口尺寸随新增条数扩大而非重置，
  // 用户正在顶部附近翻阅不被弹回尾部；prepSeq 标记已处理的追加通知。
  const prepSeqRef = useRef(0)
  let prependHandled = false
  if (win.anchor !== firstKey) {
    if (prepend && prepend.id !== prepSeqRef.current) {
      // 渲染期同步修正（React 官方「props 变化时重置 state」模式）
      prepSeqRef.current = prepend.id
      prependHandled = true
      setWin({ anchor: firstKey, size: Math.min(win.size + prepend.count, visible.length) })
    } else {
      // 渲染期同步修正（React 官方「props 变化时重置 state」模式）：立即以新
      // anchor 重渲染，首帧即按新窗口渲染，不闪烁。
      setWin({ anchor: firstKey, size: WINDOW_INITIAL })
    }
  }
  if (prependHandled) prependHandled = false

  const virtualized = visible.length > VIRTUALIZE_THRESHOLD
  const shown = virtualized
    ? visible.slice(visible.length - Math.min(win.size, visible.length))
    : visible
  // shown 首行在可见路径数组中的下标：newGroup 需要对照窗口外的相邻行，
  // 保证「AI 名牌」分组判断与全量渲染完全一致。
  const startIdx = visible.length - shown.length

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
      // v4.370：窗口已覆盖全部已加载消息（到达数据头）且仍有更早历史 → 分页拉取
      if (winRef.current.size >= visible.length) {
        if (hasOlder && !loadingOlder && onLoadOlder) onLoadOlder()
        return
      }
      setWin(prev => prev.size >= visible.length
        ? prev
        : { ...prev, size: Math.min(prev.size + WINDOW_GROW_STEP, visible.length) })
    }
    host.addEventListener('scroll', onHostScroll, { passive: true })
    return () => host.removeEventListener('scroll', onHostScroll)
  }, [virtualized, visible, hasOlder, loadingOlder, onLoadOlder])

  return (
    <div ref={flowRef} className={`chat-flow v3-reading${virtualized ? ' chat-flow-cv' : ''}`}>
      {shown.map((msg, idx) => {
        const isStreaming = !!msg.streaming && msg.key === streamKey
        const busy = streamKey !== null || !!extraBusy
        const prev = visible[startIdx + idx - 1]
        return (
          <MemoChatRow
            key={msg.key}
            msg={msg}
            text={isStreaming ? streamText : msg.content}
            streamStatus={isStreaming ? streamStatus : ''}
            streamReasoning={isStreaming ? streamReasoning : undefined}
            isStreaming={isStreaming}
            newGroup={!prev || prev.role !== msg.role}
            mode={mode}
            companionName={companionName}
            copied={copiedId === msg.key}
            speaking={speakingId === msg.key}
            versions={versions.get(msg.key) ?? null}
            onSelectVersion={stableOnSelectVersion}
            onEditSubmit={stableOnEditSubmit}
            onRegenerate={stableOnRegenerate}
            onOpenSources={stableOnOpenSources}
            onContinue={stableOnContinue}
            busy={busy}
            onCopy={stableOnCopy}
            onSpeak={stableOnSpeak}
            onRetry={stableOnRetry}
          />
        )
      })}
    </div>
  )
}
