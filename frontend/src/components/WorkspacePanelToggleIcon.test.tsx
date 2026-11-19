import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { WorkspacePanelToggleIcon } from './WorkspacePanelToggleIcon'

describe('WorkspacePanelToggleIcon', () => {
  it('收起态显示向左展开箭头', () => {
    const { container } = render(<WorkspacePanelToggleIcon open={false} />)
    const icon = container.querySelector('svg')
    expect(icon?.getAttribute('data-workspace-panel-toggle')).toBe('open')
    expect(icon?.getAttribute('viewBox')).toBe('0 0 24 24')
    expect(icon?.getAttribute('stroke')).toBe('currentColor')
    expect(icon?.getAttribute('aria-hidden')).toBe('true')
    expect(container.querySelector('path[d="M11.25 8.25 7.25 12.5l4 4.25"]')).not.toBeNull()
  })

  it('展开态显示向右收起箭头', () => {
    const { container } = render(<WorkspacePanelToggleIcon open />)
    const icon = container.querySelector('svg')
    expect(icon?.getAttribute('data-workspace-panel-toggle')).toBe('close')
    expect(container.querySelector('path[d="M7.25 8.25 11.25 12.5 7.25 16.75"]')).not.toBeNull()
  })
})
