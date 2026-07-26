/**
 * 设置数据 API
 * 封装所有后端设置调用，消除 (window as any) 和 @ts-ignore
 */

import type { TTSConfig, TTSStatus } from '../types'
import * as App from '../../wailsjs/go/app/App'

export interface BackendInfo {
  backend: string
  model?: string
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

/** 设置图片后端 */
export async function setImageBackend(backend: string, url: string, model: string): Promise<void> {
  await App.SetImageBackend(backend, url, model)
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
