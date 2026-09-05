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
type ImageHubFacade = {
  ImageHubAssets(space: string, sourceBoard: string, limit: number): Promise<Array<Record<string, unknown>>>
  ChapterArtList(chapterNum: number): Promise<Array<Record<string, unknown>>>
  AttachmentDataURL(path: string): Promise<string>
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
