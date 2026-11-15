// controller_actions_load.ts — sin fork：会话装载族（loadItemsFoldedFirst/
// loadSessionData/reconcileFinalAnswer 三件）。折叠快照优先口径与办公逐字一致
// （SinEngineResyncEvents(0) 恒返回会话全量快照，id=日志序）；sinEngineHistory
// 为 History 保底。isFinalAnswerRendered 纯函数与办公同名同语义。
import { useCallback } from "react";
import { app } from "../../../gaea/lib/bridge";
import type { Action, ControllerState, Item } from "./controller_state";
import { resumeSnapshotItems } from "./controller_state";
import type { HistoryMessage } from "../../../gaea/lib/types";
import { logBridgeError } from "./controller_actions_memory";

// isFinalAnswerRendered：最终回答是否已完整渲染（完整文本比较——流式丢尾时
// 前缀命中但正文缺失会误判；正文为空视为已渲染）。
export function isFinalAnswerRendered(rendered: string, finalContent: string): boolean {
  const full = finalContent.trim();
  if (!full) return true;
  return rendered.trimEnd().endsWith(full);
}

export function useSessionLoaders(dispatch: (a: Action) => void, store: { getState: () => ControllerState }) {
  // loadItemsFoldedFirst 对话项装载统一入口：SinEngineResyncEvents(0) 全量折叠
  // 快照优先，形状坏/空回退 SinEngineHistory+rebuildHistoryItems（h<index> 保底）。
  const loadItemsFoldedFirst = useCallback(async (): Promise<void> => {
    try {
      const snap = await app.SinEngineResyncEvents(0);
      const items = resumeSnapshotItems(snap?.items);
      if (items) {
        if (items.length) dispatch({ type: "resync", items });
        return;
      }
    } catch (err) {
      logBridgeError("loadItemsFoldedFirst SinEngineResyncEvents", err);
    }
    const ms = await app.SinEngineHistory().catch((err: unknown) => {
      logBridgeError("loadItemsFoldedFirst SinEngineHistory", err);
      return [] as HistoryMessage[];
    });
    if (ms.length) dispatch({ type: "history", messages: ms });
  }, [dispatch]);

  const loadSessionData = useCallback(async () => {
    try {
      dispatch({ type: "context", context: await app.SinEngineContext() });
      await loadItemsFoldedFirst();
    } catch (err) {
      logBridgeError("loadSessionData", err);
    }
  }, [dispatch, loadItemsFoldedFirst]);

  // 最终回答兜底：turn_done 时拉一次 History，末端缺失才补发 message 事件。
  const reconcileFinalAnswer = useCallback(() => {
    app.SinEngineHistory().then((ms) => {
      const last = [...ms].reverse().find(
        (m) => m.role === "assistant" && typeof m.content === "string" && m.content.trim() !== "",
      );
      if (!last) return;
      const st = store.getState();
      const rendered = st.items
        .filter((i) => i.kind === "assistant" && i.text)
        .map((i) => (i as Extract<Item, { kind: "assistant" }>).text)
        .join("\n");
      if (!isFinalAnswerRendered(rendered, last.content)) {
        dispatch({
          type: "event",
          e: { kind: "message", text: last.content, reasoning: (last as { reasoning?: string }).reasoning ?? "" },
        });
      }
    }).catch((err) => logBridgeError("reconcileFinalAnswer", err));
  }, [store, dispatch]);

  return { loadItemsFoldedFirst, loadSessionData, reconcileFinalAnswer };
}
