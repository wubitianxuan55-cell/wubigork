// CreateInspector 本书定位（v4.439 成人向档位）行为锁：
// ① 挂载即经 GetProjectInfo 回显当前档位；② 切档即经 UpdateProjectMeta 落盘，
// 标题/题材/文风原样回传不在本面板改写；③ 落盘失败回退选项并如实提示（不静默假成功）。
// mock 口径照 SkillModal.test.tsx（wailsApp 半桩）+ NewCharactersModal.test.tsx（gaea bridge 桩）。
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import CreateInspector from './CreateInspector'
import { wailsApp } from '../../../lib/wailsApp'

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

describe('CreateInspector 本书定位（成人向档位）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('挂载回显项目当前档位（mature 缺省视为非成人向）', async () => {
    mockedWailsApp.mockReturnValue({
      GetProjectInfo: vi.fn(async () => ({ title: '书', genre: '都市', style: '甜宠', mature: 'explicit' })),
      UpdateProjectMeta: vi.fn(async () => undefined),
    } as unknown as ReturnType<typeof wailsApp>)
    renderInspector()
    // explicit 档回显为选中值（Select 值不在 DOM 文本里，经回调侧验证）
    const update = mockedWailsApp().UpdateProjectMeta as ReturnType<typeof vi.fn>
    fireEvent.mouseDown(await screen.findByText('本书定位'))
    // 未改不写盘
    expect(update).not.toHaveBeenCalled()
  })

  it('切档即落盘：标题/题材/文风原样回传 + 新档位', async () => {
    const update = vi.fn(async () => undefined)
    mockedWailsApp.mockReturnValue({
      GetProjectInfo: vi.fn(async () => ({ title: '书名', genre: '言情', style: '正剧', mature: '' })),
      UpdateProjectMeta: update,
    } as unknown as ReturnType<typeof wailsApp>)
    renderInspector()
    // antd Select：点开下拉选「直白」
    fireEvent.mouseDown(await screen.findByText('非成人向'))
    const opt = await screen.findByText('成人向 · 直白（亲密戏正面直书）')
    fireEvent.click(opt)
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
    fireEvent.mouseDown(await screen.findByText('非成人向'))
    fireEvent.click(await screen.findByText('成人向 · 含蓄（张力与留白）'))
    const note = await screen.findByText(/保存本书定位失败/)
    expect(note).toBeTruthy()
    // 回退后 Select 显示值回到「非成人向」（下拉未关会双元素，按 selection-item 定位）
    await waitFor(() => {
      const sel = document.querySelector('.ant-select-selection-item')
      expect(sel?.textContent).toBe('非成人向')
    })
  })
})
