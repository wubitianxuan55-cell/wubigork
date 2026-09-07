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
    const sel = screen.getByTestId('sched-project-select')
    expect(sel.textContent).toContain('当前计划')
    fireEvent.mouseDown(sel.querySelector('.ant-select-selector')!)
    expect(await screen.findByText('办公楼二期', { selector: '.ant-select-item-option-content' })).toBeTruthy()
    expect(screen.queryByText('旧计划', { selector: '.ant-select-item-option-content' })).toBeNull()
  })

  it('选择工程：onChange 调 openProject(rel)（切换+重水合由 store 动作负责）', async () => {
    render(<SchedulePage />)
    await pickOption(screen.getByTestId('sched-project-select'), '办公楼二期')
    await waitFor(() => expect(openProjectSpy).toHaveBeenCalledTimes(1))
    expect(openProjectSpy).toHaveBeenCalledWith(B2_REL)
  })

  it('仅一项工程：切换器收窄为纯显示（disabled，不占交互成本）', () => {
    seed({ projects: [projectsOf[0]], currentPath: DEFAULT_REL })
    render(<SchedulePage />)
    const sel = screen.getByTestId('sched-project-select')
    expect(sel.className).toContain('ant-select-disabled')
  })

  it('projects 为空（旧 store 形态/未加载）：切换器整体隐藏（零感知）', () => {
    seed({ projects: [] })
    render(<SchedulePage />)
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
  it('四个菜单项齐全：编辑（撤销/重做/删除/清空）、视图（四档勾选）、任务（选中前禁用）', async () => {
    render(<SchedulePage />)
    const edit = await openMenu('sched-menu-edit')
    expect(edit.some((t) => t.startsWith('撤销'))).toBe(true)
    expect(edit.some((t) => t.startsWith('重做'))).toBe(true)
    expect(edit).toContain('删除选中(dis)')
    expect(edit).toContain('清空全部任务与搭接')
    const view = await openMenu('sched-menu-view')
    expect(view).toEqual(['横道图', '单代号网络图', '双代号网络图', '资源使用'])
    const task = await openMenu('sched-menu-task')
    expect(task.some((t) => t.startsWith('添加任务'))).toBe(true)
    expect(task).toContain('设为里程碑(dis)')
    expect(task).toContain('任务检查器…(dis)')
    expect(task).toContain('删除选中(dis)')
  })

  it('导入/导出下拉：新工程主入口+三路替换当前显式标注；导出含图面；选中任务后「任务」菜单解锁', async () => {
    seed({
      project: {
        name: '当前计划', startDate: '2026-09-07',
        tasks: [{ id: 'A', name: '挖土', duration: 2, level: 1, progress: 0 }],
        links: [],
      },
      selectedId: 'A',
    })
    render(<SchedulePage />)
    const imp = await openMenu('sched-import-btn')
    expect(imp).toEqual(['导入为新工程…', '导入 XML 替换当前工程…', '导入 Excel 替换当前工程…', '导入 MPP 替换当前工程…'])
    const exp = await openMenu('sched-export-menu-btn')
    expect(exp).toEqual(['导出 MS Project XML', '导出 Excel', '导出图面（PNG / PDF / 打印）…'])
    const task = await openMenu('sched-menu-task')
    expect(task).toContain('设为里程碑')
    expect(task).toContain('任务检查器…')
    expect(task.some((t) => t.startsWith('删除选中') && !t.endsWith('(dis)'))).toBe(true)
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
