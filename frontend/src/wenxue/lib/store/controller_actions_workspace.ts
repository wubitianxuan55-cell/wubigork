// controller_actions_workspace.ts — 工作区/模型族动作 hook（批 38 FE3-11
// hook 分域续刀三，自 controller.ts 原位搬移，零逻辑改动）：useWorkspaceActions
// （pickWorkspace/switchWorkspace 工作区两件 + setModel 会话环境一件；compact
// 已随批 52 B6① 裁决摘除——mock-only 绑定零调用者，真机压缩走后端会话事件）。
// refreshFactBase 跨族依赖摘除（2026-11-17 审查批6：factBase 链退役，见
// controller_actions_memory.ts 头注）。
import { useCallback, useEffect } from "react";
import { app } from "../bridge";
import { resetEventSync } from "../eventSync";
import type { Action } from "./controller_state";
import { failWrite, logBridgeError } from "./controller_actions_memory";

export function useWorkspaceActions(dispatch: (a: Action) => void) {
  // 图文工作空间变更刷新（2026-11-15 二次改造）：左栏《新建小说》落盘切换后
  // 广播「wenxue:workspace-changed」，这里做与 switchWorkspace 相同的复位+重载。
  const applyWorkspaceRefresh = useCallback(async () => {
    dispatch({ type: "reset" }); resetEventSync();
    try {
      dispatch({ type: "meta", meta: await app.Meta() });
      dispatch({ type: "context", context: await app.ContextUsage() });
    } catch (err) { logBridgeError("wenxue workspace refresh", err); }
  }, [dispatch]);
  useEffect(() => {
    const handler = () => { void applyWorkspaceRefresh(); };
    window.addEventListener("wenxue:workspace-changed", handler);
    return () => window.removeEventListener("wenxue:workspace-changed", handler);
  }, [applyWorkspaceRefresh]);
  const pickWorkspace = useCallback(async (): Promise<string> => {
    const p = await app.PickWorkspace().catch((err: unknown) => { failWrite(dispatch, "打开工作区", err); return ""; });
    if (p) {
      dispatch({ type: "reset" }); resetEventSync();
      try {
        dispatch({ type: "meta", meta: await app.Meta() });
        dispatch({ type: "context", context: await app.ContextUsage() });
      } catch (err) { logBridgeError("pickWorkspace refresh", err); }
    }
    return p;
  }, [dispatch]);
  const switchWorkspace = useCallback(async (path: string): Promise<string> => {
    const n = await app.SwitchWorkspace(path).catch((err: unknown) => { failWrite(dispatch, "切换工作区", err); return ""; });
    if (n) {
      dispatch({ type: "reset" }); resetEventSync();
      try {
        dispatch({ type: "meta", meta: await app.Meta() });
        dispatch({ type: "context", context: await app.ContextUsage() });
      } catch (err) { logBridgeError("switchWorkspace refresh", err); }
    }
    return n;
  }, [dispatch]);
  const setModel = useCallback(async (name: string) => {
    await app.SetModel(name).catch((err) => failWrite(dispatch, "切换模型", err));
    try {
      dispatch({ type: "meta", meta: await app.Meta() });
      dispatch({ type: "context", context: await app.ContextUsage() });
    } catch (err) { logBridgeError("setModel refresh", err); }
  }, [dispatch]);
  return { pickWorkspace, switchWorkspace, setModel };
}
