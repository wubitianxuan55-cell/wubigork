/**
 * 图片生成 API
 * 封装所有后端图片调用，消除 (window as any)
 */

import type { GenResult } from '../components/imagegen/types'
import * as App from '../../wailsjs/go/app/App'

export interface BackendInfo {
  backend: string
  model?: string
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
      return cf.characters.map((c: any) => ({ id: c.id, name: c.name }))
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
  } catch (e: any) {
    return { list: [], error: e?.message || 'LoRA 列表加载失败' }
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
    const images: GenResult[] = res.images.map((img: any) => ({
      image: img.image, seed: img.seed, time: img.time,
      prompt: img.prompt || prompt, model: img.model || model,
      size: img.size || size, negative: negative,
    }))
    return { images }
  }
  return {}
}

/** 取消当前正在执行的图片/视频生成任务 */
export async function cancelImageGeneration(): Promise<boolean> {
  return App.CancelImageGeneration()
}

export interface ComfyTaskProgress {
  status: string
  elapsed: number
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
}

/** 多模式媒体生成（绘梦页：图生图 / 文生视频） */
export async function generateMedia(
  params: MediaParams,
): Promise<{ error?: string; results?: GenResult[]; mode?: string }> {
  const res = await App.GenerateMedia(JSON.stringify(params))
  if (res?.error) return { error: res.error }
  if (res?.results?.length) {
    const results: GenResult[] = res.results.map((img: any) => ({
      image: img.image,
      seed: img.seed,
      time: img.time,
      prompt: img.prompt || params.prompt,
      model: img.model || params.model,
      size: img.size || params.size,
      negative: params.negative,
      kind: img.kind || 'image',
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
