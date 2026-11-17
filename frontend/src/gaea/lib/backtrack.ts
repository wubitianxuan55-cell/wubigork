import type { Item } from "./store/controller_state";

export type BacktrackReason = "in-flight" | "missing-prompt" | "partial-history" | "hidden-prompt";

export type BacktrackDecision =
  | { ok: true; prompt: string; userId: string }
  | { ok: false; reason: BacktrackReason; message: string };

export interface BacktrackInput {
  items: Item[];
  turn: number;
  running?: boolean;
  turnActive?: boolean;
  pendingUser?: string;
  historyComplete?: boolean;
  hiddenUserIds?: ReadonlySet<string>;
}

const REASON_TEXT: Record<BacktrackReason, string> = {
  "in-flight": "当前回合尚未结束，不能回退",
  "missing-prompt": "目标回合不存在或提示词为空",
  "partial-history": "会话历史不完整，不能安全回退",
  "hidden-prompt": "目标提示词不可编辑",
};

/**
 * 回退是历史截断，必须在未运行、目标完整且可见时才允许。
 * turn 是前端用户消息序号（0-based）；中途插话不落用户条目，不会伪装成回合起点。
 */
export function canBacktrack(input: BacktrackInput): BacktrackDecision {
  if (input.running || input.turnActive || input.pendingUser !== undefined) {
    return { ok: false, reason: "in-flight", message: REASON_TEXT["in-flight"] };
  }
  if (input.historyComplete === false) {
    return { ok: false, reason: "partial-history", message: REASON_TEXT["partial-history"] };
  }
  if (!Number.isInteger(input.turn) || input.turn < 0) {
    return { ok: false, reason: "missing-prompt", message: REASON_TEXT["missing-prompt"] };
  }

  const users = input.items.filter((item): item is Extract<Item, { kind: "user" }> => item.kind === "user");
  const target = users[input.turn];
  if (!target || target.text.trim() === "") {
    return { ok: false, reason: "missing-prompt", message: REASON_TEXT["missing-prompt"] };
  }
  if (input.hiddenUserIds?.has(target.id)) {
    return { ok: false, reason: "hidden-prompt", message: REASON_TEXT["hidden-prompt"] };
  }
  return { ok: true, prompt: target.text, userId: target.id };
}
