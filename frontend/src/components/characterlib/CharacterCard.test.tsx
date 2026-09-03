import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import * as antd from 'antd'
import { CharacterCard } from './CharacterCard'
import type { LibraryCharacter } from '../../api/characterlib'
import { FRONTEND_EVENTS } from '../../events'
import { WX_FOCUS_KEY } from '../../pages/wxFocus'

vi.mock('../../../src/wailsjsCompat', () => ({
  WhisperAssistantSave: vi.fn(),
}))

import { WhisperAssistantSave } from '../../../src/wailsjsCompat'

const mockedSave = vi.mocked(WhisperAssistantSave)

function makeCharacter(overrides: Partial<LibraryCharacter> = {}): LibraryCharacter {
  return {
    id: 'c1',
    name: '苏念',
    kind: 'custom',
    gender: 'female',
    age: '23',
    tags: ['剑修', '女主'],
    portraitUrl: '',
    roleType: 'protagonist',
    arc: '从逃避到直面宿命。',
    status: 'Alive',
    chatEnabled: true,
    dims: { T: 80, I: 60, S: 50, O: 70, R: 40 },
    createdAt: '',
    updatedAt: '',
    hidden: false,
    ...overrides,
  } as unknown as LibraryCharacter
}

function noop() {}

const baseProps = {
  onEdit: noop,
  onSetPersona: noop,
  onMemory: noop,
  onAssociate: noop,
  onDissociate: noop,
  onDelete: noop,
}

describe('CharacterCard', () => {
  it('渲染档案编号与类型标签', () => {
    render(<CharacterCard character={makeCharacter()} index={1} {...baseProps} />)
    expect(screen.getByText('NO.002')).toBeTruthy()
    expect(screen.getByText('自定义')).toBeTruthy()
  })

  it('渲染元数据行：定位 · 性别 · 年龄 · 状态', () => {
    render(<CharacterCard character={makeCharacter()} index={0} {...baseProps} />)
    expect(screen.getByText('主角 · 女性 · 23 · 存活')).toBeTruthy()
  })

  it('有弧线时渲染引言，无弧线时不渲染', () => {
    const { rerender } = render(<CharacterCard character={makeCharacter()} index={0} {...baseProps} />)
    expect(screen.getByText('从逃避到直面宿命。')).toBeTruthy()
    rerender(<CharacterCard character={makeCharacter({ arc: undefined })} index={0} {...baseProps} />)
    expect(screen.queryByText('从逃避到直面宿命。')).toBeNull()
  })

  it('渲染 # 前缀标签，最多 3 个', () => {
    render(<CharacterCard character={makeCharacter({ tags: ['剑修', '女主', '冷面', '第四'] })} index={0} {...baseProps} />)
    expect(screen.getByText('#剑修')).toBeTruthy()
    expect(screen.getByText('#女主')).toBeTruthy()
    expect(screen.queryByText('#第四')).toBeNull()
  })

  it('有立绘时渲染图片，无立绘时渲染首字占位', () => {
    const { container } = render(<CharacterCard character={makeCharacter({ portraitUrl: 'https://x/1.png' })} index={0} {...baseProps} />)
    expect(container.querySelector('img')?.getAttribute('src')).toBe('https://x/1.png')
    const { container: c2 } = render(<CharacterCard character={makeCharacter({ portraitUrl: '' })} index={0} {...baseProps} />)
    expect(c2.querySelector('.ccard-placeholder')?.textContent).toBe('苏')
  })

  it('仅可聊天角色渲染五维雷达与可聊天标记', () => {
    const { container } = render(<CharacterCard character={makeCharacter()} index={0} {...baseProps} />)
    expect(container.querySelector('.ccard-foot > svg')).toBeTruthy()
    expect(container.querySelector('.ccard-chat')?.textContent).toBe('可聊天')
    const { container: c2 } = render(<CharacterCard character={makeCharacter({ chatEnabled: false })} index={0} {...baseProps} />)
    expect(c2.querySelector('.ccard-foot > svg')).toBeNull()
    expect(c2.querySelector('.ccard-chat')).toBeNull()
  })

  it('点击操作按钮回调对应 handler', () => {
    const c = makeCharacter()
    const onEdit = vi.fn()
    const onSetPersona = vi.fn()
    render(<CharacterCard character={c} index={0} onEdit={onEdit} onSetPersona={onSetPersona} onMemory={noop} onAssociate={noop} onDissociate={noop} onDelete={noop} />)
    fireEvent.click(screen.getByTitle('编辑'))
    expect(onEdit).toHaveBeenCalledWith(c)
    fireEvent.click(screen.getByTitle('设为当前聊天人格'))
    expect(onSetPersona).toHaveBeenCalledWith(c)
  })

  it('isCurrentPersona 时显示当前标记', () => {
    render(<CharacterCard character={makeCharacter()} index={0} isCurrentPersona {...baseProps} />)
    expect(screen.getByText('当前人格')).toBeTruthy()
  })

  it('inProject 时显示已加入标记与按钮', () => {
    render(<CharacterCard character={makeCharacter()} index={0} inProject hasProject {...baseProps} />)
    expect(screen.getAllByText('已加入').length).toBeGreaterThan(0)
  })

  it('点击卡片主体调用 onClick', () => {
    const c = makeCharacter()
    const onClick = vi.fn()
    const { container } = render(<CharacterCard character={c} index={0} onClick={onClick} {...baseProps} />)
    fireEvent.click(container.querySelector('.ccard') as HTMLElement)
    expect(onClick).toHaveBeenCalledWith(c)
  })

  it('点击编辑按钮不触发 onClick', () => {
    const onClick = vi.fn()
    const onEdit = vi.fn()
    render(<CharacterCard character={makeCharacter()} index={0} onClick={onClick} onEdit={onEdit}
      onSetPersona={noop} onMemory={noop} onAssociate={noop} onDissociate={noop} onDelete={noop} />)
    fireEvent.click(screen.getByTitle('编辑'))
    expect(onEdit).toHaveBeenCalledTimes(1)
    expect(onClick).not.toHaveBeenCalled()
  })

  it('键盘 Enter 触发 onClick', () => {
    const c = makeCharacter()
    const onClick = vi.fn()
    const { container } = render(<CharacterCard character={c} index={0} onClick={onClick} {...baseProps} />)
    fireEvent.keyDown(container.querySelector('.ccard') as HTMLElement, { key: 'Enter' })
    expect(onClick).toHaveBeenCalledWith(c)
  })

  it('custom 卡显示「创建青鸟助手」，Popconfirm 确认后以角色为人格调 WhisperAssistantSave', async () => {
    const onAssistantCreated = vi.fn()
    render(<CharacterCard character={makeCharacter()} index={0} {...baseProps} onAssistantCreated={onAssistantCreated} />)
    const trigger = screen.getByTitle('创建青鸟助手')
    expect(trigger.getAttribute('aria-label')).toBe('创建青鸟助手 苏念')
    // Popconfirm 二次确认：点动作 → 点「创建」
    fireEvent.click(trigger)
    fireEvent.click(await screen.findByRole('button', { name: /^创\s*建$/ }))
    await waitFor(() => {
      expect(mockedSave).toHaveBeenCalledWith(expect.objectContaining({
        id: expect.stringMatching(/^wx_/),
        name: '苏念',
        personalityId: 'c1',
        enabled: true,
        portraitUrl: '',
      }))
    })
    // 创建成功后触发列表刷新回调
    expect(onAssistantCreated).toHaveBeenCalledTimes(1)
  })

  it('创建成功 → 深链跳青鸟：写焦点（sessionStorage）+ 派发 NAVIGATE(page=weixin)，提示前往绑定', async () => {
    sessionStorage.removeItem(WX_FOCUS_KEY)
    const navListener = vi.fn()
    window.addEventListener(FRONTEND_EVENTS.NAVIGATE, navListener)
    // 提示文案用 spy 断言：toast 走 antd 全局 portal，跨测试共享 DOM（RTL
    // cleanup 不清），顺序执行时直接查文本会受上一条残留 toast 干扰
    const messageSpy = vi.spyOn(antd.message, 'success')
    try {
      render(<CharacterCard character={makeCharacter()} index={0} {...baseProps} />)
      fireEvent.click(screen.getByTitle('创建青鸟助手'))
      fireEvent.click(await screen.findByRole('button', { name: /^创\s*建$/ }))
      await waitFor(() => {
        // 焦点 = 本次创建的助手 id（先落焦点再导航，青鸟页读后即清）
        const saved = mockedSave.mock.calls[mockedSave.mock.calls.length - 1][0] as { id: string }
        expect(saved.id).toMatch(/^wx_/)
        expect(sessionStorage.getItem(WX_FOCUS_KEY)).toBe(saved.id)
        expect(navListener).toHaveBeenCalledWith(expect.objectContaining({ detail: { page: 'weixin', crossSpace: true } }))
      })
      // 提示文案指向青鸟绑定（不再让用户自己找路）
      expect(messageSpy).toHaveBeenCalledWith('已创建，正在前往青鸟绑定…')
    } finally {
      messageSpy.mockRestore()
      window.removeEventListener(FRONTEND_EVENTS.NAVIGATE, navListener)
      sessionStorage.removeItem(WX_FOCUS_KEY)
    }
  })

  it('assistant / builtin 卡不渲染「创建青鸟助手」动作', () => {
    const { container: a } = render(<CharacterCard character={makeCharacter({ kind: 'assistant' })} index={0} {...baseProps} />)
    expect(a.querySelector('[title="创建青鸟助手"]')).toBeNull()
    const { container: b, unmount } = render(<CharacterCard character={makeCharacter({ kind: 'builtin' })} index={0} {...baseProps} />)
    expect(b.querySelector('[title="创建青鸟助手"]')).toBeNull()
    unmount()
    // custom 卡对照：动作存在
    const { container: c } = render(<CharacterCard character={makeCharacter({ kind: 'custom' })} index={0} {...baseProps} />)
    expect(c.querySelector('[title="创建青鸟助手"]')).not.toBeNull()
  })
})
