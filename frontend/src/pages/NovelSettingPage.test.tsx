import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

// 屏蔽 Wails 绑定：jsdom 中没有 window.go
vi.mock('../../src/wailsjsCompat', () => ({
  GetWorldview: vi.fn().mockResolvedValue('# 世界观\n\n架空中世纪'),
  SaveWorldview: vi.fn().mockResolvedValue(undefined),
  ChatWorldview: vi.fn().mockResolvedValue({ reply: 'ok', worldview: '# 新设定' }),
  GetWorldviewSections: vi.fn().mockResolvedValue({
    sections: [
      { id: 'era', title: '时代背景', content: '架空中世纪', order: 1 },
      { id: 'geography', title: '地理风貌', content: '', order: 2 },
      { id: 'factions', title: '势力格局', content: '', order: 3 },
      { id: 'rules', title: '规则体系', content: '', order: 4 },
      { id: 'culture', title: '文化习俗', content: '', order: 5 },
      { id: 'history', title: '历史事件', content: '', order: 6 },
    ],
  }),
  SaveAllWorldviewSections: vi.fn().mockResolvedValue(undefined),
  GetForeshadows: vi.fn().mockResolvedValue({ items: [] }),
  CheckConsistency: vi.fn().mockResolvedValue({ issues: [], total_issues: 0, summary: '✅ 未发现一致性问题' }),
}))

import NovelSettingPage from './NovelSettingPage'
import { useAppStore } from '../stores/appStore'
import {
  GetWorldview, SaveWorldview, ChatWorldview,
  GetWorldviewSections, SaveAllWorldviewSections, GetForeshadows, CheckConsistency,
} from '../../src/wailsjsCompat'

describe('NovelSettingPage 纯文本设定编辑', () => {
  beforeEach(() => {
    useAppStore.setState({ projectOpen: true, projectPath: 'C:/novel/test' })
    vi.clearAllMocks()
    vi.mocked(GetWorldview).mockResolvedValue('# 世界观\n\n架空中世纪')
    vi.mocked(GetForeshadows).mockResolvedValue({ items: [] })
    vi.mocked(CheckConsistency).mockResolvedValue({ issues: [], total_issues: 0, summary: '✅ 未发现一致性问题' })
  })

  it('加载已有设定并在编辑区显示', async () => {
    render(<NovelSettingPage />)
    const editor = (await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)) as HTMLTextAreaElement
    expect(editor.value).toContain('架空中世纪')
  })

  it('编辑后显示未保存状态，保存调用 SaveWorldview', async () => {
    render(<NovelSettingPage />)
    const editor = (await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)) as HTMLTextAreaElement
    fireEvent.change(editor, { target: { value: '蒸汽纪元' } })
    expect(screen.getByText('有未保存修改')).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: /保存/ }))
    expect(SaveWorldview).toHaveBeenCalledWith('蒸汽纪元')
  })

  it('切换到渲染模式直接渲染设定文本', async () => {
    render(<NovelSettingPage />)
    await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)
    fireEvent.click(screen.getByRole('radio', { name: /渲染/ }))
    expect(await screen.findByText('架空中世纪')).toBeTruthy()
  })

  it('AI 回复未自动解析时，可手动点击「应用到设定」覆盖编辑器', async () => {
    vi.mocked(ChatWorldview).mockResolvedValue({
      reply: '这是新的设定：\n```markdown\n# 新世界观\n\n末日废土，蒸汽朋克\n```',
      worldview: '',
    })
    render(<NovelSettingPage />)
    const editor = (await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)) as HTMLTextAreaElement
    expect(editor.value).toContain('架空中世纪')

    // 发送一条对话，AI 返回未带 worldview 的回复
    const chatInput = screen.getByPlaceholderText(/描述你想要的设定修改/)
    fireEvent.change(chatInput, { target: { value: '重写设定' } })
    fireEvent.keyDown(chatInput, { key: 'Enter' })

    // 等待 AI 消息渲染完成（含逐字动画）
    expect(await screen.findByText(/末日废土/, {}, { timeout: 3000 })).toBeTruthy()

    // 手动点击「应用到设定」并确认覆盖
    fireEvent.click(screen.getByRole('button', { name: '应用到设定' }))
    fireEvent.click(await screen.findByRole('button', { name: '覆盖设定' }))

    expect(editor.value).toContain('# 新世界观')
    expect(editor.value).toContain('末日废土')
    expect(editor.value).not.toContain('```markdown')
    expect(screen.getByText('有未保存修改')).toBeTruthy()
  })
})

describe('NovelSettingPage 维度化编辑器（v4.3e）', () => {
  beforeEach(() => {
    useAppStore.setState({ projectOpen: true, projectPath: 'C:/novel/test' })
    vi.clearAllMocks()
    vi.mocked(GetWorldview).mockResolvedValue('# 世界观\n\n架空中世纪')
    vi.mocked(GetWorldviewSections).mockResolvedValue({
      sections: [
        { id: 'era', title: '时代背景', content: '架空中世纪', order: 1 },
        { id: 'geography', title: '地理风貌', content: '', order: 2 },
        { id: 'factions', title: '势力格局', content: '', order: 3 },
        { id: 'rules', title: '规则体系', content: '', order: 4 },
        { id: 'culture', title: '文化习俗', content: '', order: 5 },
        { id: 'history', title: '历史事件', content: '', order: 6 },
      ],
    })
    vi.mocked(GetForeshadows).mockResolvedValue({ items: [] })
    vi.mocked(CheckConsistency).mockResolvedValue({ issues: [], total_issues: 0, summary: '✅ 未发现一致性问题' })
  })

  it('切换到「维度化」显示 6 个维度卡片并可展开收起', async () => {
    render(<NovelSettingPage />)
    await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)
    fireEvent.click(screen.getByRole('radio', { name: /维度化/ }))

    // 六个维度标题
    for (const title of ['时代背景', '地理风貌', '势力格局', '规则体系', '文化习俗', '历史事件']) {
      expect(await screen.findByText(title)).toBeTruthy()
    }
    // 默认展开：时代背景编辑框可见
    const eraInput = screen.getByPlaceholderText(/撰写「时代背景」设定/) as HTMLTextAreaElement
    expect(eraInput.value).toBe('架空中世纪')
  })

  it('就地编辑维度并整存（SaveAllWorldviewSections）', async () => {
    render(<NovelSettingPage />)
    await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)
    fireEvent.click(screen.getByRole('radio', { name: /维度化/ }))

    const eraInput = await screen.findByPlaceholderText(/撰写「时代背景」设定/) as HTMLTextAreaElement
    fireEvent.change(eraInput, { target: { value: '蒸汽纪元，机械飞升' } })
    expect(screen.getByText('维度有未保存修改')).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: /保存全部维度/ }))
    await waitFor(() => expect(SaveAllWorldviewSections).toHaveBeenCalledTimes(1))
    const payload = JSON.parse(vi.mocked(SaveAllWorldviewSections).mock.calls[0][0] as string) as Array<{ id: string; content: string }>
    expect(payload).toHaveLength(6)
    expect(payload.find((s) => s.id === 'era')?.content).toBe('蒸汽纪元，机械飞升')
  })

  it('维度加载失败降级提示且不崩溃', async () => {
    vi.mocked(GetWorldviewSections).mockRejectedValue(new Error('项目数据损坏'))
    render(<NovelSettingPage />)
    await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)
    fireEvent.click(screen.getByRole('radio', { name: /维度化/ }))

    // 降级提示出现，页面仍可用（维度卡仍渲染、Segmented 仍在）
    expect(await screen.findByText(/维度数据加载失败/)).toBeTruthy()
    expect(screen.getByRole('button', { name: /重试/ })).toBeTruthy()
    expect(screen.getByText('时代背景')).toBeTruthy()
    expect(screen.getByRole('radio', { name: /维度化/ })).toBeTruthy()
  })
})

describe('NovelSettingPage 伏笔登记表面板（v4.3f）', () => {
  beforeEach(() => {
    useAppStore.setState({ projectOpen: true, projectPath: 'C:/novel/test' })
    vi.clearAllMocks()
    vi.mocked(GetWorldview).mockResolvedValue('# 世界观\n\n架空中世纪')
    vi.mocked(GetForeshadows).mockResolvedValue({ items: [] })
    vi.mocked(CheckConsistency).mockResolvedValue({ issues: [], total_issues: 0, summary: '✅ 未发现一致性问题' })
  })

  it('展示伏笔列表：内容/章节/状态徽标 + 回收率统计', async () => {
    vi.mocked(GetForeshadows).mockResolvedValue({
      items: [
        { id: 'f1', category: 'character', description: '主角左臂的旧伤', planted_in: '001.md', status: 'planted', is_long_term: true },
        { id: 'f2', category: 'plot', description: '神秘铜匣', planted_in: '002.md', status: 'hinted', is_long_term: false },
        { id: 'f3', category: 'world', description: '星门钥匙', planted_in: '003.md', revealed_in: '006.md', status: 'revealed', is_long_term: true },
      ],
    })
    render(<NovelSettingPage />)

    expect(await screen.findByText('主角左臂的旧伤')).toBeTruthy()
    expect(screen.getByText('神秘铜匣')).toBeTruthy()
    expect(screen.getByText('星门钥匙')).toBeTruthy()
    // 状态徽标（planted→hinted→revealed）
    expect(screen.getByText('已埋设')).toBeTruthy()
    expect(screen.getByText('已暗示')).toBeTruthy()
    expect(screen.getByText('已回收')).toBeTruthy()
    // 回收率 = revealed/total = 1/3
    expect(screen.getByText(/回收率 33%/)).toBeTruthy()
  })

  it('空态引导', async () => {
    render(<NovelSettingPage />)
    expect(await screen.findByText(/还没有伏笔登记/)).toBeTruthy()
  })
})

describe('NovelSettingPage 一致性检查面板（v4.3f）', () => {
  beforeEach(() => {
    useAppStore.setState({ projectOpen: true, projectPath: 'C:/novel/test' })
    vi.clearAllMocks()
    vi.mocked(GetWorldview).mockResolvedValue('# 世界观\n\n架空中世纪')
    vi.mocked(GetForeshadows).mockResolvedValue({ items: [] })
    vi.mocked(CheckConsistency).mockResolvedValue({ issues: [], total_issues: 0, summary: '✅ 未发现一致性问题' })
  })

  it('展示三类规则告警（严重度/描述）并可「重新检查」', async () => {
    vi.mocked(CheckConsistency).mockResolvedValue({
      total_issues: 2,
      summary: '发现 2 个问题（1 错误, 1 警告, 0 提示）',
      issues: [
        { severity: 'error', category: 'attribute', entity_name: '林晚', description: '瞳孔颜色前后矛盾', location: '第 3 章', evidence: '蓝→紫', suggestion: '统一为墨绿' },
        { severity: 'warning', category: 'timeline', entity_name: '', description: '时间线出现重叠', location: '第 5 章', evidence: '同一日两场战役', suggestion: '调整章节顺序' },
      ],
    })
    render(<NovelSettingPage />)

    expect(await screen.findByText(/瞳孔颜色前后矛盾/)).toBeTruthy()
    expect(screen.getByText(/时间线出现重叠/)).toBeTruthy()
    expect(screen.getByText('错误')).toBeTruthy()
    expect(screen.getByText('警告')).toBeTruthy()
    expect(screen.getByText(/角色属性/)).toBeTruthy()
    // 「时间线」同时出现在分类标签与描述中
    expect(screen.getAllByText(/时间线/).length).toBeGreaterThan(0)
    expect(screen.getByText(/发现 2 个问题/)).toBeTruthy()

    fireEvent.click(screen.getByRole('button', { name: /重新检查/ }))
    await waitFor(() => expect(CheckConsistency).toHaveBeenCalledTimes(2))
  })

  it('全部通过时显示成功空态', async () => {
    render(<NovelSettingPage />)
    expect(await screen.findByText(/全部通过，未发现一致性问题/)).toBeTruthy()
  })
})
