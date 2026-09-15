import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { CSSProperties } from "react";
import { Layout, Modal } from "antd";
import {
  BookOpen, Check, SquarePen, Brain, ChevronDown, FolderGit2, FileText, LineChart,
  PanelRightOpen, PanelRightClose, MessageSquare, Trash2, X, Aim, List, Square, Inbox,
} from "./icons";
import { Sidebar } from "./components/Sidebar";
import { useT } from "./lib/i18n";
import { sessionTitle, sessionTime } from "./lib/session";
import { relativeTime } from "./lib/time";
import { useController, useUpdatedFilesStore } from "./lib/store";
import { app } from "./lib/bridge";
import { GenuiActionProvider } from "../genui/GenuiActionContext";
import { GenuiScopeProvider } from "../genui/scope";
import { scheduleApplyArgsOf } from "../schedule/applyDiff";
import { emitFrontendEvent, FRONTEND_EVENTS } from "../events";
import { setGenuiActionHandler } from "./lib/genuiHost";
import { clearGenuiPanel } from "./lib/genuiPanel";
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
import { WorkspacePane } from "./components/WorkspacePane";
import { ChatTabs, type ChatTabId } from "./components/ChatTabs";
import { ContextView } from "./components/ContextView";
import { ContextModal, ContextPill } from "./components/ContextModal";
import { TrajectoryView } from "./components/TrajectoryView";
import { SubagentThread } from "./components/SubagentThread";
import { FilePreview } from "./components/FilePreview";
import { PreviewNavBar } from "./components/PreviewNavBar";
import { CommandPalette, type PaletteItem } from "./components/CommandPalette";
import { useRunningBadge } from "./hooks/useRunningBadge";
import { Skeleton } from "./components/Skeleton";
import { UpdateBanner } from "./components/UpdateBanner";
import { SelectionToComposer } from "./components/SelectionToComposer";
import { NewSessionToast, JobDoneNotifier, RunStatus } from "./components/AppStatus";

import { exportAsMarkdown } from "./lib/export";
import type { MemoryView, SessionMeta, TaskTemplate } from "./lib/types";
import { useTodoExtractor } from "./hooks/useTodoExtractor";
import { useModeManager } from "./hooks/useModeManager";
import { useSessionManager } from "./hooks/useSessionManager";
import { useBridgeWatch } from "./hooks/useBridgeWatch";
import { useDrawers } from "./hooks/useDrawers";
import { useToolStats } from "./hooks/useToolStats";
import { useSidebar } from "./hooks/useSidebar";
import CompactContext from "./hooks/useCompact";
import { readWorkbenchValue, writeWorkbenchValue } from "./lib/workbenchStorage";

import {
  SIDEBAR_DEFAULT_WIDTH, SIDEBAR_MIN_WIDTH, SIDEBAR_MAX_WIDTH,
} from "./hooks/useLayoutSizes";
import { loadPersistedRightPanelState, type WorkspaceTabId } from "./lib/workspaceTabs";
import { shouldAutoOpenBrowser } from "./lib/browserPrefs";
import { setEventSyncFetcher } from "./lib/eventSync";
import { parseSidebarOpenResult } from "./lib/sidebarOpen";
import { classifyComposerCommand } from "./lib/command";
import { rankPaletteItems } from "./lib/paletteRank";
import { usePaneTabsStore } from "./lib/paneTabs";
import { normalizePreviewPath } from "./lib/officeTurnProjection";
import { SIDEBAR_REGISTRY, type WorkspacePanelContext } from "./lib/sidebarRegistry";
import { loadTemplates, FALLBACK_TEMPLATES } from "./components/Welcome";

// 瘦身 P3（巨文件拆分）：1) 会话/布局/命令等按域拆入 app/ 下 hooks，App.tsx
// 保留为主壳——default export App 与导出面不变；2) palette 命令项、JSX 布局、
// 交付导出 onPick 与既有源级回归锁耦合，原样留在本文件（App.export.test/
// App.palette.test 的 src 级断言即如）。
import { useSubagentTabs } from "./app/useSubagentTabs";
import { useTaskCards } from "./app/useTaskCards";
import { usePreviewAutoFront } from "./app/usePreviewAutoFront";
import { useDeliverables } from "./app/useDeliverables";
import { useWorkspaceLayout } from "./app/useWorkspaceLayout";
import { usePreviewPanel } from "./app/usePreviewPanel";
import { useSessionHandlers } from "./app/useSessionHandlers";
import { useAppKeyboard } from "./app/useAppKeyboard";
import { useTasksAutoOpen } from "./app/useTasksAutoOpen";

export default function App() {
  const toast = useToast();
  // 2.5e /context 居中弹层（dsh 同名能力）。ctxModalRef 非空 = 查看该子代理。
  const [ctxModalOpen, setCtxModalOpen] = useState(false);
  const [ctxModalRef, setCtxModalRef] = useState<string | null>(null);
  const [chatTab, setChatTab] = useState<ChatTabId>(() => {
    try {
      const saved = readWorkbenchValue("gaea.chatTab");
      return saved === "trajectory" || saved === "context" || saved === "memory" ? saved : "chat";
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
    regenerate,
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
  const { histView, setHistView, capsOpen, setCapsOpen, knowledgeOpen, setKnowledgeOpen, closeTopmost } = useDrawers();
  // v4.73：记忆由抽屉迁入主区 tab——视图快照在 App 层维护，动作写盘后统一刷新。
  const [memView, setMemView] = useState<MemoryView | null>(null);
  const { sidebarSessions, sidebarQuery, setSidebarQuery, newSessionDone, refreshSessions, startNewSession, handleResumeSession, handleDeleteSession, handleRenameSession, projectGroups } = useSessionManager(newSession, listSessions, listProjectSessions, resumeSession, deleteSession, renameSession, (msg) => toast.show(msg, "warn"));
  const newSessionAndReset = useCallback(async () => { await startNewSession(); }, [startNewSession]);
  // GenUI action 宿主：运行中 → steer（注入当前回合），空闲 → send（短展示文案 +
  // 完整信封原文走 SubmitDisplay raw）。paylod 紧凑化防超长。
  const genuiActionHandler = useCallback(
    (action: string, payload: Record<string, unknown>) => {
      const envelope = JSON.stringify({ action, payload });
      const raw = `[genui-action] ${envelope}`.slice(0, 1200);
      const display = `（UI 操作）${action}`.slice(0, 80);
      if (state.running) steer(raw);
      else send(display, raw);
    },
    [state.running, steer, send],
  );
  useEffect(() => {
    setGenuiActionHandler(genuiActionHandler);
    return () => setGenuiActionHandler(undefined);
  }, [genuiActionHandler]);
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
  // pane tabs 按会话持久化：会话切换时读档/复位（better-sidebar 会话隔离语义）
  useEffect(() => {
    usePaneTabsStore.getState().setSessionKey(currentSessionKey);
  }, [currentSessionKey]);
  // 独立子代理会话 tabs + task 卡歧义选择器候选 → app/useSubagentTabs
  const {
    subRunsCacheRef, subagentTabs, subagentTabId, openSubagentThread,
    closeSubagentTab, taskPickCandidates, setTaskPickCandidates,
    subagentRuns, handleChatTabSelect,
  } = useSubagentTabs({ currentSessionKey, currentSessionPath, setChatTab });
  // 会话隔离（蒸馏 dsh-better-sidebar）：右侧面板子 Tab 按会话记忆（C3）——
  // 切会话/新建/恢复时恢复该会话上次选中的子面板；显式切换（如点文件回
  // 「文件」面板）照常覆盖当前会话记忆。无会话路径（未保存草稿）回退全局 key。
  // v4.23 会话记录扩为 { tab, enabled, width }（v1 裸 id 旧值宽容兼容）：启用集
  // /宽度的权威在各自全局键（布局偏好跨会话跟随），会话记录仅随存快照。
  const [initialPanelState] = useState(() => loadPersistedRightPanelState(currentSessionKey));
  // 当前激活的 pane 视图（由 WorkspacePane 经 onActiveViewChange 回写；
  // 欢迎卡片态 = null）。pane 打开的文件 tab 归入 files 视图语义。
  const [rightTab, setRightTab] = useState<WorkspaceTabId | null>(null);
  const {
    sidebarCollapsed, sidebarWidth, sidebarResizing, effectiveSidebarWidth,
    toggleSidebar, setExpandedSidebarWidth, startSidebarResize,
    resizeSidebarWithKeyboard, handleWorkspacePreviewModeChange,
  } = useSidebar();
  // v4.23 右栏宽度/视口钳制/持久化 → app/useWorkspaceLayout
  const {
    workspaceResizing, effectiveWorkspaceWidth,
    handleAutoWidenWorkspace, startWorkspaceResize,
  } = useWorkspaceLayout({ initialPanelState, rightTab, currentSessionKey, effectiveSidebarWidth });
  // C6 运行域活动角标：活跃任务数（queued/running）；任务面板激活时视为已读
  // 不显示。v4.53 分工并入任务：运行计数角标挂在「任务」单键上，任务面板
  // （含分工段）激活即视为已读。
  const runningTasks = useRunningBadge();
  const runningDomainActive = rightTab === "tasks";
  const runningBadge = runningDomainActive ? undefined : { tasks: runningTasks };
  const [compactMode, setCompactMode] = useState(() => readWorkbenchValue("gaea.compactMode") === "1");
  const [scrollToTurn, setScrollToTurn] = useState<((turn: number) => void) | null>(null);
  const [paletteOpen, setPaletteOpen] = useState(false);
  // 7.3-1 任务收件箱：命令面板动态「存为任务」项需要拿到面板输入 query——
  // CommandPalette 的 query 是其内部状态（App 拿不到，且组件契约不在本刀足迹），
  // 这里在面板打开期间以捕获期 input 监听镜像输入框值（.palette__input 唯一
  // 类名锚定），面板开/关复位。纯用户动作驱动，零轮询零常驻（判据④）。
  const [paletteQuery, setPaletteQuery] = useState("");
  useEffect(() => {
    if (!paletteOpen) return;
    setPaletteQuery("");
    const onPaletteInput = (e: Event) => {
      const el = e.target as HTMLElement | null;
      if (el && el.classList?.contains("palette__input")) setPaletteQuery((el as HTMLInputElement).value);
    };
    document.addEventListener("input", onPaletteInput, true);
    return () => {
      document.removeEventListener("input", onPaletteInput, true);
      setPaletteQuery("");
    };
  }, [paletteOpen]);
  const [deleteConfirm, setDeleteConfirm] = useState(false);
  // 主区大预览 + 右侧工作台交互编排（预览队列/专注模式/pane 打开/拖拽）→ app/usePreviewPanel
  const {
    workspacePanelOpen, setWorkspacePanel,
    workspaceRefreshKey, setWorkspaceRefreshKey,
    previewFile, previewList, previewIndex, closeFilePreview, navTo, closePreviewAt,
    previewReloadTicks,
    focusMode, toggleFocus,
    openFilePreview, openPaneView, openPaneFile, backToFiles,
    toggleWorkspacePanel,
    previewWidth, previewMaximized, previewResizing, previewMaxWidth,
    togglePreviewMaximize, startPreviewResize,
  } = usePreviewPanel({ effectiveSidebarWidth, setRightTab, handleWorkspacePreviewModeChange });

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

  // 记忆视图：打开（主区 tab / 命令 / /memory）时取新鲜快照；
  // 写操作成功后 re-fetch，标签页即时反映落盘结果。
  const openMemory = useCallback(async () => {
    setMemView(await fetchMemory());
  }, [fetchMemory, setMemView]);

  // 主区 tab 激活「记忆」即拉取新鲜快照（后续写操作自行刷新）。
  useEffect(() => {
    if (chatTab === "memory") void openMemory();
  }, [chatTab, openMemory]);

  const openKnowledge = useCallback(() => setKnowledgeOpen(true), [setKnowledgeOpen]);
  const closeKnowledge = useCallback(() => setKnowledgeOpen(false), [setKnowledgeOpen]);

  // handleSend intercepts the slash commands that need a desktop-native action
  // before they reach the backend: "/model <ref>" rebuilds on that model,
  // "/memory" switches to the memory tab, and "/context" switches to the
  // context dashboard tab. Everything else — skills (/init, …), custom
  // commands, bare /model and the other read-only management verbs
  // (/skill, /hooks, /mcp) — goes straight to Submit, which the controller
  // resolves (a turn, or a listing Notice).
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
        setChatTab("memory");
        return;
      }
      if (command.type === "context") {
        // 2.5e：/context 打开居中弹层（不离开对话）；主区上下文 tab 保留手动切换。
        setCtxModalRef(null);
        setCtxModalOpen(true);
        return;
      }
      if (command.type === "panel") {
        if (command.action === "clear") {
          clearGenuiPanel(currentSessionPath);
          return;
        }
        openPaneView("ui");
        if (command.instruction !== undefined) {
          send(`/panel ${command.instruction}`);
        }
        return;
      }
      send(displayText.trim(), submitText.trim());
    },
    [switchModel, send, setChatTab, openPaneView, currentSessionPath],
  );

  // 会话/历史/记忆操作回调集 → app/useSessionHandlers
  const {
    openHistory, closeHistory, onResumeSession, onDeleteSession, onRenameSession,
    confirmDeleteCurrent, switchFolder, resumeSessionInProject, recentSessions,
    resumeRecentSession, onArchiveSession, onPinSession, onRestoreSession,
    onRemember, onForget, onSaveDoc, onSaveFact, onAcceptMemorySuggestion,
    onAcceptSkillSuggestion, onAcceptMergeSuggestion, onRefreshSuggestions,
    distillView, distillLoading, refreshDistill, draftDistill, ignoreDistill, crystallizeDistill,
  } = useSessionHandlers({
    t, toast, refreshSessions, deleteSession, newSessionAndReset, pickWorkspace,
    switchWorkspace, handleResumeSession, handleDeleteSession, handleRenameSession,
    projectGroups, archiveSession, unarchiveSession, pinSession, remember, forget,
    saveDoc, updateFact, fetchMemory, setMemView, setHistView, setDeleteConfirm,
    closeFilePreview, setWorkspacePanel,
  });

  useEffect(() => { void refreshSessions(); }, [cwd, refreshSessions]);

  // 全局快捷键 → app/useAppKeyboard（deps 原样）
  useAppKeyboard({
    state, closeTopmost, workspacePanelOpen, previewFile, toggleFocus,
    closeFilePreview, newSessionAndReset, openHistory, openKnowledge,
    toggleSidebar, toggleWorkspacePanel, setPaletteOpen,
  });

  const { toolCounts, skillCounts } = useToolStats(state.items);
  // 会话产物/文件变更派生 + 新产物角标自动置前 → app/useDeliverables
  const { sessionDeliverables, sessionChanges, freshDeliverablePaths, freshDeliverableCount } = useDeliverables({ state, currentSessionKey, rightTab, openPaneView });
  // 预览浮窗状态机（U2）+ 写后预览实时跟随（U4）→ app/usePreviewAutoFront
  usePreviewAutoFront({ state, currentSessionKey, workspacePanelOpen, previewFile, focusMode });
  // 新后台任务/子代理运行自动激活任务页 → app/useTasksAutoOpen
  const { openTasksAuto } = useTasksAutoOpen({ rightTab, openPaneView, currentSessionPath, subagentRuns });

  const tabBadges = runningBadge
    ? { ...runningBadge, ...(freshDeliverableCount > 0 ? { deliverables: freshDeliverableCount } : {}) }
    : freshDeliverableCount > 0 ? { deliverables: freshDeliverableCount } : undefined;

  // 编辑后自动回写刷新：docx/xlsx 预览内编辑成功 → 文件树自动刷新（替代手动刷新）
  const updatedAt = useUpdatedFilesStore((s) => s.updatedAt);
  useEffect(() => {
    if (Object.keys(updatedAt).length === 0) return;
    setWorkspaceRefreshKey((k) => k + 1);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- setWorkspaceRefreshKey 为 hook 返回的稳定 dispatch，deps 原样保留
  }, [updatedAt]);

  // 会话切换（启动装载/新建/恢复/切换工作区）后拉取会话级派生统计，
  // 回填「全会话成本」的历史部分（评审缺陷 11）。
  useEffect(() => {
    if (!currentSessionPath) return;
    void fetchSessionStats(currentSessionPath);
  }, [currentSessionPath, fetchSessionStats]);

  // ── v4.23 注册表渲染上下文：右栏面板公共依赖一次性组装 ──
  // （学 better-sidebar 框架/内容解耦：面板 props 与旧渲染分支逐一对应，行为不变）
  // eslint-disable-next-line react-hooks/exhaustive-deps -- setWorkspaceRefreshKey 为 hook 返回的稳定 dispatch，deps 原样保留
  const refreshWorkspacePanel = useCallback(() => setWorkspaceRefreshKey((k) => k + 1), []);
  const locateDeliverableSource = useCallback((turn: number) => {
    setWorkspacePanel(false);
    scrollToTurn?.(turn);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- setWorkspacePanel 为 hook 返回的稳定 dispatch，deps 原样保留
  }, [scrollToTurn]);
  // v4.24 A1 新子代理自动展开：分工面板检测到新子代理出现（且用户偏好开启）
  // 时回调——亮出右栏并切到「任务」面板（v4.53 分工并入任务同屏展示；
  // 对标 better-sidebar 任务页自动展开侧栏）。pane 语义：开任务视图 tab。
  const handleSubagentStarted = useCallback(() => {
    openTasksAuto();
  }, [openTasksAuto]);

  // v4.25 A3 reveal：产物面板「树中定位」→ 亮文件 tab，文件树展开父链+滚动+闪烁。
  // nonce 单调递增，同一文件重复定位也能再触发一次。
  const revealNonceRef = useRef(0);
  const [revealRequest, setRevealRequest] = useState<{ rel: string; nonce: number } | null>(null);
  const handleRevealInTree = useCallback((rel: string) => {
    openPaneView("files");
    revealNonceRef.current += 1;
    setRevealRequest({ rel, nonce: revealNonceRef.current });
  }, [openPaneView]);

  // v4.25 模型主动打开（对标 better-sidebar sidebar_open）：模型把关键产物/目录
  // 推到右栏文件工作台。按工具事件 id 去重；file → pane 文件 tab，
  // directory → 文件树树中定位（v4.28 起接 reveal：
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
      if (parsed.kind === "file") openPaneFile(parsed.pathRel);
      else if (parsed.kind === "directory") handleRevealInTree(parsed.pathRel);
      requested = true;
    }
    if (requested) {
      openPaneView("files");
    }
  }, [state.items, openPaneView, openPaneFile, handleRevealInTree]);

  // v4.28 A2 浏览器观察窗自动弹出：会话轨迹出现新 browser_* 工具且偏好开
  // （gaea.browserAutoOpen）→ 亮右栏开「浏览器」视图 tab。
  const browserSeenRef = useRef<Set<string>>(new Set());
  useEffect(() => {
    let requested = false;
    for (const it of state.items) {
      if (it.kind !== "tool" || !it.name.startsWith("browser_")) continue;
      if (browserSeenRef.current.has(it.id)) continue;
      browserSeenRef.current.add(it.id);
      requested = true;
    }
    if (requested && shouldAutoOpenBrowser()) {
      openPaneView("browser");
    }
  }, [state.items, openPaneView]);

  // v4.26 对话流式重造接线：
  // ① 事件序号防线 fetcher——Wails 事件流吞件（seq 跳号）时经后端从磁盘日志
  //    补拉对话项全量快照（eventSync 冷却门控频；未挂载则整条防线旁路）。
  useEffect(() => {
    setEventSyncFetcher((afterSeq) => app.ResyncEvents(afterSeq));
    return () => setEventSyncFetcher(null);
  }, []);
  // 子代理 task 卡 live 预览/跳转/歧义选择器注入 → app/useTaskCards
  useTaskCards({ subRunsCacheRef, setTaskPickCandidates, currentSessionPath, openSubagentThread });

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
      openSubagentThread: (p) => {
        openSubagentThread(p);
      },
    }),
    // eslint-disable-next-line react-hooks/exhaustive-deps -- setWorkspacePanel 为 hook 返回的稳定 dispatch（onClosePanel 内引用），deps 原样保留
    [
      state.meta?.cwd, previewFile, workspaceRefreshKey, currentSessionPath,
      sessionDeliverables, sessionChanges, freshDeliverablePaths,
      openFilePreview, refreshWorkspacePanel, locateDeliverableSource,
      handleSubagentStarted, handleRevealInTree, revealRequest, handleAutoWidenWorkspace,
      openSubagentThread,
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
      // 办公域「进度计划」入口（v4.168 刀1b）：Ctrl+K 直达 schedule 板块，与
      // ScheduleFileCard.tsx:67 同款 NAVIGATE 事件（前置 v4.167.0 导航白名单已解耦，
      // 引擎零改动）。title 复用既有键 schedCard.tag（zh=进度计划 / zh-TW=進度計畫 /
      // en=Schedule，内容层不铺新字典）。「最近工程列表+打开工作台」不由本命令项
      // 承担：办公文件面 RecentFilesBar 复用 lib/recentFiles 单源（.gsched 文件被
      // 预览/引用即入列），点摘要卡「在进度计划中打开」已达，此处仅作可检索入口。
      { id: "cmd-schedule", group: t("palette.group.commands") ?? "命令", title: t("schedCard.tag") ?? "进度计划", icon: <LineChart size={15} />, compact: true, keywords: ["schedule", "进度", "计划", "工作台"], run: () => { emitFrontendEvent(FRONTEND_EVENTS.NAVIGATE, { page: "schedule" }); } },
      { id: "cmd-memory", group: t("palette.group.commands") ?? "命令", title: t("topbar.memory") ?? "记忆", icon: <Brain size={15} />, compact: true, keywords: ["memory", "记忆"], run: () => { setChatTab("memory"); void openMemory(); } },
      { id: "cmd-history", group: t("palette.group.commands") ?? "命令", title: t("topbar.history") ?? "历史", icon: <MessageSquare size={15} />, compact: true, keywords: ["history", "历史"], run: () => void openHistory() },
      { id: "cmd-knowledge", group: t("palette.group.commands") ?? "命令", title: t("topbar.knowledge") ?? "知识库", icon: <BookOpen size={15} />, compact: true, keywords: ["knowledge", "知识库"], run: () => void openKnowledge() },
    ];
    // 右侧面板命令项：由 sidebarRegistry 注册表派生；pane 语义下命令 = 开视图
    // tab（tasks 不在命令面板，与原行为一致；欢迎卡里可点）。
    for (const reg of SIDEBAR_REGISTRY) {
      if (reg.id === "tasks") continue;
      const Icon = reg.icon;
      cmds.push({
        id: `cmd-${reg.id}`,
        group: t("palette.group.commands") ?? "命令",
        title: `${reg.label}面板`,
        icon: <Icon size={15} />,
        compact: true,
        keywords: reg.keywords,
        run: () => { openPaneView(reg.id); },
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
    // 7.3-1 任务收件箱：query 非空时尾部动态项「存为任务『query』」——办公
    // 工作台内落 work 空间（space='work'）；session=当前会话路径若有（审计链，
    // currentSessionPath 为 sidebarSessions 中 current 会话的 .jsonl 路径）；
    // 标题上限镜像后端 MaxTitleRunes=120（超长截断，存任务不该被长度卡死）。
    const saveTaskTitle = paletteQuery.trim().slice(0, 120);
    const saveTaskItems: PaletteItem[] = saveTaskTitle
      ? [{
          id: "cmd-save-task",
          group: t("palette.group.commands") ?? "命令",
          title: t("palette.saveTask", { query: saveTaskTitle }),
          icon: <Inbox size={15} />,
          compact: true,
          keywords: ["save", "task", "存为任务", "任务", "inbox"],
          run: () => {
            void app
              .GaeaTaskInboxSave(
                JSON.stringify({
                  title: saveTaskTitle,
                  space: "work",
                  source: "palette",
                  ...(currentSessionPath ? { session: currentSessionPath } : {}),
                }),
              )
              .catch(() => {});
          },
        }]
      : [];
    // v4.30 命令面板按当前视图重排（Linear 式）：当前激活的右栏面板 / 主区
    // tab 对应命令置顶，其余保持稳定原序（纯函数，见 lib/paletteRank）。
    return rankPaletteItems(
      [...cmds, ...templateItems, ...sessionItems, ...saveTaskItems],
      { chatTab, rightTab: rightTab ?? "files" },
    );
    // eslint-disable-next-line react-hooks/exhaustive-deps -- setWorkspacePanel 为 hook 返回的稳定 dispatch（模板 run 内引用），deps 原样保留
  }, [t, sidebarSessions, openMemory, openHistory, openKnowledge, onResumeSession, openPaneView, closeFilePreview, templates, send, newSessionAndReset, chatTab, rightTab, paletteQuery, currentSessionPath]);

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
  const activeSubagent = subagentTabs.find((t) => t.id === subagentTabId) ?? null;

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
          onOpenSubagentThread={openSubagentThread}
          onOpenHistory={openHistory}
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
              {/* 审计刀B c：md 分支一并收口 exportConversation 统一交付管线 */}
              <ExportMenu
                disabled={state.items.length === 0}
                onPick={(format: ExportFormat) => {
                  void exportConversation(format);
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
          <ChatTabs
            active={subagentTabId ?? chatTab}
            onChange={handleChatTabSelect}
            extraTabs={subagentTabs.map((x) => {
              const statusText =
                x.status === "running"
                  ? t("subagent.statusRunning")
                  : x.status === "failed"
                    ? t("subagent.statusFailed")
                    : t("subagent.statusDone");
              return {
                id: x.id,
                label:
                  x.task && x.task.length > 14
                    ? `${x.task.slice(0, 14)}…`
                    : (x.task || x.ref),
                status: x.status,
                detail: `${x.task || x.ref} ｜ ${statusText}${
                  x.kind === "model_tool" && !x.model
                    ? ` · ${t("subagent.modelToolLabel")}`
                    : x.model
                      ? ` · ${x.model}`
                      : ""
                }`,
              };
            })}
            onCloseExtra={closeSubagentTab}
          />
          <main className="main">
            <CompactContext.Provider value={compactMode}>
            {(state.meta?.ready === false && !state.meta?.startupErr) || switchingModel ? (
              <Skeleton />
            ) : (
              <>
                {activeSubagent ? (
                  <SubagentThread
                    key={activeSubagent.id}
                    sessionPath={activeSubagent.sessionPath}
                    target={activeSubagent.ref}
                    task={activeSubagent.task}
                    status={activeSubagent.status}
                    model={activeSubagent.model}
                    onBack={() => closeSubagentTab(activeSubagent.id)}
                  />
                ) : (
                  <>
                    {chatTab === "chat" && (
                      <GenuiScopeProvider scope={{ scope: "office", sessionKey: currentSessionKey }}>
                        <GenuiActionProvider onAction={genuiActionHandler}>
                          <Transcript onPrompt={send} running={state.running} onRewind={rewind} onRegenerate={regenerate} onFeedback onScrollToTurnReady={setScrollToTurn} cwd={state.meta?.cwd} cwdName={cwdName} sessions={recentSessions} onResumeSession={resumeRecentSession} meta={state.meta} />
                          {state.items.length > 1 && <JumpBar items={state.items} scrollToTurn={scrollToTurn ?? undefined} />}
                        </GenuiActionProvider>
                      </GenuiScopeProvider>
                    )}
                    {chatTab === "trajectory" && <TrajectoryView running={state.running} sessionPath={currentSessionPath ?? undefined} />}
                    {chatTab === "context" && (
                      <ContextView
                        running={state.running}
                        sessionPath={currentSessionPath ?? undefined}
                        onViewSubagentContext={(ref) => {
                          setCtxModalRef(ref);
                          setCtxModalOpen(true);
                        }}
                        sessionName={
                          currentSessionPath
                            ? sessionTitle(
                                (sidebarSessions.find((s) => s.path === currentSessionPath) ?? { path: currentSessionPath, title: "", preview: "" }) as SessionMeta,
                                currentSessionPath.split(/[/]/).pop() ?? "",
                              )
                            : undefined
                        }
                        model={state.meta?.label ?? undefined}
                      />
                    )}
                    {chatTab === "memory" && (
                      <MemoryPanel
                        view={memView}
                        onRemember={onRemember}
                        onForget={onForget}
                        onSaveDoc={onSaveDoc}
                        onSaveFact={onSaveFact}
                        onChangeType={changeFactType}
                        onAcceptMemorySuggestion={onAcceptMemorySuggestion}
                        onAcceptSkillSuggestion={onAcceptSkillSuggestion}
                        onAcceptMergeSuggestion={onAcceptMergeSuggestion}
                        onRefreshSuggestions={onRefreshSuggestions}
                        distillView={distillView}
                        distillLoading={distillLoading}
                        onRefreshDistill={refreshDistill}
                        onDraftDistill={draftDistill}
                        onIgnoreDistill={ignoreDistill}
                        onCrystallizeDistill={crystallizeDistill}
                      />
                    )}
                  </>
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
            {/* 2.5e 常驻「剩余上下文%」徽标（codex 式；点击打开居中弹层） */}
            <div className="flex justify-end px-4">
              <ContextPill
                used={state.context.used}
                window={state.context.window}
                onClick={() => {
                  setCtxModalRef(null);
                  setCtxModalOpen(true);
                }}
              />
            </div>
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
                reloadSignal={previewReloadTicks[normalizePreviewPath(previewFile)] ?? 0}
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
          {/* pane 工作台：空态欢迎卡 → 点卡片开视图 tab；资源管理器内点文件
              新增文件 tab（对标 better-sidebar；内容分发在 WorkspacePane 内） */}
          <WorkspacePane
            context={panelContext}
            badges={tabBadges}
            onActiveViewChange={(viewId) => {
              if (viewId !== rightTab) setRightTab(viewId);
            }}
          />
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
            scheduleApplyArgs={scheduleApplyArgsOf(state.items)}
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

      {/* 2.5e /context 居中弹层：不离开对话查看当前上下文构成 */}
      <ContextModal
        open={ctxModalOpen}
        onClose={() => {
          setCtxModalOpen(false);
          setCtxModalRef(null);
        }}
        fetchTimeline={
          ctxModalRef && currentSessionPath
            ? () => app.GaeaSubagentContextView(currentSessionPath, ctxModalRef)
            : undefined
        }
        title={ctxModalRef ? `子代理上下文 · ${ctxModalRef}` : undefined}
        running={state.running}
        sessionPath={currentSessionPath ?? undefined}
        sessionName={
          currentSessionPath
            ? sessionTitle(
                (sidebarSessions.find((s) => s.path === currentSessionPath) ?? { path: currentSessionPath, title: "", preview: "" }) as SessionMeta,
                currentSessionPath.split(/[/]/).pop() ?? "",
              )
            : undefined
        }
        model={state.meta?.label ?? undefined}
      />

      {/* v4.68 task 卡多候选歧义选择器：一张卡同时匹配 ≥2 个 running 子代理
          时（并行派发、任务文本相近），点击卡片弹此居中轻量弹层，人工挑一个
          打开对应会话 tab。点行走既有 openSubagentThread；Esc/遮罩/关闭均取消。 */}
      <Modal
        open={taskPickCandidates !== null}
        onCancel={() => setTaskPickCandidates(null)}
        footer={null}
        width={480}
        centered
        title={t("taskpick.title")}
        destroyOnHidden
      >
        <div data-testid="task-pick" className="flex flex-col gap-1">
          <div className="pb-1 text-[11.5px] leading-snug text-fg-faint">{t("taskpick.desc")}</div>
          {(taskPickCandidates ?? []).map((r) => (
            <button
              key={r.ref}
              type="button"
              data-testid="task-pick-row"
              aria-label={t("taskpick.rowAria", { task: r.task })}
              onClick={() => {
                setTaskPickCandidates(null);
                openSubagentThread({ sessionPath: currentSessionPath ?? "", ref: r.ref, task: r.task, status: r.status, model: r.model });
              }}
              className="flex w-full cursor-pointer items-center gap-2 rounded-md border border-border-soft bg-bg-soft px-2 py-1.5 text-left transition-colors hover:bg-(color:--md-sys-color-surface-container-high)"
            >
              <span className="inline-flex shrink-0 items-center gap-1 text-[10.5px] text-fg-faint">
                <span className={`h-1.5 w-1.5 rounded-full ${r.status === "running" ? "animate-pulse bg-accent" : r.status === "failed" ? "bg-err" : "bg-info/70"}`} />
                {r.status === "running" ? t("taskpick.statusRunning") : r.status === "failed" ? t("taskpick.statusFailed") : t("taskpick.statusCompleted")}
              </span>
              <span className="min-w-0 flex-1 truncate text-[12px] text-fg" title={r.task}>{r.task}</span>
              {r.model && <span className="max-w-[30%] shrink-0 truncate font-mono text-[10.5px] text-fg-faint" title={r.model}>{r.model}</span>}
              <span className="shrink-0 text-[10.5px] tabular-nums text-fg-faint" title={r.createdAt}>{relativeTime(Date.parse(r.createdAt))}</span>
            </button>
          ))}
        </div>
      </Modal>

    </>
  );
}