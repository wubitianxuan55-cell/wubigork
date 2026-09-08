// useSessionHandlers — 会话/历史/记忆操作回调集（Kun/Codex 优点蒸馏）。
// 从 App.tsx 按原声明顺序迁出：历史抽屉打开/恢复/删除/重命名、删除当前会话、
// 工作区切换、跨项目恢复、置顶/归档/恢复、记忆写操作与建议接受。全部为
// useCallback/useMemo，无 effect，deps 数组原样保留。
import { useCallback, useMemo } from "react";
import { app } from "../lib/bridge";
import type { Translator } from "../lib/i18n";
import type {
  MemorySuggestion, MemorySuggestionsView, MemoryView, ProjectGroup,
  SessionMeta, SkillSuggestion,
} from "../lib/types";
import { purgeDeletedSessionGenui } from "./sessionCleanup";

interface UseSessionHandlersParams {
  t: Translator;
  toast: { show: (text: string, kind?: "info" | "warn" | "error") => void };
  refreshSessions: () => Promise<SessionMeta[]>;
  deleteSession: (path: string) => Promise<void>;
  newSessionAndReset: () => Promise<void>;
  pickWorkspace: () => Promise<string>;
  switchWorkspace: (path: string) => Promise<string>;
  handleResumeSession: (path: string) => Promise<void>;
  handleDeleteSession: (path: string) => Promise<void>;
  handleRenameSession: (path: string, title: string) => Promise<void>;
  projectGroups: ProjectGroup[];
  archiveSession: (path: string) => Promise<void>;
  unarchiveSession: (path: string) => Promise<string>;
  pinSession: (path: string, pinned: boolean) => Promise<void>;
  remember: (scope: string, note: string) => Promise<void>;
  forget: (name: string) => Promise<void>;
  saveDoc: (path: string, body: string) => Promise<void>;
  updateFact: (name: string, body: string) => Promise<void>;
  fetchMemory: () => Promise<MemoryView>;
  setMemView: (v: MemoryView | null) => void;
  setHistView: (v: SessionMeta[] | null) => void;
  setDeleteConfirm: (v: boolean) => void;
  closeFilePreview: () => void;
  setWorkspacePanel: (v: boolean) => void;
}

export function useSessionHandlers(params: UseSessionHandlersParams) {
  const {
    t, toast, refreshSessions, deleteSession, newSessionAndReset, pickWorkspace,
    switchWorkspace, handleResumeSession, handleDeleteSession, handleRenameSession,
    projectGroups, archiveSession, unarchiveSession, pinSession, remember, forget,
    saveDoc, updateFact, fetchMemory, setMemView, setHistView, setDeleteConfirm,
    closeFilePreview, setWorkspacePanel,
  } = params;

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
    async (path: string) => {
      await handleDeleteSession(path);
      const sessions = await refreshSessions();
      setHistView(sessions);
      // 审计 2026-09 #7：列表中已无该会话（删除成功）才清理其 GenUI 交互状态。
      // handleDeleteSession 内部吞错（toast + 列表回滚），据此区分成败。
      if (!sessions.some((s) => s.path === path)) purgeDeletedSessionGenui(path);
    },
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
      if (cur) {
        await deleteSession(cur.path);
        // 审计 2026-09 #7：删除成功路径清理该会话的 GenUI 交互状态与面板内容。
        purgeDeletedSessionGenui(cur.path);
      }
    } catch {
      // 删除失败不阻塞新建会话
    }
    await newSessionAndReset();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- setDeleteConfirm 为 React dispatch（稳定引用），原 deps 语义保留
  }, [refreshSessions, deleteSession, newSessionAndReset]);

  // Workspace: open the folder chooser and switch projects. The hook resets the
  // transcript and refreshes meta on a pick; a cancel is a no-op.
  const switchFolder = useCallback(async (path?: string) => {
    const picked = path === undefined ? await pickWorkspace() : await switchWorkspace(path);
    if (picked) {
      closeFilePreview();
      setWorkspacePanel(false);
      await refreshSessions();
    }
    return picked;
    // eslint-disable-next-line react-hooks/exhaustive-deps -- setWorkspacePanel 为 React dispatch（稳定引用），原 deps 语义保留
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

  return {
    openHistory, closeHistory, onResumeSession, onDeleteSession, onRenameSession,
    confirmDeleteCurrent, switchFolder, resumeSessionInProject, recentSessions,
    resumeRecentSession, onArchiveSession, onPinSession, onRestoreSession,
    onRemember, onForget, onSaveDoc, onSaveFact, onAcceptMemorySuggestion,
    onAcceptSkillSuggestion, onAcceptMergeSuggestion, onRefreshSuggestions,
  };
}