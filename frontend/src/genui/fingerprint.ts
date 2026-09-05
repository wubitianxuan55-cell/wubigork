// 内容指纹与状态 key。

/** djb2 指纹（非安全哈希，仅用于状态/幂等键）。 */
export function fingerprint(text: string): string {
  let h = 5381;
  for (let i = 0; i < text.length; i++) {
    h = ((h << 5) + h + text.charCodeAt(i)) >>> 0;
  }
  return h.toString(36);
}

/**
 * 构造交互状态 key：板块作用域 + 会话 + 消息/面板槽位 + 内容指纹。
 * 同一内容重放 → 同 key → 恢复状态；换内容 → 新 key → 干净起点。
 */
export function genuiStateKey(
  scope: "chat" | "office",
  sessionKey: string,
  slot: string,
  content: string,
): string {
  return `genui:${scope}:${sessionKey}:${slot}:${fingerprint(content)}`;
}

/** 办公面板内容/交互 key（不绑定具体消息，面板原地更新）。 */
export function genuiPanelKey(scope: "office", sessionKey: string, panelKey = "main"): string {
  return `genui:${scope}:panel:${sessionKey}:${panelKey}`;
}
