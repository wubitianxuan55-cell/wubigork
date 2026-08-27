import { useEffect, useRef, useState } from "react";
import { app, onTaskEvent } from "../lib/bridge";
import type { TaskStatus, TaskView } from "../lib/types";

// useRunningBadge — 运行域活动角标计数（蒸馏 dsh-better-sidebar badge，C6）。
// 维护「当前活跃任务数」（queued + running），供右侧面板「运行」主 Tab 角标。
// 数据源：初始 GaeaTaskList 建立基线 + gaea-task 事件实时增量（与 TaskCenter
// 同源，零新增 RPC）；事件到达时按任务 id 全量重算活跃数（简单可靠，任务量小）。
// 设计取舍：
//  - 只统计任务，不统计子代理：子代理运行数需要新增 App 层轮询（SubagentsPanel
//    内已有 5s 轮询，App 层重复轮询浪费）；任务事件已存在，零成本。
//  - 角标只在「运行」组未激活时显示（激活即视为已读），由 App 决定传入时机。
function isActive(status: TaskStatus): boolean {
  return status === "queued" || status === "running";
}

export function useRunningBadge(): number {
  const [count, setCount] = useState(0);
  const tasksRef = useRef<Map<string, TaskView>>(new Map());

  // 初始基线 + 事件增量：统一走「任务表快照 → 重算活跃数」。
  // 事件可能先于初始拉取返回（Wails 事件与绑定调用无序），两路都全量重算，
  // 以最新任务表为准，天然幂等。
  useEffect(() => {
    let cancelled = false;
    const recalc = () => {
      if (cancelled) return;
      let n = 0;
      for (const t of tasksRef.current.values()) {
        if (isActive(t.status)) n += 1;
      }
      setCount(n);
    };
    const off = onTaskEvent((t) => {
      tasksRef.current.set(t.id, t);
      recalc();
    });
    app
      .TaskList()
      .then((list) => {
        if (cancelled) return;
        for (const t of list ?? []) tasksRef.current.set(t.id, t);
        recalc();
      })
      .catch(() => {
        /* 拉取失败：以事件流为准，静默 */
      });
    return () => {
      cancelled = true;
      off();
      tasksRef.current.clear();
    };
  }, []);

  return count;
}
