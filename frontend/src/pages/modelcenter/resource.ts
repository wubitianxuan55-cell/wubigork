// 模型中心「本地资源占用」纯计算：从 GetModelMonitor 的原始返回推导可展示快照。
// 单位与后端保持一致：内存/显存均为 GB，CPU/GPU 为百分比。

export interface ResourceMonitorEngine {
  engine: string
  name?: string
  model?: string
  isLocal?: boolean
}

export interface ResourceMonitorStats {
  cpu?: number
  memTotal?: number
  memUsed?: number
  gpuName?: string
  gpuUsage?: number
  vramUsed?: number
  vramTotal?: number
}

export interface ResourceMonitorData {
  engines?: ResourceMonitorEngine[]
  stats?: ResourceMonitorStats
  comfyRunning?: boolean
}

export interface ResourceSnapshot {
  cpu: number
  memPct: number
  memUsed: number
  memTotal: number
  gpuName: string
  gpuPct: number
  vramPct: number
  vramUsed: number
  vramTotal: number
  localEngines: string[]
  comfyRunning: boolean
}

export function computeResourceSnapshot(m: ResourceMonitorData | null | undefined): ResourceSnapshot {
  const s = m?.stats || {}
  const memTotal = s.memTotal || 0
  const memUsed = s.memUsed || 0
  const vramTotal = s.vramTotal || 0
  const vramUsed = s.vramUsed || 0
  const memPct = memTotal ? Math.round((memUsed / memTotal) * 100) : 0
  const vramPct = vramTotal ? Math.round((vramUsed / vramTotal) * 100) : 0
  const gpuPct = Math.max(0, (s.gpuUsage ?? 0) > 0 ? s.gpuUsage : vramPct)
  const localEngines = (m?.engines || [])
    .filter(e => e.isLocal)
    .map(e => `${e.engine}${e.model ? '·' + String(e.model).split('/').pop() : ''}`)

  return {
    cpu: Math.max(0, s.cpu ?? 0),
    memPct,
    memUsed,
    memTotal,
    gpuName: s.gpuName || '',
    gpuPct,
    vramPct,
    vramUsed,
    vramTotal,
    localEngines,
    comfyRunning: !!m?.comfyRunning,
  }
}

export type ResourceLevel = 'ok' | 'warn' | 'high'

export function resourceLevel(pct: number): ResourceLevel {
  if (pct >= 90) return 'high'
  if (pct >= 75) return 'warn'
  return 'ok'
}

// 3.0 Wave 4：硬编码 hex → 全局语义令牌（App.tsx 注入 :root，随 12 主题联动）
export const resourceLevelColor: Record<ResourceLevel, string> = {
  ok: 'var(--color-success)',
  warn: 'var(--color-warning)',
  high: 'var(--color-destructive)',
}

export function fmtGB(gb: number): string {
  if (!gb) return '--'
  return `${gb.toFixed(1)} GB`
}
