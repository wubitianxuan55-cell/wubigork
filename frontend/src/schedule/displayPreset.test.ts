/**
 * displayPreset.test.ts — 显示预设枚举 / 类名 / 三套色表（W6 纯数据层）
 *
 * 口径：`进度计划/gaea-schedule-zpert8-uiux-port-plan-20261114.md` §W6
 * 「护眼与黑白两个显示预设，复用 gaea 主题令牌」+ §6「令牌适配，不硬编码斑马色值」。
 *
 * 用例分四组，都是纯函数、零 DOM：
 *  1. 枚举与回落（未知值不许漏进 DOM 类名）；
 *  2. 类名映射（常规=空串，其余一对一）；
 *  3. 三套色表结构（键集一致、常规档与 schedule.css 基座逐值相等）；
 *  4. 颜色性质（黑白纯灰阶、护眼饱和度更低、文字级对比度 ≥4.5:1）。
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import {
  DISPLAY_PRESETS,
  SCHED_VARS,
  isDisplayPreset,
  normalizePreset,
  presetClass,
  presetColorMap,
  toSvgPalette,
  type PresetColorMap,
} from './displayPreset'

/** 解析 `rgb(r, g, b)` / `rgba(r, g, b, a)` → [r, g, b, a]（本模块只产出这两种写法） */
function parseColor(value: string): [number, number, number, number] {
  const m = /^rgba?\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)\s*(?:,\s*([\d.]+)\s*)?\)$/.exec(value.trim())
  if (!m) throw new Error(`非 rgb()/rgba() 色值：${value}`)
  return [Number(m[1]), Number(m[2]), Number(m[3]), m[4] === undefined ? 1 : Number(m[4])]
}

/** sRGB 分量线性化（WCAG 2.1 定义） */
function linear(channel: number): number {
  const s = channel / 255
  return s <= 0.04045 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4)
}

/** 相对亮度（省略 alpha，底纹另行按叠加说明，不当前景图形判） */
function luminance(value: string): number {
  const [r, g, b] = parseColor(value)
  return 0.2126 * linear(r) + 0.7152 * linear(g) + 0.0722 * linear(b)
}

/** 对比度（WCAG 2.1：(亮+0.05)/(暗+0.05)） */
function contrast(a: string, b: [number, number, number]): number {
  const la = luminance(a)
  const lb = luminance(`rgb(${b[0]}, ${b[1]}, ${b[2]})`)
  const hi = Math.max(la, lb)
  const lo = Math.min(la, lb)
  return (hi + 0.05) / (lo + 0.05)
}

/** HSV 饱和度（max=0 时按 0 处理；灰阶必为 0） */
function saturation(value: string): number {
  const [r, g, b] = parseColor(value)
  const max = Math.max(r, g, b)
  const min = Math.min(r, g, b)
  return max === 0 ? 0 : (max - min) / max
}

/** 参照底色：深底取 12 套主题里最浅的暗容器高，白底取石板亮容器 */
const DARK_BASE: [number, number, number] = [31, 32, 52]
const LIGHT_BASE: [number, number, number] = [241, 245, 249]

/** 承载文字/图形的前景变量（周末/网格是同底色叠合的底纹，不参与该判定） */
const FOREGROUND_VARS = [
  '--sched-critical',
  '--sched-normal',
  '--sched-link',
  '--sched-group',
] as const

/** 两个预设 × 两档：深底档走 presetColorMap，白底档走 toSvgPalette */
const PRESET_MAPS: ReadonlyArray<{ preset: 'eyeCare' | 'mono'; range: 'dark' | 'light'; map: PresetColorMap }> = [
  { preset: 'eyeCare', range: 'dark', map: presetColorMap('eyeCare') },
  { preset: 'eyeCare', range: 'light', map: toSvgPalette('eyeCare') },
  { preset: 'mono', range: 'dark', map: presetColorMap('mono') },
  { preset: 'mono', range: 'light', map: toSvgPalette('mono') },
]

describe('预设枚举与回落', () => {
  it('DISPLAY_PRESETS 三项且顺序为 常规→护眼→黑白，中文名/说明都不空', () => {
    expect(DISPLAY_PRESETS.map((p) => p.id)).toEqual(['normal', 'eyeCare', 'mono'])
    expect(DISPLAY_PRESETS.map((p) => p.label)).toEqual(['常规', '护眼', '黑白'])
    for (const info of DISPLAY_PRESETS) expect(info.desc.trim().length).toBeGreaterThan(8)
  })

  it('isDisplayPreset 只认三个已知值（不做大小写/空白兼容）', () => {
    expect(DISPLAY_PRESETS.every((p) => isDisplayPreset(p.id))).toBe(true)
    for (const bad of ['', 'eye', 'EyeCare', 'mono ', ' monochrome', 'bogus', null, undefined, 0, {}, []]) {
      expect(isDisplayPreset(bad)).toBe(false)
    }
  })

  it('normalizePreset：未知/空值一律回落 normal', () => {
    expect(normalizePreset('eyeCare')).toBe('eyeCare')
    expect(normalizePreset('mono')).toBe('mono')
    for (const bad of ['bogus', '', null, undefined, 'EYECARE']) {
      expect(normalizePreset(bad)).toBe('normal')
    }
  })
})

describe('presetClass 类名映射', () => {
  it('常规=空串（不加类=零影响），护眼/黑白各自固定类名', () => {
    expect(presetClass('normal')).toBe('')
    expect(presetClass('eyeCare')).toBe('sched-preset-eye')
    expect(presetClass('mono')).toBe('sched-preset-mono')
  })

  it('未知值回落常规：类名也为空串（脏偏好不会往 DOM 写野类名）', () => {
    for (const bad of ['bogus', '', null, undefined, 'sched-preset-eye']) {
      expect(presetClass(bad)).toBe('')
    }
  })
})

describe('三套色表结构', () => {
  it('键集一致：三套 presetColorMap 与三套 toSvgPalette 都恰好覆盖 SCHED_VARS 六项', () => {
    const expected = [...SCHED_VARS].sort()
    expect(SCHED_VARS).toHaveLength(6)
    expect(new Set(SCHED_VARS).size).toBe(6) // 无重复
    for (const preset of ['normal', 'eyeCare', 'mono'] as const) {
      expect(Object.keys(presetColorMap(preset)).sort()).toEqual(expected)
      expect(Object.keys(toSvgPalette(preset)).sort()).toEqual(expected)
    }
  })

  it('每个签发的表都是新对象（调用方改值不污染模块内常量）', () => {
    const a = presetColorMap('mono')
    a['--sched-critical'] = 'rgb(1, 2, 3)'
    expect(presetColorMap('mono')['--sched-critical']).toBe('rgb(226, 226, 226)')
    const b = toSvgPalette('mono')
    b['--sched-critical'] = 'rgb(1, 2, 3)'
    expect(toSvgPalette('mono')['--sched-critical']).toBe('rgb(24, 24, 24)')
  })

  it('常规档两档 = schedule.css 基座原值（基座改色时本用例先炸，防预设表悄悄漂移）', () => {
    const dark = presetColorMap('normal')
    const light = toSvgPalette('normal')
    // schedule.css:6-13 的 :root 缺省档
    expect(dark['--sched-critical']).toBe('rgb(240, 72, 72)')
    expect(dark['--sched-normal']).toBe('rgb(77, 141, 240)')
    expect(dark['--sched-link']).toBe('rgb(139, 152, 171)')
    expect(dark['--sched-group']).toBe('rgb(213, 218, 226)')
    expect(dark['--sched-weekend']).toBe('rgba(148, 163, 184, 0.14)')
    expect(dark['--sched-grid']).toBe('rgba(148, 163, 184, 0.16)')
    // schedule.css:18-23 的 :root[data-hl="light"] 档
    expect(light['--sched-critical']).toBe('rgb(224, 32, 32)')
    expect(light['--sched-normal']).toBe('rgb(30, 111, 217)')
    expect(light['--sched-link']).toBe('rgb(148, 163, 184)')
    expect(light['--sched-group']).toBe('rgb(31, 41, 55)')
  })

  it('色值与 schedule.css 的真实 hex 逐项等价（变量名/含义对得上，不是自说自话）', () => {
    const css = readFileSync(resolve(__dirname, 'schedule.css'), 'utf8')
    // 把基座 hex 抽成 rgb 三元组，与本模块常规档两份表对照
    const hex2rgb = (hex: string): [number, number, number] => [
      parseInt(hex.slice(1, 3), 16),
      parseInt(hex.slice(3, 5), 16),
      parseInt(hex.slice(5, 7), 16),
    ]
    const readVar = (block: string, name: string): string => {
      const m = new RegExp(`${name}:\\s*(#[0-9a-f]{6})`).exec(block)
      if (!m) throw new Error(`schedule.css 里找不到 ${name}`)
      return m[1]
    }
    const rootBlock = css.slice(
      css.search(/^:root \{/m),
      css.search(/^:root\[data-hl="light"\] \{/m),
    )
    const lightBlock = css.slice(
      css.search(/^:root\[data-hl="light"\] \{/m),
      css.search(/^\/\* ── 页面骨架/m),
    )
    const dark = presetColorMap('normal')
    const light = toSvgPalette('normal')
    for (const name of ['--sched-critical', '--sched-normal', '--sched-link', '--sched-group'] as const) {
      const [dr, dg, db] = hex2rgb(readVar(rootBlock, name))
      expect(parseColor(dark[name]).slice(0, 3)).toEqual([dr, dg, db])
      const [lr, lg, lb] = hex2rgb(readVar(lightBlock, name))
      expect(parseColor(light[name]).slice(0, 3)).toEqual([lr, lg, lb])
    }
  })

  it('未知值回落 normal（色表与常规档逐键相等，不是空表/全黑）', () => {
    expect(presetColorMap('bogus')).toEqual(presetColorMap('normal'))
    expect(toSvgPalette('bogus')).toEqual(toSvgPalette('normal'))
    expect(toSvgPalette(null)).toEqual(toSvgPalette('normal'))
  })
})

describe('颜色性质：黑白纯灰阶', () => {
  it('黑白档两档六个变量全是 R=G=B（含底纹 alpha 项）', () => {
    for (const { preset, range, map } of PRESET_MAPS) {
      if (preset !== 'mono') continue
      for (const name of SCHED_VARS) {
        const [r, g, b] = parseColor(map[name])
        expect({ range, name, rgb: [r, g, b] }).toEqual({ range, name, rgb: [r, r, r] })
        expect(saturation(map[name])).toBe(0)
      }
    }
  })

  it('黑白档亮度阶梯互不相同（关键/正常/依赖线/汇总条四档可辨，不是一片同灰）', () => {
    for (const map of [presetColorMap('mono'), toSvgPalette('mono')]) {
      const lums = FOREGROUND_VARS.map((name) => luminance(map[name]))
      expect(new Set(lums.map((l) => l.toFixed(4))).size).toBe(4)
    }
  })
})

describe('颜色性质：护眼饱和度低于常规', () => {
  it('深底档与白底档的四个前景变量，饱和度逐键低于常规档', () => {
    for (const pair of [
      { eye: presetColorMap('eyeCare'), normal: presetColorMap('normal') },
      { eye: toSvgPalette('eyeCare'), normal: toSvgPalette('normal') },
    ]) {
      for (const name of FOREGROUND_VARS) {
        expect(saturation(pair.eye[name])).toBeLessThan(saturation(pair.normal[name]))
      }
    }
  })

  it('护眼整体平均饱和度低于常规（两支关键色的差距最明显）', () => {
    const avg = (map: PresetColorMap): number =>
      FOREGROUND_VARS.reduce((sum, name) => sum + saturation(map[name]), 0) / FOREGROUND_VARS.length
    expect(avg(presetColorMap('eyeCare'))).toBeLessThan(avg(presetColorMap('normal')))
    expect(avg(toSvgPalette('eyeCare'))).toBeLessThan(avg(toSvgPalette('normal')))
    expect(saturation(presetColorMap('eyeCare')['--sched-critical'])).toBeLessThan(
      saturation(presetColorMap('normal')['--sched-critical']),
    )
    expect(saturation(toSvgPalette('eyeCare')['--sched-normal'])).toBeLessThan(
      saturation(toSvgPalette('normal')['--sched-normal']),
    )
  })
})

describe('颜色性质：对比度达标（文字级 ≥4.5:1，图形级 ≥3:1 由 4.5 兜住）', () => {
  it('护眼/黑白两档四个前景变量对各自底色都 ≥4.5:1', () => {
    for (const { preset, range, map } of PRESET_MAPS) {
      const base = range === 'dark' ? DARK_BASE : LIGHT_BASE
      for (const name of FOREGROUND_VARS) {
        // 期望对象里带上档位/变量名：失败输出直接指出是哪一项不达标
        expect({ preset, range, name, ratio: contrast(map[name], base) >= 4.5 }).toEqual({
          preset,
          range,
          name,
          ratio: true,
        })
      }
    }
  })

  it('底纹 alpha ≤0.18（周末/坐标纸只做分区提示，不压前景图形）', () => {
    for (const { map } of PRESET_MAPS) {
      for (const name of ['--sched-weekend', '--sched-grid'] as const) {
        const alpha = parseColor(map[name])[3]
        expect(alpha).toBeGreaterThan(0)
        expect(alpha).toBeLessThanOrEqual(0.18)
      }
    }
  })
})
