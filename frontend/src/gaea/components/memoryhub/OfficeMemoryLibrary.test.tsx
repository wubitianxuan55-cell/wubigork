import { describe, expect, it } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
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
});
