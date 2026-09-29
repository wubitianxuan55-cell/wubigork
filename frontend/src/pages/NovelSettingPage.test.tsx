import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within, act } from '@testing-library/react'
import { Modal } from 'antd'

// 屏蔽 Wails 绑定：jsdom 中没有 window.go。页面及其子面板（WorldviewSectionsEditor/
// ForeshadowPanel/ConsistencyPanel）经 gaea/lib/bridge 的 app 调用 NovelBindings。
// 用 importOriginal 保留原模块、仅替换 app 的 Novel 域方法；未 mock 的方法
// （SaveFileAs/PickFiles/ReadFileB64…）经 Proxy 在**每次属性访问时**回落真实代理
// （保持「调用时解析」），壳内导入/导出用例（window.go stub + gaeaToGaea 路由）不受影响。
vi.mock('../gaea/lib/bridge', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../gaea/lib/bridge')>()
  const novelStubs = {
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
  }
  return {
    ...actual,
    app: new Proxy(novelStubs, {
      get(target, prop) {
        if (prop in target) return Reflect.get(target, prop)
        return (actual.app as unknown as Record<string, unknown>)[String(prop)]
      },
    }),
  }
})

import NovelSettingPage from './NovelSettingPage'
import { useAppStore } from '../stores/appStore'
import { app } from '../gaea/lib/bridge'
import { resetNovelDirtyProviders } from '../components/novel/novelSwitchGuard'

// 跨用例清空切书闸门登记表与「已确认放弃」记账（模块级单例，否则脏态跨用例串味）
afterEach(() => { resetNovelDirtyProviders() })

// 坑复训（unsavedGuard.test 同款）：imperative Modal 的 DOM 不随 cleanup() 卸载，
// 跨用例残留会让「最新弹窗」定位错位——每例后显式销毁 + 让销毁提交一帧。
afterEach(async () => {
  Modal.destroyAll()
  await new Promise((r) => setTimeout(r, 0))
})

const confirmModals = () => Array.from(document.querySelectorAll<HTMLElement>('.ant-modal-confirm'))

/** 触发动作并返回**最新弹窗**的限定查询器（按弹窗数量增量定位，避免旧弹窗干扰）。 */
async function latestConfirm(trigger: () => void) {
  const before = confirmModals().length
  trigger()
  await waitFor(() => expect(confirmModals().length).toBeGreaterThan(before))
  const all = confirmModals()
  return within(all[all.length - 1])
}

describe('NovelSettingPage 纯文本设定编辑', () => {
  beforeEach(() => {
    useAppStore.setState({ projectOpen: true, projectPath: 'C:/novel/test' })
    vi.clearAllMocks()
    vi.mocked(app.GetWorldview).mockResolvedValue('# 世界观\n\n架空中世纪')
    vi.mocked(app.GetForeshadows).mockResolvedValue({ items: [] })
    vi.mocked(app.CheckConsistency).mockResolvedValue({ issues: [], total_issues: 0, summary: '✅ 未发现一致性问题' })
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
    expect(vi.mocked(app.SaveWorldview)).toHaveBeenCalledWith('蒸汽纪元')
  })

  it('读取失败：横幅可见、保存禁用、Ctrl+S 被拦截；重试成功后恢复（v4.361 防覆盖）', async () => {
    vi.mocked(app.GetWorldview).mockRejectedValueOnce(new Error('db locked'))
    render(<NovelSettingPage />)

    const banner = await screen.findByTestId('novel-setting-load-failed')
    expect(banner.textContent).toContain('设定读取失败')

    const saveBtn = (await screen.findByRole('button', { name: /保存/ })) as HTMLButtonElement
    expect(saveBtn.disabled).toBe(true)

    // Ctrl+S 走 handleSave 同一入口，读失败期间必须被拦
    fireEvent.keyDown(window, { key: 's', ctrlKey: true })
    expect(vi.mocked(app.SaveWorldview)).not.toHaveBeenCalled()

    // 重试成功后横幅消失、保存恢复可用
    vi.mocked(app.GetWorldview).mockResolvedValue('# 世界观\n\n重试恢复')
    fireEvent.click(banner.querySelector('button')!)
    await waitFor(() => {
      expect(screen.queryByTestId('novel-setting-load-failed')).toBeNull()
    })
    const saveBtn2 = (screen.getByRole('button', { name: /保存/ })) as HTMLButtonElement
    expect(saveBtn2.disabled).toBe(false)
  })

  // ── v4.425 B2：切书静默丢未保存设定 → 至少如实提示（对齐 CreatePage 口径） ──
  it('切书丢弃未保存设定：给出可见提示', async () => {
    render(<NovelSettingPage />)
    const editor = (await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)) as HTMLTextAreaElement
    fireEvent.change(editor, { target: { value: '# 第一本的未保存草稿' } })
    expect(screen.getByText('有未保存修改')).toBeTruthy()

    act(() => { useAppStore.setState({ projectPath: 'C:/novel/第二本' }) })

    expect(await screen.findByText(/已切换小说：上一本未保存的设定修改未保留/)).toBeTruthy()
  })

  it('切书后不把上一本的脏态挂到新书上（新书内容与草稿相同也不喊脏）', async () => {
    render(<NovelSettingPage />)
    const editor = (await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)) as HTMLTextAreaElement
    fireEvent.change(editor, { target: { value: '# 第二本的设定' } })
    expect(screen.getByText('有未保存修改')).toBeTruthy()

    // 第二本磁盘上的内容恰与本地草稿相同：切过去后不得残留「有未保存修改」
    vi.mocked(app.GetWorldview).mockResolvedValue('# 第二本的设定')
    act(() => { useAppStore.setState({ projectPath: 'C:/novel/第二本' }) })

    await waitFor(() => expect(screen.getByText('无修改')).toBeTruthy())
    expect(screen.queryByText('有未保存修改')).toBeNull()
  })

  it('切换到渲染模式直接渲染设定文本', async () => {
    render(<NovelSettingPage />)
    await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)
    fireEvent.click(screen.getByRole('radio', { name: /渲染/ }))
    expect(await screen.findByText('架空中世纪')).toBeTruthy()
  })

  it('AI 回复未自动解析时，可手动点击「应用到设定」覆盖编辑器', async () => {
    vi.mocked(app.ChatWorldview).mockResolvedValue({
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
    vi.mocked(app.GetWorldview).mockResolvedValue('# 世界观\n\n架空中世纪')
    vi.mocked(app.GetWorldviewSections).mockResolvedValue({
      sections: [
        { id: 'era', title: '时代背景', content: '架空中世纪', order: 1 },
        { id: 'geography', title: '地理风貌', content: '', order: 2 },
        { id: 'factions', title: '势力格局', content: '', order: 3 },
        { id: 'rules', title: '规则体系', content: '', order: 4 },
        { id: 'culture', title: '文化习俗', content: '', order: 5 },
        { id: 'history', title: '历史事件', content: '', order: 6 },
      ],
    })
    vi.mocked(app.GetForeshadows).mockResolvedValue({ items: [] })
    vi.mocked(app.CheckConsistency).mockResolvedValue({ issues: [], total_issues: 0, summary: '✅ 未发现一致性问题' })
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
    await waitFor(() => expect(vi.mocked(app.SaveAllWorldviewSections)).toHaveBeenCalledTimes(1))
    const payload = JSON.parse(vi.mocked(app.SaveAllWorldviewSections).mock.calls[0][0] as string) as Array<{ id: string; content: string }>
    expect(payload).toHaveLength(6)
    expect(payload.find((s) => s.id === 'era')?.content).toBe('蒸汽纪元，机械飞升')
  })

  it('维度加载失败降级提示且不崩溃', async () => {
    vi.mocked(app.GetWorldviewSections).mockRejectedValue(new Error('项目数据损坏'))
    render(<NovelSettingPage />)
    await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)
    fireEvent.click(screen.getByRole('radio', { name: /维度化/ }))

    // 降级提示出现，页面仍可用（维度卡仍渲染、Segmented 仍在）
    expect(await screen.findByText(/维度数据加载失败/)).toBeTruthy()
    expect(screen.getByRole('button', { name: /重试/ })).toBeTruthy()
    expect(screen.getByText('时代背景')).toBeTruthy()
    expect(screen.getByRole('radio', { name: /维度化/ })).toBeTruthy()
  })

  // ── v4.425 B1（P0）：切书必须重拉六维，否则「保存全部维度」会把 A 书写进 B 书 ──
  it('切书后六维重拉：显示新书内容且不残留上一本（B1 反向守卫）', async () => {
    vi.mocked(app.GetWorldviewSections).mockResolvedValue({
      sections: [{ id: 'era', title: '时代背景', content: '第一本的洪荒纪元', order: 1 }],
    })
    render(<NovelSettingPage />)
    await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)
    fireEvent.click(screen.getByRole('radio', { name: /维度化/ }))
    const eraA = await screen.findByPlaceholderText(/撰写「时代背景」设定/) as HTMLTextAreaElement
    expect(eraA.value).toBe('第一本的洪荒纪元')
    const callsBefore = vi.mocked(app.GetWorldviewSections).mock.calls.length

    vi.mocked(app.GetWorldviewSections).mockResolvedValue({
      sections: [{ id: 'era', title: '时代背景', content: '第二本的蒸汽纪元', order: 1 }],
    })
    act(() => { useAppStore.setState({ projectPath: 'C:/novel/第二本' }) })

    await waitFor(() => expect(vi.mocked(app.GetWorldviewSections).mock.calls.length).toBeGreaterThan(callsBefore))
    await waitFor(() => {
      expect((screen.getByPlaceholderText(/撰写「时代背景」设定/) as HTMLTextAreaElement).value).toBe('第二本的蒸汽纪元')
    })
    expect(screen.queryByDisplayValue('第一本的洪荒纪元')).toBeNull()
  })

  it('清空项目路径：回到引导态且六维不残留上一本', async () => {
    vi.mocked(app.GetWorldviewSections).mockResolvedValue({
      sections: [{ id: 'era', title: '时代背景', content: '第一本的洪荒纪元', order: 1 }],
    })
    render(<NovelSettingPage />)
    await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)
    fireEvent.click(screen.getByRole('radio', { name: /维度化/ }))
    expect(await screen.findByDisplayValue('第一本的洪荒纪元')).toBeTruthy()

    act(() => { useAppStore.setState({ projectOpen: false, projectPath: '' }) })

    expect(await screen.findByText(/请先在「书架」打开或创建一部小说项目/)).toBeTruthy()
    expect(screen.queryByDisplayValue('第一本的洪荒纪元')).toBeNull()
  })
})

describe('NovelSettingPage 伏笔登记表面板（v4.3f）', () => {
  beforeEach(() => {
    useAppStore.setState({ projectOpen: true, projectPath: 'C:/novel/test' })
    vi.clearAllMocks()
    vi.mocked(app.GetWorldview).mockResolvedValue('# 世界观\n\n架空中世纪')
    vi.mocked(app.GetForeshadows).mockResolvedValue({ items: [] })
    vi.mocked(app.CheckConsistency).mockResolvedValue({ issues: [], total_issues: 0, summary: '✅ 未发现一致性问题' })
  })

  it('展示伏笔列表：内容/章节/状态徽标 + 回收率统计', async () => {
    vi.mocked(app.GetForeshadows).mockResolvedValue({
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
    vi.mocked(app.GetWorldview).mockResolvedValue('# 世界观\n\n架空中世纪')
    vi.mocked(app.GetForeshadows).mockResolvedValue({ items: [] })
    vi.mocked(app.CheckConsistency).mockResolvedValue({ issues: [], total_issues: 0, summary: '✅ 未发现一致性问题' })
  })

  it('展示三类规则告警（严重度/描述）并可「重新检查」', async () => {
    vi.mocked(app.CheckConsistency).mockResolvedValue({
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
    await waitFor(() => expect(vi.mocked(app.CheckConsistency)).toHaveBeenCalledTimes(2))
  })

  it('全部通过时显示成功空态', async () => {
    render(<NovelSettingPage />)
    expect(await screen.findByText(/全部通过，未发现一致性问题/)).toBeTruthy()
  })
})

// ── 审计刀B b：导入/导出双门收口（壳内系统对话框 + 系统另存为；浏览器原生回退）──
describe('NovelSettingPage 导入/导出双门（审计刀B b）', () => {
  let saveFileAs: ReturnType<typeof vi.fn>
  let pickFiles: ReturnType<typeof vi.fn>
  let readFileB64: ReturnType<typeof vi.fn>

  /** UTF-8 安全 base64（jsdom btoa 仅 Latin1；FileReader/后端均按 UTF-8 字节编码） */
  const b64Of = (s: string) => btoa(String.fromCharCode(...new TextEncoder().encode(s)))

  /** 装真壳同款 window.go 绑定面（bridge realApp 按方法名路由到 Gaea 前缀） */
  const stubShell = (picked: unknown[], b64: string) => {
    saveFileAs = vi.fn(async () => 'C:/导出/novel_setting.md')
    pickFiles = vi.fn(async () => picked)
    readFileB64 = vi.fn(async () => b64)
    ;(window as unknown as { go?: unknown }).go = {
      app: {
        NovelSettingTestB: {
          GaeaSaveFileAs: saveFileAs,
          GaeaPickFiles: pickFiles,
          GaeaReadFileB64: readFileB64,
        },
      },
    }
  }

  beforeEach(() => {
    useAppStore.setState({ projectOpen: true, projectPath: 'C:/novel/test' })
    vi.clearAllMocks()
    vi.mocked(app.GetWorldview).mockResolvedValue('# 世界观\n\n架空中世纪')
    vi.mocked(app.GetForeshadows).mockResolvedValue({ items: [] })
    vi.mocked(app.CheckConsistency).mockResolvedValue({ issues: [], total_issues: 0, summary: '✅ 未发现一致性问题' })
  })

  afterEach(() => {
    delete (window as unknown as { go?: unknown }).go
    vi.restoreAllMocks()
  })

  it('壳内导出：GaeaSaveFileAs 收到 novel_setting.md 与内容 base64（不再 <a download>）', async () => {
    stubShell([], '')
    render(<NovelSettingPage />)
    const editor = (await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)) as HTMLTextAreaElement
    fireEvent.change(editor, { target: { value: '# 蒸汽纪元' } })
    fireEvent.click(screen.getByRole('button', { name: /导出/ }))
    await waitFor(() => expect(saveFileAs).toHaveBeenCalledTimes(1))
    expect(saveFileAs.mock.calls[0][0]).toBe('novel_setting.md')
    expect(saveFileAs.mock.calls[0][1]).toBe(b64Of('# 蒸汽纪元'))
  })

  it('壳内导入：PickFiles → ReadFileB64 还原 File 喂原 FileReader 管线', async () => {
    stubShell(
      [{ path: 'C:/novel/新设定.md', name: '新设定.md', type: 'file', size: 5 }],
      b64Of('# 导入的设定'),
    )
    render(<NovelSettingPage />)
    const editor = (await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)) as HTMLTextAreaElement
    expect(editor.value).toContain('架空中世纪')
    fireEvent.click(screen.getByRole('button', { name: /导入/ }))
    await waitFor(() => expect(readFileB64).toHaveBeenCalledWith('C:/novel/新设定.md'))
    await waitFor(() => expect(editor.value).toContain('# 导入的设定'))
    expect(await screen.findByText(/已导入「新设定.md」/)).toBeTruthy()
  })

  it('浏览器导入：回退隐藏 input.click（jsdom 无 window.go，既有口径不变）', async () => {
    const inputClick = vi.spyOn(HTMLInputElement.prototype, 'click').mockImplementation(() => {})
    render(<NovelSettingPage />)
    await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)
    fireEvent.click(screen.getByRole('button', { name: /导入/ }))
    expect(inputClick).toHaveBeenCalledTimes(1)
    inputClick.mockRestore()
  })

  // ── v4.421 线3 P2：dirty 时导入会静默覆盖未保存设定 → 走共享确认原语 ──
  // 坑复训：imperative Modal 关闭后 DOM 仍留在 document（隐藏态），跨用例会污染
  // 「按可见文案/角色」的全局查询——本组一律把页面查询限定在 render 容器内。
  it('dirty 时导入先确认，点「覆盖导入」才替换内容', async () => {
    stubShell(
      [{ path: 'C:/novel/覆盖导入.md', name: '覆盖导入.md', type: 'file', size: 5 }],
      b64Of('# 导入的设定'),
    )
    const page = render(<NovelSettingPage />)
    const editor = (await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)) as HTMLTextAreaElement
    fireEvent.change(editor, { target: { value: '# 未保存的草稿' } })
    expect(screen.getByText('有未保存修改')).toBeTruthy()

    const scoped = await latestConfirm(() =>
      fireEvent.click(within(page.container).getByRole('button', { name: /导\s*入$/ })))
    // 标题父子双匹配（antd 弹窗 wrapper + 内层 div）→ 取集合断言，不用 getByText
    expect(scoped.getAllByText('导入会覆盖未保存的设定', { exact: false }).length).toBeGreaterThan(0)
    expect(scoped.getByText(/会用文件内容覆盖编辑器/)).toBeTruthy()
    // 确认前内容保持未保存草稿（未静默覆盖）
    expect(editor.value).toBe('# 未保存的草稿')

    fireEvent.click(scoped.getByRole('button', { name: /覆盖\s*导入/ }))
    await waitFor(() => expect(editor.value).toContain('# 导入的设定'))
    expect(await screen.findByText(/已导入「覆盖导入.md」/)).toBeTruthy()
  })

  it('dirty 时导入确认点「取消」：内容与未保存修改都不变（✕/Esc 同义）', async () => {
    stubShell(
      [{ path: 'C:/novel/新设定.md', name: '新设定.md', type: 'file', size: 5 }],
      b64Of('# 导入的设定'),
    )
    const page = render(<NovelSettingPage />)
    const editor = (await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)) as HTMLTextAreaElement
    fireEvent.change(editor, { target: { value: '# 未保存的草稿' } })

    const scoped = await latestConfirm(() =>
      fireEvent.click(within(page.container).getByRole('button', { name: /导\s*入$/ })))
    fireEvent.click(scoped.getByRole('button', { name: /^取\s*消$/ }))
    await new Promise((r) => setTimeout(r, 0))

    // 选取/读取确实发生过（证明走到了导入确认），但内容未被替换
    expect(readFileB64).toHaveBeenCalledWith('C:/novel/新设定.md')
    expect(editor.value).toBe('# 未保存的草稿')
    expect(screen.getByText('有未保存修改')).toBeTruthy()
  })
})

// ── v4.421 跨线契约：窗口级 Ctrl+S 按「本页是否当前子页」门控 ──
// 反向守卫：小说五子页常驻挂载，若设定页无条件挂 window keydown，则在创作页/
// 阅读页按下 Ctrl+S 会「顺带保存设定」并吞掉别页的默认行为。
describe('NovelSettingPage Ctrl+S 门控（active，v4.421 跨线契约）', () => {
  beforeEach(() => {
    useAppStore.setState({ projectOpen: true, projectPath: 'C:/novel/test' })
    vi.clearAllMocks()
    vi.mocked(app.GetWorldview).mockResolvedValue('# 世界观\n\n架空中世纪')
    vi.mocked(app.GetForeshadows).mockResolvedValue({ items: [] })
    vi.mocked(app.CheckConsistency).mockResolvedValue({ issues: [], total_issues: 0, summary: '✅ 未发现一致性问题' })
  })

  /** 派发一次窗口级 Ctrl+S，返回事件本体（用于断言 preventDefault 是否发生） */
  const pressCtrlS = () => {
    const ev = new KeyboardEvent('keydown', { key: 's', ctrlKey: true, bubbles: true, cancelable: true })
    window.dispatchEvent(ev)
    return ev
  }

  it('active={false}：Ctrl+S 既不保存也不 preventDefault（不吞别页快捷键）', async () => {
    render(<NovelSettingPage active={false} />)
    const editor = (await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)) as HTMLTextAreaElement
    fireEvent.change(editor, { target: { value: '# 未保存的新设定' } })
    expect(screen.getByText('有未保存修改')).toBeTruthy()

    const ev = pressCtrlS()
    await new Promise((r) => setTimeout(r, 0))

    expect(ev.defaultPrevented).toBe(false)
    expect(vi.mocked(app.SaveWorldview)).not.toHaveBeenCalled()
    expect(screen.getByText('有未保存修改')).toBeTruthy()
  })

  it('不传 active（默认 true）：Ctrl+S 保存并 preventDefault（既有口径不变）', async () => {
    render(<NovelSettingPage />)
    const editor = (await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)) as HTMLTextAreaElement
    fireEvent.change(editor, { target: { value: '蒸汽纪元' } })

    const ev = pressCtrlS()

    expect(ev.defaultPrevented).toBe(true)
    await waitFor(() => expect(vi.mocked(app.SaveWorldview)).toHaveBeenCalledWith('蒸汽纪元'))
  })
})

// ── v4.421 体验项：下行「伏笔 + 一致性」区可折叠（默认展开 + 持久化） ──
describe('NovelSettingPage 下行面板折叠（v4.421）', () => {
  const COLLAPSED_KEY = 'gaea.novel.settingPanelsCollapsed'

  beforeEach(() => {
    localStorage.removeItem(COLLAPSED_KEY)
    useAppStore.setState({ projectOpen: true, projectPath: 'C:/novel/test' })
    vi.clearAllMocks()
    vi.mocked(app.GetWorldview).mockResolvedValue('# 世界观\n\n架空中世纪')
    vi.mocked(app.GetForeshadows).mockResolvedValue({ items: [] })
    vi.mocked(app.CheckConsistency).mockResolvedValue({ issues: [], total_issues: 0, summary: '✅ 未发现一致性问题' })
  })

  afterEach(() => { localStorage.removeItem(COLLAPSED_KEY) })

  it('默认展开；点「收起」隐藏双面板并写入 localStorage（aria 齐备）', async () => {
    render(<NovelSettingPage />)
    await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)

    expect(await screen.findByText('伏笔登记')).toBeTruthy()
    expect(screen.getByTestId('novel-setting-panels')).toBeTruthy()
    const toggle = screen.getByTestId('novel-setting-panels-toggle')
    expect(toggle.getAttribute('aria-expanded')).toBe('true')
    expect(screen.getByLabelText('收起伏笔与一致性面板')).toBeTruthy()

    fireEvent.click(toggle)

    expect(screen.queryByTestId('novel-setting-panels')).toBeNull()
    expect(screen.queryByText('伏笔登记')).toBeNull()
    expect(screen.getByTestId('novel-setting-panels-toggle').getAttribute('aria-expanded')).toBe('false')
    expect(screen.getByLabelText('展开伏笔与一致性面板')).toBeTruthy()
    expect(localStorage.getItem(COLLAPSED_KEY)).toBe('1')
  })

  it('折叠态持久化：重新渲染（刷新）后仍保持收起，再点即展开', async () => {
    const first = render(<NovelSettingPage />)
    await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)
    fireEvent.click(screen.getByTestId('novel-setting-panels-toggle'))
    expect(localStorage.getItem(COLLAPSED_KEY)).toBe('1')
    first.unmount()

    render(<NovelSettingPage />)
    await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)
    expect(screen.getByTestId('novel-setting-panels-toggle').getAttribute('aria-expanded')).toBe('false')
    expect(screen.queryByTestId('novel-setting-panels')).toBeNull()

    fireEvent.click(screen.getByTestId('novel-setting-panels-toggle'))
    expect(await screen.findByTestId('novel-setting-panels')).toBeTruthy()
    expect(localStorage.getItem(COLLAPSED_KEY)).toBe('0')
  })
})

// ── v4.429 观察池#2：设定 Agent 回填不得覆盖等待期的手改 ──
// ChatPanel 输入区等待期已禁（disabled={loading}），但设定**编辑器**不禁——
// 作者等待期手改正文后，AI worldview 照样 setContent 覆盖（旧缺陷）。
describe('NovelSettingPage AI 回填守卫（观察池#2）', () => {
  beforeEach(() => {
    useAppStore.setState({ projectOpen: true, projectPath: 'C:/novel/test' })
    vi.mocked(app.GetWorldview).mockResolvedValue('# 世界观\n\n架空中世纪')
    vi.mocked(app.GetForeshadows).mockResolvedValue({ items: [] })
  })

  it('等待期正文已变：不覆盖编辑器，修改稿附在回复里走手动应用', async () => {
    let resolveChat!: (v: { reply: string; worldview: string }) => void
    vi.mocked(app.ChatWorldview).mockReturnValue(new Promise((r) => { resolveChat = r }))

    render(<NovelSettingPage />)
    const editor = (await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)) as HTMLTextAreaElement
    expect(editor.value).toContain('架空中世纪')

    // 发起对话（挂起中）
    const chatInput = screen.getByPlaceholderText(/描述你想要的设定修改/)
    fireEvent.change(chatInput, { target: { value: '重写设定' } })
    fireEvent.keyDown(chatInput, { key: 'Enter' })

    // 等待期作者手改正文
    fireEvent.change(editor, { target: { value: '等待期手改的设定正文' } })

    // AI 返回 worldview——编辑器不得被覆盖
    await act(async () => { resolveChat({ reply: '改好了', worldview: '# AI 修改稿' }) })

    expect(editor.value).toBe('等待期手改的设定正文')
    // 修改稿进回复（markdown 围栏，手动「应用」可提取）+ 如实提示未自动应用
    expect(await screen.findByText(/未自动应用/, {}, { timeout: 3000 })).toBeTruthy()
  })

  it('等待期正文未变：worldview 照旧直接回填编辑器', async () => {
    vi.mocked(app.ChatWorldview).mockResolvedValue({ reply: '改好了', worldview: '# AI 新设定' })
    render(<NovelSettingPage />)
    const editor = (await screen.findByPlaceholderText(/在此撰写或粘贴小说设定/)) as HTMLTextAreaElement

    const chatInput = screen.getByPlaceholderText(/描述你想要的设定修改/)
    fireEvent.change(chatInput, { target: { value: '重写设定' } })
    fireEvent.keyDown(chatInput, { key: 'Enter' })

    await waitFor(() => expect(editor.value).toBe('# AI 新设定'), { timeout: 3000 })
  })
})
