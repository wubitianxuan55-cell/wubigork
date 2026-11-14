/**
 * displayPreset.css.test.ts — 显示预设样式表静态断言（W6 防静默退化）
 *
 * 为什么要有这个文件：`display-preset.css` 是纯变量表，组件测试覆盖不到它
 * ——少写一个变量、写回裸 hex、或把 `--md-sys-color-*` 主题令牌也覆盖了，
 * 屏上都「看起来还行」，直到换主题/导出黑白件才暴露。这里把四条口径钉死：
 *  1. 两个类各两档（深底 + 白底）四块齐全（常规档按定义没有类，就不该有块）；
 *  2. 每块恰好声明 `--sched-*` 六变量，且深/浅两档键集完全一致；
 *  3. 全文无裸 hex、无 `--md-sys-color-*`（不动主题令牌，亮暗两套基座才能叠加）；
 *  4. 每块的值与 `displayPreset.ts` 的色表逐值相等（样式与 TS 单一真源）。
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { SCHED_VARS, presetColorMap, toSvgPalette, type PresetColorMap } from './displayPreset'

const CSS_PATH = resolve(__dirname, 'display-preset.css')
const css = readFileSync(CSS_PATH, 'utf8')
/** 去注释后再解析块（注释里有花括号外的一切，留着会把注释粘进选择器） */
const cssNoComment = css.replace(/\/\*[\s\S]*?\*\//g, '')

interface CssBlock {
  selector: string
  body: string
}

/** 极简块解析：本文件没有 @media/@supports 嵌套块，正则够用 */
function parseBlocks(source: string): CssBlock[] {
  const blocks: CssBlock[] = []
  const re = /([^{}]+)\{([^{}]*)\}/g
  let m: RegExpExecArray | null
  while ((m = re.exec(source)) !== null) {
    blocks.push({ selector: m[1].trim().replace(/\s+/g, ' '), body: m[2] })
  }
  return blocks
}

const BLOCKS = parseBlocks(cssNoComment)

/** 块内声明 → [变量名, 值]（顺序保持文件写法） */
function declarations(block: CssBlock): Array<[string, string]> {
  const out: Array<[string, string]> = []
  for (const chunk of block.body.split(';')) {
    const trimmed = chunk.trim()
    if (!trimmed) continue
    const idx = trimmed.indexOf(':')
    if (idx < 0) continue
    out.push([trimmed.slice(0, idx).trim(), trimmed.slice(idx + 1).trim()])
  }
  return out
}

/** 按选择器精确取块（选择器片段互含，不能用 includes 找） */
function blockOf(selector: string): CssBlock {
  const hits = BLOCKS.filter((b) => b.selector === selector)
  if (hits.length !== 1) throw new Error(`选择器 ${selector} 命中 ${hits.length} 个块，期望 1 个`)
  return hits[0]
}

/** 块内声明 → 变量表 */
function mapOf(selector: string): PresetColorMap {
  const map: Record<string, string> = {}
  for (const [name, value] of declarations(blockOf(selector))) map[name] = value
  return map as PresetColorMap
}

const EYE_DARK = '.sched-preset-eye'
const EYE_LIGHT = ':root[data-hl="light"] .sched-preset-eye'
const MONO_DARK = '.sched-preset-mono'
const MONO_LIGHT = ':root[data-hl="light"] .sched-preset-mono'
const ALL_SELECTORS = [EYE_DARK, EYE_LIGHT, MONO_DARK, MONO_LIGHT]

describe('display-preset.css 结构与口径', () => {
  it('文件存在且非空，注释写明 W6 口径与只覆盖 --sched-*', () => {
    expect(css.length).toBeGreaterThan(500)
    expect(css).toContain('§W6')
    expect(css).toContain('--sched-*')
    expect(css).toContain('对比度取值依据')
  })

  it('恰好四块：护眼/黑白 × 深底/白底（整体块数=4，没有多余样式）', () => {
    expect(BLOCKS).toHaveLength(4)
    expect(BLOCKS.map((b) => b.selector).sort()).toEqual([...ALL_SELECTORS].sort())
    // 常规档只在 TS 里返回空串，样式表不许自造 .sched-preset-normal
    expect(css).not.toContain('.sched-preset-normal')
  })

  it('每块恰好覆盖 --sched-* 六变量，且深/浅两档键集一致', () => {
    for (const [darkSel, lightSel] of [
      [EYE_DARK, EYE_LIGHT],
      [MONO_DARK, MONO_LIGHT],
    ]) {
      const expected = [...SCHED_VARS].sort()
      expect(Object.keys(mapOf(darkSel)).sort()).toEqual(expected)
      expect(Object.keys(mapOf(lightSel)).sort()).toEqual(expected)
      for (const name of SCHED_VARS) {
        expect(declarations(blockOf(darkSel)).filter(([n]) => n === name)).toHaveLength(1)
        expect(declarations(blockOf(lightSel)).filter(([n]) => n === name)).toHaveLength(1)
      }
    }
  })

  it('只声明 --sched-* 变量，不写任何 --md-sys-color-* 主题令牌', () => {
    // 注释里允许提到主题令牌（说明用）；这里查的是声明面
    expect(cssNoComment).not.toContain('--md-sys-color-')
    for (const selector of ALL_SELECTORS) {
      const decls = declarations(blockOf(selector))
      expect(decls).toHaveLength(SCHED_VARS.length)
      for (const [name] of decls) expect(name.startsWith('--sched-')).toBe(true)
    }
  })

  it('无裸 hex：全部 rgb()/rgba()（与 local/no-raw-hex 同口径，样式也能逐值比对）', () => {
    expect(css).not.toMatch(/#[0-9a-fA-F]{3,8}\b/)
    for (const selector of ALL_SELECTORS) {
      for (const [, value] of declarations(blockOf(selector))) {
        expect(value).toMatch(/^rgba?\(\s*\d+\s*,\s*\d+\s*,\s*\d+\s*(?:,\s*[\d.]+\s*)?\)$/)
      }
    }
  })

  it('值必须与 displayPreset.ts 色表逐值相等（样式/TS 单一真源，改一处即报错）', () => {
    expect(mapOf(EYE_DARK)).toEqual(presetColorMap('eyeCare'))
    expect(mapOf(EYE_LIGHT)).toEqual(toSvgPalette('eyeCare'))
    expect(mapOf(MONO_DARK)).toEqual(presetColorMap('mono'))
    expect(mapOf(MONO_LIGHT)).toEqual(toSvgPalette('mono'))
  })

  it('不越界改基座：schedule.css 里不许出现预设类名（并发线冲突守卫）', () => {
    const base = readFileSync(resolve(__dirname, 'schedule.css'), 'utf8')
    expect(base).not.toContain('.sched-preset-')
  })
})
