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
