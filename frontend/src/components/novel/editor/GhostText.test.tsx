// GhostText.test.tsx — 内联续写接线行为锁（v4.444）。
// ①核心疑点关账（v4.429）：受控 textarea 上 Tab 接受必须让 onChange 收到
//   **新值**（原生 setter 路径）——直接 DOM 赋值方案 onChange 收旧值，已被本
//   组件弃用；②800ms 防抖请求带光标前文本；③上下文不足不发请求；
//   ④迟到响应（序号作废）不显示。
// mock 口径照 SkillModal.test.tsx（wailsApp 半桩）。
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import React, { useRef, useState } from 'react'
import GhostText from './GhostText'
import { wailsApp } from '../../../lib/wailsApp'

vi.mock('../../../lib/wailsApp', () => ({
  wailsApp: vi.fn(),
}))

const mockedWailsApp = vi.mocked(wailsApp)

/** 受控 textarea 包装（React 真链路：onChange 落 state 再回填 value）。
 *  ta 用 useRef（ref 回调在 commit 期赋值，effect 首跑即可用）——真机
 *  ChapterEditor 的 sceneTextareaRefs 同为 useRef；useState 会晚一个渲染。 */
function Harness() {
  const [val, setVal] = useState('夜里雪落无声，他在门外站了许久，终于抬起手。')
  const taRef = useRef<HTMLTextAreaElement | null>(null)
  return (
    <div style={{ position: 'relative' }}>
      <textarea
        ref={(el) => { taRef.current = el }}
        data-testid="editor"
        value={val}
        onChange={(e) => setVal(e.target.value)}
        onFocus={() => undefined}
      />
      <GhostText
        enabled
        getCursorContext={() => (taRef.current ? { textBeforeCursor: taRef.current.value.slice(0, taRef.current.selectionStart), textareaElement: taRef.current } : null)}
      />
      <div data-testid="state-len">{val.length}</div>
    </div>
  )
}

const LONG = '夜里雪落无声，他在门外站了许久，终于抬起手。'

describe('GhostText 内联续写（v4.444 接线）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  function setupTextarea(): HTMLTextAreaElement {
    const ta = document.querySelector('[data-testid=editor]') as HTMLTextAreaElement
    ta.focus()
    // 光标置于文末（selectionStart 默认 0——受控值已在 DOM，手动置尾）
    const len = ta.value.length
    ta.setSelectionRange(len, len)
    return ta
  }

  function typeText(ta: HTMLTextAreaElement, text: string) {
    // 走原生 setter + input 事件（与真实击键同链路）；受控重渲染后光标可能
    // 归零，真机浏览器原生 setter 不动光标——测试里显式保持文末光标。
    const setter = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value')!.set!
    setter.call(ta, ta.value + text)
    fireEvent.input(ta)
    const len = ta.value.length
    ta.setSelectionRange(len, len)
  }

  it('核心疑点关账：Tab 接受后受控 onChange 收到含补全的新值（原生 setter 路径）', async () => {
    mockedWailsApp.mockReturnValue({
      NovelGhostSuggest: vi.fn(async () => '门开了。'),
    } as unknown as ReturnType<typeof wailsApp>)
    render(<Harness />)
    const ta = setupTextarea()

    // 输入触发防抖
    typeText(ta, '门')
    await act(async () => { await vi.advanceTimersByTimeAsync(800) })
    expect(mockedWailsApp().NovelGhostSuggest).toHaveBeenCalledWith(expect.stringContaining(LONG))

    // ghost 悬浮出现 → Tab 接受（fake timers 下 findBy* 轮询会挂死，act 直读）
    await act(async () => {})
    expect(screen.getByText('门开了。')).toBeTruthy()
    fireEvent.keyDown(document, { key: 'Tab' })
    expect(ta.value).toContain('门开了。')
    // 受控 state（React onChange 链）必须收到新值——v4.429 疑点的正解证据
    //（fireEvent/Tab 链路同步落地，无需 waitFor；waitFor 配 fake timers 会挂死）
    expect(screen.getByTestId('state-len').textContent).toBe(String(ta.value.length))
  })

  it('上下文不足不发请求', async () => {
    mockedWailsApp.mockReturnValue({
      NovelGhostSuggest: vi.fn(async () => '不应发生'),
    } as unknown as ReturnType<typeof wailsApp>)
    render(<Harness />)
    const ta = setupTextarea()
    typeText(ta, '门')
    ta.setSelectionRange(0, 0) // 光标在头部：textBeforeCursor 为空（typeText 会归位文末，须在其后设置）
    await act(async () => { await vi.advanceTimersByTimeAsync(800) })
    expect(mockedWailsApp().NovelGhostSuggest).not.toHaveBeenCalled()
  })

  it('迟到响应（被新输入作废）不显示', async () => {
    let resolveA: (v: string) => void = () => {}
    const slow = new Promise<string>((r) => { resolveA = r })
    const suggest = vi.fn()
      .mockImplementationOnce(() => slow)
      .mockImplementationOnce(async () => '新的建议。')
    mockedWailsApp.mockReturnValue({ NovelGhostSuggest: suggest } as unknown as ReturnType<typeof wailsApp>)
    render(<Harness />)
    const ta = setupTextarea()

    typeText(ta, '一')
    await act(async () => { await vi.advanceTimersByTimeAsync(800) })
    // 第二次输入作废第一次的在途请求
    typeText(ta, '又')
    await act(async () => { await vi.advanceTimersByTimeAsync(800) })
    resolveA('迟到的旧建议。')
    await act(async () => {})
    expect(screen.queryByText('迟到的旧建议。')).toBeNull()
  })
})
