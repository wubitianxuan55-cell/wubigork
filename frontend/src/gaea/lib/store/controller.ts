// Zustand store — 主 controller reducer 状态机（useStore/applyEvent/reducer/useController）。
// P4 结构刀：由原 lib/store.ts 按 store 域拆分而来，本文件为 controller 域，导出名逐字节保留。


import { useEffect } from "react";
import { invalidateTurnCaches } from "../deliverablesTurn";
import { create } from "zustand";
import { useShallow } from "zustand/shallow";
import { app, onEvent, onReady } from "../bridge";
import { noteEventSeq } from "../eventSync";
import { logBridgeError, useMemoryActions } from "./controller_actions_memory";
import { useSessionActions } from "./controller_actions_session";
import { useTurnActions } from "./controller_actions_turn";
import { useWorkspaceActions } from "./controller_actions_workspace";
import { useRewindActions } from "./controller_actions_rewind";
import { useSessionLoaders } from "./controller_actions_load";
export * from "./controller_state";
import { initialState, reducer } from "./controller_state";
import type { Action, ControllerState, Item } from "./controller_state";
export const useStore = create<ControllerState>()((set) => ({ ...initialState, _dispatch: (a: Action) => set((s) => reducer(s, a)) } as ControllerState));

// isFinalAnswerRendered 随会话装载族出仓（controller_actions_load.ts），
// 命名 re-export 保 ./store 测试导入路径不变。
export { isFinalAnswerRendered } from "./controller_actions_load";

// ── 全局事件/就绪绑定（恰好一次，v4.119 刀10）────────────────────────
// 进度计划板块的 AI 对话 pane 与办公板块 GaeaApp 共享本模块的会话 store
// （keepAlive 树下两者可能同时挂载）。事件绑定必须**恰好一次**：多订阅会让
// text/reasoning 增量重复 dispatch（气泡翻倍）。首个 useController 挂载时绑定，
// 之后任何宿主复用；卸载不退订——隐藏期间事件继续入库，回到板块即热态。
// 注入的回调都是 store 单例上的稳定闭包，跨宿主语义一致。
interface EventBindDeps {
  loadSessionData: () => Promise<void>;
  refreshFactBase: () => void;
  reconcileFinalAnswer: () => void;
  store: typeof useStore;
  dispatch: (a: Action) => void;
}
let eventsBound = false;
function ensureEventsBound(deps: EventBindDeps): void {
  if (eventsBound) return;
  eventsBound = true;
  const { dispatch, refreshFactBase, reconcileFinalAnswer } = deps;
  onEvent((e) => {
    // v4.26 事件序号防线：payload 带 seq（可选字段）时做缺口检测，命中缺口
    // 经注入的 fetcher（App.tsx 挂 app.GaeaResyncEvents）补拉后端折叠快照，
    // 以 resync action 落库。旧后端无 seq / 未挂 fetcher 时整条防线静默旁路；
    // 5s 冷却 + 在途去重防补拉风暴。resync 只补 items，不动 running/turnActive
    // （见 reducer case "resync"）。会话切换的 seq 基线归零在各 reset 调用点
    // 经 resetEventSync() 完成。
    noteEventSeq(e, {
      onSnapshot: (snap) => dispatch({ type: "resync", items: snap.items }),
      onError: (err) => logBridgeError("eventSync 补拉", err),
    });
    // 流式 text/reasoning 用 queueMicrotask 确保每次 chunk 即时渲染，
    // 不被 React 18 自动批处理合并。同步 dispatch 会导致多个事件在同一
    // 微任务中批量更新从而不渲染中间态。
    if (e.kind === "text" || e.kind === "reasoning") {
      queueMicrotask(() => dispatch({ type: "event", e }));
    } else {
      dispatch({ type: "event", e });
    }
    if (e.kind === "turn_done") {
      // v4.382 失效接线（v4.354 记录的 dirListingsCache 欠账）：轮完成后
      // 登记与目录探测缓存整包失效——新轮交付物/新建文件不再被 TTL 内的
      // 旧探测误标缺失。invalidateTurnCaches 纯内存清空，零桥接成本。
      invalidateTurnCaches();
      app.ContextUsage().then(c => dispatch({ type: "context", context: c })).catch((err) => logBridgeError("turn_done ContextUsage", err));
      app.Balance().then(b => dispatch({ type: "balance", balance: b })).catch((err) => logBridgeError("turn_done Balance", err));
      reconcileFinalAnswer();
    }
    if (e.kind === "turn_done" || e.kind === "notice") {
      app.Jobs().then(j => dispatch({ type: "jobs", jobs: j })).catch((err) => logBridgeError("Jobs", err));
      refreshFactBase();
    }
    if (e.kind === "tool_result" && e.tool?.name?.startsWith("fact_")) {
      refreshFactBase();
    }
  });
  onReady(() => {
    void deps.loadSessionData();
    app.Balance().then(b => dispatch({ type: "balance", balance: b })).catch((err) => logBridgeError("onReady Balance", err));
    app.Jobs().then(j => dispatch({ type: "jobs", jobs: j })).catch((err) => logBridgeError("onReady Jobs", err));
    refreshFactBase();
  });
  void deps.loadSessionData();
  app.Balance().then(b => dispatch({ type: "balance", balance: b })).catch((err) => logBridgeError("init Balance", err));
  app.Jobs().then(j => dispatch({ type: "jobs", jobs: j })).catch((err) => logBridgeError("init Jobs", err));
  refreshFactBase();
}

export function useController() {
  const store = useStore;
  const state = store(useShallow(s => s));
  const dispatch = store.getState()._dispatch;

  const { loadItemsFoldedFirst, loadSessionData, reconcileFinalAnswer, refreshFactBase } = useSessionLoaders(dispatch, store);

  useEffect(() => {
    // v4.119 刀10：事件/就绪绑定提升为模块级恰好一次（见 ensureEventsBound）——
    // 进度计划板块的 AI 对话 pane 与本板块 GaeaApp 在 keepAlive 树下可能同时
    // 挂载，多订阅会让 text/reasoning 增量重复 dispatch（气泡翻倍）。
    ensureEventsBound({ loadSessionData, refreshFactBase, reconcileFinalAnswer, store, dispatch });
    // 看门狗：running=true 时每 30s 用后端真实状态校准一次，防止
    // turn_done 事件丢失导致界面永久卡在“执行中”。（每宿主各自运行：
    // 只读校准 + localCancel 幂等，多宿主共存无害。）
    const watchdog = window.setInterval(() => {
      const st = store.getState();
      if (!st.running) return;
      app.GaeaRunning().then((running) => {
        if (!running && store.getState().running) {
          dispatch({ type: "localCancel" });
          reconcileFinalAnswer();
        }
      }).catch((err) => logBridgeError("watchdog GaeaRunning", err));
    }, 30000);
    return () => { window.clearInterval(watchdog); };
  }, [loadSessionData, refreshFactBase, reconcileFinalAnswer, store, dispatch]);

  const { send, steer, cancel, approve, answerQuestion, setPermLevel } = useTurnActions(dispatch, store);
  const { newSession, listSessions, listProjectSessions, fetchSessionStats, resumeSession, archiveSession, unarchiveSession, pinSession, deleteSession, renameSession, refreshMeta } = useSessionActions(dispatch, loadItemsFoldedFirst, refreshFactBase);
  const { pickWorkspace, switchWorkspace, compact, setModel } = useWorkspaceActions(dispatch, refreshFactBase);
  const { fetchMemory, remember, forget, saveDoc, updateFact, changeFactType, clearFactBase, promoteFactBase } = useMemoryActions(dispatch, refreshFactBase);
  const { rewind, regenerate } = useRewindActions(dispatch, loadItemsFoldedFirst, store, send);

  return { state, send, steer, cancel, approve, answerQuestion, setPermLevel, newSession, listSessions, listProjectSessions, resumeSession, archiveSession, unarchiveSession, pinSession, deleteSession, renameSession, refreshMeta, pickWorkspace, switchWorkspace, compact, rewind, setModel, fetchMemory, remember, forget, saveDoc, updateFact, changeFactType, clearFactBase, promoteFactBase, fetchSessionStats, regenerate };
}

// useItems 订阅 items 数组，与 useController 分离。
// 流式输出时 items 高频变化（每次 text/reasoning 事件），通过独立 hook 避免
// useController 的 store(s=>s) 全量订阅导致 App 树全局重渲染。
// 使用 useShallow 做浅比较：仅当 items 长度或元素引用变化时才触发重渲染，
// 非 items 字段（meta/context/balance 等）的变化不会影响此 hook。
export function useItems(): Item[] {
  return useStore(s => s.items);
}

// useTurnStartAt 返回当前回合开始时间戳(ms)，用于计算思考耗时。
export function useTurnStartAt(): number {
  return useStore(s => s.turnStartAt);
}
