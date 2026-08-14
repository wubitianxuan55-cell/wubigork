import { describe, expect, it } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { OfficeMemoryLibrary } from "./OfficeMemoryLibrary";
import { LocaleProvider } from "../../lib/i18n";

const wrap = (node: React.ReactNode) => <LocaleProvider>{node}</LocaleProvider>;

describe("OfficeMemoryLibrary 办公记忆", () => {
  it("查重合并：列出疑似重复并一键合并", async () => {
    render(wrap(<OfficeMemoryLibrary />));

    expect(await screen.findByText(/User prefers tabs/)).toBeTruthy();
    fireEvent.click(screen.getByTitle("查重：合并疑似重复的记忆事实"));

    expect(await screen.findByText("桩基施工要点")).toBeTruthy();
    expect(screen.getByText(/桩基施工 要点（修订）/)).toBeTruthy();
    expect(screen.getByText("87%")).toBeTruthy();

    fireEvent.click(screen.getByTitle(/把「桩基施工 要点（修订）」合并进「桩基施工要点」/));
    await waitFor(() => expect(screen.getAllByText(/已合并「pile-v2」→「pile」/).length).toBeGreaterThan(0));
  });

  it("清理超期归档：确认弹窗 → 调用清理 → 提示已清理", async () => {
    render(wrap(<OfficeMemoryLibrary />));

    // 切到「归档」tab，出现「清理超期归档」按钮
    fireEvent.click(screen.getByRole("button", { name: /归档/ }));
    fireEvent.click(screen.getByRole("button", { name: /清理超期归档/ }));

    // 确认弹窗（antd Modal.confirm）：标题在 ant-modal-title 与 confirm-title 各渲染一次
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getAllByText("清理 90 天前归档？").length).toBeGreaterThan(0);

    // 确认清理 → mock 返回 0 → 提示已清理
    fireEvent.click(within(dialog).getByRole("button", { name: /清\s*理/ }));
    expect(await screen.findByText(/已清理 0 条超期归档/)).toBeTruthy();
  });
});
