import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, render, screen } from '@testing-library/react'
import { useEffect } from 'react'
import { isPageVisible } from '../lib/pollingGate'
import { usePollingGate } from './usePollingGate'

// ── 模拟 document.visibilityState（jsdom 惯用 defineProperty + restore）──
// jsdom 的 visibilityState 是 Document.prototype 上的只读 getter；
// 在 document 实例上 defineProperty 覆盖，测试后 delete 实例属性即恢复原型 getter。
function setVisibility(state: DocumentVisibilityState) {
  Object.defineProperty(document, 'visibilityState', {
    configurable: true,
    value: state,
  })
  document.dispatchEvent(new Event('visibilitychange'))
}

function restoreVisibility() {
  delete (document as unknown as { visibilityState?: unknown }).visibilityState
  document.dispatchEvent(new Event('visibilitychange'))
}

/** 轮询探针：与各轮询点相同的消费模式（执行体套 if (!visible) return） */
function PollProbe({ onTick, enabled = true }: { onTick: () => void; enabled?: boolean }) {
  const visible = usePollingGate(enabled)
  useEffect(() => {
    const tick = () => { if (visible) onTick() }
    tick()
    const t = window.setInterval(tick, 1000)
    return () => window.clearInterval(t)
  }, [visible, onTick])
  return <div data-testid="gate" data-visible={String(visible)} />
}

describe('pollingGate（系统级后台轮询门控）', () => {
  let onTick: ReturnType<typeof vi.fn<() => void>>

  beforeEach(() => {
    vi.useFakeTimers()
    onTick = vi.fn<() => void>()
  })

  afterEach(() => {
    act(() => { restoreVisibility() })
    vi.useRealTimers()
  })

  it('isPageVisible：jsdom 默认 visible；defineProperty 置 hidden 后为 false，restore 后恢复', () => {
    expect(isPageVisible()).toBe(true)
    act(() => { setVisibility('hidden') })
    expect(isPageVisible()).toBe(false)
    act(() => { restoreVisibility() })
    expect(isPageVisible()).toBe(true)
  })

  it('visible 时回调立即执行且随 interval 持续执行', () => {
    render(<PollProbe onTick={onTick} />)
    expect(onTick).toHaveBeenCalledTimes(1) // 挂载立即执行
    expect(screen.getByTestId('gate').getAttribute('data-visible')).toBe('true')
    act(() => { vi.advanceTimersByTime(3500) })
    expect(onTick).toHaveBeenCalledTimes(4) // 1 立即 + 3 tick
  })

  it('hidden 时回调不执行（挂载即 hidden：立即调用与 interval 均空转）', () => {
    act(() => { setVisibility('hidden') })
    render(<PollProbe onTick={onTick} />)
    act(() => { vi.advanceTimersByTime(5000) })
    expect(onTick).not.toHaveBeenCalled()
    expect(screen.getByTestId('gate').getAttribute('data-visible')).toBe('false')
  })

  it('运行中变 hidden 停止执行；恢复 visible 立即补一次并继续', () => {
    render(<PollProbe onTick={onTick} />)
    act(() => { vi.advanceTimersByTime(2000) })
    expect(onTick).toHaveBeenCalledTimes(3) // 1 立即 + 2 tick

    act(() => { setVisibility('hidden') })
    expect(screen.getByTestId('gate').getAttribute('data-visible')).toBe('false')
    act(() => { vi.advanceTimersByTime(10000) })
    expect(onTick).toHaveBeenCalledTimes(3) // hidden 期间空转零成本

    act(() => { setVisibility('visible') })
    expect(onTick).toHaveBeenCalledTimes(4) // 恢复可见立即补一次
    act(() => { vi.advanceTimersByTime(2000) })
    expect(onTick).toHaveBeenCalledTimes(6)
  })

  it('enabled=false 时即使 visible 也不执行', () => {
    render(<PollProbe onTick={onTick} enabled={false} />)
    act(() => { vi.advanceTimersByTime(3000) })
    expect(onTick).not.toHaveBeenCalled()
  })
})
