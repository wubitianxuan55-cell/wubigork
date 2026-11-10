import { describe, expect, it, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import React from 'react'

import CodeAtlasPanel from './CodeAtlasPanel'
import { LocaleProvider } from '../../gaea/lib/i18n'
import atlas from '../../data/codeAtlas.json'

// S2.2b i18n：面板经 useT 读字典，测试包 LocaleProvider。钉 zh：jsdom 的
// navigator.language=en-US 会让 detectLocale 落 en，且 en 字典 chunk 异步落地
// 前后文案会从 zh 回退态漂成英文（BootSplash.test 同款先例）。
const wrap = (node: React.ReactNode) => <LocaleProvider>{node}</LocaleProvider>

const meta = atlas.meta as { packages: number; edges: number; generated: string }

/** 层 chip 定位：节点 <title> 悬停提示也含层名，文本查询必撞双份——按
 * .ant-tag-checkable + 前缀精确取 chip 本体。 */
const chip = (container: HTMLElement, name: string): HTMLElement => {
  const el = Array.from(container.querySelectorAll('.ant-tag-checkable'))
    .find((c) => (c.textContent || '').startsWith(name))
  if (!el) throw new Error(`chip not found: ${name}`)
  return el as HTMLElement
}

describe('CodeAtlasPanel — 设置中心「代码图谱」标签页', () => {
  beforeEach(() => {
    localStorage.setItem('gaea-lang', 'zh')
  })

  it('渲染快照口径（四底数+快照日期+再生成命令）与交互面', () => {
    const { container } = render(wrap(<CodeAtlasPanel />))
    const text = container.textContent || ''
    expect(text).toContain(String(meta.packages))
    expect(text).toContain(String(meta.edges))
    expect(text).toContain(meta.generated)
    expect(text).toContain('code_atlas_doc.go')
    // 层 chips=图例+过滤（色点+名称+计数）
    expect(chip(container, '内核·共享库').textContent).toMatch(/内核·共享库\s*5[0-9]/)
    expect(screen.getByLabelText('搜索包名…')).toBeTruthy()
    expect(screen.getByRole('button', { name: /重置/ })).toBeTruthy()
  })

  it('层过滤：关掉「内核·共享库」后该层节点隐藏', () => {
    const { container } = render(wrap(<CodeAtlasPanel />))
    const kernelLib = atlas.nodes.find((n) => n.layer === 'kernel-lib')
    expect(container.querySelector(`[data-node-id="${kernelLib?.id}"]`)).toBeTruthy()
    fireEvent.click(chip(container, '内核·共享库'))
    expect(container.querySelector(`[data-node-id="${kernelLib?.id}"]`)?.getAttribute('display')).toBe('none')
  })

  it('点选节点：高亮一跳邻居并展示检视条，点空白清除', () => {
    const { container } = render(wrap(<CodeAtlasPanel />))
    // 检视条直查 aria-live 区（节点 <title> 悬停提示与包名同文，文本查询会撞双份）
    const detail = () => container.querySelector('[aria-live="polite"]')?.textContent || ''
    fireEvent.click(container.querySelector('[data-node-id="internal/gaea/boot"]')!)
    expect(detail()).toContain('internal/gaea/boot')
    expect(detail()).toContain('内核·装配执行')
    expect(detail()).toMatch(/被引\s*1/)
    // 非邻居淡显：入口 (main) 与 boot 无边 → fillOpacity 0.07
    expect(container.querySelector('[data-node-id="(main)"]')?.getAttribute('fill-opacity')).toBe('0.07')
    // 选中环落在选中节点上
    expect(container.querySelector('[data-node-ring="internal/gaea/boot"]')).toBeTruthy()
    // 点图谱 svg 空白清除（须锚 data-atlas-svg——容器里第一个 svg 是 antd 图标）
    fireEvent.click(container.querySelector('svg[data-atlas-svg]')!)
    expect(detail()).toContain('点击神经元')
  })

  it('搜索：命中计数上屏，未命中包淡显、命中包高亮', () => {
    const { container } = render(wrap(<CodeAtlasPanel />))
    fireEvent.change(screen.getByLabelText('搜索包名…'), { target: { value: 'internal/gaea/evidence' } })
    expect(screen.getByText(/个包命中/)).toBeTruthy()
    expect(container.querySelector('[data-node-id="internal/app"]')?.getAttribute('fill-opacity')).toBe('0.07')
    expect(container.querySelector('[data-node-id="internal/gaea/evidence"]')?.getAttribute('fill-opacity')).toBe('0.95')
  })

  it('重置：清层过滤与搜索态', () => {
    const { container } = render(wrap(<CodeAtlasPanel />))
    fireEvent.click(chip(container, '小说域'))
    fireEvent.change(screen.getByLabelText('搜索包名…'), { target: { value: 'xyz-无命中' } })
    fireEvent.click(screen.getByRole('button', { name: /重置/ }))
    const novel = atlas.nodes.find((n) => n.layer === 'novel')
    expect(container.querySelector(`[data-node-id="${novel?.id}"]`)?.getAttribute('display')).toBe(null)
    expect(container.querySelector('[aria-live="polite"]')?.textContent).toContain('点击神经元')
  })

  it('空态：全部层隐藏时画布居中提示', () => {
    const { container } = render(wrap(<CodeAtlasPanel />))
    const chips = ['入口', '壳层·板块装配', '内核·装配执行', '内核·共享库', '办公域', '小说域', '引擎域', '众神殿', '领域库', 'scripts 工具带']
    chips.forEach((name) => fireEvent.click(chip(container, name)))
    expect(container.querySelector('svg[data-atlas-svg]')?.textContent).toContain('已隐藏全部层')
  })

  it('前馈不变量：每条突触严格从左往右（被依赖→依赖方），(main) 是最右输出神经元', () => {
    const { container } = render(wrap(<CodeAtlasPanel />))
    const paths = Array.from(container.querySelectorAll('path[data-edge]'))
    expect(paths.length).toBe(atlas.edges.length)
    for (const p of paths) {
      const m = (p.getAttribute('d') || '').match(/^M ([\d.-]+) [\d.-]+ C .*, ([\d.-]+) [\d.-]+$/)
      expect(m).not.toBeNull()
      if (!m) continue
      expect(parseFloat(m[2])).toBeGreaterThan(parseFloat(m[1]))
    }
    // 拓扑分层尽头：(main) 的 cx 全场最大（唯一 h=11 输出神经元）
    const mainCx = parseFloat(container.querySelector('[data-node-id="(main)"]')?.getAttribute('cx') || '0')
    container.querySelectorAll('[data-node-id]').forEach((c) => {
      expect(parseFloat(c.getAttribute('cx') || '0')).toBeLessThanOrEqual(mainCx)
    })
  })
})
