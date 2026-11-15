// controller_actions_turn.ts — 轮次控制族动作 hook（批 38 FE3-11 hook 分域
// 续刀三，自 controller.ts 原位搬移，零逻辑改动）：useTurnActions（send/steer/
// cancel/approve/answerQuestion/setPermLevel 六件，useCallback 依赖数组逐字
// 保留——cancel 的运行态读取由 useController 以 { getState } 结构面注入，
// 不回导 useStore 防 controller→actions→controller 环）。useStore/
// ensureEventsBound/useController 主体见 controller.ts。
import { useCallback } from "react";
import { app } from "../bridge";
import type { Action, ControllerState } from "./controller_state";
import type { QuestionAnswer } from "../types";
import { failWrite } from "./controller_actions_memory";

export function useTurnActions(dispatch: (a: Action) => void, store: { getState: () => ControllerState }) {
  // T7-4：send 失败不再静默——保留已上屏的用户消息（可复制重发），
  // 记录 bridge 日志并给出用户可见的失败提示。
  const send = useCallback((displayText: string, submitText = displayText) => {
    dispatch({ type: "user", text: displayText });
    const display = displayText.trim(); const submit = submitText.trim();
    const p = display !== submit ? app.SubmitDisplay(display, submit) : app.Submit(submit);
    p.catch((err) => failWrite(dispatch, "发送消息", err));
  }, [dispatch]);

  // 运行中插话调整（2026-08-28，对齐豆包工作「边跑边改」）：消息注入当前
  // 回合作为补充指引，不打断执行、不开新回合；不落用户气泡（后端以 notice
  // 回显）。未运行时后端 Steer 内部兜底走 Submit 排队。
  const steer = useCallback((text: string) => {
    const t = text.trim();
    if (!t) return;
    app.Steer(t).catch((err) => failWrite(dispatch, "插话调整", err));
  }, [dispatch]);

  const cancel = useCallback((): string | undefined => {
    const cur = store.getState();
    const onFail = (err: unknown) => failWrite(dispatch, "取消", err);
    if (cur.running && cur.pendingUser !== undefined) { const text = cur.pendingUser; dispatch({ type: "unsend" }); app.Cancel().catch(onFail); return text; }
    if (cur.running) dispatch({ type: "localCancel" }); // 事件丢失时仍能复位本地运行态
    app.Cancel().catch(onFail); return undefined;
  }, [store, dispatch]);

  // T7-4：approve/answerQuestion 失败时不清掉弹窗（保留审批/提问界面），
  // 记录日志并提示用户重试；成功后才清除。
  const approve = useCallback((id: string, decision: "allow_once" | "allow_session" | "persist_allow" | "deny" | "abort") => {
    app.Approve(id, decision)
      .then(() => dispatch({ type: "clearApproval" }))
      .catch((err) => failWrite(dispatch, "审批提交", err));
  }, [dispatch]);
  const answerQuestion = useCallback((id: string, answers: QuestionAnswer[]) => {
    app.AnswerQuestion(id, answers)
      .then(() => dispatch({ type: "clearAsk" }))
      .catch((err) => failWrite(dispatch, "回答提交", err));
  }, [dispatch]);
  const setPermLevel = useCallback((level: string) => { app.SetPermLevel(level).catch((err) => failWrite(dispatch, "切换权限级别", err)); }, [dispatch]);
  return { send, steer, cancel, approve, answerQuestion, setPermLevel };
}
