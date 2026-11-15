// controller_actions_session.ts — sin fork：会话管理族（newSession/listSessions/
// resumeSession/refreshContext 四件）。归档/置顶/删除/重命名走刀7（sin 会话
// 操作绑定随会话管理刀落地），办公的 GaeaSession 族不进 sin 面。
import { useCallback } from "react";
import { app } from "../../../gaea/lib/bridge";
import { resetEventSync } from "./eventSync";
import type { Action } from "./controller_state";
import type { HistoryMessage, SessionMeta } from "../../../gaea/lib/types";
import { failWrite, logBridgeError } from "./controller_actions_memory";

export function useSessionActions(dispatch: (a: Action) => void, loadItemsFoldedFirst: () => Promise<void>) {
  const newSession = useCallback(async () => {
    try {
      await app.SinEngineNewSession();
      dispatch({ type: "reset" });
      resetEventSync(); // 新会话事件 seq 从 1 重新单调递增，补拉防线基线归零
      await loadItemsFoldedFirst();
      app.SinEngineContext().then((c) => dispatch({ type: "context", context: c })).catch((err) => logBridgeError("newSession SinEngineContext", err));
    } catch (err) {
      // 新建失败不重置界面（后端会话未切换），给出可见提示。
      failWrite(dispatch, "新建会话", err);
    }
  }, [dispatch, loadItemsFoldedFirst]);

  const listSessions = useCallback((): Promise<SessionMeta[]> =>
    app.SinEngineListSessions().catch((err) => { logBridgeError("listSessions", err); return [] as SessionMeta[]; }), []);

  const resumeSession = useCallback(async (path: string) => {
    await app.SinEngineResumeSession(path).catch((e: unknown) => {
      dispatch({
        type: "event",
        e: { kind: "notice", level: "warn", text: `恢复会话失败：${e instanceof Error ? e.message : String(e)}` },
      });
      return [] as HistoryMessage[];
    });
    dispatch({ type: "reset" });
    resetEventSync();
    await loadItemsFoldedFirst(); // §8-5：折叠快照优先（日志序 id），History 保底
    app.SinEngineContext().then((c) => dispatch({ type: "context", context: c })).catch((err) => logBridgeError("resumeSession SinEngineContext", err));
  }, [dispatch, loadItemsFoldedFirst]);

  const refreshContext = useCallback(async () => {
    try {
      dispatch({ type: "context", context: await app.SinEngineContext() });
    } catch (err) { logBridgeError("refreshContext", err); }
  }, [dispatch]);

  return { newSession, listSessions, resumeSession, refreshContext };
}
