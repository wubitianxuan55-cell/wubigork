// useTasksAutoOpen — 新后台任务/子代理运行自动激活「任务」右栏视图。
// v4.63 对标 dsh better-sidebar 的 0→N 触发 + 500ms 去抖重臂 + 偏好开关；
// 会话切换首个快照只建基线不触发（detectNewRunRefs 纯函数）。执行顺序与
// deps 原样迁自 App.tsx。
// 2026-09-27 用户拍板「右侧面板启动默认隐藏，不要打开办公板块就打开」：
// 偏好默认改关（设置中心显式开启才生效），且会话切换改挂起基线——
// 等新会话首个快照建基线，旧会话/空集基线不再把新会话子代理误判为「新」。
import { useCallback, useEffect, useRef } from "react";
import { onTaskEvent, workApp } from "../lib/bridge";
import { loadTasksAutoOpenSubagent } from "../lib/tasksPrefs";
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
  // 2026-09-27 默认改关（用户拍板「右侧面板启动默认隐藏，不要打开办公板块
  // 就打开」）：只在设置中心显式开启（"1"）后，新任务/子代理才自动切任务视图。
  // 判定单一源 = lib/tasksPrefs（"1"/"true" 为开，其余默认关）。
  const openTasksAuto = useCallback(() => {
    if (!loadTasksAutoOpenSubagent()) return;
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

  const seenSubagentRefsRef = useRef<{ path: string; pendingBaseline: boolean; refs: Set<string> }>({
    path: "", pendingBaseline: false, refs: new Set(),
  });
  const autoOpenTasksTimerRef = useRef(0);
  const subagentRunsRef = useRef<SubagentRunView[]>([]);
  useEffect(() => { subagentRunsRef.current = subagentRuns; }, [subagentRuns]);
  useEffect(() => {
    const path = currentSessionPath ?? "";
    const seen = seenSubagentRefsRef.current;
    if (seen.path !== path) {
      // 会话切换：作废旧基线并挂起，等新会话的首个快照只建基线不触发。
      // 此刻 subagentRunsRef 可能还是旧会话的（快照在途），立刻照它建基线
      // 会把空/旧集当基线——新会话的真实子代理全被判「新」误触发
      // （2026-09-27 用户实锤：进板块/切会话右栏被撑开）。
      seen.path = path;
      seen.pendingBaseline = true;
      seen.refs = new Set();
      window.clearTimeout(autoOpenTasksTimerRef.current);
      return;
    }
    if (seen.pendingBaseline) {
      // 新会话首个快照：只建基线（若快照缺失/失败则下个快照补建）
      seen.pendingBaseline = false;
      seen.refs = new Set(subagentRunsRef.current.map((r) => r.ref));
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