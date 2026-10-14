// FinalizeModal.test.tsx — 章节定稿回写弹窗关键路径（阶段八 8.4）
// 覆盖：草案三区渲染（合账/角色/伏笔）+ 建议制核心（打开只读不写）；
// 确认载荷形状（勾选过滤 + stage_recap 键 + branch 透传）；回执视图（成功计数
// + errors 留痕红显）；已定稿横幅（重新定稿语义）；空草案态。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

const draftFixture = () => ({
  chapterNum: 10,
  branch: '',
  finalized: false,
  characters: [
    { id: 'c1', name: '秦昭', field: 'status', text: '与沈砚决裂，独走' },
    { id: 'c2', name: '沈砚', field: 'status', text: '守井而居' },
  ],
  foreshadows: [
    { title: '井底徽记', content: '玄铁剑刻着秦家徽记', type: 'planted', category: 'item' },
    { title: '断指旧誓', content: '沈砚断指明誓的由来', type: 'resolved', category: 'mystery' },
  ],
  stageRecap: { stage: 1, start: 1, end: 10, text: '第一阶段：秦昭崛起与决裂。' },
  warnings: ['角色「无名氏」不在本书角色卡中，该条未纳入草案'],
})

vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return {
    ...actual,
    app: {
      FinalizeChapterDraft: vi.fn(),
      FinalizeChapterApply: vi.fn(),
      ChapterFinalizationGet: vi.fn(),
    },
  }
})

import FinalizeModal from './FinalizeModal'
import { app } from '../../gaea/lib/bridge'

describe('FinalizeModal 章节定稿回写（阶段八 8.4）', () => {
  const onFinalized = vi.fn()
  const onClose = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(app.FinalizeChapterDraft).mockResolvedValue(draftFixture())
    vi.mocked(app.FinalizeChapterApply).mockResolvedValue({
      finalized: true,
      applied: {
        stageRecap: { start: 1, end: 10, text: '第一阶段' },
        characters: 1,
        foreshadows: { plantedCount: 1, resolvedCount: 1 },
      },
      errors: ['伏笔同步: 一条回收未匹配'],
    })
    vi.mocked(app.ChapterFinalizationGet).mockResolvedValue(null)
  })

  it('草案三区渲染：合账可编辑/角色逐条/伏笔埋设回收徽标/警告如实展示', async () => {
    render(<FinalizeModal open chapterNum={10} branch="" onClose={onClose} onFinalized={onFinalized} />)

    await waitFor(() => expect(app.FinalizeChapterDraft).toHaveBeenCalledWith(10, ''))
    expect(await screen.findByTestId('finalize-recap-section')).toBeTruthy()
    const recap = screen.getByTestId('finalize-recap-text') as HTMLTextAreaElement
    expect(recap.value).toBe('第一阶段：秦昭崛起与决裂。')
    expect(screen.getByDisplayValue('与沈砚决裂，独走')).toBeTruthy()
    expect(screen.getByText(/埋设/)).toBeTruthy()
    expect(screen.getByText(/回收/)).toBeTruthy()
    expect(screen.getByText(/无名氏.*未纳入草案/)).toBeTruthy()
    // 建议制：草案阶段只调 Draft，不碰 Apply
    expect(app.FinalizeChapterApply).not.toHaveBeenCalled()
  })

  it('确认载荷：勾选过滤 + stage_recap 键 + 文本编辑生效 + branch 透传；回执计数与留痕红显', async () => {
    render(<FinalizeModal open chapterNum={10} branch="" onClose={onClose} onFinalized={onFinalized} />)

    // 取消第二条角色（沈砚）的勾选；编辑第一条文本
    await waitFor(() => expect(screen.getByDisplayValue('与沈砚决裂，独走')).toBeTruthy())
    const recap = screen.getByTestId('finalize-recap-text') as HTMLTextAreaElement
    fireEvent.change(recap, { target: { value: '改后的合账' } })
    fireEvent.change(screen.getByDisplayValue('与沈砚决裂，独走'), { target: { value: '决裂出走' } })
    // 勾选框：角色区 2 个、伏笔区 2 个——取消沈砚（角色第 2 个）
    const boxes = screen.getAllByRole('checkbox')
    fireEvent.click(boxes[2]) // 0=合账 1=秦昭 2=沈砚 3..4=伏笔

    fireEvent.click(screen.getByTestId('finalize-confirm'))

    await waitFor(() => expect(app.FinalizeChapterApply).toHaveBeenCalledTimes(1))
    const [num, branch, payloadJSON] = vi.mocked(app.FinalizeChapterApply).mock.calls[0] as [number, string, string]
    expect(num).toBe(10)
    expect(branch).toBe('')
    const payload = JSON.parse(payloadJSON)
    expect(payload.stage_recap).toEqual({ stage: 1, start: 1, end: 10, text: '改后的合账' })
    expect(payload.characters).toHaveLength(1)
    expect(payload.characters[0].id).toBe('c1')
    expect(payload.characters[0].text).toBe('决裂出走')
    expect(payload.foreshadows).toHaveLength(2)

    expect(await screen.findByTestId('finalize-receipt')).toBeTruthy()
    expect(screen.getByText(/角色状态 1 条/)).toBeTruthy()
    expect(screen.getByText(/一条回收未匹配/)).toBeTruthy()
    expect(onFinalized).toHaveBeenCalledTimes(1)
  })

  it('已定稿章：横幅提示重新定稿覆盖语义', async () => {
    vi.mocked(app.FinalizeChapterDraft).mockResolvedValue({
      ...draftFixture(),
      finalized: true,
      finalization: { version: 1, chapter_num: 10, finalized_at: '2026-10-14T23:00:00+08:00' },
    })
    render(<FinalizeModal open chapterNum={10} branch="" onClose={onClose} onFinalized={onFinalized} />)
    expect(await screen.findByText(/本章已定稿过/)).toBeTruthy()
  })

  it('空草案：无可回写变化时引导「仅做标记」', async () => {
    vi.mocked(app.FinalizeChapterDraft).mockResolvedValue({
      chapterNum: 3, branch: '', finalized: false,
      characters: [], foreshadows: [], warnings: [],
    })
    render(<FinalizeModal open chapterNum={3} branch="" onClose={onClose} onFinalized={onFinalized} />)
    expect(await screen.findByText(/仅做标记/)).toBeTruthy()
  })

  it('草案加载失败：错误如实显示，无确认钮可点', async () => {
    vi.mocked(app.FinalizeChapterDraft).mockRejectedValue(new Error('AI 客户端未就绪'))
    render(<FinalizeModal open chapterNum={10} branch="" onClose={onClose} onFinalized={onFinalized} />)
    expect(await screen.findByText(/AI 客户端未就绪/)).toBeTruthy()
    expect(screen.queryByTestId('finalize-confirm')).toBeNull()
  })
})
