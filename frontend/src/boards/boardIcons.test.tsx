import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import {
  GaeaCharacterIcon,
  GaeaChatIcon,
  GaeaCodeIcon,
  GaeaCostIcon,
  GaeaDreamIcon,
  GaeaLiteratureIcon,
  GaeaMemoryIcon,
  GaeaModelIcon,
  GaeaOfficeIcon,
  GaeaPantheonIcon,
  GaeaRuntimeIcon,
  GaeaScheduleIcon,
} from './boardIcons'

const icons = [
  GaeaChatIcon,
  GaeaLiteratureIcon,
  GaeaDreamIcon,
  GaeaOfficeIcon,
  GaeaCostIcon,
  GaeaCodeIcon,
  GaeaPantheonIcon,
  GaeaMemoryIcon,
  GaeaModelIcon,
  GaeaCharacterIcon,
  GaeaScheduleIcon,
  GaeaRuntimeIcon,
]

describe('Gaea board icons', () => {
  it.each(icons)('%# keeps the shared 24px outline geometry', (Icon) => {
    const { container } = render(<Icon />)
    const svg = container.querySelector('svg')
    expect(svg).not.toBeNull()
    expect(svg?.getAttribute('viewBox')).toBe('0 0 24 24')
    expect(svg?.getAttribute('stroke')).toBe('currentColor')
    expect(svg?.getAttribute('stroke-width')).toBe('1.6')
    expect(svg?.getAttribute('stroke-linecap')).toBe('round')
    expect(svg?.getAttribute('stroke-linejoin')).toBe('round')
    expect(svg?.getAttribute('fill')).toBe('none')
    expect(svg?.getAttribute('aria-hidden')).toBe('true')
    expect(svg?.getAttribute('data-gaea-board-icon')).toBe('')
  })

  it.each(icons)('%# does not use emoji or text glyphs as the icon', (Icon) => {
    const { container } = render(<Icon />)
    expect(container.querySelector('text')).toBeNull()
    expect(container.textContent).toBe('')
  })
})
