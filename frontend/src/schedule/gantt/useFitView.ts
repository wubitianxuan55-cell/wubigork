/**
 * schedule/gantt/useFitView.ts — 网络图缩放外壳单源（FE7-12）
 *
 * AoaView 与 PdmView 此前各自内联同一套缩放壳：zoom/zoomRef/touchedRef 三件 +
 * applyZoom（四舍五入到 0.01 后钳位）/ stepZoom（置 touched + 乘因子）+
 * fitView（视口适配）+ 自动适配 effect（用户未手动缩放才抢占）。本模块把外壳
 * 收敛为 useFitView + useAutoFitZoom，各视图注入自己的钳域与适配公式——
 * 上下限不一致是刻意的（AOA 图更密：clamp [0.1,2]、全览下限 0.3、自动适配
 * 下限 0.5、上限 1.5；PDM：clamp [0.2,2]、适配下限 0.2、上限 1），参数值冻结、
 * 不许顺手统一。GanttView 的 dayW 缩放（2~40px、锚点回写 scrollLeft）是另一套
 * 单位与机制，不并入本壳。
 *
 * 适配公式注入约定：zoomOf(el) 返回目标 zoom；返回 null=本帧不适配（视图自含
 * 自己的 guard 口径：AOA fitView 有 clientWidth>0 门槛而 PDM fitView 没有——
 * 后者此时按公式钳到下限 0.2，均与拆分前逐字段一致）。
 */
import React, { useRef, useState } from 'react'

/** 缩放外壳控制器（视图消费：zoom 读数/ref/手动缩放/全览/自动适配） */
export interface FitViewController {
  zoom: number
  zoomRef: React.MutableRefObject<number>
  touchedRef: React.MutableRefObject<boolean>
  /** 钳位缩放：round2 后钳 [min,max]（视图注入），同步镜像 zoomRef */
  applyZoom: (z: number) => void
  /** 步进缩放：置 touched（手动缩放后自动适配不再抢占）+ 乘因子 */
  stepZoom: (f: number) => void
  /** 全览：zoomOf=视口→目标 zoom（null=不适配）；touched 由调用方先置
   *  （与拆分前两视图的按钮 onClick 一致） */
  fitView: (zoomOf: (el: HTMLElement) => number | null) => void
}

/**
 * 缩放外壳（FE7-12 单源）：min/max=applyZoom 钳域（每视图注入，不统一）。
 * 必须在视图 early return 之前调用（rules-of-hooks）。
 */
export function useFitView(
  scrollRef: React.RefObject<HTMLElement | null>,
  min: number,
  max: number,
): FitViewController {
  const [zoom, setZoom] = useState(1)
  const zoomRef = useRef(1)
  const touchedRef = useRef(false)
  const applyZoom = (z: number): void => {
    const c = Math.min(max, Math.max(min, Math.round(z * 100) / 100))
    zoomRef.current = c
    setZoom(c)
  }
  const stepZoom = (f: number): void => {
    touchedRef.current = true
    applyZoom(zoomRef.current * f)
  }
  const fitView = (zoomOf: (el: HTMLElement) => number | null): void => {
    const el = scrollRef.current
    if (!el) return
    const z = zoomOf(el)
    if (z !== null) applyZoom(z)
  }
  return { zoom, zoomRef, touchedRef, applyZoom, stepZoom, fitView }
}

/**
 * 自动适配 effect（FE7-12 单源）：内容尺寸变化（key 变化）且用户未手动缩放
 * （touched=false）且内容就绪（guard=true）时适配一次。zoomOf 返回非有限/≤0/
 * null=本帧不适配（AOA 原有 Number.isFinite 防御在此口径内保留）。
 * key/guard 之外不发依赖（zoomOf 每渲染重建，进依赖会让自动适配每帧重放抢
 * touched——故对 exhaustive-deps 出豁免，同 GanttView 前锋线先例）。
 */
export function useAutoFitZoom(
  scrollRef: React.RefObject<HTMLElement | null>,
  touchedRef: React.RefObject<boolean>,
  applyZoom: (z: number) => void,
  zoomOf: (el: HTMLElement) => number | null,
  key: unknown,
  guard: boolean,
): void {
  React.useEffect(() => {
    if (touchedRef.current || !guard) return
    const el = scrollRef.current
    if (!el) return
    const z = zoomOf(el)
    if (z !== null) applyZoom(z)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, guard])
}
