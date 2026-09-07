import { useEffect, useMemo, useState } from 'react'
import { diffProjects, type ScheduleApplyDiff } from './applyDiff'
import { simulateOps, type SimOp } from './opsSim'
import { loadScheduleFile } from './api'
import { SCHEDULE_FILE_PATH } from './gschedSummary'
import type { SchedProject } from './types'

// ScheduleDiffCard — schedule_apply 审批卡的专用 diff 卡体（diff 确认闭环刀B）。
//
// Why: ask 级别下审批卡本就弹出，但此前只有「schedule_apply 当前计划」一行字
// ——有确认之形、无审阅之实。本卡体在 approval.tool==="schedule_apply" 时替换
// ApprovalModal 通用卡体（通道/决策/快捷键/超时/两宿主挂载全部复用既有件）。
//
// 数据来源与降级（设计 §2.2）：
// - before = GaeaScheduleLoad 文件实况（与 apply 实际作用对象一致）——绑定只
//   服务缺省路径，args.path 非缺省 → 降级「非当前计划，无现状对比」意图清单；
// - after = args.project（弹卡期间 Transcript 工具卡已携带完整 args，§1.3）；
// - ops 通道：simulateOps 是刀D（Go ApplyOps TS 镜像），本刀诚实降级为意图
//   清单并说明，绝不伪造 before 值。
//
// 数字全部是**预览**（前端投影）：落盘后以 apply 回执为准——卡头显式标注。
// 行数上限 60 行 +「展开全部」，防整计划生成时卡体无限膨胀（审批疲劳对策）。

const ROW_CAP = 60

interface DiffLine {
  kind: 'add' | 'del' | 'mod' | 'info'
  text: string
}

function projectChannelLines(diff: ScheduleApplyDiff): DiffLine[] {
  const lines: DiffLine[] = []
  for (const m of diff.meta) lines.push({ kind: 'mod', text: `${m.label}：${m.from} → ${m.to}` })
  for (const t of diff.tasks.added) lines.push({ kind: 'add', text: `新增任务：${t.name}` })
  for (const t of diff.tasks.removed) lines.push({ kind: 'del', text: `删除任务：${t.name}` })
  for (const t of diff.tasks.modified) {
    lines.push({ kind: 'mod', text: `修改任务「${t.name}」` })
    for (const f of t.fields) lines.push({ kind: 'mod', text: `  · ${f.label}：${f.from} → ${f.to}` })
  }
  for (const l of diff.links.added) lines.push({ kind: 'add', text: `新增搭接（${l.taskName}）：${l.to}` })
  for (const l of diff.links.changed) lines.push({ kind: 'mod', text: `改搭接（${l.taskName}）：${l.from} → ${l.to}` })
  for (const l of diff.links.removed) lines.push({ kind: 'del', text: `删除搭接（${l.taskName}）：${l.from}` })
  for (const r of diff.resources.added) lines.push({ kind: 'add', text: `新增资源：${r.name}` })
  for (const r of diff.resources.removed) {
    lines.push({ kind: 'del', text: `删除资源：${r.name}${r.cascade ? `（随行删除 ${r.cascade} 条分配）` : ''}` })
  }
  for (const r of diff.resources.modified) {
    lines.push({ kind: 'mod', text: `修改资源「${r.name}」` })
    for (const f of r.fields) lines.push({ kind: 'mod', text: `  · ${f.label}：${f.from} → ${f.to}` })
  }
  for (const a of diff.assignments.added) {
    lines.push({
      kind: 'add',
      text: `新增分配：${a.taskName} ← ${a.resourceName}${a.fields?.length ? `（${a.fields.map((f) => `${f.label}：${f.to}`).join('，')}）` : ''}`,
    })
  }
  for (const a of diff.assignments.removed) lines.push({ kind: 'del', text: `删除分配：${a.taskName} ← ${a.resourceName}` })
  return lines
}

function LineRow({ line }: { line: DiffLine }) {
  const tone = line.kind === 'add' ? 'text-ok' : line.kind === 'del' ? 'text-err' : 'text-fg-dim'
  const mark = line.kind === 'add' ? '+' : line.kind === 'del' ? '−' : '·'
  return (
    <div className={`flex gap-1.5 text-[12px] leading-[1.5] ${tone}`} data-testid="sched-diff-line">
      <span className="shrink-0 font-mono select-none" aria-hidden>{mark}</span>
      <span className="min-w-0 break-all">{line.text}</span>
    </div>
  )
}

/** 解析 schedule_apply 参数：project / ops / path 三态。解析失败返回 null。 */
function parseApplyArgs(args: string | null): { path?: string; project?: SchedProject; ops?: unknown[] } | null {
  if (!args) return null
  try {
    const parsed = JSON.parse(args) as { path?: unknown; project?: unknown; ops?: unknown }
    return {
      path: typeof parsed.path === 'string' ? parsed.path : undefined,
      project: parsed.project && typeof parsed.project === 'object' ? (parsed.project as SchedProject) : undefined,
      ops: Array.isArray(parsed.ops) ? parsed.ops : undefined,
    }
  } catch {
    return null
  }
}

export function ScheduleDiffCard({ args }: { args: string | null }) {
  const parsed = useMemo(() => parseApplyArgs(args), [args])
  const isDefaultPath = !parsed?.path || parsed.path === SCHEDULE_FILE_PATH
  // before 文件实况：仅缺省路径可得（绑定口径）；非缺省/读取失败 → null=降级
  const [before, setBefore] = useState<SchedProject | null>(null)
  const [expanded, setExpanded] = useState(false)

  const projectKey = parsed?.project ? JSON.stringify(parsed.project).length + (parsed.project.name ?? '') : ''
  useEffect(() => {
    if (!parsed || (!parsed.project && !parsed.ops) || !isDefaultPath) return
    let alive = true
    loadScheduleFile()
      .then((r) => { if (alive) setBefore(r.exists && r.project ? r.project : null) })
      .catch(() => { if (alive) setBefore(null) })
    return () => { alive = false }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- parsed 是 useMemo 派生对象，展开依赖会每帧重读文件；projectKey 已代表其内容变化
  }, [parsed?.project, parsed?.ops, isDefaultPath, projectKey])

  if (!parsed) {
    return (
      <div className="mb-3 border border-border-soft rounded-lg bg-bg-soft py-[7px] px-2 text-fg-faint text-[12px]">
        参数尚未就绪或解析失败，无法生成 diff 预览（参数原文见下方折叠区）。
      </div>
    )
  }

  // ── 统一 diff 计算：project=直比；ops=simulateOps 投影（刀D）后同管线 ──
  // 降级链（全部诚实上屏）：非缺省路径（绑定只服务缺省）→ before 读取失败 →
  // ops 模拟失败（Go 闸门/落盘权威不受影响，回执为准）→ ops 意图清单。
  let diff: ScheduleApplyDiff | null = null
  let degrade: string | null = null
  if (parsed.project) {
    diff = before ? diffProjects(before, parsed.project) : null
    if (!before) degrade = isDefaultPath ? '现状读取失败，无对比' : '非当前计划文件，无现状对比'
  } else if (before) {
    const sim = simulateOps(before, (parsed.ops ?? []) as SimOp[])
    if (sim.ok) {
      diff = diffProjects(before, sim.project)
    } else {
      degrade = `ops 投影失败（${sim.error}）：本次执行结果以 apply 回执为准`
    }
  } else if (!isDefaultPath) {
    degrade = '非当前计划文件，无现状对比（ops 意图见参数折叠区）'
  } else {
    degrade = '现状读取失败，无对比'
  }
  if (!parsed.project && !before) {
    // ops 通道降级形态：意图清单 + 原因
    return (
      <div className="mb-3 border border-border-soft rounded-lg bg-bg-soft py-[7px] px-2" data-testid="sched-diff-card">
        <div className="text-fg-dim text-[12px] leading-[1.5]">
          ops 增量调整 {(parsed.ops ?? []).length} 条：{degrade}。
          本次执行结果以 apply 回执为准，回执卡可用「回滚本次」整体还原。
        </div>
      </div>
    )
  }
  const s = diff?.summary
  const lines = diff ? projectChannelLines(diff) : []
  const shown = expanded ? lines : lines.slice(0, ROW_CAP)

  return (
    <div className="mb-3 border border-border-soft rounded-lg bg-bg-soft py-[7px] px-2" data-testid="sched-diff-card">
      {/* 卡头汇总（预览口径显式标注；审批疲劳对策=汇总行醒目+行数截断） */}
      <div className="flex items-center gap-2 flex-wrap text-[12px] leading-[1.5] mb-1.5" data-testid="sched-diff-summary">
        <span className="shrink-0 text-fg-faint font-mono text-[11px] uppercase tracking-[0.04em]">diff 预览</span>
        {diff ? (
          <>
            <span className="font-mono text-fg">
              总工期 {s?.durationFrom ?? '—'}→{s?.durationTo ?? '—'} 天
            </span>
            <span className="font-mono text-fg-dim">
              关键 {s?.criticalFrom ?? '—'}→{s?.criticalTo ?? '—'} 项
            </span>
            {(s?.costFrom !== undefined || s?.costTo !== undefined) && (
              <span className="font-mono text-fg-dim">
                总成本 {s?.costFrom ?? '—'}→{s?.costTo ?? '—'} 元
              </span>
            )}
            {s?.deadlineOk === false && (
              <span className="text-err" data-testid="sched-diff-deadline">目标竣工：超期 {s.deadlineOver} 天</span>
            )}
            {s?.deadlineOk === true && <span className="text-ok">目标竣工：可行</span>}
            {(s?.baselineShifted || s?.baselineAdded || s?.baselineRemoved) ? (
              <span className="text-fg-dim">
                基线漂移：挪移 {s.baselineShifted ?? 0} / 新增 {s.baselineAdded ?? 0} / 移除 {s.baselineRemoved ?? 0}
              </span>
            ) : null}
          </>
        ) : (
          <span className="text-fg-faint">{degrade ?? '无对比'}</span>
        )}
        <span className="ml-auto text-[10px] text-fg-faint">{"预览数字（" + (parsed.project ? "文件实况对比" : "ops 投影") + "），落盘以回执为准"}</span>
      </div>

      {/* 预览层复刻引擎 fail-closed：after CPM 不过=这单会被拒，如实提示 */}
      {s?.cpmError && (
        <div className="mb-1.5 text-err text-[12px] leading-[1.5]" data-testid="sched-diff-cpm-error">
          引擎将拒绝（CPM 不过）：{s.cpmError}——批准也不会落盘，请先修正逻辑关系。
        </div>
      )}
      {diff?.empty && (
        <div className="mb-1.5 text-fg-faint text-[12px]">与当前计划无实质改动。</div>
      )}

      {!before && !isDefaultPath && (
        <div className="mb-1.5 text-fg-dim text-[12px]">
          目标文件「{parsed.path}」非当前计划，仅列改动清单（无现状对比）。
        </div>
      )}

      {lines.length > 0 && (
        <div className="flex flex-col gap-0.5 max-h-[220px] overflow-auto">
          {shown.map((l, i) => <LineRow key={i} line={l} />)}
          {lines.length > ROW_CAP && !expanded && (
            <button
              className="mt-1 self-start text-[11px] text-fg-faint hover:text-fg cursor-pointer bg-transparent border-0 p-0"
              onClick={() => setExpanded(true)}
              data-testid="sched-diff-expand"
            >
              展开全部（共 {lines.length} 行，已截断）
            </button>
          )}
        </div>
      )}
      {before && lines.length === 0 && !diff?.empty && <div className="text-fg-faint text-[12px]">（无明细行）</div>}
    </div>
  )
}
