import { describe, expect, it } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MaterialsLibrary } from "./MaterialsLibrary";
import { usePreviewStore } from "../../lib/store";

describe("MaterialsLibrary 记忆中枢项目资料库", () => {
  it("展示可固定资料，固定后进入「已固定」区并可取消", async () => {
    usePreviewStore.setState({ previewFile: null });
    render(<MaterialsLibrary />);

    // mock 工作区资料：成本测算.xlsx 等（初始未固定）
    expect(await screen.findByText("成本测算.xlsx")).toBeTruthy();
    expect(screen.getByText("已固定 · 0")).toBeTruthy();

    // 固定第一个候选
    const pinBtns = screen.getAllByTitle("固定为常用资料（新会话自动带入）");
    fireEvent.click(pinBtns[0]);
    expect(await screen.findByText("已固定 · 1")).toBeTruthy();

    // 点击预览 → 全局预览通道
    fireEvent.click(screen.getAllByTitle("预览")[0]);
    expect(usePreviewStore.getState().previewFile).toBe("docs/成本测算.xlsx");

    // 取消固定
    fireEvent.click(screen.getAllByTitle("取消固定")[0]);
    await waitFor(() => expect(screen.queryByText("已固定 · 1")).toBeNull());
    usePreviewStore.getState().closeFilePreview();
  });
});
