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

describe('CodeAtlasPanel — 设置中心「代码图谱」标签页', () => {
  beforeEach(() => {
    localStorage.setItem('gaea-lang', 'zh')
  })

  it('渲染快照口径（包数/边数/生成日期）与再生成命令', () => {
    const { container } = render(wrap(<CodeAtlasPanel />))
    const text = container.textContent || ''
    expect(text).toContain(String(meta.packages))
    expect(text).toContain(String(meta.edges))
    expect(text).toContain(meta.generated)
    expect(text).toContain('code_atlas_doc.go')
    // 交互面齐全：层过滤 chips + 搜索框 + 重置
    expect(screen.getByText('内核·共享库')).toBeTruthy()
    expect(screen.getByLabelText('搜索包名…')).toBeTruthy()
    expect(screen.getByRole('button', { name: /重置/ })).toBeTruthy()
  })

  it('层过滤：关掉「内核·共享库」后该层节点隐藏', () => {
    const { container } = render(wrap(<CodeAtlasPanel />))
    const kernelLib = atlas.nodes.find((n) => n.layer === 'kernel-lib')
    expect(container.querySelector(`[data-node-id="${kernelLib?.id}"]`)).toBeTruthy()
    fireEvent.click(screen.getByText('内核·共享库'))
    expect(container.querySelector(`[data-node-id="${kernelLib?.id}"]`)?.getAttribute('display')).toBe('none')
  })

  it('点选节点：高亮一跳邻居并展示单行详情，点空白清除', () => {
    const { container } = render(wrap(<CodeAtlasPanel />))
    // 详情行直查 aria-live 区（节点 <title> 悬停提示与详情同文，文本查询会撞双份）
    const detail = () => container.querySelector('[aria-live="polite"]')?.textContent || ''
    fireEvent.click(container.querySelector('[data-node-id="internal/gaea/boot"]')!)
    expect(detail()).toMatch(/^internal\/gaea\/boot · 内核·装配执行 · 源码 /)
    // 非邻居淡显：入口 (main) 与 boot 无边 → fillOpacity 0.07
    expect(container.querySelector('[data-node-id="(main)"]')?.getAttribute('fill-opacity')).toBe('0.07')
    // 点图谱 svg 空白清除（须锚 data-atlas-svg——容器里第一个 svg 是 antd 图标）
    fireEvent.click(container.querySelector('svg[data-atlas-svg]')!)
    expect(detail()).toBe('点击节点高亮一跳邻居；悬停查看包详情。')
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
    fireEvent.click(screen.getByText('小说域'))
    fireEvent.change(screen.getByLabelText('搜索包名…'), { target: { value: 'xyz-无命中' } })
    fireEvent.click(screen.getByRole('button', { name: /重置/ }))
    const novel = atlas.nodes.find((n) => n.layer === 'novel')
    expect(container.querySelector(`[data-node-id="${novel?.id}"]`)?.getAttribute('display')).toBe(null)
    expect(screen.getByText('点击节点高亮一跳邻居；悬停查看包详情。')).toBeTruthy()
  })
})
