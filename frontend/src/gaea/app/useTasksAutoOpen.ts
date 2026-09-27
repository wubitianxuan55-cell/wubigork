// useTasksAutoOpen — 新后台任务/子代理运行自动激活「任务」右栏视图。
// v4.63 对标 dsh better-sidebar 的 0→N 触发 + 500ms 去抖重臂 + 偏好开关；
// 会话切换首个快照只建基线不触发（detectNewRunRefs 纯函数）。执行顺序与
// deps 原样迁自 App.tsx。
import { useCallback, useEffect, useRef } from "react";
import { onTaskEvent, workApp } from "../lib/bridge";
import { isWorkSpaceTask } from "../lib/taskSpace";
import { detectNewRunRefs } from "../lib/subagentRunsStore";
import type { SubagentRunView, TaskView } from "../lib/types";
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

  // 新后台任务（queued/running/stopping 事件）→ 自动激活任务页（宽屏；可关）。
  // 2026-09-27 两道收口（用户拍板「右侧面板默认隐藏，不要打开办公板块就打开」）：
  //  ① 挂载基线——先拉一次既有任务表把已存在的 id 记为已见，连接后引擎补推的
  //     既有任务快照事件不再当「新」触发；种子未落定前事件进缓冲，落定后统一
  //     入册（真正新出现的任务仍照常触发，v4.63 设计行为不变）。
  //  ② 会话跟随——事件带 session_id 且当前 UI 会话已定时，他 session 的任务
  //     不触发本会话的右栏自动展开。
  const seenTaskIdsRef = useRef<Set<string>>(new Set());
  const taskSeedDoneRef = useRef(false);
  const pendingTaskEventsRef = useRef<TaskView[]>([]);
  const openTasksAutoRef = useRef(openTasksAuto);
  openTasksAutoRef.current = openTasksAuto;
  useEffect(() => {
    let off: (() => void) | null = null;
    let cancelled = false;
    workApp
      .TaskList("")
      .then((list) => {
        if (cancelled) return;
        for (const t of (list ?? []).filter(isWorkSpaceTask)) seenTaskIdsRef.current.add(t.id);
        for (const t of pendingTaskEventsRef.current) seenTaskIdsRef.current.add(t.id);
        pendingTaskEventsRef.current = [];
        taskSeedDoneRef.current = true;
      })
      .catch(() => {
        // 种子失败：按「无既有任务」处理，直接放行缓冲事件（保持自动展开可用）
        if (cancelled) return;
        for (const t of pendingTaskEventsRef.current) seenTaskIdsRef.current.add(t.id);
        pendingTaskEventsRef.current = [];
        taskSeedDoneRef.current = true;
      });
    off = onTaskEvent((t) => {
      if (t.status !== "queued" && t.status !== "running" && t.status !== "stopping") return;
      if (currentSessionPath && t.session_id && t.session_id !== currentSessionPath) return;
      if (!taskSeedDoneRef.current) {
        pendingTaskEventsRef.current.push(t);
        return;
      }
      if (seenTaskIdsRef.current.has(t.id)) return;
      seenTaskIdsRef.current.add(t.id);
      openTasksAutoRef.current();
    }, "work");
    return () => {
      cancelled = true;
      off?.();
    };
    // openTasksAuto 经 ref 取最新（避免为订阅而重拉种子）；currentSessionPath
    // 内联进回调：会话切换后新事件按新会话判定
  }, [currentSessionPath]);

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