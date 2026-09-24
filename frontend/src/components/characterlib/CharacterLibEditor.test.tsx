import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { Modal } from 'antd'
import { getComfyUITaskProgress, cancelImageGeneration } from '../../api/image'
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/react'
import CharacterLibEditor from './CharacterLibEditor'
import type { LibraryCharacter } from '../../api/characterlib'

const { readFileAsDataURL: readFileAsDataURLMock } = vi.hoisted(() => ({
  readFileAsDataURL: vi.fn(),
}))

vi.mock('../../api/characterlib', () => ({
  saveCharacter: vi.fn(),
  generateFill: vi.fn(),
  generatePortrait: vi.fn(),
  generatePortraitWithRef: vi.fn(),
  generateCharacterSheet: vi.fn(),
  generateRandom: vi.fn(),
  scoreCharacterConsistency: vi.fn(),
}))

vi.mock('../../api/image', () => ({
  readFileAsDataURL: readFileAsDataURLMock,
  getComfyUITaskProgress: vi.fn().mockResolvedValue({ status: '', elapsed: 0, percent: -1, node: '' }),
  cancelImageGeneration: vi.fn().mockResolvedValue(true),
}))

import { saveCharacter, generateFill, generatePortrait, generatePortraitWithRef, generateRandom, generateCharacterSheet, scoreCharacterConsistency } from '../../api/characterlib'

const mockedSave = vi.mocked(saveCharacter)
const mockedFill = vi.mocked(generateFill)
const mockedPortrait = vi.mocked(generatePortrait)
const mockedPortraitWithRef = vi.mocked(generatePortraitWithRef)
const mockedCharacterSheet = vi.mocked(generateCharacterSheet)
const mockedRandom = vi.mocked(generateRandom)
const mockedScore = vi.mocked(scoreCharacterConsistency)
const mockedCancelGen = vi.mocked(cancelImageGeneration)

function makeCharacter(overrides: Partial<LibraryCharacter> = {}): LibraryCharacter {
  return {
    id: 'c1',
    name: '苏念',
    kind: 'custom',
    gender: 'female',
    age: '23',
    tags: ['剑修', '女主'],
    portraitUrl: '',
    roleType: 'protagonist',
    arc: '从逃避到直面宿命。',
    status: 'Alive',
    chatEnabled: false,
    dims: { T: 80, I: 60, S: 50, O: 70, R: 40 },
    createdAt: '',
    updatedAt: '',
    hidden: false,
    ...overrides,
  } as unknown as LibraryCharacter
}

function renderEditor(overrides: {
  character?: LibraryCharacter | null
  projects?: string[]
  isCurrentPersona?: boolean
} = {}) {
  const onClose = vi.fn()
  const onSaved = vi.fn()
  const character = 'character' in overrides ? overrides.character : makeCharacter()
  const utils = render(
    <CharacterLibEditor
      open
      character={character ?? null}
      projects={overrides.projects ?? []}
      index={0}
      isCurrentPersona={overrides.isCurrentPersona}
      onClose={onClose}
      onSaved={onSaved}
    />,
  )
  return { onClose, onSaved, ...utils }
}

beforeEach(() => {
  mockedSave.mockReset()
  mockedFill.mockReset()
  mockedPortrait.mockReset()
  mockedPortraitWithRef.mockReset()
  mockedCharacterSheet.mockReset()
  mockedRandom.mockReset()
  readFileAsDataURLMock.mockReset()
  readFileAsDataURLMock.mockResolvedValue('data:image/png;base64,PATHREF')
})

// 壳内用例装的 window.go stub 用完即清，防泄漏进同文件浏览器分支用例
afterEach(() => {
  delete (window as unknown as { go?: unknown }).go
})

describe('CharacterLibEditor（档案详情）', () => {
  it('编辑态渲染档案眉：编号 + 类型标签', () => {
    renderEditor()
    expect(screen.getByText('角色档案 · NO.001')).toBeTruthy()
    expect(screen.getByText('自定义')).toBeTruthy()
  })

  it('新建态渲染“新建档案”', () => {
    renderEditor({ character: null })
    expect(screen.getByText('新建档案')).toBeTruthy()
  })

  it('有立绘时横幅渲染图片，无立绘时渲染首字占位', () => {
    renderEditor({ character: makeCharacter({ portraitUrl: 'https://x/1.png' }) })
    expect(document.body.querySelector('.cd-hero-img')?.getAttribute('src')).toBe('https://x/1.png')
    renderEditor()
    expect(document.body.querySelector('.cd-hero-ph')?.textContent).toBe('苏')
  })

  it('身份栏渲染名称与元数据字段', () => {
    renderEditor()
    expect((screen.getByPlaceholderText('角色名') as HTMLInputElement).value).toBe('苏念')
    expect(screen.getByText('性别')).toBeTruthy()
    expect(screen.getByText('年龄')).toBeTruthy()
    expect(screen.getByText('定位')).toBeTruthy()
    expect(screen.getByText('状态')).toBeTruthy()
  })

  it('卷宗正文渲染四个分区', () => {
    renderEditor()
    expect(screen.getByText('小说设定')).toBeTruthy()
    expect(screen.getByText('对话样本')).toBeTruthy()
    expect(screen.getByText('备注')).toBeTruthy()
    expect(screen.getByText('聊天设定')).toBeTruthy()
  })

  it('切换可聊天开关后保存携带 chatEnabled=true', async () => {
    mockedSave.mockResolvedValue(makeCharacter({ chatEnabled: true }))
    renderEditor()
    fireEvent.click(document.body.querySelector('.cd-chat-toggle .ant-switch') as HTMLElement)
    fireEvent.click(screen.getByText('保存'))
    expect(mockedSave).toHaveBeenCalledTimes(1)
    expect(mockedSave.mock.calls[0][0].chatEnabled).toBe(true)
  })

  it('保存成功后回调 onSaved 与 onClose', async () => {
    const saved = makeCharacter({ chatEnabled: true })
    mockedSave.mockResolvedValue(saved)
    const { onClose, onSaved } = renderEditor()
    fireEvent.click(screen.getByText('保存'))
    expect(mockedSave).toHaveBeenCalledTimes(1)
    await vi.waitFor(() => {
      expect(onSaved).toHaveBeenCalledWith(saved)
      expect(onClose).toHaveBeenCalledTimes(1)
    })
  })

  it('名称为空时不保存并提示', async () => {
    renderEditor()
    fireEvent.change(screen.getByPlaceholderText('角色名'), { target: { value: '' } })
    fireEvent.click(screen.getByText('保存'))
    expect(mockedSave).not.toHaveBeenCalled()
  })

  it('isCurrentPersona 时渲染当前人格徽标', () => {
    renderEditor({ isCurrentPersona: true })
    expect(screen.getByText('当前人格')).toBeTruthy()
  })

  it('渲染项目引用信息', () => {
    renderEditor({ projects: ['星落之城'] })
    expect(screen.getByText(/被 1 个项目引用/)).toBeTruthy()
  })

  it('随机补齐调用 generateFill 并回填空缺字段', async () => {
    mockedFill.mockResolvedValue(makeCharacter({ personality: '清冷剑修，寡言重诺' }))
    renderEditor()
    fireEvent.click(screen.getByText('随机补齐'))
    await vi.waitFor(() => {
      expect(mockedFill).toHaveBeenCalledTimes(1)
      const areas = document.body.querySelectorAll('.cd-area')
      expect((areas[0] as HTMLTextAreaElement).value).toBe('清冷剑修，寡言重诺')
    })
  })

  it('生成剧照调用 generatePortrait 并更新立绘横幅', async () => {
    mockedPortrait.mockResolvedValue('data:image/png;base64,AAAA')
    renderEditor()
    fireEvent.click(screen.getByText('生成剧照'))
    await vi.waitFor(() => {
      expect(mockedPortrait).toHaveBeenCalledTimes(1)
      expect(document.body.querySelector('.cd-hero-img')?.getAttribute('src')).toBe('data:image/png;base64,AAAA')
    })
  })

  it('名称为空时随机补齐与生成剧照不调用后端', async () => {
    renderEditor()
    fireEvent.change(screen.getByPlaceholderText('角色名'), { target: { value: '' } })
    fireEvent.click(screen.getByText('随机补齐'))
    fireEvent.click(screen.getByText('生成剧照'))
    expect(mockedFill).not.toHaveBeenCalled()
    expect(mockedPortrait).not.toHaveBeenCalled()
  })

  it('全部随机调用 generateRandom(fields=all) 并回填性格', async () => {
    mockedRandom.mockResolvedValue(makeCharacter({ personality: '冷冽刀客，言出必践' }))
    renderEditor()
    fireEvent.click(screen.getByText('全部随机'))
    await vi.waitFor(() => {
      expect(mockedRandom).toHaveBeenCalledTimes(1)
      expect(mockedRandom.mock.calls[0][1]).toBe('all')
      const areas = document.body.querySelectorAll('.cd-area')
      expect((areas[0] as HTMLTextAreaElement).value).toBe('冷冽刀客，言出必践')
    })
  })

  it('字段骰子单独随机：性格', async () => {
    mockedRandom.mockResolvedValue(makeCharacter({ personality: '高冷寡言，外冷内热' }))
    renderEditor()
    fireEvent.click(screen.getByTitle('随机生成性格'))
    await vi.waitFor(() => {
      expect(mockedRandom).toHaveBeenCalledTimes(1)
      expect(mockedRandom.mock.calls[0][1]).toBe('personality')
      const areas = document.body.querySelectorAll('.cd-area')
      expect((areas[0] as HTMLTextAreaElement).value).toBe('高冷寡言，外冷内热')
    })
  })

  it('五维人格骰子调用 generateRandom(fields=dims) 并按性格回填', async () => {
    mockedRandom.mockResolvedValue(makeCharacter({ dims: { T: 20, I: 30, S: 40, O: 50, R: 80 } }))
    renderEditor()
    fireEvent.click(screen.getByTitle('按性格随机五维人格'))
    await vi.waitFor(() => {
      expect(mockedRandom).toHaveBeenCalledTimes(1)
      expect(mockedRandom.mock.calls[0][1]).toBe('dims')
    })
  })

  it('随机补齐后五维人格默认值时计入填充数', async () => {
    mockedFill.mockResolvedValue(makeCharacter({ dims: { T: 85, I: 40, S: 20, O: 70, R: 60 } }))
    renderEditor({ character: makeCharacter({ dims: { T: 50, I: 50, S: 50, O: 50, R: 50 } }) })
    fireEvent.click(screen.getByText('随机补齐'))
    await vi.waitFor(() => {
      expect(mockedFill).toHaveBeenCalledTimes(1)
      expect(screen.getByText(/已补齐 1 处空缺/)).toBeTruthy()
    })
  })

  it('枚举字段（性别）本地随机，不调用后端', () => {
    renderEditor()
    fireEvent.click(screen.getByTitle('随机生成性别'))
    expect(mockedRandom).not.toHaveBeenCalled()
  })

  // ── v4.3g 参考图管理 ─────────────────────────────────────────

  it('渲染参考图列表：data URL 直接展示为缩略图', () => {
    renderEditor({
      character: makeCharacter({
        referenceImages: ['data:image/png;base64,QUFB', 'data:image/png;base64,QkJC'],
      }),
    })
    expect(screen.getByText('参考图')).toBeTruthy()
    const thumbs = document.body.querySelectorAll('.cd-ref-img')
    expect(thumbs.length).toBe(2)
    expect(thumbs[0]?.getAttribute('src')).toBe('data:image/png;base64,QUFB')
    expect(thumbs[1]?.getAttribute('src')).toBe('data:image/png;base64,QkJC')
  })

  it('无参考图时显示空态与「添加参考图」按钮', () => {
    renderEditor()
    expect(document.body.querySelectorAll('.cd-ref').length).toBe(0)
    expect(screen.getByText('添加参考图')).toBeTruthy()
  })

  it('添加参考图：选择文件后追加 data URL 并显示缩略图', async () => {
    renderEditor()
    const input = document.body.querySelector('input[type="file"]') as HTMLInputElement
    const file = new File(['x'], 'ref.png', { type: 'image/png' })
    fireEvent.change(input, { target: { files: [file] } })
    await waitFor(() => {
      expect(document.body.querySelectorAll('.cd-ref-img').length).toBe(1)
    })
    // 保存时携带 referenceImages
    mockedSave.mockResolvedValue(makeCharacter({ referenceImages: ['data:image/png;base64,QUFB'] }))
    fireEvent.click(screen.getByText('保存'))
    await waitFor(() => {
      const payload = mockedSave.mock.calls[0][0] as Partial<LibraryCharacter>
      expect(Array.isArray(payload.referenceImages)).toBe(true)
      expect((payload.referenceImages ?? []).length).toBe(1)
    })
  })

  // ── 审计刀A：Wails 壳内「添加参考图」走 GaeaPickFiles 系统对话框 ──────────

  it('壳内添加参考图：GaeaPickFiles+GaeaReadFileB64 读回 data URL 追加列表（不点隐藏 input）', async () => {
    const pickFiles = vi.fn(async () => ([
      { path: 'C:/imgs/ref.png', name: 'ref.png', type: 'image', size: 5 },
    ]))
    const readFileB64 = vi.fn(async () => 'aGVsbG8=') // "hello"
    // 真壳同款 Gaea 前缀绑定面（bridge realApp 按方法名路由）
    ;(window as unknown as { go?: unknown }).go = {
      app: { CharLibPickTest: { GaeaPickFiles: pickFiles, GaeaReadFileB64: readFileB64 } },
    }
    // 壳内分支不得回落浏览器 input.click()（jsdom 点了也不弹框，正是 P0 本体）
    const inputClick = vi.spyOn(HTMLInputElement.prototype, 'click').mockImplementation(() => {})
    try {
      renderEditor()
      fireEvent.click(screen.getByText('添加参考图'))
      await waitFor(() => {
        expect(pickFiles).toHaveBeenCalledTimes(1)
        expect(readFileB64).toHaveBeenCalledWith('C:/imgs/ref.png')
        expect(document.body.querySelectorAll('.cd-ref-img').length).toBe(1)
      })
      expect(document.body.querySelector('.cd-ref-img')?.getAttribute('src'))
        .toBe('data:image/png;base64,aGVsbG8=')
      expect(inputClick).not.toHaveBeenCalled()
      // 保存载荷携带读回的 data URL（同浏览器路径口径）
      mockedSave.mockResolvedValue(makeCharacter())
      fireEvent.click(screen.getByText('保存'))
      await waitFor(() => {
        const payload = mockedSave.mock.calls[0][0] as Partial<LibraryCharacter>
        expect(payload.referenceImages).toEqual(['data:image/png;base64,aGVsbG8='])
      })
    } finally {
      inputClick.mockRestore()
    }
  })

  it('移除参考图：点击删除后列表减少', () => {
    renderEditor({
      character: makeCharacter({
        referenceImages: ['data:image/png;base64,QUFB', 'data:image/png;base64,QkJC'],
      }),
    })
    const removeButtons = screen.getAllByTitle('移除参考图')
    expect(removeButtons.length).toBe(2)
    fireEvent.click(removeButtons[0])
    expect(document.body.querySelectorAll('.cd-ref-img').length).toBe(1)
  })

  it('以 data URL 参考图生成剧照：调用 generatePortraitWithRef 并更新立绘', async () => {
    mockedPortraitWithRef.mockResolvedValue('data:image/png;base64,Q0hBTkc=')
    renderEditor({
      character: makeCharacter({ referenceImages: ['data:image/png;base64,UkVG'] }),
    })
    fireEvent.click(screen.getByTitle('以这张参考图生成剧照（img2img）'))
    await waitFor(() => {
      expect(mockedPortraitWithRef).toHaveBeenCalledTimes(1)
      expect(mockedPortraitWithRef.mock.calls[0][2]).toBe('data:image/png;base64,UkVG')
      expect(document.body.querySelector('.cd-hero-img')?.getAttribute('src')).toBe('data:image/png;base64,Q0hBTkc=')
    })
    // 未走 txt2img 路径
    expect(mockedPortrait).not.toHaveBeenCalled()
  })

  it('以本地路径参考图生成剧照：先经 readFileAsDataURL 读回 data URL 再传给后端', async () => {
    mockedPortraitWithRef.mockResolvedValue('data:image/png;base64,Q0hBTkc=')
    renderEditor({
      character: makeCharacter({ referenceImages: ['C:/x/portraits/c1_ref_0.png'] }),
    })
    fireEvent.click(screen.getByTitle('以这张参考图生成剧照（img2img）'))
    await waitFor(() => {
      expect(readFileAsDataURLMock).toHaveBeenCalledWith('C:/x/portraits/c1_ref_0.png')
      expect(mockedPortraitWithRef.mock.calls[0][2]).toBe('data:image/png;base64,PATHREF')
    })
  })

  it('名称为空时不调用参考图生成', async () => {
    renderEditor({
      character: makeCharacter({ referenceImages: ['data:image/png;base64,UkVG'] }),
    })
    fireEvent.change(screen.getByPlaceholderText('角色名'), { target: { value: '' } })
    fireEvent.click(screen.getByTitle('以这张参考图生成剧照（img2img）'))
    expect(mockedPortraitWithRef).not.toHaveBeenCalled()
  })

  it('参考图生成失败提示错误且不更新立绘', async () => {
    mockedPortraitWithRef.mockRejectedValue(new Error('模型暂不支持参考图生成'))
    renderEditor({
      character: makeCharacter({ referenceImages: ['data:image/png;base64,UkVG'] }),
    })
    fireEvent.click(screen.getByTitle('以这张参考图生成剧照（img2img）'))
    await waitFor(() => {
      expect(screen.getByText(/参考图生成失败/)).toBeTruthy()
    })
    expect(document.body.querySelector('.cd-hero-img')).toBeNull()
  })
})

describe('CharacterLibEditor 生成设定卡（阶段三刀 C）', () => {
  // Dropdown.Button：data-testid 挂在按钮组 wrapper 上，主按钮=组内第一个 button
  const sheetMainBtn = () => screen.getByTestId('gen-character-sheet').querySelector('button')!

  it('点击主按钮：默认三视图 qedit 产物追加进参考图列表；无参考 warning 不触发生成', async () => {
    mockedCharacterSheet.mockClear()
    mockedCharacterSheet.mockResolvedValue('data:image/png;base64,SHEET')
    // 无参考图：warning 不触发
    renderEditor({ character: makeCharacter({ name: '林晚', referenceImages: [] }) })
    fireEvent.click(sheetMainBtn())
    await vi.waitFor(() => expect(screen.getByText(/至少一张参考图/)).toBeTruthy())
    expect(mockedCharacterSheet).not.toHaveBeenCalled()
    // 带参考图：产物追加（success 消息为证），主按钮=triptych 默认模板
    cleanup()
    renderEditor({ character: makeCharacter({ name: '林晚', referenceImages: ['data:image/png;base64,R1'] }) })
    fireEvent.click(sheetMainBtn())
    await vi.waitFor(() => expect(mockedCharacterSheet).toHaveBeenCalledTimes(1))
    expect(mockedCharacterSheet.mock.calls[0][0]?.name).toBe('林晚')
    expect(mockedCharacterSheet.mock.calls[0][1]).toBe('triptych')
    await vi.waitFor(() => expect(screen.getByText(/设定卡已生成并加入参考图/)).toBeTruthy())
  })

  it('模板菜单：箭头展开选「坐姿」，按 sitting variant 调用（v4.401 模板矩阵）', async () => {
    mockedCharacterSheet.mockClear()
    mockedCharacterSheet.mockResolvedValue('data:image/png;base64,SHEET2')
    renderEditor({ character: makeCharacter({ name: '林晚', referenceImages: ['data:image/png;base64,R1'] }) })
    const [, arrow] = Array.from(screen.getByTestId('gen-character-sheet').querySelectorAll('button'))
    fireEvent.mouseEnter(arrow)
    await vi.waitFor(() => {
      expect(document.querySelector('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu')).toBeTruthy()
    })
    const item = Array.from(
      document.querySelectorAll('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu-item'),
    ).find(i => i.textContent?.includes('坐姿'))
    expect(item).toBeTruthy()
    fireEvent.click(item!)
    await vi.waitFor(() => expect(mockedCharacterSheet).toHaveBeenCalledTimes(1))
    expect(mockedCharacterSheet.mock.calls[0][1]).toBe('sitting')
  })

  it('分张连发：正/侧/背三张排队调用并全部加入参考图（v4.402）', async () => {
    mockedCharacterSheet.mockClear()
    mockedCharacterSheet.mockResolvedValue('data:image/png;base64,SPLIT')
    renderEditor({ character: makeCharacter({ name: '林晚', referenceImages: ['data:image/png;base64,R1'] }) })
    const [, arrow] = Array.from(screen.getByTestId('gen-character-sheet').querySelectorAll('button'))
    fireEvent.mouseEnter(arrow)
    await vi.waitFor(() => {
      expect(document.querySelector('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu')).toBeTruthy()
    })
    const split = Array.from(
      document.querySelectorAll('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu-item'),
    ).find(i => i.textContent?.includes('分张'))
    expect(split).toBeTruthy()
    fireEvent.click(split!)
    await vi.waitFor(() => expect(mockedCharacterSheet).toHaveBeenCalledTimes(3))
    expect(mockedCharacterSheet.mock.calls.map(c => c[1])).toEqual(['front', 'side', 'back'])
    await vi.waitFor(() => expect(screen.getByText(/三视图分张已全部生成/)).toBeTruthy())
  })

  it('分张连发：首张失败中止（系统性故障不再连发）；中途失败继续并如实汇总', async () => {
    mockedCharacterSheet.mockClear()
    // 首张失败：只调 1 次，报「分张中止」
    mockedCharacterSheet.mockRejectedValue(new Error('仅 ComfyUI 本地档'))
    renderEditor({ character: makeCharacter({ name: '林晚', referenceImages: ['data:image/png;base64,R1'] }) })
    let [, arrow] = Array.from(screen.getByTestId('gen-character-sheet').querySelectorAll('button'))
    fireEvent.mouseEnter(arrow)
    await vi.waitFor(() => {
      expect(document.querySelector('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu')).toBeTruthy()
    })
    let split = Array.from(
      document.querySelectorAll('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu-item'),
    ).find(i => i.textContent?.includes('分张'))
    fireEvent.click(split!)
    await vi.waitFor(() => expect(mockedCharacterSheet).toHaveBeenCalledTimes(1))
    await vi.waitFor(() => expect(screen.getByText(/分张中止/)).toBeTruthy())
    expect(mockedCharacterSheet).toHaveBeenCalledTimes(1)

    // 中途失败（第 2 张）：3 张都发起，结尾 warning 点名失败视图
    cleanup()
    mockedCharacterSheet.mockReset()
    mockedCharacterSheet.mockResolvedValueOnce('data:image/png;base64,F')
      .mockRejectedValueOnce(new Error('队列超时'))
      .mockResolvedValueOnce('data:image/png;base64,B')
    renderEditor({ character: makeCharacter({ name: '林晚', referenceImages: ['data:image/png;base64,R1'] }) })
    ;[, arrow] = Array.from(screen.getByTestId('gen-character-sheet').querySelectorAll('button'))
    fireEvent.mouseEnter(arrow)
    await vi.waitFor(() => {
      expect(document.querySelector('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu')).toBeTruthy()
    })
    split = Array.from(
      document.querySelectorAll('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu-item'),
    ).find(i => i.textContent?.includes('分张'))
    fireEvent.click(split!)
    await vi.waitFor(() => expect(mockedCharacterSheet).toHaveBeenCalledTimes(3))
    expect(mockedCharacterSheet.mock.calls.map(c => c[1])).toEqual(['front', 'side', 'back'])
    await vi.waitFor(() => expect(screen.getByText(/左侧面失败/)).toBeTruthy())
  })
})

describe('CharacterLibEditor 一致性评分（v4.404）', () => {
  beforeEach(() => {
    Modal.destroyAll()
  })

  it('参考图评分钮：调用 api（透传参考图）并 Modal 呈现分数与不一致点', async () => {
    mockedScore.mockClear()
    mockedScore.mockResolvedValue({ score: 82, summary: '基本一致', issues: ['鼻梁略宽'] })
    renderEditor({ character: makeCharacter({ name: '苏念', referenceImages: ['data:image/png;base64,R1'] }) })
    fireEvent.click(screen.getByTestId('score-ref-0'))
    await vi.waitFor(() => expect(mockedScore).toHaveBeenCalledTimes(1))
    expect(mockedScore.mock.calls[0][1]).toBe('data:image/png;base64,R1')
    await vi.waitFor(() => expect(document.querySelector('.ant-modal-title')?.textContent).toContain('一致性评分：82 分'))
    expect(screen.getByText(/鼻梁略宽/)).toBeTruthy()
    // 高分不提示补参考
    expect(screen.queryByText(/建议补充/)).toBeNull()
  })

  it('低分（<60）Modal 提示补参考；api 失败 message 透出且不复位卡死', async () => {
    mockedScore.mockClear()
    mockedScore.mockResolvedValueOnce({ score: 45, summary: '偏差明显', issues: [] })
    renderEditor({ character: makeCharacter({ name: '苏念', referenceImages: ['data:image/png;base64,R1'] }) })
    fireEvent.click(screen.getByTestId('score-ref-0'))
    await vi.waitFor(() => {
      const titles = document.querySelectorAll('.ant-modal-title')
      expect(titles[titles.length - 1]?.textContent).toContain('一致性评分：45 分')
    })
    expect(screen.getByText(/建议补充/)).toBeTruthy()
    cleanup()
    Modal.destroyAll()
    mockedScore.mockRejectedValueOnce(new Error('视觉模型未启用'))
    renderEditor({ character: makeCharacter({ name: '苏念', referenceImages: ['data:image/png;base64,R1'] }) })
    fireEvent.click(screen.getByTestId('score-ref-0'))
    await vi.waitFor(() => expect(screen.getByText(/一致性评分失败/)).toBeTruthy())
    // 失败后按钮复位可再点
    expect(screen.getByTestId('score-ref-0').hasAttribute('disabled')).toBe(false)
  })
})

describe('CharacterLibEditor 生成进度行（v4.406）', () => {
  it('设定卡生成中：轮询快照显示当前节点与用时；结束后消失', async () => {
    const mockedProgress = vi.mocked(getComfyUITaskProgress)
    mockedProgress.mockClear()
    mockedProgress.mockResolvedValue({ status: 'running', elapsed: 42, percent: -1, node: 'UNETLoader' })
    let resolveSheet!: (v: string) => void
    mockedCharacterSheet.mockClear()
    mockedCharacterSheet.mockImplementation(() => new Promise<string>((r) => (resolveSheet = r)))
    renderEditor({ character: makeCharacter({ name: '苏念', referenceImages: ['data:image/png;base64,R1'] }) })
    const mainBtn = screen.getByTestId('gen-character-sheet').querySelector('button')!
    fireEvent.click(mainBtn)
    await vi.waitFor(() => expect(screen.getByTestId('cd-comfy-progress')).toBeTruthy())
    expect(screen.getByTestId('cd-comfy-progress').textContent).toContain('加载模型')
    expect(screen.getByTestId('cd-comfy-progress').textContent).toContain('42')
    resolveSheet('data:image/png;base64,DONE')
    await vi.waitFor(() => expect(screen.queryByTestId('cd-comfy-progress')).toBeNull())
  })
})

describe('CharacterLibEditor 生成取消（v4.407）', () => {
  it('生成中进度行内取消钮：点击调用全局取消', async () => {
    mockedCancelGen.mockClear()
    mockedCancelGen.mockResolvedValue(true)
    const mockedProgress = vi.mocked(getComfyUITaskProgress)
    mockedProgress.mockClear()
    mockedProgress.mockResolvedValue({ status: 'running', elapsed: 12, percent: -1, node: 'UNETLoader' })
    let resolveSheet!: (v: string) => void
    mockedCharacterSheet.mockClear()
    mockedCharacterSheet.mockImplementation(() => new Promise<string>((r) => (resolveSheet = r)))
    renderEditor({ character: makeCharacter({ name: '苏念', referenceImages: ['data:image/png;base64,R1'] }) })
    const mainBtn = screen.getByTestId('gen-character-sheet').querySelector('button')!
    fireEvent.click(mainBtn)
    await vi.waitFor(() => expect(screen.getByTestId('cd-cancel-gen')).toBeTruthy())
    fireEvent.click(screen.getByTestId('cd-cancel-gen'))
    await vi.waitFor(() => expect(mockedCancelGen).toHaveBeenCalledTimes(1))
    resolveSheet('data:image/png;base64,DONE')
    await vi.waitFor(() => expect(screen.queryByTestId('cd-comfy-progress')).toBeNull())
  })

  it('取消后 promise 以 context canceled 拒绝：显示「已取消生成」而非报错', async () => {
    mockedCharacterSheet.mockClear()
    mockedCharacterSheet.mockRejectedValue(new Error('设定卡生成失败: context canceled'))
    renderEditor({ character: makeCharacter({ name: '苏念', referenceImages: ['data:image/png;base64,R1'] }) })
    const mainBtn = screen.getByTestId('gen-character-sheet').querySelector('button')!
    fireEvent.click(mainBtn)
    await vi.waitFor(() => expect(screen.getByText('已取消生成')).toBeTruthy())
    expect(screen.queryByText(/设定卡生成失败/)).toBeNull()
  })
})
