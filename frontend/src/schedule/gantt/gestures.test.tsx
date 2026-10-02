/**
 * gantt/gestures.test.ts — 窗口级拖拽三件套外壳 + 缩放外壳单源（FE7-12）
 *
 * beginWindowDrag：AoaView/GanttView 共用的 mousemove/mouseup/keydown-Esc 外壳——
 * up=移除三监听 → moved 才 commit → end；Esc=移除三监听 → end（不提交）；
 * 监听器回调内自清理（Esc 与 up 双跑安全）。
 * useFitView/useAutoFitZoom：AoaView/PdmView 共用的缩放壳——钳域注入（本测试
 * 用 PDM 域 [0.2,2] 演练；AOA 域 [0.1,2] 由 AoaView.test.tsx 的既有用例覆盖）、
 * touched 后自动适配不再抢占、guard=false 不适配。
 */
import { act, fireEvent, render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import React from 'react'
import { beginWindowDrag, type WindowDragSpec } from './useDragGesture'
import { useAutoFitZoom, useFitView } from './useFitView'

describe('beginWindowDrag 三件套外壳（FE7-12 单源）', () => {
  it('move 逐事件推进同一载荷；up 且 moved → commit 后 end', () => {
    const moves: number[] = []
    const spec: WindowDragSpec<{ dx: number }> = {
      move: (p, ev) => { p.dx = ev.clientX; moves.push(ev.clientX) },
      moved: (p) => p.dx !== 100,
      commit: vi.fn(),
      end: vi.fn(),
    }
    beginWindowDrag(spec, { dx: 0 })
    fireEvent.mouseMove(window, { clientX: 160 })
    fireEvent.mouseMove(window, { clientX: 130 })
    expect(moves).toEqual([160, 130])
    fireEvent.mouseUp(window)
    expect(vi.mocked(spec.commit)).toHaveBeenCalledTimes(1)
    expect(vi.mocked(spec.end)).toHaveBeenCalledTimes(1)
    // commit 先于 end（与 GanttView 原顺序一致）
    const commitOrder = vi.mocked(spec.commit).mock.invocationCallOrder[0]
    const endOrder = vi.mocked(spec.end).mock.invocationCallOrder[0]
    expect(commitOrder).toBeLessThan(endOrder)
  })

  it('未移动（moved=false）→ 只 end 不 commit（「未移动不提交」两视图同规）', () => {
    const spec: WindowDragSpec<{ dx: number }> = {
      move: () => {},
      moved: () => false,
      commit: vi.fn(),
      end: vi.fn(),
    }
    beginWindowDrag(spec, { dx: 0 })
    fireEvent.mouseMove(window, { clientX: 100 })
    fireEvent.mouseUp(window)
    expect(vi.mocked(spec.commit)).not.toHaveBeenCalled()
    expect(vi.mocked(spec.end)).toHaveBeenCalledTimes(1)
  })

  it('Esc → end 不提交；监听自清理：Esc 后 move/up/再 Esc 均不再触发', () => {
    const spec: WindowDragSpec<{ dx: number }> = {
      move: vi.fn(),
      moved: () => true,
      commit: vi.fn(),
      end: vi.fn(),
    }
    beginWindowDrag(spec, { dx: 0 })
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(vi.mocked(spec.end)).toHaveBeenCalledTimes(1)
    expect(vi.mocked(spec.commit)).not.toHaveBeenCalled()
    fireEvent.mouseMove(window, { clientX: 200 })
    fireEvent.mouseUp(window)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(vi.mocked(spec.move)).not.toHaveBeenCalled()
    expect(vi.mocked(spec.commit)).not.toHaveBeenCalled()
    expect(vi.mocked(spec.end)).toHaveBeenCalledTimes(1)
  })

  it('非 Esc 按键不触发收尾；up 后 Esc 不再追加 end（双跑安全）', () => {
    const spec: WindowDragSpec<{ dx: number }> = {
      move: vi.fn(),
      moved: () => true,
      commit: vi.fn(),
      end: vi.fn(),
    }
    beginWindowDrag(spec, { dx: 0 })
    fireEvent.keyDown(window, { key: 'Enter' })
    expect(vi.mocked(spec.end)).not.toHaveBeenCalled()
    fireEvent.mouseUp(window)
    expect(vi.mocked(spec.end)).toHaveBeenCalledTimes(1)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(vi.mocked(spec.end)).toHaveBeenCalledTimes(1)
  })
})

/** 测试挂具：暴露控制器 + 画一个可换 key/guard/公式的自动适配 */
function FitHarness(props: {
  scrollRef: React.MutableRefObject<HTMLDivElement | null>
  ctrlOut: React.MutableRefObject<ReturnType<typeof useFitView> | null>
  zoomOf: (el: HTMLElement) => number | null
  fitKey: unknown
  guard: boolean
}): null {
  const { scrollRef, ctrlOut, zoomOf, fitKey, guard } = props
  const ctrl = useFitView(scrollRef, 0.2, 2)
  ctrlOut.current = ctrl
  useAutoFitZoom(scrollRef, ctrl.touchedRef, ctrl.applyZoom, zoomOf, fitKey, guard)
  return null
}

/** jsdom 无布局：注入视口尺寸的假滚动容器 */
function fakeViewport(clientWidth: number, clientHeight: number): HTMLDivElement {
  const el = document.createElement('div')
  Object.defineProperty(el, 'clientWidth', { value: clientWidth, configurable: true })
  Object.defineProperty(el, 'clientHeight', { value: clientHeight, configurable: true })
  return el
}

describe('useFitView / useAutoFitZoom 缩放外壳（FE7-12 单源）', () => {
  it('applyZoom：round2 后钳入注入域 [0.2,2]，zoomRef 同步镜像', () => {
    const scrollRef = { current: null as HTMLDivElement | null }
    const ctrlOut = { current: null as ReturnType<typeof useFitView> | null }
    render(<FitHarness scrollRef={scrollRef} ctrlOut={ctrlOut} zoomOf={() => null} fitKey={0} guard={false} />)
    act(() => ctrlOut.current!.applyZoom(5))
    expect(ctrlOut.current!.zoom).toBe(2)
    expect(ctrlOut.current!.zoomRef.current).toBe(2)
    act(() => ctrlOut.current!.applyZoom(0.001))
    expect(ctrlOut.current!.zoom).toBe(0.2)
    act(() => ctrlOut.current!.applyZoom(0.567))
    expect(ctrlOut.current!.zoom).toBe(0.57)
  })

  it('stepZoom：置 touched + 乘因子；fitView 走注入公式（null=不适配）', () => {
    const scrollRef = { current: fakeViewport(800, 600) }
    const ctrlOut = { current: null as ReturnType<typeof useFitView> | null }
    render(<FitHarness scrollRef={scrollRef} ctrlOut={ctrlOut} zoomOf={() => null} fitKey={0} guard={false} />)
    act(() => ctrlOut.current!.stepZoom(1.2))
    expect(ctrlOut.current!.zoom).toBe(1.2)
    expect(ctrlOut.current!.touchedRef.current).toBe(true)
    // 公式 800/400=2 → 钳 2；null 公式不动读数
    act(() => ctrlOut.current!.fitView((el) => Math.min(el.clientWidth / 400, 1.5)))
    expect(ctrlOut.current!.zoom).toBe(1.5)
    act(() => ctrlOut.current!.fitView(() => null))
    expect(ctrlOut.current!.zoom).toBe(1.5)
  })

  it('自动适配：内容就绪（guard）且未 touched 才适配；key 变化重跑；touched 后不再抢占', () => {
    const scrollRef = { current: fakeViewport(800, 600) }
    const ctrlOut = { current: null as ReturnType<typeof useFitView> | null }
    const zoomOf = (el: HTMLElement): number | null => Math.min(el.clientWidth / 1600, 1) // 800/1600=0.5
    const { rerender } = render(
      <FitHarness scrollRef={scrollRef} ctrlOut={ctrlOut} zoomOf={zoomOf} fitKey="a" guard={true} />,
    )
    expect(ctrlOut.current!.zoom).toBe(0.5)
    // 内容尺寸变化（key 变）→ 重新适配
    rerender(<FitHarness scrollRef={scrollRef} ctrlOut={ctrlOut} zoomOf={zoomOf} fitKey="b" guard={true} />)
    expect(ctrlOut.current!.zoom).toBe(0.5)
    // guard=false（内容未就绪）→ 不适配：先手动缩放出非公式值再验证不被改写
    act(() => ctrlOut.current!.applyZoom(0.9))
    rerender(<FitHarness scrollRef={scrollRef} ctrlOut={ctrlOut} zoomOf={zoomOf} fitKey="c" guard={false} />)
    expect(ctrlOut.current!.zoom).toBe(0.9)
    rerender(<FitHarness scrollRef={scrollRef} ctrlOut={ctrlOut} zoomOf={zoomOf} fitKey="d" guard={true} />)
    expect(ctrlOut.current!.zoom).toBe(0.5)
    // 用户手动缩放（touched）→ 后续内容变化不再抢占
    act(() => ctrlOut.current!.stepZoom(1.2))
    expect(ctrlOut.current!.zoom).toBe(0.6)
    rerender(<FitHarness scrollRef={scrollRef} ctrlOut={ctrlOut} zoomOf={zoomOf} fitKey="e" guard={true} />)
    expect(ctrlOut.current!.zoom).toBe(0.6)
  })

  it('自动适配：zoomOf 返回 null（如 clientWidth=0 的 jsdom 场景）本帧不适配', () => {
    const scrollRef = { current: fakeViewport(0, 0) }
    const ctrlOut = { current: null as ReturnType<typeof useFitView> | null }
    const zoomOf = (el: HTMLElement): number | null => (el.clientWidth <= 0 ? null : 0.8)
    render(<FitHarness scrollRef={scrollRef} ctrlOut={ctrlOut} zoomOf={zoomOf} fitKey="a" guard={true} />)
    expect(ctrlOut.current!.zoom).toBe(1) // 初始值不动
  })
})
