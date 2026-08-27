import { afterEach, describe, expect, it, vi } from "vitest";
import { render, fireEvent, waitFor } from "@testing-library/react";
import { SelectionToComposer } from "./SelectionToComposer";
import { useComposerInsertStore } from "../lib/store";

// jsdom 无真实选区：stub window.getSelection 返回受控假选区
function fakeSelection(text: string, insideInput = false) {
  const anchor = insideInput
    ? { nodeType: 3, parentElement: document.createElement("input") }
    : { nodeType: 3, parentElement: document.createElement("div") };
  const range = {
    getBoundingClientRect: () => ({ left: 100, top: 200, width: 120, height: 18 }),
  };
  const sel = {
    toString: () => text,
    anchorNode: anchor,
    rangeCount: 1,
    getRangeAt: () => range,
  };
  vi.spyOn(window, "getSelection").mockReturnValue(sel as unknown as Selection);
}

function triggerSelection() {
  fireEvent.mouseUp(document);
  document.dispatchEvent(new Event("selectionchange"));
}

afterEach(() => {
  vi.restoreAllMocks();
  useComposerInsertStore.setState({ pendingText: null });
  document.body.innerHTML = "";
});

describe("SelectionToComposer 选区转对话（C4）", () => {
  it("选中正文后浮出「转为提问」，点击把选中文本以引用块插入输入框", async () => {
    fakeSelection("这段选中的文字需要继续处理");
    render(<SelectionToComposer />);
    triggerSelection();

    await waitFor(() => expect(document.body.textContent).toContain("转为提问"));
    fireEvent.click(document.body.querySelector("button")!);

    const text = useComposerInsertStore.getState().pendingText ?? "";
    expect(text).toContain("> 这段选中的文字需要继续处理");
    expect(text).toContain("请基于以上内容继续处理");
  });

  it("输入框/文本域内的选区不触发（避免干扰既有交互）", async () => {
    fakeSelection("输入框里的文字", true);
    render(<SelectionToComposer />);
    triggerSelection();
    await new Promise((r) => setTimeout(r, 20));
    expect(document.body.textContent ?? "").not.toContain("转为提问");
  });

  it("过短/空选区不触发", async () => {
    fakeSelection("一");
    render(<SelectionToComposer />);
    triggerSelection();
    await new Promise((r) => setTimeout(r, 20));
    expect(document.body.textContent ?? "").not.toContain("转为提问");

    fakeSelection("");
    document.dispatchEvent(new Event("selectionchange"));
    await new Promise((r) => setTimeout(r, 20));
    expect(document.body.textContent ?? "").not.toContain("转为提问");
  });

  it("点击 × 关闭浮动条", async () => {
    fakeSelection("多行引用\n第二行内容");
    render(<SelectionToComposer />);
    triggerSelection();
    await waitFor(() => expect(document.body.textContent).toContain("转为提问"));

    const buttons = document.body.querySelectorAll("button");
    fireEvent.click(buttons[buttons.length - 1]); // ×
    await waitFor(() => expect(document.body.textContent ?? "").not.toContain("转为提问"));
  });
});
