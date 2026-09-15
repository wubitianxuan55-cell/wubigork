// 提示词工坊面板（t6 首刀：模板可编辑覆盖层）：列表分组徽标/详情基线对照/
// 保存闭环（参数+Issues+message）/恢复内置二次确认/预览变量与 warnings/空态。
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import type { Mock } from 'vitest'

// 五绑定 mock 以 vi.hoisted 引用持有（vi.mock 工厂被提升到文件顶部；
// 断言不经过 app.* 类型面——AppBindings 尚未含这五名，tsc 待主代理 regen
// 后转绿，CreatePage.test.tsx 同款范式）。
const mocks = vi.hoisted(() => ({
  list: vi.fn().mockResolvedValue([]),
  get: vi.fn().mockResolvedValue({}),
  save: vi.fn().mockResolvedValue({ saved: true, issues: [], version: 1 }),
  reset: vi.fn().mockResolvedValue(undefined),
  preview: vi.fn().mockResolvedValue({ systemPrompt: '', warnings: [] }),
}))

vi.mock('../../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../gaea/lib/bridge')>()
  return {
    ...actual,
    app: {
      PromptTemplateList: mocks.list,
      PromptTemplateGet: mocks.get,
      PromptTemplateSave: mocks.save,
      PromptTemplateReset: mocks.reset,
      PromptTemplatePreview: mocks.preview,
    },
  }
})

import PromptWorkshopPanel from './PromptWorkshopPanel'
import type { PromptTemplateDetail, PromptTemplateMeta } from './api/prompt'

const mkMeta = (key: string, category: string, source: string, hasOverride: boolean, overrideActive: boolean, version: number): PromptTemplateMeta => ({
  key, category, description: `${key} 说明`, source, hasOverride, overrideActive, version,
  updatedAt: hasOverride ? 1758000000000 : 0,
})

const mkDetail = (key: string, source = 'override'): PromptTemplateDetail => ({
  meta: mkMeta(key, 'chapter', source, source === 'override', true, 3),
  template: {
    name: key,
    system: '覆盖版 System：请围绕 {{plot}} 创作第 {{chapter_num}} 章，约 {{word_count}} 字。',
    task: '覆盖版 Task：结合大纲推进情节。',
    output: { description: '完整章节正文' },
    parameters: ['plot', 'chapter_num', 'word_count'],
    category: 'chapter',
    description: '整章创作主模板（示例覆盖）',
  },
  base: {
    name: key,
    system: '内置 System：依据大纲与上一章摘要写作本章。',
    task: '内置 Task：推进本章情节。',
    output: { description: '完整章节正文' },
  },
})

const openPanel = () => render(<PromptWorkshopPanel open onClose={vi.fn()} />)

describe('PromptWorkshopPanel 提示词工坊（t6 首刀）', () => {
  const onClose = vi.fn()

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('列表分组与徽标：override=自定义(orange)/builtin=内置；停用覆盖加灰 Tag + 版本号', async () => {
    mocks.list.mockResolvedValue([
      mkMeta('create-chapter', 'chapter', 'override', true, true, 3),
      mkMeta('chapter-summary', 'summary', 'builtin', false, false, 0),
      mkMeta('rewrite-chapter', 'rewrite', 'builtin', true, false, 2),
    ] as never)
    openPanel()
    await waitFor(() => expect(screen.getAllByTestId('prompt-workshop-row')).toHaveLength(3))
    // 分组标题中文映射
    expect(screen.getByText('章节创作')).toBeTruthy()
    expect(screen.getByText('摘要')).toBeTruthy()
    expect(screen.getByText('重写')).toBeTruthy()
    // 来源徽标
    expect(screen.getByText('自定义')).toBeTruthy()
    expect(screen.getAllByText('内置').length).toBeGreaterThanOrEqual(2)
    // HasOverride 且未激活 → 灰 Tag「已停用」；行尾版本号
    expect(screen.getByText('已停用')).toBeTruthy()
    expect(screen.getByText('v3')).toBeTruthy()
    // 打开只拉一次（零轮询）
    expect(mocks.list).toHaveBeenCalledTimes(1)
  })

  it('选中加载详情与基线对照：表单填充生效模板，Collapse 展开内置 Base 只读对照', async () => {
    mocks.list.mockResolvedValue([
      mkMeta('create-chapter', 'chapter', 'override', true, true, 3),
    ] as never)
    mocks.get.mockResolvedValue(mkDetail('create-chapter') as never)
    openPanel()
    await waitFor(() => expect(screen.getAllByTestId('prompt-workshop-row')).toHaveLength(1))
    fireEvent.click(screen.getByText('create-chapter'))
    await waitFor(() => expect(screen.getByTestId('prompt-workshop-detail')).toBeTruthy())
    expect(mocks.get).toHaveBeenCalledWith('create-chapter')
    // 表单填充生效模板（覆盖版），非基线
    const sys = screen.getByTestId('prompt-workshop-system') as HTMLTextAreaElement
    expect(sys.value).toContain('覆盖版 System')
    expect((screen.getByTestId('prompt-workshop-task') as HTMLTextAreaElement).value).toContain('覆盖版 Task')
    expect((screen.getByTestId('prompt-workshop-category') as HTMLInputElement).value).toBe('chapter')
    // 内置基线对照：展开后可见 Base.system（与生效模板不同）
    expect(screen.queryByTestId('prompt-workshop-base')).toBeNull()
    fireEvent.click(screen.getByText('内置基线（只读对照，恢复内置将回到这里）'))
    await waitFor(() => expect(screen.getByTestId('prompt-workshop-base')).toBeTruthy())
    expect(screen.getByTestId('prompt-workshop-base').textContent).toContain('内置 System：依据大纲与上一章摘要写作本章。')
  })

  it('保存闭环：saveTemplate 参数序列化 + warn Issues 渲染 + 成功 message + 刷新列表与详情', async () => {
    mocks.list.mockResolvedValue([
      mkMeta('create-chapter', 'chapter', 'override', true, true, 3),
    ] as never)
    mocks.get.mockResolvedValue(mkDetail('create-chapter') as never)
    mocks.save.mockResolvedValue({
      saved: true,
      issues: [{ code: 'legacy-brace', severity: 'warn', message: '存在旧语法 {word_count} 占位符，建议改为 {{word_count}}' }],
      version: 4,
    } as never)
    openPanel()
    await waitFor(() => expect(screen.getAllByTestId('prompt-workshop-row')).toHaveLength(1))
    fireEvent.click(screen.getByText('create-chapter'))
    await waitFor(() => expect(screen.getByTestId('prompt-workshop-system')).toBeTruthy())
    fireEvent.change(screen.getByTestId('prompt-workshop-system'), { target: { value: '改后的 System：围绕 {{plot}} 写作。' } })
    fireEvent.click(screen.getByTestId('prompt-workshop-save'))
    await waitFor(() => expect(mocks.save).toHaveBeenCalledTimes(1))
    const [key, reqJSON] = (mocks.save as Mock).mock.calls[0]
    expect(key).toBe('create-chapter')
    const req = JSON.parse(reqJSON as string)
    expect(req.isActive).toBe(true)
    expect(req.category).toBe('chapter')
    expect(req.content.system).toBe('改后的 System：围绕 {{plot}} 写作。')
    expect(req.content.task).toContain('覆盖版 Task')
    expect(req.content.parameters).toEqual(['plot', 'chapter_num', 'word_count'])
    // Issues 逐条渲染（warn 橙档）
    await waitFor(() => expect(screen.getByTestId('prompt-workshop-issues')).toBeTruthy())
    expect(screen.getByText(/存在旧语法/)).toBeTruthy()
    expect(screen.getByText('legacy-brace（提示）')).toBeTruthy()
    // 成功 message + 刷新列表与详情
    await waitFor(() => expect(screen.getByText('已保存覆盖（生成链路即时生效）')).toBeTruthy())
    await waitFor(() => expect(mocks.list).toHaveBeenCalledTimes(2))
    await waitFor(() => expect(mocks.get).toHaveBeenCalledTimes(2))
  })

  it('恢复内置：Popconfirm 二次确认后调 resetTemplate 并重拉列表与详情', async () => {
    mocks.list.mockResolvedValue([
      mkMeta('create-chapter', 'chapter', 'override', true, true, 3),
    ] as never)
    mocks.get.mockResolvedValue(mkDetail('create-chapter') as never)
    openPanel()
    await waitFor(() => expect(screen.getAllByTestId('prompt-workshop-row')).toHaveLength(1))
    fireEvent.click(screen.getByText('create-chapter'))
    await waitFor(() => expect(screen.getByTestId('prompt-workshop-reset')).toBeTruthy())
    // 未确认前不触发
    fireEvent.click(screen.getByTestId('prompt-workshop-reset'))
    expect(mocks.reset).not.toHaveBeenCalled()
    const okBtn = await screen.findByRole('button', { name: '确认恢复' }, { timeout: 4000 })
    fireEvent.click(okBtn)
    await waitFor(() => expect(mocks.reset).toHaveBeenCalledWith('create-chapter'))
    await waitFor(() => expect(screen.getByText('已恢复内置模板')).toBeTruthy())
    await waitFor(() => expect(mocks.list).toHaveBeenCalledTimes(2))
    await waitFor(() => expect(mocks.get).toHaveBeenCalledTimes(2))
  })

  it('预览：按 parameters 生成变量输入列，渲染结果与未解析变量 warnings Tag', async () => {
    mocks.list.mockResolvedValue([
      mkMeta('create-chapter', 'chapter', 'override', true, true, 3),
    ] as never)
    mocks.get.mockResolvedValue(mkDetail('create-chapter') as never)
    mocks.preview.mockResolvedValue({
      systemPrompt: '请围绕 主角觉醒 创作第 {{chapter_num}} 章，约 3000 字。',
      warnings: ['chapter_num'],
    } as never)
    openPanel()
    await waitFor(() => expect(screen.getAllByTestId('prompt-workshop-row')).toHaveLength(1))
    fireEvent.click(screen.getByText('create-chapter'))
    await waitFor(() => expect(screen.getByTestId('prompt-workshop-var-plot')).toBeTruthy())
    // 按 parameters 动态生成三个变量输入
    expect(screen.getByTestId('prompt-workshop-var-chapter_num')).toBeTruthy()
    expect(screen.getByTestId('prompt-workshop-var-word_count')).toBeTruthy()
    fireEvent.change(screen.getByTestId('prompt-workshop-var-plot'), { target: { value: '主角觉醒' } })
    fireEvent.change(screen.getByTestId('prompt-workshop-var-word_count'), { target: { value: '3000' } })
    fireEvent.click(screen.getByTestId('prompt-workshop-preview-btn'))
    await waitFor(() => expect(mocks.preview).toHaveBeenCalledTimes(1))
    const [reqJSON, varsJSON] = (mocks.preview as Mock).mock.calls[0]
    expect(JSON.parse(reqJSON as string).content.system).toContain('覆盖版 System')
    expect(JSON.parse(varsJSON as string)).toEqual({ plot: '主角觉醒', word_count: '3000' })
    // 渲染结果只读展示 + warnings Tag
    await waitFor(() => expect(screen.getByTestId('prompt-workshop-preview-out').textContent).toContain('主角觉醒'))
    expect(screen.getByText('未解析变量：')).toBeTruthy()
    expect(screen.getByText('chapter_num')).toBeTruthy()
  })

  it('空态：无模板时给引导文案', async () => {
    mocks.list.mockResolvedValue([] as never)
    render(<PromptWorkshopPanel open onClose={onClose} />)
    await waitFor(() => expect(screen.getByText(/还没有可编辑的提示词模板/)).toBeTruthy())
  })
})
