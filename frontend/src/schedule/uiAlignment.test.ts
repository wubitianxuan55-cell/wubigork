/**
 * uiAlignment.test.ts — 进度板块斑马密度对齐守卫（并发线 C：只写断言，不写产品规则）
 *
 * 为什么要有这个文件：斑马密度对齐是三线并发（A=样式/令牌/交互态，B=键盘与 aria，
 * C=守卫+文档）。A/B 的产物全是「看得见但看不出来」的东西——少一条 focus 规则、
 * 裸写一个 hex、只在 @media 挂了降级却忘掉 .ui-reduced-motion，界面上都「还行」，
 * 直到键盘用户 Tab 过去、或换主题、或用户开了减弱动效才暴露。这里把口径钉成断言，
 * 让漂移以用例失败的形式出现，而不是以截图返工的形式出现。
 *
 * 本文件只读文本，不改任何产品文件；断言的口径来源：
 *   `进度计划/gaea-schedule-zpert8-uiux-port-plan-20261114.md` §2/§6
 *   `进度计划/gaea-schedule-zebra-density-2026-11.md`（密度审计与验收清单）
 *
 * 三条口径说明（避免把「正确的实现」误判成漂移）：
 *  1. **焦点环位置容忍**：焦点环可以挂在元素本身（`X:focus-visible`），也可以挂在
 *     它的子元素上（`X input:focus-visible`）；`<label>`/容器型元素（真实焦点在内层
 *     控件）同样接受 `:focus-within`——这两种写法都能让键盘焦点可见，用
 *     `:focus-visible` 命中不了内层控件。其余情况一律要求真焦点伪类。
 *  2. **焦点环必须真的画出来**：光有 `:focus-visible` 选择器不算数，块内要声明
 *     环/底色/描边/文字色之一（outline / box-shadow / background / border /
 *     color / inset / text-decoration），否则等于「有选择器、无反馈」。
 *  3. **裸 hex 判据与 eslint `local/no-raw-hex` 同源**：`var(--x, #fallback)` 的
 *     回退段不算裸 hex（主题缺变量时的兜底，正是令牌适配的一部分）；`url(#id)`
 *     不是颜色，一并排除。裸写色值一律算漂移。
 */
import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const SCHEDULE_DIR = __dirname
const PAGES_DIR = resolve(__dirname, '../pages')

/** 进度板块样式真源（6 个既有文件；路径相对 frontend/src/schedule） */
const SOURCE_SHEETS = [
  'schedule.css',
  'display-preset.css',
  'gantt/inplace.css',
  'gantt/ruler.css',
  'hit.css',
  'network-display.css',
] as const

/** A 线新增的斑马密度样式表（允许把 focus/降级规则放在这里） */
const DENSITY_SHEET = 'sched-zebra.css'

/**
 * 交互选择器清单：进度板块里「可点/可聚焦/承载键盘操作」的十个钩子。
 * 名词与产品代码里的 class 一字不差——清单不另造第二套命名，产物改了名字
 * 这里就红（下面的「清单与真源同源」用例会先把这一条守住）。
 */
const INTERACTIVE_SELECTORS = [
  '.sched-ribbon-command',
  '.sched-menu-item',
  '.sched-command-palette-item',
  '.sched-doc-tab',
  '.sched-colmenu-item',
  '.sched-dock-item',
  '.sched-gantt-row',
  '.sched-statusbar',
  '.sched-hit-menu-layer',
  '.sched-inplace-filter-btn',
] as const

function readSheet(rel: string): string {
  const path = resolve(SCHEDULE_DIR, rel)
  return existsSync(path) ? readFileSync(path, 'utf8') : ''
}

const V3_SOURCE = readSheet(DENSITY_SHEET)
const SHEET_SOURCES = SOURCE_SHEETS.map((rel) => ({ rel, source: readSheet(rel) }))

// ─────────────────────────────────────────────────────────────────────
// 极简 CSS 解析：够用即可（本仓样式表无 CSS 嵌套，只有 @media/@supports）
// ─────────────────────────────────────────────────────────────────────

interface CssRule {
  /** 逗号拆分后仍保持原样的整段选择器（去多余空白） */
  selector: string
  body: string
  /** 外层 at-rule 前奏链（@media (prefers-reduced-motion: reduce) 等） */
  atChain: string[]
  /** 来源文件（相对 schedule/），报错时能直接指到文件 */
  file: string
}

function stripComments(source: string): string {
  return source.replace(/\/\*[\s\S]*?\*\//g, '')
}

function parseRules(rawSource: string, file: string): CssRule[] {
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
        const atChain = frames.filter((f) => f.kind === 'at').map((f) => (f as { prelude: string }).prelude)
        frames.push({ kind: 'rule', selector: prelude.replace(/\s+/g, ' '), start: i + 1, atChain })
      }
    } else if (ch === '}') {
      const frame = frames.pop()
      buf = ''
      if (frame && frame.kind === 'rule') {
        rules.push({ selector: frame.selector, body: source.slice(frame.start, i), atChain: frame.atChain, file })
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
  ...SHEET_SOURCES.flatMap((s) => parseRules(s.source, s.rel)),
  ...parseRules(V3_SOURCE, DENSITY_SHEET),
]

/** 选择器拆成逗号片段（`.a:hover, .a:focus-visible` → 两段） */
function selectorParts(selector: string): string[] {
  return selector
    .split(',')
    .map((p) => p.trim().replace(/\s+/g, ' '))
    .filter(Boolean)
}

/** 块体拆声明（只取 `名: 值`，忽略无冒号片段） */
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

// ─────────────────────────────────────────────────────────────────────
// 判据
// ─────────────────────────────────────────────────────────────────────

/** 真焦点伪类（含内层控件用 focus-visible、外层容器用 focus-within 两种正解） */
const FOCUS_PSEUDO = /:(focus-visible|focus-within)\b/

/** 「焦点可见」的声明族：环 / 底色 / 描边 / 文字色，任一即算画出反馈 */
const FOCUS_RING_PROPS = [
  'outline',
  'outline-offset',
  'box-shadow',
  'background',
  'background-color',
  'border',
  'border-color',
  'border-width',
  'color',
  'text-decoration',
  'inset',
]

/** 该选择器是否在样式真源里存在基础规则（防清单自说自话） */
function hasBaseRule(className: string): boolean {
  return ALL_RULES.some((rule) => selectorParts(rule.selector).some((part) => part.includes(className)))
}

/** 找焦点环规则；命中返回 `文件 · 选择器` 证据，未命中返回空数组 */
function findFocusRing(className: string): string[] {
  const evidence: string[] = []
  for (const rule of ALL_RULES) {
    for (const part of selectorParts(rule.selector)) {
      if (!part.includes(className)) continue
      if (!FOCUS_PSEUDO.test(part)) continue
      const drawsRing = declarations(rule.body).some(([name]) =>
        FOCUS_RING_PROPS.some((prop) => name === prop || name.startsWith(`${prop}-`)),
      )
      if (drawsRing) evidence.push(`${rule.file}: ${part}`)
    }
  }
  return evidence
}

/** `var(--x, <回退>)` 的回退段与 `url(#id)` 都不是「裸用色值」，先掩掉再查 hex */
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
    // 保留 `var(--x,` 到逗号为止，丢掉回退段（逗号之后到右括号）
    out.push(withoutUrl.slice(start, comma < 0 ? j + 1 : comma + 1))
    i = j + 1
  }
  return out.join('')
}

const RAW_HEX = /#[0-9a-fA-F]{3,8}\b/g

function rawHexHits(source: string): string[] {
  const masked = maskNonColorHex(stripComments(source))
  return masked.match(RAW_HEX) ?? []
}

/** 降级作用域：@media (prefers-reduced-motion: reduce) 内，或选择器带 .ui-reduced-motion */
function isReducedScope(rule: CssRule): boolean {
  if (rule.atChain.some((prelude) => /prefers-reduced-motion\s*:\s*reduce/.test(prelude))) return true
  return /\.ui-reduced-motion\b/.test(rule.selector)
}

/** 块内是否把动效归零（animation/transition 一族） */
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

/** 块内是否声明了动效（用于证明「降级块真的覆盖到了有动效的那条规则」） */
function declaresMotion(body: string): boolean {
  return declarations(body).some(([name, value]) => {
    if (name !== 'animation' && name !== 'transition') return false
    return !/^none\b/.test(value.toLowerCase())
  })
}

/** 降级块的目标选择器（去掉 .ui-reduced-motion 前缀，便于回查基础规则） */
function reducedTargets(selector: string): string[] {
  return selectorParts(selector).map((part) => part.replace(/^\.ui-reduced-motion\s+/, '').trim())
}

// ─────────────────────────────────────────────────────────────────────
// B 线现状探测（B 未落地时，语义用例以 skip 呈现，不伪装成通过）
// ─────────────────────────────────────────────────────────────────────

const SCHEDULE_PAGE_PATH = resolve(PAGES_DIR, 'SchedulePage.tsx')
const PAGE_SOURCE = existsSync(SCHEDULE_PAGE_PATH) ? readFileSync(SCHEDULE_PAGE_PATH, 'utf8') : ''

/** 取出 `<div className="sched-statusbar" ...>` 的开标签（含 JSX 花括号内的 `>` 折叠） */
function openingTagOf(source: string, className: string): string {
  const at = source.indexOf(`className="${className}"`)
  if (at < 0) return ''
  const tagStart = source.lastIndexOf('<', at)
  if (tagStart < 0) return ''
  let depth = 0
  for (let i = tagStart; i < source.length; i++) {
    const ch = source[i]
    if (ch === '{') depth++
    else if (ch === '}') depth--
    else if (ch === '>' && depth === 0) return source.slice(tagStart, i + 1)
  }
  return ''
}

const STATUS_BAR_TAG = openingTagOf(PAGE_SOURCE, 'sched-statusbar')
const HAS_ROLE_STATUS = /role=\{?["']status["']\}?/.test(STATUS_BAR_TAG)
const HAS_ARIA_LIVE = /aria-live=/.test(STATUS_BAR_TAG)
/** B 线语义是否齐备（role + aria-live 都要） */
const ARIA_LANDED = HAS_ROLE_STATUS && HAS_ARIA_LIVE

// ─────────────────────────────────────────────────────────────────────
// 守卫
// ─────────────────────────────────────────────────────────────────────

describe('样式真源齐全（清单漂移即红）', () => {
  it(`A 线产物 ${DENSITY_SHEET} 已落地`, () => {
    expect(
      V3_SOURCE.length,
      `A 线样式表缺失：frontend/src/schedule/${DENSITY_SHEET}。` +
        '焦点环/令牌桥接/降级入口允许写在新文件里，但文件本身必须先存在；' +
        '文件未落地时本组用例会连带红（这是缺口的正确呈现，不是用例写错）。',
    ).toBeGreaterThan(0)
  })

  it('6 个既有样式真源都在（少一个说明文件改名或合并，清单要跟着改）', () => {
    const missing = SOURCE_SHEETS.filter((rel) => !existsSync(resolve(SCHEDULE_DIR, rel)))
    expect(missing, `样式真源缺失：${missing.join('、')}`).toEqual([])
  })

  it('交互选择器清单与真源同源：每个钩子都能找到基础规则', () => {
    const orphans = INTERACTIVE_SELECTORS.filter((sel) => !hasBaseRule(sel))
    expect(
      orphans,
      `以下类名在样式真源里查无基础规则：${orphans.join('、')}——` +
        '要么产品代码改名了（清单跟着改），要么清单凭空造了第二套命名（禁止）。',
    ).toEqual([])
  })
})

describe('交互态：十个钩子都有键盘焦点环', () => {
  for (const selector of INTERACTIVE_SELECTORS) {
    it(`${selector} 有 focus-visible 焦点环`, () => {
      const evidence = findFocusRing(selector)
      expect(
        evidence.length,
        `${selector} 在样式真源（含 ${DENSITY_SHEET}）里找不到焦点环规则。\n` +
          '判据：选择器片段需同时含该类名 + `:focus-visible`（标签/容器型可退 `:focus-within`），' +
          '且块内声明 outline / box-shadow / background / border / color 之一。\n' +
          '补法：A 线在 sched-zebra.css 里补一条即可（不要求在 schedule.css 动刀）。',
      ).toBeGreaterThan(0)
    })
  }
})

describe(`${DENSITY_SHEET} 令牌纪律`, () => {
  it('裸 hex 判据自检：裸写算违规，var 回退与 url(#id) 不算', () => {
    // 守卫的守卫：检测器坏了会「永远绿灯」，比没有断言更危险。
    expect(rawHexHits('.a { color: #abc; }')).toEqual(['#abc']) // hex-exempt
    expect(rawHexHits('.a { color: #11223344; }')).toEqual(['#11223344']) // hex-exempt
    expect(rawHexHits('.a { color: var(--md-sys-color-on-surface, #374151); }')).toEqual([])
    expect(rawHexHits('.a { fill: url(#grad); }')).toEqual([])
  })

  it('零裸 hex：色值全部走令牌（var 回退与 url(#id) 除外）', () => {
    expect(V3_SOURCE.length, `${DENSITY_SHEET} 未落地，无法断言零裸 hex`).toBeGreaterThan(0)
    const hits = rawHexHits(V3_SOURCE)
    expect(hits, `${DENSITY_SHEET} 出现裸 hex：${hits.join('、')}——改用 --v3-* / --md-sys-color-* / --sched-* 令牌`).toEqual([])
  })

  it('同时提供 prefers-reduced-motion 与 .ui-reduced-motion 两个降级入口', () => {
    expect(V3_SOURCE.length, `${DENSITY_SHEET} 未落地，无法断言降级入口`).toBeGreaterThan(0)
    expect(V3_SOURCE, '缺系统级入口 @media (prefers-reduced-motion: reduce)').toContain('prefers-reduced-motion')
    expect(V3_SOURCE, '缺应用内入口 .ui-reduced-motion（App.tsx motion=reduced 挂的根类）').toContain('.ui-reduced-motion')
  })
})

describe('动效降级：真覆盖而不是摆设', () => {
  const zeroedRules = ALL_RULES.filter((rule) => isReducedScope(rule) && zeroesMotion(rule.body))

  /** 归零规则的目标里，有多少同时在基础作用域声明了动效（= 真覆盖） */
  function verifiedOverrides(): Array<{ target: string; file: string }> {
    const out: Array<{ target: string; file: string }> = []
    for (const rule of zeroedRules) {
      for (const target of reducedTargets(rule.selector)) {
        const covered = ALL_RULES.some(
          (base) =>
            !isReducedScope(base) &&
            selectorParts(base.selector).includes(target) &&
            declaresMotion(base.body),
        )
        if (covered) out.push({ target, file: rule.file })
      }
    }
    return out
  }

  it('至少一处 transition/animation 在降级块内被归零，且基础块确有该动效', () => {
    const verified = verifiedOverrides()
    expect(
      verified.length,
      '进度板块样式表里找不到「基础块声明动效 → 降级块归零」的成对证据；' +
        '只有 @media 块却不指向任何动效规则，等于降级没生效。',
    ).toBeGreaterThan(0)
  })

  it('两个降级入口都挂了归零规则（系统偏好与应用内开关不能只认一个）', () => {
    const mediaSide = zeroedRules.filter((rule) =>
      rule.atChain.some((prelude) => /prefers-reduced-motion\s*:\s*reduce/.test(prelude)),
    )
    const classSide = zeroedRules.filter((rule) => /\.ui-reduced-motion\b/.test(rule.selector))
    expect(mediaSide.length, '@media (prefers-reduced-motion: reduce) 内没有任何归零规则').toBeGreaterThan(0)
    expect(classSide.length, '.ui-reduced-motion 作用域内没有任何归零规则（应用内减弱动效失效）').toBeGreaterThan(0)
  })
})

describe('B 线语义：状态栏是活区', () => {
  it('状态栏节点存在且类名未漂移（aria 断言的前提）', () => {
    expect(PAGE_SOURCE.length, 'SchedulePage.tsx 未找到，路径漂移').toBeGreaterThan(0)
    expect(STATUS_BAR_TAG, '找不到 className="sched-statusbar" 的开标签').toContain('sched-statusbar')
  })

  it.skipIf(!ARIA_LANDED)('状态栏 role="status" + aria-live 齐备', () => {
    expect(HAS_ROLE_STATUS, '状态栏缺 role="status"').toBe(true)
    expect(HAS_ARIA_LIVE, '状态栏缺 aria-live（保存中/已保存/保存失败这些状态变化读屏收不到）').toBe(true)
  })

  it('半落地拦截：只要出现了 role 或 aria-live 之一，就必须两者齐备', () => {
    // B 线未落地（两者皆无）时本用例只做记录，不取代语义断言；
    // 一旦 A/B 合流只落了一半，这条会立刻红——避免 skip 掩盖半成品。
    if (HAS_ROLE_STATUS || HAS_ARIA_LIVE) {
      expect(
        { role: HAS_ROLE_STATUS, ariaLive: HAS_ARIA_LIVE },
        '状态栏 aria 语义半落地：role="status" 与 aria-live 必须同时具备',
      ).toEqual({ role: true, ariaLive: true })
    } else {
      expect(STATUS_BAR_TAG).toContain('sched-statusbar')
    }
  })
})
