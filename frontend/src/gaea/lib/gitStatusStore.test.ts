import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({ status: vi.fn() }));
vi.mock("./bridge", () => ({
  app: { GaeaGitStatus: mocks.status },
}));

import {
  acquireGitStatus,
  refreshGitStatus,
  resetGitStatusStoreForTest,
  useGitStatusStore,
} from "./gitStatusStore";

describe("gitStatusStore", () => {
  beforeEach(() => {
    resetGitStatusStoreForTest();
    vi.clearAllMocks();
    mocks.status.mockResolvedValue({ isRepo: true, branch: "main", files: [] });
  });

  it("shares one status request across subscribers", async () => {
    const releaseA = acquireGitStatus();
    const releaseB = acquireGitStatus();
    await refreshGitStatus();
    expect(mocks.status).toHaveBeenCalledTimes(1);
    expect(useGitStatusStore.getState().status?.branch).toBe("main");
    releaseA();
    expect(useGitStatusStore.getState().status).not.toBeNull();
    releaseB();
    expect(useGitStatusStore.getState().status).toBeNull();
  });

  it("stores honest non-repository errors as a snapshot", async () => {
    mocks.status.mockResolvedValue({ isRepo: false, files: [], error: "当前工作区不是 Git 仓库" });
    const release = acquireGitStatus();
    await refreshGitStatus();
    expect(useGitStatusStore.getState().status?.isRepo).toBe(false);
    expect(useGitStatusStore.getState().error).toContain("不是 Git 仓库");
    release();
  });
});
