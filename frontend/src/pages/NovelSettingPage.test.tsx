import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

// 屏蔽 Wails 绑定：jsdom 中没有 window.go
vi.mock('../../src/wailsjsCompat', () => ({
  GetWorldview: vi.fn().mockResolvedValue('# 世界观\n\n架空中世纪'),
  SaveWorldview: vi.fn().mockResolvedValue(undefined),
  ChatWorldview: vi.fn().mockResolvedValue({ reply: 'ok', worldview: '# 新设定' }),
}))

import NovelSettingPage from './NovelSettingPage'
import { useAppStore } from '../stores/appStore'
import { GetWorldview, SaveWorldview } from '../../src/wailsjsCompat'

describe('NovelSettingPage 纯文本设定编辑', () => {
  beforeEach(() => {
    useAppStore.setState({ projectOpen: true, projectPath: 'C:/novel/test' })
    vi.clearAllMocks()
    vi.mocked(GetWorldview).mockResolvedValue('# 世界观\n\n架空中世纪')
  })

  it('加载已有设定并在编辑区显示', async () => {
    render(<NovelSettingPage />)
    const editor = (await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)) as HTMLTextAreaElement
    expect(editor.value).toContain('架空中世纪')
  })

  it('编辑后显示未保存状态，保存调用 SaveWorldview', async () => {
    render(<NovelSettingPage />)
    const editor = (await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)) as HTMLTextAreaElement
    fireEvent.change(editor, { target: { value: '蒸汽纪元' } })
    expect(screen.getByText('有未保存修改')).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: /保存/ }))
    expect(SaveWorldview).toHaveBeenCalledWith('蒸汽纪元')
  })

  it('切换到渲染模式直接渲染设定文本', async () => {
    render(<NovelSettingPage />)
    await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)
    fireEvent.click(screen.getByRole('radio', { name: /渲染/ }))
    expect(await screen.findByText('架空中世纪')).toBeTruthy()
  })
})
