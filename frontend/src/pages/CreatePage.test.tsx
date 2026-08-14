import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'

// Wails 绑定 mock（CreatePage 及子组件经 wailsjsCompat 调用；
// CancelCreateChapter 为 T6-7.2 新增契约，wails build 再生成 NovelB.d.ts 前由页面局部桥接调用）
const mocks = vi.hoisted(() => ({
  GetWorldview: vi.fn().mockResolvedValue('# 世界观\n\n架空中世纪'),
  GetStats: vi.fn().mockResolvedValue({ totalWords: 1200, chapterCount: 2 }),
  ListSkills: vi.fn().mockResolvedValue([]),
  GetChapterBranch: vi.fn().mockResolvedValue({ content: '旧正文' }),
  QuickBrainstormBranches: vi.fn().mockResolvedValue({ branches: [] }),
  CreateChapter: vi.fn().mockResolvedValue({ streaming: true, chapterNum: 1, nodeId: 'n1', branch: '' }),
  DeleteOutlineNode: vi.fn().mockResolvedValue(undefined),
  SaveChapterBranchContent: vi.fn().mockResolvedValue(undefined),
  SaveCharactersBatch: vi.fn().mockResolvedValue({}),
  CancelCreateChapter: vi.fn().mockResolvedValue(true),
}))
vi.mock('../../src/wailsjsCompat', () => mocks)

import CreatePage from './CreatePage'
import { useOutlineStore } from '../stores/outlineStore'
import { useAppStore } from '../stores/appStore'

type Listener = (data: unknown) => void
const runtimeListeners = new Map<string, Listener>()
const EventsOn = vi.fn((name: string, handler: Listener) => { runtimeListeners.set(name, handler) })
const EventsOff = vi.fn((name: string) => { runtimeListeners.delete(name) })

function emit(name: string, payload: unknown) {
  const handler = runtimeListeners.get(name)
  if (handler) act(() => handler(payload))
}

beforeEach(() => {
  runtimeListeners.clear()
  EventsOn.mockClear()
  EventsOff.mockClear()
  Object.defineProperty(window, 'runtime', { configurable: true, writable: true, value: { EventsOn, EventsOff } })
  useOutlineStore.setState({ outlines: [] })
  useAppStore.setState({ projectOpen: true, projectPath: 'C:/novel/test' })
  vi.mocked(mocks.CancelCreateChapter).mockResolvedValue(true)
})

/** 渲染页面并触发一次直接生成（CreateChapter 返回 chapterNum=1） */
async function startGeneration() {
  render(<CreatePage />)
  const plot = await screen.findByPlaceholderText(/或直接输入剧情要求/)
  fireEvent.change(plot, { target: { value: '主角觉醒' } })
  fireEvent.click(screen.getByRole('button', { name: /按剧情要求直接生成/ }))
  await screen.findByRole('button', { name: /停止生成/ })
}

describe('CreatePage 生成控制（T6-7.2 停止按钮 + cancelled 事件）', () => {
  it('生成中渲染停止按钮；点击调用 NovelB.CancelCreateChapter(chapNum, branch)', async () => {
    await startGeneration()
    const stop = screen.getByRole('button', { name: /停止生成/ })
    fireEvent.click(stop)
    await waitFor(() => {
      // chapNum 取 CreateChapter 返回值（1），主线分支为 ''
      expect(mocks.CancelCreateChapter).toHaveBeenCalledWith(1, '')
    })
  })

  it('cancelled 事件：generating 结束、已累积正文保留在编辑器可继续编辑', async () => {
    await startGeneration()
    emit('create-chapter-stream', { type: 'chunk', content: '第一段正文', total: 5 })
    emit('create-chapter-stream', { type: 'cancelled', chapterNum: 1, branch: '', nodeId: 'n1', total: 5, content: '第一段正文' })
    // generating 结束：停止按钮消失
    await waitFor(() => expect(screen.queryByRole('button', { name: /停止生成/ })).toBeNull())
    // 部分正文保留在编辑器
    const editor = screen.getByPlaceholderText(/AI 将在此流式呈现正文/) as HTMLTextAreaElement
    expect(editor.value).toContain('第一段正文')
    // 终态收尾：退订流式监听
    expect(EventsOff).toHaveBeenCalledWith('create-chapter-stream')
  })

  it.each([
    ['done', { type: 'done', chapterNum: 1, branch: '', total: 5000 }],
    ['error', { type: 'error', error: '生成失败' }],
    ['cancelled', { type: 'cancelled', chapterNum: 1, branch: '', nodeId: 'n1', total: 3, content: '部分' }],
  ])('终态事件 %s 收尾：generating 复位且退订监听（无悬挂）', async (_label, payload) => {
    await startGeneration()
    emit('create-chapter-stream', payload)
    await waitFor(() => expect(screen.queryByRole('button', { name: /停止生成/ })).toBeNull())
    expect(EventsOff).toHaveBeenCalledWith('create-chapter-stream')
  })

  it('生成中卸载组件：流式监听被退订（无悬挂）', async () => {
    const { unmount } = render(<CreatePage />)
    const plot = await screen.findByPlaceholderText(/或直接输入剧情要求/)
    fireEvent.change(plot, { target: { value: '主角觉醒' } })
    fireEvent.click(screen.getByRole('button', { name: /按剧情要求直接生成/ }))
    await screen.findByRole('button', { name: /停止生成/ })
    unmount()
    expect(EventsOff).toHaveBeenCalledWith('create-chapter-stream')
  })

  it('CancelCreateChapter 返回 false（幂等/未开始）：本地兜底收尾，UI 不悬挂', async () => {
    vi.mocked(mocks.CancelCreateChapter).mockResolvedValue(false)
    await startGeneration()
    fireEvent.click(screen.getByRole('button', { name: /停止生成/ }))
    await waitFor(() => expect(screen.queryByRole('button', { name: /停止生成/ })).toBeNull())
    expect(EventsOff).toHaveBeenCalledWith('create-chapter-stream')
  })
})
