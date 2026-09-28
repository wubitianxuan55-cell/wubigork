// ChapterPlanCard.test.tsx — 章节计划卡（刀1 线D）契约与硬闸提示回归。
//
// 覆盖：空态 / 未制定计划硬闸横幅与缺失清单 / 草案进入编辑态且不落盘 / 保存失败如实
// 报错且表单不丢 / 保存成功回调与预检复拉 / 偏差报告（未达成事件 + 下一章建议不落盘）/
// 切章 seq 守卫（迟到的旧章响应不得落到新章）。
//
// 坑复训：vi.clearAllMocks() **不清 mockImplementation** → 本文件用 resetAllMocks +
// 显式铺默认实现；在途态用 mockImplementationOnce / 手写 deferred。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within, act } from '@testing-library/react'

// 计划绑定面（线C 实现）：本文件按 app 代理的调用语义 mock —— 组件经
// `app as unknown as ...` 路由，mock 对象即绑定面。
const mocks = vi.hoisted(() => ({
  NovelChapterPlanGet: vi.fn(),
  NovelChapterPlanSave: vi.fn(),
  NovelChapterPlanPropose: vi.fn(),
  NovelChapterPlanDeviation: vi.fn(),
  NovelChapterGatePrecheck: vi.fn(),
}))
vi.mock('../../gaea/lib/bridge', () => ({ app: mocks }))

import ChapterPlanCard from './ChapterPlanCard'

/** Go types.ChapterPlan 七字段（json tag 逐字对齐 plot_v2.go:11-22）。 */
const FULL_PLAN = {
  sub_index: 4,
  title: '第5章',
  plot_summary: '雨夜夺符，主角暴露身份',
  key_events: ['夺符', '遁走'],
  character_focus: ['林砚'],
  emotional_tone: '紧张',
  narrative_goal: '逼主角暴露身份',
  conflict_type: '人vs人',
  ending_type: '悬念',
  estimated_words: 3000,
}

const OK_GATE = {
  chapterNum: 5, allowed: true, hasPlan: true, missing: [],
  planProblems: [], outlineIssues: [], blocking: false,
}

const NO_PLAN_GATE = {
  chapterNum: 5, allowed: false, hasPlan: false, missing: ['叙事目标', '冲突类型'],
  planProblems: [{ code: 'plan_goal_empty', severity: 'S1', message: '叙事目标为空' }],
  outlineIssues: [], blocking: true,
}

const PLAN_BUT_BLOCKED_GATE = {
  chapterNum: 5, allowed: false, hasPlan: true, missing: [],
  planProblems: [{ code: 'plan_event_duplicated', severity: 'S2', message: '关键事件「夺符」与第2章重复' }],
  outlineIssues: [], blocking: true,
}

beforeEach(() => {
  vi.resetAllMocks() // 清实现：clearAllMocks 只清调用记录，会漏掉上个用例的 mockImplementation
  mocks.NovelChapterPlanGet.mockResolvedValue(null)
  mocks.NovelChapterPlanSave.mockResolvedValue(undefined)
  mocks.NovelChapterPlanPropose.mockResolvedValue(null)
  mocks.NovelChapterPlanDeviation.mockResolvedValue(null)
  mocks.NovelChapterGatePrecheck.mockResolvedValue(NO_PLAN_GATE)
})

describe('ChapterPlanCard 计划卡与硬闸（刀1 线D）', () => {
  it('未选章：空态「先在左侧选择章节」，不发任何计划请求', () => {
    render(<ChapterPlanCard chapterNum={null} />)
    expect(screen.getByText('先在左侧选择章节')).toBeTruthy()
    expect(mocks.NovelChapterPlanGet).not.toHaveBeenCalled()
    expect(mocks.NovelChapterGatePrecheck).not.toHaveBeenCalled()
  })

  it('未制定计划：blocking 时渲染红色硬闸横幅 + 缺失中文清单 + 「先补章节计划」动作与两个补计划入口', async () => {
    const onNeedPlan = vi.fn()
    mocks.NovelChapterPlanGet.mockResolvedValue(null)
    mocks.NovelChapterGatePrecheck.mockResolvedValue(NO_PLAN_GATE)
    render(<ChapterPlanCard chapterNum={5} onNeedPlan={onNeedPlan} />)

    const banner = await screen.findByTestId('plan-gate-banner')
    expect(banner.className).toContain('ant-alert-error') // blocking → 红色
    expect(within(banner).getByText(/本章尚未制定计划（硬闸）/)).toBeTruthy()
    expect(within(banner).getByText(/缺失：叙事目标、冲突类型/)).toBeTruthy()
    expect(within(banner).getByText(/\[S1\] 叙事目标为空/)).toBeTruthy()
    fireEvent.click(within(banner).getByRole('button', { name: '先补章节计划' }))
    expect(onNeedPlan).toHaveBeenCalledTimes(1)

    // 未制定计划的两个补计划入口
    expect(screen.getByRole('button', { name: '生成计划草案' })).toBeTruthy()
    expect(screen.getByRole('button', { name: '手写计划' })).toBeTruthy()
    expect(screen.queryByRole('button', { name: '编辑计划' })).toBeNull()
  })

  it('生成草案：Propose 成功后进入编辑态（七字段入表单），且未调用 Save（AI 不落盘）', async () => {
    mocks.NovelChapterPlanGet.mockResolvedValue(null)
    mocks.NovelChapterPlanPropose.mockResolvedValue(FULL_PLAN)
    render(<ChapterPlanCard chapterNum={5} />)

    fireEvent.click(await screen.findByRole('button', { name: /生成计划草案/ }))

    const goal = await screen.findByTestId('plan-goal-input') as HTMLInputElement
    expect(goal.value).toBe('逼主角暴露身份')
    expect((screen.getByTestId('plan-key-events-input') as HTMLTextAreaElement).value).toBe('夺符\n遁走')
    expect((screen.getByTestId('plan-focus-input') as HTMLTextAreaElement).value).toBe('林砚')
    expect((screen.getByTestId('plan-emotion-input') as HTMLInputElement).value).toBe('紧张')
    expect((screen.getByTestId('plan-conflict-input') as HTMLInputElement).value).toBe('人vs人')
    expect(screen.getByText(/未落盘/)).toBeTruthy()
    expect(screen.getByRole('button', { name: /保存计划/ })).toBeTruthy()
    expect(mocks.NovelChapterPlanPropose).toHaveBeenCalledWith(5)
    expect(mocks.NovelChapterPlanSave).not.toHaveBeenCalled() // 草案不落盘
  })

  it('保存失败：如实显示后端错误（含被点名的重复事件）且表单不丢', async () => {
    mocks.NovelChapterPlanGet.mockResolvedValue(null)
    mocks.NovelChapterPlanSave.mockRejectedValue(new Error('第 5 章关键事件「夺符」与第 2 章重复'))
    render(<ChapterPlanCard chapterNum={5} />)

    fireEvent.click(await screen.findByRole('button', { name: '手写计划' }))
    fireEvent.change(screen.getByTestId('plan-goal-input'), { target: { value: '逼主角暴露身份' } })
    fireEvent.change(screen.getByTestId('plan-key-events-input'), { target: { value: '夺符' } })
    fireEvent.click(screen.getByRole('button', { name: /保存计划/ }))

    const err = await screen.findByTestId('plan-save-error')
    expect(err.textContent).toContain('第 5 章关键事件「夺符」与第 2 章重复')
    // 表单保留：编辑态与草稿原样在位，作者不必重填
    expect((screen.getByTestId('plan-key-events-input') as HTMLTextAreaElement).value).toBe('夺符')
    expect((screen.getByTestId('plan-goal-input') as HTMLInputElement).value).toBe('逼主角暴露身份')
    // 按钮名用正则：antd 两字 CJK 标签会插空格，且 loading 图标在 jsdom 里不随离场动画移除，
    // 会把 "loading" 并进可访问名（精确匹配假红）
    expect(screen.getByRole('button', { name: /保存计划/ })).toBeTruthy()
    // 载荷键名按 Go json tag（snake_case），不含 camelCase 别名
    const sent = JSON.parse(mocks.NovelChapterPlanSave.mock.calls[0][1] as string) as Record<string, unknown>
    expect(mocks.NovelChapterPlanSave.mock.calls[0][0]).toBe(5)
    expect(sent.key_events).toEqual(['夺符'])
    expect(sent.narrative_goal).toBe('逼主角暴露身份')
    expect(sent).not.toHaveProperty('narrativeGoal')
  })

  it('保存成功：Save 收到契约载荷（保留未编辑的 sub_index/title）、回调触发、预检复拉后横幅消失', async () => {
    mocks.NovelChapterPlanGet.mockResolvedValue(FULL_PLAN)
    mocks.NovelChapterPlanSave.mockResolvedValue(undefined)
    // 打开时阻断（计划存在但关键事件跨章重复），保存后复拉转为放行
    mocks.NovelChapterGatePrecheck
      .mockResolvedValueOnce(PLAN_BUT_BLOCKED_GATE)
      .mockResolvedValue(OK_GATE)
    const onPlanSaved = vi.fn()
    render(<ChapterPlanCard chapterNum={5} onPlanSaved={onPlanSaved} />)

    const banner = await screen.findByTestId('plan-gate-banner')
    expect(within(banner).getByText(/章节计划未过预检（会阻断生成）/)).toBeTruthy()

    fireEvent.click(await screen.findByRole('button', { name: '编辑计划' }))
    fireEvent.change(screen.getByTestId('plan-goal-input'), { target: { value: '改过的叙事目标' } })
    fireEvent.click(screen.getByRole('button', { name: /保存计划/ }))

    await waitFor(() => expect(onPlanSaved).toHaveBeenCalledTimes(1))
    const [num, payload] = mocks.NovelChapterPlanSave.mock.calls[0] as [number, string]
    expect(num).toBe(5)
    const sent = JSON.parse(payload) as Record<string, unknown>
    expect(sent.narrative_goal).toBe('改过的叙事目标')
    expect(sent.sub_index).toBe(4) // 不编辑的字段原样保留，避免落盘把大纲序清零
    expect(sent.title).toBe('第5章')
    expect(sent.ending_type).toBe('悬念')
    // 保存成功后复拉预检：硬闸横幅随审批结果消失（不乐观假设）
    await waitFor(() => expect(mocks.NovelChapterGatePrecheck).toHaveBeenCalledTimes(2))
    await waitFor(() => expect(screen.queryByTestId('plan-gate-banner')).toBeNull())
  })

  it('偏差报告：概要 + 未达成事件 + 结尾/情绪不符标记 + 下一章建议只展示（无落盘入口）', async () => {
    mocks.NovelChapterPlanGet.mockResolvedValue(FULL_PLAN)
    mocks.NovelChapterGatePrecheck.mockResolvedValue(OK_GATE)
    mocks.NovelChapterPlanDeviation.mockResolvedValue({
      chapterNum: 5, hasPlan: true, analyzed: true,
      missingEvents: ['夺符', '遁走'],
      endingMismatch: true, plannedEnding: '悬念', actualEnding: '情感收尾',
      emotionDrift: true, plannedEmotion: '紧张', actualEmotion: '怅然',
      duplicateEvents: ['夺符'],
      summary: '计划 2 条关键事件，实际命中 0 条',
      nextSuggestion: '下一章先回收符箓线索，再让林砚与师父摊牌',
    })
    render(<ChapterPlanCard chapterNum={5} />)

    const dev = await screen.findByTestId('plan-deviation')
    expect(within(dev).getByText(/计划 2 条关键事件，实际命中 0 条/)).toBeTruthy()
    const missing = within(dev).getByTestId('plan-deviation-missing')
    expect(within(missing).getByText('夺符')).toBeTruthy()
    expect(within(missing).getByText('遁走')).toBeTruthy()
    expect(within(dev).getByText(/结尾类型不符：计划「悬念」→ 实际「情感收尾」/)).toBeTruthy()
    expect(within(dev).getByText(/情绪走向漂移：计划「紧张」→ 实际「怅然」/)).toBeTruthy()
    expect(within(dev).getByText(/与其它章重复 1 条：夺符/)).toBeTruthy()
    const suggestion = within(dev).getByTestId('plan-next-suggestion')
    expect(suggestion.textContent).toContain('下一章先回收符箓线索，再让林砚与师父摊牌')
    expect(suggestion.textContent).toContain('仅建议，不落盘')
    // 建议不提供一键落盘
    expect(within(suggestion).queryByRole('button')).toBeNull()
    expect(within(dev).queryByRole('button', { name: /采纳|应用建议|落盘|保存建议/ })).toBeNull()
  })

  it('未分析 / 无计划的偏差报告不渲染（缺失是正常态）', async () => {
    mocks.NovelChapterPlanGet.mockResolvedValue(FULL_PLAN)
    mocks.NovelChapterGatePrecheck.mockResolvedValue(OK_GATE)
    mocks.NovelChapterPlanDeviation.mockResolvedValue({ chapterNum: 5, hasPlan: false, analyzed: false, summary: '未分析' })
    render(<ChapterPlanCard chapterNum={5} />)
    await screen.findByText(/逼主角暴露身份/)
    expect(screen.queryByTestId('plan-deviation')).toBeNull()
  })

  it('切章 seq 守卫：先选 5 再选 9，5 的迟到响应被丢弃（不覆盖新章卡片）', async () => {
    let resolveLate5: (value: unknown) => void = () => {}
    const late5 = new Promise<unknown>((resolve) => { resolveLate5 = resolve })
    const planB = { ...FULL_PLAN, narrative_goal: 'B章目标', plot_summary: 'B章摘要', key_events: ['B事件'] }
    mocks.NovelChapterPlanGet.mockImplementation((chapterNum: unknown) => (
      chapterNum === 5 ? late5 : Promise.resolve(planB)
    ))
    mocks.NovelChapterGatePrecheck.mockResolvedValue(OK_GATE)

    const { rerender } = render(<ChapterPlanCard chapterNum={5} />)
    rerender(<ChapterPlanCard chapterNum={9} />)
    expect(await screen.findByText(/B章目标/)).toBeTruthy()

    // 5 的响应此刻才到：seq 已被 9 取代 → 必须被丢弃
    await act(async () => {
      resolveLate5({ ...FULL_PLAN, narrative_goal: 'A章目标', plot_summary: 'A章摘要' })
      await Promise.resolve()
    })
    expect(screen.queryByText(/A章目标/)).toBeNull()
    expect(screen.queryByText(/A章摘要/)).toBeNull()
    expect(screen.getByText(/B章目标/)).toBeTruthy()
    expect(screen.getByText(/第 9 章/)).toBeTruthy()
  })
})
