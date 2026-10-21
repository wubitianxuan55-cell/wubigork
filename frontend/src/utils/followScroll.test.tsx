// useFollowScroll 回归（v4.514：v4.512 手势语义移植轻语）：
// ①滚轮上滚=暂停跟随（follow 不落底）②无手势 scroll 事件杀不死跟随
// （布局扰动误杀=「卡住没输出假象」同类病灶回归钉）③scrollToBottom/
// rearm 恢复跟随。jsdom 无布局：伪造滚动几何 + 可写 scrollTop 观测落底。
import { describe, expect, it } from 'vitest'
import { fireEvent, render } from '@testing-library/react'
import { useFollowScroll } from './followScroll'

function Harness({ onFollow }: { onFollow?: () => void }) {
  const { ref, atBottom, onScroll, onWheel, follow, rearm, scrollToBottom } = useFollowScroll<HTMLDivElement>()
  return (
    <div>
      <div ref={ref} onScroll={onScroll} onWheel={onWheel} data-testid="scroller">
        <button data-testid="follow" onClick={() => { follow(); onFollow?.() }} />
        <button data-testid="rearm" onClick={rearm} />
        <button data-testid="bottom" onClick={scrollToBottom} />
        <span data-testid="atbottom">{atBottom ? '1' : '0'}</span>
      </div>
    </div>
  )
}

const mockGeometry = (el: HTMLElement, top: number, height = 2000, client = 600) => {
  let cur = top
  Object.defineProperty(el, 'scrollTop', { configurable: true, get: () => cur, set: (v: number) => { cur = v } })
  Object.defineProperty(el, 'scrollHeight', { configurable: true, value: height })
  Object.defineProperty(el, 'clientHeight', { configurable: true, value: client })
  return { get: () => cur, set: (v: number) => { cur = v } }
}

describe('useFollowScroll（v4.514 手势语义）', () => {
  it('滚轮上滚=暂停：follow 不落底，atBottom 随内容自愈', () => {
    const { getByTestId } = render(<Harness />)
    const el = getByTestId('scroller') as HTMLElement
    const geo = mockGeometry(el, 2000)

    fireEvent.click(getByTestId('follow')) // 初始贴底跟随
    expect(geo.get()).toBe(2000)

    fireEvent.wheel(el, { deltaY: -120 })
    geo.set(800)
    fireEvent.scroll(el)
    fireEvent.click(getByTestId('follow')) // 流式新内容
    expect(geo.get()).toBe(800) // 不拽回
    expect(getByTestId('atbottom').textContent).toBe('0') // 按钮态自愈
  })

  it('无手势 scroll 事件（布局扰动）杀不死跟随', () => {
    const { getByTestId } = render(<Harness />)
    const el = getByTestId('scroller') as HTMLElement
    const geo = mockGeometry(el, 2000)

    // 程序性扰动：滚动锚定/夹紧把视口挪到中部（scroll 事件但无滚轮手势）
    geo.set(800)
    fireEvent.scroll(el)
    fireEvent.click(getByTestId('follow')) // 流式新内容
    expect(geo.get()).toBe(2000) // 跟随仍活着，自动补拉回底
  })

  it('scrollToBottom / rearm 恢复跟随', () => {
    const { getByTestId } = render(<Harness />)
    const el = getByTestId('scroller') as HTMLElement
    const geo = mockGeometry(el, 2000)

    fireEvent.wheel(el, { deltaY: -120 })
    geo.set(800)
    fireEvent.scroll(el)

    fireEvent.click(getByTestId('bottom'))
    expect(geo.get()).toBe(2000)

    // rearm：不立即滚动，下次 follow 落底
    fireEvent.wheel(el, { deltaY: -120 })
    geo.set(900)
    fireEvent.scroll(el)
    fireEvent.click(getByTestId('rearm'))
    expect(geo.get()).toBe(900)
    fireEvent.click(getByTestId('follow'))
    expect(geo.get()).toBe(2000)
  })

  it('短会话（不足一屏）无效上滚不产生暂停态', () => {
    const { getByTestId } = render(<Harness />)
    const el = getByTestId('scroller') as HTMLElement
    const geo = mockGeometry(el, 500, 500, 600) // 内容 500 < 视口 600

    fireEvent.wheel(el, { deltaY: -120 })
    fireEvent.click(getByTestId('follow'))
    expect(geo.get()).toBe(500) // 仍跟随（落底=500）
  })

  it('onWheel 下滚/零位移不清除跟随', () => {
    const { getByTestId } = render(<Harness />)
    const el = getByTestId('scroller') as HTMLElement
    const geo = mockGeometry(el, 2000)

    fireEvent.wheel(el, { deltaY: 120 })
    fireEvent.click(getByTestId('follow'))
    expect(geo.get()).toBe(2000)
  })
})
