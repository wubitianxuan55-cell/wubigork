/**
 * PdmView.test.tsx — 单代号视图冒烟 + 刀H 过桥法接线（v4.130.0）
 *
 * PdmView 此前无组件测试；刀H 把连线几何改走 aoaLayout 段序列
 * （findBridgeArcs/segsToPath 与双代号同源），补冒烟钉住：节点/搭接/标签
 * 渲染齐全，且真实交叉出现过桥半圆（d 含 A 命令）。
 * 交叉构型：三行起步（A/B/C 同列 0/1/2 行），C→D 竖段纵跨 2 行距，
 * B→E 跨两列的首段水平线（y=行1）从其中间穿过——PDM 几何中横竖段都在
 * 节点行中线上，普通两行项目只会端点相触，必须 ≥3 行 + 跨列长横线才有内交。
 */
import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { computeCpm } from './cpm'
import { PdmView } from './PdmView'
import type { SchedProject } from './types'

function crossingProject(): SchedProject {
  return {
    name: '过桥样板',
    startDate: '2026-09-07',
    tasks: [
      { id: 'A', name: '甲', duration: 2, level: 1, progress: 0 },
      { id: 'B', name: '乙', duration: 1, level: 1, progress: 0 },
      { id: 'C', name: '丙', duration: 4, level: 1, progress: 0 },
      { id: 'D', name: '丁', duration: 2, level: 1, progress: 0 },
      { id: 'E', name: '戊', duration: 1, level: 1, progress: 0 },
    ],
    links: [
      { from: 'C', to: 'D', type: 'FS', lag: 0 },
      { from: 'A', to: 'E', type: 'FS', lag: 0 },
      { from: 'D', to: 'E', type: 'FS', lag: 0 },
      { from: 'B', to: 'E', type: 'FS', lag: 0 },
    ],
  }
}

describe('PdmView（刀H G1 过桥接线）', () => {
  it('节点/搭接/标签渲染齐全', () => {
    const p = crossingProject()
    render(<PdmView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    const root = screen.getByTestId('sched-pdm')
    expect(root.textContent).toContain('丙')
    expect(root.textContent).toContain('FS')
    // 实搭接 4 条（虚拟 S 桩同挂 sched-net-link 类，排除后计数）
    expect(root.querySelectorAll('path.sched-net-link:not(.sched-net-link-virtual)').length).toBe(4)
  })

  it('连线交叉处出现过桥半圆（d 含 A 命令）', () => {
    const p = crossingProject()
    render(<PdmView project={p} cpm={computeCpm(p.tasks, p.links)} />)
    // 图例里有 antd 按钮图标 svg，必须按类名取网络图连线（桥画在连线 path 上）
    const links = Array.from(screen.getByTestId('sched-pdm').querySelectorAll('path.sched-net-link, path.sched-net-link-virtual'))
    const bridged = links.filter((el) => (el.getAttribute('d') ?? '').includes('A 5 5'))
    expect(bridged.length).toBe(1) // C→D 的竖段被 B→E 的长横线内交，恰一处
  })
})
