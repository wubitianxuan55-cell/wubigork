import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { LocaleProvider } from "../lib/i18n";
import { GitPanel } from "./GitPanel";
import type { GitCommitInfoView, GitStatusView } from "../lib/types";
import { resetGitStatusStoreForTest } from "../lib/gitStatusStore";

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
  worktrees: vi.fn(),
  graph: vi.fn(),
  commitDetail: vi.fn(),
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
    GaeaGitWorktrees: mocks.worktrees,
    GaeaGitGraph: mocks.graph,
    GaeaGitCommitDetail: mocks.commitDetail,
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
    resetGitStatusStoreForTest();
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

  it("本轮文件镜头复用 ChangesPanel，与 Git 状态共存于同一工作台", async () => {
    wrap(<GitPanel changes={[{ path: "docs/本轮改动.md", count: 2 }]} />);
    await screen.findByText("已暂存");
    fireEvent.click(screen.getByRole("button", { name: "本轮文件" }));
    expect(await screen.findByText("docs/本轮改动.md")).toBeTruthy();
  });

  it("统一镜头按目录合并 Git 与 Agent 来源", async () => {
    wrap(<GitPanel changes={[{ path: "docs/报告.md", count: 2 }, { path: "src/agent.ts", count: 1 }]} />);
    await screen.findByText("已暂存");
    fireEvent.click(screen.getByRole("button", { name: "统一" }));
    expect(await screen.findByTestId("unified-changes-tree")).toBeTruthy();
    expect(document.querySelector('[data-unified-file="docs/报告.md"]')?.textContent).toContain("AI 2");
    expect(document.querySelector('[data-unified-file="src/agent.ts"]')).toBeTruthy();
  });

  it("统一镜头来源筛选与快操作直连既有 Git 绑定", async () => {
    wrap(<GitPanel changes={[{ path: "docs/agent.md", count: 1 }]} />);
    await screen.findByText("已暂存");
    fireEvent.click(screen.getByRole("button", { name: "统一" }));
    await screen.findByTestId("unified-changes-tree");
    fireEvent.click(screen.getByRole("button", { name: /^AI / }));
    expect(document.querySelector('[data-unified-file="src/main.go"]')).toBeNull();
    expect(document.querySelector('[data-unified-file="docs/agent.md"]')).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /^全部 / }));
    fireEvent.click(await screen.findByLabelText("暂存 src/main.go"));
    await waitFor(() => expect(mocks.stage).toHaveBeenCalledWith(["src/main.go"]));
  });

  it("统一树文件行内展开工作区 diff", async () => {
    mocks.diff.mockResolvedValue("@@ -1 +1 @@\n-old-inline\n+new-inline\n");
    wrap(<GitPanel />);
    await screen.findByText("已暂存");
    fireEvent.click(screen.getByRole("button", { name: "统一" }));
    fireEvent.click(await screen.findByLabelText("查看 src/main.go 的 diff"));
    await waitFor(() => expect(mocks.diff).toHaveBeenCalledWith("src/main.go", false));
    const hunk = await screen.findByTestId("changes-diff-hunk");
    expect(hunk.textContent).toContain("+new-inline");
  });

  // G1 深化：工作树镜头——只读展示，主检出/当前/detached 三态徽标与分支名。
  const sampleWorktrees = [
    { path: "C:\\demo\\repo", head: "1a2b3c4", branch: "main", isMain: true, isCurrent: false },
    { path: "C:\\demo\\repo\\.worktrees\\feature-x", head: "5d6e7f8", branch: "feature/x", isCurrent: true },
    { path: "C:\\demo\\repo\\.worktrees\\experiment", head: "9f8e7d6", isDetached: true },
  ];

  it("工作树镜头展示主检出/当前/detached 徽标与分支", async () => {
    mocks.worktrees.mockResolvedValue(sampleWorktrees);
    wrap(<GitPanel />);
    await screen.findByText("已暂存");
    fireEvent.click(screen.getByRole("button", { name: "工作树" }));
    expect(await screen.findByText("主检出")).toBeTruthy();
    expect(screen.getByText("当前")).toBeTruthy();
    expect(screen.getByText("feature/x")).toBeTruthy();
    expect(screen.getByText("detached")).toBeTruthy();
    expect(mocks.worktrees).toHaveBeenCalled();
  });

  it("工作树镜头拉取失败诚实报错", async () => {
    mocks.worktrees.mockRejectedValue(new Error("git 超时（20s）"));
    wrap(<GitPanel />);
    await screen.findByText("已暂存");
    fireEvent.click(screen.getByRole("button", { name: "工作树" }));
    expect(await screen.findByText("工作树不可用")).toBeTruthy();
    expect(screen.getByText(/git 超时/)).toBeTruthy();
  });

  // G1.7：冲突独立分组——不进已暂存组（不可提交），提示诚实，不代解。
  it("冲突文件独立分组且不误入已暂存组", async () => {
    mocks.status.mockResolvedValue({
      ...repoStatus,
      files: [
        { path: "src/main.go", x: "M", y: " ", staged: true, modified: true },
        { path: "conflicted.txt", x: "U", y: "U", conflict: true },
      ],
    });
    wrap(<GitPanel />);
    await screen.findByTestId("git-group-冲突");
    expect(screen.getByTestId("git-group-冲突")).toBeTruthy();
    // 冲突文件不在已暂存组（仅 staged 的 main.go 在），且提示可见。
    expect(screen.queryByLabelText("暂存 conflicted.txt")).toBeNull();
    expect(screen.queryByLabelText("取消暂存 conflicted.txt")).toBeNull();
    expect(screen.getByText(/本面板不代解/)).toBeTruthy();
    // 展开冲突行 → 诚实解决指引而非拉 diff（「标记已解决」为行内提示独有文案）。
    fireEvent.click(screen.getByTitle("查看 conflicted.txt 的工作区 diff"));
    expect(await screen.findByText(/标记已解决/)).toBeTruthy();
    expect(mocks.diff).not.toHaveBeenCalled();
  });

  // G1.7：重命名溯源——porcelain -z 来源路径以 ← 原名显示。
  it("重命名行显示来源路径", async () => {
    mocks.status.mockResolvedValue({
      ...repoStatus,
      files: [{ path: "docs/改名后.md", x: "R", y: " ", staged: true, renamed: true, origPath: "docs/改名前.md" }],
    });
    wrap(<GitPanel />);
    expect(await screen.findByText("← 改名前.md")).toBeTruthy();
    expect(screen.getByTitle("重命名自 docs/改名前.md")).toBeTruthy();
  });

  // G1.7：二进制 diff——buildGitDiff 按前缀给诚实占位，不渲染假红绿行。
  it("二进制 diff 显示诚实占位", async () => {
    mocks.diff.mockResolvedValue("Binary files a/src/blob.bin and b/src/blob.bin differ");
    mocks.status.mockResolvedValue({
      ...repoStatus,
      files: [{ path: "src/blob.bin", x: " ", y: "M", modified: true }],
    });
    wrap(<GitPanel />);
    fireEvent.click(await screen.findByTitle("查看 src/blob.bin 的工作区 diff"));
    expect(await screen.findByText("二进制文件：不提供文本 diff")).toBeTruthy();
  });

  // G2 只读提交图：镜头渲染提交行（lane SVG+装饰），展开详情拉 numstat+diff。
  const sampleGraph = {
    commits: [
      { hash: "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0", short: "a1b2c3d", subject: "发布 v0.32.0", author: "walk", ts: 1794502086, parents: ["b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1"], refs: ["HEAD -> main"] },
      { hash: "b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1", short: "b2c3d4e", subject: "merge feature", author: "walk", ts: 1794502000, parents: ["c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2", "d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3"] },
      { hash: "c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2", short: "c3d4e5f", subject: "init", author: "walk", ts: 1794501800, parents: [], refs: ["tag: v0.1.0"] },
      { hash: "d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2e3", short: "d4e5f6a", subject: "feat: worktree", author: "walk", ts: 1794501910, parents: ["c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2"], refs: ["origin/feature"] },
    ],
    branches: ["main", "feature"],
  };

  it("提交图镜头渲染提交行与装饰，详情展开拉 numstat+diff", async () => {
    mocks.graph.mockResolvedValue(sampleGraph);
    mocks.commitDetail.mockResolvedValue({
      hash: sampleGraph.commits[0].hash,
      subject: "发布 v0.32.0",
      body: "",
      author: "walk",
      ts: 1794502086,
      parents: sampleGraph.commits[0].parents,
      refs: ["HEAD -> main"],
      files: [
        { path: "internal/gaea/workbench/fs.go", add: 12, del: 3 },
        { path: "assets/logo.png", add: -1, del: -1 },
      ],
      diff: "diff --git a/f b/f\n--- a/f\n+++ b/f\n@@ -1 +1,2 @@\n 上下文\n+新增",
    });
    wrap(<GitPanel />);
    await screen.findByText("已暂存");
    fireEvent.click(screen.getByRole("button", { name: "提交图" }));
    expect(await screen.findByText("发布 v0.32.0")).toBeTruthy();
    expect(screen.getByText("merge feature")).toBeTruthy();
    // HEAD -> main 装饰 chip 与分支下拉选项都含 "main"（≥2 处）
    expect(screen.getAllByText("main").length).toBeGreaterThan(1);
    expect(screen.getByText("🏷 v0.1.0")).toBeTruthy(); // tag 装饰
    // 展开首行详情 → numstat 文件（含二进制）+ diff 红绿行
    fireEvent.click(screen.getByTitle(/发布 v0.32.0（a1b2c3d）/));
    expect(await screen.findByText("internal/gaea/workbench/fs.go")).toBeTruthy();
    expect(screen.getByText("二进制")).toBeTruthy();
    expect(await screen.findByTestId("changes-diff-hunk")).toBeTruthy();
    expect(mocks.commitDetail).toHaveBeenCalledWith(sampleGraph.commits[0].hash);
    // 分支筛选下拉切换 → 以分支重拉
    fireEvent.change(screen.getByLabelText("按分支筛选提交图"), { target: { value: "feature" } });
    await waitFor(() => expect(mocks.graph).toHaveBeenCalledWith(0, 50, "feature"));
  });

  it("提交图镜头拉取失败诚实报错", async () => {
    mocks.graph.mockRejectedValue(new Error("非仓库"));
    wrap(<GitPanel />);
    await screen.findByText("已暂存");
    fireEvent.click(screen.getByRole("button", { name: "提交图" }));
    expect(await screen.findByText("提交图不可用")).toBeTruthy();
  });
});
