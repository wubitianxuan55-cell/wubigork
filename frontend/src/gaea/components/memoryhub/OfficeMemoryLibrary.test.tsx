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

  it("归档 tab：分页列表 + 保留期文案 + 恢复按钮", async () => {
    render(wrap(<OfficeMemoryLibrary />));

    // 切到「归档」tab：分页加载（GaeaMemoryArchivedList），显示保留期 + 归档条目
    fireEvent.click(screen.getByRole("button", { name: /归档/ }));

    // 保留期文案（retentionDays 由后端下发，mock=90）
    expect(await screen.findByText(/归档保留 90 天，超期可清理/)).toBeTruthy();
    // 分页条目（mock 2 条：pile-v2 / cost-2025）
    expect(await screen.findByText("桩基施工 要点（修订）")).toBeTruthy();
    expect(screen.getByText("2025 年机械台班价（已过期）")).toBeTruthy();

    // 恢复按钮：点击 → 调 MemoryUnarchive → 提示已恢复
    fireEvent.click(screen.getByTitle(/把「桩基施工 要点（修订）」恢复回活跃记忆/));
    expect(await screen.findByText(/已恢复「pile-v2」回活跃记忆/)).toBeTruthy();
  });
});
