import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'

// Wails 绑定 mock：7 个 wailsjsCompat 直调（GetWorldview/GetStats/GetChapterBranch/
// QuickBrainstormBranches/CreateChapter/DeleteOutlineNode/SaveChapterBranchContent）
// 已迁为 bridge app 调用；cast 族方法（GetNovelState/BuildNovelStatePatch/
// SettleNovelState/DeSlop/Rewrite/GetEntityRelations/CancelCreateChapter）走 NovelB
// 门面具名导入，故分别 mock bridge 与 NovelB 两个模块。bridge mock 用 importOriginal
// 保留原模块、仅替换 app 的上述方法；未 mock 的方法回落真实代理（dev mock）。
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
  // NovelB 门面具名导入（批次三组2 cast 族）
  GetNovelState: vi.fn().mockResolvedValue({ version: 1, entities: {} }),
  BuildNovelStatePatch: vi.fn().mockResolvedValue({ patch: 'ok' }),
  SettleNovelState: vi.fn().mockResolvedValue({ version: 2 }),
  DeSlopChapterAiTaste: vi.fn().mockResolvedValue({ done: false }),
  RewriteChapterAiTaste: vi.fn().mockResolvedValue({ done: false, reason: '无命中句' }),
  GetEntityRelations: vi.fn().mockResolvedValue({ nodes: [], edges: [] }),
  CancelCreateChapter: vi.fn().mockResolvedValue(true),
  // 平台评审批次（v4.282）：档位清单 + 单章报告
  NovelReviewPlatforms: vi.fn().mockResolvedValue([
    { id: 'general', label: '通用', form: 'chapter' },
    { id: 'fanqie', label: '番茄小说', form: 'chapter' },
  ]),
  NovelChapterReview: vi.fn().mockResolvedValue({
    chapterNum: 1, platform: 'general', platformLabel: '通用', words: 2380,
    verdict: 'CONCERNS', counts: { S1: 0, S2: 1, S3: 0, S4: 0 },
    dimensions: [{ id: 'opening_freshness', label: '开篇钩子', verdict: 'warn', severity: 'S2', detail: '前 3 段只有悬念铺垫' }],
    advisories: ['读者为什么翻下一页？'],
  }),
}))
vi.mock('../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../gaea/lib/bridge')>()
  const appStubs = {
    GetWorldview: mocks.GetWorldview,
    GetStats: mocks.GetStats,
    ListSkills: mocks.ListSkills,
    GetChapterBranch: mocks.GetChapterBranch,
    QuickBrainstormBranches: mocks.QuickBrainstormBranches,
    CreateChapter: mocks.CreateChapter,
    DeleteOutlineNode: mocks.DeleteOutlineNode,
    SaveChapterBranchContent: mocks.SaveChapterBranchContent,
    SaveCharactersBatch: mocks.SaveCharactersBatch,
  }
  return {
    ...actual,
    app: new Proxy(appStubs, {
      get(target, prop) {
        if (prop in target) return Reflect.get(target, prop)
        return (actual.app as unknown as Record<string, unknown>)[String(prop)]
      },
    }),
  }
})
vi.mock('../../wailsjs/go/app/NovelB', () => ({
  GetNovelState: mocks.GetNovelState,
  BuildNovelStatePatch: mocks.BuildNovelStatePatch,
  SettleNovelState: mocks.SettleNovelState,
  DeSlopChapterAiTaste: mocks.DeSlopChapterAiTaste,
  RewriteChapterAiTaste: mocks.RewriteChapterAiTaste,
  GetEntityRelations: mocks.GetEntityRelations,
  CancelCreateChapter: mocks.CancelCreateChapter,
  NovelReviewPlatforms: mocks.NovelReviewPlatforms,
  NovelChapterReview: mocks.NovelChapterReview,
}))

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
  it('默认使用 story-deslop 技能生成章节', async () => {
    await startGeneration()
    expect(mocks.CreateChapter).toHaveBeenCalledWith(
      expect.any(String), '', '主角觉醒', 0, '', 'story-deslop', 5000, 0,
    )
  })

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

  it('构思剧情分支在后台进行（不弹阻塞弹窗），完成后弹窗确认并生成', async () => {
    vi.mocked(mocks.QuickBrainstormBranches).mockResolvedValue({
      branches: [{ title: '帝国阴谋', summary: '朝堂暗流涌动' }],
    })
    render(<CreatePage />)

    // 点击「构思剧情方向」：构思完成前不出现弹窗，可继续操作
    fireEvent.click(await screen.findByRole('button', { name: /构思剧情方向/ }))
    expect(screen.queryByRole('dialog', { name: /剧情方向/ })).toBeNull()

    // 构思完成后弹出确认弹窗，展示分支
    const dialog = await screen.findByRole('dialog', { name: /剧情方向/ })
    expect(dialog.textContent).toContain('帝国阴谋')
    expect(dialog.textContent).toContain('朝堂暗流涌动')

    // 选择分支并生成 → CreateChapter 携带分支拼装的剧情要求
    fireEvent.click(screen.getByText(/帝国阴谋/))
    fireEvent.click(screen.getByRole('button', { name: /^生\s*成$/ }))
    await waitFor(() => {
      expect(mocks.CreateChapter).toHaveBeenCalledWith(
        expect.any(String), '', expect.stringContaining('帝国阴谋'), 0, '', 'story-deslop', 5000, 0,
      )
    })
    await waitFor(() => expect(screen.queryByRole('dialog', { name: /剧情方向/ })).toBeNull())
  })

  it('平台评审：rail 入口打开面板并拉档位清单；无当前章时评审按钮禁用', async () => {
    render(<CreatePage />)
    fireEvent.click(await screen.findByRole('button', { name: '平台评审' }))
    // 面板打开即拉档位（NovelReviewPlatforms 走 NovelB 门面具名导入）。
    await waitFor(() => expect(mocks.NovelReviewPlatforms).toHaveBeenCalledTimes(1))
    expect(await screen.findByText(/确定性评审，零模型调用/)).toBeTruthy()
    // 空大纲（beforeEach 置 outlines: []）→ 无当前章，评审按钮禁用且不调用评审绑定。
    const btn = screen.getByRole('button', { name: '评审当前章' }) as HTMLButtonElement
    expect(btn.disabled).toBe(true)
    expect(mocks.NovelChapterReview).not.toHaveBeenCalled()
  })
})
