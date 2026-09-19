import { describe, expect, it, beforeEach } from "vitest";
import { act, render, screen, fireEvent } from "@testing-library/react";
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

  // v4.349 实时化：此前只在 cwd 变化时重读，办公里新打开的文件不会出现在本条里。
  it("挂载后写入新文件：无需重挂载即出现在条上（订阅单源）", () => {
    recordRecentFile("docs/旧方案.docx");
    render(<RecentFilesBar onOpenFile={() => {}} />);
    expect(screen.queryByText("新数据.xlsx")).toBeNull();

    act(() => { recordRecentFile("docs/新数据.xlsx"); });
    expect(screen.getByText("新数据.xlsx")).toBeTruthy();
    expect(screen.getByText("旧方案.docx")).toBeTruthy();
  });

  it("空态挂载后首次写入也会出现（原先空态返回 null 后永不更新）", () => {
    const { container } = render(<RecentFilesBar onOpenFile={() => {}} />);
    expect(container.firstChild).toBeNull();

    act(() => { recordRecentFile("docs/首个.md"); });
    expect(screen.getByText("首个.md")).toBeTruthy();
  });
});
