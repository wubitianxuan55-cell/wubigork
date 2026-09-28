import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act, within } from '@testing-library/react'
import { Modal } from 'antd'

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
  // 章节计划闭环（刀1 线D）：Get/Save/Propose/Deviation + 硬闸预检（经 app 代理调用）
  NovelChapterPlanGet: vi.fn().mockResolvedValue(null),
  NovelChapterPlanSave: vi.fn().mockResolvedValue(undefined),
  NovelChapterPlanPropose: vi.fn().mockResolvedValue(null),
  NovelChapterPlanDeviation: vi.fn().mockResolvedValue(null),
  NovelChapterGatePrecheck: vi.fn().mockResolvedValue({
    chapterNum: 1, allowed: true, hasPlan: true, missing: [], planProblems: [], outlineIssues: [], blocking: false,
  }),
  // 刀1 线B：写前硬闸显式覆盖入口（9 参 allowOverride）
  CreateChapterWithOverride: vi.fn().mockResolvedValue({ streaming: true, chapterNum: 1, nodeId: 'n1', branch: '' }),
  /** true = 模拟「覆盖入口绑定未就绪」（收口前的构建），用于断言诚实降级 */
  noOverrideBinding: false,
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
    NovelChapterPlanGet: mocks.NovelChapterPlanGet,
    NovelChapterPlanSave: mocks.NovelChapterPlanSave,
    NovelChapterPlanPropose: mocks.NovelChapterPlanPropose,
    NovelChapterPlanDeviation: mocks.NovelChapterPlanDeviation,
    NovelChapterGatePrecheck: mocks.NovelChapterGatePrecheck,
    CreateChapterWithOverride: mocks.CreateChapterWithOverride,
  }
  return {
    ...actual,
    app: new Proxy(appStubs, {
      get(target, prop) {
        // 覆盖入口绑定未就绪的模拟（收口前的构建）：类型检测应落到普通入口
        if (prop === 'CreateChapterWithOverride' && mocks.noOverrideBinding) return undefined
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
// v4.421.0 契约：事件订阅走 subscribeWailsEvent，退订用 EventsOn 返回的「只摘自己」
// 的 off 函数（仓规禁止 EventsOff 全清通道）——故断言改为记录 off 调用。
const runtimeOffs: string[] = []
const EventsOn = vi.fn((name: string, handler: Listener) => {
  runtimeListeners.set(name, handler)
  return () => { runtimeOffs.push(name); runtimeListeners.delete(name) }
})
const EventsOff = vi.fn((name: string) => { runtimeListeners.delete(name) })

function emit(name: string, payload: unknown) {
  const handler = runtimeListeners.get(name)
  if (handler) act(() => handler(payload))
}

beforeEach(() => {
  runtimeListeners.clear()
  runtimeOffs.length = 0
  EventsOn.mockClear()
  EventsOff.mockClear()
  Object.defineProperty(window, 'runtime', { configurable: true, writable: true, value: { EventsOn, EventsOff } })
  useOutlineStore.setState({ outlines: [] })
  useAppStore.setState({ projectOpen: true, projectPath: 'C:/novel/test' })
  vi.mocked(mocks.CancelCreateChapter).mockResolvedValue(true)
  // 硬闸预检默认放行（缺计划用例自行覆盖）；CreateChapter 默认成功
  vi.mocked(mocks.NovelChapterGatePrecheck).mockResolvedValue({
    chapterNum: 1, allowed: true, hasPlan: true, missing: [], planProblems: [], outlineIssues: [], blocking: false,
  })
  mocks.noOverrideBinding = false
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
    expect(runtimeOffs).toContain('create-chapter-stream')
  })

  it.each([
    ['done', { type: 'done', chapterNum: 1, branch: '', total: 5000 }],
    ['error', { type: 'error', error: '生成失败' }],
    ['cancelled', { type: 'cancelled', chapterNum: 1, branch: '', nodeId: 'n1', total: 3, content: '部分' }],
  ])('终态事件 %s 收尾：generating 复位且退订监听（无悬挂）', async (_label, payload) => {
    await startGeneration()
    emit('create-chapter-stream', payload)
    await waitFor(() => expect(screen.queryByRole('button', { name: /停止生成/ })).toBeNull())
    expect(runtimeOffs).toContain('create-chapter-stream')
  })

  it('生成中卸载组件：流式监听被退订（无悬挂）', async () => {
    const { unmount } = render(<CreatePage />)
    const plot = await screen.findByPlaceholderText(/或直接输入剧情要求/)
    fireEvent.change(plot, { target: { value: '主角觉醒' } })
    fireEvent.click(screen.getByRole('button', { name: /按剧情要求直接生成/ }))
    await screen.findByRole('button', { name: /停止生成/ })
    unmount()
    expect(runtimeOffs).toContain('create-chapter-stream')
  })

  it('CancelCreateChapter 返回 false（幂等/未开始）：本地兜底收尾，UI 不悬挂', async () => {
    vi.mocked(mocks.CancelCreateChapter).mockResolvedValue(false)
    await startGeneration()
    fireEvent.click(screen.getByRole('button', { name: /停止生成/ }))
    await waitFor(() => expect(screen.queryByRole('button', { name: /停止生成/ })).toBeNull())
    expect(runtimeOffs).toContain('create-chapter-stream')
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

// ── v4.421.0 小说板块优化批（线1）：未保存保护 / 并列分支不绑 Esc / 载入失败可见化 /
// 创作参数持久化。每条都是防复发守卫（旧行为见各用例注释）。
describe('CreatePage 未保存保护与体验收口（v4.421.0）', () => {
  const confirms = () => Array.from(document.querySelectorAll<HTMLElement>('.ant-modal-confirm'))
  /** 取「最新弹窗」的限定查询器（imperative Modal 的 DOM 不随 destroy 立即卸载）。 */
  async function latestConfirm(before: number) {
    await waitFor(() => expect(confirms().length).toBeGreaterThan(before))
    const all = confirms()
    return within(all[all.length - 1])
  }
  const editorText = () => (screen.getByPlaceholderText(/AI 将在此流式呈现正文/) as HTMLTextAreaElement).value

  beforeEach(() => {
    Modal.destroyAll()
    mocks.GetChapterBranch.mockClear().mockResolvedValue({ content: '第一章正文' })
    mocks.SaveChapterBranchContent.mockClear().mockResolvedValue(undefined)
    mocks.CreateChapter.mockClear().mockResolvedValue({ streaming: true, chapterNum: 1, nodeId: 'n1', branch: '' })
    useOutlineStore.setState({
      outlines: [
        { id: 'n1', order_index: 1, title: '第1章', status: 'done', parent_id: '', summary: '' },
        { id: 'n2', order_index: 2, title: '第2章', status: 'done', parent_id: '', summary: '' },
      ] as never,
    })
  })
  afterEach(async () => {
    Modal.destroyAll()
    localStorage.removeItem('gaea.novel.genPrefs')
    await new Promise((r) => setTimeout(r, 0))
  })

  it('切章前有未保存修改：弹三选确认；点「取消」不切换、正文不丢', async () => {
    render(<CreatePage />)
    fireEvent.click(await screen.findByRole('button', { name: '选择章节 第1章' }))
    await waitFor(() => expect(editorText()).toBe('第一章正文'))
    fireEvent.change(screen.getByPlaceholderText(/AI 将在此流式呈现正文/), { target: { value: '我手写的改动' } })

    const before = confirms().length
    fireEvent.click(screen.getByRole('button', { name: '选择章节 第2章' }))
    const scoped = await latestConfirm(before)
    expect(scoped.getByText(/切到「第2章」会丢弃当前修改/)).toBeTruthy()

    fireEvent.click(scoped.getByRole('button', { name: /取\s*消/ }))
    await new Promise((r) => setTimeout(r, 0))
    // 未切换：仍是第 1 章的正文，且没有第二次载入请求
    expect(editorText()).toBe('我手写的改动')
    expect(mocks.GetChapterBranch).toHaveBeenCalledTimes(1)
  })

  it('切章确认点「先保存」：先落盘当前章，再载入目标章', async () => {
    render(<CreatePage />)
    fireEvent.click(await screen.findByRole('button', { name: '选择章节 第1章' }))
    await waitFor(() => expect(editorText()).toBe('第一章正文'))
    fireEvent.change(screen.getByPlaceholderText(/AI 将在此流式呈现正文/), { target: { value: '我手写的改动' } })

    const before = confirms().length
    fireEvent.click(screen.getByRole('button', { name: '选择章节 第2章' }))
    const scoped = await latestConfirm(before)
    fireEvent.click(scoped.getByRole('button', { name: /先\s*保\s*存/ }))

    await waitFor(() => expect(mocks.SaveChapterBranchContent).toHaveBeenCalledWith(1, '', '我手写的改动'))
    await waitFor(() => expect(mocks.GetChapterBranch).toHaveBeenCalledTimes(2))
  })

  it('生成下一章：并列分支显式成按钮，点「取消」不触发生成（旧实现把追加挂在 onCancel 上）', async () => {
    render(<CreatePage />)
    const plot = await screen.findByPlaceholderText(/或直接输入剧情要求/)
    fireEvent.change(plot, { target: { value: '主角觉醒' } })

    const before = confirms().length
    fireEvent.click(screen.getByRole('button', { name: /按剧情要求直接生成/ }))
    const scoped = await latestConfirm(before)
    expect(scoped.getByRole('button', { name: '覆盖下一章' })).toBeTruthy()
    expect(scoped.getByRole('button', { name: '作为分支追加' })).toBeTruthy()

    fireEvent.click(scoped.getByRole('button', { name: /取\s*消/ }))
    await new Promise((r) => setTimeout(r, 0))
    expect(mocks.CreateChapter).not.toHaveBeenCalled()
  })

  it('章节载入失败：如实报错（旧实现静默清空编辑区，作者会误以为该章为空）', async () => {
    mocks.GetChapterBranch.mockRejectedValueOnce(new Error('磁盘读取失败'))
    render(<CreatePage />)
    fireEvent.click(await screen.findByRole('button', { name: '选择章节 第1章' }))
    expect(await screen.findByText(/第 1 章载入失败/)).toBeTruthy()
  })

  it('创作参数持久化：预置值生效，且改动后写回 localStorage', async () => {
    localStorage.setItem('gaea.novel.genPrefs', JSON.stringify({ minWords: 7777, temperature: 0.8, skill: 'story-deslop' }))
    render(<CreatePage />)
    expect(await screen.findByDisplayValue('7777')).toBeTruthy()
    await waitFor(() => {
      const saved = JSON.parse(localStorage.getItem('gaea.novel.genPrefs') || '{}')
      expect(saved.minWords).toBe(7777)
      expect(saved.temperature).toBe(0.8)
    })
  })

  it('cancelled 残稿另存（线0 Go 侧语义）：提示落点，且不把残稿标成已保存', async () => {
    useOutlineStore.setState({ outlines: [] })
    render(<CreatePage />)
    const plot = await screen.findByPlaceholderText(/或直接输入剧情要求/)
    fireEvent.change(plot, { target: { value: '主角觉醒' } })
    fireEvent.click(screen.getByRole('button', { name: /按剧情要求直接生成/ }))
    await screen.findByRole('button', { name: /停止生成/ })

    emit('create-chapter-stream', {
      type: 'cancelled', chapterNum: 1, branch: '', total: 120,
      content: '半截正文', partialSaved: true, partialPath: 'D:/novel/001.partial-20260928.md',
    })
    expect(await screen.findByText(/未覆盖正稿，已另存为残稿/)).toBeTruthy()
    expect(screen.getByText(/001\.partial-20260928\.md/)).toBeTruthy()
  })

  it('工具轨分组：三组语义标签在册，13 个入口一个不少', async () => {
    render(<CreatePage />)
    expect(await screen.findByText('质检')).toBeTruthy()
    expect(screen.getByText('文本')).toBeTruthy()
    expect(screen.getByText('结构')).toBeTruthy()
    for (const label of [
      '章节分析', '全书体检', '平台评审', '文风指纹',
      '一键去味', '高级去味', '整章重写', '重写历史',
      '章节计划', 'AI 反推大纲', '叙事状态', '全文脑图', '提示词工坊',
    ]) {
      expect(screen.getByRole('button', { name: label })).toBeTruthy()
    }
  })
})

// ── 章节计划闭环硬闸（刀1 线D）：生成前预检 blocking → 不发起生成，改为弹窗二选；
// 「立即生成计划草案」展开章节计划卡；「仍然生成（跳过硬闸）」走线B 的专用覆盖入口
// NovelB.CreateChapterWithOverride(..., allowOverride=true)（旧 8 参 CreateChapter 恒等于
// allowOverride=false，覆盖若走旧入口会被后端硬闸照拒）。
describe('CreatePage 章节计划硬闸（刀1 线D）', () => {
  const blockedGate = {
    chapterNum: 1, allowed: false, hasPlan: false, missing: ['叙事目标', '冲突类型'],
    planProblems: [{ code: 'plan_goal_empty', severity: 'S1', message: '叙事目标为空' }],
    outlineIssues: [], blocking: true,
  }

  beforeEach(() => {
    useOutlineStore.setState({ outlines: [] })
    useAppStore.setState({ projectOpen: true, projectPath: 'C:/novel/test' })
    vi.mocked(mocks.CreateChapter).mockClear().mockResolvedValue({ streaming: true, chapterNum: 1, nodeId: 'n1', branch: '' })
    vi.mocked(mocks.CreateChapterWithOverride).mockClear().mockResolvedValue({ streaming: true, chapterNum: 1, nodeId: 'n1', branch: '' })
    mocks.noOverrideBinding = false
    vi.mocked(mocks.NovelChapterGatePrecheck).mockReset()
    vi.mocked(mocks.NovelChapterGatePrecheck).mockResolvedValue(blockedGate)
  })

  /** 页面渲染 → 填剧情要求 → 点「按剧情要求直接生成」（触发硬闸预检）。 */
  async function clickDirectGenerate() {
    render(<CreatePage />)
    const plot = await screen.findByPlaceholderText(/或直接输入剧情要求/)
    fireEvent.change(plot, { target: { value: '主角觉醒' } })
    fireEvent.click(screen.getByRole('button', { name: /按剧情要求直接生成/ }))
  }

  /**
   * 取最新硬闸弹窗的限定查询器。坑复训：antd Modal 的 DOM 不随 destroy 立即卸载
   * （rc-motion 的离场动画在 jsdom 下不结束），且所有弹窗共用同一个 aria-labelledby
   * id —— 前序用例残留的 Modal.confirm 会让 `role=dialog + name` 解析到**旧弹窗的标题**。
   * 故按 rootClassName 定位并取最后一个，断言一律 within 限定。
   */
  async function latestGateModal() {
    await waitFor(() => expect(document.querySelectorAll('.novel-plan-gate-modal').length).toBeGreaterThan(0))
    const all = document.querySelectorAll<HTMLElement>('.novel-plan-gate-modal')
    return within(all[all.length - 1])
  }

  it('缺计划：「仍然生成（跳过硬闸）」经 CreateChapterWithOverride(allowOverride=true) 放行', async () => {
    await clickDirectGenerate()
    const gate = await latestGateModal()
    expect(gate.getByText(/本章尚未制定章节计划/)).toBeTruthy()
    expect(gate.getByText(/缺失：叙事目标、冲突类型/)).toBeTruthy()
    expect(gate.getByText(/\[S1\] 叙事目标为空/)).toBeTruthy()
    expect(mocks.CreateChapter).not.toHaveBeenCalled()
    expect(mocks.CreateChapterWithOverride).not.toHaveBeenCalled()
    // 预检按目标章号调用（空大纲 → 顺延第 1 章）
    expect(mocks.NovelChapterGatePrecheck).toHaveBeenCalledWith(1)

    fireEvent.click(gate.getByRole('button', { name: '仍然生成（跳过硬闸）' }))
    await waitFor(() => expect(mocks.CreateChapterWithOverride).toHaveBeenCalledTimes(1))
    // 覆盖走专用入口（旧 8 参 CreateChapter 恒等于 allowOverride=false）
    expect(mocks.CreateChapterWithOverride).toHaveBeenCalledWith(
      expect.any(String), '', '主角觉醒', 0, '', 'story-deslop', 5000, 0, true,
    )
    expect(mocks.CreateChapter).not.toHaveBeenCalled()
  })

  it('覆盖入口绑定未就绪：如实警告并回落普通入口（不假装已覆盖）', async () => {
    mocks.noOverrideBinding = true
    await clickDirectGenerate()
    const gate = await latestGateModal()
    fireEvent.click(gate.getByRole('button', { name: '仍然生成（跳过硬闸）' }))
    expect(await screen.findByText(/跳过计划硬闸的绑定未就绪/)).toBeTruthy()
    await waitFor(() => expect(mocks.CreateChapter).toHaveBeenCalledTimes(1))
    expect(mocks.CreateChapterWithOverride).not.toHaveBeenCalled()
  })

  it('缺计划：弹窗「立即生成计划草案」展开章节计划卡（草案入口在卡内，落盘仍待审批）', async () => {
    await clickDirectGenerate()
    const gate = await latestGateModal()
    fireEvent.click(gate.getByRole('button', { name: '立即生成计划草案' }))

    // 卡片与硬闸弹窗同页共存：弹窗 DOM 不随关闭立即卸载（rc-motion），断言一律 within 限定
    const card = await screen.findByTestId('chapter-plan-card')
    await act(async () => { await new Promise((r) => setTimeout(r, 30)) })
    expect(within(card).getByText(/缺失：叙事目标、冲突类型/)).toBeTruthy()
    expect(within(card).getByRole('button', { name: /生成计划草案/ })).toBeTruthy()
    // 计划卡章号与硬闸目标章号同源（空大纲 → 第 1 章）
    expect(mocks.NovelChapterPlanGet).toHaveBeenCalledWith(1)
    expect(mocks.NovelChapterPlanSave).not.toHaveBeenCalled()
    expect(mocks.CreateChapter).not.toHaveBeenCalled()
  })

  it('硬闸弹窗点「取消」：不生成、不落盘', async () => {
    await clickDirectGenerate()
    const gate = await latestGateModal()
    fireEvent.click(gate.getByRole('button', { name: /取\s*消/ }))
    await new Promise((r) => setTimeout(r, 0))
    expect(mocks.CreateChapter).not.toHaveBeenCalled()
  })

  it('预检接口不可用：不锁死作者——如实提示后照常生成', async () => {
    vi.mocked(mocks.NovelChapterGatePrecheck).mockRejectedValue(new Error('章节计划预检接口未就绪'))
    await clickDirectGenerate()
    expect(await screen.findByText(/章节计划预检未执行/)).toBeTruthy()
    await waitFor(() => expect(mocks.CreateChapter).toHaveBeenCalledTimes(1))
  })
})
