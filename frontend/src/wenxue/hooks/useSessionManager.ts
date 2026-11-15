// 会话管理 hook — 列表刷新 + CRUD + 分页 + 侧边栏状态
import { useState, useCallback, useRef } from "react";
import type { ProjectGroup, SessionMeta } from "../lib/types";

const PAGE_SIZE = 10;

export function useSessionManager(
  newSession: () => Promise<void>,
  listSessions: () => Promise<SessionMeta[]>,
  listProjectSessions: () => Promise<ProjectGroup[]>,
  resumeSession: (path: string) => Promise<void>,
  deleteSession: (path: string) => Promise<void>,
  renameSession: (path: string, title: string) => Promise<void>,
  onError?: (message: string) => void,
) {
  const [sidebarSessions, setSidebarSessions] = useState<SessionMeta[]>([]);
  const [sidebarQuery, setSidebarQuery] = useState("");
  const [newSessionDone, setNewSessionDone] = useState(false);
  const [hasMore, setHasMore] = useState(false);
  const [projectGroups, setProjectGroups] = useState<ProjectGroup[]>([]);
  // 缓存全量列表，loadMore 不再重复请求
  const allSessionsRef = useRef<SessionMeta[]>([]);

  const refreshProjectGroups = useCallback(async () => {
    const groups = await listProjectSessions();
    setProjectGroups(groups);
    return groups;
  }, [listProjectSessions]);

  const errText = useCallback((e: unknown) => (e instanceof Error ? e.message : String(e)), []);

  const refreshSessions = useCallback(async () => {
    const sessions = await listSessions();
    allSessionsRef.current = sessions;
    setHasMore(sessions.length > PAGE_SIZE);
    setSidebarSessions(sessions.slice(0, PAGE_SIZE));
    // 项目分组与平铺列表同源刷新，保证侧边栏「项目」视图与旧入口一致
    await refreshProjectGroups();
    return sessions;
  }, [listSessions, refreshProjectGroups]);

  const loadMore = useCallback(() => {
    const all = allSessionsRef.current;
    const next = all.slice(0, sidebarSessions.length + PAGE_SIZE);
    setHasMore(next.length < all.length);
    setSidebarSessions(next);
  }, [sidebarSessions.length]);

  const startNewSession = useCallback(async () => {
    await newSession();
    setSidebarQuery("");
    await refreshSessions();
    setNewSessionDone(true);
    setTimeout(() => setNewSessionDone(false), 2000);
  }, [newSession, refreshSessions]);

  const handleResumeSession = useCallback(
    async (path: string) => {
      try {
        await resumeSession(path);
        await refreshSessions();
      } catch (e) {
        onError?.(`恢复会话失败：${errText(e)}`);
      }
    },
    [resumeSession, refreshSessions, onError, errText],
  );

  const handleDeleteSession = useCallback(
    async (path: string) => {
      try {
        await deleteSession(path);
      } catch (e) {
        // 删除失败→重新拉取列表恢复正确状态
        await refreshSessions();
        onError?.(`删除会话失败：${errText(e)}`);
        return;
      }
      // 乐观更新缓存，避免删除后列表闪烁
      allSessionsRef.current = allSessionsRef.current.filter(s => s.path !== path);
      const visible = allSessionsRef.current.slice(0, sidebarSessions.length);
      setHasMore(visible.length < allSessionsRef.current.length);
      setSidebarSessions(visible);
      void refreshProjectGroups();
    },
    [deleteSession, refreshSessions, sidebarSessions.length, refreshProjectGroups, onError, errText],
  );

  const handleRenameSession = useCallback(
    async (path: string, title: string) => {
      try {
        await renameSession(path, title);
        await refreshSessions();
      } catch (e) {
        onError?.(`重命名会话失败：${errText(e)}`);
      }
    },
    [renameSession, refreshSessions, onError, errText],
  );

  return {
    sidebarSessions, sidebarQuery, setSidebarQuery,
    newSessionDone, refreshSessions, startNewSession, loadMore,
    hasMore,
    projectGroups, refreshProjectGroups,
    handleResumeSession, handleDeleteSession, handleRenameSession,
  };
}
