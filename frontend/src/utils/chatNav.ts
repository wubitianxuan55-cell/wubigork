// chatNav.ts — 聊天板块跨页面入口信号（首页「轻语」卡片 → 聊天板块 persona 模式）

let pendingPersona = false

/** 首页轻语卡片点击：请求进入聊天板块并直接切到 persona 模式。 */
export function requestPersonaEnter(): void {
  pendingPersona = true
  window.dispatchEvent(new CustomEvent('gaea-chat-persona-enter'))
}

/** ChatPage 挂载时消费一次性请求（避免事件早于组件监听）。 */
export function consumePersonaEnter(): boolean {
  const v = pendingPersona
  pendingPersona = false
  return v
}
