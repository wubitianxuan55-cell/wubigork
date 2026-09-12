// sin/SinSidePanel.test.tsx — 右栏创作面板：页签切换/空态/插画collect/预览/持久化。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'

const apiMock = vi.hoisted(() => ({
  readFileAsDataURL: vi.fn(),
}))

vi.mock('../../api/image', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../api/image')>()),
  readFileAsDataURL: apiMock.readFileAsDataURL,
}))

import { SinSidePanel } from './SinSidePanel'
import { collectIllustrations } from './storyText'
import type { SinMessageView } from './types'
import type { SinCastCharacter } from './useSinCast'

const CAST: SinCastCharacter[] = [
  { id: 'c_lin', name: '林晚', roleType: '女主' },
  { id: 'c_gu', name: '顾城', roleType: '男主' },
]

const MESSAGES: SinMessageView[] = [
  { key: 'db_1', messageId: 1, role: 'user', content: '开场', illustrations: {} },
  {
    key: 'db_2', messageId: 2, role: 'assistant', content: '',
    illustrations: {},
  },
  {
    key: 'db_3', messageId: 3, role: 'assistant',
    content: '雨夜站台。\n\n@@插图|雨夜站台，湿风衣反光@@\n\n她上了车。\n\n@@插图|车厢内近景，暖色顶灯@@',
    illustrations: { '0': '/tmp/a.png', '1': '/tmp/b.png' },
  },
]

function renderPanel(over: Partial<Parameters<typeof SinSidePanel>[0]> = {}) {
  return render(
    <SinSidePanel
      cast={CAST}
      castSaving={false}
      onOpenPicker={vi.fn()}
      onRemoveCast={vi.fn()}
      notesDoc={{ notes: ['女主：林晚，地方台记者'], outline: '第一章：雨夜站台相遇' }}
      notesError=""
      notesLoading={false}
      messages={MESSAGES}
      {...over}
    />,
  )
}

beforeEach(() => {
  localStorage.clear()
  apiMock.readFileAsDataURL.mockReset().mockResolvedValue('data:image/png;base64,AAA')
})

describe('SinSidePanel', () => {
  it('默认页签 = 角色：渲染角色卡（计数徽标与选择按钮）', () => {
    renderPanel()
    const tabs = screen.getAllByRole('tab')
    expect(tabs.map((t) => t.textContent)).toEqual(['角色2', '大纲1', '设定1', '插图2'])
    expect(tabs[0].getAttribute('aria-selected')).toBe('true')
    expect(screen.getByText('调整角色')).toBeTruthy()
  })

  it('大纲页签：渲染大纲正文', () => {
    renderPanel()
    fireEvent.click(screen.getByRole('tab', { name: /大纲/ }))
    expect(screen.getByText(/第一章：雨夜站台相遇/)).toBeTruthy()
  })

  it('设定页签：逐条便签带序号', () => {
    renderPanel()
    fireEvent.click(screen.getByRole('tab', { name: /设定/ }))
    expect(screen.getByText('#0')).toBeTruthy()
    expect(screen.getByText('女主：林晚，地方台记者')).toBeTruthy()
  })

  it('插图页签：画廊收全故事的插图，缩略图点击打开大图预览', async () => {
    renderPanel()
    fireEvent.click(screen.getByRole('tab', { name: /插图/ }))
    await waitFor(() => expect(screen.getAllByAltText('雨夜站台，湿风衣反光').length).toBeGreaterThan(0))
    expect(screen.getByText('车厢内近景，暖色顶灯')).toBeTruthy()
    fireEvent.click(screen.getByAltText('雨夜站台，湿风衣反光'))
    await waitFor(() => expect(screen.getByText('正在读取插图…')).toBeTruthy())
    await waitFor(() => expect(document.querySelector('img.sin-gal-preview-img')).toBeTruthy())
  })

  it('空态：大纲/设定/插图各自给轨道环空态与引导话术', async () => {
    renderPanel({ notesDoc: { notes: [], outline: '' }, messages: [] })
    fireEvent.click(screen.getByRole('tab', { name: /大纲/ }))
    expect(screen.getByText('还没有大纲')).toBeTruthy()
    fireEvent.click(screen.getByRole('tab', { name: /设定/ }))
    expect(screen.getByText('还没有设定便签')).toBeTruthy()
    fireEvent.click(screen.getByRole('tab', { name: /插图/ }))
    expect(screen.getByText('还没有插图')).toBeTruthy()
  })

  it('便签读取失败：设定页签如实给错误', () => {
    renderPanel({ notesError: '便签读取失败' })
    fireEvent.click(screen.getByRole('tab', { name: /设定/ }))
    expect(screen.getByText('便签读取失败')).toBeTruthy()
  })

  it('页签记忆：切页签写入 localStorage，重挂载恢复', () => {
    const { unmount } = renderPanel()
    fireEvent.click(screen.getByRole('tab', { name: /大纲/ }))
    expect(localStorage.getItem('gaea.sin.panelTab')).toBe('outline')
    unmount()
    renderPanel()
    expect(screen.getByRole('tab', { name: /大纲/ }).getAttribute('aria-selected')).toBe('true')
  })

  it('玩法说明入口在面板头部（问号气泡触发钮）', () => {
    renderPanel()
    expect(screen.getByLabelText('玩法与协议说明')).toBeTruthy()
  })
})

describe('collectIllustrations', () => {
  it('按消息序收集全部插图，prompt 从正文标记反解', () => {
    const items = collectIllustrations(MESSAGES)
    expect(items.map((i) => i.path)).toEqual(['/tmp/a.png', '/tmp/b.png'])
    expect(items[0].prompt).toBe('雨夜站台，湿风衣反光')
    expect(items[1].prompt).toBe('车厢内近景，暖色顶灯')
    expect(items[0].key).toBe('db_3:0')
  })

  it('正文没有对应标记时 prompt 诚实留空；空路径跳过', () => {
    const ms: SinMessageView[] = [
      { key: 'db_9', messageId: 9, role: 'assistant', content: '无标记正文', illustrations: { '0': '/tmp/x.png', '2': '' } },
    ]
    const items = collectIllustrations(ms)
    expect(items).toHaveLength(1)
    expect(items[0].prompt).toBe('')
  })
})
