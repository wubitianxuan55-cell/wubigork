import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { CharacterCard } from './CharacterCard'
import type { LibraryCharacter } from '../../api/characterlib'

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
})
