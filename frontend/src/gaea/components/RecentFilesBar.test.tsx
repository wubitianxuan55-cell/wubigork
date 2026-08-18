import { describe, expect, it, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { RecentFilesBar } from "./RecentFilesBar";
import { clearRecentFilesForTest, recordRecentFile } from "../lib/recentFiles";

describe("RecentFilesBar 最近文件快捷区（P0-3）", () => {
  beforeEach(() => {
    clearRecentFilesForTest();
  });

  it("无最近文件时返回 null", () => {
    const { container } = render(<RecentFilesBar onOpenFile={() => {}} />);
    expect(container.firstChild).toBeNull();
  });

  it("渲染最近文件 chip", () => {
    recordRecentFile("docs/方案.docx");
    recordRecentFile("docs/数据.xlsx");
    render(<RecentFilesBar onOpenFile={() => {}} />);
    expect(screen.getByText("方案.docx")).toBeTruthy();
    expect(screen.getByText("数据.xlsx")).toBeTruthy();
  });

  it("点击 chip 打开文件并再次置顶", () => {
    recordRecentFile("docs/a.md");
    recordRecentFile("docs/b.md");
    let opened = "";
    render(<RecentFilesBar onOpenFile={(p) => { opened = p; }} />);
    fireEvent.click(screen.getByRole("button", { name: /预览 b\.md/ }));
    expect(opened).toBe("docs/b.md");
  });
});
