// controller_actions_memory.ts — sin fork（桥接错误三助手；记忆动作族不在刀2：
// 书级记忆管理面走刀4，届时在此扩 useSinMemoryActions）。
import { app } from "../../../gaea/lib/bridge";
import type { Action } from "./controller_state";

// 避免掩盖业务错误。T6-1.2 去静默 catch：错误必须可见。
export function logBridgeError(where: string, err: unknown): void {
  const e = (err ?? {}) as { code?: unknown; message?: unknown };
  const code = typeof e.code === "string" ? e.code : "BridgeError";
  const message = typeof e.message === "string" ? e.message : String(err);
  const lfe = app.LogFrontendError;
  if (typeof lfe !== "function") return;
  void Promise.resolve(lfe(`[${code}] ${where}: ${message}`)).catch(() => {});
}

// errText 提取用户可读的错误信息（BridgeError/Error/其他值统一字符串化）。
export function errText(err: unknown): string {
  return err instanceof Error ? err.message : String(err ?? "未知错误");
}

// failWrite 写路径失败出口（T7-4）：失败绝不静默、用户可重试。
export function failWrite(dispatch: (a: Action) => void, what: string, err: unknown): void {
  logBridgeError(what, err);
  dispatch({ type: "event", e: { kind: "notice", level: "warn", text: `${what}失败：${errText(err)}，请重试` } });
}
