// usePreviewPanel — 主区大预览 + 右侧工作台的交互编排（Codex 式布局）。
// 从 App.tsx 按原执行顺序迁出：预览队列状态（P1-1 全局 store 驱动）、专注模式、
// pane 打开/文件 tab 辅助、预览拖拽/最大化、工作台面板开关。states/effects/
// callbacks 的声明顺序与 deps 数组全部原样保留。
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { PointerEvent as ReactPointerEvent } from "react";
import {
  PREVIEW_MAX_WIDTH, PREVIEW_MIN_WIDTH, clampPreviewWidth,
  loadPreviewWidth, savePreviewWidth,
  loadPreviewMaximized, savePreviewMaximized,
} from "../lib/layoutPreferences";
import { setPaneFileOpenHandler } from "../lib/paneFileOpen";
import { usePaneTabsStore } from "../lib/paneTabs";
import { recordRecentFile } from "../lib/recentFiles";
import { SIDEBAR_REGISTRY } from "../lib/sidebarRegistry";
import { usePreviewStore } from "../lib/store";
import { readWorkbenchValue, writeWorkbenchValue } from "../lib/workbenchStorage";
import type { WorkspaceTabId } from "../lib/workspaceTabs";

interface UsePreviewPanelParams {
  effectiveSidebarWidth: number;
  setRightTab: (tab: WorkspaceTabId | null) => void;
  handleWorkspacePreviewModeChange: (active: boolean) => void;
}

export function usePreviewPanel({ effectiveSidebarWidth, setRightTab, handleWorkspacePreviewModeChange }: UsePreviewPanelParams) {
  const [workspacePanelOpen, setWorkspacePanel] = useState(false);
  const [previewWidth, setPreviewWidth] = useState(loadPreviewWidth);
  // v4.30 预览两档占幅（VS Code Toggle Maximized Panel 式）：最大化 = 占满
  // 可用宽度（视口 − 侧栏 − 聊天最小 360，与拖拽上限同源）；还原回到进入
  // 最大化前的半幅宽度（previewHalfWidthRef 记忆）。拖拽分割条自动退出最大化。
  // v4.32：最大化状态持久化（gaea.previewMaximized），半幅宽度本就落盘，
  // 恢复会话后还原仍回到上次的半幅宽度。
  const [previewMaximized, setPreviewMaximized] = useState(loadPreviewMaximized);
  const previewHalfWidthRef = useRef(previewWidth);
  const [previewResizing, setPreviewResizing] = useState(false);
  const [workspaceRefreshKey, setWorkspaceRefreshKey] = useState(0);
  // P1-1 多文件预览队列：previewFile 与队列全部由全局 store 驱动（单一数据源），
  // 局部不再持有一份副本；openFilePreview 入队、navPreview ←/→ 切换。
  const previewFile = usePreviewStore((s) => s.previewFile);
  const previewIndex = usePreviewStore((s) => s.previewIndex);
  const previewList = usePreviewStore((s) => s.previewList);
  const closeFilePreview = usePreviewStore((s) => s.closeFilePreview);
  const navTo = usePreviewStore((s) => s.navTo);
  const closePreviewAt = usePreviewStore((s) => s.closePreviewAt);
  // U4 写后预览实时跟随：主区大预览与 pane 文件 tab 共用 reloadTicks 刷新总线
  const previewReloadTicks = usePaneTabsStore((s) => s.reloadTicks);

  // ── 专注模式（Kun 精华）：一键收起侧栏与右侧面板，只留对话和输入区 ──
  const [focusMode, setFocusMode] = useState(() => {
    return readWorkbenchValue("gaea.focusMode") === "1";
  });
  const applyFocus = useCallback((active: boolean) => {
    handleWorkspacePreviewModeChange(active);
    if (active) {
      setWorkspacePanel(false);
      closeFilePreview();
    }
  }, [handleWorkspacePreviewModeChange, closeFilePreview]);
  const toggleFocus = useCallback(() => {
    const next = !focusMode;
    setFocusMode(next);
    writeWorkbenchValue("gaea.focusMode", next ? "1" : "0");
    applyFocus(next);
  }, [focusMode, applyFocus]);
  useEffect(() => {
    if (focusMode) {
      handleWorkspacePreviewModeChange(true);
      setWorkspacePanel(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- 仅启动时收敛一次
  }, []);

  // 点文件 → 收起右侧树，在主区域展开可拖宽的预览（Codex 式）
  // P0-3：预览过的文件同步进「最近文件」快捷区（lib/recentFiles 单源）
  // P1-1：经全局 store 入队，支持 ←/→ 多文件切换
  const openFilePreview = useCallback((rel: string) => {
    recordRecentFile(rel);
    setRightTab("files");
    setWorkspacePanel(false);
    usePreviewStore.getState().openFilePreview(rel);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- setRightTab 为 React dispatch（稳定引用），原 deps [] 语义保留
  }, []);

  // 对话内「交付文件」卡片与正文文件链接走 usePreviewStore（弹窗通道）。
  // 本页有嵌入式预览容器，把这类请求重定向为嵌入预览（弹窗仅保留给
  // 记忆中枢等没有嵌入容器的页面）。P1-1：重定向时保留队列（不清空），
  // 直接入队即可由预览容器渲染，无需再转发局部状态。
  useEffect(() => {
    return usePreviewStore.subscribe((s, prev) => {
      if (s.previewFile && s.previewFile !== prev.previewFile) {
        setRightTab("files");
        setWorkspacePanel(false);
      }
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps -- setRightTab 为 React dispatch（稳定引用），原 deps [] 语义保留
  }, []);

  // ── pane 视图/文件打开辅助（对标 better-sidebar：卡片/事件只开 tab）──
  const openPaneView = useCallback((viewId: WorkspaceTabId) => {
    closeFilePreview();
    setWorkspacePanel(true);
    const reg = SIDEBAR_REGISTRY.find((r) => r.id === viewId);
    usePaneTabsStore.getState().openView(viewId, reg?.label ?? viewId);
  }, [closeFilePreview]);
  const openPaneFile = useCallback((rel: string) => {
    if (!rel) return;
    closeFilePreview();
    setWorkspacePanel(true);
    const name = rel.split(/[\\/]/).pop() || rel;
    usePaneTabsStore.getState().openFile(rel, name);
  }, [closeFilePreview]);

  // 正文交付卡 / 行内附件 / 工具输出文件引用 → 统一开 pane 文件 tab
  // （组件深处无法下钻回调，经模块级注入；未注册页面回落大预览）。
  useEffect(() => {
    setPaneFileOpenHandler(openPaneFile);
    return () => setPaneFileOpenHandler(null);
  }, [openPaneFile]);

  // 预览头部“文件”按钮 → 回到资源管理器视图 tab
  const backToFiles = useCallback(() => {
    closeFilePreview();
    openPaneView("files");
  }, [closeFilePreview, openPaneView]);

  // 面板开关：预览打开时先收起预览再展开树
  const toggleWorkspacePanel = useCallback(() => {
    if (previewFile !== null) {
      closeFilePreview();
      setWorkspacePanel(true);
      return;
    }
    setWorkspacePanel((o) => !o);
  }, [previewFile, closeFilePreview]);

  // 预览最大化可用宽度：与拖拽上限同源（视口 − 侧栏 − 聊天最小 360）。
  const previewMaxWidth = useMemo(
    () => Math.min(PREVIEW_MAX_WIDTH, Math.max(PREVIEW_MIN_WIDTH, window.innerWidth - effectiveSidebarWidth - 360)),
    [effectiveSidebarWidth],
  );
  // 半幅 ↔ 最大化 切换：进入最大化时记忆当前半幅宽度；还原时写回并持久化。
  const togglePreviewMaximize = useCallback(() => {
    if (previewMaximized) {
      setPreviewMaximized(false);
      savePreviewMaximized(false);
      setPreviewWidth(previewHalfWidthRef.current);
      savePreviewWidth(previewHalfWidthRef.current);
    } else {
      previewHalfWidthRef.current = previewWidth;
      setPreviewMaximized(true);
      savePreviewMaximized(true);
    }
  }, [previewMaximized, previewWidth]);

  // 拖拽分割条调整预览宽度
  const startPreviewResize = useCallback((e: ReactPointerEvent) => {
    e.preventDefault();
    // 用户手动拖拽 = 放弃最大化，回到半幅拖拽模式
    setPreviewMaximized(false);
    savePreviewMaximized(false);
    setPreviewResizing(true);
    let next = previewWidth;
    // 预览最小 320px；最大不超过窗口减侧栏后再留 360px 给聊天区
    const minW = PREVIEW_MIN_WIDTH;
    const maxW = Math.min(PREVIEW_MAX_WIDTH, window.innerWidth - effectiveSidebarWidth - 360);
    const onMove = (me: PointerEvent) => {
      next = clampPreviewWidth(Math.max(minW, Math.min(maxW, window.innerWidth - me.clientX)));
      setPreviewWidth(next);
    };
    const onDone = () => {
      setPreviewWidth(next);
      savePreviewWidth(next);
      setPreviewResizing(false);
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
  }, [previewWidth, effectiveSidebarWidth]);

  return {
    workspacePanelOpen, setWorkspacePanel,
    workspaceRefreshKey, setWorkspaceRefreshKey,
    previewFile, previewList, previewIndex, closeFilePreview, navTo, closePreviewAt,
    previewReloadTicks,
    focusMode, toggleFocus,
    openFilePreview, openPaneView, openPaneFile, backToFiles,
    toggleWorkspacePanel,
    previewWidth, previewMaximized, previewResizing, previewMaxWidth,
    togglePreviewMaximize, startPreviewResize,
  };
}