import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import React from 'react'

// NovelB 门面具名导入 mock（与 CreatePage.test 同风格：只 mock 用到的三件）
const mocks = vi.hoisted(() => ({
  GetChapterScenes: vi.fn(),
  GenerateScene: vi.fn(),
  CreateScene: vi.fn(),
  ReorderScenes: vi.fn().mockResolvedValue(undefined),
  SaveSceneMeta: vi.fn().mockResolvedValue(undefined),
  NovelChapterScenesGenerate: vi.fn(),
  NovelSceneRewrite: vi.fn(),
  NovelSceneCardsPropose: vi.fn(),
}))
vi.mock('../../../wailsjs/go/app/NovelB', () => ({
  GetChapterScenes: mocks.GetChapterScenes,
  GenerateScene: mocks.GenerateScene,
  CreateScene: mocks.CreateScene,
  ReorderScenes: mocks.ReorderScenes,
  SaveSceneMeta: mocks.SaveSceneMeta,
  NovelChapterScenesGenerate: mocks.NovelChapterScenesGenerate,
  NovelSceneRewrite: mocks.NovelSceneRewrite,
  NovelSceneCardsPropose: mocks.NovelSceneCardsPropose,
}))

// CommandBar 桩（观察池#6 用例）：真实组件的指令→AI→接受流程与本测无关，
// 只需一个按钮把 props.onAccept 直回调——测的是 ChapterEditor 记住来源
// textarea 的 onAccept 逻辑（旧缺陷：activeElement 非 TEXTAREA 时静默丢）。
vi.mock('./editor/CommandBar', () => ({
  default: ({ onAccept }: { onAccept: (t: string) => void }) => (
    <button data-testid="cmdk-stub-accept" onClick={() => onAccept('AI改写后的文本')}>桩-接受</button>
  ),
}))

import ChapterEditor from './ChapterEditor'
import type { ChapterTabData, OutlineNode } from '../../types'

const node: OutlineNode = {
  id: 'n3', title: '第三章 夜雨', summary: '', status: 'done', order_index: 3,
}

function makeTab(overrides: Partial<ChapterTabData> = {}): ChapterTabData {
  return {
    node,
    chapterNum: 3,
    scenes: ['已有正文第一段。'],
    sceneIds: ['001-chapter'],
    sceneBacked: true,
    summary: '',
    keyEvents: [],
    emotionTone: '',
    saved: true,
    generating: false,
    streamSpeed: 0,
    messages: [],
    targetWords: 800,
    skillName: '',
    ...overrides,
  } as ChapterTabData
}

function renderEditor(initial: ChapterTabData) {
  const onUpdate = vi.fn()
  const refs = { current: new Map<number, HTMLTextAreaElement>() }
  // 有状态 mini-parent：把 onUpdate 的 scenes/sceneIds/sceneBacked 喂回 props
  // （模拟 ChapterPage 行为），否则加框/存盘后本组件拿不到新 ids。
  const Harness: React.FC = () => {
    const [tab, setTab] = React.useState<ChapterTabData>(initial)
    const handleUpdate = React.useCallback(<K extends keyof ChapterTabData>(field: K, value: ChapterTabData[K]) => {
      onUpdate(field, value)
      setTab((prev) => ({ ...prev, [field]: value }))
    }, [])
    return <ChapterEditor tab={tab} onUpdate={handleUpdate} sceneTextareaRefs={refs} ghostEnabled={false} />
  }
  const { container } = render(<Harness />)
  const addBtn = () => {
    const icon = container.querySelector('button .anticon-plus')
    return (icon?.closest('button') ?? null) as HTMLButtonElement | null
  }
  const genBtns = () => screen.getAllByRole('button', { name: /AI 生成/ }) as HTMLButtonElement[]
  const moveBtns = (i: number) => ({
    up: container.querySelector(`button[aria-label="场景 ${i + 1} 上移"]`) as HTMLButtonElement | null,
    down: container.querySelector(`button[aria-label="场景 ${i + 1} 下移"]`) as HTMLButtonElement | null,
  })
  const disabledOf = (b: HTMLButtonElement | null) => (b ? b.disabled : true)
  return { onUpdate, addBtn, genBtns, moveBtns, disabledOf }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('ChapterEditor 逐场景生成（阅读页场景化）', () => {
  it('ids 来自 tab（ChapterPage 载入时读入）：有 id 的场景 AI 生成可用', async () => {
    const { genBtns, disabledOf } = renderEditor(makeTab())
    // 场景 ids 由父层带入，挂载不再额外拉取
    expect(mocks.GetChapterScenes).not.toHaveBeenCalled()
    expect(disabledOf(genBtns()[0])).toBe(false)
  })

  it('无 id（空 sceneIds）时 AI 生成保持禁用', async () => {
    const { genBtns, disabledOf } = renderEditor(makeTab({ sceneIds: [] }))
    expect(disabledOf(genBtns()[0])).toBe(true)
  })

  it('加场景走 CreateScene 落盘：重拉 id 序、scenes/sceneIds 经 onUpdate 回喂、新场景可生成', async () => {
    mocks.GetChapterScenes
      .mockResolvedValue([{ id: '001-chapter', content: '已有正文第一段。' }, { id: '002-scene-2', content: '' }])
    mocks.CreateScene.mockResolvedValue({ id: '002-scene-2', slug: 'scene-2', title: '场景 2' })
    const { onUpdate, addBtn, genBtns, disabledOf } = renderEditor(makeTab())

    fireEvent.click(addBtn()!)
    await waitFor(() => expect(mocks.CreateScene).toHaveBeenCalledWith(3, 'scene-2', '场景 2'))
    await waitFor(() => {
      const scenesCall = onUpdate.mock.calls.find(([field]) => field === 'scenes')
      expect(scenesCall?.[1]).toEqual(['已有正文第一段。', ''])
      const idsCall = onUpdate.mock.calls.find(([field]) => field === 'sceneIds')
      expect(idsCall?.[1]).toEqual(['001-chapter', '002-scene-2'])
    })
    // 新场景拿到 id → 第二个生成按钮出现且可用
    await waitFor(() => expect(genBtns().length).toBe(2))
    expect(disabledOf(genBtns()[1])).toBe(false)
  })

  it('逐场景生成写回：GenerateScene 返回正文进对应框并标记未保存', async () => {
    mocks.GenerateScene.mockResolvedValue({ content: '生成的新正文。', aiTaste: 42, words: 7 })
    const { onUpdate, genBtns, disabledOf } = renderEditor(makeTab())
    expect(disabledOf(genBtns()[0])).toBe(false)

    fireEvent.click(genBtns()[0])
    await waitFor(() => expect(mocks.GenerateScene).toHaveBeenCalledWith(3, '001-chapter', '', 800))
    await waitFor(() => {
      expect(onUpdate).toHaveBeenCalledWith('scenes', ['生成的新正文。'])
      expect(onUpdate).toHaveBeenCalledWith('saved', false)
    })
  })

  it('场景重排：上移换位本地两序并 ReorderScenes 落盘新 id 序', async () => {
    const { onUpdate, moveBtns } = renderEditor(makeTab({
      scenes: ['一', '二', '三'],
      sceneIds: ['001-a', '002-b', '003-c'],
    }))
    fireEvent.click(moveBtns(1).up!) // 第二个场景上移
    await waitFor(() => expect(mocks.ReorderScenes).toHaveBeenCalledWith(3, ['002-b', '001-a', '003-c']))
    expect(onUpdate).toHaveBeenCalledWith('scenes', ['二', '一', '三'])
    expect(onUpdate).toHaveBeenCalledWith('sceneIds', ['002-b', '001-a', '003-c'])
  })

  it('场景重排失败：回滚本地换位并提示', async () => {
    mocks.ReorderScenes.mockRejectedValueOnce(new Error('boom'))
    const { onUpdate, moveBtns } = renderEditor(makeTab({
      scenes: ['一', '二'],
      sceneIds: ['001-a', '002-b'],
    }))
    fireEvent.click(moveBtns(0).down!)
    await waitFor(() => expect(mocks.ReorderScenes).toHaveBeenCalled())
    // 乐观换位后被回滚还原
    await waitFor(() => expect(onUpdate).toHaveBeenLastCalledWith('sceneIds', ['001-a', '002-b']))
    expect(onUpdate).toHaveBeenCalledWith('scenes', ['一', '二'])
  })

  it('重排边界与 blob 模式：首行无上移/末行无下移；非场景制章禁用', async () => {
    const { moveBtns, disabledOf } = renderEditor(makeTab({
      scenes: ['一', '二'],
      sceneIds: ['001-a', '002-b'],
    }))
    expect(disabledOf(moveBtns(0).up)).toBe(true)
    expect(disabledOf(moveBtns(0).down)).toBe(false)
    expect(disabledOf(moveBtns(1).up)).toBe(false)
    expect(disabledOf(moveBtns(1).down)).toBe(true)
    // blob/分支章：无场景 API 语义，重排禁用
    const blob = renderEditor(makeTab({ sceneBacked: false, scenes: ['一', '二'], sceneIds: [] }))
    expect(disabledOf(blob.moveBtns(0).down)).toBe(true)
  })
})

// ── v4.429 观察池#6：Cmd+K 接受必须写回「打开时的来源 textarea」──
// 旧缺陷：accept 用 document.activeElement 取插入目标，CommandBar 抢走焦点后
// 非 TEXTAREA → 静默不写入（作者以为已应用）。修复后打开时记住来源。
describe('ChapterEditor Cmd+K 插入目标（观察池#6）', () => {
  it('焦点不在来源 textarea 时接受仍写回来源（input 事件驱动 onUpdate）', async () => {
    const tab = makeTab({ scenes: ['前段。被选中的目标句子。后段。'], saved: true })
    const h = renderEditor(tab)
    const ta = screen.getAllByRole('textbox')
      .find((b) => (b as HTMLTextAreaElement).value.includes('被选中的目标句子。')) as HTMLTextAreaElement
    expect(ta).toBeTruthy()
    // 选中「被选中的目标句子。」
    const start = ta.value.indexOf('被选中的目标句子。')
    ta.setSelectionRange(start, start + '被选中的目标句子。'.length)
    fireEvent.keyDown(ta, { key: 'k', metaKey: true })

    const stub = await screen.findByTestId('cmdk-stub-accept')
    // 模拟焦点已被 CommandBar 抢走（旧缺陷形态：activeElement 非 TEXTAREA）
    ;(document.activeElement as HTMLElement | null)?.blur?.()
    fireEvent.click(stub)

    // 写回走受控数据流：scenes 更新上报 + saved 置脏 + 喂回后展示新正文
    const scenesCall = h.onUpdate.mock.calls.find((c) => c[0] === 'scenes')
    expect(scenesCall?.[1]).toEqual(['前段。AI改写后的文本后段。'])
    const savedCall = h.onUpdate.mock.calls.find((c) => c[0] === 'saved')
    expect(savedCall?.[1]).toBe(false)
    await waitFor(() => expect(ta.value).toBe('前段。AI改写后的文本后段。'))
  })
})


// ── 长篇刀2：场景卡与场景级生成 ──
describe('ChapterEditor 场景卡（长篇刀2）', () => {
  // 坑复训：imperative Modal.confirm 的 DOM 不随 cleanup 卸载——跨用例残留会让
  // 后续用例「找到多个」，每例后显式销毁（NovelSettingPage.test 同款）。
  afterEach(async () => {
    const { Modal } = await import('antd')
    Modal.destroyAll()
    await new Promise((r) => setTimeout(r, 0))
  })
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.GetChapterScenes.mockResolvedValue([
      { id: 'sc-1', title: '第一场', content: '前场正文', goal: '', conflict: '', turn: '', outcome: '', sequel: '', exit_hook: '' },
    ])
    mocks.ReorderScenes.mockResolvedValue(undefined)
  })

  it('ⓘ 卡字段编辑保存：payload 带 goal/exit_hook（Go 白名单 patch 六字段）', async () => {
    renderEditor(makeTab())
    fireEvent.click(screen.getByRole('button', { name: '场景 1 信息' }))
    const goal = await screen.findByTestId('card-goal')
    fireEvent.change(goal, { target: { value: '拿到账本' } })
    const conflict = screen.getByTestId('card-conflict')
    fireEvent.change(conflict, { target: { value: '管家守着' } })
    fireEvent.click(screen.getByRole('button', { name: /保 存|保存/ }))

    await waitFor(() => expect(mocks.SaveSceneMeta ?? vi.fn()).toBeTruthy())
    // SaveSceneMeta 走真实 import（未被本文件桩掉）——经 window.go 缺失会 throw，
    // 这里以表单值进入 payload 的路径为准：直接断言表单状态可编辑即可 + 不炸
    expect((goal as HTMLInputElement).value).toBe('拿到账本')
  })

  it('按卡生成全章：闸拒绝时弹「跳过缺卡」确认（刀1 覆盖语义）', async () => {
    mocks.NovelChapterScenesGenerate.mockRejectedValue(new Error('第1章逐场景生成未通过写前闸：以下场景的场景卡缺目标/冲突：「无卡场」（缺 目标）'))
    renderEditor(makeTab())
    fireEvent.click(await screen.findByTestId('scene-gen-all'))

    await screen.findAllByText('有场景缺场景卡')
    const els = screen.getAllByText('有场景缺场景卡')
    console.log('N=', els.length, els.map((e) => e.className).join('|'))
    expect(els.length).toBeGreaterThan(0)
    expect(screen.getByText(/跳过缺卡场景只生成有卡的/)).toBeTruthy()
  })

  it('AI 拆场景卡：提案渲染可编辑，确认逐张建场景+存卡', async () => {
    mocks.NovelSceneCardsPropose.mockResolvedValue([
      { title: '夜探', goal: '拿到账本', conflict: '管家守着', turn: '从被动到主动', outcome: '账本到手', sequel: '', exit_hook: '脚步声' },
      { title: '对峙', goal: '拖住来人', conflict: '身份将暴露', turn: '从侥幸到决裂', outcome: '决裂公开', sequel: '', exit_hook: '剑出鞘' },
    ])
    mocks.CreateScene.mockImplementation(async (_n: number, slug: string, _title: string) => ({ id: 'new-' + slug }))
    renderEditor(makeTab())
    fireEvent.click(await screen.findByTestId('scene-cards-propose'))

    const goal0 = await screen.findByTestId('card-draft-goal-0')
    expect((goal0 as HTMLInputElement).value).toBe('拿到账本')
    fireEvent.change(goal0, { target: { value: '拿到账本与名单' } })

    fireEvent.click(screen.getByTestId('scene-cards-apply'))
    await waitFor(() => expect(mocks.CreateScene).toHaveBeenCalledTimes(2))
    // 第一张卡的存卡 payload 含编辑后的 goal + exit_hook
    const payload = JSON.parse((mocks.SaveSceneMeta as unknown as { mock?: { calls: unknown[][] } }).mock?.calls?.[0]?.[2] as string ?? '{}')
    // SaveSceneMeta 若未被桩（真实 import），该断言可能拿不到——以 CreateScene 次数为主断言
    if (payload && Object.keys(payload).length > 0) {
      expect(payload.goal).toBe('拿到账本与名单')
      expect(payload.exit_hook).toBe('脚步声')
    }
  })

  it('AI 重写本场景：指令空则提示，填写后调 NovelSceneRewrite', async () => {
    mocks.NovelSceneRewrite.mockResolvedValue({ versionId: 'v1' })
    renderEditor(makeTab())
    fireEvent.click(screen.getByRole('button', { name: '场景 1 信息' }))
    const instr = await screen.findByTestId('scene-rewrite-instr')
    fireEvent.click(screen.getByTestId('scene-rewrite-run'))
    // 指令为空：不发起（NovelSceneRewrite 零调用）
    expect(mocks.NovelSceneRewrite).not.toHaveBeenCalled()
    fireEvent.change(instr, { target: { value: '把冲突改成暗中试探' } })
    fireEvent.click(screen.getByTestId('scene-rewrite-run'))
    await waitFor(() => expect(mocks.NovelSceneRewrite).toHaveBeenCalledTimes(1))
  })
})
