/**
 * LoRA 模型族识别与过滤
 *
 * ComfyUI 的 LoRA 文件名带子目录前缀（如 krea2\xxx.safetensors、zimage\xxx.safetensors、
 * flux1\xxx.safetensors），不同模型的 LoRA 不通用。本模块按「路径子目录 → 文件名前缀」
 * 判断每个 LoRA 属于哪个模型族，并在模型切换时只暴露匹配的 LoRA。
 */

export type LoraFamily = 'krea2' | 'zimage' | 'flux1' | 'generic'

function splitLora(name: string): { norm: string; file: string } {
  const norm = name.replace(/\\/g, '/').toLowerCase()
  const file = norm.split('/').pop() || norm
  return { norm, file }
}

/** 判断单个 LoRA 所属模型族 */
export function loraFamily(name: string): LoraFamily {
  const { norm, file } = splitLora(name)
  if (norm.startsWith('krea2/') || file.startsWith('krea')) return 'krea2'
  if (norm.startsWith('zimage/') || file.startsWith('zimage') || file.startsWith('z-image') || file.startsWith('zit')) return 'zimage'
  if (norm.startsWith('flux1/') || norm.startsWith('flux/') || file.startsWith('flux')) return 'flux1'
  return 'generic'
}

/** 模型 id → 允许的 LoRA 族（根目录通用 LoRA 对所有模型开放） */
export function loraFamiliesForModel(model: string): LoraFamily[] {
  const m = model.toLowerCase()
  if (m.startsWith('krea')) return ['krea2', 'generic']
  if (m.includes('flux')) return ['flux1', 'generic']
  if (m.includes('z-image') || m.includes('zimage') || m.startsWith('zit')) return ['zimage', 'generic']
  return ['generic']
}

/** 过滤出指定模型可用的 LoRA 列表 */
export function filterLorasByModel(model: string, loras: string[]): string[] {
  const allowed = loraFamiliesForModel(model)
  return loras.filter((l) => allowed.includes(loraFamily(l)))
}
