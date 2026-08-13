/** 聊天输入框的键盘提交判定（与 React KeyboardEvent 解耦，便于测试）。 */

/**
 * 是否按 Enter 提交：必须是 Enter、非 Shift（Shift+Enter 换行）、
 * 且不在输入法组合态（中文/日文 IME 候选确认的 Enter 不应触发发送）。
 */
export function shouldSubmitOnEnter(key: string, shiftKey: boolean, isComposing: boolean): boolean {
  return key === "Enter" && !shiftKey && !isComposing;
}
