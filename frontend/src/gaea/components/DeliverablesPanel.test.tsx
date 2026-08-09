import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { DeliverablesPanel } from "./DeliverablesPanel";
import { usePreviewStore, useUpdatedFilesStore } from "../lib/store";

describe("DeliverablesPanel 会话产物面板", () => {
  it("展示会话交付文件，点击打开预览", () => {
    usePreviewStore.setState({ previewFile: null });
    render(
      <DeliverablesPanel
        items={[
          { path: "exports/成本测算.xlsx", sourceId: "a1" },
          { path: ".gaea/exports/方案.docx", sourceId: "a2" },
        ]}
        onOpenFile={(p) => usePreviewStore.setState({ previewFile: p })}
      />,
    );
    expect(screen.getByText("成本测算.xlsx")).toBeTruthy();
    expect(screen.getByText("方案.docx")).toBeTruthy();
    fireEvent.click(screen.getByText("成本测算.xlsx"));
    expect(usePreviewStore.getState().previewFile).toBe("exports/成本测算.xlsx");
  });

  it("无交付文件时显示空状态", () => {
    render(<DeliverablesPanel items={[]} onOpenFile={() => {}} />);
    expect(screen.getByText(/暂无交付文件/)).toBeTruthy();
  });

  it("编辑过的文件显示「已更新」徽标", () => {
    useUpdatedFilesStore.setState({ updatedAt: { "exports/成本测算.xlsx": Date.now() } });
    render(
      <DeliverablesPanel
        items={[{ path: "exports/成本测算.xlsx", sourceId: "a1" }]}
        onOpenFile={() => {}}
      />,
    );
    expect(screen.getByText("已更新")).toBeTruthy();
    useUpdatedFilesStore.setState({ updatedAt: {} });
  });

  it("点击「跳转到生成它的消息」回调对应轮次", () => {
    const calls: number[] = [];
    render(
      <DeliverablesPanel
        items={[{ path: "exports/成本测算.xlsx", sourceId: "a1", turn: 2 }]}
        onOpenFile={() => {}}
        onLocateSource={(turn) => calls.push(turn)}
      />,
    );
    fireEvent.click(screen.getByTitle("跳转到生成它的消息"));
    expect(calls).toEqual([2]);
  });
});
