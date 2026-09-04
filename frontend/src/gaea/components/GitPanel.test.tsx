import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { LocaleProvider } from "../lib/i18n";
import { GitPanel } from "./GitPanel";
import type { GitCommitInfoView, GitStatusView } from "../lib/types";

// GitPanel 走中文文案（工作台面板语言），断言不依赖 LocaleProvider 仍包一层防 useT。
// bridge 全量 mock：Git 面板唯一数据面是 GaeaGit*。
const mocks = vi.hoisted(() => ({
  status: vi.fn(),
  diff: vi.fn(async (_p: string, _s: boolean) => ""),
  stage: vi.fn(async (_p: string[]) => {}),
  unstage: vi.fn(async (_p: string[]) => {}),
  discard: vi.fn(async (_p: string) => {}),
  commit: vi.fn(async (_m: string) => "abc1234"),
  log: vi.fn(async (_n: number): Promise<GitCommitInfoView[]> => []),
}));

vi.mock("../lib/bridge", () => ({
  app: {
    GaeaGitStatus: mocks.status,
    GaeaGitDiff: mocks.diff,
    GaeaGitStage: mocks.stage,
    GaeaGitUnstage: mocks.unstage,
    GaeaGitDiscard: mocks.discard,
    GaeaGitCommit: mocks.commit,
    GaeaGitLog: mocks.log,
  },
  onEvent: () => () => {},
}));

const wrap = (ui: React.ReactNode) => render(<LocaleProvider>{ui}</LocaleProvider>);

const repoStatus: GitStatusView = {
  isRepo: true,
  branch: "main",
  ahead: 1,
  files: [
    { path: "docs/报告.md", x: "M", y: " ", staged: true, modified: true },
    { path: "src/main.go", x: " ", y: "M", modified: true },
    { path: "新文件.txt", x: "?", y: "?", untracked: true },
  ],
};

describe("GitPanel 2b 最小集", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.status.mockResolvedValue(repoStatus);
  });

  it("渲染分支名与三组文件行（已暂存/未暂存/未跟踪）", async () => {
    wrap(<GitPanel />);
    await screen.findByText("已暂存");
    expect(screen.getByText("已暂存")).toBeTruthy();
    expect(screen.getByText("未暂存")).toBeTruthy();
    expect(screen.getByText("未跟踪")).toBeTruthy();
    expect(screen.getAllByText("报告.md").length).toBeGreaterThan(0);
    expect(screen.getAllByText("main.go").length).toBeGreaterThan(0);
    expect(screen.getAllByText("新文件.txt").length).toBeGreaterThan(0);
    expect(screen.getByText(/↑1/)).toBeTruthy();
  });

  it("非仓库错误态诚实显示", async () => {
    mocks.status.mockResolvedValue({ isRepo: false, files: [], error: "当前工作区不是 Git 仓库" });
    wrap(<GitPanel />);
    await screen.findByText("Git 不可用");
    expect(screen.getByText("当前工作区不是 Git 仓库")).toBeTruthy();
  });

  it("展开已暂存文件 → 拉暂存区 diff 并渲染红绿行", async () => {
    mocks.diff.mockResolvedValue(
      "@@ -1,2 +1,2 @@\n-old line\n+new line\n ctx\n",
    );
    wrap(<GitPanel />);
    await screen.findByText("报告.md");
    fireEvent.click(screen.getByTitle(/的暂存区 diff/));
    const hunk = await screen.findByTestId("changes-diff-hunk");
    expect(hunk.textContent).toContain("+new line");
    expect(hunk.textContent).toContain("-old line");
    expect(mocks.diff).toHaveBeenCalledWith("docs/报告.md", true);
  });

  it("暂存区非空 + 有说明才允许提交；提交后清空说明并刷新", async () => {
    mocks.commit.mockResolvedValue("abc1234");
    wrap(<GitPanel />);
    await screen.findByText("已暂存");
    const btn = screen.getByTestId("git-commit-btn") as HTMLButtonElement;
    expect(btn.disabled).toBe(true); // 说明为空
    fireEvent.change(screen.getByLabelText("提交说明"), { target: { value: "首次提交" } });
    expect((screen.getByTestId("git-commit-btn") as HTMLButtonElement).disabled).toBe(false);
    fireEvent.click(screen.getByTestId("git-commit-btn"));
    await waitFor(() => expect(mocks.commit).toHaveBeenCalledWith("首次提交"));
    expect((screen.getByLabelText("提交说明") as HTMLTextAreaElement).value).toBe("");
  });

  it("丢弃改动两击确认：首击只进入确认态不调用", async () => {
    wrap(<GitPanel />);
    await screen.findByText("main.go");
    // 首击：仅进入确认态（按钮文案切换为「确认丢弃」），未调用
    fireEvent.click(screen.getByText("丢弃"));
    expect(mocks.discard).not.toHaveBeenCalled();
    // 再击：确认执行
    fireEvent.click(screen.getByText("确认丢弃"));
    await waitFor(() => expect(mocks.discard).toHaveBeenCalledWith("src/main.go"));
  });

  it("提交历史折叠懒加载", async () => {
    mocks.log.mockResolvedValue([{ hash: "abc1234", subject: "某次提交", author: "t", ts: 1 }]);
    wrap(<GitPanel />);
    await screen.findByText("已暂存");
    expect(mocks.log).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("git-history-toggle"));
    await screen.findByText("某次提交");
    expect(mocks.log).toHaveBeenCalledWith(30);
  });
});
