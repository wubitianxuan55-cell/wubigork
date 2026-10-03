// controller_actions_workspace.ts — 工作区/模型族动作 hook（批 38 FE3-11
// hook 分域续刀三，自 controller.ts 原位搬移，零逻辑改动）：useWorkspaceActions
// （pickWorkspace/switchWorkspace 工作区两件 + setModel 会话环境一件；compact
// 已随批 52 B6① 裁决摘除——mock-only 绑定零调用者，真机压缩走后端会话事件；
// useCallback 依赖数组逐字保留——跨族依赖 refreshFactBase 由 useController
// 显式注入）。
import { useCallback } from "react";
import { app } from "../bridge";
import { resetEventSync } from "../eventSync";
import type { Action } from "./controller_state";
import { failWrite, logBridgeError } from "./controller_actions_memory";

export function useWorkspaceActions(dispatch: (a: Action) => void, refreshFactBase: () => void) {
  const pickWorkspace = useCallback(async (): Promise<string> => {
    const p = await app.PickWorkspace().catch((err: unknown) => { failWrite(dispatch, "打开工作区", err); return ""; });
    if (p) {
      dispatch({ type: "reset" }); resetEventSync(); refreshFactBase();
      try {
        dispatch({ type: "meta", meta: await app.Meta() });
        dispatch({ type: "context", context: await app.ContextUsage() });
      } catch (err) { logBridgeError("pickWorkspace refresh", err); }
    }
    return p;
  }, [dispatch, refreshFactBase]);
  const switchWorkspace = useCallback(async (path: string): Promise<string> => {
    const n = await app.SwitchWorkspace(path).catch((err: unknown) => { failWrite(dispatch, "切换工作区", err); return ""; });
    if (n) {
      dispatch({ type: "reset" }); resetEventSync(); refreshFactBase();
      try {
        dispatch({ type: "meta", meta: await app.Meta() });
        dispatch({ type: "context", context: await app.ContextUsage() });
      } catch (err) { logBridgeError("switchWorkspace refresh", err); }
    }
    return n;
  }, [dispatch, refreshFactBase]);
  const setModel = useCallback(async (name: string) => {
    await app.SetModel(name).catch((err) => failWrite(dispatch, "切换模型", err));
    try {
      dispatch({ type: "meta", meta: await app.Meta() });
      dispatch({ type: "context", context: await app.ContextUsage() });
    } catch (err) { logBridgeError("setModel refresh", err); }
  }, [dispatch]);
  return { pickWorkspace, switchWorkspace, setModel };
}
