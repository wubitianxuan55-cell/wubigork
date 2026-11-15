// controller_actions_rewind.ts — 回退/重生成族动作 hook（批 38 FE3-11 hook
// 分域续刀三，自 controller.ts 原位搬移，零逻辑改动）：useRewindActions
// （rewind/regenerate 两件；跨族依赖 loadItemsFoldedFirst 与轮次族产物 send
// 由 useController 显式注入——regenerate 编排「先截断成功、再重发」依赖二者，
// useCallback 依赖数组逐字保留；store 同 turn 族以 { getState } 结构面注入）。
import { useCallback } from "react";
import { app } from "../bridge";
import { resetEventSync } from "../eventSync";
import type { Action, ControllerState, Item } from "./controller_state";
import { failWrite, logBridgeError } from "./controller_actions_memory";

export function useRewindActions(dispatch: (a: Action) => void, loadItemsFoldedFirst: () => Promise<void>, store: { getState: () => ControllerState }, send: (text: string, submitText?: string) => void) {
  const rewind = useCallback(async (turn: number, scope: string): Promise<boolean> => {
    // T7-4：回退失败不再静默，且不触发 reset——保留当前对话现场（否则刚
    // 弹的失败提示会被 reset 清空，用户连发生了什么都看不到）。
    // 返回是否成功（v4.232：regenerate 编排需要据此决定是否重发）。
    const act = (p: Promise<unknown>): Promise<boolean> =>
      p.then(() => true).catch((err) => { failWrite(dispatch, "回退对话", err); return false; });
    let ok = false;
    if (scope === "fork") ok = await act(app.Fork(turn));
    else if (scope === "summ-from") ok = await act(app.SummarizeFrom(turn));
    else if (scope === "summ-upto") ok = await act(app.SummarizeUpTo(turn));
    else ok = await act(app.Rewind(turn, scope));
    if (!ok) return false;
    dispatch({ type: "reset" });
    resetEventSync(); // v4.26：回退重载历史，seq 基线归零
    await loadItemsFoldedFirst(); // §8-5：折叠快照优先（日志序 id），History 保底
    app.ContextUsage().then(c => dispatch({ type: "context", context: c })).catch((err) => logBridgeError("rewind ContextUsage", err));
    return true;
  }, [dispatch, loadItemsFoldedFirst]);

  // 重新生成（v4.232，对齐同类产品 ChatGPT 式 regenerate）：丢弃最后一轮
  // （含用户消息），用同一文本原样重发。复用 GaeaRewind+GaeaSend 既有原语
  // （零新绑定），编排层保证「先截断成功、再重发」，失败不动现场。
  // 只对最后一轮开放：更早的轮次要先生成中间轮才有意义，语义上交给回退。
  const regenerate = useCallback(async (turn: number): Promise<boolean> => {
    const cur = store.getState();
    if (cur.running || cur.pendingUser !== undefined) return false;
    const users = cur.items.filter((i): i is Extract<Item, { kind: "user" }> => i.kind === "user");
    if (turn < 0 || turn !== users.length - 1) return false;
    const target = users[turn];
    if (!target || !target.text.trim()) return false;
    const ok = await rewind(turn, "conversation");
    if (!ok) return false;
    send(target.text);
    return true;
  }, [store, rewind, send]);
  return { rewind, regenerate };
}
