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
// ── 网络搜索主源（众神殿刀3 主源向导）──
// key 本体落 ~/.env（与 CLI 同源，boot loadDotEnv 注入），配置面只持有 env
// 变量名；保存即热生效（builtin SetSearchConfig 重载，无需重启）。

export interface WebSearchSettingsView {
  local_searxng_url: string
  tavily_api_key_env: string
  brave_api_key_env: string
  tavily_ready: boolean
  brave_ready: boolean
  searxng_ready: boolean
  has_primary: boolean
  engine_order?: string[]
}

export interface WebSearchTestEngine {
  name: string
  outcome: string
  elapsed_ms: number
}

/** 获取搜索主源设置（可用性按解析后的 key 归出；不含密钥本体） */
export async function webSearchSettings(): Promise<WebSearchSettingsView> {
  const v = await app.WebSearchSettingsGet()
  return (v || {}) as unknown as WebSearchSettingsView
}

/** 保存搜索主源：空串=清除该源；保存后立即生效 */
export async function saveWebSearchSettings(tavilyKey: string, braveKey: string, searxngURL: string): Promise<void> {
  await app.WebSearchSettingsSet(tavilyKey, braveKey, searxngURL)
}

export interface WebSearchTestResult {
  ok: boolean
  state: string
  degraded_reason?: string
  elapsed_ms: number
  engines: WebSearchTestEngine[]
  sample: string[]
  error?: string
  hint?: string
}

/** 主源连通性实测：真实走统一检索核出网一次，按状态+逐引擎诊断回执 */
export async function testWebSearchConnection(): Promise<WebSearchTestResult> {
  const v = await app.WebSearchTestConnection()
  return (v || {}) as unknown as WebSearchTestResult
}
