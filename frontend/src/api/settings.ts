/**
 * 设置数据 API
 * 封装所有后端设置调用，消除 (window as any) 和 @ts-ignore
 */

import { app } from '../gaea/lib/bridge'

export interface BackendInfo {
  backend: string
  model?: string
  image_model?: string
  comfyui_url?: string
  image_save_dir?: string
  comfyui_path?: string
  comfyui_python_path?: string
}


/** 获取全部配置 */
export async function getConfig(): Promise<Record<string, string>> {
  const cfg = await app.GetConfig()
  return cfg as Record<string, string>
}

/** 保存单个配置项 */
export async function saveConfig(key: string, value: string): Promise<void> {
  await app.SaveConfig(key, value)
}

/** 获取图片后端信息 */
export async function getImageBackendInfo(): Promise<BackendInfo> {
  const info = await app.GetImageBackendInfo()
  return info as unknown as BackendInfo
}

/** 设置图片后端 */
export async function setImageBackend(backend: string, url: string, model: string, saveDir: string): Promise<void> {
  await app.SetImageBackend(backend, url, model, saveDir)
}

/** 获取图片后端信息 */
export async function getActiveModel(): Promise<string> {
  const m = await app.GetActiveModel()
  return (m as string) || ''
}

/** 获取语音设置 */
export async function getVoiceSettings(): Promise<Record<string, unknown>> {
  const v = await app.VoiceGetSettings()
  return v || {}
}

/** 应用语音设置补丁 */
export async function applyVoiceSettings(patch: Record<string, unknown>): Promise<void> {
  await app.VoiceApplySettings(patch)
}

/** 获取办公引擎设置摘要（GaeaSettings） */
export async function gaeaSettings(): Promise<Record<string, unknown>> {
  const v = await app.Settings()
  return (v as unknown as Record<string, unknown>) || {}
}