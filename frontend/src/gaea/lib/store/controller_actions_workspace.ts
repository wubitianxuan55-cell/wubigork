// controller_actions_workspace.ts — 工作区/模型族动作 hook（批 38 FE3-11
// hook 分域续刀三，自 controller.ts 原位搬移，零逻辑改动）：useWorkspaceActions
// （pickWorkspace/switchWorkspace 工作区两件 + compact/setModel 会话环境两件，
// 件数不足以各立文件故同仓；useCallback 依赖数组逐字保留——跨族依赖
// refreshFactBase 由 useController 显式注入）。
import { useCallback } from "react";
import { app } from "../bridge";
import { resetEventSync } from "../eventSync";
import type { Action } from "./controller_state";
import { errText, failWrite, logBridgeError } from "./controller_actions_memory";

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
  // FE3-04：Compact 是 mock-only 绑定（Go 侧无对应绑定，drift.ts MOCK_ONLY_NAMES
  // 单源），真机上下文压缩由后端会话事件自动执行、无手动入口。真机调用现在从
  // proxy 拿到显式 BridgeError（MockOnlyBinding），失败**不可重试**——不走
  // failWrite（其固定追加「请重试」会误导），诚实提示一次性原因。dev mock 下
  // Compact 正常 resolve，本 catch 不触发，行为不变。
  const compact = useCallback(() => {
    app.Compact().catch((err) => {
      logBridgeError("compact", err);
      dispatch({ type: "event", e: { kind: "notice", level: "warn", text: `压缩上下文失败：${errText(err)}` } });
    });
  }, [dispatch]);
  const setModel = useCallback(async (name: string) => {
    await app.SetModel(name).catch((err) => failWrite(dispatch, "切换模型", err));
    try {
      dispatch({ type: "meta", meta: await app.Meta() });
      dispatch({ type: "context", context: await app.ContextUsage() });
    } catch (err) { logBridgeError("setModel refresh", err); }
  }, [dispatch]);
  return { pickWorkspace, switchWorkspace, compact, setModel };
}
