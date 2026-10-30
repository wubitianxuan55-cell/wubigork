// 逐席 accent：四席四色区分（中明度，浅/暗双主题都可读）。
// grok 沿用众神殿金（--pantheon-accent），其余三席为席位品牌色（hex-exempt）。
export const AGENT_ACCENTS: Record<string, string> = {
  grok: 'var(--pantheon-accent)',
  codex: '#5b9bd5', // hex-exempt 席位品牌色
  zcode: '#9a8cf2', // hex-exempt 席位品牌色
  dsh: '#d98080', // hex-exempt 席位品牌色
}
