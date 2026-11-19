import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { WorkspaceSidebarToggleIcon } from './WorkspaceSidebarToggleIcon'

describe('WorkspaceSidebarToggleIcon', () => {
  it('收起态显示向右展开箭头', () => {
    const { container } = render(<WorkspaceSidebarToggleIcon collapsed />)
    const icon = container.querySelector('svg')
    expect(icon?.getAttribute('data-workspace-sidebar-toggle')).toBe('expand')
    expect(icon?.getAttribute('viewBox')).toBe('0 0 24 24')
    expect(icon?.getAttribute('stroke')).toBe('currentColor')
    expect(icon?.getAttribute('aria-hidden')).toBe('true')
    expect(container.querySelector('path[d="M12.5 8.25 16.5 12.5 12.5 16.75"]')).not.toBeNull()
  })

  it('展开态显示向左收起箭头', () => {
    const { container } = render(<WorkspaceSidebarToggleIcon collapsed={false} />)
    const icon = container.querySelector('svg')
    expect(icon?.getAttribute('data-workspace-sidebar-toggle')).toBe('collapse')
    expect(container.querySelector('path[d="M16.5 8.25 12.5 12.5l4 4.25"]')).not.toBeNull()
  })
})
