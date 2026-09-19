// perf-guards.test.ts — 运行开销与布局结构守卫（v4.349 前端优化刀）
//
// Why：本刀修掉的三类问题都是「结构性」的——一旦有人把 memo 拆掉、
// 把心跳轮询加回来、或把 nowrap/行数上限删掉，功能测试不会红，只有用户
// 感受到卡顿/参差。此处按仓内既有 source-guard 风格（App.tokens.test.ts /
// contrast-hex.test.ts）把这些不变量钉死。
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const read = (...p: string[]) => readFileSync(resolve(__dirname, ...p), 'utf8')

describe('主线程心跳：不再自带走 500ms 空转轮询（v4.349）', () => {
  const src = read('main.tsx')

  it('心跳戳由 rAF 探测循环维护（probe 每帧写 mainThreadBeatAt）', () => {
    expect(src).toContain('let mainThreadBeatAt = Date.now()')
    expect(src).toMatch(/probe[\s\S]{0,200}mainThreadBeatAt = Date\.now\(\)/)
  })

  it('诊断层读共享心跳戳，且不再自带 500ms 只写时间戳的 interval', () => {
    expect(src).not.toMatch(/setInterval\(\(\)\s*=>\s*\{\s*lastBeat = Date\.now\(\)\s*\},\s*500\)/)
    expect(src).toMatch(/Date\.now\(\) - mainThreadBeatAt/)
  })

  it('rAF 降级路径有兜底续跳（否则诊断层把已降级的正常态误报成卡死）', () => {
    const degradeBlock = src.slice(src.indexOf('const degrade ='), src.indexOf('// 持续探测'))
    expect(degradeBlock).toContain('mainThreadBeatAt = Date.now()')
  })
})

describe('主题令牌与 antd 主题对象 memo 化（v4.349）', () => {
  const src = read('App.tsx')

  it('getThemeTokens 包在 useMemo 里（否则每次渲染都换新令牌对象 → 主题 effect 重跑）', () => {
    expect(src).toMatch(/const tokens = useMemo\(\(\) => getThemeTokens\(/)
    expect(src).not.toMatch(/^\s*const tokens = getThemeTokens\(/m)
  })

  it('ConfigProvider theme 走 memo 对象，不再是内联字面量', () => {
    expect(src).toMatch(/const antdTheme = useMemo\(/)
    expect(src).toContain('<ConfigProvider theme={antdTheme}>')
  })
})

describe('最近文件实时化：Home 与办公文件面都订阅单源（v4.349）', () => {
  it('ModuleLauncher 首页账页用 useSyncExternalStore + subscribeRecentFiles（不再挂载时读一次）', () => {
    const src = read('components/ModuleLauncher.tsx')
    expect(src).toContain('useSyncExternalStore(subscribeRecentFiles, loadRecentFiles')
    expect(src).not.toContain('setRecentFiles(loadRecentFiles()')
  })

  it('RecentFilesBar 同样订阅单源', () => {
    const src = read('gaea/components/RecentFilesBar.tsx')
    expect(src).toContain('useSyncExternalStore(subscribeRecentFiles, loadRecentFiles')
  })
})

describe('布局守卫（v4.349）', () => {
  it('.gsched 摘要 chip 不再被压缩换行（shrink-0 + whitespace-nowrap）', () => {
    const src = read('gaea/components/ScheduleFileCard.tsx')
    const chip = src.slice(src.indexOf('function StatChip'), src.indexOf('export function ScheduleFileCard'))
    expect(chip).toContain('shrink-0')
    expect(chip).toContain('whitespace-nowrap')
  })

  it('基线槽位行可换行、固定段不压缩（工期/时间戳）', () => {
    const src = read('schedule/BaselinesPanel.tsx')
    expect(src).toContain("flexWrap: 'wrap'")
    expect(src).toMatch(/总工期 \{b\.duration\} 天/)
    // 工期 + 时间戳两处固定段都带 nowrap/不可压缩
    expect(src.match(/whiteSpace: 'nowrap', flexShrink: 0/g)?.length).toBe(2)
  })

  it('海报墙描述有行数上限（overflow:hidden 定高卡不再静默裁切）', () => {
    const css = read('components/module-launcher.css')
    const block = css.slice(css.indexOf('.p-poster-desc {'), css.indexOf('.p-poster-go {'))
    expect(block).toContain('-webkit-line-clamp: 3')
    expect(block).toContain('overflow: hidden')
  })
})

describe('可访问性守卫（v4.349）', () => {
  it('全屏看图关闭钮有可访问名', () => {
    const src = read('components/Lightbox.tsx')
    expect(src).toMatch(/<Button type="text" icon=\{<CloseOutlined \/>\}[^>]*aria-label="关闭/)
  })

  it('阻断式批准弹窗是 aria-modal="true"', () => {
    const src = read('gaea/components/ApprovalModal.tsx')
    expect(src).toContain('role="dialog" aria-modal="true"')
    expect(src).not.toContain('aria-modal="false"')
  })

  it('阅读「删除高亮」走二次确认（Popconfirm），不再裸 onClick', () => {
    const src = read('pages/chapter/readingOverlays.tsx')
    expect(src).toContain('<Popconfirm')
    expect(src).not.toContain('<Button danger size="small" onClick={() => onDeleteAnnotation(noteTarget.id)}>')
  })
})
