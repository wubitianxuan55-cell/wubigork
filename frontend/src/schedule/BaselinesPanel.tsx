/**
 * schedule/BaselinesPanel.tsx — 基线对比弹层·多基线槽位版（v4.137 #11）
 *
 * 将替换 SchedulePage 内旧 BaselinePanel（单基线版）进基线 Popover，交互与文案
 * 沿用旧版语气，升级为槽位制（上限 3，FIFO 淘汰最旧，store.upsertBaseline 裁决）：
 *  - 无槽位且无活跃基线 → 旧版「保存基线」引导（守卫 fail-closed：循环依赖/无叶任务
 *    禁用并给原因；store 返回的失败原因不静默）；保存即建槽；
 *  - 有槽位 → 槽位列表（radio 选活跃 + 名称/保存时间/总工期 + 删除 Popconfirm），
 *    活跃行高亮标「当前对比」；「保存为新基线」展开名输入（缺省名 基线{N+1}，
 *    重名=原位更新该槽），3 槽存满提示「将淘汰最旧槽位」；
 *  - 有活跃基线 → 旧版漂移摘要（总工期 X→Y/推移·新增·移除·一致/关键链进出/偏差行
 *    清单，数值一律来自 computeBaselineDrift，UI 不重复实现公式）+「更新基线」
 *    （=setBaseline(活跃名) 原位更新）+「清除对比」（=clearBaseline，只清活跃指针，
 *    槽位保留可再激活）；
 *  - 有槽位 → 槽位列表下方「导出台账」（v4.138）：CSV 下载（\ufeff BOM 防 Excel
 *    中文乱码），列=基线名称/保存时间/基线总工期/当前总工期/总工期漂移，每槽
 *    一行不汇总；导出复用 exportArtifact.saveExportBlob（壳内系统另存为）。
 * props 只收 cpm（页面级 CPM 结果）；工程与槽位动作一律取自 useScheduleStore，
 * 不依赖页面局部状态，可整体替换进基线 Popover。
 */
import React, { useMemo, useState } from 'react'
import { Button, Input, Popconfirm, Radio, Space, Tag } from 'antd'
import { computeBaselineDrift } from './baseline'
import { saveExportBlob } from './exportArtifact'
import { useScheduleStore } from './store'
import type { CpmResult, SchedBaseline } from './types'

/** 偏移量文本：+N / -N / ±0（与旧版同口径） */
function fmtDrift(n: number): string {
  return n > 0 ? `+${n}` : `${n}`
}

/** 基线漂移行类别徽标 */
const DRIFT_KIND_LABEL: Record<'shifted' | 'added' | 'removed', string> = {
  shifted: '推移',
  added: '新增',
  removed: '移除',
}

/** 保存新基线的缺省名：基线{槽位数+1} */
function defaultSlotName(count: number): string {
  return `基线${count + 1}`
}

/** CSV 单元格转义：含逗号/引号/换行的字段用引号包裹、内部引号翻倍（Excel 兼容） */
function csvCell(v: string): string {
  return /[",\r\n]/.test(v) ? `"${v.replace(/"/g, '""')}"` : v
}

/**
 * 签证台账 CSV 文本（不含 BOM，BOM 由调用方拼在 Blob 头部）：
 * 表头 + 每槽一行（当前总工期=cpm.duration；漂移=当前-基线，正=拖后），
 * 末行不汇总。
 */
function ledgerCsv(slots: SchedBaseline[], currentDuration: number): string {
  const header = '基线名称,保存时间,基线总工期(天),当前总工期(天),总工期漂移(天)'
  const rows = slots.map((b) => [
    b.name, b.savedAt, String(b.duration), String(currentDuration), String(currentDuration - b.duration),
  ].map(csvCell).join(','))
  return [header, ...rows].join('\r\n')
}

/** 基线对比弹层（多基线槽位版） */
export const BaselinesPanel: React.FC<{ cpm: CpmResult }> = ({ cpm }) => {
  const project = useScheduleStore((s) => s.project)
  const setBaseline = useScheduleStore((s) => s.setBaseline)
  const activateBaseline = useScheduleStore((s) => s.activateBaseline)
  const removeBaseline = useScheduleStore((s) => s.removeBaseline)
  const clearBaseline = useScheduleStore((s) => s.clearBaseline)

  /** 保存失败原因（store 返回，不静默） */
  const [err, setErr] = useState<string | null>(null)
  /** 保存新基线表单：null=收起；字符串=输入框草稿名 */
  const [draft, setDraft] = useState<string | null>(null)

  const slots = project.baselines ?? []
  const active = project.baseline ?? null
  const drift = useMemo(() => computeBaselineDrift(project, cpm), [project, cpm])
  const hasLeaf = project.tasks.some((t) => t.level > 0)
  /** 保存/更新守卫（沿用旧版口径）：循环依赖 > 无叶任务 */
  const guard = !cpm.ok ? '计划存在循环依赖，先修正搭接' : !hasLeaf ? '计划还没有任何任务' : null

  const openForm = () => { setErr(null); setDraft(defaultSlotName(slots.length)) }
  const closeForm = () => setDraft(null)

  /** 提交保存：空名回落缺省名；重名=原位更新该槽（store upsert 裁决） */
  const submitSave = () => {
    setErr(setBaseline((draft ?? '').trim() || defaultSlotName(slots.length)))
    closeForm()
  }

  /** 更新基线：按活跃名原位更新（=重存当前活跃槽，槽位数不变） */
  const updateActive = () => {
    if (!drift) return
    setErr(setBaseline(drift.baselineName))
  }

  /** 导出签证台账 CSV：\ufeff BOM 防 Excel 中文乱码；文件名=工程名-签证台账.csv；
   *  壳内经 saveExportBlob 走系统另存为（<a download> 不落盘，v4.162），浏览器回退下载 */
  const exportLedger = () => {
    const csv = ledgerCsv(slots, cpm.duration)
    const blob = new Blob(['\ufeff' + csv], { type: 'text/csv;charset=utf-8' })
    void saveExportBlob(blob, `${project.name || '进度计划'}-签证台账.csv`)
  }

  return (
    <div data-testid="sched-baselines-panel" style={{ display: 'grid', gap: 8, minWidth: 300, maxWidth: 420 }}>
      <span className="sched-dim">
        基线会固化当前排程结果；之后每次调整都能对比「较基线」的推移、总工期漂移与关键链变化。
        可保留最多 3 个基线槽位，随时切换对比对象。
      </span>

      {/* 引导：无任何槽位且无活跃基线（旧版文案与守卫） */}
      {draft === null && slots.length === 0 && !active && (
        <>
          {guard && <span className="sched-dim">暂不可保存：{guard}</span>}
          <Button size="small" type="primary" disabled={!!guard} data-testid="sched-baseline-save" onClick={openForm}>
            保存基线
          </Button>
        </>
      )}

      {/* 槽位列表：radio 选活跃 + 名称/保存时间/总工期 + 删除（Popconfirm） */}
      {slots.length > 0 && (
        <div style={{ display: 'grid', gap: 4 }}>
          {slots.map((b) => {
            const isActive = active?.name === b.name
            return (
              <div
                key={b.name}
                data-testid="sched-baseline-slot"
                data-slot-name={b.name}
                style={{
                  display: 'flex', alignItems: 'center', gap: 6, padding: '3px 6px', borderRadius: 6,
                  background: isActive ? 'rgba(128, 128, 128, 0.12)' : undefined,
                }}
              >
                <span data-testid="sched-baseline-activate" title="设为当前对比">
                  <Radio checked={isActive} onChange={() => activateBaseline(b.name)} />
                </span>
                <span
                  style={{
                    flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                    fontWeight: isActive ? 600 : undefined,
                  }}
                >
                  {b.name}
                </span>
                {isActive && <Tag color="processing" style={{ marginRight: 0 }}>当前对比</Tag>}
                <span className="sched-dim" style={{ fontSize: 12 }}>总工期 {b.duration} 天</span>
                <span className="sched-dim" style={{ fontSize: 12 }}>{b.savedAt}</span>
                <Popconfirm
                  title={`删除基线「${b.name}」？`}
                  description={isActive ? '该槽是当前对比，删除后对比结束' : undefined}
                  onConfirm={() => { removeBaseline(b.name); setErr(null) }}
                >
                  <Button size="small" type="text" danger data-testid="sched-baseline-remove" aria-label={`删除基线${b.name}`}>删除</Button>
                </Popconfirm>
              </div>
            )
          })}
        </div>
      )}

      {/* 签证台账导出（v4.138）：有槽位才在位，无槽位隐藏；CSV 每槽一行对比当前总工期 */}
      {slots.length > 0 && (
        <Button size="small" data-testid="sched-baseline-ledger" onClick={exportLedger}>
          导出台账
        </Button>
      )}

      {/* 保存新基线：表单展开态 / 按钮态（缺省名=基线{N+1}，重名=原位更新） */}
      {draft !== null ? (
        <Space size={6} wrap>
          <Input
            size="small" style={{ width: 150 }} autoFocus value={draft}
            data-testid="sched-baseline-name-input" placeholder="基线名"
            onChange={(e) => setDraft(e.target.value)}
            onPressEnter={submitSave}
          />
          <Button size="small" type="primary" disabled={!!guard} data-testid="sched-baseline-save-confirm" onClick={submitSave}>保存</Button>
          <Button size="small" type="text" onClick={closeForm}>取消</Button>
        </Space>
      ) : (
        (slots.length > 0 || active) && (
          <Space size={6} wrap>
            <Button size="small" type="dashed" disabled={!!guard} data-testid="sched-baseline-save" onClick={openForm}>
              保存为新基线
            </Button>
            <span className="sched-dim" style={{ fontSize: 12 }}>槽位 {slots.length}/3</span>
          </Space>
        )
      )}
      {slots.length >= 3 && (
        <span className="sched-dim" style={{ fontSize: 12 }}>已存满 3 个槽位，保存新基线将淘汰最旧槽位</span>
      )}

      {/* 漂移摘要：有活跃基线且 CPM 通过（沿用旧版展示） */}
      {drift && (
        <div style={{ display: 'grid', gap: 6, borderTop: '1px solid rgba(128, 128, 128, 0.25)', paddingTop: 6 }}>
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 8 }}>
            <strong>{drift.baselineName}</strong>
            <span className="sched-dim">保存于 {drift.baselineSavedAt}</span>
          </div>
          <div style={{ fontSize: 13 }}>
            总工期 <b>{drift.baselineDuration}</b> → <b>{drift.currentDuration}</b> 天
            {drift.durationDrift !== 0 ? (
              <span className={drift.durationDrift > 0 ? 'sched-critical-text' : 'sched-drift-good'}>
                （{fmtDrift(drift.durationDrift)} 天，{drift.durationDrift > 0 ? '拖后' : '提前'}）
              </span>
            ) : (
              <span className="sched-dim">（与基线持平）</span>
            )}
          </div>
          <div className="sched-dim">
            推移 {drift.shiftedCount} · 新增 {drift.addedCount} · 移除 {drift.removedCount} · 一致 {drift.sameCount}
          </div>
          {(drift.criticalGained.length > 0 || drift.criticalLost.length > 0) && (
            <div style={{ fontSize: 12, display: 'grid', gap: 2 }}>
              {drift.criticalGained.length > 0 && <div><span className="sched-critical-text">新进关键</span>：{drift.criticalGained.join('、')}</div>}
              {drift.criticalLost.length > 0 && <div><span className="sched-drift-good">退出关键</span>：{drift.criticalLost.join('、')}</div>}
            </div>
          )}
          {drift.rows.length > 0 && (
            <div style={{ maxHeight: 180, overflow: 'auto', display: 'grid', gap: 3 }}>
              {drift.rows.map((r) => (
                <div key={r.id} style={{ display: 'flex', gap: 6, fontSize: 12, alignItems: 'center' }}>
                  <span className={`sched-base-chip sched-base-${r.kind}`}>{DRIFT_KIND_LABEL[r.kind]}</span>
                  <span style={{ flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{r.name}</span>
                  <span className="sched-dim">
                    {r.kind === 'removed'
                      ? `原第 ${r.base!.es}~${r.base!.ef} 工作日`
                      : r.kind === 'added'
                        ? `第 ${r.now!.es}~${r.now!.ef} 工作日`
                        : `第 ${r.base!.es}→${r.now!.es} 工作日`}
                  </span>
                </div>
              ))}
            </div>
          )}
          {guard && <span className="sched-dim">暂不可更新：{guard}</span>}
          <Space size={6}>
            <Button size="small" type="primary" ghost disabled={!!guard} onClick={updateActive}>更新基线</Button>
            <Popconfirm
              title="清除当前对比？"
              description="只结束对比，基线槽位全部保留，可随时再激活"
              onConfirm={() => { clearBaseline(); setErr(null) }}
            >
              <Button size="small" danger>清除对比</Button>
            </Popconfirm>
          </Space>
        </div>
      )}

      {/* 活跃基线存在但 CPM 未通过：漂移不可算，诚实给守卫原因 */}
      {active && !drift && guard && <span className="sched-dim">暂不可对比：{guard}</span>}

      {err && <span className="sched-critical-text">{err}</span>}
    </div>
  )
}
