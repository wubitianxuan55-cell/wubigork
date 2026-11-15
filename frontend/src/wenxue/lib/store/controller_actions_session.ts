// controller_actions_session.ts — 会话管理族动作 hook（批 37 FE3-11 hook 分域
// 续刀，自 controller.ts 原位搬移，零逻辑改动）：useSessionActions（newSession/
// listSessions/listProjectSessions/fetchSessionStats/resumeSession/archiveSession/
// unarchiveSession/pinSession/deleteSession/renameSession/refreshMeta 十一件，
// useCallback 依赖数组逐字保留——loadItemsFoldedFirst/refreshFactBase 两个跨族
// 依赖由 useController 显式注入；fetchSessionStats 为域内件，被 resumeSession
// 消费的同时对外导出）。useStore/ensureEventsBound/useController 主体见 controller.ts。
import { useCallback } from "react";
import { app } from "../bridge";
import { resetEventSync } from "../eventSync";
import type { Action } from "./controller_state";
import type { HistoryMessage, ProjectGroup, SessionMeta } from "../types";
import { failWrite, logBridgeError } from "./controller_actions_memory";

export function useSessionActions(dispatch: (a: Action) => void, loadItemsFoldedFirst: () => Promise<void>, refreshFactBase: () => void) {
  const newSession = useCallback(async () => {
    try {
      await app.NewSession();
      dispatch({ type: "reset" });
      resetEventSync(); // v4.26：新会话事件 seq 从 1 重新单调递增，补拉防线基线归零
      refreshFactBase();
    } catch (err) {
      // 新建失败不重置界面（后端会话未切换），给出可见提示。
      failWrite(dispatch, "新建会话", err);
    }
  }, [dispatch, refreshFactBase]);
  const listSessions = useCallback((): Promise<SessionMeta[]> =>
    app.ListSessions().catch((err) => { logBridgeError("listSessions", err); return [] as SessionMeta[]; }), []);
  const listProjectSessions = useCallback((): Promise<ProjectGroup[]> =>
    app.ListProjectSessions().catch((err) => { logBridgeError("listProjectSessions", err); return [] as ProjectGroup[]; }), []);
  // fetchSessionStats 拉取会话级派生统计并写入 store；失败/无日志标记为不可用
  // （不阻塞恢复流程，仅影响统计面板的历史成本展示）。
  const fetchSessionStats = useCallback((path: string) => {
    app.SessionStats(path)
      .then((stats) => dispatch({ type: "sessionStats", stats }))
      .catch((err) => { logBridgeError("fetchSessionStats", err); dispatch({ type: "sessionStats", stats: undefined }); });
  }, [dispatch]);
  const resumeSession = useCallback(async (path: string) => {
    await app.ResumeSession(path).catch((e: unknown) => {
      // 恢复失败不要静默清空：给用户明确提示
      dispatch({
        type: "event",
        e: { kind: "notice", level: "warn", text: `恢复会话失败：${e instanceof Error ? e.message : String(e)}` },
      });
      return [] as HistoryMessage[];
    });
    dispatch({ type: "reset" });
    resetEventSync(); // v4.26：恢复会话同样归零 seq 基线
    await loadItemsFoldedFirst(); // §8-5：折叠快照优先（日志序 id），History 保底
    // 恢复后回填会话级派生统计（成本/用量历史，评审缺陷 11 根治）
    void fetchSessionStats(path);
    app.ContextUsage().then(c => dispatch({ type: "context", context: c })).catch((err) => logBridgeError("resumeSession ContextUsage", err));
    refreshFactBase();
  }, [dispatch, loadItemsFoldedFirst, refreshFactBase, fetchSessionStats]);
  const archiveSession = useCallback((path: string) => app.ArchiveSession(path).catch((err) => failWrite(dispatch, "归档会话", err)), [dispatch]);
  const unarchiveSession = useCallback((path: string): Promise<string> => app.UnarchiveSession(path).catch((err) => { failWrite(dispatch, "取消归档", err); return ""; }), [dispatch]);
  const pinSession = useCallback((path: string, pinned: boolean) => app.PinSession(path, pinned).catch((err) => failWrite(dispatch, "更新固定状态", err)), [dispatch]);
  const deleteSession = useCallback((path: string) => app.DeleteSession(path).catch((err) => failWrite(dispatch, "删除会话", err)), [dispatch]);
  const renameSession = useCallback((path: string, title: string) => app.RenameSession(path, title).catch((err) => failWrite(dispatch, "重命名会话", err)), [dispatch]);
  const refreshMeta = useCallback(async () => {
    try {
      dispatch({ type: "meta", meta: await app.Meta() });
      dispatch({ type: "context", context: await app.ContextUsage() });
    } catch (err) { logBridgeError("refreshMeta", err); }
  }, [dispatch]);
  return { newSession, listSessions, listProjectSessions, fetchSessionStats, resumeSession, archiveSession, unarchiveSession, pinSession, deleteSession, renameSession, refreshMeta };
}
