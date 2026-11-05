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

  it('keeps the narrow voice command dock in one stable row', () => {
    const start = css.indexOf('@media (max-width: 760px)')
    const end = css.indexOf('@container (max-width: 520px)', start)
    const mobile = css.slice(start, end)
    const rule = firstRule(mobile, '.chat-signal-redesign .chat-voice-room-main .vc2-command-row')
    expect(rule).toContain('flex-direction: row')
    expect(rule).toContain('flex-wrap: nowrap')
  })

  it('keeps the particle orb centered while the transcript floats on the right', () => {
    const start = css.indexOf('/* ═══ Standalone voice room')
    const end = css.indexOf('/* ═══ Scroll ownership', start)
    const room = css.slice(start, end)
    const layout = firstRule(room, '.chat-signal-redesign .chat-voice-room-main .vc2-layout')
    const dialogue = firstRule(room, '.chat-signal-redesign .chat-voice-room-main .vc2-dialogue')
    expect(layout).toContain('display: block')
    expect(dialogue).toContain('position: absolute')
    expect(dialogue).toContain('right: 0')
    expect(dialogue).toContain('width: clamp(320px, 30%, 400px)')
  })

  it('aligns the primary voice control with the centered sound field', () => {
    const start = css.indexOf('/* ═══ Standalone voice room')
    const end = css.indexOf('/* ═══ Scroll ownership', start)
    const room = css.slice(start, end)
    const desktopRoom = room.slice(0, room.indexOf('@container (max-width: 940px)'))
    const commandRow = firstRule(room, '.chat-signal-redesign .chat-voice-room-main .vc2-command-row')
    const primary = firstRule(room, '.chat-signal-redesign .chat-voice-room-main .vc2-primary-actions')
    const endSession = lastRule(desktopRoom, '.chat-signal-redesign .chat-voice-room-main .vc2-end-session.ant-btn')
    expect(commandRow).toContain('grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr)')
    expect(primary).toContain('grid-column: 2')
    expect(primary).toContain('justify-content: center')
    expect(endSession).toContain('grid-column: 3')
    expect(endSession).toContain('justify-self: end')
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
