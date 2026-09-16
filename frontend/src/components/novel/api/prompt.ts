/**
 * 提示词工坊 API（t6 首刀：模板可编辑覆盖层）
 * 封装 PromptTemplate* 五个后端绑定（Go NovelB 门面，bindings_novel.go），
 * character.ts 同款 `app.*` 直调范式；结构化载荷（struct 返回）直接消费，
 * 请求体走 JSON.stringify 序列化传参（generateCharacterFill 先例）。
 */

import { app } from '../../../gaea/lib/bridge'

/** 保存/预览校验问题（Go promptstore.Issue；severity: error 阻断 / warn 提示）。 */
export interface PromptTemplateIssue {
  code: string
  severity: 'error' | 'warn' | string
  message: string
}

/** 模板元信息（列表行；Go PromptTemplateMeta，正文不下发）。 */
export interface PromptTemplateMeta {
  key: string
  category: string
  description: string
  /** "override" | "builtin"：生效来源（覆盖激活=override）。 */
  source: 'override' | 'builtin' | string
  /** 覆盖行存在（无论启停）。 */
  hasOverride: boolean
  /** 覆盖行存在且启用。 */
  overrideActive: boolean
  /** 生效版本（覆盖=保存次数，内置=0）。 */
  version: number
  /** 覆盖最近保存毫秒；内置 0。 */
  updatedAt: number
}

/** 模板正文（Go prompt.Template 的宽松前端视图；全字段可缺省防御）。 */
export interface PromptTemplateView {
  name?: string
  system?: string
  task?: string
  /** 输出说明（description 字段为面板「输出说明」编辑项）。 */
  output?: { description?: string }
  constraints?: string[]
  inputs?: unknown[]
  version?: string
  category?: string
  description?: string
  /** 声明的 {{name}} 占位符名（预览变量输入列数据源）。 */
  parameters?: string[]
}

/** 模板详情：生效模板 + 磁盘/embed 内置基线（对照/恢复参照）。 */
export interface PromptTemplateDetail {
  meta: PromptTemplateMeta
  template: PromptTemplateView
  base: PromptTemplateView
}

/** 保存结果（error 不落盘 saved=false；warn 落盘带回 issues）。 */
export interface PromptSaveResult {
  saved: boolean
  issues: PromptTemplateIssue[]
  /** 保存后版本；未保存 0。 */
  version: number
}

/** 预览结果（V1 只预览 system 段；warnings=未解析变量名）。 */
export interface PromptPreviewResult {
  systemPrompt: string
  warnings: string[]
}

/** 保存请求体（Go PromptTemplateSave 的 reqJSON 结构）。 */
export interface PromptSaveRequest {
  /** 缺省 true：false=保存覆盖行但不启用（回落内置）。 */
  isActive: boolean
  category: string
  description: string
  content: PromptTemplateView
}

/** 拉取全部模板元信息（打开面板时一次，零轮询）。 */
export async function listTemplates(): Promise<PromptTemplateMeta[]> {
  return (await app.PromptTemplateList()) as unknown as PromptTemplateMeta[]
}

/** 拉取单模板详情（生效模板 + 内置基线）。 */
export async function getTemplate(key: string): Promise<PromptTemplateDetail> {
  return (await app.PromptTemplateGet(key)) as unknown as PromptTemplateDetail
}

/** 保存覆盖（整模板覆盖；请求体 JSON 序列化传参）。 */
export async function saveTemplate(key: string, req: PromptSaveRequest): Promise<PromptSaveResult> {
  return (await app.PromptTemplateSave(key, JSON.stringify(req))) as unknown as PromptSaveResult
}

/** 删除覆盖行，回落内置（不存在也幂等成功）。 */
export async function resetTemplate(key: string): Promise<void> {
  await app.PromptTemplateReset(key)
}

/** 渲染预览（未保存草稿的 system 段 {{name}} 替换；缺失变量记名不抛错）。 */
export async function previewTemplate(
  req: { content: PromptTemplateView },
  vars: Record<string, string>,
): Promise<PromptPreviewResult> {
  return (await app.PromptTemplatePreview(JSON.stringify(req), JSON.stringify(vars))) as unknown as PromptPreviewResult
}

// ── t6-C2 模板包导入导出（规格 进度计划/gaea-prompt-bundle-t6c2-20260916.md §5）──

/** 导入结果（Go promptstore.BundleImportResult；action 取值见 ACTION_*）。 */
export interface BundleImportResult {
  applied: boolean
  statistics: {
    total: number
    keptSystemDefault: number
    convertedToCustom: number
    createdOrUpdate: number
    skippedInvalid: number
    skippedUnknown: number
    skippedDuplicate: number
  }
  outcomes: Array<{ key: string; action: string; reason?: string }>
}

/** 导出模板包：回明文 JSON 字符串（调用方走 saveExportBlob 双门落盘）。 */
export async function exportBundle(): Promise<string> {
  return (await app.PromptBundleExport()) as string
}

/** 导入模板包：三态决策在 Go 侧完成，回统计与逐行结果。 */
export async function importBundle(text: string): Promise<BundleImportResult> {
  return (await app.PromptBundleImport(text)) as unknown as BundleImportResult
}
