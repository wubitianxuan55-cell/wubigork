import { useEffect, useState } from "react";

// Global clock — a single 1s interval shared by all ToolCards, replacing N
// independent setInterval calls (one per running tool). Components subscribe
// via useNow() and compute their own elapsed time from their own start ref.
let listeners: Array<() => void> = [];
let timerId: ReturnType<typeof setInterval> | null = null;

function subscribe(onChange: () => void) {
  listeners.push(onChange);
  if (!timerId) {
    timerId = setInterval(() => {
      for (const fn of listeners) fn();
    }, 1000);
  }
  return () => {
    listeners = listeners.filter(f => f !== onChange);
    if (listeners.length === 0 && timerId !== null) {
      clearInterval(timerId);
      timerId = null;
    }
  };
}

/**
 * Returns the current Unix timestamp in seconds, refreshed every 1s.
 * active=false 时不订阅全局时钟（返回最后一次值即冻结）：已完成的消息/过程卡
 * 若无条件订阅，长会话数百条消息每秒全体重渲染，O(N) 常驻开销随会话线性涨
 * （v4.350）。全部订阅者转 false 后时钟自动停摆（subscribe 的 listeners 计数）。
 */
export function useNow(active = true): number {
  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000));
  useEffect(() => {
    if (!active) return;
    return subscribe(() => setNow(Math.floor(Date.now() / 1000)));
  }, [active]);
  return now;
}
