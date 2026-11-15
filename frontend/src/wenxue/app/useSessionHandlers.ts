// useSessionHandlers — 会话/历史/记忆操作回调集（Kun/Codex 优点蒸馏）。
// 从 App.tsx 按原声明顺序迁出：历史抽屉打开/恢复/删除/重命名、删除当前会话、
// 工作区切换、跨项目恢复、置顶/归档/恢复、记忆写操作与建议接受。全部为
// useCallback/useMemo，无 effect，deps 数组原样保留。
// 7.2-2 追加：流程蒸馏（journal 历史蒸馏）的视图状态与四组 handler——本文件
// 唯一的 useState（distill 视图/加载态自包含于此，MemoryPanel 经可选 props
// 消费，未接线调用方零感知）。
import { useCallback, useMemo, useState } from "react";
import { app } from "../lib/bridge";
import type { Translator } from "../lib/i18n";
import type {
  MemorySuggestion, MemorySuggestionsView, MemoryView, ProjectGroup,
  SessionMeta, SkillDistillView, SkillRecordResult, SkillSuggestion,
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

  // ── 流程蒸馏（阶段七 7.2-2：journal 历史挖掘 → 技能结晶）──────────────
  // 契约：app.SkillDistillCandidates / SkillDistillDraft / SkillDistillDecide
  // （主代理收口在 lib/bridge 统一补齐；收口前 import 处 tsc 必红，属预期）。

  // distill 视图 + 加载态（refresh 由容器在记忆面板打开/点刷新时触发）。
  const [distillView, setDistillView] = useState<SkillDistillView | null>(null);
  const [distillLoading, setDistillLoading] = useState(false);

  // refreshDistill 复算候选（纯确定性挖掘，零 LLM）；失败静默容错（onRefreshSuggestions 同款先例）。
  const refreshDistill = useCallback(async (): Promise<SkillDistillView | null> => {
    setDistillLoading(true);
    try {
      const v = await app.SkillDistillCandidates();
      setDistillView(v);
      return v;
    } catch (e) {
      console.warn("[gaea-distill] candidates failed:", e);
      return null;
    } finally {
      setDistillLoading(false);
    }
  }, []);

  // draftDistill 对一条候选生成 LLM 蒸馏草稿（只回不落盘）。调用方
  // （MemoryPanel 容器层）拿到结果后以 SkillRecordModal preload 打开审阅。
  const draftDistill = useCallback(async (patternId: string): Promise<SkillRecordResult | null> => {
    try {
      return await app.SkillDistillDraft(patternId);
    } catch (e) {
      toast.show(String((e as Error)?.message ?? e), "warn");
      return null;
    }
  }, [toast]);

  // ignoreDistill「不再提示」：Decide(id,"ignore","") 落状态 → 复算即静默跳过。
  const ignoreDistill = useCallback(
    async (patternId: string) => {
      try {
        await app.SkillDistillDecide(patternId, "ignore", "");
      } catch (e) {
        toast.show(String((e as Error)?.message ?? e), "warn");
        return;
      }
      await refreshDistill();
    },
    [refreshDistill, toast],
  );

  // crystallizeDistill 结晶审计：审阅弹窗保存成功后回调（skillName=落盘技能
  // 名），Decide(id,"crystallized",skillName) 记录哪些会话、何时、结晶成哪个
  // 技能，随后刷新候选（该模式不再出现）。
  const crystallizeDistill = useCallback(
    async (patternId: string, skillName: string) => {
      try {
        await app.SkillDistillDecide(patternId, "crystallized", skillName);
      } catch (e) {
        // 审计落账失败不回滚已保存技能（技能本体已生效），告警即可。
        console.warn("[gaea-distill] crystallized decide failed:", e);
      }
      await refreshDistill();
    },
    [refreshDistill],
  );

  return {
    openHistory, closeHistory, onResumeSession, onDeleteSession, onRenameSession,
    confirmDeleteCurrent, switchFolder, resumeSessionInProject, recentSessions,
    resumeRecentSession, onArchiveSession, onPinSession, onRestoreSession,
    onRemember, onForget, onSaveDoc, onSaveFact, onAcceptMemorySuggestion,
    onAcceptSkillSuggestion, onAcceptMergeSuggestion, onRefreshSuggestions,
    distillView, distillLoading, refreshDistill, draftDistill, ignoreDistill,
    crystallizeDistill,
  };
}