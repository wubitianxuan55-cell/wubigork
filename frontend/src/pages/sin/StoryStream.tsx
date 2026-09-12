// sin/StoryStream.tsx — 故事流视图（图文混杂的排版层）。
//
// 复用面：正文本段交办公 Markdown 渲染器（frontend/src/gaea/components/Markdown，
// 与办公消息同一条渲染链：本地文件链接 / 代码 / mermaid / genui 全继承）；
// 插图段交 SinIllustration（绘梦后端出图）。本组件只负责分段与排布。

import { forwardRef } from 'react'
import { Markdown } from '../../gaea/components/Markdown'
import { EmptyState } from '../../gaea/components/EmptyState'
import { SinIllustration } from './SinIllustration'
import { SinProcessCard } from './SinProcessCard'
import { parseStorySegments } from './storyText'
import type { SinMessageView } from './types'

export interface StoryStreamProps {
  storyId: string
  messages: SinMessageView[]
  onIllustrationGenerated: (messageKey: string, cueKey: string, path: string) => void
  onIllustrationError: (message: string) => void
  /** 滚动容器滚动回调（父层据此判断是否贴底跟随）。 */
  onScroll?: () => void
  /** 空态引导文案（首入时提示怎么开写）。 */
  emptyHint?: string
}

export const StoryStream = forwardRef<HTMLDivElement, StoryStreamProps>(function StoryStream(
  { storyId, messages, onIllustrationGenerated, onIllustrationError, onScroll, emptyHint }, ref,
) {
  if (messages.length === 0) {
    return (
      <div className="sin-stream" ref={ref} onScroll={onScroll}>
        <EmptyState
          message={emptyHint ?? '还没有开始写 · 在下面说一句：想要什么设定、什么人物、从哪一幕开始——原罪会边写故事边配插图。'}
        />
      </div>
    )
  }
  return (
    <div className="sin-stream" ref={ref} onScroll={onScroll}>
      {messages.map((m) => {
        if (m.role === 'user') {
          return (
            <div className="sin-row is-user" key={m.key}>
              <div className="sin-bubble-user">{m.content}</div>
            </div>
          )
        }
        const segments = parseStorySegments(m.content)
        return (
          <div className={`sin-row is-assistant${m.error ? ' is-error' : ''}`} key={m.key}>
            <div className="sin-assistant-card">
              {/* 过程先于结果：思考过程与工具调用在正文之上（无内容时自行不渲染） */}
              <SinProcessCard reasoning={m.reasoning} tools={m.tools ?? []} running={!!m.streaming} />
              {segments.map((seg, i) => (
                seg.kind === 'text' ? (
                  <div className="sin-text" key={`t${i}`}>
                    <Markdown text={seg.text} />
                  </div>
                ) : (
                  <SinIllustration
                    key={`${m.key}_art_${seg.cueKey}`}
                    storyId={storyId}
                    messageId={m.messageId}
                    cueKey={seg.cueKey}
                    prompt={seg.prompt}
                    path={m.illustrations[seg.cueKey]}
                    ready={!m.streaming}
                    onGenerated={(cueKey, path) => onIllustrationGenerated(m.key, cueKey, path)}
                    onError={onIllustrationError}
                  />
                )
              ))}
              {m.streaming && <span className="sin-caret" aria-label="正在续写" />}
            </div>
          </div>
        )
      })}
    </div>
  )
})
