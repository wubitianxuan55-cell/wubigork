// useTasksAutoOpen — 新后台任务/子代理运行自动激活「任务」右栏视图。
// v4.63 对标 dsh better-sidebar 的 0→N 触发 + 500ms 去抖重臂 + 偏好开关；
// 会话切换首个快照只建基线不触发（detectNewRunRefs 纯函数）。执行顺序与
// deps 原样迁自 App.tsx。
import { useCallback, useEffect, useRef } from "react";
import { onTaskEvent } from "../lib/bridge";
import { detectNewRunRefs } from "../lib/subagentRunsStore";
import type { SubagentRunView } from "../lib/types";
import type { WorkspaceTabId } from "../lib/workspaceTabs";

interface UseTasksAutoOpenParams {
  rightTab: WorkspaceTabId | null;
  openPaneView: (viewId: WorkspaceTabId) => void;
  currentSessionPath: string | undefined;
  subagentRuns: SubagentRunView[];
}

export function useTasksAutoOpen({ rightTab, openPaneView, currentSessionPath, subagentRuns }: UseTasksAutoOpenParams) {
  // v4.63 自动展开（对标 dsh better-sidebar 的 0→N 触发 + 500ms 去抖重臂 +
  // 偏好开关）：当前会话出现新子代理/本地模型工具运行 → 自动切右栏「任务」
  // 视图（已在任务视图则只记角标语义，不打扰）。去抖的 Why 与 dsh #314 同源：
  // 派发帧与标题/状态帧分帧到达，立即判定会误触发/漏触发，等快照稳定后重评。
  // 偏好走 localStorage（默认开）；会话切换的首个快照只建立基线不触发。
  // v4.77 窄屏不强制展开：桌面宽度 < 1240 时 workspace-pane 被 CSS 隐藏，
  // 自动激活不切右栏（用户可手动打开，设置开关仍可整体关闭）。
  const openTasksAuto = useCallback(() => {
    try {
      if (localStorage.getItem("gaea.tasks.autoOpenSubagent") === "0") return;
    } catch { /* 私有模式：按默认开处理 */ }
    if (window.innerWidth < 1240) return;
    if (rightTab !== "tasks") openPaneView("tasks");
  }, [rightTab, openPaneView]);

  // 新后台任务（queued/running/stopping 事件）→ 自动激活任务页（宽屏；可关）
  const seenTaskIdsRef = useRef<Set<string>>(new Set());
  useEffect(() => {
    return onTaskEvent((t) => {
      if (t.status !== "queued" && t.status !== "running" && t.status !== "stopping") return;
      if (seenTaskIdsRef.current.has(t.id)) return;
      seenTaskIdsRef.current.add(t.id);
      openTasksAuto();
    }, "work");
  }, [openTasksAuto]);

  const seenSubagentRefsRef = useRef<{ path: string; initialized: boolean; refs: Set<string> }>({
    path: "", initialized: false, refs: new Set(),
  });
  const autoOpenTasksTimerRef = useRef(0);
  const subagentRunsRef = useRef<SubagentRunView[]>([]);
  useEffect(() => { subagentRunsRef.current = subagentRuns; }, [subagentRuns]);
  useEffect(() => {
    const path = currentSessionPath ?? "";
    const seen = seenSubagentRefsRef.current;
    if (seen.path !== path) {
      // 会话切换：以当前快照建立基线，绝不把历史子代理当「新」触发
      seen.path = path;
      seen.initialized = true;
      seen.refs = new Set(subagentRunsRef.current.map((r) => r.ref));
      window.clearTimeout(autoOpenTasksTimerRef.current);
      return;
    }
    const fresh = detectNewRunRefs(seen.refs, subagentRunsRef.current);
    if (fresh.length === 0) return;
    window.clearTimeout(autoOpenTasksTimerRef.current);
    autoOpenTasksTimerRef.current = window.setTimeout(() => {
      const still = detectNewRunRefs(seen.refs, subagentRunsRef.current);
      if (still.length === 0) return;
      for (const ref of still) seen.refs.add(ref);
      openTasksAuto();
    }, 500);
  }, [subagentRuns, currentSessionPath, openTasksAuto]);

  return { openTasksAuto };
}