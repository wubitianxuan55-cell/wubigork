import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useSessionManager } from "./useSessionManager";
import type { ProjectGroup, SessionMeta } from "../lib/types";

function meta(path: string, overrides: Partial<SessionMeta> = {}): SessionMeta {
  return { path, preview: `preview-${path}`, turns: 1, modTime: 1, current: false, ...overrides };
}

function makeDeps(overrides: Partial<{
  sessions: SessionMeta[];
  groups: ProjectGroup[];
}> = {}) {
  const newSession = vi.fn().mockResolvedValue(undefined);
  const listSessions = vi.fn().mockResolvedValue(overrides.sessions ?? [
    meta("/cur", { current: true }),
    meta("/old"),
  ]);
  const listProjectSessions = vi.fn().mockResolvedValue(overrides.groups ?? []);
  const resumeSession = vi.fn().mockResolvedValue(undefined);
  const deleteSession = vi.fn().mockResolvedValue(undefined);
  const renameSession = vi.fn().mockResolvedValue(undefined);
  const onError = vi.fn();
  return { newSession, listSessions, listProjectSessions, resumeSession, deleteSession, renameSession, onError };
}

describe("useSessionManager", () => {
  it("refreshSessions 同步拉取会话与项目分组", async () => {
    const d = makeDeps({
      groups: [{ path: "/ws", name: "ws", current: true, sessions: [], archived: [], modTime: 1 }],
    });
    const { result } = renderHook(() => useSessionManager(d.newSession, d.listSessions, d.listProjectSessions, d.resumeSession, d.deleteSession, d.renameSession, d.onError));
    await act(async () => { await result.current.refreshSessions(); });
    expect(result.current.sidebarSessions.map((s) => s.path)).toEqual(["/cur", "/old"]);
    expect(result.current.projectGroups).toHaveLength(1);
    expect(result.current.hasMore).toBe(false);
  });

  it("恢复会话失败时回调 onError", async () => {
    const d = makeDeps();
    d.resumeSession.mockRejectedValue(new Error("corrupt session"));
    const { result } = renderHook(() => useSessionManager(d.newSession, d.listSessions, d.listProjectSessions, d.resumeSession, d.deleteSession, d.renameSession, d.onError));
    await act(async () => { await result.current.handleResumeSession("/bad"); });
    expect(d.onError).toHaveBeenCalledWith("恢复会话失败：corrupt session");
  });

  it("重命名失败时回调 onError", async () => {
    const d = makeDeps();
    d.renameSession.mockRejectedValue(new Error("locked"));
    const { result } = renderHook(() => useSessionManager(d.newSession, d.listSessions, d.listProjectSessions, d.resumeSession, d.deleteSession, d.renameSession, d.onError));
    await act(async () => { await result.current.handleRenameSession("/cur", "x"); });
    expect(d.onError).toHaveBeenCalledWith("重命名会话失败：locked");
  });

  it("删除会话后乐观移除本地列表", async () => {
    const d = makeDeps();
    const { result } = renderHook(() => useSessionManager(d.newSession, d.listSessions, d.listProjectSessions, d.resumeSession, d.deleteSession, d.renameSession, d.onError));
    await act(async () => { await result.current.refreshSessions(); });
    await act(async () => { await result.current.handleDeleteSession("/old"); });
    expect(result.current.sidebarSessions.map((s) => s.path)).toEqual(["/cur"]);
    expect(d.onError).not.toHaveBeenCalled();
  });
});
