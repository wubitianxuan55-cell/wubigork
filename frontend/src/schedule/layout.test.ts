import { describe, expect, it } from 'vitest'
import { layerByTopology } from './layout'

describe('layerByTopology 拓扑分层（v4.128 单代号刀F）', () => {
  it('链式：列=最长路径深度', () => {
    const pos = layerByTopology(
      [{ id: 'a' }, { id: 'b' }, { id: 'c' }],
      [{ from: 'a', to: 'b' }, { from: 'b', to: 'c' }],
    )
    expect(pos.get('a')).toEqual({ col: 0, row: 0 })
    expect(pos.get('b')!.col).toBe(1)
    expect(pos.get('c')!.col).toBe(2)
  })

  it('并行分支同列并列（时标分层会把分支拉直成一行——本函数的核心职责）', () => {
    const pos = layerByTopology(
      [{ id: 'a' }, { id: 'b' }, { id: 'c' }, { id: 'd' }],
      [{ from: 'a', to: 'b' }, { from: 'a', to: 'c' }, { from: 'b', to: 'd' }, { from: 'c', to: 'd' }],
    )
    expect(pos.get('a')!.col).toBe(0)
    expect(pos.get('b')!.col).toBe(1)
    expect(pos.get('c')!.col).toBe(1)
    expect(pos.get('d')!.col).toBe(2)
    // b/c 并行：同行不同列不成立——必须不同行
    expect(pos.get('b')!.row).not.toBe(pos.get('c')!.row)
  })

  it('不同开工日的并行工作不再按时间拉成一条直线（列只看拓扑深度）', () => {
    const pos = layerByTopology(
      [{ id: 'p' }, { id: 'q' }, { id: 'r' }],
      [{ from: 'p', to: 'r' }],
    )
    // q 无边：col 0；p col 0；r col 1——p 与 q 同列，不再因 es 不同分列
    expect(pos.get('p')!.col).toBe(0)
    expect(pos.get('q')!.col).toBe(0)
    expect(pos.get('r')!.col).toBe(1)
    expect(pos.get('p')!.row).not.toBe(pos.get('q')!.row)
  })

  it('菱形汇流：汇点列=最长路径深度（两条分支深度一致）', () => {
    const pos = layerByTopology(
      [{ id: 'a' }, { id: 'b' }, { id: 'c' }, { id: 'd' }, { id: 'e' }],
      [
        { from: 'a', to: 'b' }, { from: 'b', to: 'd' },
        { from: 'a', to: 'c' }, { from: 'c', to: 'd' },
        { from: 'd', to: 'e' },
      ],
    )
    expect(pos.get('e')!.col).toBe(3)
  })

  it('长链深度优先：r 经 p 直连也取最大层级', () => {
    const pos = layerByTopology(
      [{ id: 's' }, { id: 'p' }, { id: 'r' }],
      [{ from: 's', to: 'p' }, { from: 'p', to: 'r' }, { from: 's', to: 'r' }],
    )
    expect(pos.get('r')!.col).toBe(2) // s→p→r 而非 s→r 的 1
  })

  it('空输入返回空表', () => {
    expect(layerByTopology([], []).size).toBe(0)
  })
})
