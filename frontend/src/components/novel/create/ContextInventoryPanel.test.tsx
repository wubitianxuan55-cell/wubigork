// ContextInventoryPanel.test.tsx — 上下文清单面板（v4.441）：NovelContextInventory
// dry-run 的用户可见面。行为锁：①打开即拉清单并按行渲染（区段/字数/说明/预览）；
// ②「成人向工艺区段」恒在列（档位生效面如实回显——explicit 带预览，未启用零字数）；
// ③合计行与尾注。mock 口径照 NewCharactersModal.test.tsx（gaea bridge 桩）。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import ContextInventoryPanel from './ContextInventoryPanel'
import { app } from '../../../gaea/lib/bridge'

vi.mock('../../../gaea/lib/bridge', () => ({
  app: { NovelContextInventory: vi.fn(async () => []) },
}))

const mockedInventory = vi.mocked(app.NovelContextInventory)

const FIXTURE = [
  { name: 'setting（小说设定）', runes: 0, note: '由编辑区传入', preview: '' },
  { name: '章节计划', runes: 120, note: '刀1 意图注入', preview: '本章：重逢→摊牌' },
  { name: '成人向工艺区段', runes: 340, note: '档位=直白；正文向纪律随整章/重写/收敛/场景注入', preview: '── 成人向写作纪律（本书） ── 本书为成人向作品（尺度：直白' },
  { name: '合计（不含 setting 槽）', runes: 460, note: '各区段另受分项与总预算约束', preview: '' },
]

describe('ContextInventoryPanel 上下文清单（v4.441）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('打开即拉清单：行渲染 + 成人向档位行可见 + 合计尾注', async () => {
    mockedInventory.mockResolvedValue(FIXTURE as never)
    render(<ContextInventoryPanel open chapterNum={3} onClose={vi.fn()} />)
    await waitFor(() => expect(mockedInventory).toHaveBeenCalledWith(3))
    expect(await screen.findByText('章节计划')).toBeTruthy()
    expect(screen.getByText('成人向工艺区段')).toBeTruthy()
    expect(screen.getByText(/档位=直白/)).toBeTruthy()
    expect(screen.getByText(/合计 460 rune/)).toBeTruthy()
  })

  it('读取失败：如实提示不静默', async () => {
    mockedInventory.mockRejectedValue(new Error('请先打开项目'))
    render(<ContextInventoryPanel open chapterNum={0} onClose={vi.fn()} />)
    expect(await screen.findByText(/上下文清单读取失败|请先打开项目/)).toBeTruthy()
  })
})
