// sin/SinSidePanel.test.tsx — 右栏创作面板：页签切换/空态/插画collect/预览/持久化。
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { Modal } from 'antd'

const apiMock = vi.hoisted(() => ({
  readFileAsDataURL: vi.fn(),
}))

vi.mock('../../api/image', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../api/image')>()),
  readFileAsDataURL: apiMock.readFileAsDataURL,
}))

import { SinSidePanel } from './SinSidePanel'
import { collectIllustrations } from './storyText'
import { clampSinPanelWidth } from './sinPanelState'
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
      onGenerateSheet={vi.fn().mockResolvedValue(undefined)}
      notesDoc={{ notes: ['女主：林晚，地方台记者'], outline: '第一章：雨夜站台相遇' }}
      notesError=""
      notesLoading={false}
      messages={MESSAGES}
      sending={false}
      onRegenerate={vi.fn().mockResolvedValue(undefined)}
      onSaveNotes={vi.fn().mockResolvedValue({ ok: true, conflict: false, message: '' })}
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
    expect(tabs.map((t) => t.textContent)).toEqual(['角色2', '大纲1', '设定1', '插图2', '书源']) // sin 书源线 t3 +书源页签（无计数徽标）
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

  it('宽度拖拽：左缘手柄向左拖变宽、实时跟手、松手持久化', () => {
    const { container } = renderPanel()
    const aside = container.querySelector('aside.sin-side') as HTMLElement
    expect(aside.style.width).toBe('268px')
    const handle = screen.getByRole('separator', { name: '调整面板宽度' })
    fireEvent.pointerDown(handle, { clientX: 1000 })
    // jsdom innerWidth=1024 → 视口上限 504；向左拖 240px：268+240=508 → 钳到 504
    fireEvent.pointerMove(window, { clientX: 760 })
    expect(aside.style.width).toBe('504px')
    fireEvent.pointerUp(window)
    expect(localStorage.getItem('gaea.sin.panelWidth')).toBe('504')
  })

  it('宽度拖拽：向右拖收窄到下限 240', () => {
    const { container } = renderPanel()
    const aside = container.querySelector('aside.sin-side') as HTMLElement
    fireEvent.pointerDown(screen.getByRole('separator', { name: '调整面板宽度' }), { clientX: 500 })
    fireEvent.pointerMove(window, { clientX: 700 })
    expect(aside.style.width).toBe('240px')
    fireEvent.pointerUp(window)
  })

  it('双击手柄复位默认宽度并持久化', () => {
    const { container } = renderPanel()
    const aside = container.querySelector('aside.sin-side') as HTMLElement
    fireEvent.pointerDown(screen.getByRole('separator', { name: '调整面板宽度' }), { clientX: 1000 })
    fireEvent.pointerMove(window, { clientX: 760 })
    fireEvent.pointerUp(window)
    fireEvent.doubleClick(screen.getByRole('separator', { name: '调整面板宽度' }))
    expect(aside.style.width).toBe('268px')
    expect(localStorage.getItem('gaea.sin.panelWidth')).toBe('268')
  })

  it('宽度记忆：重挂载恢复上次宽度', () => {
    localStorage.setItem('gaea.sin.panelWidth', '380')
    const { container } = renderPanel()
    expect((container.querySelector('aside.sin-side') as HTMLElement).style.width).toBe('380px')
  })
})

describe('SinSidePanel 画廊重新生成（v4.265）', () => {
  const openGallery = async () => {
    fireEvent.click(screen.getByRole('tab', { name: /插图/ }))
    await waitFor(() => expect(screen.getAllByAltText('雨夜站台，湿风衣反光').length).toBeGreaterThan(0))
  }

  it('悬浮钮点击回调带上 messageId/cue/prompt（SinIllustrate 回写定位）', async () => {
    const onRegenerate = vi.fn().mockResolvedValue(undefined)
    renderPanel({ onRegenerate })
    await openGallery()
    fireEvent.click(screen.getAllByTitle('按原画面描述重新生成这张插图')[0])
    expect(onRegenerate).toHaveBeenCalledTimes(1)
    expect(onRegenerate.mock.calls[0][0]).toMatchObject({
      messageId: 3, cue: '0', prompt: '雨夜站台，湿风衣反光', path: '/tmp/a.png',
    })
  })

  it('生成中进 busy：显示「重新生成中…」并挡住重复点击，结束恢复', async () => {
    let resolveRegen: () => void = () => {}
    const onRegenerate = vi.fn().mockImplementation(
      () => new Promise<void>((r) => { resolveRegen = r }),
    )
    renderPanel({ onRegenerate })
    await openGallery()
    fireEvent.click(screen.getAllByTitle('按原画面描述重新生成这张插图')[0])
    expect(screen.getByText('重新生成中…')).toBeTruthy()
    // busy 的那张按钮被状态条顶掉，另一张仍在
    expect(screen.getAllByTitle('按原画面描述重新生成这张插图').length).toBe(1)
    resolveRegen()
    await waitFor(() => expect(screen.queryByText('重新生成中…')).toBeNull())
    expect(screen.getAllByTitle('按原画面描述重新生成这张插图').length).toBe(2)
  })

  it('故事回合进行中（sending）：重新生成钮禁用并给原因', async () => {
    renderPanel({ sending: true })
    await openGallery()
    const btn = screen.getAllByTitle('故事生成中，稍后再试')[0] as HTMLButtonElement
    expect(btn.disabled).toBe(true)
  })

  it('正文无标记的插图（prompt 空）：不出重新生成钮', async () => {
    const ms: SinMessageView[] = [
      { key: 'db_9', messageId: 9, role: 'assistant', content: '无标记正文', illustrations: { '0': '/tmp/x.png' } },
    ]
    renderPanel({ messages: ms })
    fireEvent.click(screen.getByRole('tab', { name: /插图/ }))
    await waitFor(() => expect(screen.getByAltText('故事插图')).toBeTruthy())
    expect(screen.queryByText('重新生成')).toBeNull()
    expect(screen.queryByTitle('按原画面描述重新生成这张插图')).toBeNull()
  })

  it('大图预览 footer 同款重新生成；成功重载后预览跟到新图', async () => {    let resolveRegen: () => void = () => {}
    const onRegenerate = vi.fn().mockImplementation(
      () => new Promise<void>((r) => { resolveRegen = r }),
    )
    const { rerender } = renderPanel({ onRegenerate })
    await openGallery()
    fireEvent.click(screen.getByAltText('雨夜站台，湿风衣反光'))
    const footerBtn = await waitFor(() => {
      const b = document.querySelector('.ant-modal-footer button') as HTMLButtonElement
      expect(b).toBeTruthy()
      return b
    })
    fireEvent.click(footerBtn)
    expect(onRegenerate).toHaveBeenCalledTimes(1)
    // 消息重载带来新路径（key 相同 path 不同）→ 预览对象跟新
    const msNext: SinMessageView[] = [
      ...MESSAGES.slice(0, 2),
      { ...MESSAGES[2], illustrations: { '0': '/tmp/a2.png', '1': '/tmp/b.png' } },
    ]
    resolveRegen()
    rerender(
      <SinSidePanel
        cast={CAST} castSaving={false} onOpenPicker={vi.fn()} onRemoveCast={vi.fn()}
        notesDoc={{ notes: ['女主：林晚，地方台记者'], outline: '第一章：雨夜站台相遇' }}
        notesError="" notesLoading={false} messages={msNext}
        sending={false} onRegenerate={onRegenerate}
        onSaveNotes={vi.fn().mockResolvedValue({ ok: true, conflict: false, message: '' })}
      />,
    )
    await waitFor(() => expect(apiMock.readFileAsDataURL).toHaveBeenCalledWith('/tmp/a2.png'))
  })
})

describe('SinSidePanel 底稿编辑（v4.266）', () => {
  afterEach(() => {
    Modal.destroyAll()
    // antd confirm 在测试环境会重复渲染+残留根节点（记忆坑：confirm DOM 残留误报）
    document.body.querySelectorAll('.ant-modal-root').forEach((n) => n.remove())
  })

  it('大纲编辑：进入→修改→保存回调带基线快照与新值，成功后退出编辑态', async () => {
    const onSaveNotes = vi.fn().mockResolvedValue({ ok: true, conflict: false, message: '' })
    renderPanel({ onSaveNotes })
    fireEvent.click(screen.getByRole('tab', { name: /大纲/ }))
    fireEvent.click(screen.getByTitle('编辑大纲'))
    const ta = screen.getByPlaceholderText('章节走向、时间线、伏笔…') as HTMLTextAreaElement
    expect(ta.value).toBe('第一章：雨夜站台相遇')
    fireEvent.change(ta, { target: { value: '第一章：新的走向' } })
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }))
    await waitFor(() => expect(onSaveNotes).toHaveBeenCalledTimes(1))
    expect(onSaveNotes.mock.calls[0][0]).toEqual({ notes: ['女主：林晚，地方台记者'], outline: '第一章：雨夜站台相遇' })
    expect(onSaveNotes.mock.calls[0][1]).toBe('第一章：新的走向')
    expect(onSaveNotes.mock.calls[0][3]).toBe(false)
    await waitFor(() => expect(screen.queryByPlaceholderText('章节走向、时间线、伏笔…')).toBeNull())
  })

  it('空大纲入口叫「写大纲」；取消不调用保存', () => {
    const onSaveNotes = vi.fn().mockResolvedValue({ ok: true, conflict: false, message: '' })
    renderPanel({ onSaveNotes, notesDoc: { notes: [], outline: '' } })
    fireEvent.click(screen.getByRole('tab', { name: /大纲/ }))
    expect(screen.getByText('写大纲')).toBeTruthy()
    fireEvent.click(screen.getByTitle('编辑大纲'))
    expect(screen.getByPlaceholderText('章节走向、时间线、伏笔…')).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: /取\s*消/ }))
    expect(onSaveNotes).not.toHaveBeenCalled()
    expect(screen.getByText('还没有大纲')).toBeTruthy()
  })

  it('设定编辑：加一条/删一条后整包保存', async () => {
    const onSaveNotes = vi.fn().mockResolvedValue({ ok: true, conflict: false, message: '' })
    renderPanel({ onSaveNotes })
    fireEvent.click(screen.getByRole('tab', { name: /设定/ }))
    fireEvent.click(screen.getByTitle('编辑设定集'))
    fireEvent.click(screen.getByTitle('添加一条'))
    const tas = screen.getAllByPlaceholderText(/设定内容/)
    fireEvent.change(tas[tas.length - 1], { target: { value: '顾城是三年前的线人' } })
    fireEvent.click(screen.getAllByTitle('删除这条便签')[0])
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }))
    await waitFor(() => expect(onSaveNotes).toHaveBeenCalledTimes(1))
    expect(onSaveNotes.mock.calls[0][2]).toEqual(['顾城是三年前的线人'])
  })

  it('sending 中：编辑入口禁用并给原因；草稿中保存按钮也禁用', async () => {
    const onSaveNotes = vi.fn().mockResolvedValue({ ok: true, conflict: false, message: '' })
    const { rerender } = renderPanel({ onSaveNotes, sending: true })
    fireEvent.click(screen.getByRole('tab', { name: /大纲/ }))
    const entry = screen.getByTitle('回合进行中，暂不可编辑') as HTMLButtonElement
    expect(entry.disabled).toBe(true)
    // 放开 sending 进入编辑，再恢复 sending → 保存禁用
    rerender(
      <SinSidePanel
        cast={CAST} castSaving={false} onOpenPicker={vi.fn()} onRemoveCast={vi.fn()}
        notesDoc={{ notes: ['女主：林晚，地方台记者'], outline: '第一章：雨夜站台相遇' }}
        notesError="" notesLoading={false} messages={MESSAGES}
        sending={false} onRegenerate={vi.fn()}
        onSaveNotes={onSaveNotes}
      />,
    )
    fireEvent.click(screen.getByTitle('编辑大纲'))
    rerender(
      <SinSidePanel
        cast={CAST} castSaving={false} onOpenPicker={vi.fn()} onRemoveCast={vi.fn()}
        notesDoc={{ notes: ['女主：林晚，地方台记者'], outline: '第一章：雨夜站台相遇' }}
        notesError="" notesLoading={false} messages={MESSAGES}
        sending onRegenerate={vi.fn()} onSaveNotes={onSaveNotes}
      />,
    )
    const save = screen.getByTitle('回合进行中，暂不能保存') as HTMLButtonElement
    expect(save.disabled).toBe(true)
    expect(onSaveNotes).not.toHaveBeenCalled()
  })

  it('冲突：保存返回 conflict → 弹「覆盖确认」→ 确认后带 force 重试', async () => {
    let calls = 0
    const onSaveNotes = vi.fn().mockImplementation(() => {
      calls += 1
      const first = calls === 1
      return Promise.resolve(first
        ? { ok: false, conflict: true, message: '底稿冲突：便签/大纲已被其他端更新，请确认后再保存' }
        : { ok: true, conflict: false, message: '' })
    })
    renderPanel({ onSaveNotes })
    fireEvent.click(screen.getByRole('tab', { name: /设定/ }))
    fireEvent.click(screen.getByTitle('编辑设定集'))
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }))
    expect((await screen.findAllByText('底稿已被更新')).length).toBeGreaterThan(0)
    const okBtn = Array.from(document.querySelectorAll('.ant-modal-confirm .ant-btn-primary'))
      .find((b) => (b.textContent ?? '').replace(/\s/g, '') === '覆盖') as HTMLButtonElement
    expect(okBtn).toBeTruthy()
    fireEvent.click(okBtn)
    await waitFor(() => expect(onSaveNotes).toHaveBeenCalledTimes(2))
    expect(onSaveNotes.mock.calls[1][3]).toBe(true)
    await waitFor(() => expect(screen.queryByText('保存')).toBeNull())
  })

  it('普通保存失败：错误就地显示，不弹确认框', async () => {
    const onSaveNotes = vi.fn().mockResolvedValue({ ok: false, conflict: false, message: '便签文件不可写' })
    renderPanel({ onSaveNotes })
    fireEvent.click(screen.getByRole('tab', { name: /大纲/ }))
    fireEvent.click(screen.getByTitle('编辑大纲'))
    fireEvent.click(screen.getByRole('button', { name: /保\s*存/ }))
    await waitFor(() => expect(screen.getByText('便签文件不可写')).toBeTruthy())
    expect(screen.queryAllByText('底稿已被更新').length).toBe(0)
  })
})

describe('clampSinPanelWidth', () => {
  it('钳制到 [240, 视口收敛上限]，非法值回默认', () => {
    // jsdom innerWidth=1024 → 上限 min(640, 1024-520)=504
    expect(clampSinPanelWidth(9999)).toBe(504)
    expect(clampSinPanelWidth(100)).toBe(240)
    expect(clampSinPanelWidth(Number.NaN)).toBe(268)
    expect(clampSinPanelWidth(Number.POSITIVE_INFINITY)).toBe(268)
  })
})

describe('collectIllustrations', () => {
  it('按消息序收集全部插图，prompt 从正文标记反解', () => {
    const items = collectIllustrations(MESSAGES)
    expect(items.map((i) => i.path)).toEqual(['/tmp/a.png', '/tmp/b.png'])
    expect(items[0].prompt).toBe('雨夜站台，湿风衣反光')
    expect(items[1].prompt).toBe('车厢内近景，暖色顶灯')
    expect(items[0].key).toBe('db_3:0')
    // v4.265：重新生成的回写定位（SinIllustrate 按 messageId+cue 覆盖写）
    expect(items[0].messageId).toBe(3)
    expect(items[0].cue).toBe('0')
    expect(items[1].cue).toBe('1')
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
