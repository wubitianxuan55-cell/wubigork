import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { CSSProperties } from "react";
import { Layout } from "antd";
import {
  BookOpen, Check, SquarePen, Brain, ChevronDown, FolderGit2, FileText,
  Gauge, PanelRightOpen, PanelRightClose, MessageSquare, Trash2, X, Aim, List, Square,
} from "./icons";
import { Sidebar } from "./components/Sidebar";
import { useT } from "./lib/i18n";
import { sessionTitle, sessionTime } from "./lib/session";
import { useController, usePreviewStore } from "./lib/store";
import { app } from "./lib/bridge";
import { Transcript } from "./components/Transcript";
import { JumpBar } from "./components/JumpBar";
import { useToast } from "./components/Toast";
import { Composer } from "./components/Composer";
import { TodoCard } from "./components/TodoCard";
import { ApprovalModal } from "./components/ApprovalModal";
import { AskCard } from "./components/AskCard";
import { ToolbarButton } from "./components/ToolbarButton";
import { ExportMenu, type ExportFormat } from "./components/ExportMenu";
import { ModelSwitcher } from "./components/ModelSwitcher";
const MemoryPanel = lazy(() => import("./components/MemoryPanel").then(m => ({ default: m.MemoryPanel })));
const HistoryPanel = lazy(() => import("./components/HistoryPanel").then(m => ({ default: m.HistoryPanel })));
const CapabilitiesPanel = lazy(() => import("./components/CapabilitiesPanel").then(m => ({ default: m.CapabilitiesPanel })));
const KnowledgePanel = lazy(() => import("./components/KnowledgePanel").then(m => ({ default: m.KnowledgePanel })));
import { WorkspaceTabs } from "./components/WorkspaceTabs";
import { ChatTabs, type ChatTabId } from "./components/ChatTabs";
import { ContextView } from "./components/ContextView";
import { TrajectoryView } from "./components/TrajectoryView";
import { FilePreview } from "./components/FilePreview";
import { PreviewNavBar } from "./components/PreviewNavBar";
import type { SessionDeliverable } from "./components/DeliverablesPanel";
import { CommandPalette, type PaletteItem } from "./components/CommandPalette";
import { useStatsPersistence } from "./components/StatsPanel";
import { OverviewPanel } from "./components/OverviewPanel";
import { useRunningBadge } from "./hooks/useRunningBadge";
import { Skeleton } from "./components/Skeleton";
import { UpdateBanner } from "./components/UpdateBanner";
import { SelectionToComposer } from "./components/SelectionToComposer";
import { NewSessionToast, JobDoneNotifier, RunStatus } from "./components/AppStatus";

import { downloadMarkdown, exportAsMarkdown } from "./lib/export";
import type { MemorySuggestion, MemorySuggestionsView, SessionMeta, SkillSuggestion, SubagentRunView } from "./lib/types";
import { useTodoExtractor } from "./hooks/useTodoExtractor";
import { useModeManager } from "./hooks/useModeManager";
import { useSessionManager } from "./hooks/useSessionManager";
import { useBridgeWatch } from "./hooks/useBridgeWatch";
import { useDrawers } from "./hooks/useDrawers";
import { useToolStats } from "./hooks/useToolStats";
import { useSidebar } from "./hooks/useSidebar";
import { readWorkbenchValue, writeWorkbenchValue } from "./lib/workbenchStorage";

import {
  SIDEBAR_DEFAULT_WIDTH, SIDEBAR_MIN_WIDTH, SIDEBAR_MAX_WIDTH,
} from "./hooks/useLayoutSizes";
import {
  PREVIEW_MAX_WIDTH, PREVIEW_MIN_WIDTH, clampPreviewWidth,
  loadPreviewWidth, savePreviewWidth,
  loadPreviewMaximized, savePreviewMaximized,
} from "./lib/layoutPreferences";
import { shouldAutoOpenDeliverables } from "./lib/deliverablePrefs";
import CompactContext from "./hooks/useCompact";
import { DELIVERABLE_EXT_RE, deliverableMentions } from "./lib/fileLinks";
import { recordRecentFile } from "./lib/recentFiles";
import { useUpdatedFilesStore } from "./lib/store";
import { buildSessionChanges, extractDeliverablePaths, WRITE_TOOL_NAMES, type SessionChange } from "./lib/changes";
import { openEditorTab } from "./lib/editorTabs";
import { parseSidebarOpenResult } from "./lib/sidebarOpen";
import { setEventSyncFetcher } from "./lib/eventSync";
import { shouldAutoOpenBrowser } from "./lib/browserPrefs";
import { setTaskCardActivityProvider } from "./lib/taskActivity";
import { classifyComposerCommand } from "./lib/command";
import { rankPaletteItems } from "./lib/paletteRank";
import {
  WORKSPACE_MIN_WIDTH, clampWorkspaceWidth, firstEnabledTab, loadEnabledTabs, loadPersistedRightTab,
  loadPersistedRightPanelState, loadWorkspaceWidth, resolveEnabledTabs, saveEnabledTabs,
  savePersistedRightPanelState, saveWorkspaceWidth,
  type WorkspaceEnabledMap, type WorkspaceTabId,
} from "./lib/workspaceTabs";
import { SIDEBAR_REGISTRY, getWorkspaceRegistration, type WorkspacePanelContext } from "./lib/sidebarRegistry";
import { loadTemplates, FALLBACK_TEMPLATES } from "./components/Welcome";
import type { TaskTemplate } from "./lib/types";

/** 右栏拖宽时保留的对话区最小宽度（Codex 式：面板可拉很宽，但聊天不能消失）。 */
const CHAT_MIN_WIDTH = 400;

export default function App() {
  const toast = useToast();
  const [chatTab, setChatTab] = useState<ChatTabId>(() => {
    try {
      const saved = readWorkbenchValue("gaea.chatTab");
      return saved === "trajectory" || saved === "context" || saved === "overview" ? saved : "chat";
    } catch { return "chat"; }
  });
  useEffect(() => {
    writeWorkbenchValue("gaea.chatTab", chatTab);
  }, [chatTab]);
  const {
    state,
    send,
    steer,
    cancel,
    approve,
    answerQuestion,
    setPermLevel: ctrlSetPermLevel,
    newSession,
    listSessions,
    listProjectSessions,
    resumeSession,
    archiveSession,
    unarchiveSession,
    pinSession,
    deleteSession,
    renameSession,
    refreshMeta,
    pickWorkspace,
    switchWorkspace,
    rewind,
    setModel,
    fetchMemory,
    remember,
    forget,
    saveDoc,
    updateFact,
    changeFactType,
    clearFactBase,
    promoteFactBase,
    fetchSessionStats,
  } = useController();
  const t = useT();
  const { permLevel, setPermLevel, thinkLevel, handleThinkLevelChange, switchingModel, switchModel } = useModeManager(ctrlSetPermLevel, setModel);
  const { memView, setMemView, histView, setHistView, capsOpen, setCapsOpen, knowledgeOpen, setKnowledgeOpen, closeTopmost } = useDrawers();
  const { sidebarSessions, sidebarQuery, setSidebarQuery, newSessionDone, refreshSessions, startNewSession, handleResumeSession, handleDeleteSession, handleRenameSession, projectGroups } = useSessionManager(newSession, listSessions, listProjectSessions, resumeSession, deleteSession, renameSession, (msg) => toast.show(msg, "warn"));
  const newSessionAndReset = useCallback(async () => { setStatsReset(n => n + 1); await startNewSession(); }, [startNewSession]);
  const [statsReset, setStatsReset] = useState(0);
  // 当前会话标识：直接使用 Go 后端生成的 .jsonl 文件路径作为 key。
  // 每个会话文件对应唯一的 localStorage key：新会话自然空数据开始，
  // 恢复/重启同一会话则统计数据持续累加，会话之间互不干扰。
  // 定义在右侧面板状态之前：rightTab 会话级持久化（C3）也依赖它。
  const currentSessionPath = useMemo(
    () => sidebarSessions.find(s => s.current)?.path,
    [sidebarSessions],
  );
  const currentSessionKey = useMemo(() => {
    const cwdNow = state.meta?.cwd;
    return currentSessionPath
      ? currentSessionPath.replace(/[\\/:*?"<>|]/g, "_")
      : cwdNow ? `unsaved_${cwdNow.replace(/[\\/:*?"<>|]/g, "_")}` : "unsaved";
  }, [currentSessionPath, state.meta?.cwd]);
  // 会话隔离（蒸馏 dsh-better-sidebar）：右侧面板子 Tab 按会话记忆（C3）——
  // 切会话/新建/恢复时恢复该会话上次选中的子面板；显式切换（如点文件回
  // 「文件」面板）照常覆盖当前会话记忆。无会话路径（未保存草稿）回退全局 key。
  // v4.23 会话记录扩为 { tab, enabled, width }（v1 裸 id 旧值宽容兼容）：启用集
  // /宽度的权威在各自全局键（布局偏好跨会话跟随），会话记录仅随存快照。
  const [initialPanelState] = useState(() => loadPersistedRightPanelState(currentSessionKey));
  const [rightTab, setRightTab] = useState<WorkspaceTabId>(initialPanelState.tab);
  // currentSessionKey 变化（会话切换）时：从新会话 key 恢复面板 Tab；
  // 切换本身不覆盖新会话已存的记忆（仅当新会话无记录时回退默认）。
  const prevSessionKey = useRef(currentSessionKey);
  useEffect(() => {
    if (prevSessionKey.current === currentSessionKey) return;
    prevSessionKey.current = currentSessionKey;
    setRightTab(loadPersistedRightTab(currentSessionKey));
  }, [currentSessionKey]);
  // ── v4.23 声明式设置（蒸馏 better-sidebar「侧边卡片」每 tab 独立开关）──
  // 启用覆盖集走全局键：最后一次切换胜出、跨会话即时跟随（对齐宽度键语义）。
  const [enabledOverrides, setEnabledOverrides] = useState<WorkspaceEnabledMap>(() => loadEnabledTabs());
  const enabledRecord = useMemo(() => resolveEnabledTabs(enabledOverrides), [enabledOverrides]);
  const enabledSet = useMemo(
    () => new Set((Object.keys(enabledRecord) as WorkspaceTabId[]).filter((id) => enabledRecord[id])),
    [enabledRecord],
  );
  useEffect(() => { saveEnabledTabs(enabledOverrides); }, [enabledOverrides]);
  const toggleTabEnabled = useCallback((id: WorkspaceTabId, next: boolean) => {
    setEnabledOverrides((prev) => ({ ...prev, [id]: next }));
  }, []);
  // 激活 tab 被停用时收敛到第一个启用面板（学 sanitizeState 失效激活指针修正）
  useEffect(() => {
    if (!enabledRecord[rightTab]) setRightTab(firstEnabledTab(enabledRecord));
  }, [enabledRecord, rightTab]);
  // 渲染用激活 tab：收敛 effect 生效前的一帧也稳定（不闪现停用面板）
  const activeWorkspaceTab = enabledRecord[rightTab] ? rightTab : firstEnabledTab(enabledRecord);
  const {
    sidebarCollapsed, sidebarWidth, sidebarResizing, effectiveSidebarWidth,
    toggleSidebar, setExpandedSidebarWidth, startSidebarResize,
    resizeSidebarWithKeyboard, handleWorkspacePreviewModeChange,
  } = useSidebar();
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
      { v: 1, tab: rightTab, enabled: enabledOverrides, width: workspaceWidthRef.current },
      currentSessionKey,
    );
  }, [rightTab, enabledOverrides, currentSessionKey]);
  // C6 运行域活动角标：活跃任务数（queued/running）；任务面板激活时视为已读
  // 不显示。v4.53 分工并入任务：运行计数角标挂在「任务」单键上，任务面板
  // （含分工段）激活即视为已读。
  const runningTasks = useRunningBadge();
  const runningDomainActive = rightTab === "tasks";
  const runningBadge = runningDomainActive ? undefined : { tasks: runningTasks };
  const [compactMode, setCompactMode] = useState(() => readWorkbenchValue("gaea.compactMode") === "1");
  const [scrollToTurn, setScrollToTurn] = useState<((turn: number) => void) | null>(null);
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState(false);

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
  }, []);

  // 预览头部“文件”按钮 → 回到文件树
  const backToFiles = useCallback(() => {
    closeFilePreview();
    setRightTab("files");
    setWorkspacePanel(true);
  }, [closeFilePreview]);

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
  const startPreviewResize = useCallback((e: React.PointerEvent) => {
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

  // v4.23 工作台宽度拖拽：右栏左缘手柄（指针拖拽形状同 preview-resizer）。
  // v4.27 上限放开：280–1600 钳制之上再按视口收敛（视口 − 侧栏 − 400 对话区），
  // 面板可拉到很宽但聊天区始终保留（Codex 式右侧面板体验）。
  // 宽度是布局偏好而非会话内容：拖拽中实时跟手，松手写全局键（最后一次拖拽胜出，
  // 跨会话即时跟随——蒸馏 dsh-better-sidebar 全局宽度键语义）。
  const startWorkspaceResize = useCallback((e: React.PointerEvent) => {
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

  // 统一交付出口：会话成果一键导出（docx/pptx/xlsx/md/pdf 同管线；
  // pdf 经 docx 中转 + LibreOffice 转换）。
  const exportConversation = useCallback(async (format: "docx" | "pptx" | "xlsx" | "md" | "pdf") => {
    const md = exportAsMarkdown(state.items);
    if (!md.trim()) return;
    try {
      const r = await app.ExportDeliverable({
        markdown: md,
        format,
        title: "gaea 会话交付",
        cover: format === "docx",
        toc: format === "docx",
      });
      toast.show(t("toast.exported", { name: r.name }), "info");
      void app.RevealWorkspacePath(r.path).catch(() => {});
    } catch (e) {
      toast.show(e instanceof Error ? e.message : String(e), "warn");
    }
  }, [state.items, toast, t]);
  const { onReconnect } = useBridgeWatch();
  useEffect(() => {
    onReconnect(() => { refreshMeta(); });
  }, [onReconnect, refreshMeta]);

  const { todoItem, todos, showTodos, setDismissedTodo } = useTodoExtractor(state.items);

  // Memory drawer: opening fetches a fresh snapshot; writes re-fetch so the
  // panel reflects what landed on disk.
  const openMemory = useCallback(async () => {
    setMemView(await fetchMemory());
  }, [fetchMemory, setMemView]);

  const closeMemory = useCallback(() => setMemView(null), [setMemView]);

  const openKnowledge = useCallback(() => setKnowledgeOpen(true), [setKnowledgeOpen]);
  const closeKnowledge = useCallback(() => setKnowledgeOpen(false), [setKnowledgeOpen]);

  // handleSend intercepts the slash commands that need a desktop-native action
  // before they reach the backend: "/model <ref>" rebuilds on that model,
  // "/memory" opens the memory drawer, and "/context" switches to the context
  // dashboard tab. Everything else — skills (/init, …), custom commands, bare
  // /model and the other read-only management verbs (/skill, /hooks, /mcp) —
  // goes straight to Submit, which the controller resolves (a turn, or a
  // listing Notice).
  const cwd = state.meta?.cwd;
  const cwdName = cwd ? cwd.split(/[/\\]/).filter(Boolean).pop() || cwd : "";

  const handleSend = useCallback(
    (displayText: string, submitText = displayText) => {
      const command = classifyComposerCommand(displayText);
      if (command.type === "model") {
        void switchModel(command.ref);
        return;
      }
      if (command.type === "memory") {
        void openMemory();
        return;
      }
      if (command.type === "context") {
        setChatTab("context");
        return;
      }
      send(displayText.trim(), submitText.trim());
    },
    [switchModel, openMemory, send, setChatTab],
  );

  // History drawer: opening fetches the saved-session list; picking one resumes it
  // (the transcript swaps in; the model/folder are unchanged).
  const openHistory = useCallback(async () => {
    setHistView(await refreshSessions());
  }, [refreshSessions, setHistView]);
  const closeHistory = useCallback(() => setHistView(null), [setHistView]);
  const onResumeSession = useCallback(
    async (path: string) => { setHistView(null); await handleResumeSession(path); },
    [handleResumeSession, setHistView],
  );
  const onDeleteSession = useCallback(
    async (path: string) => { await handleDeleteSession(path); setHistView(await refreshSessions()); },
    [handleDeleteSession, refreshSessions, setHistView],
  );
  const onRenameSession = useCallback(
    async (path: string, title: string) => { await handleRenameSession(path, title); setHistView(await refreshSessions()); },
    [handleRenameSession, refreshSessions, setHistView],
  );

  // 删除当前会话：无需打开侧边栏，顶栏直接操作；删除后自动开启新会话
  const confirmDeleteCurrent = useCallback(async () => {
    setDeleteConfirm(false);
    try {
      const all = await refreshSessions();
      const cur = all.find((s) => s.current);
      if (cur) await deleteSession(cur.path);
    } catch {
      // 删除失败不阻塞新建会话
    }
    await newSessionAndReset();
  }, [refreshSessions, deleteSession, newSessionAndReset]);

  // Workspace: open the folder chooser and switch projects. The hook resets the
  // transcript and refreshes meta on a pick; refresh the sidebar sessions too so
  // the recent list belongs to the newly selected workspace. A cancel is a no-op.
  const switchFolder = useCallback(async (path?: string) => {
    const picked = path === undefined ? await pickWorkspace() : await switchWorkspace(path);
    if (picked) {
      closeFilePreview();
      setWorkspacePanel(false);
      await refreshSessions();
    }
    return picked;
  }, [pickWorkspace, switchWorkspace, refreshSessions, closeFilePreview]);

  // 从侧边栏点其他项目的会话：先切换到该项目工作区，再恢复该会话。
  const currentProjectPath = projectGroups.find((g) => g.current)?.path;
  const resumeSessionInProject = useCallback(
    async (path: string, projectPath: string) => {
      if (currentProjectPath && projectPath && currentProjectPath !== projectPath) {
        await switchFolder(projectPath);
      }
      await handleResumeSession(path);
    },
    [currentProjectPath, switchFolder, handleResumeSession],
  );

  // 欢迎页「最近会话」：从项目分组派生跨项目最近会话（去重、按最近排序），
  // 不再沿用旧扁平列表（旧列表仅当前工作区且被分页截断）。
  const recentSessions = useMemo(() => {
    const out: SessionMeta[] = [];
    const seen = new Set<string>();
    for (const g of projectGroups) {
      for (const s of g.sessions) {
        if (s.current || seen.has(s.path)) continue;
        seen.add(s.path);
        out.push(s);
      }
    }
    out.sort((a, b) => b.modTime - a.modTime);
    return out.slice(0, 6);
  }, [projectGroups]);

  const resumeRecentSession = useCallback(
    async (path: string) => {
      const group = projectGroups.find((g) => g.sessions.some((s) => s.path === path));
      await resumeSessionInProject(path, group?.path ?? currentProjectPath ?? "");
    },
    [projectGroups, currentProjectPath, resumeSessionInProject],
  );

  // 会话管理（Kun/Codex 优点蒸馏）：置顶、归档、恢复
  const onArchiveSession = useCallback(async (path: string) => {
    try {
      await archiveSession(path);
      await refreshSessions();
    } catch (e) {
      toast.show(t("toast.archiveFailed", { msg: e instanceof Error ? e.message : String(e) }), "warn");
    }
  }, [archiveSession, refreshSessions, toast, t]);

  const onPinSession = useCallback(async (path: string, pinned: boolean) => {
    try {
      await pinSession(path, pinned);
      await refreshSessions();
    } catch (e) {
      toast.show(`置顶操作失败：${e instanceof Error ? e.message : String(e)}`, "warn");
    }
  }, [pinSession, refreshSessions, toast]);

  const onRestoreSession = useCallback(
    async (path: string, projectPath: string) => {
      try {
        const restored = await unarchiveSession(path);
        if (restored) await resumeSessionInProject(restored, projectPath);
      } catch (e) {
        toast.show(`恢复失败：${e instanceof Error ? e.message : String(e)}`, "warn");
      }
    },
    [unarchiveSession, resumeSessionInProject, toast],
  );

  const onRemember = useCallback(
    async (scope: string, note: string) => {
      await remember(scope, note);
      setMemView(await fetchMemory());
    },
    [remember, fetchMemory, setMemView],
  );

  const onForget = useCallback(
    async (name: string) => {
      await forget(name);
      setMemView(await fetchMemory());
    },
    [forget, fetchMemory, setMemView],
  );

  const onSaveDoc = useCallback(
    async (path: string, body: string) => {
      await saveDoc(path, body);
      setMemView(await fetchMemory());
    },
    [saveDoc, fetchMemory, setMemView],
  );

  const onSaveFact = useCallback(
    async (name: string, body: string) => {
      await updateFact(name, body);
      setMemView(await fetchMemory());
    },
    [updateFact, fetchMemory, setMemView],
  );

  const onAcceptMemorySuggestion = useCallback(
    async (candidate: MemorySuggestion) => {
      await app.AcceptMemorySuggestion(candidate);
      setMemView(await fetchMemory());
    },
    [fetchMemory, setMemView],
  );

  const onAcceptSkillSuggestion = useCallback(
    async (candidate: SkillSuggestion) => {
      await app.AcceptSkillSuggestion(candidate);
    },
    [],
  );

  // 蒸馏合并（做梦 2.0）：批准后归档较旧条，刷新记忆视图。
  const onAcceptMergeSuggestion = useCallback(
    async (keep: string, archive: string) => {
      await app.AcceptMergeSuggestion(keep, archive);
      setMemView(await fetchMemory());
    },
    [fetchMemory, setMemView],
  );

  const onRefreshSuggestions = useCallback(async (): Promise<MemorySuggestionsView | null> => {
    try {
      return await app.MemorySuggestions();
    } catch {
      return null;
    }
  }, []);

  useEffect(() => { void refreshSessions(); }, [cwd, refreshSessions]);

  // 全局快捷键
  useEffect(() => {
    const onKey = (e: Event) => {
      const ke = e as globalThis.KeyboardEvent;
      const mod = ke.ctrlKey || ke.metaKey, t = ke.target as HTMLElement;
      const inInput = t.tagName === "INPUT" || t.tagName === "TEXTAREA" || t.isContentEditable;
      if (ke.key === "Escape" && !inInput && !state.running) {
        if (previewFile !== null) { ke.preventDefault(); closeFilePreview(); return; }
        if (closeTopmost()) { ke.preventDefault(); return; }
      }
      if (!mod) return;
      if (ke.key === "n" && !state.running) { ke.preventDefault(); void newSessionAndReset(); return; }
      if (ke.key === "k") { ke.preventDefault(); setPaletteOpen(true); return; }
      if (ke.key === "H" && ke.shiftKey) { ke.preventDefault(); void openHistory(); return; }
      if (ke.key === "K" && ke.shiftKey) { ke.preventDefault(); void openKnowledge(); return; }
      if (ke.key === "b") { ke.preventDefault(); toggleSidebar(); return; }
      if (ke.key === "j") { ke.preventDefault(); toggleWorkspacePanel(); return; }
      if (ke.key === "F" && ke.shiftKey) { ke.preventDefault(); toggleFocus(); return; }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [state.running, closeTopmost, workspacePanelOpen, previewFile, toggleFocus, closeFilePreview, newSessionAndReset, openHistory, openKnowledge, toggleSidebar, toggleWorkspacePanel]);

  const { toolCounts, skillCounts } = useToolStats(state.items);
  // 会话产物：从会话消息文本中提取交付文件 + 写类工具落盘的显式登记
  // （保留首现顺序；同一文件多次出现计入 versions 次数——产物版本时间线
  // 数据源，对标 Hermes 版本步进器）。显式登记修复：Agent 未在正文提及
  // 路径时成果面板漏登记的启发式缺陷。
  const sessionDeliverables = useMemo<SessionDeliverable[]>(() => {
    const order = new Map<string, { sourceId: string; turn: number; versions: number }>();
    let turn = -1;
    const register = (p: string, sourceId: string) => {
      const rec = order.get(p);
      if (rec) {
        rec.versions++;
      } else {
        order.set(p, { sourceId, turn: Math.max(0, turn), versions: 1 });
      }
    };
    for (const it of state.items) {
      if (it.kind === "user") {
        turn++;
        continue;
      }
      // 写类工具落盘 = 显式登记：工具参数里的路径是真实写入，不依赖正文提及
      if (it.kind === "tool" && WRITE_TOOL_NAMES.has(it.name)) {
        for (const p of extractDeliverablePaths(it.args || "")) {
          if (DELIVERABLE_EXT_RE.test(p)) register(p, it.id);
        }
        continue;
      }
      if (it.kind !== "assistant" || !it.text) continue;
      for (const p of deliverableMentions(it.text)) {
        register(p, it.id);
      }
    }
    const out: SessionDeliverable[] = [];
    for (const [path, rec] of order) {
      out.push({ path, sourceId: rec.sourceId, turn: rec.turn, versions: rec.versions });
    }
    return out;
  }, [state.items]);

  // 文件变更（Kun 可观察性精华）：汇总本会话写/改过的文件及次数
  const sessionChanges = useMemo<SessionChange[]>(() => buildSessionChanges(state.items), [state.items]);

  // ── v4.30 产物自动置前/角标（Devin Auto-open 式）────────────────────────
  // 本会话内新出现的产物路径：diff sessionDeliverables（首现即新），产物 tab
  // 角标显示未读数、产物面板对应行显示「新」徽标+高亮；激活产物 tab（查看）
  // 即清零（与运行角标「激活即已读」语义一致）。
  // 会话切换：重置为「当前产物全集」基线（恢复会话不误标新），并清空角标。
  const [freshDeliverablePaths, setFreshDeliverablePaths] = useState<string[]>([]);
  const seenDeliverablePathsRef = useRef<Set<string>>(new Set());
  const baselinePendingRef = useRef(true);
  useEffect(() => {
    baselinePendingRef.current = true;
    seenDeliverablePathsRef.current = new Set(sessionDeliverables.map((d) => d.path));
    setFreshDeliverablePaths([]);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- 仅会话切换时重置基线，快照取当前值
  }, [currentSessionKey]);
  useEffect(() => {
    // 切换后的首次运行：把当前会话产物全量预填为基线（不算新），之后流式
    // 增长中出现的路径才标新。
    if (baselinePendingRef.current) {
      baselinePendingRef.current = false;
      seenDeliverablePathsRef.current = new Set(sessionDeliverables.map((d) => d.path));
      return;
    }
    const seen = seenDeliverablePathsRef.current;
    const added: string[] = [];
    for (const d of sessionDeliverables) {
      if (!seen.has(d.path)) {
        seen.add(d.path);
        added.push(d.path);
      }
    }
    if (added.length > 0) {
      setFreshDeliverablePaths((prev) => [...prev, ...added]);
      // v4.32 产物自动弹出（收 v4.30 欠账「自动弹 tab 暂不做可加偏好」）：
      // 偏好开（gaea.deliverableAutoOpen，默认关，开关在 DeliverablesPanel
      // 头部胶囊）且产物 tab 未停用 → 亮右栏切「产物」tab，语义对齐
      // browserAutoOpen（tab 停用时尊重停用态不强行弹出；激活即清零角标，
      // 自动弹出 = 已查看）。不动 FilePreview——产物 tab 与主区预览不冲突。
      if (shouldAutoOpenDeliverables() && rightTab !== "deliverables" && enabledRecord.deliverables) {
        setRightTab("deliverables");
        setWorkspacePanel(true);
      }
    }
  }, [sessionDeliverables, rightTab, enabledRecord]);
  // 激活产物 tab = 已查看 → 清零角标与「新」徽标
  useEffect(() => {
    if (rightTab === "deliverables") setFreshDeliverablePaths([]);
  }, [rightTab]);
  // v4.30 产物自动置前：未查看的新产物数角标（激活产物 tab 即清零，见上方 effect）
  const freshDeliverableCount = rightTab === "deliverables" ? 0 : freshDeliverablePaths.length;
  const tabBadges = runningBadge
    ? { ...runningBadge, ...(freshDeliverableCount > 0 ? { deliverables: freshDeliverableCount } : {}) }
    : freshDeliverableCount > 0 ? { deliverables: freshDeliverableCount } : undefined;

  // 编辑后自动回写刷新：docx/xlsx 预览内编辑成功 → 文件树自动刷新（替代手动刷新）
  const updatedAt = useUpdatedFilesStore((s) => s.updatedAt);
  useEffect(() => {
    if (Object.keys(updatedAt).length === 0) return;
    setWorkspaceRefreshKey((k) => k + 1);
  }, [updatedAt]);

  // 会话切换（启动装载/新建/恢复/切换工作区）后拉取会话级派生统计，
  // 回填「全会话成本」的历史部分（评审缺陷 11）。
  useEffect(() => {
    if (!currentSessionPath) return;
    void fetchSessionStats(currentSessionPath);
  }, [currentSessionPath, fetchSessionStats]);

  const statsPersistence = useStatsPersistence(currentSessionKey, statsReset, state.turnSteps, state.perTurnUsage);

  // ── v4.23 注册表渲染上下文：右栏面板公共依赖一次性组装 ──
  // （学 better-sidebar 框架/内容解耦：面板 props 与旧渲染分支逐一对应，行为不变）
  const refreshWorkspacePanel = useCallback(() => setWorkspaceRefreshKey((k) => k + 1), []);
  const locateDeliverableSource = useCallback((turn: number) => {
    setWorkspacePanel(false);
    scrollToTurn?.(turn);
  }, [scrollToTurn]);
  // v4.24 A1 新子代理自动展开：分工面板检测到新子代理出现（且用户偏好开启）
  // 时回调——亮出右栏并切到「任务」面板（v4.53 分工并入任务同屏展示；
  // 对标 better-sidebar 任务页自动展开侧栏）；面板被用户停用时尊重停用态
  // 不强行弹出。
  const handleSubagentStarted = useCallback(() => {
    if (!enabledRecord.tasks) return;
    closeFilePreview();
    setWorkspacePanel(true);
    setRightTab("tasks");
  }, [enabledRecord.tasks, closeFilePreview]);

  // v4.25 A3 reveal：产物面板「树中定位」→ 亮文件 tab，文件树展开父链+滚动+闪烁。
  // nonce 单调递增，同一文件重复定位也能再触发一次。
  const revealNonceRef = useRef(0);
  const [revealRequest, setRevealRequest] = useState<{ rel: string; nonce: number } | null>(null);
  const handleRevealInTree = useCallback((rel: string) => {
    closeFilePreview();
    setRightTab("files");
    setWorkspacePanel(true);
    revealNonceRef.current += 1;
    setRevealRequest({ rel, nonce: revealNonceRef.current });
  }, [closeFilePreview]);

  // v4.25 模型主动打开（对标 better-sidebar sidebar_open）：模型把关键产物/目录
  // 推到右栏文件工作台。按工具事件 id 去重；file → 编辑器 tab（lib/editorTabs
  // 外部 store 程序化入口），directory → 文件树树中定位（v4.28 起接 reveal：
  // 展开父链 + 滚动 + 目录行闪烁，FileTree 目录行已带 data-path 锚点）。
  const sidebarOpenSeenRef = useRef<Set<string>>(new Set());
  useEffect(() => {
    let requested = false;
    for (const it of state.items) {
      if (it.kind !== "tool" || it.name !== "sidebar_open" || !it.output || it.status === "running") continue;
      if (sidebarOpenSeenRef.current.has(it.id)) continue;
      const parsed = parseSidebarOpenResult(it.name, it.args, it.output);
      if (!parsed) continue;
      sidebarOpenSeenRef.current.add(it.id);
      if (parsed.kind === "file") openEditorTab(parsed.pathRel);
      else if (parsed.kind === "directory") handleRevealInTree(parsed.pathRel);
      requested = true;
    }
    if (requested) {
      closeFilePreview();
      setRightTab("files");
      setWorkspacePanel(true);
    }
  }, [state.items, closeFilePreview, handleRevealInTree]);

  // v4.28 A2 浏览器观察窗自动弹出：会话轨迹出现新 browser_* 工具时，偏好开
  // （gaea.browserAutoOpen）且「浏览器」tab 未停用 → 亮右栏切「浏览器」tab
  //（对标 handleSubagentStarted 语义：tab 被停用时尊重停用态不强行弹出）。
  const browserSeenRef = useRef<Set<string>>(new Set());
  useEffect(() => {
    let requested = false;
    for (const it of state.items) {
      if (it.kind !== "tool" || !it.name.startsWith("browser_")) continue;
      if (browserSeenRef.current.has(it.id)) continue;
      browserSeenRef.current.add(it.id);
      requested = true;
    }
    if (requested && shouldAutoOpenBrowser() && enabledRecord.browser) {
      closeFilePreview();
      setRightTab("browser");
      setWorkspacePanel(true);
    }
  }, [state.items, enabledRecord.browser, closeFilePreview]);

  // v4.26 对话流式重造接线：
  // ① 事件序号防线 fetcher——Wails 事件流吞件（seq 跳号）时经后端从磁盘日志
  //    补拉对话项全量快照（eventSync 冷却门控频；未挂载则整条防线旁路）。
  useEffect(() => {
    setEventSyncFetcher((afterSeq) => app.ResyncEvents(afterSeq));
    return () => setEventSyncFetcher(null);
  }, []);
  // ② 子代理 task 卡 live 预览——运行期间 5s 轮询 GaeaSubagentRuns 喂
  //    taskActivity 注入点；派发期 args 不带 ref（ref 只在 tool_result 出现），
  //    空 ref 回退「唯一 running 分工」（taskActivity 头注释契约）。
  const subRunsCacheRef = useRef<SubagentRunView[]>([]);
  useEffect(() => {
    setTaskCardActivityProvider((ref) => {
      const runs = subRunsCacheRef.current;
      const pick = (r: SubagentRunView) =>
        r ? { lastText: r.lastText, lastTool: r.lastTool, state: r.status } : undefined;
      if (ref) {
        const hit = runs.find((r) => r.ref === ref);
        return hit ? pick(hit) : undefined;
      }
      const runningRuns = runs.filter((r) => r.status === "running");
      return runningRuns.length === 1 ? pick(runningRuns[0]) : undefined;
    });
    return () => setTaskCardActivityProvider(null);
  }, []);
  useEffect(() => {
    if (!state.running || !currentSessionPath) {
      subRunsCacheRef.current = [];
      return;
    }
    let live = true;
    const pull = () => {
      void app.SubagentRuns(currentSessionPath)
        .then((v) => { if (live) subRunsCacheRef.current = v.runs ?? []; })
        .catch(() => {});
    };
    pull();
    const timer = setInterval(pull, 5000);
    return () => { live = false; clearInterval(timer); };
  }, [state.running, currentSessionPath]);
  const panelContext = useMemo<WorkspacePanelContext>(
    () => ({
      cwd: state.meta?.cwd,
      selectedFile: previewFile ?? undefined,
      refreshKey: workspaceRefreshKey,
      currentSessionPath: currentSessionPath ?? undefined,
      sessionDeliverables,
      sessionChanges,
      freshDeliverablePaths,
      onOpenFile: openFilePreview,
      onClosePanel: () => setWorkspacePanel(false),
      onRefreshPanel: refreshWorkspacePanel,
      onLocateSource: locateDeliverableSource,
      onSubagentStarted: handleSubagentStarted,
      onRevealInTree: handleRevealInTree,
      revealRequest,
      onAutoWidenPanel: handleAutoWidenWorkspace,
    }),
    [
      state.meta?.cwd, previewFile, workspaceRefreshKey, currentSessionPath,
      sessionDeliverables, sessionChanges, freshDeliverablePaths,
      openFilePreview, refreshWorkspacePanel, locateDeliverableSource,
      handleSubagentStarted, handleRevealInTree, revealRequest, handleAutoWidenWorkspace,
    ],
  );

  // 任务模板（命令面板「任务模板」组 + 欢迎页共用数据源；loadTemplates 模块级缓存）。
  const [templates, setTemplates] = useState<TaskTemplate[]>(FALLBACK_TEMPLATES);
  useEffect(() => {
    let live = true;
    loadTemplates().then((ts) => { if (live) setTemplates(ts); });
    return () => { live = false; };
  }, []);

  const paletteItems = useMemo<PaletteItem[]>(() => {
    const cmds: PaletteItem[] = [
      { id: "cmd-new", group: t("palette.group.commands") ?? "命令", title: t("topbar.newSession") ?? "新建会话", icon: <SquarePen size={15} />, compact: true, keywords: ["new", "新建"], run: () => void newSessionAndReset() },
      { id: "cmd-memory", group: t("palette.group.commands") ?? "命令", title: t("topbar.memory") ?? "记忆", icon: <Brain size={15} />, compact: true, keywords: ["memory", "记忆"], run: () => void openMemory() },
      { id: "cmd-history", group: t("palette.group.commands") ?? "命令", title: t("topbar.history") ?? "历史", icon: <MessageSquare size={15} />, compact: true, keywords: ["history", "历史"], run: () => void openHistory() },
      { id: "cmd-knowledge", group: t("palette.group.commands") ?? "命令", title: t("topbar.knowledge") ?? "知识库", icon: <BookOpen size={15} />, compact: true, keywords: ["knowledge", "知识库"], run: () => void openKnowledge() },
      { id: "cmd-overview", group: t("palette.group.commands") ?? "命令", title: "概览面板", icon: <Gauge size={15} />, compact: true, keywords: ["overview", "概览", "统计", "token", "成本", "用量"], run: () => setChatTab("overview") },
    ];
    // 右侧面板命令项：由 sidebarRegistry 注册表派生（v4.23 注册制）。声明式
    // 设置停用的面板不进命令面板（学 better-sidebar prefs 门控「+」菜单）；
    // tasks 不在命令面板（与原行为一致）
    for (const reg of SIDEBAR_REGISTRY) {
      if (reg.id === "tasks") continue;
      if (!enabledRecord[reg.id]) continue;
      const Icon = reg.icon;
      cmds.push({
        id: `cmd-${reg.id}`,
        group: t("palette.group.commands") ?? "命令",
        title: `${reg.label}面板`,
        icon: <Icon size={15} />,
        compact: true,
        keywords: reg.keywords,
        run: () => { closeFilePreview(); setWorkspacePanel(true); setRightTab(reg.id); },
      });
    }
    const sessionItems: PaletteItem[] = sidebarSessions.slice(0, 10).map((s) => ({
      id: `sess-${s.path}`,
      group: t("palette.group.sessions") ?? "会话",
      title: sessionTitle(s, t("history.emptySession") ?? "空会话"),
      hint: s.path,
      meta: sessionTime(s.modTime),
      badge: s.current ? "当前" : undefined,
      icon: <MessageSquare size={15} />,
      keywords: ["session", "会话"],
      run: () => { if (!s.current) void onResumeSession(s.path); },
    }));
    // 任务模板组：与欢迎页同源（FALLBACK_TEMPLATES/远端），Ctrl+K 可直接发起
    const templateItems: PaletteItem[] = templates.map((tm) => ({
      id: `tpl-${tm.name}`,
      group: t("palette.group.templates") ?? "任务模板",
      title: tm.title,
      hint: `/${tm.name}`,
      meta: tm.description,
      icon: <FileText size={15} />,
      keywords: ["template", "模板", tm.name, ...tm.title.split(/\s+/)],
      run: () => { closeFilePreview(); setWorkspacePanel(false); send(tm.prompt); },
    }));
    // v4.30 命令面板按当前视图重排（Linear 式）：当前激活的右栏面板 / 主区
    // tab 对应命令置顶，其余保持稳定原序（纯函数，见 lib/paletteRank）。
    return rankPaletteItems(
      [...cmds, ...templateItems, ...sessionItems],
      { chatTab, rightTab },
    );
  }, [t, sidebarSessions, openMemory, openHistory, openKnowledge, onResumeSession, setWorkspacePanel, closeFilePreview, setRightTab, enabledRecord, templates, send, newSessionAndReset, chatTab, rightTab]);

  const layoutStyle = useMemo(
    () =>
      ({
        "--sidebar-expanded-width": `${sidebarWidth}px`,
        // v4.30 预览两档：最大化时占满可用宽度，半幅用用户拖拽宽度
        "--preview-width": `${previewMaximized ? previewMaxWidth : previewWidth}px`,
        // v4.23 工作台宽度：钳制后经 CSS 变量下发（覆盖 styles.css 340px 基线）；
        // v4.27 用视口感知的有效宽度，宽面板在小窗口不挤出聊天区
        "--workspace-width": `${effectiveWorkspaceWidth}px`,
      }) as CSSProperties,
    [sidebarWidth, previewWidth, effectiveWorkspaceWidth, previewMaximized, previewMaxWidth],
  );

  return (
    <>
    <JobDoneNotifier jobs={state.jobs} />
    <Layout className="gaea-app-layout">
      <div
        className={[
          "layout",
          sidebarCollapsed ? "layout--sidebar-collapsed" : "",
          sidebarResizing ? "layout--resizing layout--sidebar-resizing" : "",
          previewResizing ? "layout--resizing layout--preview-resizing" : "",
          workspaceResizing ? "layout--resizing" : "",
          workspacePanelOpen ? "layout--workspace-open" : "",
          previewFile ? "layout--preview-open" : "",
        ]
          .filter(Boolean)
          .join(" ")}
        style={layoutStyle}
      >
        <Sidebar
          collapsed={sidebarCollapsed}
          toggleSidebar={toggleSidebar}
          running={state.running}
          jobs={state.jobs}
          factBase={state.factBase}
          onClearFactBase={() => void clearFactBase()}
          onPromoteFactBase={promoteFactBase}
          newSessionAndReset={newSessionAndReset}
          projectGroups={projectGroups}
          onResumeSessionInProject={resumeSessionInProject}
          onArchiveSession={onArchiveSession}
          onRestoreSession={onRestoreSession}
          onPinSession={onPinSession}
          searchQuery={sidebarQuery}
          onSearchChange={setSidebarQuery}
          onDeleteSession={onDeleteSession}
          onRenameSession={handleRenameSession}
          onOpenHistory={openHistory}
          onOpenMemory={openMemory}
          onOpenCaps={() => setCapsOpen(true)}
          onOpenKnowledge={openKnowledge}
          startResize={startSidebarResize}
          resizeWithKeyboard={resizeSidebarWithKeyboard}
          onDoubleClickResize={() => setExpandedSidebarWidth(SIDEBAR_DEFAULT_WIDTH)}
          sidebarWidth={sidebarWidth}
          SIDEBAR_MIN_WIDTH={SIDEBAR_MIN_WIDTH}
          SIDEBAR_MAX_WIDTH={SIDEBAR_MAX_WIDTH}
        />

        <section className="chat-pane v3-zone">
          <header className="flex flex-shrink-0 items-center gap-3 px-12 border-b border-border-soft select-none drag-region transition-all duration-200" style={{background: "var(--gaea-glass-bg, var(--md-sys-color-surface-container))"}}>
            <div className="flex items-center gap-2 min-w-0">
              <ModelSwitcher label={state.meta?.label ?? t("status.connecting")} onPick={switchModel} />
            </div>
            <div className="flex items-center gap-2 px-3">
              {cwd && (<button className="toolbar-btn no-drag" onClick={() => void switchFolder()} disabled={state.running}><FolderGit2 size={13} /><span>{cwdName}</span><ChevronDown size={11} /></button>)}
            </div>
            <div className="flex-1" />
            <div className="flex items-center gap-2">
              <ToolbarButton onClick={() => void toggleWorkspacePanel()} title={previewFile ? t("topbar.backToFiles") : workspacePanelOpen ? t("topbar.collapseFilePanel") : t("topbar.expandFilePanel")}>
                {workspacePanelOpen || previewFile ? <PanelRightClose size={13} /> : <PanelRightOpen size={13} />}
              </ToolbarButton>
              <ToolbarButton onClick={() => { const v = !compactMode; setCompactMode(v); writeWorkbenchValue("gaea.compactMode", v ? "1" : "0"); }} title={compactMode ? t("topbar.expandMode") : t("topbar.compactMode")}>{compactMode ? <List size={13} /> : <Square size={13} />}</ToolbarButton>
              <ToolbarButton onClick={toggleFocus} title={focusMode ? t("topbar.exitFocusMode") : t("topbar.focusMode")}>
                <Aim size={13} className={focusMode ? "text-accent" : ""} />
              </ToolbarButton>
              {/* v4.29 化繁为简：导出三出口（md/Word/PDF）收进单钮下拉，管线原样保留 */}
              <ExportMenu
                disabled={state.items.length === 0}
                onPick={(format: ExportFormat) => {
                  if (format === "md") downloadMarkdown(exportAsMarkdown(state.items));
                  else void exportConversation(format);
                }}
              />
              {deleteConfirm ? (
                <span className="flex items-center gap-1 rounded-md border border-err/30 bg-del-bg px-1.5 py-1">
                  <span className="text-[11px] text-err whitespace-nowrap">{t("topbar.deleteSessionAsk")}</span>
                  <button
                    className="inline-flex items-center justify-center w-5 h-5 border-0 rounded bg-transparent text-err cursor-pointer hover:bg-err/15"
                    onClick={() => void confirmDeleteCurrent()}
                    title={t("history.confirmDelete")}
                  >
                    <Check size={12} />
                  </button>
                  <button
                    className="inline-flex items-center justify-center w-5 h-5 border-0 rounded bg-transparent text-fg-faint cursor-pointer hover:bg-bg-soft hover:text-fg"
                    onClick={() => setDeleteConfirm(false)}
                    title={t("common.cancel")}
                  >
                    <X size={12} />
                  </button>
                </span>
              ) : (
                <ToolbarButton onClick={() => setDeleteConfirm(true)} disabled={state.running} title={t("topbar.deleteSession")}>
                  <Trash2 size={13} />
                </ToolbarButton>
              )}
            </div>
          </header>

          {state.meta?.startupErr && (
            <div className="shrink-0 px-4 py-2 text-[12.5px] bg-del-bg text-err border-b border-border-soft">{t("topbar.startupError", { msg: state.meta.startupErr })}</div>
          )}

          <UpdateBanner />
          <NewSessionToast done={newSessionDone} />
          <ChatTabs active={chatTab} onChange={setChatTab} />
          <main className="main">
            <CompactContext.Provider value={compactMode}>
            {(state.meta?.ready === false && !state.meta?.startupErr) || switchingModel ? (
              <Skeleton />
            ) : (
              <>
                {chatTab === "chat" && (
                  <>
                    <Transcript onPrompt={send} running={state.running} onRewind={rewind} onScrollToTurnReady={setScrollToTurn} cwd={state.meta?.cwd} cwdName={cwdName} sessions={recentSessions} onResumeSession={resumeRecentSession} meta={state.meta} />
                    {state.items.length > 1 && <JumpBar items={state.items} scrollToTurn={scrollToTurn ?? undefined} />}
                  </>
                )}
                {chatTab === "trajectory" && <TrajectoryView running={state.running} />}
                {chatTab === "context" && <ContextView running={state.running} sessionPath={currentSessionPath ?? undefined} />}
                {chatTab === "overview" && (
                  <OverviewPanel
                    data={statsPersistence.data}
                    clearData={statsPersistence.clearData}
                    sessionStats={state.sessionStats}
                    perTurnExecutorUsage={state.perTurnExecutorUsage}
                    perTurnSubUsage={state.perTurnSubUsage}
                    turnSteps={state.turnSteps}
                    subagentModel={state.meta?.subagentLabel}
                    toolCounts={toolCounts}
                    skillCounts={skillCounts}
                  />
                )}
              </>
            )}
            </CompactContext.Provider>
          </main>

          <footer className={`shrink-0 border-t border-border-soft bg-bg px-8 ${compactMode ? "pt-2 pb-0.5" : "pt-3 pb-1"}`}>
            <CompactContext.Provider value={compactMode}>
            {showTodos && (
              <TodoCard
                todos={todos}
                onDismiss={() => { if (todoItem) setDismissedTodo(todoItem.id); }}
              />
            )}
            <RunStatus
              running={state.running}
              turnStartAt={state.turnStartAt}
              turnTokens={state.turnTokens}
              used={state.context.used}
              window={state.context.window}
            />
            <div className="composer-glow">
            <Composer
              running={state.running}
              cwd={state.meta?.cwd}
              onSend={handleSend}
              onSteer={steer}
              onCancel={cancel}
              permLevel={permLevel}
              onSetPermLevel={setPermLevel}
              thinkLevel={thinkLevel}
              onSetThinkLevel={handleThinkLevelChange}
              onPickFolder={switchFolder}
              disabled={state.meta?.ready === false || state.approval != null}
            />
            </div>
            </CompactContext.Provider>
          </footer>
        </section>

        {/* 主区域预览：点文件后右侧树收起，预览在聊天区右侧展开（宽度可拖） */}
        {previewFile && (
          <>
            <div
              className={`preview-resizer ${previewResizing ? "is-active" : ""}`}
              onPointerDown={startPreviewResize}
              role="separator"
              aria-orientation="vertical"
              title={t("topbar.resizePreview")}
            />
            <div className="preview-pane">
              <FilePreview
                relPath={previewFile}
                onClose={closeFilePreview}
                onBackToFiles={backToFiles}
                maximized={previewMaximized}
                onToggleMaximize={togglePreviewMaximize}
              />
              <PreviewNavBar
                files={previewList}
                index={previewIndex}
                onJump={navTo}
                onClose={closePreviewAt}
              />
            </div>
          </>
        )}

        {workspacePanelOpen && (
        <div className="workspace-pane flex flex-col min-w-0 overflow-hidden border-l border-border-soft bg-bg transition-all duration-200">
          {/* v4.23 工作台外壳：Tab 条（含声明式设置齿轮）+ 注册表驱动面板渲染 +
              左缘宽度拖拽手柄（学 dsh-better-sidebar 工作台外壳） */}
          <WorkspaceTabs
            active={activeWorkspaceTab}
            onChange={setRightTab}
            badges={tabBadges}
            enabledTabs={enabledSet}
            onToggleTab={toggleTabEnabled}
          />
          {/* 注册表驱动：激活面板经 sidebarRegistry 渲染（每次只挂载激活面板，与旧行为一致） */}
          <div className="flex-1 min-h-0 overflow-y-auto">
            {getWorkspaceRegistration(activeWorkspaceTab).render(panelContext)}
          </div>
          {/* 宽度拖拽手柄：复用 preview-resizer 悬停/激活样式，absolute 贴左缘 */}
          <div
            className={`preview-resizer ${workspaceResizing ? "is-active" : ""}`}
            style={{ position: "absolute", left: -4, top: 0, bottom: 0, width: 8, zIndex: 5 }}
            onPointerDown={startWorkspaceResize}
            role="separator"
            aria-orientation="vertical"
            title={t("topbar.resizeWorkspacePanel")}
          />
        </div>
        )}
      </div>
      </Layout>

      {state.approval && (
          <ApprovalModal
            approval={state.approval}
            onAnswer={(decision) => {
              approve(state.approval!.id, decision);
            }}
          />
        )}

      {state.ask && (
        <AskCard
          ask={state.ask}
          onAnswer={answerQuestion}
          onDismiss={() => answerQuestion(state.ask!.id, [])}
        />
      )}
      <Suspense fallback={null}>
        {memView !== null && (
          <MemoryPanel
            view={memView}
            onClose={closeMemory}
            onRemember={onRemember}
            onForget={onForget}
            onSaveDoc={onSaveDoc}
            onSaveFact={onSaveFact}
            onChangeType={changeFactType}
            onAcceptMemorySuggestion={onAcceptMemorySuggestion}
            onAcceptSkillSuggestion={onAcceptSkillSuggestion}
            onAcceptMergeSuggestion={onAcceptMergeSuggestion}
            onRefreshSuggestions={onRefreshSuggestions}
          />
        )}
      </Suspense>

      <Suspense fallback={null}>
        {histView !== null && (
          <HistoryPanel
            sessions={histView}
            onResume={onResumeSession}
            onDelete={onDeleteSession}
            onRename={onRenameSession}
            onClose={closeHistory}
          />
        )}
      </Suspense>

      <Suspense fallback={null}>
        {capsOpen && <CapabilitiesPanel onClose={() => setCapsOpen(false)} toolCounts={toolCounts} skillCounts={skillCounts} />}
      </Suspense>


      <Suspense fallback={null}>
        {knowledgeOpen && <KnowledgePanel onClose={closeKnowledge} />}
      </Suspense>

      <CommandPalette
        open={paletteOpen}
        items={paletteItems}
        onClose={() => setPaletteOpen(false)}
      />

      {/* C4 选区转对话：办公板内选中正文 → 浮动「转为提问」→ 引用插入输入框（v3.1.1） */}
      <SelectionToComposer />

    </>
  );
}
