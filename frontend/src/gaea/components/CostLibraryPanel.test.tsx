import { describe, expect, it } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { CostLibraryPanel } from "./CostLibraryPanel";
import { useComposerInsertStore } from "../lib/store";
import { ToastProvider } from "./Toast";

const wrap = (node: React.ReactNode) => <ToastProvider>{node}</ToastProvider>;

describe("CostLibraryPanel 办公侧成本库", () => {
  it("浏览成本条目，一键引用把结构化单价插入输入框", async () => {
    useComposerInsertStore.setState({ pendingText: null });
    render(wrap(<CostLibraryPanel />));

    expect(await screen.findByText("HP300 高频液压振动锤")).toBeTruthy();
    expect(screen.getByText("P.O 42.5 水泥")).toBeTruthy();

    fireEvent.click(screen.getAllByTitle("把该成本条目作为结构化上下文插入输入框")[0]);
    await waitFor(() => expect(useComposerInsertStore.getState().pendingText).toBeTruthy());
    const text = useComposerInsertStore.getState().pendingText ?? "";
    expect(text).toContain("【成本库】HP300 高频液压振动锤");
    expect(text).toContain("name: hp300");
    expect(text).toContain("¥3,200");
    useComposerInsertStore.getState().consumeText();
  });

  it("分类过滤生效，空结果显示提示", async () => {
    render(wrap(<CostLibraryPanel />));
    await screen.findByText("HP300 高频液压振动锤");

    fireEvent.click(screen.getByText("材料", { selector: "button" }));
    await waitFor(() => expect(screen.queryByText("HP300 高频液压振动锤")).toBeNull());
    expect(screen.getByText("P.O 42.5 水泥")).toBeTruthy();
  });

  it("多选后可批量删除并提示结果", async () => {
    render(wrap(<CostLibraryPanel />));
    await screen.findByText("HP300 高频液压振动锤");

    const boxes = screen.getAllByTitle("多选（批量删除/改状态）");
    fireEvent.click(boxes[0]);
    fireEvent.click(boxes[1]);
    expect(screen.getByText("已选 2")).toBeTruthy();

    fireEvent.click(screen.getByText("批量删除"));
    await waitFor(() => expect(screen.getByText("已删除 2 条")).toBeTruthy());
  });

  it("点击编辑打开共享编辑弹窗", async () => {
    render(wrap(<CostLibraryPanel />));
    await screen.findByText("HP300 高频液压振动锤");

    fireEvent.click(screen.getAllByTitle("编辑")[0]);
    expect(await screen.findByText("编辑成本")).toBeTruthy();
  });
});
