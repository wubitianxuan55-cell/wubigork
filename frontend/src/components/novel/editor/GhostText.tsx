import React, { useState, useEffect, useRef, useCallback } from 'react'
import { wailsApp } from '../../../lib/wailsApp'

/**
 * GhostText — 内联 AI 补全组件（v4.444 接线：v4.425 的 UI 壳此前无请求路径、
 * 无后端、挂载点禁用，实为半成品死代码；v4.429「受控值疑点」即 acceptGhost
 * 的直接 DOM 赋值——本刀以原生 setter 修复（React 属性钩子会去重 onChange，
 * 直接赋值后 dispatch input 收到旧值，v4.429 微实验实证））。
 *
 * 行为：所在 textarea 停止输入 800ms 且聚焦、上下文 ≥10 字时自动请求续写；
 * 建议以灰色斜体悬浮在光标后，Tab 接受，Esc 取消，继续输入即弃。
 * 单发非流式（1~2 句补全走 SSE 是过度设计），请求序号守卫弃迟到响应。
 *
 * Props:
 *   getCursorContext — 返回 { textBeforeCursor, textareaElement }（所属场景光标）
 *   enabled — 是否启用
 *   styleProfile — 可选的风格指导（V1 未消费，接口保留）
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

const GHOST_DEBOUNCE_MS = 800
const GHOST_MIN_CONTEXT = 10

const GhostText: React.FC<GhostTextProps> = ({ getCursorContext, enabled, styleProfile: _styleProfile }) => {
  const [ghost, setGhost] = useState<GhostState>({ text: '', visible: false, loading: false, position: null })
  // 请求序号：新输入/新请求即作废在途响应（多场景实例同挂，序号隔离各自的）
  const reqSeqRef = useRef(0)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const clearGhost = useCallback(() => {
    reqSeqRef.current++
    setGhost({ text: '', visible: false, loading: false, position: null })
  }, [])

  // 触发链：所在 textarea 的 input → 立即弃旧 ghost + 800ms 防抖请求。
  // getCursorContext 是父级每次渲染新建的闭包（latest-ref 持有最新版，effect
  // 只挂 [enabled]——否则每次渲染重挂监听且 cleanup 作废在途请求）；ta 由场景
  // 卡 refs 持有、跨渲染稳定。
  const ctxRef = useRef(getCursorContext)
  ctxRef.current = getCursorContext
  useEffect(() => {
    if (!enabled) return
    const ta = ctxRef.current()?.textareaElement
    if (!ta) return

    const request = () => {
      const ctx = ctxRef.current()
      if (!ctx?.textareaElement || document.activeElement !== ctx.textareaElement) return
      const before = ctx.textBeforeCursor
      if (before.trim().length < GHOST_MIN_CONTEXT) return
      const seq = ++reqSeqRef.current
      setGhost({ text: '', visible: false, loading: true, position: null })
      wailsApp()
        .NovelGhostSuggest(before)
        .then((text) => {
          if (seq !== reqSeqRef.current || !text) return // 迟到响应丢弃
          setGhost({ text, visible: true, loading: false, position: null })
        })
        .catch(() => {
          // 无建议/离线/上下文不足：静默不弹框（补全是助攻不是任务）
          if (seq === reqSeqRef.current) setGhost({ text: '', visible: false, loading: false, position: null })
        })
    }

    const onInput = () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
      reqSeqRef.current++ // 输入即作废在途响应
      setGhost(prev => (prev.visible || prev.loading ? { text: '', visible: false, loading: false, position: null } : prev))
      debounceRef.current = setTimeout(request, GHOST_DEBOUNCE_MS)
    }

    ta.addEventListener('input', onInput)
    return () => {
      ta.removeEventListener('input', onInput)
      if (debounceRef.current) clearTimeout(debounceRef.current)
      reqSeqRef.current++
    }
  }, [enabled])

  // 接受补全：原生 setter 写值 + dispatch input（React 属性钩子对直接赋值
  // 会去重 onChange——v4.429 微实验实证 onChange 收旧值；原生 setter 绕开钩子，
  // v4.441.1 走查配方同款，已实证受控组件 onChange 收到新值）。
  const acceptGhost = useCallback(() => {
    const ctx = getCursorContext()
    const ta = ctx?.textareaElement
    if (!ta || !ghost.text) return

    const start = ta.selectionStart
    const before = ta.value.slice(0, start)
    const after = ta.value.slice(ta.selectionEnd ?? start)
    const setter = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value')?.set
    if (setter) {
      setter.call(ta, before + ghost.text + after)
    } else {
      ta.value = before + ghost.text + after // 理论不可达：原型描述符恒在
    }
    ta.dispatchEvent(new Event('input', { bubbles: true }))

    const newPos = start + ghost.text.length
    ta.setSelectionRange(newPos, newPos)
    ta.focus()

    setGhost({ text: '', visible: false, loading: false, position: null })
  }, [ghost.text, getCursorContext])

  // 取消补全
  const dismissGhost = useCallback(() => {
    clearGhost()
  }, [clearGhost])

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
      {ghost.loading ? '…' : ghost.text}
    </div>
  )
}

export default GhostText
export type { GhostTextProps }
