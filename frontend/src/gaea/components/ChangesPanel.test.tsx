import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { ChangesPanel } from "./ChangesPanel";
import { useStore, type Item } from "../lib/store";
import { ToastProvider } from "./Toast";
import type { SessionChange } from "../lib/changes";
import type { JournalChangeRecord } from "../lib/types";

// 桥接 mock：回滚链路（GaeaJournalList → RollbackRecord）是本面板唯一下行绑定。
const mocks = vi.hoisted(() => ({
  journalList: vi.fn(async (_limit: number): Promise<JournalChangeRecord[]> => []),
  rollback: vi.fn(async (_id: string) => {}),
}));

vi.mock("../lib/bridge", () => ({
  app: {
    GaeaJournalList: (limit: number) => mocks.journalList(limit),
    RollbackRecord: (id: string) => mocks.rollback(id),
  },
  onEvent: () => () => {},
  onReady: (cb: () => void) => {
    cb();
    return () => {};
  },
}));

const changes: SessionChange[] = [
  { path: "/ws/a.md", count: 1, lastTouched: 2 },
  { path: "/ws/b.md", count: 1, lastTouched: 3 },
];

const editItem: Item = {
  kind: "tool",
  id: "t1",
  name: "edit_file",
  args: JSON.stringify({ path: "/ws/a.md", old_string: "旧标题\n共有", new_string: "新标题\n共有" }),
  readOnly: false,
  status: "done",
};
const writeItem: Item = {
  kind: "tool",
  id: "t2",
  name: "write_file",
  args: JSON.stringify({ path: "/ws/b.md", content: "全新内容" }),
  readOnly: false,
  status: "done",
};

function seedItems(items: Item[]) {
  useStore.setState({ items });
}

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

beforeEach(() => {
  mocks.journalList.mockClear();
  mocks.rollback.mockClear();
  mocks.journalList.mockResolvedValue([]);
  seedItems([editItem, writeItem]);
});

afterEach(() => {
  seedItems([]);
});

describe("ChangesPanel 文件变更面板（v4.25 diff 化）", () => {
  it("按最近改动倒序展示，并汇总文件数与改动次数（既有行为保持）", () => {
    const { container } = render(wrap(<ChangesPanel changes={changes} onOpenFile={() => {}} />));
    expect(screen.getByText(/2 个文件 · 2 次/)).toBeTruthy();
    const names = Array.from(container.querySelectorAll("span.truncate.text-fg-dim")).map((n) => n.textContent);
    expect(names).toEqual(["b.md", "a.md"]);
  });

  it("无变更时展示空状态（prop 与 store items 均为空）", () => {
    seedItems([]); // 2a 起 items 为权威源，store 有种子则不视为空
    render(wrap(<ChangesPanel changes={[]} onOpenFile={() => {}} />));
    expect(screen.getByText(/本会话暂无文件变更/)).toBeTruthy();
  });

  it("展开文件行 → edit_file 渲染行级红绿 diff；再点收起", async () => {
    const { container } = render(wrap(<ChangesPanel changes={changes} onOpenFile={() => {}} />));
    fireEvent.click(screen.getByLabelText("展开 /ws/a.md 的改动 diff"));

    const hunk = await waitFor(() => {
      const h = container.querySelector('[data-testid="changes-diff-hunk"]');
      expect(h).toBeTruthy();
      return h as HTMLElement;
    });
    expect(hunk.textContent).toContain("+新标题");
    expect(hunk.textContent).toContain("-旧标题");

    // 再点收起
    fireEvent.click(screen.getByLabelText("收起 /ws/a.md 的改动 diff"));
    await waitFor(() => expect(container.querySelector('[data-testid="changes-diff-hunk"]')).toBeNull());
  });

  it("write_file（无 old/new）→ 展开后显示写入内容预览降级态而非 diff", async () => {
    render(wrap(<ChangesPanel changes={changes} onOpenFile={() => {}} />));
    fireEvent.click(screen.getByLabelText("展开 /ws/b.md 的改动 diff"));
    await waitFor(() => expect(screen.getByTestId("changes-content-preview")).toBeTruthy());
    expect(screen.getByText(/覆盖写入/)).toBeTruthy();
    expect(screen.getByText(/全新内容/)).toBeTruthy();
    expect(document.querySelector('[data-testid="changes-diff-hunk"]')).toBeNull();
  });

  it("展开区「打开预览」按钮回调 onOpenFile（行本身改为展开切换）", () => {
    const onOpenFile = vi.fn();
    render(wrap(<ChangesPanel changes={changes} onOpenFile={onOpenFile} />));
    fireEvent.click(screen.getByLabelText("展开 /ws/a.md 的改动 diff"));
    fireEvent.click(screen.getByTitle("打开文件预览"));
    expect(onOpenFile).toHaveBeenCalledWith("/ws/a.md");
  });

  it("证据链有基线快照 → 显示回滚按钮并调用 RollbackRecord", async () => {
    mocks.journalList.mockResolvedValue([
      {
        id: "ev_42",
        sessionId: "s1",
        space: "work",
        turn: 3,
        tool: "edit_file",
        target: "a.md",
        beforeSummary: "旧",
        afterSummary: "新",
        at: 1,
        status: "applied",
        baselinePath: ".gaea/snapshots/a.md.snap",
      },
    ]);
    render(wrap(<ChangesPanel changes={changes} cwd="/ws" onOpenFile={() => {}} />));
    // cwd 锚定后行内展示相对路径 a.md
    fireEvent.click(screen.getByLabelText("展开 a.md 的改动 diff"));

    const btn = await screen.findByTitle(/回滚到写盘前基线快照/);
    fireEvent.click(btn);
    await waitFor(() => expect(mocks.rollback).toHaveBeenCalledWith("ev_42"));
  });

  it("证据链无基线 → 诚实标注暂不可回滚，不出现伪造按钮", async () => {
    render(wrap(<ChangesPanel changes={changes} onOpenFile={() => {}} />));
    fireEvent.click(screen.getByLabelText("展开 /ws/a.md 的改动 diff"));
    expect(await screen.findByText(/暂无基线快照，不可回滚/)).toBeTruthy();
    expect(screen.queryByTitle(/回滚到写盘前基线快照/)).toBeNull();
  });

  it("失败的工具调用标注「不构成变更」，不渲染红绿 diff", async () => {
    seedItems([
      {
        kind: "tool",
        id: "t9",
        name: "edit_file",
        args: JSON.stringify({ path: "/ws/a.md", old_string: "旧", new_string: "新" }),
        readOnly: false,
        status: "error",
      } as Item,
    ]);
    render(wrap(<ChangesPanel changes={changes} onOpenFile={() => {}} />));
    fireEvent.click(screen.getByLabelText("展开 /ws/a.md 的改动 diff"));
    expect(await screen.findByText(/调用未成功落盘/)).toBeTruthy();
    expect(document.querySelector('[data-testid="changes-diff-hunk"]')).toBeNull();
  });
});

describe("ChangesPanel 2a 三态折叠 + 类型筛选", () => {
  const readItem: Item = {
    kind: "tool",
    id: "t3",
    name: "read_file",
    args: JSON.stringify({ path: "/ws/a.md" }),
    readOnly: true,
    status: "done",
  };
  const visionItem: Item = {
    kind: "tool",
    id: "t4",
    name: "vision",
    args: JSON.stringify({ image_path: "/ws/图表.png" }),
    readOnly: true,
    status: "done",
  };

  beforeEach(() => {
    seedItems([editItem, writeItem, readItem, visionItem]);
  });

  it("三态独立成层：写入=write_file、编辑=edit_file、读取=read_file/vision；读取层默认收起", () => {
    render(wrap(<ChangesPanel changes={changes} onOpenFile={() => {}} />));
    // 层头计数（过滤前）
    expect(screen.getByTestId("changes-layer-write").textContent).toContain("写入");
    expect(screen.getByTestId("changes-layer-write").textContent).toContain("1");
    expect(screen.getByTestId("changes-layer-edit").textContent).toContain("编辑");
    expect(screen.getByTestId("changes-layer-edit").textContent).toContain("1");
    const readHead = screen.getByTestId("changes-layer-read");
    expect(readHead.textContent).toContain("读取");
    expect(readHead.textContent).toContain("2"); // a.md + 图表.png
    expect(readHead.getAttribute("aria-expanded")).toBe("false"); // 默认收起
    // a.md 同时在编辑层（写入层无它）
    expect(screen.getByLabelText("展开 /ws/a.md 的改动 diff")).toBeTruthy();
    expect(screen.queryByLabelText("展开 /ws/b.md 的改动 diff")).toBeTruthy();
  });

  it("读取层展开后为轻量行：点击直接回调 onOpenFile", () => {
    const onOpenFile = vi.fn();
    render(wrap(<ChangesPanel changes={changes} onOpenFile={onOpenFile} />));
    fireEvent.click(screen.getByTestId("changes-layer-read"));
    fireEvent.click(screen.getByLabelText("打开 /ws/a.md 预览"));
    expect(onOpenFile).toHaveBeenCalledWith("/ws/a.md");
    expect(screen.getAllByText(/读 1 次/).length).toBe(2); // a.md + 图表.png 各 1 次
  });

  it("类型筛选：图片分类只剩读取层的 png，代码分类提示为空", () => {
    render(wrap(<ChangesPanel changes={changes} onOpenFile={() => {}} />));
    fireEvent.click(screen.getByTestId("changes-layer-read")); // 读取层默认收起，先展开
    fireEvent.click(screen.getByTestId("changes-cat-image"));
    expect(screen.getByText("图表.png")).toBeTruthy();
    expect(screen.queryByText("a.md")).toBeNull();
    fireEvent.click(screen.getByTestId("changes-cat-code"));
    expect(screen.getByText(/该类型暂无文件/)).toBeTruthy();
    fireEvent.click(screen.getByTestId("changes-cat-all"));
    // a.md 跨层独立出现（编辑层+读取层各一行），正是「独立折叠层」语义
    expect(screen.getAllByText("a.md").length).toBeGreaterThanOrEqual(2);
  });
});
