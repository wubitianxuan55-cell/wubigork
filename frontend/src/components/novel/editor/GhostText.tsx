import React, { useState, useEffect, useRef, useCallback } from 'react'

/**
 * GhostText — 内联 AI 补全组件
 *
 * 用法: 将 getCursorContext 绑定到 textarea/编辑器的光标位置获取函数
 * 当用户停止输入 800ms 后自动请求 AI 补全建议
 * 补全文本以灰色斜体显示在光标后，Tab 接受，Esc 取消
 *
 * Props:
 *   getCursorContext — 返回 { textBeforeCursor: string, textareaElement: HTMLTextAreaElement }
 *   enabled — 是否启用
 *   styleProfile — 可选的风格指导
 */
interface GhostTextProps {
  getCursorContext: () => { textBeforeCursor: string; textareaElement: HTMLTextAreaElement | null } | null
  enabled: boolean
  styleProfile?: string
}

interface GhostState {
  text: string
  visible: boolean
  loading: boolean
  position: { top: number; left: number } | null
}

/** ghost-stream 事件动态载荷（最小消费面） */
interface GhostStreamEvent {
  type?: string
  content?: string
}

const GhostText: React.FC<GhostTextProps> = ({ getCursorContext, enabled, styleProfile: _styleProfile }) => {
  const [ghost, setGhost] = useState<GhostState>({ text: '', visible: false, loading: false, position: null })
  const currentRequestRef = useRef<string>('')
  const overlayRef = useRef<HTMLDivElement>(null)

  // 清理 SSE 监听
  useEffect(() => {
    if (!enabled) return

    const handleGhostStream = (ev: GhostStreamEvent) => {
      if (!ev?.type) return

      if (ev.type === 'chunk') {
        setGhost(prev => {
          const newText = prev.text + (ev.content || '')
          return { ...prev, text: newText, loading: false, visible: true }
        })
        currentRequestRef.current = ''
      } else if (ev.type === 'done') {
        setGhost(prev => {
          const finalText = ev.content || prev.text
          return { ...prev, text: finalText, loading: false, visible: finalText.length > 0 }
        })
        currentRequestRef.current = ''
      } else if (ev.type === 'error') {
        setGhost(prev => ({ ...prev, loading: false, visible: false }))
        currentRequestRef.current = ''
      }
    }

    try {
      window.runtime?.EventsOn?.('ghost-stream', handleGhostStream)
    } catch (_) {}

    return () => {
      try {
        window.runtime?.EventsOff?.('ghost-stream', handleGhostStream)
      } catch (_) {}
    }
  }, [enabled])

  // 接受补全：插入文本并清除 ghost
  const acceptGhost = useCallback(() => {
    const ctx = getCursorContext()
    const ta = ctx?.textareaElement
    if (!ta || !ghost.text) return

    const start = ta.selectionStart
    const before = ta.value.slice(0, start)
    const after = ta.value.slice(ta.selectionEnd ?? start)
    ta.value = before + ghost.text + after

    // 移动光标到补全文本之后
    const newPos = start + ghost.text.length
    ta.setSelectionRange(newPos, newPos)
    ta.focus()

    // 触发 input 事件（React 受控组件需要）
    ta.dispatchEvent(new Event('input', { bubbles: true }))

    setGhost({ text: '', visible: false, loading: false, position: null })
  }, [ghost.text, getCursorContext])

  // 取消补全
  const dismissGhost = useCallback(() => {
    setGhost({ text: '', visible: false, loading: false, position: null })
    currentRequestRef.current = ''
  }, [])

  // 键盘事件: Tab 接受, Esc 取消
  useEffect(() => {
    if (!enabled) return

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Tab' && ghost.visible && ghost.text) {
        e.preventDefault()
        acceptGhost()
      } else if (e.key === 'Escape' && ghost.visible) {
        e.preventDefault()
        dismissGhost()
      }
    }

    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [ghost.visible, ghost.text, enabled, acceptGhost, dismissGhost])

  // 更新 ghost 位置（跟随光标）
  const updatePosition = useCallback(() => {
    const ctx = getCursorContext()
    const ta = ctx?.textareaElement
    if (!ta || !ghost.visible) return

    // 使用 canvas 测量光标位置的像素坐标
    const style = window.getComputedStyle(ta)
    const font = `${style.fontSize} ${style.fontFamily}`

    const canvas = document.createElement('canvas')
    const cctx = canvas.getContext('2d')
    if (!cctx) return

    cctx.font = font
    const beforeCursor = ta.value.slice(0, ta.selectionStart)
    const lines = beforeCursor.split('\n')
    const lastLine = lines[lines.length - 1] || ''
    const lineHeight = parseInt(style.lineHeight) || parseInt(style.fontSize) * 1.5
    const textWidth = cctx.measureText(lastLine).width

    const paddingLeft = parseInt(style.paddingLeft) || 0
    const paddingTop = parseInt(style.paddingTop) || 0

    setGhost(prev => ({
      ...prev,
      position: {
        top: (lines.length - 1) * lineHeight + paddingTop,
        left: textWidth + paddingLeft,
      },
    }))
  }, [getCursorContext, ghost.visible])

  // 当 ghost 可见时持续更新位置
  useEffect(() => {
    if (!ghost.visible) return
    updatePosition()
    const id = setInterval(updatePosition, 500)
    return () => clearInterval(id)
  }, [ghost.visible, updatePosition])

  if (!enabled || !ghost.visible) return null

  return (
    <div
      ref={overlayRef}
      style={{
        position: 'absolute',
        top: ghost.position?.top ?? 0,
        left: ghost.position?.left ?? 0,
        color: 'var(--color-text-secondary)',
        opacity: 0.5,
        fontStyle: 'italic',
        pointerEvents: 'none',
        whiteSpace: 'pre-wrap',
        zIndex: 10,
        maxWidth: 'calc(100% - 40px)',
        overflow: 'hidden',
      }}
    >
      {ghost.loading ? '...' : ghost.text}
    </div>
  )
}

export default GhostText
export type { GhostTextProps }
