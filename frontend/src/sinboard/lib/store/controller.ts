// controller.ts — 原罪板块 controller 状态机编排层（fork 办公 gaea/lib/store/
// controller.ts，2026-11-15 升格刀）。
//
// 隔离纪律：本文件族（sinboard/lib/store/*）是独立 zustand 实例 + 独立事件
// 通道 gaea-event-sin + SinEngine* 绑定——与办公 controller store 模块单例
// 零共享（keepAlive 下两板块可同时挂载，任何共享可变状态都会互踩）。
// 状态机纯函数层（controller_state.ts/eventSync.ts）为逐字 fork：事件协议、
// item 取号、补拉防线与办公完全同构，真源差异只在通道与绑定名。

import { useEffect } from "react";
import { create } from "zustand";
import { useShallow } from "zustand/shallow";
import { app } from "../../../gaea/lib/bridge";
import { subscribe } from "../../../events";
import { setEventSyncFetcher } from "./eventSync";
import { logBridgeError } from "./controller_actions_memory";
import { useSessionActions } from "./controller_actions_session";
import { useTurnActions } from "./controller_actions_turn";
import { useSessionLoaders } from "./controller_actions_load";
export * from "./controller_state";
import { initialState, reducer } from "./controller_state";
import type { Action, ControllerState } from "./controller_state";
import type { WireEvent } from "../../../gaea/lib/types";

export const useStore = create<ControllerState>()((set) => ({ ...initialState, _dispatch: (a: Action) => set((s) => reducer(s, a)) } as ControllerState));

// isFinalAnswerRendered 随装载族出仓（controller_actions_load.ts），命名
// re-export 保与办公同名导出形态一致。
export { isFinalAnswerRendered } from "./controller_actions_load";

// ── 原罪引擎事件通道（与办公 gaea-event 完全隔离）──────────────────────
const SIN_ENGINE_CHANNEL = "gaea-event-sin";

interface EventBindDeps {
  loadSessionData: () => Promise<void>;
  reconcileFinalAnswer: () => void;
  store: typeof useStore;
  dispatch: (a: Action) => void;
}
let eventsBound = false;

// ensureEventsBound 恰好一次订阅（fork 办公同纪律）：卸载不退订——隐藏期间
// 事件继续入库，回到板块即热态。补拉防线自接线（fetcher = SinEngineResyncEvents，
// 独立 wire seq 空间，与办公补拉互不干扰）。
function ensureEventsBound(deps: EventBindDeps): void {
  if (eventsBound) return;
  eventsBound = true;
  const { dispatch, reconcileFinalAnswer } = deps;
  setEventSyncFetcher(async (afterSeq: number) => {
    const snap = await app.SinEngineResyncEvents(afterSeq);
    return { seq: snap.seq, items: snap.items as unknown[] };
  });
  subscribe(SIN_ENGINE_CHANNEL, (raw) => {
    const e = (raw || {}) as WireEvent;
    // 流式增量微任务派发（与办公同口径：不被 React 18 批处理合并）
    if (e.kind === "text" || e.kind === "reasoning") {
      queueMicrotask(() => dispatch({ type: "event", e }));
    } else {
      dispatch({ type: "event", e });
    }
    if (e.kind === "turn_done") {
      app.SinEngineContext().then((c) => dispatch({ type: "context", context: c })).catch((err) => logBridgeError("turn_done SinEngineContext", err));
      reconcileFinalAnswer();
    }
  });
  void deps.loadSessionData();
}

export function useSinController() {
  const store = useStore;
  const state = store(useShallow((s) => s));
  const dispatch = store.getState()._dispatch;

  const { loadItemsFoldedFirst, loadSessionData, reconcileFinalAnswer } = useSessionLoaders(dispatch, store);

  useEffect(() => {
    ensureEventsBound({ loadSessionData, reconcileFinalAnswer, store, dispatch });
    // 看门狗：running=true 时每 30s 用后端真实状态校准一次（fork 办公同纪律，
    // 读 SinEngineRunning——独立引擎实例）。
    const watchdog = window.setInterval(() => {
      const st = store.getState();
      if (!st.running) return;
      app.SinEngineRunning().then((running) => {
        if (!running && store.getState().running) {
          dispatch({ type: "localCancel" });
          reconcileFinalAnswer();
        }
      }).catch((err) => logBridgeError("watchdog SinEngineRunning", err));
    }, 30000);
    return () => { window.clearInterval(watchdog); };
  }, [loadSessionData, reconcileFinalAnswer, store, dispatch]);

  const { send, steer, cancel, approve, answerQuestion } = useTurnActions(dispatch, store);
  const { newSession, listSessions, resumeSession, refreshContext } = useSessionActions(dispatch, loadItemsFoldedFirst);

  return { state, send, steer, cancel, approve, answerQuestion, newSession, listSessions, resumeSession, refreshContext, loadSessionData };
}

// useItems 订阅 items 数组（与办公同名同语义，独立实例）。
export function useItems(): import("./controller_state").Item[] {
  return useStore((s) => s.items);
}

// useTurnStartAt 当前回合开始时间戳（WorkHeader fork 消费）。
export function useTurnStartAt(): number {
  return useStore((s) => s.turnStartAt);
}
