// controller_actions_load.ts — 会话装载族 hook（批 39 FE3-11 hook 分域末刀，
// 自 controller.ts 原位搬移，零逻辑改动）：useSessionLoaders（loadItemsFoldedFirst/
// loadSessionData/reconcileFinalAnswer/refreshFactBase 四件 + 随走的纯函数
// isFinalAnswerRendered〔reconcileFinalAnswer 消费，controller.ts 命名 re-export
// 保 ./store 测试导入路径不变〕；store 以 { getState } 结构面注入防环）。
// ensureEventsBound 绑定接线与 useController 主体见 controller.ts。
import { useCallback } from "react";
import { app } from "../bridge";
import { invalidateTurnCaches } from "../deliverablesTurn";
import type { Action, ControllerState, Item } from "./controller_state";
import { resumeSnapshotItems } from "./controller_state";
import type { HistoryMessage } from "../types";
import { logBridgeError } from "./controller_actions_memory";

// isFinalAnswerRendered：最终回答是否已完整渲染（T7-4 完整文本比较——
// 流式丢尾时前缀命中但正文缺失，旧「前 120 字前缀」启发式会误判已渲染；
// 正文为空（纯推理/纯工具轮）视为已渲染，避免误补发）。
export function isFinalAnswerRendered(rendered: string, finalContent: string): boolean {
  const full = finalContent.trim();
  if (!full) return true;
  return rendered.trimEnd().endsWith(full);
}

export function useSessionLoaders(dispatch: (a: Action) => void, store: { getState: () => ControllerState }) {
  // loadItemsFoldedFirst 对话项装载统一入口（GenUI 审计 §8-5 slot 口径统一）：
  // GaeaResyncEvents(0) 恒返回会话全量折叠快照（id=日志序 resyncID），resume/
  // 回退/启动三路同 id → genui 交互状态跨重启可命中；快照不可用（旧后端异常/
  // 形状坏 null/空会话）回退 History+rebuildHistoryItems（h<index> legacy 保底）。
  // 恒 resolve 不抛：装载失败不阻断会话流（错误走 logBridgeError 记录）。
  const loadItemsFoldedFirst = useCallback(async (): Promise<void> => {
    try {
      const snap = await app.ResyncEvents(0);
      const items = resumeSnapshotItems(snap?.items);
      if (items) {
        if (items.length) dispatch({ type: "resync", items });
        return;
      }
    } catch (err) {
      logBridgeError("loadItemsFoldedFirst ResyncEvents", err);
    }
    const ms = await app.History().catch((err: unknown) => {
      logBridgeError("loadItemsFoldedFirst History", err);
      return [] as HistoryMessage[];
    });
    if (ms.length) dispatch({ type: "history", messages: ms });
  }, [dispatch]);

  const loadSessionData = useCallback(async () => {
    // v4.382 失效接线：会话切换/恢复即清空登记与目录探测缓存（缓存随会话走）。
    invalidateTurnCaches();
    try {
      dispatch({ type: "meta", meta: await app.Meta() });
      dispatch({ type: "context", context: await app.ContextUsage() });
      await loadItemsFoldedFirst();
    } catch (err) {
      // 启动期后端未就绪时 Meta/Context/History 可能失败：记录错误，
      // 状态保持默认值，不再静默。
      logBridgeError("loadSessionData", err);
    }
  }, [dispatch, loadItemsFoldedFirst]);

  // 最终回答兜底：turn_done（或看门狗检测到后端已停）时拉一次 History，
  // 如果最后一条 assistant 正文没有渲染过，就补发一条 message 事件。
  // 修复末端事件（message/turn_done 密集到达时被 Wails 事件流吞掉）
  // 导致的“最终回答只有重启才可见”。
  const reconcileFinalAnswer = useCallback(() => {
    app.History().then((ms) => {
      const last = [...ms].reverse().find(
        (m) => m.role === "assistant" && typeof m.content === "string" && m.content.trim() !== "",
      );
      if (!last) return;
      const st = store.getState();
      const rendered = st.items
        .filter((i) => i.kind === "assistant" && i.text)
        .map((i) => (i as Extract<Item, { kind: "assistant" }>).text)
        .join("\n");
      // T7-4：前缀启发式（只查前 120 字是否已渲染）改为完整文本比较——
      // 流式丢尾时前缀命中但正文不完整，仍会误判“已渲染”。要求渲染文本
      // 以完整正文结尾才算渲染过，缺失才补发 message。
      if (!isFinalAnswerRendered(rendered, last.content)) {
        dispatch({
          type: "event",
          e: { kind: "message", text: last.content, reasoning: (last as { reasoning?: string }).reasoning ?? "" },
        });
      }
    }).catch((err) => logBridgeError("reconcileFinalAnswer", err));
  }, [store, dispatch]);

  const refreshFactBase = useCallback(() => {
    app.FactBase().then(factBase => dispatch({ type: "factbase", factBase })).catch((err) => logBridgeError("refreshFactBase", err));
  }, [dispatch]);

  return { loadItemsFoldedFirst, loadSessionData, reconcileFinalAnswer, refreshFactBase };
}
