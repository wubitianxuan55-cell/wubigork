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
  NovelChapterAnnotations: vi.fn().mockResolvedValue([]),
  // NovelB 门面具名导入（批次三组2 cast 族）
  GetNovelState: vi.fn().mockResolvedValue({ version: 1, entities: {} }),
  BuildNovelStatePatch: vi.fn().mockResolvedValue({ patch: 'ok' }),
  SettleNovelState: vi.fn().mockResolvedValue({ version: 2 }),
  DeSlopChapterAiTaste: vi.fn().mockResolvedValue({ done: false }),
  RewriteChapterAiTaste: vi.fn().mockResolvedValue({ done: false, reason: '无命中句' }),
  GetEntityRelations: vi.fn().mockResolvedValue({ nodes: [], edges: [] }),
  CancelCreateChapter: vi.fn().mockResolvedValue(true),
  // 反推任务化（v4.291）：Start 提交 / TaskGet 轮询 / Apply 落库
  NovelOutlineReconstructStart: vi.fn().mockResolvedValue({ taskId: 'tk-1', status: 'queued' }),
  NovelOutlineReconstructTaskGet: vi.fn().mockResolvedValue({ taskId: 'tk-1', status: 'queued' }),
  NovelOutlineReconstructApply: vi.fn().mockResolvedValue(6),
  NovelOutlineReconstruct: vi.fn().mockResolvedValue({ items: [], aiUsed: false }),
  taskCancel: vi.fn().mockResolvedValue(undefined),
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
    NovelChapterAnnotations: mocks.NovelChapterAnnotations,
    TaskCancel: mocks.taskCancel,
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
  NovelOutlineReconstructStart: mocks.NovelOutlineReconstructStart,
  NovelOutlineReconstructTaskGet: mocks.NovelOutlineReconstructTaskGet,
  NovelOutlineReconstructApply: mocks.NovelOutlineReconstructApply,
  NovelOutlineReconstruct: mocks.NovelOutlineReconstruct,
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

// 反推任务化（v4.291）：Start 入队 → 轮询 TaskGet 到 succeeded → 确认弹窗按
// 篇幅路由分叉文案 → Apply 落库卷级节点。
describe('CreatePage 反推任务化', () => {
  beforeEach(() => {
    useOutlineStore.setState({ outlines: [] })
    useAppStore.setState({ projectOpen: true, projectPath: 'C:/novel/test' })
    vi.mocked(mocks.NovelOutlineReconstructStart).mockClear().mockResolvedValue({ taskId: 'tk-9', status: 'queued' })
    vi.mocked(mocks.NovelOutlineReconstructApply).mockClear().mockResolvedValue(6)
    vi.mocked(mocks.NovelOutlineReconstructTaskGet).mockClear()
  })

  it('中篇预览轮询到 succeeded 后弹确认，文案含卷级节点，应用调 Apply', async () => {
    vi.mocked(mocks.NovelOutlineReconstructTaskGet).mockResolvedValue({
      taskId: 'tk-9',
      status: 'succeeded',
      preview: {
        aiUsed: false,
        projectTitle: '中篇',
        tier: 'mid',
        segmentSize: 10,
        items: [
          { chapterNumber: 1, chapterFrom: 1, chapterTo: 10, title: '第1-10章', summary: '开局段。' },
          { chapterNumber: 11, chapterFrom: 11, chapterTo: 20, title: '第11-20章', summary: '推进段。' },
        ],
      },
    } as never)
    render(<CreatePage />)

    const btn = await screen.findByRole('button', { name: 'AI 反推大纲' })
    fireEvent.click(btn)

    expect(await screen.findByText(/篇幅路由（中篇）/)).toBeTruthy()
    expect(screen.getByText(/2 个卷级节点/)).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: '应用到大纲' }))
    await waitFor(() => expect(mocks.NovelOutlineReconstructApply).toHaveBeenCalledTimes(1))
    const sent = JSON.parse(mocks.NovelOutlineReconstructApply.mock.calls[0][0])
    expect(sent[0].chapterFrom).toBe(1)
    expect(sent[1].chapterTo).toBe(20)
  })

  it('任务失败：错误透出且不弹确认', async () => {
    vi.mocked(mocks.NovelOutlineReconstructTaskGet).mockResolvedValue({
      taskId: 'tk-9',
      status: 'failed',
      error: '这本书还没有已写章节，无法反推大纲',
    } as never)
    render(<CreatePage />)

    const btn = await screen.findByRole('button', { name: 'AI 反推大纲' })
    fireEvent.click(btn)
    expect(await screen.findByText(/这本书还没有已写章节/)).toBeTruthy()
    expect(screen.queryByRole('button', { name: '应用到大纲' })).toBeNull()
  })
})

// tail×反推串联（v4.292）：书架派发 novel:auto-reconstruct 事件 → 本页自动开跑反推。
describe('CreatePage 反推串联事件', () => {
  it('novel:auto-reconstruct 事件触发任务化反推并弹确认', async () => {
    vi.mocked(mocks.NovelOutlineReconstructStart).mockResolvedValue({ taskId: 'tk-ev', status: 'queued' })
    vi.mocked(mocks.NovelOutlineReconstructTaskGet).mockResolvedValue({
      taskId: 'tk-ev',
      status: 'succeeded',
      preview: {
        aiUsed: false,
        projectTitle: '中篇',
        tier: 'mid',
        segmentSize: 10,
        items: [{ chapterNumber: 1, chapterFrom: 1, chapterTo: 10, title: '第1-10章', summary: '开局段。' }],
      },
    } as never)
    render(<CreatePage />)

    await act(async () => {
      window.dispatchEvent(new CustomEvent('novel:auto-reconstruct'))
      await new Promise((r) => setTimeout(r, 50))
    })

    expect(await screen.findByText(/篇幅路由（中篇）/)).toBeTruthy()
    // 应用：事件触发的链路终点是 Apply 收到骨架条目
    vi.mocked(mocks.NovelOutlineReconstructApply).mockClear()
    const okBtn = document.querySelector('.ant-modal-confirm .ant-modal-confirm-btns button:last-child')
    expect(okBtn).toBeTruthy()
    fireEvent.click(okBtn as Element)
    await waitFor(() => expect(mocks.NovelOutlineReconstructApply).toHaveBeenCalledTimes(1))
  })
})

// 反推取消（v4.340）：轮询期「取消反推」可见 → 点击停止等待 + 请求后端协作
// 取消任务（完成竞态下取消失败被吞）；同步回落路径无任务 id，不出取消入口。
describe('CreatePage 反推取消', () => {
  beforeEach(() => {
    useOutlineStore.setState({ outlines: [] })
    useAppStore.setState({ projectOpen: true, projectPath: 'C:/novel/test' })
    vi.mocked(mocks.NovelOutlineReconstructStart).mockClear().mockResolvedValue({ taskId: 'tk-c', status: 'queued' })
    vi.mocked(mocks.NovelOutlineReconstructTaskGet).mockClear().mockResolvedValue({ taskId: 'tk-c', status: 'queued' })
    mocks.taskCancel.mockClear()
  })

  it('轮询期出「取消反推」，点击停止等待并请求取消任务，不弹确认', async () => {
    render(<CreatePage />)
    const btn = await screen.findByRole('button', { name: 'AI 反推大纲' })
    fireEvent.click(btn)
    // taskId 到手后出现取消入口（Start 拿到 tk-c）
    fireEvent.click(await screen.findByTestId('reconstruct-cancel'))
    // 取消后：消息落出 + 后端取消被请求（≤3s 轮询 sleep 在 5s RTL 窗口内）
    expect(await screen.findByText(/已取消反推等待/)).toBeTruthy()
    await waitFor(() => expect(mocks.taskCancel).toHaveBeenCalledWith('tk-c'))
    expect(screen.queryByRole('button', { name: '应用到大纲' })).toBeNull()
  })

  it('同步回落路径（任务队列不可用）不出取消入口、不调 Cancel', async () => {
    vi.mocked(mocks.NovelOutlineReconstructStart).mockRejectedValue(new Error('任务队列不可用，请直接使用同步反推'))
    render(<CreatePage />)
    const btn = await screen.findByRole('button', { name: 'AI 反推大纲' })
    fireEvent.click(btn)
    expect(await screen.findByText(/反推结果为空/)).toBeTruthy()
    expect(screen.queryByTestId('reconstruct-cancel')).toBeNull()
    expect(mocks.taskCancel).not.toHaveBeenCalled()
  })
})
