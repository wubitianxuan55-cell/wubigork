// 逐席 identity accent：四席四色用于贯穿卡片、消息、工具轨迹与席位窗。
// 只作身份线索；正文/小字使用方会在运行时混入当前主题正文色保证对比度，
// 状态语义仍由图标与文字承担，避免把 DSH 珊瑚色误读为失败。
// grok 沿用众神殿金（--pantheon-accent），其余三席为席位品牌色（hex-exempt）。
export const AGENT_ACCENTS: Record<string, string> = {
  grok: 'var(--pantheon-accent)',
  codex: '#5b9bd5', // hex-exempt 席位品牌色
  zcode: '#9a8cf2', // hex-exempt 席位品牌色
  dsh: '#d98080', // hex-exempt 席位品牌色
}

interface AgentProfile {
  compactName: string
  capability: string
}

const AGENT_PROFILES: Record<string, AgentProfile> = {
  grok: { compactName: 'Grok', capability: '联网 · 识图' },
  codex: { compactName: 'Codex', capability: '联网 · 识图' },
  zcode: { compactName: 'ZCode', capability: '联网 · 识图' },
  dsh: { compactName: 'DSH', capability: '联网 · 文件' },
}

export function agentCompactName(agentId: string, fallbackName: string): string {
  return AGENT_PROFILES[agentId]?.compactName || fallbackName
}

export function agentCapabilityLabel(agentId: string): string {
  return AGENT_PROFILES[agentId]?.capability || 'Agent'
}

// 议事路径标签（2026-10-31 裁决第二刀：对照实验可表达，每场必须可打标签）。
export const PROTOCOL_LABELS: Record<string, string> = {
  council: '圆桌接力',
  independent: '独立开场',
}

export function protocolLabel(protocol?: string): string {
  return (protocol && PROTOCOL_LABELS[protocol]) || '圆桌接力'
}
