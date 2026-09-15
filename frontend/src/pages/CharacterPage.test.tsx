// CharacterPage.test.tsx — 角色页职业区块冒烟（t5 第四刀 §7.6）。
// 手法沿用 ChapterPage.test：vi.mock bridge Proxy + 重型子组件桩 + 真实 store。
// 只锁：Drawer 职业区块渲染（主职业当前值/副职业 Tag）、「设定」逐参调用、
// 副职业达上限 2 时「添加」禁用、移除逐参。全程不抛错。
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'

const bindings = vi.hoisted(() => ({
  GetCharacters: vi.fn(),
  CharacterListByProject: vi.fn(),
  SetCharacterCareer: vi.fn(),
  RemoveCharacterCareer: vi.fn(),
}))

vi.mock('../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../gaea/lib/bridge')>()
  return {
    ...actual,
    app: new Proxy({} as typeof actual.app, {
      get(_t, prop: string) {
        const m = (bindings as unknown as Record<string, unknown>)[prop]
        if (typeof m === 'function') return m
        const fb = (actual.app as unknown as Record<string, unknown>)[prop]
        if (typeof fb === 'function') return fb.bind(actual.app)
        return undefined
      },
    }),
  }
})

// 重型子组件桩：冒烟只关心 Drawer 内职业区块
vi.mock('../components/RelationGraph', () => ({ default: () => <div data-testid="rel-graph-stub" /> }))
vi.mock('../components/novel/character/PortraitLightbox', () => ({ default: () => null }))
vi.mock('../components/characterlib/PortraitImg', () => ({ default: (p: { src?: string }) => (p.src ? <img src={p.src} alt="" /> : null) }))

import CharacterPage from './CharacterPage'
import { useAppStore } from '../stores/appStore'

const character = {
  id: 'mc', name: '林晚', role_type: 'protagonist', status: 'Alive',
  gender: 'female', personality: '冷静', background: '', appearance: '', figure: '',
  motivation: '', arc: '',
  main_career_id: '剑修', main_career_stage: 3,
  sub_careers: [
    { career_id: '炼丹师', career_name: '炼丹师', stage: 2 },
    { career_id: '阵法师', stage: 1 },
  ],
}

beforeEach(() => {
  vi.clearAllMocks()
  bindings.GetCharacters.mockResolvedValue({ characters: [character], organizations: [], relationships: [] })
  bindings.CharacterListByProject.mockResolvedValue([])
  bindings.SetCharacterCareer.mockResolvedValue(undefined)
  bindings.RemoveCharacterCareer.mockResolvedValue(undefined)
  useAppStore.setState({ projectPath: 'C:\\proj\\demo' } as never)
})

afterEach(() => {
  document.body.innerHTML = ''
})

async function openDrawer() {
  render(<CharacterPage />)
  // 角色卡片渲染后点击打开 Drawer
  const card = await screen.findByText('林晚')
  fireEvent.click(card.closest('[class*="char-card"]') || card)
  await screen.findByText('职业体系')
}

describe('CharacterPage 职业区块（t5 §7.6）', () => {
  it('渲染主职业当前值与副职业 Tag（含无名快照退 career_id）', async () => {
    await openDrawer()
    expect(screen.getByText(/主职业（当前：剑修·3阶）/)).toBeTruthy()
    expect(screen.getByText('炼丹师·2阶')).toBeTruthy()
    expect(screen.getByText('阵法师·1阶')).toBeTruthy()
  })

  it('「设定」逐参调用 SetCharacterCareer 并刷新', async () => {
    await openDrawer()
    const inputs = screen.getAllByPlaceholderText('如：剑修（名称即 ID）')
    fireEvent.change(inputs[0], { target: { value: '刀客' } })
    fireEvent.click(screen.getByText(/^设\s*定$/) as HTMLElement)
    await waitFor(() => {
      expect(bindings.SetCharacterCareer).toHaveBeenCalledWith('mc', JSON.stringify({ is_main: true, career_name: '刀客', stage: 3 }))
    })
    // 刷新触发（GetCharacters 至少两次：挂载 + 职业操作后）
    await waitFor(() => expect(bindings.GetCharacters.mock.calls.length).toBeGreaterThanOrEqual(2))
  })

  it('副职业达上限 2 时「添加」禁用；移除逐参调用 RemoveCharacterCareer', async () => {
    await openDrawer()
    const addBtn = (screen.getByText(/^添\s*加$/) as HTMLElement).closest('button') as HTMLButtonElement
    expect(addBtn.disabled).toBe(true)

    // 副职业行的移除按钮：定位在「炼丹师·2阶」Tag 同行（排除工具栏清除钮）
    const tag = screen.getByText('炼丹师·2阶')
    const row = tag.closest('div')!
    const delBtn = Array.from(row.querySelectorAll('button')).find(b => b.querySelector('.anticon-close'))
    expect(delBtn).toBeTruthy()
    fireEvent.click(delBtn!)
    // Popconfirm 二次确认
    fireEvent.click(await screen.findByText(/^移\s*除$/, { selector: '.ant-popconfirm button span' }))
    await waitFor(() => {
      expect(bindings.RemoveCharacterCareer).toHaveBeenCalledWith('mc', JSON.stringify({ is_main: false, career_name: '炼丹师' }))
    })
  })
})
