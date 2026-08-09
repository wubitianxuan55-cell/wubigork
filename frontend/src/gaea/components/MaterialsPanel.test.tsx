import { describe, expect, it } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
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
});
