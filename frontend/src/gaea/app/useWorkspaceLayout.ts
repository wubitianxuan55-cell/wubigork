// useWorkspaceLayout — 右栏工作台宽度：读档/视口钳制/全局键持久化/拖拽。
// v4.23 宽度是布局偏好而非会话内容（全局键胜出、会话快照兜底）；v4.27 视口
// 感知重钳；会话记录随存快照。执行顺序与 deps 原样迁自 App.tsx。
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { PointerEvent as ReactPointerEvent } from "react";
import { CHAT_MIN_WIDTH } from "./layout";
import {
  WORKSPACE_MIN_WIDTH, clampWorkspaceWidth,
  loadWorkspaceWidth, saveWorkspaceWidth,
  savePersistedRightPanelState,
  type WorkspacePanelPersistedState, type WorkspaceTabId,
} from "../lib/workspaceTabs";

interface UseWorkspaceLayoutParams {
  initialPanelState: WorkspacePanelPersistedState;
  rightTab: WorkspaceTabId | null;
  currentSessionKey: string;
  effectiveSidebarWidth: number;
}

export function useWorkspaceLayout({ initialPanelState, rightTab, currentSessionKey, effectiveSidebarWidth }: UseWorkspaceLayoutParams) {
  // v4.23 右栏宽度：全局键（最后一次拖拽胜出、跨会话即时跟随）；全局缺省时
  // 用会话快照兜底（学 better-sidebar：session state 自带 width，全局键读档胜出）。
  const [workspaceWidth, setWorkspaceWidth] = useState<number>(() => loadWorkspaceWidth(initialPanelState.width));
  const [workspaceResizing, setWorkspaceResizing] = useState(false);
  // ref镜像：会话记录写宽度快照用，拖拽中不触发记录 effect 逐帧写 localStorage
  const workspaceWidthRef = useRef(workspaceWidth);
  useEffect(() => { workspaceWidthRef.current = workspaceWidth; }, [workspaceWidth]);
  // v4.27 视口感知：窗口尺寸变化时按「视口 − 侧栏 − 对话区最小宽度」重钳右栏，
  // 避免持久化的宽面板在小窗口/放大窗口下挤出聊天区（学 ResizableDrawer 的 viewport 追踪）。
  const [viewportWidth, setViewportWidth] = useState(() => (typeof window === "undefined" ? 1440 : window.innerWidth));
  useEffect(() => {
    const onResize = () => setViewportWidth(window.innerWidth);
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  }, []);
  const maxWorkspaceByViewport = useMemo(
    () => Math.max(WORKSPACE_MIN_WIDTH, viewportWidth - effectiveSidebarWidth - CHAT_MIN_WIDTH),
    [viewportWidth, effectiveSidebarWidth],
  );
  // 渲染用有效宽度：读档/拖拽值与视口上限取小后钳制（CSS grid 不会溢出聊天区）
  const effectiveWorkspaceWidth = useMemo(
    () => clampWorkspaceWidth(Math.min(workspaceWidth, maxWorkspaceByViewport)),
    [workspaceWidth, maxWorkspaceByViewport],
  );
  // v4.27 首次打开文件时自动加宽右栏到舒适阅读宽度（Codex 式：点文件即铺开）。
  // 仅当当前宽度低于阈值时抬升，不覆盖用户已拖宽的偏好，且不越过视口上限；
  // 同步写全局键（与拖拽松手语义一致：最后一次宽度胜出、跨会话跟随）。
  const handleAutoWidenWorkspace = useCallback(() => {
    const target = Math.min(560, maxWorkspaceByViewport);
    if (workspaceWidthRef.current >= target) return;
    workspaceWidthRef.current = target;
    setWorkspaceWidth(target);
    saveWorkspaceWidth(target);
  }, [maxWorkspaceByViewport]);
  // 会话记录持久化：激活 tab / 启用集变化时随存宽度快照（学 better-sidebar 每次持久化同步全局宽度）
  useEffect(() => {
    savePersistedRightPanelState(
      { v: 1, tab: rightTab ?? "files", enabled: null, width: workspaceWidthRef.current },
      currentSessionKey,
    );
  }, [rightTab, currentSessionKey]);
  // v4.23 工作台宽度拖拽：右栏左缘手柄（指针拖拽形状同 preview-resizer）。
  // v4.27 上限放开：280–1600 钳制之上再按视口收敛（视口 − 侧栏 − 400 对话区），
  // 面板可拉到很宽但聊天区始终保留（Codex 式右侧面板体验）。
  // 宽度是布局偏好而非会话内容：拖拽中实时跟手，松手写全局键（最后一次拖拽胜出，
  // 跨会话即时跟随——蒸馏 dsh-better-sidebar 全局宽度键语义）。
  const startWorkspaceResize = useCallback((e: ReactPointerEvent) => {
    e.preventDefault();
    setWorkspaceResizing(true);
    const onMove = (me: PointerEvent) => {
      const maxW = Math.max(WORKSPACE_MIN_WIDTH, window.innerWidth - effectiveSidebarWidth - CHAT_MIN_WIDTH);
      const next = clampWorkspaceWidth(Math.min(maxW, window.innerWidth - me.clientX));
      workspaceWidthRef.current = next;
      setWorkspaceWidth(next);
    };
    const onDone = () => {
      saveWorkspaceWidth(workspaceWidthRef.current);
      setWorkspaceResizing(false);
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onDone);
      window.removeEventListener("pointercancel", onDone);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onDone);
    window.addEventListener("pointercancel", onDone);
  }, [effectiveSidebarWidth]);

  return {
    workspaceResizing,
    effectiveWorkspaceWidth,
    handleAutoWidenWorkspace,
    startWorkspaceResize,
  };
}