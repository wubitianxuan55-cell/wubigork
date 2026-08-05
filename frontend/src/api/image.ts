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
export async function getComfyUILoras(): Promise<string[]> {
  try {
    const list = await App.GetComfyUILoras()
    return Array.isArray(list) ? list : []
  } catch (_) {
    return []
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
