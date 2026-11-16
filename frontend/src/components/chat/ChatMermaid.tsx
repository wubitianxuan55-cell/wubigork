import { memo, useEffect, useRef, useState } from 'react'
import { ensureMermaid, mermaidTheme } from '../../gaea/lib/mermaidRender'

// ChatMermaid — 聊天线 mermaid 图表（C3，移植自 deepseek-ai/chat-demo
// MermaidDiagram.tsx 的渐进容错配方，MIT；蒸馏台账
// docs/gaea-deepseek-chatdemo-distill-2026-11.md §2.4）。与办公线 MermaidBlock
// 分工：本组件刻意轻量（无导出/缩放），核心是流式容错三招——
//   ① 流式期间从 ref 取最新文本 + 100ms 合帧绘制，不逐 chunk 取消重画；
//   ② 片段语法不完整时只画「完整行前缀」，失败保留上次有效图；
//   ③ 渲染错误仅在流结束后显示（流式中的半截语法不当失败）。
// 模块加载/initialize 走 lib/mermaidRender 单源（与办公线共享缓存）。

let nextId = 0

export const ChatMermaid = memo(function ChatMermaid({
  code,
  loading = false,
}: {
  code: string
  /** 流式尾部半截语法容错开关；done 态渲染 false 即可 */
  loading?: boolean
}) {
  const [svg, setSvg] = useState('')
  const [failed, setFailed] = useState(false)
  const hostRef = useRef<HTMLDivElement>(null)
  // 流式阶段从 ref 取最新文本，避免每个片段都取消绘制；结束后立即绘制完整内容。
  const latestCodeRef = useRef(code)
  latestCodeRef.current = code
  const finalCode = loading ? null : code

  useEffect(() => {
    let cancelled = false
    let timer: ReturnType<typeof setTimeout> | undefined
    let displayedCode: string | undefined

    const draw = async (input: string) => {
      if (input === displayedCode || cancelled) {
        return
      }
      const mm = await ensureMermaid(mermaidTheme())
      if (cancelled) {
        return
      }
      const { svg: out } = await mm.render(`chat-mermaid-${nextId++}`, input)
      if (cancelled || !out) {
        return
      }
      displayedCode = input
      setSvg(out)
      setFailed(false)
    }

    const renderLatest = async () => {
      const input = latestCodeRef.current
      if (input !== displayedCode) {
        try {
          await draw(input)
        } catch {
          if (!cancelled) {
            if (loading) {
              // 流式片段的末行可能包含未完成的节点，仅绘制前面的完整行。
              const text = input.trimEnd()
              const lineEnd = Math.max(text.lastIndexOf('\n'), text.lastIndexOf('\r'))
              const prefix = text.slice(0, Math.max(0, lineEnd))
              // 已显示的内容覆盖当前前缀时保持图形。
              if (prefix && !displayedCode?.startsWith(prefix)) {
                try {
                  await draw(prefix)
                } catch {
                  // 前缀也可能尚未构成完整语法，保留已有图。
                }
              }
            } else {
              setSvg('')
              setFailed(true)
            }
          }
        }
      }
      // 当前帧结束后绘制最新片段，连续输入合并为一次绘制。
      if (!cancelled && loading) {
        timer = setTimeout(renderLatest, 100)
      }
    }

    void renderLatest()
    return () => {
      cancelled = true
      clearTimeout(timer)
    }
  }, [loading, finalCode])

  // 渲染成功后内联收口 SVG 尺寸（同办公线做法：strict 模式已消毒，不再外层过 DOMPurify）。
  useEffect(() => {
    const svgEl = hostRef.current?.querySelector('svg')
    if (svgEl) {
      svgEl.style.background = 'transparent'
      svgEl.style.width = '100%'
      svgEl.style.height = 'auto'
      svgEl.style.maxWidth = 'none'
    }
  }, [svg])

  if (!svg) {
    return (
      <div className="chat-mermaid-status" role="status"
        style={{ padding: '12px', fontSize: 12, color: 'var(--md-sys-color-outline)', textAlign: 'center' }}>
        {failed ? '图表渲染失败，可切换「源码」查看' : loading ? '图表渲染中…' : '图表加载中…'}
      </div>
    )
  }

  return <div ref={hostRef} className="chat-mermaid-diagram" style={{ padding: '8px', overflowX: 'auto' }} dangerouslySetInnerHTML={{ __html: svg }} />
})
