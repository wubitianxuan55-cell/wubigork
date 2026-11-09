import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const read = (...parts: string[]) => readFileSync(resolve(__dirname, ...parts), 'utf8')

function firstRule(css: string, selector: string): string {
  const start = css.indexOf(`${selector} {`)
  expect(start).toBeGreaterThanOrEqual(0)
  return css.slice(start, css.indexOf('}', start))
}

function lastRule(css: string, selector: string): string {
  const start = css.lastIndexOf(`${selector} {`)
  expect(start).toBeGreaterThanOrEqual(0)
  return css.slice(start, css.indexOf('}', start))
}

function ruleContaining(css: string, selector: string, declaration: string): string {
  let from = 0
  while (from < css.length) {
    const start = css.indexOf(`${selector} {`, from)
    if (start < 0) break
    const rule = css.slice(start, css.indexOf('}', start))
    if (rule.includes(declaration)) return rule
    from = start + selector.length
  }
  throw new Error(`missing ${selector} containing ${declaration}`)
}

describe('chat and voice scroll containment', () => {
  const css = read('chat-signal-redesign.css')
  const scrollCss = css.slice(css.indexOf('/* ═══ Scroll ownership'))

  it('keeps the chat list bounded and always scrollable', () => {
    const rule = firstRule(scrollCss, '.chat-signal-redesign .chat-message-scroll')
    expect(rule).toContain('overflow-y: auto !important')
    expect(rule).toContain('flex: 1 1 0%')
    expect(rule).toContain('height: auto')
    expect(rule).toContain('max-height: 100%')
    expect(rule).toContain('min-height: 0')
    expect(rule).toContain('scrollbar-gutter: stable')
  })

  it('keeps the voice transcript inside the voice room', () => {
    const rule = firstRule(scrollCss, '.chat-signal-redesign .chat-voice-room-main .vc2-thread')
    expect(rule).toContain('overflow-y: auto !important')
    expect(rule).toContain('flex: 1 1 0%')
    expect(rule).toContain('height: auto')
    expect(rule).toContain('max-height: 100%')
    expect(rule).toContain('min-height: 0')
    expect(rule).toContain('overscroll-behavior: contain')
  })

  it('re-enables visible scrollbars below the global narrow-window breakpoint', () => {
    expect(css).toContain('.chat-signal-redesign .chat-message-scroll::-webkit-scrollbar')
    expect(css).toContain('display: block !important')
    expect(css).toContain('scrollbar-width: thin !important')
  })

  it('keeps the narrow voice command dock stacked and usable', () => {
    const rule = lastRule(css, '.chat-signal-redesign .chat-voice-room-main .vc2-command-row')
    expect(rule).toContain('display: grid')
    expect(rule).toContain('grid-template-columns: minmax(0, 1fr)')
  })

  it('keeps the brain full-window and the transcript in a bounded overlay', () => {
    const start = css.indexOf('/* ═══ Voice workspace alignment reset')
    const room = css.slice(start)
    const layout = firstRule(room, '.chat-signal-redesign .chat-voice-room-main .vc2-layout')
    const dialogue = firstRule(room, '.chat-signal-redesign .chat-voice-room-main .vc2-dialogue')
    const field = firstRule(room, '.chat-signal-redesign .chat-voice-room-main .vc2-field')
    const orb = firstRule(room, '.chat-signal-redesign .chat-voice-room-main .vc2-orb')
    const centeredOrb = firstRule(
      room,
      '.chat-signal-redesign .chat-voice-room-main > .voice-console-v2:not(.is-preflight) .vc2-orb',
    )
    const stageCopy = firstRule(
      room,
      '.chat-signal-redesign .chat-voice-room-main > .voice-console-v2:not(.is-preflight) .vc2-stage-copy',
    )
    expect(layout).toContain('display: grid')
    expect(layout).toContain('grid-template-columns: minmax(0, 1fr)')
    expect(layout).toContain('grid-template-rows: minmax(0, 1fr)')
    expect(dialogue).toContain('position: absolute')
    expect(dialogue).toContain('width: var(--vc2-rail)')
    expect(field).toContain('grid-template-areas:')
    expect(orb).toContain('grid-area: orb')
    expect(centeredOrb).toContain('top: 50%')
    expect(centeredOrb).toContain('left: 50%')
    expect(centeredOrb).toContain('width: min(76%, 860px)')
    expect(centeredOrb).toContain('transform: translate(-50%, -50%)')
    expect(stageCopy).toContain('left: 50%')
    expect(stageCopy).toContain('transform: translateX(-50%)')
    expect(stageCopy).toContain('text-align: center')
  })

  it('renders the voice controls as separate floating capsule cards', () => {
    const start = css.indexOf('/* ═══ Voice workspace alignment reset')
    const room = css.slice(start)
    const dock = firstRule(room, '.chat-signal-redesign .chat-voice-room-main .vc2-command-dock')
    const commandRow = firstRule(room, '.chat-signal-redesign .chat-voice-room-main .vc2-command-row')
    const primary = firstRule(room, '.chat-signal-redesign .chat-voice-room-main .vc2-primary-button.ant-btn')
    const diagnostics = firstRule(room, '.chat-signal-redesign .chat-voice-room-main .vc2-diagnostics')
    expect(dock).toContain('background: transparent')
    expect(dock).toContain('left: 50%')
    expect(commandRow).toContain('display: flex')
    expect(commandRow).toContain('justify-content: center')
    expect(primary).toContain('border-radius: 999px')
    expect(diagnostics).toContain('border-radius: 999px')
  })

  it('keeps the transcript translucent in both light and dark themes', () => {
    const dialogueSelector =
      'html[data-hl="light"] .chat-signal-redesign .chat-voice-room-main > .voice-console-v2:not(.is-preflight) .vc2-dialogue'
    const lightDialogue = firstRule(css, dialogueSelector)
    expect(lightDialogue).toContain('background: transparent')
    expect(lightDialogue).toContain('backdrop-filter: none')
    expect(css).toContain('html[data-hl="dark"] .chat-signal-redesign')
  })
})

describe('chat side panel collapse geometry', () => {
  const css = read('chat-signal-redesign.css')

  it('collapsed side panels release their expanded widths in the final style layer', () => {
    expect(ruleContaining(
      css,
      '.chat-signal-redesign .chat-topic-sidebar.v3-panel.is-collapsed',
      'width: 52px',
    )).toContain('width: 52px')
    expect(ruleContaining(
      css,
      '.chat-signal-redesign .chat-inspector.v3-panel.is-collapsed',
      'width: 40px',
    )).toContain('width: 40px')
  })

  it('keeps the compact mobile rails at the shared 40px width', () => {
    const start = css.indexOf('@media (max-width: 700px)')
    const end = css.indexOf('@media (hover: none)', start)
    const mobile = css.slice(start, end)
    expect(ruleContaining(
      mobile,
      '.chat-signal-redesign .chat-topic-sidebar.v3-panel.is-collapsed',
      'width: 40px',
    )).toContain('width: 40px')
    expect(ruleContaining(
      mobile,
      '.chat-signal-redesign .chat-inspector.v3-panel.is-collapsed',
      'width: 40px',
    )).toContain('width: 40px')
  })
})
