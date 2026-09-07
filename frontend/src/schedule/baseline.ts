/**
 * baseline.ts — 计划基线快照与漂移对比（纯函数，v4.116.0 刀7）
 *
 * 对齐 Project「设置基线」口径：把当前排程结果（每项叶任务的 ES/EF/工期/
 * 关键标记）固化为基线，之后每次调整都可量化偏差——总工期漂移、任务推移、
 * 关键链进出。基线只存叶任务（分组行汇总由视图推导）；行值是**工作日序号**，
 * 显示日期由当前开工日期+日历换算（对比口径=相对开工日的偏移，不受开工日
 * 调整影响）。与 Go 侧 internal/schedule/baseline.go 互为镜像（测试对齐）。
 */
import type { BaselineDrift, BaselineDriftRow, CpmResult, SchedBaseline, SchedBaselineRow, SchedProject } from './types'

/**
 * 有效工期快照口径（v4.150 双工期刀1 换源）：cpm 行的等效工作日跨度 ef−es
 * （wd 任务=effDur 逐位相等；cd 任务=养护窗口内的工作日数——基线漂移仍在
 * 工作日空间对比，「等效跨度随日历/位置变化」是诚实口径非 bug）。
 */
function rowDur(row: { es: number; ef: number }): number {
  return row.ef - row.es
}

/**
 * 保存基线：以当前 CPM 结果固快照。守卫 fail-closed——计划未通过 CPM
 * （循环依赖）或没有任何叶任务时报错，绝不存出无意义的空基线。
 */
export function snapshotBaseline(
  project: SchedProject,
  cpm: CpmResult,
  savedAt: string,
  name?: string,
): { ok: true; baseline: SchedBaseline } | { ok: false; error: string } {
  if (!cpm.ok) return { ok: false, error: cpm.error || '计划未通过 CPM 计算，无法保存基线' }
  const rows: Record<string, SchedBaselineRow> = {}
  for (const t of project.tasks) {
    if (t.level === 0) continue
    const row = cpm.rows[t.id]
    if (!row) continue
    rows[t.id] = { name: t.name, es: row.es, ef: row.ef, dur: rowDur(row), critical: row.critical }
  }
  if (Object.keys(rows).length === 0) {
    return { ok: false, error: '计划中没有叶任务，无可固化的基线' }
  }
  return {
    ok: true,
    baseline: { name: name?.trim() || '基线', savedAt, duration: cpm.duration, rows },
  }
}

/**
 * 基线漂移对比。无基线返回 null；rows 只含**有偏差**的任务（推移/新增/移除，
 * 一致的行不进列表，数量在 sameCount），顺序=当前任务表序，移除行按 id 排序
 * （与 Go 镜像一致）。计划未通过 CPM 时返回 ok=false（此时无法对比）。
 */
export function computeBaselineDrift(project: SchedProject, cpm: CpmResult): BaselineDrift | null {
  const base = project.baseline
  if (!base || !cpm.ok) return null
  const rows: BaselineDriftRow[] = []
  let sameCount = 0
  let shiftedCount = 0
  let addedCount = 0
  const criticalGained: string[] = []
  const criticalLost: string[] = []
  const seen = new Set<string>()

  for (const t of project.tasks) {
    if (t.level === 0) continue
    const row = cpm.rows[t.id]
    if (!row) continue
    seen.add(t.id)
    const b = base.rows[t.id]
    if (!b) {
      addedCount++
      rows.push({
        id: t.id, name: t.name, kind: 'added', base: null,
        now: { es: row.es, ef: row.ef, dur: rowDur(row) },
        esDrift: row.es, efDrift: row.ef, durDrift: rowDur(row),
        criticalNow: row.critical, criticalBase: false,
      })
      if (row.critical) criticalGained.push(t.name)
      continue
    }
    const dur = rowDur(row)
    const esD = row.es - b.es
    const efD = row.ef - b.ef
    const durD = dur - b.dur
    const shifted = esD !== 0 || efD !== 0 || durD !== 0 || row.critical !== b.critical
    if (!shifted) {
      sameCount++
      continue
    }
    shiftedCount++
    rows.push({
      id: t.id, name: t.name, kind: 'shifted', base: b,
      now: { es: row.es, ef: row.ef, dur },
      esDrift: esD, efDrift: efD, durDrift: durD,
      criticalNow: row.critical, criticalBase: b.critical,
    })
    if (row.critical && !b.critical) criticalGained.push(t.name)
    if (!row.critical && b.critical) criticalLost.push(t.name)
  }

  // 基线里有、当前计划没有的行=已移除（按 id 排序，跨语言镜像一致）
  const removedIds = Object.keys(base.rows).filter((id) => !seen.has(id)).sort()
  const removedCount = removedIds.length
  for (const id of removedIds) {
    const b = base.rows[id]
    rows.push({
      id, name: b.name, kind: 'removed', base: b, now: null,
      esDrift: -b.es, efDrift: -b.ef, durDrift: -b.dur,
      criticalNow: false, criticalBase: b.critical,
    })
  }

  return {
    baselineName: base.name,
    baselineSavedAt: base.savedAt,
    baselineDuration: base.duration,
    currentDuration: cpm.duration,
    durationDrift: cpm.duration - base.duration,
    sameCount, shiftedCount, addedCount, removedCount,
    criticalGained, criticalLost,
    rows,
  }
}

/**
 * 基线槽位更新（v4.137 #11 多基线）：同名在原位替换（「更新基线」语义），
 * 否则追加；超出 max 按 FIFO 淘汰最旧（签证 1→N 场景默认保留 3 个槽），
 * 刚写入的槽永不被淘汰。纯函数，不改入参。
 */
export function upsertBaseline(
  list: SchedBaseline[] | undefined,
  b: SchedBaseline,
  max = 3,
): SchedBaseline[] {
  const next = [...(list ?? [])]
  const i = next.findIndex((x) => x.name === b.name)
  if (i >= 0) next[i] = b
  else next.push(b)
  while (next.length > max && next.length > 1) {
    const oldestIdx = next.findIndex((x) => x.name !== b.name)
    if (oldestIdx < 0) break
    next.splice(oldestIdx, 1)
  }
  return next
}
