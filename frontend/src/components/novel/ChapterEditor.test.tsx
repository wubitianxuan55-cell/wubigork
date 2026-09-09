import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import React from 'react'

// NovelB 门面具名导入 mock（与 CreatePage.test 同风格：只 mock 用到的三件）
const mocks = vi.hoisted(() => ({
  GetChapterScenes: vi.fn(),
  GenerateScene: vi.fn(),
  CreateScene: vi.fn(),
  ReorderScenes: vi.fn().mockResolvedValue(undefined),
}))
vi.mock('../../../wailsjs/go/app/NovelB', () => ({
  GetChapterScenes: mocks.GetChapterScenes,
  GenerateScene: mocks.GenerateScene,
  CreateScene: mocks.CreateScene,
  ReorderScenes: mocks.ReorderScenes,
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
