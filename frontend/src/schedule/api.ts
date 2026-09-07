/**
 * schedule/api.ts — 进度计划文件持久化通道（v4.113.0 刀4；v4.139 #15 刀1 多工程）
 *
 * 板块与 agent 共享同一批计划资产（进度计划/*.gsched.json，每文件一工程，
 * 「当前工程」指针收敛在 Go 侧索引 .gaea/schedule/index.json，设计文档
 * docs/gaea-schedule-multi-project-design-2026-09.md §3.1/3.3）：板块走
 * GaeaScheduleLoad/Save/Projects/Project*（轻量 IO，无证据卡）；agent 走
 * schedule_* 工具（快照+证据卡），缺省 path 同样跟随当前指针。本模块经
 * bridge `app` 门面调用——真实壳走 Wails，?mock=1 走 mock/office.ts 的
 * 会话内存态。
 */
import { app } from '../gaea/lib/bridge'
import type { SchedProject } from './types'
import { normalizeProject } from './store'

export interface ScheduleLoadOk {
  exists: boolean
  project?: SchedProject
  path: string
}

export interface ScheduleSaveOk {
  path: string
  savedAt: string
  duration: number
  critical: number
}

/** 工程摘要（GaeaScheduleProjects 逐文件算出的索引缓存条目） */
export interface ScheduleProjectSummary {
  /** 工程文件 rel（工程身份=文件，此后不再改名；agent path 引用稳定） */
  rel: string
  /** 显示名（文件内 project.name，非文件名） */
  name: string
  archived: boolean
  /** 总工期（工作日；CPM 不过时 Go 侧口径回 0） */
  duration: number
  /** 叶任务数 */
  taskCount: number
  /** CPM 通过 */
  ok: boolean
  updatedAt: string
}

/** 工程列表 + 当前指针视图（索引缓存口径，可随目录扫描重建） */
export interface ScheduleProjectsOk {
  current: string
  projects: ScheduleProjectSummary[]
}

/** 新建工程回执：新文件 rel 与切换后的当前指针 */
export interface ScheduleProjectCreateOk {
  rel: string
  current: string
}

/** 归档/删除回执：最新当前指针与最新工程列表 */
export interface ScheduleProjectMutatedOk {
  current: string
  projects: ScheduleProjectSummary[]
}

/**
 * 多工程绑定窄接口（v4.139 刀1 过渡声明）：bridge AppBindings 与
 * LegacySurfaceNames 的对应键（ScheduleSave 双参/ScheduleProjects/ScheduleProjectOpen/
 * ScheduleProjectCreate/ScheduleProjectArchive/ScheduleProjectDelete）由并行刀
 * 补齐。这里在本文件内声明本模块所需的最小签名面，仍经同一 `app` 门面代理
 * 调用（运行时按方法名路由：真壳按映射找 Gaea 前缀绑定，mock 同名直暴露），
 * tsc 不阻塞于 bridge 合入时序；bridge 补齐后可改回 app 直调并删除本接口。
 */
interface ScheduleMultiProjectSurface {
  ScheduleSave(projectJSON: string, rel: string): Promise<ScheduleSaveOk>
  ScheduleProjects(): Promise<ScheduleProjectsOk>
  ScheduleProjectOpen(rel: string): Promise<{ current: string }>
  ScheduleProjectCreate(name: string): Promise<ScheduleProjectCreateOk>
  ScheduleProjectArchive(rel: string, archived: boolean): Promise<ScheduleProjectMutatedOk>
  ScheduleProjectDelete(rel: string): Promise<ScheduleProjectMutatedOk>
}

/** 同一通道取窄接口视图：app 是调用时解析的代理，强转只影响静态类型 */
function scheduleSurface(): ScheduleMultiProjectSurface {
  return app as unknown as ScheduleMultiProjectSurface
}

/** 装载当前指针指向的计划文件；exists=false 表示尚无文件（首启/清空后） */
export async function loadScheduleFile(): Promise<ScheduleLoadOk> {
  const r = await app.ScheduleLoad()
  if (!r.exists || !r.project) return { exists: false, path: r.path }
  return { exists: true, path: r.path, project: normalizeProject(JSON.parse(r.project)) }
}

/**
 * 保存计划（整量覆盖，引擎校验+CPM fail-closed 在 Go 侧）。
 * v4.139 #15 刀1：rel=目标工程文件 rel——store 自动保存显式带装载时的工程
 * 文件（设计 §3.2 方案 A：切换只改指针不动保存目标，结构性防「切换瞬间
 * 旧内容写进新文件」竞态）；空串=跟随当前指针（兼容旧调用形态）。
 */
export async function saveScheduleFile(p: SchedProject, rel = ''): Promise<ScheduleSaveOk> {
  return scheduleSurface().ScheduleSave(JSON.stringify(normalizeProject(p)), rel)
}

/** 工程列表 + 当前指针（Go 侧逐文件 Load+Analyze 算摘要，结果缓存进索引） */
export async function listScheduleProjects(): Promise<ScheduleProjectsOk> {
  return scheduleSurface().ScheduleProjects()
}

/** 切换当前工程：只做 rel 合法性校验 + 原子写索引 current，不动任何文件 */
export async function openScheduleProject(rel: string): Promise<{ current: string }> {
  return scheduleSurface().ScheduleProjectOpen(rel)
}

/** 新建工程：Go 生成安全 slug 文件名落盘（空 Project 走全套校验）+登记索引+自动切为当前 */
export async function createScheduleProject(name: string): Promise<ScheduleProjectCreateOk> {
  return scheduleSurface().ScheduleProjectCreate(name)
}

/** 归档/反归档：索引标 archived，文件保留原位（agent 显式 path 引用不失效） */
export async function archiveScheduleProject(rel: string, archived: boolean): Promise<ScheduleProjectMutatedOk> {
  return scheduleSurface().ScheduleProjectArchive(rel, archived)
}

/** 删除工程：物理删文件+索引摘除（不可恢复）；删当前工程时 Go 侧先切到剩余第一个 */
export async function deleteScheduleProject(rel: string): Promise<ScheduleProjectMutatedOk> {
  return scheduleSurface().ScheduleProjectDelete(rel)
}

const XLSX_MIME = 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'

/** 文件 → base64（分块拼接，避开 Function.apply 参数上限） */
async function fileToBase64(file: File): Promise<string> {
  const bytes = new Uint8Array(await file.arrayBuffer())
  const CHUNK = 0x8000
  let bin = ''
  for (let i = 0; i < bytes.length; i += CHUNK) {
    bin += String.fromCharCode(...bytes.subarray(i, i + CHUNK))
  }
  return btoa(bin)
}

/** 导出上报 Excel：Go excelize 渲染（CPM fail-closed）→ base64 → Blob 下载 */
export async function exportScheduleXlsx(p: SchedProject): Promise<void> {
  const b64 = await app.ScheduleExportXlsx(JSON.stringify(normalizeProject(p)))
  const bin = atob(b64)
  const bytes = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
  const url = URL.createObjectURL(new Blob([bytes], { type: XLSX_MIME }))
  const a = document.createElement('a')
  a.href = url
  a.download = `${p.name || '进度计划'}.xlsx`
  a.click()
  URL.revokeObjectURL(url)
}

/** 导入上报 Excel：文件 → base64 → Go 解析 → 计划（排程交回 CPM 重算） */
export async function importScheduleXlsx(file: File): Promise<SchedProject> {
  return normalizeProject(JSON.parse(await app.ScheduleImportXlsx(await fileToBase64(file))))
}

/** 导入 MPP（v4.140）：二进制 MS Project 工程 → Go 解析器（MPP9/12/14 子集）→ 计划 */
export async function importScheduleMpp(file: File): Promise<SchedProject> {
  return normalizeProject(JSON.parse(await app.ScheduleImportMpp(await fileToBase64(file))))
}
