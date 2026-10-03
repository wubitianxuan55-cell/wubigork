// controller_actions_memory.ts — 记忆/画像族动作 hook（批 36 FE3-11 hook 分域
// 首刀，自 controller.ts 原位搬移，零逻辑改动）：logBridgeError/errText/failWrite
// 桥接错误三助手 + useMemoryActions（fetchMemory/remember/forget/saveDoc/
// updateFact/changeFactType/clearFactBase/promoteFactBase 八件，useCallback
// 依赖数组逐字保留——clearFactBase 的跨族依赖 refreshFactBase 由 useController
// 显式注入）。useStore/ensureEventsBound/useController 主体见 controller.ts。
import { useCallback } from "react";
import { app } from "../bridge";
import type { Action } from "./controller_state";
import type { MemoryView } from "../types";

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

// failWrite 写路径失败出口（T7-4）：bridge 层 invoke 已把错误记录到 gaea.log，
// 这里再补一条用户可见的 warn notice，保证写/提交/审批失败绝不静默、用户可重试。
export function failWrite(dispatch: (a: Action) => void, what: string, err: unknown): void {
  logBridgeError(what, err);
  dispatch({ type: "event", e: { kind: "notice", level: "warn", text: `${what}失败：${errText(err)}，请重试` } });
}

// isFinalAnswerRendered：最终回答是否已完整渲染（T7-4 完整文本比较）。
export function useMemoryActions(dispatch: (a: Action) => void, refreshFactBase: () => void) {
  const fetchMemory = useCallback((): Promise<MemoryView> => app.Memory().catch((err) => {
    logBridgeError("fetchMemory", err);
    return { docs: [], facts: [], scopes: [], storeDir: "", available: false } as MemoryView;
  }), []);
  const remember = useCallback(async (scope: string, note: string) => { await app.Remember(scope, note).catch((err) => failWrite(dispatch, "保存记忆", err)); }, [dispatch]);
  const forget = useCallback(async (name: string) => { await app.Forget(name).catch((err) => failWrite(dispatch, "删除记忆", err)); }, [dispatch]);
  const saveDoc = useCallback(async (path: string, body: string) => { await app.SaveDoc(path, body).catch((err) => failWrite(dispatch, "保存文档", err)); }, [dispatch]);
  const updateFact = useCallback(async (name: string, body: string) => { await app.UpdateFact(name, body).catch((err) => failWrite(dispatch, "更新画像", err)); }, [dispatch]);
  const changeFactType = useCallback(async (name: string, typ: string) => { await app.ChangeFactType(name, typ).catch((err) => failWrite(dispatch, "修改画像类型", err)); }, [dispatch]);
  const clearFactBase = useCallback(async () => {
    await app.FactBaseClear().catch((err) => failWrite(dispatch, "清空事实库", err));
    refreshFactBase();
  }, [dispatch, refreshFactBase]);
  const promoteFactBase = useCallback(async (): Promise<number> => {
    const n = await app.FactBasePromote().catch((err) => { failWrite(dispatch, "写入永久记忆", err); return 0; });
    return n;
  }, [dispatch]);
  return { fetchMemory, remember, forget, saveDoc, updateFact, changeFactType, clearFactBase, promoteFactBase };
}
