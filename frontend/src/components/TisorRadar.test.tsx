import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import TisorRadar from './TisorRadar'

const dims = { T: 80, I: 60, S: 50, O: 70, R: 40 }

describe('TisorRadar', () => {
  it('dims 缺档时渲染 null 不崩（真数据旧档/外部来源可能无五维；崩页会拖垮错误边界）', () => {
    const { container } = render(<TisorRadar dims={undefined} />)
    expect(container.firstElementChild).toBeNull()
  })

  it('有 dims 时渲染五轴网格与 5 个数据点（外圈光晕+内部圆点）', () => {
    const { container } = render(<TisorRadar dims={dims} />)
    const svg = container.querySelector('svg')
    expect(svg).not.toBeNull()
    // 5 组数据点，每组外圈光晕 + 内部圆点两个 circle
    expect(container.querySelectorAll('circle').length).toBe(10)
    // 参考网格 5 档五边形
    expect(container.querySelectorAll('polygon').length).toBe(5)
    // 轴标签（showLabels 缺省开）
    const labels = Array.from(container.querySelectorAll('text')).map((t) => t.textContent)
    expect(labels).toEqual(['Tend', 'Inde', 'Sens', 'Open', 'Rati'])
  })

  it('showLabels=false 时不渲染轴标签（卡片 52px 小尺寸场景）', () => {
    const { container } = render(<TisorRadar dims={dims} showLabels={false} />)
    expect(container.querySelectorAll('text').length).toBe(0)
  })
})
