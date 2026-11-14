/**
 * sched-zebra.css.test.ts — 「纯照斑马外观密度」样式层的纯文本守卫（A 线）
 *
 * 为什么要有这个文件：`sched-zebra.css` 是纯样式表，组件测试照不到它；令牌少一条、
 * 混进一处光效、某个 hover 忘了排除 disabled、或者动效降级只挂了系统入口——界面上
 * 都「还能看」，直到换主题、开减弱动效、或者用键盘 Tab 过去才暴露。这里把口径钉成
 * 文本断言，让漂移以用例失败的形式出现。
 *
 * 断言口径来源（`进度计划/gaea-schedule-zpert8-uiux-re-20261114.md`）：
 *   line 841 §13.4/§17.2 表头 #F8FAFC/#4E5969、网格线 #E2E8F0、字号 14px、行高 30px
 *   line 473 工具按钮圆角 4px；line 364 网格内编辑零内边距（不撑高行）
 *   line 370-371 两代视觉并存：要密度，不要「旧」也不要「飘」（常规面零抬升）
 *
 * 裸 hex 判据与 eslint `local/no-raw-hex` 同源：`var(--x, #fallback)` 的回退段是
 * 令牌适配的一部分，不算裸写；`url(#id)` 不是颜色。其余一律算漂移。
 */
import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const CSS_PATH = resolve(__dirname, 'sched-zebra.css')
const CSS = existsSync(CSS_PATH) ? readFileSync(CSS_PATH, 'utf8') : ''

// ─────────────────────────────────────────────────────────────────────
// 极简 CSS 解析（本仓样式表无 CSS 嵌套，只有 @media/@supports）
// ─────────────────────────────────────────────────────────────────────

interface CssRule {
  selector: string
  body: string
  atChain: string[]
}

function stripComments(source: string): string {
  return source.replace(/\/\*[\s\S]*?\*\//g, '')
}

function parseRules(rawSource: string): CssRule[] {
  const source = stripComments(rawSource)
  const rules: CssRule[] = []
  type Frame =
    | { kind: 'at'; prelude: string }
    | { kind: 'rule'; selector: string; start: number; atChain: string[] }
  const frames: Frame[] = []
  let buf = ''
  for (let i = 0; i < source.length; i++) {
    const ch = source[i]
    if (ch === '{') {
      const prelude = buf.trim()
      buf = ''
      if (prelude.startsWith('@')) {
        frames.push({ kind: 'at', prelude })
      } else {
        const atChain = frames
          .filter((f): f is { kind: 'at'; prelude: string } => f.kind === 'at')
          .map((f) => f.prelude)
        frames.push({ kind: 'rule', selector: prelude.replace(/\s+/g, ' '), start: i + 1, atChain })
      }
    } else if (ch === '}') {
      const frame = frames.pop()
      buf = ''
      if (frame && frame.kind === 'rule') {
        rules.push({ selector: frame.selector, body: source.slice(frame.start, i), atChain: frame.atChain })
      }
    } else if (ch === ';') {
      buf = ''
    } else {
      buf += ch
    }
  }
  return rules
}

const RULES = parseRules(CSS)

/** 选择器拆成逗号片段（`.a:hover, .a:focus-visible` → 两段） */
function selectorParts(selector: string): string[] {
  return selector
    .split(',')
    .map((p) => p.trim().replace(/\s+/g, ' '))
    .filter(Boolean)
}

/** 块体拆声明（只取 `名: 值`） */
function declarations(body: string): Array<[string, string]> {
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

/** 声明里是否有某个属性（含 `background` / `background-color` 这类前缀族） */
function declares(body: string, prop: string): boolean {
  return declarations(body).some(([name]) => name === prop || name.startsWith(`${prop}-`))
}

/** `var(--x, <回退>)` 的回退段与 `url(#id)` 都不是「裸用色值」，先掩掉 */
function maskNonColorHex(source: string): string {
  const withoutUrl = source.replace(/url\([^)]*\)/g, 'url(#masked)')
  const out: string[] = []
  let i = 0
  while (i < withoutUrl.length) {
    const start = withoutUrl.indexOf('var(', i)
    if (start < 0) {
      out.push(withoutUrl.slice(i))
      break
    }
    out.push(withoutUrl.slice(i, start))
    let depth = 0
    let comma = -1
    let j = start + 3 // 指向 '('
    for (; j < withoutUrl.length; j++) {
      const c = withoutUrl[j]
      if (c === '(') depth++
      else if (c === ')') {
        depth--
        if (depth === 0) break
      } else if (c === ',' && depth === 1 && comma < 0) {
        comma = j
      }
    }
    out.push(withoutUrl.slice(start, comma < 0 ? j + 1 : comma + 1))
    i = j + 1
  }
  return out.join('')
}

const RAW_HEX = /#[0-9a-fA-F]{3,8}\b/g

function rawHexHits(source: string): string[] {
  return maskNonColorHex(stripComments(source)).match(RAW_HEX) ?? []
}

/** 降级作用域：@media (prefers-reduced-motion: reduce) 内，或选择器带 .ui-reduced-motion */
function isReducedScope(rule: CssRule): boolean {
  if (rule.atChain.some((prelude) => /prefers-reduced-motion\s*:\s*reduce/.test(prelude))) return true
  return /\.ui-reduced-motion\b/.test(rule.selector)
}

/** 块内是否把动效归零 */
function zeroesMotion(body: string): boolean {
  return declarations(body).some(([name, value]) => {
    const v = value.toLowerCase()
    if (name === 'animation' || name === 'transition') {
      return /^none\b/.test(v) || /(^|\s)0m?s(\s|$)/.test(v)
    }
    if (name === 'animation-duration' || name === 'transition-duration') return /^0m?s\b/.test(v)
    return false
  })
}

// ─────────────────────────────────────────────────────────────────────
// 口径清单
// ─────────────────────────────────────────────────────────────────────

/** 必须存在的令牌：值给定=逐字相等；值 null=只要求存在且非空 */
const REQUIRED_TOKENS: Array<[string, string | null]> = [
  ['--sched-radius-sm', '4px'], // line 473 工具按钮圆角 4px
  ['--sched-radius-md', '4px'],
  ['--sched-radius-lg', '6px'],
  ['--sched-elev-1', 'none'], // 常规面零抬升（line 370-371 老壳实底为骨）
  ['--sched-elev-2', null], // 只给浮层/菜单一档
  ['--sched-motion', '100ms'],
  ['--sched-motion-ease', 'ease-out'],
  ['--sched-hairline', null],
  ['--sched-grid-line', null], // line 841 网格线 #E2E8F0
  ['--sched-head-bg', null], // line 841 表头底 #F8FAFC
  ['--sched-head-ink', null], // line 841 表头字 #4E5969
  ['--sched-font-table', '14px'], // line 841 字号 14
  ['--sched-font-meta', '12px'],
  ['--sched-row-h', '30px'], // line 841 行高 30
]

/** 零光效：这五样一个都不许出现（本文件内即违规） */
const BANNED_DECOR = ['backdrop-filter', '--v3-glow', '--v3-edge', 'linear-gradient', 'text-shadow']

/** 交互态钩子（六个必查项） */
const HOOKS = [
  '.sched-ribbon-command',
  '.sched-menu-item',
  '.sched-gantt-row',
  '.sched-gantt-cell.sched-th',
  '.sched-statusbar',
  '.sched-hit-menu-layer',
] as const

/** 找到「选择器片段含 hook + 含伪类 + 声明了某属性」的证据 */
function findStates(hook: string, pseudo: string, prop: string): string[] {
  const evidence: string[] = []
  for (const rule of RULES) {
    for (const part of selectorParts(rule.selector)) {
      if (!part.includes(hook)) continue
      if (!part.includes(pseudo)) continue
      if (!declares(rule.body, prop)) continue
      evidence.push(part)
    }
  }
  return evidence
}

// ─────────────────────────────────────────────────────────────────────
// 守卫
// ─────────────────────────────────────────────────────────────────────

describe('sched-zebra.css 已落地', () => {
  it('文件存在且非空', () => {
    expect(CSS.length, `A 线样式表缺失：frontend/src/schedule/sched-zebra.css`).toBeGreaterThan(0)
  })
})

describe('令牌清单（补齐 schedule.css 已引用但缺失的定义）', () => {
  for (const [name, expected] of REQUIRED_TOKENS) {
    it(`${name} 已定义${expected ? ` = ${expected}` : ''}`, () => {
      const pattern = new RegExp(`${name}\\s*:\\s*([^;]+);`)
      const found = CSS.match(pattern)
      expect(found, `${name} 未在 sched-zebra.css 的 :root 里定义（schedule.css 引用了它，现在是无效值）`).not.toBeNull()
      const value = (found as RegExpMatchArray)[1].trim()
      expect(value.length, `${name} 定义了空值`).toBeGreaterThan(0)
      if (expected) expect(value, `${name} 取值漂移`).toBe(expected)
    })
  }

  it('斑马密度三件套逐值对上证据：行高 30px / 表格字号 14px / 控件圆角 4px', () => {
    const grab = (name: string): string => {
      const m = CSS.match(new RegExp(`${name}\\s*:\\s*([^;]+);`))
      return (m as RegExpMatchArray)[1].trim()
    }
    expect(grab('--sched-row-h')).toBe('30px')
    expect(grab('--sched-font-table')).toBe('14px')
    expect(grab('--sched-radius-sm')).toBe('4px')
  })

  it('色值类令牌一律走 var(--token, #hex) 兜底，不硬绑斑马色值', () => {
    const grab = (name: string): string => {
      const m = CSS.match(new RegExp(`${name}\\s*:\\s*([^;]+);`))
      return (m as RegExpMatchArray)[1].trim()
    }
    for (const name of ['--sched-grid-line', '--sched-head-bg', '--sched-head-ink']) {
      expect(grab(name), `${name} 必须写成 var(--md-sys-color-*, #兜底)`).toContain('var(')
    }
    expect(grab('--sched-hairline')).toContain('1px solid')
    expect(grab('--sched-elev-2')).toContain('rgba(')
  })
})

describe('零光效：实底，不搬 v3 星枢的玻璃/高光线/光晕', () => {
  for (const banned of BANNED_DECOR) {
    it(`不含 ${banned}`, () => {
      expect(CSS.includes(banned), `sched-zebra.css 出现 ${banned}——斑马密度只要实底细线`).toBe(false)
    })
  }
})

describe('零裸 hex：只有 var 兜底段允许 hex', () => {
  it('判据自检：裸写算违规，var 回退与 url(#id) 不算', () => {
    expect(rawHexHits('.a { color: #abc; }')).toEqual(['#abc']) // hex-exempt
    expect(rawHexHits('.a { color: #11223344; }')).toEqual(['#11223344']) // hex-exempt
    expect(rawHexHits('.a { color: var(--md-sys-color-on-surface, #374151); }')).toEqual([])
    expect(rawHexHits('.a { fill: url(#grad); }')).toEqual([])
  })

  it('全文件规则体里除 fallback 外没有 hex', () => {
    const hits = rawHexHits(CSS)
    expect(hits, `出现裸 hex：${hits.join('、')}——改用 --md-sys-color-* / --sched-* 令牌`).toEqual([])
  })
})

describe('交互态：六个钩子的 hover / disabled / focus-visible', () => {
  for (const hook of HOOKS) {
    it(`${hook} 有 hover 反馈`, () => {
      const hit = findStates(hook, ':hover', 'background')
      expect(hit.length, `${hook} 缺 hover 底色`).toBeGreaterThan(0)
    })

    it(`${hook} 有 disabled 口径（opacity + not-allowed）`, () => {
      const hit = findStates(hook, ':disabled', 'opacity')
      expect(hit.length, `${hook} 缺 disabled（opacity 0.45）`).toBeGreaterThan(0)
      const cursorHit = findStates(hook, ':disabled', 'cursor')
      expect(cursorHit.length, `${hook} 缺 disabled 的 not-allowed 光标`).toBeGreaterThan(0)
    })

    it(`${hook} 有 focus-visible 焦点环`, () => {
      const hit = findStates(hook, ':focus-visible', 'outline')
      expect(hit.length, `${hook} 缺 :focus-visible + outline`).toBeGreaterThan(0)
    })
  }

  it('焦点环是 2px + 负偏移（栅格内不撑开行高）', () => {
    expect(CSS).toContain('--sched-focus-ring: 2px solid')
    const offsets = RULES.filter((r) =>
      selectorParts(r.selector).some((p) => p.includes(':focus-visible')),
    ).flatMap((r) => declarations(r.body).filter(([n]) => n === 'outline-offset').map(([, v]) => v))
    expect(offsets.length).toBeGreaterThan(0)
    for (const v of offsets) expect(v, '焦点环必须用负偏移，避免撑开 30px 行高').toBe('-2px')
  })

  it('hover 规则都排除 disabled：禁用后不吃任何反馈', () => {
    const offenders: string[] = []
    for (const rule of RULES) {
      for (const part of selectorParts(rule.selector)) {
        if (!part.includes(':hover')) continue
        if (!HOOKS.some((hook) => part.includes(hook))) continue
        if (!part.includes(':not(:disabled)')) offenders.push(part)
      }
    }
    expect(offenders, `这些 hover 规则没排除 disabled：${offenders.join('、')}`).toEqual([])
  })

  it('选中/按下走主色淡底（12% / 16%），不引入光晕', () => {
    const tokens = CSS.match(/--sched-sel-bg:\s*([^;]+);/)
    const press = CSS.match(/--sched-press-bg:\s*([^;]+);/)
    expect((tokens as RegExpMatchArray)[1]).toContain('12%')
    expect((press as RegExpMatchArray)[1]).toContain('16%')
    expect(CSS).toContain('.sched-gantt-row.sched-row-selected')
  })
})

describe('动效：只动颜色类属性，双入口归零', () => {
  it('transition 只覆盖 background / border-color / color', () => {
    const allowed = new Set(['background', 'background-color', 'border-color', 'color'])
    const offenders: string[] = []
    for (const rule of RULES) {
      for (const [name, value] of declarations(rule.body)) {
        if (name !== 'transition') continue
        for (const segment of value.split(',')) {
          const prop = segment.trim().split(/\s+/)[0]
          if (!prop || prop === 'none') continue
          if (!allowed.has(prop)) offenders.push(prop)
        }
      }
    }
    expect(offenders, `transition 混入非颜色属性：${offenders.join('、')}`).toEqual([])
  })

  it('系统偏好入口 @media (prefers-reduced-motion: reduce) 有归零规则', () => {
    const media = RULES.filter(
      (r) => r.atChain.some((p) => /prefers-reduced-motion\s*:\s*reduce/.test(p)) && zeroesMotion(r.body),
    )
    expect(media.length, '缺 @media (prefers-reduced-motion: reduce) 内的归零规则').toBeGreaterThan(0)
  })

  it('应用内开关入口 .ui-reduced-motion 有归零规则', () => {
    const cls = RULES.filter((r) => /\.ui-reduced-motion\b/.test(r.selector) && zeroesMotion(r.body))
    expect(cls.length, '缺 .ui-reduced-motion 作用域的归零规则（应用内减弱动效会失效）').toBeGreaterThan(0)
  })

  it('降级真的覆盖到了基础动效（不是摆设块）', () => {
    const targets = new Set<string>()
    for (const rule of RULES.filter((r) => isReducedScope(r) && zeroesMotion(r.body))) {
      for (const part of selectorParts(rule.selector)) {
        targets.add(part.replace(/^\.ui-reduced-motion\s+/, '').trim())
      }
    }
    const covered = [...targets].filter((target) =>
      RULES.some(
        (base) =>
          !isReducedScope(base) &&
          selectorParts(base.selector).includes(target) &&
          declarations(base.body).some(
            ([n, v]) => n === 'transition' && !/^none\b/.test(v.toLowerCase()),
          ),
      ),
    )
    expect(covered.length, '降级块里没有任何目标在基础作用域声明过 transition').toBeGreaterThan(0)
  })
})

describe('圆角纪律：控件 4px / 弹层 6px / 表格与画布 0', () => {
  it('表格行、表头格、画布窗格都是方角 0', () => {
    const zeroRadiusSelectors = RULES.filter((r) =>
      declarations(r.body).some(([n, v]) => n === 'border-radius' && v.trim() === '0'),
    ).flatMap((r) => selectorParts(r.selector))
    for (const target of [
      '.sched-gantt-row',
      '.sched-gantt-cell',
      '.sched-gantt-head',
      '.sched-gantt-canvaspane',
      '.sched-network-scroll',
    ]) {
      expect(zeroRadiusSelectors, `${target} 该是方角（斑马网格）`).toContain(target)
    }
  })

  it('浮层用弹层档 6px 令牌', () => {
    const layer = RULES.filter((r) =>
      selectorParts(r.selector).some((p) => p === '.sched-hit-menu-layer'),
    ).flatMap((r) => declarations(r.body).filter(([n]) => n === 'border-radius').map(([, v]) => v))
    expect(layer).toContain('var(--sched-radius-lg)')
  })
})

describe('密度：功能区/文档条/右舷/状态栏都收紧到斑马档', () => {
  const valueOf = (className: string, prop: string): string[] =>
    RULES.filter((r) => selectorParts(r.selector).some((p) => p === className))
      .flatMap((r) => declarations(r.body).filter(([n]) => n === prop).map(([, v]) => v))

  it('功能区组内 gap 4px、命令按钮 padding 2px 6px、控件高 24px', () => {
    expect(valueOf('.sched-ribbon-group', 'gap')).toContain('var(--sched-gap-tight)')
    expect(valueOf('.sched-ribbon-command', 'padding')).toContain('var(--sched-pad-ctl)')
    expect(valueOf('.sched-ribbon-command', 'min-height')).toContain('var(--sched-ctl-h)')
  })

  it('状态栏高 24px、行内控件高 24px、字号 12/13 档', () => {
    expect(valueOf('.sched-statusbar', 'min-height')).toContain('var(--sched-statusbar-h)')
    expect(valueOf('.sched-statusbar', 'font-size')).toContain('var(--sched-font-meta)')
    expect(valueOf('.sched-ribbon-command', 'font-size')).toContain('var(--sched-font-meta)')
    expect(valueOf('.sched-menu-item', 'font-size')).toContain('var(--sched-font-ui)')
  })

  it('表格：行高/字号走令牌，表头底、表头字、网格线各有消费者', () => {
    expect(valueOf('.sched-gantt-row', 'height')).toContain('var(--sched-row-h)')
    expect(valueOf('.sched-gantt-cell', 'font-size')).toContain('var(--sched-font-table)')
    expect(CSS).toContain('--sched-head-bg')
    expect(CSS).toContain('--sched-head-ink')
    expect(CSS).toContain('--sched-grid-line')
    expect(valueOf('.sched-gantt-cell.sched-th', 'color')).toContain('var(--sched-head-ink)')
  })
})
