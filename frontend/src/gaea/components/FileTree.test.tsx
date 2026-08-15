// T7-4（v2.37.0）FileTree 加载三态：失败=错误 + 重试按钮（不再假空目录）。
import { afterEach, describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { FileTree } from "./FileTree";

type G = { go?: { app?: Record<string, unknown> } };

describe("FileTree 加载三态（T7-4）", () => {
  afterEach(() => {
    delete (window as unknown as G).go;
  });

  it("目录加载失败：显示错误 + 重试按钮，不再呈现假空目录", async () => {
    const f: Record<string, unknown> = { GaeaListDir: vi.fn().mockRejectedValue(new Error("io error")) };
    (window as unknown as G).go = { app: { CoreB: f } };
    render(<FileTree cwd="C:/proj" onSelect={() => {}} />);

    expect(await screen.findByText(/加载失败：io error/)).toBeTruthy();
    expect(screen.getByText("重试")).toBeTruthy();
    expect(screen.queryByText("空目录")).toBeNull(); // 不再把失败伪装成空目录
  });

  it("点击重试：加载成功并显示目录条目", async () => {
    const f: Record<string, unknown> = { GaeaListDir: vi.fn().mockRejectedValue(new Error("io error")) };
    (window as unknown as G).go = { app: { CoreB: f } };
    render(<FileTree cwd="C:/proj" onSelect={() => {}} />);
    await screen.findByText(/加载失败/);

    (f.GaeaListDir as ReturnType<typeof vi.fn>).mockResolvedValue([
      { name: "docs", isDir: true },
      { name: "README.md", isDir: false },
    ]);
    fireEvent.click(screen.getByText("重试"));

    expect(await screen.findByText("README.md")).toBeTruthy();
    expect(screen.queryByText(/加载失败/)).toBeNull();
  });
});
