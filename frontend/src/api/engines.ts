/**
 * 模型引擎 API
 * 封装所有模型引擎后端调用
 */

// ── 类型定义 ────────────────────────────────────────────────

export interface ModelInfo {
  id: string
  owned_by: string
  status: string
  kind?: string // 后端分类：llm / tts / stt / image / embedding / rerank / ocr
  alias_of?: string // coding 端点家族下服务端实际服务的模型（套餐旧名自动切换）；std 家族为空（Go modelengine.ModelInfo.AliasOf）
}

export interface EngineConfig {
  id: string
  name: string
  // custom = 用户自建 OpenAI 兼容服务商（A 刀），id 以 custom- 开头
  type: 'xai' | 'ollama' | 'herdsman' | 'deepseek' | 'glm' | 'cosyvoice' | 'opencode-go' | 'opencode-zen' | 'custom'
  label?: string
  color?: string
  icon?: string
  is_local?: boolean
  base_url: string
  enabled: boolean
  default_model: string
  models: ModelInfo[]
  status?: EngineStatus
}
export interface EngineStatus {
  id: string
  connected: boolean
  model_count: number
  error: string
  last_checked: string
  latency_ms?: number
}

export interface ModelUsageStats {
  engine_id: string
  model: string
  call_count: number
  success_count: number
  fail_count: number
  input_tokens: number
  output_tokens: number
  total_tokens: number
  total_duration_ms: number
  estimated_cost?: number
  currency?: string
  billing_mode?: string // 计费口径："coding_points"=GLM 编码套餐积分内调用（费用恒 0 不入总额）；空=按量计费（Go modelengine.BillingCodingPoints）
  last_error?: string
  last_called_at?: string
}

/** 按引擎聚合小计（ModelStatsSummary.engines 的值，Go modelengine.EngineSubtotal） */
export interface EngineSubtotal {
  tokens: number
  calls: number
  estimated_cost_cny: number
}

export interface TrendPoint {
  time: string
  calls: number
  success_calls: number
  fail_calls: number
  input_tokens: number
  output_tokens: number
  total_tokens: number
  cost: number
}

export interface ModelStatsSummary {
  total_calls: number
  success_calls: number
  fail_calls: number
  total_tokens: number
  input_tokens: number
  output_tokens: number
  total_duration_ms: number
  avg_duration_ms: number
  total_cost: number
  trend: TrendPoint[]
  per_model: ModelUsageStats[]
  engines?: Record<string, EngineSubtotal> // 按引擎聚合小计（编码套餐口径以 "<engine>@coding" 单列、费用 0）；旧 stats.json 无此字段
  since?: string
  usd_to_cny?: number
}

/** Herdsman 模型库条目（来自 herdsman skill models list） */
export interface HerdsmanCatalogModel {
  name: string
  display_name: string
  type: string
  runtime: string
  inference_engines?: string[]
  capabilities?: string[]
  installed: boolean
  running: boolean
  status: string
  run_status?: string
  quantization?: string
  parameter_count?: number
  active_parameters?: number
  is_moe?: boolean
  file_size?: number
  llama_cpp_variants?: string[]
  /** 用途建议（后端按受控测评结论生成，如「日常对话/识图首选」） */
  hint?: string
}

export interface HerdsmanCatalog {
  models: HerdsmanCatalogModel[]
  total: number
  installed: number
  running: number
  source: string
  error?: string
  /** 已装模型占用（字节，E1-4 磁盘治理） */
  installed_bytes?: number
  /** 数据目录所在卷总量（字节） */
  disk_total?: number
  /** 数据目录所在卷余量（字节） */
  disk_free?: number
  disk_error?: string
}

/** Herdsman 生命周期操作结果 */
export interface HerdsmanOpResult {
  ok: boolean
  status: string
  message: string
}

/** 某模型在本机的实测启动参数（launch_records） */
export interface HerdsmanLaunchPreset {
  model: string
  engine: string
  port: number
  started_at: string
  options: Record<string, unknown>
}

/** Herdsman 单模型调用聚合（来自 model_stats/events.jsonl） */
export interface HerdsmanModelStat {
  model: string
  type: string
  runtime: string
  calls: number
  succeeded: number
  failed: number
  input_tokens: number
  output_tokens: number
  total_duration_ms: number
  avg_duration_ms: number
  avg_ttft_ms: number
  avg_prompt_tps: number
  avg_predicted_tps: number
  last_called_at: string
}

export interface HerdsmanModelStats {
  total: number
  since: string
  per_model: HerdsmanModelStat[]
  source: string
  error?: string
}

// ── 阶段 3 D3-2 分流统计（本地 vs 云端） ───────────────────

export interface UsageSide {
  calls: number
  success_calls: number
  fail_calls: number
  input_tokens: number
  output_tokens: number
  total_tokens: number
  cache_hit_tokens?: number
  cache_miss_tokens?: number
  total_duration_ms: number
  cost: number
  engines: string[]
}

export interface SavingsView {
  ref_price_per_mtok: number
  would_cost_cloud: number
  saved: number
  note: string
}

export interface UsageOverview {
  cloud: UsageSide
  local: UsageSide
  savings: SavingsView
  cache_hit_tokens?: number
  cache_miss_tokens?: number
  cache_hit_rate?: number
}

// ── 阶段 3 D3-1 语义索引状态 ───────────────────────────────

export interface SemanticIndexStatus {
  available: boolean
  counts: Record<string, number>
  error?: string
}

// ── 阶段 3 D3-3 Herdsman 受控测评 ──────────────────────────

export interface BenchmarkSummary {
  total_cases: number
  succeeded: number
  failed: number
  canceled: number
  avg_duration_ms: number
  avg_ttft_ms: number
  avg_tps: number
}

export interface BenchmarkRunSummary {
  id: string
  created_at: string
  finished_at?: string
  status: string
  model_names: string[]
  variants: string[]
  context_sizes: number[]
  summary?: BenchmarkSummary
}

export interface BenchmarkPromptRequest {
  user_prompt: string
  temperature: number
  top_p: number
  top_k: number
  repeat_penalty: number
  max_tokens: number
  stream: boolean
  timeout_seconds: number
}

export interface BenchmarkRequest {
  model_names: string[]
  variants: string[]
  context_sizes: number[]
  cache_reuse_mode: string
  warmup_count: number
  repeat_count: number
  concurrency: number
  request: BenchmarkPromptRequest
}

export interface BenchmarkCase {
  model_name: string
  variant_id: string
  context_size: number
  status: string
  started_at?: string
  ended_at?: string
  duration_ms: number
  ttft_ms_avg: number
  ttft_ms_p95: number
  input_tokens: number
  output_tokens: number
  total_tokens: number
  cached_tokens: number
  second_duration_ms: number
  second_ttft_ms_avg: number
  // D3-4 富字段：缓存复用与显存参数
  prompt_tokens_tps?: number
  output_tokens_tps?: number
  prefill_speedup_ratio?: number
  prefill_ms_saved?: number
  prompt_ms?: number
  predicted_ms?: number
  response_excerpt?: string
  effective_launch_params?: Record<string, unknown>
  error?: string
}

/** 流式探针结果（D3-4 断流/卡顿观察） */
export interface StreamProbeResult {
  model: string
  ok: boolean
  ttft_ms: number
  chunks: number
  tokens: number
  duration_ms: number
  max_gap_ms: number
  avg_gap_ms: number
  completed: boolean
  interrupted: boolean
  error?: string
  response_start?: string
}

export interface BenchmarkRunDetail {
  id: string
  created_at: string
  finished_at?: string
  status: string
  config: BenchmarkRequest
  summary: BenchmarkSummary
  cases: BenchmarkCase[]
}

// ── API 函数 ─────────────────────────────────────────────────

import type { AppFacade } from '../types/wails'

const App = (): AppFacade => window.go?.app?.App as AppFacade

/** 获取所有引擎配置 */
export async function getEngines(): Promise<EngineConfig[]> {
  const result = await App().GetEngines()
  return result as EngineConfig[]
}

/** 保存引擎配置 */
export async function saveEngine(cfg: EngineConfig): Promise<void> {
  await App().SaveEngine(cfg)
}

/** 添加自定义引擎（OpenAI 兼容，type=custom），返回引擎 ID（custom-*） */
export async function addCustomEngine(name: string, baseURL: string, apiKey: string): Promise<string> {
  const result = await App().AddCustomEngine(name, baseURL, apiKey)
  return result as string
}

/** 更新自定义引擎（apiKey 空串 = 不修改 Key） */
export async function updateCustomEngine(engineID: string, name: string, baseURL: string, apiKey: string): Promise<void> {
  await App().UpdateCustomEngine(engineID, name, baseURL, apiKey)
}

/** 删除自定义引擎（连同其功能绑定能力） */
export async function removeCustomEngine(engineID: string): Promise<void> {
  await App().RemoveCustomEngine(engineID)
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

/** 设置办公「提取文字」使用的 OCR 引擎与模型 */
export async function setActiveOCRModel(engineID: string, modelID: string): Promise<void> {
  await App().SetActiveOCRModel(engineID, modelID)
}

/** 获取当前 OCR 激活模型（空=自动选择） */
export async function getActiveOCRModel(): Promise<{ engine: string; model: string }> {
  const result = await App().GetActiveOCRModel()
  return result as { engine: string; model: string }
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
export async function getDeepseekKeyStatus(): Promise<{ configured: boolean; masked: string }> {
  const result = await App().GetDeepseekKeyStatus()
  return result as { configured: boolean; masked: string }
}

/** 设置 GLM (智谱) API Key */
export async function setGlmKey(apiKey: string): Promise<void> {
  await App().SetGlmKey(apiKey)
}

/** 设置 GLM 端点家族（std=标准按量付费 / coding=编码套餐额度；官方双端点） */
export async function setGlmEndpoint(family: 'std' | 'coding'): Promise<void> {
  await App().SetGlmEndpoint(family)
}

/** 获取 GLM Key 状态（脱敏显示） */
export async function getGlmKeyStatus(): Promise<{ configured: boolean; masked: string }> {
  const result = await App().GetGlmKeyStatus()
  return result as { configured: boolean; masked: string }
}

/** 设置 OpenCode Go API Key */
export async function setOpencodeGoKey(apiKey: string): Promise<void> {
  await App().SetOpencodeGoKey(apiKey)
}

/** 获取 OpenCode Go Key 状态（脱敏显示） */
export async function getOpencodeGoKeyStatus(): Promise<{ configured: boolean; masked: string }> {
  const result = await App().GetOpencodeGoKeyStatus()
  return result as { configured: boolean; masked: string }
}

/** 设置 OpenCode Zen API Key */
export async function setOpencodeZenKey(apiKey: string): Promise<void> {
  await App().SetOpencodeZenKey(apiKey)
}

/** 获取 OpenCode Zen Key 状态（脱敏显示） */
export async function getOpencodeZenKeyStatus(): Promise<{ configured: boolean; masked: string }> {
  const result = await App().GetOpencodeZenKeyStatus()
  return result as { configured: boolean; masked: string }
}

/** 获取模型调用统计汇总 */
export async function getModelCallStats(): Promise<ModelStatsSummary> {
  const result = await App().GetModelCallStats()
  return result as ModelStatsSummary
}

/** 获取美元→人民币汇率（费用估算折算用，默认 7.2；T6-6.2） */
export async function getUsdCnyRate(): Promise<number> {
  const result = await App().GaeaGetUsdCnyRate()
  return result as number
}

/** 设置美元→人民币汇率（持久化到 usd_cny_rate 并即时生效；T6-6.2） */
export async function setUsdCnyRate(rate: number): Promise<void> {
  await App().GaeaSetUsdCnyRate(rate)
}

/** 获取 Herdsman 完整模型目录（模型中心「模型库」） */
export async function getHerdsmanCatalog(): Promise<HerdsmanCatalog> {
  const result = await App().HerdsmanModelCatalog()
  return result as HerdsmanCatalog
}

/** 获取 Herdsman 完整模型目录（模型中心「模型库」） */
export async function getHerdsmanLaunchPresets(): Promise<HerdsmanLaunchPreset[]> {
  const result = await App().HerdsmanLaunchPresets()
  return result as HerdsmanLaunchPreset[]
}

/** 获取 Herdsman 本地调用统计（model_stats/events.jsonl 聚合） */
export async function getHerdsmanModelStats(): Promise<HerdsmanModelStats> {
  const result = await App().HerdsmanModelStats()
  return result as HerdsmanModelStats
}

/** 启动 Herdsman 模型（等冷启动完成） */
export async function startHerdsmanModel(model: string): Promise<HerdsmanOpResult> {
  const result = await App().HerdsmanModelStart(model)
  return result as HerdsmanOpResult
}

/** 停止 Herdsman 模型 */
export async function stopHerdsmanModel(model: string): Promise<HerdsmanOpResult> {
  const result = await App().HerdsmanModelStop(model)
  return result as HerdsmanOpResult
}

/** 下载 Herdsman 模型（等安装完成） */
export async function downloadHerdsmanModel(model: string): Promise<HerdsmanOpResult> {
  const result = await App().HerdsmanModelDownload(model)
  return result as HerdsmanOpResult
}

/** 卸载 Herdsman 模型 */
export async function uninstallHerdsmanModel(model: string): Promise<HerdsmanOpResult> {
  const result = await App().HerdsmanModelUninstall(model)
  return result as HerdsmanOpResult
}

/** 重置模型调用统计 */
export async function resetModelCallStats(): Promise<void> {
  await App().ResetModelCallStats()
}

/** 分流统计总览（本地 vs 云端 + 节省对比，D3-2） */
export async function getUsageOverview(): Promise<UsageOverview> {
  const result = await App().GaeaUsageOverview()
  return result as UsageOverview
}

/** 语义向量索引状态（各 kind 条数，D3-1） */
export async function getSemanticIndexStatus(): Promise<SemanticIndexStatus> {
  const result = await App().GaeaSemanticIndexStatus()
  return result as SemanticIndexStatus
}

/** Herdsman 受控测评运行列表（D3-3） */
export async function getBenchmarkList(): Promise<BenchmarkRunSummary[]> {
  const result = await App().GaeaBenchmarkList()
  return result as BenchmarkRunSummary[]
}

/** 发起一次受控测评（返回运行 ID） */
export async function startBenchmark(req: BenchmarkRequest): Promise<string> {
  const result = await App().GaeaBenchmarkStart(req)
  return result as string
}

/** 测评运行完整明细（逐 case） */
export async function getBenchmarkDetail(id: string): Promise<BenchmarkRunDetail> {
  const result = await App().GaeaBenchmarkDetail(id)
  return result as BenchmarkRunDetail
}

/** 导出测评报告（Markdown），返回文件路径 */
export async function exportBenchmark(id: string, dir: string): Promise<string> {
  const result = await App().GaeaBenchmarkExport(id, dir)
  return result as string
}

/** 流式探针：对模型发起一次 SSE 流式请求，观察 TTFT/分块间隔/断流（D3-4） */
export async function streamProbe(model: string): Promise<StreamProbeResult> {
  const result = await App().GaeaBenchmarkStreamProbe(model)
  return result as StreamProbeResult
}
