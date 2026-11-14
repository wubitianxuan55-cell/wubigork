/**
 * zebraDensity.test.ts — 「纯照斑马外观密度」守卫（并发线 C：只写断言，不碰产品代码）
 *
 * 为什么要有这个文件：本轮拍板的是**形状语言**（实底 / 细线 / 30px 行高 / 14px 表格字 /
 * 4px 工具按钮圆角 / 零装饰光效），而不是某个具体像素。形状语言的特点是「错了也看不出来」：
 * 行高在 CSS 里写 30、在 `ganttUtil.ts` 里写 29，滚动到底才抖 1px；令牌少一个，
 * `var(--x)` 静默失效回落 initial；光效从 v3 foundation 漏进来一条 `backdrop-filter`，
 * 只在弱机滚动时掉帧。这里把口径钉成断言，让漂移以用例红灯出现，而不是以截图返工出现。
 *
 * 只读文本，不写任何产品文件（A 线写 `sched-zebra.css`，B 线补键盘/aria）。
 *
 * 口径来源：
 *   进度计划/gaea-schedule-zpert8-uiux-re-20261114.md §13.4（line 353–371 视觉参数实测）、
 *   §17.2（line 740–761 行高）、line 473（工具按钮圆角 4px）、line 22/112（交互密度）；
 *   进度计划/gaea-schedule-zebra-density-2026-11.md（本轮审计与「刻意偏离」清单）。
 *
 * ── 三条容忍口径（避免把「正确的实现」误判成漂移）──────────────────
 *  1. **状态规则可分散在两个真源**：A 线可以全写在 `sched-zebra.css`，也可以就地补
 *     `schedule.css`；本文件把两个文件都解析后合并判定，不预设归属。
 *  2. **hover 可以挂在子项上**：浮层类容器（`.sched-hit-menu-layer`、`.sched-statusbar`
 *     这类包一层的钩子）的悬停反馈实际落在子元素上，
 *     故「选择器含该钩子 + 含 `:hover`」即算成立（`.X .ant-menu-item:hover` 也过）。
 *     disabled 同理接受 `[aria-disabled=…]` / `.is-disabled` / `-disabled` 类后缀
 *     ——antd 与原生控件用的是两套写法。focus 接受 `:focus-within`，
 *     与 `uiAlignment.test.ts` 的焦点环容忍口径一致。
 *  3. **有选择器还不算数**：命中块里必须至少写一条声明，否则等于「占了名字没画反馈」。
 *     只判「有声明」，不判具体属性——具体属性由 A 线的视觉稿定，守卫不越权。
 */
import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const SCHEDULE_DIR = __dirname
/** A 线新增的斑马密度样式表（令牌定义 + 交互态）；未落地时本文件给清晰失败，不崩 undefined */
const ZEBRA_SHEET = 'sched-zebra.css'
/** 既有真源：允许 A 线在这里最小修补交互态（本测试两处都读） */
const LEGACY_SHEET = 'schedule.css'
/** 行高/行高的 TS 单一真源（与 CSS 同值，防两处漂移） */
const GANTT_UTIL = resolve(SCHEDULE_DIR, 'gantt/ganttUtil.ts')
const TIMESCALE = resolve(SCHEDULE_DIR, 'timescale.ts')

// ─────────────────────────────────────────────────────────────────────
// 读文件
// ─────────────────────────────────────────────────────────────────────

function readSheet(rel: string): string {
  const path = resolve(SCHEDULE_DIR, rel)
  return existsSync(path) ? readFileSync(path, 'utf8') : ''
}

/** A 线未落地时抛**带路径的错误**，而不是让后续断言吃 `undefined` / 空串静默通过 */
function requireZebra(): string {
  const source = readSheet(ZEBRA_SHEET)
  if (!existsSync(resolve(SCHEDULE_DIR, ZEBRA_SHEET))) {
    throw new Error(
      `A 线样式表未落地：期望 frontend/src/schedule/${ZEBRA_SHEET} 存在。` +
        '并发约定是本守卫只读不写，等 A 线提交 sched-zebra.css 后复跑本用例即可；' +
        '在此之前「令牌 / 零光效 / 降级双入口 / 交互态」四组断言都会以本消息失败。',
    )
  }
  if (!source.trim()) {
    throw new Error(`${ZEBRA_SHEET} 存在但为空：A 线只落了空文件，令牌与交互态还没写进来。`)
  }
  return source
}

const ZEBRA_RAW = readSheet(ZEBRA_SHEET)
const LEGACY_RAW = readSheet(LEGACY_SHEET)

/** 去注释后扫描：注释里写「不用 backdrop-filter」不算违规，也不能算「已实现」 */
function stripComments(source: string): string {
  return source.replace(/\/\*[\s\S]*?\*\//g, '')
}

const ZEBRA = stripComments(ZEBRA_RAW)
const LEGACY = stripComments(LEGACY_RAW)

// ─────────────────────────────────────────────────────────────────────
// 极简 CSS 解析（本仓样式表无 CSS 嵌套，只有 @media/@supports）
// ─────────────────────────────────────────────────────────────────────

interface CssRule {
  selector: string
  body: string
  file: string
}

function parseRules(source: string, file: string): CssRule[] {
  const rules: CssRule[] = []
  type Frame = { kind: 'at' } | { kind: 'rule'; selector: string; start: number }
  const frames: Frame[] = []
  let buf = ''
  for (let i = 0; i < source.length; i++) {
    const ch = source[i]
    if (ch === '{') {
      const prelude = buf.trim()
      buf = ''
      if (prelude.startsWith('@')) frames.push({ kind: 'at' })
      else frames.push({ kind: 'rule', selector: prelude.replace(/\s+/g, ' '), start: i + 1 })
    } else if (ch === '}') {
      const frame = frames.pop()
      buf = ''
      if (frame && frame.kind === 'rule') {
        rules.push({ selector: frame.selector, body: source.slice(frame.start, i), file })
      }
    } else if (ch === ';') {
      buf = ''
    } else {
      buf += ch
    }
  }
  return rules
}

const ALL_RULES: CssRule[] = [
  ...parseRules(ZEBRA, ZEBRA_SHEET),
  ...parseRules(LEGACY, LEGACY_SHEET),
]

/** 选择器拆逗号片段（`.a:hover, .a:focus-visible` → 两段） */
function selectorParts(selector: string): string[] {
  return selector
    .split(',')
    .map((p) => p.trim().replace(/\s+/g, ' '))
    .filter(Boolean)
}

/** 块体拆声明（只取 `名: 值`） */
function declarationsOf(body: string): Array<[string, string]> {
  const out: Array<[string, string]> = []
  for (const chunk of body.split(';')) {
    const trimmed = chunk.trim()
    if (!trimmed) continue
    const idx = trimmed.indexOf(':')
    if (idx < 0) continue
    out.push([trimmed.slice(0, idx).trim().toLowerCase(), trimmed.slice(idx + 1).trim()])
  }
  return out
}

/** 从样式文本里取某条自定义属性的首个声明值（去掉 `!important` 与尾分号） */
function declaredValue(css: string, name: string): string | null {
  const m = new RegExp(`--${name}\\s*:\\s*([^;}]+)`).exec(css)
  return m ? m[1].trim().replace(/\s*!important$/, '') : null
}

// ─────────────────────────────────────────────────────────────────────
// 1. 令牌：形状语言落在变量上，才谈得上「密度一致」
// ─────────────────────────────────────────────────────────────────────

/** 令牌 → 期望值（`null` = 只要求存在，值随视觉稿） */
const REQUIRED_TOKENS: ReadonlyArray<readonly [string, string | null]> = [
  ['sched-radius-sm', '4px'], // 斑马工具按钮圆角 §14.3 line 473
  ['sched-row-h', '30px'], // 斑马表格行高 §13.4 line 841 / §17.2
  ['sched-font-table', '14px'], // 斑马表格字号 §13.4 line 359
  ['sched-hairline', null], // 细线（表头底/网格线）
  ['sched-grid-line', null], // 网格线（#E2E8F0 的令牌位）
  ['sched-motion', null], // 动效时长（reduced-motion 会归零，但令牌必须在）
]

describe('斑马密度令牌（sched-zebra.css）', () => {
  it(`${ZEBRA_SHEET} 已落地且非空`, () => {
    expect(requireZebra().length, 'A 线文件必须存在且有内容').toBeGreaterThan(200)
  })

  it('六个密度令牌齐全，且三条带值令牌与斑马实测同值', () => {
    const zebra = requireZebra()
    const missing: string[] = []
    const wrongValue: string[] = []
    for (const [name, expected] of REQUIRED_TOKENS) {
      const value = declaredValue(stripComments(zebra), name)
      if (value === null) {
        missing.push(`--${name}`)
        continue
      }
      if (expected !== null && value !== expected) wrongValue.push(`--${name} = ${value}，期望 ${expected}`)
    }
    expect(
      missing,
      `sched-zebra.css 缺令牌：${missing.join('、') || '无'}。` +
        '令牌名写错/漏写时 var(--x) 会静默回落 initial（行高 0、圆角 0、字号继承），屏上看不出报错。',
    ).toEqual([])
    expect(
      wrongValue,
      `sched-zebra.css 令牌取值偏离斑马实测：${wrongValue.join('；') || '无'}。` +
        '口径见 re 文档 §13.4 line 841（行高 30 / 字号 14）与 §14.3 line 473（圆角 4px）。',
    ).toEqual([])
  })
})

// ─────────────────────────────────────────────────────────────────────
// 2. 零光效：本轮明确不采用 v3 星枢的玻璃/高光线/光晕
// ─────────────────────────────────────────────────────────────────────

/** 装饰光效黑名单（命中即违规：斑马密度是实底 + 细线，没有一层是半透明模糊底） */
const GLOW_TOKENS = ['backdrop-filter', '--v3-glow', '--v3-edge', 'linear-gradient', 'text-shadow'] as const

describe('零光效（反面参照 v3/foundation.css 的 glow/edge）', () => {
  it('sched-zebra.css 不含任何装饰光效', () => {
    const zebra = requireZebra()
    const hits = GLOW_TOKENS.filter((token) => stripComments(zebra).includes(token))
    expect(
      hits,
      `sched-zebra.css 出现了 v3 星枢式光效：${hits.join('、') || '无'}。` +
        '本轮口径是「纯照斑马外观密度」：实底 + 1px 细线，不引 var(--v3-glow)/var(--v3-edge) 的' +
        'box-shadow 光晕、不做 backdrop-filter 玻璃底；网格线请用 repeating-linear-gradient 之外' +
        '的细线写法（或 --sched-grid-line 单线）。',
    ).toEqual([])
  })

  it('schedule.css 未新增 --v3-glow（历史值放行，只守「不许再引 starhub 令牌」）', () => {
    expect(
      LEGACY.includes('--v3-glow'),
      'schedule.css 引了 --v3-glow：历史里已有的 linear-gradient/grid 底纹按「允许既有历史值」放行，' +
        '但星枢光晕令牌一条都不能新进（它是 v3 玻璃风的入口）。',
    ).toBe(false)
  })
})

// ─────────────────────────────────────────────────────────────────────
// 3. 密度一致性：ROW_H（TS）与 --sched-row-h（CSS）必须同值
// ─────────────────────────────────────────────────────────────────────

/** 读 TS 常量；格式变了就报「守卫要同步」，而不是让断言空过 */
function readConstNumber(path: string, pattern: RegExp, label: string): number {
  const source = readFileSync(path, 'utf8')
  const m = pattern.exec(source)
  if (!m) throw new Error(`${label}：${path} 里没匹配到约定写法 ${pattern}。常量写法变了请同步本守卫。`)
  return Number(m[1])
}

describe('密度一致性（防 CSS / TS 两处漂移）', () => {
  it('ganttUtil.ROW_H 与 --sched-row-h 同值', () => {
    const zebra = requireZebra()
    const utilRowH = readConstNumber(GANTT_UTIL, /export const ROW_H\s*=\s*(\d+)/, 'ROW_H')
    const cssRaw = declaredValue(stripComments(zebra), 'sched-row-h')
    expect(cssRaw, '缺 --sched-row-h，无法做同值比对').not.toBeNull()
    const cssRowH = Number(/^([\d.]+)px$/.exec(cssRaw as string)?.[1])
    expect(
      Number.isFinite(cssRowH),
      `--sched-row-h 取值必须写成 px 数字（当前 "${cssRaw}"），否则与 ROW_H 无法同值比对`,
    ).toBe(true)
    expect(
      cssRowH,
      `行高两处不同值：gantt/ganttUtil.ts ROW_H = ${utilRowH}，sched-zebra.css --sched-row-h = ${cssRaw}。` +
        '表格行高走 CSS、画布行推进走 ROW_H，两者一旦分叉，滚到底会累积半行错位（斑马 §17.2 的逐行累加式命中最怕这个）。',
    ).toBe(utilRowH)
  })

  it('双标尺高度仍是 40（本轮刻意保留的偏离，不是漂移）', () => {
    const rowH = readConstNumber(TIMESCALE, /export const GANTT_RULER_ROW_H\s*=\s*(\d+)/, 'GANTT_RULER_ROW_H')
    const rows = readConstNumber(TIMESCALE, /export const GANTT_RULER_ROWS\s*=\s*(\d+)/, 'GANTT_RULER_ROWS')
    expect(
      rowH * rows,
      `双标尺总高变成 ${rowH * rows}px。口径是「刻意偏离：标尺保 40px（两行×20）」` +
        '——斑马表格表头与标尺的对齐方式与 gaea 不同，gaea 保留 40 才能让表格首行与画布首行同基线。' +
        '真要改，请先改 进度计划/gaea-schedule-zebra-density-2026-11.md 的「刻意偏离」清单再动代码。',
    ).toBe(40)
  })
})

// ─────────────────────────────────────────────────────────────────────
// 4. 交互态：六个钩子 × hover / disabled / focus-visible
// ─────────────────────────────────────────────────────────────────────

/**
 * 六钩子清单（名词与产品代码里的 class 一字不差；改名字这里就红）。
 * `tokens` 是同一选择器片段里必须同时出现的类名：`.sched-gantt-cell.sched-th` 是
 * 双类名表头单元格（GanttTable.tsx 的表头写法），只命中 `.sched-gantt-cell` 不算。
 */
const INTERACTION_TARGETS: ReadonlyArray<{ name: string; tokens: readonly string[] }> = [
  { name: '.sched-ribbon-command', tokens: ['.sched-ribbon-command'] },
  { name: '.sched-menu-item', tokens: ['.sched-menu-item'] },
  { name: '.sched-gantt-row', tokens: ['.sched-gantt-row'] },
  { name: '.sched-gantt-cell.sched-th', tokens: ['.sched-gantt-cell', '.sched-th'] },
  { name: '.sched-statusbar', tokens: ['.sched-statusbar'] },
  { name: '.sched-hit-menu-layer', tokens: ['.sched-hit-menu-layer'] },
]

interface StateSpec {
  name: string
  /** 命中判据（容忍口径见文件头第 2 条） */
  test: (part: string) => boolean
}

const STATES: readonly StateSpec[] = [
  { name: 'hover', test: (part) => /:hover\b/.test(part) },
  {
    name: 'disabled',
    test: (part) => /:disabled\b|\[disabled\]|\[aria-disabled|\.is-disabled\b|-disabled\b/.test(part),
  },
  { name: 'focus-visible', test: (part) => /:focus-visible\b|:focus-within\b/.test(part) },
]

/** 找某钩子某状态的证据：返回 `文件 · 选择器`；无证据返回 null */
function findStateRule(tokens: readonly string[], state: StateSpec): string | null {
  for (const rule of ALL_RULES) {
    for (const part of selectorParts(rule.selector)) {
      if (!tokens.every((token) => part.includes(token))) continue
      if (!state.test(part)) continue
      // 占了名字不算数：块里至少一条声明
      if (declarationsOf(rule.body).length === 0) continue
      return `${rule.file} · ${part}`
    }
  }
  return null
}

describe('交互态六钩子 × 三种状态（hover / disabled / focus-visible）', () => {
  it(`${ZEBRA_SHEET} 已落地（交互态可在两个真源里补，但令牌表必须在）`, () => {
    expect(requireZebra().length).toBeGreaterThan(200)
  })

  it('18 组（6 钩子 × 3 状态）全部有规则落地', () => {
    const missing: string[] = []
    for (const target of INTERACTION_TARGETS) {
      for (const state of STATES) {
        if (findStateRule(target.tokens, state) === null) missing.push(`${target.name} × ${state.name}`)
      }
    }
    expect(
      missing,
      `缺 ${missing.length} 组交互态：\n  ${missing.join('\n  ')}\n` +
        `补法：写在 frontend/src/schedule/${ZEBRA_SHEET}（推荐）或 ${LEGACY_SHEET} 就地补，本测试两个真源都读。\n` +
        '为什么要守：这三种状态是「做得出来但看不见」的典型——键盘用户 Tab 过去没有焦点环、' +
        '不可用项点下去没有原因、悬停没有底色，截图全都正常，只有真用户会撞上。\n' +
        '容忍口径：hover 可挂子项（`.X .ant-menu-item:hover` 算过）；disabled 接受 ' +
        '`:disabled` / `[aria-disabled]` / `.is-disabled` / `-disabled` 类后缀；focus 接受 `:focus-within`。',
    ).toEqual([])
  })
})

// ─────────────────────────────────────────────────────────────────────
// 5. 圆角纪律：表格 / 画布是方角，不许吃 --sched-radius-lg
// ─────────────────────────────────────────────────────────────────────

/** 方角领域（斑马：网格与画布没有圆角；圆角只给按钮/浮层） */
const SQUARE_SELECTORS = [
  '.sched-gantt-tablepane',
  '.sched-gantt-canvaspane',
  '.sched-gantt-row',
  '.sched-gantt-cell',
] as const

describe('圆角纪律（表格/画布方角，圆角只给工具按钮与浮层）', () => {
  it('四个方角选择器没有一个命中 border-radius: var(--sched-radius-lg)', () => {
    const violations: string[] = []
    for (const rule of ALL_RULES) {
      const decls = declarationsOf(rule.body)
      const radius = decls.find(([name]) => name === 'border-radius')
      if (!radius) continue
      if (!radius[1].includes('var(--sched-radius-lg)')) continue
      for (const part of selectorParts(rule.selector)) {
        if (SQUARE_SELECTORS.some((sel) => part.includes(sel))) {
          violations.push(`${rule.file} · ${part} → border-radius: ${radius[1]}`)
        }
      }
    }
    expect(
      violations,
      `表格/画布吃到 --sched-radius-lg：\n  ${violations.join('\n  ') || '无'}\n` +
        '斑马的网格与画布是方角（§13.4 表头/网格线是 1px 直角边），大圆角会让行首尾出现白缝、' +
        '表头与表体接缝错开。表格/画布请写 border-radius: 0，圆角只留给工具按钮（4px）与浮层。',
    ).toEqual([])
  })
})

// ─────────────────────────────────────────────────────────────────────
// 6. 降级双入口：系统开关 + gaea 应用内开关，两个都得在
// ─────────────────────────────────────────────────────────────────────

describe('降级双入口（系统 prefers-reduced-motion + 应用内 .ui-reduced-motion）', () => {
  it('两个入口都在 sched-zebra.css 内', () => {
    const zebra = requireZebra()
    const css = stripComments(zebra)
    const missing: string[] = []
    if (!css.includes('prefers-reduced-motion')) missing.push('@media (prefers-reduced-motion: reduce)')
    if (!css.includes('.ui-reduced-motion')) missing.push('.ui-reduced-motion')
    expect(
      missing,
      `sched-zebra.css 缺降级入口：${missing.join('、') || '无'}。` +
        '两个入口是两条独立链路：系统级 `prefers-reduced-motion` 只认操作系统开关，' +
        'gaea 自己「减弱动效」偏好走 `.ui-reduced-motion`（挂在根节点）。只写一个，' +
        '另一半用户开着的仍是动画——而这轮斑马密度本来就没有入场动效，' +
        '降级后应当彻底 0 过渡（--sched-motion 归零）。',
    ).toEqual([])
  })
})
