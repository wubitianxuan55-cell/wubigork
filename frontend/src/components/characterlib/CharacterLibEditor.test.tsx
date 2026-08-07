import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import CharacterLibEditor from './CharacterLibEditor'
import type { LibraryCharacter } from '../../api/characterlib'

vi.mock('../../api/characterlib', () => ({
  saveCharacter: vi.fn(),
  generateFill: vi.fn(),
  generatePortrait: vi.fn(),
}))

import { saveCharacter, generateFill, generatePortrait } from '../../api/characterlib'

const mockedSave = vi.mocked(saveCharacter)
const mockedFill = vi.mocked(generateFill)
const mockedPortrait = vi.mocked(generatePortrait)

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
    chatEnabled: false,
    dims: { T: 80, I: 60, S: 50, O: 70, R: 40 },
    createdAt: '',
    updatedAt: '',
    hidden: false,
    ...overrides,
  } as unknown as LibraryCharacter
}

function renderEditor(overrides: {
  character?: LibraryCharacter | null
  projects?: string[]
  isCurrentPersona?: boolean
} = {}) {
  const onClose = vi.fn()
  const onSaved = vi.fn()
  const character = 'character' in overrides ? overrides.character : makeCharacter()
  const utils = render(
    <CharacterLibEditor
      open
      character={character}
      projects={overrides.projects ?? []}
      index={0}
      isCurrentPersona={overrides.isCurrentPersona}
      onClose={onClose}
      onSaved={onSaved}
    />,
  )
  return { onClose, onSaved, ...utils }
}

beforeEach(() => {
  mockedSave.mockReset()
  mockedFill.mockReset()
  mockedPortrait.mockReset()
})

describe('CharacterLibEditor（档案详情）', () => {
  it('编辑态渲染档案眉：编号 + 类型标签', () => {
    renderEditor()
    expect(screen.getByText('角色档案 · NO.001')).toBeTruthy()
    expect(screen.getByText('自定义')).toBeTruthy()
  })

  it('新建态渲染“新建档案”', () => {
    renderEditor({ character: null })
    expect(screen.getByText('新建档案')).toBeTruthy()
  })

  it('有立绘时横幅渲染图片，无立绘时渲染首字占位', () => {
    renderEditor({ character: makeCharacter({ portraitUrl: 'https://x/1.png' }) })
    expect(document.body.querySelector('.cd-hero-img')?.getAttribute('src')).toBe('https://x/1.png')
    renderEditor()
    expect(document.body.querySelector('.cd-hero-ph')?.textContent).toBe('苏')
  })

  it('身份栏渲染名称与元数据字段', () => {
    renderEditor()
    expect((screen.getByPlaceholderText('角色名') as HTMLInputElement).value).toBe('苏念')
    expect(screen.getByText('性别')).toBeTruthy()
    expect(screen.getByText('年龄')).toBeTruthy()
    expect(screen.getByText('定位')).toBeTruthy()
    expect(screen.getByText('状态')).toBeTruthy()
  })

  it('卷宗正文渲染四个分区', () => {
    renderEditor()
    expect(screen.getByText('小说设定')).toBeTruthy()
    expect(screen.getByText('对话样本')).toBeTruthy()
    expect(screen.getByText('备注')).toBeTruthy()
    expect(screen.getByText('聊天设定')).toBeTruthy()
  })

  it('切换可聊天开关后保存携带 chatEnabled=true', async () => {
    mockedSave.mockResolvedValue(makeCharacter({ chatEnabled: true }))
    renderEditor()
    fireEvent.click(document.body.querySelector('.cd-chat-toggle .ant-switch') as HTMLElement)
    fireEvent.click(screen.getByText('保存'))
    expect(mockedSave).toHaveBeenCalledTimes(1)
    expect((mockedSave.mock.calls[0][0] as any).chatEnabled).toBe(true)
  })

  it('保存成功后回调 onSaved 与 onClose', async () => {
    const saved = makeCharacter({ chatEnabled: true })
    mockedSave.mockResolvedValue(saved)
    const { onClose, onSaved } = renderEditor()
    fireEvent.click(screen.getByText('保存'))
    expect(mockedSave).toHaveBeenCalledTimes(1)
    await vi.waitFor(() => {
      expect(onSaved).toHaveBeenCalledWith(saved)
      expect(onClose).toHaveBeenCalledTimes(1)
    })
  })

  it('名称为空时不保存并提示', async () => {
    renderEditor()
    fireEvent.change(screen.getByPlaceholderText('角色名'), { target: { value: '' } })
    fireEvent.click(screen.getByText('保存'))
    expect(mockedSave).not.toHaveBeenCalled()
  })

  it('isCurrentPersona 时渲染当前人格徽标', () => {
    renderEditor({ isCurrentPersona: true })
    expect(screen.getByText('当前人格')).toBeTruthy()
  })

  it('渲染项目引用信息', () => {
    renderEditor({ projects: ['星落之城'] })
    expect(screen.getByText(/被 1 个项目引用/)).toBeTruthy()
  })

  it('随机补齐调用 generateFill 并回填空缺字段', async () => {
    mockedFill.mockResolvedValue(makeCharacter({ personality: '清冷剑修，寡言重诺' }) as any)
    renderEditor()
    fireEvent.click(screen.getByText('随机补齐'))
    await vi.waitFor(() => {
      expect(mockedFill).toHaveBeenCalledTimes(1)
      const areas = document.body.querySelectorAll('.cd-area')
      expect((areas[0] as HTMLTextAreaElement).value).toBe('清冷剑修，寡言重诺')
    })
  })

  it('生成剧照调用 generatePortrait 并更新立绘横幅', async () => {
    mockedPortrait.mockResolvedValue('data:image/png;base64,AAAA')
    renderEditor()
    fireEvent.click(screen.getByText('生成剧照'))
    await vi.waitFor(() => {
      expect(mockedPortrait).toHaveBeenCalledTimes(1)
      expect(document.body.querySelector('.cd-hero-img')?.getAttribute('src')).toBe('data:image/png;base64,AAAA')
    })
  })

  it('名称为空时随机补齐与生成剧照不调用后端', async () => {
    renderEditor()
    fireEvent.change(screen.getByPlaceholderText('角色名'), { target: { value: '' } })
    fireEvent.click(screen.getByText('随机补齐'))
    fireEvent.click(screen.getByText('生成剧照'))
    expect(mockedFill).not.toHaveBeenCalled()
    expect(mockedPortrait).not.toHaveBeenCalled()
  })
})
