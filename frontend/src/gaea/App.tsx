import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { CSSProperties } from "react";
import { Layout } from "antd";
import {
  BookOpen, Check, SquarePen, Brain, ChevronDown, FolderGit2, FileText,
  PanelRightOpen, PanelRightClose, MessageSquare, Trash2, X, Aim, List, Square,
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
import { GoalCard } from "./components/GoalCard";
import { TodoCard } from "./components/TodoCard";
import { ApprovalModal } from "./components/ApprovalModal";
import { AskCard } from "./components/AskCard";
import { ToolbarButton } from "./components/ToolbarButton";
import { ContextBar } from "./components/ContextBar";
import { ModelSwitcher } from "./components/ModelSwitcher";
const MemoryPanel = lazy(() => import("./components/MemoryPanel").then(m => ({ default: m.MemoryPanel })));
const HistoryPanel = lazy(() => import("./components/HistoryPanel").then(m => ({ default: m.HistoryPanel })));
const CapabilitiesPanel = lazy(() => import("./components/CapabilitiesPanel").then(m => ({ default: m.CapabilitiesPanel })));
const KnowledgePanel = lazy(() => import("./components/KnowledgePanel").then(m => ({ default: m.KnowledgePanel })));
import { WorkspacePanel } from "./components/WorkspacePanel";
import { WorkspaceTabs } from "./components/WorkspaceTabs";
import { FilePreview } from "./components/FilePreview";
import { PreviewNavBar } from "./components/PreviewNavBar";
import { DeliverablesPanel, type SessionDeliverable } from "./components/DeliverablesPanel";
import { MaterialsPanel } from "./components/MaterialsPanel";
import { CostLibraryPanel } from "./components/CostLibraryPanel";
import { CommandPalette, type PaletteItem } from "./components/CommandPalette";
import { StatsPanel, useStatsPersistence } from "./components/StatsPanel";
import { ChangesPanel } from "./components/ChangesPanel";
import { TaskCenter } from "./components/TaskCenter";
import { SubagentsPanel } from "./components/SubagentsPanel";
import { useRunningBadge } from "./hooks/useRunningBadge";
import { Skeleton } from "./components/Skeleton";
import { UpdateBanner } from "./components/UpdateBanner";
import { SelectionToComposer } from "./components/SelectionToComposer";
import { NewSessionToast, JobDoneNotifier, RunStatus } from "./components/AppStatus";

import { downloadMarkdown, exportAsMarkdown } from "./lib/export";
import type { MemorySuggestion, MemorySuggestionsView, Requirement, SessionMeta, SkillSuggestion } from "./lib/types";
import { useTodoExtractor } from "./hooks/useTodoExtractor";
import { useModeManager } from "./hooks/useModeManager";
import { useSessionManager } from "./hooks/useSessionManager";
import { useBridgeWatch } from "./hooks/useBridgeWatch";
import { useDrawers } from "./hooks/useDrawers";
import { useToolStats } from "./hooks/useToolStats";
import { useSidebar } from "./hooks/useSidebar";

import {
  SIDEBAR_DEFAULT_WIDTH, SIDEBAR_MIN_WIDTH, SIDEBAR_MAX_WIDTH,
} from "./hooks/useLayoutSizes";
import {
  PREVIEW_MAX_WIDTH, PREVIEW_MIN_WIDTH, clampPreviewWidth,
  loadPreviewWidth, savePreviewWidth,
} from "./lib/layoutPreferences";
import CompactContext from "./hooks/useCompact";
import { deliverableMentions } from "./lib/fileLinks";
import { recordRecentFile } from "./lib/recentFiles";
import { useUpdatedFilesStore } from "./lib/store";
import { buildSessionChanges, type SessionChange } from "./lib/changes";
import { classifyComposerCommand } from "./lib/command";
import { WORKSPACE_TABS, loadPersistedRightTab, savePersistedRightTab, groupOfTab, type WorkspaceTabId } from "./lib/workspaceTabs";
import { loadTemplates, FALLBACK_TEMPLATES } from "./components/Welcome";
import type { TaskTemplate } from "./lib/types";

export default function App() {
  const toast = useToast();
  const {
    state,
    send,
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
    fetchRequirement,
    setRequirement,
    setRequirementDone,
    addRequirementItem,
    setRequirementItem,
    removeRequirementItem,
    setRequirementItemDone,
    setRequirementAutoPursue,
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
  const [rightTab, setRightTab] = useState<WorkspaceTabId>(() => loadPersistedRightTab(currentSessionKey));
  // currentSessionKey 变化（会话切换）时：从新会话 key 恢复面板 Tab；
  // 切换本身不覆盖新会话已存的记忆（仅当新会话无记录时回退默认）。
  const prevSessionKey = useRef(currentSessionKey);
  useEffect(() => {
    if (prevSessionKey.current === currentSessionKey) return;
    prevSessionKey.current = currentSessionKey;
    setRightTab(loadPersistedRightTab(currentSessionKey));
  }, [currentSessionKey]);
  useEffect(() => { savePersistedRightTab(rightTab, currentSessionKey); }, [rightTab, currentSessionKey]);
  // C6 运行域活动角标：活跃任务数（queued/running）；「运行」组激活时视为已读不显示。
  const runningTasks = useRunningBadge();
  const runningGroupActive = groupOfTab(rightTab).id === "running";
  const runningBadge = runningGroupActive ? undefined : { running: runningTasks };
  const [compactMode, setCompactMode] = useState(() => { try { return localStorage.getItem("gaea.compactMode") === "1"; } catch { return false; } });
  const [scrollToTurn, setScrollToTurn] = useState<((turn: number) => void) | null>(null);
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState(false);

  const {
    sidebarCollapsed, sidebarWidth, sidebarResizing, effectiveSidebarWidth,
    toggleSidebar, setExpandedSidebarWidth, startSidebarResize,
    resizeSidebarWithKeyboard, handleWorkspacePreviewModeChange,
  } = useSidebar();

  const [workspacePanelOpen, setWorkspacePanel] = useState(false);
  const [previewWidth, setPreviewWidth] = useState(loadPreviewWidth);
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
    try { return localStorage.getItem("gaea.focusMode") === "1"; } catch { return false; }
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
    try { localStorage.setItem("gaea.focusMode", next ? "1" : "0"); } catch { /* ignore */ }
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

  // 拖拽分割条调整预览宽度
  const startPreviewResize = useCallback((e: React.PointerEvent) => {
    e.preventDefault();
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

  // 统一交付出口：会话成果一键导出 Word（docx/pptx/xlsx 同管线）。
  const exportConversation = useCallback(async (format: "docx" | "pptx" | "xlsx" | "md") => {
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
      toast.show(`已导出 ${r.name}`, "info");
      void app.RevealWorkspacePath(r.path).catch(() => {});
    } catch (e) {
      toast.show(e instanceof Error ? e.message : String(e), "warn");
    }
  }, [state.items, toast]);
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
  // before they reach the backend: "/model <ref>" rebuilds on that model, and
  // "/memory" opens the memory drawer. Everything else — skills (/init, …),
  // custom commands, bare /model and the other read-only management verbs
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
        void openMemory();
        return;
      }
      send(displayText.trim(), submitText.trim());
    },
    [switchModel, openMemory, send],
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
      toast.show(`归档失败：${e instanceof Error ? e.message : String(e)}`, "warn");
    }
  }, [archiveSession, refreshSessions, toast]);

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
  // 会话产物：从会话消息文本中提取交付文件（保留首现顺序；同一文件多次
  // 出现计入 versions 次数——产物版本时间线数据源，对标 Hermes 版本步进器）。
  const sessionDeliverables = useMemo<SessionDeliverable[]>(() => {
    const order = new Map<string, { sourceId: string; turn: number; versions: number }>();
    let turn = -1;
    for (const it of state.items) {
      if (it.kind === "user") {
        turn++;
        continue;
      }
      if (it.kind !== "assistant" || !it.text) continue;
      for (const p of deliverableMentions(it.text)) {
        const rec = order.get(p);
        if (rec) {
          rec.versions++;
        } else {
          order.set(p, { sourceId: it.id, turn: Math.max(0, turn), versions: 1 });
        }
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

  // ── 任务目标（Kun「从需求到验收」）：会话首条消息自动成为目标，随会话持久化 ──
  const [requirement, setRequirementState] = useState<Requirement | null>(null);
  const capturedReqPathRef = useRef<string | null>(null);

  const refreshRequirement = useCallback(async () => {
    const p = currentSessionPath;
    if (!p) { setRequirementState(null); return; }
    const r = await fetchRequirement(p);
    if (r?.text) {
      capturedReqPathRef.current = p;
      setRequirementState(r);
    } else if (capturedReqPathRef.current !== p) {
      setRequirementState(null);
    }
  }, [currentSessionPath, fetchRequirement]);

  useEffect(() => { void refreshRequirement(); }, [refreshRequirement]);

  // 首条用户消息自动捕获为任务目标（每个会话只捕获一次）
  const firstUserItem = state.items.find((it) => it.kind === "user");
  useEffect(() => {
    if (!currentSessionPath || !firstUserItem) return;
    if (capturedReqPathRef.current === currentSessionPath) return;
    capturedReqPathRef.current = currentSessionPath;
    void setRequirement(currentSessionPath, firstUserItem.text).then(() => refreshRequirement());
  }, [currentSessionPath, firstUserItem, setRequirement, refreshRequirement]);

  const toggleRequirementDone = useCallback(async () => {
    if (!currentSessionPath || !requirement) return;
    await setRequirementDone(currentSessionPath, !requirement.done);
    await refreshRequirement();
  }, [currentSessionPath, requirement, setRequirementDone, refreshRequirement]);

  const mutateRequirement = useCallback(
    async (fn: (path: string) => Promise<unknown>) => {
      if (!currentSessionPath) return;
      try {
        await fn(currentSessionPath);
      } finally {
        await refreshRequirement();
      }
    },
    [currentSessionPath, refreshRequirement],
  );

  const handleAddRequirementItem = useCallback(
    (text: string) => {
      void mutateRequirement((p) => addRequirementItem(p, text));
    },
    [mutateRequirement, addRequirementItem],
  );
  const handleSetRequirementItem = useCallback(
    (index: number, text: string) => {
      void mutateRequirement((p) => setRequirementItem(p, index, text));
    },
    [mutateRequirement, setRequirementItem],
  );
  const handleRemoveRequirementItem = useCallback(
    (index: number) => {
      void mutateRequirement((p) => removeRequirementItem(p, index));
    },
    [mutateRequirement, removeRequirementItem],
  );
  const handleSetRequirementItemDone = useCallback(
    (index: number, done: boolean) => {
      void mutateRequirement((p) => setRequirementItemDone(p, index, done));
    },
    [mutateRequirement, setRequirementItemDone],
  );
  const handleToggleRequirementAutoPursue = useCallback(() => {
    void mutateRequirement((p) => setRequirementAutoPursue(p, !requirement?.autoPursue));
  }, [mutateRequirement, setRequirementAutoPursue, requirement?.autoPursue]);

  const statsPersistence = useStatsPersistence(currentSessionKey, statsReset, state.turnSteps, state.perTurnUsage);

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
    ];
    // 右侧面板命令项：由 WORKSPACE_TABS 清单派生（tasks 不在命令面板，与原行为一致）
    for (const tab of WORKSPACE_TABS) {
      if (tab.id === "tasks") continue;
      const Icon = tab.icon;
      cmds.push({
        id: `cmd-${tab.id}`,
        group: t("palette.group.commands") ?? "命令",
        title: `${tab.label}面板`,
        icon: <Icon size={15} />,
        compact: true,
        keywords: tab.keywords,
        // 原实现中 stats 命令不关闭预览（统计面板可与预览并存），其余面板关闭预览
        run: () => { if (tab.id !== "stats") closeFilePreview(); setWorkspacePanel(true); setRightTab(tab.id); },
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
    return [...cmds, ...templateItems, ...sessionItems];
  }, [t, sidebarSessions, openMemory, openHistory, openKnowledge, onResumeSession, setWorkspacePanel, closeFilePreview, setRightTab, templates, send, newSessionAndReset]);

  const layoutStyle = useMemo(
    () =>
      ({
        "--sidebar-expanded-width": `${sidebarWidth}px`,
        "--preview-width": `${previewWidth}px`,
      }) as CSSProperties,
    [sidebarWidth, previewWidth],
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
            {/* 顶栏上下文用量 — 单模型 */}
            {state.context.window > 0 && (
              <div className="flex flex-row gap-2 min-w-[260px] max-w-[360px] flex-1">
                <div className="flex-1 min-w-0">
                  <ContextBar
                    label="上下文"
                    used={state.context.used}
                    window={state.context.window}
                    color="bg-cyan-500/60"
                  />
                </div>
              </div>
            )}
            <div className="flex items-center gap-2 px-3">
              {cwd && (<button className="toolbar-btn no-drag" onClick={() => void switchFolder()} disabled={state.running}><FolderGit2 size={13} /><span>{cwdName}</span><ChevronDown size={11} /></button>)}
            </div>
            <div className="flex-1" />
            <div className="flex items-center gap-2">
              <ToolbarButton onClick={() => void toggleWorkspacePanel()} title={previewFile ? "返回文件列表" : workspacePanelOpen ? "收起文件面板" : "展开文件面板"}>
                {workspacePanelOpen || previewFile ? <PanelRightClose size={13} /> : <PanelRightOpen size={13} />}
              </ToolbarButton>
              <ToolbarButton onClick={() => { const v = !compactMode; setCompactMode(v); try { localStorage.setItem("gaea.compactMode", v ? "1" : "0"); } catch {} }} title={compactMode ? "展开模式" : "紧凑模式"}>{compactMode ? <List size={13} /> : <Square size={13} />}</ToolbarButton>
              <ToolbarButton onClick={toggleFocus} title={focusMode ? "退出专注模式 (Ctrl+Shift+F)" : "专注模式 (Ctrl+Shift+F)"}>
                <Aim size={13} className={focusMode ? "text-accent" : ""} />
              </ToolbarButton>
              <ToolbarButton onClick={() => downloadMarkdown(exportAsMarkdown(state.items))} disabled={state.items.length===0} title="导出 Markdown">导出</ToolbarButton>
              <ToolbarButton onClick={() => void exportConversation("docx")} disabled={state.items.length===0} title="导出 Word（统一交付出口）">导出 Word</ToolbarButton>
              {deleteConfirm ? (
                <span className="flex items-center gap-1 rounded-md border border-err/30 bg-del-bg px-1.5 py-1">
                  <span className="text-[11px] text-err whitespace-nowrap">删除当前会话？</span>
                  <button
                    className="inline-flex items-center justify-center w-5 h-5 border-0 rounded bg-transparent text-err cursor-pointer hover:bg-err/15"
                    onClick={() => void confirmDeleteCurrent()}
                    title="确认删除"
                  >
                    <Check size={12} />
                  </button>
                  <button
                    className="inline-flex items-center justify-center w-5 h-5 border-0 rounded bg-transparent text-fg-faint cursor-pointer hover:bg-bg-soft hover:text-fg"
                    onClick={() => setDeleteConfirm(false)}
                    title="取消"
                  >
                    <X size={12} />
                  </button>
                </span>
              ) : (
                <ToolbarButton onClick={() => setDeleteConfirm(true)} disabled={state.running} title="删除当前会话">
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
          <main className="main">
            <CompactContext.Provider value={compactMode}>
            {(state.meta?.ready === false && !state.meta?.startupErr) || switchingModel ? (
              <Skeleton />
            ) : (
              <>
                <Transcript onPrompt={send} running={state.running} onRewind={rewind} onScrollToTurnReady={setScrollToTurn} cwd={state.meta?.cwd} cwdName={cwdName} sessions={recentSessions} onResumeSession={resumeRecentSession} meta={state.meta} />
                {state.items.length > 1 && <JumpBar items={state.items} scrollToTurn={scrollToTurn ?? undefined} />}
              </>
            )}
            </CompactContext.Provider>
          </main>

          <footer className={`shrink-0 border-t border-border-soft bg-bg px-8 ${compactMode ? "pt-2 pb-0.5" : "pt-3 pb-1"}`}>
            <CompactContext.Provider value={compactMode}>
            {requirement?.text && (
              <GoalCard
                requirement={requirement}
                onToggleRequirementDone={toggleRequirementDone}
                onAddRequirementItem={handleAddRequirementItem}
                onSetRequirementItem={handleSetRequirementItem}
                onSetRequirementItemDone={handleSetRequirementItemDone}
                onRemoveRequirementItem={handleRemoveRequirementItem}
                onToggleRequirementAutoPursue={handleToggleRequirementAutoPursue}
              />
            )}
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
            />
            <div className="composer-glow">
            <Composer
              running={state.running}
              cwd={state.meta?.cwd}
              onSend={handleSend}
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
              title="拖拽调整预览宽度"
            />
            <div className="preview-pane">
              <FilePreview
                relPath={previewFile}
                onClose={closeFilePreview}
                onBackToFiles={backToFiles}
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
          <WorkspaceTabs active={rightTab} onChange={setRightTab} badges={runningBadge} />
          <div className="flex-1 min-h-0 overflow-y-auto">
            {rightTab === "files" ? (
              <WorkspacePanel
                cwd={state.meta?.cwd}
                selectedFile={previewFile ?? undefined}
                refreshKey={workspaceRefreshKey}
                onSelectFile={openFilePreview}
                onRefresh={() => setWorkspaceRefreshKey((k) => k + 1)}
                onClose={() => setWorkspacePanel(false)}
              />
            ) : null}
            {rightTab === "materials" && (
              <MaterialsPanel onOpenFile={openFilePreview} />
            )}
            {rightTab === "cost" && <CostLibraryPanel />}
            {rightTab === "stats" && (
              <StatsPanel
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
            {rightTab === "deliverables" && (
              <DeliverablesPanel
                items={sessionDeliverables}
                onOpenFile={openFilePreview}
                onLocateSource={(turn) => {
                  setWorkspacePanel(false);
                  scrollToTurn?.(turn);
                }}
              />
            )}
            {rightTab === "changes" && (
              <ChangesPanel
                changes={sessionChanges}
                cwd={state.meta?.cwd}
                onOpenFile={openFilePreview}
              />
            )}
            {rightTab === "tasks" && <TaskCenter />}
            {rightTab === "subagents" && (
              <SubagentsPanel sessionPath={currentSessionPath ?? undefined} />
            )}
          </div>
        </div>
        )}
      </div>
      </Layout>

      {state.approval && (
          <ApprovalModal
            approval={state.approval}
            onAnswer={(allow, session) => {
              approve(state.approval!.id, allow, session);
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
