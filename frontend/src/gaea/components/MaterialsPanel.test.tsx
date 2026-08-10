import { describe, expect, it } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MaterialsPanel } from "./MaterialsPanel";
import { useComposerInsertStore } from "../lib/store";

describe("MaterialsPanel 工作区资料概览", () => {
  it("按类型分组渲染资料，一键 @ 引用写入输入框通道", async () => {
    useComposerInsertStore.setState({ pendingAt: null });
    const opened: string[] = [];
    render(<MaterialsPanel onOpenFile={(p) => opened.push(p)} />);

    // mock 数据：docs/成本测算.xlsx、docs/方案.docx、docs/说明.md
    expect(await screen.findByText("文档 · 2")).toBeTruthy();
    expect(screen.getByText("表格 · 1")).toBeTruthy();

    const refBtns = screen.getAllByTitle("一键 @ 引用为对话上下文");
    fireEvent.click(refBtns[0]);
    expect(useComposerInsertStore.getState().pendingAt).toBe("docs/方案.docx");

    fireEvent.click(screen.getByText("成本测算.xlsx"));
    expect(opened).toContain("docs/成本测算.xlsx");

    useComposerInsertStore.getState().consumeAt();
  });

  it("固定常用资料：点击图钉后出现在「已固定」区并可取消", async () => {
    useComposerInsertStore.setState({ pendingAt: null });
    render(<MaterialsPanel onOpenFile={() => {}} />);

    await screen.findByText("文档 · 2");
    const pinBtns = screen.getAllByTitle("固定为常用资料（新会话自动带入）");
    fireEvent.click(pinBtns[0]);

    expect(await screen.findByText(/已固定 · 1/)).toBeTruthy();
    expect(screen.getByText(/新会话自动带入上下文/)).toBeTruthy();

    // 取消固定
    fireEvent.click(screen.getAllByTitle("取消固定")[0]);
    await waitFor(() => expect(screen.queryByText(/已固定 · 1/)).toBeNull());
  });

  it("摘要后引用：点击摘要按钮后摘要文本进入输入框通道", async () => {
    useComposerInsertStore.setState({ pendingText: null });
    render(<MaterialsPanel onOpenFile={() => {}} />);

    await screen.findByText("文档 · 2");
    fireEvent.click(screen.getAllByTitle("摘要后引用：分块摘要并插入输入框")[0]);

    await waitFor(() => expect(useComposerInsertStore.getState().pendingText).toBeTruthy());
    expect(useComposerInsertStore.getState().pendingText).toContain("分块摘要（mock）");
    useComposerInsertStore.getState().consumeText();
  });
});
