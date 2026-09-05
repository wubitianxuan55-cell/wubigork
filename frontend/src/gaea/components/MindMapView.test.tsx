import { describe, expect, it } from "vitest";
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
