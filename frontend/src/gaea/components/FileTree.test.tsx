// T7-4（v2.37.0）FileTree 加载三态：失败=错误 + 重试按钮（不再假空目录）。
// 2026-08-20 办公蒸馏（dsh-better-sidebar 文件工作台）：
// 行悬浮 @ 引用 / 右键复制路径（已复制反馈）/ 展开态持久化 / 树内搜索
// （C8：接 GaeaFileSearch 工作区递归搜索，蒸馏插件 TreePanel 的 host fs.search）。
import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { FileTree } from "./FileTree";

type G = { go?: { app?: Record<string, unknown> } };

const ROOT_ENTRIES = [
  { name: "docs", isDir: true },
  { name: "README.md", isDir: false },
  { name: "notes.txt", isDir: false },
];

// 与现有测试一致的 mock 注入：window.go.app.CoreB.GaeaListDir。
function mockListDir(impl: (dir: string) => Promise<unknown>) {
  const f: Record<string, unknown> = { GaeaListDir: vi.fn().mockImplementation(impl) };
  (window as unknown as G).go = { app: { CoreB: f } };
  return f;
}

// 注入多个 CoreB 绑定（ListDir + FileSearch）。
function mockCoreB(methods: Record<string, unknown>) {
  (window as unknown as G).go = { app: { CoreB: methods } };
}

describe("FileTree 加载三态（T7-4）", () => {
  afterEach(() => {
    delete (window as unknown as G).go;
    localStorage.clear();
  });

  it("目录加载失败：显示错误 + 重试按钮，不再呈现假空目录", async () => {
    mockListDir(() => Promise.reject(new Error("io error")));
    render(<FileTree cwd="C:/proj" onSelect={() => {}} />);

    expect(await screen.findByText(/加载失败：io error/)).toBeTruthy();
    expect(screen.getByText("重试")).toBeTruthy();
    expect(screen.queryByText("空目录")).toBeNull(); // 不再把失败伪装成空目录
  });

  it("点击重试：加载成功并显示目录条目", async () => {
    const f = mockListDir(() => Promise.reject(new Error("io error")));
    render(<FileTree cwd="C:/proj" onSelect={() => {}} />);
    await screen.findByText(/加载失败/);

    (f.GaeaListDir as ReturnType<typeof vi.fn>).mockResolvedValue(ROOT_ENTRIES);
    fireEvent.click(screen.getByText("重试"));

    expect(await screen.findByText("README.md")).toBeTruthy();
    expect(screen.queryByText(/加载失败/)).toBeNull();
  });
});

describe("文件工作台蒸馏（dsh-better-sidebar）", () => {
  afterEach(() => {
    delete (window as unknown as G).go;
    localStorage.clear();
    delete (navigator as unknown as { clipboard?: unknown }).clipboard;
  });

  it("行悬浮 @ 按钮：点击后 onReference 收到相对路径，且不触发选中", async () => {
    mockListDir(() => Promise.resolve(ROOT_ENTRIES));
    const onReference = vi.fn();
    const onSelect = vi.fn();
    render(<FileTree cwd="C:/proj" onSelect={onSelect} onReference={onReference} />);
    await screen.findByText("README.md");

    const fileRow = screen.getByText("README.md").closest('[role="button"]');
    expect(fileRow).not.toBeNull();
    fireEvent.click(within(fileRow as HTMLElement).getByLabelText("引用到输入框"));

    expect(onReference).toHaveBeenCalledTimes(1);
    expect(onReference).toHaveBeenCalledWith("README.md");
    expect(onSelect).not.toHaveBeenCalled();
  });

  it("右键复制相对路径：writeText 收到相对路径，行尾显示「已复制」", async () => {
    mockListDir(() => Promise.resolve(ROOT_ENTRIES));
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { value: { writeText }, configurable: true });
    render(<FileTree cwd="C:/proj" onSelect={() => {}} />);
    await screen.findByText("README.md");

    fireEvent.contextMenu(screen.getByText("README.md"));
    const copyItem = await screen.findByText("复制相对路径");
    fireEvent.click(copyItem);

    await waitFor(() => expect(writeText).toHaveBeenCalledWith("README.md"));
    expect(await screen.findByText("已复制")).toBeTruthy();
  });

  it("展开态持久化：展开目录落盘 localStorage，同 cwd 重挂载后目录保持展开", async () => {
    const dirs: string[] = [];
    mockListDir((dir: string) => {
      dirs.push(dir);
      if (dir === "docs") return Promise.resolve([{ name: "plan.md", isDir: false }]);
      return Promise.resolve(ROOT_ENTRIES);
    });
    const { unmount } = render(<FileTree cwd="C:/proj" onSelect={() => {}} />);
    await screen.findByText("docs");

    fireEvent.click(screen.getByText("docs"));
    expect(await screen.findByText("plan.md")).toBeTruthy();

    const key = `gaea.fileTree.expanded.${encodeURIComponent("C:/proj")}`;
    const raw = localStorage.getItem(key);
    expect(raw).not.toBeNull();
    expect(JSON.parse(raw as string)).toEqual({ docs: true });

    unmount();
    render(<FileTree cwd="C:/proj" onSelect={() => {}} />);
    // 目录保持展开：docs 重新加载，子文件再次可见（证明展开态从 localStorage 恢复）
    expect(await screen.findByText("plan.md")).toBeTruthy();
    expect(dirs.filter((d) => d === "docs")).toHaveLength(2);
  });

  it("树内搜索（C8）：非空查询走后端工作区递归搜索，点击文件命中触发预览", async () => {
    const listDir = vi.fn().mockResolvedValue(ROOT_ENTRIES);
    const fileSearch = vi.fn().mockResolvedValue([
      { path: "docs/plan.md", name: "plan.md", isDir: false },
      { path: "docs", name: "docs", isDir: true },
    ]);
    mockCoreB({ GaeaListDir: listDir, GaeaFileSearch: fileSearch });
    const onSelect = vi.fn();
    render(<FileTree cwd="C:/proj" onSelect={onSelect} />);
    await screen.findByText("README.md");

    const input = screen.getByPlaceholderText("过滤文件名");
    fireEvent.change(input, { target: { value: "plan" } });

    // 300ms 防抖后进入搜索模式：命中列表取代树（README.md 不再渲染），
    // 文件命中可点击预览、目录命中不可预览、范围提示出现。
    await waitFor(() => expect(fileSearch).toHaveBeenCalledWith("plan", 50));
    expect(await screen.findByText("docs/plan.md")).toBeTruthy();
    expect(screen.queryByText("README.md")).toBeNull();
    fireEvent.click(screen.getByText("docs/plan.md"));
    expect(onSelect).toHaveBeenCalledWith("docs/plan.md");

    const dirHit = screen.getByText("docs").closest('[role="button"]');
    expect(dirHit).not.toBeNull();
    fireEvent.click(dirHit as HTMLElement);
    expect(onSelect).toHaveBeenCalledTimes(1); // 目录命中点击不触发预览

    expect(await screen.findByText(/搜索范围：整个工作区/)).toBeTruthy();
  });

  it("树内搜索（C8）：无命中显示提示，搜索失败显示错误", async () => {
    const listDir = vi.fn().mockResolvedValue(ROOT_ENTRIES);
    const fileSearch = vi.fn().mockResolvedValue([]);
    mockCoreB({ GaeaListDir: listDir, GaeaFileSearch: fileSearch });
    render(<FileTree cwd="C:/proj" onSelect={() => {}} />);
    await screen.findByText("README.md");

    const input = screen.getByPlaceholderText("过滤文件名");
    fireEvent.change(input, { target: { value: "zzz-no-such" } });
    expect(await screen.findByText("无匹配文件")).toBeTruthy();

    (fileSearch as ReturnType<typeof vi.fn>).mockRejectedValue(new Error("search down"));
    fireEvent.change(input, { target: { value: "zzz-err" } });
    expect(await screen.findByText(/搜索失败：search down/)).toBeTruthy();
  });

  it("树内搜索（C8）：清空按钮还原为树", async () => {
    const listDir = vi.fn().mockResolvedValue(ROOT_ENTRIES);
    const fileSearch = vi.fn().mockResolvedValue([{ path: "plan.md", name: "plan.md", isDir: false }]);
    mockCoreB({ GaeaListDir: listDir, GaeaFileSearch: fileSearch });
    render(<FileTree cwd="C:/proj" onSelect={() => {}} />);
    await screen.findByText("README.md");

    const input = screen.getByPlaceholderText("过滤文件名");
    fireEvent.change(input, { target: { value: "plan" } });
    expect(await screen.findByText("plan.md")).toBeTruthy();
    expect(screen.queryByText("README.md")).toBeNull();

    fireEvent.click(screen.getByLabelText("清空搜索"));
    expect(await screen.findByText("README.md")).toBeTruthy();
  });
});
