// ImageGenPage 拆分产物：纯工具函数（行为零变化，T6-10.1）
import { readFileAsDataURL } from '../../api/image'
import { HISTORY_META_KEY, serializeHistoryMeta } from './historyMeta'
import type { GenResult } from './types'

/**
 * 引擎枚举（单一来源）：绘梦页引擎下拉 / 顶条状态 / 启动消息共用。
 * label 为简短显示名（不含「云端/本地」字尾——调用方按需拼接），
 * txt2imgOnly=仅文生图可用（GLM 官方图像端点无图生图参数）。
 */
export interface BackendOptionMeta {
  value: string
  label: string
  txt2imgOnly?: boolean
}

export const BACKEND_OPTIONS: BackendOptionMeta[] = [
  { value: 'xai', label: 'xAI' },
  { value: 'comfyui', label: 'ComfyUI' },
  { value: 'herdsman', label: 'Herdsman' },
  { value: 'ollama', label: 'Ollama' },
  // GLM 官方图像端点仅文生图（/images/generations 无图生图参数）
  { value: 'glm', label: 'GLM', txt2imgOnly: true },
]

/** 后端是否本地引擎（顶条/启停块据此区分云端/本地） */
export function isLocalBackend(backend: string): boolean {
  return ['comfyui', 'herdsman', 'ollama'].includes(backend)
}

/** 后端显示名（顶条/启动消息用）；未知后端回退原始 id */
export function backendLabel(backend: string): string {
  return BACKEND_OPTIONS.find((b) => b.value === backend)?.label || backend
}

/**
 * 引擎 × 模式是否可生成（单源判定）：
 * txt2imgOnly（GLM）非 txt2img 不可用；未知后端不武断禁用（诚实放行由后端
 * 报错）。ControlPanel 选项禁用 / GenerationBar 生成门禁 / 残留态警告共用此
 * 判定，避免各层白名单漂移。
 */
export function backendSupportsMode(backend: string, mode: string): boolean {
  const meta = BACKEND_OPTIONS.find((o) => o.value === backend)
  if (!meta) return true
  if (meta.txt2imgOnly === true && mode !== 'txt2img') return false
  return true
}

/**
 * 模板推荐画幅 → 实际画幅状态（纯函数，供应用模板时同步画幅）。
 * 模板 size 为比例标签（'1:1' / '16:9' …），绘梦画幅为具体尺寸串或 'custom'。
 * 仅文生图（txt2img）生效：图生图输出随参考图、文生视频有独立画幅体系。
 * 返回 null = 模板未声明画幅或不适用当前模式，调用方保持原值。
 */
export function templateSizeToPreset(
  ratio: string | undefined,
  mode: string | undefined,
): { size: string; customWidth?: number; customHeight?: number } | null {
  if (!ratio || mode !== 'txt2img') return null
  const PRESETS: Record<string, { size: string; customWidth?: number; customHeight?: number }> = {
    '1:1': { size: '1024x1024' },
    '4:3': { size: '1024x768' },
    '16:9': { size: '1024x576' },
    '9:16': { size: '576x1024' },
    '3:4': { size: '768x1024' },
    '21:9': { size: '1280x544' },
    // 2:3 无预置档 → 走自定义（768×1152，64 倍数对齐 ComfyUI 约束）
    '2:3': { size: 'custom', customWidth: 768, customHeight: 1152 },
  }
  return PRESETS[ratio] || null
}

export function loadHistoryMeta(): GenResult[] {
  try {
    const raw = localStorage.getItem(HISTORY_META_KEY)
    if (!raw) return []
    const items = JSON.parse(raw) as GenResult[]
    // 保留已内联的小图与 file_path；无图无路径的旧记录降级为占位
    return items.map((it) => ({ ...it, image: it.image || '' }))
  } catch {
    return []
  }
}

// 历史元数据保存：小 base64 内联、大图只存 file_path（localStorage 容量保护分级策略）
export function saveHistoryMeta(history: GenResult[]) {
  try {
    localStorage.setItem(HISTORY_META_KEY, JSON.stringify(serializeHistoryMeta(history)))
  } catch (err) {
    console.warn('[imagegen] 历史元数据保存失败', err)
  }
}

// ── 模型分类（与 ModelCenterPage 保持一致） ──

export function classifyModel(id: string): string {
  const lid = id.toLowerCase()
  if (lid.includes('tts') || lid.includes('voice') || lid.includes('edge')) return 'tts'
  if (lid.includes('sherpa') || lid.includes('whisper') || lid.includes('zipformer') || lid.includes('asr')) return 'stt'
  if (lid.includes('image') || lid.includes('zimage') || lid.includes('flux') || lid.includes('turbo') || lid.includes('sd') || lid.includes('dalle')) return 'image'
  return 'llm'
}

/** LoRA 文件名 → 可读标签（去掉扩展名/路径，下划线转空格） */
export function loraLabel(name: string): string {
  const base = name.replace(/\.(safetensors|pt|bin|ckpt|sft)$/i, '')
  const rel = base.replace(/\\/g, '/')
  const file = rel.split('/').pop() || rel
  return file.replace(/_/g, ' ')
}

// 解析可展示/下载的图像数据：优先用 file_path（本地文件为权威来源），失败回退内存 base64
export async function resolveResultImage(r: GenResult): Promise<string> {
  if (r.file_path) {
    try {
      const url = await readFileAsDataURL(r.file_path)
      if (url) return url
    } catch (err) {
      console.warn(`[imagegen] 读取本地图片失败，回退内存数据: ${r.file_path}`, err)
    }
  }
  return r.image || ''
}
