import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { MindMapView } from "./MindMapView";

const MD = "# 根\n- A\n  - A1\n- B\n";

describe("MindMapView（M1 交互导图视图）", () => {
  it("渲染大纲为交互节点（含根）", () => {
    render(<MindMapView text={MD} title="fallback" />);
    expect(screen.getAllByTestId("mind-node")).toHaveLength(4);
    expect(screen.getByText("根")).toBeTruthy();
    expect(screen.getByTestId("mind-zoom").textContent).toBe("100%");
  });

  it("点击父节点折叠：子节点消失、badge 显 +N；再点展开", () => {
    render(<MindMapView text={MD} title="f" />);
    fireEvent.click(screen.getByText("A"));
    expect(screen.queryByText("A1")).toBeNull();
    expect(screen.getAllByTestId("mind-node")).toHaveLength(3);
    expect(screen.getByTestId("mind-collapsed-badge").textContent).toBe("+1");
    fireEvent.click(screen.getByText("A"));
    expect(screen.getByText("A1")).toBeTruthy();
    expect(screen.queryByTestId("mind-collapsed-badge")).toBeNull();
  });

  it("叶子节点点击不产生折叠态", () => {
    render(<MindMapView text={MD} title="f" />);
    fireEvent.click(screen.getByText("A1"));
    expect(screen.queryByTestId("mind-collapsed-badge")).toBeNull();
    expect(screen.getAllByTestId("mind-node")).toHaveLength(4);
  });

  it("缩放按钮改百分比且夹在界内；回中复位", () => {
    render(<MindMapView text={MD} title="f" />);
    fireEvent.click(screen.getByTestId("mind-zoom-in"));
    expect(screen.getByTestId("mind-zoom").textContent).toBe("120%");
    fireEvent.click(screen.getByTestId("mind-reset"));
    expect(screen.getByTestId("mind-zoom").textContent).toBe("100%");
  });

  it("无大纲文档给空态提示；截断给上限提示", () => {
    render(<MindMapView text="只是普通段落文字，没有标题与列表。" title="t" />);
    expect(screen.getByText(/未发现大纲结构/)).toBeTruthy();

    const lines = ["# 大", ...Array.from({ length: 600 }, (_, i) => `- 项${i}`)];
    render(<MindMapView text={lines.join("\n")} title="t2" />);
    expect(screen.getByText(/仅渲染前 500 个节点/)).toBeTruthy();
  });
});

describe("MindMapView（M2 画布编辑）", () => {
  it("双击改名 + 保存条出现，Ctrl+S 回写规范大纲", () => {
    const onSave = vi.fn();
    render(<MindMapView text={MD} title="x" onSave={onSave} />);
    fireEvent.doubleClick(screen.getByText("A"));
    const input = screen.getByTestId("mind-rename-input") as HTMLInputElement;
    expect(input.value).toBe("A");
    fireEvent.change(input, { target: { value: "A 改" } });
    fireEvent.keyDown(input, { key: "Enter" });
    expect(screen.getByText("A 改")).toBeTruthy();
    expect(screen.getByTestId("mind-save")).toBeTruthy();
    fireEvent.keyDown(window, { key: "s", ctrlKey: true });
    expect(onSave).toHaveBeenCalledTimes(1);
    expect(onSave.mock.calls[0]![0] as string).toContain("## A 改");
  });

  it("Tab 加子节点、Enter 加同级、Delete 删除", () => {
    const onSave = vi.fn();
    render(<MindMapView text={MD} title="x" onSave={onSave} />);
    fireEvent.click(screen.getByText("B")); // 选中（叶子节点，无折叠副作用）
    fireEvent.keyDown(window, { key: "Tab" });
    expect(screen.getByText("新节点")).toBeTruthy();
    fireEvent.click(screen.getAllByText("新节点")[0]!.parentElement!); // 选中新节点
    fireEvent.keyDown(window, { key: "Enter" });
    expect(screen.getAllByText("新节点")).toHaveLength(2);
    fireEvent.click(screen.getAllByText("新节点")[0]!.parentElement!);
    fireEvent.keyDown(window, { key: "Delete" });
    expect(screen.getAllByText("新节点")).toHaveLength(1);
    expect(onSave).not.toHaveBeenCalled(); // 未保存前不回写
  });

  it("混合内容（skipped>0）编辑闸关闭：无保存条、快捷键不生效", () => {
    const onSave = vi.fn();
    const mixed = ["# 根", "- A", "", "这是普通段落。", ""].join("\n");
    render(<MindMapView text={mixed} title="x" onSave={onSave} />);
    expect(screen.getByText(/导图编辑暂不可用/)).toBeTruthy();
    fireEvent.click(screen.getByText("A"));
    fireEvent.keyDown(window, { key: "Tab" });
    expect(onSave).not.toHaveBeenCalled();
    expect(screen.queryByTestId("mind-save")).toBeNull();
  });

  it("未传 onSave 保持 M1 只读行为（可折叠、无编辑闸提示）", () => {
    render(<MindMapView text={MD} title="x" />);
    fireEvent.click(screen.getByText("A"));
    fireEvent.keyDown(window, { key: "Tab" });
    expect(screen.queryByTestId("mind-save")).toBeNull();
    expect(screen.queryByText(/导图编辑暂不可用/)).toBeNull();
  });
});
