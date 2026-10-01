// CreateInspector 本书定位（v4.439 成人向档位）行为锁：
// ① 挂载即经 GetProjectInfo 回显当前档位，未就绪（读取完成前）禁用选择器；
// ② 切档即经 UpdateProjectMeta 落盘，标题/题材/文风原样回传不在本面板改写；
// ③ 落盘失败回退选项并如实提示（不静默假成功）；
// ④ v4.440.1 切书边界收口（v4.429「切书重拉」同族）：NovelPage 五 tab 常驻挂载、
//    切书不重挂本面板——projectPath 变化必须重拉元信息；迟到的旧书响应不得
//    覆盖新书档位（代际守卫）。
// mock 口径照 SkillModal.test.tsx（wailsApp 半桩）+ NewCharactersModal.test.tsx（gaea bridge 桩）。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import CreateInspector from './CreateInspector'
import { wailsApp } from '../../../lib/wailsApp'
import { useAppStore } from '../../../stores/appStore'

vi.mock('../../../lib/wailsApp', () => ({
  wailsApp: vi.fn(),
}))
vi.mock('../../../gaea/lib/bridge', () => ({
  app: { ListSkills: vi.fn(async () => []) },
}))

const mockedWailsApp = vi.mocked(wailsApp)

function renderInspector() {
  return render(
    <CreateInspector
      setting=""
      onRefreshSetting={vi.fn()}
      selectedSkill={undefined}
      onSelectSkill={vi.fn()}
      minWords={5000}
      onMinWordsChange={vi.fn()}
      temperature={0.8}
      onTemperatureChange={vi.fn()}
      directPlot=""
      onDirectPlotChange={vi.fn()}
      onDirectGenerate={vi.fn()}
      onOpenWizard={vi.fn()}
      prevChapterHint={1}
      stats={null}
      chapterCount={0}
    />,
  )
}

/** 等选择器就绪（提示语切换为正式文案=GetProjectInfo 已按当前书落地）。 */
async function waitForReady() {
  await screen.findByText(/亲密戏写作纪律/)
}

function pickOption(label: string) {
  fireEvent.mouseDown(document.querySelector('.ant-select-selector')!)
  fireEvent.click(screen.getByText(label))
}

describe('CreateInspector 本书定位（成人向档位）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useAppStore.setState({ projectPath: 'C:/books/A' })
  })

  it('挂载回显项目当前档位；未改不写盘', async () => {
    mockedWailsApp.mockReturnValue({
      GetProjectInfo: vi.fn(async () => ({ title: '书', genre: '都市', style: '甜宠', mature: 'explicit' })),
      UpdateProjectMeta: vi.fn(async () => undefined),
    } as unknown as ReturnType<typeof wailsApp>)
    renderInspector()
    await waitForReady()
    const update = mockedWailsApp().UpdateProjectMeta as ReturnType<typeof vi.fn>
    expect(update).not.toHaveBeenCalled()
  })

  it('切档即落盘：标题/题材/文风原样回传 + 新档位', async () => {
    const update = vi.fn(async () => undefined)
    mockedWailsApp.mockReturnValue({
      GetProjectInfo: vi.fn(async () => ({ title: '书名', genre: '言情', style: '正剧', mature: '' })),
      UpdateProjectMeta: update,
    } as unknown as ReturnType<typeof wailsApp>)
    renderInspector()
    await waitForReady()
    pickOption('成人向 · 直白（亲密戏正面直书）')
    await waitFor(() => {
      expect(update).toHaveBeenCalledWith('书名', '言情', '正剧', 'explicit')
    })
  })

  it('落盘失败：回退选项 + 如实提示，不假成功', async () => {
    const update = vi.fn(async () => { throw new Error('档位非法: r18') })
    mockedWailsApp.mockReturnValue({
      GetProjectInfo: vi.fn(async () => ({ title: '书名', genre: '', style: '', mature: '' })),
      UpdateProjectMeta: update,
    } as unknown as ReturnType<typeof wailsApp>)
    renderInspector()
    await waitForReady()
    pickOption('成人向 · 含蓄（张力与留白）')
    const note = await screen.findByText(/保存本书定位失败/)
    expect(note).toBeTruthy()
    // 回退后 Select 显示值回到「非成人向」（下拉未关会双元素，按 selection-item 定位）
    await waitFor(() => {
      const sel = document.querySelector('.ant-select-selection-item')
      expect(sel?.textContent).toBe('非成人向')
    })
  })

  it('切书重拉：projectPath 变化后按新书元信息回显与回传（常驻挂载不重挂）', async () => {
    const update = vi.fn(async () => undefined)
    const infoByPath: Record<string, Record<string, string>> = {
      'C:/books/A': { title: '甲书', genre: '玄幻', style: '热血', mature: 'explicit' },
      'C:/books/B': { title: '乙书', genre: '言情', style: '甜宠', mature: 'sensual' },
    }
    mockedWailsApp.mockReturnValue({
      GetProjectInfo: vi.fn(async () => infoByPath[useAppStore.getState().projectPath ?? '']),
      UpdateProjectMeta: update,
    } as unknown as ReturnType<typeof wailsApp>)
    renderInspector()
    await waitForReady()
    pickOption('非成人向')
    await waitFor(() => {
      expect(update).toHaveBeenCalledWith('甲书', '玄幻', '热血', '')
    })
    update.mockClear()

    // 切书（面板不重挂）：须按乙书回显；写路径回传乙书元信息
    useAppStore.setState({ projectPath: 'C:/books/B' })
    await waitForReady()
    pickOption('非成人向')
    await waitFor(() => {
      expect(update).toHaveBeenCalledWith('乙书', '言情', '甜宠', '')
    })
  })

  it('迟到代际：切书后旧书迟到的 GetProjectInfo 不得覆盖新书档位', async () => {
    let resolveA: (v: unknown) => void = () => {}
    const infoA = new Promise((resolve) => { resolveA = resolve })
    const update = vi.fn(async () => undefined)
    mockedWailsApp.mockReturnValue({
      GetProjectInfo: vi.fn((/* 按调用时刻的书返回 */) => {
        if (useAppStore.getState().projectPath === 'C:/books/A') return infoA
        return Promise.resolve({ title: '乙书', genre: '言情', style: '甜宠', mature: 'sensual' })
      }),
      UpdateProjectMeta: update,
    } as unknown as ReturnType<typeof wailsApp>)
    renderInspector()
    // 甲书的响应挂起 → 切书 → 乙书就绪
    useAppStore.setState({ projectPath: 'C:/books/B' })
    await waitForReady()
    // 乙书就绪后，甲书迟到的响应才回来（mature=explicit）——必须被代际守卫丢弃
    resolveA({ title: '甲书', genre: '玄幻', style: '热血', mature: 'explicit' })
    await waitFor(() => {
      const sel = document.querySelector('.ant-select-selection-item')
      expect(sel?.textContent).toBe('成人向 · 含蓄（张力与留白）')
    })
    // 迟到响应不得把写路径的 metaRef 拖回甲书：切档回传的必须是乙书元信息
    pickOption('非成人向')
    await waitFor(() => {
      expect(update).toHaveBeenCalledWith('乙书', '言情', '甜宠', '')
    })
  })
})
