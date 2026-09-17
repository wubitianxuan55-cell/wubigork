// 场景圣经「视角」抽屉（刀7续 POV 视图）：已知/不知情对照为核心区/角色卡/
// 空区段如实空/错误态重试。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

const mocks = vi.hoisted(() => ({
  view: vi.fn(),
}))

vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return {
    ...actual,
    app: {
      NovelSceneBibleView: mocks.view,
    },
  }
})

import SceneBibleDrawer from './SceneBibleDrawer'

const VIEW = {
  pov: 'char-linzhao',
  title: '雨夜叩门',
  location: '旧宅门廊',
  timeOfDay: '深夜',
  mood: '压抑中带试探',
  tags: ['dialogue'],
  characters: [
    { name: '林昭', roleType: '主角', status: 'Alive', currentState: '由旁观转入局（第 14 章）', careerMain: '剑修·3阶' },
    { name: '守门人', roleType: '配角', status: 'Alive', knownBy: ['林昭'] },
  ],
  povView: '林昭知道：信物纹路与门楣浮雕一致。',
  hiddenFacts: ['守门人早已认出林昭', '信物另有一枚，在反派手中'],
  foreshadows: ['褪色的信物（预计第 26 章回收）'],
  memories: [],
  timeAnchor: '上一场景后约半个时辰',
  thread: '身世之谜逼近第一层揭开',
  style: '',
}

const open = (sceneID = 'sc-1') =>
  render(<SceneBibleDrawer open chapterNum={14} sceneID={sceneID} sceneLabel="场景 1" onClose={vi.fn()} />)

describe('SceneBibleDrawer 场景视角（刀7续 POV 视图）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('打开即拉：POV 头 + 已知（绿）/不知情（红）对照 + 角色卡 + 伏笔', async () => {
    mocks.view.mockResolvedValue(VIEW as never)
    open()
    await waitFor(() => expect(mocks.view).toHaveBeenCalledWith(14, 'sc-1'))
    await waitFor(() => expect(screen.getByTestId('scene-bible-body')).toBeTruthy())
    expect(screen.getByText('通过 char-linzhao 的眼睛')).toBeTruthy()
    expect(screen.getByText(/雨夜叩门 · 旧宅门廊 · 深夜/)).toBeTruthy()
    // 对照区：已知 + 不知情两条
    expect(screen.getByText(/信物纹路与门楣浮雕一致/)).toBeTruthy()
    expect(screen.getByText(/守门人早已认出林昭/)).toBeTruthy()
    expect(screen.getByText(/信物另有一枚/)).toBeTruthy()
    expect(screen.getByText(/POV 不知情 · 本场景不得泄露（2）/)).toBeTruthy()
    // 角色卡与状态机回灌
    expect(screen.getByText('林昭')).toBeTruthy()
    expect(screen.getByText(/由旁观转入局（第 14 章）/)).toBeTruthy()
    // 伏笔约束
    expect(screen.getByText(/褪色的信物/)).toBeTruthy()
  })

  it('空区段如实空展示（无 POV=全知；无隐藏约束）', async () => {
    mocks.view.mockResolvedValue({
      ...VIEW, pov: '', povView: '', hiddenFacts: [], characters: [], foreshadows: [],
    } as never)
    open('')
    await waitFor(() => expect(screen.getByTestId('scene-bible-body')).toBeTruthy())
    expect(screen.getByText('未设定 POV（全知视角）')).toBeTruthy()
    expect(screen.getByText(/没有对该 POV 瞒住的事实/)).toBeTruthy()
  })

  it('错误态展示并可重试', async () => {
    mocks.view.mockRejectedValueOnce(new Error('场景不存在: sc-x') as never)
      .mockResolvedValueOnce(VIEW as never)
    open('sc-x')
    await waitFor(() => expect(screen.getByText(/场景不存在/)).toBeTruthy())
    fireEvent.click(screen.getByText('重试'))
    await waitFor(() => expect(screen.getByTestId('scene-bible-body')).toBeTruthy())
    expect(mocks.view).toHaveBeenCalledTimes(2)
  })
})
