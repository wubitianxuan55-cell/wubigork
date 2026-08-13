import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { DeliverablesPanel } from "./DeliverablesPanel";
import { useComposerInsertStore, usePreviewStore, useUpdatedFilesStore } from "../lib/store";

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

  it("表格产物提供「沉淀到成本库」操作，指令进入输入框通道", () => {
    useComposerInsertStore.setState({ pendingText: null });
    render(
      <DeliverablesPanel
        items={[{ path: "exports/成本测算.xlsx", sourceId: "a1" }]}
        onOpenFile={() => {}}
      />,
    );
    fireEvent.click(screen.getByTitle("沉淀到成本库：把单价明细用 cost_save 写回成本库"));
    const text = useComposerInsertStore.getState().pendingText ?? "";
    expect(text).toContain("cost_save");
    expect(text).toContain("[成本测算.xlsx](exports/成本测算.xlsx)");
    useComposerInsertStore.getState().consumeText();
  });

  it("非表格产物不显示「沉淀到成本库」操作", () => {
    render(
      <DeliverablesPanel
        items={[{ path: ".gaea/exports/方案.docx", sourceId: "a2" }]}
        onOpenFile={() => {}}
      />,
    );
    expect(screen.queryByTitle("沉淀到成本库：把单价明细用 cost_save 写回成本库")).toBeNull();
  });

  it("图片产物渲染缩略图，非图片保留类型图标", async () => {
    const { container } = render(
      <DeliverablesPanel
        items={[
          { path: "exports/趋势.png", sourceId: "a1" },
          { path: ".gaea/exports/方案.docx", sourceId: "a2" },
        ]}
        onOpenFile={() => {}}
      />,
    );
    await waitFor(() => expect(container.querySelector("img")).toBeTruthy());
    const img = container.querySelector("img");
    expect(img?.getAttribute("src")).toContain("data:image/png");
    // 非图片产物不渲染 <img> 缩略图（仅一张图片，共两个产物）
    expect(container.querySelectorAll("img")).toHaveLength(1);
  });

  it("一键复制全部文件路径（最新在前）", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    render(
      <DeliverablesPanel
        items={[
          { path: "exports/成本测算.xlsx", sourceId: "a1" },
          { path: ".gaea/exports/方案.docx", sourceId: "a2" },
        ]}
        onOpenFile={() => {}}
      />,
    );
    fireEvent.click(screen.getByTitle("复制全部文件路径"));
    await waitFor(() =>
      expect(writeText).toHaveBeenCalledWith(".gaea/exports/方案.docx\nexports/成本测算.xlsx"),
    );
  });
});
