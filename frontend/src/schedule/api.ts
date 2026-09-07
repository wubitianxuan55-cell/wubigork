/**
 * schedule/api.ts — 进度计划文件持久化通道（v4.113.0 刀4）
 *
 * 板块与 agent 共享同一计划资产（进度计划/当前计划.gsched.json）：
 * 板块走 GaeaScheduleLoad/Save（轻量 IO，无证据卡）；agent 走 schedule_*
 * 工具（快照+证据卡）。本模块经 bridge `app` 门面调用——真实壳走 Wails，
 * ?mock=1 走 mock/office.ts 的会话内存态。
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

/** 装载默认计划文件；exists=false 表示尚无文件（首启/清空后） */
export async function loadScheduleFile(): Promise<ScheduleLoadOk> {
  const r = await app.ScheduleLoad()
  if (!r.exists || !r.project) return { exists: false, path: r.path }
  return { exists: true, path: r.path, project: normalizeProject(JSON.parse(r.project)) }
}

/** 保存计划（整量覆盖，引擎校验+CPM fail-closed 在 Go 侧） */
export async function saveScheduleFile(p: SchedProject): Promise<ScheduleSaveOk> {
  return app.ScheduleSave(JSON.stringify(normalizeProject(p)))
}

const XLSX_MIME = 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'

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
  const bytes = new Uint8Array(await file.arrayBuffer())
  const CHUNK = 0x8000
  let bin = ''
  for (let i = 0; i < bytes.length; i += CHUNK) {
    bin += String.fromCharCode(...bytes.subarray(i, i + CHUNK))
  }
  return normalizeProject(JSON.parse(await app.ScheduleImportXlsx(btoa(bin))))
}
