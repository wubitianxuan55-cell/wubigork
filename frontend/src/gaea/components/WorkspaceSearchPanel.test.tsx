import { describe, expect, it } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { WorkspaceSearchPanel } from "./WorkspaceSearchPanel";
import { ToastProvider } from "./Toast";

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

describe("WorkspaceSearchPanel 工作区搜索", () => {
  it("语义模式：开启后展示本地语义命中并可重建索引", async () => {
    render(wrap(<WorkspaceSearchPanel onOpenFile={() => {}} />));

    fireEvent.click(screen.getByTitle("语义检索（本地 bge-m3，需先重建索引）"));
    fireEvent.change(screen.getByPlaceholderText("搜索资料正文，如：成本 / 预算 / 方案…"), {
      target: { value: "打桩" },
    });

    expect(await screen.findByText(/语义命中（本地 bge-m3）/)).toBeTruthy();
    expect(screen.getByText("桩基施工方案.md")).toBeTruthy();
    expect(screen.getByText("82%")).toBeTruthy();
  });

  it("重建索引显示结果", async () => {
    render(wrap(<WorkspaceSearchPanel onOpenFile={() => {}} />));

    fireEvent.click(screen.getByTitle("重建工作区文件语义索引"));
    await waitFor(() => expect(screen.getByText(/已索引 3 个文件/)).toBeTruthy());
  });
});
