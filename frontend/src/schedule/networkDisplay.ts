/**
 * schedule/networkDisplay.ts — 图面内容四开关与几何参数的唯一解析入口（ZPert W4）
 *
 * 斑马把「图上显示」做成四个独立勾选（节点时间 / 工期 / 六参数 / 起止时间），
 * 与节点形状、箭头样式、名称排布同属 `network` 状态块。gaea 侧只保留解析与
 * 缺省值，渲染由各视图消费；功能区面板、图面组件、测试全部走这一份，
 * 禁止各自写 `style?.showX ?? true` 的第四份缺省。
 */
import type { SchedNetworkStyle } from './types'

/** 图面内容四开关（斑马 `show_*` 四键）。 */
export interface NetworkDisplayFlags {
  /** 节点时间：双代号节点框内的最早/最迟时间 */
  showDotTime: boolean
  /** 工期：关系线中段标注的工期/时距 */
  showDuration: boolean
  /** 六参数：节点旁的 ES/EF/LS/LF/TF/FF 六行 */
  showSixPara: boolean
  /** 起止时间：工作节点上的计划开始/完成 */
  showStartStopTime: boolean
}

/**
 * 首装口径：节点时间与工期默认开（斑马首装即显示这两项），
 * 六参数与起止时间默认关（信息密度太高，需用户主动打开）。
 */
export const NETWORK_DISPLAY_DEFAULTS: NetworkDisplayFlags = {
  showDotTime: true,
  showDuration: true,
  showSixPara: false,
  showStartStopTime: false,
}

/** 图面几何参数（缺省值对齐现行 AOA 渲染口径，改这里即改全部视图）。 */
export interface NetworkGeometry {
  dotStyle: 'circle' | 'rect'
  arrowStyle: 'solid' | 'hollow'
  namePos: 'auto' | 'horizontal' | 'vertical'
  /** 节点半径（px） */
  nodeRadius: number
  /** 箭头头部高度（px） */
  headHeight: number
  /** 线条宽度（px） */
  lineWidth: number
}

export const NETWORK_GEOMETRY_DEFAULTS: NetworkGeometry = {
  dotStyle: 'circle',
  arrowStyle: 'solid',
  namePos: 'auto',
  nodeRadius: 16,
  headHeight: 9,
  lineWidth: 1.5,
}

/** 四开关解析：显式 false 必须保留，所以只认 `typeof === 'boolean'`。 */
export function resolveNetworkDisplay(style?: SchedNetworkStyle | null): NetworkDisplayFlags {
  return {
    showDotTime: typeof style?.showDotTime === 'boolean' ? style.showDotTime : NETWORK_DISPLAY_DEFAULTS.showDotTime,
    showDuration: typeof style?.showDuration === 'boolean' ? style.showDuration : NETWORK_DISPLAY_DEFAULTS.showDuration,
    showSixPara: typeof style?.showSixPara === 'boolean' ? style.showSixPara : NETWORK_DISPLAY_DEFAULTS.showSixPara,
    showStartStopTime:
      typeof style?.showStartStopTime === 'boolean' ? style.showStartStopTime : NETWORK_DISPLAY_DEFAULTS.showStartStopTime,
  }
}

/** 几何参数解析：数值越界一律回落缺省（normalize 已在落盘侧夹取，这里兜运行期内存值）。 */
export function resolveNetworkGeometry(style?: SchedNetworkStyle | null): NetworkGeometry {
  const radius = style?.nodeRadius
  const head = style?.headHeight
  const line = style?.lineWidth
  return {
    dotStyle: style?.dotStyle ?? NETWORK_GEOMETRY_DEFAULTS.dotStyle,
    arrowStyle: style?.arrowStyle ?? NETWORK_GEOMETRY_DEFAULTS.arrowStyle,
    namePos: style?.namePos ?? NETWORK_GEOMETRY_DEFAULTS.namePos,
    nodeRadius:
      typeof radius === 'number' && Number.isFinite(radius) && radius >= 4 && radius <= 200
        ? radius
        : NETWORK_GEOMETRY_DEFAULTS.nodeRadius,
    headHeight:
      typeof head === 'number' && Number.isFinite(head) && head >= 2 && head <= 60
        ? head
        : NETWORK_GEOMETRY_DEFAULTS.headHeight,
    lineWidth:
      typeof line === 'number' && Number.isFinite(line) && line >= 0.5 && line <= 8
        ? line
        : NETWORK_GEOMETRY_DEFAULTS.lineWidth,
  }
}

/** 六参数一行文案（渲染层共用，避免两个视图各写一套顺序）。 */
export function sixParaLines(p: {
  es: string
  ef: string
  ls: string
  lf: string
  tf: string
  ff: string
}): string[] {
  return [`ES ${p.es}`, `EF ${p.ef}`, `LS ${p.ls}`, `LF ${p.lf}`, `TF ${p.tf}`, `FF ${p.ff}`]
}
