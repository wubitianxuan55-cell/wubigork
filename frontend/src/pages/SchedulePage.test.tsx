/**
 * SchedulePage.test.tsx — 页头多工程切换器/管理面板轻量用例（v4.139 #15）
 *
 * 覆盖：切换器渲染（未归档过滤、当前项显示、归档项不出现）、onChange 调
 * store.openProject（切换+冲刷+重水合属 store 侧职责，此处只断言回调）、
 * 仅一项工程时收窄为纯显示（disabled，老用户零感知）、projects 为空（旧 store
 * 形态）整体隐藏、管理面板（打开即 refreshProjects、当前工程行内改名走既有
 * renameProject、归档/删除回调、新建 createProject trim）、状态栏指示器 title
 * 跟随 store.currentPath（不再硬编码常量）。
 * store 契约成员（projects/currentPath/openProject/...）由 store 侧提供；用例以
 * useScheduleStore.setState 预置（zustand 运行时合并任意键），动作用 vi.fn 替身。
 * 左栏 ChatPane 依赖会话 store 全家桶，mock 掉保持用例聚焦（同 ResourcePanel.test）。
 */
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SchedulePage from './SchedulePage'
import { useScheduleStore } from '../schedule/store'
import { menuForHit } from '../schedule/hitMenu'
import { hitRegionOfElement } from '../schedule/hitRegion'

vi.mock('../schedule/ChatPane', () => ({ ScheduleChatPane: () => <div data-testid="mock-chat-pane" /> }))

// v4.144 导入为新工程：只替换 MPP 解析（分发用例不触真绑定）；其余 api 成员原样保留
vi.mock('../schedule/api', async (importOriginal) => {
  const orig = await importOriginal<typeof import('../schedule/api')>()
  return {
    ...orig,
    importScheduleMpp: vi.fn(async () => ({
      name: '', startDate: '2026-09-07',
      tasks: [{ id: 'm1', name: 'MPP任务', duration: 3, level: 1, progress: 0 }],
      links: [],
    })),
  }
})

type StorePatch = Record<string, unknown>
const seed = (patch: StorePatch) =>
  useScheduleStore.setState(patch as unknown as Parameters<typeof useScheduleStore.setState>[0])

const DEFAULT_REL = '进度计划/当前计划.gsched.json'
const B2_REL = '进度计划/办公楼二期.gsched.json'
const OLD_REL = '进度计划/旧计划.gsched.json'

const projectsOf = [
  { rel: DEFAULT_REL, name: '当前计划', archived: false, duration: 9, taskCount: 3, ok: true, updatedAt: '2026-09-01T08:00:00Z' },
  { rel: B2_REL, name: '办公楼二期', archived: false, duration: 30, taskCount: 12, ok: true, updatedAt: '2026-09-02T08:00:00Z' },
  { rel: OLD_REL, name: '旧计划', archived: true, duration: 0, taskCount: 1, ok: false, updatedAt: '2026-08-01T08:00:00Z' },
]

let openProjectSpy: ReturnType<typeof vi.fn>
let createProjectSpy: ReturnType<typeof vi.fn>
let archiveProjectSpy: ReturnType<typeof vi.fn>
let deleteProjectSpy: ReturnType<typeof vi.fn>
let refreshProjectsSpy: ReturnType<typeof vi.fn>
let copyProjectSpy: ReturnType<typeof vi.fn>

beforeEach(() => {
  localStorage.removeItem('gaea.schedule.v1')
  openProjectSpy = vi.fn(async (_rel: string) => {})
  createProjectSpy = vi.fn(async (_name: string) => {})
  archiveProjectSpy = vi.fn(async (_rel: string, _archived: boolean) => {})
  deleteProjectSpy = vi.fn(async (_rel: string) => {})
  refreshProjectsSpy = vi.fn(async () => {})
  copyProjectSpy = vi.fn(async (_rel: string, _name: string) => true)
  seed({
    project: { name: '当前计划', startDate: '2026-09-07', tasks: [], links: [] },
    view: 'gantt',
    selectedId: null,
    hydrated: true,
    sync: 'saved',
    savedAt: '08:00',
    syncError: null,
    past: [],
    future: [],
    projects: projectsOf,
    currentPath: DEFAULT_REL,
    openProject: openProjectSpy,
    createProject: createProjectSpy,
    archiveProject: archiveProjectSpy,
    deleteProject: deleteProjectSpy,
    refreshProjects: refreshProjectsSpy,
    copyProject: copyProjectSpy,
  })
})

/** 打开 antd Select 下拉并点选选项文字（文字在 .ant-select-item-option-content 直下） */
async function pickOption(trigger: HTMLElement, label: string) {
  fireEvent.mouseDown(trigger.querySelector('.ant-select-selector')!)
  const opt = await screen.findByText(label, { selector: '.ant-select-item-option-content' })
  fireEvent.click(opt.closest('.ant-select-item-option')!)
}

async function openManagePanel(): Promise<HTMLElement> {
  fireEvent.click(screen.getByTestId('sched-projects-manage'))
  return screen.findByTestId('sched-projects-manage-panel')
}

describe('SchedulePage 页头工程切换器/管理面板（v4.139 #15）', () => {
  it('切换器：列未归档工程（归档项不出现），显示当前工程名', async () => {
    render(<SchedulePage />)
    fireEvent.click(screen.getByTestId('sched-ribbon-tab-file'))
    const sel = screen.getByTestId('sched-project-select')
    expect(sel.textContent).toContain('当前计划')
    fireEvent.mouseDown(sel.querySelector('.ant-select-selector')!)
    expect(await screen.findByText('办公楼二期', { selector: '.ant-select-item-option-content' })).toBeTruthy()
    expect(screen.queryByText('旧计划', { selector: '.ant-select-item-option-content' })).toBeNull()
  })

  it('选择工程：onChange 调 openProject(rel)（切换+重水合由 store 动作负责）', async () => {
    render(<SchedulePage />)
    fireEvent.click(screen.getByTestId('sched-ribbon-tab-file'))
    await pickOption(screen.getByTestId('sched-project-select'), '办公楼二期')
    await waitFor(() => expect(openProjectSpy).toHaveBeenCalledTimes(1))
    expect(openProjectSpy).toHaveBeenCalledWith(B2_REL)
  })

  it('仅一项工程：切换器收窄为纯显示（disabled，不占交互成本）', () => {
    seed({ projects: [projectsOf[0]], currentPath: DEFAULT_REL })
    render(<SchedulePage />)
    fireEvent.click(screen.getByTestId('sched-ribbon-tab-file'))
    const sel = screen.getByTestId('sched-project-select')
    expect(sel.className).toContain('ant-select-disabled')
  })

  it('projects 为空（旧 store 形态/未加载）：切换器整体隐藏（零感知）', () => {
    seed({ projects: [] })
    render(<SchedulePage />)
    fireEvent.click(screen.getByTestId('sched-ribbon-tab-file'))
    expect(screen.queryByTestId('sched-project-select')).toBeNull()
  })

  it('状态栏同步指示器 title 跟随 store.currentPath（不再硬编码缺省路径）', () => {
    seed({ currentPath: B2_REL })
    render(<SchedulePage />)
    expect(screen.getByTitle(B2_REL)).toBeTruthy()
  })

  it('管理面板：打开即 refreshProjects；列出各工程（含归档）', async () => {
    render(<SchedulePage />)
    const panel = await openManagePanel()
    expect(panel.textContent).toContain('办公楼二期')
    expect(panel.textContent).toContain('旧计划') // 归档项在管理列表可见
    await waitFor(() => expect(refreshProjectsSpy).toHaveBeenCalled())
  })

  it('管理面板：当前工程行内改名走既有 renameProject（改文件内 name）', async () => {
    render(<SchedulePage />)
    const panel = await openManagePanel()
    const renameInput = panel.querySelector('input')! // 首行=当前工程 → 行内改名 Input
    fireEvent.change(renameInput, { target: { value: '改名后的工程' } })
    expect(useScheduleStore.getState().project.name).toBe('改名后的工程')
  })

  it('管理面板：归档/反归档调 archiveProject(rel, !archived)', async () => {
    render(<SchedulePage />)
    const panel = await openManagePanel()
    const btns = panel.querySelectorAll('[data-testid="sched-project-archive"]')
    expect(btns.length).toBe(projectsOf.length)
    fireEvent.click(btns[1]) // 办公楼二期（未归档 → 请求归档）
    await waitFor(() => expect(archiveProjectSpy).toHaveBeenCalledWith(B2_REL, true))
    fireEvent.click(btns[2]) // 旧计划（已归档 → 请求反归档）
    await waitFor(() => expect(archiveProjectSpy).toHaveBeenCalledWith(OLD_REL, false))
  })

  it('管理面板：删除经 Popconfirm（删除后不可恢复）确认后调 deleteProject(rel)', async () => {
    render(<SchedulePage />)
    const panel = await openManagePanel()
    fireEvent.click(panel.querySelectorAll('[data-testid="sched-project-delete"]')[1])
    // antd 缺省 locale 确认键为 OK（同 BaselinesPanel/TemplatesPanel 测试范式）
    fireEvent.click(await screen.findByRole('button', { name: /OK|确定/ }))
    await waitFor(() => expect(deleteProjectSpy).toHaveBeenCalledWith(B2_REL))
  })

  it('管理面板：新建工程输入 trim 后调 createProject，输入框清空', async () => {
    render(<SchedulePage />)
    await openManagePanel() // 等面板挂载后再取新建输入
    const createInput = screen.getByTestId('sched-project-create-input') as HTMLInputElement // antd Input：testid 落在原生 input 上
    const createBtn = screen.getByTestId('sched-project-create')
    expect(createBtn.hasAttribute('disabled')).toBe(true) // 空名禁用
    fireEvent.change(createInput, { target: { value: '  新工程  ' } })
    expect(createBtn.hasAttribute('disabled')).toBe(false)
    fireEvent.click(createBtn)
    await waitFor(() => expect(createProjectSpy).toHaveBeenCalledWith('新工程'))
    expect(createInput.value).toBe('')
  })
})

/** 打开 antd Dropdown 菜单（fireEvent 点击菜单栏按钮），返回当前打开菜单项文本 */
async function openMenu(testid: string): Promise<string[]> {
  const trigger = screen.getByTestId(testid)
  await act(async () => { fireEvent.click(trigger) })
  const menu = await waitFor(() => {
    const el = Array.from(document.querySelectorAll('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu')).at(-1)
    if (!el) throw new Error('菜单未打开：' + testid)
    return el
  })
  return Array.from(menu.querySelectorAll('.ant-dropdown-menu-item')).map(
    (i) => i.textContent?.trim() + (i.classList.contains('ant-dropdown-menu-item-disabled') ? '(dis)' : ''),
  )
}


describe('SchedulePage 命令区三行（v4.140：菜单栏/工具栏/信息条）', () => {
  it('ZPert 式布局：命令台/文档条/工作区/右舷检查齐备且可开关', () => {
    render(<SchedulePage />)
    expect(screen.getByTestId('sched-command-deck')).toBeTruthy()
    expect(screen.getByRole('tab', { name: '计划' }).getAttribute('aria-selected')).toBe('true')
    expect(screen.getByTestId('sched-docbar').textContent).toContain('当前计划')
    expect(screen.getByTestId('sched-workspace')).toBeTruthy()
    expect(screen.getByTestId('sched-dock')).toBeTruthy()
    fireEvent.click(screen.getByTestId('sched-project-settings-btn'))
    expect(screen.getByTestId('sched-project-settings').textContent).toContain('工程名称')
    expect(screen.getByTestId('sched-project-settings').textContent).toContain('开工日期')
    fireEvent.click(screen.getByTestId('sched-project-settings-btn'))
    fireEvent.click(screen.getByTestId('sched-ribbon-tab-resource'))
    expect(screen.getByText('资源工作表')).toBeTruthy()
    fireEvent.click(screen.getByTestId('sched-ribbon-tab-plan'))
    fireEvent.click(screen.getByTestId('sched-dock-toggle'))
    expect(screen.queryByTestId('sched-dock')).toBeNull()
    fireEvent.click(screen.getByTestId('sched-dock-toggle'))
    expect(screen.getByTestId('sched-dock')).toBeTruthy()
  })

  it('Ctrl+K 打开命令面板，搜索结果经统一派发切换视图', async () => {
    render(<SchedulePage />)
    fireEvent.keyDown(window, { key: 'k', ctrlKey: true })
    const search = await screen.findByTestId('sched-command-search')
    fireEvent.change(search, { target: { value: '单代号' } })
    fireEvent.keyDown(search, { key: 'Enter' })
    await waitFor(() => expect(useScheduleStore.getState().view).toBe('pdm'))
  })

  it('功能区页签命令齐全：文件/视图菜单可切换，计划页保留 ZPert 六面板与占位按钮', async () => {
    render(<SchedulePage />)
    fireEvent.click(screen.getByTestId('sched-ribbon-tab-file'))
    const edit = await openMenu('sched-menu-edit')
    expect(edit.some((t) => t.startsWith('撤销'))).toBe(true)
    expect(edit.some((t) => t.startsWith('重做'))).toBe(true)
    expect(edit).toContain('删除选中(dis)')
    expect(edit).toContain('清空全部任务与搭接')
    fireEvent.click(screen.getByTestId('sched-ribbon-tab-view'))
    const view = await openMenu('sched-menu-view')
    expect(view).toEqual(['横道图', '单代号网络图', '双代号网络图', '资源使用', '质量体检', '工期模拟', '挣值分析'])
    fireEvent.click(screen.getByTestId('sched-ribbon-tab-plan'))
    const planRibbon = document.querySelector('.sched-toolbar')!
    for (const label of ['视图', '编辑', '格式', '大纲', '插入', '高级', '时标网络图', '横道图', '逻辑网络图', '单代号网络图', '复制', '字体', '折叠', '工作', '关系', '里程碑', '智能布图', '按纸张', '预警', '检查']) {
      expect(planRibbon.textContent).toContain(label)
    }
    const timescaled = Array.from(planRibbon.querySelectorAll('button')).find((b) => b.textContent === '时标网络图') as HTMLButtonElement
    expect(timescaled.disabled).toBe(false)
    fireEvent.click(timescaled)
    expect(useScheduleStore.getState().view).toBe('aoa')
    expect(useScheduleStore.getState().project.networkStyle?.mode).toBe('timescaled')
  })

  it('图面样式：四显示开关经 networkStyle 单一入口落库，并随工程状态回显', async () => {
    render(<SchedulePage />)
    fireEvent.click(screen.getByTestId('sched-ribbon-tab-plan'))
    fireEvent.click(screen.getByTestId('sched-network-style-trigger'))
    const controls = await screen.findByTestId('sched-network-style-controls')
    for (const label of ['节点时间', '工期', '六参数', '起止时间']) {
      expect(controls.textContent).toContain(label)
    }
    fireEvent.click(screen.getByTestId('sched-network-toggle-showSixPara'))
    await waitFor(() => expect(useScheduleStore.getState().project.networkStyle?.showSixPara).toBe(true))
    expect(screen.getByTestId('sched-network-toggle-showSixPara').getAttribute('aria-checked')).toBe('true')
  })

  it('列筛选入口：功能区说明 + 一键清除，命令面板同源', async () => {
    render(<SchedulePage />)
    fireEvent.click(screen.getByTestId('sched-ribbon-tab-plan'))
    const entry = Array.from(document.querySelectorAll<HTMLButtonElement>('.sched-toolbar button'))
      .find((button) => button.textContent === '列筛选')!
    fireEvent.click(entry)
    const clear = await screen.findByTestId('sched-colfilter-clear')
    expect(document.body.textContent).toContain('文本包含 / 数值比较 / 日期区间 / 枚举')
    fireEvent.click(clear)
    expect(useScheduleStore.getState().view).toBe('gantt')
  })

  it('五图模式首批接通：时标逻辑图保留时间轴，横道网络图回到关系叠加的横道', () => {    render(<SchedulePage />)
    fireEvent.click(screen.getByTestId('sched-ribbon-tab-plan'))
    const findCommand = (label: string) => Array.from(document.querySelectorAll<HTMLButtonElement>('.sched-toolbar button'))
      .find((button) => button.textContent === label)!
    fireEvent.click(findCommand('时标逻辑图'))
    expect(useScheduleStore.getState().view).toBe('aoa')
    expect(useScheduleStore.getState().project.networkStyle?.mode).toBe('timescaledLogic')
    fireEvent.click(findCommand('横道网络图'))
    expect(useScheduleStore.getState().view).toBe('gantt')
    expect(useScheduleStore.getState().project.networkStyle?.mode).toBe('ganttNetwork')
  })

  it('导入/导出下拉：新工程主入口+三路替换当前显式标注；导出含图面', async () => {
    seed({
      project: {
        name: '当前计划', startDate: '2026-09-07',
        tasks: [{ id: 'A', name: '挖土', duration: 2, level: 1, progress: 0 }],
        links: [],
      },
      selectedId: 'A',
    })
    render(<SchedulePage />)
    fireEvent.click(screen.getByTestId('sched-ribbon-tab-file'))
    const imp = await openMenu('sched-import-btn')
    expect(imp).toEqual(['导入为新工程…', '导入 XML 替换当前工程…', '导入 Excel 替换当前工程…', '导入 MPP 替换当前工程…'])
    const exp = await openMenu('sched-export-menu-btn')
    expect(exp).toEqual(['导出 MS Project XML', '导出 Excel', '导出图面（PNG / PDF / 打印）…'])
  })
})

describe('SchedulePage 导出预览（v4.141：所见即所得）', () => {
  it('打开导出图面弹窗后生成预览 SVG（注入 viewBox，可等比缩放）', async () => {
    seed({
      project: {
        name: '当前计划', startDate: '2026-09-07',
        tasks: [
          { id: 'A', name: '挖土', duration: 2, level: 1, progress: 0 },
          { id: 'B', name: '垫层', duration: 2, level: 1, progress: 0 },
        ],
        links: [{ from: 'A', to: 'B', type: 'FS', lag: 0 }],
      },
      selectedId: null,
    })
    render(<SchedulePage />)
    fireEvent.click(screen.getByTestId('sched-ribbon-tab-file'))
    await act(async () => { fireEvent.click(screen.getByTestId('sched-menu-file')) })
    const item = await waitFor(() => {
      const el = Array.from(document.querySelectorAll('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu-item'))
        .find((i) => i.textContent?.includes('导出图面'))
      if (!el) throw new Error('菜单未开')
      return el
    })
    await act(async () => { fireEvent.click(item) })
    const preview = await waitFor(() => {
      const svg = document.querySelector('[data-testid="sched-export-preview"] svg')
      if (!svg) throw new Error('预览未生成')
      return svg
    }, { timeout: 3000 })
    expect(preview.getAttribute('viewBox')).toMatch(/^0 0 [\d.]+ [\d.]+$/)
  })

  it('W6 按纸张布图：勾选后预览换成多页纸堆，并给出页数/缩放读数', async () => {
    seed({
      project: {
        name: '当前计划', startDate: '2026-09-07',
        tasks: [
          { id: 'A', name: '挖土', duration: 2, level: 1, progress: 0 },
          { id: 'B', name: '垫层', duration: 2, level: 1, progress: 0 },
        ],
        links: [{ from: 'A', to: 'B', type: 'FS', lag: 0 }],
      },
      selectedId: null,
    })
    render(<SchedulePage />)
    fireEvent.click(screen.getByTestId('sched-ribbon-tab-file'))
    await act(async () => { fireEvent.click(screen.getByTestId('sched-menu-file')) })
    const item = await waitFor(() => {
      const el = Array.from(document.querySelectorAll('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu-item'))
        .find((i) => i.textContent?.includes('导出图面'))
      if (!el) throw new Error('菜单未开')
      return el
    })
    await act(async () => { fireEvent.click(item) })
    await screen.findByTestId('sched-export-modal')
    fireEvent.click(screen.getByTestId('sched-export-paper'))
    const info = await waitFor(() => screen.getByTestId('sched-export-paper-info'), { timeout: 3000 })
    expect(info.textContent).toMatch(/共 \d+ 页 · 缩放 \d+%/)
    const stack = document.querySelector('[data-testid="sched-export-preview"] svg.sched-paper-stack')
    expect(stack).not.toBeNull()
    // 纸堆里的页框数 = 读数里的页数（同一份纸堆，不另算）
    const pages = stack!.querySelectorAll('.sched-paper-page').length
    expect(info.textContent).toContain(`共 ${pages} 页`)
  })
})

describe('W6 显示预设：护眼/黑白只换领域色', () => {
  it('切「护眼」→ 根节点叠加 sched-preset-eye 并写入本机偏好', async () => {
    localStorage.removeItem('gaea.schedule.chatPrefs')
    seed({
      project: {
        name: '当前计划', startDate: '2026-09-07',
        tasks: [{ id: 'A', name: '挖土', duration: 2, level: 1, progress: 0 }],
        links: [],
      },
      selectedId: null,
    })
    render(<SchedulePage />)
    fireEvent.click(screen.getByTestId('sched-ribbon-tab-view'))
    const picker = screen.getByTestId('sched-display-preset')
    await pickOption(picker, '护眼')
    expect(screen.getByTestId('sched-page').className).toContain('sched-preset-eye')
    const prefs = JSON.parse(localStorage.getItem('gaea.schedule.chatPrefs')!)
    expect(prefs.ganttDisplayPreset).toBe('eyeCare')
  })

  it('切「黑白」后再回「常规」：类名撤掉，偏好回 normal', async () => {
    localStorage.removeItem('gaea.schedule.chatPrefs')
    seed({
      project: {
        name: '当前计划', startDate: '2026-09-07',
        tasks: [{ id: 'A', name: '挖土', duration: 2, level: 1, progress: 0 }],
        links: [],
      },
      selectedId: null,
    })
    render(<SchedulePage />)
    fireEvent.click(screen.getByTestId('sched-ribbon-tab-view'))
    const picker = screen.getByTestId('sched-display-preset')
    await pickOption(picker, '黑白')
    expect(screen.getByTestId('sched-page').className).toContain('sched-preset-mono')
    await pickOption(picker, '常规')
    const cls = screen.getByTestId('sched-page').className
    expect(cls).not.toContain('sched-preset-mono')
    expect(cls).not.toContain('sched-preset-eye')
    expect(JSON.parse(localStorage.getItem('gaea.schedule.chatPrefs')!).ganttDisplayPreset).toBe('normal')
  })
})

describe('SchedulePage 导入为新工程（v4.144）', () => {
  it('主入口按扩展名分发：MPP 解析→importAsProject（文件名兜底工程名）+成功回执', async () => {
    const importAsProjectSpy = vi.fn(async (_p: { name: string }) => true)
    seed({
      project: { name: '当前计划', startDate: '2026-09-07', tasks: [], links: [] },
      selectedId: null,
      importAsProject: importAsProjectSpy as unknown as ReturnType<typeof useScheduleStore.getState>['importAsProject'],
    })
    render(<SchedulePage />)
    fireEvent.click(screen.getByTestId('sched-ribbon-tab-file'))
    await act(async () => { fireEvent.click(screen.getByTestId('sched-import-btn')) })
    const item = await waitFor(() => {
      const el = Array.from(document.querySelectorAll('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu-item'))
        .find((i) => i.textContent?.includes('导入为新工程'))
      if (!el) throw new Error('菜单未开')
      return el
    })
    await act(async () => { fireEvent.click(item) })
    const input = await waitFor(() => {
      const el = document.querySelector('input[accept=".xml,.xlsx,.xlsm,.mpp"]') as HTMLInputElement | null
      if (!el) throw new Error('新工程文件输入未挂载')
      return el
    })
    await act(async () => {
      fireEvent.change(input, { target: { files: [new File(['MPP binary'], '签证调整2.mpp')] } })
    })
    await waitFor(() => expect(importAsProjectSpy).toHaveBeenCalledTimes(1))
    const arg = importAsProjectSpy.mock.calls[0][0] as unknown as { name: string; tasks: unknown[] }
    expect(arg.name).toBe('签证调整2') // MPP 文件内无工程名 → 文件名兜底
    expect(arg.tasks.length).toBe(1)
    expect(await screen.findByText(/已导入为新工程「签证调整2」/)).toBeTruthy()
  })

  it('不支持格式：诚实报错且不动 store', async () => {
    const importAsProjectSpy = vi.fn(async () => true)
    seed({
      project: { name: '当前计划', startDate: '2026-09-07', tasks: [], links: [] },
      selectedId: null,
      importAsProject: importAsProjectSpy as unknown as ReturnType<typeof useScheduleStore.getState>['importAsProject'],
    })
    render(<SchedulePage />)
    fireEvent.click(screen.getByTestId('sched-ribbon-tab-file'))
    await act(async () => { fireEvent.click(screen.getByTestId('sched-import-btn')) })
    const item = await waitFor(() => {
      const el = Array.from(document.querySelectorAll('.ant-dropdown:not(.ant-dropdown-hidden) .ant-dropdown-menu-item'))
        .find((i) => i.textContent?.includes('导入为新工程'))
      if (!el) throw new Error('菜单未开')
      return el
    })
    await act(async () => { fireEvent.click(item) })
    const input = await waitFor(() => {
      const el = document.querySelector('input[accept=".xml,.xlsx,.xlsm,.mpp"]') as HTMLInputElement | null
      if (!el) throw new Error('新工程文件输入未挂载')
      return el
    })
    await act(async () => {
      fireEvent.change(input, { target: { files: [new File(['x'], '计划.docx')] } })
    })
    expect(await screen.findByText(/不支持的格式 \.docx/)).toBeTruthy()
    expect(importAsProjectSpy).not.toHaveBeenCalled()
  })
})

describe('SchedulePage 工程复制（v4.145 另存为）', () => {
  it('行内「复制」→ 弹窗默认「-副本」名 → 确认调 copyProject(rel, name)', async () => {
    render(<SchedulePage />)
    const panel = await openManagePanel()
    const copyBtns = panel.querySelectorAll('[data-testid="sched-project-copy"]')
    expect(copyBtns.length).toBe(projectsOf.length)
    fireEvent.click(copyBtns[1]) // 办公楼二期
    const input = (await screen.findByTestId('sched-project-copy-input')) as HTMLInputElement
    expect(input.value).toBe('办公楼二期-副本') // 默认名 = 原名+「-副本」
    fireEvent.change(input, { target: { value: '办公楼三期' } })
    fireEvent.click(await screen.findByTestId('sched-project-copy-ok'))
    await waitFor(() => expect(copyProjectSpy).toHaveBeenCalledWith(B2_REL, '办公楼三期'))
  })

  it('空名禁用确认键；复制失败弹窗不关（syncError 由指示器展示）', async () => {
    copyProjectSpy.mockResolvedValue(false)
    render(<SchedulePage />)
    const panel = await openManagePanel()
    fireEvent.click(panel.querySelectorAll('[data-testid="sched-project-copy"]')[0])
    const input = (await screen.findByTestId('sched-project-copy-input')) as HTMLInputElement
    const ok = await screen.findByTestId('sched-project-copy-ok')
    fireEvent.change(input, { target: { value: '  ' } })
    expect(ok.hasAttribute('disabled')).toBe(true) // 空名禁用
    fireEvent.change(input, { target: { value: '快照' } })
    fireEvent.click(ok)
    await waitFor(() => expect(copyProjectSpy).toHaveBeenCalledWith(DEFAULT_REL, '快照'))
    expect(screen.findByTestId('sched-project-copy-input')).toBeTruthy() // 失败弹窗未关
  })
})

// ── W6 收尾：底部状态栏命中区 ───────────────────────────────────────────
//
// 缺口：命中区枚举里早有 `status-bar`、`hitMenu.statusBarMenu()` 也有条目，
// 但页面状态栏没挂 `data-hit`，右键上去毫无反应。这里钉住「属性 + 右键真的弹菜单」，
// 同时钉住「文案/类名不变」（挂属性不能顺手改布局）。
describe('SchedulePage 状态栏命中区（W6 收尾）', () => {
  const bar = () => document.querySelector<HTMLElement>(`[data-hit="status-bar"]`)!

  beforeEach(() => {
    seed({
      project: {
        name: '状态栏样板',
        startDate: '2026-09-07',
        tasks: [
          { id: 'g1', name: '一区', duration: 0, level: 0, progress: 0 },
          { id: 'a', name: '甲', duration: 2, level: 1, progress: 0 },
          { id: 'b', name: '乙', duration: 3, level: 1, progress: 0 },
        ],
        links: [{ from: 'a', to: 'b', type: 'FS', lag: 0 }],
      },
      selectedId: 'a',
    })
  })

  it('状态栏容器挂 data-hit="status-bar"，类名与计数文案保持原样', () => {
    render(<SchedulePage />)
    const el = bar()
    expect(el).toBeTruthy()
    // 类名逐字不变（没有为挂属性多包一层/多加点）——schedule.css 的 .sched-statusbar 照旧命中
    expect(el.className).toBe('sched-statusbar')
    // 计数文案保持（3 行 / 1 个分组 / 2 工作 / 1 关系）
    expect(el.textContent).toContain('3')
    expect(el.textContent).toContain('个分组')
    expect(el.textContent).toContain('工作')
    expect(el.textContent).toContain('关系')
    expect(el.querySelectorAll('.sched-sb-strong').length).toBeGreaterThan(0)
  })

  it('右键状态栏：走统一入口弹菜单（data-hit-region=status-bar，条目来自 hitMenu）', () => {
    render(<SchedulePage />)
    const el = bar()
    // preventDefault 生效 ⇒ fireEvent 返回 false（菜单替代浏览器原生菜单）
    expect(fireEvent.contextMenu(el)).toBe(false)
    const layer = screen.getByTestId('sched-statusbar-hit-menu')
    expect(layer.getAttribute('data-hit-region')).toBe('status-bar')
    expect(layer.textContent).toContain('显示统计口径说明')
    expect(layer.textContent).toContain('复制统计值')
    // 该区条目在纯函数层同样非空（页面只是渲染，不另写一份菜单）
    expect(menuForHit('status-bar', { hasSelection: false, selectedIsGroup: false, selectedIndex: -1, taskCount: 0 }).length).toBeGreaterThan(0)
  })

  it('右键落在状态栏内部子元素上：自限深度上溯仍命中 status-bar', () => {
    render(<SchedulePage />)
    const inner = bar().querySelector<HTMLElement>('.sched-sb-strong')!
    expect(inner.closest('[data-hit]')).toBe(bar())
    // 命中判定与页面同源：从子元素上溯得到的就是 status-bar
    expect(hitRegionOfElement(inner)).toBe('status-bar')
    fireEvent.contextMenu(inner)
    expect(screen.getByTestId('sched-statusbar-hit-menu').getAttribute('data-hit-region')).toBe('status-bar')
  })
})

// ── v3 交互语义（行为线）：功能区/页签/状态栏的键盘可达与状态广播 ──────────
//
// 口径：断言用户可见行为与 AT 可读语义——命令 Enter/Space 真的执行、禁用命令真的
// 不在 Tab 序列里（tabIndex=-1 + aria-disabled，禁用原因留在 title）、页签 roving
// 与 ←/→/Home/End、状态栏 role=status/aria-live 的播报粒度、右键菜单容器
// role=menu/menuitem + aria-disabled 与视觉禁用同源、Esc 收起后焦点回位。
describe('SchedulePage 功能区与状态栏键盘可达（v3）', () => {
  const planButton = (label: string): HTMLButtonElement => {
    const btn = Array.from(document.querySelectorAll<HTMLButtonElement>('.sched-toolbar button')).find(
      (b) => b.textContent === label,
    )
    if (!btn) throw new Error(`功能区无「${label}」`)
    return btn
  }
  const statusBar = () => document.querySelector<HTMLElement>('[data-hit="status-bar"]')!

  beforeEach(() => {
    seed({
      project: {
        name: '键盘样板',
        startDate: '2026-09-07',
        tasks: [{ id: 'a', name: '甲', duration: 2, level: 1, progress: 0 }],
        links: [],
      },
      selectedId: null,
    })
  })

  it('命令按钮在 Tab 序列里：Enter/Space 显式激活（不依赖浏览器默认点击）', () => {
    render(<SchedulePage />)
    expect(useScheduleStore.getState().view).toBe('gantt')
    const timescaled = planButton('时标网络图')
    expect(timescaled.tabIndex).toBe(0) // 启用命令占 Tab 位
    timescaled.focus()
    expect(document.activeElement).toBe(timescaled)
    fireEvent.keyDown(timescaled, { key: 'Enter' })
    expect(useScheduleStore.getState().view).toBe('aoa')
    const gantt = planButton('横道图')
    gantt.focus()
    fireEvent.keyDown(gantt, { key: ' ' })
    expect(useScheduleStore.getState().view).toBe('gantt')
  })

  it('可切换命令播 aria-pressed，非开关命令不装 pressed', () => {
    render(<SchedulePage />)
    expect(planButton('横道图').getAttribute('aria-pressed')).toBe('true')
    expect(planButton('时标网络图').getAttribute('aria-pressed')).toBe('false')
    expect(planButton('复制').getAttribute('aria-pressed')).toBeNull()
  })

  it('禁用命令不可聚焦、aria-disabled 为真，禁用原因仍留在 title', () => {
    render(<SchedulePage />)
    // 「删除」在无选中时禁用（格式刷已接真动作，改用仍属诚实禁用的命令取证）
    const del = planButton('删除')
    expect(del.getAttribute('aria-disabled')).toBe('true')
    expect(del.tabIndex).toBe(-1) // 退出 Tab 序列
    expect(del.disabled).toBe(true) // 原生 disabled：浏览器层不可聚焦、不可点
    // 不调 .focus() 断言「焦点不落进来」——jsdom 的 focus() 不遵守 disabled，
    // 真浏览器里这条由原生 disabled + tabIndex=-1 决定（上面两条即等价事实）
    expect(del.textContent).toBe('删除') // 仍在 DOM 里：命令名对 AT 可读（不是被抹掉）
    expect(del.getAttribute('title')).toContain('请先选中') // 禁用原因对鼠标用户可见
  })

  it('功能区页签：roving tabindex + ←/→/Home/End 移动并激活，Enter 也能激活', () => {
    render(<SchedulePage />)
    const tab = (key: string) => screen.getByTestId(`sched-ribbon-tab-${key}`)
    expect(tab('plan').getAttribute('tabindex')).toBe('0')
    expect(tab('file').getAttribute('tabindex')).toBe('-1')
    tab('file').focus()
    fireEvent.keyDown(tab('file'), { key: 'ArrowRight' })
    expect(tab('plan').getAttribute('aria-selected')).toBe('true')
    expect(tab('plan').getAttribute('tabindex')).toBe('0')
    expect(document.activeElement).toBe(tab('plan'))
    fireEvent.keyDown(tab('plan'), { key: 'End' })
    expect(tab('help').getAttribute('aria-selected')).toBe('true')
    fireEvent.keyDown(tab('help'), { key: 'Home' })
    expect(tab('file').getAttribute('aria-selected')).toBe('true')
    fireEvent.keyDown(tab('file'), { key: 'Enter' })
    expect(tab('file').getAttribute('aria-selected')).toBe('true')
  })

  it('状态栏 role=status / aria-live=polite / aria-atomic=false；保存指示器单独静音', () => {
    render(<SchedulePage />)
    const bar = statusBar()
    expect(bar.getAttribute('role')).toBe('status')
    expect(bar.getAttribute('aria-live')).toBe('polite')
    expect(bar.getAttribute('aria-atomic')).toBe('false') // 只播变化的那一格，不整段重播
    expect(bar.getAttribute('aria-label')).toBe('进度计划状态')
    expect(screen.getByRole('status')).toBe(bar)
    const sync = bar.querySelector('span[aria-live="off"]')!
    expect(sync).not.toBeNull()
    expect(sync.getAttribute('title')).toBe(DEFAULT_REL) // 高频文案不打断读屏
  })

  it('状态栏右键菜单：role=menu/menuitem + aria-disabled 与视觉禁用同步；Esc 收起并回焦', () => {
    render(<SchedulePage />)
    const bar = statusBar()
    expect(fireEvent.contextMenu(bar)).toBe(false) // preventDefault 生效 ⇒ 菜单替代原生菜单
    const layer = screen.getByTestId('sched-statusbar-hit-menu')
    expect(layer.querySelector('[role="menu"]')).not.toBeNull()
    const items = Array.from(layer.querySelectorAll<HTMLElement>('[role="menuitem"]'))
    expect(items.length).toBeGreaterThan(0)
    // 状态栏两条都是说明性条目（诚实禁用）：ARIA 与 ant-menu-item-disabled 同源
    expect(items.every((li) => li.className.includes('ant-menu-item-disabled'))).toBe(true)
    expect(items.every((li) => li.getAttribute('aria-disabled') === 'true')).toBe(true)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByTestId('sched-statusbar-hit-menu')).toBeNull()
    expect(document.activeElement).toBe(bar) // 焦点回触发者，不掉回文档根
  })

  it('图面样式浮层：aria-expanded 随开合，Esc 收起并把焦点还给触发命令', () => {
    render(<SchedulePage />)
    const trigger = screen.getByTestId('sched-network-style-trigger')
    expect(trigger.getAttribute('aria-expanded')).toBe('false')
    trigger.focus()
    fireEvent.click(trigger)
    expect(trigger.getAttribute('aria-expanded')).toBe('true')
    expect(screen.getByTestId('sched-network-style-controls')).toBeTruthy()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(trigger.getAttribute('aria-expanded')).toBe('false')
    expect(document.activeElement).toBe(trigger)
  })
})
