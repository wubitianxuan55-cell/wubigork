/**
 * 设置数据 API
 * 封装所有后端设置调用，消除 (window as any) 和 @ts-ignore
 */

import type { TTSConfig, TTSStatus } from '../types'
import * as App from '../../src/wailsjsCompat'

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
  const cfg = await App.GetConfig()
  return cfg as Record<string, string>
}

/** 保存单个配置项 */
export async function saveConfig(key: string, value: string): Promise<void> {
  await App.SaveConfig(key, value)
}

/** 获取图片后端信息 */
export async function getImageBackendInfo(): Promise<BackendInfo> {
  const info = await App.GetImageBackendInfo()
  return info as unknown as BackendInfo
}

/** 设置图片后端（dashscopeKey 为百炼改图 API Key，仅 backend==='dashscope' 时有意义；空串 = 保持已保存 Key） */
export async function setImageBackend(backend: string, url: string, model: string, saveDir: string, dashscopeKey = ''): Promise<void> {
  await App.SetImageBackend(backend, url, model, saveDir, dashscopeKey)
}

/** 获取 TTS 配置 */
export async function getTTSConfig(): Promise<TTSConfig> {
  const cfg = await App.GetTTSConfig()
  return cfg as unknown as TTSConfig
}

/** 获取 TTS 状态 */
export async function getTTSStatus(): Promise<TTSStatus> {
  const status = await App.GetTTSStatus()
  return status as unknown as TTSStatus
}

/** 保存 TTS 配置 */
export async function saveTTSConfig(
  modelPath: string, serverPath: string, port: number, backend: string, speed: number,
): Promise<void> {
  await App.SaveTTSConfig(modelPath, serverPath, port, backend, speed)
}

/** 启动 TTS 服务 */
export async function startTTSServer(modelPath: string, port: number, backend: string): Promise<void> {
  await App.StartTTSServer(modelPath, port, backend)
}

/** 停止 TTS 服务 */
export async function stopTTSServer(): Promise<void> {
  await App.StopTTSServer()
}


/** 迁移项目到 v4 */
export async function migrateProjectToV4(): Promise<void> {
  await App.MigrateProjectToV4()
}

/** 获取当前激活的 AI 模型名 */
export async function getActiveModel(): Promise<string> {
  const m = await App.GetActiveModel()
  return (m as string) || ''
}

/** 获取语音服务健康状态 */
export async function voiceHealth(): Promise<Record<string, unknown>> {
  const h = await App.VoiceHealth?.()
  return h || { asrReady: false, ttsReady: false }
}

/** 获取语音设置 */
export async function getVoiceSettings(): Promise<Record<string, unknown>> {
  const v = await App.VoiceGetSettings?.()
  return v || {}
}

/** 应用语音设置补丁 */
export async function applyVoiceSettings(patch: Record<string, unknown>): Promise<void> {
  await App.VoiceApplySettings?.(patch)
}

/** 获取办公引擎设置摘要（GaeaSettings） */
export async function gaeaSettings(): Promise<Record<string, unknown>> {
  const v = await App.GaeaSettings()
  return (v as unknown as Record<string, unknown>) || {}
}
