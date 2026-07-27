/**
 * 模型引擎 API
 * 封装所有模型引擎后端调用
 */

// ── 类型定义 ────────────────────────────────────────────────

export interface ModelInfo {
  id: string
  owned_by: string
  status: string
}

export interface EngineConfig {
  id: string
  name: string
  type: 'xai' | 'ollama' | 'herdsman' | 'deepseek'
  base_url: string
  enabled: boolean
  default_model: string
  models: ModelInfo[]
}
export interface EngineStatus {
  id: string
  connected: boolean
  model_count: number
  error: string
  last_checked: string
}

// ── API 函数 ─────────────────────────────────────────────────

const App = (): any => (window as any).go?.app?.App

/** 获取所有引擎配置 */
export async function getEngines(): Promise<EngineConfig[]> {
  const result = await App().GetEngines()
  return result as EngineConfig[]
}

/** 保存引擎配置 */
export async function saveEngine(cfg: EngineConfig): Promise<void> {
  await App().SaveEngine(cfg)
}

/** 测试引擎连接 */
export async function testEngineConnection(engineID: string): Promise<EngineStatus> {
  const result = await App().TestEngineConnection(engineID)
  return result as EngineStatus
}

/** 刷新引擎模型列表 */
export async function refreshEngineModels(engineID: string): Promise<ModelInfo[]> {
  const result = await App().RefreshEngineModels(engineID)
  return result as ModelInfo[]
}

/** 设置引擎默认模型 */
export async function setEngineDefaultModel(engineID: string, modelName: string): Promise<void> {
  await App().SetEngineDefaultModel(engineID, modelName)
}

/** 切换活跃引擎 */
export async function setActiveEngine(engineID: string): Promise<void> {
  await App().SetActiveEngine(engineID)
}

/** 获取当前活跃引擎 */
export async function getActiveEngine(): Promise<string> {
  const result = await App().GetActiveEngine()
  return result as string
}

/** 设置 DeepSeek API Key */
export async function setDeepseekKey(apiKey: string): Promise<void> {
  await App().SetDeepseekKey(apiKey)
}

/** 获取 DeepSeek Key 状态（脱敏显示） */
export async function getDeepseekKeyStatus(): Promise<{ configured: boolean; maskedKey: string }> {
  const result = await App().GetDeepseekKeyStatus()
  return result as { configured: boolean; maskedKey: string }
}
