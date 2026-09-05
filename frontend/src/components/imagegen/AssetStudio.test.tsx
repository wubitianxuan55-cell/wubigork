// AssetStudio（图像域 T1 画室「创作资产」面板）行为用例：
// 角色槽（CharacterList 分页/总数/选中摘要/带入参考槽）、模板槽（绘梦既有模板
// 数据源复用 + 搜索 + 套用回填）、近期作品（ImageHubAssets 按空间前 12 张，
// 点击跳素材库）。三槽数据源全部 mock 在 api/image 层（零新绑定口径）。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { LocaleProvider } from '../../gaea/lib/i18n'
import { AssetStudio } from './AssetStudio'
import { chatCharacters, imageHubAssets, readFileAsDataURL, type ChatCharacterView } from '../../api/image'
import type { Template } from '../../data/imageTemplates'

vi.mock('../../api/image', () => ({
  chatCharacters: vi.fn(),
  imageHubAssets: vi.fn(),
  readFileAsDataURL: vi.fn(),
}))

const chatMock = vi.mocked(chatCharacters)
const assetsMock = vi.mocked(imageHubAssets)
const readFileMock = vi.mocked(readFileAsDataURL)

function char(over: Partial<ChatCharacterView> = {}): ChatCharacterView {
  return {
    id: 'c1', name: '林晚', kind: 'custom', tags: ['女主角', '清冷'],
    portraitUrl: 'data:image/png;base64,P1', roleType: '女主',
    personality: '清冷聪慧，外冷内热', background: '前朝遗孤，隐姓埋名', appearance: '',
    ...over,
  }
}

function tpl(label: string, over: Partial<Template> = {}): Template {
  return { label, description: `${label} 用途`, prompt: `${label} 的提示词`, size: '1:1', tags: ['风格'], ...over }
}

interface RenderOpts {
  templates?: Template[]
  customTemplates?: Template[]
}

function renderStudio(opts: RenderOpts = {}) {
  const onClose = vi.fn()
  const onOpenLibrary = vi.fn()
  const onApplyTemplate = vi.fn()
  const onApplyCharacter = vi.fn()
  render(
    <LocaleProvider>
      <AssetStudio
        onClose={onClose}
        onOpenLibrary={onOpenLibrary}
        templates={opts.templates ?? [tpl('通用高质量'), tpl('赛博霓虹')]}
        customTemplates={opts.customTemplates ?? [{ id: 'custom_1', label: '我的模板', prompt: '自定义提示词' } as Template]}
        onApplyTemplate={onApplyTemplate}
        onApplyCharacter={onApplyCharacter}
      />
    </LocaleProvider>,
  )
  return { onClose, onOpenLibrary, onApplyTemplate, onApplyCharacter }
}

beforeEach(() => {
  localStorage.setItem('gaea-lang', 'zh')
  chatMock.mockReset().mockResolvedValue({ items: [char()], total: 1 })
  assetsMock.mockReset().mockResolvedValue([])
  readFileMock.mockReset().mockResolvedValue('data:image/png;base64,AAA')
})

describe('AssetStudio 画室「创作资产」面板', () => {
  it('角色槽：分页拉取可聊天角色，展示总数与角色卡', async () => {
    renderStudio()
    await waitFor(() => expect(chatMock).toHaveBeenCalledWith(1, 12))
    expect(screen.getByText('共 1 个可聊天角色')).toBeTruthy()
    expect(screen.getByText('林晚')).toBeTruthy()
  })

  it('角色槽空态：无可聊天角色时诚实提示', async () => {
    chatMock.mockResolvedValue({ items: [], total: 0 })
    renderStudio()
    await waitFor(() => expect(screen.getByText('角色库暂无可聊天角色。')).toBeTruthy())
  })

  it('角色槽分页：「加载更多」拉下一页追加', async () => {
    chatMock.mockImplementation(async (page: number) => ({
      items: [char({ id: `c${page}`, name: `角色${page}` })],
      total: 25,
    }))
    renderStudio()
    await waitFor(() => expect(chatMock).toHaveBeenCalledWith(1, 12))
    fireEvent.click(screen.getByText('加载更多'))
    await waitFor(() => expect(chatMock).toHaveBeenCalledWith(2, 12))
    await waitFor(() => expect(screen.getByText('角色2')).toBeTruthy())
  })

  it('选中角色带出数据摘要（立绘/性格/背景/标签）', async () => {
    renderStudio()
    await waitFor(() => expect(screen.getByText('林晚')).toBeTruthy())
    fireEvent.click(screen.getByText('林晚'))
    await waitFor(() => {
      const text = document.body.textContent || ''
      expect(text).toContain('角色数据摘要')
      expect(text).toContain('性格: 清冷聪慧，外冷内热')
      expect(text).toContain('背景: 前朝遗孤，隐姓埋名')
      expect(text).toContain('女主角')
    })
  })

  it('「带入参考槽」回调角色 id（走页面级 applyRefCharacter 既有回填路径）', async () => {
    const { onApplyCharacter } = renderStudio()
    await waitFor(() => expect(screen.getByText('林晚')).toBeTruthy())
    fireEvent.click(screen.getByText('林晚'))
    fireEvent.click(screen.getByText('带入参考槽'))
    expect(onApplyCharacter).toHaveBeenCalledWith('c1')
  })

  it('模板槽：复用绘梦既有模板库（内置 + 自定义）展示，点击套用回传模板', async () => {
    const { onApplyTemplate } = renderStudio()
    await waitFor(() => expect(screen.getByText('我的模板')).toBeTruthy())
    expect(screen.getByText('通用高质量')).toBeTruthy()
    expect(screen.getByText('赛博霓虹')).toBeTruthy()
    fireEvent.click(screen.getByText('赛博霓虹'))
    expect(onApplyTemplate).toHaveBeenCalledWith(expect.objectContaining({ label: '赛博霓虹' }))
  })

  it('模板槽搜索：按名称过滤并重置懒加载页', async () => {
    renderStudio()
    await waitFor(() => expect(screen.getByText('我的模板')).toBeTruthy())
    const input = screen.getByPlaceholderText('搜索模板名称 / 提示词') as HTMLInputElement
    fireEvent.change(input, { target: { value: '我的' } })
    await waitFor(() => expect(screen.getByText('我的模板')).toBeTruthy())
    expect(screen.queryByText('通用高质量')).toBeNull()
    expect(screen.queryByText('赛博霓虹')).toBeNull()
  })

  it('近期作品：读当前空间前 12 张，缩略懒读；点击跳素材库', async () => {
    const { onOpenLibrary } = renderStudio()
    await waitFor(() => expect(assetsMock).toHaveBeenCalledWith('work', '', 12))
    assetsMock.mockResolvedValue([
      { id: 'ih-1', kind: 'image', path: 'C:/ws/.gaea/uploads/a.png', space: 'work', source_board: 'imagegen', capability: 'media.generate', model: 'krea2', created_at: '2026-09-05T10:00:00Z' },
    ])
    // 空间切换触发重读（Segmented 选项点击）。
    const playOpt = screen.getAllByText('创作空间').find((el) => !!el.closest('.ant-segmented-item'))
    fireEvent.click(playOpt as HTMLElement)
    await waitFor(() => expect(assetsMock).toHaveBeenCalledWith('play', '', 12))
    await waitFor(() => expect(screen.getByTitle('C:/ws/.gaea/uploads/a.png')).toBeTruthy())
    fireEvent.click(screen.getByTitle('C:/ws/.gaea/uploads/a.png'))
    expect(onOpenLibrary).toHaveBeenCalledTimes(1)
  })

  it('近期作品空态：沿用素材库空态文案（自动溯源入库）', async () => {
    renderStudio()
    await waitFor(() =>
      expect(screen.getByText('该空间还没有登记素材——生成的图片会自动溯源入库。')).toBeTruthy())
  })

  it('返回生成台：触发 onClose 出口', async () => {
    const { onClose } = renderStudio()
    await waitFor(() => expect(chatMock).toHaveBeenCalled())
    fireEvent.click(screen.getByText('返回生成台'))
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
