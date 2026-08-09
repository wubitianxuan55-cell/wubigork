import { lazy, Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { CSSProperties } from "react";
import { Layout } from "antd";
import {
  BarChart3, BookOpen, Check, SquarePen, Brain, ChevronDown, Cpu, FolderGit2, FolderTree,
  PanelRightOpen, PanelRightClose, MessageSquare, Trash2, X,
} from "./icons";
import { Sidebar } from "./components/Sidebar";
import { useT } from "./lib/i18n";
import { sessionTitle, sessionTime } from "./lib/session";
import { useController } from "./lib/store";
import type { JobView } from "./lib/types";
import { app } from "./lib/bridge";
import { Transcript } from "./components/Transcript";
import { JumpBar } from "./components/JumpBar";
import { ToastProvider, useToast } from "./components/Toast";
import { Composer } from "./components/Composer";
import { TodoPanel } from "./components/TodoPanel";
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
import { FilePreview } from "./components/FilePreview";
import { FilePreviewModal } from "./components/FilePreviewModal";
import { CommandPalette, type PaletteItem } from "./components/CommandPalette";
import { StatsPanel, useStatsPersistence } from "./components/StatsPanel";
import { Skeleton } from "./components/Skeleton";
import { UpdateBanner } from "./components/UpdateBanner";

import { downloadMarkdown, exportAsMarkdown } from "./lib/export";
import type { MemorySuggestion, MemorySuggestionsView, MemoryView, SessionMeta, SkillSuggestion } from "./lib/types";
import { useTodoExtractor } from "./hooks/useTodoExtractor";
import { useModeManager } from "./hooks/useModeManager";
import { useSessionManager } from "./hooks/useSessionManager";
import { useBridgeWatch } from "./hooks/useBridgeWatch";
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
import { fmtTokens } from "./lib/stats";
import { useNow } from "./lib/useNow";

function NewSessionToast({ done }: { done: boolean }) {
  const toast = useToast();
  useEffect(() => { if (done) toast.show("新会话已创建", "info"); }, [done]);
  return null;
}

// ── JobDoneNotifier — 后台任务从运行列表消失即视为结束，弹 toast 提示 ──
function JobDoneNotifier({ jobs }: { jobs: JobView[] }) {
  const toast = useToast();
  const prevRef = useRef<Map<string, string>>(new Map()); // id -> label
  useEffect(() => {
    const prev = prevRef.current;
    const current = new Map(jobs.map((j) => [j.id, j.label] as const));
    for (const [id, label] of prev) {
      if (!current.has(id)) toast.show(`后台任务已完成：${label}`, "info");
    }
    prevRef.current = current;
  }, [jobs, toast]);
  return null;
}

// ── RunStatus — 输入框上方的运行时状态行 ─────────────────────

function RunStatus({ running, turnStartAt, turnTokens }: {
  running: boolean;
  turnStartAt: number;
  turnTokens: number;
}) {
  const now = useNow();
  if (!running) return null;
  const elapsed = turnStartAt > 0 ? Math.max(0, now - Math.floor(turnStartAt / 1000)) : 0;
  const elapsedStr = elapsed < 60 ? `${elapsed}s` : `${Math.floor(elapsed / 60)}m${elapsed % 60}s`;
  const tokStr = turnTokens > 0 ? `↓${fmtTokens(turnTokens)}` : "";
  return (
    <div className="flex items-center justify-between px-4 py-1.5 text-[11px] select-none border-b border-border-soft/50 bg-bg-soft/30">
      <div className="flex items-center gap-2 text-fg-dim tabular-nums font-mono">
        <span className="font-medium">{elapsedStr}</span>
        {tokStr && <span className="text-fg-faint">{tokStr}</span>}
      </div>
      <div className="flex items-center gap-3">
        <span className="flex items-center gap-1.5 text-fg">
          <Cpu size={12} className="text-cyan-400" />
          <span className="font-medium">执行中</span>
          <span className="inline-flex items-center gap-1 ml-0.5">
            <span className="w-1.5 h-1.5 rounded-full bg-cyan-400 animate-pulse" />
            <span className="text-[10px] text-cyan-400/70">中</span>
          </span>
        </span>
      </div>
    </div>
  );
}

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
    resumeSession,
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
  } = useController();
  const t = useT();
  const { permLevel, setPermLevel, switchingModel, switchModel } = useModeManager(ctrlSetPermLevel, setModel);
  const [memView, setMemView] = useState<MemoryView | null>(null);
  const [histView, setHistView] = useState<SessionMeta[] | null>(null);
  const { sidebarSessions, sidebarQuery, setSidebarQuery, newSessionDone, refreshSessions, startNewSession, loadMore, hasMore, handleResumeSession, handleDeleteSession, handleRenameSession } = useSessionManager(newSession, listSessions, resumeSession, deleteSession, renameSession);
  const newSessionAndReset = useCallback(async () => { setStatsReset(n => n + 1); await startNewSession(); }, [startNewSession]);
  const [statsReset, setStatsReset] = useState(0);
  const [capsOpen, setCapsOpen] = useState(false);
  const [knowledgeOpen, setKnowledgeOpen] = useState(false);
  const [rightTab, setRightTab] = useState<"files" | "stats">("files");
  const [compactMode, setCompactMode] = useState(() => { try { return localStorage.getItem("gaea.compactMode") === "1"; } catch { return false; } });
  const [scrollToTurn, setScrollToTurn] = useState<((turn: number) => void) | null>(null);
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState(false);

  const {
    sidebarCollapsed, sidebarWidth, sidebarResizing, effectiveSidebarWidth,
    toggleSidebar, setExpandedSidebarWidth, startSidebarResize,
    resizeSidebarWithKeyboard,
  } = useSidebar();

  const [workspacePanelOpen, setWorkspacePanel] = useState(false);
  const [previewFile, setPreviewFile] = useState<string | null>(null);
  const [previewWidth, setPreviewWidth] = useState(loadPreviewWidth);
  const [previewResizing, setPreviewResizing] = useState(false);
  const [workspaceRefreshKey, setWorkspaceRefreshKey] = useState(0);

  // 点文件 → 收起右侧树，在主区域展开可拖宽的预览（Codex 式）
  const openFilePreview = useCallback((rel: string) => {
    setRightTab("files");
    setWorkspacePanel(false);
    setPreviewFile(rel);
  }, []);

  // 预览头部“文件”按钮 → 回到文件树
  const backToFiles = useCallback(() => {
    setPreviewFile(null);
    setRightTab("files");
    setWorkspacePanel(true);
  }, []);

  // 面板开关：预览打开时先收起预览再展开树
  const toggleWorkspacePanel = useCallback(() => {
    if (previewFile !== null) {
      setPreviewFile(null);
      setWorkspacePanel(true);
      return;
    }
    setWorkspacePanel((o) => !o);
  }, [previewFile]);

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
  const { alive: bridgeAlive, onReconnect } = useBridgeWatch();
  useEffect(() => {
    onReconnect(() => { refreshMeta(); });
  }, [onReconnect, refreshMeta]);

  const { todoItem, todos, showTodos, setDismissedTodo } = useTodoExtractor(state.items);

  // Memory drawer: opening fetches a fresh snapshot; writes re-fetch so the
  // panel reflects what landed on disk.
  const openMemory = useCallback(async () => {
    setMemView(await fetchMemory());
  }, [fetchMemory]);

  const closeMemory = useCallback(() => setMemView(null), []);

  const openKnowledge = useCallback(() => setKnowledgeOpen(true), []);
  const closeKnowledge = useCallback(() => setKnowledgeOpen(false), []);

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
      const t = displayText.trim();
      const model = /^\/model\s+(\S+)$/.exec(t);
      if (model) {
        void switchModel(model[1]);
        return;
      }
      if (t === "/memory") {
        void openMemory();
        return;
      }
      send(t, submitText.trim());
    },
    [switchModel, openMemory, send],
  );

  // History drawer: opening fetches the saved-session list; picking one resumes it
  // (the transcript swaps in; the model/folder are unchanged).
  const openHistory = useCallback(async () => {
    setHistView(await refreshSessions());
  }, [refreshSessions]);
  const closeHistory = useCallback(() => setHistView(null), []);
  const onResumeSession = useCallback(
    async (path: string) => { setHistView(null); await handleResumeSession(path); },
    [handleResumeSession],
  );
  const onDeleteSession = useCallback(
    async (path: string) => { await handleDeleteSession(path); setHistView(await refreshSessions()); },
    [handleDeleteSession, refreshSessions],
  );
  const onRenameSession = useCallback(
    async (path: string, title: string) => { await handleRenameSession(path, title); setHistView(await refreshSessions()); },
    [handleRenameSession, refreshSessions],
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
      setPreviewFile(null);
      setWorkspacePanel(false);
      await refreshSessions();
    }
    return picked;
  }, [pickWorkspace, switchWorkspace, refreshSessions]);

  const onRemember = useCallback(
    async (scope: string, note: string) => {
      await remember(scope, note);
      setMemView(await fetchMemory());
    },
    [remember, fetchMemory],
  );

  const onForget = useCallback(
    async (name: string) => {
      await forget(name);
      setMemView(await fetchMemory());
    },
    [forget, fetchMemory],
  );

  const onSaveDoc = useCallback(
    async (path: string, body: string) => {
      await saveDoc(path, body);
      setMemView(await fetchMemory());
    },
    [saveDoc, fetchMemory],
  );

  const onSaveFact = useCallback(
    async (name: string, body: string) => {
      await updateFact(name, body);
      setMemView(await fetchMemory());
    },
    [updateFact, fetchMemory],
  );

  const onAcceptMemorySuggestion = useCallback(
    async (candidate: MemorySuggestion) => {
      await app.AcceptMemorySuggestion(candidate);
      setMemView(await fetchMemory());
    },
    [fetchMemory],
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
        if (previewFile !== null) { ke.preventDefault(); setPreviewFile(null); return; }
        if (capsOpen) { ke.preventDefault(); setCapsOpen(false); return; }
        if (memView !== null) { ke.preventDefault(); setMemView(null); return; }
        if (histView !== null) { ke.preventDefault(); setHistView(null); return; }
        if (knowledgeOpen) { ke.preventDefault(); setKnowledgeOpen(false); return; }
      }
      if (!mod) return;
      if (ke.key === "n" && !state.running) { ke.preventDefault(); void newSessionAndReset(); return; }
      if (ke.key === "k") { ke.preventDefault(); setPaletteOpen(true); return; }
      if (ke.key === "H" && ke.shiftKey) { ke.preventDefault(); void openHistory(); return; }
      if (ke.key === "K" && ke.shiftKey) { ke.preventDefault(); void openKnowledge(); return; }
      if (ke.key === "H" && ke.shiftKey) { ke.preventDefault(); void openHistory(); return; }
      if (ke.key === "b") { ke.preventDefault(); toggleSidebar(); return; }
      if (ke.key === "j") { ke.preventDefault(); toggleWorkspacePanel(); return; }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [state.running, capsOpen, memView, histView, knowledgeOpen, workspacePanelOpen, previewFile]);

  const { toolCounts, skillCounts } = useToolStats(state.items);

  // 当前会话标识：直接使用 Go 后端生成的 .jsonl 文件路径作为 key。
  // 每个会话文件对应唯一的 localStorage key：新会话自然空数据开始，
  // 恢复/重启同一会话则统计数据持续累加，会话之间互不干扰。
  const currentSessionPath = useMemo(
    () => sidebarSessions.find(s => s.current)?.path,
    [sidebarSessions],
  );
  const currentSessionKey = useMemo(() => {
    return currentSessionPath
      ? currentSessionPath.replace(/[\\/:*?"<>|]/g, "_")
      : cwd ? `unsaved_${cwd.replace(/[\\/:*?"<>|]/g, "_")}` : "unsaved";
  }, [currentSessionPath, cwd]);

  const statsPersistence = useStatsPersistence(currentSessionKey, statsReset, state.turnSteps, state.perTurnUsage);

  const paletteItems = useMemo<PaletteItem[]>(() => {
    const cmds: PaletteItem[] = [
      { id: "cmd-new", group: t("palette.group.commands") ?? "命令", title: t("topbar.newSession") ?? "新建会话", icon: <SquarePen size={15} />, compact: true, keywords: ["new", "新建"], run: () => void newSessionAndReset() },
      { id: "cmd-memory", group: t("palette.group.commands") ?? "命令", title: t("topbar.memory") ?? "记忆", icon: <Brain size={15} />, compact: true, keywords: ["memory", "记忆"], run: () => void openMemory() },
      { id: "cmd-history", group: t("palette.group.commands") ?? "命令", title: t("topbar.history") ?? "历史", icon: <MessageSquare size={15} />, compact: true, keywords: ["history", "历史"], run: () => void openHistory() },
      { id: "cmd-knowledge", group: t("palette.group.commands") ?? "命令", title: t("topbar.knowledge") ?? "知识库", icon: <BookOpen size={15} />, compact: true, keywords: ["knowledge", "知识库"], run: () => void openKnowledge() },
      { id: "cmd-files", group: t("palette.group.commands") ?? "命令", title: "文件面板", icon: <FolderGit2 size={15} />, compact: true, keywords: ["files", "文件"], run: () => { setPreviewFile(null); setWorkspacePanel(true); setRightTab("files"); } },
      { id: "cmd-stats", group: t("palette.group.commands") ?? "命令", title: "统计面板", icon: <BarChart3 size={15} />, compact: true, keywords: ["stats", "统计"], run: () => { setWorkspacePanel(true); setRightTab("stats"); } },
    ];
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
    return [...cmds, ...sessionItems];
  }, [t, sidebarSessions, startNewSession, openMemory, openHistory, openKnowledge, onResumeSession, setWorkspacePanel, setPreviewFile]);

  const layoutStyle = useMemo(
    () =>
      ({
        "--sidebar-expanded-width": `${sidebarWidth}px`,
        "--preview-width": `${previewWidth}px`,
      }) as CSSProperties,
    [sidebarWidth, previewWidth],
  );

  return (
    <ToastProvider>
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
          sessions={sidebarSessions}
          searchQuery={sidebarQuery}
          onSearchChange={setSidebarQuery}
          hasMore={hasMore}
          onLoadMore={loadMore}
          onResumeSession={onResumeSession}
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

        <section className="chat-pane">
          <header className="flex flex-shrink-0 items-center gap-3 px-12 border-b border-border-soft select-none drag-region transition-all duration-200" style={{background: "var(--ds-gradient-topbar)", boxShadow: "var(--ds-shadow-topbar)"}}>
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
              <ToolbarButton onClick={() => { const v = !compactMode; setCompactMode(v); try { localStorage.setItem("gaea.compactMode", v ? "1" : "0"); } catch {} }} title={compactMode ? "展开模式" : "紧凑模式"}>{compactMode ? "⊞" : "⊟"}</ToolbarButton>
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
                <Transcript onPrompt={send} running={state.running} onScrollToTurnReady={setScrollToTurn} cwd={state.meta?.cwd} cwdName={cwdName} sessions={sidebarSessions} onResumeSession={handleResumeSession} meta={state.meta} />
                {state.items.length > 1 && <JumpBar items={state.items} scrollToTurn={scrollToTurn ?? undefined} />}
              </>
            )}
            </CompactContext.Provider>
          </main>

          <footer className={`shrink-0 border-t border-border-soft bg-bg px-8 ${compactMode ? "pt-2 pb-0.5" : "pt-3 pb-1"}`}>
            <CompactContext.Provider value={compactMode}>
            {showTodos && <TodoPanel todos={todos} onDismiss={() => setDismissedTodo(todoItem!.id)} />}
            <RunStatus
              running={state.running}
              turnStartAt={state.turnStartAt}
              turnTokens={state.turnTokens}
            />
            <div className="composer-glow">
            <Composer
              running={state.running}
              cwd={state.meta?.cwd}
              onSend={handleSend}
              onCancel={cancel}
              permLevel={permLevel}
              onSetPermLevel={setPermLevel}
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
                onClose={() => setPreviewFile(null)}
                onBackToFiles={backToFiles}
              />
            </div>
          </>
        )}

        {workspacePanelOpen && (
        <div className="workspace-pane flex flex-col min-w-0 overflow-hidden border-l border-border-soft bg-bg transition-all duration-200">
          <div className="flex items-center border-b border-border-soft overflow-hidden shrink">
            <button
              className={`flex items-center gap-1 px-3 py-2 text-xs bg-transparent border-0 border-b-2 cursor-pointer transition-[color,border-color] duration-[var(--dur-base)] hover:text-fg text-fg-dim border-transparent ${rightTab === "files" ? "text-accent border-accent" : ""}`}
              onClick={() => setRightTab("files")}
            >
              <FolderTree size={13} />
              <span>文件</span>
            </button>
            <button
              className={`flex items-center gap-1 px-3 py-2 text-xs bg-transparent border-0 border-b-2 cursor-pointer transition-[color,border-color] duration-[var(--dur-base)] hover:text-fg text-fg-dim border-transparent ${rightTab === "stats" ? "text-accent border-accent" : ""}`}
              onClick={() => setRightTab("stats")}
            >
              <BarChart3 size={13} />
              <span>统计</span>
            </button>
          </div>
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
            {rightTab === "stats" && (
              <StatsPanel
                data={statsPersistence.data}
                clearData={statsPersistence.clearData}
                perTurnExecutorUsage={state.perTurnExecutorUsage}
                perTurnSubUsage={state.perTurnSubUsage}
                turnSteps={state.turnSteps}
                subagentModel={state.meta?.subagentLabel}
                toolCounts={toolCounts}
                skillCounts={skillCounts}
              />
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

      <FilePreviewModal />
    </ToastProvider>
  );
}
