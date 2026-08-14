// ImageGenPage 拆分产物：纯工具函数（行为零变化，T6-10.1）
import { readFileAsDataURL } from '../../api/image'
import { HISTORY_META_KEY, serializeHistoryMeta } from './historyMeta'
import type { GenResult } from './types'

export const BACKEND_OPTIONS = [
  { label: '☁️ xAI 云端', value: 'xai' },
  { label: '🏠 ComfyUI 本地', value: 'comfyui' },
  { label: '🐄 Herdsman 本地', value: 'herdsman' },
  { label: '🦙 Ollama 本地', value: 'ollama' },
]

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
