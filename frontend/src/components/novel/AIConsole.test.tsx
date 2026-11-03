import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { AIConsole } from './AIConsole'
import { useAppStore } from '../../stores/appStore'

describe('AIConsole 小说流程观测器', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useAppStore.setState({ projectPath: 'C:/novel/demo' })
    delete (window as Window & { runtime?: unknown }).runtime
  })

  it('从浮动入口打开，关闭后把焦点还给入口', async () => {
    render(<AIConsole />)

    const trigger = screen.getByRole('button', { name: '打开 AI 控制台' })
    expect(trigger.getAttribute('aria-expanded')).toBe('false')
    fireEvent.click(trigger)

    const panel = screen.getByRole('log', { name: 'AI 控制台' })
    expect(panel.id).toBe('novel-ai-console')
    expect(trigger.getAttribute('aria-controls')).toBe(panel.id)
    expect(screen.getByText('PROCESS TRACE')).toBeTruthy()

    const close = screen.getByRole('button', { name: '关闭 AI 控制台' })
    await waitFor(() => expect(document.activeElement).toBe(close))
    fireEvent.click(close)

    expect(screen.queryByRole('log', { name: 'AI 控制台' })).toBeNull()
    await waitFor(() => {
      expect(document.activeElement).toBe(screen.getByRole('button', { name: '打开 AI 控制台' }))
    })
  })
})
