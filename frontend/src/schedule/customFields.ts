/**
 * customFields.ts — 自定义字段注册表（纯函数，v4.138 #14）
 *
 * 对标 MS Project 自定义字段（蒸馏 ProjectLibre X2 机制：固定槽位 + 字段
 * 注册表，列显隐/表格编辑/xlsx 往返共用同一份元数据）。工程场景：施工部位/
 * 专业/分包单位/验收状态等表外列。
 *
 * 口径：
 *  - 固定 5 槽（文本×3 + 数值×2），key 稳定不可变（text1..num2），旧文件零迁移；
 *  - 列名可按项目覆盖（project.customLabels[key]），缺省用注册表 label——
 *    xlsx 导入导出与表格列头同口径（Go 侧 internal/schedule 镜像同一注册表）；
 *  - 任务只存有值槽（task.custom[key]），值类型 text=string / number=number。
 */
import type { SchedTask } from './types'

export type CustomFieldKey = 'text1' | 'text2' | 'text3' | 'num1' | 'num2'
export type CustomFieldType = 'text' | 'number'

export interface CustomFieldDef {
  key: CustomFieldKey
  /** 缺省列名（项目可用 customLabels 覆盖） */
  label: string
  type: CustomFieldType
}

/** 注册表（顺序=xlsx 导出列序；勿改动既有 key/label——会破坏存量 xlsx 识别） */
export const CUSTOM_FIELDS: readonly CustomFieldDef[] = [
  { key: 'text1', label: '文本1', type: 'text' },
  { key: 'text2', label: '文本2', type: 'text' },
  { key: 'text3', label: '文本3', type: 'text' },
  { key: 'num1', label: '数值1', type: 'number' },
  { key: 'num2', label: '数值2', type: 'number' },
] as const

/** 列名：项目覆盖优先，缺省注册表 label（覆盖为空串/非字符串时同样回落） */
export function customLabelOf(key: CustomFieldKey, customLabels?: Record<string, string>): string {
  const def = CUSTOM_FIELDS.find((d) => d.key === key)
  const override = customLabels?.[key]
  return typeof override === 'string' && override.trim() !== '' ? override.trim() : def?.label ?? key
}

/** 任务在槽上的值（无值/类型不符返回 undefined） */
export function customValueOf(task: SchedTask, key: CustomFieldKey): string | number | undefined {
  const v = task.custom?.[key]
  const def = CUSTOM_FIELDS.find((d) => d.key === key)
  if (def?.type === 'number') return typeof v === 'number' && Number.isFinite(v) ? v : undefined
  return typeof v === 'string' ? v : undefined
}
