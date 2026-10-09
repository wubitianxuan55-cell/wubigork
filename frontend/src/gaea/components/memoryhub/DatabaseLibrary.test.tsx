import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DatabaseLibrary } from "./DatabaseLibrary";
import type { DatabaseOverview } from "../../lib/types";
import type { ProjectGroup } from "../../lib/types/session";

vi.setConfig({ testTimeout: 15_000 });

const mocks = vi.hoisted(() => ({
  databaseOverview: vi.fn(),
  listProjectSessions: vi.fn(),
  uploadsList: vi.fn(),
  deleteUpload: vi.fn(),
  deleteSession: vi.fn(),
  reveal: vi.fn(),
}));

vi.mock("../../lib/bridge", () => ({
  app: new Proxy({}, {
    get(_t, prop) {
      if (prop === "DatabaseOverview") return mocks.databaseOverview;
      if (prop === "ListProjectSessions") return mocks.listProjectSessions;
      if (prop === "UploadsList") return mocks.uploadsList;
      if (prop === "DeleteUpload") return mocks.deleteUpload;
      if (prop === "DeleteSession") return mocks.deleteSession;
      if (prop === "RevealWorkspacePath") return mocks.reveal;
      return vi.fn().mockResolvedValue(undefined);
    },
  }),
}));

const overview: DatabaseOverview = {
  projects: [{
    path: "C:/AI/demo", name: "demo", current: true,
    sessionCount: 2, archivedCount: 1, sessionBytes: 180,
    uploadCount: 2, uploadBytes: 60, latestMod: Date.now(),
  }],
  totalSessions: 2, totalArchived: 1, totalUploads: 2, totalBytes: 240,
};

const groups: ProjectGroup[] = [{
  path: "C:/AI/demo", name: "demo", current: true,
  sessions: [
    { path: "C:/AI/demo/.gaea/sessions/work/a.jsonl", preview: "调节池算量会话", title: "", turns: 5, modTime: 1760000000000, current: false },
    { path: "C:/AI/demo/.gaea/sessions/work/b.jsonl", preview: "清单导入会话", title: "清单导入", turns: 2, modTime: 1759000000000, current: false },
  ],
  archived: [
    { path: "C:/AI/demo/.gaea/sessions/work/archive/old.jsonl", preview: "旧会话", title: "", turns: 1, modTime: 1750000000000, current: false, archived: true },
  ],
  modTime: 1760000000000,
}];

describe("DatabaseLibrary 记忆中枢数据库存档库（v4.479）", () => {
  beforeEach(() => {
    mocks.databaseOverview.mockReset().mockResolvedValue(overview);
    mocks.listProjectSessions.mockReset().mockResolvedValue(groups);
    mocks.uploadsList.mockReset().mockResolvedValue([
      { name: "attach-1-report.pdf", path: "C:/AI/demo/.gaea/uploads/attach-1-report.pdf", size: 40, modTime: 1760000000000, kind: "attach" },
      { name: "paste-2.png", path: "C:/AI/demo/.gaea/uploads/paste-2.png", size: 20, modTime: 1759000000000, kind: "paste" },
    ]);
    mocks.deleteUpload.mockReset().mockResolvedValue(undefined);
    mocks.deleteSession.mockReset().mockResolvedValue(undefined);
    mocks.reveal.mockReset().mockResolvedValue(undefined);
  });

  it("总览统计带与项目行：会话/归档/占用真实计数", async () => {
    render(<DatabaseLibrary />);
    const stats = await screen.findByTestId("db-stats");
    expect(stats.textContent).toContain("1"); // 工作区
    expect(stats.textContent).toContain("2"); // 会话
    expect(stats.textContent).toContain("1"); // 归档
    expect(stats.textContent).toContain("240 B"); // 占用
    // 项目行折叠可见
    expect(screen.getByTestId("db-project").textContent).toContain("demo");
    expect(screen.getByTestId("db-project").textContent).toContain("2 会话");
  });

  it("展开项目可见会话行；两段确认删除调 DeleteSession 并重载", async () => {
    render(<DatabaseLibrary />);
    await screen.findByTestId("db-stats");
    // 原生 details 内容常驻 DOM（v4.477 先例），直接断言会话行
    expect(screen.getByText("调节池算量会话")).toBeTruthy();
    expect(screen.getByText("清单导入")).toBeTruthy();
    // 两段确认：首点变确认态，再点执行
    const delBtns = screen.getAllByTestId("db-session-delete");
    fireEvent.click(delBtns[0]);
    expect(delBtns[0].textContent).toContain("确认删除");
    fireEvent.click(delBtns[0]);
    await waitFor(() => expect(mocks.deleteSession).toHaveBeenCalledWith(groups[0].sessions[0].path));
    // 删除后重载
    await waitFor(() => expect(mocks.databaseOverview).toHaveBeenCalledTimes(2));
  });

  it("上传文件展开懒加载；两段确认删除调 DeleteUpload", async () => {
    render(<DatabaseLibrary />);
    await screen.findByTestId("db-stats");
    expect(mocks.uploadsList).not.toHaveBeenCalled(); // 默认不拉取
    const details = screen.getByTestId("db-uploads") as HTMLDetailsElement;
    details.open = true;
    act(() => { details.dispatchEvent(new Event("toggle")); });
    await waitFor(() => expect(mocks.uploadsList).toHaveBeenCalledWith("C:/AI/demo"));
    expect(await screen.findByText("attach-1-report.pdf")).toBeTruthy();
    expect(screen.getByText(/粘贴图/)).toBeTruthy();
    const delBtn = screen.getAllByTestId("db-upload-delete")[0];
    fireEvent.click(delBtn);
    fireEvent.click(delBtn);
    await waitFor(() => expect(mocks.deleteUpload).toHaveBeenCalledWith("C:/AI/demo/.gaea/uploads/attach-1-report.pdf"));
  });

  it("删除失败可见化：错误横幅显示且可关闭", async () => {
    mocks.deleteSession.mockRejectedValue(new Error("当前会话不可删除"));
    render(<DatabaseLibrary />);
    await screen.findByTestId("db-stats");
    const delBtns = screen.getAllByTestId("db-session-delete");
    fireEvent.click(delBtns[0]);
    fireEvent.click(delBtns[0]);
    const banner = await screen.findByTestId("db-error");
    expect(banner.textContent).toContain("会话删除失败");
    // 关闭钮可横幅移除
    fireEvent.click(screen.getByRole("button", { name: "关闭错误提示" }));
    expect(screen.queryByTestId("db-error")).toBeNull();
  });

  it("统计带占位：错误态显「—」而非加载态「…」", async () => {
    mocks.databaseOverview.mockRejectedValue(new Error("绑定不可达"));
    mocks.listProjectSessions.mockRejectedValue(new Error("绑定不可达"));
    render(<DatabaseLibrary />);
    const stats = await screen.findByTestId("db-stats");
    await screen.findByTestId("db-error");
    expect(stats.textContent).toContain("—");
  });

  it("项目超大被侧栏上限截断时如实提示（总览计数 > 行数）", async () => {
    // 总览报 60 条，列表只回 2 条 = 截断
    const big: DatabaseOverview = {
      ...overview,
      projects: [{ ...overview.projects[0], sessionCount: 59, archivedCount: 1 }],
      totalSessions: 59,
    };
    mocks.databaseOverview.mockResolvedValue(big);
    render(<DatabaseLibrary />);
    await screen.findByTestId("db-stats");
    expect(await screen.findByTestId("db-capped-note")).toBeTruthy();
  });
});
