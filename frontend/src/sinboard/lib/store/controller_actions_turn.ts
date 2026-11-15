// controller_actions_turn.ts — sin fork：轮次控制族（send/steer/cancel/approve/
// answerQuestion 五件；办公的 setPermLevel 不在 sin 面——权限档位走刀7）。
// 绑定全部落 SinEngine*（独立引擎实例，与办公 Gaea* 平行）。
import { useCallback } from "react";
import { app } from "../../../gaea/lib/bridge";
import type { Action, ControllerState } from "./controller_state";
import type { QuestionAnswer } from "../../../gaea/lib/types";
import { failWrite } from "./controller_actions_memory";

export function useTurnActions(dispatch: (a: Action) => void, store: { getState: () => ControllerState }) {
  // send：display 是上屏文本，submit 是实际发模型文本（Composer 折叠长文展开）。
  // sin 输入富化（角色/底稿块）在后端 SinEngineSend 内完成。
  const send = useCallback((displayText: string, submitText = displayText) => {
    dispatch({ type: "user", text: displayText });
    const submit = (submitText || displayText).trim();
    app.SinEngineSend(submit).catch((err) => failWrite(dispatch, "发送消息", err));
  }, [dispatch]);

  const steer = useCallback((text: string) => {
    const t = text.trim();
    if (!t) return;
    app.SinEngineSteer(t).catch((err) => failWrite(dispatch, "插话调整", err));
  }, [dispatch]);

  const cancel = useCallback((): string | undefined => {
    const cur = store.getState();
    const onFail = (err: unknown) => failWrite(dispatch, "取消", err);
    if (cur.running && cur.pendingUser !== undefined) { const text = cur.pendingUser; dispatch({ type: "unsend" }); app.SinEngineCancel().catch(onFail); return text; }
    if (cur.running) dispatch({ type: "localCancel" }); // 事件丢失时仍能复位本地运行态
    app.SinEngineCancel().catch(onFail); return undefined;
  }, [store, dispatch]);

  // approve/answerQuestion 失败时不清弹窗（保留审批/提问界面），成功才清除。
  const approve = useCallback((id: string, decision: "allow_once" | "allow_session" | "persist_allow" | "deny" | "abort") => {
    app.SinEngineApprove(id, decision)
      .then(() => dispatch({ type: "clearApproval" }))
      .catch((err) => failWrite(dispatch, "审批提交", err));
  }, [dispatch]);
  const answerQuestion = useCallback((id: string, answers: QuestionAnswer[]) => {
    app.SinEngineAnswerQuestion(id, answers)
      .then(() => dispatch({ type: "clearAsk" }))
      .catch((err) => failWrite(dispatch, "回答提交", err));
  }, [dispatch]);
  return { send, steer, cancel, approve, answerQuestion };
}
