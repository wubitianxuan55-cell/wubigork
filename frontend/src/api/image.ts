/**
 * 图片生成 API
 * 封装所有后端图片调用，消除 (window as any)
 */

import type { GenResult } from '../components/imagegen/types'
import * as App from '../../src/wailsjsCompat'
import { app as bridgeApp } from '../gaea/lib/bridge'

// 三态回退（v4.58 wailsjsCompat 消费族模式）：?mock=1 下 window.go 刻意为空，
// wailsjsCompat 直调绕过 bridge mock——登记/清单读取统一经此 helper 走
// window.go.app.App 兼容代理 + bridgeApp mock 兜底（ImageHubAssets/
// ChapterArtList 已转正 AppBindings，mock 样例见 gaea/lib/mock/imagehub.ts）。
// T1 识图试用/创作资产追加：落盘 GaeaSavePastedImage、识图 GaeaRecognizeImage、
// OCR GaeaOCRText（mock 见 gaea/lib/mock/office.ts）、角色 CharacterList
// （mock 见 gaea/lib/mock/weixin.ts）——全部既有绑定，零新绑定。
type ImageHubFacade = {
  ImageHubAssets(space: string, sourceBoard: string, limit: number): Promise<Array<Record<string, unknown>>>
  ChapterArtList(chapterNum: number): Promise<Array<Record<string, unknown>>>
  AttachmentDataURL(path: string): Promise<string>
  SavePastedImage(dataUrl: string): Promise<string>
  RecognizeImage(imagePath: string, prompt: string): Promise<unknown>
  OCRText(imagePath: string): Promise<unknown>
  CharacterList(query: string, kind: string, chatOnly: boolean, page: number, pageSize: number):
    Promise<{ items?: Array<Record<string, unknown>>; total?: number }>
  HerdsmanModelCatalog(): Promise<HerdsmanCatalogView>
}
const appFacade = (): ImageHubFacade => (window.go?.app?.App ?? bridgeApp) as unknown as ImageHubFacade

export interface BackendInfo {
  backend: string
  model?: string
  image_model?: string
  comfyui_url?: string
  image_save_dir?: string
  comfyui_path?: string
  comfyui_python_path?: string
}

export interface SystemStats {
  cpu: number
  memTotal: number
  memUsed: number
  gpuName: string
  gpuUsage: number
  vramUsed: number
  vramTotal: number
}

export interface ComfyUIStatus {
  running: boolean
  port?: number
  url?: string
}

/** 后端返回的图片结果条目（动态载荷的最小消费面） */
interface GenImageLike {
  image?: string
  seed?: number
  time?: number
  prompt?: string
  model?: string
  size?: string
  kind?: string
  file_path?: string
}

/** 获取后端信息 */
export async function getImageBackendInfo(): Promise<BackendInfo> {
  const info = await App.GetImageBackendInfo()
  return info as unknown as BackendInfo
}

/** 获取角色列表 */
export async function getCharacters(): Promise<{ id: string; name: string }[]> {
  try {
    const cf = await App.GetCharacters()
    if (cf?.characters) {
      return cf.characters.map((c: { id?: string; name?: string }) => ({ id: c.id ?? '', name: c.name ?? '' }))
    }
  } catch (_) {}
  return []
}

/** 获取 ComfyUI 状态 */
export async function getComfyUIStatus(): Promise<ComfyUIStatus> {
  const s = await App.GetComfyUIStatus()
  return s as unknown as ComfyUIStatus
}

/** 获取 ComfyUI 当前可用的 LoRA 列表（models/loras 相对路径，含子目录） */
export interface ComfyLorasResult {
  list: string[]
  error?: string
}

export async function getComfyUILoras(): Promise<ComfyLorasResult> {
  try {
    const list = await App.GetComfyUILoras()
    return { list: Array.isArray(list) ? list : [] }
  } catch (e: unknown) {
    return { list: [], error: e instanceof Error ? e.message : 'LoRA 列表加载失败' }
  }
}

/** 获取系统状态 */
export async function getSystemStats(): Promise<SystemStats | null> {
  try {
    const s = await App.GetSystemStats()
    return s as unknown as SystemStats
  } catch (_) {
    return null
  }
}

/** 获取角色库剧照独立后端/模型（空 = 跟随绘梦） */
export async function getPortraitConfig(): Promise<{ backend: string; model: string }> {
  try {
    const r = await App.GetPortraitConfig()
    return (r || { backend: '', model: '' }) as { backend: string; model: string }
  } catch (_) {
    return { backend: '', model: '' }
  }
}

/** 设置角色库剧照独立后端/模型（空 = 跟随绘梦） */
export async function setPortraitConfig(backend: string, model: string): Promise<void> {
  await App.SetPortraitConfig(backend, model)
}

/** 生成图片 */
export async function generateImage(
  prompt: string, negative: string, size: string,
  model: string, seed: number, count: number,
  lora?: string,
): Promise<{ error?: string; images?: GenResult[] }> {
  const res = await App.GenerateFreeImage(prompt.trim(), negative.trim(), size, '', model, seed, count, lora || '')
  if (res?.error) return { error: res.error }
  if (res?.images?.length) {
    const images: GenResult[] = res.images.map((img: GenImageLike) => ({
      image: img.image as string, seed: img.seed as number, time: img.time as number,
      prompt: img.prompt || prompt, model: img.model || model,
      size: img.size || size, negative: negative,
      file_path: img.file_path || undefined,
    }))
    return { images }
  }
  return {}
}

/** 取消当前正在执行的图片/视频生成任务 */
export async function cancelImageGeneration(): Promise<boolean> {
  return App.CancelImageGeneration()
}

/**
 * 通过后端文件读取绑定把本地路径转为 data URL（历史图片恢复 / 下载 / 剧照）。
 * 复用现有 GaeaAttachmentDataURL（OfficeB 门面），不新增绑定。
 */
export async function readFileAsDataURL(path: string): Promise<string> {
  // 经 appFacade：mock 下落到 office.ts 的 AttachmentDataURL 占位色块。
  return appFacade().AttachmentDataURL(path)
}

/** 图像域登记视图（ImageHubAssets 绑定，T1 画室素材库）。 */
export interface ImageHubAssetView {
  id?: string
  kind?: string
  path?: string
  mime?: string
  space?: string
  source_board?: string
  capability?: string
  backend?: string
  model?: string
  cost?: string
  created_at?: string
  prompt_truncate?: string
  params?: Record<string, unknown>
}

/** 章节插图清单条目（ChapterArtList 绑定，T1）。 */
export interface ChapterArtEntry {
  chapter?: number
  asset_id?: string
  path?: string
  created_at?: string
}

/** 画室素材读取：按空间/来源筛选（失败 = 空列表，登记是辅助视图）。 */
export async function imageHubAssets(space: string, sourceBoard: string, limit: number): Promise<ImageHubAssetView[]> {
  try {
    const res = await appFacade().ImageHubAssets(space, sourceBoard, limit)
    return Array.isArray(res) ? res as unknown as ImageHubAssetView[] : []
  } catch (_) {
    return []
  }
}

/** 章节插图清单读取（失败 = 空列表，不阻断主流程）。 */
export async function chapterArtList(chapterNum: number): Promise<ChapterArtEntry[]> {
  try {
    const res = await appFacade().ChapterArtList(chapterNum)
    return Array.isArray(res) ? res as unknown as ChapterArtEntry[] : []
  } catch (_) {
    return []
  }
}

// ── T1 识图「读/懂」画室试用（零新绑定，原语 vision.read / vision.understand）──

/**
 * 粘贴/选择的图片落盘（GaeaSavePastedImage），返回可传给识图原语的本地路径。
 * 失败向上抛，由调用方诚实呈现错误原文。
 */
export async function savePastedImage(dataUrl: string): Promise<string> {
  return appFacade().SavePastedImage(dataUrl)
}

/** 识图调用结果规范视图：文本必给，模型名仅当返回里确实携带时才有。 */
export interface VisionCallResult {
  text: string
  model?: string
}

/** 归一化识图返回：字符串直取；对象按 text/content/description 取文本、model 取模型名。 */
function normalizeVisionResult(raw: unknown): VisionCallResult {
  if (typeof raw === 'string') return { text: raw }
  if (raw && typeof raw === 'object') {
    const o = raw as Record<string, unknown>
    const text = [o.text, o.content, o.description].find((v): v is string => typeof v === 'string')
    const model = typeof o.model === 'string' ? o.model : undefined
    if (text !== undefined) return { text, model }
    return { text: JSON.stringify(o), model }
  }
  return { text: String(raw ?? '') }
}

/** 识图-懂（GaeaRecognizeImage）：内容理解/描述，prompt 由调用方自填。 */
export async function visionUnderstand(imagePath: string, prompt: string): Promise<VisionCallResult> {
  return normalizeVisionResult(await appFacade().RecognizeImage(imagePath, prompt))
}

/** 识图-读（GaeaOCRText）：提取图内文字。 */
export async function visionRead(imagePath: string): Promise<VisionCallResult> {
  return normalizeVisionResult(await appFacade().OCRText(imagePath))
}

// ── T1 创作资产 · 角色槽（CharacterList 只读分页，零新绑定）──

/** 可聊天角色卡最小消费面（对齐 characterlib.Character 的展示字段）。 */
export interface ChatCharacterView {
  id?: string
  name?: string
  kind?: string
  gender?: string
  tags?: string[]
  portraitUrl?: string
  roleType?: string
  personality?: string
  background?: string
  appearance?: string
  referenceImages?: string[]
}

/** 分页拉取可聊天角色（chatOnly=true；失败 = 空列表，浏览是辅助视图）。 */
export async function chatCharacters(page: number, pageSize: number):
  Promise<{ items: ChatCharacterView[]; total: number }> {
  try {
    const res = await appFacade().CharacterList('', '', true, page, pageSize)
    const items = Array.isArray(res?.items)
      ? res.items as unknown as ChatCharacterView[]
      : []
    return { items, total: typeof res?.total === 'number' ? res.total : items.length }
  } catch (_) {
    return { items: [], total: 0 }
  }
}

// ── T1 画室「模型目录」创作语境视图（HerdsmanModelCatalog 只读，零新绑定）──
// HerdsmanModelCatalog 与 Get/SetEngineFailover 同为 legacy 绑定面（挂在后端
// ModelB 门面；模型中心「模型库」段经 api/engines.ts 消费同名绑定）。这里按
// appFacade 三态回退（window.go.app.App 兼容代理 + bridge mock 兜底）局部重述
// 返回形状（对齐 internal/app/herdsman_catalog.go 的 HerdsmanCatalog /
// HerdsmanCatalogModel json 字段）；不从 api/engines.ts import（避免 api↔api
// 环），字段漂移由消费方诚实留空 + mock 契约测试兜底。

/** 模型目录单条记录（对齐 Go HerdsmanCatalogModel json 字段；缺省 = 目录未携带）。 */
export interface HerdsmanCatalogModelView {
  name?: string
  display_name?: string
  type?: string
  runtime?: string
  inference_engines?: string[]
  capabilities?: string[]
  installed?: boolean
  running?: boolean
  status?: string
  run_status?: string
  quantization?: string
  parameter_count?: number
  active_parameters?: number
  is_moe?: boolean
  file_size?: number
  llama_cpp_variants?: string[]
  /** 本机实测/受控测评给出的用途建议（Go 侧按模型名/能力映射）。 */
  hint?: string
}

/** 模型目录完整载荷（对齐 Go HerdsmanCatalog；error 非空 = 目录来源异常）。 */
export interface HerdsmanCatalogView {
  models?: HerdsmanCatalogModelView[]
  total?: number
  installed?: number
  running?: number
  source?: string
  error?: string
}

/**
 * 画室模型目录只读读取（画室「模型目录」tab 数据源）。
 * 不做二次加工、不补默认值：目录未携带的档位/成本字段原样缺省，由视图层
 * 诚实显示「未定价」/「目录未标注」；调用失败向上抛，由视图呈现错误原文。
 */
export async function herdsmanModelCatalog(): Promise<HerdsmanCatalogView> {
  const res = await appFacade().HerdsmanModelCatalog()
  return (res && typeof res === 'object' ? res : {}) as HerdsmanCatalogView
}

export interface ComfyTaskProgress {
  status: string
  elapsed: number
  /** 0-100 实时进度；-1/缺省 = 未知（未接入 ComfyUI 实时进度） */
  percent?: number
  /** 当前执行节点 class_type（如 KSampler / CLIPLoader） */
  node?: string
}

/** 获取当前 ComfyUI 任务状态（前端轮询显示） */
export async function getComfyUITaskProgress(): Promise<ComfyTaskProgress> {
  const p = await App.GetComfyUITaskProgress()
  return (p || { status: '', elapsed: 0 }) as ComfyTaskProgress
}

/** 生成流程图/框架图：LLM 返回 Mermaid 代码，前端渲染为 PNG */
export async function generateDiagram(
  prompt: string,
): Promise<{ error?: string; code?: string }> {
  const res = await App.GenerateDiagram(prompt.trim())
  return (res || {}) as { error?: string; code?: string }
}

/** 多模式媒体生成参数（文生图 / 图生图 / 文生视频） */
export interface MediaParams {
  prompt: string
  negative: string
  size: string
  model: string
  seed: number
  lora: string
  count: number
  mode: 'txt2img' | 'img2img' | 't2v'
  initImage?: string
  denoise?: number
  frames?: number
  fps?: number
  /** T2 角色参考槽：角色 ID 与参考图列表（data URL；首张作图生图种子） */
  characterId?: string
  refImages?: string[]
  refMethod?: string
}

/** 多模式媒体生成（绘梦页：图生图 / 文生视频） */
export async function generateMedia(
  params: MediaParams,
): Promise<{ error?: string; results?: GenResult[]; mode?: string }> {
  const res = await App.GenerateMedia(JSON.stringify(params))
  if (res?.error) return { error: res.error }
  if (res?.results?.length) {
    const results: GenResult[] = res.results.map((img: GenImageLike) => ({
      image: img.image as string,
      seed: img.seed as number,
      time: img.time as number,
      prompt: img.prompt || params.prompt,
      model: img.model || params.model,
      size: img.size || params.size,
      negative: params.negative,
      kind: (img.kind || 'image') as 'image' | 'video',
      file_path: img.file_path || undefined,
    }))
    return { results, mode: res.mode }
  }
  return {}
}

/** 启动 ComfyUI */
/** 启动 ComfyUI */
export async function startComfyUI(): Promise<void> {
  await App.StartComfyUI()
}

/** 停止 ComfyUI */
export async function stopComfyUI(): Promise<void> {
  await App.StopComfyUI()
}

/** 打开图片保存目录 */
export async function openImageSaveDir(): Promise<void> {
  await App.OpenImageSaveDir()
}

/** 打开小说图片目录 */
export async function openNovelImagesDir(): Promise<void> {
  await App.OpenNovelImagesDir()
}

/** 设置角色剧照 */
export async function setCharacterPortrait(charID: string, imageData: string): Promise<void> {
  await App.SetCharacterPortrait(charID, imageData)
}
