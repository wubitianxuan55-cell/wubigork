import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { ChangesPanel } from "./ChangesPanel";
import type { SessionChange } from "../lib/changes";

const changes: SessionChange[] = [
  { path: "/ws/a.md", count: 1, lastTouched: 2 },
  { path: "/ws/b.md", count: 4, lastTouched: 5 },
];

describe("ChangesPanel 文件变更面板", () => {
  it("按最近改动倒序展示，并汇总文件数与改动次数", () => {
    const { container } = render(<ChangesPanel changes={changes} onOpenFile={() => {}} />);
    expect(screen.getByText(/2 个文件 · 5 次/)).toBeTruthy();
    const names = Array.from(container.querySelectorAll("span.truncate.text-fg-dim")).map((n) => n.textContent);
    expect(names).toEqual(["b.md", "a.md"]);
  });

  it("无变更时展示空状态", () => {
    render(<ChangesPanel changes={[]} onOpenFile={() => {}} />);
    expect(screen.getByText(/本会话暂无文件变更/)).toBeTruthy();
  });
});
