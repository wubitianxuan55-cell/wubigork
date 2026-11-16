import { useEffect, useRef, useState } from 'react'

// useThrottledStreamText — 流式 Markdown 自适应节流（C1，整体移植自
// deepseek-ai/chat-demo useThrottleMarkdownRender.ts，MIT；蒸馏台账
// docs/gaea-deepseek-chatdemo-distill-2026-11.md §2.2）。
// 与 ChatRow 既有的 findStableCut 分段渲染互补：分段解决「全量 reparse」，
// 本钩子解决「重尾段高频 reparse」——按上次渲染耗时×2 自适应节流
//（上限 2s），长表格/长代码块流入时降低更新频率，连续 chunk 合并为最新内容。
// enable=false（流结束）时直接返回最新内容，终帧立即落定不延迟。

/** 两次更新的最长间隔 */
const MAX_WAIT = 2000
/** 上次渲染耗时的倍数，过了这段时间才允许下次更新 */
const TIME_CONSUMING_RATE = 2

export function useThrottledStreamText(content: string, enable: boolean): string {
  // 首次渲染从挂载时开始计时
  const startRef = useRef(performance.now())
  const updatableAtRef = useRef(0)
  const [output, setOutput] = useState(content)

  useEffect(() => {
    const now = performance.now()
    updatableAtRef.current = now + Math.min(MAX_WAIT, (now - startRef.current) * TIME_CONSUMING_RATE)
  }, [output])

  useEffect(() => {
    if (!enable) {
      return
    }
    const update = () => {
      startRef.current = performance.now()
      setOutput(content)
    }
    const delay = updatableAtRef.current - performance.now()
    if (delay <= 0) {
      update()
      return
    }
    const timer = setTimeout(update, delay)
    return () => clearTimeout(timer)
  }, [content, enable])

  return enable ? output : content
}
